# D6 architecture merge audit — PASS

**Dispatch:** `hma-gate-d-d6-arch-merge-audit-20261030-prof`  
**Date:** 2026-10-30  
**Target:** branch `main`, HEAD `0dc529f816fb5f230c16c5c6da8f0ae1fd20107f`, plus the uncommitted `docs/architecture.md` change.  
**Scope:** faithfulness and consistency of the eight-section documentation transcription. No new architecture decisions. No implementation repair or runtime certification.

## Human review layer

Observed: all **78** `[CHANGES §N]` clauses appear verbatim in their corresponding sections after removal of the draft tags. All **51** paragraph-level trace entries retain their corresponding clauses. A reconstruction of the eight edits from the draft, using the three disclosed insertion points, equals the merged file byte for byte. All retained baseline wording survives. The only changed baseline character is the final period of the old §20 advisory bullet, changed to a semicolon to extend that list.

Assessment: **PASS**. The transcription is faithful and consistent. No material contradiction or unresolved audit finding was found. The three insertion choices preserve the normative meaning. The skipped heading levels are cosmetic. The seven deferred questions are not silently resolved.

The architecture change is high-risk under §13 even though this transcription is mechanical. PASS is this audit's recommendation. It is not a human stage approval, release approval, or proof that the wider implementation conforms.

**Requested transition:** none; manual review of a dirty documentation target. **Active waivers:** none supplied or relied upon. **Exact HMA approval challenge:** not supplied. **Automatic import eligibility:** not established; no HMA-generated packet digest, `review_dispatch_id`, or `review_nonce` was supplied for this dispatch.

The original dirty target and patch are preserved here. `docs/architecture.md` was restored to HEAD. A concurrent `AGENTS.md` edit appeared after the initial freeze and was preserved. The final worktree therefore is not wholly clean. See the restoration record below.

## Identity and review independence

Auditor: **Codex**, dispatch alias **prof**. Provider: **OpenAI**. The host instructions identify this agent as **GPT-6-based**. The exact runtime model identifier and reasoning/effort setting were not supplied to this session. A previous audit's `gpt-6.1-sol / high` metadata does not attest this session and is not reused as an identity claim. Access: this hosted Codex tool session; no independently authenticated access-service profile was supplied.

The editor/orchestrator is **Claude / Anthropic**, as stated in the dispatch. On those disclosed identities, this auditor provides **genuine provider and model-family separation**: OpenAI/GPT versus Anthropic/Claude. This is a fresh review context without the editor's conversation. The dispatch supplies the editor's placement rationale; the prior design audit and house-format reports were read as requested. This is not a blind review and has no separate frozen blind phase. Runtime provenance was not independently attested.

No coding worker, subagent, nested agent CLI, or model API was launched. The orchestration-first coding rule did not require delegation of this assigned independent documentation review. The dispatch host's exact-route `hi` preflight was not supplied to the auditor; no new dispatch required a preflight here.

**JEV_UNAVAILABLE:** the exposed host tool inventory contained no approved Jev tool. Direct auditor judgment is the recorded fallback for the result and next-action recommendation. No other model was represented as Jev.

## Frozen evidence

| Evidence | Value or artifact |
|---|---|
| Initial source state | Only ` M docs/architecture.md`; 87 insertions, 1 deletion; no staged change |
| Merged document SHA-256 | `5d6b0d29d223c48f429380415f4ea835edb8102f8163872eb4f24eb98741f6fd` |
| Binary HEAD diff SHA-256 | `bd697d6f43fad918bde3a5fa1620df7254950e994b7aefa53e4381147b338845` |
| Reviewed dirty document | [architecture.merged.md](architecture.merged.md) |
| Exact binary patch | [merge.diff](merge.diff) |
| Clean baseline | [architecture.head.md](architecture.head.md) |
| Baseline comparison | [e65ce72.diff](e65ce72.diff); `git diff e65ce72 HEAD -- docs/architecture.md` is empty |
| Source freeze | [FREEZE_START.json](FREEZE_START.json), 345 tracked regular files |
| Clause-level coverage | [CLAUSE_COVERAGE.json](CLAUSE_COVERAGE.json) |
| Trace coverage | [TRACE_COVERAGE.json](TRACE_COVERAGE.json) |
| Reproduction | [audit_checks.py](audit_checks.py), [checks.log](checks.log), [CHECKS.json](CHECKS.json) |
| Full reconstruction | [architecture.reconstructed.md](architecture.reconstructed.md); [reconstruction.diff](reconstruction.diff) is empty |
| Prior approved-design audit | [audit-r7.json](audit-r7.json), copied from `docs/handoffs/evidence/gate-a/audit-r7.json` |

