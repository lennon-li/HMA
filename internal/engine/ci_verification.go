package engine

import (
	"github.com/lennon-li/HMA/internal/model"
)

// CIVerificationRequest encapsulates the GitHub Actions context and the current chain state.
type CIVerificationRequest struct {
	CIRepository string
	CISHA        string
	CurrentStage model.Stage
	// The identity digest expected by the chain (e.g. from an approval)
	ExpectedRepositoryIdentityDigest string
	ExpectedProducedHeadDigest       string
}

// EvaluateCIVerification ensures the CI environment precisely matches the expected state.
func EvaluateCIVerification(req CIVerificationRequest) ResolutionResult {
	if req.CurrentStage != model.StageVerification && req.CurrentStage != model.StageImplementationReview {
		// Only evaluating evidence binding rules at specific stages for CI.
		return ResolutionResult{
			Decision:        DecisionRejected,
			Reason:          "INVALID_STAGE_FOR_CI_VERIFICATION",
			MachineAdvanced: false,
		}
	}

	if req.CIRepository != req.ExpectedRepositoryIdentityDigest {
		return ResolutionResult{
			Decision:        DecisionRejected,
			Reason:          "CI_REPOSITORY_MISMATCH",
			MachineAdvanced: false,
		}
	}

	if req.CISHA != req.ExpectedProducedHeadDigest {
		return ResolutionResult{
			Decision:        DecisionRejected,
			Reason:          "CI_REVISION_MISMATCH",
			MachineAdvanced: false,
		}
	}

	return ResolutionResult{
		Decision:        DecisionLegalPendingApproval,
		MachineAdvanced: false,
	}
}
