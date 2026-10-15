# Gate A v4 re-audit — FAIL

Dispatch: `hma-gate-a-audit-20261015-r4`. Date: 2026-10-15T17:24:08.206765+00:00.

HMA is frozen at `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81` on `main`, plus 105 dirty-tree files. Only v4 was added since the v3 freeze. Obsidian is at `9ba1e74a61ef5133f911903cd5c328666e2170ca`. The inspected Verify Loop controller, tests and skill match that commit. `.serena/` is excluded.

**Result:** R1 and R6 close at design level. The remaining requested items are partial. Two high findings remain. One new medium finding is open. Gate B must stay stopped. Code proof belongs to later gates.

**Isolation:** partial; this is a re-audit using v1–v3 reports, not a blind first pass. A historical controller commit message appeared during provenance inspection. Its assertions were not used as proof. The structured report records this context exposure. `PHASE1.json` freezes this assessment; there was no later unblinding.

**Threat model:** T = trusted host. HMA must detect host accident and drift at import and block closure. This audit does not require same-UID adversarial-host prevention. Both high findings have sequences that use visible genuine ledger records.

## 1. Prior finding closure

| ID | Status | Reason |
|---|---|---|
| R1 | CLOSED (design level) | V4 uses one target-based issuance rule over the two legal incoming edges. Review/Verification rework goes through human-approved Route selection; Independent validation can return directly to Implementation authorization. The retained active-stage, invalidation and monotonic-counter rules prevent old approvals from reviving after a rewind. No new stage or edge is introduced. |
| R4 | PARTIAL | Under decision T, import-time detection is an acceptable enforcement boundary. V4 adds the HMA marker, missing-hook refusal, namespace/baseline fields, stable response ID and ledger reconciliation. However, the scan excludes visible launches outside the lease window, including an accidentally late launch. No launch-time validity reconciliation is specified for expired or revoked reservations. This is a host-drift detection gap, not a demand for same-UID tamper resistance. |
| R6 | CLOSED (design level) | The required effective-policy digest now has a named location in hma.plan/v1. The existing accepted_plan_digest binds that document to a human approval. Current-policy mismatch voids the lease. Legacy runs without the new plan cannot obtain a lease under the retained fail-closed rule. Later code must canonicalize the effective hierarchy; the existing route-policy normalizer alone does not prove that full hierarchy is already implemented. |
| R7 | PARTIAL | The two specific algorithm defects are repaired: replay precedes capacity checks, and full-envelope admission occurs once. Five incremental reservations and an exhausted replay work in the literal model. The complete closure criterion also required no stale launch authority. Replay returns the old success unchanged even after the lease ends; the controller contract only requires a nonempty successful hook response. V4 does not specify current launch eligibility in that response or import rejection of a late launch. This residual is tracked once under R4. |
| R8 | PARTIAL | Packet, target and report-file hashes close the original unrelated-receipt/altered-report substitution path under T. Retained distinct-worker/context and model-family rules still forbid self-review. The written import predicates do not require receipt mode=review, successful terminal state, an exact expected review-dispatch run/namespace, or fresh production of a declared report file. The prior closure criterion explicitly required refusal of wrong-mode and failed-worker receipts. The verdict map also still omits conditional-pass, not-evaluated and an explicit unknown-value refusal. |
| A1 | PARTIAL | Initial/rework issuance, approved plan identity, dependent invalidation and policy binding now have coherent design rules. Host-written lease caches still grant no authority. R4 leaves a gap between reservation and actual launch plus incomplete late-launch detection. No self-issued initial lease path was found in the v4 design. |
| A2 | PARTIAL | The HMA marker and before-each-launch hook are specified future D5 work. Clone/live and namespace identity fields are now named; HMA re-capture remains required after transfer. The current committed controller has no HMA hook and normalizes away provider/profile/access-service fields. This is expected later implementation, not alone a Gate A failure. The proposed import rule still misses late launches (R4). Actual route resolution and current authority must be compared in the D5 acceptance tests. |
| A4 | PARTIAL | Cumulative unresolved findings, explicit independent per-ID resolution, reopened IDs and same-head fresh-evidence repair remain required. The snapshot flag remains a valid design repair. R8 still permits non-eligible receipt states to pass the named provenance predicates. A failed or misrouted review must not resolve a BLOCK. Verdict handling must be total and fail closed. |
| A6 | PARTIAL | Componentwise admission, monotonic unit counters, conservative unknown reservations and the repeated-failure rule remain sound directions. R7 repairs the two accounting/liveness contradictions. Late-launch reconciliation remains incomplete (R4). The new all-ledger-runs rule also conflicts with the retained exclusion of review launches from implementation attempts (R9). Unsupported paid/token ceilings still fail closed. |
| A8 | PARTIAL | The expanded matrix names the repaired issuance/policy/admission paths and simple packet/hash negatives. It still lacks explicit tests for visible launches before/after the lease window, revocation between reservation and launch, wrong-mode/failed reviewer receipts, stale pre-existing report files and legitimate review runs in the scanned namespace. The command package placeholder and the old journal/reservation equality row also remain to be reconciled. Execution belongs to later gates; missing runtime implementation is not the reason for this status. |

