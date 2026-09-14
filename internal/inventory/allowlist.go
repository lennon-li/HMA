package inventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const AllowlistVersionV1 = "machine-truth-allowlist/v1"

// Allowlist is the host-local file that names every discovery command HMA may
// run. It is host configuration, never a portable repository artifact.
type Allowlist struct {
	Version               string    `json:"version"`
	FreshnessSeconds      int64     `json:"freshness_seconds"`
	CommandTimeoutSeconds int64     `json:"command_timeout_seconds,omitempty"`
	MaxOutputBytes        int       `json:"max_output_bytes,omitempty"`
	Environment           []string  `json:"environment,omitempty"`
	Commands              []Command `json:"commands"`
}

var ErrInvalidAllowlist = errors.New("invalid inventory allowlist")

// LoadAllowlist decodes exactly one allowlist document, rejecting unknown
// fields, and validates every command before anything could run.
func LoadAllowlist(r io.Reader) (Allowlist, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var allowlist Allowlist
	if err := dec.Decode(&allowlist); err != nil {
		return Allowlist{}, fmt.Errorf("%w: %v", ErrInvalidAllowlist, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return Allowlist{}, fmt.Errorf("%w: multiple documents", ErrInvalidAllowlist)
	}
	if err := allowlist.validate(); err != nil {
		return Allowlist{}, err
	}
	return allowlist, nil
}

func (a Allowlist) validate() error {
	if a.Version != AllowlistVersionV1 {
		return fmt.Errorf("%w: version %q, want %q", ErrInvalidAllowlist, a.Version, AllowlistVersionV1)
	}
	if a.FreshnessSeconds <= 0 || a.CommandTimeoutSeconds < 0 || a.MaxOutputBytes < 0 || len(a.Commands) == 0 {
		return fmt.Errorf("%w: freshness, timeout, output bound, or commands out of range", ErrInvalidAllowlist)
	}
	for _, entry := range a.Environment {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" || strings.ContainsAny(entry, "\x00\n\r") {
			return fmt.Errorf("%w: environment entry must be KEY=VALUE", ErrInvalidAllowlist)
		}
	}
	seen := make(map[string]struct{}, len(a.Commands))
	for _, command := range a.Commands {
		if err := validateCommand(command); err != nil {
			return fmt.Errorf("%w: command %q: %v", ErrInvalidAllowlist, command.ID, err)
		}
		if _, exists := seen[command.ID]; exists {
			return fmt.Errorf("%w: duplicate command id %q", ErrInvalidAllowlist, command.ID)
		}
		seen[command.ID] = struct{}{}
	}
	return nil
}

// Request binds the allowlist to one capture time.
func (a Allowlist) Request(capturedAt time.Time) Request {
	return Request{
		Commands:       append([]Command(nil), a.Commands...),
		CapturedAt:     capturedAt,
		Freshness:      time.Duration(a.FreshnessSeconds) * time.Second,
		Environment:    append([]string(nil), a.Environment...),
		MaxOutputBytes: a.MaxOutputBytes,
		CommandTimeout: time.Duration(a.CommandTimeoutSeconds) * time.Second,
	}
}
