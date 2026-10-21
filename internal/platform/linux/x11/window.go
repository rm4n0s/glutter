//go:build linux

package x11

import (
	"log/slog"
	"math"
	"os"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// defaultSize is the size of a window whose config has none.
var defaultSize = ui.Size{Width: 800, Height: 600}

// window is one top-level X window, drawn with OpenGL ES through EGL.
type window struct {
	p   *Platform
	id  ui.ViewID
	xid uint64
	cfg platform.WindowConfig

	colormap uint64 // 0 when the window uses the parent's visual

	// width and height are the size of the content, in pixels.
	width, height int32

	mapped bool
	closed bool

	// cursor is the cursor over the window, set with SetCursor.
	cursor platform.MouseCursor

	// Frame pacing: a frame that is wanted waits for the timer, which fires one
	// refresh interval after the last frame.
	frameTimer *linux.Timer
	lastFrame  time.Time
	// scale is the device pixel ratio of the monitor the window is on, display the ID
	// of that monitor (hasDisplay is false when there are no monitors).
	scale      float64
	display    int
	hasDisplay bool
	// frameWanted is a frame asked for before the window was mapped.
	frameWanted bool

	// state is what the window manager says about the window; title is its title.
	state windowState
	title string
}

var _ platform.Window = (*window)(nil)

// NewWindow creates a toplevel window and maps it. The window is visible once the
// window manager has handled the map request, which happens inside Run: the
// first frame is then requested from the handler.
//
// The window uses the X visual that EGL wants for its config, so that the
// renderer can draw into it (gles.X11Visual); when EGL is not usable the window
// uses the default visual and the renderer's Open will report the failure.
//
// Flutter: no counterpart
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	if err := cfg.Icon.Validate(); err != nil {
		return nil, err
	}
	w := &window{p: p, id: p.nextView, cfg: cfg}
	p.nextView++
	// The window is made on the monitor at the origin, or the primary one.
	w.scale, w.display, w.hasDisplay = p.scaleAt(0, 0)
	w.width, w.height = w.px(cfg.Size.Width), w.px(cfg.Size.Height)
	if w.width <= 0 || w.height <= 0 {
		w.width, w.height = w.px(defaultSize.Width), w.px(defaultSize.Height)
	}

	visual, depth := w.chooseVisual()
	attrs := dl.Calloc(1, 112)
	defer dl.Free(attrs)
	a := view[xSetWindowAttributes](attrs)
	a.eventMask = structureNotifyMask | exposureMask | propertyChangeMask
	a.colormap = w.colormap
	mask := uint64(cwBackPixel | cwBorderPixel | cwEventMask)
	if w.colormap != 0 {
		mask |= cwColormap
	}
	w.xid = p.x.createWindow(p.display, p.root, 0, 0, uint32(w.width), uint32(w.height), 0, depth, inputOutput, visual, mask, attrs)
	if w.xid == 0 {
		w.freeColormap()
		return nil, errors.New("X11WindowFailed", "XCreateWindow failed", "width", w.width, "height", w.height)
	}
	p.windows[w.xid] = w
	p.hadWindow = true

	w.setProperties()
	p.setXdndAware(w)
	if p.input != nil {
		p.input.selectWindowEvents(w.xid)
	}
	p.x.mapWindow(p.display, w.xid)
	if cfg.State == platform.WindowStateMinimized {
		p.x.iconifyWindow(p.display, w.xid, p.screen)
	}
	p.x.flush(p.display)
	return w, nil
}

// chooseVisual finds the X visual and depth for the window. EGL has the say: its
// config names a visual, which needs a colormap of its own. Without usable EGL
// the window gets its parent's visual (depth 0 and visual 0 are CopyFromParent).
func (w *window) chooseVisual() (visual uintptr, depth int32) {
	p := w.p
	id, err := p.visualFor(p.display)
	if err != nil {
		slog.Warn("EGL has no visual for X11, so the window cannot be drawn with OpenGL", "error", err)
		return 0, 0
	}
	template := dl.Calloc(1, 64)
	count := dl.Calloc(1, 4)
	defer dl.Free(template)
	defer dl.Free(count)
	view[xVisualInfo](template).visualID = uint64(id)
	list := p.x.getVisualInfo(p.display, visualIDMask, template, count)
	if list == 0 || *view[int32](count) < 1 {
		slog.Warn("the X server has no visual that EGL asked for", "visual", id)
		return 0, 0
	}
	defer p.x.free(list)
	info := view[xVisualInfo](list)
	w.colormap = p.x.createColormap(p.display, p.root, info.visual, allocNone)
	return info.visual, info.depth
}

