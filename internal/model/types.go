// Package model defines HMA's portable record families and controlled
// vocabularies. It is standard-library-only and contains no transition
// advancement, provider SDK, credential or private-path field, shell string,
// schema framework, or CI behavior.
package model

// RecordKind identifies the family of a chained record.
type RecordKind string

const (
	KindStageTransition RecordKind = "stage_transition"
	KindCriterion       RecordKind = "criterion"
	KindFinding         RecordKind = "finding"
	KindWaiver          RecordKind = "waiver"
	KindEvidence        RecordKind = "evidence"
	KindReleaseRequest  RecordKind = "release_request"
	KindOutcome         RecordKind = "outcome"
	KindWaiverOperation RecordKind = "waiver_operation"
	KindFindingOverride RecordKind = "finding_override"
)

// Stage is a run stage. Stages are never waived.
type Stage string

const (
	StageGrounding                   Stage = "GROUNDING"
	StageAcceptanceCriteria          Stage = "ACCEPTANCE_CRITERIA"
	StagePlanning                    Stage = "PLANNING"
	StageRouteSelection              Stage = "ROUTE_SELECTION"
	StageImplementationAuthorization Stage = "IMPLEMENTATION_AUTHORIZATION"
	StageImplementationReview        Stage = "IMPLEMENTATION_REVIEW"
	StageVerification                Stage = "VERIFICATION"
	StageIndependentValidation       Stage = "INDEPENDENT_VALIDATION"
	StageReleaseAndClosure           Stage = "RELEASE_AND_CLOSURE"
)

// StageControlState is a stage-control state as defined by the architecture:
// DRAFT, READY_FOR_REVIEW, and APPROVED only. INVALIDATED is not a
// stage-control state; it is a distinct packet-control vocabulary, see
// PacketControlState. Keeping INVALIDATED out of this enum enforces the
// distinction at the type level without advancing any transition.
type StageControlState string

const (
	ControlDraft          StageControlState = "DRAFT"
	ControlReadyForReview StageControlState = "READY_FOR_REVIEW"
	ControlApproved       StageControlState = "APPROVED"
)

// PacketControlState is the packet-control vocabulary, distinct from the
// stage-control enum. INVALIDATED marks a packet as invalidated; it is never
// a stage-control state and never advances a stage.
type PacketControlState string

const (
	PacketInvalidated PacketControlState = "INVALIDATED"
)

// CriterionDisposition is the state of an acceptance criterion.
// Only criteria may be WAIVED.
type CriterionDisposition string

const (
	CriterionPending CriterionDisposition = "PENDING"
	CriterionPassed  CriterionDisposition = "PASSED"
	CriterionFailed  CriterionDisposition = "FAILED"
	CriterionWaived  CriterionDisposition = "WAIVED"
)

// FindingDisposition is a validator finding disposition. WAIVABLE is not a
// criterion state or an automatic waiver; an unresolved WAIVABLE finding
// blocks closure until waived, withdrawn, or corrected.
type FindingDisposition string

const (
	FindingBlock    FindingDisposition = "BLOCK"
	FindingWaivable FindingDisposition = "WAIVABLE"
	FindingAdvisory FindingDisposition = "ADVISORY"
)

// AdvisoryDisposition dispositions an ADVISORY finding.
type AdvisoryDisposition string

const (
	AdvisoryAccept AdvisoryDisposition = "ACCEPT"
	AdvisoryPark   AdvisoryDisposition = "PARK"
	AdvisoryKill   AdvisoryDisposition = "KILL"
)

// TerminalOutcome is a terminal run outcome. The ordered first-match
// precedence is ABORTED, BLOCKED, UNKNOWN, FAILED, PARTIAL,
// VERIFIED_WITH_WAIVERS, VERIFIED_SUCCESS.
type TerminalOutcome string

const (
	OutcomeAborted             TerminalOutcome = "ABORTED"
	OutcomeBlocked             TerminalOutcome = "BLOCKED"
	OutcomeUnknown             TerminalOutcome = "UNKNOWN"
	OutcomeFailed              TerminalOutcome = "FAILED"
	OutcomePartial             TerminalOutcome = "PARTIAL"
	OutcomeVerifiedWithWaivers TerminalOutcome = "VERIFIED_WITH_WAIVERS"
	OutcomeVerifiedSuccess     TerminalOutcome = "VERIFIED_SUCCESS"
)

