package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

// transitionGateHook is a test seam, nil in production: it runs after the
// host gate passes and before the core append.
var transitionGateHook func()

// requireHostTransitionGate makes architecture section 12.4 executable at the
// two coding progression edges. Implementation cannot be presented for review
// without a recorded delegation or justified direct-cost path, and review
// cannot advance to verification without a bound independent APPROVE result.
func requireHostTransitionGate(storePath string, in transition.Input) error {
	history, err := hoststate.NewStore(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	return requireHostTransitionGateIn(storePath, in, history)
}

// requireHostTransitionGateIn is requireHostTransitionGate against an already
// loaded host history, for a caller that holds the host run lock.
func requireHostTransitionGateIn(storePath string, in transition.Input, history []hoststate.Record) error {
	from := in.ExpectedApproval.CurrentStage
	to := in.ExpectedApproval.ProposedTargetStage
	if !((from == model.StageImplementationAuthorization && to == model.StageImplementationReview) ||
		(from == model.StageImplementationReview && to == model.StageVerification)) {
		return nil
	}
	if in.ExpectedApproval.UnitID == "" {
		return errors.New("HOST_UNIT_ID_REQUIRED")
	}
	if err := requireCoreRunOpen(storePath, in.RunID); err != nil {
		return err
	}
	if from == model.StageImplementationAuthorization {
		implementationDigest, err := hoststate.RequireImplementation(history, in.ExpectedApproval.UnitID)
		if err != nil {
			return err
		}
		if err := requireApprovedHostRouteIn(storePath, in.RunID, in.ExpectedApproval.UnitID, history); err != nil {
			return err
		}
		currentCoreHead, err := coreHead(storePath, in.RunID)
		if err != nil {
			return err
		}
		for _, record := range history {
			if record.HeadAnchor == implementationDigest && record.CoreHeadAnchor != currentCoreHead {
				return errors.New(hoststate.ReasonCoreBindingStale)
			}
		}
		return nil
	}
	if err := hoststate.RequireApprovedReview(history, in.ExpectedApproval.UnitID); err != nil {
		return err
	}
	// The route that produced the reviewed work must still be the approved
	// one at this second gate, not only at implementation authorization.
	if err := requireApprovedHostRouteIn(storePath, in.RunID, in.ExpectedApproval.UnitID, history); err != nil {
		return err
	}
	implementationDigest, err := hoststate.RequireImplementation(history, in.ExpectedApproval.UnitID)
	if err != nil {
		return err
	}
	currentCoreHead, err := coreHead(storePath, in.RunID)
	if err != nil {
		return err
	}
	for i := len(history) - 1; i >= 0; i-- {
		record := history[i]
		if record.Kind == hoststate.KindReview && record.Review != nil && record.Review.UnitID == in.ExpectedApproval.UnitID && record.Review.ImplementationRecordDigest == implementationDigest {
			if record.CoreHeadAnchor != currentCoreHead {
				return errors.New(hoststate.ReasonCoreBindingStale)
			}
			// The review must name the exact artifact this transition records.
			// The worktree part is the host-declared dirty content, which the
			// core transition requires to equal the live worktree; the
			// approval's own worktree binding is optional, so it alone would
			// let one review stand for different uncommitted bytes.
			a := in.ExpectedApproval
			if a.WorktreeDigest != "" && a.WorktreeDigest != in.ExpectedWorktreeDigest {
				return errors.New(hoststate.ReasonReviewArtifactMismatch)
			}
			if record.Review.ArtifactDigest != hoststate.ReviewArtifactDigest(a.ProducedHeadDigest, a.DiffDigest, in.ExpectedWorktreeDigest) {
				return errors.New(hoststate.ReasonReviewArtifactMismatch)
			}
			return nil
		}
	}
	return errors.New(hoststate.ReasonReviewMissing)
}

// requireApprovedHostRoute binds the host's active route claim to a real,
// validated core approval. Route telemetry alone is untrusted: only an
// approved Route Selection -> Implementation Authorization transition for
// this unit can supply the digest.
func requireApprovedHostRoute(storePath, runID, unitID string) error {
	if runID == "" || unitID == "" {
		return errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	history, err := hoststate.NewStore(storePath).Load(runID)
	if err != nil {
		return err
	}
	return requireApprovedHostRouteIn(storePath, runID, unitID, history)
}

// requireApprovedHostRouteIn is requireApprovedHostRoute against an already
// loaded host history.
func requireApprovedHostRouteIn(storePath, runID, unitID string, history []hoststate.Record) error {
	if runID == "" || unitID == "" {
		return errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	activeDigest, err := approvedCoreRouteDigest(storePath, runID, unitID)
	if err != nil {
		return err
	}
	var decision *hoststate.OrchestrationDecision
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Decision != nil && history[i].Decision.UnitID == unitID {
			decision = history[i].Decision
			break
		}
	}
	if decision == nil || decision.Route.RouteApprovalDigest != activeDigest {
		return errors.New(hoststate.ReasonRouteApprovalMismatch)
	}
	implementation, err := hoststate.RequireImplementation(history, unitID)
	if err != nil {
		return err
	}
	if decision.Mode == hoststate.DecisionDelegate {
		for _, rec := range history {
			if rec.HeadAnchor == implementation && (rec.Dispatch == nil || rec.Dispatch.Route.RouteApprovalDigest != activeDigest) {
				return errors.New(hoststate.ReasonRouteApprovalMismatch)
			}
		}
	}
	return nil
}

func validateRouteApprovalClaim(storePath, runID, unitID, claimedDigest string) error {
	if claimedDigest == "" {
		return errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	activeDigest, err := approvedCoreRouteDigest(storePath, runID, unitID)
	if err != nil {
		return err
	}
	if claimedDigest != activeDigest {
		return errors.New(hoststate.ReasonRouteApprovalMismatch)
	}
	return nil
}

func approvedCoreRouteDigest(storePath, runID, unitID string) (string, error) {
	if runID == "" || unitID == "" {
		return "", errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	core, err := store.New(storePath).Load(runID)
	if err != nil {
		return "", err
	}
	return approvedCoreRouteDigestIn(core, runID, unitID)
}

// validateRouteApprovalClaimIn is validateRouteApprovalClaim against an
// already loaded core chain, so a caller can check and bind one snapshot.
func validateRouteApprovalClaimIn(core []model.Record, runID, unitID, claimedDigest string) error {
	if claimedDigest == "" {
		return errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	activeDigest, err := approvedCoreRouteDigestIn(core, runID, unitID)
	if err != nil {
		return err
	}
	if claimedDigest != activeDigest {
		return errors.New(hoststate.ReasonRouteApprovalMismatch)
	}
	return nil
}

func approvedCoreRouteDigestIn(core []model.Record, runID, unitID string) (string, error) {
	if runID == "" || unitID == "" {
		return "", errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	var activeDigest string
	for _, rec := range core {
		if rec.Kind != model.KindStageTransition || rec.Control != model.ControlApproved || rec.Approval == nil {
			continue
		}
		a := rec.Approval
		if a.RunID != runID || a.UnitID != rec.UnitID {
			continue
		}
		if rec.UnitID != unitID && rec.UnitID != "" {
			// Another unit's approved return to Planning or earlier can
			// invalidate this unit as a dependent. The persisted transition
			// does not name its dependents, so every unit's route approval is
			// treated as invalidated (fail closed) until a fresh Route Selection
			// approval; Gate D narrows this with plan-derived dependents.
			switch a.ProposedTargetStage {
			case model.StageGrounding, model.StageAcceptanceCriteria, model.StagePlanning:
				activeDigest = ""
			}
			continue
		}
		if rec.UnitID == unitID && rec.Stage == model.StageRouteSelection &&
			a.CurrentStage == model.StageRouteSelection && a.ProposedTargetStage == model.StageImplementationAuthorization {
			activeDigest = a.TransitionDigest
			continue
		}
		// Replaying in chain order invalidates the approval on any later
		// unit-scoped or whole-run return to an earlier decision gate. A
		// subsequent Route Selection approval can establish a new digest.
		switch a.ProposedTargetStage {
		case model.StageGrounding, model.StageAcceptanceCriteria, model.StagePlanning, model.StageRouteSelection:
			activeDigest = ""
		}
	}
	if activeDigest == "" {
		return "", errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	return activeDigest, nil
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

	// The host gate and the core append run under one host run lock, so no
	// host record (for example an independent BLOCK review) can land between
	// the gate check and the recorded transition. Lock order is host, then
	// core (transition.Apply takes the core lock), as in appendHostRecord.
	var result transition.Result
	if in.RunID == "" {
		return errors.New("missing host input")
	}
	err = hoststate.NewStore(*storePath).WithRunLock(in.RunID, func(history []hoststate.Record) error {
		if err := requireHostTransitionGateIn(*storePath, in, history); err != nil {
			return err
		}
		if transitionGateHook != nil {
			transitionGateHook()
		}
		var err error
		result, err = transition.Apply(context.Background(), in, *storePath)
		return err
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

// requireCoreRunOpen refuses host work on a run whose core chain records a
// terminal outcome: no decision, preflight, dispatch, review, or gate may
// continue a run that has ended.
func requireCoreRunOpen(storePath, runID string) error {
	core, err := store.New(storePath).Load(runID)
	if err != nil {
		return err
	}
	if engine.ProjectStageState(core).Terminal {
		return errors.New(hoststate.ReasonCoreRunTerminal)
	}
	return nil
}
