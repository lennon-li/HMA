package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
)

func TestD5C7PLineDiffBoundedFrontier(t *testing.T) {
	old := []byte(strings.Repeat("same\n", 5000))
	newFile := []byte(strings.Repeat("new\n", 5000))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	a, d, truncated := lineDiffContext(ctx, nil, newFile)
	runtime.ReadMemStats(&after)
	if a != 5000 || d != 0 || truncated {
		t.Fatalf("new-file counts = %d/%d truncated=%v", a, d, truncated)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 12<<20 {
		t.Fatalf("new-file line diff allocated %d bytes", allocated)
	}
	a, d, truncated = lineDiffContext(ctx, old, old)
	if a != 0 || d != 0 || truncated {
		t.Fatalf("identical counts = %d/%d truncated=%v", a, d, truncated)
	}
	updated := []byte(strings.Replace(string(old), "same\n", "same\n", 1))
	updated = append([]byte("prefix\n"), updated...)
	a, d, truncated = lineDiffContext(ctx, old, updated)
	if a != 1 || d != 0 || truncated {
		t.Fatalf("near-identical counts = %d/%d truncated=%v", a, d, truncated)
	}
	leftLines, rightLines := make([]string, 5000), make([]string, 5000)
	for i := range leftLines {
		leftLines[i] = fmt.Sprintf("left-%05d\n", i)
		rightLines[i] = fmt.Sprintf("right-%05d\n", i)
	}
	left, right := []byte(strings.Join(leftLines, "")), []byte(strings.Join(rightLines, ""))
	runtime.GC()
	runtime.ReadMemStats(&before)
	a, d, truncated = lineDiffContext(context.Background(), left, right)
	runtime.ReadMemStats(&after)
	if a != 5000 || d != 5000 || !truncated {
		t.Fatalf("bounded distinct counts = %d/%d truncated=%v", a, d, truncated)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 12<<20 {
		t.Fatalf("bounded distinct line diff allocated %d bytes", allocated)
	}
}

func TestD5C7PLineDiffCapsPathologicalEditDistance(t *testing.T) {
	leftLines := make([]string, 100000)
	rightLines := make([]string, 100000)
	for i := range leftLines {
		leftLines[i] = fmt.Sprintf("left-%06d\n", i)
		rightLines[i] = fmt.Sprintf("right-%06d\n", i)
	}
	left, right := []byte(strings.Join(leftLines, "")), []byte(strings.Join(rightLines, ""))
	start := time.Now()
	a, d, truncated := lineDiffContext(context.Background(), left, right)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("pathological diff took %s", elapsed)
	}
	if !truncated || a != 100000 || d != 100000 {
		t.Fatalf("pathological counts = %d/%d truncated=%v", a, d, truncated)
	}
	for i := range leftLines {
		leftLines[i] = fmt.Sprintf("even-%06d\n", i*2)
		rightLines[i] = fmt.Sprintf("odd-%06d\n", i*2+1)
	}
	start = time.Now()
	a, d, truncated = lineDiffContext(context.Background(), []byte(strings.Join(leftLines, "")), []byte(strings.Join(rightLines, "")))
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("alternating pathological diff took %s", elapsed)
	}
	if !truncated || a != 100000 || d != 100000 {
		t.Fatalf("alternating pathological counts = %d/%d truncated=%v", a, d, truncated)
	}
}

