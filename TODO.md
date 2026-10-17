# HMA TODO

This file tracks implementation work that has not yet been folded into the normative architecture contract. `docs/architecture.md` remains the product-contract source of truth.

## Interactive orchestration and independent review

These tasks implement `docs/architecture.md` §12.4. They belong to the host
integration layer; HMA core remains non-dispatching in v1. `hma host ...`
provides host-local records and partial transition gates. It does not complete
interactive-host MVP conformance: the host still needs to establish approved
route/task bindings and perform the orchestration sequence, and HMA does not
enforce every direct-cost exception condition in §12.4.

### P1 — HMA-side record and gate subset (implemented; MVP conformance remains open)

- [x] **Add a versioned orchestration-decision record.** Record `DELEGATE` or
  `DIRECT_COST_EXCEPTION`, the bounded job, estimated direct and delegated total
  costs, comparison basis, approved permission envelope, decision timestamp,
  and actor. Reject an unrecorded direct-coding path.
- [x] **Define and record the route preflight handshake.** Before each dispatch,
  send `hi` to the exact approved profile through the approved access service
  and bind the response to profile/provider/model/access-service digests and a
  freshness timestamp. HMA records and verifies the host-observed handshake; it
  does not send the network/client message itself.
- [x] **Fail closed on unavailable or indeterminate routes.** An absent response,
  route mismatch, rate limit, or exhausted quota must prevent dispatch and
  return to an explicit superseding route decision or human route decision.
  Never substitute silently.
- [x] **Emit dispatch telemetry before work starts.** Report worker identity,
  provider, model, reasoning/effort level, access service, runtime, bounded
  task, and permission envelope.
- [x] **Add a mandatory independent-review binding for coding work.** Source,
  test, executable-configuration, build, and CI changes cannot progress from
  implementation review to verification without a reviewer record from a
  distinct agent and separate review context. Treat direct work by the
  orchestrator as implementation for this rule.
- [x] **Enforce independence by risk.** Require a different model for normal
  coding work and the §13 provider/model-family separation for high-risk and
  critical work; prohibit self-review and silent reviewer substitution.
- [x] **Add conformance fixtures.** Cover delegated work, a valid direct-cost
  exception, an unjustified direct path, successful and failed `hi` preflights,
  quota exhaustion, route mismatch, escalation, orchestrator self-review,
  same-model review, high-risk same-provider review, fixed-packet mismatch,
  valid independent review, coding-stage enforcement, and host-record tamper
  detection.

See `docs/task14-host-orchestration.md` for the executable contract and first
Fury pilot sequence.

Open before interactive-host MVP conformance:

- [x] Bind host decision, preflight, dispatch, and implementation-transition claims by unit and route-approval digest to the latest applicable approved HMA `ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION` record; reject absent, mismatched, wrong-unit, superseded, or earlier-gate-rewound approvals.
- [ ] Complete binding of task, route eligibility, capabilities, permissions, and host route telemetry to approved HMA plan/route state. Core approvals do not persist approved provider/model/profile/capability/permission fields, so this subset cannot prove those bindings; a non-empty digest alone remains insufficient.
- [ ] Persist and enforce affected-and-dependent-unit invalidation scope for plan-unit rewinds. Current core stage-transition records identify the changed unit but do not persist the dependent unit set, so a dependent unit can retain a stale route-approval digest after a dependency-invalidating plan change. Do not count this as full unit/route approval conformance.
- [ ] Enforce every §12.4 direct-cost exception condition: bounded/small task,
  approved permission envelope, route eligibility, and a total-cost comparison
  that includes setup, duplicated context, execution, and review coordination.
- [ ] Complete and exercise the host flow around the HMA records: route choice
  after deterministic eligibility checks, real exact-route preflight before
  every dispatch, artifact return, fixed review packet, and explicit human
  escalation when a route or result is indeterminate.
- [x] Preserve freshness across core-run changes. Transition gates reject
  host evidence bound to an earlier core-chain head.
- [ ] Add client-neutral adapter hooks and conformance coverage for an
  interactive CLI host; the HMA core must remain non-dispatching.

### Gate B — 2026-10-14 audit defect remediation (closure r7 PASS, committed 527b7c5)

Plan: `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md`; design:
`docs/handoffs/2026-10-15-gate-a-v3.md` to `-v7.md`. Each item has a failing-first
regression test (`*/gateb_regression_test.go` and named tests).

