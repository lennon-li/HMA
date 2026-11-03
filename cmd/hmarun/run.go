// hmarun run drives one HMA run from GROUNDING to a terminal PARTIAL outcome.
//
// It never reimplements internal/engine or internal/transition logic: every
// state-changing step shells out to the real, unmodified hma binary with a
// freshly built --input JSON file, exactly as a human host would invoke it.
// internal/host, internal/model, internal/store, internal/repostate, and
// internal/engine are imported read-only (and, for one documented bootstrap
// write with no CLI equivalent — see ensureCriteriaBaseline — a single
// store append) to decide what remains to be done and to assemble correct
// input, never to bypass a gate the CLI itself enforces.
//
// By default this drives real agent-dispatch calls for the route preflight,
// the implementation worker, and the review: see issuePreflight,
// stepWorkerDispatch, and stepReviewerDispatch. --stub-dispatch switches all
// three back to the original local stand-ins, clearly labeled at each site:
//   - STUB POINT 1 (issuePreflight, stub branch): preflights always report
//     AVAILABLE with a placeholder response digest, never a real
//     agent-dispatch "hi".
//   - STUB POINT 2 (stepWorkerStub, stepReviewerStub): "the worker runs" and
//     "the reviewer runs" are each a trivial, labeled local change (a git
//     commit; a hand-written AUDIT.json and receipt.json) standing in for a
//     real agent-dispatch run.
//
// --stub-dispatch exists so the fast, deterministic tests that only need the
// surrounding stage-transition plumbing (TestRunReachesPartial and friends)
// never have to shell out to a real or fake agent-dispatch binary at all.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

const runUsage = "usage: hmarun run --store <dir> --repo <dir> --run_id <id> --contract <file> --plan <file> [--unit <id>] [--hma-bin <path>] [--auto-approve]"

// autoApproveApprover is the fixed, clearly-labeled synthetic approver
// `--auto-approve` substitutes for every human checkpoint. It is never used
// unless that flag is explicitly passed.
const autoApproveApprover = "AUTO-APPROVE-TEST-MODE (not a human approval)"

// Stub labels (see package doc). Every stubbed value in this file carries
// one of these two constants or an adjacent "STUB" comment.
const (
	stubPreflightResponseDigest = "stub-hi-not-a-real-preflight"
	stubWorkerCommitMessage     = "hmarun STUB worker change (not a real agent-dispatch run)"
)

// stubReviewMarker is the unmistakable, human-readable label every
// host record and approval summary carries while stepReviewerStub's
// synthetic AUDIT.json stands in for a real independent review: naming
// strings alone (hmarun-stub-reviewer, stub-model-b, ...) mark the stub in
// the chain, but nothing forced a human approving verification,
// independent-validation, credit, or PARTIAL to notice that the verdict
// they are acting on rests on no real evidence. This closes that gap
// without touching the review schema itself: the marker rides in
// envelope.Actor (host.Record.Actor, free text, always shown verbatim by
// `hma host show`) and as a literal prefix on approval summaries, never in
// a field review-import or report_contract constrain to a fixed value
// (verdict, isolation) or silently drop (audit_id, limitations).
const stubReviewMarker = "STUB REVIEW (no real evidence)"

// placeholderPlanDigest stands in for accepted_plan_digest before a plan
// exists to hash (steps that target ACCEPTANCE_CRITERIA or PLANNING). No
// later check reads it; engine.IssuesLease only reads the digest on the
// ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION approval (stepRouteApproval),
// which always carries the real plan digest.
var placeholderPlanDigest = sha256Hex("hmarun/v1/no-plan-yet")

// stageOrder totally orders the run's stages, so "is this step already
// done" is "has the chain's projected stage already passed it".
var stageOrder = map[model.Stage]int{
	model.StageGrounding:                   0,
	model.StageAcceptanceCriteria:          1,
	model.StagePlanning:                    2,
	model.StageRouteSelection:              3,
	model.StageImplementationAuthorization: 4,
	model.StageImplementationReview:        5,
	model.StageVerification:                6,
	model.StageIndependentValidation:       7,
	model.StageReleaseAndClosure:           8,
}

// Local mirrors of cmd/hma's unexported host-input envelope types. cmd/hma
// is package main and not importable; these reproduce exactly the JSON
// shape its strict decoder (json.Decoder.DisallowUnknownFields) accepts,
// the same technique scripts/dogfood/hostinput/main.go already uses.
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

type reviewPacketInput struct {
	envelope
	Dispatch              hoststate.Dispatch      `json:"dispatch"`
	DispatchRunID         string                  `json:"dispatch_run_id"`
	IssuingReason         hoststate.IssuingReason `json:"issuing_reason"`
	RepositoryRoot        string                  `json:"repository_root"`
	BaseRevision          string                  `json:"base_revision"`
	DirtyWorktreeApproved bool                    `json:"dirty_worktree_approved"`
	Task                  string                  `json:"task"`
	ContractPath          string                  `json:"contract_path,omitempty"`
	PolicyPath            string                  `json:"policy_path,omitempty"`
	PlanPath              string                  `json:"plan_path,omitempty"`
	LedgerDir             string                  `json:"ledger_dir"`
}

type reviewImportInput struct {
	envelope
	ReviewDispatchID string `json:"review_dispatch_id"`
	LedgerDir        string `json:"ledger_dir"`
	ReportPath       string `json:"report_path"`
}

type findingsImportInput struct {
	envelope
	ReviewRecordDigest string `json:"review_record_digest"`
	ReportPath         string `json:"report_path"`
}

type reconcileInput struct {
	envelope
	PolicyPath string `json:"policy_path,omitempty"`
}

type creditInput struct {
	envelope
	ReviewRecord string `json:"review_record"`
	ReportPath   string `json:"report_path"`
	ContractPath string `json:"contract_path"`
}

type bindRepositoryInput struct {
	envelope
	RepositoryRoot     string `json:"repository_root"`
	RepositoryIdentity string `json:"repository_identity"`
}

// runConfig is the parsed command line.
type runConfig struct {
	storeDir, repoRoot, runID, unit, contractPath, planPath, hmaBin string
	// agentDispatchBin is the path to the agent-dispatch launcher used for
	// every real preflight, worker, and review dispatch (see
	// dispatchConfigFor). Ignored when stubDispatch is set.
	agentDispatchBin string
	// stubDispatch switches issuePreflight, the worker step, and the
	// reviewer step back to the original local stand-ins (STUB POINT 1/2 in
	// the package doc), with no process spawned and no agentDispatchBin
	// needed. It exists so the fast existing tests keep running with no
	// agent-dispatch binary, real or fake, on hand.
	stubDispatch bool
	autoApprove  bool
}

// runner holds everything that is fixed for the lifetime of one `hmarun
// run` invocation: parsed inputs and the two fixed, independent synthetic
// routes. Everything that can change while the run proceeds (stage,
// criteria, route approval digest, packet bindings) is re-read live from
// the store on every step, never cached here, so a resumed process picks up
// exactly where the chain actually is.
type runner struct {
	cfg                runConfig
	p                  *prompter
	stdout             io.Writer
	contract           model.AcceptanceContract
	contractDigest     string
	contractRaw        []byte
	plan               model.Plan
	planDigest         string
	repositoryIdentity string
	base               string
	impl, reviewer     hoststate.Route
	policyPath         string
	artifactsDir       string
	// summary is the friction summary's counters (see summary.go); it
	// accumulates across the whole run() call and is printed once at the end.
	summary runSummary
}

func runRun(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stdout)
	storeDir := fs.String("store", "", "HMA store directory")
	repoRoot := fs.String("repo", "", "repository root hmarun drives the run against")
	runID := fs.String("run_id", "", "run id")
	unit := fs.String("unit", "", "plan unit id (required when the plan has more than one unit)")
	contractPath := fs.String("contract", "", "hma.acceptance-contract/v1 document, as written by `hmarun init`")
	planPath := fs.String("plan", "", "hma.plan/v1 document, as written by `hmarun init`")
	hmaBin := fs.String("hma-bin", "", "path to the compiled hma binary (default: look up \"hma\" on PATH)")
	agentDispatchBin := fs.String("agent-dispatch-bin", "", "path to the agent-dispatch launcher used for real preflight/worker/review dispatch (default: look up \"agent-dispatch\" on PATH; ignored with --stub-dispatch)")
	stubDispatch := fs.Bool("stub-dispatch", false, "use the local stub preflight/worker/reviewer instead of a real agent-dispatch call (fast, deterministic, no process spawned; real dispatch is the default)")
	autoApprove := fs.Bool("auto-approve", false, "answer every human checkpoint with a fixed synthetic approver (test only; never the default)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storeDir == "" || *repoRoot == "" || *runID == "" || *contractPath == "" || *planPath == "" || fs.NArg() != 0 {
		return errors.New(runUsage)
	}

	cfg := runConfig{
		storeDir: *storeDir, repoRoot: *repoRoot, runID: *runID, unit: *unit,
		contractPath: *contractPath, planPath: *planPath, hmaBin: *hmaBin,
		agentDispatchBin: *agentDispatchBin, stubDispatch: *stubDispatch, autoApprove: *autoApprove,
	}
	r, err := newRunner(cfg, stdin, stdout)
	if err != nil {
		return err
	}
	return r.run(context.Background())
}

func newRunner(cfg runConfig, stdin io.Reader, stdout io.Writer) (*runner, error) {
	if cfg.hmaBin == "" {
		if path, err := exec.LookPath("hma"); err == nil {
			cfg.hmaBin = path
		} else {
			return nil, errors.New("hmarun run: no --hma-bin given and \"hma\" is not on PATH; build it first (go build -o <path> ./cmd/hma)")
		}
	}
	// This PATH lookup only runs for a real `hmarun run` invocation: every
	// test in this package either sets stubDispatch or supplies its own fake
	// agentDispatchBin explicitly, so go test never resolves or executes the
	// real agent-dispatch on this machine.
	if !cfg.stubDispatch && cfg.agentDispatchBin == "" {
		if path, err := exec.LookPath("agent-dispatch"); err == nil {
			cfg.agentDispatchBin = path
		} else {
			return nil, errors.New("hmarun run: no --agent-dispatch-bin given and \"agent-dispatch\" is not on PATH; install it, or pass --stub-dispatch for the local stand-in")
		}
	}
	contractRaw, err := os.ReadFile(cfg.contractPath)
	if err != nil {
		return nil, err
	}
	contract, contractDigest, err := model.ParseAcceptanceContract(contractRaw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cfg.contractPath, err)
	}
	planRaw, err := os.ReadFile(cfg.planPath)
	if err != nil {
		return nil, err
	}
	plan, planDigest, err := model.ParsePlan(planRaw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cfg.planPath, err)
	}
	if err := model.ValidatePlanAgainstContract(plan, contract, contractDigest); err != nil {
		return nil, err
	}
	if cfg.unit == "" {
		if len(plan.Units) != 1 {
			return nil, errors.New("hmarun run: --unit is required when the plan has more than one unit")
		}
		cfg.unit = plan.Units[0].ID
	}
	var unitFound bool
	for _, u := range plan.Units {
		if u.ID == cfg.unit {
			unitFound = true
		}
	}
	if !unitFound {
		return nil, fmt.Errorf("hmarun run: unit %q is not in the plan", cfg.unit)
	}
	repoRoot, err := filepath.Abs(cfg.repoRoot)
	if err != nil {
		return nil, err
	}
	cfg.repoRoot = repoRoot
	storeDir, err := filepath.Abs(cfg.storeDir)
	if err != nil {
		return nil, err
	}
	cfg.storeDir = storeDir
	artifactsDir := filepath.Join(cfg.storeDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		return nil, err
	}

	r := &runner{
		cfg: cfg, p: newPrompter(stdin, stdout), stdout: stdout,
		contract: contract, contractDigest: contractDigest, contractRaw: contractRaw,
		plan: plan, planDigest: planDigest, artifactsDir: artifactsDir,
	}
	if cfg.stubDispatch {
		r.impl = hmarunRoute("hmarun-stub-implementer", "stub-provider-a", "stub-family-a", "stub-model-a", "stub-model-family-a", "medium", false)
		r.reviewer = hmarunRoute("hmarun-stub-reviewer", "stub-provider-b", "stub-family-b", "stub-model-b", "stub-model-family-b", "medium", true)
	} else {
		implCfg, reviewCfg := dispatchConfigFor[hoststate.RoleImplementation], dispatchConfigFor[hoststate.RoleReview]
		r.impl = hmarunRoute(implCfg.Agent, implCfg.Provider, implCfg.ProviderFamily, implCfg.Model, implCfg.ModelFamily, implCfg.Effort, false)
		r.reviewer = hmarunRoute(reviewCfg.Agent, reviewCfg.Provider, reviewCfg.ProviderFamily, reviewCfg.Model, reviewCfg.ModelFamily, reviewCfg.Effort, true)
	}

	// base_revision is fixed for the lifetime of the run, across process
	// restarts, not just this one. A genuinely new run (empty core chain)
	// takes it from live HEAD. A resumed run (chain already has records)
	// must keep using the original base even though HEAD may have since
	// moved (stepWorkerStub commits to repoRoot), so it is read back from
	// the chain's first record instead: that record is always the
	// GROUNDING -> ACCEPTANCE_CRITERIA transition (stepAcceptanceCriteria
	// runs first and ensureCriteriaBaseline's bootstrap record always comes
	// after it), and doTransition stamps every approval's
	// BaseRevisionDigest with r.base at the time it was approved.
	core, err := store.New(cfg.storeDir).Load(cfg.runID)
	if err != nil {
		return nil, err
	}
	if len(core) > 0 {
		first := core[0]
		if first.Approval == nil || first.Approval.BaseRevisionDigest == "" {
			return nil, fmt.Errorf("hmarun run: run %q's first core record carries no base_revision_digest", cfg.runID)
		}
		r.base = first.Approval.BaseRevisionDigest
	} else {
		baseOut, err := repostate.Output(context.Background(), cfg.repoRoot, "rev-parse", "HEAD")
		if err != nil {
			return nil, err
		}
		r.base = repostate.Trim(baseOut)
	}

	if err := r.resolveRepositoryIdentity(); err != nil {
		return nil, err
	}
	if err := r.writePolicy(); err != nil {
		return nil, err
	}
	return r, nil
}

