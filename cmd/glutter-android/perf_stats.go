package main

import (
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

// The summaries of `glutter-android perf` are those of internal/perfstats, which `glutter-windows
// perf` shares.

// durStats is the minimum, the median and the maximum of some durations.
type durStats = perfstats.DurStats

// summarizeDurations computes durStats; the zero value for none.
func summarizeDurations(d []time.Duration) durStats { return perfstats.SummarizeDurations(d) }

// cpuPercent is the CPU time between two readings of utime+stime as a percentage of one core
// over the elapsed wall time; 0 for no time.
func cpuPercent(ticks0, ticks1 uint64, elapsed time.Duration) float64 {
	return perfstats.CPUPercent(ticks0, ticks1, elapsed, clockTicksPerSecond)
}

// frameSummary sums up the per-second statistics lines of a window.
type frameSummary = perfstats.FrameSummary

// summarizeFrames computes frameSummary; the zero value for none.
func summarizeFrames(fr []perfstats.Frames) frameSummary { return perfstats.SummarizeFrames(fr) }
