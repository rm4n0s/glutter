package gestures

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func off(x, y float64) ui.Offset { return ui.Offset{Dx: x, Dy: y} }

type tapEvents struct {
	down1, down2, down3, down4, down5, down6 *PointerDownEvent
	up1, up2, up3, up4, up5, up6             *PointerUpEvent
	move3, move4                             *PointerMoveEvent
}

// newTapEvents are the events at the top of tap_test.dart. They are new for
// each test, as the events of Dart are constants.
func newTapEvents() tapEvents {
	return tapEvents{
		down1: evDown(1, off(10, 10)), up1: evUp(1, off(11, 9)),
		down2: evDown(2, off(30, 30)), up2: evUp(2, off(31, 29)),
		down3: evDown(3, off(10, 10)), move3: evMove(3, off(25, 25)), up3: evUp(3, off(25, 25)),
		down4: evDown(4, off(10, 10)), move4: evMove(4, off(22, 22)), up4: evUp(4, off(22, 22)),
		down5: NewPointerDownEvent(PointerEventOpts{Pointer: 5, Position: off(20, 20), Buttons: foundation.Some(SecondaryButton)}),
		up5:   evUp(5, off(20, 20)),
		down6: NewPointerDownEvent(PointerEventOpts{Pointer: 6, Position: off(20, 20), Buttons: foundation.Some(TertiaryButton)}),
		up6:   evUp(6, off(20, 20)),
	}
}

type fakeArenaMember struct{}

func (fakeArenaMember) AcceptGesture(int) {}
func (fakeArenaMember) RejectGesture(int) {}

func TestTapRecognizes(t *testing.T) {
	g := newGestureTester(t)
	e := newTapEvents()
	tap := NewTapGestureRecognizer(g.binding)
	recognized := false
	tap.OnTap = func() { recognized = true }
	tap.AddPointer(e.down1)
	g.closeArena(1)
	g.route(e.down1)
	if recognized {
		t.Fatal("recognized before the up event")
	}
	g.route(e.up1)
	if !recognized {
		t.Fatal("tap not recognized")
	}
	g.binding.GestureArena.Sweep(1)
	tap.Dispose()
}

func TestTapSupportedDevicesOnly(t *testing.T) {
	g := newGestureTester(t)
	tap := NewTapGestureRecognizer(g.binding, TapGestureRecognizerOpts{
		SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindMouse, ui.PointerDeviceKindStylus},
	})
	recognized := false
	tap.OnTap = func() { recognized = true }
	for _, tc := range []struct {
		kind ui.PointerDeviceKind
		want bool
	}{
		{ui.PointerDeviceKindTouch, false},
		{ui.PointerDeviceKindMouse, true},
		{ui.PointerDeviceKindStylus, true},
	} {
		recognized = false
		down := NewPointerDownEvent(PointerEventOpts{Kind: tc.kind, Pointer: 1, Position: off(10, 10)})
		up := NewPointerUpEvent(PointerEventOpts{Kind: tc.kind, Pointer: 1, Position: off(11, 9)})
		tap.AddPointer(down)
		g.closeArena(1)
		g.route(down)
		if recognized {
			t.Fatalf("%v: recognized before up", tc.kind)
		}
		g.route(up)
		if recognized != tc.want {
			t.Errorf("%v: recognized = %v, want %v", tc.kind, recognized, tc.want)
		}
		g.binding.GestureArena.Sweep(1)
	}
	tap.Dispose()
}

func TestTapDetailsContainDeviceKind(t *testing.T) {
	g := newGestureTester(t)
	tap := NewTapGestureRecognizer(g.binding)
	var down TapDownDetails
	var up TapUpDetails
	tap.OnTapDown = func(d TapDownDetails) { down = d }
	tap.OnTapUp = func(d TapUpDetails) { up = d }
	d := NewPointerDownEvent(PointerEventOpts{Pointer: 1, Kind: ui.PointerDeviceKindMouse})
	u := NewPointerUpEvent(PointerEventOpts{Pointer: 1, Kind: ui.PointerDeviceKindMouse})
	tap.AddPointer(d)
	g.closeArena(1)
	g.route(d)
	if k, _ := down.Kind.Get(); k != ui.PointerDeviceKindMouse {
		t.Errorf("down kind = %v", down.Kind)
	}
	g.route(u)
	if up.Kind != ui.PointerDeviceKindMouse {
		t.Errorf("up kind = %v", up.Kind)
	}
	tap.Dispose()
}

