package main

import (
	"path/filepath"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func (f *reviewFixture) latestReview() string {
	f.t.Helper()
	h, _ := hoststate.NewStore(f.store).Load("run")
	return h[len(h)-1].HeadAnchor
}

func (f *reviewFixture) importFindings(reviewDigest, reportPath, ts string) error {
	f.t.Helper()
	input := filepath.Join(f.t.TempDir(), "findings.json")
	writeJSON(f.t, input, hostFindingsImportInput{hostEnvelope: fxEnv(ts), ReviewRecordDigest: reviewDigest, ReportPath: reportPath})
	return run([]string{"host", "findings-import", "--input", input, "--store", f.store})
}

func (f *reviewFixture) coreFindings() map[string]model.FindingDisposition {
	f.t.Helper()
	core, err := store.New(f.store).Load("run")
	if err != nil {
		f.t.Fatal(err)
	}
	out := map[string]model.FindingDisposition{}
	for _, x := range engine.ProjectResolutionState(core).Findings {
		out[x.ID] = x.Disposition
	}
	return out
}

// withFindings sets the report's findings, completing each with the fields
// the report contract requires unless the test set them.
func withFindings(fs ...map[string]any) func(map[string]any) {
	return func(r map[string]any) {
		list := make([]any, len(fs))
		for i := range fs {
			f := map[string]any{"title": "t", "diagnosis": "d", "kind": "defect", "mode": "code", "severity": "medium", "confidence": "inferred",
				"evidence": []any{}, "fix": map[string]any{"action": "a", "why": "w", "tradeoffs": "t", "alternative": "x", "effort": "small", "status": "proposed",
					"verification": []any{map[string]any{"method": "m", "expected": "e"}}}}
			for k, v := range fs[i] {
				f[k] = v
			}
			list[i] = f
		}
		r["findings"] = list
	}
}

// Audit failure -> linked remediation -> re-verification, end to end.
func TestGateCAuditFailureRemediationReverification(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)

	// Round 1: the review asks for changes. Severity never sets disposition,
	// and an unreproduced citation creates no blocker: every imported
	// finding is ADVISORY. The fix-required verdict is what blocks the gate.
	rp1 := f.report("fix-required", "isolated", withFindings(
		map[string]any{"id": "F1", "severity": "low", "hma_ref": map[string]any{"kind": "kernel_rule", "id": "K-APPROVAL-BINDING"}},
		map[string]any{"id": "F2", "severity": "critical"},
		map[string]any{"id": "F3", "severity": "high", "hma_ref": map[string]any{"kind": "declared_budget", "id": "B-COMPLEXITY"}},
	))
	f.receipt(rp1, nil)
	if err := f.importReport(rp1); err != nil {
		t.Fatal(err)
	}
	r1 := f.latestReview()
	if err := f.importFindings(r1, rp1, "2026-09-16T12:06:30Z"); err != nil {
		t.Fatal(err)
	}
	got := f.coreFindings()
	if len(got) != 3 || got["U1/F1"] != model.FindingAdvisory || got["U1/F2"] != model.FindingAdvisory || got["U1/F3"] != model.FindingAdvisory {
		t.Fatalf("round 1 dispositions %v", got)
	}
	if f.gate() == nil {
		t.Fatal("review gate passed after fix-required")
	}

	// Remediation: a new implementation dispatch. The round 1 review no
	// longer reviews the current implementation.
	f.implement("2026-09-16T12:07:00Z", "2026-09-16T12:08:00Z", "2026-09-16T12:12:00Z")
	if err := f.importFindings(r1, rp1, "2026-09-16T12:08:30Z"); err == nil {
		t.Fatal("stale review findings imported after remediation")
	}
	if f.gate() == nil {
		t.Fatal("review gate passed before re-review")
	}

	// Round 2, a fix-required review that omits the earlier findings; the
	// gate stays blocked by the verdict.
	f.packet("2026-09-16T12:09:00Z", "2026-09-16T12:09:30Z", "2026-09-16T12:14:00Z", "20261016T000000Z-ming-review-2")
	f.importTS = "2026-09-16T12:10:00Z"
	rp2 := f.report("fix-required", "isolated", withFindings(map[string]any{"id": "F4", "severity": "medium"}))
	f.receipt(rp2, nil)
	if err := f.importReport(rp2); err != nil {
		t.Fatal(err)
	}
	if err := f.importFindings(f.latestReview(), rp2, "2026-09-16T12:10:30Z"); err != nil {
		t.Fatal(err)
	}
	got = f.coreFindings()
	if got["U1/F4"] != model.FindingAdvisory {
		t.Fatalf("round 2 findings %v", got)
	}
	if f.gate() == nil {
		t.Fatal("review gate passed after second fix-required")
	}

	// Round 3, re-verification: a fresh packet and a passing report with no
	// findings. The empty snapshot clears the findings.
	f.packet("2026-09-16T12:11:00Z", "2026-09-16T12:11:30Z", "2026-09-16T12:16:00Z", "20261016T000000Z-ming-review-3")
	f.importTS = "2026-09-16T12:12:00Z"
	rp3 := f.report("pass", "isolated", nil)
	f.receipt(rp3, nil)
	if err := f.importReport(rp3); err != nil {
		t.Fatal(err)
	}
	if err := f.importFindings(f.latestReview(), rp3, "2026-09-16T12:12:30Z"); err != nil {
		t.Fatal(err)
	}
	if got = f.coreFindings(); len(got) != 0 {
		t.Fatalf("approving empty snapshot left findings %v", got)
	}
	if err := f.gate(); err != nil {
		t.Fatalf("review gate after re-verification: %v", err)
	}

	// No import path credits an acceptance criterion.
	core, _ := store.New(f.store).Load("run")
	for _, r := range core {
		if r.Kind == model.KindFinding && len(r.Criteria) != 0 {
			t.Fatal("findings import wrote criteria")
		}
	}
	for _, c := range engine.ProjectResolutionState(core).Criteria {
		if c.Disposition == model.CriterionPassed {
			t.Fatalf("criterion %s credited by import", c.ID)
		}
	}
}

