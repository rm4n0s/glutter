package explicit

import (
	"image"
	"log/slog"
	"time"

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
	// RasterCacheAfter turns on the raster cache: a layer that a scene retains (it is in this
	// many consecutive scenes) is drawn into a texture that later frames draw instead of its
	// contents (SPEC §9.1 "Raster cache"). Zero, the default, keeps nothing. The engine sets 3:
	// the layer of a RepaintBoundary that the framework did not repaint for three frames.
	RasterCacheAfter int
	// Now is the clock of the device-loss rule (a second loss within 10 seconds of the first
	// ends the app); time.Now when nil. Tests set it.
	Now func() time.Time
	// Log is where a lost device is reported, at Info level; nothing is logged when nil.
	Log *slog.Logger
	// OnCapture receives a capture whose read-back finishes after Draw returned, on a device
	// whose reads are asynchronous (WebGPU, SPEC §9.5): Draw then returns a nil image and a nil
	// error for the frame that asked, and OnCapture is called later, on the thread that
	// calls Draw, with the image or the error of the read. A device that finishes reads inside
	// Submit (Vulkan) returns the image from Draw as before. Without OnCapture a capture that is
	// late fails with RendererCaptureFailed.
	OnCapture func(img *image.RGBA, err error)
	// SkipUnsupported makes Draw leave out what the renderer cannot draw yet, and count
	// it in LastFrame, instead of failing with GPUNotImplemented. The development
	// examples use it to show what works.
	SkipUnsupported bool
}

// Renderer is the renderer for the explicit APIs (SPEC §9.3): the engine's renderer on
// Vulkan and WebGPU. Each window has a swapchain and an offscreen frame texture; a frame
// is drawn into the frame texture, then blitted into the swapchain image and presented.
// It draws everything the GLES renderer draws (SPEC §9.1): shapes, paths, clips, images, text,
// gradients, color filters, blend modes, layers, blurs, shadows and image filters, with the raster
// cache and damage tracking, and snapshots without a window (step 20.6).
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
	// pipes holds the pipelines of the primitives, one for each fixed blend.
	pipes map[pipeKey]Pipeline
	// mipSampler is the linear sampler of the downsampling pass, and imageSampler the one that
	// image draws use, which picks a mip level.
	mipSampler, imageSampler Sampler
	// res holds the textures that outlive a frame; made at the first frame.
	res *resources
	// layerPool are the free textures of layers, and dst the texture that holds a copy of the
	// pixels under a draw that blends in the shader.
	layerPool []*pooledTexture
	dst       Texture
	// filter is the pipeline of the image filter passes, made when first needed, and cache the
	// raster cache, nil when RendererOpts.RasterCacheAfter is 0.
	filter Pipeline
	cache  *gpu.LayerCache[Texture]
	// slots are the buffers of the frames in flight: the frame with number n uses slot
	// n % FramesInFlight, which the device has finished with when Begin returns.
	slots   [FramesInFlight]frameSlot
	frameNo int
	last    FrameStats
	// lost is whether the device was lost before, and lostAt when.
	lost   bool
	lostAt time.Time
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
	r := &Renderer{open: open, blit: map[TextureFormat]Pipeline{}, pipes: map[pipeKey]Pipeline{}, windows: map[gpu.SurfaceHandle]*window{}}
	if len(opts) == 1 {
		r.opts = opts[0]
		if r.opts.RasterCacheAfter > 0 {
			r.cache = gpu.NewLayerCache[Texture](r.opts.RasterCacheAfter)
		}
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
	img, err := r.draw(h, width, height, scene, capture)
	if IsDevicePending(err) {
		return nil, nil // the backend is getting a device and says when it has one
	}
	if err == nil || r.dev == nil || !isDeviceLost(err) {
		return img, err
	}
	// The device was lost (SPEC §9.4): every resource is made again, on a new device, and the frame
	// is drawn again. A second loss within lostWindow ends the app.
	now := r.now()
	if r.lost && now.Sub(r.lostAt) < lostWindow {
		return nil, err
	}
	r.lost, r.lostAt = true, now
	r.logInfo("the GPU device was lost, making it again", "error", err)
	r.releaseDevice()
	img, err = r.draw(h, width, height, scene, capture)
	if IsDevicePending(err) {
		return nil, nil
	}
	return img, err
}

// lostWindow is how long a second loss of the device after a first one ends the app.
const lostWindow = 10 * time.Second

// isDeviceLost says whether the error is the loss of the device, from Vulkan or WebGPU.
func isDeviceLost(err error) bool {
	appErr, ok := errors.FromError(err)
	return ok && (appErr.Tag == "VulkanDeviceLost" || appErr.Tag == "GPUDeviceLost")
}

// now is the clock of the device-loss rule.
func (r *Renderer) now() time.Time {
	if r.opts.Now != nil {
		return r.opts.Now()
	}
	return time.Now()
}

