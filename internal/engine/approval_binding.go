package engine

import (
	"fmt"
	"reflect"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonMalformedApproval Reason = "MALFORMED_APPROVAL_BINDING"
	ReasonStaleApproval     Reason = "STALE_APPROVAL_BINDING"
	ReasonReusedNonce       Reason = "REUSED_APPROVAL_NONCE"
)

type ApprovalBindingRequest struct {
	Expected        model.ApprovalBinding
	Presented       model.ApprovalBinding
	PredecessorHead string
	Sequence        int64
	UsedNonces      map[string]bool
	Now             time.Time
	MaxAge          time.Duration
}

type ApprovalBindingResult struct {
	Decision                   Decision `json:"decision"`
	Reason                     Reason   `json:"reason,omitempty"`
	RequiresFreshHumanApproval bool     `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool     `json:"machine_advanced"`
}

// EvaluateApprovalBinding validates one explicit, host-entered approval. It
// consumes and persists nothing and can never advance a stage.
func EvaluateApprovalBinding(req ApprovalBindingRequest) ApprovalBindingResult {
	reject := func(reason Reason) ApprovalBindingResult {
		return ApprovalBindingResult{Decision: DecisionRejected, Reason: reason, RequiresFreshHumanApproval: true}
	}
	a := req.Presented
	if a.RunID == "" || a.TransitionDigest == "" || !model.ValidStage(a.CurrentStage) || !model.ValidStage(a.ProposedTargetStage) || a.RepositoryIdentityDigest == "" || a.BaseRevisionDigest == "" || a.StageTimeDigest == "" || a.AcceptedPlanDigest == "" || a.ChallengeNonce == "" || a.Approver == "" || a.Timestamp == "" {
		return reject(ReasonMalformedApproval)
	}
	if req.UsedNonces[a.ChallengeNonce] {
		return reject(ReasonReusedNonce)
	}
	if a.StageTimeDigest != fmt.Sprintf("%s:%d", req.PredecessorHead, req.Sequence) || !reflect.DeepEqual(a, req.Expected) {
		return reject(ReasonStaleApproval)
	}
	ts, err := time.Parse(time.RFC3339, a.Timestamp)
	if err != nil || req.MaxAge <= 0 || req.Now.Before(ts) || req.Now.Sub(ts) > req.MaxAge {
		return reject(ReasonStaleApproval)
	}
	return ApprovalBindingResult{Decision: DecisionLegalPendingApproval, RequiresFreshHumanApproval: true, MachineAdvanced: false}
}