A5, R2, R3 and R5 remain CLOSED (design level). No code fix or human approval is inferred from any closure status. Exact prior closure criteria and source references are retained in `AUDIT.json`.

## 2. Approval and digest compatibility

No intrinsic conflict with section 6 or legacy digest compatibility, conditional on an exact normative amendment and implementation preserving omitted-field behavior.

The surviving proposal adds only acceptance_contract_digest to ApprovalBinding. V2 execution_lease_digest is dropped. V4 policy_digest lives inside the approved plan, not ApprovalBinding. findings_snapshot is a separate additive record flag.

The 2026-09-08 worktree amendment demonstrates optional additional binding: absent preserves old behavior; populated is enforced. A new optional omitted field can preserve exact old JSON bytes, field order and hashes. Existing legacy golden and worktree tests passed; these do not test future fields.

New contract-bearing runs must bind the acceptance contract at criteria approval and carry it downstream. Old approvals must remain valid for their original semantics; legacy runs cannot retroactively obtain new leases without a valid approved hma.plan/v1. A populated new field must participate in equality/hash/staleness checks.

Section 6 requires a concrete amendment for the new conditional binding. D1 approves direction and explicitly defers exact text. No new stage-approval channel is authorized. Existing earlier-stage head/diff rules must remain unchanged.

Older strict readers may reject newly populated fields. That is a forward-reader/versioning issue, distinct from unchanged historical digests. Document the new schema/reader requirements and retain golden negative controls at the assigned gate.

## 3. Concrete counterexamples and findings

These are specification counterexamples. The HMA attempt hook and review importer are proposed work. `probes.py` models the named rules and does not claim a runtime exploit of future code.

### R4 — Import reconciliation omits launches outside the lease window

**HIGH; inferred; Residual v3 closure gap.**

Observed/source evidence:

- `docs/handoffs/2026-10-15-gate-a-v4.md:12-14,55-82`: Host omission is promised detectable at import; scan is restricted to start time inside lease window.

- `probes.log; probes.py`: Literal v4 import filter detects an inside-window unreserved launch and ignores a visible after-expiry unreserved launch.

Diagnosis: V4 promises closure-blocking detection of an omitted HMA call, but only examines ledger runs whose start time falls inside a lease window. A visible late launch after expiry is excluded rather than classified unauthorized. Checking one-to-one IDs and the envelope digest also does not establish that a matching reservation was valid at the actual launch time. Replay intentionally skips current lease checks. The response supplies an ID and expiry, but the controller contract names only nonzero/missing-response refusal; current-authority handling after revocation is not defined.

Sequence: 1. Approve unit U with lease window [10,20). 2. A scheduler resumes late at time 21 and accidentally launches D without the HMA call. Keep the genuine ledger entry visible in the declared namespace. 3. Import the journal and ledger. D is outside [10,20), so the written scan never asks for its reservation. No hidden ledger or forged receipt is needed. Variant: reserve D before a human rewind, delay launch until after revocation, then import a matching ID; v4 does not require comparing actual launch time with authority history.

