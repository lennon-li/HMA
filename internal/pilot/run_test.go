package pilot

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

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
	return repo, trim(git(t, repo, "rev-parse", "HEAD"))
}

// pilotInput snapshots with dirty capture allowed so that a dirty-worktree
// fixture can be built at all; Run, not this helper, is what enforces the
// dirty-worktree policy under test.
func pilotInput(t *testing.T, repo, base, identity, nonce, stageTime string, now time.Time) Input {
	t.Helper()
	snap, err := snapshotRepo(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	a := model.ApprovalBinding{RunID: "run", TransitionDigest: "transition", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification, RepositoryIdentityDigest: identity, BaseRevisionDigest: base, StageTimeDigest: stageTime, AcceptedPlanDigest: "plan", ChallengeNonce: nonce, Approver: "operator-confirmed:lennon", Timestamp: now.Format(time.RFC3339), ProducedHeadDigest: snap.head, DiffDigest: snap.diff}
	return Input{RunID: "run", RepositoryRoot: repo, RepositoryIdentity: identity, BaseRevision: base, ExpectedHeadRevision: snap.head, ExpectedDiffDigest: snap.diff, Actor: "host", ExpectedApproval: a, Approval: a, ApprovalNow: now, ApprovalMaxAgeSeconds: 300, EvidenceCommands: []Command{{Executable: "git", Argv: []string{"status", "--porcelain"}}}}
}

func TestRunAcceptsOpaqueIdentityAndPersistsItInEvidence(t *testing.T) {
	repo, base := newRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	in := pilotInput(t, repo, base, "sha256:opaque-repository-identity", "nonce-1", ":1", now)
	res, err := Run(context.Background(), in, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != engine.DecisionLegalPendingApproval || res.MachineAdvanced || !res.RequiresFreshHumanApproval {
		t.Fatalf("result = %+v", res)
	}
	if res.Evidence.RepositoryIdentity != in.RepositoryIdentity {
		t.Fatalf("evidence repository identity = %q, want opaque digest %q", res.Evidence.RepositoryIdentity, in.RepositoryIdentity)
	}
}

func TestRunRejectsPersistedApprovalNonceReplay(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	in := pilotInput(t, repo, base, "repo-digest", "nonce-1", ":1", now)
	if _, err := Run(context.Background(), in, storeDir); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), in, storeDir); err == nil || !strings.Contains(err.Error(), string(engine.ReasonReusedNonce)) {
		t.Fatalf("replay error = %v, want %s", err, engine.ReasonReusedNonce)
	}
}

func TestRunUsesRealSuccessorStageTime(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	first := pilotInput(t, repo, base, "repo-digest", "nonce-1", ":1", now)
	if _, err := Run(context.Background(), first, storeDir); err != nil {
		t.Fatal(err)
	}
	records, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	predecessor := records[len(records)-1].Metadata.HeadAnchor
	second := pilotInput(t, repo, base, "repo-digest", "nonce-2", predecessor+":3", now)
	if _, err := Run(context.Background(), second, storeDir); err != nil {
		t.Fatalf("real successor rejected: %v", err)
	}
	records, err = store.New(storeDir).Load("run")
	if err != nil || len(records) != 4 {
		t.Fatalf("successor stream = %d records, %v", len(records), err)
	}
}

func TestRunRejectsMismatchedStageTime(t *testing.T) {
	repo, base := newRepo(t)
	storeDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	first := pilotInput(t, repo, base, "repo-digest", "nonce-1", ":1", now)
	if _, err := Run(context.Background(), first, storeDir); err != nil {
		t.Fatal(err)
	}
	mismatched := pilotInput(t, repo, base, "repo-digest", "nonce-2", "wrong-head:99", now)
	if _, err := Run(context.Background(), mismatched, storeDir); err == nil || !strings.Contains(err.Error(), string(engine.ReasonStaleApproval)) {
		t.Fatalf("stage-time mismatch error = %v, want %s", err, engine.ReasonStaleApproval)
	}
}

