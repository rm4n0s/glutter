//go:build darwin

package metal

import (
	"image"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The autoresizing mask of a layer that follows the size of its superlayer (kCALayerWidthSizable and
// kCALayerHeightSizable).
const layerSizable uint64 = 2 | 16

// swapchain is a CAMetalLayer that covers the layer of the window's view (SPEC §29.5): BGRA8Unorm, at most 3
// drawables, display sync on, and framebufferOnly unless a readback was asked for. A drawable is asked for
// only when a frame is drawn (Acquire), so an idle app holds none.
type swapchain struct {
	d      *Device
	layer  objc.ID // the CAMetalLayer, a sublayer of host
	host   objc.ID // the layer of the view
	w, h   int
	drawbl objc.ID // the drawable that Acquire gave, until Present
	tex    *texture
}

var _ explicit.Swapchain = (*swapchain)(nil)

// NewSwapchain implements explicit.Device: the surface is the layer of a view (gpu.SurfaceKindMetalLayer). It
// fails with MetalSwapchainFailed for another kind of surface.
func (d *Device) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	host, err := surfaceLayer(h)
	if err != nil {
		return nil, err
	}
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	sc := &swapchain{d: d, host: retain(host)}
	objcx.WithPool(func() {
		sc.layer = newObject("CAMetalLayer")
		call(sc.layer, "setDevice:", d.dev)
		call(sc.layer, "setPixelFormat:", pixelFormatBGRA8Unorm)
		call(sc.layer, "setFramebufferOnly:", !d.opts.Readback)
		call(sc.layer, "setMaximumDrawableCount:", uint64(3))
		call(sc.layer, "setDisplaySyncEnabled:", true)
		call(sc.layer, "setAllowsNextDrawableTimeout:", true)
		call(sc.layer, "setOpaque:", true)
		call(sc.layer, "setAutoresizingMask:", layerSizable)
		call(sc.layer, "setContentsScale:", send[float64](host, "contentsScale"))
		call(sc.layer, "setFrame:", send[cgRect](host, "bounds"))
		call(host, "addSublayer:", sc.layer)
	})
	return sc, nil
}

// Configure implements explicit.Swapchain: the size of the drawables, in pixels.
func (s *swapchain) Configure(width, height int) error {
	if width <= 0 || height <= 0 {
		return errors.New(TagMetalSwapchainFailed, "the size of a swap chain is positive", "width", width, "height", height)
	}
	s.w, s.h = width, height
	s.followHost()
	call(s.layer, "setDrawableSize:", cgSize{Width: float64(width), Height: float64(height)})
	return nil
}

// followHost gives the layer the scale of the view's layer, which the platform changes when the window moves to a
// display of another scale.
func (s *swapchain) followHost() {
	if scale := send[float64](s.host, "contentsScale"); scale != send[float64](s.layer, "contentsScale") {
		call(s.layer, "setContentsScale:", scale)
	}
}

// PreRotation implements explicit.Swapchain: a Mac's display is never rotated for the app.
func (s *swapchain) PreRotation() explicit.Rotation { return explicit.Rotation0 }

// Stale implements explicit.Swapchain: the layer always matches its view, which follows it with the
// autoresizing mask; the size is checked when a drawable is acquired.
func (s *swapchain) Stale() bool { return false }

// Acquire implements explicit.Swapchain: the texture of the next drawable. It fails with MetalDrawableTimeout
// when the layer gives none within its timeout (one second), and with SwapchainOutOfDate when the drawable is
// not the size that Configure set, so that the caller configures again and retries (SPEC §9.3).
func (s *swapchain) Acquire() (explicit.Texture, error) {
	if s.drawbl != 0 {
		return s.tex, nil
	}
	var drawable objc.ID
	objcx.WithPool(func() { drawable = retain(call(s.layer, "nextDrawable")) })
	if drawable == 0 {
		return nil, errors.New(TagMetalDrawableTimeout, "the layer gave no drawable within its timeout")
	}
	tex := call(drawable, "texture")
	w, h := int(send[uint64](tex, "width")), int(send[uint64](tex, "height"))
	if s.w != 0 && (w != s.w || h != s.h) {
		release(drawable)
		return nil, errors.New(explicit.SwapchainOutOfDate, "the drawable is not the size of the swap chain", "width", w, "height", h, "want_width", s.w, "want_height", s.h)
	}
	s.drawbl = drawable
	s.tex = &texture{d: s.d, obj: tex, drawable: true, desc: explicit.TextureDesc{
		Width: w, Height: h, Kind: explicit.TextureKind2D, DepthOrLayers: 1, Format: explicit.TextureFormatBGRA8Unorm, MipLevels: 1, SampleCount: 1,
		Usage: explicit.TextureRenderTarget | explicit.TextureCopySource,
	}}
	return s.tex, nil
}

// Present implements explicit.Swapchain: a command buffer that presents the drawable after the frame that drew
// into it, which is on the same queue. During a live resize the present is made in the transaction of the
// window (presentsWithTransaction), so that the content and the window's frame change together. Metal has no
// partial present, so the damage is not used.
func (s *swapchain) Present(_ []image.Rectangle) error {
	if s.drawbl == 0 {
		return errors.New(TagMetalSwapchainFailed, "Present was called without an Acquire")
	}
	if err := s.d.checkLost(); err != nil {
		return err
	}
	drawable := s.drawbl
	s.drawbl, s.tex = 0, nil
	objcx.WithPool(func() {
		live := s.inLiveResize()
		call(s.layer, "setPresentsWithTransaction:", live)
		cb := call(s.d.queue, "commandBuffer")
		if live {
			call(cb, "commit")
			call(cb, "waitUntilScheduled")
			call(drawable, "present")
		} else {
			call(cb, "presentDrawable:", drawable)
			call(cb, "commit")
		}
	})
	release(drawable)
	return nil
}

// inLiveResize says whether the window of the view is being resized with the mouse: the view is the delegate
// of its layer.
func (s *swapchain) inLiveResize() bool {
	view := call(s.host, "delegate")
	return view != 0 && sendBool(view, "respondsToSelector:", sel("inLiveResize")) && sendBool(view, "inLiveResize")
}

// Destroy implements explicit.Swapchain.
func (s *swapchain) Destroy() {
	if s.layer == 0 {
		return
	}
	release(s.drawbl)
	s.drawbl, s.tex = 0, nil
	call(s.layer, "removeFromSuperlayer")
	release(s.layer)
	release(s.host)
	s.layer, s.host = 0, 0
}