// hmarunRoute builds one hoststate.Route with the capability/permission set
// appropriate to its role. RouteApprovalDigest is filled in at each call
// site from the unit's live route approval (see routeApprovalDigest), never
// cached on the runner: it does not exist until stepRouteApproval runs.
func hmarunRoute(worker, provider, providerFamily, modelName, modelFamily, effort string, review bool) hoststate.Route {
	caps := []string{model.CapabilityBoundedImplementation, model.CapabilityEvidenceOutputCapture}
	perms := []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	if review {
		caps = []string{model.CapabilitySemanticReview, model.CapabilityEvidenceOutputCapture}
		perms = []string{model.PermissionReadOnly, model.PermissionNoExternalEffect}
	}
	return hoststate.Route{
		WorkerID: worker, Provider: provider, ProviderFamily: providerFamily, ProviderDigest: "digest-" + provider,
		Model: modelName, ModelFamily: modelFamily, ModelDigest: "digest-" + modelName, ProfileDigest: "profile-" + worker,
		AccessService: "service-" + provider, AccessServiceDigest: "digest-service-" + provider,
		Runtime: "runtime-" + worker, ReasoningEffort: effort, ExecutionContext: model.ExecutionContextProxy,
		CapabilityClasses: caps, PermissionClasses: perms,
	}
}

// dispatchAgentConfig is the real agent-dispatch identity hmarun uses for one
// dispatch role: which of agent-dispatch's own agents it is (ming/phil/...),
// which --model and --effort it launches with, and the provider/family
// labels the resulting receipt will actually carry.
//
// Provider is not a free choice: agent-run-ledger's write_receipt derives
// it from --model and the agent name (derive_provider in
// scripts/agent-run-ledger), falling back to a fixed per-agent map
// ("ming"->"anthropic", "phil"->"openai", ...) when --model matches no
// known prefix. hoststate.Route.Provider must equal that derived value
// exactly, or EvaluateReviewImport's ReviewLaunchMatches check refuses a
// perfectly real, successful review (see the dispatch report). "claude" as
// --model also independently matches derive_provider's "claude" prefix
// rule, so ming's provider is "anthropic" either way; phil's "openai" here
// is agent-dispatch's own historical agent->provider fallback label, not a
// claim about what model actually ran phil.
type dispatchAgentConfig struct {
	Agent, Model, Provider, ProviderFamily, ModelFamily, Effort string
}

// dispatchConfigFor is hmarun's one named mapping from dispatch role to the
// real agent-dispatch identity it dispatches: IMPLEMENTATION -> ming,
// REVIEW -> phil, the same two agents used for every Ming/Phil dispatch this
// session (see the dispatch report). Changing who implements or reviews
// later means changing this map; nothing else in this file names an agent,
// model, provider, or effort as a literal.
// Model, Provider and ProviderFamily were all wrong here before a live
// smoke test against the real agent-dispatch tool caught it (see the
// dispatch report): "claude" and "copilot" were never valid --model values
// ("unrecognized_model" / "is not available" against the real tool).
//
// Fixing Model alone is not enough: the first real value tried for phil
// that agent-dispatch accepted, claude-sonnet-5.5, made phil's real receipt
// report provider "anthropic" -- the same as ming's. That broke a
// different kernel check, EvaluateDispatch's reviewer-independence rule
// (internal/host/host.go, ReasonReviewerNotIndependent): for a "strong"
// review (risk high/critical, disputed, final validation, or -- always,
// here -- IssuingReason == IssueFinal), the reviewer's ProviderFamily and
// ModelFamily must differ from every implementer's. Labeling phil as
// "openai"/"copilot-family" while its receipt actually says "anthropic"
// would satisfy that independence check on paper while failing
// ReviewLaunchMatches's receipt-equality check for real (it requires
// Route.Provider to equal the receipt's provider exactly) -- i.e. every
// real review-import would fail. The actual fix is a genuinely different
// model: gpt-5.3-codex is a real --model value agent-dispatch accepts for
// phil, and its receipt reports provider "openai" for real, so it
// satisfies both checks honestly instead of trading one label-only
// mismatch for another.
//
// `hma host decision` also rejects an empty Model outright ("route missing
// model"), so these must be real, confirmed-working, non-empty values.
var dispatchConfigFor = map[hoststate.DispatchRole]dispatchAgentConfig{
	hoststate.RoleImplementation: {Agent: "ming", Model: "claude-sonnet-5", Provider: "anthropic", ProviderFamily: "anthropic", ModelFamily: "claude-family", Effort: "medium"},
	hoststate.RoleReview:         {Agent: "phil", Model: "gpt-5.3-codex", Provider: "openai", ProviderFamily: "openai", ModelFamily: "gpt-family", Effort: "medium"},
}

// withModelFlag appends "--model", model to args only when model is
// non-empty. dispatchConfigFor above always sets a real model now, but a
// future route config might reasonably want "let the agent's own runtime
// default decide" (empty), and passing agent-dispatch an empty --model
// value would be its own new way to fail, so this keeps that case safe.
func withModelFlag(args []string, model string) []string {
	if model == "" {
		return args
	}
	return append(args, "--model", model)
}

// resolveRepositoryIdentity confirms (H) the one repository identity string
// this run binds forever, starting at its first approval.
func (r *runner) resolveRepositoryIdentity() error {
	def := r.cfg.repoRoot
	if r.cfg.autoApprove {
		r.repositoryIdentity = def
		return nil
	}
	r.p.line("\nRun %q will bind repository identity for %s.\n", r.cfg.runID, r.cfg.repoRoot)
	id, err := r.p.ask("Repository identity to bind (this cannot change later)", "", def)
	if err != nil {
		return err
	}
	r.repositoryIdentity = id
	return nil
}

// writePolicy writes the fixed, empty route policy document hmarun uses for
// every host command that names a --policy/policy_path: a real deployment's
// route policy is a host concern outside hmarun's scope; this is the same
// "no real policy document yet" placeholder hmarun init uses for a plan's
// policy_digest.
func (r *runner) writePolicy() error {
	r.policyPath = filepath.Join(r.cfg.storeDir, "policy.json")
	policy := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
	b, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.policyPath, append(b, '\n'), 0o644)
}

// runStep is one named step of the run's fixed sequence; see runSteps.
type runStep struct {
	name string
	fn   func(context.Context) error
}

// runSteps is the run's one ordered step sequence, from GROUNDING to the
// PARTIAL terminal outcome. r.run and the resume tests (run_test.go) both
// build their step list from this single slice, so a reordering here can
// never silently desync what the tests exercise from what a real `hmarun
// run` executes. Each name matches the crash points named in
// docs/handoffs/evidence/mvp/audit-hmarun-step1-2-phil.md's F1 finding.
func (r *runner) runSteps() []runStep {
	// The step names stay "worker-stub"/"reviewer-stub" regardless of mode:
	// every test that resumes or stops at one of these names (run_test.go)
	// identifies a position in the fixed sequence, not a claim about which
	// function runs there. With --stub-dispatch they are literally the
	// stubs; otherwise they are stepWorkerDispatch/stepReviewerDispatch, the
	// real agent-dispatch calls.
	worker, reviewer := r.stepWorkerStub, r.stepReviewerStub
	if !r.cfg.stubDispatch {
		worker, reviewer = r.stepWorkerDispatch, r.stepReviewerDispatch
	}
	return []runStep{
		{"accept-criteria", r.stepAcceptanceCriteria},
		{"planning", r.stepPlanning},
		{"route-selection", r.stepRouteSelection},
		{"route-approval", r.stepRouteApproval},
		{"decision", r.stepDecision},
		{"impl-preflight", r.stepImplPreflight},
		{"impl-dispatch", r.stepImplDispatch},
		{"worker-stub", worker},
		{"implementation-review", r.stepImplementationReview},
		{"bind-repository", r.stepBindRepository},
		{"review-preflight", r.stepReviewPreflight},
		{"review-packet", r.stepReviewPacket},
		{"reviewer-stub", reviewer},
		{"review-import", r.stepReviewImport},
		{"findings-import", r.stepFindingsImport},
		{"reconcile", r.stepReconcile},
		{"verification", r.stepVerification},
		{"independent-validation", r.stepIndependentValidation},
		{"credit", r.stepCredit},
		{"partial", r.stepPartial},
	}
}

func (r *runner) run(ctx context.Context) error {
	for _, s := range r.runSteps() {
		if err := s.fn(ctx); err != nil {
			r.summary.outcome = fmt.Sprintf("blocked at step %s: %v", s.name, err)
			r.printSummary()
			return fmt.Errorf("hmarun run: step %s: %w", s.name, err)
		}
	}
	r.summary.outcome = string(model.OutcomePartial)
	r.printSummary()
	fmt.Fprintf(r.stdout, "hmarun run: reached terminal PARTIAL for run %q, unit %q.\n", r.cfg.runID, r.cfg.unit)
	return nil
}

// --- live state helpers -----------------------------------------------

func (r *runner) coreRecords() ([]model.Record, error) {
	return store.New(r.cfg.storeDir).Load(r.cfg.runID)
}

func (r *runner) hostHistory() ([]hoststate.Record, error) {
	return hoststate.NewStore(r.cfg.storeDir).Load(r.cfg.runID)
}

