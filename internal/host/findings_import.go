package host

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/lennon-li/HMA/internal/model"
)

const ReasonFindingsImportInvalid = "HOST_FINDINGS_IMPORT_INVALID"

// auditFinding is the part of an independent-audit finding import reads.
// Severity is deliberately not read.
type auditFinding struct {
	ID string `json:"id"`
}

// MapAuditFindings applies the import disposition rule (Gate A v3 §4.1).
// Severity never sets disposition. BLOCK needs a kernel-rule violation HMA
// reproduces, and WAIVABLE a measured excess over a declared budget; HMA has
// neither a kernel-rule registry nor plan budgets yet (Gate D/E), so every
// imported finding is ADVISORY (PARK, for a human to dispose) and a citation
// cannot create an unwaivable blocker by label alone. A pass that carries a
// citation is escalated at review import instead. Findings must be a JSON
// array of objects with unique non-empty ids.
func MapAuditFindings(raw json.RawMessage) ([]model.Finding, error) {
	var in []auditFinding
	if len(raw) == 0 || raw[0] != '[' {
		return nil, errors.New(ReasonFindingsImportInvalid)
	}
	if err := UnmarshalExact(raw, &in); err != nil {
		return nil, errors.New(ReasonFindingsImportInvalid)
	}
	seen := map[string]bool{}
	out := make([]model.Finding, 0, len(in))
	for _, f := range in {
		// A source id may not contain "/": unit "U1" with "sub/F" would
		// otherwise collide with unit "U1/sub" with "F".
		if f.ID == "" || strings.Contains(f.ID, "/") || seen[f.ID] {
			return nil, errors.New(ReasonFindingsImportInvalid)
		}
		seen[f.ID] = true
		out = append(out, model.Finding{ID: f.ID, Disposition: model.FindingAdvisory, Advisory: model.AdvisoryPark})
	}
	return out, nil
}

// UnitFindingID namespaces an imported finding id by unit, so ids from
// different units never collide in the run-wide list. Ownership is the
// finding's UnitID field, never this prefix.
func UnitFindingID(unit, id string) string { return unit + "/" + id }

func strength(d model.FindingDisposition) int {
	switch d {
	case model.FindingBlock:
		return 2
	case model.FindingWaivable:
		return 1
	}
	return 0
}

// SnapshotFindings is the run-wide findings snapshot an imported review of
// unit produces. Findings owned by other units, and unowned findings, are
// kept unchanged. For this unit, only an approving review replaces its
// findings; any other verdict keeps every earlier BLOCK and WAIVABLE finding,
// and an imported finding with the same id never weakens an earlier
// disposition, so a review asking for changes can never clear or downgrade a
// blocker.
func SnapshotFindings(prior []model.Finding, unit string, imported []model.Finding, verdict ReviewVerdict) []model.Finding {
	var out []model.Finding
	priorOwn := map[string]model.Finding{}
	for _, f := range prior {
		if f.UnitID == unit {
			priorOwn[f.ID] = f
		} else {
			out = append(out, f)
		}
	}
	have := map[string]bool{}
	for _, f := range imported {
		f.ID, f.UnitID = UnitFindingID(unit, f.ID), unit
		if p, ok := priorOwn[f.ID]; ok && verdict != ReviewApprove && strength(p.Disposition) > strength(f.Disposition) {
			f = p
		}
		have[f.ID] = true
		out = append(out, f)
	}
	if verdict != ReviewApprove {
		for _, f := range prior {
			if f.UnitID == unit && !have[f.ID] && strength(f.Disposition) > 0 {
				out = append(out, f)
			}
		}
	}
	return out
}

// AssignLegacyOwners gives each unowned current finding the unit of the core
// record that last set it, when that record names exactly one unit. Findings
// whose source names no unit stay unowned: no unit's review can clear them,
// so they stay in force until a human decides their ownership.
func AssignLegacyOwners(core []model.Record, current []model.Finding) []model.Finding {
	// Only records written before snapshots existed are ownership sources:
	// a snapshot record carries other units' findings unchanged, so its own
	// UnitID says nothing about them.
	source := map[string]string{}
	for _, r := range core {
		if !r.FindingsSnapshot {
			for _, f := range r.Findings {
				source[f.ID] = r.UnitID
			}
		}
	}
	out := make([]model.Finding, len(current))
	for i, f := range current {
		if f.UnitID == "" {
			f.UnitID = source[f.ID]
		}
		out[i] = f
	}
	return out
}

// RequireLatestImportedReview returns the review record with this digest if
// it is an imported review of the unit's current implementation and no later
// review of the unit exists.
func RequireLatestImportedReview(records []Record, digest string) (*Record, error) {
	rr := recordByDigest(records, digest)
	if rr == nil || rr.Kind != KindReview || rr.Review == nil || rr.Review.ReportDigest == "" {
		return nil, errors.New(ReasonReviewMissing)
	}
	current, err := RequireImplementation(records, rr.Review.UnitID)
	if err != nil {
		return nil, err
	}
	if current != rr.Review.ImplementationRecordDigest {
		return nil, errors.New(ReasonImplementationAlreadyBound)
	}
	if !AnswersLatestReviewDispatch(records, *rr.Review) {
		return nil, errors.New(ReasonReviewPacketMismatch)
	}
	for _, r := range records {
		if r.Sequence > rr.Sequence && r.Kind == KindReview && r.Review != nil && r.Review.UnitID == rr.Review.UnitID {
			return nil, errors.New(ReasonReviewPacketMismatch)
		}
	}
	return rr, nil
}
