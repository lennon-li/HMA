package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/repostate"
)

type observedEntry struct {
	Path, Status                                    string
	Added, Deleted                                  int
	SHA256                                          string
	OldMode, NewMode                                string
	Binary, Staged, LinesTruncated, LinesUpperBound bool
}

type treeEntry struct{ mode, typ, oid string }

// observeTarget is the shared raw-byte observation used by both the packet
// manifest and disposition budgets. Counts use a bounded Go Myers line diff;
// this avoids Git attributes and pathspec interpretation entirely.
func observeTarget(ctx context.Context, root, base string, dirty bool) ([]observedEntry, error) {
	return observeTargetLimit(ctx, root, base, dirty, maxObservedEntries)
}

// observeTargetLimit is observeTarget with an entry limit: more changed paths
// than limit refuse before this observation reads any per-path bytes (blob or
// worktree content). That excludes repostate.ChangedPaths's own, earlier
// reads, which discover the changed set for a dirty target (see below) and
// are not bounded by this cap. A limit of 0 means none (the budget
// measurement must stay exact for a large change).
func observeTargetLimit(ctx context.Context, root, base string, dirty bool, limit int) ([]observedEntry, error) {
	headOut, err := repostate.Output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	head := repostate.Trim(headOut)
	baseTree, err := listTree(ctx, root, base)
	if err != nil {
		return nil, err
	}
	headTree, err := listTree(ctx, root, head)
	if err != nil {
		return nil, err
	}
	indexOut, err := repostate.Output(ctx, root, "ls-files", "-s", "-z")
	if err != nil {
		return nil, err
	}
	index := parseIndex(indexOut)
	paths := map[string]bool{}
	for p, b := range baseTree {
		if h, ok := headTree[p]; !ok || b != h {
			paths[p] = true
		}
	}
	for p, h := range headTree {
		if b, ok := baseTree[p]; !ok || b != h {
			paths[p] = true
		}
	}
	if dirty {
		// A platform without the hardened per-entry reader must refuse before
		// repostate.ChangedPaths ever runs: that call reads every tracked
		// file's worktree bytes by plain pathname (Gate B, unchanged, and not
		// anchored), so letting it run first would defeat the Linux-only
		// restriction below instead of enforcing it.
		if errDirtyObservationUnsupported != nil {
			return nil, errDirtyObservationUnsupported
		}
		changed, e := repostate.ChangedPaths(ctx, root, head)
		if e != nil {
			return nil, e
		}
		for _, p := range changed {
			paths[p] = true
		}
	}
	ordered := make([]string, 0, len(paths))
	for p := range paths {
		ordered = append(ordered, p)
	}
	sort.Strings(ordered)
	out := make([]observedEntry, 0, len(ordered))
	// The packet refuses more than maxObservedEntries entries. Refuse on the
	// path count before this observation reads any per-path bytes (blob or
	// worktree content), so a very large change fails fast with the named
	// reason. Reads that come earlier are not covered: for a dirty target,
	// repostate.ChangedPaths (the Gate B snapshot logic) reads tracked worktree
	// files to find what changed, and the packet command takes the artifact
	// snapshot before it builds the manifest.
	if limit > 0 && len(ordered) > limit {
		return nil, errors.New(hoststate.ReasonReviewPacketTooLarge)
	}
	for _, p := range ordered {
		if !utf8.ValidString(p) {
			return nil, fmt.Errorf(hoststate.ReasonReviewPacketPathUnsupported)
		}
		if strings.HasSuffix(p, "/") {
			// git lists an untracked nested repository as "dir/"; the snapshot
			// refuses such a target, so describing it by skipping it would
			// understate the change.
			return nil, fmt.Errorf("target path %q is a directory (nested repository?)", p)
		}
		b, bok := baseTree[p]
		t, tok := headTree[p]
		var target []byte
		var targetHash string
		var targetTruncated bool
		if dirty {
			if tok && t.mode == "160000" {
				entry := observedEntry{Path: p, NewMode: t.mode, SHA256: t.oid, Status: "submodule"}
				if bok {
					entry.OldMode = b.mode
				}
				entry.Staged = !sameTree(index[p], headTree[p])
				out = append(out, entry)
				continue
			}
			var mode string
			var exists bool
			target, mode, exists, targetHash, targetTruncated, err = worktreeBytes(ctx, root, p)
			if err != nil {
				return nil, err
			}
			if !exists {
				tok = false
			} else if mode == "" {
				continue
			} else {
				t = treeEntry{mode: mode}
				tok = true
			}
		} else if tok && t.mode != "160000" {
			target, targetHash, targetTruncated, err = blobBytes(ctx, root, t.oid)
			if err != nil {
				return nil, err
			}
		}
		if !bok && !tok {
			// Absent from the base and from the target (added after the base
			// and gone again, or staged and removed): it contributes nothing.
			continue
		}
		entry := observedEntry{Path: p}
		if bok {
			entry.OldMode = b.mode
		}
		if tok {
			entry.NewMode = t.mode
		}
		entry.Staged = !sameTree(index[p], headTree[p])
		switch {
		case !bok:
			entry.Status = "added"
		case !tok:
			entry.Status = "deleted"
		case b.mode == "160000" || t.mode == "160000":
			entry.Status = "submodule"
		case isLinkOrRegular(b.mode) != isLinkOrRegular(t.mode):
			entry.Status = "type_changed"
		case strings.HasPrefix(b.mode, "100") != strings.HasPrefix(t.mode, "100"):
			entry.Status = "type_changed"
		default:
			entry.Status = "modified"
		}
		if entry.Status == "submodule" {
			if tok {
				entry.SHA256 = t.oid
			} else if bok {
				entry.SHA256 = b.oid
			}
			out = append(out, entry)
			continue
		}
		if tok {
			entry.SHA256 = targetHash
		}
		if tok && hasNUL(target) {
			entry.Binary = true
		}
		var before []byte
		beforeTruncated := false
		if bok && b.mode != "160000" {
			before, _, beforeTruncated, err = blobBytes(ctx, root, b.oid)
			if err != nil {
				return nil, err
			}
		}
		if bok && b.mode != "160000" && hasNUL(before) {
			entry.Binary = true
		}
		counts := lineDiffCounts(ctx, before, target)
		entry.Added, entry.Deleted = counts.Added, counts.Deleted
		entry.LinesTruncated = counts.Truncated || targetTruncated || beforeTruncated
		entry.LinesUpperBound = counts.UpperBound
		out = append(out, entry)
	}
	return out, nil
}

