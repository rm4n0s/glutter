package ui

import (
	"testing"
	"time"
)

func TestFrameTiming(t *testing.T) {
	ms := time.Millisecond
	ft := NewFrameTiming(10*ms, 12*ms, 20*ms, 21*ms, 30*ms, 99*ms, 7, FrameTimingOpts{LayerCacheCount: 1, LayerCacheBytes: 2, PictureCacheCount: 3, PictureCacheBytes: 4})
	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"build", ft.BuildDuration(), 8 * ms},
		{"raster", ft.RasterDuration(), 9 * ms},
		{"vsync overhead", ft.VsyncOverhead(), 2 * ms},
		{"total span", ft.TotalSpan(), 20 * ms},
		{"vsync start", ft.TimestampOf(FramePhaseVsyncStart), 10 * ms},
		{"build start", ft.TimestampOf(FramePhaseBuildStart), 12 * ms},
		{"build finish", ft.TimestampOf(FramePhaseBuildFinish), 20 * ms},
		{"raster start", ft.TimestampOf(FramePhaseRasterStart), 21 * ms},
		{"raster finish", ft.TimestampOf(FramePhaseRasterFinish), 30 * ms},
		{"wall time", ft.TimestampOf(FramePhaseRasterFinishWallTime), 99 * ms},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if ft.FrameNumber() != 7 || ft.LayerCacheCount() != 1 || ft.LayerCacheBytes() != 2 || ft.PictureCacheCount() != 3 || ft.PictureCacheBytes() != 4 {
		t.Errorf("the frame number and caches are wrong: %v", ft)
	}
	want := "FrameTiming(buildDuration: 8.0ms, rasterDuration: 9.0ms, vsyncOverhead: 2.0ms, totalSpan: 20.0ms, layerCacheCount: 1, layerCacheBytes: 2, pictureCacheCount: 3, pictureCacheBytes: 4, frameNumber: 7)"
	if got := ft.String(); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestFrameTimingPhaseInvalidPanics(t *testing.T) {
	for _, phase := range []FramePhase{FramePhaseUnspecified, FramePhaseRasterFinishWallTime + 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("TimestampOf(%d) did not panic", phase)
				}
			}()
			NewFrameTiming(0, 0, 0, 0, 0, 0, 1).TimestampOf(phase)
		}()
	}
}
