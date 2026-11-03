package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/store"
)

// TestReadLineEOFWithNoInputErrors confirms readLine reports errInputEnded,
// not ("", nil), when stdin is exhausted before any line is available: that
// ("", nil) is exactly what an empty Enter-press answer looks like, and is
// what let askRequired spin forever reprinting its prompt (F3).
func TestReadLineEOFWithNoInputErrors(t *testing.T) {
	p := newPrompter(strings.NewReader(""), &bytes.Buffer{})
	if _, err := p.readLine(); !errors.Is(err, errInputEnded) {
		t.Fatalf("readLine on empty reader: err = %v, want errInputEnded", err)
	}
}

// TestReadLineEOFAfterBlankLinesErrors confirms readLine still returns the
// blank lines themselves as real input (err nil) -- a final line with no
// trailing newline, or a line that is itself just "\n", is not EOF -- and
// only errors once input is actually exhausted.
func TestReadLineEOFAfterBlankLinesErrors(t *testing.T) {
	p := newPrompter(strings.NewReader("\n\n\n"), &bytes.Buffer{})
	for i := 0; i < 3; i++ {
		line, err := p.readLine()
		if err != nil {
			t.Fatalf("readLine #%d: unexpected error %v", i+1, err)
		}
		if line != "" {
			t.Fatalf("readLine #%d: got %q, want empty line", i+1, line)
		}
	}
	if _, err := p.readLine(); !errors.Is(err, errInputEnded) {
		t.Fatalf("readLine after input exhausted: err = %v, want errInputEnded", err)
	}
}

// TestReadLineFinalLineWithoutNewlineIsRealInput confirms a last line with
// no trailing newline (how a file or pipe commonly ends) is still returned
// as a real answer, not an error: only a read that produces nothing at all
// is EOF with no answer.
func TestReadLineFinalLineWithoutNewlineIsRealInput(t *testing.T) {
	p := newPrompter(strings.NewReader("alice"), &bytes.Buffer{})
	line, err := p.readLine()
	if err != nil {
		t.Fatalf("readLine: unexpected error %v", err)
	}
	if line != "alice" {
		t.Fatalf("readLine: got %q, want %q", line, "alice")
	}
}

// withTimeout runs fn in a goroutine and fails the test instead of hanging
// if fn does not return within d: the bug F3 describes is an unbounded busy
// loop, so a fixed call that cannot return at all is exactly the failure
// mode these tests must catch, not just a wrong return value.
func withTimeout(t *testing.T, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("did not return within %s (unbounded loop?)", d)
	}
}

// TestAskRequiredEOFErrorsInsteadOfLooping is F3's core claim: with empty
// stdin, askRequired must return an error promptly, not loop forever
// reprinting "<label> is required." (observed by Phil emitting ~300MB/s to
// stdout).
func TestAskRequiredEOFErrorsInsteadOfLooping(t *testing.T) {
	var out bytes.Buffer
	p := newPrompter(strings.NewReader(""), &out)
	var err error
	withTimeout(t, 2*time.Second, func() {
		_, err = p.askRequired("Intent", "")
	})
	if !errors.Is(err, errInputEnded) {
		t.Fatalf("askRequired on empty stdin: err = %v, want errInputEnded", err)
	}
}

// TestAskRequiredEOFAfterBlankLinesErrorsInsteadOfLooping covers Phil's
// second probe: a few blank lines (each an empty, re-prompted answer) and
// then EOF, rather than EOF immediately.
func TestAskRequiredEOFAfterBlankLinesErrorsInsteadOfLooping(t *testing.T) {
	var out bytes.Buffer
	p := newPrompter(strings.NewReader("\n\n\n"), &out)
	var err error
	withTimeout(t, 2*time.Second, func() {
		_, err = p.askRequired("Intent", "")
	})
	if !errors.Is(err, errInputEnded) {
		t.Fatalf("askRequired after blank lines: err = %v, want errInputEnded", err)
	}
}

// TestAskSilentDefaultOnEOFIsNowAnError confirms ask -- the method
// resolveRepositoryIdentity calls with a non-empty default -- no longer
// treats EOF as "accept the default silently", which is what let a run's
// repository identity bind to a default the human never actually confirmed.
func TestAskSilentDefaultOnEOFIsNowAnError(t *testing.T) {
	p := newPrompter(strings.NewReader(""), &bytes.Buffer{})
	ans, err := p.ask("Repository identity to bind", "", "/some/default/path")
	if !errors.Is(err, errInputEnded) {
		t.Fatalf("ask on empty stdin with a default: err = %v, want errInputEnded", err)
	}
	if ans != "" {
		t.Fatalf("ask on empty stdin with a default: ans = %q, want empty", ans)
	}
}

// TestRunInitInteractiveEOFErrorsPromptly drives the real `hmarun init`
// (no --answers, so it reads interactively) with empty stdin and confirms
// it returns an error promptly instead of spinning on the first required
// question, and that it wrote neither contract.json nor plan.json.
func TestRunInitInteractiveEOFErrorsPromptly(t *testing.T) {
	outDir := t.TempDir()
	var stdout bytes.Buffer
	var err error
	withTimeout(t, 5*time.Second, func() {
		err = runInit([]string{"--out-dir", outDir}, strings.NewReader(""), &stdout)
	})
	if err == nil {
		t.Fatal("runInit with empty stdin: want an error, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "contract.json")); statErr == nil {
		t.Error("contract.json should not have been written")
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "plan.json")); statErr == nil {
		t.Error("plan.json should not have been written")
	}
}

// TestRunRunInteractiveEOFErrorsPromptly drives the real `hmarun run`
// without --auto-approve and with empty stdin. The very first prompt is the
// repository-identity one (resolveRepositoryIdentity, inside newRunner,
// before any step runs), so this also confirms that prompt now errors on
// EOF instead of silently binding the default identity (the other half of
// F3), and that nothing was ever written to the core or host chains.
func TestRunRunInteractiveEOFErrorsPromptly(t *testing.T) {
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)

	var stdout bytes.Buffer
	args := []string{
		"--store", storeDir, "--repo", repo, "--run_id", "run",
		"--contract", contractPath, "--plan", planPath, "--unit", "U1",
		"--hma-bin", hmaBin, "--stub-dispatch",
	}
	var err error
	withTimeout(t, 5*time.Second, func() {
		err = runRun(args, strings.NewReader(""), &stdout)
	})
	if err == nil {
		t.Fatal("runRun with empty stdin and no --auto-approve: want an error, got nil")
	}

	core, loadErr := store.New(storeDir).Load("run")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(core) != 0 {
		t.Errorf("core chain has %d records, want 0: EOF must not approve or record anything", len(core))
	}
	host, loadErr := hoststate.NewStore(storeDir).Load("run")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(host) != 0 {
		t.Errorf("host chain has %d records, want 0: EOF must not approve or record anything", len(host))
	}
}
