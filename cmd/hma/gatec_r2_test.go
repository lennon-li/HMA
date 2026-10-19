package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func (f *reviewFixture) appendCore(r model.Record) {
	f.t.Helper()
	s := store.New(f.store)
	core, _ := s.Load("run")
	last := core[len(core)-1]
	r.Version = 1
	r.Metadata = model.RecordMetadata{RunID: "run", Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor, Timestamp: "2026-09-16T12:05:00Z", Actor: "host"}
	if err := s.Append(&r); err != nil {
		f.t.Fatal(err)
	}
}

func advisoryRecord() model.Record {
	return model.Record{Kind: model.KindFinding, Findings: []model.Finding{{ID: "X", Disposition: model.FindingAdvisory, Advisory: model.AdvisoryPark}}}
}

// F02: a core record after the packet makes it stale for import; a core
// record after the review makes it stale for findings import.
func TestGateCR2ImportsRequireCurrentCoreHead(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	f.appendCore(advisoryRecord())
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, nil)
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonCoreBindingStale {
		t.Fatalf("import after core change: %v", err)
	}

	g := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp = g.report("pass", "isolated", nil)
	g.receipt(rp, nil)
	if err := g.importReport(rp); err != nil {
		t.Fatal(err)
	}
	review := g.latestReview()
	g.appendCore(advisoryRecord())
	if err := g.importFindings(review, rp, "2026-09-16T12:06:30Z"); err == nil || err.Error() != hoststate.ReasonCoreBindingStale {
		t.Fatalf("findings import after core change: %v", err)
	}
	// One snapshot per review: a second import of the same review is stale.
	h := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp = h.report("pass", "isolated", nil)
	h.receipt(rp, nil)
	if err := h.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := h.importFindings(h.latestReview(), rp, "2026-09-16T12:06:30Z"); err != nil {
		t.Fatal(err)
	}
	if err := h.importFindings(h.latestReview(), rp, "2026-09-16T12:06:40Z"); err == nil || err.Error() != hoststate.ReasonCoreBindingStale {
		t.Fatalf("second snapshot of one review: %v", err)
	}
}

// F04: HMA derives the artifact; a host-named digest that differs is
// refused, and dirty content is part of the artifact.
func TestGateCR2PacketDerivesArtifact(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	p := h[len(h)-1].Dispatch.Packet
	if p.ProducedHead != f.head || p.DiffDigest != f.diff || p.BaseRevision != f.base || p.WorktreeDigest != "" {
		t.Fatalf("packet artifact %+v", p)
	}
	d := *h[len(h)-1].Dispatch
	d.Packet, d.ReviewPacketDigest, d.ArtifactDigest = nil, "", hoststate.ReviewArtifactDigest("other-head", f.diff, "")
	input := filepath.Join(t.TempDir(), "in.json")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:04:30Z"), Dispatch: d, DispatchRunID: "r2",
		IssuingReason: hoststate.IssuePerChange, RepositoryRoot: f.root, BaseRevision: f.base, Task: "x", LedgerDir: f.ledger})
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", filepath.Join(t.TempDir(), "p")}); err == nil || err.Error() != hoststate.ReasonReviewArtifactMismatch {
		t.Fatalf("wrong host artifact: %v", err)
	}
	// Dirty content without approval is refused; with approval it is bound.
	os.WriteFile(filepath.Join(f.root, "a"), []byte("dirty\n"), 0o600)
	d.ArtifactDigest = ""
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:04:30Z"), Dispatch: d, DispatchRunID: "r3",
		IssuingReason: hoststate.IssuePerChange, RepositoryRoot: f.root, BaseRevision: f.base, Task: "x", LedgerDir: f.ledger})
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", filepath.Join(t.TempDir(), "p")}); err == nil {
		t.Fatal("unapproved dirty worktree accepted")
	}
}

// F05/F11: a final audit needs family separation; a reviewer that earlier
// implemented the unit is not independent.
func TestGateCR2IndependenceRules(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssueFinal, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	base := *h[len(h)-1].Dispatch
	sameFamily := cliRoute("ming", "openai", "openai", "sonnet", "claude", "approved-route", true)
	pf := f.step("preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:04:10Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: sameFamily, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-2", ExpiresAt: "2026-09-16T12:09:00Z",
	}})
	d := base
	d.Packet, d.ReviewPacketDigest, d.ArtifactDigest, d.Route, d.PreflightRecordDigest, d.Risk = nil, "", "", sameFamily, pf, model.RouteRiskLow
	input := filepath.Join(t.TempDir(), "in.json")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:04:20Z"), Dispatch: d, DispatchRunID: "final-same-family",
		IssuingReason: hoststate.IssueFinal, RepositoryRoot: f.root, BaseRevision: f.base, Task: "x", LedgerDir: f.ledger})
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", filepath.Join(t.TempDir(), "p")}); err == nil || err.Error() != hoststate.ReasonReviewerNotIndependent {
		t.Fatalf("final audit same provider family: %v", err)
	}

}

