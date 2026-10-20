package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func frameTiming(n int, build, raster time.Duration) ui.FrameTiming {
	t0 := time.Duration(n) * 20 * time.Millisecond
	return ui.NewFrameTiming(t0, t0, t0+build, t0+build, t0+build+raster, time.Duration(n), n)
}

func overlayApp(show bool) widgets.Widget {
	return NewMaterialApp(MaterialAppOpts{
		DebugShowCheckedModeBanner: foundation.Some(false),
		ShowPerformanceOverlay:     show,
		Home:                       NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(widgets.NewText("body"))}),
	})
}

func TestShowPerformanceOverlayAddsTheOverlayAndListensForTimings(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 400}})
	ts.PumpWidget(overlayApp(false))
	overlay := gluttertest.ByType[*widgets.PerformanceOverlay]()
	if ts.Exists(overlay) {
		t.Fatal("the overlay is there without ShowPerformanceOverlay")
	}
	dispatcher := ts.Binding.PlatformDispatcher()
	if dispatcher.NeedsReportTimings() {
		t.Error("the engine would measure the frames with no overlay")
	}

	ts.PumpWidget(overlayApp(true))
	if !ts.Exists(overlay) {
		t.Fatal("ShowPerformanceOverlay did not add the overlay")
	}
	if got := ts.Size(overlay); got.Width != 400 || got.Height != 160 {
		t.Errorf("overlay size = %v, want 400 x 160 at the top of the app", got)
	}
	if got := ts.TopLeft(overlay); got != (ui.Offset{}) {
		t.Errorf("overlay position = %v, want the top left", got)
	}
	if !dispatcher.NeedsReportTimings() {
		t.Error("the engine would not measure the frames with an overlay")
	}

	// The frames the engine reports show in the next frame that something asks for.
	dispatcher.ReportTimings([]ui.FrameTiming{
		frameTiming(1, 2*time.Millisecond, 3*time.Millisecond),
		frameTiming(2, 4*time.Millisecond, 20*time.Millisecond),
		frameTiming(3, 3*time.Millisecond, 5*time.Millisecond),
	})
	if ts.Binding.Scheduler().HasScheduledFrame() {
		t.Error("the timings asked for a frame")
	}
	ts.Binding.Scheduler().ScheduleFrame()
	ts.Pump()
	ts.MatchesGolden("performance-overlay")

	// Without the overlay the engine stops measuring.
	ts.PumpWidget(overlayApp(false))
	if dispatcher.NeedsReportTimings() {
		t.Error("the engine still measures the frames after the overlay was removed")
	}
}
