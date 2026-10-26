//go:build linux && glutter_wine

package main

import (
	"bytes"
	"context"
	"debug/pe"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// These tests run glutter-windows itself: they build the counter and a probe program for Windows,
// start them under Wine on a headless X server, and read what they report (SPEC §25.13, step 28.14).
// Run them with `go test -tags glutter_wine -p 1 ./cmd/glutter-windows`.

const (
	counterPackage = "github.com/rm4n0s/glutter/examples/counter"
	probePackage   = "github.com/rm4n0s/glutter/cmd/glutter-windows/testdata/probe"
)

// wineHarness makes the harness and registers the check that Wine is stopped and nothing is left
// running when the test ends.
func wineHarness(t *testing.T) *winetest.Harness {
	t.Helper()
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := h.Stop(stop); err != nil {
			t.Errorf("Stop: %v", err)
		}
		if left := winetest.Leftovers(h.Root); len(left) > 0 {
			t.Errorf("processes left running:\n%s", strings.Join(left, "\n"))
		}
	})
	return h
}

// glutterWindows runs a command of glutter-windows in process and returns what it printed.
func glutterWindows(t *testing.T, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	var out bytes.Buffer
	err := run(ctx, args, &out, &out)
	return out.String(), err
}

// buildWith builds the package with `glutter-windows build` and returns the .exe.
func buildWith(t *testing.T, pkg string, flags ...string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "app.exe")
	out, err := glutterWindows(t, append(append([]string{"build", "-o", exe}, flags...), pkg)...)
	if err != nil {
		t.Fatalf("glutter-windows build: %v\n%s", err, out)
	}
	return exe
}

// goBuild builds the package with plain `go build`, with no manifest and no resources.
func goBuild(t *testing.T, h *winetest.Harness, pkg string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "plain.exe")
	out, err := wintool.ExecRunner{}.Run(context.Background(), wintool.Command{
		Dir: h.Root, Name: "go", Args: []string{"build", "-o", exe, pkg},
		Env: append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0"),
	})
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return exe
}

// probeFacts runs the probe under Wine and returns its "key: value" lines.
func probeFacts(t *testing.T, h *winetest.Harness, exe string, args ...string) map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	res, err := h.Run(ctx, exe, winetest.RunOptions{Args: args, Timeout: 3 * time.Minute})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("the probe: %v, status %d\n%s", err, res.ExitCode, res.Output)
	}
	facts := map[string]string{}
	for _, line := range strings.Split(res.Output, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ": "); ok {
			facts[k] = v
		}
	}
	return facts
}

// subsystem is the subsystem of the .exe: 2 for a GUI program, 3 for a console program.
func subsystem(t *testing.T, exe string) uint16 {
	t.Helper()
	f, err := pe.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }() // read-only
	oh, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok {
		t.Fatalf("optional header is %T", f.OptionalHeader)
	}
	return oh.Subsystem
}

// TestWineManifestSetsAwarenessCodePageAndIcon builds the probe with glutter-windows and runs it
// under Wine: the manifest makes the process per-monitor DPI aware (v2) before any code runs, gives it
// the UTF-8 code page, and the icon and the version information are in the .exe: LoadImageW of
// resource 1 returns an icon of each requested size, and the file's version is the one asked for.
func TestWineManifestSetsAwarenessCodePageAndIcon(t *testing.T) {
	h := wineHarness(t)
	exe := buildWith(t, probePackage, "-version", "1.2.3.4", "-name", "Glutter Probe")
	facts := probeFacts(t, h, exe)
	if facts["dpi-start"] != "PerMonitorV2" {
		t.Errorf("dpi-start = %q, want PerMonitorV2 (the manifest applies before the first line of the program)", facts["dpi-start"])
	}
	if facts["acp"] != "65001" {
		t.Errorf("acp = %q, want 65001", facts["acp"])
	}
	for _, n := range wintool.IconSizes {
		want := strconv.Itoa(n) + "x" + strconv.Itoa(n)
		if got := facts["icon-"+strconv.Itoa(n)]; got != want {
			t.Errorf("icon-%d = %q, want %q", n, got, want)
		}
	}
	if want := "1.2.3.4 Glutter Probe"; facts["version"] != want {
		t.Errorf("version = %q, want %q", facts["version"], want)
	}
	t.Logf("the probe says: %v", facts)
}

// TestWinePlainBuildStillRunsAndSetsAwareness builds the probe with plain `go build`: it has no
// manifest, so it starts unaware of the DPI, with the system code page and no icon of its own; it
// still runs, and the platform of Glutter makes it per-monitor DPI aware (v2) when it starts
// (SPEC §25.1).
func TestWinePlainBuildStillRunsAndSetsAwareness(t *testing.T) {
	h := wineHarness(t)
	exe := goBuild(t, h, probePackage)
	facts := probeFacts(t, h, exe, "platform")
	if facts["dpi-start"] == "PerMonitorV2" {
		t.Errorf("dpi-start = PerMonitorV2: the plain build has a manifest")
	}
	if facts["dpi-platform"] != "PerMonitorV2" {
		t.Errorf("dpi-platform = %q, want PerMonitorV2: the platform sets it at run time", facts["dpi-platform"])
	}
	if facts["acp"] == "65001" {
		t.Errorf("acp = 65001: the plain build has the manifest's UTF-8 code page")
	}
	if got := facts["icon-32"]; !strings.Contains(got, "failed") {
		t.Errorf("icon-32 = %q, want the failure of LoadImageW: the plain build has no icon", got)
	}
	t.Logf("the plain build says: %v", facts)
}

