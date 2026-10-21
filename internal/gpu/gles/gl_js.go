//go:build js && wasm

package gles

import (
	"syscall/js"

	"github.com/rm4n0s/errors"
)

// glJS is the WebGL 2 implementation of glAPI: the same interface as glLib, with the calls made
// into JavaScript instead of into a driver (SPEC §9.1). The shaders are the GLSL ES 3.00 sources
// as they are, which WebGL 2 accepts.
//
// Each call from Go into JavaScript costs about a microsecond (SPEC §22), so:
//   - every method of the context is bound once, when the binding is made, and called with
//     Invoke, which skips the lookup by name that Call does on every call
//   - the data of an upload goes through one Uint8Array that is reused and only grows, by one
//     js.CopyBytesToJS and one call; a read-back through the same array by one call and one
//     js.CopyBytesToGo
//   - a uniform of one value is set with the scalar function, which needs no array
//
// WebGL's objects are JavaScript values, and the renderer names every object by a uint32 as
// OpenGL does, so each object is kept in a table of this binding and its index is its name;
// 0 is null. A uniform location is kept the same way, and -1 is no location.
type glJS struct {
	gl      js.Value
	f       glFns
	objs    handleTable[js.Value]
	locs    handleTable[js.Value]
	scratch jsScratch

	// WebGL has no glGetStringi and no GL_NUM_EXTENSIONS: the list is read once.
	extensions []string
	// renderer is the unmasked name of the GPU when the browser gives it, else the masked one.
	renderer string
}

var _ glAPI = (*glJS)(nil)

// glFns are the methods of the WebGL2RenderingContext that the binding calls, each already bound
// to the context.
type glFns struct {
	getParameter             js.Value
	getError                 js.Value
	viewport                 js.Value
	scissor                  js.Value
	enable                   js.Value
	disable                  js.Value
	blendFunc                js.Value
	blendFuncSeparate        js.Value
	blendEquation            js.Value
	colorMask                js.Value
	clearColor               js.Value
	clearStencil             js.Value
	clear                    js.Value
	stencilFunc              js.Value
	stencilOp                js.Value
	stencilMask              js.Value
	pixelStorei              js.Value
	createShader             js.Value
	shaderSource             js.Value
	compileShader            js.Value
	getShaderParameter       js.Value
	getShaderInfoLog         js.Value
	deleteShader             js.Value
	createProgram            js.Value
	attachShader             js.Value
	linkProgram              js.Value
	getProgramParameter      js.Value
	getProgramInfoLog        js.Value
	useProgram               js.Value
	deleteProgram            js.Value
	getUniformLocation       js.Value
	getAttribLocation        js.Value
	uniform1i                js.Value
	uniform1f                js.Value
	uniform2f                js.Value
	uniform3f                js.Value
	uniform4f                js.Value
	uniform1fv               js.Value
	uniform4fv               js.Value
	uniformMatrix3fv         js.Value
	uniformMatrix4fv         js.Value
	createBuffer             js.Value
	deleteBuffer             js.Value
	bindBuffer               js.Value
	bufferData               js.Value
	bufferSubData            js.Value
	createVertexArray        js.Value
	deleteVertexArray        js.Value
	bindVertexArray          js.Value
	enableVertexAttribArray  js.Value
	disableVertexAttribArray js.Value
	vertexAttribPointer      js.Value
	vertexAttribIPointer     js.Value
	vertexAttribDivisor      js.Value
	drawArrays               js.Value
	drawArraysInstanced      js.Value
	drawElements             js.Value
	drawElementsInstanced    js.Value
	createTexture            js.Value
	deleteTexture            js.Value
	activeTexture            js.Value
	bindTexture              js.Value
	texParameteri            js.Value
	texImage2D               js.Value
	texSubImage2D            js.Value
	generateMipmap           js.Value
	copyTexSubImage2D        js.Value
	createFramebuffer        js.Value
	deleteFramebuffer        js.Value
	bindFramebuffer          js.Value
	framebufferTexture2D     js.Value
	checkFramebufferStatus   js.Value
	createRenderbuffer       js.Value
	deleteRenderbuffer       js.Value
	bindRenderbuffer         js.Value
	renderbufferStorage      js.Value
	framebufferRenderbuffer  js.Value
	blitFramebuffer          js.Value
	readPixels               js.Value
	flush                    js.Value
	finish                   js.Value
}

