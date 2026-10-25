//go:build windows

package windows

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/ime"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// defaultSize is the size, in logical pixels, of a window whose config has none.
var defaultSize = ui.Size{Width: 800, Height: 600}

// modalTimerID is the id of the timer that keeps the app running inside a modal loop of Windows.
const modalTimerID = 1

// modalTimerInterval is the period of that timer; Windows rounds it up to the system's resolution.
const modalTimerInterval = 8 * time.Millisecond

// window is one top-level Win32 window (SPEC §25.4).
type window struct {
	p     *Platform
	id    ui.ViewID
	hwnd  uintptr
	token uintptr
	cfg   platform.WindowConfig
	owner *window

	title  string
	styles styleSet
	// dpi is the DPI of the window; clientW and clientH are the size of its client area in
	// physical pixels, as the last WM_SIZE or the creation said.
	dpi              uint32
	clientW, clientH int32
	minSize, maxSize ui.Size

	state      windowState
	activated  bool
	focusKnown bool
	// created is whether CreateWindowEx returned; the messages before are ignored. shown is
	// whether the first frame was drawn and the window shown. hidden is the app's SetHidden.
	created, shown, hidden, closed bool
	// initial is the state the first show applies; restore is the state a hidden window gets back.
	initial platform.WindowState
	restore platform.WindowState

	frames frameRequests
	// sizeSeq counts the reports of the size, to know whether a call to SetWindowPos made one.
	sizeSeq int

	// fullscreen: saved is what the window had before.
	saved struct {
		styles    styleSet
		placement win32.WindowPlacement
	}
	keepOn bool
	icons  [2]uintptr
	cursor platform.MouseCursor
	// modalTimer is whether a modal loop's timer is running for this window.
	modalTimer bool

	// The translators of the pointer input (input_windows.go). synthetic is set while a scripted
	// event is sent to the window.
	mouse     *mouseTranslator
	touch     *pointerTranslator
	synthetic bool
	// text is the text input of the window: the input method and the characters (textinput_windows.go).
	text *ime.Input
}

var _ platform.Window = (*window)(nil)

// NewWindow creates a top-level window, hidden: it is shown after its first frame is drawn, so
// that it never shows a blank client area (SPEC §25.4). The size is in logical pixels, the
// client area is that size times the scale of the monitor the window opens on, and the outer size
// is worked out from the style at that DPI. It fails with Win32WindowCreateFailed and
// WindowIconInvalid.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::HostWindow
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	if err := cfg.Icon.Validate(); err != nil {
		return nil, err
	}
	w := &window{
		p: p, id: p.nextView, cfg: cfg, title: cfg.Title, styles: styleOf(cfg),
		minSize: cfg.MinSize, maxSize: cfg.MaxSize, initial: cfg.State,
	}
	switch cfg.State {
	case platform.WindowStateUnspecified:
		w.initial = platform.WindowStateNormal
	}
	w.restore = w.initial
	p.nextView++

	if cfg.Kind == platform.WindowKindDialog && cfg.HasParent {
		w.owner = p.windowByView(cfg.Parent)
	}
	var ownerHwnd uintptr
	if w.owner != nil {
		ownerHwnd = w.owner.hwnd
	}

	// The window opens on the primary monitor: the DPI is that monitor's.
	w.dpi = win32.GetDpiForMonitor(win32.MonitorFromPoint(win32.Point{}, win32.MonitorDefaultToPrimary))
	size := cfg.Size
	if size.Width <= 0 || size.Height <= 0 {
		size = defaultSize
	}
	scale := scaleOfDPI(w.dpi)
	cw, ch := physical(size.Width, scale), physical(size.Height, scale)
	ow, oh := w.outerSize(cw, ch)

	w.token = registerTarget(w)
	useDefault := int32(-1 << 31) // CW_USEDEFAULT, 0x80000000
	hwnd, err := win32.CreateWindowEx(w.styles.exStyle, windowClassName, cfg.Title, w.styles.style&^win32.WSVisible,
		useDefault, useDefault, ow, oh, ownerHwnd, 0, p.instance, w.token)
	if err != nil {
		unregisterTarget(w.token)
		return nil, err
	}
	w.hwnd = hwnd
	w.created = true
	p.windows[hwnd] = w
	w.initInput()
	// The window may open on a monitor with another scale than the primary's.
	if dpi := win32.GetDpiForWindow(hwnd); dpi != 0 {
		w.dpi = dpi
	}
	if r, err := win32.GetClientRect(hwnd); err == nil {
		w.clientW, w.clientH = r.Right-r.Left, r.Bottom-r.Top
	} else {
		w.clientW, w.clientH = cw, ch
	}

	if cfg.AppID != "" && !p.appIDSet {
		p.appIDSet = true
		if err := win32.SetCurrentProcessExplicitAppUserModelID(cfg.AppID); err != nil {
			p.log.Warn("the app id could not be set; the task bar groups the windows by the program", "err", err)
		}
	}
	if err := w.SetIcon(cfg.Icon); err != nil {
		p.log.Warn("the window icon was not set", "err", err)
	}
	w.applyTitleBarTheme()
	if ownsClientArea(cfg) {
		// WM_NCCALCSIZE came while the window was made, before it was handled: ask for it again, so that
		// the client area covers the caption from the start.
		if err := win32.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win32.SWPNoMove|win32.SWPNoSize|win32.SWPNoZOrder|win32.SWPNoActivate|win32.SWPFrameChanged); err != nil {
			p.log.Warn("the client area of the window was not recalculated", "err", err)
		}
		if r, err := win32.GetClientRect(hwnd); err == nil {
			w.clientW, w.clientH = r.Right-r.Left, r.Bottom-r.Top
		}
	}
	if cfg.Modal && w.owner != nil {
		win32.EnableWindow(w.owner.hwnd, false)
	}
	if p.handler != nil {
		// A window made while Run is going: it shows itself after the handler has it.
		p.Post(p.showNewWindows)
	}
	return w, nil
}

