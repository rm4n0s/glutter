package gles

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// RendererOpts are the optional parameters of NewRenderer.
//
// Flutter: no counterpart
type RendererOpts struct {
	// Background is the color every frame is cleared to before the scene is drawn.
	// The default is opaque white.
	Background ui.Color
	// HasBackground says that Background was set, so that a transparent black
	// background can be asked for.
	HasBackground bool
	// RasterCacheAfter turns on the raster cache: a layer that a scene retains (it is
	// in this many consecutive scenes) is drawn into a texture that later frames draw
	// instead of its contents. Zero, the default, keeps nothing. The engine sets 3: the
	// layer of a RepaintBoundary that the framework did not repaint for three frames.
	RasterCacheAfter int
	// Damage makes Render tell the compositor which part of the window changed from
	// the frame before it, when the display has EGL_KHR_swap_buffers_with_damage (or
	// the EXT extension): the scene is compared with the last one of the window, and
	// the frame is still drawn whole. It costs a walk of the scene's layers for each
	// frame.
	Damage bool
	// SkipUnsupported makes Render leave out what the renderer cannot draw yet,
	// and count it in LastFrame, instead of failing with GLNotImplemented. The
	// development examples use it to show what works.
	SkipUnsupported bool
}

// Renderer is the GPU renderer: a gpu.Renderer that draws with OpenGL ES 3.0.
// This is its skeleton (step 3.1): Render clears the surface to the background
// color and presents it; the drawing of scenes arrives in steps 3.2 to 3.5.
//
// Flutter: no counterpart (flutter::Rasterizer)
type Renderer struct {
	open       func(gpu.SurfaceHandle) (*Context, *Surface, error)
	ctx        *Context
	surfaces   map[*glSurface]struct{}
	background [4]float32 // premultiplied
	skip       bool
	last       FrameStats
	res        *resources // the caches of paths, clip masks and images; made at the first frame
	cache      *layerCache
	damage     bool // RendererOpts.Damage
	closed     bool
	// beforePresent, when set, runs after a frame is drawn and before it is
	// presented, while its pixels can still be read back (tests).
	beforePresent func(w, h int) error
}

var _ gpu.Renderer = (*Renderer)(nil)

// glSurface is the gpu.Surface of a Renderer: an EGL surface.
type glSurface struct {
	r       *Renderer
	surface *Surface
	// tracker finds what a frame changes from the one before it, for
	// RendererOpts.Damage.
	tracker *gpu.DamageTracker
}

// Destroy releases the EGL surface; doing it twice is safe.
//
// Flutter: no counterpart
func (s *glSurface) Destroy() {
	delete(s.r.surfaces, s)
	s.surface.Destroy()
}

func newRenderer(open func(gpu.SurfaceHandle) (*Context, *Surface, error), opts ...RendererOpts) *Renderer {
	if len(opts) > 1 {
		panic(errors.New("MultipleOptions", "pass at most one options value", "type", "RendererOpts", "count", len(opts)))
	}
	bg := ui.NewColor(0xFFFFFFFF)
	if len(opts) == 1 && opts[0].HasBackground {
		bg = opts[0].Background
	}
	a := float32(bg.A)
	var cache *layerCache
	if len(opts) == 1 && opts[0].RasterCacheAfter > 0 {
		cache = newLayerCache(opts[0].RasterCacheAfter)
	}
	return &Renderer{
		cache:      cache,
		damage:     len(opts) == 1 && opts[0].Damage,
		open:       open,
		surfaces:   map[*glSurface]struct{}{},
		skip:       len(opts) == 1 && opts[0].SkipUnsupported,
		background: [4]float32{float32(bg.R) * a, float32(bg.G) * a, float32(bg.B) * a, a},
	}
}

