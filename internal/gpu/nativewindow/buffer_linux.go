//go:build linux

package nativewindow

// bufferSize is the size of ANativeWindow_Buffer on a 64-bit ABI.
const bufferSize = 48

// windowBuffer mirrors ANativeWindow_Buffer of <android/native_window.h> on a 64-bit ABI:
//
//	int32_t width; int32_t height; int32_t stride; int32_t format; void *bits; uint32_t reserved[6];
//
// ANativeWindow_lock fills it in; it lives in C memory and Bits points into the window's buffer.
type windowBuffer struct {
	Width  int32
	Height int32
	// Stride is the length of a row in pixels, which can be more than Width.
	Stride int32
	Format int32
	Bits   uintptr
	_      [6]uint32
}
