package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

type acceptRejectMember struct{ accepted, rejected bool }

func (m *acceptRejectMember) AcceptGesture(int) { m.accepted = true }
func (m *acceptRejectMember) RejectGesture(int) { m.rejected = true }

// doubleTapFixture is the setUp of double_tap_test.dart.
type doubleTapFixture struct {
	t            *testing.T
	g            *gestureTester
	tap          *DoubleTapGestureRecognizer
	recognized   bool
	downDetails  *TapDownDetails
	canceled     bool
	down1, down2 *PointerDownEvent
	up1, up2     *PointerUpEvent
	down3, down4 *PointerDownEvent
	up3, up4     *PointerUpEvent
	move4        *PointerMoveEvent
	down5, down6 *PointerDownEvent
	up5, up6     *PointerUpEvent
}

func newDoubleTapFixture(t *testing.T) *doubleTapFixture {
	f := &doubleTapFixture{t: t, g: newGestureTester(t)}
	f.tap = f.newRecognizer(DoubleTapGestureRecognizerOpts{})
	f.down1, f.up1 = evDown(1, off(10, 10)), evUp(1, off(11, 9))
	f.down2, f.up2 = evDown(2, off(12, 12)), evUp(2, off(13, 11))
	f.down3, f.up3 = evDown(3, off(130, 130)), evUp(3, off(131, 129))
	f.down4, f.move4, f.up4 = evDown(4, off(10, 10)), evMove(4, off(25, 25)), evUp(4, off(25, 25))
	f.down5, f.up5 = evDown(5, off(10, 10)), evUp(5, off(11, 9))
	f.down6 = NewPointerDownEvent(PointerEventOpts{Pointer: 6, Position: off(10, 10), Buttons: foundation.Some(SecondaryMouseButton)})
	f.up6 = evUp(6, off(11, 9))
	return f
}

func (f *doubleTapFixture) newRecognizer(opts DoubleTapGestureRecognizerOpts) *DoubleTapGestureRecognizer {
	tap := NewDoubleTapGestureRecognizer(f.g.binding, opts)
	tap.OnDoubleTap = func() {
		if f.recognized {
			f.t.Error("double tap recognized twice")
		}
		f.recognized = true
	}
	tap.OnDoubleTapDown = func(d TapDownDetails) {
		if f.downDetails != nil {
			f.t.Error("double tap down sent twice")
		}
		f.downDetails = &d
	}
	tap.OnDoubleTapCancel = func() {
		if f.canceled {
			f.t.Error("double tap canceled twice")
		}
		f.canceled = true
	}
	return tap
}

// press is the sequence addPointer, closeArena, route(down), route(up), sweep.
func (f *doubleTapFixture) press(tap *DoubleTapGestureRecognizer, down *PointerDownEvent, up *PointerUpEvent) {
	tap.AddPointer(down)
	f.g.closeArena(down.Pointer())
	f.g.route(down)
	f.g.route(up)
	f.g.binding.GestureArena.Sweep(down.Pointer())
}

func (f *doubleTapFixture) expect(recognized bool, hasDown bool, canceled bool) {
	f.t.Helper()
	if f.recognized != recognized || (f.downDetails != nil) != hasDown || f.canceled != canceled {
		f.t.Errorf("recognized = %v (want %v), down details = %v (want %v), canceled = %v (want %v)",
			f.recognized, recognized, f.downDetails != nil, hasDown, f.canceled, canceled)
	}
}

func (f *doubleTapFixture) expectDown(e *PointerDownEvent) {
	f.t.Helper()
	if f.downDetails == nil {
		f.t.Fatal("no double tap down")
	}
	if f.downDetails.GlobalPosition != e.Position() || f.downDetails.LocalPosition != e.LocalPosition() {
		f.t.Errorf("down details = %+v", *f.downDetails)
	}
}

func TestDoubleTapRecognizes(t *testing.T) {
	f := newDoubleTapFixture(t)
	f.press(f.tap, f.down1, f.up1)
	f.expect(false, false, false)
	f.g.elapse(ms(100))
	f.tap.AddPointer(f.down2)
	f.g.closeArena(2)
	f.expectDown(f.down2)
	f.g.route(f.down2)
	f.expect(false, true, false)
	f.g.route(f.up2)
	f.expect(true, true, false)
	f.g.binding.GestureArena.Sweep(2)
	f.expect(true, true, false)
}

