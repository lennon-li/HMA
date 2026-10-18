package engine

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

var leaseT0 = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

type leaseFixture struct {
	t                *testing.T
	records          []model.Record
	plan, contract   []byte
	planDigest, cdig string
	seq              int64
	routeApproval    map[string]string
}

func newLeaseFixture(t *testing.T) *leaseFixture {
	c := model.AcceptanceContract{Schema: model.AcceptanceContractSchema, RunID: "run", Intent: "i", Risk: model.RouteRiskLow, NonGoals: []string{"none"}, Deliverables: []string{"code"},
		Criteria: []model.ContractCriterion{{ID: "C1", Statement: "s", Threshold: "exit 0", EvidenceRequirement: "e",
			Check: model.Check{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}}},
		Budgets:         model.ContractBudgets{MaxAttempts: 5, MaxWallMinutes: 120, CostCeiling: 100, CostUnit: "relative"},
		PermittedRoutes: []string{"impl", "esc", "val"}, PermissionEnvelope: []string{model.PermissionBoundedWrite}, PolicyDigest: "policy"}
	f := &leaseFixture{t: t, routeApproval: map[string]string{}}
	f.contract, _ = json.Marshal(c)
	_, f.cdig, _ = model.ParseAcceptanceContract(f.contract)
	unit := func(id string, deps ...string) model.PlanUnit {
		return model.PlanUnit{ID: id, Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"internal/x.go"}, AllowedActions: []string{"edit"},
			ForbiddenFiles: []string{"docs/architecture.md"}, ForbiddenActions: []string{"git push"}, PreChangeEvidence: "failing test", MinimumImplementation: "one function",
			ReuseAnalysis: "none applicable", PermissionClasses: []string{model.PermissionBoundedWrite}, ImplementerRoute: "impl", EscalationRoute: "esc", ValidatorRoute: "val",
			IndependenceRequirement: "different family", VerificationCommands: []model.Check{{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}},
			RequiredEvidence: []string{"go test"}, StopConditions: []string{"budget"}, RollbackConditions: []string{"regression"}, EscalationConditions: []string{"two failures"},
			NonGoals: []string{"refactor"}, Dependencies: append([]string{}, deps...),
			Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 200, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 60, CostCeiling: 10}}
	}
	p := model.Plan{Schema: model.PlanSchema, RunID: "run", AcceptanceContractDigest: f.cdig, PolicyDigest: "policy",
		Units: []model.PlanUnit{unit("U1"), unit("U2", "U1"), unit("U3", "U2"), unit("U4")}}
	f.plan, _ = json.Marshal(p)
	_, f.planDigest, _ = model.ParsePlan(f.plan)
	f.seq++
	f.records = append(f.records, model.Record{Kind: model.KindCriterion, Version: 1, Stage: model.StageAcceptanceCriteria,
		Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}},
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: "anchor-criteria", Timestamp: leaseT0.Format(time.RFC3339Nano)}})
	f.approve("", model.StageAcceptanceCriteria, model.StagePlanning, leaseT0)
	return f
}

// approve appends one approved transition for unit (or the whole run).
func (f *leaseFixture) approve(unit string, from, to model.Stage, at time.Time) {
	f.seq++
	f.records = append(f.records, model.Record{UnitID: unit, Kind: model.KindStageTransition, Version: 1, Stage: from, Control: model.ControlApproved,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: fmt.Sprintf("anchor-%d", f.seq), Timestamp: at.Format(time.RFC3339Nano)},
		Approval: &model.ApprovalBinding{UnitID: unit, RunID: "run", CurrentStage: from, ProposedTargetStage: to,
			AcceptedPlanDigest: f.planDigest, AcceptanceContractDigest: f.cdig, ChallengeNonce: fmt.Sprintf("n%d", f.seq),
			TransitionDigest: fmt.Sprintf("transition-%d", f.seq), Timestamp: at.Format(time.RFC3339)}})
}

func (f *leaseFixture) authorize(unit string, at time.Time) {
	f.approve(unit, model.StageRouteSelection, model.StageImplementationAuthorization, at)
	f.routeApproval[unit] = f.records[len(f.records)-1].Approval.TransitionDigest
}

