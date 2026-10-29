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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// State is the exact repository state evidence or an approval is bound to.
// Worktree is empty for a clean worktree, which is the revision-reproducible
// case; it is set only for an approved dirty-worktree capture, where the
// committed revision alone no longer describes what was seen.
// State is a repository snapshot. Staged is true when the index differs from
// HEAD; it is read in the same call as the digest it accompanies, so the
// digest (which binds the staged entries) vouches for it.
type State struct {
	Head, Diff, Worktree string
	Staged               bool
}

// Trim removes the trailing newline git appends to single-value output.
func Trim(s string) string { return strings.TrimSpace(s) }

// repositorySelectingEnv lists the variables git reports from
// `git rev-parse --local-env-vars` that choose which repository, work tree,
// index, or object store a command reads. Inherited, they would let the
// caller's environment point a read at a different repository than dir.
// The GIT_CONFIG* variables from that list are deliberately kept: hosts and CI
// use them to inject settings such as safe.directory.
var repositorySelectingEnv = []string{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_COMMON_DIR",
	"GIT_DIR",
	"GIT_GRAFT_FILE",
	"GIT_IMPLICIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_INTERNAL_SUPER_PREFIX",
	"GIT_NO_REPLACE_OBJECTS",
	"GIT_OBJECT_DIRECTORY",
	"GIT_PREFIX",
	"GIT_REPLACE_REF_BASE",
	"GIT_SHALLOW_FILE",
	"GIT_WORK_TREE",
}

// GitEnv returns environ without the repository-selecting variables. Evidence
// commands run with it so a captured command reads the checkout HMA snapshot,
// not a repository named by an inherited GIT_DIR or GIT_WORK_TREE.
//
// It also sets GIT_NO_REPLACE_OBJECTS, so a Git command the evidence runs
// reads the committed objects HMA snapshot, not replacement objects.
func GitEnv(environ []string) []string {
	// Configuration that can launch a program during a read-only command
	// is overridden: no system or global file, no fsmonitor hook, no hooks
	// path. (diff.external and textconv drivers cannot be unset this way;
	// the argv validator requires --no-ext-diff and --no-textconv.)
	env := make([]string, 0, len(environ)+8)
	for _, e := range gitEnv(environ) {
		if name, _, _ := strings.Cut(e, "="); !strings.HasPrefix(name, "GIT_CONFIG") {
			env = append(env, e)
		}
	}
	return append(env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_COUNT=5", "GIT_CONFIG_KEY_0=core.fsmonitor", "GIT_CONFIG_VALUE_0=false",
		"GIT_CONFIG_KEY_1=core.hooksPath", "GIT_CONFIG_VALUE_1=/dev/null",
		"GIT_CONFIG_KEY_2=log.showSignature", "GIT_CONFIG_VALUE_2=false",
		"GIT_CONFIG_KEY_3=diff.ignoreSubmodules", "GIT_CONFIG_VALUE_3=all",
		"GIT_CONFIG_KEY_4=status.ignoreSubmodules", "GIT_CONFIG_VALUE_4=all")
}

// SelectingVariableSet returns the name of the first repository-selecting
// variable present in environ, or "" when none is.
func SelectingVariableSet(environ []string) string {
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		for _, v := range repositorySelectingEnv {
			if name == v {
				return name
			}
		}
	}
	return ""
}

// gitEnv returns environ without the repository-selecting variables.
func gitEnv(environ []string) []string {
	env := make([]string, 0, len(environ))
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		selecting := false
		for _, v := range repositorySelectingEnv {
			if name == v {
				selecting = true
				break
			}
		}
		if !selecting {
			env = append(env, entry)
		}
	}
	// Read-only observation must not rewrite the index: the fingerprint of
	// .git would otherwise change under our own reads.
	return append(env, "GIT_OPTIONAL_LOCKS=0")
}

// observationConfig overrides every Git setting that can make status or diff
// report a changed file as unchanged, so the repository's own configuration
// cannot hide a mode, type, or content change from a snapshot. Values given
// with -c take precedence over system, global, and repository config.
var observationConfig = []string{
	"-c", "core.fileMode=true", // mode-only changes
	"-c", "core.symlinks=true", // symlink vs regular file
	"-c", "core.checkStat=default", // full stat comparison before trusting the cache
	"-c", "core.trustctime=true",
	"-c", "core.fsmonitor=false", // no external daemon decides what changed
	"-c", "core.untrackedCache=false",
	"-c", "core.autocrlf=false",
	"-c", "status.showUntrackedFiles=all",
	"-c", "core.ignoreCase=false", // a case alias is a distinct path
	"-c", "diff.external=",
	"-c", "diff.ignoreSubmodules=none",
}

