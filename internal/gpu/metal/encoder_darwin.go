//go:build darwin && !ios

package metal

import (
	"image"
	"image/color"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// The statuses of a command buffer (MTLCommandBufferStatus) and the codes of its errors
// (MTLCommandBufferError) that the device tells apart.
const (
	statusCompleted uint64 = 4
	statusError     uint64 = 5

	cbErrorTimeout       int64 = 2
	cbErrorPageFault     int64 = 3
	cbErrorAccessRevoked int64 = 4
	cbErrorOutOfMemory   int64 = 8
	cbErrorDeviceRemoved int64 = 11
)

// frameRec is a frame that was committed: its command buffer, and what is kept until the GPU has finished it.
type frameRec struct {
	cb         objc.ID
	serial     uint64
	chunks     []*stageChunk
	reads      []*pendingRead
	completes  []func()
	err        error
	finishedOK bool
}

// encoder records the commands of one frame into one command buffer (SPEC §29.5).
type encoder struct {
	d         *Device
	cb        objc.ID
	serial    uint64
	pass      *pass
	cpass     *computePass
	reads     []*pendingRead
	completes []func()
}

var _ explicit.Encoder = (*encoder)(nil)

func (e *encoder) inPass() bool { return e.pass != nil || e.cpass != nil }

// Begin implements explicit.Device: it makes the command buffer of the next frame, after waiting for the oldest
// frame when FramesInFlight of them are still on the GPU. It fails with MetalFrameOpen when the frame before
// was not submitted, and with the error of a lost device.
func (d *Device) Begin() (explicit.Encoder, error) {
	if d.open != nil {
		return nil, errors.New(TagMetalFrameOpen, "Begin was called before the frame in progress was submitted")
	}
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	d.poll()
	for len(d.inflight) >= explicit.FramesInFlight {
		d.waitOne()
	}
	var cb objc.ID
	objcx.WithPool(func() { cb = retain(call(d.queue, "commandBuffer")) })
	if cb == 0 {
		return nil, errors.New(TagMetalCallFailed, "Metal made no command buffer", "call", "commandBuffer")
	}
	d.serial++
	e := &encoder{d: d, cb: cb, serial: d.serial}
	d.open = e
	return e, nil
}

// poll finishes the frames that the GPU is done with.
func (d *Device) poll() {
	for len(d.inflight) > 0 {
		rec := d.inflight[0]
		if send[uint64](rec.cb, "status") < statusCompleted {
			return
		}
		d.inflight = d.inflight[1:]
		d.finish(rec)
	}
}

// Settle handles the frames that the GPU has finished and, when one is still running, waits for the oldest and
// handles it too: the functions of OnComplete of a frame run when it is handled, at the next Begin or Submit
// at the latest. A program that wants them now (a test, the closing of a window) calls Settle.
//
// Flutter: no counterpart
func (d *Device) Settle() {
	d.poll()
	if len(d.inflight) > 0 {
		d.waitOne()
	}
}

// waitOne waits for the oldest frame and finishes it.
func (d *Device) waitOne() {
	rec := d.inflight[0]
	d.inflight = d.inflight[1:]
	call(rec.cb, "waitUntilCompleted")
	d.finish(rec)
}

// waitAll waits for every frame that was committed.
func (d *Device) waitAll() {
	for len(d.inflight) > 0 {
		d.waitOne()
	}
}

// finish handles a frame that the GPU has finished: the error it ended with, the reads, the functions of
// OnComplete, and the memory it held.
func (d *Device) finish(rec *frameRec) {
	if send[uint64](rec.cb, "status") == statusError {
		rec.err = commandBufferError(rec.cb)
		d.log.Error("a Metal command buffer failed", "err", rec.err)
		if appErr, ok := errors.FromError(rec.err); ok && appErr.Tag == tagGPUDeviceLost {
			d.markLost(rec.err)
		}
	}
	for _, rd := range rec.reads {
		rd.deliver(rec.err)
	}
	for _, fn := range rec.completes {
		fn()
	}
	d.recycle(rec.chunks)
	release(rec.cb)
	rec.cb = 0
	d.completed = rec.serial
	rec.finishedOK = true
}

// commandBufferError makes the error of a command buffer that ended in the error status.
func commandBufferError(cb objc.ID) error {
	e := call(cb, "error")
	return errorForCode(send[int64](e, "code"), description(e))
}

// errorForCode makes the error of the code of a failed command buffer (MTLCommandBufferError): a lost device for
// the errors that remove it, out of memory, and the call that failed for the others.
func errorForCode(code int64, output string) error {
	switch code {
	case cbErrorOutOfMemory:
		return errors.New(tagGPUOutOfMemory, "the GPU is out of memory", "output", output)
	case cbErrorTimeout, cbErrorPageFault, cbErrorAccessRevoked, cbErrorDeviceRemoved:
		return errors.New(tagGPUDeviceLost, "the Metal device was removed or reset", "code", code, "output", output)
	}
	return errors.New(TagMetalCallFailed, "a Metal command buffer failed", "call", "commit", "code", code, "output", output)
}

// abandon drops the frame in progress when the device closes.
func (e *encoder) abandon() {
	if e.pass != nil {
		e.pass.End()
	}
	if e.cpass != nil {
		e.cpass.End()
	}
	release(e.cb)
	e.cb = 0
	e.d.open = nil
}

// flushWrites records the writes that wait as one blit pass: the staged bytes into their buffers and textures.
func (e *encoder) flushWrites() {
	d := e.d
	if len(d.pending) == 0 {
		return
	}
	for _, c := range d.staging {
		c.flush()
	}
	objcx.WithPool(func() {
		blit := call(e.cb, "blitCommandEncoder")
		for _, w := range d.pending {
			if w.isBuffer {
				call(blit, "copyFromBuffer:sourceOffset:toBuffer:destinationOffset:size:", w.chunk.obj, uint64(w.off), w.dst, uint64(w.dstOff), uint64(w.size))
			} else {
				call(blit, "copyFromBuffer:sourceOffset:sourceBytesPerRow:sourceBytesPerImage:sourceSize:toTexture:destinationSlice:destinationLevel:destinationOrigin:",
					w.chunk.obj, uint64(w.off), uint64(w.bytesPerRow), uint64(w.bytesPerImage), w.box, w.dst, uint64(w.slice), uint64(w.level), w.origin)
			}
			w.release()
		}
		call(blit, "endEncoding")
	})
	d.pending = d.pending[:0]
}

// blit runs fn with a blit encoder after the writes that wait.
func (e *encoder) blit(fn func(blit objc.ID)) {
	if e.inPass() {
		panic(errors.New(TagMetalUsage, "a copy or a read was recorded before the pass ended"))
	}
	e.flushWrites()
	objcx.WithPool(func() {
		blit := call(e.cb, "blitCommandEncoder")
		fn(blit)
		call(blit, "endEncoding")
	})
}

// OnComplete implements explicit.Encoder.
func (e *encoder) OnComplete(fn func()) { e.completes = append(e.completes, fn) }

// Submit implements explicit.Encoder: the writes that wait are recorded, the command buffer is committed, and
// the frames the GPU has finished are handled. A frame with reads waits for the GPU, so that the reads are
// delivered here (SPEC §9.8, as Direct3D 12's does).
func (e *encoder) Submit() error {
	d := e.d
	if d.open != e {
		panic(errors.New(TagMetalUsage, "Submit was called on a frame that is not the one in progress"))
	}
	if e.inPass() {
		panic(errors.New(TagMetalUsage, "Submit was called before the pass ended"))
	}
	e.flushWrites()
	rec := &frameRec{cb: e.cb, serial: e.serial, chunks: d.staging, reads: e.reads, completes: e.completes}
	d.staging = nil
	d.open = nil
	d.inflight = append(d.inflight, rec)
	d.stats.Frames++
	call(e.cb, "commit")
	if len(rec.reads) > 0 {
		for !rec.finishedOK {
			d.waitOne()
		}
	} else {
		d.poll()
	}
	if rec.finishedOK {
		return rec.err
	}
	return nil
}

// BeginPass implements explicit.Encoder: a pass with one color attachment (BeginRenderPass).
func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	return e.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{
		Target: target, Load: load, Clear: [4]float64{float64(clear.R), float64(clear.G), float64(clear.B), float64(clear.A)},
	}}})
}

