package main

import (
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

// fixtureUnitPermissionClasses is the permission classes seedCheckpointBoundRun
// (cmd/hma/gatee_e0_test.go) always gives U1: it includes
// model.PermissionNoExternalEffect, so T4's allowed form never fires in these
// fixtures, and it never changes across a run, so T1 never fires either.
var fixtureUnitPermissionClasses = []string{model.PermissionReadOnly, model.PermissionBoundedWrite, model.PermissionNoExternalEffect}

// ordinaryOnlyPolicy classifies the fixture's only changed path ("a") as
// "ordinary" only: a class other than "ordinary" would trigger, but
// "ordinary" alone does not, and a path a rule matches is not UNCLASSIFIED
// (T8) either. With contract risk low and no material milestone, this is the
// no-trigger case (Decision D8 (1)).
func ordinaryOnlyPolicy() *model.CheckpointPolicy {
	return &model.CheckpointPolicy{Schema: model.CheckpointPolicySchema, Rules: []model.CheckpointPolicyRule{
		{ID: "ord", Glob: "a", Classes: []string{"ordinary"}},
	}}
}

// classifyForFixture reproduces engine.ClassifyCheckpoint's gate-time call in
// requireCheckpointGate for a seedCheckpointBoundRun fixture: the run never
// re-accepts its contract or changes U1's permission classes, so Baseline
// equals Current, and the only changed path is ciRepo's "a".
func classifyForFixture(cb *checkpointRun, risk model.RouteRiskLevel, policy *model.CheckpointPolicy, milestone bool) engine.CheckpointDecision {
	baseline := engine.CheckpointBaseline{ContractDigest: cb.contractDigest, PermissionClasses: sortedCopy(fixtureUnitPermissionClasses)}
	return engine.ClassifyCheckpoint(engine.CheckpointInput{
		Contract: &model.AcceptanceContract{Risk: risk, CheckpointPolicyDigest: cb.checkpointPolicyDigest},
		Policy:   policy, Unit: model.PlanUnit{PermissionClasses: fixtureUnitPermissionClasses, MaterialMilestone: milestone},
		Baseline: baseline, Current: baseline, Manifest: []engine.ChangedPath{{Path: "a"}}, Stage: model.StageIndependentValidation,
	})
}

func triggerStrings(triggers []engine.CheckpointTrigger) []string {
	out := make([]string, 0, len(triggers))
	for _, tr := range triggers {
		out = append(out, string(tr))
	}
	return out
}

// checkpointReviewSpec is one synthetic host dispatch/review pair, exactly
// as requireCheckpointGate reads it off history: a Dispatch record (role
// REVIEW) carrying the §3.5a packet binding fields, and an approving Review
// record answering it. Leaving reportDigest empty models a review whose
// findings were never (re-)imported after this record, the same as a
// legacy/manual "hma host review" result; requireCheckpointGate's findings-
// snapshot check only applies when it is set.
type checkpointReviewSpec struct {
	dispatchDigest string
	coreHeadAnchor string
	risk           model.RouteRiskLevel
	reason         hoststate.IssuingReason
	triggers       []string
	policyDigest   string
	contractDigest string
	artifact       string
	verdict        hoststate.ReviewVerdict
	isolation      string
	noSchema       bool
}

func checkpointReviewRecords(unit string, spec checkpointReviewSpec) []hoststate.Record {
	var packet *hoststate.ReviewPacketBinding
	if spec.noSchema {
		packet = &hoststate.ReviewPacketBinding{IssuingReason: spec.reason}
	} else {
		packet = &hoststate.ReviewPacketBinding{IssuingReason: spec.reason, CheckpointSchema: hoststate.CheckpointSchemaV1,
			CheckpointTriggers: spec.triggers, CheckpointPolicyDigest: spec.policyDigest, BoundContractDigest: spec.contractDigest}
	}
	verdict := spec.verdict
	if verdict == "" {
		verdict = hoststate.ReviewApprove
	}
	isolation := spec.isolation
	if isolation == "" {
		isolation = "isolated"
	}
	return []hoststate.Record{
		{Kind: hoststate.KindDispatch, HeadAnchor: spec.dispatchDigest, CoreHeadAnchor: spec.coreHeadAnchor,
			Dispatch: &hoststate.Dispatch{UnitID: unit, Role: hoststate.RoleReview, Risk: spec.risk, Packet: packet}},
		{Kind: hoststate.KindReview, HeadAnchor: spec.dispatchDigest + "-review", CoreHeadAnchor: spec.coreHeadAnchor,
			Review: &hoststate.IndependentReview{UnitID: unit, ReviewDispatchDigest: spec.dispatchDigest, Verdict: verdict, Isolation: isolation, ArtifactDigest: spec.artifact}},
	}
}

// checkpointGateInput is the IV -> RELEASE_AND_CLOSURE transition.Input
// requireCheckpointGate reads for cb's unit U1, binding the live repository
// at cb.root/cb.base (clean checkout, DirtyWorktreeApproved false) and the
// approval's produced head/diff to cb.head/cb.diff, exactly what a human
// approving this edge after seedCheckpointBoundRun's atIV=true seeding would
// carry.
func checkpointGateInput(cb *checkpointRun) transition.Input {
	return transition.Input{RunID: "run", RepositoryRoot: cb.root, BaseRevision: cb.base, PolicyPath: cb.policy,
		ExpectedApproval: model.ApprovalBinding{UnitID: "U1", CurrentStage: model.StageIndependentValidation, ProposedTargetStage: model.StageReleaseAndClosure,
			ProducedHeadDigest: cb.head, DiffDigest: cb.diff}}
}

func checkpointGate(t *testing.T, cb *checkpointRun, history []hoststate.Record) error {
	t.Helper()
	return requireHostTransitionGateIn(cb.store, checkpointGateInput(cb), history, cb.contractPath, cb.checkpointPolicyPath)
}

// TestCheckpointNotRequiredWithoutTriggers is Decision D8 (1): a contract-
// bound unit at INDEPENDENT_VALIDATION on which no gate-time trigger fires
// needs no checkpoint review to exit; the checkpoint gate passes with no
// review on file at all.
func TestCheckpointNotRequiredWithoutTriggers(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, ordinaryOnlyPolicy(), false, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, ordinaryOnlyPolicy(), false)
	if len(decision.Triggers) != 0 {
		t.Fatalf("fixture triggers = %v, want none", decision.Triggers)
	}
	if err := checkpointGate(t, cb, nil); err != nil {
		t.Fatalf("checkpoint gate with no trigger and no review: %v", err)
	}
}

