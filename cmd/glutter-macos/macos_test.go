package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

func TestParsePSTime(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"0:01.23", 1230 * time.Millisecond, true},
		{"12:34.56", 12*time.Minute + 34560*time.Millisecond, true},
		{"1:02:03.45", time.Hour + 2*time.Minute + 3450*time.Millisecond, true},
		{"2-01:00:00.00", 49 * time.Hour, true},
		{" 0:00.00 ", 0, true},
		{"", 0, false},
		{"12", 0, false},
		{"a:b", 0, false},
		{"1:2:3:4", 0, false},
		{"x-1:00.00", 0, false},
		{"-1:00.00", 0, false},
		{"1:-5.0", 0, false},
	}
	for _, tt := range tests {
		got, ok := parsePSTime(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("parsePSTime(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParsePSAndCPUPercent(t *testing.T) {
	a, ok := parsePS("  48320   0:01.50\n")
	if !ok || a.RSSKB != 48320 || a.CPU != 1500*time.Millisecond {
		t.Fatalf("parsePS = %+v, %v", a, ok)
	}
	b, _ := parsePS("48400 0:02.00")
	if got := cpuPercent(a, b, 10*time.Second); got < 4.99 || got > 5.01 {
		t.Errorf("cpuPercent = %v", got)
	}
	if cpuPercent(b, a, time.Second) != 0 || cpuPercent(a, b, 0) != 0 {
		t.Error("a time that goes back or an empty window is not a percentage")
	}
	for _, bad := range []string{"", "1", "a b", "1 2 3", "-5 0:01.00", "5 nonsense"} {
		if _, ok := parsePS(bad); ok {
			t.Errorf("parsePS(%q) was accepted", bad)
		}
	}
}

const engineLog = `time=2026-11-09T10:00:00.000+01:00 level=INFO msg="first frame presented" since_run=212ms renderer="metal: Apple M6" since_process_start=240ms go_heap_mb=11 go_sys_mb=29
time=2026-11-09T10:00:01.000+01:00 level=INFO msg=frames renderer="metal: Apple M6" fps=60 build_avg=300µs build_max=2.1ms raster_avg=1.20ms raster_max=4ms over_16ms=1 go_heap_mb=12 go_sys_mb=30
time=2026-11-09T10:00:02.000+01:00 level=INFO msg=frames renderer="metal: Apple M6" fps=58 build_avg=310µs build_max=3ms raster_avg=1.30ms raster_max=5ms over_16ms=0 go_heap_mb=13 go_sys_mb=30
`

func TestParseFirstFrameAndFrames(t *testing.T) {
	ff, ok := parseFirstFrame(engineLog)
	if !ok || ff.SinceRun != 212*time.Millisecond || ff.SinceProcess != 240*time.Millisecond || ff.Renderer != "metal: Apple M6" || ff.GoHeapMB != 11 || ff.GoSysMB != 29 {
		t.Fatalf("first frame = %+v, %v", ff, ok)
	}
	if _, ok := parseFirstFrame("nothing here\nfirst frame presented but with no time\n"); ok {
		t.Error("a line with no since_run was accepted")
	}
	fr := framesOf(engineLog)
	if len(fr) != 2 || fr[0].FPS != 60 || fr[1].FPS != 58 || fr[0].Over16 != 1 {
		t.Errorf("frames = %+v", fr)
	}
	// Lines before the first frame are not counted.
	if got := framesOf(strings.Split(engineLog, "\n")[1] + "\n" + engineLog); len(got) != 2 {
		t.Errorf("frames before the first frame were counted: %d", len(got))
	}
}

func TestParsePerf(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		check   func(o perfOptions) bool
		wantTag string
	}{
		{"defaults", []string{"./x"}, func(o perfOptions) bool {
			return o.pkg == "./x" && o.starts == 5 && o.seconds == 10 && o.settle == 5*time.Second && !o.bf.debug && o.format == "text"
		}, ""},
		{"debug with a script", []string{"-debug", "-env", "GLUTTER_DEV_EVENTS=wait:1s", "-starts", "2", "-format", "markdown", "./x"},
			func(o perfOptions) bool {
				return o.bf.debug && o.starts == 2 && o.format == "markdown" && hasEnv(o.env, "GLUTTER_DEV_EVENTS")
			}, ""},
		{"a script in a release build", []string{"-env", "GLUTTER_DEV_EVENTS=wait:1s", "./x"}, nil, tagMacOSUsage},
		{"a bad format", []string{"-format", "html", "./x"}, nil, tagMacOSUsage},
		{"no starts", []string{"-starts", "0", "./x"}, nil, tagMacOSUsage},
		{"a negative settle", []string{"-settle", "-1s", "./x"}, nil, tagMacOSUsage},
		{"no package", nil, nil, tagMacOSUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := parsePerf(tt.args, io.Discard)
			if tt.wantTag != "" {
				if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tt.wantTag {
					t.Fatalf("err = %v, want tag %s", err, tt.wantTag)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !tt.check(o) {
				t.Errorf("options = %+v", o)
			}
		})
	}
}

func TestPerfEnvLetsEnvFlagsWin(t *testing.T) {
	o := perfOptions{renderer: "soft", env: []string{"GLUTTER_RENDERER=metal"}}
	got := perfEnv([]string{"HOME=/h"}, o, "/cache")
	want := []string{"HOME=/h", "GLUTTER_FRAME_STATS=1", "XDG_CACHE_HOME=/cache", "GLUTTER_MACOS_ACCESSORY=1", "GLUTTER_DISPLAYLINK=cv", "GLUTTER_RENDERER=soft", "GLUTTER_RENDERER=metal"}
	if !slices.Equal(got, want) {
		t.Errorf("env = %v, want %v", got, want)
	}
	if front := perfEnv(nil, perfOptions{front: true}, "/c"); slices.Contains(front, "GLUTTER_MACOS_ACCESSORY=1") {
		t.Error("-front still makes an accessory app")
	}
}

func TestPerfReportDocument(t *testing.T) {
	r := perfReport{Package: "./examples/gallery", Release: true, Machine: "macOS 27.2, arm64", Renderer: "metal: Apple M6", Env: []string{"GLUTTER_X=1"},
		Archs: []string{"arm64", "amd64"}, Size: 20 << 20, Thin: map[string]int64{"arm64": 9 << 20, "amd64": 10 << 20},
		Empty: []startResult{{First: firstFrame{SinceRun: 300 * time.Millisecond, SinceProcess: 330 * time.Millisecond}, Wall: 500 * time.Millisecond}},
		Warm: []startResult{{First: firstFrame{SinceRun: 200 * time.Millisecond}, Wall: 400 * time.Millisecond},
			{First: firstFrame{SinceRun: 220 * time.Millisecond}, Wall: 420 * time.Millisecond}},
		Window: windowResult{Seconds: 10, CPU: 0.5, RSSKB: 100 * 1024, Count: 2, Frames: perfstats.SummarizeFrames(framesOf(engineLog))}}
	var b strings.Builder
	if err := r.document().Render(&b, "glutter-macos perf", "markdown"); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	for _, want := range []string{"### glutter-macos perf", "macOS 27.2, arm64", "metal: Apple M6", "arm64+amd64", "slice arm64", "empty", "warm", "100.0", "Frames", "59.0, 58, 60"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report has no %q:\n%s", want, got)
		}
	}
	// Without frames the table is left out; a thin build has no slice rows.
	r.Window.Count, r.Thin, r.Archs = 0, map[string]int64{"arm64": 9 << 20}, []string{"arm64"}
	b.Reset()
	if err := r.document().Render(&b, "glutter-macos perf", "text"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "Frames (the seconds") || strings.Contains(b.String(), "slice arm64") {
		t.Errorf("the report has what it should not:\n%s", b.String())
	}
}

func TestParseBuildFlags(t *testing.T) {
	var bf buildFlags
	fs := newFlagSetForTest(&bf)
	pkg, err := parseWith(fs, []string{"-name", "Demo", "-id", "dev.x.demo", "-universal", "-hardened", "-debug", "-usage", "NSCameraUsageDescription=For pictures", "-o", "Out.app", "./x"}, io.Discard)
	if err != nil || pkg != "./x" {
		t.Fatalf("parse: %q, %v", pkg, err)
	}
	o, err := bf.options(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if o.Name != "Demo" || o.ID != "dev.x.demo" || !slices.Equal(o.Archs, []string{"arm64", "amd64"}) || !o.Hardened || !o.Debug || o.Out != "Out.app" ||
		o.Usage["NSCameraUsageDescription"] != "For pictures" {
		t.Errorf("options = %+v", o)
	}
	bf.usage = envFlags{"=nothing"}
	if _, err := bf.options("./x"); err == nil {
		t.Error("a usage string with no key was accepted")
	}
	// A command takes exactly one package.
	if _, err := parseWith(newFlagSetForTest(&buildFlags{}), []string{"./a", "./b"}, io.Discard); err == nil {
		t.Error("two packages were accepted")
	}
}

func TestUnknownCommandsAndHelp(t *testing.T) {
	var out strings.Builder
	if err := run(t.Context(), []string{"help"}, &out, io.Discard); err != nil || !strings.Contains(out.String(), "glutter-macos build") {
		t.Errorf("help: %v %q", err, out.String())
	}
	for _, args := range [][]string{nil, {"nope"}} {
		if err := run(t.Context(), args, io.Discard, io.Discard); err == nil {
			t.Errorf("run(%v) did not fail", args)
		}
	}
}

// tinyProgram writes a module whose program does what body says, and returns its directory.
func tinyProgram(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{"go.mod": "module tiny\n\ngo 1.21\n", "main.go": "package main\n\nimport \"os\"\n\nvar _ = os.Exit\n\n" + body} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func routeIs(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("error = %T %v", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("route = %s, want %s", appErr.Route(), route)
	}
}

func TestBuildMakesABundleAndSaysWhere(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("signs the bundle: a Mac only")
	}
	dir := tinyProgram(t, "func main() { println(\"hi\") }\n")
	app := filepath.Join(t.TempDir(), "Tiny.app")
	t.Chdir(dir)
	var out strings.Builder
	if err := run(t.Context(), []string{"build", "-o", app, "-name", "Tiny", "."}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), app) || !strings.Contains(out.String(), "Tiny 1.0.0 (dev.glutter.tiny)") {
		t.Errorf("build said %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(app, "Contents", "MacOS", "Tiny")); err != nil {
		t.Error(err)
	}
	// A package that does not build is MacOSBuildFailed.
	bad := tinyProgram(t, "func main() { nothing() }\n")
	t.Chdir(bad)
	routeIs(t, run(t.Context(), []string{"build", "-o", filepath.Join(t.TempDir(), "Bad.app"), "."}, io.Discard, io.Discard), "run->runBuild->Build->BuildOptions.compile.MacOSBuildFailed")
}

func TestRunStartsTheBundleAndReportsItsFailure(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("signs and starts the bundle: a Mac only")
	}
	ok := tinyProgram(t, "func main() { println(os.Getenv(\"TINY_SAYS\")) }\n")
	t.Chdir(ok)
	var out, errOut strings.Builder
	if err := run(t.Context(), []string{"run", "-env", "TINY_SAYS=hello from the bundle", "."}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String()+errOut.String(), "hello from the bundle") {
		t.Errorf("the program's output was %q %q", out.String(), errOut.String())
	}
	failing := tinyProgram(t, "func main() { os.Exit(3) }\n")
	t.Chdir(failing)
	routeIs(t, run(t.Context(), []string{"run", "."}, io.Discard, io.Discard), "run->runRun.MacOSRunFailed")
}

func TestPerfReportsAnAppThatNeverDrawsAFrame(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("signs and starts the bundle: a Mac only")
	}
	dir := tinyProgram(t, "func main() {}\n")
	t.Chdir(dir)
	err := run(t.Context(), []string{"perf", "-starts", "1", "-seconds", "1", "-settle", "0s", "."}, io.Discard, io.Discard)
	routeIs(t, err, "run->runPerf->oneStart.MacOSPerfFailed")
}

func TestShotWithABundleReportsAnAppThatSavesNothing(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("signs and starts the bundle: a Mac only")
	}
	dir := tinyProgram(t, "func main() {}\n")
	t.Chdir(dir)
	err := run(t.Context(), []string{"shot", "-bundle", "-timeout", "30s", "-o", filepath.Join(t.TempDir(), "x.png"), "."}, io.Discard, io.Discard)
	routeIs(t, err, "run->runShot.MacOSShotFailed")
}
