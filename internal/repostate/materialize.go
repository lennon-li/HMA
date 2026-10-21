package repostate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Materialize writes the reviewed target of root into the empty directory
// dest: every path tracked at head, and every changed or untracked
// non-ignored path (the paths Snapshot digests), taken as raw bytes with the
// committed file type and owner execute bit. Nothing else is copied: an
// untracked file that a committed .gitignore hides, a build cache, and .git
// are not part of the reviewed target, so a command run in dest can read
// only what the review saw. Paths absent from the worktree stay absent. A
// symlink that points outside the copy is refused. It returns the changed and
// untracked non-ignored paths (relative to HEAD).
func Materialize(ctx context.Context, root, head, dest string) ([]string, error) {
	changed, err := changedPaths(ctx, root, head)
	if err != nil {
		return nil, err
	}
	tree, err := Output(ctx, root, "ls-tree", "-r", "-z", "--full-tree", "--name-only", head)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var paths []string
	for _, p := range append(strings.Split(tree, "\x00"), changed...) {
		if p != "" && !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	for _, p := range paths {
		for _, part := range strings.Split(p, "/") {
			if part == ".git" || part == "" || part == "." || part == ".." {
				return nil, fmt.Errorf("target path %q is not copyable", p)
			}
		}
		fi, err := lstatInTree(root, p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		src := filepath.Join(root, filepath.FromSlash(p))
		out := filepath.Join(dest, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
			return nil, err
		}
		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(src)
			if err != nil {
				return nil, err
			}
			// A link that leaves the copy would let a check read or write the
			// live worktree: absolute targets and relative ones that climb
			// above the root are refused.
			if filepath.IsAbs(target) || strings.HasPrefix(filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(p)), target)), "../") ||
				filepath.Join(filepath.Dir(filepath.FromSlash(p)), target) == ".." {
				return nil, fmt.Errorf("symlink %q leaves the reviewed target", p)
			}
			if err := os.Symlink(target, out); err != nil {
				return nil, err
			}
		case fi.Mode().IsRegular():
			data, err := os.ReadFile(src)
			if err != nil {
				return nil, err
			}
			mode := os.FileMode(0o600)
			if fi.Mode().Perm()&0o100 != 0 {
				mode = 0o700
			}
			if err := os.WriteFile(out, data, mode); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("target path %q is neither a regular file nor a symlink", p)
		}
	}
	return changed, nil
}
