package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
)

// UnauthorizedDispatchPrefix starts the id of every core finding
// reconciliation records. Imported finding ids always contain "/", so these
// never collide with them.
const UnauthorizedDispatchPrefix = "UNAUTHORIZED_DISPATCH:"

// LedgerRun is one agent-dispatch ledger run found in a declared namespace:
// its directory name, what its receipt says (Err when the receipt is missing
// or unreadable), and the receipt bytes' SHA-256.
type LedgerRun struct {
	Namespace     string `json:"namespace"`
	Dir           string `json:"dir"`
	RunID         string `json:"run_id"`
	Mode          string `json:"mode"`
	StartedAt     string `json:"started_at"`
	ReceiptSHA256 string `json:"receipt_sha256,omitempty"`
	Err           string `json:"error,omitempty"`
	// Launch is the receipt's launch identity (route, packet, target).
	Launch LaunchIdentity `json:"launch"`
}

// Unauthorized is one ledger run or journal entry reconciliation refuses.
type Unauthorized struct {
	RunID  string `json:"run_id"`
	Reason string `json:"reason"`
}

// Reconciliation is the result of one reconciliation, recorded as a host
// record. Unlaunched reservations stay charged. InventoryDigest binds the
// exact ledger contents reconciled; the transition gate recomputes it.
type Reconciliation struct {
	Namespaces      []string       `json:"namespaces"`
	Authorized      []string       `json:"authorized"`
	Unauthorized    []Unauthorized `json:"unauthorized"`
	Unlaunched      []string       `json:"unlaunched"`
	Reservations    int            `json:"reservations"`
	InventoryDigest string         `json:"inventory_digest"`
	PolicyDigest    string         `json:"policy_digest,omitempty"`
	// RevokedPolicies are plan policies this reconciliation observed
	// replaced by PolicyDigest, revoked from ObservedAt on, like a
	// policy_revocation record.
	RevokedPolicies []string `json:"revoked_policies,omitempty"`
	ObservedAt      string   `json:"observed_at,omitempty"`
}

// ReconcileInput is everything Reconcile reads. Plans are the accepted plan
// documents by digest (from the store's artifacts); PolicyDigest is the
// current policy digest ("" when the host presented none).
type ReconcileInput struct {
	Core         []model.Record
	History      []Record
	Plans        map[string]model.Plan
	Runs         []LedgerRun
	PolicyDigest string
	Now          time.Time
}

// Namespaces returns every ledger namespace the run declares: each loop
// admission's, each HMA-generated review dispatch's, and each imported
// journal's, sorted.
func Namespaces(history []Record) []string {
	seen := map[string]bool{}
	var out []string
	add := func(ns string) {
		if ns != "" && !seen[ns] {
			seen[ns] = true
			out = append(out, ns)
		}
	}
	for _, r := range history {
		if r.Kind == KindLoopAdmission && r.Admission != nil {
			add(r.Admission.LedgerNamespace)
		}
		if r.Kind == KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil {
			add(r.Dispatch.Packet.LedgerNamespace)
		}
		// An imported journal declares its namespace too, admitted or not:
		// it can never opt the run out of reconciliation.
		if r.Kind == KindLoopEvidence && r.LoopEvidence != nil {
			add(r.LoopEvidence.LedgerNamespace)
		}
	}
	sort.Strings(out)
	return out
}

// InventoryDigest is the digest of every ledger run's namespace, directory,
// receipt digest, and error, in sorted order.
func InventoryDigest(runs []LedgerRun) string {
	rows := make([]string, 0, len(runs))
	for _, r := range runs {
		b, _ := json.Marshal([]string{r.Namespace, r.Dir, r.ReceiptSHA256, r.Err})
		rows = append(rows, string(b))
	}
	sort.Strings(rows)
	sum := sha256.Sum256([]byte(strings.Join(rows, "\n")))
	return hex.EncodeToString(sum[:])
}

