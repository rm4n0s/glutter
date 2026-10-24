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
	// KeyHandled tells the activity what the framework made of the key event with the id that went
	// with it: an event that was not handled is dispatched again to the rest of the view hierarchy
	// and the system (SPEC §24.5). It is called on the UI thread.
	KeyHandled(id int, handled bool)
	// SetCursor sets the mouse cursor, as an android.view.PointerIcon type. It is called on the UI
	// thread.
	SetCursor(pointerIconType int)
	// Inject sends a synthetic tap, swipe or key of the dev events script to the view, on the main
	// thread, through the view's own dispatch. It is safe to call from any goroutine.
	Inject(e DevEvent)
	// Command asks the activity to do something on the main thread of Android: the system's
	// interface, haptic feedback, the sounds of the system (SPEC §24.7). It is called on the UI
	// thread.
	Command(c command)
	// ScaledFontSize is how the system scales a font size in logical pixels (TypedValue.applyDimension
	// with the unit sp, as logical pixels). It is called on the UI thread and returns at once.
	ScaledFontSize(fontSize float64) float64
	// ReleaseWindow lets go of the native window that the surface held (ANativeWindow_release).
	ReleaseWindow(window uintptr)
}