// outerSize is the outer size of a window whose client area is the size, at the window's DPI.
// A window whose title bar the app draws has the client area of the whole window.
func (w *window) outerSize(cw, ch int32) (int32, int32) {
	in := w.insets()
	return in.outer(cw, ch)
}

// insets are the non-client sizes of the window's style at its DPI; zero for a window whose
// client area covers the caption (the app draws the title bar).
func (w *window) insets() insets {
	if ownsClientArea(w.cfg) && !w.fullscreen() {
		return insets{}
	}
	const probe = 100
	rc := win32.Rect{Right: probe, Bottom: probe}
	style := w.styles.style
	if w.hwnd != 0 {
		style = uint32(win32.GetWindowLongPtr(w.hwnd, win32.GWLStyle))
	}
	if err := win32.AdjustWindowRectExForDpi(&rc, style, w.styles.exStyle, w.dpi); err != nil {
		return insets{}
	}
	return insetsOf(rc, probe, probe)
}

// windowByView finds a window by its view.
func (p *Platform) windowByView(id ui.ViewID) *window {
	for _, w := range p.windows {
		if w.id == id {
			return w
		}
	}
	return nil
}

// sortedWindows lists the windows in the order they were made.
func (p *Platform) sortedWindows() []*window {
	ws := make([]*window, 0, len(p.windows))
	for _, w := range p.windows {
		ws = append(ws, w)
	}
	slices.SortFunc(ws, func(a, b *window) int { return int(a.id) - int(b.id) })
	return ws
}

// keepOnCount is the number of windows that asked to keep the screen on.
func (p *Platform) keepOnCount() int {
	n := 0
	for _, w := range p.windows {
		if w.keepOn {
			n++
		}
	}
	return n
}

// showNewWindows gives each window that is not shown its first frame and shows it.
func (p *Platform) showNewWindows() {
	for _, w := range p.sortedWindows() {
		if !w.shown && !w.closed && !w.hidden {
			w.firstShow()
		}
	}
}

// firstShow reports the metrics, draws the first frame into the hidden window and shows it with
// the state it was asked to start in, so that the window never shows at its normal size first
// (SPEC §25.4). The painted frame is kept by the renderer's presenter, which paints it again when
// the show exposes the window.
func (w *window) firstShow() {
	p := w.p
	if p.handler == nil {
		return
	}
	w.shown = true
	w.reportMetrics()
	w.renderNow()
	if w.closed {
		return
	}
	w.show(w.initial)
	if !w.focusKnown && !w.closed {
		w.reportFocus(w.activated)
	}
	p.lifecycleChanged()
	p.startDevEvents()
}

// show shows the window in the state.
func (w *window) show(st platform.WindowState) {
	switch st {
	case platform.WindowStateMaximized:
		win32.ShowWindow(w.hwnd, win32.SWShowMaximized)
	case platform.WindowStateMinimized:
		win32.ShowWindow(w.hwnd, win32.SWShowMinimized)
	case platform.WindowStateFullscreen:
		win32.ShowWindow(w.hwnd, win32.SWShowNormal)
		w.enterFullscreen()
	default:
		win32.ShowWindow(w.hwnd, win32.SWShowNormal)
	}
}

// scale is the device pixel ratio of the window.
func (w *window) scale() float64 { return scaleOfDPI(w.dpi) }

