# HMA Architecture and Product Contract

Status: bootstrap architecture

Scope: portable product contract for v1

Implementation status: not started

## 1. Product promise

HMA is a human-gated, machine-assisted stage governor for coding work in one Git repository per run.

It answers one question at each gate:

> Has this stage produced sufficient, fresh, relevant evidence for a human to authorize the next stage?

HMA is not an autonomous coding orchestrator. It may gather evidence, run approved deterministic checks, detect contradictions and drift, propose an agent/model route, and recommend one next action. HMA does not dispatch workers in v1; after approval it emits route and permission records, and a human or host starts execution. It must not silently advance state, substitute a model, waive a rule, publish changes, or declare success.

The product has three explicitly separate layers:

1. Portable core: state model, policy hierarchy, evidence contracts, outcome semantics, and schemas.
2. Host integration: optional adapters for local agent runtimes and CI systems.
3. User-local setup: verified runtime/model profiles, trust decisions, quota routes, and credentials kept outside the repository.

## 2. Goals

HMA must counter these common agent failures:

- implementation before sufficient grounding;
- success claims without fresh execution evidence;
- stale evidence reused after a revision changes;
- scope drift and unnecessary architecture;
- endless auditing that invents new closure requirements;
- repeated retries that do not learn from failure;
- inability to report `FAILED`, `BLOCKED`, or `UNKNOWN` honestly;
- implementers validating or approving their own work;
- silent model/provider/quota substitution;
- human approval reduced to a reflexive `y` prompt.

## 3. Non-goals

V1 will not:

- orchestrate coding autonomously;
- support non-Git tasks;
- treat prompts or skills as hard enforcement;
- ship a daemon, database, TUI, plugin framework, or provider SDK layer;
- require clean-room reproduction of every keystroke;
- infer model eligibility from model-list availability;
- invent new acceptance criteria during validation;
- make advisory findings block valid closure;
- claim that a passing scaffold is an end-user-ready product.

## 4. Authority model

### 4.1 Human authority

A human must approve every stage transition. Humans also:

- approve acceptance criteria and non-goals;
- approve plans and individual implementation units;
- confirm or raise risk classifications;
- approve the proposed implementer and validator routes;
- grant explicit waivers of `WAIVABLE` findings;
- resolve disputed semantic findings;
- authorize release or publication;
- stop or abort a run at any time.

### 4.2 Machine authority

HMA may:

- collect, hash, and retain evidence;
- run explicitly approved verification commands;
- check record shape, freshness, revision binding, and integrity;
- enforce the unwaivable safety kernel;
- detect scope, diff, budget, permission, and policy drift;
- propose one primary route and one escalation route;
- request bounded independent validation;
- recommend exactly one next action;
- block structurally invalid or unsafe transitions.

HMA may not grant itself permission to advance. A human cannot relabel an unwaivable safety failure as verified.

## 5. Workflow and state model

### 5.1 Stages

A run progresses through:

1. Grounding
2. Acceptance criteria
3. Planning
4. Route selection
5. Implementation authorization
6. Implementation review
7. Verification
8. Independent validation
9. Release and closure

### 5.2 Enumerated vocabularies

Stages are `GROUNDING`, `ACCEPTANCE_CRITERIA`, `PLANNING`, `ROUTE_SELECTION`,
`IMPLEMENTATION_AUTHORIZATION`, `IMPLEMENTATION_REVIEW`, `VERIFICATION`,
`INDEPENDENT_VALIDATION`, and `RELEASE_AND_CLOSURE`. Stage-control states are
`DRAFT`, `READY_FOR_REVIEW`, and `APPROVED`; stages are never waived.

Criterion dispositions are `PENDING`, `PASSED`, `FAILED`, and `WAIVED`. Only
criteria may be `WAIVED`.

Validator finding dispositions are `BLOCK`, `WAIVABLE`, and `ADVISORY`.
Advisory dispositions are `ACCEPT`, `PARK`, and `KILL`. `WAIVABLE` is not a
criterion state or an automatic waiver; an unresolved `WAIVABLE` finding blocks
closure until it is waived, withdrawn, or corrected.

Terminal run outcomes are:

- `VERIFIED_SUCCESS`
- `VERIFIED_WITH_WAIVERS`
- `PARTIAL`
- `FAILED`
- `BLOCKED`
- `UNKNOWN`
- `ABORTED`

