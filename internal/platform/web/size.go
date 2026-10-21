package web

import (
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// SanitizeDPR returns dpr when it is a usable device pixel ratio (finite and above
// zero), and 1 otherwise: a browser that reports 0 or nothing is treated as a screen
// without scaling.
//
// Flutter: lib/web_ui/lib/src/engine/display.dart EngineFlutterDisplay.devicePixelRatio
func SanitizeDPR(dpr float64) float64 {
	if math.IsNaN(dpr) || math.IsInf(dpr, 0) || dpr <= 0 {
		return 1
	}
	return dpr
}

// ResolutionQuery is the media query that stops matching when the device pixel ratio
// changes from dpr (browser zoom, or a move to a monitor with another scale). The
// platform listens to it and makes a new one for the new ratio each time it fires.
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/display_dpr_stream.dart DisplayDprStream
func ResolutionQuery(dpr float64) string {
	return "(resolution: " + strconv.FormatFloat(SanitizeDPR(dpr), 'f', -1, 64) + "dppx)"
}

// Geometry is what the browser says about the size of the view: the size of the
// canvas in CSS pixels, its size in device pixels when the browser gives it exactly
// (ResizeObserver's devicePixelContentBoxSize), and the device pixel ratio.
//
// Flutter: no counterpart
type Geometry struct {
	// CSSWidth and CSSHeight are the size of the canvas element in CSS pixels.
	CSSWidth, CSSHeight float64
	// DeviceWidth and DeviceHeight are the size in device pixels from
	// devicePixelContentBoxSize; zero when the browser does not give it.
	DeviceWidth, DeviceHeight int
	// DPR is window.devicePixelRatio.
	DPR float64
}

// PhysicalSize is the size of the view in device pixels: the exact one from
// devicePixelContentBoxSize when the browser gave it, otherwise round(CSS size × device
// pixel ratio) (SPEC §23.2).
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/dimensions_provider/full_page_dimensions_provider.dart FullPageDimensionsProvider.computePhysicalSize
func (g Geometry) PhysicalSize() (width, height int) {
	if g.DeviceWidth > 0 && g.DeviceHeight > 0 {
		return g.DeviceWidth, g.DeviceHeight
	}
	dpr := SanitizeDPR(g.DPR)
	return roundSize(g.CSSWidth * dpr), roundSize(g.CSSHeight * dpr)
}

// roundSize rounds a size to whole pixels; a size that is not a finite number of
// pixels is 0.
func roundSize(v float64) int {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return int(math.Round(v))
}

// ViewMetrics are the metrics of the view for the engine: the physical size and the
// device pixel ratio. The browser has no system areas of its own, so the paddings and
// insets are zero.
//
// Flutter: lib/web_ui/lib/src/engine/window.dart EngineFlutterView.physicalSize
func (g Geometry) ViewMetrics() platform.ViewMetrics {
	w, h := g.PhysicalSize()
	return platform.ViewMetrics{
		PhysicalSize:     ui.Size{Width: float64(w), Height: float64(h)},
		DevicePixelRatio: SanitizeDPR(g.DPR),
	}
}

// FrameTime turns the timestamp of a requestAnimationFrame callback (a
// DOMHighResTimeStamp, in milliseconds since the page's time origin) into the frame
// time of Handler.OnFrame. A timestamp that is not a finite, non-negative number is 0.
//
// Flutter: no counterpart
func FrameTime(ms float64) time.Duration {
	if math.IsNaN(ms) || math.IsInf(ms, 0) || ms < 0 {
		return 0
	}
	return time.Duration(ms * float64(time.Millisecond))
}

// DisplayOf describes the one display of a page from the screen: its size in CSS
// pixels, the device pixel ratio and a refresh rate of 60 (SPEC §23.2).
//
// Flutter: lib/web_ui/lib/src/engine/display.dart EngineFlutterDisplay
func DisplayOf(screenWidth, screenHeight, dpr float64) platform.Display {
	return platform.Display{
		ID:               0,
		Name:             "screen",
		Bounds:           ui.NewRectFromLTWH(0, 0, screenWidth, screenHeight),
		DevicePixelRatio: SanitizeDPR(dpr),
		RefreshRate:      60,
		Primary:          true,
	}
}
