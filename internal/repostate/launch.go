package repostate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
)

// launchConfig is the repository-independent configuration agent-run-ledger
// uses for its launch digests (compute_launch_digests). It must stay equal to
// that definition, or every receipt comparison fails closed.
var launchConfig = []string{
	"--no-replace-objects",
	"-c", "core.fileMode=true",
	"-c", "core.ignoreCase=false",
	"-c", "core.fsmonitor=false",
	"-c", "core.untrackedCache=false",
	"-c", "diff.external=",
}

// LaunchDigests returns the HEAD commit and the worktree digest that an
// agent-dispatch receipt records for a run launched in root now: sha256 over
// porcelain v2 status (all untracked files) followed by the binary diff
// against HEAD. It is a receipt comparison value, not an HMA artifact
// identity; HMA's own worktree digest is Snapshot's.
func LaunchDigests(ctx context.Context, root string) (string, string, error) {
	head, err := launchGit(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	status, err := launchGit(ctx, root, "status", "--porcelain=v2", "-z", "-uall", "--ignore-submodules=none")
	if err != nil {
		return "", "", err
	}
	diff, err := launchGit(ctx, root, "diff", "--binary", "--no-ext-diff", "--no-textconv", "HEAD")
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(append(status, diff...))
	return Trim(string(head)), hex.EncodeToString(sum[:]), nil
}

func launchGit(ctx context.Context, root string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, "git", append(append([]string{"-C", root}, launchConfig...), args...)...)
	c.Env = gitEnv(os.Environ())
	return c.Output()
}
