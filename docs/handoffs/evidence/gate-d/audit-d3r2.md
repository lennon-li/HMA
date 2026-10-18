# D3 independent audit

**Verdict: FAIL.**

The frozen source passed `go test ./...`, `go vet ./...`, and `go test -race ./...` in the audit copy. All 12 named D3 exit tests passed. No named exit test skipped. All four prior findings meet their stated closure criteria.

The independent probes found two remaining endpoint defects. A new reservation can use attempt data that contradicts saved controller state. Omitting a saved repeated-failure signature can obtain a launch grant. The first-consumption path can also grant a dispatch ID that already exists in the ledger. These observations support FAIL despite the passing repository tests.

**Scope:** `hma host attempt`, D3. The audit covers code and endpoint outcomes. It excludes D4 import reconciliation and D5 criterion credit or findings disposition. The D3 handoff explicitly excludes those items. This packet grants no stage transition or broader permission.

**Isolation:** isolated auditor context on a shared host. No prior implementer chat or audit verdict was supplied. No independent machine or OS-enforced read-only mount was established. Tests ran in a source copy. The audited repository remained unchanged. Direct endpoint probes do not prove that the unmodified trusted controller emits inconsistent requests or that a duplicate worker actually launches.

**Dispatch:** `hma-gate-d-audit-20261018-d3r2`. The task declares `jax / gpt-6.1-sol / high / review`. The audit did not independently attest runtime identity or access-service telemetry. It launched no worker, nested agent CLI, or model API. No callable Jev tool was available.

## Frozen target and provenance

- Repository: `/home/yeli/repos/HMA`, branch `main`.
- HEAD: `0d90eed831741db297c220ee83f123e2554e06a0`.
- Tracked binary diff SHA-256: `70b827844a42b0d90e6d0a7a6984f8ef73e2798e8f7de69d12a6d2c59828e072`.
- Phase 1 SHA-256: `fc5e4888b908c456ed4c2dc85c03ad789972f154ef299cd63c5a530c32ea6fab`.
- Phase 1 freeze time: `2026-10-18T01:57:01.775998+00:00`.
- Toolchain: `go version go1.27.1 linux/amd64`.
- [Start freeze](freeze-start.json), [end freeze](freeze-end.json), and [comparison](freeze-verification.json) retain status and every untracked-file hash. HEAD, branch, status, diff digest, and all untracked hashes match.
- [Tracked diff](tracked-diff.binary), [source hashes](source-hashes.json), [authority-source hashes](authority-source-hashes.json), and [145-file source-copy manifest](snapshot-manifest.json) retain the technical target.
- The untracked `hma` binary was hashed. It was not executed or assessed. Tests built binaries from the frozen source copy.
- `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md` were excluded from substantive review.

The authoritative objective is the outcome-consolidation handoff §5, Gate D. The approved source rules are Gate A v3 §3.1–§3.2, v4 §3–§4, v5 §1–§2, v6 §2–§3 and §5. The latest rule controls conflicts. Gate C scope §0.1 rows 1, 2, and 5 supply the D3 exit tests. Architecture §§14, 17, 18 and AGENTS.md remain normative. The verify-loop functions `authority_request`, `invoke_authority_hook`, and `authorize_launch` define the endpoint interface.

[Requirements before diff inspection](requirements-before-diff.json) retain the initial reconstruction and failure hypotheses.

## Requirement coverage

`MET` means the scoped evidence met the criterion. `PARTIAL` means some required behavior passed and a material counterexample remains. No claim here covers all possible executions.

