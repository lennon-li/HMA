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

// resolveExisting returns the absolute form of path with every symlink along
// it followed, walking down one component at a time.
//
// filepath.EvalSymlinks is not enough here: it fails outright on a path whose
// target does not exist yet, which is exactly the case that matters -- a store
// directory before its first write, reached through a link. Walking the
// components lets a dangling link still be followed to where it points.
//
// The walk is repeated to a bounded fixpoint because following one link can
// produce a path whose own components are links again -- on macOS a link
// target written through /var names the same directory as /private/var, and
// stopping after one pass would leave that unexpanded, letting a store that
// reaches into the repository compare as outside it.
func resolveExisting(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved := abs
	for i := 0; i < 40; i++ {
		next, err := resolveOnce(resolved)
		if err != nil {
			return "", err
		}
		if next == resolved {
			return next, nil
		}
		resolved = next
	}
	return resolved, nil
}

func resolveOnce(abs string) (string, error) {
	current := string(filepath.Separator)
	for _, part := range strings.Split(abs, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		// A bounded walk, so a symlink cycle cannot spin here.
		for depth := 0; depth < 40; depth++ {
			info, err := os.Lstat(current)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				break
			}
			target, err := os.Readlink(current)
			if err != nil {
				break
			}
			if filepath.IsAbs(target) {
				current = filepath.Clean(target)
			} else {
				current = filepath.Join(filepath.Dir(current), target)
			}
		}
	}
	return current, nil
}

// StoreOutsideRepository reports an error when storeDir resolves to a path
// inside root.
//
// The store must not be part of the repository whose state is being verified,
// or the act of recording would change what a head, diff, or worktree digest
// describes. Symlinks are resolved first: a store that reaches into the
// repository through a link is inside it just as much as a subdirectory is,
// and comparing unresolved paths would miss that.
func StoreOutsideRepository(root, storeDir string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("repository root is required")
	}
	if strings.TrimSpace(storeDir) == "" {
		return errors.New("store directory is required")
	}
	r, err := resolveExisting(root)
	if err != nil {
		return err
	}
	s, err := resolveExisting(storeDir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(r, s)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("store must be outside target repository")
	}
	return nil
}