Expected: Detect every observed implementation launch associated with the HMA-bound controller/unit, including launches outside valid authority intervals. A late, revoked or unreserved launch must append a closure-blocking finding. Historical replay must not represent fresh launch permission.

Preferred fix (proposed): Define the declared namespace/controller set independently of lease validity. Enumerate all associated runs, then check each actual start against the reservation, approved unit/route/baseline, and authority interval. Separate historical reservation lookup from a current launch_allowed result. Require controller refusal on invalid current authority; at import detect and block any launch that slipped through after expiry/revocation.

Closure checks:

- Omit the hook and launch visibly after expiry, before issuance, and after terminal/rewind → Import records UNAUTHORIZED_DISPATCH and refuses acceptance in every case

- Reserve, revoke/expire before launch, replay, then reconcile → Historical reservation remains counted; it grants no new launch eligibility; any stale launch blocks closure

- Launch a valid reserved job and leave another reservation unlaunched → Valid launch imports; unused reservation stays counted without a false failure

Trade-offs: This is detection within the trusted-host boundary. It does not require preventing a hostile same-UID host or scanning undisclosed namespaces. Retain append-only counts for unknown launches.

Alternative: Initially use a run-owned HMA-bound namespace and require complete namespace reconciliation at every import/closure. Refuse indeterminate association until the host repairs it.

Effort: medium. Priority: Before Gate A design acceptance; implementation proof at the assigned later gate. No fixed revision exists.

### R8 — Hashed review files do not yet prove an eligible completed review

**HIGH; inferred; Residual v3 closure gap.**

Observed/source evidence:

- `docs/handoffs/2026-10-15-gate-a-v4.md:86-95`: The named import conditions compare packet/target/report digests and independence, but not mode, terminal success, expected dispatch run or fresh report production.

- `docs/handoffs/2026-10-15-gate-a-v3.md:128-138; internal/host/host.go:567-608`: Only pass/fix-required/blocked/unknown are mapped. EvaluateReview checks host dispatch bindings but receives no ledger terminal state.

- `AgentSystem/scripts/agent-run-ledger:315-345; probes.log`: Current ledger already has mode/state/exit_code. The literal new import predicates accept matching hashes with mode=write, state=failed and exit_code=1.

Diagnosis: V4 binds the exact prompt packet, target at launch and imported report bytes. That is a substantial repair under T. However, mode is collected but not checked, terminal receipt success is not required, the receipt is not explicitly equated to the expected HMA review-dispatch run and canonical namespace, and merely hashing declared paths does not prove the current reviewer produced them. Existing host checks validate a declared REVIEW dispatch and context, not the actual receipt mode/terminal state. A complete verdict map is also absent.

Sequence: 1. HMA records current frozen packet P. 2. The eligible independent route launches P, writes a pass report with explicit resolution of F1, then fails (or a host option drift launches the worker in write mode). The genuine receipt contains the correct P, head/worktree and report-file hash, plus a failed state/wrong mode. 3. The named v4 predicates still pass, so the pass verdict can become APPROVE and the apparent F1 resolution can clear a BLOCK. No host forgery is required. Separate stale-file variant: a failed reviewer leaves an old declared AUDIT.json untouched; hashing that path at exit authenticates its bytes, not its production by this review. This variant requires an importer that does not additionally bind the report body to the current packet; v4 does not specify that check.

Expected: Only a terminal successful read-only review of the current frozen packet, produced by the expected eligible review run, may approve, resolve findings or give criterion credit. Every other receipt state or report verdict must be explicitly non-approving.

Preferred fix (proposed): Require the exact expected review-dispatch ID and pinned ledger namespace, receipt mode=review, terminal successful state/exit status, the approved identity/context and all contributing implementers including direct work. Use a fresh run-owned output directory or capture the report directly from canonical output; reject pre-existing/unproduced declared report files. Bind report body/manifest to the current packet. Complete verdict mapping for conditional-pass, not-evaluated and unknown values with a fail-closed default. Preserve cumulative explicit per-ID resolutions.

