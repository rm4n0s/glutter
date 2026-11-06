package main

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/gluttertest"
)

func TestGamepadPanelShowsThePadsInDebugBuilds(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("GLUTTER_DEV_GAMEPADS is read in debug builds only")
	}
	t.Setenv("GLUTTER_DEV_GAMEPADS", "1")
	ts := newTester(t)
	if !ts.Exists(gluttertest.ByText("Gamepads (debug)")) || !ts.Exists(gluttertest.ByText("No pad. Connect one and press a button.")) {
		t.Fatal("the panel is not shown")
	}
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Test Pad"})
	ts.Pump()
	if !ts.Exists(gluttertest.ByTextContaining("Test Pad [Generic")) || !ts.Exists(gluttertest.ByText("Test Pad connected (Generic)")) {
		t.Error("the pad is not listed")
	}
	pad.Press(gamepad.South)
	pad.Press(gamepad.DPadLeft)
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("down: a dpleft")) || !ts.Exists(gluttertest.ByText("Test Pad: a down")) {
		t.Error("the buttons are not shown")
	}
	pad.SetAxis(gamepad.LeftX, 1)
	ts.Pump()
	if !ts.Exists(gluttertest.ByTextContaining("L +1.00 +0.00")) {
		t.Error("the axes are not shown")
	}
	pad.Disconnect()
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("Test Pad removed")) {
		t.Error("the removal is not shown")
	}
}

func TestGamepadPanelIsNotThereByDefault(t *testing.T) {
	ts := newTester(t)
	if ts.Exists(gluttertest.ByText("Gamepads (debug)")) {
		t.Error("the panel is shown without GLUTTER_DEV_GAMEPADS")
	}
}
