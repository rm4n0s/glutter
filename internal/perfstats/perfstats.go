// Package perfstats reads the engine's log lines that the performance tools measure with: the
// "first frame presented" line, the per-second "frames" statistics of GLUTTER_FRAME_STATS=1 and
// the key=value pairs of slog's text format. internal/webtest (the web) and cmd/glutter-android
// (Android) both use it. It has no dependency on the platform.
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
