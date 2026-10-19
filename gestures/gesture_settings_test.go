package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestDeviceGestureSettings(t *testing.T) {
	for _, tc := range []struct {
		name         string
		physicalSlop float64
		dpr          float64
		want         foundation.Opt[float64]
		wantPan      foundation.Opt[float64]
		wantString   string
	}{
		{"no settings from the platform", 0, 2, foundation.None[float64](), foundation.None[float64](), "DeviceGestureSettings(touchSlop: null)"},
		{"the touch slop is in logical pixels", 8, 2, foundation.Some(4.0), foundation.Some(8.0), "DeviceGestureSettings(touchSlop: 4.0)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := ui.NewPlatformDispatcher(nil)
			view := d.AddView(ui.ImplicitViewID, ui.ViewConfiguration{
				PhysicalSize: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: tc.dpr,
				GestureSettings: ui.GestureSettings{PhysicalTouchSlop: tc.physicalSlop},
			})
			s := NewDeviceGestureSettingsFromView(view)
			if s.TouchSlop != tc.want || s.PanSlop() != tc.wantPan || s.String() != tc.wantString {
				t.Errorf("settings = %v, pan slop %v", s, s.PanSlop())
			}
		})
	}
}

func TestComputeSlops(t *testing.T) {
	settings := foundation.Some(DeviceGestureSettings{TouchSlop: foundation.Some(5.0)})
	none := foundation.None[DeviceGestureSettings]()
	for _, tc := range []struct {
		name            string
		kind            ui.PointerDeviceKind
		settings        foundation.Opt[DeviceGestureSettings]
		hit, pan, scale float64
	}{
		{"mouse ignores the settings", ui.PointerDeviceKindMouse, settings, PrecisePointerHitSlop, PrecisePointerPanSlop, PrecisePointerScaleSlop},
		{"touch without settings", ui.PointerDeviceKindTouch, none, TouchSlop, PanSlop, ScaleSlop},
		{"touch with settings", ui.PointerDeviceKindTouch, settings, 5, 10, ScaleSlop},
		{"touch with empty settings", ui.PointerDeviceKindTouch, foundation.Some(DeviceGestureSettings{}), TouchSlop, PanSlop, ScaleSlop},
		{"stylus is like touch", ui.PointerDeviceKindStylus, settings, 5, 10, ScaleSlop},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeHitSlop(tc.kind, tc.settings); got != tc.hit {
				t.Errorf("hit slop = %v, want %v", got, tc.hit)
			}
			if got := ComputePanSlop(tc.kind, tc.settings); got != tc.pan {
				t.Errorf("pan slop = %v, want %v", got, tc.pan)
			}
			if got := ComputeScaleSlop(tc.kind); got != tc.scale {
				t.Errorf("scale slop = %v, want %v", got, tc.scale)
			}
		})
	}
}