The prior audit records `gate_verdict=PASS` for the Gate A design re-audit. Its separate `verdict=conditional-pass` reflects that audit's disclosed partial isolation. It does not certify later runtime behavior or replace this transcription audit. The draft itself is labelled DRAFT, not merged authority. These qualifications remain intact.

## Answers and requirement coverage

### 1. Every tagged clause is present — PASS

| Destination | Tagged clauses | Result |
|---|---:|---|
| §6 | 3 | Exact match |
| §9 | 4 | Exact match |
| §10 | 13 | Exact match |
| §12.4 | 28 | Exact match |
| §14 | 12 | Exact match |
| §16 | 14 | Exact match |
| §20 | 1 | Exact match |
| §22 | 3 | Exact match |
| Total | **78** | **PASS** |

The comparison checks each clause inside its destination section. It does not merely search the whole document. The §12.4 checks also exclude adjacent §12 subsections. Requirements, conditions, equality tests, digest names, optional-field rules, intervals, and refusal outcomes are retained verbatim. The full reconstruction also checks the surrounding untagged text, lists, formula, and headings. All 51 trace rows are accounted for. No changed normative meaning was found.

### 2. Retained text is unchanged — PASS

The complete HEAD comparison has only one deleted line: the advisory-disposition bullet in §20. The same wording returns with `;` in place of its final `.`. This is list punctuation, not a requirement change. The draft's repeated §6 approval paragraphs and §10 plan-unit list match the baseline exactly after removal of the new §6 binding bullet. All other old lines remain in order. No quoted anchor or instruction to retain remaining text is violated.

The architecture file is identical at `e65ce72` and the supplied HEAD. Both suggested baselines therefore produce the same substantive merge comparison.

### 3. Insertion-point judgments — PASS for each

**§9:** Placement after the complete five-item list is appropriate. The colon-ended introduction continues to introduce that list. The contract paragraph follows the complete co-authoring procedure and precedes the existing criteria-change and advisory rules. No requirement becomes a list item or acquires a narrower scope. Inserting prose between the introduction and its list would interrupt that construction.

**§16:** Placement immediately after the validator's evaluation list follows the draft. The new rules determine which disposition applies. The retained paragraph requires every finding to cite its criterion or invariant, evidence, material impact, and one disposition from the same three-value vocabulary. It supplies reporting requirements, not an alternative classification algorithm. The criterion-failure rule does not add a fourth finding disposition: criterion credit fails through the criterion itself. The overlap therefore creates neither contradictory instructions nor a competing validator choice. The generic paragraph falling under the `Checkpoint triggers` heading is less tidy editorially, but its explicit subject, “Each finding,” remains general. It does not restrict the preceding rules to checkpoint findings.

**§22:** The subsection follows the existing hostile-OS sentence and remains before §23. This differs from an immediately-after-list placement, but retains the entire trusted-computing-base list and its closing qualification. The new paragraph still qualifies the same §22 threat model. The hostile-OS exclusion and the same-UID-host limitation are compatible. No requirement changes because of the ordering.

### 4. Internal consistency — PASS

No material contradiction was found in the merged document, including its untouched sections.

