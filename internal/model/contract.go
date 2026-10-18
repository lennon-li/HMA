package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

// AcceptanceContractSchema names the acceptance contract format (Gate A v2
// §2.2, v3 §2.1). A contract is one immutable document per approved criteria
// set. It holds criteria definitions and ceilings, never results: criterion
// dispositions stay in core criterion records.
const AcceptanceContractSchema = "hma.acceptance-contract/v1"

// PlanSchema names the accepted plan format (Gate A v3 §2.2, v4 §2). Its
// digest is the approvals' accepted_plan_digest.
const PlanSchema = "hma.plan/v1"

// Check is one explicit executable and argument vector; no shell string is
// representable.
type Check struct {
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
	WorkingDir string   `json:"working_dir"`
}

// ContractCriterion defines one acceptance criterion. Changing its
// statement, check, threshold, or evidence requirement changes the contract
// digest.
type ContractCriterion struct {
	ID                  string `json:"id"`
	Statement           string `json:"statement"`
	Check               Check  `json:"check"`
	Threshold           string `json:"threshold"`
	EvidenceRequirement string `json:"evidence_requirement"`
}

// ContractBudgets are the run-wide ceilings. A plan unit's budgets must stay
// within them; a conflict fails closed.
type ContractBudgets struct {
	MaxAttempts    int    `json:"max_attempts"`
	MaxWallMinutes int    `json:"max_wall_minutes"`
	CostCeiling    int64  `json:"cost_ceiling"`
	CostUnit       string `json:"cost_unit"`
}

// AcceptanceContract is hma.acceptance-contract/v1.
type AcceptanceContract struct {
	Schema             string              `json:"schema"`
	RunID              string              `json:"run_id"`
	Intent             string              `json:"intent"`
	NonGoals           []string            `json:"non_goals"`
	Criteria           []ContractCriterion `json:"criteria"`
	Risk               RouteRiskLevel      `json:"risk"`
	Deliverables       []string            `json:"deliverables"`
	Budgets            ContractBudgets     `json:"budgets"`
	PermittedRoutes    []string            `json:"permitted_routes"`
	PermissionEnvelope []string            `json:"permission_envelope"`
	PolicyDigest       string              `json:"policy_digest"`
}

// RouteGrant is the host's projection of the route selected for a unit: its
// stable identity (the digest of the exact route with no approval reference;
// plan route strings are such identities), its permission classes, and the
// core route approval digest it claims. A lease checks it against the plan
// and the core chain; it grants nothing by itself.
type RouteGrant struct {
	Identity          string   `json:"identity"`
	PermissionClasses []string `json:"permission_classes"`
	ApprovalDigest    string   `json:"approval_digest"`
}

// UnitBudgets are one plan unit's §10 budgets plus the lease ceilings.
type UnitBudgets struct {
	MaxComplexity  int   `json:"max_complexity"`
	MaxFiles       int   `json:"max_files"`
	MaxDiffLines   int   `json:"max_diff_lines"`
	MaxToolSteps   int   `json:"max_tool_steps"`
	MaxRetries     int   `json:"max_retries"`
	MaxAttempts    int   `json:"max_attempts"`
	MaxWallMinutes int   `json:"max_wall_minutes"`
	CostCeiling    int64 `json:"cost_ceiling"`
}

// PlanUnit carries the architecture §10 plan-unit fields.
type PlanUnit struct {
	ID                      string      `json:"id"`
	Behavior                string      `json:"behavior"`
	CriteriaAdvanced        []string    `json:"criteria_advanced"`
	AllowedFiles            []string    `json:"allowed_files"`
	AllowedActions          []string    `json:"allowed_actions"`
	ForbiddenFiles          []string    `json:"forbidden_files"`
	ForbiddenActions        []string    `json:"forbidden_actions"`
	PreChangeEvidence       string      `json:"pre_change_evidence"`
	MinimumImplementation   string      `json:"minimum_implementation"`
	ReuseAnalysis           string      `json:"reuse_analysis"`
	Budgets                 UnitBudgets `json:"budgets"`
	PermissionClasses       []string    `json:"permission_classes"`
	ImplementerRoute        string      `json:"implementer_route"`
	EscalationRoute         string      `json:"escalation_route"`
	ValidatorRoute          string      `json:"validator_route"`
	IndependenceRequirement string      `json:"independence_requirement"`
	VerificationCommands    []Check     `json:"verification_commands"`
	RequiredEvidence        []string    `json:"required_evidence"`
	Dependencies            []string    `json:"dependencies"`
	StopConditions          []string    `json:"stop_conditions"`
	RollbackConditions      []string    `json:"rollback_conditions"`
	EscalationConditions    []string    `json:"escalation_conditions"`
	NonGoals                []string    `json:"non_goals"`
}

