package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// ledgerRun writes an agent-dispatch receipt for a launch.
func (f *attemptFixture) ledgerRun(loopDir, id, mode string, started time.Time) {
	f.t.Helper()
	dir := filepath.Join(loopDir, "dispatch-ledger", "runs", id)
	os.MkdirAll(dir, 0o700)
	b, _ := json.Marshal(map[string]any{"run_id": id, "mode": mode, "started_at": started.Format(time.RFC3339),
		"agent": f.impl.WorkerID, "model": f.impl.Model, "effort": f.impl.ReasoningEffort})
	if err := os.WriteFile(filepath.Join(dir, "receipt.json"), b, 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *attemptFixture) reconcile() (reconcileResult, error) {
	f.t.Helper()
	input := filepath.Join(f.t.TempDir(), "reconcile.json")
	writeJSON(f.t, input, hostReconcileInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: time.Now().UTC().Format(time.RFC3339)}, PolicyPath: f.policy})
	var res reconcileResult
	out, err := captureStdout(f.t, func() error { return run([]string{"host", "reconcile", "--input", input, "--store", f.store}) })
	if err == nil {
		json.Unmarshal(out, &res)
	}
	return res, err
}

func (f *attemptFixture) gate() error {
	h, _ := hoststate.NewStore(f.store).Load("run")
	return requireReconciledLaunches(f.store, "run", "U1", f.policy, h)
}

