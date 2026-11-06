package viewport3d

import (
	"encoding/binary"
	"log/slog"
	"math"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/viewport3d/shaders"
)

// The sizes of the uniform data, as the shaders declare them (std140-like: a vec4 is 16 bytes).
const (
	frameUniformSize   = 272
	surfaceUniformSize = 48
	lineUniformSize    = 64
	pointUniformSize   = 32
	itemUniformSize    = 64
	instanceStride     = 64
	// uniformAlign is the alignment of the offsets of the uniform buffers (the largest that a backend asks for).
	uniformAlign = 256
	// itemBase is where the records of the lines and points start in the uniform buffer, after the frame's data.
	itemBase = 2 * uniformAlign
)

// pipeKind is which shader a pipeline runs.
type pipeKind uint8

const (
	pipeTriangles pipeKind = iota
	pipeLines
	pipePoints
)

// pipeKey is what tells the pipelines of a renderer apart.
type pipeKey struct {
	kind pipeKind
	// variant is the set of attributes of a mesh of triangles: bit 0 normals, bit 1 texture coordinates, bit 2 colors.
	variant uint8
	// blend is whether the pipeline blends (transparent faces, and always for lines and points).
	blend bool
	// noCull draws the back faces too.
	noCull bool
	// noDepthTest draws over everything (the lines of the wireframe).
	noDepthTest bool
	samples     int
	color       gpu.TextureFormat
	depth       gpu.TextureFormat
}

var variantEntries = [8]string{"vs_p", "vs_pn", "vs_pu", "vs_pnu", "vs_pc", "vs_pnc", "vs_puc", "vs_pnuc"}

// pipeline returns the pipeline of a key, made the first time. A pipeline that fails to be made is logged once and is nil
// from then on.
func (r *Renderer) pipeline(k pipeKey) *gpu.RenderPipeline {
	if p, ok := r.pipes[k]; ok {
		return p
	}
	p, err := r.makePipeline(k)
	if err != nil {
		slog.Error("viewport3d: a pipeline could not be made", "err", err, "kind", int(k.kind))
	}
	r.pipes[k] = p
	return p
}

// makePipeline makes the pipeline of a key.
func (r *Renderer) makePipeline(k pipeKey) (*gpu.RenderPipeline, error) {
	if err := r.ensureModules(); err != nil {
		return nil, err
	}
	premult := gpu.BlendState{Enabled: true, SrcFactor: gpu.BlendFactorOne, DstFactor: gpu.BlendFactorOneMinusSrcAlpha}
	target := gpu.ColorTargetState{Format: k.color}
	desc := gpu.RenderPipelineDesc{Multisample: gpu.MultisampleState{Count: k.samples}}
	depth := &gpu.DepthStencilState{Format: k.depth, DepthCompare: gpu.CompareFunctionGreater}
	switch k.kind {
	case pipeTriangles:
		target.Blend = gpu.BlendState{}
		if k.blend {
			target.Blend = premult
		}
		depth.DepthWriteEnabled = !k.blend
		cull := gpu.CullModeBack
		if k.noCull {
			cull = gpu.CullModeNone
		}
		desc.Label = "viewport3d triangles"
		desc.Vertex = gpu.VertexState{Module: r.mods.mesh, EntryPoint: variantEntries[k.variant], Buffers: triangleLayouts(k.variant)}
		desc.Fragment = gpu.FragmentState{Module: r.mods.mesh, EntryPoint: "fs_main", Targets: []gpu.ColorTargetState{target}}
		desc.Primitive = gpu.PrimitiveState{CullMode: cull}
	case pipeLines:
		target.Blend = premult
		if k.noDepthTest {
			depth.DepthCompare = gpu.CompareFunctionAlways
		}
		desc.Label = "viewport3d lines"
		desc.Vertex = gpu.VertexState{Module: r.mods.line, EntryPoint: "vs_main", Buffers: []gpu.VertexBufferLayout{
			{ArrayStride: segmentFloats * 4, PerInstance: true, Attributes: []gpu.VertexAttribute{
				{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x3, Offset: 0},
				{ShaderLocation: 1, Format: gpu.VertexFormatFloat32x3, Offset: 12},
				{ShaderLocation: 2, Format: gpu.VertexFormatFloat32, Offset: 24},
			}},
			{ArrayStride: 8, Attributes: []gpu.VertexAttribute{{ShaderLocation: 3, Format: gpu.VertexFormatFloat32x2, Offset: 0}}},
		}}
		desc.Fragment = gpu.FragmentState{Module: r.mods.line, EntryPoint: "fs_main", Targets: []gpu.ColorTargetState{target}}
	default:
		target.Blend = premult
		desc.Label = "viewport3d points"
		desc.Vertex = gpu.VertexState{Module: r.mods.point, EntryPoint: "vs_main", Buffers: []gpu.VertexBufferLayout{
			{ArrayStride: 12, PerInstance: true, Attributes: []gpu.VertexAttribute{{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x3, Offset: 0}}},
			{ArrayStride: 8, Attributes: []gpu.VertexAttribute{{ShaderLocation: 1, Format: gpu.VertexFormatFloat32x2, Offset: 0}}},
		}}
		desc.Fragment = gpu.FragmentState{Module: r.mods.point, EntryPoint: "fs_main", Targets: []gpu.ColorTargetState{target}}
	}
	desc.DepthStencil = depth
	return r.dev.NewRenderPipeline(desc)
}

