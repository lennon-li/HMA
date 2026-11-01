package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/lennon-li/HMA/internal/model"
)

// CheckpointTrigger is one of the architecture §16 checkpoint-trigger
// identifiers (T1-T8).
type CheckpointTrigger string

const (
	CheckpointContractOrPermission CheckpointTrigger = "CONTRACT_OR_PERMISSION_CHANGED"
	CheckpointHighRisk             CheckpointTrigger = "HIGH_OR_CRITICAL_RISK"
	CheckpointInterface            CheckpointTrigger = "INTERFACE_SCHEMA_OR_PROTOCOL"
	CheckpointExternalEffect       CheckpointTrigger = "IRREVERSIBLE_OR_EXTERNAL_EFFECT"
	CheckpointControlLogic         CheckpointTrigger = "SECURITY_APPROVAL_ROUTING_EVIDENCE_OR_CLOSURE"
	CheckpointMilestone            CheckpointTrigger = "APPROVED_MATERIAL_MILESTONE"
	CheckpointFinalClosure         CheckpointTrigger = "FINAL_OUTCOME_CLOSURE"
	CheckpointUnclassified         CheckpointTrigger = "UNCLASSIFIED"
)

// IssuingLevel is the minimum review level the classifier derives:
// per_change < checkpoint < final. Values match internal/host's
// IssuingReason strings so a later phase can convert directly; this package
// cannot import internal/host (internal/host already imports internal/engine).
type IssuingLevel string

const (
	IssuingLevelPerChange  IssuingLevel = "per_change"
	IssuingLevelCheckpoint IssuingLevel = "checkpoint"
	IssuingLevelFinal      IssuingLevel = "final"
)

// FinalAuditOutcomes is the set of terminal outcomes that are "final outcome
// closure" for the checkpoint triggers (architecture §16): the
// success-claiming outcomes only. FAILED, BLOCKED, UNKNOWN, and ABORTED
// claim no success and are not in this set.
var FinalAuditOutcomes = map[model.TerminalOutcome]bool{
	model.OutcomeVerifiedSuccess:     true,
	model.OutcomeVerifiedWithWaivers: true,
	model.OutcomePartial:             true,
}

// ChangedPath is the classifier's projection of one observed changed path:
// path, status, old and new mode, and the binary flag. A rename is
// represented as a delete of the old path and an add of the new path.
type ChangedPath struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	OldMode string `json:"old_mode"`
	NewMode string `json:"new_mode"`
	Binary  bool   `json:"binary"`
}

// CheckpointBaseline is a contract digest plus a unit's permission classes,
// taken at one point in time: either what a checkpoint-grade review covered
// (Baseline) or the unit's state now (Current).
type CheckpointBaseline struct {
	ContractDigest    string   `json:"contract_digest"`
	PermissionClasses []string `json:"permission_classes"`
}

// CheckpointInput is everything the checkpoint classifier reads. Contract is
// the bound contract document, already proven equal to BoundContractDigest
// by the caller. Policy nil means the run has no checkpoint policy, so every
// changed path is UNCLASSIFIED (T8). Manifest is HMA's own complete
// changed-path capture.
type CheckpointInput struct {
	Contract      *model.AcceptanceContract
	Policy        *model.CheckpointPolicy
	Unit          model.PlanUnit
	Baseline      CheckpointBaseline
	Current       CheckpointBaseline
	Manifest      []ChangedPath
	Stage         model.Stage
	TargetOutcome model.TerminalOutcome
	HostRisk      model.RouteRiskLevel
	Disputed      bool
}

// CheckpointDecision is the classifier's deterministic output. Triggers and
// UnclassifiedPaths are sorted and unique; the decision does not depend on
// the order of CheckpointInput.Manifest or CheckpointInput.Policy.Rules.
type CheckpointDecision struct {
	Level             IssuingLevel
	Triggers          []CheckpointTrigger
	UnclassifiedPaths []string
	FamilySeparation  bool
	RiskFloor         model.RouteRiskLevel
	RequiredRisk      model.RouteRiskLevel
	PolicyDigest      string
	ManifestDigest    string
}

