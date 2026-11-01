# Gate E architecture-amendment draft — independent cross-family review

**Verdict: PASS WITH FINDINGS.** No BLOCKING finding. No new narrowing beyond the four Gate A narrowings.
Six MINOR findings and five NOTEs. Fix the six MINOR items before Lennon's sign-off (list at the end).

- **Date:** 2026-10-31
- **Reviewer:** Phil / OpenAI `gpt-6-luna` / GitHub Copilot CLI, verification mode. Cross-family to the author
  (Ming / Anthropic).
- **Repository state:** `main` at `240119d`. The three reviewed files are untracked.
- **Reviewed (read-only inputs):**
  1. `docs/handoffs/2026-10-31-gate-e-arch-amendment-draft.md` ("the amendment")
  2. `docs/handoffs/2026-10-31-gate-e-arch-amendment-trace.md` ("the trace")
- **References checked against:**
  3. `docs/architecture.md` (live text, 48 289 bytes)
  4. `docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md`, **revision 4, FINAL** ("Gate A")
- **Limits:** The shared-memory startup documents were not readable in this session (permission denied). I did
  not run code. No Jev tool was exposed. I wrote only this file.

## 1. Answer first

The amendment carries every Gate A `[CHANGES §N]` tag that targets §9, §10, §12.4, §13, §16, or §20. It carries
D7, D8, D9, D10, D11 and the other settled decisions with correct substance. All six insertion/replace anchors
match the live file. I found no invented normative rule. I found six MINOR drafting defects and five NOTEs.

## 2. Findings

Severity key: BLOCKING / NARROWING-NEEDS-SIGNOFF / MINOR / NOTE. There are no BLOCKING and no
NARROWING-NEEDS-SIGNOFF findings.

### F1 — MINOR — the "four narrowings" count is incomplete (amendment header, "Drafting conventions")

- **Observed.** The header says "Four clauses narrow or depart from current text" and lists §13 raised floor (D9),
  §13 observed effects (D11), §16 "final outcome closure" (D7), §16 "cannot be classified" (D8).
- **Reference.** Gate A §3.7 lists the same four. Gate A §6 D10 (1) and §3.6 §20 row add two more departures from
  live text: (a) §20 now refuses a success-claiming outcome on a non-contract run; live §9 says a run "may use" a
  contract, and live §5.2 sets no contract requirement for `VERIFIED_SUCCESS`; (b) §12.4 now refuses packet-less
  review dispatch on contract-bound runs.
- **Assessment.** Both are stricter than live text, not looser. Lennon accepted D10 (1) in Gate A. The substance is
  settled. Only the disclosure is incomplete. A signer who relies on "four" is under-informed.
- **Change.** List the D10 departures in the header, or reword to "four narrowings and the following
  strengthenings".

### F2 — MINOR — referent break in §20 (amendment §1.6)

- **Observed.** The amendment inserts a new paragraph "after the bulleted list and before 'Once these conditions
  hold, auditing stops.'"
- **Reference.** Live §20 lines 748–762: "Once these conditions hold, auditing stops." refers to the list directly
  above it. After insertion, "these conditions" follows a paragraph that is not a condition list. The inserted
  paragraph also says `PARTIAL` "does not meet the closure list above", which makes the referent more confusing.
- **Change.** Insert the new paragraph after "Once these conditions hold, auditing stops. Potential improvement
  alone is not a blocker." instead, or begin it with an explicit pointer to the list.

### F3 — MINOR — vocabulary used normatively but not defined in `docs/architecture.md` (amendment §1.3, §1.5)

- **Observed.** Normative text relies on: `no_external_effect` (T4 allowed form), a unit's `permission_classes`
  (T1, `checkpoint_covers`), "host-gated edge" (§12.4 exit), and "review packet binding" (§12.4 table).
- **Reference.** A search of live `architecture.md` finds no `no_external_effect`, no `permission_classes`, and
  no "host gate". Live §12.2 uses "permission classes" only for route attestations. Live §10's plan-unit field
  list has no permission-classes field. Live §12.4/§14 say "bindings", not "packet binding". Gate A §1.6 defines
  the permission vocabulary only by code citation (`internal/model/route_policy.go:48-58`).
