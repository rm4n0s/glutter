//go:build windows

package d3d12

import (
	"runtime"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The part of the device that step 30.2 added (SPEC §27.1, §9.8): compute pipelines and passes, the copies
// between buffers and textures, the read of a buffer, indirect draws and timestamp queries.

// computePipeline is a compute pipeline state object with the root signature it was made for.
type computePipeline struct {
	d         *Device
	desc      explicit.ComputePipelineDesc
	root      *rootSig
	pso       win32.Object
	entry     explicit.EntryPoint
	destroyed bool
}

var _ explicit.ComputePipeline = (*computePipeline)(nil)

// Desc implements explicit.ComputePipeline.
func (p *computePipeline) Desc() explicit.ComputePipelineDesc { return p.desc }

// Destroy implements explicit.ComputePipeline: the pipeline state is released when the GPU is done with
// the frames that may use it. The root signature belongs to the device.
func (p *computePipeline) Destroy() {
	if p.destroyed {
		return
	}
	p.destroyed = true
	d := p.d
	delete(d.live, p)
	d.pipes.live--
	d.retire(func() { d.disown(p.pso) })
}

// NewComputePipeline implements explicit.Device: the root signature from the bindings of the entry point
// (kept by their shape), the bytecode through the shader cache, and the pipeline state object. The
// pipeline library does not keep compute pipelines. It fails with D3D12PipelineInvalid for a description
// that is not a compute pipeline, and with the errors of NewPipeline for the rest.
func (d *Device) NewComputePipeline(desc explicit.ComputePipelineDesc) (explicit.ComputePipeline, error) {
	d.limits.CheckComputePipeline(desc)
	if desc.Shader == nil || desc.Shader.HLSL == "" {
		return nil, errors.New(tagD3D12PipelineInvalid, "a pipeline needs a shader with HLSL")
	}
	entry, ok := desc.Shader.Entry(desc.Entry)
	if !ok || entry.Stage != explicit.ShaderStageCompute {
		return nil, errors.New(tagD3D12PipelineInvalid, "the shader lacks the compute entry point", "shader", desc.Shader.Name, "entry", desc.Entry)
	}
	plan, err := planRootCompute(entry, d.limits)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	defer func() { d.pipes.took += time.Since(start) }()
	rs, err := d.rootSignature(plan)
	if err != nil {
		return nil, err
	}
	code, err := d.bytecode(desc.Shader, desc.Entry, TargetCompute)
	if err != nil {
		return nil, err
	}
	pd := d3d12ComputePipelineStateDesc{RootSignature: rs.obj.Ptr(), CS: d3d12ShaderBytecode{Code: unsafe.Pointer(&code[0]), Length: uintptr(len(code))}}
	var out uintptr
	err = d.hr(d.dev, "ID3D12Device::CreateComputePipelineState", deviceCreateComputePipeline,
		uintptr(unsafe.Pointer(&pd)), uintptr(unsafe.Pointer(&iidPipelineState)), uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(code)
	if err != nil {
		return nil, err
	}
	d.pipes.created++
	p := &computePipeline{d: d, desc: desc, root: rs, pso: d.own(win32.ObjectFromPtr(out)), entry: entry}
	d.pipes.live++
	d.live[p] = func() { p.destroyed = true; d.pipes.live--; d.disown(p.pso) }
	return p, nil
}

// NewComputeBindGroup implements explicit.Device: like NewBindGroup, for the root signature of a compute
// pipeline.
func (d *Device) NewComputeBindGroup(p explicit.ComputePipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	cp, ok := p.(*computePipeline)
	if !ok || cp.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "compute pipeline"))
	}
	return d.makeBindGroup(cp.root, pipelineBindings(cp.desc.Shader, cp.desc.Entry), group, entries)
}

