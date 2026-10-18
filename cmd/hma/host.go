package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
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
	// ReviewEnforcement labels the run's review mode after this record
	// (packet_import or legacy_manual).
	ReviewEnforcement string `json:"review_enforcement,omitempty"`
	NextAction        string `json:"next_action"`
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

// hostCheckHook and hostAppendHook are test seams: nil in production. A
// core append started from either hook blocks until the host append
// completes, which is what the tests assert.
var hostCheckHook, hostAppendHook func()

// hostCheck validates one host command against the host history and the core
// chain it is about to bind. It runs under the host run lock, against one
// core snapshot whose head the record then binds.
type hostCheck func(history []hoststate.Record, core []model.Record) error

// appendHostRecord evaluates check and appends the record atomically with
// respect to both chains: it holds the host run lock and then the core run
// lock, checks against the core chain read under that lock, and writes the
// host record before releasing either. No core record (a terminal outcome, a
// rewind that revokes the route) can commit between the check and the host
// append, so an execution-authorizing host record is never appended after
// the core withdrew its authority. Lock order across HMA is always host then
// core; hma transition takes them in the same order.
func appendHostRecord(storePath string, envelope hostEnvelope, kind hoststate.RecordKind, decision *hoststate.OrchestrationDecision, preflight *hoststate.RoutePreflight, dispatch *hoststate.Dispatch, review *hoststate.IndependentReview, loop *hoststate.LoopEvidence, check hostCheck) (hostResult, error) {
	var rec hoststate.Record
	err := hoststate.NewStore(storePath).Transact(envelope.RunID, func(history []hoststate.Record, commit func(*hoststate.Record) error) error {
		return store.New(storePath).WithRunLock(envelope.RunID, func(core []model.Record) error {
			if len(core) == 0 {
				return errors.New(hoststate.ReasonCoreRunMissing)
			}
			if engine.ProjectStageState(core).Terminal {
				return errors.New(hoststate.ReasonCoreRunTerminal)
			}
			if hostCheckHook != nil {
				hostCheckHook()
			}
			if err := check(history, core); err != nil {
				return err
			}
			rec = hoststate.Record{
				Version:        1,
				Kind:           kind,
				RunID:          envelope.RunID,
				CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor,
				Timestamp:      envelope.Timestamp,
				Actor:          envelope.Actor,
				Decision:       decision,
				Preflight:      preflight,
				Dispatch:       dispatch,
				Review:         review,
				LoopEvidence:   loop,
			}
			if hostAppendHook != nil {
				hostAppendHook()
			}
			// Commit inside the core lock: the record, head, and directory
			// are durable before any core terminal or rewind can append.
			return commit(&rec)
		})
	})
	if err != nil {
		return hostResult{}, err
	}
	return hostResultFor(storePath, rec), nil
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
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindDecision, &in.Decision, nil, nil, nil, nil, func(history []hoststate.Record, core []model.Record) error {
		if err := validateRouteApprovalClaimIn(core, in.RunID, in.Decision.UnitID, in.Decision.Route.RouteApprovalDigest); err != nil {
			return err
		}
		return hoststate.EvaluateDecision(history, in.Decision)
	})
	if err != nil {
		return err
	}
	result.NextAction = "RUN_ROUTE_PREFLIGHT"
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
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindPreflight, nil, &in.Preflight, nil, nil, nil, func(history []hoststate.Record, core []model.Record) error {
		if err := validateRouteApprovalClaimIn(core, in.RunID, in.Preflight.UnitID, in.Preflight.Route.RouteApprovalDigest); err != nil {
			return err
		}
		return hoststate.EvaluatePreflight(history, in.Preflight, at)
	})
	if err != nil {
		return err
	}
	result.DispatchAllowed = in.Preflight.Status == hoststate.PreflightAvailable
	if result.DispatchAllowed {
		result.NextAction = "RECORD_DISPATCH"
	} else {
		result.NextAction = "RECORD_SUPERSEDING_ROUTE_DECISION"
	}
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
	// Packet provenance is created only by hma host review-packet; a
	// caller-supplied packet binding would impersonate an HMA-generated one.
	if in.Dispatch.Packet != nil {
		return errors.New(hoststate.ReasonReviewImportRequired)
	}
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindDispatch, nil, nil, &in.Dispatch, nil, nil, func(history []hoststate.Record, core []model.Record) error {
		if err := validateRouteApprovalClaimIn(core, in.RunID, in.Dispatch.UnitID, in.Dispatch.Route.RouteApprovalDigest); err != nil {
			return err
		}
		return hoststate.EvaluateDispatch(history, in.Dispatch, at)
	})
	if err != nil {
		return err
	}
	result.DispatchAllowed = true
	result.NextAction = "START_WORKER"
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
	// Import provenance fields are set only by hma host review-import.
	if r := in.Review; r.ReceiptDigest != "" || r.ReportDigest != "" || r.Isolation != "" || r.ReportVerdict != "" {
		return errors.New(hoststate.ReasonReviewImportRequired)
	}
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindReview, nil, nil, nil, &in.Review, nil, func(history []hoststate.Record, core []model.Record) error {
		// A review continues the coding path only while the unit's route
		// approval is still in force in the core chain it binds.
		if _, err := approvedCoreRouteDigestIn(core, in.RunID, in.Review.UnitID); err != nil {
			return err
		}
		return hoststate.EvaluateReview(history, in.Review)
	})
	if err != nil {
		return err
	}
	result.NextAction = "CONTINUE_HMA_TRANSITION"
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runHostShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	storePath := fs.String("store", "", "HMA store directory")
	runID := fs.String("run_id", "", "run id")
	mode := fs.Bool("mode", false, "print only the run's review enforcement mode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePath == "" || *runID == "" || fs.NArg() != 0 {
		return errors.New("usage: hma host show --store <directory> --run_id <id> [--mode]")
	}
	records, err := hoststate.NewStore(*storePath).Load(*runID)
	if err != nil {
		return err
	}
	if *mode {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"review_enforcement": hoststate.ReviewEnforcementMode(records)})
	}
	return json.NewEncoder(os.Stdout).Encode(records)
}

func runHost(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: hma host [decision|preflight|dispatch|review|review-packet|review-import|findings-import|loop-evidence|bind-repository|attempt|show] ...")
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
	case "review-packet":
		return runHostReviewPacket(args[1:])
	case "review-import":
		return runHostReviewImport(args[1:])
	case "findings-import":
		return runHostFindingsImport(args[1:])
	case "attempt":
		return runHostAttempt(args[1:])
	case "bind-repository":
		return runHostBindRepository(args[1:])
	case "loop-evidence":
		return runHostLoopEvidence(args[1:])
	case "show":
		return runHostShow(args[1:])
	default:
		return errors.New("usage: hma host [decision|preflight|dispatch|review|review-packet|review-import|findings-import|loop-evidence|bind-repository|attempt|show] ...")
	}
}

// hostResultFor reports a committed host record and the run's review
// enforcement mode after it.
func hostResultFor(storePath string, rec hoststate.Record) hostResult {
	res := hostResult{Recorded: true, Kind: rec.Kind, Sequence: rec.Sequence, HeadAnchor: rec.HeadAnchor, CoreHeadAnchor: rec.CoreHeadAnchor}
	if history, err := hoststate.NewStore(storePath).Load(rec.RunID); err == nil {
		res.ReviewEnforcement = hoststate.ReviewEnforcementMode(history)
	}
	return res
}
