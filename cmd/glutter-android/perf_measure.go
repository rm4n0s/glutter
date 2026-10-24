package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// perfADB is the part of adb that the measurement uses; androidtool.ADB is one, and the tests
// give a fake that answers from sample text.
type perfADB interface {
	Output(ctx context.Context, args ...string) (string, error)
}

// perfDeps is everything `glutter-android perf` takes from the outside world, so that a test
// can run the whole measurement without a device, an NDK or a clock.
type perfDeps struct {
	adb perfADB
	// serial names the device in the report.
	serial string
	// build makes an APK (androidtool.Build with the SDK and the real commands).
	build func(ctx context.Context, o androidtool.BuildOptions) (*androidtool.BuildResult, error)
	// sleep waits for d, or fails when ctx is done.
	sleep func(ctx context.Context, d time.Duration) error
	// now is the clock; a fake one advances in sleep.
	now func() time.Time
	// progress receives a line for each step.
	progress io.Writer
	// dir is a directory for the APKs.
	dir string
}

// perfOptions are the parameters of the measurement.
type perfOptions struct {
	// pkg is the package to build.
	pkg string
	// build is the template of the builds: Package, Release, Key, Cert, AppID and so on. The
	// ABIs and the output file are set for each build.
	build androidtool.BuildOptions
	// abis are the ABIs to build, one APK each and one with all of them.
	abis []androidtool.ABI
	// extras are the environment of the app (-env and -renderer), GLUTTER_DEV_EVENTS included.
	extras map[string]string
	// starts is the number of cold starts, settle the time before the idle window, seconds the
	// length of the idle window and of the frame window, frames whether the frame window runs.
	starts   int
	settle   time.Duration
	seconds  int
	frames   bool
	timeouts perfTimeouts
}

// perfTimeouts are the waits that a test makes short.
type perfTimeouts struct {
	// firstFrame is how long to wait for the first frame of a start.
	firstFrame time.Duration
	// poll is the time between two reads of the log.
	poll time.Duration
	// afterStop is the pause after the app was stopped, before it is started again.
	afterStop time.Duration
}

func defaultTimeouts() perfTimeouts {
	return perfTimeouts{firstFrame: 90 * time.Second, poll: 250 * time.Millisecond, afterStop: time.Second}
}

// measurer holds the state of one measurement.
type measurer struct {
	d     perfDeps
	o     perfOptions
	appID string
}

