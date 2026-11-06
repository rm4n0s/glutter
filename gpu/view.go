package gpu

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ViewFrame is what a View gives its paint function: the device, an encoder to record into, the targets and the
// numbers of the frame (SPEC §27.5). The view submits the encoder when the function returns.
//
// Flutter: no counterpart
type ViewFrame struct {
	// Device is the device of the app.
	Device *Device
	// Encoder is the encoder of the frame. Record the passes into it; the view finishes and submits it.
	Encoder *CommandEncoder
	// Color is the color target: ViewOpts.Format, the size of the view in pixels, and multisampled when
	// ViewOpts.SampleCount is 4. Draw into Color.View().
	Color *Texture
	// Resolve is the texture that a multisampled Color is resolved into, which the window shows (set it as the
	// ResolveTarget of the color attachment); nil when the view is not multisampled.
	Resolve *Texture
	// Depth is the depth target, of ViewOpts.DepthFormat and the same sample count as Color; nil when the view
	// has no depth format.
	Depth *Texture
	// Width and Height are the size of the targets in pixels: the box of the view times the device pixel ratio,
	// rounded.
	Width, Height int
	// DevicePixelRatio is the number of pixels per logical pixel.
	DevicePixelRatio float64
	// Time is the time of the frame: the elapsed time of the view's Ticker when ViewOpts.Continuous is set, else
	// the time of the frame that draws the view.
	Time time.Duration
}

// ViewController asks a View to draw (SPEC §27.5).
//
// Flutter: no counterpart
type ViewController struct {
	s *viewSurface
}

// NewViewController creates a controller. Pass it as ViewOpts.Controller.
//
// Flutter: no counterpart
func NewViewController() *ViewController { return &ViewController{} }

// MarkNeedsPaint asks the view to draw once more: its paint function runs in the next frame, which is scheduled if
// there is none. It does nothing while the controller has no view, or the view has no device.
//
// Flutter: no counterpart
func (c *ViewController) MarkNeedsPaint() {
	if c.s != nil {
		c.s.markDirty()
	}
}

// ViewOpts are the optional parameters of NewView.
//
// Flutter: no counterpart
type ViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Format is the format of the color target: RGBA8Unorm (sRGB-encoded values; the default) or RGBA16Float
	// (linear values, which the window encodes to sRGB).
	Format TextureFormat
	// DepthFormat is the format of the depth target: Depth32Float or Depth24PlusStencil8. The default is none.
	DepthFormat TextureFormat
	// SampleCount is 1 (the default) or 4.
	SampleCount int
	// Continuous draws at every frame, at the display's rate, through a Ticker that follows TickerMode. Without it
	// the view draws when it is created, resized, or marked by Controller.
	Continuous bool
	// Controller asks the view to draw.
	Controller *ViewController
	// OnInit is called with the device when the view has one: at the first frame, and again after a device was
	// lost and replaced. Make the resources of the view in it.
	OnInit func(d *Device)
	// OnDispose is called for each OnInit: when the view leaves the tree, and when its device is lost (the
	// resources of that device are gone). Release the resources in it.
	OnDispose func()
	// Fallback is the widget shown when the app has no GPU (GPUUnavailable), and until a lost device is replaced.
	// The default is an empty box of the view's size.
	Fallback widgets.Widget
	// SemanticsLabel describes the view to assistive technology; the default is none.
	SemanticsLabel string
}

// View is a widget that owns a render target the size of its box in device pixels, with an optional depth target and
// multisampling, and calls a function to draw into it when it must (SPEC §27.5). The window shows the target with no
// copy. The function runs on the UI thread after the paint phase and before the frame is composited, in paint order;
// a view that is not Continuous and not marked draws nothing, so an idle view costs no CPU. A panic in the function
// is reported as a paint error, and the view keeps its last picture.
//
// Flutter: no counterpart
type View struct {
	widgets.StatefulWidget
	onPaint func(f *ViewFrame)
	o       ViewOpts
}

