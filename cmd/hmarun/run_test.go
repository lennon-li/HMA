package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// --- shared test fixtures ----------------------------------------------

var (
	hmaBinOnce sync.Once
	hmaBinPath string
	hmaBinErr  error
)

// goBinary locates a usable `go` binary: go may not be on PATH, so this
// falls back to GOROOT and then to the fixed local install path the
// dispatch names.
func goBinary() string {
	if p, err := exec.LookPath("go"); err == nil {
		return p
	}
	if root := os.Getenv("GOROOT"); root != "" {
		if p := filepath.Join(root, "bin", "go"); fileExists(p) {
			return p
		}
	}
	const fallback = "/home/yeli/.local/go/bin/go"
	if fileExists(fallback) {
		return fallback
	}
	return "go"
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// buildHMABinary compiles the real, unmodified hma binary once per test
// process and returns its path. hmarun run shells out to exactly this
// binary, never to anything in-process: cmd/hma is package main and not
// importable.
func buildHMABinary(t *testing.T) string {
	t.Helper()
	hmaBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "hmarun-test-hma-bin-")
		if err != nil {
			hmaBinErr = err
			return
		}
		hmaBinPath = filepath.Join(dir, "hma")
		cmd := exec.Command(goBinary(), "build", "-o", hmaBinPath, "github.com/lennon-li/HMA/cmd/hma")
		cmd.Dir = repoRootForTest(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			hmaBinErr = &buildError{err, out}
		}
	})
	if hmaBinErr != nil {
		t.Fatalf("building hma: %v", hmaBinErr)
	}
	return hmaBinPath
}

type buildError struct {
	err error
	out []byte
}

func (b *buildError) Error() string { return b.err.Error() + ": " + string(b.out) }

// repoRootForTest returns the HMA repository root, derived from this test
// file's own location (cmd/hmarun), so the build works regardless of the
// process's current working directory.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd)) // cmd/hmarun -> cmd -> repo root
}

// hmarunTestRepo returns a fresh Git repository with one committed file,
// "a", which the test plan's single unit is allowed to change.
func hmarunTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("-c", "user.name=hmarun-test", "-c", "user.email=hmarun-test@example.invalid", "commit", "--allow-empty", "-qm", "base")
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("-c", "user.name=hmarun-test", "-c", "user.email=hmarun-test@example.invalid", "add", "a")
	run("-c", "user.name=hmarun-test", "-c", "user.email=hmarun-test@example.invalid", "commit", "-qm", "seed file a")
	return dir
}

// hmarunTestDocs builds a contract with two criteria and a plan with one
// unit that advances only the first, so the run's minimum path to PARTIAL
// is: C1 credited PASSED, C2 left PENDING (see the run.go package doc and
// the dispatch report for why a single-criterion plan can never reach
// PARTIAL at all).
func hmarunTestDocs(t *testing.T, outDir string) (contractPath, planPath string) {
	t.Helper()
	answers := Answers{
		RunID: "run", Intent: "hmarun MVP wrapper smoke run", Risk: "low", NonGoals: []string{"no unrelated change"},
		Criteria: []CriterionAnswer{
			{ID: "C1", Statement: "check passes", Threshold: "exit 0", Check: CheckAnswer{Executable: "true", Argv: []string{}}},
			{ID: "C2", Statement: "other criterion, not advanced by U1", Threshold: "exit 0", Check: CheckAnswer{Executable: "true", Argv: []string{}}},
		},
		Units: []UnitAnswer{
			{ID: "U1", AllowedFiles: []string{"a"}, ForbiddenFiles: []string{"docs"}, CriteriaAdvanced: []string{"C1"}},
			// U2 exists only so the contract's C2 (never advanced by U1) has
			// some unit advancing it, as model.ValidatePlanAgainstContract
			// requires; hmarun run never touches U2. This is what keeps C2 at
			// its seeded PENDING through the whole run: see the run.go
			// package doc on why PARTIAL needs at least one resolved and one
			// unresolved criterion.
			{ID: "U2", AllowedFiles: []string{"b"}, ForbiddenFiles: []string{"docs"}, CriteriaAdvanced: []string{"C2"}},
		},
	}
	contract, _, plan, err := buildContractAndPlan(answers)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contractPath = filepath.Join(outDir, "contract.json")
	planPath = filepath.Join(outDir, "plan.json")
	writeIndented(t, contractPath, contract)
	writeIndented(t, planPath, plan)
	return contractPath, planPath
}

