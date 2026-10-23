//go:build linux

package gles

import (
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Environment variables that point at an EGL or GLES library by absolute path,
// for setups where the dynamic linker does not find it (docs/SPEC.md §8.1).
const (
	eglLibraryEnv  = "GLUTTER_EGL_LIBRARY"
	glesLibraryEnv = "GLUTTER_GLES_LIBRARY"
)

// eglLib is the purego implementation of eglAPI.
type eglLib struct {
	lib *dl.Lib

	getPlatformDisplayFn       func(platform uint32, native, attribs uintptr) uintptr
	getPlatformDisplayEXTFn    func(platform uint32, native, attribs uintptr) uintptr
	getDisplayFn               func(native uintptr) uintptr
	initializeFn               func(display, major, minor uintptr) uint32
	chooseConfigFn             func(display, attribs, configs uintptr, size int32, num uintptr) uint32
	bindAPIFn                  func(api uint32) uint32
	createContextFn            func(display, config, share, attribs uintptr) uintptr
	createPlatformWindowSurfFn func(display, config, native, attribs uintptr) uintptr
	createWindowSurfaceFn      func(display, config, native, attribs uintptr) uintptr
	getConfigAttribFn          func(display, config uintptr, attribute int32, value uintptr) uint32
	makeCurrentFn              func(display, draw, read, context uintptr) uint32
	swapIntervalFn             func(display uintptr, interval int32) uint32
	swapBuffersFn              func(display, surface uintptr) uint32
	queryStringFn              func(display uintptr, name int32) uintptr
	swapWithDamageFn           func(display, surface, rects uintptr, n int32) uint32
	destroySurfaceFn           func(display, surface uintptr) uint32
	destroyContextFn           func(display, context uintptr) uint32
	terminateFn                func(display uintptr) uint32
	getErrorFn                 func() int32
	getProcAddressFn           func(name uintptr) uintptr

	hasGetPlatformDisplay    bool
	hasGetPlatformDisplayEXT bool
}

// loadEGL opens libEGL and libGLESv2 (SPEC §8.1) and binds the EGL functions. A
// library that cannot be loaded gives EGLLoadFailed; a missing symbol gives
// SymbolMissing.
func loadEGL() (*eglLib, error) {
	egl, err := dl.OpenOverride(eglLibraryEnv, "libEGL.so.1", "libEGL.so")
	if err != nil {
		md := loaderMetadata(err)
		return nil, errors.New(tagEGLLoadFailed, "could not load the EGL or GLES library",
			"library", "libEGL", "names", md["names"], "dlerror", md["dlerror"])
	}
	// libGLESv2 is opened too: it must be loaded for GL entry points to work
	// with libglvnd, and its symbols back up eglGetProcAddress.
	if _, err := dl.OpenOverride(glesLibraryEnv, "libGLESv2.so.2", "libGLESv2.so"); err != nil {
		md := loaderMetadata(err)
		return nil, errors.New(tagEGLLoadFailed, "could not load the EGL or GLES library",
			"library", "libGLESv2", "names", md["names"], "dlerror", md["dlerror"])
	}
	l := &eglLib{lib: egl}
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.getDisplayFn, "eglGetDisplay"},
		{&l.initializeFn, "eglInitialize"},
		{&l.chooseConfigFn, "eglChooseConfig"},
		{&l.bindAPIFn, "eglBindAPI"},
		{&l.createContextFn, "eglCreateContext"},
		{&l.createWindowSurfaceFn, "eglCreateWindowSurface"},
		{&l.getConfigAttribFn, "eglGetConfigAttrib"},
		{&l.makeCurrentFn, "eglMakeCurrent"},
		{&l.swapIntervalFn, "eglSwapInterval"},
		{&l.swapBuffersFn, "eglSwapBuffers"},
		{&l.queryStringFn, "eglQueryString"},
		{&l.destroySurfaceFn, "eglDestroySurface"},
		{&l.destroyContextFn, "eglDestroyContext"},
		{&l.terminateFn, "eglTerminate"},
		{&l.getErrorFn, "eglGetError"},
		{&l.getProcAddressFn, "eglGetProcAddress"},
	} {
		if err := egl.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	// EGL 1.5 has the platform functions in the core library; older drivers
	// have them as EXT extensions, found through eglGetProcAddress.
	if egl.Has("eglGetPlatformDisplay") && egl.Has("eglCreatePlatformWindowSurface") {
		if err := egl.Bind(&l.getPlatformDisplayFn, "eglGetPlatformDisplay"); err != nil {
			return nil, err
		}
		if err := egl.Bind(&l.createPlatformWindowSurfFn, "eglCreatePlatformWindowSurface"); err != nil {
			return nil, err
		}
		l.hasGetPlatformDisplay = true
	} else if ext := l.procAddress("eglGetPlatformDisplayEXT"); ext != 0 {
		if surf := l.procAddress("eglCreatePlatformWindowSurfaceEXT"); surf != 0 {
			registerFunc(&l.getPlatformDisplayEXTFn, ext)
			registerFunc(&l.createPlatformWindowSurfFn, surf)
			l.hasGetPlatformDisplayEXT = true
		}
	}
	return l, nil
}

