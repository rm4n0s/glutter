package input

import (
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestPointerEnterMotionButtonsLeave(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Enter(3, 10.5, 20)
	b.Motion(12, 22)
	b.Button(0x110, true) // left down
	b.Motion(15, 22)
	b.Button(0x111, true) // right joins
	b.Button(0x111, false)
	b.Button(0x110, false) // left up
	b.Motion(15, 30)
	b.Leave()
	got := b.Frame()
	eventsEqual(t, got, []ev{
		{change: ui.PointerChangeAdd, kind: mouse, x: 10.5, y: 20},
		{change: ui.PointerChangeHover, kind: mouse, x: 12, y: 22, dx: 1.5, dy: 2},
		{change: ui.PointerChangeDown, kind: mouse, x: 12, y: 22, buttons: 1},
		{change: ui.PointerChangeMove, kind: mouse, x: 15, y: 22, dx: 3, buttons: 1},
		{change: ui.PointerChangeMove, kind: mouse, x: 15, y: 22, buttons: 3},
		{change: ui.PointerChangeMove, kind: mouse, x: 15, y: 22, buttons: 1},
		{change: ui.PointerChangeUp, kind: mouse, x: 15, y: 22},
		{change: ui.PointerChangeHover, kind: mouse, x: 15, y: 30, dy: 8},
		{change: ui.PointerChangeRemove, kind: mouse, x: 15, y: 30},
	})
	for _, d := range got {
		if d.ViewID != 3 {
			t.Errorf("view = %d, want 3", d.ViewID)
		}
	}
	for i := 1; i < len(got); i++ {
		if got[i].TimeStamp <= got[i-1].TimeStamp {
			t.Errorf("time stamps do not increase at %d", i)
		}
	}
	if len(b.Frame()) != 0 {
		t.Error("a second frame has events")
	}
}

func TestPointerLeaveWithButtonsDownReleasesThem(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 5, 5)
	b.Button(0x110, true)
	b.Button(0x112, true)
	b.Frame()
	b.Leave()
	eventsEqual(t, b.Frame(), []ev{
		{change: ui.PointerChangeUp, kind: mouse, x: 5, y: 5, synth: true},
		{change: ui.PointerChangeRemove, kind: mouse, x: 5, y: 5},
	})
	// Entering again starts with no button down.
	b.Enter(0, 1, 1)
	b.Button(0x110, true)
	got := b.Frame()
	if got[len(got)-1].Change != ui.PointerChangeDown || got[len(got)-1].Buttons != 1 {
		t.Errorf("after the re-enter: %+v", brief2(got))
	}
}

func TestPointerEnterWhileInsideLeavesFirst(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Enter(1, 1, 1)
	b.Enter(2, 3, 3)
	got := b.Frame()
	eventsEqual(t, got, []ev{
		{change: ui.PointerChangeAdd, kind: mouse, x: 1, y: 1},
		{change: ui.PointerChangeRemove, kind: mouse, x: 1, y: 1},
		{change: ui.PointerChangeAdd, kind: mouse, x: 3, y: 3},
	})
	if got[2].ViewID != 2 || got[1].ViewID != 1 {
		t.Errorf("views = %d, %d", got[1].ViewID, got[2].ViewID)
	}
}

func TestPointerButtonMapping(t *testing.T) {
	tests := []struct {
		name string
		code uint32
		want int
	}{
		{"left", 0x110, 1}, {"right", 0x111, 2}, {"middle", 0x112, 4},
		{"side is back", 0x113, 8}, {"back", 0x116, 8},
		{"extra is forward", 0x114, 16}, {"forward", 0x115, 16},
		{"task has no bit", 0x117, 0}, {"a key code", 30, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buttonBit(tt.code); got != tt.want {
				t.Errorf("buttonBit(0x%x) = %d, want %d", tt.code, got, tt.want)
			}
			b := NewPointerBatcher(tickingClock())
			b.Enter(0, 0, 0)
			b.Frame()
			b.Button(tt.code, true)
			got := b.Frame()
			if (len(got) == 1) != (tt.want != 0) {
				t.Errorf("%d events for a press", len(got))
			}
		})
	}
}

func TestPointerIgnoresEventsOutsideAWindow(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Motion(1, 1)
	b.Button(0x110, true)
	b.AxisSource(AxisSourceWheel)
	b.Axis(AxisVertical, 10)
	b.PinchBegin()
	b.SwipeBegin()
	b.PinchUpdate(1, 1, 2, 3)
	if got := b.Frame(); len(got) != 0 {
		t.Errorf("events outside a window: %+v", brief2(got))
	}
}

