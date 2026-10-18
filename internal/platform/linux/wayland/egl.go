package wayland

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// waylandEGLLibrary is the versioned name of libwayland-egl (SPEC §8.1).
const waylandEGLLibrary = "libwayland-egl.so.1"

// waylandEGL is the part of libwayland-egl that makes a wl_surface renderable
// with EGL: a wl_egl_window is the native window handed to eglCreateWindowSurface.
type waylandEGL struct {
	create  func(surface uintptr, width, height int32) uintptr
	resize  func(window uintptr, width, height, dx, dy int32)
	destroy func(window uintptr)
}

// loadWaylandEGL opens libwayland-egl once. A failure (LibraryLoadFailed, or
// SymbolMissing) is returned on every call.
var loadWaylandEGL = sync.OnceValues(func() (*waylandEGL, error) {
	lib, err := dl.Open(waylandEGLLibrary)
	if err != nil {
		return nil, err
	}
	api := new(waylandEGL)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&api.create, "wl_egl_window_create"},
		{&api.resize, "wl_egl_window_resize"},
		{&api.destroy, "wl_egl_window_destroy"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return api, nil
})