`PARTIAL` is a terminal outcome, derived when the run stops with a truthful,
bounded result that resolves some approved criteria but cannot satisfy all
closure conditions; it is never a stage status. `VERIFIED_SUCCESS` requires
every criterion `PASSED`; `VERIFIED_WITH_WAIVERS` requires every criterion
`PASSED` or `WAIVED` and at least one active waiver. `FAILED`, `BLOCKED`,
`UNKNOWN`, and `ABORTED` are derived from the corresponding terminal condition.
HMA succeeds when it reports the state truthfully.

### 5.3 Transition rule

A machine may prepare a transition packet but cannot advance the state. Each transition requires a fresh, explicit human decision bound to the exact transition inputs.

No timeout, silence, earlier approval, agent confidence, or generic instruction such as `continue` counts as approval for a different transition.

### 5.4 Legal transitions and invalidation

All listed edges require human approval. Route selection is per implementation
unit, never a run-wide selection.

| From | Legal target | Invalidation or return effect |
| --- | --- | --- |
| Grounding | Acceptance criteria | changed grounding inputs invalidate downstream packets |
| Acceptance criteria | Planning | criteria or non-goal change rewinds to Acceptance criteria and invalidates downstream packets |
| Planning | Route selection | plan change rewinds to Planning and invalidates that unit's route and downstream packets |
| Route selection | Implementation authorization | route, permission, or independence change rewinds to Route selection |
| Implementation authorization, implementation review, verification, or independent validation | Route selection | human-confirmed escalation returns the affected unit to its approved route gate without inheriting broader permission |
| Implementation authorization | Implementation review | each unit is separately authorized; implementation evidence invalidates that unit's downstream packets |
| Implementation review | Verification | produced head/diff change returns to Implementation authorization or review, as applicable |
| Verification | Independent validation | plan-declared required evidence, head, or diff change returns to the affected earlier unit gate |
| Independent validation | Implementation authorization, Planning, or Release and closure | correction returns to implementation; criterion/plan correction rewinds to Planning; a passing result may request release |
| Release and closure | Route selection for next unit, Acceptance criteria, Planning, or terminal outcome | next unit starts its own route loop; criteria/plan changes rewind as above; release request may terminate only after CI and human release approval |
| Any nonterminal stage | Grounding, Planning, or terminal `BLOCKED`, `UNKNOWN`, `FAILED`, `ABORTED`, or `PARTIAL` | escalation, invalidated grounding, or the truthful stopping condition determines the target |

No unlisted edge is legal. A waiver operation invalidates only approvals bound to
the affected criterion, finding, or artifact; it never advances a stage.

The Phase A1 write path records human-initiated `ABORTED` and derived,
human-confirmed `PARTIAL` outcomes; see
[Task 13 terminal outcomes](task13-terminal-outcomes.md) for its criteria rule
and the outcomes deferred to later phases.

## 6. Human approval contract

Every approval is single-use and universally bound to:

- run identifier;
- transition digest;
- current stage;
- Proposed target (stage or terminal outcome). Exactly one of `proposed_target_stage` or `proposed_target_outcome` must be non-empty. The transition digest covers the populated target. The additive `proposed_target_outcome` field is omitted when empty, preserving existing approval serialization and digests.
- repository-identity digest;
- exact base-revision digest;
- stage-time digest (predecessor chain head and monotonic sequence);
- accepted plan digest;
- plan-declared required evidence digest set;
- active waiver-operation digest set;
- short-lived challenge nonce;
- approving actor and timestamp.

Implementation review, verification, independent validation, and release and
closure approvals additionally bind the produced head revision and diff digest.
Other transitions do not bind a produced head or diff. A packet becomes stale
only when a field it actually binds changes; CI applies the same rule.

An approval may additionally bind the exact worktree-content digest of an
approved dirty worktree (amendment, 2026-09-08). The binding is optional at
every stage. When it is populated, the transition is refused if the current
worktree digest differs from the approved digest — including when the worktree
is now clean, since a bound digest is never empty. When it is absent, a dirty
worktree remains governed by the explicit dirty-worktree approval of the exact
content, exactly as before; the amendment adds a binding, never a requirement.

A change to any actually bound field invalidates the approval.

The approval interface is challenge-bound rather than a simple `y/N` prompt. Exact CLI syntax remains an implementation decision and is intentionally not invented in this architecture document.

