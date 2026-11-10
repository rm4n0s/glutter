//go:build glutter_ios

package iostest_test

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/scenelines"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// installed builds the package for the Simulator, installs it on an iPhone and returns what a test needs to launch it.
func installed(t *testing.T, h *iostest.Harness, pkg string) (iostool.Device, iostool.BuildResult) {
	t.Helper()
	root, err := iostest.ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	res, err := iostool.Build(ctx, iostool.BuildOptions{
		Package: pkg, Dir: root, Out: filepath.Join(t.TempDir(), "App.app"), Simulator: true, ID: "dev.glutter.scenetest",
		Orientations: []string{iostool.OrientationPortrait, iostool.OrientationLandscapeLeft, iostool.OrientationLandscapeRight},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	dev, err := h.Device(ctx, "iphone")
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if err := h.Simctl.Install(ctx, dev.UDID, res.Layout.App); err != nil {
		t.Fatalf("install: %v", err)
	}
	return dev, res
}

// count is how many lines of the app's output contain the text.
func count(l *iostest.Launched, text string) int {
	n := 0
	for _, line := range l.Output() {
		if strings.Contains(line, text) {
			n++
		}
	}
	return n
}

// sizeAt reads the "<W>x<H>" that is the field of the line after its prefix (0 for the metrics, 1 for a frame, which has its number
// first).
func sizeAt(t *testing.T, line, prefix string, field int) (w, h int) {
	t.Helper()
	fields := strings.Fields(strings.TrimPrefix(line, prefix))
	if field >= len(fields) {
		t.Fatalf("no size in %q", line)
	}
	if _, err := fmt.Sscanf(fields[field], "%dx%d", &w, &h); err != nil {
		t.Fatalf("the size of %q: %v", line, err)
	}
	return w, h
}

// TestSceneShowsMetricsFramesAndRotates runs the scene app on an iPhone: the first metrics are a portrait size with a safe area and a
// scale, a frame of that size follows, and when the app asks for the landscape orientations the scene turns, the metrics say so and
// the next frame has the size of the turned view (SPEC §30.4, §30.11).
func TestSceneShowsMetricsFramesAndRotates(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/sceneapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, scenelines.RotateEnv+"=landscape")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	line, err := app.WaitFor(scenelines.Metrics, time.Minute)
	if err != nil {
		t.Fatalf("the first metrics: %v", err)
	}
	pw, ph := sizeAt(t, line, scenelines.Metrics, 0)
	if pw >= ph {
		t.Errorf("the first metrics %q are not portrait", line)
	}
	if !strings.Contains(line, "scale=2") && !strings.Contains(line, "scale=3") {
		t.Errorf("the scale of %q is not 2 or 3", line)
	}
	if strings.HasSuffix(line, "padding=0,0,0,0") {
		t.Errorf("an iPhone has a safe area, but the padding of %q is none", line)
	}
	if _, err := app.WaitFor(scenelines.Frame+" 1 "+fmt.Sprintf("%dx%d", pw, ph), time.Minute); err != nil {
		t.Fatalf("the first frame at the portrait size: %v", err)
	}
	// The scene turns: the metrics are landscape, and a frame has that size.
	var lw, lh int
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		for _, l := range app.Output() {
			if strings.HasPrefix(l, scenelines.Metrics) {
				if w, hgt := sizeAt(t, l, scenelines.Metrics, 0); w > hgt {
					lw, lh = w, hgt
				}
			}
		}
		if lw != 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lw == 0 {
		t.Fatalf("the scene did not turn; output:\n%s", strings.Join(app.Output(), "\n"))
	}
	if lw != ph || lh != pw {
		// The safe area does not change the pixel size of the view: a turn swaps the sides.
		t.Errorf("landscape size %dx%d, want %dx%d", lw, lh, ph, pw)
	}
	var found bool
	for end := time.Now().Add(time.Minute); time.Now().Before(end) && !found; time.Sleep(100 * time.Millisecond) {
		for _, l := range app.Output() {
			if strings.HasPrefix(l, scenelines.Frame) && strings.HasSuffix(l, fmt.Sprintf(" %dx%d", lw, lh)) {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("no frame at the landscape size %dx%d; output:\n%s", lw, lh, strings.Join(app.Output(), "\n"))
	}
}

// TestSceneInTheBackgroundAsksForNoFrame sends the scene app to the background by opening Settings: the lifecycle says paused and no
// frame is drawn while it is there; when the app comes back the lifecycle says resumed and the frames go on (SPEC §30.7, §30.11).
func TestSceneInTheBackgroundAsksForNoFrame(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/sceneapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := app.WaitFor(scenelines.Lifecycle+" resumed", time.Minute); err != nil {
		t.Fatalf("resumed: %v", err)
	}
	if _, err := app.WaitFor(scenelines.Frame+" 3 ", time.Minute); err != nil {
		t.Fatalf("frames: %v", err)
	}
	open := func(id string) {
		t.Helper()
		out, err := exec.CommandContext(ctx, "xcrun", h.Simctl.LaunchArgs(dev.UDID, id, false)...).CombinedOutput()
		if err != nil {
			t.Fatalf("launching %s: %v\n%s", id, err, out)
		}
	}
	open("com.apple.Preferences")
	if _, err := app.WaitFor(scenelines.Lifecycle+" paused", time.Minute); err != nil {
		t.Fatalf("paused: %v", err)
	}
	before := count(app, scenelines.Frame)
	time.Sleep(2 * time.Second)
	if after := count(app, scenelines.Frame); after > before+1 { // one may have been in flight
		t.Errorf("%d frames were drawn in the background", after-before)
	}
	open(res.Info.ID)
	if _, err := app.WaitFor(scenelines.Lifecycle+" resumed", time.Minute); err != nil {
		t.Fatalf("resumed again: %v", err)
	}
	resumedAt := count(app, scenelines.Frame)
	deadline := time.Now().Add(time.Minute)
	for count(app, scenelines.Frame) < resumedAt+3 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if count(app, scenelines.Frame) < resumedAt+3 {
		t.Errorf("the frames did not go on after the app came back; output:\n%s", strings.Join(app.Output(), "\n"))
	}
}

// TestCounterDrawsItsFirstFrame runs examples/counter with the engine's self-capture: the first frame that the soft renderer drew into
// the IOSurfaces of the view is saved as a PNG in the app's container, as large as the iPhone's screen in pixels and not blank.
func TestCounterDrawsItsFirstFrame(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./examples/counter")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	container, err := h.Simctl.Container(ctx, dev.UDID, res.Info.ID)
	if err != nil {
		t.Fatalf("container: %v", err)
	}
	shot := filepath.Join(container, "Documents", "first-frame.png")
	if _, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_SCREENSHOT="+shot); err != nil {
		t.Fatalf("launch: %v", err)
	}
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if info, err := os.Stat(shot); err == nil && info.Size() > 0 {
			time.Sleep(200 * time.Millisecond) // the file is written whole, then the app exits
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the app did not save its first frame")
		}
		time.Sleep(200 * time.Millisecond)
	}
	f, err := os.Open(shot)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("the saved frame: %v", err)
	}
	b := img.Bounds()
	if b.Dx() >= b.Dy() || b.Dx() < 600 {
		t.Errorf("the frame is %dx%d: not a phone's portrait screen", b.Dx(), b.Dy())
	}
	colors := map[[3]uint32]bool{}
	for y := b.Min.Y; y < b.Max.Y; y += 7 {
		for x := b.Min.X; x < b.Max.X; x += 7 {
			r, g, bl, _ := img.At(x, y).RGBA()
			colors[[3]uint32{r, g, bl}] = true
		}
	}
	if len(colors) < 8 {
		t.Errorf("the frame has %d colors: it is blank", len(colors))
	}
}