// Reconcile classifies every ledger run in every declared namespace against
// the host records (Gate A v5 §1-§2, v6 §2 and §4, v7 §1). It is pure.
//
// A run is authorized only when it matches exactly one reservation or one
// HMA-generated review dispatch by dispatch id, in that record's namespace
// and mode, used by no other authorized run, and started in
// [valid_from, valid_until): exact times, so a whole-second receipt in the
// same second as a sub-second record time is outside. valid_until is the
// minimum of the record's end and every applicable revocation: core
// rewinds per the generation's own plan, a terminal outcome, and an observed
// policy change. A changed current policy with no dated observation makes
// the generation's launches unproven. Every HMA-bound journal with attempts
// must match exactly one admission (controller, unit, envelope, namespace,
// live root, clone baseline) and have a reservation for every attempt.
func Reconcile(in ReconcileInput) Reconciliation {
	out := Reconciliation{Namespaces: Namespaces(in.History), Authorized: []string{}, Unauthorized: []Unauthorized{}, Unlaunched: []string{},
		InventoryDigest: InventoryDigest(in.Runs), PolicyDigest: in.PolicyDigest}
	bad := func(id, reason string) {
		out.Unauthorized = append(out.Unauthorized, Unauthorized{RunID: id, Reason: reason})
	}
	var first time.Time
	if len(in.Core) > 0 {
		first, _ = time.Parse(time.RFC3339Nano, in.Core[0].Metadata.Timestamp)
	}
	admissions := map[string]*Record{}
	byDispatch := map[string][]*Record{}
	for i := range in.History {
		r := &in.History[i]
		switch {
		case r.Kind == KindLoopAdmission && r.Admission != nil:
			admissions[r.HeadAnchor] = r
		case r.Kind == KindReservation && r.Reservation != nil:
			out.Reservations++
			byDispatch[r.Reservation.DispatchRunID] = append(byDispatch[r.Reservation.DispatchRunID], r)
		case r.Kind == KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil:
			byDispatch[r.Dispatch.Packet.DispatchRunID] = append(byDispatch[r.Dispatch.Packet.DispatchRunID], r)
		}
	}
	terminal := time.Time{}
	for _, r := range in.Core {
		if r.Kind == model.KindOutcome && model.ValidTerminalOutcome(r.Outcome) && terminal.IsZero() {
			terminal, _ = time.Parse(time.RFC3339Nano, r.Metadata.Timestamp)
		}
	}
	runs := append([]LedgerRun(nil), in.Runs...)
	sort.Slice(runs, func(i, j int) bool { return runs[i].Namespace+"/"+runs[i].Dir < runs[j].Namespace+"/"+runs[j].Dir })
	used := map[string]bool{}
	for _, run := range runs {
		id := run.Dir
		if run.Err != "" {
			bad(id, "LEDGER_RUN_UNREADABLE")
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, run.StartedAt)
		if err != nil {
			bad(id, "LEDGER_RUN_UNREADABLE")
			continue
		}
		if t.Before(first) {
			continue
		}
		if run.RunID != run.Dir {
			bad(id, "LEDGER_RUN_ID_MISMATCH")
			continue
		}
		matches := byDispatch[run.RunID]
		if len(matches) != 1 {
			if len(matches) == 0 {
				bad(id, "NO_MATCHING_RECORD")
			} else {
				bad(id, "AMBIGUOUS_MATCH")
			}
			continue
		}
		m := matches[0]
		ns, wantReview := "", false
		if m.Kind == KindReservation {
			adm := admissions[m.Reservation.AdmissionRecord]
			if adm == nil || adm.Admission.ControllerRunID != m.Reservation.ControllerRunID {
				bad(id, "NAMESPACE_MISMATCH")
				continue
			}
			ns = adm.Admission.LedgerNamespace
		} else {
			ns, wantReview = m.Dispatch.Packet.LedgerNamespace, true
		}
		if ns == "" || run.Namespace != ns {
			bad(id, "NAMESPACE_MISMATCH")
			continue
		}
		if (run.Mode == "review") != wantReview {
			bad(id, "MODE_MISMATCH")
			continue
		}
		if !launchMatches(m, run.Launch, in.History) {
			bad(id, "ROUTE_MISMATCH")
			continue
		}
		from, until, reason := validity(m, in, terminal)
		if reason != "" {
			bad(id, reason)
			continue
		}
		if t.Before(from) || !t.Before(until) {
			bad(id, "OUTSIDE_VALIDITY")
			continue
		}
		if used[m.HeadAnchor] {
			bad(id, "RECORD_REUSED")
			continue
		}
		used[m.HeadAnchor] = true
		out.Authorized = append(out.Authorized, id)
	}
	for i := range in.History {
		if v := in.History[i].Reservation; in.History[i].Kind == KindReservation && v != nil && !used[in.History[i].HeadAnchor] {
			out.Unlaunched = append(out.Unlaunched, v.DispatchRunID)
		}
	}
	reconcileJournals(in.History, admissions, byDispatch, bad)
	return out
}

