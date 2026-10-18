# Gate D D3 audit

**Verdict: FAIL.** Three observed high-severity findings and one observed medium-severity finding remain open. The hook can return launch eligibility after expiry, miss a used dispatch under an unchecked namespace, and bypass the unit failure guard through a new controller. It also accepts a forged clone-baseline digest.

The corrected full `go test ./...`, `go vet ./...`, and `go test -race ./...` checks passed. All 12 packages passed normal and race runs. The auditor overlay ran 12 top-level tests with `-race`: 8 passed and 4 failed on the defects below. No data-race warning appeared. Green repository tests do not resolve those failures.

Target: `main@0d90eed831741db297c220ee83f123e2554e06a0` plus the frozen working tree. Binary diff SHA-256: `70b827844a42b0d90e6d0a7a6984f8ef73e2798e8f7de69d12a6d2c59828e072`. The exact untracked-file digests are in [FREEZE.json](FREEZE.json). The repository and external snapshot were unchanged at the end.

Context isolation: fresh audit context; requirements and failure hypotheses were recorded before the diff. The D3 handoff remained blinded until Phase 1 was frozen. Tests ran from an external copy on the same host. This is not a hermetic execution environment. The supplied jax / gpt-6.1-sol / high route is dispatch metadata, not independent model telemetry. No worker or nested model API was launched.

Phase 1 SHA-256: `285ab2e12639fb51e584461007fbcfa5ac55ae98aedf3931f52180b92bddb751`. [PHASE1.json](PHASE1.json) is unchanged. [ADDENDUM.md](ADDENDUM.md) records the later scope-claim check. The machine report uses the skill-contract value `fix-required` for the same user-facing decision, `FAIL`.

## Requirement coverage

| ID | D3 status | Observation and evidence |
|---|---|---|
| D3-01 | MET | Replay precedes admission, lease validity and capacity checks; identical key/digest preserves reservation and charge; changed digest conflicts. **Observed:** Stable key and digest replay one reservation. Exhausted capacity does not block replay. Altered valid requests conflict. Evidence: `E-TEST`, `independent-complete.log:118`, `cmd/hma/attempt.go:216` |
| D3-02 | PARTIAL | First controller admission binds envelope and controller state; active lease, exact route, stop escalation and envelope ceilings <= remaining attempts, minutes and spend. Admission consumes nothing. **Observed:** Attempt, time and spend admission ceilings reject without records. Admission consumes nothing. Current-time and baseline/namespace binding have reproduced gaps. Evidence: `independent-complete.log:110`, `D3J-01`, `D3J-02`, `D3J-04` |
| D3-03 | PARTIAL | Each new attempt requires matching admission, active lease and >=1 capacity; one durable reservation charge; no rechecking full envelope ceiling after admission. **Observed:** One append charges consumption. Later attempts do not recheck the full envelope. A caller delayed on a lock can consume after lease expiry. Evidence: `independent-complete.log:68`, `D3J-01` |
| D3-04 | MET | Unknown/unlaunched reservations remain charged; totals survive rewind and lease renewal. **Observed:** Unknown or unused reservations remain charged. Counters sum every unit reservation across lease generations. Rewind/renewal tests retain spent capacity. Evidence: `E-TEST`, `independent-complete.log:118`, `cmd/hma/gated_attempt_test.go:276` |
| D3-05 | PARTIAL | A repeated material failure signature twice stops further attempts; route cannot change; non-stop escalation is refused. **Observed:** Wrong worker/model/effort and non-stop escalation are refused. Same-controller repeated signature is refused. New controller bypasses the unit failure guard. Evidence: `E-TEST`, `D3J-03` |
| D3-06 | PARTIAL | Last-slot replay remains eligible only for current active lease and unused dispatch ID; used/expired/revoked/old-generation reservation remains counted and ineligible. Capacity is not rechecked on replay. **Observed:** Normal final-slot replay, used-slot, exact-expiry and old-generation cases pass. Delayed-time and wrong-namespace cases can return true when they should not. Evidence: `independent-complete.log:88`, `independent-complete.log:118`, `D3J-01`, `D3J-02` |
| D3-07 | MET | Atomic per-run serialization prevents two callers from sharing last capacity or duplicating charge on same key. **Observed:** Two distinct controllers competing for one remaining attempt produce one winner. Eight identical concurrent callers share one reservation and one charge. No data-race warning appeared. Evidence: `independent-complete.log:73`, `independent-complete.log:78`, `E-RACE` |
| D3-08 | PARTIAL | Hook request is bound to controller state, envelope, store, HMA run/unit, live root, baseline and namespace; conflicts refuse. Exact route includes profile/provider/model/effort/access. **Observed:** Controller ID, envelope digest, store, HMA run/unit and root mismatches reject without charge. Baseline digest and namespace do not bind to controller state as required. Evidence: `independent-complete.log:57`, `D3J-02`, `D3J-04` |
| D3-09 | MET | D5 endpoint tests prove launch requires launch_eligible true and resume preserves dispatch ID through the real HMA endpoint. **Observed:** The existing D5 tests passed against a freshly built HMA binary. An independent adapter reproduction used actual authority_request, authorize_launch and invoke_authority_hook; resume reused the reservation and a used slot was refused. Evidence: `E-TEST`, `E-D5`, `independent-complete.log:95` |
| D3-10 | MET | No D1/D2, Gate B or Gate C regressions; D3 claims no D4 reconciliation/import generation/valid_until or D5 criterion/findings closure. **Observed:** All 12 packages pass normal and race suites. Vet passes. D1/D2, Gate B and Gate C regression suites are included. After Phase 1 freeze, handoff inspection found explicit D4/D5 deferrals and no out-of-scope closure claim. Evidence: `E-TEST`, `E-VET`, `E-RACE`, `docs/handoffs/2026-10-17-gate-d-d3.md:8`, `END_FREEZE.json` |

