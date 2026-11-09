package shmwindow

import (
	"encoding/binary"
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// maxSize is the largest width or height of a window that the renderer draws, in pixels (the limit
// of the software renderer).
const maxSize = 1 << 15

// maxHistory is how many frames of damage a surface remembers: a buffer older than this is copied
// whole. A pool has two or three buffers, so their ages are 1 to 4.
const maxHistory = 8

// Options are the optional parameters of NewRenderer.
//
// Flutter: no counterpart
type Options struct {
	// Background is what shows where the scene draws nothing. The window is opaque, so only the
	// color's red, green and blue count; the zero value is black.
	Background ui.Color
}

// Renderer is a gpu.Renderer that draws with soft into the buffers of a gpu.PixelTarget.
//
// Flutter: shell/platform/linux/fl_compositor_software.cc FlCompositorSoftware
type Renderer struct {
	targets func(gpu.SurfaceHandle) (gpu.PixelTarget, bool)
	bg      [3]uint8
	closed  bool
}

var _ gpu.Renderer = (*Renderer)(nil)

// NewRenderer makes a renderer. targets gives the gpu.PixelTarget of the window that a surface
// handle names, and false when there is none. More than one options value panics with
// MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(targets func(gpu.SurfaceHandle) (gpu.PixelTarget, bool), opts ...Options) *Renderer {
	var o Options
	switch len(opts) {
	case 0:
	case 1:
		o = opts[0]
	default:
		panic(errors.New(tagMultipleOptions, "NewRenderer takes at most one Options value", "count", len(opts)))
	}
	return &Renderer{targets: targets, bg: [3]uint8{uint8(o.Background.Red()), uint8(o.Background.Green()), uint8(o.Background.Blue())}}
}

// surface is the surface of one window: the pixels it shows, what each frame changed from the one
// before, and the target that presents them.
type surface struct {
	r       *Renderer
	h       gpu.SurfaceHandle
	target  gpu.PixelTarget
	tracker *gpu.DamageTracker
	// canvas holds the frame the window shows, in premultiplied RGBA over the background, so it
	// is opaque everywhere.
	canvas *image.RGBA
	// history is the damage of the frames that were presented with a buffer, oldest first.
	history   [][]image.Rectangle
	destroyed bool
}

// Destroy forgets the window. It is safe to call twice.
//
// Flutter: no counterpart
func (s *surface) Destroy() {
	s.destroyed = true
	s.canvas, s.history = nil, nil
}

// Attach starts drawing into the window that the handle names, which must be of kind
// gpu.SurfaceKindWayland, gpu.SurfaceKindX11 or gpu.SurfaceKindMetalLayer (the layer of a macOS window,
// which shows IOSurfaces, SPEC §29.5) and have a target. It fails with RendererClosed
// after Close, with SurfaceKindUnsupported for another kind and with PixelTargetMissing for a
// window with no target.
//
// Flutter: no counterpart
func (r *Renderer) Attach(h gpu.SurfaceHandle) (gpu.Surface, error) {
	if r.closed {
		return nil, errors.New(tagRendererClosed, "the renderer was closed")
	}
	if h.Kind != gpu.SurfaceKindWayland && h.Kind != gpu.SurfaceKindX11 && h.Kind != gpu.SurfaceKindMetalLayer {
		return nil, errors.New(tagSurfaceKindUnsupported, "shmwindow draws into a Wayland window, an X11 window or a macOS layer only", "kind", string(h.Kind))
	}
	t, ok := r.targets(h)
	if !ok || t == nil {
		return nil, errors.New(tagPixelTargetMissing, "the window shows no pixels of its own", "kind", string(h.Kind))
	}
	return &surface{r: r, h: h, target: t, tracker: gpu.NewDamageTracker()}, nil
}

// Invalidate says that the window lost its content, so that the next frame is drawn and presented
// whole. A surface that is not this renderer's is ignored.
//
// Flutter: no counterpart
func (r *Renderer) Invalidate(s gpu.Surface) {
	if surf, ok := s.(*surface); ok && surf.r == r {
		surf.tracker.Invalidate()
	}
}

// Render draws the scene into the surface at the size in device pixels and presents it. A nil
// scene draws the background only. The scene's coordinates are already in device pixels, so dpr
// is not used.
//
// Flutter: no counterpart
func (r *Renderer) Render(s gpu.Surface, scene *ui.Scene, size ui.Size, dpr float32) error {
	_, err := r.RenderCapture(s, scene, size, dpr, false)
	return err
}

// RenderCapture is Render that also returns, when capture is set, the frame it presented, top row
// first. The image belongs to the caller.
//
// Flutter: no counterpart
func (r *Renderer) RenderCapture(s gpu.Surface, scene *ui.Scene, size ui.Size, _ float32, capture bool) (*image.RGBA, error) {
	surf, ok := s.(*surface)
	if !ok || surf.r != r || surf.destroyed {
		return nil, errors.New(tagSurfaceInvalid, "the surface was not attached by this renderer or was destroyed")
	}
	w, h := int(size.Width), int(size.Height)
	if w <= 0 || h <= 0 || w > maxSize || h > maxSize {
		return nil, errors.New(tagSizeInvalid, "the window size is not valid", "width", w, "height", h)
	}
	if scene == nil {
		scene = ui.NewSceneBuilder().Build()
	}
	bounds := image.Rect(0, 0, w, h)
	if surf.canvas == nil || surf.canvas.Rect != bounds {
		surf.canvas, surf.history = image.NewRGBA(bounds), nil
	}
	damage := surf.tracker.Update(scene, w, h)
	rects := damage.Rects
	if damage.Full {
		rects = []image.Rectangle{bounds}
	}
	for _, rc := range rects {
		fillRect(surf.canvas, rc, r.bg)
		if err := soft.RenderInto(surf.canvas, scene, rc); err != nil {
			surf.fail()
			return nil, err
		}
	}
	if err := surf.present(rects); err != nil {
		surf.fail()
		return nil, err
	}
	if !capture {
		return nil, nil
	}
	img := image.NewRGBA(bounds)
	copy(img.Pix, surf.canvas.Pix)
	return img, nil
}

