package main

import (
	"bytes"
	"context"
	stdErrors "errors"
	"strings"
	"testing"
	"time"

	rerrors "github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// sampleReport is a report as the measurement of the gallery would make it.
func sampleReport() *perfReport {
	ms := time.Millisecond
	return &perfReport{
		Package: "./examples/gallery", Release: false,
		Env:      []string{"GLUTTER_DEV_EVENTS=wait:1s;swipe:540,1800,540,600,300", "GLUTTER_GALLERY_PAGE=data"},
		Device:   deviceInfo{Serial: "emulator-5584", Model: "sdk_gphone64_x86_64", Android: "16", SDK: "37", ABI: "x86_64"},
		Renderer: "vulkan: Adreno (TM) 740",
		Sizes: []sizeRow{
			{Label: "arm64", ABIs: []string{"arm64-v8a"}, APK: 14680064, Libs: map[string]int64{"arm64-v8a": 14548992}},
			{Label: "amd64", ABIs: []string{"x86_64"}, APK: 15728640, Libs: map[string]int64{"x86_64": 15597568}},
			{Label: "arm64+amd64", ABIs: []string{"arm64-v8a", "x86_64"}, APK: 30408704, Libs: map[string]int64{"arm64-v8a": 14548992, "x86_64": 15597568}},
		},
		Starts: []startRun{
			{Total: 912 * ms, Wait: 930 * ms, LaunchState: "COLD", Frame: firstFrame{SinceRun: 640 * ms}},
			{Total: 520 * ms, Wait: 535 * ms, LaunchState: "COLD", Frame: firstFrame{SinceRun: 310 * ms}},
			{Total: 498 * ms, Wait: 512 * ms, LaunchState: "COLD", Frame: firstFrame{SinceRun: 295 * ms}},
		},
		Idle: idleResult{Window: 10020 * ms, Ticks: 12, CPUPercent: 1.1976, Frames: 0},
		Scenario: scenarioResult{Ran: true, Window: 3 * time.Second,
			Frames: []perfstats.Frames{
				{FPS: 60, BuildAvg: 2300 * time.Microsecond, BuildMax: 5 * ms, RasterAvg: 1100 * time.Microsecond, RasterMax: 2 * ms, GoHeapMB: 14, GoSysMB: 38},
				{FPS: 52, BuildAvg: 6200 * time.Microsecond, BuildMax: 24 * ms, RasterAvg: 2900 * time.Microsecond, RasterMax: 9 * ms, Over16: 6, GoHeapMB: 17, GoSysMB: 40},
				{FPS: 60, BuildAvg: 2400 * time.Microsecond, BuildMax: 4 * ms, RasterAvg: 1000 * time.Microsecond, RasterMax: 2 * ms, GoHeapMB: 15, GoSysMB: 40},
			}},
		Memory: []memSample{
			{Label: "after the settle", PSSKB: 66436, RSSKB: 103516, HasRSS: true, VmRSSKB: 143360, VmHWMKB: 151204},
			{Label: "after the idle window", PSSKB: 66500, RSSKB: 103600, HasRSS: true, VmRSSKB: 143400, VmHWMKB: 151204},
			{Label: "after the frame window", PSSKB: 81200, HasRSS: false, VmRSSKB: 171000, VmHWMKB: 175000},
		},
	}
}

func TestRenderText(t *testing.T) {
	r := sampleReport()
	r.Scenario.Summary = summarizeFrames(r.Scenario.Frames)
	var out bytes.Buffer
	if err := r.render(&out, "text"); err != nil {
		t.Fatal(err)
	}
	const want = `glutter-android perf
app:         ./examples/gallery
build:       debug
device:      emulator-5584 sdk_gphone64_x86_64, Android 16 (API 37), runs x86_64
renderer:    vulkan: Adreno (TM) 740
environment: GLUTTER_DEV_EVENTS=wait:1s;swipe:540,1800,540,600,300 GLUTTER_GALLERY_PAGE=data

APK size
  ABIs         APK bytes  APK MB  libapp.so bytes (per ABI)
  arm64        14680064   14.00   arm64-v8a 14548992
  amd64        15728640   15.00   x86_64 15597568
  arm64+amd64  30408704   29.00   arm64-v8a 14548992, x86_64 15597568

Cold start (3 starts; am start -W, and the engine's first frame)
  start      TotalTime ms  WaitTime ms  engine first frame (since_run) ms
  1 (first)  912.0         930.0        640.0
  2          520.0         535.0        310.0
  3          498.0         512.0        295.0
  min        498.0         512.0        295.0
  median     520.0         535.0        310.0
  max        912.0         930.0        640.0

Idle CPU (no input)
  window s  CPU ticks (utime+stime)  CPU % of one core  frames logged in the window
  10.0      12                       1.20               0

Frame times (GLUTTER_FRAME_STATS=1, 3 s after the first frame)
  metric                             value
  seconds with frames                3
  frames                             172
  FPS mean (min, max)                57.3 (52, 60)
  build ms avg / worst               3.51 / 24.00
  raster ms avg / worst              1.61 / 9.00
  worst second, build+raster avg ms  9.10
  frames over 16 ms                  6 (3.5%)
  Go heap MB, max / last             17 / 15
  Go sys MB, last                    40

Frame times per second
  second  FPS  build avg ms  build max ms  raster avg ms  raster max ms  over 16 ms  Go heap MB
  1       60   2.30          5.00          1.10           2.00           0           14
  2       52   6.20          24.00         2.90           9.00           6           17
  3       60   2.40          4.00          1.00           2.00           0           15

Memory (MB)
  when                    dumpsys TOTAL PSS  dumpsys TOTAL RSS  VmRSS  VmHWM
  after the settle        64.9               101.1              140.0  147.7
  after the idle window   64.9               101.2              140.0  147.7
  after the frame window  79.3               n/a                167.0  170.9
`
	if got := out.String(); got != want {
		t.Errorf("text report:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderMarkdown(t *testing.T) {
	r := sampleReport()
	r.Scenario.Summary = summarizeFrames(r.Scenario.Frames)
	var out bytes.Buffer
	if err := r.render(&out, "markdown"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"### glutter-android perf\n",
		"- **app**: ./examples/gallery\n",
		"#### APK size\n\n| ABIs | APK bytes | APK MB | libapp.so bytes (per ABI) |\n|---|---|---|---|\n| arm64 | 14680064 | 14.00 | arm64-v8a 14548992 |\n",
		"| median | 520.0 | 535.0 | 310.0 |\n",
		"| 10.0 | 12 | 1.20 | 0 |\n",
		"| FPS mean (min, max) | 57.3 (52, 60) |\n",
		"| after the frame window | 79.3 | n/a | 167.0 | 170.9 |\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("markdown report has no %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\t") {
		t.Errorf("markdown report has tabs:\n%s", got)
	}
}

func TestRenderWithoutFrames(t *testing.T) {
	r := sampleReport()
	r.Scenario = scenarioResult{Ran: true, Window: 10 * time.Second}
	var out bytes.Buffer
	if err := r.render(&out, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "none logged: the app drew no frame in the window") {
		t.Errorf("a window with no frames:\n%s", out.String())
	}
	r.Scenario = scenarioResult{}
	out.Reset()
	if err := r.render(&out, "text"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Frame times") {
		t.Errorf("a skipped frame window is reported:\n%s", out.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, stdErrors.New("disk full") }

func TestRenderWriteFailure(t *testing.T) {
	err := sampleReport().render(failingWriter{}, "text")
	appErr, ok := rerrors.FromError(err)
	if !ok || !appErr.HasRoute("perfReport.render."+tagAndroidPerfWriteFailed) {
		t.Errorf("got %v", err)
	}
}

func TestRunPerfNeedsNoDeviceForUsageErrors(t *testing.T) {
	// A bad flag fails before any device or build is touched.
	err := run(context.Background(), []string{"perf", "-nope", "./examples/counter"}, &bytes.Buffer{}, &bytes.Buffer{})
	appErr, ok := rerrors.FromError(err)
	if !ok || !appErr.HasRoute("run->runPerf->parse."+tagAndroidUsage) {
		t.Errorf("got %v", err)
	}
}
