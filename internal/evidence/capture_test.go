package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("HMA_HELPER") != "1" {
		return
	}
	os.Stdout.WriteString(strings.Join(os.Args[1:], "\n"))
	os.Exit(0)
}

func TestCapturePassesArgvVerbatim(t *testing.T) {
	root := t.TempDir()
	args := []string{"-test.run=TestHelperProcess", "--", "a b", "$(no)", "x;y"}
	r, err := Capture(context.Background(), Request{Executable: os.Args[0], Argv: args, WorkingDir: root, RepositoryRoot: root, Environment: []string{"HMA_HELPER=1"}, MaxOutputBytes: 4096, RepositoryIdentity: "repo", BaseRevision: "base", HeadRevision: "head", DiffDigest: "diff", CapturingActor: "host"})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join(args, "\n")
	if string(r.Stdout) != want {
		t.Fatalf("argv changed: %q want %q", r.Stdout, want)
	}
	if !reflect.DeepEqual(r.Evidence.Argv, args) {
		t.Fatalf("recorded argv = %#v", r.Evidence.Argv)
	}
}

func TestCaptureRejectsInvalidBoundary(t *testing.T) {
	root := t.TempDir()
	base := Request{Executable: os.Args[0], WorkingDir: root, RepositoryRoot: root, CapturingActor: "host"}
	for name, mutate := range map[string]func(*Request){
		"empty executable":    func(r *Request) { r.Executable = "" },
		"empty working dir":   func(r *Request) { r.WorkingDir = "" },
		"metachar executable": func(r *Request) { r.Executable = "bad;name" },
		"outside root":        func(r *Request) { r.WorkingDir = filepath.Dir(root) },
	} {
		t.Run(name, func(t *testing.T) {
			r := base
			mutate(&r)
			if _, err := Capture(context.Background(), r); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestOutputDigestIsUnambiguous(t *testing.T) {
	// ("ab", "c") and ("a", "bc") must not share a digest.
	first := digestOf(t, "ab", "c")
	second := digestOf(t, "a", "bc")
	if first == second {
		t.Fatal("stdout/stderr split is not framed: digests collide")
	}
	if first != digestOf(t, "ab", "c") {
		t.Fatal("digest is not deterministic")
	}
}

func digestOf(t *testing.T, stdout, stderr string) string {
	t.Helper()
	h := sha256.New()
	writeDigestSection(h, "stdout", []byte(stdout))
	writeDigestSection(h, "stderr", []byte(stderr))
	return hex.EncodeToString(h.Sum(nil))
}

func TestCaptureRecordsResolvedAbsoluteExecutable(t *testing.T) {
	root := t.TempDir()
	r, err := Capture(context.Background(), Request{
		Executable: "true", WorkingDir: root, RepositoryRoot: root,
		MaxOutputBytes: 4096, RepositoryIdentity: "repo", BaseRevision: "base",
		HeadRevision: "head", CapturingActor: "host",
	})
	if err != nil {
		t.Skipf("no 'true' on PATH: %v", err)
	}
	if !filepath.IsAbs(r.Evidence.Executable) {
		t.Fatalf("recorded executable %q is not absolute", r.Evidence.Executable)
	}
}

func TestCaptureEnforcesAllowlist(t *testing.T) {
	root := t.TempDir()
	base := Request{
		Executable: os.Args[0], Argv: []string{"-test.run=TestHelperProcess"},
		WorkingDir: root, RepositoryRoot: root, Environment: []string{"HMA_HELPER=1"},
		MaxOutputBytes: 4096, RepositoryIdentity: "repo", BaseRevision: "base",
		HeadRevision: "head", CapturingActor: "host",
	}

	denied := base
	denied.AllowedExecutables = []string{"/nonexistent/binary"}
	if _, err := Capture(context.Background(), denied); err == nil {
		t.Fatal("non-allowlisted executable accepted")
	}

	permitted := base
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	permitted.AllowedExecutables = []string{self}
	if _, err := Capture(context.Background(), permitted); err != nil {
		t.Fatalf("allowlisted executable rejected: %v", err)
	}
}

func TestCaptureRejectsRelativeExecutablePath(t *testing.T) {
	root := t.TempDir()
	_, err := Capture(context.Background(), Request{
		Executable: "./helper", WorkingDir: root, RepositoryRoot: root,
		RepositoryIdentity: "repo", BaseRevision: "base", HeadRevision: "head",
		CapturingActor: "host",
	})
	if err == nil {
		t.Fatal("relative executable path accepted")
	}
}
