//go:build darwin && !ios

package macos

import (
	"image"
	"math"
	"sort"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/loop"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// defaultContentSize is the size of a window whose config has none, in points.
var defaultContentSize = ui.Size{Width: 800, Height: 600}

// maxContentSize stands for "no maximum" of the content size: AppKit's own default is the largest float.
const maxContentSize = 1e7

// window is one NSWindow with its GlutterView (SPEC §29.4). All methods are for the UI thread.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterViewController.mm FlutterViewController
type window struct {
	// drop is the drag that is over the window (dnd_darwin.go).
	drop   dropState
	p      *Platform
	id     ui.ViewID
	cfg    platform.WindowConfig
	style  windowStyle
	handle objcx.Handle

	ns, view, delegate, layer objc.ID
	// nsCursor is the cursor that the view shows over itself, 0 for the default.
	nsCursor objc.ID
	// devInside says that the dev events have made the mouse enter the window.
	devInside bool

	title string
	// minSize and maxSize are the content size constraints in points (0 is no limit): AppKit holds them for
	// the user's resizing only, so SetSize applies them itself.
	minSize, maxSize ui.Size
	// scale is the backing scale factor the metrics last reported.
	scale float64

	frames loop.FrameRequests
	pacer  loop.LinkPacer
	link   frameLink
	// onLinkFire is the function that a CADisplayLink's target method calls; cv is the CVDisplayLink, when
	// that is the link (displaylink_darwin.go).
	onLinkFire func(target float64)
	cv         *cvLink

	pool *darwin.SurfacePool

	// shown says the first show happened; hidden says SetHidden(true); closed says the window is gone.
	shown, hidden, closed bool
	// miniaturized, fullscreen and occluded are the states AppKit reported.
	miniaturized, fullscreen bool
	// activated is whether the window is the key window; focusKnown says it was reported once.
	activated, focusKnown bool
	// liveResizing is whether the user is dragging an edge.
	liveResizing bool
	// initial is the state the window starts in, applied at the first show.
	initial platform.WindowState
	// sleepAssertion is the IOPMAssertion that KeepScreenOn holds, 0 when none.
	sleepAssertion uint32
	// text is the text input of the window: the document that the input method reads (textinput.go).
	text textInput
	// frameInterval_ is the time between two ticks of the link, in seconds, as the last tick said.
	frameInterval_ float64
}

var (
	_ platform.Window = (*window)(nil)
	_ gpu.PixelTarget = (*window)(nil)
)

// NewWindow makes an NSWindow of the config with a view that is layer-backed (SPEC §29.4): sizes are in
// points, the style comes from the config (winstyle.go), the window is not shown until its first frame is
// drawn, and it is centered. It fails with MacOSWindowCreateFailed when AppKit makes no window.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterViewController.mm FlutterViewController
func (p *Platform) NewWindow(cfg platform.WindowConfig) (win platform.Window, err error) {
	w := &window{p: p, id: p.nextView, cfg: cfg, style: styleFor(cfg), title: cfg.Title, initial: cfg.State, scale: 1}
	size := cfg.Size
	if size.Width <= 0 || size.Height <= 0 {
		size = defaultContentSize
	}
	objcx.WithPool(func() { err = w.create(size) })
	if err != nil {
		return nil, err
	}
	p.nextView++
	p.windows[w.id] = w
	if w.style.Modal {
		if parent, ok := p.windows[cfg.Parent]; ok {
			p.openModal(w, parent)
		}
	}
	if p.handler != nil { // a window made while Run is going shows itself after the call returns
		p.Post(p.showNewWindows)
	}
	return w, nil
}

func (w *window) create(size ui.Size) error {
	s := w.p.s
	w.handle = objcx.NewHandle(w)
	rect := rectOf(0, 0, size.Width, size.Height)
	ns := objc.ID(s.window.Class).Send(sel("alloc")).Send(sel("initWithContentRect:styleMask:backing:defer:"),
		rect, w.style.mask(), backingStoreBuffered, false)
	if ns == 0 {
		objcx.DropHandle(w.handle)
		return errors.New(TagMacOSWindowCreateFailed, "NSWindow initWithContentRect returned nil", "width", size.Width, "height", size.Height)
	}
	objcx.SetHandle(ns, w.handle)
	w.ns = ns
	ns.Send(sel("setReleasedWhenClosed:"), false)
	ns.Send(sel("setAcceptsMouseMovedEvents:"), true)
	ns.Send(sel("setRestorable:"), false)
	behavior := collectionBehaviorFullScreenNone
	if w.style.FullScreen {
		behavior = collectionBehaviorFullScreenPrimary
	}
	ns.Send(sel("setCollectionBehavior:"), behavior)
	if w.style.FullSizeContent {
		ns.Send(sel("setTitlebarAppearsTransparent:"), true)
		ns.Send(sel("setTitleVisibility:"), titleHidden)
	}
	w.setTitleText(w.title)

	w.view = objc.ID(s.view.Class).Send(sel("alloc")).Send(sel("initWithFrame:"), rect)
	objcx.SetHandle(w.view, w.handle)
	w.view.Send(sel("setWantsLayer:"), true)
	w.view.Send(sel("setLayerContentsRedrawPolicy:"), viewLayerContentsRedrawNever)
	w.layer = send[objc.ID](w.view, "layer")
	w.layer.Send(sel("setOpaque:"), true)
	grav := objcx.NewString("topLeft")
	w.layer.Send(sel("setContentsGravity:"), grav)
	objcx.Release(grav)
	ns.Send(sel("setContentView:"), w.view)
	w.configureInput()

	w.delegate = s.windowDelegate.New(w.handle)
	ns.Send(sel("setDelegate:"), w.delegate)
	ns.Send(sel("setInitialFirstResponder:"), w.view)
	ns.Send(sel("makeFirstResponder:"), w.view)
	ns.Send(sel("center"))

	if cfg := w.cfg; cfg.MinSize.Width > 0 || cfg.MinSize.Height > 0 || cfg.MaxSize.Width > 0 || cfg.MaxSize.Height > 0 {
		w.applyConstraints(cfg.MinSize, cfg.MaxSize)
	}
	if w.cfg.HasParent {
		if parent, ok := w.p.windows[w.cfg.Parent]; ok && w.style.Dialog {
			parent.ns.Send(sel("addChildWindow:ordered:"), w.ns, int64(1))
		}
	}
	w.scale = send[float64](ns, "backingScaleFactor")
	w.layer.Send(sel("setContentsScale:"), w.scale)
	w.pool = darwin.NewSurfacePool(8, func(width, height int) (darwin.Surface, error) {
		surf, err := darwin.NewIOSurface(width, height)
		if err != nil {
			return nil, err
		}
		return surf, nil
	})
	link, err := w.newFrameLink(w.fireTick)
	if err != nil {
		w.p.log.Warn("the frames of the window cannot follow the display: they follow a timer", "err", err)
	} else {
		w.link = link
	}
	return nil
}

// frameInterval is the time between the frames of the display of the window, in seconds, for the target
// time of a tick that a link on its own thread cannot know.
func (w *window) frameInterval() float64 {
	if w.frameInterval_ > 0 {
		return w.frameInterval_
	}
	return 1.0 / 60
}

// ID is the view of the window.
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface is the handle of the window's layer: Window and Surface are the CALayer of the view (the
// CAMetalLayer when Metal draws, step 33.8).
func (w *window) NativeSurface() gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindMetalLayer, Window: uintptr(w.layer), Surface: uintptr(w.layer)}
}

