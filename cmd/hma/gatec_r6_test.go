package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// Liz final3 C-F17: a pass report whose required strings hold only
// U+001C..U+001F fails the Python validator (str.strip removes them), so it
// must not approve or clear a blocker.
func TestGateCR6InformationSeparatorReportNotApproved(t *testing.T) {
	for _, sep := range []string{"\x1c", "\x1d", "\x1e", "\x1f"} {
		f := newReviewFixture(t, hoststate.IssuePerChange, false)
		f.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "EXISTING-BLOCK", Disposition: model.FindingBlock}}})
		f.packet("2026-09-16T12:05:10Z", "2026-09-16T12:05:20Z", "2026-09-16T12:10:00Z", "r6-sep")
		rp := f.report("pass", "isolated", func(r map[string]any) {
			r["audit_id"], r["target"] = sep, sep
			r["requirements"] = []any{map[string]any{"id": sep, "status": "pass", "evidence_refs": []any{sep}}}
		})
		f.receipt(rp, nil)
		if err := f.importReport(rp); err != nil {
			t.Fatalf("%q: import: %v", sep, err)
		}
		h, _ := hoststate.NewStore(f.store).Load("run")
		if v := h[len(h)-1].Review.Verdict; v != hoststate.ReviewEscalate {
			t.Fatalf("%q: verdict = %s, want ESCALATE", sep, v)
		}
		_ = f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z")
		if f.coreFindings()["EXISTING-BLOCK"] != model.FindingBlock || f.gate() == nil {
			t.Fatalf("%q: invalid report cleared a blocker or opened the gate", sep)
		}
	}
}

func outputArtifact(stdout, stderr string) (string, []byte) {
	b, _ := json.Marshal(struct {
		Schema string `json:"schema"`
		Stdout []byte `json:"stdout"`
		Stderr []byte `json:"stderr"`
	}{"hma.evidence-output/v1", []byte(stdout), []byte(stderr)})
	sum := sha256.Sum256(b)
	return "sha256-" + hex.EncodeToString(sum[:]) + ".json", b
}

// Liz final3 C-F18: an existing output artifact is used only when it is a
// regular file with exactly the expected bytes. Crash residue is refused
// before either append and is left in place.
func TestGateCR6CorruptOutputArtifactRefused(t *testing.T) {
	name, want := outputArtifact("out\n", "err\n")
	for _, tc := range []struct {
		name  string
		plant func(dir, path string)
	}{
		{"empty", func(_, p string) { os.WriteFile(p, nil, 0o600) }},
		{"partial", func(_, p string) { os.WriteFile(p, want[:len(want)/2], 0o600) }},
		{"symlink-to-correct-bytes", func(dir, p string) {
			other := filepath.Join(dir, "elsewhere.json")
			os.WriteFile(other, want, 0o600)
			os.Symlink(other, p)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			root, base := seedLoopRun(t, d)
			dir := filepath.Join(d, "artifacts")
			os.MkdirAll(dir, 0o700)
			path := filepath.Join(dir, name)
			tc.plant(dir, path)
			planted, _ := os.Lstat(path)
			coreBefore, _ := store.New(d).Load("run")
			hostBefore, _ := hoststate.NewStore(d).Load("run")
			st := writeLoopState(t, d, "run", "U1", []string{"python3", "-c", "import sys; print('out'); print('err', file=sys.stderr)"}, "green", 0)
			if err := runLoopEvidence(t, d, root, base, st); err == nil || err.Error() != hoststate.ReasonEvidenceArtifactCorrupt {
				t.Fatalf("err = %v, want %s", err, hoststate.ReasonEvidenceArtifactCorrupt)
			}
			coreAfter, _ := store.New(d).Load("run")
			hostAfter, _ := hoststate.NewStore(d).Load("run")
			if len(coreAfter) != len(coreBefore) || len(hostAfter) != len(hostBefore) {
				t.Fatal("refused capture appended")
			}
			if fi, err := os.Lstat(path); err != nil || fi.Mode() != planted.Mode() || fi.Size() != planted.Size() {
				t.Fatal("corrupt artifact not preserved for diagnosis")
			}
		})
	}
}

