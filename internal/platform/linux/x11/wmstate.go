package x11

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
	"strconv"
)

// iconicState is the value of WM_STATE for a window that is iconified (ICCCM 4.1.3.1).
const iconicState = 3

// The directions of _NET_WM_MOVERESIZE (EWMH): the eight edges and corners, and the
// move.
const (
	moveResizeTopLeft     = 0
	moveResizeTop         = 1
	moveResizeTopRight    = 2
	moveResizeRight       = 3
	moveResizeBottomRight = 4
	moveResizeBottom      = 5
	moveResizeBottomLeft  = 6
	moveResizeLeft        = 7
	moveResizeMove        = 8
)

// windowState is what the window manager says about a window: the _NET_WM_STATE
// and WM_STATE properties, the _NET_WM_ALLOWED_ACTIONS, and whether it has the focus
// (SPEC §8.3).
type windowState struct {
	maximized, fullscreen, minimized, activated bool
	// allowed are the actions the window manager lets the user do, and hasAllowed
	// says it listed them; without a list the window has every one.
	allowed    ui.WindowCapabilities
	hasAllowed bool
}

func containsAtom(list []uint64, atom uint64) bool {
	for _, a := range list {
		if a == atom {
			return true
		}
	}
	return false
}

// refreshState reads what the window manager says about the window and keeps it.
func (w *window) refreshState() {
	p, at := w.p, &w.p.atoms
	state := p.readLongs(w.xid, at.netWMState, xaAtom, 64)
	w.state.maximized = containsAtom(state, at.netWMStateMaxVert) && containsAtom(state, at.netWMStateMaxHorz)
	w.state.fullscreen = containsAtom(state, at.netWMStateFullscreen)
	// A window is minimized when the window manager iconified it: WM_STATE says so
	// (ICCCM), and _NET_WM_STATE_HIDDEN does for managers that do not set it.
	wmState := p.readLongs(w.xid, at.wmState, at.wmState, 2)
	w.state.minimized = (len(wmState) > 0 && wmState[0] == iconicState) || containsAtom(state, at.netWMStateHidden)
	actions := p.readLongs(w.xid, at.netWMAllowedActions, xaAtom, 64)
	w.state.hasAllowed = actions != nil
	if w.state.hasAllowed {
		w.state.allowed = ui.WindowCapabilities{
			WindowMenu: true,
			Maximize:   containsAtom(actions, at.netWMActionMaximizeHorz) && containsAtom(actions, at.netWMActionMaximizeVert),
			Fullscreen: containsAtom(actions, at.netWMActionFullscreen),
			Minimize:   containsAtom(actions, at.netWMActionMinimize),
		}
	}
}

// onStateProperty handles a change of one of the properties that the window
// manager keeps: the state is read again, and the app hears of what changed.
func (w *window) onStateProperty(atom uint64) {
	at := &w.p.atoms
	if atom != at.netWMState && atom != at.wmState && atom != at.netWMAllowedActions {
		return
	}
	w.changeState(w.refreshState)
}

// changeState runs a change of the window's state and tells the app, once the
// window is visible, when what it is told changed.
func (w *window) changeState(change func()) {
	before := w.info()
	change()
	if w.mapped && !w.closed && w.info() != before {
		w.reportMetrics()
	}
}

// setActivated follows the keyboard focus of the window.
func (w *window) setActivated(activated bool) {
	w.changeState(func() { w.state.activated = activated })
}

// info is what the app is told about the decorations and the state of the window.
// The window manager draws the title bar of a decorated window unless the app asked
// to draw it (DecorationsApp: the manager is told to leave the window bare, and
// the moves and resizes go through _NET_WM_MOVERESIZE).
func (w *window) info() ui.WindowInfo {
	mode := ui.DecorationServerSide
	switch {
	case !w.cfg.IsDecorated() || w.cfg.Kind == platform.WindowKindBorderless:
		mode = ui.DecorationNone
	case w.cfg.Decorations == platform.DecorationsApp:
		mode = ui.DecorationApp
	}
	caps := ui.AllWindowCapabilities
	if w.state.hasAllowed {
		caps = w.state.allowed
	}
	return ui.WindowInfo{
		Decoration: mode, Capabilities: caps, Title: w.title,
		Maximized: w.state.maximized, Fullscreen: w.state.fullscreen, Minimized: w.state.minimized,
		Activated: w.state.activated,
	}
}