// loadWebGL binds the methods of the context and reads what it says about itself. A context
// that lacks one of the functions of a WebGL 2 context gives SymbolMissing.
func loadWebGL(gl js.Value) (*glJS, error) {
	var missing string
	bind := func(name string) js.Value {
		fn := gl.Get(name)
		if fn.Type() != js.TypeFunction {
			if missing == "" {
				missing = name
			}
			return js.Undefined()
		}
		return fn.Call("bind", gl)
	}
	g := &glJS{gl: gl}
	g.f = glFns{
		getParameter:             bind("getParameter"),
		getError:                 bind("getError"),
		viewport:                 bind("viewport"),
		scissor:                  bind("scissor"),
		enable:                   bind("enable"),
		disable:                  bind("disable"),
		blendFunc:                bind("blendFunc"),
		blendFuncSeparate:        bind("blendFuncSeparate"),
		blendEquation:            bind("blendEquation"),
		colorMask:                bind("colorMask"),
		clearColor:               bind("clearColor"),
		clearStencil:             bind("clearStencil"),
		clear:                    bind("clear"),
		stencilFunc:              bind("stencilFunc"),
		stencilOp:                bind("stencilOp"),
		stencilMask:              bind("stencilMask"),
		pixelStorei:              bind("pixelStorei"),
		createShader:             bind("createShader"),
		shaderSource:             bind("shaderSource"),
		compileShader:            bind("compileShader"),
		getShaderParameter:       bind("getShaderParameter"),
		getShaderInfoLog:         bind("getShaderInfoLog"),
		deleteShader:             bind("deleteShader"),
		createProgram:            bind("createProgram"),
		attachShader:             bind("attachShader"),
		linkProgram:              bind("linkProgram"),
		getProgramParameter:      bind("getProgramParameter"),
		getProgramInfoLog:        bind("getProgramInfoLog"),
		useProgram:               bind("useProgram"),
		deleteProgram:            bind("deleteProgram"),
		getUniformLocation:       bind("getUniformLocation"),
		getAttribLocation:        bind("getAttribLocation"),
		uniform1i:                bind("uniform1i"),
		uniform1f:                bind("uniform1f"),
		uniform2f:                bind("uniform2f"),
		uniform3f:                bind("uniform3f"),
		uniform4f:                bind("uniform4f"),
		uniform1fv:               bind("uniform1fv"),
		uniform4fv:               bind("uniform4fv"),
		uniformMatrix3fv:         bind("uniformMatrix3fv"),
		uniformMatrix4fv:         bind("uniformMatrix4fv"),
		createBuffer:             bind("createBuffer"),
		deleteBuffer:             bind("deleteBuffer"),
		bindBuffer:               bind("bindBuffer"),
		bufferData:               bind("bufferData"),
		bufferSubData:            bind("bufferSubData"),
		createVertexArray:        bind("createVertexArray"),
		deleteVertexArray:        bind("deleteVertexArray"),
		bindVertexArray:          bind("bindVertexArray"),
		enableVertexAttribArray:  bind("enableVertexAttribArray"),
		disableVertexAttribArray: bind("disableVertexAttribArray"),
		vertexAttribPointer:      bind("vertexAttribPointer"),
		vertexAttribIPointer:     bind("vertexAttribIPointer"),
		vertexAttribDivisor:      bind("vertexAttribDivisor"),
		drawArrays:               bind("drawArrays"),
		drawArraysInstanced:      bind("drawArraysInstanced"),
		drawElements:             bind("drawElements"),
		drawElementsInstanced:    bind("drawElementsInstanced"),
		createTexture:            bind("createTexture"),
		deleteTexture:            bind("deleteTexture"),
		activeTexture:            bind("activeTexture"),
		bindTexture:              bind("bindTexture"),
		texParameteri:            bind("texParameteri"),
		texImage2D:               bind("texImage2D"),
		texSubImage2D:            bind("texSubImage2D"),
		generateMipmap:           bind("generateMipmap"),
		copyTexSubImage2D:        bind("copyTexSubImage2D"),
		createFramebuffer:        bind("createFramebuffer"),
		deleteFramebuffer:        bind("deleteFramebuffer"),
		bindFramebuffer:          bind("bindFramebuffer"),
		framebufferTexture2D:     bind("framebufferTexture2D"),
		checkFramebufferStatus:   bind("checkFramebufferStatus"),
		createRenderbuffer:       bind("createRenderbuffer"),
		deleteRenderbuffer:       bind("deleteRenderbuffer"),
		bindRenderbuffer:         bind("bindRenderbuffer"),
		renderbufferStorage:      bind("renderbufferStorage"),
		framebufferRenderbuffer:  bind("framebufferRenderbuffer"),
		blitFramebuffer:          bind("blitFramebuffer"),
		readPixels:               bind("readPixels"),
		flush:                    bind("flush"),
		finish:                   bind("finish"),
	}
	if missing != "" {
		return nil, errors.New("SymbolMissing", "the WebGL context has no such function",
			"library", "WebGL2RenderingContext", "symbol", missing)
	}
	if list := gl.Call("getSupportedExtensions"); list.Truthy() {
		for i := 0; i < list.Length(); i++ {
			g.extensions = append(g.extensions, list.Index(i).String())
		}
	}
	g.renderer = g.str(g.f.getParameter.Invoke(glRenderer))
	// The unmasked renderer is the one that tells a GPU from a software rasterizer in the log.
	// Firefox may refuse the extension, and then the masked name stays.
	if info := gl.Call("getExtension", "WEBGL_debug_renderer_info"); info.Truthy() {
		if name := g.str(g.f.getParameter.Invoke(info.Get("UNMASKED_RENDERER_WEBGL"))); name != "" {
			g.renderer = name
		}
	}
	return g, nil
}