func (r *runner) envelope() envelope {
	return envelope{RunID: r.cfg.runID, Actor: "hmarun", Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// usingStubReview is true for exactly as long as the run uses
// stepReviewerStub's synthetic review (--stub-dispatch), so
// stubReviewEnvelope and stubPrefixed stop marking anything, with no
// further edit here, once real dispatch is in use.
func (r *runner) usingStubReview() bool {
	return r.cfg.stubDispatch
}

// stubReviewEnvelope is envelope() with stubReviewMarker appended to Actor
// when the independent review this host record carries or depends on is
// still stepReviewerStub's synthetic one. Every host command whose record
// is part of that chain (the review dispatch, review-import, findings
// import, and credit) uses this instead of envelope(), so `hma host show`'s
// verbatim dump of that record carries the marker, not just the stub route
// names.
func (r *runner) stubReviewEnvelope() envelope {
	e := r.envelope()
	if r.usingStubReview() {
		e.Actor = e.Actor + " " + stubReviewMarker
	}
	return e
}

// stubPrefixed prefixes summary with stubReviewMarker, while the stub
// review is in use, so a human typing an approval for a step whose verdict
// rests on it cannot miss that it is not real evidence.
func (r *runner) stubPrefixed(summary string) string {
	if r.usingStubReview() {
		return stubReviewMarker + " — " + summary
	}
	return summary
}

// routeApprovalDigest re-derives the unit's active route approval digest
// directly from the core chain's own approved ROUTE_SELECTION ->
// IMPLEMENTATION_AUTHORIZATION transition, mirroring cmd/hma's unexported
// approvedCoreRouteDigestIn. This is hmarun's answer to the "route approval
// digest persistence" question: read it back from the store, which is
// always authoritative and can never desync, rather than keep a second copy
// in a side-state file.
func (r *runner) routeApprovalDigest(core []model.Record) (string, error) {
	digest := ""
	for _, rec := range core {
		if rec.UnitID == r.cfg.unit && rec.Kind == model.KindStageTransition && rec.Control == model.ControlApproved &&
			rec.Approval != nil && rec.Approval.UnitID == r.cfg.unit &&
			rec.Approval.CurrentStage == model.StageRouteSelection && rec.Approval.ProposedTargetStage == model.StageImplementationAuthorization {
			digest = rec.Approval.TransitionDigest
		}
	}
	if digest == "" {
		return "", errors.New("route approval digest not found in core chain")
	}
	return digest, nil
}

func freshToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// confirm prints summary and requires one explicit typed confirmation,
// which becomes the transition's human approver. --auto-approve substitutes
// the fixed synthetic approver instead of prompting.
func (r *runner) confirm(summary string) (string, error) {
	r.summary.approvals++
	r.p.line("\n%s\n", summary)
	if r.cfg.autoApprove {
		r.p.line("--auto-approve: using synthetic approver %q\n", autoApproveApprover)
		return autoApproveApprover, nil
	}
	return r.p.askRequired("Type your name to approve", "")
}

func (r *runner) writeJSON(name string, v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(r.artifactsDir, name)
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// execHMA shells out to the real, unmodified hma binary and returns a clear
// error naming its combined output on a non-zero exit, never silently
// swallowing a refusal.
func (r *runner) execHMA(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, r.cfg.hmaBin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hma %v: %w: %s", args, err, out)
	}
	return nil
}

// --- step 1-4: core stage transitions through implementation authorization

func (r *runner) planDigestFor(to model.Stage) string {
	if to == model.StageAcceptanceCriteria || to == model.StagePlanning {
		return placeholderPlanDigest
	}
	return r.planDigest
}

// doTransition performs one host-approved stage or outcome transition if
// the chain has not already passed it. from/to/outcome, unit, and
// attachContractDoc describe the edge exactly as internal/transition and
// internal/engine require it; see cmd/hmarun's package doc and the dispatch
// report for the reasoning behind each field.
func (r *runner) doTransition(ctx context.Context, from, to model.Stage, outcome model.TerminalOutcome, unit string, attachContractDoc bool, summary string) error {
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	state := engine.ProjectStageState(core)
	if outcome != "" {
		if state.Terminal {
			return nil
		}
	} else if stageOrder[state.CurrentStage] >= stageOrder[to] || state.Terminal {
		return nil
	}
	if state.CurrentStage != from {
		return fmt.Errorf("unexpected current stage %s, want %s", state.CurrentStage, from)
	}

	approver, err := r.confirm(summary)
	if err != nil {
		return err
	}
	nonce, err := freshToken()
	if err != nil {
		return err
	}
	digest, err := freshToken()
	if err != nil {
		return err
	}

	a := model.ApprovalBinding{
		RunID: r.cfg.runID, UnitID: unit, TransitionDigest: digest,
		CurrentStage: from, ProposedTargetStage: to, ProposedTargetOutcome: outcome,
		RepositoryIdentityDigest: r.repositoryIdentity, BaseRevisionDigest: r.base,
		StageTimeDigest:    fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1),
		AcceptedPlanDigest: r.planDigestFor(to),
		ChallengeNonce:     nonce, Approver: approver, Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	// Every approval from the contract's acceptance onward carries its
	// digest (engine.EvaluateContractBinding); only the very first
	// transition, before any contract has been accepted, carries none.
	if from != model.StageGrounding {
		a.AcceptanceContractDigest = r.contractDigest
	}
	if model.ApprovalBindsHeadDiff(from) {
		snap, err := repostate.Snapshot(ctx, r.cfg.repoRoot, r.base, false)
		if err != nil {
			return err
		}
		a.ProducedHeadDigest, a.DiffDigest = snap.Head, snap.Diff
	}

	in := transition.Input{
		RunID: r.cfg.runID, RepositoryRoot: r.cfg.repoRoot, RepositoryIdentity: r.repositoryIdentity,
		BaseRevision: r.base, ExpectedApproval: a, Approval: a,
		ApprovalNow: time.Now().UTC(), ApprovalMaxAgeSeconds: 300, PolicyPath: r.policyPath,
	}
	if attachContractDoc {
		in.AcceptanceContract = r.contractRaw
	}
	path, err := r.writeJSON("transition-input.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "transition", "--input", path, "--store", r.cfg.storeDir)
}

func (r *runner) stepAcceptanceCriteria(ctx context.Context) error {
	return r.doTransition(ctx, model.StageGrounding, model.StageAcceptanceCriteria, "", "", false,
		fmt.Sprintf("Open run %q against %s at base revision %s.\nThis binds the repository identity %q forever for this run.",
			r.cfg.runID, r.cfg.repoRoot, r.base, r.repositoryIdentity))
}

// ensureCriteriaBaseline writes the run's one criterion baseline record
// (every contract criterion id, PENDING) if the chain does not have one yet.
//
// This is a documented gap in the real hma CLI, not a workaround: Apply
// (internal/transition/apply.go) routes a presented contract through
// engine.EvaluateContractBinding, which on ACCEPTANCE_CRITERIA -> PLANNING
// requires the contract's criterion ids to already equal the chain's latest
// model.KindCriterion snapshot (sameIDs against state.Criteria) -- it never
// establishes that snapshot itself. Across the whole non-test codebase the
// only production writer of a KindCriterion record is `hma host credit`
// (cmd/hma/credit.go), which cannot run this early (it requires an approved
// route, a dispatched implementation, and an imported review). The hma test
// suite's own fixtures close this same gap the same way: see
// internal/transition/contract_test.go's toAcceptance, which calls
// store.New(dir).Append on a bare model.Record{Kind: KindCriterion} right
// after the real Apply-backed GROUNDING -> ACCEPTANCE_CRITERIA transition,
// before ever presenting a contract. hmarun does the same thing here,
// against an otherwise-untouched store, and never writes anything else
// through internal/store directly.
func (r *runner) ensureCriteriaBaseline(core []model.Record, state engine.StageState) error {
	if len(state.Criteria) > 0 {
		return nil
	}
	if state.CurrentStage != model.StageAcceptanceCriteria || len(core) == 0 {
		return nil
	}
	ids := model.ContractCriterionIDs(r.contract)
	criteria := make([]model.Criterion, 0, len(ids))
	for _, id := range ids {
		criteria = append(criteria, model.Criterion{ID: id, Disposition: model.CriterionPending})
	}
	last := core[len(core)-1]
	rec := model.Record{
		Kind: model.KindCriterion, Version: 1, Stage: model.StageAcceptanceCriteria, Criteria: criteria,
		Metadata: model.RecordMetadata{
			RunID: r.cfg.runID, Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Actor: "hmarun:core-bootstrap",
		},
	}
	return store.New(r.cfg.storeDir).Append(&rec)
}

func (r *runner) stepPlanning(ctx context.Context) error {
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	state := engine.ProjectStageState(core)
	if stageOrder[state.CurrentStage] >= stageOrder[model.StagePlanning] {
		return nil
	}
	if err := r.ensureCriteriaBaseline(core, state); err != nil {
		return err
	}
	return r.doTransition(ctx, model.StageAcceptanceCriteria, model.StagePlanning, "", "", true,
		fmt.Sprintf("Accept the acceptance contract at %s (digest %s) for run %q.", r.cfg.contractPath, r.contractDigest, r.cfg.runID))
}

func (r *runner) stepRouteSelection(ctx context.Context) error {
	return r.doTransition(ctx, model.StagePlanning, model.StageRouteSelection, "", "", false,
		fmt.Sprintf("Accept the plan at %s (digest %s) for run %q.", r.cfg.planPath, r.planDigest, r.cfg.runID))
}

func (r *runner) stepRouteApproval(ctx context.Context) error {
	return r.doTransition(ctx, model.StageRouteSelection, model.StageImplementationAuthorization, "", r.cfg.unit, false,
		fmt.Sprintf("Approve the route for unit %q (implementer %s/%s).", r.cfg.unit, r.impl.Provider, r.impl.Model))
}

// --- step 5-8: delegate, dispatch, and (STUB POINT 2) "the worker runs" ---

func (r *runner) stepDecision(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	for _, rec := range history {
		if rec.Kind == hoststate.KindDecision && rec.Decision != nil && rec.Decision.UnitID == r.cfg.unit {
			return nil
		}
	}
	approver, err := r.confirm(fmt.Sprintf("Delegate unit %q's implementation to %s/%s (vs. direct cost).", r.cfg.unit, r.impl.Provider, r.impl.Model))
	if err != nil {
		return err
	}
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	routeDigest, err := r.routeApprovalDigest(core)
	if err != nil {
		return err
	}
	impl := r.impl
	impl.RouteApprovalDigest = routeDigest
	in := decisionInput{envelope: r.envelope(), Decision: hoststate.OrchestrationDecision{
		UnitID: r.cfg.unit, Mode: hoststate.DecisionDelegate, BoundedJob: "implement unit " + r.cfg.unit,
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative_units",
		ComparisonBasis:           "dispatch, execution, and review time vs. direct implementation time",
		Rationale:                 "human-approved delegation: " + approver,
		ApprovedPermissionClasses: impl.PermissionClasses,
		WorkContextDigest:         sha256Hex("hmarun/v1/work-context:impl:" + r.cfg.runID + ":" + r.cfg.unit),
		Route:                     impl,
	}}
	path, err := r.writeJSON("decision.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "decision", "--input", path, "--store", r.cfg.storeDir)
}

// findLatestValidPreflight returns the head anchor of the unit's latest
// AVAILABLE, not-yet-expired preflight of role, or ("", false, nil) when
// none qualifies (none exists, the latest is unavailable, or it expired).
func (r *runner) findLatestValidPreflight(role hoststate.DispatchRole) (string, bool, error) {
	history, err := r.hostHistory()
	if err != nil {
		return "", false, err
	}
	now := time.Now().UTC()
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		if rec.Kind == hoststate.KindPreflight && rec.Preflight != nil && rec.Preflight.UnitID == r.cfg.unit && rec.Preflight.Role == role {
			if rec.Preflight.Status != hoststate.PreflightAvailable {
				return "", false, nil
			}
			expires, err := time.Parse(time.RFC3339, rec.Preflight.ExpiresAt)
			if err != nil || !now.Before(expires) {
				return "", false, nil
			}
			return rec.HeadAnchor, true, nil
		}
	}
	return "", false, nil
}

// issuePreflight is STUB POINT 1 in --stub-dispatch mode: it always reports
// AVAILABLE with a fixed, clearly-labeled placeholder response digest, never
// a real agent-dispatch "hi". Otherwise it runs one real `agent-dispatch
// <agent> --prompt "hi"` (the agent named by dispatchConfigFor[role]) and
// classifies the real result (see classifyPreflightOutput).
//
// The host record is always written with the real observed status, never
// silently promoted to AVAILABLE: on anything other than a clean AVAILABLE
// this returns a descriptive error so the caller's step loop stops the run
// here (the "no dead-end agents" contract: name what happened, name the
// human's options, never just exit on an UNAVAILABLE/RATE_LIMITED/
// QUOTA_EXHAUSTED/INDETERMINATE status silently).
func (r *runner) issuePreflight(ctx context.Context, role hoststate.DispatchRole, route hoststate.Route) error {
	var status hoststate.PreflightStatus
	var digest string
	if r.cfg.stubDispatch {
		fmt.Fprintf(r.stdout, "STUB: not a real preflight response (role=%s)\n", role)
		status, digest = hoststate.PreflightAvailable, stubPreflightResponseDigest
	} else {
		cfg := dispatchConfigFor[role]
		runID, err := r.freshDispatchRunID(fmt.Sprintf("preflight-%s", role))
		if err != nil {
			return err
		}
		args := withModelFlag([]string{cfg.Agent}, cfg.Model)
		args = append(args, "--effort", cfg.Effort, "--run-id", runID, "--prompt", "hi")
		fmt.Fprintf(r.stdout, "Dispatching real agent-dispatch preflight: agent-dispatch %s\n", strings.Join(args, " "))
		out, exitCode, runErr := r.runAgentDispatch(ctx, args, r.unreconciledDispatchStateDir())
		if runErr != nil {
			r.summary.blockedUnrecoverable++
			return fmt.Errorf("agent-dispatch preflight for role %s: could not launch %s: %w", role, r.cfg.agentDispatchBin, runErr)
		}
		status, digest = classifyPreflightOutput(out, exitCode)
		fmt.Fprintf(r.stdout, "agent-dispatch preflight for role %s: status=%s exit=%d run-id=%s\n", role, status, exitCode, runID)
	}
	env := r.envelope()
	if role == hoststate.RoleReview {
		env = r.stubReviewEnvelope()
	}
	in := preflightInput{envelope: env, Preflight: hoststate.RoutePreflight{
		UnitID: r.cfg.unit, Role: role, Route: route, Status: status,
		ResponseDigest: digest, ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339),
	}}
	path, err := r.writeJSON(fmt.Sprintf("preflight-%s.json", role), in)
	if err != nil {
		return err
	}
	if err := r.execHMA(ctx, "host", "preflight", "--input", path, "--store", r.cfg.storeDir); err != nil {
		return err
	}
	if status != hoststate.PreflightAvailable {
		r.summary.blockedUnrecoverable++
		agent := dispatchConfigFor[role].Agent
		return fmt.Errorf(
			"agent-dispatch preflight for role %s came back %s (agent %s), not AVAILABLE; hmarun run stops here "+
				"rather than dispatching against a dead agent. The real status has been recorded in the host chain. "+
				"Options: (1) re-run `hmarun run` to retry this preflight once the underlying problem clears "+
				"(rate limit, missing credentials, a transient runtime error, ...); (2) if %s is persistently "+
				"unavailable for this role, escalate to a human route decision naming a different route for unit %q",
			role, status, agent, agent, r.cfg.unit)
	}
	return nil
}

// classifyPreflightOutput maps one real `agent-dispatch <agent> --prompt
// "hi"` invocation's combined stdout+stderr and exit code to a
// hoststate.PreflightStatus, and returns the sha256 hex digest of the exact
// output bytes observed (the preflight's response_digest).
//
// agent-dispatch's own AGENT_DISPATCH_PREFLIGHT=ok/failed line
// (scripts/agent-dispatch's preflight_runtime, emitted to stderr) only
// covers local runtime readiness: the right binary installed, the ledger
// writable, the account home present. It never talks to the provider. A
// "hi" preflight that passes that local check goes on to make one real,
// complete agent-dispatch run (that is what --prompt "hi" means); any
// provider-side rate-limit or quota signal only shows up afterward, in that
// real run's exit code and free-form output text -- agent-dispatch has no
// structured field for it. The keyword match below is therefore a
// best-effort heuristic over human-readable tool/runtime output, not a
// guaranteed contract; see the dispatch report.
// standalone429 matches a bare "429" HTTP-status token, not "429" appearing
// as a run of digits/hex inside a longer identifier such as a run id or a
// hex digest (audit-hmarun-step3-4-phil.md F7: a plain substring match
// could mislabel a failure as RATE_LIMITED off a hex token that happens to
// contain "429", even though that never changes the safety outcome --
// classifyPreflightOutput only ever downgrades from AVAILABLE here, never
// up -- it was still the wrong label and the wrong human-facing advice).
var standalone429 = regexp.MustCompile(`\b429\b`)

func classifyPreflightOutput(output string, exitCode int) (hoststate.PreflightStatus, string) {
	digest := sha256Hex(output)
	switch lastFieldValue(output, "AGENT_DISPATCH_PREFLIGHT") {
	case "ok":
		// Local runtime readiness passed; the real dispatch run itself
		// follows, and its own outcome (checked below) decides the status.
	case "failed":
		// A local readiness failure (missing binary, unwritable ledger,
		// incomplete account home, ...) is a real, present-tense
		// unavailability, not a guess.
		return hoststate.PreflightUnavailable, digest
	default:
		// No recognizable preflight line at all: the launcher itself did
		// not run as expected (wrong binary, usage error, ...). Unknown,
		// not a confident UNAVAILABLE.
		return hoststate.PreflightIndeterminate, digest
	}
	if exitCode == 0 && lastFieldValue(output, "AGENT_DISPATCH_STATE") == "completed" {
		return hoststate.PreflightAvailable, digest
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "rate-limit") || standalone429.MatchString(lower):
		return hoststate.PreflightRateLimited, digest
	case strings.Contains(lower, "quota"):
		return hoststate.PreflightQuotaExhausted, digest
	default:
		return hoststate.PreflightIndeterminate, digest
	}
}

// lastFieldValue returns the value of the last "KEY=value ..." token found
// on any line of output (agent-dispatch's own diagnostic lines are always
// "KEY=value" optionally followed by more space-separated "k=v" pairs on the
// same line, e.g. "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude"),
// or "" if key never appears. Scanning for the last occurrence (not the
// first) matters for keys agent-dispatch prints twice, once at launch and
// once at completion (AGENT_DISPATCH_RECEIPT, AGENT_DISPATCH_STATE).
func lastFieldValue(output, key string) string {
	prefix := key + "="
	value := ""
	for _, line := range strings.Split(output, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if !ok {
			continue
		}
		if fields := strings.Fields(rest); len(fields) > 0 {
			value = fields[0]
		} else {
			value = ""
		}
	}
	return value
}

// freshDispatchRunID pre-allocates one agent-dispatch --run-id: known before
// launch, so hmarun always knows exactly where a dispatch's receipt will
// land without having to parse it back out of captured output, and distinct
// per attempt (freshToken, not a fixed name) so a retried dispatch never
// collides with a failed attempt's already-created run directory --
// agent-run-ledger refuses to reuse an existing run id (`mkdir "$run_dir"`
// fails closed). label is a human-readable hint only (role/attempt), never
// parsed back.
func (r *runner) freshDispatchRunID(label string) (string, error) {
	token, err := freshToken()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("hmarun-%s-%s-%s-%s", r.cfg.runID, r.cfg.unit, label, token), nil
}

// runAgentDispatch runs one real agent-dispatch invocation with its working
// directory set to the repository hmarun is driving (agent-dispatch
// resolves its target project from the current directory's git toplevel,
// exactly as a human host running it by hand would) and
// AGENT_DISPATCH_STATE_DIR forced to stateDir, so every dispatch ledger this
// run produces is self-contained under --store and its root is computable
// in advance, never the operator's shared ~/.local/state/agent-dispatch.
//
// stateDir must be agentDispatchStateDir() for the one call whose ledger
// review-packet will declare as a reconciled namespace (the real reviewer
// dispatch; see ledgerDir), and unreconciledDispatchStateDir() for every
// other real call (preflight, the real implementer dispatch): see that
// function's doc comment for why conflating the two breaks reconciliation.
//
// err is non-nil only when the process itself could not be started (binary
// missing, not executable, ...); a non-zero exit is a normal, expected
// outcome this function's callers inspect themselves, never an error here.
func (r *runner) runAgentDispatch(ctx context.Context, args []string, stateDir string) (output string, exitCode int, err error) {
	cmd := exec.CommandContext(ctx, r.cfg.agentDispatchBin, args...)
	cmd.Dir = r.cfg.repoRoot
	cmd.Env = append(os.Environ(), "AGENT_DISPATCH_STATE_DIR="+stateDir)
	out, runErr := cmd.CombinedOutput()
	if runErr == nil {
		return string(out), 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return string(out), exitErr.ExitCode(), nil
	}
	return string(out), -1, runErr
}

// agentDispatchStateDir is the fixed AGENT_DISPATCH_STATE_DIR for the one
// real agent-dispatch call whose ledger becomes a reconciled namespace: the
// real reviewer dispatch (stepReviewerDispatch), whose run id review-packet
// records as the namespace's only authorized launch (see ledgerDir).
func (r *runner) agentDispatchStateDir() string {
	return filepath.Join(r.cfg.storeDir, "agent-dispatch-state")
}

// unreconciledDispatchStateDir is the fixed AGENT_DISPATCH_STATE_DIR for
// every real agent-dispatch call that is not the reviewer's packet-bound
// dispatch: both roles' preflight "hi" checks, and the real implementer
// dispatch. These runs are never declared as a namespace by any host
// record, so hma host reconcile never scans them -- and they must not share
// agentDispatchStateDir()'s directory with the reviewer's run, because
// hma host reconcile's rule is "every run physically present in a declared
// namespace must match a reservation or packet": if these shared that
// directory, every preflight and every implementer attempt would show up
// as a HOST_UNAUTHORIZED_DISPATCH finding once the reviewer's packet
// declares it, blocking stepVerification on every real-dispatch run,
// regardless of how clean the implementation and review actually were. A
// live dogfood run caught this: see the dispatch report.
func (r *runner) unreconciledDispatchStateDir() string {
	return filepath.Join(r.cfg.storeDir, "agent-dispatch-state-unreconciled")
}

// repoKeyFor replicates agent-dispatch's own repo_key_for exactly
// (scripts/agent-run-ledger, repo_key_for): the basename of the resolved
// real repository path, "-", and the first 16 hex characters of
// sha256(resolved real path). hmarun must predict this value, before any
// dispatch happens, to know the exact ledger root a real reviewer dispatch
// will use once AGENT_DISPATCH_STATE_DIR is set (ledger_root_for's
// AGENT_DISPATCH_STATE_DIR branch resolves the ledger to
// "$AGENT_DISPATCH_STATE_DIR/$repo_key" with no fallback) -- review-packet's
// ledger_dir field must name that exact path before the reviewer ever
// launches, not after. This is a real coupling to agent-dispatch's own
// naming, read from its source, not a guess; see the dispatch report.
func repoKeyFor(root string) (string, error) {
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	name := filepath.Base(real)
	if name == "" || name == "/" || name == "." {
		name = "repo"
	}
	name = strings.ReplaceAll(name, "/", "-")
	return fmt.Sprintf("%s-%s", name, sha256Hex(real)[:16]), nil
}

func (r *runner) stepImplPreflight(ctx context.Context) error {
	if _, ok, err := r.findLatestValidPreflight(hoststate.RoleImplementation); err != nil || ok {
		return err
	}
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	routeDigest, err := r.routeApprovalDigest(core)
	if err != nil {
		return err
	}
	impl := r.impl
	impl.RouteApprovalDigest = routeDigest
	return r.issuePreflight(ctx, hoststate.RoleImplementation, impl)
}

func (r *runner) stepImplDispatch(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	for _, rec := range history {
		if rec.Kind == hoststate.KindDispatch && rec.Dispatch != nil && rec.Dispatch.UnitID == r.cfg.unit && rec.Dispatch.Role == hoststate.RoleImplementation {
			return nil
		}
	}
	pf, ok, err := r.findLatestValidPreflight(hoststate.RoleImplementation)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no valid implementation preflight; run stepImplPreflight first")
	}
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	routeDigest, err := r.routeApprovalDigest(core)
	if err != nil {
		return err
	}
	impl := r.impl
	impl.RouteApprovalDigest = routeDigest
	in := dispatchInput{envelope: r.envelope(), Dispatch: hoststate.Dispatch{
		UnitID: r.cfg.unit, Role: hoststate.RoleImplementation, Route: impl, PreflightRecordDigest: pf,
		BoundedTask: "implement unit " + r.cfg.unit, WorkContextDigest: sha256Hex("hmarun/v1/work-context:impl:" + r.cfg.runID + ":" + r.cfg.unit),
	}}
	path, err := r.writeJSON("dispatch-impl.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "dispatch", "--input", path, "--store", r.cfg.storeDir)
}

// stepWorkerStub is STUB POINT 2, "the worker runs": it makes one trivial,
// clearly-labeled change to the unit's first allowed file and commits it
// locally in the throwaway repository, standing in for a real
// agent-dispatch implementation run. The next dispatch replaces this with a
// real agent-dispatch call; nothing else in this file depends on how that
// change is produced, only that it ends in one clean commit on repoRoot.
func (r *runner) stepWorkerStub(ctx context.Context) error {
	out, err := repostate.Output(ctx, r.cfg.repoRoot, "log", "-1", "--format=%s")
	if err == nil && repostate.Trim(out) == stubWorkerCommitMessage {
		return nil
	}
	var unit model.PlanUnit
	for _, u := range r.plan.Units {
		if u.ID == r.cfg.unit {
			unit = u
		}
	}
	if len(unit.AllowedFiles) == 0 {
		return fmt.Errorf("unit %q declares no allowed_files for the stub worker to touch", r.cfg.unit)
	}
	target := filepath.Join(r.cfg.repoRoot, unit.AllowedFiles[0])
	f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout, "STUB: not a real agent-dispatch run; appending a marker line to %s\n", target)
	_, werr := f.WriteString("# hmarun STUB worker change (not a real agent-dispatch run)\n")
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", r.cfg.repoRoot, "add", unit.AllowedFiles[0]).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.cfg.repoRoot,
		"-c", "user.name=hmarun stub worker", "-c", "user.email=hmarun-stub@example.invalid",
		"commit", "-qm", stubWorkerCommitMessage)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}

