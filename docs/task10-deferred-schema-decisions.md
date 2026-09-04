# Task 10 — The Two Deferred Schema Decisions

Status: decided and implemented. Approved by Lennon on 2026-09-04.

Both questions were left open when the audit fixes landed, because each needed
a product decision rather than a repair. Neither invents new contract: the
architecture already answers the first, and deliberately does not require the
second.

## 1. Dirty-worktree evidence and `WorktreeDigest`

`EvidenceRef.WorktreeDigest` had existed since Task 1 and was never populated,
because `pilot` refused a dirty worktree outright. The field looked dead. It
was not: architecture section 14 already settles the semantics.

> Evidence captured from a dirty worktree is admissible only when the record
> includes the exact worktree-content digest; it is not revision-reproducible
> and may support only the explicitly approved dirty-worktree use.

So the gap was an unimplemented contract, not a redundant field, and deleting
the field would have removed a requirement rather than tidying one.

**Decided:** implement it. A dirty worktree is still refused by default. It is
admissible only when the host sets `dirty_worktree_approved` *and*
`expected_worktree_digest`, and HMA's recomputed digest matches. Approving
"dirty" therefore never means approving whatever happens to be on disk at
capture time — it approves one exact worktree state. The resulting evidence
record carries `worktree_digest`, which is what marks it as not
revision-reproducible; a clean capture leaves the field empty.

The digest covers the head revision and every path Git reports as changed,
each with its status and the digest of its current bytes, or an explicit
absence marker for a deleted path. Entries are sorted, domain-separated and
length-framed, so two different worktrees cannot collide by concatenation.
`--porcelain -z` is used because the default output quotes unusual paths, which
would make the digest depend on how a path renders rather than on what it is.
Only the paths Git names are read, and nothing is written, so taking the digest
never mutates the repository or its object store.

## 2. Revision binding for waivers and finding overrides

`pilot` binds an approval to repository identity, revision and diff. Human
resolutions bound none of that. The question was whether they must.

The architecture does not require it. Section 7 says a waiver records "actor,
rationale, scope, affected criteria or artifacts, and expiry conditions", and
section 6's universal binding list governs *approvals of stage transitions*,
which a waiver explicitly is not — a waiver operation "never advances a stage".
Making revision binding mandatory would have tightened the contract by
implementation rather than by amendment.

**Decided:** make it bindable, not mandatory. `WaiverOperation` and
`FindingDispositionOverride` accept optional `repository_identity` and
`base_revision`. When an operation declares either, the evaluator enforces it
against the live value the host supplies and rejects a mismatch with
`STALE_REPOSITORY_BINDING` or `STALE_REVISION_BINDING`. An operation that
declares neither remains valid, exactly as before.

A host that wants pilot-grade binding can now have it; the contract still
permits an unbound waiver. Existing records are unaffected, because both fields
are omitted when empty.

## What remains unbound

A waiver is still not bound to a produced head or diff digest, and no approval
binds a worktree digest. Both would change section 6's universal binding list,
which is an architecture amendment and needs its own approved packet. Neither
is required by the contract as written.
