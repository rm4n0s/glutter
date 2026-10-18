package input

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func touchEv(change ui.PointerChange, id int, x, y, dx, dy float64, buttons int) ev {
	return ev{change: change, kind: ui.PointerDeviceKindTouch, device: TouchDeviceBase + id, x: x, y: y, dx: dx, dy: dy, buttons: buttons}
}

func TestTouchSequence(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Down(2, 0, 10, 20)
	b.Motion(0, 12, 19)
	b.Up(0)
	got := b.Take()
	eventsEqual(t, got, []ev{
		touchEv(ui.PointerChangeAdd, 0, 10, 20, 0, 0, 0),
		touchEv(ui.PointerChangeDown, 0, 10, 20, 0, 0, 1),
		touchEv(ui.PointerChangeMove, 0, 12, 19, 2, -1, 1),
		touchEv(ui.PointerChangeUp, 0, 12, 19, 0, 0, 0),
		touchEv(ui.PointerChangeRemove, 0, 12, 19, 0, 0, 0),
	})
	for i, d := range got {
		if d.ViewID != 2 || d.PointerIdentifier != 0 {
			t.Errorf("event %d: view %d, identifier %d", i, d.ViewID, d.PointerIdentifier)
		}
		wantPressure := 1.0
		if d.Change == ui.PointerChangeAdd || d.Change == ui.PointerChangeUp || d.Change == ui.PointerChangeRemove {
			wantPressure = 0
		}
		if d.Pressure != wantPressure || d.PressureMax != 1 {
			t.Errorf("event %d (%v): pressure %v max %v", i, d.Change, d.Pressure, d.PressureMax)
		}
	}
	if len(b.Take()) != 0 {
		t.Error("take did not clear the data")
	}
}

func TestTwoFingersAreTwoDevices(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Down(0, 7, 1, 1)
	b.Down(0, 3, 50, 50)
	b.Motion(7, 2, 2)
	b.Motion(3, 49, 51)
	got := b.Take()
	if got[0].Device == got[2].Device || got[0].Device != TouchDeviceBase+7 || got[2].Device != TouchDeviceBase+3 {
		t.Errorf("devices: %d and %d", got[0].Device, got[2].Device)
	}
	eventsEqual(t, got[4:], []ev{
		touchEv(ui.PointerChangeMove, 7, 2, 2, 1, 1, 1),
		touchEv(ui.PointerChangeMove, 3, 49, 51, -1, 1, 1),
	})
}

func TestTouchCancel(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Down(0, 5, 1, 1)
	b.Down(0, 2, 9, 9)
	b.Take()
	b.Cancel()
	// In the order of the ids.
	eventsEqual(t, b.Take(), []ev{
		touchEv(ui.PointerChangeCancel, 2, 9, 9, 0, 0, 0),
		touchEv(ui.PointerChangeRemove, 2, 9, 9, 0, 0, 0),
		touchEv(ui.PointerChangeCancel, 5, 1, 1, 0, 0, 0),
		touchEv(ui.PointerChangeRemove, 5, 1, 1, 0, 0, 0),
	})
	// Nothing is left to move or lift.
	b.Motion(5, 3, 3)
	b.Up(5)
	if got := b.Take(); len(got) != 0 {
		t.Errorf("events for fingers that were cancelled: %+v", brief2(got))
	}
}

func TestTouchCancelView(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Down(1, 0, 1, 1)
	b.Down(2, 1, 2, 2)
	b.Take()
	b.CancelView(1)
	got := b.Take()
	if len(got) != 2 || got[0].Change != ui.PointerChangeCancel || got[0].Device != TouchDeviceBase {
		t.Fatalf("got %+v", brief2(got))
	}
	// The other window's finger is still down.
	b.Motion(1, 3, 3)
	if got := b.Take(); len(got) != 1 || got[0].Change != ui.PointerChangeMove {
		t.Errorf("got %+v", brief2(got))
	}
}

func TestTouchDownOfAFingerThatIsDownCancelsTheOldOne(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Down(0, 0, 1, 1)
	b.Take()
	b.Down(0, 0, 5, 5)
	eventsEqual(t, b.Take(), []ev{
		touchEv(ui.PointerChangeCancel, 0, 1, 1, 0, 0, 0),
		touchEv(ui.PointerChangeRemove, 0, 1, 1, 0, 0, 0),
		touchEv(ui.PointerChangeAdd, 0, 5, 5, 0, 0, 0),
		touchEv(ui.PointerChangeDown, 0, 5, 5, 0, 0, 1),
	})
}

func TestTouchShapeAndOrientationShowInLaterEvents(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Down(0, 0, 1, 1)
	b.Shape(0, 12, 8) // the axes of the ellipse; Flutter wants radii
	b.Orientation(0, 90)
	b.Motion(0, 2, 2)
	got := b.Take()
	if got[1].RadiusMajor != 0 {
		t.Errorf("the down event, before the shape, has radius %v", got[1].RadiusMajor)
	}
	m := got[2]
	if m.RadiusMajor != 6 || m.RadiusMinor != 4 || math.Abs(m.Orientation-math.Pi/2) > 1e-9 {
		t.Errorf("move: radius %v x %v, orientation %v", m.RadiusMajor, m.RadiusMinor, m.Orientation)
	}
	// Unknown fingers are ignored.
	b.Shape(9, 1, 1)
	b.Orientation(9, 1)
	b.Motion(9, 1, 1)
	b.Up(9)
	if len(b.Take()) != 0 {
		t.Error("events for an unknown finger")
	}
}

func TestTouchScale(t *testing.T) {
	b := NewTouchBatcher(tickingClock())
	b.Scale = 1.5
	b.Down(0, 0, 10, 20)
	b.Motion(0, 12, 20)
	got := b.Take()
	if got[1].PhysicalX != 15 || got[1].PhysicalY != 30 || got[2].PhysicalDeltaX != 3 {
		t.Errorf("got %+v", brief2(got))
	}
}
