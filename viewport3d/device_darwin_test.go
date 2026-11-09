//go:build darwin && !ios && glutter_display

package viewport3d_test

import (
	"image"
	"image/draw"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	vshaders "github.com/rm4n0s/glutter/viewport3d/shaders"
)

// openDevice is the device of the display tests: the Mac's Metal device, which the tests run with MTL_DEBUG_LAYER=1 (SPEC
// §29.5). A Mac without one skips the test. The test keeps its OS thread, as a device is used from one.
func openDevice(t *testing.T) explicit.Device {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	d, err := metal.Open(metal.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Skipf("no Metal device: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

// deviceApp is the app of the display tests: its GPU device is the Mac's.
func deviceApp(t *testing.T) (*gpuhost.App, *gpu.Device) {
	t.Helper()
	app := gpuhost.NewBinding(t, openDevice(t))
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return app, d
}

// metalTolerance is how far a picture of Metal may be from the golden that lavapipe made: the two GPUs rasterize the
// edges of triangles and lines a little differently (a handful of pixels along an edge are off by a lot, the fill is
// the same), and round the shading differently by a level or two (SPEC §29.5).
var metalTolerance = gputest.Tolerance{Over8Permille: 3, Over32Permille: 1, Over96: 100}

// checkGolden compares the picture with its golden within metalTolerance. The goldens are made on Linux with
// -update-goldens, never on a Mac.
func checkGolden(t keepGoing, name string, img image.Image) {
	t.Helper()
	want, err := goldens.ReadPNG(filepath.Join("testdata", "goldens", name+".png"))
	if err != nil {
		t.Errorf("golden %s: %v", name, err)
		return
	}
	got := image.NewRGBA(img.Bounds())
	draw.Draw(got, got.Bounds(), img, img.Bounds().Min, draw.Src)
	if got.Bounds() != want.Bounds() {
		t.Errorf("golden %s: the picture is %v, the golden %v", name, got.Bounds(), want.Bounds())
		return
	}
	metalTolerance.Check(t, name, gputest.DiffRect(got, want, got.Bounds()))
}

// TestEveryShaderOfTheViewCompilesOnMetal compiles the MSL of every shader of package viewport3d with Metal's own
// compiler and finds the function of each entry point (SPEC §29.5: a shader that Metal refuses is a test failure).
func TestEveryShaderOfTheViewCompilesOnMetal(t *testing.T) {
	d := openDevice(t).(*metal.Device)
	for _, sh := range []*gpu.Shader{vshaders.Env, vshaders.Grid, vshaders.Line, vshaders.Mesh, vshaders.Pbr, vshaders.Pick,
		vshaders.Point, vshaders.Section, vshaders.Select, vshaders.Shadow, vshaders.Tonemap} {
		var entries []string
		for _, e := range sh.Entries {
			entries = append(entries, e.Name)
		}
		if err := d.CompileShader(sh.Name, sh.MSL, entries...); err != nil {
			t.Errorf("%s: %v", sh.Name, err)
		}
	}
}