- **Change.** Add one defining sentence for the permission-class vocabulary and the unit field, and define
  "host-gated" in the §12.4 text. Otherwise the merged architecture depends on a name that only the Go code
  defines.

### F4 — MINOR — unsupported specificity "per-run lock" (amendment §1.3 "Required review risk")

- **Observed.** "…with the core head re-checked under the host per-run lock."
- **Reference.** Gate A §3.5 E0 step 4 and D9 say "under the lock" (`cmd/hma/review.go:449`). No Gate A text, and
  no trace row, says "per-run". The live architecture has no lock concept.
- **Change.** Write "under the host lock", or "while the core head is held stable". Do not add "per-run" without a
  Gate A basis.

### F5 — MINOR — two normative sentences are looser or less testable than their source (amendment §1.5)

- **(a) Ordinary authority.** "Only organization and project policy may declare a path `ordinary`." Gate A D10
  Decision O says the same, but the amendment itself records (§2 item 7) that HMA has no way to attribute a rule
  to a layer. As merged normative text, the sentence is unenforceable and untestable. Mark it explicitly as a
  v1 limitation ("not enforced in v1") or keep it as an open question only.
- **(b) Catch-all wording.** "A catch-all rule that classes `**` as `ordinary` refuses the document." Gate A's
  test fixture is the single-class rule `{"glob":"**","classes":["ordinary"]}`. The amendment does not say
  whether `**` with `["ordinary","security"]` is refused. State the literal form: "a rule whose glob is `**`
  and whose only class is `ordinary`".
- **(c) Re-issue qualifier.** "Because a unit's latest review supersedes its earlier reviews, credit must be
  re-issued after each newer review." Gate A §3.5 "Review count" scopes this to packet mode
  (`AnswersLatestReviewDispatch`). The amendment drops "in packet mode". Restore it. Related: "part of the
  existing closure check (§20)" points at §20 prose that contains no credit check. The "existing" check is in
  code, not in the architecture. Reword to "the closure credit check".

### F6 — MINOR — `level` reading must be stated, not left open (amendment §2 item 8; §1.5 Baseline, E2, E3)

- **Observed.** The amendment writes "issued at level `checkpoint` or `final`" and asks a reviewer to confirm that
  this means `issuing_reason`.
- **Confirmation.** Confirmed. The only reading consistent with Gate A Decision D is `issuing_reason`. Under
  Decision D, one `final` review dispatched in `INDEPENDENT_VALIDATION` satisfies E2 and E3. At that stage the
  derived `minimum_issuing_reason` is only `checkpoint` (T7 needs a target outcome or stage
  `RELEASE_AND_CLOSURE`). If "level" meant `minimum_issuing_reason`, such a review would never count as `final`
  and Decision D would be unreachable.
- **Change.** Replace "issued at level X" with "whose `issuing_reason` is X" in the baseline rule, E2 and E3, and
  remove open question 8 or mark it resolved.

### F7 — NOTE — `PARTIAL` is now gated more strictly than the live report-truthfully rule (amendment §1.5 E3, §1.6)

- **Observed.** Before recording `PARTIAL`, every review a `PASSED` criterion relies on must be `final`-level.
- **Reference.** Live §5.2: `PARTIAL` is legal from any nonterminal stage, and "HMA succeeds when it reports the
  state truthfully". Gate A D7 argues that gating `BLOCKED` deadlocks and cites §7.1 "prohibition on concealed
  failures". The same argument applies in part to `PARTIAL`: if no family-separated reviewer is available, the
  run cannot record `PARTIAL`. The run can still record `FAILED`, `BLOCKED`, `UNKNOWN`, or `ABORTED`.
- **Assessment.** This is Gate A D7 (2) as accepted, so it is faithful. It is stricter, not looser. I raise it so
  that Lennon signs it knowingly. Recommend a sentence in the amendment that the truthful stop remains available
  as `BLOCKED`.

