package viewport3d

import (
	"cmp"
	"encoding/binary"
	"log/slog"
	"math"
	"slices"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d/shaders"
)

// pbrVertexStride is the size of a vertex of the buffer that PBR items are drawn from: position (12 bytes), normal (12), tangent
// (16), two texture coordinates (16), color (4), joints (8) and weights (16). What a mesh lacks is zero (the color is white), and
// the item's flags say what it has.
const pbrVertexStride = 84

// The bits of what a mesh has, as pbr.wgsl reads them from the instance.
const (
	pbrHasNormal  = 1
	pbrHasTangent = 2
	pbrHasUV0     = 4
	pbrHasUV1     = 8
	pbrSkinned    = 32
)

// pbrFrameSize is the size of the PBRFrame uniform of pbr.wgsl.
const pbrFrameSize = 176

// pbrLayouts are the vertex buffers of the PBR pipeline: the buffer of the mesh, and the instances.
func pbrLayouts() []gpu.VertexBufferLayout {
	return []gpu.VertexBufferLayout{
		{ArrayStride: pbrVertexStride, Attributes: []gpu.VertexAttribute{
			{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x3, Offset: 0},
			{ShaderLocation: 1, Format: gpu.VertexFormatFloat32x3, Offset: 12},
			{ShaderLocation: 2, Format: gpu.VertexFormatFloat32x4, Offset: 24},
			{ShaderLocation: 3, Format: gpu.VertexFormatFloat32x2, Offset: 40},
			{ShaderLocation: 4, Format: gpu.VertexFormatFloat32x2, Offset: 48},
			{ShaderLocation: 5, Format: gpu.VertexFormatUnorm8x4, Offset: 56},
			{ShaderLocation: 6, Format: gpu.VertexFormatUint16x4, Offset: 60},
			{ShaderLocation: 7, Format: gpu.VertexFormatFloat32x4, Offset: 68},
		}},
		{ArrayStride: instanceStride, PerInstance: true, Attributes: []gpu.VertexAttribute{
			{ShaderLocation: 8, Format: gpu.VertexFormatFloat32x4, Offset: 0},
			{ShaderLocation: 9, Format: gpu.VertexFormatFloat32x4, Offset: 16},
			{ShaderLocation: 10, Format: gpu.VertexFormatFloat32x4, Offset: 32},
			{ShaderLocation: 11, Format: gpu.VertexFormatFloat32x4, Offset: 48},
			{ShaderLocation: 12, Format: gpu.VertexFormatFloat32x4, Offset: 64},
			{ShaderLocation: 13, Format: gpu.VertexFormatFloat32x4, Offset: 80},
		}},
	}
}

// ensurePBRModules makes the modules of the PBR pipelines, the shadow map and the tone mapping, when the first one is needed.
func (r *Renderer) ensurePBRModules() error {
	if r.mods.pbr == nil {
		var err error
		if r.mods.pbr, err = r.dev.NewShaderModule(shaders.Pbr); err != nil {
			return err
		}
	}
	if err := r.ensureShadowModule(); err != nil {
		return err
	}
	return r.ensureToneModule()
}

// releaser is what a frame lets go of when it is done: a texture, a buffer, a bind group or a pipeline.
type releaser interface{ Release() }

// later keeps a resource that the commands of this frame name until the next frame, when they have been submitted: a resource that
// is released is not to be used by a command, even one that is encoded already.
func (r *Renderer) later(x releaser) { r.pending = append(r.pending, x) }

// releasePending lets go of what the last frame kept.
func (r *Renderer) releasePending() {
	for i, x := range r.pending {
		x.Release()
		r.pending[i] = nil
	}
	r.pending = r.pending[:0]
}

// pbrSig is what the PBR frame group was made with.
type pbrSig struct {
	spec, irr, shadow   *gpu.Texture
	samp                *gpu.Sampler
	frame, joints, wghs *gpu.Buffer
}