// Plan is hma.plan/v1.
type Plan struct {
	Schema                   string     `json:"schema"`
	RunID                    string     `json:"run_id"`
	AcceptanceContractDigest string     `json:"acceptance_contract_digest"`
	PolicyDigest             string     `json:"policy_digest"`
	Units                    []PlanUnit `json:"units"`
}

// MaxWallMinutesLimit bounds every wall-time ceiling (one year), so a
// ceiling always converts to a time.Duration without overflow.
const MaxWallMinutesLimit = 525600

// ErrInvalidDocument is wrapped by every contract and plan parse or
// validation error.
var ErrInvalidDocument = errors.New("invalid HMA document")

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidDocument, fmt.Sprintf(format, a...))
}

// ParseAcceptanceContract strictly decodes and validates a contract and
// returns it with its digest.
func ParseAcceptanceContract(raw []byte) (AcceptanceContract, string, error) {
	var c AcceptanceContract
	if err := decodeCanonicalInput(raw, &c); err != nil {
		return c, "", err
	}
	if err := ValidateAcceptanceContract(c); err != nil {
		return c, "", err
	}
	d, err := documentDigest(c)
	return c, d, err
}

// ParsePlan strictly decodes and validates a plan and returns it with its
// digest.
func ParsePlan(raw []byte) (Plan, string, error) {
	var p Plan
	if err := decodeCanonicalInput(raw, &p); err != nil {
		return p, "", err
	}
	if err := ValidatePlan(p); err != nil {
		return p, "", err
	}
	d, err := documentDigest(p)
	return p, d, err
}

// documentDigest is the SHA-256 of the document's canonical JSON: the
// struct re-serialized with Go's fixed field order, so the digest covers
// exactly what HMA read.
func documentDigest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ContractDigest returns the canonical digest of a validated contract.
func ContractDigest(c AcceptanceContract) (string, error) { return documentDigest(c) }

// PlanDigest returns the canonical digest of a validated plan.
func PlanDigest(p Plan) (string, error) { return documentDigest(p) }

// decodeCanonicalInput decodes exactly one JSON object into v. Every object
// key must be plain lowercase ASCII and unique, and every key must name a
// field: encoding/json would otherwise fold case and let the last of two
// duplicates win.
func decodeCanonicalInput(raw []byte, v any) error {
	// encoding/json silently replaces invalid UTF-8 and lone UTF-16
	// surrogate escapes with U+FFFD, so the digest would cover text other
	// than the bytes presented.
	if !utf8.Valid(raw) {
		return invalid("document is not valid UTF-8")
	}
	if err := checkSurrogates(raw); err != nil {
		return err
	}
	if err := checkKeys(raw); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return invalid("%v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return invalid("trailing data after document")
	}
	return requireAllKeys(raw, v)
}