- [x] H1 CI verifies against the projected stage and the approval for leaving it.
- [x] H2 pilot compares head/diff only at head/diff-binding stages.
- [x] H3 host store: run lock, `head.pending` intent, crash recovery, concurrent appends.
- [x] H4 host commands and coding gates refuse a terminal core run.
- [x] M1 CI evidence store must be outside the repository.
- [x] M2 `hma resolve` refuses trailing JSON documents.
- [x] M3 strictest applicable independence rule; high/critical needs provider and model-family separation.
- [x] M4 review -> verification gate re-checks the active route approval.
- [x] M5 dogfood marker bound to canonical scratch store and run; seed run must be untouched.
- [x] M6 dogfood refusals compare store byte fingerprints.
- [x] N1 review must name the exact artifact (head, diff, worktree) its gate approval binds.
- [x] N2 CI evidence requires a clean Git checkout at approved base/head/diff before and after the command.
- [x] L1 direct route-attestation structural tests.
- [x] L3 CI runs the dogfood harness.
- [x] Round-1 audit (Liz, FAIL) remediation: B1 review artifact binds host-declared dirty content;
  B2 snapshots refuse assume-unchanged/skip-worktree entries; B3 CI refuses repository-selecting
  variables and evidence commands run without them; B4 CI refuses worktree-bound approvals;
  B5/B6 host commands check and append under the host run lock against one core snapshot and
  re-check terminal state after append; B7 directory fsync after every rename/link/unlink;
  B8 every applicable independence rule binds; B9 synthetic seed must match seed-core exactly.
- [x] Round-2 audit (Liz, FAIL) remediation: B10 any unit's approved return to Planning or earlier
  invalidates every unit's route approval (fail closed; Gate D narrows with plan-derived dependents);
  B11 worktree digest v2 binds file type and executable mode (v1-bound approvals fail closed);
  B7 parent directory synced even when the directory already exists.
- [x] Round-3 audit (Liz, FAIL on B11 only) remediation: worktree digest v3 binds Git's owner-execute
  bit exactly (0610 vs 0710 differ; group/other execute ignored); v1/v2-bound approvals fail closed.
- [x] Round-4 audit (Liz, FAIL on new B12) remediation (Lennon chose option 1): every Git command HMA
  runs forces change-revealing config (core.fileMode/symlinks/checkStat/trustctime, fsmonitor and
  untrackedCache off, autocrlf off, all untracked files, no external diff); diff uses --no-ext-diff
  --no-textconv; snapshots refuse configured clean/process filters; worktree digest v4 (v1-v3 fail closed).
  Residual under decision T: .gitattributes eol normalization and a deliberate stat-preserving
  rewrite are not detected. (Superseded by round 5 below.)
- [x] Round-5 audit (Liz, FAIL: B13 eol/attribute conversion, B14 submodule ignore) remediation (Lennon
  chose option 2): change detection no longer uses git status. internal/repostate/raw.go hashes each HEAD
  entry's raw worktree bytes as a Git blob (no conversion) and compares type, owner-execute bit, and object
  id; adds staged index differences and untracked files (committed .gitignore only; core.excludesFile and
  info/exclude ignored; untracked .gitignore always a change). Submodules and sparse checkout are refused.
  Worktree digest v5 (v1-v4 fail closed). Configured filters are no longer refused (they cannot hide a
  change from raw comparison). Repositories whose checkout converts bytes (eol, smudge, LFS) read as dirty.
  Accepted under decision T (Liz r5): deliberate size+mtime-preserving rewrite of ignored files is out of
  scope; ignored files remain out of scope.
- [x] Round-6 audit (Liz, FAIL: B15 core.ignorecase hid an untracked case alias; B16 subdirectory root)
  remediation: core.ignoreCase=false forced on every HMA Git command; snapshots refuse a root that is not
  the work tree top level.
- [x] Round-7 audit (Jax standing in for Liz, FAIL: B17 replacement refs, B15 v5 digest not retired, B18
  relative top-level root refused) remediation: every HMA Git command runs with --no-replace-objects;
  worktree digest v6 (v1-v5 fail closed); top-level check resolves the root to an absolute path first.
- [x] Round-8 audit (Liz, FAIL on B17 residual) remediation: evidence commands run with
  GIT_NO_REPLACE_OBJECTS=1; review artifact digest schema v2 retires reviews whose clean identity predates
  the replacement-object fix.
- [x] Blind closure review (Liz, FAIL) remediation: F01 worktree digest v7 binds the staged index entry of
  each changed path; F02 host commands hold the core run lock then the host lock from check through append
  (a racing core commit is ordered after the host record); F03 every host append syncs all ancestor
  directories; F04 diffs never ignore submodules (--ignore-submodules=none, diff.ignoreSubmodules=none).
- [x] Blind closure review r2 (Liz, FAIL) remediation: C01 a changed/added/deleted .gitignore disables
  ignore rules for untracked enumeration; C02 ancestor sync climbs the absolute path; C03 host review import
  refuses a revoked route approval.
- [x] Blind closure review r3 (Liz, FAIL) remediation: R3-01 hma transition holds the host run lock from
  the host gate through the core append (single lock order: host, then core); R3-02 ancestor sync also
  climbs the symlink-resolved path; R3-04 task14 describes the conservative invalidation and lock order.
