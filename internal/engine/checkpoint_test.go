package engine

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

// Note: TestCheckpointPolicyCatchAllOrdinaryRefused and
// TestCheckpointPolicyUnknownClassRefused from the draft §5.1 table already
// exist in internal/model/checkpoint_policy_test.go (Phase 2), against
// model.ValidateCheckpointPolicy. Policy parsing/validation is out of scope
// for this package (ClassifyCheckpoint only consumes an already-valid
// *model.CheckpointPolicy), and internal/model may not be touched here, so
// they are not duplicated in this file.

func hasTrigger(triggers []CheckpointTrigger, want CheckpointTrigger) bool {
	for _, t := range triggers {
		if t == want {
			return true
		}
	}
	return false
}

func TestCheckpointT1ContractReaccepted(t *testing.T) {
	in := CheckpointInput{
		Baseline: CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only"}},
		Current:  CheckpointBaseline{ContractDigest: "c2", PermissionClasses: []string{"read_only"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
		Stage:    model.StageIndependentValidation,
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointContractOrPermission) {
		t.Fatalf("want T1, got %v", d.Triggers)
	}
	if d.Level != IssuingLevelCheckpoint {
		t.Fatalf("level = %q, want checkpoint", d.Level)
	}
	if d.FamilySeparation {
		t.Fatal("family separation should be false for T1 alone")
	}
}

func TestCheckpointT1PermissionChanged(t *testing.T) {
	in := CheckpointInput{
		Baseline: CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only"}},
		Current:  CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only", "bounded_write"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
		Stage:    model.StageIndependentValidation,
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointContractOrPermission) {
		t.Fatalf("want T1, got %v", d.Triggers)
	}
}

func TestCheckpointT1ClearedByCheckpointReview(t *testing.T) {
	in := CheckpointInput{
		Baseline: CheckpointBaseline{ContractDigest: "c2", PermissionClasses: []string{"read_only", "bounded_write"}},
		Current:  CheckpointBaseline{ContractDigest: "c2", PermissionClasses: []string{"read_only", "bounded_write"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
		Stage:    model.StageIndependentValidation,
	}
	d := ClassifyCheckpoint(in)
	if hasTrigger(d.Triggers, CheckpointContractOrPermission) {
		t.Fatalf("T1 should be cleared once baseline matches current, got %v", d.Triggers)
	}
}

func TestCheckpointT1FirstContractNoTrigger(t *testing.T) {
	in := CheckpointInput{
		Baseline: CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only"}},
		Current:  CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
		Stage:    model.StageIndependentValidation,
	}
	d := ClassifyCheckpoint(in)
	if hasTrigger(d.Triggers, CheckpointContractOrPermission) {
		t.Fatalf("a single, never-rewound contract must not fire T1, got %v", d.Triggers)
	}
}

func TestCheckpointT2ContractHigh(t *testing.T) {
	in := CheckpointInput{
		Contract: &model.AcceptanceContract{Risk: model.RouteRiskHigh},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointHighRisk) {
		t.Fatalf("want T2, got %v", d.Triggers)
	}
	if !d.FamilySeparation {
		t.Fatal("want family separation for high contract risk")
	}
	if d.RequiredRisk != model.RouteRiskHigh {
		t.Fatalf("required risk = %q, want high", d.RequiredRisk)
	}
}

func TestCheckpointT2ContractCritical(t *testing.T) {
	in := CheckpointInput{
		Contract: &model.AcceptanceContract{Risk: model.RouteRiskCritical},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointHighRisk) {
		t.Fatalf("want T2, got %v", d.Triggers)
	}
	if !d.FamilySeparation {
		t.Fatal("want family separation for critical contract risk")
	}
	if d.RequiredRisk != model.RouteRiskCritical {
		t.Fatalf("required risk = %q, want critical", d.RequiredRisk)
	}
}

func policyWithClasses(glob string, classes ...string) *model.CheckpointPolicy {
	return &model.CheckpointPolicy{
		Schema: model.CheckpointPolicySchema,
		Rules:  []model.CheckpointPolicyRule{{ID: "r1", Glob: glob, Classes: classes}},
	}
}

func TestCheckpointT3PublicInterface(t *testing.T) {
	in := CheckpointInput{
		Policy:   policyWithClasses("cmd/hma/**", "public_interface"),
		Manifest: []ChangedPath{{Path: "cmd/hma/review.go", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointInterface) {
		t.Fatalf("want T3, got %v", d.Triggers)
	}
	if d.RiskFloor != model.RouteRiskHigh {
		t.Fatalf("risk floor = %q, want high", d.RiskFloor)
	}
	if !d.FamilySeparation {
		t.Fatal("want family separation for T3")
	}
}

func TestCheckpointT3Schema(t *testing.T) {
	in := CheckpointInput{
		Policy:   policyWithClasses("internal/model/**", "persisted_schema"),
		Manifest: []ChangedPath{{Path: "internal/model/contract.go", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointInterface) {
		t.Fatalf("want T3, got %v", d.Triggers)
	}
	if d.RiskFloor != model.RouteRiskHigh || !d.FamilySeparation {
		t.Fatalf("want risk floor high and family separation, got floor=%q familySep=%v", d.RiskFloor, d.FamilySeparation)
	}
}

func TestCheckpointT3Protocol(t *testing.T) {
	in := CheckpointInput{
		Policy:   policyWithClasses("api/**", "protocol"),
		Manifest: []ChangedPath{{Path: "api/wire.go", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointInterface) {
		t.Fatalf("want T3, got %v", d.Triggers)
	}
	if d.RiskFloor != model.RouteRiskHigh || !d.FamilySeparation {
		t.Fatalf("want risk floor high and family separation, got floor=%q familySep=%v", d.RiskFloor, d.FamilySeparation)
	}
}

func TestCheckpointT4AllowedEffect(t *testing.T) {
	in := CheckpointInput{
		Policy:   policyWithClasses("docs/**", "ordinary"),
		Manifest: []ChangedPath{{Path: "docs/readme.md", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionBoundedWrite}}, // lacks no_external_effect
		Stage:    model.StageIndependentValidation,
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointExternalEffect) {
		t.Fatalf("want T4, got %v", d.Triggers)
	}
	if d.Level != IssuingLevelCheckpoint {
		t.Fatalf("level = %q, want checkpoint at IV", d.Level)
	}
	if d.RiskFloor != model.RouteRiskLow {
		t.Fatalf("risk floor = %q, want low (allowed-only T4 does not raise it)", d.RiskFloor)
	}
	if d.FamilySeparation {
		t.Fatal("allowed-only T4 must not require family separation")
	}
}

func TestCheckpointT4PathClass(t *testing.T) {
	in := CheckpointInput{
		Policy:   policyWithClasses("infra/**", "irreversible"),
		Manifest: []ChangedPath{{Path: "infra/migrate.go", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointExternalEffect) {
		t.Fatalf("want T4, got %v", d.Triggers)
	}
	if d.RiskFloor != model.RouteRiskHigh {
		t.Fatalf("risk floor = %q, want high (observed T4 raises it)", d.RiskFloor)
	}
	if !d.FamilySeparation {
		t.Fatal("observed T4 must require family separation")
	}
}

func TestCheckpointT5EachControlClass(t *testing.T) {
	for _, class := range []string{"security", "approval", "routing", "evidence", "closure"} {
		t.Run(class, func(t *testing.T) {
			in := CheckpointInput{
				Policy:   policyWithClasses("x/**", class),
				Manifest: []ChangedPath{{Path: "x/y.go", Status: "modified"}},
				Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
			}
			d := ClassifyCheckpoint(in)
			if !hasTrigger(d.Triggers, CheckpointControlLogic) {
				t.Fatalf("class %s: want T5, got %v", class, d.Triggers)
			}
			if d.RiskFloor != model.RouteRiskHigh {
				t.Fatalf("class %s: risk floor = %q, want high", class, d.RiskFloor)
			}
		})
	}
}

func TestCheckpointT6Milestone(t *testing.T) {
	unit := model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}, MaterialMilestone: true}

	atIV := ClassifyCheckpoint(CheckpointInput{Unit: unit, Stage: model.StageIndependentValidation})
	if !hasTrigger(atIV.Triggers, CheckpointMilestone) {
		t.Fatalf("want T6 at IV, got %v", atIV.Triggers)
	}
	if atIV.FamilySeparation {
		t.Fatal("T6 alone must not require family separation")
	}

	atReview := ClassifyCheckpoint(CheckpointInput{Unit: unit, Stage: model.StageImplementationReview})
	if hasTrigger(atReview.Triggers, CheckpointMilestone) {
		t.Fatalf("T6 must not fire outside IV, got %v", atReview.Triggers)
	}
}

func TestCheckpointT7PerOutcome(t *testing.T) {
	for _, outcome := range []model.TerminalOutcome{
		model.OutcomeAborted, model.OutcomeBlocked, model.OutcomeUnknown, model.OutcomeFailed,
		model.OutcomePartial, model.OutcomeVerifiedWithWaivers, model.OutcomeVerifiedSuccess,
	} {
		t.Run(string(outcome), func(t *testing.T) {
			in := CheckpointInput{
				Unit:          model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
				TargetOutcome: outcome,
			}
			d := ClassifyCheckpoint(in)
			want := FinalAuditOutcomes[outcome]
			got := hasTrigger(d.Triggers, CheckpointFinalClosure)
			if got != want {
				t.Fatalf("outcome %s: T7 = %v, want %v", outcome, got, want)
			}
			if want && d.Level != IssuingLevelFinal {
				t.Fatalf("outcome %s: level = %q, want final", outcome, d.Level)
			}
			if !want && d.Level == IssuingLevelFinal {
				t.Fatalf("outcome %s: level must not be final", outcome)
			}
			if want && !d.FamilySeparation {
				t.Fatalf("outcome %s: T7 must require family separation", outcome)
			}
		})
	}
}

func TestCheckpointT8NoPolicy(t *testing.T) {
	in := CheckpointInput{
		Policy: nil,
		Manifest: []ChangedPath{
			{Path: "a/b.go", Status: "modified"},
			{Path: "c/d.go", Status: "modified"},
		},
		Unit: model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointUnclassified) {
		t.Fatalf("want T8, got %v", d.Triggers)
	}
	if len(d.UnclassifiedPaths) != 2 || d.UnclassifiedPaths[0] != "a/b.go" || d.UnclassifiedPaths[1] != "c/d.go" {
		t.Fatalf("unclassified paths = %v, want every path, sorted", d.UnclassifiedPaths)
	}
	if d.RiskFloor != model.RouteRiskLow {
		t.Fatalf("risk floor = %q, want low (T8 does not raise it)", d.RiskFloor)
	}
	if d.FamilySeparation {
		t.Fatal("T8 alone must not require family separation")
	}
}

func TestCheckpointT8UnmatchedPath(t *testing.T) {
	in := CheckpointInput{
		Policy: policyWithClasses("docs/**", "ordinary"),
		Manifest: []ChangedPath{
			{Path: "docs/readme.md", Status: "modified"},
			{Path: "src/x.go", Status: "modified"},
		},
		Unit: model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointUnclassified) {
		t.Fatalf("want T8, got %v", d.Triggers)
	}
	if len(d.UnclassifiedPaths) != 1 || d.UnclassifiedPaths[0] != "src/x.go" {
		t.Fatalf("unclassified paths = %v, want only src/x.go", d.UnclassifiedPaths)
	}
}

func TestCheckpointOrdinaryDoesNotCancel(t *testing.T) {
	in := CheckpointInput{
		Policy: &model.CheckpointPolicy{
			Schema: model.CheckpointPolicySchema,
			Rules: []model.CheckpointPolicyRule{
				{ID: "ord", Glob: "x/**", Classes: []string{"ordinary"}},
				{ID: "sec", Glob: "x/**", Classes: []string{"security"}},
			},
		},
		Manifest: []ChangedPath{{Path: "x/y.go", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if !hasTrigger(d.Triggers, CheckpointControlLogic) {
		t.Fatalf("ordinary must not cancel security, want T5, got %v", d.Triggers)
	}
	if len(d.UnclassifiedPaths) != 0 {
		t.Fatalf("a path matched by ordinary+security is classified, got unclassified %v", d.UnclassifiedPaths)
	}
}

func TestCheckpointRenameAcrossClasses(t *testing.T) {
	in := CheckpointInput{
		Policy: &model.CheckpointPolicy{
			Schema: model.CheckpointPolicySchema,
			Rules: []model.CheckpointPolicyRule{
				{ID: "docs", Glob: "docs/**", Classes: []string{"ordinary"}},
				{ID: "model", Glob: "internal/model/**", Classes: []string{"persisted_schema"}},
			},
		},
		Manifest: []ChangedPath{
			{Path: "docs/a", Status: "deleted"},
			{Path: "internal/model/a", Status: "added"},
		},
		Unit: model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if len(d.UnclassifiedPaths) != 0 {
		t.Fatalf("both the old and new rename paths are classified, got unclassified %v", d.UnclassifiedPaths)
	}
	if !hasTrigger(d.Triggers, CheckpointInterface) {
		t.Fatalf("want T3 from the new path's persisted_schema class, got %v", d.Triggers)
	}
}

func TestCheckpointOrdinaryOnlyNoTrigger(t *testing.T) {
	unit := model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}}
	base := CheckpointInput{
		Policy:   policyWithClasses("docs/**", "ordinary"),
		Manifest: []ChangedPath{{Path: "docs/readme.md", Status: "modified"}},
		Unit:     unit,
		Baseline: CheckpointBaseline{ContractDigest: "c1"},
		Current:  CheckpointBaseline{ContractDigest: "c1"},
	}

	before := base
	before.Stage = model.StageImplementationReview
	d1 := ClassifyCheckpoint(before)
	if len(d1.Triggers) != 0 {
		t.Fatalf("before IV: want no triggers, got %v", d1.Triggers)
	}
	if d1.Level != IssuingLevelPerChange {
		t.Fatalf("before IV: level = %q, want per_change", d1.Level)
	}

	atIV := base
	atIV.Stage = model.StageIndependentValidation
	d2 := ClassifyCheckpoint(atIV)
	if len(d2.Triggers) != 0 {
		t.Fatalf("at IV: want no triggers, got %v", d2.Triggers)
	}
	if d2.Level != IssuingLevelPerChange {
		t.Fatalf("at IV: level = %q, want per_change (D8 (1): no trigger needs no checkpoint)", d2.Level)
	}
}

func TestCheckpointFamilyRiskInvariant(t *testing.T) {
	cases := []struct {
		name     string
		risk     model.RouteRiskLevel
		disputed bool
		outcome  model.TerminalOutcome
		stage    model.Stage
	}{
		{"low-not-disputed-no-t7", model.RouteRiskLow, false, "", model.StagePlanning},
		{"medium-not-disputed-no-t7", model.RouteRiskMedium, false, "", model.StagePlanning},
		{"high-not-disputed-no-t7", model.RouteRiskHigh, false, "", model.StagePlanning},
		{"critical-not-disputed-no-t7", model.RouteRiskCritical, false, "", model.StagePlanning},
		{"low-disputed-no-t7", model.RouteRiskLow, true, "", model.StagePlanning},
		{"low-not-disputed-t7-via-outcome", model.RouteRiskLow, false, model.OutcomeVerifiedSuccess, model.StagePlanning},
		{"low-not-disputed-t7-via-stage", model.RouteRiskLow, false, "", model.StageReleaseAndClosure},
		{"high-disputed-t7", model.RouteRiskHigh, true, model.OutcomeVerifiedSuccess, model.StageReleaseAndClosure},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := CheckpointInput{
				Contract:      &model.AcceptanceContract{Risk: c.risk},
				Unit:          model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
				Disputed:      c.disputed,
				TargetOutcome: c.outcome,
				Stage:         c.stage,
			}
			d := ClassifyCheckpoint(in)
			t7 := hasTrigger(d.Triggers, CheckpointFinalClosure)
			want := routeRiskRank(c.risk) >= routeRiskRank(model.RouteRiskHigh) || c.disputed || t7
			if d.FamilySeparation != want {
				t.Fatalf("%s: family separation = %v, want %v (risk=%q disputed=%v t7=%v)", c.name, d.FamilySeparation, want, c.risk, c.disputed, t7)
			}
		})
	}
}

// TestCheckpointMisclassifiedPathKnownGap pins the named v1 gap (architecture
// §16 "Checkpoint triggers"): an approved policy that classes an interface
// path "ordinary" removes that path's checkpoint. This is a deliberate,
// accepted v1 limitation, not a bug to fix here.
func TestCheckpointMisclassifiedPathKnownGap(t *testing.T) {
	in := CheckpointInput{
		Policy:   policyWithClasses("cmd/hma/**", "ordinary"), // a real public-interface path, misclassified
		Manifest: []ChangedPath{{Path: "cmd/hma/review.go", Status: "modified"}},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}},
	}
	d := ClassifyCheckpoint(in)
	if hasTrigger(d.Triggers, CheckpointInterface) {
		t.Fatal("accepted v1 gap: a policy-misclassified interface path must not fire T3")
	}
	if len(d.UnclassifiedPaths) != 0 {
		t.Fatalf("the path is classified (as ordinary), so it must not appear as unclassified, got %v", d.UnclassifiedPaths)
	}
}

func TestCheckpointDeterministic(t *testing.T) {
	manifest := []ChangedPath{
		{Path: "cmd/hma/a.go", Status: "modified"},
		{Path: "internal/model/b.go", Status: "modified"},
		{Path: "docs/c.md", Status: "modified"},
		{Path: "unmatched/d.go", Status: "modified"},
	}
	policy := &model.CheckpointPolicy{
		Schema: model.CheckpointPolicySchema,
		Rules: []model.CheckpointPolicyRule{
			{ID: "cli", Glob: "cmd/hma/**", Classes: []string{"public_interface"}},
			{ID: "store", Glob: "internal/model/**", Classes: []string{"persisted_schema"}},
			{ID: "docs", Glob: "docs/**", Classes: []string{"ordinary", "ordinary"}},
		},
	}
	in := CheckpointInput{
		Contract:      &model.AcceptanceContract{Risk: model.RouteRiskMedium, CheckpointPolicyDigest: "pd1"},
		Policy:        policy,
		Unit:          model.PlanUnit{PermissionClasses: []string{model.PermissionNoExternalEffect}, MaterialMilestone: true},
		Baseline:      CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only", "bounded_write"}},
		Current:       CheckpointBaseline{ContractDigest: "c2", PermissionClasses: []string{"bounded_write", "read_only"}},
		Manifest:      manifest,
		Stage:         model.StageIndependentValidation,
		TargetOutcome: model.OutcomePartial,
		HostRisk:      model.RouteRiskLow,
		Disputed:      true,
	}
	want := ClassifyCheckpoint(in)

	baselineClasses := in.Baseline.PermissionClasses
	currentClasses := in.Current.PermissionClasses

	// 200 random permutations (seeded, so the test is reproducible rather
	// than flaky) of the manifest order, the policy rules order, and the
	// Baseline/Current permission-class order, each checked against want
	// with a full field-by-field (reflect.DeepEqual) comparison. This
	// replaces the single fixed shuffle: a lone permutation could pass by
	// chance even if ClassifyCheckpoint depended on order somewhere not
	// exercised by that one shuffle.
	const iterations = 200
	rnd := rand.New(rand.NewSource(42))
	for iter := 0; iter < iterations; iter++ {
		shuffledManifest := make([]ChangedPath, len(manifest))
		copy(shuffledManifest, manifest)
		shuffledRules := make([]model.CheckpointPolicyRule, len(policy.Rules))
		copy(shuffledRules, policy.Rules)
		shuffledBaselineClasses := make([]string, len(baselineClasses))
		copy(shuffledBaselineClasses, baselineClasses)
		shuffledCurrentClasses := make([]string, len(currentClasses))
		copy(shuffledCurrentClasses, currentClasses)

		rnd.Shuffle(len(shuffledManifest), func(i, j int) { shuffledManifest[i], shuffledManifest[j] = shuffledManifest[j], shuffledManifest[i] })
		rnd.Shuffle(len(shuffledRules), func(i, j int) { shuffledRules[i], shuffledRules[j] = shuffledRules[j], shuffledRules[i] })
		rnd.Shuffle(len(shuffledBaselineClasses), func(i, j int) {
			shuffledBaselineClasses[i], shuffledBaselineClasses[j] = shuffledBaselineClasses[j], shuffledBaselineClasses[i]
		})
		rnd.Shuffle(len(shuffledCurrentClasses), func(i, j int) {
			shuffledCurrentClasses[i], shuffledCurrentClasses[j] = shuffledCurrentClasses[j], shuffledCurrentClasses[i]
		})

		shuffled := in
		shuffled.Manifest = shuffledManifest
		shuffled.Policy = &model.CheckpointPolicy{Schema: policy.Schema, Rules: shuffledRules}
		shuffled.Baseline = CheckpointBaseline{ContractDigest: in.Baseline.ContractDigest, PermissionClasses: shuffledBaselineClasses}
		shuffled.Current = CheckpointBaseline{ContractDigest: in.Current.ContractDigest, PermissionClasses: shuffledCurrentClasses}

		got := ClassifyCheckpoint(shuffled)
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("iteration %d: shuffled input produced a different decision:\nwant %#v\ngot  %#v", iter, want, got)
		}
	}
}

// TestCheckpointGlobDoubleStarMatchesZeroSegments pins the merged
// architecture §16 text, "`**` matches any number of segments" — including
// zero, not just one-or-more — against matchCheckpointGlob directly.
func TestCheckpointGlobDoubleStarMatchesZeroSegments(t *testing.T) {
	if !matchCheckpointGlob("a/**", "a") {
		t.Fatal("a/** must match the bare path a (zero segments after a/)")
	}
	if !matchCheckpointGlob("a/**", "a/b/c") {
		t.Fatal("a/** must also match the multi-segment path a/b/c")
	}
}

// TestClassifyCheckpointDoesNotMutateInput follows the pattern at
// transitions_test.go:287 (TestEvaluationDoesNotMutateInput): marshal the
// request before and after, and require the bytes to be identical.
func TestClassifyCheckpointDoesNotMutateInput(t *testing.T) {
	in := CheckpointInput{
		Contract: &model.AcceptanceContract{Risk: model.RouteRiskHigh, CheckpointPolicyDigest: "pd1"},
		Policy: &model.CheckpointPolicy{
			Schema: model.CheckpointPolicySchema,
			Rules: []model.CheckpointPolicyRule{
				{ID: "cli", Glob: "cmd/hma/**", Classes: []string{"public_interface"}},
			},
		},
		Unit:     model.PlanUnit{PermissionClasses: []string{model.PermissionBoundedWrite}, MaterialMilestone: true},
		Baseline: CheckpointBaseline{ContractDigest: "c1", PermissionClasses: []string{"read_only"}},
		Current:  CheckpointBaseline{ContractDigest: "c2", PermissionClasses: []string{"read_only", "bounded_write"}},
		Manifest: []ChangedPath{
			{Path: "cmd/hma/a.go", Status: "modified"},
			{Path: "docs/b.md", Status: "added"},
		},
		Stage:         model.StageIndependentValidation,
		TargetOutcome: model.OutcomeVerifiedSuccess,
		HostRisk:      model.RouteRiskMedium,
		Disputed:      true,
	}
	before, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	_ = ClassifyCheckpoint(in)
	after, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("input mutated\nbefore: %s\n after: %s", before, after)
	}
}
