package gles

import "unsafe"

// glAPI is the part of OpenGL ES 3.0 that the renderer uses, in Go terms: strings
// and slices instead of pointers, one object at a time instead of arrays of IDs.
// The real implementation calls the driver's functions through purego; tests
// supply a fake that records the calls.
//
// Every GL call of the renderer goes through this interface (SPEC §6.3 rule 4),
// and the device (device.go) is its only user besides the context.
type glAPI interface {
	// Queries.
	getString(name uint32) string
	getStringi(name, index uint32) string
	getInteger(name uint32) int32
	getError() uint32

	// State.
	viewport(x, y, w, h int32)
	scissor(x, y, w, h int32)
	enable(capability uint32)
	disable(capability uint32)
	blendFunc(src, dst uint32)
	blendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha uint32)
	blendEquation(mode uint32)
	colorMask(r, g, b, a bool)
	clearColor(r, g, b, a float32)
	clearStencil(s int32)
	clear(mask uint32)
	stencilFunc(fn uint32, ref int32, mask uint32)
	stencilOp(fail, zfail, zpass uint32)
	stencilMask(mask uint32)
	pixelStorei(name uint32, value int32)

	// Shaders and programs.
	createShader(kind uint32) uint32
	shaderSource(shader uint32, source string)
	compileShader(shader uint32)
	getShaderiv(shader, name uint32) int32
	getShaderInfoLog(shader uint32) string
	deleteShader(shader uint32)
	createProgram() uint32
	attachShader(program, shader uint32)
	linkProgram(program uint32)
	getProgramiv(program, name uint32) int32
	getProgramInfoLog(program uint32) string
	// Program binaries (GLES 3.0), for the program cache. programBinaryFormats is how many formats
	// the driver has (0: none); getProgramBinary returns the binary of a linked program (false when
	// the driver gives none); programBinary loads one into a program, which then has a link status
	// (the driver may refuse it); programParameteri sets a hint such as the one that makes the binary
	// retrievable, before the link. The web has none.
	programBinaryFormats() int32
	getProgramBinary(program uint32) (format uint32, data []byte, ok bool)
	programBinary(program, format uint32, data []byte)
	programParameteri(program, name uint32, value int32)
	useProgram(program uint32)
	deleteProgram(program uint32)
	getUniformLocation(program uint32, name string) int32
	getAttribLocation(program uint32, name string) int32
	uniform1i(location, v int32)
	uniform1f(location int32, v float32)
	uniform2f(location int32, x, y float32)
	uniform3f(location int32, x, y, z float32)
	uniform4f(location int32, x, y, z, w float32)
	uniform1fv(location int32, values []float32)
	uniform4fv(location int32, values []float32)
	uniformMatrix3fv(location int32, values []float32)
	uniformMatrix4fv(location int32, values []float32)

	// Buffers and vertex arrays.
	genBuffer() uint32
	deleteBuffer(buffer uint32)
	bindBuffer(target, buffer uint32)
	// bufferData allocates size bytes and fills them from data, which may be nil
	// (and then must not be longer than size).
	bufferData(target uint32, size int, data []byte, usage uint32)
	bufferSubData(target uint32, offset int, data []byte)
	genVertexArray() uint32
	deleteVertexArray(array uint32)
	bindVertexArray(array uint32)
	enableVertexAttribArray(index uint32)
	disableVertexAttribArray(index uint32)
	vertexAttribPointer(index uint32, size int32, kind uint32, normalized bool, stride int32, offset int)
	vertexAttribIPointer(index uint32, size int32, kind uint32, stride int32, offset int)
	vertexAttribDivisor(index, divisor uint32)
	drawArrays(mode uint32, first, count int32)
	drawArraysInstanced(mode uint32, first, count, instances int32)
	drawElements(mode uint32, count int32, kind uint32, offset int)
	drawElementsInstanced(mode uint32, count int32, kind uint32, offset int, instances int32)

	// Textures.
	genTexture() uint32
	deleteTexture(texture uint32)
	activeTexture(unit uint32)
	bindTexture(target, texture uint32)
	texParameteri(target, name uint32, value int32)
	// texImage2D allocates a level and fills it from data, which may be nil.
	texImage2D(target uint32, level int32, internalFormat int32, w, h int32, format, kind uint32, data []byte)
	texSubImage2D(target uint32, level, x, y, w, h int32, format, kind uint32, data []byte)
	generateMipmap(target uint32)
	copyTexSubImage2D(target uint32, level, xoffset, yoffset, x, y, w, h int32)

	// Framebuffers and renderbuffers.
	genFramebuffer() uint32
	deleteFramebuffer(framebuffer uint32)
	bindFramebuffer(target, framebuffer uint32)
	framebufferTexture2D(target, attachment, texTarget, texture uint32, level int32)
	checkFramebufferStatus(target uint32) uint32
	genRenderbuffer() uint32
	deleteRenderbuffer(renderbuffer uint32)
	bindRenderbuffer(target, renderbuffer uint32)
	renderbufferStorage(target, format uint32, w, h int32)
	framebufferRenderbuffer(target, attachment, rbTarget, renderbuffer uint32)
	blitFramebuffer(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32)
	// readPixels reads the w x h pixels at x, y of the read framebuffer into
	// pixels, bottom row first.
	readPixels(x, y, w, h int32, format, kind uint32, pixels []byte)

	flush()
	finish()
}

func floatBytes(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*4)
}
