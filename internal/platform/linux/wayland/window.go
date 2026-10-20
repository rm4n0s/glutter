package wayland

import (
	"fmt"
	"github.com/rm4n0s/glutter/foundation"
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/decor"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// fillColor is the color of the shared-memory window: #6750A4, the Material 3
// baseline primary. It stands in for rendering until step 1.6.
const fillColor uint32 = 0xFF6750A4

// defaultSize is the size, in logical pixels, of a window whose config does not
// ask for one and whose compositor does not suggest one.
var defaultSize = ui.Size{Width: 800, Height: 600}

// WindowStates holds the xdg_toplevel states the compositor last reported.
//
// Flutter: no counterpart
type WindowStates struct {
	// Maximized is whether the window fills the work area.
	Maximized bool
	// Fullscreen is whether the window fills the output.
	Fullscreen bool
	// Resizing is whether the user is resizing the window right now.
	Resizing bool
	// Activated is whether the window has keyboard focus.
	Activated bool
	// TiledLeft, TiledRight, TiledTop and TiledBottom say which edges touch
	// another window or the screen edge.
	TiledLeft, TiledRight, TiledTop, TiledBottom bool
	// Suspended is whether the compositor hides the window, so it should not
	// draw.
	Suspended bool
}

// decodeStates reads the states of an xdg_toplevel.configure event.
func decodeStates(states []uint32) WindowStates {
	var s WindowStates
	for _, v := range states {
		switch wl.XdgToplevelState(v) {
		case wl.XdgToplevelStateMaximized:
			s.Maximized = true
		case wl.XdgToplevelStateFullscreen:
			s.Fullscreen = true
		case wl.XdgToplevelStateResizing:
			s.Resizing = true
		case wl.XdgToplevelStateActivated:
			s.Activated = true
		case wl.XdgToplevelStateTiledLeft:
			s.TiledLeft = true
		case wl.XdgToplevelStateTiledRight:
			s.TiledRight = true
		case wl.XdgToplevelStateTiledTop:
			s.TiledTop = true
		case wl.XdgToplevelStateTiledBottom:
			s.TiledBottom = true
		case wl.XdgToplevelStateSuspended:
			s.Suspended = true
		}
	}
	return s
}

// window is one xdg_toplevel window, drawn with a shared-memory buffer.
type window struct {
	p   *Platform
	id  ui.ViewID
	cfg platform.WindowConfig

	surface    *wl.WlSurface
	xdgSurface *wl.XdgSurface
	toplevel   *wl.XdgToplevel
	decoration *wl.ZxdgToplevelDecorationV1
	// dialog marks the toplevel as a modal dialog (xdg_dialog_v1); nil when the
	// compositor has no xdg_wm_dialog_v1 or the window is not a modal dialog.
	dialog *wl.XdgDialogV1

	// frame is the libdecor frame of a window that libdecor decorates. Such a
	// window has no xdgSurface and no toplevel of its own: libdecor made them.
	// conf is the configure that the frame has not been told the answer to.
	frame *decor.Frame
	conf  decor.Configuration
	// deco is who draws the title bar; caps are what the user may do with the
	// window (xdg_toplevel.wm_capabilities, all of it until the compositor says).
	deco ui.DecorationMode
	caps ui.WindowCapabilities

	// width and height are the logical size of the content.
	width, height int32

	// The last toplevel.configure, applied by the xdg_surface.configure that
	// follows it.
	pendingWidth, pendingHeight int32
	pendingStates               WindowStates

	states         WindowStates
	decorationMode wl.ZxdgToplevelDecorationV1Mode
	configures     int  // how many configure cycles were acknowledged
	mapped         bool // a buffer was committed after the first configure
	closed         bool

	buffer *shmBuffer   // on display
	old    []*shmBuffer // replaced, waiting for the compositor to release them

	// The EGL path: the wl_egl_window that the renderer draws into. Zero in
	// shared-memory mode.
	egl       *waylandEGL
	eglWindow uintptr

	// Frame pacing (SPEC §8.11): at most one frame callback is in flight, and a
	// request that arrives meanwhile waits for it. A suspended window renders
	// nothing, so a frame callback may never come; frameGen numbers the callbacks
	// so that one that comes late, after the window resumed, is ignored.
	frameInFlight  bool
	frameRequested bool
	frameGen       int

	// Scale (SPEC §8.11, "Resizing and scaling"): the compositor's wishes, and what
	// the surface has been told. scale120 is wp_fractional_scale_v1's preferred
	// scale in 120ths and bufScale wl_surface's preferred_buffer_scale; each is 0
	// until the compositor sends one. entered are the outputs the surface is on.
	fractional *wl.WpFractionalScaleV1
	viewport   *wl.WpViewport
	scale120   int32
	bufScale   int32
	entered    []*wl.WlOutput
	// display is the display that the last metrics named; hasDisplay says it did.
	display      int
	hasDisplay   bool
	scale        int32 // the scale in 120ths that the buffer was sized for; 120 before the first configure
	bufferScale  int32 // the last wl_surface.set_buffer_scale
	viewportUsed bool  // wp_viewport.set_destination is in force
	physW, physH int32 // the size of the buffer in pixels

	// title is the current title of the window.
	title string

	// inhibitor is the idle inhibitor of KeepScreenOn, nil when the screen may go
	// idle. contentType is the wp_content_type_v1 of the surface, made on the first
	// hint. iconBuffers are the pictures of the icon the compositor was given.
	inhibitor   *wl.ZwpIdleInhibitorV1
	contentType *wl.WpContentTypeV1
	iconBuffers []*shmBuffer

	// cursor is the cursor over the window, set with SetCursor.
	cursor platform.MouseCursor

	// trace, when set (tests), is told "ack WxH" when a configure is acknowledged,
	// with the size of the buffer that goes with it, and "commit WxH" when a buffer
	// is committed.
	trace func(event string)
}

var _ platform.Window = (*window)(nil)

// NewWindow creates a toplevel window and shows it. The window is mapped once
// the compositor has answered the first commit with a configure, which happens
// inside Run. The title and the app id are set before that first commit, so the
// compositor matches the window to the app's .desktop file.
//
// Flutter: no counterpart
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	s := p.sess
	var egl *waylandEGL
	if !p.opts.ShmFill {
		var err error
		if egl, err = loadWaylandEGL(); err != nil {
			return nil, err
		}
	}
	w := &window{p: p, id: p.nextView, cfg: cfg, egl: egl}
	p.nextView++

	w.width, w.height = int32(cfg.Size.Width), int32(cfg.Size.Height)
	if w.width <= 0 || w.height <= 0 {
		w.width, w.height = int32(defaultSize.Width), int32(defaultSize.Height)
	}

	w.scale, w.bufferScale = 120, 1
	w.physW, w.physH = w.width, w.height
	w.surface = s.compositor.CreateSurface()
	w.surface.SetEvents(&wl.WlSurfaceEvents{
		OnEnter:                w.onEnter,
		OnLeave:                w.onLeave,
		OnPreferredBufferScale: func(factor int32) { w.bufScale = factor; w.scaleChanged() },
	})
	if s.viewporter != nil {
		w.viewport = s.viewporter.GetViewport(w.surface)
	}
	if s.fractionalScale != nil && w.viewport != nil {
		w.fractional = s.fractionalScale.GetFractionalScale(w.surface)
		w.fractional.SetEvents(&wl.WpFractionalScaleV1Events{
			OnPreferredScale: func(scale uint32) { w.scale120 = int32(scale); w.scaleChanged() },
		})
	}
	if egl != nil {
		w.eglWindow = egl.create(w.surface.Ptr(), w.width, w.height)
		if w.eglWindow == 0 {
			w.destroy()
			return nil, errors.New("EGLWindowFailed", "wl_egl_window_create failed", "width", w.width, "height", w.height)
		}
	}
	w.caps = ui.AllWindowCapabilities
	w.SetContentType(cfg.ContentType)
	w.title = cfg.Title
	w.deco = p.chooseDecoration(cfg)
	slog.Default().Debug("window decorations", "mode", w.deco.Name())
	if w.deco == ui.DecorationLibdecor {
		if err := w.decorateWithLibdecor(cfg); err != nil {
			w.destroy()
			return nil, err
		}
	} else {
		w.xdgSurface = s.wmBase.GetXdgSurface(w.surface)
		w.toplevel = w.xdgSurface.GetToplevel()
		w.xdgSurface.SetEvents(&wl.XdgSurfaceEvents{OnConfigure: w.onSurfaceConfigure})
		w.toplevel.SetEvents(&wl.XdgToplevelEvents{
			OnConfigure:      w.onToplevelConfigure,
			OnClose:          w.onClose,
			OnWmCapabilities: w.onWmCapabilities,
		})

		w.toplevel.SetTitle(cfg.Title)
		if cfg.AppID != "" {
			w.toplevel.SetAppId(cfg.AppID)
		}
		w.toplevel.SetMinSize(int32(cfg.MinSize.Width), int32(cfg.MinSize.Height))
		w.toplevel.SetMaxSize(int32(cfg.MaxSize.Width), int32(cfg.MaxSize.Height))
		if cfg.HasParent {
			if parent, ok := p.windows[cfg.Parent]; ok && parent.toplevel != nil {
				w.toplevel.SetParent(parent.toplevel)
			}
		}
		if !cfg.IsResizable() && cfg.Size.Width > 0 && cfg.Size.Height > 0 {
			// xdg-shell has no flag for it: a window whose smallest and largest sizes
			// are the same is not resizable.
			w.toplevel.SetMinSize(w.width, w.height)
			w.toplevel.SetMaxSize(w.width, w.height)
		}
		w.makeModal(w.toplevel)
		w.requestState(cfg.State)
		w.requestDecorations()
		if err := w.SetIcon(cfg.Icon); err != nil {
			w.destroy()
			return nil, err
		}

		// The initial commit, without a buffer, asks the compositor for the first
		// configure.
		w.surface.Commit()
	}

	p.windows[w.id] = w
	p.hadWindow = true
	return w, nil
}