// NewView creates a view that draws with onPaint.
//
// Flutter: no counterpart
func NewView(onPaint func(f *ViewFrame), opts ...ViewOpts) *View {
	if len(opts) > 1 {
		panic(errors.New(TagGPUViewInvalid, "gpu.NewView takes at most one ViewOpts", "count", len(opts)))
	}
	var o ViewOpts
	if len(opts) == 1 {
		o = opts[0]
	}
	if foundation.DebugMode {
		if onPaint == nil {
			panic(errors.New(TagGPUViewInvalid, "gpu.NewView: the paint function is not nil", "field", "onPaint"))
		}
		if o.Format != TextureFormatUnspecified && o.Format != TextureFormatRGBA8Unorm && o.Format != TextureFormatRGBA16Float {
			panic(errors.New(TagGPUViewInvalid, "gpu.NewView: Format is RGBA8Unorm or RGBA16Float", "field", "Format", "value", int(o.Format)))
		}
		if o.DepthFormat != TextureFormatUnspecified && o.DepthFormat != TextureFormatDepth32Float && o.DepthFormat != TextureFormatDepth24PlusStencil8 {
			panic(errors.New(TagGPUViewInvalid, "gpu.NewView: DepthFormat is Depth32Float or Depth24PlusStencil8", "field", "DepthFormat", "value", int(o.DepthFormat)))
		}
		if o.SampleCount != 0 && o.SampleCount != 1 && o.SampleCount != 4 {
			panic(errors.New(TagGPUViewInvalid, "gpu.NewView: SampleCount is 1 or 4", "field", "SampleCount", "value", o.SampleCount))
		}
	}
	if o.Format != TextureFormatRGBA16Float {
		o.Format = TextureFormatRGBA8Unorm
	}
	if o.SampleCount != 4 {
		o.SampleCount = 1
	}
	return &View{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPaint: onPaint, o: o}
}

// CreateState creates the state.
//
// Flutter: no counterpart
func (w *View) CreateState() widgets.State { return &viewState{} }

// viewState follows the device of the app and owns the surface of the view.
type viewState struct {
	widgets.StateOf[*View]
	widgets.SingleTickerProvider
	s *viewSurface
	// dev is the device the view draws on, nil when it has none.
	dev *Device
	// inited says that OnInit ran for dev and OnDispose has not.
	inited bool
	// unavailable says that the app has no GPU device at all; lostOnce that the view had a device that was lost.
	unavailable, lostOnce bool
	cancelDevice          func()
	removeLost            func()
	ticker                *scheduler.Ticker
	ctrl                  *ViewController
}

// InitState takes the device, if there is one, and waits for the next otherwise.
func (s *viewState) InitState() {
	w := s.Widget()
	s.s = &viewSurface{cfg: viewConfig{format: w.o.Format, depth: w.o.DepthFormat, samples: w.o.SampleCount}, onPaint: w.onPaint}
	s.attachController()
	b := widgets.WidgetsBindingOf(s.Context())
	s.cancelDevice = OnDevice(b, s.onDevice)
	d, err := DeviceOfBinding(b)
	if err == nil {
		s.useDevice(d)
		return
	}
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == TagGPUUnavailable {
		s.unavailable = true
		return
	}
	// The renderer is chosen at the first frame: ask again when it is over.
	b.Scheduler().AddPostFrameCallback(func(time.Duration) { s.recheck() })
}

func (s *viewState) attachController() {
	c := s.Widget().o.Controller
	if s.ctrl != nil && s.ctrl != c {
		s.ctrl.s = nil
	}
	s.ctrl = c
	if c != nil {
		c.s = s.s
	}
}

// DidUpdateWidget takes the new configuration.
func (s *viewState) DidUpdateWidget(*View) {
	w := s.Widget()
	s.s.onPaint = w.onPaint
	s.s.configure(viewConfig{format: w.o.Format, depth: w.o.DepthFormat, samples: w.o.SampleCount})
	s.attachController()
	s.syncTicker()
}

// onDevice is called from the event loop with each new device.
func (s *viewState) onDevice(d *Device) {
	if !s.Mounted() || s.dev == d {
		return
	}
	s.SetState(func() { s.useDevice(d) })
}

// useDevice makes d the device of the view and calls OnInit.
func (s *viewState) useDevice(d *Device) {
	s.dev = d
	s.unavailable = false
	s.s.useDevice(d)
	s.removeLost = d.OnLost(func(error) { s.onLost(d) })
	if f := s.Widget().o.OnInit; f != nil {
		f(d)
	}
	s.inited = true
	s.syncTicker()
}

