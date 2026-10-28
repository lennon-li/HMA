package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/store"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
)

// importFindingsOf imports a fix-required review that carries fs, then the
// findings, and returns the resulting core dispositions.
func importFindingsOf(t *testing.T, cf *creditFixture, fs ...map[string]any) map[string]model.FindingDisposition {
	t.Helper()
	rp := cf.report("fix-required", "isolated", withFindings(fs...))
	cf.receipt(rp, nil)
	if err := cf.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := cf.importFindings(cf.latestReview(), rp, "2026-09-16T12:10:30Z"); err != nil {
		t.Fatalf("findings import: %v", err)
	}
	return cf.coreFindings()
}

func ref(kind, id string) map[string]any { return map[string]any{"kind": kind, "id": id} }

// Gate A v3 §4.1 / architecture §9: disposition follows the rule, never the
// severity or the label. Exit test of Gate D part D5b.
func TestDispositionFollowsSection9(t *testing.T) {
	// Within budget; no kernel rule holds. Severity and labels change nothing.
	t.Run("unlinked, unproven and unregistered stay advisory", func(t *testing.T) {
		got := importFindingsOf(t, newCreditFixture(t, "true"),
			map[string]any{"id": "F1", "severity": "low"},
			map[string]any{"id": "F2", "severity": "critical"},
			map[string]any{"id": "F3", "severity": "critical", "hma_ref": ref("kernel_rule", "authority")},
			map[string]any{"id": "F4", "severity": "critical", "hma_ref": ref("kernel_rule", "secrets")},
			map[string]any{"id": "F5", "severity": "critical", "hma_ref": ref("kernel_rule", "no_shell")},
			map[string]any{"id": "F6", "severity": "critical", "hma_ref": ref("declared_budget", "max_files")},
			map[string]any{"id": "F7", "severity": "critical", "hma_ref": ref("declared_budget", "invented")},
			map[string]any{"id": "F8", "severity": "critical", "hma_ref": ref("approved_criterion", "C1")})
		if len(got) != 8 {
			t.Fatalf("findings %v", got)
		}
		for id, d := range got {
			if d != model.FindingAdvisory {
				t.Errorf("%s = %v, want ADVISORY", id, d)
			}
		}
	})

	// A plan command that fails the no-shell policy is a kernel violation HMA
	// reproduces: a finding naming it is BLOCK whatever its severity, and an
	// unnamed finding stays ADVISORY.
	t.Run("a reproduced kernel rule blocks", func(t *testing.T) {
		got := importFindingsOf(t, newCreditFixtureArgs(t, "/bin/sh", []string{"-c", "true"}, model.CriterionPending),
			map[string]any{"id": "F1", "severity": "low", "hma_ref": ref("kernel_rule", "no_shell")},
			map[string]any{"id": "F2", "severity": "critical"},
			map[string]any{"id": "F3", "severity": "low", "hma_ref": ref("kernel_rule", "authority")})
		if got["U1/F1"] != model.FindingBlock || got["U1/F2"] != model.FindingAdvisory || got["U1/F3"] != model.FindingAdvisory {
			t.Fatalf("dispositions %v", got)
		}
	})

	// The change touches more files than the unit's plan allows (3): an
	// auditor naming that budget gets WAIVABLE, an unrelated finding stays
	// ADVISORY, and HMA records the excess itself. The line budget (100) is
	// not exceeded, so none is raised for it.
	t.Run("a measured budget excess is waivable", func(t *testing.T) {
		creditRepoSetup = func(f *reviewFixture) {
			for i := 0; i < 4; i++ {
				os.WriteFile(filepath.Join(f.root, fmt.Sprintf("extra%d", i)), []byte("x\n"), 0o600)
			}
			gitIn(t, f.root, "add", "-A")
			gitIn(t, f.root, "commit", "-qm", "more files")
			st, err := repostate.Snapshot(context.Background(), f.root, f.base, false)
			if err != nil {
				t.Fatal(err)
			}
			f.head, f.diff = st.Head, st.Diff
		}
		defer func() { creditRepoSetup = nil }()
		got := importFindingsOf(t, newCreditFixture(t, "true"),
			map[string]any{"id": "F1", "severity": "low", "hma_ref": ref("declared_budget", "max_files")},
			map[string]any{"id": "F2", "severity": "critical"},
			map[string]any{"id": "F3", "severity": "critical", "hma_ref": ref("declared_budget", "max_diff_lines")})
		want := map[string]model.FindingDisposition{
			"U1/F1": model.FindingWaivable, "U1/F2": model.FindingAdvisory, "U1/F3": model.FindingAdvisory,
			budgetFindingID(t, got): model.FindingWaivable,
		}
		if len(got) != len(want) {
			t.Fatalf("dispositions %v", got)
		}
		for id, d := range want {
			if got[id] != d {
				t.Errorf("%s = %v, want %v", id, got[id], d)
			}
		}
	})
}