// pass records the draws of one render pass.
type pass struct {
	e       *encoder
	enc     objc.ID
	pipe    *pipeline
	w, h    int
	idxBuf  objc.ID
	idxOff  int
	idxType uint64
}

var _ explicit.Pass = (*pass)(nil)

// BeginRenderPass implements explicit.Encoder: the attachments become an MTLRenderPassDescriptor with the load
// and store actions of the description; a color attachment with a resolve target is resolved when the pass
// ends (the store action says it).
func (e *encoder) BeginRenderPass(desc explicit.PassDesc) explicit.Pass {
	d := e.d
	if e.inPass() {
		panic(errors.New(TagMetalUsage, "BeginPass was called before the previous pass ended"))
	}
	d.limits.CheckPass(desc)
	e.flushWrites()
	p := &pass{e: e}
	objcx.WithPool(func() {
		rp := call(class("MTLRenderPassDescriptor"), "renderPassDescriptor")
		colors := call(rp, "colorAttachments")
		for i, c := range desc.Color {
			t := d.ownTexture(c.Target)
			if i == 0 {
				p.w, p.h = t.desc.Width, t.desc.Height
			}
			att := call(colors, "objectAtIndexedSubscript:", uint64(i))
			call(att, "setTexture:", t.obj)
			call(att, "setLoadAction:", loadAction(c.Load))
			resolve := c.Resolve != nil
			if resolve {
				call(att, "setResolveTexture:", d.ownTexture(c.Resolve).obj)
			}
			call(att, "setStoreAction:", storeAction(c.Store, resolve))
			call(att, "setClearColor:", mtlClearColor{c.Clear[0], c.Clear[1], c.Clear[2], c.Clear[3]})
		}
		if ds := desc.Depth; ds != nil {
			t := d.ownTexture(ds.Target)
			if len(desc.Color) == 0 {
				p.w, p.h = t.desc.Width, t.desc.Height
			}
			att := call(rp, "depthAttachment")
			call(att, "setTexture:", t.obj)
			call(att, "setLoadAction:", loadAction(ds.DepthLoad))
			call(att, "setStoreAction:", storeAction(ds.DepthStore, false))
			call(att, "setClearDepth:", ds.ClearDepth)
			if t.desc.Format.HasStencil() {
				st := call(rp, "stencilAttachment")
				call(st, "setTexture:", t.obj)
				call(st, "setLoadAction:", loadAction(ds.StencilLoad))
				call(st, "setStoreAction:", storeAction(ds.StencilStore, false))
				call(st, "setClearStencil:", ds.ClearStencil)
			}
		}
		p.enc = retain(call(e.cb, "renderCommandEncoderWithDescriptor:", rp))
	})
	if p.enc == 0 {
		panic(errors.New(TagMetalCallFailed, "Metal made no render command encoder", "call", "renderCommandEncoderWithDescriptor"))
	}
	e.pass = p
	return p
}