// makeModal tells the compositor that the window is a modal dialog of its parent
// (xdg_dialog_v1.set_modal), which is what a dialog with a parent is. Without
// xdg_wm_dialog_v1 the dialog is only a transient window of its parent.
func (w *window) makeModal(toplevel *wl.XdgToplevel) {
	mgr := w.p.sess.dialogManager
	if mgr == nil || w.cfg.Kind != platform.WindowKindDialog || !w.cfg.HasParent || !w.cfg.Modal {
		return
	}
	w.dialog = mgr.GetXdgDialog(toplevel)
	w.dialog.SetModal()
}

// SetSizeConstraints sets the smallest and the largest size of the window; the
// compositor applies them with the next commit, so a frame is asked for.
//
// Flutter: widgets/_window.dart RegularWindowController.setConstraints
func (w *window) SetSizeConstraints(min, max ui.Size) {
	if w.closed {
		return
	}
	w.cfg.MinSize, w.cfg.MaxSize = min, max
	if w.frame != nil {
		w.frame.SetMinContentSize(int(min.Width), int(min.Height))
		w.frame.SetMaxContentSize(int(max.Width), int(max.Height))
	} else if w.toplevel != nil {
		w.toplevel.SetMinSize(int32(min.Width), int32(min.Height))
		w.toplevel.SetMaxSize(int32(max.Width), int32(max.Height))
	}
	w.RequestFrame()
}

