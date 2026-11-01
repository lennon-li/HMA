# Gate E — checkpoint-trigger classifier (Gate A design pass, draft)

**Date:** 2026-10-31
**Dispatch ID:** hma-gate-e-checkpoint-triggers-draft-20261031-r2
**Base:** `main` at `240119d`. The worktree holds this draft and its critique as untracked files (r1 said
"clean"; that was wrong). Spec: `docs/architecture.md` §16 "Checkpoint triggers" (merged at `e583f4a`, D6),
§5.2, §5.4, §7, §12.4, §13, §17. Roadmap: `2026-10-15-outcome-consolidation-opus-liz.md` §5 "Gate E".
**Scope:** design only. No `.go` file and no `docs/architecture.md` edit. Closes nothing; resolves amendment
open question 6 (`2026-10-16-arch-amendment-draft.md` §2) only when Lennon accepts the decisions in §6.
**Status:** **revision 4 — FINAL, all 15 decisions settled and accepted by Lennon. Ready for the architecture
amendment.** History: r2 was revised per the independent critique
`docs/handoffs/evidence/gate-d/audit-gate-e-checkpoint-draft-phil.md` (Phil / OpenAI `gpt-6-luna` / GitHub
Copilot, verdict FAIL on r1; PASS WITH FINDINGS on r2). In r3, Lennon delegated the 15 §6 decisions to Ming and
Phil under his stated MVP philosophy ("start looser and tighten slowly; HMA should be a trusted MVP, not a
blocker"), and Ming decided every §6 item. Phil's r3 safety-boundary check (same evidence file, "Revision 3
safety-boundary check") found U safe and D8 and D11 safe but narrowing current text. Lennon accepted D8 and D11
as-is on 2026-10-31 (r4). The `[CHANGES §N]` clauses are listed in §6 "All decisions settled".
**Author:** Ming / Anthropic `claude-opus-5-5` / Claude Code CLI.

Every new name in this document is **PROPOSED** unless it carries a file:line citation. Clauses that need new
`docs/architecture.md` text carry `[CHANGES §N]`.

### Revision 2 changes (critique item → where resolved)

| Critique item | Resolution |
|---|---|
| G-F wording too broad | §2.3 G-F scoped to the normal-risk, same-family path. **New correction found in r2:** `VERIFIED_*` outcomes are refused by the core today, so G-F is latent on that path and live on `PARTIAL` (§1.5 step 6). |
| G-R patch point underspecified | §3.5 E0 step 4: the check moves to the `EvaluateDispatch` caller, which holds the verified contract. `EvaluateDispatch` stays contract-free. Second caller covered in E0 step 5. |
| G-V too broad | §2.3 G-V restated as missing enforcement for trigger-positive transitions. |
| T7 covers success only | §3.4 T7 keyed to a named outcome set; §1.5 step 6 lists all paths; Decision D7. |
| Path-only classifier vs §16 | Decision D8; Decision C now states it does not meet §16 alone. |
| T4 vs `RiskFloor` | §3.4 T4, `RiskFloor`, and family rule made consistent; Decision D11. |
| Decision L contradiction | Decision L replaced by one position in Decision D10. |
| Additivity asserted in prose | §3.2, §3.3, and new §3.5a give field-by-field JSON tags and absent/zero rules. |
| Risk provenance vs §13 confirmation | Decision D9 (absorbs Decision R). |

## 1. Current behavior

### 1.1 The issuing-reason enum

- `IssuingReason` is a closed string enum with three values: `per_change`, `checkpoint`, `final`
  (`internal/host/host.go:195-202`).
- It is stored on the review packet binding as `issuing_reason` (`internal/host/host.go:211`).
- Its only validation is enum membership (`validIssuingReason`, `internal/host/host.go:459-461`, called from
  `validatePacketBinding`, `internal/host/host.go:467`).

### 1.2 Who chooses the reason

The host chooses it. HMA does not derive it.

1. The host writes `issuing_reason` in the `hma host review-packet` input (`hostReviewPacketInput.IssuingReason`,
   `cmd/hma/review.go:30`).
2. `runHostReviewPacket` copies it unchanged into the binding (`cmd/hma/review.go:373-379`, field at `:375`).
3. The packet header prints it for the reviewer (`cmd/hma/review.go:277`, `:296`). The packet digest covers it.
4. The dispatch record is appended through `hoststate.EvaluateDispatch` (`cmd/hma/review.go:458`).

No code path computes, checks, or upgrades the reason from the diff, the contract, the plan, or the stage.

### 1.3 What each reason does

| Reason | Effect in code |
|---|---|
| `per_change` | None beyond the default. |
| `checkpoint` | **None.** No code reads `IssueCheckpoint` except the enum check at `host.go:460`. It behaves exactly like `per_change`. |
| `final` | (a) Forces provider- and model-family separation from every implementer of the unit (`internal/host/host.go:980-991`). (b) An `APPROVE` with isolation other than `isolated` is refused (`internal/host/review_import.go:230-232`). |

Strong (family-separated) independence is also switched on by three other dispatch fields: `Risk` high or
critical, `Disputed`, and `FinalValidation` (`internal/host/host.go:980`). All three are host-written
(`internal/host/host.go:187-189`). For `Risk`, HMA checks only enum membership (`internal/host/host.go:666`).
It never compares the dispatch risk with the contract risk (`internal/model/contract.go:61`). A repository-wide
search finds no other reader of `contract.Risk` outside validation (`internal/model/contract.go:413`).

`EvaluateDispatch(records []Record, d Dispatch, at time.Time) error` (`internal/host/host.go:915`) takes host
history and the dispatch only. It has no contract input. It has two production callers:

- `runHostReviewPacket`, inside its `appendHostRecord` closure (`cmd/hma/review.go:448-465`, call at `:458`).
  That closure also receives the core chain.
- `runHostDispatch` (`cmd/hma/host.go:229-259`, call at `:251`). It refuses a caller-supplied packet
  (`:244-245`), so a review dispatch through it is packet-less, with host-written `risk`.

### 1.4 The `review_dispatch` record shape (§12.4 "Launch classes")

The §12.4 `review_dispatch` is a host `dispatch` record with `role: REVIEW` and a non-nil `packet`
(`internal/host/host.go:177-193`). The packet binding (`internal/host/host.go:207-227`) holds:
`review_dispatch_id`, `review_nonce`, `dispatch_run_id`, `issuing_reason`, the expected launch head and
worktree, project root, base revision, produced head, diff digest, optional worktree digest, `valid_until`
(1-hour TTL, `internal/host/host.go:231`), and ledger namespace. The dispatch adds the packet digest,
implementation record digest, artifact digest, and the host-written `risk` / `disputed` / `final_validation`.

The binding is HMA-built. `runHostReviewPacket` refuses host input that carries a packet
(`cmd/hma/review.go:321`), and `runHostDispatch` refuses one too (`cmd/hma/host.go:244-245`).

The §12.4 "required reviewer independence" binding is **not a stored field**. It is recomputed at dispatch
time from `risk`, `disputed`, `final_validation`, and `issuing_reason` (`internal/host/host.go:980-981`).

### 1.5 Where reviews are required today (real call path)

1. **Packet issuance.** `hma host review-packet` → `runHostReviewPacket` (`cmd/hma/review.go:302`). HMA
   snapshots the target, builds the bound state, and, only for a contract-bound run, builds the changed-file
   manifest (`cmd/hma/review.go:399-415`). For a contract-bound run, the contract document must hash to
   `engine.BoundContractDigest(core)` (`cmd/hma/packetstate.go:22-38`). Under the lock, the core head anchor
   is re-checked (`cmd/hma/review.go:449`). It appends the dispatch through `EvaluateDispatch`
   (`internal/host/host.go:915-993`).
2. **Import.** `hma host review-import` → `EvaluateReviewImport` (`internal/host/review_import.go:146-258`), then
   `EvaluateReview` (`internal/host/host.go:995-1029`).
3. **Findings.** `hma host findings-import` writes a core `findings_snapshot` (`cmd/hma/review.go:588-710`,
   flag at `internal/model/types.go:319`).
4. **Per-change gate.** `requireHostTransitionGateIn` (`cmd/hma/transition.go:37-121`) gates exactly two edges
   (`cmd/hma/transition.go:40-41`):
   - `IMPLEMENTATION_AUTHORIZATION -> IMPLEMENTATION_REVIEW`: an implementation record must exist.
   - `IMPLEMENTATION_REVIEW -> VERIFICATION`: `RequireApprovedReview` (`cmd/hma/transition.go:72`,
     `internal/host/host.go:1075-1093`), resolved findings (`:77`), and an artifact match (`:114`).
5. **Closure.** For `VERIFIED_SUCCESS` / `VERIFIED_WITH_WAIVERS` (`cmd/hma/transition.go:304-327`):
   resolved findings, no active waiver for success, and `requireIndependentCredit`
   (`cmd/hma/credit.go:474-521`). Credit relies on the unit's latest imported review
   (`internal/host/findings_import.go:214-235`). That review must be `APPROVE`, `isolated`, report verdict
   `pass` (`cmd/hma/credit.go:110`). **Credit does not check `issuing_reason` or family separation.**
6. **Terminal outcomes (all seven).** The host gate above runs first; `transition.Apply` runs after it
   (`cmd/hma/transition.go:301-332`).

   | Outcome | Legal from | Current requirement |
   |---|---|---|
   | `VERIFIED_SUCCESS`, `VERIFIED_WITH_WAIVERS` | `RELEASE_AND_CLOSURE` only (`internal/engine/transitions.go:106-112`, `:182`) | Host closure gate (step 5). **Then refused by the core** with `OUTCOME_NOT_IMPLEMENTED` (`internal/engine/stage_state.go:216-217`; asserted at `internal/engine/terminal_outcomes_test.go:34-35,44-49`; §5.4 defers them to Phase B). |
   | `PARTIAL` | any nonterminal stage (`internal/engine/transitions.go:96-104`) | At least one `PASSED`/`WAIVED` and one `PENDING`/`FAILED` criterion, no failure classification (`internal/engine/stage_state.go:247-266`). **No credit re-check, no artifact snapshot, no review.** |
   | `FAILED`, `BLOCKED`, `UNKNOWN` | any nonterminal stage | A matching human-approved classification (`internal/engine/stage_state.go:244-245`). No review. |
   | `ABORTED` | any nonterminal stage | Human-initiated. No review. |

   The only production writer of core criterion records is criterion credit (`cmd/hma/credit.go:246`). A
   `PASSED` criterion was therefore credited once. At `PARTIAL`, nothing checks that the credit is still current
   or that its review was a final audit.

`VERIFICATION -> INDEPENDENT_VALIDATION` and `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE` have no host gate.
Each needs only the human approval and the legal-edge check (`internal/engine/transitions.go:79-88`).

### 1.6 Related facts the classifier can reuse

- **Changed-path observation.** `observedEntry` (`cmd/hma/observe.go:20-26`) gives path, status, added and
  deleted lines, content SHA-256, old and new mode, binary, staged, and truncation flags. HMA computes it from
  its own capture, never from host input. A rename appears as a delete plus an add (`cmd/hma/disposition.go:97`).
  A manifest over `maxObservedEntries` refuses the packet (`cmd/hma/review.go:479`), so a packet never carries
  a partial manifest.
- **Contract identity.** `BoundContractDigest` (`internal/engine/contract_binding.go:25-34`). Every
  `ACCEPTANCE_CRITERIA -> PLANNING` approval records its digest, so the chain holds the digest history.
- **Plans.** Plans are retained by digest (`retainPlan` / `loadPlans`, `cmd/hma/loop.go:390`, `:403`).
  `currentPlanUnit` resolves the unit of the current lease generation (`cmd/hma/disposition.go:20-55`).
- **Permission vocabulary.** `read_only`, `bounded_write`, `no_external_effect`
  (`internal/model/route_policy.go:48-58`). The contract carries `permission_envelope`
  (`internal/model/contract.go:65`). The unit carries `permission_classes` (`internal/model/contract.go:105`).
- **Risk order.** `low`, `medium`, `high`, `critical` (`internal/model/route_policy.go:64-67`).
- **Measured facts pattern.** `findingFacts` (`cmd/hma/disposition.go:72-90`) already derives facts from HMA's
  own measurement, not from auditor text. The classifier follows the same pattern.
- **Document encoding.** Contract and plan digests are `json.Marshal` of the struct (`documentDigest`,
  `internal/model/contract.go:171-178`). Host record anchors are `json.Marshal` of the record with the anchor
  cleared (`internal/host/store.go:30-31`). The contract and plan decoder requires every key the typed value
  re-encodes (`requireAllKeys`, `internal/model/contract.go:214-262`). No contract or plan field uses
  `omitempty` today.

## 2. Actual gap

### 2.1 Per trigger condition

| # | §16 trigger | Computed today? | Data available today? |
|---|---|---|---|
| T1 | Modified acceptance criterion or permission boundary | No | **Yes.** Contract digest history in the core chain. Unit permission classes in retained plans. |
| T2 | High or critical risk | Partly. Only through host-written dispatch `risk` (§1.3). | **Yes.** `contract.Risk`. Never consulted. |
| T3 | Public interface, persisted schema, or protocol change | No | **No.** No path or interface class exists anywhere in the model. |
| T4 | Irreversible or external effect | No | **Partial.** A unit without `no_external_effect` permits external effect. Changed paths have no effect class. Runtime effects are not observable by HMA in v1 (§12: HMA does not dispatch). |
| T5 | Security, approval, routing, evidence, or closure logic change | No | **No.** Same as T3. |
| T6 | Completion of an approved material milestone | No | **No.** No milestone concept in `hma.plan/v1` (`internal/model/contract.go:93-117`). |
| T7 | Final outcome closure | **Not enforced** on any of the seven terminal outcomes (§1.5 step 6). | **Yes.** The transition target outcome. |
| T8 | Path or interface that cannot be classified | No | **Not representable** (§2.2). |

A repository-wide search of `*.go` for milestone, public interface, schema change, irreversible, and external
effect finds only the `no_external_effect` permission class.

### 2.2 "Cannot be classified" is not representable

1. No classification output exists. Nothing maps a changed path to a class.
2. No classification policy exists to classify against.
3. `IssuingReason` has no "unclassified" value, and no record stores per-path classes or a trigger list.
4. Non-contract runs build no manifest at all (`cmd/hma/review.go:399`), so their changed paths are not even
   enumerated in the packet.

### 2.3 Defects against the current spec (not new design)

These hold against the current §13, §16, and §12.4 text, independent of the checkpoint design:

- **G-F (closure without a final audit).** §16 lists "final outcome closure" as a mandatory checkpoint trigger.
  §13 requires provider/model-family separation for final validation. No terminal path checks for a final audit.
  - *`VERIFIED_*`, normal-risk path.* The host closure gate (`cmd/hma/transition.go:304-327`) accepts credit
    relying on a `per_change` review (`cmd/hma/gated_credit_test.go:51,184-203`). When the host sets dispatch
    risk `low` or `medium`, with no `disputed`, no `final_validation`, and reason `per_change`,
    `EvaluateDispatch` requires only a different worker, model, and context (`internal/host/host.go:980-989`).
    A **same-family** reviewer can then back the credit.
  - *`VERIFIED_*`, high-risk path.* A correctly supplied high or critical risk already forces family separation
    (`internal/host/host.go:980`, `:988`). The cited fixture is this case (dispatch risk high,
    `cmd/hma/gatec_review_import_test.go:81`). Here the only gap is that the review is not a final-level audit.
  - *Reachability.* The core evaluator refuses every `VERIFIED_*` outcome today (§1.5 step 6). G-F on
    `VERIFIED_*` is therefore **latent**: the host gate would accept, then the core refuses. It becomes live when
    Phase B enables verified outcomes. r1 implied it was reachable now; that overstated it.
  - *`PARTIAL` is live.* It asserts some criteria `PASSED`, closes the run, and checks no review, credit
    freshness, or artifact (§1.5 step 6).
  - *Non-success outcomes.* Under the literal §16 text, `FAILED`, `BLOCKED`, `UNKNOWN`, and `ABORTED` are also
    "final outcome closure". None requires any review. Whether they should is Decision D7.
- **G-R (self-declared review risk).** A host can dispatch a review with `risk: low` for a contract with
  `risk: high` and get normal independence (§1.3). Under the architecture's Decision T (§22, trusted-host
  assumption — not a local §6 item) this is drift that HMA neither prevents nor detects. It exists on both `EvaluateDispatch` callers (§1.3). Route coherence does not close it: it
  evaluates a caller-supplied risk and is not called at packet issuance (critique, "Confirmed-accurate claims").
