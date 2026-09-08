package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

func approvedTransition(seq int64, from, to model.Stage, nonce string) model.Record {
	return model.Record{
		Kind:    model.KindStageTransition,
		Version: 1,
		Metadata: model.RecordMetadata{
			RunID: "run", Sequence: seq,
			HeadAnchor: "anchor-" + nonce,
			Timestamp:  "2026-09-05T00:00:00Z", Actor: "operator-confirmed:lennon",
		},
		Stage:   from,
		Control: model.ControlApproved,
		Approval: &model.ApprovalBinding{
			CurrentStage: from, ProposedTargetStage: to, ChallengeNonce: nonce,
		},
	}
}

func TestProjectStageStateStartsAtGrounding(t *testing.T) {
	state := ProjectStageState(nil)
	if state.CurrentStage != model.StageGrounding {
		t.Fatalf("empty chain projects stage %q, want GROUNDING", state.CurrentStage)
	}
	if state.Sequence != 0 || state.PredecessorHead != "" || state.Terminal {
		t.Fatalf("empty chain projected as %+v", state)
	}
}

func TestProjectStageStateFollowsApprovedTargets(t *testing.T) {
	records := []model.Record{
		approvedTransition(1, model.StageGrounding, model.StageAcceptanceCriteria, "n1"),
		approvedTransition(2, model.StageAcceptanceCriteria, model.StagePlanning, "n2"),
	}
	state := ProjectStageState(records)
	if state.CurrentStage != model.StagePlanning {
		t.Fatalf("stage = %q, want PLANNING", state.CurrentStage)
	}
	if state.Sequence != 2 || state.PredecessorHead != "anchor-n2" {
		t.Fatalf("state = %+v", state)
	}
	if !state.UsedNonces["n1"] || !state.UsedNonces["n2"] {
		t.Fatalf("nonces = %v, want both spent", state.UsedNonces)
	}
}

// TestProjectStageStateIgnoresReadyForReview is the distinction the whole
// projection rests on: an evidence capture files a READY_FOR_REVIEW
// stage_transition at the current stage, which is a stage asking for a
// decision, not a stage that moved.
func TestProjectStageStateIgnoresReadyForReview(t *testing.T) {
	pending := approvedTransition(1, model.StageGrounding, model.StageAcceptanceCriteria, "n1")
	pending.Control = model.ControlReadyForReview
	if got := ProjectStageState([]model.Record{pending}).CurrentStage; got != model.StageGrounding {
		t.Fatalf("a ready-for-review record moved the run to %q", got)
	}
}

func TestProjectStageStateSpendsNoncesFromEveryFamily(t *testing.T) {
	records := []model.Record{
		{Kind: model.KindWaiverOperation, WaiverOperation: &model.WaiverOperation{ChallengeNonce: "waiver-nonce"}},
		{Kind: model.KindFindingOverride, FindingOverride: &model.FindingDispositionOverride{ChallengeNonce: "override-nonce"}},
	}
	state := ProjectStageState(records)
	if !state.UsedNonces["waiver-nonce"] || !state.UsedNonces["override-nonce"] {
		t.Fatalf("nonces = %v, want a nonce spent on any record family to be spent", state.UsedNonces)
	}
}

func TestProjectStageStateMarksTerminalOutcome(t *testing.T) {
	state := ProjectStageState([]model.Record{{Kind: model.KindOutcome, Outcome: model.OutcomeFailed}})
	if !state.Terminal || state.Outcome != model.OutcomeFailed {
		t.Fatalf("state = %+v, want terminal FAILED", state)
	}
}

