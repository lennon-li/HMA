// Package repostate captures the exact Git repository state that a record is
// bound to: the head revision, the digest of the diff from an approved base,
// and -- only for an approved dirty-worktree capture -- the digest of the
// uncommitted content a command actually saw.
//
// It is standard-library-only. It shells out to git for reads only, writes
// nothing to the repository or its object store, and never advances a stage.
package repostate

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// State is the exact repository state evidence or an approval is bound to.
// Worktree is empty for a clean worktree, which is the revision-reproducible
// case; it is set only for an approved dirty-worktree capture, where the
// committed revision alone no longer describes what was seen.
type State struct{ Head, Diff, Worktree string }

// Trim removes the trailing newline git appends to single-value output.
func Trim(s string) string { return strings.TrimSpace(s) }

// Output runs a read-only git command in dir and returns its stdout.
func Output(ctx context.Context, dir string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	b, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(b), nil
}

// statusPaths parses `git status --porcelain -z` into the set of paths git
// reports as changed. The NUL-delimited form is used because the default
// output quotes and escapes unusual paths, which would make the digest depend
// on how a path renders rather than on what it is. A rename or copy entry is
// followed by its source path as a separate record; both paths are kept,
// because both are part of what changed.
func statusPaths(status string) []string {
	fields := strings.Split(status, "\x00")
	var paths []string
	for i := 0; i < len(fields); i++ {
		entry := fields[i]
		if len(entry) < 4 {
			continue
		}
		// Porcelain v1 entry: two status codes, a space, then the path.
		code, path := entry[:2], entry[3:]
		paths = append(paths, path)
		if strings.ContainsAny(code, "RC") && i+1 < len(fields) {
			i++
			if fields[i] != "" {
				paths = append(paths, fields[i])
			}
		}
	}
	sort.Strings(paths)
	return paths
}

// worktreeContentDigest hashes the exact content of a dirty worktree: the head
// revision, then every changed path in sorted order with the digest of its
// current bytes, or an explicit absence marker for a deleted path. Entries are
// domain-separated and length-framed so no two different worktrees can produce
// the same digest by concatenation.
//
// It reads only the paths git names and writes nothing, so taking the digest
// never mutates the repository or its object store.
func worktreeContentDigest(root, head, status string) (string, error) {
	h := sha256.New()
	h.Write([]byte("hma-worktree-v1\x00" + head + "\x00"))
	for _, path := range statusPaths(status) {
		h.Write([]byte("path\x00"))
		writeFramed(h, []byte(path))
		full := filepath.Join(root, filepath.FromSlash(path))
		content, err := os.ReadFile(full)
		switch {
		case err == nil:
			h.Write([]byte("content\x00"))
			writeFramed(h, content)
		case errors.Is(err, os.ErrNotExist):
			h.Write([]byte("absent\x00"))
		default:
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeFramed(h hash.Hash, data []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	h.Write(length[:])
	h.Write(data)
}

// Snapshot captures the repository state. A dirty worktree is refused unless
// the host has explicitly approved this dirty-worktree use, per architecture
// section 14: such evidence is admissible only when the record carries the
// exact worktree-content digest, and it is never revision-reproducible.
func Snapshot(ctx context.Context, root, baseRevision string, dirtyApproved bool) (State, error) {
	head, err := Output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return State{}, err
	}
	status, err := Output(ctx, root, "status", "--porcelain", "-z", "-uall")
	if err != nil {
		return State{}, err
	}
	dirty := strings.Trim(status, "\x00") != ""
	if dirty && !dirtyApproved {
		return State{}, errors.New("dirty worktree policy violation")
	}
	diff, err := Output(ctx, root, "diff", "--binary", baseRevision+".."+Trim(head), "--")
	if err != nil {
		return State{}, err
	}
	sum := sha256.Sum256([]byte(diff))
	s := State{Head: Trim(head), Diff: hex.EncodeToString(sum[:])}
	if dirty {
		s.Worktree, err = worktreeContentDigest(root, s.Head, status)
		if err != nil {
			return State{}, err
		}
	}
	return s, nil
}

// VerifyBase reports whether baseRevision names an existing commit in root and
// is written as that commit's full object id. An abbreviated or symbolic name
// is refused: the approval binds an exact base-revision digest, and a name
// that can resolve to different commits later is not that.
func VerifyBase(ctx context.Context, root, baseRevision string) error {
	resolved, err := Output(ctx, root, "rev-parse", "--verify", baseRevision+"^{commit}")
	if err != nil {
		return err
	}
	if Trim(resolved) != baseRevision {
		return fmt.Errorf("base revision %q is not an exact commit id", baseRevision)
	}
	return nil
}
