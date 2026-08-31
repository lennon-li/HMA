package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lennon-li/HMA/internal/pilot"
)

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: hma [pilot|resolve] ...")
	}
	switch args[0] {
	case "pilot":
		return runPilot(args)
	case "resolve":
		return runResolve(args)
	default:
		return errors.New("usage: hma [pilot|resolve] ...")
	}
}

func runPilot(args []string) error {
	fs := flag.NewFlagSet("pilot", flag.ContinueOnError)
	input := fs.String("input", "", "host input file")
	store := fs.String("store", "", "host store directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *input == "" || *store == "" || fs.NArg() != 0 {
		return errors.New("usage: hma pilot --input <file> --store <directory>")
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var in pilot.Input
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
	result, err := pilot.Run(context.Background(), in, *store)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
