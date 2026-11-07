package gamepad_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/pads"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/sensors"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// app pumps an app that reads the pads, and returns the gamepads of its binding.
func app(t *testing.T, opts ...gluttertest.Opts) (*gluttertest.Tester, *gamepad.Gamepads) {
	t.Helper()
	ts := gluttertest.New(t, opts...)
	ts.PumpWidget(widgets.NewSizedBox(nil))
	return ts, gamepad.ForBinding(ts.Binding)
}

func TestTheEnumsFollowTheMappingOrder(t *testing.T) {
	for b := gamepad.Button(0); int(b) < gamepad.ButtonCount; b++ {
		if got, want := b.String(), gamepaddb.Button(b).Name(); got != want {
			t.Errorf("Button(%d) is %q, the mapping calls it %q", b, got, want)
		}
	}
	for a := gamepad.Axis(0); int(a) < gamepad.AxisCount; a++ {
		if got, want := a.String(), gamepaddb.Axis(a).Name(); got != want {
			t.Errorf("Axis(%d) is %q, the mapping calls it %q", a, got, want)
		}
	}
	if gamepad.Button(200).String() != "unknown" || gamepad.Axis(200).String() != "unknown" {
		t.Error("an unknown button or axis has no name")
	}
	if got := gamepad.South.String() + "/" + gamepad.LeftTrigger.String(); got != "a/lefttrigger" {
		t.Errorf("names = %s", got)
	}
}

func TestTheKeysAreThoseOfTheServices(t *testing.T) {
	tests := []struct {
		button gamepad.Button
		key    services.LogicalKeyboardKey
	}{
		{gamepad.South, services.LogicalKeyboardKeyGameButtonA},
		{gamepad.East, services.LogicalKeyboardKeyGameButtonB},
		{gamepad.West, services.LogicalKeyboardKeyGameButtonX},
		{gamepad.North, services.LogicalKeyboardKeyGameButtonY},
		{gamepad.LeftShoulder, services.LogicalKeyboardKeyGameButtonLeft1},
		{gamepad.RightShoulder, services.LogicalKeyboardKeyGameButtonRight1},
		{gamepad.Start, services.LogicalKeyboardKeyGameButtonStart},
		{gamepad.Back, services.LogicalKeyboardKeyGameButtonSelect},
		{gamepad.Guide, services.LogicalKeyboardKeyGameButtonMode},
		{gamepad.LeftStick, services.LogicalKeyboardKeyGameButtonThumbLeft},
		{gamepad.RightStick, services.LogicalKeyboardKeyGameButtonThumbRight},
		{gamepad.DPadUp, services.LogicalKeyboardKeyArrowUp},
		{gamepad.DPadDown, services.LogicalKeyboardKeyArrowDown},
		{gamepad.DPadLeft, services.LogicalKeyboardKeyArrowLeft},
		{gamepad.DPadRight, services.LogicalKeyboardKeyArrowRight},
	}
	for _, tt := range tests {
		t.Run(tt.button.String(), func(t *testing.T) {
			physical, logical, ok := pads.KeyIDs(gamepaddb.Button(tt.button))
			if !ok {
				t.Fatalf("%s makes no key", tt.button)
			}
			if logical != tt.key.KeyID() {
				t.Errorf("logical key = %#x, want %#x", logical, tt.key.KeyID())
			}
			wantPhysical, found := services.PhysicalKeyOf(tt.key)
			if !found {
				t.Fatalf("no physical key for %v", tt.key)
			}
			if physical != wantPhysical.USBHidUsage() {
				t.Errorf("physical key = %#x, want %#x", physical, wantPhysical.USBHidUsage())
			}
		})
	}
	for i, key := range []services.LogicalKeyboardKey{services.LogicalKeyboardKeyGameButtonLeft2, services.LogicalKeyboardKeyGameButtonRight2} {
		physical, logical := pads.TriggerKeyIDs(i)
		want, _ := services.PhysicalKeyOf(key)
		if logical != key.KeyID() || physical != want.USBHidUsage() {
			t.Errorf("trigger %d: key %#x/%#x, want %#x/%#x", i, physical, logical, want.USBHidUsage(), key.KeyID())
		}
	}
	for _, b := range []gamepad.Button{gamepad.Misc1, gamepad.Paddle1, gamepad.Touchpad} {
		if _, _, ok := pads.KeyIDs(gamepaddb.Button(b)); ok {
			t.Errorf("%s makes a key, and has no game-button key to make", b)
		}
	}
}

