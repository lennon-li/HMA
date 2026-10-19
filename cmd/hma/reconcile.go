package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

type hostReconcileInput struct {
	hostEnvelope
	// PolicyPath names the current effective route policy. It is required
	// when the run has reservations: a lease's launches are authorized only
	// under its plan's policy.
	PolicyPath string `json:"policy_path,omitempty"`
}

type reconcileResult struct {
	hostResult
	Reconciliation hoststate.Reconciliation `json:"reconciliation"`
}

// runHostReconcile reconciles every ledger run in the run's declared
// namespaces (loop admissions and HMA-generated review dispatches) with the
// host records, using each lease generation's own stored plan (Gate A v5
// §1-§2, v6 §4, v7 §1). Under the host and core locks it appends one
// launch_reconciliation host record binding the ledger inventory and, when
// any run or journal entry is unauthorized, one core finding record carrying
// the current findings plus an unowned BLOCK per entry (UNAUTHORIZED_DISPATCH).
func runHostReconcile(args []string) error {
	inputPath, storePath, err := hostCommandFlags("reconcile", args)
	if err != nil {
		return err
	}
	var in hostReconcileInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	policyDigest := ""
	if in.PolicyPath != "" {
		var policy model.RoutePolicy
		if err := decodeStrictFile(in.PolicyPath, &policy); err != nil {
			return err
		}
		proj, err := model.NormalizeRoutePolicy(policy)
		if err != nil {
			return err
		}
		if policyDigest, err = engine.RoutePolicyDigest(proj); err != nil {
			return err
		}
	}
	var out reconcileResult
	var rec hoststate.Record
	err = hoststate.NewStore(storePath).Transact(in.RunID, func(history []hoststate.Record, commit func(*hoststate.Record) error) error {
		return store.New(storePath).Transact(in.RunID, func(core []model.Record, appendCore func(*model.Record) error) error {
			if len(core) == 0 {
				return errors.New(hoststate.ReasonCoreRunMissing)
			}
			// The clock is read under both locks.
			now := time.Now().UTC()
			res, plans, err := reconcileNow(storePath, core, history, policyDigest, now)
			if err != nil {
				return err
			}
			if len(res.Unauthorized) > 0 {
				prior := engine.ProjectResolutionState(core).Findings
				findings := append([]model.Finding(nil), prior...)
				have := map[string]bool{}
				for _, f := range prior {
					have[f.ID] = true
				}
				added := false
				for _, f := range hoststate.UnauthorizedFindings(res) {
					if !have[f.ID] {
						findings = append(findings, f)
						have[f.ID] = true
						added = true
					}
				}
				if added {
					last := core[len(core)-1]
					fr := model.Record{Kind: model.KindFinding, Version: 1, Stage: engine.ProjectStageState(core).CurrentStage, Findings: findings,
						Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
							Timestamp: in.Timestamp, Actor: "hma:" + in.Actor}}
					if err := appendCore(&fr); err != nil {
						return err
					}
					core = append(core, fr)
				}
			}
			// A presented policy other than a stored plan's is an observed
			// policy change: the record carries it as a revocation, dated now
			// at the current core head, like the hook's policy_revocation, so
			// restoring the policy revives no launch authority. It does not
			// date earlier launches: they were reconciled without it.
			for _, d := range planPolicies(plans) {
				if policyDigest != "" && d != policyDigest && hoststate.PolicyRevocationNeeded(core, history, d) {
					res.RevokedPolicies = append(res.RevokedPolicies, d)
				}
			}
			if len(res.RevokedPolicies) > 0 {
				res.ObservedAt = now.Format(time.RFC3339Nano)
			}
			rec = hoststate.Record{Version: 1, Kind: hoststate.KindLaunchReconciliation, RunID: in.RunID,
				CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor, Timestamp: in.Timestamp, Actor: in.Actor, Reconciliation: &res}
			if err := commit(&rec); err != nil {
				return err
			}
			out.Reconciliation = res
			// The next action reflects the run, not only this scan: an
			// earlier unauthorized-dispatch BLOCK still needs a human.
			out.NextAction = "CONTINUE_HMA_TRANSITION"
			if len(res.Unauthorized) > 0 || hoststate.HasUnauthorizedDispatch(engine.ProjectResolutionState(core).Findings) {
				out.NextAction = "ESCALATE_TO_HUMAN"
			}
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

// reconcileNow scans every declared namespace and reconciles it at now; it
// also returns the stored plans it used.
func reconcileNow(storePath string, core []model.Record, history []hoststate.Record, policyDigest string, now time.Time) (hoststate.Reconciliation, map[string]model.Plan, error) {
	plans, err := loadPlans(storePath, core)
	if err != nil {
		return hoststate.Reconciliation{}, nil, err
	}
	var runs []hoststate.LedgerRun
	for _, ns := range hoststate.Namespaces(history) {
		runs = append(runs, scanLedger(ns)...)
	}
	return hoststate.Reconcile(hoststate.ReconcileInput{Core: core, History: history, Plans: plans, Runs: runs,
		PolicyDigest: policyDigest, Now: now}), plans, nil
}

// planPolicies returns the distinct policy digests of the plans, sorted.
func planPolicies(plans map[string]model.Plan) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range plans {
		if p.PolicyDigest != "" && !seen[p.PolicyDigest] {
			seen[p.PolicyDigest] = true
			out = append(out, p.PolicyDigest)
		}
	}
	sort.Strings(out)
	return out
}

// scanLedger lists every run directory in an agent-dispatch ledger
// namespace with what its receipt says, including its launch identity. A missing namespace has no runs; an
// unreadable one is reported as one unreadable run so it cannot hide launches.
func scanLedger(ns string) []hoststate.LedgerRun {
	entries, err := os.ReadDir(filepath.Join(ns, "runs"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []hoststate.LedgerRun{{Namespace: ns, Dir: "<unreadable>", Err: err.Error()}}
	}
	var out []hoststate.LedgerRun
	for _, e := range entries {
		run := hoststate.LedgerRun{Namespace: ns, Dir: e.Name()}
		b, err := os.ReadFile(filepath.Join(ns, "runs", e.Name(), "receipt.json"))
		var rc hoststate.Receipt
		if err != nil {
			run.Err = "receipt missing"
		} else if err := hoststate.UnmarshalExact(b, &rc); err != nil {
			sum := sha256.Sum256(b)
			run.ReceiptSHA256, run.Err = hex.EncodeToString(sum[:]), "receipt unreadable"
		} else {
			sum := sha256.Sum256(b)
			run.RunID, run.Mode, run.StartedAt, run.ReceiptSHA256 = rc.RunID, rc.Mode, rc.StartedAt, hex.EncodeToString(sum[:])
			run.Launch = rc.Launch()
		}
		out = append(out, run)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out
}

// requireReconciledLaunches refuses the host-gated transitions while the
// core findings carry an unauthorized-dispatch BLOCK, and, for a run that
// declares any ledger namespace, unless the latest reconciliation is clean,
// no launch authority or journal was recorded after it, it is bound to the
// current core head, and the ledger still holds exactly what it reconciled.
// The caller holds the host run lock, so no hook or import can record
// anything meanwhile.
func requireReconciledLaunches(storePath, runID, unit string, history []hoststate.Record) error {
	core, err := store.New(storePath).Load(runID)
	if err != nil {
		return err
	}
	if hoststate.HasUnauthorizedDispatch(engine.ProjectResolutionState(core).Findings) {
		return errors.New(hoststate.ReasonUnauthorizedDispatch)
	}
	namespaces := hoststate.Namespaces(history)
	if len(namespaces) == 0 {
		return nil
	}
	required := errors.New(hoststate.ReasonReconciliationRequired)
	var last *hoststate.Record
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Kind == hoststate.KindLaunchReconciliation && history[i].Reconciliation != nil {
			last = &history[i]
			break
		}
		if hoststate.ReconciliationRelevant(history[i]) {
			return required
		}
	}
	if last == nil || len(last.Reconciliation.Unauthorized) > 0 || len(core) == 0 || last.CoreHeadAnchor != core[len(core)-1].Metadata.HeadAnchor {
		return required
	}
	var runs []hoststate.LedgerRun
	for _, ns := range namespaces {
		runs = append(runs, scanLedger(ns)...)
	}
	if hoststate.InventoryDigest(runs) != last.Reconciliation.InventoryDigest {
		return required
	}
	return nil
}
