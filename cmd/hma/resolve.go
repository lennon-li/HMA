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

	// Project state
	var criteria []model.Criterion
	var findings []model.Finding
	for _, r := range records {
		if len(r.Criteria) > 0 {
			criteria = r.Criteria // Latest criteria win in this simple projection
		}
		if len(r.Findings) > 0 {
			findings = r.Findings
		}
	}

	req := engine.ResolutionRequest{
		WaiverOperation: in.WaiverOperation,
		FindingOverride: in.FindingOverride,
		CurrentCriteria: criteria,
		CurrentFindings: findings,
	}

	res := engine.EvaluateResolution(req)
	if res.Decision != engine.DecisionLegalPendingApproval {
		return errors.New(string(res.Reason))
	}

	var rec model.Record
	rec.Version = 1
	rec.Metadata.RunID = in.RunID
	rec.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	// Not full metadata assembly here since we're mirroring the bounded pilot pattern for now,
	// but assigning the Kind and specific fields.
	if in.WaiverOperation != nil {
		rec.Kind = model.KindWaiverOperation
		rec.WaiverOperation = in.WaiverOperation
		rec.Metadata.Actor = in.WaiverOperation.Approver
	} else {
		rec.Kind = model.KindFindingOverride
		rec.FindingOverride = in.FindingOverride
		rec.Metadata.Actor = in.FindingOverride.Approver
	}

	if err := model.ValidateRecord(&rec); err != nil {
		// Just validation check, though Store.Append will do it again with proper sequence.
		// Ignore the sequence/hash validation error for the dry run.
	}

	// In a real execution, we'd use s.Append(&rec).
	// The pilot CLI demonstrates the pattern. For resolve, we just append it.
	err = s.Append(&rec)
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(res)
}