func TestD5C7PLineDiffNumstatProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(0xD5C7))
	repo := gitRepo(t)
	for i := 0; i < 40; i++ {
		oldLines, newLines := randomTestLines(rng), randomTestLines(rng)
		old, newData := []byte(strings.Join(oldLines, "")), []byte(strings.Join(newLines, ""))
		added, deleted, truncated := lineDiffContext(context.Background(), old, newData)
		if truncated {
			t.Fatalf("small generated pair %d truncated", i)
		}
		if err := os.WriteFile(filepath.Join(repo, "a"), old, 0600); err != nil {
			t.Fatal(err)
		}
		auditGit(t, repo, "add", "a")
		auditGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", fmt.Sprintf("old-%d", i))
		oldHead := auditGit(t, repo, "rev-parse", "HEAD")
		if err := os.WriteFile(filepath.Join(repo, "a"), newData, 0600); err != nil {
			t.Fatal(err)
		}
		auditGit(t, repo, "add", "a")
		auditGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", fmt.Sprintf("new-%d", i))
		out := strings.TrimSpace(auditGit(t, repo, "diff", "--numstat", oldHead, "HEAD"))
		var gitAdded, gitDeleted int
		if out != "" {
			if _, err := fmt.Sscanf(out, "%d\t%d", &gitAdded, &gitDeleted); err != nil {
				t.Fatalf("parse numstat %q: %v", out, err)
			}
		}
		if added != gitAdded || deleted != gitDeleted {
			t.Fatalf("case %d Myers=%d/%d git=%d/%d", i, added, deleted, gitAdded, gitDeleted)
		}
	}
}

func randomTestLines(r *rand.Rand) []string {
	n := r.Intn(14)
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line-%d\n", r.Intn(8))
	}
	return lines
}

func TestD5C7PFullHashWithBoundedBlobWindow(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "seed"), "seed\n")
	auditGit(t, repo, "add", "seed")
	auditGit(t, repo, "commit", "-qm", "seed")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	data := bytesOfSize(20 << 20)
	if err := os.WriteFile(filepath.Join(repo, "large"), data, 0600); err != nil {
		t.Fatal(err)
	}
	auditGit(t, repo, "add", "large")
	auditGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "-qm", "large")
	entries, err := observeTarget(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(data)
	if len(entries) != 1 || entries[0].SHA256 != hex.EncodeToString(want[:]) || !entries[0].LinesTruncated {
		t.Fatalf("large blob observation = %+v", entries)
	}
	if err := os.WriteFile(filepath.Join(repo, "dirty"), data, 0600); err != nil {
		t.Fatal(err)
	}
	_, _, exists, dirtyHash, dirtyTruncated, err := worktreeBytes(context.Background(), repo, "dirty")
	if err != nil || !exists || dirtyHash != hex.EncodeToString(want[:]) || !dirtyTruncated {
		t.Fatalf("large worktree observation exists=%v hash=%s truncated=%v err=%v", exists, dirtyHash, dirtyTruncated, err)
	}
}

func bytesOfSize(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
		if i%80 == 79 {
			b[i] = '\n'
		}
	}
	return b
}

func TestD5C7PParentPathMatchesSnapshotSemantics(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "old", "f"), "same\n")
	auditWrite(t, filepath.Join(repo, "other", "f"), "same\n")
	auditWrite(t, filepath.Join(repo, "nested", "ok"), "nested\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	if err := os.RemoveAll(filepath.Join(repo, "old")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other", filepath.Join(repo, "old")); err != nil {
		t.Fatal(err)
	}
	entries, err := observeTarget(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]observedEntry{}
	for _, e := range entries {
		got[e.Path] = e
	}
	if e := got["old/f"]; e.Status != "deleted" || e.SHA256 != "" || e.Added != 0 || e.Deleted != 1 {
		t.Fatalf("symlink-parent entry = %+v", e)
	}
	if data, _, exists, _, _, err := worktreeBytes(context.Background(), repo, "nested/ok"); err != nil || !exists || string(data) != "nested\n" {
		t.Fatalf("normal nested path data=%q exists=%v err=%v", data, exists, err)
	}
	if err := os.Remove(filepath.Join(repo, "old")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "old"), []byte("parent"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, exists, _, _, err := worktreeBytes(context.Background(), repo, "old/f"); err != nil || exists {
		t.Fatalf("regular-file parent exists=%v err=%v", exists, err)
	}
}

