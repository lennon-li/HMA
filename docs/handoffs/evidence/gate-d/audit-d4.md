# Gate D D4 independent audit

**Verdict: FAIL.** Eight high findings remain. One medium finding remains. The report-contract encoding is `fix-required`. It expresses the same decision.

The supplied tests, vet check and race suite pass with the audit-copy settings. Independent tests reproduce unauthorized launches and incomplete reconciliation. The repository matches the starting freeze. No product files were edited.

**Current state:** D4 working tree at `main`, HEAD `a74c9f0901f89903ed0cb2425bc491ece4ea282d`. This report requests no stage transition. D4 acceptance must wait for correction and independent re-audit. D5 and D6 remain outside this audit.

**Coverage:** 3 derived requirement groups are MET. Seven are PARTIAL. Their individual failed requirements are marked NOT MET below. There is no active waiver in this audit packet. The audit did not inspect a live HMA run for waivers. No approval challenge is generated or consumed.

**Isolation:** isolated auditor context on a shared trusted host. The session began with the original objective and neutral target facts. No implementer narrative or prior verdict entered the blind pass. Execution used a disposable source copy. This was not a separate VM or an OS-enforced read-only mount.

**Auditor route requested by the dispatch:** `liz / gpt-6.1-sol / high`, review mode. This report records the requested route. It does not independently attest the host model identity. No nested agent CLI, worker model API, or worker dispatch was used. No approved callable Jev tool was exposed; the auditor used the documented fallback.

## Requirement coverage

| ID and source | Assessment | Observation | Findings |
|---|---|---|---|
| D4-01 — v5 §1–2; v6 §4.1 | **PARTIAL** | Exact-one matching works for scanned runs. Review namespaces are not bound or fully scanned. | D4-F04 |
| D4-02 — v6 §4.2–3; Gate C scope §0.1 row 3 | **MET** | No reuse, reservation charging and unlaunched listing pass supplied and independent controls. | None |
| D4-03 — v6 §2; v7 §1 | **PARTIAL** | Upper-bound expiry and revocation ties are rejected. Subsecond launches before valid_from are authorized. | D4-F05 |
| D4-04 — v7 §1–2 | **PARTIAL** | Reservation transition/outcome minima and reversed event ordering work. Review revocation, generation-bound plan selection and policy revocation fail. | D4-F02, D4-F03, D4-F08 |
| D4-05 — v7 §1–2 | **PARTIAL** | Reservation replay and old-generation launch refusal pass. Review dispatches have no owning lease generation. | D4-F03 |
| D4-06 — v5 §1 | **PARTIAL** | Mode refusal, run-ID collision and route independence controls pass. Review namespace and revocation checks fail. | D4-F03, D4-F04 |
| D4-07 — v6 §4.4–5; Gate C scope row 3 | **PARTIAL** | Envelope/namespace/unit mismatch checks pass when an admission matches. Unknown controllers skip checks. Baseline agreement is omitted. | D4-F06, D4-F07 |
| D4-08 — v5 §1–2; architecture §7, §9; user focus | **MET** | Observed unauthorized entries produce unowned BLOCK findings. Actual approving review snapshots and later clean scans preserve them. The gate remains blocked. Recommendation output has a separate medium defect. | D4-F09 |
| D4-09 — original objective Gate D; user D4 scope; architecture §7, §14 | **PARTIAL** | The CLI persists reconciliations and findings. Missing/new-reservation/stale-core reconciliations are refused. A new ledger launch and an omitted review ledger bypass freshness/coverage. The historical-plan check is also insufficient. | D4-F01, D4-F02, D4-F04, D4-F09 |
| D4-10 — user scope; Gate C scope §0.1 | **MET** | All supplied regression packages pass with the documented audit-copy settings. The post-freeze D4 handoff makes no D5/D6 completion or full Gate D pass claim. This audit grants no D5/D6 acceptance. | None |

Detailed requirement checks:

