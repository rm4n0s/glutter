package offscreen

import (
	"context"
	"image"
	"math"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// defaultMaxFrameRate is ViewOpts.MaxFrameRate when it is not set.
const defaultMaxFrameRate = 30

// Frame is a frame of a View that changed some pixels.
//
// Flutter: no counterpart
type Frame struct {
	// Image is the whole view, in premultiplied RGBA. It is the view's own and is valid
	// only during the call of the callback: the next frame draws into it, on the same
	// goroutine, as soon as the callback returns. A callback that sends the pixels
	// elsewhere copies them (the rectangles of Damage are what changed since the last
	// frame the callback saw).
	Image *image.RGBA
	// Damage are the rectangles, in pixels, that this frame redrew. They are not empty,
	// and the callback may keep the slice. The first frame, and the first after SetSize
	// changed the size in pixels, damage the whole image.
	Damage []image.Rectangle
	// Number is the number of the scene the framework made for the view, counting from 1.
	// It skips the scenes that a newer one replaced while the goroutine was drawing, and
	// those that changed no pixel, so a gap tells that frames were dropped.
	Number int
}

// ViewOpts are the optional parameters of NewView.
//
// Flutter: no counterpart
type ViewOpts struct {
	// DevicePixelRatio is the number of pixels of the image for each logical pixel; the
	// default is 1.
	DevicePixelRatio float64
	// MaxFrameRate is the most frames the view makes in a second, counting the time from
	// the start of one to the start of the next; the default is 30.
	MaxFrameRate int
	// Clipboard is the clipboard of the view. When it is nil, the view uses the clipboard of
	// the app (the parent's), as before: copy, cut and paste in the view reach the desktop.
	// When it is set, every clipboard call of the view's tree (copy, cut, paste, the check
	// that decides whether Paste is offered, the primary selection that a selection fills and
	// a middle click pastes, rich content) goes to it, and none reaches the app's clipboard or
	// the desktop's.
	//
	// SetClipboardText and GetClipboardText go to the host. If it also implements
	// ui.RichClipboardHost, SetClipboard and GetClipboard go to it, for every kind, the
	// primary selection included. If it does not, a rich call for the clipboard kind and the
	// plain text format goes to its text methods (a content with a text sets that text, and
	// its other formats are dropped), and every other rich call (the primary selection, an
	// image, a list of URIs) does nothing: it completes with an empty content and no error.
	//
	// The host's callbacks (the done functions it is given) run on the UI thread of the
	// parent: the host calls them there, or posts them with the parent binding's Post, and
	// never before the method that took them has returned. A done function that the host
	// calls after Close does nothing: the view drops it.
	//
	// Flutter: no counterpart
	Clipboard ui.ClipboardHost
}

// View runs a widget tree in a view of its own, on a widgets binding of its own that
// shares the UI thread of the app's binding, and hands the pixels of each frame to a
// callback on a goroutine of its own (SPEC §9.9, D28).
//
// The framework runs on the UI thread of the parent binding: it builds, lays out and
// paints the tree there, and the scene of each frame is detached (Detacher) and given to
// the goroutine of the view, which draws it with a Surface and calls the callback with
// the pixels and what changed. A newer scene replaces one that waits, so a slow callback
// makes the view drop frames, never queue them. The view asks for a frame at most
// MaxFrameRate times a second, whatever the app's windows do: its frame rate does not
// depend on them.
//
// Flutter: no counterpart
type View struct {
	parent  *widgets.WidgetsBinding
	binding *widgets.WidgetsBinding
	d       *ui.PlatformDispatcher
	uiView  *ui.View
	cancel  context.CancelFunc

	onFrame  func(Frame)
	report   foundation.ErrorReporter
	clock    fakeclock.Clock
	start    time.Time
	minGap   time.Duration
	dpr      float64
	cfg      ui.ViewConfiguration
	detacher *Detacher
	// clip is ViewOpts.Clipboard, nil for the app's clipboard; rich is clip as a
	// ui.RichClipboardHost, nil when it is not one.
	clip ui.ClipboardHost
	rich ui.RichClipboardHost
	// surface is the view's pixels; the raster goroutine uses it, and Close after that
	// goroutine ended.
	surface *Surface

	wake chan struct{}
	quit chan struct{}
	done chan struct{}

	mu   sync.Mutex
	idle *sync.Cond
	// closed is set by Close; nothing is scheduled, posted or kept after it.
	closed bool
	// scheduled is whether a frame is asked for and has not begun: the timer runs, or the
	// post waits.
	scheduled bool
	timer     fakeclock.Timer
	lastBegin time.Time
	haveLast  bool
	// pending is the scene that waits for the raster goroutine, and number counts the
	// scenes the framework made. busy is whether the goroutine is drawing one.
	pending *viewFrame
	number  int
	busy    bool
}

// viewFrame is a detached scene and the size of the view it is for.
type viewFrame struct {
	scene         *ui.Scene
	width, height int
	number        int
}

// gestureClock gives the gesture binding of the view (the timers of the recognizers and of
// the blinking caret) the clock that paces the frames: the system clock, except in tests.
type gestureClock struct{ c fakeclock.Clock }

func (g gestureClock) Now() time.Time { return g.c.Now() }
func (g gestureClock) AfterFunc(d time.Duration, f func()) gestures.SamplingTimer {
	return g.c.AfterFunc(d, f)
}

// viewHost is the ui.PlatformHost of a View: the view's own dispatcher asks it for
// frames and gives it the scenes. It is also the clipboard: ViewOpts.Clipboard, or the
// parent's when that is nil (SPEC §9.9). It has no
// semantics, text input, cursor, window or drag host, so the dispatcher ignores those
// calls of the framework.
type viewHost struct{ v *View }

var (
	_ ui.PlatformHost      = viewHost{}
	_ ui.ClipboardHost     = viewHost{}
	_ ui.RichClipboardHost = viewHost{}
)

// NewView runs the widget tree root in a view of size logical pixels, on a widgets binding
// of its own, and calls onFrame with the pixels of each frame that changed some. Call it
// on the UI thread of parent (the binding of the app); every other method of the view is
// called there too.
//
//   - The view's dispatcher has one view, which root fills, wrapped in a widgets.View as
//     RunWidget's WrapWithDefaultView does, so MediaQuery and the rest work. It shares the
//     text engine of the parent's dispatcher (fonts the app loaded work), and the binding
//     shares the parent's UI thread, error reporter, ErrorWidgetBuilder, debug flags,
//     asset bundle and HTTP client. The platform configuration (locale, brightness, text
//     scale) is the default one; change it with Binding().PlatformDispatcher().
//   - onFrame runs on the goroutine of the view, one call at a time. A panic in it is
//     recovered and reported to the parent's error reporter (CallbackPanicked), and the
//     view goes on. It must not call Close or wait for the UI thread: Close waits for it.
//   - A nil parent, root or onFrame, and a size that is not positive, are errors.
//
// Flutter: no counterpart
func NewView(parent *widgets.WidgetsBinding, size ui.Size, root widgets.Widget, onFrame func(Frame), o ...ViewOpts) (*View, error) {
	return newView(parent, size, root, onFrame, fakeclock.Real{}, opts.First(o))
}

// newView is NewView with the clock that paces the frames (tests give a fake one).
func newView(parent *widgets.WidgetsBinding, size ui.Size, root widgets.Widget, onFrame func(Frame), clock fakeclock.Clock, o ViewOpts) (*View, error) {
	if parent == nil {
		return nil, errors.New(tagViewParentNil, "NewView needs the widgets binding of the app")
	}
	if root == nil {
		return nil, errors.New(tagViewRootNil, "NewView needs a root widget")
	}
	if onFrame == nil {
		return nil, errors.New(tagViewCallbackNil, "NewView needs a function to call with the frames")
	}
	if o.DevicePixelRatio < 0 || math.IsNaN(o.DevicePixelRatio) || math.IsInf(o.DevicePixelRatio, 0) || o.MaxFrameRate < 0 {
		return nil, errors.New(tagViewOptionInvalid, "the device pixel ratio and the frame rate of a view are not negative",
			"devicePixelRatio", o.DevicePixelRatio, "maxFrameRate", o.MaxFrameRate)
	}
	parent.UIThread().AssertCurrent("offscreen.NewView")
	dpr := o.DevicePixelRatio
	if dpr == 0 {
		dpr = 1
	}
	rate := o.MaxFrameRate
	if rate == 0 {
		rate = defaultMaxFrameRate
	}
	cfg := ui.ViewConfiguration{PhysicalSize: ui.Size{Width: size.Width * dpr, Height: size.Height * dpr}, DevicePixelRatio: dpr}
	if !validPhysicalSize(cfg.PhysicalSize) {
		return nil, errors.New(tagViewSizeInvalid, "the size of a view is positive and fits in a surface",
			"width", size.Width, "height", size.Height, "devicePixelRatio", dpr)
	}

	v := &View{
		parent: parent, onFrame: onFrame, report: parent.ErrorReporter(), clock: clock, start: clock.Now(),
		minGap: time.Second / time.Duration(rate), dpr: dpr, cfg: cfg, detacher: NewDetacher(),
		wake: make(chan struct{}, 1), quit: make(chan struct{}), done: make(chan struct{}),
	}
	v.idle = sync.NewCond(&v.mu)
	v.clip = o.Clipboard
	v.rich, _ = o.Clipboard.(ui.RichClipboardHost)
	v.d = ui.NewPlatformDispatcher(viewHost{v})
	v.d.SetTextEngine(parent.PlatformDispatcher().TextEngine())
	v.uiView = v.d.AddView(ui.ImplicitViewID, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	v.cancel = cancel
	// The painting binding of the view keeps an image cache of its own, and reads assets
	// and the network as the parent does.
	pp := parent.Painting()
	v.binding = widgets.NewWidgetsBinding(ctx, v.d, widgets.WidgetsBindingOpts{
		Debug:              parent.DebugOptions(),
		ErrorReporter:      parent.ErrorReporter(),
		ErrorWidgetBuilder: parent.BuildOwner().ErrorWidgetBuilder,
		UIThread:           parent.UIThread(),
		Clock:              gestureClock{clock},
		Painting:           painting.PaintingBindingOpts{AssetBundle: pp.AssetBundle(), HTTPClient: pp.HTTPClient()},
	})
	go v.rasterLoop()
	v.binding.AttachRootWidget(v.binding.WrapWithDefaultView(root))
	return v, nil
}

// validPhysicalSize is whether a size in pixels, rounded up, fits in a Surface.
func validPhysicalSize(s ui.Size) bool {
	w, h := math.Ceil(s.Width), math.Ceil(s.Height)
	return w >= 1 && h >= 1 && w <= maxSurfaceSize && h <= maxSurfaceSize
}

// Binding is the widgets binding of the view. The app sends input through its
// dispatcher, on the UI thread: DispatchPointerDataPacket (positions in pixels of the
// image, ViewID ui.ImplicitViewID), DispatchKeyData and DispatchTextInput; it can
// change the platform configuration with UpdatePlatformConfiguration, and reach the
// tree with WidgetsBindingOf from a widget of the view.
//
// Flutter: no counterpart
func (v *View) Binding() *widgets.WidgetsBinding { return v.binding }

// SetSize changes the size of the view, in logical pixels. The framework hears of it as a
// change of the view's metrics and lays out again; the next frame has the new size in
// pixels and damages all of it. It fails for a size that is not positive (tag
// ViewSizeInvalid) and for a view that was closed (tag ViewClosed).
//
// Flutter: no counterpart
func (v *View) SetSize(size ui.Size) error {
	if v.isClosed() {
		return errors.New(tagViewClosed, "the view was closed")
	}
	v.parent.UIThread().AssertCurrent("offscreen.View.SetSize")
	physical := ui.Size{Width: size.Width * v.dpr, Height: size.Height * v.dpr}
	if !validPhysicalSize(physical) {
		return errors.New(tagViewSizeInvalid, "the size of a view is positive and fits in a surface",
			"width", size.Width, "height", size.Height, "devicePixelRatio", v.dpr)
	}
	if physical == v.cfg.PhysicalSize {
		return nil
	}
	v.cfg.PhysicalSize = physical
	v.d.UpdateViewConfiguration(ui.ImplicitViewID, v.cfg)
	return nil
}

// Close ends the view: the widget tree is unmounted (every State is disposed), the
// frame timer stops, the goroutine of the view ends and Close waits for it, including a
// callback that is running. After Close nothing is posted to the UI thread, no frame is
// made and the callback is not called; what was posted before is dropped when it runs.
// Calling Close again does nothing. Call it on the UI thread, and never from the
// callback.
//
// Flutter: no counterpart
func (v *View) Close() {
	v.parent.UIThread().AssertCurrent("offscreen.View.Close")
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return
	}
	v.closed = true
	v.scheduled = false
	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
	v.pending = nil
	v.mu.Unlock()
	v.cancel()

	// Unmount the tree with a frame of its own, as the tester does: nothing is rendered
	// (the view is closed) and no frame is asked for.
	v.binding.AttachRootWidget(widgets.NewViewCollection(nil, widgets.ViewCollectionOpts{Key: foundation.NewUniqueKey()}))
	v.binding.Scheduler().ScheduleFrame()
	v.d.BeginFrame(v.clock.Now().Sub(v.start))
	v.d.DrawFrame()

	close(v.quit)
	<-v.done
	v.surface = nil
}

// ScheduleFrame asks for a frame, at most MaxFrameRate times a second: a frame that is
// asked for while another waits is the same frame.
func (h viewHost) ScheduleFrame() {
	v := h.v
	v.mu.Lock()
	if v.closed || v.scheduled {
		v.mu.Unlock()
		return
	}
	v.scheduled = true
	var delay time.Duration
	if v.haveLast {
		delay = max(0, v.minGap-v.clock.Now().Sub(v.lastBegin))
	}
	if delay > 0 {
		v.timer = v.clock.AfterFunc(delay, v.fire)
		v.mu.Unlock()
		return
	}
	v.mu.Unlock()
	v.parent.Post(v.runFrame)
}

// fire is the pacing timer: it posts the frame to the UI thread, unless the view closed.
func (v *View) fire() {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return
	}
	v.timer = nil
	v.mu.Unlock()
	v.parent.Post(v.runFrame)
}