The requirements come from the original Gate D objective, the selected Gate A design sections, Gate C scope §0.1 rows 1, 2 and 5, architecture §§14/17/18, and the three verify-loop hook functions. [SOURCES.json](SOURCES.json) retains the exact scoped excerpts and source digests. The D4 and later D5 duties receive no closure credit.

## Atomic requirement checks

| ID | Requirement | Status | Evidence |
|---|---|---|---|
| C01 | Replay first, including exhausted capacity | MET | `independent-complete.log:118` |
| C02 | Same key, changed valid request digest returns HOST_RESERVATION_CONFLICT | MET | `cmd/hma/gated_attempt_test.go:208; go-test-final.log` |
| C03 | First admission consumes no attempt | MET | `audit_d3_test.go::TestAuditD3ClockMustBeReadInsideLocks; cmd/hma/attempt.go:269` |
| C04 | Admission caps attempts, minutes and spend to remaining lease amounts | MET | `independent-complete.log:110` |
| C05 | Authority checks use current time after lock wait | NOT MET | `D3J-01; independent-complete.log:30` |
| C06 | Consumption requires current generation admission and counts once | MET | `cmd/hma/gated_attempt_test.go:276; independent-complete.log:78` |
| C07 | Consumption does not recheck the full original envelope against decreasing remaining capacity | MET | `independent-complete.log:68` |
| C08 | Unknown/unlaunched reservation remains charged | MET | `independent-complete.log:118` |
| C09 | Counters carry across rewind and renewal | MET | `cmd/hma/gated_attempt_test.go:276; go-test-final.log` |
| C10 | Repeated material failure cannot be bypassed by a new controller for the same unit | NOT MET | `D3J-03; independent-complete.log:24` |
| C11 | Different worker/model/effort route is refused and approved preflight route grant is checked | MET | `cmd/hma/gated_attempt_test.go:304; go-test-final.log` |
| C12 | Envelope escalate_to must be stop | MET | `cmd/hma/gated_attempt_test.go:326; go-test-final.log` |
| C13 | Final-slot lost-response replay uses same reservation without another charge | MET | `independent-complete.log:118` |
| C14 | Used slot is ineligible under the validated declared namespace | PARTIAL | `independent-complete.log:118; D3J-02` |
| C15 | Expired, stage-revoked or old-generation slot is ineligible | PARTIAL | `independent-complete.log:88; cmd/hma/gated_attempt_test.go:373; D3J-01` |
| C16 | Two distinct controllers cannot both consume the last slot | MET | `independent-complete.log:73` |
| C17 | Concurrent identical requests charge exactly once | MET | `independent-complete.log:78` |
| C18 | Request binds controller ID, envelope digest, store, HMA run, unit and live root | MET | `independent-complete.log:57` |
| C19 | Clone-baseline digest matches controller baseline inventory | NOT MET | `D3J-04; independent-complete.log:10` |
| C20 | Ledger namespace matches the controller-defined namespace | NOT MET | `D3J-02; independent-complete.log:18` |
| C21 | Real D5 endpoint requires eligibility true and preserves dispatch identity on resume | MET | `independent-complete.log:95; independent-d5.log` |
| C22 | D1/D2, Gate B and Gate C regression suites pass | MET | `go-test-final.log; go-vet.log; go-race-final.log` |
| C23 | No D4 or later D5 completion claim | MET | `docs/handoffs/2026-10-17-gate-d-d3.md:8; ADDENDUM.md` |

