package gluttertest

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestTapCountsOnTheCounter(t *testing.T) {
	ts := New(t)
	ts.PumpWidget(newCounter())
	taps := func() int { return StateOf[*counterState](ts, ByType[*counter]()).taps }
	tests := []struct {
		name string
		do   func()
		want int
	}{
		{"nothing yet", func() {}, 0},
		{"tap the box", func() { ts.Tap(First(ByType[*swatch]())) }, 1},
		{"tap again", func() { ts.Tap(First(ByType[*swatch]())) }, 2},
		{"tap outside the box", func() { ts.TapAt(ui.Offset{Dx: 700, Dy: 500}) }, 2},
		{"tap at a point in the box", func() { ts.TapAt(ui.Offset{Dx: 5, Dy: 5}) }, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
			ts.Pump()
			if got := taps(); got != tt.want {
				t.Errorf("taps = %d, want %d", got, tt.want)
			}
		})
	}
	if got := ts.Count(ByType[*swatch]()); got != 4 {
		t.Errorf("swatches = %d, want the box and three markers", got)
	}
}

func TestEventsCarryTheFakeClockAndPosition(t *testing.T) {
	ts := New(t)
	var downs []*gestures.PointerDownEvent
	var moves []*gestures.PointerMoveEvent
	var ups []*gestures.PointerUpEvent
	l := newListener(newSwatch(grey, 200, 200))
	l.onDown = func(e *gestures.PointerDownEvent) { downs = append(downs, e) }
	l.onMove = func(e *gestures.PointerMoveEvent) { moves = append(moves, e) }
	l.onUp = func(e *gestures.PointerUpEvent) { ups = append(ups, e) }
	ts.PumpWidget(newRow(l))
	ts.Pump(time.Second)

	g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 20})
	ts.Pump(100 * time.Millisecond)
	g.MoveTo(ui.Offset{Dx: 30, Dy: 25})
	g.MoveBy(ui.Offset{Dx: 5, Dy: 5})
	g.Up()

	if len(downs) != 1 || len(moves) != 2 || len(ups) != 1 {
		t.Fatalf("events: %d down, %d move, %d up", len(downs), len(moves), len(ups))
	}
	if downs[0].Position() != (ui.Offset{Dx: 10, Dy: 20}) || downs[0].TimeStamp() != time.Second {
		t.Errorf("down = %v at %v", downs[0].Position(), downs[0].TimeStamp())
	}
	if moves[0].Delta() != (ui.Offset{Dx: 20, Dy: 5}) || moves[0].TimeStamp() != 1100*time.Millisecond {
		t.Errorf("move 0 = %v at %v", moves[0].Delta(), moves[0].TimeStamp())
	}
	if moves[1].Position() != (ui.Offset{Dx: 35, Dy: 30}) {
		t.Errorf("move 1 = %v", moves[1].Position())
	}
	if ups[0].Position() != (ui.Offset{Dx: 35, Dy: 30}) {
		t.Errorf("up = %v", ups[0].Position())
	}
	if g.Position() != (ui.Offset{Dx: 35, Dy: 30}) {
		t.Errorf("gesture position = %v", g.Position())
	}
}

func TestPointerIDsAreUniquePerGesture(t *testing.T) {
	ts := New(t)
	var ids []int
	l := newListener(newSwatch(grey, 200, 200))
	l.onDown = func(e *gestures.PointerDownEvent) { ids = append(ids, e.Pointer()) }
	ts.PumpWidget(newRow(l))
	ts.TapAt(ui.Offset{Dx: 1, Dy: 1})
	ts.TapAt(ui.Offset{Dx: 1, Dy: 1})
	ts.TapAt(ui.Offset{Dx: 1, Dy: 1}, GestureOpts{Pointer: 42})
	if len(ids) != 3 || ids[0] == ids[1] || ids[2] != 42 {
		t.Errorf("pointer IDs = %v", ids)
	}
}

func TestGestureCancel(t *testing.T) {
	ts := New(t)
	cancels := 0
	l := newListener(newSwatch(grey, 200, 200))
	l.onCancel = func(*gestures.PointerCancelEvent) { cancels++ }
	ts.PumpWidget(newRow(l))
	ts.StartGesture(ui.Offset{Dx: 1, Dy: 1}).Cancel()
	if cancels != 1 {
		t.Errorf("cancels = %d", cancels)
	}
}