// End to end: an authorized launch reconciles clean and opens the gate; an
// unreserved launch records an unowned BLOCK that keeps it closed.
func TestReconcileCommandAndGate(t *testing.T) {
	f := newAttemptFixture(t, 3)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	if err := f.gate(); err == nil || err.Error() != hoststate.ReasonReconciliationRequired {
		t.Fatalf("gate before reconciliation: %v", err)
	}
	f.ledgerRun(f.loopDir, "d-1", "write", time.Now().UTC().Add(2*time.Second))
	res, err := f.reconcile()
	if err != nil || len(res.Reconciliation.Authorized) != 1 || len(res.Reconciliation.Unauthorized) != 0 || res.NextAction != "CONTINUE_HMA_TRANSITION" {
		t.Fatalf("clean reconciliation: %+v %v", res, err)
	}
	if err := f.gate(); err != nil {
		t.Fatalf("gate after clean reconciliation: %v", err)
	}
	// A later reservation needs a newer reconciliation.
	f.mustAttempt(f.request(f.loopDir, 2, "d-2", nil))
	if err := f.gate(); err == nil {
		t.Fatal("gate open with an unreconciled reservation")
	}
	f.ledgerRun(f.loopDir, "d-rogue", "write", time.Now().UTC())
	res, err = f.reconcile()
	if err != nil || len(res.Reconciliation.Unauthorized) != 1 || res.Reconciliation.Unauthorized[0].RunID != "d-rogue" || res.NextAction != "ESCALATE_TO_HUMAN" {
		t.Fatalf("rogue reconciliation: %+v %v", res, err)
	}
	if len(res.Reconciliation.Unlaunched) != 1 || res.Reconciliation.Unlaunched[0] != "d-2" {
		t.Fatalf("unlaunched %+v", res.Reconciliation.Unlaunched)
	}
	core, _ := store.New(f.store).Load("run")
	fs := engine.ProjectResolutionState(core).Findings
	if !hoststate.HasUnauthorizedDispatch(fs) {
		t.Fatalf("no BLOCK finding: %+v", fs)
	}
	if err := f.gate(); err == nil || err.Error() != hoststate.ReasonUnauthorizedDispatch {
		t.Fatalf("gate with unauthorized dispatch: %v", err)
	}
	// Reconciling again does not duplicate the finding.
	f.reconcile()
	core, _ = store.New(f.store).Load("run")
	n := 0
	for _, x := range engine.ProjectResolutionState(core).Findings {
		if strings.HasPrefix(x.ID, hoststate.UnauthorizedDispatchPrefix) {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%d unauthorized findings", n)
	}
}

// TestNewGenerationNeedsNewReservation: after a rewind and a new issuing
// approval, an old reservation's replay starts nothing.
func TestNewGenerationNeedsNewReservation(t *testing.T) {
	f := newAttemptFixture(t, 3)
	req := f.request(f.loopDir, 1, "d-1", nil)
	f.mustAttempt(req)
	f.approveCore("U1", model.StageImplementationAuthorization, model.StageRouteSelection, "rewind")
	f.authorizeRoute()
	r, err := f.attemptRaw(req)
	if err != nil || !r.Replay || r.LaunchEligible {
		t.Fatalf("old reservation under new lease: %+v %v", r, err)
	}
}

// Liz D4 F01: the gate is bound to the ledger a clean reconciliation saw; a
// receipt added afterwards closes it until a fresh reconciliation.
func TestGateBindsReconciledLedger(t *testing.T) {
	f := newAttemptFixture(t, 3)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	f.ledgerRun(f.loopDir, "d-1", "write", time.Now().UTC().Add(2*time.Second))
	if _, err := f.reconcile(); err != nil {
		t.Fatal(err)
	}
	if err := f.gate(); err != nil {
		t.Fatalf("clean: %v", err)
	}
	f.ledgerRun(f.loopDir, "d-rogue", "write", time.Now().UTC())
	if err := f.gate(); err == nil || err.Error() != hoststate.ReasonReconciliationRequired {
		t.Fatalf("new receipt after reconciliation: %v", err)
	}
}

// F09: a clean scan still escalates while an earlier BLOCK stands.
func TestReconcileNextActionKeepsHistoricalBlock(t *testing.T) {
	f := newAttemptFixture(t, 3)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	rogue := filepath.Join(f.loopDir, "dispatch-ledger", "runs", "d-rogue")
	f.ledgerRun(f.loopDir, "d-rogue", "write", time.Now().UTC())
	if res, err := f.reconcile(); err != nil || res.NextAction != "ESCALATE_TO_HUMAN" {
		t.Fatalf("%+v %v", res, err)
	}
	os.RemoveAll(rogue)
	if res, err := f.reconcile(); err != nil || len(res.Reconciliation.Unauthorized) != 0 || res.NextAction != "ESCALATE_TO_HUMAN" {
		t.Fatalf("clean scan with standing BLOCK: %+v %v", res, err)
	}
}

// F05/F08 end to end: the reservation's valid_from is a whole second the
// hook waited for, and a changed policy is recorded as a revocation.
func TestAttemptWholeSecondValidFromAndPolicyRevocation(t *testing.T) {
	f := newAttemptFixture(t, 3)
	var slept time.Duration
	attemptSleep = func(d time.Duration) { slept += d }
	defer func() { attemptSleep = time.Sleep }()
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	h, _ := hoststate.NewStore(f.store).Load("run")
	vf, _ := time.Parse(time.RFC3339Nano, h[len(h)-1].Reservation.ValidFrom)
	if vf.Nanosecond() != 0 {
		t.Fatalf("valid_from %s not a whole second", vf)
	}
	pol := model.RoutePolicy{Version: model.RoutePolicyVersionV1, PermissionClasses: []string{model.PermissionReadOnly}}
	b, _ := json.Marshal(pol)
	os.WriteFile(f.policy, b, 0o600)
	if _, err := f.attempt(f.request(f.loopDir, 2, "d-2", nil)); err == nil {
		t.Fatal("changed policy granted")
	}
	h, _ = hoststate.NewStore(f.store).Load("run")
	n := 0
	for _, r := range h {
		if r.Kind == hoststate.KindPolicyRevocation {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%d policy revocations", n)
	}
}

// Liz D4 F04: review import reads the receipt only from the packet's ledger.
func TestReviewImportUsesPacketLedger(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, nil)
	other := t.TempDir()
	os.MkdirAll(filepath.Join(other, "runs", f.runID), 0o700)
	b, _ := os.ReadFile(filepath.Join(f.ledger, "runs", f.runID, "receipt.json"))
	os.WriteFile(filepath.Join(other, "runs", f.runID, "receipt.json"), b, 0o600)
	input := filepath.Join(t.TempDir(), "import.json")
	writeJSON(t, input, hostReviewImportInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:06:00Z"},
		ReviewDispatchID: f.rdID, LedgerDir: other, ReportPath: rp})
	if err := run([]string{"host", "review-import", "--input", input, "--store", f.store}); err == nil || err.Error() != hoststate.ReasonReviewReceiptMismatch {
		t.Fatalf("other ledger: %v", err)
	}
	if err := f.importReport(rp); err != nil {
		t.Fatalf("packet ledger: %v", err)
	}
}

