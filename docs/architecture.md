# HMA Architecture and Product Contract

Status: bootstrap architecture

Scope: portable product contract for v1

Implementation status: not started

## 1. Product promise

HMA is a human-gated, machine-assisted stage governor for coding work in one Git repository per run.

It answers one question at each gate:

> Has this stage produced sufficient, fresh, relevant evidence for a human to authorize the next stage?

HMA is not an autonomous coding orchestrator. It may gather evidence, run approved deterministic checks, detect contradictions and drift, propose an agent/model route, and recommend one next action. It must not silently advance state, dispatch a worker, substitute a model, waive a rule, publish changes, or declare success.

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

A human must approve every material stage transition. Humans also:

- approve acceptance criteria and non-goals;
- approve plans and individual implementation units;
- confirm or raise risk classifications;
- approve the proposed implementer and validator routes;
- grant explicit waivers for waivable rules;
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

### 5.2 Stage statuses

Each stage may be:

- `DRAFT`
- `READY_FOR_REVIEW`
- `APPROVED`
- `WAIVED`
- `FAILED`
- `BLOCKED`
- `UNKNOWN`
- `ABORTED`

Terminal run outcomes are:

- `VERIFIED_SUCCESS`
- `VERIFIED_WITH_WAIVERS`
- `PARTIAL`
- `FAILED`
- `BLOCKED`
- `UNKNOWN`
- `ABORTED`

Failure, uncertainty, and refusal to advance are valid product outcomes. HMA succeeds when it reports the state truthfully.

### 5.3 Transition rule

A machine may prepare a transition packet but cannot advance the state. Each transition requires a fresh, explicit human decision bound to the exact transition inputs.

No timeout, silence, earlier approval, agent confidence, or generic instruction such as `continue` counts as approval for a different transition.

## 6. Human approval contract

Every approval is single-use and bound to:

- run identifier;
- current stage;
- proposed target stage;
- repository identity;
- exact base and head revisions;
- accepted plan digest;
- evidence digest;
- validator-report digest;
- active waivers;
- short-lived challenge nonce;
- approving actor and timestamp.

A relevant change to the revision, scope, plan, evidence, validator report, or waiver set invalidates the approval.

The approval interface is challenge-bound rather than a simple `y/N` prompt. Exact CLI syntax remains an implementation decision and is intentionally not invented in this architecture document.

## 7. Policy hierarchy and waivers

Policy is evaluated in this order:

1. Hardcoded safety kernel
2. Organization policy
3. Project policy
4. Task policy
5. Explicit human waivers for rules declared waivable

A lower layer may tighten an upper layer but cannot weaken it.

### 7.1 Unwaivable kernel

The following are unwaivable:

- correct task, repository, branch/base, and revision identity;
- valid authority and permission envelope;
- authentic, current, revision-bound evidence;
- proof that required commands were executed rather than described;
- secret, destructive, infrastructure, publication, and external-effect boundaries;
- separation between implementation and approval;
- prohibition on concealed failures and altered acceptance criteria;
- transition-record integrity.

### 7.2 Waivable findings

A policy may declare findings such as these waivable:

- advisory lint or style findings;
- noncritical coverage targets;
- optional documentation;
- complexity warnings;
- minor performance targets;
- non-safety reviewer recommendations.

A waiver records actor, rationale, scope, affected criteria, and expiry conditions. It remains visible downstream. A run with any active waiver cannot end as `VERIFIED_SUCCESS`; the appropriate successful outcome is `VERIFIED_WITH_WAIVERS`.

A waiver expires when its relevant revision, scope, plan, or evidence changes.

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

A material criteria change rewinds the run to the appropriate earlier stage and invalidates affected approvals.

A validator may report an out-of-criteria observation as `ADVISORY`, unless it exposes an unwaivable safety violation. It cannot silently create a new blocker.

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

Completion, budget exhaustion, repeated failure, scope drift, or a plan change returns control to a human gate.

## 11. Minimal-implementation discipline

Minimal-implementation review is mandatory and inspired by the Ponytail approach.

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

HMA proposes routes but does not silently dispatch them.

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

Credential-free host-local profiles map verified agents, providers, models, effort settings, access services, runtimes, and trust boundaries onto the taxonomy.

Safe live discovery may identify executable and model candidates. Availability is not proof of eligibility, isolation, quota, capability, or trustworthiness.

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

## 13. Risk and validator independence

HMA computes a deterministic minimum risk floor. Organization and project policy may raise it. The human confirms or raises it. A permitted downgrade must be an explicit waiver; safety-kernel failures remain unwaivable.

