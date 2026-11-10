package ios

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestOrientationMaskOf(t *testing.T) {
	tests := []struct {
		name string
		in   []ui.DeviceOrientation
		want Orientation
		ok   bool
	}{
		{"none lets every orientation through", nil, OrientationAll, true},
		{"portrait", []ui.DeviceOrientation{ui.DeviceOrientationPortraitUp}, OrientationPortrait, true},
		{"upside down", []ui.DeviceOrientation{ui.DeviceOrientationPortraitDown}, OrientationPortraitUpsideDown, true},
		{"landscape left", []ui.DeviceOrientation{ui.DeviceOrientationLandscapeLeft}, OrientationLandscapeLeft, true},
		{"landscape right", []ui.DeviceOrientation{ui.DeviceOrientationLandscapeRight}, OrientationLandscapeRight, true},
		{"both landscapes", []ui.DeviceOrientation{ui.DeviceOrientationLandscapeLeft, ui.DeviceOrientationLandscapeRight}, OrientationLandscape, true},
		{"portrait and a repeat", []ui.DeviceOrientation{ui.DeviceOrientationPortraitUp, ui.DeviceOrientationPortraitUp}, OrientationPortrait, true},
		{"only an unknown one does nothing", []ui.DeviceOrientation{ui.DeviceOrientationUnspecified}, 0, false},
		{"an unknown one beside a known one", []ui.DeviceOrientation{ui.DeviceOrientationUnspecified, ui.DeviceOrientationPortraitUp}, OrientationPortrait, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := orientationMaskOf(tc.in)
			if got != tc.want || ok != tc.ok {
				t.Errorf("orientationMaskOf(%v) = %d, %v; want %d, %v", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
	// UIKit's own numbers: UIInterfaceOrientationMaskAll is 30.
	if OrientationAll != 30 {
		t.Errorf("OrientationAll = %d, want 30", OrientationAll)
	}
}

func TestStatusBarStyleOf(t *testing.T) {
	tests := []struct {
		in    ui.Brightness
		want  int64
		known bool
	}{
		{ui.BrightnessDark, statusBarLightContent, true},
		{ui.BrightnessLight, statusBarDarkContent, true},
		{ui.BrightnessUnspecified, statusBarDefault, false},
	}
	for _, tc := range tests {
		got, ok := statusBarStyleOf(tc.in)
		if got != tc.want || ok != tc.known {
			t.Errorf("statusBarStyleOf(%v) = %d, %v; want %d, %v", tc.in, got, ok, tc.want, tc.known)
		}
	}
	// UIKit's numbers: default 0, light content 1, dark content 3.
	if statusBarDefault != 0 || statusBarLightContent != 1 || statusBarDarkContent != 3 {
		t.Error("the status bar styles are not UIKit's numbers")
	}
}

func TestChromeFollowsTheMode(t *testing.T) {
	top, bottom := ui.SystemUIOverlayTop, ui.SystemUIOverlayBottom
	tests := []struct {
		name                    string
		mode                    ui.SystemUIMode
		overlays                []ui.SystemUIOverlay
		hideStatus, hideHomeInd bool
	}{
		{"edge to edge shows both", ui.SystemUIModeEdgeToEdge, nil, false, false},
		{"unspecified shows both", ui.SystemUIModeUnspecified, nil, false, false},
		{"lean back hides both", ui.SystemUIModeLeanBack, nil, true, true},
		{"immersive hides both", ui.SystemUIModeImmersive, nil, true, true},
		{"immersive sticky hides both", ui.SystemUIModeImmersiveSticky, nil, true, true},
		{"manual with the top", ui.SystemUIModeManual, []ui.SystemUIOverlay{top}, false, true},
		{"manual with the bottom", ui.SystemUIModeManual, []ui.SystemUIOverlay{bottom}, true, false},
		{"manual with both", ui.SystemUIModeManual, []ui.SystemUIOverlay{top, bottom}, false, false},
		{"manual with none", ui.SystemUIModeManual, nil, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A mode that hid the bars earlier does not stay: each call says all of it.
			got := chrome{hideStatusBar: true, hideHomeIndicator: true}.withMode(tc.mode, tc.overlays)
			if got.hideStatusBar != tc.hideStatus || got.hideHomeIndicator != tc.hideHomeInd {
				t.Errorf("status hidden %v, home indicator hidden %v; want %v, %v", got.hideStatusBar, got.hideHomeIndicator, tc.hideStatus, tc.hideHomeInd)
			}
		})
	}
}

func TestChromeStyleAndOrientations(t *testing.T) {
	c := chrome{}
	if c.styleSet || c.supported() != OrientationAll {
		t.Fatalf("a new chrome says %+v: no style, every orientation", c)
	}
	c = c.withStyle(ui.SystemUIOverlayStyle{StatusBarBrightness: ui.BrightnessDark})
	if !c.styleSet || c.style != statusBarLightContent {
		t.Errorf("a dark status bar gave style %d (set %v)", c.style, c.styleSet)
	}
	// A style without a brightness keeps what the app asked before.
	c = c.withStyle(ui.SystemUIOverlayStyle{})
	if !c.styleSet || c.style != statusBarLightContent {
		t.Errorf("an empty style changed the status bar to %d (set %v)", c.style, c.styleSet)
	}
	c, ok := c.withOrientations([]ui.DeviceOrientation{ui.DeviceOrientationPortraitUp})
	if !ok || c.supported() != OrientationPortrait {
		t.Errorf("portrait: ok %v, supported %d", ok, c.supported())
	}
	// A call with no known orientation does nothing.
	c2, ok := c.withOrientations([]ui.DeviceOrientation{ui.DeviceOrientationUnspecified})
	if ok || c2 != c {
		t.Errorf("an unknown orientation changed the chrome: ok %v, %+v", ok, c2)
	}
}

func TestHapticOf(t *testing.T) {
	tests := []struct {
		in   ui.HapticFeedbackKind
		want haptic
	}{
		{ui.HapticFeedbackVibrate, haptic{kind: hapticVibrate}},
		{ui.HapticFeedbackLightImpact, haptic{hapticImpact, impactLight}},
		{ui.HapticFeedbackMediumImpact, haptic{hapticImpact, impactMedium}},
		{ui.HapticFeedbackHeavyImpact, haptic{hapticImpact, impactHeavy}},
		{ui.HapticFeedbackSelectionClick, haptic{kind: hapticSelection}},
		{ui.HapticFeedbackUnspecified, haptic{}},
	}
	for _, tc := range tests {
		if got := hapticOf(tc.in); got != tc.want {
			t.Errorf("hapticOf(%v) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
	// UIKit's numbers: light 0, medium 1, heavy 2.
	if impactLight != 0 || impactMedium != 1 || impactHeavy != 2 {
		t.Error("the impact styles are not UIKit's numbers")
	}
}

func TestSoundOf(t *testing.T) {
	tests := []struct {
		in   ui.SystemSoundKind
		want uint32
		ok   bool
	}{
		{ui.SystemSoundClick, 1306, true},
		{ui.SystemSoundTick, 1157, true},
		{ui.SystemSoundAlert, 0, false},
		{ui.SystemSoundUnspecified, 0, false},
	}
	for _, tc := range tests {
		got, ok := soundOf(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("soundOf(%v) = %d, %v; want %d, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
