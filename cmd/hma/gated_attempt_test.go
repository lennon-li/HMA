package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// attemptFixture is one run with an accepted contract, an accepted plan, an
// approved route for U1, a repository binding, an implementation preflight,
// and one verify-loop controller state directory.
type attemptFixture struct {
	t                          *testing.T
	store, root, base          string
	plan, contract, policy     string
	planDigest, contractDigest string
	impl                       hoststate.Route
	loopDir                    string
	t0                         time.Time
	nonce                      int
}

func newAttemptFixture(t *testing.T, planAttempts int) *attemptFixture {
	t.Helper()
	f := &attemptFixture{t: t, store: t.TempDir(), t0: time.Now().UTC().Truncate(time.Second).Add(-2 * time.Minute)}
	f.root, f.base, _, _ = ciRepo(t)
	f.impl = cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	id, err := hoststate.RouteIdentity(f.impl)
	if err != nil {
		t.Fatal(err)
	}
	pol := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
	proj, _ := model.NormalizeRoutePolicy(pol)
	policyDigest, _ := engine.RoutePolicyDigest(proj)
	all := []string{model.PermissionReadOnly, model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	c := model.AcceptanceContract{Schema: model.AcceptanceContractSchema, RunID: "run", Intent: "one behavior", NonGoals: []string{"none"},
		Criteria: []model.ContractCriterion{{ID: "C1", Statement: "tests pass", Threshold: "exit 0", EvidenceRequirement: "fresh capture",
			Check: model.Check{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}}},
		Risk: model.RouteRiskLow, Deliverables: []string{"code"},
		Budgets:         model.ContractBudgets{MaxAttempts: 10, MaxWallMinutes: 120, CostCeiling: 20, CostUnit: "usd"},
		PermittedRoutes: []string{id, "esc", "val"}, PermissionEnvelope: all, PolicyDigest: policyDigest}
	cb, _ := json.Marshal(c)
	_, f.contractDigest, err = model.ParseAcceptanceContract(cb)
	if err != nil {
		t.Fatal(err)
	}
	p := model.Plan{Schema: model.PlanSchema, RunID: "run", AcceptanceContractDigest: f.contractDigest, PolicyDigest: policyDigest,
		Units: []model.PlanUnit{{ID: "U1", Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"a"}, AllowedActions: []string{"edit"},
			ForbiddenFiles: []string{"docs"}, ForbiddenActions: []string{"push"}, PreChangeEvidence: "failing test", MinimumImplementation: "m",
			ReuseAnalysis: "r", PermissionClasses: all, ImplementerRoute: id, EscalationRoute: "esc", ValidatorRoute: "val",
			IndependenceRequirement: "family", VerificationCommands: []model.Check{{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}},
			RequiredEvidence: []string{"go test"}, StopConditions: []string{"s"}, RollbackConditions: []string{"r"}, EscalationConditions: []string{"e"},
			NonGoals: []string{"n"}, Dependencies: []string{},
			Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 100, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: planAttempts, MaxWallMinutes: 60, CostCeiling: 10}}}}
	pb, _ := json.Marshal(p)
	_, f.planDigest, err = model.ParsePlan(pb)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	f.plan, f.contract, f.policy = filepath.Join(dir, "plan.json"), filepath.Join(dir, "contract.json"), filepath.Join(dir, "policy.json")
	polb, _ := json.Marshal(pol)
	for path, b := range map[string][]byte{f.plan: pb, f.contract: cb, f.policy: polb} {
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.appendCore(model.Record{Kind: model.KindCriterion, Stage: model.StageAcceptanceCriteria, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}})
	f.approveCore("", model.StageAcceptanceCriteria, model.StagePlanning, "contract")
	f.approveCore("", model.StagePlanning, model.StageRouteSelection, "plan")
	f.authorizeRoute()
	bindRepository(t, f.store, "run", f.root, "repo")
	rf := &reviewFixture{t: t, store: f.store}
	rf.step("decision", hostDecisionInput{hostEnvelope: fxEnv(f.ts(10 * time.Second)), Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "dispatch plus execution plus review", Rationale: "delegation is cheaper",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}, WorkContextDigest: "ctx-fury", Route: f.impl,
	}})
	rf.step("preflight", hostPreflightInput{hostEnvelope: fxEnv(f.ts(20 * time.Second)), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: f.impl, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-impl",
		ExpiresAt: f.ts(50 * time.Minute)}})
	f.loopDir = f.newLoop("vl-1", nil)
	return f
}

