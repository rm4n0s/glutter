package gles

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// fakeEGL is an eglAPI whose answers the test chooses. It records the calls.
type fakeEGL struct {
	displayFails, initFails, bindFails, noConfig, contextFails, surfaceFails, currentFails, swapFails bool
	major, minor                                                                                      int32
	errCode                                                                                           int32
	calls                                                                                             []string
	platform                                                                                          uint32
	nativeDisplay, nativeWindow                                                                       uintptr
	configAttribs, contextAttribs                                                                     []int32
	swapIntervalValue                                                                                 int32
	procs                                                                                             map[string]uintptr
	surfaceX11                                                                                        bool
	visualID                                                                                          int32
	noVisual                                                                                          bool
	surfacelessFails                                                                                  bool
	damageOK                                                                                          bool
	damageRects                                                                                       [][]int32
}

func (f *fakeEGL) getConfigAttrib(_, _ uintptr, attribute int32) (int32, bool) {
	f.call("getConfigAttrib")
	return f.visualID, attribute == eglNativeVisualID && !f.noVisual
}

func newFakeEGL() *fakeEGL {
	return &fakeEGL{major: 1, minor: 5, errCode: 0x3001, procs: map[string]uintptr{}}
}

func (f *fakeEGL) call(name string) { f.calls = append(f.calls, name) }

func (f *fakeEGL) getPlatformDisplay(platform uint32, native uintptr) uintptr {
	f.call("getPlatformDisplay")
	f.platform, f.nativeDisplay = platform, native
	if f.displayFails {
		return 0
	}
	return 0xD15
}
func (f *fakeEGL) initialize(uintptr) (int32, int32, bool) {
	f.call("initialize")
	return f.major, f.minor, !f.initFails
}
func (f *fakeEGL) chooseConfig(_ uintptr, attribs []int32) (uintptr, bool) {
	f.call("chooseConfig")
	f.configAttribs = attribs
	return 0xC0F, !f.noConfig
}
func (f *fakeEGL) bindAPI(api uint32) bool {
	f.call("bindAPI")
	return !f.bindFails && api == eglOpenGLESAPI
}
func (f *fakeEGL) createContext(_, _ uintptr, attribs []int32) uintptr {
	f.call("createContext")
	f.contextAttribs = attribs
	if f.contextFails {
		return 0
	}
	return 0xC7
}
func (f *fakeEGL) createWindowSurface(_, _, window uintptr, x11 bool) uintptr {
	f.call("createWindowSurface")
	f.nativeWindow, f.surfaceX11 = window, x11
	if f.surfaceFails {
		return 0
	}
	return 0x5F
}
func (f *fakeEGL) makeCurrent(_, draw, _, ctx uintptr) bool {
	if draw == 0 && ctx == 0 {
		f.call("makeCurrent(none)")
		return true
	}
	if draw == 0 {
		f.call("makeCurrent(surfaceless)")
		return !f.surfacelessFails
	}
	f.call("makeCurrent")
	return !f.currentFails
}
func (f *fakeEGL) swapInterval(_ uintptr, interval int32) bool {
	f.call("swapInterval")
	f.swapIntervalValue = interval
	return true
}
func (f *fakeEGL) swapBuffers(uintptr, uintptr) bool {
	f.call("swapBuffers")
	return !f.swapFails
}
func (f *fakeEGL) canSwapWithDamage(uintptr) bool { return f.damageOK }
func (f *fakeEGL) swapBuffersWithDamage(_, _ uintptr, rects []int32) bool {
	f.call("swapBuffersWithDamage")
	f.damageRects = append(f.damageRects, rects)
	return !f.swapFails
}
func (f *fakeEGL) destroySurface(uintptr, uintptr) bool { f.call("destroySurface"); return true }
func (f *fakeEGL) destroyContext(uintptr, uintptr) bool { f.call("destroyContext"); return true }
func (f *fakeEGL) terminate(uintptr) bool               { f.call("terminate"); return true }
func (f *fakeEGL) getError() int32                      { return f.errCode }
func (f *fakeEGL) getProcAddress(name string) uintptr   { return f.procs[name] }

func fakeGLLoader(g *fakeGL) func(eglAPI) (glAPI, error) {
	return func(eglAPI) (glAPI, error) { return g, nil }
}

var waylandHandle = gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Display: 0xD, Window: 0xA}