func TestTapNoDuplicateEvents(t *testing.T) {
	g := newGestureTester(t)
	e := newTapEvents()
	tap := NewTapGestureRecognizer(g.binding)
	taps := 0
	tap.OnTap = func() { taps++ }
	for want := 1; want <= 2; want++ {
		tap.AddPointer(e.down1)
		g.closeArena(1)
		g.route(e.down1)
		if taps != want-1 {
			t.Fatalf("taps = %d before up, want %d", taps, want-1)
		}
		g.route(e.up1)
		g.binding.GestureArena.Sweep(1)
		if taps != want {
			t.Fatalf("taps = %d, want %d", taps, want)
		}
	}
	tap.Dispose()
}

func TestTapOverlappingTaps(t *testing.T) {
	for _, tc := range []struct {
		name string
		fifo bool
	}{{"FIFO", true}, {"FILO", false}} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			e := newTapEvents()
			tap := NewTapGestureRecognizer(g.binding)
			taps := 0
			tap.OnTap = func() { taps++ }
			tap.AddPointer(e.down1)
			g.closeArena(1)
			g.route(e.down1)
			tap.AddPointer(e.down2)
			g.closeArena(2)
			g.route(e.down1)
			if taps != 0 {
				t.Fatalf("taps = %d", taps)
			}
			if tc.fifo {
				g.route(e.up1)
				if taps != 1 {
					t.Fatalf("taps = %d after the first up", taps)
				}
				g.binding.GestureArena.Sweep(1)
				g.route(e.up2)
				g.binding.GestureArena.Sweep(2)
			} else {
				g.route(e.up2)
				g.binding.GestureArena.Sweep(2)
				if taps != 0 {
					t.Fatalf("taps = %d after the second up", taps)
				}
				g.route(e.up1)
				if taps != 1 {
					t.Fatalf("taps = %d after the first up", taps)
				}
				g.binding.GestureArena.Sweep(1)
			}
			if taps != 1 {
				t.Errorf("taps = %d, want 1", taps)
			}
			tap.Dispose()
		})
	}
}

func TestTapDistance(t *testing.T) {
	for _, tc := range []struct {
		name         string
		pointer      int
		wantRecog    bool
		wantCanceled bool
	}{
		{"distance cancels tap", 3, false, true},
		{"short distance does not cancel tap", 4, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			e := newTapEvents()
			down, move, up := e.down3, e.move3, e.up3
			if tc.pointer == 4 {
				down, move, up = e.down4, e.move4, e.up4
			}
			tap := NewTapGestureRecognizer(g.binding)
			recognized, canceled := false, false
			tap.OnTap = func() { recognized = true }
			tap.OnTapCancel = func() { canceled = true }
			tap.AddPointer(down)
			g.closeArena(tc.pointer)
			g.route(down)
			if recognized || canceled {
				t.Fatal("something happened before the move")
			}
			g.route(move)
			if recognized || canceled != tc.wantCanceled {
				t.Fatalf("after the move: recognized = %v, canceled = %v", recognized, canceled)
			}
			g.route(up)
			g.binding.GestureArena.Sweep(tc.pointer)
			if recognized != tc.wantRecog || canceled != tc.wantCanceled {
				t.Errorf("recognized = %v (want %v), canceled = %v (want %v)", recognized, tc.wantRecog, canceled, tc.wantCanceled)
			}
			tap.Dispose()
		})
	}
}

func TestTapTimeoutDoesNotCancel(t *testing.T) {
	g := newGestureTester(t)
	e := newTapEvents()
	tap := NewTapGestureRecognizer(g.binding)
	recognized := false
	tap.OnTap = func() { recognized = true }
	tap.AddPointer(e.down1)
	g.closeArena(1)
	g.route(e.down1)
	g.elapse(500 * time.Millisecond)
	if recognized {
		t.Fatal("recognized by the timeout")
	}
	g.route(e.up1)
	if !recognized {
		t.Error("not recognized")
	}
	g.binding.GestureArena.Sweep(1)
	tap.Dispose()
}

func TestTapYieldsToOtherArenaMembers(t *testing.T) {
	for _, tc := range []struct {
		name string
		disp GestureDisposition
		want bool
	}{
		{"should yield to other arena members", GestureAccepted, false},
		{"should trigger on release of held arena", GestureRejected, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			e := newTapEvents()
			tap := NewTapGestureRecognizer(g.binding)
			recognized := false
			tap.OnTap = func() { recognized = true }
			tap.AddPointer(e.down1)
			entry := g.binding.GestureArena.Add(1, fakeArenaMember{})
			g.binding.GestureArena.Hold(1)
			g.closeArena(1)
			g.route(e.down1)
			g.route(e.up1)
			g.binding.GestureArena.Sweep(1)
			if recognized {
				t.Fatal("recognized while the arena was held")
			}
			entry.Resolve(tc.disp)
			g.flush()
			if recognized != tc.want {
				t.Errorf("recognized = %v, want %v", recognized, tc.want)
			}
			tap.Dispose()
		})
	}
}

