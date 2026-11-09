//go:build linux && glutter_display

package viewport3d_test

import (
	"io"
	"log/slog"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/gpuhost"
)

func benchDevice(b *testing.B) (*gpuhost.App, *gpu.Device) {
	b.Helper()
	hardware := os.Getenv("GLUTTER_BENCH_GPU") == "hardware"
	opts := vulkan.Options{AllowCPU: !hardware, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if !hardware {
		arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
		icd := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
		if _, err := os.Stat(icd); err != nil {
			b.Skip("Mesa's lavapipe is not installed (sudo dnf install mesa-vulkan-drivers)")
		}
		b.Setenv("VK_DRIVER_FILES", icd)
	}
	vk, err := vulkan.Open(opts)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := vk.Close(); err != nil {
			b.Error(err)
		}
	})
	b.Logf("device: %s", vk.DeviceName())
	app := gpuhost.NewBinding(b, vk)
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		b.Fatal(err)
	}
	return app, d
}

// threadCPU is the CPU time that the calling thread has used (user and system): a thread that waits for the GPU does not use any.
func threadCPU(b *testing.B) time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(1, &ru); err != nil { // RUSAGE_THREAD
		b.Fatal(err)
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
