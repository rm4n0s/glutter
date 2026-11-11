//go:build glutter_ios

package iostest_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostool"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// appPID is the process of the app on the host: a Simulator app is a process of the Mac, whose path has the device's UDID.
func appPID(t *testing.T, udid, exe string) int {
	t.Helper()
	out, err := exec.Command("ps", "-axo", "pid=,command=").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, udid) && strings.HasSuffix(line, "/"+exe) {
			if pid, err := strconv.Atoi(strings.Fields(line)[0]); err == nil {
				return pid
			}
		}
	}
	t.Fatalf("the process of %s was not found", exe)
	return 0
}

// rssMB is the resident memory of the process, from ps.
func rssMB(t *testing.T, pid int) float64 {
	t.Helper()
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		t.Fatalf("ps rss of %d: %v", pid, err)
	}
	kb, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("rss %q: %v", out, err)
	}
	return float64(kb) / 1024
}

// cpuSeconds is the CPU time of the process, from ps's [[dd-]hh:]mm:ss.ss.
func cpuSeconds(t *testing.T, pid int) float64 {
	t.Helper()
	out, err := exec.Command("ps", "-o", "cputime=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		t.Fatalf("ps cputime of %d: %v", pid, err)
	}
	total := 0.0
	for _, part := range strings.Split(strings.TrimSpace(string(out)), ":") {
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			t.Fatalf("cputime %q: %v", out, err)
		}
		total = total*60 + v
	}
	return total
}

// perfRun is what one measurement of an app on the Simulator found.
type perfRun struct {
	name                 string
	exeMB                float64
	wall, sinceRun       time.Duration
	sinceProc            time.Duration
	renderer             string
	rssStartMB, rssEndMB float64
	cpuPercent           float64
	frames               perfstats.FrameSummary
}

// measure builds the package as a release bundle for the Simulator, installs and runs it with Metal and the frame statistics, and
// measures the start to the first frame, then a window of the app that begins 3 s later: the CPU, the resident memory and the frames.
func measure(t *testing.T, h *iostest.Harness, pkg string, window time.Duration) perfRun {
	t.Helper()
	root, err := iostest.ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(pkg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	res, err := iostool.Build(ctx, iostool.BuildOptions{Package: pkg, Dir: root, Out: filepath.Join(t.TempDir(), "App.app"), Simulator: true, ID: "dev.glutter.perf"})
	if err != nil {
		t.Fatalf("build %s: %v", pkg, err)
	}
	info, err := os.Stat(res.Layout.Executable)
	if err != nil {
		t.Fatal(err)
	}
	dev, err := h.Device(ctx, "iphone")
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if err := h.Simctl.Install(ctx, dev.UDID, res.Layout.App); err != nil {
		t.Fatalf("install: %v", err)
	}
	r := perfRun{name: name, exeMB: float64(info.Size()) / (1 << 20)}
	begin := time.Now()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_RENDERER=metal", "GLUTTER_FRAME_STATS=1")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer app.Stop()
	line, err := app.WaitFor("first frame presented", 2*time.Minute)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	r.wall = time.Since(begin)
	kv := perfstats.ParseLogLine(line)
	r.renderer = kv["renderer"]
	r.sinceRun, _ = time.ParseDuration(kv["since_run"])
	r.sinceProc, _ = time.ParseDuration(kv["since_process_start"])
	pid := appPID(t, dev.UDID, filepath.Base(res.Layout.Executable))
	time.Sleep(3 * time.Second)
	cpu0, from, seen := cpuSeconds(t, pid), time.Now(), len(app.Output())
	r.rssStartMB = rssMB(t, pid)
	time.Sleep(window)
	r.cpuPercent = 100 * (cpuSeconds(t, pid) - cpu0) / time.Since(from).Seconds()
	r.rssEndMB = rssMB(t, pid)
	var fr []perfstats.Frames
	// The statistics lines that the app printed in the window (a second with no frame has none).
	for _, l := range app.Output()[seen:] {
		if f, ok := perfstats.ParseFramesLine(l); ok {
			fr = append(fr, f)
		}
	}
	r.frames = perfstats.SummarizeFrames(fr)
	return r
}

// TestSimulatorPerformance measures the counter, the gallery and an animation on the Simulator (release builds, Metal) and checks the
// numbers against limits that catch a regression, not against the targets of a device: the Simulator runs on the Mac's CPU and GPU, its
// display links do not fire when it has no screen (the pacing is a timer, SPEC §30.2), and a Simulator app is a process of the Mac, so
// the numbers are the Mac's. GLUTTER_IOS_PERF_REPORT=<file> writes them as the table of docs/PERFORMANCE.md (step 35.9).
func TestSimulatorPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and measures three apps; skipped by -short")
	}
	h := iostest.New(t)
	var rows []perfRun
	for _, pkg := range []string{"./examples/counter", "./examples/gallery", "./examples/animations"} {
		rows = append(rows, measure(t, h, pkg, 10*time.Second))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "| App | Executable | Start to the first frame (wall, since Run, since process start) | Memory at 3 s and at 13 s | CPU in 10 s | Frames a second (mean, min, max) | Build and raster (mean) | Frames over 16 ms |\n|---|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		proc := "not known"
		if r.sinceProc > 0 {
			proc = r.sinceProc.String()
		}
		fmt.Fprintf(&b, "| %s | %.1f MB | %v, %v, %v | %.0f MB and %.0f MB | %.0f %% | %.1f, %d, %d | %v and %v | %.1f %% |\n", r.name, r.exeMB,
			r.wall.Round(10*time.Millisecond), r.sinceRun, proc, r.rssStartMB, r.rssEndMB, r.cpuPercent, r.frames.MeanFPS, r.frames.MinFPS, r.frames.MaxFPS,
			r.frames.BuildAvg.Round(10*time.Microsecond), r.frames.RasterAvg.Round(10*time.Microsecond), r.frames.Over16Percent)
	}
	t.Logf("\n%s", b.String())
	if path := os.Getenv("GLUTTER_IOS_PERF_REPORT"); path != "" {
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Errorf("the report: %v", err)
		}
	}
	for _, r := range rows {
		if !strings.HasPrefix(r.renderer, "metal") {
			t.Errorf("%s: drawn by %q, want metal", r.name, r.renderer)
		}
		if r.sinceRun > 10*time.Second {
			t.Errorf("%s: %v from Run to the first frame", r.name, r.sinceRun)
		}
		if r.rssEndMB > 600 {
			t.Errorf("%s: %.0f MB resident", r.name, r.rssEndMB)
		}
	}
	// The counter draws nothing while nothing changes; the animation draws on every tick.
	if rows[0].frames.Frames > 30 {
		t.Errorf("the idle counter drew %d frames in 10 s", rows[0].frames.Frames)
	}
	if rows[2].frames.MeanFPS < 20 {
		t.Errorf("the animation drew %.1f frames a second", rows[2].frames.MeanFPS)
	}
}

