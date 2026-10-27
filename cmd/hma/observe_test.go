package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestManifestIsViewOfSharedObservation(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "base"), "base\n")
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	for _, name := range []string{":(top)x", "a[b]*", "非ascii", "tab\tname", "line\nbreak"} {
		auditWrite(t, filepath.Join(repo, name), "value\n")
	}
	auditGit(t, repo, "add", ".")
	if err := os.Symlink("base", filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	observed, err := observeTarget(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := buildChangedFileManifest(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	want := make([]packetFile, 0, len(observed))
	for _, e := range observed {
		want = append(want, packetFile{Path: e.Path, Status: e.Status, Added: e.Added, Deleted: e.Deleted, SHA256: e.SHA256, OldMode: e.OldMode, NewMode: e.NewMode, Binary: e.Binary, Staged: e.Staged, LinesTruncated: e.LinesTruncated})
	}
	if !reflect.DeepEqual(manifest, want) {
		t.Fatalf("manifest is not the observation view\nmanifest=%+v\nwant=%+v", manifest, want)
	}
	staged := false
	for _, e := range manifest {
		if e.Staged {
			staged = true
		}
	}
	if !staged {
		t.Fatal("staged index change was not marked staged")
	}
	for _, name := range []string{":(top)x", "a[b]*", "非ascii", "tab\tname", "line\nbreak", "link"} {
		found := false
		for _, e := range manifest {
			if e.Path == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("literal path %q missing", name)
		}
	}
}

func TestManifestSharedObservationFullTargetKinds(t *testing.T) {
	repo := gitRepo(t)
	for name, body := range map[string]string{
		"modified": "old\n", "staged-delete": "delete\n", "rename-old": "rename\n",
		"unstaged-delete": "gone\n", "staged-only": "before\n", "binary-base": "plain\n",
	} {
		auditWrite(t, filepath.Join(repo, name), body)
	}
	auditGit(t, repo, "add", ".")
	auditGit(t, repo, "commit", "-qm", "base")
	base := auditGit(t, repo, "rev-parse", "HEAD")
	auditWrite(t, filepath.Join(repo, "modified"), "new\n")
	auditGit(t, repo, "rm", "staged-delete")
	auditGit(t, repo, "mv", "rename-old", "rename-new")
	auditWrite(t, filepath.Join(repo, "staged-only"), "staged\n")
	auditGit(t, repo, "add", "staged-only")
	if err := os.Remove(filepath.Join(repo, "unstaged-delete")); err != nil {
		t.Fatal(err)
	}
	auditWrite(t, filepath.Join(repo, "added"), "add\n")
	auditWrite(t, filepath.Join(repo, "binary"), "\x00one\ntwo\n")
	for _, name := range []string{":(top)x", "a[b]*", "非ascii", "tab\tname", "line\nbreak"} {
		auditWrite(t, filepath.Join(repo, name), "literal\n")
	}
	if err := os.Symlink("modified", filepath.Join(repo, "symlink")); err != nil {
		t.Fatal(err)
	}
	observed, err := observeTarget(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := buildChangedFileManifest(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	want := make([]packetFile, 0, len(observed))
	for _, e := range observed {
		want = append(want, packetFile{Path: e.Path, Status: e.Status, Added: e.Added, Deleted: e.Deleted, SHA256: e.SHA256, OldMode: e.OldMode, NewMode: e.NewMode, Binary: e.Binary, Staged: e.Staged, LinesTruncated: e.LinesTruncated})
	}
	if !reflect.DeepEqual(manifest, want) {
		t.Fatal("manifest differs from observation field mapping")
	}
	byPath := map[string]packetFile{}
	for _, f := range manifest {
		byPath[f.Path] = f
	}
	for name, status := range map[string]string{"added": "added", "modified": "modified", "staged-delete": "deleted", "rename-old": "deleted", "rename-new": "added", "unstaged-delete": "deleted", "symlink": "added"} {
		if byPath[name].Status != status {
			t.Errorf("%q status=%q want=%q", name, byPath[name].Status, status)
		}
	}
	if !byPath["staged-delete"].Staged || !byPath["rename-old"].Staged || !byPath["rename-new"].Staged || !byPath["staged-only"].Staged {
		t.Error("staged index changes were not all marked staged")
	}
	if !byPath["binary"].Binary {
		t.Error("binary target was not marked binary")
	}
}
