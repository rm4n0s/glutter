//go:build darwin && !ios && glutter_display

package viewport3d_test

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// benchDevice opens the Mac's Metal device for a benchmark (SPEC §27.10, §29.5). The numbers of the benchmarks are those of
// the machine they run on (Apple silicon with its own GPU), which is the target.
func benchDevice(b *testing.B) (*gpuhost.App, *gpu.Device) {
	b.Helper()
	d, err := metal.Open(metal.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		b.Skipf("no Metal device: %v", err)
	}
	b.Cleanup(func() {
		if err := d.Close(); err != nil {
			b.Error(err)
		}
	})
	b.Logf("device: %s", d.Name())
	app := gpuhost.NewBinding(b, d)
	gd, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		b.Fatal(err)
	}
	return app, gd
}

// clockThreadCPUTime is CLOCK_THREAD_CPUTIME_ID of clock_gettime_nsec_np.
const clockThreadCPUTime = 16

// threadNanos is clock_gettime_nsec_np of libSystem, bound once.
var threadNanos = func() func(clock uint32) uint64 {
	lib, err := dl.Open("/usr/lib/libSystem.B.dylib")
	if err != nil {
		panic(err)
	}
	var f func(clock uint32) uint64
	if err := lib.Bind(&f, "clock_gettime_nsec_np"); err != nil {
		panic(err)
	}
	return f
}()

// threadCPU is the CPU time that the calling thread has used: a thread that waits for the GPU does not use any.
func threadCPU(b *testing.B) time.Duration {
	n := threadNanos(clockThreadCPUTime)
	if n == 0 {
		b.Fatal("clock_gettime_nsec_np gave no thread time")
	}
	return time.Duration(n)
}
