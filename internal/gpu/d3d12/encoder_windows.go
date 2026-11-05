//go:build windows

package d3d12

import (
	"image"
	"runtime"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// maxBindGroups is the number of bind groups of the design limits (SPEC §9.3), which the encoder
// keeps the state of.
const maxBindGroups = 4

// boundGroup is a bind group bound to the root signature of the command list, with its dynamic offsets.
type boundGroup struct {
	bg   *bindGroup
	offs []uint32
}

// BeginPass implements explicit.Encoder: a pass with one color attachment (BeginRenderPass).
func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	return e.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{
		Target: target, Load: load, Clear: [4]float64{float64(clear.R), float64(clear.G), float64(clear.B), float64(clear.A)},
	}}})
}

// BeginRenderPass implements explicit.Encoder. Every attachment must be a texture of this device with
// TextureRenderTarget. The barrier that puts them in the states RENDER_TARGET and DEPTH_WRITE is recorded
// first, then the clears of LoadClear (ClearRenderTargetView, ClearDepthStencilView), or DiscardResource
// for LoadDontCare of a color attachment (the contents are not needed, which tile-based GPUs use; SPEC
// §9.3), then the render targets, the viewport and the scissor of the whole attachments. A color
// attachment with a resolve target is resolved when the pass ends. The depth clear passes a float
// that the standard library's calls pass on amd64 only (depthclear_windows_arm64.go).
func (e *encoder) BeginRenderPass(desc explicit.PassDesc) explicit.Pass {
	d := e.d
	if e.pass != nil {
		panic(errors.New(tagD3D12PassNotEnded, "BeginPass was called before the previous pass ended"))
	}
	colors := make([]*texture, len(desc.Color))
	for i, c := range desc.Color {
		colors[i] = d.own2texture(c.Target)
		if c.Resolve != nil {
			d.own2texture(c.Resolve)
		}
	}
	var depth *texture
	if desc.Depth != nil {
		depth = d.own2texture(desc.Depth.Target)
	}
	for _, t := range colors {
		if t.rtv < 0 {
			panic(errors.New(tagD3D12TextureInvalid, "the target of a pass needs TextureRenderTarget", "usage", int(t.desc.Usage)))
		}
	}
	if depth != nil && depth.dsv < 0 {
		panic(errors.New(tagD3D12TextureInvalid, "the depth attachment of a pass needs TextureRenderTarget", "usage", int(depth.desc.Usage)))
	}
	if len(colors) == 0 && depth == nil {
		panic(errors.New(tagD3D12TextureInvalid, "a pass needs an attachment"))
	}
	d.limits.CheckPass(desc)
	d.tracker.begin()
	for _, t := range colors {
		if err := d.needTexture(t, stateRenderTarget); err != nil {
			d.tracker.end()
			panic(err)
		}
	}
	if depth != nil {
		if err := d.needTexture(depth, stateDepthWrite); err != nil {
			d.tracker.end()
			panic(err)
		}
	}
	d.barrier(e.list)
	p := &pass{e: e}
	rtvs := make([]uintptr, len(colors))
	for i, c := range desc.Color {
		t := colors[i]
		rtvs[i] = t.rtvHandle()
		switch loadActionOf(c.Load) {
		case loadActionClear:
			rgba := [4]float32{float32(c.Clear[0]), float32(c.Clear[1]), float32(c.Clear[2]), float32(c.Clear[3])}
			e.list.Call(listClearRenderTargetView, rtvs[i], uintptr(unsafe.Pointer(&rgba[0])), 0, 0)
		case loadActionDiscard:
			region := d3d12DiscardRegion{FirstSubresource: uint32(t.sub()), NumSubresources: 1}
			e.list.Call(listDiscardResource, t.res.Ptr(), uintptr(unsafe.Pointer(&region)))
		}
		if c.Resolve != nil {
			p.resolves = append(p.resolves, resolve{src: t, dst: d.own2texture(c.Resolve)})
		}
		p.width, p.height = t.desc.Width, t.desc.Height
	}
	var dsv uintptr
	if depth != nil {
		dsv = depth.dsvHandle()
		p.width, p.height = depth.desc.Width, depth.desc.Height
		var flags uint32
		if desc.Depth.DepthLoad == explicit.LoadClear {
			flags |= clearFlagDepth
		}
		if depth.desc.Format.HasStencil() && desc.Depth.StencilLoad == explicit.LoadClear {
			flags |= clearFlagStencil
		}
		if flags != 0 {
			clearDepthStencil(e.list, dsv, flags, float32(desc.Depth.ClearDepth), uint8(desc.Depth.ClearStencil))
		}
	}
	// A pass without a color attachment (depth only) names no render target view: the pointer is 0.
	if len(rtvs) == 0 {
		rtvs = make([]uintptr, 1)
	}
	if depth != nil {
		e.list.Call(listOMSetRenderTargets, uintptr(len(colors)), uintptr(unsafe.Pointer(&rtvs[0])), 0, uintptr(unsafe.Pointer(&dsv)))
	} else {
		e.list.Call(listOMSetRenderTargets, uintptr(len(colors)), uintptr(unsafe.Pointer(&rtvs[0])), 0, 0)
	}
	runtime.KeepAlive(rtvs)
	vp := d3d12Viewport{Width: float32(p.width), Height: float32(p.height), MaxDepth: 1}
	e.list.Call(listRSSetViewports, 1, uintptr(unsafe.Pointer(&vp)))
	rc := d3d12Rect{Right: int32(p.width), Bottom: int32(p.height)}
	e.list.Call(listRSSetScissorRects, 1, uintptr(unsafe.Pointer(&rc)))
	e.pass = p
	return p
}