// End implements explicit.Pass.
func (p *pass) End() {
	if p.enc == 0 {
		return
	}
	call(p.enc, "endEncoding")
	release(p.enc)
	p.enc = 0
	p.e.pass = nil
}

// SetPipeline implements explicit.Pass: the pipeline state, the depth-stencil state, the cull mode, the
// winding and the depth bias.
func (p *pass) SetPipeline(pl explicit.Pipeline) {
	mp, ok := pl.(*pipeline)
	if !ok || mp.d != p.e.d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "pipeline"))
	}
	p.pipe = mp
	call(p.enc, "setRenderPipelineState:", mp.state)
	if mp.depth != 0 {
		call(p.enc, "setDepthStencilState:", mp.depth)
	}
	call(p.enc, "setCullMode:", mp.cull)
	call(p.enc, "setFrontFacingWinding:", mp.front)
	call(p.enc, "setDepthBias:slopeScale:clamp:", mp.bias, mp.slope, mp.clamp)
}

func (p *pass) requirePipeline(op string) {
	if p.pipe == nil {
		panic(errors.New(TagMetalUsage, op+" was called before SetPipeline"))
	}
}

// renderBinder sets resources in the argument tables of the stages of a render encoder.
type renderBinder struct{ enc objc.ID }

func (r renderBinder) setBuffer(s stages, buf objc.ID, offset, index int) {
	if s&stageVertex != 0 {
		call(r.enc, "setVertexBuffer:offset:atIndex:", buf, uint64(offset), uint64(index))
	}
	if s&stageFragment != 0 {
		call(r.enc, "setFragmentBuffer:offset:atIndex:", buf, uint64(offset), uint64(index))
	}
}

func (r renderBinder) setTexture(s stages, tex objc.ID, index int) {
	if s&stageVertex != 0 {
		call(r.enc, "setVertexTexture:atIndex:", tex, uint64(index))
	}
	if s&stageFragment != 0 {
		call(r.enc, "setFragmentTexture:atIndex:", tex, uint64(index))
	}
}

func (r renderBinder) setSampler(s stages, smp objc.ID, index int) {
	if s&stageVertex != 0 {
		call(r.enc, "setVertexSamplerState:atIndex:", smp, uint64(index))
	}
	if s&stageFragment != 0 {
		call(r.enc, "setFragmentSamplerState:atIndex:", smp, uint64(index))
	}
}

