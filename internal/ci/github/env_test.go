package github

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clear env
	for _, k := range []string{"GITHUB_ACTIONS", "GITHUB_REPOSITORY", "GITHUB_SHA", "GITHUB_REF", "GITHUB_RUN_ID", "GITHUB_ACTOR"} {
		os.Unsetenv(k)
	}

	_, err := Load()
	if err == nil || err.Error() != "not running in GitHub Actions (GITHUB_ACTIONS != true)" {
		t.Errorf("expected not running error, got %v", err)
	}

	os.Setenv("GITHUB_ACTIONS", "true")
	_, err = Load()
	if err == nil || err.Error() != "missing GITHUB_REPOSITORY" {
		t.Errorf("expected missing GITHUB_REPOSITORY error, got %v", err)
	}

	os.Setenv("GITHUB_REPOSITORY", "google/uuid")
	os.Setenv("GITHUB_SHA", "0f11ee6918f41a04c201eceeadf612a377bc7fbc")
	os.Setenv("GITHUB_REF", "refs/heads/master")
	os.Setenv("GITHUB_RUN_ID", "12345")
	os.Setenv("GITHUB_ACTOR", "Lennon")

	env, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env.Repository != "google/uuid" || env.SHA != "0f11ee6918f41a04c201eceeadf612a377bc7fbc" {
		t.Errorf("unexpected env values: %+v", env)
	}
}
