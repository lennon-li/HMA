package main

// Regression tests ported from the repro tests of independent audit hma-gate-d-audit-20261023-d5c4m
// (manifest paths, submodules, type and mode changes, untracked sets, binary lines, no-contract packets).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/repostate"
)

func auditGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=A", "-c", "user.email=a@example.invalid", "-c", "protocol.file.allow=always"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func auditWrite(t *testing.T, p, body string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o700)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A1: non-ASCII path, committed case.
func TestAuditManifestNonASCIIPathCommitted(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "ü.txt"), "one\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "ü.txt"), "two\n")
	auditGit(t, repo, "commit", "-qam", "change")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	t.Logf("committed non-ASCII: files=%+v err=%v", files, err)
	if err != nil {
		t.Errorf("A1 committed manifest refuses a non-ASCII path: %v", err)
	} else if len(files) != 1 || files[0].Path != "ü.txt" {
		t.Errorf("A1 path rendered as %+v", files)
	}
}

// A1b: non-ASCII tracked path, dirty case.
func TestAuditManifestNonASCIIPathDirty(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "ü.txt"), "one\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "ü.txt"), "two\n")
	files, err := buildChangedFileManifest(context.Background(), repo, base, true)
	t.Logf("dirty non-ASCII: files=%+v err=%v", files, err)
	if err != nil {
		t.Errorf("A1b dirty manifest refuses a non-ASCII path: %v", err)
	}
}

// A2: submodule gitlink change, committed case.
func TestAuditManifestSubmoduleCommitted(t *testing.T) {
	sub := gitRepo(t)
	auditWrite(t, filepath.Join(sub, "x"), "1\n")
	auditGit(t, sub, "add", ".")
	auditGit(t, sub, "commit", "-qm", "s1")
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "a"), "1\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base0")
	auditGit(t, repo, "submodule", "add", "-q", sub, "sub")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "sub", "x"), "2\n")
	auditGit(t, filepath.Join(repo, "sub"), "commit", "-qam", "s2")
	auditGit(t, repo, "commit", "-qam", "bump")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	t.Logf("submodule bump: files=%+v err=%v", files, err)
	if err != nil {
		t.Errorf("A2 committed manifest refuses a submodule bump: %v", err)
	} else {
		want := auditGit(t, repo, "-C", "sub", "rev-parse", "HEAD")
		if len(files) != 1 || files[0].Status != "submodule" || files[0].SHA256 != want || files[0].OldMode != "160000" || files[0].NewMode != "160000" {
			t.Errorf("A2 submodule entry = %+v, want commit %s", files, want)
		}
	}
}

// A3: type change regular file -> symlink, committed. Status and digest.
func TestAuditManifestTypeChangeCommitted(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "f"), "body\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	os.Remove(filepath.Join(repo, "f"))
	os.Symlink("target", filepath.Join(repo, "f"))
	auditGit(t, repo, "add", "-A")
	auditGit(t, repo, "commit", "-qm", "tc")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	t.Logf("type change: files=%+v err=%v", files, err)
	sum := sha256.Sum256([]byte("target"))
	if err != nil || len(files) != 1 || files[0].SHA256 != hex.EncodeToString(sum[:]) || files[0].Status != "type_changed" || files[0].OldMode != "100644" || files[0].NewMode != "120000" {
		t.Errorf("A3 type change entry: %+v %v", files, err)
	}
}

// A4: textconv / filter attributes must not alter committed digest.
func TestAuditManifestCommittedDigestIgnoresFilters(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, ".gitattributes"), "*.txt diff=up filter=up\n")
	auditGit(t, repo, "config", "diff.up.textconv", "tr a-z A-Z <")
	auditGit(t, repo, "config", "filter.up.smudge", "tr a-z A-Z")
	auditGit(t, repo, "config", "filter.up.clean", "cat")
	auditWrite(t, filepath.Join(repo, "a.txt"), "one\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "a.txt"), "two\n")
	auditGit(t, repo, "commit", "-qam", "c")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	blob := auditGit(t, repo, "rev-parse", "HEAD:a.txt")
	raw, _ := exec.Command("git", "-C", repo, "cat-file", "blob", blob).Output()
	sum := sha256.Sum256(raw)
	for _, f := range files {
		if f.Path == "a.txt" && f.SHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("A4 committed digest altered by filter/textconv: %s want %s", f.SHA256, hex.EncodeToString(sum[:]))
		}
	}
	t.Logf("filters: %+v", files)
}

// A5: numstat line counts vs git for binary and CRLF; committed.
func TestAuditManifestLineCountsMatchGit(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "bin"), "a\x00b\nc\n")
	auditWrite(t, filepath.Join(repo, "t"), "x\r\ny\r\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "bin"), "a\x00b\nd\ne\n")
	auditWrite(t, filepath.Join(repo, "t"), "x\r\nz\r\n")
	auditGit(t, repo, "commit", "-qam", "c")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	ns := auditGit(t, repo, "diff", "--numstat", "--text", base+"..HEAD")
	t.Logf("manifest=%+v\ngit numstat --text:\n%s", files, ns)
}

