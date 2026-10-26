// Package perfstats reads the engine's log lines that the performance tools measure with: the
// "first frame presented" line, the per-second "frames" statistics of GLUTTER_FRAME_STATS=1 and
// the key=value pairs of slog's text format. It also sums the frames of a window up, reads the
// counters of /proc (CPU ticks, VmRSS, Pss) and renders a report as text or Markdown tables.
// internal/webtest (the web), cmd/glutter-android (Android) and cmd/glutter-windows (Windows,
// under Wine) use it. It has no dependency on the platform.
//
// Flutter: no counterpart
package perfstats

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Frames is one "frames" line of the engine's statistics (GLUTTER_FRAME_STATS=1): the frames
// of one second.
//
// Flutter: no counterpart
type Frames struct {
	Renderer string
	// FPS is the number of frames drawn in the second.
	FPS int
	// Build and Raster are the average and the worst time on the UI goroutine of the build
	// (build, layout, paint and the scene) and of the renderer's calls; Over16 is the number of
	// frames whose sum was over a 60th of a second.
	BuildAvg, BuildMax, RasterAvg, RasterMax time.Duration
	Over16                                   int
	// GoHeapMB and GoSysMB are the memory of the Go runtime.
	GoHeapMB, GoSysMB int
}

// ParseLogLine splits a log line of the engine (slog's text format, key=value with values in
// quotes when they hold blanks) into its pairs.
//
// Flutter: no counterpart
func ParseLogLine(line string) map[string]string {
	kv := map[string]string{}
	for i := 0; i < len(line); {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		start := i
		for i < len(line) && line[i] != ' ' && line[i] != '=' {
			i++
		}
		if i >= len(line) || line[i] != '=' {
			// A word that is not a pair (the time, the level, the message): skip it.
			for i < len(line) && line[i] != ' ' {
				i++
			}
			continue
		}
		key := line[start:i]
		i++
		var value string
		if i < len(line) && line[i] == '"' {
			end := i + 1
			for end < len(line) && (line[end] != '"' || line[end-1] == '\\') {
				end++
			}
			if end < len(line) {
				if v, err := strconv.Unquote(line[i : end+1]); err == nil {
					value = v
				} else {
					value = line[i+1 : end]
				}
				i = end + 1
			} else {
				value, i = line[i+1:], len(line)
			}
		} else {
			end := i
			for end < len(line) && line[end] != ' ' {
				end++
			}
			value, i = line[i:end], end
		}
		kv[key] = value
	}
	return kv
}

// IsFramesLine reports whether a log line looks like a "frames" statistics line: the message
// "frames" of the default logger ("INFO frames fps=...") or of slog's text handler
// ("msg=frames"). ParseFramesLine says whether it also has every value.
//
// Flutter: no counterpart
func IsFramesLine(line string) bool {
	return strings.Contains(line, " frames ") || strings.Contains(line, "msg=frames ")
}

// ParseFramesLine reads a "frames" line of the engine's statistics; ok is false for a line that
// is not one or has a value that is not a number or a duration.
//
// Flutter: no counterpart
func ParseFramesLine(line string) (Frames, bool) {
	kv := ParseLogLine(line)
	var f Frames
	f.Renderer = kv["renderer"]
	ints := map[string]*int{"fps": &f.FPS, "over_16ms": &f.Over16, "go_heap_mb": &f.GoHeapMB, "go_sys_mb": &f.GoSysMB}
	for k, dst := range ints {
		v, err := strconv.Atoi(kv[k])
		if err != nil {
			return Frames{}, false
		}
		*dst = v
	}
	durations := map[string]*time.Duration{"build_avg": &f.BuildAvg, "build_max": &f.BuildMax, "raster_avg": &f.RasterAvg, "raster_max": &f.RasterMax}
	for k, dst := range durations {
		v, err := time.ParseDuration(kv[k])
		if err != nil {
			return Frames{}, false
		}
		*dst = v
	}
	return f, true
}

// Median is the middle value of the durations (the upper one for an even count), or zero for none.
//
// Flutter: no counterpart
func Median(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}

