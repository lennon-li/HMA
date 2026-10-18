package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonLeaseNoIssuingApproval Reason = "LEASE_NO_ISSUING_APPROVAL"
	ReasonLeasePlanInvalid       Reason = "LEASE_PLAN_INVALID"
	ReasonLeasePlanMismatch      Reason = "LEASE_PLAN_DIGEST_MISMATCH"
	ReasonLeaseUnitNotInPlan     Reason = "LEASE_UNIT_NOT_IN_PLAN"
	ReasonLeaseContractMismatch  Reason = "LEASE_CONTRACT_MISMATCH"
	ReasonLeaseRunTerminal       Reason = "LEASE_RUN_TERMINAL"
	ReasonLeaseStage             Reason = "LEASE_UNIT_NOT_IN_IMPLEMENTATION_AUTHORIZATION"
	ReasonLeaseRewound           Reason = "LEASE_VOID_AFTER_REWIND"
	ReasonLeaseWallTime          Reason = "LEASE_WALL_TIME_EXPIRED"
	ReasonLeaseCeiling           Reason = "LEASE_CEILING_REACHED"
	ReasonLeasePolicyChanged     Reason = "LEASE_POLICY_DIGEST_CHANGED"
	ReasonLeaseCriteriaChanged   Reason = "LEASE_CRITERIA_CHANGED"
	ReasonLeaseNotYetIssued      Reason = "LEASE_NOT_YET_ISSUED"
	ReasonLeaseCountersInvalid   Reason = "LEASE_COUNTERS_INVALID"
	ReasonLeaseRouteMismatch     Reason = "LEASE_ROUTE_MISMATCH"
	ReasonLeaseWaiverChanged     Reason = "LEASE_VOID_AFTER_WAIVER_CHANGE"
)

// LeaseCounters are a unit's consumption so far, carried across rewinds
// (Gate A v3 §3.2). D3 derives them from reservations.
type LeaseCounters struct {
	Attempts  int   `json:"attempts"`
	CostSpent int64 `json:"cost_spent"`
}

// LeaseRequest is everything a lease derivation reads. Records are the core
// chain only: no host-written record can grant or extend a lease.
type LeaseRequest struct {
	Records  []model.Record
	UnitID   string
	Plan     []byte // hma.plan/v1 document the host presents
	Contract []byte // hma.acceptance-contract/v1 document the run binds
	// PolicyDigest is the policy digest HMA currently computes.
	PolicyDigest string
	Counters     LeaseCounters
	Now          time.Time
	// Route is the selected route's projection; required.
	Route *model.RouteGrant
}

// Lease is the authority one approved plan unit holds while it is active. It
// is derived, never stored as authority; LeaseDigest identifies it so a host
// record may cache it.
type Lease struct {
	UnitID string `json:"unit_id"`
	// Generation is the issuing approval's chain position and anchor: a new
	// issuing approval starts a new generation, and an old one never revives.
	IssuerSequence   int64             `json:"issuer_sequence"`
	IssuerHeadAnchor string            `json:"issuer_head_anchor"`
	IssuerStage      model.Stage       `json:"issuer_stage"`
	PlanDigest       string            `json:"plan_digest"`
	ContractDigest   string            `json:"contract_digest"`
	PolicyDigest     string            `json:"policy_digest"`
	ImplementerRoute string            `json:"implementer_route"`
	Budgets          model.UnitBudgets `json:"budgets"`
	IssuedAt         string            `json:"issued_at"`
	// ValidUntil is IssuedAt plus the unit's max_wall_minutes; the lease is
	// active only strictly before it.
	ValidUntil  string `json:"valid_until"`
	LeaseDigest string `json:"lease_digest"`
}