func (f *leaseFixture) grant(unit string) *model.RouteGrant {
	return &model.RouteGrant{Identity: "impl", PermissionClasses: []string{model.PermissionBoundedWrite}, ApprovalDigest: f.routeApproval[unit]}
}

func (f *leaseFixture) derive(unit string, now time.Time, c LeaseCounters) (Lease, Reason) {
	return DeriveLease(LeaseRequest{Records: f.records, UnitID: unit, Plan: f.plan, Contract: f.contract, PolicyDigest: "policy", Counters: c, Now: now, Route: f.grant(unit)})
}

func (f *leaseFixture) want(unit string, now time.Time, want Reason) Lease {
	f.t.Helper()
	l, got := f.derive(unit, now, LeaseCounters{})
	if got != want {
		f.t.Fatalf("%s: reason %q, want %q", unit, got, want)
	}
	return l
}

func TestNoLaunchBeforeRouteApproval(t *testing.T) {
	f := newLeaseFixture(t)
	f.want("U1", leaseT0, ReasonLeaseNoIssuingApproval)
	// A route approval for another unit grants this one nothing.
	f.authorize("U4", leaseT0)
	f.want("U1", leaseT0, ReasonLeaseNoIssuingApproval)
	f.authorize("U1", leaseT0)
	f.want("U1", leaseT0.Add(time.Minute), ReasonNone)
}

func TestPlanDigestMismatchRefused(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	var p model.Plan
	json.Unmarshal(f.plan, &p)
	p.Units[0].Budgets.MaxAttempts = 2
	f.plan, _ = json.Marshal(p)
	f.want("U1", leaseT0, ReasonLeasePlanMismatch)
	f.plan = []byte(`{"schema":"hma.plan/v1"}`)
	f.want("U1", leaseT0, ReasonLeasePlanInvalid)
	f.plan = nil
	f.want("U1", leaseT0, ReasonLeasePlanInvalid)
}

func TestLeaseNeedsBoundContract(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	var c model.AcceptanceContract
	json.Unmarshal(f.contract, &c)
	c.Intent = "changed"
	f.contract, _ = json.Marshal(c)
	f.want("U1", leaseT0, ReasonLeaseContractMismatch)
}

// TestHostWrittenLeaseGrantsNothing: derivation reads only core records and
// the presented documents; no host record and no unit in an unlisted plan
// can produce a lease.
func TestHostWrittenLeaseGrantsNothing(t *testing.T) {
	f := newLeaseFixture(t)
	f.want("U9", leaseT0, ReasonLeaseNoIssuingApproval)
	// A core record of another kind naming the unit grants nothing.
	f.seq++
	f.records = append(f.records, model.Record{UnitID: "U1", Kind: model.KindEvidence, Version: 1, Stage: model.StageImplementationAuthorization,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: "anchor-ev", Timestamp: leaseT0.Format(time.RFC3339Nano)}})
	f.want("U1", leaseT0, ReasonLeaseNoIssuingApproval)
	// An approval of an illegal edge into Implementation authorization is no issuer.
	f.approve("U1", model.StageImplementationReview, model.StageImplementationAuthorization, leaseT0)
	f.want("U1", leaseT0, ReasonLeaseNoIssuingApproval)
	// A unit outside the plan cannot hold a lease even with an approval.
	f.authorize("U9", leaseT0)
	f.want("U9", leaseT0, ReasonLeaseUnitNotInPlan)
}

func TestLeaseVoidAfterReviewTransition(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	f.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0.Add(time.Minute))
	f.want("U1", leaseT0.Add(2*time.Minute), ReasonLeaseStage)
}

func TestLeaseVoidAfterRewindOfDependency(t *testing.T) {
	f := newLeaseFixture(t)
	for _, u := range []string{"U1", "U2", "U3", "U4"} {
		f.authorize(u, leaseT0)
	}
	// U1 returns to Route selection: U2 (direct) and U3 (transitive) are
	// void, U4 (independent) stays active.
	f.approve("U1", model.StageImplementationReview, model.StageRouteSelection, leaseT0.Add(time.Minute))
	now := leaseT0.Add(2 * time.Minute)
	f.want("U1", now, ReasonLeaseRewound)
	f.want("U2", now, ReasonLeaseRewound)
	f.want("U3", now, ReasonLeaseRewound)
	f.want("U4", now, ReasonNone)
	// A fresh issuing approval starts a new generation.
	old := f.want("U4", now, ReasonNone)
	f.authorize("U2", now)
	l := f.want("U2", now, ReasonNone)
	if l.IssuerSequence <= old.IssuerSequence || l.LeaseDigest == old.LeaseDigest {
		t.Fatal("new issuing approval did not start a new generation")
	}
	// A whole-run return to Planning voids every lease.
	f.approve("", model.StageVerification, model.StagePlanning, now)
	f.want("U4", now, ReasonLeaseRewound)
}

