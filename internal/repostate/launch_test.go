package repostate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// LaunchDigests must equal agent-run-ledger's compute_launch_digests, which
// writes the values a receipt carries. Skipped where that script is absent.
func TestLaunchDigestsMatchLedger(t *testing.T) {
	home, _ := os.UserHomeDir()
	ledger := filepath.Join(home, "obsidian", "AgentSystem", "scripts", "agent-run-ledger")
	if _, err := os.Stat(ledger); err != nil {
		t.Skip("agent-run-ledger not present")
	}
	repo, _ := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "x"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "new"), []byte("u"), 0o600); err != nil {
		t.Fatal(err)
	}
	head, wt, err := LaunchDigests(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	// Extract and run only the function definition from the ledger script.
	src, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	start := strings.Index(s, "compute_launch_digests() {")
	end := strings.Index(s[start:], "\n}\n")
	if start < 0 || end < 0 {
		t.Fatal("compute_launch_digests not found")
	}
	script := s[start:start+end+3] + `compute_launch_digests "$1"; printf '%s %s' "$launch_head" "$launch_worktree_sha256"`
	out, err := exec.Command("bash", "-c", script, "bash", repo).Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Fields(string(out)); len(got) != 2 || got[0] != head || got[1] != wt {
		t.Fatalf("ledger %q, HMA %s %s", out, head, wt)
	}
}
