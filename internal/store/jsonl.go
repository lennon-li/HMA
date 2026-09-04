// Package store implements the host-trusted, detection-based JSONL chain.
//
// Two properties are deliberate. First, every Append rewrites the whole record
// stream through one atomic rename: a partial line can therefore never reach
// the committed file, at the cost of O(n) work per append. Second, the stream
// and its head anchor are two files, so they are committed in two phases with
// a durably recorded intent, and an interrupted commit is completed or
// discarded on the next load instead of leaving the run permanently unreadable.
//
// The store is host-trusted. The head anchor detects accidental truncation,
// rollback, and corruption; it is not a tamper-resistance claim against a
// same-UID adversary.
package store

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lennon-li/HMA/internal/model"
)

type Store struct{ dir string }

func New(dir string) *Store    { return &Store{dir: dir} }
func safeRunID(id string) bool { return id != "" && filepath.Base(id) == id && id != "." && id != ".." }

type runPaths struct{ stream, head, pending, lock string }

func (s *Store) paths(id string) runPaths {
	base := filepath.Join(s.dir, id)
	return runPaths{stream: base + ".jsonl", head: base + ".head", pending: base + ".head.pending", lock: base + ".lock"}
}

func digestRecord(r model.Record) (string, error) {
	r.Metadata.HeadAnchor = ""
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Load returns the committed chain for runID, completing or discarding an
// interrupted commit first.
func (s *Store) Load(runID string) ([]model.Record, error) {
	if !safeRunID(runID) {
		return nil, errors.New("invalid run id")
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return nil, err
	}
	lock, err := lockRun(s.paths(runID).lock)
	if err != nil {
		return nil, err
	}
	defer unlockRun(lock)
	return s.loadLocked(runID)
}

// readChain parses and verifies the record stream, returning the records and
// the chain tail anchor. It does not consult the head anchor file.
func readChain(runID, streamPath string) ([]model.Record, string, error) {
	b, err := os.ReadFile(streamPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		return nil, "", errors.New("torn final line")
	}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	scanner.Buffer(make([]byte, 4096), 4<<20)
	var out []model.Record
	prev := ""
	for scanner.Scan() {
		var r model.Record
		dec := json.NewDecoder(strings.NewReader(scanner.Text()))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&r); err != nil {
			return nil, "", fmt.Errorf("invalid record: %w", err)
		}
		if r.Metadata.RunID != runID || r.Metadata.Sequence != int64(len(out)+1) || r.Metadata.PredecessorHash != prev {
			return nil, "", errors.New("record chain discontinuity")
		}
		want, err := digestRecord(r)
		if err != nil || r.Metadata.HeadAnchor != want {
			return nil, "", errors.New("record head mismatch")
		}
		if err := model.ValidateRecord(&r); err != nil {
			return nil, "", fmt.Errorf("invalid record: %w", err)
		}
		prev = r.Metadata.HeadAnchor
		out = append(out, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return out, prev, nil
}

// recoverPending completes or discards an interrupted two-phase commit. The
// pending anchor is written before the stream, so it is honoured only when it
// already matches the committed chain tail; otherwise the stream never landed
// and the recorded intent is discarded. Recovery never invents, reorders, or
// accepts a record that is not already in the verified chain.
func recoverPending(p runPaths, chainTail string) error {
	pb, err := os.ReadFile(p.pending)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(pb)) == chainTail && chainTail != "" {
		if err := atomicWrite(p.head, []byte(chainTail+"\n")); err != nil {
			return err
		}
	}
	return os.Remove(p.pending)
}

func (s *Store) loadLocked(runID string) ([]model.Record, error) {
	p := s.paths(runID)
	out, chainTail, err := readChain(runID, p.stream)
	if err != nil {
		return nil, err
	}
	if err := recoverPending(p, chainTail); err != nil {
		return nil, err
	}
	if out == nil {
		if _, headErr := os.Stat(p.head); headErr == nil || !errors.Is(headErr, os.ErrNotExist) {
			return nil, errors.New("host head exists without record stream")
		}
		return nil, nil
	}
	hb, err := os.ReadFile(p.head)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(hb)) != chainTail {
		return nil, errors.New("host head mismatch")
	}
	return out, nil
}

// syncDir flushes a directory entry so a completed rename survives a crash.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	if closeErr := d.Close(); err == nil {
		err = closeErr
	}
	return err
}

func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".hma-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

// Append commits one record under an exclusive per-run lock.
func (s *Store) Append(r *model.Record) error {
	if r == nil || !safeRunID(r.Metadata.RunID) {
		return errors.New("invalid record")
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	p := s.paths(r.Metadata.RunID)
	lock, err := lockRun(p.lock)
	if err != nil {
		return err
	}
	defer unlockRun(lock)

	existing, err := s.loadLocked(r.Metadata.RunID)
	if err != nil {
		return err
	}
	wantSeq := int64(len(existing) + 1)
	prev := ""
	if len(existing) > 0 {
		prev = existing[len(existing)-1].Metadata.HeadAnchor
	}
	if r.Metadata.Sequence != wantSeq || r.Metadata.PredecessorHash != prev {
		return errors.New("record chain discontinuity")
	}
	r.Metadata.HeadAnchor, err = digestRecord(*r)
	if err != nil {
		return err
	}
	if err := model.ValidateRecord(r); err != nil {
		return err
	}
	var buf bytes.Buffer
	for _, old := range existing {
		b, err := json.Marshal(old)
		if err != nil {
			return err
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	buf.Write(b)
	buf.WriteByte('\n')

	// Phase 1: record the intended anchor before the stream changes, so an
	// interrupted commit is distinguishable from a stream that never landed.
	if err := atomicWrite(p.pending, []byte(r.Metadata.HeadAnchor+"\n")); err != nil {
		return err
	}
	// Phase 2: commit the stream, then promote the anchor.
	if err := atomicWrite(p.stream, buf.Bytes()); err != nil {
		return err
	}
	if err := atomicWrite(p.head, []byte(r.Metadata.HeadAnchor+"\n")); err != nil {
		return err
	}
	return os.Remove(p.pending)
}
