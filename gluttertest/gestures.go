package gluttertest

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
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
	// TouchSlopX and TouchSlopY are how far Drag and DragFrom move first, along
	// each axis, so that the recognizers of drags win; the default is
	// DragSlopDefault.
	//
	// Flutter: flutter_test/lib/src/controller.dart WidgetController.drag touchSlopX
	TouchSlopX, TouchSlopY foundation.Opt[float64]
	// View is the view the pointer is in; the default is the implicit view. The
	// gestures that take a Finder use the view of the element it finds.
	//
	// Flutter: flutter_test/lib/src/controller.dart WidgetController.startGesture
	View ui.ViewID
}

// viewOpts is opts with the view of the one element the finder finds, unless the
// test chose a view.
func (t *Tester) viewOpts(f Finder, opts []GestureOpts) []GestureOpts {
	var o GestureOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.View == ui.ImplicitViewID {
		if v := widgets.MaybeViewOf(t.Element(f)); v != nil {
			o.View = v.ViewID()
		}
	}
	return []GestureOpts{o}
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
		// Every gesture gets a new pointer ID, as in Flutter: the arena of a pointer
		// is not reused, so a recognizer's entry of a past gesture is harmless.
		// The mouse device is the same for all of them (Hover uses its own ID).
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
	stamp    foundation.Opt[time.Duration]
	// addedMouse is whether this gesture added the mouse, and so removes it.
	addedMouse bool
}

func (g *TestGesture) eventOpts(delta ui.Offset) gestures.PointerEventOpts {
	return gestures.PointerEventOpts{
		TimeStamp: g.timeStamp(), Pointer: g.opts.Pointer, Kind: g.opts.Kind,
		Position: g.position, Delta: delta, Buttons: g.opts.Buttons, ViewID: g.opts.View,
	}
}

// StartGesture presses a pointer at the position (logical pixels) and returns
// the gesture to move and release it. A mouse is added first, as a real one
// would be. The tester runs the posted functions after the press (and after every
// move, up and cancel), as the await in Flutter's test gestures runs the
// microtasks: a recognizer alone in the arena wins at once.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.startGesture
func (t *Tester) StartGesture(at ui.Offset, opts ...GestureOpts) *TestGesture {
	t.adopt()
	t.tb.Helper()
	g := &TestGesture{t: t, opts: t.gestureOpts(opts), position: at, down: true}
	if g.opts.Kind == ui.PointerDeviceKindMouse {
		if !t.mouseAdded {
			t.mouseAdded = true
			g.addedMouse = true
			t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerAddedEvent(g.eventOpts(ui.Offset{})))
		}
		t.mousePosition = at
	}
	t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerDownEvent(g.eventOpts(ui.Offset{})))
	t.host.RunPosted() // the await of Flutter's startGesture runs the microtasks (the arena's default resolution)
	return g
}

func (g *TestGesture) timeStamp() time.Duration {
	if g.stamp.IsSome() {
		return g.stamp.Value()
	}
	return g.t.Elapsed()
}

// Down presses the pointer again at the position, after Up or Cancel. A mouse
// that Up removed is added again first.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.down
func (g *TestGesture) Down(at ui.Offset) {
	g.t.tb.Helper()
	g.t.adopt()
	if g.down {
		g.t.fail(errors.New(tagGestureAlreadyDown, "TestGesture.Down was called on a pointer that is already pressed"))
		return
	}
	g.down = true
	g.position = at
	if g.opts.Kind == ui.PointerDeviceKindMouse {
		if !g.t.mouseAdded {
			g.t.mouseAdded = true
			g.addedMouse = true
			g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerAddedEvent(g.eventOpts(ui.Offset{})))
		}
		g.t.mousePosition = at
	}
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerDownEvent(g.eventOpts(ui.Offset{})))
	g.t.host.RunPosted()
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
	if g.opts.Kind == ui.PointerDeviceKindMouse {
		g.t.mousePosition = to
	}
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerMoveEvent(g.eventOpts(delta)))
	g.t.host.RunPosted()
}

// MoveBy moves the pressed pointer by the offset.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.moveBy
func (g *TestGesture) MoveBy(by ui.Offset) {
	g.t.tb.Helper()
	g.MoveTo(g.position.Add(by))
}

