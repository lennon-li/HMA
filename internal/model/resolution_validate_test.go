package model

import "testing"

func validWaiverOperation() WaiverOperation {
	return WaiverOperation{
		Operation:      WaiverOperationGrant,
		Scope:          ScopeSelector{Kind: ScopeCriterion, Target: "C1"},
		Justification:  "accepted risk",
		Approver:       "lennon",
		Timestamp:      "2026-09-03T00:00:00Z",
		ChallengeNonce: "nonce-1",
	}
}

func TestValidateWaiverOperation(t *testing.T) {
	if err := validateWaiverOperation(nil); err != nil {
		t.Fatalf("nil operation rejected: %v", err)
	}
	if err := validateWaiverOperation(ptrWaiver(validWaiverOperation())); err != nil {
		t.Fatalf("valid operation rejected: %v", err)
	}

	for name, mutate := range map[string]func(*WaiverOperation){
		"unknown action":     func(w *WaiverOperation) { w.Operation = "APPROVE" },
		"empty action":       func(w *WaiverOperation) { w.Operation = "" },
		"stage scope":        func(w *WaiverOperation) { w.Scope.Kind = "stage" },
		"unknown scope kind": func(w *WaiverOperation) { w.Scope.Kind = "plan_unit" },
		"missing target":     func(w *WaiverOperation) { w.Scope.Target = "" },
		"missing rationale":  func(w *WaiverOperation) { w.Justification = "" },
		"missing approver":   func(w *WaiverOperation) { w.Approver = "" },
		"missing timestamp":  func(w *WaiverOperation) { w.Timestamp = "" },
		"missing nonce":      func(w *WaiverOperation) { w.ChallengeNonce = "" },
	} {
		t.Run(name, func(t *testing.T) {
			w := validWaiverOperation()
			mutate(&w)
			if err := validateWaiverOperation(&w); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func ptrWaiver(w WaiverOperation) *WaiverOperation { return &w }

func validFindingOverride() FindingDispositionOverride {
	return FindingDispositionOverride{
		FindingID:      "F1",
		Disposition:    FindingWaivable,
		Justification:  "accepted risk",
		Approver:       "lennon",
		Timestamp:      "2026-09-03T00:00:00Z",
		ChallengeNonce: "nonce-1",
	}
}

func TestValidateFindingOverride(t *testing.T) {
	if err := validateFindingOverride(nil); err != nil {
		t.Fatalf("nil override rejected: %v", err)
	}
	valid := validFindingOverride()
	if err := validateFindingOverride(&valid); err != nil {
		t.Fatalf("valid override rejected: %v", err)
	}

	for name, mutate := range map[string]func(*FindingDispositionOverride){
		"missing finding id":  func(f *FindingDispositionOverride) { f.FindingID = "" },
		"unknown disposition": func(f *FindingDispositionOverride) { f.Disposition = "MAYBE" },
		"escalates to block":  func(f *FindingDispositionOverride) { f.Disposition = FindingBlock },
		"missing rationale":   func(f *FindingDispositionOverride) { f.Justification = "" },
		"missing approver":    func(f *FindingDispositionOverride) { f.Approver = "" },
		"missing timestamp":   func(f *FindingDispositionOverride) { f.Timestamp = "" },
		"missing nonce":       func(f *FindingDispositionOverride) { f.ChallengeNonce = "" },
	} {
		t.Run(name, func(t *testing.T) {
			f := validFindingOverride()
			mutate(&f)
			if err := validateFindingOverride(&f); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestRecordMetadataIsZero(t *testing.T) {
	if !(RecordMetadata{}).IsZero() {
		t.Fatal("empty metadata not reported as zero")
	}
	if (RecordMetadata{RunID: "run"}).IsZero() {
		t.Fatal("populated metadata reported as zero")
	}
}

func TestScopeSelectorIsZero(t *testing.T) {
	if !(ScopeSelector{}).IsZero() {
		t.Fatal("empty selector not reported as zero")
	}
	if (ScopeSelector{Kind: ScopeCriterion}).IsZero() {
		t.Fatal("populated selector reported as zero")
	}
}
