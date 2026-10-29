package repostate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// BlobWindow streams a blob, returning its complete SHA-256 and only the
// first limit bytes for bounded line counting.
func BlobWindow(ctx context.Context, root, oid string, limit int64) ([]byte, string, bool, error) {
	args := append(append([]string{"--no-replace-objects"}, observationConfig...), "cat-file", "blob", oid)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	cmd.Env = gitEnv(os.Environ())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", false, err
	}
	if err = cmd.Start(); err != nil {
		return nil, "", false, err
	}
	h := sha256.New()
	window := &boundedWindow{limit: limit}
	if _, err = io.Copy(io.MultiWriter(h, window), stdout); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, "", false, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, "", false, err
	}
	return window.data, hex.EncodeToString(h.Sum(nil)), window.overflow, nil
}

type boundedWindow struct {
	data     []byte
	limit    int64
	overflow bool
}

func (w *boundedWindow) Write(p []byte) (int, error) {
	if int64(len(w.data)) < w.limit {
		n := int(w.limit - int64(len(w.data)))
		if n > len(p) {
			n = len(p)
		}
		w.data = append(w.data, p[:n]...)
		if n < len(p) {
			w.overflow = true
		}
	} else if len(p) > 0 {
		w.overflow = true
	}
	return len(p), nil
}

// ChangedPaths is the set of paths whose state differs from head under the
// snapshot's own rules (tracked changes, staged changes, untracked paths not
// excluded by a committed .gitignore). The snapshot digests exactly these
// paths, so anything that describes the same target must use this set.
func ChangedPaths(ctx context.Context, root, head string) ([]string, error) {
	return changedPaths(ctx, root, head)
}

// EntryKind is what a WorktreeReader found at a path.
type EntryKind int

const (
	EntryAbsent EntryKind = iota
	EntryDir
	EntrySymlink
	EntryRegular
	EntrySpecial
)

// WorktreeEntry is one worktree path read once. Data is the symlink target or
// the complete file content; Exec is the owner execute bit of a regular file.
type WorktreeEntry struct {
	Kind EntryKind
	Exec bool
	Data []byte
}

// Record is a compact identity of what was read: the kind, the Git mode for a
// file or link, and the SHA-256 of the bytes. Two reads that return the same
// Record described the same bytes.
func (e WorktreeEntry) Record() string {
	switch e.Kind {
	case EntrySymlink:
		return EntryRecord("120000", sha256Hex(e.Data))
	case EntryRegular:
		mode := "100644"
		if e.Exec {
			mode = "100755"
		}
		return EntryRecord(mode, sha256Hex(e.Data))
	case EntryDir:
		return "dir"
	case EntrySpecial:
		return "special"
	}
	return "absent"
}

// EntryRecord formats a Record for a file or link of this Git mode and content
// SHA-256.
func EntryRecord(mode, sha string) string { return mode + ":" + sha }

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// WorktreeReader reads one worktree path, with the same absence rule as the
// snapshot: a path below a missing, non-directory or symlinked parent is
// absent. A reader that anchors every lookup to retained directory descriptors
// keeps a concurrent writer from steering the read outside the repository.
type WorktreeReader interface {
	ReadEntry(ctx context.Context, rel string) (WorktreeEntry, error)
}

// PathCapture is the changed-path set and the identity of every tracked path
// the discovery read, all taken from one pass of a WorktreeReader.
type PathCapture struct {
	Changed []string
	Entries map[string]string
}

// CapturePaths is ChangedPaths with every worktree read going through r.
func CapturePaths(ctx context.Context, root, head string, r WorktreeReader) (PathCapture, error) {
	entries := map[string]string{}
	changed, err := changedPathsWith(ctx, root, head, r, entries)
	if err != nil {
		return PathCapture{}, err
	}
	return PathCapture{Changed: changed, Entries: entries}, nil
}

// changedPathsWith mirrors changedPaths (raw.go) with worktree reads supplied
// by r. It records each tracked path's read in entries.
func changedPathsWith(ctx context.Context, root, head string, r WorktreeReader, entries map[string]string) ([]string, error) {
	if err := requireTopLevel(ctx, root); err != nil {
		return nil, err
	}
	if sparse, _ := Output(ctx, root, "config", "--bool", "core.sparseCheckout"); Trim(sparse) == "true" {
		return nil, errors.New("sparse checkout is not supported: the worktree omits tracked paths by design")
	}
	format, err := Output(ctx, root, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, err
	}
	newHash, err := blobHasher(Trim(format))
	if err != nil {
		return nil, err
	}
	tree, err := Output(ctx, root, "ls-tree", "-r", "-z", "--full-tree", head)
	if err != nil {
		return nil, err
	}
	changed := map[string]bool{}
	tracked := map[string]bool{}
	for _, entry := range strings.Split(tree, "\x00") {
		if entry == "" {
			continue
		}
		meta, path, ok := strings.Cut(entry, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 {
			return nil, fmt.Errorf("unexpected ls-tree entry %q", entry)
		}
		mode, oid := f[0], f[2]
		if mode == "160000" {
			return nil, fmt.Errorf("submodule %s is not supported: its content is in another repository", path)
		}
		tracked[path] = true
		same, err := worktreeMatchesWith(ctx, r, path, mode, oid, newHash, entries)
		if err != nil {
			return nil, err
		}
		if !same {
			changed[path] = true
		}
	}
	staged, err := Output(ctx, root, "diff-index", "--cached", "--no-renames", "--name-only", "-z", head, "--")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(staged, "\x00") {
		if p != "" {
			changed[p] = true
		}
	}
	index, err := Output(ctx, root, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	for _, entry := range strings.Split(index, "\x00") {
		if strings.HasPrefix(entry, "160000 ") {
			return nil, fmt.Errorf("submodule entry in index is not supported: %q", entry)
		}
	}
	others, err := Output(ctx, root, "ls-files", "--others", "--exclude-per-directory=.gitignore", "-z")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(others, "\x00") {
		if p != "" && !tracked[p] {
			changed[p] = true
		}
	}
	all, err := Output(ctx, root, "ls-files", "--others", "-z")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(all, "\x00") {
		if p != "" && !tracked[p] && (p == ".gitignore" || strings.HasSuffix(p, "/.gitignore")) {
			changed[p] = true
		}
	}
	for p := range changed {
		if p == ".gitignore" || strings.HasSuffix(p, "/.gitignore") {
			for _, q := range strings.Split(all, "\x00") {
				if q != "" && !tracked[q] {
					changed[q] = true
				}
			}
			break
		}
	}
	paths := make([]string, 0, len(changed))
	for p := range changed {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

// worktreeMatchesWith is worktreeMatches (raw.go) over one read from r.
func worktreeMatchesWith(ctx context.Context, r WorktreeReader, path, mode, oid string, newHash func() hash.Hash, entries map[string]string) (bool, error) {
	e, err := r.ReadEntry(ctx, path)
	if err != nil {
		return false, err
	}
	entries[path] = e.Record()
	if e.Kind == EntryAbsent {
		return false, nil
	}
	switch mode {
	case "120000":
		if e.Kind != EntrySymlink {
			return false, nil
		}
	case "100644", "100755":
		if e.Kind != EntryRegular || e.Exec != (mode == "100755") {
			return false, nil
		}
	default:
		return false, fmt.Errorf("unsupported tree entry mode %s for %s", mode, path)
	}
	return blobID(newHash, e.Data) == oid, nil
}