func TestOpenSucceeds(t *testing.T) {
	tests := []struct {
		name         string
		handle       gpu.SurfaceHandle
		wantPlatform uint32
	}{
		{"wayland", waylandHandle, eglPlatformWayland},
		{"x11", gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Display: 0xD, Window: 0xA}, eglPlatformX11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			egl, gl := newFakeEGL(), &fakeGL{versionString: "OpenGL ES 3.2 Mesa 24.1.0"}
			c, s, err := openWith(egl, fakeGLLoader(gl), tt.handle)
			if err != nil {
				t.Fatalf("openWith: %v", err)
			}
			if c.Version() != "OpenGL ES 3.2 Mesa 24.1.0" {
				t.Errorf("Version() = %q", c.Version())
			}
			wantCalls := []string{"getPlatformDisplay", "initialize", "bindAPI", "chooseConfig", "createContext", "createWindowSurface", "makeCurrent", "swapInterval"}
			if !reflect.DeepEqual(egl.calls, wantCalls) {
				t.Errorf("EGL calls = %v, want %v", egl.calls, wantCalls)
			}
			if egl.platform != tt.wantPlatform || egl.nativeDisplay != 0xD || egl.nativeWindow != 0xA {
				t.Errorf("platform %#x display %#x window %#x, want %#x 0xd 0xa", egl.platform, egl.nativeDisplay, egl.nativeWindow, tt.wantPlatform)
			}
			if egl.swapIntervalValue != 0 {
				t.Errorf("swap interval = %d, want 0: frames are paced by frame callbacks", egl.swapIntervalValue)
			}
			if !reflect.DeepEqual(egl.contextAttribs, []int32{eglContextClientVers, 3, eglNone}) {
				t.Errorf("context attribs = %v, want client version 3", egl.contextAttribs)
			}
			// RGBA8, OpenGL ES 3 and a window surface, NONE-terminated.
			want := map[int32]int32{eglRedSize: 8, eglGreenSize: 8, eglBlueSize: 8, eglAlphaSize: 8, eglRenderableType: eglOpenGLES3Bit, eglSurfaceType: eglWindowBit}
			got := map[int32]int32{}
			for i := 0; i+1 < len(egl.configAttribs); i += 2 {
				got[egl.configAttribs[i]] = egl.configAttribs[i+1]
			}
			delete(got, eglNone)
			if !reflect.DeepEqual(got, want) || egl.configAttribs[len(egl.configAttribs)-1] != eglNone {
				t.Errorf("config attribs = %v, want %v ending in EGL_NONE", egl.configAttribs, want)
			}
			s.Destroy()
			c.Close()
		})
	}
}