func TestLeaseVoidTerminalRun(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	f.seq++
	f.records = append(f.records, model.Record{Kind: model.KindOutcome, Outcome: model.OutcomeAborted, Version: 1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: "anchor-end", Timestamp: leaseT0.Format(time.RFC3339Nano)}})
	f.want("U1", leaseT0, ReasonLeaseRunTerminal)
}

func TestLeaseVoidAfterWallTime(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	l := f.want("U1", leaseT0.Add(60*time.Minute-time.Nanosecond), ReasonNone)
	if l.ValidUntil != leaseT0.Add(60*time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("valid_until %s", l.ValidUntil)
	}
	f.want("U1", leaseT0.Add(60*time.Minute), ReasonLeaseWallTime)
}

func TestPolicyDigestChangeVoids(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	for _, p := range []string{"", "policy-v2"} {
		if _, r := DeriveLease(LeaseRequest{Records: f.records, UnitID: "U1", Plan: f.plan, Contract: f.contract, PolicyDigest: p, Now: leaseT0, Route: f.grant("U1")}); r != ReasonLeasePolicyChanged {
			t.Fatalf("policy %q: %s", p, r)
		}
	}
}

func TestCeilingComparison(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	for _, tc := range []struct {
		c    LeaseCounters
		want Reason
	}{
		{LeaseCounters{Attempts: 2, CostSpent: 9}, ReasonNone},
		{LeaseCounters{Attempts: 3}, ReasonLeaseCeiling},
		{LeaseCounters{CostSpent: 10}, ReasonLeaseCeiling},
	} {
		if _, r := f.derive("U1", leaseT0, tc.c); r != tc.want {
			t.Fatalf("%+v: %s, want %s", tc.c, r, tc.want)
		}
	}
}

// TestLeaseWalkThroughR1 is Gate A v4 §1: no lease before the issuing
// approval; active in Implementation authorization; void after the move to
// Implementation review and at Verification; active again only after a legal
// rework edge (Independent validation -> Implementation authorization).
func TestLeaseWalkThroughR1(t *testing.T) {
	f := newLeaseFixture(t)
	at := func(m int) time.Time { return leaseT0.Add(time.Duration(m) * time.Minute) }
	f.approve("U1", model.StagePlanning, model.StageRouteSelection, at(0))
	f.want("U1", at(0), ReasonLeaseNoIssuingApproval)
	f.authorize("U1", at(1))
	f.want("U1", at(2), ReasonNone)
	f.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, at(3))
	f.want("U1", at(3), ReasonLeaseStage)
	f.approve("U1", model.StageImplementationReview, model.StageVerification, at(4))
	f.want("U1", at(4), ReasonLeaseStage)
	f.approve("U1", model.StageVerification, model.StageIndependentValidation, at(5))
	f.want("U1", at(5), ReasonLeaseStage)
	f.approve("U1", model.StageIndependentValidation, model.StageImplementationAuthorization, at(6))
	l := f.want("U1", at(7), ReasonNone)
	if l.IssuerStage != model.StageIndependentValidation {
		t.Fatalf("issuer stage %s", l.IssuerStage)
	}
	if r := EvaluateTransition(TransitionRequest{Source: model.StageImplementationReview, TargetStage: model.StageImplementationAuthorization}); r.Decision == DecisionLegalPendingApproval {
		t.Fatal("IMPLEMENTATION_REVIEW -> IMPLEMENTATION_AUTHORIZATION must stay illegal")
	}
}

// Liz D1+D2 F02: a dependency's correction after independent validation
// voids dependent leases issued before it.
func TestLeaseVoidAfterDependencyCorrection(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	f.authorize("U3", leaseT0)
	f.approve("U2", model.StageIndependentValidation, model.StageImplementationAuthorization, leaseT0.Add(time.Minute))
	f.want("U3", leaseT0.Add(2*time.Minute), ReasonLeaseRewound)
	f.want("U1", leaseT0.Add(2*time.Minute), ReasonNone)
}

