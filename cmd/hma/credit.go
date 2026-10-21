package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

// creditCheckHook, when set by a test, runs after each criterion check and
// before the worktree is observed again.
var creditCheckHook func()

type hostCreditInput struct {
	hostEnvelope
	// ReviewRecord is the host head anchor of the unit's latest imported
	// independent review; ReportPath holds that review's exact report bytes.
	ReviewRecord string `json:"review_record"`
	ReportPath   string `json:"report_path"`
	// ContractPath holds the run's bound hma.acceptance-contract/v1 bytes.
	ContractPath          string `json:"contract_path"`
	DirtyWorktreeApproved bool   `json:"dirty_worktree_approved,omitempty"`
}

type creditResult struct {
	hostResult
	Results []hoststate.CriterionResult `json:"results"`
}

// runHostCredit records acceptance-criterion credit (Gate A v3 §4.3, Gate D
// D5). A criterion is PASSED only when both hold at the independently
// approved target: HMA itself re-ran the criterion's contract check there,
// without a shell, and it exited 0; and the imported independent review of
// that exact target (APPROVE, isolated, report verdict pass) lists the
// criterion with status pass. A check that exits non-zero records FAILED.
// Anything else (a report status of fail or unknown, a missing entry)
// records PENDING: a re-check replaces earlier credit for the criteria it
// checks. Only the criteria the unit's plan advances are checked.
//
// The command holds the host run lock and then the core run lock, so no
// review, rewind, or terminal outcome can land between the checks, the
// captures, and the appends.
func runHostCredit(args []string) error {
	inputPath, storePath, err := hostCommandFlags("credit", args)
	if err != nil {
		return err
	}
	var in hostCreditInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	refused := errors.New(hoststate.ReasonCreditRefused)
	if in.ReviewRecord == "" || in.ReportPath == "" || in.ContractPath == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	report, err := os.ReadFile(in.ReportPath)
	if err != nil {
		return errors.New(hoststate.ReasonReviewReportMismatch)
	}
	contractRaw, err := os.ReadFile(in.ContractPath)
	if err != nil {
		return refused
	}
	contract, contractDigest, err := model.ParseAcceptanceContract(contractRaw)
	if err != nil {
		return refused
	}
	var out creditResult
	var rec hoststate.Record
	err = hoststate.NewStore(storePath).Transact(in.RunID, func(history []hoststate.Record, commit func(*hoststate.Record) error) error {
		return store.New(storePath).Transact(in.RunID, func(core []model.Record, appendCore func(*model.Record) error) error {
			if len(core) == 0 {
				return errors.New(hoststate.ReasonCoreRunMissing)
			}
			stage := engine.ProjectStageState(core)
			if stage.Terminal {
				return errors.New(hoststate.ReasonCoreRunTerminal)
			}
			if engine.BoundContractDigest(core) != contractDigest {
				return refused
			}
			// The review: the unit's latest imported one, for its current
			// implementation, approving, fully isolated, with these report
			// bytes.
			rr, err := hoststate.RequireLatestImportedReview(history, in.ReviewRecord)
			if err != nil {
				return err
			}
			rv := rr.Review
			sum := sha256.Sum256(report)
			if "sha256:"+hex.EncodeToString(sum[:]) != rv.ReportDigest {
				return errors.New(hoststate.ReasonReviewReportMismatch)
			}
			if rv.Verdict != hoststate.ReviewApprove || rv.Isolation != "isolated" || rv.ReportVerdict != "pass" {
				return refused
			}
			status, err := reportRequirementStatus(report)
			if err != nil {
				return errors.New(hoststate.ReasonReviewReportMismatch)
			}
			unit := rv.UnitID
			if _, err := approvedCoreRouteDigestIn(core, in.RunID, unit); err != nil {
				return err
			}
			// The review must have been dispatched under the current
			// contract and the unit's current lease generation.
			_, issuer, err := reviewBinding(core, history, rv)
			if err != nil {
				return err
			}
			// The target: the reviewed artifact, in the run's approved
			// repository at the approved base.
			p := packetFor(history, rv.ReviewDispatchDigest)
			if err := evidence.RequireSelfContainedRepository(p.ProjectRoot); err != nil {
				return err
			}
			if err := requireApprovedRepository(history, core, in.RunID, unit, p); err != nil {
				return err
			}
			repo := hoststate.RunRepository(history)
			// Everything below is bounded: a blocked git read (a FIFO named as
			// an alternates file) ends instead of holding both run locks.
			ctx, cancelCredit := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancelCredit()
			// The .git fingerprint is taken first and compared after every
			// check, so any change inside .git while credit runs (object
			// store, index, info/exclude, config, refs) refuses the credit.
			fpStart, err := repostate.Fingerprint(ctx, p.ProjectRoot)
			if err != nil {
				return err
			}
			// The object store must hold the history of the live HEAD,
			// byte for byte: a forged object planted before the first
			// reading would otherwise define the "reviewed" target.
			liveHead, err := repostate.Output(ctx, p.ProjectRoot, "rev-parse", "HEAD")
			if err != nil {
				return err
			}
			if err := evidence.VerifyRepositoryObjects(ctx, p.ProjectRoot, repostate.Trim(liveHead)); err != nil {
				return err
			}
			before, err := repostate.Snapshot(ctx, p.ProjectRoot, p.BaseRevision, in.DirtyWorktreeApproved)
			if err != nil {
				return err
			}
			if before.Head != repostate.Trim(liveHead) {
				return errors.New("worktree changed during criterion check")
			}
			if hoststate.ReviewArtifactDigest(before.Head, before.Diff, before.Worktree) != rv.ArtifactDigest {
				return errors.New(hoststate.ReasonReviewArtifactMismatch)
			}
			advanced, err := unitCriteria(storePath, core, unit)
			if err != nil {
				return err
			}
			checks := map[string]model.Check{}
			for _, c := range contract.Criteria {
				checks[c.ID] = c.Check
			}
			criteria := append([]model.Criterion(nil), engine.ProjectResolutionState(core).Criteria...)
			index := map[string]int{}
			for i, c := range criteria {
				index[c.ID] = i
			}
			var results []hoststate.CriterionResult
			for _, id := range advanced {
				check, ok := checks[id]
				i, known := index[id]
				if !ok || !known {
					return refused
				}
				argv := append([]string{check.Executable}, check.Argv...)
				if unsafeLoopArgv(argv) {
					return refused
				}
				exe, err := canonicalExecutable(argv[0])
				if err != nil {
					return refused
				}
				dir, err := checkDir(p.ProjectRoot, check.WorkingDir)
				if err != nil {
					return refused
				}
				// The check runs on a private copy of exactly the reviewed
				// target, not in the live worktree: an ignored file, a cache,
				// or anything else the review did not see cannot reach it.
				captured, err := captureOnTarget(ctx, p.ProjectRoot, p.BaseRevision, before.Head, before.Staged, dir, evidence.Request{
					Executable: exe, Argv: argv[1:], MaxOutputBytes: 1 << 20,
					RepositoryIdentity: repo.RepositoryIdentity, BaseRevision: p.BaseRevision, HeadRevision: before.Head,
					DiffDigest: before.Diff, WorktreeDigest: before.Worktree, CapturingActor: "hma:" + in.Actor,
				})
				if err != nil {
					return err
				}
				if creditCheckHook != nil {
					creditCheckHook()
				}
				after, err := repostate.Snapshot(ctx, p.ProjectRoot, p.BaseRevision, in.DirtyWorktreeApproved)
				if err != nil || after != before {
					return errors.New("worktree changed during criterion check")
				}
				// Equal snapshots do not show which bytes the check read: a
				// writer can swap a file in and restore it. File metadata
				// (status-change time) exposes that.
				if fpAfter, err := repostate.Fingerprint(ctx, p.ProjectRoot); err != nil || fpAfter != fpStart {
					return errors.New("worktree files changed during criterion check")
				}
				ref, err := retainOutput(storePath, captured.Stdout, captured.Stderr)
				if err != nil {
					return err
				}
				captured.Evidence.OutputRef = ref
				last := core[len(core)-1]
				ev := model.Record{Kind: model.KindEvidence, Version: 1, UnitID: unit, Stage: stage.CurrentStage, Evidence: &captured.Evidence,
					Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
						Timestamp: in.Timestamp, Actor: "hma:" + in.Actor}}
				if err := appendCore(&ev); err != nil {
					return err
				}
				core = append(core, ev)
				d := criterionDisposition(captured.Evidence.ExitCode, status[id])
				criteria[i].Disposition = d
				results = append(results, hoststate.CriterionResult{CriterionID: id, EvidenceRecord: ev.Metadata.HeadAnchor,
					ExitCode: captured.Evidence.ExitCode, ReportStatus: status[id], Disposition: d})
			}
			if len(results) == 0 {
				return refused
			}
			last := core[len(core)-1]
			cr := model.Record{Kind: model.KindCriterion, Version: 1, UnitID: unit, Stage: stage.CurrentStage, Criteria: criteria,
				Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
					Timestamp: in.Timestamp, Actor: "hma:" + in.Actor}}
			if err := appendCore(&cr); err != nil {
				return err
			}
			rec = hoststate.Record{Version: 1, Kind: hoststate.KindCriterionCredit, RunID: in.RunID, CoreHeadAnchor: cr.Metadata.HeadAnchor,
				Timestamp: in.Timestamp, Actor: in.Actor, CriterionCredit: &hoststate.CriterionCredit{UnitID: unit, ReviewRecord: rr.HeadAnchor,
					ReportDigest: rv.ReportDigest, ArtifactDigest: rv.ArtifactDigest, ContractDigest: contractDigest, Results: results,
					CriteriaRecord: cr.Metadata.HeadAnchor, IssuerHeadAnchor: issuer}}
			if err := commit(&rec); err != nil {
				return err
			}
			out.Results = results
			out.NextAction = creditNextAction(results)
			return nil
		})
	})
	if err != nil {
		return err
	}
	next := out.NextAction
	out.hostResult = hostResultFor(storePath, rec)
	out.NextAction = next
	return json.NewEncoder(os.Stdout).Encode(out)
}

