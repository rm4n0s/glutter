package wayland

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
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
	w := &window{p: p, id: p.nextView, cfg: cfg}
	p.nextView++

	w.width, w.height = int32(cfg.Size.Width), int32(cfg.Size.Height)
	if w.width <= 0 || w.height <= 0 {
		w.width, w.height = int32(defaultSize.Width), int32(defaultSize.Height)
	}

	w.surface = s.compositor.CreateSurface()
	w.xdgSurface = s.wmBase.GetXdgSurface(w.surface)
	w.toplevel = w.xdgSurface.GetToplevel()
	w.xdgSurface.SetEvents(&wl.XdgSurfaceEvents{OnConfigure: w.onSurfaceConfigure})
	w.toplevel.SetEvents(&wl.XdgToplevelEvents{
		OnConfigure: w.onToplevelConfigure,
		OnClose:     w.onClose,
	})

	w.toplevel.SetTitle(cfg.Title)
	if cfg.AppID != "" {
		w.toplevel.SetAppId(cfg.AppID)
	}
	w.toplevel.SetMinSize(int32(cfg.MinSize.Width), int32(cfg.MinSize.Height))
	w.toplevel.SetMaxSize(int32(cfg.MaxSize.Width), int32(cfg.MaxSize.Height))
	if cfg.HasParent {
		if parent, ok := p.windows[cfg.Parent]; ok {
			w.toplevel.SetParent(parent.toplevel)
		}
	}
	w.requestState(cfg.State)
	w.requestDecorations()

	// The initial commit, without a buffer, asks the compositor for the first
	// configure.
	w.surface.Commit()

	p.windows[w.id] = w
	p.hadWindow = true
	return w, nil
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
		OnConfigure: func(mode wl.ZxdgToplevelDecorationV1Mode) { w.decorationMode = mode },
	})
	mode := wl.ZxdgToplevelDecorationV1ModeServerSide
	if !w.cfg.Decorated {
		mode = wl.ZxdgToplevelDecorationV1ModeClientSide
	}
	w.decoration.SetMode(mode)
}

// requestState asks the compositor for a window state.
func (w *window) requestState(state platform.WindowState) {
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
	w.states = w.pendingStates

	// Draw first, then acknowledge and commit, so no frame of the old size is
	// shown after the ack (SPEC §8.11, "Resizing and scaling").
	buf, err := newShmBuffer(w.p.sess.shm, w.width, w.height, fillColor, w.onBufferReleased)
	if err != nil {
		w.p.fail(err)
		return
	}
	w.xdgSurface.AckConfigure(serial)
	w.configures++
	w.commit(buf)
	if resized || w.configures == 1 {
		w.reportMetrics()
	}
}

// commit puts buf on the surface. The buffer on display until now is freed once
// the compositor has released it.
func (w *window) commit(buf *shmBuffer) {
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
		h.OnMetrics(w.id, platform.ViewMetrics{
			// TODO(step 3.7): the scale of the output; 1 until then.
			PhysicalSize:     ui.Size{Width: float64(w.width), Height: float64(w.height)},
			DevicePixelRatio: 1,
		})
	}
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

// NativeSurface returns nil for now.
//
// TODO(step 1.6): the handle with the wl_display and wl_surface for EGL.
//
// Flutter: no counterpart
func (w *window) NativeSurface() gpu.SurfaceHandle { return nil }

// SetTitle changes the window title.
//
// Flutter: no counterpart
func (w *window) SetTitle(title string) {
	if !w.closed {
		w.toplevel.SetTitle(title)
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
	if !w.mapped {
		return // the first configure draws at this size
	}
	buf, err := newShmBuffer(w.p.sess.shm, width, height, fillColor, w.onBufferReleased)
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

// SetCursor does nothing yet.
//
// TODO(step 1.8): wp_cursor_shape_v1 or the XCursor theme.
//
// Flutter: no counterpart
func (w *window) SetCursor(platform.MouseCursor) {}

// SetTextInput does nothing yet.
//
// TODO(step 11.2): zwp_text_input_v3.
//
// Flutter: no counterpart
func (w *window) SetTextInput(platform.TextInputState) {}

// StartDrag is not implemented yet.
//
// TODO(step 14.6): wl_data_device.start_drag.
//
// Flutter: no counterpart
func (w *window) StartDrag(platform.DragData) error {
	return errors.New("NotImplemented", "drag and drop from a Wayland window is not implemented yet")
}

// Close destroys the window. When it was the last window, Run returns. Closing
// twice does nothing.
//
// Flutter: no counterpart
func (w *window) Close() {
	if w.closed {
		return
	}
	w.closed = true
	if w.decoration != nil {
		w.decoration.Destroy()
	}
	w.toplevel.Destroy()
	w.xdgSurface.Destroy()
	w.surface.Destroy()
	for _, b := range append(w.old, w.buffer) {
		if b != nil {
			b.free()
		}
	}
	w.old, w.buffer = nil, nil
	delete(w.p.windows, w.id)
	if len(w.p.windows) == 0 {
		w.p.loop.Stop()
	}
}
