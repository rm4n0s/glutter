//go:build js && wasm

package gles

import (
	"strconv"
	"syscall/js"

	"github.com/rm4n0s/glutter/internal/gpu"
)

// viewAttribute names the view a canvas shows; the web platform sets it
// (internal/platform/web ViewAttribute). It is repeated here because a GPU backend does not
// import the platform.
const viewAttribute = "data-glutter-view"

// webEGL is the part of EGL that a WebGL context stands in for. The renderer's context code
// (context.go) opens a display, a context and a surface, makes them current, swaps and closes
// them through eglAPI; on the web the canvas is all of them at once, so every call here
// succeeds trivially and the real work is done in openWeb, which makes the context from the
// canvas and reports what fails with the tags of the web. What WebGL does differently from
// EGL:
//   - there is no swap: the browser presents the canvas when the task ends, and a frame that
//     is read back is read before it, as on EGL
//   - there is no damage: the whole canvas is presented, so a frame is drawn whole and
//     CanSwapWithDamage is false
//   - the canvas has a size of its own that setSurfaceSize sets (a surface follows its window
//     on EGL)
//   - the browser can take the context away and give it back (lossy)
//
// Flutter: no counterpart
type webEGL struct {
	view   uintptr
	canvas js.Value
	gl     js.Value
	// surface is the handle of the window surface, 0 while there is none. A canvas has one
	// context and so one surface: a second window is a second canvas, and a second context.
	surface       uintptr
	width, height int
	*lossState
	lostFn, restoredFn js.Func
	// notify is told that the browser gave the context back.
	notify func()
	closed bool
}

var (
	_ eglAPI       = (*webEGL)(nil)
	_ lossy        = (*webEGL)(nil)
	_ surfaceSizer = (*webEGL)(nil)
)

// openWeb makes the WebGL 2 context of the canvas that h names and opens the renderer's context
// on it. The canvas must be new, or have had no kind of context asked of it: it keeps the first
// kind (SPEC §9.6). It fails with
//
//   - WebGLSurfaceKindUnsupported when h does not name a canvas
//   - WebGLCanvasNotFound when the page has no canvas for the view
//   - WebGL2Unavailable when the canvas gives no WebGL 2 context
//   - SymbolMissing when the context lacks a function of WebGL 2
//   - GLES3Unsupported when the context does not report WebGL 2
//
// onRestored may be nil.
//
// Flutter: no counterpart
func openWeb(h gpu.SurfaceHandle, onRestored func()) (*Context, *Surface, error) {
	if h.Kind != gpu.SurfaceKindWeb {
		return nil, nil, errWebSurfaceKind(string(h.Kind))
	}
	doc := js.Global().Get("document")
	canvas := doc.Call("querySelector", "canvas["+viewAttribute+`="`+strconv.FormatUint(uint64(h.Window), 10)+`"]`)
	if !canvas.Truthy() {
		return nil, nil, errWebCanvasNotFound(h.Window)
	}
	gl, reason := webContext(canvas)
	if !gl.Truthy() {
		return nil, nil, errWebGL2Unavailable(h.Window, reason)
	}
	e := &webEGL{view: h.Window, canvas: canvas, gl: gl, lossState: &lossState{}, notify: onRestored}
	e.lostFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		// preventDefault is what makes the browser restore the context later, and it works
		// only while the event is being dispatched.
		args[0].Call("preventDefault")
		e.onLost()
		return nil
	})
	e.restoredFn = js.FuncOf(func(js.Value, []js.Value) any {
		e.restoredEvent()
		return nil
	})
	canvas.Call("addEventListener", "webglcontextlost", e.lostFn)
	canvas.Call("addEventListener", "webglcontextrestored", e.restoredFn)
	return openWith(e, func(eglAPI) (glAPI, error) {
		g, err := loadWebGL(e.gl)
		if err != nil {
			return nil, err
		}
		return g, nil
	}, h)
}

// restoredEvent records the restoration and tells the renderer's owner.
func (e *webEGL) restoredEvent() {
	e.onRestored()
	if e.notify != nil {
		e.notify()
	}
}

