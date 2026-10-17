package main

import (
	"os"
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
)

// Argie review: an earlier or already-answered packet cannot be replayed to
// put an approving review at the chain tail.
func TestGateCReplayOfEarlierPacketRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp1 := f.report("pass", "isolated", nil)
	f.receipt(rp1, nil)
	if err := f.importReport(rp1); err != nil {
		t.Fatal(err)
	}
	// Answered once: a second import of the same packet is refused.
	f.importTS = "2026-09-16T12:06:10Z"
	if err := f.importReport(rp1); err == nil || err.Error() != hoststate.ReasonReviewPacketMismatch {
		t.Fatalf("second import of answered packet: %v", err)
	}
	// A newer packet supersedes P1; P1 cannot be imported again.
	old := *f
	f.packet("2026-09-16T12:06:20Z", "2026-09-16T12:06:30Z", "2026-09-16T12:11:00Z", "20261016T000000Z-ming-review-2")
	rp2 := f.report("fix-required", "isolated", nil)
	f.receipt(rp2, nil)
	f.importTS = "2026-09-16T12:07:00Z"
	if err := f.importReport(rp2); err != nil {
		t.Fatal(err)
	}
	old.importTS = "2026-09-16T12:07:30Z"
	old.receipt(rp1, nil)
	if err := old.importReport(rp1); err == nil || err.Error() != hoststate.ReasonReviewPacketMismatch {
		t.Fatalf("replay of superseded packet: %v", err)
	}
	if f.gate() == nil {
		t.Fatal("gate passed after replay attempt")
	}
}

// Argie review: once a packet exists, a host-written review dispatch and
// review cannot satisfy the gate.
func TestGateCLegacyReviewAfterPacketRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	d := *h[len(h)-1].Dispatch
	// Reuse the packet's preflight for a host-written dispatch.
	d.Packet, d.ReviewPacketDigest = nil, "host-packet"
	input := filepath.Join(t.TempDir(), "d.json")
	writeJSON(t, input, hostDispatchInput{hostEnvelope: fxEnv("2026-09-16T12:04:30Z"), Dispatch: d})
	if err := run([]string{"host", "dispatch", "--input", input, "--store", f.store}); err == nil || err.Error() != hoststate.ReasonReviewImportRequired {
		t.Fatalf("legacy dispatch after packet: %v", err)
	}
}

// Argie review: a non-approving review never weakens a prior disposition
// with the same id, and one unit's snapshot never touches another unit's
// findings.
func TestGateCSnapshotNoDowngradeNoCrossUnit(t *testing.T) {
	prior := []model.Finding{
		{ID: "U1/F1", UnitID: "U1", Disposition: model.FindingBlock},
		{ID: "U1/F2", UnitID: "U1", Disposition: model.FindingWaivable},
		{ID: "U2/F9", UnitID: "U2", Disposition: model.FindingBlock},
		// A3: a unit named with the prefix of another is a different owner.
		{ID: "U1/sub/F1", UnitID: "U1/sub", Disposition: model.FindingBlock},
	}
	imported := []model.Finding{{ID: "F1", Disposition: model.FindingAdvisory, Advisory: model.AdvisoryPark}}
	got := map[string]model.FindingDisposition{}
	for _, f := range hoststate.SnapshotFindings(prior, "U1", imported, hoststate.ReviewRequestChanges) {
		got[f.ID] = f.Disposition
	}
	if got["U1/F1"] != model.FindingBlock || got["U1/F2"] != model.FindingWaivable || got["U2/F9"] != model.FindingBlock {
		t.Fatalf("non-approving snapshot weakened findings: %v", got)
	}
	got = map[string]model.FindingDisposition{}
	for _, f := range hoststate.SnapshotFindings(prior, "U1", nil, hoststate.ReviewApprove) {
		got[f.ID] = f.Disposition
	}
	if len(got) != 2 || got["U2/F9"] != model.FindingBlock || got["U1/sub/F1"] != model.FindingBlock {
		t.Fatalf("approving U1 snapshot touched U2: %v", got)
	}
}

// Argie review: a pass report that carries a blocking finding escalates; a
// report with trailing JSON or invalid findings is refused.
func TestGateCContradictoryOrMalformedReport(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", withFindings(map[string]any{"id": "F1", "hma_ref": map[string]any{"kind": "kernel_rule", "id": "K"}}))
	f.receipt(rp, nil)
	if err := f.importReport(rp); err != nil {
		t.Fatal(err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	if v := h[len(h)-1].Review.Verdict; v != hoststate.ReviewEscalate {
		t.Fatalf("pass with blocker: %s", v)
	}
	for name, edit := range map[string]func(map[string]any){
		"duplicate-finding-id": withFindings(map[string]any{"id": "F1"}, map[string]any{"id": "F1"}),
		"finding-without-id":   withFindings(map[string]any{"severity": "low"}),
		"no-findings":          func(r map[string]any) { delete(r, "findings") },
	} {
		g := newReviewFixture(t, hoststate.IssuePerChange, false)
		rp := g.report("pass", "isolated", edit)
		g.receipt(rp, nil)
		if err := g.importReport(rp); err == nil || err.Error() != hoststate.ReasonFindingsImportInvalid {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestGateCTrailingJSONReportRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	appendBytes(t, rp, []byte(`{"verdict":"fix-required"}`))
	f.receipt(rp, nil)
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewReportMismatch {
		t.Fatalf("trailing JSON: %v", err)
	}
}

func appendBytes(t *testing.T, path string, b []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		t.Fatal(err)
	}
}