// stageTransitionFixture builds a request whose approval is exactly what the
// chain position demands, so each test can spoil one field at a time.
func stageTransitionFixture(state StageState, from, to model.Stage) StageTransitionRequest {
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	a := model.ApprovalBinding{
		RunID:                    "run",
		TransitionDigest:         "transition",
		CurrentStage:             from,
		ProposedTargetStage:      to,
		RepositoryIdentityDigest: "identity",
		BaseRevisionDigest:       "base",
		StageTimeDigest:          stageTime(state),
		AcceptedPlanDigest:       "plan",
		ChallengeNonce:           "fresh-nonce",
		Approver:                 "operator-confirmed:lennon",
		Timestamp:                now.Format(time.RFC3339),
	}
	return StageTransitionRequest{State: state, Expected: a, Presented: a, Now: now, MaxAge: 5 * time.Minute}
}

func stageTime(state StageState) string {
	return fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1)
}

func TestEvaluateStageTransitionAcceptsLegalApprovedEdge(t *testing.T) {
	state := ProjectStageState(nil)
	req := stageTransitionFixture(state, model.StageGrounding, model.StageAcceptanceCriteria)
	res := EvaluateStageTransition(req)
	if res.Decision != DecisionLegalPendingApproval {
		t.Fatalf("res = %+v, want legal", res)
	}
	if res.FromStage != model.StageGrounding || res.ToStage != model.StageAcceptanceCriteria {
		t.Fatalf("res = %+v", res)
	}
	if res.ControlState != model.ControlApproved || res.MachineAdvanced {
		t.Fatalf("res = %+v", res)
	}
}

func TestEvaluateStageTransitionRejects(t *testing.T) {
	base := ProjectStageState(nil)
	cases := []struct {
		name  string
		spoil func(*StageTransitionRequest)
		want  Reason
	}{
		{"unlisted edge", func(r *StageTransitionRequest) {
			r.Expected.ProposedTargetStage = model.StageVerification
			r.Presented.ProposedTargetStage = model.StageVerification
		}, ReasonUnlistedEdge},
		{"approval raised from another stage", func(r *StageTransitionRequest) {
			r.Expected.CurrentStage = model.StagePlanning
			r.Presented.CurrentStage = model.StagePlanning
			r.Expected.ProposedTargetStage = model.StageRouteSelection
			r.Presented.ProposedTargetStage = model.StageRouteSelection
		}, ReasonStageMismatch},
		{"replayed nonce", func(r *StageTransitionRequest) {
			r.State.UsedNonces = map[string]bool{"fresh-nonce": true}
		}, ReasonReusedNonce},
		{"approval bound to another chain position", func(r *StageTransitionRequest) {
			r.Expected.StageTimeDigest = "someone-elses-position:1"
			r.Presented.StageTimeDigest = "someone-elses-position:1"
		}, ReasonStaleApproval},
		{"presented differs from expected", func(r *StageTransitionRequest) {
			r.Presented.AcceptedPlanDigest = "a different plan"
		}, ReasonStaleApproval},
		{"approval bound to another worktree", func(r *StageTransitionRequest) {
			r.Expected.WorktreeDigest = "worktree-the-human-approved"
			r.Presented.WorktreeDigest = "worktree-the-human-approved"
			r.WorktreeDigest = "worktree-now"
		}, ReasonStaleWorktreeBinding},
		{"approval bound to a clean worktree", func(r *StageTransitionRequest) {
			r.Expected.WorktreeDigest = "worktree-the-human-approved"
			r.Presented.WorktreeDigest = "worktree-the-human-approved"
			r.WorktreeDigest = ""
		}, ReasonStaleWorktreeBinding},
		{"expired approval", func(r *StageTransitionRequest) {
			r.Now = r.Now.Add(time.Hour)
		}, ReasonStaleApproval},
		{"terminal run", func(r *StageTransitionRequest) {
			r.State.Terminal = true
		}, ReasonRunTerminal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := stageTransitionFixture(base, model.StageGrounding, model.StageAcceptanceCriteria)
			tc.spoil(&req)
			res := EvaluateStageTransition(req)
			if res.Decision != DecisionRejected || res.Reason != tc.want {
				t.Fatalf("res = %+v, want rejection %s", res, tc.want)
			}
		})
	}
}

