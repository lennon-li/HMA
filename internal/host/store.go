package host

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Store struct{ root string }

func NewStore(root string) *Store { return &Store{root: root} }

func safeRunID(id string) bool {
	return id != "" && filepath.Base(id) == id && id != "." && id != ".."
}

func (s *Store) runDir(runID string) string { return filepath.Join(s.root, "host", runID) }

func recordDigest(r Record) (string, error) {
	r.HeadAnchor = ""
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func decodeRecord(path string) (Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return Record{}, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var r Record
	if err := dec.Decode(&r); err != nil {
		return Record{}, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return Record{}, err
		}
		return Record{}, errors.New("multiple host record documents")
	}
	return r, nil
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".hma-host-*")
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

func (s *Store) Load(runID string) ([]Record, error) {
	if !safeRunID(runID) {
		return nil, errors.New("invalid run id")
	}
	dir := s.runDir(runID)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	out := make([]Record, 0, len(names))
	prev := ""
	for i, name := range names {
		r, err := decodeRecord(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ReasonHostChainCorrupt, err)
		}
		if r.RunID != runID || r.Sequence != int64(i+1) || r.PredecessorHash != prev {
			return nil, errors.New(ReasonHostChainCorrupt)
		}
		want, err := recordDigest(r)
		if err != nil || want != r.HeadAnchor {
			return nil, errors.New(ReasonHostChainCorrupt)
		}
		if err := ValidateRecord(&r, true); err != nil {
			return nil, fmt.Errorf("%s: %w", ReasonHostChainCorrupt, err)
		}
		prev = r.HeadAnchor
		out = append(out, r)
	}
	if len(out) == 0 {
		return out, nil
	}
	headPath := filepath.Join(dir, "head")
	headBytes, err := os.ReadFile(headPath)
	if err != nil {
		return nil, errors.New(ReasonHostHeadMismatch)
	}
	if strings.TrimSpace(string(headBytes)) != prev {
		return nil, errors.New(ReasonHostHeadMismatch)
	}
	return out, nil
}

func (s *Store) Append(r *Record) error {
	if r == nil || !safeRunID(r.RunID) {
		return errors.New(ReasonInvalidHostRecord)
	}
	existing, err := s.Load(r.RunID)
	if err != nil {
		return err
	}
	r.Sequence = int64(len(existing) + 1)
	r.PredecessorHash = ""
	if len(existing) > 0 {
		r.PredecessorHash = existing[len(existing)-1].HeadAnchor
	}
	if err := ValidateRecord(r, false); err != nil {
		return err
	}
	r.HeadAnchor, err = recordDigest(*r)
	if err != nil {
		return err
	}
	if err := ValidateRecord(r, true); err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	b = append(bytes.TrimSpace(b), '\n')

	dir := s.runDir(r.RunID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("%020d.json", r.Sequence))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return errors.New(ReasonConcurrentWrite)
	}
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(dir, "head"), []byte(r.HeadAnchor+"\n")); err != nil {
		return err
	}
	return nil
}
