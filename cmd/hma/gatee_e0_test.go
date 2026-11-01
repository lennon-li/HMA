package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// checkpointRun is a contractBoundRun (gatee_g_r_test.go) extended with an
// optional checkpoint policy document and an option to advance the chain to
// INDEPENDENT_VALIDATION, so Gate E Phase 4a's classifier wiring in
// runHostReviewPacket (cmd/hma/review.go) can be exercised at both stages.
type checkpointRun struct {
	*contractBoundRun
	checkpointPolicyPath   string
	contractDigest         string
	checkpointPolicyDigest string
}

// seedCheckpointBoundRun seeds a contract-bound run for U1 exactly like
// seedContractBoundRun, except: the contract optionally binds a checkpoint
// policy (policy == nil means none, so every changed path is unclassified
// per architecture §16), the unit is optionally a material milestone, and,
// when atIV is true, three more approved transitions carry the run's
// current stage to INDEPENDENT_VALIDATION (engine.ProjectStageState reads
// the latest approved transition's target regardless of unit, so this is
// sufficient to exercise the classifier's stage-dependent rules without a
// real checkpoint/final review cycle, which is out of scope for E0).
func seedCheckpointBoundRun(t *testing.T, risk model.RouteRiskLevel, policy *model.CheckpointPolicy, milestone, atIV bool) *checkpointRun {
	t.Helper()
	cb := &checkpointRun{contractBoundRun: &contractBoundRun{store: t.TempDir()}}
	cb.root, cb.base, cb.head, cb.diff = ciRepo(t)

	pol := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
	proj, _ := model.NormalizeRoutePolicy(pol)
	policyDigest, _ := engine.RoutePolicyDigest(proj)
	all := []string{model.PermissionReadOnly, model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	check := model.Check{Executable: "true", Argv: []string{}, WorkingDir: "."}

	checkpointPolicyDigest := ""
	if policy != nil {
		pb, err := json.Marshal(*policy)
		if err != nil {
			t.Fatal(err)
		}
		_, digest, err := model.ParseCheckpointPolicy(pb)
		if err != nil {
			t.Fatal(err)
		}
		checkpointPolicyDigest = digest
		cb.checkpointPolicyPath = filepath.Join(t.TempDir(), "checkpoint-policy.json")
		if err := os.WriteFile(cb.checkpointPolicyPath, pb, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cb.checkpointPolicyDigest = checkpointPolicyDigest

	contract := model.AcceptanceContract{Schema: model.AcceptanceContractSchema, RunID: "run", Intent: "one behavior", NonGoals: []string{"none"},
		Criteria: []model.ContractCriterion{
			{ID: "C1", Statement: "check passes", Threshold: "exit 0", EvidenceRequirement: "fresh capture", Check: check}},
		Risk: risk, Deliverables: []string{"code"},
		Budgets:         model.ContractBudgets{MaxAttempts: 10, MaxWallMinutes: 120, CostCeiling: 20, CostUnit: "usd"},
		PermittedRoutes: []string{"impl", "esc", "val"}, PermissionEnvelope: all, PolicyDigest: policyDigest,
		CheckpointPolicyDigest: checkpointPolicyDigest}
	contractBytes, err := json.Marshal(contract)
	if err != nil {
		t.Fatal(err)
	}
	_, contractDigest, err := model.ParseAcceptanceContract(contractBytes)
	if err != nil {
		t.Fatal(err)
	}
	cb.contractDigest = contractDigest
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
			NonGoals: []string{"n"}, Dependencies: []string{}, MaterialMilestone: milestone,
			Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 100, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 120, CostCeiling: 10}}}}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	parsedPlan, planDigest, err := model.ParsePlan(planBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := retainPlan(cb.store, parsedPlan, planDigest); err != nil {
		t.Fatal(err)
	}
	cb.policy = filepath.Join(t.TempDir(), "policy.json")
	policyBytes, err := json.Marshal(pol)
	if err != nil {
		t.Fatal(err)
	}
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
		a := &model.ApprovalBinding{UnitID: unit, RunID: "run", TransitionDigest: digest, CurrentStage: from, ProposedTargetStage: to,
			RepositoryIdentityDigest: "repo", BaseRevisionDigest: cb.base, StageTimeDigest: "head:1", AcceptedPlanDigest: planDigest,
			AcceptanceContractDigest: contractDigest, ChallengeNonce: fmt.Sprintf("n%d", seq), Approver: "human", Timestamp: "2026-09-16T11:00:00Z"}
		if model.ApprovalBindsHeadDiff(from) {
			a.ProducedHeadDigest, a.DiffDigest = cb.head, cb.diff
		}
		add(model.Record{UnitID: unit, Kind: model.KindStageTransition, Stage: from, Control: model.ControlApproved, Approval: a})
	}
	add(model.Record{Kind: model.KindCriterion, Stage: model.StageAcceptanceCriteria,
		Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}}})
	approve("", model.StageAcceptanceCriteria, model.StagePlanning, "contract")
	approve("", model.StagePlanning, model.StageRouteSelection, "plan")
	approve("U1", model.StageRouteSelection, model.StageImplementationAuthorization, "approved-route")
	if atIV {
		approve("U1", model.StageImplementationAuthorization, model.StageImplementationReview, "impl-review")
		approve("U1", model.StageImplementationReview, model.StageVerification, "verification")
		approve("U1", model.StageVerification, model.StageIndependentValidation, "iv")
	}

	bindRepository(t, cb.store, "run", cb.root, "repo")
	return cb
}