// SetBindGroup implements explicit.Pass.
func (p *pass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	p.requirePipeline("SetBindGroup")
	p.e.d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok || bg.d != p.e.d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "bind group"))
	}
	gt := p.pipe.table.group(group)
	if gt == nil {
		panic(errors.New(TagMetalBindGroupInvalid, "the pipeline has no such bind group", "group", group))
	}
	if len(dynamicOffsets) != len(gt.uniforms) {
		panic(errors.New(TagMetalBindGroupInvalid, "SetBindGroup needs one dynamic offset for each uniform buffer of the group",
			"group", group, "offsets", len(dynamicOffsets), "uniforms", len(gt.uniforms)))
	}
	bg.apply(renderBinder{p.enc}, gt, dynamicOffsets)
}

// SetVertexBuffer implements explicit.Pass.
func (p *pass) SetVertexBuffer(slot int, b explicit.Buffer, offset int) {
	p.e.d.limits.CheckSetVertexBuffer(slot)
	call(p.enc, "setVertexBuffer:offset:atIndex:", p.e.d.ownBuffer(b).obj, uint64(offset), vertexBufferIndex(slot))
}

// SetIndexBuffer implements explicit.Pass.
func (p *pass) SetIndexBuffer(b explicit.Buffer, offset int) {
	p.idxBuf, p.idxOff, p.idxType = p.e.d.ownBuffer(b).obj, offset, indexUInt16
}

// SetIndexBuffer32 implements explicit.Pass.
func (p *pass) SetIndexBuffer32(b explicit.Buffer, offset int) {
	p.idxBuf, p.idxOff, p.idxType = p.e.d.ownBuffer(b).obj, offset, indexUInt32
}

// SetScissor implements explicit.Pass: the rectangle is cut to the attachments, as Metal needs it inside.
func (p *pass) SetScissor(r image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, p.w, p.h))
	call(p.enc, "setScissorRect:", mtlScissorRect{uint64(r.Min.X), uint64(r.Min.Y), uint64(max(r.Dx(), 0)), uint64(max(r.Dy(), 0))})
}

// SetStencilReference implements explicit.Pass.
func (p *pass) SetStencilReference(ref uint32) { call(p.enc, "setStencilReferenceValue:", ref) }

// Draw implements explicit.Pass.
func (p *pass) Draw(vertices, instances, firstVertex, firstInstance int) {
	p.requirePipeline("Draw")
	call(p.enc, "drawPrimitives:vertexStart:vertexCount:instanceCount:baseInstance:",
		p.pipe.prim, uint64(firstVertex), uint64(vertices), uint64(instances), uint64(firstInstance))
}

// indexSize is the size of an index of the bound index buffer.
func (p *pass) indexSize() int {
	if p.idxType == indexUInt32 {
		return 4
	}
	return 2
}

// DrawIndexed implements explicit.Pass.
func (p *pass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	p.requirePipeline("DrawIndexed")
	if p.idxBuf == 0 {
		panic(errors.New(TagMetalUsage, "DrawIndexed was called before SetIndexBuffer"))
	}
	call(p.enc, "drawIndexedPrimitives:indexCount:indexType:indexBuffer:indexBufferOffset:instanceCount:baseVertex:baseInstance:",
		p.pipe.prim, uint64(indices), p.idxType, p.idxBuf, uint64(p.idxOff+firstIndex*p.indexSize()), uint64(instances), int64(baseVertex), uint64(firstInstance))
}

// DrawIndirect implements explicit.Pass: the arguments are WebGPU's, which are Metal's.
func (p *pass) DrawIndirect(b explicit.Buffer, offset int) {
	p.requirePipeline("DrawIndirect")
	mb := p.e.d.ownBuffer(b)
	p.e.d.limits.CheckIndirect(b, offset, 16)
	call(p.enc, "drawPrimitives:indirectBuffer:indirectBufferOffset:", p.pipe.prim, mb.obj, uint64(offset))
}

// DrawIndexedIndirect implements explicit.Pass.
func (p *pass) DrawIndexedIndirect(b explicit.Buffer, offset int) {
	p.requirePipeline("DrawIndexedIndirect")
	if p.idxBuf == 0 {
		panic(errors.New(TagMetalUsage, "DrawIndexedIndirect was called before SetIndexBuffer"))
	}
	mb := p.e.d.ownBuffer(b)
	p.e.d.limits.CheckIndirect(b, offset, 20)
	call(p.enc, "drawIndexedPrimitives:indexType:indexBuffer:indexBufferOffset:indirectBuffer:indirectBufferOffset:",
		p.pipe.prim, p.idxType, p.idxBuf, uint64(p.idxOff), mb.obj, uint64(offset))
}

