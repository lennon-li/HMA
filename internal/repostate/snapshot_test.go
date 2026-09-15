package repostate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	return repo, Trim(git(t, repo, "rev-parse", "HEAD"))
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
		snap, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		return snap.Worktree
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
	snap, err := Snapshot(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Worktree != "" {
		t.Fatal("clean worktree produced a worktree digest; it is revision-reproducible")
	}
}

func TestSnapshotRefusesUnapprovedDirtyWorktree(t *testing.T) {
	repo, base := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("dirty"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(context.Background(), repo, base, false); err == nil || !strings.Contains(err.Error(), "dirty worktree") {
		t.Fatalf("err = %v, want a dirty worktree refusal", err)
	}
}

// TestVerifyBaseRejectsInexactRevision is the point of VerifyBase: an
// approval binds an exact base-revision digest, so a name that can resolve to
// a different commit later is not an acceptable spelling of it.
func TestVerifyBaseRejectsInexactRevision(t *testing.T) {
	repo, base := newRepo(t)
	if err := VerifyBase(context.Background(), repo, base); err != nil {
		t.Fatalf("exact base revision rejected: %v", err)
	}
	for _, inexact := range []string{"HEAD", base[:8]} {
		if err := VerifyBase(context.Background(), repo, inexact); err == nil {
			t.Fatalf("inexact revision %q accepted as an exact base", inexact)
		}
	}
	if err := VerifyBase(context.Background(), repo, strings.Repeat("0", 40)); err == nil {
		t.Fatal("nonexistent revision accepted")
	}
}

// TestStatusPathsKeepsRenameSource covers the branch that a rename entry is
// followed by its source path: both sides of the rename are part of what
// changed, and dropping the source would let two different worktrees share a
// digest.
func TestStatusPathsKeepsRenameSource(t *testing.T) {
	// Porcelain -z: "R  new\0old\0" plus an unrelated modification.
	got := statusPaths("R  new\x00old\x00 M other\x00")
	want := []string{"new", "old", "other"}
	if len(got) != len(want) {
		t.Fatalf("statusPaths = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("statusPaths = %q, want %q", got, want)
		}
	}
}

// TestStoreOutsideRepositoryRejectsSymlinkedStore is the point of resolving
// symlinks: a store that reaches into the repository through a link is inside
// it, and a path comparison that never resolves would let it through.
func TestStoreOutsideRepositoryRejectsSymlinkedStore(t *testing.T) {
	repo, _ := newRepo(t)
	outside := t.TempDir()

	link := filepath.Join(outside, "store-link")
	if err := os.Symlink(filepath.Join(repo, "hidden-store"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := StoreOutsideRepository(repo, link); err == nil {
		t.Fatal("a store symlinked into the repository was accepted")
	}
}

func TestStoreOutsideRepositoryBoundary(t *testing.T) {
	repo, _ := newRepo(t)
	outside := t.TempDir()

	cases := []struct {
		name           string
		root, storeDir string
		wantErr        bool
	}{
		{"store beside the repository", repo, outside, false},
		{"store inside the repository", repo, filepath.Join(repo, ".hma"), true},
		{"store is the repository", repo, repo, true},
		// A store directory that does not exist yet is the ordinary first
		// run; it must still be judged by where it would be created.
		{"uncreated store inside the repository", repo, filepath.Join(repo, "a", "b", "c"), true},
		{"uncreated store outside the repository", repo, filepath.Join(outside, "a", "b"), false},
		{"empty repository root", "", outside, true},
		{"empty store directory", repo, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := StoreOutsideRepository(tc.root, tc.storeDir)
			if tc.wantErr != (err != nil) {
				t.Fatalf("StoreOutsideRepository(%q, %q) = %v, wantErr %v", tc.root, tc.storeDir, err, tc.wantErr)
			}
		})
	}
}

// TestHasAncestorComparesIdentityNotSpelling uses an unresolved symlink as a
// second spelling of the repository, standing in for a case alias or bind
// mount that string comparison cannot see.
func TestHasAncestorComparesIdentityNotSpelling(t *testing.T) {
	repo, _ := newRepo(t)
	outside := t.TempDir()
	alias := filepath.Join(outside, "repo-alias")
	if err := os.Symlink(repo, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := os.Stat(repo)
	if err != nil {
		t.Fatal(err)
	}
	if inside, err := hasAncestor(filepath.Join(alias, "not", "created"), root); err != nil || !inside {
		t.Fatalf("a second spelling of the repository was not recognised: %v, %v", inside, err)
	}
	if inside, err := hasAncestor(filepath.Join(outside, "store"), root); err != nil || inside {
		t.Fatalf("a store beside the repository was treated as inside it: %v, %v", inside, err)
	}
}

func TestHasAncestorFailsClosedOnUnreadableComponent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	repo, _ := newRepo(t)
	root, err := os.Stat(repo)
	if err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if _, err := hasAncestor(filepath.Join(locked, "inner", "store"), root); err == nil {
		t.Fatal("an unreadable path component was skipped instead of refused")
	}
}

// TestRepositorySelectingEnvMatchesGit fails when the installed git reports a
// repository-selecting variable the list does not remove, so a git upgrade
// cannot silently reopen the redirect.
func TestRepositorySelectingEnvMatchesGit(t *testing.T) {
	out, err := exec.Command("git", "rev-parse", "--local-env-vars").Output()
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, name := range repositorySelectingEnv {
		listed[name] = true
	}
	for _, name := range strings.Fields(string(out)) {
		if strings.HasPrefix(name, "GIT_CONFIG") {
			continue
		}
		if !listed[name] {
			t.Errorf("git reports repository-selecting variable %s, which Output does not remove", name)
		}
	}
}

func TestStoreOutsideRepositoryRequiresExistingRoot(t *testing.T) {
	outside := t.TempDir()
	if err := StoreOutsideRepository(filepath.Join(outside, "missing-repo"), filepath.Join(outside, "store")); err == nil {
		t.Fatal("a nonexistent repository root passed the containment check")
	}
}

func TestOutputIgnoresRepositorySelectingEnvironment(t *testing.T) {
	repo, _ := newRepo(t)
	other, _ := newRepo(t)
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(other, ".git", "index"))
	// Injected configuration is how hosts set safe.directory; it must survive.
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "hma.probe")
	t.Setenv("GIT_CONFIG_VALUE_0", "kept")

	top, err := Output(context.Background(), repo, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := filepath.EvalSymlinks(Trim(top)); got != want {
		t.Fatalf("inherited GIT_DIR/GIT_WORK_TREE redirected git: top level %q, want %q", got, want)
	}
	value, err := Output(context.Background(), repo, "config", "--get", "hma.probe")
	if err != nil || Trim(value) != "kept" {
		t.Fatalf("GIT_CONFIG_* injection was dropped: %q, %v", value, err)
	}

	// The kept channel must not redirect the work tree either: an injected
	// core.worktree pointing at a dirty repository leaves dir's status clean.
	if err := os.WriteFile(filepath.Join(other, "dirt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_KEY_0", "core.worktree")
	t.Setenv("GIT_CONFIG_VALUE_0", other)
	status, err := Output(context.Background(), repo, "status", "--porcelain")
	if err != nil || Trim(status) != "" {
		t.Fatalf("injected core.worktree changed the status of dir: %q, %v", status, err)
	}
}