func TestAPadIsConnectedAndRemoved(t *testing.T) {
	ts, pads := app(t)
	calls := 0
	pads.AddListener(func() { calls++ })
	if len(pads.Pads()) != 0 || !pads.Available() {
		t.Fatalf("a new app has pads %v or no service", pads.Pads())
	}

	a := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad A"})
	b := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad B"})
	if calls != 2 {
		t.Errorf("listeners were called %d times for two pads, want 2", calls)
	}
	list := pads.Pads()
	if len(list) != 2 || list[0].Name() != "Pad A" || list[1].Name() != "Pad B" || list[0].ID() != a.ID() || list[1].ID() != b.ID() {
		t.Fatalf("Pads() = %v, want A then B", names(list))
	}
	if list[0] != a.Pad() || list[0] != pads.Pads()[0] {
		t.Error("a pad is not the same value each time")
	}
	if !list[0].Connected() || list[0].Kind() != gamepad.KindGeneric || list[0].FaceStyle() != gamepad.FaceABXY || list[0].GUID() == "" {
		t.Errorf("pad A = connected %v, kind %v, face %v, guid %q", list[0].Connected(), list[0].Kind(), list[0].FaceStyle(), list[0].GUID())
	}

	a.Disconnect()
	if calls != 3 || len(pads.Pads()) != 1 || pads.Pads()[0].Name() != "Pad B" {
		t.Errorf("after removing A: calls %d, pads %v", calls, names(pads.Pads()))
	}
	if list[0].Connected() {
		t.Error("a removed pad says it is connected")
	}
	if a.Pad() != nil {
		t.Error("TestPad.Pad() returned a removed pad")
	}
}

func names(pads []*gamepad.Pad) []string {
	var out []string
	for _, p := range pads {
		out = append(out, p.Name())
	}
	return out
}

func TestAPadReportsItsButtonsAndAxes(t *testing.T) {
	ts, pads := app(t)
	tp := ts.ConnectPad()
	pad := pads.Pads()[0]
	var got []gamepad.Event
	remove := pad.AddListener(func(e gamepad.Event) { got = append(got, e) })

	tp.Press(gamepad.South)
	tp.SetAxis(gamepad.LeftX, 1)
	tp.SetAxis(gamepad.LeftY, -1)
	tp.SetAxis(gamepad.RightTrigger, 0.5)
	if !pad.State().Down(gamepad.South) || pad.State().Down(gamepad.East) {
		t.Error("the state of the buttons is wrong")
	}
	if v := pad.State().Axis(gamepad.LeftX); v != 1 {
		t.Errorf("LeftX = %v, want 1", v)
	}
	if v := pad.State().Axis(gamepad.LeftY); v != -1 {
		t.Errorf("LeftY = %v, want -1", v)
	}
	if v := pad.State().Axis(gamepad.RightTrigger); v < 0.4999 || v > 0.5001 {
		t.Errorf("RightTrigger = %v, want 0.5", v)
	}
	tp.Release(gamepad.South)
	remove()
	tp.Press(gamepad.East)

	want := []struct {
		kind   gamepad.EventKind
		button gamepad.Button
		axis   gamepad.Axis
	}{
		{gamepad.ButtonDown, gamepad.South, 0},
		{gamepad.AxisMotion, 0, gamepad.LeftX},
		{gamepad.AxisMotion, 0, gamepad.LeftY},
		{gamepad.AxisMotion, 0, gamepad.RightTrigger},
		{gamepad.ButtonUp, gamepad.South, 0},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		e := got[i]
		if e.Kind != w.kind || e.Pad != pad {
			t.Errorf("event %d = %+v, want kind %v", i, e, w.kind)
		}
		if w.kind != gamepad.AxisMotion && e.Button != w.button {
			t.Errorf("event %d button = %v, want %v", i, e.Button, w.button)
		}
		if w.kind == gamepad.AxisMotion && e.Axis != w.axis {
			t.Errorf("event %d axis = %v, want %v", i, e.Axis, w.axis)
		}
	}
	if !pad.State().Down(gamepad.East) {
		t.Error("the state follows the pad when no listener is there")
	}
}

