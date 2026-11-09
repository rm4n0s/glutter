//go:build darwin && !ios

package metal

import (
	"bytes"
	"log/slog"
	"runtime"
	"testing"
)

// openTestDevice opens a device, skipping the test when the machine has none, and closes it at the end. The test
// keeps its OS thread, as the device is used from one.
func openTestDevice(t *testing.T, opts Options) *Device {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(testWriter{t}, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	d, err := Open(opts)
	if err != nil {
		if appErr, ok := asTagged(err); ok && appErr == TagMetalDeviceUnavailable {
			t.Skipf("no Metal device: %v", err)
		}
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return d
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(bytes.TrimRight(p, "\n")))
	return len(p), nil
}
