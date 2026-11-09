package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

// The plain-Go parts of `glutter-macos perf`: what it reads from the output of an app (the engine's log lines) and from `ps`. They
// are tested without a window.

// parsePSTime reads the CPU time that `ps -o cputime` prints on macOS: "0:01.23", "12:34.56" or "1:02:03.45", with an optional
// "days-" before the hours. ok is false for a text that is none of these.
func parsePSTime(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	var days float64
	if d, rest, found := strings.Cut(s, "-"); found {
		n, err := strconv.Atoi(d)
		if err != nil {
			return 0, false
		}
		days, s = float64(n), rest
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	secs, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil || secs < 0 {
		return 0, false
	}
	total := secs
	mult := 60.0
	for i := len(parts) - 2; i >= 0; i-- {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return 0, false
		}
		total += float64(n) * mult
		mult *= 60
	}
	total += days * 86400
	return time.Duration(total * float64(time.Second)), true
}

// psSample is what `ps -o rss=,cputime= -p <pid>` says of a process at one moment.
type psSample struct {
	RSSKB int
	CPU   time.Duration
}

// parsePS reads the line of `ps -o rss=,cputime=`.
func parsePS(out string) (psSample, bool) {
	f := strings.Fields(out)
	if len(f) != 2 {
		return psSample{}, false
	}
	rss, err := strconv.Atoi(f[0])
	if err != nil || rss < 0 {
		return psSample{}, false
	}
	cpu, ok := parsePSTime(f[1])
	if !ok {
		return psSample{}, false
	}
	return psSample{RSSKB: rss, CPU: cpu}, true
}

// cpuPercent is the share of one core that the process used between two samples taken `elapsed` apart.
func cpuPercent(a, b psSample, elapsed time.Duration) float64 {
	if elapsed <= 0 || b.CPU < a.CPU {
		return 0
	}
	return 100 * float64(b.CPU-a.CPU) / float64(elapsed)
}

// firstFrame is the engine's "first frame presented" line.
type firstFrame struct {
	// SinceRun is the time from the start of the engine's Run to the first frame; SinceProcess the time from the start of the process
	// (the engine reads it from the system); zero when the line has none.
	SinceRun, SinceProcess time.Duration
	// Renderer is the renderer and device, "metal: Apple M6".
	Renderer string
	// GoHeapMB and GoSysMB are the memory of the Go runtime at that moment.
	GoHeapMB, GoSysMB int
}

// parseFirstFrame finds the first "first frame presented" line of the output of an app; ok is false when it has none or its
// since_run is not a duration.
func parseFirstFrame(output string) (ff firstFrame, ok bool) {
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
		ff.SinceProcess, _ = time.ParseDuration(kv["since_process_start"])
		ff.GoHeapMB, _ = strconv.Atoi(kv["go_heap_mb"])
		ff.GoSysMB, _ = strconv.Atoi(kv["go_sys_mb"])
		return ff, true
	}
	return firstFrame{}, false
}

// framesOf reads the per-second statistics lines of the output of an app that come after its first frame.
func framesOf(output string) []perfstats.Frames {
	var out []perfstats.Frames
	seenFirst := false
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "first frame presented") {
			seenFirst = true
			continue
		}
		if !seenFirst || !perfstats.IsFramesLine(line) {
			continue
		}
		if f, ok := perfstats.ParseFramesLine(line); ok {
			out = append(out, f)
		}
	}
	return out
}
