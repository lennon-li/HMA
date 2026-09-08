package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidRecordKinds(t *testing.T) {
	for _, k := range []RecordKind{
		KindStageTransition, KindCriterion, KindFinding, KindWaiver,
		KindEvidence, KindReleaseRequest, KindOutcome,
	} {
		if !ValidRecordKind(k) {
			t.Errorf("ValidRecordKind(%q) = false, want true", k)
		}
	}
	if ValidRecordKind("NOPE") {
		t.Error(`ValidRecordKind("NOPE") = true, want false`)
	}
}

func TestValidStages(t *testing.T) {
	for _, s := range []Stage{
		StageGrounding, StageAcceptanceCriteria, StagePlanning,
		StageRouteSelection, StageImplementationAuthorization,
		StageImplementationReview, StageVerification,
		StageIndependentValidation, StageReleaseAndClosure,
	} {
		if !ValidStage(s) {
			t.Errorf("ValidStage(%q) = false, want true", s)
		}
	}
	if ValidStage("BOGUS_STAGE") {
		t.Error(`ValidStage("BOGUS_STAGE") = true, want false`)
	}
}

func TestValidControlStates(t *testing.T) {
	for _, c := range []StageControlState{
		ControlDraft, ControlReadyForReview, ControlApproved,
	} {
		if !ValidControlState(c) {
			t.Errorf("ValidControlState(%q) = false, want true", c)
		}
	}
	if ValidControlState("NOPE") {
		t.Error(`ValidControlState("NOPE") = true, want false`)
	}
	if ValidControlState(StageControlState(PacketInvalidated)) {
		t.Error(`ValidControlState("INVALIDATED") = true, want false; INVALIDATED is packet-control, not stage-control`)
	}
}

func TestValidPacketControlStates(t *testing.T) {
	for _, c := range []PacketControlState{PacketInvalidated} {
		if !ValidPacketControlState(c) {
			t.Errorf("ValidPacketControlState(%q) = false, want true", c)
		}
	}
	if ValidPacketControlState("NOPE") {
		t.Error(`ValidPacketControlState("NOPE") = true, want false`)
	}
}

func TestValidCriterionDispositions(t *testing.T) {
	for _, d := range []CriterionDisposition{
		CriterionPending, CriterionPassed, CriterionFailed, CriterionWaived,
	} {
		if !ValidCriterionDisposition(d) {
			t.Errorf("ValidCriterionDisposition(%q) = false, want true", d)
		}
	}
	if ValidCriterionDisposition("NOPE") {
		t.Error(`ValidCriterionDisposition("NOPE") = true, want false`)
	}
}

func TestValidFindingDispositions(t *testing.T) {
	for _, d := range []FindingDisposition{FindingBlock, FindingWaivable, FindingAdvisory} {
		if !ValidFindingDisposition(d) {
			t.Errorf("ValidFindingDisposition(%q) = false, want true", d)
		}
	}
	if ValidFindingDisposition("NOPE") {
		t.Error(`ValidFindingDisposition("NOPE") = true, want false`)
	}
}

func TestValidAdvisoryDispositions(t *testing.T) {
	for _, d := range []AdvisoryDisposition{AdvisoryAccept, AdvisoryPark, AdvisoryKill} {
		if !ValidAdvisoryDisposition(d) {
			t.Errorf("ValidAdvisoryDisposition(%q) = false, want true", d)
		}
	}
	if ValidAdvisoryDisposition("NOPE") {
		t.Error(`ValidAdvisoryDisposition("NOPE") = true, want false`)
	}
}

func TestValidTerminalOutcomes(t *testing.T) {
	for _, o := range []TerminalOutcome{
		OutcomeAborted, OutcomeBlocked, OutcomeUnknown, OutcomeFailed,
		OutcomePartial, OutcomeVerifiedWithWaivers, OutcomeVerifiedSuccess,
	} {
		if !ValidTerminalOutcome(o) {
			t.Errorf("ValidTerminalOutcome(%q) = false, want true", o)
		}
	}
	if ValidTerminalOutcome("NOPE") {
		t.Error(`ValidTerminalOutcome("NOPE") = true, want false`)
	}
}

