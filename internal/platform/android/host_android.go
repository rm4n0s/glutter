package android

/*
#cgo LDFLAGS: -landroid -llog
#include "jni_android.h"
*/
import "C"

import "time"

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

// ReleaseWindow implements host.
func (nativeHost) ReleaseWindow(window uintptr) { C.glutter_release_window(C.uintptr_t(window)) }