- **G-V (trigger-positive IV exits unenforced).** §16 requires a checkpoint when a listed trigger fires.
  `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE` has no host gate (`cmd/hma/transition.go:37-42`), so a
  trigger-positive unit can leave `INDEPENDENT_VALIDATION` with no checkpoint-grade review. The defect is the
  **missing enforcement for trigger-positive transitions**. The current spec does not require a review on every
  IV exit. Decision D8 option (2) proposed that as a stricter design choice, not as a reading of §16; r3 does not
  choose it. IV's other
  exits (back to implementation authorization, planning, route selection) are returns, not progress, and are
  not in scope.

### 2.4 Adjacent gaps, noted and not scoped here

- §13 "deterministic minimum risk floor" has no implementation. `contract.Risk` is a human input only. The
  classifier in §3 can supply a floor (Decision D9).
- Plan `allowed_files` / `forbidden_files` are checked for non-emptiness only
  (`internal/model/contract.go:515-526`). Nothing matches them against the diff (Decision S).
- Rewind triggers (`internal/engine/invalidation.go:9-45`) take host-declared `ChangedInputs`. The only
  production caller is the read-only `hma eval invalidation` (`cmd/hma/eval.go:58`). T1 must not reuse that
  host-declared input.
- §17: the failure classes (`TRANSIENT_RUNTIME_FAILURE` … `UNKNOWN_FAILURE`) have no code. Only the
  repeated-failure-signature refusal exists (`cmd/hma/attempt.go:391`). Out of scope for Gate E.
- A run with no criterion records would satisfy `VERIFIED_SUCCESS`'s "every criterion `PASSED`" vacuously
  (`internal/model/validate.go:334-338`), and `requireIndependentCredit` checks only `PASSED` criteria
  (`cmd/hma/credit.go:504-519`). Latent while the core refuses `VERIFIED_*`. Decision D10 closes it for
  non-contract runs. Not verified for contract-bound runs.

