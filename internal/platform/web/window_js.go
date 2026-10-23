//go:build js && wasm

package web

import (
	"strconv"
	"syscall/js"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// window is the one window of the page: a canvas in the host element.
//
// Flutter: lib/web_ui/lib/src/engine/window.dart EngineFlutterView
type window struct {
	p      *Platform
	id     ui.ViewID
	canvas js.Value
	// geo is the size of the view and the device pixel ratio, as last reported.
	geo Geometry
	// frameWanted is whether a frame was asked for and has not come; rafPending is
	// whether a requestAnimationFrame is waiting for it.
	frameWanted, rafPending bool
	closed                  bool
}

var (
	_ platform.Window                = (*window)(nil)
	_ platform.ContextMenuController = (*window)(nil)
	_ platform.SurfaceRenewer        = (*window)(nil)
)

// newWindow puts a canvas into the host element and measures it. For the full page the
// body loses its margin and scroll bars and the canvas covers the viewport; the canvas
// has touch-action none so that the browser does not take touches for itself (SPEC §23.2).
func newWindow(p *Platform, id ui.ViewID, cfg platform.WindowConfig) *window {
	doc := global().Get("document")
	if p.fullPage {
		html := doc.Get("documentElement")
		setStyle(html, "height", "100%")
		for _, name := range []string{"margin", "padding", "top", "right", "bottom", "left"} {
			setStyle(p.host, name, "0")
		}
		setStyle(p.host, "position", "fixed")
		setStyle(p.host, "overflow", "hidden")
		setStyle(p.host, "touch-action", "none")
	}
	canvas := p.makeCanvas(id)
	p.host.Call("appendChild", canvas)
	w := &window{p: p, id: id, canvas: canvas}
	rect := canvas.Call("getBoundingClientRect")
	w.geo = Geometry{CSSWidth: number(rect.Get("width"), 0), CSSHeight: number(rect.Get("height"), 0), DPR: p.dpr()}
	pw, ph := w.geo.PhysicalSize()
	canvas.Set("width", pw)
	canvas.Set("height", ph)
	if cfg.Title != "" {
		doc.Set("title", cfg.Title)
	}
	if cfg.State == platform.WindowStateFullscreen {
		w.SetState(cfg.State)
	}
	return w
}

// makeCanvas makes the canvas element of a view, not yet in the page: the styles of
// SPEC §23.2 (it fills the host, or the viewport for the full page, and has touch-action none).
func (p *Platform) makeCanvas(id ui.ViewID) js.Value {
	canvas := global().Get("document").Call("createElement", "canvas")
	canvas.Call("setAttribute", ViewAttribute, strconv.Itoa(int(id)))
	setStyle(canvas, "display", "block")
	setStyle(canvas, "width", "100%")
	setStyle(canvas, "height", "100%")
	setStyle(canvas, "touch-action", "none")
	setStyle(canvas, "outline", "none")
	setStyle(canvas, "user-select", "none")
	setStyle(canvas, "-webkit-user-select", "none")
	setStyle(canvas, "-webkit-tap-highlight-color", "transparent")
	if p.fullPage {
		setStyle(canvas, "position", "absolute")
		for _, name := range []string{"top", "right", "bottom", "left"} {
			setStyle(canvas, name, "0")
		}
	}
	return canvas
}

// RenewSurface puts a new canvas in the place of the window's, with the same size, cursor and
// listeners, and takes the old one out of the page. A canvas keeps the first kind of context it
// was asked for, so the engine calls it before it tries a renderer after another one failed on
// the canvas (SPEC §9.6). The next frame is drawn into the new canvas, which is blank until then.
//
// Flutter: no counterpart
func (w *window) RenewSurface() {
	if w.closed {
		return
	}
	p := w.p
	old := w.canvas
	canvas := p.makeCanvas(w.id)
	canvas.Set("width", old.Get("width"))
	canvas.Set("height", old.Get("height"))
	if cursor := old.Get("style").Call("getPropertyValue", "cursor").String(); cursor != "" {
		setStyle(canvas, "cursor", cursor)
	}
	// The listeners that were on the old canvas move to the new one, and the observer
	// measures it.
	for i := range p.in.active {
		l := &p.in.active[i]
		if !l.target.Equal(old) {
			continue
		}
		l.target.Call("removeEventListener", l.event, l.fn)
		l.target = canvas
		if l.options != nil {
			canvas.Call("addEventListener", l.event, l.fn, l.options)
		} else {
			canvas.Call("addEventListener", l.event, l.fn)
		}
	}
	if p.in.menuOff {
		old.Call("removeEventListener", "contextmenu", p.in.preventFn)
		canvas.Call("addEventListener", "contextmenu", p.in.preventFn)
	}
	if isObject(p.observer) {
		p.observer.Call("unobserve", old)
		p.observeElement(canvas)
	}
	old.Call("replaceWith", canvas)
	w.canvas = canvas
}

// ID returns the ID of the implicit view.
//
// Flutter: lib/web_ui/lib/src/engine/window.dart EngineFlutterView.viewId
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface names the canvas: the renderer finds it through the view ID.
//
// Flutter: no counterpart
func (w *window) NativeSurface() gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: uintptr(w.id)}
}

