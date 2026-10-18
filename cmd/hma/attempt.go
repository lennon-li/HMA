package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// attemptNow is the hook's clock; tests replace it.
var attemptNow = func() time.Time { return time.Now().UTC() }

// attemptRoute is the verify-loop route in a hook request or envelope.
type attemptRoute struct {
	Worker string  `json:"worker"`
	Model  *string `json:"model"`
	Effort *string `json:"effort"`
	Cost   struct {
		SpendCapUSD   float64 `json:"spend_cap_usd"`
		PerAttemptUSD float64 `json:"per_attempt_usd"`
	} `json:"cost"`
}

// attemptRequest is the verify-loop authority hook request
// (verify-loop authority_request).
type attemptRequest struct {
	ControllerRunID      string       `json:"controller_run_id"`
	AttemptIndex         int          `json:"attempt_index"`
	DispatchRunID        string       `json:"dispatch_run_id"`
	EnvelopeSHA256       string       `json:"envelope_sha256"`
	Route                attemptRoute `json:"route"`
	LastFailureSignature *string      `json:"last_failure_signature"`
	LiveRoot             string       `json:"live_root"`
	CloneBaselineDigest  string       `json:"clone_baseline_digest"`
	LedgerNamespace      string       `json:"ledger_namespace"`
	HMA                  struct {
		Store  string `json:"store"`
		RunID  string `json:"run_id"`
		UnitID string `json:"unit_id"`
	} `json:"hma"`
}

// attemptEnvelope is the part of the controller's envelope admission reads.
type attemptEnvelope struct {
	MaxAttempts int             `json:"max_attempts"`
	MaxMinutes  float64         `json:"max_minutes"`
	Route       attemptRoute    `json:"route"`
	EscalateTo  json.RawMessage `json:"escalate_to"`
}

// pendingAttempt reports whether the controller state's latest attempt is
// the one the request names, waiting for authority: same index, dispatch
// id, route, and last failure signature, in phase "dispatching". A new
// reservation needs it; a replay of an existing one does not.
type pendingAttempt bool

type attemptResult struct {
	ReservationID  string `json:"reservation_id"`
	LaunchEligible bool   `json:"launch_eligible"`
	Replay         bool   `json:"replay"`
	DispatchRunID  string `json:"dispatch_run_id"`
	ValidUntil     string `json:"valid_until"`
	Sequence       int64  `json:"sequence"`
	NextAction     string `json:"next_action"`
}

// attemptDocs are the human-bound documents the hook argv names.
type attemptDocs struct {
	plan, contract []byte
	policyDigest   string
}

// runHostAttempt is the verify-loop authority hook (Gate A v3 §3.2, v4 §3,
// v5 §2, v6 §3). It reads one hook request on stdin (or --request) and, under
// the host and core run locks:
//
//  1. replays an existing reservation with the same key and request digest
//     unchanged (a different digest is HOST_RESERVATION_CONFLICT), with
//     launch_eligible true only while its lease generation is still active
//     by time, stage, policy, and contract (capacity is not rechecked) and no
//     ledger run with its dispatch id exists;
//  2. admits the controller run on its first call (consuming nothing):
//     active lease, escalate_to "stop", envelope route equal to the lease
//     route, envelope attempts, minutes, and spend within what remains;
//  3. charges one reservation: active lease with capacity, the admission's
//     envelope and association, and no failure signature presented twice.
//
// A reservation stays charged whether or not the worker launches. Any
// refusal exits non-zero, so the controller does not launch.
func runHostAttempt(args []string) error {
	fs := flag.NewFlagSet("attempt", flag.ContinueOnError)
	storePath := fs.String("store", "", "HMA store directory")
	planPath := fs.String("plan", "", "accepted hma.plan/v1 document")
	contractPath := fs.String("contract", "", "accepted hma.acceptance-contract/v1 document")
	policyPath := fs.String("policy", "", "effective route policy document")
	requestPath := fs.String("request", "", "hook request file (default stdin)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePath == "" || *planPath == "" || *contractPath == "" || *policyPath == "" || fs.NArg() != 0 {
		return errors.New("usage: hma host attempt --store <dir> --plan <file> --contract <file> --policy <file> [--request <file>]")
	}
	var raw []byte
	var err error
	if *requestPath != "" {
		raw, err = os.ReadFile(*requestPath)
	} else {
		raw, err = io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	}
	if err != nil {
		return err
	}
	var docs attemptDocs
	if docs.plan, err = os.ReadFile(*planPath); err != nil {
		return err
	}
	if docs.contract, err = os.ReadFile(*contractPath); err != nil {
		return err
	}
	var policy model.RoutePolicy
	if err := decodeStrictFile(*policyPath, &policy); err != nil {
		return err
	}
	proj, err := model.NormalizeRoutePolicy(policy)
	if err != nil {
		return err
	}
	if docs.policyDigest, err = engine.RoutePolicyDigest(proj); err != nil {
		return err
	}
	req, reqDigest, env, pending, err := parseAttemptRequest(raw, *storePath)
	if err != nil {
		return err
	}
	// Admission and consumption are separate appends; a run's first call
	// does both, each under its own lock.
	for i := 0; i < 2; i++ {
		res, admitted, err := attemptOnce(*storePath, req, reqDigest, env, pending, docs)
		if err != nil {
			return err
		}
		if !admitted {
			return json.NewEncoder(os.Stdout).Encode(res)
		}
	}
	return errors.New(hoststate.ReasonAttemptInvalid)
}

