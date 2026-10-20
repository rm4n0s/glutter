package rendering_test

import (
	"image/color"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// timing makes the timing of a frame that took build to build and raster to draw.
func timing(number int, build, raster time.Duration) ui.FrameTiming {
	t0 := time.Duration(number) * 20 * time.Millisecond
	return ui.NewFrameTiming(t0, t0, t0+build, t0+build, t0+build+raster, time.Duration(number), number)
}

func TestPerformanceOverlaySize(t *testing.T) {
	tests := []struct {
		name       string
		mask       int
		wantHeight float64
	}{
		{"nothing", 0, 0},
		{"the raster graph", rendering.PerformanceOverlayOptionVisualizeRasterizerStatistics.Mask(), 80},
		{"the raster text", rendering.PerformanceOverlayOptionDisplayRasterizerStatistics.Mask(), 80},
		{"the build graph", rendering.PerformanceOverlayOptionVisualizeEngineStatistics.Mask(), 80},
		{"all", rendering.PerformanceOverlayAllMask(), 160},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			r := rendering.NewRenderPerformanceOverlay(rendering.RenderPerformanceOverlayOpts{OptionsMask: tt.mask})
			// A parent that gives loose constraints: the overlay is as wide as they allow
			// and as high as its graphs.
			box := rendering.NewRenderPositionedBox(r)
			h.SetRoot(box)
			h.Pump()
			if got := r.Size(); got.Height != tt.wantHeight || got.Width != 800 {
				t.Errorf("size = %v, want 800 x %v", got, tt.wantHeight)
			}
			if got := r.GetMinIntrinsicHeight(100); got != tt.wantHeight {
				t.Errorf("intrinsic height = %v, want %v", got, tt.wantHeight)
			}
			if r.GetMaxIntrinsicWidth(100) != 0 {
				t.Errorf("intrinsic width = %v, want 0", r.GetMaxIntrinsicWidth(100))
			}
		})
	}
}

func TestPerformanceOverlayOptionMasks(t *testing.T) {
	tests := []struct {
		option rendering.PerformanceOverlayOption
		index  int
		mask   int
	}{
		{rendering.PerformanceOverlayOptionDisplayRasterizerStatistics, 0, 1},
		{rendering.PerformanceOverlayOptionVisualizeRasterizerStatistics, 1, 2},
		{rendering.PerformanceOverlayOptionDisplayEngineStatistics, 2, 4},
		{rendering.PerformanceOverlayOptionVisualizeEngineStatistics, 3, 8},
		{rendering.PerformanceOverlayOptionUnspecified, -1, 0},
	}
	for _, tt := range tests {
		if tt.option.Index() != tt.index || tt.option.Mask() != tt.mask {
			t.Errorf("option %d: index %d mask %d, want %d and %d", tt.option, tt.option.Index(), tt.option.Mask(), tt.index, tt.mask)
		}
	}
	if rendering.PerformanceOverlayAllMask() != 15 {
		t.Errorf("all = %d, want 15", rendering.PerformanceOverlayAllMask())
	}
}

func TestPerformanceOverlayListensWhileAttachedAndDoesNotAskForFrames(t *testing.T) {
	h := newHarness(t)
	r := rendering.NewRenderPerformanceOverlay(rendering.RenderPerformanceOverlayOpts{
		OptionsMask: rendering.PerformanceOverlayAllMask(), Scheduler: h.Scheduler,
	})
	if h.Dispatcher.NeedsReportTimings() {
		t.Fatal("something listens before the overlay is attached")
	}
	h.SetRoot(rendering.NewRenderPositionedBox(r))
	h.Pump()
	if !h.Dispatcher.NeedsReportTimings() {
		t.Fatal("the overlay does not listen once attached")
	}
	// A timing arrives between frames: it is kept and the overlay is marked, but no
	// frame is asked for.
	for h.Scheduler.HasScheduledFrame() {
		h.Pump()
	}
	h.Dispatcher.ReportTimings([]ui.FrameTiming{timing(1, 3*time.Millisecond, 2*time.Millisecond)})
	if r.History().Len() != 1 {
		t.Errorf("history has %d frames, want 1", r.History().Len())
	}
	if h.Scheduler.HasScheduledFrame() {
		t.Error("a timing asked for a frame: the overlay would keep the app drawing")
	}
	if foundation.DebugMode && !r.DebugNeedsPaint() {
		t.Error("the overlay is not marked for paint")
	}
	// The next frame that something else asks for paints it.
	h.Scheduler.ScheduleFrame()
	h.Pump()
	if r.DebugNeedsPaint() {
		t.Error("the frame did not paint the overlay")
	}
	if len(h.Scenes()) < 2 {
		t.Error("the frame composited no scene")
	}
	// Detached, it stops listening.
	h.SetRoot(rendering.NewRenderPositionedBox(nil))
	h.Pump()
	if h.Dispatcher.NeedsReportTimings() {
		t.Error("the overlay listens after it was removed")
	}
}

