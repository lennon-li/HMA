package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// A01: a report that breaks the report contract cannot approve or clear a
// blocker, and the gate stays closed.
func TestGateCR3ContractInvalidPassCannotApproveOrClear(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"empty-requirements": func(r map[string]any) { r["requirements"] = []any{} },
		"unfrozen-phase1":    func(r map[string]any) { r["phase1_frozen"] = false },
		"no-target":          func(r map[string]any) { delete(r, "target") },
		"no-evidence-refs": func(r map[string]any) {
			r["requirements"] = []any{map[string]any{"id": "R1", "status": "pass", "evidence_refs": []any{}}}
		},
		"invalid-severity": withFindings(map[string]any{"id": "F9", "kind": "opportunity", "severity": "urgent"}),
	} {
		t.Run(name, func(t *testing.T) {
			f := newReviewFixture(t, hoststate.IssuePerChange, false)
			f.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "F1", Disposition: model.FindingBlock}}})
			// Re-generate the packet on the new core head.
			f.packet("2026-09-16T12:05:10Z", "2026-09-16T12:05:20Z", "2026-09-16T12:10:00Z", "run-"+name)
			rp := f.report("pass", "isolated", edit)
			f.receipt(rp, nil)
			if err := f.importReport(rp); err != nil {
				t.Fatal(err)
			}
			h, _ := hoststate.NewStore(f.store).Load("run")
			if v := h[len(h)-1].Review.Verdict; v != hoststate.ReviewEscalate {
				t.Fatalf("contract-invalid pass imported as %s", v)
			}
			if err := f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z"); err != nil {
				t.Fatal(err)
			}
			if got := f.coreFindings(); got["F1"] != model.FindingBlock {
				t.Fatalf("blocker cleared by contract-invalid report: %v", got)
			}
			if f.gate() == nil {
				t.Fatal("gate passed")
			}
		})
	}
}

// A02: the live target must still equal the packet at both imports.
func TestGateCR3FindingsImportRequiresLiveTarget(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	rp := f.report("pass", "isolated", nil)
	f.receipt(rp, nil)
	if err := f.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "a"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := f.importFindings(f.latestReview(), rp, "2026-09-16T12:06:30Z"); err == nil || err.Error() != hoststate.ReasonReviewTargetChanged {
		t.Fatalf("findings import after target change: %v", err)
	}
}

// A05: a legacy unowned finding written by a unit-scoped record belongs to
// that unit, so its approving review clears it; a legacy finding with no
// unit stays in force.
func TestGateCR3LegacyFindingOwnership(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	f.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "LEGACY-U1", Disposition: model.FindingBlock}}})
	f.appendCore(model.Record{Kind: model.KindFinding, Findings: []model.Finding{
		{ID: "LEGACY-U1", Disposition: model.FindingBlock}, {ID: "LEGACY-RUN", Disposition: model.FindingBlock}}})
	core, _ := store.New(f.store).Load("run")
	cur := hoststate.AssignLegacyOwners(core, engine.ProjectResolutionState(core).Findings)
	owners := map[string]string{}
	for _, x := range cur {
		owners[x.ID] = x.UnitID
	}
	// The last record that set both names no unit: both stay unowned.
	if owners["LEGACY-U1"] != "" || owners["LEGACY-RUN"] != "" {
		t.Fatalf("ambiguous findings given an owner: %v", owners)
	}

	g := newReviewFixture(t, hoststate.IssuePerChange, false)
	g.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "LEGACY-U1", Disposition: model.FindingBlock}}})
	g.packet("2026-09-16T12:05:10Z", "2026-09-16T12:05:20Z", "2026-09-16T12:10:00Z", "run-legacy")
	rp := g.report("pass", "isolated", nil)
	g.receipt(rp, nil)
	if err := g.importReport(rp); err != nil {
		t.Fatal(err)
	}
	if err := g.importFindings(g.latestReview(), rp, "2026-09-16T12:06:30Z"); err != nil {
		t.Fatal(err)
	}
	if got := g.coreFindings(); len(got) != 0 {
		t.Fatalf("unit-owned legacy blocker not cleared by approving review: %v", got)
	}
}

// A04: the packet carries the bound plan digest, criteria, the unit's
// current findings, and the core head; a missing plan digest refuses.
func TestGateCR3PacketCarriesCoreState(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	f.appendCore(model.Record{Kind: model.KindFinding, UnitID: "U1", Findings: []model.Finding{{ID: "SENTINEL-BLOCKER", Disposition: model.FindingBlock}}})
	f.packet("2026-09-16T12:05:10Z", "2026-09-16T12:05:20Z", "2026-09-16T12:10:00Z", "run-state")
	b, _ := os.ReadFile(f.packetPath)
	core, _ := store.New(f.store).Load("run")
	for _, want := range []string{"SENTINEL-BLOCKER", `"accepted_plan_digest": "plan"`, core[len(core)-1].Metadata.HeadAnchor} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("packet lacks %q", want)
		}
	}
}

// A08: a terminal run refuses before the check runs.
func TestGateCR3LoopEvidenceTerminalRunDoesNotExecute(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	bTerminal(t, d, model.OutcomeAborted)
	marker := filepath.Join(t.TempDir(), "ran")
	if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"touch", marker}, "green", 0)); err == nil || err.Error() != hoststate.ReasonCoreRunTerminal {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("check executed on a terminal run")
	}
}

// A07: envelope digest, store, hook, escalate_to, and review-run reuse.
func TestGateCR3LoopEnvelopeChecks(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	for name, edit := range map[string]func(env, st map[string]any){
		"wrong-digest":        func(_, st map[string]any) { st["envelope_sha256"] = strings.Repeat("0", 64) },
		"edited-after-digest": func(env, _ map[string]any) { env["verify"] = map[string]any{"argv": []any{"true", "x"}} },
		"other-store": func(env, st map[string]any) {
			rehash(t, env, st, func() { env["hma"].(map[string]any)["store"] = t.TempDir() })
		},
		"no-hook":  func(env, st map[string]any) { rehash(t, env, st, func() { delete(env, "authority_hook") }) },
		"escalate": func(env, st map[string]any) { rehash(t, env, st, func() { env["escalate_to"] = "jax" }) },
	} {
		if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"true"}, "green", 0, edit)); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	// Only the repository binding.
	if h, _ := hoststate.NewStore(d).Load("run"); len(h) != 1 {
		t.Fatalf("refused imports wrote %d host records", len(h)-1)
	}
}

func rehash(t *testing.T, env, st map[string]any, change func()) {
	t.Helper()
	change()
	b, _ := jsonMarshal(env)
	sum, err := hoststate.PythonCanonicalSHA256(b)
	if err != nil {
		t.Fatal(err)
	}
	st["envelope_sha256"] = sum
}

// A09: results and host show label the review enforcement mode.
func TestGateCR3EnforcementModeVisible(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	if m := hoststate.ReviewEnforcementMode(h); m != hoststate.EnforcementPacketImport {
		t.Fatalf("mode after packet: %s", m)
	}
	if m := hoststate.ReviewEnforcementMode(h[:3]); m != hoststate.EnforcementLegacyManual {
		t.Fatalf("mode before packet: %s", m)
	}
	if err := run([]string{"host", "show", "--store", f.store, "--run_id", "run", "--mode"}); err != nil {
		t.Fatal(err)
	}
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
