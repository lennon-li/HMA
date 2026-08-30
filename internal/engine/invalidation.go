package engine

import (
	"sort"

	"github.com/lennon-li/HMA/internal/model"
)

// RewindTrigger enumerates the Gate 0 legal mid-run rewind triggers. No other
// rewind is legal.
type RewindTrigger string

const (
	// TriggerGroundingChanged: grounding input, repository identity, base
	// revision, or applicable instruction changed.
	TriggerGroundingChanged RewindTrigger = "GROUNDING_INPUT_CHANGED"
	// TriggerCriteriaChanged: an approved criterion or non-goal changed.
	TriggerCriteriaChanged RewindTrigger = "CRITERION_OR_NON_GOAL_CHANGED"
	// TriggerPlanUnitChanged: plan-unit behavior, allowed/forbidden scope,
	// budget, dependency, or required evidence changed.
	TriggerPlanUnitChanged RewindTrigger = "PLAN_UNIT_CHANGED"
	// TriggerRouteChanged: route, model/provider family, access service,
	// permission envelope, interaction mode, or validator-independence
	// requirement changed.
	TriggerRouteChanged RewindTrigger = "ROUTE_OR_INDEPENDENCE_CHANGED"
	// TriggerProducedHeadOrDiffChanged: produced head or diff changed before
	// implementation authorization/review evidence was accepted.
	TriggerProducedHeadOrDiffChanged RewindTrigger = "PRODUCED_HEAD_OR_DIFF_CHANGED"
	// TriggerReviewEvidenceDeficient: review evidence shows the
	// implementation is incomplete, out of scope, or over budget while
	// repair remains permitted.
	TriggerReviewEvidenceDeficient RewindTrigger = "REVIEW_EVIDENCE_INCOMPLETE_OUT_OF_SCOPE_OR_OVER_BUDGET"
	// TriggerVerificationInputChanged: required verification evidence, tool
	// version, head, diff, or declared command input changed.
	TriggerVerificationInputChanged RewindTrigger = "VERIFICATION_EVIDENCE_OR_COMMAND_INPUT_CHANGED"
	// TriggerValidationCorrection: independent validation requires a
	// permitted correction.
	TriggerValidationCorrection RewindTrigger = "INDEPENDENT_VALIDATION_CORRECTION_REQUIRED"
	// TriggerValidationPlanDefect: independent validation exposed a
	// criterion or plan defect. CriterionChanged selects the target.
	TriggerValidationPlanDefect RewindTrigger = "INDEPENDENT_VALIDATION_PLAN_OR_CRITERION_DEFECT"
	// TriggerClosureInputChanged: closure input, required CI result, release
	// request, or release approval changed.
	TriggerClosureInputChanged RewindTrigger = "CLOSURE_CI_OR_RELEASE_INPUT_CHANGED"
)

// RewindScope is the blast radius a rewind is permitted to have.
type RewindScope string

const (
	ScopeWholeRun                  RewindScope = "WHOLE_RUN"
	ScopeAffectedUnit              RewindScope = "AFFECTED_UNIT"
	ScopeAffectedUnitAndDependents RewindScope = "AFFECTED_UNIT_AND_DEPENDENT_UNITS"
	// ScopeBoundPackets is the scope of a waiver-applicability change: only
	// the approval packets that bound the changed waiver operation. Gate 0
	// §1 defines this as a scoped invalidation, not a rewind class.
	ScopeBoundPackets RewindScope = "BOUND_PACKETS_ONLY"
)

// rewindRule is the approved target and scope for one trigger.
type rewindRule struct {
	target model.Stage
	scope  RewindScope
}