**D4-01** — Every launch in namespace from first core record through import must match exactly one reservation or review dispatch. Labels alone confer no authorization.
- MET: No matching record.
- MET: Two matching records.
- NOT MET: Scan all run namespaces from first core record.

**D4-02** — One record can authorize at most one ledger launch; unlaunched reservations remain charged and are listed.
- MET: At most one launch per record.
- MET: Unlaunched reservations remain charged and listed.

**D4-03** — Authorize only valid_from <= started_at < valid_until; equality at expiry or any revocation is unauthorized.
- NOT MET: Exact lower bound.
- MET: Exclusive upper bound and expiry ties.

**D4-04** — Compute valid_until as minimum of original expiry and all applicable revocations in the whole chain, independent of ingestion order. Rewind affecting unit/dependency, terminal outcome, policy/contract change and generation expiry apply.
- MET: Reservation min(original expiry, transition/outcome events).
- MET: Expiry earlier than later revocation.
- MET: Core event ingestion order for tested reservation minima.
- NOT MET: Review applicable revocations.
- NOT MET: Generation-correct dependency graph.
- NOT MET: Policy-change revocation provenance.
- MET: Contract-change rewind revokes implementation generation.

**D4-05** — Bind leases to approval sequence+head anchor, reservation/review to own sequence and exactly one lease generation. Renewal never revives old records; new generation requires a new reservation.
- MET: Reservation issuing approval identity and renewal does not revive old reservation.
- MET: New generation needs new reservation.
- NOT MET: Review dispatch identity bound to one lease generation.

**D4-06** — Review dispatch requires eligible stage and independent route. Review receipt mode must be review; reservation/review run ID collision is unauthorized.
- MET: Review receipt mode and implementation relabelled review refusal.
- MET: Reservation/review run-ID collision refusal.
- MET: Independent reviewer route enforcement.
- NOT MET: Review authority remains valid after issuance.
- NOT MET: Review namespace agreement.

**D4-07** — Journal envelope digest, controller ID, live root, baseline digest and namespace must equal admission; each journal dispatch ID must have a reservation.
- MET: Envelope and namespace equality with matched admission.
- MET: Live root checked against repository during import.
- NOT MET: Controller identity equality with admission.
- NOT MET: Clone baseline equality with admission.
- NOT MET: Every HMA-bound journal dispatch ID reserved.

**D4-08** — Any unauthorized launch yields unwaivable UNAUTHORIZED_DISPATCH BLOCK. Review snapshots or later reconciliation cannot conceal unresolved violation.
- MET: UNAUTHORIZED_DISPATCH maps to unwaivable BLOCK.
- MET: Approving review snapshot preserves historical BLOCK.
- MET: Later clean reconciliation preserves historical BLOCK.

**D4-09** — hma host reconcile must persist trustworthy complete evidence; forward transition requires clean current reconciliation. No host stage forgery or envelope widening.
- MET: Persist scan and BLOCK evidence without stage advance.
- MET: Missing reconciliation for reserved unit refused.
- MET: New reservation or changed core invalidates saved scan.
- NOT MET: Current complete ledger reconciliation at host gate.
- NOT MET: Generation-bound plan required for reconciliation.
- NOT MET: Next-action output agrees with unresolved BLOCK.

**D4-10** — D1–D3, Gate B and Gate C behavior stays intact; D4 does not claim D5 criterion credit/findings disposition/full packet or D6 amendment.
- MET: D1–D3, Gate B and Gate C supplied regression suite.
- MET: D5/D6 no premature handoff claims.

## Authority and frozen evidence

The requirements were derived before diff inspection. [REQUIREMENTS.json](REQUIREMENTS.json) retains the source mapping and failure hypotheses. Authoritative sources were:

- `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md:50–65`, Gate D.
- Gate A v5: lines 10–44 (§1–§2). Gate A v6: lines 19–26 (§2) and 35–61 (§4–§5). Gate A v7: lines 9–41 (§1–§2). Only these approved design sections were printed, using `sed` line ranges. Latest text takes precedence.
- `docs/handoffs/2026-10-16-gate-c-scope.md:16–35`, §0.1 rows 3–4.
- `docs/architecture.md` §7, §9, §14, §18, §19 and §22; repository `AGENTS.md`.
- [Independent-audit skill](/home/yeli/obsidian/AgentSystem/agent-memory/skills/independent-audit/SKILL.md), code and outcome modes.

Freeze details:

- HEAD: `a74c9f0901f89903ed0cb2425bc491ece4ea282d`.
- `git diff --binary` SHA-256: `0ab71ef3d4a4c83c683c54e0f06770ea3d03aac4096a11fdc83b64b71cb14525`.
- Immutable [PHASE1.json](PHASE1.json) SHA-256: `d2c61bd65e3045354dcdaefb75dd86ffa9f31cd1b852235bd5c85fb9f71499db`.
- [FREEZE-START.json](FREEZE-START.json) and [FREEZE-END.json](FREEZE-END.json) retain `git status --short` and hashes for every untracked file. All four comparison fields match.
- [tracked.diff](tracked.diff) preserves the tracked binary diff. [SOURCE-MANIFEST.json](SOURCE-MANIFEST.json) identifies 259 copied primary files. [COPY-VERIFICATION.json](COPY-VERIFICATION.json) records no copied primary-file changes.
- `.serena/` and architecture-amendment handoffs were excluded from substantive review. Their untracked hashes were still frozen. The D4 handoff was withheld until PHASE1 was hashed. No other `/tmp/hma-gate-*` or `/tmp/claude-*` directory was inspected.

## Verification and reproducibility

Go: `go1.27.1 linux/amd64`. Git: `2.43.0`. The executable was `/home/yeli/.local/go/bin/go`. Its SHA-256 is `30969f97169d7f43fe6a085873d75613adc21e30818a8c61d95bd27275df4624`. Git inspections used `/usr/bin/git`.

| Check | Result | Complete log and command metadata |
|---|---|---|
| test | Initial run: environment failures | [test.log](test.log), [test.json](test.json) |
| race | Initial race run: same environment failures | [race.log](race.log), [race.json](race.json) |
| vet | PASS | [vet.log](vet.log), [vet.json](vet.json) |
| test-adjusted | PASS: all 12 supplied packages | [test-adjusted.log](test-adjusted.log), [test-adjusted.json](test-adjusted.json) |
| race-adjusted | PASS: all 12 supplied packages; no race diagnostics | [race-adjusted.log](race-adjusted.log), [race-adjusted.json](race-adjusted.json) |
| independent-final-race | FAIL: 12 of 17 top-level auditor tests; five controls pass; no race diagnostics | [independent-final-race.log](independent-final-race.log), [independent-final-race.json](independent-final-race.json) |

The first run put Go test temporary directories under `gotmp` while `os.TempDir()` was `tmp`. Synthetic-store safeguards rejected those paths. The copy also lacked `.git`, so nested builds failed VCS stamping. The corrected runner sets `TMPDIR == GOTMPDIR` under this output directory and `GOFLAGS=-buildvcs=false`. It keeps `GOCACHE` here, sets `GOTOOLCHAIN=local`, disables dependency downloads with `GOPROXY=off`, and puts the supplied Go binary first in `PATH`. Initial failures remain in the evidence packet. They are audit-environment failures, not D4 defects.

Every check retains complete combined stdout/stderr and metadata with its executable vector, working directory, environment, start/end UTC timestamps, exit code and FAIL-line scan. Output logs are not truncated. Tool previews of some logs were truncated; the referenced files contain the full bytes. Exit codes alone were not treated as proof of success.

Supplied tests ran with `-skip ^TestAuditD4` after auditor tests were added. This excludes only auditor-authored negative controls. The required Gate C scope row 3–4 test names all exist and were executed. The supplied association test omits controller/root/baseline cases; its green result does not establish the full association requirement.

Reproduce the auditor tests from this retained copy:

```sh
python3 /tmp/hma-gate-d-20261018-d4/run-check-adjusted.py independent-repro /home/yeli/.local/go/bin/go test -race -count=1 -v ./internal/host ./cmd/hma -run '^TestAuditD4'
```

The command is expected to fail against the frozen D4 code. Its failed assertions identify missing rejection behavior. The five passing controls check valid/invalid matching and ordering, record reuse, BLOCK retention after later scans, unreadable receipts, and a real approving review snapshot that preserves BLOCK. The tests use synthetic approvals in disposable fixture stores. They do not create real authority.

The auditor tests are [host tests](source/internal/host/audit_d4_test.go) and [CLI tests](source/cmd/hma/audit_d4_test.go). Other retained runs are `independent`, `independent-expanded`, `review-ledger` and `historical-plan`. Each has `.log` and `.json` metadata.

## Findings and proposed corrections

All nine findings are observed defects. All fixes below are proposals. None was implemented or validated by this audit. Each source locator refers to the frozen target or its identical retained copy.

### D4-F01 — The host gate accepts a reconciliation made before a new launch

**Severity:** high. **Confidence:** observed. **Effort:** medium. **Fix status:** proposed.

**Evidence:** `cmd/hma/transition.go:355–382`; `TestAuditD4GateStaleLedger` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** A clean scan opens the full host implementation-to-review gate. Adding an unreserved receipt without changing the core or host chains leaves that gate open. A fresh scan finds the rogue and then closes the gate. Root cause: The gate compares only the last reservation sequence, core head and saved unauthorized list. It does not bind or recheck the current ledger. It also returns immediately for units without reservations and does not use review dispatches or journal records as freshness inputs.

**Preferred correction:** Make the gated transition reconcile a stable ledger snapshot under a launch/transition serialization mechanism. Bind the namespace inventory, relevant host history, core head, plan-generation inputs and receipt content digests. Refuse progression when any bound input changes. Apply the requirement to review launches and imported journals, including units with no implementation reservation.

**Why:** A current core head does not prove the launch ledger stayed unchanged.

**Independent exit checks:**

- Run TestAuditD4GateStaleLedger. Expected: The new unreserved launch closes the host gate before any transition append.
- Add a review dispatch or loop evidence after a clean scan, then request progression. Expected: The prior scan is stale. A current clean scan remains a positive control.

**Trade-offs:** Adds ledger I/O and lock-order requirements. A scan alone cannot close a launch race unless dispatch and transitions share serialization or an immutable ledger generation.

**Alternative:** Seal the run namespace before final reconciliation and progression. This is simpler but stops additional launches until an explicit new execution window.

### D4-F02 — Any historical approved plan can hide a new generation dependency rewind

**Severity:** high. **Confidence:** observed. **Effort:** medium. **Fix status:** proposed.

**Evidence:** `cmd/hma/reconcile.go:62–75; internal/engine/revocation.go:22,53–55`; `TestAuditD4HistoricalPlanHidesDependencyRevocation` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** A new approved plan adds U1 -> U2. A new U1 reservation is issued. U2 then rewinds. Reconciliation with the old plan reports d-new-plan authorized and no violations. Reconciliation with the new plan reports the violation. Root cause: The CLI accepts a plan digest found in any historical approval, then uses that one dependency graph for every reservation generation.

**Preferred correction:** Resolve each reservation and review dispatch to its issuing generation and that issuer approval's accepted plan digest. Load the exact immutable plan artifact for that generation. Refuse missing or mismatched artifacts. Do not apply an arbitrary historical plan to other generations.

**Why:** Revocation dependencies belong to the plan that granted the authority. One current or historic graph is insufficient for a mixed-generation history.

**Independent exit checks:**

- Run TestAuditD4HistoricalPlanHidesDependencyRevocation. Expected: The old plan is refused for the new generation, or the generation-bound plan still detects the U2 rewind.
- Reconcile a run containing reservations from two plans, before and after each dependency rewind. Expected: Both generations use their own dependency graph. Legitimate earlier launches remain authorized.