// pass records the draws of one render pass.
type pass struct {
	e *encoder
	// width and height are the size of the attachments.
	width, height int
	// resolves are the multisampled attachments with the textures they are resolved into at End.
	resolves []resolve
	pipe     *pipeline
	// topology is the primitive topology that the input assembler was set to in this pass; 0 for none yet.
	topology uint32
	// vertex holds the vertex buffers that were set, by slot; they are given to the input assembler
	// at the next draw, with the strides of the pipeline that draws.
	vertex      [8]vertexBinding
	vertexDirty bool
}

// resolve is a multisampled attachment and the texture of one sample that receives its average.
type resolve struct{ src, dst *texture }

// vertexBinding is a vertex buffer that was set: its address from the offset, and the bytes from
// there to its end.
type vertexBinding struct {
	va   uint64
	size int
	set  bool
}

var _ explicit.Pass = (*pass)(nil)

// SetPipeline implements explicit.Pass: the root signature when it is not the one the list has (which
// loses the groups bound to the old one, as Direct3D 12 does), and the pipeline state.
func (p *pass) SetPipeline(pl explicit.Pipeline) {
	pp, ok := pl.(*pipeline)
	if !ok || pp.d != p.e.d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "pipeline"))
	}
	e := p.e
	p.pipe = pp
	if e.root != pp.root {
		e.list.Call(listSetRootSignature, pp.root.obj.Ptr())
		e.root = pp.root
		e.bound = [maxBindGroups]boundGroup{}
	}
	if e.pso != pp.pso {
		e.list.Call(listSetPipelineState, pp.pso.Ptr())
		e.pso = pp.pso
	}
	if p.topology != pp.topology {
		e.list.Call(listIASetPrimitiveTopo, uintptr(pp.topology))
		p.topology = pp.topology
	}
	p.vertexDirty = true
}