func TestGestureMisuse(t *testing.T) {
	tests := []struct {
		name string
		do   func(g *TestGesture)
	}{
		{"move after up", func(g *TestGesture) { g.Up(); g.MoveTo(ui.Offset{}) }},
		{"up twice", func(g *TestGesture) { g.Up(); g.Up() }},
		{"cancel after up", func(g *TestGesture) { g.Up(); g.Cancel() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := failure(t, func(ts *Tester) {
				ts.PumpWidget(newRow())
				tt.do(ts.StartGesture(ui.Offset{}))
			})
			requireRoute(t, appErr, "GestureNotDown")
		})
	}
}

func TestLongPressWaitsForTheTimer(t *testing.T) {
	ts := New(t)
	longPresses := 0
	rec := gestures.NewLongPressGestureRecognizer(ts.Binding.Gestures())
	rec.OnLongPress = func() { longPresses++ }
	l := newListener(newSwatch(grey, 100, 100))
	l.onDown = func(e *gestures.PointerDownEvent) { rec.AddPointer(e) }
	ts.PumpWidget(newRow(l))

	ts.TapAt(ui.Offset{Dx: 10, Dy: 10})
	if longPresses != 0 {
		t.Fatalf("a tap was a long press")
	}
	ts.LongPress(ByType[*listener]())
	if longPresses != 1 {
		t.Errorf("long presses = %d, want 1", longPresses)
	}
}

func TestDragMovesBySlopFirst(t *testing.T) {
	tests := []struct {
		name      string
		by        ui.Offset
		wantFirst ui.Offset
	}{
		{"right", ui.Offset{Dx: 100}, ui.Offset{Dx: DragSlopDefault}},
		{"left", ui.Offset{Dx: -100}, ui.Offset{Dx: -DragSlopDefault}},
		{"down", ui.Offset{Dy: 60}, ui.Offset{Dy: DragSlopDefault}},
		{"diagonal", ui.Offset{Dx: 50, Dy: -50}, ui.Offset{Dx: DragSlopDefault, Dy: -DragSlopDefault}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t)
			var deltas []ui.Offset
			l := newListener(newSwatch(grey, 300, 300))
			l.onMove = func(e *gestures.PointerMoveEvent) { deltas = append(deltas, e.Delta()) }
			ts.PumpWidget(newRow(l))
			ts.DragFrom(ui.Offset{Dx: 150, Dy: 150}, tt.by)
			if len(deltas) != 2 || deltas[0] != tt.wantFirst || deltas[0].Add(deltas[1]) != tt.by {
				t.Errorf("deltas = %v, want the slop %v then the rest of %v", deltas, tt.wantFirst, tt.by)
			}
		})
	}
}

func TestDragRecognizedByAPanRecognizer(t *testing.T) {
	ts := New(t)
	var total ui.Offset
	started, ended := 0, 0
	rec := gestures.NewPanGestureRecognizer(ts.Binding.Gestures())
	rec.OnStart = func(gestures.DragStartDetails) { started++ }
	rec.OnUpdate = func(d gestures.DragUpdateDetails) { total = total.Add(d.Delta) }
	rec.OnEnd = func(gestures.DragEndDetails) { ended++ }
	l := newListener(newSwatch(grey, 300, 300))
	l.onDown = func(e *gestures.PointerDownEvent) { rec.AddPointer(e) }
	ts.PumpWidget(newRow(l))
	// A recognizer alone in the arena wins at the pointer down, when the tester
	// runs the microtasks as the await of Flutter's startGesture does, so every
	// move is an update.
	g := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	g.MoveBy(ui.Offset{Dx: 50})
	g.MoveBy(ui.Offset{Dx: 30})
	g.Up()
	if started != 1 || ended != 1 {
		t.Errorf("started %d, ended %d", started, ended)
	}
	if total != (ui.Offset{Dx: 80}) {
		t.Errorf("total drag = %v, want 80 to the right", total)
	}
	ts.Drag(ByType[*listener](), ui.Offset{Dx: 80})
	if started != 2 || ended != 2 {
		t.Errorf("after Drag: started %d, ended %d", started, ended)
	}
}