// workerPromptPath is the scratch file hmarun writes one real worker
// dispatch attempt's task prompt to. It lives under artifactsDir, inside
// --store, never inside repoRoot: a file written into the repository would
// show up in `git status` and change launch_worktree_sha256, corrupting the
// review artifact's digest (stepReviewPacket) before the worker has even
// started.
func (r *runner) workerPromptPath(attempt int) string {
	return filepath.Join(r.artifactsDir, fmt.Sprintf("worker-prompt-%s-%d.md", r.cfg.unit, attempt))
}

// buildWorkerPrompt renders the scoped task brief the real implementer
// receives: the unit's own plan fields (never a bare "do the task") plus
// the contract's intent and the criteria this unit advances.
// priorFailureContext, when non-empty, is the previous attempt's command and
// combined output, appended as failure context for stepWorkerDispatch's one
// bounded retry.
//
// The delivery requirement below is deliberately not folded into
// forbidden_actions: the default `hmarun init` plan template
// (cmd/hmarun/build.go's buildContractAndPlan) sets forbidden_actions to
// ["git push", "git commit"], read literally meaning "do not commit" --
// yet hmarun can only observe the worker's result through a commit (exactly
// as the pre-existing stepWorkerStub already assumed). This prompt passes
// forbidden_actions through verbatim, faithful to the unit's own approved
// scope, and states the delivery requirement as a separate, explicit
// instruction rather than silently resolving the apparent contradiction;
// see the dispatch report.
func (r *runner) buildWorkerPrompt(unit model.PlanUnit, priorFailureContext string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are the implementer for HMA plan unit %q, run %q.\n\n", unit.ID, r.cfg.runID)
	fmt.Fprintf(&b, "Contract intent: %s\n\n", r.contract.Intent)
	fmt.Fprintf(&b, "Unit behavior (what this unit must do): %s\n\n", unit.Behavior)
	fmt.Fprint(&b, "Criteria this unit advances:\n")
	for _, cid := range unit.CriteriaAdvanced {
		for _, c := range r.contract.Criteria {
			if c.ID == cid {
				fmt.Fprintf(&b, "  - %s: %s (threshold: %s)\n", c.ID, c.Statement, c.Threshold)
			}
		}
	}
	fmt.Fprint(&b, "\nScope boundaries from the approved plan (hard constraints; hmarun checks allowed_files/forbidden_files itself from the actual git diff, not from your report of what you touched):\n")
	fmt.Fprintf(&b, "  allowed_files: %s\n", strings.Join(unit.AllowedFiles, ", "))
	fmt.Fprintf(&b, "  forbidden_files: %s\n", strings.Join(unit.ForbiddenFiles, ", "))
	fmt.Fprintf(&b, "  allowed_actions: %s\n", strings.Join(unit.AllowedActions, ", "))
	fmt.Fprintf(&b, "  forbidden_actions: %s\n", strings.Join(unit.ForbiddenActions, ", "))
	fmt.Fprintf(&b, "  minimum_implementation: %s\n", unit.MinimumImplementation)
	fmt.Fprintf(&b, "  stop_conditions: %s\n", strings.Join(unit.StopConditions, "; "))
	fmt.Fprintf(&b, "  escalation_conditions: %s\n", strings.Join(unit.EscalationConditions, "; "))
	fmt.Fprint(&b, "\nDelivery requirement (separate from the boundaries above): once your change is complete and "+
		"stays within allowed_files only, make exactly one git commit containing it. hmarun observes your work only "+
		"through the commit you produce; no new commit on HEAD means no implementation record, regardless of how "+
		"correct the working tree looks.\n")
	if priorFailureContext != "" {
		fmt.Fprintf(&b, "\nThis is a retry: a previous attempt at this exact unit did not succeed. Previous attempt:\n%s\n"+
			"Diagnose and fix whatever caused that failure; do not repeat it.\n", priorFailureContext)
	}
	return b.String()
}