- [x] Blind closure review r4 (Liz, FAIL) remediation: R4-01 host commands commit the host record inside
  the core run lock (`host.Store.Transact`), so no core terminal or rewind lands between check and durable
  append; R4-02 ancestor sync resolves the store path per component and syncs every lookup directory,
  including intermediate symlink hops.
- [x] Blind closure review r5 (Liz, FAIL) remediation: R5-01 a tracked path under a symlinked or
  non-directory parent is absent (`lstatInTree`, Git's "beyond a symbolic link" rule) in both the change
  scan and the digest; worktree digest domain v8 retires v7. R5-02 fixtures cover disputed and final
  validation family separation at low risk; task14 documents the host mapping.
- [x] Blind closure review r6 (Liz, FAIL) remediation: R6-01 review artifact domain advanced to
  `hma.host.review-artifact/v3`, retiring clean and dirty v2 reviews taken before the tracked-parent
  correction; replay regression with a reconstructed v2 clean review.
- [ ] F05/C04/R3-03 plan-derived dependent invalidation (deferred to Gate D by Lennon, 2026-10-16;
  conservative all-units invalidation stays in force until then) (current rule is conservative: any unit's return to Planning
  or earlier invalidates every unit's route approval): Gate D.
- [x] Independent Gate B re-audit (Liz, closure r7 PASS) and commit (527b7c5).

### Gate C — review provenance and finding import (scope: docs/handoffs/2026-10-16-gate-c-scope.md)

- [x] C1 `hma host review-packet`: HMA-generated packet, nonce, review dispatch id, expected launch digests.
- [x] C2 `hma host review-import`: receipt and report checks, total verdict map, final isolation rule.
- [x] C3 `findings_snapshot` core field; `hma host findings-import` with §9 import mapping.
- [x] C4 fixtures: failure -> remediation -> re-verification, stale and rewritten reports, same-run reuse.
- [x] Gate C r1 (Liz FAIL, 13 findings; Argie extra review) remediation: artifact derived from HMA snapshot,
  1h review TTL, receipt provider/effort/root, latest-packet-once, run-wide packet mode, current-core-head
  imports, final-audit family separation, independence against every implementation, report contract,
  advisory-only imported findings, per-unit no-downgrade snapshots, `hma host loop-evidence` bridge.
- [x] Gate C r2 (Liz hit usage limit; Jax backup FAIL, A01-A09) remediation: full report-contract port with
  parity test, live-target recheck at both imports, explicit finding ownership plus legacy migration, packet
  carries bound core state, loop envelope/store/hook checks under the host lock, enforcement-mode label.
- [x] Gate C r3 (Jax FAIL) R3-01: a newer review packet supersedes older approvals at the gate and for
  findings snapshots (`AnswersLatestReviewDispatch`). R3-02 (packet lacks criterion definitions, non-goals,
  budgets, required evidence, policy) and R3-03 (boundary) need the Gate D acceptance contract / Lennon.
- [x] Gate C final audit (Liz, FAIL) C-F14: `hma host dispatch` refuses a caller-supplied packet binding;
  C-F15: loop re-capture keeps stdout/stderr as a content-addressed store artifact (`output_ref`).
- [ ] Reservations (`hma host attempt`, admission, launch reconciliation) moved to Gate D with leases;
  exit tests listed in the scope note §0.1. Boundary accepted by Lennon 2026-10-17.
- [ ] Independent Gate C audit (Liz) and commit.


### Post-MVP — learned decision-provider integration (optional)

This work is intentionally outside the MVP acceptance boundary. The MVP host
may make soft route/result/next-step judgments itself or return them to a human,
provided deterministic HMA legality, approval, permission, and independence
rules remain authoritative. Keep the provider-neutral seam so a learned
decision provider can be added later without changing portable core semantics.

- [ ] **Add a provider-neutral decision trace schema.** Bind provider/model
  identity, state digest, typed question contract, probabilities/confidence,
  threshold-policy version, selected host action, and any override rationale.
- [ ] **Implement an optional Jev host adapter.** Support an approved Jev MCP/SDK/HTTP
  path without adding provider-specific credentials or endpoints to portable
  HMA core.
- [ ] **Optionally use Jev at the three execution checkpoints.** Pre-dispatch route choice,
  post-worker result sufficiency, and pre-next-step continue/retry/replan/
  escalate/stop. Deterministic HMA eligibility and legality checks run first.
- [ ] **Add discretionary-idea triage.** Agent-generated optional implementation
  ideas should be classified as implement/prototype/defer/reject before they can
  expand scope; an explicit human request is not vetoed by this gate.
- [ ] **Calibrate confidence policy from recorded outcomes.** Until calibrated,
  low-confidence or close Jev decisions escalate rather than force an automatic
  soft judgment.
- [ ] **Add Jev failure fixtures.** Missing tool, timeout/rate limit,
  indeterminate answer, low confidence, and attempted illegal-choice expansion
  must preserve hard policy and return control to the orchestrator/human.
- [ ] **Keep any learned provider advisory.** A provider result cannot approve a transition, waive a
  rule, satisfy evidence, replace independent review, widen permissions, or
  authorize commit/push/release.


### P2 — host adapters

- [ ] **Implement client-neutral adapter hooks** for orchestration decision,
  preflight, dispatch reporting, artifact return, and independent-review return.
  Codex, Claude Code, OpenCode, Cursor, Hermes/Fury, and other clients may
  translate these hooks but may not change their semantics.
- [ ] **Measure estimated versus observed dispatch cost** so the direct-cost
  exception can be calibrated without becoming a convenience bypass.

## Reverse-skill-inspired routing hardening

Reference reviewed: `zhaoxuya520/reverse-skill` (`AGENTS.md`, 2026-08-30 review). Useful patterns are its routing single source of truth, routing regression benchmark, routing-coherence checks, generated tool inventory, and client-neutral adapters. HMA should adopt the patterns, not the security-specific skill framework or bootstrap behavior.

These items extend HMA's existing route-selection, host-local profile, evidence, independence, and deterministic-fixture contracts. They should not create a second routing system.

### P1 — before route selection is considered implemented

- [x] **Define one versioned route-policy data contract / normalized projection as the routing SSoT.** It should operate on HMA's portable capability taxonomy and policy/risk inputs, not hard-code provider-specific client names, credentials, private paths, or local aliases. A route-policy change must produce a new digest.
- [x] **Add table-driven routing conformance fixtures.** Each row should bind plan-unit/task facts, risk, required capabilities, available/eligible host profiles, permission envelope, and relevant policy to exactly one expected primary route, one escalation route, validator-independence requirement, and forbidden fallbacks. Include ambiguous, unavailable-capability, high-risk, and no-eligible-route cases.
- [x] **Add a deterministic route-coherence validator.** At minimum verify: referenced capabilities exist; exactly one primary and one escalation route are proposed; escalation is independently eligible; implementer cannot validate itself; required model/provider-family independence is satisfied; permission bounds are not widened; and no silent fallback or access-service substitution is possible.
- [x] **Implement a host-local machine-truth inventory from allowlisted discovery only.** (`hma inventory`; not yet consumed by route verification.) Capture executable presence, version, model-list/availability metadata, timestamp/freshness, and a digest. Keep it credential-free and outside portable repository artifacts. Preserve the architecture rule that availability is evidence only and is not proof of eligibility, trust, isolation, quota, or capability.

### P2 — after the deterministic evaluator/harness nucleus exists

- [ ] **Make routing regression part of the deterministic conformance suite and CI.** A routing-policy change should fail existing expected-route fixtures unless the changed expectations are reviewed explicitly.
- [ ] **Define a client-adapter boundary for Codex, Claude Code, OpenCode, Cursor, and other hosts.** Adapters may translate HMA's approved route/permission attestation into host-specific invocation, but client behavior must not become routing policy or alter evaluator semantics.
- [ ] **Add route reproducibility checks.** The same approved harness contract + policy digest + machine-truth snapshot must produce the same route proposal and rationale classification.
- [ ] **Add negative fixtures for stale or contradictory machine truth.** Missing/stale discovery, profile-policy mismatch, or unavailable independent validator must produce an explicit `BLOCKED`/`UNKNOWN`/human-decision path rather than a guessed or silent substitute.

### Integration with existing HMA work

- Treat routing conformance as an extension of the deterministic evaluator and end-to-end decision-table work identified in audit Findings 12, 15, and 16; do not implement it as a separate prompt-driven router.
- Reuse Section 12's portable capability taxonomy and host-local profiles instead of creating a second tool/profile schema.
- Reuse HMA's existing Evidence -> Finding -> Next Action -> Verification/Validation chain rather than importing reverse-skill's case/evidence object model.
- Keep human approval and HMA's authority boundaries authoritative. Routing may recommend; it must not dispatch, widen permission, waive independence, or advance a stage in v1.

### Explicit non-goals

- [ ] Do **not** import reverse-skill's security-specific skill catalog, ACT authorization semantics, global/bootstrap prompt injection, or automatic tool installation into HMA core.
- [ ] Do **not** make a generated tool index or model list a trusted eligibility source.
- [ ] Do **not** duplicate routing truth across Markdown instructions, adapters, and executable policy data.

### Done when

Routing hardening is complete for the MVP when table-driven fixtures prove that a fixed harness/policy/machine-truth input deterministically yields one route proposal, one escalation path, the required independent validator, bounded permissions, and an explicit no-route outcome when eligibility cannot be proven.
