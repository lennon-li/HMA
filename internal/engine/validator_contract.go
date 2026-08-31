package engine

import "github.com/lennon-li/HMA/internal/model"

// FindingRefKind is the class of contract object a recorded finding cites.
type FindingRefKind string

const (
	RefApprovedCriterion FindingRefKind = "approved_criterion"
	RefDeclaredBudget    FindingRefKind = "declared_budget"
	RefKernelRule        FindingRefKind = "kernel_rule"
	RefOutOfCriteria     FindingRefKind = "out_of_criteria_observation"
)

// Validator-contract rejection reasons. They extend the engine's existing
// Reason vocabulary and never describe a semantic judgement, only a
// structural contract violation in an already-recorded validator result.
const (
	ReasonMissingRouteAttestation   Reason = "MISSING_ROUTE_ATTESTATION_DIGEST"
	ReasonValidatorNotIndependent   Reason = "VALIDATOR_ROUTE_NOT_INDEPENDENT"
	ReasonDuplicateReference        Reason = "DUPLICATE_CRITERION_OR_BUDGET_REFERENCE"
	ReasonUnknownReference          Reason = "UNKNOWN_CRITERION_OR_BUDGET_REFERENCE"
	ReasonUnknownReferenceKind      Reason = "UNKNOWN_FINDING_REFERENCE_KIND"
	ReasonMissingKernelRuleID       Reason = "MISSING_KERNEL_RULE_IDENTIFIER"
	ReasonDuplicateFindingID        Reason = "DUPLICATE_FINDING_ID"
	ReasonUnknownFindingDisposition Reason = "UNKNOWN_FINDING_DISPOSITION"
	ReasonInvalidAdvisoryPairing    Reason = "INVALID_ADVISORY_DISPOSITION_PAIRING"
	ReasonInventedCriterion         Reason = "VALIDATOR_INVENTED_ACCEPTANCE_CRITERION"
	ReasonKernelViolation           Reason = "KERNEL_RULE_VIOLATION"
	ReasonAutomaticWaiverClaimed    Reason = "AUTOMATIC_WAIVER_OR_APPROVAL_CLAIMED"
	ReasonWaiverMutation            Reason = "WAIVER_MUTATION_ATTEMPTED"
	ReasonTerminalOutcomeEmission   Reason = "TERMINAL_OUTCOME_EMISSION_ATTEMPTED"
	ReasonRewindTargetSupplied      Reason = "REWIND_TARGET_NOT_PERMITTED"
)

// RecordedFinding is one already-recorded/stubbed validator finding.
type RecordedFinding struct {
	ID                  string                    `json:"id"`
	Disposition         model.FindingDisposition  `json:"disposition"`
	Advisory            model.AdvisoryDisposition `json:"advisory,omitempty"`
	RefKind             FindingRefKind            `json:"ref_kind,omitempty"`
	RefID               string                    `json:"ref_id,omitempty"`
	ClaimsWaived        bool                      `json:"claims_waived,omitempty"`
	ClaimsHumanApproval bool                      `json:"claims_human_approval,omitempty"`
}

// ValidatorContractRequest is an already-recorded validator result offered
// for structural contract checking.
type ValidatorContractRequest struct {
	ApprovedCriterionIDs              []string              `json:"approved_criterion_ids,omitempty"`
	DeclaredBudgetIDs                 []string              `json:"declared_budget_ids,omitempty"`
	ApprovedKernelRuleIDs             []string              `json:"approved_kernel_rule_ids,omitempty"`
	ImplementerRouteAttestationDigest string                `json:"implementer_route_attestation_digest,omitempty"`
	ValidatorRouteAttestationDigest   string                `json:"validator_route_attestation_digest,omitempty"`
	DeclaredIndependent               bool                  `json:"declared_independent"`
	Findings                          []RecordedFinding     `json:"findings,omitempty"`
	RequestRestoreApprovalIDs         []string              `json:"request_restore_approval_ids,omitempty"`
	RequestWaiverChangeIDs            []string              `json:"request_waiver_change_ids,omitempty"`
	ClaimedTerminalOutcome            model.TerminalOutcome `json:"claimed_terminal_outcome,omitempty"`
	RequestedRewindTarget             model.Stage           `json:"requested_rewind_target,omitempty"`
	ClaimsMachineAdvancement          bool                  `json:"claims_machine_advancement,omitempty"`
}