| ID | D3 requirement | Decision | Evidence |
|---|---|---|---|
| D3-01 | Replay first; same key/request returns the same charged record; changed request conflicts. | **MET** | Replay/conflict test; eight final-slot replays; one reservation. |
| D3-02 | Admission binds the envelope and remaining ceilings without charging. | **MET** | Budget/association negatives; zero-charge admission; later consumption uses the admitted envelope. |
| D3-03 | Consumption charges once; unknown launches stay counted; failure guard applies. | **PARTIAL** | Charge/restart tests pass. Saved repeated-signature omission grants a new attempt: D3R2-01. |
| D3-04 | Counters survive rewind; concurrent callers cannot share the last slot. | **MET** | Eight controllers compete for one slot: one grant. Rewind and generation controls pass. |
| D3-05 | Exact route; stop escalation; repeated material failure stops the route. | **PARTIAL** | Wrong route and non-stop escalation refuse. Omitted saved signature bypasses refusal: D3R2-01. |
| D3-06 | Eligibility enforces active issuing lease, unused ID, and final-slot replay. | **PARTIAL** | Replay, expiry, stage, policy, contract and generation controls pass. Used first-consumption ID grants: D3R2-02. |
| D3-07 | Bind the request to controller state, envelope, store, run, unit, root, baseline and namespace. | **PARTIAL** | Association controls pass. Current attempt ID/index/signature contradictions grant: D3R2-01. |
| D3-08 | Real endpoint requires eligible true and preserves the dispatch ID on replay. | **MET** | Both named D5 endpoint tests and independent controller-boundary exercise pass. |
| D3-09 | Preserve D1/D2 and Gate B/C safety; claim no D4/D5 completion. | **MET** | Full test, vet and race suites pass. The handoff excludes D4/D5. |

Coverage: **5 MET; 4 PARTIAL**. Detailed sources, observations and evidence references are in [AUDIT.json](AUDIT.json).

## Verification and negative controls

| Check | Exit | Result and evidence |
|---|---:|---|
| `go test ./...` | 0 | [Full test log](verified-test.log) |
| `go vet ./...` | 0 | [Vet log](baseline-vet.log) |
| `go test -race ./...` | 0 | [Full race log](verified-race.log) |
| 12 named D3 exit tests, `-race -v -count=1` | 0 | 12 pass; 0 skips. [Log](d3-exit-tests.log) |
| Independent D3 tests, `-race -overlay -v -count=1` | 1 | 8 top-level groups pass; 3 fail. Five failing leaf cases support two findings. [Log](independent-frozen.log) |
| Prior-finding closure controls, `-race -overlay` | 0 | 21 leaf controls pass. [Log](closure-controls.log) |
| Deliberate clock-before-lock regression | 1 | The control detects the regression. [Log](negative-control-clock.log) |
| Deliberately disabled failure-signature guard | 1 | The control detects the regression. [Log](negative-control-failure.log) |

The initial full test and race runs failed when the endpoint tests tried to add VCS metadata to a build from the source copy. The repeated runs used `GOFLAGS=-buildvcs=false` and passed. The [initial test log](baseline-test.log) and [initial race log](baseline-race.log) remain available. These were audit-copy build failures, not source defect findings.

All assessed commands have a corresponding `*.result.json` file. These files retain the exact argument vector, working directory, start time, duration, exit status and complete log path. AUDIT.json adds environment values and log digests. Complete logs have no output truncation. Test caches and temporary Git fixtures stayed under this audit directory.

The [independent tests](audit_negative_test.go) reuse the repository setup helper. The assertions and failure cases were written for this audit. The real-controller boundary test executes the real hook subprocess. It stubs only controller persistence and stop-report presentation. It does not execute the top-level resume workflow or launch a worker. [Closure controls](audit_closure_test.go) were added only after Phase 1 was frozen.

To reproduce the independent failures from this retained packet:

```text
python3 /tmp/hma-gate-d-20261018-d3r2/run-check.py reproduce-independent /home/yeli/.local/go/bin/go test -race -overlay=/tmp/hma-gate-d-20261018-d3r2/overlay.json ./cmd/hma -run TestAudit -v -count=1
```

Expected exit: `1`. Expected failing groups: `TestAuditRepeatedFailureOmittedAgainstState`, `TestAuditFreshRequestForUsedDispatch`, and `TestAuditCurrentAttemptStateBinding`.

## Findings and proposed remedies

### D3R2-01 — New reservations do not bind the current attempt or failure signature to controller state

Severity: **high**. Certainty: **observed**. Fix status: **OPEN**.

parseAttemptRequest validates the envelope and association, but does not compare saved attempt n, dispatch_run_id, route or last_check.signature with the request. Consumption trusts req.LastFailureSignature. A request with index 99 or another dispatch ID was granted. A request with the repeated saved signature was refused; the same attempt with signature omitted was granted and charged. This is an API binding failure under the explicit audit requirement. It requires an inconsistent caller request; an exploit through the unmodified trusted verify-loop controller was not demonstrated.