func TestValidScopeKinds(t *testing.T) {
	for _, k := range []ScopeKind{
		ScopeCriterion, ScopeFinding, ScopeArtifact,
	} {
		if !ValidScopeKind(k) {
			t.Errorf("ValidScopeKind(%q) = false, want true", k)
		}
	}
	for _, k := range []string{"stage", "plan_unit", "closure", "NOPE"} {
		if ValidScopeKind(ScopeKind(k)) {
			t.Errorf("ValidScopeKind(%q) = true, want false", k)
		}
	}
}

func validRecord() *Record {
	return &Record{
		Kind:    KindCriterion,
		Version: 1,
		Metadata: RecordMetadata{
			RunID:      "run-1",
			Sequence:   1,
			HeadAnchor: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			Timestamp:  "2026-08-29T00:00:00Z",
			Actor:      "human-1",
		},
		Stage:   StageVerification,
		Control: ControlApproved,
		Criteria: []Criterion{
			{ID: "c1", Disposition: CriterionPassed},
			{ID: "c2", Disposition: CriterionWaived},
		},
		Findings: []Finding{
			{ID: "f1", Disposition: FindingWaivable},
			{ID: "f2", Disposition: FindingAdvisory, Advisory: AdvisoryAccept},
		},
		Waivers: []Waiver{
			{ID: "w1", Scope: ScopeSelector{Kind: ScopeCriterion, Target: "c2"}, Active: true, Actor: "human-1"},
		},
		Outcome: OutcomeVerifiedWithWaivers,
	}
}

func baseRecord() *Record {
	return &Record{
		Kind:    KindStageTransition,
		Version: 1,
		Metadata: RecordMetadata{
			RunID:      "run-1",
			Sequence:   1,
			HeadAnchor: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			Timestamp:  "2026-08-29T00:00:00Z",
			Actor:      "human-1",
		},
		Stage:   StageVerification,
		Control: ControlApproved,
	}
}

func validApproval() *ApprovalBinding {
	return &ApprovalBinding{
		RunID:                    "run-1",
		TransitionDigest:         "sha256:transition-1",
		CurrentStage:             StageVerification,
		ProposedTargetStage:      StageIndependentValidation,
		RepositoryIdentityDigest: "sha256:repo-identity",
		BaseRevisionDigest:       "sha256:base-rev",
		StageTimeDigest:          "sha256:stage-time",
		AcceptedPlanDigest:       "sha256:plan",
		RequiredEvidenceDigests:  []string{"sha256:evidence-1"},
		ActiveWaivers:            WaiverApplicabilitySet{},
		ChallengeNonce:           "nonce-1",
		Approver:                 "human-1",
		Timestamp:                "2026-08-29T00:00:00Z",
		ProducedHeadDigest:       "sha256:produced-head",
		DiffDigest:               "sha256:diff",
	}
}

func validEvidence() *EvidenceRef {
	return &EvidenceRef{
		Executable:         "/usr/local/go/bin/go",
		Argv:               []string{"test", "./..."},
		WorkingDir:         "/repo",
		StartTimestamp:     "2026-08-29T00:00:00Z",
		EndTimestamp:       "2026-08-29T00:00:05Z",
		ExitCode:           0,
		OutputDigest:       "sha256:output",
		RepositoryIdentity: "sha256:repo-identity",
		BaseRevision:       "sha256:base-rev",
		HeadRevision:       "sha256:produced-head",
		CapturingActor:     "hma-evidence-runner",
	}
}

func TestValidateRecordAcceptsValid(t *testing.T) {
	if err := ValidateRecord(validRecord()); err != nil {
		t.Fatalf("ValidateRecord(valid) = %v, want nil", err)
	}
}

func TestValidateRecordAcceptsVerifiedSuccess(t *testing.T) {
	r := baseRecord()
	// A record carrying a terminal outcome is an outcome record, not an
	// approved stage transition; the latter must carry the approval that
	// proves it.
	r.Kind = KindOutcome
	r.Criteria = []Criterion{{ID: "c1", Disposition: CriterionPassed}}
	r.Outcome = OutcomeVerifiedSuccess
	if err := ValidateRecord(r); err != nil {
		t.Fatalf("ValidateRecord(verified success) = %v, want nil", err)
	}
}

func TestValidateRecordAcceptsValidApprovalAndEvidence(t *testing.T) {
	r := baseRecord()
	r.Approval = validApproval()
	r.Evidence = validEvidence()
	if err := ValidateRecord(r); err != nil {
		t.Fatalf("ValidateRecord(valid approval+evidence) = %v, want nil", err)
	}
}