// A6: untracked nested git repository (not a submodule), dirty case.
func TestAuditManifestNestedRepoDirty(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "a"), "1\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	nested := filepath.Join(repo, "vendor", "dep")
	os.MkdirAll(nested, 0o700)
	auditGit(t, nested, "init", "-q")
	auditWrite(t, filepath.Join(nested, "x"), "1\n")
	auditGit(t, nested, "add", ".")
	auditGit(t, nested, "commit", "-qm", "n")
	// The snapshot refuses a target with an untracked nested repository, so no
	// packet can be built for it; the observation refuses too instead of
	// listing the repository's internals or silently skipping it, and the
	// manifest and the budget measurement agree on that.
	if _, err := repostate.Snapshot(context.Background(), repo, base, true); err == nil {
		t.Fatal("the snapshot accepted an untracked nested repository")
	}
	if files, err := buildChangedFileManifest(context.Background(), repo, base, true); err == nil {
		t.Errorf("manifest accepted a nested repository: %+v", files)
	}
	if _, _, merr := measureChange(context.Background(), repo, base, true); merr == nil {
		t.Error("measureChange accepted a nested repository")
	}
}

// A7: file ignored by .git/info/exclude only: manifest vs measureChange.
func TestAuditManifestVsMeasureInfoExclude(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "a"), "1\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, ".git", "info", "exclude"), "secret.log\n")
	auditWrite(t, filepath.Join(repo, "secret.log"), "1\n2\n3\n")
	auditWrite(t, filepath.Join(repo, "new.txt"), "n\n")
	files, err := buildChangedFileManifest(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	n, lines, err := measureChange(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	st, _ := exec.Command("git", "-C", repo, "status", "--porcelain", "-uall").Output()
	t.Logf("manifest=%v measure files=%d lines=%d status=%q", paths, n, lines, st)
	if n != len(files) {
		t.Errorf("A7 measureChange counts %d files, manifest lists %d", n, len(files))
	}
	got := map[string]bool{}
	for _, path := range paths {
		got[path] = true
	}
	if !got["secret.log"] || !got["new.txt"] {
		t.Errorf("A7 untracked set omitted file outside committed ignores: %v", paths)
	}
}

// A8: many ignored files in dirty mode: one git process per file.
func TestAuditManifestIgnoredDirCost(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, ".gitignore"), "build/\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	for i := 0; i < 1000; i++ {
		auditWrite(t, filepath.Join(repo, "build", "f"+strings.Repeat("x", i%7)+string(rune('a'+i%26))+"-"+itoa(i)), "x\n")
	}
	start := time.Now()
	files, err := buildChangedFileManifest(context.Background(), repo, base, true)
	t.Logf("1000 ignored files: %v, manifest=%d err=%v", time.Since(start), len(files), err)
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

// A9: no-contract packet keys vs HEAD packet keys.
func TestAuditNoContractPacketKeys(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	b, err := os.ReadFile(f.packetPath)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)Bound HMA state[^\n]*\n(\{.*?\n\})\n\nTask:`).FindSubmatch(b)
	if m == nil {
		t.Fatalf("no state: %s", b)
	}
	var raw map[string]json.RawMessage
	json.Unmarshal(m[1], &raw)
	keys := []string{}
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("no-contract packet keys: %v", keys)
	want := map[string]bool{"core_head_anchor": true, "accepted_plan_digest": true, "route_approval_digest": true, "criteria": true, "findings": true, "active_waivers": true}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("A9 no-contract packet has key absent at HEAD: %s = %s", k, raw[k])
		}
	}
	if len(keys) != len(want) {
		t.Errorf("A9 no-contract key set = %v, expected exactly %v", keys, want)
	}
	var state packetCoreState
	if err := json.Unmarshal(m[1], &state); err != nil {
		t.Fatal(err)
	}
	// As before D5c (and as the Gate C packet test pins): the plan digest the
	// unit's route approval carries, not the route approval's own digest.
	if state.AcceptedPlanDigest != f.planDigest {
		t.Errorf("no-contract accepted_plan_digest=%q, want the route approval's plan digest %q", state.AcceptedPlanDigest, f.planDigest)
	}
}

// A10: binary content lines vs budget measurement.
func TestAuditBinaryLinesMeasured(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "a"), "1\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "big.c"), "\x00\n"+strings.Repeat("int x;\n", 5000))
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "c")
	n, lines, err := measureChange(context.Background(), repo, base, false)
	files, merr := buildChangedFileManifest(context.Background(), repo, base, false)
	t.Logf("A10 measureChange files=%d lines=%d err=%v; manifest=%+v err=%v", n, lines, err, files, merr)
}

func TestAuditManifestQuotedPathsCommittedAndDirty(t *testing.T) {
	for _, committed := range []bool{false, true} {
		repo := gitRepo(t)
		names := []string{"ü.txt", `a"b`, "tab\tname", "new\nline"}
		for _, name := range names {
			auditWrite(t, filepath.Join(repo, name), "old\n")
		}
		auditGit(t, repo, "add", ".")
		auditGit(t, repo, "commit", "-qm", "base")
		base := auditGit(t, repo, "rev-parse", "HEAD")
		for _, name := range names {
			auditWrite(t, filepath.Join(repo, name), "new\n")
		}
		if committed {
			auditGit(t, repo, "add", ".")
			auditGit(t, repo, "commit", "-qm", "changed")
		}
		files, err := buildChangedFileManifest(context.Background(), repo, base, !committed)
		if err != nil {
			t.Fatalf("committed=%v: %v", committed, err)
		}
		got := map[string]bool{}
		for _, f := range files {
			got[f.Path] = true
		}
		for _, name := range names {
			if !got[name] {
				t.Errorf("committed=%v omitted exact path %q: %+v", committed, name, files)
			}
		}
		measuredFiles, _, measureErr := measureChange(context.Background(), repo, base, !committed)
		if measureErr != nil || measuredFiles != len(files) {
			t.Errorf("committed=%v measure files=%d manifest=%d err=%v", committed, measuredFiles, len(files), measureErr)
		}
	}
}

