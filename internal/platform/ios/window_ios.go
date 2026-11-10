//go:build ios

package ios

import (
	"image"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/loop"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// window is one UIWindow of the scene with its view controller and its GlutterView (SPEC §30.4). The view, its layer and its display
// link are made with the window, so that the native surface is stable from the first call; the UIWindow is made when the scene has
// connected. All methods are for the UI thread.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm FlutterViewController
type window struct {
	unsupportedWindow
	p      *Platform
	id     ui.ViewID
	cfg    platform.WindowConfig
	handle objcx.Handle

	// uiw is the UIWindow, 0 until a scene has connected; scene is the scene it is in.
	uiw, scene objc.ID
	vc, view   objc.ID
	layer      objc.ID
	link       objc.ID

	title string

	frames loop.FrameRequests
	pacer  loop.LinkPacer
	pool   *darwin.SurfacePool

	// shown says the first show happened; hidden says SetHidden(true); closed says the window is gone; active says the window's scene
	// is the active one.
	shown, hidden, closed, active bool
	// reported says metrics were given to the handler, and last is what they were.
	reported bool
	last     platform.ViewMetrics

	// The pacing of the frames: the display link ticks, or a timer does when the link never has (a Simulator that no screen is
	// attached to: SPEC §30.3). linkTicked says the link ticked once; useTimer says the frames follow the timer; frameTimer and
	// watchdog are the timers of the platform's heap that are armed, 0 for none.
	linkTicked, useTimer bool
	timerOn              bool
	frameTimer, watchdog loop.TimerID
}

var (
	_ platform.Window = (*window)(nil)
	_ gpu.PixelTarget = (*window)(nil)
)

// NewWindow makes the view of a window with its view controller, layer and display link (SPEC §30.4). The window is on the screen
// after the scene has connected and its first frame is drawn. The config's size, constraints, state and decorations mean nothing on
// iOS: the window is the scene's. It makes a window while the app runs as well: the UIWindow is made at once then.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm FlutterViewController
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	w := &window{p: p, id: p.nextView, cfg: cfg, title: cfg.Title}
	objcx.WithPool(func() { w.create() })
	p.nextView++
	p.windows[w.id] = w
	if p.scene != 0 {
		objcx.WithPool(func() { w.attach(p.scene) })
		if p.handler != nil {
			p.Post(p.showNewWindows)
		}
	}
	return w, nil
}

// create makes the view, the view controller, the pool and the display link.
func (w *window) create() {
	s := w.p.s
	w.handle = objcx.NewHandle(w)
	w.view = s.view.New(w.handle)
	w.layer = send[objc.ID](w.view, "layer")
	w.layer.Send(sel("setOpaque:"), true)
	grav := objcx.NewString("topLeft")
	w.layer.Send(sel("setContentsGravity:"), grav)
	objcx.Release(grav)
	w.layer.Send(sel("setContentsScale:"), w.screenScale())
	w.vc = class("UIViewController").Send(sel("alloc")).Send(sel("init"))
	w.vc.Send(sel("setView:"), w.view)
	w.pool = darwin.NewSurfacePool(8, func(width, height int) (darwin.Surface, error) {
		surf, err := darwin.NewIOSurface(width, height)
		if err != nil {
			return nil, err
		}
		return surf, nil
	})
}

// attach makes the UIWindow of the scene and gives it the view controller; it is made once.
func (w *window) attach(scene objc.ID) {
	if w.uiw != 0 || w.closed {
		return
	}
	w.scene = scene
	w.uiw = class("UIWindow").Send(sel("alloc")).Send(sel("initWithWindowScene:"), scene)
	w.uiw.Send(sel("setRootViewController:"), w.vc)
	w.active = w.p.scenes[scene] == activationForegroundActive
	// The display link of the view: it ticks on the main thread while it runs, and follows the screen's refresh rate by itself. It is
	// made here, not with the view: a link made before UIKit has its screen (before the scene connects) never ticks.
	l := class("CADisplayLink").Send(sel("displayLinkWithTarget:selector:"), w.view, sel("displayLinkFired:"))
	objcx.Retain(l)
	l.Send(sel("setPaused:"), true)
	mode := objcx.NewString("kCFRunLoopCommonModes")
	l.Send(sel("addToRunLoop:forMode:"), class("NSRunLoop").Send(sel("mainRunLoop")), mode)
	objcx.Release(mode)
	w.link = l
	w.pacer = loop.LinkPacer{}
}

