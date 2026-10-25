package gles

import (
	"fmt"
	"image"
	"regexp"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// Context is an EGL display with an OpenGL ES 3.0 context, shared by every
// window surface (SPEC §9.1). Use it only on the UI thread.
//
// Flutter: no counterpart
type Context struct {
	egl     eglAPI
	gl      glAPI
	dev     *Device
	display uintptr
	config  uintptr
	context uintptr
	version string
	surface map[*Surface]struct{}
	// current is the surface the context draws into, nil when none: a surface that is destroyed
	// while it is current first leaves the context current with no surface, so that the context,
	// its textures and its atlases outlive the surface (SPEC §24.4).
	current *Surface
	closed  bool
	// damage says whether the display can swap buffers with a damage region.
	damage bool
	// load binds the GL functions again when a lost context is rebuilt.
	load func(eglAPI) (glAPI, error)
}

// Surface is an EGL window surface of a Context.
//
// Flutter: no counterpart
type Surface struct {
	ctx       *Context
	handle    uintptr
	destroyed bool
}

// configAttribs asks for an RGBA8 window surface that OpenGL ES 3 can render
// to.
var configAttribs = []int32{
	eglSurfaceType, eglWindowBit,
	eglRenderableType, eglOpenGLES3Bit,
	eglRedSize, 8, eglGreenSize, 8, eglBlueSize, 8, eglAlphaSize, 8,
	eglNone,
}

// openWith is Open with the EGL and GL implementations given, for tests.
func openWith(egl eglAPI, loadGL func(eglAPI) (glAPI, error), h gpu.SurfaceHandle) (*Context, *Surface, error) {
	var platform uint32
	switch h.Kind {
	case gpu.SurfaceKindWayland:
		platform = eglPlatformWayland
	case gpu.SurfaceKindX11:
		platform = eglPlatformX11
	case gpu.SurfaceKindWeb:
		platform = eglPlatformWeb
	case gpu.SurfaceKindAndroid:
		platform = eglPlatformAndroid
	default:
		return nil, nil, errors.New(tagEGLInitFailed, "no EGL platform for this kind of surface", "kind", string(h.Kind))
	}

	display := egl.getPlatformDisplay(platform, h.Display)
	if display == 0 {
		return nil, nil, errors.New(tagEGLInitFailed, "EGL has no display for the native display", "egl_error", eglErrorCode(egl), "step", "eglGetPlatformDisplay")
	}
	major, minor, ok := egl.initialize(display)
	if !ok {
		return nil, nil, errors.New(tagEGLInitFailed, "eglInitialize failed", "egl_error", eglErrorCode(egl), "step", "eglInitialize")
	}
	if major < 1 || (major == 1 && minor < 4) {
		egl.terminate(display)
		return nil, nil, errors.New(tagEGLInitFailed, "EGL 1.4 or newer is required", "version", fmt.Sprintf("%d.%d", major, minor))
	}
	if !egl.bindAPI(eglOpenGLESAPI) {
		egl.terminate(display)
		return nil, nil, errors.New(tagGLES3Unsupported, "this EGL has no OpenGL ES API", "egl_error", eglErrorCode(egl), "step", "eglBindAPI")
	}
	config, ok := egl.chooseConfig(display, configAttribs)
	if !ok {
		egl.terminate(display)
		return nil, nil, errors.New(tagEGLNoConfig, "no EGL config has RGBA8 and OpenGL ES 3 window surfaces", "egl_error", eglErrorCode(egl))
	}
	context := egl.createContext(display, config, []int32{eglContextClientVers, 3, eglNone})
	if context == 0 {
		egl.terminate(display)
		return nil, nil, errors.New(tagGLES3Unsupported, "could not create an OpenGL ES 3.0 context", "egl_error", eglErrorCode(egl), "step", "eglCreateContext")
	}

	c := &Context{egl: egl, display: display, config: config, context: context, surface: map[*Surface]struct{}{}, load: loadGL}
	s, err := c.NewSurface(h)
	if err != nil {
		c.Close()
		return nil, nil, err
	}
	if err := c.MakeCurrent(s); err != nil {
		c.Close()
		return nil, nil, err
	}
	// With vsync off, swapping never blocks (SPEC §8.11, "Frame pacing"). A
	// driver that refuses still works; it just paces frames itself.
	_ = egl.swapInterval(display, 0)

	gl, err := loadGL(egl)
	if err != nil {
		c.Close()
		return nil, nil, err
	}
	c.gl = gl
	c.damage = egl.canSwapWithDamage(display)
	c.version = gl.getString(glVersion)
	if !isES3(c.version) {
		c.Close()
		return nil, nil, errors.New(tagGLES3Unsupported, "the driver does not provide OpenGL ES 3.0", "version", c.version)
	}
	c.dev, err = newDevice(gl)
	if err != nil {
		c.Close()
		return nil, nil, err
	}
	return c, s, nil
}

// esVersion matches the GL_VERSION of an OpenGL ES context, and webglVersion that of a WebGL
// one ("WebGL 2.0 (OpenGL ES 3.0 Chromium)").
var (
	esVersion    = regexp.MustCompile(`^OpenGL ES (\d+)\.(\d+)`)
	webglVersion = regexp.MustCompile(`^WebGL (\d+)\.(\d+)`)
)

// isES3 reports whether a GL_VERSION string names OpenGL ES 3.0 or newer, or WebGL 2.0 or
// newer, which is OpenGL ES 3.0 as far as the renderer is concerned.
func isES3(version string) bool {
	if m := webglVersion.FindStringSubmatch(version); m != nil {
		major, err := strconv.Atoi(m[1])
		return err == nil && major >= 2
	}
	m := esVersion.FindStringSubmatch(version)
	if m == nil {
		return false
	}
	major, err := strconv.Atoi(m[1])
	return err == nil && major >= 3
}

// eglErrorCode reads the EGL error code of the failed call, for the metadata "egl_error" of the
// error that the failing function makes.
func eglErrorCode(egl eglAPI) string {
	return fmt.Sprintf("0x%x", egl.getError())
}

// Device returns the GL device of the context: its programs, pools and state
// cache.
//
// Flutter: no counterpart
func (c *Context) Device() *Device { return c.dev }

// Version returns the GL_VERSION string of the context.
//
// Flutter: no counterpart
func (c *Context) Version() string { return c.version }

// NewSurface creates a window surface for the native window of h. It fails with
// EGLSurfaceFailed.
//
// Flutter: no counterpart
func (c *Context) NewSurface(h gpu.SurfaceHandle) (*Surface, error) {
	if p, ok := c.egl.(windowPreparer); ok && h.Kind == gpu.SurfaceKindAndroid {
		p.prepareWindow(c.display, c.config, h.Window)
	}
	handle := c.egl.createWindowSurface(c.display, c.config, h.Window, h.Kind == gpu.SurfaceKindX11)
	if handle == 0 {
		return nil, errors.New(tagEGLSurfaceFailed, "could not create an EGL window surface", "egl_error", eglErrorCode(c.egl))
	}
	s := &Surface{ctx: c, handle: handle}
	c.surface[s] = struct{}{}
	return s, nil
}

// windowPreparer is implemented by an eglAPI that must set up a native window before a surface is
// made for it: Android's ANativeWindow takes the pixel format of the EGL config (SPEC §24.4).
type windowPreparer interface {
	prepareWindow(display, config, window uintptr)
}

// Lost says that the browser took the context away (WebGL's webglcontextlost): nothing can be
// drawn until it is restored. It is always false for an EGL context.
//
// Flutter: no counterpart
func (c *Context) Lost() bool {
	l, ok := c.egl.(lossy)
	return ok && l.contextLost()
}

// rebuild makes the device again when the context was lost and has been restored, because
// every GL object made before is gone: the old device is dropped without deleting anything.
// It reports whether it did, so that the caller drops what it made with the old device. It
// fails with the errors of the GL binding and of the device.
func (c *Context) rebuild() (bool, error) {
	l, ok := c.egl.(lossy)
	if !ok || !l.needsRebuild() {
		return false, nil
	}
	if r, ok := c.gl.(interface{ release() }); ok {
		r.release()
	}
	gl, err := c.load(c.egl)
	if err != nil {
		return false, err
	}
	dev, err := newDevice(gl)
	if err != nil {
		return false, err
	}
	c.gl, c.dev = gl, dev
	l.rebuilt()
	return true, nil
}

// surfaceSizer is implemented by an eglAPI whose surface does not follow its window by itself:
// the web's canvas has a size of its own that the renderer must set.
type surfaceSizer interface {
	setSurfaceSize(surface uintptr, width, height int)
}

// ResizeSurface makes the surface width x height pixels when it is not that size already, for
// a surface that is not resized by its window (the web's canvas, whose pixels are cleared when
// its size is set, so call it before drawing the frame). EGL surfaces follow their native
// window, so it does nothing for them.
//
// Flutter: no counterpart
func (c *Context) ResizeSurface(s *Surface, width, height int) {
	if sz, ok := c.egl.(surfaceSizer); ok {
		sz.setSurfaceSize(s.handle, width, height)
	}
}

// MakeCurrent makes the context current for drawing into s. It fails with
// EGLMakeCurrentFailed.
//
// Flutter: no counterpart
func (c *Context) MakeCurrent(s *Surface) error {
	if !c.egl.makeCurrent(c.display, s.handle, s.handle, c.context) {
		return errors.New(tagEGLMakeCurrentFailed, "eglMakeCurrent failed", "egl_error", eglErrorCode(c.egl))
	}
	c.current = s
	return nil
}

// MakeCurrentSurfaceless makes the context current without any surface, for
// drawing into framebuffer objects only (EGL_KHR_surfaceless_context, which every
// OpenGL ES 3 driver has). It fails with EGLMakeCurrentSurfacelessFailed.
//
// Flutter: no counterpart
func (c *Context) MakeCurrentSurfaceless() error {
	if !c.egl.makeCurrent(c.display, 0, 0, c.context) {
		return errors.New(tagEGLMakeCurrentSurfacelessFailed, "eglMakeCurrent without a surface failed", "egl_error", eglErrorCode(c.egl))
	}
	c.current = nil
	return nil
}

// Viewport sets the GL viewport to the whole of a width x height surface.
//
// Flutter: no counterpart
func (c *Context) Viewport(width, height int) { c.dev.Viewport(0, 0, width, height) }

// Clear fills the current framebuffer with the color, given as premultiplied
// components between 0 and 1.
//
// Flutter: no counterpart
func (c *Context) Clear(r, g, b, a float32) {
	c.dev.ClearColor(r, g, b, a)
	c.gl.clear(glColorBufferBit)
}

// SwapBuffers presents the frame drawn into s. On Wayland this is the commit of
// the wl_surface, so the frame callback must be requested before it. It fails
// with EGLSwapFailed.
//
// Flutter: no counterpart
func (c *Context) SwapBuffers(s *Surface) error {
	if !c.egl.swapBuffers(c.display, s.handle) {
		return errors.New(tagEGLSwapFailed, "eglSwapBuffers failed", "egl_error", eglErrorCode(c.egl))
	}
	return nil
}

// CanSwapWithDamage says whether the display has EGL_KHR_swap_buffers_with_damage (or
// the EXT extension), so that SwapBuffersWithDamage tells the compositor what changed.
//
// Flutter: no counterpart
func (c *Context) CanSwapWithDamage() bool { return c.damage }

// SwapBuffersWithDamage presents the frame drawn into s as SwapBuffers does, and
// tells the compositor that only the rectangles changed from the frame before it:
// pixels from the top left of a surface that is height pixels high. The whole frame
// is drawn in the buffer anyway, so a compositor or driver that ignores the damage
// shows the same window. Without the extension it is SwapBuffers. It fails with
// EGLSwapFailed.
//
// Flutter: no counterpart
func (c *Context) SwapBuffersWithDamage(s *Surface, height int, rects []image.Rectangle) error {
	if !c.damage || len(rects) == 0 {
		return c.SwapBuffers(s)
	}
	// EGL counts from the bottom left.
	flat := make([]int32, 0, len(rects)*4)
	for _, r := range rects {
		flat = append(flat, int32(r.Min.X), int32(height-r.Max.Y), int32(r.Dx()), int32(r.Dy()))
	}
	if !c.egl.swapBuffersWithDamage(c.display, s.handle, flat) {
		return errors.New(tagEGLSwapFailed, "eglSwapBuffersWithDamage failed", "egl_error", eglErrorCode(c.egl))
	}
	return nil
}

// ReadPixels reads the width x height pixels at the origin of the current
// framebuffer, top row first. Read before SwapBuffers: a framebuffer's content
// is undefined after the swap. It fails with GLReadPixelsFailed.
//
// Flutter: no counterpart
func (c *Context) ReadPixels(width, height int) (*image.RGBA, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New(tagGLReadPixelsFailed, "the size to read must be positive", "width", width, "height", height)
	}
	raw := make([]byte, width*height*4)
	c.gl.readPixels(0, 0, int32(width), int32(height), glRgba, glUnsignedByte, raw)
	if code := c.gl.getError(); code != glNoError {
		return nil, errors.New(tagGLReadPixelsFailed, "glReadPixels reported an error", "gl_error", fmt.Sprintf("0x%x", code))
	}
	// GL's first row is the bottom one; an image's first row is the top one.
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	stride := width * 4
	for y := 0; y < height; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+stride], raw[(height-1-y)*stride:(height-y)*stride])
	}
	return img, nil
}

