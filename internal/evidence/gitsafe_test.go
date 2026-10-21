package evidence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepoWithConfig(t *testing.T, extra string) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "t"}, {"config", "user.email", "t@example.invalid"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if extra != "" {
		f, err := os.OpenFile(filepath.Join(root, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(extra)
		f.Close()
	}
	return root
}

// Liz D5a r6 F10-F12: a git command is captured only in the approved
// repository, read with git's own grammar; helpers named by any non-standard
// key, including keys hidden after comment backslashes, never start.
func TestCaptureGitRepositoryConfig(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	helper := filepath.Join(t.TempDir(), "helper.sh")
	os.WriteFile(helper, []byte("#!/bin/sh\ntouch "+marker+"\ncat\n"), 0o700)
	attack := func(cfg string) string {
		root := gitRepoWithConfig(t, cfg)
		os.WriteFile(filepath.Join(root, ".git", "info", "attributes"), []byte("a filter=audit\n"), 0o600)
		os.WriteFile(filepath.Join(root, "a"), []byte("one\n"), 0o600)
		return root
	}
	diff := []string{"diff", "--no-ext-diff", "--no-textconv"}
	run := func(root, wd string, argv []string) error {
		_, err := Capture(context.Background(), Request{Executable: "git", Argv: argv, WorkingDir: wd, RepositoryRoot: root, MaxOutputBytes: 1 << 16, CapturingActor: "t"})
		return err
	}
	for name, cfg := range map[string]string{
		"filter":            "[filter \"audit\"]\n\tclean = " + helper + "\n",
		"comment backslash": "[user]\n\tname = Safe # \\\n[filter \"audit\"] # \\\n\tclean = " + helper + "\n",
		"signature":         "[log]\n\tshowSignature = true\n[gpg]\n\tprogram = " + helper + "\n",
		"include":           "[include]\n\tpath = /x\n",
	} {
		root := attack(cfg)
		if err := run(root, root, diff); err == nil {
			t.Errorf("%s: captured", name)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%s: helper ran", name)
		}
	}
	// Controls: standard configuration captures; so does a plain subdirectory.
	root := gitRepoWithConfig(t, "")
	os.Mkdir(filepath.Join(root, "sub"), 0o700)
	if err := run(root, root, []string{"status", "-s"}); err != nil {
		t.Fatalf("control: %v", err)
	}
	if err := run(root, filepath.Join(root, "sub"), []string{"status", "-s"}); err != nil {
		t.Fatalf("subdirectory: %v", err)
	}
	// A nested repository, a gitdir file, and a worktree config are refused.
	nested := filepath.Join(root, "inner")
	os.Mkdir(nested, 0o700)
	if out, err := exec.Command("git", "-C", nested, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if err := run(root, nested, []string{"status", "-s"}); err == nil {
		t.Error("nested repository accepted")
	}
	os.WriteFile(filepath.Join(root, ".git", "config.worktree"), []byte("[core]\n"), 0o600)
	if err := run(root, root, []string{"status", "-s"}); err == nil {
		t.Error("worktree configuration accepted")
	}
	os.Remove(filepath.Join(root, ".git", "config.worktree"))
	// A .git file that points elsewhere is refused.
	other := gitRepoWithConfig(t, "")
	moved := t.TempDir()
	os.Rename(filepath.Join(other, ".git"), filepath.Join(moved, "gitdir"))
	os.WriteFile(filepath.Join(other, ".git"), []byte("gitdir: "+filepath.Join(moved, "gitdir")+"\n"), 0o600)
	if err := run(other, other, []string{"status", "-s"}); err == nil {
		t.Error("gitdir file accepted")
	}
	// A non-git executable is not affected.
	if err := runNonGit(root); err != nil {
		t.Errorf("non-git: %v", err)
	}
	// Key names are judged as git prints them.
	for name, ok := range map[string]bool{"core.filemode": true, "CORE.FileMode": true, "remote.origin.url": true, "branch.main.merge": true,
		"core.pager": false, "filter.audit.clean": false, "log.showsignature": false, "include.path": false, "x": false} {
		if gitConfigNameAllowed(name) != ok {
			t.Errorf("%s: want %v", name, ok)
		}
	}
}

func runNonGit(root string) error {
	_, err := Capture(context.Background(), Request{Executable: "true", WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1024, CapturingActor: "t"})
	return err
}

// Liz/Prof D5a r7 F02-F03: Capture itself applies the git form policy and
// the config check to the program that runs. -C cannot select another
// repository, and a symlink to git is git.
func TestCaptureGitFormsAndAliases(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	helper := filepath.Join(t.TempDir(), "helper.sh")
	os.WriteFile(helper, []byte("#!/bin/sh\ntouch "+marker+"\ncat\n"), 0o700)
	plain := gitRepoWithConfig(t, "")
	hostile := gitRepoWithConfig(t, "[filter \"audit\"]\n\tclean = "+helper+"\n")
	os.WriteFile(filepath.Join(hostile, ".git", "info", "attributes"), []byte("a filter=audit\n"), 0o600)
	os.WriteFile(filepath.Join(hostile, "a"), []byte("one\n"), 0o600)
	diff := []string{"diff", "--no-ext-diff", "--no-textconv"}
	capture := func(exe string, argv []string, root string) error {
		_, err := Capture(context.Background(), Request{Executable: exe, Argv: argv, WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1 << 16, CapturingActor: "t"})
		return err
	}
	if err := capture("git", append([]string{"-C", hostile}, diff...), plain); err == nil {
		t.Error("-C to another repository accepted")
	}
	if err := capture("git", append([]string{"--git-dir=" + filepath.Join(hostile, ".git"), "--work-tree=" + hostile}, diff...), plain); err == nil {
		t.Error("--git-dir accepted")
	}
	if err := capture("git", []string{"-c", "core.pager=" + helper, "status"}, plain); err == nil {
		t.Error("-c accepted")
	}
	if err := capture("git", []string{"config", "a.b", "c"}, plain); err == nil {
		t.Error("git config accepted")
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "approved-check")
	if err := os.Symlink(gitPath, alias); err != nil {
		t.Fatal(err)
	}
	if err := capture(alias, diff, hostile); err == nil {
		t.Error("symlink alias to git skipped the repository check")
	}
	if err := capture(alias, []string{"config", "a.b", "c"}, plain); err == nil {
		t.Error("symlink alias to git skipped the form policy")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("helper ran")
	}
	// Controls: the same forms in the approved plain repository, directly and through the alias.
	if err := capture("git", []string{"status", "-s"}, plain); err != nil {
		t.Errorf("control: %v", err)
	}
	if err := capture(alias, []string{"status", "-s"}, plain); err != nil {
		t.Errorf("alias control: %v", err)
	}
	// The record names the program that ran, not the alias.
	res, err := Capture(context.Background(), Request{Executable: alias, Argv: []string{"status", "-s"}, WorkingDir: plain, RepositoryRoot: plain, MaxOutputBytes: 1 << 16, CapturingActor: "t"})
	if want, _ := filepath.EvalSymlinks(gitPath); err != nil || res.Evidence.Executable != want {
		t.Errorf("recorded executable %q, want %q (%v)", res.Evidence.Executable, want, err)
	}
}

// Liz/Prof D5a r9 F02: git runs against a private copy of the validated
// configuration, so a writer that edits .git/config after validation cannot
// make git read a helper setting; the real repository is not written.
func TestCaptureGitUsesValidatedConfig(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	helper := filepath.Join(t.TempDir(), "helper.sh")
	os.WriteFile(helper, []byte("#!/bin/sh\ntouch "+marker+"\ncat\n"), 0o700)
	root := gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, ".git", "info", "attributes"), []byte("a filter=audit\n"), 0o600)
	os.WriteFile(filepath.Join(root, "a"), []byte("one\n"), 0o600)
	// Through Capture, git runs against the private directory, not the real one.
	res, err := Capture(context.Background(), Request{Executable: "git", Argv: []string{"rev-parse", "--git-dir"}, WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1024, CapturingActor: "t"})
	if err != nil || !strings.Contains(string(res.Stdout), "hma-gitdir-") {
		t.Fatalf("capture did not run against a private git directory: %q %v", res.Stdout, err)
	}
	cfg := filepath.Join(root, ".git", "config")
	orig, _ := os.ReadFile(cfg)
	// Edit the real configuration right after validation reads it: the
	// first git process of the capture (the validation) is followed by the
	// edit through the Capture's own environment hook of a slow wrapper.
	priv, err := PrepareGitExecution(mustGit(t), root, root, root, "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(priv)
	os.WriteFile(cfg, append(orig, []byte("[filter \"audit\"]\n\tclean = "+helper+"\n")...), 0o600)
	cmd := exec.Command(mustGit(t), "diff", "--no-ext-diff", "--no-textconv")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GIT_DIR="+priv, "GIT_WORK_TREE="+root)
	if out, err := cmd.CombinedOutput(); err != nil && len(out) == 0 {
		t.Logf("diff: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("helper named by a post-validation edit ran")
	}
	// The private directory holds a read-only copy and links, no writable config of the real repo.
	if b, _ := os.ReadFile(filepath.Join(priv, "config")); string(b) != string(orig) {
		t.Fatal("private configuration is not the validated bytes")
	}
	// Cleanup of the real repo's files: nothing new written by validation.
	if _, err := os.Stat(filepath.Join(root, ".git", "config.lock")); err == nil {
		t.Fatal("lock file left behind")
	}
}

func mustGit(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Git run against the private directory still sees the real index, HEAD and
// objects: a modified tracked file shows as modified.
func TestCaptureGitSeesRealIndex(t *testing.T) {
	root := gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, "t"), []byte("one\n"), 0o600)
	for _, args := range [][]string{{"add", "t"}, {"commit", "-qm", "c"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	os.WriteFile(filepath.Join(root, "t"), []byte("two\n"), 0o600)
	res, err := Capture(context.Background(), Request{Executable: "git", Argv: []string{"status", "-s"}, WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1024, CapturingActor: "t"})
	if err != nil || !strings.Contains(string(res.Stdout), " M t") {
		t.Fatalf("status through the private directory: %q %v", res.Stdout, err)
	}
}

func gitIn(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// D5a r15 D5AR15P-F01 / r17 D5AR17P-F01: a staged target is refused from the
// bound snapshot's state, never from a live-index read at preparation time.
func TestCaptureRefusesStagedTargetForSandboxedGit(t *testing.T) {
	root := gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, "f"), []byte("one\n"), 0o600)
	gitIn(t, root, "add", "f")
	gitIn(t, root, "commit", "-qm", "base")
	head := gitIn(t, root, "rev-parse", "HEAD")
	req := Request{Executable: mustGit(t), Argv: []string{"diff", "--cached", "--quiet", "--no-ext-diff", "--no-textconv"}, WorkingDir: root,
		RepositoryRoot: root, Sandbox: true, PinnedHead: head, StagedTarget: true}
	if _, err := Capture(context.Background(), req); err == nil || !strings.Contains(err.Error(), "staged changes") {
		t.Fatalf("staged target not refused for sandboxed git: %v", err)
	}
}

// D5a r15 D5AR15P-F02: the view's configuration is fixed, so a local
// core.filemode change cannot turn a failing mode check into a passing one.
func TestPinnedViewIgnoresLocalFilemode(t *testing.T) {
	root := gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, "f"), []byte("one\n"), 0o600)
	gitIn(t, root, "add", "f")
	gitIn(t, root, "commit", "-qm", "base")
	head := gitIn(t, root, "rev-parse", "HEAD")
	os.Chmod(filepath.Join(root, "f"), 0o700)
	diffExit := func() int {
		priv, err := PrepareGitExecution(mustGit(t), root, root, root, head)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(priv)
		c := exec.Command(mustGit(t), "diff", "--quiet", "--no-ext-diff", "--no-textconv")
		c.Dir = root
		c.Env = append(os.Environ(), "GIT_DIR="+priv, "GIT_WORK_TREE="+root, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		if err := c.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ExitCode()
			}
			t.Fatal(err)
		}
		return 0
	}
	before := diffExit()
	if before != 1 {
		t.Fatalf("control: mode change not seen, exit %d", before)
	}
	gitIn(t, root, "config", "core.filemode", "false")
	if after := diffExit(); after != 1 {
		t.Fatalf("local core.filemode=false changed the result to %d", after)
	}
}

