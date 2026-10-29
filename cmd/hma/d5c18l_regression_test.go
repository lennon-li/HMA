//go:build linux

// Audit d5c18l (independent, liz / OpenAI, commit a187765) reproduced four
// findings in D5c's capture mechanism, after the previous round's F01/F02/F03
// fix (commit a187765):
//
//   - D5C18L-F01: changed-path discovery's ignore-rule read
//     (ls-files --others --exclude-per-directory=.gitignore) goes straight to
//     git, not through the WorktreeReader, so a writer that transiently
//     rewrites .gitignore can hide an untracked file from the capture, the
//     packet manifest, and the budget measurement, even though the file sits
//     on disk the whole time.
//   - D5C18L-F02: when a caller claims dirty_worktree_approved=false,
//     captureTarget calls the legacy, unanchored Snapshot first to find out
//     whether the target is actually dirty; that path can read a newly
//     changed tracked file before refusing, on every platform, not just an
//     unsupported one.
//   - D5C18L-F03: not reproduced here as a separate test; it is a coverage
//     gap in cmd/hma/d5c16p_capture_test.go and
//     cmd/hma/d5c16p_regression_test.go (an earlier mutation in the same
//     test leaves a file changed so a later path's own check is masked).
//     Fix those two tests' assertions directly; do not weaken them further.
//   - D5C18L-F04: documentation only (handoff overclaims a 16 MiB memory
//     bound that does not cover the capture/discovery read); correct the
//     text, no test here.
//
// These tests must pass without internal/repostate/raw.go changing.
package main

import (
	"context"
	"errors"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/repostate"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type audit18Reader struct {
	root   string
	reads  map[string]int
	mutate bool
}

func (r *audit18Reader) ReadEntry(ctx context.Context, p string) (repostate.WorktreeEntry, error) {
	e, err := (anchoredReader{r.root}).ReadEntry(ctx, p)
	r.reads[p]++
	if r.mutate && p == "f" && r.reads[p] == 2 {
		e.Data = []byte("different second read\n")
	}
	return e, err
}
func TestAudit18RepeatedReadIdentity(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "stable", true: "changed"}[mutate], func(t *testing.T) {
			repo := gitRepo(t)
			auditWrite(t, filepath.Join(repo, "f"), "base\n")
			base := d9Commit(t, repo, "base")
			auditWrite(t, filepath.Join(repo, "f"), "dirty\n")
			r := &audit18Reader{root: repo, reads: map[string]int{}, mutate: mutate}
			_, err := repostate.CaptureTarget(context.Background(), repo, base, true, r)
			t.Logf("reads=%v err=%v", r.reads, err)
			if r.reads["f"] != 2 || (mutate && (err == nil || !strings.Contains(err.Error(), "changed while it was captured"))) || (!mutate && err != nil) {
				t.Fatal("repeat-read identity failure")
			}
		})
	}
}
func TestAudit18EachPathAndMeasurementBound(t *testing.T) {
	for _, kind := range []string{"tracked", "untracked", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			repo := gitRepo(t)
			auditWrite(t, filepath.Join(repo, "f"), "base\n")
			base := d9Commit(t, repo, "base")
			auditWrite(t, filepath.Join(repo, "f"), "dirty\n")
			auditWrite(t, filepath.Join(repo, "u"), "untracked\n")
			if err := os.Symlink("f", filepath.Join(repo, "l")); err != nil {
				t.Fatal(err)
			}
			c, err := captureTarget(context.Background(), repo, base, true)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, _, err = measureCaptured(context.Background(), repo, base, true, &c); err != nil {
				t.Fatalf("stable: %v", err)
			}
			switch kind {
			case "tracked":
				auditWrite(t, filepath.Join(repo, "f"), "transient\n")
			case "untracked":
				auditWrite(t, filepath.Join(repo, "u"), "transient\n")
			case "symlink":
				os.Remove(filepath.Join(repo, "l"))
				if err := os.Symlink("u", filepath.Join(repo, "l")); err != nil {
					t.Fatal(err)
				}
			}
			_, err = buildCapturedManifest(context.Background(), repo, base, true, &c)
			_, _, _, me := measureCaptured(context.Background(), repo, base, true, &c)
			t.Logf("kind=%s manifest=%v measurement=%v", kind, err, me)
			if err == nil || err.Error() != hoststate.ReasonReviewTargetChanged || me == nil || me.Error() != hoststate.ReasonReviewTargetChanged {
				t.Fatal("changed path accepted")
			}
		})
	}
}
func TestAudit18UnsupportedFalsePacketAndLive(t *testing.T) {
	for _, which := range []string{"packet", "live"} {
		t.Run(which, func(t *testing.T) {
			cf := newCreditFixture(t, "true")
			in, out := d5c5pPacketInput(t, cf, "", "audit unsupported false")
			var req hostReviewPacketInput
			if err := decodeStrictFile(in, &req); err != nil {
				t.Fatal(err)
			}
			req.DirtyWorktreeApproved = false
			writeJSON(t, in, req)
			clean, err := captureTarget(context.Background(), cf.root, cf.base, false)
			if err != nil {
				t.Fatal(err)
			}
			history := []hoststate.Record{{HeadAnchor: "dispatch", Dispatch: &hoststate.Dispatch{Packet: &hoststate.ReviewPacketBinding{ProjectRoot: cf.root, BaseRevision: cf.base, ProducedHead: clean.State.Head, DiffDigest: clean.State.Diff}}}}
			auditWrite(t, filepath.Join(cf.root, "a"), "dirty after clean binding\n")
			fd := independentWatch(t, filepath.Join(cf.root, "a"))
			sentinel := errors.New("audit unsupported platform")
			errDirtyObservationUnsupported = sentinel
			defer func() { errDirtyObservationUnsupported = nil }()
			if which == "packet" {
				err = run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
			} else {
				err = requireLiveTarget(history, "dispatch")
			}
			n := independentEvents(fd)
			t.Logf("route=%s err=%v dirty-file-event-bytes=%d", which, err, n)
			if n > 0 {
				t.Error("unsupported platform opened dirty content before refusing")
			}
		})
	}
}
func audit18IgnoreScheduler(t *testing.T, repo string) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/usr/bin/python3
import os,sys,subprocess
args=sys.argv[1:]
p=os.path.join(os.getcwd(),'.gitignore')
flip='--exclude-per-directory=.gitignore' in args and os.getcwd()==os.environ['AUDIT18_ROOT']
if flip:
    old=open(p,'rb').read() if os.path.exists(p) else None
    with open(p,'wb') as f: f.write(b'hidden\n.gitignore\n')
