package engine

import (
	"github.com/lennon-li/HMA/internal/model"
	"testing"
)

func classified(unit string, outcome model.TerminalOutcome) model.Record {
	req := outcomeRequest(outcome)
	c := &model.Classification{Outcome: outcome, FindingID: "finding"}
	req.Presented.UnitID = unit
	req.Presented.Classification = c
	return model.Record{Kind: model.KindClassification, UnitID: unit, Classification: c, Approval: &req.Presented, Stage: model.StageGrounding, Control: model.ControlApproved, Metadata: model.RecordMetadata{RunID: req.Presented.RunID}}
}

func TestClassificationPrecedenceAndSupersession(t *testing.T) {
	block := classified("blocked-unit", model.OutcomeBlocked)
	unknown := classified("unknown-unit", model.OutcomeUnknown)
	failed := classified("failed-unit", model.OutcomeFailed)
	transition := func(unit string) model.Record {
		r := classified(unit, model.OutcomeBlocked)
		r.Kind = model.KindStageTransition
		r.Classification = nil
		r.Approval.Classification = nil
		r.Approval.ProposedTargetOutcome = ""
		r.Approval.ProposedTargetStage = model.StageGrounding
		return r
	}
	change := func(unit string) model.Record {
		r := transition(unit)
		r.Kind = model.KindWaiverOperation
		r.Approval = nil
		r.WaiverOperation = &model.WaiverOperation{UnitID: unit, Operation: model.WaiverOperationGrant, Scope: model.ScopeSelector{Kind: model.ScopeFinding, Target: "finding"}, Justification: "human decision", Approver: "human", Timestamp: "2026-09-15T00:00:00Z", ChallengeNonce: "waiver"}
		return r
	}
	for _, tc := range []struct {
		name    string
		records []model.Record
		want    model.TerminalOutcome
	}{
		{"block first", []model.Record{block, unknown, failed}, model.OutcomeBlocked},
		{"block last", []model.Record{failed, unknown, block}, model.OutcomeBlocked},
		{"unknown", []model.Record{failed, unknown}, model.OutcomeUnknown},
		{"failed direct", []model.Record{failed}, model.OutcomeFailed},
		{"routine stage transition", []model.Record{block, transition(block.UnitID)}, model.OutcomeBlocked},
		{"rewind", []model.Record{block, unknown, change(block.UnitID)}, model.OutcomeUnknown},
		{"all cleared", []model.Record{block, change(block.UnitID)}, ""},
		{"other unit", []model.Record{block, change("other")}, model.OutcomeBlocked},
		{"earlier change", []model.Record{change(block.UnitID), block}, model.OutcomeBlocked},
		{"unapproved", []model.Record{block, {Kind: model.KindStageTransition, UnitID: block.UnitID}}, model.OutcomeBlocked},
		{"new classification", []model.Record{block, classified(block.UnitID, model.OutcomeFailed)}, model.OutcomeFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := ProjectStageState(tc.records)
			if state.FailureOutcome != tc.want || state.Terminal {
				t.Fatalf("state = %+v", state)
			}
		})
	}
	for _, kind := range []model.RecordKind{model.KindStageTransition, model.KindCriterion, model.KindWaiverOperation, model.KindFindingOverride} {
		r := transition(block.UnitID)
		r.Kind = kind
		if kind == model.KindWaiverOperation {
			r.Approval = nil
			r.WaiverOperation = &model.WaiverOperation{UnitID: block.UnitID, Operation: model.WaiverOperationGrant, Scope: model.ScopeSelector{Kind: model.ScopeFinding, Target: "finding"}, Justification: "human decision", Approver: "human", Timestamp: "2026-09-15T00:00:00Z", ChallengeNonce: "waiver"}
		}
		if kind == model.KindFindingOverride {
			r.Approval = nil
			r.FindingOverride = &model.FindingDispositionOverride{UnitID: block.UnitID, FindingID: "finding", Disposition: model.FindingWaivable, Justification: "human decision", Approver: "human", Timestamp: "2026-09-15T00:00:00Z", ChallengeNonce: "override"}
		}
		if kind == model.KindStageTransition || kind == model.KindCriterion {
			if got := ProjectStageState([]model.Record{block, r}); got.FailureOutcome != model.OutcomeBlocked {
				t.Fatalf("%s superseded with generic approval: %+v", kind, got)
			}
			continue
		}
		if got := ProjectStageState([]model.Record{block, r}); got.FailureOutcome != "" {
			t.Fatalf("%s did not supersede", kind)
		}

		for _, approvedUnit := range []string{"", "other"} {
			// A generic approval cannot compensate for a missing or mismatched nested unit.
			r.Approval = transition(block.UnitID).Approval
			if r.WaiverOperation != nil {
				r.WaiverOperation.UnitID = approvedUnit
			} else {
				r.FindingOverride.UnitID = approvedUnit
			}
			if got := ProjectStageState([]model.Record{block, r}); got.FailureOutcome != model.OutcomeBlocked {
				t.Fatalf("%s superseded with unapproved record unit: %+v", kind, got)
			}
		}
	}
}

func TestClassificationOutcomeRequests(t *testing.T) {
	for _, live := range []model.TerminalOutcome{model.OutcomeBlocked, model.OutcomeUnknown, model.OutcomeFailed} {
		for _, target := range []model.TerminalOutcome{model.OutcomeBlocked, model.OutcomeUnknown, model.OutcomeFailed, model.OutcomePartial, model.OutcomeAborted} {
			req := outcomeRequest(target, model.Criterion{Disposition: model.CriterionPassed}, model.Criterion{Disposition: model.CriterionPending})
			req.State.FailureOutcome = live
			got := EvaluateStageTransition(req)
			if (got.Decision == DecisionLegalPendingApproval) != (target == live || target == model.OutcomeAborted) {
				t.Fatalf("live %s target %s: %+v", live, target, got)
			}
		}
	}
}

func TestClassificationMalformedAndUnapproved(t *testing.T) {
	for _, spoil := range []func(*model.Record){
		func(r *model.Record) { r.Approval = nil },
		func(r *model.Record) { r.Classification = nil },
		func(r *model.Record) { r.UnitID = "" },
		func(r *model.Record) { r.Classification.FindingID = "" },
		func(r *model.Record) { r.Classification.Outcome = model.OutcomePartial },
		func(r *model.Record) { r.Control = model.ControlDraft },
		func(r *model.Record) { r.Approval.Approver = "" },
		func(r *model.Record) { r.Approval.Timestamp = "invalid" },
		func(r *model.Record) { r.Approval.UnitID = "other" },
		func(r *model.Record) {
			r.Approval.Classification = &model.Classification{Outcome: model.OutcomeBlocked, FindingID: "other"}
		},
	} {
		r := classified("unit", model.OutcomeBlocked)
		spoil(&r)
		if model.ValidateClassification(&r) == nil {
			t.Fatalf("accepted %+v", r)
		}
		if state := ProjectStageState([]model.Record{r}); state.FailureOutcome != "" {
			t.Fatal("invalid classification projected")
		}
	}
}