// setTitleText sets the title; the empty title is the process name, which AppKit does not do itself.
func (w *window) setTitleText(title string) {
	s := objcx.NewString(title)
	w.ns.Send(sel("setTitle:"), s)
	objcx.Release(s)
}

// SetTitle sets the window title.
func (w *window) SetTitle(title string) {
	if w.closed {
		return
	}
	w.title = title
	objcx.WithPool(func() { w.setTitleText(title) })
}

// SetSize asks for a new content size in points: the window is resized now, which reports the new metrics
// and draws a frame at that size before it returns.
func (w *window) SetSize(size ui.Size) {
	if w.closed || size.Width <= 0 || size.Height <= 0 {
		return
	}
	size = clampSize(size, w.minSize, w.maxSize)
	objcx.WithPool(func() { w.ns.Send(sel("setContentSize:"), nsSize{Width: size.Width, Height: size.Height}) })
}

// clampSize limits a size to the constraints; 0 is no limit.
func clampSize(size, min, max ui.Size) ui.Size {
	if max.Width > 0 && size.Width > max.Width {
		size.Width = max.Width
	}
	if max.Height > 0 && size.Height > max.Height {
		size.Height = max.Height
	}
	if size.Width < min.Width {
		size.Width = min.Width
	}
	if size.Height < min.Height {
		size.Height = min.Height
	}
	return size
}