## 7. Policy hierarchy and waivers

Policy is evaluated in this order:

1. Hardcoded safety kernel
2. Organization policy
3. Project policy
4. Task policy

A lower layer may narrow an upper layer but cannot weaken or expand it. Waivers
are run operations outside this hierarchy and remain constrained by `WAIVABLE`
classes defined by kernel and organization policy; project and task policy may
narrow those classes only. Any lower-layer expansion is a kernel violation.

### 7.1 Unwaivable kernel

The following are unwaivable:

- correct task, repository, branch/base, and revision identity;
- valid authority and permission envelope;
- authentic, current, revision-bound evidence;
- proof that required commands were executed rather than described;
- plan-declared commands execute only as explicit executable/argument vectors and never through a shell;
- secret, destructive, infrastructure, publication, and external-effect boundaries;
- separation between implementation and approval;
- prohibition on concealed failures and altered acceptance criteria;
- transition-record integrity.

### 7.2 Waivable findings

A kernel or organization policy may declare finding classes such as these
waivable:

- complexity-budget exceedance;
- minor performance targets;
- non-safety reviewer recommendations.

Lesser minimality observations are `ADVISORY`.

A waiver is an explicit human operation on a `WAIVABLE` finding. It records
actor, rationale, scope, affected criteria or artifacts, and expiry conditions,
and remains visible downstream. It expires only when its scoped artifacts or
criteria change. A run with any active waiver cannot end as `VERIFIED_SUCCESS`;
the appropriate successful outcome is `VERIFIED_WITH_WAIVERS`.

A waiver operation may additionally bind the produced head revision or diff
digest it was granted against (amendment, 2026-09-08). When such a binding is
populated, the operation is rejected if the repository's current state no
longer matches it. A waiver that binds neither remains valid, exactly as
before: the binding is optional, not a new requirement.

## 8. Grounding contract

Grounding is task-shaped: neither lazy guessing nor an unjustified whole-repository scan is acceptable.

The grounding packet must contain:

- canonical repository root and identity;
- branch, upstream, base revision, head revision, and dirty state;
- applicable repository and project instructions;
- immutable copy or digest of the original user request;
- proposed acceptance criteria and explicit non-goals;
- relevant entry points, call sites, tests, and configuration;
- current behavior or a reproduced failure when applicable;
- existing implementations, standard-library features, and dependencies available for reuse;
- permission and trust boundaries;
- known constraints, unknowns, and unverified assumptions;
- justification that the inspected scope is sufficient;
- grounding evidence digest.

Broad scanning requires an explicit reason and budget.

## 9. Acceptance-criteria contract

Acceptance criteria are co-authored:

1. Preserve the user request without semantic rewriting.
2. Let the planning process propose measurable criteria and non-goals.
3. Trace each derived criterion to the original request.
4. Require explicit human approval.
5. Freeze the approved set for downstream stages.

A criteria change rewinds the run to the appropriate earlier stage and
invalidates affected approvals.

Out-of-criteria observations are `ADVISORY` unless they expose an unwaivable safety violation or identify measurable exceedance of the plan-declared complexity budget. That exceedance is `WAIVABLE` under §7.2 and does not create a new acceptance criterion. A validator cannot silently create a new blocker.

## 10. Plan-unit contract

One implementation authorization covers one smallest independently verifiable behavioral increment.

Each plan unit records:

- one observable behavior or bounded enabling change;
- acceptance criteria advanced;
- exact allowed files and actions;
- exact forbidden files and actions;
- expected failing test or pre-change evidence;
- minimum intended implementation;
- existing code or dependency reuse analysis;
- complexity, file, diff, time, tool-step, and retry budgets;
- proposed implementer route;
- proposed escalation route;
- proposed validator route and independence requirement;
- verification commands and required evidence;
- dependencies on prior units;
- stop, rollback, and escalation conditions;
- explicit non-goals.

A unit's implementation-authorization approval may bind the exact
worktree-content digest when the human authorizes work against uncommitted
content (amendment, 2026-09-08); the authorization is refused if the worktree
later differs from the approved digest.

Completion, budget exhaustion, repeated failure, scope drift, or a plan change returns control to a human gate.

## 11. Minimal-implementation discipline

Minimal-implementation review is required.

Before approving an implementation shape, the gate must ask:

1. Does the requested change need to exist?
2. Can suitable repository code be reused?
3. Can standard-library or native runtime support satisfy the contract?
4. Can an already-installed dependency satisfy it?
5. What is the simplest readable expression consistent with repository conventions?
6. What is the minimum new implementation required?

The plan declares a complexity budget and forbidden speculative abstractions. Deterministic checks detect measurable drift; the independent validator assesses proportionality.

Minimality must not remove contract-required validation, trust or security boundaries, error handling, accessibility, tests, or evidence requirements. Legitimate complexity requires an explicit human waiver and rationale.

## 12. Agent and model routing

HMA does not dispatch workers in v1; after approval it emits route and permission records, and a human or host starts execution.

### 12.1 Portable capability taxonomy

The portable core describes capabilities rather than vendors, including:

- planning and architecture;
- bounded implementation;
- debugging;
- repository scanning;
- semantic review;
- browser or runtime validation;
- risk tolerance;
- isolation and permission enforcement;
- timeout and cancellation;
- evidence and output capture;
- context and output bounds;
- cost or quota class.

### 12.2 Host-local profiles

Credential-free host-local profiles map verified agents, providers, models,
effort settings, access services, runtimes, and trust boundaries onto the
taxonomy. A portable route attestation contains only an opaque profile digest,
provider family, model family, capability classes, and permission classes; it
contains no private paths, credentials, profile names, or provider-specific
secrets.

Discovery is limited to allowlisted executable-presence, version, and model-list
commands. It must not inspect or mutate credentials or configuration, and its
output is untrusted availability metadata. Availability is not proof of
eligibility, isolation, quota, capability, or trustworthiness.

The host-local machine-truth inventory runs only exact commands supplied by the
host allowlist, each named by a clean absolute executable path, without a shell,
under a per-command timeout, and with only the environment the host explicitly
supplies; nothing is inherited from the ambient environment. It records a
per-command status (absent, present, completed, exited non-zero, timed out, or
start failed), exit code, bounded parsed version or model-availability
metadata, capture time, freshness expiry, and a length-framed output digest.
Raw command output is not retained; parsed fields that are oversized,
unprintable, or taken from truncated output are dropped and marked unparsed.
Inventory records remain host-local and are never portable route-policy
artifacts. Any status other than completed is unavailable evidence, and no
status can be converted by this component into eligibility, trust, or a route
choice. Cancellation by the caller aborts discovery rather than producing a
record. `hma inventory` loads the allowlist strictly (a versioned document,
unknown fields and duplicate command IDs refused, every command validated
before any runs) and refuses an allowlist or output path that resolves inside
the repository under inspection, because host configuration and host evidence
placed there would become part of the state HMA verifies.

### 12.3 Proposal shape

For each unit, HMA presents:

- one recommended implementer route;
- eligibility and preference rationale;
- permission and interaction mode;
- estimated cost/quota impact when known;
- capability evidence and unknowns;
- one explicit escalation route;
- conditions that justify escalation;
- required validator route and independence;
- materially relevant rejected routes;
- forbidden fallbacks or access-service substitutions.

The human approves the plan unit and route together.

Route selection policy is a versioned portable contract with one canonical
normalized projection and content digest. That projection is the routing single
source of truth: policy changes change its digest, while equivalent input
ordering does not. It contains capability, permission, risk, independence,
preference, and forbidden-fallback rules only; host-local availability and
eligibility evidence remain separate untrusted inputs.

A deterministic route-coherence check validates a proposed primary route,
escalation route, and independent validator against that policy, the approved
permission envelope, and host-supplied profile snapshots. It may reject missing
eligibility, capability or permission escalation, self-validation, forbidden
fallbacks, and access-service substitution, but a coherent result remains
pending human approval and does not select or dispatch a route.

### 12.4 Interactive host orchestration

HMA core does not dispatch workers in v1. When an interactive CLI agent or host
adapter performs the dispatch after human authorization, that interactive agent
is an **orchestrator by default**. It retains the user-facing context, grounds
and decomposes the work, proposes routes and permissions, dispatches bounded
jobs, collects implementation artifacts and evidence, and presents the result
for review. It does not perform coding work itself by default.

Direct coding by the interactive agent is permitted only as a recorded cost
exception when all of the following hold:

- the job is small, bounded, and within the approved permission envelope;
- the estimated total cost of delegation, including dispatch setup, duplicated
  context, expected execution, and review coordination, is greater than direct
  execution;