func TestFlingHasTheSpeed(t *testing.T) {
	ts := New(t)
	var velocity gestures.Velocity
	rec := gestures.NewPanGestureRecognizer(ts.Binding.Gestures())
	rec.OnEnd = func(d gestures.DragEndDetails) { velocity = d.Velocity }
	l := newListener(newSwatch(grey, 300, 300))
	l.onDown = func(e *gestures.PointerDownEvent) { rec.AddPointer(e) }
	ts.PumpWidget(newRow(l))
	before := ts.Elapsed()
	ts.Fling(ByType[*listener](), ui.Offset{Dx: 200}, 1000)
	if got := ts.Elapsed() - before; got < 190*time.Millisecond || got > 210*time.Millisecond {
		t.Errorf("a fling of 200 px at 1000 px/s took %v", got)
	}
	if v := velocity.PixelsPerSecond.Dx; v < 900 || v > 1100 {
		t.Errorf("velocity = %v, want about 1000", velocity)
	}
}

func TestFlingInvalid(t *testing.T) {
	tests := []struct {
		name  string
		by    ui.Offset
		speed float64
	}{
		{"no speed", ui.Offset{Dx: 10}, 0},
		{"no distance", ui.Offset{}, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := failure(t, func(ts *Tester) {
				ts.PumpWidget(newRow())
				ts.FlingFrom(ui.Offset{}, tt.by, tt.speed)
			})
			requireRoute(t, appErr, "FlingInvalid")
		})
	}
}

func TestHoverAndScroll(t *testing.T) {
	ts := New(t)
	var hovers []ui.Offset
	var scrolls []ui.Offset
	l := newListener(newSwatch(grey, 300, 300))
	l.onHover = func(e *gestures.PointerHoverEvent) { hovers = append(hovers, e.Position()) }
	l.onSignal = func(e gestures.PointerSignalEvent) {
		if s, ok := e.(*gestures.PointerScrollEvent); ok {
			scrolls = append(scrolls, s.ScrollDelta())
		}
	}
	ts.PumpWidget(newRow(l))
	ts.Hover(ui.Offset{Dx: 10, Dy: 10})
	ts.Hover(ui.Offset{Dx: 20, Dy: 30})
	ts.Scroll(ByType[*listener](), ui.Offset{Dy: 40})
	if len(hovers) != 2 || hovers[1] != (ui.Offset{Dx: 20, Dy: 30}) {
		t.Errorf("hovers = %v", hovers)
	}
	if len(scrolls) != 1 || scrolls[0] != (ui.Offset{Dy: 40}) {
		t.Errorf("scrolls = %v", scrolls)
	}
}

func TestScrollKindAtUsesTheKindsPointer(t *testing.T) {
	ts := New(t)
	var kinds []ui.PointerDeviceKind
	var pointers []int
	l := newListener(newSwatch(grey, 300, 300))
	l.onSignal = func(e gestures.PointerSignalEvent) {
		if s, ok := e.(*gestures.PointerScrollEvent); ok {
			kinds = append(kinds, s.Kind())
			pointers = append(pointers, s.Device())
		}
	}
	ts.PumpWidget(newRow(l))
	at := ts.Center(ByType[*listener]())
	ts.ScrollKindAt(ui.PointerDeviceKindTrackpad, at, ui.Offset{Dy: 1})
	ts.ScrollKindAt(ui.PointerDeviceKindTrackpad, at, ui.Offset{Dy: 2})
	ts.ScrollKindAt(ui.PointerDeviceKindMouse, at, ui.Offset{Dy: 3})
	if len(kinds) != 3 || kinds[0] != ui.PointerDeviceKindTrackpad || kinds[2] != ui.PointerDeviceKindMouse {
		t.Fatalf("kinds = %v", kinds)
	}
	if pointers[0] != pointers[1] || pointers[0] == pointers[2] {
		t.Errorf("pointers = %v, want one per device kind", pointers)
	}
}

func TestPseudoKeysSendTheLeftKey(t *testing.T) {
	ts := New(t)
	ts.PumpWidget(newRow(newSwatch(grey, 5, 5)))
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyShift)
	pressed := ts.Binding.Keyboard().LogicalKeysPressed()
	if !pressed.Contains(services.LogicalKeyboardKeyShiftLeft) || pressed.Contains(services.LogicalKeyboardKeyShiftRight) {
		t.Errorf("pressed = %v, want ShiftLeft only", pressed)
	}
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyShift)
	if len(ts.Binding.Keyboard().LogicalKeysPressed()) != 0 {
		t.Error("a key is still pressed after the key up")
	}
}

func TestEnterTextWithoutAFieldFails(t *testing.T) {
	appErr := failure(t, func(ts *Tester) {
		ts.PumpWidget(newRow(newSwatch(grey, 5, 5)))
		ts.EnterText(ByType[*swatch](), "hi")
	})
	requireRoute(t, appErr, "EnterTextNoField")
}

