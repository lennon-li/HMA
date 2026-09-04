package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"sort"

	"github.com/lennon-li/HMA/internal/engine"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

// showDocument is the printable shape of engine.ResolutionState.
//
// The projection keys its active waivers by model.ScopeSelector, a struct,
// which encoding/json cannot use as an object key, so the selectors are
// emitted as a list. Both the selector list and the nonce list are sorted:
// they come from maps, and unsorted output would differ between two
// invocations over the same chain.
//
// This is a rendering of the projection, not a second projection. It derives
// no terminal outcome and adds no field the engine did not compute.
type showDocument struct {
	Criteria      []model.Criterion     `json:"criteria"`
	Findings      []model.Finding       `json:"findings"`
	ActiveWaivers []model.ScopeSelector `json:"active_waivers"`
	UsedNonces    []string              `json:"used_nonces"`
}

func newShowDocument(state engine.ResolutionState) showDocument {
	doc := showDocument{
		Criteria:      state.Criteria,
		Findings:      state.Findings,
		ActiveWaivers: make([]model.ScopeSelector, 0, len(state.ActiveWaivers)),
		UsedNonces:    make([]string, 0, len(state.UsedNonces)),
	}
	if doc.Criteria == nil {
		doc.Criteria = []model.Criterion{}
	}
	if doc.Findings == nil {
		doc.Findings = []model.Finding{}
	}
	for selector, active := range state.ActiveWaivers {
		if active {
			doc.ActiveWaivers = append(doc.ActiveWaivers, selector)
		}
	}
	sort.Slice(doc.ActiveWaivers, func(i, j int) bool {
		if doc.ActiveWaivers[i].Kind != doc.ActiveWaivers[j].Kind {
			return doc.ActiveWaivers[i].Kind < doc.ActiveWaivers[j].Kind
		}
		return doc.ActiveWaivers[i].Target < doc.ActiveWaivers[j].Target
	})
	for nonce := range state.UsedNonces {
		doc.UsedNonces = append(doc.UsedNonces, nonce)
	}
	sort.Strings(doc.UsedNonces)
	return doc
}

// runShow prints the resolution projection of a committed chain. It is
// read-only: it loads the chain, replays it through the engine's projection,
// and prints the result. It appends nothing and decides nothing.
func runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	storePath := fs.String("store", "", "host store directory")
	runID := fs.String("run_id", "", "run ID to project")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePath == "" || *runID == "" || fs.NArg() != 0 {
		return errors.New("usage: hma show --store <directory> --run_id <id>")
	}

	records, err := store.New(*storePath).Load(*runID)
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(newShowDocument(engine.ProjectResolutionState(records)))
}