// rewindTable is the exhaustive Gate 0 §2 rewind table.
// TriggerValidationPlanDefect carries its default (plan-defect) rule; the
// criterion-defect variant is resolved in EvaluateInvalidation.
var rewindTable = map[RewindTrigger]rewindRule{
	TriggerGroundingChanged:          {model.StageGrounding, ScopeWholeRun},
	TriggerCriteriaChanged:           {model.StageAcceptanceCriteria, ScopeWholeRun},
	TriggerPlanUnitChanged:           {model.StagePlanning, ScopeAffectedUnitAndDependents},
	TriggerRouteChanged:              {model.StageRouteSelection, ScopeAffectedUnit},
	TriggerProducedHeadOrDiffChanged: {model.StageImplementationAuthorization, ScopeAffectedUnit},
	TriggerReviewEvidenceDeficient:   {model.StageImplementationAuthorization, ScopeAffectedUnit},
	TriggerVerificationInputChanged:  {model.StageVerification, ScopeAffectedUnit},
	TriggerValidationCorrection:      {model.StageImplementationAuthorization, ScopeAffectedUnit},
	TriggerValidationPlanDefect:      {model.StagePlanning, ScopeAffectedUnit},
	TriggerClosureInputChanged:       {model.StageReleaseAndClosure, ScopeWholeRun},
}

// WaiverOp is an active human waiver operation and its canonical scope. It
// mirrors model.Waiver without re-declaring portable record shape.
type WaiverOp struct {
	ID     string          `json:"id"`
	Kind   model.ScopeKind `json:"kind"`
	Target string          `json:"target"`
	Active bool            `json:"active"`
}

// ApprovalPacket is the engine-internal projection of one previously granted
// approval: the gate it approved, the unit it covers, the opaque input
// classes it bound, and its waiver applicability set. It is deliberately not
// a model.Record and adds no packet_control field to the portable vocabulary.
type ApprovalPacket struct {
	ID string `json:"id"`
	// UnitID is empty for a whole-run (pre-planning) approval.
	UnitID string      `json:"unit_id,omitempty"`
	Stage  model.Stage `json:"stage"`
	// BoundInputs are opaque input-class keys this approval actually binds.
	BoundInputs []string `json:"bound_inputs,omitempty"`
	// WaiverIDs is the sorted waiver applicability set (Gate 0 §1).
	WaiverIDs []string `json:"waiver_ids,omitempty"`
	// Invalidated records an approval already invalidated by an earlier
	// event. It can never be restored.
	Invalidated bool `json:"invalidated,omitempty"`
}

// InvalidationRequest is an explicit, already-declared invalidation input.
// The engine derives nothing about the world from it and reads no repository.
type InvalidationRequest struct {
	Trigger RewindTrigger `json:"trigger"`
	// CriterionChanged distinguishes the two TriggerValidationPlanDefect
	// targets: a changed criterion rewinds to ACCEPTANCE_CRITERIA.
	CriterionChanged bool     `json:"criterion_changed,omitempty"`
	AffectedUnitID   string   `json:"affected_unit_id,omitempty"`
	DependentUnitIDs []string `json:"dependent_unit_ids,omitempty"`
	// ChangedInputs are the opaque input-class keys that changed.
	ChangedInputs []string `json:"changed_inputs,omitempty"`
	// ChangedWaiverIDs are waiver operations added, expired, withdrawn, or
	// changed by this event.
	ChangedWaiverIDs []string `json:"changed_waiver_ids,omitempty"`
	// ChangedWaivedTargets are criterion/finding/artifact identifiers whose
	// covered content changed; an active waiver over one of them expires.
	ChangedWaivedTargets []string         `json:"changed_waived_targets,omitempty"`
	Waivers              []WaiverOp       `json:"waivers,omitempty"`
	Approvals            []ApprovalPacket `json:"approvals,omitempty"`
	// RequestReactivateApprovalIDs is a request to restore an invalidated
	// approval. It is always rejected.
	RequestReactivateApprovalIDs []string `json:"request_reactivate_approval_ids,omitempty"`
	// ClaimsMachineAdvancement asserts the evaluation itself advances the
	// run. It is always rejected.
	ClaimsMachineAdvancement bool `json:"claims_machine_advancement,omitempty"`
}

