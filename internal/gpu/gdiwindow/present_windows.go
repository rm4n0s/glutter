//go:build windows

package gdiwindow

import (
	"image"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// NewPresenter makes the Presenter of Windows: it blits the frame with StretchDIBits into the
// device context of the window, and repaints the kept frame on WM_PAINT. Use it on the UI thread.
//
// Flutter: no counterpart
func NewPresenter() Presenter { return newWindowPresenter(gdiAPI{}) }

// gdiAPI is the windowAPI over user32 and gdi32.
type gdiAPI struct{}

func (gdiAPI) getDC(window uintptr) (uintptr, error) { return win32.GetDC(window) }
func (gdiAPI) releaseDC(window, dc uintptr)          { win32.ReleaseDC(window, dc) }

// blit copies a rectangle of a top-down 32-bit DIB to the same place in the window. The source
// is the rows of the rectangle (the pointer starts at its first row), as wide as the whole
// frame, so that the stride is the frame's and the source rectangle is plain: x from the left,
// y from 0, whatever the origin of a top-down DIB is taken to be.
func (gdiAPI) blit(dc uintptr, bgra []byte, width int, r image.Rectangle) int {
	info := win32.BitmapInfo{Header: win32.BitmapInfoHeader{
		Size: uint32(unsafe.Sizeof(win32.BitmapInfoHeader{})), Width: int32(width), Height: -int32(r.Dy()),
		Planes: 1, BitCount: 32, Compression: win32.BIRGB,
	}}
	rows := bgra[r.Min.Y*width*4 : r.Max.Y*width*4]
	return int(win32.StretchDIBits(dc, int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()),
		int32(r.Min.X), 0, int32(r.Dx()), int32(r.Dy()), rows, &info, win32.DIBRGBColors, win32.SrcCopy))
}

func (gdiAPI) paint(window uintptr, draw func(dc uintptr, clip image.Rectangle)) error {
	ps, err := win32.BeginPaint(window)
	if err != nil {
		return err
	}
	defer win32.EndPaint(window, &ps)
	draw(ps.Hdc, image.Rect(int(ps.Paint.Left), int(ps.Paint.Top), int(ps.Paint.Right), int(ps.Paint.Bottom)))
	return nil
}

func (gdiAPI) setPaintHandler(window uintptr, fn func(window uintptr)) {
	if fn == nil {
		win32.SetPaintHandler(window, nil)
		return
	}
	win32.SetPaintHandler(window, win32.PaintFunc(fn))
}