// requestDecorations asks for server-side decorations, or for none, when the
// compositor has the decoration protocol (SPEC §8.2).
func (w *window) requestDecorations() {
	mgr := w.p.sess.decorationManager
	if mgr == nil {
		return
	}
	w.decoration = mgr.GetToplevelDecoration(w.toplevel)
	w.decoration.SetEvents(&wl.ZxdgToplevelDecorationV1Events{
		OnConfigure: w.onDecorationMode,
	})
	mode := wl.ZxdgToplevelDecorationV1ModeServerSide
	if w.deco != ui.DecorationServerSide {
		mode = wl.ZxdgToplevelDecorationV1ModeClientSide
	}
	w.decoration.SetMode(mode)
}

// requestState asks the compositor for a window state.
func (w *window) requestState(state platform.WindowState) {
	if w.frame != nil {
		switch state {
		case platform.WindowStateMaximized:
			w.frame.SetMaximized()
		case platform.WindowStateMinimized:
			w.frame.SetMinimized()
		case platform.WindowStateFullscreen:
			w.frame.SetFullscreen(0)
		case platform.WindowStateNormal:
			w.frame.UnsetMaximized()
			w.frame.UnsetFullscreen()
		}
		return
	}
	switch state {
	case platform.WindowStateMaximized:
		w.toplevel.SetMaximized()
	case platform.WindowStateMinimized:
		w.toplevel.SetMinimized()
	case platform.WindowStateFullscreen:
		w.toplevel.SetFullscreen(nil)
	case platform.WindowStateNormal:
		w.toplevel.UnsetMaximized()
		w.toplevel.UnsetFullscreen()
	}
}

