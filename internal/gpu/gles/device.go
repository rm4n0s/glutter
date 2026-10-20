package gles

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// shaderFiles holds the GLSL ES 3.00 sources, as <name>.vert.glsl and
// <name>.frag.glsl pairs (SPEC §6.3 rule 4).
//
//go:embed shaders/*.glsl
var shaderFiles embed.FS

// Caps are what the driver reports about itself.
//
// Flutter: no counterpart
type Caps struct {
	// Version, Vendor and Renderer are the GL strings.
	Version, Vendor, Renderer string
	// MaxTextureSize is the largest width or height of a texture.
	MaxTextureSize int
	// MaxRenderbufferSize is the largest width or height of a renderbuffer.
	MaxRenderbufferSize int
	// MaxTextureUnits is the number of texture units a fragment shader can use.
	MaxTextureUnits int
	// Extensions are the names of the supported extensions.
	Extensions map[string]bool
}

// maxTrackedUnits is how many texture units the state cache tracks.
const maxTrackedUnits = 16

// defaultMaxIdleFrames is how many frames a pooled object may go unused before
// EndFrame frees it.
const defaultMaxIdleFrames = 120

// Device owns the GL objects of the renderer: the shader programs, and pools of
// buffers, textures and framebuffers that are reused instead of created again. It
// remembers the GL state it has set, so that setting it again costs nothing. Use
// it only on the UI thread, with its context current.
//
// Flutter: no counterpart
type Device struct {
	gl       glAPI
	caps     Caps
	state    glState
	programs map[string]*Program
	textures map[textureKey][]*Texture
	buffers  map[bufferKey][]*Buffer
	frames   map[framebufferKey][]*Framebuffer
	vaos     map[string]uint32
	frame    uint64
	live     liveCounts
	closed   bool
}

// liveCounts is how many GL objects the device has made and not deleted.
type liveCounts struct {
	textures, buffers, framebuffers, renderbuffers, programs int
}

// glState is the GL state the device has set. A field is only trusted when the
// device set it; newDevice sets all of them.
type glState struct {
	viewport       [4]int32
	scissorOn      bool
	scissor        [4]int32
	blendOn        bool
	blendSrcRGB    uint32
	blendDstRGB    uint32
	blendSrcA      uint32
	blendDstA      uint32
	blendEquation  uint32
	colorMask      [4]bool
	program        uint32
	activeUnit     uint32
	texture        [maxTrackedUnits]uint32
	framebuffer    uint32
	vao            uint32
	arrayBuffer    uint32
	stencilOn      bool
	stencilFunc    uint32
	stencilRef     int32
	stencilFuncMsk uint32
	stencilFail    uint32
	stencilZFail   uint32
	stencilZPass   uint32
	stencilMask    uint32
}

// newDevice reads the driver's capabilities, sets the GL state the renderer
// starts from, and compiles every shader program, so that a broken shader shows
// at startup and not in the middle of a frame. It fails with GLES3Unsupported when
// GL_VERSION is older than 3.0 and ShaderCompileFailed when a shader does not
// build.
func newDevice(gl glAPI) (*Device, error) {
	d := &Device{
		gl:       gl,
		programs: map[string]*Program{},
		textures: map[textureKey][]*Texture{},
		buffers:  map[bufferKey][]*Buffer{},
		frames:   map[framebufferKey][]*Framebuffer{},
		vaos:     map[string]uint32{},
	}
	d.caps = Caps{
		Version:             gl.getString(glVersion),
		Vendor:              gl.getString(glVendor),
		Renderer:            gl.getString(glRenderer),
		MaxTextureSize:      int(gl.getInteger(glMaxTextureSize)),
		MaxRenderbufferSize: int(gl.getInteger(glMaxRenderbufferSize)),
		MaxTextureUnits:     int(gl.getInteger(glMaxTextureImageUnits)),
		Extensions:          map[string]bool{},
	}
	if !isES3(d.caps.Version) {
		return nil, errors.New("GLES3Unsupported", "the driver does not provide OpenGL ES 3.0", "version", d.caps.Version)
	}
	for i, n := uint32(0), uint32(gl.getInteger(glNumExtensions)); i < n; i++ {
		d.caps.Extensions[gl.getStringi(glExtensions, i)] = true
	}
	d.resetState()
	names, err := ShaderNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if _, err := d.Program(name); err != nil {
			d.Close()
			return nil, err
		}
	}
	return d, nil
}

