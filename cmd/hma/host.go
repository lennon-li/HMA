package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/store"
)

type hostEnvelope struct {
	RunID     string `json:"run_id"`
	Actor     string `json:"actor"`
	Timestamp string `json:"timestamp"`
}

type hostDecisionInput struct {
	hostEnvelope
	Decision hoststate.OrchestrationDecision `json:"decision"`
}

type hostPreflightInput struct {
	hostEnvelope
	Preflight hoststate.RoutePreflight `json:"preflight"`
}

type hostDispatchInput struct {
	hostEnvelope
	Dispatch hoststate.Dispatch `json:"dispatch"`
}

type hostReviewInput struct {
	hostEnvelope
	Review hoststate.IndependentReview `json:"review"`
}

type hostResult struct {
	Recorded        bool                 `json:"recorded"`
	Kind            hoststate.RecordKind `json:"kind"`
	Sequence        int64                `json:"sequence"`
	HeadAnchor      string               `json:"head_anchor"`
	CoreHeadAnchor  string               `json:"core_head_anchor"`
	DispatchAllowed bool                 `json:"dispatch_allowed,omitempty"`
}

func decodeStrictFile(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple input documents")
	}
	return nil
}

func coreHead(storePath, runID string) (string, error) {
	records, err := store.New(storePath).Load(runID)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", errors.New(hoststate.ReasonCoreRunMissing)
	}
	return records[len(records)-1].Metadata.HeadAnchor, nil
}

func validateHostEnvelope(in hostEnvelope) (time.Time, error) {
	if in.RunID == "" || in.Actor == "" || in.Timestamp == "" {
		return time.Time{}, errors.New(hoststate.ReasonInvalidHostRecord)
	}
	at, err := time.Parse(time.RFC3339, in.Timestamp)
	if err != nil {
		return time.Time{}, errors.New(hoststate.ReasonInvalidHostRecord)
	}
	return at, nil
}

func appendHostRecord(storePath string, envelope hostEnvelope, kind hoststate.RecordKind, decision *hoststate.OrchestrationDecision, preflight *hoststate.RoutePreflight, dispatch *hoststate.Dispatch, review *hoststate.IndependentReview) (hostResult, error) {
	core, err := coreHead(storePath, envelope.RunID)
	if err != nil {
		return hostResult{}, err
	}
	rec := hoststate.Record{
		Version:        1,
		Kind:           kind,
		RunID:          envelope.RunID,
		CoreHeadAnchor: core,
		Timestamp:      envelope.Timestamp,
		Actor:          envelope.Actor,
		Decision:       decision,
		Preflight:      preflight,
		Dispatch:       dispatch,
		Review:         review,
	}
	hs := hoststate.NewStore(storePath)
	if err := hs.Append(&rec); err != nil {
		return hostResult{}, err
	}
	return hostResult{Recorded: true, Kind: kind, Sequence: rec.Sequence, HeadAnchor: rec.HeadAnchor, CoreHeadAnchor: rec.CoreHeadAnchor}, nil
}

func hostCommandFlags(name string, args []string) (string, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	input := fs.String("input", "", "host input file")
	storePath := fs.String("store", "", "HMA store directory")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if *input == "" || *storePath == "" || fs.NArg() != 0 {
		return "", "", errors.New("usage: hma host " + name + " --input <file> --store <directory>")
	}
	return *input, *storePath, nil
}

func runHostDecision(args []string) error {
	inputPath, storePath, err := hostCommandFlags("decision", args)
	if err != nil {
		return err
	}
	var in hostDecisionInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	history, err := hoststate.NewStore(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if err := hoststate.EvaluateDecision(history, in.Decision); err != nil {
		return err
	}
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindDecision, &in.Decision, nil, nil, nil)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runHostPreflight(args []string) error {
	inputPath, storePath, err := hostCommandFlags("preflight", args)
	if err != nil {
		return err
	}
	var in hostPreflightInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	at, err := validateHostEnvelope(in.hostEnvelope)
	if err != nil {
		return err
	}
	history, err := hoststate.NewStore(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if err := hoststate.EvaluatePreflight(history, in.Preflight, at); err != nil {
		return err
	}
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindPreflight, nil, &in.Preflight, nil, nil)
	if err != nil {
		return err
	}
	result.DispatchAllowed = in.Preflight.Status == hoststate.PreflightAvailable
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runHostDispatch(args []string) error {
	inputPath, storePath, err := hostCommandFlags("dispatch", args)
	if err != nil {
		return err
	}
	var in hostDispatchInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	at, err := validateHostEnvelope(in.hostEnvelope)
	if err != nil {
		return err
	}
	history, err := hoststate.NewStore(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if err := hoststate.EvaluateDispatch(history, in.Dispatch, at); err != nil {
		return err
	}
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindDispatch, nil, nil, &in.Dispatch, nil)
	if err != nil {
		return err
	}
	result.DispatchAllowed = true
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runHostReview(args []string) error {
	inputPath, storePath, err := hostCommandFlags("review", args)
	if err != nil {
		return err
	}
	var in hostReviewInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	history, err := hoststate.NewStore(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if err := hoststate.EvaluateReview(history, in.Review); err != nil {
		return err
	}
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindReview, nil, nil, nil, &in.Review)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runHostShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	storePath := fs.String("store", "", "HMA store directory")
	runID := fs.String("run_id", "", "run id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePath == "" || *runID == "" || fs.NArg() != 0 {
		return errors.New("usage: hma host show --store <directory> --run_id <id>")
	}
	records, err := hoststate.NewStore(*storePath).Load(*runID)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(records)
}

func runHost(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: hma host [decision|preflight|dispatch|review|show] ...")
	}
	switch args[0] {
	case "decision":
		return runHostDecision(args[1:])
	case "preflight":
		return runHostPreflight(args[1:])
	case "dispatch":
		return runHostDispatch(args[1:])
	case "review":
		return runHostReview(args[1:])
	case "show":
		return runHostShow(args[1:])
	default:
		return errors.New("usage: hma host [decision|preflight|dispatch|review|show] ...")
	}
}
