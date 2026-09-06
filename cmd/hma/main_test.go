package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}

// seedCriterion commits one criterion record so resolution has a real target.
func seedCriterion(t *testing.T, dir, runID string) *store.Store {
	t.Helper()
	s := store.New(dir)
	rec := model.Record{
		Kind:     model.KindCriterion,
		Version:  1,
		Metadata: model.RecordMetadata{RunID: runID, Sequence: 1, Timestamp: "2026-09-03T00:00:00Z", Actor: "host"},
		Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}},
	}
	if err := s.Append(&rec); err != nil {
		t.Fatal(err)
	}
	return s
}

func waiverInput(runID, nonce string) map[string]any {
	return map[string]any{
		"run_id": runID,
		"waiver_operation": map[string]any{
			"operation":       string(model.WaiverOperationGrant),
			"scope_selector":  map[string]any{"kind": string(model.ScopeCriterion), "target": "C1"},
			"justification":   "accepted risk",
			"approver":        "lennon",
			"timestamp":       "2026-09-03T00:00:00Z",
			"challenge_nonce": nonce,
		},
	}
}

func TestResolveRejectsSecondGrantOfSameScope(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(dir, "waiver.json")

	writeJSON(t, input, waiverInput("run", "nonce-1"))
	if err := runResolve([]string{"--input", input, "--store", dir}); err != nil {
		t.Fatalf("first grant rejected: %v", err)
	}

	writeJSON(t, input, waiverInput("run", "nonce-2"))
	err := runResolve([]string{"--input", input, "--store", dir})
	if err == nil {
		t.Fatal("the same scope was waived twice")
	}
	if err.Error() != "SCOPE_ALREADY_WAIVED" {
		t.Fatalf("reason = %q, want SCOPE_ALREADY_WAIVED", err)
	}
}

