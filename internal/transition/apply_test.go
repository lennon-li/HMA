package transition

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
)

const identity = "sha256:opaque-repository-identity"

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=HMA Test", "GIT_AUTHOR_EMAIL=hma@example.invalid", "GIT_COMMITTER_NAME=HMA Test", "GIT_COMMITTER_EMAIL=hma@example.invalid")
	b, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, b)
	}
	return string(b)
}

func newRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "x")
	git(t, repo, "commit", "-qm", "one")
	return repo, repostate.Trim(git(t, repo, "rev-parse", "HEAD"))
}

// approvedInput builds the input for one transition against the chain as it
// currently stands, binding whatever the stage's approval is required to bind.
func approvedInput(t *testing.T, repo, base, storeDir string, from, to model.Stage, nonce string, now time.Time) Input {
	t.Helper()
	return approvedInputAt(t, repo, base, storeDir, from, to, nonce, now, false)
}

// approvedDirtyInput builds the input for one transition against the dirty
// worktree the human approved exactly: the snapshot admits the dirty state,
// and both the host declaration and the approval itself bind the worktree
// digest of the content on disk.
func approvedDirtyInput(t *testing.T, repo, base, storeDir string, from, to model.Stage, nonce string, now time.Time) Input {
	t.Helper()
	return approvedInputAt(t, repo, base, storeDir, from, to, nonce, now, true)
}

func approvedInputAt(t *testing.T, repo, base, storeDir string, from, to model.Stage, nonce string, now time.Time, dirtyApproved bool) Input {
	t.Helper()
	records, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	state := engine.ProjectStageState(records)
	snap, err := repostate.Snapshot(context.Background(), repo, base, dirtyApproved)
	if err != nil {
		t.Fatal(err)
	}
	a := model.ApprovalBinding{
		RunID:                    "run",
		TransitionDigest:         "transition-" + nonce,
		CurrentStage:             from,
		ProposedTargetStage:      to,
		RepositoryIdentityDigest: identity,
		BaseRevisionDigest:       base,
		StageTimeDigest:          fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1),
		AcceptedPlanDigest:       "plan",
		ChallengeNonce:           nonce,
		Approver:                 "operator-confirmed:lennon",
		Timestamp:                now.Format(time.RFC3339),
	}
	if model.ApprovalBindsHeadDiff(from) {
		a.ProducedHeadDigest = snap.Head
		a.DiffDigest = snap.Diff
	}
	in := Input{
		RunID:                 "run",
		RepositoryRoot:        repo,
		RepositoryIdentity:    identity,
		BaseRevision:          base,
		ExpectedApproval:      a,
		Approval:              a,
		ApprovalNow:           now,
		ApprovalMaxAgeSeconds: 300,
	}
	if dirtyApproved {
		// The section 6 amendment: the approval record itself binds the
		// exact worktree content, alongside the host-level declaration.
		in.DirtyWorktreeApproved = true
		in.ExpectedWorktreeDigest = snap.Worktree
		a.WorktreeDigest = snap.Worktree
		in.ExpectedApproval = a
		in.Approval = a
	}
	return in
}

// walk is the architecture's ordinary path from GROUNDING to
// RELEASE_AND_CLOSURE, one approved edge at a time.
var walk = []struct{ from, to model.Stage }{
	{model.StageGrounding, model.StageAcceptanceCriteria},
	{model.StageAcceptanceCriteria, model.StagePlanning},
	{model.StagePlanning, model.StageRouteSelection},
	{model.StageRouteSelection, model.StageImplementationAuthorization},
	{model.StageImplementationAuthorization, model.StageImplementationReview},
	{model.StageImplementationReview, model.StageVerification},
	{model.StageVerification, model.StageIndependentValidation},
	{model.StageIndependentValidation, model.StageReleaseAndClosure},
}

