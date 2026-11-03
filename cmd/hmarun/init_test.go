package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

const sampleAnswersJSON = `{
  "intent": "add a --json flag to hma show",
  "non_goals": ["no change to hma show's default text output"],
  "risk": "low",
  "criteria": [
    {
      "id": "C1",
      "statement": "hma show --json emits valid JSON for a run",
      "threshold": "exit 0 and output parses as JSON",
      "check": {"executable": "go", "argv": ["test", "./cmd/hma/...", "-run", "TestShowJSON"], "working_dir": "."}
    }
  ],
  "units": [
    {
      "allowed_files": ["cmd/hma/show.go", "cmd/hma/show_test.go"],
      "forbidden_files": ["internal/engine/", "internal/transition/"]
    }
  ]
}`

// TestRunInitAnswersFileWritesParsableDocuments runs the real `hmarun init
// --answers <file>` path end to end and checks that the files it writes
// parse through model.ParseAcceptanceContract/ParsePlan, exactly as a
// consumer of hmarun's output would read them.
func TestRunInitAnswersFileWritesParsableDocuments(t *testing.T) {
	dir := t.TempDir()
	answersPath := filepath.Join(dir, "answers.json")
	if err := os.WriteFile(answersPath, []byte(sampleAnswersJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")

	var stdout bytes.Buffer
	if err := runInit([]string{"--out-dir", outDir, "--answers", answersPath}, bytes.NewReader(nil), &stdout); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	contractRaw, err := os.ReadFile(filepath.Join(outDir, "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	planRaw, err := os.ReadFile(filepath.Join(outDir, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}

	contract, digest, err := model.ParseAcceptanceContract(contractRaw)
	if err != nil {
		t.Fatalf("ParseAcceptanceContract: %v", err)
	}
	plan, _, err := model.ParsePlan(planRaw)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}
	if err := model.ValidatePlanAgainstContract(plan, contract, digest); err != nil {
		t.Fatalf("ValidatePlanAgainstContract: %v", err)
	}
}

// TestRunInitAnswersFileMissingRequiredFieldErrors checks that an answers
// file missing a required field (here: no criteria at all) fails with
// hmarun's own clear error, and writes nothing.
func TestRunInitAnswersFileMissingRequiredFieldErrors(t *testing.T) {
	dir := t.TempDir()
	answersPath := filepath.Join(dir, "answers.json")
	const badAnswers = `{"intent": "do something", "risk": "low", "criteria": []}`
	if err := os.WriteFile(answersPath, []byte(badAnswers), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")

	var stdout bytes.Buffer
	err := runInit([]string{"--out-dir", outDir, "--answers", answersPath}, bytes.NewReader(nil), &stdout)
	if err == nil {
		t.Fatal("expected an error for an answers file with no criteria")
	}
	if ae, ok := err.(*answerError); !ok || ae.path != "criteria" {
		t.Errorf("expected *answerError at path \"criteria\", got %T: %v", err, err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "contract.json")); err == nil {
		t.Error("contract.json should not have been written")
	}
}

// TestRunInitAnswersFileUnknownFieldErrors checks that a typo'd answers-file
// key (e.g. "nongoals" instead of "non_goals") is rejected rather than
// silently ignored.
func TestRunInitAnswersFileUnknownFieldErrors(t *testing.T) {
	dir := t.TempDir()
	answersPath := filepath.Join(dir, "answers.json")
	const typoAnswers = `{"intent": "do something", "risk": "low", "nongoals": ["x"], "criteria": []}`
	if err := os.WriteFile(answersPath, []byte(typoAnswers), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")

	var stdout bytes.Buffer
	err := runInit([]string{"--out-dir", outDir, "--answers", answersPath}, bytes.NewReader(nil), &stdout)
	if err == nil {
		t.Fatal("expected an error for an unknown answers-file field")
	}
}