func TestAPadThatIsRemovedLetsGo(t *testing.T) {
	ts, pads := app(t)
	tp := ts.ConnectPad()
	pad := pads.Pads()[0]
	var got []string
	pad.AddListener(func(e gamepad.Event) {
		got = append(got, e.Button.String()+e.Axis.String())
	})
	tp.Press(gamepad.North)
	tp.SetAxis(gamepad.LeftX, 0.5)
	got = nil
	tp.Disconnect()
	if len(got) != 2 {
		t.Errorf("removing a pad that holds a button and an axis gave %v, want two releases", got)
	}
	if pad.State().Down(gamepad.North) || pad.State().Axis(gamepad.LeftX) != 0 {
		t.Error("a removed pad still holds its inputs")
	}
}

func TestApplyDeadZone(t *testing.T) {
	tests := []struct {
		name          string
		v, dead, want float64
	}{
		{"inside", 0.1, 0.2, 0},
		{"on the edge", 0.2, 0.2, 0},
		{"outside, scaled", 0.6, 0.2, 0.5},
		{"negative", -0.6, 0.2, -0.5},
		{"full", 1, 0.2, 1},
		{"no dead zone", 0.3, 0, 0.3},
		{"all dead", 0.9, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gamepad.ApplyDeadZone(tt.v, tt.dead); got < tt.want-1e-12 || got > tt.want+1e-12 {
				t.Errorf("ApplyDeadZone(%v, %v) = %v, want %v", tt.v, tt.dead, got, tt.want)
			}
		})
	}
	x, y := gamepad.ApplyStickDeadZone(0.1, 0.1, 0.2)
	if x != 0 || y != 0 {
		t.Errorf("a small push gave %v,%v", x, y)
	}
	x, y = gamepad.ApplyStickDeadZone(0.3, 0.4, 0.2)
	if d := x*x + y*y; d < 0.140625-1e-9 || d > 0.140625+1e-9 || x/y < 0.75-1e-9 || x/y > 0.75+1e-9 {
		t.Errorf("a diagonal push gave %v,%v: its size and direction are wrong", x, y)
	}
	x, y = gamepad.ApplyStickDeadZone(1, 1, 0.2)
	if x*x+y*y > 1+1e-9 {
		t.Errorf("the result %v,%v is longer than 1", x, y)
	}
	if x, y = gamepad.ApplyStickDeadZone(0, 0, 0); x != 0 || y != 0 {
		t.Errorf("the centre gave %v,%v", x, y)
	}
}

// rawPS5 is a raw pad with the numbering of a DualSense under the Linux driver on Bluetooth: its GUID is in the table.
var rawPS5 = gluttertest.PadSpec{
	Name: "Wireless Controller", GUID: "030000004c050000e60c000011810000", Axes: 6, Buttons: 15, Hats: 1,
}

func TestARawPadIsMappedFromTheTable(t *testing.T) {
	ts, pads := app(t)
	tp := ts.ConnectPad(rawPS5)
	list := pads.Pads()
	if len(list) != 1 {
		t.Fatalf("the DualSense is not a gamepad: %v", names(list))
	}
	pad := list[0]
	if pad.Kind() != gamepad.KindPlayStation5 || pad.FaceStyle() != gamepad.FaceSony || pad.Name() != "PS5 Controller" {
		t.Errorf("Kind() = %v, FaceStyle() = %v, Name() = %q, want a PlayStation 5 pad", pad.Kind(), pad.FaceStyle(), pad.Name())
	}
	for _, b := range []gamepad.Button{gamepad.South, gamepad.East, gamepad.West, gamepad.North, gamepad.Start, gamepad.DPadUp} {
		if !pad.HasButton(b) {
			t.Errorf("the pad has no %s", b)
		}
	}
	if !pad.HasAxis(gamepad.LeftTrigger) || pad.HasButton(gamepad.Paddle1) {
		t.Error("HasAxis or HasButton is wrong")
	}
	var downs []gamepad.Button
	pad.AddListener(func(e gamepad.Event) {
		if e.Kind == gamepad.ButtonDown {
			downs = append(downs, e.Button)
		}
	})
	tp.RawButton(0, true)
	tp.RawButton(3, true)
	tp.RawHat(0, 1)
	tp.RawHat(0, 0)
	tp.RawHat(0, 2)
	want := []gamepad.Button{gamepad.South, gamepad.West, gamepad.DPadUp, gamepad.DPadRight}
	if !reflect.DeepEqual(downs, want) {
		t.Errorf("buttons that went down = %v, want %v", downs, want)
	}
}

