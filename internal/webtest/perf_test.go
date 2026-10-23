package webtest

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
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

func TestMedian(t *testing.T) {
	tests := []struct {
		name string
		in   []time.Duration
		want time.Duration
	}{
		{"none", nil, 0},
		{"one", []time.Duration{5}, 5},
		{"odd", []time.Duration{9, 1, 5}, 5},
		{"even takes the upper one", []time.Duration{4, 1, 3, 2}, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]time.Duration(nil), tc.in...)
			if got := Median(tc.in); got != tc.want {
				t.Errorf("Median = %v, want %v", got, tc.want)
			}
			for i := range before {
				if before[i] != tc.in[i] {
					t.Fatal("Median sorted its argument")
				}
			}
		})
	}
}

func TestPerfResultAddsLogs(t *testing.T) {
	var r PerfResult
	r.noteStart("2026/10/21 INFO first frame presented since_run=480ms renderer=\"webgl: x\"\n2026/10/21 INFO first frame presented since_run=300ms\n")
	if r.Renderer != "webgl: x" || len(r.SinceRun) != 2 || r.SinceRun[1] != 300*time.Millisecond {
		t.Errorf("noteStart gave %+v", r)
	}
	r.addLog("INFO slow frame build=60ms raster=12.5ms\n" +
		"INFO frames fps=60 build_avg=1ms build_max=2ms raster_avg=1ms raster_max=2ms over_16ms=0 go_heap_mb=1 go_sys_mb=2\n" +
		"INFO something else\n")
	if len(r.Slow) != 1 || r.Slow[0].Build != 60*time.Millisecond || r.Slow[0].Raster != 12500*time.Microsecond || len(r.Frames) != 1 {
		t.Errorf("addLog gave slow %v and frames %v", r.Slow, r.Frames)
	}
}

