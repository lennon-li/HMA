package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lennon-li/HMA/internal/ci/github"
	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func runCIGithub(args []string) error {
	fs := flag.NewFlagSet("ci-github", flag.ContinueOnError)
	storePath := fs.String("store", "", "host store directory")
	runID := fs.String("run_id", "", "run ID to verify against")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePath == "" || *runID == "" || fs.NArg() != 0 {
		return errors.New("usage: hma ci github --store <directory> --run_id <run_id>")
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

	var expectedRepo string
	var expectedHead string

	if lastRecord.Approval != nil {
		expectedRepo = lastRecord.Approval.RepositoryIdentityDigest
		expectedHead = lastRecord.Approval.ProducedHeadDigest
	} else {
		// Walk back to find the most recent approval
		for i := len(records) - 1; i >= 0; i-- {
			if records[i].Approval != nil {
				expectedRepo = records[i].Approval.RepositoryIdentityDigest
				expectedHead = records[i].Approval.ProducedHeadDigest
				break
			}
		}
		if expectedRepo == "" {
			return errors.New("no approval binding found in chain to verify against")
		}
	}

	req := engine.CIVerificationRequest{
		CIRepository:                     env.Repository,
		CISHA:                            env.SHA,
		CurrentStage:                     lastRecord.Stage,
		ExpectedRepositoryIdentityDigest: expectedRepo,
		ExpectedProducedHeadDigest:       expectedHead,
	}

	res := engine.EvaluateCIVerification(req)
	if res.Decision != engine.DecisionLegalPendingApproval {
		return errors.New(string(res.Reason))
	}

	var rec model.Record
	rec.Version = 1
	rec.Metadata.RunID = *runID
	rec.Metadata.Sequence = lastRecord.Metadata.Sequence + 1
	rec.Metadata.PredecessorHash = lastRecord.Metadata.HeadAnchor
	rec.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	rec.Metadata.Actor = env.Actor
	rec.Kind = model.KindEvidence // A CI run produces evidence
	rec.Stage = lastRecord.Stage
	rec.Evidence = &model.EvidenceRef{
		Executable:         "github-actions",
		Argv:               []string{"ci-run", env.RunID},
		WorkingDir:         "/github/workspace",
		StartTimestamp:     rec.Metadata.Timestamp,
		EndTimestamp:       rec.Metadata.Timestamp,
		RepositoryIdentity: env.Repository,
		BaseRevision:       env.SHA,
		HeadRevision:       env.SHA,
		CapturingActor:     env.Actor,
	}

	if err := model.ValidateRecord(&rec); err != nil {
		return err
	}

	err = s.Append(&rec)
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(res)
}
