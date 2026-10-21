//go:build !linux

package repostate

import (
	"context"
	"errors"
)

// Fingerprint is unavailable without Unix file metadata; callers that need it
// must refuse.
func Fingerprint(ctx context.Context, root string) (string, error) {
	return "", errors.New("file metadata fingerprint is not supported on this platform")
}
