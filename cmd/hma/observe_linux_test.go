//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// openRegular is the guard against a file swapped between the Lstat and the
// open: a FIFO, or a different file, is refused instead of read.
func TestOpenRegularRefusesSwappedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	os.WriteFile(path, []byte("one\n"), 0o600)
	want, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := openRegular(path, want)
	if err != nil {
		t.Fatalf("the unchanged file was refused: %v", err)
	}
	f.Close()
	// Swap for a FIFO: opening must neither block nor succeed.
	os.Remove(path)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		g, e := openRegular(path, want)
		if g != nil {
			g.Close()
		}
		done <- e
	}()
	select {
	case e := <-done:
		if e == nil {
			t.Error("a FIFO swapped in after the Lstat was accepted")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("openRegular blocked on a FIFO")
	}
	// Swap for a different regular file under the same name: the descriptor is
	// a regular file but not the one the Lstat saw, so it is refused.
	// The original stays on disk (moved aside) so its inode cannot be reused
	// by the replacement, which would make the two indistinguishable.
	os.Remove(path)
	if err := os.WriteFile(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".orig"); err != nil {
		t.Fatal(err)
	}
	repl := filepath.Join(dir, "repl")
	os.WriteFile(repl, []byte("three\n"), 0o600)
	want = orig
	if err := os.Rename(repl, path); err != nil {
		t.Fatal(err)
	}
	if g, e := openRegular(path, want); e == nil {
		g.Close()
		t.Error("a different regular file swapped in after the Lstat was accepted")
	}
	// Swap for a symlink to another file: refused (O_NOFOLLOW or the file check).
	os.Remove(path)
	other := filepath.Join(dir, "other")
	os.WriteFile(other, []byte("two\n"), 0o600)
	if err := os.Symlink(other, path); err != nil {
		t.Fatal(err)
	}
	if g, e := openRegular(path, want); e == nil {
		g.Close()
		t.Error("a symlink swapped in after the Lstat was accepted")
	}
}

// Audit d5c13l F01: a parent directory swapped for a symlink after it was
// checked cannot redirect the lookup: every step starts from the descriptor of
// the directory opened by the step before, so the original directory's file is
// read, never the outside file the link points to.
func TestWorktreeParentSwapCannotRedirectLookup(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "sub"), 0o700)
	os.WriteFile(filepath.Join(root, "sub", "f"), []byte("inside\n"), 0o600)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "f"), []byte("OUTSIDE\n"), 0o600)
	swapped := false
	worktreeWalkHook = func(rel string) {
		// The writer's move: the checked parent becomes a link to the outside.
		if err := os.Rename(filepath.Join(root, "sub"), filepath.Join(root, "sub.orig")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "sub")); err != nil {
			t.Error(err)
			return
		}
		swapped = true
	}
	defer func() { worktreeWalkHook = nil }()
	data, _, exists, _, _, err := worktreeBytes(context.Background(), root, "sub/f")
	if !swapped {
		t.Fatal("the hook did not run: the test exercised nothing")
	}
	if err != nil {
		t.Fatalf("worktreeBytes: %v", err)
	}
	if !exists || string(data) == "OUTSIDE\n" {
		t.Fatalf("lookup was redirected: exists %v, read %q", exists, data)
	}
	if string(data) != "inside\n" {
		t.Fatalf("read %q, want the original directory's file", data)
	}
}

// A path whose parent is already a symlink is absent, not followed (the
// snapshot's rule), and a missing /proc anchor is an error, never "absent".
func TestWorktreeSymlinkedParentIsAbsent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "f"), []byte("OUTSIDE\n"), 0o600)
	os.Symlink(outside, filepath.Join(root, "sub"))
	data, _, exists, _, _, err := worktreeBytes(context.Background(), root, "sub/f")
	if err != nil || exists || data != nil {
		t.Fatalf("symlinked parent = %q %v %v; want absent", data, exists, err)
	}
}