// DurStats is the minimum, the median and the maximum of some durations.
//
// Flutter: no counterpart
type DurStats struct {
	N                int
	Min, Median, Max time.Duration
}

// SummarizeDurations computes DurStats; the zero value for none.
//
// Flutter: no counterpart
func SummarizeDurations(d []time.Duration) DurStats {
	if len(d) == 0 {
		return DurStats{}
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return DurStats{N: len(s), Min: s[0], Median: Median(s), Max: s[len(s)-1]}
}

// CPUPercent is the CPU time between two readings of utime+stime (in ticks of hz a second) as a
// percentage of one core over the elapsed wall time; 0 for no time, a counter that went back, or no
// tick rate.
//
// Flutter: no counterpart
func CPUPercent(ticks0, ticks1 uint64, elapsed time.Duration, hz int) float64 {
	if elapsed <= 0 || ticks1 < ticks0 || hz <= 0 {
		return 0
	}
	cpu := time.Duration(ticks1-ticks0) * time.Second / time.Duration(hz)
	return 100 * float64(cpu) / float64(elapsed)
}

// FrameSummary sums up the per-second statistics lines of a window.
//
// Flutter: no counterpart
type FrameSummary struct {
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

// SummarizeFrames computes FrameSummary; the zero value for none.
//
// Flutter: no counterpart
func SummarizeFrames(fr []Frames) FrameSummary {
	var s FrameSummary
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

// ProcTicks returns utime+stime in clock ticks from the text of /proc/<pid>/stat. The second
// field, the command in parentheses, may hold blanks and parentheses, so the fields are counted from
// the last ')'. ok is false for text that is not a stat line.
//
// Flutter: no counterpart
func ProcTicks(stat string) (ticks uint64, ok bool) {
	end := strings.LastIndexByte(stat, ')')
	if end < 0 {
		return 0, false
	}
	f := strings.Fields(stat[end+1:])
	// f[0] is the state (field 3); utime and stime are fields 14 and 15.
	if len(f) < 13 {
		return 0, false
	}
	utime, errU := strconv.ParseUint(f[11], 10, 64)
	stime, errS := strconv.ParseUint(f[12], 10, 64)
	if errU != nil || errS != nil {
		return 0, false
	}
	return utime + stime, true
}

// ProcMemory reads VmRSS and VmHWM, in kilobytes, from the text of /proc/<pid>/status. ok is false
// when VmRSS is missing (a kernel thread, or a process that ended).
//
// Flutter: no counterpart
func ProcMemory(status string) (rssKB, hwmKB int, ok bool) {
	for _, line := range strings.Split(status, "\n") {
		key, rest, found := strings.Cut(line, ":")
		if !found || (key != "VmRSS" && key != "VmHWM") {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 1 {
			continue
		}
		n, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		if key == "VmRSS" {
			rssKB, ok = n, true
		} else {
			hwmKB = n
		}
	}
	return rssKB, hwmKB, ok
}

// RollupPSS reads Pss, in kilobytes, from the text of /proc/<pid>/smaps_rollup; ok is false when it
// has none.
//
// Flutter: no counterpart
func RollupPSS(rollup string) (pssKB int, ok bool) {
	for _, line := range strings.Split(rollup, "\n") {
		rest, found := strings.CutPrefix(line, "Pss:")
		if !found {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 1 {
			continue
		}
		if n, err := strconv.Atoi(f[0]); err == nil {
			return n, true
		}
	}
	return 0, false
}

// logTimeLayout is the time that the default logger of the log package puts first in a line.
const logTimeLayout = "2006/01/02 15:04:05"

// ParseLogTime reads the time at the start of a line of the default logger ("2026/10/26 03:28:05
// INFO ..."), in the given location; ok is false for a line that does not start with one. A program
// under Wine logs the time of the host, so the location is the host's.
//
// Flutter: no counterpart
func ParseLogTime(line string, loc *time.Location) (time.Time, bool) {
	if len(line) < len(logTimeLayout) {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(logTimeLayout, line[:len(logTimeLayout)], loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
