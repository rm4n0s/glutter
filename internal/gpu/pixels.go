package gpu

import "image"

// PixelFormat is the byte order of the pixels of a PixelBuffer.
//
// Flutter: no counterpart
type PixelFormat int

// The formats of a PixelBuffer.
const (
	// PixelFormatBGRX8888 is four bytes for a pixel in the order blue, green, red and a byte that
	// the window ignores (XRGB8888 of Wayland's wl_shm and of the Linux frame buffers, a 24-bit
	// ZPixmap of a little-endian X server): an opaque window.
	//
	// Flutter: no counterpart
	PixelFormatBGRX8888 PixelFormat = iota
)

// PixelBuffer is memory that a window shows once it is presented: the target of a renderer that
// draws on the CPU (package shmwindow, SPEC §9.6).
//
// Flutter: no counterpart
type PixelBuffer struct {
	// Pix is the pixels of the buffer, rows of Stride bytes with the top row first, in Format.
	// The renderer writes into it between PixelTarget.AcquirePixels and PixelTarget.PresentPixels
	// and keeps no slice of it after.
	Pix []byte
	// Stride is the length of a row in bytes, at least 4 times Width.
	Stride int
	// Width and Height are the size of the buffer in pixels.
	Width, Height int
	// Format is the byte order of Pix.
	Format PixelFormat
	// Age says what Pix holds: 0 when it is not defined (new memory), and n when it holds the frame
	// that was presented n presentations ago, so that 1 is the frame the window shows now (the
	// age of EGL_EXT_buffer_age). A renderer redraws, in a buffer of age n, what the last n
	// frames changed.
	Age int
}

// PixelTarget is a window that shows pixels the renderer wrote into its memory: a Wayland window
// through wl_shm, an X11 window through MIT-SHM or PutImage. The platform backend implements it
// on its window type and the engine gives it to the renderer by the window's SurfaceHandle. It is
// used on the UI thread only (SPEC §13).
//
// Flutter: no counterpart
type PixelTarget interface {
	// AcquirePixels returns a buffer of width x height pixels that nothing reads now. The
	// previous buffer, if it was not presented, is given up. It fails when the memory cannot be
	// made.
	AcquirePixels(width, height int) (PixelBuffer, error)
	// PresentPixels shows the buffer that AcquirePixels returned last, and tells the window
	// which rectangles of it differ from the frame it shows now (the damage; an empty list says
	// that the whole of it may). With no buffer acquired, it presents the frame again with no
	// change: the window's frame pacing goes on, and nothing is copied.
	PresentPixels(damage []image.Rectangle) error
}
