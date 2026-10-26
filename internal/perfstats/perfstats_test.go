package perfstats

import (
	"strings"
	"testing"
	"time"
)

func TestParseLogLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want map[string]string
	}{
		{"plain pairs", "2026/10/21 23:46:27 INFO frames fps=117 build_avg=660µs over_16ms=0", map[string]string{"fps": "117", "build_avg": "660µs", "over_16ms": "0"}},
		{"a quoted value with blanks", `INFO first frame presented since_run=505ms renderer="webgpu: amd rdna-3" go_heap_mb=40`, map[string]string{"since_run": "505ms", "renderer": "webgpu: amd rdna-3", "go_heap_mb": "40"}},
		{"an escaped quote", `INFO x a="say \"hi\"" b=1`, map[string]string{"a": `say "hi"`, "b": "1"}},
		{"no pairs", "INFO nothing here", map[string]string{}},
		{"an unterminated quote", `INFO x a="open`, map[string]string{"a": "open"}},
		{"an empty value", "INFO x a= b=2", map[string]string{"a": "", "b": "2"}},
		{"a logcat brief line", `I/org.glutter.counter( 4242): 2026/10/24 10:11:12 INFO first frame presented since_run=212ms renderer="vulkan: Adreno (TM) 740"`,
			map[string]string{"since_run": "212ms", "renderer": "vulkan: Adreno (TM) 740"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseLogLine(tc.line)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseLogLine = %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("ParseLogLine[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseFramesLine(t *testing.T) {
	good := `2026/10/21 23:46:27 INFO frames renderer="webgpu: amd rdna-3" fps=117 build_avg=660µs build_max=1.6ms raster_avg=2.79ms raster_max=13ms over_16ms=2 go_heap_mb=40 go_sys_mb=66`
	f, ok := ParseFramesLine(good)
	if !ok {
		t.Fatal("a frames line was not read")
	}
	if f.FPS != 117 || f.Over16 != 2 || f.GoHeapMB != 40 || f.GoSysMB != 66 || f.Renderer != "webgpu: amd rdna-3" ||
		f.BuildAvg != 660*time.Microsecond || f.BuildMax != 1600*time.Microsecond || f.RasterAvg != 2790*time.Microsecond || f.RasterMax != 13*time.Millisecond {
		t.Errorf("ParseFramesLine = %+v", f)
	}
	for _, bad := range []string{
		"INFO frames fps=1",
		strings.Replace(good, "fps=117", "fps=many", 1),
		strings.Replace(good, "build_avg=660µs", "build_avg=fast", 1),
	} {
		if _, ok := ParseFramesLine(bad); ok {
			t.Errorf("ParseFramesLine accepted %q", bad)
		}
	}
}

func TestIsFramesLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"2026/10/21 23:46:27 INFO frames fps=117", true},
		{"time=2026-10-21T23:46:27.000Z level=INFO msg=frames fps=117", true},
		{"I/org.glutter.counter( 4242): 2026/10/24 10:11:12 INFO frames renderer=x fps=60", true},
		{"INFO first frame presented since_run=5ms", false},
		{"INFO slow frame build=20ms raster=1ms", false},
	}
	for _, tc := range tests {
		if got := IsFramesLine(tc.line); got != tc.want {
			t.Errorf("IsFramesLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestMedian(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name string
		in   []time.Duration
		want time.Duration
	}{
		{"none", nil, 0},
		{"one", []time.Duration{5 * ms}, 5 * ms},
		{"odd, unsorted", []time.Duration{9 * ms, 1 * ms, 5 * ms}, 5 * ms},
		{"even takes the upper middle", []time.Duration{4 * ms, 1 * ms, 2 * ms, 3 * ms}, 3 * ms},
	}
	for _, tc := range tests {
		in := append([]time.Duration(nil), tc.in...)
		if got := Median(tc.in); got != tc.want {
			t.Errorf("%s: Median = %v, want %v", tc.name, got, tc.want)
		}
		for i := range in {
			if in[i] != tc.in[i] {
				t.Errorf("%s: Median reordered its input", tc.name)
			}
		}
	}
}

func TestSummarizeDurations(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name string
		in   []time.Duration
		want DurStats
	}{
		{"none", nil, DurStats{}},
		{"one", []time.Duration{7 * ms}, DurStats{N: 1, Min: 7 * ms, Median: 7 * ms, Max: 7 * ms}},
		{"five starts, the first the coldest", []time.Duration{900 * ms, 410 * ms, 400 * ms, 430 * ms, 420 * ms}, DurStats{N: 5, Min: 400 * ms, Median: 420 * ms, Max: 900 * ms}},
		{"even count takes the upper middle", []time.Duration{1 * ms, 2 * ms, 3 * ms, 4 * ms}, DurStats{N: 4, Min: 1 * ms, Median: 3 * ms, Max: 4 * ms}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeDurations(tt.in); got != tt.want {
				t.Errorf("SummarizeDurations = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCPUPercent(t *testing.T) {
	tests := []struct {
		name    string
		t0, t1  uint64
		elapsed time.Duration
		hz      int
		want    float64
	}{
		{"idle", 1834, 1834, 10 * time.Second, 100, 0},
		{"one tick a second is 1%", 1000, 1010, 10 * time.Second, 100, 1},
		{"a full core", 0, 1000, 10 * time.Second, 100, 100},
		{"a window of 10.5 s", 100, 121, 10500 * time.Millisecond, 100, 2},
		{"another tick rate", 0, 250, 10 * time.Second, 250, 10},
		{"no time", 1, 5, 0, 100, 0},
		{"no tick rate", 1, 5, time.Second, 0, 0},
		{"a counter that went back (a new process)", 50, 10, time.Second, 100, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CPUPercent(tt.t0, tt.t1, tt.elapsed, tt.hz); got < tt.want-1e-9 || got > tt.want+1e-9 {
				t.Errorf("CPUPercent = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeFrames(t *testing.T) {
	ms := time.Millisecond
	fr := []Frames{
		{FPS: 60, BuildAvg: 2 * ms, BuildMax: 5 * ms, RasterAvg: 4 * ms, RasterMax: 9 * ms, Over16: 1, GoHeapMB: 10, GoSysMB: 30},
		{FPS: 20, BuildAvg: 6 * ms, BuildMax: 8 * ms, RasterAvg: 12 * ms, RasterMax: 30 * ms, Over16: 10, GoHeapMB: 12, GoSysMB: 31},
	}
	got := SummarizeFrames(fr)
	want := FrameSummary{Seconds: 2, Frames: 80, MeanFPS: 40, MinFPS: 20, MaxFPS: 60,
		BuildAvg: 3 * ms, RasterAvg: 6 * ms, BuildMax: 8 * ms, RasterMax: 30 * ms, WorstSecond: 18 * ms,
		Over16: 11, Over16Percent: 13.75, HeapMaxMB: 12, HeapLastMB: 12, SysLastMB: 31}
	if got != want {
		t.Errorf("SummarizeFrames = %+v, want %+v", got, want)
	}
	if got := SummarizeFrames(nil); got != (FrameSummary{}) {
		t.Errorf("SummarizeFrames(nil) = %+v, want the zero value", got)
	}
}

func TestProcTicks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want uint64
		ok   bool
	}{
		{"a stat line", "1350337 (counter.exe) S 1350336 1350311 1350311 0 -1 4194304 52000 0 0 0 1834 66 0 0 20 0 14 0 9999 0 0 0", 1900, true},
		{"a command with blanks and a parenthesis", "42 (a (b) c) R 1 1 1 0 -1 0 0 0 0 0 5 7 0 0 20 0 1 0 1 0 0", 12, true},
		{"no parenthesis", "42 counter S 1", 0, false},
		{"too few fields", "42 (x) S 1 1", 0, false},
		{"not numbers", "42 (x) S 1 1 1 0 -1 0 0 0 0 0 a b 0 0 20 0 1 0 1", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ProcTicks(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ProcTicks = %d, %v, want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestProcMemory(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		rss, hwm int
		ok       bool
	}{
		{"a process", "Name:\tcounter.exe\nVmHWM:\t  724008 kB\nVmRSS:\t  700000 kB\nThreads:\t9\n", 700000, 724008, true},
		{"a kernel thread has none", "Name:\tkthreadd\nThreads:\t1\n", 0, 0, false},
		{"only the peak", "VmHWM:\t5 kB\n", 0, 5, false},
		{"a value that is not a number", "VmRSS:\tmany kB\n", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rss, hwm, ok := ProcMemory(tt.in)
			if rss != tt.rss || hwm != tt.hwm || ok != tt.ok {
				t.Errorf("ProcMemory = %d, %d, %v, want %d, %d, %v", rss, hwm, ok, tt.rss, tt.hwm, tt.ok)
			}
		})
	}
}

func TestRollupPSS(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
		ok   bool
	}{
		{"a rollup", "55f0-7ffe [rollup]\nRss:              724008 kB\nPss:              681974 kB\nPss_Anon:         600000 kB\n", 681974, true},
		{"none", "Rss: 5 kB\n", 0, false},
		{"empty", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := RollupPSS(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Errorf("RollupPSS = %d, %v, want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseLogTime(t *testing.T) {
	utc := time.UTC
	tests := []struct {
		name string
		in   string
		want time.Time
		ok   bool
	}{
		{"a line of the default logger", "2026/10/26 03:28:05 INFO first frame presented since_run=2.587s", time.Date(2026, 10, 26, 3, 28, 5, 0, utc), true},
		{"a line with no time", "INFO frames fps=60", time.Time{}, false},
		{"a short line", "2026/10/26", time.Time{}, false},
		{"slog's text format", "time=2026-10-26T03:28:05.000Z level=INFO msg=frames", time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseLogTime(tt.in, utc)
			if !got.Equal(tt.want) || ok != tt.ok {
				t.Errorf("ParseLogTime = %v, %v, want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
