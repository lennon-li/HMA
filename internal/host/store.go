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

	"github.com/lennon-li/HMA/internal/store"
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
	if err := os.Rename(name, path); err != nil {
		return err
	}
	// The rename is durable only once the directory entry is.
	return syncDir(filepath.Dir(path))
}

// runPaths are a run's host files. The lock and pending intent sit beside the
// run directory so the directory holds only records and the head anchor.
func (s *Store) lockPath(runID string) string { return s.runDir(runID) + ".lock" }
func (s *Store) pendingPath(runID string) string {
	return filepath.Join(s.runDir(runID), "head.pending")
}

// Load returns the committed host chain, completing or discarding an
// interrupted append first. Like the core store, loading a run that has no
// files creates nothing.
func (s *Store) Load(runID string) ([]Record, error) {
	if !safeRunID(runID) {
		return nil, errors.New("invalid run id")
	}
	if _, err := os.Stat(s.runDir(runID)); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	lock, err := store.LockFile(s.lockPath(runID))
	if err != nil {
		return nil, err
	}
	defer store.UnlockFile(lock)
	return s.loadLocked(runID)
}

// readChain parses and verifies the numbered records, returning them and the
// chain tail anchor. It does not consult the head anchor file.
func (s *Store) readChain(runID string) ([]Record, string, error) {
	dir := s.runDir(runID)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
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
			return nil, "", fmt.Errorf("%s: %w", ReasonHostChainCorrupt, err)
		}
		if r.RunID != runID || r.Sequence != int64(i+1) || r.PredecessorHash != prev || name != recordName(r.Sequence) {
			return nil, "", errors.New(ReasonHostChainCorrupt)
		}
		want, err := recordDigest(r)
		if err != nil || want != r.HeadAnchor {
			return nil, "", errors.New(ReasonHostChainCorrupt)
		}
		if err := ValidateRecord(&r, true); err != nil {
			return nil, "", fmt.Errorf("%s: %w", ReasonHostChainCorrupt, err)
		}
		prev = r.HeadAnchor
		out = append(out, r)
	}
	return out, prev, nil
}

func recordName(seq int64) string { return fmt.Sprintf("%020d.json", seq) }

// recoverPending completes or discards an interrupted append. The pending
// anchor is written before the record, so it is honoured only when it already
// equals the verified chain tail; otherwise the record never landed and the
// intent is discarded. Recovery never invents or accepts a record that is not
// already in the verified chain.
func (s *Store) recoverPending(runID, chainTail string) error {
	pb, err := os.ReadFile(s.pendingPath(runID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(pb)) == chainTail && chainTail != "" {
		if err := atomicWrite(filepath.Join(s.runDir(runID), "head"), []byte(chainTail+"\n")); err != nil {
			return err
		}
	}
	if err := os.Remove(s.pendingPath(runID)); err != nil {
		return err
	}
	return syncDir(s.runDir(runID))
}

func (s *Store) loadLocked(runID string) ([]Record, error) {
	out, tail, err := s.readChain(runID)
	if err != nil {
		return nil, err
	}
	if err := s.recoverPending(runID, tail); err != nil {
		return nil, err
	}
	headBytes, err := os.ReadFile(filepath.Join(s.runDir(runID), "head"))
	if errors.Is(err, os.ErrNotExist) && len(out) == 0 {
		return out, nil
	}
	if err != nil || strings.TrimSpace(string(headBytes)) != tail {
		return nil, errors.New(ReasonHostHeadMismatch)
	}
	return out, nil
}

// Append commits r as the next record under the run lock.
func (s *Store) Append(r *Record) error {
	if r == nil {
		return errors.New(ReasonInvalidHostRecord)
	}
	return s.AppendWith(r.RunID, func([]Record) (*Record, error) { return r, nil })
}

// AppendWith takes the run lock, loads the committed history, and asks build
// for the next record; build sees exactly the history the record will extend,
// so a check inside it cannot be invalidated by a concurrent append. The commit
// records the intended head anchor, creates the numbered record atomically,
// then moves the head, syncing the directory after each step. A crash at any
// point leaves a state Load recovers.
func (s *Store) AppendWith(runID string, build func(history []Record) (*Record, error)) error {
	return s.Transact(runID, func(history []Record, commit func(*Record) error) error {
		r, err := build(history)
		if err != nil {
			return err
		}
		return commit(r)
	})
}

// Transact takes the run lock, loads the committed history, and calls fn with
// it and a commit function that durably appends one record extending exactly
// that history. Everything fn holds while it calls commit (such as the core
// run lock) stays held until the record, head, and directory are durable.
// commit may be called at most once.
func (s *Store) Transact(runID string, fn func(history []Record, commit func(*Record) error) error) error {
	if !safeRunID(runID) {
		return errors.New(ReasonInvalidHostRecord)
	}
	dir := s.runDir(runID)
	if err := mkdirAllSynced(dir); err != nil {
		return err
	}
	return s.transactLocked(runID, fn)
}

// WithRunLock holds the host run lock while fn runs against the committed
// host history, so no host record can be appended until fn returns. Lock
// order across HMA is host run lock, then core run lock.
func (s *Store) WithRunLock(runID string, fn func(history []Record) error) error {
	if !safeRunID(runID) {
		return errors.New(ReasonInvalidHostRecord)
	}
	if err := mkdirAllSynced(filepath.Dir(s.runDir(runID))); err != nil {
		return err
	}
	lock, err := store.LockFile(s.lockPath(runID))
	if err != nil {
		return err
	}
	defer store.UnlockFile(lock)
	history, err := s.loadLocked(runID)
	if err != nil {
		return err
	}
	return fn(history)
}

func (s *Store) transactLocked(runID string, fn func(history []Record, commit func(*Record) error) error) error {
	dir := s.runDir(runID)
	// An ancestor created earlier by another process may never have been
	// synced into its own parent; sync every ancestor so an acknowledged
	// record's whole path is durable.
	if err := syncAncestors(dir); err != nil {
		return err
	}
	lock, err := store.LockFile(s.lockPath(runID))
	if err != nil {
		return err
	}
	defer store.UnlockFile(lock)
	existing, err := s.loadLocked(runID)
	if err != nil {
		return err
	}
	committed := false
	return fn(existing, func(r *Record) error {
		if committed {
			return errors.New(ReasonConcurrentWrite)
		}
		committed = true
		return s.commitLocked(runID, existing, r)
	})
}

// commitLocked durably appends r after existing. The caller holds the run lock.
func (s *Store) commitLocked(runID string, existing []Record, r *Record) error {
	dir := s.runDir(runID)
	var err error
	if r == nil || r.RunID != runID {
		return errors.New(ReasonInvalidHostRecord)
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

	if err := atomicWrite(s.pendingPath(runID), []byte(r.HeadAnchor+"\n")); err != nil {
		return err
	}
	path := filepath.Join(dir, recordName(r.Sequence))
	if err := atomicCreate(path, b); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New(ReasonConcurrentWrite)
		}
		return err
	}
	if err := atomicWrite(filepath.Join(dir, "head"), []byte(r.HeadAnchor+"\n")); err != nil {
		return err
	}
	if err := os.Remove(s.pendingPath(runID)); err != nil {
		return err
	}
	return syncDir(dir)
}

