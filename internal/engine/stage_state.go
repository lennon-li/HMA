package engine

import (
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonStageMismatch           Reason = "APPROVAL_STAGE_IS_NOT_THE_RUN_STAGE"
	ReasonRunTerminal             Reason = "RUN_ALREADY_TERMINAL"
	ReasonMissingRequiredEvidence Reason = "REQUIRED_EVIDENCE_NOT_IN_CHAIN"
	ReasonStaleRequiredEvidence   Reason = "REQUIRED_EVIDENCE_BOUND_TO_ANOTHER_REVISION"
)

// StageState is the projection of where a run's committed chain has actually
// reached. It is derived by replaying the chain: the chain is the run, and
// nothing outside it decides what stage a run is in.
type StageState struct {
	// CurrentStage is the stage the run is in. A chain that has recorded no
	// approved transition is in GROUNDING, because a run has not left its
	// first stage until a human approved leaving it.
	CurrentStage model.Stage
	// Sequence is the number of committed records, so the next record's
	// sequence is Sequence+1.
	Sequence int64
	// PredecessorHead is the head anchor of the chain tail, empty for an
	// empty chain.
	PredecessorHead string
	// Terminal reports that the run recorded a terminal outcome. A terminal
	// run accepts no further transition.
	Terminal bool
	Outcome  model.TerminalOutcome
	// UsedNonces spans every challenge nonce the chain has consumed, of any
	// record family: a nonce spent on a waiver is spent for an approval too.
	UsedNonces map[string]bool
	// Evidence is every evidence reference the chain carries, in order.
	Evidence []model.EvidenceRef
}

// ProjectStageState replays a committed chain into its stage position. It
// advances nothing and decides nothing; it reports only what the chain says.
//
// Only a stage_transition record whose control state is APPROVED moves the
// run, and it moves the run to its approval's proposed target: Control is a
// stage-control state describing the stage the record is filed under, so
// APPROVED means that stage is approved and the run may leave it. A
// READY_FOR_REVIEW stage_transition -- what an evidence capture writes --
// records that the stage awaits a decision, which is the opposite of having
// moved.
func ProjectStageState(records []model.Record) StageState {
	state := StageState{
		CurrentStage: model.StageGrounding,
		Sequence:     int64(len(records)),
		UsedNonces:   make(map[string]bool),
	}
	for i := range records {
		r := records[i]
		if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved &&
			r.Approval != nil && model.ValidStage(r.Approval.ProposedTargetStage) {
			state.CurrentStage = r.Approval.ProposedTargetStage
		}
		if r.Kind == model.KindOutcome && model.ValidTerminalOutcome(r.Outcome) {
			state.Terminal = true
			state.Outcome = r.Outcome
		}
		if r.Approval != nil && r.Approval.ChallengeNonce != "" {
			state.UsedNonces[r.Approval.ChallengeNonce] = true
		}
		if w := r.WaiverOperation; w != nil && w.ChallengeNonce != "" {
			state.UsedNonces[w.ChallengeNonce] = true
		}
		if f := r.FindingOverride; f != nil && f.ChallengeNonce != "" {
			state.UsedNonces[f.ChallengeNonce] = true
		}
		if r.Evidence != nil {
			state.Evidence = append(state.Evidence, *r.Evidence)
		}
	}
	if len(records) > 0 {
		state.PredecessorHead = records[len(records)-1].Metadata.HeadAnchor
	}
	return state
}

// StageTransitionRequest is one proposed, human-approved stage transition
// evaluated against the run's committed chain.
type StageTransitionRequest struct {
	State     StageState
	Expected  model.ApprovalBinding
	Presented model.ApprovalBinding
	Now       time.Time
	MaxAge    time.Duration
}

// StageTransitionResult is the deterministic evaluation of a proposed
// transition. MachineAdvanced is structurally always false: a legal result
// authorizes recording the human's decision, and is not itself a decision.
type StageTransitionResult struct {
	Decision                   Decision                `json:"decision"`
	Reason                     Reason                  `json:"reason,omitempty"`
	FromStage                  model.Stage             `json:"from_stage,omitempty"`
	ToStage                    model.Stage             `json:"to_stage,omitempty"`
	ControlState               model.StageControlState `json:"control_state,omitempty"`
	RequiresFreshHumanApproval bool                    `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool                    `json:"machine_advanced"`
}

// requiredEvidenceReason reports why an approval's plan-declared required
// evidence is not satisfied by the chain, or ReasonNone when it is.
//
// A digest satisfies the requirement only when the chain holds evidence with
// that output digest captured against the revision the approval binds. This
// is what stops the reuse of evidence gathered before the code changed: the
// digest still matches, but the revision it was taken at does not.
func requiredEvidenceReason(state StageState, a model.ApprovalBinding) Reason {
	for _, want := range a.RequiredEvidenceDigests {
		if want == "" {
			return ReasonMalformedApproval
		}
		present, satisfied := false, false
		for _, e := range state.Evidence {
			if e.OutputDigest != want {
				continue
			}
			present = true
			if e.BaseRevision != a.BaseRevisionDigest {
				continue
			}
			if a.ProducedHeadDigest != "" && e.HeadRevision != a.ProducedHeadDigest {
				continue
			}
			satisfied = true
			break
		}
		// Every declared digest must be satisfied. Returning on the first
		// match would enforce only the first requirement of a multi-evidence
		// plan and silently ignore the rest.
		switch {
		case satisfied:
		case present:
			return ReasonStaleRequiredEvidence
		default:
			return ReasonMissingRequiredEvidence
		}
	}
	return ReasonNone
}

// EvaluateStageTransition decides whether one approved stage transition may be
// recorded against a chain. It is pure: it reads req, returns a value, and
// changes no state. A LEGAL_PENDING_HUMAN_APPROVAL decision means the human
// decision carried in the approval binding is admissible for recording; the
// engine neither makes nor substitutes that decision.
func EvaluateStageTransition(req StageTransitionRequest) StageTransitionResult {
	from := req.State.CurrentStage
	to := req.Presented.ProposedTargetStage
	reject := func(r Reason) StageTransitionResult {
		return StageTransitionResult{
			Decision:                   DecisionRejected,
			Reason:                     r,
			FromStage:                  from,
			ToStage:                    to,
			RequiresFreshHumanApproval: true,
			MachineAdvanced:            false,
		}
	}
	if req.State.Terminal {
		return reject(ReasonRunTerminal)
	}
	if req.Presented.CurrentStage != from {
		return reject(ReasonStageMismatch)
	}
	tr := EvaluateTransition(TransitionRequest{Source: from, TargetStage: to})
	if tr.Decision != DecisionLegalPendingApproval {
		return reject(tr.Reason)
	}
	ar := EvaluateApprovalBinding(ApprovalBindingRequest{
		Expected:        req.Expected,
		Presented:       req.Presented,
		PredecessorHead: req.State.PredecessorHead,
		Sequence:        req.State.Sequence + 1,
		UsedNonces:      req.State.UsedNonces,
		Now:             req.Now,
		MaxAge:          req.MaxAge,
	})
	if ar.Decision != DecisionLegalPendingApproval {
		return reject(ar.Reason)
	}
	if r := requiredEvidenceReason(req.State, req.Presented); r != ReasonNone {
		return reject(r)
	}
	return StageTransitionResult{
		Decision:                   DecisionLegalPendingApproval,
		FromStage:                  from,
		ToStage:                    to,
		ControlState:               model.ControlApproved,
		RequiresFreshHumanApproval: true,
		MachineAdvanced:            false,
	}
}
