// Package main implements hmarun, a small wrapper CLI that helps a human
// produce the hma.acceptance-contract/v1 and hma.plan/v1 JSON documents that
// the hma CLI and internal/model require, without hand-authoring their
// strict JSON by hand. hmarun never reimplements internal/engine,
// internal/transition, or internal/model logic: it only templates documents
// and round-trips them through internal/model's own parse/validate
// functions before writing anything to disk.
package main

// CheckAnswer is the answers-file shape for one internal/model.Check: an
// explicit executable and argument vector, never a shell string.
//
// WorkingDir defaults to "." when omitted.
type CheckAnswer struct {
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
	WorkingDir string   `json:"working_dir,omitempty"`
}

// CriterionAnswer is the answers-file shape for one
// internal/model.ContractCriterion.
//
// ID defaults to "C<n>" (1-based position in the criteria list) when
// omitted. EvidenceRequirement defaults to "fresh capture" when omitted.
// Statement, Threshold, and Check are genuinely task-specific and have no
// default: omitting them is a hard error.
type CriterionAnswer struct {
	ID                  string      `json:"id,omitempty"`
	Statement           string      `json:"statement"`
	Threshold           string      `json:"threshold"`
	EvidenceRequirement string      `json:"evidence_requirement,omitempty"`
	Check               CheckAnswer `json:"check"`
}

// UnitAnswer is the answers-file shape for one internal/model.PlanUnit.
//
// ID defaults to "U<n>" (1-based position in the units list) when omitted.
// CriteriaAdvanced defaults to every criterion id in the contract, but only
// when there is exactly one unit in the plan; with more than one unit the
// mapping from unit to criteria is a task-specific decision that hmarun
// cannot safely guess, so it must be given explicitly. NonGoals defaults to
// the top-level non_goals answer when that is non-empty. VerificationCommands
// defaults to the Check of every criterion this unit advances. Dependencies
// defaults to an empty list (no prior unit).
//
// AllowedFiles and ForbiddenFiles are genuinely task-specific and have no
// default: omitting them is a hard error.
type UnitAnswer struct {
	ID                   string        `json:"id,omitempty"`
	AllowedFiles         []string      `json:"allowed_files"`
	ForbiddenFiles       []string      `json:"forbidden_files"`
	VerificationCommands []CheckAnswer `json:"verification_commands,omitempty"`
	NonGoals             []string      `json:"non_goals,omitempty"`
	CriteriaAdvanced     []string      `json:"criteria_advanced,omitempty"`
	Dependencies         []string      `json:"dependencies,omitempty"`
}

// Answers is the full answers-file schema read by `hmarun init --answers
// <file>`. It carries the same logical fields the interactive prompts
// collect, so the two input paths build identical documents through the
// same templating code.
//
// Example:
//
//	{
//	  "intent": "add a --json flag to hma show",
//	  "non_goals": ["no change to hma show's default text output"],
//	  "risk": "low",
//	  "criteria": [
//	    {
//	      "id": "C1",
//	      "statement": "hma show --json emits valid JSON for a run",
//	      "threshold": "exit 0 and output parses as JSON",
//	      "check": {"executable": "go", "argv": ["test", "./cmd/hma/...", "-run", "TestShowJSON"], "working_dir": "."}
//	    }
//	  ],
//	  "units": [
//	    {
//	      "allowed_files": ["cmd/hma/show.go", "cmd/hma/show_test.go"],
//	      "forbidden_files": ["internal/engine/", "internal/transition/"]
//	    }
//	  ]
//	}
//
// RunID defaults to a slug derived from Intent when omitted. Deliverables
// defaults to ["code"] when omitted.
type Answers struct {
	RunID        string            `json:"run_id,omitempty"`
	Intent       string            `json:"intent"`
	NonGoals     []string          `json:"non_goals,omitempty"`
	Risk         string            `json:"risk"`
	Deliverables []string          `json:"deliverables,omitempty"`
	Criteria     []CriterionAnswer `json:"criteria"`
	Units        []UnitAnswer      `json:"units,omitempty"`
}