// loaderMetadata returns what the loader reported about a library that did not load (its
// LibraryLoadFailed error), for the EGLLoadFailed of the renderer.
func loaderMetadata(err error) map[string]any {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Metadata
	}
	return map[string]any{}
}

func (l *eglLib) procAddress(name string) uintptr {
	cname := dl.CString(name)
	defer dl.Free(cname)
	return l.getProcAddressFn(cname)
}

func (l *eglLib) getProcAddress(name string) uintptr { return l.procAddress(name) }

func (l *eglLib) getPlatformDisplay(platform uint32, native uintptr) uintptr {
	switch {
	case l.hasGetPlatformDisplay:
		return l.getPlatformDisplayFn(platform, native, 0)
	case l.hasGetPlatformDisplayEXT:
		return l.getPlatformDisplayEXTFn(platform, native, 0)
	}
	return l.getDisplayFn(native) // the native display type is enough on Wayland
}

func (l *eglLib) initialize(display uintptr) (major, minor int32, ok bool) {
	mem := dl.Calloc(2, 4)
	defer dl.Free(mem)
	if l.initializeFn(display, mem, mem+4) == 0 {
		return 0, 0, false
	}
	return *(*int32)(dl.Pointer(mem)), *(*int32)(dl.Pointer(mem + 4)), true
}

func (l *eglLib) chooseConfig(display uintptr, attribs []int32) (uintptr, bool) {
	attrs := int32Array(attribs)
	defer dl.Free(attrs)
	out := dl.Calloc(1, 8) // EGLConfig
	num := dl.Calloc(1, 4) // EGLint
	defer dl.Free(out)
	defer dl.Free(num)
	if l.chooseConfigFn(display, attrs, out, 1, num) == 0 || *(*int32)(dl.Pointer(num)) < 1 {
		return 0, false
	}
	return *(*uintptr)(dl.Pointer(out)), true
}

func (l *eglLib) bindAPI(api uint32) bool { return l.bindAPIFn(api) != 0 }

func (l *eglLib) createContext(display, config uintptr, attribs []int32) uintptr {
	attrs := int32Array(attribs)
	defer dl.Free(attrs)
	return l.createContextFn(display, config, 0, attrs)
}

func (l *eglLib) createWindowSurface(display, config, nativeWindow uintptr, x11 bool) uintptr {
	if l.hasGetPlatformDisplay || l.hasGetPlatformDisplayEXT {
		native := nativeWindow
		if x11 {
			// The platform function wants a pointer to the X window ID, where the
			// older one wants the ID itself.
			mem := dl.Calloc(1, 8)
			defer dl.Free(mem)
			*(*uint64)(dl.Pointer(mem)) = uint64(nativeWindow)
			native = mem
		}
		return l.createPlatformWindowSurfFn(display, config, native, 0)
	}
	return l.createWindowSurfaceFn(display, config, nativeWindow, 0)
}

func (l *eglLib) getConfigAttrib(display, config uintptr, attribute int32) (int32, bool) {
	mem := dl.Calloc(1, 4)
	defer dl.Free(mem)
	if l.getConfigAttribFn(display, config, attribute, mem) == 0 {
		return 0, false
	}
	return *(*int32)(dl.Pointer(mem)), true
}

