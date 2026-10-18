package wayland

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// tickingClock returns a clock that advances 1 ms at each reading.
func tickingClock() func() time.Duration {
	var t time.Duration
	return func() time.Duration { t += time.Millisecond; return t }
}

// ev is the part of a ui.PointerData that the batcher tests compare.
type ev struct {
	change  ui.PointerChange
	kind    ui.PointerDeviceKind
	signal  ui.PointerSignalKind
	device  int
	x, y    float64
	dx, dy  float64
	buttons int
	synth   bool
	scroll  [2]float64
	pan     [2]float64
	panD    [2]float64
	scale   float64
	rot     float64
}

func brief2(data []ui.PointerData) []ev {
	out := make([]ev, len(data))
	for i, d := range data {
		out[i] = ev{
			d.Change, d.Kind, d.SignalKind, d.Device, d.PhysicalX, d.PhysicalY, d.PhysicalDeltaX, d.PhysicalDeltaY,
			d.Buttons, d.Synthesized, [2]float64{d.ScrollDeltaX, d.ScrollDeltaY},
			[2]float64{d.PanX, d.PanY}, [2]float64{d.PanDeltaX, d.PanDeltaY}, d.Scale, d.Rotation,
		}
	}
	return out
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func eventsEqual(t *testing.T, got []ui.PointerData, want []ev) {
	t.Helper()
	g := brief2(got)
	if len(g) != len(want) {
		t.Fatalf("%d events, want %d:\n got  %+v\n want %+v", len(g), len(want), g, want)
	}
	for i := range g {
		a, b := g[i], want[i]
		if a.change != b.change || a.kind != b.kind || a.signal != b.signal || a.device != b.device ||
			a.buttons != b.buttons || a.synth != b.synth || !near(a.x, b.x) || !near(a.y, b.y) ||
			!near(a.dx, b.dx) || !near(a.dy, b.dy) || !near(a.scale, b.scale) || !near(a.rot, b.rot) ||
			!near(a.scroll[0], b.scroll[0]) || !near(a.scroll[1], b.scroll[1]) ||
			!near(a.pan[0], b.pan[0]) || !near(a.pan[1], b.pan[1]) || !near(a.panD[0], b.panD[0]) || !near(a.panD[1], b.panD[1]) {
			t.Errorf("event %d:\n got  %+v\n want %+v", i, a, b)
		}
	}
}

const (
	mouse    = ui.PointerDeviceKindMouse
	trackpad = ui.PointerDeviceKindTrackpad
)

func TestPointerEnterMotionButtonsLeave(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.enter(3, 10.5, 20)
	b.motion(12, 22)
	b.button(0x110, true) // left down
	b.motion(15, 22)
	b.button(0x111, true) // right joins
	b.button(0x111, false)
	b.button(0x110, false) // left up
	b.motion(15, 30)
	b.leave()
	got := b.frame()
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
	if len(b.frame()) != 0 {
		t.Error("a second frame has events")
	}
}

func TestPointerLeaveWithButtonsDownReleasesThem(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.enter(0, 5, 5)
	b.button(0x110, true)
	b.button(0x112, true)
	b.frame()
	b.leave()
	eventsEqual(t, b.frame(), []ev{
		{change: ui.PointerChangeUp, kind: mouse, x: 5, y: 5, synth: true},
		{change: ui.PointerChangeRemove, kind: mouse, x: 5, y: 5},
	})
	// Entering again starts with no button down.
	b.enter(0, 1, 1)
	b.button(0x110, true)
	got := b.frame()
	if got[len(got)-1].Change != ui.PointerChangeDown || got[len(got)-1].Buttons != 1 {
		t.Errorf("after the re-enter: %+v", brief2(got))
	}
}

func TestPointerEnterWhileInsideLeavesFirst(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.enter(1, 1, 1)
	b.enter(2, 3, 3)
	got := b.frame()
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
			b := newPointerBatcher(tickingClock())
			b.enter(0, 0, 0)
			b.frame()
			b.button(tt.code, true)
			got := b.frame()
			if (len(got) == 1) != (tt.want != 0) {
				t.Errorf("%d events for a press", len(got))
			}
		})
	}
}