// Liz D4 r2 F-01/F-03 through the CLI: a correctly bound review launch is
// authorized under its generation's retained plan and policy; the same
// launch under a changed current policy with no dated revocation, or with
// the implementation worker as the actual reviewer, is an unauthorized BLOCK.
func TestReconcileReviewLaunchCLI(t *testing.T) {
	reconcileWith := func(f *reviewFixture, policy string) reconcileResult {
		t.Helper()
		in := filepath.Join(t.TempDir(), "reconcile.json")
		writeJSON(t, in, hostReconcileInput{hostEnvelope: fxEnv("2026-09-16T12:09:00Z"), PolicyPath: policy})
		raw, err := captureStdout(t, func() error { return run([]string{"host", "reconcile", "--input", in, "--store", f.store}) })
		if err != nil {
			t.Fatal(err)
		}
		var got reconcileResult
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	f.receipt(f.report("pass", "isolated", nil), nil)
	if got := reconcileWith(f, f.policy); len(got.Reconciliation.Authorized) != 1 || len(got.Reconciliation.Unauthorized) != 0 {
		t.Fatalf("control: %+v", got.Reconciliation)
	}
	changed := filepath.Join(t.TempDir(), "policy.json")
	b, _ := json.Marshal(model.RoutePolicy{Version: model.RoutePolicyVersionV1, PermissionClasses: []string{model.PermissionReadOnly}})
	if err := os.WriteFile(changed, b, 0o600); err != nil {
		t.Fatal(err)
	}
	got := reconcileWith(f, changed)
	if len(got.Reconciliation.Unauthorized) != 1 || got.Reconciliation.Unauthorized[0].Reason != "POLICY_REVOCATION_UNPROVEN" || got.NextAction != "ESCALATE_TO_HUMAN" {
		t.Fatalf("changed policy: %+v %s", got.Reconciliation, got.NextAction)
	}
	g := newReviewFixture(t, hoststate.IssuePerChange, false)
	g.receipt(g.report("pass", "isolated", nil), func(r map[string]any) {
		r["agent"], r["model"], r["provider"] = g.impl.WorkerID, g.impl.Model, g.impl.Provider
	})
	got = reconcileWith(g, g.policy)
	if len(got.Reconciliation.Unauthorized) != 1 || got.Reconciliation.Unauthorized[0].Reason != "ROUTE_MISMATCH" {
		t.Fatalf("implementation worker as reviewer: %+v", got.Reconciliation)
	}
	core, _ := store.New(g.store).Load("run")
	if !hoststate.HasUnauthorizedDispatch(engine.ProjectResolutionState(core).Findings) {
		t.Fatal("no BLOCK finding recorded")
	}
}

// Liz D4 r2 F-02: once HMA records a revocation of the plan policy,
// restoring the policy bytes revives neither a replay nor a fresh
// reservation of that generation; a separately approved new generation
// can reserve again.
func TestPolicyRestoreDoesNotReviveGeneration(t *testing.T) {
	f := newAttemptFixture(t, 4)
	req := f.request(f.loopDir, 1, "d-1", nil)
	f.mustAttempt(req)
	original, err := os.ReadFile(f.policy)
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := json.Marshal(model.RoutePolicy{Version: model.RoutePolicyVersionV1, PermissionClasses: []string{model.PermissionReadOnly}})
	if err := os.WriteFile(f.policy, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = f.attempt(f.request(f.loopDir, 2, "d-2", nil))
	wantErr(t, err, string(engine.ReasonLeasePolicyChanged))
	if f.count(hoststate.KindPolicyRevocation) != 1 {
		t.Fatal("no policy revocation recorded")
	}
	if err := os.WriteFile(f.policy, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := f.attemptRaw(req); err != nil || got.LaunchEligible || !got.Replay || got.NextAction != "DO_NOT_LAUNCH" {
		t.Fatalf("restored-policy replay: %+v %v", got, err)
	}
	_, err = f.attempt(f.request(f.loopDir, 2, "d-2", nil))
	wantErr(t, err, string(engine.ReasonLeasePolicyChanged))
	f.approveCore("U1", model.StageImplementationAuthorization, model.StageRouteSelection, "rewind")
	f.authorizeRoute()
	if got := f.mustAttempt(f.request(f.newLoop("vl-2", nil), 1, "d-3", nil)); !got.LaunchEligible {
		t.Fatalf("new generation: %+v", got)
	}
}

// Liz D4 r3 F-01: a reconciliation that sees a changed policy records the
// revocation, so restoring the policy revives neither a replay nor a fresh
// reservation of the old generation.
func TestReconcilePolicyObservationPersists(t *testing.T) {
	f := newAttemptFixture(t, 4)
	req := f.request(f.loopDir, 1, "d-1", nil)
	f.mustAttempt(req)
	original, _ := os.ReadFile(f.policy)
	changed, _ := json.Marshal(model.RoutePolicy{Version: model.RoutePolicyVersionV1, PermissionClasses: []string{model.PermissionReadOnly}})
	if err := os.WriteFile(f.policy, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := f.reconcile()
	if err != nil || len(res.Reconciliation.RevokedPolicies) != 1 || res.Reconciliation.ObservedAt == "" {
		t.Fatalf("observation not recorded: %+v %v", res.Reconciliation, err)
	}
	if err := os.WriteFile(f.policy, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := f.attemptRaw(req); err != nil || got.LaunchEligible {
		t.Fatalf("replay after restore: %+v %v", got, err)
	}
	_, err = f.attempt(f.request(f.loopDir, 2, "d-2", nil))
	wantErr(t, err, string(engine.ReasonLeasePolicyChanged))
	// Observing it again under the same generation records nothing new.
	if err := os.WriteFile(f.policy, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if res, _ := f.reconcile(); len(res.Reconciliation.RevokedPolicies) != 0 {
		t.Fatalf("duplicate observation: %+v", res.Reconciliation)
	}
}

// Liz D4 r3 F-02: a policy change observed again under a later generation is
// recorded again, so restoring the policy does not revive that generation.
func TestRepeatedPolicyChangeNewGeneration(t *testing.T) {
	f := newAttemptFixture(t, 4)
	f.mustAttempt(f.request(f.loopDir, 1, "d-1", nil))
	original, _ := os.ReadFile(f.policy)
	changed, _ := json.Marshal(model.RoutePolicy{Version: model.RoutePolicyVersionV1, PermissionClasses: []string{model.PermissionReadOnly}})
	flip := func(b []byte) {
		if err := os.WriteFile(f.policy, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	flip(changed)
	_, err := f.attempt(f.request(f.loopDir, 2, "d-2", nil))
	wantErr(t, err, string(engine.ReasonLeasePolicyChanged))
	flip(original)
	f.approveCore("U1", model.StageImplementationAuthorization, model.StageRouteSelection, "rewind")
	f.authorizeRoute()
	two := f.newLoop("vl-2", nil)
	req := f.request(two, 1, "d-3", nil)
	if got := f.mustAttempt(req); !got.LaunchEligible {
		t.Fatalf("new generation: %+v", got)
	}
	flip(changed)
	_, err = f.attempt(f.request(two, 2, "d-4", nil))
	wantErr(t, err, string(engine.ReasonLeasePolicyChanged))
	if f.count(hoststate.KindPolicyRevocation) != 2 {
		t.Fatalf("revocations: %d", f.count(hoststate.KindPolicyRevocation))
	}
	flip(original)
	if got, err := f.attemptRaw(req); err != nil || got.LaunchEligible {
		t.Fatalf("second generation revived: %+v %v", got, err)
	}
}

// Liz D4 r4 F-01: an imported journal with no admission closes the gate
// until a reconciliation, which records it as a BLOCK.
func TestUnadmittedJournalClosesGate(t *testing.T) {
	f := newAttemptFixture(t, 3)
	state := writeLoopState(t, f.store, "run", "U1", []string{"true"}, "green", 0)
	if err := runLoopEvidence(t, f.store, f.root, f.base, state); err != nil {
		t.Fatal(err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	if err := requireReconciledLaunches(f.store, "run", "U1", f.policy, h); err == nil || err.Error() != hoststate.ReasonReconciliationRequired {
		t.Fatalf("gate before reconciliation: %v", err)
	}
	res, err := f.reconcile()
	if err != nil || len(res.Reconciliation.Unauthorized) == 0 {
		t.Fatalf("reconcile: %+v %v", res.Reconciliation, err)
	}
	if err := f.gate(); err == nil {
		t.Fatal("gate opened after unadmitted journal")
	}
}

// Lennon decision 2026-10-19 (Liz D4 r2 open question): the gate needs the
// current policy and refuses when it is not the one the latest
// reconciliation used; reconciling under the new policy reopens it.
func TestGateRequiresReconciledPolicy(t *testing.T) {
	f := newAttemptFixture(t, 3)
	f.mustAttempt(f.request(f.loopDir, 1, "not-launched", nil))
	if _, err := f.reconcile(); err != nil {
		t.Fatal(err)
	}
	if err := f.gate(); err != nil {
		t.Fatalf("control: %v", err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	if err := requireReconciledLaunches(f.store, "run", "U1", "", h); err == nil || err.Error() != hoststate.ReasonPolicyNotCurrent {
		t.Fatalf("no policy presented: %v", err)
	}
	changed, _ := json.Marshal(model.RoutePolicy{Version: model.RoutePolicyVersionV1, PermissionClasses: []string{model.PermissionReadOnly}})
	if err := os.WriteFile(f.policy, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := f.gate(); err == nil || err.Error() != hoststate.ReasonPolicyNotCurrent {
		t.Fatalf("policy drift: %v", err)
	}
	if _, err := f.reconcile(); err != nil {
		t.Fatal(err)
	}
	if err := f.gate(); err != nil {
		t.Fatalf("after reconciling the new policy: %v", err)
	}
}
