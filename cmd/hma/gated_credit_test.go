package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// creditFixture is a review fixture whose run binds an acceptance contract
// with criteria C1 (check: exe) and C2 (not advanced by U1), and a plan in
// which U1 advances C1.
type creditFixture struct {
	*reviewFixture
	contractPath string
}

// creditRepoSetup, when set by a test, edits the fixture repository before
// the review packet is generated.
var creditRepoSetup func(*reviewFixture)

// creditAlias is a path a test sets to name the live repository by another
// spelling in a check.
var creditAlias string

func newCreditFixture(t *testing.T, exe string) *creditFixture {
	t.Helper()
	return newCreditFixtureC2(t, exe, model.CriterionPending)
}

// newCreditFixtureC2 is newCreditFixture with C2's initial disposition c2.
func newCreditFixtureC2(t *testing.T, exe string, c2 model.CriterionDisposition) *creditFixture {
	t.Helper()
	return newCreditFixtureArgs(t, exe, []string{}, c2)
}

// newCreditFixtureArgs is newCreditFixtureC2 with the check's argv.
func newCreditFixtureArgs(t *testing.T, exe string, argv []string, c2 model.CriterionDisposition) *creditFixture {
	t.Helper()
	cf := &creditFixture{}
	cf.reviewFixture = newReviewFixtureSeeded(t, hoststate.IssuePerChange, false, func(f *reviewFixture) {
		if creditRepoSetup != nil {
			creditRepoSetup(f)
		}
		pol := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
		proj, _ := model.NormalizeRoutePolicy(pol)
		policyDigest, _ := engine.RoutePolicyDigest(proj)
		all := []string{model.PermissionReadOnly, model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
		expanded := make([]string, len(argv))
		for i, a := range argv {
			expanded[i] = strings.ReplaceAll(strings.ReplaceAll(a, "@ROOT@", f.root), "@ALIAS@", creditAlias)
		}
		check := model.Check{Executable: exe, Argv: expanded, WorkingDir: "."}
		c := model.AcceptanceContract{Schema: model.AcceptanceContractSchema, RunID: "run", Intent: "one behavior", NonGoals: []string{"none"},
			Criteria: []model.ContractCriterion{
				{ID: "C1", Statement: "check passes", Threshold: "exit 0", EvidenceRequirement: "fresh capture", Check: check},
				{ID: "C2", Statement: "other", Threshold: "exit 0", EvidenceRequirement: "fresh capture", Check: check}},
			Risk: model.RouteRiskLow, Deliverables: []string{"code"},
			Budgets:         model.ContractBudgets{MaxAttempts: 10, MaxWallMinutes: 120, CostCeiling: 20, CostUnit: "usd"},
			PermittedRoutes: []string{"impl", "esc", "val"}, PermissionEnvelope: all, PolicyDigest: policyDigest}
		cb, _ := json.Marshal(c)
		_, contractDigest, err := model.ParseAcceptanceContract(cb)
		if err != nil {
			t.Fatal(err)
		}
		cf.contractPath = filepath.Join(t.TempDir(), "contract.json")
		f.contractPath = cf.contractPath
		if err := os.WriteFile(cf.contractPath, cb, 0o600); err != nil {
			t.Fatal(err)
		}
		p := model.Plan{Schema: model.PlanSchema, RunID: "run", AcceptanceContractDigest: contractDigest, PolicyDigest: policyDigest,
			Units: []model.PlanUnit{{ID: "U1", Behavior: "b", CriteriaAdvanced: []string{"C1"}, AllowedFiles: []string{"a"}, AllowedActions: []string{"edit"},
				ForbiddenFiles: []string{"docs"}, ForbiddenActions: []string{"push"}, PreChangeEvidence: "failing test", MinimumImplementation: "m",
				ReuseAnalysis: "r", PermissionClasses: all, ImplementerRoute: "impl", EscalationRoute: "esc", ValidatorRoute: "val",
				IndependenceRequirement: "family", VerificationCommands: []model.Check{check},
				RequiredEvidence: []string{"check"}, StopConditions: []string{"s"}, RollbackConditions: []string{"r"}, EscalationConditions: []string{"e"},
				NonGoals: []string{"n"}, Dependencies: []string{},
				Budgets: model.UnitBudgets{MaxComplexity: 5, MaxFiles: 3, MaxDiffLines: 100, MaxToolSteps: 50, MaxRetries: 1, MaxAttempts: 3, MaxWallMinutes: 120, CostCeiling: 10}}}}
		pb, _ := json.Marshal(p)
		plan, planDigest, err := model.ParsePlan(pb)
		if err != nil {
			t.Fatal(err)
		}
		if err := retainPlan(f.store, plan, planDigest); err != nil {
			t.Fatal(err)
		}
		f.planDigest = planDigest
		f.policy = filepath.Join(t.TempDir(), "policy.json")
		polb, _ := json.Marshal(pol)
		if err := os.WriteFile(f.policy, polb, 0o600); err != nil {
			t.Fatal(err)
		}
		s := store.New(f.store)
		seq := int64(0)
		add := func(r model.Record) {
			seq++
			recs, _ := s.Load("run")
			r.Version = 1
			r.Metadata = model.RecordMetadata{RunID: "run", Sequence: seq, Timestamp: "2026-09-16T11:00:00Z", Actor: "operator-confirmed:lennon"}
			if len(recs) > 0 {
				r.Metadata.PredecessorHash = recs[len(recs)-1].Metadata.HeadAnchor
			}
			if err := s.Append(&r); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
		approve := func(unit string, from, to model.Stage, digest string) {
			add(model.Record{UnitID: unit, Kind: model.KindStageTransition, Stage: from, Control: model.ControlApproved,
				Approval: &model.ApprovalBinding{UnitID: unit, RunID: "run", TransitionDigest: digest, CurrentStage: from, ProposedTargetStage: to,
					RepositoryIdentityDigest: "repo", BaseRevisionDigest: f.base, StageTimeDigest: "head:1", AcceptedPlanDigest: planDigest,
					AcceptanceContractDigest: contractDigest, ChallengeNonce: fmt.Sprintf("n%d", seq), Approver: "human", Timestamp: "2026-09-16T11:00:00Z"}})
		}
		add(model.Record{Kind: model.KindCriterion, Stage: model.StageAcceptanceCriteria,
			Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPending}, {ID: "C2", Disposition: c2}}})
		approve("", model.StageAcceptanceCriteria, model.StagePlanning, "contract")
		approve("", model.StagePlanning, model.StageRouteSelection, "plan")
		approve("U1", model.StageRouteSelection, model.StageImplementationAuthorization, "approved-route")
	})
	return cf
}