// Caps returns what the driver reported.
//
// Flutter: no counterpart
func (d *Device) Caps() Caps { return d.caps }

// resetState sets the whole GL state to what the renderer starts from, and
// records it. Premultiplied source-over is the default blend; nothing is
// scissored, stenciled, culled or dithered; pixel rows are tightly packed.
func (d *Device) resetState() {
	gl := d.gl
	gl.disable(glDepthTest)
	gl.disable(glCullFace)
	gl.disable(glDither)
	gl.pixelStorei(glUnpackAlignment, 1)
	gl.pixelStorei(glPackAlignment, 1)
	gl.disable(glScissorTest)
	gl.enable(glBlend)
	gl.blendFuncSeparate(glOne, glOneMinusSrcAlpha, glOne, glOneMinusSrcAlpha)
	gl.blendEquation(glFuncAdd)
	gl.colorMask(true, true, true, true)
	gl.disable(glStencilTest)
	gl.stencilFunc(glAlways, 0, 0xFFFFFFFF)
	gl.stencilOp(glKeep, glKeep, glKeep)
	gl.stencilMask(0xFFFFFFFF)
	gl.useProgram(0)
	gl.activeTexture(glTexture0)
	for i := 0; i < maxTrackedUnits; i++ {
		gl.activeTexture(glTexture0 + uint32(i))
		gl.bindTexture(glTexture2D, 0)
	}
	gl.activeTexture(glTexture0)
	gl.bindFramebuffer(glFramebuffer, 0)
	gl.bindVertexArray(0)
	gl.bindBuffer(glArrayBuffer, 0)
	d.state = glState{
		viewport:  [4]int32{-1, -1, -1, -1},
		scissorOn: false, blendOn: true,
		blendSrcRGB: glOne, blendDstRGB: glOneMinusSrcAlpha, blendSrcA: glOne, blendDstA: glOneMinusSrcAlpha,
		blendEquation: glFuncAdd, colorMask: [4]bool{true, true, true, true},
		stencilFunc: glAlways, stencilFuncMsk: 0xFFFFFFFF, stencilFail: glKeep, stencilZFail: glKeep, stencilZPass: glKeep,
		stencilMask: 0xFFFFFFFF,
	}
}

// ResetState sets the GL state to the starting point again. Call it after code
// that is not the device has used the GL context.
//
// Flutter: no counterpart
func (d *Device) ResetState() { d.resetState() }

// The state setters call GL only when the value changes.

// Viewport sets the viewport.
//
// Flutter: no counterpart
func (d *Device) Viewport(x, y, w, h int) {
	v := [4]int32{int32(x), int32(y), int32(w), int32(h)}
	if d.state.viewport == v {
		return
	}
	d.state.viewport = v
	d.gl.viewport(v[0], v[1], v[2], v[3])
}

// SetScissor turns the scissor test on with a rectangle, or off. The rectangle is
// in the GL's pixels, from the bottom left.
//
// Flutter: no counterpart
func (d *Device) SetScissor(on bool, x, y, w, h int) {
	if !on {
		if d.state.scissorOn {
			d.state.scissorOn = false
			d.gl.disable(glScissorTest)
		}
		return
	}
	if !d.state.scissorOn {
		d.state.scissorOn = true
		d.gl.enable(glScissorTest)
	}
	r := [4]int32{int32(x), int32(y), int32(w), int32(h)}
	if d.state.scissor != r {
		d.state.scissor = r
		d.gl.scissor(r[0], r[1], r[2], r[3])
	}
}

