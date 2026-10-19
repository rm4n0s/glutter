package semantics_test

import (
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}

func panicOf(call func()) (v any) {
	defer func() { v = recover() }()
	call()
	return nil
}

// requireTag checks that the call panics with an error whose route ends with
// the tag.
func requireTag(t *testing.T, tag string, call func()) *errors.Error {
	t.Helper()
	v := panicOf(call)
	if v == nil {
		t.Fatalf("expected a panic with tag %s", tag)
	}
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
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

func newOwner() *semantics.SemanticsOwner {
	return semantics.NewSemanticsOwner(func(*ui.SemanticsUpdate) {})
}

func rect(l, t, r, b float64) ui.Rect { return ui.NewRectFromLTRB(l, t, r, b) }

func nodeWithRect(r ui.Rect) *semantics.SemanticsNode {
	n := semantics.NewSemanticsNode()
	n.SetRect(r)
	return n
}

func boundaryConfig(merge bool) *semantics.SemanticsConfiguration {
	c := semantics.NewSemanticsConfiguration()
	c.SetIsSemanticBoundary(true)
	c.SetIsMergingSemanticsOfDescendants(merge)
	return c
}

func noop() {}
