package explicit

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
	// Background is the color each frame is cleared to when HasBackground is set;
	// otherwise opaque white.
	Background ui.Color
	// HasBackground says that Background is set.
	HasBackground bool
	// Damage tells the compositor which part of a window changed in each frame, when
	// the device can (SPEC §9.1 "Damage", §9.4).
	Damage bool
}

// Renderer is the renderer for the explicit APIs (SPEC §9.3): the engine's renderer on
// Vulkan and WebGPU. Each window has a swapchain and an offscreen frame texture; a frame
// is drawn into the frame texture, then blitted into the swapchain image and presented.
// This is its skeleton (step 20.2): a frame is the background; the drawing of scenes
// arrives with steps 20.3 to 20.6.
//
// The device is opened at the first frame, with the handle of the first window, because
// the choice of a Vulkan device depends on the window.
//
// Flutter: no counterpart (flutter::Rasterizer)
type Renderer struct {
	open    func(gpu.SurfaceHandle) (Device, error)
	opts    RendererOpts
	dev     Device
	sampler Sampler
	blit    map[TextureFormat]Pipeline
	windows map[gpu.SurfaceHandle]*window
	closed  bool
}

// window is what the renderer keeps for one window.
type window struct {
	sc      Swapchain
	frame   Texture
	fw, fh  int
	tracker *gpu.DamageTracker
}

// NewRenderer returns a renderer that has no device yet: open makes it at the first
// frame, from the handle of the first window.
//
// Flutter: no counterpart
func NewRenderer(open func(gpu.SurfaceHandle) (Device, error), opts ...RendererOpts) *Renderer {
	if len(opts) > 1 {
		panic(errors.New("MultipleOptions", "pass at most one options value", "type", "RendererOpts", "count", len(opts)))
	}
	r := &Renderer{open: open, blit: map[TextureFormat]Pipeline{}, windows: map[gpu.SurfaceHandle]*window{}}
	if len(opts) == 1 {
		r.opts = opts[0]
	}
	return r
}

// background is the color a frame is cleared to.
func (r *Renderer) background() ui.Color {
	if r.opts.HasBackground {
		return r.opts.Background
	}
	return ui.NewColor(0xFFFFFFFF)
}

// Describe names the renderer and its device, as the device does, or "" before the first
// frame.
//
// Flutter: no counterpart
func (r *Renderer) Describe() string {
	if d, ok := r.dev.(interface{ Description() string }); ok {
		return d.Description()
	}
	return ""
}

// Draw draws one frame into the window of the handle, whose size is width x height
// pixels, and presents it; with capture it also reads the frame back, top row first.
// The first call opens the device and the swapchain (their errors are returned
// unchanged, such as VulkanLoadFailed or VulkanNoDevice). A swapchain that is out of date
// is configured again and the frame retried once.
//
// Flutter: no counterpart
func (r *Renderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.closed {
		return nil, errors.New("RendererClosed", "the renderer was closed")
	}
	if width <= 0 || height <= 0 {
		return nil, errors.New("RendererSizeInvalid", "the size to render must be positive", "width", width, "height", height)
	}
	if r.dev == nil {
		dev, err := r.open(h)
		if err != nil {
			return nil, err
		}
		sampler, err := dev.NewSampler(SamplerDesc{MinFilter: FilterNearest, MagFilter: FilterNearest})
		if err != nil {
			_ = dev.Close()
			return nil, err
		}
		r.dev, r.sampler = dev, sampler
	}
	w, err := r.window(h)
	if err != nil {
		return nil, err
	}
	if err := w.sc.Configure(width, height); err != nil {
		w.tracker.Invalidate()
		return nil, err
	}
	if w.frame == nil || w.fw != width || w.fh != height {
		if w.frame != nil {
			w.frame.Destroy()
		}
		w.frame, err = r.dev.NewTexture(TextureDesc{Width: width, Height: height, Format: TextureFormatRGBA8Unorm, MipLevels: 1,
			Usage: TextureRenderTarget | TextureSampled | TextureCopySource})
		if err != nil {
			w.frame = nil
			return nil, err
		}
		w.fw, w.fh = width, height
		w.tracker.Invalidate()
	}
	img, err := r.frame(w, width, height, scene, capture)
	if err != nil {
		w.tracker.Invalidate()
		return nil, err
	}
	return img, nil
}

