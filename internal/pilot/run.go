// Package pilot composes the bounded host-trusted local pilot boundary.
package pilot

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

// snapshot is the exact repository state evidence is bound to. worktree is
// empty for a clean worktree, which is the revision-reproducible case; it is
// set only for an approved dirty-worktree capture, where the committed
// revision alone no longer describes what the command actually saw.
type snapshot struct{ head, diff, worktree string }

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

// statusPaths parses `git status --porcelain -z` into the set of paths git
// reports as changed. The NUL-delimited form is used because the default
// output quotes and escapes unusual paths, which would make the digest depend
// on how a path renders rather than on what it is. A rename or copy entry is
// followed by its source path as a separate record; both paths are kept,
// because both are part of what changed.
func statusPaths(status string) []string {
	fields := strings.Split(status, "\x00")
	var paths []string
	for i := 0; i < len(fields); i++ {
		entry := fields[i]
		if len(entry) < 4 {
			continue
		}
		// Porcelain v1 entry: two status codes, a space, then the path.
		code, path := entry[:2], entry[3:]
		paths = append(paths, path)
		if strings.ContainsAny(code, "RC") && i+1 < len(fields) {
			i++
			if fields[i] != "" {
				paths = append(paths, fields[i])
			}
		}
	}
	sort.Strings(paths)
	return paths
}

// worktreeContentDigest hashes the exact content of a dirty worktree: the head
// revision, then every changed path in sorted order with the digest of its
// current bytes, or an explicit absence marker for a deleted path. Entries are
// domain-separated and length-framed so no two different worktrees can produce
// the same digest by concatenation.
//
// It reads only the paths git names and writes nothing, so taking the digest
// never mutates the repository or its object store.
func worktreeContentDigest(root, head, status string) (string, error) {
	h := sha256.New()
	h.Write([]byte("hma-worktree-v1\x00" + head + "\x00"))
	for _, path := range statusPaths(status) {
		h.Write([]byte("path\x00"))
		writeFramed(h, []byte(path))
		full := filepath.Join(root, filepath.FromSlash(path))
		content, err := os.ReadFile(full)
		switch {
		case err == nil:
			h.Write([]byte("content\x00"))
			writeFramed(h, content)
		case errors.Is(err, os.ErrNotExist):
			h.Write([]byte("absent\x00"))
		default:
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeFramed(h hash.Hash, data []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	h.Write(length[:])
	h.Write(data)
}

// snapshotRepo captures the repository state. A dirty worktree is refused
// unless the host has explicitly approved this dirty-worktree use, per
// architecture section 14: such evidence is admissible only when the record
// carries the exact worktree-content digest, and it is never
// revision-reproducible.
func snapshotRepo(ctx context.Context, root, baseRevision string, dirtyApproved bool) (snapshot, error) {
	head, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return snapshot{}, err
	}
	status, err := gitOutput(ctx, root, "status", "--porcelain", "-z", "-uall")
	if err != nil {
		return snapshot{}, err
	}
	dirty := strings.Trim(status, "\x00") != ""
	if dirty && !dirtyApproved {
		return snapshot{}, errors.New("dirty worktree policy violation")
	}
	diff, err := gitOutput(ctx, root, "diff", "--binary", baseRevision+".."+trim(head), "--")
	if err != nil {
		return snapshot{}, err
	}
	sum := sha256.Sum256([]byte(diff))
	snap := snapshot{head: trim(head), diff: hex.EncodeToString(sum[:])}
	if dirty {
		snap.worktree, err = worktreeContentDigest(root, snap.head, status)
		if err != nil {
			return snapshot{}, err
		}
	}
	return snap, nil
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
	if in.DirtyWorktreeApproved && in.ExpectedWorktreeDigest == "" {
		return Result{}, errors.New("dirty worktree approval requires an expected worktree digest")
	}
	if !in.DirtyWorktreeApproved && in.ExpectedWorktreeDigest != "" {
		return Result{}, errors.New("worktree digest supplied without dirty worktree approval")
	}
	before, err := snapshotRepo(ctx, root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return Result{}, err
	}
	if before.worktree != in.ExpectedWorktreeDigest {
		return Result{}, errors.New("repository identity is stale")
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
	cap, err := evidence.Capture(ctx, evidence.Request{Executable: cmd.Executable, Argv: cmd.Argv, WorkingDir: root, RepositoryRoot: root, MaxOutputBytes: 1 << 20, RepositoryIdentity: in.RepositoryIdentity, BaseRevision: in.BaseRevision, HeadRevision: before.head, DiffDigest: before.diff, WorktreeDigest: before.worktree, CapturingActor: in.Actor})
	if err != nil {
		return Result{}, err
	}
	after, err := snapshotRepo(ctx, root, in.BaseRevision, in.DirtyWorktreeApproved)
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