// onLost is called when the device was lost: the view's targets are gone, and the fallback shows until the next
// device.
func (s *viewState) onLost(d *Device) {
	if s.dev != d {
		return
	}
	s.dev = nil
	s.lostOnce = true
	s.removeLost = nil
	s.s.useDevice(nil)
	s.syncTicker()
	b := widgets.WidgetsBindingOf(s.Context())
	b.PlatformDispatcher().Post(func() {
		if !s.Mounted() {
			return
		}
		s.disposeApp()
		s.SetState(func() {})
	})
	b.Scheduler().AddPostFrameCallback(func(time.Duration) { s.recheck() })
}

// recheck asks the binding again whether the app has a GPU, once the renderer is chosen.
func (s *viewState) recheck() {
	if !s.Mounted() || s.dev != nil {
		return
	}
	_, err := DeviceOfBinding(widgets.WidgetsBindingOf(s.Context()))
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == TagGPUUnavailable && !s.unavailable {
		s.SetState(func() { s.unavailable = true })
	}
}

// disposeApp calls OnDispose for the OnInit that ran.
func (s *viewState) disposeApp() {
	if !s.inited {
		return
	}
	s.inited = false
	if f := s.Widget().o.OnDispose; f != nil {
		f()
	}
}

// syncTicker runs the ticker while the view is Continuous and has a device.
func (s *viewState) syncTicker() {
	want := s.Widget().o.Continuous && s.dev != nil
	if want && s.ticker == nil {
		s.ticker = s.CreateTicker(func(elapsed time.Duration) {
			s.s.time = elapsed
			s.s.markDirty()
		})
	}
	if s.ticker == nil {
		return
	}
	switch {
	case want && !s.ticker.IsActive():
		s.ticker.Start()
	case !want && s.ticker.IsActive():
		s.ticker.Stop()
	}
}

// Dispose stops the ticker and releases the targets, after OnDispose.
func (s *viewState) Dispose() {
	if s.cancelDevice != nil {
		s.cancelDevice()
	}
	if s.removeLost != nil {
		s.removeLost()
	}
	if s.ticker != nil {
		s.ticker.Dispose()
	}
	if s.ctrl != nil {
		s.ctrl.s = nil
	}
	s.disposeApp()
	s.s.close()
}

// Build shows the view, or the fallback when there is no device to draw on.
func (s *viewState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if s.dev == nil {
		if f := w.o.Fallback; f != nil && (s.unavailable || s.lostOnce) {
			return f
		}
		return widgets.NewSizedBoxExpand(nil)
	}
	dpr, ok := widgets.MediaQueryMaybeDevicePixelRatioOf(bc)
	if !ok || dpr <= 0 {
		dpr = 1
	}
	var child widgets.Widget = &viewRenderWidget{LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(), s: s.s, dpr: dpr}
	if w.o.SemanticsLabel != "" {
		child = widgets.NewSemantics(child, widgets.SemanticsOpts{Label: foundation.Some(w.o.SemanticsLabel), Image: foundation.Some(true)})
	}
	return child
}

// viewConfig is the part of ViewOpts that decides the targets.
type viewConfig struct {
	format, depth TextureFormat
	samples       int
}

// viewTargets are the textures of the view for one size.
type viewTargets struct {
	color, resolve, depth *Texture
	// shown is the texture the window shows, and id its number in the registry.
	shown *Texture
	id    int64
	w, h  int
	dpr   float64
}

// viewSurface is what the view draws into: the targets, the device and what is asked of them. The state owns it and
// the render box paints it.
type viewSurface struct {
	cfg     viewConfig
	onPaint func(f *ViewFrame)
	dev     *Device
	box     *renderView
	t       *viewTargets
	// retired are the targets of an earlier size, released after the frame.
	retired    []*Texture
	postQueued bool
	dirty      bool
	time       time.Duration
	report     foundation.ErrorReporter
	failedSize [2]int
}

func (s *viewSurface) markDirty() {
	s.dirty = true
	if s.box != nil && s.dev != nil {
		s.box.MarkNeedsPaint()
	}
}

// configure takes a new configuration; the targets are made again when it differs.
func (s *viewSurface) configure(cfg viewConfig) {
	if cfg == s.cfg {
		return
	}
	s.cfg = cfg
	s.dropTargets()
	s.markDirty()
}

// useDevice makes d the device (nil for none). The targets of the old device go.
func (s *viewSurface) useDevice(d *Device) {
	if s.dev == d {
		return
	}
	s.dropTargets()
	s.flushRetired()
	s.dev = d
	s.markDirty()
}

// dropTargets releases the targets now.
func (s *viewSurface) dropTargets() {
	if s.t == nil {
		return
	}
	s.release(s.t)
	s.t = nil
}

