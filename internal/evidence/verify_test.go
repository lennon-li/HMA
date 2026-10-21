package evidence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func verifyRepo(t *testing.T) (root, head, blob string) {
	t.Helper()
	root = gitRepoWithConfig(t, "")
	os.WriteFile(filepath.Join(root, "f"), []byte("reviewed\n"), 0o600)
	gitIn(t, root, "add", "f")
	gitIn(t, root, "commit", "-qm", "base")
	return root, gitIn(t, root, "rev-parse", "HEAD"), gitIn(t, root, "rev-parse", "HEAD:f")
}

// D5a r25 N1: an object forged in place before the first reading is refused,
// not taken as the reviewed history.
func TestVerifyRepositoryObjectsRefusesForgedObject(t *testing.T) {
	root, head, blob := verifyRepo(t)
	if err := VerifyRepositoryObjects(context.Background(), root, head); err != nil {
		t.Fatalf("control: %v", err)
	}
	sub := exec.Command("git", "-C", root, "hash-object", "-w", "--stdin")
	sub.Stdin = strings.NewReader("FORGED\n")
	out, err := sub.Output()
	if err != nil {
		t.Fatal(err)
	}
	other := strings.TrimSpace(string(out))
	data, err := os.ReadFile(filepath.Join(root, ".git", "objects", other[:2], other[2:]))
	if err != nil {
		t.Fatal(err)
	}
	loose := filepath.Join(root, ".git", "objects", blob[:2], blob[2:])
	os.Chmod(loose, 0o600)
	os.WriteFile(loose, data, 0o400)
	if err := VerifyRepositoryObjects(context.Background(), root, head); err == nil {
		t.Fatal("a forged object was taken as reviewed history")
	}
}

// D5a r25 N2/N4: alternate object stores are refused (their content is outside
// the fingerprinted .git), including a FIFO that would block a read.
func TestVerifyRepositoryObjectsRefusesAlternates(t *testing.T) {
	root, head, _ := verifyRepo(t)
	alt := filepath.Join(root, ".git", "objects", "info", "alternates")
	os.MkdirAll(filepath.Dir(alt), 0o700)
	if err := syscall.Mkfifo(alt, 0o600); err != nil {
		t.Skip("mkfifo")
	}
	done := make(chan error, 1)
	go func() { done <- VerifyRepositoryObjects(context.Background(), root, head) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("alternates accepted")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO alternates file blocked the check")
	}
}

// D5a r25 N3: a .git that is not a real directory (pointer file, symlink) is
// refused: the fingerprint cannot follow it.
func TestVerifyRepositoryObjectsRefusesGitPointer(t *testing.T) {
	root, head, _ := verifyRepo(t)
	real := filepath.Join(t.TempDir(), "gitdir")
	if err := os.Rename(filepath.Join(root, ".git"), real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	if err := VerifyRepositoryObjects(context.Background(), root, head); err == nil {
		t.Fatal("a symlinked .git was accepted")
	}
}

// D5a r27 R1/R2/R3: git redirection files, links and special files under
// .git are refused; a plain repository, with the host ledger present, is not.
func TestRequireSelfContainedRepository(t *testing.T) {
	plain := func() string { r, _, _ := verifyRepo(t); return r }
	root := plain()
	os.MkdirAll(filepath.Join(root, ".git", "agent-dispatch", "runs"), 0o700)
	os.Symlink("/nowhere", filepath.Join(root, ".git", "agent-dispatch", "link")) // ledger exempt
	if err := RequireSelfContainedRepository(root); err != nil {
		t.Fatalf("plain repository refused: %v", err)
	}
	for name, mutate := range map[string]func(r string){
		"commondir": func(r string) { os.WriteFile(filepath.Join(r, ".git", "commondir"), []byte("/elsewhere\n"), 0o600) },
		"gitdir":    func(r string) { os.WriteFile(filepath.Join(r, ".git", "gitdir"), []byte("/elsewhere\n"), 0o600) },
		"symlinked objects": func(r string) {
			real := filepath.Join(t.TempDir(), "objects")
			os.Rename(filepath.Join(r, ".git", "objects"), real)
			os.Symlink(real, filepath.Join(r, ".git", "objects"))
		},
		"symlinked ref": func(r string) { os.Symlink("/elsewhere", filepath.Join(r, ".git", "refs", "heads", "x")) },
		"fifo config": func(r string) {
			os.Remove(filepath.Join(r, ".git", "config"))
			syscall.Mkfifo(filepath.Join(r, ".git", "config"), 0o600)
		},
	} {
		r := plain()
		mutate(r)
		if err := RequireSelfContainedRepository(r); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// D5a r28: a local configuration key outside the allow list (one that changes
// how git reads objects, or an include that pulls in more settings) is
// refused for credit; a plain repository config is not.
func TestRequireSelfContainedRepositoryRefusesNonPlainConfig(t *testing.T) {
	for name, extra := range map[string]string{
		"multipackindex": "[core]\n\tmultiPackIndex = false\n",
		"include":        "[include]\n\tpath = /elsewhere/cfg\n",
		"includeif":      "[includeIf \"gitdir:/\"]\n\tpath = /elsewhere/cfg\n",
		"commitgraph":    "[core]\n\tcommitGraph = false\n",
		"pack":           "[pack]\n\tthreads = 1\n",
	} {
		root, _, _ := verifyRepo(t)
		f, err := os.OpenFile(filepath.Join(root, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(extra)
		f.Close()
		if err := RequireSelfContainedRepository(root); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	root, _, _ := verifyRepo(t)
	gitIn(t, root, "config", "user.name", "someone")
	gitIn(t, root, "config", "remote.origin.url", "https://example.invalid/x.git")
	if err := RequireSelfContainedRepository(root); err != nil {
		t.Errorf("plain config refused: %v", err)
	}
}
