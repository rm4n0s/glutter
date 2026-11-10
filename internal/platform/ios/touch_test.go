package ios

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

func sample(id uintptr, phase touchPhase, x, y float64) touchSample {
	return touchSample{ID: id, Phase: phase, X: x, Y: y, MaxForce: 4, Force: 1, Radius: 10, Tolerance: 2}
}

func TestTouchConversion(t *testing.T) {
	tr := newTouchTracker()
	type want struct {
		change ui.PointerChange
		device int
		x, y   float64
	}
	steps := []struct {
		name  string
		in    []touchSample
		scale float64
		want  []want
	}{
		{"a finger goes down", []touchSample{sample(1, touchBegan, 10, 20)}, 3, []want{{ui.PointerChangeDown, 1, 30, 60}}},
		{"a second finger gets the next device", []touchSample{sample(2, touchBegan, 5, 5), sample(1, touchMoved, 11, 20)}, 3,
			[]want{{ui.PointerChangeDown, 2, 15, 15}, {ui.PointerChangeMove, 1, 33, 60}}},
		{"a stationary touch is nothing", []touchSample{sample(1, touchStationary, 11, 20)}, 3, nil},
		{"an up is followed by its remove", []touchSample{sample(1, touchEnded, 11, 20)}, 3,
			[]want{{ui.PointerChangeUp, 1, 33, 60}, {ui.PointerChangeRemove, 1, 33, 60}}},
		{"the device of an ended touch is reused", []touchSample{sample(3, touchBegan, 1, 1)}, 1, []want{{ui.PointerChangeDown, 1, 1, 1}}},
		{"a cancel is followed by its remove", []touchSample{sample(2, touchCancelled, 5, 5)}, 1,
			[]want{{ui.PointerChangeCancel, 2, 5, 5}, {ui.PointerChangeRemove, 2, 5, 5}}},
		{"a move of a touch that did not begin is dropped", []touchSample{sample(9, touchMoved, 1, 1), sample(9, touchEnded, 1, 1)}, 1, nil},
	}
	for _, s := range steps {
		got := tr.convert(0, s.in, s.scale)
		if len(got) != len(s.want) {
			t.Errorf("%s: %d data, want %d", s.name, len(got), len(s.want))
			continue
		}
		for i, w := range s.want {
			g := got[i]
			if g.Change != w.change || g.Device != w.device || g.PhysicalX != w.x || g.PhysicalY != w.y {
				t.Errorf("%s: data %d = %v on device %d at (%v, %v), want %v on %d at (%v, %v)", s.name, i, g.Change, g.Device, g.PhysicalX, g.PhysicalY, w.change, w.device, w.x, w.y)
			}
		}
	}
	if tr.Active() != 1 { // touch 3 is still down
		t.Errorf("%d touches are active, want 1", tr.Active())
	}
}

func TestTouchValues(t *testing.T) {
	tr := newTouchTracker()
	finger := tr.convert(7, []touchSample{{ID: 1, Phase: touchBegan, X: 1, Y: 2, Force: 2, MaxForce: 8, Radius: 6, Tolerance: 1, Time: time.Second}}, 2)[0]
	if finger.ViewID != 7 || finger.Kind != ui.PointerDeviceKindTouch || finger.Pressure != 2 || finger.PressureMax != 8 || finger.PressureMin != 0 ||
		finger.RadiusMajor != 12 || finger.RadiusMin != 10 || finger.RadiusMax != 14 || finger.TimeStamp != time.Second || finger.Scale != 1 {
		t.Errorf("finger = %+v", finger)
	}
	noForce := tr.convert(0, []touchSample{{ID: 2, Phase: touchBegan}}, 1)[0]
	if noForce.Pressure != 1 || noForce.PressureMax != 1 {
		t.Errorf("a touch with no force has pressure %v of %v, want 1 of 1", noForce.Pressure, noForce.PressureMax)
	}
	pencil := tr.convert(0, []touchSample{{ID: 3, Phase: touchBegan, Kind: touchPencil, Altitude: math.Pi / 4, Azimuth: 1.5, Force: 1, MaxForce: 4}}, 1)[0]
	if pencil.Kind != ui.PointerDeviceKindStylus || math.Abs(pencil.Tilt-math.Pi/4) > 1e-9 || pencil.Orientation != 1.5 {
		t.Errorf("pencil = kind %v tilt %v orientation %v", pencil.Kind, pencil.Tilt, pencil.Orientation)
	}
	click := tr.convert(0, []touchSample{{ID: 4, Phase: touchBegan, Kind: touchIndirectPointer}}, 1)[0]
	if click.Kind != ui.PointerDeviceKindMouse || click.Buttons != 1 {
		t.Errorf("click = kind %v buttons %d", click.Kind, click.Buttons)
	}
}

func TestSortSamples(t *testing.T) {
	s := []touchSample{{ID: 3, Time: 2}, {ID: 1, Time: 2}, {ID: 2, Time: 1}}
	sortSamples(s)
	if s[0].ID != 2 || s[1].ID != 1 || s[2].ID != 3 {
		t.Errorf("order = %d %d %d, want 2 1 3", s[0].ID, s[1].ID, s[2].ID)
	}
}