// commandSignature returns the command signature of ExecuteIndirect for a draw, an indexed draw or a
// dispatch, made at the first indirect call; the arguments are those of Direct3D 12 (the same words as
// Vulkan's). A failure to make it panics: the calls that need it return no error.
func (d *Device) commandSignature(kind int) win32.Object {
	if !d.sigs[kind].IsNil() {
		return d.sigs[kind]
	}
	types := [3]uint32{indirectArgumentDraw, indirectArgumentDrawIdx, indirectArgumentDispatch}
	strides := [3]uint32{16, 20, 12}
	arg := d3d12IndirectArgumentDesc{Type: types[kind]}
	sd := d3d12CommandSignatureDesc{ByteStride: strides[kind], NumArgumentDescs: 1, ArgumentDescs: unsafe.Pointer(&arg)}
	var out uintptr
	err := d.hr(d.dev, "ID3D12Device::CreateCommandSignature", deviceCreateCommandSignature,
		uintptr(unsafe.Pointer(&sd)), 0, uintptr(unsafe.Pointer(&iidCommandSignature)), uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(&arg)
	if err != nil {
		panic(err)
	}
	d.sigs[kind] = d.own(win32.ObjectFromPtr(out))
	return d.sigs[kind]
}

// The kinds of command signature.
const (
	sigDraw = iota
	sigDrawIndexed
	sigDispatch
)

// querySet is a query heap of timestamps.
type querySet struct {
	d         *Device
	heap      win32.Object
	count     int
	destroyed bool
}

var _ explicit.QuerySet = (*querySet)(nil)

// Count implements explicit.QuerySet.
func (q *querySet) Count() int { return q.count }

// Destroy implements explicit.QuerySet: the heap is released when the GPU is done with the frames that
// may use it.
func (q *querySet) Destroy() {
	if q.destroyed {
		return
	}
	q.destroyed = true
	d := q.d
	delete(d.live, q)
	d.retire(func() { d.disown(q.heap) })
}

// queryTimestamps asks the queue how fast its timestamps tick (GetTimestampFrequency): the device has
// timestamp queries where it answers, with the length of a tick in nanoseconds.
func (d *Device) queryTimestamps() {
	var freq uint64
	if hr := win32.FromResult(d.queue.Call(queueGetTimestampFreq, uintptr(unsafe.Pointer(&freq)))); hr.Failed() || freq == 0 {
		d.log.Debug("the timestamp frequency of the queue is not known; there are no timestamp queries", "hresult", hr.String())
		return
	}
	d.features.TimestampQuery = true
	d.features.TimestampPeriod = 1e9 / float64(freq)
}

// NewQuerySet implements explicit.Device. It fails with GPUQueryUnsupported when the queue has no
// timestamp frequency.
func (d *Device) NewQuerySet(count int) (explicit.QuerySet, error) {
	if !d.features.TimestampQuery {
		return nil, errors.New(tagGPUQueryUnsupported, "the device has no timestamp queries", "count", count)
	}
	desc := d3d12QueryHeapDesc{Type: queryHeapTypeTimestamp, Count: uint32(max(count, 1))}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateQueryHeap", deviceCreateQueryHeap,
		uintptr(unsafe.Pointer(&desc)), uintptr(unsafe.Pointer(&iidQueryHeap)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	q := &querySet{d: d, heap: d.own(win32.ObjectFromPtr(out)), count: count}
	d.live[q] = func() { q.destroyed = true; d.disown(q.heap) }
	return q, nil
}

// passTimestamps are the queries a pass writes its times to: the heap, and the index of the time at the
// start and at the end (negative for none).
type passTimestamps struct {
	heap        win32.Object
	first, last int
}

// timestampsOf is the timestamps of a pass description; the zero value writes none.
func timestampsOf(d *Device, ts *explicit.PassTimestamps) passTimestamps {
	if ts == nil {
		return passTimestamps{first: -1, last: -1}
	}
	q, ok := ts.Set.(*querySet)
	if !ok || q.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "query set"))
	}
	return passTimestamps{heap: q.heap, first: ts.Begin, last: ts.End}
}

// begin writes the time at the start of the pass.
func (t passTimestamps) begin(e *encoder) {
	if !t.heap.IsNil() && t.first >= 0 {
		e.list.Call(listEndQuery, t.heap.Ptr(), uintptr(queryTypeTimestamp), uintptr(t.first))
	}
}

// end writes the time at the end of the pass.
func (t passTimestamps) end(e *encoder) {
	if !t.heap.IsNil() && t.last >= 0 {
		e.list.Call(listEndQuery, t.heap.Ptr(), uintptr(queryTypeTimestamp), uintptr(t.last))
	}
}

