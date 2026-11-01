# Gate E checkpoint-trigger draft — independent critique

**Gate verdict: FAIL — revise before presenting this draft as a decision packet.**

**Scope and isolation:** I read the full draft and checked its central claims against `internal/`,
`cmd/hma/`, and the merged architecture. I made no code, architecture, or draft changes. This report
is the only file created for the task. The repository is `main` at `240119d`; the draft is an
untracked file, so the worktree is not clean as the draft states. No worker was used. The required
shared-memory startup documents were not accessible in this non-interactive session. No Jev
decision tool was exposed.

**Evidence:** Read the draft, `docs/handoffs/evidence/gate-a/audit-r7.md`, the Gate A handoff,
`docs/architecture.md`, `internal/host/host.go`, `internal/host/review_import.go`,
`internal/engine/route_coherence.go`, `internal/engine/contract_binding.go`,
`internal/model/contract.go`, and the cited `cmd/hma/` paths. Ran `git status --short --branch`,
`git rev-parse --short HEAD`, targeted `rg` searches, and line-numbered source reads. The targeted
test command `go test ./cmd/hma -run 'TestScorecardOnlyIndependentCredit|TestGateCFinalNeedsIsolatedReport' -count=1`
could not run because `go` is not installed (`command not found`). No test result is claimed.

## Confirmed-accurate claims

- **Three accepted issuing reasons; host-selected.** `IssuingReason` declares `per_change`,
  `checkpoint`, and `final`; `validIssuingReason` accepts only those values
  (`internal/host/host.go:195-202,459-475`). The CLI receives the host's reason and copies it into
  the packet binding (`cmd/hma/review.go:24-31,373-379`). My repository search found no derivation
  from a diff, contract, plan, or stage.
- **`checkpoint` currently has no checkpoint effect.** The only use of `IssueCheckpoint` is in the
  enum-membership predicate (`internal/host/host.go:459-461`). `final` does change dispatch
  independence (`:980-991`) and requires an `APPROVE` report to be isolated
  (`internal/host/review_import.go:230-232`).
- **Risk is host-declared at review dispatch.** Validation checks that `Dispatch.Risk` is a valid
  enum (`internal/host/host.go:651-681`); independence then uses that field
  (`:980-991`). I found no comparison with the bound contract's `Risk`. Route coherence is a
  separate evaluator over a caller-supplied risk (`internal/engine/route_coherence.go:30-45,227-305`),
  exposed by read-only `hma eval route-coherence` (`cmd/hma/eval.go:77-83`); it is not called by
  review-packet issuance (`cmd/hma/review.go:302-458`).
- **The current enforcement path lacks a checkpoint/final gate.** Packet manifests are created only
  when a contract is present (`cmd/hma/review.go:399-415`). The host transition gate returns without
  checks for edges other than two implementation edges (`cmd/hma/transition.go:37-42`). Verified
  closure checks fresh artifact, findings, waivers, and criterion credit
  (`:304-327`); criterion credit checks the latest imported review (`cmd/hma/credit.go:464-520`),
  not its issuing reason or family separation.

## Disputed or incomplete claims

- **G-F is a real current-spec enforcement gap, with one qualification.** A passing criterion can
  be credited and the outcome gate can accept that credit using a fixture whose reason is
  `IssuePerChange` (`cmd/hma/gated_credit_test.go:51,184-203`). That fixture sets review risk high,
  so it proves the missing final-level requirement, not the same-family case. The latter follows
  from the dispatch condition: only high/critical risk, `Disputed`, `FinalValidation`, or
  `IssueFinal` activate family separation (`internal/host/host.go:980-991`); when the host sets
  normal risk, the remaining checks require a distinct worker, model, and context, but not a
  distinct family (`:984-989`). Thus a same-family, normal-risk per-change review can support
  verified closure without satisfying §16's mandatory final-outcome checkpoint
  (`docs/architecture.md:656-658`). The report's broader wording should distinguish this concrete
  path from high-risk dispatches, where a correctly supplied high risk already forces family
  separation. The implementation still does not enforce that the closure review is a final audit.
- **G-R is real; no route-coherence path closes it.** A low `Dispatch.Risk` passes the enum check and
  avoids the family-separation branch even if the contract says high
  (`internal/host/host.go:651-681,980-991`; `internal/model/contract.go:54-67,413-415`).
  Architecture §13 requires an HMA risk floor and family separation for high-risk and final
  validation (`docs/architecture.md:557-572`). The route-coherence evaluator cannot supply hidden
  enforcement: it consumes its own `req.Risk` and does not read the acceptance contract
  (`internal/engine/route_coherence.go:30-45,227-305`).
