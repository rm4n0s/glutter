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
