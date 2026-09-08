// Package pilot composes the bounded host-trusted local pilot boundary.
package pilot

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

type Command struct {
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
}
type Input struct {
	RunID                string `json:"run_id"`
	RepositoryRoot       string `json:"repository_root"`
	RepositoryIdentity   string `json:"repository_identity"`
	BaseRevision         string `json:"base_revision"`
	ExpectedHeadRevision string `json:"expected_head_revision"`
	ExpectedDiffDigest   string `json:"expected_diff_digest"`
	// DirtyWorktreeApproved records that the human approved this specific
	// dirty-worktree use. Without it a dirty worktree is refused outright.
	DirtyWorktreeApproved bool `json:"dirty_worktree_approved,omitempty"`
	// ExpectedWorktreeDigest is the worktree content the human approved. It
	// is required with DirtyWorktreeApproved, so approving "dirty" never
	// means approving whatever happens to be on disk at capture time.
	ExpectedWorktreeDigest string                `json:"expected_worktree_digest,omitempty"`
	Actor                  string                `json:"actor"`
	ExpectedApproval       model.ApprovalBinding `json:"expected_approval"`
	Approval               model.ApprovalBinding `json:"approval"`
	ApprovalNow            time.Time             `json:"approval_now"`
	ApprovalMaxAgeSeconds  int64                 `json:"approval_max_age_seconds"`
	EvidenceCommands       []Command             `json:"evidence_commands"`
}
type Result struct {
	Decision                   engine.Decision   `json:"decision"`
	RequiresFreshHumanApproval bool              `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool              `json:"machine_advanced"`
	Evidence                   model.EvidenceRef `json:"evidence"`
}

func Run(ctx context.Context, in Input, storeDir string) (Result, error) {
	if len(in.EvidenceCommands) != 1 {
		return Result{}, errors.New("exactly one evidence command required")
	}
	if in.RunID == "" || in.Actor == "" || in.RepositoryIdentity == "" {
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
	before, err := repostate.Snapshot(ctx, root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return Result{}, err
	}
	// An approval that binds a worktree digest (the section 6 amendment)
	// names the exact uncommitted content the human approved; the live
	// worktree must still match it, and the refusal names the binding.
	if in.ExpectedApproval.WorktreeDigest != "" && in.ExpectedApproval.WorktreeDigest != before.Worktree {
		return Result{}, fmt.Errorf("approval rejected: %s", engine.ReasonStaleWorktreeBinding)
	}
	if before.Worktree != in.ExpectedWorktreeDigest {
		return Result{}, errors.New("repository identity is stale")
	}
	if err := repostate.VerifyBase(ctx, root, in.BaseRevision); err != nil || before.Head != in.ExpectedHeadRevision || before.Diff != in.ExpectedDiffDigest {
		return Result{}, errors.New("repository identity is stale")
	}
	expected := in.ExpectedApproval
	if expected.RunID != in.RunID || expected.RepositoryIdentityDigest != in.RepositoryIdentity || expected.BaseRevisionDigest != in.BaseRevision || expected.ProducedHeadDigest != before.Head || expected.DiffDigest != before.Diff {
		return Result{}, errors.New("expected approval binding is stale")
	}
	s := store.New(sd)
	records, err := s.Load(in.RunID)
	if err != nil {
		return Result{}, err
	}
	// One projection defines what the chain has spent. Collecting approval
	// nonces alone would let a nonce already burned on a waiver be replayed
	// as an approval nonce.
	state := engine.ProjectStageState(records)
	predecessorHead := state.PredecessorHead
	nextSequence := state.Sequence + 1
	ar := engine.EvaluateApprovalBinding(engine.ApprovalBindingRequest{Expected: expected, Presented: in.Approval, PredecessorHead: predecessorHead, Sequence: nextSequence, UsedNonces: state.UsedNonces, Now: in.ApprovalNow, MaxAge: time.Duration(in.ApprovalMaxAgeSeconds) * time.Second, WorktreeDigest: before.Worktree})
	if ar.Decision != engine.DecisionLegalPendingApproval {
		return Result{}, fmt.Errorf("approval rejected: %s", ar.Reason)
	}
	cmd := in.EvidenceCommands[0]
	cap, err := evidence.Capture(ctx, evidence.Request{Executable: cmd.Executable, Argv: cmd.Argv, WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1 << 20, RepositoryIdentity: in.RepositoryIdentity, BaseRevision: in.BaseRevision, HeadRevision: before.Head, DiffDigest: before.Diff, WorktreeDigest: before.Worktree, CapturingActor: in.Actor})
	if err != nil {
		return Result{}, err
	}
	after, err := repostate.Snapshot(ctx, root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return Result{}, err
	}
	if after != before {
		return Result{}, errors.New("repository changed during evidence capture")
	}
	approvalRecord := model.Record{Kind: model.KindStageTransition, Version: 1, Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: nextSequence, PredecessorHash: predecessorHead, Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Actor: in.Actor}, Stage: in.Approval.CurrentStage, Control: model.ControlReadyForReview, Approval: &in.Approval}
	if err := s.Append(&approvalRecord); err != nil {
		return Result{}, err
	}
	evidenceRecord := model.Record{Kind: model.KindEvidence, Version: 1, Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: nextSequence + 1, PredecessorHash: approvalRecord.Metadata.HeadAnchor, Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Actor: in.Actor}, Stage: in.Approval.CurrentStage, Control: model.ControlReadyForReview, Evidence: &cap.Evidence}
	if err := s.Append(&evidenceRecord); err != nil {
		return Result{}, err
	}
	return Result{Decision: engine.DecisionLegalPendingApproval, RequiresFreshHumanApproval: true, MachineAdvanced: false, Evidence: cap.Evidence}, nil
}
