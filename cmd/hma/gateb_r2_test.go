package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func bGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...)
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v %s", args, e, b)
	}
	return strings.TrimSpace(string(b))
}
func bStoreBytes(t *testing.T, root string) string {
	t.Helper()
	root, _ = filepath.EvalSymlinks(root)
	h := sha256.New()
	filepath.Walk(root, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if !i.IsDir() && !strings.HasSuffix(p, ".lock") {
			b, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			fmt.Fprint(h, strings.TrimPrefix(p, root))
			h.Write(b)
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}
func bCI(t *testing.T, root, base, head, diff string, exe string, args ...string) error {
	t.Helper()
	dir := t.TempDir()
	seedApprovedVerificationAt(t, dir, "run", "owner/repo", base, head, diff)
	setGitHubEnv(t, "owner/repo", head)
	a := []string{"--store", dir, "--run_id", "run", "--repo-root", root, "--exec", exe}
	for _, arg := range args {
		a = append(a, "--arg", arg)
	}
	before := bStoreBytes(t, dir)
	e := runCIGithub(a)
	rs, le := store.New(dir).Load("run")
	if le != nil {
		t.Fatal(le)
	}
	if e != nil && bStoreBytes(t, dir) != before {
		t.Fatal("refusal changed stored bytes")
	}
	if e == nil {
		t.Logf("ACCEPTED evidence=%+v", rs[len(rs)-1].Evidence)
	}
	return e
}
func TestGateBR2CIHiddenDirty(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			r, b, h, d := ciRepo(t)
			bGit(t, r, "update-index", flag, "a")
			os.WriteFile(filepath.Join(r, "a"), []byte("UNAPPROVED BYTES\n"), 0600)
			t.Logf("git status=%q content=UNAPPROVED BYTES", bGit(t, r, "status", "--porcelain"))
			if e := bCI(t, r, b, h, d, "/bin/cat", "a"); e == nil {
				t.Error("CI labelled unapproved worktree bytes as approved clean head")
			}
		})
	}
}
func TestGateBR2CIGitEnvironment(t *testing.T) {
	r, b, h, d := ciRepo(t)
	other, _, _, _ := ciRepo(t)
	bGit(t, other, "commit", "--allow-empty", "-qm", "foreign")
	foreign := bGit(t, other, "rev-parse", "HEAD")
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	e := bCI(t, r, b, h, d, "/usr/bin/git", "rev-parse", "HEAD")
	if e == nil {
		t.Errorf("CI metadata names %s but command git reads foreign HEAD %s", h, foreign)
	}
}
func TestGateBR2CISymlinkAndWorktree(t *testing.T) {
	r, b, h, d := ciRepo(t)
	ln := filepath.Join(t.TempDir(), "link")
	if e := os.Symlink(r, ln); e != nil {
		t.Fatal(e)
	}
	if e := bCI(t, ln, b, h, d, "/bin/true"); e != nil {
		t.Fatalf("valid symlink: %v", e)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	bGit(t, r, "worktree", "add", "--detach", wt, h)
	if e := bCI(t, wt, b, h, d, "/bin/true"); e != nil {
		t.Fatalf("valid linked worktree: %v", e)
	}
	os.WriteFile(filepath.Join(wt, "a"), []byte("dirty"), 0600)
	if e := bCI(t, wt, b, h, d, "/bin/true"); e == nil {
		t.Fatal("dirty linked worktree accepted")
	}
}
func TestGateBR2CIStoreSymlink(t *testing.T) {
	r, b, h, d := ciRepo(t)
	inside := filepath.Join(r, "store")
	os.Mkdir(inside, 0700)
	outside := filepath.Join(t.TempDir(), "alias")
	os.Symlink(inside, outside)
	seedApprovedVerificationAt(t, outside, "run", "owner/repo", b, h, d)
	setGitHubEnv(t, "owner/repo", h)
	before := bStoreBytes(t, outside)
	e := runCIGithub([]string{"--store", outside, "--run_id", "run", "--repo-root", r, "--exec", "/bin/true"})
	if e == nil || !strings.Contains(e.Error(), "outside") {
		t.Fatalf("symlink store: %v", e)
	}
	if bStoreBytes(t, outside) != before {
		t.Fatal("store changed")
	}
}
func TestGateBR2CIProjectedTransitions(t *testing.T) {
	dir := t.TempDir()
	s := seedRouteAuthorization(t, dir, "run", "U1", "route")
	rs, _ := s.Load("run")
	a := *rs[0].Approval
	a.CurrentStage = model.StageImplementationAuthorization
	a.ProposedTargetStage = model.StageImplementationReview
	a.ChallengeNonce = "enter-review"
	a.StageTimeDigest = rs[0].Metadata.HeadAnchor + ":2"
	rec := model.Record{Kind: model.KindStageTransition, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Approval: &a, Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: rs[0].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
	if e := s.Append(&rec); e != nil {
		t.Fatal(e)
	}
	setGitHubEnv(t, "repo", "head")
	if e := runCIGithub([]string{"--store", dir, "--run_id", "run", "--verify-only"}); e == nil || !strings.Contains(e.Error(), "no approval") {
		t.Fatalf("must not reuse authorization source-stage approval: %v", e)
	}
	a.CurrentStage = model.StageImplementationReview
	a.ProposedTargetStage = model.StageVerification
	a.ProducedHeadDigest = "head"
	a.DiffDigest = "diff"
	a.ChallengeNonce = "review-packet"
	rec.Metadata.Sequence = 3
	rec.Metadata.PredecessorHash = rec.Metadata.HeadAnchor
	rec.Metadata.HeadAnchor = ""
	rec.Stage = a.CurrentStage
	rec.Control = model.ControlReadyForReview
	rec.Approval = &a
	if e := s.Append(&rec); e != nil {
		t.Fatal(e)
	}
	if e := runCIGithub([]string{"--store", dir, "--run_id", "run", "--verify-only"}); e != nil {
		t.Fatalf("review current-stage packet: %v", e)
	}
	dir = t.TempDir()
	seedApprovedVerification(t, dir, "run", "owner/repo", "head")
	s = store.New(dir)
	rs, _ = s.Load("run")
	rec = rs[len(rs)-1]
	rec.Metadata.Sequence++
	rec.Metadata.PredecessorHash = rec.Metadata.HeadAnchor
	rec.Metadata.HeadAnchor = ""
	rec.Control = model.ControlApproved
	rec.Approval.ChallengeNonce = "leave-verification"
	if e := s.Append(&rec); e != nil {
		t.Fatal(e)
	}
	setGitHubEnv(t, "owner/repo", "head")
	if e := runCIGithub([]string{"--store", dir, "--run_id", "run", "--verify-only"}); e == nil {
		t.Fatal("CI reused previous stage approval in independent validation")
	}
}
func TestGateBR2ResolveStrict(t *testing.T) {
	for _, trail := range []string{" {}", " garbage", " \n\t"} {
		t.Run(fmt.Sprintf("%q", trail), func(t *testing.T) {
			d := t.TempDir()
			seedCriterion(t, d, "run")
			p := filepath.Join(t.TempDir(), "in")
			writeJSON(t, p, waiverInput("run", "n"))
			b, _ := os.ReadFile(p)
			os.WriteFile(p, append(b, []byte(trail)...), 0600)
			before := bStoreBytes(t, d)
			e := runResolve([]string{"--input", p, "--store", d})
			if strings.TrimSpace(trail) == "" {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || before != bStoreBytes(t, d) {
				t.Fatalf("strict refusal or bytes failed: %v", e)
			}
		})
	}
}
func bDecision(t *testing.T, d, worker string) hostDecisionInput {
	t.Helper()
	r := cliRoute(worker, "openai", "openai", worker, "gpt", "approved-route", false)
	return hostDecisionInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "host", Timestamp: "2026-09-16T12:00:00Z"}, Decision: hoststate.OrchestrationDecision{UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "bounded", EstimatedDirectCost: 2, EstimatedDelegatedCost: 1, CostUnit: "relative", ComparisonBasis: "total", Rationale: "delegate", ApprovedPermissionClasses: r.PermissionClasses, WorkContextDigest: "ctx", Route: r}}
}
func bTerminal(t *testing.T, d string, outcome model.TerminalOutcome) {
	t.Helper()
	s := store.New(d)
	rs, e := s.Load("run")
	if e != nil {
		t.Fatal(e)
	}
	a := *rs[0].Approval
	a.CurrentStage = model.StageImplementationAuthorization
	a.ProposedTargetStage = ""
	a.ProposedTargetOutcome = outcome
	a.ChallengeNonce = "terminal"
	a.StageTimeDigest = fmt.Sprintf("%s:%d", rs[len(rs)-1].Metadata.HeadAnchor, len(rs)+1)
	rec := model.Record{Kind: model.KindOutcome, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Outcome: outcome, Approval: &a, Metadata: model.RecordMetadata{RunID: "run", Sequence: int64(len(rs) + 1), PredecessorHash: rs[len(rs)-1].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
	if e := s.Append(&rec); e != nil {
		t.Fatal(e)
	}
}
func TestGateBR2AllTerminalHostPaths(t *testing.T) {
	for _, oc := range []model.TerminalOutcome{model.OutcomeAborted, model.OutcomePartial, model.OutcomeBlocked, model.OutcomeUnknown, model.OutcomeFailed} {
		t.Run(string(oc), func(t *testing.T) {
			d := t.TempDir()
			seedRouteAuthorization(t, d, "run", "U1", "approved-route")
			bTerminal(t, d, oc)
			p := filepath.Join(t.TempDir(), "in")
			de := bDecision(t, d, "impl")
			pr := hostPreflightInput{hostEnvelope: de.hostEnvelope, Preflight: hoststate.RoutePreflight{UnitID: "U1", Role: hoststate.RoleImplementation, Route: de.Decision.Route, Status: hoststate.PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:06:00Z"}}
			di := hostDispatchInput{hostEnvelope: de.hostEnvelope, Dispatch: hoststate.Dispatch{UnitID: "U1"}}
			re := hostReviewInput{hostEnvelope: de.hostEnvelope, Review: hoststate.IndependentReview{UnitID: "U1"}}
			for _, x := range []struct {
				op string
				in any
			}{{"decision", de}, {"preflight", pr}, {"dispatch", di}, {"review", re}} {
				writeJSON(t, p, x.in)
				before := bStoreBytes(t, d)
				e := run([]string{"host", x.op, "--input", p, "--store", d})
				if e == nil || e.Error() != hoststate.ReasonCoreRunTerminal || before != bStoreBytes(t, d) {
					t.Fatalf("%s terminal refusal: %v", x.op, e)
				}
			}
		})
	}
}

// Scheduling-only hook pauses after the command's checks and before append.
func TestGateBR2TerminalInterleaving(t *testing.T) {
	d := t.TempDir()
	seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	p := filepath.Join(t.TempDir(), "in")
	writeJSON(t, p, bDecision(t, d, "impl"))
	before, err := coreHead(d, "run")
	if err != nil {
		t.Fatal(err)
	}
	done := raceTerminal(t, d)
	e := runHostDecision([]string{"--input", p, "--store", d})
	<-done
	assertOrderedBeforeTerminal(t, d, before, e)
}

// raceTerminal arms hostAppendHook to start a concurrent ABORTED core commit
// while the host command holds its locks; the returned channel closes when
// that commit has landed.
func raceTerminal(t *testing.T, d string) chan struct{} {
	t.Helper()
	done := make(chan struct{})
	hostAppendHook = func() {
		hostAppendHook = nil
		go func() { defer close(done); bTerminal(t, d, model.OutcomeAborted) }()
	}
	t.Cleanup(func() { hostAppendHook = nil })
	return done
}

// assertOrderedBeforeTerminal checks the host append and a concurrent terminal
// commit were serialized: the host record binds the pre-terminal core head
// (it committed first), the terminal outcome is in the core, and every later
// host command refuses the terminal run.
func assertOrderedBeforeTerminal(t *testing.T, d, before string, hostErr error) {
	t.Helper()
	if hostErr != nil {
		t.Fatalf("host command: %v", hostErr)
	}
	rs, err := hoststate.NewStore(d).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if got := rs[len(rs)-1].CoreHeadAnchor; got != before {
		t.Fatalf("host record bound core head %s, want pre-terminal head %s", got, before)
	}
	if err := requireCoreRunOpen(d, "run"); err == nil || err.Error() != hoststate.ReasonCoreRunTerminal {
		t.Fatalf("terminal outcome not committed after the host record: %v", err)
	}
	p := filepath.Join(t.TempDir(), "again")
	writeJSON(t, p, bDecision(t, d, "impl-after"))
	if err := runHostDecision([]string{"--input", p, "--store", d}); err == nil || err.Error() != hoststate.ReasonCoreRunTerminal {
		t.Fatalf("host command after terminal: %v, want %s", err, hoststate.ReasonCoreRunTerminal)
	}
}
func TestGateBR2ConcurrentHostDecisions(t *testing.T) {
	d := t.TempDir()
	seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	const n = 16
	var wg sync.WaitGroup
	es := make(chan error, n)
	for k := 0; k < n; k++ {
		p := filepath.Join(t.TempDir(), "in")
		writeJSON(t, p, bDecision(t, d, fmt.Sprintf("impl-%d", k)))
		wg.Add(1)
		go func(p string) { defer wg.Done(); es <- runHostDecision([]string{"--input", p, "--store", d}) }(p)
	}
	wg.Wait()
	close(es)
	ok := 0
	for e := range es {
		if e == nil {
			ok++
		}
	}
	rs, err := hoststate.NewStore(d).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if ok != 1 || len(rs) != 1 {
		t.Fatalf("accepted %d conflicting unsuperseded decisions, chain length %d; want exactly 1", ok, len(rs))
	}
}

// Keep imported evidence helpers explicit for the command-output digest test.
func TestGateBR2EvidenceEnvControl(t *testing.T) {
	r, _, _, _ := ciRepo(t)
	e, err := evidence.Capture(context.Background(), evidence.Request{Executable: "/bin/true", WorkingDir: r, RepositoryRoot: r})
	if err != nil || e.Evidence.ExitCode != 0 {
		t.Fatal(err)
	}
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], 0)
}

func TestGateBR2CIBoundWorktreeGone(t *testing.T) {
	r, b, h, d := ciRepo(t)
	dir := t.TempDir()
	seedApprovedVerificationAt(t, dir, "run", "owner/repo", b, h, d)
	st := store.New(dir)
	rs, _ := st.Load("run")
	rec := rs[len(rs)-1]
	rec.Metadata.Sequence++
	rec.Metadata.PredecessorHash = rec.Metadata.HeadAnchor
	rec.Metadata.HeadAnchor = ""
	rec.Approval.WorktreeDigest = "approved-dirty-content"
	rec.Approval.ChallengeNonce = "dirty-packet"
	if e := st.Append(&rec); e != nil {
		t.Fatal(e)
	}
	setGitHubEnv(t, "owner/repo", h)
	e := runCIGithub([]string{"--store", dir, "--run_id", "run", "--repo-root", r, "--exec", "/bin/true"})
	if e == nil {
		t.Error("CI accepted a clean tree against an explicitly dirty-content-bound approval")
	}
}

func TestGateBR2TerminalDispatchInterleaving(t *testing.T) {
	d := t.TempDir()
	seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	p := filepath.Join(t.TempDir(), "in")
	de := bDecision(t, d, "impl")
	writeJSON(t, p, de)
	if e := runHostDecision([]string{"--input", p, "--store", d}); e != nil {
		t.Fatal(e)
	}
	pr := hostPreflightInput{hostEnvelope: de.hostEnvelope, Preflight: hoststate.RoutePreflight{UnitID: "U1", Role: hoststate.RoleImplementation, Route: de.Decision.Route, Status: hoststate.PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:06:00Z"}}
	writeJSON(t, p, pr)
	if e := runHostPreflight([]string{"--input", p, "--store", d}); e != nil {
		t.Fatal(e)
	}
	rs, _ := hoststate.NewStore(d).Load("run")
	di := hostDispatchInput{hostEnvelope: de.hostEnvelope, Dispatch: hoststate.Dispatch{UnitID: "U1", Role: hoststate.RoleImplementation, Route: de.Decision.Route, PreflightRecordDigest: rs[len(rs)-1].HeadAnchor, BoundedTask: "bounded", WorkContextDigest: "impl-context"}}
	writeJSON(t, p, di)
	before, err := coreHead(d, "run")
	if err != nil {
		t.Fatal(err)
	}
	done := raceTerminal(t, d)
	e := runHostDispatch([]string{"--input", p, "--store", d})
	<-done
	assertOrderedBeforeTerminal(t, d, before, e)
}
func TestGateBR2DependentRouteInvalidation(t *testing.T) {
	d := t.TempDir()
	st := seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	res := engine.EvaluateInvalidation(engine.InvalidationRequest{Trigger: engine.TriggerPlanUnitChanged, AffectedUnitID: "U0", DependentUnitIDs: []string{"U1"}, Approvals: []engine.ApprovalPacket{{ID: "approved-route", UnitID: "U1", Stage: model.StageRouteSelection}}})
	if len(res.InvalidatedApprovalIDs) != 1 || res.InvalidatedApprovalIDs[0] != "approved-route" {
		t.Fatalf("invalidation control %+v", res)
	}
	core, _ := st.Load("run")
	a := *core[0].Approval
	a.UnitID = "U0"
	a.CurrentStage = model.StageImplementationAuthorization
	a.ProposedTargetStage = model.StagePlanning
	a.ChallengeNonce = "upstream-change"
	a.StageTimeDigest = core[0].Metadata.HeadAnchor + ":2"
	rec := model.Record{Kind: model.KindStageTransition, Version: 1, UnitID: "U0", Stage: a.CurrentStage, Control: model.ControlApproved, Approval: &a, Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: core[0].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
	if e := st.Append(&rec); e != nil {
		t.Fatal(e)
	}
	if e := validateRouteApprovalClaim(d, "run", "U1", "approved-route"); e == nil {
		t.Error("dependent U1 route remains valid; stored U0 rewind cannot represent the evaluator's dependent invalidation scope")
	}
}
func TestGateBR2CIMutationHiddenByIndex(t *testing.T) {
	r, b, h, d := ciRepo(t)
	bGit(t, r, "update-index", "--assume-unchanged", "a")
	if e := bCI(t, r, b, h, d, "/usr/bin/truncate", "-s", "0", "a"); e == nil {
		t.Error("mutating command accepted because Git index hid the resulting tracked-file change")
	}
}

// F02: a human-approved rewind that revokes the route, started while a host
// dispatch holds its locks, is serialized after the dispatch record; the
// dispatch binds the pre-rewind core head and every later host command
// refuses the revoked route.
func TestGateBClosureRouteRevocationSerialized(t *testing.T) {
	d := t.TempDir()
	st := seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	p := filepath.Join(t.TempDir(), "in")
	de := bDecision(t, d, "impl")
	writeJSON(t, p, de)
	if e := runHostDecision([]string{"--input", p, "--store", d}); e != nil {
		t.Fatal(e)
	}
	pr := hostPreflightInput{hostEnvelope: de.hostEnvelope, Preflight: hoststate.RoutePreflight{UnitID: "U1", Role: hoststate.RoleImplementation, Route: de.Decision.Route, Status: hoststate.PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:06:00Z"}}
	writeJSON(t, p, pr)
	if e := runHostPreflight([]string{"--input", p, "--store", d}); e != nil {
		t.Fatal(e)
	}
	rs, _ := hoststate.NewStore(d).Load("run")
	di := hostDispatchInput{hostEnvelope: de.hostEnvelope, Dispatch: hoststate.Dispatch{UnitID: "U1", Role: hoststate.RoleImplementation, Route: de.Decision.Route, PreflightRecordDigest: rs[len(rs)-1].HeadAnchor, BoundedTask: "bounded", WorkContextDigest: "impl-context"}}
	writeJSON(t, p, di)
	before, err := coreHead(d, "run")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	hostAppendHook = func() {
		hostAppendHook = nil
		go func() {
			defer close(done)
			core, err := st.Load("run")
			if err != nil {
				t.Error(err)
				return
			}
			a := *core[0].Approval
			a.CurrentStage, a.ProposedTargetStage = model.StageImplementationAuthorization, model.StagePlanning
			a.ChallengeNonce, a.StageTimeDigest = "concurrent-rewind", core[0].Metadata.HeadAnchor+":2"
			rec := model.Record{Kind: model.KindStageTransition, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Approval: &a,
				Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: core[0].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
			if err := st.Append(&rec); err != nil {
				t.Error(err)
			}
		}()
	}
	t.Cleanup(func() { hostAppendHook = nil })
	if e := runHostDispatch([]string{"--input", p, "--store", d}); e != nil {
		t.Fatalf("dispatch ordered before the rewind: %v", e)
	}
	<-done
	hs, _ := hoststate.NewStore(d).Load("run")
	if got := hs[len(hs)-1].CoreHeadAnchor; got != before {
		t.Fatalf("dispatch bound core head %s, want pre-rewind %s", got, before)
	}
	if err := validateRouteApprovalClaim(d, "run", "U1", "approved-route"); err == nil {
		t.Fatal("route still valid after the committed rewind")
	}
	writeJSON(t, p, bDecision(t, d, "impl-after"))
	if e := runHostDecision([]string{"--input", p, "--store", d}); e == nil {
		t.Fatal("host decision accepted on a revoked route")
	}
}

// C03: host review import refuses once the unit's route approval is revoked.
func TestGateBClosureReviewImportRefusesRevokedRoute(t *testing.T) {
	d := t.TempDir()
	st := seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	core, err := st.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	a := *core[0].Approval
	a.CurrentStage, a.ProposedTargetStage = model.StageImplementationAuthorization, model.StagePlanning
	a.ChallengeNonce, a.StageTimeDigest = "rewind-before-review", core[0].Metadata.HeadAnchor+":2"
	rec := model.Record{Kind: model.KindStageTransition, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Approval: &a,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: core[0].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
	if err := st.Append(&rec); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "in")
	writeJSON(t, p, hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "host", Timestamp: "2026-09-16T12:05:00Z"}, Review: hoststate.IndependentReview{UnitID: "U1"}})
	err = runHostReview([]string{"--input", p, "--store", d})
	if err == nil || err.Error() != hoststate.ReasonRouteApprovalMissing {
		t.Fatalf("review import after revocation: %v, want %s", err, hoststate.ReasonRouteApprovalMissing)
	}
}

// R3-01: a host append (for example an independent BLOCK review) started
// after the transition gate passed waits until the core transition is
// recorded; it cannot land between the gate check and the core append.
func TestGateBClosureTransitionHoldsHostLock(t *testing.T) {
	d := t.TempDir()
	seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	p := filepath.Join(t.TempDir(), "in")
	writeJSON(t, p, transition.Input{RunID: "run"})
	var hostDone atomic.Bool
	done := make(chan struct{})
	transitionGateHook = func() {
		transitionGateHook = nil
		go func() {
			defer close(done)
			if err := appendHostRecordForTest(d); err != nil {
				t.Error(err)
			}
			hostDone.Store(true)
		}()
		time.Sleep(200 * time.Millisecond)
		if hostDone.Load() {
			t.Error("host append landed between the transition gate and the core append")
		}
	}
	t.Cleanup(func() { transitionGateHook = nil })
	_ = runTransition([]string{"--input", p, "--store", d})
	<-done
}

func appendHostRecordForTest(d string) error {
	in := bDecisionPlain(d)
	_, err := appendHostRecord(d, in.hostEnvelope, hoststate.KindDecision, &in.Decision, nil, nil, nil, func([]hoststate.Record, []model.Record) error { return nil })
	return err
}

func bDecisionPlain(d string) hostDecisionInput {
	r := cliRoute("late", "openai", "openai", "late", "gpt", "approved-route", false)
	return hostDecisionInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "host", Timestamp: "2026-09-16T12:00:00Z"}, Decision: hoststate.OrchestrationDecision{UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "bounded", EstimatedDirectCost: 2, EstimatedDelegatedCost: 1, CostUnit: "relative", ComparisonBasis: "total", Rationale: "delegate", ApprovedPermissionClasses: r.PermissionClasses, WorkContextDigest: "ctx", Route: r}}
}
