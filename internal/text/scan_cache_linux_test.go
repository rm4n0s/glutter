//go:build linux

package text

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// TestScanMatcherWithoutACacheDir checks that a machine with no cache directory
// (neither $XDG_CACHE_HOME nor $HOME) fails with FontScanFailed, where
// os.UserCacheDir fails.
func TestScanMatcherWithoutACacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	_, err := newScanMatcher("", nil)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("newScanMatcher->defaultCacheDir.FontScanFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
