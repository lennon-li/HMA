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

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
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
	files, lines, _, err := measureChange(context.Background(), repo, base, true)
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

// F3: the byte window truncates (counts are not a bound; budget exceeded); the work cap gives
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
	_, lines, _, err := measureChange(context.Background(), repo, base, false)
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

// A file past the byte window is truncated: its counts are not a bound and the budget fact is exceeded.
func TestD5C9MByteWindowIsTruncated(t *testing.T) {
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

// Audit d5c11m F3 (narrowed after d5c13l F03): more changed paths than the
// manifest allows are refused from the path count alone, before the observation
// reads any per-path bytes. The test covers the committed case; a dirty target
// still pays the snapshot's own reads first (Gate B code, unchanged). A change just under the
// limit still succeeds, and the budget measurement (no limit) stays exact.
func TestD5C11MPathCapRefusesBeforePerPathReads(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "keep"), "k\n")
	base := d9Commit(t, repo, "base")
	for i := 0; i < maxObservedEntries+1; i++ {
		auditWrite(t, filepath.Join(repo, "many", fmt.Sprintf("f%05d", i)), "x\n")
	}
	d9Commit(t, repo, "many files")
	start := time.Now()
	_, err := observeTarget(context.Background(), repo, base, false)
	if err == nil || err.Error() != hoststate.ReasonReviewPacketTooLarge {
		t.Fatalf("observeTarget = %v, want %s", err, hoststate.ReasonReviewPacketTooLarge)
	}
	if time.Since(start) > 3*time.Second {
		t.Errorf("the refusal took %s: per-path bytes were read before the cap applied", time.Since(start))
	}
	// The budget measurement has no entry limit and stays exact.
	files, lines, _, err := measureChange(context.Background(), repo, base, false)
	if err != nil || files != maxObservedEntries+1 || lines != maxObservedEntries+1 {
		t.Errorf("measureChange = %d files, %d lines, %v; want %d and %d", files, lines, err, maxObservedEntries+1, maxObservedEntries+1)
	}
}

// Audit d5c11m F1: when a side is past the 16 MiB window, the entry is marked
// lines_truncated and its counts are of the first 16 MiB of each side only, not
// a bound on the whole change; the budget fact is "exceeded" whatever they are.
// Fixture: the base has 160 lines of A then 160 lines of B (64 KiB each, 20 MiB
// in all); the target keeps only the B lines (10 MiB). The true change is
// 0 added and 160 deleted, but the first 16 MiB of the base is not the whole.
func TestD5C11MTruncatedCountsAreNotABoundButBudgetIsExceeded(t *testing.T) {
	repo := gitRepo(t)
	line := func(c byte) string { return strings.Repeat(string(c), 65535) + "\n" }
	var base, target strings.Builder
	for i := 0; i < 160; i++ {
		base.WriteString(line('A'))
	}
	for i := 0; i < 160; i++ {
		base.WriteString(line('B'))
		target.WriteString(line('B'))
	}
	auditWrite(t, filepath.Join(repo, "f"), base.String())
	b := d9Commit(t, repo, "base")
	auditWrite(t, filepath.Join(repo, "f"), target.String())
	d9Commit(t, repo, "keep B")
	entries, err := observeTarget(context.Background(), repo, b, false)
	if err != nil || len(entries) != 1 {
		t.Fatalf("observation = %+v, %v", entries, err)
	}
	e := entries[0]
	if !e.LinesTruncated {
		t.Fatalf("entry = %+v, want lines_truncated (the base is past the 16 MiB window)", e)
	}
	if e.Added == 0 && e.Deleted == 160 {
		t.Log("the window counts happen to equal the true change here")
	}
	_, lines, truncated, err := measureChange(context.Background(), repo, b, false)
	if err != nil || !truncated {
		t.Errorf("measureChange = %d lines, truncated %v, %v; want the truncation reported as its own fact", lines, truncated, err)
	}
	// The packet documents the semantics: the field comment must not call the
	// counts a bound.
	src, rerr := os.ReadFile("review.go")
	if rerr != nil {
		t.Fatal(rerr)
	}
	if strings.Contains(string(src), "the counts are a lower bound") || strings.Contains(string(src), "counts are a LOWER bound") {
		t.Error("a comment still calls the truncated counts a lower bound")
	}
}

// Audit d5c11m A1: findings import re-checks the live target after it measured
// the facts, so they describe the bound target. The check is in the code path;
// the unit coverage is the existing live-target tests plus this ordering test.
func TestD5C11MFindingsImportRechecksTargetAfterFacts(t *testing.T) {
	b, err := os.ReadFile("review.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	i := strings.Index(src, "findingFacts(boundedCtx()")
	j := strings.Index(src[i:], "requireLiveTarget(history, rr.Review.ReviewDispatchDigest)")
	if i < 0 || j < 0 {
		t.Fatal("findings import does not re-check the live target after measuring the facts")
	}
}

// Audit d5c13l F04: a truncated measurement exceeds the line budget even when
// the plan declares the largest integer budget (a numeric sentinel could not).
func TestD5C13LTruncatedExceedsEvenTheLargestLineBudget(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		name      string
		lines     int
		truncated bool
		budget    int
		want      bool
	}{
		{"truncated, huge budget", 5, true, maxInt, true},
		{"not truncated, huge budget", 5, false, maxInt, false},
		{"not truncated, over budget", 11, false, 10, true},
		{"not truncated, within budget", 10, false, 10, false},
		{"truncated, small budget", 0, true, 10, true},
	} {
		facts := hoststate.FindingFacts{BudgetExceeded: map[string]bool{}}
		budgetFacts(&facts, 1, tc.lines, tc.truncated, model.UnitBudgets{MaxFiles: 10, MaxDiffLines: tc.budget})
		if got := facts.BudgetExceeded["max_diff_lines"]; got != tc.want {
			t.Errorf("%s: max_diff_lines exceeded = %v, want %v", tc.name, got, tc.want)
		}
	}
}
