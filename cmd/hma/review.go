package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// hostReviewPacketInput asks HMA to generate a review packet. The dispatch is
// a review dispatch without packet binding or packet digest; HMA adds both.
type hostReviewPacketInput struct {
	hostEnvelope
	Dispatch       hoststate.Dispatch      `json:"dispatch"`
	DispatchRunID  string                  `json:"dispatch_run_id"`
	IssuingReason  hoststate.IssuingReason `json:"issuing_reason"`
	RepositoryRoot string                  `json:"repository_root"`
	BaseRevision   string                  `json:"base_revision"`
	// DirtyWorktreeApproved permits reviewing uncommitted content; its exact
	// bytes are then part of the artifact.
	DirtyWorktreeApproved bool   `json:"dirty_worktree_approved"`
	Task                  string `json:"task"`
	// ContractPath is the approved acceptance contract. When the run has a
	// bound contract it is required and its digest must equal the bound one;
	// the packet carries it so the reviewer sees what was approved.
	ContractPath string `json:"contract_path,omitempty"`
	PolicyPath   string `json:"policy_path,omitempty"`
	PlanPath     string `json:"plan_path,omitempty"`
	// CheckpointPolicyPath is the checkpoint-trigger policy document
	// (architecture §9, §16). Required exactly when the bound contract
	// carries a checkpoint_policy_digest; with no digest, every changed
	// path is unclassified and this is ignored.
	CheckpointPolicyPath string `json:"checkpoint_policy_path,omitempty"`
	// LedgerDir is the agent-dispatch ledger the reviewer will launch in;
	// review import reads the receipt only from it, and reconciliation
	// scans it.
	LedgerDir string `json:"ledger_dir"`
}

type hostReviewImportInput struct {
	hostEnvelope
	ReviewDispatchID string `json:"review_dispatch_id"`
	LedgerDir        string `json:"ledger_dir"`
	ReportPath       string `json:"report_path"`
}

type reviewPacketResult struct {
	hostResult
	PacketPath       string `json:"packet_path"`
	PacketSHA256     string `json:"packet_sha256"`
	ReviewDispatchID string `json:"review_dispatch_id"`
	DispatchRunID    string `json:"dispatch_run_id"`
}

