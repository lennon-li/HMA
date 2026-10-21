package evidence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// gitConfigAllowed lists the repository-local git configuration keys a
// captured git command may run under. Any other key (a filter, a gpg or diff
// program, a pager, an alias, an include, an extension that adds a
// configuration file) can start a program or change what a read-only command
// does, so a repository that sets one is refused. Section and key names are
// lower case; "*" stands for one subsection.
var gitConfigAllowed = map[string]bool{
	"core.repositoryformatversion": true, "core.filemode": true, "core.bare": true, "core.logallrefupdates": true,
	"core.ignorecase": true, "core.precomposeunicode": true, "core.symlinks": true, "core.autocrlf": true,
	"user.name": true, "user.email": true, "extensions.objectformat": true,
	"remote.*.url": true, "remote.*.fetch": true, "branch.*.remote": true, "branch.*.merge": true,
}

// prepareHook, when set by a test, runs after the configuration copy is
// validated and before the view is built.
var prepareHook func()

// IsGit reports whether a resolved executable is git.
func IsGit(resolved string) bool {
	b := path.Base(filepath.ToSlash(resolved))
	return b == "git" || b == "git.exe"
}

// PrepareGitExecution refuses a git command unless the repository it will
// read is exactly the approved one and its local configuration is plain, and
// returns a private git directory to run it against. Checking the real
// .git/config and then letting git reopen it would leave a window for a
// writer; instead the configuration is read once, copied into a private
// directory (mode 0400), checked there with git's own parser, and git is
// pointed at that directory (GIT_DIR) for the command, so the bytes checked
// are the bytes git reads. The private directory copies HEAD and links the real refs,
// objects, index, packed-refs, shallow and info; git replaces the index link
// with its own file when it refreshes, so the real index is not written.
//
// The approved root must hold a real .git directory (not a file that points
// elsewhere) with no worktree configuration, and no directory from the
// working directory up to (not including) root may hold its own .git, so git
// cannot pick a nested repository. The caller removes the returned directory.
func PrepareGitExecution(git, repo, root, wd, pin string) (string, error) {
	bad := errors.New("git repository configuration refused")
	// A live writer can make a preparation read block (a FIFO named as an
	// alternates file, say); every subprocess is bounded.
	gctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	gitDir := filepath.Join(repo, ".git")
	fi, err := os.Lstat(gitDir)
	if err != nil || !fi.IsDir() {
		return "", bad
	}
	if _, err := os.Lstat(filepath.Join(gitDir, "config.worktree")); err == nil {
		return "", bad
	}
	for dir := wd; dir != root; dir = filepath.Dir(dir) {
		if dir == filepath.Dir(dir) {
			return "", bad
		}
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return "", bad
		}
	}
	raw, err := os.ReadFile(filepath.Join(gitDir, "config"))
	if err != nil {
		return "", bad
	}
	priv, err := os.MkdirTemp("", "hma-gitdir-")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(priv)
		}
	}()
	cfg := filepath.Join(priv, "config")
	if err := os.WriteFile(cfg, raw, 0o400); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(gctx, git, "config", "--file", cfg, "--list", "--name-only", "-z")
	cmd.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "HOME=/nonexistent", "PATH=" + os.Getenv("PATH")}
	cmd.Dir = os.TempDir()
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", bad
	}
	for _, name := range strings.Split(out.String(), "\x00") {
		if name != "" && !gitConfigNameAllowed(name) {
			return "", bad
		}
	}
	if prepareHook != nil {
		prepareHook()
	}
	// git does not accept a symlinked HEAD; it is one line, so copy it.
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", bad
	}
	if err := os.WriteFile(filepath.Join(priv, "HEAD"), head, 0o400); err != nil {
		return "", err
	}
	if pin != "" {
		// A pinned view holds only the reviewed state: HEAD detached at the
		// reviewed commit, empty refs, a private copy of the index. No live
		// ref can change what a check reads. The caller points
		// GIT_OBJECT_DIRECTORY at the object store.
		os.Remove(filepath.Join(priv, "HEAD"))
		if err := os.WriteFile(filepath.Join(priv, "HEAD"), []byte(pin+"\n"), 0o600); err != nil {
			return "", err
		}
		for _, d := range []string{"refs/heads", "refs/tags"} {
			if err := os.MkdirAll(filepath.Join(priv, d), 0o700); err != nil {
				return "", err
			}
		}
		// The review observed the repository under fixed settings (see
		// repostate.observationConfig), not under the live .git/config, which
		// no digest binds. The view's configuration is therefore fixed too;
		// only the object format is carried over.
		format := "sha1"
		if out, err := func() ([]byte, error) {
			c := exec.CommandContext(gctx, git, "config", "--file", cfg, "--get", "extensions.objectformat")
			c.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "HOME=/nonexistent", "PATH=" + os.Getenv("PATH")}
			c.Dir = os.TempDir()
			return c.Output()
		}(); err == nil && strings.TrimSpace(string(out)) == "sha256" {
			format = "sha256"
		}
		fixed := "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tsymlinks = true\n\tautocrlf = false\n\tignorecase = false\n\tbare = false\n"
		if format == "sha256" {
			fixed = "[core]\n\trepositoryformatversion = 1\n\tfilemode = true\n\tsymlinks = true\n\tautocrlf = false\n\tignorecase = false\n\tbare = false\n[extensions]\n\tobjectformat = sha256\n"
		}
		if err := os.Remove(cfg); err != nil {
			return "", err
		}
		if err := os.WriteFile(cfg, []byte(fixed), 0o400); err != nil {
			return "", err
		}
		if _, err := os.Lstat(filepath.Join(gitDir, "shallow")); err == nil {
			return "", errors.New("shallow repository not supported for sandboxed git checks")
		}
		if err := buildPinnedStore(gctx, git, priv, filepath.Join(gitDir, "objects"), pin); err != nil {
			return "", err
		}
		ok = true
		return priv, nil
	}
	for _, name := range []string{"refs", "objects", "index", "packed-refs", "shallow", "info"} {
		src := filepath.Join(gitDir, name)
		if _, err := os.Lstat(src); err != nil {
			continue
		}
		if err := os.Symlink(src, filepath.Join(priv, name)); err != nil {
			return "", err
		}
	}
	ok = true
	return priv, nil
}

