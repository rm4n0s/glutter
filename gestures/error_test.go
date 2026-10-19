package gestures

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

type reports []error

func (r *reports) report(err error) { *r = append(*r, err) }

func (r reports) checkOne(t *testing.T, context string) *errors.Error {
	t.Helper()
	if len(r) != 1 {
		t.Fatalf("%d errors reported, want 1: %v", len(r), []error(r))
	}
	appErr, ok := errors.FromError(r[0])
	if !ok || !appErr.HasRoute(foundation.CallbackPanicked) {
		t.Fatalf("error = %v, want CallbackPanicked", r[0])
	}
	if got, _ := appErr.Metadata["context"].(string); !strings.Contains(got, context) {
		t.Errorf("context = %q, want it to contain %q", got, context)
	}
	return appErr
}

func TestPanicInARecognizerCallbackIsReported(t *testing.T) {
	g := newGestureTester(t)
	var errs reports
	g.binding.SetErrorReporter(errs.report)
	e := newTapEvents()
	tap := NewTapGestureRecognizer(g.binding)
	var order []string
	tap.OnTapDown = func(TapDownDetails) { order = append(order, "down"); panic("bad callback") }
	tap.OnTap = func() { order = append(order, "tap") }
	tap.AddPointer(e.down1)
	g.closeArena(1)
	g.route(e.down1)
	g.route(e.up1)
	g.binding.GestureArena.Sweep(1)
	// The panic of onTapDown did not stop the recognizer: the tap was recognized.
	if !slices.Equal(order, []string{"down", "tap"}) {
		t.Errorf("order = %v, want the tap to follow the failed tap down", order)
	}
	appErr := errs.checkOne(t, "while handling a gesture")
	if appErr.Metadata["callback"] != "onTapDown" || appErr.Metadata["panic"] != "bad callback" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
	tap.Dispose()
}

func TestPanicInARecognizerCallbackWithoutAReporterGoesOn(t *testing.T) {
	g := newGestureTester(t)
	e := newTapEvents()
	tap := NewTapGestureRecognizer(g.binding)
	tap.OnTapDown = func(TapDownDetails) { panic("bad callback") }
	tap.AddPointer(e.down1)
	g.closeArena(1)
	defer func() {
		if r := recover(); r != "bad callback" {
			t.Errorf("recovered %v, want the panic of the callback", r)
		}
	}()
	g.route(e.down1)
}

func TestInvokeCallbackResultReturnsTheZeroValueAfterAPanic(t *testing.T) {
	g := newGestureTester(t)
	var errs reports
	g.binding.SetErrorReporter(errs.report)
	tap := NewTapGestureRecognizer(g.binding)
	if got := InvokeCallbackResult(&tap.GestureRecognizerBase, "onFoo", func() int { panic("bad") }); got != 0 {
		t.Errorf("result = %d, want 0", got)
	}
	if got := InvokeCallbackResult(&tap.GestureRecognizerBase, "onFoo", func() int { return 7 }); got != 7 {
		t.Errorf("result = %d, want 7", got)
	}
	_ = errs.checkOne(t, "while handling a gesture")
}

func TestPanicInARouteIsReportedAndTheNextRouteRuns(t *testing.T) {
	g := newGestureTester(t)
	var errs reports
	g.binding.SetErrorReporter(errs.report)
	var calls []string
	g.binding.PointerRouter.AddRoute(1, func(PointerEvent) { calls = append(calls, "first"); panic("bad route") }, nil)
	g.binding.PointerRouter.AddRoute(1, func(PointerEvent) { calls = append(calls, "second") }, nil)
	g.binding.PointerRouter.AddGlobalRoute(func(PointerEvent) { calls = append(calls, "global") }, nil)
	p := newTestPointer(1)
	g.binding.PointerRouter.Route(p.down(ui.Offset{}))
	if !slices.Equal(calls, []string{"first", "second", "global"}) {
		t.Errorf("calls = %v, want every route to run", calls)
	}
	_ = errs.checkOne(t, "while routing a pointer event")
}

type panickingTarget struct{ calls *[]string }

func (p panickingTarget) HandleEvent(PointerEvent, HitTestEntry) {
	*p.calls = append(*p.calls, "panicking")
	panic("bad target")
}

type okTarget struct{ calls *[]string }

func (p okTarget) HandleEvent(PointerEvent, HitTestEntry) { *p.calls = append(*p.calls, "ok") }

func TestPanicInAHitTestTargetIsReportedAndTheNextTargetGetsTheEvent(t *testing.T) {
	g := newGestureTester(t)
	var errs reports
	g.binding.SetErrorReporter(errs.report)
	var calls []string
	result := NewHitTestResult()
	result.Add(NewHitTestEntry(panickingTarget{&calls}))
	result.Add(NewHitTestEntry(okTarget{&calls}))
	g.binding.DispatchEvent(newTestPointer(1).down(ui.Offset{}), result)
	if !slices.Equal(calls, []string{"panicking", "ok"}) {
		t.Errorf("calls = %v", calls)
	}
	appErr := errs.checkOne(t, "while dispatching a pointer event")
	if _, ok := appErr.Metadata["event"]; !ok {
		t.Errorf("metadata = %v, want the event", appErr.Metadata)
	}
}

func TestPanicWhileRoutingAnEventWithoutAHitTest(t *testing.T) {
	g := newGestureTester(t)
	var errs reports
	g.binding.SetErrorReporter(errs.report)
	g.binding.PointerRouter.AddGlobalRoute(func(PointerEvent) { panic("bad route") }, nil)
	g.binding.DispatchEvent(NewPointerAddedEvent(PointerEventOpts{Pointer: 1}), nil)
	_ = errs.checkOne(t, "while routing a pointer event")
}

func TestPanicInAPointerSignalCallbackIsReported(t *testing.T) {
	g := newGestureTester(t)
	var errs reports
	g.binding.SetErrorReporter(errs.report)
	ev := NewPointerScrollEvent(PointerEventOpts{Pointer: 1})
	g.binding.PointerSignalResolver.Register(ev, func(PointerSignalEvent) { panic("bad signal") })
	g.binding.PointerSignalResolver.Resolve(ev)
	_ = errs.checkOne(t, "while resolving a PointerSignalEvent")
	// The resolver is ready for the next event.
	called := false
	g.binding.PointerSignalResolver.Register(ev, func(PointerSignalEvent) { called = true })
	g.binding.PointerSignalResolver.Resolve(ev)
	if !called {
		t.Error("the resolver was left with the old callback")
	}
}