// computePass records the dispatches of one compute pass.
type computePass struct {
	e    *encoder
	enc  objc.ID
	pipe *computePipeline
}

var _ explicit.ComputePass = (*computePass)(nil)

// BeginComputePass implements explicit.Encoder.
func (e *encoder) BeginComputePass(desc explicit.ComputePassDesc) explicit.ComputePass {
	if e.inPass() {
		panic(errors.New(TagMetalUsage, "BeginComputePass was called before the previous pass ended"))
	}
	e.d.limits.CheckTimestamps(e.d.feats, desc.Timestamps)
	e.flushWrites()
	cp := &computePass{e: e}
	objcx.WithPool(func() { cp.enc = retain(call(e.cb, "computeCommandEncoder")) })
	if cp.enc == 0 {
		panic(errors.New(TagMetalCallFailed, "Metal made no compute command encoder", "call", "computeCommandEncoder"))
	}
	e.cpass = cp
	return cp
}

// End implements explicit.ComputePass.
func (p *computePass) End() {
	if p.enc == 0 {
		return
	}
	call(p.enc, "endEncoding")
	release(p.enc)
	p.enc = 0
	p.e.cpass = nil
}

// SetPipeline implements explicit.ComputePass.
func (p *computePass) SetPipeline(pl explicit.ComputePipeline) {
	mp, ok := pl.(*computePipeline)
	if !ok || mp.d != p.e.d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "compute pipeline"))
	}
	p.pipe = mp
	call(p.enc, "setComputePipelineState:", mp.state)
}

// computeBinder sets resources in the argument tables of a compute encoder.
type computeBinder struct{ enc objc.ID }

func (c computeBinder) setBuffer(_ stages, buf objc.ID, offset, index int) {
	call(c.enc, "setBuffer:offset:atIndex:", buf, uint64(offset), uint64(index))
}

func (c computeBinder) setTexture(_ stages, tex objc.ID, index int) {
	call(c.enc, "setTexture:atIndex:", tex, uint64(index))
}

func (c computeBinder) setSampler(_ stages, smp objc.ID, index int) {
	call(c.enc, "setSamplerState:atIndex:", smp, uint64(index))
}

// SetBindGroup implements explicit.ComputePass.
func (p *computePass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	if p.pipe == nil {
		panic(errors.New(TagMetalUsage, "SetBindGroup was called before SetPipeline"))
	}
	p.e.d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok || bg.d != p.e.d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "bind group"))
	}
	gt := p.pipe.table.group(group)
	if gt == nil {
		panic(errors.New(TagMetalBindGroupInvalid, "the pipeline has no such bind group", "group", group))
	}
	if len(dynamicOffsets) != len(gt.uniforms) {
		panic(errors.New(TagMetalBindGroupInvalid, "SetBindGroup needs one dynamic offset for each uniform buffer of the group",
			"group", group, "offsets", len(dynamicOffsets), "uniforms", len(gt.uniforms)))
	}
	bg.apply(computeBinder{p.enc}, gt, dynamicOffsets)
}

// Dispatch implements explicit.ComputePass: x, y and z are workgroups; the size of one is the entry point's.
func (p *computePass) Dispatch(x, y, z int) {
	if p.pipe == nil {
		panic(errors.New(TagMetalUsage, "Dispatch was called before SetPipeline"))
	}
	p.e.d.limits.CheckDispatch(x, y, z)
	g := p.pipe.group
	call(p.enc, "dispatchThreadgroups:threadsPerThreadgroup:", mtlSize{uint64(x), uint64(y), uint64(z)}, mtlSize{uint64(g[0]), uint64(g[1]), uint64(g[2])})
}

// DispatchIndirect implements explicit.ComputePass.
func (p *computePass) DispatchIndirect(b explicit.Buffer, offset int) {
	if p.pipe == nil {
		panic(errors.New(TagMetalUsage, "DispatchIndirect was called before SetPipeline"))
	}
	mb := p.e.d.ownBuffer(b)
	p.e.d.limits.CheckIndirect(b, offset, 12)
	g := p.pipe.group
	call(p.enc, "dispatchThreadgroupsWithIndirectBuffer:indirectBufferOffset:threadsPerThreadgroup:", mb.obj, uint64(offset), mtlSize{uint64(g[0]), uint64(g[1]), uint64(g[2])})
}