// fail forgets what the pixels and the window hold, after an error: the next frame is drawn and
// presented whole.
func (s *surface) fail() {
	s.tracker.Invalidate()
	s.history = nil
}

// present puts the frame in the target. rects are what the frame changed from the one before.
func (s *surface) present(rects []image.Rectangle) error {
	if len(rects) == 0 {
		return s.target.PresentPixels(nil) // nothing changed: no buffer, and the window's frame pacing goes on
	}
	bounds := s.canvas.Rect
	buf, err := s.target.AcquirePixels(bounds.Dx(), bounds.Dy())
	if err != nil {
		return err
	}
	if buf.Width != bounds.Dx() || buf.Height != bounds.Dy() || buf.Stride < 4*buf.Width || len(buf.Pix) < buf.Stride*(buf.Height-1)+4*buf.Width {
		return errors.New(tagPixelBufferInvalid, "the buffer is not the size that was asked for",
			"width", buf.Width, "height", buf.Height, "stride", buf.Stride, "bytes", len(buf.Pix), "wanted_width", bounds.Dx(), "wanted_height", bounds.Dy())
	}
	for _, rc := range missing(rects, s.history, buf.Age, bounds) {
		copyBGRX(buf, s.canvas, rc)
	}
	if err := s.target.PresentPixels(rects); err != nil {
		return err
	}
	s.history = append(s.history, rects)
	if len(s.history) > maxHistory {
		s.history = s.history[len(s.history)-maxHistory:]
	}
	return nil
}

// missing lists the rectangles of a buffer of the given age that must be written for it to show the
// frame whose damage is rects: that damage and the damage of the frames since the buffer was shown,
// which are the last age-1 entries of history. A buffer with no defined pixels (age 0) or older than
// the history is written whole. The rectangles are merged where they overlap, so that no pixel is
// copied twice.
func missing(rects []image.Rectangle, history [][]image.Rectangle, age int, bounds image.Rectangle) []image.Rectangle {
	if age <= 0 || age-1 > len(history) {
		return []image.Rectangle{bounds}
	}
	out := append([]image.Rectangle(nil), rects...)
	for _, past := range history[len(history)-(age-1):] {
		out = append(out, past...)
	}
	for merged := true; merged; {
		merged = false
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				if out[i].Overlaps(out[j]) {
					out[i] = out[i].Union(out[j])
					out = append(out[:j], out[j+1:]...)
					merged = true
					j--
				}
			}
		}
	}
	for i := range out {
		out[i] = out[i].Intersect(bounds)
	}
	return out
}

// fillRect sets the pixels of the rectangle to the opaque background color.
func fillRect(img *image.RGBA, r image.Rectangle, bg [3]uint8) {
	r = r.Intersect(img.Rect)
	if r.Empty() {
		return
	}
	first := img.Pix[img.PixOffset(r.Min.X, r.Min.Y):][:4*r.Dx()]
	for x := 0; x < len(first); x += 4 {
		first[x], first[x+1], first[x+2], first[x+3] = bg[0], bg[1], bg[2], 255
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		copy(img.Pix[img.PixOffset(r.Min.X, y):], first)
	}
}

// copyBGRX writes the rectangle of the canvas, in RGBA, into the buffer, in blue, green, red and a
// byte of 255. The canvas is opaque.
func copyBGRX(dst gpu.PixelBuffer, src *image.RGBA, r image.Rectangle) {
	r = r.Intersect(src.Rect)
	n := r.Dx()
	if n <= 0 {
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		s := src.Pix[src.PixOffset(r.Min.X, y):][:4*n]
		d := dst.Pix[y*dst.Stride+4*r.Min.X:][:4*n]
		for x := 0; x < len(s); x += 4 {
			v := binary.LittleEndian.Uint32(s[x:])
			binary.LittleEndian.PutUint32(d[x:], 0xFF000000|(v&0xFF)<<16|v&0xFF00|(v>>16)&0xFF)
		}
	}
}

// Snapshot draws the scene into a new image of the size, without a window.
//
// Flutter: no counterpart
func (r *Renderer) Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error) {
	if scene == nil {
		scene = ui.NewSceneBuilder().Build()
	}
	return soft.Render(scene, int(size.Width), int(size.Height))
}

// Close stops the renderer; its surfaces are destroyed by their owners. It is safe to call twice.
//
// Flutter: no counterpart
func (r *Renderer) Close() error {
	r.closed = true
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMultipleOptions        = "MultipleOptions"
	tagPixelBufferInvalid     = "PixelBufferInvalid"
	tagPixelTargetMissing     = "PixelTargetMissing"
	tagRendererClosed         = "RendererClosed"
	tagSizeInvalid            = "SizeInvalid"
	tagSurfaceInvalid         = "SurfaceInvalid"
	tagSurfaceKindUnsupported = "SurfaceKindUnsupported"
)
