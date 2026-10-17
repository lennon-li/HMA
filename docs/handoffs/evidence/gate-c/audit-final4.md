# HMA Gate C audit

**Verdict: FAIL.** `loop-evidence` executes a shell command from an imported journal. This violates the safety kernel. The real CLI returned `EVIDENCE_RECORDED` and appended evidence.

**Isolation: compromised.** This session did not inherit implementer history. Before Phase 1, two boundary reads printed the scope note §4a heading, including a prior verdict. The section body was not read. This is a blinding breach. Do not treat this report as a fully independent blind audit. The failing command and its records are reproducible.

Audit ID: `hma-gate-c-audit-20261017-final4`. Modes: code and outcome. Date: 2026-10-17. Requested route: liz / gpt-6.1-sol / high. Serving-route telemetry was not independently attested.

Target: branch `main`, HEAD `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, plus the frozen working tree. Binary diff SHA-256: `c0e2e3dbfe2e563f319c5bf6d4ce75a081d490a7b44daf16c98674e03a46574b`. All 27 untracked-file hashes are in [freeze-start.json](freeze-start.json). [freeze-end.json](freeze-end.json) confirms that HEAD, branch, status, diff bytes and all untracked hashes stayed unchanged. `.serena/` and the two Gate D architecture amendment files were excluded from evaluation.

The accepted human scope decision is authoritative. Gate D retains leases, reservation/reconciliation authority, positive criterion credit, reproduced kernel/budget disposition, and full contract-dependent packet inputs. This audit does not call those features implemented.

## Evidence and outcome

| Check | Observed result | Evidence |
|---|---|---|
| Go 1.27.1 linux/amd64; `go test -count=1 ./...` | Exit 0 | [check-1.log](check-1.log) |
| `go vet ./...` | Exit 0 | [check-2.log](check-2.log) |
| `go test -race -count=1 ./...` | Exit 0 | [check-3.log](check-3.log) |
| `scripts/dogfood/host-dogfood.sh --keep` | Exit 0; 17 synthetic real-CLI checks | [check-4.log](check-4.log) |
| Auditor tests under `-race` | 10 top-level checks pass; shell-refusal check fails | [probes.log](probes.log), [probes-extra.log](probes-extra.log) |
| Python contract/hash/Unicode whitespace parity | Three tests pass; none skipped | [parity.log](parity.log) |
| Contract-valid synthetic verify-loop journal | Python accepts both snapshots; HMA recaptures failed check then repair, exits 1 then 0; no criterion credit | [result.json](contract-journal/result.json) |
| Real CLI shell negative probe | Exit 0; marker created; core and host records appended | [reproduction.json](shell-cli/reproduction.json) |

The full repository tests ran before auditor tests were added to the copy. Auditor tests use existing setup helpers but add separate assertions. [checks.jsonl](checks.jsonl) records commands, working directories, exit codes and durations. All execution and scratch Git writes occurred under this output directory. No worker, nested agent CLI or model API ran. Synthetic approvals are explicitly labelled as fixtures.

## Requirement coverage

Requirements were derived before reading the diff. Sources are the original objective §§5–6, the allowed Gate A sections, accepted scope §§0–4, architecture, AGENTS.md, task14 and the external interface contracts. [requirements-derived.json](requirements-derived.json) preserves the initial requirements and failure hypotheses.

| ID | In-scope requirement | Assessment | Evidence |
|---|---|---|---|
| C01 | HMA generates fixed packet, nonce and dispatch binding from exact repository artifact and current core state | **MET** | E03, E09 |
| C02 | Dispatch enforces route preflight, review stage legality and independence against all retained implementations; final reason enforces family separation | **MET** | E02, E09 |
| C03 | Receipt exact route/run/namespace/mode/completion/packet/launch/report bindings and half-open short launch validity | **MET** | E03, E05, E09 |
| C04 | Only contract-valid current pass approves; invalid/unknown reports escalate; final approval needs isolated context | **MET** | E03, E05, E09 |
| C05 | Latest packet and exact current core/live target required; single-use report/snapshot; stale/replayed/other-unit review grants nothing | **MET** | E03, E04, E09 |
| C06 | Nonapproving snapshots preserve blockers and strength; approving exact own-unit snapshot resolves only its findings; legacy records preserve ownership/waivers | **MET** | E03, E09, E11 |
| C07 | No imported citation creates unsupported BLOCK/WAIVABLE and no evidence/review import credits criteria | **MET** | E02, E03, E08, E10 |
| C08 | Existing verify-loop interface feeds HMA own no-shell exact-target recapture; journal/envelope association and repair links retained; output evidence inspectable | **PARTIAL** | E03, E06, E07, E08 |
| C09 | Closed/missing run refuses before argv; concurrency serializes terminal authority; no forged stage or broader permission | **MET** | E03, E04 |
| C10 | Migration leaves legacy history readable and enforcement visible; generated-packet runs reject new manual approval paths | **MET** | E02, E04, E09, E11 |
| C11 | Gate B approval binding, locking, crash consistency, target/stage/terminal and route invalidation guarantees do not regress | **MET** | E02, E09, E11 |
| C12 | Human-deferred rows retained with Gate D exit tests; Gate C claims no bounded execution/criterion credit/closure from missing features | **MET** | E10 |
| C13 | Fixtures exercise failure→linked repair→fresh verification, stale packets, missing report, self review and negative authority | **MET** | E02, E03, E08 |

C08 is PARTIAL because recapture, target binding, journal links and output retention work, but shell execution is permitted. C11 is MET within the tested Gate B scope. The shared executor weakness predates Gate C; the new loop adapter exposes it. Gate D positive behaviors were assessed only for preserved exit tests and absence of authority claims.

## C-F19 — Imported loop argv can execute a shell

**Severity: high. Certainty: observed. Status: OPEN. Fix: proposed. Priority: 1. Effort: medium.**

Observed facts:

- [cmd/hma/loop.go](/home/yeli/repos/HMA/cmd/hma/loop.go:231) checks only that `verify.argv` exists.
- [cmd/hma/loop.go](/home/yeli/repos/HMA/cmd/hma/loop.go:142) passes that argv into the shared evidence capture.
- [internal/evidence/capture.go](/home/yeli/repos/HMA/internal/evidence/capture.go:139) resolves and executes `/bin/sh` without rejecting shell command mode.
- The real binary accepts `argv: ["/bin/sh", "-c", "printf auditor-shell > /tmp/hma-gate-c-20261017-final4/shell-cli/shell-marker"]`.
- The marker contains `auditor-shell`. Core evidence and host `loop_evidence` records are durable under [shell-cli/store](shell-cli/store).
- The actual verify-loop contract calls the same argv a `shell command string`. See [contract-comparison.json](shell-cli/contract-comparison.json).

Diagnosis: `exec.Command` avoids an implicit shell. It still invokes a shell when the executable is `/bin/sh`. The loop adapter checks the envelope digest but does not enforce the external argv safety contract. The shared executor does not enforce it either. This violates architecture §§7.1 and 14. The proof does not require a forged receipt, stage advance, positive criterion credit, or hostile same-UID actor.

Reproduce against the retained scratch fixture:

```bash
/tmp/hma-gate-c-20261017-final4/tmp/hma-host-dogfood.qS2xed/bin/hma host loop-evidence \
  --input /tmp/hma-gate-c-20261017-final4/shell-cli/input.json \
  --store /tmp/hma-gate-c-20261017-final4/shell-cli/store
