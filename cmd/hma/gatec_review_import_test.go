package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/transition"
)

// reviewFixture is a run with a delegated implementation, an available review
// preflight, and an HMA-generated review packet.
type reviewFixture struct {
	t                         *testing.T
	store, root, head, diff   string
	ledger, packetPath        string
	packetSHA, rdID, runID    string
	packetNonce, reviewWorker string
	launchHead, launchWT      string
	impl, reviewer            hoststate.Route
	implementation            string
	final                     bool
	reason                    hoststate.IssuingReason
	importTS                  string
	base, packetTS            string
	policy, planDigest        string
}

// step runs one host command with v as input and returns the new host head.
func (f *reviewFixture) step(cmd string, v any) string {
	f.t.Helper()
	input := filepath.Join(f.t.TempDir(), "host.json")
	writeJSON(f.t, input, v)
	if err := run([]string{"host", cmd, "--input", input, "--store", f.store}); err != nil {
		f.t.Fatalf("%s: %v", cmd, err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	return h[len(h)-1].HeadAnchor
}

func fxEnv(ts string) hostEnvelope { return hostEnvelope{RunID: "run", Actor: "fury", Timestamp: ts} }

// implement records one implementation dispatch on the delegated route.
func (f *reviewFixture) implement(preflightTS, dispatchTS, expires string) {
	f.t.Helper()
	pf := f.step("preflight", hostPreflightInput{hostEnvelope: fxEnv(preflightTS), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: f.impl, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-impl", ExpiresAt: expires,
	}})
	f.implementation = f.step("dispatch", hostDispatchInput{hostEnvelope: fxEnv(dispatchTS), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: f.impl, PreflightRecordDigest: pf,
		BoundedTask: "change one bounded unit", WorkContextDigest: "ctx-jax",
	}})
}

// packet records a review preflight and an HMA-generated review packet for
// the current implementation, with a new dispatch run id.
func (f *reviewFixture) packet(preflightTS, packetTS, expires, runID string) {
	f.t.Helper()
	rpf := f.step("preflight", hostPreflightInput{hostEnvelope: fxEnv(preflightTS), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: f.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: expires,
	}})
	f.runID = runID
	f.packetPath = filepath.Join(f.t.TempDir(), "packet.md")
	input := filepath.Join(f.t.TempDir(), "packet.json")
	writeJSON(f.t, input, hostReviewPacketInput{hostEnvelope: fxEnv(packetTS), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: f.reviewer, PreflightRecordDigest: rpf,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: f.implementation,
		ArtifactDigest: hoststate.ReviewArtifactDigest(f.head, f.diff, ""), Risk: model.RouteRiskHigh, FinalValidation: f.final,
	}, DispatchRunID: runID, IssuingReason: f.reason, RepositoryRoot: f.root, BaseRevision: f.base, Task: "Review the change.", LedgerDir: f.ledger})
	f.packetTS = packetTS
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", f.packetPath}); err != nil {
		f.t.Fatalf("review-packet: %v", err)
	}
	h, _ := hoststate.NewStore(f.store).Load("run")
	p := h[len(h)-1].Dispatch
	f.packetSHA, f.rdID, f.packetNonce = p.ReviewPacketDigest, p.Packet.ReviewDispatchID, p.Packet.ReviewNonce
	f.launchHead, f.launchWT = p.Packet.ExpectedLaunchHead, p.Packet.ExpectedLaunchWorktree
	b, _ := os.ReadFile(f.packetPath)
	if sum := sha256.Sum256(b); hex.EncodeToString(sum[:]) != f.packetSHA || !strings.Contains(string(b), f.packetNonce) {
		f.t.Fatal("packet file does not match its recorded digest and nonce")
	}
}