// reconcileJournals checks every imported verify-loop journal against
// exactly one admission and its attempts against its reservations.
func reconcileJournals(history []Record, admissions map[string]*Record, byDispatch map[string][]*Record, bad func(id, reason string)) {
	for _, r := range history {
		l := r.LoopEvidence
		if r.Kind != KindLoopEvidence || l == nil {
			continue
		}
		var matched []*LoopAdmission
		for _, a := range admissions {
			if a.Admission.ControllerRunID == l.ControllerRunID {
				matched = append(matched, a.Admission)
			}
		}
		// Every imported journal binds exactly one admission, whatever
		// its attempt count.
		if len(matched) != 1 {
			bad(l.ControllerRunID, "JOURNAL_ADMISSION_MISSING")
		}
		if len(matched) == 1 {
			adm := matched[0]
			if adm.EnvelopeSHA256 != l.EnvelopeSHA256 || adm.LedgerNamespace != l.LedgerNamespace || adm.UnitID != l.UnitID ||
				adm.CloneBaselineDigest != l.CloneBaselineDigest || adm.LiveRoot != l.LiveRoot {
				bad(l.ControllerRunID, "JOURNAL_ADMISSION_MISMATCH")
			}
		}
		for _, a := range l.Attempts {
			if a.DispatchRunID == "" {
				continue
			}
			ok := false
			for _, m := range byDispatch[a.DispatchRunID] {
				if m.Kind == KindReservation && m.Reservation.ControllerRunID == l.ControllerRunID && m.Reservation.UnitID == l.UnitID {
					ok = true
				}
			}
			if !ok {
				bad(a.DispatchRunID, "JOURNAL_ATTEMPT_UNRESERVED")
			}
		}
	}
}

// launchMatches reports whether a launch's receipt identity is the one its
// record authorizes. A review launch must be exactly the dispatch's route,
// packet, and target. An implementation launch must be the worker, model,
// and effort of the implementation preflight route the reservation's route
// identity names: the fields the verify-loop route binds.
func launchMatches(m *Record, l LaunchIdentity, history []Record) bool {
	if m.Kind != KindReservation {
		return ReviewLaunchMatches(m.Dispatch, l)
	}
	v := m.Reservation
	for _, r := range history {
		p := r.Preflight
		if r.Kind != KindPreflight || p == nil || p.UnitID != v.UnitID || p.Role != RoleImplementation {
			continue
		}
		if g, err := RouteGrant(p.Route); err == nil && g.Identity == v.RouteIdentity {
			return l.Agent == p.Route.WorkerID && l.Model == p.Route.Model && l.Effort == p.Route.ReasoningEffort
		}
	}
	return false
}

