//go:build linux

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"syscall"
)

// observeOpenFlags: do not follow a final symlink and do not block on a FIFO.
const observeOpenFlags = syscall.O_RDONLY | syscall.O_NOFOLLOW | syscall.O_NONBLOCK | syscall.O_CLOEXEC

// worktreeWalkHook, when set by a test, runs after every parent directory of
// the path is open and before the last component is examined: the moment a
// writer would swap a parent.
var worktreeWalkHook func(rel string)

// errDirtyObservationUnsupported is nil on Linux: the hardened per-entry
// reader below is available, so dirty observation is supported and
// observeTargetLimit's platform check always passes.
var errDirtyObservationUnsupported error

// anchored names name inside the already-open directory dir through
// /proc/self/fd, so the lookup starts at that directory's inode whatever
// happens to its path afterwards.
func anchored(dir *os.File, name string) string {
	p := fmt.Sprintf("/proc/self/fd/%d/%s", dir.Fd(), name)
	runtime.KeepAlive(dir)
	return p
}

// openWorktreeEntry looks up rel below root one component at a time through
// retained directory descriptors, never following a symlink in any component.
// A parent that is missing, a symlink or not a directory makes the entry
// absent, as in the snapshot. Because every step starts from the descriptor of
// the directory opened by the step before, a writer that swaps a parent for a
// link after it was checked cannot redirect the lookup: the retained descriptor
// still names the original directory.
func openWorktreeEntry(root, rel string) (worktreeEntry, error) {
	parts := strings.Split(rel, "/")
	for _, c := range parts {
		if c == "" || c == "." || c == ".." {
			return worktreeEntry{}, fmt.Errorf("target path %q is not observable", rel)
		}
	}
	dir, err := os.OpenFile(root, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return worktreeEntry{}, err
	}
	// Without procfs the anchored names do not resolve, which would read as
	// "absent": refuse instead.
	if _, err := os.Stat(fmt.Sprintf("/proc/self/fd/%d", dir.Fd())); err != nil {
		dir.Close()
		return worktreeEntry{}, fmt.Errorf("dirty target observation needs /proc: %w", err)
	}
	for _, c := range parts[:len(parts)-1] {
		next, e := os.OpenFile(anchored(dir, c), os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		dir.Close()
		if e != nil {
			if errors.Is(e, fs.ErrNotExist) || errors.Is(e, syscall.ENOTDIR) || errors.Is(e, syscall.ELOOP) {
				return worktreeEntry{kind: entryAbsent}, nil
			}
			return worktreeEntry{}, e
		}
		dir = next
	}
	defer dir.Close()
	if worktreeWalkHook != nil {
		worktreeWalkHook(rel)
	}
	leaf := anchored(dir, parts[len(parts)-1])
	fi, err := os.Lstat(leaf)
	if errors.Is(err, fs.ErrNotExist) {
		return worktreeEntry{kind: entryAbsent}, nil
	}
	if err != nil {
		return worktreeEntry{}, err
	}
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		link, e := os.Readlink(leaf)
		if e != nil {
			return worktreeEntry{}, e
		}
		return worktreeEntry{kind: entrySymlink, link: link}, nil
	case fi.IsDir():
		return worktreeEntry{kind: entryDir}, nil
	case !fi.Mode().IsRegular():
		return worktreeEntry{kind: entrySpecial}, nil
	}
	f, err := openRegular(leaf, fi)
	if err != nil {
		return worktreeEntry{}, err
	}
	return worktreeEntry{kind: entryRegular, file: f, exec: fi.Mode().Perm()&0100 != 0}, nil
}

// openRegular opens path read-only without following a final symlink and
// without blocking on a FIFO, then requires the opened descriptor to be a
// regular file and the same file the earlier Lstat saw (device and inode).
func openRegular(path string, want os.FileInfo) (*os.File, error) {
	f, err := os.OpenFile(path, observeOpenFlags, 0)
	if err != nil {
		return nil, err
	}
	got, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !got.Mode().IsRegular() || !os.SameFile(want, got) {
		f.Close()
		return nil, fmt.Errorf("target path %q changed while it was observed", path)
	}
	return f, nil
}
