package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// perfDeps is everything `glutter-windows perf` takes from the outside world, so that a test can run
// the whole measurement without Wine, the go command or a clock.
type perfDeps struct {
	// launcher runs the .exe and waits for it (Wine); procs lists the Wine processes while it runs.
	launcher wintool.Launcher
	procs    procSource
	// build makes the .exe files (wintool.Build with the real commands).
	build func(ctx context.Context, o wintool.BuildOptions) (*wintool.BuildResult, error)
	// clearCache empties the shader cache of the Wine prefix, so that the next start compiles the
	// shaders.
	clearCache func() error
	// sleep waits for d, or fails when ctx is done; now is the clock (a fake one advances in sleep).
	sleep func(ctx context.Context, d time.Duration) error
	now   func() time.Time
	// loc is the location of the times in the app's log: the host's.
	loc *time.Location
	// progress receives a line for each step.
	progress io.Writer
	// dir is a directory for the screenshots of the cold starts.
	dir string
}

// perfOptions are the parameters of the measurement.
type perfOptions struct {
	// pkg is the package to build.
	pkg string
	// build is the template of the build: Release, Name, AppID and so on. The architectures and the
	// output are set here.
	build wintool.BuildOptions
	// archs are the architectures to build: amd64 runs under Wine, the others are only sized.
	archs []string
	// extras are the environment of the app (-env and -renderer), KEY=value.
	extras []string
	// starts is the number of cold starts for each state of the shader cache, settle the time the app
	// runs before the idle window, seconds the length of the idle window and of the frame window, frames
	// whether the frame window runs.
	starts   int
	settle   time.Duration
	seconds  int
	frames   bool
	timeouts perfTimeouts
}

// perfTimeouts are the waits that a test makes short.
type perfTimeouts struct {
	// process is how long to wait for the app's process to show in /proc after it was started.
	process time.Duration
	// poll is the time between two looks at /proc.
	poll time.Duration
	// startup is the time the frame window gives the app to draw its first frame before the seconds
	// of the window begin to count.
	startup time.Duration
	// run is the longest a cold start may take.
	run time.Duration
}

func defaultTimeouts() perfTimeouts {
	return perfTimeouts{process: 60 * time.Second, poll: 250 * time.Millisecond, startup: 6 * time.Second, run: 3 * time.Minute}
}

// measurer holds the state of one measurement.
type measurer struct {
	d   perfDeps
	o   perfOptions
	exe string
}

// The environment variables that the phases decide on (engine.FrameStatsEnv, windows.DevEventsEnv,
// engine.ScreenshotEnv).
const (
	frameStatsKey = "GLUTTER_FRAME_STATS"
	devEventsKey  = "GLUTTER_DEV_EVENTS"
	screenshotKey = "GLUTTER_SCREENSHOT"
)

// measurePerf builds the app and measures it under Wine (SPEC §25.13, step 28.15): the size of the
// .exe files, the cold start with an empty and with a warm shader cache, the idle CPU and the memory
// of the Wine processes, and the frame times of a scenario.
func measurePerf(ctx context.Context, d perfDeps, o perfOptions) (*perfReport, error) {
	if o.timeouts == (perfTimeouts{}) {
		o.timeouts = defaultTimeouts()
	}
	m := &measurer{d: d, o: o}
	rep := &perfReport{Package: o.pkg, Release: o.build.Release, Env: append([]string(nil), o.extras...)}
	sort.Strings(rep.Env)
	if err := m.buildAll(ctx, rep); err != nil {
		return nil, err
	}
	if err := m.coldStarts(ctx, rep); err != nil {
		return nil, err
	}
	if err := m.idle(ctx, rep); err != nil {
		return nil, err
	}
	if o.frames {
		if err := m.scenario(ctx, rep); err != nil {
			return nil, err
		}
	}
	return rep, nil
}

// note writes a progress line; a failing progress stream does not stop the measurement.
func (m *measurer) note(format string, a ...any) {
	if m.d.progress != nil {
		_, _ = fmt.Fprintf(m.d.progress, "perf: "+format+"\n", a...) // progress only
	}
}