func TestPointerScale(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Scale = 2
	b.Enter(0, 10, 20)
	b.Motion(11, 22)
	b.AxisSource(AxisSourceWheel)
	b.AxisValue120(AxisVertical, 120)
	got := b.Frame()
	eventsEqual(t, got, []ev{
		{change: ui.PointerChangeAdd, kind: mouse, x: 20, y: 40},
		{change: ui.PointerChangeHover, kind: mouse, x: 22, y: 44, dx: 2, dy: 4},
		{change: ui.PointerChangeHover, kind: mouse, signal: ui.PointerSignalKindScroll, x: 22, y: 44, scroll: [2]float64{0, 106}},
	})
}

func scrollEv(dx, dy float64) ev {
	return ev{change: ui.PointerChangeHover, kind: mouse, signal: ui.PointerSignalKindScroll, x: 4, y: 6, scroll: [2]float64{dx, dy}}
}

func TestPointerWheelScrolling(t *testing.T) {
	const v, h = AxisVertical, AxisHorizontal
	tests := []struct {
		name string
		feed func(b *PointerBatcher)
		want []ev
	}{
		{
			name: "one notch of a high-resolution wheel",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.Axis(v, 15) // the pixel value is ignored when value120 is there
				b.AxisValue120(v, 120)
			},
			want: []ev{scrollEv(0, 53)},
		},
		{
			name: "a notch back",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.Axis(v, -15)
				b.AxisValue120(v, -120)
			},
			want: []ev{scrollEv(0, -53)},
		},
		{
			name: "a half notch",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.AxisValue120(v, 60)
			},
			want: []ev{scrollEv(0, 26.5)},
		},
		{
			name: "several events in one frame add up",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.AxisValue120(v, 120)
				b.AxisValue120(v, 120)
			},
			want: []ev{scrollEv(0, 106)},
		},
		{
			name: "both axes in one signal",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.AxisValue120(v, 120)
				b.AxisValue120(h, -240)
			},
			want: []ev{scrollEv(-106, 53)},
		},
		{
			name: "a tilt wheel scrolls sideways",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheelTilt)
				b.AxisValue120(h, 120)
			},
			want: []ev{scrollEv(53, 0)},
		},
		{
			name: "an older compositor sends clicks",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.Axis(v, 30)
				b.AxisDiscrete(v, 2)
			},
			want: []ev{scrollEv(0, 106)},
		},
		{
			name: "a wheel with only pixels scrolls by them",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.Axis(v, 12.5)
			},
			want: []ev{scrollEv(0, 12.5)},
		},
		{
			name: "a continuous source scrolls by its pixels",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceContinuous)
				b.Axis(v, 7)
				b.Axis(h, -3.5)
			},
			want: []ev{scrollEv(-3.5, 7)},
		},
		{
			name: "no source at all (version 4 compositors)",
			feed: func(b *PointerBatcher) { b.Axis(v, 10) },
			want: []ev{scrollEv(0, 10)},
		},
		{
			name: "a stop on a wheel does nothing",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.AxisStop(v)
			},
		},
		{
			name: "no movement, no signal",
			feed: func(b *PointerBatcher) {
				b.AxisSource(AxisSourceWheel)
				b.AxisValue120(v, 0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewPointerBatcher(tickingClock())
			b.Enter(0, 4, 6)
			b.Frame()
			tt.feed(b)
			eventsEqual(t, b.Frame(), tt.want)
			// The next frame starts clean.
			if got := b.Frame(); len(got) != 0 {
				t.Errorf("the next frame has %+v", brief2(got))
			}
		})
	}
}

func panEv(change ui.PointerChange, panX, panY, dx, dy, scale, rot float64) ev {
	return ev{change: change, kind: trackpad, device: TrackpadDevice, x: 4, y: 6, pan: [2]float64{panX, panY}, panD: [2]float64{dx, dy}, scale: scale, rot: rot}
}

