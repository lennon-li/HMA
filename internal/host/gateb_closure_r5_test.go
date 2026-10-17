package host

import (
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

// R5-02 (M3): disputed and final validation are host-declared on the review
// dispatch and require provider- and model-family separation at any risk
// level, as high and critical risk do. Low-risk ordinary review keeps the
// weaker different-model rule.
func TestGateBClosureR5DisputedAndFinalValidationSeparateFamilies(t *testing.T) {
	history, impl, implementation := delegatedImplementationHistory()
	cases := []struct {
		name            string
		disputed, final bool
		sameProvider    bool
		sameModelFamily bool
		want            string
	}{
		{"disputed-same-provider", true, false, true, false, ReasonReviewerNotIndependent},
		{"disputed-same-model-family", true, false, false, true, ReasonReviewerNotIndependent},
		{"final-same-provider", false, true, true, false, ReasonReviewerNotIndependent},
		{"final-same-model-family", false, true, false, true, ReasonReviewerNotIndependent},
		{"disputed-separated", true, false, false, false, ""},
		{"final-separated", false, true, false, false, ""},
		{"ordinary-low-same-family", false, false, true, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true)
			if tc.sameProvider {
				r.ProviderFamily = impl.ProviderFamily
			}
			if tc.sameModelFamily {
				r.ModelFamily = impl.ModelFamily
			}
			p := RoutePreflight{UnitID: "U1", Role: RoleReview, Route: r, Status: PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:10:00Z"}
			pr := record(4, "review-preflight", KindPreflight)
			pr.Timestamp = "2026-09-16T12:02:00Z"
			pr.Preflight = &p
			d := Dispatch{UnitID: "U1", Role: RoleReview, Route: r, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review fixed packet", WorkContextDigest: "ctx-review", ImplementationRecordDigest: implementation, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskLow, Disputed: tc.disputed, FinalValidation: tc.final}
			err := EvaluateDispatch(append(history, pr), d, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC))
			if tc.want == "" && err != nil {
				t.Fatalf("separated review refused: %v", err)
			}
			if tc.want != "" && (err == nil || err.Error() != tc.want) {
				t.Fatalf("err = %v, want %s", err, tc.want)
			}
		})
	}
}
