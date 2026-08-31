// Package store implements the host-trusted, detection-based JSONL chain.
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
func (s *Store) paths(id string) (string, string) {
	return filepath.Join(s.dir, id+".jsonl"), filepath.Join(s.dir, id+".head")
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

func (s *Store) Load(runID string) ([]model.Record, error) {
	if !safeRunID(runID) {
		return nil, errors.New("invalid run id")
	}
	stream, head := s.paths(runID)
	b, err := os.ReadFile(stream)
	if errors.Is(err, os.ErrNotExist) {
		if _, headErr := os.Stat(head); headErr == nil || !errors.Is(headErr, os.ErrNotExist) {
			return nil, errors.New("host head exists without record stream")
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		return nil, errors.New("torn final line")
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
			return nil, fmt.Errorf("invalid record: %w", err)
		}
		if r.Metadata.RunID != runID || r.Metadata.Sequence != int64(len(out)+1) || r.Metadata.PredecessorHash != prev {
			return nil, errors.New("record chain discontinuity")
		}
		want, err := digestRecord(r)
		if err != nil || r.Metadata.HeadAnchor != want {
			return nil, errors.New("record head mismatch")
		}
		if err := model.ValidateRecord(&r); err != nil {
			return nil, fmt.Errorf("invalid record: %w", err)
		}
		prev = r.Metadata.HeadAnchor
		out = append(out, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	hb, err := os.ReadFile(head)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(hb)) != prev {
		return nil, errors.New("host head mismatch")
	}
	return out, nil
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
	return os.Rename(name, path)
}

func (s *Store) Append(r *model.Record) error {
	if r == nil || !safeRunID(r.Metadata.RunID) {
		return errors.New("invalid record")
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	existing, err := s.Load(r.Metadata.RunID)
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
	stream, head := s.paths(r.Metadata.RunID)
	var buf bytes.Buffer
	for _, old := range existing {
		b, _ := json.Marshal(old)
		buf.Write(b)
		buf.WriteByte('\n')
	}
	b, _ := json.Marshal(r)
	buf.Write(b)
	buf.WriteByte('\n')
	if err := atomicWrite(stream, buf.Bytes()); err != nil {
		return err
	}
	return atomicWrite(head, []byte(r.Metadata.HeadAnchor+"\n"))
}
