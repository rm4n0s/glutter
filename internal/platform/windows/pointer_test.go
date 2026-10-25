package windows

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// step is one pointer message: the message and the contact it is about, with the contacts of the
// rest of its frame.
type step struct {
	msg   uint32
	c     pointerContact
	frame []pointerContact
	// handled is what the translator answers; true unless the step says it is not.
	notHandled bool
}

const (
	pDown   = win32.WMPointerDown
	pUpdate = win32.WMPointerUpdate
	pUp     = win32.WMPointerUp
	pEnter  = win32.WMPointerEnter
	pLeave  = win32.WMPointerLeave
	pCaptur = win32.WMPointerCaptureChanged

	inContact = win32.PointerFlagInContact
	inRange   = win32.PointerFlagInRange
)

func penAt(id, flags, penFlags uint32, x, y int32) pointerContact {
	return pointerContact{ID: id, Type: win32.PTPen, Flags: flags, PenFlags: penFlags, X: x, Y: y, Pressure: 720}
}

func touchAt(id, flags uint32, x, y int32) pointerContact {
	return pointerContact{ID: id, Type: win32.PTTouch, Flags: flags, X: x, Y: y}
}

func runSteps(t *testing.T, steps []step) (*pointerTracker, *pointerTranslator, []ui.PointerData) {
	t.Helper()
	tracker := newPointerTracker(tickingClock())
	tr := newPointerTranslator(tracker, 0)
	for i, s := range steps {
		if got := tr.handle(s.msg, s.c, s.frame); got == s.notHandled {
			t.Errorf("step %d: handled = %v", i, got)
		}
	}
	return tracker, tr, tracker.take()
}

