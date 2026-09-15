package model

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
)

func TestApprovalOutcomeTargets(t *testing.T) {
	for _, tc := range []struct {
		stage   Stage
		outcome TerminalOutcome
		valid   bool
	}{
		{StageIndependentValidation, "", true}, {"", OutcomeAborted, true}, {"", OutcomePartial, true},
		{StageIndependentValidation, OutcomeAborted, false}, {"", "", false}, {"", "BOGUS", false},
	} {
		a := validApproval()
		a.ProposedTargetStage, a.ProposedTargetOutcome = tc.stage, tc.outcome
		if err := validateApproval(a); (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}

func TestOutcomeApprovalRecordBinding(t *testing.T) {
	for _, field := range []string{"valid", "kind", "outcome", "stage", "control"} {
		t.Run(field, func(t *testing.T) {
			r := baseRecord()
			r.Kind, r.Outcome, r.Approval = KindOutcome, OutcomeAborted, validApproval()
			r.Approval.ProposedTargetStage = ""
			r.Approval.ProposedTargetOutcome = OutcomeAborted
			switch field {
			case "kind":
				r.Kind = KindStageTransition
			case "outcome":
				r.Outcome = OutcomePartial
			case "stage":
				r.Stage = StagePlanning
			case "control":
				r.Control = ControlReadyForReview
			}
			if err := ValidateRecord(r); (err == nil) != (field == "valid") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

// Golden bytes were captured from the pre-A1 ApprovalBinding shape. The record
// golden also protects the exact hashing input used by the store (empty anchor).
func TestLegacyApprovalSerializationGolden(t *testing.T) {
	const approval = `{"run_id":"run-1","transition_digest":"sha256:transition-1","current_stage":"VERIFICATION","proposed_target_stage":"INDEPENDENT_VALIDATION","repository_identity_digest":"sha256:repo-identity","base_revision_digest":"sha256:base-rev","stage_time_digest":"sha256:stage-time","accepted_plan_digest":"sha256:plan","required_evidence_digests":["sha256:evidence-1"],"challenge_nonce":"nonce-1","approver":"human-1","timestamp":"2026-08-29T00:00:00Z","produced_head_digest":"sha256:produced-head","diff_digest":"sha256:diff"}`
	b, err := json.Marshal(validApproval())
	if err != nil || string(b) != approval {
		t.Fatalf("legacy approval changed: %s, %v", b, err)
	}
	r := baseRecord()
	r.Metadata.HeadAnchor = ""
	r.Approval = validApproval()
	const prefix = `{"kind":"stage_transition","version":1,"metadata":{"run_id":"run-1","sequence":1,"predecessor_hash":"","head_anchor":"","timestamp":"2026-08-29T00:00:00Z","actor":"human-1"},"stage":"VERIFICATION","control":"APPROVED","approval":`
	b, err = json.Marshal(r)
	golden := prefix + approval + "}"
	if err != nil || string(b) != golden {
		t.Fatalf("legacy record changed: %s, %v", b, err)
	}
	if got, want := fmt.Sprintf("%x", sha256.Sum256(b)), fmt.Sprintf("%x", sha256.Sum256([]byte(golden))); got != want {
		t.Fatalf("record digest = %s, want %s", got, want)
	}
	a := validApproval()
	a.ProposedTargetStage = ""
	a.ProposedTargetOutcome = OutcomeAborted
	b, err = json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip ApprovalBinding
	if err := json.Unmarshal(b, &roundTrip); err != nil || roundTrip.ProposedTargetOutcome != OutcomeAborted || roundTrip.ProposedTargetStage != "" {
		t.Fatalf("round trip = %+v, %v", roundTrip, err)
	}
	a.ProposedTargetOutcome = OutcomePartial
	changed, err := json.Marshal(a)
	if err != nil || sha256.Sum256(b) == sha256.Sum256(changed) {
		t.Fatal("outcome not covered by serialization digest")
	}
}