func writeIndented(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- end-to-end -----------------------------------------------------------

// TestRunReachesPartial runs `hmarun run --auto-approve` end to end against
// a throwaway repository and a scratch store, and asserts it reaches
// PARTIAL with exactly the criterion dispositions the fixture's plan should
// produce: C1 (the only criterion U1 advances) credited PASSED by `hma
// host credit`, C2 left at its seeded PENDING, never touched.
func TestRunReachesPartial(t *testing.T) {
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)

	var stdout bytes.Buffer
	args := []string{
		"--store", storeDir, "--repo", repo, "--run_id", "run",
		"--contract", contractPath, "--plan", planPath, "--unit", "U1",
		"--hma-bin", hmaBin, "--auto-approve", "--stub-dispatch",
	}
	if err := runRun(args, bytes.NewReader(nil), &stdout); err != nil {
		t.Fatalf("hmarun run: %v\n--- output ---\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "reached terminal PARTIAL") {
		t.Fatalf("stdout does not report success: %s", stdout.String())
	}

	core, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	state := engine.ProjectStageState(core)
	if !state.Terminal || state.Outcome != model.OutcomePartial {
		t.Fatalf("terminal=%v outcome=%q, want PARTIAL", state.Terminal, state.Outcome)
	}
	got := map[string]model.CriterionDisposition{}
	for _, c := range engine.ProjectResolutionState(core).Criteria {
		got[c.ID] = c.Disposition
	}
	if got["C1"] != model.CriterionPassed {
		t.Fatalf("C1 = %v, want PASSED", got["C1"])
	}
	if got["C2"] != model.CriterionPending {
		t.Fatalf("C2 = %v, want PENDING", got["C2"])
	}

	// The stub worker's commit actually landed in the repository.
	out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%s").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if msg := strings.TrimSpace(string(out)); msg != stubWorkerCommitMessage {
		t.Fatalf("HEAD commit message = %q, want %q", msg, stubWorkerCommitMessage)
	}

	// Re-running a completed run is a no-op: no step repeats or duplicates
	// a record.
	recordsBefore := len(core)
	hostBefore, err := hoststate.NewStore(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	hostRecordsBefore := len(hostBefore)
	var stdout2 bytes.Buffer
	if err := runRun(args, bytes.NewReader(nil), &stdout2); err != nil {
		t.Fatalf("second hmarun run: %v\n--- output ---\n%s", err, stdout2.String())
	}
	core2, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(core2) != recordsBefore {
		t.Fatalf("re-running a completed run changed the core chain: %d -> %d records", recordsBefore, len(core2))
	}
	host2, err := hoststate.NewStore(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(host2) != hostRecordsBefore {
		t.Fatalf("re-running a completed run changed the host chain: %d -> %d records", hostRecordsBefore, len(host2))
	}
}

// TestRunResumesAfterPartialProgress simulates a crash partway through the
// flow (by calling the runner's own step methods directly up to a point,
// bypassing run's loop) and confirms a fresh `hmarun run` invocation against
// the same store and repository picks up from the first undone step,
// without redoing or duplicating anything already committed.
func TestRunResumesAfterPartialProgress(t *testing.T) {
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)

	cfg := runConfig{
		storeDir: storeDir, repoRoot: repo, runID: "run", unit: "U1",
		contractPath: contractPath, planPath: planPath, hmaBin: hmaBin, autoApprove: true, stubDispatch: true,
	}
	r, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// Advance only through the implementation dispatch (step 7 of the
	// reference sequence): the core chain is mid-flow, no host decision's
	// implementation has even been produced yet.
	for _, step := range []func(context.Context) error{
		r.stepAcceptanceCriteria, r.stepPlanning, r.stepRouteSelection, r.stepRouteApproval,
		r.stepDecision, r.stepImplPreflight, r.stepImplDispatch,
	} {
		if err := step(ctx); err != nil {
			t.Fatalf("partial progress: %v", err)
		}
	}

	core, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	state := engine.ProjectStageState(core)
	if state.CurrentStage != model.StageImplementationAuthorization {
		t.Fatalf("stage after partial progress = %s, want IMPLEMENTATION_AUTHORIZATION", state.CurrentStage)
	}
	history, err := hoststate.NewStore(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	var decisions, dispatches int
	for _, rec := range history {
		if rec.Kind == hoststate.KindDecision {
			decisions++
		}
		if rec.Kind == hoststate.KindDispatch {
			dispatches++
		}
	}
	if decisions != 1 || dispatches != 1 {
		t.Fatalf("decisions=%d dispatches=%d after partial progress, want 1 and 1", decisions, dispatches)
	}

	// A fresh runner against the same store and repository resumes and
	// completes, reaching PARTIAL exactly as the end-to-end test does.
	var stdout bytes.Buffer
	args := []string{
		"--store", storeDir, "--repo", repo, "--run_id", "run",
		"--contract", contractPath, "--plan", planPath, "--unit", "U1",
		"--hma-bin", hmaBin, "--auto-approve", "--stub-dispatch",
	}
	if err := runRun(args, bytes.NewReader(nil), &stdout); err != nil {
		t.Fatalf("resumed hmarun run: %v\n--- output ---\n%s", err, stdout.String())
	}

	core2, err := store.New(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	state2 := engine.ProjectStageState(core2)
	if !state2.Terminal || state2.Outcome != model.OutcomePartial {
		t.Fatalf("resumed run: terminal=%v outcome=%q, want PARTIAL", state2.Terminal, state2.Outcome)
	}
	history2, err := hoststate.NewStore(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	var decisions2, dispatches2 int
	for _, rec := range history2 {
		if rec.Kind == hoststate.KindDecision {
			decisions2++
		}
		if rec.Kind == hoststate.KindDispatch && rec.Dispatch != nil && rec.Dispatch.Role == hoststate.RoleImplementation {
			dispatches2++
		}
	}
	if decisions2 != 1 {
		t.Fatalf("decisions after resume = %d, want 1 (resume must not duplicate the already-recorded decision)", decisions2)
	}
	if dispatches2 != 1 {
		t.Fatalf("implementation dispatches after resume = %d, want 1 (resume must not duplicate the already-recorded dispatch)", dispatches2)
	}
}

// TestStubReviewMarkerAppearsInChainAndApprovalPrompts is F2's test: while
// stepReviewerStub's synthetic AUDIT.json stands in for a real independent
// review, every host record that carries or depends on it must show an
// unmistakable stub marker in a field `hma host show` actually prints
// (envelope.Actor, not audit_id/limitations, which review-import never
// captures -- see stubReviewMarker's doc comment), and every human-facing
// approval summary for a step whose verdict rests on that review
// (verification, independent-validation, credit, PARTIAL) must carry the
// same marker as a prefix.
func TestStubReviewMarkerAppearsInChainAndApprovalPrompts(t *testing.T) {
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)

	var stdout bytes.Buffer
	args := []string{
		"--store", storeDir, "--repo", repo, "--run_id", "run",
		"--contract", contractPath, "--plan", planPath, "--unit", "U1",
		"--hma-bin", hmaBin, "--auto-approve", "--stub-dispatch",
	}
	if err := runRun(args, bytes.NewReader(nil), &stdout); err != nil {
		t.Fatalf("hmarun run: %v\n--- output ---\n%s", err, stdout.String())
	}

	host, err := hoststate.NewStore(storeDir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	checkedKinds := map[hoststate.RecordKind]bool{}
	for _, rec := range host {
		switch rec.Kind {
		case hoststate.KindDispatch:
			if rec.Dispatch == nil || rec.Dispatch.Role != hoststate.RoleReview {
				continue
			}
		case hoststate.KindReview, hoststate.KindCriterionCredit:
			// fall through to the marker check below.
		default:
			continue
		}
		checkedKinds[rec.Kind] = true
		if !strings.Contains(rec.Actor, stubReviewMarker) {
			t.Errorf("host record kind=%s actor=%q does not carry the stub review marker %q", rec.Kind, rec.Actor, stubReviewMarker)
		}
	}
	for _, kind := range []hoststate.RecordKind{hoststate.KindDispatch, hoststate.KindReview, hoststate.KindCriterionCredit} {
		if !checkedKinds[kind] {
			t.Errorf("no host record of kind %s was found to check for the stub review marker", kind)
		}
	}

	out := stdout.String()
	for _, want := range []string{
		"reviewed implementation for verification",
		"independent validation",
		"criterion",
		"as PARTIAL",
	} {
		idx := strings.Index(out, want)
		if idx < 0 {
			t.Fatalf("stdout does not contain expected approval summary fragment %q:\n%s", want, out)
		}
		// The marker must prefix (appear at or before) the summary line
		// that contains this fragment, on the same printed block.
		lineStart := strings.LastIndex(out[:idx], "\n")
		if !strings.Contains(out[lineStart+1:idx], stubReviewMarker) {
			t.Errorf("approval summary containing %q is not prefixed with the stub review marker; line: %q", want, out[lineStart+1:idx+len(want)])
		}
	}
}

// resumeAtStep advances a brand-new runner through the run's fixed step
// sequence (runSteps) up to and including stopAfter, then abandons it
// without calling run -- standing in for a crash right after that step,
// the same place in the sequence a second real `hmarun run` process would
// have stopped. It returns the original base_revision r computed, so the
// caller can confirm a later, independently constructed runner reads the
// same one back from the chain instead of recomputing it from a moved HEAD
// (F1).
func resumeAtStep(t *testing.T, cfg runConfig, stopAfter string) (originalBase string) {
	t.Helper()
	r, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, s := range r.runSteps() {
		if err := s.fn(ctx); err != nil {
			t.Fatalf("partial progress at step %s: %v", s.name, err)
		}
		if s.name == stopAfter {
			return r.base
		}
	}
	t.Fatalf("stopAfter %q does not name a step in runSteps", stopAfter)
	return ""
}

// assertResumesToPartial builds a fresh runner against cfg's store and
// repository -- exactly as a second, independent `hmarun run` process would
// after a crash, never the same *runner the interrupted progress used -- and
// confirms it resumes to terminal PARTIAL without error. When originalBase
// is non-empty it also confirms the fresh runner's base_revision matches it
// exactly, which is the heart of the F1 fix: a resumed run must keep using
// the base the interrupted run was opened against, not recompute it from
// whatever the repository's HEAD has since moved to (e.g. stepWorkerStub's
// commit).
func assertResumesToPartial(t *testing.T, cfg runConfig, originalBase string) {
	t.Helper()
	r2, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("fresh runner: %v", err)
	}
	if originalBase != "" && r2.base != originalBase {
		t.Fatalf("resumed runner base = %q, want original base %q", r2.base, originalBase)
	}
	if err := r2.run(context.Background()); err != nil {
		t.Fatalf("resumed run: %v", err)
	}
	core2, err := store.New(cfg.storeDir).Load(cfg.runID)
	if err != nil {
		t.Fatal(err)
	}
	state2 := engine.ProjectStageState(core2)
	if !state2.Terminal || state2.Outcome != model.OutcomePartial {
		t.Fatalf("resumed run: terminal=%v outcome=%q, want PARTIAL", state2.Terminal, state2.Outcome)
	}
}

// TestRunResumesAfterWorkerStub is the gap in test coverage Phil's review
// named directly (F1): TestRunResumesAfterPartialProgress only stops before
// stepWorkerStub's commit, which is exactly why it passed even with the F1
// bug (newRunner always recomputing base from live HEAD) in place. This test
// stops right after that commit -- which moves the repository's HEAD past
// the base every earlier approval in the chain is bound to -- and confirms a
// fresh runner still resumes correctly, rather than wedging permanently with
// HOST_REVIEW_REPOSITORY_MISMATCH / HOST_REVIEW_ARTIFACT_MISMATCH /
// HOST_CREDIT_REFUSED the way Phil's crash injection demonstrated.
func TestRunResumesAfterWorkerStub(t *testing.T) {
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)
	cfg := runConfig{
		storeDir: storeDir, repoRoot: repo, runID: "run", unit: "U1",
		contractPath: contractPath, planPath: planPath, hmaBin: hmaBin, autoApprove: true, stubDispatch: true,
	}

	base := resumeAtStep(t, cfg, "worker-stub")

	headOut, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if head := strings.TrimSpace(string(headOut)); head == base {
		t.Fatalf("stepWorkerStub did not move HEAD past the frozen base %q", base)
	}

	assertResumesToPartial(t, cfg, base)
}

// TestRunResumesAtFurtherCrashPointsAfterWorkerStub covers the further
// resume points Phil's review listed as failing before the F1 fix: a crash
// in review-packet, reconcile, or credit, all after the worker's commit has
// already moved HEAD. Each subtest starts its own fresh repository and
// store, so the cases are independent.
func TestRunResumesAtFurtherCrashPointsAfterWorkerStub(t *testing.T) {
	hmaBin := buildHMABinary(t)
	for _, stopAfter := range []string{"review-packet", "reconcile", "credit"} {
		t.Run(stopAfter, func(t *testing.T) {
			repo := hmarunTestRepo(t)
			storeDir := t.TempDir()
			docsDir := t.TempDir()
			contractPath, planPath := hmarunTestDocs(t, docsDir)
			cfg := runConfig{
				storeDir: storeDir, repoRoot: repo, runID: "run", unit: "U1",
				contractPath: contractPath, planPath: planPath, hmaBin: hmaBin, autoApprove: true, stubDispatch: true,
			}
			base := resumeAtStep(t, cfg, stopAfter)
			assertResumesToPartial(t, cfg, base)
		})
	}
}
