package engine

import (
	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonResolutionMultipleOperations Reason = "MULTIPLE_OPERATIONS_PROVIDED"
	ReasonResolutionMissingOperation   Reason = "MISSING_OPERATION"
	ReasonResolutionTargetNotFound     Reason = "TARGET_NOT_FOUND"
	ReasonResolutionWaiveBlockFinding  Reason = "CANNOT_WAIVE_BLOCK_FINDING"
	ReasonResolutionOverrideBlock      Reason = "CANNOT_OVERRIDE_BLOCK_FINDING"
	ReasonResolutionAlreadyWaived      Reason = "SCOPE_ALREADY_WAIVED"
	ReasonResolutionNoActiveWaiver     Reason = "NO_ACTIVE_WAIVER_FOR_SCOPE"
	ReasonResolutionReusedNonce        Reason = "REUSED_RESOLUTION_NONCE"
	ReasonResolutionStalePosition      Reason = "STALE_CHAIN_POSITION"
	ReasonResolutionInvalidOperation   Reason = "INVALID_WAIVER_OPERATION"
	ReasonResolutionStaleRepository    Reason = "STALE_REPOSITORY_BINDING"
	ReasonResolutionStaleRevision      Reason = "STALE_REVISION_BINDING"
)

// ResolutionState is the projection of a run's committed chain that human
// resolution is evaluated against. It is derived by replaying the chain, so a
// waiver or override already recorded is visible to the next operation.
type ResolutionState struct {
	Criteria      []model.Criterion
	Findings      []model.Finding
	ActiveWaivers map[model.ScopeSelector]bool
	UsedNonces    map[string]bool
}

// ProjectResolutionState replays a committed chain into the state that human
// resolution operates on. It advances nothing and decides nothing; it only
// reports what the chain already says.
func ProjectResolutionState(records []model.Record) ResolutionState {
	state := ResolutionState{
		ActiveWaivers: make(map[model.ScopeSelector]bool),
		UsedNonces:    make(map[string]bool),
	}
	for i := range records {
		r := records[i]
		if len(r.Criteria) > 0 {
			state.Criteria = r.Criteria
		}
		if len(r.Findings) > 0 {
			state.Findings = r.Findings
		}
		for _, w := range r.Waivers {
			if w.Active {
				state.ActiveWaivers[w.Scope] = true
			} else {
				delete(state.ActiveWaivers, w.Scope)
			}
		}
		if r.Approval != nil && r.Approval.ChallengeNonce != "" {
			state.UsedNonces[r.Approval.ChallengeNonce] = true
		}
		if w := r.WaiverOperation; w != nil {
			if w.ChallengeNonce != "" {
				state.UsedNonces[w.ChallengeNonce] = true
			}
			switch w.Operation {
			case model.WaiverOperationGrant:
				state.ActiveWaivers[w.Scope] = true
			case model.WaiverOperationExpire, model.WaiverOperationWithdraw:
				delete(state.ActiveWaivers, w.Scope)
			}
		}
		if f := r.FindingOverride; f != nil {
			if f.ChallengeNonce != "" {
				state.UsedNonces[f.ChallengeNonce] = true
			}
			for j := range state.Findings {
				if state.Findings[j].ID == f.FindingID {
					updated := state.Findings[j]
					updated.Disposition = f.Disposition
					if updated.Disposition != model.FindingAdvisory {
						updated.Advisory = ""
					}
					state.Findings[j] = updated
					break
				}
			}
		}
	}
	return state
}

// ResolutionRequest encapsulates the current chain state and the proposed human resolution operation.
type ResolutionRequest struct {
	// Only one of these may be provided per request
	WaiverOperation *model.WaiverOperation
	FindingOverride *model.FindingDispositionOverride

	CurrentCriteria []model.Criterion
	CurrentFindings []model.Finding

	// ActiveWaivers and UsedNonces come from the projected chain. A nil
	// UsedNonces map disables replay rejection, so callers holding a chain
	// must always populate it.
	ActiveWaivers map[model.ScopeSelector]bool
	UsedNonces    map[string]bool

	// ExpectedPredecessorHead, when non-empty, binds the operation to one
	// exact chain position. PredecessorHead is the live chain tail.
	ExpectedPredecessorHead string
	PredecessorHead         string

	// RepositoryIdentity and BaseRevision are the live values the host
	// observes. They are compared against whatever the operation itself
	// declares; an operation that declares neither is not revision-bound,
	// which the architecture permits.
	RepositoryIdentity string
	BaseRevision       string
}