// pbrState is what a renderer holds for the PBR materials.
type pbrState struct {
	env    envState
	hdr    hdrState
	shadow shadowState

	// frame is the uniform of the PBR frame, group the group of the environment, the shadow map and the skins, and sig what the group
	// names.
	frame *gpu.Buffer
	group *gpu.BindGroup
	sig   pbrSig
	// joints and weights are the storage buffers of the joint matrices and the morph weights of the frame's items, with their sizes.
	joints, weights *gpu.Buffer
	jointCap, wCap  int
	// jf and wf are their contents for the frame, and jb and wb the same as bytes.
	jf, wf []float32
	jb, wb []byte
	jm     []vecmath.Mat4

	// zero is the buffer of the deltas of a mesh with no morph targets, and zeroGroup and zeroShadow the groups that name it.
	zero                  *gpu.Buffer
	zeroGroup, zeroShadow *gpu.BindGroup

	// casters are the items of the shadow pass, the meshes of the other materials first.
	simple, full []shadowCaster
	light        shadowLight
	lightOn      bool

	colors [1]gpu.RenderPassColorAttachment
	ds     gpu.RenderPassDepthStencilAttachment
	// frameBytes and toneBytes are the uniform data of the frame while it is written.
	frameBytes [pbrFrameSize]byte
	toneBytes  [16]byte
}

// reset empties the lists of a frame.
func (p *pbrState) reset() {
	p.jf, p.wf = p.jf[:0], p.wf[:0]
	p.simple, p.full = p.simple[:0], p.full[:0]
}

// release lets go of everything that the state holds (Renderer.Dispose).
func (p *pbrState) release(r *Renderer) {
	r.dropPBRGroup()
	p.env.release(r)
	p.hdr.release(r)
	for _, t := range []*gpu.Texture{p.shadow.color, p.shadow.depth, p.shadow.dummy} {
		if t != nil {
			t.Release()
		}
	}
	for _, g := range []*gpu.BindGroup{p.shadow.group, p.zeroGroup, p.zeroShadow} {
		if g != nil {
			g.Release()
		}
	}
	for _, b := range []*gpu.Buffer{p.frame, p.joints, p.weights, p.zero} {
		if b != nil {
			b.Release()
		}
	}
	*p = pbrState{}
}

// dropPBRGroup lets go of the group of the environment and the shadow map: the next frame makes it again.
func (r *Renderer) dropPBRGroup() {
	p := &r.pbr
	for _, g := range []**gpu.BindGroup{&p.group, &p.shadow.group} {
		if *g != nil {
			r.later(*g)
			*g = nil
		}
	}
	p.sig = pbrSig{}
}

// drawCtx is what the items of a frame share while the list is built.
type drawCtx struct {
	style *resolvedStyle
	base  pipeKey
	// pbrKey is the key of the PBR pipelines (without the culling and the blending), and pbrPipe the pipeline of the key, made
	// when the first PBR item is met.
	pbrKey  pipeKey
	pbrPipe *gpu.RenderPipeline
	pbrDone bool
	// shadowOn says that the frame has a shadow map: the items are casters.
	shadowOn bool
	eye      vecmath.Vec3
	planes   int
	capsOK   bool
}

// pbrPipeline returns the PBR pipeline of the frame, made the first time.
func (r *Renderer) pbrPipeline(dc *drawCtx) *gpu.RenderPipeline {
	if !dc.pbrDone {
		dc.pbrDone = true
		dc.pbrKey = dc.base
		dc.pbrKey.kind, dc.pbrKey.color = pipePBR, gpu.TextureFormatRGBA16Float
		dc.pbrPipe = r.pipeline(dc.pbrKey)
	}
	return dc.pbrPipe
}