func newReviewFixture(t *testing.T, reason hoststate.IssuingReason, final bool) *reviewFixture {
	t.Helper()
	f := &reviewFixture{t: t, store: t.TempDir(), ledger: t.TempDir(), reviewWorker: "ming", final: final, reason: reason}
	f.root, f.base, f.head, f.diff = ciRepo(t)
	f.planDigest, f.policy = retainReviewPlan(t, f.store)
	seedRouteAuthorizationPlan(t, f.store, "run", "U1", "approved-route", f.base, f.planDigest)
	bindRepository(t, f.store, "run", f.root, "repo")
	f.impl = cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	f.reviewer = cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	f.step("decision", hostDecisionInput{hostEnvelope: fxEnv("2026-09-16T12:00:00Z"), Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "dispatch plus execution plus review", Rationale: "delegation is cheaper",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}, WorkContextDigest: "ctx-fury", Route: f.impl,
	}})
	f.implement("2026-09-16T12:01:00Z", "2026-09-16T12:02:00Z", "2026-09-16T12:06:00Z")
	f.packet("2026-09-16T12:03:00Z", "2026-09-16T12:04:00Z", "2026-09-16T12:08:00Z", "20261016T000000Z-ming-review-1")
	return f
}

// report writes an AUDIT.json bound to the packet; edit may change it first.
func (f *reviewFixture) report(verdict, isolation string, edit func(map[string]any)) string {
	f.t.Helper()
	r := map[string]any{"audit_id": "a", "target": "t", "modes": []any{"code"}, "phase1_frozen": true, "limitations": []any{},
		"verdict": verdict, "isolation": isolation, "findings": []any{},
		"requirements": []any{map[string]any{"id": "R1", "status": "pass", "evidence_refs": []any{"e1"}}},
		"hma_binding":  map[string]any{"packet_sha256": f.packetSHA, "review_dispatch_id": f.rdID, "review_nonce": f.packetNonce}}
	if edit != nil {
		edit(r)
	}
	path := filepath.Join(f.t.TempDir(), "AUDIT.json")
	b, _ := json.Marshal(r)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		f.t.Fatal(err)
	}
	return path
}

// receipt writes the agent-dispatch receipt for the review run; edit may
// change it first.
func (f *reviewFixture) receipt(reportPath string, edit func(map[string]any)) {
	f.t.Helper()
	b, _ := os.ReadFile(reportPath)
	sum := sha256.Sum256(b)
	ts, _ := time.Parse(time.RFC3339, f.packetTS)
	r := map[string]any{"run_id": f.runID, "agent": f.reviewWorker, "model": "sonnet", "provider": "anthropic", "effort": "medium",
		"started_at": ts.Add(30 * time.Second).Format(time.RFC3339), "project_root": f.root,
		"mode": "review", "state": "completed", "exit_code": 0,
		"packet_sha256": f.packetSHA, "launch_head": f.launchHead, "launch_worktree_sha256": f.launchWT,
		"report_files": []any{map[string]any{"path": reportPath, "sha256": hex.EncodeToString(sum[:])}}}
	if edit != nil {
		edit(r)
	}
	dir := filepath.Join(f.ledger, "runs", f.runID)
	os.MkdirAll(dir, 0o700)
	out, _ := json.Marshal(r)
	if err := os.WriteFile(filepath.Join(dir, "receipt.json"), out, 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *reviewFixture) importReport(reportPath string) error {
	f.t.Helper()
	input := filepath.Join(f.t.TempDir(), "import.json")
	ts := f.importTS
	if ts == "" {
		ts = "2026-09-16T12:06:00Z"
	}
	writeJSON(f.t, input, hostReviewImportInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: ts},
		ReviewDispatchID: f.rdID, LedgerDir: f.ledger, ReportPath: reportPath})
	return run([]string{"host", "review-import", "--input", input, "--store", f.store})
}

// gate reconciles the run's ledgers (as a host must before a gated
// transition, Gate D D4) and then checks the review gate.
func (f *reviewFixture) gate() error {
	f.t.Helper()
	input := filepath.Join(f.t.TempDir(), "reconcile.json")
	writeJSON(f.t, input, hostReconcileInput{hostEnvelope: fxEnv("2026-09-16T12:09:00Z"), PolicyPath: f.policy})
	if _, err := captureStdout(f.t, func() error { return run([]string{"host", "reconcile", "--input", input, "--store", f.store}) }); err != nil {
		return err
	}
	return f.gateOnly()
}

