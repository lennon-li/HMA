# Architecture amendment trace — Gate E checkpoint triggers

**Date:** 2026-10-31
**Dispatch ID:** hma-gate-e-arch-amendment-draft-20261031-r1
**Scope:** clause-level provenance mapping for `docs/handoffs/2026-10-31-gate-e-arch-amendment-draft.md`.
**Source:** "Gate A" = `docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md` r4 (FINAL). Section numbers
below are Gate A sections unless prefixed "arch" (`docs/architecture.md`).

| Amendment clause | Source location (Gate A) | Notes |
|---|---|---|
| §9 — optional `checkpoint_policy_digest` names the `hma.checkpoint-policy/v1` digest | §3.2 "Carrier (Decision P)" table; §6 Decision P | `[CHANGES §9]` at Gate A line 273 |
| §9 — same digest format as `policy_digest`; equals the retained policy's digest | §3.2 carrier table "Present" column | Retention mechanism open (draft §2 item 6) |
| §9 — absent means no policy; every changed path unclassified | §3.2 carrier table "Absent"; §3.2 bullet "With no policy…"; §3.6 §9 row | |
| §9 — `""` or `null` refused; absence is the only encoding | §3.2 "One byte form per meaning"; §3.6 §9 row "Present-but-empty is refused" | |
| §9 — `omitempty`; existing contract digests unchanged | §3.2 "Digest compatibility"; §4 "Stored records and documents" | Byte proof is a §5.5 test (draft §2 item 12) |
| §9 — policy change is a contract revision; rewinds and fires T1 | §3.2 "Rewind" | |
| §9 — older binary refuses the field; one binary governs a run | §3.2 "Forward compatibility" | |
| §10 — optional `material_milestone` on a plan unit | §3.3 table; §6 Decision M | `[CHANGES §10]` at Gate A line 291 |
| §10 — absent means `false`; only `true` accepted; `false`/`null` refused | §3.3 table "Absent" / "Present" | One-byte-form rule from §3.2 |
| §10 — existing plan digests unchanged | §3.3 last sentence | |
| §10 — `true` marks an approved material milestone for T6 | §3.4 T6 row; §6 Decision M "T6 fires only where the plan opts in" | |
| §12.4 — subsection scope: contract-bound runs; non-contract keep today's packet path | §3.5 opening; §6 Decision D10 (1); §4 "Non-contract runs" | |
| §12.4 — HMA runs the classifier at packet issuance over its own complete capture | §3.5 E0 step 1; §1.6 "Changed-path observation" | |
| §12.4 — order `per_change` < `checkpoint` < `final` | §3.4 `CheckpointDecision.Level` comment | |
| §12.4 — host may equal or exceed the minimum, never below | §3.1; §3.6 §12.4 row; §3.5 E0 step 2 | |
| §12.4 — below-minimum refused, `HOST_CHECKPOINT_REASON_TOO_LOW`, derived level reported | §3.5 E0 step 2 | Reason name PROPOSED (draft §2 item 2) |
| §12.4 — no silent upgrade (no-silent-substitution) | §3.5 E0 step 2 | Ties to existing arch §12.4 "Silent substitution is prohibited" |
| §12.4 — packet header shows fields; packet digest covers them | §3.5 E0 step 3 | |
| §12.4 — authoritative contract risk = proven-bound contract document, head re-checked under lock | §3.5 E0 step 4 "Authoritative input"; §6 Decision D9 "Provenance" | |
| §12.4 — required review risk = `max(contract risk, risk floor)` | §3.4 `RequiredRisk`; §3.4 rule "RequiredRisk"; §3.6 §12.4 row | |
| §12.4 — dispatch risk below required refused, `HOST_REVIEW_RISK_BELOW_REQUIRED`; host field not rewritten | §3.5 E0 step 4 "Check" | Closes G-R on the packet caller (§2.3 G-R) |
| §12.4 — independence evaluated against dispatch risk; high yields family separation | §3.5 E0 step 4 "Effect" | |
| §12.4 — `family_separation: true` also requires family separation (T7, disputed) | §3.5 E0 step 4 "Remaining term" | |
| §12.4 — contract-bound reviews only via HMA-built packet | §3.5 E0 step 5; §6 Decision D10 (1); §3.6 §12.4 row | Closes G-R on the second caller |
| §12.4 — packet-less review dispatch refused on contract-bound run, `HOST_REVIEW_IMPORT_REQUIRED` | §3.5 E0 step 5 | Existing reason. "General host dispatch route" = `hma host dispatch` / `runHostDispatch` (§1.3); CLI name kept out per arch §6 convention |
| §12.4 — packet-less and legacy reviews never satisfy E2/E3 on any run | §6 Decision D10 (1); §4 "Legacy (packet-less) reviews" | |
| §12.4 — legacy `final_validation: true` does not substitute | §4 "Legacy (packet-less) reviews" | |
| §12.4 — HMA writes binding fields; host cannot | §3.5a opening; §1.4 | |
| §12.4 — fields omitted when absent | §3.5a opening "All use `omitempty`" | |
| §12.4 — `checkpoint_schema` row | §3.5a table `CheckpointSchema` row | |
| §12.4 — `minimum_issuing_reason` row | §3.5a table `MinimumIssuingReason` row | |
| §12.4 — `checkpoint_triggers` row | §3.5a table `CheckpointTriggers` row | |
| §12.4 — `unclassified_paths` row | §3.5a table `UnclassifiedPaths` row | Gate A bound "at most `maxObservedEntries`" omitted as an implementation constant; manifest limit already refuses a partial manifest (§1.6) |
| §12.4 — `family_separation` row | §3.5a table `FamilySeparation` row | |
| §12.4 — `risk_floor` row | §3.5a table `RiskFloor` row | |
| §12.4 — `contract_risk` row | §3.5a table `ContractRisk` row | |
| §12.4 — `bound_contract_digest` row | §3.5a table `BoundContractDigest` row | |
| §12.4 — `checkpoint_policy_digest` row | §3.5a table `CheckpointPolicyDigest` row | |
| §12.4 — `manifest_digest` row | §3.5a table `ManifestDigest` row | Exact encoding open (draft §2 item 4) |
| §12.4 — `checkpoint_covers` row; becomes next baseline once approved | §3.5a table `CheckpointCovers` row; §3.5a "Field renamed from r1" | Sort normativity open (draft §2 item 3) |
| §12.4 — non-schema fields interpreted only with schema; binding with field but no schema refused | §3.5a rule "Absent vs false"; §5.5 `TestCheckpointFieldsWithoutSchemaRefused` | |
| §12.4 — with schema, absent = false/empty; without, unknown and never counts | §3.5a rule "Absent vs false" | |
| §12.4 — existing records and anchors unchanged | §3.5a rule "Record bytes" | Gate A notes "not yet proven by a test" (draft §2 item 12) |
| §12.4 — only new packet files change | §3.5a rule "Packet bytes" | |
| §12.4 — recorded `contract_risk`/`risk_floor` trusted only while `bound_contract_digest` matches | §6 Decision D9 "Provenance"; §3.5 E0 step 4 "Record" | |
| §12.4 — IV → R&C is a host-gated edge requiring E2 when any trigger fires | §3.6 §12.4 row; §3.5 E2; §2.3 G-V | `[CHANGES §12.4]` at Gate A line 448 |
| §12.4 — per-change review unchanged; classification never removes it | §3.5 E1 | |
| §13 — "human confirms or raises it" gains exception clause | §6 Decision D9 (1) "Flag"; §3.7 bullet D9 | Explicit wording change, listed under "Changed wording" |
| §13 — classifier supplies the deterministic floor | §3.6 §13 row; §3.4 "RiskFloor"; §2.4 first bullet | `[CHANGES §13]` at Gate A line 377 |
| §13 — floor `high` on T3, observed T4, T5; else `low` | §3.4 "RiskFloor" | |
| §13 — T8 and allowed-only T4 do not raise the floor | §3.4 "RiskFloor"; §6 Decisions U (2), D11 (2) | |
| §13 — floor only raises | §3.4 "RiskFloor" "It only raises"; §3.1 | "never lowers contract or host risk" restates "only raises" |
| §13 — raised floor applies at once, shown in later packets, no re-confirmation | §6 Decision D9 (1) option text and "Decided" | |
| §13 — contract keeps lower human-confirmed risk | §6 Decision D9 "Flag" | |
| §13 — only a downgrade needs a human, as an explicit waiver | §6 Decision D9 "Flag" | Waiver path not designed (draft §2 item 9) |
| §13 — amends "human confirms" for raised floors only; raised floor only strengthens | §6 Decision D9 "Decided" rationale; §3.7 | |
| §13 — trigger list: "external side effects" → "observed external side effects" | §6 Decision D11 "Accepted by Lennon" ("needs a [CHANGES §13] amendment stating that the automatic … trigger covers effects HMA observes"); §3.7 last paragraph | Explicit wording change |
| §13 — observed = path classed `external_effect`/`irreversible` | §6 Decision D11 "observed" definition; §3.4 T4 row | |
| §13 — permission-only effect is not high-risk; fires checkpoint, normal independence | §6 Decision D11 (2) "Decided"; §3.4 "Family separation" last bullet | |
| §13 — HMA does not observe runtime effects in v1 | §2.1 T4 row; §6 Decision D11 | |
| §13 — unobserved runtime effect relies on human-confirmed contract risk | §6 Decision D11 "Reading of §13" and "Accepted by Lennon" | |
| §13 — narrows unqualified text; accepted v1 limitation | §6 Decision D11 "Accepted by Lennon, 2026-10-31" | |
| §16 Edit A — "final outcome closure" = `VERIFIED_SUCCESS`, `VERIFIED_WITH_WAIVERS`, `PARTIAL` | §6 Decision D7 (2) and "narrows the literal §16 text"; §3.4 `FinalAuditOutcomes`; §3.7 bullet D7; §3.6 §5.2/§16 row | |
| §16 Edit A — stopping outcomes claim no success, do not fire T7, keep requirements | §6 Decision D7 rationale; §3.5 E3 last bullet | |
| §16 Edit A — narrows earlier unqualified text | §6 Decision D7 "(2) narrows the literal §16 text" | |
| §16 Edit A — "cannot be classified" = no rule matches, or no policy | §6 Decision D8 (1) "Flag"; §3.2 bullets "no rule matches" / "With no policy" | |
| §16 Edit A — a matched path, including `ordinary`, is classified | §6 Decision D8 (1); §3.7 bullet D8 ("policy misclassification is not 'cannot be classified'") | |
| §16 Edit A — named v1 gap: mislabeled `ordinary` loses checkpoint and family separation; per-change review only | §6 Decision D8 "Named residual"; §6 Decision C | |
| §16 Edit A — policy bound into human-accepted contract digest | §6 Decision D8 "Named residual" | |
| §16 policy — `hma.checkpoint-policy/v1`, canonical JSON, strict decoder | §3.2 opening | |
| §16 policy — bound by `checkpoint_policy_digest` | §3.2 carrier; §6 Decision P | |
| §16 policy — `schema`, `rules`, rule `id`/`glob`/`classes` | §3.2 JSON example | Field set read from the example; full schema open (draft §2 item 1) |
| §16 policy — every field required | §3.2 last bullet | |
| §16 policy — closed class vocabulary; unknown class refuses | §3.2 bullet 1; §5.1 `TestCheckpointPolicyUnknownClassRefused` | |
| §16 policy — glob `*` one segment, `**` any segments | §3.2 bullet 2 | Edge cases open (draft §2 item 5); "standard library only" omitted as implementation |
| §16 policy — path only, no content or language detection in v1 | §3.2 bullet 2; §6 Decision C (1) | |
| §16 policy — union of classes; `ordinary` never cancels | §3.2 bullets 3–4 | |
| §16 policy — unmatched path unclassified; no policy → all unclassified | §3.2 bullets 5–6 | |
| §16 policy — only org and project policy may declare `ordinary` | §6 Decision O | Layer-attribution mechanism open (draft §2 item 7) |
| §16 policy — literal catch-all `**: ordinary` refused, with reason | §6 Decision O; §5.1 `TestCheckpointPolicyCatchAllOrdinaryRefused` | |
| §16 policy — broad-glob equivalents not detected | §6 Decision O "Known limitation" | |
| §16 policy — no kernel-reserved paths in v1 | §6 Decision O "no kernel reservations in v1" | |
| §16 classifier — pure, deterministic | §3.4 opening; §3.4 "Determinism" | |
| §16 classifier — input list | §3.4 `CheckpointInput` | `Contract == nil` path omitted: reachable only under rejected D10 (2) |
| §16 classifier — manifest projection fields | §3.4 `ChangedPath` | |
| §16 classifier — rename classified as both paths | §1.6 "A rename appears as a delete plus an add"; §5.1 `TestCheckpointRenameAcrossClasses` | |
| §16 classifier — sorted, input-order independent | §3.4 "Determinism" | |
| §16 classifier — contract-bound runs only | §3.5 opening; §6 Decision D10 (1) | |
| §16 classifier — T1 row | §3.4 rules table T1; §3.4 constant `CONTRACT_OR_PERMISSION_CHANGED` | |
| §16 classifier — T2 row | §3.4 rules table T2; constant `HIGH_OR_CRITICAL_RISK` | |
| §16 classifier — T3 row | §3.4 rules table T3; constant `INTERFACE_SCHEMA_OR_PROTOCOL` | |
| §16 classifier — T4 row (allowed / observed forms) | §3.4 rules table T4; constant `IRREVERSIBLE_OR_EXTERNAL_EFFECT`; §6 Decision D11 | |
| §16 classifier — T5 row | §3.4 rules table T5; constant `SECURITY_APPROVAL_ROUTING_EVIDENCE_OR_CLOSURE` | |
| §16 classifier — T6 row | §3.4 rules table T6; constant `APPROVED_MATERIAL_MILESTONE` | |
| §16 classifier — T7 row | §3.4 rules table T7; constant `FINAL_OUTCOME_CLOSURE`; §6 Decision D7 | |
| §16 classifier — T8 row | §3.4 rules table T8; constant `UNCLASSIFIED`; §6 Decision D8 | Second T8 clause (`Contract == nil`) omitted per D10 (1) |
| §16 classifier — baseline = `checkpoint_covers` on latest approved checkpoint/final review | §3.4 "Baseline"; §3.5a `CheckpointCovers` | "Level" means the issued `issuing_reason` (draft §2 item 8, resolved) |
| §16 classifier — no such review → first acceptance digest + first lease generation's permission classes | §3.4 "Baseline" | |
| §16 classifier — first contract does not fire T1; re-acceptance fires until checkpoint passes | §3.4 "Baseline" | |
| §16 classifier — level rule (`final` if T7; `checkpoint` at IV with any trigger; else `per_change`) | §3.4 "Level"; §6 Decision D8 (1) | |
| §16 classifier — contract-bound unit with no trigger needs no checkpoint review | §6 Decision D8 (1); §3.5 E2 "a contract-bound unit with no trigger passes E2" | |
| §16 classifier — risk floor defined in §13; required review risk | §3.4 "RiskFloor", "RequiredRisk" | Cross-reference only |
| §16 classifier — family-separation rule | §3.4 "Family separation" formula | |
| §16 classifier — T3/observed T4/T5 via floor; T7 without risk | §3.4 "Family separation" bullets 1 and 3 | |
| §16 classifier — T1, T6, T8, allowed T4 alone: normal independence | §3.4 "Family separation" bullet 4; §6 Decisions U (2), D11 (2) | |
| §16 classifier — persisted derived value trusted only when it names the bound contract digest | §3.1 last sentence | |
| §16 gates — apply to contract-bound runs | §3.5 opening; §6 Decision D10 (1) | |
| §16 E2 — on IV → R&C, recompute at gate time on live target | §3.5 E2 | `[CHANGES §16]` at Gate A line 448 |
| §16 E2 — APPROVE, isolated, findings snapshot applied, no unresolved BLOCK/WAIVABLE | §3.5 E2 bullet 1 | |
| §16 E2 — `checkpoint_schema` present; `issuing_reason` checkpoint or final | §3.5 E2 bullet 2 | |
| §16 E2 — dispatched after VERIFICATION → IV approval | §3.5 E2 bullet 3 | |
| §16 E2 — triggers superset; policy digest equal | §3.5 E2 bullet 4 | |
| §16 E2 — bound contract digest equal; risk ≥ gate-time required; re-acceptance stales | §3.5 E2 bullet 5 | |
| §16 E2 — artifact equals the transition's artifact | §3.5 E2 bullet 6 | |
| §16 E2 — three refusal reasons | §3.5 E2 "Refusal reasons (PROPOSED)" | Draft §2 item 2 |
| §16 E2 — no trigger → passes with no checkpoint review; other gates still apply | §3.5 E2 "Under Decision D8 (1)"; §5.2 `TestCheckpointNotRequiredWithoutTriggers` | |
| §16 E3 — every relied-on review: schema present, `issuing_reason` equal to `final`, bound digest equal | §3.5 E3 bullets; §6 Decision F (1) | `[CHANGES §16]` at Gate A line 460 |
| §16 E3 — `HOST_FINAL_AUDIT_MISSING` | §3.5 E3 | Draft §2 item 2 |
| §16 E3 — family separation, isolation, artifact match are existing requirements | §3.5 E3 paragraph after bullets | |
| §16 E3 — `VERIFIED_*`: part of the closure credit check | §3.5 E3 outcome list bullet 1 | Latency/Phase B timing → draft §1.7 Decision G note |
| §16 E3 — `PARTIAL`: snapshot artifact, re-check credit for `PASSED`, apply gate; `WAIVED` need nothing; no-active-waiver n/a | §3.5 E3 outcome list bullet 2; §6 Decision D7 (2) | |
| §16 E3 — stopping outcomes do not run the gate | §3.5 E3 outcome list bullet 3; §6 Decision D7 (2) | |
| §16 gates — one `final` review in IV on closure artifact can satisfy E2 and E3 | §6 Decision D; §3.5 "Review count" | |
| §16 gates — stays separate from per-change review | §6 Decision D | |
| §16 gates — credit re-issued after each newer review, in packet mode | §3.5 "Review count" | Scoped to packet mode via `AnswersLatestReviewDispatch` |
| §16 gates — non-contract runs: gates n/a; success claim refused (§20) | §6 Decision D10 (1) | |
| §16 gates — residual: non-contract run may leave IV without checkpoint; accepted because no success claim | §6 Decision D10 "Residual" | |
| §16 gates — missing/stale review is a gate refusal, not a finding; not waivable | §3.7 first paragraph; §6 Decision K | §7.1 unchanged (draft §1.7) |
| §20 — last bullet period → semicolon | Structural only | Same edit pattern as the D6 merge of the 2026-10-16 draft |
| §20 — bullet: run is contract-bound | §3.6 §20 row; §6 Decision D10 (1) | `[CHANGES §20]` at Gate A line 460 |
| §20 — bullet: every credited unit has a passing `final` review of the closure artifact | §3.6 §20 row; §6 Decision F (1); §3.5 E3 | |
| §20 — success-claiming outcome requires contract-bound run; `HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM` | §6 Decision D10 (1); §5.3 `TestNonContractRunCannotClaimSuccess` | |
| §20 — non-contract run may still stop FAILED/BLOCKED/UNKNOWN/ABORTED | §6 Decision D10 (1) | |
| §20 — `PARTIAL` also requires the final gate for its `PASSED` criteria | §3.5 E3 `PARTIAL` bullet; §6 Decision D7 (2) | |