// TestCheckpointMissingAtIVExit is the G-V fix's core case: a trigger fires
// (T6 material milestone, T8 unclassified) and no checkpoint review exists
// at all, so the IV exit is refused.
func TestCheckpointMissingAtIVExit(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	if len(decision.Triggers) == 0 {
		t.Fatal("fixture triggers empty, want at least T6/T8")
	}
	err := checkpointGate(t, cb, nil)
	if err == nil || err.Error() != reasonCheckpointReviewMissing {
		t.Fatalf("checkpoint gate with no review: got %v, want %s", err, reasonCheckpointReviewMissing)
	}
}

// TestCheckpointPerChangeReviewCannotStandIn constructs a review dispatched
// before the unit's approved VERIFICATION -> INDEPENDENT_VALIDATION
// transition (core index 5 of the 7 seedCheckpointBoundRun(..., atIV: true)
// records; the IV approval itself is index 6): exactly the per-change review
// a coding change would have gone through at IMPLEMENTATION_REVIEW ->
// VERIFICATION. Even though it is otherwise well-formed (checkpoint schema,
// APPROVE, isolated, covering triggers), its dispatch predates the stage
// audit, so it cannot stand in for it.
func TestCheckpointPerChangeReviewCannotStandIn(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(core) != 7 {
		t.Fatalf("fixture core has %d records, want 7 (seedCheckpointBoundRun's fixed seeding)", len(core))
	}
	beforeIV := core[5].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: beforeIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""),
	})
	err = checkpointGate(t, cb, history)
	if err == nil || err.Error() != reasonCheckpointReviewMissing {
		t.Fatalf("checkpoint gate with a review dispatched before the IV transition: got %v, want %s", err, reasonCheckpointReviewMissing)
	}
}

