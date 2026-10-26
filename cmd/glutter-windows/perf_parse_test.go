package main

import (
	"testing"
	"time"
)

// logOf is the output of the counter under Wine as step 28.15 saw it (shortened).
const logOf = `2026/10/26 03:28:02 INFO startup stage="connected to the display" since_run=90ms
MESA-EGL: warning: DRI3 error: Could not get DRI3 device
2026/10/26 03:28:03 INFO Direct3D 12 device device="d3d12: NVIDIA GeForce GTX 470 (10DE:06CD)" feature_level=11_0
2026/10/26 03:28:05 INFO startup stage="first frame of the renderer" renderer=d3d12 took=2.4301s
2026/10/26 03:28:05 INFO startup stage="GPU context or device" renderer=d3d12 took=156.5ms
2026/10/26 03:28:05 INFO startup stage=shaders renderer=d3d12 took=266.3ms built=10 from_cache=0 refused=0
2026/10/26 03:28:05 INFO startup stage="frame without the above" renderer=d3d12 took=2.0073s
2026/10/26 03:28:05 INFO first frame presented since_run=2.587s renderer="d3d12: NVIDIA GeForce GTX 470 (10DE:06CD)" go_heap_mb=8 go_sys_mb=38
2026/10/26 03:28:06 INFO frames renderer=d3d12 fps=60 build_avg=2.1ms build_max=5ms raster_avg=1.5ms raster_max=4ms over_16ms=1 go_heap_mb=9 go_sys_mb=38
2026/10/26 03:28:07 INFO frames renderer=d3d12 fps=58 build_avg=2.2ms build_max=6ms raster_avg=1.6ms raster_max=5ms over_16ms=0 go_heap_mb=9 go_sys_mb=38
2026/10/26 03:28:08 INFO frames renderer=d3d12 fps=59 build_avg=2.0ms build_max=4ms raster_avg=1.4ms raster_max=3ms over_16ms=0 go_heap_mb=10 go_sys_mb=39
`

func TestParseFirstFrame(t *testing.T) {
	utc := time.UTC
	tests := []struct {
		name   string
		output string
		want   firstFrame
		ok     bool
	}{
		{"the counter's log", logOf, firstFrame{SinceRun: 2587 * time.Millisecond, Renderer: "d3d12: NVIDIA GeForce GTX 470 (10DE:06CD)",
			At: time.Date(2026, 10, 26, 3, 28, 5, 0, utc), GoHeapMB: 8, GoSysMB: 38}, true},
		{"a line without the time", `INFO first frame presented since_run=300ms renderer=soft`, firstFrame{SinceRun: 300 * time.Millisecond, Renderer: "soft"}, true},
		{"since_run that is not a duration is skipped for the next line", "INFO first frame presented since_run=soon\nINFO first frame presented since_run=1s renderer=soft", firstFrame{SinceRun: time.Second, Renderer: "soft"}, true},
		{"no first frame", "INFO startup stage=shaders took=1ms", firstFrame{}, false},
		{"empty", "", firstFrame{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFirstFrame(tt.output, utc)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseFirstFrame = %+v, %v, want %+v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseStartup(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name   string
		output string
		want   startupStages
	}{
		{"an empty cache", logOf, startupStages{Device: 156*ms + 500*time.Microsecond, Shaders: 266*ms + 300*time.Microsecond, Rest: 2007*ms + 300*time.Microsecond,
			Whole: 2430*ms + 100*time.Microsecond, Built: 10, HasShaders: true}},
		{"a warm cache", `INFO startup stage=shaders renderer=d3d12 took=12ms built=0 from_cache=10`, startupStages{Shaders: 12 * ms, FromCache: 10, HasShaders: true}},
		{"a stage with a bad duration is skipped", `INFO startup stage=shaders took=fast built=1`, startupStages{}},
		{"no startup lines", "INFO frames fps=60", startupStages{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStartup(tt.output); got != tt.want {
				t.Errorf("parseStartup = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseFrameLines(t *testing.T) {
	utc := time.UTC
	at := func(sec int) time.Time { return time.Date(2026, 10, 26, 3, 28, sec, 0, utc) }
	tests := []struct {
		name string
		w    frameWindow
		fps  []int
	}{
		{"all lines", frameWindow{}, []int{60, 58, 59}},
		{"after the first frame", frameWindow{afterFirstFrame: true}, []int{60, 58, 59}},
		{"from the second of 03:28:07", frameWindow{from: at(7).Add(300 * time.Millisecond)}, []int{58, 59}},
		{"up to 03:28:07", frameWindow{to: at(7)}, []int{60, 58}},
		{"from and to", frameWindow{from: at(7), to: at(7)}, []int{58}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int
			for _, f := range parseFrameLines(logOf, tt.w, utc) {
				got = append(got, f.FPS)
			}
			if len(got) != len(tt.fps) {
				t.Fatalf("fps = %v, want %v", got, tt.fps)
			}
			for i := range got {
				if got[i] != tt.fps[i] {
					t.Fatalf("fps = %v, want %v", got, tt.fps)
				}
			}
		})
	}
	t.Run("lines before the first frame are left out", func(t *testing.T) {
		out := "2026/10/26 03:28:01 INFO frames fps=5 build_avg=1ms build_max=1ms raster_avg=1ms raster_max=1ms over_16ms=0 go_heap_mb=1 go_sys_mb=1\n" + logOf
		if got := parseFrameLines(out, frameWindow{afterFirstFrame: true}, utc); len(got) != 3 {
			t.Errorf("%d lines, want 3", len(got))
		}
		if got := parseFrameLines(out, frameWindow{}, utc); len(got) != 4 {
			t.Errorf("%d lines, want 4", len(got))
		}
	})
	t.Run("a frames line with a bad value is skipped", func(t *testing.T) {
		if got := parseFrameLines("INFO frames fps=x build_avg=1ms", frameWindow{}, utc); len(got) != 0 {
			t.Errorf("%v, want none", got)
		}
	})
}

func TestTailLines(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"a\nb\nc\nd", 2, "c\nd"},
		{"a\nb", 5, "a\nb"},
		{"  a\n", 1, "a"},
		{"", 3, ""},
	}
	for _, tt := range tests {
		if got := tailLines(tt.in, tt.n); got != tt.want {
			t.Errorf("tailLines(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}
