package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"github.com/lennon-li/HMA/internal/transition"
)

// runTransition records one human-approved stage, classification, or supported outcome. It is the only
// command that changes a run's stage, and it does so only by recording an
// approval a human already gave; it never chooses a target or supplies a
// decision of its own.
func runTransition(args []string) error {
	fs := flag.NewFlagSet("transition", flag.ContinueOnError)
	inputPath := fs.String("input", "", "host input file containing one approved stage, classification, or supported outcome")
	storePath := fs.String("store", "", "host store directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inputPath == "" || *storePath == "" || fs.NArg() != 0 {
		return errors.New("usage: hma transition --input <file> --store <directory>")
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var in transition.Input
	if err := dec.Decode(&in); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple input documents")
	}

	result, err := transition.Apply(context.Background(), in, *storePath)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