// TestAnimatingAppMemoryStaysFlat runs an app that animates for ever with Metal for GLUTTER_IOS_SOAK seconds (60 by default) and reads
// its resident memory every 5 s after the first 15: the memory after the soak is not more than 30 MB above the memory at 15 s, and the
// Go heap in the app's own statistics does not grow either (SPEC §15, "Memory over time").
func TestAnimatingAppMemoryStaysFlat(t *testing.T) {
	if testing.Short() {
		t.Skip("soaks an app; skipped by -short")
	}
	soak := 60 * time.Second
	if s := os.Getenv("GLUTTER_IOS_SOAK"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			soak = time.Duration(n) * time.Second
		}
	}
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/spinner")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_RENDERER=metal", "GLUTTER_FRAME_STATS=1")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := app.WaitFor("first frame presented", 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	pid := appPID(t, dev.UDID, "spinner")
	time.Sleep(15 * time.Second)
	base := rssMB(t, pid)
	var samples []float64
	for start := time.Now(); time.Since(start) < soak; time.Sleep(5 * time.Second) {
		samples = append(samples, rssMB(t, pid))
	}
	last := samples[len(samples)-1]
	t.Logf("resident memory: %.0f MB at 15 s, then %v", base, samples)
	if last-base > 30 {
		t.Errorf("the resident memory grew from %.0f MB to %.0f MB in %v", base, last, soak)
	}
	var heaps []int
	for _, l := range app.Output() {
		if f, ok := perfstats.ParseFramesLine(l); ok {
			heaps = append(heaps, f.GoHeapMB)
		}
	}
	if len(heaps) < 10 {
		t.Fatalf("only %d statistics lines", len(heaps))
	}
	if first, lastHeap := heaps[5], heaps[len(heaps)-1]; lastHeap-first > 10 {
		t.Errorf("the Go heap grew from %d MB to %d MB", first, lastHeap)
	}
}