func TestAPadWithoutAMappingBecomesOneWhenTheAppAddsOne(t *testing.T) {
	ts, pads := app(t)
	notified := 0
	pads.AddListener(func() { notified++ })
	guid := "03000000aaaa00000001000001000000"
	tp := ts.ConnectPad(gluttertest.PadSpec{Name: "No Name Pad", GUID: guid, Axes: 2, Buttons: 4})
	if len(pads.Pads()) != 0 || notified != 0 {
		t.Fatalf("a pad that nothing is known of is a gamepad: %v", names(pads.Pads()))
	}
	n, err := pads.AddMappings("# comment\n" + guid + ",No Name Pad,a:b2,b:b3,leftx:a0,lefty:a1,platform:Windows,\n" +
		guid + ",Custom Name,a:b2,b:b3,leftx:a0,lefty:a1,platform:Linux,\n")
	if err != nil || n != 1 {
		t.Fatalf("AddMappings() = %d, %v, want 1 mapping (the one for Linux)", n, err)
	}
	if notified != 1 || len(pads.Pads()) != 1 {
		t.Fatalf("after the mapping: notified %d, pads %v", notified, names(pads.Pads()))
	}
	pad := pads.Pads()[0]
	if pad.Name() != "Custom Name" {
		t.Errorf("Name() = %q, want the name of the mapping", pad.Name())
	}
	var got []gamepad.Button
	pad.AddListener(func(e gamepad.Event) { got = append(got, e.Button) })
	tp.RawButton(2, true)
	if !reflect.DeepEqual(got, []gamepad.Button{gamepad.South}) {
		t.Errorf("raw button 2 gave %v, want South", got)
	}

	// A new mapping for the same pad changes what its buttons are.
	if _, err := pads.AddMappings(guid + ",Custom Name,a:b0,b:b1,leftx:a0,lefty:a1,platform:Linux,\n"); err != nil {
		t.Fatal(err)
	}
	got = nil
	tp.RawButton(2, false)
	tp.RawButton(0, true)
	if !reflect.DeepEqual(got, []gamepad.Button{gamepad.South}) {
		t.Errorf("after the new mapping raw button 0 gave %v, want South", got)
	}
}

func TestARawPadUsesTheMappingTheBackendMade(t *testing.T) {
	ts, pads := app(t)
	tp := ts.ConnectPad(gluttertest.PadSpec{Name: "Spec Pad", GUID: "03000000bbbb00000002000001000000", Axes: 4, Buttons: 3,
		Mapping: "a:b0,b:b1,x:b2,leftx:a0,lefty:a1,rightx:a2,righty:a3,"})
	list := pads.Pads()
	if len(list) != 1 || list[0].Name() != "Spec Pad" {
		t.Fatalf("pads = %v", names(list))
	}
	var axes []float64
	list[0].AddListener(func(e gamepad.Event) {
		if e.Kind == gamepad.AxisMotion {
			axes = append(axes, e.Value)
		}
	})
	tp.RawAxis(0, 0)
	tp.RawAxis(0, 32767)
	tp.RawAxis(0, -32768)
	if !reflect.DeepEqual(axes, []float64{1, -1}) {
		t.Errorf("axis values = %v, want [1 -1]", axes)
	}
}

