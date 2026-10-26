//go:build windows && glutter_wine

package engine

import (
	"context"
	"image"
	"image/color"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/winfonts"
	"github.com/rm4n0s/glutter/ui"
)

// TestRunDrawsIntoAWindowAndSavesTheFrame runs the engine on the Win32 platform under Wine
// (SPEC §25.13): the first window opens, the soft renderer draws the scene into it with GDI, and
// the self-capture saves the frame as a PNG and ends Run. Run it with
// GOOS=windows go test -tags glutter_wine -exec scripts/wine-exec.sh -p 1 ./internal/engine
func TestRunDrawsIntoAWindowAndSavesTheFrame(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	shot := filepath.Join(t.TempDir(), "frame.png")
	env := map[string]string{ScreenshotEnv: shot}
	scene := gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 50, 30), gputest.FillPaint(0xFF3366CC))
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := Run(ctx, Config{
		Logger: quiet(), Getenv: func(k string) string { return env[k] },
		Window: platform.WindowConfig{Title: "Engine", Size: ui.Size{Width: 200, Height: 100}},
		Scene:  func(ui.Size, float64) *ui.Scene { return scene },
	})
	if err != nil {
		t.Fatal(err)
	}
	img, err := goldens.ReadPNG(shot)
	if err != nil {
		t.Fatal(err)
	}
	scale := float64(img.Bounds().Dx()) / 200
	if scale < 1 || img.Bounds().Dy() != int(100*scale) {
		t.Fatalf("the frame is %v, want 200x100 logical pixels", img.Bounds().Size())
	}
	at := func(x, y float64) color.RGBA { return img.RGBAAt(int(x*scale), int(y*scale)) }
	if got := at(30, 25); got != (color.RGBA{0x33, 0x66, 0xCC, 255}) {
		t.Errorf("a pixel of the rectangle = %v, want #3366CC", got)
	}
	if got := at(150, 80); got.A != 255 {
		t.Errorf("a pixel outside = %v, want an opaque one", got)
	}
}

// TestSystemFontMatcherIsTheMatcherOfWindows checks that the engine gives the Collection the
// matcher of Windows (SPEC §25.8), here over the fonts of the prefix of Wine, and that a font it
// finds loads.
func TestSystemFontMatcherIsTheMatcherOfWindows(t *testing.T) {
	t.Setenv("LocalAppData", t.TempDir())
	ctx := context.Background()
	m := SystemFontMatcher(ctx, quiet())
	if _, ok := m.(*winfonts.Matcher); !ok {
		t.Fatalf("the matcher is %T, want the matcher of winfonts", m)
	}
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	files, err := c.Match(ctx, "Liberation Sans", 400, false, "")
	if err != nil || len(files) == 0 {
		t.Fatalf("Match(Liberation Sans) = %+v, %v", files, err)
	}
	face, err := c.Face(ctx, files[0])
	if err != nil || face == nil {
		t.Fatalf("Face = %v, %v", face, err)
	}
	families, err := c.Families(ctx)
	if err != nil || len(families) == 0 {
		t.Errorf("Families = %v, %v", families, err)
	}
}

// engineRun is what one run of the engine on Wine left: the frame it saved (nil when it saved none),
// its log, how many frames the scene was asked for, and the error of Run.
type engineRun struct {
	frame  *image.RGBA
	log    string
	scenes int
	err    error
}

// rectangleScene is the scene of the tests of the renderer's choice: a blue rectangle and a red circle over the
// development background.
func rectangleScene() *ui.Scene {
	return gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 60, 40), gputest.FillPaint(0xFF3366CC))
		c.DrawCircle(ui.Offset{Dx: 120, Dy: 60}, 30, gputest.FillPaint(0xFFCC3333))
	})
}

// runEngine runs the engine on the thread of the test with the environment and the self-capture, and
// reads what it saved. The window is 200x100 logical pixels. The shader cache is the one of the prefix
// (SPEC §9.7), which the first run of any test fills.
func runEngine(t *testing.T, env map[string]string) engineRun {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	shot := filepath.Join(t.TempDir(), "frame.png")
	all := map[string]string{ScreenshotEnv: shot}
	for k, v := range env {
		all[k] = v
	}
	log, buf := recordingLog()
	var scenes atomic.Int32
	scene := rectangleScene()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	err := Run(ctx, Config{
		Logger: log, Getenv: func(k string) string { return all[k] },
		Window: platform.WindowConfig{Title: "Engine", Size: ui.Size{Width: 200, Height: 100}},
		Scene:  func(ui.Size, float64) *ui.Scene { scenes.Add(1); return scene },
	})
	out := engineRun{log: buf.String(), scenes: int(scenes.Load()), err: err}
	if err == nil {
		img, rerr := goldens.ReadPNG(shot)
		if rerr != nil {
			t.Fatalf("the engine saved no frame: %v\n%s", rerr, out.log)
		}
		out.frame = img
	}
	return out
}

