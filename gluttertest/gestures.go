package gluttertest

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// DragSlopDefault is how far a drag moves first so that it passes the touch
// slop of the recognizers, in logical pixels.
//
// Flutter: flutter_test/lib/src/controller.dart kDragSlopDefault
const DragSlopDefault = 20.0

// GestureOpts are the optional parameters of the gestures of the tester.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.startGesture
type GestureOpts struct {
	// Kind is the kind of device; the default is touch.
	Kind ui.PointerDeviceKind
	// Buttons are the buttons pressed; the default is the primary button.
	Buttons foundation.Opt[int]
	// Pointer is the ID of the pointer; the default is a new one each time.
	Pointer int
}

func (t *Tester) gestureOpts(opts []GestureOpts) GestureOpts {
	var o GestureOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Kind == ui.PointerDeviceKindUnspecified {
		o.Kind = ui.PointerDeviceKindTouch
	}
	if o.Pointer == 0 {
		t.pointerID++
		o.Pointer = t.pointerID
	}
	return o
}

// TestGesture is a pointer that the test moves: it sent a down event when it was
// created and sends the moves, the up or the cancel when the test calls them.
// The events carry the time of the fake clock.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture
type TestGesture struct {
	t        *Tester
	opts     GestureOpts
	position ui.Offset
	down     bool
}

func (g *TestGesture) eventOpts(delta ui.Offset) gestures.PointerEventOpts {
	return gestures.PointerEventOpts{
		TimeStamp: g.t.Elapsed(), Pointer: g.opts.Pointer, Kind: g.opts.Kind,
		Position: g.position, Delta: delta, Buttons: g.opts.Buttons,
	}
}

// StartGesture presses a pointer at the position (logical pixels) and returns
// the gesture to move and release it. A mouse is added first, as a real one
// would be.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.startGesture
func (t *Tester) StartGesture(at ui.Offset, opts ...GestureOpts) *TestGesture {
	t.adopt()
	t.tb.Helper()
	g := &TestGesture{t: t, opts: t.gestureOpts(opts), position: at, down: true}
	if g.opts.Kind == ui.PointerDeviceKindMouse {
		t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerAddedEvent(g.eventOpts(ui.Offset{})))
	}
	t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerDownEvent(g.eventOpts(ui.Offset{})))
	return g
}

// Position is where the pointer is.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.location
func (g *TestGesture) Position() ui.Offset { return g.position }

// MoveTo moves the pressed pointer to the position.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.moveTo
func (g *TestGesture) MoveTo(to ui.Offset) {
	g.t.tb.Helper()
	g.requireDown("MoveTo")
	delta := to.Sub(g.position)
	g.position = to
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerMoveEvent(g.eventOpts(delta)))
}

// MoveBy moves the pressed pointer by the offset.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.moveBy
func (g *TestGesture) MoveBy(by ui.Offset) {
	g.t.tb.Helper()
	g.MoveTo(g.position.Add(by))
}

// Up releases the pointer.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.up
func (g *TestGesture) Up() {
	g.t.tb.Helper()
	g.requireDown("Up")
	g.down = false
	o := g.eventOpts(ui.Offset{})
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerUpEvent(o))
	if g.opts.Kind == ui.PointerDeviceKindMouse {
		g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerRemovedEvent(o))
	}
}

// Cancel cancels the pointer, as the system does when it takes the touch away.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.cancel
func (g *TestGesture) Cancel() {
	g.t.tb.Helper()
	g.requireDown("Cancel")
	g.down = false
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerCancelEvent(g.eventOpts(ui.Offset{})))
}

func (g *TestGesture) requireDown(method string) {
	g.t.adopt()
	g.t.tb.Helper()
	if !g.down {
		g.t.fail(errors.New("GestureNotDown", "TestGesture."+method+" was called on a pointer that is not pressed", "method", method))
	}
}

// TapAt presses and releases a pointer at the position, without time passing in
// between.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.tapAt
func (t *Tester) TapAt(at ui.Offset, opts ...GestureOpts) {
	t.tb.Helper()
	t.StartGesture(at, opts...).Up()
}

// Tap taps the center of the one element the finder finds.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.tap
func (t *Tester) Tap(f Finder, opts ...GestureOpts) {
	t.tb.Helper()
	t.TapAt(t.Center(f), opts...)
}

// LongPressAt presses a pointer at the position, lets enough fake time pass for
// a long press to be recognized, and releases it. It pumps while it waits.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.longPressAt
func (t *Tester) LongPressAt(at ui.Offset, opts ...GestureOpts) {
	t.tb.Helper()
	g := t.StartGesture(at, opts...)
	t.Pump(gestures.LongPressTimeout + gestures.PressTimeout)
	g.Up()
}

// LongPress long-presses the center of the one element the finder finds.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.longPress
func (t *Tester) LongPress(f Finder, opts ...GestureOpts) {
	t.tb.Helper()
	t.LongPressAt(t.Center(f), opts...)
}

// DragFrom presses a pointer at the position and moves it by the offset, first
// by the slop (so that the recognizers of drags win), then by the rest, and
// releases it.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.dragFrom
func (t *Tester) DragFrom(at, by ui.Offset, opts ...GestureOpts) {
	t.tb.Helper()
	g := t.StartGesture(at, opts...)
	slop := ui.Offset{Dx: sign(by.Dx) * DragSlopDefault, Dy: sign(by.Dy) * DragSlopDefault}
	g.MoveBy(slop)
	g.MoveBy(by.Sub(slop))
	g.Up()
}

