package evidence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Liz final4 C-F19: Capture refuses every shell entry point before running
// anything.
func TestCaptureRefusesShell(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	link := filepath.Join(dir, "innocent")
	if err := os.Symlink(sh, link); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "s.sh")
	os.WriteFile(script, []byte("touch "+marker+"\n"), 0o700)
	for _, tc := range []struct {
		name string
		exe  string
		argv []string
	}{
		{"bare-sh-c", "sh", []string{"-c", "touch " + marker}},
		{"absolute", sh, []string{"-c", "touch " + marker}},
		{"script-entry", "bash", []string{script}},
		{"symlinked-shell", link, []string{"-c", "touch " + marker}},
		{"env-wrapper", "env", []string{"sh", "-c", "touch " + marker}},
		{"versioned", "env", []string{"/bin/bash5.2", "-c", "touch " + marker}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Capture(context.Background(), Request{Executable: tc.exe, Argv: tc.argv, WorkingDir: dir, RepositoryRoot: dir})
			if err == nil || !strings.Contains(err.Error(), "no-shell policy") {
				t.Fatalf("err = %v, want no-shell refusal", err)
			}
			if _, err := os.Stat(marker); err == nil {
				t.Fatal("shell ran")
			}
		})
	}
}

func TestRejectShellAllowsPlainExecutables(t *testing.T) {
	for _, argv := range [][]string{{"go", "test", "./..."}, {"python3", "-c", "print(1)"}, {"/usr/bin/git", "diff", "--check"}, {"make", "shellcheck"}} {
		if r := RejectShell(argv[0], "/usr/bin/"+filepath.Base(argv[0]), argv[1:]); r != "" {
			t.Errorf("%v refused: %s", argv, r)
		}
	}
}

// Liz final5: a shell name that is plain data is not refused.
func TestRejectShellAllowsShellNameAsData(t *testing.T) {
	for _, argv := range [][]string{{"printf", "%s", "sh"}, {"grep", "-r", "bash", "docs"}, {"go", "test", "-run", "sh"}} {
		if r := RejectShell(argv[0], "/usr/bin/"+argv[0], argv[1:]); r != "" {
			t.Errorf("%v refused: %s", argv, r)
		}
	}
	for _, argv := range [][]string{{"python3", "x.py", "sh", "-c", "id"}, {"xargs", "sh"}, {"nice", "/bin/dash", "s.sh"}} {
		if RejectShell(argv[0], "/usr/bin/"+argv[0], argv[1:]) == "" {
			t.Errorf("%v accepted", argv)
		}
	}
}

// Liz final6 C-F22: Capture refuses a #!/bin/sh script run directly.
func TestCaptureRefusesShebangScript(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	p := filepath.Join(dir, "check")
	os.WriteFile(p, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o700)
	_, err := Capture(context.Background(), Request{Executable: p, WorkingDir: dir, RepositoryRoot: dir})
	if err == nil || !strings.Contains(err.Error(), "no-shell policy") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("script ran")
	}
	py := filepath.Join(dir, "ok")
	os.WriteFile(py, []byte("#!/usr/bin/env python3\nprint(1)\n"), 0o700)
	// An env-launched script is refused too (env is a wrapper); name the
	// interpreter directly, as python3 ok.py.
	if RejectShell(py, py, nil) == "" {
		t.Fatal("#!/usr/bin/env script admitted")
	}
	if r := RejectShell("python3", "/usr/bin/python3", []string{py}); r != "" {
		t.Fatalf("python3 script refused: %s", r)
	}
}