// A complete existing artifact is reused, and no temporary file is left.
func TestGateCR6IntactOutputArtifactReused(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	name, want := outputArtifact("out\n", "err\n")
	dir := filepath.Join(d, "artifacts")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, name), want, 0o600)
	st := writeLoopState(t, d, "run", "U1", []string{"python3", "-c", "import sys; print('out'); print('err', file=sys.stderr)"}, "green", 0)
	if err := runLoopEvidence(t, d, root, base, st); err != nil {
		t.Fatal(err)
	}
	core, _ := store.New(d).Load("run")
	if ref := core[len(core)-1].Evidence.OutputRef; ref != "artifacts/"+name {
		t.Fatalf("ref = %s", ref)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("artifacts dir has %d entries, want 1", len(entries))
	}
}

// Liz final4 C-F19: loop-evidence refuses a journal whose verify argv is a
// shell or an operation verify-loop forbids, before running it and before
// either append.
func TestGateCR7LoopEvidenceRefusesUnsafeArgv(t *testing.T) {
	for _, tc := range []struct {
		name string
		argv func(marker string) []string
	}{
		{"sh-c", func(m string) []string { return []string{"/bin/sh", "-c", "touch " + m} }},
		{"bash-script", func(m string) []string { return []string{"bash", m + ".sh"} }},
		{"env-sh", func(m string) []string { return []string{"env", "sh", "-c", "touch " + m} }},
		{"git-push", func(string) []string { return []string{"git", "push"} }},
		{"curl", func(string) []string { return []string{"curl", "http://example.invalid"} }},
		{"rm-r", func(m string) []string { return []string{"rm", "-r", m} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			root, base := seedLoopRun(t, d)
			marker := filepath.Join(t.TempDir(), "ran")
			os.WriteFile(marker+".sh", []byte("touch "+marker+"\n"), 0o700)
			coreBefore, _ := store.New(d).Load("run")
			hostBefore, _ := hoststate.NewStore(d).Load("run")
			st := writeLoopState(t, d, "run", "U1", tc.argv(marker), "green", 0)
			if err := runLoopEvidence(t, d, root, base, st); err == nil || err.Error() != hoststate.ReasonLoopEvidenceInvalid {
				t.Fatalf("err = %v, want %s", err, hoststate.ReasonLoopEvidenceInvalid)
			}
			if _, err := os.Stat(marker); err == nil {
				t.Fatal("shell ran")
			}
			coreAfter, _ := store.New(d).Load("run")
			hostAfter, _ := hoststate.NewStore(d).Load("run")
			if len(coreAfter) != len(coreBefore) || len(hostAfter) != len(hostBefore) {
				t.Fatal("refused capture appended")
			}
		})
	}
}

// Liz final5: a report with an invalid UTF-8 byte fails the Python validator,
// so it neither imports as an approval nor clears a blocker.
func TestGateCR8InvalidUTF8ReportNotApproved(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	f.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "EXISTING-BLOCK", Disposition: model.FindingBlock}}})
	f.packet("2026-09-16T12:05:10Z", "2026-09-16T12:05:20Z", "2026-09-16T12:10:00Z", "r8-utf8")
	rp := f.report("pass", "isolated", nil)
	appendKey(t, rp, "\"limitations_note\":\"\xff\"")
	f.receipt(rp, nil)
	_ = f.importReport(rp)
	if h, _ := hoststate.NewStore(f.store).Load("run"); h[len(h)-1].Kind == hoststate.KindReview && h[len(h)-1].Review.Verdict == hoststate.ReviewApprove {
		t.Fatal("invalid UTF-8 report approved")
	}
	_ = f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z")
	if f.coreFindings()["EXISTING-BLOCK"] != model.FindingBlock || f.gate() == nil {
		t.Fatal("invalid UTF-8 report cleared a blocker or opened the gate")
	}
}

