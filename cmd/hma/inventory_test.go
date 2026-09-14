package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lennon-li/HMA/internal/inventory"
)

func writeInventoryAllowlist(t *testing.T, path string) {
	t.Helper()
	writeJSON(t, path, map[string]any{
		"version":           inventory.AllowlistVersionV1,
		"freshness_seconds": 600,
		"commands": []map[string]any{
			{"id": "printf", "kind": "version", "subject": "printf", "executable": "/usr/bin/printf", "argv": []string{"%s\\n", "tool 1.0"}},
			{"id": "missing", "kind": "executable_presence", "subject": "missing", "executable": "/hma/missing/tool"},
		},
	})
}

// gitRepo returns a fresh directory that is the top level of a Git work tree.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

func TestInventoryWritesVerifiableHostLocalRecord(t *testing.T) {
	repo, host := gitRepo(t), t.TempDir()
	allowlist, out := filepath.Join(host, "allowlist.json"), filepath.Join(host, "inventory.json")
	writeInventoryAllowlist(t, allowlist)

	if err := run([]string{"inventory", "--allowlist", allowlist, "--repo-root", repo, "--out", out}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var inv inventory.Inventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		t.Fatal(err)
	}
	digest, err := inventory.Digest(inv)
	if err != nil || digest != inv.Digest {
		t.Fatalf("written inventory does not verify: %q != %q (err %v)", digest, inv.Digest, err)
	}
	if len(inv.Entries) != 2 || inv.Entries[0].Status != inventory.StatusAbsent || inv.Entries[1].Version != "tool 1.0" {
		t.Fatalf("entries = %+v", inv.Entries)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(host, ".hma-inventory-*")); len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}
}

func TestInventoryRefusesPathsInsideRepository(t *testing.T) {
	repo, host := gitRepo(t), t.TempDir()
	outside := filepath.Join(host, "allowlist.json")
	inside := filepath.Join(repo, "allowlist.json")
	writeInventoryAllowlist(t, outside)
	writeInventoryAllowlist(t, inside)
	link := filepath.Join(host, "into-repo")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}

	for name, args := range map[string][]string{
		"allowlist inside":        {"--allowlist", inside, "--out", filepath.Join(host, "out.json")},
		"output inside":           {"--allowlist", outside, "--out", filepath.Join(repo, "out.json")},
		"output through symlink":  {"--allowlist", outside, "--out", filepath.Join(link, "out.json")},
		"allowlist through link":  {"--allowlist", filepath.Join(link, "allowlist.json"), "--out", filepath.Join(host, "out.json")},
		"missing output argument": {"--allowlist", outside},
	} {
		t.Run(name, func(t *testing.T) {
			err := run(append([]string{"inventory", "--repo-root", repo}, args...))
			if err == nil {
				t.Fatal("run() succeeded, want refusal")
			}
			if name != "missing output argument" && !strings.Contains(err.Error(), "outside target repository") {
				t.Fatalf("error = %v", err)
			}
		})
	}
	// .git plus the planted allowlist.json, and nothing written by a refusal.
	if entries, _ := os.ReadDir(repo); len(entries) != 2 {
		t.Fatalf("repository gained files: %v", entries)
	}
}

func TestInventoryRejectsInvalidAllowlistBeforeRunning(t *testing.T) {
	repo, host := gitRepo(t), t.TempDir()
	allowlist, out := filepath.Join(host, "allowlist.json"), filepath.Join(host, "inventory.json")
	writeJSON(t, allowlist, map[string]any{
		"version":           inventory.AllowlistVersionV1,
		"freshness_seconds": 600,
		"commands":          []map[string]any{{"id": "sh", "kind": "version", "subject": "sh", "executable": "sh"}},
	})
	if err := run([]string{"inventory", "--allowlist", allowlist, "--repo-root", repo, "--out", out}); err == nil {
		t.Fatal("run() accepted a bare executable name")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output written despite invalid allowlist: %v", err)
	}
}

func TestInventoryRequiresRealRepositoryRoot(t *testing.T) {
	repo, host := gitRepo(t), t.TempDir()
	allowlist := filepath.Join(host, "allowlist.json")
	writeInventoryAllowlist(t, allowlist)
	notRepo := t.TempDir()
	file := filepath.Join(host, "plain-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(repo, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	for name, root := range map[string]string{
		"nonexistent":   filepath.Join(host, "missing-repo"),
		"regular file":  file,
		"not a git dir": notRepo,
	} {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(host, name+".json")
			if err := run([]string{"inventory", "--allowlist", allowlist, "--repo-root", root, "--out", out}); err == nil {
				t.Fatal("run() accepted an invalid repository root")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("output written despite invalid root: %v", err)
			}
		})
	}

	// A subdirectory root still guards the whole work tree.
	err := run([]string{"inventory", "--allowlist", allowlist, "--repo-root", subdir, "--out", filepath.Join(repo, "out.json")})
	if err == nil || !strings.Contains(err.Error(), "outside target repository") {
		t.Fatalf("subdirectory root let output into the work tree: %v", err)
	}
}
