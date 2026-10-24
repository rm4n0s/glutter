package android

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// TestRequestedOrientation has every set of orientations that PlatformChannel.decodeOrientations
// tells apart, and the sets Android cannot describe (the first orientation stands for them).
func TestRequestedOrientation(t *testing.T) {
	up, left, down, right := ui.DeviceOrientationPortraitUp, ui.DeviceOrientationLandscapeLeft,
		ui.DeviceOrientationPortraitDown, ui.DeviceOrientationLandscapeRight
	tests := []struct {
		name string
		in   []ui.DeviceOrientation
		want int
	}{
		{"none", nil, screenOrientationUnspecified},
		{"portrait up", []ui.DeviceOrientation{up}, screenOrientationPortrait},
		{"landscape left", []ui.DeviceOrientation{left}, screenOrientationLandscape},
		{"portrait down", []ui.DeviceOrientation{down}, screenOrientationReversePortrait},
		{"landscape right", []ui.DeviceOrientation{right}, screenOrientationReverseLandscape},
		{"both portraits", []ui.DeviceOrientation{up, down}, screenOrientationUserPortrait},
		{"both landscapes", []ui.DeviceOrientation{left, right}, screenOrientationUserLandscape},
		{"three", []ui.DeviceOrientation{up, left, right}, screenOrientationUser},
		{"all four", []ui.DeviceOrientation{up, left, down, right}, screenOrientationFullUser},
		{"the same twice", []ui.DeviceOrientation{left, left}, screenOrientationLandscape},
		{"up and left: the first stands", []ui.DeviceOrientation{up, left}, screenOrientationPortrait},
		{"left and up: the first stands", []ui.DeviceOrientation{left, up}, screenOrientationLandscape},
		{"right and down: the first stands", []ui.DeviceOrientation{right, down}, screenOrientationReverseLandscape},
		{"down, left, right: down stands", []ui.DeviceOrientation{down, left, right}, screenOrientationReversePortrait},
		{"unspecified is nothing", []ui.DeviceOrientation{ui.DeviceOrientationUnspecified}, screenOrientationUnspecified},
	}
	for _, tt := range tests {
		if got := requestedOrientation(tt.in); got != tt.want {
			t.Errorf("%s: requestedOrientation = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestEncodeOverlayStyle(t *testing.T) {
	black, yes, no := ui.NewColor(0xFF000000), true, false
	tests := []struct {
		name  string
		style ui.SystemUIOverlayStyle
		want  string
	}{
		{"nothing said", ui.SystemUIOverlayStyle{}, ",,,,,,"},
		{"light icons on a black bar", ui.SystemUIOverlayStyle{
			StatusBarColor: &black, StatusBarIconBrightness: ui.BrightnessLight, SystemStatusBarContrastEnforced: &no,
		}, "4278190080,1,0,,,,"},
		{"dark navigation bar icons", ui.SystemUIOverlayStyle{
			SystemNavigationBarColor: &black, SystemNavigationBarDividerColor: &black,
			SystemNavigationBarIconBrightness: ui.BrightnessDark, SystemNavigationBarContrastEnforced: &yes,
		}, ",,,4278190080,4278190080,2,1"},
	}
	for _, tt := range tests {
		if got := encodeOverlayStyle(tt.style); got != tt.want {
			t.Errorf("%s: encodeOverlayStyle = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestTheWindowSendsCommands checks what the window asks of the activity for each call of the
// framework.
func TestTheWindowSendsCommands(t *testing.T) {
	h := newHarness(t)
	h.run()
	chrome, ok := h.win.(ui.SystemChromeHost)
	if !ok {
		t.Fatal("the window is not a ui.SystemChromeHost")
	}
	black := ui.NewColor(0xFF000000)
	h.sync(func() {
		chrome.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationLandscapeLeft})
		chrome.SetEnabledSystemUIMode(ui.SystemUIModeManual, []ui.SystemUIOverlay{ui.SystemUIOverlayBottom})
		chrome.SetSystemUIOverlayStyle(ui.SystemUIOverlayStyle{StatusBarColor: &black})
		chrome.PerformHapticFeedback(ui.HapticFeedbackHeavyImpact)
		chrome.PlaySystemSound(ui.SystemSoundClick)
		h.win.SetState(platform.WindowStateFullscreen)
		h.win.SetState(platform.WindowStateNormal)
		h.win.SetState(platform.WindowStateMaximized)
		if err := h.win.KeepScreenOn(true); err != nil {
			t.Errorf("KeepScreenOn: %v", err)
		}
		_ = h.win.KeepScreenOn(false)
	})
	want := []command{
		{Op: opOrientation, A: screenOrientationLandscape},
		{Op: opUIMode, A: int64(ui.SystemUIModeManual), B: overlayBottom},
		{Op: opOverlayStyle, S: "4278190080,,,,,,"},
		{Op: opHaptic, A: int64(ui.HapticFeedbackHeavyImpact)},
		{Op: opSound, A: int64(ui.SystemSoundClick)},
		{Op: opUIMode, A: int64(ui.SystemUIModeImmersiveSticky)},
		{Op: opUIMode, A: int64(ui.SystemUIModeEdgeToEdge)},
		{Op: opKeepScreenOn, A: 1},
		{Op: opKeepScreenOn, A: 0},
	}
	if got := h.host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Errorf("the commands were\n%+v\nwant\n%+v", got, want)
	}
}

// TestSystemUIChangesReachTheCallback checks that the bars appearing and disappearing are told to
// the function the framework set, on the UI thread, and not after it was removed.
func TestSystemUIChangesReachTheCallback(t *testing.T) {
	h := newHarness(t)
	h.run()
	chrome := h.win.(ui.SystemChromeHost)
	var seen []bool
	h.sync(func() { chrome.SetSystemUIChangeCallback(func(v bool) { seen = append(seen, v) }) })
	h.b.systemUIChanged(true)
	h.b.systemUIChanged(false)
	h.sync(func() {})
	if !reflect.DeepEqual(seen, []bool{true, false}) {
		t.Errorf("the callback heard %v", seen)
	}
	h.sync(func() { chrome.SetSystemUIChangeCallback(nil) })
	h.b.systemUIChanged(true)
	h.sync(func() {})
	if len(seen) != 2 {
		t.Errorf("a removed callback was called: %v", seen)
	}
	got := h.host.sentCommands()
	if len(got) != 2 || got[0] != (command{Op: opUIChangeListener, A: 1}) || got[1] != (command{Op: opUIChangeListener}) {
		t.Errorf("the listener commands were %+v", got)
	}
}