func (f *attemptFixture) ts(d time.Duration) string { return f.t0.Add(d).Format(time.RFC3339) }

func (f *attemptFixture) appendCore(r model.Record) {
	f.t.Helper()
	s := store.New(f.store)
	recs, _ := s.Load("run")
	r.Version = 1
	r.Metadata = model.RecordMetadata{RunID: "run", Sequence: int64(len(recs) + 1), Timestamp: f.ts(0), Actor: "operator-confirmed:lennon"}
	if len(recs) > 0 {
		r.Metadata.PredecessorHash = recs[len(recs)-1].Metadata.HeadAnchor
	}
	if err := s.Append(&r); err != nil {
		f.t.Fatalf("append core: %v", err)
	}
}

func (f *attemptFixture) approveCore(unit string, from, to model.Stage, digest string) {
	f.nonce++
	f.appendCore(model.Record{UnitID: unit, Kind: model.KindStageTransition, Stage: from, Control: model.ControlApproved,
		Approval: &model.ApprovalBinding{UnitID: unit, RunID: "run", TransitionDigest: digest, CurrentStage: from, ProposedTargetStage: to,
			RepositoryIdentityDigest: "repo", BaseRevisionDigest: f.base, StageTimeDigest: "t", AcceptedPlanDigest: f.planDigest,
			AcceptanceContractDigest: f.contractDigest, ChallengeNonce: fmt.Sprintf("n%d", f.nonce), Approver: "human", Timestamp: f.ts(0)}})
}

func (f *attemptFixture) authorizeRoute() {
	f.approveCore("U1", model.StageRouteSelection, model.StageImplementationAuthorization, "approved-route")
}

// newLoop writes a verify-loop controller state for runID and returns its
// directory; edit may change the envelope before it is hashed.
func (f *attemptFixture) newLoop(runID string, edit func(env map[string]any)) string {
	f.t.Helper()
	dir := f.t.TempDir()
	env := map[string]any{"hma": map[string]any{"store": f.store, "run_id": "run", "unit_id": "U1"},
		"authority_hook": map[string]any{"argv": []any{"hma", "host", "attempt"}}, "escalate_to": "stop",
		"max_attempts": 2, "max_minutes": 30,
		"route":  map[string]any{"worker": "jax", "model": "luna", "effort": "medium", "cost": map[string]any{"spend_cap_usd": 4, "per_attempt_usd": 1}},
		"verify": map[string]any{"argv": []any{"true"}}}
	if edit != nil {
		edit(env)
	}
	eb, _ := json.Marshal(env)
	sum, err := hoststate.PythonCanonicalSHA256(eb)
	if err != nil {
		f.t.Fatal(err)
	}
	st := map[string]any{"run_id": runID, "status": "running", "envelope": env, "envelope_sha256": sum, "repo": map[string]any{"root": f.root}, "attempts": []any{}}
	b, _ := json.Marshal(st)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), b, 0o600); err != nil {
		f.t.Fatal(err)
	}
	return dir
}

// request builds a hook request as verify-loop authority_request does.
func (f *attemptFixture) request(loopDir string, n int, dispatch string, sig any) map[string]any {
	b, _ := os.ReadFile(filepath.Join(loopDir, "state.json"))
	var st map[string]any
	json.Unmarshal(b, &st)
	env := st["envelope"].(map[string]any)
	return map[string]any{"controller_run_id": st["run_id"], "attempt_index": n, "dispatch_run_id": dispatch,
		"envelope_sha256": st["envelope_sha256"], "route": env["route"], "last_failure_signature": sig,
		"live_root": f.root, "clone_baseline_digest": emptyBaselineDigest, "ledger_namespace": filepath.Join(loopDir, "dispatch-ledger"), "hma": env["hma"]}
}

// emptyBaselineDigest is verify-loop's digest of an absent baseline
// inventory: sha256 of "{}".
const emptyBaselineDigest = "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"