## 3. Proposed contract and code changes

### 3.1 Principle

HMA derives the minimum review level from its own observation and the core chain. The host may request a
stronger level and never a weaker one. Every derived input is either in the core chain, in an HMA-retained
document whose digest the chain binds, or in HMA's own capture. A persisted derived value is trusted at a later
gate only when it names the contract digest that is bound at that gate.

### 3.2 Classification policy document (new capture)

PROPOSED `hma.checkpoint-policy/v1`, a canonical JSON document parsed with the existing strict decoder
(`decodeCanonicalInput`, `internal/model/contract.go:190`):

```json
{
  "schema": "hma.checkpoint-policy/v1",
  "rules": [
    {"id": "cli-surface", "glob": "cmd/hma/**", "classes": ["public_interface"]},
    {"id": "store-schema", "glob": "internal/model/**", "classes": ["persisted_schema"]},
    {"id": "approval", "glob": "internal/engine/approval_binding.go", "classes": ["approval"]},
    {"id": "docs", "glob": "docs/**", "classes": ["ordinary"]}
  ]
}
```

- Closed class vocabulary: `public_interface`, `persisted_schema`, `protocol`, `external_effect`,
  `irreversible`, `security`, `approval`, `routing`, `evidence`, `closure`, `ordinary`. An unknown class
  refuses the document.
- Match is a slash-separated path glob with `*` (one segment) and `**` (any segments). Standard library only.
  No content matching in v1 (Decision C, Decision D8).
- A path takes the union of the classes of every rule that matches it.
- A path with any class other than `ordinary` is triggering. `ordinary` never cancels another class.
- A path that no rule matches is `UNCLASSIFIED` (T8).
- With no policy, every changed path is `UNCLASSIFIED`. This is the fail-closed default that §16 requires.
- Authority over `ordinary` follows Decision O.
- Every policy field is required, under the existing rule (`internal/model/contract.go:214-219`).

**Carrier (Decision P).** One additive field on `hma.acceptance-contract/v1`
(`internal/model/contract.go:55-67`). [CHANGES §9]

| Go field | JSON tag | Absent | Present |
|---|---|---|---|
| `CheckpointPolicyDigest string` | `checkpoint_policy_digest,omitempty` | No policy: every changed path is `UNCLASSIFIED`. | Same digest format as `policy_digest`; must equal the retained policy's digest. |

- **Digest compatibility.** `documentDigest` is `json.Marshal` of the struct. With `omitempty`, a contract
  without the field marshals to the same bytes, so existing contract digests do not change.
- **One byte form per meaning.** This is the first `omitempty` field in the contract format. `requireAllKeys`
  walks only the keys the typed value re-encodes, so `""` or `null` would decode and hash the same as absent.
  PROPOSED: refuse the key when present with `""` or `null`. Absent is the only encoding of "no policy".
- **Rewind.** A policy change changes the contract digest. That rewinds through the existing contract rules and
  fires T1.
- **Forward compatibility.** An older HMA binary refuses a contract carrying the field (`DisallowUnknownFields`,
  `internal/model/contract.go:204`). Acceptable: one binary governs a run.

### 3.3 Plan field for milestones (new capture)

One additive field on a plan unit (`internal/model/contract.go:93-117`). [CHANGES §10] (Decision M.)

| Go field | JSON tag | Absent | Present |
|---|---|---|---|
| `MaterialMilestone bool` | `material_milestone,omitempty` | `false`. | Only `true` is accepted. Explicit `false` or `null` is refused (same one-byte-form rule as §3.2). |

With `omitempty`, the digests of existing plans do not change (`internal/model/contract.go:171-178`).

### 3.4 The classifier (pure, `internal/engine`)

PROPOSED, standard-library-only, side-effect-free, consistent with the engine package contract
(`internal/engine/transitions.go:1-9`):

```go
type CheckpointTrigger string // PROPOSED; prefixed to avoid a clash with RewindTrigger's Trigger* names

const (
    CheckpointContractOrPermission CheckpointTrigger = "CONTRACT_OR_PERMISSION_CHANGED"     // T1
    CheckpointHighRisk             CheckpointTrigger = "HIGH_OR_CRITICAL_RISK"              // T2
    CheckpointInterface            CheckpointTrigger = "INTERFACE_SCHEMA_OR_PROTOCOL"       // T3
    CheckpointExternalEffect       CheckpointTrigger = "IRREVERSIBLE_OR_EXTERNAL_EFFECT"    // T4
    CheckpointControlLogic         CheckpointTrigger = "SECURITY_APPROVAL_ROUTING_EVIDENCE_OR_CLOSURE" // T5
    CheckpointMilestone            CheckpointTrigger = "APPROVED_MATERIAL_MILESTONE"        // T6
    CheckpointFinalClosure         CheckpointTrigger = "FINAL_OUTCOME_CLOSURE"              // T7
    CheckpointUnclassified         CheckpointTrigger = "UNCLASSIFIED"                       // T8
)

// FinalAuditOutcomes is the set of terminal outcomes that fire T7. Its content is Decision D7.
// Literal §16: all seven. Decided (D7): VERIFIED_SUCCESS, VERIFIED_WITH_WAIVERS, PARTIAL.
var FinalAuditOutcomes map[model.TerminalOutcome]bool

type ChangedPath struct { Path, Status, OldMode, NewMode string; Binary bool } // projection of observedEntry

type CheckpointBaseline struct { // contract digest and unit permission classes a checkpoint-grade review covered
    ContractDigest    string   `json:"contract_digest"`
    PermissionClasses []string `json:"permission_classes"`
}

type CheckpointInput struct {
    Contract       *model.AcceptanceContract // bound contract document, already proven equal to BoundContractDigest
    Policy         *CheckpointPolicy         // nil means "no policy": every path is UNCLASSIFIED
    Unit           model.PlanUnit
    Baseline       CheckpointBaseline
    Current        CheckpointBaseline
    Manifest       []ChangedPath             // complete, from HMA's own capture
    Stage          model.Stage               // the unit's projected stage at issuance or gate time
    TargetOutcome  model.TerminalOutcome     // set only at a terminal-outcome gate
    HostRisk       model.RouteRiskLevel      // host-written dispatch risk; may only raise
    Disputed       bool
}

type CheckpointDecision struct {
    Level             IssuingLevel        // per_change < checkpoint < final
    Triggers          []CheckpointTrigger // sorted, unique
    UnclassifiedPaths []string            // sorted
    FamilySeparation  bool
    RiskFloor         model.RouteRiskLevel
    RequiredRisk      model.RouteRiskLevel // max(Contract.Risk, RiskFloor); the dispatch risk must reach it
    PolicyDigest      string
    ManifestDigest    string
}

func ClassifyCheckpoint(in CheckpointInput) CheckpointDecision
```

Rules:

| # | Fires when | Source |
|---|---|---|
| T1 | `Current.ContractDigest != Baseline.ContractDigest`, or the unit's permission classes differ as sets | core chain + retained plan |
| T2 | `max(contract.Risk, HostRisk, RiskFloor)` is `high` or `critical` | contract document + floor |
| T3 | any path has `public_interface`, `persisted_schema`, or `protocol` | policy + manifest |
| T4 | **allowed effect:** the unit's permission classes lack `no_external_effect` (checkpoint only); **or observed effect:** any path has `external_effect` or `irreversible` (checkpoint, floor, family separation) (Decision D11) | plan + policy + manifest |
| T5 | any path has `security`, `approval`, `routing`, `evidence`, or `closure` | policy + manifest |
| T6 | `Unit.MaterialMilestone` and `Stage == INDEPENDENT_VALIDATION` | plan |
| T7 | `TargetOutcome ∈ FinalAuditOutcomes` (Decision D7), or `Stage == RELEASE_AND_CLOSURE` | transition input |
| T8 | any path is unclassified, or `Contract == nil` with a non-empty manifest (reachable only under Decision D10 option (2)) | policy + manifest |

- **Baseline.** The baseline is the `checkpoint_covers` value (§3.5a) recorded on the unit's latest approved
  review with level `checkpoint` or `final`. That value is the `Current` state at the review's issuance. With no
  such review, the baseline is the contract digest of the run's first contract acceptance and the unit's
  permission classes in its first lease generation's plan. The first contract does not fire T1 by itself. Any
  re-acceptance fires T1 until a checkpoint-grade review passes on it.
- **Level.** `final` if T7. Otherwise `checkpoint` if the stage is `INDEPENDENT_VALIDATION` and any trigger
  fires (Decision D8 (1)). Otherwise `per_change`.
- **RiskFloor.** `high` when T3, T4 in its observed form, or T5 fires. Otherwise `low`. T8 (Decision U (2)) and
  an allowed-only T4 (Decision D11 (2)) do not raise it. It only raises. [CHANGES §13] (Decision D9.)
