package model

// CheckpointPolicySchema names the checkpoint policy format (architecture
// §16 "Checkpoint policy"). A contract's optional checkpoint_policy_digest
// (§9) binds one document of this schema.
const CheckpointPolicySchema = "hma.checkpoint-policy/v1"

// checkpointPolicyClasses is the closed class vocabulary (architecture §16
// "Checkpoint policy"). An unknown class refuses the document.
var checkpointPolicyClasses = map[string]bool{
	"public_interface": true,
	"persisted_schema": true,
	"protocol":         true,
	"external_effect":  true,
	"irreversible":     true,
	"security":         true,
	"approval":         true,
	"routing":          true,
	"evidence":         true,
	"closure":          true,
	"ordinary":         true,
}

// CheckpointPolicyRule matches changed paths to classes. Glob is a
// slash-separated path pattern: "*" matches one path segment, "**" matches
// any number of segments. Matching and class union over a changed-path
// manifest are classifier logic (Phase 3), not part of this type.
type CheckpointPolicyRule struct {
	ID      string   `json:"id"`
	Glob    string   `json:"glob"`
	Classes []string `json:"classes"`
}

// CheckpointPolicy is hma.checkpoint-policy/v1. Every field is required;
// this document has no optional fields of its own.
type CheckpointPolicy struct {
	Schema string                 `json:"schema"`
	Rules  []CheckpointPolicyRule `json:"rules"`
}

// ParseCheckpointPolicy strictly decodes and validates a checkpoint policy
// and returns it with its digest.
func ParseCheckpointPolicy(raw []byte) (CheckpointPolicy, string, error) {
	var p CheckpointPolicy
	if err := decodeCanonicalInput(raw, &p); err != nil {
		return p, "", err
	}
	if err := ValidateCheckpointPolicy(p); err != nil {
		return p, "", err
	}
	d, err := documentDigest(p)
	return p, d, err
}

// ValidateCheckpointPolicy checks a checkpoint policy's form: schema,
// non-empty and unique rule ids, a non-empty glob and class list per rule,
// and the closed class vocabulary. A rule whose glob is "**" and whose only
// class is "ordinary" refuses the document, because it would make trigger
// T8 ("cannot be classified") unreachable by configuration (architecture
// §16 "Checkpoint policy"). v1 refuses that literal catch-all only: an
// equivalent set of broad globs, and a "**" rule with additional classes
// beyond "ordinary", are not detected.
func ValidateCheckpointPolicy(p CheckpointPolicy) error {
	if p.Schema != CheckpointPolicySchema {
		return invalid("checkpoint policy schema is required")
	}
	if len(p.Rules) == 0 {
		return invalid("checkpoint policy needs at least one rule")
	}
	seen := map[string]bool{}
	for _, r := range p.Rules {
		if !nonEmpty(r.ID) || seen[r.ID] {
			return invalid("checkpoint policy rule id %q is empty or duplicated", r.ID)
		}
		seen[r.ID] = true
		if !nonEmpty(r.Glob) {
			return invalid("checkpoint policy rule %s needs a glob", r.ID)
		}
		if len(r.Classes) == 0 {
			return invalid("checkpoint policy rule %s needs at least one class", r.ID)
		}
		for _, c := range r.Classes {
			if !checkpointPolicyClasses[c] {
				return invalid("checkpoint policy rule %s has unknown class %q", r.ID, c)
			}
		}
		if r.Glob == "**" && len(r.Classes) == 1 && r.Classes[0] == "ordinary" {
			return invalid("checkpoint policy rule %s is a catch-all ordinary rule, which would make trigger T8 unreachable", r.ID)
		}
	}
	return nil
}
