package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
)

type hostBindRepositoryInput struct {
	hostEnvelope
	RepositoryRoot     string `json:"repository_root"`
	RepositoryIdentity string `json:"repository_identity"`
}

// runHostBindRepository records the run's one repository: its canonical root
// and the repository identity its core approvals bind. It is an explicit host
// step before the first review packet or loop evidence, never inferred from a
// packet or journal, and it is written once: a second binding is refused. A
// run whose core approvals bind another identity, or none, is refused.
func runHostBindRepository(args []string) error {
	inputPath, storePath, err := hostCommandFlags("bind-repository", args)
	if err != nil {
		return err
	}
	var in hostBindRepositoryInput
	if err := decodeStrictFile(inputPath, &in); err != nil {
		return err
	}
	if _, err := validateHostEnvelope(in.hostEnvelope); err != nil {
		return err
	}
	if in.RepositoryRoot == "" || in.RepositoryIdentity == "" {
		return errors.New(hoststate.ReasonInvalidHostRecord)
	}
	root, err := filepath.Abs(in.RepositoryRoot)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return err
	}
	top, err := repostate.Output(context.Background(), root, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	if top, err = filepath.EvalSymlinks(repostate.Trim(top)); err != nil || top != root {
		return errors.New(hoststate.ReasonRepositoryMismatch)
	}
	if err := repostate.StoreOutsideRepository(root, storePath); err != nil {
		return err
	}
	var rec hoststate.Record
	err = hoststate.NewStore(storePath).Transact(in.RunID, func(history []hoststate.Record, commit func(*hoststate.Record) error) error {
		return store.New(storePath).WithRunLock(in.RunID, func(core []model.Record) error {
			if len(core) == 0 {
				return errors.New(hoststate.ReasonCoreRunMissing)
			}
			if engine.ProjectStageState(core).Terminal {
				return errors.New(hoststate.ReasonCoreRunTerminal)
			}
			if hoststate.RunRepository(history) != nil {
				return errors.New(hoststate.ReasonRepositoryBindingExists)
			}
			bound := false
			for _, r := range core {
				if r.Kind == model.KindStageTransition && r.Control == model.ControlApproved && r.Approval != nil {
					if r.Approval.RepositoryIdentityDigest != in.RepositoryIdentity {
						return errors.New(hoststate.ReasonRepositoryMismatch)
					}
					bound = true
				}
			}
			if !bound {
				return errors.New(hoststate.ReasonRepositoryMismatch)
			}
			rec = hoststate.Record{Version: 1, Kind: hoststate.KindRepositoryBinding, RunID: in.RunID,
				CoreHeadAnchor: core[len(core)-1].Metadata.HeadAnchor, Timestamp: in.Timestamp, Actor: in.Actor,
				Repository: &hoststate.RepositoryBinding{Root: root, RepositoryIdentity: in.RepositoryIdentity}}
			return commit(&rec)
		})
	})
	if err != nil {
		return err
	}
	res := hostResultFor(storePath, rec)
	// The binding enables review packets and loop evidence for the run.
	res.NextAction = "GENERATE_REVIEW_PACKET_OR_IMPORT_LOOP_EVIDENCE"
	return json.NewEncoder(os.Stdout).Encode(res)
}