// validity returns a matched record's half-open launch window, or a reason
// it cannot be established.
func validity(m *Record, in ReconcileInput, terminal time.Time) (time.Time, time.Time, string) {
	if m.Kind == KindReservation {
		v := m.Reservation
		from, err1 := time.Parse(time.RFC3339Nano, v.ValidFrom)
		until, err2 := time.Parse(time.RFC3339Nano, v.ValidUntil)
		if err1 != nil || err2 != nil {
			return from, until, "RECORD_TIMES_UNREADABLE"
		}
		plan, ok := generationPlan(in, v.Generation.IssuerSequence)
		if !ok {
			return from, until, "GENERATION_PLAN_MISSING"
		}
		until = engine.RevokedAt(in.Core, plan, v.UnitID, v.Generation.IssuerSequence, until)
		until, reason := policyBound(in, plan, v.Generation.IssuerSequence, until)
		return from, until, reason
	}
	// A review dispatch belongs to the unit's lease generation current at
	// its core head: it ends no later than that generation's wall-time end,
	// and the generation's policy revocations end it too.
	p := m.Dispatch.Packet
	from, err1 := parseTimeNano(m.Timestamp)
	until, err2 := parseTimeNano(p.ValidUntil)
	if err1 != nil || err2 != nil {
		return from, until, "RECORD_TIMES_UNREADABLE"
	}
	if !terminal.IsZero() && terminal.Before(until) {
		until = terminal
	}
	after := int64(-1)
	var issuer *model.Record
	for i, r := range in.Core {
		if after < 0 && engine.IssuesLease(r, m.Dispatch.UnitID) {
			issuer = &in.Core[i]
		}
		if r.Metadata.HeadAnchor == m.CoreHeadAnchor {
			after = r.Metadata.Sequence
		}
	}
	if after < 0 {
		return from, until, "RECORD_TIMES_UNREADABLE"
	}
	if issuer == nil {
		return from, until, "GENERATION_MISSING"
	}
	plan, ok := in.Plans[issuer.Approval.AcceptedPlanDigest]
	var unit *model.PlanUnit
	for i := range plan.Units {
		if plan.Units[i].ID == m.Dispatch.UnitID {
			unit = &plan.Units[i]
		}
	}
	if !ok || unit == nil {
		return from, until, "GENERATION_PLAN_MISSING"
	}
	issued, err := parseTimeNano(issuer.Approval.Timestamp)
	if err != nil {
		return from, until, "RECORD_TIMES_UNREADABLE"
	}
	if wall := issued.Add(time.Duration(unit.Budgets.MaxWallMinutes) * time.Minute); wall.Before(until) {
		until = wall
	}
	// Revocations count from the generation's issuer, not from the packet:
	// a dependency rewind before the packet was issued still revokes it.
	until = engine.RevokedReviewAt(in.Core, plan, m.Dispatch.UnitID, issuer.Metadata.Sequence, until)
	until, reason := policyBound(in, plan, issuer.Metadata.Sequence, until)
	return from, until, reason
}

// policyBound ends a generation's launches at the first recorded revocation
// of its plan's policy, and refuses them when the host presented no current
// policy or the current one differs with no applicable dated revocation.
func policyBound(in ReconcileInput, plan model.Plan, issuerSeq int64, until time.Time) (time.Time, string) {
	t, revoked := PolicyRevokedAt(in.Core, in.History, plan.PolicyDigest, issuerSeq)
	if revoked && t.Before(until) {
		until = t
	}
	if in.PolicyDigest == "" {
		return until, "POLICY_NOT_PRESENTED"
	}
	if in.PolicyDigest != plan.PolicyDigest && !revoked {
		// The change is not dated: no launch of this generation can be
		// shown to predate it.
		return until, "POLICY_REVOCATION_UNPROVEN"
	}
	return until, ""
}