Closure checks:

- Use correct hashes but failed/running/missing terminal result, write mode, wrong run or namespace → Import refuses; F1 remains open and criterion credit stays zero

- Leave an old pass report at a declared path and make the current reviewer fail without producing a report → Old report cannot approve or resolve F1

- Use valid isolated review output, then tamper bytes or change contract/evidence/findings at unchanged Git head → Valid control imports; tampered/stale variants refuse

- Exercise all five skill verdicts plus unknown and legacy host BLOCK → Only eligible authenticated pass approves; others persist explicit non-approving results without changing legacy BLOCK semantics

Trade-offs: A wrapper error after a complete report will require reconciliation or a fresh review. This conservative rule matches the previous closure criterion. Output hashes prove integrity, not semantic correctness or true context isolation.

Alternative: Allow only a trusted local adapter that launches the exact review packet and captures canonical output. Keep manual report import evidence-only until this adapter exists.

Effort: medium. Priority: Before Gate A design acceptance; implementation proof at the assigned later gate. No fixed revision exists.

### R9 — All-ledger-runs reconciliation conflicts with review accounting

**MEDIUM; observed; New v4 overbroad reconciliation rule conflicts with retained v2/v3 accounting.**

Observed/source evidence:

- `docs/handoffs/2026-10-15-gate-a-v4.md:78-81`: Every ledger run in the namespace/window must map to a reservation.

- `docs/handoffs/2026-10-15-gate-a-v2.md:139-142; docs/handoffs/2026-10-15-gate-a-v3.md:50-52`: Review launches are excluded from implementation attempts, while per-change independent reviews can occur inside an active lease.

- `probes.log`: A valid same-namespace review without an implementation reservation is flagged by the literal all-runs filter.

Diagnosis: The new scan says every ledger run starting inside the lease window must map one-to-one to an implementation reservation. The retained budget rule says reviews are not implementation attempts. V4 does not prohibit a legitimate review from using the same ledger namespace or define a separate review authorization class in this scan. Such a review either creates a false UNAUTHORIZED_DISPATCH finding or is charged as an implementation attempt.

Sequence: 1. Reserve and launch implementation I. 2. During the same active lease, launch its approved independent review R in the declared namespace. 3. Do not debit R as an implementation attempt, as v2 requires. 4. Import: R has no implementation reservation, so the literal v4 all-runs rule emits UNAUTHORIZED_DISPATCH. Debiting R to silence the error instead consumes the wrong budget.

Expected: Reconcile both implementation and review launches against their appropriate authorizations. Review launches must neither disappear from provenance checks nor debit implementation-attempt counters.

Preferred fix (proposed): Tag ledger associations by HMA run/unit/controller and operation role. Map implementation launches to attempt reservations and reviews to approved review dispatches/receipts. Alternatively, require distinct implementation/review namespaces and make that separation an explicit checked contract. Add a negative fixture for an unapproved review, too.

Closure checks:

- Run one implementation plus one legitimate independent review in the permitted namespace arrangement → One implementation attempt charged; both launches authorized; no false BLOCK

- Add an unapproved review or relabel an implementation as review → Import refuses the unauthorized launch; role labels alone cannot bypass attempt accounting

Trade-offs: A role-aware scan needs an authenticated association; trusting worker-supplied role text would recreate the gap. Separate namespaces are simpler but require explicit inventory and path binding.

Alternative: Use a checked implementation-only ledger namespace, with separate pinned review namespace and independent reconciliation.

Effort: small. Priority: Before Gate A design acceptance; implementation proof at the assigned later gate. No fixed revision exists.