// SetBlend turns blending on or off.
//
// Flutter: no counterpart
func (d *Device) SetBlend(on bool) {
	if d.state.blendOn == on {
		return
	}
	d.state.blendOn = on
	if on {
		d.gl.enable(glBlend)
	} else {
		d.gl.disable(glBlend)
	}
}

// BlendFunc sets the blend factors for color and alpha.
//
// Flutter: no counterpart
func (d *Device) BlendFunc(srcRGB, dstRGB, srcAlpha, dstAlpha uint32) {
	s := &d.state
	if s.blendSrcRGB == srcRGB && s.blendDstRGB == dstRGB && s.blendSrcA == srcAlpha && s.blendDstA == dstAlpha {
		return
	}
	s.blendSrcRGB, s.blendDstRGB, s.blendSrcA, s.blendDstA = srcRGB, dstRGB, srcAlpha, dstAlpha
	d.gl.blendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha)
}

// BlendEquation sets how the source and the destination are combined.
//
// Flutter: no counterpart
func (d *Device) BlendEquation(mode uint32) {
	if d.state.blendEquation == mode {
		return
	}
	d.state.blendEquation = mode
	d.gl.blendEquation(mode)
}

// ColorMask says which color channels are written.
//
// Flutter: no counterpart
func (d *Device) ColorMask(r, g, b, a bool) {
	m := [4]bool{r, g, b, a}
	if d.state.colorMask == m {
		return
	}
	d.state.colorMask = m
	d.gl.colorMask(r, g, b, a)
}

// SetStencil turns the stencil test on or off.
//
// Flutter: no counterpart
func (d *Device) SetStencil(on bool) {
	if d.state.stencilOn == on {
		return
	}
	d.state.stencilOn = on
	if on {
		d.gl.enable(glStencilTest)
	} else {
		d.gl.disable(glStencilTest)
	}
}

// StencilFunc sets the stencil comparison.
//
// Flutter: no counterpart
func (d *Device) StencilFunc(fn uint32, ref int32, mask uint32) {
	s := &d.state
	if s.stencilFunc == fn && s.stencilRef == ref && s.stencilFuncMsk == mask {
		return
	}
	s.stencilFunc, s.stencilRef, s.stencilFuncMsk = fn, ref, mask
	d.gl.stencilFunc(fn, ref, mask)
}

// StencilOp sets what happens to the stencil value on each outcome of the tests.
//
// Flutter: no counterpart
func (d *Device) StencilOp(fail, zfail, zpass uint32) {
	s := &d.state
	if s.stencilFail == fail && s.stencilZFail == zfail && s.stencilZPass == zpass {
		return
	}
	s.stencilFail, s.stencilZFail, s.stencilZPass = fail, zfail, zpass
	d.gl.stencilOp(fail, zfail, zpass)
}

// StencilMask sets which stencil bits are written.
//
// Flutter: no counterpart
func (d *Device) StencilMask(mask uint32) {
	if d.state.stencilMask == mask {
		return
	}
	d.state.stencilMask = mask
	d.gl.stencilMask(mask)
}

// UseProgram makes a program current.
//
// Flutter: no counterpart
func (d *Device) UseProgram(p *Program) {
	if d.state.program == p.id {
		return
	}
	d.state.program = p.id
	d.gl.useProgram(p.id)
}

// BindTexture binds a 2D texture to a texture unit, and makes the unit active.
//
// Flutter: no counterpart
func (d *Device) BindTexture(unit int, t *Texture) {
	d.bindTextureID(unit, t.id)
}