func (w *window) freeColormap() {
	if w.colormap != 0 {
		w.p.x.freeColormap(w.p.display, w.colormap)
		w.colormap = 0
	}
}

// setProperties sets the ICCCM and EWMH properties of the window: the protocols
// it takes part in, its names, size hints, decorations and type (SPEC §8.3).
func (w *window) setProperties() {
	p, cfg := w.p, w.cfg
	at := &p.atoms

	protocols := dl.Calloc(2, 8)
	view[[2]uint64](protocols)[0], view[[2]uint64](protocols)[1] = at.wmDeleteWindow, at.netWMPing
	p.x.setWMProtocols(p.display, w.xid, protocols, 2)
	dl.Free(protocols)

	w.SetTitle(cfg.Title)
	if err := w.SetIcon(cfg.Icon); err != nil {
		slog.Warn("the window icon was not set", "error", err)
	}

	appID := cfg.AppID
	if appID == "" {
		appID = "glutter"
	}
	cid := dl.CString(appID)
	hint := dl.Calloc(1, 16)
	*view[xClassHint](hint) = xClassHint{resName: cid, resClass: cid}
	p.x.setClassHint(p.display, w.xid, hint)
	dl.Free(hint)
	dl.Free(cid)

	p.setLongs(w.xid, at.netWMPID, xaCardinal, []uint64{uint64(os.Getpid())})
	w.setSizeHints()

	if !cfg.IsDecorated() || cfg.Kind == platform.WindowKindBorderless || cfg.Decorations == platform.DecorationsApp {
		// _MOTIF_WM_HINTS: flags (MWM_HINTS_DECORATIONS), functions, decorations,
		// input mode, status. No decorations: there is no title bar, or the app draws
		// it.
		p.setLongs(w.xid, at.motifWMHints, at.motifWMHints, []uint64{1 << 1, 0, 0, 0, 0})
	}
	windowType := at.netWMWindowTypeNormal
	if cfg.Kind == platform.WindowKindDialog {
		windowType = at.netWMWindowTypeDialog
	}
	p.setLongs(w.xid, at.netWMWindowType, xaAtom, []uint64{windowType})
	if cfg.HasParent {
		if parent := p.windowByView(cfg.Parent); parent != nil {
			p.x.setTransientForHint(p.display, w.xid, parent.xid)
		}
	}
	var state []uint64
	switch cfg.State {
	case platform.WindowStateMaximized:
		state = append(state, at.netWMStateMaxVert, at.netWMStateMaxHorz)
	case platform.WindowStateFullscreen:
		state = append(state, at.netWMStateFullscreen)
	}
	if cfg.Modal && cfg.Kind == platform.WindowKindDialog {
		state = append(state, at.netWMStateModal)
	}
	if len(state) > 0 {
		p.setLongs(w.xid, at.netWMState, xaAtom, state)
	}
}

// setSizeHints sets WM_NORMAL_HINTS: the initial size, the smallest and largest
// size, which are the size itself for a window that cannot be resized.
func (w *window) setSizeHints() {
	mem := dl.Calloc(1, 80)
	defer dl.Free(mem)
	h := view[xSizeHints](mem)
	h.flags = pSize
	h.width, h.height = w.width, w.height
	min, max := sizeLimits(w.cfg, w.width, w.height, w.scale)
	if min != nil {
		h.flags |= pMinSize
		h.minWidth, h.minHeight = min[0], min[1]
	}
	if max != nil {
		h.flags |= pMaxSize
		h.maxWidth, h.maxHeight = max[0], max[1]
	}
	w.p.x.setWMNormalHints(w.p.display, w.xid, mem)
}

// sizeLimits returns the smallest and largest size of a window, or nil for no
// limit. A window that cannot be resized is limited to its own size.
func sizeLimits(cfg platform.WindowConfig, width, height int32, scale float64) (min, max *[2]int32) {
	if !cfg.IsResizable() {
		return &[2]int32{width, height}, &[2]int32{width, height}
	}
	if cfg.MinSize.Width > 0 || cfg.MinSize.Height > 0 {
		min = &[2]int32{int32(math.Round(cfg.MinSize.Width * scale)), int32(math.Round(cfg.MinSize.Height * scale))}
	}
	if cfg.MaxSize.Width > 0 || cfg.MaxSize.Height > 0 {
		max = &[2]int32{int32(math.Round(cfg.MaxSize.Width * scale)), int32(math.Round(cfg.MaxSize.Height * scale))}
	}
	return min, max
}

