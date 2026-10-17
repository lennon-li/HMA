package host

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func appendDecision(t *testing.T, s *Store) Record {
	t.Helper()
	d := decisionFor(testRoute("impl", "p", "pf", "m", "mf", "approved", false), DecisionDelegate)
	r := record(1, "", KindDecision)
	r.Decision = &d
	if err := s.Append(&r); err != nil {
		t.Fatal(err)
	}
	return r
}

// H3: crash after the record landed but before the head moved. The pending
// intent names the record, so Load completes the commit.
func TestGateBHostInterruptedHead(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	first := appendDecision(t, s)
	second := appendDecision(t, s)
	runDir := filepath.Join(dir, "host", "run")
	// Reconstruct the on-disk state between the record write and the head
	// write of the second append: pending = new anchor, head = old anchor.
	if err := os.WriteFile(filepath.Join(runDir, "head"), []byte(first.HeadAnchor+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "head.pending"), []byte(second.HeadAnchor+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("run")
	if err != nil {
		t.Fatalf("interrupted host append has no recovery: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("recovered %d records, want 2", len(got))
	}
	if _, err := os.Stat(filepath.Join(runDir, "head.pending")); !os.IsNotExist(err) {
		t.Fatalf("pending intent left behind: %v", err)
	}
	if err := appendDecisionErr(s); err != nil {
		t.Fatalf("append after recovery: %v", err)
	}
}

// H3: crash after the intent but before the record landed. The intent is
// discarded and the chain is unchanged.
func TestGateBHostInterruptedBeforeRecord(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	appendDecision(t, s)
	runDir := filepath.Join(dir, "host", "run")
	if err := os.WriteFile(filepath.Join(runDir, "head.pending"), []byte("never-written\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("run")
	if err != nil || len(got) != 1 {
		t.Fatalf("Load = %d records, %v; want 1, nil", len(got), err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "head.pending")); !os.IsNotExist(err) {
		t.Fatalf("stale intent not discarded: %v", err)
	}
}

// Without a matching intent, a head that lags or is missing is still drift.
func TestGateBHostHeadDriftStillDetected(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	first := appendDecision(t, s)
	appendDecision(t, s)
	runDir := filepath.Join(dir, "host", "run")
	if err := os.WriteFile(filepath.Join(runDir, "head"), []byte(first.HeadAnchor+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("run"); err == nil || err.Error() != ReasonHostHeadMismatch {
		t.Fatalf("lagging head without intent: %v, want %s", err, ReasonHostHeadMismatch)
	}
	if err := os.Remove(filepath.Join(runDir, "head")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("run"); err == nil {
		t.Fatal("missing head with records accepted")
	}
}

// H3: concurrent appenders are serialized by the run lock; none is lost and
// the chain stays continuous.
func TestGateBHostConcurrentAppends(t *testing.T) {
	dir := t.TempDir()
	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- appendDecisionErr(NewStore(dir))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	got, err := NewStore(dir).Load("run")
	if err != nil || len(got) != n {
		t.Fatalf("Load = %d records, %v; want %d", len(got), err, n)
	}
}

func appendDecisionErr(s *Store) error {
	d := decisionFor(testRoute("impl", "p", "pf", "m", "mf", "approved", false), DecisionDelegate)
	r := record(1, "", KindDecision)
	r.Decision = &d
	return s.Append(&r)
}

// C02: a relative store path still syncs absolute ancestors. syncAncestors
// must climb past "." to the filesystem root.
func TestGateBSyncAncestorsRelative(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	if err := os.MkdirAll(filepath.Join("rel", "host", "run"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syncAncestors(filepath.Join("rel", "host", "run")); err != nil {
		t.Fatalf("relative ancestors: %v", err)
	}
	s := NewStore("rel")
	if err := appendDecisionErr(s); err != nil {
		t.Fatalf("append with relative store: %v", err)
	}
}