// SetSizeConstraints sets the smallest and the largest content size in points; zero means no limit.
func (w *window) SetSizeConstraints(min, max ui.Size) {
	if w.closed {
		return
	}
	objcx.WithPool(func() { w.applyConstraints(min, max) })
}

func (w *window) applyConstraints(min, max ui.Size) {
	w.minSize, w.maxSize = min, max
	if max.Width <= 0 {
		max.Width = maxContentSize
	}
	if max.Height <= 0 {
		max.Height = maxContentSize
	}
	w.ns.Send(sel("setContentMinSize:"), nsSize{Width: math.Max(0, min.Width), Height: math.Max(0, min.Height)})
	w.ns.Send(sel("setContentMaxSize:"), nsSize{Width: max.Width, Height: max.Height})
}

// SetState asks for the window to be zoomed, minimized, full screen or back to normal (SPEC §29.4): zoom is
// macOS's maximize, full screen is the native space.
func (w *window) SetState(state platform.WindowState) {
	if w.closed {
		return
	}
	if !w.shown {
		w.initial = state // applied at the first show
		return
	}
	objcx.WithPool(func() { w.applyState(state) })
}

func (w *window) applyState(state platform.WindowState) {
	zoomed := send[bool](w.ns, "isZoomed")
	switch state {
	case platform.WindowStateMaximized:
		w.leaveSpecial(false, true)
		if !send[bool](w.ns, "isZoomed") {
			w.ns.Send(sel("zoom:"), objc.ID(0))
		}
	case platform.WindowStateMinimized:
		if !send[bool](w.ns, "isMiniaturized") {
			w.ns.Send(sel("miniaturize:"), objc.ID(0))
		}
	case platform.WindowStateFullscreen:
		if !w.isFullScreen() && w.style.FullScreen {
			w.ns.Send(sel("toggleFullScreen:"), objc.ID(0))
		}
	default: // normal
		w.leaveSpecial(true, true)
		if zoomed && send[bool](w.ns, "isZoomed") {
			w.ns.Send(sel("zoom:"), objc.ID(0))
		}
	}
}

// leaveSpecial takes the window out of full screen and out of the Dock.
func (w *window) leaveSpecial(fullscreen, miniaturized bool) {
	if miniaturized && send[bool](w.ns, "isMiniaturized") {
		w.ns.Send(sel("deminiaturize:"), objc.ID(0))
	}
	if fullscreen && w.isFullScreen() {
		w.ns.Send(sel("toggleFullScreen:"), objc.ID(0))
	}
}

// maskFullScreen is NSWindowStyleMaskFullScreen, which AppKit sets in a window that is in a full-screen space.
const maskFullScreen uint64 = 1 << 14

func (w *window) isFullScreen() bool {
	return send[uint64](w.ns, "styleMask")&maskFullScreen != 0
}

// SetHidden takes the window off the screen (true) or puts it back and asks for the focus (false). A
// hidden window asks for no frame; one that was asked for is drawn once the window is shown.
func (w *window) SetHidden(hidden bool) bool {
	if w.closed {
		return true
	}
	w.hidden = hidden
	objcx.WithPool(func() {
		if hidden {
			w.ns.Send(sel("orderOut:"), objc.ID(0))
		} else if w.shown {
			w.orderFront()
		}
	})
	w.updateLink()
	w.p.updateLifecycle()
	return true
}