// Destroy releases the surface. Do it before the native window goes away.
//
// Flutter: no counterpart
func (s *Surface) Destroy() {
	if s.destroyed {
		return
	}
	s.destroyed = true
	delete(s.ctx.surface, s)
	if s.ctx.current == s {
		// Leave the context current with no surface, so that it survives the surface. If EGL
		// refuses, the next MakeCurrent sets it right.
		_ = s.ctx.egl.makeCurrent(s.ctx.display, 0, 0, s.ctx.context)
		s.ctx.current = nil
	}
	// Nothing can be done if EGL refuses: the surface is gone with the display.
	_ = s.ctx.egl.destroySurface(s.ctx.display, s.handle)
}

// Close destroys the surfaces, the context and the display. Closing twice does
// nothing.
//
// Flutter: no counterpart
func (c *Context) Close() {
	if c.closed {
		return
	}
	c.closed = true
	if c.dev != nil {
		c.dev.Close()
	}
	if r, ok := c.gl.(interface{ release() }); ok {
		r.release()
	}
	_ = c.egl.makeCurrent(c.display, 0, 0, 0)
	for s := range c.surface {
		s.destroyed = true
		_ = c.egl.destroySurface(c.display, s.handle)
	}
	c.surface = nil
	_ = c.egl.destroyContext(c.display, c.context)
	_ = c.egl.terminate(c.display)
}

