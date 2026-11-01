package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
)

// legacyReviewDispatchRecord is a review dispatch record in the shape
// internal/host/host.go had before Gate E Phase 4a: a ReviewPacketBinding
// carrying only the pre-Phase-4a fields. Phases 1-3 did not touch
// ReviewPacketBinding's fields or json tags, so this is also the pre-Gate-E
// shape.
func legacyReviewDispatchRecord() Record {
	return Record{
		Version: 1, Kind: KindDispatch, RunID: "run", Sequence: 3,
		PredecessorHash: "pred", CoreHeadAnchor: "core-head", Timestamp: "2026-09-16T12:04:00Z", Actor: "fury",
		Dispatch: &Dispatch{
			UnitID: "U1", Role: RoleReview, Route: Route{
				WorkerID: "ming", Provider: "anthropic", ProviderFamily: "anthropic", ProviderDigest: "pd",
				Model: "sonnet", ModelFamily: "claude", ModelDigest: "md", ProfileDigest: "prof",
				AccessService: "cli", AccessServiceDigest: "asd", Runtime: "claude-code", ReasoningEffort: "medium",
				ExecutionContext: "isolated", CapabilityClasses: []string{"semantic_review"},
				PermissionClasses: []string{"read_only"}, RouteApprovalDigest: "approved-route",
			},
			PreflightRecordDigest: "preflight-digest", BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming",
			ImplementationRecordDigest: "impl-digest", ArtifactDigest: "artifact-digest", ReviewPacketDigest: "packet-digest",
			Risk: "high",
			Packet: &ReviewPacketBinding{
				ReviewDispatchID: "hma-rd-1", ReviewNonce: "0123456789abcdef0123456789abcdef", DispatchRunID: "run-1",
				IssuingReason: IssuePerChange, ExpectedLaunchHead: "launch-head", ExpectedLaunchWorktree: "launch-wt",
				ProjectRoot: "/repo", BaseRevision: "base", ProducedHead: "head", DiffDigest: "diff",
				ValidUntil: "2026-09-16T13:04:00Z", LedgerNamespace: "/ledger",
			},
		},
	}
}

// preGateEPhase4ADispatchAnchor is the dispatch record digest Ming computed
// against this exact fixture before Phase 4a added the checkpoint binding
// fields to ReviewPacketBinding (recordDigest = sha256 of json.Marshal with
// HeadAnchor cleared, internal/host/store.go). Pinning it, instead of only
// comparing two fresh computations against each other, is the technique
// Phase 2's hardening pass used for the contract and plan digests
// (internal/model/checkpoint_fields_test.go): it is what actually catches a
// future change that alters the anchor of an existing record.
const preGateEPhase4ADispatchAnchor = "e7f628d873a3c754d95869a35f84a2cbf564d3b613dde00a87cdb10c868338b1"

// TestLegacyHostRecordAnchorUnchanged is §5.5: a stored pre-Gate-E-Phase-4a
// dispatch record re-marshals to the same bytes and anchor now that
// ReviewPacketBinding carries the new checkpoint fields, all omitempty.
func TestLegacyHostRecordAnchorUnchanged(t *testing.T) {
	r := legacyReviewDispatchRecord()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"checkpoint_schema", "minimum_issuing_reason", "checkpoint_triggers",
		"unclassified_paths", "family_separation", "risk_floor", "contract_risk", "bound_contract_digest",
		"checkpoint_policy_digest", "manifest_digest", "checkpoint_covers"} {
		if strings.Contains(string(b), field) {
			t.Fatalf("zero-valued checkpoint field %q present in marshaled record", field)
		}
	}
	sum := sha256.Sum256(b)
	digest := hex.EncodeToString(sum[:])
	if digest != preGateEPhase4ADispatchAnchor {
		t.Fatalf("anchor %s != pinned pre-Phase-4a anchor %s", digest, preGateEPhase4ADispatchAnchor)
	}
}