// ClassifyCheckpoint derives the checkpoint-trigger decision for one unit
// (architecture §16 "Checkpoint triggers" and "Checkpoint classifier"). It is
// pure: standard-library-only, no I/O, and it never mutates in or anything
// in reaches from it (Manifest entries, Policy rules, pointers).
func ClassifyCheckpoint(in CheckpointInput) CheckpointDecision {
	triggers := map[CheckpointTrigger]bool{}

	if in.Current.ContractDigest != in.Baseline.ContractDigest || !stringSetEqual(in.Current.PermissionClasses, in.Baseline.PermissionClasses) {
		triggers[CheckpointContractOrPermission] = true
	}

	pathClasses, unclassified := classifyChangedPaths(in.Policy, in.Manifest)
	hasAnyClass := func(names ...string) bool {
		for _, cls := range pathClasses {
			for _, n := range names {
				if cls[n] {
					return true
				}
			}
		}
		return false
	}

	t3 := hasAnyClass("public_interface", "persisted_schema", "protocol")
	if t3 {
		triggers[CheckpointInterface] = true
	}
	t4Observed := hasAnyClass("external_effect", "irreversible")
	t4Allowed := !stringSliceContains(in.Unit.PermissionClasses, model.PermissionNoExternalEffect)
	if t4Observed || t4Allowed {
		triggers[CheckpointExternalEffect] = true
	}
	t5 := hasAnyClass("security", "approval", "routing", "evidence", "closure")
	if t5 {
		triggers[CheckpointControlLogic] = true
	}
	if in.Unit.MaterialMilestone && in.Stage == model.StageIndependentValidation {
		triggers[CheckpointMilestone] = true
	}
	t7 := FinalAuditOutcomes[in.TargetOutcome] || in.Stage == model.StageReleaseAndClosure
	if t7 {
		triggers[CheckpointFinalClosure] = true
	}
	if len(unclassified) > 0 {
		triggers[CheckpointUnclassified] = true
	}

	riskFloor := model.RouteRiskLow
	if t3 || t4Observed || t5 {
		riskFloor = model.RouteRiskHigh
	}

	var contractRisk model.RouteRiskLevel
	policyDigest := ""
	if in.Contract != nil {
		contractRisk = in.Contract.Risk
		policyDigest = in.Contract.CheckpointPolicyDigest
	}

	combinedRisk := maxRouteRisk(contractRisk, in.HostRisk, riskFloor)
	if routeRiskRank(combinedRisk) >= routeRiskRank(model.RouteRiskHigh) {
		triggers[CheckpointHighRisk] = true
	}
	requiredRisk := maxRouteRisk(contractRisk, riskFloor)
	familySeparation := routeRiskRank(combinedRisk) >= routeRiskRank(model.RouteRiskHigh) || in.Disputed || t7

	level := IssuingLevelPerChange
	switch {
	case t7:
		level = IssuingLevelFinal
	case in.Stage == model.StageIndependentValidation && len(triggers) > 0:
		level = IssuingLevelCheckpoint
	}

	sortedTriggers := make([]CheckpointTrigger, 0, len(triggers))
	for t := range triggers {
		sortedTriggers = append(sortedTriggers, t)
	}
	sort.Slice(sortedTriggers, func(i, j int) bool { return sortedTriggers[i] < sortedTriggers[j] })

	return CheckpointDecision{
		Level:             level,
		Triggers:          sortedTriggers,
		UnclassifiedPaths: unclassified,
		FamilySeparation:  familySeparation,
		RiskFloor:         riskFloor,
		RequiredRisk:      requiredRisk,
		PolicyDigest:      policyDigest,
		ManifestDigest:    manifestDigest(in.Manifest),
	}
}

// classifyChangedPaths unions, per path, the classes of every policy rule
// that matches it (duplicate classes within or across rules have no extra
// effect, since the union is a set). A nil policy, or a path no rule
// matches, is unclassified. The returned unclassified list is sorted.
func classifyChangedPaths(policy *model.CheckpointPolicy, manifest []ChangedPath) (map[string]map[string]bool, []string) {
	classes := make(map[string]map[string]bool, len(manifest))
	for _, cp := range manifest {
		cls := map[string]bool{}
		if policy != nil {
			for _, rule := range policy.Rules {
				if matchCheckpointGlob(rule.Glob, cp.Path) {
					for _, c := range rule.Classes {
						cls[c] = true
					}
				}
			}
		}
		classes[cp.Path] = cls
	}
	unclassifiedSet := map[string]bool{}
	for path, cls := range classes {
		if len(cls) == 0 {
			unclassifiedSet[path] = true
		}
	}
	unclassified := make([]string, 0, len(unclassifiedSet))
	for path := range unclassifiedSet {
		unclassified = append(unclassified, path)
	}
	sort.Strings(unclassified)
	return classes, unclassified
}

// matchCheckpointGlob matches a slash-separated path against a
// slash-separated glob where "*" matches exactly one path segment and "**"
// matches any number of segments (including zero). It is a plain dynamic-
// program over segments, not a regexp, so it has no pathological backtracking
// cost for repeated "**" segments.
func matchCheckpointGlob(glob, path string) bool {
	g := strings.Split(glob, "/")
	p := strings.Split(path, "/")
	dp := make([][]bool, len(g)+1)
	for i := range dp {
		dp[i] = make([]bool, len(p)+1)
	}
	for i := len(g); i >= 0; i-- {
		for j := len(p); j >= 0; j-- {
			if i == len(g) {
				dp[i][j] = j == len(p)
				continue
			}
			switch seg := g[i]; {
			case seg == "**":
				match := dp[i+1][j]
				if !match && j < len(p) {
					match = dp[i][j+1]
				}
				dp[i][j] = match
			case j < len(p) && (seg == "*" || seg == p[j]):
				dp[i][j] = dp[i+1][j+1]
			default:
				dp[i][j] = false
			}
		}
	}
	return dp[0][0]
}

func stringSetEqual(a, b []string) bool {
	sa, sb := map[string]bool{}, map[string]bool{}
	for _, v := range a {
		sa[v] = true
	}
	for _, v := range b {
		sb[v] = true
	}
	if len(sa) != len(sb) {
		return false
	}
	for v := range sa {
		if !sb[v] {
			return false
		}
	}
	return true
}

func stringSliceContains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// maxRouteRisk returns the risk level with the highest routeRiskRank among
// its arguments. An empty ("" / unknown) level ranks lowest.
func maxRouteRisk(risks ...model.RouteRiskLevel) model.RouteRiskLevel {
	best := model.RouteRiskLevel("")
	bestRank := -1
	for _, r := range risks {
		if rank := routeRiskRank(r); rank > bestRank {
			bestRank, best = rank, r
		}
	}
	return best
}

// manifestDigest is the SHA-256 hex digest of the JSON encoding of manifest,
// sorted into a deterministic order first, without mutating the caller's
// slice.
func manifestDigest(manifest []ChangedPath) string {
	sorted := make([]ChangedPath, len(manifest))
	copy(sorted, manifest)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Status != b.Status {
			return a.Status < b.Status
		}
		if a.OldMode != b.OldMode {
			return a.OldMode < b.OldMode
		}
		if a.NewMode != b.NewMode {
			return a.NewMode < b.NewMode
		}
		return !a.Binary && b.Binary
	})
	b, err := json.Marshal(sorted)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
