package main

import (
	"context"
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
)

// An observation held to a capture refuses bytes that differ from the capture,
// whatever a later check would find once the writer restores the file.
func TestObserveCapturedRefusesBytesOutsideTheCapture(t *testing.T) {
	repo := gitRepo(t)
	auditWrite(t, filepath.Join(repo, "f"), "one\n")
	base := d9Commit(t, repo, "base")
	auditWrite(t, filepath.Join(repo, "f"), "two\n")
	auditWrite(t, filepath.Join(repo, "u"), "untracked\n")
	c, err := captureTarget(context.Background(), repo, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := observeCaptured(context.Background(), repo, base, true, 0, &c); err != nil {
		t.Fatalf("unchanged target: %v", err)
	}
	for _, name := range []string{"f", "u"} {
		original := map[string]string{"f": "two\n", "u": "untracked\n"}[name]
		auditWrite(t, filepath.Join(repo, name), "transient\n")
		_, err := observeCaptured(context.Background(), repo, base, true, 0, &c)
		if err == nil || err.Error() != hoststate.ReasonReviewTargetChanged {
			t.Errorf("%s changed after capture: error = %v, want %s", name, err, hoststate.ReasonReviewTargetChanged)
		}
		auditWrite(t, filepath.Join(repo, name), original)
	}
}