func TestOpenFailures(t *testing.T) {
	tests := []struct {
		name    string
		handle  gpu.SurfaceHandle
		tweak   func(*fakeEGL, *fakeGL)
		loader  func(eglAPI) (glAPI, error)
		tag     string
		route   string
		cleanup []string // calls that must have run to release what was created
	}{
		{
			name: "a surface kind without an EGL platform", handle: gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake},
			tweak: func(*fakeEGL, *fakeGL) {}, tag: "EGLInitFailed", route: "openWith.EGLInitFailed",
		},
		{
			name: "no EGL display", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.displayFails = true }, tag: "EGLInitFailed", route: "openWith->eglError.EGLInitFailed",
		},
		{
			name: "eglInitialize fails", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.initFails = true }, tag: "EGLInitFailed", route: "openWith->eglError.EGLInitFailed",
		},
		{
			name: "EGL older than 1.4", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.minor = 3 }, tag: "EGLInitFailed", route: "openWith.EGLInitFailed",
			cleanup: []string{"terminate"},
		},
		{
			name: "no OpenGL ES API", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.bindFails = true }, tag: "GLES3Unsupported", route: "openWith->eglError.GLES3Unsupported",
			cleanup: []string{"terminate"},
		},
		{
			name: "no config", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.noConfig = true }, tag: "EGLNoConfig", route: "openWith->eglError.EGLNoConfig",
			cleanup: []string{"terminate"},
		},
		{
			name: "no ES 3 context", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.contextFails = true }, tag: "GLES3Unsupported", route: "openWith->eglError.GLES3Unsupported",
			cleanup: []string{"terminate"},
		},
		{
			name: "no window surface", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.surfaceFails = true }, tag: "EGLSurfaceFailed",
			route:   "openWith->Context.NewSurface->eglError.EGLSurfaceFailed",
			cleanup: []string{"destroyContext", "terminate"},
		},
		{
			name: "eglMakeCurrent fails", handle: waylandHandle,
			tweak: func(e *fakeEGL, _ *fakeGL) { e.currentFails = true }, tag: "EGLMakeCurrentFailed",
			route:   "openWith->Context.MakeCurrent->eglError.EGLMakeCurrentFailed",
			cleanup: []string{"destroySurface", "destroyContext", "terminate"},
		},
		{
			name: "a GL function is missing", handle: waylandHandle,
			tweak: func(*fakeEGL, *fakeGL) {},
			loader: func(eglAPI) (glAPI, error) {
				return nil, errors.New("SymbolMissing", "no glClear", "symbol", "glClear")
			},
			tag:     "SymbolMissing",
			route:   "SymbolMissing", // returned unchanged
			cleanup: []string{"destroySurface", "destroyContext", "terminate"},
		},
		{
			name: "the driver reports OpenGL ES 2.0", handle: waylandHandle,
			tweak: func(_ *fakeEGL, g *fakeGL) { g.versionString = "OpenGL ES 2.0 Old Driver" }, tag: "GLES3Unsupported",
			route:   "openWith.GLES3Unsupported",
			cleanup: []string{"destroySurface", "destroyContext", "terminate"},
		},
		{
			name: "the driver reports desktop OpenGL", handle: waylandHandle,
			tweak: func(_ *fakeEGL, g *fakeGL) { g.versionString = "4.6 (Core Profile) Mesa 24.1" }, tag: "GLES3Unsupported",
			route: "openWith.GLES3Unsupported", cleanup: []string{"terminate"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			egl, gl := newFakeEGL(), &fakeGL{versionString: "OpenGL ES 3.0 Mesa"}
			tt.tweak(egl, gl)
			loader := fakeGLLoader(gl)
			if tt.loader != nil {
				loader = tt.loader
			}
			c, s, err := openWith(egl, loader, tt.handle)
			if c != nil || s != nil {
				t.Fatalf("openWith returned a context or surface on failure")
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag {
				t.Errorf("Tag = %q, want %q (%s)", appErr.Tag, tt.tag, appErr.Message)
			}
			if tt.route != "SymbolMissing" && !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), tt.route)
			}
			for _, want := range tt.cleanup {
				if !contains(egl.calls, want) {
					t.Errorf("%s was not called to release what was created; calls: %v", want, egl.calls)
				}
			}
			if len(tt.cleanup) == 0 && contains(egl.calls, "terminate") {
				t.Errorf("terminate was called before anything needed releasing: %v", egl.calls)
			}
		})
	}
}

