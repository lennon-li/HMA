package main

import (
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

func minimalAnswers() Answers {
	return Answers{
		Intent: "add a --json flag to hma show",
		Risk:   "low",
		Criteria: []CriterionAnswer{
			{
				Statement: "hma show --json emits valid JSON for a run",
				Threshold: "exit 0 and output parses as JSON",
				Check:     CheckAnswer{Executable: "go", Argv: []string{"test", "./cmd/hma/...", "-run", "TestShowJSON"}},
			},
		},
		Units: []UnitAnswer{
			{
				AllowedFiles:   []string{"cmd/hma/show.go", "cmd/hma/show_test.go"},
				ForbiddenFiles: []string{"internal/engine/", "internal/transition/"},
				NonGoals:       []string{"no change to hma show's default text output"},
			},
		},
	}
}

// TestBuildMinimalAnswersRoundTrips checks that a minimal answers set
// produces a contract and plan that already round-tripped through
// model.ParseAcceptanceContract/ParsePlan inside buildContractAndPlan, and
// that the returned documents carry the expected defaults.
func TestBuildMinimalAnswersRoundTrips(t *testing.T) {
	contract, digest, plan, err := buildContractAndPlan(minimalAnswers())
	if err != nil {
		t.Fatalf("buildContractAndPlan: %v", err)
	}
	if contract.RunID != "add-a-json-flag-to-hma-show" {
		t.Errorf("default run_id = %q", contract.RunID)
	}
	if plan.RunID != contract.RunID {
		t.Errorf("plan run_id %q != contract run_id %q", plan.RunID, contract.RunID)
	}
	if plan.AcceptanceContractDigest != digest {
		t.Errorf("plan.AcceptanceContractDigest %q != returned digest %q", plan.AcceptanceContractDigest, digest)
	}
	if plan.PolicyDigest != contract.PolicyDigest {
		t.Errorf("plan.PolicyDigest %q != contract.PolicyDigest %q", plan.PolicyDigest, contract.PolicyDigest)
	}
	if len(plan.Units) != 1 || plan.Units[0].ID != "U1" {
		t.Fatalf("unexpected units: %+v", plan.Units)
	}
	u := plan.Units[0]
	if len(u.CriteriaAdvanced) != 1 || u.CriteriaAdvanced[0] != "C1" {
		t.Errorf("default criteria_advanced = %v", u.CriteriaAdvanced)
	}
	if len(u.VerificationCommands) != 1 {
		t.Errorf("expected verification commands defaulted from the criterion check, got %+v", u.VerificationCommands)
	}

	// Re-validate independently of buildContractAndPlan's own self-check,
	// so the test does not just assert "no error" against the function
	// under test's own internal validation call.
	if err := model.ValidateAcceptanceContract(contract); err != nil {
		t.Errorf("ValidateAcceptanceContract: %v", err)
	}
	if err := model.ValidatePlan(plan); err != nil {
		t.Errorf("ValidatePlan: %v", err)
	}
	if err := model.ValidatePlanAgainstContract(plan, contract, digest); err != nil {
		t.Errorf("ValidatePlanAgainstContract: %v", err)
	}
}

// TestBuildExplicitFieldsOverrideDefaults checks that a fully-specified
// answers set is honored rather than silently replaced by defaults.
func TestBuildExplicitFieldsOverrideDefaults(t *testing.T) {
	a := minimalAnswers()
	a.RunID = "my-run"
	a.NonGoals = []string{"no unrelated refactor"}
	a.Deliverables = []string{"code", "docs"}
	a.Criteria[0].ID = "SHOW1"
	a.Criteria[0].EvidenceRequirement = "CI log"
	a.Units[0].ID = "UNIT-A"
	a.Units[0].CriteriaAdvanced = []string{"SHOW1"}
	a.Units[0].Dependencies = []string{}

	contract, _, plan, err := buildContractAndPlan(a)
	if err != nil {
		t.Fatalf("buildContractAndPlan: %v", err)
	}
	if contract.RunID != "my-run" {
		t.Errorf("run_id = %q, want my-run", contract.RunID)
	}
	if contract.Criteria[0].ID != "SHOW1" || contract.Criteria[0].EvidenceRequirement != "CI log" {
		t.Errorf("criterion not honored: %+v", contract.Criteria[0])
	}
	if len(contract.Deliverables) != 2 {
		t.Errorf("deliverables not honored: %v", contract.Deliverables)
	}
	if plan.Units[0].ID != "UNIT-A" {
		t.Errorf("unit id = %q, want UNIT-A", plan.Units[0].ID)
	}
}

// TestBuildTwoUnitsRequiresExplicitCriteriaAdvanced checks that hmarun
// refuses to guess which unit advances which criterion once there is more
// than one unit, with a clear, path-named error rather than an opaque
// validator failure about an uncovered criterion.
func TestBuildTwoUnitsRequiresExplicitCriteriaAdvanced(t *testing.T) {
	a := minimalAnswers()
	a.Criteria = append(a.Criteria, CriterionAnswer{
		Statement: "a second criterion",
		Threshold: "exit 0",
		Check:     CheckAnswer{Executable: "go", Argv: []string{"vet", "./..."}},
	})
	a.Units = append(a.Units, UnitAnswer{
		AllowedFiles:   []string{"cmd/hma/other.go"},
		ForbiddenFiles: []string{"internal/engine/"},
		NonGoals:       []string{"no unrelated refactor"},
	})

	_, _, _, err := buildContractAndPlan(a)
	if err == nil {
		t.Fatal("expected an error when two units omit criteria_advanced")
	}
	var ae *answerError
	if ok := asAnswerError(err, &ae); !ok {
		t.Fatalf("expected an *answerError, got %T: %v", err, err)
	}
	if ae.path != "units[0].criteria_advanced" {
		t.Errorf("error path = %q, want units[0].criteria_advanced", ae.path)
	}
}

func asAnswerError(err error, target **answerError) bool {
	if ae, ok := err.(*answerError); ok {
		*target = ae
		return true
	}
	return false
}

// TestBuildMissingRequiredFieldsGiveClearErrors checks that hmarun's own
// Q&A/templating code catches a missing required answer with a clear,
// path-named error, rather than letting it fall through to an opaque
// validator failure.
func TestBuildMissingRequiredFieldsGiveClearErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(a *Answers)
		wantSub string
	}{
		{"missing intent", func(a *Answers) { a.Intent = "" }, "intent"},
		{"missing risk", func(a *Answers) { a.Risk = "" }, "risk"},
		{"invalid risk", func(a *Answers) { a.Risk = "extreme" }, "risk"},
		{"no criteria", func(a *Answers) { a.Criteria = nil }, "criteria"},
		{"criterion missing statement", func(a *Answers) { a.Criteria[0].Statement = "" }, "criteria[0].statement"},
		{"criterion missing threshold", func(a *Answers) { a.Criteria[0].Threshold = "" }, "criteria[0].threshold"},
		{"criterion missing check executable", func(a *Answers) { a.Criteria[0].Check.Executable = "" }, "criteria[0].check.executable"},
		{"criterion missing check argv", func(a *Answers) { a.Criteria[0].Check.Argv = nil }, "criteria[0].check.argv"},
		{"unit missing allowed_files", func(a *Answers) { a.Units[0].AllowedFiles = nil }, "units[0].allowed_files"},
		{"unit missing forbidden_files", func(a *Answers) { a.Units[0].ForbiddenFiles = nil }, "units[0].forbidden_files"},
		{"unit missing non_goals with no top-level fallback", func(a *Answers) { a.Units[0].NonGoals = nil }, "units[0].non_goals"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := minimalAnswers()
			tc.mutate(&a)
			_, _, _, err := buildContractAndPlan(a)
			if err == nil {
				t.Fatalf("expected an error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestBuildUnitNonGoalsFallBackToTopLevel checks that a unit with no
// non_goals of its own, but a non-empty top-level non_goals, is accepted.
func TestBuildUnitNonGoalsFallBackToTopLevel(t *testing.T) {
	a := minimalAnswers()
	a.NonGoals = []string{"no unrelated refactor"}
	a.Units[0].NonGoals = nil

	_, _, plan, err := buildContractAndPlan(a)
	if err != nil {
		t.Fatalf("buildContractAndPlan: %v", err)
	}
	if len(plan.Units[0].NonGoals) != 1 || plan.Units[0].NonGoals[0] != "no unrelated refactor" {
		t.Errorf("unit non_goals = %v, want fallback to top-level", plan.Units[0].NonGoals)
	}
}
