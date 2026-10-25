package gles

import (
	"fmt"
	"strings"
)

// fakeGL is a glAPI that records its calls and keeps just enough state to behave
// like a driver: object names are handed out and counted until deleted,
// shaders compile unless the test says otherwise, and framebuffers are complete.
type fakeGL struct {
	// Answers the test chooses.
	versionString string
	vendor        string
	renderer      string
	maxTexture    int32 // 4096 when 0
	maxRenderbuf  int32 // 4096 when 0
	maxUnits      int32 // 16 when 0
	extensionList []string
	pixels        []byte // bottom row first
	errCode       uint32 // returned by every getError
	errOnce       uint32 // returned by the next getError only
	failShader    string // "vertex" or "fragment": that stage does not compile
	failLink      bool
	// Program binaries: binaryFormats is how many formats the driver has; refuseBinary makes
	// programBinary leave the program unlinked.
	binaryFormats int32
	refuseBinary  bool
	linked        map[uint32]bool
	shaderLog     string
	fbStatus      uint32 // glFramebufferComplete when 0

	// What was set.
	viewportW, viewportH int32
	cleared              [4]float32
	clears               int
	sources              []string

	// Bookkeeping.
	calls    map[string]int
	log      []string
	next     uint32
	live     map[string]map[uint32]bool
	shaderOf map[uint32]uint32 // shader name -> kind
	locs     map[string]int32
	uploads  []int // byte lengths given to texSubImage2D
}

func (g *fakeGL) rec(name string, args ...any) {
	if g.calls == nil {
		g.calls = map[string]int{}
	}
	g.calls[name]++
	g.log = append(g.log, name+fmt.Sprint(args...))
}

func (g *fakeGL) gen(kind string) uint32 {
	if g.live == nil {
		g.live = map[string]map[uint32]bool{}
	}
	if g.live[kind] == nil {
		g.live[kind] = map[uint32]bool{}
	}
	g.next++
	g.live[kind][g.next] = true
	return g.next
}

func (g *fakeGL) del(kind string, id uint32) { delete(g.live[kind], id) }

// liveCount is how many objects of the kind are made and not deleted.
func (g *fakeGL) liveCount(kind string) int { return len(g.live[kind]) }

// resetCalls forgets the calls so far.
func (g *fakeGL) resetCalls() { g.calls, g.log = nil, nil }

func (g *fakeGL) getString(name uint32) string {
	g.rec("getString")
	switch name {
	case glVersion:
		return g.versionString
	case glVendor:
		return g.vendor
	case glRenderer:
		return g.renderer
	}
	return ""
}

func (g *fakeGL) getStringi(_, index uint32) string {
	g.rec("getStringi")
	return g.extensionList[index]
}

func (g *fakeGL) getInteger(name uint32) int32 {
	g.rec("getInteger")
	orDefault := func(v, d int32) int32 {
		if v == 0 {
			return d
		}
		return v
	}
	switch name {
	case glMaxTextureSize:
		return orDefault(g.maxTexture, 4096)
	case glMaxRenderbufferSize:
		return orDefault(g.maxRenderbuf, 4096)
	case glMaxTextureImageUnits:
		return orDefault(g.maxUnits, 16)
	case glNumExtensions:
		return int32(len(g.extensionList))
	}
	return 0
}

func (g *fakeGL) getError() uint32 {
	g.rec("getError")
	if g.errOnce != 0 {
		e := g.errOnce
		g.errOnce = 0
		return e
	}
	return g.errCode
}