func TestValidateRecordAcceptsApprovalWithoutHeadDiff(t *testing.T) {
	r := baseRecord()
	a := validApproval()
	a.CurrentStage = StageGrounding
	a.ProposedTargetStage = StageAcceptanceCriteria
	a.ProducedHeadDigest = ""
	a.DiffDigest = ""
	r.Stage = StageGrounding
	r.Approval = a
	if err := ValidateRecord(r); err != nil {
		t.Fatalf("ValidateRecord(approval grounding without head/diff) = %v, want nil", err)
	}
}

func TestValidateRecordRejectsMissingChainMetadata(t *testing.T) {
	r := validRecord()
	r.Metadata = RecordMetadata{}
	if err := ValidateRecord(r); err == nil {
		t.Fatal("ValidateRecord(missing chain metadata) = nil, want error")
	}
}

func TestValidateRecordRejectsMismatchedChainMetadata(t *testing.T) {
	r := validRecord()
	r.Metadata.Sequence = 5
	r.Metadata.PredecessorHash = ""
	if err := ValidateRecord(r); err == nil {
		t.Fatal("ValidateRecord(non-genesis without predecessor) = nil, want error")
	}

	r2 := validRecord()
	r2.Metadata.Sequence = 1
	r2.Metadata.PredecessorHash = "sha256:should-not-be-here"
	if err := ValidateRecord(r2); err == nil {
		t.Fatal("ValidateRecord(genesis with predecessor) = nil, want error")
	}
}