// syncPending records req's attempt as the controller's pending attempt, as
// verify-loop does before it calls the hook (attempt appended, phase
// "dispatching", last_check carrying the failure signature).
func (f *attemptFixture) syncPending(req map[string]any) {
	f.t.Helper()
	ns, _ := req["ledger_namespace"].(string)
	path := filepath.Join(filepath.Dir(ns), "state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var st map[string]any
	json.Unmarshal(b, &st)
	var check any
	if req["last_failure_signature"] != nil {
		check = map[string]any{"signature": req["last_failure_signature"]}
	}
	attempts, _ := st["attempts"].([]any)
	pending := map[string]any{"n": req["attempt_index"], "dispatch_run_id": req["dispatch_run_id"], "phase": "dispatching",
		"route": req["route"], "last_check": check}
	if n := len(attempts); n > 0 {
		if last, _ := attempts[n-1].(map[string]any); last["dispatch_run_id"] == req["dispatch_run_id"] {
			attempts = attempts[:n-1]
		}
	}
	st["attempts"] = append(attempts, pending)
	b, _ = json.Marshal(st)
	os.WriteFile(path, b, 0o600)
}

func (f *attemptFixture) attempt(req map[string]any) (attemptResult, error) {
	f.t.Helper()
	f.syncPending(req)
	return f.attemptRaw(req)
}

func (f *attemptFixture) attemptRaw(req map[string]any) (attemptResult, error) {
	f.t.Helper()
	path := filepath.Join(f.t.TempDir(), "request.json")
	b, _ := json.Marshal(req)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		f.t.Fatal(err)
	}
	var res attemptResult
	out, err := captureStdout(f.t, func() error {
		return run([]string{"host", "attempt", "--store", f.store, "--plan", f.plan, "--contract", f.contract, "--policy", f.policy, "--request", path})
	})
	if err == nil {
		if jerr := json.Unmarshal(out, &res); jerr != nil {
			f.t.Fatalf("result %s: %v", out, jerr)
		}
	}
	return res, err
}

func (f *attemptFixture) mustAttempt(req map[string]any) attemptResult {
	f.t.Helper()
	res, err := f.attempt(req)
	if err != nil {
		f.t.Fatalf("attempt: %v", err)
	}
	return res
}

func (f *attemptFixture) count(kind hoststate.RecordKind) int {
	h, _ := hoststate.NewStore(f.store).Load("run")
	n := 0
	for _, r := range h {
		if r.Kind == kind {
			n++
		}
	}
	return n
}

func wantErr(t *testing.T, err error, reason string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), reason) {
		t.Fatalf("err = %v, want %s", err, reason)
	}
}

func TestReservationIdempotent(t *testing.T) {
	f := newAttemptFixture(t, 3)
	req := f.request(f.loopDir, 1, "d-1", nil)
	first := f.mustAttempt(req)
	again := f.mustAttempt(req)
	if !first.LaunchEligible || first.Replay || !again.Replay || again.ReservationID != first.ReservationID || !again.LaunchEligible {
		t.Fatalf("first %+v again %+v", first, again)
	}
	if f.count(hoststate.KindReservation) != 1 || f.count(hoststate.KindLoopAdmission) != 1 {
		t.Fatal("replay charged again")
	}
	// Same key with another request digest conflicts; a dispatch id cannot
	// be reused under another attempt.
	_, err := f.attempt(f.request(f.loopDir, 1, "d-1", "changed"))
	wantErr(t, err, hoststate.ReasonReservationConflict)
	_, err = f.attempt(f.request(f.loopDir, 2, "d-1", nil))
	wantErr(t, err, hoststate.ReasonReservationConflict)
}

func TestNewIDWithoutCapacityRefused(t *testing.T) {
	f := newAttemptFixture(t, 2)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	f.mustAttempt(f.request(f.loopDir, 2, "d-2", nil))
	_, err := f.attempt(f.request(f.loopDir, 3, "d-3", nil))
	wantErr(t, err, string(engine.ReasonLeaseCeiling))
}