func TestAuditBinaryLineCountsManifestAndDeleted(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "base"), "base\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	body := []byte("\x00\n" + strings.Repeat("line\n", 5000))
	if err := os.WriteFile(filepath.Join(repo, "added.bin"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "binary")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !files[0].Binary || files[0].Added < 5001 {
		t.Fatalf("binary manifest count = %+v", files)
	}
	_, lines, err := measureChange(context.Background(), repo, base, false)
	if err != nil || lines < 5001 {
		t.Fatalf("binary measure lines=%d err=%v", lines, err)
	}
	base = auditGit(t, repo, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(repo, "added.bin")); err != nil {
		t.Fatal(err)
	}
	auditGit(t, repo, "add", "-A")
	auditGit(t, repo, "commit", "-qm", "delete binary")
	files, err = buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !files[0].Binary || files[0].Deleted < 5001 || files[0].Status != "deleted" {
		t.Fatalf("deleted binary manifest = %+v", files)
	}
	_, lines, err = measureChange(context.Background(), repo, base, false)
	if err != nil || lines < 5001 {
		t.Fatalf("deleted binary measure lines=%d err=%v", lines, err)
	}
}

func TestAuditManifestModeOnlyChangeHasModesAndZeroLines(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "executable"), "#!/bin/sh\nexit 0\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	if err := os.Chmod(filepath.Join(repo, "executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	auditGit(t, repo, "add", "executable")
	auditGit(t, repo, "commit", "-qm", "mode")
	files, err := buildChangedFileManifest(context.Background(), repo, base, false)
	if err != nil || len(files) != 1 {
		t.Fatalf("mode-only manifest=%+v err=%v", files, err)
	}
	f := files[0]
	if f.Added != 0 || f.Deleted != 0 || f.OldMode != "100644" || f.NewMode != "100755" || f.Status != "modified" {
		t.Errorf("mode-only entry = %+v", f)
	}
}

func TestAuditManifestGitProcessCountDoesNotScalePerPath(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, ".gitignore"), "ignored/\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	for i := 0; i < 1000; i++ {
		auditWrite(t, filepath.Join(repo, "ignored", "file-"+itoa(i)), "x\n")
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "git.log")
	script := "#!/bin/sh\nprintf 'x\\n' >> \"$HMA_TEST_GIT_LOG\"\nexec " + realGit + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	oldPath, hadPath := os.LookupEnv("PATH")
	oldLog, hadLog := os.LookupEnv("HMA_TEST_GIT_LOG")
	if err := os.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("HMA_TEST_GIT_LOG", logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadPath {
			_ = os.Setenv("PATH", oldPath)
		} else {
			_ = os.Unsetenv("PATH")
		}
		if hadLog {
			_ = os.Setenv("HMA_TEST_GIT_LOG", oldLog)
		} else {
			_ = os.Unsetenv("HMA_TEST_GIT_LOG")
		}
	})
	files, err := buildChangedFileManifest(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("ignored manifest entries=%d", len(files))
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if calls := strings.Count(string(log), "x\n"); calls > 20 {
		t.Fatalf("manifest launched %d git processes for 1000 ignored paths", calls)
	}
}
