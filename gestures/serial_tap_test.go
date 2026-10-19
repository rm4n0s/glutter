package gestures

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

var serialTapDelay = DoubleTapTimeout / 2

// winningGestureRecognizer wins the arena at the first event.
type winningGestureRecognizer struct {
	PrimaryPointerGestureRecognizer
}

func newWinning(b *GestureBinding) *winningGestureRecognizer {
	r := &winningGestureRecognizer{}
	r.InitPrimaryPointer(b, r, PrimaryPointerOpts{})
	return r
}
func (r *winningGestureRecognizer) DebugDescription() string          { return "winner" }
func (r *winningGestureRecognizer) HandlePrimaryPointer(PointerEvent) { r.Resolve(GestureAccepted) }

// releaseGestureRecognizer wins the arena when the pointer is lifted.
type releaseGestureRecognizer struct {
	PrimaryPointerGestureRecognizer
	onRelease func()
}

func newRelease(b *GestureBinding) *releaseGestureRecognizer {
	r := &releaseGestureRecognizer{}
	r.InitPrimaryPointer(b, r, PrimaryPointerOpts{})
	return r
}
func (r *releaseGestureRecognizer) DebugDescription() string { return "release" }
func (r *releaseGestureRecognizer) HandlePrimaryPointer(e PointerEvent) {
	if _, ok := e.(*PointerUpEvent); ok {
		r.Resolve(GestureAccepted)
		if r.onRelease != nil {
			r.onRelease()
		}
	}
}

type serialFixture struct {
	g      *gestureTester
	events log
	serial *SerialTapGestureRecognizer
	e      map[string]PointerEvent
}

func newSerialFixture(t *testing.T) *serialFixture {
	f := &serialFixture{g: newGestureTester(t)}
	f.serial = f.newSerial(true)
	f.e = map[string]PointerEvent{
		"down1": evDown(1, off(10, 10)), "up1": evUp(1, off(11, 9)), "cancel1": NewPointerCancelEvent(PointerEventOpts{Pointer: 1}),
		"down2": evDown(2, off(12, 12)), "up2": evUp(2, off(13, 11)),
		"down3": evDown(3, off(12, 12)), "up3": evUp(3, off(13, 11)),
		"down4": evDown(4, off(130, 130)), "up4": evUp(4, off(131, 129)),
		"down5": evDown(5, off(10, 10)), "move5": evMove(5, off(25, 25)), "up5": evUp(5, off(25, 25)),
		"down6": NewPointerDownEvent(PointerEventOpts{Pointer: 6, Position: off(10, 10), Buttons: foundation.Some(SecondaryMouseButton)}),
		"up6":   evUp(6, off(11, 9)),
	}
	return f
}

func (f *serialFixture) newSerial(callbacks bool) *SerialTapGestureRecognizer {
	s := NewSerialTapGestureRecognizer(f.g.binding)
	if callbacks {
		s.OnSerialTapDown = func(d SerialTapDownDetails) { f.events.add(fmt.Sprintf("down#%d", d.Count)) }
		s.OnSerialTapCancel = func(d SerialTapCancelDetails) { f.events.add(fmt.Sprintf("cancel#%d", d.Count)) }
		s.OnSerialTapUp = func(d SerialTapUpDetails) { f.events.add(fmt.Sprintf("up#%d", d.Count)) }
	}
	return s
}

func (f *serialFixture) down(r GestureRecognizer, name string) {
	d := f.e["down"+name].(*PointerDownEvent)
	r.AddPointer(d)
}

// tap is the sequence of a tap by the recognizers: add the pointer to each, close the
// arena, route down and up, sweep.
func (f *serialFixture) tap(n int, up string, rs ...GestureRecognizer) {
	name := fmt.Sprint(n)
	for _, r := range rs {
		f.down(r, name)
	}
	f.g.closeArena(n)
	f.g.route(f.e["down"+name])
	f.g.route(f.e[up])
	f.g.binding.GestureArena.Sweep(n)
}

func TestSerialTapRecognizes(t *testing.T) {
	f := newSerialFixture(t)
	f.tap(1, "up1", f.serial)
	f.events.check(t, "down#1", "up#1")
	f.events.clear()
	f.g.elapse(serialTapDelay)
	f.tap(2, "up2", f.serial)
	f.events.check(t, "down#2", "up#2")
	f.events.clear()
	f.g.elapse(serialTapDelay)
	f.tap(3, "up3", f.serial)
	f.events.check(t, "down#3", "up#3")
}

func TestSerialTapWinsOverTap(t *testing.T) {
	for _, serialFirst := range []bool{false, true} {
		name := "below it in the tree"
		if serialFirst {
			name = "above it in the tree"
		}
		t.Run(name, func(t *testing.T) {
			f := newSerialFixture(t)
			recognized, canceled := false, false
			single := NewTapGestureRecognizer(f.g.binding)
			single.OnTap = func() { recognized = true }
			single.OnTapCancel = func() { canceled = true }
			if serialFirst {
				f.down(f.serial, "1")
				f.down(single, "1")
			} else {
				f.down(single, "1")
				f.down(f.serial, "1")
			}
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.g.elapse(PressTimeout) // to register the possible single tap
			f.g.route(f.e["up1"])
			f.g.binding.GestureArena.Sweep(1)
			f.events.check(t, "down#1", "up#1")
			if recognized || !canceled {
				t.Errorf("single tap recognized = %v, canceled = %v", recognized, canceled)
			}
		})
	}
}