// securityPathPolicy classifies the fixture's only changed path ("a",
// ciRepo's convention) as "security", so it fires T5 and raises the risk
// floor without touching contract.Risk.
func securityPathPolicy() *model.CheckpointPolicy {
	return &model.CheckpointPolicy{Schema: model.CheckpointPolicySchema, Rules: []model.CheckpointPolicyRule{
		{ID: "sec", Glob: "a", Classes: []string{"security"}},
	}}
}

// TestCheckpointReasonTooLowRefused is §5.2's roadmap-ask test: at
// INDEPENDENT_VALIDATION with a material-milestone unit (T6), a host
// requesting issuing_reason per_change is refused; the error reports the
// derived level.
func TestCheckpointReasonTooLowRefused(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	implementation := cbImplement(t, cb.contractBoundRun, impl)

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
	if err == nil || !strings.Contains(err.Error(), hoststate.ReasonCheckpointReasonTooLow) || !strings.Contains(err.Error(), "checkpoint") {
		t.Fatalf("review-packet with reason below derived level: got %v, want %s reporting level checkpoint", err, hoststate.ReasonCheckpointReasonTooLow)
	}
	after, _ := hoststate.NewStore(cb.store).Load("run")
	if len(after) != len(before) {
		t.Fatalf("refused dispatch appended: before %d after %d", len(before), len(after))
	}
	if _, statErr := os.Stat(packetPath); !os.IsNotExist(statErr) {
		t.Fatalf("packet file not removed on refusal: %v", statErr)
	}
}