- the comparison and direct-execution rationale are recorded before work starts;
- the direct route is eligible for the task and does not widen permissions; and
- the resulting coding work still receives independent review.

Convenience, agent confidence, or failure to check an available route does not
establish the cost exception.

Before every worker dispatch, the host performs a minimal route-availability
handshake: it sends `hi` to the exact approved agent/model profile through the
approved access service and verifies that the route responds without reporting
an exhausted quota or rate limit. The preflight record binds the profile,
provider and model families, access service, timestamp, and result. A successful
greeting proves only current reachability; it does not prove eligibility,
capability, trust, or future quota. A failed or indeterminate preflight blocks
dispatch to that route and returns to the approved escalation route or a human
decision. Silent substitution is prohibited.

Before dispatch, the host also reports and records the worker identity,
provider, model, reasoning or effort level, access service, runtime, bounded
task, and permission envelope.

Every change to source code, tests, executable configuration, build logic, or
CI behavior requires independent review before it may satisfy a coding
criterion or support closure. The reviewer must be distinct from the
implementer and operate in a separate review context over a fixed, read-only
packet. Normal work requires a different model; high-risk, disputed, and final
validation additionally follow the provider/model-family separation rules in
§13. When the interactive orchestrator uses the direct-execution exception, it
is the implementer for independence purposes and therefore cannot review or
validate that work itself.


### 12.5 Advisory decision providers

Interactive hosts may use a learned decision provider for bounded soft
judgments while HMA core remains deterministic and vendor-neutral. Lennon's
default host profile uses Jev for this role.

A decision provider may recommend:

- one worker from the set already proven eligible by route policy;
- task complexity or specialist-reasoning need;
- whether a worker result appears sufficient to enter the normal
  verification/review path, or should be retried or escalated;
- one currently legal next action after a bounded unit; and
- whether an agent-generated discretionary idea should be implemented,
  prototyped, deferred, or rejected.

The decision provider never expands the legal choice set and never supplies
authority. Deterministic route, permission, risk, independence, transition, and
safety checks run first. Provider output cannot grant approval, widen
permissions, waive findings, satisfy mandatory evidence, replace independent
review, advance a stage, publish, or release.

The host records provider identity, model identity when reported, timestamp,
input-state digest, typed question/choice contract, probabilities/confidence,
configured threshold-policy version, resulting host action, and any
orchestrator/human override. Credentials and unnecessary raw private state are
excluded.

If the configured decision provider is unavailable or indeterminate, the host
records that condition and returns the soft judgment to the orchestrator or
human. It must not silently substitute another model under the configured
provider identity or weaken any deterministic requirement.

The portable core defines only this provider-neutral contract. Jev-specific
MCP, SDK, HTTP, credential, retry, quota, and endpoint handling belongs to host
integration and user-local setup.


## 13. Risk and validator independence

HMA computes a deterministic minimum risk floor. Organization and project policy may raise it. The human confirms or raises it. A permitted downgrade must be an explicit waiver; safety-kernel failures remain unwaivable.

Automatic high-risk triggers include security/authentication, secrets/privacy, destructive operations, production/infrastructure, publication, schema migration, public API or architecture changes, scientific assumption changes, broad diffs, missing tests, external side effects, waived correctness criteria, conflicting validator findings, and materially unknown environments.

Independence scales with risk:

- separate validation session always;
- different model for normal work;
- different provider/model family for high-risk, disputed, or final validation;
- no silent validator substitution.

If a required independent validator is unavailable, the result is `BLOCKED`
pending an eligible independent model or human validator; HMA must not silently
downgrade the independence requirement.

## 14. Evidence acquisition and record

Agent-submitted evidence is admissible as a lead. Plan-declared required evidence must be freshly captured or independently reproduced by HMA. Trusted CI evidence may be imported only when bound to the exact revision.

Agent prose is never proof.

An evidence record includes:

- exact executable and argument vector;
- working directory;
- start and end timestamps;
- exit code;
- complete stdout/stderr or immutable references with explicit truncation status;
- repository identity, base, and head revision;
- changed-file list and binary diff digest;
- relevant tool versions;
- acceptance criterion addressed;
- capturing actor;
- freshness and integrity result;
- unavailable evidence, omissions, and deviations.