// str reads a JavaScript string that may be null.
func (g *glJS) str(v js.Value) string {
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}

// release drops the references to the objects of a context that is lost or closed.
func (g *glJS) release() {
	g.objs, g.locs = handleTable[js.Value]{}, handleTable[js.Value]{}
}

// obj returns the JavaScript object of a name, or null for 0 and a name that is not in use.
func (g *glJS) obj(id uint32) js.Value {
	if v, ok := g.objs.get(id); ok {
		return v
	}
	return js.Null()
}

// loc returns the JavaScript uniform location of a location, or null for -1.
func (g *glJS) loc(l int32) js.Value {
	if l >= 0 {
		if v, ok := g.locs.get(uint32(l) + 1); ok {
			return v
		}
	}
	return js.Null()
}

// create adds the object that a createX function returned, and returns its name: 0 when the
// context could not make it (a lost context gives null).
func (g *glJS) create(v js.Value) uint32 {
	if !v.Truthy() {
		return 0
	}
	return g.objs.add(v)
}

// drop forgets an object and tells WebGL to delete it.
func (g *glJS) drop(del js.Value, id uint32) {
	if v, ok := g.objs.remove(id); ok {
		del.Invoke(v)
	}
}

// number reads a parameter that WebGL gives as a number or a boolean.
func number(v js.Value) int32 {
	switch v.Type() {
	case js.TypeNumber:
		return int32(v.Int())
	case js.TypeBoolean:
		if v.Bool() {
			return 1
		}
	}
	return 0
}

// jsScratch is the array that data goes through to JavaScript and back. It is an ArrayBuffer
// with a Uint8Array and a Float32Array over it, and only grows.
type jsScratch struct {
	size    int
	u8, f32 js.Value
}

// ensure makes the array at least n bytes.
func (s *jsScratch) ensure(n int) {
	if n <= s.size {
		return
	}
	size := max(n, 2*s.size, 4096)
	size = (size + 3) &^ 3
	buf := js.Global().Get("ArrayBuffer").New(size)
	s.u8 = js.Global().Get("Uint8Array").New(buf)
	s.f32 = js.Global().Get("Float32Array").New(buf)
	s.size = size
}

// bytes copies data into the array and returns the Uint8Array.
func (g *glJS) bytes(data []byte) js.Value {
	g.scratch.ensure(len(data))
	js.CopyBytesToJS(g.scratch.u8, data)
	return g.scratch.u8
}

// floats copies values into the array and returns the Float32Array.
func (g *glJS) floats(v []float32) js.Value {
	g.scratch.ensure(len(v) * 4)
	js.CopyBytesToJS(g.scratch.u8, floatBytes(v))
	return g.scratch.f32
}

