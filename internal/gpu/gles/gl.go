package gles

import (
	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// GL constants (GLES3/gl3.h).
const (
	glNoError        = 0
	glVersion        = 0x1F02
	glColorBufferBit = 0x4000
	glRGBA           = 0x1908
	glUnsignedByte   = 0x1401
)

// glAPI is the part of OpenGL ES that the context uses, in Go terms. The real
// implementation calls the driver's functions through purego; tests supply a
// fake.
type glAPI interface {
	// version returns the GL_VERSION string.
	version() string
	viewport(width, height int32)
	clearColor(r, g, b, a float32)
	clear()
	// readPixels reads the width x height pixels at the origin of the current
	// framebuffer as RGBA, bottom row first, into pixels.
	readPixels(width, height int32, pixels []byte)
	getError() uint32
}

// glLib is the purego implementation of glAPI.
type glLib struct {
	getStringFn  func(name uint32) uintptr
	clearColorFn func(r, g, b, a float32)
	clearFn      func(mask uint32)
	viewportFn   func(x, y, w, h int32)
	readPixelsFn func(x, y, w, h int32, format, typ uint32, pixels uintptr)
	getErrorFn   func() uint32
}

// loadGL binds the GL functions that the context uses. They are resolved
// through eglGetProcAddress (SPEC §9.1), which needs a current context on older
// drivers, so call it after makeCurrent. A function that no driver entry point
// provides gives SymbolMissing.
func loadGL(egl eglAPI) (glAPI, error) {
	g := new(glLib)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&g.getStringFn, "glGetString"},
		{&g.clearColorFn, "glClearColor"},
		{&g.clearFn, "glClear"},
		{&g.viewportFn, "glViewport"},
		{&g.readPixelsFn, "glReadPixels"},
		{&g.getErrorFn, "glGetError"},
	} {
		address := egl.getProcAddress(b.symbol)
		if address == 0 {
			return nil, errors.New("SymbolMissing", "the GL driver has no such function",
				"library", "eglGetProcAddress", "symbol", b.symbol)
		}
		purego.RegisterFunc(b.fptr, address)
	}
	return g, nil
}

func (g *glLib) version() string { return dl.GoString(g.getStringFn(glVersion)) }

func (g *glLib) viewport(width, height int32) { g.viewportFn(0, 0, width, height) }

func (g *glLib) clearColor(r, gr, b, a float32) { g.clearColorFn(r, gr, b, a) }

func (g *glLib) clear() { g.clearFn(glColorBufferBit) }

func (g *glLib) readPixels(width, height int32, pixels []byte) {
	mem := dl.Malloc(uintptr(len(pixels)))
	defer dl.Free(mem)
	g.readPixelsFn(0, 0, width, height, glRGBA, glUnsignedByte, mem)
	copy(pixels, dl.Bytes(mem, len(pixels)))
}

func (g *glLib) getError() uint32 { return g.getErrorFn() }