Evidence:

- `cmd/hma/attempt.go:185`: Controller state is read, but the parser returns after envelope and baseline checks without checking st.Attempts.
- `cmd/hma/attempt.go:308`: The repeated-failure rule uses only the request signature.
- `independent-frozen.log:55`: The repeated-signature refusal control succeeded, then omitted signature yielded launch_eligible:true for audit-fail-3.
- `independent-frozen.log:118`: Dispatch ID, attempt index, and signature mismatches each yielded a fresh launch grant.

Preferred fix (**proposed**, not validated): Decode saved attempt n, dispatch_run_id, route and last_check.signature. Before creating a new reservation, require a matching pending controller attempt and derive the failure signature from that saved attempt. Reject contradictory request fields. Preserve replay-first lookup and historical reservation returns, with independent live eligibility.

Reason: The endpoint must bind new authority to the controller state it claims to represent and must not let an omitted wire field bypass the saved failure evidence.

Verification:

- Run TestAuditCurrentAttemptStateBinding and TestAuditRepeatedFailureOmittedAgainstState against the revised endpoint. Expected: Every contradiction refuses without a new reservation. The repeated material signature remains refused when omitted on the wire.
- Run the real-controller boundary, historical replay, resume, admission and last-slot tests with real pending-attempt fixtures. Expected: Valid saved requests and historical idempotent replays retain their specified behavior.

Trade-offs: Existing synthetic fixtures with empty attempts must become realistic. Do not add a controller-lock acquisition that deadlocks a controller holding that lock while calling the hook. The trusted controller snapshot remains the source of association; this does not solve hostile-host tampering.

Alternative: Generate the request in a dedicated trusted adapter from a saved attempt snapshot and require HMA to validate the bound snapshot digest. This needs coordinated controller-interface changes.

Effort: **medium**. Apply the bounded fix, preserve the evidence, and obtain independent re-test before claiming D3 complete.

### D3R2-02 — First consumption grants an already used dispatch ID

Severity: **medium**. Certainty: **observed**. Fix status: **OPEN**.

The new-reservation branch sets launch_eligible:true without checking the namespace runs/<dispatch_run_id> path. The used-ID check exists only in replayResult. Creating that path before the first hook call yields a new launch grant. This finding concerns endpoint eligibility, not D4 import reconciliation. No duplicate real worker launch was attempted; the trusted dispatcher may independently reject that ID.

Evidence:

- `cmd/hma/attempt.go:333`: The first-consumption response defaults to an eligible launch and checks only time after commit.
- `cmd/hma/attempt.go:364`: The ledger existence guard occurs only in replayResult.
- `independent-frozen.log:111`: An existing runs/audit-already-used directory yielded launch_eligible:true and a new reservation.

Preferred fix (**proposed**, not validated): Apply the fail-closed namespace dispatch-existence guard to first consumption as well as replay. Prefer refusing before append if the path already exists. If detected after a committed charge, retain the charge and return launch_eligible:false.

Reason: A dispatch ID that is already used must not receive fresh endpoint launch eligibility. This can be done without implementing D4 reconciliation.

Verification:

- Run TestAuditFreshRequestForUsedDispatch with a directory, file and symlink at runs/<id>, plus an absent-path positive control. Expected: Existing or unreadable paths grant no launch; an absent path can grant exactly one charged reservation.
- Re-run final-slot, replay, unknown-launch and charge-conservation tests. Expected: No cancellation or charge deletion is introduced; unused reservations remain recoverable.

Trade-offs: Ledger inspection is another filesystem check under run locks. A check alone does not atomically prevent a later launcher race; the trusted dispatcher must continue enforcing dispatch-ID uniqueness.

Alternative: Make the dispatcher atomically reject reused IDs and document the endpoint grant as provisional. This is weaker evidence for the explicit endpoint used-slot criterion.

Effort: **small**. Apply the bounded fix, preserve the evidence, and obtain independent re-test before claiming D3 complete.

## Prior finding closure and controlled unblinding

