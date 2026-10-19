package gestures

import (
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/scheduler"
	"testing"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/ui"
)

// fakeHost is the engine of a test: it keeps the functions posted to the "UI
// thread" until the test runs them.
type fakeHost struct {
	posted []func()
}

func (h *fakeHost) ScheduleFrame()                        {}
func (h *fakeHost) RenderView(ui.ViewID, *ui.Scene) error { return nil }
func (h *fakeHost) RequestAppExit()                       {}
func (h *fakeHost) Post(fn func())                        { h.posted = append(h.posted, fn) }

// flush runs what was posted, including what that posts: flutter_test's
// FakeAsync.flushMicrotasks.
func (h *fakeHost) flush() {
	for len(h.posted) > 0 {
		fn := h.posted[0]
		h.posted = h.posted[1:]
		fn()
	}
}

// microtasks is a queue of functions for a GestureArenaManager on its own.
type microtasks struct{ queue []func() }

func (m *microtasks) schedule(fn func()) { m.queue = append(m.queue, fn) }

func (m *microtasks) flush() {
	for len(m.queue) > 0 {
		fn := m.queue[0]
		m.queue = m.queue[1:]
		fn()
	}
}

// testPointer generates the events of one pointer, like flutter_test's
// TestPointer.
type testPointer struct {
	pointer  int
	kind     ui.PointerDeviceKind
	device   int
	buttons  int
	isDown   bool
	location ui.Offset
	pan      ui.Offset
}

func newTestPointer(pointer ...int) *testPointer {
	p := &testPointer{pointer: 1, kind: ui.PointerDeviceKindTouch, buttons: PrimaryButton}
	if len(pointer) > 0 {
		p.pointer = pointer[0]
	}
	return p
}

func (p *testPointer) down(location ui.Offset) *PointerDownEvent {
	p.isDown = true
	p.location = location
	return NewPointerDownEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location,
		Buttons: someInt(p.buttons),
	})
}

func (p *testPointer) move(location ui.Offset) *PointerMoveEvent {
	delta := location.Sub(p.location)
	p.location = location
	return NewPointerMoveEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location, Delta: delta,
		Buttons: someInt(p.buttons),
	})
}

func (p *testPointer) up() *PointerUpEvent {
	p.isDown = false
	return NewPointerUpEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: p.location,
	})
}

func (p *testPointer) cancel() *PointerCancelEvent {
	p.isDown = false
	return NewPointerCancelEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: p.location,
	})
}

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

func us(n int) time.Duration { return time.Duration(n) * time.Microsecond }

type sampleClock struct{ *fakeclock.Fake }

func (c sampleClock) AfterFunc(d time.Duration, f func()) SamplingTimer {
	return c.Fake.AfterFunc(d, f)
}

type fakeScheduler struct {
	postFrameCallback scheduler.FrameCallback
	frameTime         time.Duration
}

func (*fakeScheduler) Locked() bool                                      { return false }
func (*fakeScheduler) AddUnlockedCallback(func())                        {}
func (s *fakeScheduler) AddPostFrameCallback(cb scheduler.FrameCallback) { s.postFrameCallback = cb }
func (s *fakeScheduler) CurrentSystemFrameTimeStamp() time.Duration      { return s.frameTime }

// gestureTester is flutter_test's GestureTester: a binding with a fake clock,
// whose microtasks run when the test says so.
type gestureTester struct {
	t       *testing.T
	host    *fakeHost
	clock   *fakeclock.Fake
	binding *GestureBinding
}

func newGestureTester(t *testing.T) *gestureTester {
	t.Helper()
	host := &fakeHost{}
	d := ui.NewPlatformDispatcher(host)
	d.AddView(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1})
	clock := fakeclock.New(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	b := NewGestureBinding(d, &fakeScheduler{}, GestureBindingOpts{Clock: sampleClock{clock}})
	return &gestureTester{t: t, host: host, clock: clock, binding: b}
}

func (g *gestureTester) closeArena(pointer int) { g.binding.GestureArena.Close(pointer) }

