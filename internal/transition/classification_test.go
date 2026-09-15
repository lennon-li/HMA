package transition

import (
	"context"
	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"testing"
	"time"
)

func TestApplyClassificationAndOutcome(t *testing.T) {
	repo, base := newRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	for _, outcome := range []model.TerminalOutcome{model.OutcomeBlocked, model.OutcomeUnknown, model.OutcomeFailed} {
		t.Run(string(outcome), func(t *testing.T) {
			dir := t.TempDir()
			makeInput := func() Input {
				in := approvedInput(t, repo, base, dir, model.StageGrounding, "", "classification", now)
				in.Classification = &model.Classification{Outcome: outcome, FindingID: "F1"}
				in.ExpectedApproval.UnitID = "U1"
				in.ExpectedApproval.Classification = &model.Classification{Outcome: outcome, FindingID: "F1"}
				in.ExpectedApproval.ProposedTargetOutcome = outcome
				in.Approval = in.ExpectedApproval
				return in
			}
			for _, spoil := range []func(*Input){
				func(in *Input) { in.Approval = model.ApprovalBinding{} },
				func(in *Input) { in.Classification.FindingID = "other" },
				func(in *Input) { in.Approval.UnitID = "other" },
				func(in *Input) { in.ApprovalNow = now.Add(time.Hour) },
				func(in *Input) { in.Approval.StageTimeDigest = "stale" },
				func(in *Input) { in.Classification.Outcome = model.OutcomePartial },
			} {
				in := makeInput()
				spoil(&in)
				if _, err := Apply(context.Background(), in, dir); err == nil {
					t.Fatal("malformed classification accepted")
				}
				records, err := store.New(dir).Load("run")
				if err != nil || len(records) != 0 {
					t.Fatalf("refusal mutated chain: %v", err)
				}
			}
			in := makeInput()
			if _, err := Apply(context.Background(), in, dir); err != nil {
				t.Fatal(err)
			}
			records, err := store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			state := engine.ProjectStageState(records)
			if len(records) != 1 || records[0].Kind != model.KindClassification || state.Terminal || state.FailureOutcome != outcome {
				t.Fatalf("classification: %+v", state)
			}
			if _, err := Apply(context.Background(), in, dir); err == nil {
				t.Fatal("replay accepted")
			}
			// A routine approved stage change must preserve this unit's classification.
			advance := approvedInput(t, repo, base, dir, model.StageGrounding, model.StageAcceptanceCriteria, "advance", now)
			advance.ExpectedApproval.UnitID = "U1"
			advance.Approval = advance.ExpectedApproval
			if _, err := Apply(context.Background(), advance, dir); err != nil {
				t.Fatal(err)
			}
			records, err = store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			state = engine.ProjectStageState(records)
			if state.FailureOutcome != outcome || state.CurrentStage != model.StageAcceptanceCriteria {
				t.Fatalf("stage transition did not preserve classification: %+v", state)
			}
			// Reclassify at the new stage, then confirm the terminal outcome.
			in = approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, "", "reclassify", now)
			in.Classification = &model.Classification{Outcome: outcome, FindingID: "F1"}
			in.ExpectedApproval.UnitID = "U1"
			in.ExpectedApproval.Classification = in.Classification
			in.ExpectedApproval.ProposedTargetOutcome = outcome
			in.Approval = in.ExpectedApproval
			if _, err := Apply(context.Background(), in, dir); err != nil {
				t.Fatal(err)
			}
			next := approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, "", "outcome", now)
			next.ExpectedApproval.ProposedTargetOutcome = outcome
			next.Approval = next.ExpectedApproval
			if _, err := Apply(context.Background(), next, dir); err != nil {
				t.Fatal(err)
			}
			records, err = store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			state = engine.ProjectStageState(records)
			if !state.Terminal || state.Outcome != outcome {
				t.Fatalf("outcome: %+v", state)
			}
		})
	}
}