## Findings

### D3J-01 — HIGH: Lease time is sampled before waiting for both run locks

**Observed.** The hook returned launch_eligible=true with a trusted test clock at 02:32:05Z and valid_until at 02:31:05Z. The test held the real host lock, started consumption, observed the pre-lock clock read, advanced the injected clock beyond expiry, and released the lock. This is a deterministic simulated clock advance, not a one-hour real-time wait.

**Cause.** attemptOnce captures now before host and core lock acquisition. Lock wait can carry that stale value past the lease and preflight expiry. Consumption and replay use the stale value for authority and record time.

**Evidence.** cmd/hma/attempt.go:202; cmd/hma/attempt.go:204; cmd/hma/attempt.go:205; cmd/hma/attempt.go:306; independent-complete.log:30; audit_d3_test.go::TestAuditD3ClockMustBeReadInsideLocks.

**Proposed fix.** Read the host clock after both locks are acquired. Refresh launch eligibility before returning an affirmative response after a durable commit; preserve the charge if the lease expires during commit.

**Verification.** Run the clock-wait reproduction for new consumption and replay, and a control whose clock remains within validity. Expected: Expired callers never receive launch_eligible=true. A committed reservation remains counted. In-window control remains eligible.

**Alternative.** Use a bounded lock acquisition deadline tied to lease expiry, plus a final current-time eligibility check.

**Trade-off.** A successful reservation can become ineligible before response. This is conservative and must not refund or cancel it. Effort: small. Priority: P1. Fix status: proposed; no repair was applied or validated.

### D3J-02 — HIGH: An unchecked namespace lets replay miss an existing dispatch

**Observed.** A request used <controller>/wrong-ledger while the real layout is <controller>/dispatch-ledger. Admission succeeded. After creating the dispatch under the real namespace/runs/ID, replay still returned launch_eligible=true with one charged reservation.

**Cause.** parseAttemptRequest selects state.json from the namespace parent but never verifies the namespace child is the controller-defined dispatch-ledger. Admission freezes the unchecked string. replayResult then tests that unchecked path for prior use.

**Evidence.** cmd/hma/attempt.go:172; cmd/hma/attempt.go:186; cmd/hma/attempt.go:281; cmd/hma/attempt.go:334; independent-complete.log:18; audit_d3_test.go::TestAuditD3NamespaceMustMatchControllerLayout.

**Proposed fix.** Derive the canonical ledger namespace from the loaded controller-state directory and require the request namespace to match it. Persist and use the derived namespace for admission and replay.

**Verification.** Run the wrong-namespace reproduction, a canonical namespace control, a used-slot control, and symlink/path-alias cases under the declared path policy. Expected: Wrong namespace is refused before authority records are appended. A used dispatch in the derived namespace is ineligible.

**Alternative.** Accept an explicitly supplied controller state path through trusted hook configuration and derive the namespace from that path.

**Trade-off.** Keep valid path aliases only if the host defines their canonical equivalence. This finding does not assert D4 reconciliation is implemented or required here. Effort: small. Priority: P1. Fix status: proposed; no repair was applied or validated.

