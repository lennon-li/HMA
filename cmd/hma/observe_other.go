//go:build !linux

package main

import "errors"

// errDirtyObservationUnsupported: reading the worktree of a dirty target safely
// needs descriptor-anchored lookups, which this build has only on Linux (where
// credit and the sandbox run). The observation of committed targets, which reads
// git objects, is unaffected.
var errDirtyObservationUnsupported = errors.New("dirty target observation requires Linux")

func openWorktreeEntry(root, rel string) (worktreeEntry, error) {
	return worktreeEntry{}, errDirtyObservationUnsupported
}