// retainReviewPlan stores the accepted plan of the review fixture's lease
// generation (U1, two hours of wall time from the route approval) and
// writes its route policy; it returns the plan digest and policy path.
func retainReviewPlan(t *testing.T, storePath string) (string, string) {
	t.Helper()
	pol := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
	proj, _ := model.NormalizeRoutePolicy(pol)
	policyDigest, _ := engine.RoutePolicyDigest(proj)
	all := []string{model.PermissionReadOnly, model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	p := model.Plan{Schema: model.PlanSchema, RunID: "run", AcceptanceContractDigest: "contract", PolicyDigest: policyDigest,
		Units: []model.PlanUnit{{ID: "U1", Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"a"}, AllowedActions: []string{"edit"},
			ForbiddenFiles: []string{"docs"}, ForbiddenActions: []string{"push"}, PreChangeEvidence: "failing test", MinimumImplementation: "m",
			ReuseAnalysis: "r", PermissionClasses: all, ImplementerRoute: "impl", EscalationRoute: "esc", ValidatorRoute: "val",
			IndependenceRequirement: "family", VerificationCommands: []model.Check{{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}},
			RequiredEvidence: []string{"go test"}, StopConditions: []string{"s"}, RollbackConditions: []string{"r"}, EscalationConditions: []string{"e"},
			NonGoals: []string{"n"}, Dependencies: []string{},
			Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 100, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 120, CostCeiling: 10}}}}
	pb, _ := json.Marshal(p)
	plan, digest, err := model.ParsePlan(pb)
	if err != nil {
		t.Fatal(err)
	}
	if err := retainPlan(storePath, plan, digest); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "policy.json")
	b, _ := json.Marshal(pol)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return digest, path
}

// gateOnly checks the review gate without reconciling first.
func (f *reviewFixture) gateOnly() error {
	return requireHostTransitionGate(f.store, transition.Input{RunID: "run", PolicyPath: f.policy, ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
		ProducedHeadDigest: f.head, DiffDigest: f.diff,
	}})
}

func (f *reviewFixture) records() int {
	h, err := hoststate.NewStore(f.store).Load("run")
	if err != nil {
		f.t.Fatal(err)
	}
	return len(h)
}

func TestGateCVerdictMapIsTotal(t *testing.T) {
	for _, tc := range []struct {
		verdict string
		want    hoststate.ReviewVerdict
	}{
		{"pass", hoststate.ReviewApprove},
		{"fix-required", hoststate.ReviewRequestChanges},
		{"conditional-pass", hoststate.ReviewEscalate},
		{"blocked", hoststate.ReviewEscalate},
		{"not-evaluated", hoststate.ReviewEscalate},
		{"unknown", hoststate.ReviewEscalate},
		{"", hoststate.ReviewEscalate},
	} {
		t.Run("verdict-"+tc.verdict, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			rp := f.report(tc.verdict, "isolated", nil)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err != nil {
				t.Fatalf("import: %v", err)
			}
			h, _ := hoststate.NewStore(f.store).Load("run")
			if got := h[len(h)-1].Review.Verdict; got != tc.want {
				t.Fatalf("verdict %s, want %s", got, tc.want)
			}
			if err := f.gate(); err == nil || err.Error() != hoststate.ReasonFindingsSnapshotMissing && tc.want == hoststate.ReviewApprove {
				t.Fatalf("gate before findings snapshot: %v", err)
			}
			if err := f.importFindings(h[len(h)-1].HeadAnchor, rp, "2026-09-16T12:06:30Z"); err != nil {
				t.Fatalf("findings import: %v", err)
			}
			gateErr := f.gate()
			if (tc.want == hoststate.ReviewApprove) != (gateErr == nil) {
				t.Fatalf("gate after %s: %v", tc.want, gateErr)
			}
		})
	}
}

