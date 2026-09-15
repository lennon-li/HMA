package engine

import (
	"github.com/lennon-li/HMA/internal/model"
	"testing"
)

func outcomeRequest(outcome model.TerminalOutcome, criteria ...model.Criterion) StageTransitionRequest {
	state := ProjectStageState(nil)
	state.Criteria = criteria
	req := stageTransitionFixture(state, model.StageGrounding, "")
	req.Expected.ProposedTargetOutcome = outcome
	req.Presented = req.Expected
	return req
}

func TestPhaseA1Outcomes(t *testing.T) {
	for _, tc := range []struct {
		name         string
		outcome      model.TerminalOutcome
		dispositions []model.CriterionDisposition
		legal        bool
	}{
		{"abort empty", model.OutcomeAborted, nil, true},
		{"partial passed pending", model.OutcomePartial, []model.CriterionDisposition{model.CriterionPassed, model.CriterionPending}, true},
		{"partial waived failed", model.OutcomePartial, []model.CriterionDisposition{model.CriterionWaived, model.CriterionFailed}, true},
		{"partial empty", model.OutcomePartial, nil, false},
		{"partial all passed", model.OutcomePartial, []model.CriterionDisposition{model.CriterionPassed}, false},
		{"partial all resolved", model.OutcomePartial, []model.CriterionDisposition{model.CriterionPassed, model.CriterionWaived}, false},
		{"partial none resolved", model.OutcomePartial, []model.CriterionDisposition{model.CriterionPending, model.CriterionFailed}, false},
		{"blocked", model.OutcomeBlocked, nil, false},
		{"unknown", model.OutcomeUnknown, nil, false},
		{"failed", model.OutcomeFailed, nil, false},
		{"verified", model.OutcomeVerifiedSuccess, nil, false},
		{"verified waivers", model.OutcomeVerifiedWithWaivers, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var criteria []model.Criterion
			for _, d := range tc.dispositions {
				criteria = append(criteria, model.Criterion{Disposition: d})
			}
			req := outcomeRequest(tc.outcome, criteria...)
			// Exercise closure refusal even at the architecturally legal source.
			if tc.outcome == model.OutcomeVerifiedSuccess || tc.outcome == model.OutcomeVerifiedWithWaivers {
				req.State.CurrentStage = model.StageReleaseAndClosure
				req.Expected.CurrentStage = req.State.CurrentStage
				req.Presented = req.Expected
			}
			res := EvaluateStageTransition(req)
			if (res.Decision == DecisionLegalPendingApproval) != tc.legal || res.MachineAdvanced || res.ToOutcome != tc.outcome || res.ToStage != "" {
				t.Fatalf("result = %+v", res)
			}
		})
	}
}

func TestOutcomeApprovalRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		spoil  func(*StageTransitionRequest)
		reason Reason
	}{
		{"stage mismatch", func(r *StageTransitionRequest) { r.Presented.CurrentStage = model.StagePlanning }, ReasonStageMismatch},
		{"stale position", func(r *StageTransitionRequest) { r.State.Sequence++ }, ReasonStaleApproval},
		{"replayed nonce", func(r *StageTransitionRequest) { r.State.UsedNonces[r.Presented.ChallengeNonce] = true }, ReasonReusedNonce},
		{"target changed", func(r *StageTransitionRequest) { r.Presented.ProposedTargetOutcome = model.OutcomePartial }, ReasonStaleApproval},
		{"both targets", func(r *StageTransitionRequest) { r.Presented.ProposedTargetStage = model.StagePlanning }, ReasonAmbiguousTarget},
		{"neither target", func(r *StageTransitionRequest) { r.Presented.ProposedTargetOutcome = "" }, ReasonMissingTarget},
		{"unknown target", func(r *StageTransitionRequest) { r.Presented.ProposedTargetOutcome = "BOGUS" }, ReasonUnknownTarget},
		{"terminal", func(r *StageTransitionRequest) { r.State.Terminal = true }, ReasonRunTerminal},
		{"expired", func(r *StageTransitionRequest) { r.Now = r.Now.Add(r.MaxAge * 2) }, ReasonStaleApproval},
		{"required evidence", func(r *StageTransitionRequest) {
			r.Expected.RequiredEvidenceDigests = []string{"missing"}
			r.Presented = r.Expected
		}, ReasonMissingRequiredEvidence},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := outcomeRequest(model.OutcomeAborted)
			tc.spoil(&req)
			res := EvaluateStageTransition(req)
			if res.Decision != DecisionRejected || res.Reason != tc.reason || res.MachineAdvanced {
				t.Fatalf("result = %+v", res)
			}
		})
	}
}

func TestOutcomeCriteriaProjection(t *testing.T) {
	old := []model.Criterion{{ID: "c1", Disposition: model.CriterionPassed}, {ID: "c2", Disposition: model.CriterionPending}}
	latest := []model.Criterion{{ID: "c1", Disposition: model.CriterionPassed}, {ID: "c2", Disposition: model.CriterionPassed}}
	state := ProjectStageState([]model.Record{{Criteria: old}, {Criteria: latest}, {Kind: model.KindEvidence}})
	req := outcomeRequest(model.OutcomePartial)
	req.State.Criteria = state.Criteria
	if res := EvaluateStageTransition(req); res.Reason != ReasonPartialCriteria {
		t.Fatalf("stale criteria accepted: %+v", res)
	}
}