// runFrame is the frame, on the UI thread: the begin and the draw phase of the
// framework. It does nothing for a view that closed, or when no frame is asked for.
func (v *View) runFrame() {
	v.mu.Lock()
	if v.closed || !v.scheduled {
		v.mu.Unlock()
		return
	}
	v.scheduled = false
	now := v.clock.Now()
	v.lastBegin, v.haveLast = now, true
	v.mu.Unlock()
	v.d.BeginFrame(now.Sub(v.start))
	v.d.DrawFrame()
}

// RenderView detaches the scene and gives it to the raster goroutine, replacing the one
// that waits.
func (h viewHost) RenderView(_ ui.ViewID, scene *ui.Scene) error {
	v := h.v
	if v.isClosed() || scene == nil {
		return nil
	}
	physical := v.uiView.PhysicalSize()
	f := &viewFrame{scene: v.detacher.Detach(scene), width: int(math.Ceil(physical.Width)), height: int(math.Ceil(physical.Height))}
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return nil
	}
	v.number++
	f.number = v.number
	v.pending = f
	v.mu.Unlock()
	select {
	case v.wake <- struct{}{}:
	default:
	}
	return nil
}

// SetClipboardText puts the text on the clipboard of the view (ViewOpts.Clipboard), or of
// the app when the view has none: a text field in it copies and pastes through the
// parent's engine then.
func (h viewHost) SetClipboardText(ctx context.Context, text string, done func(err error)) {
	if h.v.clip != nil {
		h.v.clip.SetClipboardText(ctx, text, h.guardErr(done))
		return
	}
	h.v.parent.PlatformDispatcher().SetClipboardText(ctx, text, h.guardErr(done))
}

