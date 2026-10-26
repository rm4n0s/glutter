package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// fakeWine is a Wine on a clock that only the sleeps of the measurement advance. It is the launcher
// (the app on it draws its first frame three seconds after its start, logs a frames line a second when
// it was started with GLUTTER_GALLERY_ANIMATE, and its process uses idleTicks clock ticks a second) and
// the list of the processes of the prefix.
type fakeWine struct {
	mu  sync.Mutex
	now time.Time

	startedAt time.Time
	running   bool // a run in the background is on
	exe       string

	idleTicks uint64
	baseTicks uint64
	cleared   int
	cacheWarm bool          // a start since the cache was emptied filled it
	endRun    chan struct{} // closed by a test to end the run in the background by itself
	launches  []winetest.RunOptions
	// the faults
	exit    int  // the exit code of a cold start
	noFrame bool // the app logs no first frame
	// noFrameBackground leaves out the first frame of the runs in the background only.
	noFrameBackground bool
	noProcess         bool   // the app's process never shows
	dies              bool   // the background run ends by itself soon after it began
	launchTag         string // the tag of the error the launcher fails with
}

func newFakeWine() *fakeWine {
	return &fakeWine{now: time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC), idleTicks: 2, baseTicks: 1800, endRun: make(chan struct{})}
}

func (f *fakeWine) sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	time.Sleep(100 * time.Microsecond) // lets the run in the background start, as real time would
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
	return nil
}

func (f *fakeWine) clock() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// at is a line's time: the fake clock plus d.
func (f *fakeWine) stamp(t time.Time) string { return t.Format("2006/01/02 15:04:05") }

// startupLines are the lines of a start: the stages differ with the cache the start found.
func (f *fakeWine) startupLines(warm bool) string {
	var b strings.Builder
	start := f.stamp(f.startedAt)
	if warm {
		fmt.Fprintf(&b, "%s INFO startup stage=\"GPU context or device\" renderer=d3d12 took=150ms\n", start)
		fmt.Fprintf(&b, "%s INFO startup stage=shaders renderer=d3d12 took=12.5ms built=0 from_cache=10 refused=0\n", start)
		fmt.Fprintf(&b, "%s INFO startup stage=\"frame without the above\" renderer=d3d12 took=2.1s\n", start)
	} else {
		fmt.Fprintf(&b, "%s INFO startup stage=\"GPU context or device\" renderer=d3d12 took=160ms\n", start)
		fmt.Fprintf(&b, "%s INFO startup stage=shaders renderer=d3d12 took=900ms built=10 from_cache=0 refused=0\n", start)
		fmt.Fprintf(&b, "%s INFO startup stage=\"frame without the above\" renderer=d3d12 took=2.2s\n", start)
	}
	return b.String()
}

func (f *fakeWine) Launch(ctx context.Context, exe string, o winetest.RunOptions) (winetest.Result, error) {
	f.mu.Lock()
	f.launches = append(f.launches, o)
	f.exe = exe
	launchTag := f.launchTag
	f.mu.Unlock()
	if launchTag != "" {
		return winetest.Result{}, errors.New(launchTag, "the launcher failed")
	}
	env := strings.Join(o.Env, "\n")
	if strings.Contains(env, "GLUTTER_SCREENSHOT=") {
		// A cold start: the app runs to its first frame and ends.
		f.mu.Lock()
		f.startedAt = f.now
		warm := f.cacheWarm
		f.cacheWarm = true
		f.now = f.now.Add(3 * time.Second)
		f.mu.Unlock()
		out := f.startupLines(warm)
		if !f.noFrame {
			out += f.stamp(f.startedAt.Add(3*time.Second)) + ` INFO first frame presented since_run=2.5s renderer="d3d12: Fake GPU" go_heap_mb=8 go_sys_mb=38` + "\n"
		}
		return winetest.Result{ExitCode: f.exit, Output: out}, nil
	}
	// A run in the background: the process is there until the context ends.
	f.mu.Lock()
	f.startedAt, f.running = f.now, true
	f.mu.Unlock()
	if f.dies {
		f.mu.Lock()
		f.running = false
		f.mu.Unlock()
		return winetest.Result{ExitCode: 1, Output: "panic: boom\n"}, nil
	}
	select {
	case <-ctx.Done():
	case <-f.endRun:
		f.mu.Lock()
		f.running = false
		f.mu.Unlock()
		return winetest.Result{ExitCode: 1, Output: "panic: boom\n"}, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.running = false
	var b strings.Builder
	if !f.noFrame && !f.noFrameBackground {
		first := f.startedAt.Add(3 * time.Second)
		b.WriteString(f.stamp(first) + ` INFO first frame presented since_run=2.5s renderer="d3d12: Fake GPU" go_heap_mb=8 go_sys_mb=38` + "\n")
		if strings.Contains(env, "GLUTTER_GALLERY_ANIMATE=1") {
			for t := first.Add(time.Second); !t.After(f.now); t = t.Add(time.Second) {
				b.WriteString(f.stamp(t) + " INFO frames renderer=d3d12 fps=60 build_avg=2ms build_max=5ms raster_avg=4ms raster_max=9ms over_16ms=1 go_heap_mb=10 go_sys_mb=40\n")
			}
		}
	}
	return winetest.Result{Output: b.String()}, errors.New(winetest.TagWineStartFailed, "the run was ended")
}

func (f *fakeWine) ProgramPath(hostPath string) string { return "Z:" + hostPath }

// Sample is the list of Wine processes: the server always, the app while it runs.
func (f *fakeWine) Sample() ([]procSample, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	secs := uint64(f.now.Sub(f.startedAt) / time.Second)
	out := []procSample{{PID: 10, Name: "wineserver", Cmd: "/usr/bin/wineserver", Ticks: 300 + secs/2, RSSKB: 9000, HWMKB: 9100, PSSKB: 7000, HasPSS: true}}
	if f.running && !f.noProcess {
		out = append(out, procSample{PID: 20, Name: "counter.exe", Cmd: f.exe, Ticks: f.baseTicks + f.idleTicks*secs,
			RSSKB: 700000, HWMKB: 724000, PSSKB: 680000, HasPSS: true})
	}
	return out, nil
}

// clearCache empties the fake's shader cache: the next start compiles the shaders.
func (f *fakeWine) clearCache() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleared++
	f.cacheWarm = false
	return nil
}

