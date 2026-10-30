# Architecture amendment draft — Gate A approved design

**Date:** 2026-10-16
**Dispatch ID:** hma-arch-amendment-draft-20261016-r2
**Scope:** normative amendment text for `docs/architecture.md` reflecting Lennon direction D1 and the trusted-host decision T, as designed in `2026-10-15-gate-a-v3.md` through `v7.md` and audited in `audit-r7.json`.
**Status:** MERGED into `docs/architecture.md` on 2026-10-30, commit `e583f4a8cd8345f71146104f9d76d635d4d15dfb`.
See "Status and acceptance record" at the end of this document.

## 1. Sections to be inserted or replaced

### 1.1 §6 — Human approval contract

Replace the bulleted binding list and the paragraph that follows it with:

Every approval is single-use and universally bound to:

- run identifier;
- transition digest;
- current stage;
- Proposed target (stage or terminal outcome). Exactly one of `proposed_target_stage` or `proposed_target_outcome` must be non-empty. The transition digest covers the populated target. The additive `proposed_target_outcome` field is omitted when empty, preserving existing approval serialization and digests.
- `acceptance_contract_digest` when the run uses `hma.acceptance-contract/v1`; omitted when empty. [CHANGES §6] This field is required on the `ACCEPTANCE_CRITERIA -> PLANNING` transition when a contract is in use, and it is carried by every later approval of that run. [CHANGES §6] A transition is refused when the presented contract digest differs from the bound value. [CHANGES §6]
- repository-identity digest;
- exact base-revision digest;
- stage-time digest (predecessor chain head and monotonic sequence);
- accepted plan digest;
- plan-declared required evidence digest set;
- active waiver-operation digest set;
- short-lived challenge nonce;
- approving actor and timestamp.

Implementation review, verification, independent validation, and release and
closure approvals additionally bind the produced head revision and diff digest.
Other transitions do not bind a produced head or diff. A packet becomes stale
only when a field it actually binds changes; CI applies the same rule.

An approval may additionally bind the exact worktree-content digest of an
approved dirty worktree (amendment, 2026-09-08). The binding is optional at
every stage. When it is populated, the transition is refused if the current
worktree digest differs from the approved digest — including when the worktree
is now clean, since a bound digest is never empty. When it is absent, a dirty
worktree remains governed by the explicit dirty-worktree approval of the exact
content, exactly as before; the amendment adds a binding, never a requirement.

A change to any actually bound field invalidates the approval.

The approval interface is challenge-bound rather than a simple `y/N` prompt. Exact CLI syntax remains an implementation decision and is intentionally not invented in this architecture document.

---

### 1.2 §9 — Acceptance-criteria contract

Insert after the first paragraph of §9 ("Acceptance criteria are co-authored:"):

A run may use a versioned acceptance contract, `hma.acceptance-contract/v1`. [CHANGES §9] The contract binds the approved acceptance criteria, non-goals, risk class, permitted routes, permission envelope, and immutable policy and contract digests. [CHANGES §9] A revision to the contract invalidates affected approvals and evidence, and requires human reauthorization where existing HMA policy demands it. [CHANGES §9] The contract digest is bound to every approval after `ACCEPTANCE_CRITERIA -> PLANNING` as `acceptance_contract_digest` (see §6). [CHANGES §9]

Retain the remaining text of §9 unchanged.

---

### 1.3 §10 — Plan-unit contract

Replace the second and third paragraphs of §10 ("Each plan unit records:" through "explicit non-goals.") with:

Each plan unit records:

- one observable behavior or bounded enabling change;
- acceptance criteria advanced;
- exact allowed files and actions;
- exact forbidden files and actions;
- expected failing test or pre-change evidence;
- minimum intended implementation;
- existing code or dependency reuse analysis;
- complexity, file, diff, time, tool-step, and retry budgets;
- proposed implementer route;
- proposed escalation route;
- proposed validator route and independence requirement;
- verification commands and required evidence;
- dependencies on prior units;
- stop, rollback, and escalation conditions;
- explicit non-goals.

The approved plan is encoded as `hma.plan/v1`, a canonical JSON document. [CHANGES §10] Its digest is the existing `accepted_plan_digest`. [CHANGES §10] The document contains the unit fields above plus a required `policy_digest` (digest of the effective kernel→organization→project→task policy projection), per-unit cost, time, and attempt ceilings, and the dependency graph among units. [CHANGES §10]

