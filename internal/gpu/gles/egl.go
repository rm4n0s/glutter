package gles

import (
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Environment variables that point at an EGL or GLES library by absolute path,
// for setups where the dynamic linker does not find it (docs/SPEC.md §8.1).
const (
	eglLibraryEnv  = "GLUTTER_EGL_LIBRARY"
	glesLibraryEnv = "GLUTTER_GLES_LIBRARY"
)

// EGL constants (EGL/egl.h, EGL/eglext.h).
const (
	eglNone              = 0x3038
	eglRedSize           = 0x3024
	eglGreenSize         = 0x3023
	eglBlueSize          = 0x3022
	eglAlphaSize         = 0x3021
	eglSurfaceType       = 0x3033
	eglWindowBit         = 0x0004
	eglRenderableType    = 0x3040
	eglOpenGLES3Bit      = 0x0040
	eglOpenGLESAPI       = 0x30A0
	eglContextClientVers = 0x3098
	eglPlatformWayland   = 0x31D8 // EGL_PLATFORM_WAYLAND_KHR
	eglPlatformX11       = 0x31D5 // EGL_PLATFORM_X11_KHR
	eglSuccess           = 0x3000
)

// eglAPI is the part of EGL that the context uses, in Go terms. The real
// implementation calls libEGL through purego; tests supply a fake.
type eglAPI interface {
	getPlatformDisplay(platform uint32, nativeDisplay uintptr) uintptr
	initialize(display uintptr) (major, minor int32, ok bool)
	// chooseConfig returns the best config for the attributes, or ok false when
	// no config matches.
	chooseConfig(display uintptr, attribs []int32) (config uintptr, ok bool)
	bindAPI(api uint32) bool
	createContext(display, config uintptr, attribs []int32) uintptr
	createWindowSurface(display, config, nativeWindow uintptr) uintptr
	makeCurrent(display, draw, read, context uintptr) bool
	swapInterval(display uintptr, interval int32) bool
	swapBuffers(display, surface uintptr) bool
	destroySurface(display, surface uintptr) bool
	destroyContext(display, context uintptr) bool
	terminate(display uintptr) bool
	getError() int32
	// getProcAddress returns the address of a GL function, or 0.
	getProcAddress(name string) uintptr
}

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
	makeCurrentFn              func(display, draw, read, context uintptr) uint32
	swapIntervalFn             func(display uintptr, interval int32) uint32
	swapBuffersFn              func(display, surface uintptr) uint32
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
		return nil, loadFailed("libEGL", err)
	}
	// libGLESv2 is opened too: it must be loaded for GL entry points to work
	// with libglvnd, and its symbols back up eglGetProcAddress.
	if _, err := dl.OpenOverride(glesLibraryEnv, "libGLESv2.so.2", "libGLESv2.so"); err != nil {
		return nil, loadFailed("libGLESv2", err)
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
		{&l.makeCurrentFn, "eglMakeCurrent"},
		{&l.swapIntervalFn, "eglSwapInterval"},
		{&l.swapBuffersFn, "eglSwapBuffers"},
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

// loadFailed turns the loader's LibraryLoadFailed into the EGLLoadFailed of the
// renderer, keeping what the loader reported.
func loadFailed(library string, err error) error {
	md := map[string]any{}
	if appErr, ok := errors.FromError(err); ok {
		md = appErr.Metadata
	}
	return errors.New("EGLLoadFailed", "could not load the EGL or GLES library",
		"library", library, "names", md["names"], "dlerror", md["dlerror"])
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

func (l *eglLib) createWindowSurface(display, config, nativeWindow uintptr) uintptr {
	if l.hasGetPlatformDisplay || l.hasGetPlatformDisplayEXT {
		return l.createPlatformWindowSurfFn(display, config, nativeWindow, 0)
	}
	return l.createWindowSurfaceFn(display, config, nativeWindow, 0)
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
