# Independent Gate C audit

**Verdict: PASS. Gate C consolidation only.**

All 13 in-scope requirements are MET. Of 44 prior findings, 37 are FIXED. The remaining 7 are DEFERRED-BY-HUMAN. No in-scope defect was demonstrated. Gate D requirements remain open.

## Human review packet

- **Current state:** `main`, HEAD `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, plus the frozen working tree. The repository remained unchanged.
- **Recommendation:** Lennon should review this packet for explicit Gate C acceptance.
- **Coverage:** 13 MET; 0 PARTIAL; 0 NOT MET. Details and qualifications are below.
- **Required evidence:** Full tests, vet, race tests, 17 synthetic host-dogfood checks, independent negative controls, baseline migration, real verify-loop contract fixtures, and real-binary response checks passed.
- **Scope and budget drift:** No audited repository file was edited. No coding budget was supplied. This audit makes no budget-compliance claim.
- **Findings:** No new actionable in-scope finding. The accepted Gate D deferrals remain visible.
- **Risk and waivers:** This is an authority, evidence-import, and migration audit. No live HMA run or active product-waiver inventory was supplied. A synthetic legacy waiver survived migration.
- **Owner and route:** Lennon; human review. No worker execution is requested.
- **Approval challenge:** None was supplied or generated. This audit cannot authorize an HMA stage transition. A later transition needs its own authentic challenge-bound approval.

## Scope, provenance, and isolation

Dispatch: `hma-gate-c-audit-20261017-final12`. Declared auditor route: Liz / OpenAI / `gpt-6.1-sol` / high / Codex CLI. Repository operations were read-only. No nested agent CLI or model API was called. No subagent was dispatched. Coordinator preflight and model telemetry were not independently re-verified.

Isolation: **isolated context** for the current Gate C audit. The session did not inherit implementer conversation. The sandbox could permit source writes; read-only operation was a task constraint, not a read-only mount. Tests and fixture mutations ran outside the repository. The limitations below distinguish permitted source comments from withheld review narratives.

Authoritative sources: consolidation handoff §5 Gate C and §6; approved design v3 §§3–4,6, v4 §§3–5, v5 §§1–3, v6 §§1,5, and v7; the accepted scope §0.1; `docs/architecture.md`; `AGENTS.md`; `docs/task14-host-orchestration.md`; external ledger, audit-report, and verify-loop contracts; sanitized prior closure criteria. Only the specified design sections were read, apart from section headings used to locate them.

Ignored audit inputs: `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`. Their untracked hashes are still in the mandatory start/end inventory. No prior-review directory was opened.

Before Phase 1, the scope note was displayed only with `sed -n '1,86p'`. Original requirements were recorded before diff inspection in `requirements-before-diff.json`. Initial bulk tool output truncated some closure criteria. A completeness check recovered all 44 permitted criteria before the freeze. No withheld current Gate C narrative was used to recover them.

Phase 1 SHA-256: `e3edcfed6331b764ec81fef745803081b60596642b1058ab83805e3343f1e8b6`. The original `PHASE1.json` was created exclusively and was not overwritten. The external report validator accepted it. After freezing it, the auditor read scope §4a onward and TODO’s Gate C section. These added no requirement, finding, or verdict change. See `AUDIT.json.controlled_unblinding`.

## Target freeze

Binary working-tree diff SHA-256: `2abaadf6cbdf68e314bfdbb5f95db7e2f1eb610d25f91d84fca2ed6570fefab0`.

`freeze-start.json` and `freeze-end.json` contain the exact status and every untracked file hash. `freeze-verification.json` compares HEAD, branch, status, diff hash, and all untracked hashes. Every field matched. `audited-working-tree.patch` retains the tracked diff. `source/` retains executable source and tests. `input-digests.json` identifies the requirement and interface inputs.

## Requirement coverage

| ID | Status | Requirement and independent observation | Evidence |
|---|---|---|---|
| C01 | MET | Generated packet path, live root/base/head/diff/worktree and dispatch checks are enforced. Forged legacy packet input is refused. Wrong approved base/root/clone controls pass in the supplied suite. Source: scope §1; v4 §5; v6 §1; architecture §§7,14,16 | E05, E08, E00 |
| C02 | MET | Packet contains bound core head, accepted-plan/route digests, projected criteria, current/unowned findings and active waivers. Core drift at commit refuses. Full request/definitions/non-goals/budgets/policy/evidence contents remain explicitly Gate D. Source: v4 §5; architecture §16; accepted scope §0.1 | E08, E09, E12 |
| C03 | MET | Normal route checks worker, exact model and context. High/critical/disputed/final checks both families. Every recorded implementation contribution is considered, including earlier routes. Supplied multi-route and final-reason tests pass. Source: v3 §6; v4 §5; architecture §§12.4,13 | E01, E03, E08 |
| C04 | MET | Wrong/missing route/run/root/digest fields refuse unchanged chains. Launch at issuance is valid; before issuance or at expiry is invalid; one nanosecond before expiry is valid. Concurrent duplicate imports append once. Old approvals cannot supersede newer failures or pending packets. Source: scope §2; v4 §5; v5 §3; v6 §1; v7 | E05, E07, E08 |
| C05 | MET | Malformed bytes refuse; contract-invalid reports cannot APPROVE and safe ESCALATE records are retained where shape permits. Python-invalid fields, separators and UTF-8 controls preserve blockers and close the host gate. Total verdict map and final isolation controls pass. Source: scope §2; independent-audit report contract | E05, E11 |
| C06 | MET | REQUEST_CHANGES and ESCALATE keep omitted and same-ID BLOCK/WAIVABLE findings. Authentic current APPROVE resolves only exact-unit findings; U1/sub findings remain. Both import stages reject stale core and live targets. Snapshot is required before the host gate passes. Source: scope §3; v3 §§4.1–4.2; architecture §§7,9 | E05, E07, E09 |
| C07 | MET | All imported audit findings are ADVISORY/PARK regardless of severity or unproved citations. Imports write no criterion dispositions. Kernel/budget reproduction and positive credit remain Gate D. Source: scope §3 and accepted §0.1; v3 §4.1 | E05, E06, E09, E12 |
| C08 | MET | A journal/report generated by verify-loop contract functions was ingested after failed check, synthetic repair and fresh check. HMA ran its own check, retained output and exact dirty target, then refused modified envelope data. No worker launch, reservation, criterion credit or stage authority was inferred. Source: Gate C objective; v3 §3.2; architecture §§7,14 | E05, E06, E10 |
| C09 | MET | Missing and ABORTED runs refuse before marker execution. In-check nonblocking flock probe sees core run lock HELD. Open-run captures complete without deadlock. Host/core lock order matches the existing core append discipline. Source: Gate B regression; task14 host store contract | E05, E07, E10 |
| C10 | MET | Supplied failed-review/remediation/fresh-review fixture passes; independent real-controller journal fixture passes; stale report/packet/self-review/no-report/no-credit controls pass. This is fixture evidence, not Gate F live-model dogfood. Source: scope §4; consolidation §5 Gate C; §6 | E01, E05, E06 |
| C11 | MET | Baseline 527b7c5 writer produced core and host records that current loaders read with identical hashes. Mode is legacy_manual before migration and packet_import after generation. Current approving import resolves attributed legacy findings, preserves active waiver and rejects manual override; ambiguous findings stay owned by nobody and remain in force. Source: Gate C migration objective; prior closure criteria F13,A05,A09 | E06, E13, E14 |
| C12 | MET | Full suite, race suite, vet and 17-check real-CLI synthetic dogfood pass. Imports have no stage-write path. Core stage changes still require transition.Apply and bound approval; successful host gates do not manufacture human approval. Source: consolidation §5 Gate B/C; architecture §§5–7,15 | E01, E03, E04, E07, E14 |
| C13 | MET | Accepted scope records Gate D prerequisites and exit tests; no evaluated Gate C implementation or documentation claim relies on those deferred mechanisms. The full proof matrix spans Gates B–F; Gate C approval is limited to consolidation. Source: human decision 2026-10-17; scope §0.1 | E12 |

## Verification evidence

The full original suite ran before auditor tests were added. Later tests added only auditor-owned files to the external source copy. No product code was changed there. Every Go command used the requested toolchain and a GOCACHE under this output directory.

| Evidence | Command or inspection | Observed result |
|---|---|---|
| E00 | freeze-start.json; input-digests.json; requirements-before-diff.json | main at 527b7c5ff79090c558f5bbfed5ec87a85f06d1e5; unstaged binary diff SHA-256 2abaadf6cbdf68e314bfdbb5f95db7e2f1eb610d25f91d84fca2ed6570fefab0; every untracked file hashed; requirements derived before diff inspection. |
| E01 | go-test.json; go-test.log | go test -count=1 ./...: exit 0; 12 packages passed; source copy includes original dirty/untracked code, before auditor tests were added. |
| E02 | go-vet.json; go-vet.log | go vet ./...: exit 0. |
| E03 | go-race.json; go-race.log | go test -race -count=1 ./...: exit 0; all 12 packages passed. |
| E04 | host-dogfood.json; host-dogfood.log | scripts/dogfood/host-dogfood.sh --keep: exit 0; 17 synthetic checks passed; real CLI binaries; host gate success still refused stage append without human approval. |
| E05 | independent-tests.log; source/cmd/hma/auditor_final12_test.go | 9 independently authored top-level tests and 68 subtests passed (77 PASS entries). Covers nonapproving snapshots, unchanged-chain refusals, Python-invalid reports, exact JSON variants, missing/drifted receipt fields, nanosecond launch boundaries, concurrent imports, stale core/live targets, output retention/truncation, held core lock, no-shell entry points, missing and terminal runs. |
| E06 | independent-extra.log; source/cmd/hma/auditor_extra12_test.go; make_real_loop.py | 2 further independently authored tests passed. Baseline HEAD code wrote legacy core and host records; current loader preserved exact hashes; migration resolved unit-owned legacy findings, retained active waiver and rejected manual approval. Real verify-loop functions validate_envelope/build_state/run_check/save_state/load_state/build_report produced a failed-check -> synthetic repair -> passing-check journal/report; HMA independently re-captured it; altered envelope refused. |
| E07 | independent-race.log | go test -race -count=1 -timeout=5m -run TestAuditor12 -v ./cmd/hma: exit 0; all 11 auditor tests and 68 subtests passed; no race diagnostic. |
| E08 | cmd/hma/review.go:152; cmd/hma/review.go:296; internal/host/review_import.go:116; cmd/hma/repository.go:28; internal/host/host.go:740 | Packet generation observes approved root/base and artifact; random nonce and packet digest originate at HMA. Manual packet injection refused at host dispatch. Import checks exact route/run/root/launch/report bindings, half-open launch interval, one import and latest dispatch. Core head/live target and approved repository checked at both imports. |
| E09 | internal/host/findings_import.go:27; internal/host/findings_import.go:71; cmd/hma/review.go:395; cmd/hma/transition.go:326; internal/model/validate.go:283 | Unsupported references map to ADVISORY/PARK. Failed/escalating snapshots preserve all prior BLOCK/WAIVABLE findings. Ownership uses exact UnitID. Approving current snapshots can resolve own findings only. Imported review needs its snapshot before the host Verification gate passes; other later core records make it stale. |
| E10 | cmd/hma/loop.go:87; cmd/hma/loop.go:231; cmd/hma/loop.go:306; internal/store/jsonl.go:251; internal/evidence/noshell.go:50 | Loop admission and capture hold host then core lock across terminal/route checks, execution and append. Captured output is retained atomically as a content-addressed artifact; preexisting artifact bytes/type are verified. Captures use explicit executable argv; shell names, symlinks, shebang interpreters and wrappers are refused. Commands inside general programs remain a trusted-host/approved-argv limitation. |
| E11 | internal/host/exact_json.go:21; internal/host/report_contract.go:20; internal/host/gatec_r3_parity_test.go | Exact lowercase-key decoder avoids Go case-fold aliases; invalid UTF-8 refused; report form/verdict validator compared to Python. All Unicode code points checked for Python whitespace-predicate parity in the supplied full suite. |
| E12 | docs/handoffs/2026-10-16-gate-c-scope.md:9-41 (read before Phase 1 only by sed -n 1,86p); accepted human instruction 2026-10-17; docs/task14-host-orchestration.md:130-223 | Accepted Gate D deferrals retain reservation/admission/replay/association/revocation/authority-hook, positive-credit and reproduced-kernel/budget exit criteria. Full acceptance-contract packet inputs are also deferred by human decision. Current host documentation describes a partial integration and makes no lease, criterion-credit or closure claim. |
| E13 | internal/host/host.go:302; internal/host/findings_import.go:104; baseline-seed; baseline/cmd/auditseed/main.go | Legacy mode is explicit; first HMA-generated packet switches the whole run to packet_import. Additive omitted fields preserve old serialization. Baseline writer built from /usr/bin/git archive HEAD; current baseline migration test checked exact core and host record hashes. Ambiguous owners stay unresolved; active waiver is not dropped. |
| E14 | independent-binary.log; source/cmd/hma/auditor_binary12_test.go; hma | Real HMA binary built from frozen code. Independent bind-repository, loop-evidence, review-import and findings-import commands exercised. Enforcement labels equal host show --mode; snapshot next actions follow verdict and host gate. |
| E15 | freeze-end.json; freeze-verification.json | HEAD, branch, git status --short, binary diff SHA-256 and SHA-256 of every untracked file equal the starting freeze. |
| E16 | retained-loop.log; retained-real-loop/journal/state.json; retained-real-loop/journal/report.json; retained-real-loop/hma-store; retained-real-loop/original-path-map.json; source/cmd/hma/auditor_retain12_test.go | Supplementary post-freeze capture preserves the actual synthetic verify-loop journal/report, raw red/green stdout/stderr, fixture checkout, HMA core/host records and retained output artifact. Contract producer confirmed red exit 1 then fresh exit 0; HMA recapture exit 0; no criterion credit. Original temporary paths are mapped to retained copies. No model or worker was launched. |

The original suite and race suite passed all 12 packages. The first independent suite passed 9 top-level tests and 68 subtests. Two extra tests checked the real controller contract and baseline migration. Those 11 top-level tests and 68 subtests also passed under `-race`. A separate real-binary test passed four response-contract cases. A final supplementary test retained the raw real-controller fixture. The independent tests reuse setup helpers, but their failure hypotheses, inputs, and assertions were authored for this audit.

`retained-real-loop/` holds a synthetic failing check, a synthetic repair, a fresh passing check, the controller journal/report, the HMA recapture, and retained stdout/stderr. It does not show a live worker dispatch or execution lease. Original temporary paths are preserved in the records and mapped to retained copies in `original-path-map.json`.

The supplementary Python check in `retained-output-verification.log` independently recomputes the HMA output digest and controller envelope digest from retained bytes (E17). It confirms failed controller check 1, fresh check 0, and HMA recapture 0.

## Prior-finding closure

The status covers the stated defect, with the qualification in each row. Deferred positive behavior is not reported as implemented. The exact original closure checks remain in `AUDIT.json.prior_findings`.

| Finding | Status | Independent result and limits | Evidence |
|---|---|---|
| F01 — A failed review can downgrade an existing blocker by ID | FIXED | Both nonapproving verdicts preserve same-ID and omitted blockers; exact-unit approving control resolves only its own findings. | E05, E09 |
| F02 — Imports accept stale core bindings and can erase newer blockers | FIXED | Current core head is checked for review and snapshot import. Finding/criterion mutations refuse with unchanged chains; supplied tests also cover other core-record invalidation. Same-review snapshot remains allowed. | E05, E08, E09 |
| F03 — Replaying an old passing run overrides a newer failed review | FIXED | Duplicate concurrent imports yield one append. An old pass cannot override a later failure or a newer unanswered packet. | E05, E07, E08 |
| F04 — Generated packets do not bind the observed repository artifact | FIXED | HMA derives artifact from observed target, requires separately bound repository root and approved base, and checks live target on import. Wrong artifact/base/root/clone controls refuse. | E01, E05, E08 |
| F05 — Final-audit reason does not enforce final-review family separation | FIXED | issuing_reason=final forces family separation even when FinalValidation=false and risk low. Normal review keeps normal independence rule. | E01, E08 |
| F06 — Receipt import ignores provider and effort drift | FIXED | Every receipt identity field is mandatory and exact, including provider and effort. Missing and wrong variants refuse without append. | E05, E08 |
| F07 — Invalid audit reports can approve and clear findings | FIXED | Invalid reports never APPROVE or clear blockers. Trailing/malformed/null findings refuse; contract-invalid but parseable reports may append ESCALATE, as explicitly permitted by A01. No assertion that safe escalation leaves the host chain unchanged. | E05, E11 |
| F08 — Unverified reference labels create new unwaivable blockers | DEFERRED-BY-HUMAN | In-scope negative behavior is fixed: unsupported/fake labels cannot create blockers. Positive reproduced-kernel and measured-budget mappings are Gate D with unchanged exit criteria. | E01, E09, E12 |
| F09 — The verify-loop evidence bridge is absent | FIXED | Real verify-loop contract journal/report plus failed check, synthetic repair and HMA fresh recapture passed. Changed envelope refuses. Imports grant no positive credit or launch authority. | E05, E06, E10 |
| F10 — Review dispatches have no short validity interval | FIXED | One-hour frozen review interval is half-open at nanosecond boundaries. Core rewind/renewal makes the prior packet stale; caller import timestamp does not set launch eligibility. | E01, E05, E08 |
| F11 — Reviewer independence checks only the latest implementation | FIXED | Dispatch tests compare reviewer against all delegated/direct recorded contributions for the unit. Earlier worker/model contribution is rejected; independent route control passes. | E01, E03, E08 |
| F12 — Reservation checks assigned to Gate C are not implemented | DEFERRED-BY-HUMAN | Reservations, launch reconciliation, replay eligibility and monotone generations remain Gate D. Exact v6/v7 test names are preserved; Gate C makes no bounded-execution claim. | E12 |
| F13 — New legacy manual-review writes bypass the consolidated import path | FIXED | Baseline records remain readable and labelled legacy_manual. First generated packet migrates the run; later manual approval and packet impersonation refuse. | E06, E13 |
| A01 — Invalid audit reports approve, clear blockers, and satisfy the host Verification gate | FIXED | Independent Python-invalid candidates cannot approve/clear/open the host gate. A complete valid control still imports and opens only the host gate after its snapshot. | E05, E11 |
| A02 — An old live-target review can clear a blocker after the worktree changes | FIXED | Tracked and untracked target changes refuse both import stages. Supplied dirty/live-target controls and independent unchanged-target control pass. | E01, E05, E08 |
| A03 — Finding namespace prefixes allow a review to clear another unit blocker | FIXED | Exact UnitID ownership replaces prefix inference. U1 review preserves U1/sub blocker. Slash-containing source finding IDs refuse, preventing ambiguous combined IDs. | E05, E09 |
| A04 — HMA-generated review packets omit the contract and findings snapshot | DEFERRED-BY-HUMAN | In-scope findings/core/plan snapshot is present and core drift is checked. Full contract-dependent packet contents are deferred with R3-02 by accepted human boundary. | E01, E08, E12 |
| A05 — Existing unprefixed core findings have no import migration | FIXED | Baseline-written unprefixed findings resolve only with safe unit attribution. Baseline active waiver survives unchanged; supplied ambiguous-owner cases remain unresolved, not reassigned. | E06, E13 |
| A06 — Required disposition, criterion recheck and independent-credit integration are absent | DEFERRED-BY-HUMAN | No positive credit or reproduced kernel/budget classification is claimed in Gate C. Unsupported links remain advisory. Positive paths require Gate D acceptance contract. | E09, E12 |
| A07 — Gate C launch-reconciliation integration is deferred in full | DEFERRED-BY-HUMAN | In-scope envelope checksum/store/root checks pass and correct control recaptures. Full namespace/reservation/generation reconciliation is explicitly Gate D. | E01, E06, E10, E12 |
| A08 — loop-evidence executes verify argv before checking terminal state | FIXED | Missing/terminal runs refuse before argv; independent in-check lock probe shows core lock held; open run recaptures without deadlock. | E05, E07, E10 |
| A09 — Legacy compatibility runs do not expose an enforcement-mode label | FIXED | Mode visible on baseline and migrated writes and host show --mode. Baseline migration refuses later manual review. | E06, E13 |
| R3-01 — A newer unanswered packet does not invalidate an older approving review | FIXED | New pending packet supersedes old imported pass for both snapshots and host gate; duplicate import race has one winner. Latest valid report/snapshot control still passes. | E01, E05, E07 |
| R3-02 — The fixed review packet still omits mandatory review inputs | DEFERRED-BY-HUMAN | Full immutable request/criteria/non-goals/budgets/policy/required-evidence packet is accepted Gate D scope. Current packet is not claimed to be that standalone acceptance packet. | E12 |
| R3-03 — The broader Gate C acceptance-boundary deferral is not approved and leaves design requirements incomplete | DEFERRED-BY-HUMAN | Lennon explicitly accepted scope §0.1 on 2026-10-17. Broader design rows remain at their prerequisite gates, with no Gate C closure/credit/execution claim. | E12 |
| C-F14 — Legacy dispatch input can impersonate an HMA-generated review packet | FIXED | Legacy host dispatch rejects any caller Packet before append, in legacy and migrated modes; genuine generated packet/import/snapshot control works. | E01, E08 |
| C-F15 — Loop recapture discards the verification output | FIXED | Distinct random stdout/stderr and exit 7 persist. Length-framed digest recomputes from retained bytes. >1 MiB per stream records explicit truncation; mutating capture is refused. | E05, E10 |
| C-F16 — Go reads case-variant JSON keys that the Python contract ignores | FIXED | Case variants cannot change raw fix-required into APPROVE. Exact decoding applies recursively to report, binding, findings, receipt and loop state. Unicode fold tests pass. | E01, E05, E11 |
| C-F17 — Go and Python disagree on required nonempty report fields | FIXED | U+001C..U+001F required fields never approve or clear. Supplied all-codepoint Go/Python whitespace parity test passes. | E01, E05, E11 |
| C-F18 — Interrupted output-artifact writes are trusted as complete on retry | FIXED | Empty/partial/symlink residue is refused and preserved; exact existing artifact reused. Atomic temporary-write/link protocol exposes complete final files only. | E01, E10 |
| C-F19 — loop-evidence executes a shell command from an imported journal | FIXED | Shell commands, wrapper/symlink entry points and forbidden loop operations refuse before execution; plain executable control runs. | E01, E05, E10 |
| C-F20 — Invalid UTF-8 report passes Go checks but fails the Python validator | FIXED | Invalid UTF-8 report cannot approve or clear; receipt and state decoding also explicitly require valid UTF-8. Python-invalid report negative control reproduced. | E01, E05, E11 |
| C-F21 — The shell filter refuses a shell name used as plain data | FIXED | Direct Capture printf %s sh runs and returns sh. Shell entry points still refuse; no marker created. | E01, E05, E10 |
| C-F22 — Executable shell scripts and wrapper aliases bypass the no-shell kernel | FIXED | Executable shebang scripts, env shebang, symlinked interpreters and wrapper aliases refuse. General compiled/interpreter programs that themselves start shells remain an explicit trusted-check limit, not claimed detected. | E01, E05, E10 |
| C-F23 — A concurrent trusted core terminal append can precede loop check execution | FIXED | Core transaction holds exclusive run lock from terminal check through recapture and core append. Independent check sees HELD; direct core append uses the same lock. | E05, E07, E10 |
| C-F24 — A review with a different approved base can clear the current unit blocker | FIXED | Separate write-once repository binding required; packet root and approved base checked before usable packet and at both imports. Wrong-root and same-base clone controls refuse. Initial root-to-opaque-identity association is still a trusted host assertion, not derived identity proof. | E01, E08, E13 |
| J8-01 — First generated packet can adopt another checkout and clear the current blocker | FIXED | A write-once independent host repository binding is required before first generated packet; packet never supplies it. Clone/same-base root mismatch refuses. Initial association remains a trusted host assertion. | E01, E08, E13 |
| J8-02 — Slash-separated finding IDs collide across exact unit owners | FIXED | Imported slash IDs refuse; exact UnitID ownership avoids prefix inference; duplicate resulting global IDs refuse before core append. | E01, E09 |
| J8-03 — Loop recapture can attach another repository artifact to an anchored unit | FIXED | Loop recapture checks bound root, approved base/identity and journal repo.root before execution. Wrong-root/base and drifted-journal controls refuse. | E01, E06, E10 |
| J9-01 — Executable aliases bypass the loop recapture operation filter | FIXED | Loop resolves executable symlinks before operation checks and runs that canonical executable. Supplied direct/absolute/alias rm/curl/git controls pass. | E01, E10 |
| J10-01 — bind-repository returns an empty next action | FIXED | Real binary bind-repository returns a nonempty next_action naming the next integration step. | E14 |
| J10-02 — loop-evidence omits the review enforcement label | FIXED | Real binary loop-evidence returns legacy_manual equal to host show --mode. | E14 |
| J11-01 — A non-approving findings snapshot recommends stage continuation | FIXED | Real binary findings-import with no blocker returns RECORD_FINDINGS_AND_REMEDIATE for fix-required, ESCALATE_TO_HUMAN for blocked, and CONTINUE_HMA_TRANSITION only for pass. Host gate agrees. | E14 |
| J11-02 — findings-import omits the review enforcement label | FIXED | Real binary findings-import includes packet_import equal to host show --mode for all three verdict controls. | E14 |

For invalid report inputs, HMA distinguishes malformed provenance from a parseable report that violates the audit contract. The former refuses without append. The latter can append a safe `ESCALATE`. A01 explicitly permits that response. Neither can approve, weaken a blocker, or satisfy the host Verification gate. This report does not claim that every invalid report leaves the host chain unchanged.

## Accepted deferral trace

The human decision on 2026-10-17 controls this boundary. These remain Gate D requirements:

| Deferred requirement | Recorded exit tests or acceptance check |
|---|---|
| Reservations, replay, admission, conservative counting, route and stop rules | `TestReservationIdempotent`, `TestConcurrentLastAttempt`, `TestUnknownLaunchCounts`, `TestCountersCarryAcrossRewind`, `TestSameFailureSignatureTwiceRefused`, `TestDifferentRouteRefused`, `TestEscalateToMustBeStop` |
| Final-slot launch eligibility | `TestFinalSlotReplayLaunchesOnce`, `TestFinalSlotReplayAfterUseNotEligible`, `TestNewIDWithoutCapacityRefused` |
| Launch association and unauthorized dispatch | `TestOneReservationPerLaunchNoReuse`, `TestUnlaunchedReservationCharged`, `TestUnreservedDispatchBlocks`, `TestEnvelopeAndAssociationMustMatchAdmission` |
| Half-open validity, revocation, and fixed generations | `TestLaunchAtValidUntilUnauthorized`, `TestLaunchAtRevocationTimeUnauthorized`, `TestValidUntilMinOfTwoRevocations`, `TestExpiryThenLaterRevocation`, `TestRenewalDoesNotReviveOldGeneration`, `TestNewGenerationNeedsNewReservation`, `TestLaunchAtEarliestEndUnauthorized`, `TestLaunchBeforeEarliestEndAuthorized` |
| HMA authority-hook endpoint and resume | `test_hma_bound_launch_requires_eligible_true`, `test_resume_replay_uses_same_dispatch_id`, against `hma host attempt` |
| Positive independent criterion credit | `TestScorecardOnlyIndependentCredit`, `TestPartialNeverVerified`; exact-target successful recapture plus independent report criterion verification |
| Reproduced kernel violations and measured budget exceedance | Real low/high severity kernel violations must BLOCK; only measured permitted budget excess is WAIVABLE; fake/nonreproducing links remain advisory. Scope §0.1 and F08/A06/R3-03 retain these checks. |
| Full acceptance-contract packet inputs | R3-02/A04 retain sentinel request/non-goal/policy/evidence checks, omitted/drifted-input refusal, and `TestAuditorClosurePacketStandaloneInputs` plus nonempty approved-contract control. |

The broader design proof matrix also spans outcome/lease work in Gate D, checkpoint enforcement in Gate E, and live end-to-end dogfood in Gate F. Gate C does not supply those proofs. The unchanged Gate B suite supplies regression evidence for approval binding, invalidation, chain recovery, host/core locking, terminal coherence, and revision-bound capture.

## Findings and remediation roadmap

No new actionable in-scope defect was demonstrated. No Gate C code change is recommended by this audit. No additional improvement is necessary for the accepted consolidation boundary. Gate D work remains governed by its existing acceptance criteria and human approval.

## Limits and uncertainty

- Fresh review conversation and no implementer conversation inherited. Isolation means context isolation, not an OS-enforced read-only mount. The host sandbox permits repository writes; this audit performed none. All code/test changes were under the output directory.
- Permitted design v7 contains a historical Gate A verdict in its header. Normative Task14 and audited code/test comments contain author statements. These were not treated as current Gate C verdicts or proof. No withheld Gate C narrative or prior audit directory was read before Phase 1.
- Receipts, host clocks, repository binding and stored records remain trusted-host evidence. A same-UID adversary, forged receipts or a hostile operating system are out of scope. The first bind-repository association of an opaque approved identity to a root is a host assertion; subsequent mismatch/drift is checked.
- An explicit executable can itself start arbitrary subprocesses. Static no-shell checks detect named shells, wrappers, symlink targets and shebang entry points; they do not prove arbitrary program behavior. Gate D must bind checks to accepted-plan authority.
- Reservation/lease enforcement, launch namespace reconciliation, positive criterion credit, full acceptance-contract packet inputs, kernel reproduction and budget classification remain human-deferred Gate D work. Gate C pass is not bounded-execution authority, end-user readiness, verified success, release approval, or Gate D/E/F completion.
- Host dogfood and approval/receipt records are synthetic fixtures. No real worker/model API, nested agent CLI, live release, or human stage approval was invoked. Real verify-loop contract functions ran locally, but no controller worker was launched.
- Tests ran from an external code copy, excluding TODO.md, docs/handoffs and .serena; all executable source/test files and dogfood scripts were copied. Baseline fixture writer used HEAD code. Temporary fixture paths may no longer exist after test cleanup; generators and full logs are retained for reproduction.
- Jev was not exposed through an allowed callable tool. User forbade model APIs. Audit judgments used the evaluator path without claiming Jev output. Coordinator preflight and actual model telemetry were not independently re-verified in this session.
- Generic prior-criteria wording about unchanged chains is interpreted for refusal paths. A01 explicitly allows safe ESCALATE for a parseable invalid report; such a host record is evidence of nonapproval, not a successful import. The report distinguishes these cases.
- Initial bulk tool output was truncated. A completeness check recovered eight permitted J8/J9/J10/J11 closure-criteria rows before Phase 1 freeze. Their original invariants were already covered by repository binding, findings namespace, recapture and response-contract review; the omitted detailed criteria were then checked directly. No withheld narrative was read.
- Initial auditor harness compilation required correction of an unused import, a record-kind name and a fixture-helper name. The baseline build required -buildvcs=false because its archive has no .git. These were auditor-harness setup errors; no product failure is inferred.

These tests falsify specific failure hypotheses. They do not prove all possible programs, concurrency schedules, platform behaviors, or same-UID attack paths safe. The pass applies only to the frozen target and accepted Gate C scope.

## Reproduction and report validation

Run `python3 run_checks.py` for the original standard checks from `source/`. The current copy includes the later auditor tests. To reproduce the recorded standard-only run, omit files named `auditor_*12_test.go` in a new external copy. `AUDIT.json.commands` records exact focused commands and logs. Build the baseline fixture writer with `go build -buildvcs=false -o ../baseline-seed ./cmd/auditseed` from `baseline/`. Build `hma` from `source/` before the real-binary fixture. Keep all such work outside the audited repository.

Auditor harness setup failures and corrections are retained in `harness-setup-notes.md`. These include the optional-false-field handling in the final output verifier. The corrected verifier passed.

The report-contract validator checks form, not the truth of findings or independence. Validation output is in `report-validation.log`. `SHA256SUMS` and the source/input manifests identify retained evidence. The `verdict` field uses the contract spelling `pass`; `gate_c_verdict` uses the requested spelling `PASS`. They express one verdict.

## Single next action — architecture §18

**Lennon: review this frozen packet for explicit Gate C acceptance.** The permission envelope is human review of this target and evidence. No worker route or repository mutation is requested. Acceptance does not authorize a core stage transition. Any stage transition still needs a fresh challenge-bound human approval. Stop this acceptance review if a target hash changes or a contradictory reproduction appears; return the affected evidence for independent review.