**Trade-offs:** Requires retaining plan artifacts by digest and a migration policy for historic records. Do not silently choose the latest plan for old records.

**Alternative:** Require the caller to supply a complete digest-keyed plan bundle for all generations and fail closed when it is incomplete.

### D4-F03 — Review dispatch validity ignores rewinds and lease generations

**Severity:** high. **Confidence:** observed. **Effort:** medium. **Fix status:** proposed.

**Evidence:** `internal/host/reconcile.go:195–213; internal/host/host.go:192–208`; `TestAuditD4ReviewAfterRewind; TestAuditD4GeneratedReviewRewind` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** Reviews launched at a unit rewind, dependency rewind or whole-run rewind are authorized. The CLI also authorizes a real HMA-generated review packet after its unit rewinds. Root cause: Only reservations call RevokedAt. Review validity takes the packet TTL and one terminal timestamp. ReviewPacketBinding has no lease generation, and review validity is not capped at its generation's wall-time end.

**Preferred correction:** Persist the review dispatch's own identity and exact owning lease generation at issuance. Derive its end as the minimum of its TTL, its generation expiry and every applicable revocation. Preserve prior generation ends after renewal. Use a review-specific revocation rule so a legitimate forward move into review does not revoke the review itself.

**Why:** Gate A v7 applies the monotone minimum and generation identity to both reservation and review records.

**Independent exit checks:**

- Run both listed tests. Expected: Every review at or after the applicable rewind is unauthorized; a launch just before the rewind is authorized.
- Renew a lease and launch the old review ID, then issue a fresh review in the new generation. Expected: Old dispatch remains revoked. The new dispatch is independently eligible when stage and route rules pass.

**Trade-offs:** Adds fields to review records and needs fail-closed handling for old packet records. Blindly reusing the reservation stage-exit rule would reject legitimate reviews.

**Alternative:** Invalidate and reissue all pending review packets on any generation change, with an explicit immutable cancellation record used during import.

### D4-F04 — Review ledgers are neither bound nor included in scan coverage

**Severity:** high. **Confidence:** observed. **Effort:** medium. **Fix status:** proposed.

**Evidence:** `internal/host/reconcile.go:69–79,124–136; cmd/hma/review.go:32–43,329; internal/host/host.go:192–208`; `TestAuditD4ReviewNamespace; TestAuditD4ReviewLedgerCoverage` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** The classifier authorizes a review in an unrelated namespace. An actual imported review in a separate ledger and a rogue write launch in that ledger are both omitted by hma host reconcile. The full host review-to-verification gate then opens. Root cause: Namespaces come only from loop admissions. Review dispatches have no ledger namespace binding, and review import does not persist its ledger directory for reconciliation.

**Preferred correction:** Bind the canonical ledger namespace when issuing each review dispatch. Require receipt import from that namespace. Include every run-declared review and implementation namespace in reconciliation for the whole run lifetime. Check a review receipt namespace before consuming its record.

**Why:** A scan of implementation admission namespaces cannot certify a separate review namespace.

**Independent exit checks:**

- Run both listed tests. Expected: A wrong-namespace review is unauthorized or refused. The separate review ledger is scanned, the rogue becomes BLOCK, and the review gate closes.
- Put implementation and review receipts in one namespace in reversed scan order. Expected: Both valid launches remain authorized and implementation attempt accounting is unchanged.

**Trade-offs:** Changes review-packet input and persisted binding. Canonicalize paths and preserve legacy records explicitly rather than inventing a namespace.

**Alternative:** Require a single immutable run namespace at run creation and refuse every dispatch/import outside it.

### D4-F05 — Rounding valid_from down authorizes pre-issuance launches

**Severity:** high. **Confidence:** observed. **Effort:** small. **Fix status:** proposed.

**Evidence:** `internal/host/reconcile.go:138–144`; `TestAuditD4ExactLowerBound` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** For valid_from 2026-10-18T12:00:02.900Z, started_at values 12:00:02Z and 12:00:02.800Z are both authorized. The exact valid_from control is also authorized. Root cause: The comparison floors the lower bound to whole seconds even when the receipt provides subsecond precision.