// fullscreen is whether the window is in full screen.
func (w *window) fullscreen() bool { return w.state.fullscreen }

// info is the WindowInfo for the engine.
func (w *window) info() ui.WindowInfo {
	style := w.styles.style
	if w.hwnd != 0 {
		style = uint32(win32.GetWindowLongPtr(w.hwnd, win32.GWLStyle))
	}
	if w.fullscreen() {
		style = w.saved.styles.style
	}
	return windowInfoOf(w.cfg, style, w.state, w.activated, w.title)
}

// reportMetrics tells the handler the size and the scale of the window. A minimized window keeps
// the size it had.
func (w *window) reportMetrics() {
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	w.sizeSeq++
	m := platform.ViewMetrics{
		PhysicalSize:     ui.Size{Width: float64(w.clientW), Height: float64(w.clientH)},
		DevicePixelRatio: w.scale(),
		Window:           w.info(),
	}
	if mon := win32.MonitorFromWindow(w.hwnd, win32.MonitorDefaultToNearest); mon != 0 {
		m.Display = foundation.Some(int(mon))
	}
	h.OnMetrics(w.id, m)
}

// renderNow asks the handler for a frame at once: a resize draws at the new size inside the
// handler of WM_SIZE, as a Wayland configure does (SPEC §8.11). A frame that was wanted is drawn
// by it.
func (w *window) renderNow() {
	h := w.p.handler
	if h == nil || w.closed || !w.canDraw() {
		return
	}
	w.frames.clear()
	h.OnFrame(w.id, time.Since(w.p.start))
}

// canDraw is whether a frame can be drawn now: the window is shown, not hidden and not minimized
// (SPEC §25.3), and has a size.
func (w *window) canDraw() bool {
	return w.shown && !w.hidden && !w.closed && !w.state.minimized && w.clientW > 0 && w.clientH > 0
}

// reportFocus tells the handler that the view got or lost the focus.
func (w *window) reportFocus(focused bool) {
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	w.focusKnown = true
	h.OnViewFocus(w.id, focused)
}

// reportKeyboardFocus tells a handler that follows the keyboard focus (platform.KeyboardFocusHandler)
// which window the keys go to.
func (w *window) reportKeyboardFocus(focused bool) {
	if w.closed {
		return
	}
	if kf, ok := w.p.handler.(platform.KeyboardFocusHandler); ok {
		kf.OnKeyboardFocus(w.id, focused)
	}
}

// handleMessage is the window procedure of the window (SPEC §25.4).
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage
func (w *window) handleMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	if w.hwnd == 0 {
		w.hwnd = hwnd // the messages of CreateWindowEx come before it returns
	}
	if !w.created {
		return win32.DefWindowProc(hwnd, msg, wparam, lparam)
	}
	return w.p.guard(func() uintptr { return w.dispatch(hwnd, msg, wparam, lparam) })
}