func TestValidateRecordRejectsContradictions(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(*Record)
		wantSubstr string
	}{
		{"S8 version zero", func(r *Record) { r.Version = 0 }, "record version must be 1"},
		{"S8 version unknown", func(r *Record) { r.Version = 2 }, "record version must be 1"},
		{"unknown kind", func(r *Record) { r.Kind = "BOGUS_KIND" }, "unknown record kind"},
		{"unknown stage", func(r *Record) { r.Stage = "BOGUS_STAGE" }, "unknown stage"},
		{"unknown terminal outcome", func(r *Record) { r.Outcome = "BOGUS_OUTCOME" }, "unknown terminal outcome"},
		{"S2 control INVALIDATED is not stage-control", func(r *Record) {
			r.Control = StageControlState(PacketInvalidated)
		}, "unknown control state"},

		{"S1 stage waiver", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: "stage", Target: "GROUNDING"}, Active: true, Actor: "h"}}
		}, "waive a stage"},
		{"S1 plan_unit scope kind", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeKind("plan_unit"), Target: "u1"}, Active: true, Actor: "h"}}
		}, "invalid scope kind"},
		{"S1 closure scope kind", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeKind("closure"), Target: "k1"}, Active: true, Actor: "h"}}
		}, "invalid scope kind"},
		{"S1 invalid scope kind", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeKind("NOPE"), Target: "x"}, Active: true, Actor: "h"}}
		}, "invalid scope kind"},
		{"S1 active criterion waiver targets unknown criterion", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeCriterion, Target: "c9"}, Active: true, Actor: "h"}}
		}, "unknown criterion"},
		{"S1 active finding waiver targets non-waivable finding", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeFinding, Target: "f2"}, Active: true, Actor: "h"}}
		}, "non-waivable or unknown finding"},
		{"S1 active finding waiver targets unknown finding", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeFinding, Target: "f9"}, Active: true, Actor: "h"}}
		}, "non-waivable or unknown finding"},
		{"S1 missing scope target", func(r *Record) {
			r.Waivers = []Waiver{{ID: "wbad", Scope: ScopeSelector{Kind: ScopeCriterion, Target: ""}, Active: false, Actor: "h"}}
		}, "missing scope target"},
		{"S4 VERIFIED_SUCCESS with active waiver", func(r *Record) {
			r.Outcome = OutcomeVerifiedSuccess
		}, "VERIFIED_SUCCESS with an active waiver"},
		{"S4 VERIFIED_SUCCESS non-PASSED criterion", func(r *Record) {
			r.Waivers = nil
			r.Outcome = OutcomeVerifiedSuccess
		}, "VERIFIED_SUCCESS requires every criterion PASSED"},
		{"S4 VERIFIED_WITH_WAIVERS without active waiver", func(r *Record) {
			r.Waivers = nil
			r.Outcome = OutcomeVerifiedWithWaivers
		}, "VERIFIED_WITH_WAIVERS without an active waiver"},
		{"S4 VERIFIED_WITH_WAIVERS non-PASSED/WAIVED criterion", func(r *Record) {
			r.Criteria[1].Disposition = CriterionFailed
		}, "VERIFIED_WITH_WAIVERS requires every criterion PASSED or WAIVED"},
		{"S7 advisory set on non-advisory finding", func(r *Record) {
			r.Findings[0].Advisory = AdvisoryAccept
		}, "advisory disposition set for non-advisory finding"},
		{"S7 advisory finding missing disposition", func(r *Record) {
			r.Findings[1].Advisory = ""
		}, "advisory disposition missing or unknown"},
		{"S7 advisory finding unknown disposition", func(r *Record) {
			r.Findings[1].Advisory = AdvisoryDisposition("NOPE")
		}, "advisory disposition missing or unknown"},
		{"S3 approval missing universal run_id", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.RunID = ""
		}, "approval missing run_id"},
		{"S3 approval missing universal accepted_plan_digest", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.AcceptedPlanDigest = ""
		}, "approval missing accepted_plan_digest"},
		{"S3 approval current_stage not approved stage", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.CurrentStage = "BOGUS_STAGE"
		}, "approval current_stage"},
		{"S3 approval proposed_target_stage not approved stage", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.ProposedTargetStage = "BOGUS_STAGE"
		}, "approval proposed_target_stage"},
		{"S3 verification approval missing produced_head_digest", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.ProducedHeadDigest = ""
		}, "approval missing produced_head_digest"},
		{"S3 verification approval missing diff_digest", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.DiffDigest = ""
		}, "approval missing diff_digest"},
		{"S3 grounding approval must not bind produced_head_digest", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.CurrentStage = StageGrounding
			r.Approval.ProposedTargetStage = StageAcceptanceCriteria
		}, "approval must not bind produced_head_digest"},
		{"S3 grounding approval must not bind diff_digest", func(r *Record) {
			r.Approval = validApproval()
			r.Approval.CurrentStage = StageGrounding
			r.Approval.ProposedTargetStage = StageAcceptanceCriteria
			r.Approval.ProducedHeadDigest = ""
		}, "approval must not bind diff_digest"},
		{"S3 evidence missing executable", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.Executable = ""
		}, "evidence missing executable"},
		{"S3 evidence missing argv vector", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.Argv = nil
		}, "evidence missing argv vector"},
		{"S3 evidence missing working_dir", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.WorkingDir = ""
		}, "evidence missing working_dir"},
		{"S3 evidence missing start_timestamp", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.StartTimestamp = ""
		}, "evidence missing start_timestamp"},
		{"S3 evidence missing end_timestamp", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.EndTimestamp = ""
		}, "evidence missing end_timestamp"},
		{"S3 evidence missing repository_identity", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.RepositoryIdentity = ""
		}, "evidence missing repository_identity"},
		{"S3 evidence missing base_revision", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.BaseRevision = ""
		}, "evidence missing base_revision"},
		{"S3 evidence missing head_revision", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.HeadRevision = ""
		}, "evidence missing head_revision"},
		{"S3 evidence missing capturing_actor", func(r *Record) {
			r.Evidence = validEvidence()
			r.Evidence.CapturingActor = ""
		}, "evidence missing capturing_actor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRecord()
			tc.mutate(r)
			err := ValidateRecord(r)
			if err == nil {
				t.Fatalf("ValidateRecord(%s) = nil, want error containing %q", tc.name, tc.wantSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("ValidateRecord(%s) = %q, want error containing %q", tc.name, err.Error(), tc.wantSubstr)
			}
		})
	}
}

func TestVocabularyValidFixture(t *testing.T) {
	rec, err := loadFixture("vocabulary-valid.json")
	if err != nil {
		t.Fatalf("load valid fixture: %v", err)
	}
	if err := ValidateRecord(rec); err != nil {
		t.Fatalf("ValidateRecord(valid fixture) = %v, want nil", err)
	}
}

