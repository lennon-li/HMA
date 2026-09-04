package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// evalFixtureCase reads one case out of an existing engine fixture. It covers
// both fixture shapes in testdata: a nested "expect" object and the flat
// "expect_decision"/"expect_reason" pair. Decoding is deliberately lenient
// here — the engine's own tests already hold the fixtures to strict decoding,
// and this test only needs each case's request and its expected verdict. The
// request itself is handed to the CLI verbatim, so the CLI's strict decoder
// is exercised on real fixture input.
type evalFixtureCase struct {
	Name    string          `json:"name"`
	Request json.RawMessage `json:"request"`
	Expect  struct {
		Decision engine.Decision `json:"decision"`
		Reason   engine.Reason   `json:"reason"`
	} `json:"expect"`
	ExpectDecision engine.Decision `json:"expect_decision"`
	ExpectReason   engine.Reason   `json:"expect_reason"`
}

func (c evalFixtureCase) expected() (engine.Decision, engine.Reason) {
	if c.ExpectDecision != "" {
		return c.ExpectDecision, c.ExpectReason
	}
	return c.Expect.Decision, c.Expect.Reason
}

func readFixture(t *testing.T, path string, into any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}
}

func loadEvalCases(t *testing.T, path string) []evalFixtureCase {
	t.Helper()
	var cases []evalFixtureCase
	readFixture(t, path, &cases)
	if len(cases) == 0 {
		t.Fatalf("fixture %s is empty", path)
	}
	return cases
}

// loadInvalidationCases takes the rewind half of the invalidation fixture.
// The waiver_changes half drives EvaluateWaiverChange, which this task does
// not expose on the CLI.
func loadInvalidationCases(t *testing.T) []evalFixtureCase {
	t.Helper()
	var fixture struct {
		Rewinds []evalFixtureCase `json:"rewinds"`
	}
	readFixture(t, "../../testdata/engine/invalidation.json", &fixture)
	if len(fixture.Rewinds) == 0 {
		t.Fatal("invalidation fixture declares no rewinds")
	}
	return fixture.Rewinds
}

// fillRouteCoherencePolicyDigest mirrors what the engine's own fixture test
// does: most route coherence cases leave policy_digest empty and expect it to
// be the digest of the policy they carry. The CLI must not fill that in
// itself — it would be deciding something for the host — so the test supplies
// it, exactly as the engine test does.
func fillRouteCoherencePolicyDigest(t *testing.T, raw json.RawMessage) json.RawMessage {
	t.Helper()
	var req engine.RouteCoherenceRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("decode route coherence request: %v", err)
	}
	if req.PolicyDigest == "" {
		req.PolicyDigest = engine.EvaluateRoutePolicy(engine.RoutePolicyRequest{Policy: req.Policy}).Digest
	}
	filled, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("re-encode route coherence request: %v", err)
	}
	return filled
}

// seedCriteria commits one criterion record carrying several criteria, so a
// projection can hold more than one active waiver scope.
func seedCriteria(t *testing.T, dir, runID string, ids ...string) {
	t.Helper()
	criteria := make([]model.Criterion, 0, len(ids))
	for _, id := range ids {
		criteria = append(criteria, model.Criterion{ID: id, Disposition: model.CriterionPending})
	}
	rec := model.Record{
		Kind:     model.KindCriterion,
		Version:  1,
		Metadata: model.RecordMetadata{RunID: runID, Sequence: 1, Timestamp: "2026-09-03T00:00:00Z", Actor: "host"},
		Criteria: criteria,
	}
	if err := store.New(dir).Append(&rec); err != nil {
		t.Fatal(err)
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns what
// the command printed along with the error it returned.
func captureStdout(t *testing.T, fn func() error) ([]byte, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdout")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = f
	runErr := fn()
	os.Stdout = saved
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return out, runErr
}

// decodeSingleDocument enforces the output contract: exactly one JSON
// document on stdout per invocation, and nothing after it. The second decode
// must reach io.EOF, not merely fail: any other error means stdout carried
// trailing bytes, which is as much a contract violation as a second document.
func decodeSingleDocument(t *testing.T, out []byte) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(out))
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v (%q)", err, out)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			t.Fatalf("stdout carried a second document: %q", out)
		}
		t.Fatalf("stdout carried trailing bytes after its document: %v (%q)", err, out)
	}
	return doc
}