- **Family separation.** `FamilySeparation = max(contract.Risk, HostRisk, RiskFloor) ∈ {high, critical} ||
  Disputed || T7`. This one rule makes family separation and `RiskFloor` consistent:
  - T3, observed T4, and T5 raise the floor, so they set family separation through risk. §13 lists security,
    schema migration, public API, and external side effects as automatic high-risk triggers.
  - T2 sets it directly. T2 also fires whenever the floor is high. That is intended: the floor is a risk.
  - T7 sets it without raising risk, because §13 requires family separation for final validation, not high risk.
  - T1, T6, T8, and an allowed-only T4 alone set neither. They get a checkpoint with §13 normal independence
    (Decisions U and D11).
- **RequiredRisk.** `max(contract.Risk, RiskFloor)`. E0 refuses a dispatch whose risk is below it (§3.5).
- **Determinism.** Output is sorted, with no map-iteration order and no I/O. The same input in any order gives
  the same output.

### 3.5 Where it plugs in

Scope follows Decision D10. As decided, E0, E2, and E3 apply to contract-bound runs, and a non-contract run
cannot record a success-claiming outcome.

**E0 — packet issuance** (`runHostReviewPacket`, `cmd/hma/review.go:302`).

1. After the capture (`:348`) and manifest (`:403`), call `ClassifyCheckpoint`. Add the §3.5a fields to the
   binding before the packet is built (`:420`).
2. Refuse if the host-requested `issuing_reason` is below `decision.Level`, with PROPOSED reason
   `HOST_CHECKPOINT_REASON_TOO_LOW`. Report the derived level. HMA does not silently upgrade the reason,
   because §12.4 prohibits silent substitution.
3. Print the §3.5a fields in the packet header (`cmd/hma/review.go:272-284`) so the reviewer sees why it was
   dispatched. The packet digest covers them.
4. **Risk provenance (fixes G-R on the packet path).** `EvaluateDispatch` stays contract-free. Its caller owns
   the contract-risk check, inside the `appendHostRecord` closure (`cmd/hma/review.go:448-465`), after the core
   head check (`:449`) and before the `EvaluateDispatch` call (`:458`):
   - *Authoritative input.* `state.Contract.Risk`. HMA read that document and proved it hashes to
     `engine.BoundContractDigest(core)` (`cmd/hma/packetstate.go:22-38`). The head check at `:449` proves, under
     the lock, that the core has not moved since, so the contract is still the bound one.
   - *Check.* Refuse if `in.Dispatch.Risk` ranks below `decision.RequiredRisk` (order at
     `internal/model/route_policy.go:64-67`), with PROPOSED reason `HOST_REVIEW_RISK_BELOW_REQUIRED`. HMA refuses
     rather than rewriting the host field, as in step 2.
   - *Effect.* The existing `strong` term (`internal/host/host.go:980`) now sees `d.Risk ≥ high` whenever the
     contract or the floor is high. It applies family separation with no new input.
   - *Remaining term.* Add only `d.Packet != nil && d.Packet.FamilySeparation` to `strong`, for the cases risk
     does not express (T7, `Disputed`). `EvaluateDispatch` reads an HMA-built field, not a host field (§1.4).
   - *Record.* Store `bound_contract_digest`, `contract_risk`, `risk_floor`, and `family_separation` on the
     binding (§3.5a), so later gates can re-verify them.
5. **Packet-less review path (fixes G-R on the second caller).** `runHostDispatch`
   (`cmd/hma/host.go:229-259`) reaches `EvaluateDispatch` with a host-written review risk. Its closure has the
   core chain but not the contract document, so it cannot run step 4. PROPOSED: refuse `role: REVIEW` there
   when `engine.BoundContractDigest(core) != ""`, with the existing `hoststate.ReasonReviewImportRequired`
   (`HOST_REVIEW_IMPORT_REQUIRED`, `internal/host/host.go:105`). Contract-bound reviews then go only through
   `review-packet`. (Decision D10.)

**E1 — per-change gate (unchanged).** `IMPLEMENTATION_REVIEW -> VERIFICATION` still requires an imported
approving review for every coding change (`cmd/hma/transition.go:72`). Classification never removes it,
consistent with roadmap §3.

**E2 — checkpoint gate (new).** On `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE`, extend
`requireHostTransitionGateIn` (`cmd/hma/transition.go:40`). It recomputes `ClassifyCheckpoint` at gate time on
the live target. It applies when any trigger fires (the G-V fix). Under Decision D8 (1), a contract-bound unit
with no trigger passes E2 with no checkpoint review. When E2 applies, it requires that the unit's latest
imported review:

- is `APPROVE`, `isolated`, and has its findings snapshot applied with no unresolved `BLOCK` or `WAIVABLE`
  finding (reuse `cmd/hma/transition.go:77`, `:100-104`);
- has `checkpoint_schema` present and level `checkpoint` or `final`;
- was dispatched after the unit's approved `VERIFICATION -> INDEPENDENT_VALIDATION` transition, so the
  per-change review cannot stand in for the stage audit;
- has recorded `checkpoint_triggers` that are a superset of the gate-time triggers, and a
  `checkpoint_policy_digest` equal to the current one;
- has `bound_contract_digest == engine.BoundContractDigest(core)` at gate time, and a dispatch `risk` at least
  the gate-time `RequiredRisk`. A contract re-accepted after dispatch therefore stales the review;
- has an artifact equal to the transition's artifact (same rule as `cmd/hma/transition.go:114`).

Refusal reasons (PROPOSED): `HOST_CHECKPOINT_REVIEW_MISSING`, `HOST_CHECKPOINT_TRIGGERS_UNCOVERED`,
`HOST_CHECKPOINT_REVIEW_STALE`. [CHANGES §16] [CHANGES §12.4]

**E3 — final gate (new).** Put the check inside `requireIndependentCredit` (`cmd/hma/credit.go:474-521`), so
every outcome that relies on credit shares it. Require that every review a credit relies on
(`cr.credit.ReviewRecord`, `cmd/hma/credit.go:516`):

- has `checkpoint_schema` present and level `final`;
- has `bound_contract_digest` equal to the current bound digest. Credit already checks its own contract digest
  (`cmd/hma/credit.go:513`).

Family separation and isolation for `final` are already enforced (`internal/host/host.go:980-981`,
`internal/host/review_import.go:230`, `cmd/hma/credit.go:110`). The artifact match already holds
(`cmd/hma/credit.go:513`). Refusal reason (PROPOSED): `HOST_FINAL_AUDIT_MISSING`. [CHANGES §20] [CHANGES §16]

E3 runs on the outcomes in `FinalAuditOutcomes` (Decision D7):

- `VERIFIED_SUCCESS` / `VERIFIED_WITH_WAIVERS`: the existing closure branch (`cmd/hma/transition.go:304-327`).
  Latent until Phase B enables these outcomes in the core (§2.3). E3 must land before or with that enablement.
- `PARTIAL` (in the set under D7 (2)): extend the branch at `cmd/hma/transition.go:304` to `PARTIAL`. Snapshot the
  artifact and run `requireIndependentCredit`. Only `PASSED` criteria need credit; `WAIVED` criteria are a
  human operation and need nothing more. No-active-waiver does not apply.
- `FAILED`, `BLOCKED`, `UNKNOWN`, `ABORTED`: not in the set under D7 (2). E3 does not run; these outcomes keep
  today's requirements. (D7 (1) would have needed a new rule: a `final` review of the live artifact for every
  unit with an implementation.)

**Review count.** Because the latest review supersedes earlier ones in packet mode
(`AnswersLatestReviewDispatch`, `internal/host/host.go:1099-1110`), credit must be re-issued after each newer
review. A `final` review dispatched in `INDEPENDENT_VALIDATION` on an unchanged artifact can satisfy both E2
and E3 (Decision D). A single-unit triggered run then needs two reviews (per-change, then final), not three. In
a multi-unit run, each earlier unit needs its own checkpoint at its IV exit and a final review at the closure
artifact.

**Next actions.** The host-facing output gains PROPOSED next actions `ISSUE_CHECKPOINT_REVIEW` and
`ISSUE_FINAL_REVIEW` (§18).

### 3.5a Packet binding schema (new fields)

All fields are added to `ReviewPacketBinding` (`internal/host/host.go:207-227`). HMA writes them; the host
cannot (§1.4). All use `omitempty`.

| Go field | JSON tag | Type | Absent means | Present rule |
|---|---|---|---|---|
| `CheckpointSchema` | `checkpoint_schema,omitempty` | string, fixed `hma.checkpoint/v1` | The classifier did not run (legacy or non-contract packet). E2 and E3 treat the review as **not** checkpoint-grade. | Set on every packet the classifier ran for. |
| `MinimumIssuingReason` | `minimum_issuing_reason,omitempty` | `IssuingReason` | — (only with schema) | Derived level. `issuing_reason` is never below it (E0 step 2). |
| `CheckpointTriggers` | `checkpoint_triggers,omitempty` | `[]CheckpointTrigger`, sorted, unique | No trigger fired. | Non-empty list. |
| `UnclassifiedPaths` | `unclassified_paths,omitempty` | `[]string`, sorted | No unclassified path. | At most `maxObservedEntries`. |
| `FamilySeparation` | `family_separation,omitempty` | bool | `false`. | `true` only. |
| `RiskFloor` | `risk_floor,omitempty` | `model.RouteRiskLevel` | — (only with schema) | Always set when the schema is present (`low` or `high`), so never omitted there. |
| `ContractRisk` | `contract_risk,omitempty` | `model.RouteRiskLevel` | Non-contract run. | Copied from the verified contract (E0 step 4). |
| `BoundContractDigest` | `bound_contract_digest,omitempty` | string | Non-contract run. | Equals `engine.BoundContractDigest(core)` at issuance. |
| `CheckpointPolicyDigest` | `checkpoint_policy_digest,omitempty` | string | No policy: every path was unclassified. | Equals the contract's `checkpoint_policy_digest`. |
| `ManifestDigest` | `manifest_digest,omitempty` | string, SHA-256 hex of `json.Marshal` of the sorted `[]ChangedPath` | No manifest (non-contract run). | Present whenever a manifest was built. |
| `CheckpointCovers` | `checkpoint_covers,omitempty` | `*CheckpointBaseline` (`contract_digest`, `permission_classes` sorted; both required inside) | — (only with schema) | `Current` at issuance; becomes the next baseline once the review is approved. |

