//go:build linux

package repostate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

// Fingerprint digests the file metadata of the whole worktree and of .git
// (except the host's dispatch ledger, .git/agent-dispatch):
// every file and symlink (type, size, inode, device, modification and
// status-change time) and every directory, the root and empty ones included
// (mode, inode, device, status-change time). No ignore rule narrows it: not
// .gitignore, not .git/info/exclude, not a global excludes file, so a host-
// or user-local rule cannot hide a path a check could read. A swap or
// restore of a file, or the creation and removal of an entry anywhere,
// leaves the same bytes behind but a new status-change time that userland
// cannot set back, so the digest differs. It detects such a change; it does
// not prevent one. Credit runs checks on a private copy (Materialize), so
// the fingerprint covers the copy window and external writers to the live
// worktree.
func Fingerprint(ctx context.Context, root string) (string, error) {
	// A symlinked root would make the walk see only the link itself.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(resolved); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("fingerprint root %s is not a directory", root)
	}
	root = resolved
	var rows []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		// The host's own dispatch ledger lives in .git but is read by no
		// command a check or snapshot runs; everything else under .git
		// (objects, index, refs, config, info) is the reviewed state's source
		// and is covered.
		if rel == ".git/agent-dispatch" && d.IsDir() {
			return filepath.SkipDir
		}
		fi, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("no file metadata for %s", rel)
		}
		if d.IsDir() {
			rows = append(rows, fmt.Sprintf("dir\x00%s\x00%o\x00%d\x00%d\x00%d.%d", rel, st.Mode, st.Ino, st.Dev, st.Ctim.Sec, st.Ctim.Nsec))
			return nil
		}
		rows = append(rows, fmt.Sprintf("file\x00%s\x00%o\x00%d\x00%d\x00%d\x00%d.%d\x00%d.%d", rel, st.Mode, st.Size, st.Ino, st.Dev,
			st.Mtim.Sec, st.Mtim.Nsec, st.Ctim.Sec, st.Ctim.Nsec))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(rows)
	h := sha256.New()
	h.Write([]byte("hma-fingerprint-v4\x00"))
	for _, r := range rows {
		writeFramed(h, []byte(r))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