// TestEvaluateStageTransitionAcceptsWorktreeBoundApproval is the other half
// of the section 6 amendment: a worktree-bound approval whose digest matches
// the live worktree is legal, so the binding is enforced exactly when
// populated and never otherwise.
func TestEvaluateStageTransitionAcceptsWorktreeBoundApproval(t *testing.T) {
	state := ProjectStageState(nil)
	req := stageTransitionFixture(state, model.StageGrounding, model.StageAcceptanceCriteria)
	req.Expected.WorktreeDigest = "worktree-now"
	req.Presented.WorktreeDigest = "worktree-now"
	req.WorktreeDigest = "worktree-now"
	if res := EvaluateStageTransition(req); res.Decision != DecisionLegalPendingApproval {
		t.Fatalf("res = %+v, want legal", res)
	}
}

// TestEvaluateStageTransitionEnforcesRequiredEvidence is the "no evidence, no
// transition" rule made concrete: a plan-declared required digest must be in
// the chain, and must have been captured at the revision the approval binds.
func TestEvaluateStageTransitionEnforcesRequiredEvidence(t *testing.T) {
	newState := func(e ...model.EvidenceRef) StageState {
		s := ProjectStageState(nil)
		s.Evidence = e
		return s
	}
	fresh := model.EvidenceRef{OutputDigest: "digest-1", BaseRevision: "base", HeadRevision: "head"}
	stale := model.EvidenceRef{OutputDigest: "digest-1", BaseRevision: "an-older-base", HeadRevision: "head"}

	cases := []struct {
		name  string
		state StageState
		want  Reason
	}{
		{"evidence present at the bound revision", newState(fresh), ReasonNone},
		{"required evidence never captured", newState(), ReasonMissingRequiredEvidence},
		{"evidence captured before the code changed", newState(stale), ReasonStaleRequiredEvidence},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := stageTransitionFixture(tc.state, model.StageGrounding, model.StageAcceptanceCriteria)
			req.Expected.RequiredEvidenceDigests = []string{"digest-1"}
			req.Presented.RequiredEvidenceDigests = []string{"digest-1"}
			res := EvaluateStageTransition(req)
			if tc.want == ReasonNone {
				if res.Decision != DecisionLegalPendingApproval {
					t.Fatalf("res = %+v, want legal", res)
				}
				return
			}
			if res.Decision != DecisionRejected || res.Reason != tc.want {
				t.Fatalf("res = %+v, want rejection %s", res, tc.want)
			}
		})
	}
}

// TestEvaluateStageTransitionEnforcesEveryRequiredDigest: a plan may declare
// several required evidence digests, and satisfying the first must not excuse
// the rest.
func TestEvaluateStageTransitionEnforcesEveryRequiredDigest(t *testing.T) {
	state := ProjectStageState(nil)
	state.Evidence = []model.EvidenceRef{
		{OutputDigest: "digest-1", BaseRevision: "base"},
		{OutputDigest: "digest-2", BaseRevision: "an-older-base"},
	}
	cases := []struct {
		name     string
		required []string
		want     Reason
	}{
		{"all satisfied", []string{"digest-1"}, ReasonNone},
		{"second never captured", []string{"digest-1", "digest-3"}, ReasonMissingRequiredEvidence},
		{"second captured at another revision", []string{"digest-1", "digest-2"}, ReasonStaleRequiredEvidence},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := stageTransitionFixture(state, model.StageGrounding, model.StageAcceptanceCriteria)
			req.Expected.RequiredEvidenceDigests = tc.required
			req.Presented.RequiredEvidenceDigests = tc.required
			res := EvaluateStageTransition(req)
			if tc.want == ReasonNone {
				if res.Decision != DecisionLegalPendingApproval {
					t.Fatalf("res = %+v, want legal", res)
				}
				return
			}
			if res.Decision != DecisionRejected || res.Reason != tc.want {
				t.Fatalf("res = %+v, want rejection %s", res, tc.want)
			}
		})
	}
}