func isLinkOrRegular(mode string) bool { return mode == "120000" || strings.HasPrefix(mode, "100") }
func sameTree(a, b treeEntry) bool     { return a == b }

func listTree(ctx context.Context, root, rev string) (map[string]treeEntry, error) {
	raw, err := repostate.Output(ctx, root, "ls-tree", "-r", "-z", "--full-tree", rev)
	if err != nil {
		return nil, err
	}
	m := map[string]treeEntry{}
	for _, rec := range strings.Split(raw, "\x00") {
		if rec == "" {
			continue
		}
		i := strings.IndexByte(rec, '\t')
		if i < 0 {
			return nil, fmt.Errorf("malformed ls-tree record")
		}
		f := strings.Fields(rec[:i])
		if len(f) != 3 {
			return nil, fmt.Errorf("malformed ls-tree header")
		}
		m[rec[i+1:]] = treeEntry{f[0], f[1], f[2]}
	}
	return m, nil
}
func parseIndex(raw string) map[string]treeEntry {
	m := map[string]treeEntry{}
	for _, rec := range strings.Split(raw, "\x00") {
		if rec == "" {
			continue
		}
		i := strings.IndexByte(rec, '\t')
		if i < 0 {
			continue
		}
		f := strings.Fields(rec[:i])
		if len(f) == 3 && f[2] == "0" {
			m[rec[i+1:]] = treeEntry{f[0], "blob", f[1]}
		}
	}
	return m
}
func blobBytes(ctx context.Context, root, oid string) ([]byte, string, bool, error) {
	return repostate.BlobWindow(ctx, root, oid, maxCountedBytes)
}

// entryKind classifies a worktree path found without following any symlink.
type entryKind int

const (
	entryAbsent entryKind = iota
	entryDir
	entrySymlink
	entryRegular
	entrySpecial
)

// worktreeEntry is what openWorktreeEntry found. For a regular file, file is
// open and the caller closes it.
type worktreeEntry struct {
	kind entryKind
	link string
	file *os.File
	exec bool
}

func worktreeBytes(ctx context.Context, root, p string) ([]byte, string, bool, string, bool, error) {
	ent, err := openWorktreeEntry(root, p)
	if err != nil {
		return nil, "", false, "", false, err
	}
	switch ent.kind {
	case entrySymlink:
		data := []byte(ent.link)
		return data, "120000", true, hashBytes(data), false, nil
	case entryRegular:
		data, sum, truncated, e := streamWindow(ctx, ent.file)
		closeErr := ent.file.Close()
		if e != nil {
			return nil, "", false, "", false, e
		}
		if closeErr != nil {
			return nil, "", false, "", false, closeErr
		}
		mode := "100644"
		if ent.exec {
			mode = "100755"
		}
		return data, mode, true, sum, truncated, nil
	case entrySpecial:
		// A FIFO, socket or device is not part of the target (git does not
		// track it and the digest does not cover it); it is never opened.
		return nil, "", true, "", false, nil
	default:
		// Absent, or a directory at a path that is a file in the tree (foo.go
		// replaced by foo/): there is no file entry here. A nested untracked
		// repository is refused earlier (its listed path ends in "/").
		return nil, "", false, "", false, nil
	}
}

// ctxReader fails a read once the context is done.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