// repoHead returns repoRoot's current HEAD commit.
func (r *runner) repoHead(ctx context.Context) (string, error) {
	out, err := repostate.Output(ctx, r.cfg.repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return repostate.Trim(out), nil
}

// changedFilesSince returns the repo-relative paths that differ between two
// revisions of repoRoot, or nil if they are the same revision.
func (r *runner) changedFilesSince(ctx context.Context, before, after string) ([]string, error) {
	if before == after {
		return nil, nil
	}
	out, err := repostate.Output(ctx, r.cfg.repoRoot, "diff", "--name-only", before, after)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(repostate.Trim(out), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// firstOutOfScope returns the first path in changed that violates the
// unit's own allowed_files/forbidden_files (not in allowed, when allowed is
// non-empty; or in forbidden), or "" if every changed path is in scope.
// hmarun decides this itself from the real git diff, never from the
// worker's own report of what it touched.
func firstOutOfScope(changed, allowed, forbidden []string) string {
	for _, f := range changed {
		if len(allowed) > 0 && !containsPath(allowed, f) {
			return f
		}
		if containsPath(forbidden, f) {
			return f
		}
	}
	return ""
}

func containsPath(list []string, f string) bool {
	for _, p := range list {
		if p == f {
			return true
		}
	}
	return false
}

// truncateForPrompt bounds how much of a dispatch's raw output is echoed
// back into a retry prompt or an error message, so one runaway worker
// cannot blow up the next attempt's prompt size or an error log.
func truncateForPrompt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]...\n"
}

// stepWorkerDispatch is stepWorkerStub's real counterpart: it dispatches the
// real implementer (ming) with a prompt built from the unit's own plan
// fields, then verifies the result itself rather than trusting the
// worker's self-report: HEAD must have actually moved, and every changed
// path must be confined to allowed_files and outside forbidden_files
// (checked via a real `git diff --name-only`, exactly the discipline this
// project has used for every dispatch all session).
//
// Recovery ("no dead-end agents"): a dispatch that produces no commit at
// all (crash, exit code, local tool error, ...) is recoverable -- it gets
// exactly one automatic retry, carrying the previous attempt's command and
// output forward as failure context, before hmarun gives up and asks a
// human to decide. A dispatch that commits something outside the unit's
// scope, or that commits but exits non-zero, is not retried automatically:
// retrying on top of an existing, possibly-wrong commit would just risk
// compounding it, so both of those stop immediately with a precise
// description of what happened and what a human needs to decide (revert,
// widen scope, or re-dispatch under tighter instructions). This is a
// focused first version, not a general failure-recovery system; see the
// dispatch report for exactly what is and is not handled here.
func (r *runner) stepWorkerDispatch(ctx context.Context) error {
	headBefore, err := r.repoHead(ctx)
	if err != nil {
		return err
	}
	if headBefore != r.base {
		// A prior attempt (this process or an earlier, crashed one) already
		// moved HEAD past the run's frozen base; resuming must not dispatch
		// a second worker on top of it.
		return nil
	}

	var unit model.PlanUnit
	var found bool
	for _, u := range r.plan.Units {
		if u.ID == r.cfg.unit {
			unit, found = u, true
		}
	}
	if !found {
		return fmt.Errorf("unit %q not found in plan", r.cfg.unit)
	}

	cfg := dispatchConfigFor[hoststate.RoleImplementation]
	const maxAttempts = 2 // one real attempt, one bounded retry; see the doc comment above.
	var priorFailureContext string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Count the retry itself here, once, regardless of how it
			// concludes below (success, an out-of-scope commit, a non-zero
			// exit, a launch error, or exhaustion). An earlier version only
			// counted this on the success and exhaustion paths, so a retry
			// that hit the out-of-scope/non-zero-exit/launch-error branches
			// was blocked but never counted as an attempted recovery.
			r.summary.recoveryAttempts++
		}
		promptPath := r.workerPromptPath(attempt)
		if err := os.WriteFile(promptPath, []byte(r.buildWorkerPrompt(unit, priorFailureContext)), 0o644); err != nil {
			return err
		}
		runID, err := r.freshDispatchRunID(fmt.Sprintf("impl-attempt%d", attempt))
		if err != nil {
			return err
		}
		args := withModelFlag([]string{cfg.Agent}, cfg.Model)
		args = append(args, "--effort", cfg.Effort, "--run-id", runID, "--prompt-file", promptPath)
		fmt.Fprintf(r.stdout, "Dispatching real implementer: agent-dispatch %s (attempt %d/%d)\n", strings.Join(args, " "), attempt, maxAttempts)
		out, exitCode, runErr := r.runAgentDispatch(ctx, args, r.unreconciledDispatchStateDir())
		if runErr != nil {
			r.summary.blockedUnrecoverable++
			return fmt.Errorf("agent-dispatch worker dispatch: could not launch %s: %w", r.cfg.agentDispatchBin, runErr)
		}
		receiptPath := lastFieldValue(out, "AGENT_DISPATCH_RECEIPT")

		headAfter, err := r.repoHead(ctx)
		if err != nil {
			return err
		}
		if headAfter != headBefore {
			changed, err := r.changedFilesSince(ctx, headBefore, headAfter)
			if err != nil {
				return err
			}
			if bad := firstOutOfScope(changed, unit.AllowedFiles, unit.ForbiddenFiles); bad != "" {
				r.summary.blockedUnrecoverable++
				return fmt.Errorf(
					"agent-dispatch worker dispatch (run-id %s) committed %q, which is outside unit %q's approved scope "+
						"(allowed_files=%v forbidden_files=%v); HEAD moved %s -> %s. hmarun does not revert or retry "+
						"this automatically: a human must decide whether to revert the commit, widen the unit's "+
						"allowed_files, or re-dispatch under tighter instructions. Receipt: %s",
					runID, bad, r.cfg.unit, unit.AllowedFiles, unit.ForbiddenFiles, headBefore, headAfter, receiptPath)
			}
			if exitCode != 0 {
				r.summary.blockedUnrecoverable++
				return fmt.Errorf(
					"agent-dispatch worker dispatch (run-id %s) produced an in-scope commit %s but exited %d (not a "+
						"clean success); hmarun does not auto-revert or retry on top of an existing commit -- a human "+
						"must inspect receipt %s and decide whether to keep this commit, revert it, or re-dispatch "+
						"for cleanup.", runID, headAfter, exitCode, receiptPath)
			}
			if attempt > 1 {
				// This attempt is the one bounded retry, and it succeeded:
				// one recoverable block, resolved automatically. The retry
				// itself was already counted above, when this attempt began.
				r.summary.blockedRecoverable++
				r.summary.recoverySucceeded++
			}
			fmt.Fprintf(r.stdout, "Real implementer dispatch (run-id %s) produced commit %s, confined to %v.\n", runID, headAfter, changed)
			return nil
		}

		// No commit: recoverable. Carry this attempt's command and output
		// forward as failure context for exactly one retry.
		priorFailureContext = fmt.Sprintf("command: agent-dispatch %s\nexit code: %d\nreceipt: %s\noutput (combined stdout+stderr):\n%s",
			strings.Join(args[1:], " "), exitCode, receiptPath, truncateForPrompt(out, 8000))
		if attempt == maxAttempts {
			// The one bounded retry was attempted and still produced no
			// commit: the retry is exhausted, so this block is unrecoverable.
			// The retry itself was already counted above, when it began.
			r.summary.blockedUnrecoverable++
			return fmt.Errorf(
				"agent-dispatch worker dispatch for unit %q produced no commit after %d attempt(s) (last run-id %s, "+
					"exit code %d, receipt %s). hmarun's one automatic retry is exhausted; this needs a human "+
					"decision: inspect the receipt/output below, then either re-run `hmarun run` once the underlying "+
					"problem is fixed, or escalate to a different route for this unit.\nLast attempt's output:\n%s",
				r.cfg.unit, maxAttempts, runID, exitCode, receiptPath, truncateForPrompt(out, 4000))
		}
		fmt.Fprintf(r.stdout, "Real implementer dispatch (run-id %s) produced no commit (exit %d); retrying once with failure context.\n", runID, exitCode)
	}
	return nil // unreachable: the loop above always returns by the last attempt.
}