func TestPointerTranslator(t *testing.T) {
	const touch0, pen0 = touchDeviceBase, stylusDeviceBase
	tests := []struct {
		name  string
		steps []step
		want  []ev
	}{
		// flutter_window_unittests.cc OnPointerStarSendsDeviceType, for a touch and a pen.
		{"OnPointerStarSendsDeviceType: a touch moves, goes down and up, and leaves", []step{
			{msg: pUpdate, c: touchAt(1, 0, 10, 10)},
			{msg: pDown, c: touchAt(1, inContact|win32.PointerFlagDown, 10, 10)},
			{msg: pUp, c: touchAt(1, win32.PointerFlagUp, 10, 10)},
			{msg: pLeave, c: touchAt(1, 0, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kTouch, device: touch0, x: 10, y: 10},
			{change: cHover, kind: kTouch, device: touch0, x: 10, y: 10},
			{change: cDown, kind: kTouch, device: touch0, x: 10, y: 10, buttons: buttonPrimary},
			{change: cUp, kind: kTouch, device: touch0, x: 10, y: 10},
			{change: cRemove, kind: kTouch, device: touch0, x: 10, y: 10},
		}},
		{"OnPointerStarSendsDeviceType: a pen moves, goes down and up, and leaves", []step{
			{msg: pUpdate, c: penAt(1, 0, 0, 10, 10)},
			{msg: pDown, c: penAt(1, inContact|win32.PointerFlagDown, 0, 10, 10)},
			{msg: pUp, c: penAt(1, win32.PointerFlagUp, 0, 10, 10)},
			{msg: pLeave, c: penAt(1, 0, 0, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cHover, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary},
			{change: cUp, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cRemove, kind: kStylus, device: pen0, x: 10, y: 10},
		}},
		{"OnStylusPointerDown", []step{{msg: pDown, c: penAt(1, inContact, 0, 10, 10)}}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary},
		}},
		{"OnStylusPointerMove: an update in contact is the down of a pen the platform has not seen", []step{
			{msg: pUpdate, c: penAt(1, inContact|win32.PointerFlagUpdate, 0, 15, 20)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 15, y: 20},
			{change: cDown, kind: kStylus, device: pen0, x: 15, y: 20, buttons: buttonPrimary},
		}},
		{"OnStylusPointerMove: and the next update is a move", []step{
			{msg: pDown, c: penAt(1, inContact, 0, 15, 20)},
			{msg: pUpdate, c: penAt(1, inContact|win32.PointerFlagUpdate, 0, 16, 22)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 15, y: 20},
			{change: cDown, kind: kStylus, device: pen0, x: 15, y: 20, buttons: buttonPrimary},
			{change: cMove, kind: kStylus, device: pen0, x: 16, y: 22, dx: 1, dy: 2, buttons: buttonPrimary},
		}},
		{"OnStylusPointerUp: the buttons are 0", []step{
			{msg: pDown, c: penAt(1, inContact, 0, 25, 30)},
			{msg: pUp, c: penAt(1, win32.PointerFlagUp, 0, 25, 30)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 25, y: 30},
			{change: cDown, kind: kStylus, device: pen0, x: 25, y: 30, buttons: buttonPrimary},
			{change: cUp, kind: kStylus, device: pen0, x: 25, y: 30},
		}},
		{"OnStylusPointerLeave: a leave of a pen the platform has not seen is ignored", []step{
			{msg: pLeave, c: penAt(1, win32.PointerFlagUp, 0, 35, 40)},
		}, nil},
		{"OnStylusPointerHover: an update that is not in contact ends the contact the pen had", []step{
			{msg: pDown, c: penAt(1, inContact, 0, 45, 50)},
			{msg: pUpdate, c: penAt(1, win32.PointerFlagUpdate, 0, 45, 50)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 45, y: 50},
			{change: cDown, kind: kStylus, device: pen0, x: 45, y: 50, buttons: buttonPrimary},
			{change: cUp, kind: kStylus, device: pen0, x: 45, y: 50},
		}},
		{"OnStylusHoverAfterPointerUp", []step{
			{msg: pDown, c: penAt(1, inContact, 0, 10, 10)},
			{msg: pUp, c: penAt(1, win32.PointerFlagUp, 0, 10, 10)},
			{msg: pUpdate, c: penAt(1, win32.PointerFlagUpdate, 0, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary},
			{change: cUp, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cHover, kind: kStylus, device: pen0, x: 10, y: 10},
		}},
		{"OnStylusBarrelButtonUsesPenFlags", []step{
			{msg: pDown, c: penAt(1, inContact, win32.PenFlagBarrel, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary | buttonSecondary},
		}},
		{"OnStylusEraserButtonUsesPenFlags", []step{
			{msg: pDown, c: penAt(1, inContact, win32.PenFlagEraser, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary | buttonMiddle},
		}},
		{"OnInvertedStylusPointerDownUsesDeviceKind", []step{
			{msg: pDown, c: penAt(1, inContact, win32.PenFlagInverted, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kInverted, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kInverted, device: pen0, x: 10, y: 10, buttons: buttonPrimary},
		}},
		{"OnStylusBarrelButtonUpdateMovesWithUpdatedButtons", []step{
			{msg: pDown, c: penAt(1, inContact, 0, 10, 10)},
			{msg: pUpdate, c: penAt(1, inContact, win32.PenFlagBarrel, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary},
			{change: cMove, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary | buttonSecondary},
		}},
		{"OnStylusBarrelButtonUpdateMovesWithReleasedButton", []step{
			{msg: pDown, c: penAt(1, inContact, win32.PenFlagBarrel, 10, 10)},
			{msg: pUpdate, c: penAt(1, inContact, 0, 10, 10)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 10, y: 10},
			{change: cDown, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary | buttonSecondary},
			{change: cMove, kind: kStylus, device: pen0, x: 10, y: 10, buttons: buttonPrimary},
		}},
		{"OnTouchPointerDown", []step{
			{msg: pDown, c: touchAt(1, inContact|win32.PointerFlagFirstButton|win32.PointerFlagDown, 55, 60)},
		}, []ev{
			{change: cAdd, kind: kTouch, device: touch0, x: 55, y: 60},
			{change: cDown, kind: kTouch, device: touch0, x: 55, y: 60, buttons: buttonPrimary},
		}},
		{"OnMousePointerDown: a mouse pointer is for DefWindowProc, which makes the classic messages", []step{
			{msg: pDown, c: pointerContact{ID: 1, Type: win32.PTMouse, Flags: inContact | win32.PointerFlagFirstButton | win32.PointerFlagDown, X: 45, Y: 50}, notHandled: true},
		}, nil},
		{"a touchpad pointer is for DefWindowProc too", []step{
			{msg: pUpdate, c: pointerContact{ID: 1, Type: win32.PTTouchpad, X: 1, Y: 1}, notHandled: true},
		}, nil},
		{"a pen in range is added, hovers, and is removed when it leaves", []step{
			{msg: pEnter, c: penAt(1, inRange, 0, 5, 6)},
			{msg: pUpdate, c: penAt(1, inRange|win32.PointerFlagUpdate, 0, 7, 8)},
			{msg: pLeave, c: penAt(1, 0, 0, 7, 8)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 5, y: 6},
			{change: cHover, kind: kStylus, device: pen0, x: 7, y: 8, dx: 2, dy: 2},
			{change: cRemove, kind: kStylus, device: pen0, x: 7, y: 8},
		}},
		{"a pen that leaves while it touches is lifted first", []step{
			{msg: pDown, c: penAt(1, inContact, 0, 5, 6)},
			{msg: pLeave, c: penAt(1, 0, 0, 5, 6)},
		}, []ev{
			{change: cAdd, kind: kStylus, device: pen0, x: 5, y: 6},
			{change: cDown, kind: kStylus, device: pen0, x: 5, y: 6, buttons: buttonPrimary},
			{change: cUp, kind: kStylus, device: pen0, x: 5, y: 6, synth: true},
			{change: cRemove, kind: kStylus, device: pen0, x: 5, y: 6},
		}},
		{"a lost capture cancels the contact", []step{
			{msg: pDown, c: touchAt(1, inContact, 5, 6)},
			{msg: pCaptur, c: touchAt(1, 0, 5, 6)},
		}, []ev{
			{change: cAdd, kind: kTouch, device: touch0, x: 5, y: 6},
			{change: cDown, kind: kTouch, device: touch0, x: 5, y: 6, buttons: buttonPrimary},
			{change: cCancel, kind: kTouch, device: touch0, x: 5, y: 6},
			{change: cRemove, kind: kTouch, device: touch0, x: 5, y: 6},
		}},
		{"a pen keeps the kind it came with when the eraser flag changes in between", []step{
			{msg: pDown, c: penAt(1, inContact, win32.PenFlagInverted, 5, 6)},
			{msg: pUpdate, c: penAt(1, inContact, 0, 6, 6)},
		}, []ev{
			{change: cAdd, kind: kInverted, device: pen0, x: 5, y: 6},
			{change: cDown, kind: kInverted, device: pen0, x: 5, y: 6, buttons: buttonPrimary},
			{change: cMove, kind: kInverted, device: pen0, x: 6, y: 6, dx: 1, buttons: buttonPrimary},
		}},
		{"two fingers in one frame are one batch, delivered with the first message", []step{
			{msg: pDown, c: touchAt(10, inContact|win32.PointerFlagDown, 1, 1), frame: []pointerContact{touchAt(11, inContact|win32.PointerFlagDown, 2, 2)}},
			{msg: pDown, c: touchAt(11, inContact|win32.PointerFlagDown, 2, 2), frame: []pointerContact{touchAt(10, inContact|win32.PointerFlagDown, 1, 1)}},
			{msg: pUpdate, c: touchAt(10, inContact, 3, 3), frame: []pointerContact{touchAt(11, inContact, 4, 4)}},
			{msg: pUpdate, c: touchAt(11, inContact, 4, 4), frame: []pointerContact{touchAt(10, inContact, 3, 3)}},
		}, []ev{
			{change: cAdd, kind: kTouch, device: touch0, x: 1, y: 1},
			{change: cDown, kind: kTouch, device: touch0, x: 1, y: 1, buttons: buttonPrimary},
			{change: cAdd, kind: kTouch, device: touch0 + 1, x: 2, y: 2},
			{change: cDown, kind: kTouch, device: touch0 + 1, x: 2, y: 2, buttons: buttonPrimary},
			{change: cMove, kind: kTouch, device: touch0, x: 3, y: 3, dx: 2, dy: 2, buttons: buttonPrimary},
			{change: cMove, kind: kTouch, device: touch0 + 1, x: 4, y: 4, dx: 2, dy: 2, buttons: buttonPrimary},
		}},
		{"a message of another kind for a contact of the frame is not dropped", []step{
			{msg: pUpdate, c: touchAt(10, inContact, 1, 1), frame: []pointerContact{touchAt(11, inContact, 2, 2)}},
			{msg: pLeave, c: touchAt(11, 0, 2, 2)},
		}, []ev{
			// The first update is the down of 10 and of 11; the leave is not the update that was done.
			{change: cAdd, kind: kTouch, device: touch0, x: 1, y: 1},
			{change: cDown, kind: kTouch, device: touch0, x: 1, y: 1, buttons: buttonPrimary},
			{change: cAdd, kind: kTouch, device: touch0 + 1, x: 2, y: 2},
			{change: cDown, kind: kTouch, device: touch0 + 1, x: 2, y: 2, buttons: buttonPrimary},
			{change: cUp, kind: kTouch, device: touch0 + 1, x: 2, y: 2, synth: true},
			{change: cRemove, kind: kTouch, device: touch0 + 1, x: 2, y: 2},
		}},
		{"the id of a contact that left is given again", []step{
			{msg: pDown, c: touchAt(500, inContact, 1, 1)},
			{msg: pDown, c: touchAt(501, inContact, 2, 2)},
			{msg: pLeave, c: touchAt(500, 0, 1, 1)},
			{msg: pDown, c: touchAt(777, inContact, 3, 3)},
		}, []ev{
			{change: cAdd, kind: kTouch, device: touch0, x: 1, y: 1},
			{change: cDown, kind: kTouch, device: touch0, x: 1, y: 1, buttons: buttonPrimary},
			{change: cAdd, kind: kTouch, device: touch0 + 1, x: 2, y: 2},
			{change: cDown, kind: kTouch, device: touch0 + 1, x: 2, y: 2, buttons: buttonPrimary},
			{change: cUp, kind: kTouch, device: touch0, x: 1, y: 1, synth: true},
			{change: cRemove, kind: kTouch, device: touch0, x: 1, y: 1},
			{change: cAdd, kind: kTouch, device: touch0, x: 3, y: 3},
			{change: cDown, kind: kTouch, device: touch0, x: 3, y: 3, buttons: buttonPrimary},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, got := runSteps(t, tt.steps)
			sameEvents(t, got, tt.want)
		})
	}
}

func TestPointerDetails(t *testing.T) {
	type detail struct {
		pressure, pressureMax, tilt, orientation, radiusMajor, radiusMinor float64
	}
	pen := func(flags uint32, mod func(*pointerContact)) pointerContact {
		c := penAt(1, flags, 0, 10, 10)
		if mod != nil {
			mod(&c)
		}
		return c
	}
	touch := func(mod func(*pointerContact)) pointerContact {
		c := touchAt(1, inContact, 10, 10)
		if mod != nil {
			mod(&c)
		}
		return c
	}
	tests := []struct {
		name string
		c    pointerContact
		want detail
	}{
		{"a pen's pressure is 0 to 1024 divided by 1024", pen(inContact, func(c *pointerContact) { c.Pressure = 720 }), detail{pressure: 720.0 / 1024, pressureMax: 1}},
		{"full pressure is 1", pen(inContact, func(c *pointerContact) { c.Pressure = 1024 }), detail{pressure: 1, pressureMax: 1}},
		{"a pressure above the range is 1", pen(inContact, func(c *pointerContact) { c.Pressure = 2000 }), detail{pressure: 1, pressureMax: 1}},
		{"a pen in the air presses nothing, whatever the driver says", pen(inRange, func(c *pointerContact) { c.Pressure = 720 }), detail{pressureMax: 1}},
		{"the rotation is the orientation in radians", pen(inContact, func(c *pointerContact) { c.Pressure, c.Rotation = 512, 90 }), detail{pressure: 0.5, pressureMax: 1, orientation: math.Pi / 2}},
		{"Flutter's rotation of 10 degrees", pen(inContact, func(c *pointerContact) { c.Rotation = 10 }), detail{pressure: 720.0 / 1024, pressureMax: 1, orientation: 10 * math.Pi / 180}},
		{"a tilt of 45 degrees along x", pen(inContact, func(c *pointerContact) { c.Pressure, c.TiltX = 0, 45 }), detail{pressureMax: 1, tilt: math.Pi / 4}},
		{"a tilt along both axes is the angle of the vector", pen(inContact, func(c *pointerContact) { c.Pressure, c.TiltX, c.TiltY = 0, 45, 45 }), detail{pressureMax: 1, tilt: math.Atan(math.Sqrt2)}},
		{"a negative tilt is the same angle", pen(inContact, func(c *pointerContact) { c.Pressure, c.TiltY = 0, -30 }), detail{pressureMax: 1, tilt: math.Pi / 6}},
		{"a tilt of 90 degrees is the flat pen, pi/2", pen(inContact, func(c *pointerContact) { c.Pressure, c.TiltX = 0, 90 }), detail{pressureMax: 1, tilt: math.Pi / 2}},
		{"a touch with no pressure information is 1", touch(nil), detail{pressure: 1, pressureMax: 1}},
		{"a touch with pressure", touch(func(c *pointerContact) { c.TouchPressure = 256 }), detail{pressure: 0.25, pressureMax: 1}},
		{"the contact ellipse: half the longer side is the major radius", touch(func(c *pointerContact) { c.ContactWidth, c.ContactHeight = 40, 20 }), detail{pressure: 1, pressureMax: 1, radiusMajor: 20, radiusMinor: 10}},
		{"the contact ellipse, the other way round", touch(func(c *pointerContact) { c.ContactWidth, c.ContactHeight = 10, 30 }), detail{pressure: 1, pressureMax: 1, radiusMajor: 15, radiusMinor: 5}},
		{"the orientation of the ellipse in radians", touch(func(c *pointerContact) { c.Orientation = 180 }), detail{pressure: 1, pressureMax: 1, orientation: math.Pi}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := uint32(pDown)
			if tt.c.Flags&inContact == 0 {
				msg = pUpdate // a pen in the air hovers
			}
			_, _, got := runSteps(t, []step{{msg: msg, c: tt.c}})
			if len(got) == 0 {
				t.Fatal("no events")
			}
			// The down (or the hover) is the last event of the message, after the add.
			d := got[len(got)-1]
			g := detail{d.Pressure, d.PressureMax, d.Tilt, d.Orientation, d.RadiusMajor, d.RadiusMinor}
			if !near(g.pressure, tt.want.pressure) || !near(g.pressureMax, tt.want.pressureMax) || !near(g.tilt, tt.want.tilt) ||
				!near(g.orientation, tt.want.orientation) || !near(g.radiusMajor, tt.want.radiusMajor) || !near(g.radiusMinor, tt.want.radiusMinor) {
				t.Errorf("details %+v, want %+v", g, tt.want)
			}
		})
	}
}

func TestPointerPressureIsZeroWhereNothingPresses(t *testing.T) {
	_, _, got := runSteps(t, []step{
		{msg: pDown, c: touchAt(1, inContact, 5, 5)},
		{msg: pUp, c: touchAt(1, win32.PointerFlagUp, 5, 5)},
		{msg: pLeave, c: touchAt(1, 0, 5, 5)},
	})
	for _, d := range got {
		want := 0.0
		if d.Change == cDown {
			want = 1
		}
		if d.Pressure != want {
			t.Errorf("%v has pressure %v, want %v", d.Change, d.Pressure, want)
		}
	}
}

func TestContactOfInfo(t *testing.T) {
	info := win32.PointerInfo{PointerType: win32.PTPen, PointerID: 7, FrameID: 9, PointerFlags: inContact, PixelLocation: win32.Point{X: 125, Y: 135}}
	pen := &win32.PointerPenInfo{PenFlags: win32.PenFlagBarrel, Pressure: 720, Rotation: 10, TiltX: -5, TiltY: 6}
	touchInfo := win32.PointerInfo{PointerType: win32.PTTouch, PointerID: 8, PixelLocation: win32.Point{X: 20, Y: 30}}
	touch := &win32.PointerTouchInfo{Contact: win32.Rect{Left: 10, Top: 20, Right: 50, Bottom: 40}, Orientation: 90, Pressure: 100}
	// PointerMessageScreenCoordinatesAreConvertedToClient: the window is at (110, 115).
	toClient := func(p win32.Point) win32.Point { return win32.Point{X: p.X - 110, Y: p.Y - 115} }

	got := contactOfInfo(info, pen, nil, toClient)
	want := pointerContact{ID: 7, Type: win32.PTPen, Frame: 9, Flags: inContact, X: 15, Y: 20, PenFlags: win32.PenFlagBarrel, Pressure: 720, Rotation: 10, TiltX: -5, TiltY: 6}
	if got != want {
		t.Errorf("pen contact = %+v, want %+v", got, want)
	}
	got = contactOfInfo(touchInfo, nil, touch, nil)
	want = pointerContact{ID: 8, Type: win32.PTTouch, X: 20, Y: 30, ContactWidth: 40, ContactHeight: 20, Orientation: 90, TouchPressure: 100}
	if got != want {
		t.Errorf("touch contact = %+v, want %+v", got, want)
	}
	if got := contactOfInfo(info, nil, nil, nil); got.PenFlags != 0 || got.Pressure != 0 || got.X != 125 {
		t.Errorf("contact without the pen information = %+v", got)
	}
}

func TestPenButtonsAndTouchButtons(t *testing.T) {
	tests := []struct {
		name         string
		penFlags     uint32
		pointerFlags uint32
		want         int
	}{
		{"in the air: no buttons even with the barrel button", win32.PenFlagBarrel, inRange, 0},
		{"the tip", 0, inContact, buttonPrimary},
		{"the tip and the barrel button", win32.PenFlagBarrel, inContact, buttonPrimary | buttonSecondary},
		{"the tip and the eraser button", win32.PenFlagEraser, inContact, buttonPrimary | buttonMiddle},
		{"the eraser end alone is the tip of the eraser", win32.PenFlagInverted, inContact, buttonPrimary},
		{"all of them", win32.PenFlagBarrel | win32.PenFlagEraser, inContact, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := penButtons(tt.penFlags, tt.pointerFlags); got != tt.want {
				t.Errorf("penButtons = %#x, want %#x", got, tt.want)
			}
		})
	}
	if touchButtons(inContact) != buttonPrimary || touchButtons(inRange) != 0 || touchButtons(0) != 0 {
		t.Error("a touch is the primary button while it is in contact, and nothing else")
	}
}

func TestMessageOfFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags uint32
		want  uint32
	}{
		{"down", inContact | win32.PointerFlagDown, pDown},
		{"up", win32.PointerFlagUp, pUp},
		{"update", inContact | win32.PointerFlagUpdate, pUpdate},
		{"no flag", 0, pUpdate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageOfFlags(tt.flags); got != tt.want {
				t.Errorf("messageOfFlags = %#x, want %#x", got, tt.want)
			}
		})
	}
}

