package main

// Regression tests for independent audit hma-gate-d-audit-20261024-d5c9m
// (findings F1-F6), ported from its reproductions.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/repostate"
)

func d9Commit(t *testing.T, repo, msg string) string {
	t.Helper()
	auditGit(t, repo, "add", "-A")
	auditGit(t, repo, "commit", "-qm", msg, "--allow-empty")
	return auditGit(t, repo, "rev-parse", "HEAD")
}

// F1: a committed file replaced by a directory is an ordinary refactor. The
// snapshot accepts the dirty target, so the observation does too: the file is
// deleted, the files below the directory are added, and the dirty result
// equals the committed one.
func TestD5C9MFileReplacedByDirectory(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "a"), "file\n")
	base := d9Commit(t, repo, "base")
	auditGit(t, repo, "rm", "-q", "a")
	auditWrite(t, filepath.Join(repo, "a", "b"), "nested\n")
	d9Commit(t, repo, "a becomes a directory")
	auditWrite(t, filepath.Join(repo, "untracked.txt"), "dirty\n")
	if _, err := repostate.Snapshot(context.Background(), repo, base, true); err != nil {
		t.Fatalf("snapshot refuses the target: %v", err)
	}
	committed, err := observeTarget(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	dirty, err := observeTarget(context.Background(), repo, base, true)
	if err != nil {
		t.Fatalf("dirty observation refuses a target the snapshot accepts: %v", err)
	}
	status := map[string]string{}
	for _, e := range dirty {
		status[e.Path] = e.Status
	}
	if status["a"] != "deleted" || status["a/b"] != "added" || status["untracked.txt"] != "added" {
		t.Fatalf("dirty statuses = %v", status)
	}
	for _, c := range committed {
		if status[c.Path] != c.Status {
			t.Errorf("%s: committed %q, dirty %q", c.Path, c.Status, status[c.Path])
		}
	}
	files, lines, err := measureChange(context.Background(), repo, base, true)
	if err != nil || files != 3 || lines != 3 {
		t.Fatalf("measureChange = %d files, %d lines, %v; want 3, 3", files, lines, err)
	}
}

// F2: line slicing must not cost memory per line. Two 16 MiB files of short
// lines finish quickly and stay far below the cost of one slice header per line.
func TestD5C9MLineDiffMemoryBounded(t *testing.T) {
	a := bytes.Repeat([]byte("\n"), 16<<20)
	b := append(bytes.Repeat([]byte("\n"), 16<<20-1), 'x')
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	c := lineDiffCounts(context.Background(), a, b)
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	if used := (after.TotalAlloc - before.TotalAlloc) >> 20; used > 256 {
		t.Errorf("line diff allocated %d MiB", used)
	}
	if elapsed > time.Second {
		t.Errorf("line diff took %s", elapsed)
	}
	if !c.UpperBound || c.Added != 16<<20 || c.Deleted != 16<<20 {
		t.Errorf("counts = %+v, want the conservative upper bound", c)
	}
}

// F3: the byte window gives a LOWER bound (budget exceeded); the work cap gives
// an UPPER bound that the budget counts as added plus deleted. A fully
// rewritten 5000-line file is exactly 10000 lines and within a 20000 budget.
func TestD5C9MWorkCapIsUpperBoundNotExceeded(t *testing.T) {
	repo := gitRepo(t)
	var a, b strings.Builder
	for i := 0; i < 5000; i++ {
		fmt.Fprintf(&a, "left-%05d\n", i)
		fmt.Fprintf(&b, "right-%05d\n", i)
	}
	auditWrite(t, filepath.Join(repo, "f"), a.String())
	base := d9Commit(t, repo, "base")
	auditWrite(t, filepath.Join(repo, "f"), b.String())
	d9Commit(t, repo, "rewrite")
	m, err := buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil || len(m) != 1 {
		t.Fatalf("manifest = %+v, %v", m, err)
	}
	if m[0].Added != 5000 || m[0].Deleted != 5000 || !m[0].LinesUpperBound || m[0].LinesTruncated {
		t.Errorf("entry = %+v, want 5000/5000 upper bound, not truncated", m[0])
	}
	_, lines, err := measureChange(context.Background(), repo, base, false)
	if err != nil || lines != 10000 {
		t.Errorf("measureChange lines = %d, %v; want 10000", lines, err)
	}
}

// The depth and work cap are real: without them a 15000-line file with about
// a sixth of its lines edited would return an exact count; the capped result is
// the conservative bound. Removing the work check must change this result.
func TestD5C9MWorkCapChangesResult(t *testing.T) {
	var a, b strings.Builder
	for i := 0; i < 15000; i++ {
		fmt.Fprintf(&a, "line-%05d\n", i)
		if i%6 == 0 {
			fmt.Fprintf(&b, "edited-%05d\n", i)
		} else {
			fmt.Fprintf(&b, "line-%05d\n", i)
		}
	}
	c := lineDiffCounts(context.Background(), []byte(a.String()), []byte(b.String()))
	if !c.UpperBound || c.Added != 15000 || c.Deleted != 15000 {
		t.Fatalf("counts = %+v, want the capped upper bound 15000/15000", c)
	}
}

// A file past the byte window gives a lower bound: the budget fact is exceeded.
func TestD5C9MByteWindowIsTruncatedLowerBound(t *testing.T) {
	big := bytes.Repeat([]byte("x\n"), 9<<20)
	c := lineDiffCounts(context.Background(), nil, big)
	if !c.Truncated {
		t.Fatalf("counts = %+v, want truncated", c)
	}
}

// F4: a path absent from both the base and the target contributes nothing.
func TestD5C9MAbsentInBaseAndTargetIsOmitted(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "keep"), "k\n")
	base := d9Commit(t, repo, "base")
	auditWrite(t, filepath.Join(repo, "x"), "x1\nx2\n")
	d9Commit(t, repo, "add x")
	os.Remove(filepath.Join(repo, "x"))
	auditWrite(t, filepath.Join(repo, "y"), "y\n")
	auditGit(t, repo, "add", "y")
	os.Remove(filepath.Join(repo, "y"))
	if _, err := repostate.Snapshot(context.Background(), repo, base, true); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	entries, err := observeTarget(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Path == "y" {
			t.Errorf("y (staged, then removed; absent in base and target) is listed: %+v", e)
		}
	}
	// x was added after the base and is gone from the target: also absent in both.
	for _, e := range entries {
		if e.Path == "x" && e.Status == "added" {
			t.Errorf("x is listed as added although it is absent from the target: %+v", e)
		}
	}
}

// F5: contract-bound packets name how lines are counted; the field is absent
// from the JSON of a state without a manifest.
func TestD5C9MPacketNamesLineCounting(t *testing.T) {
	cf := newCreditFixture(t, "true")
	st := packetState(t, cf.packetPath)
	if st.LineCounts != "minimal-lcs" {
		t.Errorf("line_counts = %q, want minimal-lcs", st.LineCounts)
	}
	raw, _ := json.Marshal(packetCoreState{})
	if strings.Contains(string(raw), "line_counts") {
		t.Errorf("a state without a manifest carries line_counts: %s", raw)
	}
}
