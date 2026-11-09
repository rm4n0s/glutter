//go:build darwin && !ios && glutter_display

package gpu_test

import (
	"io"
	"log/slog"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
	"github.com/rm4n0s/glutter/internal/gpuhost"
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

// deviceApp is an app whose GPU device is the Mac's.
func deviceApp(t *testing.T) (*gpuhost.App, *gpu.Device) {
	t.Helper()
	app := gpuhost.NewBinding(t, openDevice(t))
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return app, d
}