Rules:

- **Absent vs false.** Every field except `checkpoint_schema` is interpreted only when `checkpoint_schema` is
  present. `validatePacketBinding` (`internal/host/host.go:467`) refuses a binding that carries any checkpoint
  field without `checkpoint_schema`. With the schema present, an absent `family_separation`,
  `checkpoint_triggers`, or `unclassified_paths` means `false` or empty. Without it, the value is unknown, and
  the review never counts for E2 or E3.
- **Record bytes.** Host record anchors are `json.Marshal` of the record with the anchor cleared
  (`internal/host/store.go:30-31`). A binding without the new fields marshals to the same bytes, so existing
  anchors do not change. This is the expected `encoding/json` behavior, not yet proven by a test here (§5.5).
- **Packet bytes.** New fields change only new packet files. The packet digest is SHA-256 of the packet bytes
  (`cmd/hma/review.go:424`). Existing packets are unchanged.
- **Field renamed from r1.** r1's `checkpoint_baseline` is now `checkpoint_covers`, because it records what the
  review covered, not the baseline it was compared against.

### 3.6 Architecture clauses needed (summary)

| Section | Clause |
|---|---|
| §5.2 / §16 | Which terminal outcomes count as "final outcome closure" for the mandatory final audit (Decision D7). The decided narrowing is a [CHANGES §16] amendment, not an interpretation. |
| §9 | Contract gains optional `checkpoint_policy_digest`. Absence means every changed path is unclassified. Present-but-empty is refused. [CHANGES §9] |
| §10 | Plan unit gains optional `material_milestone`. [CHANGES §10] |
| §12.4 | HMA derives the minimum `issuing_reason`. The host may only raise it. The review dispatch risk must reach `max(contract risk, floor)`. Contract-bound review dispatches require an HMA packet. `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE` is a host-gated edge. [CHANGES §12.4] |
| §13 | The checkpoint classifier supplies the deterministic minimum risk floor. It only raises. How the "human confirms" clause applies to a raised floor (Decision D9). [CHANGES §13] |
| §16 | Trigger semantics T1–T8, the policy format, the baseline rule, gates E2 and E3, and the classifier-completeness rule (Decision D8). [CHANGES §16] |
| §20 | Closure additionally requires a `final`-level review of the closure artifact for every credited unit. Success-claiming outcomes require a contract-bound run (Decision D10). [CHANGES §20] |

### 3.7 Kernel (§7.1) impact

Nothing above changes §7.1. Two items could be argued as kernel rules and are flagged, not proposed:
"unclassified ⇒ mandatory checkpoint" and "closure ⇒ final audit" (Decision K). The design makes a missing
checkpoint or final review a **gate refusal, not a finding**. It is therefore outside the §7.2 waivable
classes, and no waiver can clear it. The design does not make checkpoint policy waivable.

Two r3 decisions narrow current §16 text and one departs from current §13 text. Each needs an explicit
amendment that Lennon accepts; none is applied silently:

- Decision D7 (2): non-success outcomes need no final audit. [CHANGES §16]
- Decision D8 (1): policy misclassification is not "cannot be classified". The critique noted that a path rule
  calling an interface `ordinary` weakens the §16 guarantee in practice; D8 (1) accepts that as a named v1 gap.
  [CHANGES §16]
- Decision D9 (1): a floor HMA raises applies without human re-confirmation. [CHANGES §13]

None of the three removes an independence check that the code enforces today, touches approval binding, or
changes §7.1. Phil's r3 pass confirmed the U (2) reading of §13. It found that D11 (2) narrows current §13
("external side effects", unqualified by "observed"), so D11 (2) also needs a [CHANGES §13] amendment. Lennon
accepted D8 (1) and D11 (2) as-is (§6).

## 4. Compatibility impact

- **Stored records and documents.** Field-by-field rules are in §3.2, §3.3, and §3.5a. Existing contract and
  plan digests and existing host record anchors do not change, by `json.Marshal` with `omitempty`. *To prove in
  implementation:* the §5.5 byte-compatibility tests. The contract and plan formats gain their first optional
  fields. That departs from the "every field required" rule (`internal/model/contract.go:214-219`), with the
  one-byte-form refusal as the guard.
- **Non-contract runs.** Decision D10. As decided, they keep today's packet path, and they cannot
  record `VERIFIED_*` or `PARTIAL`.
- **In-flight contract-bound runs.** They have no `checkpoint_policy_digest`, so every path is unclassified.
  E2 then requires a checkpoint review for every unit, with normal independence (Decision U (2)), and E3
  requires a final review at closure. Their earlier
  reviews lack `checkpoint_schema` and never count for E2 or E3. This is stricter, not weaker.
- **Legacy (packet-less) reviews.** These carry no `issuing_reason` and no `checkpoint_schema`. Under Decision
  D10 they never satisfy E2 or E3, on any run, and new packet-less review dispatches are refused on
  contract-bound runs (E0 step 5). Their `final_validation: true` flag does not substitute for a final review.
- **Tests.** `cmd/hma` has 57 lines naming the `IssuingReason` constants across 13 test files. Most use
  `newReviewFixture(..., IssuePerChange, ...)` (`cmd/hma/gatec_review_import_test.go:97-123`). Their packets are
  issued before `INDEPENDENT_VALIDATION`, so the derived level is `per_change` and they remain valid. Their
  reviewer (anthropic/claude) is already family-separated from the implementer (openai/gpt)
  (`cmd/hma/gatec_review_import_test.go:113-114`), so derived family separation does not break them. Their
  dispatch risk is `high` (`cmd/hma/gatec_review_import_test.go:81`), so the E0 step 4 risk check does not
  refuse them.
  **Expected to break:** `cmd/hma/gated_credit_test.go`. Its fixture uses `IssuePerChange`
  (`cmd/hma/gated_credit_test.go:51`), and its outcome gate calls `requireIndependentCredit` directly
  (`:178`), where E3 lives. It needs a `final` review step.
  **To check:** `transition_outcomes_test.go` and `gateb_r2_test.go` name `PARTIAL`. Any that records `PARTIAL`
  over a credited `PASSED` criterion breaks under Decision D7 (2). `gateb_regression_test.go` and
  the same files may assert host-gate results for `VERIFIED_*`. The core refuses those outcomes today, so no
  test records one end to end. Not verified here.
- **Dogfood harness.** `scripts/dogfood/hostinput/main.go:301` sets a synthetic `Risk: high`. Its seed writes
  only a `PENDING` criterion (`scripts/dogfood/hostinput/main.go:186-191`). Whether a dogfood run reaches
  `PARTIAL` with a `PASSED` criterion is not verified here.
- **Host layer (Obsidian agent-dispatch / verify-loop adapters).** The host must issue checkpoint and final
  packets at the new points, set review risk at least `RequiredRisk`, stop using packet-less review dispatch on
  contract-bound runs, and handle the two new next actions. Amendment open question 7 already places this
  contract outside HMA core.
- **Docs.** `docs/task14-host-orchestration.md:146-149` describes `final` as host-selected. It needs an update.
- **Unverified assumption.** Packet issuance and findings import during `INDEPENDENT_VALIDATION` are assumed to
  work. `leasePlanDigest` (`cmd/hma/packetstate.go:86-94`) does not require an active lease, and the route
  approval survives `VERIFICATION -> INDEPENDENT_VALIDATION` (`cmd/hma/transition.go:246-249`). No test
  exercises this path today. A Gate E red test must confirm it first.
- **Not verified.** Whether `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE` approvals always carry a `unit_id`.
  E2 needs one, as the existing gate does (`cmd/hma/transition.go:44-46`).

## 5. Test plan

Unit tests for the pure classifier go in `internal/engine`. Gate tests go in `cmd/hma` and reuse
`newReviewFixture`. Tests marked *(D…)* or *(U)* follow that r3 decision in §6.

### 5.1 Per trigger

