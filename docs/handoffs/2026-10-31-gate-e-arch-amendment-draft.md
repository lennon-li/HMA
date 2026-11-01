# Architecture amendment draft — Gate E checkpoint triggers

**Date:** 2026-10-31
**Dispatch ID:** hma-gate-e-arch-amendment-draft-20261031-r1
**Scope:** normative amendment text for `docs/architecture.md` §9, §10, §12.4, §13, §16, and §20 that carries the accepted Gate E checkpoint-trigger design (`2026-10-31-gate-e-checkpoint-triggers-draft.md`, revision 4, FINAL) into the architecture.
**Status:** DRAFT — pending independent review and Lennon's sign-off; not merged.
**Revision:** 3. Responds to Phil's re-review of revision 2
(`docs/handoffs/evidence/gate-e/audit-gate-e-arch-amendment-phil-r2.md`), PASS WITH FINDINGS; fixes N1-N2.
**Author:** Ming / Anthropic `claude-opus-5-5` / Claude Code CLI.
**Base:** `main` at `240119d`. Source design: `docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md` r4
(below: "the Gate A draft"); all 15 of its §6 decisions are settled and accepted by Lennon (2026-10-31).
Provenance: `docs/handoffs/2026-10-31-gate-e-arch-amendment-trace.md`.
**Resolves:** open question 6 of `2026-10-16-arch-amendment-draft.md` §2 ("The checkpoint trigger list is
qualitative; the deterministic detector that classifies a change as one of the listed triggers is deferred to
implementation"). This amendment defines the detector's semantics at architecture level. Its code remains
implementation work, which waits for this amendment to merge.

### Drafting conventions

- `[CHANGES §N]` marks each new or changed sentence, table row, or list item. Unmarked sentences inside a
  "Replace … with" block are existing `docs/architecture.md` text, retained verbatim.
- New field names, schema values, trigger identifiers, and refusal reasons are the Gate A draft's names, carried
  unchanged. Whether they form a stable public enum is open (§2 item 2).
- Gate A implementation names (`ClassifyCheckpoint`, `CheckpointInput`, `requireIndependentCredit`, Go field
  names, file:line citations) are intentionally not in the normative text. The trace maps each clause back to
  them.
- Gates keep the Gate A draft's labels in parentheses: checkpoint gate (E2), final gate (E3).
- Four clauses narrow current text. Each is stated as an explicit amendment where it occurs, not as a
  reinterpretation: §13 raised-floor confirmation (Decision D9), §13 observed external side effects
  (Decision D11), §16 "final outcome closure" (Decision D7), §16 "cannot be classified" (Decision D8). Two
  further clauses, both from Decision D10, add new requirements that depart from current text without narrowing
  an existing guarantee: §20 refuses a success-claiming outcome on a non-contract run (current §9 says a run
  "may use" a contract; current §5.2 sets no contract requirement for `VERIFIED_SUCCESS`), and §12.4 refuses
  packet-less review dispatch through the general host-dispatch route on a contract-bound run.

## 1. Sections to be inserted or replaced

### 1.1 §9 — Acceptance-criteria contract

Insert after the paragraph that begins "A run may use a versioned acceptance contract,
`hma.acceptance-contract/v1`." and ends "… as `acceptance_contract_digest` (see §6).", and before "A criteria
change rewinds the run to the appropriate earlier stage":

The contract may carry one optional field, `checkpoint_policy_digest`: the digest of the run's
`hma.checkpoint-policy/v1` document (§16). [CHANGES §9] It uses the same digest format as `policy_digest` and
must equal the digest of the checkpoint policy that HMA retains for the run. [CHANGES §9] When the field is
absent, the run has no checkpoint policy, and every changed path is unclassified (§16). [CHANGES §9] Absence is
the only encoding of "no policy": a contract that carries the key with an empty string or `null` is refused.
[CHANGES §9] The field is omitted when absent, so a contract without it serializes to the same bytes and keeps
its existing digest. [CHANGES §9] Because the policy digest is part of the contract, a policy change is a
contract revision: it rewinds under the existing contract rules and fires checkpoint trigger T1 (§16).
[CHANGES §9] An HMA binary that predates the field refuses a contract that carries it; one binary governs a run.
[CHANGES §9]

Retain the remaining text of §9 unchanged.

---

### 1.2 §10 — Plan-unit contract

Insert after the paragraph that begins "The approved plan is encoded as `hma.plan/v1`, a canonical JSON
document." and ends "… and the dependency graph among units.", and before "A unit's lease derives from the
unit's latest approved core transition":

A plan unit may carry one optional field, `material_milestone`. [CHANGES §10] Absent means `false`.
[CHANGES §10] When the key is present, only `true` is accepted; an explicit `false` or `null` refuses the plan,
so each meaning has one byte form. [CHANGES §10] The field is omitted when absent, so existing plans keep their
digests. [CHANGES §10] `material_milestone: true` marks the unit as an approved material milestone for
checkpoint trigger T6 (§16). [CHANGES §10]

Retain the remaining text of §10 unchanged.

---

### 1.3 §12.4 — Interactive host orchestration

Insert at the end of §12.4, after the paragraph that begins "Every change to source code, tests, executable
configuration, build logic, or CI behavior requires independent review" and ends "… and therefore cannot review
or validate that work itself.", and before "### 12.5 Advisory decision providers":

#### Checkpoint level, review risk, and checkpoint binding

This subsection applies to contract-bound runs, that is, runs with a bound `acceptance_contract_digest`.
[CHANGES §12.4] Non-contract runs keep the existing review-packet path, and their packets carry no checkpoint
binding. [CHANGES §12.4]

**Derived minimum issuing reason.** When HMA builds a review packet, it runs the checkpoint classifier (§16)
over its own complete changed-path capture and derives the minimum issuing reason. [CHANGES §12.4] The order is
`per_change` < `checkpoint` < `final`. [CHANGES §12.4] The host-requested `issuing_reason` may equal or exceed
the derived minimum and may never be below it. [CHANGES §12.4] A request below the minimum is refused
(`HOST_CHECKPOINT_REASON_TOO_LOW`), and the refusal reports the derived level. [CHANGES §12.4] HMA does not raise
the host's reason silently, consistent with the prohibition on silent substitution above. [CHANGES §12.4] The
packet header shows the checkpoint binding fields below, so the reviewer sees why the review was issued; the
packet digest covers them. [CHANGES §12.4]

**Required review risk.** The authoritative contract risk is the `risk` of the contract document that HMA has
proven equal to the bound contract digest at packet issuance, with the core head re-checked under the host
lock. [CHANGES §12.4] The required review risk is `max(contract risk, risk floor)`, where the risk floor
is the §13 floor that the checkpoint classifier supplies. [CHANGES §12.4] The host-written risk of a review
dispatch must reach the required review risk. [CHANGES §12.4] A lower value is refused
(`HOST_REVIEW_RISK_BELOW_REQUIRED`); HMA does not rewrite the host field. [CHANGES §12.4] Reviewer independence
is evaluated against that dispatch risk, so a `high` or `critical` contract risk or floor requires
provider/model-family separation (§13). [CHANGES §12.4] Independence also requires provider/model-family
separation whenever the HMA-written binding records `family_separation: true`; this covers final outcome closure
and disputed validation, which the risk value does not express. [CHANGES §12.4]

**Packet-only review dispatch.** On a contract-bound run, a review may be dispatched only through the
HMA-built review packet. [CHANGES §12.4] A packet-less review dispatch through the general host dispatch route is
refused on a contract-bound run (`HOST_REVIEW_IMPORT_REQUIRED`). [CHANGES §12.4] Packet-less and legacy reviews
never satisfy the checkpoint gate or the final gate (§16), on any run. [CHANGES §12.4] A legacy
`final_validation: true` flag does not substitute for a review whose `issuing_reason` is `final`. [CHANGES §12.4]

**Checkpoint binding.** On a contract-bound run, HMA writes the following fields into the review packet
binding — the structured binding fields on an issued review packet record — that the `review_dispatch` record
carries; the host cannot supply them. [CHANGES §12.4] Each field is omitted when absent. [CHANGES §12.4]

| Field | Absent means | Present rule |
| --- | --- | --- |
| `checkpoint_schema` | The classifier did not run (legacy or non-contract packet). The checkpoint and final gates treat the review as **not** checkpoint-grade. | Fixed value `hma.checkpoint/v1`. Set on every packet the classifier ran for. [CHANGES §12.4] |
| `minimum_issuing_reason` | — (always present with the schema) | The derived minimum. `issuing_reason` is never below it. [CHANGES §12.4] |
| `checkpoint_triggers` | No trigger fired. | Sorted, unique, non-empty list of trigger identifiers (§16). [CHANGES §12.4] |
| `unclassified_paths` | No unclassified path. | Sorted list of changed paths that cannot be classified (§16). [CHANGES §12.4] |
| `family_separation` | `false`. | `true` only. [CHANGES §12.4] |
| `risk_floor` | — (always present with the schema) | `low` or `high` (§13). [CHANGES §12.4] |
| `contract_risk` | Non-contract run. | Copied from the verified contract at issuance. [CHANGES §12.4] |
| `bound_contract_digest` | Non-contract run. | Equals the bound contract digest at issuance. [CHANGES §12.4] |
| `checkpoint_policy_digest` | No policy: every changed path was unclassified. | Equals the contract's `checkpoint_policy_digest`. [CHANGES §12.4] |
| `manifest_digest` | No manifest (non-contract run). | SHA-256 (hex) of the encoding of the sorted changed-path manifest. Present whenever a manifest was built. [CHANGES §12.4] |
| `checkpoint_covers` | — (always present with the schema) | The current contract digest (`contract_digest`) and the unit's sorted permission classes (`permission_classes`) at issuance; both required. It becomes the unit's next checkpoint baseline once the review is approved (§16). [CHANGES §12.4] |

- Every checkpoint field other than `checkpoint_schema` is interpreted only when `checkpoint_schema` is present.
  [CHANGES §12.4] A binding that carries any checkpoint field without `checkpoint_schema` is refused.
  [CHANGES §12.4]
- With the schema present, an absent `family_separation`, `checkpoint_triggers`, or `unclassified_paths` means
  `false` or empty. [CHANGES §12.4] Without the schema, the values are unknown, and the review never counts for
  the checkpoint gate or the final gate. [CHANGES §12.4]
- A binding without the new fields serializes to the same bytes as before, so existing `review_dispatch` records
  and their host record anchors are unchanged. [CHANGES §12.4] The new fields change only new packet files;
  existing packets and their digests are unchanged. [CHANGES §12.4]
- A later gate trusts the recorded `contract_risk` and `risk_floor` only while the recorded
  `bound_contract_digest` equals the contract digest bound at that gate. [CHANGES §12.4]
- Permission classes are the portable interaction-envelope classes of the route policy: `read_only`,
  `bounded_write`, and `no_external_effect`. [CHANGES §12.4]

**Host-gated independent-validation exit.** `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE` is a host-gated
edge: one requiring the host's deterministic approval, not merely a human confirmation. [CHANGES §12.4] In
addition to human approval and the legal-edge check, it requires the checkpoint gate (§16) to pass whenever any
checkpoint trigger fires for the unit. [CHANGES §12.4] The per-change independent
review required above is unchanged; classification never removes it. [CHANGES §12.4]

Retain the remaining text of §12.4 unchanged.

---

### 1.4 §13 — Risk and validator independence

Replace the first two paragraphs of §13 — the paragraph "HMA computes a deterministic minimum risk floor. …
safety-kernel failures remain unwaivable." and the paragraph "Automatic high-risk triggers include
security/authentication, … and materially unknown environments." — with the text below. Two existing sentences
change wording; both changes are tagged and listed in "Changed wording" at the end of this subsection.

HMA computes a deterministic minimum risk floor. Organization and project policy may raise it. The human
confirms or raises it, except that a floor HMA raises above the human-confirmed contract risk applies as stated
below. [CHANGES §13] A permitted downgrade must be an explicit waiver; safety-kernel failures remain
unwaivable.

The checkpoint classifier (§16) supplies this deterministic minimum risk floor. [CHANGES §13] The floor is
`high` when a changed path is classed as a public interface, persisted schema, or protocol (trigger T3); as
security, approval, routing, evidence, or closure logic (trigger T5); or as an observed external or irreversible
effect (trigger T4, observed form). [CHANGES §13] Otherwise the floor is `low`. [CHANGES §13] An unclassified
path (trigger T8) and an allowed-only effect (trigger T4, allowed form) do not raise the floor. [CHANGES §13] The
floor only raises; it never lowers the contract risk or the host-written review risk. [CHANGES §13]

A floor that HMA raises above the human-confirmed contract risk applies at once to reviewer independence and is
shown in every later review packet. [CHANGES §13] It does not require human re-confirmation. [CHANGES §13] The
contract keeps the lower human-confirmed risk while HMA acts on the higher floor. [CHANGES §13] Only a downgrade
needs a human, as an explicit waiver. [CHANGES §13] This amends "the human confirms or raises it" for raised
floors only; a raised floor can only strengthen reviewer independence. [CHANGES §13]

Automatic high-risk triggers include security/authentication, secrets/privacy, destructive operations,
production/infrastructure, publication, schema migration, public API or architecture changes, scientific
assumption changes, broad diffs, missing tests, observed external side effects, waived correctness criteria,
conflicting validator findings, and materially unknown environments. [CHANGES §13]

An external side effect is observed when the approved checkpoint policy (§16) classes a changed path
`external_effect` or `irreversible`. [CHANGES §13] A unit's permission to cause an external effect, with no
observed effect, is not an automatic high-risk trigger; it fires a §16 checkpoint with normal independence.
[CHANGES §13] HMA does not observe runtime effects in v1. [CHANGES §13] A runtime effect that HMA does not
observe at a classified path relies on the human-confirmed contract risk: a human who expects a real external
effect sets the contract risk to `high` or `critical`. [CHANGES §13] This narrows the previous unqualified
"external side effects" trigger and is an accepted v1 limitation. [CHANGES §13]

Changed wording (for the merger):

- "The human confirms or raises it." becomes "The human confirms or raises it, except that a floor HMA raises
  above the human-confirmed contract risk applies as stated below."
- In the automatic high-risk trigger list, "external side effects" becomes "observed external side effects".
  No other item in that list changes.

Retain the remaining text of §13 unchanged.

---

### 1.5 §16 — Independent validation contract

Two edits.

**Edit A.** Insert immediately after the paragraph under "#### Checkpoint triggers" that begins "Mandatory
independent validation checkpoints are triggered by:" and ends "… (provider and model-family separation).", and
before "Each finding must cite the exact criterion or invariant":

For these triggers, "final outcome closure" means recording a success-claiming terminal outcome:
`VERIFIED_SUCCESS`, `VERIFIED_WITH_WAIVERS`, or `PARTIAL`. [CHANGES §16] `FAILED`, `BLOCKED`, `UNKNOWN`, and
`ABORTED` claim no success. [CHANGES §16] They do not fire the final-outcome-closure trigger and keep their
existing requirements. [CHANGES §16] This narrows the earlier unqualified text, under which every terminal
outcome was final outcome closure. [CHANGES §16]

A path "cannot be classified" when no rule of the approved checkpoint policy matches it, or when the run has no
checkpoint policy. [CHANGES §16] A path that a rule of the approved policy matches is classified, including a
path that the policy classes `ordinary`. [CHANGES §16] Known v1 gap: an approved policy that classes an
interface, schema, protocol, or control-logic path `ordinary` removes that path's checkpoint and its
provider/model-family separation; only the mandatory per-change review (§12.4) sees the change. [CHANGES §16]
The policy is bound into the human-accepted contract digest (§9), so such an error is in a human-approved
document. [CHANGES §16]

**Edit B.** Append at the end of §16, after the paragraph "The validator may not edit code, evidence, or state;
… or continue auditing after every criterion is resolved.":

#### Checkpoint policy

The checkpoint policy is `hma.checkpoint-policy/v1`, a canonical JSON document parsed with the same strict
decoding as the acceptance contract and the plan. [CHANGES §16] The contract's `checkpoint_policy_digest` binds
its digest (§9). [CHANGES §16] The document holds a `schema` value and a list of `rules`; each rule has an `id`,
a path `glob`, and a list of `classes`. [CHANGES §16] Every policy field is required. [CHANGES §16]

The class vocabulary is closed: `public_interface`, `persisted_schema`, `protocol`, `external_effect`,
`irreversible`, `security`, `approval`, `routing`, `evidence`, `closure`, and `ordinary`. [CHANGES §16] An
unknown class refuses the document. [CHANGES §16]

A glob is a slash-separated path pattern: `*` matches one path segment, and `**` matches any number of segments.
[CHANGES §16] In v1, rules match paths only; they do not match file content and do not detect interfaces by
language. [CHANGES §16] A changed path takes the union of the classes of every rule that matches it.
[CHANGES §16] A path with any class other than `ordinary` is triggering; `ordinary` never cancels another class.
[CHANGES §16] A path that no rule matches is unclassified. [CHANGES §16] With no policy, every changed path is
unclassified. [CHANGES §16]

Only organization and project policy may declare a path `ordinary`. [CHANGES §16] This authority restriction is
not mechanically enforced in v1; HMA cannot currently attribute a policy rule to the layer that authored it.
[CHANGES §16] A rule whose glob is `**` and whose only class is `ordinary` refuses the document, because it
would make trigger T8 unreachable by configuration. [CHANGES §16] v1 refuses that literal catch-all only; an
equivalent set of broad globs, and a `**` rule with additional classes beyond `ordinary`, are not detected.
[CHANGES §16] v1 defines no kernel-reserved paths that policy may not class `ordinary`. [CHANGES §16]

#### Checkpoint classifier

HMA classifies a unit with a pure, deterministic function. [CHANGES §16] Its inputs are the bound contract
document, the checkpoint policy, the plan unit, the unit's checkpoint baseline and current state, HMA's own
complete changed-path manifest, the unit's projected stage, the target terminal outcome when one is requested,
the host-written review risk, and the disputed flag. [CHANGES §16] Each manifest entry projects the path,
status, old and new mode, and binary flag. [CHANGES §16] A rename is classified as both its old path and its new
path. [CHANGES §16] The output is sorted and does not depend on input order. [CHANGES §16] The classifier runs
only on contract-bound runs. [CHANGES §16]

| Trigger | Recorded identifier | Fires when |
| --- | --- | --- |
| T1 | `CONTRACT_OR_PERMISSION_CHANGED` | The current bound contract digest differs from the baseline contract digest, or the unit's current permission classes differ from the baseline's as sets. [CHANGES §16] |
| T2 | `HIGH_OR_CRITICAL_RISK` | `max(contract risk, host-written review risk, risk floor)` is `high` or `critical`. [CHANGES §16] |
| T3 | `INTERFACE_SCHEMA_OR_PROTOCOL` | Any changed path has class `public_interface`, `persisted_schema`, or `protocol`. [CHANGES §16] |
| T4 | `IRREVERSIBLE_OR_EXTERNAL_EFFECT` | Allowed form: the unit's permission classes lack `no_external_effect`. Observed form: any changed path has class `external_effect` or `irreversible`. [CHANGES §16] |
| T5 | `SECURITY_APPROVAL_ROUTING_EVIDENCE_OR_CLOSURE` | Any changed path has class `security`, `approval`, `routing`, `evidence`, or `closure`. [CHANGES §16] |
| T6 | `APPROVED_MATERIAL_MILESTONE` | The unit has `material_milestone: true` (§10) and the unit's stage is `INDEPENDENT_VALIDATION`. [CHANGES §16] |
| T7 | `FINAL_OUTCOME_CLOSURE` | The target outcome is a success-claiming terminal outcome (above), or the unit's stage is `RELEASE_AND_CLOSURE`. [CHANGES §16] |
| T8 | `UNCLASSIFIED` | Any changed path cannot be classified (above). [CHANGES §16] |

**Baseline.** A unit's checkpoint baseline is the `checkpoint_covers` value (§12.4) recorded on the unit's
latest approved review whose `issuing_reason` is `checkpoint` or `final`. [CHANGES §16] With no such review, the baseline
is the contract digest of the run's first contract acceptance and the unit's permission classes in its first
lease generation's plan. [CHANGES §16] The first contract therefore does not fire T1 by itself. [CHANGES §16] Any
re-acceptance fires T1 until a checkpoint-grade review passes on it. [CHANGES §16]

**Level.** The derived level is `final` when T7 fires. [CHANGES §16] Otherwise it is `checkpoint` when the unit's
stage is `INDEPENDENT_VALIDATION` and any trigger fires. [CHANGES §16] Otherwise it is `per_change`.
[CHANGES §16] A contract-bound unit with no trigger needs no checkpoint review. [CHANGES §16]

**Risk.** The risk floor is defined in §13. [CHANGES §16] The required review risk is
`max(contract risk, risk floor)` (§12.4). [CHANGES §16]

**Family separation.** Provider/model-family separation is required when
`max(contract risk, host-written review risk, risk floor)` is `high` or `critical`, when the validation is
disputed, or when T7 fires. [CHANGES §16] T3, observed T4, and T5 require it through the floor. [CHANGES §16]
T7 requires it without raising risk, because §13 requires family separation for final validation, not high risk.
[CHANGES §16] T1, T6, T8, and allowed-form T4, alone, require a checkpoint with §13 normal independence
(different model), not family separation. [CHANGES §16]

**Trust in derived values.** A derived value that HMA persists is trusted at a later gate only when it names the
contract digest bound at that gate. [CHANGES §16]

#### Checkpoint and final gates

The checkpoint gate and the final gate apply to contract-bound runs. [CHANGES §16]

**Checkpoint gate (E2).** On `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE`, HMA recomputes the classification
at gate time on the live target. [CHANGES §16] When any trigger fires, the unit's latest imported review must
meet all of the following: [CHANGES §16]

- it is `APPROVE` and `isolated`, and its findings snapshot has been applied with no unresolved `BLOCK` or
  `WAIVABLE` finding; [CHANGES §16]
- its binding has `checkpoint_schema` present, and its `issuing_reason` is `checkpoint` or `final`;
  [CHANGES §16]
- it was dispatched after the unit's approved `VERIFICATION -> INDEPENDENT_VALIDATION` transition, so the
  per-change review cannot stand in for the stage audit; [CHANGES §16]
- its recorded `checkpoint_triggers` are a superset of the gate-time triggers, and its recorded
  `checkpoint_policy_digest` equals the current one; [CHANGES §16]
- its recorded `bound_contract_digest` equals the contract digest bound at gate time, and its dispatch risk is at
  least the gate-time required review risk, so a contract re-accepted after dispatch makes the review stale;
  [CHANGES §16]
- its artifact equals the transition's artifact. [CHANGES §16]

The refusal reasons are `HOST_CHECKPOINT_REVIEW_MISSING`, `HOST_CHECKPOINT_TRIGGERS_UNCOVERED`, and
`HOST_CHECKPOINT_REVIEW_STALE`. [CHANGES §16] When no trigger fires, the checkpoint gate passes with no checkpoint
review; the per-change review and the other existing gates still apply. [CHANGES §16]

**Final gate (E3).** Before HMA records a success-claiming terminal outcome, every review that a criterion
credit relies on must have `checkpoint_schema` present, must have `issuing_reason` equal to `final`, and must
have a recorded `bound_contract_digest` equal to the currently bound contract digest. [CHANGES §16] The refusal reason
is `HOST_FINAL_AUDIT_MISSING`. [CHANGES §16] Provider/model-family separation and `isolated` review for a
`final` review, and the artifact match for credit, are existing requirements and are not repeated by this gate.
[CHANGES §16]

- For `VERIFIED_SUCCESS` and `VERIFIED_WITH_WAIVERS`, the final gate is part of the closure credit check
  (§20). [CHANGES §16]
- For `PARTIAL`, HMA snapshots the artifact, re-checks credit for every `PASSED` criterion, and applies the
  final gate to the reviews those credits rely on. [CHANGES §16] `WAIVED` criteria are a human operation and
  need nothing more. [CHANGES §16] The no-active-waiver rule does not apply to `PARTIAL`. [CHANGES §16]
- `FAILED`, `BLOCKED`, `UNKNOWN`, and `ABORTED` do not run the final gate. [CHANGES §16]

One `final` review dispatched in `INDEPENDENT_VALIDATION` on the closure artifact may satisfy both the
checkpoint gate and the final gate. [CHANGES §16] It remains separate from the per-change review.
[CHANGES §16] Because a unit's latest review supersedes its earlier reviews, credit must be re-issued after each
newer review, in packet mode. [CHANGES §16]

**Non-contract runs.** The checkpoint gate and the final gate do not apply to non-contract runs, which may not
record a success-claiming outcome (§20). [CHANGES §16] Known residual: a non-contract run can leave
`INDEPENDENT_VALIDATION` without a checkpoint, even when a trigger would fire. [CHANGES §16] This is accepted
only because such a run cannot end in a success claim. [CHANGES §16]

**Gate status.** A missing or stale checkpoint or final review is a gate refusal, not a finding. [CHANGES §16]
It is outside the §7.2 waivable classes, and no waiver clears it. [CHANGES §16]

Retain the remaining text of §16 unchanged.

---

### 1.6 §20 — Closure semantics

In the bulleted list under "Closure is allowed when:", change the terminal period of the last bullet ("… the
latest findings snapshot, if any, has been applied and no unresolved `BLOCK` or `WAIVABLE` finding remains.") to
a semicolon, and append:

- the run is contract-bound; [CHANGES §20]
- every credited unit has a review of the closure artifact whose `issuing_reason` is `final` that passes the
  final gate (§16). [CHANGES §20]

Insert after "Once these conditions hold, auditing stops. Potential improvement alone is not a blocker." and
before "## 21. CI release gate":

A success-claiming terminal outcome (`VERIFIED_SUCCESS`, `VERIFIED_WITH_WAIVERS`, or `PARTIAL`) requires a
contract-bound run. [CHANGES §20] On a non-contract run it is refused
(`HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM`). [CHANGES §20] A non-contract run may still stop as `FAILED`,
`BLOCKED`, `UNKNOWN`, or `ABORTED`. [CHANGES §20] `PARTIAL` does not meet the closure list above, but it also
requires the final gate (§16) for every review its `PASSED` criteria rely on. [CHANGES §20]

Retain the remaining text of §20 unchanged.

---

### 1.7 Sections deliberately not amended

- **§5.2.** The Gate A draft's §3.6 row "§5.2 / §16" places Decision D7. Its §6 "All decisions settled" list
  names §16 only. The definition of "final outcome closure" is therefore in §16 (Edit A). The §5.2 terminal-outcome
  list is unchanged.
- **§7.1.** Decision K: no kernel change. The checkpoint and final gates are gate refusals, not findings
  (§16 "Gate status").
- **§12.3.** Decision P (1) keeps the policy out of the route policy projection, so §12.3 needs no change. The
  Gate A draft's `[CHANGES §12.3]` belongs to the rejected option (2).
- **§14.** No Gate A clause is tagged for §14. The checkpoint fields are packet bindings, already within §14
  "bindings"; their contract is in §12.4.
- **§18.** See §2 item 10.
- **Decision S** (plan file scope) is out of Gate E. **Decision G** (G-R first; G-F on `PARTIAL` with the final
  gate; G-F on `VERIFIED_*` before or with Phase B) is implementation sequencing, not architecture text. It
  carries one constraint for implementers: the final gate must be in force no later than the change that enables
  `VERIFIED_*` outcomes in the core.

## 2. Open questions

This amendment does not resolve the following, except item 8, which this revision resolves and keeps numbered
here for continuity with the trace file. The Gate A draft leaves each of the others open or silent.

1. **Policy schema.** The exact JSON Schema for `hma.checkpoint-policy/v1`: field order, canonical
   normalization, whether a rule's `classes` may be empty, and whether rule `id` values must be unique.
2. **Stable identifiers.** Whether the new refusal reasons (`HOST_CHECKPOINT_REASON_TOO_LOW`,
   `HOST_REVIEW_RISK_BELOW_REQUIRED`, `HOST_CHECKPOINT_REVIEW_MISSING`, `HOST_CHECKPOINT_TRIGGERS_UNCOVERED`,
   `HOST_CHECKPOINT_REVIEW_STALE`, `HOST_FINAL_AUDIT_MISSING`, `HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM`), the
   eight trigger identifiers, and the `hma.checkpoint/v1` schema value form a stable public enum. The Gate A
   draft marks all of them PROPOSED.
3. **`permission_classes` ordering.** The Gate A draft says `checkpoint_covers.permission_classes` is sorted, and
   T1 compares permission classes as sets. Whether the sort order is part of the normative byte form (packet
   digest, record anchor) or an implementation detail is not stated.
4. **Manifest digest byte form.** The Gate A draft defines `manifest_digest` as SHA-256 hex of `json.Marshal` of
   the sorted changed-path list. It gives that projection no JSON field names and no sort key. The exact encoding
   is therefore open.
5. **Glob edge cases.** Whether `**` matches zero segments, and how leading slashes, dot segments, case, and
   symlinked paths match.
6. **Policy retention.** How the host presents the policy document and how HMA retains it so that it can check
   "equals the retained policy's digest". The Gate A draft names the retained policy but not the mechanism (the
   plan's present-and-recompute pattern in §10 is the nearest analogue).
7. **Layer attribution for `ordinary`.** Decision O allows only organization and project policy to declare
   `ordinary`. The policy is one document bound by the contract. How HMA attributes a rule to a policy layer is
   not specified.
8. **Meaning of "level" — resolved.** The baseline and E2/E3 rules now say a review "whose `issuing_reason` is
   `checkpoint` or `final`", not the derived `minimum_issuing_reason`. This is confirmed by Decision D's
   reachability requirement: a single review whose `issuing_reason` is `final` dispatched in
   `INDEPENDENT_VALIDATION` must satisfy
   both the checkpoint gate and the final gate, which is only possible if `issuing_reason` is what is being
   checked, since `minimum_issuing_reason` at that stage is only `checkpoint`.
9. **Downgrade waiver path.** §13 keeps "only a downgrade needs a human, as an explicit waiver". The Gate A draft
   designs no waiver operation that lets a review dispatch fall below the required review risk. Until one is
   designed, the `HOST_REVIEW_RISK_BELOW_REQUIRED` refusal has no waiver path.
10. **§18 next actions.** The Gate A draft §3.5 proposes host next actions `ISSUE_CHECKPOINT_REVIEW` and
    `ISSUE_FINAL_REVIEW`, but tags no `[CHANGES §18]` and §6 does not list §18. Whether §18 needs text is open.
11. **Decision U reading of §13.** Decision U relies on reading §13 "materially unknown environments" as the run
    environment, not a repository path that no rule matches. Phil's r3 pass found the reading plausible. No §13
    wording change was settled, so this draft adds none. A reviewer may judge that a clarifying sentence is
    needed.
12. **Unverified implementation assumptions.** Packet issuance and findings import during
    `INDEPENDENT_VALIDATION`, a `unit_id` on every `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE` approval, and
    the byte-compatibility claims (Gate A §5.5) are not yet proven. They are not architecture text; the Gate E
    red tests must confirm them.
13. **Carried from the 2026-10-16 draft.** Open questions 1–5 and 7 of `2026-10-16-arch-amendment-draft.md` §2
    are unaffected. Only open question 6 is resolved here.

## Next action (§18)

Independent cross-family review of this draft (a non-Anthropic reviewer) against two references:
(a) `docs/architecture.md`, for unintended text loss, contradiction, or an unclear insertion point; and
(b) the Gate A draft, confirming that every `[CHANGES §N]` tag in its §3 (and the §6 decisions D7–D11 that
require amendment text) is represented here and that nothing is added beyond it. Then Lennon's sign-off. Then
the merge into `docs/architecture.md`, as a separate step. This draft does not merge itself.

## Status and acceptance record (2026-10-31)

- Merged into `docs/architecture.md` at commit `pending commit, worktree change only` (pre-merge HEAD:
  `240119d`). All six named section edits (§9, §10, §12.4, §13, §16, §20) applied.
- Insertion points resolved by the merger's own judgment: none. Every anchor the draft specified ("Insert
  after …", "Replace … with …") matched the then-current `docs/architecture.md` text exactly, including the
  two-edit split of §16 and the end-of-§12.4 insertion before §12.5.
- Independent review of the amendment text: Phil (OpenAI `gpt-6-luna` via GitHub Copilot CLI), two rounds,
  both PASS WITH FINDINGS, all findings fixed by revision 3.
- Accepted by Lennon, 2026-10-31 ("go ahead").
- Next action: the merge itself has not yet been independently reviewed. Independent review of the merge
  (confirming every `[CHANGES §N]` clause landed verbatim at its specified point, tags stripped, no retained
  text altered beyond the two tagged §13 wording changes, and no §16/§5.2/§7.1/§12.3/§14/§18 cross-reference
  broken) is a separate, not-yet-dispatched step, to run before this change is committed.
