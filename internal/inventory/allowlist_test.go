package inventory

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const validAllowlist = `{
  "version": "machine-truth-allowlist/v1",
  "freshness_seconds": 600,
  "command_timeout_seconds": 5,
  "max_output_bytes": 4096,
  "environment": ["LANG=C"],
  "commands": [{"id": "printf", "kind": "executable_presence", "subject": "printf", "executable": "/usr/bin/printf"}]
}`

func TestLoadAllowlistBuildsRequest(t *testing.T) {
	allowlist, err := LoadAllowlist(strings.NewReader(validAllowlist))
	if err != nil {
		t.Fatalf("LoadAllowlist() error = %v", err)
	}
	now := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	request := allowlist.Request(now)
	if request.Freshness != 10*time.Minute || request.CommandTimeout != 5*time.Second || request.MaxOutputBytes != 4096 ||
		!request.CapturedAt.Equal(now) || len(request.Commands) != 1 || len(request.Environment) != 1 {
		t.Fatalf("request = %+v", request)
	}
}

func TestLoadAllowlistRejectsInvalidDocuments(t *testing.T) {
	for name, doc := range map[string]string{
		"unknown field":      strings.Replace(validAllowlist, `"freshness_seconds"`, `"shell": "sh", "freshness_seconds"`, 1),
		"wrong version":      strings.Replace(validAllowlist, "allowlist/v1", "allowlist/v0", 1),
		"no freshness":       strings.Replace(validAllowlist, `"freshness_seconds": 600`, `"freshness_seconds": 0`, 1),
		"negative timeout":   strings.Replace(validAllowlist, `"command_timeout_seconds": 5`, `"command_timeout_seconds": -1`, 1),
		"no commands":        `{"version": "machine-truth-allowlist/v1", "freshness_seconds": 600, "commands": []}`,
		"bare executable":    strings.Replace(validAllowlist, `"/usr/bin/printf"`, `"printf"`, 1),
		"env without equals": strings.Replace(validAllowlist, `"LANG=C"`, `"NOEQUALS"`, 1),
		"env empty key":      strings.Replace(validAllowlist, `"LANG=C"`, `"=C"`, 1),
		"multiple docs":      validAllowlist + validAllowlist,
		"duplicate command":  strings.Replace(validAllowlist, `}]`, `}, {"id": "printf", "kind": "executable_presence", "subject": "again", "executable": "/usr/bin/printf"}]`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadAllowlist(strings.NewReader(doc)); !errors.Is(err, ErrInvalidAllowlist) {
				t.Fatalf("error = %v, want ErrInvalidAllowlist", err)
			}
		})
	}
}
