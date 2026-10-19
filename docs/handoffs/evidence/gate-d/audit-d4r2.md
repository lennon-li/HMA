# D4 audit: FAIL

Dispatch: `hma-gate-d-audit-20261018-d4r2r2r2`. Date: 2026-10-18.
Modes: code audit and outcome audit.
Requested auditor route: liz / gpt-6.1-sol / high.
The underlying runtime model and access service were not independently attested.

Three independent reproductions expose high-severity defects.
Reconciliation can classify an unauthorized review launch as authorized.
The attempt hook can revive an old reservation after a recorded policy revocation.
Do not advance Gate D on this evidence.

The audit is incomplete in two required areas.
New process execution failed with `Permission denied`.
The auditor could not read and classify the prior closure criteria after Phase 1.
The auditor could not verify the final repository freeze.
These omissions do not negate the reproduced defects.

## Scope and isolation

The target is `main` at `a74c9f0901f89903ed0cb2425bc491ece4ea282d`, plus the frozen working tree.
The binary diff SHA-256 is:

```text
b128298fd4a16e7a3ba8e7fb45fc5c060dd9275ec22a61a8e66714c7a7ab1169
```

[freeze-start.json](freeze-start.json) contains the initial status and every untracked file hash.
[diff-frozen.patch](diff-frozen.patch) retains the tracked diff.
[freeze-end.json](freeze-end.json) records that final equality is **UNKNOWN**.

Isolation level: **isolated context, procedural read-only repository access**.
This available session did not contain implementer discussion or earlier verdicts.
The auditor derived requirements before reading the diff.
The operating-system permission profile could write the repository.
It did not enforce a separate read-only mount.
The auditor made no intentional repository edits or auditor Git write commands.
Tests ran in copies under this output directory.
Fixture tests create temporary Git repositories.

The auditor did not read `.serena/` or the excluded architecture amendment files.
The initial manifest hashed those files as explicitly requested.
The auditor did not read the D4 implementer handoff.
The auditor did not read other forbidden temporary audit or Claude directories.
Natural source and test comments included historical finding labels.
Those comments were artifacts, not authoritative judgments.

No available Jev tool was found.
The auditor used the documented fallback.
No nested agent CLI or model API was invoked.

## Requirements derived before the diff

The original objective is Gate D in
`docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md:50-65`.
The approved design sources were read only in the requested section ranges:

- v5: lines 10–44, §1–§2.
- v6: lines 19–26 and 35–61, §2 and §4–§5.
- v7: lines 9–41, §1–§2. The latest design controls.
- Gate C scope: lines 16–35, §0.1, especially exit-test rows 3 and 4.
- Architecture: §7, §9, §14, §18, §19 and the trusted-host model in §22.
- `AGENTS.md` and the explicit dispatch instructions.

[requirements-derived.json](requirements-derived.json) preserves the pre-diff requirements and failure hypotheses.

| ID | D4 requirement | Assessment | Independent evidence |
|---|---|---|---|
| D4-01 | Monotone minimum; fixed generations; renewal cannot revive old authority | PARTIAL | Reservation minimum controls pass. F-01 and F-02 fail. |
| D4-02 | Exactly one record per launch; no reuse; full declared namespace | PARTIAL | Zero/two-match, namespace and validity controls pass. F-03 shows an actual launch can still fail its authority binding. |
| D4-03 | Half-open validity; all applicable revocations and generation wall end | PARTIAL | Reservation boundary controls pass. Review revocation and hook restoration fail. |
| D4-04 | Correct launch mode; independent reviewer route; collision refusal | PARTIAL | Mode and collision controls pass. The actual wrong reviewer route is authorized by reconciliation. |
| D4-05 | Journal envelope, controller, root, baseline, namespace and attempts bind admission | MET | The independent seven-case journal matrix passes. Existing mismatch tests also run. |
| D4-06 | Unlaunched reservations stay charged and visible; no stage or criterion credit is granted | MET | Existing unlaunched test and source inspection. Reconciliation records findings and telemetry, not an approval. |
| D4-07 | Unauthorized dispatch becomes BLOCK; snapshots and later scans cannot erase it | PARTIAL | Existing BLOCK persistence mechanisms and independent controls pass. F-03 suppresses the initial finding by misclassifying the launch. |
| D4-08 | Gate requires clean current reconciliation and preserves human authority | PARTIAL | Core, authority and receipt-inventory checks exist. A clean result can contain F-01/F-03 misclassifications. No forged human approval was demonstrated. |
| D4-09 | Record ingestion and ledger scan order do not change validity | MET | Independent minimum, reversed revocation ingestion, host-history and scan-order controls pass. Valid stored-chain ordering is assumed. |
| D4-10 | Preserve D1–D3 and Gate B/C; do not claim D5/D6 | PARTIAL | F-02 affects D3 eligibility. Vet passes. Complete clean regression assurance and implementer-handoff claim review remain unavailable. |