// triangleLayouts are the vertex buffers of a pipeline of triangles for a set of attributes: one buffer for each attribute, in
// the order position, normal, texture coordinates, color, and the instance buffer last.
func triangleLayouts(variant uint8) []gpu.VertexBufferLayout {
	layouts := []gpu.VertexBufferLayout{{ArrayStride: 12, Attributes: []gpu.VertexAttribute{{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x3}}}}
	if variant&1 != 0 {
		layouts = append(layouts, gpu.VertexBufferLayout{ArrayStride: 12, Attributes: []gpu.VertexAttribute{{ShaderLocation: 1, Format: gpu.VertexFormatFloat32x3}}})
	}
	if variant&2 != 0 {
		layouts = append(layouts, gpu.VertexBufferLayout{ArrayStride: 8, Attributes: []gpu.VertexAttribute{{ShaderLocation: 2, Format: gpu.VertexFormatFloat32x2}}})
	}
	if variant&4 != 0 {
		layouts = append(layouts, gpu.VertexBufferLayout{ArrayStride: 4, Attributes: []gpu.VertexAttribute{{ShaderLocation: 3, Format: gpu.VertexFormatUnorm8x4}}})
	}
	return append(layouts, gpu.VertexBufferLayout{ArrayStride: instanceStride, PerInstance: true, Attributes: []gpu.VertexAttribute{
		{ShaderLocation: 4, Format: gpu.VertexFormatFloat32x4, Offset: 0},
		{ShaderLocation: 5, Format: gpu.VertexFormatFloat32x4, Offset: 16},
		{ShaderLocation: 6, Format: gpu.VertexFormatFloat32x4, Offset: 32},
		{ShaderLocation: 7, Format: gpu.VertexFormatFloat32x4, Offset: 48},
	}})
}

// ensureModules makes the shader modules, once.
func (r *Renderer) ensureModules() error {
	if r.mods.mesh != nil {
		return nil
	}
	var err error
	if r.mods.mesh, err = r.dev.NewShaderModule(shaders.Mesh); err != nil {
		return err
	}
	if r.mods.line, err = r.dev.NewShaderModule(shaders.Line); err != nil {
		return err
	}
	r.mods.point, err = r.dev.NewShaderModule(shaders.Point)
	return err
}

// ensureShared makes what the draws share: the corners of the quads of lines and points, a white texture and a sampler.
func (r *Renderer) ensureShared() error {
	if r.corners != nil {
		return nil
	}
	lineCorners := []float32{0, -1, 1, -1, 1, 1, 0, -1, 1, 1, 0, 1}
	pointCorners := []float32{-1, -1, 1, -1, 1, 1, -1, -1, 1, 1, -1, 1}
	var err error
	if r.corners, err = r.newBuffer("viewport3d line corners", floatBytes(nil, lineCorners), gpu.BufferUsageVertex); err != nil {
		return err
	}
	if r.pointCorners, err = r.newBuffer("viewport3d point corners", floatBytes(nil, pointCorners), gpu.BufferUsageVertex); err != nil {
		return err
	}
	if r.sampler, err = r.dev.NewSampler(gpu.SamplerDesc{Label: "viewport3d", AddressModeU: gpu.AddressModeRepeat, AddressModeV: gpu.AddressModeRepeat,
		MagFilter: gpu.FilterModeLinear, MinFilter: gpu.FilterModeLinear, MipmapFilter: gpu.FilterModeLinear}); err != nil {
		return err
	}
	r.white, err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d white", Size: gpu.Extent3D{Width: 1, Height: 1, DepthOrArrayLayers: 1},
		Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		return err
	}
	r.dev.Queue().WriteTexture(gpu.TextureCopy{Texture: r.white}, []byte{255, 255, 255, 255}, gpu.DataLayout{BytesPerRow: 4}, gpu.Extent3D{Width: 1, Height: 1, DepthOrArrayLayers: 1})
	return nil
}