// appendWaiver records an active human waiver for one finding.
func appendWaiver(t *testing.T, cf *creditFixture, finding string) {
	t.Helper()
	s := store.New(cf.store)
	core, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	last := core[len(core)-1]
	r := model.Record{Kind: model.KindWaiver, Version: 1, UnitID: "U1", Findings: engine.ProjectResolutionState(core).Findings,
		Waivers: []model.Waiver{{ID: "w1", Scope: model.ScopeSelector{Kind: model.ScopeFinding, Target: finding}, Active: true, Actor: "human"}},
		Metadata: model.RecordMetadata{RunID: "run", Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
			Timestamp: "2026-09-16T12:20:00Z", Actor: "operator-confirmed:lennon"}}
	if err := s.Append(&r); err != nil {
		t.Fatal(err)
	}
}

// F2: HMA re-raises a held kernel BLOCK on every import, so a later approving
// review that omits it cannot clear it; the unit stays unresolved.
func TestKernelBlockSurvivesApprovingReview(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/bin/sh", []string{"-c", "true"}, model.CriterionPending)
	got := importFindingsOf(t, cf)
	if got["U1/"+hoststate.KernelFindingPrefix+"no_shell"] != model.FindingBlock {
		t.Fatalf("HMA did not raise the held rule: %v", got)
	}
	if err := requireFindingsResolved(cf.store, "run", "U1"); err == nil {
		t.Fatal("a BLOCK finding did not keep the unit unresolved")
	}
	cf.packet("2026-09-16T12:11:00Z", "2026-09-16T12:11:30Z", "2026-09-16T12:16:00Z", "20261016T000000Z-ming-review-3")
	cf.importTS = "2026-09-16T12:12:00Z"
	rp := cf.report("pass", "isolated", nil)
	cf.receipt(rp, nil)
	if err := cf.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := cf.importFindings(cf.latestReview(), rp, "2026-09-16T12:12:30Z"); err != nil {
		t.Fatal(err)
	}
	if cf.coreFindings()["U1/"+hoststate.KernelFindingPrefix+"no_shell"] != model.FindingBlock {
		t.Fatal("an approving review cleared a held kernel BLOCK")
	}
	if err := requireFindingsResolved(cf.store, "run", "U1"); err == nil {
		t.Fatal("approving review resolved a held kernel BLOCK")
	}
	// The review-to-verification gate refuses an approved review while the
	// BLOCK stands.
	if err := cf.gate(); err == nil || err.Error() != hoststate.ReasonFindingsUnresolved {
		t.Fatalf("review gate with a standing BLOCK: %v", err)
	}
}

// F3: the gate. A WAIVABLE finding keeps the unit unresolved until a human
// waiver exists for that exact finding; an unrelated finding does not.
func TestWaivableNeedsHumanWaiver(t *testing.T) {
	creditRepoSetup = func(f *reviewFixture) {
		for i := 0; i < 4; i++ {
			os.WriteFile(filepath.Join(f.root, fmt.Sprintf("extra%d", i)), []byte("x\n"), 0o600)
		}
		gitIn(t, f.root, "add", "-A")
		gitIn(t, f.root, "commit", "-qm", "more files")
		st, err := repostate.Snapshot(context.Background(), f.root, f.base, false)
		if err != nil {
			t.Fatal(err)
		}
		f.head, f.diff = st.Head, st.Diff
	}
	defer func() { creditRepoSetup = nil }()
	cf := newCreditFixture(t, "true")
	got := importFindingsOf(t, cf, map[string]any{"id": "F1", "severity": "critical"})
	id := budgetFindingID(t, got)
	if got[id] != model.FindingWaivable || got["U1/F1"] != model.FindingAdvisory {
		t.Fatalf("dispositions %v", got)
	}
	if err := requireFindingsResolved(cf.store, "run", "U1"); err == nil {
		t.Fatal("an unwaived WAIVABLE finding did not keep the unit unresolved")
	}
	if err := requireFindingsResolved(cf.store, "run", ""); err == nil {
		t.Fatal("closure accepted an unwaived WAIVABLE finding")
	}
	if err := requireFindingsResolved(cf.store, "run", "U2"); err != nil {
		t.Fatalf("another unit's finding blocked this unit: %v", err)
	}
	// A waiver for an ADVISORY finding is not even recordable.
	func() {
		defer func() { recover() }()
		s := store.New(cf.store)
		core, _ := s.Load("run")
		last := core[len(core)-1]
		r := model.Record{Kind: model.KindWaiver, Version: 1, UnitID: "U1", Findings: engine.ProjectResolutionState(core).Findings,
			Waivers: []model.Waiver{{ID: "w0", Scope: model.ScopeSelector{Kind: model.ScopeFinding, Target: "U1/F1"}, Active: true, Actor: "human"}},
			Metadata: model.RecordMetadata{RunID: "run", Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
				Timestamp: "2026-09-16T12:19:00Z", Actor: "operator-confirmed:lennon"}}
		if s.Append(&r) == nil {
			t.Error("a waiver for an ADVISORY finding was recorded")
		}
	}()
	appendWaiver(t, cf, id)
	if err := requireFindingsResolved(cf.store, "run", "U1"); err != nil {
		t.Fatalf("a waived WAIVABLE finding still blocks: %v", err)
	}
	// N2, documented: a waiver is a core record after the review, so the
	// review gate treats that review as stale. The path is a fresh independent
	// review of the same artifact; it re-raises the same finding id, which the
	// waiver already covers.
	cf.packet("2026-09-16T12:21:00Z", "2026-09-16T12:21:30Z", "2026-09-16T12:26:00Z", "20261016T000000Z-ming-review-4")
	cf.importTS = "2026-09-16T12:22:00Z"
	rp := cf.report("pass", "isolated", nil)
	cf.receipt(rp, nil)
	if err := cf.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := cf.importFindings(cf.latestReview(), rp, "2026-09-16T12:22:30Z"); err != nil {
		t.Fatal(err)
	}
	if err := cf.gate(); err != nil {
		t.Fatalf("gate after a waiver and a fresh review of the same artifact: %v", err)
	}
}