// windowByView returns the window of a view, or nil.
func (p *Platform) windowByView(id ui.ViewID) *window {
	for _, w := range p.windows {
		if w.id == id {
			return w
		}
	}
	return nil
}

// setLongs sets a property of format 32. Xlib takes the 32-bit items as an array
// of C longs, which are 8 bytes on this platform.
func (p *Platform) setLongs(window, property, typ uint64, values []uint64) {
	mem := dl.Calloc(uintptr(len(values)), 8)
	defer dl.Free(mem)
	copy(unsafe.Slice((*uint64)(dl.Pointer(mem)), len(values)), values)
	p.x.changeProperty(p.display, window, property, typ, 32, propModeReplace, mem, int32(len(values)))
}

// setBytes sets a property of format 8.
func (p *Platform) setBytes(window, property, typ uint64, data []byte) {
	mem := dl.Malloc(uintptr(len(data)) + 1)
	defer dl.Free(mem)
	copy(dl.Bytes(mem, len(data)), data)
	p.x.changeProperty(p.display, window, property, typ, 8, propModeReplace, mem, int32(len(data)))
}

// onClientMessage handles a message from the window manager: a request to close
// the window, or a ping to find out whether the app still responds.
func (w *window) onClientMessage(first uint64, ev *xClientMessageEvent) {
	p := w.p
	if ev.messageType != p.atoms.wmProtocols {
		return
	}
	switch first {
	case p.atoms.wmDeleteWindow:
		w.onClose()
	case p.atoms.netWMPing:
		// Answer with the same message, sent to the root window.
		ev.window = p.root
		p.x.sendEvent(p.display, p.root, 0, substructureNotifyMask|substructureRedirMask, p.event)
	}
}

// onClose asks the handler whether to close the window; it closes if it agrees.
func (w *window) onClose() {
	h := w.p.handler
	if h == nil || h.OnCloseRequest(w.id) {
		w.Close()
	}
}

// onConfigure handles the new size of the window.
func (w *window) onConfigure(width, height int32) {
	if width <= 0 || height <= 0 || (width == w.width && height == w.height) {
		// The window may still have moved to another monitor.
		w.updateScale()
		return
	}
	w.width, w.height = width, height
	scale := w.scale
	w.updateScale() // may report the metrics with the new scale
	if w.mapped {
		if scale == w.scale {
			w.reportMetrics()
		}
		w.renderNow() // track the resize without waiting for the timer
	}
}

// onMap handles the window becoming visible: the first metrics and the first
// frame.
func (w *window) onMap() {
	if w.mapped {
		return
	}
	w.mapped = true
	w.refreshState()
	w.reportMetrics()
	w.renderNow()
}

// onExpose redraws a window that was uncovered.
func (w *window) onExpose() {
	if w.mapped {
		// The server keeps nothing of the window when it is not composited: the next
		// frame must present all of it.
		if h := w.p.handler; h != nil {
			h.OnContentLost(w.id)
		}
		w.RequestFrame()
	}
}

// onDestroyedByServer handles a window that the server destroyed (its client
// connection being the only one that should): the window is forgotten.
func (w *window) onDestroyedByServer() {
	w.closed = true
	w.cancelFrame()
	w.freeColormap()
	delete(w.p.windows, w.xid)
	if len(w.p.windows) == 0 {
		w.p.loop.Stop()
	}
}

// reportMetrics tells the handler the size of the window.
func (w *window) reportMetrics() {
	if h := w.p.handler; h != nil {
		scale := w.scale
		if scale <= 0 {
			scale = 1
		}
		m := platform.ViewMetrics{
			PhysicalSize:     ui.Size{Width: float64(w.width), Height: float64(w.height)},
			DevicePixelRatio: scale,
			Window:           w.info(),
		}
		if w.hasDisplay {
			m.Display = foundation.Some(w.display)
		}
		h.OnMetrics(w.id, m)
	}
}

// renderNow asks the handler for a frame at once, and starts the interval to the
// next one.
func (w *window) renderNow() {
	if w.closed {
		return
	}
	w.cancelFrame()
	w.frameWanted = false
	w.lastFrame = time.Now()
	if h := w.p.handler; h != nil {
		h.OnFrame(w.id, w.p.now())
	}
}

