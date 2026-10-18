package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func testContract() AcceptanceContract {
	return AcceptanceContract{Schema: AcceptanceContractSchema, RunID: "run", Intent: "add one bounded behavior",
		NonGoals: []string{"no refactor"},
		Criteria: []ContractCriterion{
			{ID: "C1", Statement: "unit tests pass", Check: Check{Executable: "go", Argv: []string{"test", "./..."}, WorkingDir: "."}, Threshold: "exit 0", EvidenceRequirement: "fresh capture"},
			{ID: "C2", Statement: "vet clean", Check: Check{Executable: "go", Argv: []string{"vet", "./..."}, WorkingDir: "."}, Threshold: "exit 0", EvidenceRequirement: "fresh capture"},
		},
		Risk: RouteRiskMedium, Deliverables: []string{"code"},
		Budgets:         ContractBudgets{MaxAttempts: 5, MaxWallMinutes: 120, CostCeiling: 100, CostUnit: "relative"},
		PermittedRoutes: []string{"impl", "esc", "val"}, PermissionEnvelope: []string{PermissionBoundedWrite}, PolicyDigest: "policy"}
}

func testPlan(contractDigest string) Plan {
	unit := func(id string, deps ...string) PlanUnit {
		return PlanUnit{ID: id, Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"internal/x.go"}, AllowedActions: []string{"edit"},
			ForbiddenFiles: []string{"docs/architecture.md"}, ForbiddenActions: []string{"git push"}, PreChangeEvidence: "failing test", MinimumImplementation: "one function",
			ReuseAnalysis: "none applicable", PermissionClasses: []string{PermissionBoundedWrite}, ImplementerRoute: "impl", EscalationRoute: "esc", ValidatorRoute: "val",
			IndependenceRequirement: "different family", VerificationCommands: []Check{{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}},
			RequiredEvidence: []string{"go test"}, StopConditions: []string{"budget"}, RollbackConditions: []string{"regression"}, EscalationConditions: []string{"two failures"},
			NonGoals: []string{"refactor"}, Dependencies: append([]string{}, deps...),
			Budgets: UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 200, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 60, CostCeiling: 50}}
	}
	p := Plan{Schema: PlanSchema, RunID: "run", AcceptanceContractDigest: contractDigest, PolicyDigest: "policy",
		Units: []PlanUnit{unit("U1"), unit("U2", "U1")}}
	p.Units[1].CriteriaAdvanced = []string{"C2"}
	return p
}