type reviewImportResult struct {
	hostResult
	Verdict hoststate.ReviewVerdict `json:"verdict"`
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildReviewPacket renders the packet the reviewer receives. Its header is
// the binding the report must copy back; the packet digest itself is the
// SHA-256 of these bytes, which the reviewer reads from the dispatch receipt
// instruction rather than from the packet.
// packetCoreState is the bound HMA state a review packet carries, so the
// reviewer sees the approved plan, criteria, current findings, and waivers it
// reviews against, not only the caller's task text.
type packetCoreState struct {
	CoreHeadAnchor     string                `json:"core_head_anchor"`
	AcceptedPlanDigest string                `json:"accepted_plan_digest"`
	PlanSourceRecord   string                `json:"plan_source_record,omitempty"`
	RouteApproval      string                `json:"route_approval_digest"`
	Criteria           []model.Criterion     `json:"criteria"`
	Findings           []model.Finding       `json:"findings"`
	ActiveWaivers      []model.ScopeSelector `json:"active_waivers"`
	// The fixed validator packet of architecture §16: the approved request,
	// criteria and non-goals, the plan unit and its budgets, the evidence
	// bundle and the applicable policy. Absent only for a run with no bound
	// acceptance contract.
	Contract     *packetContract              `json:"contract,omitempty"`
	Unit         *packetUnit                  `json:"unit,omitempty"`
	Evidence     []packetEvidence             `json:"evidence,omitempty"`
	PolicyDigest string                       `json:"policy_digest,omitempty"`
	RequestBasis string                       `json:"request_basis,omitempty"`
	Request      string                       `json:"request,omitempty"`
	Policy       *model.RoutePolicyProjection `json:"policy,omitempty"`
	Manifest     []packetFile                 `json:"changed_files,omitempty"`
	// LineCounts names how added/deleted are counted: the minimal edit
	// distance between the two sides (not git's heuristic numstat, which can
	// be higher). Present with the manifest, i.e. for contract-bound packets.
	LineCounts  string         `json:"line_counts,omitempty"`
	WaiverTerms []packetWaiver `json:"waiver_terms,omitempty"`
}

type packetContract struct {
	Digest string `json:"digest"`
	model.AcceptanceContract
}

type packetUnit struct{ model.PlanUnit }

type packetEvidence struct {
	Record  string `json:"record"`
	Current bool   `json:"current"`
	model.EvidenceRef
}

type packetWaiver struct {
	Selector  model.ScopeSelector    `json:"selector"`
	Record    string                 `json:"record"`
	Operation *model.WaiverOperation `json:"waiver_operation,omitempty"`
	Waiver    *model.Waiver          `json:"waiver,omitempty"`
	Legacy    bool                   `json:"legacy,omitempty"`
}
type packetFile struct {
	Path           string `json:"path"`
	Status         string `json:"status"`
	Added          int    `json:"added"`
	Deleted        int    `json:"deleted"`
	SHA256         string `json:"sha256"`
	OldMode        string `json:"old_mode,omitempty"`
	NewMode        string `json:"new_mode,omitempty"`
	Binary         bool   `json:"binary,omitempty"`
	Staged         bool   `json:"staged,omitempty"`
	LinesTruncated bool   `json:"lines_truncated,omitempty"`
	// LinesUpperBound: the edit-distance work or line cap was hit; added and
	// deleted are an upper bound (every line of both sides). LinesTruncated:
	// a side exceeded the 16 MiB window; added and deleted count only the first
	// 16 MiB of each side and are not a bound on the whole change.
	LinesUpperBound bool `json:"lines_upper_bound,omitempty"`
}

// reviewPacketBeforeManifestHook lets tests model a target mutation between
// the packet snapshot and manifest collection.
var reviewPacketBeforeManifestHook func()

// corePacketState assembles the packet's bound core state for unit. It
// refuses when a mandatory input is missing: no approved route for the unit,
// or no accepted plan digest on that approval.
func corePacketState(storePath string, core []model.Record, runID, unit, contractPath, policyPath, planPath string, binding hoststate.ReviewPacketBinding) (packetCoreState, error) {
	state, _, _, err := corePacketStateWithPlan(storePath, core, runID, unit, contractPath, policyPath, planPath, binding, "")
	return state, err
}

func corePacketStateWithPlan(storePath string, core []model.Record, runID, unit, contractPath, policyPath, planPath string, binding hoststate.ReviewPacketBinding, repositoryIdentity string) (packetCoreState, *model.Plan, string, error) {
	route, err := approvedCoreRouteDigestIn(core, runID, unit)
	if err != nil {
		return packetCoreState{}, nil, "", err
	}
	plan := leasePlanDigest(core, unit)
	planRecord := leasePlanSourceRecord(core, unit)
	if plan == "" || planRecord == "" {
		return packetCoreState{}, nil, "", errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	res := engine.ProjectResolutionState(core)
	st := packetCoreState{CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor, AcceptedPlanDigest: plan, PlanSourceRecord: planRecord, RouteApproval: route, Criteria: res.Criteria, Findings: []model.Finding{}, ActiveWaivers: []model.ScopeSelector{}}
	for _, f := range hoststate.AssignLegacyOwners(core, res.Findings) {
		if f.UnitID == unit || f.UnitID == "" {
			st.Findings = append(st.Findings, f)
		}
	}
	suppliedPlan, suppliedPlanDigest, err := addContractStateWithPlan(&st, storePath, core, unit, contractPath, policyPath, planPath)
	if err != nil {
		return packetCoreState{}, nil, "", err
	}
	if st.Contract == nil {
		// A run without a bound contract keeps the packet exactly as before
		// D5c: the plan digest the unit's route approval carries.
		st.AcceptedPlanDigest = routeApprovalPlanDigest(core, unit, route)
		st.PlanSourceRecord = ""
	}
	for w, on := range res.ActiveWaivers {
		if on {
			st.ActiveWaivers = append(st.ActiveWaivers, w)
		}
	}
	if st.Contract != nil {
		st.WaiverTerms, err = packetWaiverTerms(core, res.ActiveWaivers)
		if err != nil {
			return packetCoreState{}, nil, "", err
		}
	}
	sort.Slice(st.ActiveWaivers, func(i, j int) bool { return fmt.Sprint(st.ActiveWaivers[i]) < fmt.Sprint(st.ActiveWaivers[j]) })
	if st.Contract != nil {
		st.Evidence = packetEvidenceForTarget(core, unit, binding, repositoryIdentity)
	}
	return st, suppliedPlan, suppliedPlanDigest, nil
}

func packetWaiverTerms(core []model.Record, active map[model.ScopeSelector]bool) ([]packetWaiver, error) {
	latest := make(map[model.ScopeSelector]packetWaiver)
	for _, rec := range core {
		for i := range rec.Waivers {
			w := rec.Waivers[i]
			if w.Active {
				copy := w
				latest[w.Scope] = packetWaiver{Selector: w.Scope, Record: rec.Metadata.HeadAnchor, Waiver: &copy, Legacy: true}
			} else {
				delete(latest, w.Scope)
			}
		}
		if w := rec.WaiverOperation; w != nil {
			switch w.Operation {
			case model.WaiverOperationGrant:
				copy := *w
				latest[w.Scope] = packetWaiver{Selector: w.Scope, Record: rec.Metadata.HeadAnchor, Operation: &copy}
			case model.WaiverOperationExpire, model.WaiverOperationWithdraw:
				delete(latest, w.Scope)
			}
		}
	}
	terms := make([]packetWaiver, 0, len(active))
	for selector, on := range active {
		if !on {
			continue
		}
		term, ok := latest[selector]
		if !ok {
			return nil, errors.New(hoststate.ReasonReviewPacketWaiverIncomplete)
		}
		terms = append(terms, term)
	}
	sort.Slice(terms, func(i, j int) bool {
		if terms[i].Selector.Kind != terms[j].Selector.Kind {
			return terms[i].Selector.Kind < terms[j].Selector.Kind
		}
		return terms[i].Selector.Target < terms[j].Selector.Target
	})
	return terms, nil
}

func packetEvidenceForTarget(core []model.Record, unit string, binding hoststate.ReviewPacketBinding, repositoryIdentity ...string) []packetEvidence {
	identity := ""
	if len(repositoryIdentity) > 0 {
		identity = repositoryIdentity[0]
	}
	evidence := []packetEvidence{}
	for _, r := range core {
		if r.Kind == model.KindEvidence && r.Evidence != nil && (r.UnitID == unit || r.UnitID == "") {
			e := *r.Evidence
			current := e.RepositoryIdentity == identity && e.BaseRevision == binding.BaseRevision && e.HeadRevision == binding.ProducedHead && e.DiffDigest == binding.DiffDigest && e.WorktreeDigest == binding.WorktreeDigest
			evidence = append(evidence, packetEvidence{Record: r.Metadata.HeadAnchor, Current: current, EvidenceRef: e})
		}
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		if evidence[i].Current != evidence[j].Current {
			return evidence[i].Current
		}
		return evidence[i].Record < evidence[j].Record
	})
	return evidence
}

// checkpointPacketHeader renders the §3.5a checkpoint binding fields for the
// packet header, so a reviewer sees why the review was issued; empty when
// the classifier did not run (no checkpoint_schema).
func checkpointPacketHeader(b hoststate.ReviewPacketBinding) string {
	if b.CheckpointSchema == "" {
		return ""
	}
	covers := ""
	if b.CheckpointCovers != nil {
		cb, _ := json.Marshal(b.CheckpointCovers)
		covers = string(cb)
	}
	return fmt.Sprintf(`checkpoint_schema: %s
minimum_issuing_reason: %s
checkpoint_triggers: %s
unclassified_paths: %s
family_separation: %t
risk_floor: %s
contract_risk: %s
bound_contract_digest: %s
checkpoint_policy_digest: %s
manifest_digest: %s
checkpoint_covers: %s
`, b.CheckpointSchema, b.MinimumIssuingReason, strings.Join(b.CheckpointTriggers, ","), strings.Join(b.UnclassifiedPaths, ","),
		b.FamilySeparation, b.RiskFloor, b.ContractRisk, b.BoundContractDigest, b.CheckpointPolicyDigest, b.ManifestDigest, covers)
}

func buildReviewPacket(in hostReviewPacketInput, binding hoststate.ReviewPacketBinding, state packetCoreState) []byte {
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	return []byte(fmt.Sprintf(`HMA review packet
review_dispatch_id: %s
review_nonce: %s
dispatch_run_id: %s
unit_id: %s
issuing_reason: %s
artifact_digest: %s
implementation_record_digest: %s
base_revision: %s
produced_head: %s
diff_digest: %s
worktree_digest: %s
valid_until: %s
%s
Report binding (required): your AUDIT.json must contain a top-level object
"hma_binding": {"packet_sha256": <SHA-256 hex of this packet file's bytes>,
"review_dispatch_id": %q, "review_nonce": %q}.
A report without this exact binding is not imported.

Bound HMA state (review against it; it is part of this packet's digest):
%s

Task:
%s
`, binding.ReviewDispatchID, binding.ReviewNonce, binding.DispatchRunID, in.Dispatch.UnitID, binding.IssuingReason,
		in.Dispatch.ArtifactDigest, in.Dispatch.ImplementationRecordDigest, binding.BaseRevision, binding.ProducedHead,
		binding.DiffDigest, binding.WorktreeDigest, binding.ValidUntil, checkpointPacketHeader(binding),
		binding.ReviewDispatchID, binding.ReviewNonce, stateJSON, in.Task))
}

// issuingLevelRank orders engine.IssuingLevel the same way hoststate's
// IssuingReason is ordered (per_change < checkpoint < final); the two types
// share the same underlying string values (internal/engine/checkpoint.go).
func issuingLevelRank(level engine.IssuingLevel) int {
	switch level {
	case engine.IssuingLevelPerChange:
		return 1
	case engine.IssuingLevelCheckpoint:
		return 2
	case engine.IssuingLevelFinal:
		return 3
	default:
		return 0
	}
}

// toChangedPaths projects a packet's changed-file manifest into the
// classifier's input shape (internal/engine.ChangedPath).
func toChangedPaths(manifest []packetFile) []engine.ChangedPath {
	out := make([]engine.ChangedPath, 0, len(manifest))
	for _, f := range manifest {
		out = append(out, engine.ChangedPath{Path: f.Path, Status: f.Status, OldMode: f.OldMode, NewMode: f.NewMode, Binary: f.Binary})
	}
	return out
}

// sortedCopy returns a sorted copy of values, so a CheckpointBaseline's
// permission classes serialize deterministically regardless of a plan's
// declared order.
func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

// loadCheckpointPolicy reads and validates the checkpoint policy document
// the host names, when the bound contract carries a checkpoint_policy_digest
// (architecture §9). A contract with no checkpoint_policy_digest returns a
// nil policy: the classifier then treats every changed path as unclassified
// (§16).
func loadCheckpointPolicy(path, digest string) (*model.CheckpointPolicy, error) {
	if digest == "" {
		return nil, nil
	}
	if path == "" {
		return nil, errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	policy, got, err := model.ParseCheckpointPolicy(raw)
	if err != nil || got != digest {
		return nil, errors.New(hoststate.ReasonPolicyNotCurrent)
	}
	return &policy, nil
}

// firstContractAcceptance returns the acceptance contract digest of the
// run's first (not latest) approved ACCEPTANCE_CRITERIA -> PLANNING
// transition, or "" when the run never accepted one. It is the classifier's
// fallback contract baseline (§3.4 "Baseline") when the unit has no earlier
// checkpoint-grade review.
func firstContractAcceptance(core []model.Record) string {
	for _, r := range core {
		if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.CurrentStage == model.StageAcceptanceCriteria && r.Approval.ProposedTargetStage == model.StagePlanning {
			return r.Approval.AcceptanceContractDigest
		}
	}
	return ""
}

// firstLeasePermissionClasses returns the unit's permission classes in the
// plan of its first (not latest) lease generation, or nil when none is
// retained. It is the classifier's fallback permission-class baseline
// (§3.4 "Baseline") when the unit has no earlier checkpoint-grade review.
func firstLeasePermissionClasses(storePath string, core []model.Record, unit string) ([]string, error) {
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			plans, err := loadPlans(storePath, core)
			if err != nil {
				return nil, err
			}
			plan, ok := plans[r.Approval.AcceptedPlanDigest]
			if !ok {
				return nil, nil
			}
			for _, u := range plan.Units {
				if u.ID == unit {
					return u.PermissionClasses, nil
				}
			}
			return nil, nil
		}
	}
	return nil, nil
}