func (d *Device) bindTextureID(unit int, id uint32) {
	if unit < 0 || unit >= maxTrackedUnits {
		panic(errors.New("GLTextureUnitInvalid", "the texture unit is out of range", "unit", unit, "max", maxTrackedUnits-1))
	}
	// The unit is made the active one even when the texture is bound to it
	// already: the calls that follow (texImage2D, texSubImage2D, texParameteri,
	// generateMipmap) act on the texture of the active unit.
	if d.state.activeUnit != uint32(unit) {
		d.state.activeUnit = uint32(unit)
		d.gl.activeTexture(glTexture0 + uint32(unit))
	}
	if d.state.texture[unit] == id {
		return
	}
	d.state.texture[unit] = id
	d.gl.bindTexture(glTexture2D, id)
}

// bindFramebufferID binds a framebuffer for drawing and reading; 0 is the window's.
func (d *Device) bindFramebufferID(id uint32) {
	if d.state.framebuffer == id {
		return
	}
	d.state.framebuffer = id
	d.gl.bindFramebuffer(glFramebuffer, id)
}

// BindFramebuffer makes a framebuffer the target of drawing and reading; nil is
// the window's.
//
// Flutter: no counterpart
func (d *Device) BindFramebuffer(f *Framebuffer) {
	if f == nil {
		d.bindFramebufferID(0)
		return
	}
	d.bindFramebufferID(f.id)
}

// BindVertexArray makes a vertex array current; 0 is none.
//
// Flutter: no counterpart
func (d *Device) BindVertexArray(vao uint32) {
	if d.state.vao == vao {
		return
	}
	d.state.vao = vao
	d.gl.bindVertexArray(vao)
}

// BindArrayBuffer binds a buffer as the source of vertex attributes; nil unbinds.
//
// Flutter: no counterpart
func (d *Device) BindArrayBuffer(b *Buffer) {
	var id uint32
	if b != nil {
		id = b.id
	}
	if d.state.arrayBuffer == id {
		return
	}
	d.state.arrayBuffer = id
	d.gl.bindBuffer(glArrayBuffer, id)
}

// checkError turns a GL error into a tagged one. It reads, and so clears, the
// error flag.
func (d *Device) checkError(operation string) error {
	if code := d.gl.getError(); code != glNoError {
		return errors.New("GLError", "OpenGL reported an error", "operation", operation, "gl_error", fmt.Sprintf("0x%x", code))
	}
	return nil
}

// Program is a linked shader program, with its uniform and attribute locations
// remembered after the first lookup.
//
// Flutter: no counterpart
type Program struct {
	name     string
	id       uint32
	dev      *Device
	uniforms map[string]int32
	attribs  map[string]int32
}

// Name is the name of the program: the name of its shader files.
//
// Flutter: no counterpart
func (p *Program) Name() string { return p.name }

// Uniform returns the location of a uniform, or -1 when the program has no such
// uniform (or the compiler removed it because it is unused).
//
// Flutter: no counterpart
func (p *Program) Uniform(name string) int32 {
	if loc, ok := p.uniforms[name]; ok {
		return loc
	}
	loc := p.dev.gl.getUniformLocation(p.id, name)
	p.uniforms[name] = loc
	return loc
}

// Attrib returns the location of a vertex attribute, or -1.
//
// Flutter: no counterpart
func (p *Program) Attrib(name string) int32 {
	if loc, ok := p.attribs[name]; ok {
		return loc
	}
	loc := p.dev.gl.getAttribLocation(p.id, name)
	p.attribs[name] = loc
	return loc
}

// ShaderNames returns the names of the shader programs in the executable, in
// alphabetical order: each has a <name>.vert.glsl and a <name>.frag.glsl.
//
// Flutter: no counterpart
func ShaderNames() ([]string, error) {
	entries, err := shaderFiles.ReadDir("shaders")
	if err != nil {
		return nil, errors.NewErr("ShaderSourceMissing", err)
	}
	set := map[string]bool{}
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".vert.glsl"); ok {
			set[name] = true
		}
	}
	var names []string
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// ShaderSource returns the GLSL source of a stage ("vert" or "frag") of a
// program. A line of the form //#include "name.glsl" is replaced by the contents
// of that file of the shaders directory, so that shaders share code.
//
// Flutter: no counterpart
func ShaderSource(name, stage string) (string, error) {
	data, err := shaderFiles.ReadFile("shaders/" + name + "." + stage + ".glsl")
	if err != nil {
		return "", errors.NewErr("ShaderSourceMissing", err).SetMetadata("program", name, "stage", stage)
	}
	return expandIncludes(name, stage, string(data), func(file string) ([]byte, error) {
		return shaderFiles.ReadFile("shaders/" + file)
	})
}