func (l *eglLib) makeCurrent(display, draw, read, context uintptr) bool {
	return l.makeCurrentFn(display, draw, read, context) != 0
}
func (l *eglLib) swapInterval(display uintptr, interval int32) bool {
	return l.swapIntervalFn(display, interval) != 0
}
func (l *eglLib) swapBuffers(display, surface uintptr) bool {
	return l.swapBuffersFn(display, surface) != 0
}

// canSwapWithDamage looks for the extension in the display's extension string and
// binds its function; both the KHR and the EXT form are the same call.
func (l *eglLib) canSwapWithDamage(display uintptr) bool {
	if l.swapWithDamageFn != nil {
		return true
	}
	exts := dl.GoString(l.queryStringFn(display, eglExtensions))
	for _, ext := range strings.Fields(exts) {
		if ext != "EGL_KHR_swap_buffers_with_damage" && ext != "EGL_EXT_swap_buffers_with_damage" {
			continue
		}
		for _, name := range []string{"eglSwapBuffersWithDamageKHR", "eglSwapBuffersWithDamageEXT"} {
			if addr := l.procAddress(name); addr != 0 {
				registerFunc(&l.swapWithDamageFn, addr)
				return true
			}
		}
	}
	return false
}

func (l *eglLib) swapBuffersWithDamage(display, surface uintptr, rects []int32) bool {
	if l.swapWithDamageFn == nil {
		return false
	}
	mem := int32Array(rects)
	defer dl.Free(mem)
	return l.swapWithDamageFn(display, surface, mem, int32(len(rects)/4)) != 0
}

func (l *eglLib) destroySurface(display, surface uintptr) bool {
	return l.destroySurfaceFn(display, surface) != 0
}
func (l *eglLib) destroyContext(display, context uintptr) bool {
	return l.destroyContextFn(display, context) != 0
}
func (l *eglLib) terminate(display uintptr) bool { return l.terminateFn(display) != 0 }
func (l *eglLib) getError() int32                { return l.getErrorFn() }

// int32Array copies values into new C memory and returns its address. The
// caller frees it.
func int32Array(values []int32) uintptr {
	mem := dl.Calloc(uintptr(max(len(values), 1)), 4)
	for i, v := range values {
		*(*int32)(unsafe.Add(dl.Pointer(mem), i*4)) = v
	}
	return mem
}

// registerFunc binds a C function at a known address (from eglGetProcAddress).
func registerFunc(fptr any, address uintptr) { purego.RegisterFunc(fptr, address) }

// Open loads EGL and GLES, creates an OpenGL ES 3.0 context on the display of h,
// creates the first window surface for h, makes it current, and turns vsync
// off (eglSwapInterval 0: frames are paced by the compositor's frame callbacks,
// SPEC §8.11). It fails with
//
//   - EGLLoadFailed when libEGL or libGLESv2 cannot be loaded
//   - EGLInitFailed when there is no EGL display or it will not initialize
//   - EGLNoConfig when no config has RGBA8 and OpenGL ES 3 window surfaces
//   - GLES3Unsupported when no ES 3.0 context can be created or the driver
//     reports an older GL_VERSION
//
// Close releases everything.
//
// Flutter: no counterpart
func Open(h gpu.SurfaceHandle) (*Context, *Surface, error) {
	egl, err := loadEGL()
	if err != nil {
		return nil, nil, err
	}
	return openWith(egl, loadGL, h)
}

// X11Visual returns the ID of the X visual that windows must use to be drawn with
// OpenGL ES through EGL on the Xlib display: the visual of the EGL config that
// Open chooses. Create the X window with this visual (and a colormap made for
// it), or eglCreateWindowSurface fails with BadMatch. It fails with
//
//   - EGLLoadFailed, EGLInitFailed and EGLNoConfig as Open does
//   - EGLConfigQueryFailed when the config has no native visual
//
// Flutter: no counterpart
func X11Visual(display uintptr) (uint32, error) {
	egl, err := loadEGL()
	if err != nil {
		return 0, err
	}
	return x11VisualWith(egl, display)
}

// NewRenderer returns a renderer that has no GL context yet: the first Attach
// creates it. Passing more than one RendererOpts panics with MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(opts ...RendererOpts) *Renderer {
	return newRenderer(Open, opts...)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEGLLoadFailed = "EGLLoadFailed"
)
