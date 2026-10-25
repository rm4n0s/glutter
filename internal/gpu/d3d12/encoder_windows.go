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

// BeginPass implements explicit.Encoder. The target must be a texture of this device with
// TextureRenderTarget. The barrier that puts it in the state RENDER_TARGET is recorded first, then
// the clear of LoadClear, or DiscardResource for LoadDontCare (the contents are not needed, which
// tile-based GPUs use; SPEC §9.3), then the render target, the viewport and the scissor of the whole
// target, and the triangle list.
func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	d := e.d
	t := d.own2texture(target)
	if e.pass != nil {
		panic(errors.New(tagD3D12PassNotEnded, "BeginPass was called before the previous pass ended"))
	}
	if t.rtv < 0 {
		panic(errors.New(tagD3D12TextureInvalid, "the target of a pass needs TextureRenderTarget", "usage", int(t.desc.Usage)))
	}
	d.tracker.begin()
	if err := d.tracker.need(t.root, t.base, stateRenderTarget); err != nil {
		d.tracker.end()
		panic(err)
	}
	d.barrier(e.list)
	switch loadActionOf(load) {
	case loadActionClear:
		rgba := [4]float32{clear.R, clear.G, clear.B, clear.A}
		e.list.Call(listClearRenderTargetView, t.rtvHandle(), uintptr(unsafe.Pointer(&rgba[0])), 0, 0)
	case loadActionDiscard:
		region := d3d12DiscardRegion{FirstSubresource: uint32(t.base), NumSubresources: 1}
		e.list.Call(listDiscardResource, t.res.Ptr(), uintptr(unsafe.Pointer(&region)))
	}
	rtv := t.rtvHandle()
	e.list.Call(listOMSetRenderTargets, 1, uintptr(unsafe.Pointer(&rtv)), 0, 0)
	w, h := t.desc.Width, t.desc.Height
	vp := d3d12Viewport{Width: float32(w), Height: float32(h), MaxDepth: 1}
	e.list.Call(listRSSetViewports, 1, uintptr(unsafe.Pointer(&vp)))
	rc := d3d12Rect{Right: int32(w), Bottom: int32(h)}
	e.list.Call(listRSSetScissorRects, 1, uintptr(unsafe.Pointer(&rc)))
	e.list.Call(listIASetPrimitiveTopo, uintptr(topologyTriangleList))
	e.pass = &pass{e: e, target: t}
	return e.pass
}

// pass records the draws of one render pass.
type pass struct {
	e      *encoder
	target *texture
	pipe   *pipeline
	// vertex holds the vertex buffers that were set, by slot; they are given to the input assembler
	// at the next draw, with the strides of the pipeline that draws.
	vertex      [8]vertexBinding
	vertexDirty bool
}

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
		if err := d.tracker.need(t.root, t.stateLevel(), statePixelShaderResource); err != nil {
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
	r = r.Intersect(image.Rect(0, 0, p.target.desc.Width, p.target.desc.Height))
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

// End implements explicit.Pass: the target stays in the state RENDER_TARGET until the next
// operation that needs another.
func (p *pass) End() {
	p.e.d.tracker.end()
	p.e.pass = nil
}