// F03: wall time runs from the approval timestamp, and no lease exists
// before it.
func TestLeaseWallTimeFromApproval(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	// HMA appended the approval two minutes after the human gave it.
	f.records[len(f.records)-1].Metadata.Timestamp = leaseT0.Add(2 * time.Minute).Format(time.RFC3339Nano)
	f.want("U1", leaseT0.Add(60*time.Minute), ReasonLeaseWallTime)
	f.want("U1", leaseT0.Add(-time.Second), ReasonLeaseNotYetIssued)
}

// F07: a change of the core criterion set after acceptance voids leases.
func TestLeaseVoidAfterCriteriaChange(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	f.seq++
	f.records = append(f.records, model.Record{Kind: model.KindCriterion, Version: 1, Stage: model.StageImplementationAuthorization,
		Criteria: []model.Criterion{{ID: "OTHER", Disposition: model.CriterionPending}},
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: "anchor-c2", Timestamp: leaseT0.Format(time.RFC3339Nano)}})
	f.want("U1", leaseT0, ReasonLeaseCriteriaChanged)
}

// F08: the lease digest is over {transition digest, unit, plan digest}.
func TestLeaseDigestFields(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	l := f.want("U1", leaseT0, ReasonNone)
	f.records[len(f.records)-1].Metadata.HeadAnchor = "another-anchor"
	if f.want("U1", leaseT0, ReasonNone).LeaseDigest != l.LeaseDigest {
		t.Fatal("lease digest depends on the record anchor")
	}
	f.records[len(f.records)-1].Approval.TransitionDigest = "another-transition"
	f.routeApproval["U1"] = "another-transition"
	if f.want("U1", leaseT0, ReasonNone).LeaseDigest == l.LeaseDigest {
		t.Fatal("lease digest ignores the transition digest")
	}
}

// F09: negative counters are refused.
func TestLeaseNegativeCountersRefused(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	for _, c := range []LeaseCounters{{Attempts: -1}, {CostSpent: -1}} {
		if _, r := f.derive("U1", leaseT0, c); r != ReasonLeaseCountersInvalid {
			t.Fatalf("%+v: %s", c, r)
		}
	}
}

// Liz D1+D2 r3: a return to Grounding or Acceptance criteria voids every
// lease even when tagged with an unrelated unit; Planning stays scoped by the
// plan graph.
func TestLeaseVoidAfterWholeRunRewindTaggedWithOtherUnit(t *testing.T) {
	for _, target := range []model.Stage{model.StageGrounding, model.StageAcceptanceCriteria} {
		f := newLeaseFixture(t)
		f.authorize("U1", leaseT0)
		f.authorize("U4", leaseT0)
		f.approve("U4", model.StageImplementationAuthorization, target, leaseT0.Add(time.Minute))
		f.want("U1", leaseT0.Add(2*time.Minute), ReasonLeaseRewound)
	}
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	f.authorize("U4", leaseT0)
	f.approve("U4", model.StageImplementationAuthorization, model.StagePlanning, leaseT0.Add(time.Minute))
	f.want("U1", leaseT0.Add(2*time.Minute), ReasonNone)
}

// Liz D1+D2 r4: the selected route must be the plan unit's implementer
// route, within its permission grant, and reference the unit's route
// approval (kept across an Independent validation correction).
func TestLeaseRouteGrant(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	derive := func(g *model.RouteGrant) Reason {
		_, r := DeriveLease(LeaseRequest{Records: f.records, UnitID: "U1", Plan: f.plan, Contract: f.contract, PolicyDigest: "policy", Now: leaseT0, Route: g})
		return r
	}
	ok := f.grant("U1")
	if r := derive(ok); r != ReasonNone {
		t.Fatal(r)
	}
	for name, g := range map[string]*model.RouteGrant{
		"absent":         nil,
		"other-identity": {Identity: "esc", PermissionClasses: ok.PermissionClasses, ApprovalDigest: ok.ApprovalDigest},
		"wider":          {Identity: "impl", PermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionReadOnly}, ApprovalDigest: ok.ApprovalDigest},
		"no-permissions": {Identity: "impl", ApprovalDigest: ok.ApprovalDigest},
		"other-approval": {Identity: "impl", PermissionClasses: ok.PermissionClasses, ApprovalDigest: "transition-other"},
	} {
		if r := derive(g); r != ReasonLeaseRouteMismatch {
			t.Errorf("%s: %s", name, r)
		}
	}
	// A correction from Independent validation keeps the route approval.
	f.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0)
	f.approve("U1", model.StageImplementationReview, model.StageVerification, leaseT0)
	f.approve("U1", model.StageVerification, model.StageIndependentValidation, leaseT0)
	f.approve("U1", model.StageIndependentValidation, model.StageImplementationAuthorization, leaseT0)
	if r := derive(ok); r != ReasonNone {
		t.Fatalf("correction issuer: %s", r)
	}
}

