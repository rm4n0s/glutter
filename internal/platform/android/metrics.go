package android

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// defaultDensity is the device pixel ratio before the display said (the baseline density of
// Android, 160 dpi).
const defaultDensity = 1.0

// defaultRefreshRate is the refresh rate before the display said.
const defaultRefreshRate = 60.0

// pixelRatio returns the device pixel ratio of the surface, defaultDensity while it is unknown or
// not a number above zero.
func (s *surfaceState) pixelRatio() float64 {
	if s.density > 0 {
		return s.density
	}
	return defaultDensity
}

// viewMetrics are the metrics of the view: the physical size of the surface and the device
// pixel ratio (SPEC §24.7). The insets come in step 25.7.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterView.java FlutterView.sendViewportMetricsToFlutter
func (s *surfaceState) viewMetrics() platform.ViewMetrics {
	return platform.ViewMetrics{
		PhysicalSize:     ui.Size{Width: float64(s.width), Height: float64(s.height)},
		DevicePixelRatio: s.pixelRatio(),
	}
}

// display is the one display, in logical pixels as the other platforms give it, with the
// refresh rate Android reports.
//
// Flutter: shell/platform/android/android_display.cc AndroidDisplay
func (s *surfaceState) display() platform.Display {
	dpr := s.pixelRatio()
	refresh := s.refresh
	if refresh <= 0 {
		refresh = defaultRefreshRate
	}
	w, h := s.displayWidth, s.displayHeight
	if w <= 0 || h <= 0 {
		w, h = s.width, s.height
	}
	return platform.Display{
		ID:               0,
		Name:             "display",
		Bounds:           ui.NewRectFromLTWH(0, 0, float64(w)/dpr, float64(h)/dpr),
		DevicePixelRatio: dpr,
		RefreshRate:      refresh,
		Primary:          true,
	}
}
