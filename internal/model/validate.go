// Package model validation: reject unknown enum values and structural
// contradictions. Standard-library-only; no state advancement.
package model

import (
	"errors"
	"fmt"
	"strings"
)

// ValidRecordKind reports whether k is an approved record family.
func ValidRecordKind(k RecordKind) bool {
	switch k {
	case KindStageTransition, KindCriterion, KindFinding, KindWaiver,
		KindEvidence, KindReleaseRequest, KindOutcome,
		KindWaiverOperation, KindFindingOverride:
		return true
	}
	return false
}

// ValidStage reports whether s is an approved stage.
func ValidStage(s Stage) bool {
	switch s {
	case StageGrounding, StageAcceptanceCriteria, StagePlanning,
		StageRouteSelection, StageImplementationAuthorization,
		StageImplementationReview, StageVerification,
		StageIndependentValidation, StageReleaseAndClosure:
		return true
	}
	return false
}

// ValidControlState reports whether c is an approved stage-control state.
// INVALIDATED is not a stage-control state; it is packet-control vocabulary.
func ValidControlState(c StageControlState) bool {
	switch c {
	case ControlDraft, ControlReadyForReview, ControlApproved:
		return true
	}
	return false
}

// ValidPacketControlState reports whether c is an approved packet-control
// state.
func ValidPacketControlState(c PacketControlState) bool {
	switch c {
	case PacketInvalidated:
		return true
	}
	return false
}

// ValidCriterionDisposition reports whether d is an approved criterion state.
func ValidCriterionDisposition(d CriterionDisposition) bool {
	switch d {
	case CriterionPending, CriterionPassed, CriterionFailed, CriterionWaived:
		return true
	}
	return false
}

// ValidFindingDisposition reports whether d is an approved finding disposition.
func ValidFindingDisposition(d FindingDisposition) bool {
	switch d {
	case FindingBlock, FindingWaivable, FindingAdvisory:
		return true
	}
	return false
}

// ValidAdvisoryDisposition reports whether d is an approved advisory disposition.
func ValidAdvisoryDisposition(d AdvisoryDisposition) bool {
	switch d {
	case AdvisoryAccept, AdvisoryPark, AdvisoryKill:
		return true
	}
	return false
}

// ValidTerminalOutcome reports whether o is an approved terminal outcome.
func ValidTerminalOutcome(o TerminalOutcome) bool {
	switch o {
	case OutcomeAborted, OutcomeBlocked, OutcomeUnknown, OutcomeFailed,
		OutcomePartial, OutcomeVerifiedWithWaivers, OutcomeVerifiedSuccess:
		return true
	}
	return false
}

// ValidScopeKind reports whether k is an approved waiver scope kind. "stage"
// is not approved because stages are never waived; "plan_unit" and "closure"
// are not approved waiver scopes.
func ValidScopeKind(k ScopeKind) bool {
	switch k {
	case ScopeCriterion, ScopeFinding, ScopeArtifact:
		return true
	}
	return false
}

// requiresHeadDiff reports whether an approval at stage s must additionally
// bind the produced head revision and diff digest, per the human-approval
// contract.
func requiresHeadDiff(s Stage) bool {
	switch s {
	case StageImplementationReview, StageVerification,
		StageIndependentValidation, StageReleaseAndClosure:
		return true
	}
	return false
}

func criterionExists(criteria []Criterion, id string) bool {
	for _, c := range criteria {
		if c.ID == id {
			return true
		}
	}
	return false
}

func waivableFindingExists(findings []Finding, id string) bool {
	for _, f := range findings {
		if f.ID == id && f.Disposition == FindingWaivable {
			return true
		}
	}
	return false
}

