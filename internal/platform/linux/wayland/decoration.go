package wayland

import (
	"fmt"
	"log/slog"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/decor"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// chooseDecoration picks who draws the title bar of a window, in the order of
// SPEC §8.2: the compositor (zxdg_decoration_manager_v1), then libdecor, then the
// title bar of the framework. A window that asks for none, or for its own, skips
// that order.
func (p *Platform) chooseDecoration(cfg platform.WindowConfig) ui.DecorationMode {
	switch {
	case !cfg.IsDecorated() || cfg.Kind == platform.WindowKindBorderless:
		return ui.DecorationNone
	case cfg.Decorations == platform.DecorationsApp:
		return ui.DecorationApp
	case cfg.Decorations == platform.DecorationsLibdecor && p.libdecorContext() != nil:
		return ui.DecorationLibdecor
	case p.sess.decorationManager != nil:
		return ui.DecorationServerSide
	case p.libdecorContext() != nil:
		return ui.DecorationLibdecor
	}
	return ui.DecorationApp
}

// libdecorContext returns the libdecor context, making it on the first call. It
// is nil when libdecor is missing or does not start, which is logged once; the
// windows then draw their own title bar.
func (p *Platform) libdecorContext() *decor.Context {
	if p.libdecor != nil || p.libdecorTried || p.opts.NoLibdecor {
		return p.libdecor
	}
	p.libdecorTried = true
	ctx, err := decor.New(p.sess.conn.DisplayPtr(), func(code decor.Error, message string) {
		slog.Default().Error("libdecor", "code", int(code), "message", message)
	})
	if err != nil {
		slog.Default().Info("libdecor is not available; windows draw their own title bar", "error", fmt.Sprint(err))
		return nil
	}
	p.libdecor = ctx
	if fd := ctx.FD(); fd >= 0 && fd != p.src.FD() {
		if err := p.loop.AddSource(&libdecorSource{ctx: ctx, fd: fd}); err != nil {
			slog.Default().Error("libdecor: the loop did not take its file descriptor", "error", fmt.Sprint(err))
		}
	}
	return ctx
}

// libdecorSource plugs the file descriptor of libdecor into the event loop.
// libdecor reads the connection through its own queue; the plugin of GTK also
// has a connection of its own, which this descriptor stands for.
type libdecorSource struct {
	ctx *decor.Context
	fd  int
}

var _ linux.Source = (*libdecorSource)(nil)

func (s *libdecorSource) FD() int                        { return s.fd }
func (s *libdecorSource) WantsWrite() bool               { return false }
func (s *libdecorSource) BeforeWait() error              { return nil }
func (s *libdecorSource) AfterWait(_ linux.Events) error { return s.ctx.Dispatch() }

// decorateWithLibdecor makes the frame of the window: libdecor makes the xdg_surface
// and the xdg_toplevel, draws the title bar, and says when to configure.
func (w *window) decorateWithLibdecor(cfg platform.WindowConfig) error {
	ctx := w.p.libdecor
	frame, err := ctx.Decorate(w.surface.Ptr(), libdecorHandler{w})
	if err != nil {
		return err
	}
	w.frame = frame
	w.frame.SetTitle(cfg.Title)
	w.title = cfg.Title
	if cfg.AppID != "" {
		w.frame.SetAppID(cfg.AppID)
	}
	w.frame.SetMinContentSize(int(cfg.MinSize.Width), int(cfg.MinSize.Height))
	w.frame.SetMaxContentSize(int(cfg.MaxSize.Width), int(cfg.MaxSize.Height))
	if !cfg.IsResizable() {
		w.frame.UnsetCapabilities(decor.CanResize)
	}
	if cfg.HasParent {
		if parent, ok := w.p.windows[cfg.Parent]; ok && parent.frame != nil {
			w.frame.SetParent(parent.frame)
		}
	}
	if top := w.frame.Toplevel(); top != 0 {
		w.makeModal(w.p.sess.conn.WrapXdgToplevel(top))
	}
	w.requestState(cfg.State)
	if err := w.SetIcon(cfg.Icon); err != nil {
		return err
	}
	w.frame.Map() // the first configure follows
	return nil
}

// libdecorHandler gives the events of a libdecor frame to its window.
type libdecorHandler struct{ w *window }

func (h libdecorHandler) Configure(c decor.Configuration) {
	w := h.w
	if w.closed {
		return
	}
	w.pendingWidth, w.pendingHeight = 0, 0
	if width, height, ok := c.ContentSize(); ok {
		w.pendingWidth, w.pendingHeight = int32(width), int32(height)
	}
	w.pendingStates = WindowStates{}
	if st, ok := c.WindowState(); ok {
		w.pendingStates = statesFromLibdecor(st)
	} else {
		w.pendingStates = w.states
	}
	w.conf = c
	w.onSurfaceConfigure(0)
}

func (h libdecorHandler) Close() { h.w.onClose() }

func (h libdecorHandler) Commit() {
	if !h.w.closed {
		h.w.surface.Commit()
	}
}

func (h libdecorHandler) DismissPopup(string) {}

// statesFromLibdecor reads the window state of a libdecor configure.
func statesFromLibdecor(st decor.State) WindowStates {
	return WindowStates{
		Maximized:   st&decor.StateMaximized != 0,
		Fullscreen:  st&decor.StateFullscreen != 0,
		Activated:   st&decor.StateActive != 0,
		TiledLeft:   st&decor.StateTiledLeft != 0,
		TiledRight:  st&decor.StateTiledRight != 0,
		TiledTop:    st&decor.StateTiledTop != 0,
		TiledBottom: st&decor.StateTiledBottom != 0,
		Suspended:   st&decor.StateSuspended != 0,
	}
}

// ackConfigure answers the configure that is being applied: an xdg_surface
// acknowledges it, and a libdecor frame commits the content size.
func (w *window) ackConfigure(serial uint32) {
	if w.frame != nil {
		conf := w.conf
		w.conf = decor.Configuration{}
		w.frame.Commit(int(w.width), int(w.height), conf)
		return
	}
	w.xdgSurface.AckConfigure(serial)
}

// onWmCapabilities stores what the compositor lets the user do with the window.
func (w *window) onWmCapabilities(capabilities []uint32) {
	var caps ui.WindowCapabilities
	for _, c := range capabilities {
		switch wl.XdgToplevelWmCapabilities(c) {
		case wl.XdgToplevelWmCapabilitiesWindowMenu:
			caps.WindowMenu = true
		case wl.XdgToplevelWmCapabilitiesMaximize:
			caps.Maximize = true
		case wl.XdgToplevelWmCapabilitiesFullscreen:
			caps.Fullscreen = true
		case wl.XdgToplevelWmCapabilitiesMinimize:
			caps.Minimize = true
		}
	}
	w.caps = caps
	if w.configures > 0 {
		w.reportMetrics()
	}
}

// onDecorationMode runs when the compositor answers the request for a decoration
// mode. A compositor that makes the window draw its own (client side) when it was
// asked for server-side ones leaves the app to draw the title bar.
func (w *window) onDecorationMode(mode wl.ZxdgToplevelDecorationV1Mode) {
	w.decorationMode = mode
	if w.deco == ui.DecorationServerSide && mode == wl.ZxdgToplevelDecorationV1ModeClientSide {
		w.deco = ui.DecorationApp
		if w.configures > 0 {
			w.reportMetrics()
		}
	}
}

// info is what the app is told about the decorations and the state of the window.
func (w *window) info() ui.WindowInfo {
	return ui.WindowInfo{
		Decoration:   w.deco,
		Capabilities: w.caps,
		Title:        w.title,
		Maximized:    w.states.Maximized,
		Fullscreen:   w.states.Fullscreen,
		Activated:    w.states.Activated,
		Tiled:        w.states.TiledLeft || w.states.TiledRight || w.states.TiledTop || w.states.TiledBottom,
	}
}

// PerformWindowAction does what the title bar of the app asks. The moves, the
// resizes and the menu are given the seat and the serial of the pointer press that
// started them.
//
// Flutter: no counterpart
func (w *window) PerformWindowAction(action ui.WindowAction) {
	if w.closed {
		return
	}
	seat, serial := w.p.sess.seat, w.p.pointerSerial
	switch action.Kind {
	case ui.WindowActionMove:
		if seat == nil {
			return
		}
		if w.frame != nil {
			w.frame.Move(seat.Ptr(), serial)
		} else {
			w.toplevel.Move(seat, serial)
		}
	case ui.WindowActionResize:
		if seat == nil {
			return
		}
		if w.frame != nil {
			w.frame.Resize(seat.Ptr(), serial, libdecorEdge(action.Edge))
		} else {
			w.toplevel.Resize(seat, serial, xdgEdge(action.Edge))
		}
	case ui.WindowActionShowMenu:
		if seat == nil || !w.caps.WindowMenu {
			return
		}
		x, y := int32(action.Position.Dx), int32(action.Position.Dy)
		if w.frame != nil {
			w.frame.ShowWindowMenu(seat.Ptr(), serial, int(x), int(y))
		} else {
			w.toplevel.ShowWindowMenu(seat, serial, x, y)
		}
	case ui.WindowActionMinimize:
		w.requestState(platform.WindowStateMinimized)
	case ui.WindowActionMaximize:
		if w.frame != nil {
			w.frame.SetMaximized()
		} else {
			w.toplevel.SetMaximized()
		}
	case ui.WindowActionRestore:
		if w.frame != nil {
			w.frame.UnsetMaximized()
		} else {
			w.toplevel.UnsetMaximized()
		}
	case ui.WindowActionClose:
		w.onClose()
	case ui.WindowActionFullscreen:
		w.requestState(platform.WindowStateFullscreen)
	case ui.WindowActionUnfullscreen:
		if w.frame != nil {
			w.frame.UnsetFullscreen()
		} else {
			w.toplevel.UnsetFullscreen()
		}
	case ui.WindowActionActivate:
		// xdg-shell cannot raise a window or take the focus for it; xdg_activation_v1
		// can, with a token. Without the protocol a window that is minimized comes
		// back only through the compositor's own shell.
		w.activate()
	}
}

// xdgEdge is the xdg_toplevel resize edge for an edge of the window.
func xdgEdge(e ui.ResizeEdge) wl.XdgToplevelResizeEdge {
	switch e {
	case ui.ResizeEdgeTop:
		return wl.XdgToplevelResizeEdgeTop
	case ui.ResizeEdgeBottom:
		return wl.XdgToplevelResizeEdgeBottom
	case ui.ResizeEdgeLeft:
		return wl.XdgToplevelResizeEdgeLeft
	case ui.ResizeEdgeRight:
		return wl.XdgToplevelResizeEdgeRight
	case ui.ResizeEdgeTopLeft:
		return wl.XdgToplevelResizeEdgeTopLeft
	case ui.ResizeEdgeTopRight:
		return wl.XdgToplevelResizeEdgeTopRight
	case ui.ResizeEdgeBottomLeft:
		return wl.XdgToplevelResizeEdgeBottomLeft
	case ui.ResizeEdgeBottomRight:
		return wl.XdgToplevelResizeEdgeBottomRight
	}
	return wl.XdgToplevelResizeEdgeNone
}

// libdecorEdge is the libdecor resize edge for an edge of the window.
func libdecorEdge(e ui.ResizeEdge) decor.ResizeEdge {
	switch e {
	case ui.ResizeEdgeTop:
		return decor.ResizeTop
	case ui.ResizeEdgeBottom:
		return decor.ResizeBottom
	case ui.ResizeEdgeLeft:
		return decor.ResizeLeft
	case ui.ResizeEdgeRight:
		return decor.ResizeRight
	case ui.ResizeEdgeTopLeft:
		return decor.ResizeTopLeft
	case ui.ResizeEdgeTopRight:
		return decor.ResizeTopRight
	case ui.ResizeEdgeBottomLeft:
		return decor.ResizeBottomLeft
	case ui.ResizeEdgeBottomRight:
		return decor.ResizeBottomRight
	}
	return decor.ResizeNone
}