// Liz final6 C-F22: shell scripts, symlinked shells, and command wrappers
// are refused by loop-evidence before running and before either append.
func TestGateCR9ShellEntryPointsRefused(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	env, err := exec.LookPath("env")
	if err != nil {
		t.Skip("no env")
	}
	for _, tc := range []struct {
		name string
		argv func(dir, marker string) []string
	}{
		{"shebang-script", func(dir, m string) []string {
			p := filepath.Join(dir, "verify-check")
			os.WriteFile(p, []byte("#!/bin/sh\ntouch "+m+"\n"), 0o700)
			return []string{p}
		}},
		{"env-shebang-script", func(dir, m string) []string {
			p := filepath.Join(dir, "verify-check")
			os.WriteFile(p, []byte("#!/usr/bin/env sh\ntouch "+m+"\n"), 0o700)
			return []string{p}
		}},
		{"env-with-symlinked-shell", func(dir, m string) []string {
			l := filepath.Join(dir, "check-runner")
			os.Symlink(sh, l)
			p := filepath.Join(dir, "verify-check")
			os.WriteFile(p, []byte("touch "+m+"\n"), 0o600)
			return []string{env, l, p}
		}},
		{"symlinked-env-alias", func(dir, m string) []string {
			l := filepath.Join(dir, "launch-check")
			os.Symlink(env, l)
			p := filepath.Join(dir, "verify-check")
			os.WriteFile(p, []byte("touch "+m+"\n"), 0o600)
			return []string{l, "sh", p}
		}},
		{"script-interpreted-by-symlinked-shell", func(dir, m string) []string {
			l := filepath.Join(dir, "interp")
			os.Symlink(sh, l)
			p := filepath.Join(dir, "verify-check")
			os.WriteFile(p, []byte("#!"+l+"\ntouch "+m+"\n"), 0o700)
			return []string{p}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			root, base := seedLoopRun(t, d)
			dir := t.TempDir()
			marker := filepath.Join(t.TempDir(), "ran")
			coreBefore, _ := store.New(d).Load("run")
			hostBefore, _ := hoststate.NewStore(d).Load("run")
			st := writeLoopState(t, d, "run", "U1", tc.argv(dir, marker), "green", 0)
			if err := runLoopEvidence(t, d, root, base, st); err == nil {
				t.Fatal("shell entry point accepted")
			}
			if _, err := os.Stat(marker); err == nil {
				t.Fatal("shell ran")
			}
			coreAfter, _ := store.New(d).Load("run")
			hostAfter, _ := hoststate.NewStore(d).Load("run")
			if len(coreAfter) != len(coreBefore) || len(hostAfter) != len(hostBefore) {
				t.Fatal("refused capture appended")
			}
		})
	}
}

// Liz final6 C-F23: the core run lock is held while the loop check runs, so
// no core record (a direct terminal append included) can land between the
// terminal check and the check run. The check itself probes the lock.
func TestGateCR9CoreLockHeldDuringLoopCheck(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	lock := filepath.Join(d, "run.lock")
	probe := `import fcntl, sys
f = open(sys.argv[1], "a")
try:
    fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
    print("free")
except BlockingIOError:
    print("held")`
	if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"python3", "-c", probe, lock}, "green", 0)); err != nil {
		t.Fatal(err)
	}
	core, _ := store.New(d).Load("run")
	b, err := os.ReadFile(filepath.Join(d, core[len(core)-1].Evidence.OutputRef))
	if err != nil {
		t.Fatal(err)
	}
	var art struct{ Stdout []byte }
	json.Unmarshal(b, &art)
	if string(art.Stdout) != "held\n" {
		t.Fatalf("core lock during check: %q, want held", art.Stdout)
	}
}