// GetClipboardText reads the text of the clipboard of the view, or of the app.
func (h viewHost) GetClipboardText(ctx context.Context, done func(text string, ok bool, err error)) {
	guarded := func(text string, ok bool, err error) {
		if !h.v.isClosed() {
			done(text, ok, err)
		}
	}
	if h.v.clip != nil {
		h.v.clip.GetClipboardText(ctx, guarded)
		return
	}
	h.v.parent.PlatformDispatcher().GetClipboardText(ctx, guarded)
}

// SetClipboard puts the content on the selection of the view's clipboard, or of the app's
// when the view has none. A view clipboard that is not a ui.RichClipboardHost takes the
// text of a content for the clipboard kind, and nothing else.
func (h viewHost) SetClipboard(ctx context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(err error)) {
	v := h.v
	switch {
	case v.rich != nil:
		v.rich.SetClipboard(ctx, kind, content, h.guardErr(done))
	case v.clip != nil:
		if kind != ui.ClipboardKindPrimary && content.HasText {
			v.clip.SetClipboardText(ctx, content.Text, h.guardErr(done))
			return
		}
		if done != nil {
			h.Post(func() { done(nil) })
		}
	default:
		v.parent.PlatformDispatcher().SetClipboard(ctx, kind, content, h.guardErr(done))
	}
}