func TestPointerIgnoresEventsOutsideAWindow(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.motion(1, 1)
	b.button(0x110, true)
	b.axisSource(wl.WlPointerAxisSourceWheel)
	b.axis(wl.WlPointerAxisVerticalScroll, 10)
	b.pinchBegin()
	b.swipeBegin()
	b.pinchUpdate(1, 1, 2, 3)
	if got := b.frame(); len(got) != 0 {
		t.Errorf("events outside a window: %+v", brief2(got))
	}
}

func TestPointerScale(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.scale = 2
	b.enter(0, 10, 20)
	b.motion(11, 22)
	b.axisSource(wl.WlPointerAxisSourceWheel)
	b.axisValue120(wl.WlPointerAxisVerticalScroll, 120)
	got := b.frame()
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
	const v, h = wl.WlPointerAxisVerticalScroll, wl.WlPointerAxisHorizontalScroll
	tests := []struct {
		name string
		feed func(b *pointerBatcher)
		want []ev
	}{
		{
			name: "one notch of a high-resolution wheel",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axis(v, 15) // the pixel value is ignored when value120 is there
				b.axisValue120(v, 120)
			},
			want: []ev{scrollEv(0, 53)},
		},
		{
			name: "a notch back",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axis(v, -15)
				b.axisValue120(v, -120)
			},
			want: []ev{scrollEv(0, -53)},
		},
		{
			name: "a half notch",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axisValue120(v, 60)
			},
			want: []ev{scrollEv(0, 26.5)},
		},
		{
			name: "several events in one frame add up",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axisValue120(v, 120)
				b.axisValue120(v, 120)
			},
			want: []ev{scrollEv(0, 106)},
		},
		{
			name: "both axes in one signal",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axisValue120(v, 120)
				b.axisValue120(h, -240)
			},
			want: []ev{scrollEv(-106, 53)},
		},
		{
			name: "a tilt wheel scrolls sideways",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheelTilt)
				b.axisValue120(h, 120)
			},
			want: []ev{scrollEv(53, 0)},
		},
		{
			name: "an older compositor sends clicks",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axis(v, 30)
				b.axisDiscrete(v, 2)
			},
			want: []ev{scrollEv(0, 106)},
		},
		{
			name: "a wheel with only pixels scrolls by them",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axis(v, 12.5)
			},
			want: []ev{scrollEv(0, 12.5)},
		},
		{
			name: "a continuous source scrolls by its pixels",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceContinuous)
				b.axis(v, 7)
				b.axis(h, -3.5)
			},
			want: []ev{scrollEv(-3.5, 7)},
		},
		{
			name: "no source at all (version 4 compositors)",
			feed: func(b *pointerBatcher) { b.axis(v, 10) },
			want: []ev{scrollEv(0, 10)},
		},
		{
			name: "a stop on a wheel does nothing",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axisStop(v)
			},
		},
		{
			name: "no movement, no signal",
			feed: func(b *pointerBatcher) {
				b.axisSource(wl.WlPointerAxisSourceWheel)
				b.axisValue120(v, 0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newPointerBatcher(tickingClock())
			b.enter(0, 4, 6)
			b.frame()
			tt.feed(b)
			eventsEqual(t, b.frame(), tt.want)
			// The next frame starts clean.
			if got := b.frame(); len(got) != 0 {
				t.Errorf("the next frame has %+v", brief2(got))
			}
		})
	}
}

func panEv(change ui.PointerChange, panX, panY, dx, dy, scale, rot float64) ev {
	return ev{change: change, kind: trackpad, device: trackpadDevice, x: 4, y: 6, pan: [2]float64{panX, panY}, panD: [2]float64{dx, dy}, scale: scale, rot: rot}
}

