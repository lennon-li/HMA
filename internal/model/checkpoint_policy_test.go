package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func testCheckpointPolicy() CheckpointPolicy {
	return CheckpointPolicy{
		Schema: CheckpointPolicySchema,
		Rules: []CheckpointPolicyRule{
			{ID: "cli-surface", Glob: "cmd/hma/**", Classes: []string{"public_interface"}},
			{ID: "store-schema", Glob: "internal/model/**", Classes: []string{"persisted_schema"}},
			{ID: "docs", Glob: "docs/**", Classes: []string{"ordinary"}},
		},
	}
}

// A valid checkpoint policy parses and digests deterministically.
func TestCheckpointPolicyValidParses(t *testing.T) {
	p := testCheckpointPolicy()
	raw := mustJSON(p)
	got, digest, err := ParseCheckpointPolicy(raw)
	if err != nil {
		t.Fatal(err)
	}
	if digest == "" {
		t.Fatal("empty digest")
	}
	if len(got.Rules) != 3 {
		t.Fatalf("got %d rules, want 3", len(got.Rules))
	}
}

// A contract carrying a checkpoint_policy_digest parses.
func TestContractWithCheckpointPolicyDigestValidParses(t *testing.T) {
	_, policyDigest, err := ParseCheckpointPolicy(mustJSON(testCheckpointPolicy()))
	if err != nil {
		t.Fatal(err)
	}
	c := testContract()
	c.CheckpointPolicyDigest = policyDigest
	got, _, err := ParseAcceptanceContract(mustJSON(c))
	if err != nil {
		t.Fatal(err)
	}
	if got.CheckpointPolicyDigest != policyDigest {
		t.Fatalf("checkpoint policy digest %q != %q", got.CheckpointPolicyDigest, policyDigest)
	}
}

// The literal single-class-ordinary catch-all refuses the document; a "**"
// rule with an extra class, or a non-catch-all glob, is not detected as
// such and still parses.
func TestCheckpointPolicyCatchAllOrdinaryRefused(t *testing.T) {
	p := CheckpointPolicy{Schema: CheckpointPolicySchema, Rules: []CheckpointPolicyRule{
		{ID: "catch-all", Glob: "**", Classes: []string{"ordinary"}},
	}}
	if _, _, err := ParseCheckpointPolicy(mustJSON(p)); err == nil {
		t.Fatal("catch-all ordinary rule accepted")
	}

	notDetected := CheckpointPolicy{Schema: CheckpointPolicySchema, Rules: []CheckpointPolicyRule{
		{ID: "catch-all-plus", Glob: "**", Classes: []string{"ordinary", "security"}},
	}}
	if _, _, err := ParseCheckpointPolicy(mustJSON(notDetected)); err != nil {
		t.Fatalf("\"**\" rule with an extra class refused: %v", err)
	}

	// Near-misses that must NOT be refused: v1 detects the literal glob "**"
	// only (Phil's review, finding M-3), so a single-segment "*" or an
	// equivalent "**/*" with only the "ordinary" class still parses, even
	// though both are broad enough to make trigger T8 unreachable in
	// practice.
	for name, glob := range map[string]string{
		"single-segment-star": "*",
		"double-star-slash":   "**/*",
	} {
		p := CheckpointPolicy{Schema: CheckpointPolicySchema, Rules: []CheckpointPolicyRule{
			{ID: "near-miss", Glob: glob, Classes: []string{"ordinary"}},
		}}
		if _, _, err := ParseCheckpointPolicy(mustJSON(p)); err != nil {
			t.Errorf("%s: glob %q with only [ordinary] refused: %v", name, glob, err)
		}
	}

	// A "**" rule with a duplicate-class list ["ordinary", "ordinary"] is
	// current behavior: the check requires len(r.Classes) == 1, and this
	// list has length 2, so it is not detected as the catch-all and parses.
	// Whether that is correct per architecture §16's wording "whose only
	// class is `ordinary`" is genuinely ambiguous: a list of two identical
	// entries arguably still has only one *distinct* class, which would
	// argue for refusal, but the code's literal len == 1 test does not see
	// it that way. This test documents the current (accepting) behavior; it
	// does not take a position on which reading §16 intends. Flagged, not
	// resolved, per Phil's review finding M-3 — no production-code change
	// made here.
	dup := CheckpointPolicy{Schema: CheckpointPolicySchema, Rules: []CheckpointPolicyRule{
		{ID: "catch-all-dup", Glob: "**", Classes: []string{"ordinary", "ordinary"}},
	}}
	if _, _, err := ParseCheckpointPolicy(mustJSON(dup)); err != nil {
		t.Fatalf("\"**\" rule with duplicate-class [ordinary, ordinary] refused (current behavior is to accept): %v", err)
	}
}

func TestCheckpointPolicyUnknownClassRefused(t *testing.T) {
	p := testCheckpointPolicy()
	p.Rules[0].Classes = []string{"misc"}
	if _, _, err := ParseCheckpointPolicy(mustJSON(p)); err == nil {
		t.Fatal("unknown class accepted")
	}
}

func TestCheckpointPolicyValidation(t *testing.T) {
	for name, edit := range map[string]func(*CheckpointPolicy){
		"bad-schema":    func(p *CheckpointPolicy) { p.Schema = "other" },
		"no-rules":      func(p *CheckpointPolicy) { p.Rules = nil },
		"empty-id":      func(p *CheckpointPolicy) { p.Rules[0].ID = "" },
		"duplicate-id":  func(p *CheckpointPolicy) { p.Rules[1].ID = p.Rules[0].ID },
		"empty-glob":    func(p *CheckpointPolicy) { p.Rules[0].Glob = "" },
		"empty-classes": func(p *CheckpointPolicy) { p.Rules[0].Classes = nil },
	} {
		p := testCheckpointPolicy()
		edit(&p)
		if ValidateCheckpointPolicy(p) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

// Every checkpoint policy field is required and non-null, per the existing
// decodeCanonicalInput rule.
func TestCheckpointPolicyFieldsRequired(t *testing.T) {
	raw := string(mustJSON(testCheckpointPolicy()))
	for name, doc := range map[string]string{
		"no-schema":   strings.Replace(raw, `"schema":"hma.checkpoint-policy/v1",`, ``, 1),
		"null-schema": strings.Replace(raw, `"schema":"hma.checkpoint-policy/v1"`, `"schema":null`, 1),
		"no-glob":     strings.Replace(raw, `"glob":"cmd/hma/**",`, ``, 1),
		"null-glob":   strings.Replace(raw, `"glob":"cmd/hma/**"`, `"glob":null`, 1),
		"no-classes":  strings.Replace(raw, `"classes":["public_interface"]`, ``, 1),
	} {
		if doc == raw {
			t.Fatalf("%s: fixture edit did not apply", name)
		}
		if _, _, err := ParseCheckpointPolicy([]byte(doc)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

// Unknown fields and duplicate/case-variant keys are refused, same as the
// contract and plan decoder.
func TestCheckpointPolicyDecodingIsStrict(t *testing.T) {
	raw, _ := json.Marshal(testCheckpointPolicy())
	body := string(raw[:len(raw)-1])
	for name, doc := range map[string]string{
		"unknown-field": body + `,"extra":1}`,
		"trailing":      string(raw) + ` {}`,
	} {
		if _, _, err := ParseCheckpointPolicy([]byte(doc)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