func TestEGLErrorCodeIsReported(t *testing.T) {
	egl := newFakeEGL()
	egl.noConfig = true
	egl.errCode = 0x3009 // EGL_BAD_MATCH
	_, _, err := openWith(egl, fakeGLLoader(&fakeGL{}), waylandHandle)
	appErr, _ := errors.FromError(err)
	if appErr == nil || appErr.Metadata["egl_error"] != "0x3009" {
		t.Errorf("Metadata = %v, want egl_error 0x3009", appErr)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestIsES3(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"OpenGL ES 3.0 Mesa 24.1.0", true},
		{"OpenGL ES 3.2 NVIDIA 550.54", true},
		{"OpenGL ES 3.1", true},
		{"OpenGL ES 4.0 Future", true},
		{"OpenGL ES 2.0 Mesa", false},
		{"OpenGL ES-CM 1.1", false},
		{"4.6 (Core Profile) Mesa", false},
		{"", false},
		{"OpenGL ES", false},
	}
	for _, tt := range tests {
		if got := isES3(tt.version); got != tt.want {
			t.Errorf("isES3(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func openFake(t *testing.T) (*Context, *Surface, *fakeEGL, *fakeGL) {
	t.Helper()
	egl, gl := newFakeEGL(), &fakeGL{versionString: "OpenGL ES 3.0 Mesa"}
	c, s, err := openWith(egl, fakeGLLoader(gl), waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, egl, gl
}

func TestContextOperations(t *testing.T) {
	c, s, egl, gl := openFake(t)

	c.Viewport(640, 400)
	if gl.viewportW != 640 || gl.viewportH != 400 {
		t.Errorf("viewport = %dx%d, want 640x400", gl.viewportW, gl.viewportH)
	}
	c.Clear(0.4, 0.31, 0.64, 1)
	if gl.cleared != [4]float32{0.4, 0.31, 0.64, 1} || gl.clears != 1 {
		t.Errorf("clear color %v, clears %d", gl.cleared, gl.clears)
	}
	if err := c.SwapBuffers(s); err != nil {
		t.Errorf("SwapBuffers: %v", err)
	}
	if err := c.MakeCurrent(s); err != nil {
		t.Errorf("MakeCurrent: %v", err)
	}

	egl.swapFails = true
	err := c.SwapBuffers(s)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "EGLSwapFailed" || !appErr.HasRoute("Context.SwapBuffers->eglError.EGLSwapFailed") {
		t.Errorf("SwapBuffers = %v, want EGLSwapFailed via Context.SwapBuffers", err)
	}
	egl.swapFails = false
	egl.currentFails = true
	if err := c.MakeCurrent(s); err == nil {
		t.Error("MakeCurrent succeeded although EGL refused")
	}
	egl.currentFails = false

	s.Destroy()
	s.Destroy() // twice is harmless
	c.Close()
	c.Close()
	destroys := 0
	for _, call := range egl.calls {
		if call == "destroySurface" {
			destroys++
		}
	}
	if destroys != 1 {
		t.Errorf("the surface was destroyed %d times, want 1", destroys)
	}
	tail := egl.calls[len(egl.calls)-3:]
	if !reflect.DeepEqual(tail, []string{"makeCurrent(none)", "destroyContext", "terminate"}) {
		t.Errorf("Close ended with %v, want makeCurrent(none), destroyContext, terminate", tail)
	}
}

func TestCloseDestroysSurfacesThatAreStillOpen(t *testing.T) {
	c, _, egl, _ := openFake(t)
	if _, err := c.NewSurface(waylandHandle); err != nil {
		t.Fatal(err)
	}
	c.Close()
	destroys := 0
	for _, call := range egl.calls {
		if call == "destroySurface" {
			destroys++
		}
	}
	if destroys != 2 {
		t.Errorf("Close destroyed %d surfaces, want both", destroys)
	}
}

func TestReadPixelsFlipsRows(t *testing.T) {
	c, _, _, gl := openFake(t)
	// A 2x3 framebuffer, bottom row first: rows are 1, 2, 3 (one distinct color each).
	gl.pixels = []byte{
		1, 1, 1, 255, 1, 1, 1, 255, // bottom
		2, 2, 2, 255, 2, 2, 2, 255,
		3, 3, 3, 255, 3, 3, 3, 255, // top
	}
	img, err := c.ReadPixels(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	for y, want := range []byte{3, 2, 1} { // top row first
		for x := range 2 {
			if got := img.RGBAAt(x, y).R; got != want {
				t.Errorf("pixel (%d,%d) = %d, want %d: rows must be flipped", x, y, got, want)
			}
		}
	}
	if b := img.Bounds(); b.Dx() != 2 || b.Dy() != 3 {
		t.Errorf("bounds = %v, want 2x3", b)
	}

	t.Run("a GL error", func(t *testing.T) {
		gl.errCode = 0x502
		_, err := c.ReadPixels(2, 3)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "GLReadPixelsFailed" || appErr.Metadata["gl_error"] != "0x502" ||
			!appErr.HasRoute("Context.ReadPixels.GLReadPixelsFailed") {
			t.Errorf("ReadPixels = %v, want GLReadPixelsFailed with gl_error 0x502", err)
		}
	})
	t.Run("a bad size", func(t *testing.T) {
		for _, size := range [][2]int{{0, 3}, {2, 0}, {-1, -1}} {
			_, err := c.ReadPixels(size[0], size[1])
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GLReadPixelsFailed" {
				t.Errorf("ReadPixels(%v) = %v, want GLReadPixelsFailed", size, err)
			}
		}
	})
}

func TestInt32Array(t *testing.T) {
	// Not a pointer test: the values must land in C memory in order.
	if got := int32Array(nil); got == 0 {
		t.Error("an empty array still needs valid memory")
	}
}

func TestOpenReportsEGLLoadFailed(t *testing.T) {
	missing := "/nonexistent-glutter-dir/libEGL.so.1"
	t.Setenv(eglLibraryEnv, missing)
	_, _, err := Open(waylandHandle)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "EGLLoadFailed" || !appErr.HasRoute("Open->loadEGL->loadFailed.EGLLoadFailed") {
		t.Errorf("Tag = %q, route = %s, want EGLLoadFailed via Open->loadEGL->loadFailed", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["library"] != "libEGL" {
		t.Errorf("Metadata[library] = %v, want libEGL", appErr.Metadata["library"])
	}
	names, _ := appErr.Metadata["names"].([]string)
	if len(names) != 1 || names[0] != missing {
		t.Errorf("Metadata[names] = %v, want the override path", appErr.Metadata["names"])
	}
	if s, _ := appErr.Metadata["dlerror"].(string); !strings.Contains(s, "nonexistent") {
		t.Errorf("Metadata[dlerror] = %q, want the loader's message", s)
	}
}

func TestOpenReportsGLESLoadFailed(t *testing.T) {
	// libEGL loads normally here; only libGLESv2 is pointed at a missing file.
	if _, err := loadEGL(); err != nil {
		t.Skipf("libEGL is not available: %v", err)
	}
	t.Setenv(glesLibraryEnv, "/nonexistent-glutter-dir/libGLESv2.so.2")
	_, _, err := Open(waylandHandle)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "EGLLoadFailed" || appErr.Metadata["library"] != "libGLESv2" {
		t.Errorf("Open returned %v, want EGLLoadFailed for libGLESv2", err)
	}
}

func TestX11VisualReturnsTheVisualOfTheChosenConfig(t *testing.T) {
	f := newFakeEGL()
	f.visualID = 0x21
	id, err := x11VisualWith(f, 0xD0)
	if err != nil {
		t.Fatal(err)
	}
	if id != 0x21 {
		t.Errorf("visual = 0x%x, want 0x21", id)
	}
	if f.platform != eglPlatformX11 || f.nativeDisplay != 0xD0 {
		t.Errorf("platform 0x%x on display 0x%x, want the X11 platform on the Xlib display", f.platform, f.nativeDisplay)
	}
	if !reflect.DeepEqual(f.configAttribs, configAttribs) {
		t.Error("the visual comes from a different config than the one Open uses")
	}
	if last := f.calls[len(f.calls)-1]; last != "terminate" {
		t.Errorf("the last call is %s: the display must be terminated again", last)
	}
}

func TestX11VisualFailures(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(*fakeEGL)
		route string
	}{
		{"no display", func(f *fakeEGL) { f.displayFails = true }, "x11VisualWith->eglError.EGLInitFailed"},
		{"no initialization", func(f *fakeEGL) { f.initFails = true }, "x11VisualWith->eglError.EGLInitFailed"},
		{"no GLES", func(f *fakeEGL) { f.bindFails = true }, "x11VisualWith->eglError.GLES3Unsupported"},
		{"no config", func(f *fakeEGL) { f.noConfig = true }, "x11VisualWith->eglError.EGLNoConfig"},
		{"no native visual", func(f *fakeEGL) { f.noVisual = true }, "x11VisualWith->eglError.EGLConfigQueryFailed"},
		{"visual 0", func(f *fakeEGL) { f.visualID = 0 }, "x11VisualWith->eglError.EGLConfigQueryFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeEGL()
			f.visualID = 0x21
			tt.tweak(f)
			_, err := x11VisualWith(f, 1)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
}

func TestX11SurfaceIsMadeFromTheWindowID(t *testing.T) {
	f := newFakeEGL()
	g := &fakeGL{versionString: "OpenGL ES 3.2 Mesa"}
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Display: 0xD0, Window: 0x4000007}
	_, s, err := openWith(f, func(eglAPI) (glAPI, error) { return g, nil }, h)
	if err != nil || s == nil {
		t.Fatalf("openWith = %v", err)
	}
	if !f.surfaceX11 || f.nativeWindow != 0x4000007 || f.platform != eglPlatformX11 {
		t.Errorf("x11 %v window 0x%x platform 0x%x", f.surfaceX11, f.nativeWindow, f.platform)
	}
	// A Wayland surface is not an X11 one.
	f2 := newFakeEGL()
	if _, _, err := openWith(f2, func(eglAPI) (glAPI, error) { return g, nil }, gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Display: 1, Window: 2}); err != nil {
		t.Fatal(err)
	}
	if f2.surfaceX11 {
		t.Error("a Wayland window was passed as an X11 one")
	}
}
