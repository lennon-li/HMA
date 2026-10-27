package repostate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
)

// BlobWindow streams a blob, returning its complete SHA-256 and only the
// first limit bytes for bounded line counting.
func BlobWindow(ctx context.Context, root, oid string, limit int64) ([]byte, string, bool, error) {
	args := append(append([]string{"--no-replace-objects"}, observationConfig...), "cat-file", "blob", oid)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	cmd.Env = gitEnv(os.Environ())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", false, err
	}
	if err = cmd.Start(); err != nil {
		return nil, "", false, err
	}
	h := sha256.New()
	window := &boundedWindow{limit: limit}
	if _, err = io.Copy(io.MultiWriter(h, window), stdout); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, "", false, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, "", false, err
	}
	return window.data, hex.EncodeToString(h.Sum(nil)), window.overflow, nil
}

type boundedWindow struct {
	data     []byte
	limit    int64
	overflow bool
}

func (w *boundedWindow) Write(p []byte) (int, error) {
	if int64(len(w.data)) < w.limit {
		n := int(w.limit - int64(len(w.data)))
		if n > len(p) {
			n = len(p)
		}
		w.data = append(w.data, p[:n]...)
		if n < len(p) {
			w.overflow = true
		}
	} else if len(p) > 0 {
		w.overflow = true
	}
	return len(p), nil
}

// ChangedPaths is the set of paths whose state differs from head under the
// snapshot's own rules (tracked changes, staged changes, untracked paths not
// excluded by a committed .gitignore). The snapshot digests exactly these
// paths, so anything that describes the same target must use this set.
func ChangedPaths(ctx context.Context, root, head string) ([]string, error) {
	return changedPaths(ctx, root, head)
}

// BlobLines counts newline-delimited lines in the first 16 MiB of a Git blob.
// It streams object bytes and stops the object reader at the cap.
func BlobLines(ctx context.Context, root, oid string) (int, bool, error) {
	args := append(append([]string{"--no-replace-objects"}, observationConfig...), "cat-file", "blob", oid)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	cmd.Env = gitEnv(os.Environ())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, false, err
	}
	if err = cmd.Start(); err != nil {
		return 0, false, err
	}
	const limit = 16 << 20
	counter := &blobLineCounter{last: '\n'}
	n, readErr := io.Copy(counter, io.LimitReader(stdout, limit))
	if readErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 0, false, readErr
	}
	truncated := false
	if n == limit {
		var probe [1]byte
		readN, probeErr := stdout.Read(probe[:])
		if readN > 0 {
			truncated = true
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		} else if probeErr != nil && probeErr != io.EOF {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return 0, false, probeErr
		} else if probeErr == io.EOF {
			if waitErr := cmd.Wait(); waitErr != nil {
				return 0, false, waitErr
			}
		}
	} else if waitErr := cmd.Wait(); waitErr != nil {
		return 0, false, waitErr
	}
	if counter.seen && counter.last != '\n' {
		counter.lines++
	}
	return counter.lines, truncated, nil
}

type blobLineCounter struct {
	lines int
	last  byte
	seen  bool
}

func (c *blobLineCounter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		for _, b := range p {
			if b == '\n' {
				c.lines++
			}
		}
		c.last = p[len(p)-1]
		c.seen = true
	}
	return len(p), nil
}