// window returns the state of the window, making its swapchain on first use.
func (r *Renderer) window(h gpu.SurfaceHandle) (*window, error) {
	if w, ok := r.windows[h]; ok {
		return w, nil
	}
	sc, err := r.dev.NewSwapchain(h)
	if err != nil {
		return nil, err
	}
	w := &window{sc: sc, tracker: gpu.NewDamageTracker()}
	r.windows[h] = w
	return w, nil
}

// frame records, submits and presents one frame.
func (r *Renderer) frame(w *window, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	enc, err := r.dev.Begin()
	if err != nil {
		return nil, err
	}
	img, err := w.sc.Acquire()
	if IsSwapchainOutOfDate(err) {
		if err = w.sc.Configure(width, height); err == nil {
			img, err = w.sc.Acquire()
		}
	}
	if err != nil {
		_ = enc.Submit() // an empty frame: the encoder must be closed
		return nil, err
	}
	blit, err := r.blitPipeline(img.Desc().Format)
	if err != nil {
		_ = enc.Submit()
		return nil, err
	}
	bg := r.background()

	// The frame texture: the background, and later the scene (steps 20.3 to 20.6).
	enc.BeginPass(w.frame, LoadClear, bg).End()

	// The blit into the swapchain image, whatever its format.
	group, err := r.dev.NewBindGroup(blit, 0, []Binding{{Binding: 0, Texture: w.frame}, {Binding: 1, Sampler: r.sampler}})
	if err != nil {
		_ = enc.Submit()
		return nil, err
	}
	pass := enc.BeginPass(img, LoadClear, bg)
	pass.SetPipeline(blit)
	pass.SetBindGroup(0, group)
	pass.Draw(3, 1, 0, 0)
	pass.End()

	var shot *image.RGBA
	if capture {
		enc.ReadTexture(w.frame, image.Rect(0, 0, width, height), func(i *image.RGBA, _ error) { shot = i })
	}
	damage := r.damage(w, scene, width, height)
	if err := enc.Submit(); err != nil {
		return nil, err
	}
	if capture && shot == nil {
		return nil, errors.New("RendererCaptureFailed", "the device did not deliver the frame it was asked to read back")
	}
	if err := w.sc.Present(damage); err != nil {
		return nil, err
	}
	return shot, nil
}

// damage returns the rectangles to tell the compositor about, from the top left; nil
// means the whole window. A frame that changed nothing is a pixel, the smallest damage.
func (r *Renderer) damage(w *window, scene *ui.Scene, width, height int) []image.Rectangle {
	if !r.opts.Damage {
		return nil
	}
	d := w.tracker.Update(scene, width, height)
	if d.Full {
		return nil
	}
	if len(d.Rects) == 0 {
		return []image.Rectangle{image.Rect(0, 0, 1, 1)}
	}
	return d.Rects
}

// blitPipeline returns the pipeline that copies the frame texture into an image of the
// format, made once.
func (r *Renderer) blitPipeline(format TextureFormat) (Pipeline, error) {
	if p, ok := r.blit[format]; ok {
		return p, nil
	}
	p, err := r.dev.NewPipeline(PipelineDesc{Shader: ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: format})
	if err != nil {
		return nil, err
	}
	r.blit[format] = p
	return p, nil
}

// Release destroys what the renderer keeps for the window of the handle, which must
// still exist; a handle that was never drawn into is ignored.
//
// Flutter: no counterpart
func (r *Renderer) Release(h gpu.SurfaceHandle) {
	w, ok := r.windows[h]
	if !ok {
		return
	}
	delete(r.windows, h)
	if w.frame != nil {
		w.frame.Destroy()
	}
	w.sc.Destroy()
}

// Invalidate says that the window lost its content, so that its next frame is presented
// whole; a handle that was never drawn into is ignored.
//
// Flutter: no counterpart
func (r *Renderer) Invalidate(h gpu.SurfaceHandle) {
	if w, ok := r.windows[h]; ok {
		w.tracker.Invalidate()
	}
}

// Close destroys the windows' state and the device. It can be called twice.
//
// Flutter: no counterpart
func (r *Renderer) Close() {
	if r.closed {
		return
	}
	r.closed = true
	for h := range r.windows {
		r.Release(h)
	}
	for _, p := range r.blit {
		p.Destroy()
	}
	if r.sampler != nil {
		r.sampler.Destroy()
	}
	if r.dev != nil {
		_ = r.dev.Close()
	}
}