| Contract area | Assessment |
|---|---|
| Contract approval | §§6 and 9 agree on the contract binding. §6 explicitly includes `ACCEPTANCE_CRITERIA -> PLANNING`; §9's “after” wording does not exclude that required transition. |
| Legal authority | §10 names exactly the two incoming implementation-authorization edges allowed by §5.4. Host lease caches do not create stage authority. |
| Replay and capacity | Reservation replay and launch permission remain distinct. The replay rule returns an existing record even after exhaustion or expiry. `launch_eligible` still checks time, stage, policy, contract, and prior use. The already charged final slot is not a new capacity request, consistent with v6 §3. |
| Launch classes | The explicit implementation/review classes in §12.4 distinguish implementation reservations from HMA-issued review dispatches. The reservation protocol describes the implementation controller path. Reviews must satisfy their separate record and import rules; a double match is expressly unauthorized. |
| Finding classification | The new §16 classification implements the existing §9 rule. The older §16 enum does not override it. §7.2's permissive examples of policy-declared waivable classes do not require a validator to invent a new waivable finding outside these rules. |
| Review checkpoints | §§12.4 and 16 retain separate-context and §13 independence requirements. The checkpoint list adds validation occasions; it grants no exception to per-coding-change review or human stage approval. |
| Review versus human approval | §14's `APPROVE` is the review-result mapping. It does not replace the human transition approval required by §§4–6. Non-pass/unknown review results cannot grant approval. |
| Closure snapshots | §20 repeats the existing no-unresolved-`BLOCK`/`WAIVABLE` condition while adding application of the latest snapshot. This redundancy is consistent. A waived finding is resolved under the existing waiver rules; unresolved findings still block closure. |
| Trusted host | §22's trusted-host qualification is consistent with §15's detection-based append-only store. It does not turn validator prose into proof or promise prevention against a hostile host. Omitted calls can be detected through import reconciliation without a claim that HMA dispatches or prevents host launches itself. |


These are interpretations of the retained contract and its supplied design context. They do not add architecture decisions or certify every code path.

### 5. Numbered cross-references resolve — PASS

Line numbers refer to the frozen merged document, not the subsequently restored source file.

| Reference | Actual target heading | Merged line |
|---|---|---:|
| §6 | `## 6. Human approval contract` | 170 |
| §7.1 | `### 7.1 Unwaivable kernel` | 219 |
| §7.2 | `### 7.2 Waivable findings` | 233 |
| §5.4 | `### 5.4 Legal transitions and invalidation` | 142 |
| §13 | `## 13. Risk and validator independence` | 557 |
| §17 | `## 17. Failure, retry, and escalation` | 668 |

These are numeric prose references. Their destination numbers and subjects exist; no hyperlink resolver is needed. See [HEADINGS.json](HEADINGS.json).

### 6. Markdown structure — PASS; cosmetic hierarchy issue

`#### Reservation protocol` and `#### Launch classes and import reconciliation` are correctly nested beneath `### 12.4`.

`#### Review import eligibility` in §14 and the two new `####` headings in §16 skip level 3. Markdown accepts those headings. The skipped level does not move content outside the corresponding numbered section, change the text, or prevent a heading anchor from being generated. It makes the outline less regular and less convenient for heading navigation. No repository contract identified in this review requires consecutive heading levels or interprets this skip as a different normative scope. This is a cosmetic hierarchy issue, not a transcription or safety failure. No heading change was made.

### 7. Deferred questions remain deferred — PASS

| Draft open question | What the merged text does | Assessment |
|---|---|---|
| 1. Exact JSON Schema shapes and normalization | Names versioned documents and required concepts. It does not publish a schema or specify complete field order/sets. Existing opening status and §25 still defer schema publication. | No silent resolution |
| 2. Root/baseline/namespace derivation and host configuration | Requires the fields and equality checks. It does not define their derivation or configuration binding. | No silent resolution |
| 3. Nonce entropy, storage, expiry, rotation | Retains the draft's fresh random 128-bit requirement. It does not select an entropy source or lifecycle mechanism. | No silent resolution |
| 4. Plan-ceiling/envelope mapping | Retains the draft's admission comparison. It does not define an adapter mapping algorithm. | No silent resolution |
| 5. Whole-chain reducer, ties, clock assumptions | Retains the prescribed minimum and half-open interval. It does not choose an implementation mechanism or clock-resolution model. Equality at the end was already specified by the draft. | No silent resolution |
| 6. Deterministic checkpoint classifier | Retains the qualitative trigger list and fail-closed unknown case. It does not claim that a detector is designed or implemented. | No silent resolution |
| 7. D5/D6 host integration contracts | States the amendment's obligations. It does not specify the complete Obsidian adapter or receipt integration contract. | No silent resolution |

The dispatch requests eight section edits, not insertion of the draft's open-question appendix. Its absence from the architecture is not evidence that those questions have been decided. This audit does not determine which deferred implementation details later code may have implemented.

