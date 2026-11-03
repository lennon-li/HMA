package main

import (
	"errors"
	"fmt"
	"os"
)

const usage = "usage: hmarun init --out-dir <dir> [--answers <file>]\n" +
	"       hmarun run --store <dir> --repo <dir> --run_id <id> --contract <file> --plan <file> [--unit <id>] [--auto-approve]"

func run(args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], os.Stdin, os.Stdout)
	case "run":
		return runRun(args[1:], os.Stdin, os.Stdout)
	default:
		return errors.New(usage)
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