## Gate A `[CHANGES §N]` tag coverage

| Gate A location | Tag | Represented in amendment |
|---|---|---|
| §3.2 carrier (line 273) | §9 | §1.1 |
| §3.3 (line 291) | §10 | §1.2 |
| §3.4 RiskFloor (line 377) | §13 | §1.4 floor paragraphs |
| §3.5 E2 (line 448) | §16, §12.4 | §1.5 E2; §1.3 host-gated exit |
| §3.5 E3 (line 460) | §20, §16 | §1.6; §1.5 E3 |
| §3.6 table rows (lines 521–527) | §5.2/§16, §9, §10, §12.4, §13, §16, §20 | All; §5.2 placed in §16 (draft §1.7) |
| §3.7 D7 / D8 / D9 / D11 bullets (lines 539–547) | §16, §16, §13, §13 | §1.5 Edit A; §1.4 |
| §6 Decision C, D7, D8 (lines 710, 787, 810, 821) | §16 | §1.5 Edit A, policy |
| §6 Decision P (line 712) | §9 | §1.1 |
| §6 Decision P option (2) (line 713) | §12.3 | **Not carried**: rejected option |
| §6 Decision M (line 736) | §10 | §1.2 |
| §6 Decision D9 (line 836) | §13 | §1.4 |
| §6 Decision D11 (line 884) | §13 | §1.4 |