// buildPinnedStore gives the private git directory its own object store with
// exactly the objects reachable from the reviewed commit, and an index built
// from that commit. Later objects in the live store, and the live index, are
// never visible. Dirty worktree content is not in the index: a check sees it
// as modified, which is what the review saw.
func buildPinnedStore(gctx context.Context, git, priv, liveObjects, pin string) error {
	bad := errors.New("pinned git view refused")
	pack := filepath.Join(priv, "objects", "pack")
	if err := os.MkdirAll(pack, 0o700); err != nil {
		return err
	}
	env := []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "HOME=/nonexistent",
		"PATH=" + os.Getenv("PATH"), "GIT_DIR=" + priv, "GIT_OBJECT_DIRECTORY=" + filepath.Join(priv, "objects"), "GIT_ALTERNATE_OBJECT_DIRECTORIES=" + liveObjects, "GIT_INDEX_FILE=" + filepath.Join(priv, "index")}
	list := exec.CommandContext(gctx, git, "rev-list", "--objects", pin)
	list.Env, list.Dir = env, os.TempDir()
	var ids bytes.Buffer
	list.Stdout = &ids
	if err := list.Run(); err != nil {
		return bad
	}
	var names bytes.Buffer
	for _, line := range strings.Split(ids.String(), "\n") {
		if id, _, _ := strings.Cut(line, " "); id != "" {
			names.WriteString(id + "\n")
		}
	}
	pk := exec.CommandContext(gctx, git, "pack-objects", "-q", filepath.Join(pack, "pack"))
	pk.Env, pk.Dir, pk.Stdin = env, os.TempDir(), &names
	if err := pk.Run(); err != nil {
		return bad
	}
	// The pack was read from the live store, which a writer can change. Verify
	// every packed object hashes to its id, in the private store only, so a
	// substituted object under a reviewed id is refused, not read as reviewed.
	fsck := exec.CommandContext(gctx, git, "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "fsck", "--full", "--strict", "--no-dangling", "--no-progress")
	fsck.Env, fsck.Dir = append([]string(nil), env...), os.TempDir()
	fsck.Env = append(fsck.Env[:5:5], "GIT_DIR="+priv, "GIT_OBJECT_DIRECTORY="+filepath.Join(priv, "objects"))
	if out, err := fsck.CombinedOutput(); err != nil {
		return fmt.Errorf("pinned git view refused: object verification failed: %s", strings.TrimSpace(string(out)))
	}
	rt := exec.CommandContext(gctx, git, "read-tree", pin)
	rt.Env, rt.Dir = append([]string(nil), env...), os.TempDir()
	rt.Env = append(rt.Env[:5:5], "GIT_DIR="+priv, "GIT_OBJECT_DIRECTORY="+filepath.Join(priv, "objects"), "GIT_INDEX_FILE="+filepath.Join(priv, "index"))
	if err := rt.Run(); err != nil {
		return bad
	}
	return nil
}

// gitConfigNameAllowed judges one key as git prints it with --name-only:
// section[.subsection].key, section and key lower case.
func gitConfigNameAllowed(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	section, key := strings.ToLower(parts[0]), strings.ToLower(parts[len(parts)-1])
	full := section + "." + key
	if len(parts) > 2 {
		full = section + ".*." + key
	}
	return gitConfigAllowed[full]
}