func (g *glJS) getString(name uint32) string {
	switch name {
	case glRenderer:
		return g.renderer
	case glVersion, glVendor, glShadingLanguageVersion:
		return g.str(g.f.getParameter.Invoke(name))
	}
	return ""
}

func (g *glJS) getStringi(name, index uint32) string {
	if name == glExtensions && int(index) < len(g.extensions) {
		return g.extensions[index]
	}
	return ""
}

func (g *glJS) getInteger(name uint32) int32 {
	if name == glNumExtensions {
		return int32(len(g.extensions))
	}
	return number(g.f.getParameter.Invoke(name))
}

func (g *glJS) getError() uint32 { return uint32(g.f.getError.Invoke().Int()) }

func (g *glJS) viewport(x, y, w, h int32)           { g.f.viewport.Invoke(x, y, w, h) }
func (g *glJS) scissor(x, y, w, h int32)            { g.f.scissor.Invoke(x, y, w, h) }
func (g *glJS) enable(c uint32)                     { g.f.enable.Invoke(c) }
func (g *glJS) disable(c uint32)                    { g.f.disable.Invoke(c) }
func (g *glJS) blendFunc(src, dst uint32)           { g.f.blendFunc.Invoke(src, dst) }
func (g *glJS) blendFuncSeparate(a, b, c, d uint32) { g.f.blendFuncSeparate.Invoke(a, b, c, d) }
func (g *glJS) blendEquation(mode uint32)           { g.f.blendEquation.Invoke(mode) }
func (g *glJS) colorMask(r, gr, b, a bool)          { g.f.colorMask.Invoke(r, gr, b, a) }
func (g *glJS) clearColor(r, gr, b, a float32)      { g.f.clearColor.Invoke(r, gr, b, a) }
func (g *glJS) clearStencil(s int32)                { g.f.clearStencil.Invoke(s) }
func (g *glJS) clear(mask uint32)                   { g.f.clear.Invoke(mask) }
func (g *glJS) stencilFunc(fn uint32, ref int32, m uint32) {
	g.f.stencilFunc.Invoke(fn, ref, m)
}
func (g *glJS) stencilOp(fail, zfail, zpass uint32) { g.f.stencilOp.Invoke(fail, zfail, zpass) }
func (g *glJS) stencilMask(mask uint32)             { g.f.stencilMask.Invoke(mask) }
func (g *glJS) pixelStorei(name uint32, v int32)    { g.f.pixelStorei.Invoke(name, v) }