func (f *fakeWine) deps(t *testing.T, sizes map[string]int64) perfDeps {
	return perfDeps{
		launcher: f, procs: f, dir: t.TempDir(), loc: time.UTC, now: f.clock, sleep: f.sleep, clearCache: f.clearCache,
		build: func(_ context.Context, o wintool.BuildOptions) (*wintool.BuildResult, error) {
			res := &wintool.BuildResult{AppID: "org.glutter.counter"}
			for _, a := range o.Archs {
				res.Exes = append(res.Exes, wintool.Built{Arch: a, Path: filepath.Join(filepath.Dir(o.Out), "counter_"+a+".exe"), Size: sizes[a]})
			}
			return res, nil
		},
	}
}

func testOptions() perfOptions {
	return perfOptions{pkg: "./examples/counter", build: wintool.BuildOptions{Release: true}, archs: []string{"amd64", "arm64"},
		extras: []string{"GLUTTER_RENDERER=d3d12"}, starts: 2, settle: 5 * time.Second, seconds: 10,
		timeouts: perfTimeouts{process: time.Minute, poll: 250 * time.Millisecond, startup: 6 * time.Second, run: time.Minute}}
}

var sizes = map[string]int64{"amd64": 38797312, "arm64": 36700160}

func TestMeasurePerfOfTheCounter(t *testing.T) {
	f := newFakeWine()
	var progress bytes.Buffer
	d := f.deps(t, sizes)
	d.progress = &progress
	rep, err := measurePerf(context.Background(), d, testOptions())
	if err != nil {
		t.Fatalf("measurePerf: %v", err)
	}
	if got := rep.Sizes; len(got) != 2 || got[0] != (sizeRow{"amd64", 38797312}) || got[1] != (sizeRow{"arm64", 36700160}) {
		t.Errorf("Sizes = %+v", got)
	}
	if rep.Renderer != "d3d12: Fake GPU" {
		t.Errorf("Renderer = %q", rep.Renderer)
	}
	if f.cleared != 2 {
		t.Errorf("the shader cache was emptied %d times, want before each of the 2 empty starts", f.cleared)
	}
	// Two starts with the cache emptied, two after them.
	if len(rep.Starts) != 4 {
		t.Fatalf("%d cold starts, want 4", len(rep.Starts))
	}
	for i, s := range rep.Starts {
		want := cacheEmpty
		if i >= 2 {
			want = cacheWarm
		}
		if s.Cache != want {
			t.Errorf("start %d is %s, want %s", i, s.Cache, want)
		}
		if s.Frame.SinceRun != 2500*time.Millisecond || s.Wall != 3*time.Second {
			t.Errorf("start %d: first frame %v, wall %v", i, s.Frame.SinceRun, s.Wall)
		}
	}
	// The idle window: the app uses 2 ticks a second for 10 s, the server 1 tick every 2 seconds.
	idle := rep.Idle
	if idle.Window != 10*time.Second || idle.AppTicks != 20 || idle.WineTicks != 25 || idle.Frames != 0 {
		t.Errorf("Idle = %+v", idle)
	}
	if idle.AppPercent < 1.99 || idle.AppPercent > 2.01 || idle.WinePercent < 2.49 || idle.WinePercent > 2.51 {
		t.Errorf("idle CPU %.2f%% (app), %.2f%% (Wine), want 2 and 2.5", idle.AppPercent, idle.WinePercent)
	}
	if len(rep.Memory) != 2 || rep.Memory[0].Label != "after the settle" || rep.Memory[1].Label != "after the idle window" {
		t.Fatalf("Memory = %+v", rep.Memory)
	}
	app, wine := rep.Memory[1].totals()
	if app.RSSKB != 700000 || app.HWMKB != 724000 || app.PSSKB != 680000 || wine.PSSKB != 687000 {
		t.Errorf("memory: app %+v, Wine %+v", app, wine)
	}
	if rep.Scenario.Ran {
		t.Error("the frame window ran, but frames was not set")
	}
	// The cold starts use the self-capture and the statistics, the idle run only the statistics.
	var shots, stats int
	for _, o := range f.launches {
		env := strings.Join(o.Env, " ")
		if strings.Contains(env, "GLUTTER_SCREENSHOT=Z:"+d.dir) {
			shots++
		}
		if strings.Contains(env, "GLUTTER_FRAME_STATS=1") {
			stats++
		}
		if !strings.Contains(env, "GLUTTER_RENDERER=d3d12") {
			t.Errorf("a start without the renderer of -renderer: %v", o.Env)
		}
	}
	if shots != 4 || stats != len(f.launches) {
		t.Errorf("%d launches with a screenshot (want 4) and %d with statistics (want all %d)", shots, stats, len(f.launches))
	}
	if !strings.Contains(progress.String(), "cold start 1 of 2, empty shader cache") {
		t.Errorf("progress: %q", progress.String())
	}
	var out bytes.Buffer
	if err := rep.render(&out, "text"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".exe size", "amd64", "Cold start", "Idle CPU", "Memory (MB"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the report has no %q:\n%s", want, out.String())
		}
	}
}

