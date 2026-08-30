// Package engine implements HMA's deterministic, side-effect-free state
// control: it decides whether a proposed stage transition is one of the
// architecture's legal edges, and computes the approved Gate 0 rewind and
// invalidation result for an explicit input.
//
// The package is standard-library-only. It never persists a record, executes
// a command, opens a network connection, dispatches a route, or advances a
// stage. Every legal transition still requires a fresh, challenge-bound human
// approval, and no evaluation here mutates its caller's input.
package engine

import "github.com/lennon-li/HMA/internal/model"

// Decision is the deterministic classification of a proposed transition or
// invalidation input. A machine decision is never an approval.
type Decision string

const (
	// DecisionLegalPendingApproval means the proposal matches an approved
	// edge or rewind rule and may be presented to a human. It does not
	// advance anything.
	DecisionLegalPendingApproval Decision = "LEGAL_PENDING_HUMAN_APPROVAL"
	// DecisionRejected means the proposal is not representable as an
	// approved edge or rewind and must not be presented as advanceable.
	DecisionRejected Decision = "REJECTED"
)

// Reason is the deterministic cause of a rejection. It is empty when the
// decision is not a rejection.
type Reason string

const (
	ReasonNone                      Reason = ""
	ReasonUnknownSourceStage        Reason = "UNKNOWN_SOURCE_STAGE"
	ReasonUnknownTarget             Reason = "UNKNOWN_TARGET"
	ReasonMissingTarget             Reason = "MISSING_TARGET"
	ReasonAmbiguousTarget           Reason = "AMBIGUOUS_TARGET"
	ReasonSourceEqualsTarget        Reason = "SOURCE_EQUALS_TARGET"
	ReasonUnlistedEdge              Reason = "UNLISTED_EDGE"
	ReasonTerminalNotPermitted      Reason = "TERMINAL_OUTCOME_NOT_PERMITTED_FROM_STAGE"
	ReasonMachineAdvancement        Reason = "MACHINE_ADVANCEMENT_ATTEMPTED"
	ReasonApprovalReactivation      Reason = "INVALIDATED_APPROVAL_REACTIVATION_ATTEMPTED"
	ReasonUnknownRewindTrigger      Reason = "UNKNOWN_REWIND_TRIGGER"
	ReasonMissingAffectedUnit       Reason = "MISSING_AFFECTED_UNIT"
	ReasonUnitScopeOnWholeRunRewind Reason = "UNIT_SCOPE_DECLARED_FOR_WHOLE_RUN_REWIND"
)

// stageOrder is the architecture's stage sequence. It orders "downstream of
// the target gate" for invalidation; it does not by itself make an edge legal.
var stageOrder = map[model.Stage]int{
	model.StageGrounding:                   0,
	model.StageAcceptanceCriteria:          1,
	model.StagePlanning:                    2,
	model.StageRouteSelection:              3,
	model.StageImplementationAuthorization: 4,
	model.StageImplementationReview:        5,
	model.StageVerification:                6,
	model.StageIndependentValidation:       7,
	model.StageReleaseAndClosure:           8,
}

// ordinaryEdges holds the architecture's approved ordinary legal targets per
// source stage (§5.4). The universal escalation row -- any nonterminal stage
// may return to GROUNDING or PLANNING -- is applied separately by
// IsLegalEdge so it is not duplicated in every row.
var ordinaryEdges = map[model.Stage][]model.Stage{
	model.StageGrounding:          {model.StageAcceptanceCriteria},
	model.StageAcceptanceCriteria: {model.StagePlanning},
	model.StagePlanning:           {model.StageRouteSelection},
	model.StageRouteSelection:     {model.StageImplementationAuthorization},
	model.StageImplementationAuthorization: {
		model.StageImplementationReview,
		model.StageRouteSelection,
	},
	model.StageImplementationReview: {
		model.StageVerification,
		model.StageRouteSelection,
	},
	model.StageVerification: {
		model.StageIndependentValidation,
		model.StageRouteSelection,
	},
	model.StageIndependentValidation: {
		model.StageImplementationAuthorization,
		model.StagePlanning,
		model.StageReleaseAndClosure,
		model.StageRouteSelection,
	},
	model.StageReleaseAndClosure: {
		model.StageRouteSelection,
		model.StageAcceptanceCriteria,
		model.StagePlanning,
	},
}