// GetClipboard reads the selection of the view's clipboard, or of the app's when the view
// has none. A view clipboard that is not a ui.RichClipboardHost answers the clipboard kind
// in the text format from its text methods, and every other call with an empty content.
func (h viewHost) GetClipboard(ctx context.Context, kind ui.ClipboardKind, format string, done func(content ui.ClipboardContent, err error)) {
	v := h.v
	guarded := func(content ui.ClipboardContent, err error) {
		if !v.isClosed() {
			done(content, err)
		}
	}
	switch {
	case v.rich != nil:
		v.rich.GetClipboard(ctx, kind, format, guarded)
	case v.clip != nil:
		if kind != ui.ClipboardKindPrimary && format == ui.ClipboardFormatText {
			v.clip.GetClipboardText(ctx, func(text string, ok bool, err error) {
				guarded(ui.ClipboardContent{Text: text, HasText: ok}, err)
			})
			return
		}
		h.Post(func() { guarded(ui.ClipboardContent{}, nil) })
	default:
		v.parent.PlatformDispatcher().GetClipboard(ctx, kind, format, guarded)
	}
}

// guardErr makes a result function that does nothing once the view closed; nil stays nil.
func (h viewHost) guardErr(done func(err error)) func(err error) {
	if done == nil {
		return nil
	}
	return func(err error) {
		if !h.v.isClosed() {
			done(err)
		}
	}
}