func TestDoubleTapWithSecondaryButton(t *testing.T) {
	f := newDoubleTapFixture(t)
	tap := f.newRecognizer(DoubleTapGestureRecognizerOpts{AllowedButtonsFilter: func(b int) bool { return b == SecondaryButton }})
	down7 := NewPointerDownEvent(PointerEventOpts{Pointer: 7, Position: off(10, 10), Buttons: foundation.Some(SecondaryMouseButton)})
	up7 := evUp(7, off(11, 9))
	f.press(tap, f.down6, f.up6)
	f.expect(false, false, false)
	f.g.elapse(ms(100))
	tap.AddPointer(down7)
	f.g.closeArena(7)
	f.expectDown(down7)
	f.g.route(down7)
	f.expect(false, true, false)
	f.g.route(up7)
	f.expect(true, true, false)
}

func TestDoubleTapDoesNotRecognize(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(f *doubleTapFixture)
	}{
		{"inter-tap distance", func(f *doubleTapFixture) {
			f.press(f.tap, f.down1, f.up1)
			f.press(f.tap, f.down3, f.up3)
		}},
		{"intra-tap distance", func(f *doubleTapFixture) {
			f.tap.AddPointer(f.down4)
			f.g.closeArena(4)
			f.g.route(f.down4)
			f.g.route(f.move4)
			f.g.route(f.up4)
			f.g.binding.GestureArena.Sweep(4)
			f.tap.AddPointer(f.down1)
			f.g.closeArena(1)
			f.g.route(f.down2)
			f.g.route(f.up1)
			f.g.binding.GestureArena.Sweep(1)
		}},
		{"inter-tap delay", func(f *doubleTapFixture) {
			f.press(f.tap, f.down1, f.up1)
			f.g.elapse(ms(5000))
			f.press(f.tap, f.down2, f.up2)
		}},
		{"over-rapid taps", func(f *doubleTapFixture) {
			f.press(f.tap, f.down1, f.up1)
			f.g.elapse(ms(10))
			f.press(f.tap, f.down2, f.up2)
		}},
		{"two overlapping taps", func(f *doubleTapFixture) {
			f.tap.AddPointer(f.down1)
			f.g.closeArena(1)
			f.g.route(f.down1)
			f.tap.AddPointer(f.down2)
			f.g.closeArena(2)
			f.g.route(f.down1)
			f.g.route(f.up1)
			f.g.binding.GestureArena.Sweep(1)
			f.g.route(f.up2)
			f.g.binding.GestureArena.Sweep(2)
		}},
		{"a button change interrupts the sequence", func(f *doubleTapFixture) {
			f.press(f.tap, f.down1, f.up1)
			f.g.elapse(ms(100))
			f.press(f.tap, f.down6, f.up6)
			f.g.elapse(ms(100))
			f.expect(false, false, false)
			f.press(f.tap, f.down2, f.up2)
		}},
		{"a button change with a filter interrupts the sequence", func(f *doubleTapFixture) {
			tap := f.newRecognizer(DoubleTapGestureRecognizerOpts{AllowedButtonsFilter: func(b int) bool { return b == PrimaryButton }})
			f.press(tap, f.down1, f.up1)
			f.g.elapse(ms(100))
			f.press(tap, f.down6, f.up6)
			f.g.elapse(ms(100))
			f.expect(false, false, false)
			f.press(tap, f.down2, f.up2)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDoubleTapFixture(t)
			tc.run(f)
			f.expect(false, false, false)
			f.tap.Dispose()
		})
	}
}

func TestDoubleTapRecognizesAfterReset(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first func(f *doubleTapFixture)
	}{
		{"inter-tap delay resets double tap, allowing the third tap to be a double tap", func(f *doubleTapFixture) {
			f.press(f.tap, f.down1, f.up1)
			f.g.elapse(ms(5000))
			f.press(f.tap, f.down2, f.up2)
		}},
		{"over-rapid taps reset double tap, allowing the third tap to be a double tap", func(f *doubleTapFixture) {
			f.press(f.tap, f.down1, f.up1)
			f.g.elapse(ms(10))
			f.press(f.tap, f.down2, f.up2)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDoubleTapFixture(t)
			tc.first(f)
			f.expect(false, false, false)
			f.g.elapse(ms(100))
			f.tap.AddPointer(f.down5)
			f.g.closeArena(5)
			f.g.route(f.down5)
			f.expectDown(f.down5)
			f.expect(false, true, false)
			f.g.route(f.up5)
			f.expect(true, true, false)
			f.g.binding.GestureArena.Sweep(5)
			f.expect(true, true, false)
		})
	}
}