### D3J-03 — HIGH: A new controller run resets the repeated-failure guard

**Observed.** One unit obtained an initial reservation, then a reservation with material-failure-A. Another request with that signature in the same controller was refused. A new controller for the same unit presented material-failure-A and received launch_eligible=true, raising reservations to three.

**Cause.** The signature lookup filters by ControllerRunID. Architecture §17 and the approved design apply the material-failure stopping rule to the plan unit. Starting another controller changes no unit failure evidence or approval.

**Evidence.** cmd/hma/attempt.go:289; cmd/hma/attempt.go:291; independent-complete.log:24; audit_d3_test.go::TestAuditD3FailureGuardSurvivesNewController.

**Proposed fix.** Apply the repeated-material-failure guard to the unit across controller runs, using authoritative failure evidence and the approved reset/renewal policy. Do not treat a fresh controller ID as a reset.

**Verification.** Run the cross-controller reproduction, same-controller refusal, a different-signature control, and another-unit control. Expected: The same unit cannot obtain another reservation for the repeated signature. Unrelated units and a permitted distinct correction are not conflated.

**Alternative.** Persist a per-unit failure-evidence summary and consult it under the same locks, rather than scanning reservations.

**Trade-off.** Define any human-approved reset explicitly. Preserve monotonic counters and avoid inventing an automatic reset on a new lease. Effort: small. Priority: P1. Fix status: proposed; no repair was applied or validated.

### D3J-04 — MEDIUM: The clone-baseline digest is copied without checking controller state

**Observed.** The control populated repo.workspace_baseline_inventory and supplied its Python-canonical SHA-256; it passed. Replacing only clone_baseline_digest with forged-baseline also returned launch_eligible=true and appended one reservation.

**Cause.** loopState reads only repo.root. parseAttemptRequest never computes the digest of repo.workspace_baseline_inventory. Admission therefore binds the caller assertion to itself, rather than binding it to the controller clone baseline.

**Evidence.** cmd/hma/attempt.go:177; cmd/hma/attempt.go:186; cmd/hma/attempt.go:271; cmd/hma/loop.go:37; independent-complete.log:10; audit_d3_test.go::TestAuditD3BaselineMustMatchState.

**Proposed fix.** Decode the controller baseline inventory and compare its canonical digest with clone_baseline_digest, using the exact authority_request default and canonicalization. Refuse mismatch before admission.

**Verification.** Run the matching/mismatching baseline controls, an empty/missing baseline case with the controller-defined default, and a changed-inventory case. Expected: Only a digest computed from the loaded controller state is admitted; mismatch appends no admission or reservation.

**Alternative.** Move association derivation into a trusted controller-state loader and accept only its derived values.

**Trade-off.** This is an input-binding defect under the trusted-host model. It does not show a correct unmodified verify-loop emitting a forged baseline. Effort: small. Priority: P2. Fix status: proposed; no repair was applied or validated.

## Evidence and reproduction

Use the unchanged source snapshot and the auditor overlay under this output directory. The original repository is not a test working directory. [COMMANDS.json](COMMANDS.json) retains executable argument vectors, environment, working directory, exit codes, raw-log paths and the external adapter digest. [SNAPSHOT.json](SNAPSHOT.json) binds the copied source bytes. [END_FREEZE.json](END_FREEZE.json) records the final integrity comparison.

```bash
cd /tmp/hma-gate-d-20261018-d3j/snapshot
export PATH=/home/yeli/.local/go/bin:/usr/bin:/bin
export GOCACHE=/tmp/hma-gate-d-20261018-d3j/gocache
export TMPDIR=/tmp/hma-gate-d-20261018-d3j/runtime
export GOTMPDIR=/tmp/hma-gate-d-20261018-d3j/runtime
export GOFLAGS=-buildvcs=false
export PYTHONDONTWRITEBYTECODE=1
/home/yeli/.local/go/bin/go test ./...
/home/yeli/.local/go/bin/go vet ./...
/home/yeli/.local/go/bin/go test -race ./...
/home/yeli/.local/go/bin/go test -overlay=/tmp/hma-gate-d-20261018-d3j/overlay.json -race ./cmd/hma -run '^TestAuditD3' -count=1 -v
```