// isClosed is whether Close began.
func (v *View) isClosed() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.closed
}

// RequestAppExit does nothing: the app that made the view decides when it ends.
func (viewHost) RequestAppExit() {}

// Post runs the function on the UI thread of the parent, unless the view closed by then.
func (h viewHost) Post(fn func()) {
	v := h.v
	if v.isClosed() {
		return
	}
	v.parent.Post(func() {
		if !v.isClosed() {
			fn()
		}
	})
}

// rasterLoop is the goroutine of the view: it draws the scene that waits, and gives
// the pixels to the callback, until the view closes.
func (v *View) rasterLoop() {
	defer close(v.done)
	for {
		select {
		case <-v.wake:
		case <-v.quit:
			return
		}
		for {
			v.mu.Lock()
			f := v.pending
			v.pending = nil
			if f == nil || v.closed {
				v.busy = false
				v.idle.Broadcast()
				v.mu.Unlock()
				break
			}
			v.busy = true
			v.mu.Unlock()
			v.draw(f)
		}
	}
}

// draw draws the frame into the surface and calls the callback with what changed.
func (v *View) draw(f *viewFrame) {
	switch {
	case v.surface == nil:
		v.surface = NewSurface(f.width, f.height)
	case v.surface.Width() != f.width || v.surface.Height() != f.height:
		v.surface.Resize(f.width, f.height)
	}
	damage, err := v.surface.Draw(f.scene)
	if err != nil {
		v.reportToParent(err)
		return
	}
	if len(damage) == 0 {
		return
	}
	frame := Frame{Image: v.surface.Image(), Damage: append([]image.Rectangle(nil), damage...), Number: f.number}
	if v.isClosed() {
		return
	}
	foundation.Guard(v.reportToParent, foundation.CallbackPanicked, "while calling the frame callback of an offscreen view", nil, func() {
		v.onFrame(frame)
	})
}

// reportToParent gives the error to the parent's error reporter, on the UI thread.
func (v *View) reportToParent(err error) {
	v.parent.Post(func() { v.report(err) })
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagViewParentNil     = "ViewParentNil"
	tagViewRootNil       = "ViewRootNil"
	tagViewCallbackNil   = "ViewCallbackNil"
	tagViewOptionInvalid = "ViewOptionInvalid"
	tagViewSizeInvalid   = "ViewSizeInvalid"
	tagViewClosed        = "ViewClosed"
)
