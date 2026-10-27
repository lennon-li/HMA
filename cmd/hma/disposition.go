package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// currentPlanUnit is the unit's entry in the plan of its current lease
// generation.
func currentPlanUnit(storePath string, core []model.Record, unit string, supplied ...*model.Plan) (model.PlanUnit, error) {
	digest := ""
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			digest = r.Approval.AcceptedPlanDigest
		}
	}
	var plan model.Plan
	if len(supplied) > 0 && supplied[0] != nil {
		plan = *supplied[0]
		b, err := json.Marshal(plan)
		if err != nil {
			return model.PlanUnit{}, err
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != digest {
			return model.PlanUnit{}, errors.New(hoststate.ReasonFindingsImportInvalid)
		}
	} else {
		plans, err := loadPlans(storePath, core)
		if err != nil {
			return model.PlanUnit{}, err
		}
		var ok bool
		plan, ok = plans[digest]
		if !ok {
			return model.PlanUnit{}, errors.New(hoststate.ReasonFindingsImportInvalid)
		}
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
// neutralised. Git-marked binary entries use bounded raw-byte line counts.
// A rename counts as a deletion and an
// addition; a merge is measured as the base-to-target tree difference.
func measureChange(ctx context.Context, root, base string, dirty bool) (int, int, error) {
	entries, err := observeTarget(ctx, root, base, dirty)
	if err != nil {
		return 0, 0, err
	}
	lines := 0
	for _, e := range entries {
		if e.LinesTruncated {
			return len(entries), int(^uint(0) >> 1), nil
		}
		lines += e.Added + e.Deleted
	}
	return len(entries), lines, nil
}

const maxCountedBytes = 16 << 20

// untrackedLines remains a bounded helper for its focused regression tests.
// Production manifests and budget facts use observeTarget.
func untrackedLines(path string) (int, bool) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, false
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return 1, false
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	const limit = 16 << 20
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return 0, false
	}
	overflow := len(data) > limit
	if overflow {
		data = data[:limit]
	}
	n := bytes.Count(data, []byte{'\n'})
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	if len(data) > 0 && n == 0 {
		n = 1
	}
	return n, overflow
}

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
