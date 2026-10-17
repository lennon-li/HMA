package repostate

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// changedPaths returns, in sorted order, every path whose state on disk or in
// the index differs from HEAD, plus every untracked, non-ignored path. It does
// not ask git status: status compares through repository-controlled settings
// (attributes, filters, eol conversion, submodule ignore rules, stat caches),
// any of which can make changed bytes read as unchanged. Instead each HEAD
// entry's raw worktree bytes are hashed as a Git blob without any conversion
// and compared with the committed object id, and file type and the executable
// bit are compared with the committed mode.
//
// Repositories whose worktree HMA cannot describe this way are refused:
// submodules (their content lives in another repository) and sparse checkout
// (paths absent by design).
func changedPaths(ctx context.Context, root, head string) ([]string, error) {
	// Tree paths and untracked listings are relative to the top level; a
	// subdirectory root would compare the wrong files.
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
		// "<mode> SP <type> SP <object> TAB <path>"
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
		same, err := worktreeMatches(root, path, mode, oid, newHash)
		if err != nil {
			return nil, err
		}
		if !same {
			changed[path] = true
		}
	}
	// Index entries that differ from HEAD (staged changes, staged additions,
	// staged deletions). Comparing index and tree compares object ids and
	// modes only; no worktree conversion is involved.
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
	// Untracked files are in scope unless a committed .gitignore excludes
	// them. core.excludesFile and .git/info/exclude are host-local and could
	// hide any new file, so they are not consulted; an untracked .gitignore
	// is itself a change, and is found even when it ignores itself.
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
	// Ignore rules apply only as committed: if any .gitignore differs from
	// HEAD (changed, added, or deleted), it could hide files, so every
	// untracked path is listed without ignore rules.
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

// worktreeMatches reports whether the worktree entry at path has exactly the
// committed mode and object id.
func worktreeMatches(root, path, mode, oid string, newHash func() hash.Hash) (bool, error) {
	full := filepath.Join(root, filepath.FromSlash(path))
	fi, err := lstatInTree(root, path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var data []byte
	switch mode {
	case "120000":
		if fi.Mode()&os.ModeSymlink == 0 {
			return false, nil
		}
		target, err := os.Readlink(full)
		if err != nil {
			return false, err
		}
		data = []byte(target)
	case "100644", "100755":
		if !fi.Mode().IsRegular() || (fi.Mode().Perm()&0o100 != 0) != (mode == "100755") {
			return false, nil
		}
		if data, err = os.ReadFile(full); err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("unsupported tree entry mode %s for %s", mode, path)
	}
	return blobID(newHash, data) == oid, nil
}

// lstatInTree is os.Lstat of path below root, except that a path whose parent
// components are not all real directories is reported as not existing. Git
// does the same ("beyond a symbolic link"): a tracked dir/file reached
// through a symlink or under a regular file is not in the worktree, even if
// following the link would find identical bytes.
func lstatInTree(root, path string) (os.FileInfo, error) {
	parts := strings.Split(path, "/")
	dir := root
	for _, part := range parts[:len(parts)-1] {
		dir = filepath.Join(dir, part)
		fi, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, os.ErrNotExist
		}
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, os.ErrNotExist
		}
	}
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
	if errors.Is(err, syscall.ENOTDIR) {
		return nil, os.ErrNotExist
	}
	return fi, err
}

// blobID is Git's object id for a blob with exactly these bytes. SHA-1 here
// is Git's own object naming for sha1 repositories, used only to compare with
// the committed id, not as an HMA security digest.
func blobID(newHash func() hash.Hash, data []byte) string {
	h := newHash()
	h.Write([]byte("blob " + strconv.Itoa(len(data)) + "\x00"))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func blobHasher(format string) (func() hash.Hash, error) {
	switch format {
	case "sha1":
		return sha1.New, nil
	case "sha256":
		return sha256.New, nil
	}
	return nil, fmt.Errorf("unsupported object format %q", format)
}

// requireTopLevel fails unless root is the top level of its Git work tree.
func requireTopLevel(ctx context.Context, root string) error {
	top, err := Output(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	want, err := filepath.EvalSymlinks(Trim(top))
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	got, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	if want != got {
		return fmt.Errorf("repository root %s is not the work tree top level %s", root, Trim(top))
	}
	return nil
}