Automatic high-risk triggers include security/authentication, secrets/privacy, destructive operations, production/infrastructure, publication, schema migration, public API or architecture changes, scientific assumption changes, broad diffs, missing tests, external side effects, waived correctness criteria, conflicting validator findings, and materially unknown environments.

Independence scales with risk:

- separate validation session always;
- different model for normal work;
- different provider/model family for high-risk, disputed, or final validation;
- no silent validator substitution.

## 14. Evidence acquisition and record

Agent-submitted evidence is admissible as a lead. Critical evidence must be freshly captured or independently reproduced by HMA. Trusted CI evidence may be imported only when bound to the exact revision.

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

Shell interpolation is not required for v1 evidence execution. The runner should prefer explicit executable/argument arrays.

## 15. Persistence and portable bundles

The canonical in-progress record lives in a host-managed append-only store outside the worker's writable project tree.

Each correction creates a new record and references the superseded record. History is not overwritten.

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
- `WAIVEABLE`
- `ADVISORY`

The validator may not edit code, evidence, or state; invent requirements; demand unrelated cleanup; redesign architecture; or continue auditing after every criterion is resolved.

## 17. Failure, retry, and escalation

Failures are classified before deciding what to do:

- `TRANSIENT_RUNTIME_FAILURE`: one identical retry may be proposed if it cannot duplicate external effects.
- `CORRECTABLE_EXECUTION_FAILURE`: one bounded correction by the same route may be proposed using the failure evidence.
- `CAPABILITY_MISMATCH`: do not repeat; return with the explicit escalation route.
- `VALIDATION_FAILURE`: return to planning or implementation; do not merely switch to a stronger model.
- `PERMISSION_OR_SAFETY_BLOCK`: stop as `BLOCKED`; no route may bypass the boundary.
- `UNKNOWN_FAILURE`: stop as `UNKNOWN` and ask for human judgment.

The same material failure twice ends that route as `FAILED`. A third blind attempt is prohibited.

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
4. material evidence and failures;
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
- no unresolved `BLOCK` finding remains;
- required evidence is fresh and revision-bound;
- required CI checks pass;
- scope and diff are reconciled;
- known limitations are represented honestly;
- advisory findings are dispositioned as `ACCEPT`, `PARK`, or `KILL`.

Once these conditions hold, auditing stops. Potential improvement alone is not a blocker.

## 21. CI release gate

CI verifies both transition-chain integrity and fresh release-critical behavior against the exact proposed revision.

It must:

- verify repository and revision identity;
- verify every mandatory transition approval;
- reject approvals made stale by later changes;
- verify waiver validity and propagation;
- verify implementer/validator independence requirements;
- verify plan, diff, evidence, and validator digests;
- reject missing, malformed, or rewritten records;
- rerun release-critical tests and checks;
- map results to approved criteria;
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
- external web content;
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

### 24.1 Deterministic fixture suite

Fixtures must prove that HMA:

- rejects missing or stale repository grounding;
- binds approvals to one challenge and digest set;
- invalidates approvals after relevant changes;
- proposes one primary and one escalation route;
- catches unnecessary dependency or abstraction proposals;
- limits implementation to one approved plan unit and budget;
- detects scope and diff drift;
- rejects agent success prose as evidence;
- captures verification commands and outputs independently;
- represents `FAILED`, `BLOCKED`, `UNKNOWN`, `PARTIAL`, and `ABORTED` honestly;
- prevents validators from inventing requirements;
- permits closure with dispositioned advisory findings;
- propagates waivers into `VERIFIED_WITH_WAIVERS`;
- rejects tampered bundles and stale approvals in CI mode;
- reruns critical checks against the exact revision;
- always proposes one next action, owner, route, and required approval.

### 24.2 Real-repository pilot

After fixtures pass, run one small task in one separately named Git repository. The pilot must reach truthful closure without bypassing a gate. The pilot repository is not the HMA product repository.

## 25. Bootstrap milestone and open decisions

This architecture document is the bounded bootstrap milestone. It does not establish that HMA is implemented, installable, secure, or ready for users.

Open decisions for the implementation-planning gate:

- exact Go module and package layout;
- exact CLI command surface;
- JSON Schema definitions;
- local store location and retention policy;
- challenge expiry duration and actor identity mechanism;
- CI platform adapter used for the pilot;
- initial host-local runtime-profile schema;
- fixture repository design;
- first real pilot repository;
- license and public-release posture.

Each decision must preserve the authority, portability, minimality, and evidence boundaries defined above.