// TestReviewPacketRiskBelowFloorRefused is §5.2's roadmap-ask test: contract
// risk low, a checkpoint policy that classes the changed path "security"
// (T5), dispatch risk low. The required review risk is the raised floor,
// not the contract risk alone, so this refuses only after Phase 4a's
// floor-aware G-R check.
func TestReviewPacketRiskBelowFloorRefused(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, securityPathPolicy(), false, false)
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	implementation := cbImplement(t, cb.contractBoundRun, impl)

	before, _ := hoststate.NewStore(cb.store).Load("run")
	input := filepath.Join(t.TempDir(), "packet.json")
	packetPath := filepath.Join(t.TempDir(), "packet.md")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:04:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, ImplementationRecordDigest: implementation,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming",
		ArtifactDigest: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""), Risk: model.RouteRiskLow,
	}, DispatchRunID: "20261016T000000Z-ming-review-1", IssuingReason: hoststate.IssuePerChange,
		RepositoryRoot: cb.root, BaseRevision: cb.base, Task: "Review the change.",
		ContractPath: cb.contractPath, PolicyPath: cb.policy, CheckpointPolicyPath: cb.checkpointPolicyPath, LedgerDir: t.TempDir()})

	err := run([]string{"host", "review-packet", "--input", input, "--store", cb.store, "--out", packetPath})
	if err == nil || err.Error() != hoststate.ReasonReviewRiskBelowRequired {
		t.Fatalf("review-packet at risk below floor: got %v, want %s", err, hoststate.ReasonReviewRiskBelowRequired)
	}
	after, _ := hoststate.NewStore(cb.store).Load("run")
	if len(after) != len(before) {
		t.Fatalf("refused dispatch appended: before %d after %d", len(before), len(after))
	}
	if _, statErr := os.Stat(packetPath); !os.IsNotExist(statErr) {
		t.Fatalf("packet file not removed on refusal: %v", statErr)
	}
}

// TestCheckpointBindingFieldsPopulatedForTriggeredUnit is the basic positive
// test: packet issuance for a triggered unit (T6, material milestone, at
// INDEPENDENT_VALIDATION) populates every expected §3.5a binding field.
func TestCheckpointBindingFieldsPopulatedForTriggeredUnit(t *testing.T) {
	cb := seedCheckpointBoundRun(t, model.RouteRiskLow, nil, true, true)
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	implementation := cbImplement(t, cb.contractBoundRun, impl)
	reviewPreflight := cbStep(t, cb.contractBoundRun, "preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:03:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:08:00Z",
	}})

	input := filepath.Join(t.TempDir(), "packet.json")
	packetPath := filepath.Join(t.TempDir(), "packet.md")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:04:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, PreflightRecordDigest: reviewPreflight, ImplementationRecordDigest: implementation,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming",
		ArtifactDigest: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""), Risk: model.RouteRiskLow,
	}, DispatchRunID: "20261016T000000Z-ming-review-1", IssuingReason: hoststate.IssueCheckpoint,
		RepositoryRoot: cb.root, BaseRevision: cb.base, Task: "Review the change.",
		ContractPath: cb.contractPath, PolicyPath: cb.policy, LedgerDir: t.TempDir()})

	if err := run([]string{"host", "review-packet", "--input", input, "--store", cb.store, "--out", packetPath}); err != nil {
		t.Fatalf("review-packet: %v", err)
	}
	h, err := hoststate.NewStore(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	p := h[len(h)-1].Dispatch.Packet
	if p.CheckpointSchema != hoststate.CheckpointSchemaV1 {
		t.Fatalf("checkpoint_schema = %q, want %q", p.CheckpointSchema, hoststate.CheckpointSchemaV1)
	}
	if p.MinimumIssuingReason != hoststate.IssueCheckpoint {
		t.Fatalf("minimum_issuing_reason = %q, want checkpoint", p.MinimumIssuingReason)
	}
	wantTriggers := map[string]bool{"APPROVED_MATERIAL_MILESTONE": true, "UNCLASSIFIED": true}
	if len(p.CheckpointTriggers) != len(wantTriggers) {
		t.Fatalf("checkpoint_triggers = %v, want %v", p.CheckpointTriggers, wantTriggers)
	}
	for _, tr := range p.CheckpointTriggers {
		if !wantTriggers[tr] {
			t.Fatalf("unexpected trigger %q in %v", tr, p.CheckpointTriggers)
		}
	}
	if len(p.UnclassifiedPaths) != 1 || p.UnclassifiedPaths[0] != "a" {
		t.Fatalf("unclassified_paths = %v, want [a]", p.UnclassifiedPaths)
	}
	if p.FamilySeparation {
		t.Fatal("family_separation true, want false (T6/T8 alone set neither, Decision U)")
	}
	if p.RiskFloor != model.RouteRiskLow {
		t.Fatalf("risk_floor = %q, want low", p.RiskFloor)
	}
	if p.ContractRisk != model.RouteRiskLow {
		t.Fatalf("contract_risk = %q, want low", p.ContractRisk)
	}
	if p.BoundContractDigest != cb.contractDigest {
		t.Fatalf("bound_contract_digest = %q, want %q", p.BoundContractDigest, cb.contractDigest)
	}
	if p.CheckpointPolicyDigest != "" {
		t.Fatalf("checkpoint_policy_digest = %q, want empty (no policy)", p.CheckpointPolicyDigest)
	}
	if p.ManifestDigest == "" {
		t.Fatal("manifest_digest empty, want set")
	}
	if p.CheckpointCovers == nil || p.CheckpointCovers.ContractDigest != cb.contractDigest {
		t.Fatalf("checkpoint_covers = %+v, want contract_digest %q", p.CheckpointCovers, cb.contractDigest)
	}
	packetBytes, err := os.ReadFile(packetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(packetBytes), "checkpoint_schema: "+hoststate.CheckpointSchemaV1) {
		t.Fatal("packet header does not print checkpoint_schema")
	}
}