// expandIncludes replaces the include lines of a shader's source with the files
// that read returns.
func expandIncludes(name, stage, source string, read func(file string) ([]byte, error)) (string, error) {
	var out strings.Builder
	for _, line := range strings.SplitAfter(source, "\n") {
		trimmed := strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(trimmed, "//#include")
		if !ok {
			out.WriteString(line)
			continue
		}
		file, ok := strings.CutSuffix(strings.TrimPrefix(strings.TrimSpace(rest), "\""), "\"")
		if !ok || !strings.HasPrefix(strings.TrimSpace(rest), "\"") {
			return "", errors.New("ShaderIncludeInvalid", "an include line needs a quoted file name", "program", name, "stage", stage, "line", trimmed)
		}
		inc, err := read(file)
		if err != nil {
			return "", errors.NewErr("ShaderSourceMissing", err).SetMetadata("program", name, "stage", stage, "include", file)
		}
		out.Write(inc)
		if !strings.HasSuffix(string(inc), "\n") {
			out.WriteByte('\n')
		}
	}
	return out.String(), nil
}

// Program returns the program of the named shader files, compiling it the first
// time. It fails with ShaderSourceMissing when there are no such files, and
// ShaderCompileFailed, with the driver's log in its metadata, when a stage does
// not compile or the program does not link.
//
// Flutter: no counterpart
func (d *Device) Program(name string) (*Program, error) {
	if p, ok := d.programs[name]; ok {
		return p, nil
	}
	vert, err := ShaderSource(name, "vert")
	if err != nil {
		return nil, err
	}
	frag, err := ShaderSource(name, "frag")
	if err != nil {
		return nil, err
	}
	p, err := d.NewProgram(name, vert, frag)
	if err != nil {
		return nil, err
	}
	d.programs[name] = p
	return p, nil
}

// NewProgram compiles and links a program from GLSL ES 3.00 sources. It fails
// with ShaderCompileFailed, whose metadata has the program, the stage ("vertex
// shader", "fragment shader" or "link") and the driver's log.
//
// Flutter: no counterpart
func (d *Device) NewProgram(name, vertexSource, fragmentSource string) (*Program, error) {
	vs, err := d.compile(name, "vertex shader", glVertexShader, vertexSource)
	if err != nil {
		return nil, err
	}
	fs, err := d.compile(name, "fragment shader", glFragmentShader, fragmentSource)
	if err != nil {
		d.gl.deleteShader(vs)
		return nil, err
	}
	id := d.gl.createProgram()
	d.gl.attachShader(id, vs)
	d.gl.attachShader(id, fs)
	d.gl.linkProgram(id)
	// The shaders are no longer needed once linked, or once linking failed.
	d.gl.deleteShader(vs)
	d.gl.deleteShader(fs)
	if d.gl.getProgramiv(id, glLinkStatus) == 0 {
		log := d.gl.getProgramInfoLog(id)
		d.gl.deleteProgram(id)
		return nil, errors.New("ShaderCompileFailed", "the shader program did not link",
			"program", name, "stage", "link", "log", log)
	}
	d.live.programs++
	return &Program{name: name, id: id, dev: d, uniforms: map[string]int32{}, attribs: map[string]int32{}}, nil
}

