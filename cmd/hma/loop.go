package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/evidence"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

type hostLoopEvidenceInput struct {
	hostEnvelope
	UnitID                string `json:"unit_id"`
	StatePath             string `json:"state_path"`
	RepositoryRoot        string `json:"repository_root"`
	RepositoryIdentity    string `json:"repository_identity"`
	BaseRevision          string `json:"base_revision"`
	DirtyWorktreeApproved bool   `json:"dirty_worktree_approved"`
}

// loopState is the part of a verify-loop state.json HMA reads.
type loopState struct {
	RunID string `json:"run_id"`
	Repo  *struct {
		Root string `json:"root"`
	} `json:"repo"`
	Status         string          `json:"status"`
	EnvelopeSHA256 string          `json:"envelope_sha256"`
	EnvelopeRaw    json.RawMessage `json:"-"`
	Envelope       struct {
		HMA *struct {
			Store  string `json:"store"`
			RunID  string `json:"run_id"`
			UnitID string `json:"unit_id"`
		} `json:"hma"`
		AuthorityHook *struct {
			Argv []string `json:"argv"`
		} `json:"authority_hook"`
		EscalateTo string `json:"escalate_to"`
		Verify     struct {
			Argv []string `json:"argv"`
		} `json:"verify"`
	} `json:"envelope"`
	Attempts []struct {
		N             int    `json:"n"`
		DispatchRunID string `json:"dispatch_run_id"`
		Verify        *struct {
			ExitCode  *int   `json:"exit_code"`
			Signature string `json:"signature"`
		} `json:"verify"`
		Regression *struct {
			ExitCode *int `json:"exit_code"`
		} `json:"regression"`
	} `json:"attempts"`
}

type loopEvidenceResult struct {
	hostResult
	RecaptureExitCode int `json:"recapture_exit_code"`
}

