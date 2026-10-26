package main

import (
	"bytes"
	"context"
	stdErrors "errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rerrors "github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
	"github.com/rm4n0s/glutter/internal/wintool"
)

func TestPerfUsageErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		route string
		in    string
	}{
		{name: "no package", args: []string{"perf"}, route: "run->runPerf->parse." + tagWindowsUsage, in: "one package"},
		{name: "a bad format", args: []string{"perf", "-format", "html", "./examples/counter"}, route: "run->runPerf." + tagWindowsPerfBadFormat, in: "text or markdown"},
		{name: "zero starts", args: []string{"perf", "-starts", "0", "./examples/counter"}, route: "run->runPerf." + tagWindowsPerfBadCount, in: "at least 1"},
		{name: "zero seconds", args: []string{"perf", "-seconds", "0", "./examples/counter"}, route: "run->runPerf." + tagWindowsPerfBadCount},
		{name: "an env that is not GLUTTER_", args: []string{"perf", "-env", "HOME=/x", "./examples/counter"}, route: "run->runPerf->parseEnvs." + tagWindowsUsage},
		{name: "dev events need a debug build", args: []string{"perf", "-env", "GLUTTER_DEV_EVENTS=wait:1s", "./examples/counter"}, route: "run->runPerf." + tagWindowsPerfDevEventsRelease, in: "-release=false"},
		{name: "amd64 is the architecture that runs", args: []string{"perf", "-arch", "arm64", "./examples/counter"}, route: "run->runPerf." + tagWindowsPerfBadArch, in: "amd64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args, io.Discard, io.Discard)
			appErr, ok := rerrors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
			if !strings.Contains(appErr.Message, tt.in) {
				t.Errorf("message %q has no %q", appErr.Message, tt.in)
			}
		})
	}
}

func TestPerfNeedsWine(t *testing.T) {
	if _, err := wineOf(wintool.LocalLauncher{}); err == nil {
		t.Fatal("wineOf accepted the launcher that runs the .exe itself")
	} else if appErr, ok := rerrors.FromError(err); !ok || !appErr.HasRoute("wineOf."+tagWindowsPerfUnsupported) || !strings.Contains(appErr.Message, "Task Manager") {
		t.Errorf("got %v, want WindowsPerfUnsupported from wineOf", err)
	}
	if w, err := wineOf(wintool.WineLauncher{Headless: true}); err != nil || !w.Headless {
		t.Errorf("wineOf(WineLauncher) = %+v, %v", w, err)
	}
}

func TestPerfTempDirFailure(t *testing.T) {
	// A temporary directory that does not exist: the measurement cannot even keep its screenshots.
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "nothing", "here"))
	err := run(context.Background(), []string{"perf", "./examples/counter"}, io.Discard, io.Discard)
	appErr, ok := rerrors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("run->runPerf." + tagWindowsPerfTempDir) {
		t.Errorf("route %s, want run->runPerf.%s", appErr.Route(), tagWindowsPerfTempDir)
	}
}

func TestClearShaderCache(t *testing.T) {
	prefix := t.TempDir()
	cache := filepath.Join(prefix, "drive_c", "users", "alice", "AppData", "Local", "glutter", "shaders")
	other := filepath.Join(prefix, "drive_c", "users", "alice", "AppData", "Local", "glutter", "fonts")
	for _, d := range []string{cache, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "x.bin"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := clearShaderCache(prefix); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Errorf("the shader cache is still there: %v", err)
	}
	if _, err := os.Stat(filepath.Join(other, "x.bin")); err != nil {
		t.Errorf("the fonts cache was removed: %v", err)
	}
	// No cache at all is no failure.
	if err := clearShaderCache(t.TempDir()); err != nil {
		t.Errorf("a prefix with no cache: %v", err)
	}
}

func TestClearShaderCacheFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can remove anything")
	}
	prefix := t.TempDir()
	parent := filepath.Join(prefix, "drive_c", "users", "alice", "AppData", "Local", "glutter")
	cache := filepath.Join(parent, "shaders")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "x.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) }) // so that the temporary directory can be removed
	err := clearShaderCache(prefix)
	appErr, ok := rerrors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("clearShaderCache." + tagWindowsPerfCacheClear) {
		t.Errorf("route %s, want clearShaderCache.%s", appErr.Route(), tagWindowsPerfCacheClear)
	}
	if appErr.Metadata["dir"] != cache {
		t.Errorf("metadata %v, want dir %s", appErr.Metadata, cache)
	}
}

// failingWriter fails every write, as a closed pipe does.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, stdErrors.New("closed pipe") }

func TestWriteReport(t *testing.T) {
	rep := &perfReport{Package: "./examples/counter", Sizes: []sizeRow{{Arch: "amd64", Bytes: 1 << 20}}}
	dir := t.TempDir()
	t.Run("to the standard output", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeReport(rep, "text", "", &out, io.Discard); err != nil || !strings.Contains(out.String(), "glutter-windows perf") {
			t.Errorf("writeReport = %v, output %q", err, out.String())
		}
	})
	t.Run("to a file", func(t *testing.T) {
		file := filepath.Join(dir, "sub", "report.md")
		var msg bytes.Buffer
		if err := writeReport(rep, "markdown", file, io.Discard, &msg); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(file)
		if err != nil || !strings.Contains(string(got), "### glutter-windows perf") || !strings.Contains(msg.String(), "wrote "+file) {
			t.Errorf("file %q (%v), message %q", got, err, msg.String())
		}
	})
	failures := []struct {
		name  string
		out   string
		w     io.Writer
		route string
	}{
		{name: "a closed standard output", w: failingWriter{}, route: "writeReport." + tagWindowsPerfWriteFailed},
		{name: "a directory that cannot be made", out: filepath.Join(dir, "sub", "report.md", "x", "y.md"), w: io.Discard, route: "writeReport." + tagWindowsPerfWriteFailed},
		{name: "a file that cannot be written", out: dir, w: io.Discard, route: "writeReport." + tagWindowsPerfWriteFailed},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			err := writeReport(rep, "text", tt.out, tt.w, io.Discard)
			appErr, ok := rerrors.FromError(err)
			if !ok || !appErr.HasRoute(tt.route) {
				t.Errorf("got %v, want %s", err, tt.route)
			}
		})
	}
}

func TestRenderFailure(t *testing.T) {
	err := (&perfReport{}).render(failingWriter{}, "text")
	appErr, ok := rerrors.FromError(err)
	if !ok || !appErr.HasRoute("perfReport.render->Doc.Render."+perfstats.TagPerfWriteFailed) {
		t.Errorf("got %v (%v), want PerfWriteFailed from perfReport.render->Doc.Render", err, appErr)
	}
}

func sampleReport() *perfReport {
	ms := time.Millisecond
	return &perfReport{
		Package: "./examples/gallery", Release: true,
		Env:      []string{"GLUTTER_GALLERY_ANIMATE=1", "GLUTTER_RENDERER=d3d12"},
		Renderer: "d3d12: NVIDIA GeForce GTX 470 (10DE:06CD)",
		Sizes:    []sizeRow{{Arch: "amd64", Bytes: 38797312}, {Arch: "arm64", Bytes: 36700160}},
		Starts: []startRun{
			{Cache: cacheEmpty, Wall: 7 * time.Second, Frame: firstFrame{SinceRun: 5200 * ms}, Stages: startupStages{Device: 160 * ms, Shaders: 900 * ms, Rest: 2200 * ms, Built: 10, HasShaders: true}},
			{Cache: cacheEmpty, Wall: 6 * time.Second, Frame: firstFrame{SinceRun: 4800 * ms}, Stages: startupStages{Device: 150 * ms, Shaders: 800 * ms, Rest: 2100 * ms, Built: 10, HasShaders: true}},
			{Cache: cacheWarm, Wall: 4 * time.Second, Frame: firstFrame{SinceRun: 2500 * ms}, Stages: startupStages{Device: 150 * ms, Shaders: 12 * ms, Rest: 2100 * ms, FromCache: 10, HasShaders: true}},
		},
		Idle: idleResult{Window: 10 * time.Second, Frames: 0, Procs: []procCPU{
			{PID: 20, Name: "counter.exe", IsApp: true, Ticks: 12, Percent: 1.2}, {PID: 10, Name: "wineserver", Ticks: 8, Percent: 0.8}}},
		Scenario: scenarioResult{Ran: true, Window: 2 * time.Second, Frames: []perfstats.Frames{
			{FPS: 60, BuildAvg: 2 * ms, BuildMax: 5 * ms, RasterAvg: 4 * ms, RasterMax: 9 * ms, Over16: 1, GoHeapMB: 10, GoSysMB: 40},
			{FPS: 58, BuildAvg: 3 * ms, BuildMax: 6 * ms, RasterAvg: 5 * ms, RasterMax: 12 * ms, Over16: 2, GoHeapMB: 11, GoSysMB: 40}}},
		Memory: []memSample{
			{Label: "after the settle", Procs: []procMem{{PID: 20, Name: "counter.exe", IsApp: true, RSSKB: 700000, HWMKB: 724000, PSSKB: 680000, HasPSS: true}, {PID: 10, Name: "wineserver", RSSKB: 9000, HWMKB: 9100, PSSKB: 7000, HasPSS: true}}},
			{Label: "after the frame window", Procs: []procMem{{PID: 20, Name: "gallery.exe", IsApp: true, RSSKB: 900000, HWMKB: 910000}, {PID: 10, Name: "wineserver", RSSKB: 9000, HWMKB: 9100}}},
		},
	}
}

