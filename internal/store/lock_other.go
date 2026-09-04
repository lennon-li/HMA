//go:build !unix

package store

import (
	"errors"
	"os"
)

// lockRun fails loudly on platforms without advisory locking rather than
// silently downgrading to an unsynchronized rewrite that can lose records.
func lockRun(string) (*os.File, error) {
	return nil, errors.New("cross-process run locking is unsupported on this platform")
}

func unlockRun(*os.File) error { return nil }