// buildAll builds the package for each architecture, reads the sizes, and keeps the amd64 .exe to run.
func (m *measurer) buildAll(ctx context.Context, rep *perfReport) error {
	opts := m.o.build
	opts.Package, opts.Archs = m.o.pkg, m.o.archs
	opts.Out = filepath.Join(m.d.dir, filepath.Base(filepath.FromSlash(m.o.pkg))+".exe")
	m.note("building %s for %s", m.o.pkg, strings.Join(m.o.archs, ", "))
	res, err := m.d.build(ctx, opts)
	if err != nil {
		return err
	}
	for _, e := range res.Exes {
		rep.Sizes = append(rep.Sizes, sizeRow{Arch: e.Arch, Bytes: e.Size})
		if e.Arch == "amd64" {
			m.exe = e.Path
		}
	}
	if m.exe == "" {
		return errors.New(tagWindowsPerfNoExe, "the build made no amd64 .exe to run under Wine", "package", m.o.pkg)
	}
	return nil
}

// env is the environment of a start: the extras of -env without the dev events unless wanted, plus
// the entries given.
func (m *measurer) env(devEvents bool, more ...string) []string {
	var out []string
	for _, kv := range m.o.extras {
		if k, _, _ := strings.Cut(kv, "="); k == devEventsKey && !devEvents || k == frameStatsKey || k == screenshotKey {
			continue
		}
		out = append(out, kv)
	}
	return append(out, more...)
}

// coldStarts runs the app to its first frame o.starts times with an empty shader cache and o.starts
// times with the cache the empty ones left, with the self-capture (the app saves its first idle frame
// and ends), and reads the engine's own timing from the output.
func (m *measurer) coldStarts(ctx context.Context, rep *perfReport) error {
	shot := filepath.Join(m.d.dir, "first-frame.png")
	for _, state := range []cacheState{cacheEmpty, cacheWarm} {
		for i := 0; i < m.o.starts; i++ {
			if state == cacheEmpty {
				if err := m.d.clearCache(); err != nil {
					return err
				}
			}
			m.note("cold start %d of %d, %s shader cache", i+1, m.o.starts, state)
			env := m.env(false, frameStatsKey+"=1", screenshotKey+"="+m.d.launcher.ProgramPath(shot))
			t0 := m.d.now()
			res, err := m.d.launcher.Launch(ctx, m.exe, winetest.RunOptions{Env: env, Timeout: m.o.timeouts.run})
			wall := m.d.now().Sub(t0)
			if err != nil {
				return err
			}
			if res.ExitCode != 0 {
				return errors.New(tagWindowsPerfAppDied, "the app ended with a failure before it saved its first frame", "exe", m.exe, "exitCode", res.ExitCode, "log", tailLines(res.Output, 25))
			}
			ff, ok := parseFirstFrame(res.Output, m.d.loc)
			if !ok {
				return errors.New(tagWindowsPerfNoFrame, "the app logged no first frame", "exe", m.exe, "log", tailLines(res.Output, 25))
			}
			rep.Starts = append(rep.Starts, startRun{Cache: state, Wall: wall, Frame: ff, Stages: parseStartup(res.Output)})
			if rep.Renderer == "" {
				rep.Renderer = ff.Renderer
			}
		}
	}
	return nil
}

// appRun is an app that runs in the background.
type appRun struct {
	done chan launchOutcome
}

type launchOutcome struct {
	res winetest.Result
	err error
}