// approve imports an isolated pass review whose report lists status for C1.
func (cf *creditFixture) approve(status string) string {
	cf.t.Helper()
	report := cf.report("pass", "isolated", func(r map[string]any) {
		r["requirements"] = []any{map[string]any{"id": "C1", "status": status, "evidence_refs": []any{"e1"}}}
	})
	cf.receipt(report, nil)
	if err := cf.importReport(report); err != nil {
		cf.t.Fatalf("import: %v", err)
	}
	return report
}

func (cf *creditFixture) credit(report string) (creditResult, error) {
	cf.t.Helper()
	in := filepath.Join(cf.t.TempDir(), "credit.json")
	writeJSON(cf.t, in, hostCreditInput{hostEnvelope: fxEnv("2026-09-16T12:07:00Z"), ReviewRecord: cf.latestReview(),
		ReportPath: report, ContractPath: cf.contractPath})
	var res creditResult
	out, err := captureStdout(cf.t, func() error { return run([]string{"host", "credit", "--input", in, "--store", cf.store}) })
	if err == nil {
		if jerr := json.Unmarshal(out, &res); jerr != nil {
			cf.t.Fatal(jerr)
		}
	}
	return res, err
}

func (cf *creditFixture) criteria() map[string]model.CriterionDisposition {
	core, _ := store.New(cf.store).Load("run")
	out := map[string]model.CriterionDisposition{}
	for _, c := range engine.ProjectResolutionState(core).Criteria {
		out[c.ID] = c.Disposition
	}
	return out
}

// outcomeGate runs the verified-outcome credit check against the live
// repository as the closure artifact (dirty worktrees allowed, so an
// unreviewed edit shows up as a different artifact).
func (cf *creditFixture) outcomeGate() error {
	h, _ := hoststate.NewStore(cf.store).Load("run")
	snap, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
	if err != nil {
		return err
	}
	return requireIndependentCredit(cf.store, "run", hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree), h)
}

// Gate C §0.1 / Gate A v3 §4.3: a criterion is PASSED only when HMA's own
// check passed at the reviewed target and the independent review lists it
// as pass; the criterion the unit does not advance is untouched.
func TestScorecardOnlyIndependentCredit(t *testing.T) {
	// A self-reported scorecard (a criterion record written directly)
	// credits nothing: the verified-outcome gate refuses it.
	sc := newCreditFixture(t, "true")
	sc.appendCore(model.Record{Kind: model.KindCriterion, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPassed}, {ID: "C2", Disposition: model.CriterionPending}}})
	if err := sc.outcomeGate(); err == nil {
		t.Fatal("self-reported scorecard accepted")
	}
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	res, err := cf.credit(report)
	if err != nil || len(res.Results) != 1 || res.Results[0].Disposition != model.CriterionPassed || res.Results[0].ExitCode != 0 {
		t.Fatalf("credit: %+v %v", res, err)
	}
	if got := cf.criteria(); got["C1"] != model.CriterionPassed || got["C2"] != model.CriterionPending {
		t.Fatalf("criteria: %v", got)
	}
	if err := cf.outcomeGate(); err != nil {
		t.Fatalf("credited outcome gate: %v", err)
	}
	// A later self-written PASSED for C2 is not laundered by the earlier credit.
	cf.appendCore(model.Record{Kind: model.KindCriterion, Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionPassed}, {ID: "C2", Disposition: model.CriterionPassed}}})
	if err := cf.outcomeGate(); err == nil {
		t.Fatal("self-written criterion record after credit accepted")
	}
}