func TestFinalSlotReplayLaunchesOnce(t *testing.T) {
	f := newAttemptFixture(t, 2)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	last := f.request(f.loopDir, 2, "d-2", nil)
	f.mustAttempt(last)
	// The response was lost: the replay may launch the charged slot.
	if r := f.mustAttempt(last); !r.Replay || !r.LaunchEligible {
		t.Fatalf("lost-response replay %+v", r)
	}
	if f.count(hoststate.KindReservation) != 2 {
		t.Fatal("replay charged a new slot")
	}
}

func TestFinalSlotReplayAfterUseNotEligible(t *testing.T) {
	f := newAttemptFixture(t, 2)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	last := f.request(f.loopDir, 2, "d-2", nil)
	f.mustAttempt(last)
	if err := os.MkdirAll(filepath.Join(f.loopDir, "dispatch-ledger", "runs", "d-2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if r := f.mustAttempt(last); !r.Replay || r.LaunchEligible || r.NextAction != "DO_NOT_LAUNCH" {
		t.Fatalf("used slot replay %+v", r)
	}
}

func TestUnknownLaunchCounts(t *testing.T) {
	f := newAttemptFixture(t, 2)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil)) // never launched
	// A second controller run sees one attempt left: an envelope asking
	// for two is not admitted, one is.
	two := f.newLoop("vl-2", nil)
	_, err := f.attempt(f.request(two, 1, "d-2", nil))
	wantErr(t, err, hoststate.ReasonAdmissionRefused)
	one := f.newLoop("vl-3", func(env map[string]any) { env["max_attempts"] = 1 })
	f.mustAttempt(f.request(one, 1, "d-3", nil))
	_, err = f.attempt(f.request(one, 2, "d-4", nil))
	wantErr(t, err, string(engine.ReasonLeaseCeiling))
}

func TestCountersCarryAcrossRewind(t *testing.T) {
	f := newAttemptFixture(t, 2)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	f.approveCore("U1", model.StageImplementationAuthorization, model.StageRouteSelection, "rewind")
	f.authorizeRoute()
	// The old controller run's admission belongs to the old generation.
	_, err := f.attempt(f.request(f.loopDir, 2, "d-2", nil))
	wantErr(t, err, hoststate.ReasonAdmissionRefused)
	// A new controller run under the new lease has one attempt left.
	two := f.newLoop("vl-2", nil)
	_, err = f.attempt(f.request(two, 1, "d-3", nil))
	wantErr(t, err, hoststate.ReasonAdmissionRefused)
	one := f.newLoop("vl-3", func(env map[string]any) { env["max_attempts"] = 1 })
	f.mustAttempt(f.request(one, 1, "d-4", nil))
	_, err = f.attempt(f.request(one, 2, "d-5", nil))
	wantErr(t, err, string(engine.ReasonLeaseCeiling))
}

func TestSameFailureSignatureTwiceRefused(t *testing.T) {
	f := newAttemptFixture(t, 3)
	loop := f.newLoop("vl-2", func(env map[string]any) { env["max_attempts"] = 3 })
	f.mustAttempt(f.request(loop, 1, "d-1", nil))
	f.mustAttempt(f.request(loop, 2, "d-2", "sig-A"))
	_, err := f.attempt(f.request(loop, 3, "d-3", "sig-A"))
	wantErr(t, err, hoststate.ReasonRepeatedFailure)
	f.mustAttempt(f.request(loop, 3, "d-4", "sig-B"))
}

func TestDifferentRouteRefused(t *testing.T) {
	f := newAttemptFixture(t, 3)
	for name, edit := range map[string]func(r map[string]any){
		"model":  func(r map[string]any) { r["model"] = "other" },
		"worker": func(r map[string]any) { r["worker"] = "liz" },
		"effort": func(r map[string]any) { r["effort"] = "high" },
	} {
		req := f.request(f.loopDir, 1, "d-"+name, nil)
		route := map[string]any{}
		for k, v := range req["route"].(map[string]any) {
			route[k] = v
		}
		edit(route)
		req["route"] = route
		_, err := f.attempt(req)
		wantErr(t, err, hoststate.ReasonAttemptRouteMismatch)
	}
	if f.count(hoststate.KindReservation) != 0 {
		t.Fatal("refused attempt charged")
	}
}