func TestPointerTouchpadScrollWithStop(t *testing.T) {
	const v, h = AxisVertical, AxisHorizontal
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 4, 6)
	b.Frame()

	// Frame 1: two fingers start moving up (the axis says "down").
	b.AxisSource(AxisSourceFinger)
	b.Axis(v, 4)
	b.Axis(h, -1)
	eventsEqual(t, b.Frame(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 1, -4, 1, -4, 1, 0),
	})

	// Frame 2 continues the same gesture.
	b.AxisSource(AxisSourceFinger)
	b.Axis(v, 2.5)
	eventsEqual(t, b.Frame(), []ev{
		panEv(ui.PointerChangePanZoomUpdate, 1, -6.5, 0, -2.5, 1, 0),
	})

	// The fingers lift: a stop, with no movement. (The compositor does the kinetic
	// scrolling itself; the app gets the end of the gesture.)
	b.AxisSource(AxisSourceFinger)
	b.AxisStop(v)
	b.AxisStop(h)
	eventsEqual(t, b.Frame(), []ev{panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0)})

	// A new gesture starts from zero.
	b.AxisSource(AxisSourceFinger)
	b.Axis(v, 1)
	got := b.Frame()
	eventsEqual(t, got, []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 0, -1, 0, -1, 1, 0),
	})
	if got[0].Buttons != 0 || got[0].Kind != ui.PointerDeviceKindTrackpad {
		t.Errorf("pan/zoom data: %+v", got[0])
	}
}

func TestPointerTouchpadStopWithoutGestureAndMovementAndStopTogether(t *testing.T) {
	const v = AxisVertical
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 4, 6)
	b.Frame()

	// A stop for a gesture that never started makes nothing.
	b.AxisSource(AxisSourceFinger)
	b.AxisStop(v)
	if got := b.Frame(); len(got) != 0 {
		t.Errorf("a lone stop made %+v", brief2(got))
	}
	// A frame that has the last movement and the stop ends the gesture after the update.
	b.AxisSource(AxisSourceFinger)
	b.Axis(v, 3)
	b.AxisStop(v)
	eventsEqual(t, b.Frame(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 0, -3, 0, -3, 1, 0),
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
	})
}

func TestPointerPinch(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 4, 6)
	b.Frame()

	b.PinchBegin()
	b.PinchUpdate(2, -1, 1.5, 90)
	b.PinchUpdate(1, 1, 2, 180)
	b.PinchEnd()
	eventsEqual(t, b.Take(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 2, -1, 2, -1, 1.5, math.Pi/2),
		panEv(ui.PointerChangePanZoomUpdate, 3, 0, 1, 1, 2, math.Pi),
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
	})
}

func TestPointerSwipeIsAPan(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 4, 6)
	b.Frame()
	b.SwipeBegin()
	b.SwipeUpdate(-5, 2)
	b.SwipeEnd()
	eventsEqual(t, b.Take(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, -5, 2, -5, 2, 1, 0),
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
	})
}

func TestPointerGesturesEndOneAnotherAndOnLeave(t *testing.T) {
	const v = AxisVertical
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 4, 6)
	b.Frame()

	// A pinch that starts while a touchpad scroll is going ends the scroll first.
	b.AxisSource(AxisSourceFinger)
	b.Axis(v, 1)
	b.Frame()
	b.PinchBegin()
	got := b.Take()
	eventsEqual(t, got, []ev{
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
	})
	// Updates and ends of another kind of gesture are ignored.
	b.SwipeUpdate(1, 1)
	b.SwipeEnd()
	b.PinchUpdate(1, 1, 1, 0)
	b.Take()
	// The pointer leaving ends the gesture too.
	b.Leave()
	eventsEqual(t, b.Take(), []ev{
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
		{change: ui.PointerChangeRemove, kind: mouse, x: 4, y: 6},
	})
}

func TestPointerAxisDirectionIsRecorded(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.AxisDirection(AxisVertical, true)
	b.AxisDirection(AxisHorizontal, false)
	if !reflect.DeepEqual(b.inverted, [2]bool{true, false}) {
		t.Errorf("inverted = %v", b.inverted)
	}
}

func TestLeavingForgetsHalfCollectedAxes(t *testing.T) {
	b := NewPointerBatcher(tickingClock())
	b.Enter(0, 1, 1)
	b.Frame()
	b.AxisSource(AxisSourceWheel)
	b.AxisValue120(AxisVertical, 120)
	b.Leave() // before the frame
	b.Enter(0, 1, 1)
	for _, d := range b.Frame() {
		if d.SignalKind == ui.PointerSignalKindScroll {
			t.Error("a scroll from before the pointer left")
		}
	}
}
