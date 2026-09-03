# Task 8 — Versioned Route-Policy Contract Proposal

Status: implementation drafted in the working tree under Lennon's approval;
Go-toolchain verification and independent review remain pending. This proposal
does not authorize route selection, host discovery, client adapters, worker
dispatch, or changes to HMA's v1 authority boundaries.

## 1. Objective

Define one versioned, portable route-policy data contract and its deterministic
normalized projection. The projection is the single source of truth for future
route selection: a policy change must produce a different digest, while
semantically equivalent input ordering must produce the same digest.

This is a post-v1 Task 8 proposal. It extends architecture §12 without creating
a second routing system or importing provider-specific client configuration.

## 2. Bounded scope

In scope:

- portable capability, permission, risk, independence, preference, and fallback
  rule vocabulary needed to describe route policy;
- a versioned route-policy input and canonical normalized projection;
- deterministic set/map normalization and content digesting using the existing
  standard-library conventions;
- pure validation and projection functions with exact rejection reasons;
- table-driven fixtures proving canonicalization and policy-digest behavior;
- the smallest architecture-contract update needed to make the SSoT explicit.

Out of scope:

- choosing a route or implementing the route-selection algorithm;
- host-local machine discovery or model availability checks;
- Codex, Claude Code, OpenCode, Cursor, CI, or provider SDK adapters;
- credentials, private paths, profile names, aliases, endpoints, or secrets;
- changing `RouteAttestation`, approval semantics, stage transitions, or the
  v1 CLI surface;
- worker dispatch, autonomous execution, publication, release, or deployment.

## 3. Proposed implementation route

The implementation route is a focused standard-library Go change followed by
independent read-only review.

### Files proposed for the implementation packet

- `internal/model/route_policy.go` — versioned portable input and normalized
  projection types, closed vocabularies, and structural validation helpers.
- `internal/model/route_policy_test.go` — model-level validation and
  canonicalization tests.
- `internal/engine/route_policy.go` — pure normalization and digest evaluation;
  no I/O, discovery, dispatch, stage advancement, or route choice.
- `internal/engine/route_policy_test.go` — deterministic behavior and exact
  rejection-reason tests.
- `testdata/engine/route-policy.json` — valid and invalid table-driven cases.
- `docs/architecture.md` — narrowly update §12 and §25 to identify the
  versioned normalized projection and digest as the routing SSoT.

No existing production file is to be modified unless the approved packet shows
that an existing validator or shared canonicalization helper must be reused.
The packet must identify any such change before implementation begins.

## 4. Contract requirements

The contract must make these facts explicit and machine-checkable:

1. The policy version is required and participates in the digest.
2. Inputs use portable capability and permission classes, not vendor/client
   names or host-local identifiers.
3. Rule identifiers and referenced classes are unique and non-empty.
4. Normalization sorts unordered sets and rule collections by documented stable
   keys, rejects duplicate/conflicting definitions, and preserves semantic
   distinctions rather than silently merging them.
5. The normalized projection contains only policy inputs and normalized rule
   data; it contains no credentials, private paths, profile names, runtime
   availability claims, or execution results.
6. The digest is computed from one documented canonical representation and is
   stable across equivalent input ordering.
7. A changed policy version, rule, capability, permission, risk threshold, or
   independence requirement changes the digest.
8. Unknown vocabulary, malformed references, contradictory bounds, and
   unsupported policy versions are rejected deterministically.
9. The evaluator returns a proposal/validation result only. It cannot approve,
   dispatch, select, substitute, widen permission, waive independence, or
   advance a stage.

## 5. Required fixture matrix

The fixture file must cover at least:

- one minimal valid policy;
- equivalent policies with reordered rules and class sets that yield the same
  normalized projection and digest;
- each material field change yielding a different digest;
- duplicate rule and duplicate class rejection;
- unknown capability and permission rejection;
- unknown policy-version rejection;
- malformed or contradictory risk/independence/permission rule rejection;
- forbidden provider/client/credential/private-path fields rejected or made
  unrepresentable by the portable types;
- an empty/no-route policy representation that remains valid policy data but
  does not claim a route or authorize execution.

The fixtures must not encode real credentials, local paths, host profile names,
or provider-specific model/client identifiers.

## 6. Verification gates

Before implementation authorization:

- Lennon approves this bounded scope, file list, route, budget, and stop rules.

During implementation:

- `gofmt -d` is empty for changed Go files;
- `go test -count=1 ./...` passes;
- `go vet ./...` passes;
- `go build ./...` passes;
- `git diff --check` passes;
- fixture execution is deterministic across repeated runs;
- no route is selected and no external command is executed by the new code.

After implementation:

- an independent read-only review checks the contract against architecture §12,
  Task 6 route-attestation boundaries, and this exact file scope;
- the parent independently re-runs all checks and inspects the final diff;
- publication, if later requested, is a separate human-authorized action.

## 7. Stop conditions

Stop and return for clarification if the work requires any of the following:

- choosing a provider, model, client, access service, or host profile;
- adding machine-truth discovery or reading credentials/configuration;
- defining a route-selection preference not represented in the approved contract;
- changing portable route-attestation or approval-binding semantics;
- adding a CLI command, dispatch behavior, or autonomous authority;
- changing architecture boundaries beyond the narrow §12/§25 clarification;
- introducing a dependency, schema framework, or non-standard-library runtime;
- discovering that a canonicalization decision changes existing record semantics.

## 8. Approval request

The implementation above follows this approved scope. The next review packet
must bind the exact Go types, closed vocabularies, canonical JSON representation,
digest algorithm, token/effort budget, independent reviewer route, and any
publication decision.