// chosen is the renderer that the log of a run says was chosen ("" when none was).
func (r engineRun) chosen() string {
	for _, l := range strings.Split(r.log, "\n") {
		if strings.Contains(l, "renderer chosen") {
			return l
		}
	}
	return ""
}

// TestWineAutoChoosesDirect3D12AndItsFrameEqualsSoft: with nothing asked for, the engine draws with d3d12 on
// the hardware adapter of the prefix (vkd3d's), logs it, and the frame it saves is soft's within the tolerance
// of the GPU renderers (SPEC §9.6, §9.8). Run it with
// GOOS=windows go test -tags glutter_wine -exec scripts/wine-exec.sh -p 1 ./internal/engine
func TestWineAutoChoosesDirect3D12AndItsFrameEqualsSoft(t *testing.T) {
	gpuRun := runEngine(t, nil)
	if gpuRun.err != nil {
		t.Fatalf("Run: %+v\n%s", gpuRun.err, gpuRun.log)
	}
	if line := gpuRun.chosen(); !strings.Contains(line, "d3d12:") {
		t.Errorf("the renderer chosen is %q, want d3d12\n%s", line, gpuRun.log)
	}
	if strings.Contains(gpuRun.log, "renderer not usable") {
		t.Errorf("a renderer was given up although d3d12 works:\n%s", gpuRun.log)
	}
	softRun := runEngine(t, map[string]string{RendererEnv: "soft"})
	if softRun.err != nil {
		t.Fatalf("Run with soft: %+v\n%s", softRun.err, softRun.log)
	}
	if line := softRun.chosen(); !strings.Contains(line, "soft: GDI") {
		t.Errorf("GLUTTER_RENDERER=soft chose %q\n%s", line, softRun.log)
	}
	if gpuRun.frame.Bounds() != softRun.frame.Bounds() {
		t.Fatalf("the frames are %v and %v", gpuRun.frame.Bounds(), softRun.frame.Bounds())
	}
	gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, "d3d12 against soft", gputest.DiffRect(gpuRun.frame, softRun.frame, gpuRun.frame.Bounds()))
	scale := float64(gpuRun.frame.Bounds().Dx()) / 200
	if got := gpuRun.frame.RGBAAt(int(30*scale), int(25*scale)); got != (color.RGBA{0x33, 0x66, 0xCC, 255}) {
		t.Errorf("a pixel of the rectangle = %v, want #3366CC", got)
	}
}

// TestWineAutoWithoutTheAdapterEndsOnSoft: GLUTTER_D3D12_ADAPTER names an adapter that is not there (or
// WARP, which Wine does not have), so d3d12 fails its first frame: auto logs the failure once with its
// tag, leaves WARP out (the variable names an adapter), and ends on soft, whose frame is drawn.
func TestWineAutoWithoutTheAdapterEndsOnSoft(t *testing.T) {
	for _, adapter := range []string{"no-such-adapter-9", "warp", "7"} {
		t.Run(adapter, func(t *testing.T) {
			r := runEngine(t, map[string]string{"GLUTTER_D3D12_ADAPTER": adapter})
			if r.err != nil {
				t.Fatalf("Run: %+v\n%s", r.err, r.log)
			}
			if n := strings.Count(r.log, "renderer not usable"); n != 1 {
				t.Errorf("%d failed tries in the log, want 1:\n%s", n, r.log)
			}
			if !strings.Contains(r.log, "tag=D3D12NoAdapter") || !strings.Contains(r.log, "renderer=d3d12 ") {
				t.Errorf("the log does not say that d3d12 had no adapter:\n%s", r.log)
			}
			if line := r.chosen(); !strings.Contains(line, "soft: GDI") {
				t.Errorf("the renderer chosen is %q, want soft\n%s", line, r.log)
			}
			scale := float64(r.frame.Bounds().Dx()) / 200
			if got := r.frame.RGBAAt(int(30*scale), int(25*scale)); got != (color.RGBA{0x33, 0x66, 0xCC, 255}) {
				t.Errorf("the frame of soft: a pixel of the rectangle = %v, want #3366CC", got)
			}
		})
	}
}