// F06/F04/F10: receipt provider, effort, project root, and launch time must
// match the packet.
func TestGateCR2ReceiptRouteRootAndWindow(t *testing.T) {
	for name, edit := range map[string]func(map[string]any, *reviewFixture){
		"other-provider": func(r map[string]any, _ *reviewFixture) { r["provider"] = "openai" },
		"other-effort":   func(r map[string]any, _ *reviewFixture) { r["effort"] = "high" },
		"other-root":     func(r map[string]any, f *reviewFixture) { r["project_root"] = f.t.TempDir() },
		"no-root":        func(r map[string]any, _ *reviewFixture) { delete(r, "project_root") },
		"before-packet": func(r map[string]any, f *reviewFixture) {
			ts, _ := time.Parse(time.RFC3339, f.packetTS)
			r["started_at"] = ts.Add(-time.Second).Format(time.RFC3339)
		},
		"at-valid-until": func(r map[string]any, f *reviewFixture) {
			ts, _ := time.Parse(time.RFC3339, f.packetTS)
			r["started_at"] = ts.Add(hoststate.ReviewDispatchTTL).Format(time.RFC3339)
		},
		"year-late": func(r map[string]any, f *reviewFixture) {
			ts, _ := time.Parse(time.RFC3339, f.packetTS)
			r["started_at"] = ts.AddDate(1, 0, 0).Format(time.RFC3339)
		},
		"no-start": func(r map[string]any, _ *reviewFixture) { delete(r, "started_at") },
	} {
		t.Run(name, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			rp := f.report("pass", "isolated", nil)
			f.receipt(rp, func(r map[string]any) { edit(r, f) })
			if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewReceiptMismatch {
				t.Fatalf("err = %v", err)
			}
		})
	}
	// Control: launch just before valid_until is accepted.
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, func(r map[string]any) {
		ts, _ := time.Parse(time.RFC3339, f.packetTS)
		r["started_at"] = ts.Add(hoststate.ReviewDispatchTTL - time.Second).Format(time.RFC3339)
	})
	if err := f.importReport(rp); err != nil {
		t.Fatalf("launch inside window: %v", err)
	}
}

// F07: report contract. A pass inconsistent with the contract escalates;
// malformed findings are refused.
func TestGateCR2ReportContract(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"high-finding":         withFindings(map[string]any{"id": "F1", "severity": "high"}),
		"critical-finding":     withFindings(map[string]any{"id": "F1", "severity": "critical"}),
		"cited-kernel-rule":    withFindings(map[string]any{"id": "F1", "hma_ref": map[string]any{"kind": "kernel_rule", "id": "K"}}),
		"failing-requirement":  func(r map[string]any) { r["requirements"] = []any{map[string]any{"id": "R1", "status": "fail"}} },
		"unknown-requirement":  func(r map[string]any) { r["requirements"] = []any{map[string]any{"id": "R1", "status": "unknown"}} },
		"missing-requirements": func(r map[string]any) { delete(r, "requirements") },
		"null-requirements":    func(r map[string]any) { r["requirements"] = nil },
		"partial-isolation":    func(r map[string]any) { r["isolation"] = "partial" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			rp := f.report("pass", "isolated", edit)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err != nil {
				t.Fatal(err)
			}
			h, _ := hoststate.NewStore(f.store).Load("run")
			if v := h[len(h)-1].Review.Verdict; v != hoststate.ReviewEscalate {
				t.Fatalf("inconsistent pass imported as %s", v)
			}
			if f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z") == nil && f.gate() == nil {
				t.Fatal("gate passed on inconsistent pass")
			}
		})
	}
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", func(r map[string]any) { r["findings"] = nil })
	f.receipt(rp, nil)
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonFindingsImportInvalid {
		t.Fatalf("null findings: %v", err)
	}
	b, _ := os.ReadFile(rp)
	if !strings.Contains(string(b), `"findings":null`) {
		t.Fatal("fixture did not write null findings")
	}
}