func TestKeyEvents(t *testing.T) {
	ts := New(t)
	var got []ui.KeyData
	ts.d.OnKeyData = func(d ui.KeyData) bool { got = append(got, d); return d.Type == ui.KeyEventTypeDown }
	ts.Pump(250 * time.Millisecond)

	if !ts.SendKeyDownEvent(0x61, KeyOpts{Character: "a"}) {
		t.Errorf("the key down was not handled")
	}
	if ts.SendKeyUpEvent(0x61) {
		t.Errorf("the key up was handled")
	}
	if !ts.SendKeyEvent(0x62, KeyOpts{Character: "b"}) {
		t.Errorf("SendKeyEvent did not return the result of the down")
	}
	want := []ui.KeyData{
		{TimeStamp: 250 * time.Millisecond, Type: ui.KeyEventTypeDown, Logical: 0x61, Physical: 0x70004, Character: "a"},
		{TimeStamp: 250 * time.Millisecond, Type: ui.KeyEventTypeUp, Logical: 0x61, Physical: 0x70004},
		{TimeStamp: 250 * time.Millisecond, Type: ui.KeyEventTypeDown, Logical: 0x62, Physical: 0x70005, Character: "b"},
		{TimeStamp: 250 * time.Millisecond, Type: ui.KeyEventTypeUp, Logical: 0x62, Physical: 0x70005},
	}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestKeyEventsWithoutAListener(t *testing.T) {
	ts := New(t)
	ts.d.OnKeyData = nil
	if ts.SendKeyEvent(0x61) {
		t.Errorf("a key was handled with nothing listening")
	}
}

func TestStateGoPostsWorkToTheTester(t *testing.T) {
	ts := New(t)
	ts.PumpWidget(newCounter())
	s := StateOf[*counterState](ts, ByType[*counter]())
	done := make(chan struct{})
	go func() {
		s.SetState(func() { s.taps = 7 })
		close(done)
	}()
	<-done
	ts.FlushPosted()
	ts.Pump()
	if s.taps != 7 || ts.Count(ByType[*swatch]()) != 8 {
		t.Errorf("taps = %d, swatches = %d", s.taps, ts.Count(ByType[*swatch]()))
	}
	var _ widgets.State = s
}

func TestTrackpadGestureSendsPanZoomEvents(t *testing.T) {
	ts := New(t)
	var starts []*gestures.PointerPanZoomStartEvent
	var updates []*gestures.PointerPanZoomUpdateEvent
	var ends []*gestures.PointerPanZoomEndEvent
	ts.PumpWidget(newRow(widgets.NewListener(newSwatch(grey, 200, 200), widgets.ListenerOpts{
		Behavior:               rendering.HitTestBehaviorOpaque,
		OnPointerPanZoomStart:  func(e *gestures.PointerPanZoomStartEvent) { starts = append(starts, e) },
		OnPointerPanZoomUpdate: func(e *gestures.PointerPanZoomUpdateEvent) { updates = append(updates, e) },
		OnPointerPanZoomEnd:    func(e *gestures.PointerPanZoomEndEvent) { ends = append(ends, e) },
	})))
	ts.Pump(time.Second)

	g := ts.StartTrackpadGesture(ui.Offset{Dx: 10, Dy: 20})
	ts.Pump(50 * time.Millisecond)
	g.PanBy(ui.Offset{Dx: 3, Dy: 4})
	g.PanBy(ui.Offset{Dx: 1, Dy: 1})
	g.Update(ui.Offset{Dx: 4, Dy: 5}, 2, 0.5)
	g.End()

	if len(starts) != 1 || len(updates) != 3 || len(ends) != 1 {
		t.Fatalf("events: %d start, %d update, %d end", len(starts), len(updates), len(ends))
	}
	if starts[0].Position() != (ui.Offset{Dx: 10, Dy: 20}) || starts[0].TimeStamp() != time.Second {
		t.Errorf("start at %v, %v", starts[0].Position(), starts[0].TimeStamp())
	}
	tests := []struct {
		i         int
		pan, step ui.Offset
		scale     float64
		rotation  float64
	}{
		{0, ui.Offset{Dx: 3, Dy: 4}, ui.Offset{Dx: 3, Dy: 4}, 1, 0},
		{1, ui.Offset{Dx: 4, Dy: 5}, ui.Offset{Dx: 1, Dy: 1}, 1, 0},
		{2, ui.Offset{Dx: 4, Dy: 5}, ui.Offset{}, 2, 0.5},
	}
	for _, tt := range tests {
		u := updates[tt.i]
		if u.Pan() != tt.pan || u.PanDelta() != tt.step || u.Scale() != tt.scale || u.Rotation() != tt.rotation {
			t.Errorf("update %d: pan %v delta %v scale %v rotation %v", tt.i, u.Pan(), u.PanDelta(), u.Scale(), u.Rotation())
		}
		if u.Position() != (ui.Offset{Dx: 10, Dy: 20}) || u.Kind() != ui.PointerDeviceKindTrackpad {
			t.Errorf("update %d: at %v by %v", tt.i, u.Position(), u.Kind())
		}
		if u.Pointer() != starts[0].Pointer() {
			t.Errorf("update %d has the pointer %d, the start has %d", tt.i, u.Pointer(), starts[0].Pointer())
		}
	}
}

func TestTrackpadGestureMisuse(t *testing.T) {
	tests := []struct {
		name string
		do   func(g *TestTrackpadGesture)
	}{
		{"update after end", func(g *TestTrackpadGesture) { g.End(); g.PanBy(ui.Offset{Dx: 1}) }},
		{"end twice", func(g *TestTrackpadGesture) { g.End(); g.End() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := failure(t, func(ts *Tester) {
				ts.PumpWidget(newRow())
				tt.do(ts.StartTrackpadGesture(ui.Offset{}))
			})
			requireRoute(t, appErr, "GestureNotDown")
		})
	}
}

func TestTrackpadFlingHasTheSpeedAndIsInvalidWithoutOne(t *testing.T) {
	ts := New(t)
	var updates []*gestures.PointerPanZoomUpdateEvent
	ts.PumpWidget(newRow(widgets.NewListener(newSwatch(grey, 200, 200), widgets.ListenerOpts{
		Behavior:               rendering.HitTestBehaviorOpaque,
		OnPointerPanZoomUpdate: func(e *gestures.PointerPanZoomUpdateEvent) { updates = append(updates, e) },
	})))
	start := ts.Elapsed()
	ts.TrackpadFlingFrom(ui.Offset{Dx: 50, Dy: 50}, ui.Offset{Dy: -100}, 1000)
	if len(updates) != 50 {
		t.Fatalf("updates = %d, want 50", len(updates))
	}
	if got := updates[49].Pan(); got != (ui.Offset{Dy: -100}) {
		t.Errorf("total pan = %v", got)
	}
	if got := ts.Elapsed() - start; got != 100*time.Millisecond {
		t.Errorf("the fling took %v, want 100ms", got)
	}
	appErr := failure(t, func(ts *Tester) {
		ts.PumpWidget(newRow())
		ts.TrackpadFlingFrom(ui.Offset{}, ui.Offset{Dy: 1}, 0)
	})
	requireRoute(t, appErr, "FlingInvalid")
}

func TestMoveByAtSetsTheEventTimeStamp(t *testing.T) {
	ts := New(t)
	var moves []*gestures.PointerMoveEvent
	l := newListener(newSwatch(grey, 200, 200))
	l.onMove = func(e *gestures.PointerMoveEvent) { moves = append(moves, e) }
	ts.PumpWidget(newRow(l))
	g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
	g.MoveByAt(ui.Offset{Dy: 1}, 20*time.Millisecond)
	g.MoveBy(ui.Offset{Dy: 1}) // keeps the time stamp
	if len(moves) != 2 || moves[0].TimeStamp() != 20*time.Millisecond || moves[1].TimeStamp() != 20*time.Millisecond {
		t.Fatalf("moves: %v", moves)
	}
	g.Up()
}

func TestDragTouchSlopOptions(t *testing.T) {
	ts := New(t)
	var moves []*gestures.PointerMoveEvent
	l := newListener(newSwatch(grey, 200, 200))
	l.onMove = func(e *gestures.PointerMoveEvent) { moves = append(moves, e) }
	ts.PumpWidget(newRow(l))
	ts.DragFrom(ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dy: 50}, GestureOpts{TouchSlopY: foundation.Some(5.0)})
	if len(moves) != 2 || moves[0].Delta() != (ui.Offset{Dy: 5}) || moves[1].Delta() != (ui.Offset{Dy: 45}) {
		t.Fatalf("moves: %v", moves)
	}
}