func TestEscalateToMustBeStop(t *testing.T) {
	f := newAttemptFixture(t, 3)
	for i, v := range []any{"jax", map[string]any{"worker": "liz", "cost": map[string]any{"spend_cap_usd": 1, "per_attempt_usd": 1}}} {
		loop := f.newLoop(fmt.Sprintf("vl-esc-%d", i), func(env map[string]any) { env["escalate_to"] = v })
		_, err := f.attempt(f.request(loop, 1, fmt.Sprintf("d-esc-%d", i), nil))
		if err == nil {
			t.Fatalf("escalate_to %v admitted", v)
		}
	}
	if f.count(hoststate.KindLoopAdmission) != 0 {
		t.Fatal("refused envelope admitted")
	}
}

func TestConcurrentLastAttempt(t *testing.T) {
	// One attempt left; two controller runs race for it.
	f := newAttemptFixture(t, 1)
	paths := make([]string, 2)
	for i := range paths {
		loop := f.newLoop(fmt.Sprintf("vl-race-%d", i), func(env map[string]any) { env["max_attempts"] = 1 })
		req := f.request(loop, 1, fmt.Sprintf("d-race-%d", i), nil)
		f.syncPending(req)
		b, _ := json.Marshal(req)
		paths[i] = filepath.Join(t.TempDir(), "request.json")
		os.WriteFile(paths[i], b, 0o600)
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range paths {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = runHostAttempt([]string{"--store", f.store, "--plan", f.plan, "--contract", f.contract, "--policy", f.policy, "--request", paths[i]})
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, e := range errs {
		if e == nil {
			ok++
		}
	}
	if ok != 1 || f.count(hoststate.KindReservation) != 1 {
		t.Fatalf("concurrent last slot: %d succeeded, %d reservations (%v)", ok, f.count(hoststate.KindReservation), errs)
	}
}

// The lease, not the reservation, decides: after the unit leaves
// Implementation authorization a replay is not launch-eligible and a new
// attempt is refused.
func TestReplayAfterLeaseEndsNotEligible(t *testing.T) {
	f := newAttemptFixture(t, 3)
	req := f.request(f.loopDir, 1, "d-1", nil)
	f.mustAttempt(req)
	f.approveCore("U1", model.StageImplementationAuthorization, model.StageImplementationReview, "review")
	if r := f.mustAttempt(req); r.LaunchEligible {
		t.Fatal("replay eligible after the stage moved")
	}
	_, err := f.attempt(f.request(f.loopDir, 2, "d-2", nil))
	wantErr(t, err, hoststate.ReasonLeaseInactive)
}

// A request must name this store and a controller state whose envelope
// hashes to its digest.
func TestAttemptRequestBinding(t *testing.T) {
	f := newAttemptFixture(t, 3)
	for name, edit := range map[string]func(r map[string]any){
		"envelope-digest": func(r map[string]any) { r["envelope_sha256"] = "0000" },
		"other-store": func(r map[string]any) {
			r["hma"] = map[string]any{"store": t.TempDir(), "run_id": "run", "unit_id": "U1"}
		},
		"other-root":      func(r map[string]any) { r["live_root"] = t.TempDir() },
		"no-state":        func(r map[string]any) { r["ledger_namespace"] = filepath.Join(t.TempDir(), "dispatch-ledger") },
		"bad-dispatch-id": func(r map[string]any) { r["dispatch_run_id"] = "../x" },
	} {
		req := f.request(f.loopDir, 1, "d-"+name, nil)
		edit(req)
		if _, err := f.attempt(req); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if f.count(hoststate.KindReservation) != 0 {
		t.Fatal("refused attempt charged")
	}
}

// Jax D3 D3J-02/D3J-04: the namespace must be the controller's own
// dispatch-ledger and the clone baseline digest the controller's.
func TestAttemptNamespaceAndBaselineFromState(t *testing.T) {
	f := newAttemptFixture(t, 3)
	for name, edit := range map[string]func(r map[string]any){
		"sibling-namespace":  func(r map[string]any) { r["ledger_namespace"] = filepath.Join(f.loopDir, "other-ledger") },
		"relative-namespace": func(r map[string]any) { r["ledger_namespace"] = "dispatch-ledger" },
		"wrong-baseline":     func(r map[string]any) { r["clone_baseline_digest"] = "baseline" },
	} {
		req := f.request(f.loopDir, 1, "d-"+name, nil)
		edit(req)
		if _, err := f.attempt(req); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if f.count(hoststate.KindLoopAdmission) != 0 {
		t.Fatal("refused request admitted")
	}
}

// D3J-03: the repeated-failure rule is unit-wide; a new controller run does
// not reset it.
func TestRepeatedFailureAcrossControllerRuns(t *testing.T) {
	f := newAttemptFixture(t, 4)
	a := f.newLoop("vl-a", func(env map[string]any) { env["max_attempts"] = 2 })
	f.mustAttempt(f.request(a, 1, "d-1", "sig-A"))
	b := f.newLoop("vl-b", func(env map[string]any) { env["max_attempts"] = 2 })
	_, err := f.attempt(f.request(b, 1, "d-2", "sig-A"))
	wantErr(t, err, hoststate.ReasonRepeatedFailure)
}

// D3J-01: the clock is read under the locks, and authority that ends during
// commit keeps the charge but grants no launch.
func TestAttemptClockReadUnderLocks(t *testing.T) {
	f := newAttemptFixture(t, 3)
	loop := f.newLoop("vl-clock", func(env map[string]any) { env["max_minutes"] = 1 })
	calls := 0
	expiry := f.t0.Add(50 * time.Minute) // the route preflight's expiry
	attemptNow = func() time.Time {
		calls++
		if calls >= 2 {
			return expiry
		}
		return expiry.Add(-time.Second)
	}
	defer func() { attemptNow = func() time.Time { return time.Now().UTC() } }()
	// First call: admission (clock read 1 under its lock), then the
	// reservation step reads the clock under its own lock at expiry.
	_, err := f.attempt(f.request(loop, 1, "d-1", nil))
	wantErr(t, err, hoststate.ReasonAttemptRouteMismatch)
	calls = -10
	attemptNow = func() time.Time {
		calls++
		if calls > -9 {
			return expiry
		}
		return expiry.Add(-time.Second)
	}
	r, err := f.attempt(f.request(loop, 1, "d-1", nil))
	if err != nil || r.LaunchEligible || f.count(hoststate.KindReservation) != 1 {
		t.Fatalf("lease ending during commit: %+v %v", r, err)
	}
}

// Jax D3 r2 D3R2-01: a new reservation must answer the controller's pending
// attempt: same index, dispatch id, route, and failure signature.
func TestNewReservationMatchesPendingAttempt(t *testing.T) {
	f := newAttemptFixture(t, 3)
	loop := f.newLoop("vl-p", func(env map[string]any) { env["max_attempts"] = 3 })
	f.syncPending(f.request(loop, 1, "d-1", "sig-A"))
	for name, req := range map[string]map[string]any{
		"other-index":       f.request(loop, 99, "d-1", "sig-A"),
		"other-dispatch":    f.request(loop, 1, "d-other", "sig-A"),
		"omitted-signature": f.request(loop, 1, "d-1", nil),
		"other-signature":   f.request(loop, 1, "d-1", "sig-B"),
	} {
		if _, err := f.attemptRaw(req); err == nil {
			t.Errorf("%s granted", name)
		}
	}
	if f.count(hoststate.KindReservation) != 0 {
		t.Fatal("refused request charged")
	}
	if _, err := f.attemptRaw(f.request(loop, 1, "d-1", "sig-A")); err != nil {
		t.Fatalf("matching pending attempt refused: %v", err)
	}
}

// D3R2-02: a first request for a dispatch id the ledger already ran is
// refused before any charge.
func TestNewReservationRefusesUsedDispatchID(t *testing.T) {
	f := newAttemptFixture(t, 3)
	os.MkdirAll(filepath.Join(f.loopDir, "dispatch-ledger", "runs", "d-1"), 0o700)
	_, err := f.attempt(f.request(f.loopDir, 1, "d-1", nil))
	wantErr(t, err, hoststate.ReasonReservationConflict)
	if f.count(hoststate.KindReservation) != 0 {
		t.Fatal("used dispatch id charged")
	}
}
