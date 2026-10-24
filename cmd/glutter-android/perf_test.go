package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// fakeDevice is a device that answers adb from sample text, on a clock that only the sleeps
// of the measurement advance. The app on it draws its first frame at once, uses idleTicks
// clock ticks a second while it is idle, and logs a frames line a second when it was started
// with GLUTTER_DEV_EVENTS (a scenario).
type fakeDevice struct {
	abilist   string
	now       time.Time
	pid       int
	nextPid   int
	startedAt time.Time
	clearedAt time.Time
	extras    string // the extras of the last start, as the command line has them
	starts    []string
	calls     []string

	idleTicks uint64
	baseTicks uint64
	noFrame   bool // the app starts but logs no first frame
	dies      bool // the app has no process after it was started
	amOut     string
	failOn    string // a command (joined arguments, a prefix) that fails
}

func newFakeDevice() *fakeDevice {
	return &fakeDevice{abilist: "x86_64,arm64-v8a", now: time.Unix(1000, 0), nextPid: 4000, idleTicks: 2, baseTicks: 1800, amOut: amStartAndroid14}
}

func (f *fakeDevice) sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.now = f.now.Add(d)
	return nil
}

func (f *fakeDevice) clock() time.Time { return f.now }

func (f *fakeDevice) running() time.Duration { return f.now.Sub(f.startedAt) }

func (f *fakeDevice) Output(ctx context.Context, args ...string) (string, error) {
	cmd := strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	f.now = f.now.Add(40 * time.Millisecond) // every adb call takes a little time
	if f.failOn != "" && strings.HasPrefix(cmd, f.failOn) {
		return "", errors.New(androidtool.TagAndroidCommandFailed, "the fake failed", "command", cmd)
	}
	switch {
	case cmd == "shell getprop ro.product.model":
		return "Pixel 8\n", nil
	case cmd == "shell getprop ro.build.version.release":
		return "14\n", nil
	case cmd == "shell getprop ro.build.version.sdk":
		return "34\n", nil
	case cmd == "shell getprop ro.product.cpu.abilist":
		return f.abilist + "\n", nil
	case strings.HasPrefix(cmd, "install "):
		return "Success\n", nil
	case cmd == "logcat -G 8M":
		return "", nil
	case cmd == "logcat -c":
		f.clearedAt = f.now
		return "", nil
	case strings.HasPrefix(cmd, "shell am force-stop"):
		f.pid = 0
		return "", nil
	case strings.HasPrefix(cmd, "shell am start -W"):
		f.starts = append(f.starts, cmd)
		f.extras = cmd
		f.startedAt = f.now
		f.nextPid++
		if !f.dies {
			f.pid = f.nextPid
		}
		return f.amOut, nil
	case strings.HasPrefix(cmd, "shell pidof -s"):
		if f.pid == 0 {
			return "", errors.New(androidtool.TagAndroidCommandFailed, "pidof exits with 1", "command", cmd)
		}
		return fmt.Sprintf("%d\n", f.pid), nil
	case strings.HasPrefix(cmd, "logcat -d"):
		return f.logcat(), nil
	case strings.HasSuffix(cmd, "/stat"):
		secs := uint64(f.running() / time.Second)
		return fmt.Sprintf("%d (org.glutter.counter) S 1 1 0 0 -1 1077936448 1 0 0 0 %d %d 0 0 20 0 28 0 1 1 1\n", f.pid, f.baseTicks+f.idleTicks*secs, uint64(311)), nil
	case strings.HasSuffix(cmd, "/status"):
		return "Name:\torg.glutter.counter\nVmHWM:\t  151204 kB\nVmRSS:\t  143360 kB\n", nil
	case strings.HasPrefix(cmd, "shell dumpsys meminfo"):
		return meminfoAndroid14, nil
	}
	return "", errors.New(androidtool.TagAndroidCommandFailed, "the fake does not know this command", "command", cmd)
}

// logcat is the log of the app since the last clear.
func (f *fakeDevice) logcat() string {
	var b strings.Builder
	if f.startedAt.After(f.clearedAt) && !f.noFrame && !f.dies {
		b.WriteString(`I/org.glutter.counter( 4242): 2026/10/24 10:11:12 INFO first frame presented since_run=212ms renderer="vulkan: Adreno (TM) 740" go_heap_mb=9 go_sys_mb=22` + "\n")
	}
	if strings.Contains(f.extras, "GLUTTER_DEV_EVENTS") && strings.Contains(f.extras, "GLUTTER_FRAME_STATS") {
		from := f.clearedAt
		if f.startedAt.After(from) {
			from = f.startedAt
		}
		for s := 1; s <= int(f.now.Sub(from)/time.Second); s++ {
			over := 0
			build := "3.2ms"
			if s == 2 {
				over, build = 3, "7.5ms"
			}
			fmt.Fprintf(&b, "I/org.glutter.counter( 4242): 2026/10/24 10:11:%02d INFO frames renderer=\"vulkan: Adreno (TM) 740\" fps=%d build_avg=%s build_max=21.4ms raster_avg=1.9ms raster_max=4.1ms over_16ms=%d go_heap_mb=%d go_sys_mb=38\n",
				12+s, 60-over, build, over, 12+s)
		}
	}
	return b.String()
}

