//go:build linux

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This control directly tests whether an imported budget finding can disappear
// through the clean-target HEAD gap. Each subtest owns its repository and store.
func TestAudit20FollowupCleanFindingsSwapRestore(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "stable", true: "swap_restore"}[mutate], func(t *testing.T) {
			creditRepoSetup = func(f *reviewFixture) {
				auditWrite(t, filepath.Join(f.root, "a"), strings.Repeat("approved over budget\n", 150))
				f.head = a20commit(t, f.root)
				c, e := captureTarget(context.Background(), f.root, f.base, false)
				if e != nil {
					t.Fatal(e)
				}
				f.diff = c.State.Diff
			}
			defer func() { creditRepoSetup = nil }()
			cf := newCreditFixture(t, "true")
			creditRepoSetup = nil
			approved := cf.head
			auditWrite(t, filepath.Join(cf.root, "a"), "small\n")
			other := a20commit(t, cf.root)
			a20git(t, cf.root, "reset", "--hard", approved)
			report := cf.approve("pass")
			restored := ""
			if mutate {
				bin := t.TempDir()
				counter := filepath.Join(bin, "count")
				restored = filepath.Join(bin, "restored")
				script := `#!/usr/bin/python3
import os,sys,subprocess
args=sys.argv[1:]
p=subprocess.run(['/usr/bin/git']+args)
if os.getcwd()==os.environ['A20F_ROOT']:
    if 'ls-tree' in args and args[-1]==os.environ['A20F_APPROVED']:
        path=os.environ['A20F_COUNTER']
        n=int(open(path).read())+1 if os.path.exists(path) else 1
        open(path,'w').write(str(n))
        if n==2:
            subprocess.run(['/usr/bin/git','update-ref','HEAD',os.environ['A20F_OTHER']],check=True)
    if 'cat-file' in args and not os.path.exists(os.environ['A20F_RESTORED']):
        subprocess.run(['/usr/bin/git','update-ref','HEAD',os.environ['A20F_APPROVED']],check=True)
        open(os.environ['A20F_RESTORED'],'w').write('restored\n')
sys.exit(p.returncode)
`
				auditWrite(t, filepath.Join(bin, "git"), script)
				if e := os.Chmod(filepath.Join(bin, "git"), 0700); e != nil {
					t.Fatal(e)
				}
				t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
				t.Setenv("A20F_ROOT", cf.root)
				t.Setenv("A20F_APPROVED", approved)
				t.Setenv("A20F_OTHER", other)
				t.Setenv("A20F_COUNTER", counter)
				t.Setenv("A20F_RESTORED", restored)
			}
			before, e := captureTarget(context.Background(), cf.root, cf.base, false)
			// The instrumentation above would trigger on this control capture. Reset
			// its scheduler and HEAD before the actual import starts.
			if mutate {
				a20git(t, cf.root, "update-ref", "HEAD", approved)
				os.Remove(os.Getenv("A20F_COUNTER"))
				os.Remove(restored)
			}
			if e != nil {
				t.Fatal(e)
			}
			ie := cf.importFindings(cf.latestReview(), report, "2026-09-16T12:07:00Z")
			if mutate && ie != nil {
				a20git(t, cf.root, "update-ref", "HEAD", approved)
				if !strings.Contains(ie.Error(), "HOST_REVIEW_TARGET_CHANGED") {
					t.Fatalf("unexpected refusal: %v", ie)
				}
				return
			}
			if mutate {
				if _, e := os.Stat(restored); e != nil {
					t.Fatalf("restoration hook was not reached: %v import=%v", e, ie)
				}
			}
			after, e := captureTarget(context.Background(), cf.root, cf.base, false)
			findings := cf.coreFindings()
			t.Logf("mutate=%v approved=%s other=%s importErr=%v findings=%+v restored_equal=%v captureErr=%v", mutate, approved, other, ie, findings, before.State == after.State, e)
			if ie != nil {
				if !mutate {
					t.Fatal(ie)
				}
				return
			}
			if len(findings) == 0 {
				t.Error("findings import omitted approved committed target's 150-line budget excess")
			}
		})
	}
}

func TestAudit20FollowupCleanUnsupported(t *testing.T) {
	cf := newCreditFixture(t, "true")
	in, out := d5c5pPacketInput(t, cf, "", "Audit a clean unsupported target")
	var req hostReviewPacketInput
	if e := decodeStrictFile(in, &req); e != nil {
		t.Fatal(e)
	}
	req.DirtyWorktreeApproved = false
	writeJSON(t, in, req)
	sentinel := errors.New("independent clean-platform sentinel")
	errDirtyObservationUnsupported = sentinel
	defer func() { errDirtyObservationUnsupported = nil }()
	err := run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	t.Logf("clean_target=true dirty_approved=false error=%v", err)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected unconditional platform refusal, got %v", err)
	}
}