MET is limited to the stated checks.
It is not a proof of correctness.
The scope excludes D5 criterion credit, general finding disposition, the full packet, and D6 amendment approval.

## Evidence and execution

Tool version: `go version go1.27.1 linux/amd64`.
The auditor used `/home/yeli/.local/go/bin/go`.
`GOCACHE` and `GOTMPDIR` were inside this output directory.
[checks.jsonl](checks.jsonl) retains argument vectors, working directories, timestamps, exit codes, and log names.
Later entries also retain the explicit environment.

The main independent command was:

```text
/home/yeli/.local/go/bin/go test -count=1 -v -run TestAuditorD4 ./internal/host ./cmd/hma
```

Working directory: `/tmp/hma-gate-d-20261018-d4r2/source`.
[independent-v3.log](independent-v3.log) contains the complete output.
The auditor checked FAIL lines and did not rely only on exit status.

| Check | Observed result | Qualification |
|---|---|---|
| Independent adversarial tests | FAIL | Review policy, correction generation, wall end, missing policy, restored-policy replay, actual reviewer route, and an exploratory policy-drift probe fail. |
| 12-case authorization matrix | PASS | Positive controls and negative cases cover start/end boundaries, zero/two matches, namespace, mode, rewind, policy and terminal ties. |
| 7-case journal matrix | PASS | Control plus six independent association/dispatch mismatches. |
| Revocation minimum and history/scan permutations | PASS | Includes expiry before revocation and reverse ingestion of two revocations. |
| Snapshot and later clean-scan BLOCK retention | PASS | Tests preserve a previously recorded unauthorized finding. |
| `go test ./...` and later uncached runs | Not fully green | Initial copy lacked Git metadata. A scratch-store guard assumes the repository is outside the temporary directory. Later process execution failed. |
| `go vet ./...` | PASS | See `check-3.log`, `baseline-vet.log`, and `baseline-vet-isolated.log`. |
| `go test -race ./...` and uncached runs | Not fully green | The early race run reports the scratch-location failure. Later runs fail while execution becomes unavailable. No complete race pass is claimed. |
| Mutation control via `-overlay` | Invalid | The mutation made variables unused. The compiler rejected it. This is not counted as a successful control. |

The first independent host test had an invalid terminal enum name.
The auditor corrected that test before v2 and v3.
The original failed build remains in `independent-tests.log`.
The initial VCS-sensitive build errors were environmental.
A raw copy of Git metadata was added only to the isolated copy.
No original Git metadata was intentionally changed.

The final regression attempts encountered broad process failures.
Their FAIL lines are not presented as demonstrated HMA regressions.
No clean full-suite result is claimed.

## F-01 — Review validity omits policy revocation and generation bounds

Severity: **high**. Confidence: **observed**.
Affected requirements: D4-01 and D4-03.

The review validity branch at `internal/host/reconcile.go:304-323` starts from the review TTL.
It applies terminal and selected rewind events.
It does not apply the policy-revocation map used by reservations.
It does not bind or clamp the review to the issuing lease generation.
The own-unit rule at `internal/engine/revocation.go:70-73` misses correction from Independent validation to Implementation authorization.