// orderFront puts the window on the screen: the key window of a regular app, and in front without the
// keys in an accessory one, which never takes the focus from the user's apps (SPEC §29.13).
func (w *window) orderFront() {
	if send[int64](w.p.app, "activationPolicy") == activationPolicyRegular {
		w.ns.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
	} else {
		w.ns.Send(sel("orderFront:"), objc.ID(0))
	}
}

// RequestFrame asks for OnFrame at the next tick of the display link (SPEC §29.3): requests merge, and the
// link runs only while a frame is wanted.
func (w *window) RequestFrame() {
	if w.closed {
		return
	}
	w.frames.Request()
	w.updateLink()
}

// canDraw says the window can show a frame now: it is shown, not hidden and not minimized. A window that
// nothing shows (covered, or on a display that sleeps) is not told apart here: CADisplayLink does not tick
// for it, which is the pause that macOS wants.
func (w *window) canDraw() bool {
	return w.shown && !w.closed && !w.hidden && !w.miniaturized
}

// updateLink starts or stops the display link as the pacer says.
func (w *window) updateLink() {
	if w.link == nil {
		return
	}
	start, stop := w.pacer.Update(w.frames.Wanted() && w.canDraw())
	switch {
	case start:
		w.link.setRunning(true)
	case stop:
		w.link.setRunning(false)
	}
}

// linkFired is the displayLinkFired: method of the view: the target time of the tick is read from the link.
func (w *window) linkFired(link objc.ID) {
	target := send[float64](link, "targetTimestamp")
	if d := target - send[float64](link, "timestamp"); d > 0 {
		w.frameInterval_ = d
	}
	w.fireTick(target)
}

// fireTick delivers the frame of a tick of the link, on the main thread: the window draws when a frame is
// wanted and it can, and the link stops when nothing is wanted any more.
func (w *window) fireTick(target float64) {
	w.p.wakes.Add(1)
	h := w.p.handler
	if h != nil && w.frames.Take(w.canDraw()) {
		h.OnFrame(w.id, w.p.frameTime(target))
		w.p.framesDrawn++
	}
	if !w.closed {
		w.updateLink()
	}
}

// frameTime turns a time of the media clock into the duration since the platform started.
func (p *Platform) frameTime(media float64) time.Duration {
	return time.Duration((media - p.base) * float64(time.Second))
}

// cvOffer is called on the thread of a CVDisplayLink.
func (w *window) cvOffer() {
	if c := w.cv; c != nil {
		c.offer()
	}
}

// displayID is the display that the window is on (CGDirectDisplayID).
func (w *window) displayID() (uint32, bool) {
	screen := send[objc.ID](w.ns, "screen")
	if screen == 0 {
		return 0, false
	}
	return uint32(screenID(screen)), true
}

// PerformWindowAction does what the title bar of the app asks (SPEC §29.4). A move is AppKit's own, started
// from the press that is down. A resize from a chosen edge and the window menu have no counterpart on macOS
// (the edges of the window resize it, and there is no window menu), so they do nothing. The maximize and the
// restore that come from a double click on the bar do what the system setting "Double-click a window's title
// bar to" says.
func (w *window) PerformWindowAction(action ui.WindowAction) {
	if w.closed {
		return
	}
	if k := action.Kind; k == ui.WindowActionMaximize || k == ui.WindowActionRestore {
		var ok bool
		if action.Kind, ok = w.titleBarAction(k); !ok {
			return
		}
	}
	switch action.Kind {
	case ui.WindowActionMove:
		if !w.blocked() {
			w.startMove()
		}
	case ui.WindowActionActivate:
		if !w.blocked() {
			w.activate()
		}
	case ui.WindowActionMinimize:
		w.SetState(platform.WindowStateMinimized)
	case ui.WindowActionMaximize:
		w.SetState(platform.WindowStateMaximized)
	case ui.WindowActionRestore:
		w.SetState(platform.WindowStateNormal)
	case ui.WindowActionFullscreen:
		w.SetState(platform.WindowStateFullscreen)
	case ui.WindowActionUnfullscreen:
		w.SetState(platform.WindowStateNormal)
	case ui.WindowActionClose:
		objcx.WithPool(func() { w.ns.Send(sel("performClose:"), objc.ID(0)) })
	}
}