// TestCheckpointFieldsWithoutSchemaRefused is §5.5: validatePacketBinding
// refuses a binding that carries a checkpoint field without
// checkpoint_schema, accepts it once checkpoint_schema is present, and
// refuses an unknown checkpoint_schema or minimum_issuing_reason value.
func TestCheckpointFieldsWithoutSchemaRefused(t *testing.T) {
	base := func() *ReviewPacketBinding {
		return &ReviewPacketBinding{
			ReviewDispatchID: "hma-rd-1", ReviewNonce: strings.Repeat("a", 32), DispatchRunID: "run-1",
			IssuingReason: IssuePerChange, ExpectedLaunchHead: "launch-head",
			ProjectRoot: "/repo", BaseRevision: "base", ProducedHead: "head", DiffDigest: "diff",
			ValidUntil: "2026-09-16T13:04:00Z",
		}
	}
	if err := validatePacketBinding(base()); err != nil {
		t.Fatalf("legacy-shaped binding refused: %v", err)
	}
	// withCheckpointRequired fills in the §3.5a fields validatePacketBinding
	// now requires whenever checkpoint_schema is present (minimum_issuing_
	// reason, risk_floor, checkpoint_covers), leaving alone any the case
	// under test already set, so each case still isolates one field.
	withCheckpointRequired := func(p *ReviewPacketBinding) {
		if p.MinimumIssuingReason == "" {
			p.MinimumIssuingReason = IssueCheckpoint
		}
		if p.RiskFloor == "" {
			p.RiskFloor = model.RouteRiskLow
		}
		if p.CheckpointCovers == nil {
			p.CheckpointCovers = &engine.CheckpointBaseline{ContractDigest: "cd", PermissionClasses: []string{"read_only"}}
		}
	}
	for name, edit := range map[string]func(*ReviewPacketBinding){
		"minimum_issuing_reason":   func(p *ReviewPacketBinding) { p.MinimumIssuingReason = IssueCheckpoint },
		"checkpoint_triggers":      func(p *ReviewPacketBinding) { p.CheckpointTriggers = []string{"T"} },
		"unclassified_paths":       func(p *ReviewPacketBinding) { p.UnclassifiedPaths = []string{"a"} },
		"family_separation":        func(p *ReviewPacketBinding) { p.FamilySeparation = true },
		"risk_floor":               func(p *ReviewPacketBinding) { p.RiskFloor = model.RouteRiskHigh },
		"contract_risk":            func(p *ReviewPacketBinding) { p.ContractRisk = model.RouteRiskLow },
		"bound_contract_digest":    func(p *ReviewPacketBinding) { p.BoundContractDigest = "cd" },
		"checkpoint_policy_digest": func(p *ReviewPacketBinding) { p.CheckpointPolicyDigest = "pd" },
		"manifest_digest":          func(p *ReviewPacketBinding) { p.ManifestDigest = "md" },
		"checkpoint_covers": func(p *ReviewPacketBinding) {
			p.CheckpointCovers = &engine.CheckpointBaseline{ContractDigest: "cd", PermissionClasses: []string{"read_only"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := base()
			edit(p)
			if err := validatePacketBinding(p); err == nil {
				t.Fatalf("%s without checkpoint_schema accepted", name)
			}
			p.CheckpointSchema = CheckpointSchemaV1
			withCheckpointRequired(p)
			if err := validatePacketBinding(p); err != nil {
				t.Fatalf("%s with checkpoint_schema refused: %v", name, err)
			}
		})
	}

	badSchema := base()
	badSchema.CheckpointSchema = "wrong"
	if err := validatePacketBinding(badSchema); err == nil {
		t.Fatal("unknown checkpoint_schema accepted")
	}

	badReason := base()
	badReason.CheckpointSchema = CheckpointSchemaV1
	badReason.MinimumIssuingReason = "bogus"
	if err := validatePacketBinding(badReason); err == nil {
		t.Fatal("invalid minimum_issuing_reason accepted")
	}
}

// TestCheckpointSchemaMissingRequiredFieldRefused is Phil's finding 2: §3.5a
// marks minimum_issuing_reason, risk_floor, and checkpoint_covers as always
// present with the schema. validatePacketBinding must refuse a binding that
// carries checkpoint_schema but omits any one of the three, even though every
// other checkpoint field (e.g. checkpoint_triggers, contract_risk) is
// individually optional.
func TestCheckpointSchemaMissingRequiredFieldRefused(t *testing.T) {
	full := func() *ReviewPacketBinding {
		return &ReviewPacketBinding{
			ReviewDispatchID: "hma-rd-1", ReviewNonce: strings.Repeat("a", 32), DispatchRunID: "run-1",
			IssuingReason: IssuePerChange, ExpectedLaunchHead: "launch-head",
			ProjectRoot: "/repo", BaseRevision: "base", ProducedHead: "head", DiffDigest: "diff",
			ValidUntil: "2026-09-16T13:04:00Z", CheckpointSchema: CheckpointSchemaV1,
			MinimumIssuingReason: IssueCheckpoint, RiskFloor: model.RouteRiskLow,
			CheckpointCovers: &engine.CheckpointBaseline{ContractDigest: "cd", PermissionClasses: []string{"read_only"}},
		}
	}
	if err := validatePacketBinding(full()); err != nil {
		t.Fatalf("binding with all three required fields refused: %v", err)
	}
	for name, edit := range map[string]func(*ReviewPacketBinding){
		"minimum_issuing_reason": func(p *ReviewPacketBinding) { p.MinimumIssuingReason = "" },
		"risk_floor":             func(p *ReviewPacketBinding) { p.RiskFloor = "" },
		"checkpoint_covers":      func(p *ReviewPacketBinding) { p.CheckpointCovers = nil },
	} {
		t.Run(name, func(t *testing.T) {
			p := full()
			edit(p)
			if err := validatePacketBinding(p); err == nil {
				t.Fatalf("checkpoint_schema with %s missing accepted", name)
			}
		})
	}
}

// TestFamilySeparationFromPacketAddsStrong is §3.5 E0 step 4's "remaining
// term": EvaluateDispatch's strong independence check now also fires on
// d.Packet.FamilySeparation (T7/Disputed cases the dispatch risk does not
// express), without weakening any existing term.
func TestFamilySeparationFromPacketAddsStrong(t *testing.T) {
	history, impl, implementation := delegatedImplementationHistory()
	producedHead, diff := "head", "diff"
	artifact := ReviewArtifactDigest(producedHead, diff, "")
	cases := []struct {
		name             string
		familySeparation bool
		sameProvider     bool
		sameModelFamily  bool
		want             string
	}{
		{"family-separation-same-provider", true, true, false, ReasonReviewerNotIndependent},
		{"family-separation-same-model-family", true, false, true, ReasonReviewerNotIndependent},
		{"family-separation-separated", true, false, false, ""},
		{"no-family-separation-same-family-low-risk", false, true, true, ""},
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
			d := Dispatch{UnitID: "U1", Role: RoleReview, Route: r, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review fixed packet",
				WorkContextDigest: "ctx-review", ImplementationRecordDigest: implementation, ArtifactDigest: artifact, ReviewPacketDigest: "packet",
				Risk: model.RouteRiskLow,
				Packet: &ReviewPacketBinding{
					ReviewDispatchID: "hma-rd-fs", ReviewNonce: strings.Repeat("b", 32), DispatchRunID: "run-fs",
					IssuingReason: IssuePerChange, ExpectedLaunchHead: "launch-head",
					ProjectRoot: "/repo", BaseRevision: "base", ProducedHead: producedHead, DiffDigest: diff,
					ValidUntil: "2026-09-16T13:04:00Z", CheckpointSchema: CheckpointSchemaV1, FamilySeparation: tc.familySeparation,
					MinimumIssuingReason: IssuePerChange, RiskFloor: model.RouteRiskLow,
					CheckpointCovers: &engine.CheckpointBaseline{ContractDigest: "cd", PermissionClasses: []string{"read_only"}},
				},
			}
			err := EvaluateDispatch(append(history, pr), d, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC))
			if tc.want == "" && err != nil {
				t.Fatalf("case %s refused: %v", tc.name, err)
			}
			if tc.want != "" && (err == nil || err.Error() != tc.want) {
				t.Fatalf("case %s: err = %v, want %s", tc.name, err, tc.want)
			}
		})
	}
}
