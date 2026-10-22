package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// currentPlanUnit is the unit's entry in the plan of its current lease
// generation.
func currentPlanUnit(storePath string, core []model.Record, unit string) (model.PlanUnit, error) {
	digest := ""
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			digest = r.Approval.AcceptedPlanDigest
		}
	}
	plans, err := loadPlans(storePath, core)
	if err != nil {
		return model.PlanUnit{}, err
	}
	plan, ok := plans[digest]
	if !ok {
		return model.PlanUnit{}, errors.New(hoststate.ReasonFindingsImportInvalid)
	}
	for _, u := range plan.Units {
		if u.ID == unit {
			return u, nil
		}
	}
	return model.PlanUnit{}, errors.New(hoststate.ReasonFindingsImportInvalid)
}

// shellPolicyViolated reports whether a command the plan would run reaches a
// shell under the evidence layer's no-shell policy (the same RejectShell the
// capture applies). A tool that is merely absent from PATH is not a shell
// violation.
func shellPolicyViolated(c model.Check) bool {
	resolved, err := evidence.ResolveExecutable(c.Executable)
	if err != nil {
		resolved = c.Executable
	}
	return evidence.RejectShell(c.Executable, resolved, c.Argv) != ""
}

// findingFacts establishes, from HMA's own state and measurement, which
// registered kernel rules hold now and which plan budgets are exceeded. It
// reads no auditor text and runs no auditor command.
func findingFacts(ctx context.Context, storePath string, core []model.Record, unit string, p *hoststate.ReviewPacketBinding, dirty bool) (hoststate.FindingFacts, error) {
	facts := hoststate.FindingFacts{RuleHolds: map[string]bool{}, BudgetExceeded: map[string]bool{}}
	u, err := currentPlanUnit(storePath, core, unit)
	if err != nil {
		return facts, err
	}
	// no_shell: a command the unit's plan would run reaches a shell.
	for _, c := range u.VerificationCommands {
		if shellPolicyViolated(c) {
			facts.RuleHolds["no_shell"] = true
		}
	}
	files, lines, err := measureChange(ctx, p.ProjectRoot, p.BaseRevision, dirty)
	if err != nil {
		return facts, err
	}
	facts.BudgetExceeded["max_files"] = files > u.Budgets.MaxFiles
	facts.BudgetExceeded["max_diff_lines"] = lines > u.Budgets.MaxDiffLines
	return facts, nil
}

// measureChange counts the files and the changed lines (added plus deleted)
// from the approved base to the target: the committed change, and for a
// dirty-approved target also the uncommitted tracked change and the
// untracked, non-ignored files. Repository attributes and diff settings are
// neutralised and every file is read as text, so neither a -diff attribute
// nor a binary label can hide lines. A rename counts as a deletion and an
// addition; a merge is measured as the base-to-target tree difference.
func measureChange(ctx context.Context, root, base string, dirty bool) (int, int, error) {
	empty, err := repostate.Output(ctx, root, "hash-object", "-t", "tree", "/dev/null")
	if err != nil {
		return 0, 0, err
	}
	numstat := func(args ...string) (map[string]int, error) {
		full := append([]string{"-c", "core.attributesFile=/dev/null", "-c", "core.bigFileThreshold=2g", "--attr-source=" + repostate.Trim(empty),
			"diff", "--numstat", "--text", "--no-renames", "--no-ext-diff", "--no-textconv"}, args...)
		out, err := repostate.Output(ctx, root, full...)
		if err != nil {
			return nil, err
		}
		rows := map[string]int{}
		for _, row := range strings.Split(out, "\n") {
			f := strings.SplitN(row, "\t", 3)
			if len(f) < 3 {
				continue
			}
			n := 0
			for _, v := range f[:2] {
				if c, err := strconv.Atoi(v); err == nil {
					n += c
				}
			}
			rows[f[2]] = n
		}
		return rows, nil
	}
	var rows map[string]int
	if dirty {
		// Base against the worktree: committed, staged and unstaged tracked changes.
		rows, err = numstat(base, "--")
	} else {
		head, herr := repostate.Output(ctx, root, "rev-parse", "HEAD")
		if herr != nil {
			return 0, 0, herr
		}
		rows, err = numstat(base+".."+repostate.Trim(head), "--")
	}
	if err != nil {
		return 0, 0, err
	}
	if dirty {
		others, err := repostate.Output(ctx, root, "ls-files", "--others", "--exclude-per-directory=.gitignore", "-z")
		if err != nil {
			return 0, 0, err
		}
		for _, path := range strings.Split(others, "\x00") {
			if path == "" {
				continue
			}
			rows[path] = untrackedLines(filepath.Join(root, filepath.FromSlash(path)))
		}
	}
	lines := 0
	for _, n := range rows {
		lines += n
	}
	return len(rows), lines, nil
}

// maxCountedBytes bounds the bytes read from one untracked file; a file past
// it counts the lines of its first part, which already exceeds any sane budget.
const maxCountedBytes = 16 << 20

// untrackedLines counts the lines git would count for an untracked path: a
// symlink is its link text (one line), a regular file its lines (read with a
// bound), and anything else (a FIFO, a device, a socket) one line without
// being opened.
func untrackedLines(path string) int {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0
	}
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		return 1
	case !fi.Mode().IsRegular():
		return 1
	}
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	n, total := 0, 0
	var last byte = '\n'
	for total < maxCountedBytes {
		r, err := f.Read(buf)
		n += bytes.Count(buf[:r], []byte{'\n'})
		if r > 0 {
			last = buf[r-1]
			total += r
		}
		if err != nil {
			break
		}
	}
	if last != '\n' {
		n++
	}
	return n
}

// requireFindingsResolved refuses while a recorded finding stays unresolved:
// any BLOCK finding (unwaivable), and any WAIVABLE finding without an active
// human waiver (architecture §5.2, §9). unit limits the check to that unit's
// findings and unowned ones; an empty unit checks every finding (closure).
func requireFindingsResolved(storePath, runID, unit string) error {
	core, err := store.New(storePath).Load(runID)
	if err != nil {
		return err
	}
	state := engine.ProjectResolutionState(core)
	findings := hoststate.AssignLegacyOwners(core, state.Findings)
	for _, f := range findings {
		if unit != "" && f.UnitID != "" && f.UnitID != unit {
			continue
		}
		switch f.Disposition {
		case model.FindingBlock:
			return errors.New(hoststate.ReasonFindingsUnresolved)
		case model.FindingWaivable:
			if !state.ActiveWaivers[model.ScopeSelector{Kind: model.ScopeFinding, Target: f.ID}] {
				return errors.New(hoststate.ReasonFindingsUnresolved)
			}
		}
	}
	return nil
}
