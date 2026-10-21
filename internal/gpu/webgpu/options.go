package webgpu

import (
	"log/slog"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
)

// Options are the optional parameters of Acquire.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the log of the backend: a lost device, the messages the browser's validation
	// reports; slog.Default() when nil.
	Logger *slog.Logger
	// Post runs fn on the UI goroutine. The browser's callbacks (a finished read-back, a lost
	// device, a device that came after a loss) call it so that their work happens where the
	// framework runs (SPEC §23.3). When nil, fn runs inside the callback.
	Post func(fn func())
	// OnDeviceReady is called on the UI goroutine, through Post, when the device that was
	// asked for after a loss has arrived; the handle is the window the last device was opened
	// for. The engine asks for a frame of that window, which opens the device.
	OnDeviceReady func(h gpu.SurfaceHandle)
	// Timeout is how long Acquire waits for the adapter and the device; 20 seconds when zero.
	Timeout time.Duration
}
