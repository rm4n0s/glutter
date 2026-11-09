package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
)

// skipUnlessDebug skips a test that needs the debug-only checks of the framework.
func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}

// requireErrorTag fails unless err is an *errors.Error whose route ends in tag.
func requireErrorTag(t *testing.T, err error, tag string) *errors.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error with tag %s, got none", tag)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(tag) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tag)
	}
	return appErr
}

// firstError is the first error that the framework reported while the tester ran.
func firstError(t *testing.T, ts *gluttertest.Tester) error {
	t.Helper()
	errs := ts.TakeErrors()
	if len(errs) == 0 {
		t.Fatal("the framework reported no error")
	}
	return errs[0]
}

// colorOf makes the colors of the tests: 0xFF0000NN.
func colorOf(n uint32) ui.Color { return ui.NewColor(0xFF000000 | n) }

// asDynamic is the CupertinoDynamicColor in a ui.ColorValue.
func asDynamic(t *testing.T, c ui.ColorValue) cupertino.CupertinoDynamicColor {
	t.Helper()
	d, ok := c.(cupertino.CupertinoDynamicColor)
	if !ok {
		t.Fatalf("%T is not a CupertinoDynamicColor", c)
	}
	return d
}

// requireErrorTagFromPanic checks the value of a recovered panic like requireErrorTag.
func requireErrorTagFromPanic(t *testing.T, v any, tag string) *errors.Error {
	t.Helper()
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	return requireErrorTag(t, err, tag)
}