func TestMeasurePerfFrames(t *testing.T) {
	f := newFakeWine()
	o := testOptions()
	o.frames = true
	o.starts = 1
	o.extras = append(o.extras, "GLUTTER_GALLERY_ANIMATE=1", "GLUTTER_DEV_EVENTS=wait:1s")
	rep, err := measurePerf(context.Background(), f.deps(t, sizes), o)
	if err != nil {
		t.Fatalf("measurePerf: %v", err)
	}
	if !rep.Scenario.Ran || rep.Scenario.Window != 10*time.Second {
		t.Fatalf("Scenario = %+v", rep.Scenario)
	}
	// The app ran 6+10+1 s after its start and drew its first frame at 3 s: the window is the ten seconds
	// after it, even though the run went on a few seconds longer.
	s := rep.Scenario.Summary
	if s.Seconds != 10 || s.Frames != 600 || s.MeanFPS != 60 {
		t.Errorf("Summary = %+v, want 10 seconds of 60 frames", s)
	}
	if len(rep.Memory) != 3 || rep.Memory[2].Label != "after the frame window" {
		t.Errorf("Memory = %+v", rep.Memory)
	}
	// The dev events reach the scenario run and not the cold starts or the idle run.
	for i, l := range f.launches {
		has := strings.Contains(strings.Join(l.Env, " "), "GLUTTER_DEV_EVENTS=")
		if want := i == len(f.launches)-1; has != want {
			t.Errorf("launch %d: dev events %v, want %v", i, has, want)
		}
	}
	var out bytes.Buffer
	if err := rep.render(&out, "markdown"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"### glutter-windows perf", "#### Frame times", "| second | FPS |", "GLUTTER_DEV_EVENTS=wait:1s"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the markdown report has no %q:\n%s", want, out.String())
		}
	}
}

