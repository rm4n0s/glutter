package ui

import (
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
)

// FramePhase is a moment of a frame that a FrameTiming records.
//
// Flutter: ui/platform_dispatcher.dart FramePhase
type FramePhase int

// The phases of a frame, in the order they happen.
const (
	// FramePhaseUnspecified is the zero value: not a phase.
	//
	// Flutter: no counterpart
	FramePhaseUnspecified FramePhase = iota
	// FramePhaseVsyncStart is when the platform asked for the frame (the compositor's
	// frame callback).
	//
	// Flutter: ui/platform_dispatcher.dart FramePhase.vsyncStart
	FramePhaseVsyncStart
	// FramePhaseBuildStart is when the framework began to build the frame (OnBeginFrame).
	//
	// Flutter: ui/platform_dispatcher.dart FramePhase.buildStart
	FramePhaseBuildStart
	// FramePhaseBuildFinish is when the framework handed the scene over to be drawn.
	//
	// Flutter: ui/platform_dispatcher.dart FramePhase.buildFinish
	FramePhaseBuildFinish
	// FramePhaseRasterStart is when the renderer began to draw the scene.
	//
	// Flutter: ui/platform_dispatcher.dart FramePhase.rasterStart
	FramePhaseRasterStart
	// FramePhaseRasterFinish is when the renderer was done drawing and had presented the
	// frame.
	//
	// Flutter: ui/platform_dispatcher.dart FramePhase.rasterFinish
	FramePhaseRasterFinish
	// FramePhaseRasterFinishWallTime is the wall-clock time at which the frame was
	// rasterized.
	//
	// Flutter: ui/platform_dispatcher.dart FramePhase.rasterFinishWallTime
	FramePhaseRasterFinishWallTime
)

// FrameTimingOpts are the optional arguments of NewFrameTiming.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.new
type FrameTimingOpts struct {
	// LayerCacheCount and LayerCacheBytes are the number and the size of the layers
	// in the raster cache.
	LayerCacheCount, LayerCacheBytes int
	// PictureCacheCount and PictureCacheBytes are the number and the size of the pictures
	// in the raster cache.
	PictureCacheCount, PictureCacheBytes int
}

// FrameTiming says when the phases of a frame happened and how long its parts took, as
// the engine measured them: a tool to find the frames that were too slow.
//
// Deviation: the times are time.Duration from an arbitrary fixed point, not integers of
// microseconds, and vsyncStart is the start of the build when the platform's
// frame time is not on the clock the engine measures with (SPEC §12.10).
//
// Flutter: ui/platform_dispatcher.dart FrameTiming
type FrameTiming struct {
	phases      [FramePhaseRasterFinishWallTime + 1]time.Duration
	frameNumber int
	opts        FrameTimingOpts
}

// NewFrameTiming creates the record of a frame. The times are from one fixed point,
// each not before the one before it, except the wall time.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.new
func NewFrameTiming(vsyncStart, buildStart, buildFinish, rasterStart, rasterFinish, rasterFinishWallTime time.Duration, frameNumber int, opts ...FrameTimingOpts) FrameTiming {
	t := FrameTiming{frameNumber: frameNumber, opts: oneOpts(opts)}
	t.phases[FramePhaseVsyncStart] = vsyncStart
	t.phases[FramePhaseBuildStart] = buildStart
	t.phases[FramePhaseBuildFinish] = buildFinish
	t.phases[FramePhaseRasterStart] = rasterStart
	t.phases[FramePhaseRasterFinish] = rasterFinish
	t.phases[FramePhaseRasterFinishWallTime] = rasterFinishWallTime
	return t
}

// TimestampOf returns when the phase happened. It panics with the tag FramePhaseInvalid
// for a value that is not a phase.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.timestampInMicroseconds
func (t FrameTiming) TimestampOf(phase FramePhase) time.Duration {
	if phase <= FramePhaseUnspecified || phase > FramePhaseRasterFinishWallTime {
		panic(errors.New("FramePhaseInvalid", "the value is not a frame phase", "phase", int(phase)))
	}
	return t.phases[phase]
}

// BuildDuration is how long the framework took to build the frame: the time spent in
// the animation callbacks, the build, the layout and the paint.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.buildDuration
func (t FrameTiming) BuildDuration() time.Duration {
	return t.phases[FramePhaseBuildFinish] - t.phases[FramePhaseBuildStart]
}

// RasterDuration is how long the renderer took to draw the frame and present it.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.rasterDuration
func (t FrameTiming) RasterDuration() time.Duration {
	return t.phases[FramePhaseRasterFinish] - t.phases[FramePhaseRasterStart]
}

// VsyncOverhead is how long it took from the platform's request for the frame to the
// start of the build.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.vsyncOverhead
func (t FrameTiming) VsyncOverhead() time.Duration {
	return t.phases[FramePhaseBuildStart] - t.phases[FramePhaseVsyncStart]
}

// TotalSpan is the time from the platform's request for the frame to the end of the
// raster.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.totalSpan
func (t FrameTiming) TotalSpan() time.Duration {
	return t.phases[FramePhaseRasterFinish] - t.phases[FramePhaseVsyncStart]
}

// FrameNumber is the number of the frame, counted from 1 by the engine.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.frameNumber
func (t FrameTiming) FrameNumber() int { return t.frameNumber }

// LayerCacheCount is the number of layers in the raster cache.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.layerCacheCount
func (t FrameTiming) LayerCacheCount() int { return t.opts.LayerCacheCount }

// LayerCacheBytes is the size of the layers in the raster cache.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.layerCacheBytes
func (t FrameTiming) LayerCacheBytes() int { return t.opts.LayerCacheBytes }

// PictureCacheCount is the number of pictures in the raster cache.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.pictureCacheCount
func (t FrameTiming) PictureCacheCount() int { return t.opts.PictureCacheCount }

// PictureCacheBytes is the size of the pictures in the raster cache.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.pictureCacheBytes
func (t FrameTiming) PictureCacheBytes() int { return t.opts.PictureCacheBytes }

// String describes the timing as Flutter does.
//
// Flutter: ui/platform_dispatcher.dart FrameTiming.toString
func (t FrameTiming) String() string {
	ms := func(d time.Duration) string { return strconv.FormatFloat(float64(d.Microseconds())/1000, 'f', 1, 64) }
	return "FrameTiming(buildDuration: " + ms(t.BuildDuration()) + "ms, rasterDuration: " + ms(t.RasterDuration()) +
		"ms, vsyncOverhead: " + ms(t.VsyncOverhead()) + "ms, totalSpan: " + ms(t.TotalSpan()) +
		"ms, layerCacheCount: " + strconv.Itoa(t.opts.LayerCacheCount) + ", layerCacheBytes: " + strconv.Itoa(t.opts.LayerCacheBytes) +
		", pictureCacheCount: " + strconv.Itoa(t.opts.PictureCacheCount) + ", pictureCacheBytes: " + strconv.Itoa(t.opts.PictureCacheBytes) +
		", frameNumber: " + strconv.Itoa(t.frameNumber) + ")"
}
