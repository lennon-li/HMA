package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func TestBuildChangedFileManifestDirtyKindsAndNoFIFOBlock(t *testing.T) {
	repo := gitRepo(t)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=HMA Test", "-c", "user.email=hma@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	for name, body := range map[string]string{"modified.txt": "old\n", "deleted.txt": "gone\n"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git("add", "modified.txt", "deleted.txt")
	git("commit", "-qm", "base")
	base := git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "modified.txt"), []byte("new\nline\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "added.txt"), []byte("added\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside-large")
	if err := os.WriteFile(outside, []byte(strings.Repeat("outside\n", 1<<16)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "outside-link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := syscall.Mkfifo(filepath.Join(repo, "pipe"), 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	done := make(chan error, 1)
	go func() { _, err := buildChangedFileManifest(context.Background(), repo, base, true); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("manifest blocked while handling FIFO")
	}
	files, err := buildChangedFileManifest(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]packetFile{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	for _, p := range []string{"added.txt", "modified.txt", "deleted.txt", "outside-link"} {
		if _, ok := byPath[p]; !ok {
			t.Errorf("manifest missing %s: %+v", p, files)
		}
	}
	if f := byPath["added.txt"]; f.Status != "added" || f.Added != 1 || f.SHA256 == "" {
		t.Errorf("added entry = %+v", f)
	}
	if f := byPath["modified.txt"]; f.Status != "modified" || f.Added == 0 || f.SHA256 == "" {
		t.Errorf("modified entry = %+v", f)
	}
	if f := byPath["deleted.txt"]; f.Status != "deleted" || f.SHA256 != "" {
		t.Errorf("deleted entry = %+v", f)
	}
	linkHash := sha256.Sum256([]byte(outside))
	if f := byPath["outside-link"]; f.Added != 1 || f.SHA256 != hex.EncodeToString(linkHash[:]) {
		t.Errorf("symlink entry followed target or has wrong line count: %+v", f)
	}
	// A FIFO is not part of the target: git does not track it and the
	// snapshot digest does not cover it, so the manifest (which describes the
	// same set) does not list it, and building the manifest never opened it.
	if f, ok := byPath["pipe"]; ok {
		t.Errorf("FIFO listed in the manifest: %+v", f)
	}
}

// worktreeBytes is the live reader of the dirty target. A file larger than the
// 16 MiB window is hashed in full while only the window is kept; a symlink is
// its link text and is never followed; a FIFO is never opened.
func TestWorktreeBytesWindowSymlinkAndFIFO(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	body := []byte(strings.Repeat("x\n", (16<<20)/2+4))
	if err := os.WriteFile(regular, body, 0o600); err != nil {
		t.Fatal(err)
	}
	data, mode, exists, digest, truncated, err := worktreeBytes(context.Background(), dir, "regular")
	if err != nil || !exists || mode != "100644" {
		t.Fatalf("regular = %q %v %v", mode, exists, err)
	}
	if len(data) != 16<<20 || !truncated {
		t.Fatalf("window = %d bytes, truncated %v; want the 16 MiB window and truncated", len(data), truncated)
	}
	want := sha256.Sum256(body)
	if digest != hex.EncodeToString(want[:]) {
		t.Fatal("regular file digest did not cover the complete file")
	}
	outside := filepath.Join(t.TempDir(), "big")
	os.WriteFile(outside, []byte(strings.Repeat("y\n", 5000)), 0o600)
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	data, mode, exists, _, _, err = worktreeBytes(context.Background(), dir, "link")
	if err != nil || !exists || mode != "120000" || string(data) != outside {
		t.Fatalf("symlink = %q %q %v %v; want its link text, not the target's bytes", data, mode, exists, err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		d, m, ex, h, _, e := worktreeBytes(context.Background(), dir, "fifo")
		if d != nil || m != "" || !ex || h != "" || e != nil {
			t.Errorf("FIFO result = %q %q %v %q %v", d, m, ex, h, e)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("worktreeBytes opened or blocked on a FIFO")
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPacketEvidenceCurrentFlagsAndOrdering(t *testing.T) {
	binding := hoststate.ReviewPacketBinding{BaseRevision: "base", ProducedHead: "target", DiffDigest: "diff", WorktreeDigest: "work"}
	ref := func(head string) *model.EvidenceRef {
		return &model.EvidenceRef{RepositoryIdentity: "repo", BaseRevision: "base", HeadRevision: head, DiffDigest: "diff", WorktreeDigest: "work", OutputDigest: "sha256:out", OutputRef: "evidence/out"}
	}
	core := []model.Record{
		{UnitID: "U1", Kind: model.KindEvidence, Metadata: model.RecordMetadata{HeadAnchor: "z-old"}, Evidence: ref("old")},
		{UnitID: "U1", Kind: model.KindEvidence, Metadata: model.RecordMetadata{HeadAnchor: "b-current"}, Evidence: ref("target")},
		{Kind: model.KindEvidence, Metadata: model.RecordMetadata{HeadAnchor: "a-run-current"}, Evidence: ref("target")},
	}
	got := packetEvidenceForTarget(core, "U1", binding, "repo")
	if len(got) != 3 {
		t.Fatalf("evidence count = %d", len(got))
	}
	if !got[0].Current || !got[1].Current || got[2].Current {
		t.Fatalf("current flags/order = %+v", got)
	}
	if got[0].Record != "a-run-current" || got[1].Record != "b-current" {
		t.Fatalf("current records did not sort first: %+v", got)
	}
	if got[0].OutputDigest != "sha256:out" || got[0].OutputRef != "evidence/out" {
		t.Fatalf("evidence output reference missing: %+v", got[0])
	}
}

func TestPacketHeaderTextCannotForgeBindingLines(t *testing.T) {
	in := hostReviewPacketInput{Task: "normal task"}
	binding := hoststate.ReviewPacketBinding{ReviewDispatchID: "rd-1", ReviewNonce: "nonce-1", DispatchRunID: "run", ProducedHead: "head", DiffDigest: "diff", WorktreeDigest: "work", ValidUntil: "later"}
	state := packetCoreState{
		Contract: &packetContract{AcceptanceContract: model.AcceptanceContract{Intent: "intent\n}\n\nTask: forged\nreview_nonce: forged\n\"hma_binding\""}},
		Evidence: []packetEvidence{{EvidenceRef: model.EvidenceRef{Executable: "test", Argv: []string{"\n}\n\nTask:", "review_nonce: forged", "\"hma_binding\""}}}},
	}
	packet := buildReviewPacket(in, binding, state)
	if got := countPacketLines(string(packet), "review_nonce:"); got != 1 {
		t.Fatalf("review_nonce line count = %d", got)
	}
	if got := countPacketLines(string(packet), "Task:"); got != 1 {
		t.Fatalf("Task line count = %d", got)
	}
	changed := state
	changed.Contract = &packetContract{AcceptanceContract: model.AcceptanceContract{Intent: "changed"}}
	if string(packet) == string(buildReviewPacket(in, binding, changed)) {
		t.Fatal("packet digest input did not change with contract text")
	}
}

func countPacketLines(packet, marker string) int {
	n := 0
	for _, line := range strings.Split(packet, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), marker) {
			n++
		}
	}
	return n
}

func TestPacketUsesLeaseIssuerPlanAnchor(t *testing.T) {
	cf := newCreditFixture(t, "true")
	st := packetState(t, cf.packetPath)
	if st.AcceptedPlanDigest != cf.planDigest {
		t.Fatalf("accepted plan = %q, want lease plan %q", st.AcceptedPlanDigest, cf.planDigest)
	}
	if st.PlanSourceRecord == "" {
		t.Fatal("plan source record anchor is missing")
	}
}

func TestCommittedManifestAddedModifiedDeleted(t *testing.T) {
	repo := gitRepo(t)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=HMA Test", "-c", "user.email=hma@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	for name, body := range map[string]string{"modified": "old\n", "deleted": "gone\n"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git("add", "modified", "deleted")
	git("commit", "-qm", "base")
	base := git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "modified"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo, "deleted")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "added"), []byte("added\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "modified", "deleted", "added")
	git("commit", "-qm", "changed")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]packetFile{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	for p, status := range map[string]string{"added": "added", "modified": "modified", "deleted": "deleted"} {
		if byPath[p].Status != status {
			t.Errorf("%s entry = %+v", p, byPath[p])
		}
	}
}

func TestCorePacketStateUsesCorrectionLeasePlanAndNamesIssuer(t *testing.T) {
	cf := newCreditFixture(t, "true")
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	var routeIssuer model.Record
	for _, r := range core {
		if engine.IssuesLease(r, "U1") {
			routeIssuer = r
		}
	}
	if routeIssuer.Approval == nil {
		t.Fatal("fixture has no route lease issuer")
	}
	planBytes, err := os.ReadFile(filepath.Join(cf.store, "artifacts", planArtifactName(cf.planDigest)))
	if err != nil {
		t.Fatal(err)
	}
	plan, _, err := model.ParsePlan(planBytes)
	if err != nil {
		t.Fatal(err)
	}
	plan.Units[0].Behavior = "corrected behavior"
	correctedBytes, _ := json.Marshal(plan)
	correctedPlan, correctedDigest, err := model.ParsePlan(correctedBytes)
	if err != nil {
		t.Fatal(err)
	}
	if correctedDigest == routeIssuer.Approval.AcceptedPlanDigest {
		t.Fatal("correction plan digest unexpectedly matches route approval")
	}
	if err := retainPlan(cf.store, correctedPlan, correctedDigest); err != nil {
		t.Fatal(err)
	}
	approval := *routeIssuer.Approval
	approval.CurrentStage = model.StageIndependentValidation
	approval.AcceptedPlanDigest = correctedDigest
	approval.ChallengeNonce = "correction-nonce"
	core = append(core, model.Record{UnitID: "U1", Kind: model.KindStageTransition, Control: model.ControlApproved, Approval: &approval, Metadata: model.RecordMetadata{HeadAnchor: "correction-approval-anchor"}})
	binding := hoststate.ReviewPacketBinding{ProducedHead: cf.head, DiffDigest: cf.diff, WorktreeDigest: ""}
	state, err := corePacketState(cf.store, core, "run", "U1", cf.contractPath, cf.policy, "", binding)
	if err != nil {
		t.Fatal(err)
	}
	if state.AcceptedPlanDigest != correctedDigest {
		t.Fatalf("packet digest %q, want correction lease %q", state.AcceptedPlanDigest, correctedDigest)
	}
	if state.Unit == nil || state.Unit.Behavior != "corrected behavior" {
		t.Fatalf("packet unit did not come from correction approval: %+v", state.Unit)
	}
	if state.PlanSourceRecord != "correction-approval-anchor" {
		t.Fatalf("plan source = %q", state.PlanSourceRecord)
	}
	if state.RouteApproval == "" {
		t.Fatal("route approval digest was lost")
	}
}

func TestPacketSizeLimitsRefuseOversize(t *testing.T) {
	cases := []struct {
		name   string
		packet []byte
		state  packetCoreState
	}{
		{name: "bytes", packet: make([]byte, (2<<20)+1), state: packetCoreState{Contract: &packetContract{}}},
		{name: "evidence", state: packetCoreState{Contract: &packetContract{}, Evidence: make([]packetEvidence, 2001)}},
		{name: "manifest", state: packetCoreState{Contract: &packetContract{}, Manifest: make([]packetFile, 2001)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateReviewPacketSize(tc.packet, tc.state); err == nil || err.Error() != hoststate.ReasonReviewPacketTooLarge {
				t.Fatalf("size result = %v", err)
			}
		})
	}
}

func TestAddContractStatePolicyAndPlanHydration(t *testing.T) {
	cf := newCreditFixture(t, "true")
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	validPlanPath := filepath.Join(t.TempDir(), "plan.json")
	planBytes, err := os.ReadFile(filepath.Join(cf.store, "artifacts", planArtifactName(cf.planDigest)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPlanPath, planBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(cf.store, "artifacts", planArtifactName(cf.planDigest))
	if err := os.Remove(artifact); err != nil {
		t.Fatal(err)
	}
	var hydrated packetCoreState
	plan, digest, err := addContractStateWithPlan(&hydrated, cf.store, core, "U1", cf.contractPath, cf.policy, validPlanPath)
	if err != nil {
		t.Fatalf("matching plan_path was refused: %v", err)
	}
	if plan == nil || digest != cf.planDigest {
		t.Fatalf("validated plan not returned: %v %q", plan, digest)
	}
	// Validation never writes: a packet that is later refused must leave no
	// retained plan. The command retains it only after every check passed.
	if _, err := os.Stat(artifact); err == nil {
		t.Fatal("addContractState retained the plan while only validating")
	}
	if err := retainPlan(cf.store, *plan, digest); err != nil {
		t.Fatal(err)
	}
	var second packetCoreState
	if err := addContractState(&second, cf.store, core, "U1", cf.contractPath, cf.policy, ""); err != nil {
		t.Fatalf("retained plan unavailable to second packet: %v", err)
	}
	var missing packetCoreState
	if err := addContractState(&missing, cf.store, core, "U1", cf.contractPath, "", ""); err == nil || err.Error() != "HOST_REVIEW_PACKET_INPUT_MISSING" {
		t.Fatalf("missing policy reason = %v", err)
	}
	badPolicy := filepath.Join(t.TempDir(), "wrong-policy.json")
	if err := os.WriteFile(badPolicy, []byte(`{"version":"invalid"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var wrong packetCoreState
	if err := addContractState(&wrong, cf.store, core, "U1", cf.contractPath, badPolicy, ""); err == nil || err.Error() != hoststate.ReasonPolicyNotCurrent {
		t.Fatalf("wrong policy reason = %v", err)
	}
	wrongPlan := filepath.Join(t.TempDir(), "wrong-plan.json")
	if err := os.WriteFile(wrongPlan, []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var corrupt packetCoreState
	if err := addContractState(&corrupt, cf.store, core, "U1", cf.contractPath, cf.policy, wrongPlan); err == nil || !strings.Contains(err.Error(), hoststate.ReasonReviewPacketPlanMissing) {
		t.Fatalf("corrupt plan reason = %v", err)
	}
	var absent packetCoreState
	if err := addContractState(&absent, cf.store, core, "U1", cf.contractPath, cf.policy, filepath.Join(t.TempDir(), "absent.json")); err == nil || !strings.Contains(err.Error(), hoststate.ReasonReviewPacketPlanMissing) {
		t.Fatalf("absent plan reason = %v", err)
	}
}

func TestContractRefusedWhenRunHasNoBoundContract(t *testing.T) {
	bound := newCreditFixture(t, "true")
	plain := newReviewFixture(t, hoststate.IssuePerChange, false)
	core, err := store.New(plain.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	var state packetCoreState
	if err := addContractState(&state, plain.store, core, "U1", bound.contractPath, bound.policy, ""); err == nil || err.Error() != string(engine.ReasonContractMismatch) {
		t.Fatalf("contract-on-unbound-run reason = %v", err)
	}
}

func TestReviewPacketRefusesManifestTargetRaceBeforeWrites(t *testing.T) {
	cf := newCreditFixture(t, "true")
	pf := cf.step("preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:30:00Z"), Preflight: hoststate.RoutePreflight{UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review-race", ExpiresAt: "2026-09-16T13:30:00Z"}})
	out := filepath.Join(t.TempDir(), "race-packet.md")
	input := filepath.Join(t.TempDir(), "input.json")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:31:00Z"), Dispatch: hoststate.Dispatch{UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, PreflightRecordDigest: pf, BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: cf.implementation, Risk: model.RouteRiskHigh}, DispatchRunID: "race-review-run", IssuingReason: cf.reason, RepositoryRoot: cf.root, BaseRevision: cf.base, DirtyWorktreeApproved: true, Task: "Review change.", ContractPath: cf.contractPath, PolicyPath: cf.policy, LedgerDir: cf.ledger})
	before, err := hoststate.NewStore(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	reviewPacketBeforeManifestHook = func() {
		if err := os.WriteFile(filepath.Join(cf.root, "a"), []byte("manifest race\n"), 0o600); err != nil {
			t.Errorf("mutate target: %v", err)
		}
	}
	defer func() { reviewPacketBeforeManifestHook = nil }()
	err = run([]string{"host", "review-packet", "--input", input, "--store", cf.store, "--out", out})
	if err == nil || err.Error() != hoststate.ReasonReviewTargetChanged {
		t.Fatalf("race refusal = %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("packet exists after target race: %v", err)
	}
	after, err := hoststate.NewStore(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("host records changed on refusal: before=%d after=%d", len(before), len(after))
	}
}

func TestPacketCarriesActiveWaiverGrantTerms(t *testing.T) {
	cf := newCreditFixture(t, "true")
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	scope := model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"}
	core = append(core, model.Record{Kind: model.KindWaiverOperation, UnitID: "U1", Metadata: model.RecordMetadata{HeadAnchor: "waiver-grant-anchor", Timestamp: "2026-09-16T12:40:00Z"}, WaiverOperation: &model.WaiverOperation{UnitID: "U1", Operation: model.WaiverOperationGrant, Scope: scope, Justification: "approved waiver", Approver: "human-reviewer", Timestamp: "2026-09-16T12:40:00Z", ChallengeNonce: "waiver-nonce"}})
	state, err := corePacketState(cf.store, core, "run", "U1", cf.contractPath, cf.policy, "", hoststate.ReviewPacketBinding{ProducedHead: cf.head, DiffDigest: cf.diff})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ActiveWaivers) != 1 || state.ActiveWaivers[0] != scope {
		t.Fatalf("active waiver selectors = %+v", state.ActiveWaivers)
	}
	if len(state.WaiverTerms) != 1 {
		t.Fatalf("waiver terms = %+v", state.WaiverTerms)
	}
	w := state.WaiverTerms[0]
	if w.Operation == nil || w.Operation.Approver != "human-reviewer" || w.Operation.Timestamp != "2026-09-16T12:40:00Z" || w.Record != "waiver-grant-anchor" || w.Operation.Operation != model.WaiverOperationGrant || w.Operation.ChallengeNonce != "waiver-nonce" {
		t.Fatalf("grant terms = %+v", w)
	}
}

func TestReviewPacketPolicyRefusalWritesNothing(t *testing.T) {
	for _, tc := range []struct{ name, policy, want string }{{"missing", "", "HOST_REVIEW_PACKET_INPUT_MISSING"}, {"wrong", "invalid", hoststate.ReasonPolicyNotCurrent}} {
		t.Run(tc.name, func(t *testing.T) {
			cf := newCreditFixture(t, "true")
			pf := cf.step("preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:30:00Z"), Preflight: hoststate.RoutePreflight{UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-policy-test", ExpiresAt: "2026-09-16T13:30:00Z"}})
			policyPath := tc.policy
			if policyPath == "invalid" {
				policyPath = filepath.Join(t.TempDir(), "policy.json")
				if err := os.WriteFile(policyPath, []byte(`{"version":"invalid"}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			out, input := filepath.Join(t.TempDir(), "packet.md"), filepath.Join(t.TempDir(), "input.json")
			writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:31:00Z"), Dispatch: hoststate.Dispatch{UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, PreflightRecordDigest: pf, BoundedTask: "review", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: cf.implementation, Risk: model.RouteRiskHigh}, DispatchRunID: "policy-test", IssuingReason: cf.reason, RepositoryRoot: cf.root, BaseRevision: cf.base, Task: "Review.", ContractPath: cf.contractPath, PolicyPath: policyPath, LedgerDir: cf.ledger})
			before, err := hoststate.NewStore(cf.store).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			err = run([]string{"host", "review-packet", "--input", input, "--store", cf.store, "--out", out})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("policy refusal = %v, want %s", err, tc.want)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("packet exists after policy refusal: %v", err)
			}
			after, err := hoststate.NewStore(cf.store).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			if len(after) != len(before) {
				t.Fatalf("host records changed on refusal: before=%d after=%d", len(before), len(after))
			}
		})
	}
}

func TestContractBoundPacketNeedsPlanOrPlanPath(t *testing.T) {
	cf := newCreditFixture(t, "true")
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(cf.store, "artifacts", planArtifactName(cf.planDigest))
	if err := os.Remove(artifact); err != nil {
		t.Fatal(err)
	}
	var state packetCoreState
	err = addContractState(&state, cf.store, core, "U1", cf.contractPath, cf.policy, "")
	if err == nil || !strings.Contains(err.Error(), hoststate.ReasonReviewPacketPlanMissing) || !strings.Contains(err.Error(), "supply plan_path") {
		t.Fatalf("missing plan error = %v", err)
	}
}

func TestPlanPathWithDifferentLeaseDigestIsRefused(t *testing.T) {
	cf := newCreditFixture(t, "true")
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(cf.store, "artifacts", planArtifactName(cf.planDigest)))
	if err != nil {
		t.Fatal(err)
	}
	plan, _, err := model.ParsePlan(b)
	if err != nil {
		t.Fatal(err)
	}
	plan.Units[0].Behavior += " changed"
	mutated, _ := json.Marshal(plan)
	path := filepath.Join(t.TempDir(), "different-plan.json")
	if err := os.WriteFile(path, mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	var state packetCoreState
	err = addContractState(&state, cf.store, core, "U1", cf.contractPath, cf.policy, path)
	if err == nil || !strings.Contains(err.Error(), hoststate.ReasonReviewPacketPlanMissing) {
		t.Fatalf("wrong plan digest reason = %v", err)
	}
}

func TestNormalizedPolicyDigestMustMatchBoundContract(t *testing.T) {
	cf := newCreditFixture(t, "true")
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	var policy model.RoutePolicy
	b, err := os.ReadFile(cf.policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &policy); err != nil {
		t.Fatal(err)
	}
	policy.PermissionClasses = []string{model.PermissionReadOnly}
	changed, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "policy-changed.json")
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	var state packetCoreState
	if err := addContractState(&state, cf.store, core, "U1", cf.contractPath, path, ""); err == nil || err.Error() != hoststate.ReasonPolicyNotCurrent {
		t.Fatalf("valid but stale policy reason = %v", err)
	}
}

// The size cap runs through the whole command: an oversize packet and a
// packet with too many entries are refused with HOST_REVIEW_PACKET_TOO_LARGE
// before any packet file or dispatch record exists; the same input with a
// normal task succeeds (control).
func TestReviewPacketCommandRefusesOversizeBeforeWrites(t *testing.T) {
	cf := newCreditFixture(t, "true")
	history := func() int {
		h, _ := hoststate.NewStore(cf.store).Load("run")
		return len(h)
	}
	packet := func(task, out string) error {
		rpf := cf.step("preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:40:00Z"), Preflight: hoststate.RoutePreflight{
			UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T13:40:00Z",
		}})
		in := filepath.Join(t.TempDir(), "packet.json")
		writeJSON(t, in, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:41:00Z"), Dispatch: hoststate.Dispatch{
			UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, PreflightRecordDigest: rpf,
			BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: cf.implementation,
			ArtifactDigest: hoststate.ReviewArtifactDigest(cf.head, cf.diff, ""), Risk: model.RouteRiskHigh,
		}, DispatchRunID: "20261016T000000Z-ming-review-9", IssuingReason: cf.reason, RepositoryRoot: cf.root, BaseRevision: cf.base,
			Task: task, ContractPath: cf.contractPath, PolicyPath: cf.policy, LedgerDir: cf.ledger})
		return run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	}
	out := filepath.Join(t.TempDir(), "big.md")
	before := history()
	preflightOnly := before + 1
	err := packet(strings.Repeat("x", 3<<20), out)
	if err == nil || err.Error() != hoststate.ReasonReviewPacketTooLarge {
		t.Fatalf("oversize packet: %v", err)
	}
	if _, serr := os.Stat(out); serr == nil {
		t.Fatal("a packet file exists for a refused packet")
	}
	if history() != preflightOnly {
		t.Fatalf("a dispatch record exists for a refused packet: %d records, want %d (preflight only)", history(), preflightOnly)
	}
	if err := packet("Review the change.", filepath.Join(t.TempDir(), "ok.md")); err != nil {
		t.Fatalf("control packet refused: %v", err)
	}
}
