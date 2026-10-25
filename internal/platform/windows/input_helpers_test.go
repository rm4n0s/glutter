package windows

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The shorthands of the pointer tests.
const (
	cAdd      = ui.PointerChangeAdd
	cHover    = ui.PointerChangeHover
	cDown     = ui.PointerChangeDown
	cMove     = ui.PointerChangeMove
	cUp       = ui.PointerChangeUp
	cRemove   = ui.PointerChangeRemove
	cCancel   = ui.PointerChangeCancel
	kMouse    = ui.PointerDeviceKindMouse
	kTouch    = ui.PointerDeviceKindTouch
	kStylus   = ui.PointerDeviceKindStylus
	kInverted = ui.PointerDeviceKindInvertedStylus
)

// ev is the part of a ui.PointerData that the pointer tests compare. A zero view is the first one.
type ev struct {
	change  ui.PointerChange
	kind    ui.PointerDeviceKind
	device  int
	x, y    float64
	dx, dy  float64
	buttons int
	synth   bool
	view    ui.ViewID
	scroll  [2]float64
	signal  bool
}

// brief reads the compared part of the data.
func brief(data []ui.PointerData) []ev {
	out := make([]ev, len(data))
	for i, d := range data {
		out[i] = ev{
			change: d.Change, kind: d.Kind, device: d.Device, x: d.PhysicalX, y: d.PhysicalY,
			dx: d.PhysicalDeltaX, dy: d.PhysicalDeltaY, buttons: d.Buttons, synth: d.Synthesized, view: d.ViewID,
			scroll: [2]float64{d.ScrollDeltaX, d.ScrollDeltaY}, signal: d.SignalKind == ui.PointerSignalKindScroll,
		}
	}
	return out
}

// tickingClock returns a clock that advances 1 ms at each reading.
func tickingClock() func() time.Duration {
	var t time.Duration
	return func() time.Duration { t += time.Millisecond; return t }
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// sameEvents fails the test when the data is not the events, in order.
func sameEvents(t *testing.T, got []ui.PointerData, want []ev) {
	t.Helper()
	g := brief(got)
	if len(g) != len(want) {
		t.Fatalf("%d events, want %d:\n got  %+v\n want %+v", len(g), len(want), g, want)
	}
	for i := range g {
		a, b := g[i], want[i]
		if a.change != b.change || a.kind != b.kind || a.device != b.device || a.buttons != b.buttons ||
			a.synth != b.synth || a.view != b.view || a.signal != b.signal ||
			!near(a.x, b.x) || !near(a.y, b.y) || !near(a.dx, b.dx) || !near(a.dy, b.dy) ||
			!near(a.scroll[0], b.scroll[0]) || !near(a.scroll[1], b.scroll[1]) {
			t.Errorf("event %d:\n got  %+v\n want %+v", i, a, b)
		}
	}
}
