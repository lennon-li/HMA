package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
)

// answerError reports a problem with the answers themselves (missing or
// malformed field), as opposed to a document that fails internal/model's
// own validation. It names the exact answers-file path so the human can fix
// their input instead of reading an opaque validator error.
type answerError struct {
	path string
	msg  string
}

func (e *answerError) Error() string { return fmt.Sprintf("%s: %s", e.path, e.msg) }

func errAt(path, format string, a ...any) error {
	return &answerError{path: path, msg: fmt.Sprintf(format, a...)}
}

// defaultBudgets are the fixed, documented defaults hmarun fills in for a
// small bounded task when the human does not override them. They are not
// prompted for: the dispatch scope asks for task-specific decisions only,
// and these ceilings are generic policy, not a decision about this task.
var (
	defaultContractBudgets = model.ContractBudgets{
		MaxAttempts: 5, MaxWallMinutes: 120, CostCeiling: 100, CostUnit: "relative_units",
	}
	defaultUnitBudgets = model.UnitBudgets{
		MaxComplexity: 5, MaxFiles: 5, MaxDiffLines: 300, MaxToolSteps: 50,
		MaxRetries: 2, MaxAttempts: 3, MaxWallMinutes: 60, CostCeiling: 50,
	}
)

// defaultRouteNames and defaultPermissionClasses are the fixed route and
// permission-class defaults hmarun assigns to every unit's
// implementer/escalation/validator routes and permission envelope. hmarun
// has no route-selection logic of its own (that lives in internal/model's
// route policy and the real route selector); these are placeholder route
// identities a human or a later `hmarun run` step can rename, kept
// consistent between the contract's permitted_routes/permission_envelope
// and each unit's own route and permission fields so the documents are
// internally coherent on write.
const (
	defaultImplementerRoute = "implementer"
	defaultEscalationRoute  = "escalation"
	defaultValidatorRoute   = "validator"
)

var defaultPermissionClasses = []string{model.PermissionBoundedWrite}

// defaultPolicyDigest is the real digest of the canonical empty route
// policy document (hma.route-policy/v1, no fields set beyond its version),
// computed exactly as internal/engine.RoutePolicyDigest computes any other
// route policy's digest: hmarun init does not author a real, populated
// RoutePolicy, but this is not a free-standing placeholder either, because
// `hmarun run` (cmd/hmarun/run.go's writePolicy) writes exactly this same
// empty document and passes its path to every hma host command that reads
// a route policy. A document whose digest did not match a real policy
// hmarun run can actually produce would make every such command refuse
// with HOST_POLICY_NOT_CURRENT; this keeps hmarun init's output and hmarun
// run's expectations consistent by construction, with no coincidental
// string match to maintain by hand.
var defaultPolicyDigest = emptyRoutePolicyDigest()

