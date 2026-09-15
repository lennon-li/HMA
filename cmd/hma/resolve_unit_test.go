package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

func TestResolveUnitBinding(t *testing.T) {
	for _, kind := range []string{"waiver_operation", "finding_override"} {
		for _, unit := range []string{"", "approved-unit"} {
			t.Run(kind+"/"+unit, func(t *testing.T) {
				dir := t.TempDir()
				s := seedCriterion(t, dir, "run")
				initial, err := s.Load("run")
				if err != nil {
					t.Fatal(err)
				}
				seed := model.Record{Kind: model.KindFinding, Version: 1,
					Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, Timestamp: "2026-09-03T00:00:00Z", Actor: "host"},
					Findings: []model.Finding{{ID: "F1", Disposition: model.FindingWaivable}}}
				seed.Metadata.PredecessorHash = initial[0].Metadata.HeadAnchor
				if err := s.Append(&seed); err != nil {
					t.Fatal(err)
				}
				in := waiverInput("run", "nonce-1")
				payload := in["waiver_operation"].(map[string]any)
				if kind == "finding_override" {
					delete(in, "waiver_operation")
					delete(payload, "operation")
					delete(payload, "scope_selector")
					payload["finding_id"] = "F1"
					payload["disposition"] = string(model.FindingAdvisory)
					in[kind] = payload
				}
				if unit != "" {
					payload["unit_id"] = unit
				}
				input := filepath.Join(dir, "resolution.json")
				in["unit_id"] = "victim-unit"
				writeJSON(t, input, in)
				if err := runResolve([]string{"--input", input, "--store", dir}); err == nil || !strings.Contains(err.Error(), `unknown field "unit_id"`) {
					t.Fatalf("top-level unit accepted: %v", err)
				}
				records, err := s.Load("run")
				if err != nil || len(records) != 2 {
					t.Fatalf("rejected input changed store: %d, %v", len(records), err)
				}
				delete(in, "unit_id")
				writeJSON(t, input, in)
				if err := runResolve([]string{"--input", input, "--store", dir}); err != nil {
					t.Fatal(err)
				}
				records, err = s.Load("run")
				if err != nil || len(records) != 3 {
					t.Fatalf("records: %d, %v", len(records), err)
				}
				r := records[2]
				if r.UnitID != unit {
					t.Fatalf("record unit = %q, want %q", r.UnitID, unit)
				}
				if r.WaiverOperation != nil && r.WaiverOperation.UnitID != unit || r.FindingOverride != nil && r.FindingOverride.UnitID != unit {
					t.Fatal("payload unit not persisted")
				}
				r.UnitID = "victim-unit"
				if err := model.ValidateRecord(&r); err == nil || !strings.Contains(err.Error(), "unit_id differs") {
					t.Fatalf("mismatched unit validation: %v", err)
				}
				if model.HumanApprovedStateChange(r) {
					t.Fatal("mismatched unit treated as human approved")
				}
			})
		}
	}
}