// requireAllKeys refuses a document that omits a field or sets it to null:
// Go would read either as the zero value, so an absent retry budget would
// silently become 0 and an absent list empty. Every field of the contract
// and plan formats is required; an empty list is written as [] and a zero
// count as 0. It compares the raw document with the typed value re-encoded,
// object by object.
func requireAllKeys(raw []byte, v any) error {
	var doc, typed any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return invalid("%v", err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return invalid("%v", err)
	}
	if err := json.Unmarshal(b, &typed); err != nil {
		return invalid("%v", err)
	}
	var walk func(path string, d, t any) error
	walk = func(path string, d, t any) error {
		switch tv := t.(type) {
		case map[string]any:
			dm, ok := d.(map[string]any)
			if !ok {
				return invalid("%s must be an object", path)
			}
			for k, sub := range tv {
				dsub, present := dm[k]
				if !present || dsub == nil {
					return invalid("%s.%s is required and may not be null", path, k)
				}
				if err := walk(path+"."+k, dsub, sub); err != nil {
					return err
				}
			}
		case []any:
			da, ok := d.([]any)
			if !ok || len(da) != len(tv) {
				return invalid("%s must be an array", path)
			}
			for i := range tv {
				if err := walk(fmt.Sprintf("%s[%d]", path, i), da[i], tv[i]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk("$", doc, typed)
}

// checkSurrogates refuses a \u escape that is a lone or misordered UTF-16
// surrogate. A high surrogate must be followed directly by a low one.
func checkSurrogates(raw []byte) error {
	hex4 := func(i int) (rune, bool) {
		if i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
			return 0, false
		}
		var r rune
		for _, c := range raw[i+2 : i+6] {
			r <<= 4
			switch {
			case c >= '0' && c <= '9':
				r |= rune(c - '0')
			case c >= 'a' && c <= 'f':
				r |= rune(c-'a') + 10
			case c >= 'A' && c <= 'F':
				r |= rune(c-'A') + 10
			default:
				return 0, false
			}
		}
		return r, true
	}
	inString := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if !inString {
			if c == '"' {
				inString = true
			}
			continue
		}
		switch c {
		case '"':
			inString = false
		case '\\':
			r, ok := hex4(i)
			if !ok {
				i++ // skip the escaped character
				continue
			}
			switch {
			case r >= 0xd800 && r <= 0xdbff:
				lo, ok := hex4(i + 6)
				if !ok || lo < 0xdc00 || lo > 0xdfff {
					return invalid("lone UTF-16 surrogate escape")
				}
				i += 11
			case r >= 0xdc00 && r <= 0xdfff:
				return invalid("lone UTF-16 surrogate escape")
			default:
				i += 5
			}
		}
	}
	return nil
}

func checkKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var walk func() error
	walk = func() error {
		t, err := dec.Token()
		if err != nil {
			return invalid("%v", err)
		}
		switch t {
		case json.Delim('{'):
			seen := map[string]bool{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return invalid("%v", err)
				}
				k := kt.(string)
				if seen[k] || !plainKey(k) {
					return invalid("key %q is duplicated or not plain lowercase", k)
				}
				seen[k] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case json.Delim('['):
			for dec.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		}
		return nil
	}
	return walk()
}

func plainKey(k string) bool {
	if k == "" {
		return false
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

func nonEmpty(s string) bool { return strings.TrimSpace(s) != "" }

func validCheck(c Check) bool {
	if !nonEmpty(c.Executable) || c.WorkingDir == "" || c.Argv == nil || strings.ContainsAny(c.Executable, "|&;<>()$`\\\n\r") {
		return false
	}
	for _, a := range c.Argv {
		if a == "" {
			return false
		}
	}
	return true
}

// ValidateAcceptanceContract checks a contract's form. Criterion ids are
// unique, non-empty, and may not contain "/".
func ValidateAcceptanceContract(c AcceptanceContract) error {
	if c.Schema != AcceptanceContractSchema || !nonEmpty(c.RunID) || !nonEmpty(c.Intent) {
		return invalid("contract schema, run_id, and intent are required")
	}
	if len(c.Criteria) == 0 {
		return invalid("contract needs at least one criterion")
	}
	seen := map[string]bool{}
	for _, cr := range c.Criteria {
		if !nonEmpty(cr.ID) || strings.Contains(cr.ID, "/") || seen[cr.ID] {
			return invalid("criterion id %q is empty, contains /, or is duplicated", cr.ID)
		}
		seen[cr.ID] = true
		if !nonEmpty(cr.Statement) || !nonEmpty(cr.Threshold) || !nonEmpty(cr.EvidenceRequirement) || !validCheck(cr.Check) {
			return invalid("criterion %s needs a statement, check, threshold, and evidence requirement", cr.ID)
		}
	}
	if !ValidRouteRiskLevel(c.Risk) {
		return invalid("contract risk %q is not valid", c.Risk)
	}
	b := c.Budgets
	if b.MaxAttempts <= 0 || b.MaxWallMinutes <= 0 || b.CostCeiling <= 0 || !nonEmpty(b.CostUnit) {
		return invalid("contract budgets must be positive with a cost unit")
	}
	if b.MaxWallMinutes > MaxWallMinutesLimit {
		return invalid("contract max_wall_minutes exceeds %d", MaxWallMinutesLimit)
	}
	if len(c.PermittedRoutes) == 0 || len(c.PermissionEnvelope) == 0 || !nonEmpty(c.PolicyDigest) {
		return invalid("contract needs permitted routes, a permission envelope, and a policy digest")
	}
	for _, p := range c.PermissionEnvelope {
		if !ValidRoutePolicyPermission(p) {
			return invalid("contract permission %q is not a permission class", p)
		}
	}
	for _, list := range [][]string{c.NonGoals, c.Deliverables, c.PermittedRoutes, c.PermissionEnvelope} {
		for _, s := range list {
			if !nonEmpty(s) {
				return invalid("contract list entries must be non-empty")
			}
		}
	}
	return nil
}

// ValidatePlan checks a plan's form: unique unit ids, positive lease
// ceilings, dependencies that exist, and an acyclic dependency graph.
func ValidatePlan(p Plan) error {
	if p.Schema != PlanSchema || !nonEmpty(p.RunID) || !nonEmpty(p.PolicyDigest) {
		return invalid("plan schema, run_id, and policy_digest are required")
	}
	if len(p.Units) == 0 {
		return invalid("plan needs at least one unit")
	}
	if !nonEmpty(p.AcceptanceContractDigest) {
		return invalid("plan must bind an acceptance contract")
	}
	units := map[string]*PlanUnit{}
	for i := range p.Units {
		u := &p.Units[i]
		if !nonEmpty(u.ID) || units[u.ID] != nil {
			return invalid("unit id %q is empty or duplicated", u.ID)
		}
		units[u.ID] = u
		if err := validateUnit(u); err != nil {
			return err
		}
	}
	for _, u := range p.Units {
		for _, d := range u.Dependencies {
			if units[d] == nil || d == u.ID {
				return invalid("unit %s depends on unknown or self unit %q", u.ID, d)
			}
		}
	}
	// Depth-first search for a cycle.
	state := map[string]int{}
	var visit func(id string) bool
	visit = func(id string) bool {
		switch state[id] {
		case 1:
			return false
		case 2:
			return true
		}
		state[id] = 1
		for _, d := range units[id].Dependencies {
			if !visit(d) {
				return false
			}
		}
		state[id] = 2
		return true
	}
	ids := make([]string, 0, len(units))
	for id := range units {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !visit(id) {
			return invalid("plan dependencies contain a cycle through %s", id)
		}
	}
	return nil
}

// validateUnit requires every architecture §10 field. Every list is
// required and non-empty except dependencies, which must be present and may
// be empty (no prior unit).
func validateUnit(u *PlanUnit) error {
	for name, v := range map[string]string{"behavior": u.Behavior, "pre_change_evidence": u.PreChangeEvidence,
		"minimum_implementation": u.MinimumImplementation, "reuse_analysis": u.ReuseAnalysis,
		"implementer_route": u.ImplementerRoute, "escalation_route": u.EscalationRoute, "validator_route": u.ValidatorRoute,
		"independence_requirement": u.IndependenceRequirement} {
		if !nonEmpty(v) {
			return invalid("unit %s needs %s", u.ID, name)
		}
	}
	for name, list := range map[string][]string{"criteria_advanced": u.CriteriaAdvanced, "allowed_files": u.AllowedFiles,
		"allowed_actions": u.AllowedActions, "forbidden_files": u.ForbiddenFiles, "forbidden_actions": u.ForbiddenActions,
		"permission_classes": u.PermissionClasses, "required_evidence": u.RequiredEvidence, "stop_conditions": u.StopConditions,
		"rollback_conditions": u.RollbackConditions, "escalation_conditions": u.EscalationConditions, "non_goals": u.NonGoals} {
		if len(list) == 0 {
			return invalid("unit %s needs a non-empty %s", u.ID, name)
		}
		for _, x := range list {
			if !nonEmpty(x) {
				return invalid("unit %s has an empty %s entry", u.ID, name)
			}
		}
	}
	if u.Dependencies == nil {
		return invalid("unit %s must declare dependencies (empty list for none)", u.ID)
	}
	for _, p := range u.PermissionClasses {
		if !ValidRoutePolicyPermission(p) {
			return invalid("unit %s permission %q is not a permission class", u.ID, p)
		}
	}
	if len(u.VerificationCommands) == 0 {
		return invalid("unit %s needs verification commands", u.ID)
	}
	for _, c := range u.VerificationCommands {
		if !validCheck(c) {
			return invalid("unit %s has an invalid verification command", u.ID)
		}
	}
	b := u.Budgets
	if b.MaxComplexity <= 0 || b.MaxFiles <= 0 || b.MaxDiffLines <= 0 || b.MaxToolSteps <= 0 || b.MaxRetries < 0 ||
		b.MaxAttempts <= 0 || b.MaxWallMinutes <= 0 || b.MaxWallMinutes > MaxWallMinutesLimit || b.CostCeiling <= 0 {
		return invalid("unit %s budgets must be positive (retries may be zero) and wall time at most %d minutes", u.ID, MaxWallMinutesLimit)
	}
	return nil
}

// ValidatePlanAgainstContract checks a plan against the contract it binds:
// the contract digest, policy digest, criteria advanced, and that every
// unit's ceilings stay within the contract's. A conflict fails closed.
func ValidatePlanAgainstContract(p Plan, c AcceptanceContract, contractDigest string) error {
	if p.AcceptanceContractDigest != contractDigest || p.RunID != c.RunID || p.PolicyDigest != c.PolicyDigest {
		return invalid("plan does not bind this contract, run, and policy")
	}
	criteria := map[string]bool{}
	for _, cr := range c.Criteria {
		criteria[cr.ID] = true
	}
	routes := map[string]bool{}
	for _, r := range c.PermittedRoutes {
		routes[r] = true
	}
	envelope := map[string]bool{}
	for _, p := range c.PermissionEnvelope {
		envelope[p] = true
	}
	covered := map[string]bool{}
	for _, u := range p.Units {
		for _, r := range []string{u.ImplementerRoute, u.EscalationRoute, u.ValidatorRoute} {
			if !routes[r] {
				return invalid("unit %s route %q is not a contract permitted route", u.ID, r)
			}
		}
		for _, perm := range u.PermissionClasses {
			if !envelope[perm] {
				return invalid("unit %s permission %q is outside the contract envelope", u.ID, perm)
			}
		}
		for _, id := range u.CriteriaAdvanced {
			if !criteria[id] {
				return invalid("unit %s advances unknown criterion %q", u.ID, id)
			}
			covered[id] = true
		}
		if u.Budgets.MaxAttempts > c.Budgets.MaxAttempts || u.Budgets.MaxWallMinutes > c.Budgets.MaxWallMinutes || u.Budgets.CostCeiling > c.Budgets.CostCeiling {
			return invalid("unit %s budgets exceed the contract ceilings", u.ID)
		}
	}
	for id := range criteria {
		if !covered[id] {
			return invalid("no unit advances criterion %q", id)
		}
	}
	return nil
}

// ContractCriterionIDs returns the contract's criterion ids, sorted.
func ContractCriterionIDs(c AcceptanceContract) []string {
	out := make([]string, 0, len(c.Criteria))
	for _, cr := range c.Criteria {
		out = append(out, cr.ID)
	}
	sort.Strings(out)
	return out
}