## Gate A content intentionally not carried into normative text

| Item | Gate A location | Reason |
|---|---|---|
| Go types, functions, file:line citations | §1, §3.4, §3.5 | Implementation names; architecture states semantics only |
| `Contract == nil` T8 clause | §3.4 T8 row | Reachable only under rejected Decision D10 (2) |
| `maxObservedEntries` bound on `unclassified_paths` | §3.5a table | Implementation constant |
| "Standard library only" for glob matching | §3.2 bullet 2 | Implementation constraint (arch §23 governs dependencies) |
| Next actions `ISSUE_CHECKPOINT_REVIEW`, `ISSUE_FINAL_REVIEW` | §3.5 "Next actions" | No `[CHANGES §18]` tag; §18 not in §6 list (draft §2 item 10) |
| Decision G sequencing | §6 Decision G | Implementation order; one constraint noted in draft §1.7 |
| Decision S | §6 Decision S | Out of Gate E |
| Decision U reading of "materially unknown environments" | §6 Decision U | Reading, no settled text change (draft §2 item 11) |
| Test plan | §5 | Implementation |
| Compatibility notes for tests, dogfood, host layer, `task14-host-orchestration.md` | §4 | Implementation and doc follow-up, not architecture |

## Open questions

| Open question (draft §2) | Origin | Reason it is unresolved |
|---|---|---|
| 1. Policy JSON Schema | §3.2 (example only) | Schema publication deferred, as for contract and plan |
| 2. Stable identifier enum | Gate A header "every new name … is PROPOSED" | Not decided in §6 |
| 3. `permission_classes` sort normativity | §3.5a `CheckpointCovers` "sorted"; §3.4 T1 "as sets" | Gate A silent on byte-form status |
| 4. Manifest digest encoding | §3.5a `ManifestDigest`; §3.4 `ChangedPath` (no JSON tags) | Projection encoding unspecified |
| 5. Glob edge cases | §3.2 bullet 2 | Gate A silent |
| 6. Policy retention mechanism | §3.1, §3.2 carrier "retained policy" | Gate A silent |
| 7. Layer attribution for `ordinary` | §6 Decision O | Gate A silent on mechanism |
| 8. "Level" = `issuing_reason` vs `minimum_issuing_reason` | §3.4 "Baseline"; §3.5 E2, E3 | Resolved in draft §2 item 8: "level" means the issued `issuing_reason`, confirmed by Decision D's reachability requirement |
| 9. Downgrade waiver path | §6 Decision D9 "Flag"; §3.5 E0 step 4 | No waiver operation designed |
| 10. §18 next actions | §3.5 "Next actions" | Untagged |
| 11. Decision U reading of §13 | §6 Decision U | Reading only |
| 12. Unverified implementation assumptions | §4 "Unverified assumption", "Not verified"; §5.5 | Proof is a Gate E red test |
| 13. 2026-10-16 open questions 1–5, 7 | `2026-10-16-arch-amendment-draft.md` §2 | Unaffected; only item 6 resolved |