// The findings report must be the exact report the review imported, and a
// host-written review cannot feed a snapshot.
func TestGateCFindingsImportBindsImportedReport(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("fix-required", "isolated", withFindings(map[string]any{"id": "F1", "hma_ref": map[string]any{"kind": "kernel_rule", "id": "K"}}))
	f.receipt(rp, nil)
	if err := f.importReport(rp); err != nil {
		t.Fatal(err)
	}
	other := f.report("pass", "isolated", nil)
	if err := f.importFindings(f.latestReview(), other, "2026-09-16T12:06:30Z"); err == nil || err.Error() != hoststate.ReasonReviewReportMismatch {
		t.Fatalf("other report: %v", err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	if err := f.importFindings(h[len(h)-2].HeadAnchor, rp, "2026-09-16T12:06:30Z"); err == nil || err.Error() != hoststate.ReasonReviewMissing {
		t.Fatalf("non-review record: %v", err)
	}
	if len(f.coreFindings()) != 0 {
		t.Fatal("refused import wrote findings")
	}
}

// A record without the flag keeps today's behaviour: no findings, no change.
func TestGateCRecordWithoutFlagNoChange(t *testing.T) {
	recs := []model.Record{
		{Kind: model.KindFinding, Findings: []model.Finding{{ID: "F1", Disposition: model.FindingBlock}}},
		{Kind: model.KindFinding},
	}
	if got := engine.ProjectResolutionState(recs).Findings; len(got) != 1 {
		t.Fatalf("unflagged empty record changed findings: %v", got)
	}
	recs = append(recs, model.Record{Kind: model.KindFinding, FindingsSnapshot: true, FindingsSource: &model.FindingsSource{ReviewRecordDigest: "r", ReportDigest: "p"}})
	if got := engine.ProjectResolutionState(recs).Findings; len(got) != 0 {
		t.Fatalf("empty snapshot kept findings: %v", got)
	}
}

// Only snapshots from the same review may follow it in the core chain; any
// other core record still makes the review stale.
func TestGateCOtherCoreRecordAfterReviewStillStale(t *testing.T) {
	for _, tc := range []struct {
		name string
		rec  func(reviewDigest string) model.Record
	}{
		{"plain-finding", func(string) model.Record {
			return model.Record{Kind: model.KindFinding, Findings: []model.Finding{{ID: "X", Disposition: model.FindingAdvisory, Advisory: model.AdvisoryPark}}}
		}},
		{"snapshot-from-other-review", func(string) model.Record {
			return model.Record{Kind: model.KindFinding, FindingsSnapshot: true, FindingsSource: &model.FindingsSource{ReviewRecordDigest: "other-review", ReportDigest: "sha256:x"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			rp := f.report("pass", "isolated", nil)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err != nil {
				t.Fatal(err)
			}
			if err := f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z"); err != nil {
				t.Fatal(err)
			}
			if err := f.gate(); err != nil {
				t.Fatalf("own snapshot made review stale: %v", err)
			}
			s := store.New(f.store)
			core, _ := s.Load("run")
			last := core[len(core)-1]
			r := tc.rec(f.latestReview())
			r.Version = 1
			r.Metadata = model.RecordMetadata{RunID: "run", Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor, Timestamp: "2026-09-16T12:07:00Z", Actor: "host"}
			if err := s.Append(&r); err != nil {
				t.Fatal(err)
			}
			if err := f.gate(); err == nil || err.Error() != hoststate.ReasonCoreBindingStale {
				t.Fatalf("err = %v, want %s", err, hoststate.ReasonCoreBindingStale)
			}
		})
	}
}
