package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
)

// packetState reads the "Bound HMA state" JSON out of the generated packet.
func packetState(t *testing.T, path string) packetCoreState {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)Bound HMA state[^\n]*\n(\{.*?\n\})\n\nTask:`).FindSubmatch(b)
	if m == nil {
		t.Fatalf("no bound state in packet:\n%s", b)
	}
	var st packetCoreState
	if err := json.Unmarshal(m[1], &st); err != nil {
		t.Fatal(err)
	}
	return st
}

// D5c (architecture §16): the review packet carries the fixed validator
// packet: approved criteria and non-goals, the plan unit and its budgets, the
// evidence bundle and the policy. The reviewer sees what was approved, not
// only the caller's task text.
func TestReviewPacketCarriesContractAndUnit(t *testing.T) {
	cf := newCreditFixture(t, "true")
	st := packetState(t, cf.packetPath)
	if st.Contract == nil || st.Unit == nil {
		t.Fatalf("packet lacks the contract or the unit: %+v", st)
	}
	if st.Contract.Intent != "one behavior" || len(st.Contract.NonGoals) != 1 || st.Contract.NonGoals[0] != "none" {
		t.Errorf("contract intent or non-goals missing: %+v", st.Contract)
	}
	if len(st.Contract.Criteria) != 2 || st.Contract.Criteria[0].ID != "C1" || st.Contract.Criteria[0].Threshold != "exit 0" {
		t.Errorf("criteria missing: %+v", st.Contract.Criteria)
	}
	if st.Unit.ID != "U1" || st.Unit.Budgets.MaxFiles != 3 || st.Unit.Budgets.MaxDiffLines != 100 {
		t.Errorf("unit budgets missing: %+v", st.Unit)
	}
	if len(st.Unit.AllowedFiles) != 1 || st.Unit.AllowedFiles[0] != "a" || len(st.Unit.ForbiddenFiles) != 1 || len(st.Unit.RequiredEvidence) != 1 {
		t.Errorf("unit scope or required evidence missing: %+v", st.Unit)
	}
	if st.PolicyDigest == "" || st.Contract.Digest == "" {
		t.Errorf("policy or contract digest missing: %+v", st)
	}
	if st.RequestBasis != "contract.intent" || st.Request != st.Contract.Intent {
		t.Errorf("request basis missing: basis=%q request=%q", st.RequestBasis, st.Request)
	}
	if len(st.Unit.VerificationCommands) == 0 || st.Unit.ImplementerRoute == "" || st.Unit.ReuseAnalysis == "" || st.Unit.RollbackConditions == nil || st.Unit.EscalationConditions == nil || st.Unit.Dependencies == nil {
		t.Errorf("full plan unit fields missing: %+v", st.Unit)
	}
	if st.Policy == nil {
		t.Error("normalized policy missing")
	}
	if len(st.Manifest) == 0 {
		t.Error("changed-file manifest missing")
	}

}

// The packet binds the contract: a different contract, no contract for a run
// that has one, and a contract for a run that has none are all refused before
// any packet exists.
func TestReviewPacketRefusesWrongContract(t *testing.T) {
	cf := newCreditFixture(t, "true")
	b, err := os.ReadFile(cf.contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var c model.AcceptanceContract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	c.Intent = "a different behavior"
	other, _ := json.Marshal(c)
	otherPath := filepath.Join(t.TempDir(), "other.json")
	os.WriteFile(otherPath, other, 0o600)

	packet := func(contract string) error {
		rpf := cf.step("preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:30:00Z"), Preflight: hoststate.RoutePreflight{
			UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T13:30:00Z",
		}})
		in := filepath.Join(t.TempDir(), "packet.json")
		writeJSON(t, in, hostReviewPacketInput{hostEnvelope: fxEnv("2026-09-16T12:31:00Z"), Dispatch: hoststate.Dispatch{
			UnitID: "U1", Role: hoststate.RoleReview, Route: cf.reviewer, PreflightRecordDigest: rpf,
			BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: cf.implementation,
			ArtifactDigest: hoststate.ReviewArtifactDigest(cf.head, cf.diff, ""), Risk: model.RouteRiskHigh,
		}, DispatchRunID: "20261016T000000Z-ming-review-9", IssuingReason: cf.reason, RepositoryRoot: cf.root, BaseRevision: cf.base,
			Task: "Review the change.", ContractPath: contract, LedgerDir: cf.ledger})
		return run([]string{"host", "review-packet", "--input", in, "--store", cf.store, "--out", filepath.Join(t.TempDir(), "p.md")})
	}
	if err := packet(otherPath); err == nil || err.Error() != string(engine.ReasonContractMismatch) {
		t.Errorf("wrong contract reason = %v", err)
	}
	if err := packet(""); err == nil || !strings.Contains(err.Error(), "PACKET_INPUT_MISSING") {
		t.Errorf("a packet was generated with no contract for a run that binds one: %v", err)
	}
}
