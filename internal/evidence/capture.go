// Package evidence captures one caller-allowlisted executable and argv without a shell.
package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

type Request struct {
	Executable         string
	Argv               []string
	WorkingDir         string
	RepositoryRoot     string
	Environment        []string
	MaxOutputBytes     int
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
	limit := req.MaxOutputBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	out := &limitedBuffer{n: limit}
	erout := &limitedBuffer{n: limit}
	start := time.Now().UTC()
	cmd := exec.CommandContext(ctx, req.Executable, req.Argv...)
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
	h := sha256.New()
	h.Write(out.b.Bytes())
	h.Write(erout.b.Bytes())
	e := model.EvidenceRef{Executable: req.Executable, Argv: append([]string(nil), req.Argv...), WorkingDir: wd, StartTimestamp: start.Format(time.RFC3339Nano), EndTimestamp: end.Format(time.RFC3339Nano), ExitCode: exitCode, OutputDigest: hex.EncodeToString(h.Sum(nil)), OutputTruncated: out.truncated || erout.truncated, RepositoryIdentity: req.RepositoryIdentity, BaseRevision: req.BaseRevision, HeadRevision: req.HeadRevision, ChangedFiles: append([]string(nil), req.ChangedFiles...), DiffDigest: req.DiffDigest, WorktreeDigest: req.WorktreeDigest, CapturingActor: req.CapturingActor}
	return Result{Evidence: e, Stdout: out.b.Bytes(), Stderr: erout.b.Bytes()}, nil
}