- **G-V is stated too broadly.** It is true that there is no general host gate on leaving
  `INDEPENDENT_VALIDATION` (`cmd/hma/transition.go:37-42`). That alone is not a defect for every
  such transition. The current §16 rule requires a checkpoint when one of its listed triggers fires
  (`docs/architecture.md:656-658`). State G-V as the missing enforcement for trigger-positive
  transitions, not as a requirement to review every IV exit.
- **T7 does not implement the stated trigger.** The current §16 says “final outcome closure,” not
  only successful closure (`docs/architecture.md:656-658`), and the architecture defines
  `FAILED`, `BLOCKED`, `UNKNOWN`, `ABORTED`, and `PARTIAL` outcomes (`docs/architecture.md:118-134,159`).
  The proposed classifier fires T7 only for `VERIFIED_SUCCESS`, `VERIFIED_WITH_WAIVERS`, or stage
  `RELEASE_AND_CLOSURE`
  (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:255-264`). A non-success terminal transition from another stage
  can therefore miss T7; E3 also attaches only to the verified-success closure branch
  (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:316-321`;
  `cmd/hma/transition.go:304-327`). The draft must ask Lennon whether all terminal
  outcomes require the final audit or whether the existing spec should be narrowed.
- **Path-only classification can miss a mandatory trigger.** The design says an interface change in
  an `ordinary` path is invisible
  (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:453-455`), yet recommends that path-only option. That
  cannot guarantee §16's mandatory trigger for an interface that the path policy incorrectly calls
  ordinary or cannot semantically classify (`docs/architecture.md:656-658`). This is not only a
  policy-quality caveat: it is an unresolved choice between changing the classifier and explicitly
  amending the guarantee. Do not present the recommendation as satisfying current §16.
- **T4 and `RiskFloor` do not align.** T4 fires when a unit merely lacks `no_external_effect`, as
  well as when a path is classified as external or irreversible; `RiskFloor` is high only for the
  latter path-class case
  (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:260,272-275`). Clarify whether permission to cause an effect is
  itself a checkpoint/high-risk trigger, and make the family and risk outputs consistent.
- **The G-R patch point is underspecified.** The draft asks `EvaluateDispatch` to add contract risk
  to `strong` (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:280-293`), but that function accepts host-history records and a `Dispatch`,
  not the core contract (`internal/host/host.go:915-917`). The classifier runs in the CLI proposal,
  but the draft does not state how the evaluator verifies that `FamilySeparation` and risk were
  derived from the currently bound contract rather than trusting persisted host fields. Define the
  authoritative input and verification path before calling this a fix.
- **Decision L contradicts itself.** It offers enforcement for all runs or only contract-bound runs,
  then recommends the second option while also requiring E3 for all runs
  (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:473-475`). That
  hybrid leaves non-contract E0/E2, packet-less reviews, and final-review provenance unresolved.
- **The compatibility claim remains a proposal, not established behavior.** The plan's
  `material_milestone` can preserve prior digests if false is omitted, consistent with the current
  `json.Marshal` digest (`internal/model/contract.go:168-184`). But the draft does not define the
  exact JSON tags and absent/false semantics for each new packet field. The assertion that every
  field is additive and `omitempty`
  (`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:349-363`) needs a field-by-field schema, especially for
  optional `family_separation`, policy digest, and baseline. No existing code currently proves
  those record-byte compatibility claims.

No weakening of an existing §7.1 guarantee is explicit in the proposed gates; gate refusal rather
than waivable findings is directionally consistent (`docs/architecture.md:219-231,233-248`;
`docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md:342-347`). However, path rules that silently classify an unrecognized interface as `ordinary`
would weaken the current §16 mandatory-checkpoint guarantee in practice.

## Add to §6 decisions

1. **D7 — terminal outcome coverage:** does “final outcome closure” require final independent review
   for every terminal outcome, including unsuccessful outcomes, or only verified-success outcomes?
   Specify the transition paths and evidence needed for each.
2. **D8 — classifier completeness:** how does the design satisfy “path or interface that cannot be
   classified” when path-only rules can miss semantic interface changes? Choose a fail-closed method
   or request an explicit architecture amendment.
3. **D9 — risk provenance and confirmation:** identify the contract-bound risk value consumed by the
   host gate and evaluator; specify how HMA detects a lower host value. Resolve the tension between
   Decision R's “independence only” recommendation and §13's human-confirmed risk floor.
