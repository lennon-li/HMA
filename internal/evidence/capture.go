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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
)

type Request struct {
	Executable     string
	Argv           []string
	WorkingDir     string
	RepositoryRoot string
	// GitRepository, when set, is the approved repository whose .git a git
	// command reads while RepositoryRoot is a private copy of its worktree.
	GitRepository string
	// Sandbox runs the command in an operating-system sandbox that shows it
	// only RepositoryRoot (read-write), the system's read-only program
	// directories and a private scratch directory: no network, no other host
	// path. PinnedHead is the commit a git command sees as HEAD, with no
	// other ref. Capture fails if the sandbox cannot start.
	Sandbox    bool
	PinnedHead string
	// StagedTarget says the reviewed target has staged changes, known from the
	// bound snapshot and not read from the live index. A sandboxed git command
	// is refused for it: the pinned view holds the reviewed commit's index.
	StagedTarget   bool
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
func resolveExecutable(name string) (string, error) { return ResolveExecutable(name) }

// ResolveExecutable is the capture's own resolution of a command name to one
// absolute path, exported so a kernel-rule fact judges the same program.
func ResolveExecutable(name string) (string, error) {
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
	// Containment is judged on the filesystem, not lexically: a symlink
	// component cannot move the check outside the repository. The check
	// then runs in the resolved directory.
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return Result{}, errors.New("working directory outside repository")
	}
	if wd, err = filepath.EvalSymlinks(wd); err != nil {
		return Result{}, errors.New("working directory outside repository")
	}
	rel, err := filepath.Rel(root, wd)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Result{}, errors.New("working directory outside repository")
	}
	req.WorkingDir = wd
	resolved, err := resolveExecutable(req.Executable)
	if err != nil {
		return Result{}, err
	}
	if reason := RejectShell(req.Executable, resolved, req.Argv); reason != "" {
		return Result{}, fmt.Errorf("no-shell policy: %s", reason)
	}
	// Policy is applied to the program that runs, not the name it was
	// given: a symlink to git is git. The canonical path is what runs and
	// what the record shows.
	var gitExec []string
	var gitPriv, gitObjects string
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		return Result{}, err
	}
	if IsGit(resolved) || IsGit(canonical) {
		// A git command is one of the approved forms, runs only in the
		// approved repository, under plain local configuration: no
		// repository selector, alias or configured helper can apply.
		if !GitFormAllowed(req.Argv) {
			return Result{}, errors.New("git command form not approved")
		}
		repo := root
		if req.GitRepository != "" {
			if repo, err = filepath.EvalSymlinks(req.GitRepository); err != nil {
				return Result{}, errors.New("git repository configuration refused")
			}
		}
		pin := ""
		if req.Sandbox {
			if req.StagedTarget {
				return Result{}, errors.New("staged changes are not supported for sandboxed git checks")
			}
			if req.PinnedHead == "" {
				return Result{}, errors.New("sandboxed git command needs a pinned head")
			}
			pin = req.PinnedHead
		}
		priv, err := PrepareGitExecution(canonical, repo, root, wd, pin)
		if err != nil {
			return Result{}, err
		}
		defer os.RemoveAll(priv)
		gitExec = []string{"GIT_DIR=" + priv, "GIT_WORK_TREE=" + root}
		if req.Sandbox {
			gitPriv, gitObjects = priv, filepath.Join(priv, "objects")
			gitExec = append(gitExec, "GIT_OBJECT_DIRECTORY="+sandboxObjects)
		}
	}
	resolved = canonical
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
	env := req.Environment
	if env == nil {
		env = os.Environ()
	}
	cmd.Env = append(repostate.GitEnv(env), gitExec...)
	if req.Sandbox {
		live := req.GitRepository
		if live == "" {
			live = root
		}
		if cmd, err = sandboxCommand(ctx, resolved, req.Argv, root, wd, live, gitPriv, gitObjects, cmd.Env); err != nil {
			return Result{}, err
		}
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
