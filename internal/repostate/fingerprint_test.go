//go:build linux

package repostate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func fpRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "t"}, {"config", "user.email", "t@example.invalid"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	os.Mkdir(filepath.Join(root, "sub"), 0o700)
	os.WriteFile(filepath.Join(root, "a"), []byte("two\n"), 0o600)
	os.WriteFile(filepath.Join(root, "sub", "b"), []byte("x\n"), 0o600)
	return root
}

// D5a r7 AUD-D5A-01: a swap and restore leaves identical bytes but a new
// status-change time, so the fingerprint differs; touching nothing leaves
// it equal.
func TestFingerprintDetectsSwapAndRestore(t *testing.T) {
	root := fpRepo(t)
	ctx := context.Background()
	before, err := Fingerprint(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := Fingerprint(ctx, root); again != before {
		t.Fatal("fingerprint is not stable")
	}
	for name, swap := range map[string]func(){
		"rewrite same bytes": func() { os.WriteFile(filepath.Join(root, "a"), []byte("two\n"), 0o600) },
		"replace file": func() {
			os.Rename(filepath.Join(root, "a"), filepath.Join(root, "a.orig"))
			os.WriteFile(filepath.Join(root, "a"), []byte("SAFE\n"), 0o600)
			os.Remove(filepath.Join(root, "a"))
			os.Rename(filepath.Join(root, "a.orig"), filepath.Join(root, "a"))
		},
		"nested file": func() {
			p := filepath.Join(root, "sub", "b")
			os.WriteFile(p, []byte("SAFE\n"), 0o600)
			os.WriteFile(p, []byte("x\n"), 0o600)
		},
		"new root file created and removed": func() {
			os.WriteFile(filepath.Join(root, "transient"), []byte("TRANSIENT\n"), 0o600)
			os.Remove(filepath.Join(root, "transient"))
		},
		"chmod round trip": func() { os.Chmod(filepath.Join(root, "a"), 0o700); os.Chmod(filepath.Join(root, "a"), 0o600) },
	} {
		base, _ := Fingerprint(ctx, root)
		time.Sleep(20 * time.Millisecond)
		swap()
		after, err := Fingerprint(ctx, root)
		if err != nil || after == base {
			t.Errorf("%s: fingerprint unchanged (%v)", name, err)
		}
	}
}

// Every directory is covered, whatever any ignore rule says: a path a
// check could read under .gitignore, .git/info/exclude or a global excludes
// file is inside the digest, and so is an empty directory.
func TestFingerprintCoversEverythingButGitDir(t *testing.T) {
	ctx := context.Background()
	for name, ignore := range map[string]func(root string){
		".gitignore": func(root string) {
			os.WriteFile(filepath.Join(root, ".gitignore"), []byte("excluded/\ncache/\n"), 0o600)
		},
		"info/exclude": func(root string) {
			os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("excluded/\ncache/\n"), 0o600)
		},
	} {
		root := fpRepo(t)
		os.Mkdir(filepath.Join(root, "excluded"), 0o700)
		os.Mkdir(filepath.Join(root, "empty"), 0o700)
		os.WriteFile(filepath.Join(root, "cache"), []byte("x"), 0o600)
		ignore(root)
		base, err := Fingerprint(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
		for what, change := range map[string]func(){
			"entry created and removed in an excluded directory": func() {
				os.WriteFile(filepath.Join(root, "excluded", "new"), []byte("T"), 0o600)
				os.Remove(filepath.Join(root, "excluded", "new"))
			},
			"entry created and removed in an empty directory": func() {
				os.WriteFile(filepath.Join(root, "empty", "t"), []byte("1"), 0o600)
				os.Remove(filepath.Join(root, "empty", "t"))
			},
			"excluded file rewritten in place": func() { os.WriteFile(filepath.Join(root, "cache"), []byte("x"), 0o600) },
		} {
			change()
			if again, _ := Fingerprint(ctx, root); again == base {
				t.Errorf("%s / %s: fingerprint unchanged", name, what)
			}
			base, _ = Fingerprint(ctx, root)
			time.Sleep(20 * time.Millisecond)
		}
	}
	// .git is inside the digest (objects, index, config, info), except the
	// host's dispatch ledger.
	root := fpRepo(t)
	os.MkdirAll(filepath.Join(root, ".git", "agent-dispatch"), 0o700)
	base, _ := Fingerprint(ctx, root)
	time.Sleep(20 * time.Millisecond)
	os.WriteFile(filepath.Join(root, ".git", "agent-dispatch", "receipt"), []byte("1"), 0o600)
	if again, _ := Fingerprint(ctx, root); again != base {
		t.Error("the dispatch ledger changed the fingerprint")
	}
	for _, rel := range []string{".git/info/exclude", ".git/config", ".git/objects/zz"} {
		base, _ = Fingerprint(ctx, root)
		time.Sleep(20 * time.Millisecond)
		os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o700)
		os.WriteFile(filepath.Join(root, rel), []byte("T"), 0o600)
		if again, _ := Fingerprint(ctx, root); again == base {
			t.Errorf("a change to %s went unseen", rel)
		}
	}
}

// D5a r20 D5AR20P-F01: a .git file (worktree or submodule pointer) is an
// ordinary entry; it must not stop the walk and hide its siblings.
func TestFingerprintCoversSiblingsOfGitFile(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /elsewhere\n"), 0o600)
	os.WriteFile(filepath.Join(root, "zfile"), []byte("one\n"), 0o600)
	ctx := context.Background()
	before, err := Fingerprint(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	orig, _ := os.ReadFile(filepath.Join(root, "zfile"))
	os.WriteFile(filepath.Join(root, "zfile"), []byte("TRANSIENT\n"), 0o600)
	os.WriteFile(filepath.Join(root, "zfile"), orig, 0o600)
	after, err := Fingerprint(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("a swap and restore beside a .git file went unseen")
	}
}

// D5a r24 F2: a symlinked root is resolved, not walked as a single link.
func TestFingerprintResolvesSymlinkedRoot(t *testing.T) {
	root := fpRepo(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	viaRoot, err := Fingerprint(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	viaLink, err := Fingerprint(ctx, link)
	if err != nil || viaLink != viaRoot {
		t.Fatalf("symlinked root fingerprint differs: %v", err)
	}
	if _, err := Fingerprint(ctx, filepath.Join(root, "a")); err == nil {
		t.Fatal("a file accepted as the root")
	}
}

// D5a r24 F1: a substituted loose object that is restored before the second
// reading leaves a new status-change time in .git, so the fingerprint differs.
func TestFingerprintSeesSwapAndRestoreInGitDir(t *testing.T) {
	root := fpRepo(t)
	ctx := context.Background()
	idx := filepath.Join(root, ".git", "config")
	base, err := Fingerprint(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	orig, _ := os.ReadFile(idx)
	os.WriteFile(idx, append(orig, []byte("# t\n")...), 0o600)
	os.WriteFile(idx, orig, 0o600)
	if again, _ := Fingerprint(ctx, root); again == base {
		t.Fatal("a swap and restore inside .git went unseen")
	}
}
