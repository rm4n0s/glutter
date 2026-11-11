package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// perfOptions are what `perf` was asked for.
type perfOptions struct {
	pkg      string
	bf       buildFlags
	env      envFlags
	renderer string
	format   string
	out      string
	starts   int
	seconds  int
	settle   time.Duration
	front    bool
}

// parsePerf reads the flags of `perf`.
func parsePerf(args []string, stderr io.Writer) (perfOptions, error) {
	fs := flag.NewFlagSet("perf", flag.ContinueOnError)
	var o perfOptions
	o.bf.add(fs, false)
	fs.Var(&o.env, "env", "an environment variable for the app, GLUTTER_X=v (repeatable); GLUTTER_DEV_EVENTS=script drives the frame window and needs -debug")
	fs.StringVar(&o.renderer, "renderer", "", "the renderer: auto, metal or soft (GLUTTER_RENDERER); empty leaves the choice to the app")
	fs.IntVar(&o.starts, "starts", 5, "how many times the app is started to time the start, the first with an empty shader cache")
	fs.IntVar(&o.seconds, "seconds", 10, "the length of the idle and frame window, in seconds")
	fs.DurationVar(&o.settle, "settle", 5*time.Second, "how long the app runs after its start before the window")
	fs.BoolVar(&o.front, "front", false, "run the app as a regular app, in front, instead of an accessory app that never takes the focus")
	fs.StringVar(&o.format, "format", "text", "the report: text or markdown")
	fs.StringVar(&o.out, "o", "", "write the report to this file (default: standard output)")
	pkg, err := parseWith(fs, args, stderr)
	if err != nil {
		return o, err
	}
	o.pkg = pkg
	if o.format != "text" && o.format != "markdown" {
		return o, errors.New(tagMacOSUsage, "-format is text or markdown", "format", o.format)
	}
	if o.starts < 1 || o.seconds < 1 || o.settle < 0 {
		return o, errors.New(tagMacOSUsage, "-starts and -seconds are at least 1, and -settle is not negative", "starts", o.starts, "seconds", o.seconds)
	}
	if !o.bf.debug && hasEnv(o.env, "GLUTTER_DEV_EVENTS") {
		return o, errors.New(tagMacOSUsage, "GLUTTER_DEV_EVENTS is read only by debug builds: add -debug (its frame times are those of a debug build)")
	}
	return o, nil
}

// hasEnv says whether a list of NAME=value has the name.
func hasEnv(env []string, name string) bool {
	for _, kv := range env {
		if strings.HasPrefix(kv, name+"=") {
			return true
		}
	}
	return false
}

// perfEnv is the environment of the app for the measurements: the engine's statistics, a cache directory of its own for the shaders
// (empty at the first start), the accessory policy with the display link that waits for frames (unless -front), the renderer, and
// the variables of -env last, so that they win.
func perfEnv(base []string, o perfOptions, cache string) []string {
	env := append([]string(nil), base...)
	env = append(env, "GLUTTER_FRAME_STATS=1", "XDG_CACHE_HOME="+cache)
	if !o.front {
		env = append(env, "GLUTTER_MACOS_ACCESSORY=1", "GLUTTER_DISPLAYLINK=cv")
	}
	if o.renderer != "" {
		env = append(env, "GLUTTER_RENDERER="+o.renderer)
	}
	return append(env, o.env...)
}

// syncBuffer is a buffer that a process writes while the tool reads.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.b.Write(p) // a bytes.Buffer never fails, but the error is not dropped
	if err != nil {
		return n, errors.NewErr(tagMacOSPerfFailed, err)
	}
	return n, nil
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// startResult is one start of the app.
type startResult struct {
	First firstFrame
	Wall  time.Duration
}

// runPerf is `glutter-macos perf` (SPEC §15 "macOS", step 33.15): it builds the package as a release bundle and prints a report with
// the size of the executable (and of the slices with -universal), the start with an empty and with a warm shader cache, the idle CPU
// and the memory of the process, and the frame times of a window (with -env GLUTTER_DEV_EVENTS=script and -debug, of a
// scenario). The numbers are the machine's: they are for noticing regressions, not a guarantee.
func runPerf(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	o, err := parsePerf(args, stderr)
	if err != nil {
		return err
	}
	dir, cleanup, err := tempDir("glutter-macos-perf-")
	if err != nil {
		return err
	}
	defer cleanup()
	res, err := buildBundle(ctx, o.pkg, &o.bf, dir)
	if err != nil {
		return err
	}
	cache := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return errors.NewErr(tagMacOSPerfFailed, err)
	}
	r := perfReport{Package: o.pkg, Release: !o.bf.debug, Env: o.env, Archs: res.Archs, Size: res.Size, Thin: res.Thin, Machine: machine(ctx)}
	for i := 0; i < o.starts; i++ {
		s, err := oneStart(ctx, res.Layout.Executable, o, dir, cache)
		if err != nil {
			return err
		}
		if i == 0 {
			r.Empty = append(r.Empty, s)
		} else {
			r.Warm = append(r.Warm, s)
		}
		if i == 0 {
			r.Renderer = s.First.Renderer
		}
	}
	w, err := window(ctx, res.Layout.Executable, o, cache)
	if err != nil {
		return err
	}
	r.Window = w
	var buf bytes.Buffer
	if err := r.document().Render(&buf, "glutter-macos perf", o.format); err != nil {
		return err
	}
	if o.out != "" {
		if err := os.WriteFile(o.out, buf.Bytes(), 0o644); err != nil {
			return errors.NewErr(tagMacOSPerfFailed, err).SetMetadata("file", o.out)
		}
		return nil
	}
	if _, err := stdout.Write(buf.Bytes()); err != nil {
		return errors.NewErr(tagMacOSPerfFailed, err)
	}
	return nil
}