// CopyTexture implements explicit.Encoder.
func (e *encoder) CopyTexture(src explicit.Texture, r image.Rectangle, dst explicit.Texture, at image.Point) {
	s, t := e.d.ownTexture(src), e.d.ownTexture(dst)
	if r.Empty() {
		return
	}
	dstRect := image.Rectangle{Min: at, Max: at.Add(r.Size())}
	if !r.In(image.Rect(0, 0, s.desc.Width, s.desc.Height)) || !dstRect.In(image.Rect(0, 0, t.desc.Width, t.desc.Height)) {
		panic(errors.New(TagMetalUsage, "the copy is outside a texture", "op", "CopyTexture", "src", r.String(), "dst", dstRect.String()))
	}
	e.blit(func(blit objc.ID) {
		call(blit, "copyFromTexture:sourceSlice:sourceLevel:sourceOrigin:sourceSize:toTexture:destinationSlice:destinationLevel:destinationOrigin:",
			s.obj, uint64(0), uint64(0), mtlOrigin{uint64(r.Min.X), uint64(r.Min.Y), 0}, mtlSize{uint64(r.Dx()), uint64(r.Dy()), 1},
			t.obj, uint64(0), uint64(0), mtlOrigin{uint64(at.X), uint64(at.Y), 0})
	})
}

// CopyBuffer implements explicit.Encoder.
func (e *encoder) CopyBuffer(src explicit.Buffer, srcOffset int, dst explicit.Buffer, dstOffset, size int) {
	e.d.limits.CheckCopyBuffer(src, srcOffset, dst, dstOffset, size)
	s, t := e.d.ownBuffer(src), e.d.ownBuffer(dst)
	if size == 0 {
		return
	}
	e.blit(func(blit objc.ID) {
		call(blit, "copyFromBuffer:sourceOffset:toBuffer:destinationOffset:size:", s.obj, uint64(srcOffset), t.obj, uint64(dstOffset), uint64(size))
	})
}

// imageBytes is the size of one image of a buffer layout: the rows of the image.
func imageBytes(layout explicit.BufferLayout, height int) int {
	rows := layout.RowsPerImage
	if rows == 0 {
		rows = height
	}
	return layout.BytesPerRow * rows
}

// CopyBufferToTexture implements explicit.Encoder: each layer of an array or a cube is a slice of its own.
func (e *encoder) CopyBufferToTexture(src explicit.Buffer, layout explicit.BufferLayout, dst explicit.Texture, region explicit.TextureRegion) {
	t := e.d.ownTexture(dst)
	e.d.limits.CheckBufferTexture(src, layout, t.desc, region, true)
	s := e.d.ownBuffer(src)
	w, h, depth := region.Size[0], region.Size[1], region.Size[2]
	if w <= 0 || h <= 0 || depth <= 0 {
		return
	}
	per := imageBytes(layout, h)
	layers := t.desc.Kind == explicit.TextureKind2DArray || t.desc.Kind == explicit.TextureKindCube
	e.blit(func(blit objc.ID) {
		if layers {
			for z := 0; z < depth; z++ {
				call(blit, "copyFromBuffer:sourceOffset:sourceBytesPerRow:sourceBytesPerImage:sourceSize:toTexture:destinationSlice:destinationLevel:destinationOrigin:",
					s.obj, uint64(layout.Offset+z*per), uint64(layout.BytesPerRow), uint64(per), mtlSize{uint64(w), uint64(h), 1},
					t.obj, uint64(region.Origin[2]+z), uint64(region.Level), mtlOrigin{uint64(region.Origin[0]), uint64(region.Origin[1]), 0})
			}
			return
		}
		call(blit, "copyFromBuffer:sourceOffset:sourceBytesPerRow:sourceBytesPerImage:sourceSize:toTexture:destinationSlice:destinationLevel:destinationOrigin:",
			s.obj, uint64(layout.Offset), uint64(layout.BytesPerRow), uint64(per), mtlSize{uint64(w), uint64(h), uint64(depth)},
			t.obj, uint64(0), uint64(region.Level), mtlOrigin{uint64(region.Origin[0]), uint64(region.Origin[1]), uint64(region.Origin[2])})
	})
}