// Drag drags from the center of the one element the finder finds by the offset.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.drag
func (t *Tester) Drag(f Finder, by ui.Offset, opts ...GestureOpts) {
	t.tb.Helper()
	t.DragFrom(t.Center(f), by, opts...)
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// FlingFrom presses a pointer at the position and moves it by the offset in
// 50 steps at the speed (logical pixels per second), pumping between the steps
// so that the velocity the recognizers measure is the speed, then releases it.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.flingFrom
func (t *Tester) FlingFrom(at, by ui.Offset, speed float64, opts ...GestureOpts) {
	t.tb.Helper()
	const moveCount = 50
	if speed <= 0 || by.Distance() == 0 {
		t.fail(errors.New("FlingInvalid", "a fling needs a speed above zero and an offset that is not zero",
			"speed", speed, "offset", by.Distance()))
		return
	}
	step := by.Mul(1.0 / moveCount)
	interval := time.Duration(float64(time.Second) * by.Distance() / speed / moveCount)
	g := t.StartGesture(at, opts...)
	for i := 0; i < moveCount; i++ {
		t.Pump(interval)
		g.MoveBy(step)
	}
	g.Up()
}

// Fling flings from the center of the one element the finder finds.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.fling
func (t *Tester) Fling(f Finder, by ui.Offset, speed float64, opts ...GestureOpts) {
	t.tb.Helper()
	t.FlingFrom(t.Center(f), by, speed, opts...)
}

// Hover moves a mouse to the position without pressing it; the first hover adds
// the mouse.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.hover
func (t *Tester) Hover(at ui.Offset) {
	t.adopt()
	t.tb.Helper()
	g := t.Binding.Gestures()
	o := gestures.PointerEventOpts{TimeStamp: t.Elapsed(), Pointer: t.mousePointer(), Kind: ui.PointerDeviceKindMouse, Position: at}
	if !t.mouseAdded {
		t.mouseAdded = true
		g.HandlePointerEvent(gestures.NewPointerAddedEvent(o))
	}
	o.Delta = at.Sub(t.mousePosition)
	t.mousePosition = at
	g.HandlePointerEvent(gestures.NewPointerHoverEvent(o))
}

func (t *Tester) mousePointer() int {
	if t.mouseID == 0 {
		t.pointerID++
		t.mouseID = t.pointerID
	}
	return t.mouseID
}

// ScrollAt sends a scroll event of a mouse wheel at the position, as the system
// does for one notch or a trackpad scroll, and pumps nothing.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.sendEventToBinding / TestPointer.scroll
func (t *Tester) ScrollAt(at, delta ui.Offset) {
	t.adopt()
	t.tb.Helper()
	g := t.Binding.Gestures()
	o := gestures.PointerEventOpts{TimeStamp: t.Elapsed(), Pointer: t.mousePointer(), Kind: ui.PointerDeviceKindMouse, Position: at}
	if !t.mouseAdded {
		t.mouseAdded = true
		g.HandlePointerEvent(gestures.NewPointerAddedEvent(o))
	}
	o.ScrollDelta = delta
	g.HandlePointerEvent(gestures.NewPointerScrollEvent(o))
}

// Scroll scrolls the wheel over the center of the one element the finder finds.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.scroll
func (t *Tester) Scroll(f Finder, delta ui.Offset) {
	t.tb.Helper()
	t.ScrollAt(t.Center(f), delta)
}

// EnterText types the text into the text field the finder finds. Text input
// arrives with the text input service (step 11.4); until then the test fails
// with the tag EnterTextNotImplemented.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.enterText
func (t *Tester) EnterText(f Finder, text string) {
	t.tb.Helper()
	t.fail(errors.New("EnterTextNotImplemented", "EnterText needs the text input service, which comes with step 11.4",
		"finder", f.Describe(), "text", text))
}

// KeyOpts are the optional parameters of the key events of the tester.
//
// Flutter: flutter_test/lib/src/event_simulation.dart simulateKeyDownEvent
type KeyOpts struct {
	// Physical is the ID of the physical key; 0 for none.
	Physical uint64
	// Character is the text the key produces, for a key down.
	Character string
}

func (t *Tester) sendKey(typ ui.KeyEventType, logical uint64, opts []KeyOpts) bool {
	t.adopt()
	var o KeyOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	d := ui.KeyData{TimeStamp: t.Elapsed(), Type: typ, Logical: logical, Physical: o.Physical}
	if typ == ui.KeyEventTypeDown {
		d.Character = o.Character
	}
	return t.d.DispatchKeyData(d)
}

// SendKeyDownEvent sends a key down for the logical key and returns whether
// something handled it.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyDownEvent
func (t *Tester) SendKeyDownEvent(logical uint64, opts ...KeyOpts) bool {
	t.tb.Helper()
	return t.sendKey(ui.KeyEventTypeDown, logical, opts)
}

// SendKeyUpEvent sends a key up for the logical key and returns whether
// something handled it.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyUpEvent
func (t *Tester) SendKeyUpEvent(logical uint64, opts ...KeyOpts) bool {
	t.tb.Helper()
	return t.sendKey(ui.KeyEventTypeUp, logical, opts)
}

// SendKeyEvent sends a key down and a key up for the logical key and returns
// whether the down was handled.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyEvent
func (t *Tester) SendKeyEvent(logical uint64, opts ...KeyOpts) bool {
	t.tb.Helper()
	handled := t.sendKey(ui.KeyEventTypeDown, logical, opts)
	t.sendKey(ui.KeyEventTypeUp, logical, opts)
	return handled
}
