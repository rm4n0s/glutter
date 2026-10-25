//go:build linux

package gles

import (
	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// glLib is the purego implementation of glAPI. Each field is one driver function,
// bound by name in the table of loadGL.
type glLib struct {
	getStringFn   func(name uint32) uintptr
	getStringiFn  func(name, index uint32) uintptr
	getIntegervFn func(name uint32, data uintptr)
	getErrorFn    func() uint32

	viewportFn          func(x, y, w, h int32)
	scissorFn           func(x, y, w, h int32)
	enableFn            func(c uint32)
	disableFn           func(c uint32)
	blendFuncFn         func(src, dst uint32)
	blendFuncSeparateFn func(srcRGB, dstRGB, srcA, dstA uint32)
	blendEquationFn     func(mode uint32)
	colorMaskFn         func(r, g, b, a uint32)
	clearColorFn        func(r, g, b, a float32)
	clearStencilFn      func(s int32)
	clearFn             func(mask uint32)
	stencilFuncFn       func(fn uint32, ref int32, mask uint32)
	stencilOpFn         func(fail, zfail, zpass uint32)
	stencilMaskFn       func(mask uint32)
	pixelStoreiFn       func(name uint32, v int32)

	createShaderFn       func(kind uint32) uint32
	shaderSourceFn       func(shader uint32, count int32, strs, lengths uintptr)
	compileShaderFn      func(shader uint32)
	getShaderivFn        func(shader, name uint32, params uintptr)
	getShaderInfoLogFn   func(shader uint32, bufSize int32, length, log uintptr)
	deleteShaderFn       func(shader uint32)
	createProgramFn      func() uint32
	attachShaderFn       func(program, shader uint32)
	linkProgramFn        func(program uint32)
	getProgramivFn       func(program, name uint32, params uintptr)
	getProgramInfoLogFn  func(program uint32, bufSize int32, length, log uintptr)
	getProgramBinaryFn   func(program uint32, bufSize int32, length, format, binary uintptr)
	programBinaryFn      func(program, format uint32, binary uintptr, length int32)
	programParameteriFn  func(program, name uint32, value int32)
	useProgramFn         func(program uint32)
	deleteProgramFn      func(program uint32)
	getUniformLocationFn func(program uint32, name uintptr) int32
	getAttribLocationFn  func(program uint32, name uintptr) int32
	uniform1iFn          func(loc, v int32)
	uniform1fFn          func(loc int32, v float32)
	uniform2fFn          func(loc int32, x, y float32)
	uniform3fFn          func(loc int32, x, y, z float32)
	uniform4fFn          func(loc int32, x, y, z, w float32)
	uniform1fvFn         func(loc, count int32, v uintptr)
	uniform4fvFn         func(loc, count int32, v uintptr)
	uniformMatrix3fvFn   func(loc, count int32, transpose uint32, v uintptr)
	uniformMatrix4fvFn   func(loc, count int32, transpose uint32, v uintptr)

	genBuffersFn               func(n int32, buffers uintptr)
	deleteBuffersFn            func(n int32, buffers uintptr)
	bindBufferFn               func(target, buffer uint32)
	bufferDataFn               func(target uint32, size int, data uintptr, usage uint32)
	bufferSubDataFn            func(target uint32, offset, size int, data uintptr)
	genVertexArraysFn          func(n int32, arrays uintptr)
	deleteVertexArraysFn       func(n int32, arrays uintptr)
	bindVertexArrayFn          func(array uint32)
	enableVertexAttribArrayFn  func(index uint32)
	disableVertexAttribArrayFn func(index uint32)
	vertexAttribPointerFn      func(index uint32, size int32, kind uint32, normalized uint32, stride int32, offset uintptr)
	vertexAttribIPointerFn     func(index uint32, size int32, kind uint32, stride int32, offset uintptr)
	vertexAttribDivisorFn      func(index, divisor uint32)
	drawArraysFn               func(mode uint32, first, count int32)
	drawArraysInstancedFn      func(mode uint32, first, count, instances int32)
	drawElementsFn             func(mode uint32, count int32, kind uint32, offset uintptr)
	drawElementsInstancedFn    func(mode uint32, count int32, kind uint32, offset uintptr, instances int32)

	genTexturesFn       func(n int32, textures uintptr)
	deleteTexturesFn    func(n int32, textures uintptr)
	activeTextureFn     func(unit uint32)
	bindTextureFn       func(target, texture uint32)
	texParameteriFn     func(target, name uint32, value int32)
	texImage2DFn        func(target uint32, level, internalFormat, w, h, border int32, format, kind uint32, data uintptr)
	texSubImage2DFn     func(target uint32, level, x, y, w, h int32, format, kind uint32, data uintptr)
	generateMipmapFn    func(target uint32)
	copyTexSubImage2DFn func(target uint32, level, xoffset, yoffset, x, y, w, h int32)

	genFramebuffersFn         func(n int32, ids uintptr)
	deleteFramebuffersFn      func(n int32, ids uintptr)
	bindFramebufferFn         func(target, fb uint32)
	framebufferTexture2DFn    func(target, attachment, texTarget, texture uint32, level int32)
	checkFramebufferStatusFn  func(target uint32) uint32
	genRenderbuffersFn        func(n int32, ids uintptr)
	deleteRenderbuffersFn     func(n int32, ids uintptr)
	bindRenderbufferFn        func(target, rb uint32)
	renderbufferStorageFn     func(target, format uint32, w, h int32)
	framebufferRenderbufferFn func(target, attachment, rbTarget, rb uint32)
	blitFramebufferFn         func(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32)
	readPixelsFn              func(x, y, w, h int32, format, kind uint32, pixels uintptr)

	flushFn  func()
	finishFn func()

	// scratch is C memory that Go data is copied into for a call: Go memory is
	// never handed to the driver. It grows and is freed by release.
	scratch    uintptr
	scratchCap int
	// ids is a small C buffer for the IDs that gen and delete functions pass.
	ids uintptr
}

// glSymbols is the table of every driver function the renderer binds, in one
// place so that a missing one is a clear SymbolMissing (docs/IMPLEMENTATION.md
// step 3.1).
func (g *glLib) glSymbols() []struct {
	fptr   any
	symbol string
} {
	return []struct {
		fptr   any
		symbol string
	}{
		{&g.getStringFn, "glGetString"}, {&g.getStringiFn, "glGetStringi"}, {&g.getIntegervFn, "glGetIntegerv"}, {&g.getErrorFn, "glGetError"},
		{&g.viewportFn, "glViewport"}, {&g.scissorFn, "glScissor"}, {&g.enableFn, "glEnable"}, {&g.disableFn, "glDisable"},
		{&g.blendFuncFn, "glBlendFunc"}, {&g.blendFuncSeparateFn, "glBlendFuncSeparate"}, {&g.blendEquationFn, "glBlendEquation"},
		{&g.colorMaskFn, "glColorMask"}, {&g.clearColorFn, "glClearColor"}, {&g.clearStencilFn, "glClearStencil"}, {&g.clearFn, "glClear"},
		{&g.stencilFuncFn, "glStencilFunc"}, {&g.stencilOpFn, "glStencilOp"}, {&g.stencilMaskFn, "glStencilMask"}, {&g.pixelStoreiFn, "glPixelStorei"},
		{&g.createShaderFn, "glCreateShader"}, {&g.shaderSourceFn, "glShaderSource"}, {&g.compileShaderFn, "glCompileShader"},
		{&g.getShaderivFn, "glGetShaderiv"}, {&g.getShaderInfoLogFn, "glGetShaderInfoLog"}, {&g.deleteShaderFn, "glDeleteShader"},
		{&g.createProgramFn, "glCreateProgram"}, {&g.attachShaderFn, "glAttachShader"}, {&g.linkProgramFn, "glLinkProgram"},
		{&g.getProgramivFn, "glGetProgramiv"}, {&g.getProgramInfoLogFn, "glGetProgramInfoLog"}, {&g.useProgramFn, "glUseProgram"},
		{&g.getProgramBinaryFn, "glGetProgramBinary"}, {&g.programBinaryFn, "glProgramBinary"}, {&g.programParameteriFn, "glProgramParameteri"},
		{&g.deleteProgramFn, "glDeleteProgram"}, {&g.getUniformLocationFn, "glGetUniformLocation"}, {&g.getAttribLocationFn, "glGetAttribLocation"},
		{&g.uniform1iFn, "glUniform1i"}, {&g.uniform1fFn, "glUniform1f"}, {&g.uniform2fFn, "glUniform2f"}, {&g.uniform3fFn, "glUniform3f"},
		{&g.uniform4fFn, "glUniform4f"}, {&g.uniform1fvFn, "glUniform1fv"}, {&g.uniform4fvFn, "glUniform4fv"},
		{&g.uniformMatrix3fvFn, "glUniformMatrix3fv"}, {&g.uniformMatrix4fvFn, "glUniformMatrix4fv"},
		{&g.genBuffersFn, "glGenBuffers"}, {&g.deleteBuffersFn, "glDeleteBuffers"}, {&g.bindBufferFn, "glBindBuffer"},
		{&g.bufferDataFn, "glBufferData"}, {&g.bufferSubDataFn, "glBufferSubData"},
		{&g.genVertexArraysFn, "glGenVertexArrays"}, {&g.deleteVertexArraysFn, "glDeleteVertexArrays"}, {&g.bindVertexArrayFn, "glBindVertexArray"},
		{&g.enableVertexAttribArrayFn, "glEnableVertexAttribArray"}, {&g.disableVertexAttribArrayFn, "glDisableVertexAttribArray"},
		{&g.vertexAttribPointerFn, "glVertexAttribPointer"}, {&g.vertexAttribIPointerFn, "glVertexAttribIPointer"}, {&g.vertexAttribDivisorFn, "glVertexAttribDivisor"},
		{&g.drawArraysFn, "glDrawArrays"}, {&g.drawArraysInstancedFn, "glDrawArraysInstanced"},
		{&g.drawElementsFn, "glDrawElements"}, {&g.drawElementsInstancedFn, "glDrawElementsInstanced"},
		{&g.genTexturesFn, "glGenTextures"}, {&g.deleteTexturesFn, "glDeleteTextures"}, {&g.activeTextureFn, "glActiveTexture"},
		{&g.bindTextureFn, "glBindTexture"}, {&g.texParameteriFn, "glTexParameteri"}, {&g.texImage2DFn, "glTexImage2D"},
		{&g.texSubImage2DFn, "glTexSubImage2D"}, {&g.generateMipmapFn, "glGenerateMipmap"}, {&g.copyTexSubImage2DFn, "glCopyTexSubImage2D"},
		{&g.genFramebuffersFn, "glGenFramebuffers"}, {&g.deleteFramebuffersFn, "glDeleteFramebuffers"}, {&g.bindFramebufferFn, "glBindFramebuffer"},
		{&g.framebufferTexture2DFn, "glFramebufferTexture2D"}, {&g.checkFramebufferStatusFn, "glCheckFramebufferStatus"},
		{&g.genRenderbuffersFn, "glGenRenderbuffers"}, {&g.deleteRenderbuffersFn, "glDeleteRenderbuffers"}, {&g.bindRenderbufferFn, "glBindRenderbuffer"},
		{&g.renderbufferStorageFn, "glRenderbufferStorage"}, {&g.framebufferRenderbufferFn, "glFramebufferRenderbuffer"},
		{&g.blitFramebufferFn, "glBlitFramebuffer"}, {&g.readPixelsFn, "glReadPixels"},
		{&g.flushFn, "glFlush"}, {&g.finishFn, "glFinish"},
	}
}

// loadGL binds the GL functions that the renderer uses. They are resolved through
// eglGetProcAddress (SPEC §9.1), which needs a current context on older drivers,
// so call it after makeCurrent. A function that no driver entry point provides
// gives SymbolMissing, naming it.
func loadGL(egl eglAPI) (glAPI, error) {
	g := new(glLib)
	for _, b := range g.glSymbols() {
		address := egl.getProcAddress(b.symbol)
		if address == 0 {
			return nil, errors.New(tagSymbolMissing, "the GL driver has no such function",
				"library", "eglGetProcAddress", "symbol", b.symbol)
		}
		purego.RegisterFunc(b.fptr, address)
	}
	g.ids = dl.Calloc(1, 16)
	return g, nil
}

// release frees the C memory the library holds.
func (g *glLib) release() {
	if g.scratch != 0 {
		dl.Free(g.scratch)
		g.scratch, g.scratchCap = 0, 0
	}
	if g.ids != 0 {
		dl.Free(g.ids)
		g.ids = 0
	}
}

// put copies data into the scratch memory and returns its address; 0 for no data.
func (g *glLib) put(data []byte) uintptr {
	if len(data) == 0 {
		return 0
	}
	if len(data) > g.scratchCap {
		if g.scratch != 0 {
			dl.Free(g.scratch)
		}
		g.scratchCap = max(len(data), 2*g.scratchCap, 4096)
		g.scratch = dl.Malloc(uintptr(g.scratchCap))
	}
	copy(dl.Bytes(g.scratch, len(data)), data)
	return g.scratch
}

func b2u(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// genOne calls a glGen* function for one object and returns its ID.
func (g *glLib) genOne(fn func(n int32, ids uintptr)) uint32 {
	fn(1, g.ids)
	return *(*uint32)(dl.Pointer(g.ids))
}

// deleteOne calls a glDelete* function for one ID.
func (g *glLib) deleteOne(fn func(n int32, ids uintptr), id uint32) {
	*(*uint32)(dl.Pointer(g.ids)) = id
	fn(1, g.ids)
}

func (g *glLib) getString(name uint32) string { return dl.GoString(g.getStringFn(name)) }
func (g *glLib) getStringi(name, index uint32) string {
	return dl.GoString(g.getStringiFn(name, index))
}
func (g *glLib) getInteger(name uint32) int32 {
	*(*int32)(dl.Pointer(g.ids)) = 0
	g.getIntegervFn(name, g.ids)
	return *(*int32)(dl.Pointer(g.ids))
}
func (g *glLib) getError() uint32 { return g.getErrorFn() }

func (g *glLib) viewport(x, y, w, h int32)                  { g.viewportFn(x, y, w, h) }
func (g *glLib) scissor(x, y, w, h int32)                   { g.scissorFn(x, y, w, h) }
func (g *glLib) enable(c uint32)                            { g.enableFn(c) }
func (g *glLib) disable(c uint32)                           { g.disableFn(c) }
func (g *glLib) blendFunc(src, dst uint32)                  { g.blendFuncFn(src, dst) }
func (g *glLib) blendFuncSeparate(a, b, c, d uint32)        { g.blendFuncSeparateFn(a, b, c, d) }
func (g *glLib) blendEquation(mode uint32)                  { g.blendEquationFn(mode) }
func (g *glLib) colorMask(r, gr, b, a bool)                 { g.colorMaskFn(b2u(r), b2u(gr), b2u(b), b2u(a)) }
func (g *glLib) clearColor(r, gr, b, a float32)             { g.clearColorFn(r, gr, b, a) }
func (g *glLib) clearStencil(s int32)                       { g.clearStencilFn(s) }
func (g *glLib) clear(mask uint32)                          { g.clearFn(mask) }
func (g *glLib) stencilFunc(fn uint32, ref int32, m uint32) { g.stencilFuncFn(fn, ref, m) }
func (g *glLib) stencilOp(fail, zfail, zpass uint32)        { g.stencilOpFn(fail, zfail, zpass) }
func (g *glLib) stencilMask(mask uint32)                    { g.stencilMaskFn(mask) }
func (g *glLib) pixelStorei(name uint32, v int32)           { g.pixelStoreiFn(name, v) }

func (g *glLib) createShader(kind uint32) uint32 { return g.createShaderFn(kind) }
func (g *glLib) shaderSource(shader uint32, source string) {
	str := dl.CString(source)
	defer dl.Free(str)
	// glShaderSource takes an array of string pointers; one string, null
	// terminated, so no lengths.
	*(*uintptr)(dl.Pointer(g.ids)) = str
	g.shaderSourceFn(shader, 1, g.ids, 0)
}
func (g *glLib) compileShader(shader uint32) { g.compileShaderFn(shader) }
func (g *glLib) getShaderiv(shader, name uint32) int32 {
	*(*int32)(dl.Pointer(g.ids)) = 0
	g.getShaderivFn(shader, name, g.ids)
	return *(*int32)(dl.Pointer(g.ids))
}
func (g *glLib) getShaderInfoLog(shader uint32) string {
	n := g.getShaderiv(shader, glInfoLogLength)
	return g.infoLog(n, func(size int32, length, log uintptr) { g.getShaderInfoLogFn(shader, size, length, log) })
}
func (g *glLib) deleteShader(shader uint32) { g.deleteShaderFn(shader) }
func (g *glLib) createProgram() uint32      { return g.createProgramFn() }
func (g *glLib) attachShader(p, s uint32)   { g.attachShaderFn(p, s) }
func (g *glLib) linkProgram(p uint32)       { g.linkProgramFn(p) }
func (g *glLib) getProgramiv(program, name uint32) int32 {
	*(*int32)(dl.Pointer(g.ids)) = 0
	g.getProgramivFn(program, name, g.ids)
	return *(*int32)(dl.Pointer(g.ids))
}
func (g *glLib) getProgramInfoLog(program uint32) string {
	n := g.getProgramiv(program, glInfoLogLength)
	return g.infoLog(n, func(size int32, length, log uintptr) { g.getProgramInfoLogFn(program, size, length, log) })
}

func (g *glLib) programBinaryFormats() int32 {
	*(*int32)(dl.Pointer(g.ids)) = 0
	g.getIntegervFn(glNumProgramBinaryFormats, g.ids)
	return *(*int32)(dl.Pointer(g.ids))
}

func (g *glLib) getProgramBinary(program uint32) (uint32, []byte, bool) {
	n := g.getProgramiv(program, glProgramBinaryLength)
	if n <= 0 {
		return 0, nil, false
	}
	binary := dl.Malloc(uintptr(n))
	defer dl.Free(binary)
	length, format := g.ids+8, g.ids+12
	*(*int32)(dl.Pointer(length)), *(*uint32)(dl.Pointer(format)) = 0, 0
	g.getProgramBinaryFn(program, n, length, format, binary)
	got := int(*(*int32)(dl.Pointer(length)))
	if got <= 0 || got > int(n) {
		return 0, nil, false
	}
	return *(*uint32)(dl.Pointer(format)), append([]byte(nil), dl.Bytes(binary, got)...), true
}

func (g *glLib) programBinary(program, format uint32, data []byte) {
	g.programBinaryFn(program, format, g.put(data), int32(len(data)))
}

func (g *glLib) programParameteri(program, name uint32, value int32) {
	g.programParameteriFn(program, name, value)
}

// infoLog reads an info log of up to n bytes through the callback.
func (g *glLib) infoLog(n int32, read func(size int32, length, log uintptr)) string {
	if n <= 1 {
		return ""
	}
	log := dl.Malloc(uintptr(n))
	defer dl.Free(log)
	length := g.ids + 8
	*(*int32)(dl.Pointer(length)) = 0
	read(n, length, log)
	got := int(*(*int32)(dl.Pointer(length)))
	return string(dl.Bytes(log, min(got, int(n))))
}

func (g *glLib) useProgram(p uint32)    { g.useProgramFn(p) }
func (g *glLib) deleteProgram(p uint32) { g.deleteProgramFn(p) }
func (g *glLib) getUniformLocation(program uint32, name string) int32 {
	s := dl.CString(name)
	defer dl.Free(s)
	return g.getUniformLocationFn(program, s)
}
func (g *glLib) getAttribLocation(program uint32, name string) int32 {
	s := dl.CString(name)
	defer dl.Free(s)
	return g.getAttribLocationFn(program, s)
}
func (g *glLib) uniform1i(loc, v int32)                  { g.uniform1iFn(loc, v) }
func (g *glLib) uniform1f(loc int32, v float32)          { g.uniform1fFn(loc, v) }
func (g *glLib) uniform2f(loc int32, x, y float32)       { g.uniform2fFn(loc, x, y) }
func (g *glLib) uniform3f(loc int32, x, y, z float32)    { g.uniform3fFn(loc, x, y, z) }
func (g *glLib) uniform4f(loc int32, x, y, z, w float32) { g.uniform4fFn(loc, x, y, z, w) }
func (g *glLib) uniform1fv(loc int32, v []float32) {
	g.uniform1fvFn(loc, int32(len(v)), g.put(floatBytes(v)))
}
func (g *glLib) uniform4fv(loc int32, v []float32) {
	g.uniform4fvFn(loc, int32(len(v)/4), g.put(floatBytes(v)))
}
func (g *glLib) uniformMatrix3fv(loc int32, v []float32) {
	g.uniformMatrix3fvFn(loc, int32(len(v)/9), 0, g.put(floatBytes(v)))
}
func (g *glLib) uniformMatrix4fv(loc int32, v []float32) {
	g.uniformMatrix4fvFn(loc, int32(len(v)/16), 0, g.put(floatBytes(v)))
}

func (g *glLib) genBuffer() uint32           { return g.genOne(g.genBuffersFn) }
func (g *glLib) deleteBuffer(b uint32)       { g.deleteOne(g.deleteBuffersFn, b) }
func (g *glLib) bindBuffer(target, b uint32) { g.bindBufferFn(target, b) }
func (g *glLib) bufferData(target uint32, size int, data []byte, usage uint32) {
	g.bufferDataFn(target, size, g.put(data), usage)
}
func (g *glLib) bufferSubData(target uint32, offset int, data []byte) {
	g.bufferSubDataFn(target, offset, len(data), g.put(data))
}
func (g *glLib) genVertexArray() uint32            { return g.genOne(g.genVertexArraysFn) }
func (g *glLib) deleteVertexArray(a uint32)        { g.deleteOne(g.deleteVertexArraysFn, a) }
func (g *glLib) bindVertexArray(a uint32)          { g.bindVertexArrayFn(a) }
func (g *glLib) enableVertexAttribArray(i uint32)  { g.enableVertexAttribArrayFn(i) }
func (g *glLib) disableVertexAttribArray(i uint32) { g.disableVertexAttribArrayFn(i) }
func (g *glLib) vertexAttribPointer(index uint32, size int32, kind uint32, normalized bool, stride int32, offset int) {
	g.vertexAttribPointerFn(index, size, kind, b2u(normalized), stride, uintptr(offset))
}
func (g *glLib) vertexAttribIPointer(index uint32, size int32, kind uint32, stride int32, offset int) {
	g.vertexAttribIPointerFn(index, size, kind, stride, uintptr(offset))
}
func (g *glLib) vertexAttribDivisor(index, divisor uint32)  { g.vertexAttribDivisorFn(index, divisor) }
func (g *glLib) drawArrays(mode uint32, first, count int32) { g.drawArraysFn(mode, first, count) }
func (g *glLib) drawArraysInstanced(mode uint32, first, count, instances int32) {
	g.drawArraysInstancedFn(mode, first, count, instances)
}
func (g *glLib) drawElements(mode uint32, count int32, kind uint32, offset int) {
	g.drawElementsFn(mode, count, kind, uintptr(offset))
}
func (g *glLib) drawElementsInstanced(mode uint32, count int32, kind uint32, offset int, instances int32) {
	g.drawElementsInstancedFn(mode, count, kind, uintptr(offset), instances)
}

func (g *glLib) genTexture() uint32                         { return g.genOne(g.genTexturesFn) }
func (g *glLib) deleteTexture(t uint32)                     { g.deleteOne(g.deleteTexturesFn, t) }
func (g *glLib) activeTexture(unit uint32)                  { g.activeTextureFn(unit) }
func (g *glLib) bindTexture(target, t uint32)               { g.bindTextureFn(target, t) }
func (g *glLib) texParameteri(target, name uint32, v int32) { g.texParameteriFn(target, name, v) }
func (g *glLib) texImage2D(target uint32, level, internalFormat, w, h int32, format, kind uint32, data []byte) {
	g.texImage2DFn(target, level, internalFormat, w, h, 0, format, kind, g.put(data))
}
func (g *glLib) texSubImage2D(target uint32, level, x, y, w, h int32, format, kind uint32, data []byte) {
	g.texSubImage2DFn(target, level, x, y, w, h, format, kind, g.put(data))
}
func (g *glLib) generateMipmap(target uint32) { g.generateMipmapFn(target) }
func (g *glLib) copyTexSubImage2D(target uint32, level, xo, yo, x, y, w, h int32) {
	g.copyTexSubImage2DFn(target, level, xo, yo, x, y, w, h)
}

func (g *glLib) genFramebuffer() uint32           { return g.genOne(g.genFramebuffersFn) }
func (g *glLib) deleteFramebuffer(f uint32)       { g.deleteOne(g.deleteFramebuffersFn, f) }
func (g *glLib) bindFramebuffer(target, f uint32) { g.bindFramebufferFn(target, f) }
func (g *glLib) framebufferTexture2D(target, attachment, texTarget, texture uint32, level int32) {
	g.framebufferTexture2DFn(target, attachment, texTarget, texture, level)
}
func (g *glLib) checkFramebufferStatus(target uint32) uint32 {
	return g.checkFramebufferStatusFn(target)
}
func (g *glLib) genRenderbuffer() uint32           { return g.genOne(g.genRenderbuffersFn) }
func (g *glLib) deleteRenderbuffer(r uint32)       { g.deleteOne(g.deleteRenderbuffersFn, r) }
func (g *glLib) bindRenderbuffer(target, r uint32) { g.bindRenderbufferFn(target, r) }
func (g *glLib) renderbufferStorage(target, format uint32, w, h int32) {
	g.renderbufferStorageFn(target, format, w, h)
}
func (g *glLib) framebufferRenderbuffer(target, attachment, rbTarget, rb uint32) {
	g.framebufferRenderbufferFn(target, attachment, rbTarget, rb)
}
func (g *glLib) blitFramebuffer(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {
	g.blitFramebufferFn(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1, mask, filter)
}
func (g *glLib) readPixels(x, y, w, h int32, format, kind uint32, pixels []byte) {
	mem := dl.Malloc(uintptr(len(pixels)))
	defer dl.Free(mem)
	g.readPixelsFn(x, y, w, h, format, kind, mem)
	copy(pixels, dl.Bytes(mem, len(pixels)))
}

func (g *glLib) flush()  { g.flushFn() }
func (g *glLib) finish() { g.finishFn() }