| Test (PROPOSED) | Setup | Expected |
|---|---|---|
| `TestCheckpointT1ContractReaccepted` | rewind and re-accept a new contract digest; unit at IV | T1; level `checkpoint`; family sep false |
| `TestCheckpointT1PermissionChanged` | new lease generation whose plan adds `bounded_write` to the unit | T1 |
| `TestCheckpointT1ClearedByCheckpointReview` | after an approved checkpoint review on the new contract | T1 absent |
| `TestCheckpointT1FirstContractNoTrigger` | single contract, no rewinds | T1 absent |
| `TestCheckpointT2ContractHigh` / `Critical` | `contract.risk` high / critical | T2; family sep; `RequiredRisk` high / critical |
| `TestCheckpointT3PublicInterface` / `Schema` / `Protocol` | change a path matched by each class | T3; RiskFloor high; family sep |
| `TestCheckpointT4AllowedEffect` *(D11)* | unit permission classes lack `no_external_effect`; docs-only diff | T4; level `checkpoint` at IV; RiskFloor low; family sep false |
| `TestCheckpointT4PathClass` *(D11)* | change a path classed `irreversible` | T4; RiskFloor high; family sep |
| `TestCheckpointT5EachControlClass` | one case per `security`/`approval`/`routing`/`evidence`/`closure` | T5; RiskFloor high |
| `TestCheckpointT6Milestone` | `material_milestone: true`; stage IV vs stage IMPLEMENTATION_REVIEW | T6 only at IV; family sep false |
| `TestCheckpointT7PerOutcome` *(D7)* | each of the seven terminal outcomes | T7 and level `final` exactly for `FinalAuditOutcomes`; family sep with T7 |
| `TestCheckpointT8NoPolicy` *(U)* | contract without `checkpoint_policy_digest` | every path in `UnclassifiedPaths`; T8; RiskFloor low; family sep false |
| `TestCheckpointT8UnmatchedPath` | policy covers `docs/**` only; change `src/x.go` | T8 for `src/x.go` only |
| `TestCheckpointOrdinaryDoesNotCancel` | path matches `ordinary` and `security` | T5 |
| `TestCheckpointRenameAcrossClasses` | rename `docs/a` → `internal/model/a` | both paths classified; T3 |
| `TestCheckpointOrdinaryOnlyNoTrigger` | only `ordinary` paths, low risk, `no_external_effect` present, no contract change | no triggers; level `per_change` before and at IV *(D8 (1))* |
| `TestCheckpointFamilyRiskInvariant` | table over trigger combinations, risks, `Disputed` | `FamilySeparation == (max risk ≥ high ∨ Disputed ∨ T7)` in every row |
| `TestCheckpointMisclassifiedPathKnownGap` *(D8 (1))* | interface change inside a path the policy classes `ordinary` | no trigger; pins the named v1 gap so tightening it later is a visible test change |
| `TestCheckpointPolicyCatchAllOrdinaryRefused` *(O)* | rule `{"glob": "**", "classes": ["ordinary"]}` | policy parse refused |
| `TestCheckpointPolicyUnknownClassRefused` | class `misc` | policy parse refused |
| `TestCheckpointDeterministic` | manifest and rules in shuffled order | identical decision |
| `TestClassifyCheckpointDoesNotMutateInput` | as `internal/engine/transitions_test.go:307` | input unchanged |

### 5.2 Roadmap asks: stale, missing, and self-review refusal

| Test (PROPOSED) | Expected |
|---|---|
| `TestCheckpointReasonTooLowRefused`: host requests `per_change` at IV with triggers | `HOST_CHECKPOINT_REASON_TOO_LOW`; no dispatch record; packet file removed (`cmd/hma/review.go:466-468`) |
| `TestReviewPacketRiskBelowContractRefused` (G-R red test; fails on `240119d`): contract high, dispatch low | `HOST_REVIEW_RISK_BELOW_REQUIRED`; no dispatch record; packet file removed |
| `TestReviewPacketRiskBelowFloorRefused`: contract low, T5 path, dispatch low | `HOST_REVIEW_RISK_BELOW_REQUIRED` |
| `TestHostDispatchReviewOnContractRunRefused` *(D10)*: packet-less review via `hma host dispatch` on a contract-bound run | `HOST_REVIEW_IMPORT_REQUIRED` |
| `TestCheckpointMissingAtIVExit`: triggers fire, no checkpoint review | `HOST_CHECKPOINT_REVIEW_MISSING` |
| `TestCheckpointNotRequiredWithoutTriggers` *(D8 (1))*: contract-bound, ordinary-only, no checkpoint review | IV exit passes E2 (E1 and the other existing gates still apply) |
| `TestCheckpointPerChangeReviewCannotStandIn`: only the E1 review exists | `HOST_CHECKPOINT_REVIEW_MISSING` (dispatched before the IV transition) |
| `TestCheckpointLegacyReviewCannotStandIn`: review without `checkpoint_schema` | `HOST_CHECKPOINT_REVIEW_MISSING` |
| `TestCheckpointStaleAfterTargetChange`: worktree changes after the checkpoint packet | existing `HOST_REVIEW_TARGET_CHANGED` |
| `TestCheckpointStaleAfterPolicyChange`: policy digest differs at gate | `HOST_CHECKPOINT_TRIGGERS_UNCOVERED` (or lease revocation first; assert which) |
| `TestCheckpointStaleAfterContractReaccepted`: `bound_contract_digest` differs at gate | `HOST_CHECKPOINT_REVIEW_STALE` (or rewind first; assert which) |
| `TestCheckpointTriggersUncovered`: packet recorded `{T3}`; gate computes `{T3,T5}` | `HOST_CHECKPOINT_TRIGGERS_UNCOVERED` |
| `TestCheckpointSelfReviewSameWorker` / `SameModel` / `SameContext` | `HOST_REVIEWER_NOT_INDEPENDENT` |
| `TestCheckpointSameFamilyRefusedWhenRequired` | `HOST_REVIEWER_NOT_INDEPENDENT` |
| `TestCheckpointDirectCostOrchestratorCannotReview`: implementer via `DIRECT_COST_EXCEPTION` reviews | `HOST_REVIEWER_NOT_INDEPENDENT` (`internal/host/host.go:452-453`) |
| `TestCheckpointPartialIsolationRefused` | not `isolated` → refused at gate |

### 5.3 Roadmap ask: false completion

| Test (PROPOSED) | Expected |
|---|---|
| `TestVerifiedOutcomeRefusedByCoreToday` (pins §2.3 latency) | host gate passes; core refuses `OUTCOME_NOT_IMPLEMENTED`. Removed when Phase B enables verified outcomes. |
| `TestClosureWithPerChangeReviewOnlyRefused` (G-F; host-gate level) | `HOST_FINAL_AUDIT_MISSING` from `requireIndependentCredit` |
| `TestClosureSameFamilyNormalRiskRefused` (G-F normal-risk path) | dispatch risk `low`, same-family reviewer, `per_change` → `HOST_FINAL_AUDIT_MISSING` at closure |
| `TestPartialWithPerChangeCreditRefused` *(D7)* (live G-F red test; fails on `240119d`) | `HOST_FINAL_AUDIT_MISSING` |
| `TestPartialWithStaleCreditRefused` *(D7)*: artifact changed after credit | existing `HOST_CREDIT_REFUSED` |
| `TestPartialWaivedOnlyNeedsNoReview` *(D7)* | `PARTIAL` with `WAIVED` + `PENDING` criteria and no `PASSED` passes |
| `TestStoppingOutcomesNeedNoReview` *(D7)* | `FAILED`, `BLOCKED`, `UNKNOWN`, `ABORTED` unchanged from today |
| `TestNonContractRunCannotClaimSuccess` *(D10)* | `PARTIAL` / `VERIFIED_*` on a non-contract run refused with PROPOSED `HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM` |
| `TestClosureFinalReviewOfOtherArtifactRefused` | existing `HOST_CREDIT_REFUSED` |
| `TestClosureFinalReviewSameFamilyRefused` | refused at dispatch (existing) |
| `TestClosureSelfReportedCriterionRecordRefused` | existing `HOST_CREDIT_REFUSED` (regression) |
| `TestClosureCheckpointUnresolvedBlock` | existing `HOST_FINDINGS_UNRESOLVED` |
| `TestFinalReviewSatisfiesBothE2AndE3` (if Decision D = yes) | IV exit and closure both pass with one final review |
| `TestIVStagePacketIssuance` (assumption check, §4) | packet issuance and findings import succeed at IV |

### 5.4 Compatibility regressions

The full existing suite must pass, apart from the tests that §4 lists as broken or to check. Those get a final-
review step and no assertion weakening. Also run `go test -race ./...` and the dogfood harness.

### 5.5 Schema and byte compatibility

| Test (PROPOSED) | Expected |
|---|---|
| `TestLegacyHostRecordAnchorUnchanged` | a stored pre-Gate-E dispatch record re-marshals to the same bytes and anchor |
| `TestContractDigestUnchangedWithoutPolicyDigest` | an existing contract fixture keeps its digest |
| `TestPlanDigestUnchangedWithoutMilestone` | an existing plan fixture keeps its digest |
| `TestExplicitZeroOptionalFieldRefused` | `"checkpoint_policy_digest": ""`, `null`, and `"material_milestone": false` refused |
| `TestCheckpointFieldsWithoutSchemaRefused` | binding with `family_separation: true` and no `checkpoint_schema` refused by `validatePacketBinding` |

## 6. Decisions

### Decision philosophy applied