// packetAt runs review-packet for f's unit against root and base and returns
// its error.
func (f *reviewFixture) packetAt(root, base, packetTS, runID string) error {
	f.t.Helper()
	rpf := f.step("preflight", hostPreflightInput{hostEnvelope: fxEnv(packetTS), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: f.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:30:00Z",
	}})
	input := filepath.Join(f.t.TempDir(), "packet.json")
	writeJSON(f.t, input, hostReviewPacketInput{hostEnvelope: fxEnv(packetTS), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: f.reviewer, PreflightRecordDigest: rpf,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: f.implementation,
		Risk: model.RouteRiskHigh,
	}, DispatchRunID: runID, IssuingReason: hoststate.IssuePerChange, RepositoryRoot: root, BaseRevision: base, Task: "Review the change.", LedgerDir: f.ledger})
	return run([]string{"host", "review-packet", "--input", input, "--store", f.store, "--out", filepath.Join(f.t.TempDir(), "packet.md")})
}

// Liz final7 C-F24: a review packet must review the base the unit's route
// approval binds, in the run's one repository root.
func TestGateCR10PacketBoundToApprovedRepository(t *testing.T) {
	t.Run("other-base-same-root", func(t *testing.T) {
		f := newReviewFixture(t, hoststate.IssuePerChange, false)
		if err := f.packetAt(f.root, f.head, "2026-09-16T12:05:20Z", "r10-head"); err == nil || err.Error() != hoststate.ReasonReviewRepositoryMismatch {
			t.Fatalf("err = %v, want %s", err, hoststate.ReasonReviewRepositoryMismatch)
		}
		// Only the review preflight was recorded; no packet dispatch.
		if h, _ := hoststate.NewStore(f.store).Load("run"); h[len(h)-1].Kind != hoststate.KindPreflight {
			t.Fatalf("refused packet appended a %s record", h[len(h)-1].Kind)
		}
	})
	t.Run("other-repository", func(t *testing.T) {
		f := newReviewFixture(t, hoststate.IssuePerChange, false)
		otherRoot, otherBase, _, _ := ciRepo(t)
		for _, base := range []string{otherBase, f.base} {
			if err := f.packetAt(otherRoot, base, "2026-09-16T12:05:20Z", "r10-other"); err == nil || err.Error() != hoststate.ReasonReviewRepositoryMismatch {
				t.Fatalf("base %s: err = %v, want %s", base, err, hoststate.ReasonReviewRepositoryMismatch)
			}
		}
	})
	t.Run("clone-with-same-base", func(t *testing.T) {
		f := newReviewFixture(t, hoststate.IssuePerChange, false)
		clone := filepath.Join(t.TempDir(), "clone")
		if out, err := exec.Command("/usr/bin/git", "clone", "-q", f.root, clone).CombinedOutput(); err != nil {
			t.Fatalf("clone: %v %s", err, out)
		}
		if err := f.packetAt(clone, f.base, "2026-09-16T12:05:20Z", "r10-clone"); err == nil || err.Error() != hoststate.ReasonReviewRepositoryMismatch {
			t.Fatalf("err = %v, want %s", err, hoststate.ReasonReviewRepositoryMismatch)
		}
	})
	t.Run("approved-base-and-root", func(t *testing.T) {
		f := newReviewFixture(t, hoststate.IssuePerChange, false)
		if err := f.packetAt(f.root, f.base, "2026-09-16T12:05:20Z", "r10-ok"); err != nil {
			t.Fatal(err)
		}
	})
}

