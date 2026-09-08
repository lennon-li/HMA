// Package transition records a human-approved stage transition into a run's
// append-only chain.
//
// It is the only path by which a run changes stage, and it changes one only
// because a human already decided to: the decision arrives as a single-use,
// challenge-bound approval binding, and this package verifies that the
// decision is fresh, is bound to the repository state that actually exists
// now, and names an edge the architecture permits. It never selects a target,
// substitutes an approval, waives a requirement, or advances a run on its own.
// Recording a human's decision is not making one.
package transition

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// Input is the host's explicit description of one approved transition. The
// actor is not a separate field: the only act being recorded is the human
// decision, so the record's actor is the approver.
type Input struct {
	RunID              string `json:"run_id"`
	RepositoryRoot     string `json:"repository_root"`
	RepositoryIdentity string `json:"repository_identity"`
	BaseRevision       string `json:"base_revision"`
	// DirtyWorktreeApproved and ExpectedWorktreeDigest carry the same
	// meaning as in a pilot capture: a dirty worktree is refused unless the
	// human approved this dirty use and bound the exact content.
	DirtyWorktreeApproved  bool                  `json:"dirty_worktree_approved,omitempty"`
	ExpectedWorktreeDigest string                `json:"expected_worktree_digest,omitempty"`
	ExpectedApproval       model.ApprovalBinding `json:"expected_approval"`
	Approval               model.ApprovalBinding `json:"approval"`
	ApprovalNow            time.Time             `json:"approval_now"`
	ApprovalMaxAgeSeconds  int64                 `json:"approval_max_age_seconds"`
}

// Result reports what was recorded. MachineAdvanced is always false: the
// machine wrote a record, it did not decide anything.
type Result struct {
	Decision                   engine.Decision `json:"decision"`
	FromStage                  model.Stage     `json:"from_stage"`
	ToStage                    model.Stage     `json:"to_stage"`
	Sequence                   int64           `json:"sequence"`
	HeadAnchor                 string          `json:"head_anchor"`
	RequiresFreshHumanApproval bool            `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool            `json:"machine_advanced"`
}

// Apply verifies and records one approved stage transition.
func Apply(ctx context.Context, in Input, storeDir string) (Result, error) {
	if in.RunID == "" || in.RepositoryIdentity == "" || in.BaseRevision == "" {
		return Result{}, errors.New("missing host input")
	}
	// An empty repository root or store path would silently resolve to the
	// process working directory, applying this to whatever repository the
	// caller happens to be standing in.
	if err := repostate.StoreOutsideRepository(in.RepositoryRoot, storeDir); err != nil {
		return Result{}, err
	}
	root, err := filepath.Abs(in.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	sd, err := filepath.Abs(storeDir)
	if err != nil {
		return Result{}, err
	}
	if in.DirtyWorktreeApproved && in.ExpectedWorktreeDigest == "" {
		return Result{}, errors.New("dirty worktree approval requires an expected worktree digest")
	}
	if !in.DirtyWorktreeApproved && in.ExpectedWorktreeDigest != "" {
		return Result{}, errors.New("worktree digest supplied without dirty worktree approval")
	}

	// The approval binds a repository state; the transition is admissible
	// only against the repository that is actually there now.
	snap, err := repostate.Snapshot(ctx, root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return Result{}, err
	}
	// An approval that binds a worktree digest (the section 6 amendment)
	// carries the human's decision about exactly which uncommitted content
	// the transition covers, so it is compared against the live worktree
	// before the host-level declaration and the refusal names the binding
	// that failed. A dirty worktree whose approval record binds no digest
	// of its own is still governed by the host-level declaration below.
	if in.ExpectedApproval.WorktreeDigest != "" && in.ExpectedApproval.WorktreeDigest != snap.Worktree {
		return Result{}, fmt.Errorf("transition rejected: %s", engine.ReasonStaleWorktreeBinding)
	}
	if snap.Worktree != in.ExpectedWorktreeDigest {
		return Result{}, errors.New("repository identity is stale")
	}
	if err := repostate.VerifyBase(ctx, root, in.BaseRevision); err != nil {
		return Result{}, errors.New("repository identity is stale")
	}

	expected := in.ExpectedApproval
	if expected.RunID != in.RunID || expected.RepositoryIdentityDigest != in.RepositoryIdentity || expected.BaseRevisionDigest != in.BaseRevision {
		return Result{}, errors.New("expected approval binding is stale")
	}
	// Only the four stages whose approvals bind a produced head and diff are
	// checked against them. A packet becomes stale only when a field it
	// actually binds changes (architecture section 6), so comparing a head
	// an earlier-stage approval never bound would invent a staleness rule
	// the contract does not have.
	if model.ApprovalBindsHeadDiff(expected.CurrentStage) {
		if expected.ProducedHeadDigest != snap.Head || expected.DiffDigest != snap.Diff {
			return Result{}, errors.New("expected approval binding is stale")
		}
	}

	s := store.New(sd)
	records, err := s.Load(in.RunID)
	if err != nil {
		return Result{}, err
	}
	state := engine.ProjectStageState(records)

	res := engine.EvaluateStageTransition(engine.StageTransitionRequest{
		State:          state,
		Expected:       expected,
		Presented:      in.Approval,
		Now:            in.ApprovalNow,
		MaxAge:         time.Duration(in.ApprovalMaxAgeSeconds) * time.Second,
		WorktreeDigest: snap.Worktree,
	})
	if res.Decision != engine.DecisionLegalPendingApproval {
		return Result{}, fmt.Errorf("transition rejected: %s", res.Reason)
	}

	// The record is filed under the stage being approved, and carries the
	// approval whose proposed target the chain will replay as the run's new
	// stage. Store.Append computes the head anchor and then validates the
	// complete record.
	rec := model.Record{
		Kind:    model.KindStageTransition,
		Version: 1,
		Metadata: model.RecordMetadata{
			RunID:           in.RunID,
			Sequence:        state.Sequence + 1,
			PredecessorHash: state.PredecessorHead,
			Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
			Actor:           in.Approval.Approver,
		},
		Stage:    res.FromStage,
		Control:  model.ControlApproved,
		Approval: &in.Approval,
	}
	if err := s.Append(&rec); err != nil {
		return Result{}, err
	}

	return Result{
		Decision:                   res.Decision,
		FromStage:                  res.FromStage,
		ToStage:                    res.ToStage,
		Sequence:                   rec.Metadata.Sequence,
		HeadAnchor:                 rec.Metadata.HeadAnchor,
		RequiresFreshHumanApproval: true,
		MachineAdvanced:            false,
	}, nil
}
