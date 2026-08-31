// Package pilot composes the bounded host-trusted local pilot boundary.
package pilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

type Command struct {
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
}
type Input struct {
	RunID                 string                `json:"run_id"`
	RepositoryRoot        string                `json:"repository_root"`
	RepositoryIdentity    string                `json:"repository_identity"`
	BaseRevision          string                `json:"base_revision"`
	ExpectedHeadRevision  string                `json:"expected_head_revision"`
	ExpectedDiffDigest    string                `json:"expected_diff_digest"`
	Actor                 string                `json:"actor"`
	ExpectedApproval      model.ApprovalBinding `json:"expected_approval"`
	Approval              model.ApprovalBinding `json:"approval"`
	ApprovalNow           time.Time             `json:"approval_now"`
	ApprovalMaxAgeSeconds int64                 `json:"approval_max_age_seconds"`
	EvidenceCommands      []Command             `json:"evidence_commands"`
}
type Result struct {
	Decision                   engine.Decision   `json:"decision"`
	RequiresFreshHumanApproval bool              `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool              `json:"machine_advanced"`
	Evidence                   model.EvidenceRef `json:"evidence"`
}
type snapshot struct{ head, diff string }

func trim(s string) string { return strings.TrimSpace(s) }
func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	b, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(b), nil
}
func snapshotRepo(ctx context.Context, root, baseRevision string) (snapshot, error) {
	head, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return snapshot{}, err
	}
	status, err := gitOutput(ctx, root, "status", "--porcelain", "-uall")
	if err != nil {
		return snapshot{}, err
	}
	if trim(status) != "" {
		return snapshot{}, errors.New("dirty worktree policy violation")
	}
	diff, err := gitOutput(ctx, root, "diff", "--binary", baseRevision+".."+trim(head), "--")
	if err != nil {
		return snapshot{}, err
	}
	sum := sha256.Sum256([]byte(diff))
	return snapshot{trim(head), hex.EncodeToString(sum[:])}, nil
}
func inside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func Run(ctx context.Context, in Input, storeDir string) (Result, error) {
	if len(in.EvidenceCommands) != 1 {
		return Result{}, errors.New("exactly one evidence command required")
	}
	if in.RunID == "" || in.Actor == "" || in.RepositoryIdentity == "" {
		return Result{}, errors.New("missing host input")
	}
	root, err := filepath.Abs(in.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	sd, err := filepath.Abs(storeDir)
	if err != nil {
		return Result{}, err
	}
	if inside(root, sd) {
		return Result{}, errors.New("store must be outside target repository")
	}
	before, err := snapshotRepo(ctx, root, in.BaseRevision)
	if err != nil {
		return Result{}, err
	}
	base, err := gitOutput(ctx, root, "rev-parse", "--verify", in.BaseRevision+"^{commit}")
	if err != nil || trim(base) != in.BaseRevision || before.head != in.ExpectedHeadRevision || before.diff != in.ExpectedDiffDigest {
		return Result{}, errors.New("repository identity is stale")
	}
	expected := in.ExpectedApproval
	if expected.RunID != in.RunID || expected.RepositoryIdentityDigest != in.RepositoryIdentity || expected.BaseRevisionDigest != in.BaseRevision || expected.ProducedHeadDigest != before.head || expected.DiffDigest != before.diff {
		return Result{}, errors.New("expected approval binding is stale")
	}
	s := store.New(sd)
	records, err := s.Load(in.RunID)
	if err != nil {
		return Result{}, err
	}
	predecessorHead := ""
	if len(records) > 0 {
		predecessorHead = records[len(records)-1].Metadata.HeadAnchor
	}
	nextSequence := int64(len(records) + 1)
	usedNonces := make(map[string]bool)
	for i := range records {
		if records[i].Approval != nil {
			usedNonces[records[i].Approval.ChallengeNonce] = true
		}
	}
	ar := engine.EvaluateApprovalBinding(engine.ApprovalBindingRequest{Expected: expected, Presented: in.Approval, PredecessorHead: predecessorHead, Sequence: nextSequence, UsedNonces: usedNonces, Now: in.ApprovalNow, MaxAge: time.Duration(in.ApprovalMaxAgeSeconds) * time.Second})
	if ar.Decision != engine.DecisionLegalPendingApproval {
		return Result{}, fmt.Errorf("approval rejected: %s", ar.Reason)
	}
	cmd := in.EvidenceCommands[0]
	cap, err := evidence.Capture(ctx, evidence.Request{Executable: cmd.Executable, Argv: cmd.Argv, WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1 << 20, RepositoryIdentity: in.RepositoryIdentity, BaseRevision: in.BaseRevision, HeadRevision: before.head, DiffDigest: before.diff, CapturingActor: in.Actor})
	if err != nil {
		return Result{}, err
	}
	after, err := snapshotRepo(ctx, root, in.BaseRevision)
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