// runApp starts the app in the background, waits until its process shows in /proc, calls window, ends
// the run and returns what the app printed. The app is ended by cancelling its context (the harness then
// kills the group and stops Wine). It fails with WindowsPerfNoProcess when no process shows, with
// WindowsPerfAppDied when the app ends before window does, and with the error of the launcher when it
// cannot start the app.
func (m *measurer) runApp(ctx context.Context, env []string, limit time.Duration, window func(ctx context.Context, app *appRun) error) (string, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	app := &appRun{done: make(chan launchOutcome, 1)}
	go func() {
		res, err := m.d.launcher.Launch(runCtx, m.exe, winetest.RunOptions{Env: env, Timeout: limit})
		app.done <- launchOutcome{res, err}
	}()
	// stop ends the run and collects the output.
	stop := func() launchOutcome {
		cancel()
		return <-app.done
	}
	deadline := m.d.now().Add(m.o.timeouts.process)
	for {
		if _, ok, err := m.appProcess(); err != nil {
			stop()
			return "", err
		} else if ok {
			break
		}
		select {
		case out := <-app.done:
			if out.err != nil {
				return out.res.Output, out.err
			}
			return "", errors.New(tagWindowsPerfAppDied, "the app ended before its process was seen", "exe", m.exe, "exitCode", out.res.ExitCode, "log", tailLines(out.res.Output, 25))
		default:
		}
		if m.d.now().After(deadline) {
			out := stop()
			return "", errors.New(tagWindowsPerfNoProcess, "the process of the app did not show in /proc in time", "exe", m.exe, "timeout", m.o.timeouts.process.String(), "log", tailLines(out.res.Output, 25))
		}
		if err := m.d.sleep(ctx, m.o.timeouts.poll); err != nil {
			stop()
			return "", err
		}
	}
	if werr := window(ctx, app); werr != nil {
		// The measurement stopped (the context ended, or a reading failed): that is the error, not the
		// end of the run that follows it.
		out := stop()
		return out.res.Output, werr
	}
	select {
	case out := <-app.done:
		// The run ended by itself while the window was open: the app died (or the launcher failed).
		if out.err != nil {
			return out.res.Output, out.err
		}
		return "", errors.New(tagWindowsPerfAppDied, "the app ended during the measurement", "exe", m.exe, "exitCode", out.res.ExitCode, "log", tailLines(out.res.Output, 25))
	default:
	}
	return stop().res.Output, nil
}

// appProcess finds the process of the app among the Wine processes.
func (m *measurer) appProcess() (procSample, bool, error) {
	procs, err := m.d.procs.Sample()
	if err != nil {
		return procSample{}, false, err
	}
	for _, p := range procs {
		if strings.Contains(p.Cmd, m.exe) {
			return p, true, nil
		}
	}
	return procSample{}, false, nil
}

// memory reads the memory of the Wine processes now.
func (m *measurer) memory(label string) (memSample, error) {
	procs, err := m.d.procs.Sample()
	if err != nil {
		return memSample{}, err
	}
	s := memSample{Label: label}
	for _, p := range procs {
		s.Procs = append(s.Procs, procMem{PID: p.PID, Name: p.Name, IsApp: strings.Contains(p.Cmd, m.exe),
			RSSKB: p.RSSKB, HWMKB: p.HWMKB, PSSKB: p.PSSKB, HasPSS: p.HasPSS})
	}
	sort.Slice(s.Procs, func(i, j int) bool { return s.Procs[i].PID < s.Procs[j].PID })
	return s, nil
}

// idle starts the app with the statistics, lets it settle and measures the CPU time of the Wine
// processes and the frames of a window with no input, and the memory after the settle and after the
// window.
func (m *measurer) idle(ctx context.Context, rep *perfReport) error {
	m.note("idle: starting, settling %s, then measuring %d s", m.o.settle, m.o.seconds)
	window := time.Duration(m.o.seconds) * time.Second
	var res idleResult
	var t0 time.Time
	limit := m.o.settle + window + 2*time.Minute
	out, err := m.runApp(ctx, m.env(false, frameStatsKey+"=1"), limit, func(ctx context.Context, app *appRun) error {
		if err := m.d.sleep(ctx, m.o.settle); err != nil {
			return err
		}
		settled, err := m.memory("after the settle")
		if err != nil {
			return err
		}
		before, err := m.d.procs.Sample()
		if err != nil {
			return err
		}
		t0 = m.d.now()
		if err := m.d.sleep(ctx, window); err != nil {
			return err
		}
		after, err := m.d.procs.Sample()
		if err != nil {
			return err
		}
		res.Window = m.d.now().Sub(t0)
		res.Procs = m.cpuOf(before, after, res.Window)
		end, err := m.memory("after the idle window")
		if err != nil {
			return err
		}
		rep.Memory = append(rep.Memory, settled, end)
		return nil
	})
	if err != nil {
		return err
	}
	if ff, ok := parseFirstFrame(out, m.d.loc); ok {
		if rep.Renderer == "" {
			rep.Renderer = ff.Renderer
		}
	} else {
		return errors.New(tagWindowsPerfNoFrame, "the app logged no first frame while it settled", "exe", m.exe, "log", tailLines(out, 25))
	}
	for _, f := range parseFrameLines(out, frameWindow{afterFirstFrame: true, from: t0}, m.d.loc) {
		res.Frames += f.FPS
	}
	res.sum()
	rep.Idle = res
	return nil
}

