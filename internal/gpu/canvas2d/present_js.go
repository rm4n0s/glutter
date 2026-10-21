//go:build js && wasm

package canvas2d

import (
	"image"
	"strconv"
	"syscall/js"

	"github.com/rm4n0s/glutter/internal/gpu"
)

// viewAttribute names the view a canvas shows; the web platform sets it
// (internal/platform/web ViewAttribute). It is repeated here because a GPU backend
// does not import the platform.
const viewAttribute = "data-glutter-view"

// New makes the renderer of the page's canvases: soft draws each frame, and putImageData
// shows it. The canvas of a handle is the element with the attribute data-glutter-view
// equal to the handle's Window.
//
// Flutter: no counterpart
func New(opts ...Options) *Renderer {
	return NewRenderer(&canvasPresenter{targets: map[uintptr]*target{}}, opts...)
}

// target is a canvas that frames are put into, with the buffers it reuses for frames of
// its size.
type target struct {
	canvas, ctx js.Value
	width       int
	height      int
	// data is the Uint8ClampedArray that image holds as its data, so one CopyBytesToJS
	// per frame fills the ImageData.
	data, image js.Value
	compact     []byte // the frame's rows with no padding, when the image has any
}

// canvasPresenter is the Presenter of the page.
type canvasPresenter struct {
	targets map[uintptr]*target
}

// find returns the target of the handle's view, looking its canvas up the first time.
func (p *canvasPresenter) find(h gpu.SurfaceHandle) (*target, error) {
	if t, ok := p.targets[h.Window]; ok && t.canvas.Get("isConnected").Bool() {
		return t, nil
	}
	canvas := js.Global().Get("document").Call("querySelector", "canvas["+viewAttribute+`="`+strconv.FormatUint(uint64(h.Window), 10)+`"]`)
	if !canvas.Truthy() {
		return nil, errCanvasNotFound(h.Window)
	}
	ctx := canvas.Call("getContext", "2d", map[string]any{"alpha": false})
	if !ctx.Truthy() {
		return nil, errContextUnavailable(h.Window)
	}
	t := &target{canvas: canvas, ctx: ctx}
	p.targets[h.Window] = t
	return t, nil
}

// Present puts the image into the canvas: it sets the canvas's size when it differs
// (which clears the canvas, and the whole frame covers it), copies the bytes into
// JavaScript once, and calls putImageData.
func (p *canvasPresenter) Present(h gpu.SurfaceHandle, img *image.RGBA) error {
	t, err := p.find(h)
	if err != nil {
		return err
	}
	w, ht := img.Rect.Dx(), img.Rect.Dy()
	if t.width != w || t.height != ht || !t.image.Truthy() {
		if t.canvas.Get("width").Int() != w {
			t.canvas.Set("width", w)
		}
		if t.canvas.Get("height").Int() != ht {
			t.canvas.Set("height", ht)
		}
		t.width, t.height = w, ht
		t.data = js.Global().Get("Uint8ClampedArray").New(w * ht * 4)
		t.image = js.Global().Get("ImageData").New(t.data, w, ht)
		t.compact = nil
	}
	pix := img.Pix
	if img.Stride != 4*w || img.Rect.Min != (image.Point{}) {
		if t.compact == nil {
			t.compact = make([]byte, 4*w*ht)
		}
		for y := 0; y < ht; y++ {
			copy(t.compact[4*w*y:4*w*(y+1)], img.Pix[img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y+y):])
		}
		pix = t.compact
	}
	js.CopyBytesToJS(t.data, pix)
	t.ctx.Call("putImageData", t.image, 0, 0)
	return nil
}

// Release forgets the canvas of the handle.
func (p *canvasPresenter) Release(h gpu.SurfaceHandle) { delete(p.targets, h.Window) }