// TestEvalFixtures drives every CLI evaluator through the engine fixtures it
// shares with the library tests, proving the shipped binary reaches the same
// verdict on the same input.
func TestEvalFixtures(t *testing.T) {
	table := []struct {
		evaluator string
		cases     []evalFixtureCase
		prepare   func(*testing.T, json.RawMessage) json.RawMessage
	}{
		{evaluator: "transition", cases: loadEvalCases(t, "../../testdata/engine/transitions.json")},
		{evaluator: "invalidation", cases: loadInvalidationCases(t)},
		{evaluator: "validator-contract", cases: loadEvalCases(t, "../../testdata/validator-contract/valid.json")},
		{evaluator: "validator-contract", cases: loadEvalCases(t, "../../testdata/validator-contract/invalid.json")},
		{evaluator: "route-verification", cases: loadEvalCases(t, "../../testdata/engine/route-verification.json")},
		{evaluator: "route-coherence", cases: loadEvalCases(t, "../../testdata/engine/route-coherence.json"), prepare: fillRouteCoherencePolicyDigest},
		{evaluator: "route-policy", cases: loadEvalCases(t, "../../testdata/engine/route-policy.json")},
	}

	for _, group := range table {
		for _, tc := range group.cases {
			t.Run(group.evaluator+"/"+tc.Name, func(t *testing.T) {
				request := tc.Request
				if group.prepare != nil {
					request = group.prepare(t, request)
				}
				input := filepath.Join(t.TempDir(), "input.json")
				if err := os.WriteFile(input, request, 0600); err != nil {
					t.Fatal(err)
				}

				out, err := captureStdout(t, func() error {
					return runEval([]string{group.evaluator, "--input", input})
				})
				doc := decodeSingleDocument(t, out)

				wantDecision, wantReason := tc.expected()
				if got := doc["decision"]; got != string(wantDecision) {
					t.Fatalf("decision = %v, want %s", got, wantDecision)
				}
				gotReason, _ := doc["reason"].(string)
				if gotReason != string(wantReason) {
					t.Fatalf("reason = %q, want %q", gotReason, wantReason)
				}

				// Zero exit only on LEGAL_PENDING_HUMAN_APPROVAL; any
				// other decision, REJECTED included, exits non-zero.
				if wantDecision == engine.DecisionRejected {
					if err == nil {
						t.Fatal("a rejection exited zero")
					}
					if err.Error() != string(wantReason) {
						t.Fatalf("exit reason = %q, want %q", err, wantReason)
					}
				} else if err != nil {
					t.Fatalf("a legal result exited non-zero: %v", err)
				}

				// No evaluator result may claim the machine advanced.
				if advanced, present := doc["machine_advanced"]; present && advanced != false {
					t.Fatalf("machine_advanced = %v", advanced)
				}
			})
		}
	}
}

// snapshotDir records every entry in dir and the bytes of every regular file
// under it, so a comparison catches an in-place modification or a
// rename-and-replace, not only a change in the number of entries.
func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			snapshot[entry.Name()+"/"] = ""
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		snapshot[entry.Name()] = string(content)
	}
	return snapshot
}

// TestEvalWritesNothing is the regression test for decision 1: eval is
// strictly read-only. It takes no store flag at all, and an invocation must
// leave the filesystem exactly as it found it — the same entries, with the
// same bytes.
func TestEvalWritesNothing(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	writeJSON(t, input, map[string]any{"source": "GROUNDING", "target_stage": "ACCEPTANCE_CRITERIA"})

	before := snapshotDir(t, dir)
	if _, err := captureStdout(t, func() error {
		return runEval([]string{"transition", "--input", input})
	}); err != nil {
		t.Fatal(err)
	}
	if after := snapshotDir(t, dir); !reflect.DeepEqual(after, before) {
		t.Fatalf("eval changed the filesystem:\nbefore: %v\n after: %v", before, after)
	}
}

