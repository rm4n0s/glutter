//go:build js && wasm

package webgpu

import (
	"image"
	"strconv"
	"syscall/js"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// viewAttribute names the view a canvas shows; the web platform sets it
// (internal/platform/web ViewAttribute). It is repeated here because a GPU backend does not import
// the platform.
const viewAttribute = "data-glutter-view"

// swapchain is the webgpu context of a canvas (SPEC §9.5): getCurrentTexture is the image to draw
// into, and the browser presents it when the task ends.
type swapchain struct {
	d          *Device
	view       uintptr
	canvas     js.Value
	ctx        js.Value
	format     explicit.TextureFormat
	formatName string
	configured bool
	w, h       int
	destroyed  bool
}

var _ explicit.Swapchain = (*swapchain)(nil)

// newSwapchain finds the canvas of the handle and its webgpu context. It fails with
//
//   - WebGPUSurfaceKindUnsupported when the handle does not name a canvas of a web page
//   - WebGPUCanvasNotFound when the page has no canvas for the view
//   - WebGPUContextUnavailable when the canvas gives no webgpu context: it has a context of
//     another kind already
func (d *Device) newSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	if h.Kind != gpu.SurfaceKindWeb {
		return nil, errors.New(tagWebGPUSurfaceKindUnsupported, "WebGPU draws into the canvas of a web page only", "kind", string(h.Kind))
	}
	doc := js.Global().Get("document")
	canvas := doc.Call("querySelector", "canvas["+viewAttribute+`="`+strconv.FormatUint(uint64(h.Window), 10)+`"]`)
	if !canvas.Truthy() {
		return nil, errors.New(tagWebGPUCanvasNotFound, "there is no canvas for the view in the page", "view", strconv.FormatUint(uint64(h.Window), 10))
	}
	ctx, err := tryCall("getContext", func() js.Value { return canvas.Call("getContext", "webgpu") })
	if err != nil {
		return nil, err
	}
	if !ctx.Truthy() {
		return nil, errors.New(tagWebGPUContextUnavailable, "the canvas gave no webgpu context", "view", strconv.FormatUint(uint64(h.Window), 10))
	}
	name := d.p.helper.Call("preferredFormat").String()
	format := formatOf(name)
	if format == explicit.TextureFormatUnspecified {
		return nil, errors.NewErr(tagWebGPUConfigureFailed, js.Error{Value: js.ValueOf("the canvas format " + name + " is not one the renderer draws into")})
	}
	return &swapchain{d: d, view: h.Window, canvas: canvas, ctx: ctx, format: format, formatName: name}, nil
}

// Configure implements explicit.Swapchain: it sets the canvas's size, which clears it (the frame
// that follows covers all of it), and configures the context once.
func (s *swapchain) Configure(width, height int) error {
	if err := s.d.usable(); err != nil {
		return err
	}
	if s.canvas.Get("width").Int() != width || s.canvas.Get("height").Int() != height {
		s.canvas.Set("width", width)
		s.canvas.Set("height", height)
	}
	s.w, s.h = width, height
	if s.configured {
		return nil
	}
	if err := s.configureContext(); err != nil {
		return err
	}
	s.configured = true
	return nil
}

// configureContext configures the canvas context, and returns what the browser threw as
// WebGPUConfigureFailed.
func (s *swapchain) configureContext() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if je, ok := r.(js.Error); ok {
				err = errors.NewErr(tagWebGPUConfigureFailed, je).SetMetadata("call", "configure")
				return
			}
			panic(r)
		}
	}()
	s.ctx.Call("configure", map[string]any{"device": s.d.dev, "format": s.formatName, "alphaMode": "opaque"})
	return nil
}

// Acquire implements explicit.Swapchain: the canvas's current texture. It never reports an out
// of date swapchain: the texture follows the canvas's size.
func (s *swapchain) Acquire() (explicit.Texture, error) {
	if err := s.d.usable(); err != nil {
		return nil, err
	}
	v, err := tryCall("getCurrentTexture", func() js.Value { return s.ctx.Call("getCurrentTexture") })
	if err != nil {
		return nil, err
	}
	desc := explicit.TextureDesc{Width: v.Get("width").Int(), Height: v.Get("height").Int(), Format: s.format, MipLevels: 1,
		Usage: explicit.TextureRenderTarget}
	return &texture{d: s.d, desc: desc, tex: v, view: v.Call("createView"), external: true}, nil
}

// Stale implements explicit.Swapchain: a canvas is never turned under the page.
func (s *swapchain) Stale() bool { return false }

// PreRotation implements explicit.Swapchain: a canvas is never rotated.
func (s *swapchain) PreRotation() explicit.Rotation { return explicit.Rotation0 }

// Present implements explicit.Swapchain: the browser presents the canvas when the task ends, and
// has no damage to be told.
func (s *swapchain) Present([]image.Rectangle) error { return s.d.usable() }

// Destroy implements explicit.Swapchain: the context is unconfigured.
func (s *swapchain) Destroy() {
	if s.destroyed {
		return
	}
	s.destroyed = true
	if s.configured && !s.d.loss.isLost() {
		_, _ = tryCall("unconfigure", func() js.Value { return s.ctx.Call("unconfigure") })
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebGPUCanvasNotFound         = "WebGPUCanvasNotFound"
	tagWebGPUConfigureFailed        = "WebGPUConfigureFailed"
	tagWebGPUContextUnavailable     = "WebGPUContextUnavailable"
	tagWebGPUSurfaceKindUnsupported = "WebGPUSurfaceKindUnsupported"
)