// cpuOf is the CPU time of each process that is in both samples over the window.
func (m *measurer) cpuOf(before, after []procSample, window time.Duration) []procCPU {
	ticks0 := map[int]uint64{}
	for _, p := range before {
		ticks0[p.PID] = p.Ticks
	}
	var out []procCPU
	for _, p := range after {
		t0, ok := ticks0[p.PID]
		if !ok {
			continue
		}
		out = append(out, procCPU{PID: p.PID, Name: p.Name, IsApp: strings.Contains(p.Cmd, m.exe), Ticks: p.Ticks - min(p.Ticks, t0),
			Percent: perfstats.CPUPercent(t0, p.Ticks, window, clockTicksPerSecond)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}

// scenario starts the app with the statistics and the dev events of -env and reads the per-second
// statistics of the seconds after its first frame.
func (m *measurer) scenario(ctx context.Context, rep *perfReport) error {
	m.note("frames: starting, measuring %d s", m.o.seconds)
	window := time.Duration(m.o.seconds) * time.Second
	var mem memSample
	limit := m.o.timeouts.startup + window + 2*time.Minute
	out, err := m.runApp(ctx, m.env(true, frameStatsKey+"=1"), limit, func(ctx context.Context, app *appRun) error {
		// The first frame comes a few seconds after the process under Wine: the window is the seconds
		// after it, which the log's times cut out below.
		if err := m.d.sleep(ctx, m.o.timeouts.startup+window+time.Second); err != nil {
			return err
		}
		var err error
		mem, err = m.memory("after the frame window")
		return err
	})
	if err != nil {
		return err
	}
	ff, ok := parseFirstFrame(out, m.d.loc)
	if !ok {
		return errors.New(tagWindowsPerfNoFrame, "the app logged no first frame in the frame window", "exe", m.exe, "log", tailLines(out, 25))
	}
	if rep.Renderer == "" {
		rep.Renderer = ff.Renderer
	}
	fw := frameWindow{afterFirstFrame: true}
	if !ff.At.IsZero() {
		fw.to = ff.At.Truncate(time.Second).Add(window)
	}
	fr := parseFrameLines(out, fw, m.d.loc)
	rep.Scenario = scenarioResult{Ran: true, Window: window, Frames: fr, Summary: perfstats.SummarizeFrames(fr)}
	rep.Memory = append(rep.Memory, mem)
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagWindowsPerfNoExe is the tag of the error for a build that made no amd64 .exe.
	tagWindowsPerfNoExe = "WindowsPerfNoExe"
	// tagWindowsPerfAppDied is the tag of the error when the app ends with a failure or before the
	// measurement is over.
	tagWindowsPerfAppDied = "WindowsPerfAppDied"
	// tagWindowsPerfNoFrame is the tag of the error when the app logs no first frame.
	tagWindowsPerfNoFrame = "WindowsPerfNoFrame"
	// tagWindowsPerfNoProcess is the tag of the error when the app's process does not show in /proc.
	tagWindowsPerfNoProcess = "WindowsPerfNoProcess"
)