// mkdirAllSynced creates dir and any missing parents, syncing each parent
// after creating a child so a crash cannot lose a directory entry the store
// later relies on.
//
// The parent is synced even when dir already exists: a concurrent creator may
// not have synced it yet, and this caller must not acknowledge a record whose
// directory entry is not durable.
func mkdirAllSynced(dir string) error {
	fi, err := os.Stat(dir)
	if err == nil && !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	parent := filepath.Dir(dir)
	if err != nil {
		if parent != dir {
			if err := mkdirAllSynced(parent); err != nil {
				return err
			}
		}
		if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	if parent == dir {
		return nil
	}
	return syncDir(parent)
}

// syncAncestors syncs every directory whose entry is needed to reach dir, so
// an acknowledged record's whole path is durable. It resolves dir one
// component at a time and syncs each directory a component is looked up in:
// the lexical ancestors, the directory holding every symlink followed
// (including intermediate hops), and the ancestors of each link target. A
// relative path is made absolute first so the walk reaches the root.
func syncAncestors(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	dirs, real, err := lookupDirs(abs)
	if err != nil {
		return err
	}
	dirs[real] = true
	for d := range dirs {
		if err := syncDir(d); err != nil {
			return err
		}
	}
	return nil
}

// maxSymlinkHops bounds symlink resolution, as the kernel's ELOOP limit does.
const maxSymlinkHops = 40

// lookupDirs resolves the absolute path abs component by component and
// returns every directory a component was looked up in, plus the resolved
// path.
func lookupDirs(abs string) (map[string]bool, string, error) {
	dirs := map[string]bool{}
	rest := strings.Split(abs, string(filepath.Separator))
	cur := string(filepath.Separator)
	hops := 0
	for len(rest) > 0 {
		c := rest[0]
		rest = rest[1:]
		switch c {
		case "", ".":
			continue
		case "..":
			// cur is already resolved, so its lexical parent is its real parent.
			cur = filepath.Dir(cur)
			continue
		}
		dirs[cur] = true
		next := filepath.Join(cur, c)
		fi, err := os.Lstat(next)
		if err != nil {
			return nil, "", err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			cur = next
			continue
		}
		if hops++; hops > maxSymlinkHops {
			return nil, "", fmt.Errorf("too many symlinks resolving %s", abs)
		}
		target, err := os.Readlink(next)
		if err != nil {
			return nil, "", err
		}
		if filepath.IsAbs(target) {
			cur = string(filepath.Separator)
		}
		rest = append(strings.Split(target, string(filepath.Separator)), rest...)
	}
	return dirs, cur, nil
}

// atomicCreate writes data to a temporary file and links it to path, so path
// either does not exist or holds the complete record; it never overwrites.
func atomicCreate(path string, data []byte) error {
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
	if err := os.Link(name, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