// SetBindGroup implements explicit.Pass. The group must fit the group of the pipeline (the same
// bindings), have been made for this frame, and come with one dynamic offset for each of its uniform
// buffers. The textures it samples are put in the state PIXEL_SHADER_RESOURCE by one barrier before the
// draw; the offsets are added to the addresses of the buffers and given as the root constant buffer
// views; the tables are set from the group's block of the shader-visible heaps, switching the heaps
// when the group is in another block than the one set (SPEC §9.8 "Descriptors").
func (p *pass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	e := p.e
	d := e.d
	d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok || bg.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "bind group"))
	}
	if p.pipe == nil {
		panic(errors.New(tagD3D12BindGroupInvalid, "SetBindGroup was called before SetPipeline", "group", group))
	}
	plan := p.pipe.root.plan
	if group < 0 || group >= len(plan.Groups) || group >= maxBindGroups {
		panic(errors.New(tagD3D12BindGroupInvalid, "the pipeline has no such bind group", "group", group, "groups", len(plan.Groups)))
	}
	gp := plan.Groups[group]
	switch {
	case bg.slot != e.slot || bg.gen != d.slots[bg.slot].gen:
		panic(errors.New(tagD3D12BindGroupInvalid, "the bind group was made for another frame", "group", group))
	case bg.group != group || bg.shape != gp.Shape:
		panic(errors.New(tagD3D12BindGroupInvalid, "the bind group does not fit this group of the pipeline",
			"group", group, "made_for", bg.group, "shape", bg.shape, "wanted", gp.Shape))
	case len(dynamicOffsets) != len(gp.Uniforms):
		panic(errors.New(tagD3D12BindGroupInvalid, "SetBindGroup needs one dynamic offset for each uniform buffer of the group",
			"group", group, "offsets", len(dynamicOffsets), "uniforms", len(gp.Uniforms)))
	}
	for _, t := range bg.textures {
		if t == nil {
			continue
		}
		if err := d.needTexture(t, statePixelShaderResource); err != nil {
			panic(err)
		}
	}
	d.barrier(e.list)
	e.bound[group] = boundGroup{bg: bg, offs: append([]uint32(nil), dynamicOffsets...)}
	e.bindTables(group)
	for i, param := range gp.UniformParams {
		e.list.Call(listSetRootCBV, uintptr(param), uintptr(bg.cbv[i]+uint64(dynamicOffsets[i])))
	}
}

// bindTables sets the descriptor tables of the group that was just bound. The heaps are the latest
// blocks of the slot; a group bound earlier that is in an older block is copied into the latest
// one again (rehomed), since only one heap of each kind can be set; when the heaps change, every table of
// the groups bound is set again.
func (e *encoder) bindTables(group int) {
	d := e.d
	s := &d.slots[e.slot]
	root := e.root.plan
	dirty := [maxBindGroups]bool{}
	home := func(i int) (v, smp int, has bool) {
		b := e.bound[i].bg
		if b == nil || !b.hasTables() {
			return 0, 0, false
		}
		return b.vBlock, b.sBlock, true
	}
	rehome := func(i int) error { return e.bound[i].bg.place() }
	moved, ok, err := settleBlocks(len(e.bound), home, func() (int, int) { return s.views.latest(), s.samplers.latest() }, rehome, maxBindGroups+1)
	if err != nil {
		panic(err) // out of memory: a method without an error return cannot say it
	}
	if !ok {
		panic(errors.New(tagD3D12BindGroupInvalid, "the descriptors of the bound groups do not fit one block of the shader-visible heaps"))
	}
	copy(dirty[:], moved)
	lv, ls := s.views.latest(), s.samplers.latest()
	if bg := e.bound[group].bg; bg.hasTables() {
		dirty[group] = true
	}
	if lv != e.viewHeap || ls != e.samplerHeap {
		var heaps [2]uintptr
		n := 0
		if lv >= 0 {
			heaps[n] = s.views.heaps[lv].obj.Ptr()
			n++
		}
		if ls >= 0 {
			heaps[n] = s.samplers.heaps[ls].obj.Ptr()
			n++
		}
		e.list.Call(listSetDescriptorHeaps, uintptr(n), uintptr(unsafe.Pointer(&heaps[0])))
		runtime.KeepAlive(heaps)
		e.viewHeap, e.samplerHeap = lv, ls
		for g := range e.bound {
			if b := e.bound[g].bg; b != nil && b.hasTables() {
				dirty[g] = true
			}
		}
	}
	for g := range dirty {
		if !dirty[g] || g >= len(root.Groups) {
			continue
		}
		b, gpg := e.bound[g].bg, root.Groups[g]
		if gpg.TextureParam >= 0 && b.vBlock >= 0 {
			e.list.Call(listSetRootTable, uintptr(gpg.TextureParam), uintptr(s.views.gpuHandle(b.vBlock, b.vOffset)))
		}
		if gpg.SamplerParam >= 0 && b.sBlock >= 0 {
			e.list.Call(listSetRootTable, uintptr(gpg.SamplerParam), uintptr(s.samplers.gpuHandle(b.sBlock, b.sOffset)))
		}
	}
}