// ScopeKind names a canonical waiver scope selector target. "stage" is
// deliberately absent: stages are never waived. Only criterion, finding, and
// artifact scopes conform to the contract; "plan_unit" and "closure" are not
// approved waiver scopes.
type ScopeKind string

const (
	ScopeCriterion ScopeKind = "criterion"
	ScopeFinding   ScopeKind = "finding"
	ScopeArtifact  ScopeKind = "artifact"
)

// ScopeSelector is the canonical scope of a waiver operation. An unscoped
// waiver is invalid and cannot become active.
type ScopeSelector struct {
	Kind   ScopeKind `json:"kind"`
	Target string    `json:"target"`
}

// IsZero reports whether the selector is unset.
func (s ScopeSelector) IsZero() bool {
	return s.Kind == "" && s.Target == ""
}

// WaiverApplicabilitySet is the sorted set of active waiver-operation digests
// whose scope overlaps an approval packet's inputs.
type WaiverApplicabilitySet []string

// Waiver is a human waiver operation on a WAIVABLE finding or criterion.
type Waiver struct {
	ID     string        `json:"id"`
	Scope  ScopeSelector `json:"scope"`
	Active bool          `json:"active"`
	Actor  string        `json:"actor"`
}

// Criterion is an acceptance criterion and its disposition.
type Criterion struct {
	ID          string               `json:"id"`
	Disposition CriterionDisposition `json:"disposition"`
}

// Finding is a validator finding with its disposition. Advisory is used only
// when Disposition is ADVISORY.
type Finding struct {
	ID          string              `json:"id"`
	Disposition FindingDisposition  `json:"disposition"`
	Advisory    AdvisoryDisposition `json:"advisory,omitempty"`
}

// ApprovalBinding binds a single-use, challenge-bound human approval to its
// exact transition inputs. ProducedHeadDigest and DiffDigest are bound only
// for review, verification, independent validation, and release transitions.
// WorktreeDigest is optional at every stage (the section 6 amendment): when
// populated, the transition is refused if the current worktree digest differs,
// so approving work against uncommitted content approves one exact worktree
// state, never whatever happens to be on disk later.
type ApprovalBinding struct {
	RunID                    string                 `json:"run_id"`
	TransitionDigest         string                 `json:"transition_digest"`
	CurrentStage             Stage                  `json:"current_stage"`
	ProposedTargetStage      Stage                  `json:"proposed_target_stage"`
	RepositoryIdentityDigest string                 `json:"repository_identity_digest"`
	BaseRevisionDigest       string                 `json:"base_revision_digest"`
	StageTimeDigest          string                 `json:"stage_time_digest"`
	AcceptedPlanDigest       string                 `json:"accepted_plan_digest"`
	RequiredEvidenceDigests  []string               `json:"required_evidence_digests,omitempty"`
	ActiveWaivers            WaiverApplicabilitySet `json:"active_waivers,omitempty"`
	ChallengeNonce           string                 `json:"challenge_nonce"`
	Approver                 string                 `json:"approver"`
	Timestamp                string                 `json:"timestamp"`
	ProducedHeadDigest       string                 `json:"produced_head_digest,omitempty"`
	DiffDigest               string                 `json:"diff_digest,omitempty"`
	WorktreeDigest           string                 `json:"worktree_digest,omitempty"`
}

// EvidenceRef references freshly captured, revision-bound evidence. The
// command is an explicit executable and argument vector; no shell string,
// pipeline, redirection, or expansion is representable here.
type EvidenceRef struct {
	Executable         string   `json:"executable"`
	Argv               []string `json:"argv"`
	WorkingDir         string   `json:"working_dir"`
	StartTimestamp     string   `json:"start_timestamp"`
	EndTimestamp       string   `json:"end_timestamp"`
	ExitCode           int      `json:"exit_code"`
	OutputDigest       string   `json:"output_digest,omitempty"`
	OutputTruncated    bool     `json:"output_truncated,omitempty"`
	OutputRef          string   `json:"output_ref,omitempty"`
	RepositoryIdentity string   `json:"repository_identity"`
	BaseRevision       string   `json:"base_revision"`
	HeadRevision       string   `json:"head_revision"`
	ChangedFiles       []string `json:"changed_files,omitempty"`
	DiffDigest         string   `json:"diff_digest,omitempty"`
	WorktreeDigest     string   `json:"worktree_digest,omitempty"`
	CapturingActor     string   `json:"capturing_actor"`

	// Route captures the portable attestation of what executed the model call that
	// produced this evidence. Contains no credential, private path, or provider
	// secret (architecture.md §12.2). Optional.
	Route *RouteAttestation `json:"route,omitempty"`
}