func (g *gestureTester) route(event PointerEvent) {
	g.binding.PointerRouter.Route(event)
	g.host.flush()
}

// elapse moves the clock and runs the timers that fire, then the microtasks.
func (g *gestureTester) elapse(d time.Duration) {
	g.clock.Advance(d)
	g.host.flush()
}

// flush runs the microtasks.
func (g *gestureTester) flush() { g.host.flush() }

// log records what happened, in order.
type log struct{ entries []string }

func (l *log) add(s string) { l.entries = append(l.entries, s) }
func (l *log) clear()       { l.entries = nil }
func (l *log) check(t *testing.T, want ...string) {
	t.Helper()
	if len(l.entries) != len(want) {
		t.Fatalf("log = %q, want %q", l.entries, want)
	}
	for i := range want {
		if l.entries[i] != want[i] {
			t.Fatalf("log = %q, want %q", l.entries, want)
		}
	}
}

// A step of a scripted gesture test.
type step func(g *gestureTester, l *log)

func route(e PointerEvent) step { return func(g *gestureTester, _ *log) { g.route(e) } }
func elapse(d time.Duration) step {
	return func(g *gestureTester, _ *log) { g.elapse(d) }
}
func closeArena(p int) step { return func(g *gestureTester, _ *log) { g.closeArena(p) } }
func flushMicrotasks() step { return func(g *gestureTester, _ *log) { g.flush() } }
func do(fn func()) step     { return func(*gestureTester, *log) { fn() } }

// want checks the log; the next steps start with an empty log if clear is true.
func want(entries ...string) step {
	return func(g *gestureTester, l *log) {
		g.t.Helper()
		l.check(g.t, entries...)
	}
}

// wantClear checks the log and empties it.
func wantClear(entries ...string) step {
	return func(g *gestureTester, l *log) {
		g.t.Helper()
		l.check(g.t, entries...)
		l.clear()
	}
}

func run(g *gestureTester, l *log, steps ...step) {
	for _, s := range steps {
		s(g, l)
	}
}

func (p *testPointer) downAt(location ui.Offset, ts time.Duration) *PointerDownEvent {
	p.isDown = true
	p.location = location
	return NewPointerDownEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location, TimeStamp: ts,
		Buttons: someInt(p.buttons),
	})
}

func (p *testPointer) moveAt(location ui.Offset, ts time.Duration) *PointerMoveEvent {
	delta := location.Sub(p.location)
	p.location = location
	return NewPointerMoveEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location, Delta: delta, TimeStamp: ts,
		Buttons: someInt(p.buttons),
	})
}

// moveButtons moves with other buttons than the pointer's.
func (p *testPointer) moveButtons(location ui.Offset, buttons int) *PointerMoveEvent {
	delta := location.Sub(p.location)
	p.location = location
	return NewPointerMoveEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location, Delta: delta,
		Buttons: someInt(buttons),
	})
}

func (p *testPointer) upAt(ts time.Duration) *PointerUpEvent {
	p.isDown = false
	return NewPointerUpEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: p.location, TimeStamp: ts,
	})
}

func (p *testPointer) panZoomStart(location ui.Offset) *PointerPanZoomStartEvent {
	p.location, p.pan = location, ui.Offset{}
	return NewPointerPanZoomStartEvent(PointerEventOpts{Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location})
}

func (p *testPointer) panZoomUpdate(location, pan ui.Offset, scale float64, ts time.Duration) *PointerPanZoomUpdateEvent {
	e := NewPointerPanZoomUpdateEvent(PointerEventOpts{
		Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: location, Pan: pan, PanDelta: pan.Sub(p.pan),
		Scale: someFloat(scale), TimeStamp: ts,
	})
	p.pan = pan
	return e
}

func (p *testPointer) panZoomEnd(ts time.Duration) *PointerPanZoomEndEvent {
	return NewPointerPanZoomEndEvent(PointerEventOpts{Pointer: p.pointer, Kind: p.kind, Device: p.device, Position: p.location, TimeStamp: ts})
}
