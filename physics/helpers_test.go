package physics

import (
	"github.com/rm4n0s/glutter/foundation"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
)

// closeTo is flutter_test's moreOrLessEquals: a is within epsilon of b.
func closeTo(a, b, epsilon float64) bool {
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return a == b
	}
	return math.Abs(a-b) <= epsilon
}

// precisionErrorTolerance is foundation.PrecisionErrorTolerance, which flutter_test's
// moreOrLessEquals uses as its default epsilon.
const defaultEpsilon = 1e-10

// check fails the test when got is not within epsilon of want.
func check(t *testing.T, name string, got, want, epsilon float64) {
	t.Helper()
	if !closeTo(got, want, epsilon) {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, epsilon)
	}
}

// skipUnlessDebug skips a test of the checks that only debug builds make.
func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}

// recoverError runs fn and returns the error it panicked with.
func recoverError(t *testing.T, fn func()) (err error) {
	t.Helper()
	skipUnlessDebug(t) // the panics of this package are assertions of debug builds
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

// assertRoute checks that err is a tagged error whose route contains route.
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

// testSimulation is the TestSimulation of to_string_test.dart.
type testSimulation struct{ SimulationBase }

func (*testSimulation) X(float64) float64   { return 0 }
func (*testSimulation) Dx(float64) float64  { return 0 }
func (*testSimulation) IsDone(float64) bool { return true }