// TestGUIAndConsoleSubsystems: a build is a GUI program (it opens no console window) unless
// -console is given, and the header of the .exe says so.
func TestGUIAndConsoleSubsystems(t *testing.T) {
	gui := buildWith(t, counterPackage)
	console := buildWith(t, counterPackage, "-console")
	if got := subsystem(t, gui); got != 2 {
		t.Errorf("the default build has subsystem %d, want 2 (Windows GUI)", got)
	}
	if got := subsystem(t, console); got != 3 {
		t.Errorf("the -console build has subsystem %d, want 3 (Windows console)", got)
	}
}

// TestWineBuiltCounterShotEqualsGolden: `glutter-windows shot` of the counter saves the frame that the
// built .exe draws under Wine. The regions without text equal the golden of the soft renderer, pixel for
// pixel, when the app uses soft, and are within the tolerance of the GPU renderers when it uses d3d12 (the
// default); the text is Liberation Sans standing for Segoe UI, so it only has to be there.
func TestWineBuiltCounterShotEqualsGolden(t *testing.T) {
	h := wineHarness(t)
	want, err := goldens.ReadPNG(filepath.Join(h.Root, "examples", "counter", "testdata", "goldens", "counter.png"))
	if err != nil {
		t.Fatal(err)
	}
	shot := func(name string, flags ...string) *image.RGBA {
		path := filepath.Join(t.TempDir(), name+".png")
		out, err := glutterWindows(t, append(append([]string{"shot", "-o", path}, flags...), counterPackage)...)
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
		got, err := goldens.ReadPNG(path)
		if err != nil {
			t.Fatalf("%s saved no frame: %v\n%s", name, err, out)
		}
		if got.Bounds() != want.Bounds() {
			t.Fatalf("%s: the frame is %v, the golden %v", name, got.Bounds(), want.Bounds())
		}
		return got
	}
	flat := []image.Rectangle{image.Rect(390, 550, 480, 640), image.Rect(0, 0, 480, 10), image.Rect(0, 400, 380, 640)}
	soft := shot("soft", "-env", "GLUTTER_RENDERER=soft")
	for _, r := range flat {
		if n := gputest.DiffRect(want, soft, r); n.Over8 != 0 || n.Max != 0 {
			t.Errorf("soft: %v differs from the golden: %+v", r, n)
		}
	}
	gpu := shot("d3d12")
	for _, r := range flat {
		gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, "d3d12 "+r.String(), gputest.DiffRect(want, gpu, r))
	}
	for _, img := range []*image.RGBA{soft, gpu} {
		for _, r := range []image.Rectangle{image.Rect(0, 0, 480, 56), image.Rect(60, 310, 420, 380)} {
			dark := 0
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					if c := img.RGBAAt(x, y); int(c.R)+int(c.G)+int(c.B) < 300 {
						dark++
					}
				}
			}
			if dark < 50 {
				t.Errorf("only %d dark pixels in %v: the text is missing", dark, r)
			}
		}
	}
}

// TestWineShotScreenSavesTheRootWindow: `shot -screen` saves the whole screen of the X server (1280x800)
// with the counter's window on it.
func TestWineShotScreenSavesTheRootWindow(t *testing.T) {
	wineHarness(t)
	path := filepath.Join(t.TempDir(), "screen.png")
	out, err := glutterWindows(t, "shot", "-screen", "-settle", "4s", "-o", path, counterPackage)
	if err != nil {
		t.Fatalf("shot -screen: %v\n%s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 1280 || b.Dy() != 800 {
		t.Errorf("the screen is %v, want 1280x800", b)
	}
	// The counter's window is white and blue-ish on the black root window: some pixels are light.
	light := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 4 {
		for x := b.Min.X; x < b.Max.X; x += 4 {
			if r, g, bl, _ := img.At(x, y).RGBA(); (r+g+bl)>>8 > 600 {
				light++
			}
		}
	}
	if light < 500 {
		t.Errorf("only %d light samples: the window of the counter is not on the screen", light)
	}
}

// TestWineRunShowsTheOutputAndLeavesNothingBehind: `run -headless` builds and starts the counter,
// shows its log when it ends, and stops Wine and the server; a run that is cut by -timeout fails with
// the status of `timeout` and leaves nothing either.
func TestWineRunShowsTheOutputAndLeavesNothingBehind(t *testing.T) {
	h := wineHarness(t)
	frame := filepath.Join(t.TempDir(), "run.png")
	out, err := glutterWindows(t, "run", "-headless", "-env", "GLUTTER_SCREENSHOT="+wintool.WineLauncher{}.ProgramPath(frame), counterPackage)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, want := range []string{"built " + counterPackage, "running " + counterPackage, `renderer chosen renderer="`} {
		if !strings.Contains(out, want) {
			t.Errorf("the output has no %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(frame); err != nil {
		t.Errorf("the app did not save its frame: %v", err)
	}
	if left := winetest.Leftovers(h.Root); len(left) > 0 {
		t.Errorf("processes left running after run:\n%s", strings.Join(left, "\n"))
	}

	// An app that does not end is cut by the timeout.
	out, err = glutterWindows(t, "run", "-headless", "-timeout", "5s", counterPackage)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("runRun->Run.WindowsRunFailed") || appErr.Metadata["exitCode"] != 124 {
		t.Errorf("got %v, want WindowsRunFailed from runRun->Run with exitCode 124\n%s", err, out)
	}
	if left := winetest.Leftovers(h.Root); len(left) > 0 {
		t.Errorf("processes left running after a run that timed out:\n%s", strings.Join(left, "\n"))
	}
}