// TestApplyWalksEveryStage is the point of the package: before it existed, a
// run could be evaluated but never moved, so the nine-stage model was
// unreachable. Each step is a separate human approval; none is inherited.
func TestApplyWalksEveryStage(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	for i, step := range walk {
		in := approvedInput(t, repo, base, storeDir, step.from, step.to, fmt.Sprintf("nonce-%d", i), now)
		res, err := Apply(context.Background(), in, storeDir)
		if err != nil {
			t.Fatalf("%s -> %s: %v", step.from, step.to, err)
		}
		if res.FromStage != step.from || res.ToStage != step.to {
			t.Fatalf("step %d recorded %s -> %s", i, res.FromStage, res.ToStage)
		}
		if res.MachineAdvanced || !res.RequiresFreshHumanApproval {
			t.Fatalf("step %d result = %+v", i, res)
		}
	}

	records, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != len(walk) {
		t.Fatalf("chain holds %d records, want %d", len(records), len(walk))
	}
	if got := engine.ProjectStageState(records).CurrentStage; got != model.StageReleaseAndClosure {
		t.Fatalf("run replays to %q, want RELEASE_AND_CLOSURE", got)
	}
}

// TestApplyRecordsTheDecisionItWasGiven checks the record's shape: it is filed
// under the stage that was approved, marked APPROVED, and attributed to the
// approver rather than to the host that ran the command.
func TestApplyRecordsTheDecisionItWasGiven(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	in := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if _, err := Apply(context.Background(), in, storeDir); err != nil {
		t.Fatal(err)
	}
	records, err := store.New(storeDir).Load("run")
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %d, %v", len(records), err)
	}
	r := records[0]
	if r.Kind != model.KindStageTransition || r.Control != model.ControlApproved {
		t.Fatalf("record = %+v", r)
	}
	if r.Stage != model.StageGrounding {
		t.Fatalf("record filed under %q, want the stage that was approved", r.Stage)
	}
	if r.Approval == nil || r.Approval.ProposedTargetStage != model.StageAcceptanceCriteria {
		t.Fatalf("record approval = %+v", r.Approval)
	}
	if r.Metadata.Actor != "operator-confirmed:lennon" {
		t.Fatalf("actor = %q, want the approver", r.Metadata.Actor)
	}
}

func TestApplyRejectsUnlistedEdge(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	in := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageVerification, "nonce-1", now)
	_, err := Apply(context.Background(), in, storeDir)
	if err == nil || !strings.Contains(err.Error(), string(engine.ReasonUnlistedEdge)) {
		t.Fatalf("err = %v, want %s", err, engine.ReasonUnlistedEdge)
	}
	if records, _ := store.New(storeDir).Load("run"); len(records) != 0 {
		t.Fatalf("a rejected transition wrote %d records", len(records))
	}
}

// TestApplyRejectsReplayedApproval: an approval is single-use, so presenting
// the same one twice must not move the run twice.
func TestApplyRejectsReplayedApproval(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	in := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if _, err := Apply(context.Background(), in, storeDir); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), in, storeDir); err == nil {
		t.Fatal("a single-use approval was accepted twice")
	}
}

// TestApplyRejectsApprovalForAnotherRunPosition: the approval binds the exact
// chain position, so one prepared against an empty chain cannot be used later.
func TestApplyRejectsApprovalForAnotherRunPosition(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	prepared := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-2", now)

	first := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if _, err := Apply(context.Background(), first, storeDir); err != nil {
		t.Fatal(err)
	}
	// The run is at ACCEPTANCE_CRITERIA now; the prepared approval was
	// raised from GROUNDING against sequence 1.
	if _, err := Apply(context.Background(), prepared, storeDir); err == nil {
		t.Fatal("an approval for an earlier chain position was accepted")
	}
}

// TestApplyRejectsStaleHeadForBindingStage: an IMPLEMENTATION_REVIEW approval
// binds the produced head, so a commit landing after the human approved must
// invalidate it.
func TestApplyRejectsStaleHeadForBindingStage(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	for i, step := range walk[:5] {
		in := approvedInput(t, repo, base, storeDir, step.from, step.to, fmt.Sprintf("nonce-%d", i), now)
		if _, err := Apply(context.Background(), in, storeDir); err != nil {
			t.Fatal(err)
		}
	}
	in := approvedInput(t, repo, base, storeDir, model.StageImplementationReview, model.StageVerification, "nonce-review", now)

	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "x")
	git(t, repo, "commit", "-qm", "two")

	if _, err := Apply(context.Background(), in, storeDir); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("err = %v, want a stale binding refusal", err)
	}
}

