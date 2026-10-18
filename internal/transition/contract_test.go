package transition

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func contractDoc(threshold string, ids ...string) (json.RawMessage, string) {
	c := model.AcceptanceContract{Schema: model.AcceptanceContractSchema, RunID: "run", Intent: "one behavior", NonGoals: []string{"none"},
		Risk: model.RouteRiskLow, Deliverables: []string{"code"}, PolicyDigest: "policy",
		Budgets:         model.ContractBudgets{MaxAttempts: 3, MaxWallMinutes: 60, CostCeiling: 10, CostUnit: "relative"},
		PermittedRoutes: []string{"r"}, PermissionEnvelope: []string{model.PermissionBoundedWrite}}
	for _, id := range ids {
		c.Criteria = append(c.Criteria, model.ContractCriterion{ID: id, Statement: "s", Threshold: threshold, EvidenceRequirement: "e",
			Check: model.Check{Executable: "go", Argv: []string{"test"}, WorkingDir: "."}})
	}
	raw, _ := json.Marshal(c)
	_, d, err := model.ParseAcceptanceContract(raw)
	if err != nil {
		panic(err)
	}
	return raw, d
}

// toAcceptance walks a run to ACCEPTANCE_CRITERIA and records its criteria.
func toAcceptance(t *testing.T, repo, base, dir string, now time.Time, ids ...string) {
	t.Helper()
	if _, err := Apply(context.Background(), approvedInput(t, repo, base, dir, model.StageGrounding, model.StageAcceptanceCriteria, "n-ac", now), dir); err != nil {
		t.Fatal(err)
	}
	s := store.New(dir)
	recs, _ := s.Load("run")
	var cs []model.Criterion
	for _, id := range ids {
		cs = append(cs, model.Criterion{ID: id, Disposition: model.CriterionPending})
	}
	r := model.Record{Kind: model.KindCriterion, Version: 1, Stage: model.StageAcceptanceCriteria, Criteria: cs,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: int64(len(recs) + 1), PredecessorHash: recs[len(recs)-1].Metadata.HeadAnchor,
			Timestamp: now.Format(time.RFC3339), Actor: "operator-confirmed:lennon"}}
	if err := s.Append(&r); err != nil {
		t.Fatal(err)
	}
}

func withContract(in Input, raw json.RawMessage, digest string) Input {
	in.Approval.AcceptanceContractDigest = digest
	in.ExpectedApproval.AcceptanceContractDigest = digest
	in.AcceptanceContract = raw
	return in
}

func wantRefusal(t *testing.T, err error, reason engine.Reason) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), string(reason)) {
		t.Fatalf("err = %v, want %s", err, reason)
	}
}

// TestCriterionIDSetMustMatch: the contract's criterion ids must equal the
// run's core criterion records exactly.
func TestCriterionIDSetMustMatch(t *testing.T) {
	for name, ids := range map[string][]string{"missing": {"C1"}, "extra": {"C1", "C2", "C3"}, "renamed": {"C1", "C9"}} {
		t.Run(name, func(t *testing.T) {
			repo, base := newRepo(t)
			dir := t.TempDir()
			now := time.Now().UTC().Truncate(time.Second)
			toAcceptance(t, repo, base, dir, now, "C1", "C2")
			raw, d := contractDoc("exit 0", ids...)
			in := withContract(approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, model.StagePlanning, "n-pl", now), raw, d)
			_, err := Apply(context.Background(), in, dir)
			wantRefusal(t, err, engine.ReasonContractCriteriaDiffer)
		})
	}
	t.Run("exact", func(t *testing.T) {
		repo, base := newRepo(t)
		dir := t.TempDir()
		now := time.Now().UTC().Truncate(time.Second)
		toAcceptance(t, repo, base, dir, now, "C1", "C2")
		raw, d := contractDoc("exit 0", "C2", "C1")
		in := withContract(approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, model.StagePlanning, "n-pl", now), raw, d)
		if _, err := Apply(context.Background(), in, dir); err != nil {
			t.Fatal(err)
		}
	})
}