Lennon's direction: start looser and tighten slowly, because HMA should be a trusted MVP that grows, not a
blocker. Every tie between friction and looseness therefore went to the looser option, except where the stricter
option is already enforced today, is required by current architecture text, or protects §7.1, approval binding,
or an existing independence check. That changed three r2 recommendations: **U** (unclassified paths get a
checkpoint with normal independence), **D8** (no checkpoint on untriggered units; misclassification is a named
v1 gap), and **D11** (an allowed-only effect gets a checkpoint, not high-risk treatment). It did not change C, P,
M, F, D, K, S, D7, D9, or D10, where r2 already chose the looser option; O keeps one strict element (no
catch-all `ordinary`), because without it one rule would switch §16 off by configuration. Decision G is not a
looseness choice: G-R and G-F are defects against the current spec, so only their timing is decided, and the
three decisions that narrow or depart from current text (D7, D8, D9) are flagged for explicit amendment, not
applied silently.

### Decisions

Ming made the first pass in r3. Phil's cross-family safety-boundary check on r3 is complete: U is clean, and D8
and D11 are safe for §7.1, the kernel, approval binding, and the independence code, but each narrows current
`docs/architecture.md` text. Lennon accepted both D8 and D11 as-is on 2026-10-31. All 15 decisions are now
settled. Decision D (letter) and Decisions D7–D11 are unrelated.

- **Decision C — classification basis.** (1) Path glob only. (2) Path glob plus byte-regex content rules.
  (3) Language-aware interface detection. **Decided: (1).** It is deterministic, standard-library-only, and
  cheap to audit, so it can ship now. Content rules and language-aware detection are tightening steps for when
  real misses show where they pay off. **(1) alone does not meet the literal §16 text.** An interface change
  inside a path the policy calls `ordinary` is invisible to it. Decision D8 (1) names that gap and carries the
  [CHANGES §16] amendment for it.
- **Decision P — policy carrier.** (1) A separate `hma.checkpoint-policy/v1`, bound by an optional contract
  `checkpoint_policy_digest` ([CHANGES §9]). (2) Embed the rules in the route policy projection. That needs
  [CHANGES §12.3], because §12.3 says the projection holds route rules "only". **Decided: (1).** The field is
  optional, so a project that writes no policy gives up nothing it has today. Its paths are unclassified and get
  a normal-independence checkpoint (Decision U). (2) costs an extra amendment for no reduction in friction.
- **Decision O — authority over `ordinary`.** Which layer may declare a path `ordinary`? Is a catch-all
  `**: ordinary` refused? Should kernel-reserved paths (CI config, HMA policy, contract, and plan files when
  in-repo) never be `ordinary`? **Decided: org and project only; refuse a catch-all `**: ordinary`; no kernel
  reservations in v1.** No reservations is the loose half: the kernel cannot know repository layout, and
  reservations would add refusals before any real misuse is seen. The catch-all refusal is kept because Decision
  D8 (1) defines "cannot be classified" as "no rule matches". One `**: ordinary` rule would then make T8
  unreachable and turn §16 off by configuration, not by honest mistake. Known limitation: v1 refuses the literal
  catch-all only. An equivalent set of broad globs is not detected.
- **Decision U — independence for unclassified paths.** (1) Family separation, with the floor raised to `high`
  so the outputs stay consistent (§3.4). (2) Normal (different model). **Decided: (2). Changed from r2's (1).**
  T8 still fires a mandatory checkpoint, so §16's "cannot be classified is a mandatory trigger" holds as written.
  Under (1), every run without a policy, including every in-flight contract-bound run (§4), would need a
  family-separated, high-risk review on every unit. That is the blanket friction Lennon wants to avoid.
  - *Reading of §13.* "Materially unknown environments" refers to the run environment, not to a repository path
    no rule matches. On that reading, (2) drops no current §13 automatic trigger. If Phil's pass reads §13 as
    covering unclassified paths, (1) is required and this decision reverts. Phil's r3 pass found (2)
    safe-to-loosen and the reading plausible, so the decision stands.
  - *Cost.* An unclassified high-risk path gets a same-family checkpoint reviewer unless the contract risk or
    the host risk is high, or the review is disputed. A project removes the gap by classifying the path; HMA can
    tighten later by moving to (1).
- **Decision M — material milestone.** (1) Explicit plan-unit flag ([CHANGES §10]). (2) Every unit completion.
  (3) A unit that completes a contract criterion. **Decided: (1).** T6 fires only where the plan opts in. (2)
  and (3) would put a checkpoint on most units by default.
- **Decision F — final-audit granularity.** (1) A per-unit `final` review at the closure artifact for every
  credited unit. (2) A new run-level final-review record kind. **Decided: (1).** It reuses the existing
  per-unit packet, review, and credit path, so it ships without a new record kind or import path. Cost: a
  multi-unit run needs one final review per credited unit. If that count causes friction in practice, (2) is the
  later step that reduces it.
- **Decision D — can one `final` review satisfy E2 and E3?** **Decided: yes**, when it is dispatched in
  `INDEPENDENT_VALIDATION` on the closure artifact. A single-unit triggered run then needs two reviews, not
  three. It stays separate from the per-change review, as roadmap §3 requires, so no existing independence
  check is merged away.
- **Decision L — scope.** Replaced by Decision D10. r1's recommendation ("contract-bound only, with E3 for all
  runs") contradicted itself and is withdrawn.
- **Decision R — risk floor above contract risk.** Merged into Decision D9.
- **Decision K — kernel status.** Add "unclassified ⇒ checkpoint" or "closure ⇒ final audit" to §7.1?
  **Decided: no.** §7.1 stays unchanged. They are gate refusals, so no waiver can clear them already (§3.7).
  Kernel status would add no protection and would freeze rules this MVP expects to tune as it tightens.
- **Decision S — plan file scope.** Should paths outside `allowed_files` or inside `forbidden_files` be a
  trigger (T8-like) or a separate kernel finding? **Decided: out of Gate E; track as its own item.** A
  plan-scope trigger now would add a new refusal class before the checkpoint gates have run on real work.
  Today's behavior (non-emptiness check only, §2.4) is unchanged.
- **Decision G — sequencing of G-R and G-F.** **Decided: the sequencing below. The fixes themselves are not
  loosened.** G-R and G-F are defects against current §12.4, §13, and §16 text, not new design, so the MVP
  philosophy does not apply to whether they are fixed, only to when.
  - **G-R first, as a Gate B-style defect patch.** It is live today. The patch is E0 step 4 using
    `contract.Risk` only (no floor yet) plus E0 step 5, with the red tests in §5.2. It waited only on D10's
    packet-less-review rule; D10 is now decided (1), so it is unblocked.
  - **G-F on `VERIFIED_*`: before or with Phase B.** It is latent (§2.3). E3 must land no later than the change
    that enables verified outcomes in the core.
  - **G-F on `PARTIAL`: with E3.** It is live. Its fix waited on D7; D7 is now decided (2), so it is unblocked.
- **Decision D7 — terminal-outcome coverage.** Does §16 "final outcome closure" require a final independent
  review for every terminal outcome? Paths and evidence per outcome are in §1.5 step 6 and §3.5 E3.
  - (1) All seven outcomes (the literal §16 text). Every terminal transition needs a `final` review of the live
    artifact for every implemented unit.
  - (2) Success-claiming outcomes only: `VERIFIED_SUCCESS`, `VERIFIED_WITH_WAIVERS`, and `PARTIAL`. `PARTIAL`
    needs current credit plus a `final` review for each `PASSED` criterion only.
  - (3) `VERIFIED_*` only.

  **Decided: (2).** Stopping outcomes stay as free as they are today, and the final audit lands only where a run
  claims success. (3) is looser still but would leave `PARTIAL`, the only live G-F path, unfixed, which
  Decision G rules out. (2) is therefore the loosest option that fixes the current-spec defect. `PARTIAL`
  asserts that work satisfies criteria, and that assertion is what the final audit protects. `FAILED`,
  `BLOCKED`, `UNKNOWN`, and `ABORTED` claim no success. Requiring a final review for them causes harm:
  - §13 makes `BLOCKED` the required result when an independent validator is unavailable. Requiring one to
    record `BLOCKED` deadlocks.
  - `ABORTED` is a human stop. `FAILED`, `BLOCKED`, and `UNKNOWN` already need a human-approved
    classification (`internal/engine/stage_state.go:244-245`).
  - Blocking a truthful stop works against §7.1 "prohibition on concealed failures" and §5.2 "HMA succeeds
    when it reports the state truthfully".

  **(2) narrows the literal §16 text.** It needs a [CHANGES §16] amendment that defines "final outcome closure"
  as a success-claiming terminal outcome. It is flagged here, not applied silently: Lennon accepts it with the
  amendment.
