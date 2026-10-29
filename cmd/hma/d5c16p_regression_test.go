//go:build linux

// Audit d5c16p (independent, prof / OpenAI / gpt-6-astra, commit f4daede)
// reproduced three HIGH-severity findings in D5c's dirty-target handling:
//
//   - D5C16P-F01: review-packet generation and the shared requireLiveTarget
//     check (used by review-import and findings-import) can read tracked
//     worktree content without ever checking errDirtyObservationUnsupported;
//     only observeTargetLimit checked it.
//   - D5C16P-F02: a writer can change a tracked file, let it be measured,
//     then restore it before the second ("live target") snapshot; both
//     snapshots match, but the accepted packet manifest and findings import
//     can describe the transient bytes, and a real budget excess can be
//     silently dropped.
//   - D5C16P-F03: dirty-path discovery (repostate.ChangedPaths, Gate B,
//     unchanged by this file) does unanchored Lstat+os.ReadFile by pathname;
//     a concurrent writer swapping a tracked directory for a symlink can
//     make it open a file outside the repository.
//
// These tests must pass without internal/repostate/raw.go changing.
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/repostate"
)

func independentWatch(t *testing.T, path string) int {
	t.Helper()
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syscall.Close(fd) })
	if _, err = syscall.InotifyAddWatch(fd, path, syscall.IN_OPEN|syscall.IN_ACCESS); err != nil {
		t.Fatal(err)
	}
	return fd
}
func independentEvents(fd int) int {
	b := make([]byte, 8192)
	n, _ := syscall.Read(fd, b)
	if n < 0 {
		return 0
	}
	return n
}

func TestIndependentUnsupportedPacketRead(t *testing.T) {
	cf := newCreditFixture(t, "true")
	auditWrite(t, filepath.Join(cf.root, "a"), "dirty target\n")
	in, out := d5c5pPacketInput(t, cf, "", "Audit platform refusal.")
	fd := independentWatch(t, filepath.Join(cf.root, "a"))
	sentinel := errors.New("independent unsupported platform")
	errDirtyObservationUnsupported = sentinel
	defer func() { errDirtyObservationUnsupported = nil }()
	err := run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	events := independentEvents(fd)
	t.Logf("packet error=%v file-access-event-bytes=%d", err, events)
	if !errors.Is(err, sentinel) {
		t.Fatalf("wrong refusal: %v", err)
	}
	if events > 0 {
		t.Error("dirty tracked file was opened before platform refusal")
	}
}

func TestIndependentUnsupportedLiveTargetRead(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "f"), "one\n")
	base := d9Commit(t, repo, "base")
	auditWrite(t, filepath.Join(repo, "f"), "two\n")
	snap, err := repostate.Snapshot(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	history := []hoststate.Record{{HeadAnchor: "dispatch", Dispatch: &hoststate.Dispatch{Packet: &hoststate.ReviewPacketBinding{ProjectRoot: repo, BaseRevision: base, ProducedHead: snap.Head, DiffDigest: snap.Diff, WorktreeDigest: snap.Worktree}}}}
	fd := independentWatch(t, filepath.Join(repo, "f"))
	sentinel := errors.New("independent unsupported platform")
	errDirtyObservationUnsupported = sentinel
	defer func() { errDirtyObservationUnsupported = nil }()
	err = requireLiveTarget(history, "dispatch")
	events := independentEvents(fd)
	t.Logf("live-target error=%v file-access-event-bytes=%d", err, events)
	if err == nil || events > 0 {
		t.Error("shared import target check bypassed unsupported-platform refusal")
	}
}

func TestIndependentPacketSwapRestore(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "stable", true: "swap_restore"}[mutate], func(t *testing.T) {
			cf := newCreditFixture(t, "true")
			a := filepath.Join(cf.root, "a")
			original, err := os.ReadFile(a)
			if err != nil {
				t.Fatal(err)
			}
			auditWrite(t, filepath.Join(cf.root, "z"), "stable dirty file\n")
			before, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
			if err != nil {
				t.Fatal(err)
			}
			in, out := d5c5pPacketInput(t, cf, "", "Audit exact target bytes.")
			transient := []byte("transient bytes not in bound artifact\n")
			restored := false
			if mutate {
				reviewPacketBeforeManifestHook = func() {
					if err := os.WriteFile(a, transient, 0600); err != nil {
						t.Fatal(err)
					}
				}
				worktreeWalkHook = func(rel string) {
					if rel == "z" {
						if err := os.WriteFile(a, original, 0600); err != nil {
							t.Fatal(err)
						}
						restored = true
					}
				}
			}
			defer func() { reviewPacketBeforeManifestHook = nil; worktreeWalkHook = nil }()
			err = run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
			if err != nil {
				if !mutate {
					t.Fatal(err)
				}
				t.Logf("mutation refused: %v", err)
				return
			}
			after, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
			if err != nil {
				t.Fatal(err)
			}
			st := packetState(t, out)
			var observed string
			for _, e := range st.Manifest {
				if e.Path == "a" {
					observed = e.SHA256
				}
			}
			t.Logf("mutate=%v restored=%v snapshots_equal=%v manifest_sha=%s actual_sha=%s transient_sha=%s", mutate, restored, before == after, observed, hashBytes(original), hashBytes(transient))
			if observed != hashBytes(original) {
				t.Error("accepted packet manifest hashes transient bytes outside its bound target")
			}
		})
	}
}

