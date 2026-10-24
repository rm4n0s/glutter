package main

import (
	"sort"
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

// The summaries of `glutter-android perf`: pure functions of the measured numbers.

// durStats is the minimum, the median and the maximum of some durations.
type durStats struct {
	N                int
	Min, Median, Max time.Duration
}

// summarizeDurations computes durStats; the zero value for none.
func summarizeDurations(d []time.Duration) durStats {
	if len(d) == 0 {
		return durStats{}
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return durStats{N: len(s), Min: s[0], Median: perfstats.Median(s), Max: s[len(s)-1]}
}

// cpuPercent is the CPU time between two readings of utime+stime as a percentage of one core
// over the elapsed wall time; 0 for no time.
func cpuPercent(ticks0, ticks1 uint64, elapsed time.Duration) float64 {
	if elapsed <= 0 || ticks1 < ticks0 {
		return 0
	}
	cpu := time.Duration(ticks1-ticks0) * time.Second / clockTicksPerSecond
	return 100 * float64(cpu) / float64(elapsed)
}

// frameSummary sums up the per-second statistics lines of a window.
type frameSummary struct {
	// Seconds is the number of statistics lines (a second without a frame has none) and Frames
	// the frames in them.
	Seconds, Frames int
	// MeanFPS is the frames of the seconds that had some, per second; MinFPS and MaxFPS are the
	// lowest and the highest second.
	MeanFPS        float64
	MinFPS, MaxFPS int
	// BuildAvg and RasterAvg are the averages over all frames; BuildMax and RasterMax the worst
	// frame; WorstSecond the highest average build+raster of one second.
	BuildAvg, RasterAvg, BuildMax, RasterMax, WorstSecond time.Duration
	// Over16 is the number of frames over a 60th of a second, and Over16Percent its share.
	Over16        int
	Over16Percent float64
	// HeapMaxMB and HeapLastMB are the Go heap in use, SysLastMB what the runtime got from the
	// system.
	HeapMaxMB, HeapLastMB, SysLastMB int
}

// summarizeFrames computes frameSummary; the zero value for none.
func summarizeFrames(fr []perfstats.Frames) frameSummary {
	var s frameSummary
	if len(fr) == 0 {
		return s
	}
	var build, raster time.Duration
	s.MinFPS = fr[0].FPS
	for _, f := range fr {
		s.Seconds++
		s.Frames += f.FPS
		build += f.BuildAvg * time.Duration(f.FPS)
		raster += f.RasterAvg * time.Duration(f.FPS)
		s.BuildMax, s.RasterMax = max(s.BuildMax, f.BuildMax), max(s.RasterMax, f.RasterMax)
		s.WorstSecond = max(s.WorstSecond, f.BuildAvg+f.RasterAvg)
		s.MinFPS, s.MaxFPS = min(s.MinFPS, f.FPS), max(s.MaxFPS, f.FPS)
		s.Over16 += f.Over16
		s.HeapMaxMB = max(s.HeapMaxMB, f.GoHeapMB)
	}
	last := fr[len(fr)-1]
	s.HeapLastMB, s.SysLastMB = last.GoHeapMB, last.GoSysMB
	s.MeanFPS = float64(s.Frames) / float64(s.Seconds)
	if s.Frames > 0 {
		s.BuildAvg = build / time.Duration(s.Frames)
		s.RasterAvg = raster / time.Duration(s.Frames)
		s.Over16Percent = 100 * float64(s.Over16) / float64(s.Frames)
	}
	return s
}