func streamWindow(ctx context.Context, r io.Reader) ([]byte, string, bool, error) {
	h := sha256.New()
	var window limitedObservationWindow
	if _, err := io.Copy(io.MultiWriter(h, &window), ctxReader{ctx, r}); err != nil {
		return nil, "", false, err
	}
	return window.data, hex.EncodeToString(h.Sum(nil)), window.overflow, nil
}

type limitedObservationWindow struct {
	data     []byte
	overflow bool
}

func (w *limitedObservationWindow) Write(p []byte) (int, error) {
	if len(w.data) < maxCountedBytes {
		n := maxCountedBytes - len(w.data)
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

func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// lineCounts is the result of a line diff. Truncated: a side exceeded the
// 16 MiB observation window; the counts are those of the first 16 MiB of each
// side and are NOT a bound on the whole change (the budget treats the entry as
// exceeded, and the packet says so). UpperBound: the
// edit-distance work or the line cap was hit, so (Added, Deleted) is a sound
// UPPER bound (every line of the two sides counted as changed).
type lineCounts struct {
	Added, Deleted        int
	Truncated, UpperBound bool
}

const (
	// maxObservedEntries is the manifest cap (also applied to the packet).
	maxObservedEntries = 2000
	observationWindow  = 16 << 20
	maxObservedLines   = 1_000_000
	maxEditDepth       = 20000
	maxEditWork        = 50_000_000
)

// lineIndex holds a side's lines as offsets and 64-bit hashes (12 bytes per
// line) instead of one slice header per line.
type lineIndex struct {
	data  []byte
	start []uint32
	hash  []uint64
}

func newLineIndex(data []byte, lines int) lineIndex {
	ix := lineIndex{data: data, start: make([]uint32, 0, lines+1), hash: make([]uint64, 0, lines)}
	begin := 0
	h := uint64(14695981039346656037)
	for i, c := range data {
		h ^= uint64(c)
		h *= 1099511628211
		if c == '\n' {
			ix.start = append(ix.start, uint32(begin))
			ix.hash = append(ix.hash, h)
			begin = i + 1
			h = 14695981039346656037
		}
	}
	if begin < len(data) {
		ix.start = append(ix.start, uint32(begin))
		ix.hash = append(ix.hash, h)
	}
	ix.start = append(ix.start, uint32(len(data)))
	return ix
}

func (ix lineIndex) line(i int) []byte { return ix.data[ix.start[i]:ix.start[i+1]] }

func lineDiffCounts(ctx context.Context, a, b []byte) lineCounts {
	var c lineCounts
	c.Truncated = len(a) > observationWindow || len(b) > observationWindow
	if len(a) > observationWindow {
		a = a[:observationWindow]
	}
	if len(b) > observationWindow {
		b = b[:observationWindow]
	}
	if bytes.Equal(a, b) {
		return c
	}
	// Count first, slice never: a side's line count costs one pass and no
	// allocation.
	n, m := observedCountLines(a), observedCountLines(b)
	if n == 0 {
		c.Added = m
		return c
	}
	if m == 0 {
		c.Deleted = n
		return c
	}
	upper := func() lineCounts {
		c.Added, c.Deleted, c.UpperBound = m, n, true
		return c
	}
	if hasNUL(a) || hasNUL(b) || n+m > maxObservedLines {
		return upper()
	}
	x, y := newLineIndex(a, n), newLineIndex(b, m)
	maxD := n + m
	if maxD > maxEditDepth {
		maxD = maxEditDepth
	}
	offset := maxD + 1
	v := make([]int, 2*maxD+3)
	for d := 0; d <= maxD; d++ {
		if (n+m)*d > maxEditWork {
			return upper()
		}
		if d&31 == 0 && ctx.Err() != nil {
			return upper()
		}
		for k := -d; k <= d; k += 2 {
			idx := offset + k
			var xx int
			if k == -d || (k != d && v[idx-1] < v[idx+1]) {
				xx = v[idx+1]
			} else {
				xx = v[idx-1] + 1
			}
			yy := xx - k
			steps := 0
			for xx < n && yy < m && x.hash[xx] == y.hash[yy] && bytes.Equal(x.line(xx), y.line(yy)) {
				xx++
				yy++
				steps++
				if steps&4095 == 0 && ctx.Err() != nil {
					return upper()
				}
			}
			v[idx] = xx
			if xx >= n && yy >= m {
				delta := m - n
				c.Added, c.Deleted = (d+delta)/2, (d-delta)/2
				return c
			}
		}
	}
	return upper()
}

func hasNUL(b []byte) bool {
	for i := 0; i < len(b) && i < 8000; i++ {
		if b[i] == 0 {
			return true
		}
	}
	return false
}
func observedCountLines(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	n := 0
	for _, c := range b {
		if c == '\n' {
			n++
		}
	}
	if b[len(b)-1] != '\n' {
		n++
	}
	if n == 0 {
		return 1
	}
	return n
}