// parseAttemptRequest validates the request and the controller state it
// names: the state's envelope must hash to the request's envelope digest
// (as verify-loop computes it), and the state must be this controller run,
// bound to this HMA store, run, and unit, at the request's live root.
func parseAttemptRequest(raw []byte, storePath string) (attemptRequest, string, attemptEnvelope, pendingAttempt, error) {
	bad := errors.New(hoststate.ReasonAttemptInvalid)
	var req attemptRequest
	var env attemptEnvelope
	if err := hoststate.UnmarshalExact(raw, &req); err != nil {
		return req, "", env, false, bad
	}
	digest, err := hoststate.PythonCanonicalSHA256(raw)
	if err != nil {
		return req, "", env, false, bad
	}
	if req.ControllerRunID == "" || req.AttemptIndex < 1 || req.DispatchRunID == "" || req.EnvelopeSHA256 == "" ||
		req.LiveRoot == "" || req.LedgerNamespace == "" || req.HMA.RunID == "" || req.HMA.UnitID == "" ||
		filepath.Base(req.DispatchRunID) != req.DispatchRunID || !sameStore(req.HMA.Store, storePath) {
		return req, "", env, false, bad
	}
	// verify-loop's ledger namespace is <state dir>/dispatch-ledger; any
	// other request namespace is refused, so admission and replay always use
	// the controller's own ledger.
	stateDir := filepath.Dir(filepath.Clean(req.LedgerNamespace))
	if filepath.Clean(req.LedgerNamespace) != filepath.Join(stateDir, "dispatch-ledger") || !filepath.IsAbs(req.LedgerNamespace) {
		return req, "", env, false, bad
	}
	req.LedgerNamespace = filepath.Join(stateDir, "dispatch-ledger")
	stateRaw, err := os.ReadFile(filepath.Join(stateDir, "state.json"))
	if err != nil {
		return req, "", env, false, bad
	}
	var st loopState
	var top map[string]json.RawMessage
	if hoststate.UnmarshalExact(stateRaw, &st) != nil || json.Unmarshal(stateRaw, &top) != nil {
		return req, "", env, false, bad
	}
	if sum, err := hoststate.PythonCanonicalSHA256(top["envelope"]); err != nil || sum != st.EnvelopeSHA256 || sum != req.EnvelopeSHA256 {
		return req, "", env, false, bad
	}
	h := st.Envelope.HMA
	if st.RunID != req.ControllerRunID || h == nil || h.RunID != req.HMA.RunID || h.UnitID != req.HMA.UnitID || !sameStore(h.Store, storePath) ||
		st.Repo == nil || !sameStore(st.Repo.Root, req.LiveRoot) {
		return req, "", env, false, bad
	}
	if hoststate.UnmarshalExact(top["envelope"], &env) != nil {
		return req, "", env, false, bad
	}
	// The clone baseline digest must be the controller's, computed as
	// verify-loop's authority_request does.
	var repo map[string]json.RawMessage
	if json.Unmarshal(top["repo"], &repo) != nil {
		return req, "", env, false, bad
	}
	inventory := repo["workspace_baseline_inventory"]
	if len(inventory) == 0 {
		inventory = json.RawMessage(`{}`)
	}
	if want, err := hoststate.PythonCanonicalSHA256(inventory); err != nil || want != req.CloneBaselineDigest {
		return req, "", env, false, bad
	}
	return req, digest, env, pendingFor(top["attempts"], raw), nil
}