// ensureFull makes the buffer of a mesh with the layout of the PBR shader, and the deltas of its morph targets.
func (r *Renderer) ensureFull(g *meshGPU, m *Mesh) bool {
	if g.full != nil {
		return true
	}
	if g.fullFailed {
		return false
	}
	d := &m.data
	n := len(d.Positions) / 3
	buf := make([]byte, n*pbrVertexStride)
	put := func(off int, v float32) { binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(v)) }
	g.flags = 0
	if len(d.Normals) != 0 {
		g.flags |= pbrHasNormal
	}
	if len(d.Tangents) != 0 {
		g.flags |= pbrHasTangent
	}
	if len(d.TexCoords) != 0 {
		g.flags |= pbrHasUV0
	}
	if len(d.TexCoords1) != 0 {
		g.flags |= pbrHasUV1
	}
	for v := 0; v < n; v++ {
		o := v * pbrVertexStride
		for k := 0; k < 3; k++ {
			put(o+4*k, d.Positions[3*v+k])
			if len(d.Normals) != 0 {
				put(o+12+4*k, d.Normals[3*v+k])
			}
		}
		if len(d.Tangents) != 0 {
			for k := 0; k < 4; k++ {
				put(o+24+4*k, d.Tangents[4*v+k])
			}
		}
		if len(d.TexCoords) != 0 {
			put(o+40, d.TexCoords[2*v])
			put(o+44, d.TexCoords[2*v+1])
		}
		if len(d.TexCoords1) != 0 {
			put(o+48, d.TexCoords1[2*v])
			put(o+52, d.TexCoords1[2*v+1])
		}
		if len(d.Colors) != 0 {
			copy(buf[o+56:o+60], d.Colors[4*v:4*v+4])
		} else {
			copy(buf[o+56:o+60], []byte{255, 255, 255, 255})
		}
		if len(d.Joints) != 0 {
			for k := 0; k < 4; k++ {
				j := d.Joints[4*v+k]
				binary.LittleEndian.PutUint16(buf[o+60+2*k:], j)
				if d.Weights[4*v+k] > 0 {
					g.maxJoint = max(g.maxJoint, int(j))
				}
				put(o+68+4*k, d.Weights[4*v+k])
			}
		}
	}
	g.joints = len(d.Joints) != 0
	var err error
	if g.full, err = r.newBuffer("viewport3d pbr vertices", buf, gpu.BufferUsageVertex); err != nil {
		slog.Error("viewport3d: a mesh could not be uploaded", "err", err, "what", "pbr vertices")
		g.fullFailed = true
		return false
	}
	g.targets = len(d.MorphTargets)
	if g.targets > 0 {
		// For each vertex and target, the change of the position and then of the normal, as vec4.
		deltas := make([]byte, n*g.targets*32)
		for t, mt := range d.MorphTargets {
			for v := 0; v < n; v++ {
				o := (v*g.targets + t) * 32
				for k := 0; k < 3; k++ {
					if len(mt.Positions) != 0 {
						binary.LittleEndian.PutUint32(deltas[o+4*k:], math.Float32bits(mt.Positions[3*v+k]))
					}
					if len(mt.Normals) != 0 {
						binary.LittleEndian.PutUint32(deltas[o+16+4*k:], math.Float32bits(mt.Normals[3*v+k]))
					}
				}
			}
		}
		if g.deltas, err = r.newBuffer("viewport3d morph deltas", deltas, gpu.BufferUsageStorage); err != nil {
			slog.Error("viewport3d: a mesh could not be uploaded", "err", err, "what", "morph deltas")
			g.targets = 0
		}
	}
	return true
}

// morphGroup returns the group of the morph deltas of a mesh for the pipeline's layout of group 3 (group 1 of the shadow pipeline), or
// the one of the zero deltas for a mesh with none.
func (r *Renderer) morphGroup(g *meshGPU, pipe *gpu.RenderPipeline, shadow bool) *gpu.BindGroup {
	p := &r.pbr
	slot, layout := &g.morphGroup, 3
	zero := &p.zeroGroup
	if shadow {
		slot, layout, zero = &g.morphShadow, 1, &p.zeroShadow
	}
	buf := g.deltas
	if buf == nil {
		slot = zero
		if p.zero == nil {
			var err error
			if p.zero, err = r.newBuffer("viewport3d no morph deltas", make([]byte, 32), gpu.BufferUsageStorage); err != nil {
				slog.Error("viewport3d: the morph deltas could not be made", "err", err)
				return nil
			}
		}
		buf = p.zero
	}
	if *slot == nil {
		grp, err := r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d morph deltas", Layout: pipe.BindGroupLayout(layout), Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: buf}}})
		if err != nil {
			slog.Error("viewport3d: the morph deltas could not be made", "err", err)
			return nil
		}
		*slot = grp
	}
	return *slot
}

// pbrExtra makes the four numbers that the instance of a PBR item carries (the first of its joint matrices, the first of its morph
// weights, the number of morph targets, and the flags of what the mesh has), putting the joint matrices and the weights in the
// frame's lists.
func (r *Renderer) pbrExtra(it *Item, g *meshGPU) [4]float32 {
	p := &r.pbr
	flags := g.flags
	var joint, weight float32
	if g.joints && it.skin != nil && g.maxJoint < it.skin.JointCount() {
		flags |= pbrSkinned
		joint = float32(len(p.jf) / 16)
		n := it.skin.JointCount()
		p.jm = slices.Grow(p.jm[:0], n)[:n]
		it.skin.JointMatrices(it.transform, it.mesh.data.Origin, p.jm)
		for i := range p.jm {
			f := p.jm[i].Float32()
			p.jf = append(p.jf, f[:]...)
		}
	}
	targets := 0
	if g.targets > 0 {
		for _, w := range it.morphWeights {
			if w != 0 {
				targets = g.targets
				break
			}
		}
		if targets > 0 {
			weight = float32(len(p.wf))
			for t := 0; t < targets; t++ {
				w := float32(0)
				if t < len(it.morphWeights) {
					w = it.morphWeights[t]
				}
				p.wf = append(p.wf, w)
			}
		}
	}
	return [4]float32{joint, weight, float32(targets), float32(flags)}
}