// TestContractThresholdChangeRefusesTransition: after a contract is
// accepted, every later approval must carry its digest, and a contract with
// a changed threshold under the same ids is refused.
func TestContractThresholdChangeRefusesTransition(t *testing.T) {
	repo, base := newRepo(t)
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	toAcceptance(t, repo, base, dir, now, "C1")
	raw, d := contractDoc("exit 0", "C1")
	if _, err := Apply(context.Background(), withContract(approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, model.StagePlanning, "n-pl", now), raw, d), dir); err != nil {
		t.Fatal(err)
	}
	changed, cd := contractDoc("exit 0 on two runs", "C1")
	next := func(nonce string) Input {
		return approvedInput(t, repo, base, dir, model.StagePlanning, model.StageRouteSelection, nonce, now)
	}
	_, err := Apply(context.Background(), withContract(next("n-rs1"), changed, cd), dir)
	wantRefusal(t, err, engine.ReasonContractMismatch)
	_, err = Apply(context.Background(), withContract(next("n-rs2"), changed, d), dir)
	wantRefusal(t, err, engine.ReasonContractMismatch)
	_, err = Apply(context.Background(), next("n-rs3"), dir) // digest dropped
	wantRefusal(t, err, engine.ReasonContractMismatch)
	if _, err := Apply(context.Background(), withContract(next("n-rs4"), raw, d), dir); err != nil {
		t.Fatalf("bound contract refused: %v", err)
	}
	// The digest alone, without presenting the document again, is accepted.
	if _, err := Apply(context.Background(), withContract(approvedInput(t, repo, base, dir, model.StageRouteSelection, model.StageImplementationAuthorization, "n-ia", now), nil, d), dir); err != nil {
		t.Fatalf("carried digest refused: %v", err)
	}
}

// Accepting a contract needs the document, and a run that used a contract
// cannot accept criteria again without one.
func TestContractDocumentRequired(t *testing.T) {
	repo, base := newRepo(t)
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	toAcceptance(t, repo, base, dir, now, "C1")
	_, d := contractDoc("exit 0", "C1")
	_, err := Apply(context.Background(), withContract(approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, model.StagePlanning, "n-pl", now), nil, d), dir)
	wantRefusal(t, err, engine.ReasonContractRequired)
	other, _ := contractDoc("exit 0", "C1")
	_, err = Apply(context.Background(), withContract(approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, model.StagePlanning, "n-pl2", now), other, "sha-the-host-claims"), dir)
	wantRefusal(t, err, engine.ReasonContractMismatch)
}

// A run without a contract is unchanged, and a contract cannot be presented
// to it after Planning.
func TestRunWithoutContractUnchanged(t *testing.T) {
	repo, base := newRepo(t)
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	for i, step := range walk[:3] {
		if _, err := Apply(context.Background(), approvedInput(t, repo, base, dir, step.from, step.to, "plain-"+string(rune('a'+i)), now), dir); err != nil {
			t.Fatal(err)
		}
	}
	raw, d := contractDoc("exit 0", "C1")
	_, err := Apply(context.Background(), withContract(approvedInput(t, repo, base, dir, model.StageRouteSelection, model.StageImplementationAuthorization, "late", now), raw, d), dir)
	wantRefusal(t, err, engine.ReasonContractMismatch)
	recs, _ := store.New(dir).Load("run")
	for _, r := range recs {
		b, _ := json.Marshal(r.Approval)
		if strings.Contains(string(b), "acceptance_contract_digest") {
			t.Fatal("approval without a contract serialized the new field")
		}
	}
}

// Liz D1+D2 F06: a contract presented with an approval that names no
// contract digest is refused, never silently dropped.
func TestPresentedContractNeverDropped(t *testing.T) {
	repo, base := newRepo(t)
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	toAcceptance(t, repo, base, dir, now, "C1")
	raw, _ := contractDoc("exit 0", "C1")
	in := approvedInput(t, repo, base, dir, model.StageAcceptanceCriteria, model.StagePlanning, "n-pl", now)
	in.AcceptanceContract = raw
	_, err := Apply(context.Background(), in, dir)
	wantRefusal(t, err, engine.ReasonContractMismatch)
}