// dispatch handles one message.
func (w *window) dispatch(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	p := w.p
	switch msg {
	case win32.WMEraseBkgnd:
		return 1 // the renderer paints every pixel
	case win32.WMPaint:
		return w.onPaint(hwnd, msg, wparam, lparam)
	case win32.WMSize:
		w.onSize(wparam, lparam)
		return 0
	case win32.WMGetMinMaxInfo:
		w.onGetMinMaxInfo(win32.View[win32.MinMaxInfo](lparam))
		return 0
	case win32.WMDpiChanged:
		w.onDpiChanged(wparam, win32.View[win32.Rect](lparam))
		return 0
	case win32.WMQueryEndSession:
		return p.onQueryEndSession()
	case win32.WMEndSession:
		p.onEndSession(sessionEnding(wparam))
		return 0
	case win32.WMShowWindow:
		p.lifecycleChanged()
	case win32.WMActivate:
		w.activated = wparam&0xFFFF != win32.WAInactive
		p.lifecycleChanged()
		w.reportFocus(w.activated)
		if w.shown {
			w.reportMetrics() // the window info says whether the window is active
		}
		// DefWindowProc moves the keyboard focus to the window.
	case win32.WMSetFocus:
		w.reportKeyboardFocus(true)
		p.keys.focusGained()
		if w.text != nil {
			w.text.FocusChanged()
		}
	case win32.WMKillFocus:
		// The keys held down are released before the handler is told that the keys go elsewhere.
		p.keys.releaseAll()
		w.reportKeyboardFocus(false)
	case win32.WMKeyDown, win32.WMKeyUp, win32.WMSysKeyDown, win32.WMSysKeyUp:
		if result, handled := w.onKey(msg, wparam, lparam); handled {
			return result
		}
	case win32.WMChar, win32.WMDeadChar, win32.WMSysChar, win32.WMSysDeadChar:
		if result, handled := w.onChar(msg, wparam); handled {
			return result
		}
	case win32.WMIMESetContext, win32.WMIMEStartComposition, win32.WMIMEComposition, win32.WMIMEEndComposition:
		result, handled, out := w.onIME(msg, wparam, lparam)
		if handled {
			return result
		}
		lparam = out
	case win32.WMUniChar:
		if wparam == win32.UnicodeNoChar {
			return 1 // this window understands WM_UNICHAR; DefWindowProc makes WM_CHAR of the others
		}
	case win32.WMInputLangChange:
		p.keys.layoutChanged()
	case win32.WMMouseMove, win32.WMLButtonDown, win32.WMLButtonUp, win32.WMRButtonDown, win32.WMRButtonUp,
		win32.WMMButtonDown, win32.WMMButtonUp, win32.WMXButtonDown, win32.WMXButtonUp, win32.WMMouseLeave:
		if result, handled := w.onMouse(msg, wparam, lparam); handled {
			return result
		}
	case win32.WMMouseWheel, win32.WMMouseHWheel:
		w.onWheel(msg, wparam, lparam)
		return 0
	case win32.WMCaptureChanged:
		w.onCaptureChanged(lparam)
	case win32.WMCancelMode:
		w.onCancelMode()
	case win32.WMPointerEnter, win32.WMPointerDown, win32.WMPointerUpdate, win32.WMPointerUp, win32.WMPointerLeave, win32.WMPointerCaptureChanged:
		if w.onPointer(msg, wparam) {
			return 0
		}
	case win32.WMSetCursor:
		if w.onSetCursor(lparam) {
			return 1
		}
	case win32.WMClose:
		w.onClose()
		return 0
	case win32.WMNCCalcSize:
		if ownsClientArea(w.cfg) && wparam != 0 {
			w.onNCCalcSize(win32.View[win32.NcCalcSizeParams](lparam))
			return 0
		}
	case win32.WMNCHitTest:
		if ownsClientArea(w.cfg) {
			return w.onNCHitTest(lparam)
		}
	case win32.WMEnterSizeMove, win32.WMEnterMenuLoop:
		w.startModalTimer()
	case win32.WMExitSizeMove, win32.WMExitMenuLoop:
		w.stopModalTimer()
	case win32.WMTimer:
		if wparam == modalTimerID {
			p.modalTurn()
			return 0
		}
	case win32.WMSettingChange, win32.WMDwmColorizationColorChanged:
		p.settingChanged()
		if msg == win32.WMDwmColorizationColorChanged {
			return 0
		}
	case win32.WMNCDestroy:
		w.onDestroyed()
	}
	return win32.DefWindowProc(hwnd, msg, wparam, lparam)
}

// onPaint answers WM_PAINT: the renderer's presenter paints the kept frame when it has one (an
// exposure needs no new frame); otherwise the region is validated and nothing is drawn.
func (w *window) onPaint(hwnd uintptr, _ uint32, _, _ uintptr) uintptr {
	if paint := win32.PaintHandler(hwnd); paint != nil {
		paint(hwnd)
		return 0
	}
	ps, err := win32.BeginPaint(hwnd)
	if err == nil {
		win32.EndPaint(hwnd, &ps)
	}
	return 0
}

// onSize handles WM_SIZE: it works out the state and the client size, and lays out, renders and
// presents a frame at the new size before it returns (SPEC §25.4, §25.3).
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleResize
func (w *window) onSize(wparam, lparam uintptr) {
	st, ok := stateOfSize(wparam, w.fullscreen())
	if !ok {
		return
	}
	cw, ch := int32(lparam&0xFFFF), int32((lparam>>16)&0xFFFF)
	p := w.p
	defer p.lifecycleChanged()
	if st.minimized {
		changed := !w.state.minimized
		w.state = st
		if w.shown && changed {
			w.reportMetrics() // with the size it had: the window info says it is minimized
		}
		return
	}
	changed := cw != w.clientW || ch != w.clientH || st != w.state
	w.state = st
	if cw > 0 && ch > 0 {
		w.clientW, w.clientH = cw, ch
	}
	if !w.shown || w.hidden || !changed || cw <= 0 || ch <= 0 {
		return
	}
	w.reportMetrics()
	w.renderNow()
}

