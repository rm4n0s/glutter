package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestTitleBarPadding(t *testing.T) {
	frame := boxRect{W: 480, H: 668}
	layout := boxRect{W: 480, H: 640}
	for _, scale := range []float64{1, 1.5, 2, 3} {
		if got, want := titleBarPadding(frame, layout, scale), 28*scale; got != want {
			t.Errorf("scale %v: padding %v, want %v", scale, got, want)
		}
	}
	if got := titleBarPadding(boxRect{H: 100}, boxRect{H: 120}, 2); got != 0 {
		t.Errorf("a layout rectangle taller than the frame gave a padding of %v", got)
	}
}

func TestTrafficLightArea(t *testing.T) {
	// The three buttons of a window with a 28-point title bar, in the coordinates of the bar (the origin at
	// its bottom left): 14 points wide, 20 apart, 7 points from the bottom.
	buttons := []boxRect{{X: 7, Y: 7, W: 14, H: 14}, {X: 27, Y: 7, W: 14, H: 14}, {X: 47, Y: 7, W: 14, H: 14}}
	tests := []struct {
		scale float64
		want  ui.Rect
	}{
		{1, ui.Rect{Left: 7, Top: 7, Right: 61, Bottom: 21}},
		{2, ui.Rect{Left: 14, Top: 14, Right: 122, Bottom: 42}},
		{3, ui.Rect{Left: 21, Top: 21, Right: 183, Bottom: 63}},
	}
	for _, tc := range tests {
		if got := trafficLightArea(buttons, 28, tc.scale); got != tc.want {
			t.Errorf("scale %v: %+v, want %+v", tc.scale, got, tc.want)
		}
	}
	// A dialog has no zoom button: the missing one is left out.
	if got, want := trafficLightArea([]boxRect{buttons[0], buttons[1], {}}, 28, 1), (ui.Rect{Left: 7, Top: 7, Right: 41, Bottom: 21}); got != want {
		t.Errorf("without the zoom button: %+v, want %+v", got, want)
	}
	if got := trafficLightArea(nil, 28, 2); got != (ui.Rect{}) {
		t.Errorf("no buttons: %+v, want the zero rectangle", got)
	}
}

func TestTitleDoubleClick(t *testing.T) {
	tests := []struct {
		setting string
		asked   ui.WindowActionKind
		want    ui.WindowActionKind
		ok      bool
	}{
		{"Maximize", ui.WindowActionMaximize, ui.WindowActionMaximize, true},
		{"Maximize", ui.WindowActionRestore, ui.WindowActionRestore, true},
		{"", ui.WindowActionMaximize, ui.WindowActionMaximize, true},
		{"something new", ui.WindowActionRestore, ui.WindowActionRestore, true},
		{"Minimize", ui.WindowActionMaximize, ui.WindowActionMinimize, true},
		{"Minimize", ui.WindowActionRestore, ui.WindowActionMinimize, true},
		{"None", ui.WindowActionMaximize, 0, false},
	}
	for _, tc := range tests {
		got, ok := titleDoubleClick(tc.setting, tc.asked)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("setting %q, asked %v: %v, %v; want %v, %v", tc.setting, tc.asked, got, ok, tc.want, tc.ok)
		}
	}
}

func TestTrafficLightY(t *testing.T) {
	tests := []struct {
		name               string
		bar, button, wantY float64
	}{
		{"AppKit's own bar", 28, 14, 7},
		{"a higher bar of the app", 40, 14, 13},
		{"an odd height", 35, 16, 9.5},
		{"a bar lower than the button", 10, 14, 0},
	}
	for _, tc := range tests {
		if got := trafficLightY(tc.bar, tc.button); got != tc.wantY {
			t.Errorf("%s: trafficLightY(%v, %v) = %v, want %v", tc.name, tc.bar, tc.button, got, tc.wantY)
		}
	}
}