func (w *window) cancelFrame() {
	if w.frameTimer != nil {
		w.frameTimer.Stop()
		w.frameTimer = nil
	}
}

// ID returns the ID of the view this window shows.
//
// Flutter: no counterpart
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface returns the handle that EGL draws into: the Xlib display and the
// window ID.
//
// Flutter: no counterpart
func (w *window) NativeSurface() gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Display: w.p.display, Window: uintptr(w.xid)}
}

// RequestFrame asks for Handler.OnFrame to be called for this window. X11 has no
// frame callbacks, so the frames are paced by a timer: the next frame comes one
// refresh interval after the last one, and requests made meanwhile are merged
// into it (SPEC §8.3).
//
// Flutter: no counterpart
func (w *window) RequestFrame() {
	if w.closed {
		return
	}
	if !w.mapped {
		w.frameWanted = true // the map draws the first frame
		return
	}
	if w.frameTimer != nil {
		return
	}
	delay := max(0, w.p.interval-time.Since(w.lastFrame))
	t, err := w.p.loop.AfterFunc(delay, func() {
		w.frameTimer = nil
		w.renderNow()
	})
	if err != nil {
		w.p.fail(err)
		return
	}
	w.frameTimer = t
}

// SetTitle sets the window title: _NET_WM_NAME as UTF-8 for modern window
// managers and WM_NAME as the older Latin-1 string.
//
// Flutter: no counterpart
func (w *window) SetTitle(title string) {
	if w.closed {
		return
	}
	p := w.p
	w.title = title
	p.setBytes(w.xid, p.atoms.netWMName, p.atoms.utf8String, []byte(title))
	p.setBytes(w.xid, xaWMName, xaString, []byte(latin1(title)))
}

// latin1 converts a string to Latin-1, with ? for the characters it lacks.
func latin1(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r < 0x100 {
			out = append(out, byte(r))
		} else {
			out = append(out, '?')
		}
	}
	return string(out)
}

// SetSize asks for a new size of the content. The window manager may refuse; the
// answer arrives as a configure event.
//
// Flutter: no counterpart
func (w *window) SetSize(size ui.Size) {
	width, height := w.px(size.Width), w.px(size.Height)
	if w.closed || width <= 0 || height <= 0 {
		return
	}
	w.p.x.resizeWindow(w.p.display, w.xid, uint32(width), uint32(height))
}

// SetState asks the window manager to maximize, minimize or fullscreen the
// window, or to restore it (_NET_WM_STATE, ICCCM iconify).
//
// Flutter: no counterpart
func (w *window) SetState(state platform.WindowState) {
	if w.closed {
		return
	}
	p, at := w.p, &w.p.atoms
	switch state {
	case platform.WindowStateMaximized:
		w.sendState(netWMStateAdd, at.netWMStateMaxVert, at.netWMStateMaxHorz)
	case platform.WindowStateFullscreen:
		w.sendState(netWMStateAdd, at.netWMStateFullscreen, 0)
	case platform.WindowStateMinimized:
		p.x.iconifyWindow(p.display, w.xid, p.screen)
	case platform.WindowStateNormal, platform.WindowStateUnspecified:
		w.sendState(netWMStateRemove, at.netWMStateMaxVert, at.netWMStateMaxHorz)
		w.sendState(netWMStateRemove, at.netWMStateFullscreen, 0)
		p.x.mapWindow(p.display, w.xid) // brings back a window that was iconified
	}
	p.x.flush(p.display)
}

// sendState sends the window manager a _NET_WM_STATE request to add or remove
// one or two states.
func (w *window) sendState(action int64, first, second uint64) {
	p := w.p
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	ev := view[xClientMessageEvent](mem)
	ev.typ = clientMessage
	ev.window = w.xid
	ev.messageType = p.atoms.netWMState
	ev.format = 32
	ev.data = [5]int64{action, int64(first), int64(second), 1 /* from an application */, 0}
	p.x.sendEvent(p.display, p.root, 0, substructureNotifyMask|substructureRedirMask, mem)
}

// SetCursor sets the cursor over the window.
//
// Flutter: no counterpart
func (w *window) SetCursor(cursor platform.MouseCursor) {
	w.cursor = cursor
	if !w.closed {
		w.p.cursors.apply(w, cursor)
	}
}

// SetTextInput gives the input method the state of the text field of the
// window. IBus or Fcitx5 is reached over D-Bus (SPEC §8.3, §8.6); when neither
// is running the state is dropped and the keys go to the app, with the compose
// table of xkbcommon.
//
// Flutter: no counterpart
func (w *window) SetTextInput(state ui.TextInputState) {
	w.p.setTextInput(w, state)
}

