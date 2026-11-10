package ios

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/pointerconv"
	"github.com/rm4n0s/glutter/ui"
)

func TestHoverData(t *testing.T) {
	for _, tc := range []struct {
		phase gesturePhase
		want  ui.PointerChange
		none  bool
	}{
		{gestureBegan, ui.PointerChangeHover, false},
		{gestureChanged, ui.PointerChangeHover, false},
		{gestureEnded, ui.PointerChangeRemove, false},
		{gestureCancelled, ui.PointerChangeRemove, false},
		{0, 0, true},
	} {
		got := hoverData(1, tc.phase, 10, 20, 2, time.Second)
		if tc.none {
			if len(got) != 0 {
				t.Errorf("phase %d gave %d data, want none", tc.phase, len(got))
			}
			continue
		}
		if len(got) != 1 || got[0].Change != tc.want || got[0].PhysicalX != 20 || got[0].PhysicalY != 40 || got[0].Kind != ui.PointerDeviceKindMouse || got[0].Device != deviceMouse {
			t.Errorf("phase %d = %+v", tc.phase, got)
		}
	}
}

func TestPanZoomSession(t *testing.T) {
	g := &panZoom{device: deviceScroll}
	if got := g.data(0, gestureChanged, 0, 0, 5, 5, 1, 0, 1, 0); len(got) != 0 {
		t.Error("an update before the start gave data")
	}
	start := g.data(0, gestureBegan, 10, 10, 4, 4, 1, 0, 2, 0)
	if len(start) != 1 || start[0].Change != ui.PointerChangePanZoomStart || start[0].PanX != 0 || start[0].Scale != 1 || start[0].PhysicalX != 20 {
		t.Fatalf("start = %+v", start)
	}
	if got := g.data(0, gestureBegan, 0, 0, 0, 0, 1, 0, 2, 0); len(got) != 0 {
		t.Error("a second start gave data")
	}
	up := g.data(0, gestureChanged, 10, 10, 3, -6, 1.5, 0.25, 2, 0)
	if len(up) != 1 || up[0].Change != ui.PointerChangePanZoomUpdate || up[0].PanX != 6 || up[0].PanY != -12 || up[0].Scale != 1.5 || up[0].Rotation != 0.25 {
		t.Fatalf("update = %+v", up)
	}
	end := g.data(0, gestureEnded, 10, 10, 3, -6, 1.5, 0.25, 2, 0)
	if len(end) != 1 || end[0].Change != ui.PointerChangePanZoomEnd {
		t.Fatalf("end = %+v", end)
	}
	if got := g.data(0, gestureEnded, 0, 0, 0, 0, 1, 0, 1, 0); len(got) != 0 {
		t.Error("a second end gave data")
	}
}

// TestPointerThroughTheConverter runs a touch and a hover through the converter as the platform does: the add comes before the first
// data of a device (the hover that has not moved after it is not given), and the pan of a trackpad gets its deltas.
func TestPointerThroughTheConverter(t *testing.T) {
	c := pointerconv.New()
	var out []ui.PointerData
	out = append(out, c.Convert(hoverData(0, gestureBegan, 5, 5, 1, 0))...)
	g := &panZoom{device: deviceScroll}
	out = append(out, c.Convert(g.data(0, gestureBegan, 5, 5, 0, 0, 1, 0, 1, 0))...)
	out = append(out, c.Convert(g.data(0, gestureChanged, 5, 5, 4, 0, 1, 0, 1, 0))...)
	out = append(out, c.Convert(g.data(0, gestureChanged, 5, 5, 10, 2, 1, 0, 1, 0))...)
	var changes []ui.PointerChange
	for _, d := range out {
		changes = append(changes, d.Change)
	}
	// The converter gives the first hover its add and keeps the hover that has not moved.
	want := []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeAdd, ui.PointerChangePanZoomStart, ui.PointerChangePanZoomUpdate, ui.PointerChangePanZoomUpdate}
	if len(changes) != len(want) {
		t.Fatalf("changes = %v, want %v", changes, want)
	}
	for i := range want {
		if changes[i] != want[i] {
			t.Fatalf("changes = %v, want %v", changes, want)
		}
	}
	if last := out[len(out)-1]; last.PanDeltaX != 6 || last.PanDeltaY != 2 {
		t.Errorf("the last update has delta (%v, %v), want (6, 2)", last.PanDeltaX, last.PanDeltaY)
	}
}