func TestDoubleTapIntraTapDelayDoesNotCancel(t *testing.T) {
	f := newDoubleTapFixture(t)
	f.tap.AddPointer(f.down1)
	f.g.closeArena(1)
	f.g.route(f.down1)
	f.g.elapse(ms(1000))
	f.g.route(f.up1)
	f.g.binding.GestureArena.Sweep(1)
	f.expect(false, false, false)
	f.tap.AddPointer(f.down2)
	f.g.closeArena(2)
	f.g.route(f.down2)
	f.expectDown(f.down2)
	f.g.route(f.up2)
	f.expect(true, true, false)
	f.g.binding.GestureArena.Sweep(2)
	f.expect(true, true, false)
}

func TestDoubleTapOneTapOfGroupFollowedBySecondTap(t *testing.T) {
	f := newDoubleTapFixture(t)
	f.tap.AddPointer(f.down1)
	f.g.closeArena(1)
	f.g.route(f.down1)
	f.tap.AddPointer(f.down2)
	f.g.closeArena(2)
	f.g.route(f.down1)
	f.g.route(f.up1)
	f.g.binding.GestureArena.Sweep(1)
	f.g.route(f.up2)
	f.g.binding.GestureArena.Sweep(2)
	f.expect(false, false, false)
	f.g.elapse(ms(100))
	f.tap.AddPointer(f.down1)
	f.g.closeArena(1)
	f.g.route(f.down1)
	f.expectDown(f.down1)
	f.expect(false, true, false)
	f.g.route(f.up1)
	f.expect(true, true, false)
	f.g.binding.GestureArena.Sweep(1)
	f.expect(true, true, false)
}

func TestDoubleTapArenaRejection(t *testing.T) {
	t.Run("during the first tap", func(t *testing.T) {
		f := newDoubleTapFixture(t)
		f.tap.AddPointer(f.down1)
		member := &acceptRejectMember{}
		entry := f.g.binding.GestureArena.Add(1, member)
		f.g.closeArena(1)
		f.g.route(f.down1)
		f.g.route(f.up1)
		entry.Resolve(GestureAccepted)
		if !member.accepted {
			t.Error("the member did not win")
		}
		f.g.binding.GestureArena.Sweep(1)
		f.press(f.tap, f.down2, f.up2)
		f.expect(false, false, false)
	})
	t.Run("between taps", func(t *testing.T) {
		f := newDoubleTapFixture(t)
		f.tap.AddPointer(f.down1)
		member := &acceptRejectMember{}
		entry := f.g.binding.GestureArena.Add(1, member)
		f.g.closeArena(1)
		f.g.route(f.down1)
		f.g.route(f.up1)
		f.g.binding.GestureArena.Sweep(1)
		entry.Resolve(GestureAccepted)
		if !member.accepted {
			t.Error("the member did not win")
		}
		f.press(f.tap, f.down2, f.up2)
		f.expect(false, false, false)
	})
	t.Run("during the last tap", func(t *testing.T) {
		f := newDoubleTapFixture(t)
		f.tap.AddPointer(f.down1)
		member := &acceptRejectMember{}
		entry := f.g.binding.GestureArena.Add(1, member)
		f.g.closeArena(1)
		f.g.route(f.down1)
		f.g.route(f.up1)
		f.g.binding.GestureArena.Sweep(1)
		f.expect(false, false, false)
		f.g.elapse(ms(100))
		f.tap.AddPointer(f.down2)
		f.g.closeArena(2)
		f.g.route(f.down2)
		f.expectDown(f.down2)
		f.expect(false, true, false)
		entry.Resolve(GestureAccepted)
		if !member.accepted {
			t.Error("the member did not win")
		}
		f.expect(false, true, true)
		f.g.route(f.up2)
		f.g.binding.GestureArena.Sweep(2)
		f.expect(false, true, true)
	})
	t.Run("a passive gesture triggers when the double tap times out", func(t *testing.T) {
		f := newDoubleTapFixture(t)
		f.tap.AddPointer(f.down1)
		member := &acceptRejectMember{}
		f.g.binding.GestureArena.Add(1, member)
		f.g.closeArena(1)
		f.g.route(f.down1)
		f.g.route(f.up1)
		f.g.binding.GestureArena.Sweep(1)
		if member.accepted {
			t.Fatal("accepted too early")
		}
		f.g.elapse(ms(5000))
		if !member.accepted {
			t.Error("the passive member was not accepted")
		}
		f.expect(false, false, false)
	})
}