**Preferred correction:** Compare started_at with the exact valid_from. If a whole-second receipt cannot establish ordering, reject the ambiguous launch or improve ledger timestamp precision. Do not widen the approved interval.

**Why:** The approved interval is [valid_from, valid_until). Host clock disclosure does not approve a lower-bound exception.

**Independent exit checks:**

- Run TestAuditD4ExactLowerBound. Expected: Both earlier timestamps are unauthorized; the exact bound is authorized.
- Launch at valid_until and one timestamp just before it. Expected: The upper tie is unauthorized. The strictly earlier in-window control is authorized.

**Trade-offs:** Existing whole-second receipts may fail closed for legitimate same-second launches. Raising ledger precision avoids that compatibility cost.

**Alternative:** Keep whole-second receipts but delay launch to the next second after reservation issuance and retain the exact comparison.

### D4-F06 — An unknown journal controller bypasses all journal invariants

**Severity:** high. **Confidence:** observed. **Effort:** small. **Fix status:** proposed.

**Evidence:** `internal/host/reconcile.go:167–188; cmd/hma/loop.go:200–206`; `TestAuditD4JournalUnknownController; TestAuditD4JournalControllerMutation` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** Changing an admitted journal's controller run_id to unknown-controller and adding an unreserved attempt is accepted by loop-evidence import. Reconciliation reports no unauthorized entries. Root cause: The admission lookup uses the supplied controller ID. If no admission matches, reconciliation continues before checking the journal attempts.

**Preferred correction:** For an HMA-bound journal, require exactly one matching admission. Validate controller, unit, envelope, namespace and generation against it. Check every nonempty journal dispatch ID for a reservation even when admission lookup fails; record the mismatch as UNAUTHORIZED_DISPATCH.

**Why:** Absence of a matching admission is evidence of missing authority, not evidence that no hook ran.

**Independent exit checks:**

- Run both listed tests. Expected: The unknown controller and unreserved attempt are refused or become BLOCK.
- Import the original admitted controller with its real reservation. Expected: The matching journal is accepted and introduces no BLOCK.

**Trade-offs:** Legacy HMA-bound journals without admissions will no longer produce a clean D4 result. They need an explicit migration or a disclosed non-authorizing compatibility path.

**Alternative:** Reject non-admitted HMA-bound journals before executing the recapture command. Keep independent legacy evidence ingestion outside launch authorization.

### D4-F07 — Journal clone baseline is discarded before reconciliation

**Severity:** high. **Confidence:** observed. **Effort:** small. **Fix status:** proposed.

**Evidence:** `cmd/hma/loop.go:35–39,200–206; internal/host/host.go:326–340; internal/host/reconcile.go:176–178`; `TestAuditD4JournalBaselineMutation` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** After reservation admission with the empty baseline digest, changing state.repo.workspace_baseline_inventory is accepted by loop-evidence import. Reconciliation still reports d-1 authorized with no violations. Root cause: The importer parses only repo.root, never records the journal baseline digest in LoopEvidence, and reconciliation compares only envelope, namespace and unit.

**Preferred correction:** At journal import, compute workspace_baseline_inventory's canonical digest using the same absent-inventory rule as admission. Bind that digest and the canonical live root to the captured journal evidence. Compare the full association tuple to the exact admission before returning a clean result.

**Why:** Gate A v6 §4.4 expressly requires clone baseline and live-root agreement, not just envelope agreement.

**Independent exit checks:**

- Run TestAuditD4JournalBaselineMutation. Expected: A changed baseline is refused or yields an unauthorized finding.
- Test unchanged baseline, absent inventory {}, and a different live root. Expected: Matching empty baseline passes. Any changed baseline or live root fails.

**Trade-offs:** Adds evidence fields. Historic journal records lack enough retained association data for complete proof and must remain non-authorizing until reconstructed.