// MoveByAt moves the pressed pointer by the offset with the event time stamp
// set to timeStamp (since the start of the test), instead of the time of the fake
// clock. The next events keep the time stamp until another call sets it.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestGesture.moveBy timeStamp
func (g *TestGesture) MoveByAt(by ui.Offset, timeStamp time.Duration) {
	g.t.tb.Helper()
	g.stamp = foundation.Some(timeStamp)
	g.MoveBy(by)
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
	g.t.host.RunPosted()
	if g.addedMouse {
		g.t.mouseAdded = false
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
	g.t.host.RunPosted()
}

func (g *TestGesture) requireDown(method string) {
	g.t.adopt()
	g.t.tb.Helper()
	if !g.down {
		g.t.fail(errors.New(tagGestureNotDown, "TestGesture."+method+" was called on a pointer that is not pressed", "method", method))
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
	t.TapAt(t.Center(f), t.viewOpts(f, opts)...)
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
	t.LongPressAt(t.Center(f), t.viewOpts(f, opts)...)
}

// DragFrom presses a pointer at the position and moves it by the offset, first
// by the slop (so that the recognizers of drags win), then by the rest, and
// releases it.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.dragFrom
func (t *Tester) DragFrom(at, by ui.Offset, opts ...GestureOpts) {
	t.tb.Helper()
	g := t.StartGesture(at, opts...)
	var o GestureOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	slop := ui.Offset{Dx: sign(by.Dx) * o.TouchSlopX.OrElse(DragSlopDefault), Dy: sign(by.Dy) * o.TouchSlopY.OrElse(DragSlopDefault)}
	g.MoveBy(slop)
	g.MoveBy(by.Sub(slop))
	g.Up()
}

// Drag drags from the center of the one element the finder finds by the offset.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.drag
func (t *Tester) Drag(f Finder, by ui.Offset, opts ...GestureOpts) {
	t.tb.Helper()
	t.DragFrom(t.Center(f), by, t.viewOpts(f, opts)...)
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
		t.fail(errors.New(tagFlingInvalid, "a fling needs a speed above zero and an offset that is not zero",
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
	t.FlingFrom(t.Center(f), by, speed, t.viewOpts(f, opts)...)
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

// ScrollKindAt is ScrollAt for a device of another kind, such as a trackpad. Each
// kind has one pointer of its own, which stays for the life of the tester.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.scroll
func (t *Tester) ScrollKindAt(kind ui.PointerDeviceKind, at, delta ui.Offset) {
	if kind == ui.PointerDeviceKindMouse {
		t.ScrollAt(at, delta)
		return
	}
	t.adopt()
	t.tb.Helper()
	if t.scrollPointers == nil {
		t.scrollPointers = map[ui.PointerDeviceKind]int{}
	}
	g := t.Binding.Gestures()
	id, ok := t.scrollPointers[kind]
	if !ok {
		t.pointerID++
		id = t.pointerID
		t.scrollPointers[kind] = id
	}
	o := gestures.PointerEventOpts{TimeStamp: t.Elapsed(), Pointer: id, Device: id, Kind: kind, Position: at}
	if !ok {
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

// ShowKeyboard gives the focus to the text field the finder finds (or whose
// descendant it finds) and opens its input connection, as the tap on a field
// does. The test fails with the tag EnterTextNoField when the finder finds no
// EditableText.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.showKeyboard
func (t *Tester) ShowKeyboard(f Finder) *widgets.EditableTextState {
	t.tb.Helper()
	t.adopt()
	found := Descendant(f, ByType[*widgets.EditableText](), DescendantOpts{MatchRoot: true})
	els := t.Elements(found)
	if len(els) == 0 {
		t.fail(errors.New(tagEnterTextNoField, "the finder found no EditableText", "finder", f.Describe()))
		return nil
	}
	stateful, ok := els[0].(*widgets.StatefulElement)
	if !ok {
		t.fail(errors.New(tagEnterTextNoField, "the EditableText element has no state", "finder", f.Describe()))
		return nil
	}
	state := stateful.State().(*widgets.EditableTextState)
	state.RequestKeyboard()
	t.Pump()
	return state
}

// EnterText types the text into the text field the finder finds: the field gets
// the focus, and its text becomes the text, with the caret at the end, as if the
// input method had sent the new value.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.enterText
func (t *Tester) EnterText(f Finder, text string) {
	t.tb.Helper()
	state := t.ShowKeyboard(f)
	if state == nil {
		return
	}
	state.UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: text, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(text))),
	}))
	t.Pump()
}

// KeyOpts are the optional parameters of the key events of the tester.
//
// Flutter: flutter_test/lib/src/event_simulation.dart simulateKeyDownEvent
type KeyOpts struct {
	// Physical is the physical key; the default is the key of the same name as the
	// logical key (services.PhysicalKeyOf), or none.
	Physical services.PhysicalKeyboardKey
	// Character is the text the key produces, for a key down or a repeat.
	Character string
}

// leftKeyOf maps the pseudo-keys to the key on the left of the pair.
var leftKeyOf = map[services.LogicalKeyboardKey]services.LogicalKeyboardKey{
	services.LogicalKeyboardKeyShift:   services.LogicalKeyboardKeyShiftLeft,
	services.LogicalKeyboardKeyControl: services.LogicalKeyboardKeyControlLeft,
	services.LogicalKeyboardKeyAlt:     services.LogicalKeyboardKeyAltLeft,
	services.LogicalKeyboardKeyMeta:    services.LogicalKeyboardKeyMetaLeft,
}

func (t *Tester) sendKey(typ ui.KeyEventType, logical services.LogicalKeyboardKey, opts []KeyOpts) bool {
	t.adopt()
	var o KeyOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	// A pseudo-key stands for a real one: the left key of the pair, as the
	// platform key maps of Flutter's test simulation give.
	if left, ok := leftKeyOf[logical]; ok {
		logical = left
	}
	if o.Physical == 0 {
		o.Physical, _ = services.PhysicalKeyOf(logical)
	}
	d := ui.KeyData{TimeStamp: t.Elapsed(), Type: typ, Logical: logical.KeyID(), Physical: o.Physical.USBHidUsage()}
	if typ != ui.KeyEventTypeUp {
		d.Character = o.Character
	}
	handled := t.d.DispatchKeyData(d)
	// Flutter's simulation awaits the event, which runs the microtasks it queued,
	// such as the focus changes of a Tab key.
	t.FlushPosted()
	return handled
}

// SendKeyDownEvent sends a key down for the logical key and returns whether
// something handled it.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyDownEvent
func (t *Tester) SendKeyDownEvent(logical services.LogicalKeyboardKey, opts ...KeyOpts) bool {
	t.tb.Helper()
	return t.sendKey(ui.KeyEventTypeDown, logical, opts)
}

// SendKeyRepeatEvent sends a key repeat for the logical key and returns whether
// something handled it.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyRepeatEvent
func (t *Tester) SendKeyRepeatEvent(logical services.LogicalKeyboardKey, opts ...KeyOpts) bool {
	t.tb.Helper()
	return t.sendKey(ui.KeyEventTypeRepeat, logical, opts)
}

// SendKeyUpEvent sends a key up for the logical key and returns whether
// something handled it.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyUpEvent
func (t *Tester) SendKeyUpEvent(logical services.LogicalKeyboardKey, opts ...KeyOpts) bool {
	t.tb.Helper()
	return t.sendKey(ui.KeyEventTypeUp, logical, opts)
}

// SendKeyEvent sends a key down and a key up for the logical key and returns
// whether the down was handled.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.sendKeyEvent
func (t *Tester) SendKeyEvent(logical services.LogicalKeyboardKey, opts ...KeyOpts) bool {
	t.tb.Helper()
	handled := t.sendKey(ui.KeyEventTypeDown, logical, opts)
	t.sendKey(ui.KeyEventTypeUp, logical, opts)
	return handled
}

// TestTrackpadGesture is a pan/zoom gesture of a trackpad that the test updates:
// it sent a pan/zoom start event when it was created and sends the updates or the
// end when the test calls them.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.panZoomStart
type TestTrackpadGesture struct {
	t        *Tester
	pointer  int
	position ui.Offset
	pan      ui.Offset
	scale    float64
	rotation float64
	active   bool
}

func (g *TestTrackpadGesture) eventOpts() gestures.PointerEventOpts {
	return gestures.PointerEventOpts{TimeStamp: g.t.Elapsed(), Pointer: g.pointer, Kind: ui.PointerDeviceKindTrackpad, Position: g.position}
}

// StartTrackpadGesture starts a pan/zoom gesture of a trackpad at the position
// (logical pixels). The tester runs the posted functions after every event, as
// StartGesture does.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.panZoomStart
func (t *Tester) StartTrackpadGesture(at ui.Offset) *TestTrackpadGesture {
	t.adopt()
	t.tb.Helper()
	t.pointerID++
	g := &TestTrackpadGesture{t: t, pointer: t.pointerID, position: at, scale: 1, active: true}
	t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerPanZoomStartEvent(g.eventOpts()))
	t.host.RunPosted()
	return g
}

// Update reports the totals of the gesture since it started: the pan, the scale
// and the rotation (radians).
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.panZoomUpdate
func (g *TestTrackpadGesture) Update(pan ui.Offset, scale, rotation float64) {
	g.t.adopt()
	g.t.tb.Helper()
	g.requireActive("Update")
	o := g.eventOpts()
	o.Pan, o.PanDelta, o.Scale, o.Rotation = pan, pan.Sub(g.pan), foundation.Some(scale), rotation
	g.pan, g.scale, g.rotation = pan, scale, rotation
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerPanZoomUpdateEvent(o))
	g.t.host.RunPosted()
}