func TestRenderTextAndMarkdown(t *testing.T) {
	r := sampleReport()
	r.Idle.sum()
	r.Scenario.Summary = perfstats.SummarizeFrames(r.Scenario.Frames)
	tests := []struct {
		format string
		want   []string
	}{
		{"text", []string{
			"glutter-windows perf\n", "app:         ./examples/gallery\n", "build:       release (-tags glutter_release, stripped)\n",
			"renderer:    d3d12: NVIDIA GeForce GTX 470 (10DE:06CD)\n", ".exe size\n", "amd64", "37.00", "35.00",
			"Cold start", "empty", "warm", "10 / 0", "0 / 10", "median", "Idle CPU (no input", "1.20", "2.00", "counter.exe (the app)",
			"Frame times (GLUTTER_FRAME_STATS=1, 2 s after the first frame)", "FPS mean (min, max)", "59.0 (58, 60)", "Frame times per second",
			"Memory (MB; PSS", "after the settle", "after the frame window", "n/a",
		}},
		{"markdown", []string{
			"### glutter-windows perf\n", "- **app**: ./examples/gallery\n", "#### .exe size\n", "| architecture | bytes | MiB |\n", "|---|---|---|\n",
			"| amd64 | 38797312 | 37.00 |\n", "#### Idle CPU (no input", "#### Frame times per second\n", "#### Memory of each Wine process after the frame window (MB)\n",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var out bytes.Buffer
			if err := r.render(&out, tt.format); err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(out.String(), w) {
					t.Errorf("the %s report has no %q:\n%s", tt.format, w, out.String())
				}
			}
		})
	}
}

func TestIdleResultSum(t *testing.T) {
	r := idleResult{Procs: []procCPU{{Ticks: 12, Percent: 1.2, IsApp: true}, {Ticks: 8, Percent: 0.8}, {Ticks: 5, Percent: 0.5}}}
	r.sum()
	if r.AppTicks != 12 || r.WineTicks != 25 || r.AppPercent != 1.2 || r.WinePercent < 2.49 || r.WinePercent > 2.51 {
		t.Errorf("sum = %+v", r)
	}
}

func TestMemSampleTotals(t *testing.T) {
	s := memSample{Procs: []procMem{{IsApp: true, RSSKB: 5, HWMKB: 6, PSSKB: 4, HasPSS: true}, {RSSKB: 3, HWMKB: 3, PSSKB: 2, HasPSS: true}}}
	app, wine := s.totals()
	if app.RSSKB != 5 || wine.RSSKB != 8 || wine.PSSKB != 6 || wine.HWMKB != 9 || !wine.HasPSS {
		t.Errorf("totals = %+v, %+v", app, wine)
	}
}
