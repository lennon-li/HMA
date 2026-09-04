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

	s := store.New(*storePath)
	records, err := s.Load(*runID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.New("no records found for run_id")
	}
	lastRecord := records[len(records)-1]

	var approval *model.ApprovalBinding
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Approval != nil {
			approval = records[i].Approval
			break
		}
	}
	if approval == nil {
		return errors.New("no approval binding found in chain to verify against")
	}

	res := engine.EvaluateCIVerification(engine.CIVerificationRequest{
		CIRepository:               env.Repository,
		CISHA:                      env.SHA,
		CurrentStage:               lastRecord.Stage,
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

	captured, err := evidence.Capture(context.Background(), evidence.Request{
		Executable:         *executable,
		Argv:               argv,
		WorkingDir:         *repoRoot,
		RepositoryRoot:     *repoRoot,
		MaxOutputBytes:     1 << 20,
		RepositoryIdentity: env.Repository,
		BaseRevision:       approval.BaseRevisionDigest,
		HeadRevision:       env.SHA,
		DiffDigest:         approval.DiffDigest,
		CapturingActor:     actor,
	})
	if err != nil {
		return err
	}

	rec := model.Record{
		Kind:    model.KindEvidence,
		Version: 1,
		Metadata: model.RecordMetadata{
			RunID:           *runID,
			Sequence:        lastRecord.Metadata.Sequence + 1,
			PredecessorHash: lastRecord.Metadata.HeadAnchor,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Actor:           actor,
		},
		Stage:    lastRecord.Stage,
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
