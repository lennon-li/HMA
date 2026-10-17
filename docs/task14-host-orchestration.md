# Task 14: interactive host orchestration P1

Status: partial HMA-side record and transition-gate implementation. This does
not make an interactive CLI host MVP-conforming. Remaining MVP requirements are
tracked in `TODO.md`. Learned decision-provider integrations under §12.5,
including Jev, are optional post-MVP work and are not part of this task's MVP
acceptance boundary.

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
hma host review-packet   --input <file> --store <directory> --out <packet file>
hma host review-import   --input <file> --store <directory>
hma host findings-import --input <file> --store <directory>
hma host loop-evidence   --input <file> --store <directory>
hma host show      --store <directory> --run_id <id> [--mode]
```

Each write input has `run_id`, `actor`, and an RFC3339 `timestamp`, plus exactly
one operation payload. Unknown fields and multiple JSON documents are refused.
A host record is appended only when the core HMA run already exists; each host
record binds the current core-chain head anchor. The two coding transition
gates reject the implementation or review evidence when that bound core head is
stale. This protects those gates from reusing host telemetry after the core run
changes; it does not establish that the recorded route approval digest matches
an approved core route.

## 1. Orchestration decision

The host records one of:

- `DELEGATE`
- `DIRECT_COST_EXCEPTION`

The decision binds the plan unit, bounded job, direct and delegated total-cost
estimates, comparison basis, rationale, approved permission envelope, work
context digest, and the complete route telemetry.

A direct-cost exception is valid only when the recorded estimated direct cost
is strictly lower than the recorded estimated delegated total cost. This is a
mechanical floor, not proof that the estimate is accurate. The current HMA
check does not yet prove that the job is small and within its approved plan,
that the route is eligible, or that the estimate includes setup, duplicated
context, expected execution, and review coordination. Convenience or retained
context alone is not a valid bypass; the interactive host must enforce the full
§12.4 conditions until those bindings are executable here.

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
findings digest. A host-written review verdict is `APPROVE` or `BLOCK`.

### Imported reviews (Gate C)

`hma host review-packet` makes HMA generate the review packet. HMA takes its
own snapshot of the repository (`repository_root`, `base_revision`, and
uncommitted content only if `dirty_worktree_approved`), derives the review
`artifact_digest` from it (a host-named digest that differs is refused), and
records the canonical project root and the agent-dispatch launch digests it
expects (`launch_head`, `launch_worktree_sha256`, computed as
`agent-run-ledger` does). It creates a fresh 128-bit `review_nonce` and a
`review_dispatch_id`, writes the packet outside the repository, and appends
the review dispatch through the normal dispatch checks. The reviewer must
launch within one hour (`[dispatch time, valid_until)`), with
`agent-dispatch --run-id <dispatch_run_id>`, and copies `hma_binding`
{`packet_sha256`, `review_dispatch_id`, `review_nonce`} into its report. A
final audit (`issuing_reason: final`) needs provider- and model-family
separation, as high or critical risk, disputed, and final validation do. The
reviewer must be independent of every implementation of the unit, not only
the latest. Once a run has any HMA-generated packet, host-written review
dispatches and reviews are refused (`HOST_REVIEW_IMPORT_REQUIRED`).
`hma host dispatch` never accepts a `packet` binding; only `review-packet`
creates one.

The packet also carries the bound HMA state it reviews against: the core head,
the unit's approved route and `accepted_plan_digest` (refused if missing),
the projected criteria, the unit's current and unowned findings, and active
waivers. Every write result and `hma host show --mode` report the run's
`review_enforcement`: `legacy_manual` until the run's first HMA-generated
packet, `packet_import` after it.

`hma host review-import` answers only the unit's latest packet, once, and only
while the core head is the one the packet bound. It reads the receipt at
`<ledger_dir>/runs/<dispatch_run_id>/receipt.json` and imports only if: the run
id, `mode: review`, `state: completed`, and `exit_code: 0` match; the receipt
packet digest, launch digests, and project root equal the recorded values;
the receipt agent, model, provider, and effort equal the review route;
`started_at` is inside the packet's validity interval; the report bytes hash
to a `report_files` entry and are one JSON document with a `findings` array;
the live repository still equals the packet's artifact (`HOST_REVIEW_TARGET_CHANGED`);
the report `hma_binding` equals the packet; and, for a final audit or final
validation, an approving report has `isolation: isolated`. Otherwise it refuses
with `HOST_REVIEW_RECEIPT_MISMATCH`, `HOST_REVIEW_REPORT_MISMATCH`,
`HOST_FINDINGS_IMPORT_INVALID`, `HOST_REVIEW_PACKET_MISMATCH`,
`HOST_CORE_BINDING_STALE`, or `HOST_REVIEW_ISOLATION_INSUFFICIENT` and
appends nothing. The verdict map is total: `pass` is `APPROVE`,
`fix-required` is `REQUEST_CHANGES`, and any other or missing verdict is
`ESCALATE` (next action `ESCALATE_TO_HUMAN`). The report is checked against
the full independent-audit report contract (a Go port of
`validate_report.py`, kept equal by a parity test); a report that breaks it
is recorded as `ESCALATE`, as is a `pass` that carries an HMA citation in
`hma_ref`. Under the trusted-host model the receipt is
host-written, so these checks detect drift and accident, not forgery.

`hma host findings-import` writes one core `finding` record with
`findings_snapshot: true` and `findings_source` from the latest imported review
of the current implementation and its exact report bytes, only while the core
head is the one the review bound (one snapshot per review) and the live
repository still equals the packet's artifact. Each imported finding records
its owning unit in `unit_id` (ids are also namespaced `<unit>/<id>`);
ownership is that field, never the prefix. Other units' findings are kept. A
legacy finding without `unit_id` belongs to the unit of the pre-snapshot core
record that last set it; if that record names no unit, the finding stays in
force until a human decides its ownership. Severity
never sets disposition, and HMA has no kernel-rule registry or plan budgets
yet, so every imported finding is `ADVISORY` (`PARK`); a citation cannot create
an unwaivable blocker by label alone. A snapshot from a non-approving review
keeps every earlier `BLOCK` and `WAIVABLE` finding of the unit and never
weakens one with the same id; an approving review's snapshot replaces the
unit's findings. An imported review satisfies the Verification gate only
after its snapshot is recorded (`HOST_FINDINGS_SNAPSHOT_MISSING`); any other
core record after the review makes it stale. No import path writes
acceptance-criterion dispositions.

`hma host loop-evidence` imports a verify-loop `state.json` whose envelope
`hma` block names this run, unit, and store, whose `envelope_sha256` equals
the canonical hash of the envelope (as verify-loop computes it), and whose
envelope names an `authority_hook` and `escalate_to: stop`. It holds the host
run lock throughout, refuses a terminal run or a unit without an approved
route before running anything, and refuses an attempt id equal to a review
launch's run id. HMA does not use the loop's own result:
it re-runs the loop's targeted `verify.argv` itself, without a shell, through
the CI evidence capture, against the snapshot it observes before and after
(a check that changes the worktree is refused), appends that as a core
`evidence` record, and appends a host `loop_evidence` record binding the state
bytes, the envelope digest, each attempt's `dispatch_run_id`, and the core
record. The captured stdout and stderr are kept as a content-addressed artifact
under `<store>/artifacts/`, named by the evidence `output_ref`, from which the
evidence `output_digest` can be recomputed. It credits no criterion and authorizes no launch; launch
reconciliation of those ids is Gate D.

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

These gates establish only that a current host record exists for the named
unit. They do not make the host select an eligible route, validate the route
approval against the approved plan, or implement the host's full decision and
dispatch flow. Host decision, preflight, dispatch, and the implementation-authorization gate now bind their claimed route approval to the latest applicable approved core `ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION` transition for that same unit. It reads the currently loaded, chain-validated core run and compares the host `route_approval_digest` with the core approval's `TransitionDigest`. Missing approvals, another unit's approval, a digest mismatch, or a later unit-scoped/whole-run approved rewind to Route Selection or an earlier gate fail closed. This is a bounded approval-digest binding only: core approval records do not persist approved provider, model, profile, capability, or permission fields, so host route telemetry cannot yet be proven equal to those approved route facts. The broader MVP item remains open in `TODO.md`. Persisted core transitions do not list dependent units, so the lookup is conservative: any unit's approved return to Grounding, Acceptance criteria, or Planning invalidates every unit's route approval until a fresh Route Selection approval (Gate B, B10). Plan-derived dependent scope is Gate D work. Review preflight and dispatch also carry the same plan-unit route-approval digest as implementation, not a separate approved reviewer-route identity. Neither behavior proves the chosen model, profile, or permission envelope was human-approved. Learned decision-provider route choice, result-sufficiency,
next-step, and discretionary-idea judgments remain optional post-MVP host
extensions; see architecture §12.5.

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

Appends take the run lock `<store>/host/<run_id>.lock` (the core store's lock
discipline), write a `head.pending` intent, create the numbered record
atomically, then move `head`. `Load` completes an interrupted append whose
intent equals the verified chain tail and discards any other intent. A head
that disagrees with the chain without a matching intent is still
`HOST_HEAD_MISMATCH` (Gate B, H3).

Every host command and both coding gates refuse a run whose core chain records
a terminal outcome (`HOST_CORE_RUN_TERMINAL`, Gate B, H4). The Implementation
review -> Verification gate also re-checks the active route approval and
requires the review's `artifact_digest` to equal
`ReviewArtifactDigest(produced_head_digest, diff_digest, expected_worktree_digest)`,
where the worktree part is the host-declared dirty content that the core
transition checks against the live worktree, not the approval's optional
worktree field (`HOST_REVIEW_ARTIFACT_MISMATCH`, Gate B, N1/M4/B1), so one
review cannot stand for a different artifact.

A review dispatch declares `risk`, and the host sets `disputed` and
`final_validation` when the review resolves a disputed finding or is the final
validation (architecture §17). If any of these is true (risk high or critical,
disputed, or final validation), the reviewer must differ from the implementer
in provider family and model family. Otherwise a different worker, model, and
work context is enough (`HOST_REVIEWER_NOT_INDEPENDENT`, Gate B, M3/R5-02).

Lock order is host run lock, then core run lock. Host commands check and
append under both: the core lock is released only after the host record,
`head`, and run directory are durable, so a core terminal outcome or rewind
either precedes the check or follows the commit (Gate B closure, R4-01).
Before acknowledging, an append syncs every directory a store path
component is looked up in, including the directory of each intermediate
symlink hop (R4-02). `hma transition` holds the host lock from its host gate
check through the core append, so no host record can land in between.

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
- coding-stage transition enforcement;
- unit-scoped and whole-run stale route approval rejection (including earlier-gate rewinds);
- forged route approval rejection at decision, preflight, and dispatch entrypoints; and
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