// policyObservation returns the time of r's observed revocation of policy,
// if r records one: a policy_revocation, or a launch reconciliation that
// observed the policy replaced.
func policyObservation(r Record, policy string) (string, bool) {
	if v := r.PolicyRevocation; r.Kind == KindPolicyRevocation && v != nil && v.PlanPolicyDigest == policy {
		return r.Timestamp, true
	}
	if v := r.Reconciliation; r.Kind == KindLaunchReconciliation && v != nil {
		for _, d := range v.RevokedPolicies {
			if d == policy {
				return v.ObservedAt, true
			}
		}
	}
	return "", false
}

// PolicyRevocationNeeded reports whether HMA, observing a current policy
// other than policy, must record a revocation of it: no recorded revocation
// applies to the latest lease generation in core. Deduplication is per
// generation, so a repeated change under a later generation is recorded.
func PolicyRevocationNeeded(core []model.Record, history []Record, policy string) bool {
	latest := int64(0)
	for _, r := range core {
		if engine.IssuesLease(r, r.UnitID) && r.Metadata.Sequence > latest {
			latest = r.Metadata.Sequence
		}
	}
	_, revoked := PolicyRevokedAt(core, history, policy, latest)
	return !revoked
}

// PolicyRevokedAt returns the earliest recorded revocation of policy (a
// policy_revocation or a reconciliation's observation) that applies to the lease generation issued at core sequence issuerSeq: one
// recorded when the core head was at or after the issuer. A later approved
// generation is not revoked by it; restoring the policy revives nothing. A
// revocation whose time or core head cannot be read applies from the start
// (fail closed).
func PolicyRevokedAt(core []model.Record, history []Record, policy string, issuerSeq int64) (time.Time, bool) {
	seq := map[string]int64{}
	for _, r := range core {
		seq[r.Metadata.HeadAnchor] = r.Metadata.Sequence
	}
	var end time.Time
	found := false
	for _, r := range history {
		at, applies := policyObservation(r, policy)
		if !applies {
			continue
		}
		head, ok := seq[r.CoreHeadAnchor]
		if ok && head < issuerSeq {
			continue
		}
		t, err := parseTimeNano(at)
		if err != nil || !ok {
			t = time.Time{}
		}
		if !found || t.Before(end) {
			end, found = t, true
		}
	}
	return end, found
}

// generationPlan returns the plan the generation's issuing approval binds.
func generationPlan(in ReconcileInput, issuerSeq int64) (model.Plan, bool) {
	for _, r := range in.Core {
		if r.Metadata.Sequence == issuerSeq && r.Approval != nil {
			p, ok := in.Plans[r.Approval.AcceptedPlanDigest]
			return p, ok
		}
	}
	return model.Plan{}, false
}

func parseTimeNano(v string) (time.Time, error) { return time.Parse(time.RFC3339Nano, v) }

// UnauthorizedFindings returns the core findings for a reconciliation: one
// unowned BLOCK per unauthorized entry. Unowned findings survive every
// review snapshot and only a human can dispose of them.
func UnauthorizedFindings(r Reconciliation) []model.Finding {
	out := []model.Finding{}
	for _, u := range r.Unauthorized {
		out = append(out, model.Finding{ID: UnauthorizedDispatchPrefix + u.RunID + ":" + u.Reason, Disposition: model.FindingBlock})
	}
	return out
}

// HasUnauthorizedDispatch reports a BLOCK unauthorized-dispatch finding in
// the current core findings.
func HasUnauthorizedDispatch(findings []model.Finding) bool {
	for _, f := range findings {
		if strings.HasPrefix(f.ID, UnauthorizedDispatchPrefix) && f.Disposition == model.FindingBlock {
			return true
		}
	}
	return false
}

// ReconciliationRelevant reports host records that change what a
// reconciliation would conclude: any record that adds a launch authority or
// a journal.
func ReconciliationRelevant(r Record) bool {
	switch r.Kind {
	case KindLoopAdmission, KindReservation, KindLoopEvidence, KindPolicyRevocation:
		return true
	case KindDispatch:
		return r.Dispatch != nil && r.Dispatch.Packet != nil
	}
	return false
}