## Code-term confirmation

All eight requested terms occur in committed production code at the supplied HEAD. The complete matches are in [committed-code-terms.log](committed-code-terms.log); the per-term summary is [code-terms.json](code-terms.json). The worktree grep is also retained in [code-terms.log](code-terms.log).

| Term | Committed production reference |
|---|---|
| `hma.plan/v1` | `internal/model/contract.go:24` |
| `hma.acceptance-contract/v1` | `internal/model/contract.go:20` |
| `acceptance_contract_digest` | `internal/model/types.go:197` |
| `HOST_RESERVATION_CONFLICT` | `internal/host/host.go:116` |
| `review_nonce` | `internal/host/review_import.go:73` |
| `loop_admission` | `internal/host/host.go:40` |
| `UNAUTHORIZED_DISPATCH` | `internal/host/reconcile.go:18` |
| `launch_eligible` | `cmd/hma/attempt.go:72` |

This confirms that the names and associated code predate the uncommitted documentation edit. Grep is not proof of runtime conformance. No Go suite, race suite, schema validator, or host dogfood ran for this documentation-only transcription audit. The reproducible checks test the requested text comparison, retention, and references.

## Execution limits and restoration

The default outer command launcher failed before executing the first shell command: `bwrap: execvp codex-linux-sandbox: No such file or directory`. The authorized `require_escalated` fallback succeeded. No automatic approval rejection occurred. Command execution was therefore not confined by the missing outer launcher. Audit writes stayed within this output directory, except for the explicitly requested final restoration of `docs/architecture.md`.

Some broad tool displays were truncated. Focused reads recovered the relevant omitted text. Full source bytes and hashes are retained. The first audit-helper attempts had assertion failures caused by draft-block parsing and a redundant baseline comparison. The helper was corrected; the final run passed. These were helper failures, not target defects. See [PROCESS.md](PROCESS.md).

The first restoration guard stopped before writing because `AGENTS.md` had changed concurrently. The architecture target still matched the frozen digest. The concurrent edit adds human-output style instructions; it does not change the reviewed architecture. The updated instructions were read and applied. Their exact bytes and diff are retained as [AGENTS.concurrent.md](AGENTS.concurrent.md) and [AGENTS.concurrent.diff](AGENTS.concurrent.diff).

The second guard confirmed that the architecture target and the other 343 tracked files had not changed. It restored only `docs/architecture.md` from the preserved, verified HEAD bytes. The final state is the same HEAD, no architecture diff, no staged diff, and exactly ` M AGENTS.md` in status. That concurrent edit was preserved byte for byte. Thus the authorized audit target is restored, but the requested whole-worktree clean condition is not achieved because of independent concurrent work. This limitation does not change the transcription verdict. No commit, push, checkout, reset, branch operation, or history change ran. See [PRE_RESTORE.json](PRE_RESTORE.json) and [RESTORATION.json](RESTORATION.json).

## Single next action — architecture §18

**PRESENT_ARCHITECTURE_AUDIT_TO_HUMAN.** Claude should present this PASS packet and the exact preserved amendment at the existing human architecture gate.

**Current outcome and evidence:** PASS for the transcription; coverage and target digest are recorded above. The audit target is restored. The concurrent worktree limitation is recorded in the restoration section.

**Rationale:** The audit supports the editor's merge without a repair request. Human authority remains necessary for applicable stage and architecture approval.

**Owner and route:** Claude / Anthropic, existing orchestrator session. No worker execution is required for this handoff. The editor's exact model and effort were not supplied; no replacement route is proposed.

**Permission envelope:** Read the audit and preserved amendment; prepare the existing approval packet. This audit authorizes no commit, push, release, new coding work, or stage advancement.

**Exact approval required:** No additional approval is needed to return or present this report. Use an existing valid exact approval if one already covers the amendment. Otherwise the human must decide the applicable challenge-bound architecture/stage packet before that transition. No exact challenge was supplied here, and PASS does not manufacture one.

**Stop/escalation conditions:** Stop if the amendment bytes or bound inputs change, if required authority is absent or stale, or if an unresolved kernel failure is identified. Return changed content for the applicable review. Do not treat this manual audit as an authenticated review import or final product acceptance.