func TestRunCommittedDeltaChangesDiffAndRejectsStaleBinding(t *testing.T) {
	repo, base := newRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	stale := pilotInput(t, repo, base, "repo-digest", "nonce-1", ":1", now)
	oldDiff := stale.ExpectedDiffDigest
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "x")
	git(t, repo, "commit", "-qm", "two")
	current, err := snapshotRepo(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if current.diff == oldDiff {
		t.Fatal("committed repository delta did not change relevant diff digest")
	}
	stale.ExpectedHeadRevision = current.head
	stale.ExpectedDiffDigest = current.diff
	if _, err := Run(context.Background(), stale, t.TempDir()); err == nil || !strings.Contains(err.Error(), "expected approval binding is stale") {
		t.Fatalf("stale binding error = %v, want expected approval binding path", err)
	}
}

// dirtyPilotInput builds an input for an approved dirty-worktree capture,
// binding the worktree digest the human approved.
func dirtyPilotInput(t *testing.T, repo, base string, now time.Time) Input {
	t.Helper()
	snap, err := snapshotRepo(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if snap.worktree == "" {
		t.Fatal("worktree digest empty for a dirty worktree")
	}
	in := pilotInput(t, repo, base, "identity", "nonce-dirty", ":1", now)
	in.DirtyWorktreeApproved = true
	in.ExpectedWorktreeDigest = snap.worktree
	return in
}

func TestRunRefusesDirtyWorktreeWithoutApproval(t *testing.T) {
	repo, base := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	in := pilotInput(t, repo, base, "identity", "nonce-1", ":1", now)
	_, err := Run(context.Background(), in, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "dirty worktree") {
		t.Fatalf("err = %v, want a dirty worktree refusal", err)
	}
}

// TestRunBindsApprovedDirtyWorktree covers architecture section 14: approved
// dirty evidence is admissible only when the record carries the exact
// worktree-content digest.
func TestRunBindsApprovedDirtyWorktree(t *testing.T) {
	repo, base := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	in := dirtyPilotInput(t, repo, base, now)

	res, err := Run(context.Background(), in, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if res.Evidence.WorktreeDigest == "" {
		t.Fatal("dirty evidence recorded without a worktree digest")
	}
	if res.Evidence.WorktreeDigest != in.ExpectedWorktreeDigest {
		t.Fatal("recorded worktree digest is not the approved one")
	}
}

// TestRunRejectsStaleWorktreeContent is the point of binding the digest:
// approving "dirty" must not mean approving whatever is on disk later.
func TestRunRejectsStaleWorktreeContent(t *testing.T) {
	repo, base := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("approved content"), 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	in := dirtyPilotInput(t, repo, base, now)

	// The worktree changes after the human approved that exact content.
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("something else"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), in, t.TempDir()); err == nil {
		t.Fatal("evidence captured against unapproved worktree content")
	}
}

func TestRunRejectsIncompleteDirtyApproval(t *testing.T) {
	repo, base := newRepo(t)
	now := time.Now().UTC().Truncate(time.Second)

	approvedWithoutDigest := pilotInput(t, repo, base, "identity", "nonce-1", ":1", now)
	approvedWithoutDigest.DirtyWorktreeApproved = true
	if _, err := Run(context.Background(), approvedWithoutDigest, t.TempDir()); err == nil {
		t.Fatal("dirty approval accepted with no expected worktree digest")
	}

	digestWithoutApproval := pilotInput(t, repo, base, "identity", "nonce-2", ":1", now)
	digestWithoutApproval.ExpectedWorktreeDigest = "some-digest"
	if _, err := Run(context.Background(), digestWithoutApproval, t.TempDir()); err == nil {
		t.Fatal("worktree digest accepted without dirty approval")
	}
}

// TestWorktreeDigestDistinguishesContent guards the framing: two different
// worktrees must not collide, and the digest must be stable for one state.
func TestWorktreeDigestDistinguishesContent(t *testing.T) {
	repo, base := newRepo(t)

	write := func(content string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, "x"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		snap, err := snapshotRepo(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		return snap.worktree
	}

	first := write("ab")
	if first != write("ab") {
		t.Fatal("worktree digest is not stable for one worktree state")
	}
	if first == write("ba") {
		t.Fatal("different worktree content produced the same digest")
	}
}

func TestSnapshotLeavesCleanWorktreeDigestEmpty(t *testing.T) {
	repo, base := newRepo(t)
	snap, err := snapshotRepo(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if snap.worktree != "" {
		t.Fatal("clean worktree produced a worktree digest; it is revision-reproducible")
	}
}