// SetVertexBuffer implements explicit.Pass: the buffer is given to the input assembler at the next
// draw, with the stride of the pipeline's layout for the slot.
func (p *pass) SetVertexBuffer(slot int, b explicit.Buffer, offset int) {
	d := p.e.d
	d.limits.CheckSetVertexBuffer(slot)
	vb, ok := b.(*buffer)
	if !ok || vb.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	if slot < 0 || slot >= len(p.vertex) || offset < 0 || offset > vb.size {
		panic(errors.New(tagD3D12BindGroupInvalid, "the vertex buffer slot or offset is outside the limits", "slot", slot, "offset", offset, "size", vb.size))
	}
	p.vertex[slot] = vertexBinding{va: vb.va + uint64(offset), size: vb.size - offset, set: true}
	p.vertexDirty = true
}

// SetIndexBuffer implements explicit.Pass: 16 bit indices from the offset to the end of the buffer.
func (p *pass) SetIndexBuffer(b explicit.Buffer, offset int) {
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	view := d3d12IndexBufferView{BufferLocation: ib.va + uint64(offset), SizeInBytes: uint32(ib.size - offset), Format: dxgiFormatR16Uint}
	p.e.list.Call(listIASetIndexBuffer, uintptr(unsafe.Pointer(&view)))
}

// SetScissor implements explicit.Pass. The rectangle is clipped to the target.
func (p *pass) SetScissor(r image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, p.width, p.height))
	rc := d3d12Rect{Left: int32(r.Min.X), Top: int32(r.Min.Y), Right: int32(r.Max.X), Bottom: int32(r.Max.Y)}
	p.e.list.Call(listRSSetScissorRects, 1, uintptr(unsafe.Pointer(&rc)))
}

// flush gives the vertex buffers that were set to the input assembler, with the strides of the
// pipeline, and fails with D3D12PipelineInvalid when no pipeline was set.
func (p *pass) flush(command string) {
	if p.pipe == nil {
		panic(errors.New(tagD3D12PipelineInvalid, "a draw needs SetPipeline first", "command", command))
	}
	if !p.vertexDirty {
		return
	}
	p.vertexDirty = false
	for slot, vb := range p.pipe.desc.VertexBuffers {
		b := p.vertex[slot]
		if !b.set {
			continue
		}
		view := d3d12VertexBufferView{BufferLocation: b.va, SizeInBytes: uint32(b.size), StrideInBytes: uint32(vb.Stride)}
		p.e.list.Call(listIASetVertexBuffers, uintptr(slot), 1, uintptr(unsafe.Pointer(&view)))
	}
}

// Draw implements explicit.Pass: DrawInstanced.
func (p *pass) Draw(vertices, instances, firstVertex, firstInstance int) {
	p.flush("Draw")
	p.e.list.Call(listDrawInstanced, uintptr(vertices), uintptr(instances), uintptr(firstVertex), uintptr(firstInstance))
}

// DrawIndexed implements explicit.Pass: DrawIndexedInstanced.
func (p *pass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	p.flush("DrawIndexed")
	p.e.list.Call(listDrawIndexedInstanced, uintptr(indices), uintptr(instances), uintptr(firstIndex), uintptr(uint32(int32(baseVertex))), uintptr(firstInstance))
}

// SetStencilReference implements explicit.Pass: OMSetStencilRef.
func (p *pass) SetStencilReference(ref uint32) {
	p.e.list.Call(listOMSetStencilRef, uintptr(ref))
}

// End implements explicit.Pass: the targets stay in the state RENDER_TARGET until the next operation
// that needs another. The multisampled attachments are resolved: each is put in the state RESOLVE_SOURCE
// and its resolve target in RESOLVE_DEST by one barrier, and ResolveSubresource averages the samples.
func (p *pass) End() {
	e := p.e
	d := e.d
	d.tracker.end()
	e.pass = nil
	if len(p.resolves) == 0 {
		return
	}
	d.tracker.begin()
	defer d.tracker.end()
	for _, r := range p.resolves {
		if err := d.needTexture(r.src, stateResolveSource); err != nil {
			panic(err)
		}
		if err := d.needTexture(r.dst, stateResolveDest); err != nil {
			panic(err)
		}
	}
	d.barrier(e.list)
	for _, r := range p.resolves {
		e.list.Call(listResolveSubresource, r.dst.res.Ptr(), uintptr(r.dst.sub()), r.src.res.Ptr(), uintptr(r.src.sub()), uintptr(dxgiFormat(r.src.desc.Format)))
	}
}
