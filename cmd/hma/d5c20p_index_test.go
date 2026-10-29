//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAudit20IndexMetadataSwapRestore(t *testing.T) {
	cf := newCreditFixture(t, "true")
	p := filepath.Join(cf.root, "a")
	original, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	auditWrite(t, p, "staged different bytes\n")
	a20git(t, cf.root, "add", "a")
	stagedOID := a20git(t, cf.root, "rev-parse", ":a")
	auditWrite(t, p, string(original))
	before, e := captureTarget(context.Background(), cf.root, cf.base, true)
	if e != nil {
		t.Fatal(e)
	}
	stable, e := buildCapturedManifest(context.Background(), cf.root, cf.base, true, &before)
	if e != nil {
		t.Fatal(e)
	}
	if len(stable) != 1 || !stable[0].Staged {
		t.Fatalf("stable staged control: %+v", stable)
	}
	in, out := d5c5pPacketInput(t, cf, "", "Audit staged metadata binding")
	armed, restored := false, false
	reviewPacketBeforeManifestHook = func() { a20git(t, cf.root, "reset", "-q", "HEAD", "--", "a"); armed = true }
	worktreeWalkHook = func(rel string) {
		if armed && rel == "a" {
			a20git(t, cf.root, "update-index", "--cacheinfo", "100644", stagedOID, "a")
			armed = false
			restored = true
		}
	}
	defer func() { reviewPacketBeforeManifestHook = nil; worktreeWalkHook = nil }()
	e = run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", out})
	if e != nil {
		t.Logf("packet safely refused: %v", e)
		return
	}
	if !restored {
		t.Fatal("scheduled index restoration did not run")
	}
	after, e := captureTarget(context.Background(), cf.root, cf.base, true)
	if e != nil {
		t.Fatal(e)
	}
	st := packetState(t, out)
	t.Logf("stable=%+v accepted=%+v before=%+v after=%+v equal=%v restored=%v", stable, st.Manifest, before.State, after.State, before.State == after.State, restored)
	if before.State == after.State && len(st.Manifest) == 1 && !st.Manifest[0].Staged {
		t.Error("accepted manifest staged=false for the captured staged index; snapshots are equal")
	}
}
