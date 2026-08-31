package engine

import (
	"github.com/lennon-li/HMA/internal/model"
)

// ResolutionRequest encapsulates the current chain state and the proposed human resolution operation.
type ResolutionRequest struct {
	// Only one of these may be provided per request
	WaiverOperation *model.WaiverOperation
	FindingOverride *model.FindingDispositionOverride

	CurrentCriteria []model.Criterion
	CurrentFindings []model.Finding
}

// ResolutionResult returns the validation decision for the human resolution.
// Because it evaluates human intent that alters the record stream (but never the stage),
// it mirrors the design of the other engine evaluators.
type ResolutionResult struct {
	Decision        Decision `json:"decision"`
	Reason          Reason   `json:"reason,omitempty"`
	MachineAdvanced bool     `json:"machine_advanced"`
}

// EvaluateResolution evaluates a single human resolution operation.
// It ensures that the target actually exists in the current state and that
// BLOCK findings are not illegally overridden. It never advances the stage.
func EvaluateResolution(req ResolutionRequest) ResolutionResult {
	if req.WaiverOperation != nil && req.FindingOverride != nil {
		return ResolutionResult{
			Decision:        DecisionRejected,
			Reason:          "MULTIPLE_OPERATIONS_PROVIDED",
			MachineAdvanced: false,
		}
	}
	if req.WaiverOperation == nil && req.FindingOverride == nil {
		return ResolutionResult{
			Decision:        DecisionRejected,
			Reason:          "MISSING_OPERATION",
			MachineAdvanced: false,
		}
	}

	if req.WaiverOperation != nil {
		w := req.WaiverOperation
		targetFound := false
		switch w.Scope.Kind {
		case model.ScopeCriterion:
			for _, c := range req.CurrentCriteria {
				if c.ID == w.Scope.Target {
					targetFound = true
					break
				}
			}
		case model.ScopeFinding:
			for _, f := range req.CurrentFindings {
				if f.ID == w.Scope.Target {
					targetFound = true
					if f.Disposition == model.FindingBlock {
						return ResolutionResult{
							Decision:        DecisionRejected,
							Reason:          "CANNOT_WAIVE_BLOCK_FINDING",
							MachineAdvanced: false,
						}
					}
					break
				}
			}
		case model.ScopeArtifact:
			targetFound = true // Assuming artifacts aren't strictly enumerable in the state right now
		}

		if !targetFound {
			return ResolutionResult{
				Decision:        DecisionRejected,
				Reason:          "TARGET_NOT_FOUND",
				MachineAdvanced: false,
			}
		}
	}

	if req.FindingOverride != nil {
		f := req.FindingOverride
		targetFound := false
		var currentDisposition model.FindingDisposition
		for _, curF := range req.CurrentFindings {
			if curF.ID == f.FindingID {
				targetFound = true
				currentDisposition = curF.Disposition
				break
			}
		}

		if !targetFound {
			return ResolutionResult{
				Decision:        DecisionRejected,
				Reason:          "TARGET_NOT_FOUND",
				MachineAdvanced: false,
			}
		}

		if currentDisposition == model.FindingBlock {
			return ResolutionResult{
				Decision:        DecisionRejected,
				Reason:          "CANNOT_OVERRIDE_BLOCK_FINDING",
				MachineAdvanced: false,
			}
		}
	}

	return ResolutionResult{
		Decision:        DecisionLegalPendingApproval, // It's valid, can be appended to JSONL
		MachineAdvanced: false,
	}
}