func (s *viewSurface) release(t *viewTargets) {
	for _, tex := range []*Texture{t.color, t.resolve, t.depth} {
		if tex != nil {
			tex.Release()
		}
	}
}

// close releases everything.
func (s *viewSurface) close() {
	s.dropTargets()
	s.flushRetired()
	s.dev = nil
	s.box = nil
}

// retire releases the targets after the frame: the frame that is being composited may still show them.
func (s *viewSurface) retire(t *viewTargets) {
	for _, tex := range []*Texture{t.color, t.resolve, t.depth} {
		if tex != nil {
			s.retired = append(s.retired, tex)
		}
	}
	if s.postQueued || s.dev == nil {
		return
	}
	s.postQueued = true
	s.dev.st.b.Scheduler().AddPostFrameCallback(func(time.Duration) {
		s.postQueued = false
		s.flushRetired()
	})
}

func (s *viewSurface) flushRetired() {
	for _, tex := range s.retired {
		tex.Release()
	}
	s.retired = nil
}

// pixelsOf is the number of device pixels of a length: rounded, and at least 1 for a length that is not empty.
func pixelsOf(length, dpr float64) int {
	if !(length > 0) || math.IsInf(length, 0) {
		return 0
	}
	return max(1, int(math.Round(length*dpr)))
}

// prepare makes the targets for the size of the box, and returns the id of the texture to show. It makes new
// targets when the size or the device pixel ratio changed, and the view draws again then.
func (s *viewSurface) prepare(size ui.Size, dpr float64) (id int64, ok bool) {
	if s.dev == nil || s.dev.lostErr != nil {
		return 0, false
	}
	w, h := pixelsOf(size.Width, dpr), pixelsOf(size.Height, dpr)
	if w == 0 || h == 0 {
		if s.t != nil {
			s.retire(s.t)
			s.t = nil
		}
		return 0, false
	}
	if s.t != nil && s.t.w == w && s.t.h == h && s.t.dpr == dpr {
		return s.t.id, true
	}
	if s.failedSize == [2]int{w, h} {
		return 0, false
	}
	t, err := s.makeTargets(w, h, dpr)
	if err != nil {
		s.failedSize = [2]int{w, h}
		if s.t != nil {
			s.retire(s.t)
			s.t = nil
		}
		if s.report != nil {
			s.report(err)
		}
		return 0, false
	}
	if s.t != nil {
		s.retire(s.t)
	}
	s.failedSize = [2]int{}
	s.t = t
	s.dirty = true
	return t.id, true
}

// makeTargets makes the textures of the view and registers the one that the window shows.
func (s *viewSurface) makeTargets(w, h int, dpr float64) (*viewTargets, error) {
	d, cfg := s.dev, s.cfg
	t := &viewTargets{w: w, h: h, dpr: dpr}
	fail := func(err error) (*viewTargets, error) {
		s.release(t)
		return nil, err
	}
	size := Extent3D{Width: w, Height: h}
	var err error
	if cfg.samples == 1 {
		t.color, err = d.NewTexture(TextureDesc{Label: "gpu.View color", Size: size, Format: cfg.format, Usage: TextureUsageRenderAttachment | TextureUsageTextureBinding})
		t.shown = t.color
	} else {
		t.color, err = d.NewTexture(TextureDesc{Label: "gpu.View color", Size: size, Format: cfg.format, SampleCount: cfg.samples, Usage: TextureUsageRenderAttachment})
		if err == nil {
			t.resolve, err = d.NewTexture(TextureDesc{Label: "gpu.View resolve", Size: size, Format: cfg.format, Usage: TextureUsageRenderAttachment | TextureUsageTextureBinding})
			t.shown = t.resolve
		}
	}
	if err != nil {
		return fail(err)
	}
	if cfg.depth != TextureFormatUnspecified {
		t.depth, err = d.NewTexture(TextureDesc{Label: "gpu.View depth", Size: size, Format: cfg.depth, SampleCount: cfg.samples, Usage: TextureUsageRenderAttachment})
		if err != nil {
			return fail(err)
		}
	}
	t.id = d.RegisterTexture(t.shown)
	return t, nil
}

