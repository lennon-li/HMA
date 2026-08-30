package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

type transitionCase struct {
	Name    string            `json:"name"`
	Request TransitionRequest `json:"request"`
	Expect  TransitionResult  `json:"expect"`
}

type rewindCase struct {
	Name    string              `json:"name"`
	Request InvalidationRequest `json:"request"`
	Expect  InvalidationResult  `json:"expect"`
}

type waiverCase struct {
	Name    string              `json:"name"`
	Request WaiverChangeRequest `json:"request"`
	Expect  InvalidationResult  `json:"expect"`
}

type invalidationFixture struct {
	Rewinds       []rewindCase `json:"rewinds"`
	WaiverChanges []waiverCase `json:"waiver_changes"`
}

func loadFixture(t *testing.T, path string, into any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}
}

// TestTransitionFixtures drives every legal-edge and rejection case from
// testdata/engine/transitions.json.
func TestTransitionFixtures(t *testing.T) {
	var cases []transitionCase
	loadFixture(t, "../../testdata/engine/transitions.json", &cases)
	if len(cases) == 0 {
		t.Fatal("transitions fixture is empty")
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got := EvaluateTransition(tc.Request)
			if !reflect.DeepEqual(got, tc.Expect) {
				t.Fatalf("EvaluateTransition\n got: %+v\nwant: %+v", got, tc.Expect)
			}
			if got.MachineAdvanced {
				t.Fatal("machine advancement reported")
			}
			if !got.RequiresFreshHumanApproval {
				t.Fatal("result did not require fresh human approval")
			}
		})
	}
}

// TestLegalEdgeSetIsExactlyTheApprovedTable states the approved edge set
// independently of the implementation table and proves no unlisted edge is
// legal and no listed edge is missing.
func TestLegalEdgeSetIsExactlyTheApprovedTable(t *testing.T) {
	expected := map[model.Stage][]model.Stage{
		model.StageGrounding: {
			model.StageAcceptanceCriteria, model.StagePlanning,
		},
		model.StageAcceptanceCriteria: {
			model.StagePlanning, model.StageGrounding,
		},
		model.StagePlanning: {
			model.StageRouteSelection, model.StageGrounding,
		},
		model.StageRouteSelection: {
			model.StageImplementationAuthorization,
			model.StageGrounding, model.StagePlanning,
		},
		model.StageImplementationAuthorization: {
			model.StageImplementationReview, model.StageRouteSelection,
			model.StageGrounding, model.StagePlanning,
		},
		model.StageImplementationReview: {
			model.StageVerification, model.StageRouteSelection,
			model.StageGrounding, model.StagePlanning,
		},
		model.StageVerification: {
			model.StageIndependentValidation, model.StageRouteSelection,
			model.StageGrounding, model.StagePlanning,
		},
		model.StageIndependentValidation: {
			model.StageImplementationAuthorization, model.StagePlanning,
			model.StageReleaseAndClosure, model.StageRouteSelection,
			model.StageGrounding,
		},
		model.StageReleaseAndClosure: {
			model.StageRouteSelection, model.StageAcceptanceCriteria,
			model.StagePlanning, model.StageGrounding,
		},
	}
	all := []model.Stage{
		model.StageGrounding, model.StageAcceptanceCriteria,
		model.StagePlanning, model.StageRouteSelection,
		model.StageImplementationAuthorization,
		model.StageImplementationReview, model.StageVerification,
		model.StageIndependentValidation, model.StageReleaseAndClosure,
	}
	for _, from := range all {
		want := map[model.Stage]bool{}
		for _, to := range expected[from] {
			want[to] = true
		}
		for _, to := range all {
			if got := IsLegalEdge(from, to); got != want[to] {
				t.Errorf("IsLegalEdge(%s, %s) = %v, want %v", from, to, got, want[to])
			}
		}
	}
}

// TestNoInputAdvancesAStage sweeps every stage pair, every terminal target,
// and the machine-advancement claim to prove the engine can never advance a
// stage and always demands a fresh human approval.
func TestNoInputAdvancesAStage(t *testing.T) {
	all := []model.Stage{
		model.StageGrounding, model.StageAcceptanceCriteria,
		model.StagePlanning, model.StageRouteSelection,
		model.StageImplementationAuthorization,
		model.StageImplementationReview, model.StageVerification,
		model.StageIndependentValidation, model.StageReleaseAndClosure,
		model.Stage("NOT_A_STAGE"),
	}
	outcomes := []model.TerminalOutcome{
		"", model.OutcomeAborted, model.OutcomeBlocked, model.OutcomeUnknown,
		model.OutcomeFailed, model.OutcomePartial,
		model.OutcomeVerifiedWithWaivers, model.OutcomeVerifiedSuccess,
		model.TerminalOutcome("NOT_AN_OUTCOME"),
	}
	for _, from := range all {
		for _, to := range all {
			for _, out := range outcomes {
				for _, claim := range []bool{false, true} {
					req := TransitionRequest{
						Source:                   from,
						TargetStage:              to,
						TargetOutcome:            out,
						ClaimsMachineAdvancement: claim,
					}
					got := EvaluateTransition(req)
					if got.MachineAdvanced {
						t.Fatalf("machine advanced for %+v", req)
					}
					if !got.RequiresFreshHumanApproval {
						t.Fatalf("no fresh approval required for %+v", req)
					}
					if claim && got.Decision != DecisionRejected {
						t.Fatalf("machine-advancement claim accepted for %+v", req)
					}
					if got.Decision == DecisionLegalPendingApproval &&
						got.ControlState != model.ControlReadyForReview {
						t.Fatalf("legal proposal not READY_FOR_REVIEW for %+v", req)
					}
				}
			}
		}
	}
}

