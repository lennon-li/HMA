package repostate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Materialize copies the reviewed target only: tracked and changed paths,
// not committed-ignored files, caches, or .git.
func TestMaterializeCopiesReviewedTargetOnly(t *testing.T) {
	root := fpRepo(t)
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("cache/\n"), 0o600)
	os.WriteFile(filepath.Join(root, "run.sh"), []byte("#!/bin/sh\n"), 0o700)
	git("add", "-A")
	git("commit", "-qm", "base")
	head := Trim(git("rev-parse", "HEAD"))
	os.WriteFile(filepath.Join(root, "a"), []byte("edited\n"), 0o600)
	os.WriteFile(filepath.Join(root, "new"), []byte("n\n"), 0o600)
	os.Mkdir(filepath.Join(root, "cache"), 0o700)
	os.WriteFile(filepath.Join(root, "cache", "x"), []byte("c\n"), 0o600)
	os.Symlink("a", filepath.Join(root, "link"))
	dest := t.TempDir()
	changed, err := Materialize(context.Background(), root, head, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) == 0 {
		t.Fatal("changed paths not returned")
	}
	read := func(p string) string { b, _ := os.ReadFile(filepath.Join(dest, p)); return string(b) }
	if read("a") != "edited\n" || read("new") != "n\n" || read("sub/b") != "x\n" {
		t.Fatal("tracked, changed or untracked content not copied")
	}
	if fi, err := os.Stat(filepath.Join(dest, "run.sh")); err != nil || fi.Mode().Perm()&0o100 == 0 {
		t.Fatal("executable bit lost")
	}
	if tgt, err := os.Readlink(filepath.Join(dest, "link")); err != nil || tgt != "a" {
		t.Fatal("symlink not copied")
	}
	for _, p := range []string{"cache", ".git"} {
		if _, err := os.Lstat(filepath.Join(dest, p)); err == nil {
			t.Fatalf("%s copied", p)
		}
	}
}

// A symlink that points outside the copy is refused: absolute, climbing
// above the root, or reaching the root's parent through a subdirectory.
func TestMaterializeRefusesEscapingSymlink(t *testing.T) {
	for name, target := range map[string]string{"abs": "/etc/hostname", "up": "../outside", "deep": "sub/../../outside"} {
		root := fpRepo(t)
		os.Symlink(target, filepath.Join(root, "link"))
		out, _ := exec.Command("git", "-C", root, "add", "-A").CombinedOutput()
		_ = out
		exec.Command("git", "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-qm", "x").Run()
		head, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
		if _, err := Materialize(context.Background(), root, Trim(string(head)), t.TempDir()); err == nil {
			t.Fatalf("%s: escaping symlink copied", name)
		}
	}
	root := fpRepo(t)
	os.Symlink("sub/b", filepath.Join(root, "ok"))
	exec.Command("git", "-C", root, "add", "-A").Run()
	exec.Command("git", "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-qm", "x").Run()
	head, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if _, err := Materialize(context.Background(), root, Trim(string(head)), t.TempDir()); err != nil {
		t.Fatalf("internal symlink refused: %v", err)
	}
}

// Staged reports whether the index differs from HEAD, from the same read as
// the digest.
func TestSnapshotReportsStaged(t *testing.T) {
	root := fpRepo(t)
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("add", "-A")
	git("commit", "-qm", "base")
	head, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	base := Trim(string(head))
	os.WriteFile(filepath.Join(root, "a"), []byte("edited\n"), 0o600)
	st, err := Snapshot(context.Background(), root, base, true)
	if err != nil || st.Staged {
		t.Fatalf("unstaged edit reported staged: %v %+v", err, st)
	}
	git("add", "a")
	st, err = Snapshot(context.Background(), root, base, true)
	if err != nil || !st.Staged {
		t.Fatalf("staged edit not reported: %v %+v", err, st)
	}
}

// D5a r18 D5AR18P-F01: the staged flag is a pure function of one index read
// and one tree read, so it cannot disagree with the digest taken from that
// same index read.
func TestIndexDiffersFromTree(t *testing.T) {
	tree := "100644 blob aaa\tf\x00100755 blob bbb\tg\x00"
	cases := map[string]struct {
		index string
		want  bool
	}{
		"same":      {"100644 aaa 0\tf\x00100755 bbb 0\tg\x00", false},
		"object":    {"100644 ccc 0\tf\x00100755 bbb 0\tg\x00", true},
		"mode":      {"100755 aaa 0\tf\x00100755 bbb 0\tg\x00", true},
		"added":     {"100644 aaa 0\tf\x00100755 bbb 0\tg\x00100644 ddd 0\th\x00", true},
		"removed":   {"100644 aaa 0\tf\x00", true},
		"conflict":  {"100644 aaa 1\tf\x00100755 bbb 0\tg\x00", true},
		"emptyboth": {"", false},
	}
	for name, c := range cases {
		tr := tree
		if name == "emptyboth" {
			tr = ""
		}
		if got := indexDiffersFromTree(c.index, tr); got != c.want {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
	}
}