// CopyTextureToBuffer implements explicit.Encoder.
func (e *encoder) CopyTextureToBuffer(src explicit.Texture, region explicit.TextureRegion, dst explicit.Buffer, layout explicit.BufferLayout) {
	t := e.d.ownTexture(src)
	e.d.limits.CheckBufferTexture(dst, layout, t.desc, region, false)
	b := e.d.ownBuffer(dst)
	w, h, depth := region.Size[0], region.Size[1], region.Size[2]
	if w <= 0 || h <= 0 || depth <= 0 {
		return
	}
	per := imageBytes(layout, h)
	layers := t.desc.Kind == explicit.TextureKind2DArray || t.desc.Kind == explicit.TextureKindCube
	e.blit(func(blit objc.ID) {
		if layers {
			for z := 0; z < depth; z++ {
				call(blit, "copyFromTexture:sourceSlice:sourceLevel:sourceOrigin:sourceSize:toBuffer:destinationOffset:destinationBytesPerRow:destinationBytesPerImage:",
					t.obj, uint64(region.Origin[2]+z), uint64(region.Level), mtlOrigin{uint64(region.Origin[0]), uint64(region.Origin[1]), 0}, mtlSize{uint64(w), uint64(h), 1},
					b.obj, uint64(layout.Offset+z*per), uint64(layout.BytesPerRow), uint64(per))
			}
			return
		}
		call(blit, "copyFromTexture:sourceSlice:sourceLevel:sourceOrigin:sourceSize:toBuffer:destinationOffset:destinationBytesPerRow:destinationBytesPerImage:",
			t.obj, uint64(0), uint64(region.Level), mtlOrigin{uint64(region.Origin[0]), uint64(region.Origin[1]), uint64(region.Origin[2])}, mtlSize{uint64(w), uint64(h), uint64(depth)},
			b.obj, uint64(layout.Offset), uint64(layout.BytesPerRow), uint64(per))
	})
}

// ResolveQuerySet implements explicit.Encoder: there are no query sets (NewQuerySet fails), so there is nothing
// to resolve.
func (e *encoder) ResolveQuerySet(explicit.QuerySet, int, int, explicit.Buffer, int) {
	panic(errors.New(TagMetalUsage, "the Metal backend has no query sets"))
}

// pendingRead is a read that waits for the frame to finish: a readback buffer with the rows in it.
type pendingRead struct {
	buf       objc.ID
	rowPitch  int
	rect      image.Rectangle
	format    explicit.TextureFormat
	done      func(*image.RGBA, error)
	doneBytes func(data []byte, rowBytes int, err error)
	// doneRaw, when set, is a ReadBuffer: the readback buffer holds size bytes and nothing else is used.
	doneRaw func(data []byte, err error)
	size    int
}

// readbackBuffer makes the buffer that a read copies into: shared on unified memory, managed otherwise, so
// that the CPU can read it once the GPU has finished.
func (d *Device) readbackBuffer(size int) (objc.ID, error) {
	storage := storageShared
	if !d.unif {
		storage = storageManaged
	}
	var buf objc.ID
	objcx.WithPool(func() {
		buf = call(d.dev, "newBufferWithLength:options:", uint64(max(size, 16)), resourceOptions(storage, false))
	})
	if buf == 0 {
		return 0, errors.New(tagGPUOutOfMemory, "Metal made no readback buffer", "size", size)
	}
	return buf, nil
}

// fail tells the read that it failed.
func (rd *pendingRead) fail(err error) {
	switch {
	case rd.doneRaw != nil:
		rd.doneRaw(nil, err)
	case rd.doneBytes != nil:
		rd.doneBytes(nil, 0, err)
	default:
		rd.done(nil, err)
	}
}