// Attach starts drawing into the native window of h. The first call opens the
// GL context (see Open for its failures) and so creates the device; later calls
// add a surface to the same context. It fails with RendererClosed after Close.
//
// Flutter: no counterpart
func (r *Renderer) Attach(h gpu.SurfaceHandle) (gpu.Surface, error) {
	if r.closed {
		return nil, errors.New("RendererClosed", "the renderer was closed")
	}
	var s *Surface
	if r.ctx == nil {
		ctx, first, err := r.open(h)
		if err != nil {
			return nil, err
		}
		r.ctx, s = ctx, first
	} else {
		var err error
		if s, err = r.ctx.NewSurface(h); err != nil {
			return nil, err
		}
	}
	gs := &glSurface{r: r, surface: s, tracker: gpu.NewDamageTracker()}
	r.surfaces[gs] = struct{}{}
	return gs, nil
}

// Render makes the surface current, clears it to the background color, draws the
// scene over it and presents it. A nil scene is an empty one. It fails with
// RendererSurfaceInvalid when s is not a surface of this renderer or was
// destroyed, RendererSizeInvalid when the size is not positive, GLNotImplemented
// when the scene has something the renderer cannot draw yet (unless
// SkipUnsupported is set), and the errors of MakeCurrent and SwapBuffers. Nothing
// is presented when it fails.
//
// Flutter: no counterpart
func (r *Renderer) Render(s gpu.Surface, scene *ui.Scene, size ui.Size, dpr float32) error {
	_, err := r.render(s, scene, size, false)
	return err
}

// RenderCapture is Render that also returns the frame it drew, top row first, read
// back before it was presented (the content of a window is undefined after the
// swap). It also fails with GLReadPixelsFailed.
//
// Flutter: no counterpart
func (r *Renderer) RenderCapture(s gpu.Surface, scene *ui.Scene, size ui.Size, dpr float32) (*image.RGBA, error) {
	return r.render(s, scene, size, true)
}

func (r *Renderer) render(s gpu.Surface, scene *ui.Scene, size ui.Size, capture bool) (*image.RGBA, error) {
	gs, ok := s.(*glSurface)
	if r.closed || !ok || gs.r != r || gs.surface.destroyed {
		return nil, errors.New("RendererSurfaceInvalid", "the surface does not belong to this renderer or was destroyed")
	}
	w, h := int(size.Width), int(size.Height)
	if w <= 0 || h <= 0 {
		return nil, errors.New("RendererSizeInvalid", "the size to render must be positive", "width", size.Width, "height", size.Height)
	}
	if err := r.ctx.MakeCurrent(gs.surface); err != nil {
		return nil, err
	}
	d := r.ctx.dev
	d.BindFramebuffer(nil)
	d.SetScissor(false, 0, 0, 0, 0)
	d.ColorMask(true, true, true, true)
	d.Viewport(0, 0, w, h)
	r.ctx.Clear(r.background[0], r.background[1], r.background[2], r.background[3])
	if err := r.drawScene(scene, w, h, true); err != nil {
		gs.tracker.Invalidate()
		return nil, err
	}
	useDamage := r.damage && r.ctx.CanSwapWithDamage()
	var dmg gpu.Damage
	if useDamage {
		dmg = gs.tracker.Update(scene, w, h)
	}
	if r.beforePresent != nil {
		if err := r.beforePresent(w, h); err != nil {
			return nil, err
		}
	}
	var img *image.RGBA
	if capture {
		var err error
		if img, err = r.ctx.ReadPixels(w, h); err != nil {
			return nil, err
		}
	}
	d.EndFrame()
	if !useDamage {
		return img, r.ctx.SwapBuffers(gs.surface)
	}
	r.last.DamagedPixels = w * h
	if dmg.Full {
		return img, r.swapOrForget(gs, r.ctx.SwapBuffers(gs.surface))
	}
	rects := dmg.Rects
	if len(rects) == 0 {
		// Nothing changed: a pixel is the smallest damage there is.
		rects = []image.Rectangle{image.Rect(0, 0, 1, 1)}
	}
	r.last.DamagedPixels, r.last.DamageRects = 0, len(rects)
	for _, rc := range rects {
		r.last.DamagedPixels += rc.Dx() * rc.Dy()
	}
	return img, r.swapOrForget(gs, r.ctx.SwapBuffersWithDamage(gs.surface, h, rects))
}