// waiverOp appends a waiver operation and returns its record digest.
func (f *leaseFixture) waiverOp(op model.WaiverOperationAction, target string) string {
	f.seq++
	anchor := fmt.Sprintf("waiver-%d", f.seq)
	f.records = append(f.records, model.Record{Kind: model.KindWaiverOperation, Version: 1, Stage: model.StageImplementationAuthorization,
		WaiverOperation: &model.WaiverOperation{Operation: op, Scope: model.ScopeSelector{Kind: model.ScopeFinding, Target: target}, Approver: "human",
			ChallengeNonce: anchor, Justification: "j", Timestamp: leaseT0.Format(time.RFC3339)},
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: anchor, Timestamp: leaseT0.Format(time.RFC3339Nano)}})
	return anchor
}

// Liz D1+D2 r5: a later withdrawal, expiry, or replacement of a waiver the
// issuing approval bound voids the lease, and a regrant does not revive it.
func TestLeaseVoidAfterBoundWaiverChange(t *testing.T) {
	for _, op := range []model.WaiverOperationAction{model.WaiverOperationWithdraw, model.WaiverOperationExpire, model.WaiverOperationGrant} {
		t.Run(string(op), func(t *testing.T) {
			f := newLeaseFixture(t)
			g := f.waiverOp(model.WaiverOperationGrant, "F1")
			f.authorize("U1", leaseT0)
			f.records[len(f.records)-1].Approval.ActiveWaivers = model.WaiverApplicabilitySet{g}
			f.want("U1", leaseT0, ReasonNone)
			f.waiverOp(model.WaiverOperationGrant, "F2") // unrelated scope
			f.want("U1", leaseT0, ReasonNone)
			f.waiverOp(op, "F1")
			f.want("U1", leaseT0, ReasonLeaseWaiverChanged)
			f.waiverOp(model.WaiverOperationGrant, "F1")
			f.want("U1", leaseT0, ReasonLeaseWaiverChanged)
		})
	}
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	f.records[len(f.records)-1].Approval.ActiveWaivers = model.WaiverApplicabilitySet{"no-such-grant"}
	f.want("U1", leaseT0, ReasonLeaseWaiverChanged)
}

// Liz D1+D2 r6: every bound waiver digest must name the live grant of its
// scope at the issuer; a withdrawn, expired, replaced, or repeated reference
// voids the lease.
func TestLeaseBoundWaiverMustBeLiveAtIssuer(t *testing.T) {
	for name, setup := range map[string]func(f *leaseFixture) model.WaiverApplicabilitySet{
		"withdrawn": func(f *leaseFixture) model.WaiverApplicabilitySet {
			g := f.waiverOp(model.WaiverOperationGrant, "F1")
			f.waiverOp(model.WaiverOperationWithdraw, "F1")
			return model.WaiverApplicabilitySet{g}
		},
		"expired": func(f *leaseFixture) model.WaiverApplicabilitySet {
			g := f.waiverOp(model.WaiverOperationGrant, "F1")
			f.waiverOp(model.WaiverOperationExpire, "F1")
			return model.WaiverApplicabilitySet{g}
		},
		"replaced": func(f *leaseFixture) model.WaiverApplicabilitySet {
			g := f.waiverOp(model.WaiverOperationGrant, "F1")
			f.waiverOp(model.WaiverOperationGrant, "F1")
			return model.WaiverApplicabilitySet{g}
		},
		"duplicate": func(f *leaseFixture) model.WaiverApplicabilitySet {
			g := f.waiverOp(model.WaiverOperationGrant, "F1")
			return model.WaiverApplicabilitySet{g, g}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLeaseFixture(t)
			bound := setup(f)
			f.authorize("U1", leaseT0)
			f.records[len(f.records)-1].Approval.ActiveWaivers = bound
			f.want("U1", leaseT0, ReasonLeaseWaiverChanged)
		})
	}
	f := newLeaseFixture(t)
	f.waiverOp(model.WaiverOperationGrant, "F1")
	g := f.waiverOp(model.WaiverOperationGrant, "F1")
	f.authorize("U1", leaseT0)
	f.records[len(f.records)-1].Approval.ActiveWaivers = model.WaiverApplicabilitySet{g}
	f.want("U1", leaseT0, ReasonNone)
}

