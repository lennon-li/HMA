package host

import (
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

// A route's identity ignores its approval reference and changes with any
// identity field.
func TestRouteIdentity(t *testing.T) {
	r := Route{WorkerID: "jax", Provider: "openai", ProviderFamily: "openai", ProviderDigest: "p", Model: "m", ModelFamily: "f", ModelDigest: "md",
		ProfileDigest: "pd", AccessService: "s", AccessServiceDigest: "sd", Runtime: "codex", ReasoningEffort: "high",
		ExecutionContext: model.ExecutionContextProxy, CapabilityClasses: []string{model.CapabilityIsolationPermission}, PermissionClasses: []string{model.PermissionBoundedWrite},
		RouteApprovalDigest: "approval-1"}
	a, err := RouteIdentity(r)
	if err != nil {
		t.Fatal(err)
	}
	r2 := r
	r2.RouteApprovalDigest = "approval-2"
	if b, _ := RouteIdentity(r2); b != a {
		t.Fatal("identity depends on the approval reference")
	}
	r3 := r
	r3.ReasoningEffort = "low"
	if c, _ := RouteIdentity(r3); c == a {
		t.Fatal("identity ignores reasoning effort")
	}
	g, err := RouteGrant(r)
	if err != nil || g.Identity != a || g.ApprovalDigest != "approval-1" || len(g.PermissionClasses) != 1 {
		t.Fatalf("grant %+v %v", g, err)
	}
}
