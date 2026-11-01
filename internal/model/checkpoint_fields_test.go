package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// preGateEContractDigest is the contract digest Phil independently computed
// against a pre-Gate-E (`git archive HEAD`) checkout for the same
// testContract fixture. Pinning it, instead of comparing two fresh
// computations against each other, is what actually catches a future
// digest-changing regression (Phil's review, finding M-1).
const preGateEContractDigest = "3891a798704b77d765126b51c6affa21239e741ba5ec4ed4a82319a6675ace36"

// preGateEPlanDigest is the plan digest Phil independently computed against
// the same pre-Gate-E checkout for the testPlan fixture (finding M-1).
const preGateEPlanDigest = "8882eff28fcbe9e40e746993eee29e9a2b4fb6899a2fa6002f1577090be6b24d"

// §9: a contract fixture's digest is unchanged by adding the optional
// checkpoint_policy_digest field's absence. Comparing against a contract
// marshaled before the field existed is not possible in Go source, so this
// proves the weaker, equivalent fact: a contract with the field left at its
// zero value serializes to the same bytes, and therefore the same digest,
// as one that never mentions the field in its JSON. It also pins the digest
// against the real pre-Gate-E value, so a future change that alters the
// digest of an existing document is caught here, not just internal
// consistency between two fresh computations.
func TestContractDigestUnchangedWithoutPolicyDigest(t *testing.T) {
	c := testContract()
	raw := mustJSON(c)
	if strings.Contains(string(raw), "checkpoint_policy_digest") {
		t.Fatal("zero-valued field present in marshaled contract")
	}
	_, digest, err := ParseAcceptanceContract(raw)
	if err != nil {
		t.Fatal(err)
	}
	if digest != preGateEContractDigest {
		t.Fatalf("digest %s != pinned pre-Gate-E digest %s", digest, preGateEContractDigest)
	}

	withField := testContract()
	withField.CheckpointPolicyDigest = ""
	rawWithField := mustJSON(withField)
	if string(rawWithField) != string(raw) {
		t.Fatalf("zero-valued field changed the serialized bytes")
	}
	_, digest2, err := ParseAcceptanceContract(rawWithField)
	if err != nil {
		t.Fatal(err)
	}
	if digest != digest2 {
		t.Fatalf("digest changed: %s != %s", digest, digest2)
	}
}

// §10: same proof for a plan unit's optional material_milestone field, also
// pinned against the real pre-Gate-E digest (finding M-1).
func TestPlanDigestUnchangedWithoutMilestone(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))
	p := testPlan(cd)
	raw := mustJSON(p)
	if strings.Contains(string(raw), "material_milestone") {
		t.Fatal("zero-valued field present in marshaled plan")
	}
	_, digest, err := ParsePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if digest != preGateEPlanDigest {
		t.Fatalf("digest %s != pinned pre-Gate-E digest %s", digest, preGateEPlanDigest)
	}

	withField := testPlan(cd)
	withField.Units[0].MaterialMilestone = false
	rawWithField := mustJSON(withField)
	if string(rawWithField) != string(raw) {
		t.Fatalf("zero-valued field changed the serialized bytes")
	}
	_, digest2, err := ParsePlan(rawWithField)
	if err != nil {
		t.Fatal(err)
	}
	if digest != digest2 {
		t.Fatalf("digest changed: %s != %s", digest, digest2)
	}
}

// A present checkpoint_policy_digest with a real value parses, and the
// digest changes from the zero-value document.
func TestContractWithNonZeroPolicyDigestChangesDigest(t *testing.T) {
	c := testContract()
	_, zeroDigest, err := ParseAcceptanceContract(mustJSON(c))
	if err != nil {
		t.Fatal(err)
	}
	c.CheckpointPolicyDigest = "abc123"
	got, digest, err := ParseAcceptanceContract(mustJSON(c))
	if err != nil {
		t.Fatal(err)
	}
	if got.CheckpointPolicyDigest != "abc123" {
		t.Fatalf("checkpoint_policy_digest not round-tripped: %q", got.CheckpointPolicyDigest)
	}
	if digest == zeroDigest {
		t.Fatal("digest unchanged by a non-zero checkpoint_policy_digest")
	}
}

