package host

import (
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

// F11: the reviewer must be independent of every implementation of the unit,
// including one superseded by a later decision on another route.
func TestGateCR2ReviewerIndependentOfEveryImplementation(t *testing.T) {
	history, first, _ := delegatedImplementationHistory()
	second := testRoute("phil", "google", "google", "gem", "gemini", "route-b", false)
	d2 := decisionFor(second, DecisionDelegate)
	dr2 := record(4, "decision-2", KindDecision)
	dr2.Decision = &d2
	p2 := RoutePreflight{UnitID: "U1", Role: RoleImplementation, Route: second, Status: PreflightAvailable, ResponseDigest: "hi-2", ExpiresAt: "2026-09-16T12:05:00Z"}
	pr2 := record(5, "preflight-2", KindPreflight)
	pr2.Preflight = &p2
	disp2 := Dispatch{UnitID: "U1", Role: RoleImplementation, Route: second, PreflightRecordDigest: "preflight-2", BoundedTask: d2.BoundedJob, WorkContextDigest: "ctx-phil"}
	ir2 := record(6, "implementation-2", KindDispatch)
	ir2.Dispatch = &disp2
	history = append(history, dr2, pr2, ir2)

	for name, tc := range map[string]struct {
		reviewer Route
		want     string
	}{
		"first-implementer": {func() Route {
			r := testRoute(first.WorkerID, "anthropic", "anthropic", "sonnet", "claude", "route-review", true)
			return r
		}(), ReasonReviewerNotIndependent},
		"first-implementer-model": {func() Route {
			r := testRoute("ming", "anthropic", "anthropic", first.Model, "claude", "route-review", true)
			return r
		}(), ReasonReviewerNotIndependent},
		"independent-of-both": {testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true), ""},
	} {
		t.Run(name, func(t *testing.T) {
			p := RoutePreflight{UnitID: "U1", Role: RoleReview, Route: tc.reviewer, Status: PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:10:00Z"}
			pr := record(7, "review-preflight", KindPreflight)
			pr.Timestamp = "2026-09-16T12:02:00Z"
			pr.Preflight = &p
			d := Dispatch{UnitID: "U1", Role: RoleReview, Route: tc.reviewer, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review", WorkContextDigest: "ctx-review",
				ImplementationRecordDigest: ir2.HeadAnchor, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskLow}
			err := EvaluateDispatch(append(append([]Record{}, history...), pr), d, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC))
			if tc.want == "" && err != nil {
				t.Fatalf("independent reviewer refused: %v", err)
			}
			if tc.want != "" && (err == nil || err.Error() != tc.want) {
				t.Fatalf("err = %v, want %s", err, tc.want)
			}
		})
	}
}
