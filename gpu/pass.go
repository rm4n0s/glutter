package gpu

import (
	"image"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// boundGroup is a bind group that a pass has set, with its dynamic offsets.
type boundGroup struct {
	bg      *BindGroup
	offsets []uint32
}

// checkBound panics in debug builds when a group that the pipeline uses is not set, or is set to a bind group of
// another layout.
func checkBound(call string, pipe pipelineRef, bound []boundGroup) {
	if !foundation.DebugMode {
		return
	}
	for i := 0; i < pipe.groupCount(); i++ {
		if bound[i].bg == nil {
			misuse(false, call, "every bind group the pipeline uses is set", "group", i)
			continue
		}
		if !bound[i].bg.layout.sameShape(pipe.layout(i)) {
			misuse(false, call, "a bind group has the layout of the group of the pipeline", "group", i)
		}
	}
}

// group returns the explicit bind group of bg for the pipeline, made once for the submission.
func (r *replay) group(bg *BindGroup, pipe pipelineRef, i int) (explicit.BindGroup, bool) {
	k := groupKey{bg, pipe}
	if g, ok := r.groups[k]; ok {
		return g, true
	}
	g, err := pipe.newGroup(r.d.dev, i, bg.entries)
	if err != nil {
		r.fail(err)
		return nil, false
	}
	r.groups[k] = g
	for _, b := range bg.buffers {
		r.markUsed(b.buf)
	}
	return g, true
}

// fail keeps the first error of a submission's commands.
func (r *replay) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

// passState is what a pass remembers while it records and while it runs.
type passState struct {
	pipe  pipelineRef
	bound []boundGroup
	// broken says that a bind group could not be made: the draws that follow do nothing.
	broken bool
}

// bindable is the part of an explicit pass that binds groups.
type bindable interface {
	SetBindGroup(i int, g explicit.BindGroup, offsets ...uint32)
}

// reset forgets what the pass remembers: the pipeline and the groups.
func (s *passState) reset() {
	s.pipe, s.broken = nil, false
	clear(s.bound)
}

// setBindGroup records the group and, with a pipeline, binds it on the explicit pass.
func (s *passState) setBindGroup(r *replay, i int, bg *BindGroup, offsets []uint32, pass bindable) {
	s.bound[i] = boundGroup{bg, offsets}
	if s.pipe == nil || i >= s.pipe.groupCount() || !bg.layout.sameShape(s.pipe.layout(i)) {
		return
	}
	bg.usableAll("SetBindGroup")
	g, ok := r.group(bg, s.pipe, i)
	if !ok {
		s.broken = true
		return
	}
	pass.SetBindGroup(i, g, offsets...)
}

// rebind binds the groups that were set before the pipeline was, when they fit it.
func (s *passState) rebind(r *replay, p pipelineRef, pass bindable) {
	for i := 0; i < p.groupCount(); i++ {
		if b := s.bound[i]; b.bg != nil && b.bg.layout.sameShape(p.layout(i)) {
			g, ok := r.group(b.bg, p, i)
			if !ok {
				s.broken = true
				return
			}
			pass.SetBindGroup(i, g, b.offsets...)
		}
	}
}

// opKind is what a recorded command of a pass is.
type opKind uint8

const (
	opSetPipeline opKind = iota
	opSetBindGroup
	opSetVertexBuffer
	opSetIndexBuffer
	opSetIndexBuffer32
	opSetScissor
	opSetStencilReference
	opDraw
	opDrawIndexed
	opDrawIndirect
	opDrawIndexedIndirect
	opDispatch
	opDispatchIndirect
)

// op is a command of a pass: a kind and its arguments, kept in a list that the pass keeps for the next frame (a closure for
// each command would be an allocation for each command, each frame).
type op struct {
	kind opKind
	rp   *RenderPipeline
	cp   *ComputePipeline
	bg   *BindGroup
	buf  *Buffer
	offs []uint32
	n    [5]int
}

// addOffsets keeps the dynamic offsets of a SetBindGroup in the pass's list and returns them.
func addOffsets(list *[]uint32, offsets []uint32) []uint32 {
	start := len(*list)
	*list = append(*list, offsets...)
	return (*list)[start:len(*list):len(*list)]
}

// RenderPass records the draws into the attachments of a render pass (WebGPU's GPURenderPassEncoder). End it before
// anything else is recorded in its encoder. A pass is kept by the queue for the next one once its encoder is submitted.
//
// Flutter: no counterpart
type RenderPass struct {
	e    *CommandEncoder
	d    *Device
	desc explicit.PassDesc
	// depthDesc is what desc.Depth points to.
	depthDesc explicit.DepthAttachment
	ops       []op
	offsets   []uint32
	// ended says that End was called.
	ended bool
	// The attachments, to check the pipelines and the scissor.
	colors  []TextureFormat
	depth   TextureFormat
	samples int
	width   int
	height  int
	// The state of the commands recorded so far.
	pipe    *RenderPipeline
	bound   []boundGroup
	vbufs   []vertexBinding
	ibuf    *Buffer
	ioffset int
	i32     bool
	// run is the state of the pass while its commands run.
	run renderRun
}

type vertexBinding struct {
	buf    *Buffer
	offset int
}

// renderRun is a render pass while its commands run.
type renderRun struct {
	r *replay
	p explicit.Pass
	passState
}

// reset empties the pass for the next frame; the lists keep their storage.
func (p *RenderPass) reset() {
	clear(p.ops)
	p.ops = p.ops[:0]
	p.offsets = p.offsets[:0]
	clear(p.desc.Color)
	p.desc = explicit.PassDesc{Color: p.desc.Color[:0]}
	p.depthDesc = explicit.DepthAttachment{}
	p.colors = p.colors[:0]
	p.depth, p.samples, p.width, p.height = 0, 0, 0, 0
	p.pipe, p.ibuf, p.ioffset, p.i32 = nil, nil, 0, false
	clear(p.bound)
	clear(p.vbufs)
	p.e, p.ended = nil, false
	p.run.reset()
	p.run.r, p.run.p = nil, nil
}

// takeRenderPass is a pass from the queue's list, or a new one.
func (e *CommandEncoder) takeRenderPass() *RenderPass {
	d, q := e.d, e.d.queue
	var p *RenderPass
	if n := len(q.renders); n > 0 {
		p = q.renders[n-1]
		q.renders[n-1] = nil
		q.renders = q.renders[:n-1]
	} else {
		p = &RenderPass{d: d, bound: make([]boundGroup, d.limits.MaxBindGroups), vbufs: make([]vertexBinding, d.limits.MaxVertexBuffers)}
		p.run.bound = make([]boundGroup, d.limits.MaxBindGroups)
	}
	p.e = e
	e.renders = append(e.renders, p)
	return p
}

// BeginRenderPass starts a pass that draws into the attachments (WebGPU's GPUCommandEncoder.beginRenderPass).
// There is at least one attachment, at most Limits.MaxColorAttachments colors and one depth-stencil. They are views of
// 2D textures (a LayerView for a layer of an array or a cube, a LevelView for a mip level) with
// TextureUsageRenderAttachment, of one size and one sample count. A multisampled color view has a resolve view of
// one sample, of its size and format.
//
// Flutter: no counterpart
func (e *CommandEncoder) BeginRenderPass(desc RenderPassDesc) *RenderPass {
	const call = "CommandEncoder.BeginRenderPass"
	e.open(call)
	d := e.d
	p := e.takeRenderPass()
	e.pass = true
	misuse(len(desc.ColorAttachments) > 0 || desc.DepthStencilAttachment != nil, call, "a pass has an attachment")
	if foundation.DebugMode && len(desc.ColorAttachments) > d.limits.MaxColorAttachments {
		misuse(false, call, "a pass has at most Limits.MaxColorAttachments color attachments", "value", len(desc.ColorAttachments))
	}
	bytes := 0
	for i := range desc.ColorAttachments {
		c := &desc.ColorAttachments[i]
		p.checkAttachment(call, c.View, "a color attachment")
		misuse(!c.View.desc.Format.IsDepthStencil(), call, "a color attachment has a color format")
		bytes += c.View.desc.Format.RenderTargetPixelBytes()
		pc := explicit.ColorAttachment{Target: c.View.exp, Load: explicit.LoadOp(c.LoadOp), Store: explicit.StoreOp(c.StoreOp),
			Clear: [4]float64{c.ClearValue.R, c.ClearValue.G, c.ClearValue.B, c.ClearValue.A}}
		if r := c.ResolveTarget; r != nil {
			misuse(r.tex.d == d, call, "the resolve target is a view of a texture of this device")
			r.tex.usable(call)
			misuse(c.View.desc.SampleCount > 1, call, "only a multisampled attachment is resolved")
			misuse(r.desc.SampleCount == 1 && r.desc.Format == c.View.desc.Format, call, "a resolve target has one sample and the format of its attachment")
			misuse(r.desc.Width == c.View.desc.Width && r.desc.Height == c.View.desc.Height && r.desc.Kind == explicit.TextureKind2D, call,
				"a resolve target is a 2D view of the size of its attachment")
			misuse(r.tex.desc.Usage&explicit.TextureRenderTarget != 0, call, "a resolve target has TextureUsageRenderAttachment")
			pc.Resolve = r.exp
		}
		p.desc.Color = append(p.desc.Color, pc)
		p.colors = append(p.colors, TextureFormat(c.View.desc.Format))
	}
	if foundation.DebugMode && bytes > d.limits.MaxColorAttachmentBytesPerSample {
		misuse(false, call, "the attachments take at most Limits.MaxColorAttachmentBytesPerSample bytes per sample", "value", bytes)
	}
	if ds := desc.DepthStencilAttachment; ds != nil {
		p.checkAttachment(call, ds.View, "the depth-stencil attachment")
		misuse(ds.View.desc.Format.IsDepthStencil(), call, "the depth-stencil attachment has a depth format")
		p.depth = TextureFormat(ds.View.desc.Format)
		p.depthDesc = explicit.DepthAttachment{Target: ds.View.exp, DepthLoad: explicit.LoadOp(ds.DepthLoadOp), DepthStore: explicit.StoreOp(ds.DepthStoreOp),
			ClearDepth: ds.DepthClearValue, StencilLoad: explicit.LoadOp(ds.StencilLoadOp), StencilStore: explicit.StoreOp(ds.StencilStoreOp), ClearStencil: ds.StencilClearValue}
		p.desc.Depth = &p.depthDesc
	}
	p.desc.Timestamps = d.timestamps(call, desc.TimestampWrites)
	return p
}

// checkAttachment checks an attachment view and takes the size and the sample count of the pass from the first one.
func (p *RenderPass) checkAttachment(call string, v *TextureView, what string) {
	if foundation.DebugMode && (v == nil || v.tex.d != p.d) {
		misuse(false, call, what+" is a view of a texture of this device")
	}
	v.tex.usable(call)
	if foundation.DebugMode && (v.tex.desc.Usage&explicit.TextureRenderTarget == 0) {
		misuse(false, call, what+" has TextureUsageRenderAttachment")
	}
	if foundation.DebugMode && (v.desc.Kind != explicit.TextureKind2D) {
		misuse(false, call, what+" is a 2D view: use LayerView for a layer of an array or a cube", "kind", int(v.desc.Kind))
	}
	if p.width == 0 {
		p.width, p.height, p.samples = v.desc.Width, v.desc.Height, v.desc.SampleCount
		return
	}
	if foundation.DebugMode && (v.desc.Width != p.width || v.desc.Height != p.height) {
		misuse(false, call, "the attachments have one size", "width", v.desc.Width, "height", v.desc.Height, "want_width", p.width, "want_height", p.height)
	}
	if foundation.DebugMode && (v.desc.SampleCount != p.samples) {
		misuse(false, call, "the attachments have one sample count", "samples", v.desc.SampleCount, "want", p.samples)
	}
}

// SetPipeline chooses the pipeline of the next draws (WebGPU's GPURenderPassEncoder.setPipeline). Its targets are
// the attachments of the pass: the same formats in the same order, the same depth-stencil format and the same
// sample count. The bind groups that were set stay, when they fit the pipeline.
//
// Flutter: no counterpart
func (p *RenderPass) SetPipeline(pipe *RenderPipeline) {
	const call = "RenderPass.SetPipeline"
	p.live(call)
	misuse(pipe != nil && pipe.d == p.d, call, "the pipeline was made by this device")
	pipe.usable(call)
	if foundation.DebugMode && (len(pipe.targets) != len(p.colors)) {
		misuse(false, call, "the pipeline has a target for each color attachment of the pass", "targets", len(pipe.targets), "attachments", len(p.colors))
	}
	for i := 0; i < min(len(pipe.targets), len(p.colors)); i++ {
		if foundation.DebugMode && (pipe.targets[i].Format != p.colors[i]) {
			misuse(false, call, "a target of the pipeline has the format of its attachment", "index", i)
		}
	}
	if foundation.DebugMode && (pipe.depth != p.depth) {
		misuse(false, call, "the pipeline has the depth-stencil format of the pass", "pipeline", int(pipe.depth), "pass", int(p.depth))
	}
	if foundation.DebugMode && (pipe.samples != p.samples) {
		misuse(false, call, "the pipeline has the sample count of the pass", "pipeline", pipe.samples, "pass", p.samples)
	}
	p.pipe = pipe
	p.ops = append(p.ops, op{kind: opSetPipeline, rp: pipe})
}

// live checks that the pass can record.
func (p *RenderPass) live(call string) {
	p.d.assertThread(call)
	misuse(!p.ended, call, "the pass is not ended")
}

// SetBindGroup binds the group of index to the bind group (WebGPU's GPURenderPassEncoder.setBindGroup).
// dynamicOffsets has one offset for each uniform buffer of the group, in binding order, each a multiple of
// Limits.MinUniformBufferOffsetAlignment; the range of the binding, moved by the offset, is inside the buffer. The
// group was made for the layout of group index of the pipeline.
//
// Flutter: no counterpart
func (p *RenderPass) SetBindGroup(index int, bg *BindGroup, dynamicOffsets ...uint32) {
	const call = "RenderPass.SetBindGroup"
	p.live(call)
	misuse(bg != nil && bg.d == p.d, call, "the bind group was made by this device")
	bg.usableAll(call)
	if foundation.DebugMode && (index < 0 || index >= len(p.bound) || bg.layout.group != index) {
		misuse(false, call, "the index is the group the bind group was made for", "index", index)
	}
	bg.check(call, dynamicOffsets)
	offsets := addOffsets(&p.offsets, dynamicOffsets)
	p.bound[index] = boundGroup{bg, offsets}
	p.ops = append(p.ops, op{kind: opSetBindGroup, bg: bg, offs: offsets, n: [5]int{index}})
}

// SetVertexBuffer binds a vertex buffer to a slot, from offset (WebGPU's GPURenderPassEncoder.setVertexBuffer).
// The buffer has BufferUsageVertex and the offset is a multiple of 4.
//
// Flutter: no counterpart
func (p *RenderPass) SetVertexBuffer(slot int, b *Buffer, offset int) {
	const call = "RenderPass.SetVertexBuffer"
	p.live(call)
	if foundation.DebugMode && (slot < 0 || slot >= len(p.vbufs)) {
		misuse(false, call, "the slot is below Limits.MaxVertexBuffers", "slot", slot)
	}
	misuse(b.sameDevice(p.d), call, "the buffer was made by this device")
	b.usable(call)
	misuse(b.usage&BufferUsageVertex != 0, call, "the buffer has BufferUsageVertex")
	if foundation.DebugMode && (offset < 0 || offset%4 != 0 || offset > b.size) {
		misuse(false, call, "the offset is a multiple of 4 inside the buffer", "offset", offset, "buffer", b.size)
	}
	p.vbufs[slot] = vertexBinding{b, offset}
	p.ops = append(p.ops, op{kind: opSetVertexBuffer, buf: b, n: [5]int{slot, offset}})
}

// SetIndexBuffer binds a buffer of 16 bit indices, from offset (WebGPU's GPURenderPassEncoder.setIndexBuffer with
// the format uint16). The buffer has BufferUsageIndex and the offset is a multiple of 2.
//
// Flutter: no counterpart
func (p *RenderPass) SetIndexBuffer(b *Buffer, offset int) {
	p.setIndexBuffer("RenderPass.SetIndexBuffer", b, offset, false)
}

// SetIndexBuffer32 binds a buffer of 32 bit indices, from offset (WebGPU's GPURenderPassEncoder.setIndexBuffer
// with the format uint32). The buffer has BufferUsageIndex and the offset is a multiple of 4.
//
// Flutter: no counterpart
func (p *RenderPass) SetIndexBuffer32(b *Buffer, offset int) {
	p.setIndexBuffer("RenderPass.SetIndexBuffer32", b, offset, true)
}

func (p *RenderPass) setIndexBuffer(call string, b *Buffer, offset int, wide bool) {
	p.live(call)
	misuse(b.sameDevice(p.d), call, "the buffer was made by this device")
	b.usable(call)
	misuse(b.usage&BufferUsageIndex != 0, call, "the buffer has BufferUsageIndex")
	size := 2
	if wide {
		size = 4
	}
	if foundation.DebugMode && (offset < 0 || offset%size != 0 || offset > b.size) {
		misuse(false, call, "the offset is a multiple of the index size inside the buffer", "offset", offset, "buffer", b.size)
	}
	p.ibuf, p.ioffset, p.i32 = b, offset, wide
	kind := opSetIndexBuffer
	if wide {
		kind = opSetIndexBuffer32
	}
	p.ops = append(p.ops, op{kind: kind, buf: b, n: [5]int{offset}})
}

// SetScissorRect limits the draws to a rectangle of the attachments, in pixels (WebGPU's
// GPURenderPassEncoder.setScissorRect). The rectangle is inside the attachments.
//
// Flutter: no counterpart
func (p *RenderPass) SetScissorRect(x, y, width, height int) {
	const call = "RenderPass.SetScissorRect"
	p.live(call)
	if foundation.DebugMode && (x < 0 || y < 0 || width < 0 || height < 0 || x+width > p.width || y+height > p.height) {
		misuse(false, call, "the rectangle is inside the attachments",
			"x", x, "y", y, "width", width, "height", height, "attachments_width", p.width, "attachments_height", p.height)
	}
	p.ops = append(p.ops, op{kind: opSetScissor, n: [5]int{x, y, x + width, y + height}})
}

// SetStencilReference sets the reference value of the stencil tests (WebGPU's GPURenderPassEncoder.setStencilReference).
// It is 0 at the start of a pass.
//
// Flutter: no counterpart
func (p *RenderPass) SetStencilReference(ref uint32) {
	p.live("RenderPass.SetStencilReference")
	p.ops = append(p.ops, op{kind: opSetStencilReference, n: [5]int{int(ref)}})
}

// checkDraw panics in debug builds when a draw cannot run: no pipeline, a bind group missing or of another layout, a
// vertex buffer missing or too small for the vertices (or instances) drawn.
func (p *RenderPass) checkDraw(call string, vertices, firstVertex, instances, firstInstance int, indexed bool) {
	p.live(call)
	misuse(p.pipe != nil, call, "a pipeline is set before a draw")
	if p.pipe == nil {
		return
	}
	checkBound(call, p.pipe, p.bound)
	if foundation.DebugMode && (vertices < 0 || instances < 0 || firstVertex < 0 || firstInstance < 0) {
		misuse(false, call, "a draw has no negative count", "vertices", vertices, "instances", instances)
	}
	if foundation.DebugMode && (p.pipe.vs.ReadsInstanceIndex && firstInstance != 0) {
		misuse(false, call, "a shader that reads instance_index is drawn with a first instance of 0 only", "first_instance", firstInstance)
	}
	if foundation.DebugMode && (p.pipe.vs.ReadsVertexIndex && firstVertex != 0) {
		misuse(false, call, "a shader that reads vertex_index is drawn with a first vertex of 0 only", "first_vertex", firstVertex)
	}
	if !foundation.DebugMode {
		return
	}
	for slot, l := range p.pipe.buffers {
		vb := p.vbufs[slot]
		if vb.buf == nil {
			misuse(false, call, "a vertex buffer is set for each slot of the pipeline", "slot", slot)
			continue
		}
		vb.buf.usable(call)
		n := firstVertex + vertices
		if l.PerInstance {
			n = firstInstance + instances
		} else if indexed {
			continue // the indices decide which vertices are read
		}
		end := 0
		for _, a := range l.Attributes {
			end = max(end, a.Offset+explicit.VertexFormat(a.Format).Size())
		}
		if n > 0 && vertices > 0 && instances > 0 && (vb.offset+(n-1)*l.ArrayStride+end > vb.buf.size) {
			misuse(false, call, "the vertices of a draw are inside the vertex buffer", "slot", slot,
				"need", vb.offset+(n-1)*l.ArrayStride+end, "buffer", vb.buf.size)
		}
	}
}

// Draw draws vertices, instances times (WebGPU's GPURenderPassEncoder.draw). A shader that reads vertex_index
// is drawn with firstVertex 0, one that reads instance_index with firstInstance 0.
//
// Flutter: no counterpart
func (p *RenderPass) Draw(vertices, instances, firstVertex, firstInstance int) {
	p.checkDraw("RenderPass.Draw", vertices, firstVertex, instances, firstInstance, false)
	p.ops = append(p.ops, op{kind: opDraw, n: [5]int{vertices, instances, firstVertex, firstInstance}})
}

// DrawIndexed draws indices, instances times (WebGPU's GPURenderPassEncoder.drawIndexed). The indices are
// inside the index buffer.
//
// Flutter: no counterpart
func (p *RenderPass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	const call = "RenderPass.DrawIndexed"
	p.checkDraw(call, indices, 0, instances, firstInstance, true)
	misuse(p.ibuf != nil, call, "an index buffer is set before an indexed draw")
	if foundation.DebugMode && (baseVertex != 0 && p.pipe != nil && p.pipe.vs.ReadsVertexIndex) {
		misuse(false, call, "a shader that reads vertex_index is drawn with a base vertex of 0 only", "base_vertex", baseVertex)
	}
	if p.ibuf != nil {
		size := 2
		if p.i32 {
			size = 4
		}
		if foundation.DebugMode && (firstIndex < 0 || p.ioffset+(firstIndex+indices)*size > p.ibuf.size) {
			misuse(false, call, "the indices of a draw are inside the index buffer",
				"first_index", firstIndex, "indices", indices, "buffer", p.ibuf.size)
		}
	}
	p.ops = append(p.ops, op{kind: opDrawIndexed, n: [5]int{indices, instances, firstIndex, baseVertex, firstInstance}})
}

// DrawIndirect draws with the arguments in a buffer: four 32 bit unsigned integers at offset, the vertex count, the
// instance count, the first vertex and the first instance (WebGPU's GPURenderPassEncoder.drawIndirect). The buffer
// has BufferUsageIndirect and the offset is a multiple of 4. The first vertex and the first instance are 0 for a
// shader that reads vertex_index or instance_index: the CPU cannot check them.
//
// Flutter: no counterpart
func (p *RenderPass) DrawIndirect(b *Buffer, offset int) {
	p.indirect("RenderPass.DrawIndirect", b, offset, 16, false)
}

// DrawIndexedIndirect draws with the arguments in a buffer: five 32 bit integers at offset, the index count, the
// instance count, the first index, the base vertex (signed) and the first instance (WebGPU's
// GPURenderPassEncoder.drawIndexedIndirect). The rules of DrawIndirect hold.
//
// Flutter: no counterpart
func (p *RenderPass) DrawIndexedIndirect(b *Buffer, offset int) {
	p.indirect("RenderPass.DrawIndexedIndirect", b, offset, 20, true)
}

func (p *RenderPass) indirect(call string, b *Buffer, offset, size int, indexed bool) {
	p.checkDraw(call, 0, 0, 0, 0, indexed)
	misuse(b.sameDevice(p.d), call, "the buffer was made by this device")
	b.usable(call)
	misuse(b.usage&BufferUsageIndirect != 0, call, "the buffer has BufferUsageIndirect")
	if foundation.DebugMode && (offset < 0 || offset%4 != 0 || offset+size > b.size) {
		misuse(false, call, "the arguments are a multiple of 4 inside the buffer", "offset", offset, "size", size, "buffer", b.size)
	}
	if indexed {
		misuse(p.ibuf != nil, call, "an index buffer is set before an indexed draw")
	}
	kind := opDrawIndirect
	if indexed {
		kind = opDrawIndexedIndirect
	}
	p.ops = append(p.ops, op{kind: kind, buf: b, n: [5]int{offset}})
}

// End finishes the pass (WebGPU's GPURenderPassEncoder.end). The encoder can record other commands then.
//
// Flutter: no counterpart
func (p *RenderPass) End() {
	p.live("RenderPass.End")
	p.ended = true
	p.e.pass = false
	p.e.cmds = append(p.e.cmds, command{rp: p})
}

// runs the commands of the pass on the explicit encoder of a submission.
func (p *RenderPass) runOn(r *replay) {
	pr := &p.run
	pr.r, pr.p = r, r.enc.BeginRenderPass(p.desc)
	pr.reset()
	for i := range p.ops {
		pr.exec(&p.ops[i])
	}
	pr.p.End()
}

// exec runs one command.
func (pr *renderRun) exec(o *op) {
	switch o.kind {
	case opSetPipeline:
		pipe := o.rp
		if pr.pipe == pipelineRef(pipe) {
			return
		}
		pr.pipe = pipe
		pr.p.SetPipeline(pipe.exp)
		pr.rebind(pr.r, pipe, pr.p)
	case opSetBindGroup:
		pr.setBindGroup(pr.r, o.n[0], o.bg, o.offs, pr.p)
	case opSetVertexBuffer:
		o.buf.usable("RenderPass.SetVertexBuffer")
		pr.r.markUsed(o.buf)
		pr.p.SetVertexBuffer(o.n[0], o.buf.exp, o.n[1])
	case opSetIndexBuffer, opSetIndexBuffer32:
		o.buf.usable("RenderPass.SetIndexBuffer")
		pr.r.markUsed(o.buf)
		if o.kind == opSetIndexBuffer32 {
			pr.p.SetIndexBuffer32(o.buf.exp, o.n[0])
		} else {
			pr.p.SetIndexBuffer(o.buf.exp, o.n[0])
		}
	case opSetScissor:
		pr.p.SetScissor(image.Rect(o.n[0], o.n[1], o.n[2], o.n[3]))
	case opSetStencilReference:
		pr.p.SetStencilReference(uint32(o.n[0]))
	case opDraw:
		if !pr.broken {
			pr.p.Draw(o.n[0], o.n[1], o.n[2], o.n[3])
		}
	case opDrawIndexed:
		if !pr.broken {
			pr.p.DrawIndexed(o.n[0], o.n[1], o.n[2], o.n[3], o.n[4])
		}
	case opDrawIndirect, opDrawIndexedIndirect:
		if pr.broken {
			return
		}
		o.buf.usable("RenderPass.DrawIndirect")
		pr.r.markUsed(o.buf)
		if o.kind == opDrawIndexedIndirect {
			pr.p.DrawIndexedIndirect(o.buf.exp, o.n[0])
		} else {
			pr.p.DrawIndirect(o.buf.exp, o.n[0])
		}
	}
}

// ComputePass records the dispatches of a compute pass (WebGPU's GPUComputePassEncoder). End it before anything
// else is recorded in its encoder. A pass is kept by the queue for the next one once its encoder is submitted.
//
// Flutter: no counterpart
type ComputePass struct {
	e       *CommandEncoder
	d       *Device
	desc    explicit.ComputePassDesc
	ops     []op
	offsets []uint32
	ended   bool
	pipe    *ComputePipeline
	bound   []boundGroup
	run     computeRun
}

// computeRun is a compute pass while its commands run.
type computeRun struct {
	r *replay
	p explicit.ComputePass
	passState
}

func (p *ComputePass) reset() {
	clear(p.ops)
	p.ops = p.ops[:0]
	p.offsets = p.offsets[:0]
	p.desc = explicit.ComputePassDesc{}
	p.pipe, p.e, p.ended = nil, nil, false
	clear(p.bound)
	p.run.reset()
	p.run.r, p.run.p = nil, nil
}

// BeginComputePass starts a compute pass (WebGPU's GPUCommandEncoder.beginComputePass). Compute passes need no
// attachments. The timestamps, when given, need Features.TimestampQuery.
//
// Flutter: no counterpart
func (e *CommandEncoder) BeginComputePass(desc ComputePassDesc) *ComputePass {
	const call = "CommandEncoder.BeginComputePass"
	e.open(call)
	e.pass = true
	d, q := e.d, e.d.queue
	var p *ComputePass
	if n := len(q.computes); n > 0 {
		p = q.computes[n-1]
		q.computes[n-1] = nil
		q.computes = q.computes[:n-1]
	} else {
		p = &ComputePass{d: d, bound: make([]boundGroup, d.limits.MaxBindGroups)}
		p.run.bound = make([]boundGroup, d.limits.MaxBindGroups)
	}
	p.e = e
	e.computes = append(e.computes, p)
	p.desc = explicit.ComputePassDesc{Timestamps: d.timestamps(call, desc.TimestampWrites)}
	return p
}

func (p *ComputePass) live(call string) {
	p.d.assertThread(call)
	misuse(!p.ended, call, "the pass is not ended")
}

// SetPipeline chooses the pipeline of the next dispatches (WebGPU's GPUComputePassEncoder.setPipeline). The bind
// groups that were set stay, when they fit the pipeline.
//
// Flutter: no counterpart
func (p *ComputePass) SetPipeline(pipe *ComputePipeline) {
	const call = "ComputePass.SetPipeline"
	p.live(call)
	misuse(pipe != nil && pipe.d == p.d, call, "the pipeline was made by this device")
	pipe.usable(call)
	p.pipe = pipe
	p.ops = append(p.ops, op{kind: opSetPipeline, cp: pipe})
}

// SetBindGroup binds the group of index to the bind group, with the rules of RenderPass.SetBindGroup (WebGPU's
// GPUComputePassEncoder.setBindGroup). A storage buffer has no dynamic offset: its range is the entry's.
//
// Flutter: no counterpart
func (p *ComputePass) SetBindGroup(index int, bg *BindGroup, dynamicOffsets ...uint32) {
	const call = "ComputePass.SetBindGroup"
	p.live(call)
	misuse(bg != nil && bg.d == p.d, call, "the bind group was made by this device")
	bg.usableAll(call)
	if foundation.DebugMode && (index < 0 || index >= len(p.bound) || bg.layout.group != index) {
		misuse(false, call, "the index is the group the bind group was made for", "index", index)
	}
	bg.check(call, dynamicOffsets)
	offsets := addOffsets(&p.offsets, dynamicOffsets)
	p.bound[index] = boundGroup{bg, offsets}
	p.ops = append(p.ops, op{kind: opSetBindGroup, bg: bg, offs: offsets, n: [5]int{index}})
}

func (p *ComputePass) checkDispatch(call string) {
	p.live(call)
	misuse(p.pipe != nil, call, "a pipeline is set before a dispatch")
	if p.pipe != nil {
		checkBound(call, p.pipe, p.bound)
	}
}

// DispatchWorkgroups runs x by y by z workgroups (WebGPU's GPUComputePassEncoder.dispatchWorkgroups). Each count
// is from 0 to Limits.MaxComputeWorkgroupsPerDimension.
//
// Flutter: no counterpart
func (p *ComputePass) DispatchWorkgroups(x, y, z int) {
	const call = "ComputePass.DispatchWorkgroups"
	p.checkDispatch(call)
	limit := p.d.limits.MaxComputeWorkgroupsPerDimension
	if foundation.DebugMode && (x < 0 || y < 0 || z < 0 || x > limit || y > limit || z > limit) {
		misuse(false, call, "the counts of a dispatch are up to Limits.MaxComputeWorkgroupsPerDimension", "x", x, "y", y, "z", z, "max", limit)
	}
	p.ops = append(p.ops, op{kind: opDispatch, n: [5]int{x, y, z}})
}

// DispatchWorkgroupsIndirect runs the workgroups whose counts are in a buffer: three 32 bit unsigned integers at
// offset (WebGPU's GPUComputePassEncoder.dispatchWorkgroupsIndirect). The buffer has BufferUsageIndirect and the offset
// is a multiple of 4.
//
// Flutter: no counterpart
func (p *ComputePass) DispatchWorkgroupsIndirect(b *Buffer, offset int) {
	const call = "ComputePass.DispatchWorkgroupsIndirect"
	p.checkDispatch(call)
	misuse(b.sameDevice(p.d), call, "the buffer was made by this device")
	b.usable(call)
	misuse(b.usage&BufferUsageIndirect != 0, call, "the buffer has BufferUsageIndirect")
	if foundation.DebugMode && (offset < 0 || offset%4 != 0 || offset+12 > b.size) {
		misuse(false, call, "the arguments are a multiple of 4 inside the buffer", "offset", offset, "buffer", b.size)
	}
	p.ops = append(p.ops, op{kind: opDispatchIndirect, buf: b, n: [5]int{offset}})
}

// End finishes the pass (WebGPU's GPUComputePassEncoder.end). The encoder can record other commands then.
//
// Flutter: no counterpart
func (p *ComputePass) End() {
	p.live("ComputePass.End")
	p.ended = true
	p.e.pass = false
	p.e.cmds = append(p.e.cmds, command{cp: p})
}

// runOn runs the commands of the pass on the explicit encoder of a submission.
func (p *ComputePass) runOn(r *replay) {
	pr := &p.run
	pr.r, pr.p = r, r.enc.BeginComputePass(p.desc)
	pr.reset()
	for i := range p.ops {
		pr.exec(&p.ops[i])
	}
	pr.p.End()
}

// exec runs one command.
func (pr *computeRun) exec(o *op) {
	switch o.kind {
	case opSetPipeline:
		pipe := o.cp
		if pr.pipe == pipelineRef(pipe) {
			return
		}
		pr.pipe = pipe
		pr.p.SetPipeline(pipe.exp)
		pr.rebind(pr.r, pipe, pr.p)
	case opSetBindGroup:
		pr.setBindGroup(pr.r, o.n[0], o.bg, o.offs, pr.p)
	case opDispatch:
		if !pr.broken {
			pr.p.Dispatch(o.n[0], o.n[1], o.n[2])
		}
	case opDispatchIndirect:
		if pr.broken {
			return
		}
		o.buf.usable("ComputePass.DispatchWorkgroupsIndirect")
		pr.r.markUsed(o.buf)
		pr.p.DispatchIndirect(o.buf.exp, o.n[0])
	}
}
