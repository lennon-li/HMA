package engine

import (
	"github.com/lennon-li/HMA/internal/model"
)

// CIVerificationRequest encapsulates the GitHub Actions context and the current chain state.
type CIVerificationRequest struct {
	CIRepository string
	CISHA        string
	CurrentStage model.Stage
	// ExpectedRepositoryIdentity is the portable repository identity bound by
	// the approval. For the GitHub adapter the host must bind it to the
	// "owner/repo" slug, because that is the only identity a GitHub Actions
	// runner can derive independently from its own environment. It is an
	// identity string, not a digest.
	ExpectedRepositoryIdentity string
	// ExpectedProducedHead is the approved head revision, compared against
	// the revision CI actually checked out.
	ExpectedProducedHead string
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

	if req.CIRepository != req.ExpectedRepositoryIdentity {
		return ResolutionResult{
			Decision:        DecisionRejected,
			Reason:          "CI_REPOSITORY_MISMATCH",
			MachineAdvanced: false,
		}
	}

	if req.CISHA != req.ExpectedProducedHead {
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