// webContext asks the canvas for a WebGL 2 context. It returns the null value and the reason
// when there is none: getContext answers null for a browser without WebGL 2 and for a canvas
// that has another kind of context, and may throw.
func webContext(canvas js.Value) (gl js.Value, reason string) {
	defer func() {
		if r := recover(); r != nil {
			gl, reason = js.Null(), "getContext threw: "+toString(r)
		}
	}()
	gl = canvas.Call("getContext", "webgl2", webContextAttributes())
	if !gl.Truthy() {
		return js.Null(), "getContext(\"webgl2\") returned null: no WebGL 2 in this browser, or the canvas has a context of another kind"
	}
	if gl.Call("isContextLost").Bool() {
		return js.Null(), "the context was lost as it was made"
	}
	return gl, ""
}

func toString(v any) string {
	if err, ok := v.(error); ok {
		return err.Error()
	}
	if s, ok := v.(string); ok {
		return s
	}
	return "unknown"
}

// NewRenderer returns a renderer that has no GL context yet: the first Attach makes it, from the
// canvas of the handle (see openWeb for its failures). Passing more than one RendererOpts
// panics with MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(opts ...RendererOpts) *Renderer {
	var restored func()
	if len(opts) == 1 {
		restored = opts[0].OnContextRestored
	}
	return newRenderer(func(h gpu.SurfaceHandle) (*Context, *Surface, error) {
		return openWeb(h, restored)
	}, opts...)
}

func (e *webEGL) getPlatformDisplay(uint32, uintptr) uintptr { return 1 }
func (e *webEGL) initialize(uintptr) (int32, int32, bool)    { return 1, 5, true }
func (e *webEGL) chooseConfig(uintptr, []int32) (uintptr, bool) {
	return 1, true
}
func (e *webEGL) bindAPI(uint32) bool                                 { return true }
func (e *webEGL) createContext(uintptr, uintptr, []int32) uintptr     { return 1 }
func (e *webEGL) getConfigAttrib(_, _ uintptr, _ int32) (int32, bool) { return 0, false }

// createWindowSurface gives the canvas's one surface, named by the view; a second one, for
// another view, fails.
func (e *webEGL) createWindowSurface(_, _, native uintptr, _ bool) uintptr {
	if e.surface != 0 && native != e.view {
		return 0
	}
	e.surface = e.view + 1
	return e.surface
}

// contextLost says whether the context is lost: the browser's event said so, or the context itself
// does (loseContext takes effect at once, and its event comes in a task of its own).
func (e *webEGL) contextLost() bool {
	return e.lossState.contextLost() || e.gl.Call("isContextLost").Bool()
}

// makeCurrent says whether the context can be used: a lost one cannot, except to let go of it.
func (e *webEGL) makeCurrent(_, _, _, context uintptr) bool {
	return context == 0 || !e.contextLost()
}
func (e *webEGL) swapInterval(uintptr, int32) bool { return true }

// swapBuffers does nothing: the browser presents the canvas when the task ends.
func (e *webEGL) swapBuffers(uintptr, uintptr) bool                  { return true }
func (e *webEGL) canSwapWithDamage(uintptr) bool                     { return false }
func (e *webEGL) swapBuffersWithDamage(_, _ uintptr, _ []int32) bool { return false }
func (e *webEGL) destroySurface(_, surface uintptr) bool {
	if surface == e.surface {
		e.surface = 0
	}
	return true
}

// destroyContext stops listening and gives the context's GPU memory back at once instead of when
// the canvas is collected: a page may have only a few live WebGL contexts.
func (e *webEGL) destroyContext(uintptr, uintptr) bool {
	if e.closed {
		return true
	}
	e.closed = true
	e.canvas.Call("removeEventListener", "webglcontextlost", e.lostFn)
	e.canvas.Call("removeEventListener", "webglcontextrestored", e.restoredFn)
	e.lostFn.Release()
	e.restoredFn.Release()
	if ext := e.gl.Call("getExtension", "WEBGL_lose_context"); ext.Truthy() {
		ext.Call("loseContext")
	}
	return true
}
func (e *webEGL) terminate(uintptr) bool        { return true }
func (e *webEGL) getError() int32               { return 0 }
func (e *webEGL) getProcAddress(string) uintptr { return 0 }

// setSurfaceSize sets the canvas's width and height when they are not the frame's; setting them
// clears the canvas, and the frame covers all of it.
func (e *webEGL) setSurfaceSize(_ uintptr, width, height int) {
	if width == e.width && height == e.height {
		return
	}
	e.canvas.Set("width", width)
	e.canvas.Set("height", height)
	e.width, e.height = width, height
}