func TestMeasurePerfFailures(t *testing.T) {
	tests := []struct {
		name  string
		fault func(*fakeWine, *perfOptions)
		deps  func(*perfDeps)
		route string
		tag   string // the tag, when the error is made outside this package's functions (route then stops before it)
		in    string
	}{
		{name: "the app logs no first frame", fault: func(f *fakeWine, _ *perfOptions) { f.noFrame = true },
			route: "measurePerf->measurer.coldStarts." + tagWindowsPerfNoFrame, in: "no first frame"},
		{name: "the app ends with a failure", fault: func(f *fakeWine, _ *perfOptions) { f.exit = 3 },
			route: "measurePerf->measurer.coldStarts." + tagWindowsPerfAppDied, in: "failure"},
		{name: "the build makes no amd64 .exe", fault: func(_ *fakeWine, o *perfOptions) { o.archs = []string{"arm64"} },
			route: "measurePerf->measurer.buildAll." + tagWindowsPerfNoExe, in: "no amd64"},
		{name: "the launcher cannot start the app", fault: func(f *fakeWine, _ *perfOptions) { f.launchTag = winetest.TagWineNotFound },
			route: "measurePerf->measurer.coldStarts->fakeWine.Launch." + winetest.TagWineNotFound},
		{name: "the shader cache cannot be emptied", deps: func(d *perfDeps) {
			d.clearCache = func() error { return errors.New(tagWindowsPerfCacheClear, "denied") }
		}, route: "measurePerf->measurer.coldStarts", tag: tagWindowsPerfCacheClear},
		{name: "the build fails", deps: func(d *perfDeps) {
			d.build = func(context.Context, wintool.BuildOptions) (*wintool.BuildResult, error) {
				return nil, errors.New(wintool.TagWindowsBuildInvalid, "no package")
			}
		}, route: "measurePerf->measurer.buildAll", tag: wintool.TagWindowsBuildInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeWine()
			o := testOptions()
			if tt.fault != nil {
				tt.fault(f, &o)
			}
			d := f.deps(t, sizes)
			if tt.deps != nil {
				tt.deps(&d)
			}
			_, err := measurePerf(context.Background(), d, o)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
			if tt.tag != "" && appErr.Tag != tt.tag {
				t.Errorf("tag %s, want %s", appErr.Tag, tt.tag)
			}
			if !strings.Contains(appErr.Message, tt.in) {
				t.Errorf("message %q has no %q", appErr.Message, tt.in)
			}
		})
	}
}

// TestMeasurePerfFailuresOfTheBackgroundRun checks the faults of the runs in the background (the idle
// window and the frame window), which cold starts do not have.
func TestMeasurePerfFailuresOfTheBackgroundRun(t *testing.T) {
	tests := []struct {
		name  string
		fault func(*fakeWine)
		route string
	}{
		{name: "the process never shows", fault: func(f *fakeWine) { f.noProcess = true }, route: "measurePerf->measurer.idle->measurer.runApp." + tagWindowsPerfNoProcess},
		{name: "the app ends before its process is seen", fault: func(f *fakeWine) { f.dies = true }, route: "measurePerf->measurer.idle->measurer.runApp." + tagWindowsPerfAppDied},
		{name: "no first frame in the idle run", fault: func(f *fakeWine) { f.noFrameBackground = true }, route: "measurePerf->measurer.idle." + tagWindowsPerfNoFrame},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeWine()
			tt.fault(f)
			o := testOptions()
			o.starts = 1
			_, err := measurePerf(context.Background(), f.deps(t, sizes), o)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
}

func TestMeasurePerfAppDiesDuringTheWindow(t *testing.T) {
	f := newFakeWine()
	d := f.deps(t, sizes)
	o := testOptions()
	o.starts = 1
	// The process shows, and the run ends by itself while the idle window is open.
	var once sync.Once
	d.sleep = func(ctx context.Context, dur time.Duration) error {
		if dur == o.settle {
			once.Do(func() { close(f.endRun) })
			time.Sleep(50 * time.Millisecond) // the run in the background reports its end
		}
		return f.sleep(ctx, dur)
	}
	_, err := measurePerf(context.Background(), d, o)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if want := "measurePerf->measurer.idle->measurer.runApp." + tagWindowsPerfAppDied; !appErr.HasRoute(want) {
		t.Errorf("route %s, want %s", appErr.Route(), want)
	}
}

func TestMeasurePerfStopsWhenTheContextEnds(t *testing.T) {
	f := newFakeWine()
	ctx, cancel := context.WithCancel(context.Background())
	d := f.deps(t, sizes)
	d.sleep = func(c context.Context, dur time.Duration) error {
		if dur == 5*time.Second {
			cancel()
		}
		if c.Err() != nil {
			return sleepContext(c, time.Hour) // returns at once: the context is done
		}
		return nil
	}
	o := testOptions()
	o.starts = 1
	_, err := measurePerf(ctx, d, o)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagWindowsPerfInterrupted {
		t.Fatalf("got %v, want WindowsPerfInterrupted", err)
	}
	f.mu.Lock()
	running := f.running
	f.mu.Unlock()
	// The run in the background was ended with the measurement.
	deadline := time.Now().Add(2 * time.Second)
	for running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		f.mu.Lock()
		running = f.running
		f.mu.Unlock()
	}
	if running {
		t.Error("the app still runs after the measurement failed")
	}
}

var _ = io.Discard