// pendingFor compares the controller state's latest attempt with the
// request, field by field as verify-loop's authority_request builds it.
func pendingFor(attemptsRaw json.RawMessage, reqRaw []byte) pendingAttempt {
	var attempts []map[string]json.RawMessage
	var req map[string]json.RawMessage
	if json.Unmarshal(attemptsRaw, &attempts) != nil || len(attempts) == 0 || json.Unmarshal(reqRaw, &req) != nil {
		return false
	}
	last := attempts[len(attempts)-1]
	same := func(a, b json.RawMessage) bool {
		da, ea := hoststate.PythonCanonicalSHA256(a)
		db, eb := hoststate.PythonCanonicalSHA256(b)
		return ea == nil && eb == nil && da == db
	}
	var phase string
	if json.Unmarshal(last["phase"], &phase) != nil || phase != "dispatching" {
		return false
	}
	if !same(last["n"], req["attempt_index"]) || !same(last["dispatch_run_id"], req["dispatch_run_id"]) || !same(last["route"], req["route"]) {
		return false
	}
	sig := json.RawMessage("null")
	var check map[string]json.RawMessage
	if len(last["last_check"]) > 0 && json.Unmarshal(last["last_check"], &check) == nil {
		if v, ok := check["signature"]; ok {
			sig = v
		}
	}
	want := req["last_failure_signature"]
	if len(want) == 0 {
		want = json.RawMessage("null")
	}
	return pendingAttempt(same(sig, want))
}

