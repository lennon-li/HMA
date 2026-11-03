package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// errInputEnded is returned by readLine (and so by every prompter method
// built on it) when stdin is exhausted before an answer was given. Without
// this, EOF on stdin looked identical to an empty Enter press, so a
// required-answer loop like askRequired would spin forever re-prompting
// instead of refusing; see the stepWorkerStub-adjacent F3 finding.
var errInputEnded = errors.New("input ended before an answer was given")

// prompter reads interactive answers from in, echoing prompts to out. It
// builds exactly the same Answers struct the --answers JSON path builds, so
// buildContractAndPlan (build.go) is the only place templating logic lives.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{in: bufio.NewReader(in), out: out}
}

func (p *prompter) line(format string, a ...any) {
	fmt.Fprintf(p.out, format, a...)
}

// readLine reads one line of input with its trailing newline trimmed. A
// final line with no trailing newline is still returned as real input (err
// nil); EOF with nothing left to read (s empty) is reported as
// errInputEnded rather than silently treated as an empty answer, so a caller
// like askRequired errors instead of looping forever re-prompting stdin that
// will never produce anything.
func (p *prompter) readLine() (string, error) {
	s, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && s == "" {
		return "", errInputEnded
	}
	return strings.TrimRight(s, "\r\n"), nil
}

// ask prompts for a single required or defaulted value. An empty answer
// keeps def (which may itself be empty, meaning "required": the caller is
// responsible for re-prompting or erroring on a required field).
func (p *prompter) ask(label, help, def string) (string, error) {
	if help != "" {
		p.line("%s\n", help)
	}
	if def != "" {
		p.line("%s [%s]: ", label, def)
	} else {
		p.line("%s: ", label)
	}
	ans, err := p.readLine()
	if err != nil {
		return "", err
	}
	ans = strings.TrimSpace(ans)
	if ans == "" {
		return def, nil
	}
	return ans, nil
}

// askRequired keeps asking until a non-empty answer is given.
func (p *prompter) askRequired(label, help string) (string, error) {
	for {
		ans, err := p.ask(label, help, "")
		if err != nil {
			return "", err
		}
		if ans != "" {
			return ans, nil
		}
		p.line("%s is required.\n", label)
	}
}

