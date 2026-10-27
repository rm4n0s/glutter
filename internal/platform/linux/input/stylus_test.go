//go:build linux && !android

package input

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func stylusEv(change ui.PointerChange, kind ui.PointerDeviceKind, id int, x, y, dx, dy float64, buttons int) ev {
	return ev{change: change, kind: kind, device: StylusDeviceBase + id, x: x, y: y, dx: dx, dy: dy, buttons: buttons}
}

func TestStylusSequence(t *testing.T) {
	pen := ui.PointerDeviceKindStylus
	b := NewStylusBatcher(tickingClock(), nil)
	b.AddTool(4, pen)
	// The tool comes near at (10, 20) and hovers.
	b.ProximityIn(4, 3)
	b.Motion(4, 10, 20)
	b.Distance(4, 32768)
	b.Frame(4)
	b.Motion(4, 12, 21)
	b.Frame(4)
	// It touches, draws and lifts.
	b.Down(4)
	b.Pressure(4, 65535)
	b.Frame(4)
	b.Motion(4, 15, 25)
	b.Frame(4)
	b.Up(4)
	b.Pressure(4, 0)
	b.Frame(4)
	b.ProximityOut(4)
	b.Frame(4)
	got := b.Take()
	eventsEqual(t, got, []ev{
		stylusEv(ui.PointerChangeAdd, pen, 4, 10, 20, 0, 0, 0),
		stylusEv(ui.PointerChangeHover, pen, 4, 12, 21, 2, 1, 0),
		stylusEv(ui.PointerChangeDown, pen, 4, 12, 21, 0, 0, 1),
		stylusEv(ui.PointerChangeMove, pen, 4, 15, 25, 3, 4, 1),
		stylusEv(ui.PointerChangeUp, pen, 4, 15, 25, 0, 0, 0),
		stylusEv(ui.PointerChangeRemove, pen, 4, 15, 25, 0, 0, 0),
	})
	// The tool has no contact when it is added and hovers: the identifier is 0, then the first
	// contact gives a new one that its move, up and remove carry.
	wantID := []int{0, 0, 1, 1, 1, 1}
	for i, d := range got {
		if d.ViewID != 3 || d.PointerIdentifier != wantID[i] || d.PressureMax != 1 || d.DistanceMax != 1 {
			t.Errorf("event %d: view %d, identifier %d, pressure max %v, distance max %v", i, d.ViewID, d.PointerIdentifier, d.PressureMax, d.DistanceMax)
		}
	}
	if got[0].Distance < 0.49 || got[0].Distance > 0.51 {
		t.Errorf("distance = %v, want about 0.5", got[0].Distance)
	}
	if got[2].Pressure != 1 || got[3].Pressure != 1 || got[1].Pressure != 0 || got[4].Pressure != 0 {
		t.Errorf("pressures = %v %v %v %v", got[1].Pressure, got[2].Pressure, got[3].Pressure, got[4].Pressure)
	}
}

func TestStylusKinds(t *testing.T) {
	for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus, ui.PointerDeviceKindTouch, ui.PointerDeviceKindMouse} {
		b := NewStylusBatcher(tickingClock(), nil)
		b.AddTool(1, kind)
		b.ProximityIn(1, 0)
		b.Frame(1)
		got := b.Take()
		if len(got) != 1 || got[0].Kind != kind || got[0].Change != ui.PointerChangeAdd {
			t.Errorf("kind %v: %+v", kind, got)
		}
	}
}

func TestStylusBarrelButtons(t *testing.T) {
	pen := ui.PointerDeviceKindStylus
	tests := []struct {
		name string
		code uint32
		want int
	}{
		{"first barrel button", btnStylus, 0x02},
		{"second barrel button", btnStylus2, 0x04},
		{"left button of a mouse tool", 0x110, 0x01},
		{"a button with no bit", 0x149, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewStylusBatcher(tickingClock(), nil)
			b.AddTool(1, pen)
			b.ProximityIn(1, 0)
			b.Motion(1, 5, 5)
			b.Frame(1)
			b.Button(1, tt.code, true)
			b.Frame(1)
			b.Take()
			b.Down(1)
			b.Frame(1)
			b.Up(1)
			b.Button(1, tt.code, false)
			b.Frame(1)
			got := b.Take()
			if len(got) != 2 {
				t.Fatalf("%d events: %+v", len(got), got)
			}
			if want := 1 | tt.want; got[0].Buttons != want {
				t.Errorf("down buttons = %#x, want %#x", got[0].Buttons, want)
			}
			if got[1].Buttons != 0 {
				t.Errorf("up buttons = %#x, want 0", got[1].Buttons)
			}
		})
	}
	// During a hover the barrel button shows in the hover.
	b := NewStylusBatcher(tickingClock(), nil)
	b.AddTool(1, pen)
	b.ProximityIn(1, 0)
	b.Frame(1)
	b.Button(1, btnStylus, true)
	b.Frame(1)
	got := b.Take()
	if len(got) != 2 || got[1].Change != ui.PointerChangeHover || got[1].Buttons != 2 {
		t.Errorf("hover with a button: %+v", got)
	}
}

