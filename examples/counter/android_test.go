package main

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

// androidDPR and androidSize are the screen of the emulator's virtual devices (SPEC §24.13): 1080 x
// 2400 pixels at 420 dpi, which is 2.625 pixels per logical pixel.
const androidDPR = 2.625

// TestAndroidGolden draws the counter as it is on an Android phone, with the framework's Android
// branches (it is the default platform of the tests), at the size and density of the emulator's
// virtual device, with the insets that the system gives it. The device test (android_device_test.go) compares the first frame that the app
// draws on the emulator with this image.
func TestAndroidGolden(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{
		Size:             m.Size{Width: 1080 / androidDPR, Height: 2400 / androidDPR},
		DevicePixelRatio: androidDPR,
		// The system bars and the cutout of the emulator's API 37 image: the status bar is 128 pixels tall
		// (a display cutout), the navigation bar's gesture area 63 (SPEC §24.7).
		ViewPadding: ui.ViewPadding{Top: 128, Bottom: 63},
	})
	ts.PumpWidget(NewMyApp())
	if img := ts.Screenshot(); img.Rect.Dx() != 1080 || img.Rect.Dy() != 2400 {
		t.Fatalf("the screenshot is %v, want 1080x2400", img.Rect)
	}
	ts.MatchesGolden("counter-android")
}
