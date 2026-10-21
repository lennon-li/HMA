package evidence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func sandboxed(t *testing.T, root string, argv ...string) int {
	t.Helper()
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not installed")
	}
	res, err := Capture(context.Background(), Request{Executable: "/usr/bin/test", Argv: argv, WorkingDir: root, RepositoryRoot: root, Sandbox: true})
	if err != nil {
		t.Fatal(err)
	}
	return res.Evidence.ExitCode
}

// A sandboxed command sees the copy and nothing else of the host's files.
func TestSandboxShowsOnlyTheCopy(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(os.Getenv("HOME"), ".hma-sandbox-probe-"+filepath.Base(root))
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Skip("cannot write probe in HOME")
	}
	defer os.Remove(outside)
	os.WriteFile(filepath.Join(root, "in"), []byte("x"), 0o600)
	if sandboxed(t, root, "-f", filepath.Join(root, "in")) != 0 {
		t.Fatal("copy file not visible")
	}
	if sandboxed(t, root, "-e", outside) == 0 {
		t.Fatal("host file visible in the sandbox")
	}
	if sandboxed(t, root, "-d", "/proc/1") != 0 {
		// own pid namespace: pid 1 is the sandbox itself, still a directory
		t.Fatal("proc not mounted")
	}
}

// D5a r19 D5AR19P-F01: a repository inside a directory the sandbox shows
// whole is refused before the sandbox starts, so it is never visible beside
// the private copy.
func TestSandboxRefusesRepositoryInsideSystemDirectory(t *testing.T) {
	for _, live := range []string{"/usr/local/src/x", "/usr/share", "/bin", "/lib", "/etc/ssl", "/etc/alternatives", "/etc/ld.so.conf.d", "/etc"} {
		if _, err := os.Stat(live); err != nil {
			continue
		}
		_, err := sandboxCommand(context.Background(), "/usr/bin/true", nil, t.TempDir(), t.TempDir(), live, "", "", nil)
		if err == nil || !(strings.Contains(err.Error(), "system directory") || strings.Contains(err.Error(), "overlaps system path")) {
			t.Fatalf("%s: not refused before the sandbox probe: %v", live, err)
		}
	}
}
