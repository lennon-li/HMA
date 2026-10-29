// Independent audit controls retained as regression tests.
//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/repostate"
)

func a20git(t *testing.T, root string, args ...string) string {
	t.Helper()
	c := exec.Command("/usr/bin/git", append([]string{"-C", root, "-c", "user.name=Auditor", "-c", "user.email=audit@example.invalid"}, args...)...)
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %q: %v: %s", args, e, b)
	}
	return strings.TrimSpace(string(b))
}
func a20commit(t *testing.T, root string) string {
	a20git(t, root, "add", "-A")
	a20git(t, root, "commit", "-qm", "audit fixture")
	return a20git(t, root, "rev-parse", "HEAD")
}

func TestAudit20IgnoreCaseParity(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, ".gitignore"), "SECRET\n")
	base := a20commit(t, repo)
	auditWrite(t, filepath.Join(repo, "secret"), strings.Repeat("unreviewed\n", 150))
	global := filepath.Join(t.TempDir(), "global.config")
	auditWrite(t, global, "[core]\n ignoreCase = true\n")
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	legacy, e := repostate.Snapshot(context.Background(), repo, base, true)
	if e != nil {
		t.Fatal(e)
	}
	c, e := captureTarget(context.Background(), repo, base, true)
	if e != nil {
		t.Fatal(e)
	}
	m, e := buildCapturedManifest(context.Background(), repo, base, true, &c)
	if e != nil {
		t.Fatal(e)
	}
	n, l, _, e := measureCaptured(context.Background(), repo, base, true, &c)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("legacy=%+v capture=%+v changed=%v manifest=%+v measurement=(%d,%d)", legacy, c.State, c.Changed, m, n, l)
	if c.State != legacy || n != 1 || l != 150 {
		t.Error("captured ignore rules omitted a case-distinct file under inherited core.ignoreCase=true")
	}
}

func TestAudit20IgnoreSemanticsParity(t *testing.T) {
	cases := []struct {
		name, root, nested string
		untrackedIgnore    bool
	}{
		{"nested-negation", "*.log\n!keep.log\n", "*.tmp\n!keep.tmp\n", false},
		{"excluded-parent", "build/\n!build/keep\n", "", false},
		{"reincluded-parent", "build/*\n!build/keep\n", "", false},
		{"directory-only", "plain/\nsub/drop/\n", "", false},
		{"globstar", "**/drop*\n!sub/keep.tmp\n", "", false},
		{"untracked-self-ignore", ".gitignore\n*\n", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := gitRepo(t)
			auditWrite(t, filepath.Join(repo, "tracked"), "base\n")
			if !tc.untrackedIgnore {
				auditWrite(t, filepath.Join(repo, ".gitignore"), tc.root)
				auditWrite(t, filepath.Join(repo, "sub/.gitignore"), tc.nested)
			}
			base := a20commit(t, repo)
			if tc.untrackedIgnore {
				auditWrite(t, filepath.Join(repo, ".gitignore"), tc.root)
			}
			for _, p := range []string{"drop.log", "keep.log", "sub/drop.tmp", "sub/keep.tmp", "sub/keep.log", "build/drop", "build/keep", "plain", "global-only", "info-only"} {
				auditWrite(t, filepath.Join(repo, p), "new\n")
			}
			auditWrite(t, filepath.Join(repo, ".git/info/exclude"), "info-only\n")
			globalRules := filepath.Join(t.TempDir(), "excludes")
			auditWrite(t, globalRules, "global-only\n")
			a20git(t, repo, "config", "core.excludesFile", globalRules)
			old, e := repostate.Snapshot(context.Background(), repo, base, true)
			if e != nil {
				t.Fatal(e)
			}
			c, e := captureTarget(context.Background(), repo, base, true)
			if e != nil {
				t.Fatal(e)
			}
			oldPaths, e := repostate.ChangedPaths(context.Background(), repo, base)
			if e != nil {
				t.Fatal(e)
			}
			if old != c.State || !reflect.DeepEqual(oldPaths, c.Changed) {
				t.Errorf("legacy=%+v %v capture=%+v %v", old, oldPaths, c.State, c.Changed)
			}
			t.Logf("changed=%v", c.Changed)
		})
	}
}

type a20Reader struct {
	root                   string
	reads                  map[string]int
	liveFlip, returnedFlip bool
}