func (g *glJS) createShader(kind uint32) uint32 { return g.create(g.f.createShader.Invoke(kind)) }
func (g *glJS) shaderSource(shader uint32, source string) {
	g.f.shaderSource.Invoke(g.obj(shader), source)
}
func (g *glJS) compileShader(shader uint32) { g.f.compileShader.Invoke(g.obj(shader)) }
func (g *glJS) getShaderiv(shader, name uint32) int32 {
	return number(g.f.getShaderParameter.Invoke(g.obj(shader), name))
}
func (g *glJS) getShaderInfoLog(shader uint32) string {
	return g.str(g.f.getShaderInfoLog.Invoke(g.obj(shader)))
}
func (g *glJS) deleteShader(shader uint32) { g.drop(g.f.deleteShader, shader) }
func (g *glJS) createProgram() uint32      { return g.create(g.f.createProgram.Invoke()) }
func (g *glJS) attachShader(p, s uint32)   { g.f.attachShader.Invoke(g.obj(p), g.obj(s)) }
func (g *glJS) linkProgram(p uint32)       { g.f.linkProgram.Invoke(g.obj(p)) }
func (g *glJS) getProgramiv(program, name uint32) int32 {
	return number(g.f.getProgramParameter.Invoke(g.obj(program), name))
}
func (g *glJS) getProgramInfoLog(program uint32) string {
	return g.str(g.f.getProgramInfoLog.Invoke(g.obj(program)))
}
func (g *glJS) useProgram(p uint32)    { g.f.useProgram.Invoke(g.obj(p)) }
func (g *glJS) deleteProgram(p uint32) { g.drop(g.f.deleteProgram, p) }
func (g *glJS) getUniformLocation(program uint32, name string) int32 {
	v := g.f.getUniformLocation.Invoke(g.obj(program), name)
	if !v.Truthy() {
		return -1
	}
	return int32(g.locs.add(v)) - 1
}
func (g *glJS) getAttribLocation(program uint32, name string) int32 {
	return int32(g.f.getAttribLocation.Invoke(g.obj(program), name).Int())
}
func (g *glJS) uniform1i(loc, v int32)            { g.f.uniform1i.Invoke(g.loc(loc), v) }
func (g *glJS) uniform1f(loc int32, v float32)    { g.f.uniform1f.Invoke(g.loc(loc), v) }
func (g *glJS) uniform2f(loc int32, x, y float32) { g.f.uniform2f.Invoke(g.loc(loc), x, y) }
func (g *glJS) uniform3f(loc int32, x, y, z float32) {
	g.f.uniform3f.Invoke(g.loc(loc), x, y, z)
}
func (g *glJS) uniform4f(loc int32, x, y, z, w float32) {
	g.f.uniform4f.Invoke(g.loc(loc), x, y, z, w)
}
func (g *glJS) uniform1fv(loc int32, v []float32) {
	switch len(v) {
	case 0:
	case 1:
		g.f.uniform1f.Invoke(g.loc(loc), v[0])
	default:
		g.f.uniform1fv.Invoke(g.loc(loc), g.floats(v), 0, len(v))
	}
}
func (g *glJS) uniform4fv(loc int32, v []float32) {
	switch {
	case len(v) < 4:
	case len(v) == 4:
		g.f.uniform4f.Invoke(g.loc(loc), v[0], v[1], v[2], v[3])
	default:
		g.f.uniform4fv.Invoke(g.loc(loc), g.floats(v), 0, len(v))
	}
}
func (g *glJS) uniformMatrix3fv(loc int32, v []float32) {
	if len(v) >= 9 {
		g.f.uniformMatrix3fv.Invoke(g.loc(loc), false, g.floats(v), 0, len(v))
	}
}
func (g *glJS) uniformMatrix4fv(loc int32, v []float32) {
	if len(v) >= 16 {
		g.f.uniformMatrix4fv.Invoke(g.loc(loc), false, g.floats(v), 0, len(v))
	}
}

func (g *glJS) genBuffer() uint32     { return g.create(g.f.createBuffer.Invoke()) }
func (g *glJS) deleteBuffer(b uint32) { g.drop(g.f.deleteBuffer, b) }
func (g *glJS) bindBuffer(target, b uint32) {
	g.f.bindBuffer.Invoke(target, g.obj(b))
}
func (g *glJS) bufferData(target uint32, size int, data []byte, usage uint32) {
	switch {
	case len(data) == 0:
		g.f.bufferData.Invoke(target, size, usage)
	case len(data) == size:
		g.f.bufferData.Invoke(target, g.bytes(data), usage, 0, len(data))
	default:
		g.f.bufferData.Invoke(target, size, usage)
		g.f.bufferSubData.Invoke(target, 0, g.bytes(data), 0, len(data))
	}
}
func (g *glJS) bufferSubData(target uint32, offset int, data []byte) {
	if len(data) > 0 {
		g.f.bufferSubData.Invoke(target, offset, g.bytes(data), 0, len(data))
	}
}
func (g *glJS) genVertexArray() uint32     { return g.create(g.f.createVertexArray.Invoke()) }
func (g *glJS) deleteVertexArray(a uint32) { g.drop(g.f.deleteVertexArray, a) }
func (g *glJS) bindVertexArray(a uint32)   { g.f.bindVertexArray.Invoke(g.obj(a)) }
func (g *glJS) enableVertexAttribArray(i uint32) {
	g.f.enableVertexAttribArray.Invoke(i)
}
func (g *glJS) disableVertexAttribArray(i uint32) {
	g.f.disableVertexAttribArray.Invoke(i)
}
func (g *glJS) vertexAttribPointer(index uint32, size int32, kind uint32, normalized bool, stride int32, offset int) {
	g.f.vertexAttribPointer.Invoke(index, size, kind, normalized, stride, offset)
}
func (g *glJS) vertexAttribIPointer(index uint32, size int32, kind uint32, stride int32, offset int) {
	g.f.vertexAttribIPointer.Invoke(index, size, kind, stride, offset)
}
func (g *glJS) vertexAttribDivisor(index, divisor uint32) {
	g.f.vertexAttribDivisor.Invoke(index, divisor)
}
func (g *glJS) drawArrays(mode uint32, first, count int32) {
	g.f.drawArrays.Invoke(mode, first, count)
}
func (g *glJS) drawArraysInstanced(mode uint32, first, count, instances int32) {
	g.f.drawArraysInstanced.Invoke(mode, first, count, instances)
}
func (g *glJS) drawElements(mode uint32, count int32, kind uint32, offset int) {
	g.f.drawElements.Invoke(mode, count, kind, offset)
}
func (g *glJS) drawElementsInstanced(mode uint32, count int32, kind uint32, offset int, instances int32) {
	g.f.drawElementsInstanced.Invoke(mode, count, kind, offset, instances)
}

