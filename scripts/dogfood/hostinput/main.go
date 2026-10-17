// Command hostinput builds synthetic inputs for the HMA host-orchestration
// dogfood harness (scripts/dogfood/host-dogfood.sh).
//
// It never produces a real human approval. The only approval record it can
// write is an explicitly labelled synthetic fixture, and only into a scratch
// store that this command itself initialised and marked as synthetic. Stage
// transition inputs it emits carry no approval at all, so the core transition
// path must refuse them after the host gate has been evaluated.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

const (
	// markerFile identifies a scratch store created by seed-core. Writing a
	// synthetic approval anywhere else is refused.
	markerFile = ".hma-synthetic-dogfood"

	syntheticActor    = "synthetic-dogfood-harness"
	syntheticApprover = "SYNTHETIC-DOGFOOD-FIXTURE (not a human approval)"
	syntheticHost     = "synthetic-dogfood-host"

	// The review artifact the synthetic review names and the
	// review->verification input binds (see hoststate.ReviewArtifactDigest).
	syntheticHead = "synthetic-produced-head"
	syntheticDiff = "synthetic-diff"
)

// baseTime anchors every synthetic timestamp so runs are reproducible.
var baseTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func stamp(minute int) string {
	return baseTime.Add(time.Duration(minute) * time.Minute).Format(time.RFC3339)
}

// Mirrors the unexported envelope types in cmd/hma/host.go so the emitted
// JSON is accepted by the strict decoder there.
type envelope struct {
	RunID     string `json:"run_id"`
	Actor     string `json:"actor"`
	Timestamp string `json:"timestamp"`
}

type decisionInput struct {
	envelope
	Decision hoststate.OrchestrationDecision `json:"decision"`
}

type preflightInput struct {
	envelope
	Preflight hoststate.RoutePreflight `json:"preflight"`
}

type dispatchInput struct {
	envelope
	Dispatch hoststate.Dispatch `json:"dispatch"`
}

type reviewInput struct {
	envelope
	Review hoststate.IndependentReview `json:"review"`
}

func route(review bool, approval string) hoststate.Route {
	r := hoststate.Route{
		WorkerID: "synthetic-impl-worker", Provider: "synthetic-provider-a", ProviderFamily: "synthetic-family-a",
		ProviderDigest: "synthetic-provider-a-digest", Model: "synthetic-model-a", ModelFamily: "synthetic-model-family-a",
		ModelDigest: "synthetic-model-a-digest", ProfileDigest: "synthetic-impl-profile",
		AccessService: "synthetic-service-a", AccessServiceDigest: "synthetic-service-a-digest",
		Runtime: "synthetic-runtime", ReasoningEffort: "medium", ExecutionContext: model.ExecutionContextProxy,
		CapabilityClasses:   []string{model.CapabilityBoundedImplementation, model.CapabilityEvidenceOutputCapture},
		PermissionClasses:   []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect},
		RouteApprovalDigest: approval,
	}
	if review {
		// Distinct worker, model, provider family, and model family so the
		// high-risk independence rule is satisfied.
		r.WorkerID, r.Provider, r.ProviderFamily = "synthetic-review-worker", "synthetic-provider-b", "synthetic-family-b"
		r.ProviderDigest, r.Model, r.ModelFamily = "synthetic-provider-b-digest", "synthetic-model-b", "synthetic-model-family-b"
		r.ModelDigest, r.ProfileDigest = "synthetic-model-b-digest", "synthetic-review-profile"
		r.AccessService, r.AccessServiceDigest = "synthetic-service-b", "synthetic-service-b-digest"
		r.CapabilityClasses = []string{model.CapabilitySemanticReview, model.CapabilityEvidenceOutputCapture}
		r.PermissionClasses = []string{model.PermissionReadOnly, model.PermissionNoExternalEffect}
	}
	return r
}

func latest(records []hoststate.Record, match func(hoststate.Record) bool) (string, error) {
	for i := len(records) - 1; i >= 0; i-- {
		if match(records[i]) {
			return records[i].HeadAnchor, nil
		}
	}
	return "", errors.New("required prior host record not found")
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// markerContent names the store's canonical path and run, so a marker copied
// into or linked from another directory does not match.
func markerContent(canonicalDir, runID string) string {
	return fmt.Sprintf("synthetic HMA dogfood store; contains no human approvals\nstore=%s\nrun_id=%s\n", canonicalDir, runID)
}

// scratchDir resolves dir and requires it to lie under the system temporary
// directory, the only place the harness creates stores.
func scratchDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	tmp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(tmp, canonical); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("refusing: %s is not under the temporary directory %s", canonical, tmp)
	}
	return canonical, nil
}

// requireSyntheticStore accepts only a scratch store seed-core created for this
// run: a regular (not symlinked) marker naming this canonical store and run.
func requireSyntheticStore(dir, runID string) error {
	canonical, err := scratchDir(dir)
	if err != nil {
		return err
	}
	path := filepath.Join(canonical, markerFile)
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return fmt.Errorf("refusing: %s is not a synthetic dogfood store (no regular %s)", dir, markerFile)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != markerContent(canonical, runID) {
		return fmt.Errorf("refusing: %s marker does not name this store and run", dir)
	}
	return nil
}