// compile compiles one shader stage.
func (d *Device) compile(program, stage string, kind uint32, source string) (uint32, error) {
	id := d.gl.createShader(kind)
	d.gl.shaderSource(id, source)
	d.gl.compileShader(id)
	if d.gl.getShaderiv(id, glCompileStatus) == 0 {
		log := d.gl.getShaderInfoLog(id)
		d.gl.deleteShader(id)
		return 0, errors.New("ShaderCompileFailed", "a shader did not compile",
			"program", program, "stage", stage, "log", log)
	}
	return id, nil
}

// VertexArray returns the vertex array of the name, creating it the first time
// and calling setup with it (bound) to describe its attributes. The device
// deletes it when it is closed.
//
// Flutter: no counterpart
func (d *Device) VertexArray(name string, setup func(vao uint32)) uint32 {
	if vao, ok := d.vaos[name]; ok {
		return vao
	}
	vao := d.gl.genVertexArray()
	d.vaos[name] = vao
	d.BindVertexArray(vao)
	setup(vao)
	return vao
}

// Stats says how many GL objects the device has, for tests and diagnostics.
//
// Flutter: no counterpart
type Stats struct {
	// Live objects: made and not deleted, whether in use or pooled.
	Textures, Buffers, Framebuffers, Renderbuffers, Programs int
	// Pooled objects: free, waiting to be reused.
	FreeTextures, FreeBuffers, FreeFramebuffers int
}

// Stats returns the counts of GL objects.
//
// Flutter: no counterpart
func (d *Device) Stats() Stats {
	s := Stats{
		Textures: d.live.textures, Buffers: d.live.buffers, Framebuffers: d.live.framebuffers,
		Renderbuffers: d.live.renderbuffers, Programs: d.live.programs,
	}
	for _, l := range d.textures {
		s.FreeTextures += len(l)
	}
	for _, l := range d.buffers {
		s.FreeBuffers += len(l)
	}
	for _, l := range d.frames {
		s.FreeFramebuffers += len(l)
	}
	return s
}

// EndFrame marks the end of a frame and frees the pooled objects that have not
// been used for a long time.
//
// Flutter: no counterpart
func (d *Device) EndFrame() {
	d.frame++
	d.Trim(defaultMaxIdleFrames)
}

// Trim deletes the pooled objects that have been free for more than maxIdle
// frames; 0 deletes all of them.
//
// Flutter: no counterpart
func (d *Device) Trim(maxIdle uint64) {
	old := func(last uint64) bool { return maxIdle == 0 || d.frame-last > maxIdle }
	for k, list := range d.textures {
		keep := list[:0]
		for _, t := range list {
			if old(t.lastUsed) {
				d.deleteTexture(t)
			} else {
				keep = append(keep, t)
			}
		}
		if len(keep) == 0 {
			delete(d.textures, k)
		} else {
			d.textures[k] = keep
		}
	}
	for k, list := range d.buffers {
		keep := list[:0]
		for _, b := range list {
			if old(b.lastUsed) {
				d.deleteBuffer(b)
			} else {
				keep = append(keep, b)
			}
		}
		if len(keep) == 0 {
			delete(d.buffers, k)
		} else {
			d.buffers[k] = keep
		}
	}
	for k, list := range d.frames {
		keep := list[:0]
		for _, f := range list {
			if old(f.lastUsed) {
				d.deleteFramebuffer(f)
			} else {
				keep = append(keep, f)
			}
		}
		if len(keep) == 0 {
			delete(d.frames, k)
		} else {
			d.frames[k] = keep
		}
	}
}

// Close deletes every GL object the device made that is pooled or a program. Objects
// still in use by the caller are the caller's to release first. Closing twice
// does nothing. The context must still be current.
//
// Flutter: no counterpart
func (d *Device) Close() {
	if d.closed {
		return
	}
	d.closed = true
	d.Trim(0)
	for _, vao := range d.vaos {
		if d.state.vao == vao {
			d.BindVertexArray(0)
		}
		d.gl.deleteVertexArray(vao)
	}
	d.vaos = nil
	for _, p := range d.programs {
		d.gl.deleteProgram(p.id)
		d.live.programs--
	}
	d.programs = nil
}
