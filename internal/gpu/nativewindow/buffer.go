package nativewindow

import "image"

// windowFormatRGBA8888 is WINDOW_FORMAT_RGBA_8888 of <android/native_window.h>: the bytes of a
// pixel are red, green, blue, alpha, which is the order of image.RGBA.
const windowFormatRGBA8888 = 1

// windowFormatRGBX8888 is WINDOW_FORMAT_RGBX_8888: the same bytes, the alpha ignored.
const windowFormatRGBX8888 = 2

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

// copyFrame copies the rows of img into the window's buffer, which holds Height rows of Stride
// pixels of four bytes. Only the rows and columns both have are copied. dst is the buffer's
// memory.
func copyFrame(dst []byte, stride, width, height int, img *image.RGBA) {
	w := min(width, img.Rect.Dx())
	h := min(height, img.Rect.Dy())
	for y := 0; y < h; y++ {
		src := img.Pix[img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y+y):][:4*w]
		copy(dst[y*stride*4:], src)
	}
}