// onToplevelConfigure stores the configuration; the xdg_surface.configure that
// ends the batch applies it.
func (w *window) onToplevelConfigure(width, height int32, states []uint32) {
	w.pendingWidth, w.pendingHeight = width, height
	w.pendingStates = decodeStates(states)
}

// onSurfaceConfigure applies the stored configuration: it draws at the new
// size, acknowledges the configure, and commits the buffer.
func (w *window) onSurfaceConfigure(serial uint32) {
	if w.closed {
		return
	}
	resized := w.pendingWidth > 0 && w.pendingHeight > 0 &&
		(w.pendingWidth != w.width || w.pendingHeight != w.height)
	if resized {
		w.width, w.height = w.pendingWidth, w.pendingHeight
	}
	wasSuspended := w.states.Suspended
	infoBefore := w.info()
	w.states = w.pendingStates
	infoChanged := w.info() != infoBefore
	if wasSuspended && !w.states.Suspended {
		// The frame callback in flight, if any, will not come now; the window
		// draws at once, and its time is the time of that frame.
		w.frameInFlight = false
		w.frameGen++
		w.frameRequested = true
	}

	if w.egl != nil {
		// The renderer draws into the wl_egl_window, and its swap is the commit.
		// So resize the native window, acknowledge, and render the new size at
		// once, without waiting for a frame callback: the first commit after the
		// ack already has the new size (SPEC §8.11, "Resizing and scaling").
		if resized || w.configures == 0 {
			w.resizeBuffer()
		}
		w.ackConfigure(serial)
		first := w.configures == 0
		w.configures++
		if first {
			w.p.useStartupToken(w)
		}
		if resized || first || infoChanged {
			w.reportMetrics()
		}
		w.renderNow()
		return
	}

	// Draw first, then acknowledge and commit, so no frame of the old size is
	// shown after the ack (SPEC §8.11, "Resizing and scaling").
	w.applyScale()
	buf, err := newShmBuffer(w.p.sess.shm, w.physW, w.physH, fillColor, w.onBufferReleased)
	if err != nil {
		w.p.fail(err)
		return
	}
	w.note("ack", w.physW, w.physH)
	w.ackConfigure(serial)
	w.configures++
	if w.configures == 1 {
		w.p.useStartupToken(w)
	}
	w.commit(buf)
	if resized || w.configures == 1 || infoChanged {
		w.reportMetrics()
	}
}

// commit puts buf on the surface. The buffer on display until now is freed once
// the compositor has released it.
func (w *window) commit(buf *shmBuffer) {
	w.note("commit", buf.width, buf.height)
	w.surface.Attach(buf.buffer, 0, 0)
	w.surface.DamageBuffer(0, 0, buf.width, buf.height)
	w.surface.Commit()
	if w.buffer != nil {
		w.old = append(w.old, w.buffer)
	}
	w.buffer = buf
	w.mapped = true
	w.collectReleased()
}

// note tells the trace of the tests, if there is one, about an event of the buffer.
func (w *window) note(event string, width, height int32) {
	if w.trace != nil {
		w.trace(fmt.Sprintf("%s %dx%d", event, width, height))
	}
}

// onBufferReleased runs when the compositor is done with a buffer. One that is
// no longer on display is freed at once.
func (w *window) onBufferReleased(*shmBuffer) {
	if !w.closed {
		w.collectReleased()
	}
}

// collectReleased frees the replaced buffers that the compositor released.
func (w *window) collectReleased() {
	kept := w.old[:0]
	for _, b := range w.old {
		if b.released {
			b.free()
		} else {
			kept = append(kept, b)
		}
	}
	w.old = kept
}

// reportMetrics tells the handler the size of the view.
func (w *window) reportMetrics() {
	if h := w.p.handler; h != nil {
		m := platform.ViewMetrics{
			PhysicalSize:     ui.Size{Width: float64(w.physW), Height: float64(w.physH)},
			DevicePixelRatio: float64(w.scale) / 120,
			Window:           w.info(),
		}
		if id, ok := w.displayOf(); ok {
			m.Display = foundation.Some(id)
		}
		w.display, w.hasDisplay = w.displayOf()
		h.OnMetrics(w.id, m)
	}
}