// attemptOnce runs one locked step. It returns admitted=true when it only
// appended the controller run's admission, so the caller runs again to
// charge the reservation.
func attemptOnce(storePath string, req attemptRequest, reqDigest string, env attemptEnvelope, pending pendingAttempt, docs attemptDocs) (attemptResult, bool, error) {
	var res attemptResult
	admitted := false
	unit, runID := req.HMA.UnitID, req.HMA.RunID
	err := hoststate.NewStore(storePath).Transact(runID, func(history []hoststate.Record, commit func(*hoststate.Record) error) error {
		return store.New(storePath).WithRunLock(runID, func(core []model.Record) error {
			// The clock is read only once both locks are held, so lock
			// waiting cannot carry a stale time past an expiry.
			now := attemptNow()
			if len(core) == 0 {
				return errors.New(hoststate.ReasonCoreRunMissing)
			}
			repo := hoststate.RunRepository(history)
			if repo == nil {
				return errors.New(hoststate.ReasonRepositoryBindingMissing)
			}
			if !sameStore(req.LiveRoot, repo.Root) {
				return errors.New(hoststate.ReasonRepositoryMismatch)
			}
			// 1. Replay first.
			for _, r := range history {
				v := r.Reservation
				if r.Kind != hoststate.KindReservation || v == nil {
					continue
				}
				sameKey := v.ControllerRunID == req.ControllerRunID && v.AttemptIndex == req.AttemptIndex && v.DispatchRunID == req.DispatchRunID
				if sameKey && v.RequestSHA256 != reqDigest {
					return errors.New(hoststate.ReasonReservationConflict)
				}
				if sameKey {
					res = replayResult(r, core, history, req, docs, now)
					return nil
				}
				if v.DispatchRunID == req.DispatchRunID || v.ControllerRunID == req.ControllerRunID && v.AttemptIndex == req.AttemptIndex {
					return errors.New(hoststate.ReasonReservationConflict)
				}
			}
			for _, r := range history {
				if r.Dispatch != nil && r.Dispatch.Packet != nil && r.Dispatch.Packet.DispatchRunID == req.DispatchRunID {
					return errors.New(hoststate.ReasonReservationConflict)
				}
			}
			// A new reservation answers only the controller's pending
			// attempt, and never a dispatch id the ledger already ran.
			if !pending {
				return errors.New(hoststate.ReasonAttemptInvalid)
			}
			if dispatchUsed(req.LedgerNamespace, req.DispatchRunID) {
				return errors.New(hoststate.ReasonReservationConflict)
			}
			grant, routeID, preflightUntil, err := attemptRouteGrant(history, unit, req.Route, env.Route, now)
			if err != nil {
				return err
			}
			counters := unitCounters(history, unit)
			lease, reason := engine.DeriveLease(engine.LeaseRequest{Records: core, UnitID: unit, Plan: docs.plan, Contract: docs.contract,
				PolicyDigest: docs.policyDigest, Counters: counters, Now: now, Route: grant})
			if reason != engine.ReasonNone {
				return fmt.Errorf("%s: %s", hoststate.ReasonLeaseInactive, reason)
			}
			gen := hoststate.LeaseGeneration{IssuerSequence: lease.IssuerSequence, IssuerHeadAnchor: lease.IssuerHeadAnchor, LeaseDigest: lease.LeaseDigest}
			until, _ := time.Parse(time.RFC3339Nano, lease.ValidUntil)
			var adm *hoststate.Record
			for i := range history {
				if a := history[i].Admission; history[i].Kind == hoststate.KindLoopAdmission && a != nil && a.ControllerRunID == req.ControllerRunID {
					adm = &history[i]
				}
			}
			// 2. Admission, once per controller run.
			if adm == nil {
				contract, _, err := model.ParseAcceptanceContract(docs.contract)
				if err != nil {
					return errors.New(hoststate.ReasonAdmissionRefused)
				}
				remainingCost := lease.Budgets.CostCeiling - counters.CostSpent
				if string(env.EscalateTo) != `"stop"` || env.MaxAttempts < 1 || env.MaxAttempts > lease.Budgets.MaxAttempts-counters.Attempts ||
					env.MaxMinutes <= 0 || env.MaxMinutes > until.Sub(now).Minutes() ||
					contract.Budgets.CostUnit != "usd" || env.Route.Cost.SpendCapUSD < 0 || env.Route.Cost.SpendCapUSD > float64(remainingCost) {
					return errors.New(hoststate.ReasonAdmissionRefused)
				}
				rec := hoststate.Record{Version: 1, Kind: hoststate.KindLoopAdmission, RunID: runID, CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor,
					Timestamp: now.Format(time.RFC3339Nano), Actor: "hma-hook",
					Admission: &hoststate.LoopAdmission{UnitID: unit, ControllerRunID: req.ControllerRunID, EnvelopeSHA256: req.EnvelopeSHA256,
						LiveRoot: repo.Root, CloneBaselineDigest: req.CloneBaselineDigest, LedgerNamespace: req.LedgerNamespace, Generation: gen}}
				if err := commit(&rec); err != nil {
					return err
				}
				admitted = true
				return nil
			}
			a := adm.Admission
			if a.UnitID != unit || a.EnvelopeSHA256 != req.EnvelopeSHA256 || !sameStore(a.LiveRoot, req.LiveRoot) ||
				a.CloneBaselineDigest != req.CloneBaselineDigest || a.LedgerNamespace != req.LedgerNamespace || a.Generation != gen {
				return errors.New(hoststate.ReasonAdmissionRefused)
			}
			// 3. Consumption.
			sig := ""
			if req.LastFailureSignature != nil {
				sig = *req.LastFailureSignature
			}
			if sig != "" {
				for _, r := range history {
					// Unit-wide (architecture §17): a new controller run
					// does not reset the material-failure rule.
					if v := r.Reservation; r.Kind == hoststate.KindReservation && v != nil && v.UnitID == unit && v.FailureSignature == sig {
						return errors.New(hoststate.ReasonRepeatedFailure)
					}
				}
			}
			cost := int64(math.Ceil(env.Route.Cost.PerAttemptUSD))
			if cost < 0 || counters.CostSpent+cost > lease.Budgets.CostCeiling {
				return fmt.Errorf("%s: %s", hoststate.ReasonLeaseInactive, engine.ReasonLeaseCeiling)
			}
			rec := hoststate.Record{Version: 1, Kind: hoststate.KindReservation, RunID: runID, CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor,
				Timestamp: now.Format(time.RFC3339Nano), Actor: "hma-hook",
				Reservation: &hoststate.Reservation{UnitID: unit, ControllerRunID: req.ControllerRunID, AttemptIndex: req.AttemptIndex,
					DispatchRunID: req.DispatchRunID, RequestSHA256: reqDigest, AdmissionRecord: adm.HeadAnchor, RouteIdentity: routeID,
					FailureSignature: sig, Cost: cost, ValidFrom: now.Format(time.RFC3339Nano), ValidUntil: lease.ValidUntil, Generation: gen}}
			if err := commit(&rec); err != nil {
				return err
			}
			res = attemptResult{ReservationID: rec.HeadAnchor, LaunchEligible: true, DispatchRunID: req.DispatchRunID,
				ValidUntil: lease.ValidUntil, Sequence: rec.Sequence, NextAction: "LAUNCH_WORKER"}
			// If the lease or route preflight ended, or the ledger ran the
			// dispatch id, while the record was being committed, the charge stays but no launch is granted.
			if after := attemptNow(); !after.Before(until) || !after.Before(preflightUntil) || dispatchUsed(req.LedgerNamespace, req.DispatchRunID) {
				res.LaunchEligible, res.NextAction = false, "DO_NOT_LAUNCH"
			}
			return nil
		})
	})
	return res, admitted, err
}