func (g *fakeGL) viewport(x, y, w, h int32) {
	g.rec("viewport", x, y, w, h)
	g.viewportW, g.viewportH = w, h
}
func (g *fakeGL) scissor(x, y, w, h int32)                { g.rec("scissor", x, y, w, h) }
func (g *fakeGL) enable(c uint32)                         { g.rec("enable", c) }
func (g *fakeGL) disable(c uint32)                        { g.rec("disable", c) }
func (g *fakeGL) blendFunc(src, dst uint32)               { g.rec("blendFunc", src, dst) }
func (g *fakeGL) blendFuncSeparate(a, b, c, d uint32)     { g.rec("blendFuncSeparate", a, b, c, d) }
func (g *fakeGL) blendEquation(mode uint32)               { g.rec("blendEquation", mode) }
func (g *fakeGL) colorMask(r, gr, b, a bool)              { g.rec("colorMask", r, gr, b, a) }
func (g *fakeGL) clearStencil(s int32)                    { g.rec("clearStencil", s) }
func (g *fakeGL) stencilFunc(f uint32, r int32, m uint32) { g.rec("stencilFunc", f, r, m) }
func (g *fakeGL) stencilOp(a, b, c uint32)                { g.rec("stencilOp", a, b, c) }
func (g *fakeGL) stencilMask(m uint32)                    { g.rec("stencilMask", m) }
func (g *fakeGL) pixelStorei(name uint32, v int32)        { g.rec("pixelStorei", name, v) }
func (g *fakeGL) clearColor(r, gr, b, a float32) {
	g.rec("clearColor", r, gr, b, a)
	g.cleared = [4]float32{r, gr, b, a}
}
func (g *fakeGL) clear(mask uint32) {
	g.rec("clear", mask)
	g.clears++
}

func (g *fakeGL) createShader(kind uint32) uint32 {
	g.rec("createShader", kind)
	id := g.gen("shader")
	if g.shaderOf == nil {
		g.shaderOf = map[uint32]uint32{}
	}
	g.shaderOf[id] = kind
	return id
}
func (g *fakeGL) shaderSource(_ uint32, source string) {
	g.rec("shaderSource")
	g.sources = append(g.sources, source)
}
func (g *fakeGL) compileShader(s uint32) { g.rec("compileShader", s) }
func (g *fakeGL) getShaderiv(s, name uint32) int32 {
	g.rec("getShaderiv")
	if name != glCompileStatus {
		return 0
	}
	kind := g.shaderOf[s]
	if (g.failShader == "vertex" && kind == glVertexShader) || (g.failShader == "fragment" && kind == glFragmentShader) {
		return 0
	}
	return 1
}
func (g *fakeGL) getShaderInfoLog(uint32) string {
	g.rec("getShaderInfoLog")
	if g.shaderLog != "" {
		return g.shaderLog
	}
	return "0:1: error: fake compile error"
}
func (g *fakeGL) deleteShader(s uint32)    { g.rec("deleteShader", s); g.del("shader", s) }
func (g *fakeGL) createProgram() uint32    { g.rec("createProgram"); return g.gen("program") }
func (g *fakeGL) attachShader(p, s uint32) { g.rec("attachShader", p, s) }
func (g *fakeGL) linkProgram(p uint32) {
	g.rec("linkProgram", p)
	if g.linked == nil {
		g.linked = map[uint32]bool{}
	}
	g.linked[p] = !g.failLink
}
func (g *fakeGL) getProgramiv(p, name uint32) int32 {
	g.rec("getProgramiv")
	switch name {
	case glLinkStatus:
		if g.failLink && !g.linked[p] {
			return 0
		}
		if !g.linked[p] && g.refuseBinary {
			return 0
		}
		return 1
	case glProgramBinaryLength:
		return int32(len(g.binaryOf(p)))
	}
	return 1
}

// binaryOf is the binary the fake driver makes of a program: its name, so that a test can tell them apart.
func (g *fakeGL) binaryOf(p uint32) []byte { return []byte(fmt.Sprintf("binary-of-%d", p)) }

