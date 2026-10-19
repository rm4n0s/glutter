//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// requireTaggedPanic checks that call panics with an error that has the tag and
// the route.
func requireTaggedPanic(t *testing.T, tag string, call func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic with tag %s", tag)
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic value is %T, want an error", r)
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T", err)
		}
		if !appErr.HasRoute(tag) {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	}()
	call()
}