// validateApproval validates an ApprovalBinding's universal bindings, enum
// fields, and stage-specific produced-head/diff binding. It never advances a
// transition.
func validateApproval(a *ApprovalBinding) error {
	if a == nil {
		return nil
	}
	universal := []struct{ name, val string }{
		{"run_id", a.RunID},
		{"transition_digest", a.TransitionDigest},
		{"current_stage", string(a.CurrentStage)},
		{"proposed_target_stage", string(a.ProposedTargetStage)},
		{"repository_identity_digest", a.RepositoryIdentityDigest},
		{"base_revision_digest", a.BaseRevisionDigest},
		{"stage_time_digest", a.StageTimeDigest},
		{"accepted_plan_digest", a.AcceptedPlanDigest},
		{"challenge_nonce", a.ChallengeNonce},
		{"approver", a.Approver},
		{"timestamp", a.Timestamp},
	}
	for _, f := range universal {
		if f.val == "" {
			return fmt.Errorf("approval missing %s", f.name)
		}
	}
	if !ValidStage(a.CurrentStage) {
		return fmt.Errorf("approval current_stage %q is not an approved stage", a.CurrentStage)
	}
	if !ValidStage(a.ProposedTargetStage) {
		return fmt.Errorf("approval proposed_target_stage %q is not an approved stage", a.ProposedTargetStage)
	}
	if requiresHeadDiff(a.CurrentStage) {
		if a.ProducedHeadDigest == "" {
			return fmt.Errorf("approval missing produced_head_digest for stage %q", a.CurrentStage)
		}
		if a.DiffDigest == "" {
			return fmt.Errorf("approval missing diff_digest for stage %q", a.CurrentStage)
		}
	} else {
		if a.ProducedHeadDigest != "" {
			return fmt.Errorf("approval must not bind produced_head_digest for stage %q", a.CurrentStage)
		}
		if a.DiffDigest != "" {
			return fmt.Errorf("approval must not bind diff_digest for stage %q", a.CurrentStage)
		}
	}
	return nil
}

// validateEvidence structurally validates an EvidenceRef as an explicit
// executable/argv record with required revision and evidence-provenance
// fields. No shell string is representable here.
func validateEvidence(e *EvidenceRef) error {
	if e == nil {
		return nil
	}
	if e.Executable == "" {
		return errors.New("evidence missing executable")
	}
	if err := validateRouteAttestation(e.Route); err != nil {
		return err
	}
	if e.Argv == nil {
		return errors.New("evidence missing argv vector")
	}
	if e.WorkingDir == "" {
		return errors.New("evidence missing working_dir")
	}
	if e.StartTimestamp == "" {
		return errors.New("evidence missing start_timestamp")
	}
	if e.EndTimestamp == "" {
		return errors.New("evidence missing end_timestamp")
	}
	if e.RepositoryIdentity == "" {
		return errors.New("evidence missing repository_identity")
	}
	if e.BaseRevision == "" {
		return errors.New("evidence missing base_revision")
	}
	if e.HeadRevision == "" {
		return errors.New("evidence missing head_revision")
	}
	if e.CapturingActor == "" {
		return errors.New("evidence missing capturing_actor")
	}
	return nil
}