func (r *runner) stepImplementationReview(ctx context.Context) error {
	return r.doTransition(ctx, model.StageImplementationAuthorization, model.StageImplementationReview, "", r.cfg.unit, false,
		fmt.Sprintf("Present unit %q's implementation for review.", r.cfg.unit))
}

// --- step 10-16: repository binding through launch reconciliation ---

func (r *runner) stepBindRepository(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	if hoststate.RunRepository(history) != nil {
		return nil
	}
	in := bindRepositoryInput{envelope: r.envelope(), RepositoryRoot: r.cfg.repoRoot, RepositoryIdentity: r.repositoryIdentity}
	path, err := r.writeJSON("bind-repository.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "bind-repository", "--input", path, "--store", r.cfg.storeDir)
}

func (r *runner) stepReviewPreflight(ctx context.Context) error {
	if _, ok, err := r.findLatestValidPreflight(hoststate.RoleReview); err != nil || ok {
		return err
	}
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	routeDigest, err := r.routeApprovalDigest(core)
	if err != nil {
		return err
	}
	reviewer := r.reviewer
	reviewer.RouteApprovalDigest = routeDigest
	return r.issuePreflight(ctx, hoststate.RoleReview, reviewer)
}

func (r *runner) reviewPacketPath() string {
	return filepath.Join(r.artifactsDir, fmt.Sprintf("review-packet-%s.md", r.cfg.unit))
}

// ledgerDir is the agent-dispatch ledger root review-packet names as
// ledger_dir and review-import reads the reviewer's receipt from. In
// --stub-dispatch mode it is the fixed fake ledger directory
// stepReviewerStub always wrote to. Otherwise it is the exact root a real
// agent-dispatch invocation will use once hmarun sets
// AGENT_DISPATCH_STATE_DIR (see runAgentDispatch): computed, via
// repoKeyFor, not guessed, and fixed before the reviewer ever launches,
// since review-packet must record it in the packet binding ahead of that
// dispatch.
func (r *runner) ledgerDir() (string, error) {
	if r.cfg.stubDispatch {
		return filepath.Join(r.cfg.storeDir, "ledger"), nil
	}
	key, err := repoKeyFor(r.cfg.repoRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.agentDispatchStateDir(), key), nil
}

// reviewDispatchRunID is deterministic (not random) so a resumed run always
// names the same agent-dispatch run id it already recorded, in both
// --stub-dispatch mode and real dispatch (where it is the --run-id
// pre-allocated for the reviewer's agent-dispatch call; see
// stepReviewerDispatch).
func (r *runner) reviewDispatchRunID() string {
	return "hmarun-review-" + r.cfg.runID + "-" + r.cfg.unit
}

// findReviewDispatch returns the host dispatch record of the unit's latest
// HMA-generated review packet for the given implementation record digest,
// or nil.
func findReviewDispatch(history []hoststate.Record, unit, implementationDigest string) *hoststate.Record {
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		if rec.Kind == hoststate.KindDispatch && rec.Dispatch != nil && rec.Dispatch.UnitID == unit &&
			rec.Dispatch.Role == hoststate.RoleReview && rec.Dispatch.Packet != nil &&
			rec.Dispatch.ImplementationRecordDigest == implementationDigest {
			return &history[i]
		}
	}
	return nil
}

// reviewDispatchNeedsReissue reports whether an existing review dispatch
// record for this unit is a dead end that stepReviewPacket must replace with
// a fresh packet (fresh nonce, fresh review_dispatch_id, fresh run id)
// rather than resume: true exactly when the dispatch was never imported AND
// its agent-dispatch attempt (if any) never produced usable output. A
// dispatch that was already imported (a real verdict exists) is never
// reissued, no matter what its output looked like -- that would let a
// second reviewer attempt overwrite a verdict a human may already be
// looking at. A dispatch that has not been attempted yet, or whose attempt
// produced usable output awaiting import, is not reissued either: it is
// still a live attempt for stepReviewerDispatch/stepReviewImport to pick up.
func (r *runner) reviewDispatchNeedsReissue(history []hoststate.Record, dispatch *hoststate.Record) (bool, error) {
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit &&
			rec.Review.ReviewDispatchDigest == dispatch.HeadAnchor && rec.Review.ReportDigest != "" {
			return false, nil // already imported.
		}
	}
	ledgerDir, err := r.ledgerDir()
	if err != nil {
		return false, err
	}
	receiptPath := filepath.Join(ledgerDir, "runs", dispatch.Dispatch.Packet.DispatchRunID, "receipt.json")
	if usable, _ := reviewerDispatchOutputUsable(receiptPath, r.reportPath()); usable {
		return false, nil // a real attempt already landed usable output; let it be imported, don't reissue.
	}
	// Not imported, and no usable output at this run id -- either a prior
	// attempt crashed, or (a resumed run only, after hmarun itself crashed
	// between minting the packet and ever invoking agent-dispatch) no
	// attempt was made yet. The two look identical from here (both are just
	// "no receipt at this path"), and reissuing is safe either way -- it
	// costs at most one unused run id -- so this always reissues rather
	// than trying to tell them apart.
	return true, nil
}