// DeriveLease derives the unit's lease from the core chain and the presented
// plan and contract, and reports ReasonNone only while it is active (Gate A
// v3 §2.2, v4 §1-§2). It is pure.
//
// Issuer: the unit's latest approved transition targeting
// IMPLEMENTATION_AUTHORIZATION, from ROUTE_SELECTION or
// INDEPENDENT_VALIDATION, the only legal such edges. The presented plan must
// validate, hash to that approval's accepted_plan_digest, contain the unit,
// bind the run's accepted contract, and stay within its ceilings.
//
// Active while: the run is not terminal; the unit's projected stage is
// IMPLEMENTATION_AUTHORIZATION; no later approved return to an earlier gate
// by the unit, the whole run, or a unit it transitively depends on in the
// plan; now is before the wall-time limit; counters are below the ceilings;
// and the current policy digest equals the plan's.
func DeriveLease(req LeaseRequest) (Lease, Reason) {
	var lease Lease
	issuer := -1
	for i, r := range req.Records {
		if r.UnitID == req.UnitID && r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.UnitID == req.UnitID && r.Approval.ProposedTargetStage == model.StageImplementationAuthorization &&
			(r.Approval.CurrentStage == model.StageRouteSelection || r.Approval.CurrentStage == model.StageIndependentValidation) {
			issuer = i
		}
	}
	if req.UnitID == "" || issuer < 0 {
		return lease, ReasonLeaseNoIssuingApproval
	}
	ir := req.Records[issuer]
	a := ir.Approval

	plan, planDigest, err := model.ParsePlan(req.Plan)
	if err != nil {
		return lease, ReasonLeasePlanInvalid
	}
	if planDigest != a.AcceptedPlanDigest || plan.RunID != a.RunID {
		return lease, ReasonLeasePlanMismatch
	}
	var unit *model.PlanUnit
	for i := range plan.Units {
		if plan.Units[i].ID == req.UnitID {
			unit = &plan.Units[i]
		}
	}
	if unit == nil {
		return lease, ReasonLeaseUnitNotInPlan
	}
	bound := BoundContractDigest(req.Records)
	contract, contractDigest, err := model.ParseAcceptanceContract(req.Contract)
	if err != nil || bound == "" || contractDigest != bound || a.AcceptanceContractDigest != bound ||
		model.ValidatePlanAgainstContract(plan, contract, contractDigest) != nil {
		return lease, ReasonLeaseContractMismatch
	}

	// Wall time runs from the human approval, not from when HMA appended it.
	issued, err := time.Parse(time.RFC3339Nano, a.Timestamp)
	if err != nil {
		return lease, ReasonLeaseNoIssuingApproval
	}
	until := issued.Add(time.Duration(unit.Budgets.MaxWallMinutes) * time.Minute)
	// lease_digest = digest of {approval transition digest, unit, plan
	// digest} (Gate A v3 §2.2). The generation is the issuer's sequence
	// and anchor, carried separately.
	sum := sha256.Sum256(mustJSON(struct {
		TransitionDigest string `json:"transition_digest"`
		UnitID           string `json:"unit_id"`
		PlanDigest       string `json:"plan_digest"`
	}{a.TransitionDigest, req.UnitID, planDigest}))
	lease = Lease{UnitID: req.UnitID, IssuerSequence: ir.Metadata.Sequence, IssuerHeadAnchor: ir.Metadata.HeadAnchor,
		IssuerStage: a.CurrentStage, PlanDigest: planDigest, ContractDigest: bound, PolicyDigest: plan.PolicyDigest,
		ImplementerRoute: unit.ImplementerRoute, Budgets: unit.Budgets,
		IssuedAt: issued.UTC().Format(time.RFC3339Nano), ValidUntil: until.UTC().Format(time.RFC3339Nano),
		LeaseDigest: hex.EncodeToString(sum[:])}

	state := ProjectStageState(req.Records)
	if state.Terminal {
		return lease, ReasonLeaseRunTerminal
	}
	// The core criterion set must still be the contract's.
	if !sameIDs(model.ContractCriterionIDs(contract), state.Criteria) {
		return lease, ReasonLeaseCriteriaChanged
	}
	if issuerWaiversChanged(req.Records, issuer) {
		return lease, ReasonLeaseWaiverChanged
	}
	deps := dependencyClosure(plan, req.UnitID)
	// The authority chain starts at the unit's route approval: for a
	// correction issuer, everything since the route approval must leave it
	// valid, and only the unit's own legal forward walk back to the
	// correction is allowed in between.
	start := issuer
	if ri := routeApprovalFor(req.Records[:issuer+1], req.UnitID); ri >= 0 && ri < issuer {
		start = ri
	}
	for i, r := range req.Records[start+1:] {
		if r.Kind != model.KindStageTransition || r.Control != model.ControlApproved || r.Approval == nil {
			continue
		}
		if start+1+i <= issuer && (r.UnitID == req.UnitID || r.UnitID == "") && !isEarlierGate(r.Approval.ProposedTargetStage) {
			// The unit's own forward walk between the route approval
			// and the correction issuer (review, verification,
			// validation, the correction itself).
			continue
		}
		// A return to Grounding or Acceptance criteria reopens the whole
		// run's intent or criteria, whatever unit the record names.
		if t := r.Approval.ProposedTargetStage; t == model.StageGrounding || t == model.StageAcceptanceCriteria {
			return lease, ReasonLeaseRewound
		}
		switch {
		case r.UnitID == req.UnitID || r.UnitID == "":
			// The unit's own (or a whole-run) later transition leaves
			// Implementation authorization, forward or back.
			if r.Approval.ProposedTargetStage != model.StageImplementationAuthorization {
				if isEarlierGate(r.Approval.ProposedTargetStage) {
					return lease, ReasonLeaseRewound
				}
				return lease, ReasonLeaseStage
			}
		case deps[r.UnitID]:
			// A dependency returning to an earlier gate, or correcting
			// its implementation after independent validation, changes
			// what this unit was built on.
			if isEarlierGate(r.Approval.ProposedTargetStage) ||
				r.Approval.CurrentStage == model.StageIndependentValidation && r.Approval.ProposedTargetStage == model.StageImplementationAuthorization {
				return lease, ReasonLeaseRewound
			}
		}
	}
	// The selected route must be the plan unit's implementer route, hold no
	// permission outside the unit's grant, and reference the unit's route
	// approval: the issuer itself, or for a correction issued from
	// Independent validation the latest Route selection approval before it.
	routeIdx := routeApprovalFor(req.Records[:issuer+1], req.UnitID)
	if req.Route == nil || req.Route.Identity != unit.ImplementerRoute || req.Route.ApprovalDigest == "" || routeIdx < 0 ||
		req.Route.ApprovalDigest != req.Records[routeIdx].Approval.TransitionDigest || len(req.Route.PermissionClasses) == 0 {
		return lease, ReasonLeaseRouteMismatch
	}
	// The route approval must itself still be valid: a correction issuer
	// cannot carry a route approval a waiver change has invalidated. There is
	// no fallback to an older route approval.
	if routeIdx != issuer && (issuerWaiversChanged(req.Records, routeIdx) || req.Records[routeIdx].Approval.AcceptedPlanDigest != planDigest) {
		return lease, ReasonLeaseWaiverChanged
	}
	granted := map[string]bool{}
	for _, p := range unit.PermissionClasses {
		granted[p] = true
	}
	for _, p := range req.Route.PermissionClasses {
		if !granted[p] {
			return lease, ReasonLeaseRouteMismatch
		}
	}
	if req.Now.Before(issued) {
		return lease, ReasonLeaseNotYetIssued
	}
	if !req.Now.Before(until) {
		return lease, ReasonLeaseWallTime
	}
	if req.Counters.Attempts < 0 || req.Counters.CostSpent < 0 {
		return lease, ReasonLeaseCountersInvalid
	}
	if req.Counters.Attempts >= unit.Budgets.MaxAttempts || req.Counters.CostSpent >= unit.Budgets.CostCeiling {
		return lease, ReasonLeaseCeiling
	}
	if req.PolicyDigest == "" || req.PolicyDigest != plan.PolicyDigest {
		return lease, ReasonLeasePolicyChanged
	}
	return lease, ReasonNone
}