// ResolveQuerySet implements explicit.Encoder: ResolveQueryData into the buffer, in the state COPY_DEST.
func (e *encoder) ResolveQuerySet(set explicit.QuerySet, first, count int, dst explicit.Buffer, dstOffset int) {
	d := e.d
	d.limits.CheckResolveQuerySet(set, first, count, dst, dstOffset)
	q, ok := set.(*querySet)
	b, ok2 := dst.(*buffer)
	if !ok || !ok2 || q.d != d || b.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "query set or buffer"))
	}
	if e.inPass() {
		panic(errors.New(tagD3D12PassNotEnded, "ResolveQuerySet was called before the pass ended"))
	}
	if count == 0 {
		return
	}
	d.opBegin()
	defer d.opEnd()
	if err := d.needBuffer(b, stateCopyDest); err != nil {
		panic(err)
	}
	d.barrier(e.list)
	e.list.Call(listResolveQueryData, q.heap.Ptr(), uintptr(queryTypeTimestamp), uintptr(first), uintptr(count), b.res.Ptr(), uintptr(dstOffset))
}

// computePass records the dispatches of one compute pass.
type computePass struct {
	e          *encoder
	pipe       *computePipeline
	timestamps passTimestamps
}

var _ explicit.ComputePass = (*computePass)(nil)

// BeginComputePass implements explicit.Encoder. The pass starts with no pipeline and no bind group, as the
// passes of the other devices do.
func (e *encoder) BeginComputePass(desc explicit.ComputePassDesc) explicit.ComputePass {
	if e.inPass() {
		panic(errors.New(tagD3D12PassNotEnded, "BeginComputePass was called before the previous pass ended"))
	}
	e.d.limits.CheckTimestamps(e.d.features, desc.Timestamps)
	timestamps := timestampsOf(e.d, desc.Timestamps)
	e.d.opBegin()
	e.croot, e.cpso, e.cbound = nil, win32.Object{}, [maxBindGroups]boundGroup{}
	p := &computePass{e: e, timestamps: timestamps}
	p.timestamps.begin(e)
	e.cpass = p
	return p
}

// SetPipeline implements explicit.ComputePass: the compute root signature when it is not the one the list
// has, and the pipeline state (which the list has one of for both kinds of pipeline, so that the next draw
// sets its own again).
func (p *computePass) SetPipeline(pl explicit.ComputePipeline) {
	cp, ok := pl.(*computePipeline)
	if !ok || cp.d != p.e.d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "compute pipeline"))
	}
	e := p.e
	p.pipe = cp
	if e.croot != cp.root {
		e.list.Call(listSetComputeRootSig, cp.root.obj.Ptr())
		e.croot = cp.root
		e.cbound = [maxBindGroups]boundGroup{}
	}
	if e.cpso != cp.pso {
		e.list.Call(listSetPipelineState, cp.pso.Ptr())
		e.cpso = cp.pso
		e.pso = win32.Object{}
	}
}

// SetBindGroup implements explicit.ComputePass.
func (p *computePass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	p.e.setBindGroup(true, p.e.croot, group, g, dynamicOffsets)
}

// Dispatch implements explicit.ComputePass. A UAV barrier follows, so that the next dispatch sees what this
// one wrote, as in WebGPU.
func (p *computePass) Dispatch(x, y, z int) {
	p.e.d.limits.CheckDispatch(x, y, z)
	p.requirePipeline("Dispatch")
	p.e.list.Call(listDispatch, uintptr(x), uintptr(y), uintptr(z))
	p.e.d.uavBarrier(p.e.list)
}

// DispatchIndirect implements explicit.ComputePass.
func (p *computePass) DispatchIndirect(b explicit.Buffer, offset int) {
	e := p.e
	d := e.d
	d.limits.CheckIndirect(b, offset, 12)
	ib, ok := b.(*buffer)
	if !ok || ib.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	p.requirePipeline("DispatchIndirect")
	if err := d.needBuffer(ib, stateIndirectArg); err != nil {
		panic(err)
	}
	d.barrier(e.list)
	e.list.Call(listExecuteIndirect, d.commandSignature(sigDispatch).Ptr(), 1, ib.res.Ptr(), uintptr(offset), 0, 0)
	d.uavBarrier(e.list)
}

// requirePipeline fails with D3D12PipelineInvalid when no pipeline was set.
func (p *computePass) requirePipeline(command string) {
	if p.pipe == nil {
		panic(errors.New(tagD3D12PipelineInvalid, "a dispatch needs SetPipeline first", "command", command))
	}
}

// End implements explicit.ComputePass.
func (p *computePass) End() {
	e := p.e
	p.timestamps.end(e)
	e.d.opEnd()
	e.cpass = nil
}