// PanBy pans the gesture by the offset from where it is.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.panZoomUpdate
func (g *TestTrackpadGesture) PanBy(by ui.Offset) {
	g.t.tb.Helper()
	g.Update(g.pan.Add(by), g.scale, g.rotation)
}

// End ends the gesture.
//
// Flutter: flutter_test/lib/src/test_pointer.dart TestPointer.panZoomEnd
func (g *TestTrackpadGesture) End() {
	g.t.adopt()
	g.t.tb.Helper()
	g.requireActive("End")
	g.active = false
	g.t.Binding.Gestures().HandlePointerEvent(gestures.NewPointerPanZoomEndEvent(g.eventOpts()))
	g.t.host.RunPosted()
}

func (g *TestTrackpadGesture) requireActive(method string) {
	g.t.tb.Helper()
	if !g.active {
		g.t.fail(errors.New(tagGestureNotDown, "TestTrackpadGesture."+method+" was called on a gesture that has ended", "method", method))
	}
}

// TrackpadFlingFrom starts a trackpad pan at the position and pans it by the
// offset in 50 steps at the speed (logical pixels per second), pumping between
// the steps, then ends it.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.trackpadFlingFrom
func (t *Tester) TrackpadFlingFrom(at, by ui.Offset, speed float64) {
	t.tb.Helper()
	const moveCount = 50
	if speed <= 0 || by.Distance() == 0 {
		t.fail(errors.New(tagFlingInvalid, "a fling needs a speed above zero and an offset that is not zero",
			"speed", speed, "offset", by.Distance()))
		return
	}
	step := by.Mul(1.0 / moveCount)
	interval := time.Duration(float64(time.Second) * by.Distance() / speed / moveCount)
	g := t.StartTrackpadGesture(at)
	for i := 0; i < moveCount; i++ {
		t.Pump(interval)
		g.PanBy(step)
	}
	g.End()
}

// TrackpadFling flings a trackpad from the center of the one element the finder
// finds.
//
// Flutter: flutter_test/lib/src/controller.dart WidgetController.trackpadFling
func (t *Tester) TrackpadFling(f Finder, by ui.Offset, speed float64) {
	t.tb.Helper()
	t.TrackpadFlingFrom(t.Center(f), by, speed)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEnterTextNoField   = "EnterTextNoField"
	tagFlingInvalid       = "FlingInvalid"
	tagGestureAlreadyDown = "GestureAlreadyDown"
	tagGestureNotDown     = "GestureNotDown"
)