// newBuffer makes a buffer with the data, for the usage.
func (r *Renderer) newBuffer(label string, data []byte, usage gpu.BufferUsage) (*gpu.Buffer, error) {
	size := (len(data) + 3) &^ 3
	if size == 0 {
		size = 4
	}
	b, err := r.dev.NewBuffer(gpu.BufferDesc{Label: label, Size: size, Usage: usage | gpu.BufferUsageCopyDst})
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		if len(data)%4 != 0 {
			padded := make([]byte, size)
			copy(padded, data)
			data = padded
		}
		r.dev.Queue().WriteBuffer(b, 0, data)
	}
	return b, nil
}

// floatBytes appends the float32 values as little-endian bytes to dst.
func floatBytes(dst []byte, src []float32) []byte {
	n := len(dst)
	dst = append(dst, make([]byte, 4*len(src))...)
	for i, v := range src {
		binary.LittleEndian.PutUint32(dst[n+4*i:], math.Float32bits(v))
	}
	return dst
}

// meshGPU is what a renderer holds on the GPU for a mesh.
type meshGPU struct {
	id uint32
	// variant is the set of attributes, for a mesh of triangles.
	variant uint8
	pos     *gpu.Buffer
	normals *gpu.Buffer
	uvs     *gpu.Buffer
	colors  *gpu.Buffer
	index   *gpu.Buffer
	wide    bool
	// segments are the segments of a mesh of lines, for the instances of the line shader.
	segments *gpu.Buffer
	// count is the number of indices (or vertices), segments or points to draw.
	count int
}

// release frees the buffers of the mesh.
func (g *meshGPU) release() {
	for _, b := range []*gpu.Buffer{g.pos, g.normals, g.uvs, g.colors, g.index, g.segments} {
		if b != nil {
			b.Release()
		}
	}
	*g = meshGPU{}
}

// meshState returns what the renderer holds for a mesh, uploading the mesh the first time.
func (r *Renderer) meshState(m *Mesh) *meshGPU {
	for i, x := range m.renderers {
		if x == r {
			return m.states[i]
		}
	}
	g := r.uploadMesh(m)
	m.renderers = append(m.renderers, r)
	m.states = append(m.states, g)
	r.meshes = append(r.meshes, m)
	return g
}

// uploadMesh makes the buffers of a mesh.
func (r *Renderer) uploadMesh(m *Mesh) *meshGPU {
	r.nextID++
	g := &meshGPU{id: r.nextID}
	d := &m.data
	var err error
	fail := func(what string) {
		slog.Error("viewport3d: a mesh could not be uploaded", "err", err, "what", what)
		g.release()
		g.count = 0
	}
	switch d.Primitive {
	case PrimitiveLines:
		g.segments, err = r.newBuffer("viewport3d segments", floatBytes(nil, m.segments(r.segs)), gpu.BufferUsageVertex)
		g.count = m.IndexCount() / 2
	case PrimitivePoints:
		var pos []float32
		pos = d.Positions
		if m.indexed {
			// The points are the vertices that the indices name.
			pos = make([]float32, 0, 3*m.IndexCount())
			for i := 0; i < m.IndexCount(); i++ {
				v := m.index(i)
				pos = append(pos, d.Positions[3*v], d.Positions[3*v+1], d.Positions[3*v+2])
			}
		}
		g.pos, err = r.newBuffer("viewport3d points", floatBytes(nil, pos), gpu.BufferUsageVertex)
		g.count = len(pos) / 3
	default:
		if g.pos, err = r.newBuffer("viewport3d positions", floatBytes(nil, d.Positions), gpu.BufferUsageVertex); err != nil {
			fail("positions")
			return g
		}
		if len(d.Normals) != 0 {
			g.variant |= 1
			if g.normals, err = r.newBuffer("viewport3d normals", floatBytes(nil, d.Normals), gpu.BufferUsageVertex); err != nil {
				fail("normals")
				return g
			}
		}
		if len(d.TexCoords) != 0 {
			g.variant |= 2
			if g.uvs, err = r.newBuffer("viewport3d texture coordinates", floatBytes(nil, d.TexCoords), gpu.BufferUsageVertex); err != nil {
				fail("texture coordinates")
				return g
			}
		}
		if len(d.Colors) != 0 {
			g.variant |= 4
			if g.colors, err = r.newBuffer("viewport3d colors", d.Colors, gpu.BufferUsageVertex); err != nil {
				fail("colors")
				return g
			}
		}
		g.count = m.IndexCount()
		switch {
		case d.Indices16 != nil && m.indexed:
			b := make([]byte, 2*len(d.Indices16))
			for i, v := range d.Indices16 {
				binary.LittleEndian.PutUint16(b[2*i:], v)
			}
			g.index, err = r.newBuffer("viewport3d indices", b, gpu.BufferUsageIndex)
		case d.Indices32 != nil && m.indexed:
			b := make([]byte, 4*len(d.Indices32))
			for i, v := range d.Indices32 {
				binary.LittleEndian.PutUint32(b[4*i:], v)
			}
			g.index, err = r.newBuffer("viewport3d indices", b, gpu.BufferUsageIndex)
			g.wide = true
		}
	}
	if err != nil {
		fail("buffers")
	}
	return g
}