Evidence from a clean bound revision is revision-reproducible. Evidence captured from a dirty worktree is admissible only when the record includes the exact worktree-content digest; it is not revision-reproducible and may support only the explicitly approved dirty-worktree use. The evidence runner MUST execute an explicit executable and argument vector and MUST NOT invoke a shell. This is a kernel rule.

## 15. Persistence and portable bundles

The canonical in-progress record lives in a host-managed append-only store outside the worker's writable project tree. Each record carries the hash of its predecessor, a monotonically increasing per-run sequence, and the resulting per-run head anchor. Missing sequence continuity, a wrong predecessor hash, or a head-anchor mismatch detects truncation or rewrite.

Each correction creates a new record and references the superseded record. Append-only in v1 is detection-based, not an assertion of immutable storage. CI authenticates an exported run-head anchor against configured trust roots and rejects a failed authentication or chain check. The exact trust mechanism is an open implementation decision.

Portable export consists of content-addressed JSON records conforming to published JSON Schema. A small repository or CI manifest may point to the required bundle and digests. The repository must not contain host-local credentials, model profiles, private paths, or secret endpoint details.

## 16. Independent validation contract

The semantic validator runs in a separate, read-only context and receives only a fixed packet:

- immutable request;
- approved criteria and non-goals;
- approved plan unit and budgets;
- repository base/head and diff;
- evidence bundle;
- active waivers;
- applicable policy.

It may evaluate only:

- acceptance-criterion coverage;
- evidence relevance and sufficiency;
- material scope or complexity drift;
- safety-kernel compliance;
- contradictions between claims and evidence;
- minimal-implementation compliance.

Each finding must cite the exact criterion or invariant, evidence, material impact, and one disposition:

- `BLOCK`
- `WAIVABLE`
- `ADVISORY`

The validator may not edit code, evidence, or state; invent requirements; demand unrelated cleanup; redesign architecture; or continue auditing after every criterion is resolved.

## 17. Failure, retry, and escalation

HMA applies deterministic rules to classify observed failure evidence, subject to human confirmation; it never accepts an agent's self-classification as the failure result. The per-unit retry budget caps all attempts, including retries, corrections, and re-routed attempts. Failures are classified before deciding what to do:

- `TRANSIENT_RUNTIME_FAILURE`: one identical retry may be proposed if it cannot duplicate external effects.
- `CORRECTABLE_EXECUTION_FAILURE`: one bounded correction by the same route may be proposed using the failure evidence.
- `CAPABILITY_MISMATCH`: do not repeat; return with the explicit escalation route.
- `VALIDATION_FAILURE`: return to planning or implementation; do not merely switch to a stronger model.
- `PERMISSION_OR_SAFETY_BLOCK`: stop as `BLOCKED`; no route may bypass the boundary.
- `UNKNOWN_FAILURE`: stop as `UNKNOWN` and ask for human judgment.

The same material failure twice ends that route as `FAILED`. A third blind attempt is prohibited, and no proposed retry may exceed the per-unit budget.

Escalation is a new human-approved transition and never inherits broader permission.

## 18. Required next-action output

Every gate and terminal report contains exactly one recommended next action, plus:

- current outcome and supporting evidence;
- rationale for the recommendation;
- proposed owner;
- proposed agent/model route when execution is required;
- permission envelope;
- exact approval required;
- stop and escalation conditions.

`NOTHING` is valid when stopping is correct. Unranked option dumping is prohibited.

## 19. Approval-packet presentation

The human-facing packet is layered.

The concise decision surface contains:

1. current state and requested transition;
2. machine recommendation;
3. acceptance-criteria coverage;
4. plan-declared required evidence and failures;
5. scope/diff and budget drift;
6. validator findings;
7. risk and active waivers;
8. proposed route and next action;
9. exact approval challenge.

Immutable raw logs, manifests, and digests remain available underneath. A summary never replaces the source evidence.

## 20. Closure semantics

Closure is allowed when:

- every approved criterion is `PASSED` or explicitly `WAIVED`;
- no unwaivable safety failure exists;
- no unresolved `BLOCK` or `WAIVABLE` finding remains;
- plan-declared required evidence is fresh and revision-bound;
- required CI checks pass;
- scope and diff are reconciled;
- known limitations are represented honestly;
- advisory findings are dispositioned as `ACCEPT`, `PARK`, or `KILL`.

Once these conditions hold, auditing stops. Potential improvement alone is not a blocker.