// creditNextAction is the single next action after a credit: remediate a
// failed check; escalate a criterion the review did not verify; otherwise
// continue through the normal human-approved transition.
func creditNextAction(results []hoststate.CriterionResult) string {
	next := "CONTINUE_HMA_TRANSITION"
	for _, r := range results {
		switch r.Disposition {
		case model.CriterionFailed:
			return "REMEDIATE_OR_ESCALATE"
		case model.CriterionPending:
			next = "ESCALATE_TO_HUMAN"
		}
	}
	return next
}

// criterionDisposition is the credit rule: PASSED only when HMA's own check
// exited 0 and the independent report lists the criterion as pass; FAILED
// when the check failed; PENDING otherwise.
func criterionDisposition(exit int, reportStatus string) model.CriterionDisposition {
	switch {
	case exit != 0:
		return model.CriterionFailed
	case reportStatus == "pass":
		return model.CriterionPassed
	}
	return model.CriterionPending
}

// reportRequirementStatus returns each requirement id's status in an
// independent-audit report. A repeated id is refused.
func reportRequirementStatus(report []byte) (map[string]string, error) {
	var rep struct {
		Requirements []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"requirements"`
	}
	if err := hoststate.UnmarshalExact(report, &rep); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rep.Requirements {
		if _, dup := out[r.ID]; dup {
			return nil, errors.New("duplicate requirement id")
		}
		out[r.ID] = r.Status
	}
	return out, nil
}

// unitCriteria returns the criteria the unit's current lease generation
// plan advances, from that generation's stored plan.
func unitCriteria(storePath string, core []model.Record, unit string) ([]string, error) {
	digest := ""
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			digest = r.Approval.AcceptedPlanDigest
		}
	}
	plans, err := loadPlans(storePath, core)
	if err != nil {
		return nil, err
	}
	plan, ok := plans[digest]
	if !ok {
		return nil, errors.New(hoststate.ReasonCreditRefused)
	}
	for _, u := range plan.Units {
		if u.ID == unit {
			return u.CriteriaAdvanced, nil
		}
	}
	return nil, errors.New(hoststate.ReasonCreditRefused)
}

// checkDir resolves a contract check's working directory inside root,
// through symlinks: the resolved directory must be inside the resolved root.
// captureOnTarget runs the request in a private, single-use copy of the
// repository's reviewed target (see repostate.Materialize) at the same
// relative working directory as dir below root, and records the live
// directory in the evidence. Git commands read the approved repository's
// own .git through the validated private git directory.
func captureOnTarget(ctx context.Context, root, base, head string, staged bool, dir string, req evidence.Request) (evidence.Result, error) {
	rroot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return evidence.Result{}, err
	}
	rel, err := filepath.Rel(rroot, dir)
	if err != nil {
		return evidence.Result{}, err
	}
	// The live repository path in the command would let it read or write
	// outside the copy.
	for _, live := range []string{root, rroot} {
		for _, a := range append([]string{req.Executable}, req.Argv...) {
			if strings.Contains(a, live) {
				return evidence.Result{}, errors.New("check names the live repository path")
			}
		}
	}
	copyRoot, err := os.MkdirTemp("", "hma-target-")
	if err != nil {
		return evidence.Result{}, err
	}
	defer os.RemoveAll(copyRoot)
	dirty, err := repostate.Materialize(ctx, rroot, head, copyRoot)
	if err != nil {
		return evidence.Result{}, err
	}
	committed, err := repostate.Output(ctx, rroot, "diff", "--name-only", "-z", "--no-renames", "--no-ext-diff", base, head, "--")
	if err != nil {
		return evidence.Result{}, err
	}
	seen := map[string]bool{}
	for _, c := range append(dirty, strings.Split(committed, "\x00")...) {
		if c != "" && !seen[c] {
			seen[c] = true
			req.ChangedFiles = append(req.ChangedFiles, c)
		}
	}
	sort.Strings(req.ChangedFiles)
	wd, err := checkDir(copyRoot, rel)
	if err != nil {
		return evidence.Result{}, err
	}
	req.WorkingDir, req.RepositoryRoot, req.GitRepository = wd, copyRoot, rroot
	req.Sandbox, req.PinnedHead, req.StagedTarget = true, head, staged
	return evidence.Capture(ctx, req)
}

func checkDir(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", errors.New("absolute working dir")
	}
	rroot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	dir, err := filepath.EvalSymlinks(filepath.Join(rroot, rel))
	if err != nil {
		return "", err
	}
	r, err := filepath.Rel(rroot, dir)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("working dir outside repository")
	}
	return dir, nil
}

// reviewBinding returns the contract digest and the unit's lease issuer
// anchor in force when the review was dispatched (at its dispatch's core
// head), and requires both to still be the current ones: a review of an
// earlier contract or generation credits nothing.
func reviewBinding(core []model.Record, history []hoststate.Record, rv *hoststate.IndependentReview) (string, string, error) {
	refused := errors.New(hoststate.ReasonCreditRefused)
	anchor := ""
	for _, r := range history {
		if r.HeadAnchor == rv.ReviewDispatchDigest {
			anchor = r.CoreHeadAnchor
		}
	}
	at := -1
	for i, r := range core {
		if r.Metadata.HeadAnchor == anchor {
			at = i
		}
	}
	if anchor == "" || at < 0 {
		return "", "", refused
	}
	contract, issuer := currentBinding(core[:at+1], rv.UnitID)
	nowContract, nowIssuer := currentBinding(core, rv.UnitID)
	if contract == "" || issuer == "" || contract != nowContract || issuer != nowIssuer {
		return "", "", refused
	}
	return contract, issuer, nil
}

// currentBinding returns the bound contract digest and the unit's latest
// lease issuer anchor in core.
func currentBinding(core []model.Record, unit string) (string, string) {
	issuer := ""
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			issuer = r.Metadata.HeadAnchor
		}
	}
	return engine.BoundContractDigest(core), issuer
}

// requireIndependentCredit refuses a verified outcome unless every PASSED
// criterion in the current core state is backed by HMA's own credit: the
// latest core criterion record is one a criterion_credit host record
// appended, and for each PASSED criterion its latest credit result is
// PASSED, under the current bound contract and the unit's current lease
// generation, at the closure artifact (artifact, the live snapshot of the
// transition's repository at its base), relying on a review that is still
// the unit's latest for its current implementation. A criterion record
// written any other way (a self-reported scorecard) credits nothing, and
// carrying it forward does not launder it.
func requireIndependentCredit(storePath, runID, artifact string, history []hoststate.Record) error {
	core, err := store.New(storePath).Load(runID)
	if err != nil {
		return err
	}
	refused := errors.New(hoststate.ReasonCreditRefused)
	latest := ""
	for _, r := range core {
		if r.Kind == model.KindCriterion && len(r.Criteria) > 0 {
			latest = r.Metadata.HeadAnchor
		}
	}
	fromCredit := false
	type credited struct {
		result hoststate.CriterionResult
		credit *hoststate.CriterionCredit
	}
	last := map[string]credited{}
	for _, r := range history {
		c := r.CriterionCredit
		if r.Kind != hoststate.KindCriterionCredit || c == nil {
			continue
		}
		if c.CriteriaRecord == latest {
			fromCredit = true
		}
		for _, res := range c.Results {
			last[res.CriterionID] = credited{res, c}
		}
	}
	for _, c := range engine.ProjectResolutionState(core).Criteria {
		if c.Disposition != model.CriterionPassed {
			continue
		}
		cr, ok := last[c.ID]
		if !fromCredit || !ok || cr.result.Disposition != model.CriterionPassed {
			return refused
		}
		contract, issuer := currentBinding(core, cr.credit.UnitID)
		if cr.credit.ContractDigest != contract || cr.credit.IssuerHeadAnchor != issuer || cr.credit.ArtifactDigest != artifact {
			return refused
		}
		if _, err := hoststate.RequireLatestImportedReview(history, cr.credit.ReviewRecord); err != nil {
			return refused
		}
	}
	return nil
}
