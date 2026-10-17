package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func seeded(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := seedCore(dir, "run"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSyntheticStoreAcceptsSeededRun(t *testing.T) {
	dir := seeded(t)
	if err := requireSyntheticStore(dir, "run"); err != nil {
		t.Fatal(err)
	}
	if err := seedSyntheticRouteApproval(dir, "run", "U1", "d"); err != nil {
		t.Fatal(err)
	}
}

// M5: a marker copied from another store names that store, so it is refused.
func TestSyntheticStoreRefusesCopiedMarker(t *testing.T) {
	src := seeded(t)
	dst := t.TempDir()
	b, err := os.ReadFile(filepath.Join(src, markerFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, markerFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireSyntheticStore(dst, "run"); err == nil {
		t.Fatal("accepted a marker copied from another store")
	}
}

// M5: a symlinked marker is refused even when its target is valid.
func TestSyntheticStoreRefusesSymlinkedMarker(t *testing.T) {
	src := seeded(t)
	dst := t.TempDir()
	if err := os.Symlink(filepath.Join(src, markerFile), filepath.Join(dst, markerFile)); err != nil {
		t.Fatal(err)
	}
	if err := requireSyntheticStore(dst, "run"); err == nil {
		t.Fatal("accepted a symlinked marker")
	}
}

func TestSyntheticStoreRefusesOtherRun(t *testing.T) {
	dir := seeded(t)
	if err := requireSyntheticStore(dir, "other-run"); err == nil {
		t.Fatal("accepted a marker for a different run")
	}
}

func TestSyntheticStoreRefusesNonScratchDir(t *testing.T) {
	if _, err := scratchDir("/"); err == nil {
		t.Fatal("accepted / as a scratch store")
	}
	if wd, err := os.Getwd(); err == nil {
		if _, err := scratchDir(wd); err == nil {
			t.Fatalf("accepted the repository directory %s as a scratch store", wd)
		}
	}
}

// M5: seeding refuses a run that holds anything beyond the seed-core record.
func TestSeedRouteApprovalRefusesNonSeedRun(t *testing.T) {
	dir := seeded(t)
	s := store.New(dir)
	prior, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	extra := model.Record{Kind: model.KindCriterion, Version: 1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: prior[0].Metadata.HeadAnchor, Timestamp: stamp(-50), Actor: "someone"},
		Criteria: []model.Criterion{{ID: "OTHER", Disposition: model.CriterionPending}}}
	if err := s.Append(&extra); err != nil {
		t.Fatal(err)
	}
	if err := seedSyntheticRouteApproval(dir, "run", "U1", "d"); err == nil {
		t.Fatal("seeded an approval fixture into a run with extra records")
	}
	if err := seedSyntheticRouteApproval(seeded(t), "run", "U1", "d"); err != nil {
		t.Fatalf("control: %v", err)
	}
}

// B9: a scratch run whose only record differs from the seed in any field
// (here the disposition) is not the untouched seed run.
func TestSeedRouteApprovalRefusesAlteredSeed(t *testing.T) {
	dir := t.TempDir()
	if err := seedCore(dir, "run"); err != nil {
		t.Fatal(err)
	}
	alt := t.TempDir()
	canonical, err := scratchDir(alt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonical, markerFile), []byte(markerContent(canonical, "run")), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := seedRecord("run")
	rec.Criteria[0].Disposition = model.CriterionPassed
	if err := store.New(alt).Append(&rec); err != nil {
		t.Fatal(err)
	}
	if err := seedSyntheticRouteApproval(alt, "run", "U1", "d"); err == nil {
		t.Fatal("seeded an approval fixture into a run whose seed record was altered")
	}
	if err := seedSyntheticRouteApproval(dir, "run", "U1", "d"); err != nil {
		t.Fatalf("control: %v", err)
	}
	if err := seedSyntheticRouteApproval(dir, "run", "U1", "d"); err == nil {
		t.Fatal("seeded a second approval fixture")
	}
}