### F8 — NOTE — Decision U reading (amendment §2 item 11)

My r3 pass found the reading of "materially unknown environments" as run-environment uncertainty plausible
(`audit-gate-e-checkpoint-draft-phil.md`, "Revision 3 safety-boundary check", U). No §13 wording change is
needed. The amendment's decision to add none is correct.

### F9 — NOTE — trigger and gate labels in normative text

The amendment uses `T1`–`T8` in §9, §10, §13 and §16, and "(E2)"/"(E3)" in §16. The `T` labels are defined in
the §16 table, but §9, §10 and §13 reference them before that table. Open question 2 treats identifiers as
unstable. Acceptable for a draft. The merger should confirm the labels are intended to be stable.

### F10 — NOTE — pre-existing text vs Gate A (not an amendment defect)

Live §12.4 says the `review_dispatch` record binds "required reviewer independence". Gate A §1.4 says this is
not a stored field and is recomputed at dispatch. The amendment does not touch or reconcile the sentence. It is
not caused by the amendment, so I do not require a change.

### F11 — NOTE — policy retention (amendment §1.1, open question 6)

"Must equal the digest of the checkpoint policy that HMA retains for the run" has no retention mechanism in Gate A
or the amendment. The amendment declares this open. The requirement is not testable until it is closed.

## 3. Anchor check against live `docs/architecture.md`

| Instruction | Live location | Result |
|---|---|---|
| §1.1 insert after "A run may use … (see §6)." before "A criteria change rewinds" | §9, lines 287–291 | Match. |
| §1.2 insert after "The approved plan is encoded … among units." before "A unit's lease derives" | §10, lines 318–320 | Match. |
| §1.3 insert after "Every change to source code … validate that work itself." before `### 12.5` | §12.4, lines 506–517 | Match. Live text wraps lines; wording equal. |
| §1.4 replace first two §13 paragraphs | §13, lines 559–561 | Match. Both retained sentences verbatim, except the two tagged changes. |
| §1.5 Edit A insert after "Mandatory … (provider and model-family separation)." before "Each finding must cite" | §16, lines 658–660 | Match. |
| §1.5 Edit B append after "The validator may not edit … resolved." | §16, line 666 | Match. |
| §1.6 semicolon on last bullet "the latest findings snapshot …" | §20, lines 748–757 | Match. See F2 for the later insertion point. |

No quoted anchor mismatches the live file. I found no silent deletion of live normative text. The "Changed
wording" list for §13 is complete and accurate.

## 4. Gate A tag and decision coverage

| Gate A item | Amendment location | Result |
|---|---|---|
| §3.2 carrier `[CHANGES §9]` (line 273) | §1.1 | Covered. |
| §3.3 `[CHANGES §10]` (line 291) | §1.2 | Covered. |
| §3.4 RiskFloor `[CHANGES §13]` (line 377) | §1.4 | Covered. |
| §3.5 E2 `[CHANGES §16][§12.4]` (line 448) | §1.5 E2; §1.3 exit | Covered. |
| §3.5 E3 `[CHANGES §20][§16]` (line 460) | §1.5 E3; §1.6 | Covered. |
| §3.6 rows §5.2/§16, §9, §10, §12.4, §13, §16, §20 | §1.1–§1.6; §1.7 for §5.2 | Covered. §5.2 correctly left unamended. |
| §3.7 D7, D8, D9, D11 bullets | §1.5 Edit A; §1.4 | Covered. |
| D (letter): one `final` review satisfies E2 and E3 | §1.5 "One `final` review…" | Covered. |
| D7 (2): success-claiming outcomes only | §1.5 Edit A, E3; §1.6 | Covered. Substance correct. |
| D8 (1): "no rule matches" + named gap | §1.5 Edit A | Covered. Gap wording matches. |
| D9 (1): raised floor, no re-confirmation; provenance | §1.4; §1.3 | Covered. Provenance rule present. |
| D10 (1): contract-bound scope; success refusal; packet-less refusal; residual | §1.3, §1.5, §1.6 | Covered. See F1 for disclosure. |
| D11 (2): allowed vs observed; §13 wording change | §1.4; §1.5 T4 | Covered. Wording change listed. |
| C (1), P (1), M (1), F (1), O, U (2) | §1.5, §1.1, §1.2, §1.6, §1.5 | Covered. |
| K (no kernel change), S (out), G (sequencing), P (2) rejected `[CHANGES §12.3]` | §1.7 | Correctly not amended. |
| §3.5 "Next actions" `ISSUE_*_REVIEW` (no `[CHANGES §18]` tag) | §2 item 10 | Correctly deferred. |