func newLoggingTap(g *gestureTester, l *log) *TapGestureRecognizer {
	tap := NewTapGestureRecognizer(g.binding)
	tap.OnTapDown = func(TapDownDetails) { l.add("down") }
	tap.OnTapUp = func(TapUpDetails) { l.add("up") }
	tap.OnTap = func() { l.add("tap") }
	tap.OnTapCancel = func() { l.add("cancel") }
	return tap
}

func TestTapPointerCancel(t *testing.T) {
	for _, withDrag := range []bool{false, true} {
		name := "cancels tap"
		if withDrag {
			name = "after exceeding the deadline cancels tap"
		}
		t.Run(name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			down := evDown(5, off(10, 10))
			cancel := NewPointerCancelEvent(PointerEventOpts{Pointer: 5, Position: off(10, 10)})
			tap := newLoggingTap(g, &l)
			var drag *HorizontalDragGestureRecognizer
			if withDrag {
				drag = NewHorizontalDragGestureRecognizer(g.binding)
				drag.OnStart = func(DragStartDetails) {} // needs a callback to compete
				tap.AddPointer(down)
				drag.AddPointer(down)
				g.closeArena(5)
				g.route(down)
				l.check(t)
				g.elapse(time.Second)
			} else {
				tap.AddPointer(down)
				g.closeArena(5)
				g.elapse(5 * time.Second)
			}
			l.check(t, "down")
			g.route(cancel)
			l.check(t, "down", "cancel")
			tap.Dispose()
			if drag != nil {
				drag.Dispose()
			}
		})
	}
}

func TestTapLosingDoesNotSendCancel(t *testing.T) {
	g := newGestureTester(t)
	e := newTapEvents()
	var l log
	tap := newLoggingTap(g, &l)
	drag := NewHorizontalDragGestureRecognizer(g.binding)
	tap.AddPointer(e.down3)
	drag.AddPointer(e.down3)
	g.closeArena(3)
	g.route(e.move3)
	g.binding.GestureArena.Sweep(3)
	l.check(t)
	tap.Dispose()
	drag.Dispose()
}

func TestTapNonPrimaryPointerDoesNotTriggerTimeout(t *testing.T) {
	g := newGestureTester(t)
	e := newTapEvents()
	var l log
	tap := newLoggingTap(g, &l)
	tap.AddPointer(e.down1)
	g.closeArena(1)
	tap.AddPointer(e.down2)
	g.closeArena(2)
	l.check(t)
	g.route(e.up1)
	g.binding.GestureArena.Sweep(1)
	l.check(t, "down", "up", "tap")
	l.clear()
	g.elapse(200 * time.Millisecond)
	l.check(t)
	g.route(e.up2)
	g.binding.GestureArena.Sweep(2)
	l.check(t)
	tap.Dispose()
}

func TestTapConsistentButtons(t *testing.T) {
	both := func() *PointerMoveEvent {
		return NewPointerMoveEvent(PointerEventOpts{Pointer: 1, Position: off(10, 10), Buttons: foundation.Some(PrimaryMouseButton | SecondaryMouseButton)})
	}
	right := func() *PointerMoveEvent {
		return NewPointerMoveEvent(PointerEventOpts{Pointer: 1, Position: off(10, 10), Buttons: foundation.Some(SecondaryMouseButton)})
	}
	newTap := func(g *gestureTester, l *log) *TapGestureRecognizer {
		tap := NewTapGestureRecognizer(g.binding)
		tap.OnTapDown = func(TapDownDetails) { l.add("down") }
		tap.OnTapUp = func(TapUpDetails) { l.add("up") }
		tap.OnTapCancel = func() { l.add("cancel") }
		return tap
	}
	t.Run("changing buttons before TapDown cancels without sending cancel", func(t *testing.T) {
		g := newGestureTester(t)
		e := newTapEvents()
		var l log
		tap := newTap(g, &l)
		tap.AddPointer(e.down1)
		g.closeArena(1)
		g.route(both())
		g.route(right())
		g.route(e.up1)
		l.check(t)
		tap.Dispose()
	})
	t.Run("changing buttons before TapDown does not prevent the next tap", func(t *testing.T) {
		g := newGestureTester(t)
		e := newTapEvents()
		var l log
		tap := newTap(g, &l)
		tap.AddPointer(e.down1)
		g.closeArena(1)
		g.route(both())
		g.route(right())
		g.route(e.up1)
		l.check(t)
		tap.AddPointer(e.down2)
		g.closeArena(2)
		g.elapse(time.Second)
		g.route(e.up2)
		l.check(t, "down", "up")
		tap.Dispose()
	})
	t.Run("changing buttons after TapDown cancels and sends cancel", func(t *testing.T) {
		g := newGestureTester(t)
		e := newTapEvents()
		var l log
		tap := newTap(g, &l)
		tap.AddPointer(e.down1)
		g.closeArena(1)
		l.check(t)
		g.elapse(time.Second)
		l.check(t, "down")
		g.route(both())
		l.check(t, "down", "cancel")
		g.route(right())
		g.route(e.up1)
		l.check(t, "down", "cancel")
		tap.Dispose()
	})
	t.Run("changing buttons after TapDown does not prevent the next tap", func(t *testing.T) {
		g := newGestureTester(t)
		e := newTapEvents()
		var l log
		tap := newTap(g, &l)
		tap.AddPointer(e.down1)
		g.closeArena(1)
		g.elapse(time.Second)
		g.route(both())
		g.route(right())
		g.route(e.up1)
		g.binding.GestureArena.Sweep(1)
		l.check(t, "down", "cancel")
		tap.AddPointer(e.down2)
		g.closeArena(2)
		g.elapse(time.Second)
		g.route(e.up2)
		g.binding.GestureArena.Sweep(2)
		l.check(t, "down", "cancel", "down", "up")
		tap.Dispose()
	})
}