func (g *fakeGL) programBinaryFormats() int32 { g.rec("programBinaryFormats"); return g.binaryFormats }
func (g *fakeGL) getProgramBinary(p uint32) (uint32, []byte, bool) {
	g.rec("getProgramBinary", p)
	if g.binaryFormats == 0 || !g.linked[p] {
		return 0, nil, false
	}
	return 0x1234, g.binaryOf(p), true
}
func (g *fakeGL) programBinary(p, format uint32, data []byte) {
	g.rec("programBinary", p, format, string(data))
	if g.refuseBinary || format != 0x1234 {
		g.errOnce = glInvalidEnum
		return
	}
	if g.linked == nil {
		g.linked = map[uint32]bool{}
	}
	g.linked[p] = true
}
func (g *fakeGL) programParameteri(p, name uint32, value int32) {
	g.rec("programParameteri", p, name, value)
}
func (g *fakeGL) getProgramInfoLog(uint32) string {
	g.rec("getProgramInfoLog")
	return "error: fake link error"
}
func (g *fakeGL) useProgram(p uint32)    { g.rec("useProgram", p) }
func (g *fakeGL) deleteProgram(p uint32) { g.rec("deleteProgram", p); g.del("program", p) }
func (g *fakeGL) getUniformLocation(p uint32, name string) int32 {
	g.rec("getUniformLocation", p, name)
	if strings.HasPrefix(name, "missing") {
		return -1
	}
	if g.locs == nil {
		g.locs = map[string]int32{}
	}
	key := fmt.Sprint(p, name)
	if _, ok := g.locs[key]; !ok {
		g.locs[key] = int32(len(g.locs))
	}
	return g.locs[key]
}
func (g *fakeGL) getAttribLocation(p uint32, name string) int32 {
	g.rec("getAttribLocation", p, name)
	if strings.HasPrefix(name, "missing") {
		return -1
	}
	return 0
}
func (g *fakeGL) uniform1i(l, v int32)                  { g.rec("uniform1i", l, v) }
func (g *fakeGL) uniform1f(l int32, v float32)          { g.rec("uniform1f", l, v) }
func (g *fakeGL) uniform2f(l int32, x, y float32)       { g.rec("uniform2f", l, x, y) }
func (g *fakeGL) uniform3f(l int32, x, y, z float32)    { g.rec("uniform3f", l, x, y, z) }
func (g *fakeGL) uniform4f(l int32, x, y, z, w float32) { g.rec("uniform4f", l, x, y, z, w) }
func (g *fakeGL) uniform1fv(l int32, v []float32)       { g.rec("uniform1fv", l, len(v)) }
func (g *fakeGL) uniform4fv(l int32, v []float32)       { g.rec("uniform4fv", l, len(v)) }
func (g *fakeGL) uniformMatrix3fv(l int32, v []float32) { g.rec("uniformMatrix3fv", l, len(v)) }
func (g *fakeGL) uniformMatrix4fv(l int32, v []float32) { g.rec("uniformMatrix4fv", l, len(v)) }

func (g *fakeGL) genBuffer() uint32      { g.rec("genBuffer"); return g.gen("buffer") }
func (g *fakeGL) deleteBuffer(b uint32)  { g.rec("deleteBuffer", b); g.del("buffer", b) }
func (g *fakeGL) bindBuffer(t, b uint32) { g.rec("bindBuffer", t, b) }
func (g *fakeGL) bufferData(t uint32, size int, data []byte, u uint32) {
	g.rec("bufferData", t, size, len(data), u)
}
func (g *fakeGL) bufferSubData(t uint32, off int, data []byte) {
	g.rec("bufferSubData", t, off, len(data))
}
func (g *fakeGL) genVertexArray() uint32            { g.rec("genVertexArray"); return g.gen("vao") }
func (g *fakeGL) deleteVertexArray(a uint32)        { g.rec("deleteVertexArray", a); g.del("vao", a) }
func (g *fakeGL) bindVertexArray(a uint32)          { g.rec("bindVertexArray", a) }
func (g *fakeGL) enableVertexAttribArray(i uint32)  { g.rec("enableVertexAttribArray", i) }
func (g *fakeGL) disableVertexAttribArray(i uint32) { g.rec("disableVertexAttribArray", i) }
func (g *fakeGL) vertexAttribPointer(i uint32, size int32, kind uint32, n bool, stride int32, off int) {
	g.rec("vertexAttribPointer", i, size, kind, n, stride, off)
}
func (g *fakeGL) vertexAttribIPointer(i uint32, size int32, kind uint32, stride int32, off int) {
	g.rec("vertexAttribIPointer", i, size, kind, stride, off)
}
func (g *fakeGL) vertexAttribDivisor(i, d uint32)            { g.rec("vertexAttribDivisor", i, d) }
func (g *fakeGL) drawArrays(mode uint32, first, count int32) { g.rec("drawArrays", mode, first, count) }
func (g *fakeGL) drawArraysInstanced(mode uint32, first, count, n int32) {
	g.rec("drawArraysInstanced", mode, first, count, n)
}
func (g *fakeGL) drawElements(mode uint32, count int32, kind uint32, off int) {
	g.rec("drawElements", mode, count, kind, off)
}
func (g *fakeGL) drawElementsInstanced(mode uint32, count int32, kind uint32, off int, n int32) {
	g.rec("drawElementsInstanced", mode, count, kind, off, n)
}

