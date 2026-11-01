# Gate E architecture-amendment draft — re-review of revision 2

- **Date:** 2026-10-31
- **Reviewer:** Phil / OpenAI `gpt-6-luna` / GitHub Copilot CLI, verification mode.
- **Scope:** Re-review of revision 2, following up on `audit-gate-e-arch-amendment-phil.md`.
- **Inputs (read-only):** `docs/handoffs/2026-10-31-gate-e-arch-amendment-draft.md` (revision 2);
  my prior review; live `docs/architecture.md` §20; `internal/model/route_policy.go:48-58`;
  `docs/handoffs/2026-10-31-gate-e-arch-amendment-trace.md` (grep only).
- **Limits:** No revision-1 copy exists (all files untracked, none in git). I could not run a byte diff. Step 3
  rests on my prior review's quotations and anchor table. I ran no code.

## Verdict: PASS WITH FINDINGS

No BLOCKING finding. No narrowing beyond the four Gate A narrowings and the two disclosed D10 strengthenings.
Two new MINOR findings (N1, N2) and one NOTE (N3). All are one-line fixes. None needs a new decision.

## 1. F1–F6 status

| # | Status | Proof in revision 2 |
|---|---|---|
| F1 | RESOLVED | Header: "Two further clauses, both from Decision D10 … §20 refuses a success-claiming outcome on a non-contract run … and §12.4 refuses packet-less review dispatch … on a contract-bound run." Both are labelled as strengthenings, not narrowings. |
| F2 | RESOLVED | §1.6: "Insert after 'Once these conditions hold, auditing stops. Potential improvement alone is not a blocker.' and before '## 21. CI release gate'." Live `architecture.md` lines 760–762 match. The list-to-sentence referent is intact. |
| F3 | RESOLVED-WITH-NEW-ISSUE | Defined now: "host-gated edge: one requiring the host's deterministic approval, not merely a human confirmation"; "the review packet binding — the structured binding fields on an issued review packet record"; "the unit's sorted permission classes (`permission_classes`)". The permission-class vocabulary is not defined. It is cross-referenced to code (see N1). |
| F4 | RESOLVED | "with the core head re-checked under the host lock." "per-run" is gone. Matches Gate A E0 step 4 and trace row 29 ("under lock"). |
| F5 | RESOLVED | (a) "This authority restriction is not mechanically enforced in v1; HMA cannot currently attribute a policy rule to the layer that authored it." (b) "A rule whose glob is `**` and whose only class is `ordinary` refuses the document … a `**` rule with additional classes beyond `ordinary` [is] not detected." (c) "credit must be re-issued after each newer review, in packet mode"; "the final gate is part of the closure credit check (§20)". |
| F6 | RESOLVED | Baseline: "latest approved review whose `issuing_reason` is `checkpoint` or `final`". E2: "its `issuing_reason` is `checkpoint` or `final`". E3: "`issuing_reason` equal to `final`". §2 item 8 is marked "resolved" and gives the reachability argument. |

Check on the F5(c) wording "closure credit check (§20)": live §20 prose has no credit check. The amendment's new §20
bullet ("every credited unit has a `final`-level review … that passes the final gate") now supplies the hook. The
reference is acceptable. It was my own earlier objection. I do not raise it again.

## 2. New findings

### N1 — MINOR — code citation in normative text, and vocabulary still defined only by code (§1.3)

- **Observed.** The new bullet says: "The permission-class vocabulary used here, including the `no_external_effect`
  class, is defined in code at `internal/model/route_policy.go:48-58`, not restated in architecture text."
- **Reference.** I verified the citation: lines 48–58 define `read_only`, `bounded_write`, `no_external_effect`.
  The citation is correct.
- **Problem 1.** The draft's own "Drafting conventions" say file:line citations and Go names are "intentionally not
  in the normative text". This bullet breaks that rule. A merged architecture document would carry a line number
  that rots.
- **Problem 2.** My F3 asked for one defining sentence. A code pointer does not define. T4 allowed form and T1 still
  depend on a name that architecture text does not state. My checklist allowed a cross-reference "where defined",
  so I count this as partly resolved, not unresolved.
- **Change.** Replace the sentence with a definition, for example: "Permission classes are the portable
  interaction-envelope classes of the route policy: `read_only`, `bounded_write`, and `no_external_effect`."
  Remove the file:line. Keep the code pointer in the trace only.

### N2 — MINOR — residual undefined "level" shorthand (§1.3, §1.6)

- **Observed.** §1.6 bullet: "a `final`-level review of the closure artifact". §1.3: "does not substitute for a
  `final`-level review".
- **Problem.** F6 fixed "level" in Baseline, E2 and E3. These two uses remain. Together with the "**Level.**
  The derived level is …" paragraph, they leave room for the `minimum_issuing_reason` reading that F6 closed. The
  §20 bullet is closure text, so it matters most.
- **Change.** Write "a review whose `issuing_reason` is `final`" in both places.

### N3 — NOTE — dispatch ID still reads `-r1`

The header "Dispatch ID: hma-gate-e-arch-amendment-draft-20261031-r1" is unchanged while "Revision: 2". The trace
header carries the same ID. This is cosmetic. Leave it, or bump both, as Ming prefers.

## 3. Out-of-scope change check

- I found nothing in revision 2 that I did not ask for, except the F1–F6 fixes and the closing sentence of the
  §2 preamble ("except item 8, which this revision resolves"). That preamble change is a harmless and needed
  consequence of F6.
- The §20 insertion point moved (F2). This is the only structural move, and I asked for it.
- The seven anchors from my prior review's §3 are unchanged in wording. §1.1, §1.2, §1.4 (the §13 text, its
  tagged changed-wording list) and §1.7 read the same as I recorded them.
- **Limit:** with no revision-1 copy, I cannot prove byte-level absence of other edits. Evidence of absence is
  consistency with my prior quotations only.

## 4. F7–F11 unchanged?

| # | Check | Result |
|---|---|---|
| F7 | E3 `PARTIAL` bullet and §1.6 `PARTIAL` sentence are still as reviewed. The optional "truthful stop remains `BLOCKED`" sentence was not added. It was a recommendation, not a requirement. | Unchanged. |
| F8 | §2 item 11 (Decision U reading) | Unchanged. |
| F9 | `T1`–`T8` and "(E2)"/"(E3)" labels remain; §2 item 2 still treats identifiers as unstable | Unchanged. |
| F10 | §12.4 `review_dispatch` independence sentence is not touched | Unchanged. |
| F11 | "equals the digest of the checkpoint policy that HMA retains for the run"; §2 item 6 | Unchanged. |

## 5. Trace check

I grepped the trace only. Its rows already carry the revision-2 readings: row 29 ("under lock"), row 107 and
item 8 ("Level means `issuing_reason` … resolved"), row 135 ("in packet mode"), and the §20 non-contract row. The
revision changed no row's cited Gate A source. I made no full trace re-check, because F1–F6 touch wording, not
provenance. After N1 and N2 are fixed, no trace row changes either; the trace already maps the code citation.

## 6. Next action

Ming applies N1 and N2 (two small edits). I do not need to see the draft again if the edits are exactly those in
§2. Then Lennon signs off. The merge into `docs/architecture.md` stays a separate step.