// detach drops the UIWindow of a scene that went away: the window waits for the next scene.
func (w *window) detach() {
	if w.uiw == 0 {
		return
	}
	w.uiw.Send(sel("setHidden:"), true)
	objcx.Release(w.uiw)
	w.uiw, w.scene = 0, 0
	w.shown = false
	if w.link != 0 {
		w.link.Send(sel("invalidate"))
		objcx.Release(w.link)
		w.link = 0
	}
}

// screenScale is the scale of the screen of the window, or of the main screen before the window has one.
func (w *window) screenScale() float64 {
	if w.uiw != 0 {
		if screen := send[objc.ID](w.uiw, "screen"); screen != 0 {
			return send[float64](screen, "scale")
		}
	}
	return send[float64](class("UIScreen").Send(sel("mainScreen")), "scale")
}

// ID is the view of the window.
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface is the handle of the window's layer (the CAMetalLayer when Metal draws, step 35.3).
func (w *window) NativeSurface() gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindMetalLayer, Window: uintptr(w.layer), Surface: uintptr(w.layer)}
}

// SetTitle keeps the title, which an iOS window does not show.
func (w *window) SetTitle(title string) { w.title = title }

// SetSize ignores the size: the window is the scene's, which the system sizes.
func (w *window) SetSize(ui.Size) {}

// SetSizeConstraints ignores the constraints.
func (w *window) SetSizeConstraints(_, _ ui.Size) {}

// SetState ignores the state: an iOS window is always full.
func (w *window) SetState(platform.WindowState) {}

// SetHidden takes the window off the screen (true) or puts it back. A hidden window asks for no frame; one that was asked for is
// drawn once the window is shown.
func (w *window) SetHidden(hidden bool) bool {
	if w.closed {
		return true
	}
	w.hidden = hidden
	if w.uiw != 0 && w.shown {
		objcx.WithPool(func() { w.uiw.Send(sel("setHidden:"), hidden) })
	}
	w.updateLink()
	return true
}

// SetCursor ignores the cursor: there is none until the pointer of iPadOS (step 35.4).
func (w *window) SetCursor(platform.MouseCursor) {}

// SetTextInput ignores the text input until step 35.5.
func (w *window) SetTextInput(ui.TextInputState) {}

// PerformWindowAction does nothing: an iOS window has no title bar.
func (w *window) PerformWindowAction(ui.WindowAction) {}

// SetIcon ignores the icon: the icon of an iOS app is the bundle's.
func (w *window) SetIcon(ui.WindowIcon) error { return nil }

// SetContentType ignores the hint.
func (w *window) SetContentType(ui.ContentType) {}

// KeepScreenOn keeps the screen from dimming and locking from idleness (UIApplication idleTimerDisabled) or lets it (SPEC §30.4).
func (w *window) KeepScreenOn(on bool) error {
	objcx.WithPool(func() { class("UIApplication").Send(sel("sharedApplication")).Send(sel("setIdleTimerDisabled:"), on) })
	return nil
}

// RequestFrame asks for OnFrame at the next tick of the display link: requests merge, and the link runs only while a frame is
// wanted.
func (w *window) RequestFrame() {
	if w.closed {
		return
	}
	w.frames.Request()
	w.updateLink()
}

// canDraw says the window can show a frame now: it is shown, not hidden, and its scene is on the screen.
func (w *window) canDraw() bool {
	return w.shown && !w.closed && !w.hidden && w.p.foreground()
}