// stoppingOutcomes are the truthful stopping conditions reachable from any
// nonterminal stage (§5.4, final row).
var stoppingOutcomes = map[model.TerminalOutcome]bool{
	model.OutcomeBlocked: true,
	model.OutcomeUnknown: true,
	model.OutcomeFailed:  true,
	model.OutcomeAborted: true,
	model.OutcomePartial: true,
}

// closureOutcomes are the successful terminal outcomes. §5.4 permits them
// only from RELEASE_AND_CLOSURE, and only after CI and human release
// approval, which this evaluator never supplies.
var closureOutcomes = map[model.TerminalOutcome]bool{
	model.OutcomeVerifiedSuccess:     true,
	model.OutcomeVerifiedWithWaivers: true,
}

// TransitionRequest is a proposed stage transition. Exactly one of
// TargetStage and TargetOutcome must be set. ClaimsMachineAdvancement records
// a proposal that asserts the evaluation itself advances the run; it is
// always rejected.
type TransitionRequest struct {
	Source                   model.Stage           `json:"source"`
	TargetStage              model.Stage           `json:"target_stage,omitempty"`
	TargetOutcome            model.TerminalOutcome `json:"target_outcome,omitempty"`
	ClaimsMachineAdvancement bool                  `json:"claims_machine_advancement,omitempty"`
}

// TransitionResult is the deterministic evaluation of a TransitionRequest.
// MachineAdvanced is structurally always false and RequiresFreshHumanApproval
// structurally always true: no input can make this engine advance a stage.
type TransitionResult struct {
	Decision                   Decision                `json:"decision"`
	Reason                     Reason                  `json:"reason,omitempty"`
	ControlState               model.StageControlState `json:"control_state,omitempty"`
	RequiresFreshHumanApproval bool                    `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool                    `json:"machine_advanced"`
}

// IsLegalEdge reports whether from -> to is one of the approved ordinary
// edges or the universal escalation row. A same-stage pair is not a
// transition and is never legal.
func IsLegalEdge(from, to model.Stage) bool {
	if !model.ValidStage(from) || !model.ValidStage(to) || from == to {
		return false
	}
	if to == model.StageGrounding || to == model.StagePlanning {
		return true
	}
	for _, t := range ordinaryEdges[from] {
		if t == to {
			return true
		}
	}
	return false
}

// EvaluateTransition classifies a proposed transition. It is pure: it reads
// req, returns a value, and changes no state.
func EvaluateTransition(req TransitionRequest) TransitionResult {
	rejected := func(r Reason) TransitionResult {
		return TransitionResult{
			Decision:                   DecisionRejected,
			Reason:                     r,
			RequiresFreshHumanApproval: true,
			MachineAdvanced:            false,
		}
	}
	if req.ClaimsMachineAdvancement {
		return rejected(ReasonMachineAdvancement)
	}
	if !model.ValidStage(req.Source) {
		return rejected(ReasonUnknownSourceStage)
	}
	hasStage := req.TargetStage != ""
	hasOutcome := req.TargetOutcome != ""
	switch {
	case hasStage && hasOutcome:
		return rejected(ReasonAmbiguousTarget)
	case !hasStage && !hasOutcome:
		return rejected(ReasonMissingTarget)
	case hasOutcome:
		if !model.ValidTerminalOutcome(req.TargetOutcome) {
			return rejected(ReasonUnknownTarget)
		}
		if closureOutcomes[req.TargetOutcome] && req.Source != model.StageReleaseAndClosure {
			return rejected(ReasonTerminalNotPermitted)
		}
		if !closureOutcomes[req.TargetOutcome] && !stoppingOutcomes[req.TargetOutcome] {
			return rejected(ReasonTerminalNotPermitted)
		}
	default:
		if !model.ValidStage(req.TargetStage) {
			return rejected(ReasonUnknownTarget)
		}
		if req.Source == req.TargetStage {
			return rejected(ReasonSourceEqualsTarget)
		}
		if !IsLegalEdge(req.Source, req.TargetStage) {
			return rejected(ReasonUnlistedEdge)
		}
	}
	return TransitionResult{
		Decision:                   DecisionLegalPendingApproval,
		ControlState:               model.ControlReadyForReview,
		RequiresFreshHumanApproval: true,
		MachineAdvanced:            false,
	}
}