// checkpointReviewBaseline derives the unit's checkpoint baseline (§3.4
// "Baseline"): the engine.CheckpointBaseline recorded as checkpoint_covers
// on the unit's latest approved review dispatched at level checkpoint or
// final, or, with no such review, the run's first contract acceptance
// digest and the unit's permission classes in its first lease generation's
// plan. The first contract does not fire T1 by itself; any re-acceptance
// fires T1 until a checkpoint-grade review passes on it.
func checkpointReviewBaseline(storePath string, history []hoststate.Record, core []model.Record, unit string) (engine.CheckpointBaseline, error) {
	for i := len(history) - 1; i >= 0; i-- {
		r := history[i]
		if r.Kind != hoststate.KindReview || r.Review == nil || r.Review.UnitID != unit || r.Review.Verdict != hoststate.ReviewApprove {
			continue
		}
		p := packetFor(history, r.Review.ReviewDispatchDigest)
		if p == nil || p.CheckpointSchema == "" || p.CheckpointCovers == nil {
			continue
		}
		if p.IssuingReason != hoststate.IssueCheckpoint && p.IssuingReason != hoststate.IssueFinal {
			continue
		}
		return *p.CheckpointCovers, nil
	}
	classes, err := firstLeasePermissionClasses(storePath, core, unit)
	if err != nil {
		return engine.CheckpointBaseline{}, err
	}
	return engine.CheckpointBaseline{ContractDigest: firstContractAcceptance(core), PermissionClasses: sortedCopy(classes)}, nil
}

