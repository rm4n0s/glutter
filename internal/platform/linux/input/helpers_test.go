//go:build linux

package input

import (
	"math"
	"testing"
	"time"

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
