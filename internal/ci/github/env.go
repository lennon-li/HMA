package github

import (
	"errors"
	"os"
)

// Environment captures the verified GitHub Actions runtime context.
type Environment struct {
	Repository string
	SHA        string
	Ref        string
	RunID      string
	Actor      string
}

// Load securely reads and verifies the deterministic GitHub Actions environment.
// It fails fast if not running inside GitHub Actions or if required variables are missing,
// without making any remote HTTP calls.
func Load() (Environment, error) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return Environment{}, errors.New("not running in GitHub Actions (GITHUB_ACTIONS != true)")
	}

	env := Environment{
		Repository: os.Getenv("GITHUB_REPOSITORY"),
		SHA:        os.Getenv("GITHUB_SHA"),
		Ref:        os.Getenv("GITHUB_REF"),
		RunID:      os.Getenv("GITHUB_RUN_ID"),
		Actor:      os.Getenv("GITHUB_ACTOR"),
	}

	if env.Repository == "" {
		return Environment{}, errors.New("missing GITHUB_REPOSITORY")
	}
	if env.SHA == "" {
		return Environment{}, errors.New("missing GITHUB_SHA")
	}
	if env.Ref == "" {
		return Environment{}, errors.New("missing GITHUB_REF")
	}
	if env.RunID == "" {
		return Environment{}, errors.New("missing GITHUB_RUN_ID")
	}
	if env.Actor == "" {
		return Environment{}, errors.New("missing GITHUB_ACTOR")
	}

	return env, nil
}