func runHostReviewPacket(args []string) error {
	fs := flag.NewFlagSet("review-packet", flag.ContinueOnError)
	input := fs.String("input", "", "host input file")
	storePath := fs.String("store", "", "HMA store directory")
	out := fs.String("out", "", "packet file to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" || *storePath == "" || *out == "" || fs.NArg() != 0 {
		return errors.New("usage: hma host review-packet --input <file> --store <directory> --out <packet file>")
	}
	var in hostReviewPacketInput
	if err := decodeStrictFile(*input, &in); err != nil {
		return err
	}
	at, err := validateHostEnvelope(in.hostEnvelope)
	if err != nil {
		return err
	}
	if in.Dispatch.Role != hoststate.RoleReview || in.Dispatch.Packet != nil || in.Dispatch.ReviewPacketDigest != "" || in.Task == "" || in.RepositoryRoot == "" || in.BaseRevision == "" || in.LedgerDir == "" || !filepath.IsAbs(in.LedgerDir) {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	root, err := filepath.Abs(in.RepositoryRoot)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return err
	}
	// The packet must not change the worktree the receipt will describe.
	if err := repostate.StoreOutsideRepository(root, filepath.Dir(*out)); err != nil {
		return fmt.Errorf("packet path: %w", err)
	}
	// The packet must review the approved base in the run's repository root;
	// checked first, so a wrong root or base fails with its named reason.
	if core, err := store.New(*storePath).Load(in.RunID); err != nil {
		return err
	} else if history, err := hoststate.NewStore(*storePath).Load(in.RunID); err != nil {
		return err
	} else if len(core) > 0 {
		if err := requireApprovedRepository(history, core, in.RunID, in.Dispatch.UnitID, &hoststate.ReviewPacketBinding{ProjectRoot: root, BaseRevision: in.BaseRevision}); err != nil {
			return err
		}
	}
	// HMA observes the artifact itself; the host may name the digest it
	// expects, but never supplies it.
	captured, err := captureTarget(boundedCtx(), root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return err
	}
	snap := captured.State
	artifact := hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree)
	if in.Dispatch.ArtifactDigest != "" && in.Dispatch.ArtifactDigest != artifact {
		return errors.New(hoststate.ReasonReviewArtifactMismatch)
	}
	in.Dispatch.ArtifactDigest = artifact
	head, worktree, err := repostate.LaunchDigests(context.Background(), root)
	if err != nil {
		return err
	}
	if head != snap.Head {
		return errors.New(hoststate.ReasonReviewArtifactMismatch)
	}
	nonce, err := randomHex(16)
	if err != nil {
		return err
	}
	id, err := randomHex(16)
	if err != nil {
		return err
	}
	binding := hoststate.ReviewPacketBinding{
		ReviewDispatchID: "hma-rd-" + id, ReviewNonce: nonce, DispatchRunID: in.DispatchRunID,
		IssuingReason: in.IssuingReason, ExpectedLaunchHead: head, ExpectedLaunchWorktree: worktree,
		ProjectRoot: root, BaseRevision: in.BaseRevision, ProducedHead: snap.Head, DiffDigest: snap.Diff, WorktreeDigest: snap.Worktree,
		ValidUntil:      at.Add(hoststate.ReviewDispatchTTL).UTC().Format(time.RFC3339),
		LedgerNamespace: filepath.Clean(in.LedgerDir),
	}
	core, err := store.New(*storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if len(core) == 0 {
		return errors.New(hoststate.ReasonCoreRunMissing)
	}
	history, err := hoststate.NewStore(*storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	repositoryIdentity := ""
	if repository := hoststate.RunRepository(history); repository != nil {
		repositoryIdentity = repository.RepositoryIdentity
	}
	state, suppliedPlan, suppliedPlanDigest, err := corePacketStateWithPlan(*storePath, core, in.RunID, in.Dispatch.UnitID, in.ContractPath, in.PolicyPath, in.PlanPath, binding, repositoryIdentity)
	if err != nil {
		return err
	}
	// decision is the checkpoint classifier's output (architecture §16),
	// computed below only for a contract-bound run (§12.4 "Checkpoint
	// level, review risk, and checkpoint binding" applies to contract-bound
	// runs only); its zero value is never read otherwise. The checkpoint
	// policy and the classifier's baseline (which may read a retained plan,
	// checkpointReviewBaseline) are read here, in the same single read phase
	// as corePacketStateWithPlan's contract/plan/route-policy reads and
	// before reviewPacketBeforeManifestHook's injection point: re-reading
	// either from disk after that point would reopen the D5c TOCTOU class
	// corePacketStateWithPlan already closes (TestD5C5PBoundInputSwapsAfterParse).
	var decision engine.CheckpointDecision
	var checkpointPolicy *model.CheckpointPolicy
	var checkpointBaseline engine.CheckpointBaseline
	if state.Contract != nil {
		checkpointPolicy, err = loadCheckpointPolicy(in.CheckpointPolicyPath, state.Contract.CheckpointPolicyDigest)
		if err != nil {
			return err
		}
		checkpointBaseline, err = checkpointReviewBaseline(*storePath, history, core, in.Dispatch.UnitID)
		if err != nil {
			return err
		}
	}
	if state.Contract != nil {
		if reviewPacketBeforeManifestHook != nil {
			reviewPacketBeforeManifestHook()
		}
		state.Manifest, err = buildCapturedManifest(boundedCtx(), root, in.BaseRevision, in.DirtyWorktreeApproved, &captured)
		if err != nil {
			return err
		}
		state.LineCounts = "minimal-lcs"
		second, err := captureTarget(boundedCtx(), root, in.BaseRevision, in.DirtyWorktreeApproved)
		if err != nil {
			return err
		}
		if second.State.Head != binding.ProducedHead || second.State.Diff != binding.DiffDigest || second.State.Worktree != binding.WorktreeDigest {
			return errors.New(hoststate.ReasonReviewTargetChanged)
		}

		current := engine.CheckpointBaseline{ContractDigest: state.Contract.Digest, PermissionClasses: sortedCopy(state.Unit.PermissionClasses)}
		decision = engine.ClassifyCheckpoint(engine.CheckpointInput{
			Contract: &state.Contract.AcceptanceContract, Policy: checkpointPolicy, Unit: state.Unit.PlanUnit,
			Baseline: checkpointBaseline, Current: current, Manifest: toChangedPaths(state.Manifest),
			Stage: engine.ProjectStageState(core).CurrentStage, HostRisk: in.Dispatch.Risk, Disputed: in.Dispatch.Disputed,
		})
		// §3.5 E0 step 2: HMA refuses rather than silently raising the
		// host's declared reason.
		if issuingLevelRank(engine.IssuingLevel(in.IssuingReason)) < issuingLevelRank(decision.Level) {
			return fmt.Errorf("%s: minimum issuing reason is %s", hoststate.ReasonCheckpointReasonTooLow, decision.Level)
		}
		triggers := make([]string, 0, len(decision.Triggers))
		for _, tr := range decision.Triggers {
			triggers = append(triggers, string(tr))
		}
		binding.CheckpointSchema = hoststate.CheckpointSchemaV1
		binding.MinimumIssuingReason = hoststate.IssuingReason(decision.Level)
		binding.CheckpointTriggers = triggers
		binding.UnclassifiedPaths = decision.UnclassifiedPaths
		binding.FamilySeparation = decision.FamilySeparation
		binding.RiskFloor = decision.RiskFloor
		binding.ContractRisk = state.Contract.Risk
		binding.BoundContractDigest = state.Contract.Digest
		binding.CheckpointPolicyDigest = decision.PolicyDigest
		binding.ManifestDigest = decision.ManifestDigest
		binding.CheckpointCovers = &current
	}
	// Checked before the packet file is written, and again under the locks.
	if err := requireApprovedRepository(history, core, in.RunID, in.Dispatch.UnitID, &binding); err != nil {
		return err
	}
	packet := buildReviewPacket(in, binding, state)
	if err := validateReviewPacketSize(packet, state); err != nil {
		return err
	}
	sum := sha256.Sum256(packet)
	in.Dispatch.Packet = &binding
	in.Dispatch.ReviewPacketDigest = hex.EncodeToString(sum[:])

	// Create the packet first, exclusively; remove it if the record is not
	// appended, so no packet exists without its dispatch record.
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(packet)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = syncParent(*out)
	}
	if err != nil {
		os.Remove(*out)
		return err
	}
	result, err := appendHostRecord(*storePath, in.hostEnvelope, hoststate.KindDispatch, nil, nil, &in.Dispatch, nil, nil, func(history []hoststate.Record, core []model.Record) error {
		if core[len(core)-1].Metadata.HeadAnchor != state.CoreHeadAnchor {
			return errors.New(hoststate.ReasonCoreBindingStale)
		}
		// G-R: a host cannot dispatch a review risk below the required
		// review risk, max(contract risk, checkpoint risk floor)
		// (decision.RequiredRisk, §3.5 E0 step 4). state.Contract was
		// already proved equal to the bound contract digest
		// (corePacketStateWithPlan); the head check above proves the core
		// has not moved since.
		if state.Contract != nil && riskRank(in.Dispatch.Risk) < riskRank(decision.RequiredRisk) {
			return errors.New(hoststate.ReasonReviewRiskBelowRequired)
		}
		if err := validateRouteApprovalClaimIn(core, in.RunID, in.Dispatch.UnitID, in.Dispatch.Route.RouteApprovalDigest); err != nil {
			return err
		}
		if err := requireApprovedRepository(history, core, in.RunID, in.Dispatch.UnitID, in.Dispatch.Packet); err != nil {
			return err
		}
		if err := hoststate.EvaluateDispatch(history, in.Dispatch, at); err != nil {
			return err
		}
		if suppliedPlan != nil {
			return retainPlan(*storePath, *suppliedPlan, suppliedPlanDigest)
		}
		return nil
	})
	if err != nil {
		os.Remove(*out)
		return err
	}
	result.DispatchAllowed = true
	result.NextAction = "START_REVIEWER"
	return json.NewEncoder(os.Stdout).Encode(reviewPacketResult{
		hostResult: result, PacketPath: *out, PacketSHA256: in.Dispatch.ReviewPacketDigest,
		ReviewDispatchID: binding.ReviewDispatchID, DispatchRunID: binding.DispatchRunID,
	})
}

// riskRank orders model.RouteRiskLevel the same way the route-coherence
// check does (internal/engine/route_coherence.go's routeRiskRank): low <
// medium < high < critical. An unrecognized value ranks lowest so it never
// satisfies a required risk.
func riskRank(risk model.RouteRiskLevel) int {
	switch risk {
	case model.RouteRiskLow:
		return 1
	case model.RouteRiskMedium:
		return 2
	case model.RouteRiskHigh:
		return 3
	case model.RouteRiskCritical:
		return 4
	default:
		return 0
	}
}

func validateReviewPacketSize(packet []byte, state packetCoreState) error {
	if state.Contract != nil && (len(packet) > 2<<20 || len(state.Evidence) > maxObservedEntries || len(state.Manifest) > maxObservedEntries) {
		return errors.New(hoststate.ReasonReviewPacketTooLarge)
	}
	return nil
}

func runHostReviewImport(args []string) error {
	inputPath, storePath, err := hostCommandFlags("review-import", args)
	if err != nil {
		return err
	}
	var in hostReviewImportInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	if in.ReviewDispatchID == "" || in.LedgerDir == "" || in.ReportPath == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	var review hoststate.IndependentReview
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindReview, nil, nil, nil, &review, nil, func(history []hoststate.Record, core []model.Record) error {
		var runID string
		for _, r := range history {
			if r.Kind == hoststate.KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == in.ReviewDispatchID {
				runID = r.Dispatch.Packet.DispatchRunID
				// Any core record since the packet (rewind, terminal,
				// other findings) makes the packet stale.
				if r.CoreHeadAnchor != core[len(core)-1].Metadata.HeadAnchor {
					return errors.New(hoststate.ReasonCoreBindingStale)
				}
			}
		}
		if runID == "" {
			return errors.New(hoststate.ReasonReviewMissing)
		}
		// HMA reads the receipt from the ledger by run id, not from a path
		// the host names.
		// The receipt must come from the ledger the packet named.
		if p := packetFor(history, findDispatchDigest(history, in.ReviewDispatchID)); p != nil && p.LedgerNamespace != "" && filepath.Clean(in.LedgerDir) != p.LedgerNamespace {
			return errors.New(hoststate.ReasonReviewReceiptMismatch)
		}
		receipt, err := os.ReadFile(filepath.Join(in.LedgerDir, "runs", runID, "receipt.json"))
		if err != nil {
			return errors.New(hoststate.ReasonReviewReceiptMismatch)
		}
		report, err := os.ReadFile(in.ReportPath)
		if err != nil {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		if err := requireLiveTarget(history, findDispatchDigest(history, in.ReviewDispatchID)); err != nil {
			return err
		}
		if p := packetFor(history, findDispatchDigest(history, in.ReviewDispatchID)); p == nil {
			return errors.New(hoststate.ReasonReviewMissing)
		} else if err := requireApprovedRepository(history, core, in.RunID, reviewUnit(history, in.ReviewDispatchID), p); err != nil {
			return err
		}
		imported, err := hoststate.EvaluateReviewImport(history, hoststate.ReviewImport{
			ReviewDispatchID: in.ReviewDispatchID, ReceiptBytes: receipt, ReportPath: in.ReportPath, ReportBytes: report,
		})
		if err != nil {
			return err
		}
		if _, err := approvedCoreRouteDigestIn(core, in.RunID, imported.UnitID); err != nil {
			return err
		}
		if err := hoststate.EvaluateReview(history, imported); err != nil {
			return err
		}
		review = imported
		return nil
	})
	if err != nil {
		return err
	}
	switch review.Verdict {
	case hoststate.ReviewApprove:
		result.NextAction = "CONTINUE_HMA_TRANSITION"
	case hoststate.ReviewRequestChanges:
		result.NextAction = "RECORD_FINDINGS_AND_REMEDIATE"
	default:
		result.NextAction = "ESCALATE_TO_HUMAN"
	}
	return json.NewEncoder(os.Stdout).Encode(reviewImportResult{hostResult: result, Verdict: review.Verdict})
}

type hostFindingsImportInput struct {
	hostEnvelope
	ReviewRecordDigest string `json:"review_record_digest"`
	ReportPath         string `json:"report_path"`
}

type findingsImportResult struct {
	Recorded       bool            `json:"recorded"`
	CoreSequence   int64           `json:"core_sequence"`
	CoreHeadAnchor string          `json:"core_head_anchor"`
	Findings       []model.Finding `json:"findings"`
	NextAction     string          `json:"next_action"`
	// ReviewEnforcement is the run's review enforcement mode, as on every
	// host write result.
	ReviewEnforcement string `json:"review_enforcement"`
}

// runHostFindingsImport writes one core findings snapshot from the report of
// an imported review. It holds the host run lock while it reads the host
// chain and appends to the core chain, which takes the core lock (host, then
// core); the core append refuses if the core head moved since it was read.
func runHostFindingsImport(args []string) error {
	inputPath, storePath, err := hostCommandFlags("findings-import", args)
	if err != nil {
		return err
	}
	var in hostFindingsImportInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	if in.ReviewRecordDigest == "" || in.ReportPath == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	var out findingsImportResult
	var verdict hoststate.ReviewVerdict
	err = hoststate.NewStore(storePath).WithRunLock(in.RunID, func(history []hoststate.Record) error {
		rr, err := hoststate.RequireLatestImportedReview(history, in.ReviewRecordDigest)
		if err != nil {
			return err
		}
		report, err := os.ReadFile(in.ReportPath)
		if err != nil {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		sum := sha256.Sum256(report)
		if "sha256:"+hex.EncodeToString(sum[:]) != rr.Review.ReportDigest {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		var rep struct {
			Findings json.RawMessage `json:"findings"`
		}
		if err := hoststate.UnmarshalExact(report, &rep); err != nil {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		s := store.New(storePath)
		core, err := s.Load(in.RunID)
		if err != nil {
			return err
		}
		if len(core) == 0 {
			return errors.New(hoststate.ReasonCoreRunMissing)
		}
		if engine.ProjectStageState(core).Terminal {
			return errors.New(hoststate.ReasonCoreRunTerminal)
		}
		last := core[len(core)-1]
		// The review must bind the current core head: a core record since
		// the review (including an earlier snapshot of it) makes it stale,
		// so an old review cannot replace newer findings.
		if rr.CoreHeadAnchor != last.Metadata.HeadAnchor {
			return errors.New(hoststate.ReasonCoreBindingStale)
		}
		live, err := captureLiveTarget(history, rr.Review.ReviewDispatchDigest)
		if err != nil {
			return err
		}
		if p := packetFor(history, rr.Review.ReviewDispatchDigest); p == nil {
			return errors.New(hoststate.ReasonReviewMissing)
		} else if err := requireApprovedRepository(history, core, in.RunID, rr.Review.UnitID, p); err != nil {
			return err
		}
		p := packetFor(history, rr.Review.ReviewDispatchDigest)
		facts, err := findingFacts(boundedCtx(), storePath, core, rr.Review.UnitID, p, p.WorktreeDigest != "", live)
		if err != nil {
			return err
		}
		// The facts were measured from the live repository. Require that the
		// target is still the one the review bound, so they describe it and not
		// a state a writer produced during the measurement.
		if err := requireLiveTarget(history, rr.Review.ReviewDispatchDigest); err != nil {
			return err
		}
		imported, err := hoststate.MapAuditFindings(rep.Findings, facts)
		if err != nil {
			return err
		}
		imported = hoststate.AddEstablishedFindings(imported, facts, rr.Review.ArtifactDigest)
		prior := hoststate.AssignLegacyOwners(core, engine.ProjectResolutionState(core).Findings)
		rec := model.Record{
			Kind: model.KindFinding, Version: 1, UnitID: rr.Review.UnitID,
			Findings:         hoststate.SnapshotFindings(prior, rr.Review.UnitID, imported, rr.Review.Verdict),
			FindingsSnapshot: true,
			FindingsSource:   &model.FindingsSource{ReviewRecordDigest: rr.HeadAnchor, ReportDigest: rr.Review.ReportDigest},
			Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: last.Metadata.Sequence + 1,
				PredecessorHash: last.Metadata.HeadAnchor, Timestamp: in.Timestamp, Actor: in.Actor},
		}
		ids := map[string]bool{}
		for _, f := range rec.Findings {
			if ids[f.ID] {
				return errors.New(hoststate.ReasonFindingsImportInvalid)
			}
			ids[f.ID] = true
		}
		if err := s.Append(&rec); err != nil {
			return err
		}
		out = findingsImportResult{Recorded: true, CoreSequence: rec.Metadata.Sequence, CoreHeadAnchor: rec.Metadata.HeadAnchor, Findings: rec.Findings,
			ReviewEnforcement: hoststate.ReviewEnforcementMode(history)}
		verdict = rr.Review.Verdict
		return nil
	})
	if err != nil {
		return err
	}
	// The review verdict decides first (total map): only an approving
	// review with no remaining blocker recommends continuing.
	switch verdict {
	case hoststate.ReviewApprove:
		out.NextAction = "CONTINUE_HMA_TRANSITION"
		for _, f := range out.Findings {
			if f.Disposition != model.FindingAdvisory {
				out.NextAction = "REMEDIATE_OR_ESCALATE"
			}
		}
	case hoststate.ReviewRequestChanges:
		out.NextAction = "RECORD_FINDINGS_AND_REMEDIATE"
	default:
		out.NextAction = "ESCALATE_TO_HUMAN"
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

// syncParent makes the directory entry of path durable.
func syncParent(path string) error {
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func findDispatchDigest(history []hoststate.Record, reviewDispatchID string) string {
	for _, r := range history {
		if r.Kind == hoststate.KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == reviewDispatchID {
			return r.HeadAnchor
		}
	}
	return ""
}

// requireLiveTarget re-snapshots the repository a packet bound and refuses
// unless head, diff, and worktree still equal the packet's artifact, so a
// review of a historical target never resolves current findings.
func requireLiveTarget(history []hoststate.Record, dispatchDigest string) error {
	_, err := captureLiveTarget(history, dispatchDigest)
	return err
}

// captureLiveTarget is requireLiveTarget that also returns the capture, so a
// measurement taken afterwards can be held to the bytes it covers.
func captureLiveTarget(history []hoststate.Record, dispatchDigest string) (*repostate.Capture, error) {
	var p *hoststate.ReviewPacketBinding
	for _, r := range history {
		if r.HeadAnchor == dispatchDigest && r.Dispatch != nil {
			p = r.Dispatch.Packet
		}
	}
	if p == nil {
		return nil, errors.New(hoststate.ReasonReviewMissing)
	}
	dirty := p.WorktreeDigest != ""
	c, err := captureTarget(context.Background(), p.ProjectRoot, p.BaseRevision, dirty)
	if dirty && err != nil && errors.Is(err, errDirtyObservationUnsupported) {
		return nil, err
	}
	if err != nil || c.State.Head != p.ProducedHead || c.State.Diff != p.DiffDigest || c.State.Worktree != p.WorktreeDigest {
		return nil, errors.New(hoststate.ReasonReviewTargetChanged)
	}
	return &c, nil
}

// packetFor returns the packet binding of the dispatch record with this
// digest, or nil.
func packetFor(history []hoststate.Record, dispatchDigest string) *hoststate.ReviewPacketBinding {
	for _, r := range history {
		if r.HeadAnchor == dispatchDigest && r.Dispatch != nil {
			return r.Dispatch.Packet
		}
	}
	return nil
}

// reviewUnit returns the unit of the HMA-generated review dispatch with this
// review dispatch id.
func reviewUnit(history []hoststate.Record, reviewDispatchID string) string {
	for _, r := range history {
		if r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == reviewDispatchID {
			return r.Dispatch.UnitID
		}
	}
	return ""
}

// approvedRoute returns the unit's active route approval binding.
func approvedRoute(core []model.Record, runID, unit string) (*model.ApprovalBinding, error) {
	route, err := approvedCoreRouteDigestIn(core, runID, unit)
	if err != nil {
		return nil, err
	}
	var a *model.ApprovalBinding
	for _, r := range core {
		if r.UnitID == unit && r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil && r.Approval.TransitionDigest == route {
			a = r.Approval
		}
	}
	if a == nil {
		return nil, errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	return a, nil
}

// requireApprovedRepository refuses a packet that does not review the run's
// bound repository at the unit's approved base: the run must have a
// repository binding (hma host bind-repository, never taken from a packet),
// the packet root must be that root, the unit's route approval must bind that
// repository identity and the packet's base, and the root must contain the
// base. Checked before a packet is written, under the locks at dispatch, and
// at review import and findings import.
func requireApprovedRepository(history []hoststate.Record, core []model.Record, runID, unit string, p *hoststate.ReviewPacketBinding) error {
	bad := errors.New(hoststate.ReasonReviewRepositoryMismatch)
	repo := hoststate.RunRepository(history)
	if repo == nil {
		return errors.New(hoststate.ReasonRepositoryBindingMissing)
	}
	a, err := approvedRoute(core, runID, unit)
	if err != nil {
		return err
	}
	if p == nil || !sameStore(repo.Root, p.ProjectRoot) || a.RepositoryIdentityDigest != repo.RepositoryIdentity || p.BaseRevision != a.BaseRevisionDigest {
		return bad
	}
	if err := repostate.VerifyBase(boundedCtx(), p.ProjectRoot, a.BaseRevisionDigest); err != nil {
		return bad
	}
	return nil
}

// boundedCtx is a context that ends: a git read that blocks (a FIFO named as
// an alternates file, say) must not hold the run locks forever.
func boundedCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	time.AfterFunc(10*time.Minute, cancel)
	return ctx
}

func buildChangedFileManifest(ctx context.Context, root, base string, dirty bool) ([]packetFile, error) {
	return buildCapturedManifest(ctx, root, base, dirty, nil)
}

// buildCapturedManifest is buildChangedFileManifest with the manifest held to a
// capture of the target, when there is one.
func buildCapturedManifest(ctx context.Context, root, base string, dirty bool, capture *repostate.Capture) ([]packetFile, error) {
	entries, err := observeCaptured(ctx, root, base, dirty, maxObservedEntries, capture)
	if err != nil {
		return nil, err
	}
	files := make([]packetFile, 0, len(entries))
	for _, e := range entries {
		files = append(files, packetFile{Path: e.Path, Status: e.Status, Added: e.Added, Deleted: e.Deleted, SHA256: e.SHA256, OldMode: e.OldMode, NewMode: e.NewMode, Binary: e.Binary, Staged: e.Staged, LinesTruncated: e.LinesTruncated, LinesUpperBound: e.LinesUpperBound})
	}
	return files, nil
}

// routeApprovalPlanDigest is the accepted plan digest carried by the unit's
// approved route-selection transition (the value packets held before D5c).
func routeApprovalPlanDigest(core []model.Record, unit, route string) string {
	var plan string
	for _, r := range core {
		if r.UnitID == unit && r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.ProposedTargetStage == model.StageImplementationAuthorization && r.Approval.TransitionDigest == route {
			plan = r.Approval.AcceptedPlanDigest
		}
	}
	return plan
}