// NewResolutionRequest builds a request from a projected chain state.
func NewResolutionRequest(state ResolutionState) ResolutionRequest {
	return ResolutionRequest{
		CurrentCriteria: state.Criteria,
		CurrentFindings: state.Findings,
		ActiveWaivers:   state.ActiveWaivers,
		UsedNonces:      state.UsedNonces,
	}
}

// ResolutionResult returns the validation decision for the human resolution.
// Because it evaluates human intent that alters the record stream (but never the stage),
// it mirrors the design of the other engine evaluators.
type ResolutionResult struct {
	Decision        Decision `json:"decision"`
	Reason          Reason   `json:"reason,omitempty"`
	MachineAdvanced bool     `json:"machine_advanced"`
}

func rejectResolution(reason Reason) ResolutionResult {
	return ResolutionResult{Decision: DecisionRejected, Reason: reason, MachineAdvanced: false}
}

// EvaluateResolution evaluates a single human resolution operation against the
// projected chain state. It ensures the target exists, that BLOCK findings are
// not illegally waived or overridden, that a waiver is not granted twice or
// withdrawn when none is active, and that the operation's challenge nonce has
// not already been used in this run. It never advances the stage.
func EvaluateResolution(req ResolutionRequest) ResolutionResult {
	if req.WaiverOperation != nil && req.FindingOverride != nil {
		return rejectResolution(ReasonResolutionMultipleOperations)
	}
	if req.WaiverOperation == nil && req.FindingOverride == nil {
		return rejectResolution(ReasonResolutionMissingOperation)
	}
	if req.ExpectedPredecessorHead != "" && req.ExpectedPredecessorHead != req.PredecessorHead {
		return rejectResolution(ReasonResolutionStalePosition)
	}

	var nonce, boundRepository, boundRevision string
	if req.WaiverOperation != nil {
		nonce = req.WaiverOperation.ChallengeNonce
		boundRepository = req.WaiverOperation.RepositoryIdentity
		boundRevision = req.WaiverOperation.BaseRevision
	} else {
		nonce = req.FindingOverride.ChallengeNonce
		boundRepository = req.FindingOverride.RepositoryIdentity
		boundRevision = req.FindingOverride.BaseRevision
	}
	if boundRepository != "" && boundRepository != req.RepositoryIdentity {
		return rejectResolution(ReasonResolutionStaleRepository)
	}
	if boundRevision != "" && boundRevision != req.BaseRevision {
		return rejectResolution(ReasonResolutionStaleRevision)
	}
	if nonce != "" && req.UsedNonces[nonce] {
		return rejectResolution(ReasonResolutionReusedNonce)
	}

	if w := req.WaiverOperation; w != nil {
		switch w.Operation {
		case model.WaiverOperationGrant, model.WaiverOperationExpire, model.WaiverOperationWithdraw:
		default:
			return rejectResolution(ReasonResolutionInvalidOperation)
		}
		granting := w.Operation == model.WaiverOperationGrant
		waiverActive := req.ActiveWaivers[w.Scope]

		if granting {
			targetFound := false
			switch w.Scope.Kind {
			case model.ScopeCriterion:
				for _, c := range req.CurrentCriteria {
					if c.ID == w.Scope.Target {
						targetFound = true
						break
					}
				}
			case model.ScopeFinding:
				for _, f := range req.CurrentFindings {
					if f.ID == w.Scope.Target {
						targetFound = true
						if f.Disposition == model.FindingBlock {
							return rejectResolution(ReasonResolutionWaiveBlockFinding)
						}
						break
					}
				}
			case model.ScopeArtifact:
				// Artifacts are not enumerable in the projected state; the
				// host binds artifact identity outside this evaluator.
				targetFound = true
			}
			if !targetFound {
				return rejectResolution(ReasonResolutionTargetNotFound)
			}
			if waiverActive {
				return rejectResolution(ReasonResolutionAlreadyWaived)
			}
		} else if !waiverActive {
			return rejectResolution(ReasonResolutionNoActiveWaiver)
		}
	}

	if f := req.FindingOverride; f != nil {
		targetFound := false
		var currentDisposition model.FindingDisposition
		for _, curF := range req.CurrentFindings {
			if curF.ID == f.FindingID {
				targetFound = true
				currentDisposition = curF.Disposition
				break
			}
		}
		if !targetFound {
			return rejectResolution(ReasonResolutionTargetNotFound)
		}
		if currentDisposition == model.FindingBlock {
			return rejectResolution(ReasonResolutionOverrideBlock)
		}
	}

	return ResolutionResult{
		Decision:        DecisionLegalPendingApproval, // It's valid, can be appended to JSONL
		MachineAdvanced: false,
	}
}