// seedCore creates a fresh scratch store holding a core run with one pending
// criterion and no approvals, so host commands have a core head to bind.
func seedCore(dir, runID string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("refusing: %s is not empty", dir)
	}
	canonical, err := scratchDir(dir)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(canonical, markerFile), []byte(markerContent(canonical, runID)), 0o600); err != nil {
		return err
	}
	rec := seedRecord(runID)
	return store.New(dir).Append(&rec)
}

// seedRecord is the only record seed-core writes.
func seedRecord(runID string) model.Record {
	return model.Record{
		Kind: model.KindCriterion, Version: 1,
		Metadata: model.RecordMetadata{RunID: runID, Sequence: 1, Timestamp: stamp(-60), Actor: syntheticActor},
		Criteria: []model.Criterion{{ID: "SYNTHETIC-C1", Disposition: model.CriterionPending}},
	}
}

// isSeedRecord reports whether r is byte-for-byte the record seed-core
// writes, apart from the store-computed head anchor.
func isSeedRecord(r model.Record, runID string) bool {
	r.Metadata.HeadAnchor = ""
	got, err1 := json.Marshal(r)
	want, err2 := json.Marshal(seedRecord(runID))
	return err1 == nil && err2 == nil && string(got) == string(want)
}

// seedSyntheticRouteApproval appends a labelled fixture standing in for the
// approved ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION record that a real
// human would produce. It is the harness's only approval-shaped record and is
// never presented as human-originated.
func seedSyntheticRouteApproval(dir, runID, unitID, digest string) error {
	if err := requireSyntheticStore(dir, runID); err != nil {
		return err
	}
	s := store.New(dir)
	prior, err := s.Load(runID)
	if err != nil {
		return err
	}
	// The run must be exactly what seed-core wrote: one synthetic criterion
	// record and no approval of any kind.
	if len(prior) != 1 || !isSeedRecord(prior[0], runID) {
		return errors.New("refusing: core run is not the untouched seed-core run")
	}
	head := prior[len(prior)-1].Metadata
	a := &model.ApprovalBinding{
		UnitID: unitID, RunID: runID, TransitionDigest: digest,
		CurrentStage: model.StageRouteSelection, ProposedTargetStage: model.StageImplementationAuthorization,
		RepositoryIdentityDigest: "synthetic-repo", BaseRevisionDigest: "synthetic-base",
		StageTimeDigest: fmt.Sprintf("%s:%d", head.HeadAnchor, head.Sequence+1), AcceptedPlanDigest: "synthetic-plan",
		ChallengeNonce: "synthetic-dogfood-nonce", Approver: syntheticApprover, Timestamp: stamp(-30),
	}
	rec := model.Record{
		UnitID: unitID, Kind: model.KindStageTransition, Version: 1,
		Metadata: model.RecordMetadata{RunID: runID, Sequence: head.Sequence + 1, PredecessorHash: head.HeadAnchor,
			Timestamp: stamp(-30), Actor: syntheticActor},
		Stage: model.StageRouteSelection, Control: model.ControlApproved, Approval: a,
	}
	return s.Append(&rec)
}