// onGetMinMaxInfo gives the tracking limits from the constraints (SPEC §25.4).
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::HandleMessage (WM_GETMINMAXINFO)
func (w *window) onGetMinMaxInfo(info *win32.MinMaxInfo) {
	if w.fullscreen() {
		return
	}
	screenW := win32.GetSystemMetrics(win32.SMCxVirtualScreen)
	screenH := win32.GetSystemMetrics(win32.SMCyVirtualScreen)
	mn, mx, hasMin, hasMax := minMaxLimits(w.minSize, w.maxSize, w.scale(), w.insets(), screenW, screenH)
	if hasMin {
		info.MinTrackSize = mn
	}
	if hasMax {
		info.MaxTrackSize = mx
	}
}

// onDpiChanged applies the rectangle that Windows suggests for the new DPI, which keeps the
// logical size, and draws the frame at the new scale inside the handler, so that no blurry frame
// shows in between (SPEC §25.4, §8.11).
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::HandleMessage (WM_DPICHANGED)
func (w *window) onDpiChanged(wparam uintptr, suggested *win32.Rect) {
	if dpi := uint32(wparam & 0xFFFF); dpi != 0 {
		w.dpi = dpi
	}
	before := w.sizeSeq
	_ = win32.SetWindowPos(w.hwnd, 0, suggested.Left, suggested.Top, suggested.Right-suggested.Left, suggested.Bottom-suggested.Top,
		win32.SWPNoZOrder|win32.SWPNoActivate) // a window that cannot move keeps its size; the metrics below still follow the new scale
	if w.sizeSeq == before && w.shown && !w.hidden && !w.state.minimized {
		w.reportMetrics()
		w.renderNow()
	}
	w.p.displaysChanged()
}

// onClose asks the handler whether to close the window, and closes it if it agrees.
func (w *window) onClose() {
	h := w.p.handler
	if h == nil || h.OnCloseRequest(w.id) {
		w.Close()
	}
}

// onNCCalcSize makes the client area of a window whose title bar the app draws cover the caption
// (WM_NCCALCSIZE answered with 0 leaves the proposed rectangle, the whole window, as the client
// area); a maximized window is inset by the frame, or it would reach beyond the screen.
//
// Flutter: no counterpart
func (w *window) onNCCalcSize(params *win32.NcCalcSizeParams) {
	if !win32.IsZoomed(w.hwnd) || w.fullscreen() {
		return
	}
	inset := maximizedInset(win32.GetSystemMetricsForDpi(win32.SMCxSizeFrame, w.dpi), win32.GetSystemMetricsForDpi(win32.SMCxPaddedBorder, w.dpi))
	r := &params.Rects[0]
	r.Left += inset
	r.Top += inset
	r.Right -= inset
	r.Bottom -= inset
}

// onNCHitTest answers the resize directions around a window whose title bar the app draws.
func (w *window) onNCHitTest(lparam uintptr) uintptr {
	pt := win32.Point{X: int32(int16(lparam & 0xFFFF)), Y: int32(int16((lparam >> 16) & 0xFFFF))}
	rc, err := win32.GetWindowRect(w.hwnd)
	if err != nil {
		return win32.HTClient
	}
	border := win32.GetSystemMetricsForDpi(win32.SMCxSizeFrame, w.dpi) + win32.GetSystemMetricsForDpi(win32.SMCxPaddedBorder, w.dpi)
	return hitTest(pt, rc, border, w.cfg.IsResizable(), win32.IsZoomed(w.hwnd) || w.fullscreen())
}

// startModalTimer starts the timer that keeps the loop going while Windows runs a modal loop of
// its own (SPEC §25.3): the posted functions, the timers and the frames that were asked for.
func (w *window) startModalTimer() {
	if w.modalTimer {
		return
	}
	if err := win32.SetTimer(w.hwnd, modalTimerID, modalTimerInterval); err != nil {
		w.p.log.Warn("the timer of the modal loop could not be started", "err", err)
		return
	}
	w.modalTimer = true
	w.p.modalTimers++
}

func (w *window) stopModalTimer() {
	if !w.modalTimer {
		return
	}
	win32.KillTimer(w.hwnd, modalTimerID)
	w.modalTimer = false
	w.p.modalTimers--
}

// onDestroyed handles the end of the window: it leaves the platform's table and the message table,
// and the last one ends Run.
func (w *window) onDestroyed() {
	p := w.p
	if w.touch != nil {
		w.touch.cancelAll() // the pointers of a window that goes away are cancelled
		w.flushPointers()
	}
	w.closed = true
	w.modalTimer = false
	w.releaseIcons()
	delete(p.windows, w.hwnd)
	p.lifecycleChanged()
	unregisterTarget(w.token)
	if w.keepOn {
		w.keepOn = false
		p.updateExecutionState()
	}
	if len(p.windows) == 0 {
		p.quit = true
	}
}

