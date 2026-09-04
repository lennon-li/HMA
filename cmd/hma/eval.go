package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/lennon-li/HMA/internal/engine"
)

// hma eval exposes the deterministic evaluators as a command family. It is
// strictly read-only: it opens no store, appends no record, advances no
// stage, grants no approval, selects no route, and dispatches nothing. It
// reads one host input document, calls exactly one pure evaluator, and prints
// that evaluator's result. Any writing is the host's to do, with a human
// approval the evaluators never supply.

// evalOutcome is one evaluator's answer: the document to print, plus the
// decision and reason lifted out of it so the exit code can be derived
// without re-inspecting the document's type.
type evalOutcome struct {
	Document any
	Decision engine.Decision
	Reason   engine.Reason
}

// routeVerificationResult gives EvaluateRouteAttestation's two return values
// the same JSON shape the other evaluators already print. It adds no field
// the evaluator does not produce.
type routeVerificationResult struct {
	Decision engine.Decision `json:"decision"`
	Reason   engine.Reason   `json:"reason,omitempty"`
}

// evaluators maps a CLI evaluator name to the function that decodes its host
// input and evaluates it. Each entry calls exactly one engine evaluator and
// passes the decoded request through unchanged: the CLI supplies no default,
// fills in no field, and interprets no result.
var evaluators = map[string]func(path string) (evalOutcome, error){
	"transition": func(path string) (evalOutcome, error) {
		var req engine.TransitionRequest
		if err := decodeEvalInput(path, &req); err != nil {
			return evalOutcome{}, err
		}
		res := engine.EvaluateTransition(req)
		return evalOutcome{res, res.Decision, res.Reason}, nil
	},
	"invalidation": func(path string) (evalOutcome, error) {
		var req engine.InvalidationRequest
		if err := decodeEvalInput(path, &req); err != nil {
			return evalOutcome{}, err
		}
		res := engine.EvaluateInvalidation(req)
		return evalOutcome{res, res.Decision, res.Reason}, nil
	},
	"validator-contract": func(path string) (evalOutcome, error) {
		var req engine.ValidatorContractRequest
		if err := decodeEvalInput(path, &req); err != nil {
			return evalOutcome{}, err
		}
		res := engine.EvaluateValidatorContract(req)
		return evalOutcome{res, res.Decision, res.Reason}, nil
	},
	"route-verification": func(path string) (evalOutcome, error) {
		var req engine.RouteAttestationRequest
		if err := decodeEvalInput(path, &req); err != nil {
			return evalOutcome{}, err
		}
		decision, reason := engine.EvaluateRouteAttestation(req)
		return evalOutcome{routeVerificationResult{decision, reason}, decision, reason}, nil
	},
	"route-coherence": func(path string) (evalOutcome, error) {
		var req engine.RouteCoherenceRequest
		if err := decodeEvalInput(path, &req); err != nil {
			return evalOutcome{}, err
		}
		res := engine.EvaluateRouteCoherence(req)
		return evalOutcome{res, res.Decision, res.Reason}, nil
	},
	"route-policy": func(path string) (evalOutcome, error) {
		var req engine.RoutePolicyRequest
		if err := decodeEvalInput(path, &req); err != nil {
			return evalOutcome{}, err
		}
		res := engine.EvaluateRoutePolicy(req)
		return evalOutcome{res, res.Decision, res.Reason}, nil
	},
}

// evaluatorNames returns the evaluator names in a stable order, so the usage
// message does not depend on map iteration order.
func evaluatorNames() []string {
	names := make([]string, 0, len(evaluators))
	for name := range evaluators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func evalUsage() string {
	return "usage: hma eval [" + strings.Join(evaluatorNames(), "|") + "] --input <file>"
}

// decodeEvalInput reads exactly one JSON document from path into v. Unknown
// fields and a trailing second document are refused, so a host cannot smuggle
// an unread field past an evaluator or hide a second request behind the first.
func decodeEvalInput(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple input documents")
	}
	return nil
}

func runEval(args []string) error {
	if len(args) == 0 {
		return errors.New(evalUsage())
	}
	name := args[0]
	evaluate, known := evaluators[name]
	if !known {
		return errors.New(evalUsage())
	}

	fs := flag.NewFlagSet("eval "+name, flag.ContinueOnError)
	inputPath := fs.String("input", "", "host input file containing one evaluator request")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *inputPath == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: hma eval %s --input <file>", name)
	}

	outcome, err := evaluate(*inputPath)
	if err != nil {
		return err
	}

	// The evaluation is the deliverable, so the result document is printed
	// for a rejection too: a host needs the reason, not just a failed exit.
	// The rejection is still reported as an error, which exits non-zero.
	if err := json.NewEncoder(os.Stdout).Encode(outcome.Document); err != nil {
		return err
	}
	if outcome.Decision == engine.DecisionLegalPendingApproval {
		return nil
	}
	if outcome.Reason != "" {
		return errors.New(string(outcome.Reason))
	}
	return errors.New(string(outcome.Decision))
}