// askYesNo asks a yes/no question, defaulting to def when the answer is
// empty.
func (p *prompter) askYesNo(label string, def bool) (bool, error) {
	d := "n"
	if def {
		d = "y"
	}
	ans, err := p.ask(label+" (y/n)", "", d)
	if err != nil {
		return false, err
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes", nil
}

// askList collects comma-separated entries into a []string, trimming
// whitespace and dropping empty entries. A completely empty answer yields
// nil (not an empty slice), so callers can tell "not answered" from
// "answered empty" when choosing a default.
func (p *prompter) askList(label, help string) ([]string, error) {
	if help != "" {
		p.line("%s\n", help)
	}
	p.line("%s (comma-separated, blank for none): ", label)
	raw, err := p.readLine()
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out, nil
}

func (p *prompter) askCheck(prefix string) (CheckAnswer, error) {
	exe, err := p.askRequired(prefix+" executable", "The program to run, e.g. \"go\" (never a shell string: no pipes, &&, or redirects).")
	if err != nil {
		return CheckAnswer{}, err
	}
	argv, err := p.askList(prefix+" arguments", "The arguments to that program, one shell word each, e.g. \"test, ./...\".")
	if err != nil {
		return CheckAnswer{}, err
	}
	if argv == nil {
		argv = []string{}
	}
	wd, err := p.ask(prefix+" working directory", "Directory the command runs from, relative to the repo root.", ".")
	if err != nil {
		return CheckAnswer{}, err
	}
	return CheckAnswer{Executable: exe, Argv: argv, WorkingDir: wd}, nil
}

// collectAnswers runs the full interactive Q&A and returns the Answers it
// built.
func collectAnswers(p *prompter) (Answers, error) {
	var a Answers

	intent, err := p.askRequired("Intent", "One or two sentences: what is this task supposed to accomplish?")
	if err != nil {
		return a, err
	}
	a.Intent = intent

	runID, err := p.ask("Run id", "A short identifier for this run. Leave blank to derive one from the intent.", "")
	if err != nil {
		return a, err
	}
	a.RunID = runID

	nonGoals, err := p.askList("Non-goals", "Things this task explicitly will NOT do (helps keep scope bounded).")
	if err != nil {
		return a, err
	}
	a.NonGoals = nonGoals

	risk, err := p.askRequired("Risk level", "How risky is this change: low, medium, high, or critical?")
	if err != nil {
		return a, err
	}
	a.Risk = risk

	deliverables, err := p.askList("Deliverables", "What this task produces, e.g. \"code\". Leave blank to default to [\"code\"].")
	if err != nil {
		return a, err
	}
	a.Deliverables = deliverables

	p.line("\nNow the acceptance criteria: each one is a single checkable claim about the finished work.\n")
	for i := 1; ; i++ {
		p.line("\n-- Criterion %d --\n", i)
		statement, err := p.askRequired(fmt.Sprintf("Criterion %d statement", i), "The claim itself, e.g. \"all existing tests still pass\".")
		if err != nil {
			return a, err
		}
		threshold, err := p.askRequired(fmt.Sprintf("Criterion %d threshold", i), "What result counts as meeting the claim, e.g. \"exit 0\".")
		if err != nil {
			return a, err
		}
		check, err := p.askCheck(fmt.Sprintf("Criterion %d verification command", i))
		if err != nil {
			return a, err
		}
		a.Criteria = append(a.Criteria, CriterionAnswer{Statement: statement, Threshold: threshold, Check: check})

		more, err := p.askYesNo("Add another criterion?", false)
		if err != nil {
			return a, err
		}
		if !more {
			break
		}
	}

	p.line("\nNow the plan unit(s): the bounded piece(s) of work an implementer will actually do.\n")
	for i := 1; ; i++ {
		p.line("\n-- Plan unit %d --\n", i)
		allowedFiles, err := p.askList(fmt.Sprintf("Unit %d allowed files", i), "Files/paths this unit is allowed to change.")
		if err != nil {
			return a, err
		}
		for len(allowedFiles) == 0 {
			p.line("At least one allowed file is required.\n")
			allowedFiles, err = p.askList(fmt.Sprintf("Unit %d allowed files", i), "")
			if err != nil {
				return a, err
			}
		}
		forbiddenFiles, err := p.askList(fmt.Sprintf("Unit %d forbidden files", i), "Files/paths this unit must NOT touch.")
		if err != nil {
			return a, err
		}
		for len(forbiddenFiles) == 0 {
			p.line("At least one forbidden file is required.\n")
			forbiddenFiles, err = p.askList(fmt.Sprintf("Unit %d forbidden files", i), "")
			if err != nil {
				return a, err
			}
		}
		unitNonGoals, err := p.askList(fmt.Sprintf("Unit %d non-goals", i), "Leave blank to reuse the top-level non-goals.")
		if err != nil {
			return a, err
		}

		criteriaAdvanced, err := p.askList(fmt.Sprintf("Unit %d criteria advanced", i),
			"Which criterion ids (by id, e.g. \"C1\") does this unit advance? Leave blank only if this plan ends up with exactly one unit.")
		if err != nil {
			return a, err
		}

		a.Units = append(a.Units, UnitAnswer{
			AllowedFiles: allowedFiles, ForbiddenFiles: forbiddenFiles, NonGoals: unitNonGoals, CriteriaAdvanced: criteriaAdvanced,
		})

		more, err := p.askYesNo("Add another plan unit?", false)
		if err != nil {
			return a, err
		}
		if !more {
			break
		}
	}

	return a, nil
}