func TestIndependentDirtyDiscoveryReadsOutsideParent(t *testing.T) {
	repo := gitRepo(t)
	sub := filepath.Join(repo, "sub")
	saved := filepath.Join(repo, "saved")
	auditWrite(t, filepath.Join(sub, "f"), "inside\n")
	base := d9Commit(t, repo, "base")
	outside := t.TempDir()
	auditWrite(t, filepath.Join(outside, "f"), "outside synthetic test data\n")
	fd := independentWatch(t, filepath.Join(outside, "f"))
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if os.Rename(sub, saved) != nil {
				continue
			}
			if os.Symlink(outside, sub) == nil {
				time.Sleep(time.Microsecond)
				os.Remove(sub)
			}
			os.Rename(saved, sub)
		}
	}()
	deadline := time.Now().Add(8 * time.Second)
	events := 0
	attempts := 0
	for time.Now().Before(deadline) {
		attempts++
		observeTarget(context.Background(), repo, base, true)
		if n := independentEvents(fd); n > 0 {
			events += n
			break
		}
	}
	close(stop)
	wg.Wait()
	t.Logf("observations=%d outside-file-access-event-bytes=%d", attempts, events)
	if events > 0 {
		t.Error("dirty discovery followed a swapped parent and opened an outside file")
	} else {
		t.Log("race not observed within bounded run; no safety conclusion")
	}
}

func TestIndependentFindingsBudgetSwapRestore(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "stable", true: "swap_restore"}[mutate], func(t *testing.T) {
			cf := newCreditFixture(t, "true")
			a := filepath.Join(cf.root, "a")
			original := []byte("")
			for i := 0; i < 150; i++ {
				original = append(original, []byte("over budget\n")...)
			}
			if err := os.WriteFile(a, original, 0600); err != nil {
				t.Fatal(err)
			}
			auditWrite(t, filepath.Join(cf.root, "z"), "stable dirty file\n")
			in, out := d5c5pPacketInput(t, cf, "", "Audit budget binding.")
			if err := run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out}); err != nil {
				t.Fatal(err)
			}
			history, err := hoststate.NewStore(cf.store).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			d := history[len(history)-1].Dispatch
			cf.packetPath = out
			cf.packetSHA = d.ReviewPacketDigest
			cf.rdID = d.Packet.ReviewDispatchID
			cf.packetNonce = d.Packet.ReviewNonce
			cf.runID = d.Packet.DispatchRunID
			cf.packetTS = "2026-09-16T12:31:00Z"
			cf.importTS = "2026-09-16T12:32:00Z"
			cf.launchHead = d.Packet.ExpectedLaunchHead
			cf.launchWT = d.Packet.ExpectedLaunchWorktree
			report := cf.approve("pass")
			before, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
			if err != nil {
				t.Fatal(err)
			}
			changed, restored := false, false
			if mutate {
				worktreeWalkHook = func(rel string) {
					if rel == "a" {
						if err := os.WriteFile(a, []byte("small\n"), 0600); err != nil {
							t.Fatal(err)
						}
						changed = true
					}
					if rel == "z" {
						if err := os.WriteFile(a, original, 0600); err != nil {
							t.Fatal(err)
						}
						restored = true
					}
				}
			}
			defer func() { worktreeWalkHook = nil }()
			err = cf.importFindings(cf.latestReview(), report, "2026-09-16T12:33:00Z")
			if err != nil {
				if !mutate {
					t.Fatal(err)
				}
				t.Logf("mutation refused: %v", err)
				return
			}
			after, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
			if err != nil {
				t.Fatal(err)
			}
			got := cf.coreFindings()
			t.Logf("mutate=%v changed=%v restored=%v snapshots_equal=%v findings=%v", mutate, changed, restored, before == after, got)
			if len(got) == 0 {
				t.Error("findings import omitted actual line-budget excess after transient small measurement")
			}
		})
	}
}