// addPBRItem adds an item with a PBR material to the frame's lists. The mesh must have its buffers (meshState).
func (r *Renderer) addPBRItem(it *Item, pm *PBRMaterial, mesh *meshGPU, mv *[16]float32, dc *drawCtx) {
	pipe := r.pbrPipeline(dc)
	if pipe == nil || !r.ensureFull(mesh, it.mesh) {
		return
	}
	mat := r.materialState(pm, pipe)
	if mat.group == nil {
		return
	}
	pk := dc.pbrKey
	pk.noCull = pm.doubleSided
	blend := pm.alphaMode == AlphaModeBlend
	pk.blend = blend
	r.items = append(r.items, drawItem{mesh: mesh, mat: mat, pk: pk, mv: *mv, pbr: true})
	index := int32(len(r.items) - 1)
	d := &r.items[index]
	d.extra = r.pbrExtra(it, mesh)
	if it.selected {
		d.selected, d.tint = true, tintOf(&dc.style.selection)
		if dc.style.selection.outline {
			r.selItems = append(r.selItems, it)
		}
	}
	if blend {
		d.dist = it.sphere.Center.DistanceSqTo(dc.eye)
		r.ptransp = append(r.ptransp, index)
	} else {
		sig := uint64(0)
		if pk.noCull {
			sig |= 8
		}
		if d.selected {
			sig |= 16
		}
		d.key = sig<<56 | uint64(mat.id&0xFFFFFF)<<32 | uint64(mesh.id)
		r.popaque = append(r.popaque, index)
	}
	r.stats.Items++
	if dc.planes > 0 {
		sd := sectionDraw{mesh: it.mesh, g: mesh, mv: *mv}
		if dc.capsOK && !blend && it.mesh.Closed() {
			sd.capped = true
			sd.cap = dc.style.section.capColor.OrElse(pm.color)
		}
		r.sect = append(r.sect, sd)
	}
	if dc.shadowOn && !blend {
		r.addCaster(it, mesh, true, d.extra)
	}
}

// addCaster adds an item to the shadow pass.
func (r *Renderer) addCaster(it *Item, mesh *meshGPU, full bool, extra [4]float32) {
	p := &r.pbr
	var c shadowCaster
	c.mesh, c.full, c.extra = mesh, full, extra
	// The matrix of the item to the clip space of the light, made in float64 from the world's.
	m := p.light.lightWorld.Mul(it.model)
	for i, v := range m {
		c.mv[i] = float32(v)
	}
	if full {
		p.full = append(p.full, c)
	} else {
		p.simple = append(p.simple, c)
	}
}