// build is the fake of androidtool.Build: it writes an APK with a libapp.so per ABI.
func (f *fakeDevice) build(dir string, sizes map[string]int) func(context.Context, androidtool.BuildOptions) (*androidtool.BuildResult, error) {
	return func(ctx context.Context, o androidtool.BuildOptions) (*androidtool.BuildResult, error) {
		files := map[string]int{"classes.dex": 2000, "AndroidManifest.xml": 300}
		for _, short := range o.ABIs {
			for _, a := range androidtool.ABIs {
				if a.Short == short {
					files["lib/"+a.Name+"/libapp.so"] = sizes[a.Name]
				}
			}
		}
		if err := os.MkdirAll(filepath.Dir(o.Out), 0o755); err != nil {
			return nil, err
		}
		fh, err := os.Create(o.Out)
		if err != nil {
			return nil, err
		}
		zw := zip.NewWriter(fh)
		for name, n := range files {
			w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
			if err != nil {
				return nil, err
			}
			if _, err := w.Write(make([]byte, n)); err != nil {
				return nil, err
			}
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		if err := fh.Close(); err != nil {
			return nil, err
		}
		st, err := os.Stat(o.Out)
		if err != nil {
			return nil, err
		}
		return &androidtool.BuildResult{APK: o.Out, AppID: "org.glutter.counter", Size: st.Size()}, nil
	}
}

func (f *fakeDevice) deps(t *testing.T) (perfDeps, *bytes.Buffer) {
	progress := &bytes.Buffer{}
	return perfDeps{adb: f, serial: "emulator-5584", sleep: f.sleep, now: f.clock, progress: progress, dir: t.TempDir(),
		build: f.build(t.TempDir(), map[string]int{"arm64-v8a": 9000, "x86_64": 10000})}, progress
}

func testPerfOptions() perfOptions {
	return perfOptions{pkg: "./examples/counter", build: androidtool.BuildOptions{Release: true}, abis: androidtool.ABIs,
		extras: map[string]string{"GLUTTER_GALLERY_PAGE": "data"}, starts: 3, settle: 4 * time.Second, seconds: 5, frames: true,
		timeouts: perfTimeouts{firstFrame: 5 * time.Second, poll: 250 * time.Millisecond, afterStop: time.Second}}
}

func TestMeasurePerf(t *testing.T) {
	f := newFakeDevice()
	deps, progress := f.deps(t)
	o := testPerfOptions()
	o.extras["GLUTTER_DEV_EVENTS"] = "wait:1s;tap:540,1200"
	rep, err := measurePerf(context.Background(), deps, o)
	if err != nil {
		t.Fatalf("measurePerf: %v\n%s", err, progress)
	}

	// Sizes: one APK per ABI and one with both, and the libapp.so of each inside.
	if len(rep.Sizes) != 3 || rep.Sizes[0].Label != "arm64" || rep.Sizes[1].Label != "amd64" || rep.Sizes[2].Label != "arm64+amd64" {
		t.Fatalf("sizes %+v", rep.Sizes)
	}
	if rep.Sizes[0].Libs["arm64-v8a"] != 9000 || rep.Sizes[1].Libs["x86_64"] != 10000 || len(rep.Sizes[2].Libs) != 2 {
		t.Errorf("libs %+v", rep.Sizes)
	}
	if rep.Sizes[2].APK <= rep.Sizes[0].APK || rep.Sizes[2].APK <= rep.Sizes[1].APK {
		t.Errorf("the APK with both ABIs is not the biggest: %+v", rep.Sizes)
	}
	// The device runs x86_64: the APK of that ABI is installed.
	installs := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, "install ") {
			installs++
			if !strings.HasSuffix(c, "amd64.apk") {
				t.Errorf("installed %q, want the amd64 APK", c)
			}
		}
	}
	if installs != 1 {
		t.Errorf("%d installs, want 1", installs)
	}
	if rep.Device.Model != "Pixel 8" || rep.Device.SDK != "34" || rep.Device.ABI != "x86_64" || rep.Renderer != "vulkan: Adreno (TM) 740" {
		t.Errorf("device %+v renderer %q", rep.Device, rep.Renderer)
	}

	// Cold starts: 3 stops and starts, then the idle start, then the scenario start.
	if len(rep.Starts) != 3 || rep.Starts[0].Total != 523*time.Millisecond || rep.Starts[2].Wait != 541*time.Millisecond ||
		rep.Starts[1].Frame.SinceRun != 212*time.Millisecond || rep.Starts[0].LaunchState != "COLD" {
		t.Errorf("starts %+v", rep.Starts)
	}
	if len(f.starts) != 3+1+1 {
		t.Fatalf("%d starts: %q", len(f.starts), f.starts)
	}
	for i, s := range f.starts {
		hasStats := strings.Contains(s, "GLUTTER_FRAME_STATS")
		hasEvents := strings.Contains(s, "GLUTTER_DEV_EVENTS")
		if !strings.Contains(s, "-n org.glutter.counter/"+androidtool.ActivityClass) || !strings.Contains(s, "'GLUTTER_GALLERY_PAGE' 'data'") {
			t.Errorf("start %d: %s", i, s)
		}
		switch {
		case i < 3 && (hasStats || hasEvents):
			t.Errorf("cold start %d has the statistics or the dev events: %s", i, s)
		case i == 3 && (!hasStats || hasEvents):
			t.Errorf("the idle start must have the statistics and no dev events: %s", s)
		case i == 4 && (!hasStats || !hasEvents || !strings.Contains(s, "'wait:1s;tap:540,1200'")):
			t.Errorf("the scenario start must have both: %s", s)
		}
	}

	// Idle: 5 s of 2 ticks a second is 10 ticks, 0.1 s of CPU over a window of a little over 5 s.
	if rep.Idle.Ticks != 10 || rep.Idle.CPUPercent < 1.8 || rep.Idle.CPUPercent > 2.1 || rep.Idle.Frames != 0 {
		t.Errorf("idle %+v", rep.Idle)
	}
	if rep.Idle.Window < 5*time.Second || rep.Idle.Window > 6*time.Second {
		t.Errorf("idle window %v", rep.Idle.Window)
	}

	// The scenario: the frames lines of the 5 seconds.
	sc := rep.Scenario
	if !sc.Ran || sc.Summary.Seconds != 5 || sc.Summary.Frames != 60*5-3 || sc.Summary.Over16 != 3 || sc.Summary.BuildMax != 21400*time.Microsecond {
		t.Errorf("scenario %+v", sc)
	}

	// Memory: after the settle, after the idle window, after the scenario.
	if len(rep.Memory) != 3 || rep.Memory[0].PSSKB != 66436 || !rep.Memory[1].HasRSS || rep.Memory[2].VmHWMKB != 151204 || rep.Memory[0].Label != "after the settle" {
		t.Errorf("memory %+v", rep.Memory)
	}

	// The app is stopped at the end, and nothing else is left to stop.
	if last := f.calls[len(f.calls)-1]; last != "shell am force-stop org.glutter.counter" || f.pid != 0 {
		t.Errorf("last command %q, pid %d", last, f.pid)
	}
	if !strings.Contains(progress.String(), "cold start 3 of 3") {
		t.Errorf("progress:\n%s", progress)
	}
}