// ValidateRecord rejects unknown enum values and structural contradictions:
// an unknown record version, a waived stage, an unscoped or out-of-contract
// active waiver, an active waiver inconsistent with the record, a
// contradictory successful outcome, an advisory disposition paired with the
// wrong finding class, and missing or mismatched chain metadata. It also
// validates ApprovalBinding and EvidenceRef whenever non-nil. It never
// advances state, derives outcomes, or invokes a shell.
func ValidateRecord(r *Record) error {
	if r == nil {
		return errors.New("record is nil")
	}
	if r.Metadata.IsZero() {
		return errors.New("missing chain metadata")
	}
	if r.Metadata.RunID == "" {
		return errors.New("missing run id")
	}
	if r.Metadata.Sequence < 1 {
		return errors.New("invalid sequence")
	}
	if r.Metadata.HeadAnchor == "" {
		return errors.New("missing head anchor")
	}
	if r.Metadata.Sequence == 1 && r.Metadata.PredecessorHash != "" {
		return errors.New("genesis record must not carry a predecessor hash")
	}
	if r.Metadata.Sequence > 1 && r.Metadata.PredecessorHash == "" {
		return errors.New("non-genesis record missing predecessor hash")
	}
	if r.Version != 1 {
		return fmt.Errorf("record version must be 1, got %d", r.Version)
	}
	if r.Kind != "" && !ValidRecordKind(r.Kind) {
		return fmt.Errorf("unknown record kind %q", r.Kind)
	}
	if r.Stage != "" && !ValidStage(r.Stage) {
		return fmt.Errorf("unknown stage %q", r.Stage)
	}
	if r.Control != "" && !ValidControlState(r.Control) {
		return fmt.Errorf("unknown control state %q", r.Control)
	}
	if r.Outcome != "" && !ValidTerminalOutcome(r.Outcome) {
		return fmt.Errorf("unknown terminal outcome %q", r.Outcome)
	}
	for i := range r.Criteria {
		d := r.Criteria[i].Disposition
		if !ValidCriterionDisposition(d) {
			return fmt.Errorf("criterion %d unknown disposition %q", i, d)
		}
	}
	for i := range r.Findings {
		f := r.Findings[i]
		if !ValidFindingDisposition(f.Disposition) {
			return fmt.Errorf("finding %d unknown disposition %q", i, f.Disposition)
		}
		if f.Disposition == FindingAdvisory {
			if !ValidAdvisoryDisposition(f.Advisory) {
				return fmt.Errorf("finding %d advisory disposition missing or unknown", i)
			}
		} else if f.Advisory != "" {
			return fmt.Errorf("finding %d advisory disposition set for non-advisory finding", i)
		}
	}
	activeWaiver := false
	for i := range r.Waivers {
		w := r.Waivers[i]
		if strings.EqualFold(string(w.Scope.Kind), "stage") {
			return fmt.Errorf("waiver %d attempts to waive a stage; stages are never waived", i)
		}
		if !ValidScopeKind(w.Scope.Kind) {
			return fmt.Errorf("waiver %d invalid scope kind %q", i, w.Scope.Kind)
		}
		if w.Scope.Target == "" {
			return fmt.Errorf("waiver %d missing scope target", i)
		}
		if w.Active {
			switch w.Scope.Kind {
			case ScopeCriterion:
				if !criterionExists(r.Criteria, w.Scope.Target) {
					return fmt.Errorf("waiver %d active criterion waiver targets unknown criterion %q", i, w.Scope.Target)
				}
			case ScopeFinding:
				if !waivableFindingExists(r.Findings, w.Scope.Target) {
					return fmt.Errorf("waiver %d active finding waiver targets non-waivable or unknown finding %q", i, w.Scope.Target)
				}
			case ScopeArtifact:
			}
			activeWaiver = true
		}
	}
	switch r.Outcome {
	case OutcomeVerifiedSuccess:
		if activeWaiver {
			return errors.New("VERIFIED_SUCCESS with an active waiver")
		}
		for i := range r.Criteria {
			if r.Criteria[i].Disposition != CriterionPassed {
				return errors.New("VERIFIED_SUCCESS requires every criterion PASSED")
			}
		}
	case OutcomeVerifiedWithWaivers:
		if !activeWaiver {
			return errors.New("VERIFIED_WITH_WAIVERS without an active waiver")
		}
		for i := range r.Criteria {
			d := r.Criteria[i].Disposition
			if d != CriterionPassed && d != CriterionWaived {
				return errors.New("VERIFIED_WITH_WAIVERS requires every criterion PASSED or WAIVED")
			}
		}
	}
	if err := validateApproval(r.Approval); err != nil {
		return err
	}
	if err := validateEvidence(r.Evidence); err != nil {
		return err
	}
	if err := validateWaiverOperation(r.WaiverOperation); err != nil {
		return err
	}
	if err := validateFindingOverride(r.FindingOverride); err != nil {
		return err
	}
	return nil
}

func validateWaiverOperation(w *WaiverOperation) error {
	if w == nil {
		return nil
	}
	if w.Operation != WaiverOperationGrant && w.Operation != WaiverOperationExpire && w.Operation != WaiverOperationWithdraw {
		return fmt.Errorf("invalid waiver operation %q", w.Operation)
	}
	if !ValidScopeKind(w.Scope.Kind) {
		return fmt.Errorf("invalid waiver scope kind %q", w.Scope.Kind)
	}
	if strings.EqualFold(string(w.Scope.Kind), "stage") {
		return errors.New("waiver operation attempts to waive a stage; stages are never waived")
	}
	if w.Scope.Target == "" {
		return errors.New("waiver operation missing scope target")
	}
	if w.Justification == "" {
		return errors.New("waiver operation missing justification")
	}
	if w.Approver == "" {
		return errors.New("waiver operation missing approver")
	}
	if w.Timestamp == "" {
		return errors.New("waiver operation missing timestamp")
	}
	if w.ChallengeNonce == "" {
		return errors.New("waiver operation missing challenge_nonce")
	}
	return nil
}

func validateFindingOverride(f *FindingDispositionOverride) error {
	if f == nil {
		return nil
	}
	if f.FindingID == "" {
		return errors.New("finding override missing finding_id")
	}
	if !ValidFindingDisposition(f.Disposition) {
		return fmt.Errorf("invalid finding override disposition %q", f.Disposition)
	}
	if f.Disposition == FindingBlock {
		return errors.New("finding override cannot set BLOCK disposition")
	}
	if f.Justification == "" {
		return errors.New("finding override missing justification")
	}
	if f.Approver == "" {
		return errors.New("finding override missing approver")
	}
	if f.Timestamp == "" {
		return errors.New("finding override missing timestamp")
	}
	if f.ChallengeNonce == "" {
		return errors.New("finding override missing challenge_nonce")
	}
	return nil
}