// floatsToBytes writes the numbers as little-endian bytes into the scratch, which it grows, and returns them.
func floatsToBytes(scratch *[]byte, src []float32) []byte {
	need := 4 * len(src)
	if cap(*scratch) < need {
		*scratch = make([]byte, need, need*2)
	}
	b := (*scratch)[:need]
	for i, v := range src {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	return b
}

// ensureStorage makes the buffers of the joint matrices and the morph weights, with room for the frame's, and uploads them.
func (r *Renderer) ensureStorage() bool {
	p := &r.pbr
	need := func(b **gpu.Buffer, have *int, size, minimum int, label string) bool {
		if *b != nil && *have >= size {
			return true
		}
		if *b != nil {
			r.later(*b)
			*b = nil
			r.dropPBRGroup()
		}
		c := max(minimum, size*2)
		nb, err := r.dev.NewBuffer(gpu.BufferDesc{Label: label, Size: c, Usage: gpu.BufferUsageStorage | gpu.BufferUsageCopyDst})
		if err != nil {
			slog.Error("viewport3d: a buffer could not be made", "err", err, "what", label)
			return false
		}
		*b, *have = nb, c
		return true
	}
	if !need(&p.joints, &p.jointCap, 4*len(p.jf), 64*16, "viewport3d joint matrices") || !need(&p.weights, &p.wCap, 4*len(p.wf), 256, "viewport3d morph weights") {
		return false
	}
	q := r.dev.Queue()
	if len(p.jf) > 0 {
		q.WriteBuffer(p.joints, 0, floatsToBytes(&p.jb, p.jf))
	}
	if len(p.wf) > 0 {
		q.WriteBuffer(p.weights, 0, floatsToBytes(&p.wb, p.wf))
	}
	return true
}

// ensureShadowDummy makes the 1 by 1 map that stands for a shadow map when there is none.
func (r *Renderer) ensureShadowDummy() bool {
	s := &r.pbr.shadow
	if s.dummy != nil {
		return true
	}
	t, err := r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d no shadow", Size: gpu.Extent3D{Width: 1, Height: 1, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatR32Float,
		Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		slog.Error("viewport3d: a texture could not be made", "err", err)
		return false
	}
	r.dev.Queue().WriteTexture(gpu.TextureCopy{Texture: t}, floatBytes(nil, []float32{1}), gpu.DataLayout{BytesPerRow: 4}, gpu.Extent3D{Width: 1, Height: 1, DepthOrArrayLayers: 1})
	s.dummy = t
	return true
}

// pbrGroup returns the group of the frame's environment, shadow map and skins for the layout of the PBR pipeline, made again when
// what it names changed.
func (r *Renderer) pbrGroup(pipe *gpu.RenderPipeline, shadowMap *gpu.Texture) *gpu.BindGroup {
	p := &r.pbr
	sig := pbrSig{spec: p.env.spec, irr: p.env.irr, shadow: shadowMap, samp: p.env.samp, frame: p.frame, joints: p.joints, wghs: p.weights}
	if p.group != nil && p.sig == sig {
		return p.group
	}
	if p.group != nil {
		r.later(p.group)
		p.group = nil
	}
	grp, err := r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d pbr frame", Layout: pipe.BindGroupLayout(2), Entries: []gpu.BindGroupEntry{
		{Binding: 0, Buffer: p.frame, Size: pbrFrameSize},
		{Binding: 1, TextureView: p.env.spec.View()},
		{Binding: 2, TextureView: p.env.irr.View()},
		{Binding: 3, Sampler: p.env.samp},
		{Binding: 4, TextureView: shadowMap.View()},
		{Binding: 5, Buffer: p.joints},
		{Binding: 6, Buffer: p.weights},
	}})
	if err != nil {
		slog.Error("viewport3d: the group of the PBR frame could not be made", "err", err)
		return nil
	}
	p.group, p.sig = grp, sig
	return grp
}

// writePBRFrame writes the uniform of the PBR frame: the matrix from the view space to the shadow map, the one from the view's
// directions to the world's, and the numbers of the environment and the shadow.
func (r *Renderer) writePBRFrame(style *resolvedStyle, rot *vecmath.Mat4, eye vecmath.Vec3, shadowOn bool) {
	p := &r.pbr
	b := &p.frameBytes
	clear(b[:])
	rotT := rot.Transpose()
	var vp [16]float32
	index, bias, offset, strength, size := float32(-1), float32(0), float32(0), float32(0), float32(1)
	if shadowOn {
		m := p.light.lightWorld.Mul(vecmath.NewMat4Translation(eye)).Mul(rotT)
		vp = m.Float32()
		index, bias, offset = float32(p.light.index), float32(p.light.bias), float32(p.light.offset)
		strength, size = float32(style.shadow.strength), float32(p.shadow.size)
	}
	putMat(b[:], 0, &vp)
	w := rotT.Float32()
	putMat(b[:], 64, &w)
	putVec4(b[:], 128, float32(style.envIntensity), index, bias, offset)
	putVec4(b[:], 144, strength, size, 0, 0)
	putVec4(b[:], 160, specularLevels-1, 0, 0, 0)
	r.dev.Queue().WriteBuffer(p.frame, 0, b[:])
}

