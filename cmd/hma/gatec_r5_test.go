package main

import (
	"os"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
)

// appendKey appends a raw "key":value member to a JSON object file, after
// every other member, so Go's last-match-wins case folding would prefer it.
func appendKey(t *testing.T, path, member string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b[:len(b)-1], []byte(","+member+"}")...)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Liz final2 C-F16: reports are read with exact keys, as the Python report
// validator reads them. A case-variant "Verdict":"pass" after a valid
// fix-required verdict must not approve or clear a blocker.
func TestGateCR5CaseVariantVerdictIgnored(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	f.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "EXISTING-BLOCK", Disposition: model.FindingBlock}}})
	f.packet("2026-09-16T12:05:10Z", "2026-09-16T12:05:20Z", "2026-09-16T12:10:00Z", "r5-case-verdict")
	rp := f.report("fix-required", "isolated", func(r map[string]any) {
		r["requirements"] = []any{map[string]any{"id": "R1", "status": "fail", "evidence_refs": []any{"failed-test"}}}
	})
	appendKey(t, rp, `"Verdict":"pass"`)
	f.receipt(rp, nil)
	if err := f.importReport(rp); err != nil {
		t.Fatalf("fix-required report must import: %v", err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	if v := h[len(h)-1].Review.Verdict; v != hoststate.ReviewRequestChanges {
		t.Fatalf("verdict = %s, want REQUEST_CHANGES", v)
	}
	_ = f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z")
	if f.coreFindings()["EXISTING-BLOCK"] != model.FindingBlock || f.gate() == nil {
		t.Fatal("case-variant verdict cleared a blocker or opened the gate")
	}
}

// A Unicode case-fold variant ("iſolation", long s) must not stand in for a
// final audit's isolation.
func TestGateCR5FoldVariantIsolationIgnored(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssueFinal, false)
	rp := f.report("pass", "partial", nil)
	appendKey(t, rp, `"iſolation":"isolated"`)
	f.receipt(rp, nil)
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewIsolationInsufficient {
		t.Fatalf("err = %v, want %s", err, hoststate.ReasonReviewIsolationInsufficient)
	}
}

// Receipts are read with exact keys too: an "EXIT_CODE":0 after a failing
// exit_code does not make the run succeed.
func TestGateCR5CaseVariantReceiptIgnored(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, func(r map[string]any) { r["exit_code"] = 1 })
	appendKey(t, f.ledger+"/runs/"+f.runID+"/receipt.json", `"EXIT_CODE":0`)
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewReceiptMismatch {
		t.Fatalf("err = %v, want %s", err, hoststate.ReasonReviewReceiptMismatch)
	}
}