- **Decision D8 — classifier completeness under misclassification.** How does the design keep §16's
  mandatory trigger when path rules miss a semantic interface change?
  - (1) Accept Decision C (1) as is. Amend §16 so "cannot be classified" means "no rule of the approved policy
    matches". Residual: a policy that calls an interface, schema, protocol, or control-logic path `ordinary`
    loses its checkpoint **and** its family separation. Only the per-change review (different model) sees it.
  - (2) Fail closed on checkpoint existence: every contract-bound unit needs a checkpoint-grade review to leave
    `INDEPENDENT_VALIDATION`, whatever the classifier says. The classifier then decides only strength (family
    separation, risk floor) and records triggers.
  - (3) (2), plus reviewer escalation: the checkpoint report carries a closed-vocabulary list of observed
    classes. A triggering class absent from the packet's triggers is recorded at import, and E2 refuses until a
    review with the matching strength covers it. Reviewer text may only raise.

  **Decided: (1), with the gap named. Changed from r2's (2).** (2) puts a checkpoint on every contract-bound
  unit whatever its content. With Decision D = yes, that costs nothing extra on a single-unit run, but every
  earlier unit of a multi-unit run pays one more review even when nothing triggered. The only benefit is
  catching an approved policy that mislabels a path. An MVP can accept that gap if it is named.
  - *What (1) keeps.* Every coding change still gets the mandatory per-change review from a different model (E1,
    unchanged). Triggered units still get their checkpoint (the G-V fix). A run with no policy still gets a
    checkpoint on every unit through T8. (1) removes no independence check that the code enforces today.
  - *Flag: this narrows the literal §16 text.* §16 says "a path or interface that cannot be classified is a
    mandatory trigger". [CHANGES §16] must define "cannot be classified" as "no rule of the approved policy
    matches", and must state that misclassification by an approved policy is a known v1 gap.
  - *Named residual.* A policy that calls an interface, schema, protocol, or control-logic path `ordinary` loses
    that path's checkpoint and its family separation. Only the per-change review sees it. The policy is bound into
    the human-accepted contract digest (§3.2), so the error is in a human-approved document. Decision O blocks
    the catch-all form. `TestCheckpointMisclassifiedPathKnownGap` (§5.1) pins the gap.
  - *Tightening path.* (2), then (3), or Decision C (2)/(3), when a real miss shows where it pays. (3) also
    changes the report contract (`internal/host/report_contract.go`).
  - **Accepted by Lennon, 2026-10-31, as-is (no refinement).** Phil's r3 pass found D8 (1) safe for §7.1,
    approval binding, and the independence checks, but a narrowing of §16's mandatory-trigger guarantee. Lennon
    accepted that tradeoff: a policy-mislabeled path loses its extra checkpoint and keeps its per-change review.
    No move to option (2) in v1. The [CHANGES §16] amendment above is required.
- **Decision D9 — risk provenance and confirmation.** Absorbs Decision R.
  - *Provenance (no option; this is the design).* The authoritative value is `contract.Risk` from the contract
    document proven equal to `engine.BoundContractDigest(core)` at packet issuance (`cmd/hma/packetstate.go:22-38`),
    with the core head re-checked under the lock (`cmd/hma/review.go:449`). HMA detects a lower host value by
    rank comparison and refuses (E0 step 4). Later gates trust the recorded `contract_risk` and `risk_floor`
    only while `bound_contract_digest` equals the bound digest (E2, E3).
  - *Confirmation.* §13 says "the human confirms or raises" the floor. When the classifier raises the floor
    above the human-confirmed contract risk: (1) the raised floor applies at once to reviewer independence and is
    shown in every later packet, with no re-confirmation; (2) progress is refused until a human re-accepts a
    contract at the raised risk.
  - **Decided: (1).** A raised floor only strengthens reviewer independence, so applying it at once never
    weakens anything. (2) would cost a whole-run rewind per discovery. Under Decisions U (2) and D11 (2), the floor
    rises only on T3, observed T4, and T5, so the departure below is narrower than in r2.
    **Flag: (1) departs from §13's "human confirms" for raised floors.** The
    contract keeps the lower human-confirmed risk while HMA acts on the higher one. It needs a [CHANGES §13]
    sentence saying a floor HMA raises above the confirmed risk applies without re-confirmation, and only a
    downgrade needs a human (as an explicit waiver).
- **Decision D10 — scope and compatibility.** One position, replacing Decision L:
  - (1) **Contract-bound runs get all of E0, E2, and E3. Non-contract runs keep today's packet path and may not
    record a success-claiming outcome** (`VERIFIED_*`, or `PARTIAL`): PROPOSED
    `HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM`. They may still stop `FAILED`, `BLOCKED`, `UNKNOWN`, or
    `ABORTED`. Packet-less and legacy reviews never satisfy E2 or E3 on any run, and packet-less review dispatch is
    refused on contract-bound runs (E0 step 5). In-flight contract-bound runs become stricter (§4).
  - (2) All runs get E0, E2, and E3. Non-contract runs get a manifest (remove the skip at `cmd/hma/review.go:399`).
    With no contract, no policy is bound, so every path is unclassified (T8), and no contract risk exists.

  **Decided: (1).** Non-contract runs keep today's review path, and the success-claim refusal changes nothing
  they can reach today. The only new friction lands on contract-bound runs (E0 step 5), and the G-R fix needs it
  (Decision G). Credit already requires a bound contract (`cmd/hma/credit.go:95`), so a non-contract run
  cannot hold a credited `PASSED` criterion today. (1) makes that explicit and closes the vacuous
  `VERIFIED_SUCCESS` case (§2.4). G-F then cannot occur on a non-contract run, so E3 needs no non-contract
  variant. **Residual:** a non-contract run can leave `INDEPENDENT_VALIDATION` without a checkpoint, even when a
  §16 trigger would fire. That is acceptable only because it cannot end in a success claim. Choose (2) if
  non-contract runs must honor §16 checkpoints anyway.
- **Decision D11 — external-effect semantics.** HMA distinguishes three kinds of effect:
  - *allowed*: the unit's permission classes lack `no_external_effect`;
  - *observed*: a changed path is classed `external_effect` or `irreversible`;
  - *planned*: no plan field declares a planned effect in v1, so a planned effect equals an allowed one.

  Runtime effects are not observable by HMA in v1 (§2.1 T4). Options:
  - (1) Allowed and observed effects each fire T4, set family separation, and raise the floor to `high`.
  - (2) An allowed effect fires T4 (checkpoint) only, with normal independence and no floor. An observed effect
    fires T4, family separation, and a `high` floor.
  - (3) Only an observed effect fires T4.

  **Decided: (2). Changed from r2's (1).** An allowed effect fires T4 and requires a checkpoint at the IV exit,
  with normal independence and no floor raise. An observed effect fires T4, family separation, and a `high`
  floor. §3.4 is consistent with this.
  - *Tradeoff.* HMA loses automatic high-risk treatment for runtime effects it cannot observe in v1. In exchange,
    a unit that merely holds effect permission is not forced into a family-separated, high-risk review.
  - *Reading of §13.* §13 lists "external side effects" as an automatic high-risk trigger. (2) reads a
    permission as not yet an effect. §13 stays covered for effects HMA can see (observed). For runtime effects,
    it is covered through the human-confirmed contract risk: a human who expects a real effect sets
    `risk: high`, and T2 fires. Phil's r3 pass tested this reading: see the acceptance line below.
  - *Why not (3).* With no runtime observation, the allowed-effect checkpoint is §16's only signal for runtime
    effects. (3) would leave §16's "irreversible or external effect" trigger almost unreachable. That is a spec
    narrowing, not MVP looseness.
  - *Tightening path.* (1), or a plan field that declares a planned effect.
  - **Accepted by Lennon, 2026-10-31, as-is (no refinement).** Phil's r3 pass found the permission-only half
    safe, but found that the observed-only high-risk condition narrows §13, because §13 says "external side
    effects", not "observed external side effects". Lennon accepted that tradeoff: a real effect HMA never
    observes at a classified path loses automatic high-risk treatment. The only mitigation is a human who sets
    `risk: high` on the contract. No "planned effect" plan field in v1. This needs a [CHANGES §13] amendment
    stating that the automatic "external side effects" trigger covers effects HMA observes, and that unobserved
    runtime effects rely on the human-confirmed contract risk.

### All decisions settled (2026-10-31)

Every one of the 15 decisions has a final, accepted value: C (1), P (1), O (org and project only, catch-all
`ordinary` refused, no kernel reservations), U (2), M (1), F (1), D (yes), K (no), S (out of Gate E), G (G-R
first, G-F on `PARTIAL` with E3, G-F on `VERIFIED_*` before or with Phase B), D7 (2), D8 (1), D9 (1), D10 (1),
and D11 (2). Implementation needs a merged `docs/architecture.md` amendment for these sections first: **§9**
(optional `checkpoint_policy_digest`; P), **§10** (optional `material_milestone`; M), **§12.4** (derived minimum
`issuing_reason`, required review risk, packet-only contract-bound review dispatch, host-gated IV exit; D9,
D10), **§13** (classifier risk floor and no re-confirmation for a raised floor, D9; observed-effect reading of
"external side effects", D11), **§16** (T1–T8, policy format, baseline, E2 and E3, "final outcome closure" as a
success-claiming outcome, D7; "cannot be classified" as "no rule matches", D8), and **§20** (final review per
credited unit, success claims require a contract-bound run; F, D10). §3.6 has the clause summary; its §13 row
predates the D11 amendment.

## 7. Next action (§18)

The next step is to draft the actual `docs/architecture.md` amendment text for the sections listed in §6 "All
decisions settled". That is a new, separate dispatch that follows the pattern of
`docs/handoffs/2026-10-16-arch-amendment-draft.md`. It is not implementation. Implementation, including the
Decision G G-R patch, still waits until that amendment is accepted and merged, per the project order: Gate A
design, then architecture merge, then implementation.