// CopyBuffer implements explicit.Encoder: CopyBufferRegion, the source in COPY_SOURCE and the destination
// in COPY_DEST.
func (e *encoder) CopyBuffer(src explicit.Buffer, srcOffset int, dst explicit.Buffer, dstOffset, size int) {
	d := e.d
	d.limits.CheckCopyBuffer(src, srcOffset, dst, dstOffset, size)
	s, ok1 := src.(*buffer)
	t, ok2 := dst.(*buffer)
	if !ok1 || !ok2 || s.d != d || t.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	if e.inPass() {
		panic(errors.New(tagD3D12PassNotEnded, "CopyBuffer was called before the pass ended"))
	}
	if size == 0 {
		return
	}
	d.opBegin()
	defer d.opEnd()
	if err := d.needBuffer(s, stateCopySource); err != nil {
		panic(err)
	}
	if err := d.needBuffer(t, stateCopyDest); err != nil {
		panic(err)
	}
	d.barrier(e.list)
	e.list.Call(listCopyBufferRegion, t.res.Ptr(), uintptr(dstOffset), s.res.Ptr(), uintptr(srcOffset), uintptr(size))
}

// ReadBuffer implements explicit.Encoder: the bytes are copied into a readback buffer at this point of the
// commands, and done runs at the end of Submit, after the GPU finished.
func (e *encoder) ReadBuffer(src explicit.Buffer, offset, size int, done func(data []byte, err error)) {
	d := e.d
	d.limits.CheckReadBuffer(src, offset, size)
	s, ok := src.(*buffer)
	if !ok || s.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	if e.inPass() {
		panic(errors.New(tagD3D12PassNotEnded, "ReadBuffer was called before the pass ended"))
	}
	n := max(size, 4)
	res, err := d.createBuffer(heapTypeReadback, n)
	if err != nil {
		done(nil, err)
		return
	}
	if size > 0 {
		d.opBegin()
		if err := d.needBuffer(s, stateCopySource); err != nil {
			d.opEnd()
			panic(err)
		}
		d.barrier(e.list)
		d.opEnd()
		e.list.Call(listCopyBufferRegion, res.Ptr(), 0, s.res.Ptr(), uintptr(offset), uintptr(size))
	}
	e.reads = append(e.reads, &pendingRead{res: res, layout: copyLayout{Size: n}, size: size, doneRaw: done})
}

// OnComplete implements explicit.Encoder: Submit waits for the GPU and runs fn.
func (e *encoder) OnComplete(fn func()) { e.completes = append(e.completes, fn) }

// SetIndexBuffer32 implements explicit.Pass.
func (p *pass) SetIndexBuffer32(b explicit.Buffer, offset int) {
	p.setIndexBuffer(b, offset, dxgiFormatR32U)
}

// indirectBuffer checks the buffer of an indirect draw and puts it in the state INDIRECT_ARGUMENT.
func (p *pass) indirectBuffer(b explicit.Buffer, offset, size int, command string) *buffer {
	p.e.d.limits.CheckIndirect(b, offset, size)
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	p.flush(command)
	p.use(ib, stateIndirectArg)
	return ib
}

// DrawIndirect implements explicit.Pass: ExecuteIndirect with the signature of a draw.
func (p *pass) DrawIndirect(b explicit.Buffer, offset int) {
	ib := p.indirectBuffer(b, offset, 16, "DrawIndirect")
	d := p.e.d
	p.e.list.Call(listExecuteIndirect, d.commandSignature(sigDraw).Ptr(), 1, ib.res.Ptr(), uintptr(offset), 0, 0)
}

// DrawIndexedIndirect implements explicit.Pass: ExecuteIndirect with the signature of an indexed draw.
func (p *pass) DrawIndexedIndirect(b explicit.Buffer, offset int) {
	ib := p.indirectBuffer(b, offset, 20, "DrawIndexedIndirect")
	d := p.e.d
	p.e.list.Call(listExecuteIndirect, d.commandSignature(sigDrawIndexed).Ptr(), 1, ib.res.Ptr(), uintptr(offset), 0, 0)
}

// The copies between a buffer and a texture. Direct3D 12 places the rows of a copy at a footprint whose
// offset is a multiple of 512 bytes and whose layers are as far apart as one row pitch times the height,
// while the explicit device lets the layout start at any multiple of 4 and its layers be RowsPerImage rows
// apart. A copy of one image at an offset that is a multiple of 512 goes straight; any other goes through
// a buffer of its own that holds each image at a multiple of 512 (scratch), copied from or to the buffer
// with CopyBufferRegion, row by row when the buffer is the destination, so that the bytes between the rows
// are left as they are.