func TestPointerTranslatorGoneAndCancelAll(t *testing.T) {
	tracker := newPointerTracker(tickingClock())
	tr := newPointerTranslator(tracker, 0)
	tr.handle(pDown, touchAt(1, inContact, 5, 5), nil)
	tr.handle(pDown, touchAt(2, inContact, 6, 6), nil)
	tracker.take()
	// The system has forgotten the contact 1: its leave is handled from what the platform knows.
	if !tr.handleGone(pLeave, 1) {
		t.Error("the leave of a known contact was not handled")
	}
	if tr.handleGone(pLeave, 1) {
		t.Error("the leave of a contact that is gone was handled twice")
	}
	if tr.handleGone(pCaptur, 99) {
		t.Error("a contact the translator never saw was handled")
	}
	sameEvents(t, tracker.take(), []ev{
		{change: cUp, kind: kTouch, device: touchDeviceBase, x: 5, y: 5, synth: true},
		{change: cRemove, kind: kTouch, device: touchDeviceBase, x: 5, y: 5},
	})
	tr.cancelAll()
	sameEvents(t, tracker.take(), []ev{
		{change: cCancel, kind: kTouch, device: touchDeviceBase + 1, x: 6, y: 6},
		{change: cRemove, kind: kTouch, device: touchDeviceBase + 1, x: 6, y: 6},
	})
	if len(tr.active) != 0 || len(tr.pending) != 0 {
		t.Errorf("the translator remembers %d contacts and %d messages", len(tr.active), len(tr.pending))
	}
}
