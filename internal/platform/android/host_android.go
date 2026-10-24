package android

/*
#cgo LDFLAGS: -landroid -llog
#include "jni_android.h"
#include <stdlib.h>
*/
import "C"

import (
	"time"
)

// nativeHost is the host of the UI thread: an ALooper with an eventfd that Wake writes, and an
// AChoreographer whose frame callbacks the looper delivers (SPEC §24.3). The looper is prepared
// by initLooper, on the UI thread, before the host is used.
type nativeHost struct{}

var _ host = nativeHost{}

// initLooper prepares the looper of the calling thread, which must be the UI thread, and returns
// 0 on success.
func initLooper() int { return int(C.glutter_host_init()) }

// Poll implements host.
func (nativeHost) Poll() (time.Duration, bool) {
	var t C.longlong
	if C.glutter_host_poll(&t) == 1 {
		return time.Duration(t), true
	}
	return 0, false
}

// Wake implements host.
func (nativeHost) Wake() { C.glutter_host_wake() }

// RequestVsync implements host.
func (nativeHost) RequestVsync() { C.glutter_host_request_vsync() }

// Inject implements host: the Java side builds the events and dispatches them on the main thread.
func (nativeHost) Inject(e DevEvent) {
	var kind C.int
	switch e.Kind {
	case "tap":
		kind = 1
	case "swipe":
		kind = 2
	case "key":
		kind = 3
	case "text":
		kind = 4
	case "compose":
		kind = 5
	case "action":
		kind = 6
	case "back":
		kind = 7
	default:
		return
	}
	key := C.CBytes([]byte(e.Key))
	defer C.free(key)
	C.glutter_inject(kind, C.float(e.X1), C.float(e.Y1), C.float(e.X2), C.float(e.Y2), C.int(e.Millis), (*C.char)(key), C.int(len(e.Key)))
}

// KeyHandled implements host.
func (nativeHost) KeyHandled(id int, handled bool) {
	h := C.int(0)
	if handled {
		h = 1
	}
	C.glutter_key_handled(C.int(id), h)
}

// SetCursor implements host.
func (nativeHost) SetCursor(pointerIconType int) { C.glutter_set_cursor(C.int(pointerIconType)) }

// ReleaseWindow implements host.
func (nativeHost) ReleaseWindow(window uintptr) { C.glutter_release_window(C.uintptr_t(window)) }

// Command implements host.
func (nativeHost) Command(c command) {
	// The text goes as bytes of UTF-8 (any valid text, with NUL and four-byte characters).
	text := C.CBytes([]byte(c.S))
	defer C.free(text)
	C.glutter_command(C.int(c.Op), C.longlong(c.A), C.longlong(c.B), C.longlong(c.C), C.longlong(c.D), (*C.char)(text), C.int(len(c.S)))
}

// ScaledFontSize implements host: the system's own scaling of a size in sp.
func (nativeHost) ScaledFontSize(size float64) float64 {
	return float64(C.glutter_scaled_font_size(C.float(size)))
}