// SetTitle sets document.title.
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart EnginePlatformDispatcher.setApplicationSwitcherDescription
func (w *window) SetTitle(title string) { global().Get("document").Set("title", title) }

// SetSize does nothing: the size of the view is the size of its host element (SPEC §23.2).
//
// Flutter: no counterpart
func (w *window) SetSize(ui.Size) {}

// SetSizeConstraints does nothing (SPEC §23.2).
//
// Flutter: no counterpart
func (w *window) SetSizeConstraints(_, _ ui.Size) {}

// SetState makes the host fullscreen or leaves fullscreen; the browser allows
// requestFullscreen only during a user action, and a refusal is ignored. Every other
// state is ignored (SPEC §23.2).
//
// Flutter: no counterpart
func (w *window) SetState(state platform.WindowState) {
	doc := global().Get("document")
	switch state {
	case platform.WindowStateFullscreen:
		if isObject(doc.Get("fullscreenElement")) {
			return
		}
		try(func() {
			if r := w.p.host.Call("requestFullscreen"); isObject(r) {
				r.Call("catch", w.p.noopFn)
			}
		})
	case platform.WindowStateNormal, platform.WindowStateMaximized:
		if isObject(doc.Get("fullscreenElement")) {
			try(func() {
				if r := doc.Call("exitFullscreen"); isObject(r) {
					r.Call("catch", w.p.noopFn)
				}
			})
		}
	}
}

// SetCursor sets the CSS cursor of the canvas to the cursor, mapped as Flutter web maps it; the
// arrow removes the property, so that the page's style decides.
//
// Deviation: Flutter sets the cursor of the page's body, because the framework does not say
// which view; the canvas is the view here, so a host element in a bigger page keeps the
// cursor of the rest of the page (SPEC §12.10).
//
// Flutter: lib/web_ui/lib/src/engine/mouse/cursor.dart MouseCursor.activateSystemCursor
func (w *window) SetCursor(c platform.MouseCursor) {
	if css := CursorCSS(c); css == defaultCursor {
		w.canvas.Get("style").Call("removeProperty", "cursor")
	} else {
		setStyle(w.canvas, "cursor", css)
	}
}

// SetBrowserContextMenuEnabled shows or hides the browser's context menu over the canvas: while
// it is hidden, a listener prevents the default action of the contextmenu event.
//
// Flutter: lib/web_ui/lib/src/engine/mouse/context_menu.dart ContextMenu
func (w *window) SetBrowserContextMenuEnabled(enabled bool) {
	in := &w.p.in
	if enabled == !in.menuOff {
		return
	}
	in.menuOff = !enabled
	if enabled {
		w.canvas.Call("removeEventListener", "contextmenu", in.preventFn)
	} else {
		w.canvas.Call("addEventListener", "contextmenu", in.preventFn)
	}
}

// SetTextInput makes, changes, places and fills the hidden text field, or takes it away: the browser
// types, composes and pastes into it (SPEC §23.5).
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart HybridTextEditing.acceptCommand
func (w *window) SetTextInput(state ui.TextInputState) { w.p.text.set(state) }

// RequestFrame asks for one requestAnimationFrame; requests while one is waiting are
// merged into it. Before Run starts, the request is kept and Run schedules it (SPEC §23.3).
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart EnginePlatformDispatcher.scheduleFrame
func (w *window) RequestFrame() {
	w.frameWanted = true
	if w.rafPending || !w.p.running || w.closed {
		return
	}
	w.scheduleFrame()
}

// scheduleFrame asks the browser for the next animation frame.
func (w *window) scheduleFrame() {
	w.rafPending = true
	global().Call("requestAnimationFrame", w.p.rafFunc)
}

// StartDrag fails with DragUnsupported: drag and drop is not in 1.1.
//
// Flutter: no counterpart
func (w *window) StartDrag(ui.DragRequest) error { return errDragUnsupported() }

// PerformWindowAction ignores the action: a page cannot move, resize or close itself.
//
// Flutter: no counterpart
func (w *window) PerformWindowAction(ui.WindowAction) {}

// SetHidden ignores the request and returns false: the canvas is the only view of
// the page, and a page cannot take itself off the screen (SPEC §23.2). Whether the
// page is shown is the browser's, and the lifecycle follows it.
//
// Flutter: no counterpart
func (w *window) SetHidden(bool) bool { return false }

// Close takes the canvas out of the page.
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/flutter_view_manager.dart FlutterViewManager.unregisterView
func (w *window) Close() {
	if w.closed {
		return
	}
	w.closed = true
	w.p.text.disable()
	w.canvas.Call("remove")
}

// SetIcon does nothing: the icon of a page is its favicon.
//
// Flutter: no counterpart
func (w *window) SetIcon(ui.WindowIcon) error { return nil }

// SetContentType does nothing.
//
// Flutter: no counterpart
func (w *window) SetContentType(ui.ContentType) {}

// KeepScreenOn fails with IdleInhibitUnavailable when asked to keep the screen on.
//
// Flutter: no counterpart
func (w *window) KeepScreenOn(on bool) error {
	if on {
		return errScreenOnUnsupported()
	}
	return nil
}
