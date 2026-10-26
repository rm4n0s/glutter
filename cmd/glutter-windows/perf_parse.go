package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

// The plain-Go parts of `glutter-windows perf`: what it reads from the output of an app (the
// engine's log lines) and from /proc. They are tested without Wine.

// firstFrame is the engine's "first frame presented" line.
type firstFrame struct {
	// SinceRun is the time from the start of the engine's Run to the first frame (Windows does not know
	// when the process started, so there is no since_process_start).
	SinceRun time.Duration
	// Renderer is the renderer and device, "d3d12: NVIDIA GeForce GTX 470 (10DE:06CD)".
	Renderer string
	// At is the time of the line (a second's resolution), zero when the line has none.
	At time.Time
	// GoHeapMB and GoSysMB are the memory of the Go runtime at that moment.
	GoHeapMB, GoSysMB int
}

// parseFirstFrame finds the first "first frame presented" line of the output of an app and reads it;
// ok is false when it has none or its since_run is not a duration. loc is the location of the times
// in the log.
func parseFirstFrame(output string, loc *time.Location) (ff firstFrame, ok bool) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "first frame presented") {
			continue
		}
		kv := perfstats.ParseLogLine(line)
		run, err := time.ParseDuration(kv["since_run"])
		if err != nil {
			continue
		}
		ff = firstFrame{SinceRun: run, Renderer: kv["renderer"]}
		ff.GoHeapMB, _ = strconv.Atoi(kv["go_heap_mb"])
		ff.GoSysMB, _ = strconv.Atoi(kv["go_sys_mb"])
		if t, found := perfstats.ParseLogTime(line, loc); found {
			ff.At = t
		}
		return ff, true
	}
	return firstFrame{}, false
}

// startupStages are the parts of the first frame that the engine logs with GLUTTER_FRAME_STATS=1
// ("startup" lines, SPEC §9.7): opening the GPU device, making the shaders (built, or taken from the
// shader cache), the rest of the renderer's first frame, and the whole of it.
type startupStages struct {
	// Device, Shaders, Rest and Whole are the "took" of the stages "GPU context or device", "shaders",
	// "frame without the above" and "first frame of the renderer"; zero for a stage that was not
	// logged.
	Device, Shaders, Rest, Whole time.Duration
	// Built and FromCache are the shaders that were compiled and those read from the cache; HasShaders
	// says the line was there.
	Built, FromCache int
	HasShaders       bool
}

// parseStartup reads the "startup" lines of the output of an app.
func parseStartup(output string) startupStages {
	var s startupStages
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "startup") {
			continue
		}
		kv := perfstats.ParseLogLine(line)
		took, err := time.ParseDuration(kv["took"])
		if err != nil {
			continue
		}
		switch kv["stage"] {
		case "GPU context or device":
			s.Device = took
		case "shaders":
			s.Shaders, s.HasShaders = took, true
			s.Built, _ = strconv.Atoi(kv["built"])
			s.FromCache, _ = strconv.Atoi(kv["from_cache"])
		case "frame without the above":
			s.Rest = took
		case "first frame of the renderer":
			s.Whole = took
		}
	}
	return s
}

// frameWindow says which "frames" lines of an output count.
type frameWindow struct {
	// afterFirstFrame leaves out the lines before the "first frame presented" line.
	afterFirstFrame bool
	// from and to, when not zero, keep the lines whose time is at or after the second of from and at or
	// before the second of to. The log's times have a second's resolution.
	from, to time.Time
}

// parseFrameLines reads the "frames" statistics lines of the output of an app that are in the window,
// in order; lines with a value that is not a number are skipped. loc is the location of the log's
// times.
func parseFrameLines(output string, w frameWindow, loc *time.Location) []perfstats.Frames {
	var out []perfstats.Frames
	seenFirst := !w.afterFirstFrame
	for _, line := range strings.Split(output, "\n") {
		if !seenFirst {
			seenFirst = strings.Contains(line, "first frame presented")
			continue
		}
		if !perfstats.IsFramesLine(line) {
			continue
		}
		if t, ok := perfstats.ParseLogTime(line, loc); ok {
			if !w.from.IsZero() && t.Before(w.from.Truncate(time.Second)) || !w.to.IsZero() && t.After(w.to.Truncate(time.Second)) {
				continue
			}
		}
		if f, ok := perfstats.ParseFramesLine(line); ok {
			out = append(out, f)
		}
	}
	return out
}

// tailLines is the last n lines of s.
func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// clockTicksPerSecond is USER_HZ, the unit of utime and stime in /proc/<pid>/stat: 100 on every
// Linux that matters.
const clockTicksPerSecond = 100
