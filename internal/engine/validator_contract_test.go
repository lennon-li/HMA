package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

// validatorContractCase is one recorded/stubbed validator-contract fixture
// row. Rule names the required fixture-matrix rule the row proves.
type validatorContractCase struct {
	Name    string                   `json:"name"`
	Rule    string                   `json:"rule"`
	Request ValidatorContractRequest `json:"request"`
	Expect  ValidatorContractResult  `json:"expect"`
}

// requiredValidRules and requiredInvalidRules restate the approved fixture
// matrix independently of the fixture files, so a dropped rule fails.
var requiredValidRules = []string{"V1", "V2", "V3", "V4", "V5", "V6"}

var requiredInvalidRules = []string{"I1", "I2", "I3", "I4", "I5", "I6", "I7", "KR1"}

// assertValidatorNeverAuthorizes proves the authority boundary structurally:
// a recorded validator result never advances, never creates a criterion,
// never grants a waiver, never names a stage target, and never emits a
// terminal outcome.
func assertValidatorNeverAuthorizes(t *testing.T, got ValidatorContractResult) {
	t.Helper()
	if got.MachineAdvanced {
		t.Fatal("validator contract reported machine advancement")
	}
	if !got.RequiresFreshHumanApproval {
		t.Fatal("validator contract result did not require fresh human approval")
	}
	if len(got.CreatedCriterionIDs) != 0 {
		t.Fatalf("validator created acceptance criteria: %v", got.CreatedCriterionIDs)
	}
	if len(got.GrantedWaiverIDs) != 0 {
		t.Fatalf("validator granted waivers: %v", got.GrantedWaiverIDs)
	}
	if got.RewindTarget != "" {
		t.Fatalf("validator reported a stage target %q", got.RewindTarget)
	}
	if got.TerminalOutcome != "" {
		t.Fatalf("validator reported a terminal outcome %q", got.TerminalOutcome)
	}
}

func assertRuleCoverage(t *testing.T, cases []validatorContractCase, required []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, tc := range cases {
		if tc.Rule == "" {
			t.Fatalf("fixture case %q declares no rule", tc.Name)
		}
		seen[tc.Rule] = true
	}
	for _, r := range required {
		if !seen[r] {
			t.Errorf("fixture matrix rule %s has no case", r)
		}
		delete(seen, r)
	}
	for r := range seen {
		t.Errorf("fixture declares rule %s that is not in the approved matrix", r)
	}
}

// TestValidatorContractValidFixtures drives every recorded/stubbed valid
// contract case through strict decoding.
func TestValidatorContractValidFixtures(t *testing.T) {
	var cases []validatorContractCase
	loadFixture(t, "../../testdata/validator-contract/valid.json", &cases)
	if len(cases) == 0 {
		t.Fatal("valid validator-contract fixture is empty")
	}
	assertRuleCoverage(t, cases, requiredValidRules)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got := EvaluateValidatorContract(tc.Request)
			if !reflect.DeepEqual(got, tc.Expect) {
				t.Fatalf("EvaluateValidatorContract\n got: %+v\nwant: %+v", got, tc.Expect)
			}
			if got.Decision != DecisionLegalPendingApproval {
				t.Fatalf("valid case decided %q", got.Decision)
			}
			if got.Reason != ReasonNone {
				t.Fatalf("valid case carried rejection reason %q", got.Reason)
			}
			assertValidatorNeverAuthorizes(t, got)
		})
	}
}

// TestValidatorContractInvalidFixtures drives every rejection rule through
// strict decoding and proves a rejected result classifies no finding.
func TestValidatorContractInvalidFixtures(t *testing.T) {
	var cases []validatorContractCase
	loadFixture(t, "../../testdata/validator-contract/invalid.json", &cases)
	if len(cases) == 0 {
		t.Fatal("invalid validator-contract fixture is empty")
	}
	assertRuleCoverage(t, cases, requiredInvalidRules)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got := EvaluateValidatorContract(tc.Request)
			if !reflect.DeepEqual(got, tc.Expect) {
				t.Fatalf("EvaluateValidatorContract\n got: %+v\nwant: %+v", got, tc.Expect)
			}
			if got.Decision != DecisionRejected {
				t.Fatalf("invalid case decided %q", got.Decision)
			}
			if got.Reason == ReasonNone {
				t.Fatal("rejection carried no reason")
			}
			if len(got.BlockingFindingIDs)+len(got.KernelViolationFindingIDs)+
				len(got.UnresolvedWaivableFindingIDs)+len(got.AdvisoryFindingIDs) != 0 {
				t.Fatalf("rejected result classified findings: %+v", got)
			}
			assertValidatorNeverAuthorizes(t, got)
		})
	}
}