func TestStylusLeavingWhileTouching(t *testing.T) {
	b := NewStylusBatcher(tickingClock(), nil)
	b.AddTool(1, ui.PointerDeviceKindStylus)
	b.ProximityIn(1, 0)
	b.Frame(1)
	b.Down(1)
	b.Frame(1)
	b.ProximityOut(1)
	b.Frame(1)
	var changes []ui.PointerChange
	for _, d := range b.Take() {
		changes = append(changes, d.Change)
	}
	want := []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove}
	if len(changes) != len(want) {
		t.Fatalf("changes = %v, want %v", changes, want)
	}
	for i := range want {
		if changes[i] != want[i] {
			t.Fatalf("changes = %v, want %v", changes, want)
		}
	}
}

func TestStylusTool(t *testing.T) {
	t.Run("removing a tool that is near removes it from the view", func(t *testing.T) {
		b := NewStylusBatcher(tickingClock(), nil)
		b.AddTool(1, ui.PointerDeviceKindStylus)
		b.ProximityIn(1, 0)
		b.Frame(1)
		b.Down(1)
		b.Frame(1)
		b.Take()
		b.RemoveTool(1)
		got := b.Take()
		if len(got) != 2 || got[0].Change != ui.PointerChangeUp || got[1].Change != ui.PointerChangeRemove {
			t.Errorf("got %+v", got)
		}
		b.Frame(1) // a tool that is gone does nothing
		if len(b.Take()) != 0 {
			t.Error("a removed tool made data")
		}
	})
	t.Run("a window that goes away cancels the tool on it", func(t *testing.T) {
		b := NewStylusBatcher(tickingClock(), nil)
		b.AddTool(1, ui.PointerDeviceKindStylus)
		b.AddTool(2, ui.PointerDeviceKindStylus)
		b.ProximityIn(1, 7)
		b.ProximityIn(2, 8)
		b.Frame(1)
		b.Frame(2)
		b.Down(1)
		b.Frame(1)
		b.Take()
		b.CancelView(7)
		got := b.Take()
		if len(got) != 2 || got[0].Change != ui.PointerChangeCancel || got[1].Change != ui.PointerChangeRemove || got[0].Device != StylusDeviceBase+1 {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("scale", func(t *testing.T) {
		b := NewStylusBatcher(tickingClock(), nil)
		b.Scale = 2
		b.AddTool(1, ui.PointerDeviceKindStylus)
		b.ProximityIn(1, 0)
		b.Motion(1, 3, 4)
		b.Frame(1)
		if got := b.Take(); got[0].PhysicalX != 6 || got[0].PhysicalY != 8 {
			t.Errorf("position = %v,%v", got[0].PhysicalX, got[0].PhysicalY)
		}
	})
}

func TestTiltAngles(t *testing.T) {
	tests := []struct {
		name          string
		x, y          float64
		tilt, azimuth float64
	}{
		{"upright", 0, 0, 0, 0},
		{"tilted along x", 45, 0, math.Pi / 4, 0},
		{"tilted along y", 0, 45, math.Pi / 4, math.Pi / 2},
		{"tilted against y", 0, -45, math.Pi / 4, 3 * math.Pi / 2},
		{"lying down", 90, 0, math.Pi / 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tilt, az := tiltAngles(tt.x, tt.y)
			if math.Abs(tilt-tt.tilt) > 1e-6 || math.Abs(az-tt.azimuth) > 1e-6 {
				t.Errorf("tiltAngles(%v, %v) = %v, %v; want %v, %v", tt.x, tt.y, tilt, az, tt.tilt, tt.azimuth)
			}
		})
	}
	b := NewStylusBatcher(tickingClock(), nil)
	b.AddTool(1, ui.PointerDeviceKindStylus)
	b.ProximityIn(1, 0)
	b.Tilt(1, 0, 45)
	b.Frame(1)
	if got := b.Take()[0]; math.Abs(got.Tilt-math.Pi/4) > 1e-6 || math.Abs(got.Orientation-math.Pi/2) > 1e-6 {
		t.Errorf("tilt %v, orientation %v", got.Tilt, got.Orientation)
	}
	b = NewStylusBatcher(tickingClock(), nil)
	b.AddTool(1, ui.PointerDeviceKindStylus)
	b.ProximityIn(1, 0)
	b.Rotation(1, 90)
	b.Frame(1)
	if got := b.Take()[0]; math.Abs(got.Orientation-math.Pi/2) > 1e-6 || got.Tilt != 0 {
		t.Errorf("rotation: tilt %v, orientation %v", got.Tilt, got.Orientation)
	}
}