// issuerWaiversChanged reports whether a waiver the issuing approval bound
// (its active_waivers: waiver-operation record digests) was withdrawn,
// expired, replaced, or otherwise changed after the issuer, using the same
// packet-invalidation rule as EvaluateWaiverChange. A bound digest that does
// not name the live grant of its scope at the issuer, or is repeated,
// counts as changed (fail closed). A later regrant
// of the same scope does not revive the generation.
func issuerWaiversChanged(records []model.Record, issuer int) bool {
	bound := records[issuer].Approval.ActiveWaivers
	if len(bound) == 0 {
		return false
	}
	// Replay waiver history up to the issuer: each scope's live grant is
	// the latest GRANT not since withdrawn, expired, or replaced (including
	// by a legacy waiver record). Every bound digest must name a live grant,
	// once.
	live := map[model.ScopeSelector]string{}
	for _, r := range records[:issuer] {
		if w := r.WaiverOperation; w != nil {
			if w.Operation == model.WaiverOperationGrant {
				live[w.Scope] = r.Metadata.HeadAnchor
			} else {
				delete(live, w.Scope)
			}
		}
		for _, lw := range r.Waivers {
			delete(live, lw.Scope)
		}
	}
	scopeOf := map[string]model.ScopeSelector{}
	for scope, d := range live {
		scopeOf[d] = scope
	}
	seen := map[string]bool{}
	for _, d := range bound {
		if seen[d] {
			return true
		}
		seen[d] = true
		if _, ok := scopeOf[d]; !ok {
			return true
		}
	}
	var changed []string
	for _, d := range bound {
		scope := scopeOf[d]
		for _, r := range records[issuer+1:] {
			if w := r.WaiverOperation; w != nil && w.Scope == scope {
				changed = append(changed, d)
				break
			}
			hit := false
			for _, lw := range r.Waivers {
				if lw.Scope == scope {
					hit = true
				}
			}
			if hit {
				changed = append(changed, d)
				break
			}
		}
	}
	if len(changed) == 0 {
		return false
	}
	res := EvaluateWaiverChange(WaiverChangeRequest{ChangedWaiverIDs: changed,
		Approvals: []ApprovalPacket{{ID: "issuer", WaiverIDs: append([]string(nil), bound...)}}})
	return res.Decision != DecisionLegalPendingApproval || len(res.InvalidatedApprovalIDs) > 0
}

// routeApprovalFor returns the index of the unit's latest approved
// ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION in records, or -1.
func routeApprovalFor(records []model.Record, unit string) int {
	idx := -1
	for i, r := range records {
		if r.UnitID == unit && r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.CurrentStage == model.StageRouteSelection && r.Approval.ProposedTargetStage == model.StageImplementationAuthorization {
			idx = i
		}
	}
	return idx
}

// isEarlierGate reports a return to a gate before implementation
// authorization: the plan, criteria, or route is being decided again.
func isEarlierGate(s model.Stage) bool {
	switch s {
	case model.StageGrounding, model.StageAcceptanceCriteria, model.StagePlanning, model.StageRouteSelection:
		return true
	}
	return false
}

// dependencyClosure returns every unit the given unit transitively depends
// on in the plan.
func dependencyClosure(p model.Plan, unit string) map[string]bool {
	byID := map[string][]string{}
	for _, u := range p.Units {
		byID[u.ID] = u.Dependencies
	}
	out := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		for _, d := range byID[id] {
			if !out[d] {
				out[d] = true
				visit(d)
			}
		}
	}
	visit(unit)
	return out
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