// TestValidatorContractNeverSelfAuthorizes sweeps the finding vocabulary,
// reference kinds, and every self-authorization claim to prove no recorded
// input can make this evaluator advance, waive, or approve anything.
func TestValidatorContractNeverSelfAuthorizes(t *testing.T) {
	dispositions := []model.FindingDisposition{
		"", model.FindingBlock, model.FindingWaivable, model.FindingAdvisory,
		model.FindingDisposition("NOT_A_DISPOSITION"),
	}
	advisories := []model.AdvisoryDisposition{
		"", model.AdvisoryAccept, model.AdvisoryPark, model.AdvisoryKill,
		model.AdvisoryDisposition("NOT_AN_ADVISORY"),
	}
	refKinds := []FindingRefKind{
		"", RefApprovedCriterion, RefDeclaredBudget, RefKernelRule,
		RefOutOfCriteria, FindingRefKind("NOT_A_REF_KIND"),
	}
	refIDs := []string{"", "C-1", "B-1", "K-1", "C-404"}
	for _, d := range dispositions {
		for _, a := range advisories {
			for _, k := range refKinds {
				for _, id := range refIDs {
					for _, claim := range []bool{false, true} {
						req := ValidatorContractRequest{
							ApprovedCriterionIDs:              []string{"C-1"},
							DeclaredBudgetIDs:                 []string{"B-1"},
							ImplementerRouteAttestationDigest: "sha256:impl",
							ValidatorRouteAttestationDigest:   "sha256:val",
							DeclaredIndependent:               true,
							Findings: []RecordedFinding{{
								ID:                  "F-1",
								Disposition:         d,
								Advisory:            a,
								RefKind:             k,
								RefID:               id,
								ClaimsWaived:        claim,
								ClaimsHumanApproval: claim,
							}},
							ClaimsMachineAdvancement: claim,
						}
						got := EvaluateValidatorContract(req)
						assertValidatorNeverAuthorizes(t, got)
						if claim && got.Decision != DecisionRejected {
							t.Fatalf("self-authorizing claim accepted for %+v", req)
						}
						if got.Decision == DecisionLegalPendingApproval &&
							len(got.UnresolvedWaivableFindingIDs) > 0 &&
							len(got.GrantedWaiverIDs) > 0 {
							t.Fatalf("waivable finding auto-resolved for %+v", req)
						}
					}
				}
			}
		}
	}
}

// TestValidatorContractRejectsNonIndependentRoutes proves route-independence
// failure is structural and cannot be bypassed by a declaration alone.
func TestValidatorContractRejectsNonIndependentRoutes(t *testing.T) {
	base := ValidatorContractRequest{
		ApprovedCriterionIDs:              []string{"C-1"},
		ImplementerRouteAttestationDigest: "sha256:impl",
		ValidatorRouteAttestationDigest:   "sha256:val",
		DeclaredIndependent:               true,
	}
	if got := EvaluateValidatorContract(base); got.Decision != DecisionLegalPendingApproval {
		t.Fatalf("independent route rejected: %+v", got)
	}
	same := base
	same.ValidatorRouteAttestationDigest = same.ImplementerRouteAttestationDigest
	if got := EvaluateValidatorContract(same); got.Reason != ReasonValidatorNotIndependent {
		t.Fatalf("identical route digests accepted: %+v", got)
	}
	declaredFalse := base
	declaredFalse.DeclaredIndependent = false
	if got := EvaluateValidatorContract(declaredFalse); got.Reason != ReasonValidatorNotIndependent {
		t.Fatalf("false independence declaration accepted: %+v", got)
	}
	for _, missing := range []ValidatorContractRequest{
		{ApprovedCriterionIDs: []string{"C-1"}, ValidatorRouteAttestationDigest: "sha256:val", DeclaredIndependent: true},
		{ApprovedCriterionIDs: []string{"C-1"}, ImplementerRouteAttestationDigest: "sha256:impl", DeclaredIndependent: true},
		{ApprovedCriterionIDs: []string{"C-1"}, DeclaredIndependent: true},
	} {
		if got := EvaluateValidatorContract(missing); got.Reason != ReasonMissingRouteAttestation {
			t.Fatalf("missing route attestation accepted: %+v", got)
		}
	}
}

// TestValidatorContractDoesNotMutateInput proves the evaluator is pure with
// respect to caller-owned slices.
func TestValidatorContractDoesNotMutateInput(t *testing.T) {
	req := ValidatorContractRequest{
		ApprovedCriterionIDs:              []string{"C-2", "C-1"},
		DeclaredBudgetIDs:                 []string{"B-2", "B-1"},
		ImplementerRouteAttestationDigest: "sha256:impl",
		ValidatorRouteAttestationDigest:   "sha256:val",
		DeclaredIndependent:               true,
		Findings: []RecordedFinding{
			{ID: "F-2", Disposition: model.FindingBlock, RefKind: RefApprovedCriterion, RefID: "C-1"},
			{ID: "F-1", Disposition: model.FindingWaivable, RefKind: RefDeclaredBudget, RefID: "B-2"},
		},
	}
	before, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	_ = EvaluateValidatorContract(req)
	after, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("input mutated\nbefore: %s\n after: %s", before, after)
	}
}