4. **D10 — scope and compatibility:** resolve Decision L's all-runs/contract-runs inconsistency,
   including packet-less and in-flight runs and non-success closure.
5. **D11 — external-effect semantics:** distinguish an allowed effect from an observed or planned
   effect, and state whether each sets a checkpoint, family separation, and high risk.

**Single next action — architecture §18:** **REVISE. Owner: Ming.** Update the design with the
findings above before asking Lennon for decisions. Proposed route: Ming / Anthropic /
`claude-opus-5-5`; no dispatch or approval is authorized by this report. Permission envelope:
draft-only revision; no code or architecture edit. Exact approval: Lennon decides the surfaced
product and architecture choices after the revision; no implementation begins before the required
architecture amendment. Stop and escalate if a correction would narrow a current mandatory
checkpoint or risk guarantee.

## Revision 2 re-check (2026-10-31)

**Verdict: PASS WITH FINDINGS — r2 is sound enough to present to Lennon as a decision packet.**
The r1 disputes checked here are materially resolved in the draft. The remaining findings below
are bounded drafting defects. They do not hide the choices that Lennon must make.

**G-R — resolved as a concrete proposal, not yet implemented.** Section 3.5 E0 step 4 now names
the authoritative value (`state.Contract.Risk`), the digest proof (`engine.BoundContractDigest(core)`),
the core-head check under the append lock, the rank comparison against `decision.RequiredRisk`, the
refusal behavior, and the binding fields for later verification. The cited code supports that path:
`addContractStateWithPlan` parses the contract and checks its digest
(`cmd/hma/packetstate.go:22-38`); the dispatch callback re-checks the captured core head before
`EvaluateDispatch` (`cmd/hma/review.go:448-465`). E0 step 5 also closes the packet-less contract
dispatch bypass. This is now implementable and testable. The current code still does not perform
the new risk comparison; r2 correctly presents it as proposed work.

**T7 — resolved and the live/latent distinction is correct.** The engine accepts only
`ABORTED`, `PARTIAL`, and valid failure outcomes at the outcome gate; it rejects the two
`VERIFIED_*` outcomes with `OUTCOME_NOT_IMPLEMENTED` (`internal/engine/stage_state.go:216-217`;
`internal/engine/terminal_outcomes_test.go:34-49`). `PARTIAL` is accepted when criteria include
both a resolved and unresolved disposition and there is no active failure classification
(`internal/engine/stage_state.go:244-266`; `internal/engine/transitions.go:96-112`). The current
host closure checks run only for verified outcomes (`cmd/hma/transition.go:301-332`), so the
`PARTIAL` route is reachable without that final review/credit/artifact gate. R2 explicitly calls
for extending the gate to `PARTIAL` if D7 selects it (§1.5 step 6, §2.3, §3.5 E3). Its seven-outcome
set and D7 alternatives now distinguish literal §16 coverage from the recommended success-claiming
subset (§3.4, §6).

**Decision L — resolved, not just renamed.** D10 recommends one consistent scope: contract-bound
runs use E0/E2/E3; non-contract runs cannot record `VERIFIED_*` or `PARTIAL`; non-success stop
outcomes remain available. It states the packet-less and in-flight rules and labels all-runs
enforcement as an alternative, not a second requirement (§3.5, §6 D10). The old L item is marked
withdrawn and replaced.

**T4 / `RiskFloor` — aligned.** T4 covers both an allowed external effect and an observed external
or irreversible path. Either form raises `RiskFloor` to `high`; the family-separation rule uses
that floor, and D11 recommends the same behavior (§3.4, §6 D11). The trigger, floor, and family
outputs are consistent.

**New drafting findings.**

- Section 2.3 says G-R is “under Decision T,” but §6 lists no Decision T. D9 covers risk provenance
  and confirmation, but the intended cross-reference must be stated rather than inferred.
- `CheckpointBaseline` in §3.4 declares `ContractDigest` and `PermissionClasses` without JSON tags.
  Section 3.5a requires the nested JSON keys `contract_digest` and `permission_classes`. Go's
  default JSON names would not produce those keys. Add explicit tags or state the serialization
  mapping so the proposed schema is unambiguous.

Section references inspected are present, including §3.5a and architecture §20. Section 6 has
distinct decisions D and D7–D11; L and R remain only as notes that they were replaced or merged.
The §3.4 classifier fields map to the §3.5a binding fields; the nested `CheckpointBaseline` key
spelling is the exception noted above.

