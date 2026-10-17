package main

import (
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
)

// R3-01: a newer packet supersedes an earlier approving review: the old
// approval neither passes the gate nor feeds a findings snapshot.
func TestGateCR4NewerPacketSupersedesOlderApproval(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, nil)
	if err := f.importReport(rp); err != nil {
		t.Fatal(err)
	}
	review := f.latestReview()
	f.packet("2026-09-16T12:06:10Z", "2026-09-16T12:06:20Z", "2026-09-16T12:11:00Z", "run-newer")
	if err := f.importFindings(review, rp, "2026-09-16T12:06:30Z"); err == nil || err.Error() != hoststate.ReasonReviewPacketMismatch {
		t.Fatalf("old review fed a snapshot: %v", err)
	}
	if err := f.gate(); err == nil {
		t.Fatal("old approval passed the gate while a newer packet is pending")
	}
}