// InvalidationResult is the deterministic, side-effect-free invalidation
// report. It records facts for later append-only recording by a caller; it
// never writes them, and it is never a human approval.
type InvalidationResult struct {
	Decision Decision `json:"decision"`
	Reason   Reason   `json:"reason,omitempty"`
	// RewindTarget is the stage the run returns to. Empty on rejection.
	RewindTarget model.Stage `json:"rewind_target,omitempty"`
	Scope        RewindScope `json:"scope,omitempty"`
	// ControlState is the posture of the rewind target: READY_FOR_REVIEW.
	ControlState model.StageControlState `json:"control_state,omitempty"`
	// PacketControl is the packet-control vocabulary applied to each
	// invalidated approval. It is never a stage and never a model.Record
	// control field.
	PacketControl              model.PacketControlState `json:"packet_control,omitempty"`
	InvalidatedApprovalIDs     []string                 `json:"invalidated_approval_ids,omitempty"`
	PreservedApprovalIDs       []string                 `json:"preserved_approval_ids,omitempty"`
	AlreadyInvalidatedIDs      []string                 `json:"already_invalidated_approval_ids,omitempty"`
	ExpiredWaiverIDs           []string                 `json:"expired_waiver_ids,omitempty"`
	RequiresFreshHumanApproval bool                     `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool                     `json:"machine_advanced"`
	RestoredApprovalIDs        []string                 `json:"restored_approval_ids,omitempty"`
}

func contains(set []string, want string) bool {
	for _, s := range set {
		if s == want {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	for _, x := range a {
		if contains(b, x) {
			return true
		}
	}
	return false
}

// sorted returns a new sorted copy, never aliasing or reordering the input.
func sorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}

// expiredWaivers returns the active waivers whose covered criterion, finding,
// or artifact changed (Gate 0 §1). An expired waiver is itself a waiver
// applicability change.
func expiredWaivers(waivers []WaiverOp, changedTargets []string) []string {
	var out []string
	for _, w := range waivers {
		if w.Active && w.Target != "" && contains(changedTargets, w.Target) {
			out = append(out, w.ID)
		}
	}
	return sorted(out)
}

// inScope reports whether an approval falls inside the rewind's blast radius.
func inScope(a ApprovalPacket, req InvalidationRequest, scope RewindScope) bool {
	switch scope {
	case ScopeWholeRun:
		return true
	case ScopeAffectedUnit:
		return a.UnitID != "" && a.UnitID == req.AffectedUnitID
	case ScopeAffectedUnitAndDependents:
		return a.UnitID != "" &&
			(a.UnitID == req.AffectedUnitID || contains(req.DependentUnitIDs, a.UnitID))
	}
	return false
}

// EvaluateInvalidation computes the approved rewind target, scope, and
// invalidated-approval set for an explicit invalidation event.
//
// It is pure. It does not mutate a record, advance a run, restore an
// invalidated approval, expire anything in storage, or stand in for the fresh
// human approval that every replacement packet still requires.
func EvaluateInvalidation(req InvalidationRequest) InvalidationResult {
	rejected := func(r Reason) InvalidationResult {
		return InvalidationResult{
			Decision:                   DecisionRejected,
			Reason:                     r,
			RequiresFreshHumanApproval: true,
			MachineAdvanced:            false,
		}
	}
	if req.ClaimsMachineAdvancement {
		return rejected(ReasonMachineAdvancement)
	}
	if len(req.RequestReactivateApprovalIDs) > 0 {
		return rejected(ReasonApprovalReactivation)
	}
	rule, ok := rewindTable[req.Trigger]
	if !ok {
		return rejected(ReasonUnknownRewindTrigger)
	}
	if req.Trigger == TriggerValidationPlanDefect && req.CriterionChanged {
		// Gate 0 §2: a criterion defect that changes the criterion itself
		// rewinds to ACCEPTANCE_CRITERIA, which is a whole-run class.
		rule = rewindRule{model.StageAcceptanceCriteria, ScopeWholeRun}
	}
	if rule.scope == ScopeWholeRun && req.AffectedUnitID != "" {
		// A whole-run trigger must not be narrowed to one unit.
		return rejected(ReasonUnitScopeOnWholeRunRewind)
	}
	if rule.scope != ScopeWholeRun && req.AffectedUnitID == "" {
		return rejected(ReasonMissingAffectedUnit)
	}

	expired := expiredWaivers(req.Waivers, req.ChangedWaivedTargets)
	changedWaivers := append(sorted(req.ChangedWaiverIDs), expired...)
	changedWaivers = sorted(changedWaivers)
	targetIdx := stageOrder[rule.target]

	var invalidated, preserved, already []string
	for _, a := range req.Approvals {
		if a.Invalidated {
			// An invalidated approval stays invalidated. Its supersession
			// fact is preserved for later append-only recording.
			already = append(already, a.ID)
			continue
		}
		idx, known := stageOrder[a.Stage]
		bindingChanged := intersects(a.BoundInputs, req.ChangedInputs) ||
			intersects(a.WaiverIDs, changedWaivers)
		switch {
		case inScope(a, req, rule.scope) && (!known || idx >= targetIdx):
			// Downstream of the target gate, inside the rewind scope.
			invalidated = append(invalidated, a.ID)
		case bindingChanged:
			// Upstream or out of scope, but a bound input actually changed.
			invalidated = append(invalidated, a.ID)
		default:
			preserved = append(preserved, a.ID)
		}
	}

	return InvalidationResult{
		Decision:                   DecisionLegalPendingApproval,
		RewindTarget:               rule.target,
		Scope:                      rule.scope,
		ControlState:               model.ControlReadyForReview,
		PacketControl:              model.PacketInvalidated,
		InvalidatedApprovalIDs:     sorted(invalidated),
		PreservedApprovalIDs:       sorted(preserved),
		AlreadyInvalidatedIDs:      sorted(already),
		ExpiredWaiverIDs:           expired,
		RequiresFreshHumanApproval: true,
		MachineAdvanced:            false,
		RestoredApprovalIDs:        nil,
	}
}

// WaiverChangeRequest is a waiver operation that is added, expired,
// withdrawn, or changed. Gate 0 §1 makes this a scoped packet invalidation,
// not one of the §2 rewind classes: it never advances a stage and never
// rewinds the whole run.
type WaiverChangeRequest struct {
	ChangedWaiverIDs             []string         `json:"changed_waiver_ids,omitempty"`
	ChangedWaivedTargets         []string         `json:"changed_waived_targets,omitempty"`
	Waivers                      []WaiverOp       `json:"waivers,omitempty"`
	Approvals                    []ApprovalPacket `json:"approvals,omitempty"`
	RequestReactivateApprovalIDs []string         `json:"request_reactivate_approval_ids,omitempty"`
	ClaimsMachineAdvancement     bool             `json:"claims_machine_advancement,omitempty"`
}

// EvaluateWaiverChange reports which approval packets a waiver operation
// invalidates. An unrelated waiver invalidates nothing. A waiver operation is
// scoped packet invalidation, not a Gate 0 §2 rewind class, so it never
// invents a RewindTarget or READY_FOR_REVIEW posture.
func EvaluateWaiverChange(req WaiverChangeRequest) InvalidationResult {
	rejected := func(r Reason) InvalidationResult {
		return InvalidationResult{
			Decision:                   DecisionRejected,
			Reason:                     r,
			RequiresFreshHumanApproval: true,
			MachineAdvanced:            false,
		}
	}
	if req.ClaimsMachineAdvancement {
		return rejected(ReasonMachineAdvancement)
	}
	if len(req.RequestReactivateApprovalIDs) > 0 {
		return rejected(ReasonApprovalReactivation)
	}

	expired := expiredWaivers(req.Waivers, req.ChangedWaivedTargets)
	changedWaivers := sorted(append(sorted(req.ChangedWaiverIDs), expired...))

	var invalidated, preserved, already []string
	for _, a := range req.Approvals {
		if a.Invalidated {
			already = append(already, a.ID)
			continue
		}
		if intersects(a.WaiverIDs, changedWaivers) || intersects(a.BoundInputs, req.ChangedWaivedTargets) {
			invalidated = append(invalidated, a.ID)
			continue
		}
		preserved = append(preserved, a.ID)
	}

	res := InvalidationResult{
		Decision:                   DecisionLegalPendingApproval,
		Scope:                      ScopeBoundPackets,
		InvalidatedApprovalIDs:     sorted(invalidated),
		PreservedApprovalIDs:       sorted(preserved),
		AlreadyInvalidatedIDs:      sorted(already),
		ExpiredWaiverIDs:           expired,
		RequiresFreshHumanApproval: true,
		MachineAdvanced:            false,
	}
	if len(invalidated) > 0 {
		res.PacketControl = model.PacketInvalidated
	}
	return res
}