**Next action — architecture §18:** present r2 to Lennon for the explicit decisions, while correcting
the two bounded drafting findings above. No implementation is authorized by this re-check.

## Revision 3 safety-boundary check (2026-10-31)

**Verdict: r3 is not ready for final sign-off as written.** U is a safe MVP choice. D8 narrows a
current §16 requirement and needs Lennon to accept the explicit amendment. D11's permission-only
choice is safe by itself, but its observed-only high-risk condition leaves actual, unobserved
external effects outside §13; revise or explicitly amend that rule before sign-off.

**Current independence enforcement.** `internal/host/host.go:980-991` sets `strong` for high or
critical dispatch risk, disputed or final validation, or a packet issued for `final`. For every
implementation contribution, it always rejects a matching worker ID, model, or work-context
digest. It rejects a matching provider or model family only when `strong` is true. `checkpoint`
is not in the `strong` condition. Thus U's T8 checkpoint does not itself get family separation in
the current host code; that stronger T8 treatment existed only in r1/r2 of this unmerged draft.
The host code has no T8 or misclassified-path logic today. None of U, D8, or D11 authorizes a
same-worker, same-model, or same-context reviewer to satisfy a review that is dispatched: the
existing checks remain. D8's no-checkpoint case still has the per-change review.

**U — SAFE-TO-LOOSEN.** §16 requires a checkpoint for a path or interface that cannot be classified,
but does not assign that trigger high-risk independence. Normal independence therefore preserves
§13's different-model rule and the host's distinct-worker/model/context checks. Ming's reading of
“materially unknown environments” as run-environment uncertainty rather than an unmatched
repository path is plausible: §13 does not equate an unclassified path with an unknown
environment. Any actual high-risk, disputed, or final-validation condition still requires family
separation.

**D8 — BLOCKS.** §16 currently says, “A path or interface that cannot be classified is a mandatory
trigger.” D8 accepts that an approved policy can label an interface/schema/protocol/control path
`ordinary` and then omit its checkpoint. The draft correctly calls this a narrowing and flags a
`[CHANGES §16]` amendment. It is not a rollback of a path-specific check already implemented in
`host.go`, but it is a rollback of the current architecture's mandatory-trigger guarantee, not
merely a new review-strength default. It does not change §7.1, approval binding, or the review
independence checks. Do not treat it as compliant with current §16 unless Lennon explicitly accepts
the amendment.

**D11 — BLOCKS as written.** Permission to cause an effect is not itself an external side effect;
assigning that permission alone a normal-independence checkpoint is a plausible MVP choice and does
not roll back a current high-risk requirement. But D11 also makes family separation and a high
floor depend on an *observed* effect path. That is narrower than the current §13 trigger. The exact
§13 text is:

> HMA computes a deterministic minimum risk floor. Organization and project policy may raise it.
> The human confirms or raises it. A permitted downgrade must be an explicit waiver; safety-kernel
> failures remain unwaivable.
>
> Automatic high-risk triggers include security/authentication, secrets/privacy, destructive
> operations, production/infrastructure, publication, schema migration, public API or architecture
> changes, scientific assumption changes, broad diffs, missing tests, external side effects, waived
> correctness criteria, conflicting validator findings, and materially unknown environments.
>
> Independence scales with risk:
> - separate validation session always;
> - different model for normal work;
> - different provider/model family for high-risk, disputed, or final validation;
> - no silent validator substitution.

§13 says “external side effects,” not “observed external side effects.” The draft also says runtime
effects are not observable by HMA in v1. Therefore no observed changed path does not establish that
no external side effect occurred. The reading “permission is not an effect” is honest; the reading
“only an observed effect is covered by §13” is not established by the text and leaves a real
external effect without the stated high-risk treatment. This does not make an approval forgeable or
rewrite §7.1's permission boundary, but it does silently narrow current §13. Separate mere
permission from actual or planned effects, and state how an actual effect without an observed
repository path retains §13 treatment.

**Kernel boundary.** None of the three edits changes the §7.1 list or makes approval binding
forgeable. The list continues to protect authority and permission, evidence integrity, external
effect boundaries, separation of implementation and approval, and transition-record integrity
(`docs/architecture.md:219-231`). D8 and D11 nevertheless depart from current architecture
requirements outside that list; “not a §7.1 change” does not make those departures current-spec
compliant.

**Overall — something must change first:** clarify/amend D11 so actual external effects retain §13
high-risk treatment, and present D8 explicitly as a proposed §16 amendment for Lennon's decision.
Do not present r3 as already conforming to the current architecture.
