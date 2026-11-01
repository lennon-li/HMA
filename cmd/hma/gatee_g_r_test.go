package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// contractBoundRun is a repository plus a core chain that has accepted an
// acceptance contract at a given risk and approved U1's route, through
// StageImplementationAuthorization. It mirrors gated_credit_test.go's
// newCreditFixtureArgs seeding, trimmed to what Gate E's G-R tests need: a
// bound contract whose Risk field is the review-risk floor on the packet
// path (cmd/hma/review.go's appendHostRecord closure).
type contractBoundRun struct {
	store, root, base, head, diff string
	contractPath, policy          string
}

func seedContractBoundRun(t *testing.T, risk model.RouteRiskLevel) *contractBoundRun {
	t.Helper()
	cb := &contractBoundRun{store: t.TempDir()}
	cb.root, cb.base, cb.head, cb.diff = ciRepo(t)

	pol := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
	proj, _ := model.NormalizeRoutePolicy(pol)
	policyDigest, _ := engine.RoutePolicyDigest(proj)
	all := []string{model.PermissionReadOnly, model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	check := model.Check{Executable: "true", Argv: []string{}, WorkingDir: "."}
	contract := model.AcceptanceContract{Schema: model.AcceptanceContractSchema, RunID: "run", Intent: "one behavior", NonGoals: []string{"none"},
		Criteria: []model.ContractCriterion{
			{ID: "C1", Statement: "check passes", Threshold: "exit 0", EvidenceRequirement: "fresh capture", Check: check}},
		Risk: risk, Deliverables: []string{"code"},
		Budgets:         model.ContractBudgets{MaxAttempts: 10, MaxWallMinutes: 120, CostCeiling: 20, CostUnit: "usd"},
		PermittedRoutes: []string{"impl", "esc", "val"}, PermissionEnvelope: all, PolicyDigest: policyDigest}
	contractBytes, _ := json.Marshal(contract)
	_, contractDigest, err := model.ParseAcceptanceContract(contractBytes)
	if err != nil {
		t.Fatal(err)
	}
	cb.contractPath = filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(cb.contractPath, contractBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	plan := model.Plan{Schema: model.PlanSchema, RunID: "run", AcceptanceContractDigest: contractDigest, PolicyDigest: policyDigest,
		Units: []model.PlanUnit{{ID: "U1", Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"a"}, AllowedActions: []string{"edit"},
			ForbiddenFiles: []string{"docs"}, ForbiddenActions: []string{"push"}, PreChangeEvidence: "failing test", MinimumImplementation: "m",
			ReuseAnalysis: "r", PermissionClasses: all, ImplementerRoute: "impl", EscalationRoute: "esc", ValidatorRoute: "val",
			IndependenceRequirement: "family", VerificationCommands: []model.Check{check},
			RequiredEvidence: []string{"check"}, StopConditions: []string{"s"}, RollbackConditions: []string{"r"}, EscalationConditions: []string{"e"},
			NonGoals: []string{"n"}, Dependencies: []string{},
			Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 100, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 120, CostCeiling: 10}}}}
	planBytes, _ := json.Marshal(plan)
	parsedPlan, planDigest, err := model.ParsePlan(planBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := retainPlan(cb.store, parsedPlan, planDigest); err != nil {
		t.Fatal(err)
	}
	cb.policy = filepath.Join(t.TempDir(), "policy.json")
	policyBytes, _ := json.Marshal(pol)
	if err := os.WriteFile(cb.policy, policyBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	s := store.New(cb.store)
	seq := int64(0)
	add := func(r model.Record) {
		seq++
		recs, _ := s.Load("run")
		r.Version = 1
		r.Metadata = model.RecordMetadata{RunID: "run", Sequence: seq, Timestamp: "2026-09-16T11:00:00Z", Actor: "operator-confirmed:lennon"}
		if len(recs) > 0 {
			r.Metadata.PredecessorHash = recs[len(recs)-1].Metadata.HeadAnchor
		}
		if err := s.Append(&r); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	approve := func(unit string, from, to model.Stage, digest string) {
		add(model.Record{UnitID: unit, Kind: model.KindStageTransition, Stage: from, Control: model.ControlApproved,
			Approval: &model.ApprovalBinding{UnitID: unit, RunID: "run", TransitionDigest: digest, CurrentStage: from, ProposedTargetStage: to,
				RepositoryIdentityDigest: "repo", BaseRevisionDigest: cb.base, StageTimeDigest: "head:1", AcceptedPlanDigest: planDigest,
				AcceptanceContractDigest: contractDigest, ChallengeNonce: fmt.Sprintf("n%d", seq), Approver: "human", Timestamp: "2026-09-16T11:00:00Z"}})
	}
	add(model.Record{Kind: model.KindCriterion, Stage: model.StageAcceptanceCriteria,
		Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}})
	approve("", model.StageAcceptanceCriteria, model.StagePlanning, "contract")
	approve("", model.StagePlanning, model.StageRouteSelection, "plan")
	approve("U1", model.StageRouteSelection, model.StageImplementationAuthorization, "approved-route")

	bindRepository(t, cb.store, "run", cb.root, "repo")
	return cb
}

// cbStep runs one host command with v as input against cb.store and returns
// the new host head, exactly like reviewFixture.step.
func cbStep(t *testing.T, cb *contractBoundRun, cmd string, v any) string {
	t.Helper()
	input := filepath.Join(t.TempDir(), "host.json")
	writeJSON(t, input, v)
	if err := run([]string{"host", cmd, "--input", input, "--store", cb.store}); err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	h, _ := hoststate.NewStore(cb.store).Load("run")
	return h[len(h)-1].HeadAnchor
}

// cbImplement delegates and dispatches the implementation for U1 on cb.
func cbImplement(t *testing.T, cb *contractBoundRun, impl hoststate.Route) string {
	t.Helper()
	cbStep(t, cb, "decision", hostDecisionInput{hostEnvelope: fxEnv("2026-09-16T12:00:00Z"), Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "dispatch plus execution plus review", Rationale: "delegation is cheaper",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}, WorkContextDigest: "ctx-fury", Route: impl,
	}})
	implPreflight := cbStep(t, cb, "preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:01:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-impl", ExpiresAt: "2026-09-16T12:06:00Z",
	}})
	return cbStep(t, cb, "dispatch", hostDispatchInput{hostEnvelope: fxEnv("2026-09-16T12:02:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, PreflightRecordDigest: implPreflight,
		BoundedTask: "change one bounded unit", WorkContextDigest: "ctx-jax",
	}})
}