func TestMeasurePerfOneABIWithoutFrames(t *testing.T) {
	f := newFakeDevice()
	deps, _ := f.deps(t)
	o := testPerfOptions()
	o.abis, o.frames, o.starts = androidtool.ABIs[1:], false, 1
	rep, err := measurePerf(context.Background(), deps, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Sizes) != 1 || rep.Scenario.Ran || len(rep.Memory) != 2 || len(f.starts) != 2 {
		t.Errorf("sizes %+v, scenario %+v, %d memory samples, %d starts", rep.Sizes, rep.Scenario, len(rep.Memory), len(f.starts))
	}
}

func TestMeasurePerfErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeDevice, o *perfOptions)
		route string
	}{
		{name: "a device with none of the ABIs", setup: func(f *fakeDevice, o *perfOptions) { f.abilist = "armeabi-v7a" },
			route: "measurePerf->pickABI." + tagAndroidPerfNoABI},
		{name: "only the other ABI was built", setup: func(f *fakeDevice, o *perfOptions) { f.abilist, o.abis = "x86_64", androidtool.ABIs[:1] },
			route: "measurePerf->pickABI." + tagAndroidPerfNoABI},
		{name: "no first frame", setup: func(f *fakeDevice, o *perfOptions) { f.noFrame = true },
			route: "measurePerf->measurer.coldStarts->measurer.waitFirstFrame." + tagAndroidPerfNoFrame},
		{name: "the app dies at once", setup: func(f *fakeDevice, o *perfOptions) { f.dies = true },
			route: "measurePerf->measurer.coldStarts->measurer.waitFirstFrame." + tagAndroidPerfAppDied},
		{name: "am start says the activity does not exist", setup: func(f *fakeDevice, o *perfOptions) { f.amOut = amStartMissingActivity },
			route: "measurePerf->measurer.coldStarts->measurer.startApp->parseAmStart." + tagAndroidPerfStartRejected},
		{name: "am start prints no times", setup: func(f *fakeDevice, o *perfOptions) { f.amOut = "Status: ok\nComplete\n" },
			route: "measurePerf->measurer.coldStarts->measurer.startApp->parseAmStart." + tagAndroidPerfAmStartParse},
		{name: "adb fails to install", setup: func(f *fakeDevice, o *perfOptions) { f.failOn = "install" },
			route: "measurePerf->fakeDevice.Output." + androidtool.TagAndroidCommandFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeDevice()
			deps, _ := f.deps(t)
			o := testPerfOptions()
			tt.setup(f, &o)
			_, err := measurePerf(context.Background(), deps, o)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
			if f.pid != 0 && tt.name != "adb fails to install" {
				t.Errorf("the app is left running (pid %d)", f.pid)
			}
		})
	}
}

