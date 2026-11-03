package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const initUsage = "usage: hmarun init --out-dir <dir> [--answers <file>]"

// loadAnswersFile strictly decodes an answers JSON file: an unknown field is
// almost always a typo (e.g. "nongoals" instead of "non_goals"), and a
// silent ignore would produce a document that quietly lacks what the human
// thought they asked for.
func loadAnswersFile(path string) (Answers, error) {
	var a Answers
	f, err := os.Open(path)
	if err != nil {
		return a, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return a, fmt.Errorf("decoding %s: %w", path, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return a, fmt.Errorf("%s: trailing data after the JSON document", path)
	}
	return a, nil
}

func runInit(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stdout)
	outDir := fs.String("out-dir", "", "directory to write contract.json and plan.json into")
	answersPath := fs.String("answers", "", "answers JSON file (non-interactive mode); omit to be prompted interactively")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outDir == "" || fs.NArg() != 0 {
		return errors.New(initUsage)
	}

	var a Answers
	if *answersPath != "" {
		var err error
		a, err = loadAnswersFile(*answersPath)
		if err != nil {
			return err
		}
	} else {
		var err error
		a, err = collectAnswers(newPrompter(stdin, stdout))
		if err != nil {
			return err
		}
	}

	contract, _, plan, err := buildContractAndPlan(a)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	contractBytes, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return err
	}
	planBytes, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*outDir, "contract.json"), append(contractBytes, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*outDir, "plan.json"), append(planBytes, '\n'), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "wrote %s\n", filepath.Join(*outDir, "contract.json"))
	fmt.Fprintf(stdout, "wrote %s\n", filepath.Join(*outDir, "plan.json"))
	return nil
}