func TestGateCMissingVerdictEscalates(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("", "isolated", func(r map[string]any) { delete(r, "verdict") })
	f.receipt(rp, nil)
	if err := f.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := f.gate(); err == nil {
		t.Fatal("report without verdict satisfied the review gate")
	}
}

// Every failed provenance check refuses and appends nothing.
func TestGateCImportRefusals(t *testing.T) {
	cases := []struct {
		name    string
		report  func(map[string]any)
		receipt func(map[string]any)
		want    string
	}{
		{"old-report-other-nonce", func(r map[string]any) {
			r["hma_binding"].(map[string]any)["review_nonce"] = strings.Repeat("0", 32)
		}, nil, hoststate.ReasonReviewReportMismatch},
		{"no-binding", func(r map[string]any) { delete(r, "hma_binding") }, nil, hoststate.ReasonReviewReportMismatch},
		{"bad-isolation", func(r map[string]any) { r["isolation"] = "none" }, nil, hoststate.ReasonReviewReportMismatch},
		{"write-mode", nil, func(r map[string]any) { r["mode"] = "write" }, hoststate.ReasonReviewReceiptMismatch},
		{"not-completed", nil, func(r map[string]any) { r["state"] = "failed" }, hoststate.ReasonReviewReceiptMismatch},
		{"nonzero-exit", nil, func(r map[string]any) { r["exit_code"] = 1 }, hoststate.ReasonReviewReceiptMismatch},
		{"no-exit-code", nil, func(r map[string]any) { delete(r, "exit_code") }, hoststate.ReasonReviewReceiptMismatch},
		{"other-packet", nil, func(r map[string]any) { r["packet_sha256"] = strings.Repeat("a", 64) }, hoststate.ReasonReviewReceiptMismatch},
		{"other-head", nil, func(r map[string]any) { r["launch_head"] = strings.Repeat("b", 40) }, hoststate.ReasonReviewReceiptMismatch},
		{"other-worktree", nil, func(r map[string]any) { r["launch_worktree_sha256"] = strings.Repeat("c", 64) }, hoststate.ReasonReviewReceiptMismatch},
		{"other-agent", nil, func(r map[string]any) { r["agent"] = "jax" }, hoststate.ReasonReviewReceiptMismatch},
		{"other-model", nil, func(r map[string]any) { r["model"] = "luna" }, hoststate.ReasonReviewReceiptMismatch},
		{"other-run", nil, func(r map[string]any) { r["run_id"] = "another-run" }, hoststate.ReasonReviewReceiptMismatch},
		{"report-not-listed", nil, func(r map[string]any) { r["report_files"] = []any{} }, hoststate.ReasonReviewReportMismatch},
		{"report-rewritten", nil, func(r map[string]any) {
			r["report_files"] = []any{map[string]any{"path": r["report_files"].([]any)[0].(map[string]any)["path"], "sha256": strings.Repeat("d", 64)}}
		}, hoststate.ReasonReviewReportMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			rp := f.report("pass", "isolated", tc.report)
			f.receipt(rp, tc.receipt)
			before := f.records()
			err := f.importReport(rp)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %s", err, tc.want)
			}
			if f.records() != before {
				t.Fatal("refused import appended a record")
			}
			if f.gate() == nil {
				t.Fatal("review gate satisfied after refused import")
			}
		})
	}
}

// A reviewer that completed without writing a report imports nothing.
func TestGateCCompletedReviewerWithoutReportImportsNothing(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	missing := filepath.Join(t.TempDir(), "AUDIT.json")
	f.receipt(f.report("pass", "isolated", nil), func(r map[string]any) { r["report_files"] = []any{} })
	if err := f.importReport(missing); err == nil || err.Error() != hoststate.ReasonReviewReportMismatch {
		t.Fatalf("err = %v", err)
	}
	if f.gate() == nil {
		t.Fatal("gate satisfied without a report")
	}
}

