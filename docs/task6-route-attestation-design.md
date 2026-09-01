# Task 6 — Route Attestation: Schema and Verification Semantics

Status: planning only. No Go code, CLI surface, or schema file is authorized by this document. It defines the target shape for a future implementation plan and references the existing `internal/model` and `internal/engine` conventions it must fit into.

## 0. Problem framing

HMA already captures *that* a command ran and *what changed* (`EvidenceRef`, §14 of `docs/architecture.md`). It does not capture *what executed the model* — which provider, which model identity, through which proxy or credential boundary, under whose authorization. §12.2 of the architecture already commits to the shape of the answer without defining it formally:

> A portable route attestation contains only an opaque profile digest, provider family, model family, capability classes, and permission classes; it contains no private paths, credentials, profile names, or provider-specific secrets.

Task 6 turns that sentence into a schema and a deterministic verification procedure, consistent with the "no complex PKI" constraint: V1 has no certificate chains, no signed JWTs from providers, no hardware attestation. Trust is host-declared and Git-anchored, the same trust model the local pilot (`internal/pilot`) and CI adapter (`internal/ci/github`) already use for repository and revision identity.

## 1. Route identity

A route is the tuple of *who ran the model* for one implementation or validation unit. It is uniquely identified by:

| Field | Meaning | Analogous to |
|---|---|---|
| `provider_family` | Portable vendor class (e.g. `anthropic`, `openai`, `google`, `local`) | §12.1 capability taxonomy — never a raw SDK/vendor string beyond the family label |
| `model_family` | Portable model class (e.g. `claude-opus`, `gemini-pro`), not a dated/private model ID | Task 1's portable model vocabulary (`internal/model`) |
| `execution_context` | How the call was made: `direct_api`, `proxy`, `local_runtime`, `ci_runner` | New — needed because a proxied route has a different trust boundary than a direct one |
| `profile_digest` | Opaque SHA-256 digest of the host-local runtime profile that resolved the above (credentials, endpoint, proxy address) | §12.2 "opaque profile digest" |
| `capability_classes` | Subset of the §12.1 taxonomy this route is asserted to satisfy (e.g. `bounded_implementation`, `semantic_review`) | §12.1 |
| `permission_classes` | Access/interaction mode granted to this route (e.g. `read_only`, `bounded_write`, `no_external_effect`) | §4.2, §12.3 permission envelope |

Route identity is **per plan unit**, never run-wide — this matches §5.4's "Route selection is per implementation unit, never a run-wide selection." Two units in the same run may attest different routes; each attestation binds to exactly one unit and one evidence capture.

What route identity deliberately excludes, per the kernel boundary in §12.2 and §23: private paths, raw credentials, profile names, provider-specific secrets, or any string that could re-identify a specific host account or proxy endpoint. If two hosts use different credentials behind the same provider/model/capability/permission tuple, their attestations are indistinguishable by design — that indistinguishability is the point, not a gap.

## 2. Attestation schema

The attestation is not a new `RecordKind`. It travels as an optional nested object on `EvidenceRef`, because a route attestation is meaningless without the evidence it produced — it answers "who produced this evidence," not a free-standing claim. This mirrors how `ApprovalBinding` and `EvidenceRef` are optional fields on `Record` rather than separate record families.

```go
// RouteAttestation binds an evidence capture to the route that produced it.
// It contains no credential, private path, profile name, or provider-specific
// secret; see the portable capability taxonomy (architecture.md §12.1) and
// host-local profile boundary (§12.2).
type RouteAttestation struct {
    ProviderFamily     string   `json:"provider_family"`
    ModelFamily        string   `json:"model_family"`
    ExecutionContext   string   `json:"execution_context"`
    ProfileDigest      string   `json:"profile_digest"`
    CapabilityClasses  []string `json:"capability_classes"`
    PermissionClasses  []string `json:"permission_classes"`
    RouteApprovalDigest string  `json:"route_approval_digest"`
}
```

Field notes:

- `RouteApprovalDigest` binds this attestation to the `ApprovalBinding.TransitionDigest` of the Route Selection stage transition that approved this unit's route (§5.4: `Route selection → Implementation authorization`). This is what prevents an attestation from claiming a route the human never approved for this unit.
- `ExecutionContext` is a closed vocabulary (`direct_api`, `proxy`, `local_runtime`, `ci_runner`), not a free string, following the existing enum-not-string convention in `internal/model/types.go`.
- `CapabilityClasses` and `PermissionClasses` are sorted string sets drawn from a fixed portable taxonomy (extending §12.1's list), not host-defined free text — an unrecognized class value is a validation failure, matching how `ValidFindingDisposition` etc. reject unknown enum values today.

`EvidenceRef` gains one new optional field:

```go
type EvidenceRef struct {
    // ... existing fields unchanged ...
    Route *RouteAttestation `json:"route,omitempty"`
}
```

Attestation is optional at the type level (not every evidence capture is model-execution evidence — e.g. `git diff` output has no route) but is **required by policy** for evidence submitted at Implementation Review and Independent Validation gates, where "who ran the model" is exactly the question being gated. That requirement is a Verification Semantics rule (§3), not a structural one, matching the existing split between `ValidateRecord` (structural) and engine evaluators (policy-bearing).

## 3. Verification semantics

No PKI means HMA cannot cryptographically prove a provider's identity. What it can do — consistent with the CI adapter's `EvaluateCIVerification` pattern (`internal/engine/ci_verification.go`) — is deterministic **consistency and authorization checking** against host-declared trust state, not remote authentication.

### 3.1 Trust input: the allowed-routes manifest

The host supplies an allowed-routes manifest, structurally analogous to the host-local profiles described in §12.2 and kept outside the repository per §23 ("Repository-owned portable artifacts must not encode a maintainer's home path, runtime profile names, credentials, provider choices..."). Each manifest entry maps a `profile_digest` to the `provider_family` / `model_family` / `execution_context` / `capability_classes` / `permission_classes` it is authorized to assert. The manifest is loaded at verification time the same way CI trust roots are "explicitly configured trust roots" per §22 — it is part of the trusted computing base, not part of the portable record stream.

### 3.2 EvaluateRouteAttestation

A new deterministic evaluator, same shape as `EvaluateCIVerification` and `EvaluateResolution`: pure function, no side effects, returns a `Decision`/`Reason` pair, never advances a stage.

```go
type RouteAttestationRequest struct {
    Attestation           model.RouteAttestation
    AllowedRoutes         map[string]AllowedRouteEntry // keyed by profile_digest
    ApprovedRouteApprovalDigest string // from the unit's Route Selection approval
    ApprovedCapabilityClasses   []string
    ApprovedPermissionClasses   []string
}
```

`EvaluateRouteAttestation` checks, in order:

1. **Structural completeness** — all `RouteAttestation` fields non-empty and enum fields drawn from the closed vocabulary (mirrors `validateEvidence`'s required-field checks).
2. **Approval binding** — `Attestation.RouteApprovalDigest` equals the transition digest of the currently active Route Selection approval for this unit. A mismatch means the attestation claims a route different from what the human approved, or the approval is stale (§6: "A change to any actually bound field invalidates the approval").
3. **Manifest membership** — `Attestation.ProfileDigest` exists as a key in `AllowedRoutes`.
4. **Manifest consistency** — the manifest entry's `provider_family`, `model_family`, and `execution_context` exactly equal the attested values. The manifest is the source of truth; the attestation is only a claim about which manifest entry produced this evidence.
5. **Capability/permission containment** — `Attestation.CapabilityClasses` and `PermissionClasses` are each a subset of what both the manifest entry *and* the Route Selection approval authorize. A route cannot self-escalate capability or permission beyond what was approved, matching §4.2 ("HMA may not grant itself permission to advance") and the escalation rule in §5.4 ("human-confirmed escalation returns the affected unit to its approved route gate without inheriting broader permission").

If all five hold, the result is `DecisionLegalPendingApproval` — same as CI verification, this is *evidence eligible to support a transition packet*, not an autonomous advance. Machine authority here is limited to §4.2's "check record shape, freshness, revision binding, and integrity"; it never authorizes on its own.

### 3.3 What this does and does not prove

This is explicitly **not** proof that a specific vendor actually executed the call — there is no cryptographic signature from Anthropic, OpenAI, or Google in scope for V1. It proves:

- the evidence-producing route is one the human explicitly authorized for this unit (via the Route Selection approval binding), and
- the route matches a host-declared, out-of-repository trust entry the human/host controls.

This is consistent with §22's threat model, which places "model capability claims" and "runtime-discovery output" in the *adversarial-input* category and the "explicitly configured trust roots" in the *trusted computing base* category — route attestation verification is a trust-root check, not a capability or identity oracle. A compromised or lying local host remains out of scope ("A fully hostile local operating system is out of scope for v1").

## 4. Failure modes

Following the existing `Reason` enum convention (`internal/engine/transitions.go`, `ci_verification.go`), route attestation verification failures are exact, closed reason codes, never a generic string:

| Reason | Condition |
|---|---|
| `ROUTE_ATTESTATION_MISSING` | Evidence required at a gate that mandates attestation (§2) carries no `Route` field |
| `ROUTE_ATTESTATION_INCOMPLETE` | A required `RouteAttestation` field is empty or an enum field is not in the closed vocabulary |
| `ROUTE_APPROVAL_MISMATCH` | `RouteApprovalDigest` does not equal the active Route Selection approval's transition digest for this unit |
| `ROUTE_APPROVAL_STALE` | The referenced Route Selection approval has been invalidated by a later change (§5.4, §6) |
| `ROUTE_PROFILE_UNKNOWN` | `ProfileDigest` is not present in the host-supplied allowed-routes manifest |
| `ROUTE_MANIFEST_MISMATCH` | Attested `provider_family`/`model_family`/`execution_context` disagrees with the manifest entry for that `ProfileDigest` |
| `ROUTE_CAPABILITY_ESCALATION` | Attested `CapabilityClasses` is not a subset of the approved+manifest-authorized set |
| `ROUTE_PERMISSION_ESCALATION` | Attested `PermissionClasses` is not a subset of the approved+manifest-authorized set |
| `ROUTE_UNIT_SCOPE_VIOLATION` | Attestation is presented for a plan unit other than the one its bound Route Selection approval covers |

Every failure mode yields `DecisionRejected`; none of them are `WAIVABLE` — route/permission-envelope correctness is listed under the unwaivable kernel in §7.1 ("valid authority and permission envelope"; "authentic, current, revision-bound evidence"). A rejected route attestation blocks the transition packet from being presented as approvable; it does not itself produce a terminal outcome. Repeated rejection at the same unit follows the ordinary retry/escalation classification in §17 (most naturally `CAPABILITY_MISMATCH` or `PERMISSION_OR_SAFETY_BLOCK`, decided when this is implemented, not here).

## 5. Open questions for the implementation-planning gate

These are deliberately left undecided, consistent with §25's pattern of naming open decisions rather than inventing answers prematurely:

1. Exact closed vocabulary for `capability_classes` / `permission_classes` — extend §12.1's list or define a separate enum in `internal/model`.
2. Whether `execution_context: ci_runner` routes reuse the existing `internal/ci/github` environment verification (`Environment.Actor`) as an additional binding, or remain fully separate.
3. Allowed-routes manifest file format and location (parallel to CI's "configured trust roots," per §15/§22 — likely host-local, outside the repository, not a new portable JSON Schema artifact).
4. Whether attestation is mandatory at Verification and Independent Validation gates in addition to Implementation Review, given §13's independence requirements ("different provider/model family for high-risk, disputed, or final validation") — attestation is the natural mechanism to *check* that independence claim deterministically, but that wiring is Task 6's natural follow-on, not this document's scope.