// runHostLoopEvidence imports one HMA-bound verify-loop run as evidence.
// HMA does not trust the loop's own check result: it re-runs the loop's
// targeted check argv itself, without a shell, through the same capture as
// CI evidence, against the snapshot it observes before and after, and
// appends that as a core evidence record. The host record then binds the
// loop state bytes, its attempts, and that core record. Nothing here credits
// an acceptance criterion or authorizes a launch.
//
// The whole operation holds the host run lock, which hma transition also
// takes first, and then the core run lock, so no terminal outcome or rewind
// (through HMA or a direct core append) can land between the checks, the
// re-capture, and the appends.
func runHostLoopEvidence(args []string) error {
	inputPath, storePath, err := hostCommandFlags("loop-evidence", args)
	if err != nil {
		return err
	}
	var in hostLoopEvidenceInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	if in.UnitID == "" || in.StatePath == "" || in.RepositoryRoot == "" || in.RepositoryIdentity == "" || in.BaseRevision == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	if err := repostate.StoreOutsideRepository(in.RepositoryRoot, storePath); err != nil {
		return err
	}
	raw, err := os.ReadFile(in.StatePath)
	if err != nil {
		return err
	}
	st, attempts, err := parseLoopState(raw, in, storePath)
	if err != nil {
		return err
	}
	absState, err := filepath.Abs(in.StatePath)
	if err != nil {
		return err
	}
	baseline, err := journalBaselineDigest(raw)
	if err != nil {
		return err
	}
	argv := st.Envelope.Verify.Argv
	// Run the canonical file the filter judged, not the alias.
	if exe, err := canonicalExecutable(argv[0]); err != nil {
		return errors.New(hoststate.ReasonLoopEvidenceInvalid)
	} else {
		argv = append([]string{exe}, argv[1:]...)
	}
	var out loopEvidenceResult
	var rec hoststate.Record
	err = hoststate.NewStore(storePath).Transact(in.RunID, func(history []hoststate.Record, commit func(*hoststate.Record) error) error {
		// The core run lock is held (host lock first, then core) from the
		// terminal and route checks through the check run and the core
		// append, so no core record, including a direct terminal append,
		// can land in between.
		return store.New(storePath).Transact(in.RunID, func(core []model.Record, appendCore func(*model.Record) error) error {
			if len(core) == 0 {
				return errors.New(hoststate.ReasonCoreRunMissing)
			}
			stage := engine.ProjectStageState(core)
			if stage.Terminal {
				return errors.New(hoststate.ReasonCoreRunTerminal)
			}
			// The unit must hold an approved route, as for any host work.
			if _, err := approvedCoreRouteDigestIn(core, in.RunID, in.UnitID); err != nil {
				return err
			}
			// The check runs only in the run's bound repository, at the unit's
			// approved base, and the journal must name that repository root.
			repo := hoststate.RunRepository(history)
			if repo == nil {
				return errors.New(hoststate.ReasonRepositoryBindingMissing)
			}
			route, err := approvedRoute(core, in.RunID, in.UnitID)
			if err != nil {
				return err
			}
			if !sameStore(in.RepositoryRoot, repo.Root) || in.RepositoryIdentity != repo.RepositoryIdentity ||
				route.RepositoryIdentityDigest != repo.RepositoryIdentity || in.BaseRevision != route.BaseRevisionDigest ||
				st.Repo == nil || !sameStore(st.Repo.Root, repo.Root) {
				return errors.New(hoststate.ReasonRepositoryMismatch)
			}
			// A loop attempt may not reuse a review launch's run id.
			for _, r := range history {
				if r.Kind == hoststate.KindDispatch && r.Dispatch != nil && r.Dispatch.Packet != nil {
					for _, a := range attempts {
						if a.DispatchRunID == r.Dispatch.Packet.DispatchRunID {
							return errors.New(hoststate.ReasonLoopEvidenceInvalid)
						}
					}
				}
			}
			ctx := context.Background()
			before, err := repostate.Snapshot(ctx, in.RepositoryRoot, in.BaseRevision, in.DirtyWorktreeApproved)
			if err != nil {
				return err
			}
			captured, err := evidence.Capture(ctx, evidence.Request{
				Executable: argv[0], Argv: argv[1:], WorkingDir: in.RepositoryRoot, RepositoryRoot: in.RepositoryRoot,
				MaxOutputBytes: 1 << 20, RepositoryIdentity: in.RepositoryIdentity, BaseRevision: in.BaseRevision,
				HeadRevision: before.Head, DiffDigest: before.Diff, WorktreeDigest: before.Worktree, CapturingActor: "hma:" + in.Actor,
			})
			if err != nil {
				return err
			}
			after, err := repostate.Snapshot(ctx, in.RepositoryRoot, in.BaseRevision, in.DirtyWorktreeApproved)
			if err != nil || after != before {
				return errors.New("worktree changed during loop check re-capture")
			}
			// Keep the captured streams, so the evidence output digest can be
			// recomputed later from durable bytes.
			ref, err := retainOutput(storePath, captured.Stdout, captured.Stderr)
			if err != nil {
				return err
			}
			captured.Evidence.OutputRef = ref
			last := core[len(core)-1]
			ev := model.Record{
				Kind: model.KindEvidence, Version: 1, UnitID: in.UnitID, Stage: stage.CurrentStage, Evidence: &captured.Evidence,
				Metadata: model.RecordMetadata{RunID: in.RunID, Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor,
					Timestamp: in.Timestamp, Actor: "hma:" + in.Actor},
			}
			if err := appendCore(&ev); err != nil {
				return err
			}
			stateSum := sha256.Sum256(raw)
			argvJSON, _ := json.Marshal(argv)
			argvSum := sha256.Sum256(argvJSON)
			rec = hoststate.Record{Version: 1, Kind: hoststate.KindLoopEvidence, RunID: in.RunID, CoreHeadAnchor: ev.Metadata.HeadAnchor,
				Timestamp: in.Timestamp, Actor: in.Actor, LoopEvidence: &hoststate.LoopEvidence{
					UnitID: in.UnitID, ControllerRunID: st.RunID, EnvelopeSHA256: st.EnvelopeSHA256,
					StateSHA256: hex.EncodeToString(stateSum[:]), LoopStatus: st.Status, Attempts: attempts,
					RecaptureRecord: ev.Metadata.HeadAnchor, RecaptureExitCode: captured.Evidence.ExitCode,
					RecaptureArgvSHA256: hex.EncodeToString(argvSum[:]),
					LedgerNamespace:     filepath.Join(filepath.Dir(absState), "dispatch-ledger"),
					LiveRoot:            repo.Root,
					CloneBaselineDigest: baseline,
				}}
			if err := commit(&rec); err != nil {
				return err
			}
			out = loopEvidenceResult{RecaptureExitCode: captured.Evidence.ExitCode}
			return nil
		})
	})
	if err != nil {
		return err
	}
	// Built after the locks are released, through the shared result helper.
	out.hostResult = hostResultFor(storePath, rec)
	out.NextAction = "EVIDENCE_RECORDED"
	if out.RecaptureExitCode != 0 || !strings.EqualFold(st.Status, "green") {
		out.NextAction = "REMEDIATE_OR_ESCALATE"
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

// journalBaselineDigest is verify-loop's clone baseline digest of a state
// file: the canonical digest of repo.workspace_baseline_inventory, {} when
// absent.
func journalBaselineDigest(raw []byte) (string, error) {
	var top map[string]json.RawMessage
	var repo map[string]json.RawMessage
	if json.Unmarshal(raw, &top) != nil || json.Unmarshal(top["repo"], &repo) != nil {
		return "", errors.New(hoststate.ReasonLoopEvidenceInvalid)
	}
	inv := repo["workspace_baseline_inventory"]
	if len(inv) == 0 {
		inv = json.RawMessage(`{}`)
	}
	return hoststate.PythonCanonicalSHA256(inv)
}

// parseLoopState checks a verify-loop state file is an HMA-bound loop for
// this run, unit, and store: the envelope digest is the canonical hash of the
// envelope as verify-loop computes it, the envelope names the authority hook
// and escalate_to stop, and attempt dispatch ids are present and unique.
func parseLoopState(raw []byte, in hostLoopEvidenceInput, storePath string) (loopState, []hoststate.LoopAttempt, error) {
	bad := errors.New(hoststate.ReasonLoopEvidenceInvalid)
	var st loopState
	var envelope struct {
		Envelope json.RawMessage `json:"envelope"`
	}
	// Exact keys, as verify-loop (Python) reads them; the envelope is hashed
	// from its original bytes.
	var top map[string]json.RawMessage
	if hoststate.UnmarshalExact(raw, &st) != nil || json.Unmarshal(raw, &top) != nil {
		return st, nil, bad
	}
	envelope.Envelope = top["envelope"]
	if len(envelope.Envelope) == 0 {
		return st, nil, bad
	}
	if st.RunID == "" || st.EnvelopeSHA256 == "" {
		return st, nil, bad
	}
	if sum, err := hoststate.PythonCanonicalSHA256(envelope.Envelope); err != nil || sum != st.EnvelopeSHA256 {
		return st, nil, bad
	}
	h := st.Envelope.HMA
	if h == nil || h.RunID != in.RunID || h.UnitID != in.UnitID || !sameStore(h.Store, storePath) {
		return st, nil, errors.New("HOST_LOOP_NOT_BOUND_TO_RUN")
	}
	if st.Envelope.AuthorityHook == nil || len(st.Envelope.AuthorityHook.Argv) == 0 || st.Envelope.EscalateTo != "stop" {
		return st, nil, bad
	}
	if argv := st.Envelope.Verify.Argv; len(argv) == 0 || argv[0] == "" || unsafeLoopArgv(argv) {
		return st, nil, bad
	}
	attempts := make([]hoststate.LoopAttempt, 0, len(st.Attempts))
	seen := map[string]bool{}
	for _, a := range st.Attempts {
		if a.DispatchRunID == "" || seen[a.DispatchRunID] {
			return st, nil, bad
		}
		seen[a.DispatchRunID] = true
		la := hoststate.LoopAttempt{N: a.N, DispatchRunID: a.DispatchRunID}
		if a.Verify != nil {
			la.VerifyExit, la.VerifySignature = a.Verify.ExitCode, a.Verify.Signature
		}
		if a.Regression != nil {
			la.RegressionExit = a.Regression.ExitCode
		}
		attempts = append(attempts, la)
	}
	return st, attempts, nil
}

func sameStore(a, b string) bool {
	ra, err := filepath.Abs(a)
	if err == nil {
		ra, err = filepath.EvalSymlinks(ra)
	}
	if err != nil {
		return false
	}
	rb, err := filepath.Abs(b)
	if err == nil {
		rb, err = filepath.EvalSymlinks(rb)
	}
	return err == nil && ra == rb
}

// retainOutput writes the captured stdout and stderr into the store as one
// content-addressed artifact, <store>/artifacts/sha256-<hex>.json, and
// returns its reference. The artifact holds both streams separately, so the
// evidence OutputDigest (length-framed stdout then stderr) can be recomputed.
// The bytes are written and synced to a temporary file first and then
// linked to the final name, which never replaces a file: the final name only
// ever holds complete bytes. An existing artifact is used only if it is a
// regular file with exactly these bytes; anything else (for example crash
// residue from an older writer) is refused and left for diagnosis.
func retainOutput(storePath string, stdout, stderr []byte) (string, error) {
	b, err := json.Marshal(struct {
		Schema string `json:"schema"`
		Stdout []byte `json:"stdout"`
		Stderr []byte `json:"stderr"`
	}{"hma.evidence-output/v1", stdout, stderr})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	name := "sha256-" + hex.EncodeToString(sum[:]) + ".json"
	return "artifacts/" + name, retainArtifact(storePath, name, b)
}

// retainArtifact writes b as <store>/artifacts/name, once: the bytes are
// written and synced to a temporary file first and then linked to the final
// name, which never replaces a file. An existing artifact is used only if it
// is a regular file with exactly these bytes.
func retainArtifact(storePath, name string, b []byte) error {
	dir := filepath.Join(storePath, "artifacts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	if _, err := os.Lstat(path); err == nil {
		return checkArtifact(path, b)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-"+name+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(b)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if err := os.Link(tmp.Name(), path); errors.Is(err, os.ErrExist) {
		return checkArtifact(path, b)
	} else if err != nil {
		return err
	}
	if err := syncParent(path); err != nil {
		return err
	}
	return syncParent(dir)
}

// planArtifactName names a stored hma.plan/v1 document by its digest.
func planArtifactName(digest string) string { return "plan-" + digest + ".json" }

// retainPlan stores the canonical form of an accepted plan (whose SHA-256 is
// its digest) so reconciliation can use each lease generation's exact plan.
func retainPlan(storePath string, plan model.Plan, digest string) error {
	b, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	if sum := sha256.Sum256(b); hex.EncodeToString(sum[:]) != digest {
		return errors.New(hoststate.ReasonEvidenceArtifactCorrupt)
	}
	return retainArtifact(storePath, planArtifactName(digest), b)
}

// loadPlans returns every stored plan an approval in core binds, by digest.
// A missing plan is absent from the map; a corrupt one is an error.
func loadPlans(storePath string, core []model.Record) (map[string]model.Plan, error) {
	out := map[string]model.Plan{}
	for _, r := range core {
		if r.Approval == nil || r.Approval.AcceptedPlanDigest == "" {
			continue
		}
		d := r.Approval.AcceptedPlanDigest
		if _, ok := out[d]; ok || filepath.Base(d) != d {
			continue
		}
		b, err := os.ReadFile(filepath.Join(storePath, "artifacts", planArtifactName(d)))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		p, got, err := model.ParsePlan(b)
		if err != nil || got != d {
			return nil, errors.New(hoststate.ReasonEvidenceArtifactCorrupt)
		}
		out[d] = p
	}
	return out, nil
}

// checkArtifact refuses an existing artifact that is not a regular file
// holding exactly want.
func checkArtifact(path string, want []byte) error {
	bad := errors.New(hoststate.ReasonEvidenceArtifactCorrupt)
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return bad
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		return bad
	}
	return nil
}

// canonicalExecutable resolves name through PATH and every symlink to the
// file that runs; any resolution error refuses the check.
func canonicalExecutable(name string) (string, error) {
	p := name
	if !filepath.IsAbs(p) {
		if strings.ContainsRune(p, filepath.Separator) {
			return "", errors.New("relative executable path")
		}
		var err error
		if p, err = exec.LookPath(p); err != nil {
			return "", err
		}
	}
	return filepath.EvalSymlinks(p)
}

// unsafeLoopArgv applies verify-loop's reject_unsafe_argv to a verify argv,
// and additionally refuses any shell (the capture refuses it again): a
// journal whose check verify-loop itself would refuse is not admitted.
func unsafeLoopArgv(argv []string) bool {
	canonical, err := canonicalExecutable(argv[0])
	if err != nil {
		return true
	}
	if evidence.RejectShell(argv[0], canonical, argv[1:]) != "" {
		return true
	}
	// The operation rules below read the executable's canonical name, so a
	// symlink alias (check-tool -> rm) is judged as what it runs.
	argv = append([]string{canonical}, argv[1:]...)
	unsafe := map[string]bool{"sudo": true, "ssh": true, "scp": true, "sftp": true, "rsync": true, "shutdown": true, "reboot": true,
		"poweroff": true, "halt": true, "mkfs": true, "mkfs.ext4": true, "dd": true, "fdisk": true, "mkswap": true, "swapon": true,
		"swapoff": true, "curl": true, "wget": true, "nc": true, "ncat": true, "telnet": true, "ftp": true}
	for i, a := range argv {
		if a == "" {
			return true
		}
		base := path.Base(a)
		if unsafe[base] {
			return true
		}
		next := ""
		if i+1 < len(argv) {
			next = argv[i+1]
		}
		switch base {
		case "git", "git.exe":
			if !evidence.GitFormAllowed(argv[i+1:]) {
				return true
			}
		case "gh":
			if next == "pr" || next == "release" || next == "auth" {
				return true
			}
		case "npm":
			if next == "publish" {
				return true
			}
		case "rm":
			for _, f := range argv[i+1 : min(i+3, len(argv))] {
				if f == "-rf" || f == "-fr" || f == "-r" || f == "--recursive" {
					return true
				}
			}
		}
	}
	return false
}