func TestPointerTouchpadScrollWithStop(t *testing.T) {
	const v, h = wl.WlPointerAxisVerticalScroll, wl.WlPointerAxisHorizontalScroll
	b := newPointerBatcher(tickingClock())
	b.enter(0, 4, 6)
	b.frame()

	// Frame 1: two fingers start moving up (the axis says "down").
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axis(v, 4)
	b.axis(h, -1)
	eventsEqual(t, b.frame(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 1, -4, 1, -4, 1, 0),
	})

	// Frame 2 continues the same gesture.
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axis(v, 2.5)
	eventsEqual(t, b.frame(), []ev{
		panEv(ui.PointerChangePanZoomUpdate, 1, -6.5, 0, -2.5, 1, 0),
	})

	// The fingers lift: a stop, with no movement. (The compositor does the kinetic
	// scrolling itself; the app gets the end of the gesture.)
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axisStop(v)
	b.axisStop(h)
	eventsEqual(t, b.frame(), []ev{panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0)})

	// A new gesture starts from zero.
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axis(v, 1)
	got := b.frame()
	eventsEqual(t, got, []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 0, -1, 0, -1, 1, 0),
	})
	if got[0].Buttons != 0 || got[0].Kind != ui.PointerDeviceKindTrackpad {
		t.Errorf("pan/zoom data: %+v", got[0])
	}
}

func TestPointerTouchpadStopWithoutGestureAndMovementAndStopTogether(t *testing.T) {
	const v = wl.WlPointerAxisVerticalScroll
	b := newPointerBatcher(tickingClock())
	b.enter(0, 4, 6)
	b.frame()

	// A stop for a gesture that never started makes nothing.
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axisStop(v)
	if got := b.frame(); len(got) != 0 {
		t.Errorf("a lone stop made %+v", brief2(got))
	}
	// A frame that has the last movement and the stop ends the gesture after the update.
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axis(v, 3)
	b.axisStop(v)
	eventsEqual(t, b.frame(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 0, -3, 0, -3, 1, 0),
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
	})
}

func TestPointerPinch(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.enter(0, 4, 6)
	b.frame()

	b.pinchBegin()
	b.pinchUpdate(2, -1, 1.5, 90)
	b.pinchUpdate(1, 1, 2, 180)
	b.pinchEnd()
	eventsEqual(t, b.take(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, 2, -1, 2, -1, 1.5, math.Pi/2),
		panEv(ui.PointerChangePanZoomUpdate, 3, 0, 1, 1, 2, math.Pi),
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
	})
}

func TestPointerSwipeIsAPan(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.enter(0, 4, 6)
	b.frame()
	b.swipeBegin()
	b.swipeUpdate(-5, 2)
	b.swipeEnd()
	eventsEqual(t, b.take(), []ev{
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomUpdate, -5, 2, -5, 2, 1, 0),
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
	})
}

func TestPointerGesturesEndOneAnotherAndOnLeave(t *testing.T) {
	const v = wl.WlPointerAxisVerticalScroll
	b := newPointerBatcher(tickingClock())
	b.enter(0, 4, 6)
	b.frame()

	// A pinch that starts while a touchpad scroll is going ends the scroll first.
	b.axisSource(wl.WlPointerAxisSourceFinger)
	b.axis(v, 1)
	b.frame()
	b.pinchBegin()
	got := b.take()
	eventsEqual(t, got, []ev{
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
		panEv(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 1, 0),
	})
	// Updates and ends of another kind of gesture are ignored.
	b.swipeUpdate(1, 1)
	b.swipeEnd()
	b.pinchUpdate(1, 1, 1, 0)
	b.take()
	// The pointer leaving ends the gesture too.
	b.leave()
	eventsEqual(t, b.take(), []ev{
		panEv(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 1, 0),
		{change: ui.PointerChangeRemove, kind: mouse, x: 4, y: 6},
	})
}

func TestPointerAxisDirectionIsRecorded(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.axisDirection(wl.WlPointerAxisVerticalScroll, wl.WlPointerAxisRelativeDirectionInverted)
	b.axisDirection(wl.WlPointerAxisHorizontalScroll, wl.WlPointerAxisRelativeDirectionIdentical)
	if !reflect.DeepEqual(b.inverted, [2]bool{true, false}) {
		t.Errorf("inverted = %v", b.inverted)
	}
}

func TestLeavingForgetsHalfCollectedAxes(t *testing.T) {
	b := newPointerBatcher(tickingClock())
	b.enter(0, 1, 1)
	b.frame()
	b.axisSource(wl.WlPointerAxisSourceWheel)
	b.axisValue120(wl.WlPointerAxisVerticalScroll, 120)
	b.leave() // before the frame
	b.enter(0, 1, 1)
	for _, d := range b.frame() {
		if d.SignalKind == ui.PointerSignalKindScroll {
			t.Error("a scroll from before the pointer left")
		}
	}
}
