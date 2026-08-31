package engine

import (
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

func TestEvaluateResolution(t *testing.T) {
	stateCriteria := []model.Criterion{{ID: "C1"}}
	stateFindings := []model.Finding{
		{ID: "F1", Disposition: model.FindingWaivable},
		{ID: "F2", Disposition: model.FindingBlock},
	}

	tests := []struct {
		name string
		req  ResolutionRequest
		want Decision
	}{
		{
			name: "valid criterion waiver",
			req: ResolutionRequest{
				WaiverOperation: &model.WaiverOperation{
					Scope: model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"},
				},
				CurrentCriteria: stateCriteria,
				CurrentFindings: stateFindings,
			},
			want: DecisionLegalPendingApproval,
		},
		{
			name: "missing target criterion waiver",
			req: ResolutionRequest{
				WaiverOperation: &model.WaiverOperation{
					Scope: model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C99"},
				},
				CurrentCriteria: stateCriteria,
				CurrentFindings: stateFindings,
			},
			want: DecisionRejected,
		},
		{
			name: "valid finding override",
			req: ResolutionRequest{
				FindingOverride: &model.FindingDispositionOverride{
					FindingID: "F1",
				},
				CurrentCriteria: stateCriteria,
				CurrentFindings: stateFindings,
			},
			want: DecisionLegalPendingApproval,
		},
		{
			name: "override block finding",
			req: ResolutionRequest{
				FindingOverride: &model.FindingDispositionOverride{
					FindingID: "F2",
				},
				CurrentCriteria: stateCriteria,
				CurrentFindings: stateFindings,
			},
			want: DecisionRejected,
		},
		{
			name: "waive block finding",
			req: ResolutionRequest{
				WaiverOperation: &model.WaiverOperation{
					Scope: model.ScopeSelector{Kind: model.ScopeFinding, Target: "F2"},
				},
				CurrentCriteria: stateCriteria,
				CurrentFindings: stateFindings,
			},
			want: DecisionRejected,
		},
		{
			name: "missing operation",
			req: ResolutionRequest{
				CurrentCriteria: stateCriteria,
				CurrentFindings: stateFindings,
			},
			want: DecisionRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateResolution(tt.req)
			if got.Decision != tt.want {
				t.Errorf("EvaluateResolution() = %v, want %v", got.Decision, tt.want)
			}
			if got.MachineAdvanced {
				t.Error("MachineAdvanced should always be false")
			}
		})
	}
}