// TestReviewPacketRiskBelowContractRefused is the G-R red test (§5.2): a
// contract-bound run whose contract.Risk is high, dispatched a review packet
// at risk low. HMA must refuse rather than silently upgrading the host's
// declared risk (§3.5 E0 step 4).
func TestReviewPacketRiskBelowContractRefused(t *testing.T) {
	cb := seedContractBoundRun(t, model.RouteRiskHigh)
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	implementation := cbImplement(t, cb, impl)

	before, _ := hoststate.NewStore(cb.store).Load("run")
	input := filepath.Join(t.TempDir(), "packet.json")
	packetPath := filepath.Join(t.TempDir(), "packet.md")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:04:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, ImplementationRecordDigest: implementation,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming",
		ArtifactDigest: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""), Risk: model.RouteRiskLow,
	}, DispatchRunID: "20261016T000000Z-ming-review-1", IssuingReason: hoststate.IssuePerChange,
		RepositoryRoot: cb.root, BaseRevision: cb.base, Task: "Review the change.",
		ContractPath: cb.contractPath, PolicyPath: cb.policy, LedgerDir: t.TempDir()})

	err := run([]string{"host", "review-packet", "--input", input, "--store", cb.store, "--out", packetPath})
	if err == nil || err.Error() != hoststate.ReasonReviewRiskBelowRequired {
		t.Fatalf("review-packet at risk below contract: got %v, want %s", err, hoststate.ReasonReviewRiskBelowRequired)
	}
	after, _ := hoststate.NewStore(cb.store).Load("run")
	if len(after) != len(before) {
		t.Fatalf("refused dispatch appended: before %d after %d", len(before), len(after))
	}
	if _, statErr := os.Stat(packetPath); !os.IsNotExist(statErr) {
		t.Fatalf("packet file not removed on refusal: %v", statErr)
	}
}

// TestHostDispatchReviewOnContractRunRefused is the G-R test for the second
// EvaluateDispatch caller (§3.5 E0 step 5, Decision D10): a contract-bound
// run cannot take a packet-less review dispatch through hma host dispatch,
// even when no prior packet has been issued for the run (packetMode would
// otherwise be false and EvaluateDispatch would not itself refuse it).
func TestHostDispatchReviewOnContractRunRefused(t *testing.T) {
	cb := seedContractBoundRun(t, model.RouteRiskLow)
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	implementation := cbImplement(t, cb, impl)
	reviewPreflight := cbStep(t, cb, "preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:03:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:08:00Z",
	}})

	before, _ := hoststate.NewStore(cb.store).Load("run")
	input := filepath.Join(t.TempDir(), "review-dispatch.json")
	writeJSON(t, input, hostDispatchInput{hostEnvelope: fxEnv("2026-09-16T12:04:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, PreflightRecordDigest: reviewPreflight,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: implementation,
		ArtifactDigest: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""), ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}})
	err := run([]string{"host", "dispatch", "--input", input, "--store", cb.store})
	if err == nil || err.Error() != hoststate.ReasonReviewImportRequired {
		t.Fatalf("packet-less review dispatch on contract-bound run: got %v, want %s", err, hoststate.ReasonReviewImportRequired)
	}
	after, _ := hoststate.NewStore(cb.store).Load("run")
	if len(after) != len(before) {
		t.Fatalf("refused dispatch appended: before %d after %d", len(before), len(after))
	}
}

// TestReviewPacketNonContractRunRiskCheckSkipped confirms the G-R risk check
// (cmd/hma/review.go's appendHostRecord closure) does not run at all for a
// run with no bound acceptance contract: a review-packet dispatch at risk
// low succeeds exactly as it did before this phase.
func TestReviewPacketNonContractRunRiskCheckSkipped(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	before := f.records()
	rpf := f.step("preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:05:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: f.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review-2", ExpiresAt: "2026-09-16T12:10:00Z",
	}})
	input := filepath.Join(t.TempDir(), "packet2.json")
	packetPath := filepath.Join(t.TempDir(), "packet2.md")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:06:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: f.reviewer, PreflightRecordDigest: rpf, ImplementationRecordDigest: f.implementation,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming-2",
		ArtifactDigest: hoststate.ReviewArtifactDigest(f.head, f.diff, ""), Risk: model.RouteRiskLow,
	}, DispatchRunID: "20261016T000000Z-ming-review-2", IssuingReason: f.reason, RepositoryRoot: f.root, BaseRevision: f.base,
		Task: "Review the change.", LedgerDir: f.ledger})
	if err := run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", packetPath}); err != nil {
		t.Fatalf("review-packet at risk low on non-contract run: %v", err)
	}
	if after := f.records(); after != before+2 {
		t.Fatalf("got %d new records, want 2 (preflight, dispatch)", after-before)
	}
}