func TestPerfSummary(t *testing.T) {
	r := PerfResult{
		Renderer:    "webgpu: test",
		FirstFrames: []time.Duration{1200 * time.Millisecond, 600 * time.Millisecond, 500 * time.Millisecond, 700 * time.Millisecond},
		SinceRun:    []time.Duration{900 * time.Millisecond, 400 * time.Millisecond, 380 * time.Millisecond, 420 * time.Millisecond},
		Frames: []PerfFrames{
			{FPS: 120, BuildAvg: time.Millisecond, RasterAvg: 3 * time.Millisecond, BuildMax: 2 * time.Millisecond, RasterMax: 9 * time.Millisecond, GoHeapMB: 10, GoSysMB: 20},
			{FPS: 100, BuildAvg: 3 * time.Millisecond, RasterAvg: 3 * time.Millisecond, BuildMax: 4 * time.Millisecond, RasterMax: 5 * time.Millisecond, Over16: 2, GoHeapMB: 11, GoSysMB: 21},
		},
		RAFCalls: 100, TimerCalls: 0, CPU: 500 * time.Millisecond, Window: 10 * time.Second,
		Intervals: []time.Duration{8 * time.Millisecond, 8 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond},
		Clicks:    []PerfClick{{X: 100, Y: 218, Latency: 80 * time.Millisecond, Frames: []time.Duration{70 * time.Millisecond, 5 * time.Millisecond}, Max: 70 * time.Millisecond, Gap: 15 * time.Millisecond}},
		Slow:      []PerfSlow{{Build: 50 * time.Millisecond, Raster: 20 * time.Millisecond}},
	}
	got := map[string]string{}
	for _, line := range strings.Split(r.Summary(), "\n") {
		if k, v, ok := strings.Cut(line, " "); ok {
			if _, dup := got[k]; !dup {
				got[k] = v
			}
		}
	}
	want := map[string]string{
		"renderer": "webgpu: test", "first_frame_cold_ms": "1200.00", "first_frame_warm_ms": "600.00", "first_frame_warm_loads": "3",
		"since_run_warm_ms": "400.00", "window_s": "10.00", "cpu_ms": "500.00", "cpu_percent": "5.00", "raf_calls": "100", "timer_calls": "0",
		"frame_lines": "2", "fps": "110.00", "over_16ms": "2", "build_avg_ms": "2.00", "raster_avg_ms": "3.00", "ui_avg_ms": "5.00",
		"ui_worst_second_avg_ms": "6.00", "build_max_ms": "4.00", "raster_max_ms": "9.00", "go_heap_mb": "11", "go_sys_mb": "21",
		"raf_intervals": "4", "raf_interval_mean_ms": "19.00", "raf_interval_median_ms": "20.00", "raf_interval_max_ms": "40.00",
		"raf_intervals_over_16.7ms": "2", "raf_intervals_over_33.3ms": "1",
		"click": "0 100,218 latency_ms 80.00 max_frame_ms 70.00 max_gap_ms 15.00 frames_ms 70.0,5.0", "slow": "build_ms 50.00 raster_ms 20.00",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestInjectRAFCounter(t *testing.T) {
	page := []byte("<!doctype html>\n<html><head><title>x</title></head><body></body></html>")
	got := string(InjectRAFCounter(page))
	head := strings.Index(got, "<head>")
	base := strings.Index(got, `<base href="/">`)
	counter := strings.Index(got, "glutterRafCount")
	title := strings.Index(got, "<title>")
	if base < head || counter < base || title < counter {
		t.Errorf("the base and the counter must come first in the head, before everything the page has: head %d base %d counter %d title %d", head, base, counter, title)
	}
	for _, name := range []string{"glutterRafCount", "glutterTimerCount", "glutterIntervals", "glutterFrames"} {
		if !strings.Contains(got, name) {
			t.Errorf("the script lacks %s", name)
		}
	}
	// A page with no head still gets the script, in front.
	if got := string(InjectRAFCounter([]byte("<body></body>"))); !strings.HasPrefix(got, "<base") {
		t.Errorf("a page without a head gets %q", got)
	}
}

func TestPerfFramePage(t *testing.T) {
	page := string(PerfFramePage("/__glutter/perf-app.html?GLUTTER_FRAME_STATS=1&a=<b>", 1536, 864, 4, [][2]float64{{100, 218}, {100, 306}}, 1500*time.Millisecond))
	for _, want := range []string{"width:1536px;height:864px", "loads=4", "clicks=[[100,218],[100,306]]", "gap=1500", "/__glutter/results", "glutterTimings"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	if strings.Contains(page, "<b>") {
		t.Error("the address is not escaped")
	}
	if page := string(PerfFramePage("/x", 10, 10, 1, nil, time.Second)); !strings.Contains(page, "clicks=[]") {
		t.Error("no clicks must be an empty list")
	}
}

func TestPerfErrorRoutes(t *testing.T) {
	check := func(t *testing.T, err error, route string) {
		t.Helper()
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagWebPerfFailed || !appErr.HasRoute(route) {
			t.Fatalf("got %v, want WebPerfFailed via %s", err, route)
		}
	}
	t.Run("a build without index.html", func(t *testing.T) {
		_, err := Perf(context.Background(), PerfOptions{Dir: t.TempDir(), Browser: Chromium})
		check(t, err, "Perf.WebPerfFailed")
	})
	proc := &Process{done: make(chan struct{})}
	t.Run("times of the starts that are not a list", func(t *testing.T) {
		srv := newTestServer(t)
		srv.results <- Result{Kind: "starts", Data: json.RawMessage(`{"not":"a list"}`)}
		var res PerfResult
		check(t, awaitStarts(context.Background(), srv, proc, &res), "awaitStarts.WebPerfFailed")
	})
	t.Run("responses to the clicks that are not a list", func(t *testing.T) {
		srv := newTestServer(t)
		srv.results <- Result{Kind: "clicks", Data: json.RawMessage(`"text"`)}
		var res PerfResult
		check(t, awaitClicks(context.Background(), srv, proc, &res), "awaitClicks.WebPerfFailed")
	})
}

func TestAwaitStartsAndClicksRead(t *testing.T) {
	proc := &Process{done: make(chan struct{})}
	srv := newTestServer(t)
	srv.results <- Result{Kind: "log", Log: "INFO first frame presented since_run=480ms renderer=\"r\"\nINFO first frame presented since_run=250ms\n"}
	srv.results <- Result{Kind: "starts", Data: json.RawMessage(`[{"instantiated":300,"firstFrame":1200.5},{"instantiated":100,"firstFrame":600}]`)}
	var res PerfResult
	if err := awaitStarts(context.Background(), srv, proc, &res); err != nil {
		t.Fatal(err)
	}
	if len(res.FirstFrames) != 2 || res.FirstFrames[0] != 1200500*time.Microsecond || res.Renderer != "r" {
		t.Errorf("awaitStarts gave %+v", res)
	}
	// A log line that comes after the starts is still taken (the loader posts once a second)...
	srv.results <- Result{Kind: "log", Log: "INFO first frame presented since_run=300ms\n"}
	late := PerfResult{FirstFrames: []time.Duration{1, 2}}
	late.SinceRun = []time.Duration{time.Millisecond}
	srv.results <- Result{Kind: "starts", Data: json.RawMessage(`[]`)}
	if err := awaitStarts(context.Background(), srv, proc, &late); err != nil {
		t.Fatal(err)
	}
	if len(late.SinceRun) != 2 || late.SinceRun[1] != 300*time.Millisecond {
		t.Errorf("the late line was not taken: %v", late.SinceRun)
	}
	srv.results <- Result{Kind: "clicks", Data: json.RawMessage(`[{"x":100,"y":218,"latency":80.5,"frames":[70,5],"max":70,"gap":15}]`)}
	if err := awaitClicks(context.Background(), srv, proc, &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Clicks) != 1 || res.Clicks[0].Latency != 80500*time.Microsecond || len(res.Clicks[0].Frames) != 2 || res.Clicks[0].Gap != 15*time.Millisecond {
		t.Errorf("awaitClicks gave %+v", res.Clicks)
	}
	// The program ending is the failure of the measurement.
	srv.results <- Result{Kind: "exit", ExitCode: 2, Log: "boom\n"}
	var other PerfResult
	appErr, ok := errors.FromError(awaitStarts(context.Background(), srv, proc, &other))
	if !ok || appErr.Tag != tagWebAppFailed || !appErr.HasRoute("awaitStarts.WebAppFailed") {
		t.Errorf("an exit gave %v", appErr)
	}
	// A browser that ends, and a context that ends.
	closed := &Process{done: make(chan struct{})}
	close(closed.done)
	appErr, ok = errors.FromError(awaitStarts(context.Background(), srv, closed, &other))
	if !ok || appErr.Tag != tagBrowserExited || !appErr.HasRoute("awaitStarts.BrowserExited") {
		t.Errorf("a closed browser gave %v", appErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	appErr, ok = errors.FromError(awaitClicks(ctx, srv, proc, &other))
	if !ok || appErr.Tag != tagBrowserTimeout {
		t.Errorf("a timeout gave %v", appErr)
	}
}

func TestRAFData(t *testing.T) {
	d, ok := rafData(Result{Data: json.RawMessage(`{"count":7,"timers":2,"intervals":[8.5,9]}`)})
	if !ok || d.Count != 7 || d.Timers != 2 || len(d.Intervals) != 2 {
		t.Errorf("rafData = %+v, %v", d, ok)
	}
	if _, ok := rafData(Result{Data: json.RawMessage(`{"count":-1}`)}); ok {
		t.Error("a page that could not count must not count")
	}
	if _, ok := rafData(Result{Data: json.RawMessage(`[1]`)}); ok {
		t.Error("data that is not an object must not count")
	}
}
