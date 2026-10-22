package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
)

// Receipt is the part of an agent-dispatch run receipt that review import
// checks. Under the trusted-host model the receipt is host-written: the
// checks detect drift and accident (a wrong run, packet, target, or report),
// not a forging host.
type Receipt struct {
	RunID                string       `json:"run_id"`
	Agent                string       `json:"agent"`
	Model                string       `json:"model"`
	Provider             string       `json:"provider"`
	Effort               string       `json:"effort"`
	StartedAt            string       `json:"started_at"`
	ProjectRoot          string       `json:"project_root"`
	Mode                 string       `json:"mode"`
	State                string       `json:"state"`
	ExitCode             *int         `json:"exit_code"`
	PacketSHA256         string       `json:"packet_sha256"`
	LaunchHead           string       `json:"launch_head"`
	LaunchWorktreeSHA256 string       `json:"launch_worktree_sha256"`
	ReportFiles          []ReportFile `json:"report_files"`
}

// LaunchIdentity is what a ledger receipt says, from launch on, about who
// ran and against what: the actual route, the packet, and the target.
type LaunchIdentity struct {
	Agent                string `json:"agent"`
	Model                string `json:"model"`
	Provider             string `json:"provider"`
	Effort               string `json:"effort"`
	PacketSHA256         string `json:"packet_sha256"`
	ProjectRoot          string `json:"project_root"`
	LaunchHead           string `json:"launch_head"`
	LaunchWorktreeSHA256 string `json:"launch_worktree_sha256"`
}

// Launch returns the receipt's launch identity.
func (rc Receipt) Launch() LaunchIdentity {
	return LaunchIdentity{Agent: rc.Agent, Model: rc.Model, Provider: rc.Provider, Effort: rc.Effort, PacketSHA256: rc.PacketSHA256,
		ProjectRoot: rc.ProjectRoot, LaunchHead: rc.LaunchHead, LaunchWorktreeSHA256: rc.LaunchWorktreeSHA256}
}

// ReviewLaunchMatches reports whether a launch is exactly the one the
// HMA-generated review dispatch d authorizes: its route (worker, model,
// provider, effort), its packet, and its target (root, head, worktree).
// Review import and launch reconciliation share it.
func ReviewLaunchMatches(d *Dispatch, l LaunchIdentity) bool {
	p := d.Packet
	return p != nil && l.Agent == d.Route.WorkerID && l.Model == d.Route.Model && l.Provider == d.Route.Provider && l.Effort == d.Route.ReasoningEffort &&
		l.PacketSHA256 == d.ReviewPacketDigest && l.LaunchHead == p.ExpectedLaunchHead && l.LaunchWorktreeSHA256 == p.ExpectedLaunchWorktree &&
		sameDir(l.ProjectRoot, p.ProjectRoot)
}

// ReportFile is one report path the dispatcher saw written during the run.
type ReportFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ReportBinding is the hma_binding object a reviewer copies from the packet
// into its report.
type ReportBinding struct {
	PacketSHA256     string `json:"packet_sha256"`
	ReviewDispatchID string `json:"review_dispatch_id"`
	ReviewNonce      string `json:"review_nonce"`
}

// auditReport is the part of an independent-audit report import reads.
type auditReport struct {
	Verdict      *string         `json:"verdict"`
	Isolation    string          `json:"isolation"`
	HMABinding   *ReportBinding  `json:"hma_binding"`
	Findings     json.RawMessage `json:"findings"`
	Requirements json.RawMessage `json:"requirements"`
}

// citesHMARule reports whether any finding carries an hma_ref citation. HMA
// believes no citation: a pass that carries one is escalated, never approved,
// and a finding's disposition comes only from what HMA itself established.
func citesHMARule(raw json.RawMessage) bool {
	var fs []struct {
		HMARef json.RawMessage `json:"hma_ref"`
	}
	if UnmarshalExact(raw, &fs) != nil {
		return true
	}
	for _, f := range fs {
		if len(f.HMARef) > 0 && string(f.HMARef) != "null" {
			return true
		}
	}
	return false
}

// ReviewImport is the evidence for one review import.
type ReviewImport struct {
	ReviewDispatchID string
	ReceiptBytes     []byte
	ReportPath       string
	ReportBytes      []byte
}