// A review that does not list the criterion, or a check that fails, never
// credits it; a pass report listing it fail or unknown is not an approving
// review (the report contract escalates it), so it credits nothing at all.
func TestPartialNeverVerified(t *testing.T) {
	for _, tc := range []struct {
		exe, status string
		want        model.CriterionDisposition
	}{{"true", "", model.CriterionPending}, {"false", "pass", model.CriterionFailed}} {
		cf := newCreditFixture(t, tc.exe)
		var report string
		if tc.status == "" {
			report = cf.report("pass", "isolated", nil)
			cf.receipt(report, nil)
			if err := cf.importReport(report); err != nil {
				t.Fatal(err)
			}
		} else {
			report = cf.approve(tc.status)
		}
		res, err := cf.credit(report)
		if err != nil || len(res.Results) != 1 || res.Results[0].Disposition != tc.want {
			t.Fatalf("%s/%q: %+v %v", tc.exe, tc.status, res, err)
		}
		if got := cf.criteria(); got["C1"] != tc.want {
			t.Fatalf("%s/%q criteria: %v", tc.exe, tc.status, got)
		}
		if err := cf.outcomeGate(); err != nil {
			t.Fatalf("%s/%q: no PASSED criterion, gate: %v", tc.exe, tc.status, err)
		}
	}
	for _, status := range []string{"unknown", "fail"} {
		cf := newCreditFixture(t, "true")
		report := cf.report("pass", "isolated", func(r map[string]any) {
			r["requirements"] = []any{map[string]any{"id": "C1", "status": status, "evidence_refs": []any{"e1"}}}
		})
		cf.receipt(report, nil)
		if err := cf.importReport(report); err != nil {
			t.Fatal(err)
		}
		if _, err := cf.credit(report); err == nil || cf.criteria()["C1"] != model.CriterionPending {
			t.Fatalf("%s: credited %v", status, err)
		}
	}
}

// Credit needs an approving, isolated, latest review of the current target
// with its exact report bytes and the bound contract.
func TestCreditRefusals(t *testing.T) {
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	other := filepath.Join(t.TempDir(), "AUDIT.json")
	b, _ := os.ReadFile(report)
	os.WriteFile(other, append(b, ' '), 0o600)
	if _, err := cf.credit(other); err == nil {
		t.Fatal("other report bytes accepted")
	}
	// A valid contract other than the run's bound one.
	good := cf.contractPath
	var c map[string]any
	cb, _ := os.ReadFile(good)
	json.Unmarshal(cb, &c)
	c["intent"] = "another behavior"
	cb, _ = json.Marshal(c)
	cf.contractPath = filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(cf.contractPath, cb, 0o600)
	if _, _, err := model.ParseAcceptanceContract(cb); err != nil {
		t.Fatalf("other contract fixture: %v", err)
	}
	if _, err := cf.credit(report); err == nil {
		t.Fatal("unbound contract accepted")
	}
	cf.contractPath = good
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("control: %v", err)
	}
	// The repository moved off the reviewed artifact (a new commit).
	g := newCreditFixture(t, "true")
	r := g.approve("pass")
	bGit(t, g.root, "commit", "-q", "--allow-empty", "-m", "moved")
	if _, err := g.credit(r); err == nil {
		t.Fatal("moved target accepted")
	}
	// Non-approving and non-isolated reviews credit nothing.
	for _, tc := range []struct{ verdict, isolation string }{{"fix-required", "isolated"}, {"pass", "partial"}} {
		g := newCreditFixture(t, "true")
		r := g.report(tc.verdict, tc.isolation, func(r map[string]any) {
			r["requirements"] = []any{map[string]any{"id": "C1", "status": "pass", "evidence_refs": []any{"e1"}}}
		})
		g.receipt(r, nil)
		if err := g.importReport(r); err != nil {
			t.Fatalf("import %v: %v", tc, err)
		}
		if _, err := g.credit(r); err == nil {
			t.Fatalf("%v credited", tc)
		}
	}
}

// A self-written PASSED for a criterion the unit does not advance, carried
// into the credit's criterion record, is not credit.
func TestCarriedScorecardNotCredit(t *testing.T) {
	cf := newCreditFixtureC2(t, "true", model.CriterionPassed)
	if _, err := cf.credit(cf.approve("pass")); err != nil {
		t.Fatal(err)
	}
	if got := cf.criteria(); got["C1"] != model.CriterionPassed || got["C2"] != model.CriterionPassed {
		t.Fatalf("criteria: %v", got)
	}
	if err := cf.outcomeGate(); err == nil {
		t.Fatal("carried self-written PASSED accepted")
	}
}