func TestSerialTapAndReleaseGesture(t *testing.T) {
	for _, tc := range []struct {
		name         string
		releaseFirst bool
		want         []string
		released     bool
	}{
		{"loses to release gesture below it in the tree", true, []string{"down#1", "cancel#1"}, true},
		{"wins over release gesture above it in the tree", false, []string{"down#1", "up#1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSerialFixture(t)
			released := false
			release := newRelease(f.g.binding)
			release.onRelease = func() { released = true }
			if tc.releaseFirst {
				f.down(release, "1")
				f.down(f.serial, "1")
			} else {
				f.down(f.serial, "1")
				f.down(release, "1")
			}
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.g.route(f.e["up1"])
			f.g.binding.GestureArena.Sweep(1)
			f.events.check(t, tc.want...)
			if released != tc.released {
				t.Errorf("released = %v, want %v", released, tc.released)
			}
		})
	}
}

func TestSerialTapCancelsIfCompetitorWins(t *testing.T) {
	f := newSerialFixture(t)
	winner := newWinning(f.g.binding)
	f.down(winner, "1")
	f.down(f.serial, "1")
	f.g.closeArena(1)
	f.g.route(f.e["down1"])
	f.g.route(f.e["up1"])
	f.g.binding.GestureArena.Sweep(1)
	f.events.check(t, "down#1", "cancel#1")
}

func TestSerialTapWinsOverDoubleTap(t *testing.T) {
	for _, doubleFirst := range []bool{true, false} {
		name := "below it in the tree"
		if !doubleFirst {
			name = "above it in the tree"
		}
		t.Run(name, func(t *testing.T) {
			f := newSerialFixture(t)
			recognized := false
			double := NewDoubleTapGestureRecognizer(f.g.binding)
			double.OnDoubleTap = func() { recognized = true }
			order := func(n int) []GestureRecognizer {
				if doubleFirst {
					return []GestureRecognizer{double, f.serial}
				}
				return []GestureRecognizer{f.serial, double}
			}
			f.tap(1, "up1", order(1)...)
			f.events.check(t, "down#1", "up#1")
			f.events.clear()
			f.g.elapse(serialTapDelay)
			f.tap(2, "up2", order(2)...)
			f.events.check(t, "down#2", "up#2")
			f.events.clear()
			f.g.elapse(serialTapDelay)
			f.tap(3, "up3", f.serial)
			f.events.check(t, "down#3", "up#3")
			if recognized {
				t.Error("the double tap won")
			}
		})
	}
}

func TestSerialTapResets(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps func(f *serialFixture)
		first []string
	}{
		{"PointerCancelEvent", func(f *serialFixture) {
			f.down(f.serial, "1")
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.g.route(f.e["cancel1"])
			f.g.binding.GestureArena.Sweep(1)
			f.events.check(f.g.t, "down#1", "cancel#1")
			f.events.clear()
			f.g.elapse(ms(100))
			f.tap(2, "up2", f.serial)
		}, nil},
		{"drag past the slop tolerance", func(f *serialFixture) {
			f.down(f.serial, "5")
			f.g.closeArena(5)
			f.g.route(f.e["down5"])
			f.g.route(f.e["move5"])
			f.g.route(f.e["up5"])
			f.g.binding.GestureArena.Sweep(5)
			f.events.check(f.g.t, "down#1", "cancel#1")
			f.events.clear()
			f.g.elapse(ms(1000))
			f.tap(1, "up1", f.serial)
		}, nil},
		{"timeout in between taps", func(f *serialFixture) {
			f.tap(1, "up1", f.serial)
			f.events.check(f.g.t, "down#1", "up#1")
			f.events.clear()
			f.g.elapse(ms(1000))
			f.tap(2, "up2", f.serial)
		}, nil},
		{"taps far apart", func(f *serialFixture) {
			f.tap(1, "up1", f.serial)
			f.events.check(f.g.t, "down#1", "up#1")
			f.events.clear()
			f.g.elapse(ms(100))
			f.tap(4, "up4", f.serial)
		}, nil},
		{"different buttons", func(f *serialFixture) {
			f.tap(1, "up1", f.serial)
			f.events.check(f.g.t, "down#1", "up#1")
			f.events.clear()
			f.g.elapse(ms(1000))
			f.tap(6, "up6", f.serial)
		}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSerialFixture(t)
			tc.steps(f)
			f.events.check(t, "down#1", "up#1")
		})
	}
}

func TestSerialTapInterleavingTaps(t *testing.T) {
	f := newSerialFixture(t)
	f.down(f.serial, "1")
	f.g.closeArena(1)
	f.g.route(f.e["down1"])
	f.down(f.serial, "2")
	f.g.closeArena(2)
	f.g.route(f.e["down2"])
	f.g.route(f.e["up1"])
	f.g.binding.GestureArena.Sweep(1)
	f.g.route(f.e["up2"])
	f.g.binding.GestureArena.Sweep(2)
	f.events.check(t, "down#1", "cancel#1", "down#1", "up#1")
}

func TestSerialTapNoCallbacksIsNoOp(t *testing.T) {
	f := newSerialFixture(t)
	serial := f.newSerial(false)
	f.down(serial, "1")
	f.g.closeArena(1)
	f.g.route(f.e["down1"])
	if serial.IsTrackingPointer() {
		t.Error("tracking a pointer")
	}
	f.g.route(f.e["up1"])
	f.g.binding.GestureArena.Sweep(1)
	f.events.check(t)
}

func TestSerialTapNonPrimaryButton(t *testing.T) {
	f := newSerialFixture(t)
	f.tap(6, "up6", f.serial)
	f.events.check(t, "down#1", "up#1")
}
