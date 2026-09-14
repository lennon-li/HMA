// Package inventory captures credential-free, host-local machine truth from
// explicitly allowlisted discovery commands. Its output is availability
// evidence only; it does not establish route eligibility or trust.
package inventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const VersionV1 = "machine-truth/v1"

const (
	defaultMaxOutputBytes = 1 << 20
	defaultCommandTimeout = 10 * time.Second
	// maxFieldBytes and maxModels bound what parsed output may be retained;
	// anything larger is recorded as unparsed rather than truncated, so a
	// retained field is never a partial reading.
	maxFieldBytes = 256
	maxModels     = 1024
)

type DiscoveryKind string

const (
	KindExecutablePresence DiscoveryKind = "executable_presence"
	KindVersion            DiscoveryKind = "version"
	KindModelAvailability  DiscoveryKind = "model_availability"
)

// Status says what happened to one discovery command. Only StatusCompleted
// carries parsed metadata; every other status is unavailable evidence.
type Status string

const (
	StatusAbsent        Status = "absent"
	StatusPresent       Status = "present"
	StatusCompleted     Status = "completed"
	StatusExitedNonzero Status = "exited_nonzero"
	StatusTimedOut      Status = "timed_out"
	StatusStartFailed   Status = "start_failed"
)

// Command is one host-allowlisted, exact executable invocation. Executable
// must be a clean absolute path so the record names what ran. Executable and
// Argv are host-local metadata and must not be copied into portable
// repository artifacts.
type Command struct {
	ID         string        `json:"id"`
	Kind       DiscoveryKind `json:"kind"`
	Subject    string        `json:"subject"`
	Executable string        `json:"executable"`
	Argv       []string      `json:"argv,omitempty"`
}

type Entry struct {
	ID              string        `json:"id"`
	Kind            DiscoveryKind `json:"kind"`
	Subject         string        `json:"subject"`
	Executable      string        `json:"executable"`
	Argv            []string      `json:"argv,omitempty"`
	Status          Status        `json:"status"`
	ExitCode        int           `json:"exit_code"`
	Version         string        `json:"version,omitempty"`
	Models          []string      `json:"models,omitempty"`
	OutputDigest    string        `json:"output_digest,omitempty"`
	OutputTruncated bool          `json:"output_truncated,omitempty"`
	OutputUnparsed  bool          `json:"output_unparsed,omitempty"`
}

// Inventory is host-local machine truth. Digest covers the version, capture
// times, and normalized entries, excluding Digest itself.
type Inventory struct {
	Version    string  `json:"version"`
	CapturedAt string  `json:"captured_at"`
	FreshUntil string  `json:"fresh_until"`
	Entries    []Entry `json:"entries"`
	Digest     string  `json:"digest"`
}

type Request struct {
	Commands   []Command
	CapturedAt time.Time
	Freshness  time.Duration
	// Environment is the complete environment given to every discovery
	// command. Nil means an empty environment: nothing is inherited.
	Environment    []string
	MaxOutputBytes int
	// CommandTimeout bounds each command; zero means defaultCommandTimeout.
	CommandTimeout time.Duration
}

var (
	ErrInvalidRequest = errors.New("invalid inventory request")
	ErrInvalidCommand = errors.New("invalid inventory command")
	ErrDigestFailure  = errors.New("inventory digest failure")
)

func validKind(kind DiscoveryKind) bool {
	switch kind {
	case KindExecutablePresence, KindVersion, KindModelAvailability:
		return true
	default:
		return false
	}
}

func validateCommand(command Command) error {
	if command.ID == "" || command.Subject == "" || command.Executable == "" || !validKind(command.Kind) {
		return ErrInvalidCommand
	}
	if !filepath.IsAbs(command.Executable) || filepath.Clean(command.Executable) != command.Executable {
		return ErrInvalidCommand
	}
	if strings.ContainsAny(command.Executable, " \t|&;<>()$`\\\n\r") {
		return ErrInvalidCommand
	}
	for _, arg := range command.Argv {
		if strings.ContainsAny(arg, "\x00\n\r") {
			return ErrInvalidCommand
		}
	}
	return nil
}

// writeDigestSection length-frames one captured stream so that the digest of
// (stdout, stderr) is unambiguous. It mirrors the evidence package's framing
// under a distinct domain tag.
func writeDigestSection(h hash.Hash, name string, data []byte) {
	h.Write([]byte("hma-inventory-v1\x00" + name + "\x00"))
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	h.Write(length[:])
	h.Write(data)
}