func (r *Renderer) draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
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
	defer func() {
		if r.res != nil {
			r.res.endFrame()
			r.sweepLayers()
		}
	}()
	fb, err := r.build(scene, width, height, true)
	if err != nil {
		w.tracker.Invalidate()
		return nil, err
	}
	img, err := r.frame(w, fb, width, height, scene, capture)
	if err != nil {
		w.tracker.Invalidate()
		return nil, err
	}
	if r.cache != nil {
		r.cache.EndFrame(r.releaseLayer)
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
func (r *Renderer) frame(w *window, fb *frameBuilder, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	enc, err := r.dev.Begin()
	if err != nil {
		return nil, err
	}
	slot := &r.slots[r.frameNo%FramesInFlight]
	r.frameNo++
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

	// The frame texture: the background, then the scene.
	if err := r.drawScene(enc, slot, w.frame, fb, width, height, bg); err != nil {
		_ = enc.Submit()
		return nil, err
	}

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
	var returned, sync bool
	if capture {
		enc.ReadTexture(w.frame, image.Rect(0, 0, width, height), func(i *image.RGBA, err error) {
			if !returned {
				shot, sync = i, true
				return
			}
			// The device finished the read after Draw returned.
			if r.opts.OnCapture != nil {
				r.opts.OnCapture(i, err)
			}
		})
	}
	damage := r.damage(w, scene, width, height)
	r.last.DamagedPixels, r.last.DamageRects = width*height, 0
	if r.opts.Damage && damage != nil {
		r.last.DamagedPixels, r.last.DamageRects = 0, len(damage)
		for _, rc := range damage {
			r.last.DamagedPixels += rc.Dx() * rc.Dy()
		}
	}
	err = enc.Submit()
	returned = true
	if err != nil {
		return nil, err
	}
	late := capture && !sync && r.opts.OnCapture != nil
	if capture && shot == nil && !late {
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

// build flattens the scene and makes its instances, and the textures they need. It fails with
// GPUNotImplemented for a feature the renderer cannot draw, unless SkipUnsupported is set, and
// records what the frame is in LastFrame.
func (r *Renderer) build(scene *ui.Scene, width, height int, useCache bool) (*frameBuilder, error) {
	r.last = FrameStats{}
	if r.res == nil {
		res, err := newResources(r.dev)
		if err != nil {
			return nil, err
		}
		r.res = res
	}
	r.res.beginFrame()
	tiles0, clips0, images0 := r.res.masks.Rasterized, r.res.clipMasksBuilt, r.res.imagesUploaded
	fb := &frameBuilder{w: width, h: height, skip: r.opts.SkipUnsupported, res: r.res}
	if scene != nil {
		fo := gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, float64(width), float64(height))}
		if useCache && r.cache != nil {
			fb.cache = r.cache
			r.cache.BeginFrame(scene.Root())
			fo.Boundary = r.cache.Boundary
		}
		list, err := gpu.Flatten(scene, fo)
		if err != nil {
			return nil, err
		}
		for _, c := range list.Commands {
			if err := fb.command(c); err != nil {
				return nil, err
			}
		}
	}
	fb.stats.Instances = fb.items(pipeShape) + fb.items(pipeMask) + fb.items(pipeImage)
	fb.stats.Vertices, fb.stats.Batches = fb.items(pipeVertices), len(fb.calls)
	fb.stats.TilesRasterized = r.res.masks.Rasterized - tiles0
	fb.stats.ClipMasksBuilt = r.res.clipMasksBuilt - clips0
	fb.stats.ImagesUploaded = r.res.imagesUploaded - images0
	r.last = fb.stats
	return fb, nil
}

// LastFrame says what the last call of Draw drew and left out.
//
// Flutter: no counterpart
func (r *Renderer) LastFrame() FrameStats { return r.last }

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
	r.releaseDevice()
}

// releaseDevice destroys everything that belongs to the device, the device included, and leaves
// the renderer as it was before the first frame: the next frame opens a device again. Errors of
// the destruction are the lost device's business and are ignored.
func (r *Renderer) releaseDevice() {
	for h := range r.windows {
		r.Release(h)
	}
	for f, p := range r.blit {
		p.Destroy()
		delete(r.blit, f)
	}
	r.destroyPools()
	if r.cache != nil {
		r.cache.Release(func(t Texture) { t.Destroy() })
		r.cache = gpu.NewLayerCache[Texture](r.opts.RasterCacheAfter)
	}
	if r.filter != nil {
		r.filter.Destroy()
		r.filter = nil
	}
	if r.res != nil {
		r.res.release()
		r.res = nil
	}
	for k, p := range r.pipes {
		p.Destroy()
		delete(r.pipes, k)
	}
	if r.mipSampler != nil {
		r.mipSampler.Destroy()
		r.mipSampler = nil
	}
	if r.imageSampler != nil {
		r.imageSampler.Destroy()
		r.imageSampler = nil
	}
	for i := range r.slots {
		r.slots[i].destroy()
	}
	if r.sampler != nil {
		r.sampler.Destroy()
		r.sampler = nil
	}
	if r.dev != nil {
		_ = r.dev.Close()
		r.dev = nil
	}
}

// logInfo logs at Info level when the renderer has a logger.
func (r *Renderer) logInfo(msg string, args ...any) {
	if r.opts.Log != nil {
		r.opts.Log.Info(msg, args...)
	}
}