// RecordMetadata is the append-only chain metadata for a record: the
// predecessor hash, a monotonically increasing per-run sequence, and the
// resulting per-run head anchor.
type RecordMetadata struct {
	RunID           string `json:"run_id"`
	Sequence        int64  `json:"sequence"`
	PredecessorHash string `json:"predecessor_hash"`
	HeadAnchor      string `json:"head_anchor"`
	Timestamp       string `json:"timestamp"`
	Actor           string `json:"actor"`
}

// IsZero reports whether chain metadata is absent.
func (m RecordMetadata) IsZero() bool {
	return m.RunID == "" && m.Sequence == 0 && m.PredecessorHash == "" &&
		m.HeadAnchor == "" && m.Timestamp == "" && m.Actor == ""
}

type WaiverOperationAction string

const (
	WaiverOperationGrant    WaiverOperationAction = "GRANT"
	WaiverOperationExpire   WaiverOperationAction = "EXPIRE"
	WaiverOperationWithdraw WaiverOperationAction = "WITHDRAW"
)

// WaiverOperation is one human resolution operation on a criterion, finding,
// or artifact.
//
// RepositoryIdentity and BaseRevision are optional. The architecture requires
// a waiver to record actor, rationale, scope and expiry, not a revision, so
// they are not mandatory here. When a host does set them the evaluator
// enforces them, which lets an operation be pinned to the repository state the
// human was looking at instead of applying to whatever the run later became.
//
// Head and DiffDigest are likewise optional (the section 7 amendment): they
// bind the produced head commit and the diff digest against the approved base
// that the human waived against, and the evaluator rejects the operation when
// the repository's current state no longer matches. A waiver that binds none
// of these fields remains valid, exactly as before.
type WaiverOperation struct {
	Operation          WaiverOperationAction `json:"operation"`
	Scope              ScopeSelector         `json:"scope_selector"`
	Justification      string                `json:"justification"`
	Approver           string                `json:"approver"`
	Timestamp          string                `json:"timestamp"`
	ChallengeNonce     string                `json:"challenge_nonce"`
	RepositoryIdentity string                `json:"repository_identity,omitempty"`
	BaseRevision       string                `json:"base_revision,omitempty"`
	Head               string                `json:"head,omitempty"`
	DiffDigest         string                `json:"diff_digest,omitempty"`
}

// FindingDispositionOverride re-dispositions one finding. RepositoryIdentity
// and BaseRevision are optional and enforced when present, exactly as on
// WaiverOperation.
type FindingDispositionOverride struct {
	FindingID          string             `json:"finding_id"`
	Disposition        FindingDisposition `json:"disposition"`
	Justification      string             `json:"justification"`
	Approver           string             `json:"approver"`
	Timestamp          string             `json:"timestamp"`
	ChallengeNonce     string             `json:"challenge_nonce"`
	RepositoryIdentity string             `json:"repository_identity,omitempty"`
	BaseRevision       string             `json:"base_revision,omitempty"`
}

// Record is a portable, chained HMA record.
type Record struct {
	Kind     RecordKind        `json:"kind"`
	Version  int               `json:"version"`
	Metadata RecordMetadata    `json:"metadata"`
	Stage    Stage             `json:"stage,omitempty"`
	Control  StageControlState `json:"control,omitempty"`
	Criteria []Criterion       `json:"criteria,omitempty"`
	Findings []Finding         `json:"findings,omitempty"`
	Waivers  []Waiver          `json:"waivers,omitempty"`
	Outcome  TerminalOutcome   `json:"outcome,omitempty"`
	Approval *ApprovalBinding  `json:"approval,omitempty"`
	Evidence *EvidenceRef      `json:"evidence,omitempty"`

	WaiverOperation *WaiverOperation            `json:"waiver_operation,omitempty"`
	FindingOverride *FindingDispositionOverride `json:"finding_override,omitempty"`
}
