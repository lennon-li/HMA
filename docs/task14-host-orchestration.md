# Task 14: interactive host orchestration P1

This task makes the architecture §12.4 host boundary executable without turning
HMA into a dispatcher. HMA records and checks what an interactive host such as
Fury decided and observed; the host still performs the real `hi` preflight and
starts the worker after HMA says the recorded conditions permit it.

## Authority boundary

HMA does **not**:

- select a route on its own;
- send the `hi` message;
- start a worker or model call;
- grant a human route or stage approval;
- silently substitute a provider, model, access service, runtime, or reviewer.

The host supplies those facts. HMA validates their coherence, records them in a
host-local hash chain, and blocks the two coding progression edges when the
required orchestration evidence is absent.

## CLI

```text
hma host decision  --input <file> --store <directory>
hma host preflight --input <file> --store <directory>
hma host dispatch  --input <file> --store <directory>
hma host review    --input <file> --store <directory>
hma host show      --store <directory> --run_id <id>
```

Each write input has `run_id`, `actor`, and an RFC3339 `timestamp`, plus exactly
one operation payload. Unknown fields and multiple JSON documents are refused.
A host record is appended only when the core HMA run already exists; each host
record binds the current core-chain head anchor.

## 1. Orchestration decision

The host records one of:

- `DELEGATE`
- `DIRECT_COST_EXCEPTION`

The decision binds the plan unit, bounded job, direct and delegated total-cost
estimates, comparison basis, rationale, approved permission envelope, work
context digest, and the complete route telemetry.

A direct-cost exception is valid only when the recorded estimated direct cost
is strictly lower than the recorded estimated delegated total cost. This is a
mechanical floor, not proof that the estimate is accurate. Convenience or
retained context alone is not a valid bypass.

Changing the route or mode requires an explicit superseding decision naming the
prior decision digest. A route change after an unavailable preflight is thereby
represented as an explicit escalation rather than silent fallback. A new human
route approval may also supersede the prior decision by carrying a new
`route_approval_digest`.

## 2. Route preflight

Before every implementation or review dispatch the host performs the actual
minimal `hi` handshake against the exact route, then records one status:

- `AVAILABLE`
- `UNAVAILABLE`
- `INDETERMINATE`
- `RATE_LIMITED`
- `QUOTA_EXHAUSTED`

The record binds worker, provider and provider family, exact model and model
family, profile digest, access service and digest, runtime, effort,
capabilities, permissions, route approval, response digest when available, and
freshness expiry.

Only `AVAILABLE` can support dispatch. All other statuses are recorded but fail
closed. A stale preflight also blocks dispatch.

## 3. Dispatch telemetry

An implementation dispatch must match the active `DELEGATE` decision, its
bounded job, permission envelope, exact route, and a fresh successful preflight.
The record includes the worker identity, provider, model, effort, access
service, runtime, bounded task, work-context digest, capabilities, and
permissions.

A review dispatch additionally binds:

- the exact implementation-record digest;
- immutable artifact digest;
- fixed review-packet digest;
- task risk classification.

The review route must have semantic-review capability and a read-only permission
envelope.

`hma host dispatch` records that these conditions were satisfied. It never
executes the worker itself.

## 4. Independent review

For normal coding work the reviewer must be:

- a distinct worker/agent;
- a different exact model;
- a separate work/review context.

For `high` or `critical` risk it must additionally use a different provider
family and model family. The review route is fixed by the review dispatch and
may not be silently replaced.

The review result binds the exact implementation record, review-dispatch record,
artifact digest, fixed review packet, reviewer identity, verdict, and optional
findings digest. Verdict is `APPROVE` or `BLOCK`.

Direct work by the interactive orchestrator is implementation for this rule and
therefore still requires an eligible independent reviewer.

## 5. Stage enforcement

The ordinary HMA transition command now checks the host record chain on the two
coding progression edges:

```text
IMPLEMENTATION_AUTHORIZATION -> IMPLEMENTATION_REVIEW
```

requires a current recorded implementation path: either a valid delegated
implementation dispatch or a valid `DIRECT_COST_EXCEPTION` decision.

```text
IMPLEMENTATION_REVIEW -> VERIFICATION
```

requires an `APPROVE` review bound to that exact current implementation.

Both edges require `approval.unit_id`, preventing a host event from satisfying a
different plan unit's gate. The pure core transition evaluator remains
side-effect-free; this enforcement belongs to the interactive-host CLI boundary.

## Host record store

Host orchestration records live below the HMA store at:

```text
<store>/host/<run_id>/
```

They form a separate host-local hash chain with sequence numbers, predecessor
hashes, content head anchors, and a host head file. Every record also binds the
core HMA head visible when it was recorded. The chain detects accidental or
uncoordinated record modification within the same host-trusted boundary as the
rest of HMA; it is not a same-UID tamper-resistance claim.

The separate host chain is deliberate. Provider/model/runtime/access-service
telemetry is host integration state and must not become a second portable route
policy or contaminate HMA's vendor-neutral core records.

## Conformance coverage

The P1 tests cover:

- delegated implementation;
- a valid and invalid direct-cost exception;
- route mismatch;
- successful and stale preflights;
- quota exhaustion and fail-closed dispatch;
- explicit route escalation after failed preflight;
- orchestrator/self-review rejection;
- same-model rejection;
- high-risk same-provider rejection;
- same-context rejection;
- valid independent review;
- fixed review-packet mismatch;
- coding-stage transition enforcement; and
- host-record tamper detection.

## First Fury pilot

For the first real pilot, Fury should:

1. establish the ordinary HMA run and approved unit/route;
2. write `hma host decision` before coding;
3. perform the real `hi` against the selected implementation route;
4. record `hma host preflight`;
5. call `hma host dispatch`; only a successful result permits Fury to start the worker;
6. after implementation, create an immutable artifact and review-packet digest;
7. repeat preflight + dispatch for the independent reviewer;
8. record the reviewer result with `hma host review`; and
9. use the ordinary HMA stage transition, which now refuses coding progression
   when these bindings are absent.

Uatu may independently inspect the core chain plus `hma host show`; Uatu should
not be used as the reviewer merely because it is the governor. Reviewer routing
must continue to follow the approved independence policy.
