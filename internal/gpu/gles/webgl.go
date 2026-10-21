package gles

import (
	"strconv"
	"sync"

	"github.com/rm4n0s/errors"
)

// This file is the part of the WebGL 2 binding (gl_js.go, context_js.go) that is plain Go: what
// to ask the browser for, what the context's loss and restoration do to the renderer, and the
// errors. It builds on every OS so that the normal tests and the tests under Node run it.

// eglPlatformWeb stands for the page's canvas where EGL would have a platform: the web has no
// EGL, and the renderer's context code goes through a small imitation of it (webEGL in
// context_js.go).
const eglPlatformWeb = 0x7EB0

// webContextAttributes are the attributes of getContext("webgl2", ...) (SPEC §9.1):
//   - alpha with premultipliedAlpha: the same RGBA8 premultiplied surface that EGL gives on
//     Linux, so that every blend and every read-back is the one the tests know
//   - antialias false: edges are analytic, with no multisampling
//   - depth and stencil false: the window's framebuffer needs neither (clips use coverage masks
//     and layers have their own framebuffers with a stencil renderbuffer where they need one)
//   - preserveDrawingBuffer false: the browser may present without copying; a frame is drawn
//     whole and read back, when it is, before the task ends
//   - failIfMajorPerformanceCaveat false: a browser whose WebGL is software still gives a
//     context, which is faster than soft in WebAssembly
//   - powerPreference default: a UI should not wake a discrete GPU
func webContextAttributes() map[string]any {
	return map[string]any{
		"alpha":                        true,
		"premultipliedAlpha":           true,
		"antialias":                    false,
		"depth":                        false,
		"stencil":                      false,
		"preserveDrawingBuffer":        false,
		"failIfMajorPerformanceCaveat": false,
		"powerPreference":              "default",
	}
}

// lossy is implemented by an eglAPI whose context can be taken from the renderer and given back
// by the browser (webglcontextlost and webglcontextrestored): the web's.
type lossy interface {
	// contextLost says that the context is lost now: nothing can be drawn, and every
	// resource made in it is gone.
	contextLost() bool
	// needsRebuild says that a context was lost since the device was made and is usable
	// again, so every resource has to be made again. It stays true until rebuilt is called.
	needsRebuild() bool
	// rebuilt says that the renderer made its resources again.
	rebuilt()
}

// lossState is what the two events of a WebGL context do to a renderer. The browser's callbacks
// run on goroutines of their own and the renderer on the UI goroutine, so it has a lock.
type lossState struct {
	mu      sync.Mutex
	lost    bool
	rebuild bool
}

// onLost records webglcontextlost.
func (s *lossState) onLost() {
	s.mu.Lock()
	s.lost, s.rebuild = true, true
	s.mu.Unlock()
}

// onRestored records webglcontextrestored.
func (s *lossState) onRestored() {
	s.mu.Lock()
	s.lost = false
	s.mu.Unlock()
}

func (s *lossState) contextLost() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lost
}

func (s *lossState) needsRebuild() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rebuild && !s.lost
}

func (s *lossState) rebuilt() {
	s.mu.Lock()
	s.rebuild = false
	s.mu.Unlock()
}

// errWebCanvasNotFound is the error for a view whose canvas is not in the page.
func errWebCanvasNotFound(view uintptr) error {
	return errors.New("WebGLCanvasNotFound", "there is no canvas for the view in the page", "view", strconv.FormatUint(uint64(view), 10))
}

// errWebGL2Unavailable is the error for a canvas that gives no WebGL 2 context: the browser has
// none (or it is turned off, or the GPU is blocklisted), or the canvas already has a context of
// another kind (a canvas keeps the first kind it was asked for, SPEC §9.6).
func errWebGL2Unavailable(view uintptr, reason string) error {
	return errors.New("WebGL2Unavailable", "the canvas gave no WebGL 2 context",
		"view", strconv.FormatUint(uint64(view), 10), "reason", reason)
}

// errWebSurfaceKind is the error for a handle that does not name a canvas.
func errWebSurfaceKind(kind string) error {
	return errors.New("WebGLSurfaceKindUnsupported", "WebGL draws into the canvas of a web page only", "kind", kind)
}

// errContextLost is the error for a read-back or a snapshot while the context is lost.
func errContextLost() error {
	return errors.New("WebGLContextLost", "the WebGL context is lost; the next frame is drawn when the browser restores it")
}
