package engine

import (
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

func validApproval() (ApprovalBindingRequest, model.ApprovalBinding) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	a := model.ApprovalBinding{RunID: "run", TransitionDigest: "transition", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification, RepositoryIdentityDigest: "repo", BaseRevisionDigest: "base", StageTimeDigest: "head:7", AcceptedPlanDigest: "plan", RequiredEvidenceDigests: []string{"evidence"}, ActiveWaivers: model.WaiverApplicabilitySet{"waiver"}, ChallengeNonce: "nonce", Approver: "operator-confirmed:lennon", Timestamp: now.Format(time.RFC3339), ProducedHeadDigest: "revision", DiffDigest: "diff"}
	r := ApprovalBindingRequest{Expected: a, Presented: a, PredecessorHead: "head", Sequence: 7, Now: now.Add(time.Minute), MaxAge: 5 * time.Minute}
	return r, a
}

func TestEvaluateApprovalBinding(t *testing.T) {
	req, _ := validApproval()
	got := EvaluateApprovalBinding(req)
	if got.Decision != DecisionLegalPendingApproval || got.MachineAdvanced || !got.RequiresFreshHumanApproval {
		t.Fatalf("valid approval = %+v", got)
	}

	cases := map[string]func(*ApprovalBindingRequest){
		"empty":             func(r *ApprovalBindingRequest) { r.Presented = model.ApprovalBinding{} },
		"stale predecessor": func(r *ApprovalBindingRequest) { r.PredecessorHead = "old" },
		"stale sequence":    func(r *ApprovalBindingRequest) { r.Sequence = 8 },
		"wrong repository":  func(r *ApprovalBindingRequest) { r.Presented.RepositoryIdentityDigest = "other" },
		"wrong base":        func(r *ApprovalBindingRequest) { r.Presented.BaseRevisionDigest = "other" },
		"wrong diff":        func(r *ApprovalBindingRequest) { r.Presented.DiffDigest = "other" },
		"reused nonce":      func(r *ApprovalBindingRequest) { r.UsedNonces = map[string]bool{"nonce": true} },
		"expired":           func(r *ApprovalBindingRequest) { r.Now = r.Now.Add(10 * time.Minute) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := validApproval()
			mutate(&r)
			got := EvaluateApprovalBinding(r)
			if got.Decision != DecisionRejected || got.MachineAdvanced {
				t.Fatalf("got %+v", got)
			}
			if name == "reused nonce" && got.Reason != ReasonReusedNonce {
				t.Fatalf("reason = %s, want %s", got.Reason, ReasonReusedNonce)
			}
		})
	}
}

// TestEvaluateApprovalBindingEnforcesWorktreeBinding is the section 6
// amendment made concrete: an approval that binds a worktree digest is
// refused when the current worktree digest differs from the approved one.
func TestEvaluateApprovalBindingEnforcesWorktreeBinding(t *testing.T) {
	bound := func() ApprovalBindingRequest {
		r, a := validApproval()
		a.WorktreeDigest = "worktree-now"
		r.Expected = a
		r.Presented = a
		r.WorktreeDigest = "worktree-now"
		return r
	}

	if got := EvaluateApprovalBinding(bound()); got.Decision != DecisionLegalPendingApproval {
		t.Fatalf("matching worktree binding rejected: %+v", got)
	}

	stale := bound()
	stale.WorktreeDigest = "worktree-the-human-approved"
	if got := EvaluateApprovalBinding(stale); got.Decision != DecisionRejected || got.Reason != ReasonStaleWorktreeBinding {
		t.Fatalf("got %+v, want rejection %s", got, ReasonStaleWorktreeBinding)
	}

	// A clean worktree has an empty digest, so an approval that binds a
	// worktree digest can never match one.
	clean := bound()
	clean.WorktreeDigest = ""
	if got := EvaluateApprovalBinding(clean); got.Reason != ReasonStaleWorktreeBinding {
		t.Fatalf("reason = %q, want %q", got.Reason, ReasonStaleWorktreeBinding)
	}

	// The binding is optional: an approval that binds no worktree digest is
	// not newly invalidated by a dirty worktree, which it never bound. That
	// case belongs to the host-level dirty-worktree approval.
	unbound, _ := validApproval()
	unbound.WorktreeDigest = "worktree-now"
	if got := EvaluateApprovalBinding(unbound); got.Decision != DecisionLegalPendingApproval {
		t.Fatalf("unbound approval rejected: %+v", got)
	}
}
