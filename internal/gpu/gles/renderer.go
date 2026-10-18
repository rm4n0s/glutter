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
}

// Destroy releases the EGL surface; doing it twice is safe.
//
// Flutter: no counterpart
func (s *glSurface) Destroy() {
	delete(s.r.surfaces, s)
	s.surface.Destroy()
}

// NewRenderer returns a renderer that has no GL context yet: the first Attach
// creates it. Passing more than one RendererOpts panics with MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(opts ...RendererOpts) *Renderer {
	return newRenderer(Open, opts...)
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
	return &Renderer{
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
	gs := &glSurface{r: r, surface: s}
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
	if err := r.drawScene(scene, w, h); err != nil {
		return nil, err
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
	return img, r.ctx.SwapBuffers(gs.surface)
}

// LastFrame says what the last call of Render drew and left out.
//
// Flutter: no counterpart
func (r *Renderer) LastFrame() FrameStats { return r.last }

// Snapshot is not available yet: it arrives with step 3.6. It fails with
// GLNotImplemented.
//
// Flutter: no counterpart
func (r *Renderer) Snapshot(_ *ui.Scene, _ ui.Size) (*image.RGBA, error) {
	return nil, errors.New("GLNotImplemented", "the GPU renderer cannot make snapshots yet (step 3.6)")
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
		r.ctx.Close()
	}
	return nil
}
