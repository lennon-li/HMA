package engine

import (
	"github.com/lennon-li/HMA/internal/model"
	"testing"
)

func TestEvaluateCIVerification(t *testing.T) {
	tests := []struct {
		name string
		req  CIVerificationRequest
		want Decision
	}{
		{
			name: "valid match",
			req: CIVerificationRequest{
				CIRepository:                     "google/uuid",
				CISHA:                            "12345",
				CurrentStage:                     model.StageVerification,
				ExpectedRepositoryIdentityDigest: "google/uuid",
				ExpectedProducedHeadDigest:       "12345",
			},
			want: DecisionLegalPendingApproval,
		},
		{
			name: "invalid stage",
			req: CIVerificationRequest{
				CIRepository:                     "google/uuid",
				CISHA:                            "12345",
				CurrentStage:                     model.StageGrounding,
				ExpectedRepositoryIdentityDigest: "google/uuid",
				ExpectedProducedHeadDigest:       "12345",
			},
			want: DecisionRejected,
		},
		{
			name: "repository mismatch",
			req: CIVerificationRequest{
				CIRepository:                     "other/repo",
				CISHA:                            "12345",
				CurrentStage:                     model.StageVerification,
				ExpectedRepositoryIdentityDigest: "google/uuid",
				ExpectedProducedHeadDigest:       "12345",
			},
			want: DecisionRejected,
		},
		{
			name: "revision mismatch",
			req: CIVerificationRequest{
				CIRepository:                     "google/uuid",
				CISHA:                            "54321",
				CurrentStage:                     model.StageVerification,
				ExpectedRepositoryIdentityDigest: "google/uuid",
				ExpectedProducedHeadDigest:       "12345",
			},
			want: DecisionRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCIVerification(tt.req)
			if got.Decision != tt.want {
				t.Errorf("EvaluateCIVerification() = %v, want %v", got.Decision, tt.want)
			}
		})
	}
}
