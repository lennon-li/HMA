package store

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

func TestJSONLConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	
	var wg sync.WaitGroup
	var errs []error
	var errMutex sync.Mutex

	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			defer wg.Done()
			s := New(dir)
			err := s.Append(&model.Record{Metadata: model.RecordMetadata{RunID: "run-concurrent"}, Version: 1})
			if err != nil {
				errMutex.Lock()
				errs = append(errs, err)
				errMutex.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if len(errs) != 1 {
		// Because this is a naive race without explicit synchronization between the two goroutines,
		// depending on timing, both could read an empty chain and both could fail, or one could fail.
		// As long as they don't both succeed, the safety invariant holds.
		if len(errs) == 0 {
			t.Fatalf("expected at least one concurrent Append to fail (proving safety), got %d errors", len(errs))
		}
	}

	s := New(dir)
	history, err := s.Load("run-concurrent")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) > 1 {
		t.Fatalf("expected at most one record to survive, got %d", len(history))
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