func (g *glJS) genTexture() uint32        { return g.create(g.f.createTexture.Invoke()) }
func (g *glJS) deleteTexture(t uint32)    { g.drop(g.f.deleteTexture, t) }
func (g *glJS) activeTexture(unit uint32) { g.f.activeTexture.Invoke(unit) }
func (g *glJS) bindTexture(target, t uint32) {
	g.f.bindTexture.Invoke(target, g.obj(t))
}
func (g *glJS) texParameteri(target, name uint32, v int32) {
	g.f.texParameteri.Invoke(target, name, v)
}
func (g *glJS) texImage2D(target uint32, level, internalFormat, w, h int32, format, kind uint32, data []byte) {
	if len(data) == 0 {
		g.f.texImage2D.Invoke(target, level, internalFormat, w, h, 0, format, kind, js.Null())
		return
	}
	g.f.texImage2D.Invoke(target, level, internalFormat, w, h, 0, format, kind, g.bytes(data), 0)
}
func (g *glJS) texSubImage2D(target uint32, level, x, y, w, h int32, format, kind uint32, data []byte) {
	if len(data) == 0 {
		return
	}
	g.f.texSubImage2D.Invoke(target, level, x, y, w, h, format, kind, g.bytes(data), 0)
}
func (g *glJS) generateMipmap(target uint32) { g.f.generateMipmap.Invoke(target) }
func (g *glJS) copyTexSubImage2D(target uint32, level, xo, yo, x, y, w, h int32) {
	g.f.copyTexSubImage2D.Invoke(target, level, xo, yo, x, y, w, h)
}

func (g *glJS) genFramebuffer() uint32     { return g.create(g.f.createFramebuffer.Invoke()) }
func (g *glJS) deleteFramebuffer(f uint32) { g.drop(g.f.deleteFramebuffer, f) }
func (g *glJS) bindFramebuffer(target, f uint32) {
	g.f.bindFramebuffer.Invoke(target, g.obj(f))
}
func (g *glJS) framebufferTexture2D(target, attachment, texTarget, texture uint32, level int32) {
	g.f.framebufferTexture2D.Invoke(target, attachment, texTarget, g.obj(texture), level)
}
func (g *glJS) checkFramebufferStatus(target uint32) uint32 {
	return uint32(g.f.checkFramebufferStatus.Invoke(target).Int())
}
func (g *glJS) genRenderbuffer() uint32     { return g.create(g.f.createRenderbuffer.Invoke()) }
func (g *glJS) deleteRenderbuffer(r uint32) { g.drop(g.f.deleteRenderbuffer, r) }
func (g *glJS) bindRenderbuffer(target, r uint32) {
	g.f.bindRenderbuffer.Invoke(target, g.obj(r))
}
func (g *glJS) renderbufferStorage(target, format uint32, w, h int32) {
	g.f.renderbufferStorage.Invoke(target, format, w, h)
}
func (g *glJS) framebufferRenderbuffer(target, attachment, rbTarget, rb uint32) {
	g.f.framebufferRenderbuffer.Invoke(target, attachment, rbTarget, g.obj(rb))
}
func (g *glJS) blitFramebuffer(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {
	g.f.blitFramebuffer.Invoke(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1, mask, filter)
}
func (g *glJS) readPixels(x, y, w, h int32, format, kind uint32, pixels []byte) {
	if len(pixels) == 0 {
		return
	}
	g.scratch.ensure(len(pixels))
	g.f.readPixels.Invoke(x, y, w, h, format, kind, g.scratch.u8, 0)
	js.CopyBytesToGo(pixels, g.scratch.u8)
}

func (g *glJS) flush()  { g.f.flush.Invoke() }
func (g *glJS) finish() { g.f.finish.Invoke() }