// reviewTaskText is the review packet's "Task" field (buildReviewPacket,
// cmd/hma/review.go): the absolute path the reviewer's AUDIT.json must be
// written to, plus the AUDIT.json schema itself. hma_binding's shape is
// already in the packet's own "Report binding (required)" section, rendered
// above this text, so this does not repeat that; it does spell out every
// other field, with exact allowed key names and values -- a live smoke test
// against a real reviewer (gpt-5.3-codex via agent-dispatch, see the
// dispatch report) found that without this, a real, capable model produces
// a plausible-looking but non-conformant report (wrong key names such as
// "criteria" instead of "requirements", a missing "isolation" field,
// "PASS" instead of the required lowercase "pass"), which
// hoststate.EvaluateReviewImport then rejects outright
// (HOST_REVIEW_REPORT_MISMATCH) even though the reviewer did real,
// substantive work. The schema below must stay equal to auditReport
// (internal/host/review_import.go) and ValidateAuditReport
// (internal/host/report_contract.go) -- those are the actual contract; this
// is instructions for producing a document that satisfies it, not a third,
// independent definition of it.
func (r *runner) reviewTaskText() string {
	var unit model.PlanUnit
	for _, u := range r.plan.Units {
		if u.ID == r.cfg.unit {
			unit = u
		}
	}
	ids := unit.CriteriaAdvanced
	firstID := "C1"
	idList := `"C1"`
	if len(ids) > 0 {
		firstID = ids[0]
		quoted := make([]string, len(ids))
		for i, id := range ids {
			quoted[i] = fmt.Sprintf("%q", id)
		}
		idList = strings.Join(quoted, ", ")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Review unit %s.\n\n", r.cfg.unit)
	fmt.Fprintf(&b, "Write your AUDIT.json to exactly this absolute path: %s\n", r.reportPath())
	b.WriteString("(create any missing parent directories). Follow this packet's own \"Report\n")
	b.WriteString("binding (required)\" section above for what hma_binding must contain.\n\n")
	b.WriteString("Your AUDIT.json MUST be a single JSON object with EXACTLY these top-level\n")
	b.WriteString("keys -- no others, no renamed keys, no nested restructuring:\n\n")
	b.WriteString(`  "audit_id":      a nonempty string you choose (e.g. a short slug).` + "\n")
	fmt.Fprintf(&b, "  %-16s a nonempty string naming what you reviewed (e.g. %q).\n", `"target":`, "unit "+r.cfg.unit)
	b.WriteString(`  "modes":         a nonempty array drawn only from "plan", "code", "reasoning",` + "\n")
	b.WriteString(`                   "outcome" (use ["code"] for a code review).` + "\n")
	b.WriteString(`  "phase1_frozen": the boolean true.` + "\n")
	b.WriteString(`  "limitations":   an array of strings describing what you could not check` + "\n")
	b.WriteString(`                   (an empty array [] if none).` + "\n")
	b.WriteString(`  "verdict":       exactly the lowercase string "pass" if every requirement` + "\n")
	b.WriteString(`                   below passed with no blocking finding, or exactly` + "\n")
	b.WriteString(`                   "fix-required" if not. No other value is read as either` + "\n")
	b.WriteString(`                   outcome -- "conditional-pass", "blocked", "PASS", "Pass", etc.` + "\n")
	b.WriteString(`                   are all read as an escalation, not as your actual verdict.` + "\n")
	b.WriteString(`  "isolation":     exactly one of "isolated", "partial", or "compromised"` + "\n")
	b.WriteString(`                   (lowercase), describing how independently you reviewed this` + "\n")
	b.WriteString(`                   (usually "isolated").` + "\n")
	b.WriteString(`  "hma_binding":   {"packet_sha256": ..., "review_dispatch_id": ...,` + "\n")
	b.WriteString(`                   "review_nonce": ...} exactly as specified above.` + "\n")
	fmt.Fprintf(&b, "  %-16s an array with exactly one entry per criterion this unit advances: %s.\n", `"requirements":`, idList)
	b.WriteString(`                   Each entry: {"id": "<criterion id>",` + "\n")
	b.WriteString(`                   "status": "pass"|"fail"|"unknown", "evidence_refs": [<nonempty` + "\n")
	b.WriteString(`                   strings>]}; evidence_refs is required whenever status is` + "\n")
	b.WriteString(`                   "pass" or "fail" (cite what you actually checked, e.g. a` + "\n")
	b.WriteString(`                   command you ran).` + "\n")
	b.WriteString(`  "findings":      an array of finding objects, or an empty array [] if you` + "\n")
	b.WriteString(`                   have none. Each finding (if any): {"id": <unique string>,` + "\n")
	b.WriteString(`                   "title": <nonempty string>, "diagnosis": <nonempty string>,` + "\n")
	b.WriteString(`                   "kind": "defect"|"risk"|"gap"|"opportunity", "mode": one of` + "\n")
	b.WriteString(`                   the modes you listed above, "severity":` + "\n")
	b.WriteString(`                   "critical"|"high"|"medium"|"low", "confidence":` + "\n")
	b.WriteString(`                   "observed"|"inferred"|"hypothesis", "evidence":` + "\n")
	b.WriteString(`                   [{"source": <string>, "observation": <string>}, ...]` + "\n")
	b.WriteString(`                   (required, non-empty, when confidence is "observed").` + "\n")
	b.WriteString(`                   A finding whose "kind" is "defect", "risk", or "gap" ALSO` + "\n")
	b.WriteString(`                   requires a "fix" object (omit it only for "kind": "opportunity"):` + "\n")
	b.WriteString(`                   {"action": <nonempty string: what to change>,` + "\n")
	b.WriteString(`                   "why": <nonempty string: why this fixes it>,` + "\n")
	b.WriteString(`                   "tradeoffs": <nonempty string: cost/risk of the fix, or "none">,` + "\n")
	b.WriteString(`                   "alternative": <nonempty string: another option, or "none">,` + "\n")
	b.WriteString(`                   "effort": "small"|"medium"|"large"|"unknown",` + "\n")
	b.WriteString(`                   "status": "proposed"|"validated"|"remediation_blocked",` + "\n")
	b.WriteString(`                   "verification": [{"method": <nonempty string>,` + "\n")
	b.WriteString(`                   "expected": <nonempty string>}, ...] (nonempty array)}.` + "\n")
	b.WriteString(`                   A defect/risk/gap finding reported WITHOUT a conformant "fix"` + "\n")
	b.WriteString(`                   object is read as contract-invalid and your verdict is escalated` + "\n")
	b.WriteString(`                   rather than mapped to "pass" or "fix-required", regardless of` + "\n")
	b.WriteString(`                   what "verdict" you wrote.` + "\n\n")
	b.WriteString("Minimal worked example for a clean pass with no findings (adapt the ids,\n")
	b.WriteString("hma_binding, and whatever you actually observed -- do not copy this verbatim):\n\n")
	fmt.Fprintf(&b, "  {\n")
	fmt.Fprintf(&b, "    \"audit_id\": \"review-%s\",\n", r.cfg.unit)
	fmt.Fprintf(&b, "    \"target\": \"unit %s\",\n", r.cfg.unit)
	b.WriteString("    \"modes\": [\"code\"],\n")
	b.WriteString("    \"phase1_frozen\": true,\n")
	b.WriteString("    \"limitations\": [],\n")
	b.WriteString("    \"verdict\": \"pass\",\n")
	b.WriteString("    \"isolation\": \"isolated\",\n")
	b.WriteString("    \"hma_binding\": {\"packet_sha256\": \"...\", \"review_dispatch_id\": \"...\", \"review_nonce\": \"...\"},\n")
	fmt.Fprintf(&b, "    \"requirements\": [{\"id\": %q, \"status\": \"pass\", \"evidence_refs\": [\"ran the criterion's verification command\"]}],\n", firstID)
	b.WriteString("    \"findings\": []\n")
	b.WriteString("  }\n\n")
	b.WriteString("Any top-level key not listed above, or a value outside the exact allowed set\n")
	b.WriteString("given, is not read as you intend: use these exact names and values.")
	return b.String()
}

func (r *runner) stepReviewPacket(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	implementationDigest, err := hoststate.RequireImplementation(history, r.cfg.unit)
	if err != nil {
		return err
	}
	runID := r.reviewDispatchRunID()
	if existing := findReviewDispatch(history, r.cfg.unit, implementationDigest); existing != nil {
		switch reissue, err := r.reviewDispatchNeedsReissue(history, existing); {
		case err != nil:
			return err
		case !reissue:
			return nil // already imported, or its dispatch output is still usable: resume it, don't reissue.
		default:
			// The existing dispatch produced no usable output and was never
			// imported, so its run id is permanently consumed (agent-run-ledger
			// refuses to reuse a run id once it reaches mkdir) and can never be
			// retried -- see stepReviewerDispatch's doc comment. Mint a fresh
			// packet below with a fresh run id instead of resuming the dead one.
			runID, err = r.freshDispatchRunID("review-reissue")
			if err != nil {
				return err
			}
		}
	}
	pf, ok, err := r.findLatestValidPreflight(hoststate.RoleReview)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no valid review preflight; run stepReviewPreflight first")
	}
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	routeDigest, err := r.routeApprovalDigest(core)
	if err != nil {
		return err
	}
	reviewer := r.reviewer
	reviewer.RouteApprovalDigest = routeDigest

	packetPath := r.reviewPacketPath()
	if _, err := os.Stat(packetPath); err == nil {
		// No host record referenced this path (checked above); it can only
		// be debris from a crashed previous attempt that never committed.
		if err := os.Remove(packetPath); err != nil {
			return err
		}
	}
	ledgerDir, err := r.ledgerDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ledgerDir, 0o755); err != nil {
		return err
	}

	in := reviewPacketInput{envelope: r.stubReviewEnvelope(), Dispatch: hoststate.Dispatch{
		UnitID: r.cfg.unit, Role: hoststate.RoleReview, Route: reviewer, PreflightRecordDigest: pf,
		ImplementationRecordDigest: implementationDigest,
		BoundedTask:                "review unit " + r.cfg.unit,
		WorkContextDigest:          sha256Hex("hmarun/v1/work-context:review:" + r.cfg.runID + ":" + r.cfg.unit),
		Risk:                       model.RouteRiskHigh,
	},
		DispatchRunID: runID, IssuingReason: hoststate.IssueFinal,
		RepositoryRoot: r.cfg.repoRoot, BaseRevision: r.base, Task: r.reviewTaskText(),
		ContractPath: r.cfg.contractPath, PolicyPath: r.policyPath, PlanPath: r.cfg.planPath, LedgerDir: ledgerDir,
	}
	path, err := r.writeJSON("review-packet-input.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "review-packet", "--input", path, "--store", r.cfg.storeDir, "--out", packetPath)
}

// stepReviewerStub is STUB POINT 2, "the reviewer runs": it hand-writes a
// clearly-labeled AUDIT.json (verdict pass, every requirement pass) and the
// agent-dispatch receipt review-import reads, both bound to the real packet
// hma host review-packet already generated and wrote to disk. The next
// dispatch replaces this with a real agent-dispatch reviewer run; review
// hma_binding and the receipt's packet_sha256/launch_* fields must stay
// real (copied verbatim from the packet record), which is exactly what a
// real reviewer run would also have to produce to pass import.
func (r *runner) stepReviewerStub(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	implementationDigest, err := hoststate.RequireImplementation(history, r.cfg.unit)
	if err != nil {
		return err
	}
	dispatch := findReviewDispatch(history, r.cfg.unit, implementationDigest)
	if dispatch == nil {
		return errors.New("no review dispatch found; run stepReviewPacket first")
	}
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit &&
			rec.Review.ReviewDispatchDigest == dispatch.HeadAnchor && rec.Review.ReportDigest != "" {
			return nil // already imported; nothing to (re)write.
		}
	}
	binding := dispatch.Dispatch.Packet
	var advanced string
	for _, u := range r.plan.Units {
		if u.ID == r.cfg.unit && len(u.CriteriaAdvanced) > 0 {
			advanced = u.CriteriaAdvanced[0]
		}
	}
	report := map[string]any{
		"audit_id": "hmarun-stub-audit", "target": r.cfg.unit, "modes": []any{"code"}, "phase1_frozen": true, "limitations": []any{},
		"verdict": "pass", "isolation": "isolated", "findings": []any{},
		"requirements": []any{map[string]any{"id": advanced, "status": "pass", "evidence_refs": []any{"hmarun-stub"}}},
		"hma_binding": map[string]any{
			"packet_sha256": dispatch.Dispatch.ReviewPacketDigest, "review_dispatch_id": binding.ReviewDispatchID, "review_nonce": binding.ReviewNonce,
		},
	}
	fmt.Fprintln(r.stdout, "STUB: not a real agent-dispatch reviewer run; writing a synthetic AUDIT.json and receipt")
	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	reportPath := r.reportPath()
	// The receipt's sha256 must match these exact on-disk bytes; no trailing
	// newline is appended here (unlike writeJSON's transient inputs), since
	// review-import recomputes the digest straight from the file it reads.
	if err := os.WriteFile(reportPath, reportBytes, 0o644); err != nil {
		return err
	}
	sum := sha256Hex(string(reportBytes))
	receipt := map[string]any{
		"run_id": binding.DispatchRunID, "agent": r.reviewer.WorkerID, "model": r.reviewer.Model, "provider": r.reviewer.Provider,
		"effort": "medium", "started_at": time.Now().UTC().Format(time.RFC3339), "project_root": r.cfg.repoRoot,
		"mode": "review", "state": "completed", "exit_code": 0,
		"packet_sha256": dispatch.Dispatch.ReviewPacketDigest, "launch_head": binding.ExpectedLaunchHead, "launch_worktree_sha256": binding.ExpectedLaunchWorktree,
		"report_files": []any{map[string]any{"path": reportPath, "sha256": sum}},
	}
	ledgerDir, err := r.ledgerDir()
	if err != nil {
		return err
	}
	receiptDir := filepath.Join(ledgerDir, "runs", binding.DispatchRunID)
	if err := os.MkdirAll(receiptDir, 0o755); err != nil {
		return err
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(receiptDir, "receipt.json"), append(receiptBytes, '\n'), 0o644)
}

