package engine

import (
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

// RevokedAt returns the effective end of a reservation of unit issued under
// the lease generation whose issuing approval has sequence issuerSeq (Gate A
// v7 §1): the minimum of original and the time of every applicable
// revocation event after the issuer. Applicable events are any approved
// return to Grounding or Acceptance criteria (whatever unit it names); any
// later approved transition of the unit or the whole run out of
// Implementation authorization; a return of a plan dependency to an earlier
// gate or its correction from Independent validation; and a terminal outcome.
// An event's time is the earlier of its approval time and its record time.
// The value is computed from the whole chain, so ingestion order does not
// matter, and adding events can only lower it.
func RevokedAt(records []model.Record, plan model.Plan, unit string, issuerSeq int64, original time.Time) time.Time {
	return revokedAt(records, dependencyClosure(plan, unit), unit, issuerSeq, original, false)
}

// RevokedReviewAt is RevokedAt for a review dispatch issued when the core
// head had sequence afterSeq, under the generation whose plan is plan: the
// unit's own forward moves (into review, verification, validation) do not
// revoke its review; a return of the unit to Implementation authorization
// (a correction generation) or to an earlier gate does.
func RevokedReviewAt(records []model.Record, plan model.Plan, unit string, afterSeq int64, original time.Time) time.Time {
	return revokedAt(records, dependencyClosure(plan, unit), unit, afterSeq, original, true)
}

func revokedAt(records []model.Record, deps map[string]bool, unit string, issuerSeq int64, original time.Time, review bool) time.Time {
	end := original
	lower := func(r model.Record) {
		t, ok := eventTime(r)
		if !ok {
			// An event without a readable time revokes from the issuer
			// on (fail closed).
			t = time.Time{}
		}
		if t.Before(end) {
			end = t
		}
	}
	for _, r := range records {
		if r.Metadata.Sequence <= issuerSeq {
			continue
		}
		if r.Kind == model.KindOutcome && model.ValidTerminalOutcome(r.Outcome) {
			lower(r)
			continue
		}
		if r.Kind != model.KindStageTransition || r.Control != model.ControlApproved || r.Approval == nil {
			continue
		}
		to := r.Approval.ProposedTargetStage
		switch {
		case to == model.StageGrounding || to == model.StageAcceptanceCriteria:
			lower(r)
		case r.UnitID == unit || r.UnitID == "":
			if review && (isEarlierGate(to) || to == model.StageImplementationAuthorization) || !review && to != model.StageImplementationAuthorization {
				lower(r)
			}
		case deps[r.UnitID]:
			if isEarlierGate(to) || r.Approval.CurrentStage == model.StageIndependentValidation && to == model.StageImplementationAuthorization {
				lower(r)
			}
		}
	}
	return end
}

func eventTime(r model.Record) (time.Time, bool) {
	m, err := time.Parse(time.RFC3339Nano, r.Metadata.Timestamp)
	if err != nil {
		return time.Time{}, false
	}
	if r.Approval != nil {
		if a, err := time.Parse(time.RFC3339Nano, r.Approval.Timestamp); err == nil && a.Before(m) {
			return a, true
		}
	}
	return m, true
}