// modalTurn is a turn of the loop run by the timer of a modal loop.
func (p *Platform) modalTurn() {
	p.runPosted()
	p.timers.runDue(time.Now())
	p.deliverFrames()
}

// scheduleVsync asks the vsync thread for a tick when some window wants a frame and can draw it.
func (p *Platform) scheduleVsync() {
	if p.vsync == nil {
		return
	}
	wanted := false
	for _, w := range p.windows {
		if w.frames.wanted && w.canDraw() {
			wanted = true
			break
		}
	}
	if p.gate.shouldRequest(wanted) {
		p.vsync.request()
	}
}

// onVsync handles the tick of the vsync thread: the windows that want a frame draw it, and the
// next tick is asked for when a frame is wanted again.
func (p *Platform) onVsync() {
	p.gate.delivered()
	p.deliverFrames()
	p.scheduleVsync()
}

// deliverFrames asks the handler for the frame of each window that wanted one. The frame time is
// the time of the tick.
func (p *Platform) deliverFrames() {
	if p.handler == nil {
		return
	}
	now := time.Since(p.start)
	for _, w := range p.sortedWindows() {
		if w.frames.take(w.canDraw()) {
			p.handler.OnFrame(w.id, now)
		}
	}
}

// updateExecutionState keeps the display on while a window asked for it.
func (p *Platform) updateExecutionState() {
	state := win32.ESContinuous
	if p.keepOnCount() > 0 {
		state |= win32.ESDisplayRequired
	}
	win32.SetThreadExecutionState(state)
}

// ID returns the ID of the view this window shows.
//
// Flutter: no counterpart
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface returns the handle that the renderers draw into: the HWND and the HINSTANCE
// (SPEC §25.5).
//
// Flutter: no counterpart
func (w *window) NativeSurface() gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Display: w.p.instance, Window: w.hwnd}
}

// SetTitle sets the window title.
//
// Flutter: no counterpart
func (w *window) SetTitle(title string) {
	if w.closed {
		return
	}
	w.title = title
	if err := win32.SetWindowText(w.hwnd, title); err != nil {
		w.p.log.Warn("the window title was not set", "err", err)
	}
}

// RequestFrame asks for Handler.OnFrame to be called for this window at the next vsync. Requests
// made while one is wanted are one frame, and a window that cannot draw (minimized, hidden, not
// shown yet) keeps the request until it can (SPEC §25.3).
//
// Flutter: no counterpart
func (w *window) RequestFrame() {
	if w.closed {
		return
	}
	w.frames.request()
	w.p.scheduleVsync()
}

// SetSize asks for a new size of the content, in logical pixels. The window is resized with
// SetWindowPos, which reports the new size and draws a frame at it before this returns. A
// maximized, minimized or full-screen window ignores it.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::SetContentSize
func (w *window) SetSize(size ui.Size) {
	if w.closed || size.Width <= 0 || size.Height <= 0 || w.state.maximized || w.state.minimized || w.fullscreen() {
		return
	}
	scale := w.scale()
	ow, oh := w.outerSize(physical(size.Width, scale), physical(size.Height, scale))
	if err := win32.SetWindowPos(w.hwnd, 0, 0, 0, ow, oh, win32.SWPNoMove|win32.SWPNoZOrder|win32.SWPNoActivate); err != nil {
		w.p.log.Warn("the window could not be resized", "err", err)
	}
}

// SetSizeConstraints sets the smallest and the largest size of the window in logical pixels; 0 is
// no limit. Windows asks for them (WM_GETMINMAXINFO) at the next resize.
//
// Flutter: widgets/_window.dart RegularWindowController.setConstraints
func (w *window) SetSizeConstraints(min, max ui.Size) {
	w.minSize, w.maxSize = min, max
	w.cfg.MinSize, w.cfg.MaxSize = min, max
}

// SetState asks for the window to be maximized, minimized, full screen or back to normal. A
// window that is hidden or not shown yet takes the state when it is shown.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::SetMaximized, SetMinimized, SetFullscreen
func (w *window) SetState(state platform.WindowState) {
	if w.closed {
		return
	}
	if state == platform.WindowStateUnspecified {
		state = platform.WindowStateNormal
	}
	if !w.shown || w.hidden {
		if w.hidden {
			w.restore = state
		} else {
			w.initial = state
		}
		return
	}
	switch state {
	case platform.WindowStateFullscreen:
		if win32.IsIconic(w.hwnd) {
			win32.ShowWindow(w.hwnd, win32.SWRestore)
		}
		w.enterFullscreen()
	case platform.WindowStateMaximized:
		w.leaveFullscreen()
		win32.ShowWindow(w.hwnd, win32.SWShowMaximized)
	case platform.WindowStateMinimized:
		win32.ShowWindow(w.hwnd, win32.SWMinimize)
	default:
		w.leaveFullscreen()
		win32.ShowWindow(w.hwnd, win32.SWRestore)
	}
}

