package engine

import (
	"runtime"
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

// frameBudget is how long a frame may take for the statistics: a 60th of a second.
const frameBudget = time.Second / 60

// frameStats are the frames of the last second, for FrameStatsEnv.
type frameStats struct {
	since                time.Time
	frames, slow         int
	build, raster        time.Duration
	worstBuild, worstRas time.Duration
}

// noteFrame is called after each frame that the renderer drew: the first one is logged
// with how long the program took to start, and with FrameStatsEnv the frames are summed
// and logged once a second.
func (h *handler) noteFrame() {
	now := time.Now()
	if !h.firstLogged {
		h.firstLogged = true
		attrs := []any{"since_run", now.Sub(h.runStart).Round(time.Millisecond)}
		if name := h.rendererName(); name != "" {
			attrs = append(attrs, "renderer", name)
		}
		if start, ok := processStart(); ok {
			attrs = append(attrs, "since_process_start", now.Sub(start).Round(10*time.Millisecond))
		}
		h.log.Info("first frame presented", append(attrs, goMemory()...)...)
	}
	if !h.frameStats {
		return
	}
	s := &h.stats
	if s.since.IsZero() {
		s.since = now
	}
	raster := h.clock.rasterFinish - h.clock.rasterStart
	build := h.clock.rasterStart - h.clock.buildStart
	if build < 0 {
		build = 0
	}
	s.frames++
	s.build += build
	s.raster += raster
	s.worstBuild, s.worstRas = max(s.worstBuild, build), max(s.worstRas, raster)
	if build+raster > frameBudget {
		s.slow++
	}
	if now.Sub(s.since) >= time.Second {
		attrs := []any{
			"fps", s.frames, "build_avg", (s.build / time.Duration(s.frames)).Round(10 * time.Microsecond),
			"build_max", s.worstBuild.Round(10 * time.Microsecond), "raster_avg", (s.raster / time.Duration(s.frames)).Round(10 * time.Microsecond),
			"raster_max", s.worstRas.Round(10 * time.Microsecond), "over_16ms", s.slow,
		}
		if name := h.rendererName(); name != "" {
			attrs = append([]any{"renderer", name}, attrs...)
		}
		h.log.Info("frames", append(attrs, goMemory()...)...)
		*s = frameStats{since: now}
	}
}

// rendererName is the renderer and its device, as the renderer says them ("gles: llvmpipe
// (LLVM 19.1.7, 256 bits)"), or "" when the renderer does not.
func (h *handler) rendererName() string {
	if d, ok := h.r.(describer); ok {
		return d.Describe()
	}
	return ""
}

// goMemory is the memory of the Go runtime in megabytes, as log attributes: the heap in
// use and everything the runtime got from the system. The resident size of the process
// also has the graphics driver's, which is not the framework's.
func goMemory() []any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return []any{"go_heap_mb", int(m.HeapAlloc >> 20), "go_sys_mb", int(m.Sys >> 20)}
}