func (r *a20Reader) ReadEntry(ctx context.Context, p string) (repostate.WorktreeEntry, error) {
	e, err := (anchoredReader{r.root}).ReadEntry(ctx, p)
	r.reads[p]++
	if p == ".gitignore" && r.reads[p] == 2 {
		if r.liveFlip {
			err = os.WriteFile(filepath.Join(r.root, p), []byte("hidden\n"), 0600)
		}
		if r.returnedFlip {
			e.Data = []byte("hidden\n")
		}
	}
	return e, err
}
func TestAudit20IgnoreCapturedBytes(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(fmt.Sprint(live), func(t *testing.T) {
			repo := gitRepo(t)
			auditWrite(t, filepath.Join(repo, ".gitignore"), "# original\n")
			base := a20commit(t, repo)
			auditWrite(t, filepath.Join(repo, "hidden"), "visible\n")
			r := &a20Reader{root: repo, reads: map[string]int{}, liveFlip: live, returnedFlip: !live}
			c, e := repostate.CaptureTarget(context.Background(), repo, base, true, r)
			t.Logf("live=%v reads=%v state=%+v changed=%v err=%v", live, r.reads, c.State, c.Changed, e)
			if !live {
				if e == nil || !strings.Contains(e.Error(), "changed while it was captured") {
					t.Fatal("disagreeing ignore read not refused")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(c.Changed, []string{"hidden"}) {
				t.Fatal("live ignore bytes influenced captured evaluation")
			}
		})
	}
}

func TestAudit20CleanCaptureHeadBinding(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "f"), "base\n")
	base := a20commit(t, repo)
	auditWrite(t, filepath.Join(repo, "f"), "approved\n")
	approved := a20commit(t, repo)
	c, e := captureTarget(context.Background(), repo, base, false)
	if e != nil {
		t.Fatal(e)
	}
	auditWrite(t, filepath.Join(repo, "f"), strings.Repeat("other revision\n", 150))
	other := a20commit(t, repo)
	m, e := buildCapturedManifest(context.Background(), repo, base, false, &c)
	n, l, _, me := measureCaptured(context.Background(), repo, base, false, &c)
	a20git(t, repo, "reset", "--hard", approved)
	after, ae := captureTarget(context.Background(), repo, base, false)
	t.Logf("approved=%s other=%s manifest=%+v manifestErr=%v measured=(%d,%d) measureErr=%v restored_equal=%v afterErr=%v", approved, other, m, e, n, l, me, c.State == after.State, ae)
	if e == nil || me == nil {
		t.Error("clean observation and measurement accepted a HEAD different from supplied capture")
	}
}