func TestTapRecognizersOfDifferentButtonsDoNotCompete(t *testing.T) {
	setup := func(g *gestureTester, l *log) (primary, primary2, secondary, tertiary *TapGestureRecognizer) {
		primary = NewTapGestureRecognizer(g.binding)
		primary.OnTapDown = func(TapDownDetails) { l.add("primaryDown") }
		primary.OnTapUp = func(TapUpDetails) { l.add("primaryUp") }
		primary.OnTapCancel = func() { l.add("primaryCancel") }
		primary2 = NewTapGestureRecognizer(g.binding)
		primary2.OnTapDown = func(TapDownDetails) { l.add("primary2Down") }
		primary2.OnTapUp = func(TapUpDetails) { l.add("primary2Up") }
		primary2.OnTapCancel = func() { l.add("primary2Cancel") }
		secondary = NewTapGestureRecognizer(g.binding)
		secondary.OnSecondaryTapDown = func(TapDownDetails) { l.add("secondaryDown") }
		secondary.OnSecondaryTapUp = func(TapUpDetails) { l.add("secondaryUp") }
		secondary.OnSecondaryTapCancel = func() { l.add("secondaryCancel") }
		tertiary = NewTapGestureRecognizer(g.binding)
		tertiary.OnTertiaryTapDown = func(TapDownDetails) { l.add("tertiaryDown") }
		tertiary.OnTertiaryTapUp = func(TapUpDetails) { l.add("tertiaryUp") }
		tertiary.OnTertiaryTapCancel = func() { l.add("tertiaryCancel") }
		return
	}
	for _, other := range []string{"secondary", "tertiary"} {
		t.Run("a primary tap recognizer does not compete with a "+other+" one", func(t *testing.T) {
			g := newGestureTester(t)
			e := newTapEvents()
			var l log
			primary, _, secondary, tertiary := setup(g, &l)
			primary.AddPointer(e.down1)
			if other == "secondary" {
				secondary.AddPointer(e.down1)
			} else {
				tertiary.AddPointer(e.down1)
			}
			g.closeArena(1)
			g.route(e.down1)
			l.check(t, "primaryDown")
			l.clear()
			g.route(e.up1)
			l.check(t, "primaryUp")
		})
	}
	t.Run("a primary tap recognizer competes with another primary one", func(t *testing.T) {
		g := newGestureTester(t)
		e := newTapEvents()
		var l log
		primary, primary2, _, _ := setup(g, &l)
		primary.AddPointer(e.down1)
		primary2.AddPointer(e.down1)
		g.closeArena(1)
		g.route(e.down1)
		l.check(t)
		g.elapse(500 * time.Millisecond)
		l.check(t, "primaryDown", "primary2Down")
	})
}

func TestTapButtonsTriggerTheirCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		down     func(e tapEvents) *PointerDownEvent
		finish   func(e tapEvents) PointerEvent
		pointer  int
		wantDown string
		wantEnd  []string
	}{
		{"primary tap", func(e tapEvents) *PointerDownEvent { return e.down1 }, func(e tapEvents) PointerEvent { return e.up1 }, 1, "primaryDown", []string{"primaryUp", "primary"}},
		{"primary cancel", func(e tapEvents) *PointerDownEvent { return e.down1 }, func(tapEvents) PointerEvent { return NewPointerCancelEvent(PointerEventOpts{Pointer: 1}) }, 1, "primaryDown", []string{"primaryCancel"}},
		{"secondary tap", func(e tapEvents) *PointerDownEvent { return e.down5 }, func(e tapEvents) PointerEvent { return e.up5 }, 5, "secondaryDown", []string{"secondaryUp"}},
		{"tertiary tap", func(e tapEvents) *PointerDownEvent { return e.down6 }, func(e tapEvents) PointerEvent { return e.up6 }, 6, "tertiaryDown", []string{"tertiaryUp"}},
		{"secondary cancel", func(e tapEvents) *PointerDownEvent { return e.down5 }, func(tapEvents) PointerEvent { return NewPointerCancelEvent(PointerEventOpts{Pointer: 5}) }, 5, "secondaryDown", []string{"secondaryCancel"}},
		{"tertiary cancel", func(e tapEvents) *PointerDownEvent { return e.down6 }, func(tapEvents) PointerEvent { return NewPointerCancelEvent(PointerEventOpts{Pointer: 6}) }, 6, "tertiaryDown", []string{"tertiaryCancel"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			e := newTapEvents()
			var l log
			tap := NewTapGestureRecognizer(g.binding)
			tap.OnTapDown = func(TapDownDetails) { l.add("primaryDown") }
			tap.OnTap = func() { l.add("primary") }
			tap.OnTapUp = func(TapUpDetails) { l.add("primaryUp") }
			tap.OnTapCancel = func() { l.add("primaryCancel") }
			tap.OnSecondaryTapDown = func(TapDownDetails) { l.add("secondaryDown") }
			tap.OnSecondaryTapUp = func(TapUpDetails) { l.add("secondaryUp") }
			tap.OnSecondaryTapCancel = func() { l.add("secondaryCancel") }
			tap.OnTertiaryTapDown = func(TapDownDetails) { l.add("tertiaryDown") }
			tap.OnTertiaryTapUp = func(TapUpDetails) { l.add("tertiaryUp") }
			tap.OnTertiaryTapCancel = func() { l.add("tertiaryCancel") }
			tap.AddPointer(tc.down(e))
			g.closeArena(tc.pointer)
			l.check(t)
			g.elapse(500 * time.Millisecond)
			l.check(t, tc.wantDown)
			l.clear()
			g.route(tc.finish(e))
			g.binding.GestureArena.Sweep(tc.pointer)
			l.check(t, tc.wantEnd...)
			tap.Dispose()
		})
	}
}

func TestTapSecondTapAfterRejectionIsIgnored(t *testing.T) {
	g := newGestureTester(t)
	didTap := false
	tap := NewTapGestureRecognizer(g.binding)
	tap.OnTap = func() { didTap = true }
	drag := NewHorizontalDragGestureRecognizer(g.binding)
	drag.OnStart = func(DragStartDetails) {}
	pointer1 := newTestPointer()
	down := pointer1.down(ui.Offset{})
	drag.AddPointer(down)
	tap.AddPointer(down)
	g.closeArena(1)
	g.route(down)
	g.route(pointer1.move(off(50, 0)))
	pointer2 := newTestPointer(2)
	down2 := pointer2.down(off(10, 20))
	drag.AddPointer(down2)
	tap.AddPointer(down2)
	g.closeArena(2)
	g.route(down2)
	if didTap {
		t.Error("tapped")
	}
	tap.Dispose()
	drag.Dispose()
}

func TestTapMove(t *testing.T) {
	g := newGestureTester(t)
	var details *TapMoveDetails
	tap := NewTapGestureRecognizer(g.binding, TapGestureRecognizerOpts{PostAcceptSlopTolerance: SlopToleranceNone})
	tap.OnTapMove = func(d TapMoveDetails) { details = &d }
	pointer1 := newTestPointer()
	down := pointer1.down(ui.Offset{})
	tap.AddPointer(down)
	g.closeArena(1)
	g.route(down)
	g.route(pointer1.move(off(50, 0)))
	if details == nil || details.GlobalPosition != off(50, 0) || details.Delta != off(50, 0) {
		t.Fatalf("details = %+v", details)
	}
	details = nil
	g.route(pointer1.move(off(60, 10)))
	if details == nil || details.GlobalPosition != off(60, 10) || details.Delta != off(10, 10) {
		t.Fatalf("details = %+v", details)
	}
	tap.Dispose()
}