A unit's lease derives from the unit's latest approved core transition whose target is `IMPLEMENTATION_AUTHORIZATION`. [CHANGES §10] The only legal incoming edges for such a transition are `ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION` and `INDEPENDENT_VALIDATION -> IMPLEMENTATION_AUTHORIZATION` (see §5.4). [CHANGES §10] The host presents the `hma.plan/v1` document; HMA recomputes its digest and requires equality with that approval's `accepted_plan_digest` and that the unit exists in the plan. [CHANGES §10] A host-written lease record only caches this derivation and grants no authority by itself. [CHANGES §10]

The lease is active only while all of the following hold: the core run is not terminal; the unit's projected stage is `IMPLEMENTATION_AUTHORIZATION`; no later approved rewind of the unit, the run, or a dependency has occurred; wall time since approval is below the unit's `max_wall_minutes`; per-unit counters are below the declared ceilings; the current computed policy digest equals the plan's `policy_digest`; and the current contract digest equals the bound `acceptance_contract_digest`. [CHANGES §10] A new lease after an approved rewind belongs to a new generation; spent counters carry forward, and reservations from an earlier generation are not revived. [CHANGES §10]

Each lease generation, reservation, and review dispatch has a `valid_from` (record time) and a `valid_until`. [CHANGES §10] The value is:

    valid_until = min(original_expiry, t_r for every applicable revocation event r)

Applicable revocation events are any core rewind affecting the unit or a dependency, terminal outcome, policy or contract digest change, and the lease wall-time end of the record's generation. [CHANGES §10] The value is computed from the whole chain at import, so ingestion order does not matter; it can only decrease as events are added. [CHANGES §10] A launch at time `t` is authorized only if `valid_from <= t < valid_until`; equality with `valid_until` is unauthorized. [CHANGES §10]

Retain the remaining text of §10 unchanged.

---

### 1.4 §12.4 — Interactive host orchestration

Insert after the preflight and worker-identity paragraph ("Before dispatch, the host also reports and records the worker identity...") and before the independent-review paragraph:

#### Reservation protocol

Before each worker launch, including after resume, the host calls the HMA reservation hook under the host per-run lock. [CHANGES §12.4] The call supplies the controller run identifier, attempt index, pre-allocated dispatch run identifier, envelope digest, exact route JSON, the last failure signature when applicable, the live repository root, the clone baseline digest, and the ledger namespace digest. [CHANGES §12.4]

HMA processes the call in this order:

1. **Replay first.** If a reservation exists for (`controller_run_id`, `attempt_index`, `dispatch_run_id`) with the same request digest, HMA returns it unchanged — even if the lease has since ended or counters are exhausted. [CHANGES §12.4] A matching key with a different request digest is refused (`HOST_RESERVATION_CONFLICT`). [CHANGES §12.4]
2. **Admission (first call per controller run).** If no `loop_admission` record exists for this `controller_run_id`, HMA requires an active lease, the envelope's `escalate_to` value to be `"stop"`, the envelope route to equal the lease route, and the envelope's `max_attempts`, `max_minutes`, and spend cap each to be less than or equal to the lease's remaining amounts at this moment. [CHANGES §12.4] HMA appends a `loop_admission` record binding the controller run identifier and envelope digest. [CHANGES §12.4] Admission consumes nothing. [CHANGES §12.4]
3. **Consumption (each attempt).** HMA requires the admission record with the same envelope digest, an active lease, at least one remaining attempt, and the failure-signature rule from §17. [CHANGES §12.4] HMA appends one `reservation` record and counts it immediately. [CHANGES §12.4] Envelope-vs-remaining is not rechecked here. [CHANGES §12.4]
4. An unknown launch stays counted, and there is no cancellation path. [CHANGES §12.4] Counters are per unit and monotonic for the run; a new lease carries the same totals, and spent attempts are never reset. [CHANGES §12.4]

The hook response contains a `reservation_id` valid only for that `dispatch_run_id` and expiring at lease end. [CHANGES §12.4] It also contains `launch_eligible`: `true` only when the lease is still active by time, stage, policy, and contract and no ledger run with that `dispatch_run_id` exists yet; otherwise `false`. [CHANGES §12.4] The host must refuse launch unless `launch_eligible` is `true`. [CHANGES §12.4]

#### Launch classes and import reconciliation

Every ledger run in the declared namespace for the run's lifetime is classified by exactly one HMA record: [CHANGES §12.4]

- an implementation launch matches one `reservation`; [CHANGES §12.4]
- a review launch matches one `review_dispatch` record that HMA appends when it generates the review packet, binding `review_dispatch_id`, `dispatch_run_id`, packet digest, target, required reviewer independence, and issuing reason. [CHANGES §12.4]