**Alternative:** Retain the raw journal as an immutable content-addressed artifact and derive the association during reconciliation. This costs more I/O but preserves re-verification inputs.

### D4-F08 — Policy revocation is enforced by the hook but omitted by reconciliation

**Severity:** high. **Confidence:** observed. **Effort:** medium. **Fix status:** proposed.

**Evidence:** `cmd/hma/reconcile.go:19–23,47–75; internal/engine/revocation.go:34–57`; `TestAuditD4PolicyRevocation` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** Changing the current policy makes replay return launch_eligible false. A launch that ignores that refusal is nevertheless classified authorized by hma host reconcile. Root cause: Reconciliation has no current-policy input or immutable policy-change event history. RevokedAt handles transitions and outcomes, so it cannot include the policy-change time in the validity minimum.

**Preferred correction:** Record applicable policy changes with immutable timestamps and digests under trusted host control, bound to affected lease generations. Include those events in the whole-chain minimum. Capture current policy identity during reconciliation and refuse a clean result when required revocation provenance is missing.

**Why:** A launch forbidden by the active lease cannot become authorized merely because the importer lacks policy history.

**Independent exit checks:**

- Run TestAuditD4PolicyRevocation. Expected: The post-change launch is unauthorized, matching the hook's refusal.
- Launch just before, at and after a recorded policy change, then restore the old policy. Expected: Before-change launch passes; tie and later launches fail. Restoring the policy does not revive the old generation.

**Trade-offs:** Needs an immutable policy-event representation and host integration. Reading only the latest policy file cannot distinguish valid historic launches from post-change launches.

**Alternative:** Containment: refuse reconciliation of any generation whose policy identity no longer matches until the host can provide trustworthy change-time provenance. This may overblock legitimate earlier launches.

### D4-F09 — A later clean scan recommends continuation while a historical BLOCK persists

**Severity:** medium. **Confidence:** observed. **Effort:** small. **Fix status:** proposed.

**Evidence:** `cmd/hma/reconcile.go:109–113`; `TestAuditD4LaterCleanScanRecommendation` in [independent-final-race.log](independent-final-race.log).

**Diagnosis:** After a rogue receipt records an unauthorized-dispatch BLOCK, removing only the scratch receipt and reconciling again returns CONTINUE_HMA_TRANSITION. The historical BLOCK remains and the gate correctly rejects progression. Root cause: The result next_action depends only on this scan's unauthorized list and does not inspect projected unresolved findings.

**Preferred correction:** Choose the result next_action from both the scan and the projected core findings after the transaction. Return ESCALATE_TO_HUMAN while an unresolved unauthorized-dispatch BLOCK remains, even when this scan is clean.

**Why:** The human decision surface must not direct an action the deterministic gate refuses.

**Independent exit checks:**

- Run TestAuditD4LaterCleanScanRecommendation. Expected: The later clean scan preserves the BLOCK and recommends escalation.
- Run a clean reconciliation with no historic violation. Expected: The clean control can still recommend continuation.

**Trade-offs:** No authorization expansion is required. Keep historical evidence visible; do not clear the BLOCK to make the recommendation consistent.

**Alternative:** Expose an explicit blocked field and derive one required next action centrally from the same gate decision.

## Evidence interpretation and remediation sequence

The matching controls reject a missing record, a double match, wrong implementation namespace, wrong review/implementation mode, record reuse and an expiry tie. Reservation revocation minima stay unchanged under the tested event-order permutation. The original expiry remains earlier than a later revocation. Old reservation generations are not revived by renewal. Supplied Gate B, Gate C and D1–D3 tests remain green.

The passing controls do not cover the missing review and policy rules. They also do not turn a saved scan into current evidence. These are observed authorization-classification defects. No test demonstrated a forged human approval or an actual unauthorized core stage append.

Containment: keep D4 unaccepted. Correct the generation/plan/namespace bindings and policy provenance first. Then connect complete current reconciliation to the serialized transition path. Correct the lower-bound, journal and recommendation defects in the same bounded remediation cycle. Preserve all BLOCK findings. Re-run the supplied checks and these independent counterexamples on the exact corrected snapshot. Independent re-audit is the exit condition. No criteria rebaseline is justified.