**Question 3, by attack class:** No initial self-issued lease path was found: a host cache without the authentic applicable core approval and approved plan grants nothing. The v4 replay/admission changes fix double charging and the false rejection of a second intended attempt. A late unreserved launch can remain uncharged and unflagged under the window-filtered scan (R4). Ordinary same-worker/model/context self-review is refused by retained rules and the selected host tests. Wrong-mode/failed-review receipts and potentially stale report files remain unexcluded (R8). Clearing a BLOCK still requires explicit independent resolution; the R8 sequence attacks whether that purported resolution is an eligible completed review. V4 does not make arbitrary advisory imports a valid way to erase prior findings.

**Question 4:** R9 is new in v4. R4 and R8 are residual incomplete closure criteria. No new illegal transition, same-UID tamper-resistance requirement or intrinsic optional-field hash incompatibility was found. The obsolete v3 equality row for journal/reservation sets should become “launches are a subset of reservations,” because conservative unlaunched reservations remain counted.

## 4. Human decisions before Gate B

The recorded set covers the product/authority decision categories identified so far. It is not a complete execution authorization. No new discretionary D7 decision is required to correct these design gaps.

T is directly confirmed by the current user instruction and controls this audit. D1/D3/D4/D5/D6 are evaluated as the recorded decision claims in v4 section 0; the underlying human dialogue was not independently inspected.

D3 includes Phase B. D4 is conditional commit-to-main authority after each gate passes the specified independent audit and full tests; push remains separately approved. This re-audit has no Git write authority.

D5 covers the planned controller hook. D6 covers receipt digest changes. Neither makes those changes implemented or reviewed. VL commit existence and relevant bytes are independently verified.

D1 direction is sufficient to prepare concrete normative text within authorized docs work. It does not make a not-yet-written amendment an approved product contract. Resolve exact authority/compatibility text before dependent semantic implementation, not necessarily before every routine Gate B defect fix.

Before Gate B work can start, Gate A must actually pass. Each bounded Gate B unit still needs its approved plan/route/permission/budget and required challenge-bound transition authority. Section 0, a passing audit or a generic continue cannot supply those approvals. No such run-specific approval packet was supplied in this audit.

## 5. Verification and evidence

| Check | Result | Raw evidence |
|---|---|---|
| Prior-tree byte comparison | 104 prior files unchanged; only v4 added | `prior-comparison.json` |
| Go checks | 12 top-level test functions passed in 3 pure packages | `go-checks.log` |
| Controller checks | 6 selected tests passed | `controller-unit-checks.log`, `controller_unit_checks.py` |
| Current controller, mocked | HMA marker is not yet enforced; hook is not called; route normalization drops profile/provider/access-service fields | `probes.py`, `probes.log` |
| V4 design models | Replay/admission positive controls pass; late-launch and receipt-state counterexamples reproduced | `probes.py`, `probes.log` |

The absence of the future D5 hook and D6 fields in current code is expected. It is not itself a Gate A failure. D5 must preserve actual resolved route, clone/live baseline, canonical ledger namespace and current authority through start and resume. The current repair provides private-clone execution, exact preallocated IDs, reconciliation and controller-run checks; it does not supply the proposed HMA bridge.

