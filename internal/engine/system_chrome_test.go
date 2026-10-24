package engine

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// chromeWindow is a window that can control the system's interface and records what it is asked.
type chromeWindow struct {
	platform.Window
	calls []any
}

func (w *chromeWindow) SetPreferredOrientations(o []ui.DeviceOrientation) {
	w.calls = append(w.calls, "orientations", o)
}
func (w *chromeWindow) SetEnabledSystemUIMode(m ui.SystemUIMode, o []ui.SystemUIOverlay) {
	w.calls = append(w.calls, "mode", m, o)
}
func (w *chromeWindow) SetSystemUIOverlayStyle(s ui.SystemUIOverlayStyle) {
	w.calls = append(w.calls, "style", s.StatusBarIconBrightness)
}
func (w *chromeWindow) SetSystemUIChangeCallback(fn func(bool)) {
	w.calls = append(w.calls, "callback", fn != nil)
}
func (w *chromeWindow) PerformHapticFeedback(k ui.HapticFeedbackKind) {
	w.calls = append(w.calls, "haptic", k)
}
func (w *chromeWindow) PlaySystemSound(k ui.SystemSoundKind) { w.calls = append(w.calls, "sound", k) }

// TestSystemChromeGoesToTheWindowThatCanDoIt checks the engine forwards the dispatcher's calls to the
// window of the view, and does nothing for a window that cannot (the desktop's).
func TestSystemChromeGoesToTheWindowThatCanDoIt(t *testing.T) {
	e := newWindowsEngine(t)
	// Nothing happens, and nothing panics, for a window without them.
	e.d.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationPortraitUp})
	e.d.PerformHapticFeedback(ui.HapticFeedbackVibrate)

	w := &chromeWindow{Window: e.h.windowOf(ui.ImplicitViewID).Window}
	e.h.windowOf(ui.ImplicitViewID).Window = w
	e.d.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationLandscapeLeft})
	e.d.SetEnabledSystemUIMode(ui.SystemUIModeManual, []ui.SystemUIOverlay{ui.SystemUIOverlayTop})
	e.d.SetSystemUIOverlayStyle(ui.SystemUIOverlayStyle{StatusBarIconBrightness: ui.BrightnessDark})
	e.d.SetSystemUIChangeCallback(func(bool) {})
	e.d.PerformHapticFeedback(ui.HapticFeedbackSelectionClick)
	e.d.PlaySystemSound(ui.SystemSoundClick)
	want := []string{"orientations", "mode", "style", "callback", "haptic", "sound"}
	var got []string
	for _, c := range w.calls {
		if s, ok := c.(string); ok {
			got = append(got, s)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the window was asked %v, want %v", got, want)
	}
}