// enterFullscreen removes the frame (saving the style and the placement), makes the window a
// WS_POPUP and puts it on the rectangle of its monitor at the top of the Z order, which covers
// the task bar.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::SetFullscreen
func (w *window) enterFullscreen() {
	if w.fullscreen() || w.closed {
		return
	}
	placement, err := win32.GetWindowPlacement(w.hwnd)
	if err != nil {
		w.p.log.Warn("the window cannot go full screen", "err", err)
		return
	}
	mi, err := win32.GetMonitorInfo(win32.MonitorFromWindow(w.hwnd, win32.MonitorDefaultToNearest))
	if err != nil {
		w.p.log.Warn("the window cannot go full screen", "err", err)
		return
	}
	w.saved.styles = styleSet{style: uint32(win32.GetWindowLongPtr(w.hwnd, win32.GWLStyle)), exStyle: uint32(win32.GetWindowLongPtr(w.hwnd, win32.GWLExStyle))}
	w.saved.placement = placement
	fs := fullscreenStyle(w.saved.styles)
	fs.style &^= win32.WSMaximize
	w.state.fullscreen = true
	win32.SetWindowLongPtr(w.hwnd, win32.GWLStyle, uintptr(fs.style))
	win32.SetWindowLongPtr(w.hwnd, win32.GWLExStyle, uintptr(fs.exStyle))
	m := mi.Monitor
	if err := win32.SetWindowPos(w.hwnd, 0, m.Left, m.Top, m.Right-m.Left, m.Bottom-m.Top, win32.SWPFrameChanged|win32.SWPNoActivate|win32.SWPShowWindow); err != nil {
		w.p.log.Warn("the window could not be put on its monitor", "err", err)
	}
}

// leaveFullscreen restores the style and the placement that full screen saved.
func (w *window) leaveFullscreen() {
	if !w.fullscreen() || w.closed {
		return
	}
	w.state.fullscreen = false
	rs := restoredStyle(w.saved.styles)
	win32.SetWindowLongPtr(w.hwnd, win32.GWLStyle, uintptr(rs.style))
	win32.SetWindowLongPtr(w.hwnd, win32.GWLExStyle, uintptr(rs.exStyle))
	if err := win32.SetWindowPlacement(w.hwnd, w.saved.placement); err != nil {
		w.p.log.Warn("the window placement was not restored", "err", err)
	}
	if err := win32.SetWindowPos(w.hwnd, 0, 0, 0, 0, 0, win32.SWPNoMove|win32.SWPNoSize|win32.SWPNoZOrder|win32.SWPFrameChanged|win32.SWPNoActivate); err != nil {
		w.p.log.Warn("the frame of the window was not restored", "err", err)
	}
	// The placement can leave the size as it was in full screen; WM_SIZE reports what is.
	if r, err := win32.GetClientRect(w.hwnd); err == nil && !w.state.minimized {
		if cw, ch := r.Right-r.Left, r.Bottom-r.Top; cw > 0 && ch > 0 && (cw != w.clientW || ch != w.clientH) {
			w.clientW, w.clientH = cw, ch
			w.reportMetrics()
			w.renderNow()
		}
	}
}

// SetHidden takes the window off the screen (ShowWindow with SW_HIDE, which also removes the
// task-bar button), or shows it in the state it had and asks for the foreground. A hidden window
// still exists, and asks for no frame; the next frame after the show is presented whole.
//
// Flutter: no counterpart
func (w *window) SetHidden(hidden bool) bool {
	if w.closed || w.hidden == hidden {
		return true
	}
	w.hidden = hidden
	defer w.p.lifecycleChanged()
	if hidden {
		switch {
		case !w.shown:
			w.restore = w.initial
		case w.fullscreen():
			w.restore = platform.WindowStateFullscreen
		case win32.IsZoomed(w.hwnd):
			w.restore = platform.WindowStateMaximized
		case win32.IsIconic(w.hwnd):
			w.restore = platform.WindowStateMinimized
		default:
			w.restore = platform.WindowStateNormal
		}
		win32.ShowWindow(w.hwnd, win32.SWHide)
		if w.focusKnown {
			w.activated = false
			w.reportFocus(false) // a window off the screen has no focus
		}
		return true
	}
	if !w.shown {
		w.hidden = false
		w.p.showNewWindows()
		return true
	}
	switch w.restore {
	case platform.WindowStateMaximized:
		win32.ShowWindow(w.hwnd, win32.SWShowMaximized)
	case platform.WindowStateMinimized:
		win32.ShowWindow(w.hwnd, win32.SWShowMinimized)
	default:
		win32.ShowWindow(w.hwnd, win32.SWShowNormal)
	}
	if h := w.p.handler; h != nil {
		h.OnContentLost(w.id)
	}
	if w.restore != platform.WindowStateMinimized {
		win32.SetForegroundWindow(w.hwnd)
	}
	w.p.scheduleVsync()
	return true
}

