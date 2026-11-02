package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

// The checkpoint gate's refusal reasons (architecture section 16 "Checkpoint
// and final gates", E2). They are defined here, not in internal/host, because
// Gate E Phase 4b is scoped to cmd/hma/transition.go only; a later phase
// that touches internal/host may consolidate them there alongside the other
// HOST_* reasons.
const (
	reasonCheckpointReviewMissing     = "HOST_CHECKPOINT_REVIEW_MISSING"
	reasonCheckpointTriggersUncovered = "HOST_CHECKPOINT_TRIGGERS_UNCOVERED"
	reasonCheckpointReviewStale       = "HOST_CHECKPOINT_REVIEW_STALE"
)

// reasonContractRequiredForSuccessClaim is the final gate's (E3, architecture
// section 16/20, Decision D10) refusal when a non-contract run attempts a
// success-claiming terminal outcome. It is defined here, not in
// internal/host, for the same reason as the three reasons above: Gate E's
// implementation phases are scoped to cmd/hma.
const reasonContractRequiredForSuccessClaim = "HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM"

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
	return requireHostTransitionGateIn(storePath, in, history, "", "")
}

// requireHostTransitionGateIn is requireHostTransitionGate against an already
// loaded host history, for a caller that holds the host run lock.
// contractPath and checkpointPolicyPath are host-supplied inputs read only at
// the checkpoint gate (architecture section 16, E2); every other caller,
// including requireHostTransitionGate's external callers, passes "", "".
func requireHostTransitionGateIn(storePath string, in transition.Input, history []hoststate.Record, contractPath, checkpointPolicyPath string) error {
	from := in.ExpectedApproval.CurrentStage
	to := in.ExpectedApproval.ProposedTargetStage
	checkpointEdge := from == model.StageIndependentValidation && to == model.StageReleaseAndClosure
	if !((from == model.StageImplementationAuthorization && to == model.StageImplementationReview) ||
		(from == model.StageImplementationReview && to == model.StageVerification) ||
		checkpointEdge) {
		return nil
	}
	if checkpointEdge {
		// The checkpoint edge's own UnitID/run-open requirements are
		// conditional on the run being contract-bound (Decision D10): see
		// requireCheckpointGate. The two edges below are inherently always
		// per-unit transitions, so they keep their unconditional requirement.
		return requireCheckpointGate(storePath, in, history, contractPath, checkpointPolicyPath)
	}
	if in.ExpectedApproval.UnitID == "" {
		return errors.New("HOST_UNIT_ID_REQUIRED")
	}
	if err := requireCoreRunOpen(storePath, in.RunID); err != nil {
		return err
	}
	if err := requireReconciledLaunches(storePath, in.RunID, in.ExpectedApproval.UnitID, in.PolicyPath, history); err != nil {
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
	// A BLOCK finding, or a WAIVABLE one without an active human waiver,
	// keeps the unit in review.
	if err := requireFindingsResolved(storePath, in.RunID, in.ExpectedApproval.UnitID); err != nil {
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
				if err := requireOnlyOwnSnapshotsSince(storePath, in.RunID, record.CoreHeadAnchor, record.HeadAnchor); err != nil {
					return err
				}
			} else if record.Review.ReportDigest != "" {
				// An imported review counts only after its findings
				// snapshot is in the core chain.
				return errors.New(hoststate.ReasonFindingsSnapshotMissing)
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

// requireCheckpointGate implements architecture section 16 "Checkpoint and
// final gates", E2: on INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE, a
// contract-bound unit on which any gate-time trigger fires must show a
// qualifying checkpoint-grade review. It does not apply to a non-contract
// run (Decision D10), and it requires nothing new when no trigger fires
// (Decision D8 (1)): the per-change gate (E1) and the other existing gates
// are unaffected either way. Unlike the other two gated edges, it does not
// call requireReconciledLaunches: the architecture's E2 requirements list
// does not name it, since unreconciled-launch checking is an E1 (per-change
// review) concern with no named analogue at the IV exit.
//
// contractPath and checkpointPolicyPath are optional host inputs (hma
// transition --contract/--checkpoint-policy); they are read only once the
// run is known to be contract-bound.
func requireCheckpointGate(storePath string, in transition.Input, history []hoststate.Record, contractPath, checkpointPolicyPath string) error {
	unit := in.ExpectedApproval.UnitID
	core, err := store.New(storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if engine.BoundContractDigest(core) == "" {
		// A non-contract run gets nothing new from this edge: no UnitID
		// requirement, no open-run requirement, no review requirement
		// (Decision D10). At HEAD before Gate E, this edge matched nothing
		// and returned nil immediately for every caller; this preserves
		// that exactly.
		return nil
	}
	if unit == "" {
		return errors.New("HOST_UNIT_ID_REQUIRED")
	}
	if err := requireCoreRunOpen(storePath, in.RunID); err != nil {
		return err
	}

	root, err := filepath.Abs(in.RepositoryRoot)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return err
	}
	// HMA's own changed-path manifest between the approved base and the
	// live worktree/head, exactly as runHostReviewPacket (cmd/hma/review.go)
	// captures it at packet issuance, recomputed here at gate time.
	manifest, err := buildChangedFileManifest(boundedCtx(), root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return err
	}

	// corePacketState reads and proves the live bound contract and the
	// unit's current plan unit, the same way packet issuance does; its
	// Evidence/Waivers output is unused here.
	state, err := corePacketState(storePath, core, in.RunID, unit, contractPath, in.PolicyPath, "", hoststate.ReviewPacketBinding{})
	if err != nil {
		return err
	}
	if state.Contract == nil || state.Unit == nil {
		return errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	checkpointPolicy, err := loadCheckpointPolicy(checkpointPolicyPath, state.Contract.CheckpointPolicyDigest)
	if err != nil {
		return err
	}
	baseline, err := checkpointReviewBaseline(storePath, history, core, unit)
	if err != nil {
		return err
	}
	current := engine.CheckpointBaseline{ContractDigest: state.Contract.Digest, PermissionClasses: sortedCopy(state.Unit.PlanUnit.PermissionClasses)}
	decision := engine.ClassifyCheckpoint(engine.CheckpointInput{
		Contract: &state.Contract.AcceptanceContract, Policy: checkpointPolicy, Unit: state.Unit.PlanUnit,
		Baseline: baseline, Current: current, Manifest: toChangedPaths(manifest), Stage: model.StageIndependentValidation,
	})
	if len(decision.Triggers) == 0 {
		return nil
	}

	// A BLOCK finding, or a WAIVABLE one without an active human waiver,
	// keeps the unit in review; this is the same requirement E1 enforces,
	// applied here because E2 otherwise adds no gate at all when untriggered.
	if err := requireFindingsResolved(storePath, in.RunID, unit); err != nil {
		return err
	}

	// The unit's latest imported review must satisfy every bullet below; an
	// earlier qualifying review superseded by a later, non-qualifying one
	// does not count (the same "latest review" rule AnswersLatestReviewDispatch
	// applies at E1).
	var latestReview *hoststate.Record
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Kind == hoststate.KindReview && history[i].Review != nil && history[i].Review.UnitID == unit {
			latestReview = &history[i]
			break
		}
	}
	if latestReview == nil || !hoststate.AnswersLatestReviewDispatch(history, *latestReview.Review) {
		return errors.New(reasonCheckpointReviewMissing)
	}
	review := latestReview.Review
	if review.Verdict != hoststate.ReviewApprove || review.Isolation != "isolated" {
		return errors.New(reasonCheckpointReviewMissing)
	}
	packet := packetFor(history, review.ReviewDispatchDigest)
	if packet == nil || packet.CheckpointSchema == "" ||
		(packet.IssuingReason != hoststate.IssueCheckpoint && packet.IssuingReason != hoststate.IssueFinal) {
		return errors.New(reasonCheckpointReviewMissing)
	}

	// The review must have been dispatched after the unit's approved
	// VERIFICATION -> INDEPENDENT_VALIDATION transition, so an earlier
	// per-change review cannot stand in for the stage audit.
	var dispatchRecord *hoststate.Record
	for i := range history {
		if history[i].Kind == hoststate.KindDispatch && history[i].HeadAnchor == review.ReviewDispatchDigest {
			dispatchRecord = &history[i]
			break
		}
	}
	if dispatchRecord == nil {
		return errors.New(reasonCheckpointReviewMissing)
	}
	ivApprovalIndex := -1
	for i, r := range core {
		if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.UnitID == unit && r.Approval.CurrentStage == model.StageVerification && r.Approval.ProposedTargetStage == model.StageIndependentValidation {
			ivApprovalIndex = i
		}
	}
	dispatchIndex := coreRecordIndex(core, dispatchRecord.CoreHeadAnchor)
	if ivApprovalIndex < 0 || dispatchIndex < 0 || dispatchIndex < ivApprovalIndex {
		return errors.New(reasonCheckpointReviewMissing)
	}

	// The review's findings snapshot must already be in the core chain
	// (the same rule E1 applies at :116-124 above).
	currentCoreHead, err := coreHead(storePath, in.RunID)
	if err != nil {
		return err
	}
	if latestReview.CoreHeadAnchor != currentCoreHead {
		if err := requireOnlyOwnSnapshotsSince(storePath, in.RunID, latestReview.CoreHeadAnchor, latestReview.HeadAnchor); err != nil {
			return err
		}
	} else if review.ReportDigest != "" {
		return errors.New(hoststate.ReasonFindingsSnapshotMissing)
	}

	// Recorded checkpoint_triggers must be a superset of the gate-time
	// triggers, and checkpoint_policy_digest must equal the current one.
	recordedTriggers := make(map[string]bool, len(packet.CheckpointTriggers))
	for _, tr := range packet.CheckpointTriggers {
		recordedTriggers[tr] = true
	}
	for _, tr := range decision.Triggers {
		if !recordedTriggers[string(tr)] {
			return errors.New(reasonCheckpointTriggersUncovered)
		}
	}
	if packet.CheckpointPolicyDigest != decision.PolicyDigest {
		return errors.New(reasonCheckpointTriggersUncovered)
	}

	// Recorded bound_contract_digest must equal the digest bound at gate
	// time, and the review's dispatch risk must reach the gate-time
	// required review risk; either staling the review.
	if packet.BoundContractDigest != state.Contract.Digest {
		return errors.New(reasonCheckpointReviewStale)
	}
	if riskRank(dispatchRecord.Dispatch.Risk) < riskRank(decision.RequiredRisk) {
		return errors.New(reasonCheckpointReviewStale)
	}

	// The review must name the exact artifact this transition records (the
	// same rule E1 applies at :125-136 above).
	a := in.ExpectedApproval
	if a.WorktreeDigest != "" && a.WorktreeDigest != in.ExpectedWorktreeDigest {
		return errors.New(hoststate.ReasonReviewArtifactMismatch)
	}
	if review.ArtifactDigest != hoststate.ReviewArtifactDigest(a.ProducedHeadDigest, a.DiffDigest, in.ExpectedWorktreeDigest) {
		return errors.New(hoststate.ReasonReviewArtifactMismatch)
	}
	return nil
}

// coreRecordIndex returns the index in core of the record whose head anchor
// is headAnchor, or -1 when none matches.
func coreRecordIndex(core []model.Record, headAnchor string) int {
	for i, r := range core {
		if r.Metadata.HeadAnchor == headAnchor {
			return i
		}
	}
	return -1
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
	contractPath := fs.String("contract", "", "bound acceptance contract document; required at the checkpoint gate (INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE) on a contract-bound run")
	checkpointPolicyPath := fs.String("checkpoint-policy", "", "checkpoint-trigger policy document; required at the checkpoint gate when the bound contract carries a checkpoint_policy_digest")
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
		if err := requireHostTransitionGateIn(*storePath, in, history, *contractPath, *checkpointPolicyPath); err != nil {
			return err
		}
		if o := in.ExpectedApproval.ProposedTargetOutcome; o == model.OutcomeVerifiedSuccess || o == model.OutcomeVerifiedWithWaivers || o == model.OutcomePartial {
			// Decision D10 / architecture section 20: a success-claiming
			// outcome requires a contract-bound run.
			core, err := store.New(*storePath).Load(in.RunID)
			if err != nil {
				return err
			}
			if engine.BoundContractDigest(core) == "" {
				return errors.New(reasonContractRequiredForSuccessClaim)
			}
			// The closure artifact is what HMA observes now in the
			// transition's repository at its base.
			snap, err := repostate.Snapshot(context.Background(), in.RepositoryRoot, in.BaseRevision, in.DirtyWorktreeApproved)
			if err != nil {
				return err
			}
			artifact := hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree)
			if err := requireFindingsResolved(*storePath, in.RunID, ""); err != nil {
				return err
			}
			if o == model.OutcomeVerifiedSuccess {
				if len(engine.ProjectResolutionState(core).ActiveWaivers) > 0 {
					return errors.New(hoststate.ReasonFindingsUnresolved)
				}
			}
			if err := requireIndependentCredit(*storePath, in.RunID, artifact, history); err != nil {
				return err
			}
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

// requireOnlyOwnSnapshotsSince allows the core chain to have moved past the
// review's bound head only by findings snapshots imported from that same
// review (hma host findings-import). Any other core record since the review
// makes it stale.
func requireOnlyOwnSnapshotsSince(storePath, runID, boundHead, reviewDigest string) error {
	core, err := store.New(storePath).Load(runID)
	if err != nil {
		return err
	}
	at := -1
	for i, r := range core {
		if r.Metadata.HeadAnchor == boundHead {
			at = i
		}
	}
	if at < 0 {
		return errors.New(hoststate.ReasonCoreBindingStale)
	}
	for _, r := range core[at+1:] {
		if r.Kind != model.KindFinding || !r.FindingsSnapshot || r.FindingsSource == nil || r.FindingsSource.ReviewRecordDigest != reviewDigest {
			return errors.New(hoststate.ReasonCoreBindingStale)
		}
	}
	return nil
}
