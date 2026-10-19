package gluttertest

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// fakeTB is a TB that records how a test fails instead of failing: Fatal ends
// the function it runs in, as testing.T does, by panicking with fatalStop.
type fakeTB struct {
	errorfs  []string
	fatals   []any
	cleanups []func()
}

type fatalStop struct{}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Errorf(format string, args ...any) {
	f.errorfs = append(f.errorfs, fmt.Sprintf(format, args...))
}
func (f *fakeTB) Fatalf(format string, args ...any) {
	f.errorfs = append(f.errorfs, fmt.Sprintf(format, args...))
	panic(fatalStop{})
}
func (f *fakeTB) Fatal(args ...any) {
	f.fatals = append(f.fatals, args...)
	panic(fatalStop{})
}
func (f *fakeTB) Logf(string, ...any) {}
func (f *fakeTB) Cleanup(fn func())   { f.cleanups = append(f.cleanups, fn) }

// runCleanups runs the cleanups last first, as testing.T does.
func (f *fakeTB) runCleanups() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
	f.cleanups = nil
}

// failure runs fn with a tester on a fake TB and returns the error that made fn
// call Fatal; it fails the real test when fn did not fail.
func failure(t *testing.T, fn func(ts *Tester)) *errors.Error {
	t.Helper()
	tb := &fakeTB{}
	ts := New(tb)
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(fatalStop); !ok {
					panic(r)
				}
			}
		}()
		fn(ts)
	}()
	if len(tb.fatals) == 0 {
		t.Fatalf("expected the test under test to fail; it did not")
	}
	err, ok := tb.fatals[0].(error)
	if !ok {
		t.Fatalf("Fatal got %T, expected an error", tb.fatals[0])
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	return appErr
}

// requireRoute fails when the failure does not go through the route.
func requireRoute(t *testing.T, appErr *errors.Error, route string) {
	t.Helper()
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s (want it to contain %s)", appErr.Route(), route)
	}
}

// swatch is a leaf render object widget: a colored box of a fixed size. The
// optional fields make it stand for the widgets of later steps: Text, Icon,
// Tooltip and Semantics.
type swatch struct {
	widgets.LeafRenderObjectWidget
	color         ui.Color
	width, height float64
	text          string
	icon          string
	tooltip       string
	label         string
}

func newSwatch(c ui.Color, w, h float64) *swatch {
	return &swatch{LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(), color: c, width: w, height: h}
}

func (s *swatch) keyed(k foundation.Key) *swatch {
	c := *s
	c.LeafRenderObjectWidget = widgets.NewLeafRenderObjectWidgetBase(widgets.WidgetOpts{Key: k})
	return &c
}

func (s *swatch) PlainText() string      { return s.text }
func (s *swatch) IconValue() any         { return s.icon }
func (s *swatch) TooltipMessage() string { return s.tooltip }

func (s *swatch) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	var box rendering.Box = rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(s.color)}),
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: s.width, Height: s.height}), nil))
	if s.label != "" {
		box = rendering.NewRenderSemanticsAnnotations(box, &semantics.SemanticsProperties{Label: foundation.Some(s.label)},
			rendering.SemanticsAnnotationsOpts{TextDirection: ui.TextDirectionLTR})
	}
	return box
}

func (s *swatch) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {}

// row is a horizontal flex of its children, from the left edge.
type row struct {
	widgets.MultiChildRenderObjectWidget
}

func newRow(children ...widgets.Widget) *row {
	return &row{widgets.NewMultiChildRenderObjectWidgetBase(children)}
}

func (r *row) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return rendering.NewRenderFlex(nil, rendering.RenderFlexOpts{Direction: painting.AxisHorizontal,
		TextDirection: ui.TextDirectionLTR, CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
}

// hidden stands for Offstage: it is offstage and lays out its child.
type hidden struct {
	widgets.SingleChildRenderObjectWidget
}

func newHidden(child widgets.Widget) *hidden {
	return &hidden{widgets.NewSingleChildRenderObjectWidgetBase(child)}
}

func (h *hidden) IsOffstage() bool { return true }
func (h *hidden) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return rendering.NewRenderOffstage(nil)
}

// listener reports the pointer events that reach its box.
type listener struct {
	widgets.SingleChildRenderObjectWidget
	onDown   func(*gestures.PointerDownEvent)
	onMove   func(*gestures.PointerMoveEvent)
	onUp     func(*gestures.PointerUpEvent)
	onCancel func(*gestures.PointerCancelEvent)
	onHover  func(*gestures.PointerHoverEvent)
	onSignal func(gestures.PointerSignalEvent)
}

func newListener(child widgets.Widget) *listener {
	return &listener{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child)}
}

func (l *listener) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := rendering.NewRenderPointerListener(nil, rendering.RenderPointerListenerOpts{Behavior: rendering.HitTestBehaviorOpaque})
	l.UpdateRenderObject(nil, r)
	return r
}

func (l *listener) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*rendering.RenderPointerListener)
	r.OnPointerDown, r.OnPointerMove, r.OnPointerUp = nil, nil, nil
	if l.onDown != nil {
		r.OnPointerDown = l.onDown
	}
	if l.onMove != nil {
		r.OnPointerMove = l.onMove
	}
	if l.onUp != nil {
		r.OnPointerUp = l.onUp
	}
	if l.onCancel != nil {
		r.OnPointerCancel = l.onCancel
	}
	if l.onHover != nil {
		r.OnPointerHover = l.onHover
	}
	if l.onSignal != nil {
		r.OnPointerSignal = l.onSignal
	}
}

// counter is a stateful widget that counts the taps on its box, with a tap
// recognizer; it stands for the counter app.
type counter struct {
	widgets.StatefulWidget
}

func newCounter() *counter { return &counter{widgets.NewStatefulWidgetBase()} }

func (c *counter) CreateState() widgets.State { return &counterState{} }

type counterState struct {
	widgets.StateOf[*counter]
	taps int
	rec  *gestures.TapGestureRecognizer
}

func (s *counterState) InitState() {
	s.rec = gestures.NewTapGestureRecognizer(widgets.WidgetsBindingOf(s.Context()).Gestures())
	s.rec.OnTap = func() { s.SetState(func() { s.taps++ }) }
}

func (s *counterState) Dispose() { s.rec.Dispose() }

func (s *counterState) Build(widgets.BuildContext) widgets.Widget {
	l := newListener(newSwatch(ui.NewColor(0xFF6750A4), 100, 50))
	l.onDown = func(e *gestures.PointerDownEvent) { s.rec.AddPointer(e) }
	children := []widgets.Widget{l}
	for i := 0; i < s.taps; i++ {
		children = append(children, newSwatch(ui.NewColor(0xFFFFB74D), 20, 20))
	}
	return newRow(children...)
}

// builder stands for Builder: a stateless widget that builds with a function.
type builder struct {
	widgets.StatelessWidget
	fn func(bc widgets.BuildContext) widgets.Widget
}

func newBuilder(fn func(bc widgets.BuildContext) widgets.Widget) *builder {
	return &builder{widgets.NewStatelessWidgetBase(), fn}
}

func (b *builder) Build(bc widgets.BuildContext) widgets.Widget { return b.fn(bc) }

// requireTagged returns the tagged error, failing when err is nil or not tagged.
func requireTagged(t *testing.T, err error) *errors.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	return appErr
}

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only semantics nodes of the render objects")
	}
}