// TestWineD3D12ByNameNeverFallsBack: with GLUTTER_RENDERER=d3d12 a missing adapter ends Run with the tag of
// the device, not with a frame of another renderer (SPEC §9.6).
func TestWineD3D12ByNameNeverFallsBack(t *testing.T) {
	for _, adapter := range []string{"no-such-adapter-9", "warp"} {
		t.Run(adapter, func(t *testing.T) {
			r := runEngine(t, map[string]string{RendererEnv: "d3d12", "GLUTTER_D3D12_ADAPTER": adapter})
			appErr, ok := errors.FromError(r.err)
			if !ok || appErr.Tag != "D3D12NoAdapter" || !appErr.HasRoute("Device.openDevice") {
				t.Fatalf("Run = %+v, want D3D12NoAdapter from Device.openDevice\n%s", r.err, r.log)
			}
			if strings.Contains(r.log, "renderer chosen") || strings.Contains(r.log, "renderer not usable") {
				t.Errorf("a renderer was chosen or given up:\n%s", r.log)
			}
		})
	}
}

// TestWineRenderersThatWindowsDoesNotHave: gles and vulkan are RendererUnavailable on Windows, and a name that
// is nobody's is RendererUnknown, both before the display is touched (SPEC §9.6).
func TestWineRenderersThatWindowsDoesNotHave(t *testing.T) {
	tests := []struct{ renderer, tag string }{
		{"gles", "RendererUnavailable"}, {"vulkan", "RendererUnavailable"}, {"webgpu", "RendererUnavailable"},
		{"webgl", "RendererUnavailable"}, {"direct3d", "RendererUnknown"},
	}
	for _, tt := range tests {
		t.Run(tt.renderer, func(t *testing.T) {
			r := runEngine(t, map[string]string{RendererEnv: tt.renderer})
			appErr, ok := errors.FromError(r.err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute("Run->newChooser."+tt.tag) {
				t.Fatalf("Run = %+v, want %s from newChooser", r.err, tt.tag)
			}
			if r.scenes != 0 {
				t.Errorf("the scene was drawn %d times before the renderer was known to be missing", r.scenes)
			}
		})
	}
}

// activateWindow makes the engine's window with the title the foreground window, as the user would, and
// waits until it is. It reports false when it is not one after ten seconds or when ctx ends first.
func activateWindow(ctx context.Context, title string) bool {
	deadline := time.Now().Add(10 * time.Second)
	for ctx.Err() == nil && time.Now().Before(deadline) {
		if hwnd := win32.FindWindow("GlutterWindow", title); hwnd != 0 {
			win32.SetForegroundWindow(hwnd)
			if win32.GetForegroundWindow() == hwnd {
				return true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// TestWineAnIdleD3D12AppAsksForNoFrames: once the first frames are drawn, an app that changes nothing is not
// asked for another one: the scene is not built again and the swap chain presents nothing, for seconds
// (SPEC §25.3, §15).
func TestWineAnIdleD3D12AppAsksForNoFrames(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	log, buf := recordingLog()
	var scenes atomic.Int32
	scene := rectangleScene()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	type counts struct{ first, settled, end int32 }
	done := make(chan counts, 1)
	go func() {
		var c counts
		for scenes.Load() == 0 && ctx.Err() == nil {
			time.Sleep(20 * time.Millisecond)
		}
		c.first = scenes.Load()
		// The first frames come before the window is active: Wine's X11 driver activates a new window on
		// its own time (about four seconds after the window is created, on a headless server with no window manager), and the
		// focus event that follows is input, which asks for a frame. That frame is not an idle app
		// asking for one, so the measure starts once the window is the active one.
		if !activateWindow(ctx, "Idle") {
			t.Errorf("the window \"Idle\" did not become the foreground window")
		}
		time.Sleep(time.Second) // the frame after the focus, and whatever else is settling
		c.settled = scenes.Load()
		time.Sleep(3 * time.Second)
		c.end = scenes.Load()
		cancel()
		done <- c
	}()
	err := Run(ctx, Config{
		Logger: log, Getenv: func(string) string { return "" },
		Window: platform.WindowConfig{Title: "Idle", Size: ui.Size{Width: 200, Height: 100}},
		Scene:  func(ui.Size, float64) *ui.Scene { scenes.Add(1); return scene },
	})
	if err != nil {
		t.Fatalf("Run: %+v\n%s", err, buf.String())
	}
	c := <-done
	if !strings.Contains(buf.String(), "d3d12:") {
		t.Fatalf("the app was not drawn by d3d12:\n%s", buf.String())
	}
	t.Logf("frames: %d at the first, %d after a second, %d after four", c.first, c.settled, c.end)
	if c.end != c.settled || c.settled > 3 {
		t.Errorf("an idle app drew %d frames in 3 seconds after settling at %d (%d at the first)", c.end-c.settled, c.settled, c.first)
	}
}
