package evidence

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RequireSelfContainedRepository refuses a repository whose .git is not a
// real directory (a pointer file or a link the fingerprint cannot follow) or
// that names alternate object stores (content outside .git). It reads no
// git object, so a blocking alternates file cannot hang it.
func RequireSelfContainedRepository(repo string) error {
	refused := func(why string) error { return errors.New("repository object store refused: " + why) }
	gitDir := filepath.Join(repo, ".git")
	if fi, err := os.Lstat(gitDir); err != nil || !fi.IsDir() {
		return refused(".git is not a directory")
	}
	// Git reads commondir in any git directory and then takes objects, refs
	// and configuration from the directory it names, outside the fingerprint.
	for _, name := range []string{"commondir", "gitdir"} {
		if _, err := os.Lstat(filepath.Join(gitDir, name)); err == nil {
			return refused(".git/" + name + " redirects git outside .git")
		}
	}
	if _, err := os.Lstat(filepath.Join(gitDir, "objects", "info", "alternates")); err == nil {
		return refused("alternate object stores are not supported")
	}
	// Every entry under .git is a regular file or a directory: a link would
	// send git (and the fingerprint's own reading) to a place the fingerprint
	// does not follow, and a FIFO or device would block a read. The host's
	// dispatch ledger is exempt, as in the fingerprint.
	ledger := filepath.Join(gitDir, "agent-dispatch")
	if err := filepath.WalkDir(gitDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == ledger && d.IsDir() {
			return filepath.SkipDir
		}
		if t := d.Type(); t&^fs.ModeDir != 0 {
			return refused("a link or special file under .git: " + path)
		}
		return nil
	}); err != nil {
		return err
	}
	return requirePlainConfig(repo)
}

// requirePlainConfig refuses a repository whose local configuration names any
// key outside the allow list PrepareGitExecution uses. HMA's own reads (the
// snapshot, the path listing, the copy) run under the live configuration, so
// a key that changes how git reads objects (core.multiPackIndex, an include
// that pulls in more settings, pack.*, a commit graph) could make them see a
// different object than the verified store holds. The configuration is read
// once into a private copy and listed by git's own parser with includes not
// followed; an include key is itself outside the list and refused.
func requirePlainConfig(repo string) error {
	refused := errors.New("repository object store refused: local git configuration is not plain")
	raw, err := os.ReadFile(filepath.Join(repo, ".git", "config"))
	if err != nil {
		return refused
	}
	found, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	git, err := filepath.EvalSymlinks(found)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "hma-config-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	cfg := filepath.Join(dir, "config")
	if err := os.WriteFile(cfg, raw, 0o400); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c := exec.CommandContext(ctx, git, "config", "--file", cfg, "--list", "--name-only", "-z")
	c.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "HOME=/nonexistent", "PATH=" + os.Getenv("PATH")}
	c.Dir = os.TempDir()
	out, err := c.Output()
	if err != nil {
		return refused
	}
	for _, name := range strings.Split(string(out), "\x00") {
		if name != "" && !gitConfigNameAllowed(name) {
			return refused
		}
	}
	return nil
}

// VerifyRepositoryObjects refuses a repository whose object store cannot be
// shown to hold, byte for byte, the history that ends at head. It requires a
// real .git directory (not a pointer file or a link), no alternate object
// stores (their content lives outside the fingerprinted .git), and then
// builds a private store from the objects reachable from head and checks
// every one hashes to its id with git fsck. A forged or substituted object,
// loose or packed, makes it fail. It writes nothing outside a private
// temporary directory. The caller fingerprints .git before and after, so a
// change made after this check is caught as a change.
func VerifyRepositoryObjects(ctx context.Context, repo, head string) error {
	bad := errors.New("repository object store refused")
	if err := RequireSelfContainedRepository(repo); err != nil {
		return err
	}
	gitDir := filepath.Join(repo, ".git")
	found, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	git, err := filepath.EvalSymlinks(found)
	if err != nil {
		return err
	}
	priv, err := os.MkdirTemp("", "hma-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(priv)
	for _, d := range []string{"refs/heads", "refs/tags"} {
		if err := os.MkdirAll(filepath.Join(priv, d), 0o700); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(priv, "HEAD"), []byte(head+"\n"), 0o600); err != nil {
		return err
	}
	cfg := "[core]\n\trepositoryformatversion = 0\n\tbare = false\n"
	// RequireSelfContainedRepository showed .git/config is a regular file.
	live, err := os.ReadFile(filepath.Join(gitDir, "config"))
	if err != nil {
		return bad
	}
	tmp := filepath.Join(priv, "livecfg")
	if err := os.WriteFile(tmp, live, 0o400); err != nil {
		return err
	}
	c := exec.CommandContext(ctx, git, "config", "--file", tmp, "--get", "extensions.objectformat")
	c.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "HOME=/nonexistent", "PATH=" + os.Getenv("PATH")}
	c.Dir = os.TempDir()
	if out, err := c.Output(); err == nil && strings.TrimSpace(string(out)) == "sha256" {
		cfg = "[core]\n\trepositoryformatversion = 1\n\tbare = false\n[extensions]\n\tobjectformat = sha256\n"
	}
	os.Remove(tmp)
	if err := os.WriteFile(filepath.Join(priv, "config"), []byte(cfg), 0o400); err != nil {
		return err
	}
	return buildPinnedStore(ctx, git, priv, filepath.Join(gitDir, "objects"), head)
}
