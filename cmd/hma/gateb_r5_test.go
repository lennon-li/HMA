package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/lennon-li/HMA/internal/repostate"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Each negative requires refusal, rather than merely asserting that Git hides it.
func r5AttrRepo(t *testing.T, attr string) (root, base, head, diff string) {
	t.Helper()
	root, base, _, _ = ciRepo(t)
	if e := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("a "+attr+"\n"), 0600); e != nil {
		t.Fatal(e)
	}
	bGit(t, root, "add", ".gitattributes")
	bGit(t, root, "commit", "-qm", "tracked attributes")
	s, e := repostate.Snapshot(context.Background(), root, base, false)
	if e != nil {
		t.Fatal(e)
	}
	return root, base, s.Head, s.Diff
}
func TestGateBR5CINormalizedTrackedBytes(t *testing.T) {
	for _, attr := range []string{"text eol=lf", "text eol=crlf", "text=auto"} {
		t.Run(attr, func(t *testing.T) {
			root, base, head, diff := r5AttrRepo(t, attr)
			if e := bCI(t, root, base, head, diff, "/bin/cat", "a"); e != nil {
				t.Fatalf("unchanged control: %v", e)
			}
			if e := os.WriteFile(filepath.Join(root, "a"), []byte("two\r\n"), 0600); e != nil {
				t.Fatal(e)
			}
			s, e := repostate.Snapshot(context.Background(), root, base, false)
			t.Logf("attr=%q status=%q live=%q snapshot=%+v err=%v", attr, bGit(t, root, "status", "--porcelain"), "two\r\n", s, e)
			if e := bCI(t, root, base, head, diff, "/bin/cat", "a"); e == nil {
				t.Error("CI appended revision evidence for byte-distinct normalized tracked content")
			}
		})
	}
}
func TestGateBR5CINormalizationMutation(t *testing.T) {
	root, base, head, diff := r5AttrRepo(t, "text eol=lf")
	if e := bCI(t, root, base, head, diff, "/bin/sh", "-c", "printf 'two\\r\\n' > a"); e == nil {
		t.Error("CI appended revision evidence after command changed LF bytes to CRLF")
	}
}
func r5MixedRepo(t *testing.T, attr string) (root, base, head, diff string) {
	t.Helper()
	root, base, _, _ = r5AttrRepo(t, attr)
	if e := os.WriteFile(filepath.Join(root, "a"), []byte("a\r\nb\n"), 0600); e != nil {
		t.Fatal(e)
	}
	bGit(t, root, "add", "a")
	bGit(t, root, "commit", "-qm", "mixed EOL file")
	s, e := repostate.Snapshot(context.Background(), root, base, false)
	if e != nil {
		t.Fatal(e)
	}
	return root, base, s.Head, s.Diff
}
func r5SubmoduleRepo(t *testing.T, policy string) (root, base, head, diff string) {
	t.Helper()
	root, base, _, _ = ciRepo(t)
	child, _, _, _ := ciRepo(t)
	bGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", "-q", child, "sub")
	bGit(t, root, "commit", "-qam", "add submodule")
	if policy == "module" {
		bGit(t, root, "config", "submodule.sub.ignore", "all")
	} else if policy == "tracked" {
		bGit(t, root, "config", "-f", ".gitmodules", "submodule.sub.ignore", "all")
		bGit(t, root, "commit", "-qam", "tracked ignore")
	} else {
		bGit(t, root, "config", "diff.ignoreSubmodules", "all")
	}
	s, e := repostate.Snapshot(context.Background(), root, base, false)
	if e != nil {
		t.Fatal(e)
	}
	return root, base, s.Head, s.Diff
}
func TestGateBR5SizeMtimeRewrite(t *testing.T) {
	root, base, _, _ := ciRepo(t)
	p := filepath.Join(root, "a")
	// Seed a trusted Git stat cache, then preserve both the size and mtime.
	bGit(t, root, "update-index", "--refresh")
	s0, e := repostate.Snapshot(context.Background(), root, base, false)
	if e != nil {
		t.Fatal(e)
	}
	st, e := os.Stat(p)
	if e != nil {
		t.Fatal(e)
	}
	time.Sleep(20 * time.Millisecond)
	if e := os.WriteFile(p, []byte("six\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Chtimes(p, st.ModTime(), st.ModTime()); e != nil {
		t.Fatal(e)
	}
	s1, e := repostate.Snapshot(context.Background(), root, base, false)
	t.Logf("same size and mtime: before=%+v after=%+v err=%v", s0, s1, e)
	if e == nil {
		t.Error("same-size same-mtime write hidden")
	}
}
func TestGateBR5DiffDriverIsolation(t *testing.T) {
	root, base, _, _ := ciRepo(t)
	if e := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("a diff=hide\n"), 0600); e != nil {
		t.Fatal(e)
	}
	bGit(t, root, "add", ".gitattributes")
	bGit(t, root, "commit", "-qm", "attributes")
	a, e := repostate.Snapshot(context.Background(), root, base, false)
	if e != nil {
		t.Fatal(e)
	}
	for _, kv := range [][2]string{{"diff.hide.command", "/bin/true"}, {"diff.hide.textconv", "/bin/true"}, {"diff.external", "/bin/true"}} {
		bGit(t, root, "config", kv[0], kv[1])
	}
	b, e := repostate.Snapshot(context.Background(), root, base, false)
	if e != nil {
		t.Fatal(e)
	}
	if a != b {
		t.Fatalf("external driver changed snapshot: %+v %+v", a, b)
	}
	t.Logf("external driver overrides preserved diff=%s", a.Diff)
}
func r5RawDigest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