// SetIcon ignores the icon: a macOS window has none (the icon of the app is the Dock's).
func (w *window) SetIcon(ui.WindowIcon) error { return nil }

// SetContentType ignores the hint.
func (w *window) SetContentType(ui.ContentType) {}

// kIOPMAssertionLevelOn is the level of an assertion that is on.
const pmAssertionLevelOn = 255

// KeepScreenOn holds an IOPMAssertion that prevents the display from sleeping from idleness (on is true)
// or releases it (SPEC §29.4). It fails with IdleInhibitUnavailable when the system refuses.
func (w *window) KeepScreenOn(on bool) error {
	s := w.p.s
	if !on {
		if w.sleepAssertion != 0 {
			s.pmRelease(w.sleepAssertion)
			w.sleepAssertion = 0
		}
		return nil
	}
	if w.sleepAssertion != 0 {
		return nil
	}
	kind := cfString("PreventUserIdleDisplaySleep")
	name := cfString("Glutter window keeps the screen on")
	defer releaseCF(kind)
	defer releaseCF(name)
	var id uint32
	if rc := s.pmCreate(kind, pmAssertionLevelOn, name, &id); rc != 0 {
		return errors.New(TagIdleInhibitUnavailable, "IOPMAssertionCreateWithName refused the assertion", "ioreturn", int(rc))
	}
	w.sleepAssertion = id
	return nil
}

// Close destroys the window. The platform forgets it, and the loop ends when it was the last.
func (w *window) Close() {
	if w.closed {
		return
	}
	objcx.WithPool(func() {
		w.ns.Send(sel("close"))
	})
	w.finishClose() // when close already told windowWillClose this does nothing
}

// shouldClose is windowShouldClose: the user's close button asks the handler.
func (w *window) shouldClose() bool {
	if w.blocked() { // a window with a modal window over it cannot be closed by the user
		w.redirectToModal()
		return false
	}
	if w.p.handler == nil {
		return true
	}
	return w.p.handler.OnCloseRequest(w.id)
}

// willClose is windowWillClose: the window is going away.
func (w *window) willClose() { w.finishClose() }

func (w *window) finishClose() {
	if w.closed {
		return
	}
	w.p.ptr.cancelView(w.id, w.p.frameTime(w.p.s.mediaTime()))
	w.p.flushPointers()
	w.closed = true
	if w.sleepAssertion != 0 {
		w.p.s.pmRelease(w.sleepAssertion)
		w.sleepAssertion = 0
	}
	if w.link != nil {
		w.link.close()
		w.link = nil
	}
	w.cv = nil
	w.pool.Close()
	delete(w.p.windows, w.id)
	w.p.closedModal(w)
	w.p.updateLifecycle()
	w.ns.Send(sel("setDelegate:"), objc.ID(0))
	if w.cfg.HasParent {
		if parent, ok := w.p.windows[w.cfg.Parent]; ok {
			parent.ns.Send(sel("removeChildWindow:"), w.ns)
		}
	}
	if len(w.p.windows) == 0 && w.p.handler != nil && w.p.quitOnLastClose {
		w.p.Post(w.p.stopLoop)
	}
	objcx.Release(w.delegate)
	objcx.Release(w.view)
	objcx.Release(w.ns)
	objcx.DropHandle(w.handle)
	w.delegate, w.view, w.ns = 0, 0, 0
}

// info is the WindowInfo for the engine.
func (w *window) info() ui.WindowInfo {
	return ui.WindowInfo{
		Decoration:   w.style.decoration(),
		Capabilities: w.style.capabilities(),
		Title:        w.title,
		Maximized:    send[bool](w.ns, "isZoomed") && !w.fullscreen,
		Fullscreen:   w.fullscreen,
		Activated:    w.activated,
		Minimized:    w.miniaturized,
	}
}

