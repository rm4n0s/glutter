package gles

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
	eglPlatformAndroid   = 0x3141 // EGL_PLATFORM_ANDROID_KHR
	eglNativeVisualID    = 0x302E // EGL_NATIVE_VISUAL_ID
	eglSuccess           = 0x3000
	eglExtensions        = 0x3055 // EGL_EXTENSIONS
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
	// createWindowSurface makes a window surface for the native window: a
	// wl_egl_window, or (x11 true) the ID of an X window.
	createWindowSurface(display, config, nativeWindow uintptr, x11 bool) uintptr
	// getConfigAttrib reads one attribute of a config.
	getConfigAttrib(display, config uintptr, attribute int32) (value int32, ok bool)
	makeCurrent(display, draw, read, context uintptr) bool
	swapInterval(display uintptr, interval int32) bool
	swapBuffers(display, surface uintptr) bool
	// canSwapWithDamage says whether the display has EGL_KHR_swap_buffers_with_damage
	// or the EXT extension of the same name, so that swapBuffersWithDamage works.
	canSwapWithDamage(display uintptr) bool
	// swapBuffersWithDamage presents the surface and tells the compositor which
	// rectangles changed: x, y, width and height for each, from the bottom left of
	// the surface (as EGL counts).
	swapBuffersWithDamage(display, surface uintptr, rects []int32) bool
	destroySurface(display, surface uintptr) bool
	destroyContext(display, context uintptr) bool
	terminate(display uintptr) bool
	getError() int32
	// getProcAddress returns the address of a GL function, or 0.
	getProcAddress(name string) uintptr
}
