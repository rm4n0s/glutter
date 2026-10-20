package mcu

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

func some[T any](v T) foundation.Opt[T] { return foundation.Some(v) }

func asError(r any) (*errors.Error, bool) {
	err, ok := r.(error)
	if !ok {
		return nil, false
	}
	return errors.FromError(err)
}

// skipUnlessDebug skips a test of an assertion that release builds do not make.
func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("assertions are checked in debug builds only")
	}
}