// SetIcon sets the icon of the window: _NET_WM_ICON, a list of pictures, each as
// its width, its height and its pixels as ARGB values. The icon theme name has no
// X11 counterpart (the window manager finds the icon of the app by WM_CLASS in its
// .desktop file) and is ignored. An empty icon removes the property. It fails with
// WindowIconInvalid when a picture is wrong.
//
// Flutter: no counterpart
func (w *window) SetIcon(icon ui.WindowIcon) error {
	if err := icon.Validate(); err != nil {
		return err
	}
	if w.closed {
		return nil
	}
	p := w.p
	if len(icon.Images) == 0 {
		p.x.deleteProperty(p.display, w.xid, p.atoms.netWMIcon)
		p.x.flush(p.display)
		return nil
	}
	var data []uint64
	for _, img := range icon.Images {
		data = append(data, uint64(img.Size), uint64(img.Size))
		for i := 0; i+3 < len(img.RGBA); i += 4 {
			r, g, b, a := uint64(img.RGBA[i]), uint64(img.RGBA[i+1]), uint64(img.RGBA[i+2]), uint64(img.RGBA[i+3])
			data = append(data, a<<24|r<<16|g<<8|b)
		}
	}
	p.setLongs(w.xid, p.atoms.netWMIcon, xaCardinal, data)
	p.x.flush(p.display)
	return nil
}

// edgeDirection maps a resize edge to the direction of _NET_WM_MOVERESIZE.
func edgeDirection(e ui.ResizeEdge) int64 {
	switch e {
	case ui.ResizeEdgeTop:
		return moveResizeTop
	case ui.ResizeEdgeBottom:
		return moveResizeBottom
	case ui.ResizeEdgeLeft:
		return moveResizeLeft
	case ui.ResizeEdgeRight:
		return moveResizeRight
	case ui.ResizeEdgeTopLeft:
		return moveResizeTopLeft
	case ui.ResizeEdgeTopRight:
		return moveResizeTopRight
	case ui.ResizeEdgeBottomLeft:
		return moveResizeBottomLeft
	case ui.ResizeEdgeBottomRight:
		return moveResizeBottomRight
	}
	return moveResizeMove
}

// moveResize asks the window manager to move the window or to resize it from an
// edge (_NET_WM_MOVERESIZE) with the pointer button that is down; the manager takes
// the pointer from the app, which lets go of its own grab first. Without a button
// that is down it does nothing.
func (w *window) moveResize(direction int64) {
	in := w.p.input
	if in == nil || !in.press.valid || in.buttonsDown&(1<<uint(in.press.button)) == 0 {
		return
	}
	in.ungrabPointer()
	w.sendRootMessage(w.p.atoms.netWMMoveResize, int64(in.press.rootX), int64(in.press.rootY), direction, int64(in.press.button), 1)
}

// showMenu asks the window manager for the menu of the window at a position of
// the window, in logical pixels (_GTK_SHOW_WINDOW_MENU, which GNOME and KDE
// answer).
func (w *window) showMenu(at ui.Offset) {
	in := w.p.input
	if in == nil || !in.press.valid {
		return
	}
	ox, oy := w.rootOrigin()
	in.ungrabPointer()
	w.sendRootMessage(w.p.atoms.gtkShowWindowMenu, int64(in.press.device), int64(ox)+int64(at.Dx*w.scale), int64(oy)+int64(at.Dy*w.scale), 0, 0)
}

// sendRootMessage sends the window manager a client message about the window, in
// the way an application does (to the root window, for the manager to redirect).
func (w *window) sendRootMessage(typ uint64, data ...int64) {
	p := w.p
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	ev := view[xClientMessageEvent](mem)
	ev.typ = clientMessage
	ev.window = w.xid
	ev.messageType = typ
	ev.format = 32
	copy(ev.data[:], data)
	p.x.sendEvent(p.display, p.root, 0, substructureNotifyMask|substructureRedirMask, mem)
	p.x.flush(p.display)
}

var _ platform.PortalHandleProvider = (*window)(nil)

// PortalHandle gives done "x11:" and the window ID in hex, which a portal takes as
// the parent of its dialog. It is called at once.
//
// Flutter: no counterpart
func (w *window) PortalHandle(done func(handle string)) {
	if w.closed {
		done("")
		return
	}
	done("x11:" + strconv.FormatUint(w.xid, 16))
}