func TestEvalRejectsUnknownFieldsAndTrailingDocuments(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")

	writeJSON(t, input, map[string]any{"source": "GROUNDING", "target_stage": "PLANNING", "approve": true})
	if _, err := captureStdout(t, func() error {
		return runEval([]string{"transition", "--input", input})
	}); err == nil {
		t.Fatal("an unknown input field was accepted")
	}

	trailing := `{"source":"GROUNDING","target_stage":"PLANNING"}
{"source":"PLANNING","target_stage":"ROUTE_SELECTION"}`
	if err := os.WriteFile(input, []byte(trailing), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := captureStdout(t, func() error {
		return runEval([]string{"transition", "--input", input})
	}); err == nil {
		t.Fatal("a trailing input document was accepted")
	}
}

func TestEvalRejectsUnusableInvocations(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.json")
	writeJSON(t, input, map[string]any{"source": "GROUNDING", "target_stage": "PLANNING"})

	for _, args := range [][]string{
		{},
		{"not-an-evaluator", "--input", input},
		{"transition"},
		{"transition", "--input", input, "extra"},
		{"transition", "--input", filepath.Join(t.TempDir(), "absent.json")},
	} {
		if _, err := captureStdout(t, func() error { return runEval(args) }); err == nil {
			t.Fatalf("accepted eval %v", args)
		}
	}
}

func TestRunDispatchesEvalAndShow(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	writeJSON(t, input, map[string]any{"source": "GROUNDING", "target_stage": "ACCEPTANCE_CRITERIA"})
	if _, err := captureStdout(t, func() error {
		return run([]string{"eval", "transition", "--input", input})
	}); err != nil {
		t.Fatalf("hma eval transition: %v", err)
	}

	seedCriterion(t, dir, "run")
	if _, err := captureStdout(t, func() error {
		return run([]string{"show", "--store", dir, "--run_id", "run"})
	}); err != nil {
		t.Fatalf("hma show: %v", err)
	}
}

// TestShowProjectsTheReplayedChain proves hma show reports what the chain
// already says — a granted waiver and its spent nonce — and writes nothing.
func TestShowProjectsTheReplayedChain(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(dir, "waiver.json")
	writeJSON(t, input, waiverInput("run", "nonce-1"))
	if _, err := captureStdout(t, func() error {
		return runResolve([]string{"--input", input, "--store", dir})
	}); err != nil {
		t.Fatal(err)
	}

	before, err := store.New(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		return runShow([]string{"--store", dir, "--run_id", "run"})
	})
	if err != nil {
		t.Fatal(err)
	}

	var doc showDocument
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("show output is not a projection document: %v (%q)", err, out)
	}
	if len(doc.Criteria) != 1 || doc.Criteria[0].ID != "C1" {
		t.Fatalf("criteria = %+v", doc.Criteria)
	}
	want := model.ScopeSelector{Kind: model.ScopeCriterion, Target: "C1"}
	if len(doc.ActiveWaivers) != 1 || doc.ActiveWaivers[0] != want {
		t.Fatalf("active waivers = %+v, want [%+v]", doc.ActiveWaivers, want)
	}
	if len(doc.UsedNonces) != 1 || doc.UsedNonces[0] != "nonce-1" {
		t.Fatalf("used nonces = %v", doc.UsedNonces)
	}

	// show is read-only: the chain it projected is unchanged. The records
	// are compared in full, not merely counted — a mutation that preserves
	// the record count is exactly the kind this test exists to catch.
	after, err := store.New(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("show changed the chain:\nbefore: %+v\n after: %+v", before, after)
	}
}

// TestShowIsDeterministic proves the sorted rendering of the projection's
// maps produces byte-identical output across invocations.
func TestShowIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	seedCriteria(t, dir, "run", "C1", "C2")
	input := filepath.Join(dir, "waiver.json")
	for i, nonce := range []string{"nonce-a", "nonce-b"} {
		op := waiverInput("run", nonce)
		op["waiver_operation"].(map[string]any)["scope_selector"] = map[string]any{
			"kind": string(model.ScopeCriterion), "target": []string{"C1", "C2"}[i],
		}
		writeJSON(t, input, op)
		if _, err := captureStdout(t, func() error {
			return runResolve([]string{"--input", input, "--store", dir})
		}); err != nil {
			t.Fatalf("%s: %v", nonce, err)
		}
	}

	first, err := captureStdout(t, func() error {
		return runShow([]string{"--store", dir, "--run_id", "run"})
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		again, err := captureStdout(t, func() error {
			return runShow([]string{"--store", dir, "--run_id", "run"})
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(again) != string(first) {
			t.Fatalf("show output differs between runs:\n%q\n%q", first, again)
		}
	}
}

func TestShowRejectsUnusableInvocations(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{},
		{"--store", dir},
		{"--run_id", "run"},
		{"--store", dir, "--run_id", "run", "extra"},
	} {
		if _, err := captureStdout(t, func() error { return runShow(args) }); err == nil {
			t.Fatalf("accepted show %v", args)
		}
	}
}