// scaleFactor is the number of physical pixels per logical pixel of the window.
func (w *window) scaleFactor() float64 { return float64(w.scale) / 120 }

// onEnter and onLeave track the outputs that the surface is on, whose scales are the
// last resort for the window's scale.
func (w *window) onEnter(output *wl.WlOutput) {
	w.entered = append(w.entered, output)
	w.displayChanged()
}

func (w *window) onLeave(output *wl.WlOutput) {
	for i, o := range w.entered {
		if o == output {
			w.entered = append(w.entered[:i], w.entered[i+1:]...)
			break
		}
	}
	w.displayChanged()
}

// displayChanged runs when the outputs the window is on, or what they say, changed:
// the scale may have changed, and the display the window is on, with the same scale
// or not, is told to the app.
func (w *window) displayChanged() {
	if w.closed {
		return
	}
	w.scaleChanged()
	if w.configures == 0 {
		return
	}
	if id, ok := w.displayOf(); ok != w.hasDisplay || id != w.display {
		w.reportMetrics()
	}
}

// effectiveScale is the scale the window should be drawn at, in 120ths, and whether it
// is fractional, so that the buffer is mapped to the window by a viewport. In order of
// preference: wp_fractional_scale_v1's, the surface's preferred_buffer_scale
// (wl_surface v6), and the largest scale of the outputs that the surface is on.
func (w *window) effectiveScale() (scale int32, fractional bool) {
	if w.viewport != nil && w.scale120 > 0 {
		return w.scale120, true
	}
	if w.bufScale > 0 {
		return w.bufScale * 120, false
	}
	best := int32(1)
	for _, o := range w.entered {
		if s := w.p.sess.outputScale[o]; s > best {
			best = s
		}
	}
	return best * 120, false
}

// applyScale tells the surface how its buffer is sized: with a fractional scale, the
// buffer is ceil(size × scale) pixels and the viewport maps it to the window's logical
// size; with an integer scale, the buffer has scale times the pixels and
// wl_surface.set_buffer_scale says so. It sets physW, physH and scale.
func (w *window) applyScale() {
	scale, fractional := w.effectiveScale()
	if scale != w.scale {
		defer w.p.cursors.scaleChanged(w)
	}
	w.scale = scale
	defer w.p.inputScaleChanged(w)
	if fractional {
		w.physW, w.physH = (w.width*scale+119)/120, (w.height*scale+119)/120
		w.viewport.SetDestination(w.width, w.height)
		w.viewportUsed = true
		if w.bufferScale != 1 {
			w.surface.SetBufferScale(1)
			w.bufferScale = 1
		}
		return
	}
	n := scale / 120
	w.physW, w.physH = w.width*n, w.height*n
	if w.bufferScale != n {
		w.surface.SetBufferScale(n)
		w.bufferScale = n
	}
	if w.viewportUsed {
		w.viewport.SetDestination(-1, -1)
		w.viewportUsed = false
	}
}

// resizeBuffer sizes the wl_egl_window for the logical size and the scale.
func (w *window) resizeBuffer() {
	w.applyScale()
	w.egl.resize(w.eglWindow, w.physW, w.physH, 0, 0)
}

// scaleChanged runs when something that decides the scale changed. The window draws
// again at once at the new scale, so that no frame at the old scale (blurry, or too
// big) is shown in between. Before the first configure there is nothing to redraw: it
// sizes the buffer. A change in how the same scale is expressed (a viewport instead of
// the buffer scale, as when a compositor sends its first fractional scale) is told to
// the surface and waits for the next frame.
func (w *window) scaleChanged() {
	if w.closed || w.configures == 0 {
		return
	}
	scale, fractional := w.effectiveScale()
	if scale == w.scale {
		if fractional != w.viewportUsed && w.egl != nil {
			w.applyScale()
		}
		return
	}
	if w.egl != nil {
		w.resizeBuffer()
		w.reportMetrics()
		w.renderNow()
		return
	}
	w.applyScale()
	buf, err := newShmBuffer(w.p.sess.shm, w.physW, w.physH, fillColor, w.onBufferReleased)
	if err != nil {
		w.p.fail(err)
		return
	}
	w.commit(buf)
	w.reportMetrics()
}