// TestApplyIgnoresHeadForNonBindingStage is the other half of the same rule: a
// GROUNDING approval binds no produced head, so a commit does not invalidate
// it. A packet is stale only when a field it actually binds changes.
func TestApplyIgnoresHeadForNonBindingStage(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	in := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)

	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "x")
	git(t, repo, "commit", "-qm", "two")

	if _, err := Apply(context.Background(), in, storeDir); err != nil {
		t.Fatalf("a grounding approval was invalidated by a field it never bound: %v", err)
	}
}

func TestApplyRefusesDirtyWorktreeWithoutApproval(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	in := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("uncommitted"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), in, storeDir); err == nil || !strings.Contains(err.Error(), "dirty worktree") {
		t.Fatalf("err = %v, want a dirty worktree refusal", err)
	}
}

// TestApplyBindsApprovalToApprovedWorktreeContent is the section 6 amendment
// end to end: a worktree-bound approval records against the exact uncommitted
// content the human approved, and once the worktree drifts underneath it the
// transition is refused and the refusal names the binding that failed.
func TestApplyBindsApprovalToApprovedWorktreeContent(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("uncommitted"), 0600); err != nil {
		t.Fatal(err)
	}
	in := approvedDirtyInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if _, err := Apply(context.Background(), in, storeDir); err != nil {
		t.Fatalf("worktree-bound approval refused against the approved content: %v", err)
	}
	records, err := store.New(storeDir).Load("run")
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %d, %v", len(records), err)
	}
	if got := records[0].Approval.WorktreeDigest; got == "" || got != in.ExpectedWorktreeDigest {
		t.Fatalf("record worktree binding = %q, want the approved digest", got)
	}

	// The worktree changes after the human approved. The host still
	// declares the digest the human approved; the approval's own binding is
	// what the refusal must name.
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("different uncommitted"), 0600); err != nil {
		t.Fatal(err)
	}
	drifted := approvedDirtyInput(t, repo, base, storeDir, model.StageAcceptanceCriteria, model.StagePlanning, "nonce-2", now)
	drifted.ExpectedWorktreeDigest = in.ExpectedWorktreeDigest
	drifted.ExpectedApproval.WorktreeDigest = in.ExpectedWorktreeDigest
	drifted.Approval.WorktreeDigest = in.ExpectedWorktreeDigest
	if _, err := Apply(context.Background(), drifted, storeDir); err == nil ||
		!strings.Contains(err.Error(), string(engine.ReasonStaleWorktreeBinding)) {
		t.Fatalf("err = %v, want %s", err, engine.ReasonStaleWorktreeBinding)
	}
	if records, _ := store.New(storeDir).Load("run"); len(records) != 1 {
		t.Fatalf("a refused transition wrote %d records, want the 1 already committed", len(records))
	}
}

func TestApplyRefusesStoreInsideRepository(t *testing.T) {
	repo, base := newRepo(t)
	inner := filepath.Join(repo, ".hma-store")
	if err := os.MkdirAll(inner, 0700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	in := approvedInput(t, repo, base, t.TempDir(), model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if _, err := Apply(context.Background(), in, inner); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("err = %v, want a refusal to keep the store inside the repository", err)
	}
}

func TestApplyRejectsRunAfterTerminalOutcome(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	s := store.New(storeDir)
	terminal := model.Record{
		Kind: model.KindOutcome, Version: 1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: 1, Timestamp: now.Format(time.RFC3339), Actor: "operator-confirmed:lennon"},
		Outcome:  model.OutcomeAborted,
	}
	if err := s.Append(&terminal); err != nil {
		t.Fatal(err)
	}
	in := approvedInput(t, repo, base, storeDir, model.StageGrounding, model.StageAcceptanceCriteria, "nonce-1", now)
	if _, err := Apply(context.Background(), in, storeDir); err == nil || !strings.Contains(err.Error(), string(engine.ReasonRunTerminal)) {
		t.Fatalf("err = %v, want %s", err, engine.ReasonRunTerminal)
	}
}