// machine says which Mac the numbers are from: the system version and the architecture.
func machine(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "sw_vers", "-productVersion").Output()
	v := "unknown"
	if err == nil {
		v = strings.TrimSpace(string(out))
	}
	return "macOS " + v + ", " + runtime.GOARCH
}

// oneStart runs the app until it saves its first frame (GLUTTER_SCREENSHOT makes it end), and reads how long that took.
func oneStart(ctx context.Context, exe string, o perfOptions, dir, cache string) (startResult, error) {
	shot := filepath.Join(dir, "start.png")
	_ = os.Remove(shot) // a stale file must not pass for the new one
	var log syncBuffer
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.Command(exe)
	cmd.Env = append(perfEnv(os.Environ(), o, cache), "GLUTTER_SCREENSHOT="+shot)
	cmd.Stdout, cmd.Stderr = &log, &log
	ownProcessGroup(cmd)
	begin := time.Now()
	if err := cmd.Start(); err != nil {
		return startResult{}, errors.NewErr(tagMacOSPerfFailed, err).SetMetadata("package", o.pkg)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return startResult{}, errors.NewErr(tagMacOSPerfFailed, err).SetMetadata("package", o.pkg, "output", log.String())
		}
	case <-ctx.Done():
		killGroup(cmd)
		<-done
		return startResult{}, errors.New(tagMacOSPerfFailed, "the app did not start in time", "package", o.pkg, "output", log.String())
	}
	wall := time.Since(begin)
	ff, ok := parseFirstFrame(log.String())
	if !ok {
		return startResult{}, errors.New(tagMacOSPerfFailed, "the app did not log its first frame", "package", o.pkg, "output", log.String())
	}
	return startResult{First: ff, Wall: wall}, nil
}

// windowResult is the idle and frame window.
type windowResult struct {
	Seconds int
	CPU     float64
	RSSKB   int
	Frames  perfstats.FrameSummary
	Count   int
}

// window starts the app, waits for it to settle, samples its CPU and memory over the window with `ps`, and sums the frames that it
// logged. The app ends with the window.
func window(ctx context.Context, exe string, o perfOptions, cache string) (windowResult, error) {
	var w windowResult
	var log syncBuffer
	cmd := exec.Command(exe)
	cmd.Env = perfEnv(os.Environ(), o, cache)
	cmd.Stdout, cmd.Stderr = &log, &log
	ownProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return w, errors.NewErr(tagMacOSPerfFailed, err).SetMetadata("package", o.pkg)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer func() {
		killGroup(cmd)
		<-done
	}()
	sleep := func(d time.Duration) error {
		select {
		case <-time.After(d):
			return nil
		case err := <-done:
			done <- err
			return errors.New(tagMacOSPerfFailed, "the app ended during the measurement", "package", o.pkg, "output", log.String())
		case <-ctx.Done():
			return errors.NewErr(tagMacOSPerfFailed, ctx.Err())
		}
	}
	if err := sleep(o.settle); err != nil {
		return w, err
	}
	a, err := sampleOf(ctx, cmd.Process.Pid)
	if err != nil {
		return w, err
	}
	start := time.Now()
	if err := sleep(time.Duration(o.seconds) * time.Second); err != nil {
		return w, err
	}
	b, err := sampleOf(ctx, cmd.Process.Pid)
	if err != nil {
		return w, err
	}
	w.Seconds, w.RSSKB = o.seconds, b.RSSKB
	w.CPU = cpuPercent(a, b, time.Since(start))
	frames := framesOf(log.String())
	w.Count, w.Frames = len(frames), perfstats.SummarizeFrames(frames)
	return w, nil
}

// sampleOf reads the memory and CPU time of a process.
func sampleOf(ctx context.Context, pid int) (psSample, error) {
	out, err := exec.CommandContext(ctx, "ps", "-o", "rss=,cputime=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return psSample{}, errors.NewErr(tagMacOSPerfFailed, err).SetMetadata("pid", pid)
	}
	s, ok := parsePS(string(out))
	if !ok {
		return psSample{}, errors.New(tagMacOSPerfFailed, "ps said something that is not memory and CPU time", "pid", pid, "output", string(out))
	}
	return s, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagMacOSPerfFailed = "MacOSPerfFailed"
