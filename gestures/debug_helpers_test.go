//go:build debug_mode

package gestures

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// recoverError returns the error a function panics with.
func recoverError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic")
		}
		e, ok := r.(error)
		if !ok {
			t.Fatalf("expected an error panic, got %T (%v)", r, r)
		}
		err = e
	}()
	fn()
	return nil
}

// assertRoute checks that err is a tagged error with the route.
func assertRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

// testGestureRecognizer is recognizer_test.dart's TestGestureRecognizer.
type testGestureRecognizer struct{ GestureRecognizerBase }

func (r *testGestureRecognizer) DebugDescription() string                    { return "debugDescription content" }
func (r *testGestureRecognizer) AddPointer(*PointerDownEvent)                {}
func (r *testGestureRecognizer) AcceptGesture(int)                           {}
func (r *testGestureRecognizer) RejectGesture(int)                           {}
func (r *testGestureRecognizer) AddPointerPanZoom(*PointerPanZoomStartEvent) {}