// Credit stands only while its review is the unit's latest for the current
// implementation: a new implementation dispatch voids it.
func TestCreditVoidAfterNewImplementation(t *testing.T) {
	cf := newCreditFixture(t, "true")
	if _, err := cf.credit(cf.approve("pass")); err != nil {
		t.Fatal(err)
	}
	if err := cf.outcomeGate(); err != nil {
		t.Fatalf("control: %v", err)
	}
	cf.implement("2026-09-16T12:07:10Z", "2026-09-16T12:07:20Z", "2026-09-16T12:20:00Z")
	if err := cf.outcomeGate(); err == nil {
		t.Fatal("credit survived a new implementation")
	}
}

// Liz D5a F02: credit stands only at the reviewed artifact; an unreviewed
// edit makes the closure artifact differ and the guard refuses.
func TestCreditGuardTargetDrift(t *testing.T) {
	cf := newCreditFixture(t, "true")
	if _, err := cf.credit(cf.approve("pass")); err != nil {
		t.Fatal(err)
	}
	if err := cf.outcomeGate(); err != nil {
		t.Fatalf("control: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cf.root, "a"), []byte("UNREVIEWED"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cf.outcomeGate(); err == nil {
		t.Fatal("credit accepted after target drift")
	}
}

// Liz D5a F03: a working directory that resolves outside the repository is
// refused before the check runs; an internal symlink is fine.
func TestCreditSymlinkWorkingDir(t *testing.T) {
	outside := t.TempDir()
	if _, err := checkDir(t.TempDir(), "."); err != nil {
		t.Fatalf("control: %v", err)
	}
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "sub"), 0o700)
	os.Symlink(outside, filepath.Join(root, "out"))
	os.Symlink(filepath.Join(root, "sub"), filepath.Join(root, "in"))
	if _, err := checkDir(root, "out"); err == nil {
		t.Fatal("external symlink accepted")
	}
	if _, err := checkDir(root, "in"); err != nil {
		t.Fatalf("internal symlink: %v", err)
	}
	if _, err := checkDir(root, "missing"); err == nil {
		t.Fatal("dangling working dir accepted")
	}
}

// Liz D5a F04: a credit response carries one next action from its results.
func TestCreditNextAction(t *testing.T) {
	for _, tc := range []struct {
		d    []model.CriterionDisposition
		want string
	}{{[]model.CriterionDisposition{model.CriterionPassed}, "CONTINUE_HMA_TRANSITION"},
		{[]model.CriterionDisposition{model.CriterionPassed, model.CriterionPending}, "ESCALATE_TO_HUMAN"},
		{[]model.CriterionDisposition{model.CriterionPending, model.CriterionFailed}, "REMEDIATE_OR_ESCALATE"}} {
		var rs []hoststate.CriterionResult
		for _, d := range tc.d {
			rs = append(rs, hoststate.CriterionResult{Disposition: d})
		}
		if got := creditNextAction(rs); got != tc.want {
			t.Fatalf("%v: %s", tc.d, got)
		}
	}
	cf := newCreditFixture(t, "true")
	if res, err := cf.credit(cf.approve("pass")); err != nil || res.NextAction != "CONTINUE_HMA_TRANSITION" {
		t.Fatalf("%+v %v", res, err)
	}
}

// Liz D5a F01: a review dispatched under an earlier lease generation of the
// unit credits nothing after the unit is re-authorized, and its earlier
// credit no longer passes the guard.
func TestCreditRefusedAfterReauthorization(t *testing.T) {
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("control: %v", err)
	}
	reauth := func(from, to model.Stage, n string) {
		cf.appendCore(model.Record{UnitID: "U1", Kind: model.KindStageTransition, Stage: from, Control: model.ControlApproved,
			Approval: &model.ApprovalBinding{UnitID: "U1", RunID: "run", TransitionDigest: n, CurrentStage: from, ProposedTargetStage: to,
				RepositoryIdentityDigest: "repo", BaseRevisionDigest: cf.base, StageTimeDigest: "head:1",
				AcceptedPlanDigest: cf.planDigest, ChallengeNonce: n, Approver: "human", Timestamp: "2026-09-16T12:06:30Z"}})
	}
	reauth(model.StageImplementationAuthorization, model.StageRouteSelection, "rw")
	reauth(model.StageRouteSelection, model.StageImplementationAuthorization, "ra")
	if err := cf.outcomeGate(); err == nil {
		t.Fatal("credit of an earlier generation accepted")
	}
	if _, err := cf.credit(report); err == nil {
		t.Fatal("review of an earlier generation credited")
	}
}