// Output runs a read-only git command in dir and returns its stdout. The
// repository is chosen by dir alone, never by inherited GIT_DIR-style
// variables, and change detection is not weakened by repository config.
func Output(ctx context.Context, dir string, args ...string) (string, error) {
	// --no-replace-objects: replacement refs would let ls-tree and diff read
	// substitute objects instead of the committed ones the approval names.
	c := exec.CommandContext(ctx, "git", append(append([]string{"--no-replace-objects"}, observationConfig...), args...)...)
	c.Dir = dir
	c.Env = gitEnv(os.Environ())
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
// revision, then every changed path in sorted order with its file type,
// Git-relevant executable bit, and raw bytes (or symlink target), or an
// explicit absence marker for a deleted path. Entries are domain-separated
// and length-framed so no two different worktrees can produce the same digest
// by concatenation.
//
// It reads only the paths it is given and writes nothing, so taking the
// digest never mutates the repository or its object store.
func worktreeContentDigest(root, head string, paths []string, staged map[string]string) (string, error) {
	return digestWorktree(context.Background(), root, head, paths, staged, nil, nil)
}

// digestWorktree is worktreeContentDigest; with a reader every path is read once
// through it and its Record stored in entries, and a path the changed-path
// discovery already recorded must read the same again.
func digestWorktree(ctx context.Context, root, head string, paths []string, staged map[string]string, r WorktreeReader, entries map[string]string) (string, error) {
	h := sha256.New()
	// v5 takes its changed-path set from a raw comparison of the worktree
	// against HEAD (see changedPaths), not from git status. Earlier domains
	// (v1 ignored mode and type; v2 collapsed all execute bits; v3 and v4
	// visited only the paths git status reported, which repository config,
	// attributes, and submodule settings can hide) never equal a v5 digest,
	// so approvals and reviews bound to them fail closed and need a fresh
	// capture of the live content. v6 retired v5 digests taken before
	// core.ignoreCase was forced; v7 adds the staged index entry per path;
	// v8 retires v7 digests that read a tracked path through a symlinked or
	// non-directory parent instead of recording it absent.
	h.Write([]byte("hma-worktree-v8\x00" + head + "\x00"))
	for _, path := range paths {
		h.Write([]byte("path\x00"))
		writeFramed(h, []byte(path))
		// v7 also binds the staged index entry (mode, object, stage) for the
		// path, so restaging different content under unchanged disk bytes is
		// a different artifact.
		h.Write([]byte("index\x00"))
		writeFramed(h, []byte(staged[path]))
		if r != nil {
			e, err := r.ReadEntry(ctx, path)
			if err != nil {
				return "", err
			}
			rec := e.Record()
			if prev, ok := entries[path]; ok && prev != rec {
				return "", fmt.Errorf("worktree path %s changed while it was captured", path)
			}
			entries[path] = rec
			switch e.Kind {
			case EntryAbsent:
				h.Write([]byte("absent\x00"))
			case EntrySymlink:
				h.Write([]byte("symlink\x00"))
				writeFramed(h, e.Data)
			case EntryRegular:
				if e.Exec {
					h.Write([]byte("executable\x00"))
				} else {
					h.Write([]byte("regular\x00"))
				}
				writeFramed(h, e.Data)
			default:
				return "", fmt.Errorf("worktree path %s is neither a regular file nor a symlink", path)
			}
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(path))
		fi, err := lstatInTree(root, path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			h.Write([]byte("absent\x00"))
		case err != nil:
			return "", err
		case fi.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(full)
			if err != nil {
				return "", err
			}
			h.Write([]byte("symlink\x00"))
			writeFramed(h, []byte(target))
		case fi.Mode().IsRegular():
			content, err := os.ReadFile(full)
			if err != nil {
				return "", err
			}
			// Git records a regular file as 100755 exactly when the owner
			// execute bit is set; group and other execute bits are ignored.
			if fi.Mode().Perm()&0o100 != 0 {
				h.Write([]byte("executable\x00"))
			} else {
				h.Write([]byte("regular\x00"))
			}
			writeFramed(h, content)
		default:
			return "", fmt.Errorf("worktree path %s is neither a regular file nor a symlink", path)
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
	c, err := capture(ctx, root, baseRevision, dirtyApproved, nil)
	return c.State, err
}

// Capture is a Snapshot taken through one WorktreeReader: the changed paths
// and the identity of every worktree path that was read, all from the same
// reads that produced the digest. Complete is true; a path absent from Entries
// was not read.
type Capture struct {
	State    State
	Changed  []string
	Entries  map[string]string
	Complete bool
}

// CaptureTarget is Snapshot with every worktree read going through r. Callers
// that later read the same bytes compare what they read with Entries, so what
// they describe is what the digest covers and not a state a writer produced
// between the reads.
func CaptureTarget(ctx context.Context, root, baseRevision string, dirtyApproved bool, r WorktreeReader) (Capture, error) {
	return capture(ctx, root, baseRevision, dirtyApproved, r)
}

func capture(ctx context.Context, root, baseRevision string, dirtyApproved bool, r WorktreeReader) (Capture, error) {
	var entries map[string]string
	if r != nil {
		entries = map[string]string{}
	}
	head, err := Output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return Capture{}, err
	}
	// assume-unchanged and skip-worktree make git status report a changed
	// tracked file as clean, so a snapshot cannot describe the bytes on disk
	// while either flag is set.
	if err := refuseHiddenIndexEntries(ctx, root); err != nil {
		return Capture{}, err
	}
	var paths []string
	if r != nil {
		paths, err = changedPathsWith(ctx, root, Trim(head), r, entries)
	} else {
		paths, err = changedPaths(ctx, root, Trim(head))
	}
	if err != nil {
		return Capture{}, err
	}
	dirty := len(paths) > 0
	if dirty && !dirtyApproved {
		return Capture{}, errors.New("dirty worktree policy violation")
	}
	diff, err := Output(ctx, root, "diff", "--binary", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", baseRevision+".."+Trim(head), "--")
	if err != nil {
		return Capture{}, err
	}
	sum := sha256.Sum256([]byte(diff))
	s := State{Head: Trim(head), Diff: hex.EncodeToString(sum[:])}
	// The index is read once, and both the digest and the staged flag come
	// from that one read, so they cannot describe two different indexes.
	index, err := Output(ctx, root, "ls-files", "--stage", "-z", "--")
	if err != nil {
		return Capture{}, err
	}
	tree, err := Output(ctx, root, "ls-tree", "-r", "-z", "--full-tree", s.Head)
	if err != nil {
		return Capture{}, err
	}
	s.Staged = indexDiffersFromTree(index, tree)
	if dirty {
		s.Worktree, err = digestWorktree(ctx, root, s.Head, paths, stagedEntries(index, paths), r, entries)
		if err != nil {
			return Capture{}, err
		}
	}
	return Capture{State: s, Changed: paths, Entries: entries, Complete: r != nil}, nil
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
	rootInfo, err := os.Stat(r)
	if err != nil {
		return fmt.Errorf("repository root: %w", err)
	}
	inside, err := hasAncestor(s, rootInfo)
	if err != nil {
		return err
	}
	if inside {
		return errors.New("store must be outside target repository")
	}
	return nil
}

// hasAncestor reports whether path, or any existing directory above it, is the
// same file as root. Comparing resolved strings misses a second spelling of the
// same directory -- a case alias on a case-insensitive file system, or a bind
// mount -- so identity is checked as well. Components that do not exist yet
// are skipped: an uncreated store is judged by where it would be created. Any
// other stat failure is returned, because an unreadable component could hide
// a repository alias and skipping it would let the check fail open.
func hasAncestor(path string, root os.FileInfo) (bool, error) {
	for current := path; ; {
		info, err := os.Stat(current)
		switch {
		case err == nil:
			if os.SameFile(info, root) {
				return true, nil
			}
		case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		default:
			return false, fmt.Errorf("store path: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false, nil
		}
		current = parent
	}
}

// refuseHiddenIndexEntries fails when any index entry carries the
// assume-unchanged or skip-worktree flag. `git ls-files -v` tags
// assume-unchanged entries with a lower-case letter and skip-worktree
// entries with S.
func refuseHiddenIndexEntries(ctx context.Context, root string) error {
	out, err := Output(ctx, root, "ls-files", "-v", "-z")
	if err != nil {
		return err
	}
	for _, entry := range strings.Split(out, "\x00") {
		if entry == "" {
			continue
		}
		tag := entry[0]
		if tag == 'S' || (tag >= 'a' && tag <= 'z') {
			return fmt.Errorf("index hides worktree changes (assume-unchanged or skip-worktree): %s", strings.TrimSpace(entry[1:]))
		}
	}
	return nil
}

// stagedEntries maps each of paths to its `ls-files --stage` entries
// ("<mode> <object> <stage>", several when unmerged), or "" when the index has
// no entry for it.
func stagedEntries(index string, paths []string) map[string]string {
	want := make(map[string]bool, len(paths))
	for _, p := range paths {
		want[p] = true
	}
	out := make(map[string]string, len(paths))
	for _, entry := range strings.Split(index, "\x00") {
		meta, path, ok := strings.Cut(entry, "\t")
		if !ok || !want[path] {
			continue
		}
		if out[path] != "" {
			out[path] += ";"
		}
		out[path] += meta
	}
	return out
}

// indexDiffersFromTree reports whether the index (ls-files --stage -z) holds
// any entry that is not exactly the tree's (ls-tree -r -z): a different mode
// or object, an added or removed path, or a conflict stage.
func indexDiffersFromTree(index, tree string) bool {
	want := map[string]string{}
	for _, e := range strings.Split(tree, "\x00") {
		meta, path, ok := strings.Cut(e, "\t")
		f := strings.Fields(meta)
		if ok && len(f) == 3 {
			want[path] = f[0] + " " + f[2]
		}
	}
	seen := 0
	for _, e := range strings.Split(index, "\x00") {
		if e == "" {
			continue
		}
		meta, path, ok := strings.Cut(e, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 || f[2] != "0" || want[path] != f[0]+" "+f[1] {
			return true
		}
		seen++
	}
	return seen != len(want)
}
