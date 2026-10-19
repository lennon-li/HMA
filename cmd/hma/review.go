package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// hostReviewPacketInput asks HMA to generate a review packet. The dispatch is
// a review dispatch without packet binding or packet digest; HMA adds both.
type hostReviewPacketInput struct {
	hostEnvelope
	Dispatch       hoststate.Dispatch      `json:"dispatch"`
	DispatchRunID  string                  `json:"dispatch_run_id"`
	IssuingReason  hoststate.IssuingReason `json:"issuing_reason"`
	RepositoryRoot string                  `json:"repository_root"`
	BaseRevision   string                  `json:"base_revision"`
	// DirtyWorktreeApproved permits reviewing uncommitted content; its exact
	// bytes are then part of the artifact.
	DirtyWorktreeApproved bool   `json:"dirty_worktree_approved"`
	Task                  string `json:"task"`
	// LedgerDir is the agent-dispatch ledger the reviewer will launch in;
	// review import reads the receipt only from it, and reconciliation
	// scans it.
	LedgerDir string `json:"ledger_dir"`
}

type hostReviewImportInput struct {
	hostEnvelope
	ReviewDispatchID string `json:"review_dispatch_id"`
	LedgerDir        string `json:"ledger_dir"`
	ReportPath       string `json:"report_path"`
}

type reviewPacketResult struct {
	hostResult
	PacketPath       string `json:"packet_path"`
	PacketSHA256     string `json:"packet_sha256"`
	ReviewDispatchID string `json:"review_dispatch_id"`
	DispatchRunID    string `json:"dispatch_run_id"`
}