A ledger run with no matching record, or matching two records, is `UNAUTHORIZED_DISPATCH` and produces a `BLOCK` kernel finding. [CHANGES §12.4] A `review_dispatch` whose receipt shows write mode (`mode != review`) or whose `dispatch_run_id` also appears in an implementation reservation is unauthorized. [CHANGES §12.4]

At evidence import HMA scans all ledger runs in the declared namespace from the run's first core record to import time. [CHANGES §12.4] For each ledger run at time `t`: [CHANGES §12.4]

- no matching record → `UNAUTHORIZED_DISPATCH` (BLOCK); [CHANGES §12.4]
- matching record but `t < valid_from` or `t >= valid_until` → `UNAUTHORIZED_DISPATCH` (BLOCK); [CHANGES §12.4]
- matching record and `valid_from <= t < valid_until` → authorized. [CHANGES §12.4]

A reservation with no matching ledger run stays counted and is listed as `unlaunched`. [CHANGES §12.4]

For HMA-bound verify-loop runs, the journal envelope digest must equal the `loop_admission` envelope digest, and the journal controller run identifier, live root, clone baseline digest, and ledger namespace must equal the admission's values. [CHANGES §12.4] Every journal attempt with a `dispatch_run_id` must have a reservation with that identifier; otherwise the import fails and records `UNAUTHORIZED_DISPATCH` (BLOCK). [CHANGES §12.4]

Retain the remaining text of §12.4 unchanged.

---

### 1.5 §14 — Evidence acquisition and record

Insert before "Evidence from a clean bound revision...":

#### Review import eligibility

HMA generates review packets itself and records each packet's digest before dispatch. [CHANGES §14] The packet contains the target, contract, findings snapshot, bindings, a fresh random 128-bit `review_nonce`, and the `review_dispatch_id`. [CHANGES §14]

A review result imports only when all of the following hold, in addition to the independence requirements in §13: [CHANGES §14]

- the receipt's `packet_sha256` equals the HMA-generated packet digest; [CHANGES §14]
- the receipt's `head` and `worktree_sha256` equal the bound target; [CHANGES §14]
- the receipt's `dispatch_run_id` equals the `review_dispatch` record's identifier and is in the declared namespace; [CHANGES §14]
- the receipt's `mode == review` and the terminal state is `completed` with exit code 0; [CHANGES §14]
- each imported report file's SHA-256 is in the receipt's `report_files`; [CHANGES §14]
- the imported report JSON contains `packet_sha256`, `review_dispatch_id`, and `review_nonce` equal to the packet's values; [CHANGES §14]
- the reviewer provider and model satisfy §13 independence against the implementer reservations for the unit; [CHANGES §14]
- the receipt's ledger run is not one of the unit's implementation reservations. [CHANGES §14]

The verdict map is total: `pass` maps to APPROVE; `fix-required` maps to REQUEST_CHANGES; `conditional-pass`, `blocked`, `unknown`, `not-evaluated`, missing, or any other value maps to no approval and next action `ESCALATE_TO_HUMAN`. [CHANGES §14]

Retain the remaining text of §14 unchanged.

---

### 1.6 §16 — Independent validation contract

Insert after the list of things the validator may evaluate:

#### Findings snapshot and disposition

Core `finding` records gain an additive `findings_snapshot: true` flag, omitted when false. [CHANGES §16] With the flag, the record's `Findings` list replaces the previous snapshot even when the list is empty. [CHANGES §16] Without the flag, a record with no findings does not change state. [CHANGES §16] An empty snapshot is accepted only from an import bound to an independent review of the exact current target. [CHANGES §16]

Severity never sets disposition. [CHANGES §16] The rules are:

- a kernel (§7.1) violation with reproduction → `BLOCK` (unwaivable); [CHANGES §16]
- evidence that a contract criterion's check fails → no new finding class; the criterion is not credited (`FAILED`/not verified) and closure is blocked through the criterion itself; [CHANGES §16]
- measurable plan complexity-budget exceedance → `WAIVABLE` (§7.2); [CHANGES §16]
- everything else, any severity → `ADVISORY`. [CHANGES §16]

A claimed criterion link must name a criterion identifier in the contract and include a reproducible check; HMA re-runs it. [CHANGES §16] A fabricated or non-reproducing link is recorded as `ADVISORY`. [CHANGES §16]

#### Checkpoint triggers

Mandatory independent validation checkpoints are triggered by: a modified acceptance criterion or permission boundary; high or critical risk changes; public interface or persisted schema or protocol change; irreversible or external effect; security, approval, routing, evidence, or closure logic change; completion of an approved material milestone; and final outcome closure. [CHANGES §16] A path or interface that cannot be classified is a mandatory trigger. [CHANGES §16] Disputed validation uses the high-risk independence rule from §13 (provider and model-family separation). [CHANGES §16]