Reproduce the bounded non-dispatch probes:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 /tmp/hma-gate-a-20261015-r4/probes.py
TMPDIR=/tmp/hma-gate-a-20261015-r4/go-tmp PYTHONDONTWRITEBYTECODE=1 python3 /tmp/hma-gate-a-20261015-r4/controller_unit_checks.py
```

The Go executable was `/home/yeli/.local/go/bin/go`, version `go1.27.1 linux/amd64`. The command selected named tests in `./internal/model ./internal/engine ./internal/host` with `-count=1 -v`. `GOCACHE`, `GOTMPDIR` and `TMPDIR` were under the output directory. The exact selection is recorded in `run-checks.txt`. An initial PATH failure is retained separately.

HMA manifest SHA-256: `573f8bcfe577f0ee25fe12538ff97edada2a9d0dac5fc435cf83594a11c19157`. V4 SHA-256: `2bf7ee7e3507ad65025c34df0f409ae34de76da980df05feca952da0e03c4fce`. The full snapshots, manifests and raw logs remain beside this report. `AUDIT.json` retains requirement coverage, prior criteria, findings and remediation checks. The report validator passed (form only). `target-verification.json` confirms unchanged HEAD, status and all captured bytes in both repositories, including supplemental inputs.

## 6. Remediation and limits

Minimal roadmap: amend the design for complete launch reconciliation and eligible review-result import; resolve the review-accounting rule; add the named negative schedules to the matrix. Re-audit those deltas before Gate B. Implement and execute the resulting checks at their assigned later gates. No unrelated redesign is proposed.

- This is a re-audit with partial isolation. Prior v1-v3 audit packets are expressly permitted inputs. It is not a blind first pass. V4 and its inherited design text are objects of audit, not authority for success claims.

- Incidental context exposure: git show --format=fuller --stat displayed the historical Verify Loop repair commit message while verifying revision provenance. No message claim about authorship, review or test success was accepted as evidence. The skill isolation_compromised flag records this exposure; a fresh restart was not attempted because the requested task is a re-audit. Design defects independently require FAIL.

- Gate A is design-level only. Existing missing HMA hook/importer/receipt fields are assigned later-gate implementation work. This report does not demand their implementation before Gate A and does not mark any proposed code fix validated.

- Both repositories were read-only by procedure, not mounted read-only. No repository edits, Git writes, nested agent CLI, model API, network request, real preflight or real worker launch ran. Popen was mocked. All audit outputs and Go caches are under this output directory.

- The HMA freeze includes 105 tracked/nonignored files with dirty/new bytes; .serena, Git internals and ignored files are excluded. Hashes identify this exact dirty tree; HEAD alone does not.

- Initial external test/skill paths were expanded incorrectly as AgentSystem/boot and AgentSystem/skills. They were absent and recorded as deleted. The actual AgentSystem/agent-memory paths were located, frozen in the supplemental manifest, and verified byte-for-byte against 9ba1e74a. The controller was correctly frozen at the start.

- Verification was bounded: 12 selected existing Go functions, 6 selected pure controller tests, mocked current-controller observations, and literal design models. No full suite, real dispatcher, migration, final closure, CI release, billing/token meter or concurrency integration test ran.

- The literal models exercise written predicates only. They are counterexamples to an incomplete specification, not runtime exploits of future HMA code. Retained v2/v3 checks were credited where applicable; no hostile same-UID host or hidden ledger is required for the reported main sequences.

- The declared route is liz / OpenAI / gpt-6.1-sol / high / review through this managed Codex session. The coordinator preflight and runtime configuration were not independently verified. No subagent was dispatched.

- JEV_UNAVAILABLE: no callable approved Jev tool was present in the exposed inventory. Auditor judgment supplies the fallback. No other model was presented as Jev.

- Some broad exploratory displays were truncated; material findings use targeted reads and preserved bytes. One Go invocation failed because go was not on PATH; the retained log records it. The successful run used /home/yeli/.local/go/bin/go (go1.27.1 linux/amd64).

- The report validator checks structure only and forbids pass under partial isolation. The gate FAIL is independently supported by unresolved high design findings, not merely by that form rule.

## 7. One next action (§18)

**REPLAN — Ming amends Gate A to close R4/R8 and R9, then submits the frozen amendment for the bounded read-only re-audit. Gate B stays stopped.**

Owner: Ming. Route: Ming / Anthropic / claude-opus-5-5, approved host profile and effort as established by the orchestrator. This audit does not dispatch that route or invent missing route facts.

Permission: Prepare only the authorized Gate A design amendment. No Gate B coding, push, release or expanded budget is authorized by this report.

Approval: No new product-decision category is identified. Use the recorded D1/D5/D6 directions to draft the correction. Retain authentic applicable unit/stage authority; obtain exact human approval of any resulting normative authority change before dependent implementation.

Stop/escalation: Keep Gate B stopped while Gate A fails. Escalate if a repair changes approved scope/risk/permission/budget, requires a new authority semantic, conflicts with the kernel or lacks an eligible independent reviewer.