func TestDoubleTapButtonChangeStartsValidSequence(t *testing.T) {
	f := newDoubleTapFixture(t)
	f.press(f.tap, f.down6, f.up6)
	f.g.elapse(ms(100))
	f.press(f.tap, f.down1, f.up1)
	f.expect(false, false, false)
	f.g.elapse(ms(100))
	f.tap.AddPointer(f.down2)
	f.g.closeArena(2)
	f.g.route(f.down2)
	f.expectDown(f.down2)
	f.g.route(f.up2)
	f.g.binding.GestureArena.Sweep(2)
	f.expect(true, true, false)
}

func TestDoubleTapDoesNotCompeteAcrossButtons(t *testing.T) {
	setup := func(f *doubleTapFixture, l *log) (tapPrimary, tapSecondary *TapGestureRecognizer, doubleTap *DoubleTapGestureRecognizer) {
		tapPrimary = NewTapGestureRecognizer(f.g.binding)
		tapPrimary.OnTapDown = func(TapDownDetails) { l.add("tapPrimary") }
		tapSecondary = NewTapGestureRecognizer(f.g.binding)
		tapSecondary.OnSecondaryTapDown = func(TapDownDetails) { l.add("tapSecondary") }
		doubleTap = NewDoubleTapGestureRecognizer(f.g.binding)
		doubleTap.OnDoubleTap = func() { l.add("doubleTap") }
		return
	}
	t.Run("a primary double tap does not compete with a secondary tap", func(t *testing.T) {
		f := newDoubleTapFixture(t)
		var l log
		_, tapSecondary, doubleTap := setup(f, &l)
		doubleTap.AddPointer(f.down6)
		tapSecondary.AddPointer(f.down6)
		f.g.closeArena(6)
		f.g.route(f.down6)
		l.check(t, "tapSecondary")
	})
	t.Run("a primary double tap competes with a primary tap", func(t *testing.T) {
		f := newDoubleTapFixture(t)
		var l log
		tapPrimary, _, doubleTap := setup(f, &l)
		doubleTap.AddPointer(f.down1)
		tapPrimary.AddPointer(f.down1)
		f.g.closeArena(1)
		f.g.route(f.down1)
		l.check(t)
		f.g.elapse(ms(300))
		l.check(t, "tapPrimary")
	})
}

func TestDoubleTapSecondaryDoesNotTriggerPrimary(t *testing.T) {
	f := newDoubleTapFixture(t)
	var l log
	doubleTap := NewDoubleTapGestureRecognizer(f.g.binding)
	doubleTap.OnDoubleTap = func() { l.add("primary") }
	down7 := NewPointerDownEvent(PointerEventOpts{Pointer: 7, Position: off(10, 10), Buttons: foundation.Some(SecondaryMouseButton)})
	doubleTap.AddPointer(f.down6)
	f.g.closeArena(6)
	f.g.route(f.down6)
	f.g.route(f.up6)
	f.g.binding.GestureArena.Sweep(6)
	f.g.elapse(ms(100))
	doubleTap.AddPointer(down7)
	f.g.closeArena(7)
	f.g.route(down7)
	f.g.route(evUp(7, off(11, 9)))
	l.check(t)
	doubleTap.Dispose()
}

func TestDoubleTapButtonsFilterCancelsInvalidTaps(t *testing.T) {
	f := newDoubleTapFixture(t)
	var l log
	doubleTap := NewDoubleTapGestureRecognizer(f.g.binding, DoubleTapGestureRecognizerOpts{AllowedButtonsFilter: func(int) bool { return false }})
	doubleTap.OnDoubleTap = func() { l.add("primary") }
	f.press(doubleTap, f.down1, f.up1)
	f.g.elapse(ms(100))
	f.press(doubleTap, f.down2, f.up2)
	l.check(t)
	doubleTap.Dispose()
}
