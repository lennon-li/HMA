//go:build unix

package store

import (
	"os"
	"syscall"
)

// lockRun takes an exclusive advisory lock for one run. Append rewrites the
// whole record stream, so two unsynchronized writers could each read the same
// predecessor state and each rewrite the file, silently discarding the other's
// record. The lock makes that interleaving impossible; a losing writer instead
// observes the committed chain and fails the sequence check.
func lockRun(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func unlockRun(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
