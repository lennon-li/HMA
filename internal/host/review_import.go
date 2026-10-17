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
// cannot yet reproduce a cited kernel rule or measure a cited budget, so a
// pass that carries one is escalated, never approved.
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
		rc.PacketSHA256 != d.ReviewPacketDigest || rc.LaunchHead != p.ExpectedLaunchHead || rc.LaunchWorktreeSHA256 != p.ExpectedLaunchWorktree ||
		rc.Agent != d.Route.WorkerID || rc.Model != d.Route.Model || rc.Provider != d.Route.Provider || rc.Effort != d.Route.ReasoningEffort ||
		!sameDir(rc.ProjectRoot, p.ProjectRoot) {
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
	if _, err := MapAuditFindings(rep.Findings); err != nil {
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
