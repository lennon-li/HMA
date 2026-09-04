// Package evidence captures one explicit executable and argument vector
// without a shell. The caller may additionally supply an allowlist of resolved
// absolute executable paths; when it is empty no allowlist is enforced, which
// is a host-trust decision the caller makes explicitly rather than a property
// this package claims.
package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

type Request struct {
	Executable     string
	Argv           []string
	WorkingDir     string
	RepositoryRoot string
	Environment    []string
	MaxOutputBytes int
	// AllowedExecutables, when non-empty, restricts execution to these
	// resolved absolute paths.
	AllowedExecutables []string
	RepositoryIdentity string
	BaseRevision       string
	HeadRevision       string
	ChangedFiles       []string
	DiffDigest         string
	WorktreeDigest     string
	CapturingActor     string
}
type Result struct {
	Evidence       model.EvidenceRef
	Stdout, Stderr []byte
}
type limitedBuffer struct {
	b         bytes.Buffer
	n         int
	truncated bool
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	room := w.n - w.b.Len()
	if room > 0 {
		if room < len(p) {
			w.b.Write(p[:room])
		} else {
			w.b.Write(p)
		}
	}
	if original > room {
		w.truncated = true
	}
	return original, nil
}

// writeDigestSection length-frames one captured stream so that the digest of
// (stdout, stderr) is unambiguous: without framing, ("ab", "c") and
// ("a", "bc") hash identically.
func writeDigestSection(h hash.Hash, name string, data []byte) {
	h.Write([]byte("hma-evidence-v1\x00" + name + "\x00"))
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	h.Write(length[:])
	h.Write(data)
}

// resolveExecutable pins the command to one absolute path. exec.Command
// resolves a bare name against the parent process PATH rather than cmd.Env,
// so an unresolved name does not identify what actually ran; recording the
// resolved path keeps the evidence record self-describing.
func resolveExecutable(name string) (string, error) {
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}
	if strings.ContainsRune(name, filepath.Separator) {
		return "", errors.New("relative executable path is not permitted")
	}
	found, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return filepath.Abs(found)
}

func Capture(ctx context.Context, req Request) (Result, error) {
	if req.Executable == "" || strings.ContainsAny(req.Executable, "|&;<>()$`\\\n\r") || req.WorkingDir == "" || req.RepositoryRoot == "" {
		return Result{}, errors.New("invalid command boundary")
	}
	root, err := filepath.Abs(req.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	wd, err := filepath.Abs(req.WorkingDir)
	if err != nil {
		return Result{}, err
	}
	rel, err := filepath.Rel(root, wd)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Result{}, errors.New("working directory outside repository")
	}
	resolved, err := resolveExecutable(req.Executable)
	if err != nil {
		return Result{}, err
	}
	if len(req.AllowedExecutables) > 0 {
		permitted := false
		for _, allowed := range req.AllowedExecutables {
			if allowed == resolved {
				permitted = true
				break
			}
		}
		if !permitted {
			return Result{}, fmt.Errorf("executable %q is not allowlisted", resolved)
		}
	}
	limit := req.MaxOutputBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	out := &limitedBuffer{n: limit}
	erout := &limitedBuffer{n: limit}
	start := time.Now().UTC()
	cmd := exec.CommandContext(ctx, resolved, req.Argv...)
	cmd.Dir = wd
	if req.Environment != nil {
		cmd.Env = req.Environment
	}
	cmd.Stdout = out
	cmd.Stderr = erout
	runErr := cmd.Run()
	end := time.Now().UTC()
	exitCode := 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitCode()
		} else {
			return Result{}, runErr
		}
	}
	// An executable invoked with no arguments has an empty argument vector,
	// not an absent one; the record must be able to say so.
	h := sha256.New()
	writeDigestSection(h, "stdout", out.b.Bytes())
	writeDigestSection(h, "stderr", erout.b.Bytes())
	e := model.EvidenceRef{Executable: resolved, Argv: append([]string{}, req.Argv...), WorkingDir: wd, StartTimestamp: start.Format(time.RFC3339Nano), EndTimestamp: end.Format(time.RFC3339Nano), ExitCode: exitCode, OutputDigest: hex.EncodeToString(h.Sum(nil)), OutputTruncated: out.truncated || erout.truncated, RepositoryIdentity: req.RepositoryIdentity, BaseRevision: req.BaseRevision, HeadRevision: req.HeadRevision, ChangedFiles: append([]string(nil), req.ChangedFiles...), DiffDigest: req.DiffDigest, WorktreeDigest: req.WorktreeDigest, CapturingActor: req.CapturingActor}
	return Result{Evidence: e, Stdout: out.b.Bytes(), Stderr: erout.b.Bytes()}, nil
}