The independent tests show:

- A launch before the policy revocation is authorized. This is the positive control.
- A launch exactly at and after the policy revocation is also authorized.
- An old review remains authorized exactly at and after a correction issues a new generation.
- A review remains authorized after the issuing generation's wall end.
- A contract-bound review is authorized without current policy evidence.

The failing tests are
`TestAuditorD4RejectReviewAfterPolicyRevocation`,
`TestAuditorD4RejectOldReviewAfterCorrectionGeneration`,
`TestAuditorD4ReviewWallEnd`, and
`TestAuditorD4ReviewPolicyNotPresented`.
They use the pure reconciliation function and repository fixture builders.
They do not forge or advance a human approval.

Preferred fix, **proposed**: bind a review dispatch to its issuing generation.
Compute its end as the minimum of original TTL, generation wall end and every applicable revocation.
Apply policy revocations to reviews.
Revoke an old review when its unit enters a correction generation.
Refuse authorization when required generation or policy evidence is missing.

Verification: the pre-event and forward-progression controls must remain authorized.
The policy, correction-generation and wall-end ties must become unauthorized.
Repeat the checks through the CLI with retained generation plans and reversed event ingestion.

Effort: **medium**. Priority: **1**.
Trade-off: older packets lack explicit generation fields.
Do not infer authority when their anchored history cannot establish it.
Alternative: derive the immutable generation from the anchored core prefix instead of storing a new field.
This still needs the same strict evidence and minimum calculation.
A different review lifetime needs a human-approved contract change.

## F-02 — Policy restoration revives a revoked reservation

Severity: **high**. Confidence: **observed**.
Affected requirements: D4-01, D4-03 and D4-10.

`TestAuditorD4PolicyRestoreDoesNotReviveHook` calls the real attempt command.
It obtains one reservation.
It changes the policy.
HMA refuses the changed policy and records `policy_revocation`.
The test restores the original policy bytes.
Replay then returns:

```text
LaunchEligible:true
Replay:true
DispatchRunID:old-reservation
NextAction:LAUNCH_WORKER
err=<nil>
```

The exact reservation anchor and expiry are in `independent-v3.log`.
The original reservation ID is reused.
This contradicts the requirement that a validity end never increases.

Root cause: `cmd/hma/attempt.go:449-471` checks the currently derived lease.
That derivation reads core records and current policy.
It does not read the persisted host policy-revocation history.

Preferred fix, **proposed**: check persisted generation revocations before replay or fresh issuance grants launch eligibility.
Restored policy bytes must not revive the old generation.
Require a new approved generation and a new reservation for a new launch.
Preserve the historical charge and ineligible replay record.

Verification: run the existing reproduction.
Add controls for fresh issuance after policy restoration and for a separately approved new generation.
Hook eligibility and reconciliation must agree at the revocation tie.

Effort: **medium**. Priority: **2**.
Trade-off: policy rollback requires renewed human authority.
Alternative: represent policy revocation as a core event consumed by lease derivation.
That alternative changes the core event contract and needs broader review.

## F-03 — Reconciliation accepts the actual implementation worker as reviewer

Severity: **high**. Confidence: **observed**.
Affected requirements: D4-02, D4-04 and D4-07.

`TestAuditorD4ReceiptRouteMustMatchReviewAuthority` creates a real independent-review packet.
It writes a receipt with the implementation worker, model and provider.
The receipt still has the packet's run ID, namespace, review mode and valid launch time.

The existing review-import command rejects that receipt.
The reconciliation command returns:

```text
Authorized:[20261016T000000Z-ming-review-1]
Unauthorized:[]
Reservations:0
```

The full receipt setup and result are preserved in
[source/cmd/hma/auditor_d4_test.go](source/cmd/hma/auditor_d4_test.go)
and [independent-v3.log](independent-v3.log).

