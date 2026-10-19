package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

func revocationPlan(f *leaseFixture) model.Plan {
	var p model.Plan
	json.Unmarshal(f.plan, &p)
	return p
}

func TestValidUntilMinOfTwoRevocations(t *testing.T) {
	expiry := leaseT0.Add(60 * time.Minute)
	for _, order := range []string{"chronological", "reverse"} {
		t.Run(order, func(t *testing.T) {
			f := newLeaseFixture(t)
			f.authorize("U3", leaseT0)
			issuer := f.records[len(f.records)-1].Metadata.Sequence
			first, second := leaseT0.Add(10*time.Minute), leaseT0.Add(20*time.Minute)
			if order == "reverse" {
				// The later-recorded event carries the earlier time.
				first, second = second, first
			}
			f.approve("U2", model.StageImplementationAuthorization, model.StagePlanning, first) // dependency rewind
			f.approve("", model.StageVerification, model.StagePlanning, second)                 // whole-run rewind
			if got := RevokedAt(f.records, revocationPlan(f), "U3", issuer, expiry); !got.Equal(leaseT0.Add(10 * time.Minute)) {
				t.Fatalf("valid_until %s", got)
			}
		})
	}
}

func TestExpiryThenLaterRevocation(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	issuer := f.records[len(f.records)-1].Metadata.Sequence
	expiry := leaseT0.Add(60 * time.Minute)
	f.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0.Add(90*time.Minute))
	if got := RevokedAt(f.records, revocationPlan(f), "U1", issuer, expiry); !got.Equal(expiry) {
		t.Fatalf("valid_until %s, want expiry", got)
	}
}

// An unrelated unit's forward move does not revoke; a terminal outcome does.
func TestRevocationScope(t *testing.T) {
	f := newLeaseFixture(t)
	f.authorize("U1", leaseT0)
	issuer := f.records[len(f.records)-1].Metadata.Sequence
	expiry := leaseT0.Add(60 * time.Minute)
	f.approve("U4", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0.Add(5*time.Minute))
	if got := RevokedAt(f.records, revocationPlan(f), "U1", issuer, expiry); !got.Equal(expiry) {
		t.Fatalf("unrelated move revoked: %s", got)
	}
	f.seq++
	f.records = append(f.records, model.Record{Kind: model.KindOutcome, Outcome: model.OutcomeAborted, Version: 1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: f.seq, HeadAnchor: "end", Timestamp: leaseT0.Add(7 * time.Minute).Format(time.RFC3339Nano)}})
	if got := RevokedAt(f.records, revocationPlan(f), "U1", issuer, expiry); !got.Equal(leaseT0.Add(7 * time.Minute)) {
		t.Fatalf("terminal outcome: %s", got)
	}
}

// Liz D4 F03: a review dispatch is revoked by the unit's return to an
// earlier gate, a dependency rewind, or a whole-run rewind, not by the
// unit's own forward move.
func TestRevokedReviewAt(t *testing.T) {
	expiry := leaseT0.Add(60 * time.Minute)
	f := newLeaseFixture(t)
	f.authorize("U3", leaseT0)
	after := f.records[len(f.records)-1].Metadata.Sequence
	plan := revocationPlan(f)
	f.approve("U3", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0.Add(time.Minute))
	if got := RevokedReviewAt(f.records, plan, "U3", after, expiry); !got.Equal(expiry) {
		t.Fatalf("forward move revoked the review: %s", got)
	}
	f.approve("U4", model.StageImplementationAuthorization, model.StagePlanning, leaseT0.Add(2*time.Minute))
	if got := RevokedReviewAt(f.records, plan, "U3", after, expiry); !got.Equal(expiry) {
		t.Fatalf("unrelated unit revoked: %s", got)
	}
	f.approve("U2", model.StageImplementationAuthorization, model.StageRouteSelection, leaseT0.Add(3*time.Minute))
	if got := RevokedReviewAt(f.records, plan, "U3", after, expiry); !got.Equal(leaseT0.Add(3 * time.Minute)) {
		t.Fatalf("dependency rewind: %s", got)
	}
	g := newLeaseFixture(t)
	g.authorize("U1", leaseT0)
	a := g.records[len(g.records)-1].Metadata.Sequence
	gp := revocationPlan(g)
	g.approve("U1", model.StageImplementationReview, model.StagePlanning, leaseT0.Add(4*time.Minute))
	if got := RevokedReviewAt(g.records, gp, "U1", a, expiry); !got.Equal(leaseT0.Add(4 * time.Minute)) {
		t.Fatalf("own rewind: %s", got)
	}
	// Liz D4 r2 F-01: a correction from Independent validation opens a new
	// generation and revokes the old review.
	h := newLeaseFixture(t)
	h.authorize("U1", leaseT0)
	b := h.records[len(h.records)-1].Metadata.Sequence
	hp := revocationPlan(h)
	h.approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, leaseT0.Add(time.Minute))
	h.approve("U1", model.StageIndependentValidation, model.StageImplementationAuthorization, leaseT0.Add(5*time.Minute))
	if got := RevokedReviewAt(h.records, hp, "U1", b, expiry); !got.Equal(leaseT0.Add(5 * time.Minute)) {
		t.Fatalf("correction generation: %s", got)
	}
}
