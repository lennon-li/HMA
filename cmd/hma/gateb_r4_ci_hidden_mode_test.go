package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGateBR4CIHiddenOwnerExecute(t *testing.T) {
	root, base, head, diff := ciRepo(t)
	bGit(t, root, "config", "core.filemode", "false")
	if e := os.Chmod(filepath.Join(root, "a"), 0700); e != nil {
		t.Fatal(e)
	}
	t.Logf("status=%q actual_mode_diff=%s", bGit(t, root, "status", "--porcelain"), bGit(t, root, "-c", "core.filemode=true", "diff", "--summary"))
	if e := bCI(t, root, base, head, diff, "/usr/bin/stat", "-c", "%a", "a"); e == nil {
		t.Fatal("CI appended clean revision-labelled evidence for unapproved owner-execute mode")
	}
}

func TestGateBR4CIHiddenOwnerExecuteMutation(t *testing.T) {
	root, base, head, diff := ciRepo(t)
	bGit(t, root, "config", "core.filemode", "false")
	if e := bCI(t, root, base, head, diff, "/usr/bin/chmod", "700", "a"); e == nil {
		t.Fatalf("CI appended clean evidence after command changed tracked owner-execute; hidden_status=%q real_mode_diff=%s", bGit(t, root, "status", "--porcelain"), bGit(t, root, "-c", "core.filemode=true", "diff", "--summary"))
	}
}