// TestCheckpointFieldsAbsentForNonContractRun is the basic negative test:
// packet issuance on a non-contract run never runs the classifier, so every
// checkpoint binding field, starting with checkpoint_schema, stays absent.
func TestCheckpointFieldsAbsentForNonContractRun(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, err := hoststate.NewStore(f.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	p := h[len(h)-1].Dispatch.Packet
	if p.CheckpointSchema != "" {
		t.Fatalf("checkpoint_schema = %q on a non-contract run, want absent", p.CheckpointSchema)
	}
	if p.FamilySeparation || p.RiskFloor != "" || p.CheckpointCovers != nil {
		t.Fatalf("checkpoint fields present on a non-contract run: %+v", p)
	}
	if strings.Contains(string(mustRead(t, f.packetPath)), "checkpoint_schema:") {
		t.Fatal("packet header prints checkpoint_schema on a non-contract run")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// retainBaselinePlan stores a minimal valid plan whose only distinguishing
// content is unit's permission classes, and returns its digest, so
// firstLeasePermissionClasses (cmd/hma/review.go) can read it back via
// loadPlans. Each call with a different permissionClasses value retains a
// plan at a different digest.
func retainBaselinePlan(t *testing.T, storePath, unit string, permissionClasses []string) string {
	t.Helper()
	check := model.Check{Executable: "true", Argv: []string{}, WorkingDir: "."}
	plan := model.Plan{Schema: model.PlanSchema, RunID: "run", AcceptanceContractDigest: "contract", PolicyDigest: "policy",
		Units: []model.PlanUnit{{ID: unit, Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"a"}, AllowedActions: []string{"edit"},
			ForbiddenFiles: []string{"docs"}, ForbiddenActions: []string{"push"}, PreChangeEvidence: "failing test", MinimumImplementation: "m",
			ReuseAnalysis: "r", PermissionClasses: permissionClasses, ImplementerRoute: "impl", EscalationRoute: "esc", ValidatorRoute: "val",
			IndependenceRequirement: "family", VerificationCommands: []model.Check{check},
			RequiredEvidence: []string{"check"}, StopConditions: []string{"s"}, RollbackConditions: []string{"r"}, EscalationConditions: []string{"e"},
			NonGoals: []string{"n"}, Dependencies: []string{},
			Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 100, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 120, CostCeiling: 10}}}}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	parsed, digest, err := model.ParsePlan(planBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := retainPlan(storePath, parsed, digest); err != nil {
		t.Fatal(err)
	}
	return digest
}

// contractAcceptanceRecord is a minimal approved ACCEPTANCE_CRITERIA ->
// PLANNING core transition binding contractDigest, exactly what
// firstContractAcceptance (cmd/hma/review.go) reads.
func contractAcceptanceRecord(contractDigest string) model.Record {
	return model.Record{Kind: model.KindStageTransition, Control: model.ControlApproved, Approval: &model.ApprovalBinding{
		CurrentStage: model.StageAcceptanceCriteria, ProposedTargetStage: model.StagePlanning, AcceptanceContractDigest: contractDigest,
	}}
}

// leaseGenerationRecord is a minimal approved ROUTE_SELECTION ->
// IMPLEMENTATION_AUTHORIZATION core transition for unit, bound to planDigest
// (a plan retainBaselinePlan stored) -- exactly what engine.IssuesLease and
// firstLeasePermissionClasses (cmd/hma/review.go) read.
func leaseGenerationRecord(unit, planDigest string) model.Record {
	return model.Record{UnitID: unit, Kind: model.KindStageTransition, Control: model.ControlApproved, Approval: &model.ApprovalBinding{
		UnitID: unit, CurrentStage: model.StageRouteSelection, ProposedTargetStage: model.StageImplementationAuthorization, AcceptedPlanDigest: planDigest,
	}}
}

// approvedCheckpointReview is a minimal approved host dispatch/review pair at
// level checkpoint, exactly what checkpointReviewBaseline's backward scan
// over history reads: a dispatch whose packet carries checkpoint_schema,
// issuing_reason checkpoint, and checkpoint_covers set to covers, answered by
// an approving review of it.
func approvedCheckpointReview(dispatchDigest, unit string, covers engine.CheckpointBaseline) []hoststate.Record {
	return []hoststate.Record{
		{Kind: hoststate.KindDispatch, HeadAnchor: dispatchDigest, Dispatch: &hoststate.Dispatch{UnitID: unit, Role: hoststate.RoleReview,
			Packet: &hoststate.ReviewPacketBinding{IssuingReason: hoststate.IssueCheckpoint, CheckpointSchema: hoststate.CheckpointSchemaV1, CheckpointCovers: &covers}}},
		{Kind: hoststate.KindReview, Review: &hoststate.IndependentReview{UnitID: unit, ReviewDispatchDigest: dispatchDigest, Verdict: hoststate.ReviewApprove}},
	}
}

// approvedPerChangeReview is an approved host dispatch/review pair issued
// per_change, for the "must not be picked up as a baseline" cases: withSchema
// true models a contract-bound run where the classifier ran but derived
// per_change (checkpoint_schema present, checkpoint_covers set, but
// issuing_reason is still per_change); withSchema false models a non-
// contract-bound run (no classifier run at all, checkpoint_schema absent).
// Either way checkpointReviewBaseline must skip it.
func approvedPerChangeReview(dispatchDigest, unit string, withSchema bool) []hoststate.Record {
	p := &hoststate.ReviewPacketBinding{IssuingReason: hoststate.IssuePerChange}
	if withSchema {
		p.CheckpointSchema = hoststate.CheckpointSchemaV1
		p.CheckpointCovers = &engine.CheckpointBaseline{ContractDigest: "must-not-be-picked", PermissionClasses: []string{"must-not-be-picked"}}
	}
	return []hoststate.Record{
		{Kind: hoststate.KindDispatch, HeadAnchor: dispatchDigest, Dispatch: &hoststate.Dispatch{UnitID: unit, Role: hoststate.RoleReview, Packet: p}},
		{Kind: hoststate.KindReview, Review: &hoststate.IndependentReview{UnitID: unit, ReviewDispatchDigest: dispatchDigest, Verdict: hoststate.ReviewApprove}},
	}
}

// TestCheckpointBaselineUsesLatestApprovedReview is Phil's finding 1's first
// scenario: with two approved checkpoint-grade reviews over time, the
// baseline checkpointReviewBaseline derives for the next packet is the
// latest one's checkpoint_covers, not the earlier review's and not the
// first-acceptance/first-lease fallback.
func TestCheckpointBaselineUsesLatestApprovedReview(t *testing.T) {
	store := t.TempDir()
	unit := "U1"
	firstPlanDigest := retainBaselinePlan(t, store, unit, []string{model.PermissionReadOnly})
	core := []model.Record{contractAcceptanceRecord("contract-1"), leaseGenerationRecord(unit, firstPlanDigest)}

	var history []hoststate.Record
	earlier := engine.CheckpointBaseline{ContractDigest: "contract-1", PermissionClasses: []string{model.PermissionReadOnly}}
	latest := engine.CheckpointBaseline{ContractDigest: "contract-2", PermissionClasses: []string{model.PermissionBoundedWrite}}
	history = append(history, approvedCheckpointReview("d1", unit, earlier)...)
	history = append(history, approvedCheckpointReview("d2", unit, latest)...)

	got, err := checkpointReviewBaseline(store, history, core, unit)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, latest) {
		t.Fatalf("baseline = %+v, want the latest approved review's checkpoint_covers %+v (not the earlier review %+v or the fallback)", got, latest, earlier)
	}
}

// TestCheckpointBaselineFallsBackToFirstAcceptanceAndFirstLease is Phil's
// finding 1's second scenario: with no prior checkpoint-grade review, the
// unit's second contract re-acceptance and second lease generation (a
// different plan, different permission classes) must not move the baseline;
// it stays the run's first contract acceptance and the unit's first lease
// generation's permission classes. The mismatch against the current state
// is exactly what should fire T1.
func TestCheckpointBaselineFallsBackToFirstAcceptanceAndFirstLease(t *testing.T) {
	store := t.TempDir()
	unit := "U1"
	firstPlanDigest := retainBaselinePlan(t, store, unit, []string{model.PermissionReadOnly})
	secondPlanDigest := retainBaselinePlan(t, store, unit, []string{model.PermissionBoundedWrite})
	core := []model.Record{
		contractAcceptanceRecord("contract-1"),
		leaseGenerationRecord(unit, firstPlanDigest),
		contractAcceptanceRecord("contract-2"),
		leaseGenerationRecord(unit, secondPlanDigest),
	}

	got, err := checkpointReviewBaseline(store, nil, core, unit)
	if err != nil {
		t.Fatal(err)
	}
	want := engine.CheckpointBaseline{ContractDigest: "contract-1", PermissionClasses: []string{model.PermissionReadOnly}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("baseline = %+v, want the first acceptance/first lease %+v (not the current contract-2/bounded_write)", got, want)
	}

	current := engine.CheckpointBaseline{ContractDigest: "contract-2", PermissionClasses: []string{model.PermissionBoundedWrite}}
	decision := engine.ClassifyCheckpoint(engine.CheckpointInput{Baseline: got, Current: current, Unit: model.PlanUnit{PermissionClasses: current.PermissionClasses}})
	fired := false
	for _, tr := range decision.Triggers {
		if tr == engine.CheckpointContractOrPermission {
			fired = true
		}
	}
	if !fired {
		t.Fatal("first-acceptance/first-lease baseline against the current (re-accepted) state did not fire T1")
	}
}

// TestCheckpointBaselineIgnoresPerChangeReview is Phil's finding 1's third
// scenario: a per_change-only review, whether or not the classifier ran for
// it, must never be picked up as the baseline; a unit with only such
// reviews still falls back to the first-acceptance/first-lease baseline.
func TestCheckpointBaselineIgnoresPerChangeReview(t *testing.T) {
	store := t.TempDir()
	unit := "U1"
	firstPlanDigest := retainBaselinePlan(t, store, unit, []string{model.PermissionReadOnly})
	core := []model.Record{contractAcceptanceRecord("contract-1"), leaseGenerationRecord(unit, firstPlanDigest)}

	var history []hoststate.Record
	history = append(history, approvedPerChangeReview("d1", unit, false)...)
	history = append(history, approvedPerChangeReview("d2", unit, true)...)

	got, err := checkpointReviewBaseline(store, history, core, unit)
	if err != nil {
		t.Fatal(err)
	}
	want := engine.CheckpointBaseline{ContractDigest: "contract-1", PermissionClasses: []string{model.PermissionReadOnly}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("baseline = %+v, a per_change review's artifact was picked up; want the fallback %+v", got, want)
	}
}
