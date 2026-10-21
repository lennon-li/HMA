package evidence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// sandboxObjects is where a sandboxed git command sees the object store.
const sandboxObjects = "/hma-objects"

// sandboxEnvAllowed lists the environment a sandboxed command keeps. Nothing else
// from the host (tokens, cache locations, repository selectors) reaches it.
func sandboxEnvAllowed(name string) bool {
	switch name {
	case "PATH", "LANG", "GOROOT", "GOFLAGS", "GOTOOLCHAIN", "GOPROXY", "CGO_ENABLED", "GOEXPERIMENT":
		return true
	}
	return strings.HasPrefix(name, "LC_") || strings.HasPrefix(name, "GIT_")
}

// systemDirs are shown read-only when they exist.
var systemDirs = []string{"/usr", "/bin", "/sbin", "/lib", "/lib32", "/lib64", "/libx32"}

// systemFiles are the few /etc entries programs need to start.
var systemFiles = []string{"/etc/ld.so.cache", "/etc/ld.so.conf", "/etc/ld.so.conf.d", "/etc/alternatives", "/etc/ssl", "/etc/passwd", "/etc/group", "/etc/localtime"}

// sandboxCommand wraps the command in bubblewrap: a fresh mount, pid, ipc,
// uts and network namespace in which only the private copy (root, read-write),
// the system program directories and the executable's own tree (read-only),
// a scratch directory and, for git, the pinned git directory and the object
// store (read-only) exist. The live repository and every other host path are
// absent, so no symlink, absolute path or computed path can reach them. If
// the sandbox cannot be started the command is refused; it never runs
// unconfined.
func sandboxCommand(ctx context.Context, exe string, argv []string, root, wd, live, gitPriv, gitObjects string, env []string) (*exec.Cmd, error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return nil, errors.New("sandbox unavailable: bwrap not found")
	}
	if bwrap, err = filepath.Abs(bwrap); err != nil {
		return nil, err
	}
	// The system directories are shown whole. A repository that lives inside
	// one (a checkout under /usr/local/src, say) would stay visible next to
	// the private copy, so it is refused. Paths are resolved, so a link into
	// a system directory counts.
	// Every path the sandbox will bind from the host, directories and the
	// /etc entries alike, must neither contain nor lie inside the live
	// repository or the copy root.
	for _, p := range []string{live, root} {
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return nil, fmt.Errorf("sandbox refused: %v", err)
		}
		for _, m := range append(append([]string(nil), systemDirs...), systemFiles...) {
			if _, serr := os.Stat(m); serr != nil {
				continue
			}
			src, err := filepath.EvalSymlinks(m)
			if err != nil {
				return nil, fmt.Errorf("sandbox refused: %v", err)
			}
			if within(resolved, src) || within(src, resolved) {
				return nil, fmt.Errorf("sandbox refused: %s overlaps system path %s the sandbox shows", resolved, src)
			}
		}
	}
	args := []string{"--die-with-parent", "--unshare-all", "--clearenv", "--proc", "/proc", "--dev", "/dev"}
	for _, d := range systemDirs {
		if _, err := os.Stat(d); err == nil {
			args = append(args, "--ro-bind", d, d)
		}
	}
	for _, f := range systemFiles {
		if _, err := os.Stat(f); err == nil {
			args = append(args, "--ro-bind", f, f)
		}
	}
	// The executable's tree, when it is not a system directory (a toolchain
	// such as GOROOT). It must not contain, or sit inside, the live
	// repository.
	if !underAny(exe, systemDirs) {
		tree := filepath.Dir(exe)
		if filepath.Base(tree) == "bin" {
			tree = filepath.Dir(tree)
		}
		if tree == "/" || within(live, tree) || within(tree, live) || within(tree, root) {
			return nil, fmt.Errorf("sandbox refused: executable tree %s overlaps the repository", tree)
		}
		args = append(args, "--ro-bind", tree, tree)
	}
	args = append(args, "--tmpfs", "/tmp", "--tmpfs", "/scratch", "--bind", root, root)
	if gitPriv != "" {
		args = append(args, "--ro-bind", gitPriv, "/hma-git", "--ro-bind", gitObjects, sandboxObjects)
	}
	args = append(args, "--chdir", wd)
	kept := []string{"HOME=/scratch/home", "TMPDIR=/scratch", "GOCACHE=/scratch/gocache"}
	for _, kv := range env {
		name, val, _ := strings.Cut(kv, "=")
		if !sandboxEnvAllowed(name) {
			continue
		}
		if name == "GIT_DIR" && gitPriv != "" {
			val = "/hma-git"
		}
		kept = append(kept, name+"="+val)
	}
	for _, kv := range kept {
		name, val, _ := strings.Cut(kv, "=")
		args = append(args, "--setenv", name, val)
	}
	// A sandbox that cannot start must not read as a failed check: probe
	// first with a command whose success is certain.
	probe := exec.CommandContext(ctx, bwrap, append(append([]string(nil), args...), "/usr/bin/true")...)
	if out, err := probe.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("sandbox unavailable: %v: %s", err, strings.TrimSpace(string(out)))
	}
	full := append(append(append([]string(nil), args...), "--", exe), argv...)
	cmd := exec.CommandContext(ctx, bwrap, full...)
	cmd.Dir = os.TempDir()
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	return cmd, nil
}

func within(path, dir string) bool {
	r, err := filepath.Rel(dir, path)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

func underAny(path string, dirs []string) bool {
	for _, d := range dirs {
		if within(path, d) {
			return true
		}
	}
	return false
}