Phase 1 was written and hashed before reading the closure criteria or the D3 implementation handoff. [PHASE1.json](PHASE1.json) remains unchanged. No `/tmp/hma-gate-*` directory other than this output directory was read before the freeze. The criteria directory was read only after the freeze. No prior verdict was read.

| Prior ID | Decision | Independent closure evidence |
|---|---|---|
| D3J-01 | **FIXED** | Both host-lock and core-lock wait controls passed for new consumption and replay, at exact lease expiry and in-window. Crossing preflight expiry during commit kept the reservation charged and returned false. |
| D3J-02 | **FIXED** | Sibling namespace refused before admission. Canonical namespace, dot alias, parent symlink and ledger symlink all detected used slots on replay. A dangling used-run symlink also refused replay eligibility. This closure is for namespace checking on replay; D3R2-02 covers first consumption. |
| D3J-03 | **FIXED** | A supplied repeated signature is refused across controller runs and within one unit. A distinct correction and another unit with the same signature remain allowed. D3R2-01 separately covers omission of the saved signature on the wire. |
| D3J-04 | **FIXED** | Missing/empty inventories use the controller-defined {} default. A matching nonempty inventory is admitted. A wrong digest or inventory changed after digest computation creates neither admission nor reservation. |

These closure decisions do not withdraw either new finding. D3J-03 is fixed for repeated signatures supplied to the hook. D3R2-01 concerns the separate failure to bind that field to the saved attempt. D3J-02 is fixed for replay namespace checking. D3R2-02 concerns the first-consumption branch.

The unblinded handoff does not claim D4 or D5 complete. Its older command section describes the failure guard as controller-local. Its r1 remediation section and the implementation use the unit-wide rule. The implementation controls the behavioral assessment.

## Improvement opportunity

**O-01, deferred:** make the external-controller integration job explicit. The repository endpoint tests skip when verify-loop is absent from the home directory. Both tests ran and passed here. A dedicated integration job can require that dependency and report its absence clearly. This opportunity does not affect this verdict. No implementation was added.

## Limits and deviations

- The source is a dirty working tree. The hashes and source copy bind this audit to those bytes. This is not a claim that HEAD alone reproduces D3.
- The first `sed` range for Gate A v4 accidentally included §1–§2 in addition to the requested §3–§4. These were primary design sections. No withheld implementation handoff or prior audit content was exposed by that read.
- Source and test comments contain author assertions and earlier finding IDs. The audit treated them as artifacts, not proof.
- A direct API request can contradict saved state. This audit reproduces acceptance of that request. It does not establish that the unmodified trusted controller produces it.
- The existing-ID probe demonstrates an endpoint grant. It does not demonstrate a second real worker launch. The trusted dispatcher may reject duplicate IDs independently.
- D4 launch reconciliation, `UNAUTHORIZED_DISPATCH`, revocation-minimum logic, import-generation rules, D5 criterion credit, and findings disposition remain out of scope.
- The task supplied the fresh-context and route label. This audit cannot externally attest session ancestry or model identity.
- The report validator checks JSON form and internal verdict consistency. It does not prove findings, independence, or proposed fixes.

## Single next action — architecture §18

**Return this failed D3 packet to the D3 implementation owner for bounded remediation of D3R2-01 and D3R2-02.**

Current outcome: FAIL. The new counterexamples remain open. Passing repository tests and closure of the four older findings do not remove them.

Proposed owner: the human coordinator and the existing approved D3 implementation owner. Proposed execution route: the existing human-approved D3 route. The handoff names Ming / Anthropic / `claude-opus-5-5` / Claude Code CLI. Effort and current route eligibility are not attested. Confirm the exact recorded route and required `hi` preflight before any dispatch.

Permission envelope: D3 request binding, launch eligibility, related tests and handoff corrections. This report authorizes no commit, push, release, permission widening, D4 import work, or D5 criterion work.

Exact approval: this audit grants no stage-transition approval. Apply existing human D3 remediation authorization. A change to approved route, scope, or permission needs human confirmation under architecture §17.

Stop if remediation needs an unapproved route or permission, D4/D5 scope, or a product-contract amendment. Stop if independent replay, charging, or concurrency checks fail.