// TestCheckpointLegacyReviewCannotStandIn is the same construction as a
// qualifying review (dispatched after the IV transition, APPROVE, isolated,
// covering triggers) except its packet carries no checkpoint_schema: a
// legacy or non-contract-run-style review. Decision D10: it never counts for
// E2.
func TestCheckpointLegacyReviewCannotStandIn(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	afterIV := core[len(core)-1].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""), noSchema: true,
	})
	err = checkpointGate(t, cb, history)
	if err == nil || err.Error() != reasonCheckpointReviewMissing {
		t.Fatalf("checkpoint gate with a legacy (no checkpoint_schema) review: got %v, want %s", err, reasonCheckpointReviewMissing)
	}
}

// TestCheckpointTriggersUncovered constructs a qualifying, correctly
// ordered, correctly staged review whose recorded checkpoint_triggers (T6
// only) are a strict subset of the gate-time triggers (T6 and T8, since this
// fixture carries no checkpoint policy): the review covered less than the
// live target now shows.
func TestCheckpointTriggersUncovered(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	foundT6, foundT8 := false, false
	for _, tr := range decision.Triggers {
		foundT6 = foundT6 || tr == engine.CheckpointMilestone
		foundT8 = foundT8 || tr == engine.CheckpointUnclassified
	}
	if !foundT6 || !foundT8 {
		t.Fatalf("fixture triggers = %v, want both T6 and T8", decision.Triggers)
	}
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	afterIV := core[len(core)-1].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: []string{string(engine.CheckpointMilestone)}, policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""),
	})
	err = checkpointGate(t, cb, history)
	if err == nil || err.Error() != reasonCheckpointTriggersUncovered {
		t.Fatalf("checkpoint gate with a recorded trigger subset: got %v, want %s", err, reasonCheckpointTriggersUncovered)
	}
}

// TestCheckpointGateQualifyingReviewPasses is the basic positive test: a
// review dispatched after the IV transition, APPROVE, isolated, covering
// every gate-time trigger, at the current policy digest, contract digest,
// and required risk, with the transition's own artifact, lets the IV exit
// pass.
func TestCheckpointGateQualifyingReviewPasses(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	afterIV := core[len(core)-1].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""),
	})
	if err := checkpointGate(t, cb, history); err != nil {
		t.Fatalf("checkpoint gate with a qualifying review: %v", err)
	}
}

// TestCheckpointReviewStaleFieldMismatchReachable confirms
// HOST_CHECKPOINT_REVIEW_STALE's two conditions are each reachable directly
// (bound_contract_digest mismatch, dispatch risk below the gate-time
// required risk), as a single-field, non-rewind counterpart to
// TestCheckpointStaleAfterContractReaccepted/TestCheckpointStaleAfterPolicyChange,
// whose own route-approval-invalidation finding means neither reaches this
// branch through a real contract or policy change.
func TestCheckpointReviewStaleFieldMismatchReachable(t *testing.T) {
	for _, tc := range []struct {
		name           string
		contractDigest string
		risk           model.RouteRiskLevel
	}{
		{"contract-digest-mismatch", "stale-contract-digest", ""},
		{"risk-below-required", "", model.RouteRiskLow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := seedCheckpointBoundRun(t, model.RouteRiskLow, securityPathPolicy(), false, true)
			decision := classifyForFixture(cb, model.RouteRiskLow, securityPathPolicy(), false)
			if decision.RequiredRisk != model.RouteRiskHigh {
				t.Fatalf("fixture RequiredRisk = %q, want high (T5 raises the floor)", decision.RequiredRisk)
			}
			core, err := store.New(cb.store).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			afterIV := core[len(core)-1].Metadata.HeadAnchor
			contractDigest := cb.contractDigest
			if tc.contractDigest != "" {
				contractDigest = tc.contractDigest
			}
			risk := decision.RequiredRisk
			if tc.risk != "" {
				risk = tc.risk
			}
			history := checkpointReviewRecords("U1", checkpointReviewSpec{
				dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: risk, reason: hoststate.IssueCheckpoint,
				triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: contractDigest,
				artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""),
			})
			err = checkpointGate(t, cb, history)
			if err == nil || err.Error() != reasonCheckpointReviewStale {
				t.Fatalf("checkpoint gate with %s: got %v, want %s", tc.name, err, reasonCheckpointReviewStale)
			}
		})
	}
}