// replayResult returns an existing reservation unchanged. launch_eligible is
// true only while its generation's lease is still active by time, stage,
// policy, and contract (capacity is not rechecked: the slot is already
// charged) and no ledger run with its dispatch id exists.
func replayResult(r hoststate.Record, core []model.Record, history []hoststate.Record, req attemptRequest, docs attemptDocs, now time.Time) attemptResult {
	v := r.Reservation
	res := attemptResult{ReservationID: r.HeadAnchor, Replay: true, DispatchRunID: v.DispatchRunID, ValidUntil: v.ValidUntil,
		Sequence: r.Sequence, NextAction: "DO_NOT_LAUNCH"}
	grant, _, _, err := attemptRouteGrant(history, v.UnitID, req.Route, req.Route, now)
	if err != nil {
		return res
	}
	lease, reason := engine.DeriveLease(engine.LeaseRequest{Records: core, UnitID: v.UnitID, Plan: docs.plan, Contract: docs.contract,
		PolicyDigest: docs.policyDigest, Now: now, Route: grant})
	until, perr := time.Parse(time.RFC3339Nano, v.ValidUntil)
	if reason != engine.ReasonNone || perr != nil || !now.Before(until) || lease.IssuerHeadAnchor != v.Generation.IssuerHeadAnchor {
		return res
	}
	if dispatchUsed(req.LedgerNamespace, v.DispatchRunID) {
		return res
	}
	res.LaunchEligible = true
	res.NextAction = "LAUNCH_WORKER"
	return res
}

// dispatchUsed reports whether the ledger has (or may have) a run with this
// dispatch id; anything but "does not exist" counts as used.
func dispatchUsed(namespace, id string) bool {
	_, err := os.Lstat(filepath.Join(namespace, "runs", id))
	return !errors.Is(err, os.ErrNotExist)
}

// attemptRouteGrant finds the unit's latest available implementation route
// preflight (unexpired at now) and requires the request and envelope routes
// to name its worker, model, and effort. The preflight route, not the
// controller's, is what the lease checks.
func attemptRouteGrant(history []hoststate.Record, unit string, reqRoute, envRoute attemptRoute, now time.Time) (*model.RouteGrant, string, time.Time, error) {
	var pf *hoststate.RoutePreflight
	for i := range history {
		if p := history[i].Preflight; history[i].Kind == hoststate.KindPreflight && p != nil && p.UnitID == unit && p.Role == hoststate.RoleImplementation {
			pf = p
		}
	}
	bad := errors.New(hoststate.ReasonAttemptRouteMismatch)
	if pf == nil || pf.Status != hoststate.PreflightAvailable {
		return nil, "", time.Time{}, bad
	}
	exp, err := time.Parse(time.RFC3339, pf.ExpiresAt)
	if err != nil || !now.Before(exp) {
		return nil, "", time.Time{}, bad
	}
	match := func(r attemptRoute) bool {
		return r.Worker == pf.Route.WorkerID && r.Model != nil && *r.Model == pf.Route.Model && r.Effort != nil && *r.Effort == pf.Route.ReasoningEffort
	}
	if !match(reqRoute) || !match(envRoute) {
		return nil, "", time.Time{}, bad
	}
	g, err := hoststate.RouteGrant(pf.Route)
	if err != nil {
		return nil, "", time.Time{}, bad
	}
	return &g, g.Identity, exp, nil
}

// unitCounters sums a unit's reservations across every generation: counters
// are monotonic for the run and never reset by a new lease.
func unitCounters(history []hoststate.Record, unit string) engine.LeaseCounters {
	var c engine.LeaseCounters
	for _, r := range history {
		if v := r.Reservation; r.Kind == hoststate.KindReservation && v != nil && v.UnitID == unit {
			c.Attempts++
			c.CostSpent += v.Cost
		}
	}
	return c
}