// reportMetrics tells the handler the size and the scale of the window.
func (w *window) reportMetrics() {
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	bounds := send[nsRect](w.view, "bounds")
	w.scale = send[float64](w.ns, "backingScaleFactor")
	px := physicalSize(ui.Size{Width: bounds.Size.Width, Height: bounds.Size.Height}, w.scale)
	m := platform.ViewMetrics{PhysicalSize: px, DevicePixelRatio: w.scale, Window: w.info()}
	if w.style.FullSizeContent {
		// The title bar is the part of the window that the content layout rectangle leaves out.
		frame := send[nsRect](w.ns, "frame")
		layout := send[nsRect](w.ns, "contentLayoutRect")
		m.ViewPadding.Top = titleBarPadding(boxRect{W: frame.Size.Width, H: frame.Size.Height}, boxRect{W: layout.Size.Width, H: layout.Size.Height}, w.scale)
	}
	if id, ok := w.displayID(); ok {
		m.Display = foundation.Some(int(id))
	}
	h.OnMetrics(w.id, m)
}

// renderNow asks the handler for a frame at once: a resize draws at the new size before AppKit goes on
// (SPEC §29.3). A frame that was wanted is drawn by it.
func (w *window) renderNow() {
	h := w.p.handler
	if h == nil || w.closed || !w.shown || w.hidden {
		return
	}
	w.frames.Clear()
	h.OnFrame(w.id, w.p.frameTime(w.p.s.mediaTime()))
	w.p.framesDrawn++
	w.updateLink()
}

// didResize is windowDidResize: the metrics and a frame at the new size, before it returns.
func (w *window) didResize() {
	if !w.shown {
		return
	}
	w.reportMetrics()
	w.renderNow()
}

// didChangeBacking is windowDidChangeBackingProperties: another display with another scale.
func (w *window) didChangeBacking() {
	w.scale = send[float64](w.ns, "backingScaleFactor")
	w.layer.Send(sel("setContentsScale:"), w.scale)
	if w.cv != nil {
		w.cv.screenChanged()
	}
	if !w.shown {
		return
	}
	w.reportMetrics()
	w.renderNow()
}

// didChangeScreen is windowDidChangeScreen.
func (w *window) didChangeScreen() {
	if w.cv != nil {
		w.cv.screenChanged()
	}
	if w.shown {
		w.reportMetrics()
	}
}

// keyChanged is the window becoming or no longer being the key window: the focus of the view and of the keyboard.
func (w *window) keyChanged(key bool) {
	if key && w.blocked() { // the keys of a window with a modal window over it go to that one
		w.redirectToModal()
		return
	}
	w.activated = key
	if !key && w.p.kb != nil {
		// The keys that are down go up in another window or app: the framework is told now, and does not
		// keep them pressed.
		w.p.kb.resp.releaseAll(w.p.frameTime(w.p.s.mediaTime()))
	}
	w.reportFocus(key)
	if w.shown {
		w.reportMetrics() // WindowInfo.Activated
	}
}

// reportFocus tells the handler that the view got or lost the focus. On macOS the keyboard goes to the
// key window, so the keyboard focus is the same.
func (w *window) reportFocus(focused bool) {
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	w.focusKnown = true
	h.OnViewFocus(w.id, focused)
	if kh, ok := h.(platform.KeyboardFocusHandler); ok {
		kh.OnKeyboardFocus(w.id, focused)
	}
	w.p.updateLifecycle()
}

// stateChanged is a change of the minimized, zoomed or full-screen state.
func (w *window) stateChanged() {
	w.miniaturized = send[bool](w.ns, "isMiniaturized")
	w.fullscreen = w.isFullScreen()
	if !w.shown {
		return
	}
	w.p.updateLifecycle()
	w.reportMetrics()
	w.updateLink()
	if w.canDraw() {
		w.renderNow()
	}
}

// liveResize is the start and the end of the user dragging an edge.
func (w *window) liveResize(start bool) { w.liveResizing = start }

