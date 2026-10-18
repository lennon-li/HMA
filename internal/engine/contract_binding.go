package engine

import (
	"sort"

	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonContractRequired       Reason = "ACCEPTANCE_CONTRACT_REQUIRED"
	ReasonContractMismatch       Reason = "ACCEPTANCE_CONTRACT_DIGEST_MISMATCH"
	ReasonContractCriteriaDiffer Reason = "ACCEPTANCE_CONTRACT_CRITERIA_MISMATCH"
)

// PresentedContract is a contract document the host presents with a
// transition, already parsed and validated, with its recomputed digest.
type PresentedContract struct {
	Contract model.AcceptanceContract
	Digest   string
}

// BoundContractDigest returns the acceptance contract digest the run's chain
// binds: the digest on the latest approved ACCEPTANCE_CRITERIA -> PLANNING
// transition, or "" when the run has never accepted a contract.
func BoundContractDigest(records []model.Record) string {
	bound := ""
	for _, r := range records {
		if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.CurrentStage == model.StageAcceptanceCriteria && r.Approval.ProposedTargetStage == model.StagePlanning {
			bound = r.Approval.AcceptanceContractDigest
		}
	}
	return bound
}

// usedContract reports whether any approval in the chain accepted a
// contract; once one has, a run cannot fall back to having none.
func usedContract(records []model.Record) bool {
	for _, r := range records {
		if r.Approval != nil && r.Approval.AcceptanceContractDigest != "" {
			return true
		}
	}
	return false
}

// EvaluateContractBinding checks a presented approval against the run's
// acceptance contract (Gate A v3 §2.1). It is pure.
//
//   - ACCEPTANCE_CRITERIA -> PLANNING accepts a contract: when the approval
//     names a digest, the host must present the contract, its recomputed
//     digest must equal the approval's, its run must be this run, and its
//     criterion ids must equal the run's core criterion records exactly. A
//     run that has accepted a contract must accept one again here.
//   - Every other approval must carry the bound digest unchanged ("" for a
//     run without a contract), and a presented contract must hash to it, so
//     changing a threshold under the same ids is refused until a human
//     approves a rewind and a new contract. The core criterion set must
//     still equal the contract's.
func EvaluateContractBinding(records []model.Record, presented model.ApprovalBinding, doc *PresentedContract) Reason {
	state := ProjectStageState(records)
	if presented.CurrentStage == model.StageAcceptanceCriteria && presented.ProposedTargetStage == model.StagePlanning && presented.ProposedTargetOutcome == "" {
		if presented.AcceptanceContractDigest == "" {
			// A presented document is never silently dropped.
			if doc != nil {
				return ReasonContractMismatch
			}
			if usedContract(records) {
				return ReasonContractRequired
			}
			return ReasonNone
		}
		if doc == nil {
			return ReasonContractRequired
		}
		if doc.Digest != presented.AcceptanceContractDigest || doc.Contract.RunID != presented.RunID {
			return ReasonContractMismatch
		}
		if !sameIDs(model.ContractCriterionIDs(doc.Contract), state.Criteria) {
			return ReasonContractCriteriaDiffer
		}
		return ReasonNone
	}
	bound := BoundContractDigest(records)
	if presented.AcceptanceContractDigest != bound {
		return ReasonContractMismatch
	}
	if bound == "" {
		if doc != nil {
			return ReasonContractMismatch
		}
		return ReasonNone
	}
	if doc != nil {
		if doc.Digest != bound {
			return ReasonContractMismatch
		}
		if !sameIDs(model.ContractCriterionIDs(doc.Contract), state.Criteria) {
			return ReasonContractCriteriaDiffer
		}
	}
	if !criteriaMatchBound(records, state.Criteria) {
		return ReasonContractCriteriaDiffer
	}
	return ReasonNone
}

// criteriaMatchBound checks the current core criterion set still has the
// ids it had when the bound contract was accepted (they equalled the
// contract's then).
func criteriaMatchBound(records []model.Record, current []model.Criterion) bool {
	var latest, atAccept []model.Criterion
	for _, r := range records {
		if len(r.Criteria) > 0 {
			latest = r.Criteria
		}
		if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.CurrentStage == model.StageAcceptanceCriteria && r.Approval.ProposedTargetStage == model.StagePlanning {
			atAccept = latest
		}
	}
	accepted := make([]string, 0, len(atAccept))
	for _, c := range atAccept {
		accepted = append(accepted, c.ID)
	}
	sort.Strings(accepted)
	return sameIDs(accepted, current)
}

func sameIDs(want []string, criteria []model.Criterion) bool {
	got := make([]string, 0, len(criteria))
	seen := map[string]bool{}
	for _, c := range criteria {
		if seen[c.ID] {
			return false
		}
		seen[c.ID] = true
		got = append(got, c.ID)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