// forgetMesh lets go of what the renderer holds for a mesh (Mesh.Release).
func (r *Renderer) forgetMesh(m *Mesh) {
	for i, x := range m.renderers {
		if x == r {
			m.states[i].release()
			m.renderers = append(m.renderers[:i], m.renderers[i+1:]...)
			m.states = append(m.states[:i], m.states[i+1:]...)
			break
		}
	}
	for i, x := range r.meshes {
		if x == m {
			r.meshes = append(r.meshes[:i], r.meshes[i+1:]...)
			break
		}
	}
}

// matGPU is what a renderer holds on the GPU for a material: its uniform buffer and its bind group.
type matGPU struct {
	id      uint32
	buf     *gpu.Buffer
	group   *gpu.BindGroup
	version uint64
	// texture is the texture the group was made with.
	texture *gpu.Texture
	// bytes are the uniforms that the buffer holds.
	bytes [lineUniformSize]byte
	// pipe is the pipeline whose layout the group was made for, which keeps the layout's owner alive.
	pipe *gpu.RenderPipeline
}

func (g *matGPU) release() {
	if g.group != nil {
		g.group.Release()
	}
	if g.buf != nil {
		g.buf.Release()
	}
	*g = matGPU{}
}

// putVec4 writes four float32 numbers.
func putVec4(b []byte, off int, x, y, z, w float32) {
	binary.LittleEndian.PutUint32(b[off:], math.Float32bits(x))
	binary.LittleEndian.PutUint32(b[off+4:], math.Float32bits(y))
	binary.LittleEndian.PutUint32(b[off+8:], math.Float32bits(z))
	binary.LittleEndian.PutUint32(b[off+12:], math.Float32bits(w))
}

// putColor writes a color as four numbers, not premultiplied.
func putColor(b []byte, off int, c ui.Color) {
	putVec4(b, off, float32(c.R), float32(c.G), float32(c.B), float32(c.A))
}

// putMat writes a 4x4 matrix.
func putMat(b []byte, off int, m *[16]float32) {
	for i, v := range m {
		binary.LittleEndian.PutUint32(b[off+4*i:], math.Float32bits(v))
	}
}

// surfaceOf returns the surface material of a material of triangles, or nil.
func surfaceOf(m Material) *surfaceMaterial {
	switch x := m.(type) {
	case *UnlitMaterial:
		return &x.surfaceMaterial
	case *PhongMaterial:
		return &x.surfaceMaterial
	}
	return nil
}

// uniformsOf writes the uniform data of a material into dst and returns the size.
func uniformsOf(m Material, dst *[lineUniformSize]byte) int {
	switch x := m.(type) {
	case *UnlitMaterial:
		return surfaceUniforms(&x.surfaceMaterial, dst)
	case *PhongMaterial:
		return surfaceUniforms(&x.surfaceMaterial, dst)
	case *LineMaterial:
		return lineUniforms(x.color, x.width, x.dashes, &x.stipple, x.pull, dst)
	case *PointMaterial:
		square := float32(0)
		if x.shape == PointShapeSquare {
			square = 1
		}
		putColor(dst[:], 0, x.color)
		putVec4(dst[:], 16, float32(x.size), square, 0, 0)
		return pointUniformSize
	}
	return 0
}