// checkpointReacceptContract appends a second ACCEPTANCE_CRITERIA ->
// PLANNING approval directly to cb's core chain, binding newContractDigest,
// exactly as engine.BoundContractDigest reads it; it does not replay the
// intervening rewind stages transition.Apply would otherwise require, since
// this seeds the host-gate test's store directly (as seedCheckpointBoundRun
// itself does), not through transition.Apply.
func checkpointReacceptContract(t *testing.T, cb *checkpointRun, newContractDigest string) {
	t.Helper()
	s := store.New(cb.store)
	core, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	last := core[len(core)-1]
	r := model.Record{Kind: model.KindStageTransition, Stage: model.StageAcceptanceCriteria, Control: model.ControlApproved,
		Approval: &model.ApprovalBinding{RunID: "run", TransitionDigest: "contract-2", CurrentStage: model.StageAcceptanceCriteria,
			ProposedTargetStage: model.StagePlanning, AcceptanceContractDigest: newContractDigest,
			RepositoryIdentityDigest: "repo", BaseRevisionDigest: cb.base, StageTimeDigest: "head:2", AcceptedPlanDigest: leasePlanDigest(core, "U1"),
			ChallengeNonce: "reaccept", Approver: "human", Timestamp: "2026-09-16T11:45:00Z"},
		Version:  1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: last.Metadata.Sequence + 1, Timestamp: "2026-09-16T11:45:00Z", Actor: "operator-confirmed:lennon", PredecessorHash: last.Metadata.HeadAnchor}}
	if err := s.Append(&r); err != nil {
		t.Fatalf("reaccept contract: %v", err)
	}
}

// TestCheckpointStaleAfterContractReaccepted is sec5.2's roadmap-ask test: a
// qualifying checkpoint review exists, but the run then re-accepts a new
// contract (a different bound_contract_digest at gate time). A contract
// re-acceptance invalidates U1's route approval (cmd/hma/transition.go's
// approvedCoreRouteDigestIn, read through corePacketState before this
// gate's own staleness checks run), so that fires first: the gate refuses
// with HOST_ROUTE_APPROVAL_MISSING, not HOST_CHECKPOINT_REVIEW_STALE.
func TestCheckpointStaleAfterContractReaccepted(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	afterIV := core[len(core)-1].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""),
	})
	checkpointReacceptContract(t, cb, "contract-2")

	err = checkpointGate(t, cb, history)
	if err == nil || err.Error() != hoststate.ReasonRouteApprovalMissing {
		t.Fatalf("checkpoint gate after a contract re-acceptance: got %v, want %s (route invalidation fires before the checkpoint gate's own staleness check)", err, hoststate.ReasonRouteApprovalMissing)
	}
}

// TestCheckpointStaleAfterPolicyChange mirrors
// TestCheckpointStaleAfterContractReaccepted for a checkpoint-policy change:
// architecture sec9 carries checkpoint_policy_digest on the contract itself
// (Decision P), so a policy change is also a contract re-acceptance and hits
// the same route-approval invalidation first.
func TestCheckpointStaleAfterPolicyChange(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, securityPathPolicy(), false, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, securityPathPolicy(), false)
	if len(decision.Triggers) == 0 {
		t.Fatal("fixture triggers empty, want T5 from the security-path policy")
	}
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	afterIV := core[len(core)-1].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""),
	})
	checkpointReacceptContract(t, cb, "contract-2-new-policy")

	err = checkpointGate(t, cb, history)
	if err == nil || err.Error() != hoststate.ReasonRouteApprovalMissing {
		t.Fatalf("checkpoint gate after a policy-carrying contract re-acceptance: got %v, want %s (route invalidation fires before the checkpoint gate's own staleness check)", err, hoststate.ReasonRouteApprovalMissing)
	}
}