// The worktree changed after the packet: the live repository no longer
// matches the packet's artifact (A02), and the receipt's launch digest no
// longer equals the expected one.
func TestGateCWorktreeChangedAfterPacketRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, func(r map[string]any) {
		os.WriteFile(filepath.Join(f.root, "late"), []byte("x"), 0o600)
		_, wt, _ := launchDigestsFor(f.root)
		r["launch_worktree_sha256"] = wt
	})
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewTargetChanged {
		t.Fatalf("err = %v", err)
	}
	// The worktree restored, the receipt still names the changed launch.
	os.Remove(filepath.Join(f.root, "late"))
	if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewReceiptMismatch {
		t.Fatalf("restored worktree, stale receipt: err = %v", err)
	}
}

func TestGateCFinalNeedsIsolatedReport(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reason hoststate.IssuingReason
		final  bool
	}{{"final-audit", hoststate.IssueFinal, false}, {"final-validation", hoststate.IssuePerChange, true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewFixture(t, tc.reason, tc.final)
			rp := f.report("pass", "partial", nil)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err == nil || err.Error() != hoststate.ReasonReviewIsolationInsufficient {
				t.Fatalf("err = %v", err)
			}
			rp = f.report("pass", "isolated", nil)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err != nil {
				t.Fatalf("isolated final import: %v", err)
			}
		})
	}
}

// An HMA-generated packet cannot be answered by a host-written review, with
// or without forged provenance fields.
func TestGateCHandWrittenReviewRefusedForHMAPacket(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	d := h[len(h)-1]
	base := hoststate.IndependentReview{UnitID: "U1", ImplementationRecordDigest: d.Dispatch.ImplementationRecordDigest, ReviewDispatchDigest: d.HeadAnchor,
		ArtifactDigest: d.Dispatch.ArtifactDigest, ReviewPacketDigest: d.Dispatch.ReviewPacketDigest, ReviewerID: "ming", Verdict: hoststate.ReviewApprove}
	forged := base
	forged.ReceiptDigest, forged.ReportDigest, forged.Isolation, forged.ReportVerdict = "sha256:x", "sha256:y", "isolated", "pass"
	for name, r := range map[string]hoststate.IndependentReview{"plain": base, "forged": forged} {
		input := filepath.Join(t.TempDir(), "review.json")
		writeJSON(t, input, hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:05:00Z"}, Review: r})
		err := run([]string{"host", "review", "--input", input, "--store", f.store})
		if err == nil || err.Error() != hoststate.ReasonReviewImportRequired {
			t.Fatalf("%s: err = %v, want %s", name, err, hoststate.ReasonReviewImportRequired)
		}
	}
}

// A packet written inside the repository would change the worktree the
// receipt describes.
func TestGateCPacketInsideRepositoryRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	d := *h[len(h)-1].Dispatch
	d.Packet, d.ReviewPacketDigest = nil, ""
	input := filepath.Join(t.TempDir(), "in.json")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:04:30Z"},
		Dispatch: d, DispatchRunID: "other-run", IssuingReason: hoststate.IssuePerChange, RepositoryRoot: f.root, BaseRevision: f.base, Task: "x", LedgerDir: f.ledger})
	out := filepath.Join(f.root, "packet.md")
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", out}); err == nil {
		t.Fatal("packet inside repository accepted")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("packet file left inside repository")
	}
}

// A second packet may not reuse the dispatch run id of the first.
func TestGateCPacketRunIDReuseRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	d := *h[len(h)-1].Dispatch
	d.Packet, d.ReviewPacketDigest = nil, ""
	input := filepath.Join(t.TempDir(), "in.json")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:04:30Z"},
		Dispatch: d, DispatchRunID: f.runID, IssuingReason: hoststate.IssuePerChange, RepositoryRoot: f.root, BaseRevision: f.base, Task: "x", LedgerDir: f.ledger})
	out := filepath.Join(t.TempDir(), "packet2.md")
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", out}); err == nil || err.Error() != hoststate.ReasonReviewPacketMismatch {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("refused packet file left behind")
	}
}

func launchDigestsFor(root string) (string, string, error) {
	return repostate.LaunchDigests(context.Background(), root)
}
