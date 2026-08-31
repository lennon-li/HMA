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

func pilotInput(t *testing.T, repo, base, identity, nonce, stageTime string, now time.Time) Input {
	t.Helper()
	snap, err := snapshotRepo(context.Background(), repo, base)
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
	current, err := snapshotRepo(context.Background(), repo, base)
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
