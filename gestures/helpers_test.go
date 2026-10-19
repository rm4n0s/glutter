package gestures

import (
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