// surfaceUniforms writes the data of a material of triangles.
func surfaceUniforms(m *surfaceMaterial, dst *[lineUniformSize]byte) int {
	b := dst[:]
	putColor(b, 0, m.color)
	spec := m.specular
	putVec4(b, 16, float32(spec.R)*float32(spec.A), float32(spec.G)*float32(spec.A), float32(spec.B)*float32(spec.A), float32(m.shininess))
	flag := func(on bool) float32 {
		if on {
			return 1
		}
		return 0
	}
	opaque := !m.Transparent()
	putVec4(b, 32, flag(m.lit), flag(m.vertexColors), flag(m.texture != nil), flag(opaque))
	return surfaceUniformSize
}

// lineUniforms writes the data of a line material: the color, the width, the number and the sum of the dash lengths, the
// pull towards the camera, and the pattern.
func lineUniforms(c ui.Color, width float64, dashes int, pattern *[maxStipple]float64, pull float32, dst *[lineUniformSize]byte) int {
	b := dst[:]
	sum := 0.0
	for i := 0; i < dashes; i++ {
		sum += pattern[i]
	}
	putColor(b, 0, c)
	putVec4(b, 16, float32(width), float32(dashes), pull, float32(sum))
	for i := 0; i < maxStipple; i += 4 {
		putVec4(b, 32+4*i, float32(pattern[i]), float32(pattern[i+1]), float32(pattern[i+2]), float32(pattern[i+3]))
	}
	return lineUniformSize
}

// materialState returns what the renderer holds for a material, making it the first time and updating it when the
// material changed. pipe is a pipeline of the kind that draws with the material: its layout is the bind group's.
func (r *Renderer) materialState(m Material, pipe *gpu.RenderPipeline) *matGPU {
	c := m.core()
	for i, x := range c.renderers {
		if x == r {
			g := c.states[i]
			r.refreshMaterial(g, m, pipe)
			return g
		}
	}
	r.nextID++
	g := &matGPU{id: r.nextID}
	c.renderers = append(c.renderers, r)
	c.states = append(c.states, g)
	r.mats = append(r.mats, c)
	r.refreshMaterial(g, m, pipe)
	return g
}

// refreshMaterial makes the buffer and the bind group of a material, or writes its uniforms when they changed.
func (r *Renderer) refreshMaterial(g *matGPU, m Material, pipe *gpu.RenderPipeline) {
	if pipe == nil {
		return
	}
	if g.group != nil && g.version == m.core().version {
		return // nothing changed since the last frame
	}
	var data [lineUniformSize]byte
	size := uniformsOf(m, &data)
	var tex *gpu.Texture
	if s := surfaceOf(m); s != nil {
		tex = s.texture
	}
	if g.buf == nil {
		var err error
		if g.buf, err = r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d material", Size: lineUniformSize, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}); err != nil {
			slog.Error("viewport3d: a material could not be made", "err", err)
			return
		}
	}
	if data != g.bytes || g.group == nil {
		r.dev.Queue().WriteBuffer(g.buf, 0, data[:])
		g.bytes = data
	}
	if g.group == nil || g.texture != tex {
		if g.group != nil {
			g.group.Release()
			g.group = nil
		}
		entries := []gpu.BindGroupEntry{{Binding: 0, Buffer: g.buf, Size: size}}
		if surfaceOf(m) != nil {
			view := r.white.View()
			if tex != nil {
				view = tex.View()
			}
			entries = append(entries, gpu.BindGroupEntry{Binding: 1, TextureView: view}, gpu.BindGroupEntry{Binding: 2, Sampler: r.sampler})
		}
		group, err := r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d material", Layout: pipe.BindGroupLayout(1), Entries: entries})
		if err != nil {
			slog.Error("viewport3d: a material could not be made", "err", err)
			return
		}
		g.group, g.texture, g.pipe = group, tex, pipe
	}
	g.version = m.core().version
}

// forgetMaterial lets go of what the renderer holds for a material (Material.Release).
func (r *Renderer) forgetMaterial(c *materialCore) {
	for i, x := range c.renderers {
		if x == r {
			c.states[i].release()
			c.renderers = append(c.renderers[:i], c.renderers[i+1:]...)
			c.states = append(c.states[:i], c.states[i+1:]...)
			break
		}
	}
	for i, x := range r.mats {
		if x == c {
			r.mats = append(r.mats[:i], r.mats[i+1:]...)
			break
		}
	}
}