// textureSub is the subresource of the region's level of a texture, for the layer z of the region (an array
// or a cube) or the one slice of a volume.
func (t *texture) textureSub(level, z int) int {
	if t.desc.Kind == explicit.TextureKind2DArray || t.desc.Kind == explicit.TextureKindCube {
		return (t.baseLayer+z)*t.mips() + t.base + level
	}
	return t.baseLayer*t.mips() + t.base + level
}

// copyShape is what a copy between a buffer and a texture is made of.
type copyShape struct {
	w, h, n   int // the box: width and height in pixels, and the number of layers or slices
	bpp       int
	rowBytes  int
	span      int // the bytes from the first byte of an image to the last of its last row
	stride    int // the distance between two images in the scratch buffer
	volume    bool
	layered   bool
	direct    bool
	scratchSz int
}

func shapeOf(t *texture, layout explicit.BufferLayout, region explicit.TextureRegion) copyShape {
	c := copyShape{w: region.Size[0], h: region.Size[1], n: region.Size[2], bpp: t.desc.Format.BytesPerPixel()}
	c.rowBytes = c.w * c.bpp
	c.span = layout.BytesPerRow*(c.h-1) + c.rowBytes
	c.volume = t.desc.Kind == explicit.TextureKind3D
	c.layered = t.desc.Kind == explicit.TextureKind2DArray || t.desc.Kind == explicit.TextureKindCube
	c.stride = alignUp(layout.BytesPerRow*c.h, textureDataPlacementAlignment)
	c.direct = c.n == 1 && layout.Offset%textureDataPlacementAlignment == 0
	c.scratchSz = c.stride*(c.n-1) + c.span
	return c
}

// needBox puts the subresources of the box of a copy in the state.
func (d *Device) needBox(t *texture, region explicit.TextureRegion, c copyShape, st resState) {
	if c.layered {
		for z := 0; z < c.n; z++ {
			if err := d.tracker.need(t.root, t.textureSub(region.Level, region.Origin[2]+z), st); err != nil {
				panic(err)
			}
		}
		return
	}
	if err := d.tracker.need(t.root, t.textureSub(region.Level, 0), st); err != nil {
		panic(err)
	}
}

// imageBox is the box of the image z of a copy: its position in the level, and the slice a volume's image
// is at.
func imageBox(region explicit.TextureRegion, c copyShape, z int) (box d3d12Box, dstZ int) {
	box = d3d12Box{Left: uint32(region.Origin[0]), Top: uint32(region.Origin[1]), Right: uint32(region.Origin[0] + c.w), Bottom: uint32(region.Origin[1] + c.h), Back: 1}
	if c.volume {
		box.Front, box.Back = uint32(region.Origin[2]+z), uint32(region.Origin[2]+z+1)
		dstZ = region.Origin[2] + z
	}
	return box, dstZ
}

// footprint is the placed footprint of one image of the copy in a buffer at the offset.
func footprint(t *texture, c copyShape, layout explicit.BufferLayout, offset int) d3d12TextureCopyLocation {
	return d3d12TextureCopyLocation{Type: textureCopyPlacedFootprint, Placed: d3d12PlacedFootprint{
		Offset: uint64(offset),
		Footprint: d3d12SubresourceFootprint{
			Format: dxgiCopyFormat(t.desc.Format), Width: uint32(c.w), Height: uint32(c.h), Depth: 1, RowPitch: uint32(layout.BytesPerRow),
		},
	}}
}