func TestVocabularyInvalidFixture(t *testing.T) {
	rec, err := loadFixture("vocabulary-invalid.json")
	if err != nil {
		t.Fatalf("load invalid fixture: %v", err)
	}
	err = ValidateRecord(rec)
	if err == nil {
		t.Fatal("ValidateRecord(invalid fixture) = nil, want error")
	}
	if want := "invalid scope kind"; !strings.Contains(err.Error(), want) {
		t.Fatalf("ValidateRecord(invalid fixture) = %q, want error containing %q", err.Error(), want)
	}
}

func loadFixture(name string) (*Record, error) {
	path := filepath.Join("..", "..", "testdata", "engine", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// TestValidateRecordRejectsApprovedTransitionWithoutApproval: an APPROVED
// stage_transition asserts a human approved leaving that stage. Without the
// binding there is nothing that proves it, and the chain would replay a stage
// change no approval backs.
func TestValidateRecordRejectsApprovedTransitionWithoutApproval(t *testing.T) {
	r := baseRecord()
	r.Approval = nil
	if err := ValidateRecord(r); err == nil {
		t.Fatal("approved stage_transition accepted with no approval binding")
	}
}

// TestValidateRecordRejectsTransitionFiledUnderAnotherStage: the record must
// be filed under the stage its approval was raised from. Otherwise a record
// could carry an approval for one transition while filing itself under
// another stage, and the projection would move the run somewhere no human
// approved.
func TestValidateRecordRejectsTransitionFiledUnderAnotherStage(t *testing.T) {
	for _, control := range []StageControlState{ControlApproved, ControlReadyForReview} {
		r := baseRecord()
		r.Control = control
		r.Approval = validApproval()
		r.Stage = StagePlanning // the approval is raised from VERIFICATION
		if err := ValidateRecord(r); err == nil {
			t.Fatalf("%s stage_transition accepted while filed under the wrong stage", control)
		}
	}
}

func TestValidateRecordAcceptsMatchingTransition(t *testing.T) {
	for _, control := range []StageControlState{ControlApproved, ControlReadyForReview} {
		r := baseRecord()
		r.Control = control
		r.Approval = validApproval()
		r.Stage = r.Approval.CurrentStage
		if err := ValidateRecord(r); err != nil {
			t.Fatalf("%s stage_transition rejected while correctly filed: %v", control, err)
		}
	}
}

// TestApprovalWorktreeDigestJSONRoundTrip: the section 6 amendment's optional
// worktree binding round-trips under its contract name and stays absent when
// unbound, so existing approval records are unaffected.
func TestApprovalWorktreeDigestJSONRoundTrip(t *testing.T) {
	a := validApproval()
	a.WorktreeDigest = "sha256:worktree-content"
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"worktree_digest":"sha256:worktree-content"`) {
		t.Fatalf("marshalled approval omits the worktree binding: %s", b)
	}
	var back ApprovalBinding
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.WorktreeDigest != a.WorktreeDigest {
		t.Fatalf("round trip lost the worktree binding: %+v", back)
	}

	unbound := validApproval()
	b2, err := json.Marshal(unbound)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b2), "worktree_digest") {
		t.Fatalf("unbound approval marshalled a worktree digest: %s", b2)
	}
}

// TestWaiverOperationHeadDiffJSONRoundTrip: the section 7 amendment's
// optional head and diff bindings round-trip under their contract names and
// stay absent when unbound.
func TestWaiverOperationHeadDiffJSONRoundTrip(t *testing.T) {
	w := WaiverOperation{
		Operation:      WaiverOperationGrant,
		Scope:          ScopeSelector{Kind: ScopeCriterion, Target: "C1"},
		Justification:  "accepted risk",
		Approver:       "lennon",
		Timestamp:      "2026-09-03T00:00:00Z",
		ChallengeNonce: "nonce-1",
		Head:           "head-the-human-waived-against",
		DiffDigest:     "diff-the-human-waived-against",
	}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"head":"head-the-human-waived-against"`, `"diff_digest":"diff-the-human-waived-against"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("marshalled waiver operation omits a binding (%s): %s", want, b)
		}
	}
	var back WaiverOperation
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Head != w.Head || back.DiffDigest != w.DiffDigest {
		t.Fatalf("round trip lost a binding: %+v", back)
	}

	w.Head = ""
	w.DiffDigest = ""
	b2, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b2), "head") || strings.Contains(string(b2), "diff_digest") {
		t.Fatalf("unbound waiver operation marshalled a binding: %s", b2)
	}
}