// stepReviewerDispatch is stepReviewerStub's real counterpart: it dispatches
// the real reviewer (phil) directly against the packet hma host
// review-packet already wrote, using the packet file itself as the prompt,
// byte for byte -- so agent-dispatch's own packet_sha256 (which it computes
// over whatever it was given as --prompt-file) and HMA's ReviewPacketDigest
// (sha256 of the exact packet file hma wrote) have a chance to be the same
// digest; see the dispatch report for a real blocker found here: agent-
// dispatch's prompt="$(cat "$prompt_file")" is a bash command substitution,
// which strips the packet file's trailing newline before hashing it, so in
// production this equality currently fails even with no wrapper text added.
//
// Unlike stepWorkerDispatch, this function does not itself decide success
// or failure beyond "did the dispatch produce something stepReviewImport can
// actually evaluate" (reviewerDispatchOutputUsable): all real content
// validation (packet/launch/report binding matching, and the verdict
// itself) is stepReviewImport's and hoststate.EvaluateReviewImport's job,
// unchanged. If the real reviewer's shape does not match what that code
// expects, stepReviewImport's own error is the honest signal -- this
// function does not try to paper over that with a shim.
//
// Recovery ("no dead-end agents") here is NOT a same-invocation retry, and
// deliberately does not mirror stepWorkerDispatch's shape: agent-run-ledger
// creates the run directory with a bare `mkdir` (no -p) before any worker
// is ever launched, so once --run-id has reached that point once -- which a
// genuine crash or infra hiccup implies, since the crash represents the
// *worker* failing after the ledger already accepted the run, not
// agent-dispatch's own argument validation rejecting it -- that run id can
// never be reused; a second invocation with the same --run-id fails
// immediately in the real tool ("Run id already exists", exit 73), it does
// not get a second real attempt. And unlike the worker's fresh per-attempt
// run id, this packet's DispatchRunID is baked into the packet's own hashed
// bytes (ReviewPacketDigest), so a fresh run id needs an entirely fresh
// packet (new nonce, new review_dispatch_id, new digest) -- not a retried
// flag value, and not an action hmarun takes on its own authority. So a
// dispatch that produces no usable output at all -- no receipt, a receipt
// that does not even parse or carries no exit_code, or a report file that
// cannot be read -- looks like a crash or infra hiccup, but still escalates
// immediately, in exactly one invocation, with a precise, actionable
// message naming why a retry can't work and what a human needs to do next
// (an earlier version of this function retried once with the same run id;
// that retry call would reliably fail against the real tool for exactly
// the crash scenarios it existed to handle, so it was removed in favor of
// honest escalation). A dispatch that actually completes -- a parseable
// receipt with an exit_code, and a readable report -- is, as before, never
// retried or escalated, no matter what verdict that report carries: APPROVE,
// REQUEST_CHANGES, BLOCK, and ESCALATE are all real results a human must
// see, not something to paper over by asking again.
func (r *runner) stepReviewerDispatch(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	implementationDigest, err := hoststate.RequireImplementation(history, r.cfg.unit)
	if err != nil {
		return err
	}
	dispatch := findReviewDispatch(history, r.cfg.unit, implementationDigest)
	if dispatch == nil {
		return errors.New("no review dispatch found; run stepReviewPacket first")
	}
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit &&
			rec.Review.ReviewDispatchDigest == dispatch.HeadAnchor && rec.Review.ReportDigest != "" {
			return nil // already imported; nothing to (re)dispatch.
		}
	}
	binding := dispatch.Dispatch.Packet
	ledgerDir, err := r.ledgerDir()
	if err != nil {
		return err
	}
	receiptPath := filepath.Join(ledgerDir, "runs", binding.DispatchRunID, "receipt.json")
	if usable, _ := reviewerDispatchOutputUsable(receiptPath, r.reportPath()); usable {
		// A prior attempt already landed usable output at the pre-allocated
		// run id; resume rather than dispatch a second reviewer against the
		// same packet (agent-run-ledger would refuse to reuse the run id
		// anyway). A merely-existing-but-unusable receipt (e.g. a process
		// killed mid-write) does NOT count here, so a resumed run gets its
		// own fresh attempt(s) rather than wedging on crash debris.
		return nil
	}

	cfg := dispatchConfigFor[hoststate.RoleReview]
	args := withModelFlag([]string{cfg.Agent, "--review"}, cfg.Model)
	args = append(args, "--effort", cfg.Effort, "--run-id", binding.DispatchRunID, "--prompt-file", r.reviewPacketPath(),
		// agent-run-ledger only lists a file in the receipt's report_files
		// (which stepReviewImport's binding check requires: internal/host/
		// review_import.go's listed-file loop) when told to track it via
		// --report-file; it is not auto-detected. A live smoke test against
		// the real tool found this flag missing here, which left
		// report_files empty on every real reviewer dispatch and made every
		// real review-import fail with HOST_REVIEW_REPORT_MISMATCH even
		// though the reviewer wrote a correct, well-formed AUDIT.json.
		"--report-file", r.reportPath())

	fmt.Fprintf(r.stdout, "Dispatching real reviewer: agent-dispatch %s\n", strings.Join(args, " "))
	out, exitCode, runErr := r.runAgentDispatch(ctx, args, r.agentDispatchStateDir())
	if runErr != nil {
		r.summary.blockedUnrecoverable++
		return fmt.Errorf("agent-dispatch review dispatch: could not launch %s: %w", r.cfg.agentDispatchBin, runErr)
	}
	if usable, _ := reviewerDispatchOutputUsable(receiptPath, r.reportPath()); usable {
		fmt.Fprintf(r.stdout, "Real reviewer dispatch (run-id %s, exit %d) produced a receipt and a report; stepReviewImport will validate them.\n", binding.DispatchRunID, exitCode)
		return nil
	}
	_, detail := reviewerDispatchOutputUsable(receiptPath, r.reportPath())
	r.summary.blockedUnrecoverable++
	return fmt.Errorf(
		"agent-dispatch review dispatch (run-id %s) produced no usable output (exit %d: %s). "+
			"This cannot be retried with the same run id: agent-run-ledger refuses to reuse a run id once "+
			"it has created that run's directory, and this review packet's run id is baked into the packet's "+
			"own hashed bytes, so a fresh attempt needs a fresh packet, not a retried flag value. A human "+
			"decision is needed: inspect ledger directory %s to diagnose the failure, then either re-run "+
			"`hmarun run` (stepReviewPacket will detect this dead dispatch and mint a fresh packet with a "+
			"new run id automatically), or escalate to a different route for this unit."+
			"\nLast attempt's output:\n%s",
		binding.DispatchRunID, exitCode, detail, ledgerDir, truncateForPrompt(out, 8000))
}

// reviewerDispatchOutputUsable reports whether one stepReviewerDispatch
// attempt produced something stepReviewImport can actually evaluate: a
// receipt file that parses as JSON and carries a non-nil exit_code, and a
// readable report file. This is deliberately shallow -- real content
// validation (packet/launch/report binding, the verdict itself) is
// stepReviewImport's and hoststate.EvaluateReviewImport's job; this only
// distinguishes "the dispatch crashed or produced nothing" (recoverable,
// retry) from "the dispatch actually ran to completion" (never retried,
// regardless of what verdict it carries).
func reviewerDispatchOutputUsable(receiptPath, reportPath string) (ok bool, detail string) {
	receiptBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		return false, fmt.Sprintf("no receipt at %s", receiptPath)
	}
	var rc hoststate.Receipt
	if err := json.Unmarshal(receiptBytes, &rc); err != nil {
		return false, fmt.Sprintf("receipt at %s is not valid JSON (%v)", receiptPath, err)
	}
	if rc.ExitCode == nil {
		return false, fmt.Sprintf("receipt at %s carries no exit_code", receiptPath)
	}
	if _, err := os.ReadFile(reportPath); err != nil {
		return false, fmt.Sprintf("report at %s is unreadable (%v)", reportPath, err)
	}
	return true, ""
}

func (r *runner) reportPath() string {
	return filepath.Join(r.artifactsDir, fmt.Sprintf("AUDIT-%s.json", r.cfg.unit))
}

func (r *runner) stepReviewImport(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	implementationDigest, err := hoststate.RequireImplementation(history, r.cfg.unit)
	if err != nil {
		return err
	}
	dispatch := findReviewDispatch(history, r.cfg.unit, implementationDigest)
	if dispatch == nil {
		return errors.New("no review dispatch found; run stepReviewPacket first")
	}
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit &&
			rec.Review.ReviewDispatchDigest == dispatch.HeadAnchor && rec.Review.ReportDigest != "" {
			return nil
		}
	}
	ledgerDir, err := r.ledgerDir()
	if err != nil {
		return err
	}
	in := reviewImportInput{envelope: r.stubReviewEnvelope(), ReviewDispatchID: dispatch.Dispatch.Packet.ReviewDispatchID, LedgerDir: ledgerDir, ReportPath: r.reportPath()}
	path, err := r.writeJSON("review-import.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "review-import", "--input", path, "--store", r.cfg.storeDir)
}

// latestImportedReview returns the host head anchor of the unit's latest
// imported (reports.ReportDigest != "") review record.
func (r *runner) latestImportedReview(history []hoststate.Record) (*hoststate.Record, error) {
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit && rec.Review.ReportDigest != "" {
			return &history[i], nil
		}
	}
	return nil, errors.New("no imported review found; run stepReviewImport first")
}

func (r *runner) stepFindingsImport(ctx context.Context) error {
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	review, err := r.latestImportedReview(history)
	if err != nil {
		return err
	}
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	for _, rec := range core {
		if rec.Kind == model.KindFinding && rec.FindingsSnapshot && rec.FindingsSource != nil && rec.FindingsSource.ReviewRecordDigest == review.HeadAnchor {
			return nil
		}
	}
	in := findingsImportInput{envelope: r.stubReviewEnvelope(), ReviewRecordDigest: review.HeadAnchor, ReportPath: r.reportPath()}
	path, err := r.writeJSON("findings-import.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "findings-import", "--input", path, "--store", r.cfg.storeDir)
}

func (r *runner) stepReconcile(ctx context.Context) error {
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	if len(core) == 0 {
		return errors.New("core run missing")
	}
	state := engine.ProjectStageState(core)
	// Reconcile exists only to unlock the IMPLEMENTATION_REVIEW ->
	// VERIFICATION gate (stepVerification); once that has happened the core
	// chain keeps moving (verification, independent validation, credit,
	// PARTIAL), which would make any reconciliation look stale against the
	// live core head forever after, with nothing left that needs it. A run
	// already at or past VERIFICATION needs no further reconciliation.
	if stageOrder[state.CurrentStage] >= stageOrder[model.StageVerification] || state.Terminal {
		return nil
	}
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	currentCoreHead := core[len(core)-1].Metadata.HeadAnchor
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		if rec.Kind == hoststate.KindLaunchReconciliation && rec.Reconciliation != nil {
			if rec.CoreHeadAnchor == currentCoreHead && len(rec.Reconciliation.Unauthorized) == 0 {
				return nil
			}
			break
		}
	}
	in := reconcileInput{envelope: r.envelope(), PolicyPath: r.policyPath}
	path, err := r.writeJSON("reconcile.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "reconcile", "--input", path, "--store", r.cfg.storeDir)
}

// --- step 17-21: verification through the PARTIAL terminal outcome ---

func (r *runner) stepVerification(ctx context.Context) error {
	return r.doTransition(ctx, model.StageImplementationReview, model.StageVerification, "", r.cfg.unit, false,
		r.stubPrefixed(fmt.Sprintf("Present unit %q's reviewed implementation for verification.", r.cfg.unit)))
}

func (r *runner) stepIndependentValidation(ctx context.Context) error {
	return r.doTransition(ctx, model.StageVerification, model.StageIndependentValidation, "", r.cfg.unit, false,
		r.stubPrefixed(fmt.Sprintf("Advance unit %q to independent validation.", r.cfg.unit)))
}

// advancedCriterionID returns the one criterion id the run's unit advances
// (hmarun's own fixed single-criterion-per-unit convention; see report).
func (r *runner) advancedCriterionID() (string, error) {
	for _, u := range r.plan.Units {
		if u.ID == r.cfg.unit && len(u.CriteriaAdvanced) > 0 {
			return u.CriteriaAdvanced[0], nil
		}
	}
	return "", fmt.Errorf("unit %q advances no criterion", r.cfg.unit)
}

func (r *runner) stepCredit(ctx context.Context) error {
	core, err := r.coreRecords()
	if err != nil {
		return err
	}
	criterionID, err := r.advancedCriterionID()
	if err != nil {
		return err
	}
	for _, c := range engine.ProjectResolutionState(core).Criteria {
		if c.ID == criterionID && c.Disposition == model.CriterionPassed {
			return nil
		}
	}
	history, err := r.hostHistory()
	if err != nil {
		return err
	}
	review, err := r.latestImportedReview(history)
	if err != nil {
		return err
	}
	if _, err := r.confirm(r.stubPrefixed(fmt.Sprintf("Credit unit %q's criterion %q from the imported independent review.", r.cfg.unit, criterionID))); err != nil {
		return err
	}
	in := creditInput{envelope: r.stubReviewEnvelope(), ReviewRecord: review.HeadAnchor, ReportPath: r.reportPath(), ContractPath: r.cfg.contractPath}
	path, err := r.writeJSON("credit.json", in)
	if err != nil {
		return err
	}
	return r.execHMA(ctx, "host", "credit", "--input", path, "--store", r.cfg.storeDir)
}

func (r *runner) stepPartial(ctx context.Context) error {
	return r.doTransition(ctx, model.StageIndependentValidation, "", model.OutcomePartial, r.cfg.unit, false,
		r.stubPrefixed(fmt.Sprintf("Record unit %q's run %q as PARTIAL: at least one criterion resolved, at least one not.", r.cfg.unit, r.cfg.runID)))
}
