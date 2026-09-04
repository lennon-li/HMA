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
					Operation: model.WaiverOperationGrant,
					Scope:     model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"},
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
					Operation: model.WaiverOperationGrant,
					Scope:     model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C99"},
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
					Operation: model.WaiverOperationGrant,
					Scope:     model.ScopeSelector{Kind: model.ScopeFinding, Target: "F2"},
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

func waiverOpRecord(run string, seq int64, op model.WaiverOperationAction, scope model.ScopeSelector, nonce string) model.Record {
	return model.Record{
		Kind:     model.KindWaiverOperation,
		Version:  1,
		Metadata: model.RecordMetadata{RunID: run, Sequence: seq},
		WaiverOperation: &model.WaiverOperation{
			Operation: op, Scope: scope, Justification: "j", Approver: "lennon",
			Timestamp: "2026-09-03T00:00:00Z", ChallengeNonce: nonce,
		},
	}
}

func TestProjectResolutionStateReplaysWaiversAndOverrides(t *testing.T) {
	scope := model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"}
	records := []model.Record{
		{Version: 1, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}},
			Findings: []model.Finding{{ID: "F1", Disposition: model.FindingWaivable}}},
		waiverOpRecord("run", 2, model.WaiverOperationGrant, scope, "n1"),
		{Version: 1, FindingOverride: &model.FindingDispositionOverride{
			FindingID: "F1", Disposition: model.FindingAdvisory, Justification: "j",
			Approver: "lennon", Timestamp: "2026-09-03T00:00:00Z", ChallengeNonce: "n2"}},
	}

	state := ProjectResolutionState(records)
	if !state.ActiveWaivers[scope] {
		t.Fatal("granted waiver is not active in the projection")
	}
	if !state.UsedNonces["n1"] || !state.UsedNonces["n2"] {
		t.Fatalf("nonces not collected: %v", state.UsedNonces)
	}
	if state.Findings[0].Disposition != model.FindingAdvisory {
		t.Fatalf("override not applied: %v", state.Findings[0].Disposition)
	}

	withdrawn := ProjectResolutionState(append(records,
		waiverOpRecord("run", 4, model.WaiverOperationWithdraw, scope, "n3")))
	if withdrawn.ActiveWaivers[scope] {
		t.Fatal("withdrawn waiver is still active")
	}
}

func TestEvaluateResolutionRejectsDoubleGrant(t *testing.T) {
	scope := model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"}
	state := ProjectResolutionState([]model.Record{
		{Version: 1, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}},
		waiverOpRecord("run", 2, model.WaiverOperationGrant, scope, "n1"),
	})

	req := NewResolutionRequest(state)
	req.WaiverOperation = &model.WaiverOperation{
		Operation: model.WaiverOperationGrant, Scope: scope, ChallengeNonce: "n2",
	}
	if got := EvaluateResolution(req); got.Reason != ReasonResolutionAlreadyWaived {
		t.Fatalf("second grant reason = %q, want %q", got.Reason, ReasonResolutionAlreadyWaived)
	}
}

func TestEvaluateResolutionRejectsWithdrawWithoutActiveWaiver(t *testing.T) {
	state := ProjectResolutionState([]model.Record{
		{Version: 1, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}},
	})
	req := NewResolutionRequest(state)
	req.WaiverOperation = &model.WaiverOperation{
		Operation: model.WaiverOperationWithdraw,
		Scope:     model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"},
	}
	if got := EvaluateResolution(req); got.Reason != ReasonResolutionNoActiveWaiver {
		t.Fatalf("reason = %q, want %q", got.Reason, ReasonResolutionNoActiveWaiver)
	}
}

func TestEvaluateResolutionRejectsReplayedNonce(t *testing.T) {
	scope := model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"}
	state := ProjectResolutionState([]model.Record{
		{Version: 1, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}},
		waiverOpRecord("run", 2, model.WaiverOperationGrant, scope, "n1"),
		waiverOpRecord("run", 3, model.WaiverOperationWithdraw, scope, "n2"),
	})

	req := NewResolutionRequest(state)
	req.WaiverOperation = &model.WaiverOperation{
		Operation: model.WaiverOperationGrant, Scope: scope, ChallengeNonce: "n1",
	}
	if got := EvaluateResolution(req); got.Reason != ReasonResolutionReusedNonce {
		t.Fatalf("replayed nonce reason = %q, want %q", got.Reason, ReasonResolutionReusedNonce)
	}
}

func TestEvaluateResolutionRejectsStaleChainPosition(t *testing.T) {
	state := ProjectResolutionState([]model.Record{
		{Version: 1, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}},
	})
	req := NewResolutionRequest(state)
	req.WaiverOperation = &model.WaiverOperation{
		Operation:      model.WaiverOperationGrant,
		Scope:          model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"},
		ChallengeNonce: "n1",
	}
	req.ExpectedPredecessorHead = "anchor-the-host-expected"
	req.PredecessorHead = "anchor-actually-on-disk"
	if got := EvaluateResolution(req); got.Reason != ReasonResolutionStalePosition {
		t.Fatalf("reason = %q, want %q", got.Reason, ReasonResolutionStalePosition)
	}
}

func TestEvaluateResolutionRejectsUnknownOperation(t *testing.T) {
	state := ProjectResolutionState(nil)
	req := NewResolutionRequest(state)
	req.WaiverOperation = &model.WaiverOperation{
		Scope: model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"},
	}
	if got := EvaluateResolution(req); got.Reason != ReasonResolutionInvalidOperation {
		t.Fatalf("reason = %q, want %q", got.Reason, ReasonResolutionInvalidOperation)
	}
}