// deliver converts what the GPU copied into the readback buffer (BGRA to RGBA, R to a gray pixel as the other
// backends do; ReadTextureBytes packs the rows as they are), calls done and frees the buffer. err is the
// error of the frame, which every read of it gets.
func (rd *pendingRead) deliver(err error) {
	defer func() { release(rd.buf); rd.buf = 0 }()
	if err != nil {
		rd.fail(err)
		return
	}
	if rd.doneRaw != nil {
		var raw []byte
		if rd.size > 0 && rd.buf != 0 {
			raw = append([]byte(nil), dl.Bytes(send[uintptr](rd.buf, "contents"), rd.size)...)
		}
		rd.doneRaw(raw, nil)
		return
	}
	w, h := rd.rect.Dx(), rd.rect.Dy()
	bpp := rd.format.BytesPerPixel()
	if rd.buf == 0 {
		if rd.doneBytes != nil {
			rd.doneBytes(nil, w*bpp, nil)
			return
		}
		rd.done(image.NewRGBA(image.Rect(0, 0, w, h)), nil)
		return
	}
	raw := dl.Bytes(send[uintptr](rd.buf, "contents"), rd.rowPitch*h)
	if rd.doneBytes != nil {
		out := make([]byte, w*bpp*h)
		for y := 0; y < h; y++ {
			copy(out[y*w*bpp:(y+1)*w*bpp], raw[y*rd.rowPitch:])
		}
		rd.doneBytes(out, w*bpp, nil)
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := raw[y*rd.rowPitch:]
		for x := 0; x < w; x++ {
			var c color.RGBA
			switch rd.format {
			case explicit.TextureFormatBGRA8Unorm:
				c = color.RGBA{row[x*4+2], row[x*4+1], row[x*4], row[x*4+3]}
			case explicit.TextureFormatR8Unorm:
				c = color.RGBA{row[x], row[x], row[x], 255}
			default:
				c = color.RGBA{row[x*bpp], row[x*bpp+1], row[x*bpp+2], row[x*bpp+3]}
			}
			img.SetRGBA(x, y, c)
		}
	}
	rd.done(img, nil)
}

// ReadTexture implements explicit.Encoder.
func (e *encoder) ReadTexture(src explicit.Texture, r image.Rectangle, done func(*image.RGBA, error)) {
	s := e.d.ownTexture(src)
	if e.inPass() {
		panic(errors.New(TagMetalUsage, "ReadTexture was called before the pass ended"))
	}
	e.d.limits.CheckRead(s.desc, false)
	e.read(s, r, &pendingRead{rect: r, format: s.desc.Format, done: done})
}

// ReadTextureBytes implements explicit.Encoder.
func (e *encoder) ReadTextureBytes(src explicit.Texture, r image.Rectangle, done func(data []byte, rowBytes int, err error)) {
	s := e.d.ownTexture(src)
	if e.inPass() {
		panic(errors.New(TagMetalUsage, "ReadTextureBytes was called before the pass ended"))
	}
	e.d.limits.CheckRead(s.desc, true)
	e.read(s, r, &pendingRead{rect: r, format: s.desc.Format, doneBytes: done})
}

// read copies the rectangle of the texture into a readback buffer, for the read to finish at Submit.
func (e *encoder) read(s *texture, r image.Rectangle, rd *pendingRead) {
	if !r.In(image.Rect(0, 0, s.desc.Width, s.desc.Height)) {
		panic(errors.New(TagMetalUsage, "the read is outside the texture", "rect", r.String(), "width", s.desc.Width, "height", s.desc.Height))
	}
	e.reads = append(e.reads, rd)
	if r.Empty() {
		return
	}
	rd.rowPitch = alignUp(r.Dx()*s.desc.Format.BytesPerPixel(), 16)
	buf, err := e.d.readbackBuffer(rd.rowPitch * r.Dy())
	if err != nil {
		e.d.markLost(err)
		return
	}
	rd.buf = buf
	e.blit(func(blit objc.ID) {
		call(blit, "copyFromTexture:sourceSlice:sourceLevel:sourceOrigin:sourceSize:toBuffer:destinationOffset:destinationBytesPerRow:destinationBytesPerImage:",
			s.obj, uint64(0), uint64(0), mtlOrigin{uint64(r.Min.X), uint64(r.Min.Y), 0}, mtlSize{uint64(r.Dx()), uint64(r.Dy()), 1},
			buf, uint64(0), uint64(rd.rowPitch), uint64(rd.rowPitch*r.Dy()))
		if !e.d.unif {
			call(blit, "synchronizeResource:", buf)
		}
	})
}

// ReadBuffer implements explicit.Encoder.
func (e *encoder) ReadBuffer(src explicit.Buffer, offset, size int, done func(data []byte, err error)) {
	e.d.limits.CheckReadBuffer(src, offset, size)
	b := e.d.ownBuffer(src)
	rd := &pendingRead{doneRaw: done, size: size}
	e.reads = append(e.reads, rd)
	if size == 0 {
		return
	}
	buf, err := e.d.readbackBuffer(size)
	if err != nil {
		e.d.markLost(err)
		return
	}
	rd.buf = buf
	e.blit(func(blit objc.ID) {
		call(blit, "copyFromBuffer:sourceOffset:toBuffer:destinationOffset:size:", b.obj, uint64(offset), buf, uint64(0), uint64(size))
		if !e.d.unif {
			call(blit, "synchronizeResource:", buf)
		}
	})
}
