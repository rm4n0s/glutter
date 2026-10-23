//go:build android

package gles

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// androidWindow is libandroid.so's ANativeWindow_setBuffersGeometry, bound once.
var androidWindow = sync.OnceValue(func() func(window uintptr, width, height, format int32) int32 {
	lib, err := dl.Open("libandroid.so")
	if err != nil {
		return nil
	}
	var set func(window uintptr, width, height, format int32) int32
	if err := lib.Bind(&set, "ANativeWindow_setBuffersGeometry"); err != nil {
		return nil
	}
	return set
})

// prepareWindow gives the ANativeWindow the pixel format of the EGL config (its
// EGL_NATIVE_VISUAL_ID), keeping the size, as Android's EGL documentation asks before
// eglCreateWindowSurface. Without libandroid the window keeps its own format, which is RGBA_8888
// for a SurfaceView, so a failure is not an error.
func (l *eglLib) prepareWindow(display, config, window uintptr) {
	set := androidWindow()
	if set == nil {
		return
	}
	if format, ok := l.getConfigAttrib(display, config, eglNativeVisualID); ok {
		set(window, 0, 0, format)
	}
}
