package engine

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// frameClock measures the frames for ui.PlatformDispatcher.OnReportTimings. The
// handler marks the start of a frame's build and the two ends of its raster; the
// timing is made at the end of the frame, not inside it, because the callback runs
// framework code (the performance overlay marks itself for paint) that must not run
// in the middle of a build.
type frameClock struct {
	// epoch is the fixed point the timings count from.
	epoch time.Time
	// buildStart is when the frame began; zero if none did.
	buildStart time.Duration
	// raster is the span of the last renderer call of the frame, with done set once
	// it ended.
	rasterStart, rasterFinish time.Duration
	rastered                  bool
	number                    int
}

func (c *frameClock) now() time.Duration {
	if c.epoch.IsZero() {
		c.epoch = time.Now()
	}
	return time.Since(c.epoch)
}

// beginFrame marks the start of a frame's build.
func (c *frameClock) beginFrame() {
	c.buildStart = c.now()
	c.rastered = false
}

// beginRaster and endRaster mark the renderer's work on the frame's scene.
func (c *frameClock) beginRaster() { c.rasterStart = c.now() }
func (c *frameClock) endRaster() {
	c.rasterFinish = c.now()
	c.rastered = true
}

// finish returns the timing of the frame if the renderer drew one in it, and says
// whether there is one. A frame whose build was not run in this call (a window that
// presents a scene the framework rendered during another window's frame) has no build.
func (c *frameClock) finish(built bool) (ui.FrameTiming, bool) {
	if !c.rastered {
		return ui.FrameTiming{}, false
	}
	c.rastered = false
	c.number++
	start, finish := c.buildStart, c.rasterStart
	if !built {
		start, finish = c.rasterStart, c.rasterStart
	}
	// The compositor's frame time is on its own clock, so the vsync is taken to be the
	// start of the build (SPEC §12.10).
	return ui.NewFrameTiming(start, start, finish, c.rasterStart, c.rasterFinish, time.Duration(time.Now().UnixNano()), c.number), true
}

// reportFrame gives the dispatcher the timing of the frame that just ended, if anything
// listens.
func (h *handler) reportFrame(built bool) {
	if h.disp == nil || !h.disp.NeedsReportTimings() {
		h.clock.rastered = false
		return
	}
	if t, ok := h.clock.finish(built); ok {
		h.disp.ReportTimings([]ui.FrameTiming{t})
	}
}

// traceRaster marks the renderer's work in the execution trace (SPEC §14.3).
func traceRaster() foundation.TraceRegion { return foundation.StartTraceRegion("RASTER") }