func TestPerformanceOverlayPaintsTheGraphs(t *testing.T) {
	h := renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	history := rendering.NewFrameTimingHistory(0)
	// Five frames: the raster time goes over the budget of 16.67 ms in the fourth, the
	// build time never.
	for i, raster := range []time.Duration{2, 4, 8, 30, 5} {
		history.Add(timing(i+1, time.Duration(i+1)*time.Millisecond, raster*time.Millisecond))
	}
	r := rendering.NewRenderPerformanceOverlay(rendering.RenderPerformanceOverlayOpts{
		OptionsMask: rendering.PerformanceOverlayOptionVisualizeRasterizerStatistics.Mask() | rendering.PerformanceOverlayOptionVisualizeEngineStatistics.Mask(),
		History:     history,
	})
	h.SetRoot(rendering.NewRenderPositionedBox(r, rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentTopLeft}))
	h.Pump()
	img, err := h.Image()
	if err != nil {
		t.Fatal(err)
	}
	// The scene is drawn over transparent pixels: the background of a graph is 80%
	// black, and nothing is drawn outside the two graphs.
	green, red := color.RGBA{0x4C, 0xAF, 0x50, 0xFF}, color.RGBA{0xF4, 0x43, 0x36, 0xFF}
	dark, none := color.RGBA{0, 0, 0, 0xCC}, color.RGBA{}
	// The raster graph is the top 80 rows and the newest frame is at the right edge:
	// five bars of 2 pixels end at x = 200. The fourth frame (30 ms, over the budget)
	// is the second from the right; the bars grow from the bottom of the graph.
	tests := []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{"the newest raster bar (5 ms) is green", 199, 78, green},
		{"the slow raster bar (30 ms) is red", 197, 78, red},
		{"the slow raster bar is 72 pixels high: its top row", 197, 8, red},
		{"the slow raster bar is 72 pixels high: above it", 197, 7, dark},
		{"the 8 ms bar is green", 195, 78, green},
		{"above the 8 ms bar there is the background", 195, 20, dark},
		{"left of the first bar there is the background", 100, 70, dark},
		{"the build graph is below, with green bars", 199, 158, green},
		{"nothing is drawn below the graphs", 100, 170, none},
	}
	for _, tt := range tests {
		if got := img.RGBAAt(tt.x, tt.y); got != tt.want {
			t.Errorf("%s: pixel (%d, %d) = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
	// The line at the frame budget is half way up the graph.
	if got := img.RGBAAt(100, 40); got == dark {
		t.Errorf("the budget line is missing at (100, 40): %v", got)
	}
}

func TestFrameTimingHistoryKeepsTheNewestFrames(t *testing.T) {
	h := rendering.NewFrameTimingHistory(3)
	for i := 1; i <= 5; i++ {
		h.Add(timing(i, time.Millisecond, time.Millisecond))
	}
	if h.Len() != 3 {
		t.Fatalf("len = %d, want 3", h.Len())
	}
	last := h.Last(10)
	if len(last) != 3 || last[0].FrameNumber() != 3 || last[2].FrameNumber() != 5 {
		t.Errorf("last = %v", last)
	}
	if two := h.Last(2); len(two) != 2 || two[0].FrameNumber() != 4 {
		t.Errorf("last two = %v", two)
	}
}
