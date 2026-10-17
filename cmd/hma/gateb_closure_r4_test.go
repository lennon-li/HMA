package main

import (
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// R4-01: the core run lock stays held until the host record is durable, so a
// core contender that gets the lock always sees the committed host record.
func TestGateBClosureR4CoreLockCoversHostCommit(t *testing.T) {
	d := t.TempDir()
	seedRouteAuthorization(t, d, "run", "U1", "approved-route")
	before, err := hoststate.NewStore(d).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "in")
	writeJSON(t, p, bDecision(t, d, "impl"))
	seen := make(chan int, 1)
	hostAppendHook = func() {
		hostAppendHook = nil
		go func() {
			_ = store.New(d).WithRunLock("run", func([]model.Record) error {
				// Read the record files directly: Load takes the host lock,
				// which would hide the window.
				rs, _ := filepath.Glob(filepath.Join(d, "host", "run", "*.json"))
				seen <- len(rs)
				return nil
			})
		}()
	}
	t.Cleanup(func() { hostAppendHook = nil })
	if err := runHostDecision([]string{"--input", p, "--store", d}); err != nil {
		t.Fatal(err)
	}
	if got := <-seen; got != len(before)+1 {
		t.Fatalf("core contender saw %d host records, want %d: core lock released before host commit", got, len(before)+1)
	}
}