```

The first recorded setup attempt had no scratch store and refused without executing a check. The retained command log records that failure and the corrected setup. The independent Go test provides a fresh fixture on each run:

```bash
cd /tmp/hma-gate-c-20261017-final4/source
PATH="$HOME/.local/go/bin:$PATH" \
GOCACHE=/tmp/hma-gate-c-20261017-final4/gocache \
TMPDIR=/tmp/hma-gate-c-20261017-final4/tmp \
go test -race -count=1 -v -run "^TestAuditorFinal4LoopRejectsShell$" ./cmd/hma
```

Preferred fix: enforce a no-shell executable/argv policy in `internal/evidence.Capture` before execution. Apply the same check at loop admission. Resolve executable identity and refuse supported shell command/script entry points. Refuse before the process starts or either chain changes. Replace current shell-based output fixtures with explicit executable argv.

Verification: the saved shell probe must refuse before marker creation. Both chains must stay unchanged. Add absolute-path, PATH alias, symlink, combined-flag and supported Windows-shell controls. Keep positive explicit-executable checks with literal metacharacter arguments, nonzero exits, capped output and exact target binding. The existing repository checks must still pass.

Trade-off: shell-based checks will need conversion. Platform shell classification needs care. A denylist cannot prove that an arbitrary executable never starts a shell internally. Preserve the approved-command/trusted-host boundary and state this limit. Alternative containment: disable loop recapture execution until the shared guard is tested. A host allowlist of approved resolved executables is more restrictive.

## Prior finding closure

The neutral criteria file contains 30 findings. This audit assigns 22 FIXED, 1 PARTIAL and 7 DEFERRED-BY-HUMAN. The deferred label does not mean implemented. Mixed findings retain their implemented negative controls and their human-deferred positive paths.

| ID | Status | Observation and qualification |
|---|---|---|
| F01 | **FIXED** | Snapshot strength tests cover BLOCK/WAIVABLE omissions and same-ID downgrade attempts under REQUEST_CHANGES and ESCALATE; approving control resolves exact own unit. |
| F02 | **FIXED** | Core head is checked at both imports. Auditor finding/criterion mutations and live tracked/untracked changes refuse without changing either chain; baseline route/waiver invalidation suites also pass. |
| F03 | **FIXED** | Two concurrent imports produce exactly one success. Newer unanswered packets invalidate the old approval. Existing replay-after-failure tests also pass. |
| F04 | **FIXED** | Packet derives artifact from Snapshot; receipt root/launch fields and live target are checked. Wrong artifact/root and dirty-target controls in repository tests pass. |
| F05 | **FIXED** | Final issuing reason activates both provider-family and model-family separation independently of final_validation. Existing same-family controls pass. |
| F06 | **FIXED** | Auditor wrong and missing provider, effort, model and agent cases refuse; exact-route control passes. |
| F07 | **FIXED** | Auditor valid control and missing target/requirements/evidence/findings, failed requirement, unfrozen and partial cases agree with Python and cannot clear BLOCK or open gate. Existing malformed/trailing/high-finding fixtures pass. |
| F08 | **DEFERRED-BY-HUMAN** | Unsupported citations cannot create BLOCK/WAIVABLE. Positive reproduced-kernel and measured-budget paths are explicitly deferred by the accepted boundary. |
| F09 | **PARTIAL** | The evidence bridge exists, retains journal/attempt links and streams, re-captures red then repaired checks and credits no criterion. C-F19 leaves its required no-shell guarantee open. |
| F10 | **FIXED** | Auditor tests before issue, at issue, just before expiry, exactly at expiry and after expiry. Only [from,until) passes. Stale core/route controls cover rewind rejection. |
| F11 | **FIXED** | implementationContributions scans all implementation dispatches and direct decisions. Full suite multi-route independence control passes. |
| F12 | **DEFERRED-BY-HUMAN** | Accepted §0.1 retains reservation, namespace, replay, half-open expiry and generation tests at Gate D; no Gate C positive authority claim is accepted. |
| F13 | **FIXED** | Once packet mode begins, dispatch/manual-review entry points cannot supply imported provenance or override failures. Legacy history remains readable and mode is labelled. |
| A01 | **FIXED** | Same evidence as F07. Invalid reports cannot clear the seeded blocker or open Verification; valid report plus snapshot does. |
| A02 | **FIXED** | Both imports re-snapshot the recorded target. Independent tracked/untracked drift checks refuse, preserve core findings and leave chains unchanged. |
| A03 | **FIXED** | SnapshotFindings uses UnitID, not string prefixes. Slash-containing unit and finding identifiers preserve other-unit blockers in auditor assertions. |
| A04 | **DEFERRED-BY-HUMAN** | Current packet contains bound core head, plan digest, criteria projection, findings and waivers; exact core commit check prevents a stale assembled packet append. Full contract-dependent immutable inputs remain human-deferred. |
| A05 | **FIXED** | Legacy omitted ownership is assigned conservatively; unowned findings stay in force. Auditor other-unit and active-waiver projection checks pass; authentic approving own-unit legacy resolution passes in full suite. |
| A06 | **DEFERRED-BY-HUMAN** | Positive disposition reproduction, plan-budget checks and independent criterion credit require Gate D data and are human-deferred. Negative no-credit and unsupported-citation controls pass. |
| A07 | **DEFERRED-BY-HUMAN** | Auditor/full suite wrong envelope/store checks fail closed; valid recapture succeeds. Namespace/reservation/generation reconciliation is human-deferred. This status does not claim that launch authority exists. |
| A08 | **FIXED** | Closed and missing runs refuse before marker creation. An open capture holds the host lock; terminalization following the CLI lock order waits until capture finishes, then further captures refuse. Race probe passes without deadlock. |
| A09 | **FIXED** | legacy_manual and packet_import are exposed by show --mode and normal host write results. Migration rejects new manual review approval. |
| R3-01 | **FIXED** | Auditor newer pending packet invalidates old snapshot/gate; completing the latest exact pass and snapshot reopens only the host gate. Concurrent double-import is tested under -race. |
| R3-02 | **DEFERRED-BY-HUMAN** | Full request, criterion definitions, non-goals, budgets, required-evidence and policy bytes are explicitly human-deferred. Available packet inputs are core-head bound and current findings are present. |
| R3-03 | **DEFERRED-BY-HUMAN** | Human scope decision is authoritative. Accepted §0.1 preserves the deferred rows and exit criteria. No reservation, criterion-credit or closure success is inferred. |
| C-F14 | **FIXED** | Auditor caller Packet injection is refused without append. The real HMA-generated packet/import/snapshot control passes; legacy packet/manual review cases in repository suite remain distinct. |
| C-F15 | **FIXED** | Auditor recapture retains distinct stdout/stderr, exit 7 and output above cap. The retained bytes reproduce the framed digest; truncation is explicit. Mutating-check negative control passes. |
| C-F16 | **FIXED** | UnmarshalExact discards case/fold variants before struct decoding. Auditor verdict/isolation/binding and route-field cases grant no authority; full Unicode receipt tests pass. |
| C-F17 | **FIXED** | U+001C..U+001F invalid passes preserve blockers and close gate. Explicit full-code-point Python/Go whitespace parity executes and passes. |
| C-F18 | **FIXED** | Auditor empty, partial and symlink artifact residue refuses and remains unchanged; intact output artifact is reused. Temp-write/link implementation prevents partial bytes under the final name. |

## Evidence ledger

| ID | Source | Observation |
|---|---|---|
| E01 | freeze-start.json | main at 527b7c5ff79090c558f5bbfed5ec87a85f06d1e5; binary diff SHA-256 c0e2e3dbfe2e563f319c5bf6d4ce75a081d490a7b44daf16c98674e03a46574b; hashes of 27 untracked files. |
| E02 | checks.jsonl; check-0.log through check-4.log | Go 1.27.1 linux/amd64; go test ./..., go vet ./..., go test -race ./... exit 0; synthetic real-CLI host dogfood passes 17 checks. |
| E03 | probes.log; source/cmd/hma/auditor_final4_test.go | Auditor matrix checks report/Python contract parity, Unicode keys, route fields, exact snapshots, concurrent replay, newer unanswered packet, target/core drift, output/truncation, legacy ownership, and half-open validity. Eight top-level checks pass. Shell refusal check fails. |
| E04 | probes-extra.log | Auditor caller-supplied packet, corrupt output residue, intact artifact, missing/closed run, and terminal lock race checks pass under -race. Two top-level tests pass. |
| E05 | parity.log; internal/host/gatec_r3_parity_test.go | Canonical hash, external report-validator parity, and Python whitespace predicate parity over all Unicode code points pass. These three tests executed without skips. |
| E06 | shell-cli/reproduction.json; shell-cli/input.json; shell-cli/state.json; shell-cli/store/ | Real CLI accepts /bin/sh -c, exits 0, creates shell-marker containing auditor-shell, and appends core evidence plus host loop_evidence. The initial setup attempt failed because the scratch store did not exist; it ran no check. Creating that scratch directory allowed the same CLI probe to reproduce the defect. All attempts remain logged. |
| E07 | shell-cli/contract-comparison.json; /home/yeli/obsidian/AgentSystem/scripts/verify-loop:318 | verify-loop reject_unsafe_argv identifies the exact argv accepted by HMA as shell command string. |
| E08 | contract-journal.log; contract-journal/result.json; contract_journal_probe.py | Both synthetic journal snapshots pass the actual verify-loop load_state contract. A failed check and linked repair recapture with exits 1 then 0. Full journals, reports, command logs and HMA records remain available. No criterion credit, worker dispatch or lease authority is created. |
| E09 | cmd/hma/review.go:157; internal/host/review_import.go:125; internal/host/findings_import.go:79; cmd/hma/transition.go:323 | Static review confirms packet generation, exact receipt/report binding, live target/core freshness, current-dispatch checks, own-unit ownership, snapshot provenance, and manual/import separation. |
| E10 | docs/handoffs/2026-10-16-gate-c-scope.md:16; docs/task14-host-orchestration.md:133 | Accepted §0.1 lists Gate D exit tests and forbids bounded-execution, positive-credit or closure reliance. Task14 labels launch reconciliation Gate D and current imported findings advisory only. |
| E11 | internal/model/types.go:148; internal/model/validate.go:283; internal/engine/resolution.go:47; internal/host/findings_import.go:113 | New optional core fields omit zero values. Legacy unowned findings derive ownership from the last non-snapshot record or stay unowned. Snapshot application leaves waiver projection intact. Full baseline compatibility and Gate B suites pass. |
| E12 | auditor session disclosure | No implementer conversation inherited. The scope section extractor and a later sed boundary read printed the §4a heading, including Liz FAIL, before Phase 1. The body and later sections were not read. This is prohibited prior-verdict leakage; isolation_compromised=true. No restart was made. An earlier heading inventory also listed excluded Gate A section headings, without their bodies. |

## Phase 1 and controlled unblinding

Phase 1 is immutable at [PHASE1.json](PHASE1.json). SHA-256: `0e760038ecde5991f8673f0d9de4fbbbb31911d32d8a7f1fd56b88bdf7103ac2`. The external report validator accepted its form. After that freeze, the auditor read only the Gate C section of TODO.md as optional unblinding. It records the accepted boundary and leaves the audit/commit unchecked. No requirement, finding or prior-finding status changed. No prior audit directory or scope §4a body was read.

The final JSON uses contract verdict `fix-required`, which maps to the user verdict **FAIL**. These express one decision. Structural validation does not establish the truth of findings.

## Limits

- Isolation is compromised by prohibited scope §4a heading/prior-verdict exposure before Phase 1. No body of §4a or later and no prior report directory was read before freeze. Do not describe this as a fully independent blind audit.
- This is one auditor session using the route requested by the dispatch (liz / gpt-6.1-sol / high). The session cannot independently attest its serving provider/model/effort telemetry. No worker, nested agent CLI or model API was invoked.
- Tests ran on a filesystem copy under this output directory. Production source bytes were copied unchanged; TODO.md, .serena/ and Gate D amendment inputs were omitted. Auditor tests and scratch Git fixtures exist only under the output directory. Repository access remained read-only.
- Existing test helpers supply synthetic setup; auditor-authored assertions and the standalone real-CLI probe supply independent behavior checks. Prior named auditor source tests were not imported from other audit directories.
- Fixtures use synthetic approvals explicitly labelled not human approvals. They prove command behavior, not authentic human authorization or provider identity. The host dogfood is synthetic and launches no worker.
- The contract-valid verify-loop journal is synthetic and starts no worker. Gate D lease/admission/reservation/namespace authority and positive criterion credit are not implemented acceptance claims here.
- Tests are adversarial samples, not proof of absence. No crash injection into the new output writer was performed; residue tests and static temp/sync/link inspection cover that writer. Existing crash/restart suites were executed.
- No callable Jev tool was exposed. The auditor used the allowed orchestrator/human fallback. No advisory model is represented as Jev.
- Some exploratory tool output was truncated. Mandatory report-contract, key source paths and relevant architecture sections were reread in bounded outputs. Raw test outputs are retained untruncated in log files.

No additional redesign opportunity is proposed. The smallest required work is the no-shell repair and a fresh audit. Raw logs, fixtures, checksums, source references and the full remediation contract are retained in [AUDIT.json](AUDIT.json) and [evidence-digests.json](evidence-digests.json).

## Required next action (§18)

**REMEDIATE_C_F19.** The HMA implementation owner must obtain human authorization for one bounded no-shell repair unit and its exact eligible route. The permission envelope is source/test changes for this defect, with no commit, push, publish, release or external effect authorized by this audit. Use the existing approved implementation route; this report does not select a new route. Stop and escalate if the fix needs broader permissions or contract weakening. A fresh-context audit must verify the exact repaired tree before Gate C acceptance.