// swapOrForget returns the error of a swap, after forgetting the surface's last frame
// if it failed: what the compositor shows is then not that frame.
func (r *Renderer) swapOrForget(gs *glSurface, err error) error {
	if err != nil {
		gs.tracker.Invalidate()
	}
	return err
}

// Invalidate says that the window of the surface lost its content (it was exposed),
// so that its next frame reports the whole window as changed.
//
// Flutter: no counterpart
func (r *Renderer) Invalidate(s gpu.Surface) {
	if gs, ok := s.(*glSurface); ok && gs.r == r {
		gs.tracker.Invalidate()
	}
}

// LastFrame says what the last call of Render drew and left out.
//
// Flutter: no counterpart
func (r *Renderer) LastFrame() FrameStats { return r.last }

// Snapshot draws the scene into a new image of the size, without a window: the work
// of Picture.ToImage and Scene.ToImage. The scene is drawn over transparent pixels
// into a pooled framebuffer, which is read back; the pixels are premultiplied, top
// row first. The context must exist, so a surface has to have been attached at
// some point; it is made current with one of the live surfaces, or without any.
// It fails with RendererClosed, RendererNotAttached, RendererSizeInvalid,
// GLNotImplemented as Render does, and the errors of MakeCurrent, of the
// framebuffer's pool and of reading the pixels back.
//
// Flutter: no counterpart
func (r *Renderer) Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error) {
	if r.closed {
		return nil, errors.New("RendererClosed", "the renderer was closed")
	}
	if r.ctx == nil {
		return nil, errors.New("RendererNotAttached", "a snapshot needs the renderer's GL context, which the first Attach creates")
	}
	w, h := int(size.Width), int(size.Height)
	if w <= 0 || h <= 0 {
		return nil, errors.New("RendererSizeInvalid", "the size to render must be positive", "width", size.Width, "height", size.Height)
	}
	var err error
	for s := range r.surfaces {
		if err = r.ctx.MakeCurrent(s.surface); err == nil {
			break
		}
	}
	if len(r.surfaces) == 0 {
		err = r.ctx.MakeCurrentSurfaceless()
	}
	if err != nil {
		return nil, err
	}
	d := r.ctx.dev
	fb, err := d.AcquireFramebuffer(w, h, false)
	if err != nil {
		return nil, err
	}
	defer d.ReleaseFramebuffer(fb)
	defer d.BindFramebuffer(nil)
	d.BindFramebuffer(fb)
	d.SetScissor(false, 0, 0, 0, 0)
	d.ColorMask(true, true, true, true)
	d.Viewport(0, 0, w, h)
	r.ctx.Clear(0, 0, 0, 0)
	if err := r.drawScene(scene, w, h, false); err != nil {
		return nil, err
	}
	img, err := r.ctx.ReadPixels(w, h)
	if err != nil {
		return nil, err
	}
	d.EndFrame()
	return img, nil
}

var _ ui.Rasterizer = (*Renderer)(nil)

// RasterizeScene draws the scene into a new image of width x height pixels, as
// Snapshot does: the ui.Rasterizer of the GPU renderer, for Picture.ToImage and
// Scene.ToImage.
//
// Flutter: no counterpart
func (r *Renderer) RasterizeScene(scene *ui.Scene, width, height int) (*ui.Image, error) {
	img, err := r.Snapshot(scene, ui.Size{Width: float64(width), Height: float64(height)})
	if err != nil {
		return nil, err
	}
	return ui.NewImageFromRGBA(img.Pix, width, height), nil
}

// Close destroys every surface, the device and the context. Closing twice does
// nothing.
//
// Flutter: no counterpart
func (r *Renderer) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	for s := range r.surfaces {
		s.surface.Destroy()
	}
	r.surfaces = nil
	if r.ctx != nil {
		if r.cache != nil {
			r.cache.release(r.ctx.dev)
		}
		if r.res != nil {
			// The context must be current to delete what the caches hold; the last
			// surface is destroyed already, so Close does it with the display only.
			r.res.release()
			r.res = nil
		}
		r.ctx.Close()
	}
	return nil
}