func TestAddMappingsRejectsABrokenLine(t *testing.T) {
	_, pads := app(t)
	n, err := pads.AddMappings("03000000cccc00000003000001000000,Good,a:b0,platform:Linux,\n" +
		"03000000dddd00000004000001000000,Bad,aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:b0,platform:Linux,\n" +
		"03000000eeee00000005000001000000,After,a:b0,platform:Linux,\n")
	if n != 1 {
		t.Errorf("added %d mappings, want the 1 before the broken line", n)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != gamepad.TagMappingInvalid || !appErr.HasRoute("Gamepads.AddMappings->Manager.AddMappings->Table.AddText->Table.add->ParseBindings.GamepadMappingInvalid") {
		t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
	}
}

func TestRumbleAndTheLight(t *testing.T) {
	ts, pads := app(t)
	capable := ts.ConnectPad(gluttertest.PadSpec{Name: "Rumbler", Rumble: true, RumbleTriggers: true, LED: true})
	plain := ts.ConnectPad(gluttertest.PadSpec{Name: "Plain"})
	list := pads.Pads()
	if !list[0].CanRumble() || !list[0].CanRumbleTriggers() || !list[0].HasLED() || list[1].CanRumble() || list[1].HasLED() {
		t.Error("CanRumble, CanRumbleTriggers or HasLED is wrong")
	}
	if err := list[0].Rumble(0.5, 2, 250*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := list[0].RumbleTriggers(1, -1, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := list[0].SetLED(ui.Color{R: 1, G: 0.5, B: 0, A: 1}); err != nil {
		t.Fatal(err)
	}
	want := []gluttertest.PadRumble{
		{Low: 32768, High: 65535, Duration: 250 * time.Millisecond},
		{Triggers: true, Low: 65535, High: 0, Duration: time.Second},
		{LED: true, Color: [3]uint8{255, 128, 0}},
	}
	if got := capable.Rumbles(); !reflect.DeepEqual(got, want) {
		t.Errorf("Rumbles() = %+v, want %+v", got, want)
	}
	if got := plain.Rumbles(); len(got) != 0 {
		t.Errorf("the plain pad rumbled: %+v", got)
	}

	tests := []struct {
		name  string
		do    func() error
		route string
		tag   string
	}{
		{"rumble", func() error { return list[1].Rumble(1, 1, time.Second) }, "Pad.Rumble->Pad.Rumble->Gamepads.Rumble.GamepadRumbleUnavailable", gamepad.TagRumbleUnavailable},
		{"triggers", func() error { return list[1].RumbleTriggers(1, 1, time.Second) }, "Pad.RumbleTriggers->Pad.RumbleTriggers->Gamepads.RumbleTriggers.GamepadRumbleUnavailable", gamepad.TagRumbleUnavailable},
		{"light", func() error { return list[1].SetLED(ui.Color{}) }, "Pad.SetLED->Pad.SetLED->Gamepads.SetLED.GamepadLEDUnavailable", gamepad.TagLEDUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.do()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestTheBatteryIsUnknownWithoutAReport(t *testing.T) {
	ts, pads := app(t)
	ts.ConnectPad()
	if b := pads.Pads()[0].Battery(); b.Known {
		t.Errorf("Battery() = %+v, want unknown", b)
	}
}

func TestTheBatteryFollowsTheReports(t *testing.T) {
	ts, pads := app(t)
	pad := ts.ConnectPad()
	p := pads.Pads()[0]
	pad.SetBattery(0.55, true)
	if b := p.Battery(); !b.Known || b.Level != 0.55 || !b.Charging || b.Charged {
		t.Errorf("Battery() = %+v", b)
	}
	pad.SetBattery(0.25, false)
	if b := p.Battery(); !b.Known || b.Level != 0.25 || b.Charging {
		t.Errorf("Battery() = %+v", b)
	}
}

func TestTheTouchpadReportsFingers(t *testing.T) {
	ts, pads := app(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Touchpads: 1})
	p := pads.Pads()[0]
	if p.Touchpads() != 1 {
		t.Fatalf("Touchpads() = %d", p.Touchpads())
	}
	var got []gamepad.Touch
	remove := p.AddTouchListener(func(tc gamepad.Touch) { got = append(got, tc) })

	pad.Touch(0, 0, true, 0.25, 0.75)
	pad.Touch(0, 1, true, 0.5, 0.5)
	if len(got) != 2 || got[0].Pad != p || !got[0].Down || got[0].X != 0.25 || got[0].Y != 0.75 || got[0].Pressure != 1 || got[1].Finger != 1 {
		t.Fatalf("touches = %+v", got)
	}
	if fingers := p.Touches(); len(fingers) != 2 || fingers[0].X != 0.25 {
		t.Errorf("Touches() = %+v", fingers)
	}
	pad.Touch(0, 0, false, 0.25, 0.75)
	if len(got) != 3 || got[2].Down || got[2].Pressure != 0 || len(p.Touches()) != 1 {
		t.Errorf("after a finger left: %+v", got)
	}
	remove()
	pad.Touch(0, 0, true, 0.1, 0.1)
	if len(got) != 3 {
		t.Error("a listener that was removed was called")
	}
	// A pad without a touchpad has none.
	ts.ConnectPad()
	if n := pads.Pads()[1].Touchpads(); n != 0 {
		t.Errorf("a pad without a touchpad has %d", n)
	}
}

func TestPadsWorkTheFocusOfAnApp(t *testing.T) {
	var log []string
	n1, n2 := widgets.NewFocusNode(), widgets.NewFocusNode()
	defer n1.Dispose()
	defer n2.Dispose()
	tap := func(name string) func() { return func() { log = append(log, name) } }
	ts := gluttertest.New(t)
	ts.PumpWidget(material.NewMaterialApp(material.MaterialAppOpts{
		Home: material.NewMaterial(widgets.NewCenter(widgets.NewRow([]widgets.Widget{
			widgets.NewSizedBox(material.NewInkWell(nil, material.InkWellOpts{Autofocus: true, FocusNode: n1, OnTap: tap("first")}), widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(50.0)}),
			widgets.NewSizedBox(material.NewInkWell(nil, material.InkWellOpts{FocusNode: n2, OnTap: tap("second")}), widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(50.0)}),
		}))),
	}))
	ts.Pump()
	tp := ts.ConnectPad()
	if !n1.HasFocus() {
		t.Fatal("the autofocus node should have the focus")
	}
	tp.Tap(gamepad.South)
	ts.Pump()
	tp.Tap(gamepad.DPadRight)
	ts.Pump()
	if !n2.HasFocus() {
		t.Errorf("the D-pad did not move the focus to the second button (first has it: %v)", n1.HasFocus())
	}
	tp.Tap(gamepad.South)
	ts.Pump()
	if want := []string{"first", "second"}; !reflect.DeepEqual(log, want) {
		t.Errorf("taps = %v, want %v", log, want)
	}

	gamepad.ForBinding(ts.Binding).SetKeyEvents(false)
	if gamepad.ForBinding(ts.Binding).KeyEvents() {
		t.Error("KeyEvents() is on after SetKeyEvents(false)")
	}
	tp.Tap(gamepad.DPadLeft)
	tp.Tap(gamepad.South)
	ts.Pump()
	if !n2.HasFocus() || len(log) != 2 {
		t.Errorf("with key events off the pad still works the app: focus on second %v, taps %v", n2.HasFocus(), log)
	}
	gamepad.ForBinding(ts.Binding).SetKeyEvents(true)
	tp.Tap(gamepad.DPadLeft)
	ts.Pump()
	if !n1.HasFocus() {
		t.Error("with key events on again the D-pad should move the focus")
	}
}

func TestTheTriggersAreKeysFromHalfTheirTravel(t *testing.T) {
	var keys []string
	node := widgets.NewFocusNode()
	defer node.Dispose()
	ts := gluttertest.New(t)
	ts.PumpWidget(widgets.NewFocus(widgets.NewSizedBox(nil), widgets.FocusOpts{FocusNode: node, Autofocus: true,
		OnKeyEvent: func(_ *widgets.FocusNode, e services.KeyEvent) widgets.KeyEventResult {
			keys = append(keys, fmt.Sprintf("%T %s %v", e, e.LogicalKey().DebugName(), e.DeviceType()))
			return widgets.KeyEventResultHandled
		}}))
	ts.Pump()
	tp := ts.ConnectPad()
	tp.SetAxis(gamepad.LeftTrigger, 0.3)
	tp.SetAxis(gamepad.LeftTrigger, 0.6)
	tp.SetAxis(gamepad.LeftTrigger, 0.7)
	tp.SetAxis(gamepad.LeftTrigger, 0.1)
	tp.Tap(gamepad.DPadUp)
	want := []string{
		"*services.KeyDownEvent Game Button Left 2 " + fmt.Sprint(ui.KeyEventDeviceTypeGamepad),
		"*services.KeyUpEvent Game Button Left 2 " + fmt.Sprint(ui.KeyEventDeviceTypeGamepad),
		"*services.KeyDownEvent Arrow Up " + fmt.Sprint(ui.KeyEventDeviceTypeDirectionalPad),
		"*services.KeyUpEvent Arrow Up " + fmt.Sprint(ui.KeyEventDeviceTypeDirectionalPad),
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %q, want %q", keys, want)
	}
}

func TestAPadsSensorsAreSensorsOfTheSensorsPackage(t *testing.T) {
	ts, pads := app(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Motion", Sensors: true})
	plain := ts.ConnectPad(gluttertest.PadSpec{Name: "Plain"})
	p, q := pads.Pads()[0], pads.Pads()[1]
	if !p.HasSensors() || p.SensorRate() != 250 || q.HasSensors() {
		t.Fatalf("HasSensors = %v and %v, rate %v", p.HasSensors(), q.HasSensors(), p.SensorRate())
	}
	_ = plain

	if _, err := q.OpenSensor(sensors.Gyroscope); err == nil {
		t.Fatal("a pad with no sensors opened one")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagGamepadSensorsUnavailable {
		t.Fatalf("OpenSensor of a plain pad: %v", err)
	}

	gyro, err := p.OpenSensor(sensors.Gyroscope)
	if err != nil {
		t.Fatal(err)
	}
	if !pad.SensorsOn() || gyro.Kind() != sensors.Gyroscope || gyro.Rate() != 250 {
		t.Errorf("sensors on %v, kind %v, rate %v", pad.SensorsOn(), gyro.Kind(), gyro.Rate())
	}
	accel, err := p.OpenSensor(sensors.Accelerometer)
	if err != nil {
		t.Fatal(err)
	}
	var gyroGot, accelGot []sensors.Reading
	gyro.AddListener(func(r []sensors.Reading) { gyroGot = append(gyroGot, r...) })
	accel.AddListener(func(r []sensors.Reading) { accelGot = append(accelGot, r...) })

	pad.Sensor(sensors.Gyroscope, 0.5, -1, 2)
	pad.Sensor(sensors.Accelerometer, 0, 0, 9.8)
	pad.Sensor(sensors.Gyroscope, 1, 1, 1)
	ts.Pump(16 * time.Millisecond)
	if len(gyroGot) != 2 || len(accelGot) != 1 {
		t.Fatalf("the gyroscope got %d readings and the accelerometer %d, want 2 and 1", len(gyroGot), len(accelGot))
	}
	if gyroGot[0].Values != [3]float64{0.5, -1, 2} || accelGot[0].Values != [3]float64{0, 0, float64(float32(9.8))} {
		t.Errorf("values %v and %v", gyroGot[0].Values, accelGot[0].Values)
	}
	if r, ok := gyro.Latest(); !ok || r.Values != [3]float64{1, 1, 1} {
		t.Errorf("Latest = %v, %v", r, ok)
	}

	// The pad's sensors stay on while one is open, and go off with the last.
	if err := gyro.Close(); err != nil || !pad.SensorsOn() {
		t.Errorf("after the first Close: %v, sensors on %v", err, pad.SensorsOn())
	}
	if err := accel.Close(); err != nil || pad.SensorsOn() {
		t.Errorf("after the last Close: %v, sensors on %v", err, pad.SensorsOn())
	}
	pad.Sensor(sensors.Gyroscope, 3, 3, 3)
	ts.Pump(16 * time.Millisecond)
	if len(gyroGot) != 2 {
		t.Error("a closed sensor of a pad was called")
	}
}