// drawIfDirty runs the paint function for the targets with the id, if the view must draw. It is called while the
// scene is built, so that the window composites the result in the same frame.
func (s *viewSurface) drawIfDirty(id int64) {
	t := s.t
	if !s.dirty || t == nil || t.id != id || s.dev == nil || s.dev.lostErr != nil {
		return
	}
	s.dirty = false
	d := s.dev
	f := &ViewFrame{Device: d, Color: t.color, Resolve: t.resolve, Depth: t.depth, Width: t.w, Height: t.h, DevicePixelRatio: t.dpr, Time: s.time}
	if f.Time == 0 {
		f.Time = d.st.b.Scheduler().CurrentFrameTimeStamp()
	}
	var submitErr error
	panicked := foundation.Guard(s.report, foundation.PaintPanicked, "during the paint function of a gpu.View", nil, func() {
		f.Encoder = d.NewCommandEncoder()
		s.onPaint(f)
		submitErr = d.Queue().Submit(f.Encoder.Finish())
	})
	if panicked {
		return
	}
	if submitErr != nil && !IsDeviceLost(submitErr) {
		if s.report == nil {
			panic(submitErr)
		}
		s.report(submitErr)
	}
	d.textures().MarkFrameAvailable(id)
}

// viewRenderWidget is the widget of the render box of a view.
type viewRenderWidget struct {
	widgets.LeafRenderObjectWidget
	s   *viewSurface
	dpr float64
}

func (w *viewRenderWidget) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	b := &renderView{s: w.s, dpr: w.dpr}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (w *viewRenderWidget) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	b := ro.(*renderView)
	if b.s != w.s {
		b.s.detach(b)
		b.s = w.s
		if b.Attached() {
			b.s.attach(b)
		}
	}
	if b.dpr != w.dpr {
		b.dpr = w.dpr
		b.MarkNeedsPaint()
	}
}

// renderView is the box of a view: it takes as much room as it is given, and paints a layer that draws the view
// when the scene is built.
type renderView struct {
	rendering.RenderBox
	s   *viewSurface
	dpr float64
}

func (b *renderView) Attach(owner *rendering.PipelineOwner) {
	b.RenderBox.Attach(owner)
	b.s.attach(b)
}

func (b *renderView) Detach() {
	b.s.detach(b)
	b.RenderBox.Detach()
}

func (s *viewSurface) attach(b *renderView) {
	s.box = b
	s.report = b.DebugOptions().ReportError
}

func (s *viewSurface) detach(b *renderView) {
	if s.box == b {
		s.box = nil
	}
}

func (b *renderView) SizedByParent() bool          { return true }
func (b *renderView) AlwaysNeedsCompositing() bool { return true }
func (b *renderView) IsRepaintBoundary() bool      { return true }
func (b *renderView) HitTestSelf(ui.Offset) bool   { return true }
func (b *renderView) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Biggest()
}

// Paint makes the targets for the size and adds the layer that draws the view.
func (b *renderView) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	size := b.Size()
	b.s.report = b.DebugOptions().ReportError
	id, ok := b.s.prepare(size, b.dpr)
	if !ok {
		return
	}
	l := &viewLayer{rect: ui.NewRectFromLTWH(offset.Dx, offset.Dy, size.Width, size.Height), id: id, s: b.s}
	rendering.InitLayer(&l.LayerBase, l)
	context.AddLayer(l)
}

// viewLayer shows the texture of a view, and draws the view first when it must.
type viewLayer struct {
	rendering.LayerBase
	rect ui.Rect
	id   int64
	s    *viewSurface
}

// AddToScene runs the paint function if the view is marked, then adds the texture.
func (l *viewLayer) AddToScene(builder *ui.SceneBuilder) {
	l.s.drawIfDirty(l.id)
	builder.AddTexture(l.id, ui.AddTextureOpts{Offset: l.rect.TopLeft(), Width: l.rect.Width(), Height: l.rect.Height(), FilterQuality: ui.FilterQualityLow})
}

// FindAnnotations finds nothing.
func (l *viewLayer) FindAnnotations(*rendering.AnnotationResult, ui.Offset, bool) bool { return false }

var _ rendering.Layer = (*viewLayer)(nil)

// The tags of the errors of package gpu that the view makes.
const (
	// TagGPUViewInvalid is the tag of the programmer error of a NewView call with a configuration that cannot be
	// made: a nil paint function, a format that is not one of the view's, a sample count that is not 1 or 4, or
	// more than one ViewOpts. Debug builds panic with it; the metadata "field" names the option.
	//
	// Flutter: no counterpart
	TagGPUViewInvalid = "GPUViewInvalid"
)