// MapReportVerdict is the total verdict map (Gate A v5 §3). Only pass
// approves; anything not named, including a missing verdict, escalates.
func MapReportVerdict(v *string) ReviewVerdict {
	if v == nil {
		return ReviewEscalate
	}
	switch *v {
	case "pass":
		return ReviewApprove
	case "fix-required":
		return ReviewRequestChanges
	}
	return ReviewEscalate
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// reviewDispatchByID returns the HMA-generated review dispatch record with
// this review dispatch id.
func reviewDispatchByID(records []Record, id string) *Record {
	for i := range records {
		r := &records[i]
		if r.Kind == KindDispatch && r.Dispatch != nil && r.Dispatch.Role == RoleReview && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == id {
			return r
		}
	}
	return nil
}

// EvaluateReviewImport checks an imported review against the HMA-generated
// review dispatch it answers and returns the review record to append. It
// appends nothing itself; any failed check returns an error and no review.
func EvaluateReviewImport(records []Record, in ReviewImport) (IndependentReview, error) {
	dr := reviewDispatchByID(records, in.ReviewDispatchID)
	if dr == nil {
		return IndependentReview{}, errors.New(ReasonReviewMissing)
	}
	d := dr.Dispatch
	p := d.Packet
	// Only the unit's latest review dispatch is answerable, and only once:
	// an earlier or already-answered packet cannot be replayed to put a new
	// review at the chain tail.
	for _, r := range records {
		if r.Sequence > dr.Sequence && r.Kind == KindDispatch && r.Dispatch != nil && r.Dispatch.Role == RoleReview && r.Dispatch.UnitID == d.UnitID {
			return IndependentReview{}, errors.New(ReasonReviewPacketMismatch)
		}
		if r.Kind == KindReview && r.Review != nil && r.Review.ReviewDispatchDigest == dr.HeadAnchor {
			return IndependentReview{}, errors.New(ReasonReviewPacketMismatch)
		}
	}
	current, err := RequireImplementation(records, d.UnitID)
	if err != nil {
		return IndependentReview{}, err
	}
	if current != d.ImplementationRecordDigest {
		return IndependentReview{}, errors.New(ReasonImplementationAlreadyBound)
	}

	var rc Receipt
	if err := UnmarshalExact(in.ReceiptBytes, &rc); err != nil {
		return IndependentReview{}, errors.New(ReasonReviewReceiptMismatch)
	}
	if rc.RunID != p.DispatchRunID || rc.Mode != "review" || rc.State != "completed" || rc.ExitCode == nil || *rc.ExitCode != 0 ||
		!ReviewLaunchMatches(d, rc.Launch()) {
		return IndependentReview{}, errors.New(ReasonReviewReceiptMismatch)
	}
	// Launch must fall in the packet's half-open validity interval.
	started, err := parseTime(rc.StartedAt)
	from, ferr := parseTime(dr.Timestamp)
	until, uerr := parseTime(p.ValidUntil)
	if err != nil || ferr != nil || uerr != nil || started.Before(from) || !started.Before(until) {
		return IndependentReview{}, errors.New(ReasonReviewReceiptMismatch)
	}

	// The report must be a file the dispatcher saw written during this run,
	// with exactly these bytes.
	reportSum := sha256Hex(in.ReportBytes)
	listed := false
	for _, f := range rc.ReportFiles {
		if f.Path == in.ReportPath && f.SHA256 == reportSum {
			listed = true
		}
	}
	if !listed {
		return IndependentReview{}, errors.New(ReasonReviewReportMismatch)
	}
	// One JSON document only, read with exact keys as the Python report
	// validator reads it: a case-variant duplicate such as "Verdict" is
	// ignored, never preferred.
	var rep auditReport
	if err := UnmarshalExact(in.ReportBytes, &rep); err != nil {
		return IndependentReview{}, errors.New(ReasonReviewReportMismatch)
	}
	// Findings are validated here, so a later findings import of this exact
	// report cannot fail on them.
	// A report that breaks the independent-audit report contract is recorded
	// but escalated: it cannot approve or clear findings.
	contractValid := len(ValidateAuditReport(in.ReportBytes)) == 0
	if _, err := MapAuditFindings(rep.Findings, FindingFacts{}); err != nil {
		return IndependentReview{}, err
	}
	// The nonce is fresh per packet: a previous report, rewritten or not,
	// carries another one.
	if rep.HMABinding == nil || rep.HMABinding.PacketSHA256 != d.ReviewPacketDigest ||
		rep.HMABinding.ReviewDispatchID != p.ReviewDispatchID || rep.HMABinding.ReviewNonce != p.ReviewNonce {
		return IndependentReview{}, errors.New(ReasonReviewReportMismatch)
	}
	switch rep.Isolation {
	case "isolated", "partial", "compromised":
	default:
		return IndependentReview{}, errors.New(ReasonReviewReportMismatch)
	}
	verdict := MapReportVerdict(rep.Verdict)
	if !contractValid && verdict != ReviewApprove {
		verdict = ReviewEscalate
	}
	if verdict == ReviewApprove && rep.Isolation != "isolated" && (p.IssuingReason == IssueFinal || d.FinalValidation) {
		return IndependentReview{}, errors.New(ReasonReviewIsolationInsufficient)
	}
	if verdict == ReviewApprove && (!contractValid || citesHMARule(rep.Findings)) {
		verdict = ReviewEscalate
	}
	raw := ""
	if rep.Verdict != nil {
		raw = *rep.Verdict
	}
	if raw == "" {
		raw = "missing"
	}
	findings := "sha256:" + sha256Hex(rep.Findings)
	return IndependentReview{
		UnitID:                     d.UnitID,
		ImplementationRecordDigest: d.ImplementationRecordDigest,
		ReviewDispatchDigest:       dr.HeadAnchor,
		ArtifactDigest:             d.ArtifactDigest,
		ReviewPacketDigest:         d.ReviewPacketDigest,
		ReviewerID:                 d.Route.WorkerID,
		Verdict:                    verdict,
		FindingsDigest:             findings,
		ReceiptDigest:              "sha256:" + sha256Hex(in.ReceiptBytes),
		ReportDigest:               "sha256:" + reportSum,
		Isolation:                  rep.Isolation,
		ReportVerdict:              raw,
	}, nil
}

// sameDir reports whether two paths name the same existing directory.
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false
	}
	rb, err := filepath.EvalSymlinks(b)
	return err == nil && ra == rb
}
