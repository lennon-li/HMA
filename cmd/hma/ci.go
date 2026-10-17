package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lennon-li/HMA/internal/ci/github"
	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// argvFlag collects a repeated --arg value into an argument vector. Arguments
// are passed as separate values so no shell string is representable.
type argvFlag []string

func (a *argvFlag) String() string { return fmt.Sprint(*a) }
func (a *argvFlag) Set(v string) error {
	*a = append(*a, v)
	return nil
}

type ciResult struct {
	Verification  engine.ResolutionResult `json:"verification"`
	Evidence      *model.EvidenceRef      `json:"evidence,omitempty"`
	RecordWritten bool                    `json:"record_written"`
}

// runCIGithub verifies that the CI runner is on the exact approved repository
// and revision, and, unless --verify-only is given, captures real evidence by
// executing one explicit command. HMA does not manufacture an evidence record
// from environment variables alone: a record is written only when a command
// actually ran and produced an exit code and an output digest.
func runCIGithub(args []string) error {
	fs := flag.NewFlagSet("ci-github", flag.ContinueOnError)
	storePath := fs.String("store", "", "host store directory")
	runID := fs.String("run_id", "", "run ID to verify against")
	verifyOnly := fs.Bool("verify-only", false, "verify repository and revision without recording evidence")
	repoRoot := fs.String("repo-root", "", "checked-out repository root for evidence capture")
	executable := fs.String("exec", "", "explicit executable to run for evidence capture")
	var argv argvFlag
	fs.Var(&argv, "arg", "one argument for --exec; repeat for each argument")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePath == "" || *runID == "" || fs.NArg() != 0 {
		return errors.New("usage: hma ci github --store <directory> --run_id <id> [--verify-only | --repo-root <dir> --exec <prog> [--arg <a>]...]")
	}
	if !*verifyOnly && (*executable == "" || *repoRoot == "") {
		return errors.New("evidence capture requires --repo-root and --exec; pass --verify-only to check the revision without recording evidence")
	}

	env, err := github.Load()
	if err != nil {
		return fmt.Errorf("github ci adapter: %w", err)
	}
	// Evidence is written to the store, so the store must not be one the
	// tested checkout can rewrite. A repository-selecting variable such as
	// GIT_DIR would let the command read a repository other than the one HMA
	// snapshots, so CI refuses to run with one set.
	if !*verifyOnly {
		if v := repostate.SelectingVariableSet(os.Environ()); v != "" {
			return fmt.Errorf("refusing evidence capture: repository-selecting variable %s is set", v)
		}
		if err := repostate.StoreOutsideRepository(*repoRoot, *storePath); err != nil {
			return err
		}
	}

	s := store.New(*storePath)
	records, err := s.Load(*runID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.New("no records found for run_id")
	}
	// The stage comes from the projection, not the last record: a
	// stage-transition record carries its from-stage and a resolution record
	// may carry none. The approval packet verified against is the latest one
	// for leaving the current stage, recorded since the run entered it, so a
	// packet from an earlier visit to the stage is never reused.
	state := engine.ProjectStageState(records)
	if state.Terminal {
		return fmt.Errorf("run is terminal (%s)", state.Outcome)
	}
	entered := -1
	for i, r := range records {
		if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil && r.Approval.ProposedTargetStage == state.CurrentStage {
			entered = i
		}
	}
	var approval *model.ApprovalBinding
	for i := len(records) - 1; i > entered; i-- {
		if a := records[i].Approval; a != nil && a.CurrentStage == state.CurrentStage {
			approval = a
			break
		}
	}
	if approval == nil {
		return errors.New("no approval binding found in chain to verify against")
	}

	res := engine.EvaluateCIVerification(engine.CIVerificationRequest{
		CIRepository:               env.Repository,
		CISHA:                      env.SHA,
		CurrentStage:               state.CurrentStage,
		ExpectedRepositoryIdentity: approval.RepositoryIdentityDigest,
		ExpectedProducedHead:       approval.ProducedHeadDigest,
	})
	if res.Decision != engine.DecisionLegalPendingApproval {
		return errors.New(string(res.Reason))
	}

	if *verifyOnly {
		return json.NewEncoder(os.Stdout).Encode(ciResult{Verification: res})
	}

	// The GitHub-supplied actor is unauthenticated environment data, so it is
	// recorded with its provenance prefix rather than as a bare human identity.
	actor := "github-actions:" + env.Actor

	// Environment strings only claim a revision. The checkout must be a clean
	// Git worktree at the approved base, head, and diff, and the command must
	// leave it so; otherwise the evidence would describe content HMA never saw.
	ctx := context.Background()
	before, err := ciCheckoutState(ctx, *repoRoot, approval, env.SHA)
	if err != nil {
		return err
	}
	captured, err := evidence.Capture(ctx, evidence.Request{
		Executable:         *executable,
		Argv:               argv,
		WorkingDir:         *repoRoot,
		RepositoryRoot:     *repoRoot,
		MaxOutputBytes:     1 << 20,
		RepositoryIdentity: env.Repository,
		BaseRevision:       approval.BaseRevisionDigest,
		HeadRevision:       before.Head,
		DiffDigest:         before.Diff,
		CapturingActor:     actor,
	})
	if err != nil {
		return err
	}
	after, err := ciCheckoutState(ctx, *repoRoot, approval, env.SHA)
	if err != nil {
		return fmt.Errorf("checkout changed during evidence capture: %w", err)
	}
	if after != before {
		return errors.New("checkout changed during evidence capture")
	}

	rec := model.Record{
		Kind:    model.KindEvidence,
		Version: 1,
		Metadata: model.RecordMetadata{
			RunID:           *runID,
			Sequence:        state.Sequence + 1,
			PredecessorHash: state.PredecessorHead,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Actor:           actor,
		},
		Stage:    state.CurrentStage,
		Evidence: &captured.Evidence,
	}
	if err := s.Append(&rec); err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(ciResult{
		Verification:  res,
		Evidence:      &captured.Evidence,
		RecordWritten: true,
	})
}

// ciCheckoutState snapshots root as a clean Git worktree and requires it to be
// the approved base, the CI head, and the approved produced head and diff.
func ciCheckoutState(ctx context.Context, root string, approval *model.ApprovalBinding, ciSHA string) (repostate.State, error) {
	// CI tests only clean checkouts; an approval that binds dirty content
	// describes bytes a clean checkout does not have.
	if approval.WorktreeDigest != "" {
		return repostate.State{}, errors.New("checkout is not the approved repository state: approval binds uncommitted worktree content")
	}
	if err := repostate.VerifyBase(ctx, root, approval.BaseRevisionDigest); err != nil {
		return repostate.State{}, fmt.Errorf("checkout is not the approved repository state: %w", err)
	}
	st, err := repostate.Snapshot(ctx, root, approval.BaseRevisionDigest, false)
	if err != nil {
		return repostate.State{}, fmt.Errorf("checkout is not the approved repository state: %w", err)
	}
	if st.Head != ciSHA || st.Head != approval.ProducedHeadDigest || st.Diff != approval.DiffDigest {
		return repostate.State{}, errors.New("checkout is not the approved repository state: head or diff differs")
	}
	return st, nil
}