func emptyRoutePolicyDigest() string {
	projection, err := model.NormalizeRoutePolicy(model.RoutePolicy{Version: model.RoutePolicyVersionV1})
	if err != nil {
		panic(err)
	}
	digest, err := engine.RoutePolicyDigest(projection)
	if err != nil {
		panic(err)
	}
	return digest
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// slugify turns s into a lowercase, hyphen-separated token suitable for a
// run id: runs of anything other than ASCII letters/digits collapse to a
// single hyphen, and leading/trailing hyphens are trimmed. It is pure
// string manipulation with no randomness or clock read, so the same intent
// always yields the same default run id.
func slugify(s string) string {
	var b strings.Builder
	prevHyphen := true // suppress a leading hyphen
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		default:
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	out := strings.TrimSuffix(b.String(), "-")
	if len(out) > 40 {
		out = strings.TrimSuffix(out[:40], "-")
	}
	return out
}

func (c CheckAnswer) toCheck(path string) (model.Check, error) {
	if strings.TrimSpace(c.Executable) == "" {
		return model.Check{}, errAt(path+".executable", "is required")
	}
	if c.Argv == nil {
		return model.Check{}, errAt(path+".argv", "is required (an empty list is written as [])")
	}
	wd := c.WorkingDir
	if wd == "" {
		wd = "."
	}
	return model.Check{Executable: c.Executable, Argv: c.Argv, WorkingDir: wd}, nil
}

// buildContractAndPlan turns Answers into a validated AcceptanceContract and
// Plan, applying every documented default. It returns an *answerError (a
// clear, path-named problem with the input) for a missing required answer,
// and otherwise self-checks the result by marshaling each document and
// round-tripping it through model.ParseAcceptanceContract/ParsePlan before
// returning: a templating bug therefore surfaces as that validator's exact
// error, never a silently-written invalid file.
func buildContractAndPlan(a Answers) (model.AcceptanceContract, string, model.Plan, error) {
	var zero model.AcceptanceContract
	var zeroPlan model.Plan

	if strings.TrimSpace(a.Intent) == "" {
		return zero, "", zeroPlan, errAt("intent", "is required")
	}
	if a.Risk == "" {
		return zero, "", zeroPlan, errAt("risk", "is required (one of: low, medium, high, critical)")
	}
	risk := model.RouteRiskLevel(strings.ToLower(a.Risk))
	if !model.ValidRouteRiskLevel(risk) {
		return zero, "", zeroPlan, errAt("risk", "must be one of: low, medium, high, critical (got %q)", a.Risk)
	}
	if len(a.Criteria) == 0 {
		return zero, "", zeroPlan, errAt("criteria", "must have at least one entry")
	}

	runID := a.RunID
	if runID == "" {
		runID = slugify(a.Intent)
		if runID == "" {
			return zero, "", zeroPlan, errAt("run_id", "could not be derived from intent; set run_id explicitly")
		}
	}

	nonGoals := a.NonGoals
	if nonGoals == nil {
		nonGoals = []string{}
	}
	deliverables := a.Deliverables
	if deliverables == nil {
		deliverables = []string{"code"}
	}

	criteria := make([]model.ContractCriterion, 0, len(a.Criteria))
	criterionIDs := make([]string, 0, len(a.Criteria))
	for i, ca := range a.Criteria {
		path := fmt.Sprintf("criteria[%d]", i)
		id := ca.ID
		if id == "" {
			id = fmt.Sprintf("C%d", i+1)
		}
		if strings.TrimSpace(ca.Statement) == "" {
			return zero, "", zeroPlan, errAt(path+".statement", "is required")
		}
		if strings.TrimSpace(ca.Threshold) == "" {
			return zero, "", zeroPlan, errAt(path+".threshold", "is required")
		}
		check, err := ca.Check.toCheck(path + ".check")
		if err != nil {
			return zero, "", zeroPlan, err
		}
		evidence := ca.EvidenceRequirement
		if evidence == "" {
			evidence = "fresh capture"
		}
		criteria = append(criteria, model.ContractCriterion{
			ID: id, Statement: ca.Statement, Check: check, Threshold: ca.Threshold, EvidenceRequirement: evidence,
		})
		criterionIDs = append(criterionIDs, id)
	}

	contract := model.AcceptanceContract{
		Schema:       model.AcceptanceContractSchema,
		RunID:        runID,
		Intent:       a.Intent,
		NonGoals:     nonGoals,
		Criteria:     criteria,
		Risk:         risk,
		Deliverables: deliverables,
		Budgets:      defaultContractBudgets,
		PermittedRoutes: []string{
			defaultImplementerRoute, defaultEscalationRoute, defaultValidatorRoute,
		},
		PermissionEnvelope: append([]string{}, defaultPermissionClasses...),
		PolicyDigest:       defaultPolicyDigest,
	}

	contractBytes, err := json.Marshal(contract)
	if err != nil {
		return zero, "", zeroPlan, fmt.Errorf("marshaling templated contract: %w", err)
	}
	parsedContract, contractDigest, err := model.ParseAcceptanceContract(contractBytes)
	if err != nil {
		return zero, "", zeroPlan, fmt.Errorf("hmarun templated an invalid contract (this is a hmarun bug, not an answers problem): %w", err)
	}

	unitAnswers := a.Units
	if len(unitAnswers) == 0 {
		unitAnswers = []UnitAnswer{{}}
	}

	units := make([]model.PlanUnit, 0, len(unitAnswers))
	for i, ua := range unitAnswers {
		path := fmt.Sprintf("units[%d]", i)
		id := ua.ID
		if id == "" {
			id = fmt.Sprintf("U%d", i+1)
		}
		if len(ua.AllowedFiles) == 0 {
			return zero, "", zeroPlan, errAt(path+".allowed_files", "must have at least one entry")
		}
		if len(ua.ForbiddenFiles) == 0 {
			return zero, "", zeroPlan, errAt(path+".forbidden_files", "must have at least one entry")
		}

		criteriaAdvanced := ua.CriteriaAdvanced
		if len(criteriaAdvanced) == 0 {
			if len(unitAnswers) != 1 {
				return zero, "", zeroPlan, errAt(path+".criteria_advanced",
					"is required when the plan has more than one unit (hmarun cannot guess which unit advances which criterion)")
			}
			criteriaAdvanced = append([]string{}, criterionIDs...)
		}

		unitNonGoals := ua.NonGoals
		if len(unitNonGoals) == 0 {
			unitNonGoals = nonGoals
		}
		if len(unitNonGoals) == 0 {
			return zero, "", zeroPlan, errAt(path+".non_goals",
				"is required (every plan unit needs at least one non-goal); set it here or set the top-level non_goals")
		}

		var verificationChecks []model.Check
		if len(ua.VerificationCommands) > 0 {
			for j, vc := range ua.VerificationCommands {
				check, err := vc.toCheck(fmt.Sprintf("%s.verification_commands[%d]", path, j))
				if err != nil {
					return zero, "", zeroPlan, err
				}
				verificationChecks = append(verificationChecks, check)
			}
		} else {
			for _, cid := range criteriaAdvanced {
				for _, cr := range criteria {
					if cr.ID == cid {
						verificationChecks = append(verificationChecks, cr.Check)
					}
				}
			}
		}
		if len(verificationChecks) == 0 {
			return zero, "", zeroPlan, errAt(path+".verification_commands", "must have at least one entry")
		}

		dependencies := ua.Dependencies
		if dependencies == nil {
			dependencies = []string{}
		}

		requiredEvidence := make([]string, 0, len(verificationChecks))
		for _, vc := range verificationChecks {
			requiredEvidence = append(requiredEvidence, fmt.Sprintf("%s %s exit status and output", vc.Executable, strings.Join(vc.Argv, " ")))
		}

		units = append(units, model.PlanUnit{
			ID:               id,
			Behavior:         a.Intent,
			CriteriaAdvanced: criteriaAdvanced,
			AllowedFiles:     ua.AllowedFiles,
			AllowedActions:   []string{"edit"},
			ForbiddenFiles:   ua.ForbiddenFiles,
			ForbiddenActions: []string{"git push", "git commit"},
			PreChangeEvidence: fmt.Sprintf(
				"no passing evidence exists yet for: %s; the verification commands below are the first evidence captured for this unit", a.Intent),
			MinimumImplementation:   "the smallest change within allowed_files that satisfies the unit behavior and advances its criteria_advanced; no unrelated refactor or cleanup",
			ReuseAnalysis:           "not independently investigated by hmarun init; the implementer must confirm before writing new code that no existing code already provides this behavior",
			Budgets:                 defaultUnitBudgets,
			PermissionClasses:       append([]string{}, defaultPermissionClasses...),
			ImplementerRoute:        defaultImplementerRoute,
			EscalationRoute:         defaultEscalationRoute,
			ValidatorRoute:          defaultValidatorRoute,
			IndependenceRequirement: "validator_route must be staffed by a distinct provider and model family from implementer_route",
			VerificationCommands:    verificationChecks,
			RequiredEvidence:        requiredEvidence,
			Dependencies:            dependencies,
			StopConditions:          []string{"unit max_attempts budget exhausted", "unit max_wall_minutes budget exhausted"},
			RollbackConditions:      []string{"a verification command fails after all allowed retries", "a change touches a forbidden file or action"},
			EscalationConditions: []string{
				"two consecutive verification command failures",
				"the behavior cannot be completed within allowed_files without touching a forbidden file or action",
			},
			NonGoals: unitNonGoals,
		})
	}

	plan := model.Plan{
		Schema:                   model.PlanSchema,
		RunID:                    runID,
		AcceptanceContractDigest: contractDigest,
		PolicyDigest:             defaultPolicyDigest,
		Units:                    units,
	}

	planBytes, err := json.Marshal(plan)
	if err != nil {
		return zero, "", zeroPlan, fmt.Errorf("marshaling templated plan: %w", err)
	}
	parsedPlan, _, err := model.ParsePlan(planBytes)
	if err != nil {
		return zero, "", zeroPlan, fmt.Errorf("hmarun templated an invalid plan (this is a hmarun bug, not an answers problem): %w", err)
	}

	if err := model.ValidatePlanAgainstContract(parsedPlan, parsedContract, contractDigest); err != nil {
		return zero, "", zeroPlan, fmt.Errorf("hmarun templated a plan that does not bind its own contract (this is a hmarun bug, not an answers problem): %w", err)
	}

	return parsedContract, contractDigest, parsedPlan, nil
}
