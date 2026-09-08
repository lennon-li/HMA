package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

type resolveInput struct {
	RunID           string                            `json:"run_id"`
	WaiverOperation *model.WaiverOperation            `json:"waiver_operation,omitempty"`
	FindingOverride *model.FindingDispositionOverride `json:"finding_override,omitempty"`

	// ExpectedPredecessorHead, when set, binds this operation to the exact
	// chain tail the human approved against. It is host input and is not
	// persisted in the record.
	ExpectedPredecessorHead string `json:"expected_predecessor_head,omitempty"`

	// RepositoryIdentity, BaseRevision, Head and DiffDigest are the live
	// repository values the host observes now. An operation that declares
	// any of them must match, so a waiver bound to the head or diff it was
	// granted against is refused once the repository moved on.
	RepositoryIdentity string `json:"repository_identity,omitempty"`
	BaseRevision       string `json:"base_revision,omitempty"`
	Head               string `json:"head,omitempty"`
	DiffDigest         string `json:"diff_digest,omitempty"`
}

func runResolve(args []string) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	inputPath := fs.String("input", "", "host input file containing one resolution operation")
	storePath := fs.String("store", "", "host store directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inputPath == "" || *storePath == "" || fs.NArg() != 0 {
		return errors.New("usage: hma resolve --input <file> --store <directory>")
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var in resolveInput
	if err := dec.Decode(&in); err != nil {
		return err
	}

	if in.RunID == "" {
		return errors.New("missing run_id")
	}

	s := store.New(*storePath)
	records, err := s.Load(in.RunID)
	if err != nil {
		return err
	}

	predecessorHead := ""
	if len(records) > 0 {
		predecessorHead = records[len(records)-1].Metadata.HeadAnchor
	}

	// Evaluate against the replayed chain, so waivers and overrides already
	// recorded are visible and their nonces cannot be replayed.
	req := engine.NewResolutionRequest(engine.ProjectResolutionState(records))
	req.WaiverOperation = in.WaiverOperation
	req.FindingOverride = in.FindingOverride
	req.ExpectedPredecessorHead = in.ExpectedPredecessorHead
	req.PredecessorHead = predecessorHead
	req.RepositoryIdentity = in.RepositoryIdentity
	req.BaseRevision = in.BaseRevision
	req.Head = in.Head
	req.DiffDigest = in.DiffDigest

	res := engine.EvaluateResolution(req)
	if res.Decision != engine.DecisionLegalPendingApproval {
		return errors.New(string(res.Reason))
	}

	rec := model.Record{
		Version: 1,
		Metadata: model.RecordMetadata{
			RunID:           in.RunID,
			Sequence:        int64(len(records) + 1),
			PredecessorHash: predecessorHead,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		},
	}
	if in.WaiverOperation != nil {
		rec.Kind = model.KindWaiverOperation
		rec.WaiverOperation = in.WaiverOperation
		rec.Metadata.Actor = in.WaiverOperation.Approver
	} else {
		rec.Kind = model.KindFindingOverride
		rec.FindingOverride = in.FindingOverride
		rec.Metadata.Actor = in.FindingOverride.Approver
	}

	// Store.Append computes the head anchor and then validates the complete
	// record; a pre-append ValidateRecord call could never pass, because the
	// anchor it requires does not exist yet.
	if err := s.Append(&rec); err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(res)
}
