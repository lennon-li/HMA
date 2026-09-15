package model

import "testing"

func TestClassificationRecordValidation(t *testing.T) {
	for _, field := range []string{"valid", "kind", "missing payload", "missing approval", "unit", "finding", "outcome", "target", "run", "stage", "control", "nonce", "timestamp", "head"} {
		t.Run(field, func(t *testing.T) {
			r := baseRecord()
			r.Kind = KindClassification
			r.UnitID = "U1"
			r.Classification = &Classification{Outcome: OutcomeBlocked, FindingID: "F1"}
			r.Approval = validApproval()
			r.Approval.UnitID = r.UnitID
			r.Approval.Classification = &Classification{Outcome: OutcomeBlocked, FindingID: "F1"}
			r.Approval.ProposedTargetStage = ""
			r.Approval.ProposedTargetOutcome = OutcomeBlocked
			switch field {
			case "kind":
				r.Kind = KindFinding
			case "missing payload":
				r.Classification = nil
			case "missing approval":
				r.Approval = nil
			case "unit":
				r.UnitID = "other"
			case "finding":
				r.Classification.FindingID = " "
			case "outcome":
				r.Classification.Outcome = "free text"
			case "target":
				r.Approval.ProposedTargetOutcome = OutcomeFailed
			case "run":
				r.Approval.RunID = "other"
			case "stage":
				r.Stage = StagePlanning
			case "control":
				r.Control = ControlDraft
			case "nonce":
				r.Approval.ChallengeNonce = ""
			case "timestamp":
				r.Approval.Timestamp = "bad"
			case "head":
				r.Approval.ProducedHeadDigest = ""
			}
			if err := ValidateRecord(r); (err == nil) != (field == "valid") {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}