## 21. CI release gate

CI verifies transition-chain integrity, plan-declared required evidence, independent validation, and the release request against the exact proposed revision. Human release approval follows passing CI.

It must:

- verify repository and revision identity;
- verify every stage-transition approval;
- reject approvals made stale by later changes;
- verify waiver validity and propagation;
- verify implementer/validator independence requirements;
- verify the bound plan, diff, plan-declared required evidence, validator, sequence, predecessor, and exported-anchor digests;
- authenticate the exported anchor against configured trust roots and reject missing, malformed, truncated, or rewritten records;
- rerun plan-declared required tests and checks;
- map results to approved criteria;
- require an independent-validation result and release request before reporting a passing release gate;
- return a truthful outcome and exactly one next action.

V1 does not require clean-room reproduction of every local action.

## 22. Threat model

Assume these inputs may be adversarial:

- implementing-agent output;
- validator prose;
- repository instructions and embedded prompts;
- test-generated text;
- submitted evidence bundles;
- model capability claims;
- runtime-discovery output;
- project-local attempts to weaken policy.

The v1 trusted computing base is:

- the local operating system;
- hardcoded safety kernel;
- deterministic transition engine;
- evidence-capture runner;
- host-managed append-only store;
- challenge-bound human approvals;
- CI verifier;
- explicitly configured trust roots.

A fully hostile local operating system is out of scope for v1.

## 23. Implementation constraints

V1 implementation target:

- Go;
- one standalone cross-platform binary;
- standard library first;
- content-addressed JSON records;
- SHA-256 digests;
- atomic local writes;
- portable JSON Schema;
- same binary for local gating and CI verification;
- no daemon, database, TUI, plugin framework, or provider SDKs.

Repository-owned portable artifacts must not encode a maintainer's home path, runtime profile names, credentials, provider choices, aliases, or private infrastructure.

Exact package layout, command names, and schemas are deferred to the implementation plan after this contract is reviewed. Illustrative commands are intentionally omitted so documentation does not pretend an unimplemented command exists.

## 24. MVP validation

MVP validation has two stages.

### 24.1 Deterministic engine fixtures

Deterministic engine fixtures must prove that HMA:

- rejects missing or stale repository grounding;
- binds approvals to one challenge and digest set;
- invalidates approvals after relevant changes;
- proposes one primary and one escalation route;
- limits implementation to one approved plan unit and budget;
- detects scope and diff drift;
- rejects agent success prose as evidence;
- captures verification commands and outputs independently;
- represents `FAILED`, `BLOCKED`, `UNKNOWN`, `PARTIAL`, and `ABORTED` honestly;
- deterministically interprets recorded findings without semantic model judgment;
- permits closure with dispositioned advisory findings;
- propagates waivers into `VERIFIED_WITH_WAIVERS`;
- rejects tampered bundles and stale approvals in CI mode;
- reruns plan-declared required checks against the exact revision;
- always proposes one next action, owner, route, and required approval.

### 24.2 Validator-contract fixtures

Validator-contract fixtures use recorded or stubbed validator outputs and must
prove that finding vocabulary, independence blocking, waiver operations,
criterion traceability, closure behavior, unnecessary dependency or abstraction
proposals, and prevention of validator-invented requirements are handled
consistently without treating the validator as a deterministic engine fixture.
Deterministic engine tests interpret those recorded findings; they do not make
semantic model judgments. Any numeric drift threshold used by either suite is
declared by policy, not invented by the fixture.

### 24.3 Real-repository pilot

After fixtures pass, run one small task in one separately named Git repository. The pilot must reach truthful closure without bypassing a gate. The pilot repository is not the HMA product repository.

## 25. Bootstrap milestone and open decisions

This architecture document is the bounded bootstrap milestone. It does not establish that HMA is implemented, installable, secure, or ready for users.

Open decisions for the implementation-planning gate:

- exact Go module and package layout;
- exact CLI command surface;
- JSON Schema definitions;
- local store location and retention policy;
- challenge expiry duration and actor identity mechanism;
- exported-anchor authentication mechanism and configured trust-root format;
- CI platform adapter used for the pilot;
- initial host-local runtime-profile schema;
- versioned route-policy contract and canonical normalized projection;
- fixture repository design;
- first real pilot repository;
- license and public-release posture.

Each decision must preserve the authority, portability, minimality, and evidence boundaries defined above.
