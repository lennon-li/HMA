package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func writeLoopState(t *testing.T, storeDir, runID, unitID string, argv []string, status string, loopExit int, edit ...func(env, st map[string]any)) string {
	t.Helper()
	env := map[string]any{"hma": map[string]any{"store": storeDir, "run_id": runID, "unit_id": unitID}, "verify": map[string]any{"argv": argv},
		"authority_hook": map[string]any{"argv": []any{"hma-hook"}}, "escalate_to": "stop"}
	st := map[string]any{"run_id": "vl-1", "status": status, "envelope": env,
		"attempts": []any{map[string]any{"n": 1, "dispatch_run_id": "d-1", "verify": map[string]any{"exit_code": loopExit, "signature": "sig"}}}}
	envJSON, _ := json.Marshal(env)
	sum, err := hoststate.PythonCanonicalSHA256(envJSON)
	if err != nil {
		t.Fatal(err)
	}
	st["envelope_sha256"] = sum
	for _, e := range edit {
		e(env, st)
	}
	p := filepath.Join(t.TempDir(), "state.json")
	b, _ := json.Marshal(st)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func runLoopEvidence(t *testing.T, storeDir, root, base, statePath string) error {
	t.Helper()
	// verify-loop records the repository root it ran in.
	b, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var st map[string]any
	if err := json.Unmarshal(b, &st); err != nil {
		t.Fatal(err)
	}
	if _, ok := st["repo"]; !ok {
		st["repo"] = map[string]any{"root": root}
		b, _ = json.Marshal(st)
		if err := os.WriteFile(statePath, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	input := filepath.Join(t.TempDir(), "loop.json")
	writeJSON(t, input, hostLoopEvidenceInput{hostEnvelope: fxEnv("2026-09-16T12:03:00Z"), UnitID: "U1", StatePath: statePath,
		RepositoryRoot: root, RepositoryIdentity: "repo", BaseRevision: base})
	return run([]string{"host", "loop-evidence", "--input", input, "--store", storeDir})
}

// The loop's own "green" never counts: HMA re-runs the check. A loop that
// claims success while the check fails is recorded with HMA's failing exit
// code, and no import credits a criterion.
func TestGateCLoopEvidenceRecapturesCheck(t *testing.T) {
	for _, tc := range []struct {
		name     string
		argv     []string
		wantExit int
	}{{"check-passes", []string{"true"}, 0}, {"loop-claims-green-check-fails", []string{"false"}, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			root, base := seedLoopRun(t, d)
			if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", tc.argv, "green", 0)); err != nil {
				t.Fatal(err)
			}
			h, _ := hoststate.NewStore(d).Load("run")
			l := h[len(h)-1].LoopEvidence
			if l == nil || l.RecaptureExitCode != tc.wantExit || len(l.Attempts) != 1 || l.Attempts[0].DispatchRunID != "d-1" {
				t.Fatalf("loop evidence %+v", l)
			}
			core, _ := store.New(d).Load("run")
			ev := core[len(core)-1]
			if ev.Kind != model.KindEvidence || ev.Metadata.HeadAnchor != l.RecaptureRecord || ev.Evidence.ExitCode != tc.wantExit || len(ev.Criteria) != 0 {
				t.Fatalf("core evidence %+v", ev)
			}
			for _, c := range engine.ProjectResolutionState(core).Criteria {
				if c.Disposition == model.CriterionPassed {
					t.Fatal("loop evidence credited a criterion")
				}
			}
		})
	}
}

func TestGateCLoopEvidenceRefusals(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	for name, path := range map[string]string{
		"other-run":  writeLoopState(t, d, "other", "U1", []string{"true"}, "green", 0),
		"other-unit": writeLoopState(t, d, "run", "U2", []string{"true"}, "green", 0),
		"no-argv":    writeLoopState(t, d, "run", "U1", nil, "green", 0),
	} {
		if err := runLoopEvidence(t, d, root, base, path); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	// A check that changes the worktree is refused.
	if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"touch", "new-file"}, "green", 0)); err == nil {
		t.Fatal("mutating check accepted")
	}
	// Only the repository binding.
	if h, _ := hoststate.NewStore(d).Load("run"); len(h) != 1 {
		t.Fatalf("refused loop imports wrote %d host records", len(h)-1)
	}
}