// CopyBufferToTexture implements explicit.Encoder.
func (e *encoder) CopyBufferToTexture(src explicit.Buffer, layout explicit.BufferLayout, dst explicit.Texture, region explicit.TextureRegion) {
	d := e.d
	d.limits.CheckBufferTexture(src, layout, dst.Desc(), region, true)
	sb, ok := src.(*buffer)
	if !ok || sb.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	t := d.own2texture(dst)
	if e.inPass() {
		panic(errors.New(tagD3D12PassNotEnded, "CopyBufferToTexture was called before the pass ended"))
	}
	c := shapeOf(t, layout, region)
	from, offset := sb, layout.Offset
	if !c.direct {
		sc, err := d.newGPUBuffer(alignUp(max(c.scratchSz, 4), 4), alignUp(max(c.scratchSz, 4), 4), 0)
		if err != nil {
			panic(err)
		}
		defer sc.Destroy()
		d.opBegin()
		if err := d.needBuffer(sb, stateCopySource); err != nil {
			d.opEnd()
			panic(err)
		}
		if err := d.needBuffer(sc, stateCopyDest); err != nil {
			d.opEnd()
			panic(err)
		}
		d.barrier(e.list)
		d.opEnd()
		for z := 0; z < c.n; z++ {
			e.list.Call(listCopyBufferRegion, sc.res.Ptr(), uintptr(z*c.stride), sb.res.Ptr(), uintptr(layout.Offset+z*layout.BytesPerRow*layout.RowsPerImage), uintptr(c.span))
		}
		from, offset = sc, 0
	}
	d.opBegin()
	defer d.opEnd()
	if err := d.needBuffer(from, stateCopySource); err != nil {
		panic(err)
	}
	d.needBox(t, region, c, stateCopyDest)
	d.barrier(e.list)
	for z := 0; z < c.n; z++ {
		box, dstZ := imageBox(region, c, z)
		_ = box
		sub := t.textureSub(region.Level, region.Origin[2]+z)
		if !c.layered {
			sub = t.textureSub(region.Level, 0)
		}
		dstLoc := d3d12TextureCopyIndex{Resource: t.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(sub)}
		srcLoc := footprint(t, c, layout, offset)
		srcLoc.Resource = from.res.Ptr()
		srcLoc.Placed.Offset = uint64(offset + z*c.stride)
		if c.direct {
			srcLoc.Placed.Offset = uint64(offset)
		}
		x, y := region.Origin[0], region.Origin[1]
		e.list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), uintptr(x), uintptr(y), uintptr(dstZ), uintptr(unsafe.Pointer(&srcLoc)), 0)
	}
}

// CopyTextureToBuffer implements explicit.Encoder.
func (e *encoder) CopyTextureToBuffer(src explicit.Texture, region explicit.TextureRegion, dst explicit.Buffer, layout explicit.BufferLayout) {
	d := e.d
	d.limits.CheckBufferTexture(dst, layout, src.Desc(), region, false)
	db, ok := dst.(*buffer)
	if !ok || db.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	t := d.own2texture(src)
	if e.inPass() {
		panic(errors.New(tagD3D12PassNotEnded, "CopyTextureToBuffer was called before the pass ended"))
	}
	c := shapeOf(t, layout, region)
	into, offset := db, layout.Offset
	var sc *buffer
	if !c.direct {
		var err error
		if sc, err = d.newGPUBuffer(alignUp(max(c.scratchSz, 4), 4), alignUp(max(c.scratchSz, 4), 4), 0); err != nil {
			panic(err)
		}
		defer sc.Destroy()
		into, offset = sc, 0
	}
	d.opBegin()
	if err := d.needBuffer(into, stateCopyDest); err != nil {
		d.opEnd()
		panic(err)
	}
	d.needBox(t, region, c, stateCopySource)
	d.barrier(e.list)
	d.opEnd()
	for z := 0; z < c.n; z++ {
		box, _ := imageBox(region, c, z)
		sub := t.textureSub(region.Level, 0)
		if c.layered {
			sub = t.textureSub(region.Level, region.Origin[2]+z)
		}
		srcLoc := d3d12TextureCopyIndex{Resource: t.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(sub)}
		dstLoc := footprint(t, c, layout, offset)
		dstLoc.Resource = into.res.Ptr()
		if !c.direct {
			dstLoc.Placed.Offset = uint64(z * c.stride)
		}
		e.list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), 0, 0, 0, uintptr(unsafe.Pointer(&srcLoc)), uintptr(unsafe.Pointer(&box)))
	}
	if c.direct {
		return
	}
	// Copy the rows of the scratch buffer into the destination, leaving the bytes between them alone.
	d.opBegin()
	defer d.opEnd()
	if err := d.needBuffer(sc, stateCopySource); err != nil {
		panic(err)
	}
	if err := d.needBuffer(db, stateCopyDest); err != nil {
		panic(err)
	}
	d.barrier(e.list)
	for z := 0; z < c.n; z++ {
		for y := 0; y < c.h; y++ {
			e.list.Call(listCopyBufferRegion, db.res.Ptr(), uintptr(layout.Offset+z*layout.BytesPerRow*layout.RowsPerImage+y*layout.BytesPerRow),
				sc.res.Ptr(), uintptr(z*c.stride+y*layout.BytesPerRow), uintptr(c.rowBytes))
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUQueryUnsupported is the tag of the error of NewQuerySet on a device without timestamp queries
	// (explicit.TagGPUQueryUnsupported).
	tagGPUQueryUnsupported = "GPUQueryUnsupported"
)
