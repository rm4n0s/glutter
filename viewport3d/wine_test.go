//go:build windows && glutter_wine

package viewport3d_test

import (
	"image"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/d3d12"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/scenetest"
)

// checkAgainstGolden compares the picture of a case on a backend with the golden, which lavapipe's Vulkan made, within the
// tolerance (SPEC §27.12).
func checkAgainstGolden(t *testing.T, name string, got *image.RGBA, tol gputest.Tolerance) {
	t.Helper()
	want, err := goldens.ReadPNG(filepath.Join("testdata", "goldens", name+".png"))
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	if got.Bounds() != want.Bounds() {
		t.Errorf("%s: the picture is %v, the golden %v", name, got.Bounds(), want.Bounds())
		return
	}
	tol.Check(t, name, gputest.DiffRect(got, want, want.Bounds()))
}

// TestWineScenesAgainstTheirGoldens draws the scenes of package scenetest on Direct3D 12 under Wine (vkd3d over lavapipe) and
// compares each picture with the golden that Vulkan made, within the cross-backend tolerance (SPEC §27.12). The device is used
// from the thread that made it.
func TestWineScenesAgainstTheirGoldens(t *testing.T) {
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	dev, err := d3d12.Open(d3d12.Options{Debug: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Open: %+v", err)
	}
	t.Cleanup(func() {
		if errs := dev.DebugErrors(); len(errs) != 0 {
			t.Errorf("the debug layer reported %d errors: %q", len(errs), errs)
		}
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	app := gpuhost.NewBinding(t, dev)
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range scenetest.Cases() {
		img, err := scenetest.Render(d, waitFor(app), c)
		if err != nil {
			t.Errorf("%s: %v", c.Name, err)
			continue
		}
		checkAgainstGolden(t, c.Name, img, gputest.ScenesTolerance)
	}
}
