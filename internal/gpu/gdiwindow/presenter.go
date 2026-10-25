package gdiwindow

import (
	"image"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// windowAPI is the part of GDI that the presenter calls, so that the sequence of calls is plain Go
// that a test runs with a fake (the Windows build binds it to user32 and gdi32, present_windows.go).
type windowAPI interface {
	// getDC is GetDC of the window's client area; releaseDC gives it back.
	getDC(window uintptr) (dc uintptr, err error)
	releaseDC(window, dc uintptr)
	// blit is one StretchDIBits at 1:1: the pixels are rows of width pixels in BGRA order, top
	// row first, and r is the rectangle of them to copy to the same place in the window. It
	// returns the number of lines copied, 0 when the call failed.
	blit(dc uintptr, bgra []byte, width int, r image.Rectangle) int
	// paint is the painting of WM_PAINT: BeginPaint, draw with the device context and the invalid
	// rectangle, EndPaint (which validates the window). draw is not called when BeginPaint fails.
	paint(window uintptr, draw func(dc uintptr, clip image.Rectangle)) error
	// setPaintHandler sets the function that WM_PAINT calls for the window; nil removes it.
	setPaintHandler(window uintptr, fn func(window uintptr))
}

// keptFrame is the last frame shown in a window, in the order GDI takes: BGRA, top row first, with
// no padding (a row of 32-bit pixels is a multiple of 4 bytes already).
type keptFrame struct {
	width, height int
	bgra          []byte
}

// bounds is the rectangle of the whole frame.
func (k *keptFrame) bounds() image.Rectangle { return image.Rect(0, 0, k.width, k.height) }

// windowPresenter is the Presenter over a windowAPI.
type windowPresenter struct {
	api windowAPI
	mu  sync.Mutex
	// frames are the last frame shown in each window.
	frames map[uintptr]*keptFrame
}

var _ Presenter = (*windowPresenter)(nil)

func newWindowPresenter(api windowAPI) *windowPresenter {
	return &windowPresenter{api: api, frames: map[uintptr]*keptFrame{}}
}

// Present copies the damaged rectangles of the image into the window's kept frame and blits each
// with one call, into the window's device context. A new size, a first frame and a Full damage
// copy all of it. The first frame of a window sets the function that repaints the kept frame on
// WM_PAINT. It fails with GDIPresentFailed, with the window and the rectangle in the metadata,
// when GDI copies nothing.
//
// Flutter: shell/platform/windows/compositor_software.cc CompositorSoftware::CollectBackingStore
func (p *windowPresenter) Present(h gpu.SurfaceHandle, img *image.RGBA, damage gpu.Damage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	w, ht := img.Rect.Dx(), img.Rect.Dy()
	if w <= 0 || ht <= 0 {
		return nil
	}
	k := p.frames[h.Window]
	whole := damage.Full
	if k == nil || k.width != w || k.height != ht {
		if k == nil {
			p.api.setPaintHandler(h.Window, p.repaint)
		}
		k = &keptFrame{width: w, height: ht, bgra: make([]byte, w*ht*4)}
		p.frames[h.Window] = k
		whole = true
	}
	rects := damage.Rects
	if whole {
		rects = []image.Rectangle{k.bounds()}
	}
	var drawn []image.Rectangle
	for _, r := range rects {
		if r = r.Intersect(k.bounds()); !r.Empty() {
			copyBGRA(k, img, r)
			drawn = append(drawn, r)
		}
	}
	if len(drawn) == 0 {
		return nil
	}
	dc, err := p.api.getDC(h.Window)
	if err != nil {
		return err
	}
	defer p.api.releaseDC(h.Window, dc)
	for _, r := range drawn {
		if p.api.blit(dc, k.bgra, k.width, r) == 0 {
			return errors.New(TagGDIPresentFailed, "StretchDIBits copied nothing", "window", h.Window,
				"x", r.Min.X, "y", r.Min.Y, "width", r.Dx(), "height", r.Dy())
		}
	}
	return nil
}

// repaint is WM_PAINT: it paints the kept frame in the invalid rectangle. It begins and ends the
// paint whatever happens, or the window would get WM_PAINT again and again.
func (p *windowPresenter) repaint(window uintptr) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := p.frames[window]
	_ = p.api.paint(window, func(dc uintptr, clip image.Rectangle) {
		if k == nil {
			return
		}
		if r := clip.Intersect(k.bounds()); !r.Empty() {
			p.api.blit(dc, k.bgra, k.width, r) // a failure shows as an unpainted exposure, and the next frame repairs it
		}
	})
}

// Release forgets the kept frame of the window and stops its repainting.
func (p *windowPresenter) Release(h gpu.SurfaceHandle) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.frames[h.Window]; ok {
		p.api.setPaintHandler(h.Window, nil)
		delete(p.frames, h.Window)
	}
}

// copyBGRA copies the rectangle of the RGBA image into the kept frame, swapping red and blue.
func copyBGRA(k *keptFrame, img *image.RGBA, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		src := img.Pix[img.PixOffset(img.Rect.Min.X+r.Min.X, img.Rect.Min.Y+y):][:4*r.Dx()]
		dst := k.bgra[(y*k.width+r.Min.X)*4:][:4*r.Dx()]
		for i := 0; i < len(src); i += 4 {
			dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagGDIPresentFailed is the tag of the error for a frame that GDI does not copy into the
	// window; the metadata are "window", "x", "y", "width" and "height".
	TagGDIPresentFailed = "GDIPresentFailed"
)
