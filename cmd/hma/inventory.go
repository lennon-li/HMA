package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lennon-li/HMA/internal/inventory"
	"github.com/lennon-li/HMA/internal/repostate"
)

// inventorySummary is what runInventory prints. The full inventory, with its
// host-local executable paths, goes only to the --out file.
type inventorySummary struct {
	Out        string `json:"out"`
	Digest     string `json:"digest"`
	CapturedAt string `json:"captured_at"`
	FreshUntil string `json:"fresh_until"`
	Entries    int    `json:"entries"`
}

// runInventory captures host-local machine truth from an allowlist and writes
// it outside the repository. The result is availability evidence only: it
// selects no route, grants no eligibility, and changes no run.
func runInventory(args []string) error {
	fs := flag.NewFlagSet("inventory", flag.ContinueOnError)
	allowlistPath := fs.String("allowlist", "", "host-local allowlist file")
	repoRoot := fs.String("repo-root", "", "repository the inventory must stay outside")
	outPath := fs.String("out", "", "host-local output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *allowlistPath == "" || *repoRoot == "" || *outPath == "" || fs.NArg() != 0 {
		return errors.New("usage: hma inventory --allowlist <file> --repo-root <dir> --out <file>")
	}
	root, err := repositoryToplevel(*repoRoot)
	if err != nil {
		return err
	}
	// Both files are host configuration and host evidence; inside the
	// repository they would become part of the state HMA verifies.
	if err := repostate.StoreOutsideRepository(root, *allowlistPath); err != nil {
		return fmt.Errorf("allowlist: %w", err)
	}
	if err := repostate.StoreOutsideRepository(root, *outPath); err != nil {
		return fmt.Errorf("inventory output: %w", err)
	}

	f, err := os.Open(*allowlistPath)
	if err != nil {
		return err
	}
	allowlist, err := inventory.LoadAllowlist(f)
	f.Close()
	if err != nil {
		return err
	}

	inv, err := inventory.Discover(context.Background(), allowlist.Request(time.Now()))
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomic(*outPath, append(encoded, '\n')); err != nil {
		return err
	}
	// The store is host-trusted, so this detects rather than prevents a
	// parent directory swapped for a link into the repository mid-capture.
	if err := repostate.StoreOutsideRepository(root, *outPath); err != nil {
		_ = os.Remove(*outPath)
		return fmt.Errorf("inventory output moved into repository during capture: %w", err)
	}
	out, err := filepath.Abs(*outPath)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(inventorySummary{Out: out, Digest: inv.Digest, CapturedAt: inv.CapturedAt, FreshUntil: inv.FreshUntil, Entries: len(inv.Entries)})
}

// repositoryToplevel requires dir to be an existing directory inside a Git
// work tree and returns that work tree's top level. Checking containment
// against a mistyped or non-directory root would pass trivially.
func repositoryToplevel(dir string) (string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("repository root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root %q is not a directory", dir)
	}
	top, err := repostate.Output(context.Background(), dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("repository root %q is not inside a git work tree: %w", dir, err)
	}
	return repostate.Trim(top), nil
}

// writeFileAtomic replaces path so a reader never sees a partial inventory.
// The result has mode 0600 regardless of any file it replaces, and the parent
// directory is not synced: a capture interrupted by a crash is simply re-run.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".hma-inventory-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