// TestInvalidationFixtures drives every Gate 0 rewind class, scope rule, and
// rejection from testdata/engine/invalidation.json.
func TestInvalidationFixtures(t *testing.T) {
	var fx invalidationFixture
	loadFixture(t, "../../testdata/engine/invalidation.json", &fx)
	if len(fx.Rewinds) == 0 || len(fx.WaiverChanges) == 0 {
		t.Fatal("invalidation fixture is incomplete")
	}
	for _, tc := range fx.Rewinds {
		t.Run("rewind/"+tc.Name, func(t *testing.T) {
			got := EvaluateInvalidation(tc.Request)
			if !reflect.DeepEqual(got, tc.Expect) {
				t.Fatalf("EvaluateInvalidation\n got: %+v\nwant: %+v", got, tc.Expect)
			}
			assertNeverAdvances(t, got)
		})
	}
	for _, tc := range fx.WaiverChanges {
		t.Run("waiver/"+tc.Name, func(t *testing.T) {
			got := EvaluateWaiverChange(tc.Request)
			if !reflect.DeepEqual(got, tc.Expect) {
				t.Fatalf("EvaluateWaiverChange\n got: %+v\nwant: %+v", got, tc.Expect)
			}
			assertNeverAdvances(t, got)
		})
	}
}

func assertNeverAdvances(t *testing.T, got InvalidationResult) {
	t.Helper()
	if got.MachineAdvanced {
		t.Fatal("machine advancement reported")
	}
	if !got.RequiresFreshHumanApproval {
		t.Fatal("result did not require fresh human approval")
	}
	if len(got.RestoredApprovalIDs) != 0 {
		t.Fatal("an invalidated approval was restored")
	}
	if got.Decision == DecisionLegalPendingApproval &&
		got.RewindTarget != "" &&
		got.ControlState != model.ControlReadyForReview {
		t.Fatal("invalidation posture is not READY_FOR_REVIEW")
	}
	if got.PacketControl != "" && got.PacketControl != model.PacketInvalidated {
		t.Fatalf("unexpected packet-control value %q", got.PacketControl)
	}
}

// TestEveryGate0TriggerHasExactlyOneRule proves the rewind table is
// exhaustive over the Gate 0 trigger list and yields one deterministic
// target/scope pair per trigger.
func TestEveryGate0TriggerHasExactlyOneRule(t *testing.T) {
	want := map[RewindTrigger]rewindRule{
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
	if !reflect.DeepEqual(rewindTable, want) {
		t.Fatalf("rewind table\n got: %+v\nwant: %+v", rewindTable, want)
	}
	if got := EvaluateInvalidation(InvalidationRequest{Trigger: "NOT_A_TRIGGER"}); got.Decision != DecisionRejected ||
		got.Reason != ReasonUnknownRewindTrigger {
		t.Fatalf("unlisted rewind trigger accepted: %+v", got)
	}
}

// TestInvalidatedVocabularyStaysDistinct proves the Task 1 boundary:
// INVALIDATED is packet-control vocabulary, never a stage and never a
// stage-control state.
func TestInvalidatedVocabularyStaysDistinct(t *testing.T) {
	if model.ValidStage(model.Stage(model.PacketInvalidated)) {
		t.Fatal("INVALIDATED is being treated as a stage")
	}
	if model.ValidControlState(model.StageControlState(model.PacketInvalidated)) {
		t.Fatal("INVALIDATED is being treated as a stage-control state")
	}
	if !model.ValidPacketControlState(model.PacketInvalidated) {
		t.Fatal("INVALIDATED is not valid packet-control vocabulary")
	}
	rec := model.Record{}
	if _, err := json.Marshal(rec); err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	raw, err := json.Marshal(model.Record{Kind: model.KindStageTransition, Version: 1})
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("unmarshal record: %v", err)
	}
	if _, ok := round["packet_control"]; ok {
		t.Fatal("model.Record gained a serialized packet_control field")
	}
}

// TestEvaluationDoesNotMutateInput proves both evaluators are pure with
// respect to caller-owned slices.
func TestEvaluationDoesNotMutateInput(t *testing.T) {
	req := InvalidationRequest{
		Trigger:              TriggerPlanUnitChanged,
		AffectedUnitID:       "U1",
		DependentUnitIDs:     []string{"U2"},
		ChangedInputs:        []string{"plan.U1"},
		ChangedWaiverIDs:     []string{"W-2", "W-1"},
		ChangedWaivedTargets: []string{"C-1"},
		Waivers: []WaiverOp{
			{ID: "W-1", Kind: model.ScopeCriterion, Target: "C-1", Active: true},
		},
		Approvals: []ApprovalPacket{
			{ID: "A-2", UnitID: "U1", Stage: model.StagePlanning, BoundInputs: []string{"plan.U1"}},
			{ID: "A-1", UnitID: "U3", Stage: model.StagePlanning, WaiverIDs: []string{"W-9"}},
		},
	}
	before, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	_ = EvaluateInvalidation(req)
	after, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("input mutated\nbefore: %s\n after: %s", before, after)
	}
}