// drawShadow draws the casters into the shadow map, from the light.
func (r *Renderer) drawShadow(enc *gpu.CommandEncoder, depthKey pipeKey) bool {
	p := &r.pbr
	s := &p.shadow
	simple := r.pipeline(withKind(depthKey, pipeShadowSimple))
	full := r.pipeline(withKind(depthKey, pipeShadowFull))
	if simple == nil || full == nil {
		return false
	}
	if s.group == nil && len(p.full) > 0 {
		var err error
		s.group, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d shadow skins", Layout: full.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{
			{Binding: 0, Buffer: p.joints}, {Binding: 1, Buffer: p.weights}}})
		if err != nil {
			slog.Error("viewport3d: the shadow group could not be made", "err", err)
			return false
		}
	}
	p.colors[0] = gpu.RenderPassColorAttachment{View: s.color.View(), ClearValue: gpu.Color{R: 1, G: 1, B: 1, A: 1}}
	p.ds = gpu.RenderPassDepthStencilAttachment{View: s.depth.View(), DepthClearValue: 1}
	pass := enc.BeginRenderPass(gpu.RenderPassDesc{Label: "viewport3d shadow", ColorAttachments: p.colors[:], DepthStencilAttachment: &p.ds})
	pass.SetPipeline(simple)
	for i := range p.simple {
		c := &p.simple[i]
		pass.SetVertexBuffer(0, c.mesh.pos, 0)
		pass.SetVertexBuffer(1, r.instances, int(c.instance)*instanceStride)
		r.drawMeshOnce(pass, c.mesh)
	}
	if len(p.full) > 0 {
		pass.SetPipeline(full)
		pass.SetBindGroup(0, s.group)
		for i := range p.full {
			c := &p.full[i]
			grp := r.morphGroup(c.mesh, full, true)
			if grp == nil {
				continue
			}
			pass.SetBindGroup(1, grp)
			pass.SetVertexBuffer(0, c.mesh.full, 0)
			pass.SetVertexBuffer(1, r.instances, int(c.instance)*instanceStride)
			r.drawMeshOnce(pass, c.mesh)
		}
	}
	pass.End()
	return true
}

// withKind is the key with another kind of pipeline and the formats of the shadow map.
func withKind(k pipeKey, kind pipeKind) pipeKey {
	k.kind, k.samples = kind, 1
	k.color, k.depth = gpu.TextureFormatR32Float, gpu.TextureFormatDepth32Float
	k.variant, k.blend, k.noCull, k.noDepthTest = 0, false, false, false
	return k
}

// drawMeshOnce draws the mesh as one instance with the indices it has.
func (r *Renderer) drawMeshOnce(pass *gpu.RenderPass, g *meshGPU) {
	r.setIndices(pass, g)
	if g.index != nil {
		pass.DrawIndexed(g.count, 1, 0, 0, 0)
	} else {
		pass.Draw(g.count, 1, 0, 0)
	}
}