// updateLink starts or stops the pacing as the pacer says: the display link, or the timer that stands in for it.
func (w *window) updateLink() {
	if w.link == 0 {
		return
	}
	start, stop := w.pacer.Update(w.frames.Wanted() && w.canDraw())
	switch {
	case start:
		w.startPacing()
	case stop:
		w.stopPacing()
	}
}

// pacingEnv is the environment variable that picks the pacing, for the tests: "timer" paces the frames by a timer from the start,
// "ca" by the display link only; anything else (or nothing) is the display link, with the timer taking over when the link does not
// tick (a Simulator that was booted with no screen: its display links never fire).
const pacingEnv = "GLUTTER_DISPLAYLINK"

// frameInterval is the time between two frames of the timer: the refresh rate of the main screen.
func (w *window) frameInterval() time.Duration {
	rate := send[int64](class("UIScreen").Send(sel("mainScreen")), "maximumFramesPerSecond")
	if rate <= 0 {
		rate = 60
	}
	return time.Second / time.Duration(rate)
}

// startPacing runs the display link, or the timer when the frames follow it.
func (w *window) startPacing() {
	mode := w.p.getenv(pacingEnv)
	if mode == "timer" {
		w.useTimer = true
	}
	if w.useTimer {
		w.timerOn = true
		w.armFrameTimer()
		return
	}
	w.link.Send(sel("setPaused:"), false)
	if !w.linkTicked && mode != "ca" {
		w.watchdog = w.p.addTimer(time.Now().Add(linkWatchdog), w.linkWatch)
	}
}

// stopPacing stops the link and the timers.
func (w *window) stopPacing() {
	w.link.Send(sel("setPaused:"), true)
	w.timerOn = false
	if w.frameTimer != 0 {
		w.p.timers.Cancel(w.frameTimer)
		w.frameTimer = 0
		w.p.rearm()
	}
	if w.watchdog != 0 {
		w.p.timers.Cancel(w.watchdog)
		w.watchdog = 0
		w.p.rearm()
	}
}

// linkWatchdog is how long the display link may run without a tick before the frames follow a timer instead.
const linkWatchdog = 300 * time.Millisecond

// linkWatch is the watchdog: a link that was started and has not ticked is not going to, so the frames follow the timer.
func (w *window) linkWatch() {
	w.watchdog = 0
	if w.closed || w.linkTicked || !w.pacer.Running() {
		return
	}
	w.p.log.Info("the display link does not tick: the frames follow a timer")
	w.useTimer = true
	w.link.Send(sel("setPaused:"), true)
	w.timerOn = true
	w.armFrameTimer()
}

// armFrameTimer arms the timer for the next frame.
func (w *window) armFrameTimer() {
	w.frameTimer = w.p.addTimer(time.Now().Add(w.frameInterval()), w.timerTick)
}

// timerTick is a frame of the timer: it delivers it like a tick of the link and goes on while a frame is wanted.
func (w *window) timerTick() {
	w.frameTimer = 0
	if w.closed || !w.timerOn {
		return
	}
	w.fireTick(w.p.s.mediaTime())
	if !w.closed && w.timerOn && w.frames.Wanted() && w.canDraw() {
		w.armFrameTimer()
	}
}

// linkFired is the displayLinkFired: method of the view: the target time of the tick is read from the link.
func (w *window) linkFired(link objc.ID) {
	w.linkTicked = true
	w.fireTick(send[float64](link, "targetTimestamp"))
}

// fireTick delivers the frame of a tick of the link, on the main thread: the window draws when a frame is wanted and it can, and
// the link stops when nothing is wanted any more.
func (w *window) fireTick(target float64) {
	w.p.wakes.Add(1)
	if h := w.p.handler; h != nil && w.frames.Take(w.canDraw()) {
		h.OnFrame(w.id, w.p.frameTime(target))
		w.p.framesDrawn++
	}
	if !w.closed {
		w.updateLink()
	}
}

// renderNow asks the handler for a frame at once: a rotation draws at the new size before UIKit goes on, so that no frame of the
// animation shows the old layout. A frame that was wanted is drawn by it.
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