// Liz D1+D2 r7: a correction issuer cannot reuse a route approval that a
// waiver change has invalidated.
func TestCorrectionLeaseNeedsValidRouteApproval(t *testing.T) {
	f := newLeaseFixture(t)
	g := f.waiverOp(model.WaiverOperationGrant, "F1")
	f.authorize("U1", leaseT0)
	f.records[len(f.records)-1].Approval.ActiveWaivers = model.WaiverApplicabilitySet{g}
	f.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0)
	f.approve("U1", model.StageImplementationReview, model.StageVerification, leaseT0)
	f.approve("U1", model.StageVerification, model.StageIndependentValidation, leaseT0)
	f.waiverOp(model.WaiverOperationWithdraw, "F1")
	f.approve("U1", model.StageIndependentValidation, model.StageImplementationAuthorization, leaseT0)
	f.want("U1", leaseT0, ReasonLeaseWaiverChanged)
	// A fresh route approval restores a lease.
	f.approve("U1", model.StageImplementationAuthorization, model.StageRouteSelection, leaseT0)
	f.authorize("U1", leaseT0)
	f.want("U1", leaseT0, ReasonNone)
}

// Liz D1+D2 r8: a correction issuer cannot reuse a route approval that a
// rewind between it and the correction invalidated.
func TestCorrectionLeaseRouteApprovalSurvivesNoRewind(t *testing.T) {
	walk := func(f *leaseFixture, mid func()) {
		f.authorize("U1", leaseT0)
		f.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0)
		f.approve("U1", model.StageImplementationReview, model.StageVerification, leaseT0)
		f.approve("U1", model.StageVerification, model.StageIndependentValidation, leaseT0)
		mid()
		f.approve("U1", model.StageIndependentValidation, model.StageImplementationAuthorization, leaseT0)
	}
	for name, mid := range map[string]func(f *leaseFixture){
		"whole-run-grounding": func(f *leaseFixture) { f.approve("", model.StageIndependentValidation, model.StageGrounding, leaseT0) },
		"other-unit-criteria": func(f *leaseFixture) {
			f.approve("U4", model.StageImplementationAuthorization, model.StageAcceptanceCriteria, leaseT0)
		},
		"own-planning": func(f *leaseFixture) { f.approve("U1", model.StageIndependentValidation, model.StagePlanning, leaseT0) },
	} {
		t.Run(name, func(t *testing.T) {
			f := newLeaseFixture(t)
			walk(f, func() { mid(f) })
			if _, r := f.derive("U1", leaseT0, LeaseCounters{}); r == ReasonNone {
				t.Fatal("correction lease active after an intervening rewind")
			}
		})
	}
	// U3 depends on U2: U2 returning to Planning between U3's route
	// approval and U3's correction voids it.
	f := newLeaseFixture(t)
	f.authorize("U3", leaseT0)
	f.approve("U3", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0)
	f.approve("U3", model.StageImplementationReview, model.StageVerification, leaseT0)
	f.approve("U3", model.StageVerification, model.StageIndependentValidation, leaseT0)
	f.approve("U2", model.StageImplementationAuthorization, model.StagePlanning, leaseT0)
	f.approve("U3", model.StageIndependentValidation, model.StageImplementationAuthorization, leaseT0)
	f.want("U3", leaseT0, ReasonLeaseRewound)
	// Control: the plain correction walk stays active.
	g := newLeaseFixture(t)
	walk(g, func() {})
	g.want("U1", leaseT0, ReasonNone)
}