// onClose handles xdg_toplevel.close: the user asked to close the window, and
// the handler decides.
func (w *window) onClose() {
	if h := w.p.handler; h != nil && !h.OnCloseRequest(w.id) {
		return
	}
	w.Close()
}

// States returns the toplevel states the compositor reported last.
//
// Flutter: no counterpart
func (w *window) States() WindowStates { return w.states }

// ID returns the view ID of the window.
//
// Flutter: no counterpart
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface returns the handle for the renderer: the wl_display and the
// wl_egl_window. In shared-memory mode there is nothing to render into, and the
// handle is empty.
//
// Flutter: no counterpart
func (w *window) NativeSurface() gpu.SurfaceHandle {
	if w.eglWindow == 0 {
		return gpu.SurfaceHandle{}
	}
	return gpu.SurfaceHandle{
		Kind:    gpu.SurfaceKindWayland,
		Display: w.p.sess.conn.DisplayPtr(),
		Window:  w.eglWindow,
	}
}

// renderNow asks the handler for a frame at once, as a resize needs. The frame
// callback for the commit that the handler's swap makes is requested first, so
// it is attached to that commit.
func (w *window) renderNow() {
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	if w.states.Suspended {
		// The compositor hides the window: nothing is drawn until it is shown, and
		// then the frame is drawn at once (SPEC §8.11, "Frame pacing").
		w.frameRequested = true
		return
	}
	w.armFrameCallback()
	h.OnFrame(w.id, w.p.now())
}

// armFrameCallback requests a wl_surface.frame callback unless one is in
// flight.
func (w *window) armFrameCallback() {
	if w.frameInFlight {
		return
	}
	w.frameInFlight = true
	w.frameGen++
	gen := w.frameGen
	cb := w.surface.Frame()
	cb.SetEvents(&wl.WlCallbackEvents{OnDone: func(uint32) {
		cb.ProxyDestroy() // a wl_callback has no destructor request
		if gen == w.frameGen {
			w.onFrameDone()
		}
	}})
}

// onFrameDone runs when the compositor is ready for the next frame. A frame that
// was requested meanwhile is rendered now.
func (w *window) onFrameDone() {
	w.frameInFlight = false
	if w.closed || !w.frameRequested {
		return
	}
	w.frameRequested = false
	w.renderNow()
}

// RequestFrame asks for Handler.OnFrame at the next moment the compositor wants
// a frame. When no frame is in flight, the frame is rendered on the next pass of
// the loop, not inside this call; otherwise it waits for the frame callback of
// the frame in flight. The handler must present exactly one frame per OnFrame,
// or no callback arrives and no further frame follows. Shared-memory windows
// ignore the request.
//
// Flutter: no counterpart
func (w *window) RequestFrame() {
	if w.closed || w.egl == nil {
		return
	}
	w.frameRequested = true
	if w.frameInFlight || w.states.Suspended {
		return
	}
	w.p.loop.Post(func() {
		if w.closed || !w.frameRequested || w.frameInFlight {
			return
		}
		w.frameRequested = false
		w.renderNow()
	})
}

// SetTitle changes the window title.
//
// Flutter: no counterpart
func (w *window) SetTitle(title string) {
	if w.closed {
		return
	}
	w.title = title
	if w.frame != nil {
		w.frame.SetTitle(title)
		return
	}
	w.toplevel.SetTitle(title)
	if w.deco == ui.DecorationApp {
		w.reportMetrics() // the title bar of the app shows it
	}
}

// SetSize resizes the content of the window. A Wayland client sets its own
// size by drawing a buffer of the new size; the compositor may still override
// it with the next configure.
//
// Flutter: no counterpart
func (w *window) SetSize(size ui.Size) {
	width, height := int32(size.Width), int32(size.Height)
	if w.closed || width <= 0 || height <= 0 || (width == w.width && height == w.height) {
		return
	}
	w.width, w.height = width, height
	if w.egl != nil {
		if w.configures == 0 {
			return // the first configure draws at this size
		}
		if w.frame != nil {
			w.frame.Commit(int(w.width), int(w.height), decor.Configuration{})
		}
		w.resizeBuffer()
		w.reportMetrics()
		w.renderNow()
		return
	}
	if !w.mapped {
		return // the first configure draws at this size
	}
	w.applyScale()
	buf, err := newShmBuffer(w.p.sess.shm, w.physW, w.physH, fillColor, w.onBufferReleased)
	if err != nil {
		w.p.fail(err)
		return
	}
	w.commit(buf)
	w.reportMetrics()
}

