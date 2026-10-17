package host

import (
	"os"
	"path/filepath"
	"testing"
)

// R4-02: every directory holding a followed symlink, including intermediate
// hops, is synced, not only the lexical and final resolved chains.
func TestGateBClosureR4IntermediateSymlinkParents(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"a", "hop", "real/store"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// a/s -> ../hop/l (relative), hop/l -> <base>/real (absolute).
	if err := os.Symlink(filepath.Join(base, "real"), filepath.Join(base, "hop", "l")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "hop", "l"), filepath.Join(base, "a", "s")); err != nil {
		t.Fatal(err)
	}
	dirs, real, err := lookupDirs(filepath.Join(base, "a", "s", "store"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "real", "store"); real != want {
		t.Fatalf("resolved %s, want %s", real, want)
	}
	for _, want := range []string{base, filepath.Join(base, "a"), filepath.Join(base, "hop"), filepath.Join(base, "real"), "/"} {
		if !dirs[want] {
			t.Errorf("directory %s not in synced set %v", want, dirs)
		}
	}
	if err := syncAncestors(filepath.Join(base, "a", "s", "store")); err != nil {
		t.Fatal(err)
	}
}

func TestGateBClosureR4SymlinkLoopFails(t *testing.T) {
	base := t.TempDir()
	if err := os.Symlink("loop", filepath.Join(base, "loop")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lookupDirs(filepath.Join(base, "loop", "x")); err == nil {
		t.Fatal("symlink loop resolved")
	}
}
