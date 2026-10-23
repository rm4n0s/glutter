package android

import "time"

// host is what the UI thread's loop needs from the operating system, behind an interface so that
// the platform is plain Go that a test runs with a fake: the ALooper with its eventfd and the
// AChoreographer are in host_android.go (SPEC §24.3).
//
// Flutter: no counterpart
type host interface {
	// Poll blocks the UI thread until Wake was called or a frame callback arrived. When a frame
	// callback fired, vsync is true and frameTime is its vsync time.
	Poll() (frameTime time.Duration, vsync bool)
	// Wake makes Poll return. It is the one method for any thread.
	Wake()
	// RequestVsync asks for one frame callback. It is called on the UI thread, and not again before
	// the callback came.
	RequestVsync()
	// ReleaseWindow lets go of the native window that the surface held (ANativeWindow_release).
	ReleaseWindow(window uintptr)
}