func TestResolveRejectsReplayedNonce(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(dir, "waiver.json")

	writeJSON(t, input, waiverInput("run", "nonce-1"))
	if err := runResolve([]string{"--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}

	withdraw := waiverInput("run", "nonce-3")
	withdraw["waiver_operation"].(map[string]any)["operation"] = string(model.WaiverOperationWithdraw)
	writeJSON(t, input, withdraw)
	if err := runResolve([]string{"--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}

	// Re-granting with the first operation's nonce must be refused.
	writeJSON(t, input, waiverInput("run", "nonce-1"))
	err := runResolve([]string{"--input", input, "--store", dir})
	if err == nil || err.Error() != "REUSED_RESOLUTION_NONCE" {
		t.Fatalf("reason = %v, want REUSED_RESOLUTION_NONCE", err)
	}
}

func TestResolveRejectsStaleChainPosition(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(dir, "waiver.json")

	stale := waiverInput("run", "nonce-1")
	stale["expected_predecessor_head"] = "an-anchor-that-is-no-longer-the-tail"
	writeJSON(t, input, stale)
	err := runResolve([]string{"--input", input, "--store", dir})
	if err == nil || err.Error() != "STALE_CHAIN_POSITION" {
		t.Fatalf("reason = %v, want STALE_CHAIN_POSITION", err)
	}
}

func TestResolveRecordsAreReplayable(t *testing.T) {
	dir := t.TempDir()
	s := seedCriterion(t, dir, "run")
	input := filepath.Join(dir, "waiver.json")
	writeJSON(t, input, waiverInput("run", "nonce-1"))
	if err := runResolve([]string{"--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}
	records, err := s.Load("run")
	if err != nil {
		t.Fatalf("chain unreadable after resolve: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("chain holds %d records, want 2", len(records))
	}
	if records[1].Kind != model.KindWaiverOperation {
		t.Fatalf("record kind = %q", records[1].Kind)
	}
}

func seedApprovedVerification(t *testing.T, dir, runID, repo, sha string) {
	t.Helper()
	s := store.New(dir)
	rec := model.Record{
		Kind:     model.KindStageTransition,
		Version:  1,
		Metadata: model.RecordMetadata{RunID: runID, Sequence: 1, Timestamp: "2026-09-03T00:00:00Z", Actor: "lennon"},
		Stage:    model.StageVerification,
		Control:  model.ControlReadyForReview,
		Approval: &model.ApprovalBinding{
			RunID:                    runID,
			TransitionDigest:         "transition",
			CurrentStage:             model.StageVerification,
			ProposedTargetStage:      model.StageIndependentValidation,
			RepositoryIdentityDigest: repo,
			BaseRevisionDigest:       "base-revision",
			StageTimeDigest:          ":1",
			AcceptedPlanDigest:       "plan",
			ChallengeNonce:           "nonce-approval",
			Approver:                 "lennon",
			Timestamp:                "2026-09-03T00:00:00Z",
			ProducedHeadDigest:       sha,
			DiffDigest:               "diff",
		},
	}
	if err := s.Append(&rec); err != nil {
		t.Fatal(err)
	}
}

func setGitHubEnv(t *testing.T, repo, sha string) {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REPOSITORY", repo)
	t.Setenv("GITHUB_SHA", sha)
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_RUN_ID", "1234")
	t.Setenv("GITHUB_ACTOR", "someone")
}

func TestCIRejectsRevisionMismatch(t *testing.T) {
	dir := t.TempDir()
	seedApprovedVerification(t, dir, "run", "owner/repo", "approved-sha")
	setGitHubEnv(t, "owner/repo", "a-different-sha")
	err := runCIGithub([]string{"--store", dir, "--run_id", "run", "--verify-only"})
	if err == nil || err.Error() != "CI_REVISION_MISMATCH" {
		t.Fatalf("reason = %v, want CI_REVISION_MISMATCH", err)
	}
}

// TestCIWritesNoEvidenceWithoutACommand is the regression test for evidence
// manufactured from environment variables alone.
func TestCIWritesNoEvidenceWithoutACommand(t *testing.T) {
	dir := t.TempDir()
	seedApprovedVerification(t, dir, "run", "owner/repo", "approved-sha")
	setGitHubEnv(t, "owner/repo", "approved-sha")

	if err := runCIGithub([]string{"--store", dir, "--run_id", "run"}); err == nil {
		t.Fatal("evidence capture ran without an explicit command")
	}
	if err := runCIGithub([]string{"--store", dir, "--run_id", "run", "--verify-only"}); err != nil {
		t.Fatalf("verify-only: %v", err)
	}
	records, err := store.New(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("verification wrote %d extra records; want none", len(records)-1)
	}
}

func TestCICapturesRealEvidence(t *testing.T) {
	if _, err := os.Stat("/bin/true"); err != nil {
		t.Skip("/bin/true unavailable")
	}
	dir := t.TempDir()
	repoRoot := t.TempDir()
	seedApprovedVerification(t, dir, "run", "owner/repo", "approved-sha")
	setGitHubEnv(t, "owner/repo", "approved-sha")

	if err := runCIGithub([]string{"--store", dir, "--run_id", "run", "--repo-root", repoRoot, "--exec", "/bin/true"}); err != nil {
		t.Fatal(err)
	}
	records, err := store.New(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("chain holds %d records, want 2", len(records))
	}
	e := records[1].Evidence
	if e == nil {
		t.Fatal("no evidence recorded")
	}
	if e.Executable != "/bin/true" {
		t.Fatalf("executable = %q", e.Executable)
	}
	if e.OutputDigest == "" {
		t.Fatal("evidence has no output digest")
	}
	if e.CapturingActor != "github-actions:someone" {
		t.Fatalf("capturing actor = %q", e.CapturingActor)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	for _, args := range [][]string{{}, {"nope"}, {"ci"}, {"ci", "gitlab"}} {
		if err := run(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestTransitionUsage(t *testing.T) {
	cases := [][]string{
		{"transition"},
		{"transition", "--input", "x.json"},
		{"transition", "--store", "dir"},
		{"transition", "--input", "x.json", "--store", "dir", "extra"},
	}
	for _, args := range cases {
		if err := run(args); err == nil {
			t.Fatalf("run(%v) = nil, want a usage error", args)
		}
	}
}

// TestTransitionRejectsUnknownInputFields: an input the host did not mean to
// give must not be silently ignored, or a field a human thought bound the
// approval would quietly bind nothing.
func TestTransitionRejectsUnknownInputFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.json")
	writeJSON(t, path, map[string]any{"run_id": "run", "not_a_field": true})
	if err := run([]string{"transition", "--input", path, "--store", filepath.Join(dir, "store")}); err == nil {
		t.Fatal("unknown input field accepted")
	}
}

func TestTransitionRejectsMultipleInputDocuments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.json")
	if err := os.WriteFile(path, []byte(`{"run_id":"a"}{"run_id":"b"}`), 0600); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"transition", "--input", path, "--store", filepath.Join(dir, "store")})
	if err == nil || err.Error() != "multiple input documents" {
		t.Fatalf("err = %v, want multiple input documents", err)
	}
}

// TestShowReportsTheRunStage: show is what a human reads before approving, so
// it must say where the run actually is.
func TestShowReportsTheRunStage(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	rec := model.Record{
		Kind:     model.KindStageTransition,
		Version:  1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: 1, Timestamp: "2026-09-05T00:00:00Z", Actor: "operator-confirmed:lennon"},
		Stage:    model.StageGrounding,
		Control:  model.ControlApproved,
		Approval: &model.ApprovalBinding{
			RunID: "run", TransitionDigest: "t", CurrentStage: model.StageGrounding,
			ProposedTargetStage: model.StageAcceptanceCriteria, RepositoryIdentityDigest: "id",
			BaseRevisionDigest: "base", StageTimeDigest: ":1", AcceptedPlanDigest: "plan",
			ChallengeNonce: "n1", Approver: "operator-confirmed:lennon", Timestamp: "2026-09-05T00:00:00Z",
		},
	}
	if err := s.Append(&rec); err != nil {
		t.Fatal(err)
	}
	records, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	doc := newShowDocument(engine.ProjectResolutionState(records), engine.ProjectStageState(records))
	if doc.CurrentStage != model.StageAcceptanceCriteria {
		t.Fatalf("show reports stage %q, want ACCEPTANCE_CRITERIA", doc.CurrentStage)
	}
	if doc.Sequence != 1 || doc.Terminal {
		t.Fatalf("doc = %+v", doc)
	}
}
