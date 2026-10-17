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

// B11: the worktree digest binds executable mode and file type, not bytes alone.
func TestWorktreeDigestBindsModeAndType(t *testing.T) {
	repo, base := newRepo(t)
	path := filepath.Join(repo, "x")
	if err := os.WriteFile(path, []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := func() string {
		t.Helper()
		st, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		return st.Worktree
	}
	regular := digest()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := digest()
	if executable == regular {
		t.Fatal("executable-mode change kept the worktree digest")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if digest() != regular {
		t.Fatal("restoring the mode did not restore the digest")
	}
	target := filepath.Join(t.TempDir(), "same-bytes")
	if err := os.WriteFile(target, []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if digest() == regular {
		t.Fatal("symlink to identical bytes kept the worktree digest")
	}
}

// B11 residual: Git distinguishes 0610 (100644) from 0710 (100755); the digest
// must too, and must not change for group/other execute bits Git ignores.
func TestWorktreeDigestBindsOwnerExecuteBit(t *testing.T) {
	repo, base := newRepo(t)
	path := filepath.Join(repo, "x")
	if err := os.WriteFile(path, []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := func(mode os.FileMode) string {
		t.Helper()
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		st, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		return st.Worktree
	}
	if digest(0o610) == digest(0o710) {
		t.Fatal("owner-execute change (Git 100644 -> 100755) kept the worktree digest")
	}
	if digest(0o600) != digest(0o611) {
		t.Fatal("group/other execute bits Git ignores changed the worktree digest")
	}
}

// B12: repository config must not hide a change from a snapshot.
func TestSnapshotIgnoresConfigThatHidesChanges(t *testing.T) {
	cases := []struct {
		name   string
		config []string
		change func(t *testing.T, repo string)
	}{
		{"core.filemode=false hides owner-execute", []string{"core.filemode", "false"}, func(t *testing.T, repo string) {
			if err := os.Chmod(filepath.Join(repo, "x"), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"status.showUntrackedFiles=no hides new files", []string{"status.showUntrackedFiles", "no"}, func(t *testing.T, repo string) {
			if err := os.WriteFile(filepath.Join(repo, "new"), []byte("n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := newRepo(t)
			git(t, repo, append([]string{"config"}, tc.config...)...)
			if _, err := Snapshot(context.Background(), repo, base, false); err != nil {
				t.Fatalf("clean control: %v", err)
			}
			tc.change(t, repo)
			if _, err := Snapshot(context.Background(), repo, base, false); err == nil {
				t.Fatal("hidden change read as a clean worktree")
			}
			st, err := Snapshot(context.Background(), repo, base, true)
			if err != nil || st.Worktree == "" {
				t.Fatalf("approved-dirty snapshot did not bind the change: %+v, %v", st, err)
			}
		})
	}
}

// B12/B13: a clean filter or text/eol attribute makes git status compare
// converted bytes. Raw comparison must still see the change on disk.
func TestSnapshotSeesChangesHiddenByConversion(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, repo string)
		write []byte
	}{
		{"clean filter returning committed bytes", func(t *testing.T, repo string) {
			git(t, repo, "config", "filter.hide.clean", "sh -c 'cat >/dev/null; printf one'")
			writeAttr(t, repo, "x filter=hide\n")
		}, []byte("two")},
		{"eol normalization", func(t *testing.T, repo string) {
			writeAttr(t, repo, "x text eol=lf\n")
		}, []byte("one\r\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := newRepo(t)
			tc.setup(t, repo)
			if _, err := Snapshot(context.Background(), repo, base, false); err != nil {
				t.Fatalf("clean control: %v", err)
			}
			if err := os.WriteFile(filepath.Join(repo, "x"), tc.write, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Snapshot(context.Background(), repo, base, false); err == nil {
				t.Fatal("converted change read as a clean worktree")
			}
			st, err := Snapshot(context.Background(), repo, base, true)
			if err != nil || st.Worktree == "" {
				t.Fatalf("approved-dirty snapshot did not bind the change: %+v, %v", st, err)
			}
		})
	}
}

// writeAttr writes repository-local attributes outside the worktree, so the
// attribute file itself is not an untracked change.
func writeAttr(t *testing.T, repo, content string) {
	t.Helper()
	info := filepath.Join(repo, ".git", "info")
	if err := os.MkdirAll(info, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info, "attributes"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// B14: submodules and sparse checkout are refused, not silently skipped.
func TestSnapshotRefusesSubmoduleAndSparse(t *testing.T) {
	repo, base := newRepo(t)
	sub, _ := newRepo(t)
	git(t, repo, "-c", "protocol.file.allow=always", "submodule", "add", "-q", sub, "sub")
	git(t, repo, "commit", "-qm", "sub")
	if _, err := Snapshot(context.Background(), repo, base, true); err == nil || !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("submodule not refused: %v", err)
	}
	repo2, base2 := newRepo(t)
	git(t, repo2, "config", "core.sparseCheckout", "true")
	if _, err := Snapshot(context.Background(), repo2, base2, true); err == nil || !strings.Contains(err.Error(), "sparse") {
		t.Fatalf("sparse checkout not refused: %v", err)
	}
}

// Host-local excludes and an untracked self-ignoring .gitignore must not hide
// new untracked files.
func TestSnapshotSeesUntrackedHiddenByLocalExcludes(t *testing.T) {
	cases := map[string]func(t *testing.T, repo string){
		"core.excludesFile": func(t *testing.T, repo string) {
			ex := filepath.Join(t.TempDir(), "ignore-all")
			if err := os.WriteFile(ex, []byte("*\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			git(t, repo, "config", "core.excludesFile", ex)
		},
		"info/exclude": func(t *testing.T, repo string) {
			if err := os.WriteFile(filepath.Join(repo, ".git", "info", "exclude"), []byte("*\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"untracked self-ignoring .gitignore": func(t *testing.T, repo string) {
			if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("*\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, hide := range cases {
		t.Run(name, func(t *testing.T) {
			repo, base := newRepo(t)
			if err := os.MkdirAll(filepath.Join(repo, ".git", "info"), 0o700); err != nil {
				t.Fatal(err)
			}
			hide(t, repo)
			if err := os.WriteFile(filepath.Join(repo, "new"), []byte("n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Snapshot(context.Background(), repo, base, false); err == nil {
				t.Fatal("untracked file hidden by host-local excludes read as clean")
			}
		})
	}
}

// A committed .gitignore still excludes its build outputs (control).
func TestSnapshotHonoursCommittedGitignore(t *testing.T) {
	repo, _ := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("out/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-qm", "ignore")
	base := Trim(git(t, repo, "rev-parse", "HEAD"))
	if err := os.MkdirAll(filepath.Join(repo, "out"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "out", "bin"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(context.Background(), repo, base, false); err != nil {
		t.Fatalf("committed .gitignore output treated as dirty: %v", err)
	}
}

// B15: core.ignorecase must not hide an untracked case alias.
func TestSnapshotSeesCaseAliasUnderIgnoreCase(t *testing.T) {
	repo, base := newRepo(t)
	git(t, repo, "config", "core.ignorecase", "true")
	if err := os.WriteFile(filepath.Join(repo, "X"), []byte("alias"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(context.Background(), repo, base, false); err == nil {
		t.Fatal("untracked case alias read as a clean worktree")
	}
}

// B16: a subdirectory of the work tree is not a repository root.
func TestSnapshotRefusesSubdirectoryRoot(t *testing.T) {
	repo, base := newRepo(t)
	sub := filepath.Join(repo, "build-copy")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "x"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(context.Background(), sub, base, true); err == nil || !strings.Contains(err.Error(), "top level") {
		t.Fatalf("subdirectory root accepted: %v", err)
	}
}

// B17: a replacement ref must not let a different committed tree stand for
// HEAD; snapshots read the original objects.
func TestSnapshotIgnoresReplaceRefs(t *testing.T) {
	repo, base := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "commit", "-qam", "two")
	head := Trim(git(t, repo, "rev-parse", "HEAD"))
	want, err := Snapshot(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	// Replace HEAD's tree with base's tree, then make the disk match base.
	baseTree := Trim(git(t, repo, "rev-parse", base+"^{tree}"))
	headTree := Trim(git(t, repo, "rev-parse", head+"^{tree}"))
	git(t, repo, "replace", headTree, baseTree)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Snapshot(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Diff != want.Diff || got.Worktree == "" {
		t.Fatalf("replace ref changed the observed repository: want diff %s with dirty x, got %+v", want.Diff, got)
	}
}

// B18: a relative path to the top level is a valid root.
func TestSnapshotAcceptsRelativeTopLevelRoot(t *testing.T) {
	repo, base := newRepo(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	if _, err := Snapshot(context.Background(), ".", base, false); err != nil {
		t.Fatalf("relative top-level root refused: %v", err)
	}
}

// F01: restaging different content while the disk bytes stay the same is a
// different artifact.
func TestWorktreeDigestBindsStagedContent(t *testing.T) {
	repo, base := newRepo(t)
	path := filepath.Join(repo, "x")
	stage := func(content string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		git(t, repo, "add", "x")
		if err := os.WriteFile(path, []byte("disk"), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		return st.Worktree
	}
	if stage("staged-A") == stage("staged-B") {
		t.Fatal("different staged content under identical disk bytes kept the worktree digest")
	}
}

// F04: diff.ignoreSubmodules must not drop a gitlink change from the
// approved base..head diff.
func TestSnapshotDiffIncludesHistoricalGitlink(t *testing.T) {
	repo, _ := newRepo(t)
	sub, _ := newRepo(t)
	subHead := Trim(git(t, sub, "rev-parse", "HEAD"))
	git(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+subHead+",sub")
	git(t, repo, "commit", "-qm", "gitlink")
	base := Trim(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "rm", "-q", "--cached", "sub")
	git(t, repo, "commit", "-qm", "drop gitlink")
	want, err := Snapshot(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	git(t, repo, "config", "diff.ignoreSubmodules", "all")
	got, err := Snapshot(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Diff != want.Diff {
		t.Fatalf("diff.ignoreSubmodules changed the approved diff identity: %s != %s", got.Diff, want.Diff)
	}
}

// C01: an uncommitted change to a committed .gitignore must not hide the
// files it now ignores.
func TestSnapshotSeesFilesHiddenByChangedGitignore(t *testing.T) {
	repo, _ := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("out/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-qm", "ignore")
	base := Trim(git(t, repo, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("out/\nhidden\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := func(content string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, "hidden"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		return st.Worktree
	}
	if digest("A") == digest("B") {
		t.Fatal("file hidden by an uncommitted .gitignore change kept the worktree digest")
	}
}
