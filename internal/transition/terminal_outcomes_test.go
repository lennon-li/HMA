package transition

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func TestApplyPhaseA1Outcomes(t *testing.T) {
	for _, outcome := range []model.TerminalOutcome{model.OutcomeAborted, model.OutcomePartial} {
		t.Run(string(outcome), func(t *testing.T) {
			repo, base := newRepo(t)
			dir := t.TempDir()
			now := time.Now().UTC().Truncate(time.Second)
			if outcome == model.OutcomePartial {
				seed := model.Record{Kind: model.KindCriterion, Version: 1,
					Metadata: model.RecordMetadata{RunID: "run", Sequence: 1, Actor: "human", Timestamp: now.Format(time.RFC3339)},
					Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPassed}, {ID: "C2", Disposition: model.CriterionPending}},
				}
				if err := store.New(dir).Append(&seed); err != nil {
					t.Fatal(err)
				}
			}
			in := approvedInput(t, repo, base, dir, model.StageGrounding, "", "outcome-nonce", now)
			in.ExpectedApproval.ProposedTargetOutcome = outcome
			in.Approval = in.ExpectedApproval
			res, err := Apply(context.Background(), in, dir)
			if err != nil {
				t.Fatal(err)
			}
			if res.ToOutcome != outcome || res.ToStage != "" || res.FromStage != model.StageGrounding || res.MachineAdvanced {
				t.Fatalf("result = %+v", res)
			}
			records, err := store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			tail := records[len(records)-1]
			if tail.Kind != model.KindOutcome || tail.Outcome != outcome || tail.Approval.ProposedTargetOutcome != outcome || tail.Control != model.ControlApproved {
				t.Fatalf("record = %+v", tail)
			}
			if outcome == model.OutcomePartial && len(tail.Criteria) != 2 {
				t.Fatal("criterion snapshot not persisted")
			}
			state := engine.ProjectStageState(records)
			if !state.Terminal || state.Outcome != outcome || state.CurrentStage != model.StageGrounding || !state.UsedNonces["outcome-nonce"] {
				t.Fatalf("state = %+v", state)
			}
			for _, next := range []Input{in, approvedInput(t, repo, base, dir, model.StageGrounding, model.StageAcceptanceCriteria, "next", now)} {
				if _, err := Apply(context.Background(), next, dir); err == nil || !strings.Contains(err.Error(), string(engine.ReasonRunTerminal)) {
					t.Fatalf("terminal replay: %v", err)
				}
			}
			after, err := store.New(dir).Load("run")
			if err != nil || len(after) != len(records) {
				t.Fatalf("rejection mutated chain: %v", err)
			}
			// Corrupt the serialized target without recomputing the trusted anchor.
			path := filepath.Join(dir, "run.jsonl")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			tampered := strings.Replace(string(raw), `"proposed_target_outcome":"`+string(outcome)+`"`, `"proposed_target_outcome":"FAILED"`, 1)
			if tampered == string(raw) {
				t.Fatal("tamper did not alter fixture")
			}
			if err := os.WriteFile(path, []byte(tampered), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.New(dir).Load("run"); err == nil {
				t.Fatal("tampered chain loaded")
			}
			if _, err := Apply(context.Background(), in, dir); err == nil || !strings.Contains(err.Error(), "head mismatch") {
				t.Fatalf("tampered chain accepted: %v", err)
			}
		})
	}
}

func TestApplyRefusesUnsupportedOrUnsubstantiatedOutcomes(t *testing.T) {
	repo, base := newRepo(t)
	for _, outcome := range []model.TerminalOutcome{model.OutcomePartial, model.OutcomeBlocked, model.OutcomeUnknown, model.OutcomeFailed, model.OutcomeVerifiedSuccess, model.OutcomeVerifiedWithWaivers} {
		t.Run(string(outcome), func(t *testing.T) {
			dir := t.TempDir()
			in := approvedInput(t, repo, base, dir, model.StageGrounding, "", "n", time.Now().UTC().Truncate(time.Second))
			in.ExpectedApproval.ProposedTargetOutcome = outcome
			in.Approval = in.ExpectedApproval
			if _, err := Apply(context.Background(), in, dir); err == nil {
				t.Fatal("outcome accepted")
			}
			records, err := store.New(dir).Load("run")
			if err != nil || len(records) != 0 {
				t.Fatalf("refusal wrote a record: %v", err)
			}
		})
	}
}