func digestOutput(stdout, stderr []byte) string {
	h := sha256.New()
	writeDigestSection(h, "stdout", stdout)
	writeDigestSection(h, "stderr", stderr)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func retainableField(value string) bool {
	if len(value) > maxFieldBytes {
		return false
	}
	for _, r := range value {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func parseModels(stdout []byte) ([]string, bool) {
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(stdout), "\n") {
		model := strings.TrimSpace(line)
		if model == "" {
			continue
		}
		if !retainableField(model) {
			return nil, false
		}
		seen[model] = struct{}{}
		if len(seen) > maxModels {
			return nil, false
		}
	}
	if len(seen) == 0 {
		return nil, true
	}
	models := make([]string, 0, len(seen))
	for model := range seen {
		models = append(models, model)
	}
	sort.Strings(models)
	return models, true
}

func parseVersion(stdout []byte) (string, bool) {
	for _, line := range strings.Split(string(stdout), "\n") {
		if version := strings.TrimSpace(line); version != "" {
			if !retainableField(version) {
				return "", false
			}
			return version, true
		}
	}
	return "", true
}

func captureCommand(ctx context.Context, command Command, request Request) (Entry, error) {
	entry := Entry{ID: command.ID, Kind: command.Kind, Subject: command.Subject, Executable: command.Executable, Argv: append([]string(nil), command.Argv...), ExitCode: -1}
	// LookPath on an absolute path checks that exact file; PATH is not consulted.
	if _, err := exec.LookPath(command.Executable); err != nil {
		entry.Status = StatusAbsent
		return entry, nil
	}
	if command.Kind == KindExecutablePresence {
		entry.Status = StatusPresent
		return entry, nil
	}

	timeout := request.CommandTimeout
	if timeout <= 0 {
		timeout = defaultCommandTimeout
	}
	limit := request.MaxOutputBytes
	if limit <= 0 {
		limit = defaultMaxOutputBytes
	}
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, command.Executable, command.Argv...)
	// Discovery must not inherit HOME, tokens, provider credentials, PATH, or
	// other ambient configuration; only the host-supplied environment applies.
	cmd.Env = append([]string{}, request.Environment...)
	// A grandchild holding the output pipes must not outlive the timeout.
	cmd.WaitDelay = time.Second
	stdout := &limitedBuffer{limit: limit}
	stderr := &limitedBuffer{limit: limit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if err := ctx.Err(); err != nil {
		// The caller abandoned discovery; that is not evidence about the host.
		return Entry{}, err
	}
	if commandCtx.Err() != nil {
		entry.Status = StatusTimedOut
		return entry, nil
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			entry.Status = StatusStartFailed
			return entry, nil
		}
		entry.Status = StatusExitedNonzero
		entry.ExitCode = exitErr.ExitCode()
	} else {
		entry.Status = StatusCompleted
		entry.ExitCode = 0
	}
	entry.OutputDigest = digestOutput(stdout.buf.Bytes(), stderr.buf.Bytes())
	entry.OutputTruncated = stdout.truncated || stderr.truncated
	if entry.Status != StatusCompleted {
		return entry, nil
	}
	if entry.OutputTruncated {
		// A partial stream could omit models or a version line.
		entry.OutputUnparsed = true
		return entry, nil
	}
	parsed := true
	switch command.Kind {
	case KindVersion:
		entry.Version, parsed = parseVersion(stdout.buf.Bytes())
	case KindModelAvailability:
		entry.Models, parsed = parseModels(stdout.buf.Bytes())
	}
	entry.OutputUnparsed = !parsed
	return entry, nil
}

// Discover runs only the exact commands supplied in the host allowlist. It
// invokes no shell, retains no raw command output beyond bounded parsed
// fields, and never makes an eligibility or route-selection decision.
func Discover(ctx context.Context, request Request) (Inventory, error) {
	if ctx == nil || request.CapturedAt.IsZero() || request.Freshness <= 0 || len(request.Commands) == 0 {
		return Inventory{}, ErrInvalidRequest
	}
	seen := make(map[string]struct{}, len(request.Commands))
	for _, command := range request.Commands {
		if err := validateCommand(command); err != nil {
			return Inventory{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
		}
		if _, exists := seen[command.ID]; exists {
			return Inventory{}, fmt.Errorf("%w: duplicate command id %q", ErrInvalidRequest, command.ID)
		}
		seen[command.ID] = struct{}{}
	}
	entries := make([]Entry, 0, len(request.Commands))
	for _, command := range request.Commands {
		entry, err := captureCommand(ctx, command, request)
		if err != nil {
			return Inventory{}, err
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	capturedAt := request.CapturedAt.UTC()
	inventory := Inventory{Version: VersionV1, CapturedAt: capturedAt.Format(time.RFC3339Nano), FreshUntil: capturedAt.Add(request.Freshness).Format(time.RFC3339Nano), Entries: entries}
	digest, err := Digest(inventory)
	if err != nil {
		return Inventory{}, fmt.Errorf("%w: %v", ErrDigestFailure, err)
	}
	inventory.Digest = digest
	return inventory, nil
}

// Digest hashes the canonical inventory projection without its self-referential
// Digest field. Entry ordering is normalized by Discover.
func Digest(inventory Inventory) (string, error) {
	projection := inventory
	projection.Digest = ""
	canonical, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func (inventory Inventory) Fresh(now time.Time) bool {
	if inventory.FreshUntil == "" || now.IsZero() {
		return false
	}
	freshUntil, err := time.Parse(time.RFC3339Nano, inventory.FreshUntil)
	return err == nil && now.Before(freshUntil)
}

// limitedBuffer deliberately does not embed *bytes.Buffer: that would promote
// ReadFrom, and io.Copy in os/exec would then bypass Write and the limit.
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	room := b.limit - b.buf.Len()
	if room >= len(p) {
		return b.buf.Write(p)
	}
	if room > 0 {
		_, _ = b.buf.Write(p[:room])
	}
	b.truncated = true
	return len(p), nil
}