// firstShow reports the metrics, draws the first frame into the hidden window and shows it in the state
// it was asked to start in, so that the window never shows at its normal size first (SPEC §29.4).
func (w *window) firstShow() {
	p := w.p
	if p.handler == nil || w.shown || w.closed || w.hidden {
		return
	}
	w.shown = true
	w.miniaturized, w.fullscreen = false, false
	w.reportMetrics()
	w.renderNow()
	if w.closed {
		return
	}
	w.orderFront()
	w.applyState(w.initial)
	if !w.focusKnown && !w.closed {
		w.activated = send[bool](w.ns, "isKeyWindow")
		w.reportFocus(w.activated)
	}
	p.updateLifecycle()
	p.startDevEvents()
}

// showNewWindows gives each window that was made and not shown its first frame and shows it.
func (p *Platform) showNewWindows() {
	for _, w := range p.sortedWindows() {
		w.firstShow()
	}
}

// sortedWindows are the windows in the order they were made.
func (p *Platform) sortedWindows() []*window {
	out := make([]*window, 0, len(p.windows))
	for _, w := range p.windows {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// startDevEvents starts the script of GLUTTER_DEV_EVENTS once, after the first window is shown.
func (p *Platform) startDevEvents() {
	if p.devStarted || len(p.devEvents) == 0 {
		return
	}
	p.devStarted = true
	if p.focusForDevKeys() {
		p.waitForFrameAfterFocus(time.Now().Add(devKeyFocusLimit), p.framesDrawn)
		return
	}
	p.runDevEvents(0)
}

// waitForFrameAfterFocus starts the script once a frame was drawn after the window got the focus, and not
// before devKeyFocusDelay has passed: the framework moves its focus into the window's route in the frame after
// the focus event, and keys that came before would be lost. It starts anyway at the limit, for an app that
// draws no frame then. A fixed delay alone lost the keys of a script when the machine was busy.
func (p *Platform) waitForFrameAfterFocus(limit time.Time, framesBefore uint64) {
	start := time.Now().Add(devKeyFocusDelay)
	var poll func()
	poll = func() {
		if p.framesDrawn > framesBefore || !time.Now().Before(limit) {
			p.runDevEvents(0)
			return
		}
		p.addTimer(time.Now().Add(devKeyFocusPoll), poll)
	}
	p.addTimer(start, poll)
}

// ScriptPending says that a script of GLUTTER_DEV_EVENTS was given and has not run its last step yet
// (platform.ScriptWaiter): the engine's self-capture waits for it.
func (p *Platform) ScriptPending() bool { return len(p.devEvents) > 0 && !p.devDone }

// runDevEvents runs the steps from index i; a wait arms a timer for the rest.
func (p *Platform) runDevEvents(i int) {
	for ; i < len(p.devEvents); i++ {
		ev := p.devEvents[i]
		if p.runDevPointer(ev) || p.runDevKey(ev) || p.runDevText(ev) {
			continue
		}
		if ev.Kind == "wait" {
			next := i + 1
			p.addTimer(time.Now().Add(ev.Wait), func() { p.runDevEvents(next) })
			return
		}
	}
	p.devDone = true
}

// AcquirePixels gives a buffer of the size that nothing reads now (gpu.PixelTarget, SPEC §29.5): an
// IOSurface of the window's pool, locked for the CPU.
func (w *window) AcquirePixels(width, height int) (gpu.PixelBuffer, error) {
	b, err := w.pool.Acquire(width, height)
	if err != nil {
		return gpu.PixelBuffer{}, err
	}
	return gpu.PixelBuffer{Pix: b.Pix, Stride: b.Stride, Width: b.Width, Height: b.Height, Format: gpu.PixelFormatBGRX8888, Age: b.Age}, nil
}

// PresentPixels shows the buffer that AcquirePixels gave out as the contents of the view's layer.
func (w *window) PresentPixels([]image.Rectangle) error {
	surf, ok := w.pool.Present()
	if !ok {
		return nil // nothing changed: the window shows what it showed
	}
	io, ok := surf.(*darwin.IOSurface)
	if !ok {
		return nil
	}
	objcx.WithPool(func() {
		tx := class("CATransaction")
		tx.Send(sel("begin"))
		tx.Send(sel("setDisableActions:"), true)
		w.layer.Send(sel("setContents:"), objc.ID(io.Ref()))
		tx.Send(sel("commit"))
	})
	return nil
}