// metrics reads the size, the scale and the safe area of the view.
func (w *window) metrics() platform.ViewMetrics {
	bounds := send[darwin.CGRect](w.view, "bounds")
	scale := w.screenScale()
	insets := send[darwin.NSEdgeInsets](w.view, "safeAreaInsets")
	return platform.ViewMetrics{
		PhysicalSize:     physicalSize(ui.Size{Width: bounds.Size.Width, Height: bounds.Size.Height}, scale),
		DevicePixelRatio: scale,
		ViewPadding:      viewPadding(edgeInsets{Top: insets.Top, Left: insets.Left, Bottom: insets.Bottom, Right: insets.Right}, scale),
		Display:          foundation.Some(0),
		Window:           ui.WindowInfo{Title: w.title, Fullscreen: true, Activated: w.active},
	}
}

// reportMetrics tells the handler the metrics when they changed, or always when force is set.
func (w *window) reportMetrics(force bool) bool {
	h := w.p.handler
	if h == nil || w.closed || w.view == 0 {
		return false
	}
	m := w.metrics()
	if !force && w.reported && m == w.last {
		return false
	}
	w.reported, w.last = true, m
	if w.layer != 0 {
		w.layer.Send(sel("setContentsScale:"), m.DevicePixelRatio)
	}
	h.OnMetrics(w.id, m)
	return true
}

// layout is layoutSubviews: a rotation, a change of the size of the scene or of the safe area gives new metrics and a frame at the
// new size before UIKit goes on.
func (w *window) layout() {
	if !w.shown {
		return
	}
	if w.reportMetrics(false) {
		w.renderNow()
	}
}

// activationChanged is the window's scene becoming active or not: the focus of the view, the frames and the metrics (the window's
// Activated flag) follow.
func (w *window) activationChanged(active bool) {
	w.active = active
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	if w.shown {
		w.reportMetrics(true)
		h.OnViewFocus(w.id, active)
		if kh, ok := h.(platform.KeyboardFocusHandler); ok {
			kh.OnKeyboardFocus(w.id, active)
		}
	}
	w.updateLink()
}

// firstShow puts the window on the screen, reports the metrics and draws the first frame (SPEC §30.4).
func (w *window) firstShow() {
	p := w.p
	if p.handler == nil || w.shown || w.closed || w.hidden || w.uiw == 0 {
		return
	}
	objcx.WithPool(func() {
		w.uiw.Send(sel("makeKeyAndVisible"))
		w.view.Send(sel("layoutIfNeeded"))
	})
	w.shown = true
	w.reportMetrics(true)
	w.renderNow()
	if w.closed {
		return
	}
	if h := p.handler; w.active {
		h.OnViewFocus(w.id, true)
		if kh, ok := h.(platform.KeyboardFocusHandler); ok {
			kh.OnKeyboardFocus(w.id, true)
		}
	}
	p.updateLifecycle()
}

// Close takes the window off the screen and releases it. The app goes on: an iOS app does not end with its last window.
func (w *window) Close() {
	if w.closed {
		return
	}
	w.closed = true
	if w.link != 0 {
		w.link.Send(sel("invalidate"))
		objcx.Release(w.link)
		w.link = 0
	}
	objcx.WithPool(func() {
		if w.uiw != 0 {
			w.uiw.Send(sel("setHidden:"), true)
			w.uiw.Send(sel("setRootViewController:"), objc.ID(0))
			objcx.Release(w.uiw)
			w.uiw = 0
		}
	})
	w.pool.Close()
	delete(w.p.windows, w.id)
	objcx.Release(w.vc)
	objcx.Release(w.view)
	objcx.DropHandle(w.handle)
	w.vc, w.view, w.layer = 0, 0, 0
	w.p.updateLifecycle()
}

// AcquirePixels gives a buffer of the size that nothing reads now (gpu.PixelTarget, SPEC §30.5): an IOSurface of the window's pool,
// locked for the CPU.
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