Root cause: `cmd/hma/reconcile.go:149-171` retains only run ID, mode and start time.
The actual route and launch bindings never reach reconciliation.
`internal/host/review_import.go:146-150` has stricter receipt checks.
Those import checks do not cause the reconciliation to record the required BLOCK finding.

Preferred fix, **proposed**: retain launch route, packet and target identity in the ledger scan.
Compare them with the matching dispatch before authorizing the launch.
Check the actual review route's required independence.
Apply equivalent available route checks to implementation reservations.

Verification: the failing receipt must produce `UNAUTHORIZED_DISPATCH` as BLOCK without a report import.
Vary worker, model, provider, effort, packet digest and target separately.
Correctly bound running and completed launches must remain authorized.
An unfinished report must grant no review credit.
An unfinished report alone must not invalidate a legitimate launch.

Effort: **medium**. Priority: **3**.
Trade-off: legacy receipts with missing identity fields need an explicit compatibility rule.
Alternative: share a launch-binding validator between review import and reconciliation.
Keep report completion and report-credit validation separate.

## Exploratory result and unresolved question

`TestAuditorD4GateMustDetectPolicyDrift` returns a nil gate error after a policy file edit.
The core, history and ledger inventory are unchanged.
The probe has an unlaunched reservation.
It does not prove that recomputing reconciliation would change a launch classification.
This audit does **not** count that probe as a separate blocking defect.
The contract for binding the effective policy at the transition gate remains a review question.

No separate redesign opportunity is recommended.
The smallest useful improvements are the shared checks described in the fixes.

## Prior finding closure and controlled unblinding

Phase 1 is preserved in [PHASE1.json](PHASE1.json).
Its SHA-256 is:

```text
885316a4d995bc5282b3b4f756c74fee03e7696c650dbe48a494bc5ad94d7c0f
```

The digest was computed over the exact ASCII bytes submitted to the output writer.
The SHA-256 implementation passed the standard `abc` test vector.
A filesystem re-read of this file was blocked.
[PHASE1.sha256](PHASE1.sha256) records the digest.

The auditor did not unblind.
The prior criteria file was withheld before Phase 1 as required.
After the freeze, execution remained unavailable.
The auditor could not read `/tmp/hma-gate-d-criteria/criteria-d4r2.json`.
Individual FIXED / PARTIAL / OPEN assessments are therefore **not supplied**.
No prior finding is declared closed.
Guessing their IDs or dispositions would fabricate evidence.

## Limitations and completion state

The observed defects support FAIL.
Audit completion remains limited by the execution failure.
Final repository equality is UNKNOWN.
Prior closure criteria are unassessed.
The report-validator script did not run.
The output is structured to the report contract but is **not validator-checked**.

Some source and log displays were truncated by the tool output budget.
Full subprocess logs remain available in this directory.
Tests ran against a dirty snapshot, not a revision-reproducible clean checkout.
The freeze manifest binds the intended snapshot.
No actual external deployment was exercised.
The trusted-host threat model applies.
No hostile operating-system claim is tested or made.

## Single next action — architecture §18

**Return F-01 through F-03 to the human coordinator for a bounded D4 remediation dispatch and independent re-audit.**

Outcome: FAIL. The reproductions above show incorrect authorization and revived launch eligibility.
Rationale: these defects violate the accepted D4 validity and reconciliation rules.
Owner: the human coordinator, with implementation by the designated implementer and review by a fresh auditor.
Proposed route: the existing designated implementer route.
The exact provider, model, effort and access service are unresolved.
Deterministic eligibility and exact-route `hi` preflight must run before dispatch.
Permission envelope: a separately authorized bounded D4 coding task.
This audit grants no transition, criteria-change, amendment, commit, push, publication or release authority.
Exact approval required: human authorization for that concrete remediation dispatch.
Stop and escalate if the route fails preflight, scope enters D5/D6, safe execution remains unavailable, or the frozen target cannot be verified.
The re-audit must classify the prior criteria and complete the final freeze comparison.