// Jax final8j J8-01: without a repository binding no packet is generated,
// and a binding is written once and never taken from a packet.
func TestGateCR11RepositoryBindingRequired(t *testing.T) {
	d := t.TempDir()
	root, base, _, _ := ciRepo(t)
	seedRouteAuthorizationAt(t, d, "run", "U1", "approved-route", base)
	core, _ := store.New(d).Load("run")
	if err := requireApprovedRepository(nil, core, "run", "U1", &hoststate.ReviewPacketBinding{ProjectRoot: root, BaseRevision: base}); err == nil || err.Error() != hoststate.ReasonRepositoryBindingMissing {
		t.Fatalf("unbound run: err = %v, want %s", err, hoststate.ReasonRepositoryBindingMissing)
	}
	bind := func(root, identity string) error {
		input := filepath.Join(t.TempDir(), "bind.json")
		writeJSON(t, input, hostBindRepositoryInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T11:30:00Z"},
			RepositoryRoot: root, RepositoryIdentity: identity})
		return run([]string{"host", "bind-repository", "--input", input, "--store", d})
	}
	if err := bind(root, "other/repo"); err == nil || err.Error() != hoststate.ReasonRepositoryMismatch {
		t.Fatalf("identity not in approvals: err = %v", err)
	}
	if err := bind(filepath.Join(root, "sub-not-top"), "repo"); err == nil {
		t.Fatal("non-top-level root bound")
	}
	if err := bind(root, "repo"); err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(t.TempDir(), "clone")
	if out, err := exec.Command("/usr/bin/git", "clone", "-q", root, clone).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v %s", err, out)
	}
	if err := bind(clone, "repo"); err == nil || err.Error() != hoststate.ReasonRepositoryBindingExists {
		t.Fatalf("second binding: err = %v", err)
	}
}

// J8-01 at import: a clone holding the approved base, even as the run's
// first packet, cannot review for the bound repository.
func TestGateCR11ClonePacketRefused(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	clone := filepath.Join(t.TempDir(), "clone")
	if out, err := exec.Command("/usr/bin/git", "clone", "-q", f.root, clone).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v %s", err, out)
	}
	if err := f.packetAt(clone, f.base, "2026-09-16T12:05:20Z", "r11-clone"); err == nil || err.Error() != hoststate.ReasonReviewRepositoryMismatch {
		t.Fatalf("err = %v", err)
	}
}

// Jax final8j J8-02: an imported finding id containing "/" is refused, so
// unit "U1" with "sub/B" cannot collide with unit "U1/sub" with "B".
func TestGateCR11SlashFindingIDRefused(t *testing.T) {
	if _, err := hoststate.MapAuditFindings([]byte(`[{"id":"sub/B"}]`)); err == nil {
		t.Fatal("slash id accepted")
	}
	if _, err := hoststate.MapAuditFindings([]byte(`[{"id":"B"}]`)); err != nil {
		t.Fatal(err)
	}
}

// Jax final8j J8-03: loop evidence runs only in the bound repository at the
// approved base, and the journal must name that root.
func TestGateCR11LoopEvidenceBoundToRepository(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	otherRoot, otherBase, _, _ := ciRepo(t)
	for _, tc := range []struct {
		name             string
		root, base, repo string
	}{
		{"other-root-and-base", otherRoot, otherBase, otherRoot},
		{"other-root-approved-base", otherRoot, base, otherRoot},
		{"journal-names-other-root", root, base, otherRoot},
	} {
		st := writeLoopState(t, d, "run", "U1", []string{"true"}, "green", 0, func(_, st map[string]any) { st["repo"] = map[string]any{"root": tc.repo} })
		if err := runLoopEvidence(t, d, tc.root, tc.base, st); err == nil || err.Error() != hoststate.ReasonRepositoryMismatch {
			t.Fatalf("%s: err = %v, want %s", tc.name, err, hoststate.ReasonRepositoryMismatch)
		}
	}
	if h, _ := hoststate.NewStore(d).Load("run"); len(h) != 1 {
		t.Fatalf("refused imports wrote %d host records", len(h)-1)
	}
	if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"true"}, "green", 0)); err != nil {
		t.Fatal(err)
	}
}