// x11VisualWith is X11Visual with the EGL implementation given, for tests.
func x11VisualWith(egl eglAPI, display uintptr) (uint32, error) {
	d := egl.getPlatformDisplay(eglPlatformX11, display)
	if d == 0 {
		return 0, errors.New(tagEGLInitFailed, "EGL has no display for the native display", "egl_error", eglErrorCode(egl), "step", "eglGetPlatformDisplay")
	}
	if _, _, ok := egl.initialize(d); !ok {
		return 0, errors.New(tagEGLInitFailed, "eglInitialize failed", "egl_error", eglErrorCode(egl), "step", "eglInitialize")
	}
	// The display is terminated again: EGL keeps the driver's connection to the X
	// display until then, and the Xlib display may be closed (and its address
	// reused by the next one) before Open initializes it again.
	defer egl.terminate(d)
	if !egl.bindAPI(eglOpenGLESAPI) {
		return 0, errors.New(tagGLES3Unsupported, "this EGL has no OpenGL ES API", "egl_error", eglErrorCode(egl), "step", "eglBindAPI")
	}
	config, ok := egl.chooseConfig(d, configAttribs)
	if !ok {
		return 0, errors.New(tagEGLNoConfig, "no EGL config has RGBA8 and OpenGL ES 3 window surfaces", "egl_error", eglErrorCode(egl))
	}
	id, ok := egl.getConfigAttrib(d, config, eglNativeVisualID)
	if !ok || id == 0 {
		return 0, errors.New(tagEGLConfigQueryFailed, "the EGL config has no native visual", "egl_error", eglErrorCode(egl), "step", "eglGetConfigAttrib")
	}
	return uint32(id), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEGLConfigQueryFailed            = "EGLConfigQueryFailed"
	tagEGLInitFailed                   = "EGLInitFailed"
	tagEGLMakeCurrentFailed            = "EGLMakeCurrentFailed"
	tagEGLMakeCurrentSurfacelessFailed = "EGLMakeCurrentSurfacelessFailed"
	tagEGLNoConfig                     = "EGLNoConfig"
	tagEGLSurfaceFailed                = "EGLSurfaceFailed"
	tagEGLSwapFailed                   = "EGLSwapFailed"
	tagGLES3Unsupported                = "GLES3Unsupported"
	tagGLReadPixelsFailed              = "GLReadPixelsFailed"
	// tagSymbolMissing is made by the loaders of the GL functions (gl_purego.go, gl_js.go) and
	// used by the tests of this file, which build on every OS.
	tagSymbolMissing = "SymbolMissing"
)