func TestD5C7PLegacyWaiverReplayAndCompleteness(t *testing.T) {
	scope := model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"}
	legacy := model.Waiver{ID: "legacy", Scope: scope, Active: true, Actor: "human"}
	core := []model.Record{{Kind: model.KindCriterion, Metadata: model.RecordMetadata{HeadAnchor: "criterion-anchor"}, Waivers: []model.Waiver{legacy}}}
	terms, err := packetWaiverTerms(core, map[model.ScopeSelector]bool{scope: true})
	if err != nil || len(terms) != 1 || !terms[0].Legacy || terms[0].Record != "criterion-anchor" || terms[0].Waiver == nil || *terms[0].Waiver != legacy {
		t.Fatalf("legacy criterion term=%+v err=%v", terms, err)
	}
	core = append(core, model.Record{Kind: model.KindFinding, Metadata: model.RecordMetadata{HeadAnchor: "clear-anchor"}, Waivers: []model.Waiver{{Scope: scope, Active: false}}})
	if terms, err = packetWaiverTerms(core, map[model.ScopeSelector]bool{scope: false}); err != nil || len(terms) != 0 {
		t.Fatalf("cleared legacy terms=%+v err=%v", terms, err)
	}
	op := model.WaiverOperation{Operation: model.WaiverOperationGrant, Scope: scope, Justification: "approved"}
	core = append(core, model.Record{Kind: model.KindFinding, Metadata: model.RecordMetadata{HeadAnchor: "operation-anchor"}, WaiverOperation: &op})
	if terms, err = packetWaiverTerms(core, map[model.ScopeSelector]bool{scope: true}); err != nil || len(terms) != 1 || terms[0].Operation == nil || terms[0].Record != "operation-anchor" {
		t.Fatalf("operation grant terms=%+v err=%v", terms, err)
	}
	mixedScope := model.ScopeSelector{Kind: model.ScopeFinding, Target: "F1"}
	mixedOperation := model.WaiverOperation{Operation: model.WaiverOperationGrant, Scope: mixedScope, Justification: "operation grant"}
	mixedCore := []model.Record{
		{Kind: model.KindCriterion, Metadata: model.RecordMetadata{HeadAnchor: "legacy-mixed"}, Waivers: []model.Waiver{legacy}},
		{Kind: model.KindFinding, Metadata: model.RecordMetadata{HeadAnchor: "operation-mixed"}, WaiverOperation: &mixedOperation},
	}
	terms, err = packetWaiverTerms(mixedCore, map[model.ScopeSelector]bool{scope: true, mixedScope: true})
	if err != nil || len(terms) != 2 || terms[0].Legacy == terms[1].Legacy {
		t.Fatalf("mixed active waiver terms=%+v err=%v", terms, err)
	}
	if _, err = packetWaiverTerms(nil, map[model.ScopeSelector]bool{scope: true}); err == nil || err.Error() != hoststate.ReasonReviewPacketWaiverIncomplete {
		t.Fatalf("incomplete error=%v", err)
	}
}

func TestD5C7PInvalidUTF8ManifestPathRefused(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "seed"), "seed\n")
	auditGit(t, repo, "add", "seed")
	auditGit(t, repo, "commit", "-qm", "seed")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	name := string([]byte{'p', 0xfe, 'q'})
	path := filepath.Join(repo, name)
	if err := os.WriteFile(path, []byte("content\n"), 0600); err != nil {
		t.Skipf("filesystem refuses invalid UTF-8 name: %v", err)
	}
	auditGit(t, repo, "add", "--", name)
	auditGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "-qm", "invalid path")
	if _, err := buildChangedFileManifest(context.Background(), repo, base, false); err == nil || err.Error() != hoststate.ReasonReviewPacketPathUnsupported {
		t.Fatalf("invalid path refusal=%v", err)
	}
	if _, _, _, err := measureChange(context.Background(), repo, base, false); err == nil || err.Error() != hoststate.ReasonReviewPacketPathUnsupported {
		t.Fatalf("invalid path budget result=%v", err)
	}
}

// lineDiffContext adapts the live lineDiffCounts to the (added, deleted,
// truncated) shape these tests use; truncated folds both bound flags.
func lineDiffContext(ctx context.Context, a, b []byte) (added, deleted int, truncated bool) {
	c := lineDiffCounts(ctx, a, b)
	return c.Added, c.Deleted, c.Truncated || c.UpperBound
}