I found nothing in the amendment that is more specific than Gate A without a clear basis, except F4 ("per-run")
and F5(b) (wording of the catch-all). No Gate A decision is carried with changed substance.

## 5. Trace spot-check

I checked 26 trace rows against the cited Gate A text. All 26 are supported. Rows checked: §9 carrier;
§9 one-byte-form; §9 rewind; §10 `material_milestone` and one-byte-form; §12.4 level order; §12.4 host may
only raise; §12.4 packet-less refusal and existing reason; §12.4 packet header; §12.4 authoritative risk;
§12.4 `unclassified_paths` omitted bound; §12.4 `checkpoint_covers`; §12.4 schema-gating and the
`TestCheckpointFieldsWithoutSchemaRefused` citation (Gate A §5.5, line 680); §12.4 host-gated exit and line
448; §13 raised floor and D9 "Flag"; §13 D11 wording change; §16 Edit A D7 and D8; §16 policy catch-all and
the §5.1 test citations; §16 rename (Gate A §1.6 and §5.1 line 618); §16 baseline; §16 E2 bullets and
`TestCheckpointNotRequiredWithoutTriggers` (line 636); §16 E3 `PARTIAL` bullet; §16 one-review-both-gates;
§20 bullets and `TestNonContractRunCannotClaimSuccess` (line 659). The tag-coverage table's line numbers
(273, 291, 377, 448, 460, 521–527, 539–547, 710–713, 736, 787, 810, 821, 836, 884) all match Gate A.

Trace weaknesses, none blocking:

- No trace row supports "per-run lock" (F4) or the dropped "in packet mode" qualifier (F5c).
- The §20 row "last bullet period → semicolon" cites "same edit pattern as the D6 merge". It is structural and
  has no Gate A source. I did not verify the D6 merge claim.

## 6. Check-off list

| Amendment subsection | Faithful to Gate A | Faithful to live `architecture.md` text |
|---|---|---|
| §9 (§1.1) | PASS | PASS |
| §10 (§1.2) | PASS | PASS |
| §12.4 (§1.3) | FAIL-WITH-FINDINGS (F4 minor extra detail) | FAIL-WITH-FINDINGS (F3 undefined vocabulary; F10 note) |
| §13 (§1.4) | PASS | PASS |
| §16 (§1.5) | FAIL-WITH-FINDINGS (F5, F6) | FAIL-WITH-FINDINGS (F3; F7 note) |
| §20 (§1.6) | PASS | FAIL-WITH-FINDINGS (F2 referent; F1 disclosure) |

"FAIL-WITH-FINDINGS" here means a MINOR drafting defect, not a substantive mismatch.

## 7. What must change before Lennon's sign-off

1. F1: list the D10 departures (non-contract success refusal; packet-less dispatch refusal) in the header.
2. F2: move the §20 insertion so "Once these conditions hold" still follows the list.
3. F3: define the permission-class vocabulary, the unit field, "host-gated", and the packet-binding term, or
   cross-reference where they are defined.
4. F4: remove "per-run" from the lock phrase.
5. F5: mark "only org and project may declare `ordinary`" as not enforced in v1; state the literal catch-all
   form; restore "in packet mode"; reword "existing closure check".
6. F6: write `issuing_reason` instead of "level", and close open question 8.

F7–F11 need no change. Recommended: add the F7 sentence. Next action: Ming revises the amendment (F1–F6) and
re-submits it; then Lennon signs off; the merge into `docs/architecture.md` remains a separate step.
