package android

import (
	"github.com/rm4n0s/glutter/internal/gpu"
)

// surfaceState is what the UI thread knows about the surface of the view and the display it is
// on. The events of the view's SurfaceHolder change it, one after the other, on the UI thread
// (SPEC §24.3, §24.4):
//
//	created   a new ANativeWindow exists (the old one, if any, was destroyed first)
//	changed   the surface has a size, and the display a density and a refresh rate
//	destroyed the window must not be used again
//
// Flutter: shell/platform/android/android_surface.h AndroidSurface, android_shell_holder.cc
type surfaceState struct {
	// window is the ANativeWindow* of the surface, 0 while there is none.
	window uintptr
	// width and height are the size of the surface in pixels; 0 until the first change.
	width, height int
	// density is DisplayMetrics.density, the device pixel ratio; refresh is the refresh rate of
	// the display in hertz; displayWidth and displayHeight are its size in pixels.
	density, refresh            float64
	displayWidth, displayHeight int
}

// created records a new window and returns the one it replaces, 0 when there was none: the
// caller lets go of that one.
func (s *surfaceState) created(window uintptr) (old uintptr) {
	old, s.window = s.window, window
	return old
}

// changed records the size of the surface and the display. It reports whether the metrics of the
// view or the displays differ from before.
func (s *surfaceState) changed(width, height int, density, refresh float64, displayWidth, displayHeight int) bool {
	next := surfaceState{window: s.window, width: width, height: height, density: density, refresh: refresh,
		displayWidth: displayWidth, displayHeight: displayHeight}
	differs := next != *s
	*s = next
	return differs
}

// destroyed forgets the window and returns it, 0 when there was none. The size stays: the next
// surface of the view is usually the same size.
func (s *surfaceState) destroyed() (old uintptr) {
	old, s.window = s.window, 0
	return old
}

// ready is whether a frame can be drawn: there is a window and it has a size.
func (s *surfaceState) ready() bool { return s.window != 0 && s.width > 0 && s.height > 0 }

// handle is the handle of the GPU backends for the surface (SurfaceKindAndroid); the zero handle
// while there is no window.
func (s *surfaceState) handle() gpu.SurfaceHandle {
	if s.window == 0 {
		return gpu.SurfaceHandle{}
	}
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: s.window}
}