// emit writes one host or transition input. Digests of prior host records are
// read from the store so the script never hand-copies them.
func emit(step, dir, runID, unitID, digest, out string) error {
	history, err := hoststate.NewStore(dir).Load(runID)
	if err != nil {
		return err
	}
	env := func(minute int) envelope {
		return envelope{RunID: runID, Actor: syntheticHost, Timestamp: stamp(minute)}
	}
	impl, rev := route(false, digest), route(true, digest)
	isKind := func(kind hoststate.RecordKind, role hoststate.DispatchRole) func(hoststate.Record) bool {
		return func(r hoststate.Record) bool {
			switch kind {
			case hoststate.KindPreflight:
				return r.Preflight != nil && r.Preflight.UnitID == unitID && r.Preflight.Role == role
			case hoststate.KindDispatch:
				return r.Dispatch != nil && r.Dispatch.UnitID == unitID && r.Dispatch.Role == role
			}
			return false
		}
	}

	switch step {
	case "decision":
		return writeJSON(out, decisionInput{env(0), hoststate.OrchestrationDecision{
			UnitID: unitID, Mode: hoststate.DecisionDelegate, BoundedJob: "synthetic bounded dogfood job",
			EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
			ComparisonBasis: "synthetic: setup plus execution plus review coordination", Rationale: "synthetic dogfood delegation",
			ApprovedPermissionClasses: impl.PermissionClasses, WorkContextDigest: "synthetic-ctx-orchestrator", Route: impl,
		}})
	case "preflight-impl":
		return writeJSON(out, preflightInput{env(1), hoststate.RoutePreflight{
			UnitID: unitID, Role: hoststate.RoleImplementation, Route: impl, Status: hoststate.PreflightAvailable,
			ResponseDigest: "synthetic-hi-impl", ExpiresAt: stamp(6),
		}})
	case "dispatch-impl":
		pf, err := latest(history, isKind(hoststate.KindPreflight, hoststate.RoleImplementation))
		if err != nil {
			return err
		}
		return writeJSON(out, dispatchInput{env(2), hoststate.Dispatch{
			UnitID: unitID, Role: hoststate.RoleImplementation, Route: impl, PreflightRecordDigest: pf,
			BoundedTask: "synthetic bounded dogfood job", WorkContextDigest: "synthetic-ctx-impl",
		}})
	case "preflight-review":
		return writeJSON(out, preflightInput{env(3), hoststate.RoutePreflight{
			UnitID: unitID, Role: hoststate.RoleReview, Route: rev, Status: hoststate.PreflightAvailable,
			ResponseDigest: "synthetic-hi-review", ExpiresAt: stamp(8),
		}})
	case "dispatch-review":
		pf, err := latest(history, isKind(hoststate.KindPreflight, hoststate.RoleReview))
		if err != nil {
			return err
		}
		implementation, err := hoststate.RequireImplementation(history, unitID)
		if err != nil {
			return err
		}
		return writeJSON(out, dispatchInput{env(4), hoststate.Dispatch{
			UnitID: unitID, Role: hoststate.RoleReview, Route: rev, PreflightRecordDigest: pf,
			BoundedTask: "synthetic fixed-packet review", WorkContextDigest: "synthetic-ctx-review",
			ImplementationRecordDigest: implementation, ArtifactDigest: hoststate.ReviewArtifactDigest(syntheticHead, syntheticDiff, ""),
			ReviewPacketDigest: "synthetic-review-packet", Risk: model.RouteRiskHigh,
		}})
	case "review":
		dispatch, err := latest(history, isKind(hoststate.KindDispatch, hoststate.RoleReview))
		if err != nil {
			return err
		}
		implementation, err := hoststate.RequireImplementation(history, unitID)
		if err != nil {
			return err
		}
		return writeJSON(out, reviewInput{env(5), hoststate.IndependentReview{
			UnitID: unitID, ImplementationRecordDigest: implementation, ReviewDispatchDigest: dispatch,
			ArtifactDigest: hoststate.ReviewArtifactDigest(syntheticHead, syntheticDiff, ""), ReviewPacketDigest: "synthetic-review-packet",
			ReviewerID: rev.WorkerID, Verdict: hoststate.ReviewApprove,
		}})
	case "transition-impl-review", "transition-review-verification":
		from, to := model.StageImplementationAuthorization, model.StageImplementationReview
		if step == "transition-review-verification" {
			from, to = model.StageImplementationReview, model.StageVerification
		}
		// Deliberately carries no approval and no repository binding: the
		// host gate is evaluated first, then core must refuse to record.
		a := model.ApprovalBinding{UnitID: unitID, RunID: runID, CurrentStage: from, ProposedTargetStage: to}
		if to == model.StageVerification {
			a.ProducedHeadDigest, a.DiffDigest = syntheticHead, syntheticDiff
		}
		return writeJSON(out, transition.Input{RunID: runID, ExpectedApproval: a})
	default:
		return fmt.Errorf("unknown emit step %q", step)
	}
}

const usage = `usage:
  hostinput seed-core --store <dir> --run_id <id>
  hostinput seed-synthetic-route-approval --store <dir> --run_id <id> --unit_id <id> --digest <d>
  hostinput emit <step> --store <dir> --run_id <id> --unit_id <id> --digest <d> --out <file>
steps: decision preflight-impl dispatch-impl preflight-review dispatch-review review
       transition-impl-review transition-review-verification`

func run(args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cmd, rest := args[0], args[1:]
	step := ""
	if cmd == "emit" {
		if len(rest) == 0 {
			return errors.New(usage)
		}
		step, rest = rest[0], rest[1:]
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	dir := fs.String("store", "", "synthetic HMA store directory")
	runID := fs.String("run_id", "", "run id")
	unitID := fs.String("unit_id", "", "plan unit id")
	digest := fs.String("digest", "", "route approval digest to claim")
	out := fs.String("out", "", "output file")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *dir == "" || *runID == "" || fs.NArg() != 0 {
		return errors.New(usage)
	}
	switch cmd {
	case "seed-core":
		return seedCore(*dir, *runID)
	case "seed-synthetic-route-approval":
		if *unitID == "" || *digest == "" {
			return errors.New(usage)
		}
		return seedSyntheticRouteApproval(*dir, *runID, *unitID, *digest)
	case "emit":
		if *unitID == "" || *digest == "" || *out == "" {
			return errors.New(usage)
		}
		if err := requireSyntheticStore(*dir, *runID); err != nil {
			return err
		}
		return emit(step, *dir, *runID, *unitID, *digest, *out)
	default:
		return errors.New(usage)
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