// D5a r16: no git subprocess of the preparation step reads the live
// .git/config: a helper named there never starts, even before the sandbox.
func TestPreparationStartsNoLiveConfigHelper(t *testing.T) {
	root := gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, "f"), []byte("one\n"), 0o600)
	gitIn(t, root, "add", "f")
	gitIn(t, root, "commit", "-qm", "base")
	head := gitIn(t, root, "rev-parse", "HEAD")
	marker := filepath.Join(t.TempDir(), "marker")
	helper := filepath.Join(t.TempDir(), "fsm.sh")
	os.WriteFile(helper, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o700)
	// The edit lands after the configuration copy is validated, as in the
	// audit's race: the live config now names a monitor.
	cfg := filepath.Join(root, ".git", "config")
	orig, _ := os.ReadFile(cfg)
	prepareHook = func() {
		os.WriteFile(cfg, append(orig, []byte("[core]\n\tfsmonitor = "+helper+"\n")...), 0o600)
	}
	defer func() { prepareHook = nil }()
	if priv, err := PrepareGitExecution(mustGit(t), root, root, root, head); err == nil {
		os.RemoveAll(priv)
	}
	if _, serr := os.Stat(marker); serr == nil {
		t.Fatal("a helper named in the live configuration started")
	}
}

// D5a r22 (Prof's partial audit): an object overwritten in the live store
// under a reviewed id is refused when the pinned store is built, not packed as
// reviewed content.
func TestPinnedStoreRefusesSubstitutedObject(t *testing.T) {
	root := gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, "f"), []byte("reviewed\n"), 0o600)
	gitIn(t, root, "add", "f")
	gitIn(t, root, "commit", "-qm", "base")
	head := gitIn(t, root, "rev-parse", "HEAD")
	oid := gitIn(t, root, "rev-parse", "HEAD:f")
	other := gitIn(t, root, "hash-object", "-w", "--stdin")
	_ = other
	loose := filepath.Join(root, ".git", "objects", oid[:2], oid[2:])
	if _, err := os.Stat(loose); err != nil {
		t.Skip("object is not loose")
	}
	sub := exec.Command("git", "-C", root, "hash-object", "-w", "--stdin")
	sub.Stdin = strings.NewReader("POISONED\n")
	subOut, err := sub.Output()
	if err != nil {
		t.Fatal(err)
	}
	poison := strings.TrimSpace(string(subOut))
	data, err := os.ReadFile(filepath.Join(root, ".git", "objects", poison[:2], poison[2:]))
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod(loose, 0o600)
	if err := os.WriteFile(loose, data, 0o400); err != nil {
		t.Fatal(err)
	}
	if priv, err := PrepareGitExecution(mustGit(t), root, root, root, head); err == nil {
		os.RemoveAll(priv)
		t.Fatal("a substituted object was packed as reviewed content")
	}
}