// ValidatorContractResult is the structural contract decision.
type ValidatorContractResult struct {
	Decision                     Decision              `json:"decision"`
	Reason                       Reason                `json:"reason,omitempty"`
	BlockingFindingIDs           []string              `json:"blocking_finding_ids,omitempty"`
	KernelViolationFindingIDs    []string              `json:"kernel_violation_finding_ids,omitempty"`
	UnresolvedWaivableFindingIDs []string              `json:"unresolved_waivable_finding_ids,omitempty"`
	AdvisoryFindingIDs           []string              `json:"advisory_finding_ids,omitempty"`
	CreatedCriterionIDs          []string              `json:"created_criterion_ids,omitempty"`
	GrantedWaiverIDs             []string              `json:"granted_waiver_ids,omitempty"`
	RewindTarget                 model.Stage           `json:"rewind_target,omitempty"`
	TerminalOutcome              model.TerminalOutcome `json:"terminal_outcome,omitempty"`
	RequiresFreshHumanApproval   bool                  `json:"requires_fresh_human_approval"`
	MachineAdvanced              bool                  `json:"machine_advanced"`
}

// EvaluateValidatorContract decides only whether an already-recorded or
// stubbed validator result is structurally eligible to be presented to a
// human as a contract-conforming result.
//
// It is pure and standard-library-only. It reads no repository, runtime, or
// configuration state, selects/verifies no route, executes nothing, decides
// no semantic truth, creates no acceptance criterion, grants no waiver,
// alters no approval, names no stage target, and emits no terminal outcome.
// A conforming result is still only LEGAL_PENDING_HUMAN_APPROVAL.
func EvaluateValidatorContract(req ValidatorContractRequest) ValidatorContractResult {
	rejected := func(r Reason) ValidatorContractResult {
		return ValidatorContractResult{
			Decision:                   DecisionRejected,
			Reason:                     r,
			RequiresFreshHumanApproval: true,
			MachineAdvanced:            false,
		}
	}

	// A recorded validator result may never claim an authority HMA reserves
	// for a human gate.
	if req.ClaimsMachineAdvancement {
		return rejected(ReasonMachineAdvancement)
	}
	if len(req.RequestRestoreApprovalIDs) > 0 {
		return rejected(ReasonApprovalReactivation)
	}
	if len(req.RequestWaiverChangeIDs) > 0 {
		return rejected(ReasonWaiverMutation)
	}
	if req.ClaimedTerminalOutcome != "" {
		return rejected(ReasonTerminalOutcomeEmission)
	}
	if req.RequestedRewindTarget != "" {
		return rejected(ReasonRewindTargetSupplied)
	}

	// Route-attestation digests are opaque, already-declared inputs. This
	// engine never resolves, persists, or cryptographically verifies them;
	// it only checks that two distinct ones exist and that the host already
	// declared the validator independent.
	if req.ImplementerRouteAttestationDigest == "" || req.ValidatorRouteAttestationDigest == "" {
		return rejected(ReasonMissingRouteAttestation)
	}
	if req.ImplementerRouteAttestationDigest == req.ValidatorRouteAttestationDigest ||
		!req.DeclaredIndependent {
		return rejected(ReasonValidatorNotIndependent)
	}

	// The frozen approved criterion set and the plan-declared measurable
	// budget set must each be a well-formed, non-overlapping set.
	criteria := make(map[string]bool, len(req.ApprovedCriterionIDs))
	for _, id := range req.ApprovedCriterionIDs {
		if id == "" {
			return rejected(ReasonUnknownReference)
		}
		if criteria[id] {
			return rejected(ReasonDuplicateReference)
		}
		criteria[id] = true
	}
	budgets := make(map[string]bool, len(req.DeclaredBudgetIDs))
	for _, id := range req.DeclaredBudgetIDs {
		if id == "" {
			return rejected(ReasonUnknownReference)
		}
		if budgets[id] || criteria[id] {
			return rejected(ReasonDuplicateReference)
		}
		budgets[id] = true
	}

	var blocking, kernel, waivable, advisory []string
	seen := make(map[string]bool, len(req.Findings))
	for _, f := range req.Findings {
		if f.ID == "" || seen[f.ID] {
			return rejected(ReasonDuplicateFindingID)
		}
		seen[f.ID] = true
		if !model.ValidFindingDisposition(f.Disposition) {
			return rejected(ReasonUnknownFindingDisposition)
		}
		// A recorded finding never waives itself and never stands in for a
		// human approval.
		if f.ClaimsWaived || f.ClaimsHumanApproval {
			return rejected(ReasonAutomaticWaiverClaimed)
		}
		if f.Disposition == model.FindingAdvisory {
			if !model.ValidAdvisoryDisposition(f.Advisory) {
				return rejected(ReasonInvalidAdvisoryPairing)
			}
		} else if f.Advisory != "" {
			return rejected(ReasonInvalidAdvisoryPairing)
		}

		switch f.RefKind {
		case RefApprovedCriterion:
			if !criteria[f.RefID] {
				// A blocking or waivable finding against an unapproved
				// criterion is an invented acceptance criterion (§9).
				if f.Disposition != model.FindingAdvisory {
					return rejected(ReasonInventedCriterion)
				}
				return rejected(ReasonUnknownReference)
			}
		case RefDeclaredBudget:
			if !budgets[f.RefID] {
				if f.Disposition != model.FindingAdvisory {
					return rejected(ReasonInventedCriterion)
				}
				return rejected(ReasonUnknownReference)
			}
		case RefKernelRule:
			// A kernel-rule finding stays outside the acceptance-criterion
			// set: it cites an unwaivable kernel rule, not a criterion or a
			// budget, and creates neither. It requires a non-empty ref_id,
			// must not collide with a criterion or budget id, and if the host
			// supplies a non-nil approved kernel-rule list the cited id must
			// be on it or it is rejected as an invented criterion.
			if f.RefID == "" {
				return rejected(ReasonMissingKernelRuleID)
			}
			if criteria[f.RefID] || budgets[f.RefID] {
				return rejected(ReasonDuplicateReference)
			}
			if req.ApprovedKernelRuleIDs != nil {
				approved := false
				for _, k := range req.ApprovedKernelRuleIDs {
					if k == f.RefID {
						approved = true
						break
					}
				}
				if !approved {
					return rejected(ReasonInventedCriterion)
				}
			}
		case RefOutOfCriteria:
			// Out-of-criteria observations are ADVISORY only (§9).
			if f.Disposition != model.FindingAdvisory {
				return rejected(ReasonInventedCriterion)
			}
		default:
			return rejected(ReasonUnknownReferenceKind)
		}

		switch f.Disposition {
		case model.FindingBlock:
			blocking = append(blocking, f.ID)
			if f.RefKind == RefKernelRule {
				kernel = append(kernel, f.ID)
			}
		case model.FindingWaivable:
			// Reported as unresolved: only a human waiver operation can
			// resolve it, and this engine performs none.
			waivable = append(waivable, f.ID)
		case model.FindingAdvisory:
			advisory = append(advisory, f.ID)
		}
	}

	return ValidatorContractResult{
		Decision:                     DecisionLegalPendingApproval,
		BlockingFindingIDs:           sorted(blocking),
		KernelViolationFindingIDs:    sorted(kernel),
		UnresolvedWaivableFindingIDs: sorted(waivable),
		AdvisoryFindingIDs:           sorted(advisory),
		CreatedCriterionIDs:          nil,
		GrantedWaiverIDs:             nil,
		RewindTarget:                 "",
		TerminalOutcome:              "",
		RequiresFreshHumanApproval:   true,
		MachineAdvanced:              false,
	}
}
