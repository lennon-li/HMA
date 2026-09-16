package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/transition"
)

// requireHostTransitionGate makes architecture section 12.4 executable at the
// two coding progression edges. Implementation cannot be presented for review
// without a recorded delegation or justified direct-cost path, and review
// cannot advance to verification without a bound independent APPROVE result.
func requireHostTransitionGate(storePath string, in transition.Input) error {
	from := in.ExpectedApproval.CurrentStage
	to := in.ExpectedApproval.ProposedTargetStage
	if !((from == model.StageImplementationAuthorization && to == model.StageImplementationReview) ||
		(from == model.StageImplementationReview && to == model.StageVerification)) {
		return nil
	}
	if in.ExpectedApproval.UnitID == "" {
		return errors.New("HOST_UNIT_ID_REQUIRED")
	}
	history, err := hoststate.NewStore(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if from == model.StageImplementationAuthorization {
		_, err = hoststate.RequireImplementation(history, in.ExpectedApproval.UnitID)
		return err
	}
	return hoststate.RequireApprovedReview(history, in.ExpectedApproval.UnitID)
}

// runTransition records one human-approved stage, classification, or supported outcome. It is the only
// command that changes a run's stage, and it does so only by recording an
// approval a human already gave; it never chooses a target or supplies a
// decision of its own.
func runTransition(args []string) error {
	fs := flag.NewFlagSet("transition", flag.ContinueOnError)
	inputPath := fs.String("input", "", "host input file containing one approved stage, classification, or supported outcome")
	storePath := fs.String("store", "", "host store directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inputPath == "" || *storePath == "" || fs.NArg() != 0 {
		return errors.New("usage: hma transition --input <file> --store <directory>")
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var in transition.Input
	if err := dec.Decode(&in); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple input documents")
	}

	if err := requireHostTransitionGate(*storePath, in); err != nil {
		return err
	}

	result, err := transition.Apply(context.Background(), in, *storePath)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
