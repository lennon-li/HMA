package store

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

// TestJSONLConcurrentAppend races two writers that each observe the same
// predecessor state and then append a structurally valid record. Without the
// per-run lock both rewrites land and the loser silently discards the winner's
// record; with it, exactly one commits and the other fails the sequence check.
func TestJSONLConcurrentAppend(t *testing.T) {
	dir := t.TempDir()

	// Both writers must observe the same predecessor state before either
	// commits; otherwise the second simply loads the first's record and
	// legitimately appends after it, which is not the race under test.
	var loaded, wg sync.WaitGroup
	loaded.Add(2)
	wg.Add(2)
	var errMutex sync.Mutex
	var errs []error

	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			s := New(dir)
			existing, err := s.Load("run-concurrent")
			loaded.Done()
			loaded.Wait()
			if err == nil {
				prev := ""
				if len(existing) > 0 {
					prev = existing[len(existing)-1].Metadata.HeadAnchor
				}
				r := record("run-concurrent", int64(len(existing)+1), prev)
				err = s.Append(&r)
			}
			if err != nil {
				errMutex.Lock()
				errs = append(errs, err)
				errMutex.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(errs) != 1 {
		t.Fatalf("expected exactly one writer to be rejected, got %d: %v", len(errs), errs)
	}
	s := New(dir)
	history, err := s.Load("run-concurrent")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("expected exactly one committed record, got %d", len(history))
	}
}

// TestJSONLConcurrentAppendLosesNoRecord is the regression test for silent
// clobbering: every writer that reports success must be present in the chain.
func TestJSONLConcurrentAppendLosesNoRecord(t *testing.T) {
	dir := t.TempDir()
	const writers = 8

	var wg sync.WaitGroup
	var mu sync.Mutex
	committed := 0

	wg.Add(writers)
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			s := New(dir)
			for attempt := 0; attempt < 50; attempt++ {
				existing, err := s.Load("run-retry")
				if err != nil {
					continue
				}
				prev := ""
				if len(existing) > 0 {
					prev = existing[len(existing)-1].Metadata.HeadAnchor
				}
				r := record("run-retry", int64(len(existing)+1), prev)
				if err := s.Append(&r); err == nil {
					mu.Lock()
					committed++
					mu.Unlock()
					return
				}
			}
		}()
	}
	wg.Wait()

	history, err := New(dir).Load("run-retry")
	if err != nil {
		t.Fatal(err)
	}
	if committed != writers {
		t.Fatalf("expected %d writers to commit, got %d", writers, committed)
	}
	if len(history) != committed {
		t.Fatalf("chain holds %d records but %d writers reported success", len(history), committed)
	}
}

// TestJSONLRecoversInterruptedCommit simulates a crash after the stream landed
// but before the head anchor was promoted. The run must recover, not brick.
func TestJSONLRecoversInterruptedCommit(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	r1 := record("run", 1, "")
	if err := s.Append(&r1); err != nil {
		t.Fatal(err)
	}
	r2 := record("run", 2, r1.Metadata.HeadAnchor)
	if err := s.Append(&r2); err != nil {
		t.Fatal(err)
	}

	// Roll the head anchor back one record and restore the pending intent,
	// exactly as a crash between the two commit phases would leave it.
	if err := os.WriteFile(filepath.Join(dir, "run.head"), []byte(r1.Metadata.HeadAnchor+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.head.pending"), []byte(r2.Metadata.HeadAnchor+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := s.Load("run")
	if err != nil {
		t.Fatalf("interrupted commit not recovered: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("recovered %d records, want 2", len(got))
	}
	if _, err := os.Stat(filepath.Join(dir, "run.head.pending")); !os.IsNotExist(err) {
		t.Fatal("pending anchor survived recovery")
	}
	r3 := record("run", 3, r2.Metadata.HeadAnchor)
	if err := s.Append(&r3); err != nil {
		t.Fatalf("append after recovery: %v", err)
	}
}

// TestJSONLDiscardsUnlandedIntent covers the mirrored crash: the pending anchor
// was written but the stream never changed. The intent must be discarded, and
// it must not be mistaken for a valid anchor.
func TestJSONLDiscardsUnlandedIntent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	r1 := record("run", 1, "")
	if err := s.Append(&r1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.head.pending"), []byte("never-committed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("loaded %d records, want 1", len(got))
	}
	if _, err := os.Stat(filepath.Join(dir, "run.head.pending")); !os.IsNotExist(err) {
		t.Fatal("stale intent survived load")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "run.head"))
	if strings.TrimSpace(string(b)) != r1.Metadata.HeadAnchor {
		t.Fatal("stale intent was promoted to the head anchor")
	}
}

// TestJSONLTruncationStillDetected proves recovery did not weaken rollback
// detection: dropping a committed record is still rejected.
func TestJSONLTruncationStillDetected(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	r1 := record("run", 1, "")
	if err := s.Append(&r1); err != nil {
		t.Fatal(err)
	}
	r2 := record("run", 2, r1.Metadata.HeadAnchor)
	if err := s.Append(&r2); err != nil {
		t.Fatal(err)
	}
	stream := filepath.Join(dir, "run.jsonl")
	b, _ := os.ReadFile(stream)
	first := bytes.SplitAfter(b, []byte("\n"))[0]
	if err := os.WriteFile(stream, first, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("run"); err == nil {
		t.Fatal("truncated chain accepted")
	}
}

func record(run string, seq int64, prev string) model.Record {
	return model.Record{Kind: model.KindEvidence, Version: 1, Metadata: model.RecordMetadata{RunID: run, Sequence: seq, PredecessorHash: prev, Timestamp: "2026-08-31T12:00:00Z", Actor: "host"}, Stage: model.StageVerification}
}

func TestJSONLChain(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	r1 := record("run", 1, "")
	if err := s.Append(&r1); err != nil {
		t.Fatal(err)
	}
	r2 := record("run", 2, r1.Metadata.HeadAnchor)
	if err := s.Append(&r2); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("run")
	if err != nil || len(got) != 2 {
		t.Fatalf("load = %d, %v", len(got), err)
	}

	bad := record("run", 4, r2.Metadata.HeadAnchor)
	if err := s.Append(&bad); err == nil {
		t.Fatal("sequence gap accepted")
	}
	bad = record("run", 3, "wrong")
	if err := s.Append(&bad); err == nil {
		t.Fatal("wrong predecessor accepted")
	}

	stream := filepath.Join(dir, "run.jsonl")
	b, _ := os.ReadFile(stream)
	if err := os.WriteFile(stream, append(b, []byte("{\"torn\"")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("run"); err == nil {
		t.Fatal("torn line accepted")
	}
}

func TestJSONLDetectsHistoricalRewrite(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	r := record("run", 1, "")
	if err := s.Append(&r); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "run.jsonl")
	b, _ := os.ReadFile(p)
	b = []byte(strings.Replace(string(b), "host", "evil", 1))
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("run"); err == nil {
		t.Fatal("rewrite accepted")
	}
}
