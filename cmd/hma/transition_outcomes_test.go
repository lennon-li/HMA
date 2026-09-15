package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

func TestTransitionCLIPhaseA1(t *testing.T) {
	for _, outcome := range []model.TerminalOutcome{model.OutcomeAborted, model.OutcomePartial} {
		t.Run(string(outcome), func(t *testing.T) {
			repo, dir := gitRepo(t), t.TempDir()
			cmd := exec.Command("git", "-C", repo, "-c", "user.name=HMA Test", "-c", "user.email=hma@example.invalid", "commit", "--allow-empty", "-qm", "fixture")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git commit: %v: %s", err, out)
			}
			snap, err := repostate.Snapshot(context.Background(), repo, "HEAD", false)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC().Truncate(time.Second)
			if outcome == model.OutcomePartial {
				seed := model.Record{Kind: model.KindCriterion, Version: 1,
					Metadata: model.RecordMetadata{RunID: "run", Sequence: 1, Actor: "human", Timestamp: now.Format(time.RFC3339)},
					Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionWaived}, {ID: "C2", Disposition: model.CriterionFailed}},
				}
				if err := store.New(dir).Append(&seed); err != nil {
					t.Fatal(err)
				}
			}
			records, err := store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			state := engine.ProjectStageState(records)
			a := model.ApprovalBinding{RunID: "run", TransitionDigest: "target-" + string(outcome), CurrentStage: model.StageGrounding,
				ProposedTargetOutcome: outcome, RepositoryIdentityDigest: "repo", BaseRevisionDigest: snap.Head,
				StageTimeDigest: fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1), AcceptedPlanDigest: "plan",
				ChallengeNonce: "outcome", Approver: "human", Timestamp: now.Format(time.RFC3339)}
			in := transition.Input{RunID: "run", RepositoryRoot: repo, RepositoryIdentity: "repo", BaseRevision: snap.Head,
				ExpectedApproval: a, Approval: a, ApprovalNow: now, ApprovalMaxAgeSeconds: 300}
			input := filepath.Join(t.TempDir(), "input.json")
			writeJSON(t, input, in)
			if err := run([]string{"transition", "--input", input, "--store", dir}); err != nil {
				t.Fatal(err)
			}
			records, err = store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			if tail := records[len(records)-1]; tail.Kind != model.KindOutcome || tail.Outcome != outcome {
				t.Fatalf("tail = %+v", tail)
			}
			if err := run([]string{"transition", "--input", input, "--store", dir}); err == nil || !strings.Contains(err.Error(), string(engine.ReasonRunTerminal)) {
				t.Fatalf("replay = %v", err)
			}
			// An altered host anchor must be refused by the command before writing.
			if err := os.WriteFile(filepath.Join(dir, "run.head"), []byte("tampered\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := run([]string{"transition", "--input", input, "--store", dir}); err == nil || !strings.Contains(err.Error(), "host head mismatch") {
				t.Fatalf("tamper = %v", err)
			}
		})
	}
}