func TestAudit20CleanPacketHeadSwapRestore(t *testing.T) {
	cf := newCreditFixture(t, "true")
	approved := a20git(t, cf.root, "rev-parse", "HEAD")
	original, e := os.ReadFile(filepath.Join(cf.root, "a"))
	if e != nil {
		t.Fatal(e)
	}
	transient := []byte("manifest from a different committed revision\n")
	auditWrite(t, filepath.Join(cf.root, "a"), string(transient))
	other := a20commit(t, cf.root)
	a20git(t, cf.root, "reset", "--hard", approved)
	in, out := d5c5pPacketInput(t, cf, "", "Audit clean HEAD binding.")
	var req hostReviewPacketInput
	if e = decodeStrictFile(in, &req); e != nil {
		t.Fatal(e)
	}
	req.DirtyWorktreeApproved = false
	writeJSON(t, in, req)
	bin := t.TempDir()
	armed := filepath.Join(bin, "armed")
	restored := filepath.Join(bin, "restored")
	script := `#!/usr/bin/python3
import os,sys,subprocess
args=sys.argv[1:]
p=subprocess.run(['/usr/bin/git']+args)
if os.getcwd()==os.environ['A20_ROOT'] and 'cat-file' in args and os.path.exists(os.environ['A20_ARMED']):
    os.unlink(os.environ['A20_ARMED'])
    subprocess.run(['/usr/bin/git','update-ref','HEAD',os.environ['A20_APPROVED']],check=True)
    open(os.environ['A20_RESTORED'],'w').write('restored\n')
sys.exit(p.returncode)
`
	auditWrite(t, filepath.Join(bin, "git"), script)
	if e = os.Chmod(filepath.Join(bin, "git"), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	t.Setenv("A20_ROOT", cf.root)
	t.Setenv("A20_APPROVED", approved)
	t.Setenv("A20_ARMED", armed)
	t.Setenv("A20_RESTORED", restored)
	reviewPacketBeforeManifestHook = func() { a20git(t, cf.root, "update-ref", "HEAD", other); auditWrite(t, armed, "armed\n") }
	defer func() { reviewPacketBeforeManifestHook = nil }()
	e = run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	if e != nil {
		a20git(t, cf.root, "update-ref", "HEAD", approved)
		if !strings.Contains(e.Error(), hoststate.ReasonReviewTargetChanged) {
			t.Fatalf("unexpected refusal: %v", e)
		}
		return
	}
	if _, err := os.Stat(restored); err != nil {
		t.Fatalf("schedule did not restore HEAD: %v (packet %v)", err, e)
	}
	st := packetState(t, out)
	if dest := os.Getenv("A20_ARTIFACT_DIR"); dest != "" {
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		auditWrite(t, filepath.Join(dest, "head-swap-packet.md"), string(b))
	}
	var digest string
	for _, f := range st.Manifest {
		if f.Path == "a" {
			digest = f.SHA256
		}
	}
	history, e := hoststate.NewStore(cf.store).Load("run")
	if e != nil {
		t.Fatal(e)
	}
	p := history[len(history)-1].Dispatch.Packet
	live, le := captureLiveTarget(history, history[len(history)-1].HeadAnchor)
	t.Logf("bound_head=%s approved=%s other=%s manifest_sha=%s approved_sha=%s other_sha=%s liveErr=%v live=%+v", p.ProducedHead, approved, other, digest, hashBytes(original), hashBytes(transient), le, live)
	if p.ProducedHead == approved && digest == hashBytes(transient) && le == nil {
		t.Error("accepted packet binds approved HEAD but hashes bytes of transient other HEAD; live check also accepts")
	}
}

func TestAudit20IndependentPathChecks(t *testing.T) {
	for _, p := range []string{"tracked", "new", "link"} {
		t.Run(p, func(t *testing.T) {
			repo := gitRepo(t)
			auditWrite(t, filepath.Join(repo, "tracked"), "base\n")
			base := a20commit(t, repo)
			auditWrite(t, filepath.Join(repo, "tracked"), "dirty\n")
			auditWrite(t, filepath.Join(repo, "new"), "new\n")
			if e := os.Symlink("tracked", filepath.Join(repo, "link")); e != nil {
				t.Fatal(e)
			}
			c, e := captureTarget(context.Background(), repo, base, true)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = buildCapturedManifest(context.Background(), repo, base, true, &c); e != nil {
				t.Fatal(e)
			}
			if p == "link" {
				os.Remove(filepath.Join(repo, p))
				if e = os.Symlink("new", filepath.Join(repo, p)); e != nil {
					t.Fatal(e)
				}
			} else {
				auditWrite(t, filepath.Join(repo, p), "changed\n")
			}
			_, e = buildCapturedManifest(context.Background(), repo, base, true, &c)
			_, _, _, me := measureCaptured(context.Background(), repo, base, true, &c)
			if e == nil || me == nil {
				t.Errorf("path=%s manifest=%v measure=%v", p, e, me)
			}
			keys := []string{}
			for k := range c.Entries {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			t.Logf("isolated path=%s captured=%v manifest=%v measure=%v", p, keys, e, me)
		})
	}
}

func TestAudit20IgnoreCasePacketAndFindings(t *testing.T) {
	global := filepath.Join(t.TempDir(), "global.config")
	auditWrite(t, global, "[core]\n ignoreCase = true\n")
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	creditRepoSetup = func(f *reviewFixture) {
		auditWrite(t, filepath.Join(f.root, ".gitignore"), "SECRET\n")
		f.head = a20commit(t, f.root)
		st, e := repostate.Snapshot(context.Background(), f.root, f.base, false)
		if e != nil {
			t.Fatal(e)
		}
		f.diff = st.Diff
	}
	defer func() { creditRepoSetup = nil }()
	cf := newCreditFixture(t, "true")
	creditRepoSetup = nil
	auditWrite(t, filepath.Join(cf.root, "secret"), strings.Repeat("unreviewed\n", 150))
	in, out := d5c5pPacketInput(t, cf, "", "Audit ignore configuration.")
	e := run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	if e != nil {
		t.Fatalf("packet: %v", e)
	}
	history, e := hoststate.NewStore(cf.store).Load("run")
	if e != nil {
		t.Fatal(e)
	}
	last := history[len(history)-1]
	p := last.Dispatch.Packet
	st := packetState(t, out)
	seen := false
	for _, f := range st.Manifest {
		if f.Path == "secret" {
			seen = true
		}
	}
	cf.packetPath = out
	cf.packetSHA = last.Dispatch.ReviewPacketDigest
	cf.rdID = p.ReviewDispatchID
	cf.packetNonce = p.ReviewNonce
	cf.runID = p.DispatchRunID
	cf.packetTS = "2026-09-16T12:31:00Z"
	cf.importTS = "2026-09-16T12:32:00Z"
	cf.launchHead = p.ExpectedLaunchHead
	cf.launchWT = p.ExpectedLaunchWorktree
	report := cf.approve("pass")
	ie := cf.importFindings(cf.latestReview(), report, "2026-09-16T12:33:00Z")
	findings := cf.coreFindings()
	t.Logf("packet_worktree=%q secret_in_manifest=%v findings_import_err=%v findings=%+v", p.WorktreeDigest, seen, ie, findings)
	if dest := os.Getenv("A20_ARTIFACT_DIR"); dest != "" {
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		auditWrite(t, filepath.Join(dest, "ignorecase-packet.md"), string(b))
		b, _ = json.MarshalIndent(findings, "", "  ")
		auditWrite(t, filepath.Join(dest, "ignorecase-findings.json"), string(b)+"\n")
	}
	if p.WorktreeDigest == "" && !seen && ie == nil && len(findings) == 0 {
		t.Error("accepted clean packet and empty findings snapshot while omitted untracked file exceeds approved line budget")
	}
}