No separate optional redesign is proposed. The corrections above are the smallest changes identified to meet the original requirements. The listed alternatives describe trade-offs; they do not authorize implementation.

## Controlled unblinding addendum

Phase 1 was frozen before the D4 handoff was read. Its digest remains `d2c61bd65e3045354dcdaefb75dd86ffa9f31cd1b852235bd5c85fb9f71499db`. The Phase 1 file was not overwritten.

The handoff explicitly discloses lower-bound flooring, omitted policy-change reconstruction, loop-only namespace coverage and missing journal baseline. These disclosures match the reproduced behavior. They do not amend Gate A v5/v6/v7 or waive the original requirements. No D5/D6 completion, criterion-credit or full Gate D pass claim was found.

No finding or verdict changed. D4-10 scope-claim coverage is now MET instead of its pre-unblinding unknown subcheck.

## Audit limitations

- Fresh audit conversation with no implementer narrative or previous verdict supplied. Primary comments and supplied tests were inspected as artifacts only. Blinding is context isolation on a shared trusted host, not a separate VM, container or OS-enforced repository read-only mount.
- The target is a dirty main worktree, not a committed D4 revision. Freeze manifests and the copied source hashes identify the exact bytes. The copy excludes .git, .serena/, architecture-amendment handoffs and the initially blinded D4 handoff.
- All tests ran in the disposable source copy. CLI fixture helpers create synthetic chained approvals in scratch stores. No real human approval or production stage transition was exercised. Host gate functions were exercised directly; no finding asserts that human challenge binding was forged.
- The initial full test and race runs failed because GOTMPDIR differed from TMPDIR and the copy lacked VCS metadata for nested builds. Corrected runs use one temporary root and GOFLAGS=-buildvcs=false. Original failure logs are retained; they are not attributed to D4 defects.
- The host clock and host-managed ledger are trusted, as required by the scope. The audit does not assess a hostile operating system, compromised host store, or arbitrary forged host records.
- No approved callable Jev adapter was exposed in the host tool set. The auditor used the explicit fallback path. No worker dispatch, nested agent CLI or model API was invoked.
- D5 criterion credit, imported findings disposition/full packet and D6 amendment are excluded. Their full acceptance is not inferred from passing regression tests.
- Proposed fixes have not been implemented or independently validated. The race detector found no DATA RACE diagnostics in the supplied suite or the independent negative controls. This does not prove absence of other races.
- A clean supplied suite is regression evidence only. The named journal association test does not cover all five required association fields, and its passing result does not override the independent failures.

The final report JSON follows the independent-audit report contract. `fix-required` is its machine encoding for this audit's single FAIL verdict. Structural validation checks form and consistency only. It cannot prove the evidence or validate proposed fixes.

## Single next action — architecture §18

**Return D4 to the implementer for bounded correction and independent re-audit.** The eight high defects prevent acceptance. The medium defect misstates a blocked next action. Proposed owner: the HMA orchestrator and implementation author, followed by the independent auditor.

Proposed implementation route: Ming / Anthropic / `claude-opus-5-5` / Claude Code CLI, as declared in the post-freeze handoff. That handoff does not state effort. The coordinator must choose an approved effort, verify eligibility and perform exact-route `hi` preflight before dispatch. This audit grants no route approval and dispatches no worker.

Permission envelope: bounded D4 code and meaningful regression tests under the coordinator's existing authority. This auditor remains read-only on `/home/yeli/repos`. No publication, git write, stage advance, criterion credit or D6 amendment is authorized by this report. Exact approval: no new approval is requested here. Any later stage transition still needs its own current challenge-bound human approval. Stop if a correction exceeds the approved design or needs an unresolved contract decision. Return that decision to the human gate. Resume acceptance review only after the exact corrected snapshot and raw checks are independently audited.