// StartDrag starts a drag and drop operation from the window with the data of the
// request, with XDND. The pointer button that started it must be down; the end is
// reported through Handler.OnDragEnd.
//
// Flutter: no counterpart
func (w *window) StartDrag(request ui.DragRequest) error {
	return w.p.xdnd.startDrag(w, request)
}

// SetContentType does nothing: X11 has no content type hint.
//
// Flutter: no counterpart
func (w *window) SetContentType(ui.ContentType) {}

// KeepScreenOn fails with IdleInhibitUnavailable: the X11 backend has no way to
// inhibit the screensaver yet.
//
// Flutter: no counterpart
func (w *window) KeepScreenOn(bool) error {
	return errors.New("IdleInhibitUnavailable", "the X11 backend cannot keep the screen on")
}

// PerformWindowAction does what the title bar of the app asks: minimize, maximize,
// restore, fullscreen, activate and close are requests to the window manager
// (_NET_WM_STATE, ICCCM iconify, _NET_ACTIVE_WINDOW), and the move, the resize from
// an edge and the window menu are too (_NET_WM_MOVERESIZE and _GTK_SHOW_WINDOW_MENU,
// for a window whose title bar the app draws). They need the pointer button that is
// down; without one they do nothing.
//
// Flutter: no counterpart
func (w *window) PerformWindowAction(action ui.WindowAction) {
	if w.closed {
		return
	}
	p, at := w.p, &w.p.atoms
	switch action.Kind {
	case ui.WindowActionMinimize:
		w.SetState(platform.WindowStateMinimized)
	case ui.WindowActionMaximize:
		w.sendState(netWMStateAdd, at.netWMStateMaxVert, at.netWMStateMaxHorz)
	case ui.WindowActionRestore:
		w.sendState(netWMStateRemove, at.netWMStateMaxVert, at.netWMStateMaxHorz)
	case ui.WindowActionFullscreen:
		w.sendState(netWMStateAdd, at.netWMStateFullscreen, 0)
	case ui.WindowActionUnfullscreen:
		w.sendState(netWMStateRemove, at.netWMStateFullscreen, 0)
	case ui.WindowActionActivate:
		p.x.mapWindow(p.display, w.xid) // brings back a window that was iconified
		w.sendActivate()
	case ui.WindowActionClose:
		w.onClose()
	case ui.WindowActionMove:
		w.moveResize(moveResizeMove)
	case ui.WindowActionResize:
		w.moveResize(edgeDirection(action.Edge))
	case ui.WindowActionShowMenu:
		w.showMenu(action.Position)
	}
	p.x.flush(p.display)
}

// sendActivate asks the window manager to activate the window
// (_NET_ACTIVE_WINDOW, from an application).
func (w *window) sendActivate() {
	p := w.p
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	ev := view[xClientMessageEvent](mem)
	ev.typ = clientMessage
	ev.window = w.xid
	ev.messageType = p.atoms.netActiveWindow
	ev.format = 32
	ev.data = [5]int64{1 /* from an application */, 0, 0, 0, 0}
	p.x.sendEvent(p.display, p.root, 0, substructureNotifyMask|substructureRedirMask, mem)
}

// SetSizeConstraints sets the smallest and the largest size of the window
// (WM_NORMAL_HINTS). The window manager may ignore them.
//
// Flutter: widgets/_window.dart RegularWindowController.setConstraints
func (w *window) SetSizeConstraints(min, max ui.Size) {
	if w.closed {
		return
	}
	w.cfg.MinSize, w.cfg.MaxSize = min, max
	w.setSizeHints()
	w.p.x.flush(w.p.display)
}

// Close destroys the window. Destroy the renderer's surface for it first. When
// this was the last window, Run returns. Closing twice does nothing.
//
// Flutter: no counterpart
func (w *window) Close() {
	if w.closed {
		return
	}
	w.closed = true
	w.cancelFrame()
	if w.p.text != nil {
		w.p.text.windowClosed(w)
	}
	if w.p.input != nil {
		w.p.input.windowClosed(w)
	}
	w.p.xdnd.windowClosed(w)
	w.p.x.destroyWindow(w.p.display, w.xid)
	w.freeColormap()
	delete(w.p.windows, w.xid)
	w.p.x.flush(w.p.display)
	if len(w.p.windows) == 0 {
		w.p.loop.Stop()
	}
}