// TestCheckpointGateIgnoresNonContractRun confirms the checkpoint gate does
// not apply at all (Decision D10) to a run with no bound acceptance
// contract: the per-change review fixture from newReviewFixture, advanced to
// the checkpoint edge, passes with no checkpoint review and no contract or
// checkpoint-policy path supplied. It covers both a populated and an empty
// UnitID: a non-contract run gets no UnitID requirement either (F1, Phil's
// review of Phase 4b: unit_id is documented in types.go as optional, "empty
// means no unit supersession", and at HEAD before Gate E this edge matched
// nothing and returned nil for every caller regardless of UnitID).
func TestCheckpointGateIgnoresNonContractRun(t *testing.T) {
	for _, unit := range []string{"U1", ""} {
		t.Run("unit="+unit, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			err := requireHostTransitionGateIn(f.store, transition.Input{RunID: "run", RepositoryRoot: f.root, BaseRevision: f.base,
				ExpectedApproval: model.ApprovalBinding{UnitID: unit, CurrentStage: model.StageIndependentValidation, ProposedTargetStage: model.StageReleaseAndClosure,
					ProducedHeadDigest: f.head, DiffDigest: f.diff}}, nil, "", "")
			if err != nil {
				t.Fatalf("checkpoint gate on a non-contract run (unit=%q): %v", unit, err)
			}
		})
	}
}

// TestCheckpointContractBoundEmptyUnitIDRefused is the contract-bound
// counterpart to TestCheckpointGateIgnoresNonContractRun's empty-UnitID
// case: once a run is contract-bound, the checkpoint edge does require a
// UnitID (a checkpoint-grade review lookup is inherently per-unit), and an
// empty one is refused with HOST_UNIT_ID_REQUIRED, fail-closed, exactly as
// Phil's review confirmed is the correct behavior for that case.
func TestCheckpointContractBoundEmptyUnitIDRefused(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, ordinaryOnlyPolicy(), false, true)
	err := requireHostTransitionGateIn(cb.store, transition.Input{RunID: "run", RepositoryRoot: cb.root, BaseRevision: cb.base, PolicyPath: cb.policy,
		ExpectedApproval: model.ApprovalBinding{UnitID: "", CurrentStage: model.StageIndependentValidation, ProposedTargetStage: model.StageReleaseAndClosure,
			ProducedHeadDigest: cb.head, DiffDigest: cb.diff}}, nil, cb.contractPath, cb.checkpointPolicyPath)
	if err == nil || err.Error() != "HOST_UNIT_ID_REQUIRED" {
		t.Fatalf("checkpoint gate on a contract-bound run with empty UnitID: got %v, want HOST_UNIT_ID_REQUIRED", err)
	}
}

// TestCheckpointArtifactMismatchRefused is F3 (Phil's review of Phase 4b): no
// E2-specific test previously reached hoststate.ReasonReviewArtifactMismatch
// through the checkpoint edge, even though the branch is a direct copy of
// the pattern IMPLEMENTATION_REVIEW -> VERIFICATION already uses and tests.
// This constructs an otherwise-qualifying checkpoint review whose recorded
// ArtifactDigest does not match the transition's declared artifact.
func TestCheckpointArtifactMismatchRefused(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	decision := classifyForFixture(cb, model.RouteRiskLow, nil, true)
	core, err := store.New(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	afterIV := core[len(core)-1].Metadata.HeadAnchor
	history := checkpointReviewRecords("U1", checkpointReviewSpec{
		dispatchDigest: "d1", coreHeadAnchor: afterIV, risk: decision.RequiredRisk, reason: hoststate.IssueCheckpoint,
		triggers: triggerStrings(decision.Triggers), policyDigest: decision.PolicyDigest, contractDigest: cb.contractDigest,
		artifact: "wrong-artifact-digest",
	})
	err = checkpointGate(t, cb, history)
	if err == nil || err.Error() != hoststate.ReasonReviewArtifactMismatch {
		t.Fatalf("checkpoint gate with a mismatched review artifact digest: got %v, want %s", err, hoststate.ReasonReviewArtifactMismatch)
	}
}

// TestCheckpointRegressionExistingEdgesUnaffected is a direct regression
// check (independent of the full suite) that the two existing edges'
// behavior did not change: requireHostTransitionGate (whose checkpointPolicy
// and contract paths are always "") still gates
// IMPLEMENTATION_AUTHORIZATION -> IMPLEMENTATION_REVIEW exactly as before.
func TestCheckpointRegressionExistingEdgesUnaffected(t *testing.T) {
	dir := t.TempDir()
	seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: model.StageImplementationReview,
	}})
	if err == nil || err.Error() != hoststate.ReasonDecisionMissing {
		t.Fatalf("implementation gate without a decision: got %v, want %s", err, hoststate.ReasonDecisionMissing)
	}
}