Retain the remaining text of §16 unchanged.

---

### 1.7 §20 — Closure semantics

Append to the bulleted list:

- the latest findings snapshot, if any, has been applied and no unresolved `BLOCK` or `WAIVABLE` finding remains. [CHANGES §20]

Retain the remaining text of §20 unchanged.

---

### 1.8 §22 — Threat model

Append to §22 after the existing trusted computing base list:

### Trusted-host assumption

Decision T: for v1 HMA detects accident and drift by the host; it does not prevent a same-UID adversarial host. [CHANGES §22] Host omission of an HMA call is detected at import and blocks closure; it is not prevented at launch time by HMA. [CHANGES §22] Receipts are host-written; under T they are trusted for drift and accident detection only. [CHANGES §22]

Retain the remaining text of §22 unchanged.

---

## 2. Open questions

1. Exact JSON Schema shape for `hma.acceptance-contract/v1` and `hma.plan/v1` (field order, required/optional sets, canonical normalization) is deferred to schema-publication work.
2. The live-repository-root, clone-baseline-digest, and ledger-namespace-digest fields in the reservation hook are controller-supplied; their exact derivation and host-configuration binding are deferred to host-adapter design.
3. The `review_nonce` generation, storage, and expiry mechanism are specified only as a 128-bit fresh random value; concrete entropy source and rotation policy are deferred.
4. The exact mapping between `hma.plan/v1` per-unit ceilings and verify-loop envelope fields (`max_attempts`, `max_minutes`, spend cap) is deferred to host-adapter implementation.
5. The mechanism that computes `valid_until` from the whole core chain at import time, including tie resolution and clock-resolution assumptions, is deferred to implementation.
6. The checkpoint trigger list is qualitative; the deterministic detector that classifies a change as one of the listed triggers is deferred to implementation.
7. The `agent-dispatch` receipt field set (D6) and the verify-loop `authority_hook` adapter (D5) live in the Obsidian host layer; their exact integration contracts are out of scope for this architecture amendment.


## Status and acceptance record (2026-10-30)

- Merged into `docs/architecture.md` at commit `e583f4a8cd8345f71146104f9d76d635d4d15dfb`. All eight
  named section edits (§6, §9, §10, §12.4, §14, §16, §20, §22) applied; three insertion points not
  fully pinned by the draft's literal wording (§9, §16, §22) were resolved by the orchestrator (Ming,
  Claude Sonnet 5) and independently reviewed.
- Final independent review: Prof, Codex/OpenAI, GPT-6 family (exact model ID and effort not
  disclosed by the runtime): PASS. All 78 `[CHANGES §N]` clauses verified verbatim in their target
  sections; all 51 trace-doc entries covered; no retained wording altered beyond one punctuation
  character (§20 bullet, period to semicolon, needed to extend the list); all three insertion-point
  judgment calls assessed as preserving normative meaning; no internal contradiction found; all
  cross-references (§6, §7.1, §7.2, §5.4, §13, §17) resolve; the heading-level skip under §14/§16
  confirmed cosmetic; none of the draft's 7 deferred open questions silently resolved. Evidence:
  `evidence/gate-d/audit-d6-arch-merge-prof.md`, `.json`, `merge-d6-arch.diff`.
- Independence: genuine provider/model-family separation from the orchestrator (Claude/Anthropic),
  who both authored this merge and independently re-verified it before dispatch. This is a
  documentation-transcription audit, not a code-behavior audit; it does not run the Go test suite,
  race suite, or host dogfood harness, since no executable code changed.
- Independence exception, accepted by Lennon on 2026-10-30 ("go"): no further review was required
  beyond the one cross-family audit, consistent with this being a mechanical documentation merge of
  already-approved, already-audited design text (`audit-r7.json`) rather than a new design decision.
- Open, unresolved by this merge, by design: the draft's seven deferred open questions (JSON Schema
  shapes, reservation-hook field derivation, `review_nonce` entropy/rotation, plan-ceiling/envelope
  mapping, `valid_until` reducer implementation, checkpoint-trigger classifier, D5/D6 host-adapter
  contracts) remain open. None was silently resolved by the merge; see §25 for the existing deferred-
  decision list if any of these should be added there later.

## Next action (§18)

No further action named by this amendment. The next architecture-roadmap step per the D5c handoff
and docs/architecture.md §21 is the CI release gate, not yet started.