// StartDrag fails with DragUnsupported until step 28.12.
//
// Flutter: no counterpart
func (w *window) StartDrag(ui.DragRequest) error {
	return errors.New(tagDragUnsupported, "drag and drop from a window of Windows comes with step 28.12")
}

// PerformWindowAction does what the title bar of the app asks (SPEC §25.4): moves and resizes
// release the capture and hand the pointer to the system's loop with WM_NCLBUTTONDOWN, the window
// menu is the system menu, and minimize, maximize, restore, full screen and close are ShowWindow
// and the close request. A hidden window does only the close.
//
// Flutter: no counterpart
func (w *window) PerformWindowAction(action ui.WindowAction) {
	if w.closed || (w.hidden && action.Kind != ui.WindowActionClose) {
		return
	}
	switch action.Kind {
	case ui.WindowActionMinimize:
		w.SetState(platform.WindowStateMinimized)
	case ui.WindowActionMaximize:
		w.SetState(platform.WindowStateMaximized)
	case ui.WindowActionRestore, ui.WindowActionUnfullscreen:
		w.SetState(platform.WindowStateNormal)
	case ui.WindowActionFullscreen:
		w.SetState(platform.WindowStateFullscreen)
	case ui.WindowActionActivate:
		if win32.IsIconic(w.hwnd) {
			win32.ShowWindow(w.hwnd, win32.SWRestore)
		}
		win32.SetForegroundWindow(w.hwnd)
	case ui.WindowActionClose:
		w.onClose()
	case ui.WindowActionMove:
		w.startSystemLoop(win32.HTCaption)
	case ui.WindowActionResize:
		w.startSystemLoop(edgeHit(action.Edge))
	case ui.WindowActionShowMenu:
		w.showSystemMenu(action.Position)
	}
}

// startSystemLoop hands the pointer to the system's move or size loop for the hit-test area.
func (w *window) startSystemLoop(hit uintptr) {
	if w.state.maximized || w.fullscreen() {
		return
	}
	pt, _ := win32.GetCursorPos()
	win32.ReleaseCapture()
	win32.SendMessage(w.hwnd, win32.WMNCLButtonDown, hit, uintptr(uint32(uint16(pt.X))|uint32(uint16(pt.Y))<<16))
}

// showSystemMenu opens the window menu at a position of the window, in logical pixels.
func (w *window) showSystemMenu(at ui.Offset) {
	menu := win32.GetSystemMenu(w.hwnd)
	if menu == 0 {
		return
	}
	scale := w.scale()
	pt, ok := win32.ClientToScreen(w.hwnd, win32.Point{X: physical(at.Dx, scale), Y: physical(at.Dy, scale)})
	if !ok {
		return
	}
	cmd := win32.TrackPopupMenuEx(menu, win32.TPMReturnCmd, pt.X, pt.Y, w.hwnd)
	if cmd != 0 {
		win32.SendMessage(w.hwnd, win32.WMSysCommand, uintptr(cmd), 0)
	}
}

// Close destroys the window. Destroy the renderer's surface for it first. When it was the last
// window, Run returns. Closing twice does nothing.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::~HostWindow
func (w *window) Close() {
	if w.closed {
		return
	}
	w.stopModalTimer()
	if w.cfg.Modal && w.owner != nil && !w.owner.closed {
		win32.EnableWindow(w.owner.hwnd, true) // before the destroy, or the owner would lose the focus
	}
	if err := win32.DestroyWindow(w.hwnd); err != nil {
		w.p.log.Warn("the window could not be destroyed", "err", err)
		w.onDestroyed()
	}
}

// KeepScreenOn stops the display from going idle while any window asked for it
// (SetThreadExecutionState with ES_CONTINUOUS | ES_DISPLAY_REQUIRED).
//
// Flutter: no counterpart
func (w *window) KeepScreenOn(on bool) error {
	if w.closed {
		return nil
	}
	w.keepOn = on
	w.p.updateExecutionState()
	return nil
}

// SetContentType does nothing: Windows has no content type hint.
//
// Flutter: no counterpart
func (w *window) SetContentType(ui.ContentType) {}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragUnsupported = "DragUnsupported"
)
