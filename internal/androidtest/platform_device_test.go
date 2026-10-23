//go:build glutter_android

package androidtest

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/png"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
	"github.com/rm4n0s/glutter/internal/goldens"
)

// decodePNG turns the bytes of a PNG into an RGBA image.
func decodePNG(t *testing.T, b []byte) *image.RGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("not a PNG (%d bytes): %v\n%s", len(b), err, b[:min(len(b), 200)])
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return rgba
}

// installApp builds a package of the module for the emulator and installs it.
func installApp(ctx context.Context, t *testing.T, e *Emulator, pkg string) string {
	t.Helper()
	apk, appID, err := e.BuildAPK(ctx, pkg, t.TempDir(), androidtool.BuildOptions{})
	if err != nil {
		t.Fatal(Describe(err))
	}
	if err := e.Install(ctx, apk); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

func shell(ctx context.Context, t *testing.T, e *Emulator, args ...string) string {
	t.Helper()
	out, err := e.ADB.Output(ctx, append([]string{"shell"}, args...)...)
	if err != nil {
		t.Fatalf("adb shell %v: %v", args, err)
	}
	return out
}

// TestCounterFirstFrameIsItsSoftGolden checks that the first frame of examples/counter, drawn on the
// emulator by the soft renderer into the window, is the golden that the same app gives on the host
// (step 25.3).
func TestCounterFirstFrameIsItsSoftGolden(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/counter")
	b, err := e.Screenshot(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft"}, time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	got := decodePNG(t, b)
	mod, err := ModuleRoot()
	if err != nil {
		t.Fatal(Describe(err))
	}
	want, err := goldens.ReadPNG(filepath.Join(mod, "examples", "counter", "testdata", "goldens", "counter-android.png"))
	if err != nil {
		t.Fatal(Describe(err))
	}
	if res := goldens.Compare(want, got, 0); !res.OK() {
		t.Errorf("the first frame differs from the golden: size mismatch %v, %d pixels, largest difference %d (got %v, want %v)",
			res.SizeMismatch, res.Pixels, res.MaxDiff, got.Rect, want.Rect)
	}
}

// TestTurningTheDeviceGivesAFrameOfTheNewSize rotates the emulator while the app runs: the activity
// is not recreated (configChanges), the surface changes size, and the frame has the new size.
func TestTurningTheDeviceGivesAFrameOfTheNewSize(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/counter")
	shell(ctx, t, e, "settings", "put", "system", "accelerometer_rotation", "1")
	t.Cleanup(func() {
		// Back to portrait, whatever happened.
		_, _ = e.ADB.Output(context.Background(), "shell", "settings", "put", "system", "accelerometer_rotation", "0")
		_, _ = e.ADB.Output(context.Background(), "shell", "settings", "put", "system", "user_rotation", "0")
	})
	if err := e.ADB.RemoveAppFile(ctx, appID, ScreenshotFile); err != nil {
		t.Fatal(Describe(err))
	}
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_SCREENSHOT": "1", "GLUTTER_SCREENSHOT_AFTER": "9s"}); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.ADB.Output(ctx, "emu", "rotate"); err != nil {
		t.Fatal(Describe(err))
	}
	if err := e.WaitExit(ctx, appID, time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	b, err := e.ADB.PullAppFile(ctx, appID, ScreenshotFile)
	if err != nil {
		t.Fatal(Describe(err))
	}
	got := decodePNG(t, b)
	if got.Rect.Dx() != 2400 || got.Rect.Dy() != 1080 {
		t.Errorf("the frame after turning the device is %v, want 2400x1080", got.Rect)
	}
	log, _ := e.Log(ctx, appID)
	if strings.Contains(log, "FATAL") {
		t.Errorf("the app crashed:\n%s", log)
	}
}

// TestHomeAndBackDrawsAgain sends the app to the background and brings it back: the new surface is
// blank, and the app draws it again.
func TestHomeAndBackDrawsAgain(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/counter")
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft"}); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	goHomeAndBack(ctx, t, e, appID)
	// The text of the counter is in the middle of the screen; the status bar (not covered until step
	// 25.7) is not. A surface that was not drawn again would show nothing there.
	time.Sleep(2 * time.Second)
	b, err := e.ADB.Screencap(ctx)
	if err != nil {
		t.Fatal(Describe(err))
	}
	got := decodePNG(t, b)
	mod, err := ModuleRoot()
	if err != nil {
		t.Fatal(Describe(err))
	}
	want, err := goldens.ReadPNG(filepath.Join(mod, "examples", "counter", "testdata", "goldens", "counter-android.png"))
	if err != nil {
		t.Fatal(Describe(err))
	}
	if got.Rect != want.Rect {
		t.Fatalf("the screen is %v, the golden %v", got.Rect, want.Rect)
	}
	band := image.Rect(0, 1000, 1080, 1400)
	res := goldens.Compare(want.SubImage(band).(*image.RGBA), got.SubImage(band).(*image.RGBA), 0)
	if !res.OK() {
		t.Errorf("after Home and back the middle of the screen differs from the golden in %d pixels (largest difference %d)", res.Pixels, res.MaxDiff)
	}
}

// goHomeAndBack sends the app to the background and brings its activity to the front again, in
// the same process.
func goHomeAndBack(ctx context.Context, t *testing.T, e *Emulator, appID string) {
	t.Helper()
	pid, err := e.ADB.Pid(ctx, appID)
	if err != nil || pid == 0 {
		t.Fatalf("the app has no process before Home: %v", err)
	}
	shell(ctx, t, e, "input", "keyevent", "KEYCODE_HOME")
	time.Sleep(1500 * time.Millisecond)
	if again, _ := e.ADB.Pid(ctx, appID); again != pid {
		t.Fatalf("the process changed in the background: %d, then %d", pid, again)
	}
	shell(ctx, t, e, "am", "start", "-n", appID+"/"+androidtool.ActivityClass)
	time.Sleep(1500 * time.Millisecond)
	if again, _ := e.ADB.Pid(ctx, appID); again != pid {
		t.Fatalf("the process changed when the app came back: %d, then %d", pid, again)
	}
}

// TestSurfaceDestroyedDuringFramesNeitherCrashesNorBlocks sends an app that draws all the time to
// the background and back many times: Android destroys its surface while a frame may be drawn, and
// surfaceDestroyed must wait for the renderer without crashing it or waiting longer than the limit.
func TestSurfaceDestroyedDuringFramesNeitherCrashesNorBlocks(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/animations")
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft"}); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	for i := 0; i < 8; i++ {
		start := time.Now()
		shell(ctx, t, e, "input", "keyevent", "KEYCODE_HOME")
		shell(ctx, t, e, "am", "start", "-n", appID+"/"+androidtool.ActivityClass)
		if d := time.Since(start); d > 20*time.Second {
			t.Errorf("round %d took %v", i, d)
		}
	}
	time.Sleep(time.Second)
	pid, _ := e.ADB.Pid(ctx, appID)
	if pid == 0 {
		t.Fatal("the app died")
	}
	log, err := e.Log(ctx, appID)
	if err != nil {
		t.Fatal(Describe(err))
	}
	full, _ := e.FullLog(ctx)
	if strings.Contains(log, "did not let go of the surface") {
		t.Errorf("surfaceDestroyed waited out the limit:\n%s", log)
	}
	if crashed := appCrashed(full, appID, pid); crashed != "" {
		t.Errorf("the app crashed: %s", crashed)
	}
}

// appCrashed returns the line of the log that says that the app crashed (a Java exception of its
// process, or a fatal signal of its pid), or "". The device has crashes of its own (the emulator has
// no UWB chip), which are not the app's.
func appCrashed(full, appID string, pid int) string {
	for _, l := range strings.Split(full, "\n") {
		if strings.Contains(l, "Process: "+appID+",") ||
			(strings.Contains(l, "Fatal signal") && strings.Contains(l, " pid "+strconv.Itoa(pid)+" (")) {
			return l
		}
	}
	return ""
}