// Liz D5a r2-r4 (F05-F08): git checks are limited to approved forms. Exact
// option spellings only (no abbreviation can stand for another option), a
// read-only subcommand list, and patch subcommands must disable configured
// diff helpers. Aliases, mutating subcommands, and any other option are
// refused. Shared by credit and loop evidence.
func TestUnsafeLoopArgvGitOptions(t *testing.T) {
	nd := []string{"--no-ext-diff", "--no-textconv"}
	cat := func(parts ...[]string) []string {
		var out []string
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	for _, tc := range []struct {
		argv   []string
		unsafe bool
	}{
		{[]string{"git", "status"}, false},
		{[]string{"git", "--no-pager", "status", "-s"}, false},
		{[]string{"git", "-C", "/x", "status"}, true},
		{[]string{"git", "-C", ".", "status"}, true},
		{cat([]string{"git", "diff"}, nd, []string{"--stat", "HEAD"}), false},
		{cat([]string{"git", "log"}, nd, []string{"--oneline", "-n"}), true},
		{cat([]string{"git", "log"}, nd, []string{"--oneline"}), false},
		{[]string{"git", "rev-parse", "--is-inside-work-tree"}, false},
		{[]string{"git", "cat-file", "-p", "HEAD:a"}, false},
		{[]string{"git", "config", "a.b", "c"}, true},
		{[]string{"git", "-C", "/x", "config", "a.b", "c"}, true},
		{[]string{"git", "-C", "/x", "-C", "y", "commit"}, true},
		{[]string{"git", "auditconfig"}, true},
		{[]string{"git", "init", "-q", "/x"}, true},
		{[]string{"git", "clone", "/x", "/y"}, true},
		{[]string{"git", "-c", "alias.s=!rm", "s"}, true},
		{[]string{"git", "--git-dir=/x", "status"}, true},
		{[]string{"git", "--work-tree", "/x", "status"}, true},
		{[]string{"git", "--config-env=a=B", "status"}, true},
		{[]string{"git", "-C"}, true},
		{[]string{"git"}, true},
		{[]string{"git", "grep", "x"}, true},
		{[]string{"git", "blame", "a"}, true},
		{[]string{"git", "diff", "HEAD^", "HEAD"}, true},
		{cat([]string{"git", "diff", "--no-ext-diff"}, []string{"HEAD"}), true},
		{cat([]string{"git", "diff"}, nd, []string{"--output=/x"}), true},
		{cat([]string{"git", "diff"}, nd, []string{"--ext-diff"}), true},
		{cat([]string{"git", "diff"}, nd, []string{"--ext"}), true},
		{cat([]string{"git", "diff"}, nd, []string{"--no-ext-d"}), true},
		{cat([]string{"git", "show"}, nd, []string{"HEAD"}), false},
		{[]string{"git", "show", "HEAD"}, true},
		{[]string{"git", "cat-file", "--filters", "HEAD:a"}, true},
		{[]string{"git", "status", "--ignored"}, true},
		{[]string{"git", "status", ""}, true},
		{cat([]string{"git", "diff"}, nd, []string{"--", "--output=x"}), false},
	} {
		if got := unsafeLoopArgv(tc.argv); got != tc.unsafe {
			t.Errorf("%v: unsafe=%v want %v", tc.argv, got, tc.unsafe)
		}
	}
}

// Liz D5a r5 F09 end to end: a git check is refused, and nothing is
// credited, when the repository's local configuration changed after the
// independent review to name a signature program.
func TestCreditRefusedUnderHelperConfiguration(t *testing.T) {
	ctl := newCreditFixtureArgs(t, "git", []string{"status", "-s"}, model.CriterionPending)
	if _, err := ctl.credit(ctl.approve("pass")); err != nil {
		t.Fatalf("control: a plain git status check credits: %v", err)
	}
	cf := newCreditFixtureArgs(t, "git", []string{"status", "-s"}, model.CriterionPending)
	report := cf.approve("pass")
	cfgPath := filepath.Join(cf.root, ".git", "config")
	orig, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "marker")
	helper := filepath.Join(t.TempDir(), "gpg.sh")
	os.WriteFile(helper, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o700)
	os.WriteFile(cfgPath, append(orig, []byte("[log]\n\tshowSignature = true\n[gpg]\n\tprogram = "+helper+"\n")...), 0o600)
	if _, err := cf.credit(report); err == nil {
		t.Fatal("credit granted under helper configuration")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("helper ran")
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
}

// D5a r7 AUD-D5A-01: a file swapped in during the check and restored before
// the second snapshot leaves equal snapshots; the metadata fingerprint
// still refuses the credit.
func TestCreditRefusedAfterSwapAndRestore(t *testing.T) {
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	tracked := filepath.Join(cf.root, "a")
	orig, err := os.ReadFile(tracked)
	if err != nil {
		t.Fatal(err)
	}
	creditCheckHook = func() {
		os.WriteFile(tracked, []byte("TRANSIENT\n"), 0o600)
		os.WriteFile(tracked, orig, 0o600)
	}
	defer func() { creditCheckHook = nil }()
	_, err = cf.credit(report)
	if err == nil {
		t.Fatal("credit granted after a swap and restore")
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
	creditCheckHook = nil
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("control without a writer: %v", err)
	}
}

// D5a r11 D5AR11P-F01: a file that a committed .gitignore hides is not part
// of the reviewed target. A check that needs it fails even after the file
// appears, because the check runs on a private copy of
// the target, not in the live worktree.
func TestCreditIgnoredInputNotVisibleToCheck(t *testing.T) {
	creditRepoSetup = func(f *reviewFixture) {
		if err := os.WriteFile(filepath.Join(f.root, ".gitignore"), []byte("scratch/\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"add", ".gitignore"}, {"commit", "-qm", "ignore scratch"}} {
			c := exec.Command("git", append([]string{"-C", f.root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...)
			if out, err := c.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v %s", args, err, out)
			}
		}
		st, err := repostate.Snapshot(context.Background(), f.root, f.base, false)
		if err != nil {
			t.Fatal(err)
		}
		f.head, f.diff = st.Head, st.Diff
	}
	defer func() { creditRepoSetup = nil }()
	cf := newCreditFixtureArgs(t, "/usr/bin/test", []string{"-f", "scratch/input"}, model.CriterionPending)
	report := cf.approve("pass")
	if err := os.MkdirAll(filepath.Join(cf.root, "scratch"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cf.root, "scratch", "input"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("credit: %v", err)
	}
	if cf.criteria()["C1"] != model.CriterionFailed {
		t.Fatalf("C1 = %v: an ignored file outside the reviewed target must not satisfy the check", cf.criteria()["C1"])
	}
}

// The check runs on a copy: its writes never reach the live worktree.
func TestCreditCheckWritesStayInCopy(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/usr/bin/touch", []string{"made-by-check"}, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("credit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cf.root, "made-by-check")); err == nil {
		t.Fatal("check wrote into the live worktree")
	}
	if cf.criteria()["C1"] != model.CriterionPassed {
		t.Fatalf("C1 = %v, want PASSED", cf.criteria()["C1"])
	}
}

// D5a r12 D5AR12P-F01: a committed symlink that leaves the target, or a check
// that names the live repository path, never reaches the live worktree.
func TestCreditRefusesLiveWorktreeEscapes(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/usr/bin/touch", []string{"a"}, model.CriterionPending)
	report := cf.approve("pass")
	os.Symlink(cf.root, filepath.Join(cf.root, "escape"))
	if _, err := cf.credit(report); err == nil {
		t.Fatal("credit granted with an escaping symlink in the target")
	}
	os.Remove(filepath.Join(cf.root, "escape"))

}

func TestCreditRefusesCheckNamingLiveRepository(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/usr/bin/test", []string{"-f", "@ROOT@/a"}, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err == nil {
		t.Fatal("credit granted for a check that names the live repository path")
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
}

// D5a r12 D5AR12P-F02: the evidence names the directory where the command
// ran and the changed paths of the reviewed target.
func TestCreditEvidenceRecordsExecutionAndChangedFiles(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/usr/bin/pwd", nil, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatal(err)
	}
	core, _ := store.New(cf.store).Load("run")
	var ev *model.EvidenceRef
	for _, r := range core {
		if r.Kind == model.KindEvidence {
			ev = r.Evidence
		}
	}
	if ev == nil {
		t.Fatal("no evidence")
	}
	if !strings.Contains(ev.WorkingDir, "hma-target-") {
		t.Fatalf("recorded dir %q is not where the command ran", ev.WorkingDir)
	}
	found := false
	for _, c := range ev.ChangedFiles {
		found = found || c == "a"
	}
	if !found {
		t.Fatalf("changed files %v omit the reviewed change", ev.ChangedFiles)
	}
}

func gitIn(t *testing.T, root string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

// D5a r13 D5AR13P-F01: another spelling of the live repository path (an
// absolute symlink alias) reaches neither an ignored input nor a write: the
// sandbox shows the check only the private copy.
func TestCreditAliasToLiveRepositoryUnreachable(t *testing.T) {
	alias := filepath.Join(t.TempDir(), "alias")
	creditAlias = alias
	creditRepoSetup = func(f *reviewFixture) {
		os.WriteFile(filepath.Join(f.root, ".gitignore"), []byte("scratch/\n"), 0o600)
		gitIn(t, f.root, "add", ".gitignore")
		gitIn(t, f.root, "commit", "-qm", "ignore scratch")
		st, err := repostate.Snapshot(context.Background(), f.root, f.base, false)
		if err != nil {
			t.Fatal(err)
		}
		f.head, f.diff = st.Head, st.Diff
		os.Remove(alias)
		if err := os.Symlink(f.root, alias); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { creditRepoSetup, creditAlias = nil, "" }()
	for _, tc := range []struct {
		exe  string
		argv []string
	}{{"/usr/bin/test", []string{"-f", "@ALIAS@/scratch/input"}}, {"/usr/bin/touch", []string{"@ALIAS@/scratch/written"}}} {
		cf := newCreditFixtureArgs(t, tc.exe, tc.argv, model.CriterionPending)
		report := cf.approve("pass")
		os.MkdirAll(filepath.Join(cf.root, "scratch"), 0o700)
		os.WriteFile(filepath.Join(cf.root, "scratch", "input"), []byte("x\n"), 0o600)
		cf.credit(report)
		if tc.exe == "/usr/bin/test" && cf.criteria()["C1"] == model.CriterionPassed {
			t.Fatal("an ignored live file satisfied the check through an alias")
		}
		if _, err := os.Stat(filepath.Join(cf.root, "scratch", "written")); err == nil {
			t.Fatal("the check wrote into the live worktree")
		}
	}
}

// D5a r13 D5AR13P-F02: two links that are each inside the target but resolve
// outside it once composed never reach a file outside the copy.
func TestCreditChainedSymlinkUnreachable(t *testing.T) {
	marker := filepath.Join(os.TempDir(), fmt.Sprintf("hma-chain-marker-%d", os.Getpid()))
	os.WriteFile(marker, []byte("x\n"), 0o600)
	defer os.Remove(marker)
	creditRepoSetup = func(f *reviewFixture) {
		os.Mkdir(filepath.Join(f.root, "sub"), 0o700)
		os.Symlink("..", filepath.Join(f.root, "sub", "up"))
		os.Symlink("sub/up/../"+filepath.Base(marker), filepath.Join(f.root, "escape"))
		gitIn(t, f.root, "add", "-A")
		gitIn(t, f.root, "commit", "-qm", "links")
		st, err := repostate.Snapshot(context.Background(), f.root, f.base, false)
		if err != nil {
			t.Fatal(err)
		}
		f.head, f.diff = st.Head, st.Diff
	}
	defer func() { creditRepoSetup = nil }()
	cf := newCreditFixtureArgs(t, "/usr/bin/test", []string{"-f", "escape"}, model.CriterionPending)
	report := cf.approve("pass")
	cf.credit(report)
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("a composed symlink reached a file outside the private copy")
	}
}

// D5a r13 D5AR13P-F03: a ref created in the live repository after review is
// invisible to a git check: it sees the reviewed HEAD and no other ref.
func TestCreditGitSeesOnlyReviewedHead(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/usr/bin/git", []string{"merge-base", "--is-ancestor", "refs/heads/auditor-side", "HEAD"}, model.CriterionPending)
	report := cf.approve("pass")
	gitIn(t, cf.root, "update-ref", "refs/heads/auditor-side", "HEAD")
	if _, err := cf.credit(report); err != nil {
		t.Fatal(err)
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("a live ref created after review satisfied the check")
	}
}

// D5a r14 D5AR14P-F01: an object added to the live repository after review
// is not in the check's private object store.
func TestCreditGitSeesOnlyReviewedObjects(t *testing.T) {
	// The id is that of the blob the test adds after review.
	c := exec.Command("git", "hash-object", "--stdin")
	c.Stdin = strings.NewReader("unreviewed object\n")
	out, err := c.Output()
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(string(out))
	cf := newCreditFixtureArgs(t, "/usr/bin/git", []string{"cat-file", "-e", id}, model.CriterionPending)
	report := cf.approve("pass")
	add := exec.Command("git", "-C", cf.root, "hash-object", "-w", "--stdin")
	add.Stdin = strings.NewReader("unreviewed object\n")
	if _, err := add.Output(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", cf.root, "cat-file", "-e", id).Run(); err != nil {
		t.Fatal("control: the object is not in the live repository")
	}
	if _, err := cf.credit(report); err != nil {
		t.Fatal(err)
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("an object added after review satisfied the check")
	}
}

// D5a r14 D5AR14P-F02: the check's index is built from the reviewed commit,
// not copied from the live index, so a staged entry never appears in it.
func TestCreditGitIndexIsReviewedCommit(t *testing.T) {
	cf := newCreditFixtureArgs(t, "/usr/bin/git", []string{"ls-files", "--error-unmatch", "phantom"}, model.CriterionPending)
	report := cf.approve("pass")
	os.WriteFile(filepath.Join(cf.root, "phantom"), []byte("p\n"), 0o600)
	gitIn(t, cf.root, "add", "phantom")
	if _, err := cf.credit(report); err == nil && cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("a live staged entry satisfied the check")
	}
}

// D5a r24 F1: a live writer that changes .git state (here .git/info/exclude,
// which decides what Materialize copies) while a check runs and restores it
// is refused: the fingerprint covers .git.
func TestCreditRefusedAfterGitDirSwapAndRestore(t *testing.T) {
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	ex := filepath.Join(cf.root, ".git", "info", "exclude")
	orig, _ := os.ReadFile(ex)
	creditCheckHook = func() {
		os.WriteFile(ex, append(append([]byte(nil), orig...), []byte("a\n")...), 0o600)
		os.WriteFile(ex, orig, 0o600)
	}
	defer func() { creditCheckHook = nil }()
	if _, err := cf.credit(report); err == nil {
		t.Fatal("credit granted after a swap and restore inside .git")
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
	creditCheckHook = nil
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("control without a writer: %v", err)
	}
}

// D5a r24 F2: a project root reached through a symlink is fingerprinted as
// the directory it names.
func TestCreditThroughSymlinkedRoot(t *testing.T) {
	ctx := context.Background()
	cf := newCreditFixture(t, "true")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(cf.root, link); err != nil {
		t.Fatal(err)
	}
	a, err := repostate.Fingerprint(ctx, link)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := repostate.Fingerprint(ctx, cf.root)
	if a != b {
		t.Fatal("symlinked root not resolved")
	}
}

// D5a r25 N1: a subtree object forged in place before the review and the
// credit (git ls-tree does not hash-check subtrees, so the snapshot, the
// review digest and the copy all agree on the forged history) is refused by
// the object-store check.
func TestCreditRefusedWithForgedObject(t *testing.T) {
	creditRepoSetup = func(f *reviewFixture) {
		os.MkdirAll(filepath.Join(f.root, "sub"), 0o700)
		os.WriteFile(filepath.Join(f.root, "sub", "k"), []byte("k\n"), 0o600)
		os.WriteFile(filepath.Join(f.root, ".gitignore"), []byte("evil.local\n"), 0o600)
		gitIn(t, f.root, "add", "-A")
		gitIn(t, f.root, "commit", "-qm", "sub")
		gitOut := func(stdin string, args ...string) string {
			c := exec.Command("git", append([]string{"-C", f.root}, args...)...)
			c.Stdin = strings.NewReader(stdin)
			o, err := c.Output()
			if err != nil {
				t.Fatalf("git %v: %v", args, err)
			}
			return strings.TrimSpace(string(o))
		}
		sub := gitOut("", "rev-parse", "HEAD:sub")
		evil := gitOut("evil\n", "hash-object", "-w", "--stdin")
		forged := gitOut(gitOut("", "ls-tree", "HEAD:sub")+"\n100644 blob "+evil+"\tevil.local\n", "mktree")
		os.WriteFile(filepath.Join(f.root, "sub", "evil.local"), []byte("evil\n"), 0o600)
		path := func(id string) string { return filepath.Join(f.root, ".git", "objects", id[:2], id[2:]) }
		data, err := os.ReadFile(path(forged))
		if err != nil {
			t.Skipf("forged tree not loose: %v", err)
		}
		os.Chmod(path(sub), 0o600)
		os.WriteFile(path(sub), data, 0o400)
		st, err := repostate.Snapshot(context.Background(), f.root, f.base, false)
		if err != nil {
			t.Fatal(err)
		}
		f.head, f.diff = st.Head, st.Diff
	}
	defer func() { creditRepoSetup = nil }()
	cf := newCreditFixtureArgs(t, "/usr/bin/test", []string{"-f", "sub/evil.local"}, model.CriterionPending)
	report := cf.approve("pass")
	_, err := cf.credit(report)
	if err == nil {
		t.Fatal("credit granted over a forged subtree")
	}
	if !strings.Contains(err.Error(), "object") {
		t.Fatalf("refused for another reason: %v", err)
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
}

// D5a r27 R1: a .git/commondir file redirects git to an object store outside
// .git (outside the fingerprint and outside the object check); credit is
// refused before any git read.
func TestCreditRefusedWithCommonDirRedirect(t *testing.T) {
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	os.WriteFile(filepath.Join(cf.root, ".git", "commondir"), []byte(t.TempDir()+"\n"), 0o600)
	_, err := cf.credit(report)
	if err == nil {
		t.Fatal("credit granted with a commondir redirect")
	}
	if !strings.Contains(err.Error(), "commondir") {
		t.Fatalf("refused for another reason: %v", err)
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
}

// D5a r28: a local setting that changes how git reads objects
// (core.multiPackIndex) makes HMA's own reads disagree with the verified
// store; credit is refused before any git read. A plain config credits.
func TestCreditRefusedWithObjectReadingConfig(t *testing.T) {
	cf := newCreditFixture(t, "true")
	report := cf.approve("pass")
	gitIn(t, cf.root, "config", "core.multiPackIndex", "false")
	_, err := cf.credit(report)
	if err == nil {
		t.Fatal("credit granted under core.multiPackIndex")
	}
	if !strings.Contains(err.Error(), "not plain") {
		t.Fatalf("refused for another reason: %v", err)
	}
	if cf.criteria()["C1"] == model.CriterionPassed {
		t.Fatal("criterion credited")
	}
	gitIn(t, cf.root, "config", "--unset", "core.multiPackIndex")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("control with a plain config: %v", err)
	}
}