// measurePerf builds the app, installs it on the device and measures it (SPEC §24.13, step
// 26.5): the size of the APK per ABI, the cold start, the idle CPU, the frame times of a
// scenario and the memory. It leaves the app stopped.
func measurePerf(ctx context.Context, d perfDeps, o perfOptions) (*perfReport, error) {
	if o.timeouts == (perfTimeouts{}) {
		o.timeouts = defaultTimeouts()
	}
	m := &measurer{d: d, o: o}
	rep := &perfReport{Package: o.pkg, Release: o.build.Release, Env: envList(o.extras)}

	// The device first: a device that runs none of the ABIs fails before the builds.
	if err := m.deviceInfo(ctx, rep); err != nil {
		return nil, err
	}
	abi, err := pickABI(rep.Device.ABIList, o.abis)
	if err != nil {
		return nil, err
	}
	rep.Device.ABI = abi.Name
	apk, err := m.buildAll(ctx, rep, abi)
	if err != nil {
		return nil, err
	}
	m.note("installing %s on %s", filepath.Base(apk), d.serial)
	if _, err := d.adb.Output(ctx, "install", "-r", "-t", apk); err != nil {
		return nil, err
	}
	// Leave nothing running, even when a step fails or the context is done (a fresh context).
	defer func() { _, _ = m.shell(context.Background(), "am", "force-stop", m.appID) }() // best effort: the device may be gone
	// A bigger log buffer, so that the lines of a long scenario are not pushed out; best effort.
	_, _ = d.adb.Output(ctx, "logcat", "-G", "8M")

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

func (m *measurer) shell(ctx context.Context, args ...string) (string, error) {
	return m.d.adb.Output(ctx, append([]string{"shell"}, args...)...)
}

// prop reads a system property of the device.
func (m *measurer) prop(ctx context.Context, name string) (string, error) {
	out, err := m.shell(ctx, "getprop", name)
	return strings.TrimSpace(out), err
}

func (m *measurer) deviceInfo(ctx context.Context, rep *perfReport) error {
	rep.Device.Serial = m.d.serial
	for _, p := range []struct {
		name string
		dst  *string
	}{
		{"ro.product.model", &rep.Device.Model},
		{"ro.build.version.release", &rep.Device.Android},
		{"ro.build.version.sdk", &rep.Device.SDK},
		{"ro.product.cpu.abilist", &rep.Device.ABIList},
	} {
		v, err := m.prop(ctx, p.name)
		if err != nil {
			return err
		}
		*p.dst = v
	}
	return nil
}

// buildAll builds one APK per ABI and, with more than one, one with all of them; it reads
// their sizes into the report and returns the APK of the device's ABI.
func (m *measurer) buildAll(ctx context.Context, rep *perfReport, device androidtool.ABI) (string, error) {
	var install string
	build := func(label string, abis []androidtool.ABI) (*androidtool.BuildResult, error) {
		opts := m.o.build
		opts.Package = m.o.pkg
		opts.ABIs, opts.Out = nil, filepath.Join(m.d.dir, strings.ReplaceAll(label, "+", "-")+".apk")
		for _, a := range abis {
			opts.ABIs = append(opts.ABIs, a.Short)
		}
		m.note("building %s for %s", m.o.pkg, label)
		res, err := m.d.build(ctx, opts)
		if err != nil {
			return nil, err
		}
		info, err := readAPK(res.APK)
		if err != nil {
			return nil, err
		}
		row := sizeRow{Label: label, APK: res.Size, Libs: info.LibBytes}
		for _, a := range abis {
			row.ABIs = append(row.ABIs, a.Name)
		}
		rep.Sizes = append(rep.Sizes, row)
		m.appID = res.AppID
		return res, nil
	}
	for _, a := range m.o.abis {
		res, err := build(a.Short, []androidtool.ABI{a})
		if err != nil {
			return "", err
		}
		if a.Name == device.Name {
			install = res.APK
		}
	}
	if len(m.o.abis) > 1 {
		var shorts []string
		for _, a := range m.o.abis {
			shorts = append(shorts, a.Short)
		}
		if _, err := build(strings.Join(shorts, "+"), m.o.abis); err != nil {
			return "", err
		}
	}
	return install, nil
}

// extrasFor is the environment of a start: the extras of -env without the dev events and the
// statistics unless wanted, plus the statistics for the phases that read them.
func (m *measurer) extrasFor(devEvents, frameStats bool) map[string]string {
	out := map[string]string{}
	for k, v := range m.o.extras {
		if k == devEventsKey && !devEvents || k == frameStatsKey && !frameStats {
			continue
		}
		out[k] = v
	}
	if frameStats {
		out[frameStatsKey] = "1"
	}
	return out
}

// The environment variables that the phases decide on (engine.FrameStatsEnv, android.DevEventsEnv).
const (
	frameStatsKey = "GLUTTER_FRAME_STATS"
	devEventsKey  = "GLUTTER_DEV_EVENTS"
)

// clearLog empties the device's log. Right after a boot it can fail for a while, so it is tried
// again a few times.
func (m *measurer) clearLog(ctx context.Context) error {
	var err error
	for try := 0; try < 5; try++ {
		if _, err = m.d.adb.Output(ctx, "logcat", "-c"); err == nil {
			return nil
		}
		if serr := m.d.sleep(ctx, 2*time.Second); serr != nil {
			return serr
		}
	}
	return err
}

// log is the app's log lines so far.
func (m *measurer) log(ctx context.Context) (string, error) {
	return m.d.adb.Output(ctx, "logcat", "-d", "-v", "brief", "-s", m.appID+":V")
}

// stopApp stops the app, clears the log and waits a moment, so that the next start is cold.
func (m *measurer) stopApp(ctx context.Context) error {
	if _, err := m.shell(ctx, "am", "force-stop", m.appID); err != nil {
		return err
	}
	if err := m.clearLog(ctx); err != nil {
		return err
	}
	return m.d.sleep(ctx, m.o.timeouts.afterStop)
}

// startApp runs `am start -W` with the extras and returns what it prints, parsed.
func (m *measurer) startApp(ctx context.Context, extras map[string]string) (amStart, error) {
	args := []string{"am", "start", "-W", "-n", m.appID + "/" + androidtool.ActivityClass}
	keys := make([]string, 0, len(extras))
	for k := range extras {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--es", androidtool.ShellQuote(k), androidtool.ShellQuote(extras[k]))
	}
	out, err := m.shell(ctx, args...)
	if err != nil {
		return amStart{}, err
	}
	return parseAmStart(out)
}

// parsePid reads the output of `pidof -s`; 0 for nothing.
func parsePid(out string) int {
	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || pid < 0 {
		return 0
	}
	return pid
}

// pid is the process ID of the app, 0 when it does not run (pidof exits with 1 then).
func (m *measurer) pid(ctx context.Context) int {
	out, err := m.shell(ctx, "pidof", "-s", m.appID)
	if err != nil {
		return 0
	}
	return parsePid(out)
}

// waitFirstFrame waits until the engine has logged its first frame. It fails with
// AndroidPerfAppDied when the process is gone and AndroidPerfNoFrame when the time is over.
func (m *measurer) waitFirstFrame(ctx context.Context) (firstFrame, error) {
	deadline := m.d.now().Add(m.o.timeouts.firstFrame)
	for {
		log, err := m.log(ctx)
		if err != nil {
			return firstFrame{}, err
		}
		if ff, ok := parseFirstFrame(log); ok {
			return ff, nil
		}
		if m.pid(ctx) == 0 {
			return firstFrame{}, errors.New(tagAndroidPerfAppDied, "the app ended before its first frame", "appID", m.appID, "log", tailLines(log, 25))
		}
		if m.d.now().After(deadline) {
			return firstFrame{}, errors.New(tagAndroidPerfNoFrame, "the app drew no frame in time", "appID", m.appID, "timeout", m.o.timeouts.firstFrame.String(), "log", tailLines(log, 25))
		}
		if err := m.d.sleep(ctx, m.o.timeouts.poll); err != nil {
			return firstFrame{}, err
		}
	}
}

// tailLines is the last n lines of s.
func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// coldStarts stops the app and starts it again, starts times, and notes how long each took.
func (m *measurer) coldStarts(ctx context.Context, rep *perfReport) error {
	extras := m.extrasFor(false, false)
	for i := 0; i < m.o.starts; i++ {
		if err := m.stopApp(ctx); err != nil {
			return err
		}
		m.note("cold start %d of %d", i+1, m.o.starts)
		am, err := m.startApp(ctx, extras)
		if err != nil {
			return err
		}
		ff, err := m.waitFirstFrame(ctx)
		if err != nil {
			return err
		}
		rep.Starts = append(rep.Starts, startRun{Total: am.Total, Wait: am.Wait, This: am.This, LaunchState: am.LaunchState, Frame: ff})
		if rep.Renderer == "" {
			rep.Renderer = ff.Renderer
		}
	}
	return nil
}

// sample reads the memory of the running app: dumpsys meminfo and /proc/<pid>/status.
func (m *measurer) sample(ctx context.Context, label string, pid int) (memSample, error) {
	out, err := m.shell(ctx, "dumpsys", "meminfo", m.appID)
	if err != nil {
		return memSample{}, err
	}
	mi, err := parseMeminfo(out)
	if err != nil {
		return memSample{}, err
	}
	status, err := m.shell(ctx, "cat", fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return memSample{}, err
	}
	ps, err := parseProcStatus(status)
	if err != nil {
		return memSample{}, err
	}
	return memSample{Label: label, PSSKB: mi.PSSKB, RSSKB: mi.RSSKB, HasRSS: mi.HasRSS, VmRSSKB: ps.VmRSSKB, VmHWMKB: ps.VmHWMKB}, nil
}

// ticks reads utime+stime of the process.
func (m *measurer) ticks(ctx context.Context, pid int) (uint64, error) {
	out, err := m.shell(ctx, "cat", fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	return parseProcStat(out)
}

// idle starts the app with the statistics, lets it settle and measures the CPU time and the
// frames of a window with no input, and the memory after the settle and after the window.
func (m *measurer) idle(ctx context.Context, rep *perfReport) error {
	if err := m.stopApp(ctx); err != nil {
		return err
	}
	m.note("idle: starting, settling %s, then measuring %d s", m.o.settle, m.o.seconds)
	if _, err := m.startApp(ctx, m.extrasFor(false, true)); err != nil {
		return err
	}
	if _, err := m.waitFirstFrame(ctx); err != nil {
		return err
	}
	if err := m.d.sleep(ctx, m.o.settle); err != nil {
		return err
	}
	pid := m.pid(ctx)
	if pid == 0 {
		return errors.New(tagAndroidPerfAppDied, "the app ended while it settled", "appID", m.appID)
	}
	settled, err := m.sample(ctx, "after the settle", pid)
	if err != nil {
		return err
	}
	if err := m.clearLog(ctx); err != nil {
		return err
	}
	t0 := m.d.now()
	ticks0, err := m.ticks(ctx, pid)
	if err != nil {
		return err
	}
	if err := m.d.sleep(ctx, time.Duration(m.o.seconds)*time.Second); err != nil {
		return err
	}
	ticks1, err := m.ticks(ctx, pid)
	if err != nil {
		return err
	}
	elapsed := m.d.now().Sub(t0)
	log, err := m.log(ctx)
	if err != nil {
		return err
	}
	if now := m.pid(ctx); now != pid {
		return errors.New(tagAndroidPerfAppDied, "the app's process changed during the idle window", "appID", m.appID, "before", pid, "after", now)
	}
	after, err := m.sample(ctx, "after the idle window", pid)
	if err != nil {
		return err
	}
	frames := 0
	for _, f := range parseFrameLines(log) {
		frames += f.FPS
	}
	rep.Idle = idleResult{Window: elapsed, Ticks: ticks1 - ticks0, CPUPercent: cpuPercent(ticks0, ticks1, elapsed), Frames: frames}
	rep.Memory = append(rep.Memory, settled, after)
	return nil
}

// scenario starts the app with the statistics and the dev events of -env and reads the
// per-second statistics of the seconds after its first frame.
func (m *measurer) scenario(ctx context.Context, rep *perfReport) error {
	if err := m.stopApp(ctx); err != nil {
		return err
	}
	m.note("frames: starting, measuring %d s", m.o.seconds)
	if _, err := m.startApp(ctx, m.extrasFor(true, true)); err != nil {
		return err
	}
	ff, err := m.waitFirstFrame(ctx)
	if err != nil {
		return err
	}
	if rep.Renderer == "" {
		rep.Renderer = ff.Renderer
	}
	window := time.Duration(m.o.seconds) * time.Second
	if err := m.d.sleep(ctx, window); err != nil {
		return err
	}
	log, err := m.log(ctx)
	if err != nil {
		return err
	}
	pid := m.pid(ctx)
	if pid == 0 {
		return errors.New(tagAndroidPerfAppDied, "the app ended during the frame window", "appID", m.appID, "log", tailLines(log, 25))
	}
	mem, err := m.sample(ctx, "after the frame window", pid)
	if err != nil {
		return err
	}
	fr := parseFrameLines(log)
	rep.Scenario = scenarioResult{Ran: true, Window: window, Frames: fr, Summary: summarizeFrames(fr)}
	rep.Memory = append(rep.Memory, mem)
	return nil
}

// envList is the extras as sorted KEY=value strings.
func envList(extras map[string]string) []string {
	var out []string
	for k, v := range extras {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}
