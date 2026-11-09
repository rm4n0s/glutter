//go:build linux && glutter_display

package gpu_test

import (
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/gpuhost"
)

// openLavapipe opens lavapipe with validation, and checks at the end of the test that the layer reported nothing.
func openLavapipe(t *testing.T) *vulkan.Device {
	t.Helper()
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
	icd := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
	if _, err := os.Stat(icd); err != nil {
		t.Skip("Mesa's lavapipe is not installed (sudo dnf install mesa-vulkan-drivers)")
	}
	if _, err := os.Stat("/usr/share/vulkan/explicit_layer.d/VkLayer_khronos_validation.json"); err != nil {
		t.Skip("the validation layer is not installed (sudo dnf install vulkan-validation-layers)")
	}
	t.Setenv("VK_DRIVER_FILES", icd)
	d, err := vulkan.Open(vulkan.Options{AllowCPU: true, Validation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if errs := d.ValidationErrors(); len(errs) > 0 {
			t.Errorf("the validation layer reported %d errors:\n%s", len(errs), strings.Join(errs, "\n"))
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

// lavapipeApp is an app whose GPU device is lavapipe.
func deviceApp(t *testing.T) (*gpuhost.App, *gpu.Device) {
	t.Helper()
	app := gpuhost.NewBinding(t, openLavapipe(t))
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return app, d
}

// openDevice is the device of the display tests: lavapipe.
func openDevice(t *testing.T) explicit.Device { return openLavapipe(t) }