func (g *fakeGL) genTexture() uint32     { g.rec("genTexture"); return g.gen("texture") }
func (g *fakeGL) deleteTexture(t uint32) { g.rec("deleteTexture", t); g.del("texture", t) }
func (g *fakeGL) activeTexture(u uint32) { g.rec("activeTexture", u) }
func (g *fakeGL) bindTexture(t, id uint32) {
	g.rec("bindTexture", t, id)
}
func (g *fakeGL) texParameteri(t, name uint32, v int32) { g.rec("texParameteri", t, name, v) }
func (g *fakeGL) texImage2D(t uint32, level, internal, w, h int32, format, kind uint32, data []byte) {
	g.rec("texImage2D", t, level, internal, w, h, format, kind, len(data))
}
func (g *fakeGL) texSubImage2D(t uint32, level, x, y, w, h int32, format, kind uint32, data []byte) {
	g.rec("texSubImage2D", t, level, x, y, w, h, format, kind, len(data))
	g.uploads = append(g.uploads, len(data))
}
func (g *fakeGL) generateMipmap(t uint32) { g.rec("generateMipmap", t) }
func (g *fakeGL) copyTexSubImage2D(t uint32, level, xo, yo, x, y, w, h int32) {
	g.rec("copyTexSubImage2D", t, level, xo, yo, x, y, w, h)
}

func (g *fakeGL) genFramebuffer() uint32      { g.rec("genFramebuffer"); return g.gen("framebuffer") }
func (g *fakeGL) deleteFramebuffer(f uint32)  { g.rec("deleteFramebuffer", f); g.del("framebuffer", f) }
func (g *fakeGL) bindFramebuffer(t, f uint32) { g.rec("bindFramebuffer", t, f) }
func (g *fakeGL) framebufferTexture2D(t, a, tt, tex uint32, level int32) {
	g.rec("framebufferTexture2D", t, a, tt, tex, level)
}
func (g *fakeGL) checkFramebufferStatus(uint32) uint32 {
	g.rec("checkFramebufferStatus")
	if g.fbStatus != 0 {
		return g.fbStatus
	}
	return glFramebufferComplete
}
func (g *fakeGL) genRenderbuffer() uint32 { g.rec("genRenderbuffer"); return g.gen("renderbuffer") }
func (g *fakeGL) deleteRenderbuffer(r uint32) {
	g.rec("deleteRenderbuffer", r)
	g.del("renderbuffer", r)
}
func (g *fakeGL) bindRenderbuffer(t, r uint32) { g.rec("bindRenderbuffer", t, r) }
func (g *fakeGL) renderbufferStorage(t, f uint32, w, h int32) {
	g.rec("renderbufferStorage", t, f, w, h)
}
func (g *fakeGL) framebufferRenderbuffer(t, a, rt, r uint32) {
	g.rec("framebufferRenderbuffer", t, a, rt, r)
}
func (g *fakeGL) blitFramebuffer(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {
	g.rec("blitFramebuffer", sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1, mask, filter)
}
func (g *fakeGL) readPixels(x, y, w, h int32, format, kind uint32, pixels []byte) {
	g.rec("readPixels", x, y, w, h)
	copy(pixels, g.pixels)
}
func (g *fakeGL) flush()  { g.rec("flush") }
func (g *fakeGL) finish() { g.rec("finish") }