// F6: a committed attribute that hides lines from git diff does not hide them
// from the measurement; F7: uncommitted content of a dirty target counts.
func TestMeasureChangeIgnoresAttributesAndCountsDirty(t *testing.T) {
	root, base, _, _ := ciRepo(t)
	big := make([]byte, 0, 4000)
	for i := 0; i < 500; i++ {
		big = append(big, []byte("line\n")...)
	}
	os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("* -diff\n"), 0o600)
	os.WriteFile(filepath.Join(root, "big"), big, 0o600)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-qm", "big")
	files, lines, _, err := measureChange(context.Background(), root, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if files < 2 || lines < 500 {
		t.Fatalf("attributes hid lines: %d files %d lines", files, lines)
	}
	os.WriteFile(filepath.Join(root, "untracked"), []byte("a\nb\n"), 0o600)
	os.WriteFile(filepath.Join(root, "big"), append(big, []byte("more\n")...), 0o600)
	f2, l2, _, err := measureChange(context.Background(), root, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if f2 != files+1 || l2 < lines+3 {
		t.Fatalf("dirty content not counted: %d/%d -> %d/%d", files, lines, f2, l2)
	}
}

// F5: a plan command that merely names a missing tool is not a shell.
func TestShellPolicyViolatedOnlyForShells(t *testing.T) {
	if shellPolicyViolated(model.Check{Executable: "definitely-not-installed-tool"}) {
		t.Error("a missing tool read as a shell")
	}
	if !shellPolicyViolated(model.Check{Executable: "/bin/sh", Argv: []string{"-c", "true"}}) {
		t.Error("sh -c not recognised")
	}
	if !shellPolicyViolated(model.Check{Executable: "env", Argv: []string{"sh"}}) {
		t.Error("env wrapper not recognised")
	}
}

// budgetFindingID is the id of the one HMA-raised budget finding in got.
func budgetFindingID(t *testing.T, got map[string]model.FindingDisposition) string {
	t.Helper()
	id := ""
	for k := range got {
		if strings.HasPrefix(k, "U1/"+hoststate.MeasuredFindingPrefix+"max_files-") {
			id = k
		}
	}
	if id == "" || got[id] != model.FindingWaivable {
		t.Fatalf("no WAIVABLE budget finding in %v", got)
	}
	return id
}

// N1: a waiver covers the budget finding of the artifact the human saw. A new
// artifact is a new finding, so the old waiver does not cover it.
func TestBudgetWaiverDoesNotSurviveNewArtifact(t *testing.T) {
	a := "sha256:aaaaaaaaaaaaaaaaaaaa"
	b := "sha256:bbbbbbbbbbbbbbbbbbbb"
	facts := hoststate.FindingFacts{BudgetExceeded: map[string]bool{"max_files": true}}
	one := hoststate.AddEstablishedFindings(nil, facts, a)
	two := hoststate.AddEstablishedFindings(nil, facts, b)
	again := hoststate.AddEstablishedFindings(nil, facts, a)
	if one[0].ID == two[0].ID {
		t.Fatalf("two artifacts share the finding id %q", one[0].ID)
	}
	if one[0].ID != again[0].ID {
		t.Fatalf("the same artifact changed the finding id: %q vs %q", one[0].ID, again[0].ID)
	}
}

// N3: the live reader counts an untracked symlink as its link text (one line,
// not followed), never opens a FIFO, and counts a regular file's lines.
func TestWorktreeLinesBounded(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "big")
	os.WriteFile(outside, []byte(strings.Repeat("x\n", 5000)), 0o600)
	os.Symlink(outside, filepath.Join(dir, "link"))
	data, mode, _, _, _, err := worktreeBytes(context.Background(), dir, "link")
	if err != nil || mode != "120000" || observedCountLines(data) != 1 {
		t.Errorf("symlink = %q %q %v; want one line of link text", data, mode, err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skip("mkfifo")
	}
	done := make(chan string, 1)
	go func() { _, m, _, _, _, _ := worktreeBytes(context.Background(), dir, "fifo"); done <- m }()
	select {
	case m := <-done:
		if m != "" {
			t.Errorf("fifo mode %q, want none", m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a FIFO blocked the measurement")
	}
	os.WriteFile(filepath.Join(dir, "reg"), []byte("a\nb\nc"), 0o600)
	data, _, _, _, _, err = worktreeBytes(context.Background(), dir, "reg")
	if err != nil || observedCountLines(data) != 3 {
		t.Errorf("regular file counted %d lines, %v; want 3", observedCountLines(data), err)
	}
}

// N4: a human override cannot change the disposition of a finding HMA raised
// from a fact it measured; a report-sourced WAIVABLE can still be overridden.
func TestOverrideRefusedForEstablishedFinding(t *testing.T) {
	fs := []model.Finding{
		{ID: "U1/HMA-BUDGET-max_files-0123", UnitID: "U1", Disposition: model.FindingWaivable},
		{ID: "U1/F1", UnitID: "U1", Disposition: model.FindingWaivable},
	}
	req := func(id string) engine.ResolutionResult {
		return engine.EvaluateResolution(engine.ResolutionRequest{CurrentFindings: fs, UsedNonces: map[string]bool{},
			FindingOverride: &model.FindingDispositionOverride{FindingID: id, Disposition: model.FindingAdvisory}})
	}
	if r := req("U1/HMA-BUDGET-max_files-0123"); r.Decision != engine.DecisionRejected || r.Reason != engine.ReasonResolutionOverrideEstablished {
		t.Errorf("established finding overridden: %+v", r)
	}
	if r := req("U1/F1"); r.Decision == engine.DecisionRejected {
		t.Errorf("report-sourced WAIVABLE no longer overridable: %+v", r)
	}
}

// A modified tracked file in a dirty target counts only its changed lines (one
// added, one deleted), never the whole file as if it were untracked; a new
// untracked file counts its lines. Both the budget measurement and the packet
// manifest must agree.
func TestDirtyMeasureCountsTrackedChangesByDiff(t *testing.T) {
	root := gitRepo(t)
	ten := strings.Repeat("line\n", 10)
	os.WriteFile(filepath.Join(root, "tracked"), []byte(ten), 0o600)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-qm", "ten lines")
	baseOut, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	base := strings.TrimSpace(string(baseOut))
	os.WriteFile(filepath.Join(root, "tracked"), []byte(strings.Replace(ten, "line", "LINE", 1)), 0o600)
	os.WriteFile(filepath.Join(root, "fresh"), []byte("a\nb\nc\n"), 0o600)
	files, lines, _, err := measureChange(context.Background(), root, base, true)
	if err != nil {
		t.Fatal(err)
	}
	// Against the base: "tracked" differs in one line (one added, one deleted)
	// and "fresh" is new with 3 lines. Compute the tracked part from git itself.
	out, _ := exec.Command("git", "-C", root, "diff", "--numstat", "--text", "--no-renames", base, "--").Output()
	want := 3
	for _, row := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(row)
		a, _ := strconv.Atoi(f[0])
		d, _ := strconv.Atoi(f[1])
		want += a + d
	}
	if lines != want {
		t.Fatalf("lines = %d, want %d (tracked changes by diff, untracked by content)", lines, want)
	}
	manifest, err := buildChangedFileManifest(context.Background(), root, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest) != files {
		t.Fatalf("manifest has %d entries, measurement counts %d files", len(manifest), files)
	}
	sum := 0
	for _, f := range manifest {
		sum += f.Added + f.Deleted
	}
	if sum != lines {
		t.Fatalf("manifest lines %d, measurement lines %d", sum, lines)
	}
}