// TestAcceptanceContractRoundTrip: a contract parses, re-serializes to the
// same digest, and any change to a definition changes the digest.
func TestAcceptanceContractRoundTrip(t *testing.T) {
	c := testContract()
	raw, _ := json.Marshal(c)
	got, digest, err := ParseAcceptanceContract(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := json.Marshal(got)
	if _, d2, err := ParseAcceptanceContract(again); err != nil || d2 != digest {
		t.Fatalf("round trip digest %s != %s (%v)", d2, digest, err)
	}
	for name, edit := range map[string]func(*AcceptanceContract){
		"threshold": func(c *AcceptanceContract) { c.Criteria[0].Threshold = "exit 0 twice" },
		"statement": func(c *AcceptanceContract) { c.Criteria[0].Statement = "most tests pass" },
		"check":     func(c *AcceptanceContract) { c.Criteria[0].Check.Argv = []string{"test", "./internal/..."} },
		"budget":    func(c *AcceptanceContract) { c.Budgets.MaxAttempts = 6 },
		"non-goal":  func(c *AcceptanceContract) { c.NonGoals = nil },
	} {
		m := testContract()
		edit(&m)
		raw, _ := json.Marshal(m)
		if _, d, err := ParseAcceptanceContract(raw); err == nil && d == digest {
			t.Errorf("%s change kept the digest", name)
		}
	}
}

// Contracts and plans are read with exact unique keys, no unknown fields,
// and one document only.
func TestDocumentDecodingIsStrict(t *testing.T) {
	raw, _ := json.Marshal(testContract())
	body := string(raw[:len(raw)-1])
	for name, doc := range map[string]string{
		"case-variant":  body + `,"Run_ID":"other"}`,
		"duplicate":     body + `,"run_id":"other"}`,
		"unknown-field": body + `,"extra":1}`,
		"trailing":      string(raw) + ` {}`,
		"result-field":  strings.Replace(string(raw), `"threshold":"exit 0"`, `"threshold":"exit 0","disposition":"PASSED"`, 1),
	} {
		if _, _, err := ParseAcceptanceContract([]byte(doc)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestContractValidation(t *testing.T) {
	for name, edit := range map[string]func(*AcceptanceContract){
		"duplicate-id": func(c *AcceptanceContract) { c.Criteria[1].ID = "C1" },
		"slash-id":     func(c *AcceptanceContract) { c.Criteria[0].ID = "a/b" },
		"shell-check":  func(c *AcceptanceContract) { c.Criteria[0].Check.Executable = "sh;id" },
		"no-criteria":  func(c *AcceptanceContract) { c.Criteria = nil },
		"bad-risk":     func(c *AcceptanceContract) { c.Risk = "extreme" },
		"zero-budget":  func(c *AcceptanceContract) { c.Budgets.MaxWallMinutes = 0 },
		"no-policy":    func(c *AcceptanceContract) { c.PolicyDigest = "" },
	} {
		c := testContract()
		edit(&c)
		if ValidateAcceptanceContract(c) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

// Plan validation: unique ids, existing dependencies, acyclic graph,
// positive lease ceilings; and a plan must stay within its contract.
func TestPlanValidation(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))
	p := testPlan(cd)
	raw := mustJSON(p)
	got, digest, err := ParsePlan(raw)
	if err != nil || digest == "" {
		t.Fatal(err)
	}
	if err := ValidatePlanAgainstContract(got, testContract(), cd); err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(*Plan){
		"duplicate-unit": func(p *Plan) { p.Units[1].ID = "U1" },
		"unknown-dep":    func(p *Plan) { p.Units[1].Dependencies = []string{"U9"} },
		"self-dep":       func(p *Plan) { p.Units[0].Dependencies = []string{"U1"} },
		"cycle":          func(p *Plan) { p.Units[0].Dependencies = []string{"U2"} },
		"no-wall":        func(p *Plan) { p.Units[0].Budgets.MaxWallMinutes = 0 },
		"no-policy":      func(p *Plan) { p.PolicyDigest = "" },
	} {
		q := testPlan(cd)
		edit(&q)
		if ValidatePlan(q) == nil {
			t.Errorf("%s accepted", name)
		}
	}
	for name, edit := range map[string]func(*Plan){
		"other-contract":   func(p *Plan) { p.AcceptanceContractDigest = "other" },
		"other-policy":     func(p *Plan) { p.PolicyDigest = "other" },
		"unknown-criteria": func(p *Plan) { p.Units[0].CriteriaAdvanced = []string{"C9"} },
		"over-ceiling":     func(p *Plan) { p.Units[0].Budgets.MaxAttempts = 6 },
	} {
		q := testPlan(cd)
		edit(&q)
		if ValidatePlanAgainstContract(q, testContract(), cd) == nil {
			t.Errorf("%s accepted against the contract", name)
		}
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// Liz D1+D2 F04: invalid UTF-8 and lone surrogate escapes are refused; a
// valid pair and literal U+FFFD are kept.
func TestDocumentUnicodeIntegrity(t *testing.T) {
	raw := string(mustJSON(testContract()))
	for name, intent := range map[string]string{"invalid-byte": "a\xffb", "lone-high": `a\ud800b`, "lone-low": `a\udc00b`, "reversed": `\udc00\ud800`} {
		doc := strings.Replace(raw, `"intent":"add one bounded behavior"`, `"intent":"`+intent+`"`, 1)
		if _, _, err := ParseAcceptanceContract([]byte(doc)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	for name, intent := range map[string]string{"pair": `😀`, "replacement": `�`, "escaped-backslash": `a\\ud800`} {
		doc := strings.Replace(raw, `"intent":"add one bounded behavior"`, `"intent":"`+intent+`"`, 1)
		if _, _, err := ParseAcceptanceContract([]byte(doc)); err != nil {
			t.Errorf("%s refused: %v", name, err)
		}
	}
}

// F05: every §10 field is required; dependencies must be declared.
func TestPlanUnitFieldsRequired(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))
	for name, edit := range map[string]func(*PlanUnit){
		"criteria":      func(u *PlanUnit) { u.CriteriaAdvanced = nil },
		"allowed-files": func(u *PlanUnit) { u.AllowedFiles = nil },
		"forbidden":     func(u *PlanUnit) { u.ForbiddenActions = nil },
		"pre-change":    func(u *PlanUnit) { u.PreChangeEvidence = "" },
		"minimum":       func(u *PlanUnit) { u.MinimumImplementation = "" },
		"reuse":         func(u *PlanUnit) { u.ReuseAnalysis = "" },
		"escalation":    func(u *PlanUnit) { u.EscalationRoute = "" },
		"independence":  func(u *PlanUnit) { u.IndependenceRequirement = "" },
		"evidence":      func(u *PlanUnit) { u.RequiredEvidence = nil },
		"stop":          func(u *PlanUnit) { u.StopConditions = nil },
		"rollback":      func(u *PlanUnit) { u.RollbackConditions = nil },
		"non-goals":     func(u *PlanUnit) { u.NonGoals = nil },
		"permissions":   func(u *PlanUnit) { u.PermissionClasses = []string{"root"} },
		"complexity":    func(u *PlanUnit) { u.Budgets.MaxComplexity = 0 },
		"deps-missing":  func(u *PlanUnit) { u.Dependencies = nil },
		"wall-overflow": func(u *PlanUnit) { u.Budgets.MaxWallMinutes = MaxWallMinutesLimit + 1 },
	} {
		p := testPlan(cd)
		edit(&p.Units[0])
		if _, _, err := ParsePlan(mustJSON(p)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	p := testPlan(cd)
	p.AcceptanceContractDigest = ""
	if ValidatePlan(p) == nil {
		t.Error("plan without a contract digest accepted")
	}
}

// F01: plan routes and permissions must lie within the contract, and every
// contract criterion must be advanced by some unit.
func TestPlanWithinContractRoutesAndPermissions(t *testing.T) {
	c := testContract()
	_, cd, _ := ParseAcceptanceContract(mustJSON(c))
	for name, edit := range map[string]func(*Plan){
		"implementer": func(p *Plan) { p.Units[0].ImplementerRoute = "other" },
		"escalation":  func(p *Plan) { p.Units[0].EscalationRoute = "other" },
		"validator":   func(p *Plan) { p.Units[0].ValidatorRoute = "other" },
		"permission":  func(p *Plan) { p.Units[0].PermissionClasses = []string{PermissionReadOnly} },
		"coverage":    func(p *Plan) { p.Units[1].CriteriaAdvanced = []string{"C1"} },
	} {
		p := testPlan(cd)
		edit(&p)
		if ValidatePlanAgainstContract(p, c, cd) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

// Liz D1+D2 r2 D12R2-01: every field must be present and non-null; [] and
// 0 are explicit values.
func TestDocumentFieldsRequired(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))
	plan := string(mustJSON(testPlan(cd)))
	contract := string(mustJSON(testContract()))
	for name, doc := range map[string]string{
		"plan-no-retries":       strings.Replace(plan, `"max_retries":1,`, ``, 1),
		"plan-null-retries":     strings.Replace(plan, `"max_retries":1`, `"max_retries":null`, 1),
		"plan-no-argv":          strings.Replace(plan, `"argv":["test"],`, ``, 1),
		"plan-null-deps":        strings.Replace(plan, `"dependencies":[]`, `"dependencies":null`, 1),
		"contract-no-non-goals": strings.Replace(contract, `"non_goals":["no refactor"],`, ``, 1),
		"contract-no-deliver":   strings.Replace(contract, `"deliverables":["code"],`, ``, 1),
		"contract-null-argv":    strings.Replace(contract, `"argv":["test","./..."]`, `"argv":null`, 1),
	} {
		if !strings.Contains(doc, "") || doc == plan || doc == contract {
			t.Fatalf("%s: fixture edit did not apply", name)
		}
		var err error
		if strings.HasPrefix(name, "plan") {
			_, _, err = ParsePlan([]byte(doc))
		} else {
			_, _, err = ParseAcceptanceContract([]byte(doc))
		}
		if err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// Explicit zero retries and an empty argv are valid.
	ok := strings.Replace(strings.Replace(plan, `"max_retries":1`, `"max_retries":0`, 1), `"argv":["test"]`, `"argv":[]`, 1)
	if _, _, err := ParsePlan([]byte(ok)); err != nil {
		t.Fatalf("explicit zero and empty list refused: %v", err)
	}
	// Re-parsing a valid document keeps its digest.
	if _, d, err := ParseAcceptanceContract([]byte(contract)); err != nil || d != cd {
		t.Fatalf("digest changed: %v", err)
	}
}
