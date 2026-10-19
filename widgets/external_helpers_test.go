package widgets_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The tests of this package that use only the public API run on gluttertest, as
// an app's tests do; this file holds the small widgets they build (Flutter's
// test_widgets.dart).

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

// box is a leaf render object widget with a tight size.
type box struct {
	LeafRenderObjectWidget
	width, height float64
}

func newBox(w, h float64, opts ...WidgetOpts) *box {
	return &box{LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(opts...), width: w, height: h}
}

func (w *box) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: w.width, Height: w.height}), nil)
}

func (w *box) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	ro.(*rendering.RenderConstrainedBox).SetAdditionalConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: w.width, Height: w.height}))
}

// newLeaf is a widget that the test does not care about: a box of one pixel.
func newLeaf(opts ...WidgetOpts) *box { return newBox(1, 1, opts...) }

// builder is a stateless widget that builds with a function (Builder).
type builder struct {
	StatelessWidget
	fn func(bc BuildContext) Widget
}

func builderOf(fn func(bc BuildContext) Widget) *builder {
	return &builder{NewStatelessWidgetBase(), fn}
}

func (b *builder) Build(bc BuildContext) Widget { return b.fn(bc) }

func panicOf(call func()) (v any) {
	defer func() { v = recover() }()
	call()
	return nil
}

// requireTag runs call and fails unless it panics with an *errors.Error whose
// route ends in tag.
func requireTag(t *testing.T, tag string, call func()) {
	t.Helper()
	v := panicOf(call)
	if v == nil {
		t.Fatalf("expected a panic with tag %s, got none", tag)
	}
	_ = requireErrorTagFromPanic(t, v, tag)
}

// requireErrorTagFromPanic is requireErrorTag for a value that was already
// recovered.
func requireErrorTagFromPanic(t *testing.T, v any, tag string) *errors.Error {
	t.Helper()
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	return requireErrorTag(t, err, tag)
}

// takeFirst takes every error that the framework reported and returns the first
// one; the test fails when there is none.
func takeFirst(t *testing.T, ts *gluttertest.Tester) error {
	t.Helper()
	errs := ts.TakeErrors()
	if len(errs) == 0 {
		t.Fatalf("the framework reported no error")
	}
	return errs[0]
}

// stateful is a stateful widget that builds with a function that gets SetState
// (Flutter's StatefulBuilder).
type stateful struct {
	StatefulWidget
	build func(bc BuildContext, setState func(func())) Widget
}

func newBuilder(build func(bc BuildContext, setState func(func())) Widget, opts ...WidgetOpts) *stateful {
	return &stateful{NewStatefulWidgetBase(opts...), build}
}

func (w *stateful) CreateState() State { return &statefulState{} }

type statefulState struct{ StateOf[*stateful] }

func (s *statefulState) Build(bc BuildContext) Widget { return s.Widget().build(bc, s.SetState) }

// multi is a widget with a list of children: a vertical flex.
type multi struct {
	MultiChildRenderObjectWidget
}

func newMulti(children ...Widget) *multi {
	return &multi{NewMultiChildRenderObjectWidgetBase(children)}
}

func (w *multi) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderFlex(nil, rendering.RenderFlexOpts{Direction: painting.AxisVertical, TextDirection: ui.TextDirectionLTR})
}

// pumpRaw makes the widget the root of the app in a RawView of the view of the
// tester: it has a render tree and a View, but no MediaQuery, as a tree of an app
// that does not use the default View has.
func pumpRaw(ts *gluttertest.Tester, w Widget) {
	ts.PumpWidget(NewRawView(ts.View, w), gluttertest.PumpWidgetOpts{NoView: true})
}

// pumpUnwrapped makes the widget the root of the app without the default View.
func pumpUnwrapped(ts *gluttertest.Tester, w Widget) {
	ts.PumpWidget(w, gluttertest.PumpWidgetOpts{NoView: true})
}

// nothing is a widget with no children and no render object: a tree that ends
// there shows nothing.
type nothing struct{ WidgetBase }

func newNothing() *nothing { return &nothing{NewWidgetBase()} }

type nothingElement struct{ ElementBase }

func (w *nothing) CreateElement() Element {
	e := &nothingElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

func (e *nothingElement) PerformRebuild() {}
