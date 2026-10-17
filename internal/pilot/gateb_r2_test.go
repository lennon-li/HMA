package pilot

import (
	"context"
	"github.com/lennon-li/HMA/internal/model"
	"testing"
	"time"
)

func TestGateBR2PlanningPilot(t *testing.T) {
	r, b := newRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	in := pilotInput(t, r, b, "repo", "plan-nonce", ":1", now)
	in.ExpectedApproval.CurrentStage = model.StagePlanning
	in.ExpectedApproval.ProposedTargetStage = model.StageRouteSelection
	in.ExpectedApproval.ProducedHeadDigest = ""
	in.ExpectedApproval.DiffDigest = ""
	in.Approval = in.ExpectedApproval
	if _, e := Run(context.Background(), in, t.TempDir()); e != nil {
		t.Fatal(e)
	}
}