The last command must fail on this frozen source. The four failing top-level tests are `TestAuditD3BaselineMustMatchState`, `TestAuditD3NamespaceMustMatchControllerLayout`, `TestAuditD3FailureGuardSurvivesNewController`, and `TestAuditD3ClockMustBeReadInsideLocks`. The remaining eight top-level tests pass.

Raw evidence: [normal suite](go-test-final.log), [vet](go-vet.log), [race suite](go-race-final.log), [independent checks](independent-complete.log), [real D5 adapter](independent-d5.log), [auditor tests](audit_d3_test.go), and [overlay map](overlay.json). Test logs combine complete stdout and stderr. Display compression did not truncate the saved files.

The real D5 reproduction called `authority_request`, `authorize_launch`, and `invoke_authority_hook` from `/home/yeli/obsidian/AgentSystem/scripts/verify-loop`. It used a freshly built HMA binary, persisted controller state, reloaded that state for resume, preserved the dispatch ID and reservation, and refused a used slot. It launched no model worker.

## Limits and assumptions

- Context isolation: fresh assigned audit context, original requirements before diff, no implementer handoff read before Phase 1 freeze. Source comments and fixture helpers are naturally author-written.
- Execution isolation: external source snapshot and overlays on the same trusted host. This is not a hermetic container or independent host. Tests used scratch repositories outside /home/yeli/repos.
- The dispatch names jax / gpt-6.1-sol / high / review. Model, provider and effort telemetry are user-supplied route metadata, not independently attested by this audit. No worker or nested model API was dispatched.
- Fixture construction helpers are reused from the candidate tests. Acceptance assertions, malformed inputs, concurrent-controller tests and clock-wait reproduction are auditor-authored.
- The clock-wait finding uses an injected trusted clock and real host lock. It does not reproduce a one-hour wall-clock stall.
- D4 launch reconciliation, UNAUTHORIZED_DISPATCH, revocation minima and import-generation rules are out of scope. D5 criterion credit and findings disposition are out of scope. No closure credit is awarded for them.
- The original external Python boot tests were not run as an external suite. Their two D5 endpoint obligations were exercised by existing Go endpoint tests and an independent real-adapter reproduction.
- No power-loss campaign or filesystem fault injection was performed. Durability depends on the existing audited transactional store, source inspection and regression tests. No universal absence-of-bugs claim is made.
- Three preliminary test runs had auditor snapshot/environment errors. Their logs are retained and explicitly superseded by the corrected successful full runs.
- Shell tool output was sometimes display-truncated or compressed. Complete test stdout/stderr remains in raw log files. No raw test log is truncated.
- No approved Jev host tool was available. Requirement classification, sufficiency and next-step decisions used the auditor fallback. Jev was not simulated.

Assumption: the host and controller-state store are trusted, as requested. The namespace and baseline findings concern missing consistency checks; they do not claim hostile-host resistance. The namespace reproduction supplies a malformed request. The baseline reproduction changes only the asserted digest.

No optional redesign is recommended. Fix the four bounded defects. Then obtain independent review of the repaired snapshot. The present audit supplies no acceptance credit for later reconciliation, criterion-credit, findings-disposition, full-packet or final-closure work.

The untracked root `hma` binary was hashed but not executed. Endpoint tests used a fresh build from the frozen source. Exact per-command start/end times were not instrumented; log modification times are available but are not precise command timings.

## Required next action — architecture §18

Prepare one bounded D3 remediation dispatch packet for D3J-01 through D3J-04.

**Outcome and rationale:** FAIL; the observed authority and binding defects prevent D3 acceptance. **Owner:** human-gated interactive orchestrator. **Route:** packet preparation requires no worker dispatch. A future repair must use the exact approved implementer route and pass its `hi` preflight. **Permission envelope:** prepare the packet only; no repository edits, git writes, launches or stage advance are authorized by this audit. **Exact approval required:** none for packet preparation; later coding needs the applicable challenge-bound human approval. **Stop and escalation:** keep D3 unaccepted; stop if route, approval binding or scope cannot be established; require fresh independent review after repair.