func TestMeasurePerfStopsWhenInterrupted(t *testing.T) {
	f := newFakeDevice()
	deps, _ := f.deps(t)
	ctx, cancel := context.WithCancel(context.Background())
	sleeps := 0
	deps.sleep = func(c context.Context, d time.Duration) error {
		if sleeps++; sleeps < 6 {
			return f.sleep(c, d)
		}
		cancel()
		return sleepContext(c, time.Hour) // the real one fails once ctx is done
	}
	_, err := measurePerf(ctx, deps, testPerfOptions())
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagAndroidPerfInterrupted {
		t.Fatalf("got %v, want %s", err, tagAndroidPerfInterrupted)
	}
	if last := f.calls[len(f.calls)-1]; last != "shell am force-stop org.glutter.counter" {
		t.Errorf("the app was not stopped after the interruption: last command %q", last)
	}
}

func TestSleepContext(t *testing.T) {
	if err := sleepContext(context.Background(), time.Millisecond); err != nil {
		t.Errorf("sleepContext: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sleepContext(ctx, time.Hour)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("sleepContext."+tagAndroidPerfInterrupted) || !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

func TestPerfUsageErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		route string
		in    string
	}{
		{name: "no package", args: []string{"perf"}, route: "run->runPerf->parse." + tagAndroidUsage, in: "one package"},
		{name: "a bad format", args: []string{"perf", "-format", "html", "./examples/counter"}, route: "run->runPerf." + tagAndroidPerfBadFormat, in: "text or markdown"},
		{name: "zero starts", args: []string{"perf", "-starts", "0", "./examples/counter"}, route: "run->runPerf." + tagAndroidPerfBadCount, in: "at least 1"},
		{name: "zero seconds", args: []string{"perf", "-seconds", "0", "./examples/counter"}, route: "run->runPerf." + tagAndroidPerfBadCount},
		{name: "an env that is not GLUTTER_", args: []string{"perf", "-env", "HOME=/x", "./examples/counter"}, route: "run->runPerf->parseEnvs." + tagAndroidUsage},
		{name: "dev events need a debug build", args: []string{"perf", "-env", "GLUTTER_DEV_EVENTS=wait:1s", "./examples/counter"}, route: "run->runPerf." + tagAndroidPerfDevEventsRelease, in: "-release=false"},
		{name: "an unknown ABI", args: []string{"perf", "-abi", "armv7", "./examples/counter"}, route: "run->runPerf->parseABIs." + tagAndroidPerfBadABI, in: "armv7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args, io.Discard, io.Discard)
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.route) || !strings.Contains(appErr.Message, tt.in) {
				t.Errorf("got %v, want route %s with %q", err, tt.route, tt.in)
			}
		})
	}
}

func TestPerfFlagDefaults(t *testing.T) {
	// The defaults the help text promises: a release build of both ABIs, five starts.
	var out bytes.Buffer
	_ = run(context.Background(), []string{"perf", "-h"}, io.Discard, &out) // -h is an error of the flag set
	for _, want := range []string{"-release", "(default true)", `(default "arm64,amd64")`, "-starts int", "(default 5)", "-settle duration", "(default 8s)", "-seconds int", "-format string"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help of perf has no %q:\n%s", want, out.String())
		}
	}
}

func TestPerfHelpListsIt(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"help"}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "glutter-android perf") {
		t.Errorf("help has no perf:\n%s", out.String())
	}
}