// Jax final9j J9-01: forbidden operations are judged on the canonical
// executable, so a symlink alias of rm, curl, or git is refused like the
// direct command, before running and before either append.
func TestGateCR12AliasedForbiddenOperationRefused(t *testing.T) {
	for _, tc := range []struct {
		tool string
		args func(victim string) []string
	}{
		{"rm", func(v string) []string { return []string{"-r", v} }},
		{"curl", func(string) []string { return []string{"http://example.invalid"} }},
		{"git", func(string) []string { return []string{"push"} }},
	} {
		target, err := exec.LookPath(tc.tool)
		if err != nil {
			continue
		}
		for _, form := range []string{"direct", "absolute", "alias"} {
			t.Run(tc.tool+"-"+form, func(t *testing.T) {
				d := t.TempDir()
				root, base := seedLoopRun(t, d)
				victim := filepath.Join(t.TempDir(), "victim")
				os.MkdirAll(filepath.Join(victim, "x"), 0o700)
				exe := tc.tool
				switch form {
				case "absolute":
					exe = target
				case "alias":
					exe = filepath.Join(t.TempDir(), "check-tool")
					if err := os.Symlink(target, exe); err != nil {
						t.Fatal(err)
					}
				}
				argv := append([]string{exe}, tc.args(victim)...)
				if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", argv, "green", 0)); err == nil || err.Error() != hoststate.ReasonLoopEvidenceInvalid {
					t.Fatalf("err = %v, want %s", err, hoststate.ReasonLoopEvidenceInvalid)
				}
				if _, err := os.Stat(victim); err != nil {
					t.Fatal("aliased command ran")
				}
				if h, _ := hoststate.NewStore(d).Load("run"); len(h) != 1 {
					t.Fatalf("refused import wrote %d host records", len(h)-1)
				}
			})
		}
	}
}

// Jax final10j J10-01/J10-02: bind-repository names a next action, and
// loop-evidence reports the review enforcement mode.
func TestGateCR13ResultContract(t *testing.T) {
	d := t.TempDir()
	root, base, _, _ := ciRepo(t)
	seedRouteAuthorizationAt(t, d, "run", "U1", "approved-route", base)
	input := filepath.Join(t.TempDir(), "bind.json")
	writeJSON(t, input, hostBindRepositoryInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T11:30:00Z"},
		RepositoryRoot: root, RepositoryIdentity: "repo"})
	out, err := captureStdout(t, func() error { return run([]string{"host", "bind-repository", "--input", input, "--store", d}) })
	if err != nil {
		t.Fatal(err)
	}
	var res map[string]any
	if json.Unmarshal(out, &res) != nil || res["next_action"] == "" || res["next_action"] == nil {
		t.Fatalf("bind-repository result %s lacks next_action", out)
	}
	out, err = captureStdout(t, func() error {
		return runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"true"}, "green", 0))
	})
	if err != nil {
		t.Fatal(err)
	}
	res = nil
	if json.Unmarshal(out, &res) != nil || (res["review_enforcement"] != "legacy_manual" && res["review_enforcement"] != "packet_import") {
		t.Fatalf("loop-evidence result %s lacks review_enforcement", out)
	}
}

// Liz final11 J11-01/J11-02: findings import chooses its next action from
// the review verdict first and reports the review enforcement mode.
func TestGateCR14FindingsImportResult(t *testing.T) {
	for _, tc := range []struct {
		verdict, want string
		edit          func(map[string]any)
	}{
		{"pass", "CONTINUE_HMA_TRANSITION", nil},
		{"fix-required", "RECORD_FINDINGS_AND_REMEDIATE", func(r map[string]any) {
			r["requirements"] = []any{map[string]any{"id": "R1", "status": "fail", "evidence_refs": []any{"e1"}}}
		}},
		{"blocked", "ESCALATE_TO_HUMAN", nil},
	} {
		t.Run(tc.verdict, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			rp := f.report(tc.verdict, "isolated", tc.edit)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err != nil {
				t.Fatal(err)
			}
			out, err := captureStdout(t, func() error { return f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z") })
			if err != nil {
				t.Fatal(err)
			}
			var res map[string]any
			if err := json.Unmarshal(out, &res); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			if res["next_action"] != tc.want || res["review_enforcement"] != "packet_import" {
				t.Fatalf("result %s, want next_action %s and review_enforcement packet_import", out, tc.want)
			}
		})
	}
}