type reviewImportResult struct {
	hostResult
	Verdict hoststate.ReviewVerdict `json:"verdict"`
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildReviewPacket renders the packet the reviewer receives. Its header is
// the binding the report must copy back; the packet digest itself is the
// SHA-256 of these bytes, which the reviewer reads from the dispatch receipt
// instruction rather than from the packet.
// packetCoreState is the bound HMA state a review packet carries, so the
// reviewer sees the approved plan, criteria, current findings, and waivers it
// reviews against, not only the caller's task text.
type packetCoreState struct {
	CoreHeadAnchor     string                `json:"core_head_anchor"`
	AcceptedPlanDigest string                `json:"accepted_plan_digest"`
	RouteApproval      string                `json:"route_approval_digest"`
	Criteria           []model.Criterion     `json:"criteria"`
	Findings           []model.Finding       `json:"findings"`
	ActiveWaivers      []model.ScopeSelector `json:"active_waivers"`
}

// corePacketState assembles the packet's bound core state for unit. It
// refuses when a mandatory input is missing: no approved route for the unit,
// or no accepted plan digest on that approval.
func corePacketState(core []model.Record, runID, unit string) (packetCoreState, error) {
	route, err := approvedCoreRouteDigestIn(core, runID, unit)
	if err != nil {
		return packetCoreState{}, err
	}
	var plan string
	for _, r := range core {
		if r.UnitID == unit && r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil &&
			r.Approval.ProposedTargetStage == model.StageImplementationAuthorization && r.Approval.TransitionDigest == route {
			plan = r.Approval.AcceptedPlanDigest
		}
	}
	if plan == "" {
		return packetCoreState{}, errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	res := engine.ProjectResolutionState(core)
	st := packetCoreState{CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor, AcceptedPlanDigest: plan, RouteApproval: route,
		Criteria: res.Criteria, Findings: []model.Finding{}, ActiveWaivers: []model.ScopeSelector{}}
	for _, f := range hoststate.AssignLegacyOwners(core, res.Findings) {
		if f.UnitID == unit || f.UnitID == "" {
			st.Findings = append(st.Findings, f)
		}
	}
	for w, on := range res.ActiveWaivers {
		if on {
			st.ActiveWaivers = append(st.ActiveWaivers, w)
		}
	}
	sort.Slice(st.ActiveWaivers, func(i, j int) bool {
		return fmt.Sprint(st.ActiveWaivers[i]) < fmt.Sprint(st.ActiveWaivers[j])
	})
	return st, nil
}

func buildReviewPacket(in hostReviewPacketInput, binding hoststate.ReviewPacketBinding, state packetCoreState) []byte {
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	return []byte(fmt.Sprintf(`HMA review packet
review_dispatch_id: %s
review_nonce: %s
dispatch_run_id: %s
unit_id: %s
issuing_reason: %s
artifact_digest: %s
implementation_record_digest: %s
base_revision: %s
produced_head: %s
diff_digest: %s
worktree_digest: %s
valid_until: %s

Report binding (required): your AUDIT.json must contain a top-level object
"hma_binding": {"packet_sha256": <SHA-256 hex of this packet file's bytes>,
"review_dispatch_id": %q, "review_nonce": %q}.
A report without this exact binding is not imported.

Bound HMA state (review against it; it is part of this packet's digest):
%s

Task:
%s
`, binding.ReviewDispatchID, binding.ReviewNonce, binding.DispatchRunID, in.Dispatch.UnitID, binding.IssuingReason,
		in.Dispatch.ArtifactDigest, in.Dispatch.ImplementationRecordDigest, binding.BaseRevision, binding.ProducedHead,
		binding.DiffDigest, binding.WorktreeDigest, binding.ValidUntil,
		binding.ReviewDispatchID, binding.ReviewNonce, stateJSON, in.Task))
}

func runHostReviewPacket(args []string) error {
	fs := flag.NewFlagSet("review-packet", flag.ContinueOnError)
	input := fs.String("input", "", "host input file")
	storePath := fs.String("store", "", "HMA store directory")
	out := fs.String("out", "", "packet file to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" || *storePath == "" || *out == "" || fs.NArg() != 0 {
		return errors.New("usage: hma host review-packet --input <file> --store <directory> --out <packet file>")
	}
	var in hostReviewPacketInput
	if err := decodeStrictFile(*input, &in); err != nil {
		return err
	}
	at, err := validateHostEnvelope(in.hostEnvelope)
	if err != nil {
		return err
	}
	if in.Dispatch.Role != hoststate.RoleReview || in.Dispatch.Packet != nil || in.Dispatch.ReviewPacketDigest != "" || in.Task == "" || in.RepositoryRoot == "" || in.BaseRevision == "" || in.LedgerDir == "" || !filepath.IsAbs(in.LedgerDir) {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	root, err := filepath.Abs(in.RepositoryRoot)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return err
	}
	// The packet must not change the worktree the receipt will describe.
	if err := repostate.StoreOutsideRepository(root, filepath.Dir(*out)); err != nil {
		return fmt.Errorf("packet path: %w", err)
	}
	// The packet must review the approved base in the run's repository root;
	// checked first, so a wrong root or base fails with its named reason.
	if core, err := store.New(*storePath).Load(in.RunID); err != nil {
		return err
	} else if history, err := hoststate.NewStore(*storePath).Load(in.RunID); err != nil {
		return err
	} else if len(core) > 0 {
		if err := requireApprovedRepository(history, core, in.RunID, in.Dispatch.UnitID, &hoststate.ReviewPacketBinding{ProjectRoot: root, BaseRevision: in.BaseRevision}); err != nil {
			return err
		}
	}
	// HMA observes the artifact itself; the host may name the digest it
	// expects, but never supplies it.
	snap, err := repostate.Snapshot(context.Background(), root, in.BaseRevision, in.DirtyWorktreeApproved)
	if err != nil {
		return err
	}
	artifact := hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree)
	if in.Dispatch.ArtifactDigest != "" && in.Dispatch.ArtifactDigest != artifact {
		return errors.New(hoststate.ReasonReviewArtifactMismatch)
	}
	in.Dispatch.ArtifactDigest = artifact
	head, worktree, err := repostate.LaunchDigests(context.Background(), root)
	if err != nil {
		return err
	}
	if head != snap.Head {
		return errors.New(hoststate.ReasonReviewArtifactMismatch)
	}
	nonce, err := randomHex(16)
	if err != nil {
		return err
	}
	id, err := randomHex(16)
	if err != nil {
		return err
	}
	binding := hoststate.ReviewPacketBinding{
		ReviewDispatchID: "hma-rd-" + id, ReviewNonce: nonce, DispatchRunID: in.DispatchRunID,
		IssuingReason: in.IssuingReason, ExpectedLaunchHead: head, ExpectedLaunchWorktree: worktree,
		ProjectRoot: root, BaseRevision: in.BaseRevision, ProducedHead: snap.Head, DiffDigest: snap.Diff, WorktreeDigest: snap.Worktree,
		ValidUntil:      at.Add(hoststate.ReviewDispatchTTL).UTC().Format(time.RFC3339),
		LedgerNamespace: filepath.Clean(in.LedgerDir),
	}
	core, err := store.New(*storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	if len(core) == 0 {
		return errors.New(hoststate.ReasonCoreRunMissing)
	}
	state, err := corePacketState(core, in.RunID, in.Dispatch.UnitID)
	if err != nil {
		return err
	}
	history, err := hoststate.NewStore(*storePath).Load(in.RunID)
	if err != nil {
		return err
	}
	// Checked before the packet file is written, and again under the locks.
	if err := requireApprovedRepository(history, core, in.RunID, in.Dispatch.UnitID, &binding); err != nil {
		return err
	}
	packet := buildReviewPacket(in, binding, state)
	sum := sha256.Sum256(packet)
	in.Dispatch.Packet = &binding
	in.Dispatch.ReviewPacketDigest = hex.EncodeToString(sum[:])

	// Create the packet first, exclusively; remove it if the record is not
	// appended, so no packet exists without its dispatch record.
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(packet)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = syncParent(*out)
	}
	if err != nil {
		os.Remove(*out)
		return err
	}
	result, err := appendHostRecord(*storePath, in.hostEnvelope, hoststate.KindDispatch, nil, nil, &in.Dispatch, nil, nil, func(history []hoststate.Record, core []model.Record) error {
		if core[len(core)-1].Metadata.HeadAnchor != state.CoreHeadAnchor {
			return errors.New(hoststate.ReasonCoreBindingStale)
		}
		if err := validateRouteApprovalClaimIn(core, in.RunID, in.Dispatch.UnitID, in.Dispatch.Route.RouteApprovalDigest); err != nil {
			return err
		}
		if err := requireApprovedRepository(history, core, in.RunID, in.Dispatch.UnitID, in.Dispatch.Packet); err != nil {
			return err
		}
		return hoststate.EvaluateDispatch(history, in.Dispatch, at)
	})
	if err != nil {
		os.Remove(*out)
		return err
	}
	result.DispatchAllowed = true
	result.NextAction = "START_REVIEWER"
	return json.NewEncoder(os.Stdout).Encode(reviewPacketResult{
		hostResult: result, PacketPath: *out, PacketSHA256: in.Dispatch.ReviewPacketDigest,
		ReviewDispatchID: binding.ReviewDispatchID, DispatchRunID: binding.DispatchRunID,
	})
}

func runHostReviewImport(args []string) error {
	inputPath, storePath, err := hostCommandFlags("review-import", args)
	if err != nil {
		return err
	}
	var in hostReviewImportInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	if in.ReviewDispatchID == "" || in.LedgerDir == "" || in.ReportPath == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	var review hoststate.IndependentReview
	result, err := appendHostRecord(storePath, in.hostEnvelope, hoststate.KindReview, nil, nil, nil, &review, nil, func(history []hoststate.Record, core []model.Record) error {
		var runID string
		for _, r := range history {
			if r.Kind == hoststate.KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == in.ReviewDispatchID {
				runID = r.Dispatch.Packet.DispatchRunID
				// Any core record since the packet (rewind, terminal,
				// other findings) makes the packet stale.
				if r.CoreHeadAnchor != core[len(core)-1].Metadata.HeadAnchor {
					return errors.New(hoststate.ReasonCoreBindingStale)
				}
			}
		}
		if runID == "" {
			return errors.New(hoststate.ReasonReviewMissing)
		}
		// HMA reads the receipt from the ledger by run id, not from a path
		// the host names.
		// The receipt must come from the ledger the packet named.
		if p := packetFor(history, findDispatchDigest(history, in.ReviewDispatchID)); p != nil && p.LedgerNamespace != "" && filepath.Clean(in.LedgerDir) != p.LedgerNamespace {
			return errors.New(hoststate.ReasonReviewReceiptMismatch)
		}
		receipt, err := os.ReadFile(filepath.Join(in.LedgerDir, "runs", runID, "receipt.json"))
		if err != nil {
			return errors.New(hoststate.ReasonReviewReceiptMismatch)
		}
		report, err := os.ReadFile(in.ReportPath)
		if err != nil {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		if err := requireLiveTarget(history, findDispatchDigest(history, in.ReviewDispatchID)); err != nil {
			return err
		}
		if p := packetFor(history, findDispatchDigest(history, in.ReviewDispatchID)); p == nil {
			return errors.New(hoststate.ReasonReviewMissing)
		} else if err := requireApprovedRepository(history, core, in.RunID, reviewUnit(history, in.ReviewDispatchID), p); err != nil {
			return err
		}
		imported, err := hoststate.EvaluateReviewImport(history, hoststate.ReviewImport{
			ReviewDispatchID: in.ReviewDispatchID, ReceiptBytes: receipt, ReportPath: in.ReportPath, ReportBytes: report,
		})
		if err != nil {
			return err
		}
		if _, err := approvedCoreRouteDigestIn(core, in.RunID, imported.UnitID); err != nil {
			return err
		}
		if err := hoststate.EvaluateReview(history, imported); err != nil {
			return err
		}
		review = imported
		return nil
	})
	if err != nil {
		return err
	}
	switch review.Verdict {
	case hoststate.ReviewApprove:
		result.NextAction = "CONTINUE_HMA_TRANSITION"
	case hoststate.ReviewRequestChanges:
		result.NextAction = "RECORD_FINDINGS_AND_REMEDIATE"
	default:
		result.NextAction = "ESCALATE_TO_HUMAN"
	}
	return json.NewEncoder(os.Stdout).Encode(reviewImportResult{hostResult: result, Verdict: review.Verdict})
}

type hostFindingsImportInput struct {
	hostEnvelope
	ReviewRecordDigest string `json:"review_record_digest"`
	ReportPath         string `json:"report_path"`
}

type findingsImportResult struct {
	Recorded       bool            `json:"recorded"`
	CoreSequence   int64           `json:"core_sequence"`
	CoreHeadAnchor string          `json:"core_head_anchor"`
	Findings       []model.Finding `json:"findings"`
	NextAction     string          `json:"next_action"`
	// ReviewEnforcement is the run's review enforcement mode, as on every
	// host write result.
	ReviewEnforcement string `json:"review_enforcement"`
}

// runHostFindingsImport writes one core findings snapshot from the report of
// an imported review. It holds the host run lock while it reads the host
// chain and appends to the core chain, which takes the core lock (host, then
// core); the core append refuses if the core head moved since it was read.
func runHostFindingsImport(args []string) error {
	inputPath, storePath, err := hostCommandFlags("findings-import", args)
	if err != nil {
		return err
	}
	var in hostFindingsImportInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	if in.ReviewRecordDigest == "" || in.ReportPath == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	var out findingsImportResult
	var verdict hoststate.ReviewVerdict
	err = hoststate.NewStore(storePath).WithRunLock(in.RunID, func(history []hoststate.Record) error {
		rr, err := hoststate.RequireLatestImportedReview(history, in.ReviewRecordDigest)
		if err != nil {
			return err
		}
		report, err := os.ReadFile(in.ReportPath)
		if err != nil {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		sum := sha256.Sum256(report)
		if "sha256:"+hex.EncodeToString(sum[:]) != rr.Review.ReportDigest {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		var rep struct {
			Findings json.RawMessage `json:"findings"`
		}
		if err := hoststate.UnmarshalExact(report, &rep); err != nil {
			return errors.New(hoststate.ReasonReviewReportMismatch)
		}
		imported, err := hoststate.MapAuditFindings(rep.Findings)
		if err != nil {
			return err
		}
		s := store.New(storePath)
		core, err := s.Load(in.RunID)
		if err != nil {
			return err
		}
		if len(core) == 0 {
			return errors.New(hoststate.ReasonCoreRunMissing)
		}
		if engine.ProjectStageState(core).Terminal {
			return errors.New(hoststate.ReasonCoreRunTerminal)
		}
		last := core[len(core)-1]
		// The review must bind the current core head: a core record since
		// the review (including an earlier snapshot of it) makes it stale,
		// so an old review cannot replace newer findings.
		if rr.CoreHeadAnchor != last.Metadata.HeadAnchor {
			return errors.New(hoststate.ReasonCoreBindingStale)
		}
		if err := requireLiveTarget(history, rr.Review.ReviewDispatchDigest); err != nil {
			return err
		}
		if p := packetFor(history, rr.Review.ReviewDispatchDigest); p == nil {
			return errors.New(hoststate.ReasonReviewMissing)
		} else if err := requireApprovedRepository(history, core, in.RunID, rr.Review.UnitID, p); err != nil {
			return err
		}
		prior := hoststate.AssignLegacyOwners(core, engine.ProjectResolutionState(core).Findings)
		rec := model.Record{
			Kind: model.KindFinding, Version: 1, UnitID: rr.Review.UnitID,
			Findings:         hoststate.SnapshotFindings(prior, rr.Review.UnitID, imported, rr.Review.Verdict),
			FindingsSnapshot: true,
			FindingsSource:   &model.FindingsSource{ReviewRecordDigest: rr.HeadAnchor, ReportDigest: rr.Review.ReportDigest},
			Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: last.Metadata.Sequence + 1,
				PredecessorHash: last.Metadata.HeadAnchor, Timestamp: in.Timestamp, Actor: in.Actor},
		}
		ids := map[string]bool{}
		for _, f := range rec.Findings {
			if ids[f.ID] {
				return errors.New(hoststate.ReasonFindingsImportInvalid)
			}
			ids[f.ID] = true
		}
		if err := s.Append(&rec); err != nil {
			return err
		}
		out = findingsImportResult{Recorded: true, CoreSequence: rec.Metadata.Sequence, CoreHeadAnchor: rec.Metadata.HeadAnchor, Findings: rec.Findings,
			ReviewEnforcement: hoststate.ReviewEnforcementMode(history)}
		verdict = rr.Review.Verdict
		return nil
	})
	if err != nil {
		return err
	}
	// The review verdict decides first (total map): only an approving
	// review with no remaining blocker recommends continuing.
	switch verdict {
	case hoststate.ReviewApprove:
		out.NextAction = "CONTINUE_HMA_TRANSITION"
		for _, f := range out.Findings {
			if f.Disposition != model.FindingAdvisory {
				out.NextAction = "REMEDIATE_OR_ESCALATE"
			}
		}
	case hoststate.ReviewRequestChanges:
		out.NextAction = "RECORD_FINDINGS_AND_REMEDIATE"
	default:
		out.NextAction = "ESCALATE_TO_HUMAN"
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

// syncParent makes the directory entry of path durable.
func syncParent(path string) error {
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func findDispatchDigest(history []hoststate.Record, reviewDispatchID string) string {
	for _, r := range history {
		if r.Kind == hoststate.KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == reviewDispatchID {
			return r.HeadAnchor
		}
	}
	return ""
}

// requireLiveTarget re-snapshots the repository a packet bound and refuses
// unless head, diff, and worktree still equal the packet's artifact, so a
// review of a historical target never resolves current findings.
func requireLiveTarget(history []hoststate.Record, dispatchDigest string) error {
	var p *hoststate.ReviewPacketBinding
	for _, r := range history {
		if r.HeadAnchor == dispatchDigest && r.Dispatch != nil {
			p = r.Dispatch.Packet
		}
	}
	if p == nil {
		return errors.New(hoststate.ReasonReviewMissing)
	}
	snap, err := repostate.Snapshot(context.Background(), p.ProjectRoot, p.BaseRevision, p.WorktreeDigest != "")
	if err != nil || snap.Head != p.ProducedHead || snap.Diff != p.DiffDigest || snap.Worktree != p.WorktreeDigest {
		return errors.New(hoststate.ReasonReviewTargetChanged)
	}
	return nil
}

// packetFor returns the packet binding of the dispatch record with this
// digest, or nil.
func packetFor(history []hoststate.Record, dispatchDigest string) *hoststate.ReviewPacketBinding {
	for _, r := range history {
		if r.HeadAnchor == dispatchDigest && r.Dispatch != nil {
			return r.Dispatch.Packet
		}
	}
	return nil
}

// reviewUnit returns the unit of the HMA-generated review dispatch with this
// review dispatch id.
func reviewUnit(history []hoststate.Record, reviewDispatchID string) string {
	for _, r := range history {
		if r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.ReviewDispatchID == reviewDispatchID {
			return r.Dispatch.UnitID
		}
	}
	return ""
}

// approvedRoute returns the unit's active route approval binding.
func approvedRoute(core []model.Record, runID, unit string) (*model.ApprovalBinding, error) {
	route, err := approvedCoreRouteDigestIn(core, runID, unit)
	if err != nil {
		return nil, err
	}
	var a *model.ApprovalBinding
	for _, r := range core {
		if r.UnitID == unit && r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil && r.Approval.TransitionDigest == route {
			a = r.Approval
		}
	}
	if a == nil {
		return nil, errors.New(hoststate.ReasonRouteApprovalMissing)
	}
	return a, nil
}

// requireApprovedRepository refuses a packet that does not review the run's
// bound repository at the unit's approved base: the run must have a
// repository binding (hma host bind-repository, never taken from a packet),
// the packet root must be that root, the unit's route approval must bind that
// repository identity and the packet's base, and the root must contain the
// base. Checked before a packet is written, under the locks at dispatch, and
// at review import and findings import.
func requireApprovedRepository(history []hoststate.Record, core []model.Record, runID, unit string, p *hoststate.ReviewPacketBinding) error {
	bad := errors.New(hoststate.ReasonReviewRepositoryMismatch)
	repo := hoststate.RunRepository(history)
	if repo == nil {
		return errors.New(hoststate.ReasonRepositoryBindingMissing)
	}
	a, err := approvedRoute(core, runID, unit)
	if err != nil {
		return err
	}
	if p == nil || !sameStore(repo.Root, p.ProjectRoot) || a.RepositoryIdentityDigest != repo.RepositoryIdentity || p.BaseRevision != a.BaseRevisionDigest {
		return bad
	}
	if err := repostate.VerifyBase(context.Background(), p.ProjectRoot, a.BaseRevisionDigest); err != nil {
		return bad
	}
	return nil
}
