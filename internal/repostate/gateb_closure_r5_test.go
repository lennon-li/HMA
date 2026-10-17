package repostate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// R5-01: a tracked directory replaced by an ignored symlink to identical
// bytes is a change. Git reports dir/tracked deleted; the snapshot must too.
func TestGateBClosureR5DirectoryBecomesSymlink(t *testing.T) {
	for _, alsoDirty := range []bool{false, true} {
		repo, _ := newRepo(t)
		if err := os.MkdirAll(filepath.Join(repo, "dir"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, "dir", "tracked"), []byte("same"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/dir\n/copy/\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		git(t, repo, "add", "-f", ".gitignore", "dir/tracked")
		git(t, repo, "commit", "-qm", "dir")
		base := Trim(git(t, repo, "rev-parse", "HEAD"))
		if alsoDirty {
			// Another staged change, so the before state is dirty and the
			// digest itself must tell the two states apart.
			if err := os.WriteFile(filepath.Join(repo, "x"), []byte("two"), 0o600); err != nil {
				t.Fatal(err)
			}
			git(t, repo, "add", "x")
		}
		before, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(repo, "dir"), filepath.Join(repo, "copy")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("copy", filepath.Join(repo, "dir")); err != nil {
			t.Fatal(err)
		}
		if _, err := Snapshot(context.Background(), repo, base, false); err == nil {
			t.Fatal("symlinked tracked directory accepted as clean")
		}
		got, err := Snapshot(context.Background(), repo, base, true)
		if err != nil {
			t.Fatal(err)
		}
		if got.Worktree == "" || got.Worktree == before.Worktree {
			t.Fatalf("worktree identity unchanged: %q", got.Worktree)
		}
	}
}