// drawPBR draws the PBR items: the shadow map, then the items into the float target, then the tone mapping over the view's target
// (cleared to the background). It returns whether it did, so that the frame's own pass loads what these passes drew.
func (r *Renderer) drawPBR(f *gpu.ViewFrame, dc *drawCtx, rot *vecmath.Mat4, bg gpu.Color) bool {
	p := &r.pbr
	style := dc.style
	enc := f.Encoder
	pipe := r.pbrPipeline(dc)
	if pipe == nil {
		return false
	}
	tk := pipeKey{kind: pipeTonemap, samples: f.Color.SampleCount(), color: f.Color.Format()}
	tonePipe := r.pipeline(tk)
	if tonePipe == nil {
		return false
	}
	if !r.ensureEnvironment(enc, style.environment) || !r.ensureStorage() || !r.ensureShadowDummy() || !r.ensureHDR(f.Width, f.Height, f.Color.SampleCount(), tonePipe) {
		return false
	}
	if p.frame == nil {
		var err error
		if p.frame, err = r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d pbr frame", Size: 256, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}); err != nil {
			slog.Error("viewport3d: the PBR frame could not be made", "err", err)
			return false
		}
	}
	shadowMap := p.shadow.dummy
	shadowOn := dc.shadowOn && p.lightOn
	if shadowOn {
		if !r.ensureShadowTargets(style.shadow.size) {
			shadowOn = false
		} else if r.drawShadow(enc, tk) {
			shadowMap = p.shadow.color
		} else {
			shadowOn = false
		}
	}
	r.writePBRFrame(style, rot, dc.eye, shadowOn)
	group := r.pbrGroup(pipe, shadowMap)
	if group == nil {
		return false
	}
	tone := &style.tone
	putVec4(p.toneBytes[:], 0, tone.exposure, tone.curve, 0, 0)
	r.dev.Queue().WriteBuffer(p.hdr.toneBuf, 0, p.toneBytes[:])

	// The items, as linear radiance in the float target (cleared to nothing: the alpha says where something was drawn).
	hdr := &p.hdr
	p.colors[0] = gpu.RenderPassColorAttachment{View: hdr.color.View()}
	if hdr.resolve != nil {
		p.colors[0].ResolveTarget = hdr.resolve.View()
		p.colors[0].StoreOp = gpu.StoreOpDiscard
	}
	p.ds = gpu.RenderPassDepthStencilAttachment{View: f.Depth.View(), DepthClearValue: 0}
	pass := enc.BeginRenderPass(gpu.RenderPassDesc{Label: "viewport3d pbr", ColorAttachments: p.colors[:], DepthStencilAttachment: &p.ds})
	pass.SetBindGroup(0, r.frameGroup, 0)
	pass.SetBindGroup(2, group, 0)
	var curPipe *gpu.RenderPipeline
	var curMat *matGPU
	var curMesh *meshGPU
	for _, list := range [2][]int32{r.popaque, r.ptransp} {
		var lastKey pipeKey
		var lastKeyPipe *gpu.RenderPipeline
		keyKnown := false
		for i := 0; i < len(list); {
			d := &r.items[list[i]]
			count := 1
			for j := i + 1; j < len(list) && !d.pk.blend && r.items[list[j]].key == d.key; j++ {
				count++
			}
			if !keyKnown || d.pk != lastKey {
				lastKey, lastKeyPipe, keyKnown = d.pk, r.pipeline(d.pk), true
			}
			pl := lastKeyPipe
			grp := r.morphGroup(d.mesh, pipe, false)
			if pl == nil || grp == nil {
				i += count
				continue
			}
			if pl != curPipe {
				pass.SetPipeline(pl)
				curPipe, curMat, curMesh = pl, nil, nil
			}
			if curMat != d.mat {
				pass.SetBindGroup(1, d.mat.group, 0)
				curMat = d.mat
			}
			if curMesh != d.mesh {
				pass.SetBindGroup(3, grp)
				pass.SetVertexBuffer(0, d.mesh.full, 0)
				r.setIndices(pass, d.mesh)
				curMesh = d.mesh
			}
			pass.SetVertexBuffer(1, r.instances, int(d.instance)*instanceStride)
			if d.mesh.index != nil {
				pass.DrawIndexed(d.mesh.count, count, 0, 0, 0)
			} else {
				pass.Draw(d.mesh.count, count, 0, 0)
			}
			r.stats.Draws++
			r.stats.Batches++
			r.stats.Instances += count
			i += count
		}
	}
	pass.End()

	// The tone mapping: the background, then the float image over it.
	p.colors[0] = gpu.RenderPassColorAttachment{View: f.Color.View(), ClearValue: bg}
	tpass := enc.BeginRenderPass(gpu.RenderPassDesc{Label: "viewport3d tone mapping", ColorAttachments: p.colors[:]})
	tpass.SetPipeline(tonePipe)
	tpass.SetBindGroup(0, hdr.toneGroup, 0)
	tpass.Draw(3, 1, 0, 0)
	tpass.End()
	return true
}

// shadowSetup fits the light of the frame to the scene, when the style asks for a shadow.
func (r *Renderer) shadowSetup(style *resolvedStyle) bool {
	p := &r.pbr
	p.light, p.lightOn = r.fitShadow(style, style.shadow.size)
	return p.lightOn
}

// sortPBR sorts the PBR items: the opaque ones by batch and the transparent ones back to front.
func (r *Renderer) sortPBR() {
	byKey := func(a, b int32) int { return cmp.Compare(r.items[a].key, r.items[b].key) }
	if !slices.IsSortedFunc(r.popaque, byKey) {
		slices.SortFunc(r.popaque, byKey)
	}
	slices.SortFunc(r.ptransp, func(a, b int32) int { return cmp.Compare(r.items[b].dist, r.items[a].dist) })
}

// clearColor is the color a view is cleared to, premultiplied.
func clearColor(bg ui.Color) gpu.Color {
	return gpu.Color{R: float64(bg.R * bg.A), G: float64(bg.G * bg.A), B: float64(bg.B * bg.A), A: float64(bg.A)}
}
