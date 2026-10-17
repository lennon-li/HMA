package pilot

import (
	"context"
	"github.com/lennon-li/HMA/internal/model"
	"testing"
	"time"
)

func TestGateBEarlyStagePilot(t *testing.T) {
	repo, base := newRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	in := pilotInput(t, repo, base, "repo", "audit-nonce", ":1", now)
	in.ExpectedApproval.CurrentStage = model.StageGrounding
	in.ExpectedApproval.ProposedTargetStage = model.StageAcceptanceCriteria
	in.ExpectedApproval.ProducedHeadDigest = ""
	in.ExpectedApproval.DiffDigest = ""
	in.Approval = in.ExpectedApproval
	if _, err := Run(context.Background(), in, t.TempDir()); err != nil {
		t.Errorf("valid early-stage capture rejected: %v", err)
	}
}
