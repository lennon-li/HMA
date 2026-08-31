package evidence

import (
	"context"
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
