package main

import (
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"os"
	"path/filepath"
	"testing"
)

func TestGateBR2SyntheticMarkerDirectory(t *testing.T) {
	d := seeded(t)
	p := filepath.Join(d, markerFile)
	os.Remove(p)
	os.Mkdir(p, 0700)
	if e := requireSyntheticStore(d, "run"); e == nil {
		t.Fatal("directory marker accepted")
	}
}
func TestGateBR2SyntheticSecondSeed(t *testing.T) {
	d := seeded(t)
	if e := seedSyntheticRouteApproval(d, "run", "U1", "d"); e != nil {
		t.Fatal(e)
	}
	if e := seedSyntheticRouteApproval(d, "run", "U1", "d"); e == nil {
		t.Fatal("second approval seeding accepted")
	}
}
func TestGateBR2SyntheticForeignSeed(t *testing.T) {
	d := t.TempDir()
	canonical, e := scratchDir(d)
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, markerFile), []byte(markerContent(canonical, "run")), 0600); e != nil {
		t.Fatal(e)
	}
	r := model.Record{Kind: model.KindCriterion, Version: 1, Metadata: model.RecordMetadata{RunID: "run", Sequence: 1, Timestamp: stamp(-60), Actor: syntheticActor}, Criteria: []model.Criterion{{ID: "SYNTHETIC-C1", Disposition: model.CriterionPassed}}}
	if e := store.New(d).Append(&r); e != nil {
		t.Fatal(e)
	}
	if e := seedSyntheticRouteApproval(d, "run", "U1", "d"); e == nil {
		t.Error("accepted foreign PASSED seed although seedCore only creates PENDING")
	}
}
