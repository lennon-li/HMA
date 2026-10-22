package host

import (
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

// The mapping reads the claim and the facts only: a rule that holds blocks, a
// budget that is exceeded is waivable, and neither a held-back rule, an
// unknown name, a wrong kind, nor severity changes a thing.
func TestMapAuditFindingsDisposition(t *testing.T) {
	facts := FindingFacts{
		RuleHolds:      map[string]bool{"no_shell": true, "secrets": true},
		BudgetExceeded: map[string]bool{"max_files": true, "invented": true},
	}
	in := `[
	 {"id":"holds","severity":"low","hma_ref":{"kind":"kernel_rule","id":"no_shell"}},
	 {"id":"unregistered","severity":"critical","hma_ref":{"kind":"kernel_rule","id":"secrets"}},
	 {"id":"notheld","hma_ref":{"kind":"kernel_rule","id":"authority"}},
	 {"id":"exceeded","hma_ref":{"kind":"declared_budget","id":"max_files"}},
	 {"id":"unknownbudget","hma_ref":{"kind":"declared_budget","id":"invented"}},
	 {"id":"within","hma_ref":{"kind":"declared_budget","id":"max_diff_lines"}},
	 {"id":"wrongkind","hma_ref":{"kind":"declared_budget","id":"no_shell"}},
	 {"id":"crossed","hma_ref":{"kind":"kernel_rule","id":"max_files"}},
	 {"id":"plain","severity":"critical"}
	]`
	got, err := MapAuditFindings([]byte(in), facts)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]model.FindingDisposition{
		"holds": model.FindingBlock, "unregistered": model.FindingAdvisory, "notheld": model.FindingAdvisory,
		"exceeded": model.FindingWaivable, "unknownbudget": model.FindingAdvisory, "within": model.FindingAdvisory,
		"wrongkind": model.FindingAdvisory, "crossed": model.FindingAdvisory, "plain": model.FindingAdvisory,
	}
	for _, f := range got {
		if f.Disposition != want[f.ID] {
			t.Errorf("%s = %v, want %v", f.ID, f.Disposition, want[f.ID])
		}
		if f.Disposition == model.FindingAdvisory && f.Advisory != model.AdvisoryPark {
			t.Errorf("%s: advisory finding is not PARK", f.ID)
		}
	}
	// HMA raises its own findings whatever the report says, under reserved ids
	// a report may not use.
	out := AddEstablishedFindings(nil, FindingFacts{
		BudgetExceeded: map[string]bool{"max_files": true},
		RuleHolds:      map[string]bool{"no_shell": true, "authority": true},
	}, "sha256:0123456789abcdef0123")
	if len(out) != 2 || out[0].ID != MeasuredFindingPrefix+"max_files-0123456789abcdef" || out[0].Disposition != model.FindingWaivable ||
		out[1].ID != KernelFindingPrefix+"no_shell" || out[1].Disposition != model.FindingBlock {
		t.Fatalf("established findings %v", out)
	}
	for _, id := range []string{"HMA-BUDGET-max_files", "HMA-KERNEL-no_shell", "HMA-anything"} {
		if _, err := MapAuditFindings([]byte(`[{"id":"`+id+`"}]`), FindingFacts{}); err == nil {
			t.Errorf("reserved id %q accepted from a report", id)
		}
	}
}
