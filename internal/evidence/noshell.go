package evidence

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// shellNames are command interpreters a capture never runs, under any name
// form (bare, absolute, or a symlink that resolves to one).
var shellNames = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true, "fish": true, "ksh": true, "mksh": true,
	"ash": true, "csh": true, "tcsh": true, "busybox": true, "rbash": true, "yash": true, "posh": true,
}

func isShell(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), ".exe")
	return shellNames[base] || strings.HasPrefix(base, "bash") && len(base) > 4 && (base[4] == '-' || base[4] >= '0' && base[4] <= '9')
}

// wrappers run their arguments as a command, so a shell among their
// arguments is a shell entry point.
var wrappers = map[string]bool{
	"env": true, "xargs": true, "nice": true, "nohup": true, "timeout": true, "stdbuf": true, "setsid": true,
	"time": true, "command": true, "exec": true, "sudo": true, "doas": true, "chroot": true, "flock": true,
	"ionice": true, "taskset": true, "unbuffer": true, "script": true, "su": true, "runuser": true, "find": true,
	"watch": true, "strace": true, "ltrace": true, "valgrind": true, "fakeroot": true, "parallel": true,
}

// RejectShell returns a reason when running executable (resolved to the
// absolute path resolved) would reach a shell. Architecture §14: evidence runs
// one explicit executable and argv, never a shell command string or a shell
// script entry point. It refuses:
//   - an executable that is, or is a symlink to, a shell;
//   - an executable that is, or is a symlink to, a command wrapper (env,
//     xargs, nice, ...): what a wrapper runs is an operand HMA would have to
//     parse per wrapper, so wrappers are not admitted at all;
//   - a script whose #! interpreter, followed through symlinks, nested #!
//     lines, and env-style launchers, is a shell or a wrapper;
//   - a shell name followed by a command-string flag (-c, -lc, -ec, ...)
//     anywhere in argv, as verify-loop's reject_unsafe_argv does.
//
// A shell name that is plain data (printf %s sh) is not refused. A compiled
// program that starts a shell itself cannot be told apart from any other
// program by inspection; under the trusted-host model that is bounded by the
// approved check argv (Gate D binds the check to the accepted plan).
func RejectShell(executable, resolved string, argv []string) string {
	if isShell(executable) {
		return "shell executable"
	}
	if wrappers[strings.TrimSuffix(filepath.Base(executable), ".exe")] {
		return "command wrapper"
	}
	if r := rejectProgram(resolved, 0); r != "" {
		return r
	}
	for i, a := range argv {
		if isShell(a) && i+1 < len(argv) && commandFlag(argv[i+1]) {
			return "shell command string"
		}
	}
	return ""
}

// rejectProgram checks one program path: its own name, every name on its
// symlink chain's target, and its #! interpreter, recursively (Linux follows
// nested interpreters a few levels deep; more than that is refused).
func rejectProgram(path string, depth int) string {
	if depth > 4 {
		return "interpreter chain too deep"
	}
	names := []string{path}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		names = append(names, real)
		path = real
	}
	for _, n := range names {
		base := strings.TrimSuffix(filepath.Base(n), ".exe")
		if isShell(n) {
			return "shell executable"
		}
		if wrappers[base] {
			return "command wrapper"
		}
	}
	interp, args, ok := shebang(path)
	if !ok {
		return ""
	}
	if r := rejectProgram(interp, depth+1); r != "" {
		return "script interpreter: " + r
	}
	// An env-style launcher named in #! is refused above as a wrapper, so
	// args here are the interpreter's own options; a shell among them would
	// still be a shell entry point (#!/usr/bin/python3 -m sh is not, but
	// refusing a shell-named option is the safe side).
	for _, a := range args {
		if isShell(a) {
			return "script interpreter: shell"
		}
	}
	return ""
}

// shebang reads a #! line from the first bytes of path.
func shebang(path string) (string, []string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, false
	}
	defer f.Close()
	buf := make([]byte, 256)
	n, _ := io.ReadFull(f, buf)
	buf = buf[:n]
	if !bytes.HasPrefix(buf, []byte("#!")) {
		return "", nil, false
	}
	line := buf[2:]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(string(line))
	if len(fields) == 0 {
		return "", nil, false
	}
	interp := fields[0]
	if !filepath.IsAbs(interp) {
		if p, err := exec.LookPath(interp); err == nil {
			interp = p
		}
	}
	return interp, fields[1:], true
}

// commandFlag reports a short-option cluster containing c, such as -c, -lc,
// -ec, or -xc.
func commandFlag(a string) bool {
	return len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], 'c')
}