try: code=subprocess.call(['/usr/bin/git']+args)
finally:
    if flip:
        if old is None: os.unlink(p)
        else:
            with open(p,'wb') as f: f.write(old)
sys.exit(code)
`
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUDIT18_ROOT", repo)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
}
func TestAudit18IgnoreReadIsBound(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, ".gitignore"), "# no ignore rules\n")
	base := d9Commit(t, repo, "base")
	auditWrite(t, filepath.Join(repo, "hidden"), strings.Repeat("over budget\n", 150))
	stable, err := captureTarget(context.Background(), repo, base, true)
	if err != nil || stable.State.Worktree == "" {
		t.Fatalf("stable control: %+v %v", stable, err)
	}
	audit18IgnoreScheduler(t, repo)
	c, err := captureTarget(context.Background(), repo, base, true)
	t.Logf("stable changed=%v scheduled changed=%v worktree=%s err=%v", stable.Changed, c.Changed, c.State.Worktree, err)
	if err != nil {
		return
	}
	m, me := buildCapturedManifest(context.Background(), repo, base, true, &c)
	files, lines, _, fe := measureCaptured(context.Background(), repo, base, true, &c)
	second, se := captureTarget(context.Background(), repo, base, true)
	t.Logf("manifest=%+v err=%v measured=(%d,%d) err=%v second_equal=%v second_err=%v", m, me, files, lines, fe, c.State == second.State, se)
	if me == nil && fe == nil && se == nil && c.State == second.State && len(m) == 0 {
		t.Error("transient unbound ignore rules concealed untracked content from matching captures, manifest and budget")
	}
}

func TestAudit18IgnorePacketAndLive(t *testing.T) {
	cf := newCreditFixture(t, "true")
	// The committed target is initially clean. An untracked file remains on disk.
	auditWrite(t, filepath.Join(cf.root, "hidden"), strings.Repeat("over budget\n", 150))
	in, out := d5c5pPacketInput(t, cf, "", "audit ignore schedule")
	stable, err := captureTarget(context.Background(), cf.root, cf.base, true)
	if err != nil {
		t.Fatal(err)
	}
	audit18IgnoreScheduler(t, cf.root)
	err = run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	if err != nil {
		t.Fatalf("packet refused: %v", err)
	}
	history, err := hoststate.NewStore(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	last := history[len(history)-1]
	p := last.Dispatch.Packet
	live, le := captureLiveTarget(history, last.HeadAnchor)
	st := packetState(t, out)
	seen := false
	for _, e := range st.Manifest {
		if e.Path == "hidden" {
			seen = true
		}
	}
	if _, err := os.Stat(filepath.Join(cf.root, "hidden")); err != nil {
		t.Fatal(err)
	}
	t.Logf("stable_worktree=%s packet_worktree=%q hidden_in_manifest=%v live_err=%v live=%+v", stable.State.Worktree, p.WorktreeDigest, seen, le, live)
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
	t.Logf("findings_import_err=%v findings=%+v", ie, cf.coreFindings())
	if p.WorktreeDigest == "" && !seen && le == nil {
		t.Error("packet and live import check accept a clean artifact while 150-line untracked file is present")
	}
	if ie == nil && len(cf.coreFindings()) == 0 {
		t.Error("findings import drops 150-line untracked budget excess")
	}

}