// A present material_milestone: true parses, and the digest changes from
// the zero-value document.
func TestPlanWithTrueMilestoneChangesDigest(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))
	p := testPlan(cd)
	_, zeroDigest, err := ParsePlan(mustJSON(p))
	if err != nil {
		t.Fatal(err)
	}
	p.Units[0].MaterialMilestone = true
	got, digest, err := ParsePlan(mustJSON(p))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Units[0].MaterialMilestone {
		t.Fatal("material_milestone not round-tripped")
	}
	if digest == zeroDigest {
		t.Fatal("digest unchanged by material_milestone: true")
	}
}

// An explicit empty string, null, or false for one of these optional
// fields is refused; only key absence encodes the zero meaning.
func TestExplicitZeroOptionalFieldRefused(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))

	contractRaw := string(mustJSON(testContract()))
	planRaw := string(mustJSON(testPlan(cd)))

	for name, doc := range map[string]string{
		"contract-empty-string": strings.Replace(contractRaw, `"policy_digest":"policy"}`,
			`"policy_digest":"policy","checkpoint_policy_digest":""}`, 1),
		"contract-null": strings.Replace(contractRaw, `"policy_digest":"policy"}`,
			`"policy_digest":"policy","checkpoint_policy_digest":null}`, 1),
	} {
		if doc == contractRaw {
			t.Fatalf("%s: fixture edit did not apply", name)
		}
		if _, _, err := ParseAcceptanceContract([]byte(doc)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}

	doc := strings.Replace(planRaw, `"non_goals":["refactor"]`,
		`"non_goals":["refactor"],"material_milestone":false`, 1)
	if doc == planRaw {
		t.Fatal("plan-false: fixture edit did not apply")
	}
	if _, _, err := ParsePlan([]byte(doc)); err == nil {
		t.Error("plan-false accepted")
	}

	docNull := strings.Replace(planRaw, `"non_goals":["refactor"]`,
		`"non_goals":["refactor"],"material_milestone":null`, 1)
	if docNull == planRaw {
		t.Fatal("plan-null: fixture edit did not apply")
	}
	if _, _, err := ParsePlan([]byte(docNull)); err == nil {
		t.Error("plan-null accepted")
	}
}

// TestExplicitZeroOptionalFieldRefused above injects an explicit
// material_milestone:false into unit 0 only, which does not exercise the
// "[]" iteration in walkExplicitZero across every unit of a plan. This test
// builds a plan with 3 units and injects the explicit-zero value into each
// unit in turn, confirming every single injection is refused (Phil's review,
// finding M-2).
func TestExplicitZeroMaterialMilestoneRefusedAcrossAllUnits(t *testing.T) {
	_, cd, _ := ParseAcceptanceContract(mustJSON(testContract()))
	p := testPlan(cd)
	u3 := p.Units[1]
	u3.ID = "U3"
	u3.Dependencies = []string{}
	p.Units = append(p.Units, u3)
	if len(p.Units) != 3 {
		t.Fatalf("fixture needs exactly 3 units, got %d", len(p.Units))
	}

	for i := range p.Units {
		var doc map[string]any
		if err := json.Unmarshal(mustJSON(p), &doc); err != nil {
			t.Fatal(err)
		}
		units, ok := doc["units"].([]any)
		if !ok || len(units) != 3 {
			t.Fatalf("unit %d: unexpected units shape", i)
		}
		unit, ok := units[i].(map[string]any)
		if !ok {
			t.Fatalf("unit %d: unexpected unit shape", i)
		}
		if _, present := unit["material_milestone"]; present {
			t.Fatalf("unit %d: fixture already carries material_milestone", i)
		}
		unit["material_milestone"] = false
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := ParsePlan(raw); err == nil {
			t.Errorf("unit %d: explicit material_milestone:false accepted", i)
		}
	}
}
