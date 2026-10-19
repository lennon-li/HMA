package host

import (
	"fmt"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

var rcT0 = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func rcAt(m int) string { return rcT0.Add(time.Duration(m) * time.Minute).Format(time.RFC3339Nano) }

var rcBudgets = model.UnitBudgets{MaxAttempts: 3, MaxWallMinutes: 120, CostCeiling: 1}

type rcFixture struct {
	core      []model.Record
	history   []Record
	plan      model.Plan
	ns        string
	preflight bool
}

func newRC() *rcFixture {
	f := &rcFixture{ns: "/ledger/vl-1/dispatch-ledger",
		plan: model.Plan{PolicyDigest: "pol", Units: []model.PlanUnit{{ID: "U1", Dependencies: []string{}, Budgets: rcBudgets},
			{ID: "U2", Dependencies: []string{"U1"}, Budgets: rcBudgets}}}}
	f.coreRec(model.Record{Kind: model.KindCriterion}, 0)
	f.approve("U2", model.StageRouteSelection, model.StageImplementationAuthorization, 0)
	f.host(Record{Kind: KindLoopAdmission, HeadAnchor: "adm-1", Admission: &LoopAdmission{UnitID: "U2", ControllerRunID: "vl-1",
		EnvelopeSHA256: "env", LedgerNamespace: f.ns, LiveRoot: "/repo", CloneBaselineDigest: "base", Generation: LeaseGeneration{IssuerSequence: 2, IssuerHeadAnchor: "c2"}}})
	return f
}

func (f *rcFixture) coreRec(r model.Record, m int) {
	r.Metadata = model.RecordMetadata{Sequence: int64(len(f.core) + 1), HeadAnchor: fmt.Sprintf("c%d", len(f.core)+1), Timestamp: rcAt(m)}
	f.core = append(f.core, r)
}

func (f *rcFixture) approve(unit string, from, to model.Stage, m int) {
	f.coreRec(model.Record{UnitID: unit, Kind: model.KindStageTransition, Control: model.ControlApproved,
		Approval: &model.ApprovalBinding{UnitID: unit, CurrentStage: from, ProposedTargetStage: to, Timestamp: rcAt(m), AcceptedPlanDigest: "plan-1"}}, m)
}

func (f *rcFixture) host(r Record) {
	r.Sequence = int64(len(f.history) + 1)
	if r.HeadAnchor == "" {
		r.HeadAnchor = fmt.Sprintf("h%d", r.Sequence)
	}
	// Every host record is appended at a core head.
	if r.CoreHeadAnchor == "" {
		r.CoreHeadAnchor = f.core[len(f.core)-1].Metadata.HeadAnchor
	}
	f.history = append(f.history, r)
}

var (
	rcImpl     = testRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	rcReviewer = testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
)

// reserve adds a reservation of U2 valid from minute from to minute until,
// on the implementation route the run's preflight names.
func (f *rcFixture) reserve(id string, from, until int, issuerSeq int64) {
	g, _ := RouteGrant(rcImpl)
	if !f.preflight {
		f.preflight = true
		f.host(Record{Kind: KindPreflight, Preflight: &RoutePreflight{UnitID: "U2", Role: RoleImplementation, Route: rcImpl, Status: PreflightAvailable}})
	}
	f.host(Record{Kind: KindReservation, Reservation: &Reservation{UnitID: "U2", ControllerRunID: "vl-1", AttemptIndex: len(f.history),
		DispatchRunID: id, AdmissionRecord: "adm-1", ValidFrom: rcAt(from), ValidUntil: rcAt(until), RouteIdentity: g.Identity,
		Generation: LeaseGeneration{IssuerSequence: issuerSeq}}})
}

func (f *rcFixture) launch(id string, m int) LedgerRun {
	return LedgerRun{Namespace: f.ns, Dir: id, RunID: id, Mode: "write", StartedAt: rcAt(m),
		Launch: LaunchIdentity{Agent: rcImpl.WorkerID, Model: rcImpl.Model, Effort: rcImpl.ReasoningEffort}}
}

func (f *rcFixture) run(runs ...LedgerRun) Reconciliation {
	return Reconcile(ReconcileInput{Core: f.core, History: f.history, Plans: map[string]model.Plan{"plan-1": f.plan}, Runs: runs, PolicyDigest: "pol"})
}

func (f *rcFixture) verdict(t *testing.T, runs ...LedgerRun) map[string]string {
	t.Helper()
	res := f.run(runs...)
	out := map[string]string{}
	for _, id := range res.Authorized {
		out[id] = "ok"
	}
	for _, u := range res.Unauthorized {
		out[u.RunID] = u.Reason
	}
	return out
}

func TestUnreservedDispatchBlocks(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	got := f.verdict(t, f.launch("d-1", 2), f.launch("d-x", 3))
	if got["d-1"] != "ok" || got["d-x"] != "NO_MATCHING_RECORD" {
		t.Fatalf("%v", got)
	}
	res := f.run(f.launch("d-x", 3))
	fs := UnauthorizedFindings(res)
	if !HasUnauthorizedDispatch(fs) || fs[0].UnitID != "" || fs[0].Disposition != model.FindingBlock {
		t.Fatalf("findings %+v", fs)
	}
}

func TestOneReservationPerLaunchNoReuse(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	// A second ledger run claiming the same dispatch id, and a ledger run
	// whose receipt names another id, are unauthorized.
	got := f.verdict(t, f.launch("d-1", 2), LedgerRun{Namespace: f.ns, Dir: "d-1-again", RunID: "d-1", Mode: "write", StartedAt: rcAt(3)})
	if got["d-1"] != "ok" || got["d-1-again"] != "LEDGER_RUN_ID_MISMATCH" {
		t.Fatalf("%v", got)
	}
	// A dispatch id held by both a reservation and a review dispatch is
	// ambiguous for every launch.
	f.host(Record{Kind: KindDispatch, Timestamp: rcAt(1), Dispatch: &Dispatch{Packet: &ReviewPacketBinding{DispatchRunID: "d-1", ValidUntil: rcAt(60), LedgerNamespace: f.ns}}})
	if got := f.verdict(t, f.launch("d-1", 2)); got["d-1"] != "AMBIGUOUS_MATCH" {
		t.Fatalf("%v", got)
	}
}

func TestUnlaunchedReservationCharged(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.reserve("d-2", 1, 60, 2)
	res := f.run(f.launch("d-1", 2))
	if len(res.Unlaunched) != 1 || res.Unlaunched[0] != "d-2" || res.Reservations != 2 {
		t.Fatalf("%+v", res)
	}
}

func TestReconcileNamespaceModeAndReadability(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	other := f.launch("d-1", 2)
	other.Namespace = "/ledger/other/dispatch-ledger"
	if got := f.verdict(t, other); got["d-1"] != "NAMESPACE_MISMATCH" {
		t.Fatalf("%v", got)
	}
	review := f.launch("d-1", 2)
	review.Mode = "review"
	if got := f.verdict(t, review); got["d-1"] != "MODE_MISMATCH" {
		t.Fatalf("%v", got)
	}
	if got := f.verdict(t, LedgerRun{Namespace: f.ns, Dir: "d-9", Err: "receipt missing"}); got["d-9"] != "LEDGER_RUN_UNREADABLE" {
		t.Fatalf("%v", got)
	}
	// Runs that started before the run's first core record belong to no run.
	if got := f.verdict(t, f.launch("d-old", -5)); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestLaunchAtValidUntilUnauthorized(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	if got := f.verdict(t, f.launch("d-1", 60)); got["d-1"] != "OUTSIDE_VALIDITY" {
		t.Fatalf("%v", got)
	}
	if got := f.verdict(t, f.launch("d-1", 0)); got["d-1"] != "OUTSIDE_VALIDITY" {
		t.Fatalf("before valid_from: %v", got)
	}
}

func TestLaunchAtRevocationTimeUnauthorized(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.approve("U1", model.StageImplementationAuthorization, model.StagePlanning, 10) // dependency rewind
	if got := f.verdict(t, f.launch("d-1", 10)); got["d-1"] != "OUTSIDE_VALIDITY" {
		t.Fatalf("%v", got)
	}
}

func TestLaunchAtEarliestEndUnauthorized(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.approve("", model.StageVerification, model.StagePlanning, 20)
	f.approve("U2", model.StageImplementationAuthorization, model.StageImplementationReview, 15)
	if got := f.verdict(t, f.launch("d-1", 15)); got["d-1"] != "OUTSIDE_VALIDITY" {
		t.Fatalf("%v", got)
	}
}

func TestLaunchBeforeEarliestEndAuthorized(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.approve("", model.StageVerification, model.StagePlanning, 20)
	f.approve("U2", model.StageImplementationAuthorization, model.StageImplementationReview, 15)
	if got := f.verdict(t, f.launch("d-1", 14)); got["d-1"] != "ok" {
		t.Fatalf("%v", got)
	}
}

func TestRenewalDoesNotReviveOldGeneration(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.approve("U2", model.StageImplementationAuthorization, model.StageRouteSelection, 10)
	f.approve("U2", model.StageRouteSelection, model.StageImplementationAuthorization, 11) // new generation
	if got := f.verdict(t, f.launch("d-1", 12)); got["d-1"] != "OUTSIDE_VALIDITY" {
		t.Fatalf("old reservation launched under the new lease: %v", got)
	}
}

func TestEnvelopeAndAssociationMustMatchAdmission(t *testing.T) {
	for name, l := range map[string]LoopEvidence{
		"envelope":   {UnitID: "U2", ControllerRunID: "vl-1", EnvelopeSHA256: "other", LedgerNamespace: "/ledger/vl-1/dispatch-ledger"},
		"namespace":  {UnitID: "U2", ControllerRunID: "vl-1", EnvelopeSHA256: "env", LedgerNamespace: "/ledger/x/dispatch-ledger"},
		"unit":       {UnitID: "U1", ControllerRunID: "vl-1", EnvelopeSHA256: "env", LedgerNamespace: "/ledger/vl-1/dispatch-ledger", LiveRoot: "/repo", CloneBaselineDigest: "base"},
		"unreserved": {UnitID: "U2", ControllerRunID: "vl-1", EnvelopeSHA256: "env", LedgerNamespace: "/ledger/vl-1/dispatch-ledger", LiveRoot: "/repo", CloneBaselineDigest: "base", Attempts: []LoopAttempt{{N: 1, DispatchRunID: "d-z"}}},
	} {
		f := newRC()
		f.reserve("d-1", 1, 60, 2)
		l := l
		f.host(Record{Kind: KindLoopEvidence, LoopEvidence: &l})
		if res := f.run(); len(res.Unauthorized) == 0 {
			t.Errorf("%s accepted", name)
		}
	}
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.host(Record{Kind: KindLoopEvidence, LoopEvidence: &LoopEvidence{UnitID: "U2", ControllerRunID: "vl-1", EnvelopeSHA256: "env",
		LedgerNamespace: f.ns, LiveRoot: "/repo", CloneBaselineDigest: "base", Attempts: []LoopAttempt{{N: 1, DispatchRunID: "d-1"}}}})
	if res := f.run(); len(res.Unauthorized) != 0 {
		t.Fatalf("matching journal refused: %+v", res)
	}
}

// Liz D4 F05: valid_from is compared exactly; a whole-second receipt time
// before a sub-second valid_from is outside.
func TestLaunchBeforeExactValidFromUnauthorized(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.history[len(f.history)-1].Reservation.ValidFrom = rcT0.Add(time.Minute + 900*time.Millisecond).Format(time.RFC3339Nano)
	for _, at := range []time.Duration{time.Minute, time.Minute + 800*time.Millisecond} {
		l := f.launch("d-1", 0)
		l.StartedAt = rcT0.Add(at).Format(time.RFC3339Nano)
		if got := f.verdict(t, l); got["d-1"] != "OUTSIDE_VALIDITY" {
			t.Fatalf("%s: %v", at, got)
		}
	}
	l := f.launch("d-1", 0)
	l.StartedAt = rcT0.Add(time.Minute + 900*time.Millisecond).Format(time.RFC3339Nano)
	if got := f.verdict(t, l); got["d-1"] != "ok" {
		t.Fatalf("exact valid_from: %v", got)
	}
}

// A stray run in another namespace claiming a reservation is unauthorized
// and does not take the reservation from the real launch, in either scan
// order.
func TestReservationMatchedOnce(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	second := f.launch("d-1", 3)
	second.Namespace = "/ledger/a-copy/dispatch-ledger"
	for _, ns := range []string{"/ledger/a-copy/dispatch-ledger", "/ledger/z-copy/dispatch-ledger"} {
		second.Namespace = ns
		res := f.run(f.launch("d-1", 2), second)
		if len(res.Authorized) != 1 || len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "NAMESPACE_MISMATCH" {
			t.Fatalf("%s: %+v", ns, res)
		}
	}
}

// Liz D4 F02: each reservation uses its own generation's plan. A new plan
// adds U1 -> U2; a reservation under it is revoked by U1's rewind even
// though the older plan had no such dependency.
func TestReconcileUsesGenerationPlan(t *testing.T) {
	f := newRC()
	f.plan.Units[1].Dependencies = []string{} // plan-1: U2 independent
	newPlan := model.Plan{PolicyDigest: "pol", Units: []model.PlanUnit{{ID: "U1", Dependencies: []string{}}, {ID: "U2", Dependencies: []string{"U1"}}}}
	f.coreRec(model.Record{UnitID: "U2", Kind: model.KindStageTransition, Control: model.ControlApproved,
		Approval: &model.ApprovalBinding{UnitID: "U2", CurrentStage: model.StageRouteSelection, ProposedTargetStage: model.StageImplementationAuthorization,
			Timestamp: rcAt(1), AcceptedPlanDigest: "plan-2"}}, 1)
	f.reserve("d-1", 2, 60, int64(len(f.core)))
	f.approve("U1", model.StageImplementationAuthorization, model.StagePlanning, 10)
	res := Reconcile(ReconcileInput{Core: f.core, History: f.history, Plans: map[string]model.Plan{"plan-1": f.plan, "plan-2": newPlan},
		Runs: []LedgerRun{f.launch("d-1", 11)}, PolicyDigest: "pol"})
	if len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "OUTSIDE_VALIDITY" {
		t.Fatalf("%+v", res)
	}
	res = Reconcile(ReconcileInput{Core: f.core, History: f.history, Plans: map[string]model.Plan{"plan-1": f.plan},
		Runs: []LedgerRun{f.launch("d-1", 11)}, PolicyDigest: "pol"})
	if len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "GENERATION_PLAN_MISSING" {
		t.Fatalf("missing plan: %+v", res)
	}
}

func (f *rcFixture) review(id string, m int) {
	f.host(Record{Kind: KindDispatch, Timestamp: rcAt(m), CoreHeadAnchor: f.core[len(f.core)-1].Metadata.HeadAnchor,
		Dispatch: &Dispatch{UnitID: "U2", Role: RoleReview, Route: rcReviewer, ReviewPacketDigest: "packet",
			Packet: &ReviewPacketBinding{DispatchRunID: id, ValidUntil: rcAt(m + 60), LedgerNamespace: "/ledger/review",
				ProjectRoot: "/", ExpectedLaunchHead: "head", ExpectedLaunchWorktree: "wt"}}})
}

// rcReviewLaunch is the launch identity the fixture's review dispatch binds.
var rcReviewLaunch = LaunchIdentity{Agent: rcReviewer.WorkerID, Model: rcReviewer.Model, Provider: rcReviewer.Provider,
	Effort: rcReviewer.ReasoningEffort, PacketSHA256: "packet", ProjectRoot: "/", LaunchHead: "head", LaunchWorktreeSHA256: "wt"}

func reviewLaunch(id string, m int) LedgerRun {
	return LedgerRun{Namespace: "/ledger/review", Dir: id, RunID: id, Mode: "review", StartedAt: rcAt(m), Launch: rcReviewLaunch}
}

// F03/F04: a review dispatch is bound to its ledger namespace and revoked by
// rewinds; the unit's own forward move into review does not revoke it.
func TestReviewDispatchNamespaceAndRevocation(t *testing.T) {
	f := newRC()
	f.review("r-1", 5)
	if got := f.verdict(t, reviewLaunch("r-1", 6)); got["r-1"] != "ok" {
		t.Fatalf("%v", got)
	}
	other := reviewLaunch("r-1", 6)
	other.Namespace = "/ledger/elsewhere"
	if got := f.verdict(t, other); got["r-1"] != "NAMESPACE_MISMATCH" {
		t.Fatalf("%v", got)
	}
	f.approve("U2", model.StageImplementationAuthorization, model.StageImplementationReview, 7)
	if got := f.verdict(t, reviewLaunch("r-1", 8)); got["r-1"] != "ok" {
		t.Fatalf("forward move revoked review: %v", got)
	}
	for name, mv := range map[string]func(g *rcFixture){
		"own-rewind": func(g *rcFixture) { g.approve("U2", model.StageImplementationReview, model.StagePlanning, 7) },
		"dependency": func(g *rcFixture) {
			g.approve("U1", model.StageImplementationAuthorization, model.StageRouteSelection, 7)
		},
		"whole-run": func(g *rcFixture) { g.approve("", model.StageVerification, model.StageAcceptanceCriteria, 7) },
	} {
		g := newRC()
		g.review("r-1", 5)
		mv(g)
		if got := g.verdict(t, reviewLaunch("r-1", 7)); got["r-1"] != "OUTSIDE_VALIDITY" {
			t.Errorf("%s: %v", name, got)
		}
	}
	if ns := Namespaces(f.history); len(ns) != 2 {
		t.Fatalf("namespaces %v", ns)
	}
}

// F06: a journal with attempts but no matching admission is unauthorized,
// and its unreserved attempts are reported.
func TestJournalWithoutAdmissionUnauthorized(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.host(Record{Kind: KindLoopEvidence, LoopEvidence: &LoopEvidence{UnitID: "U2", ControllerRunID: "unknown-controller", EnvelopeSHA256: "env",
		LedgerNamespace: f.ns, Attempts: []LoopAttempt{{N: 1, DispatchRunID: "d-z"}}}})
	res := f.run()
	reasons := map[string]bool{}
	for _, u := range res.Unauthorized {
		reasons[u.Reason] = true
	}
	if !reasons["JOURNAL_ADMISSION_MISSING"] || !reasons["JOURNAL_ATTEMPT_UNRESERVED"] {
		t.Fatalf("%+v", res)
	}
}

// F07: the journal's clone baseline and live root must equal the admission's.
func TestJournalBaselineMustMatchAdmission(t *testing.T) {
	for name, edit := range map[string]func(l *LoopEvidence){
		"baseline": func(l *LoopEvidence) { l.CloneBaselineDigest = "changed" },
		"root":     func(l *LoopEvidence) { l.LiveRoot = "/elsewhere" },
	} {
		f := newRC()
		f.reserve("d-1", 1, 60, 2)
		l := LoopEvidence{UnitID: "U2", ControllerRunID: "vl-1", EnvelopeSHA256: "env", LedgerNamespace: f.ns, LiveRoot: "/repo", CloneBaselineDigest: "base",
			Attempts: []LoopAttempt{{N: 1, DispatchRunID: "d-1"}}}
		edit(&l)
		f.host(Record{Kind: KindLoopEvidence, LoopEvidence: &l})
		if res := f.run(); len(res.Unauthorized) == 0 {
			t.Errorf("%s accepted", name)
		}
	}
}

// F08: a launch at or after an observed policy change is unauthorized; a
// changed current policy with no observation is unproven; no policy at all
// is refused.
func TestReconcilePolicyRevocation(t *testing.T) {
	f := newRC()
	f.reserve("d-1", 1, 60, 2)
	f.host(Record{Kind: KindPolicyRevocation, Timestamp: rcAt(10), PolicyRevocation: &PolicyRevocation{PlanPolicyDigest: "pol", ObservedPolicyDigest: "pol-2"}})
	run := func(policy string, at int) string {
		res := Reconcile(ReconcileInput{Core: f.core, History: f.history, Plans: map[string]model.Plan{"plan-1": f.plan},
			Runs: []LedgerRun{f.launch("d-1", at)}, PolicyDigest: policy})
		if len(res.Unauthorized) == 0 {
			return "ok"
		}
		return res.Unauthorized[0].Reason
	}
	if got := run("pol-2", 9); got != "ok" {
		t.Fatalf("before change: %s", got)
	}
	if got := run("pol-2", 10); got != "OUTSIDE_VALIDITY" {
		t.Fatalf("at change: %s", got)
	}
	if got := run("", 9); got != "POLICY_NOT_PRESENTED" {
		t.Fatalf("no policy: %s", got)
	}
	g := newRC()
	g.reserve("d-1", 1, 60, 2)
	res := Reconcile(ReconcileInput{Core: g.core, History: g.history, Plans: map[string]model.Plan{"plan-1": g.plan},
		Runs: []LedgerRun{g.launch("d-1", 9)}, PolicyDigest: "pol-3"})
	if len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "POLICY_REVOCATION_UNPROVEN" {
		t.Fatalf("unobserved change: %+v", res)
	}
}

// Liz D4 r2 F-01: a review dispatch ends at its generation's policy
// revocation, at a correction generation of its unit, and at its
// generation's wall-time end; it needs a presented current policy.
func TestReviewValidityGenerationBounds(t *testing.T) {
	policy := func(f *rcFixture) {
		f.host(Record{Kind: KindPolicyRevocation, Timestamp: rcAt(10), PolicyRevocation: &PolicyRevocation{PlanPolicyDigest: "pol", ObservedPolicyDigest: "pol-new"}})
	}
	correction := func(f *rcFixture) {
		f.approve("U2", model.StageImplementationAuthorization, model.StageImplementationReview, 6)
		f.approve("U2", model.StageImplementationReview, model.StageVerification, 7)
		f.approve("U2", model.StageVerification, model.StageIndependentValidation, 8)
		f.approve("U2", model.StageIndependentValidation, model.StageImplementationAuthorization, 10)
	}
	wall := func(f *rcFixture) { f.plan.Units[1].Budgets.MaxWallMinutes = 10 }
	for name, edit := range map[string]func(*rcFixture){"policy": policy, "correction": correction, "wall": wall} {
		for _, at := range []int{9, 10, 11} {
			f := newRC()
			f.review("r-1", 5)
			edit(f)
			cur := "pol"
			if name == "policy" {
				cur = "pol-new"
			}
			res := Reconcile(ReconcileInput{Core: f.core, History: f.history, Plans: map[string]model.Plan{"plan-1": f.plan},
				PolicyDigest: cur, Runs: []LedgerRun{reviewLaunch("r-1", at)}})
			if ok := len(res.Authorized) == 1; ok != (at < 10) {
				t.Fatalf("%s at %d: %+v", name, at, res)
			}
		}
	}
	f := newRC()
	f.review("r-1", 5)
	res := Reconcile(ReconcileInput{Core: f.core, History: f.history, Plans: map[string]model.Plan{"plan-1": f.plan},
		Runs: []LedgerRun{reviewLaunch("r-1", 6)}})
	if len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "POLICY_NOT_PRESENTED" {
		t.Fatalf("no policy: %+v", res)
	}
	res = Reconcile(ReconcileInput{Core: f.core, History: f.history, PolicyDigest: "pol", Runs: []LedgerRun{reviewLaunch("r-1", 6)}})
	if len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "GENERATION_PLAN_MISSING" {
		t.Fatalf("no plan: %+v", res)
	}
}

// Liz D4 r2 F-02: a policy revocation applies to generations issued at or
// before the core head it was recorded at, not to a later approved
// generation; one with an unknown core head applies to all.
func TestPolicyRevokedAtGeneration(t *testing.T) {
	f := newRC()
	f.host(Record{Kind: KindPolicyRevocation, Timestamp: rcAt(10), PolicyRevocation: &PolicyRevocation{PlanPolicyDigest: "pol", ObservedPolicyDigest: "x"}})
	if at, ok := PolicyRevokedAt(f.core, f.history, "pol", 2); !ok || at.Format(time.RFC3339Nano) != rcAt(10) {
		t.Fatalf("issuing generation: %v %v", at, ok)
	}
	if _, ok := PolicyRevokedAt(f.core, f.history, "other", 2); ok {
		t.Fatal("other policy revoked")
	}
	f.approve("U2", model.StageImplementationAuthorization, model.StageImplementationReview, 11)
	f.approve("U2", model.StageIndependentValidation, model.StageImplementationAuthorization, 12)
	if _, ok := PolicyRevokedAt(f.core, f.history, "pol", 4); ok {
		t.Fatal("later approved generation revoked")
	}
	f.history[len(f.history)-1].CoreHeadAnchor = "unknown"
	if at, ok := PolicyRevokedAt(f.core, f.history, "pol", 4); !ok || !at.IsZero() {
		t.Fatalf("unknown core head: %v %v", at, ok)
	}
}

// Liz D4 r2 F-03: a launch is authorized only on the exact route, packet,
// and target its record binds; each field is checked on its own.
func TestLaunchIdentityMustMatchRecord(t *testing.T) {
	edits := map[string]func(*LaunchIdentity){
		"control":  func(*LaunchIdentity) {},
		"agent":    func(l *LaunchIdentity) { l.Agent = rcImpl.WorkerID },
		"model":    func(l *LaunchIdentity) { l.Model = rcImpl.Model },
		"provider": func(l *LaunchIdentity) { l.Provider = rcImpl.Provider },
		"effort":   func(l *LaunchIdentity) { l.Effort = "high" },
		"packet":   func(l *LaunchIdentity) { l.PacketSHA256 = "other" },
		"root":     func(l *LaunchIdentity) { l.ProjectRoot = "/other" },
		"head":     func(l *LaunchIdentity) { l.LaunchHead = "other" },
		"worktree": func(l *LaunchIdentity) { l.LaunchWorktreeSHA256 = "other" },
	}
	for name, edit := range edits {
		f := newRC()
		f.review("r-1", 5)
		run := reviewLaunch("r-1", 6)
		edit(&run.Launch)
		res := f.run(run)
		if ok := len(res.Authorized) == 1; ok != (name == "control") {
			t.Fatalf("review %s: %+v", name, res)
		}
		if name != "control" && res.Unauthorized[0].Reason != "ROUTE_MISMATCH" {
			t.Fatalf("review %s reason: %+v", name, res)
		}
	}
	for name, edit := range map[string]func(*LaunchIdentity){
		"control": func(*LaunchIdentity) {},
		"agent":   func(l *LaunchIdentity) { l.Agent = "other" },
		"model":   func(l *LaunchIdentity) { l.Model = "other" },
		"effort":  func(l *LaunchIdentity) { l.Effort = "other" },
	} {
		f := newRC()
		f.reserve("impl", 1, 60, 2)
		run := f.launch("impl", 2)
		edit(&run.Launch)
		if ok := len(f.run(run).Authorized) == 1; ok != (name == "control") {
			t.Fatalf("reservation %s", name)
		}
	}
	// A reservation whose route no implementation preflight names is refused.
	f := newRC()
	f.reserve("impl", 1, 60, 2)
	f.history[len(f.history)-1].Reservation.RouteIdentity = "unknown"
	if res := f.run(f.launch("impl", 2)); len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "ROUTE_MISMATCH" {
		t.Fatalf("unknown route: %+v", res)
	}
}

// Liz D4 r2 F-01: a review dispatch with no issuing lease generation at its
// core head is refused.
func TestReviewWithoutGenerationRefused(t *testing.T) {
	f := newRC()
	f.core = f.core[:1]
	f.review("r-1", 5)
	if res := f.run(reviewLaunch("r-1", 6)); len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "GENERATION_MISSING" {
		t.Fatalf("%+v", res)
	}
}

// Liz D4 r3 F-03: revocations count from the review's generation issuer, so
// a dependency rewind before the packet was issued still revokes it; an
// unrelated unit's rewind does not.
func TestReviewIssuedAfterDependencyRewind(t *testing.T) {
	for _, tc := range []struct {
		unit string
		to   model.Stage
		ok   bool
	}{{"U1", model.StageRouteSelection, false}, {"U1", model.StagePlanning, false}, {"U3", model.StageRouteSelection, true}} {
		f := newRC()
		f.plan.Units = append(f.plan.Units, model.PlanUnit{ID: "U3", Dependencies: []string{}, Budgets: rcBudgets})
		f.approve(tc.unit, model.StageImplementationAuthorization, tc.to, 3)
		f.review("r-1", 5)
		if res := f.run(reviewLaunch("r-1", 6)); (len(res.Authorized) == 1) != tc.ok {
			t.Fatalf("%s to %s: %+v", tc.unit, tc.to, res)
		}
	}
}

// Liz D4 r4: every imported journal, even one without attempts, binds
// exactly one admission, and declares its namespace for reconciliation.
func TestEveryJournalBindsAdmission(t *testing.T) {
	f := newRC()
	f.host(Record{Kind: KindLoopEvidence, LoopEvidence: &LoopEvidence{UnitID: "U2", ControllerRunID: "vl-1", EnvelopeSHA256: "env",
		LedgerNamespace: f.ns, LiveRoot: "/repo", CloneBaselineDigest: "base"}})
	if res := f.run(); len(res.Unauthorized) != 0 {
		t.Fatalf("matching empty journal: %+v", res)
	}
	g := &rcFixture{core: f.core[:2], plan: f.plan, ns: f.ns}
	g.host(Record{Kind: KindLoopEvidence, LoopEvidence: &LoopEvidence{UnitID: "U2", ControllerRunID: "unknown", EnvelopeSHA256: "env",
		LedgerNamespace: "/ledger/other", LiveRoot: "/repo", CloneBaselineDigest: "base"}})
	if res := g.run(); len(res.Unauthorized) != 1 || res.Unauthorized[0].Reason != "JOURNAL_ADMISSION_MISSING" {
		t.Fatalf("unadmitted empty journal: %+v", res)
	}
	if ns := Namespaces(g.history); len(ns) != 1 || ns[0] != "/ledger/other" {
		t.Fatalf("journal namespace: %v", ns)
	}
}