// SetState asks the compositor to maximize, minimize or fullscreen the window,
// or to restore it.
//
// Flutter: no counterpart
func (w *window) SetState(state platform.WindowState) {
	if !w.closed {
		w.requestState(state)
	}
}

// SetCursor sets the cursor over the window. It takes effect at once when the
// pointer is in the window, and whenever the pointer enters it.
//
// Flutter: no counterpart
func (w *window) SetCursor(cursor platform.MouseCursor) {
	w.cursor = cursor
	if pt := w.p.pointer; pt != nil && pt.focus == w {
		w.p.cursors.apply(pt, cursor)
	}
	if t := w.p.tablet; t != nil {
		t.cursorChanged(w)
	}
}

// SetTextInput gives the input method the state of the text field of the window:
// zwp_text_input_v3 is enabled while the field has the focus and the window has
// the text-input focus, and disabled otherwise. Without the protocol (the
// compositor lacks zwp_text_input_manager_v3) it does nothing, and the app gets
// the keys with the compose table of xkbcommon. Without the protocol the input
// method is IBus or Fcitx5 over D-Bus, if one is running (SPEC §8.3).
//
// Flutter: no counterpart
func (w *window) SetTextInput(state ui.TextInputState) {
	if t := w.p.text; t != nil {
		t.set(w, state)
	} else if d := w.p.dbusIME; d != nil {
		d.set(w, state)
	}
}

// StartDrag starts a drag and drop operation from the window with the data of the
// request, through wl_data_device.start_drag. The pointer button that started it
// must be down; the end is reported through Handler.OnDragEnd.
//
// Flutter: no counterpart
func (w *window) StartDrag(request ui.DragRequest) error {
	return w.p.clip.startDrag(w, request)
}

// Close destroys the window. Destroy the renderer's surface for it first, since
// that surface draws into the window's native window. When this was the last
// window, Run returns. Closing twice does nothing.
//
// Flutter: no counterpart
func (w *window) Close() {
	if w.closed {
		return
	}
	w.destroy()
	delete(w.p.windows, w.id)
	if len(w.p.windows) == 0 {
		w.p.loop.Stop()
	}
}

// destroy releases the protocol objects, the native window and the buffers of the
// window. It does not touch the platform's list of windows.
func (w *window) destroy() {
	w.closed = true
	// The input devices let go of the window while it still counts: held keys are
	// released, the pointer leaves, and fingers are cancelled.
	if kb := w.p.keyboard; kb != nil {
		kb.windowClosed(w)
	}
	if pt := w.p.pointer; pt != nil {
		pt.windowClosed(w)
	}
	if t := w.p.touch; t != nil {
		t.windowClosed(w)
	}
	if t := w.p.tablet; t != nil {
		t.windowClosed(w)
	}
	if t := w.p.text; t != nil {
		t.forget(w)
	}
	if d := w.p.dbusIME; d != nil {
		d.forget(w)
	}
	if w.eglWindow != 0 {
		w.egl.destroy(w.eglWindow) // before the wl_surface it draws on
		w.eglWindow = 0
	}
	if w.inhibitor != nil {
		w.inhibitor.Destroy()
		w.inhibitor = nil
	}
	if w.contentType != nil {
		w.contentType.Destroy()
		w.contentType = nil
	}
	if w.fractional != nil {
		w.fractional.Destroy()
	}
	if w.viewport != nil {
		w.viewport.Destroy()
	}
	if w.decoration != nil {
		w.decoration.Destroy()
	}
	if w.dialog != nil {
		w.dialog.Destroy()
		w.dialog = nil
	}
	if w.frame != nil {
		w.frame.Free() // destroys the xdg_toplevel and the xdg_surface it made
		w.frame = nil
	}
	if w.toplevel != nil {
		w.toplevel.Destroy()
	}
	if w.xdgSurface != nil {
		w.xdgSurface.Destroy()
	}
	w.surface.Destroy()
	for _, b := range append(w.old, w.buffer) {
		if b != nil {
			b.free()
		}
	}
	w.old, w.buffer = nil, nil
	w.freeIconBuffers()
}
