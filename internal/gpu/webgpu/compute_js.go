//go:build js && wasm

package webgpu

import (
	"syscall/js"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The part of the device that step 30.2 added (SPEC §27.1): compute pipelines and passes, the copies
// between buffers and textures, the read of a buffer, indirect draws and timestamp queries.

// computePipeline is a GPUComputePipeline with the layouts of its bind groups.
type computePipeline struct {
	d        *Device
	desc     explicit.ComputePipelineDesc
	pipe     js.Value
	layouts  []js.Value
	bindings []map[int]bindingPlan
}

var _ explicit.ComputePipeline = (*computePipeline)(nil)

// Desc implements explicit.ComputePipeline.
func (p *computePipeline) Desc() explicit.ComputePipelineDesc { return p.desc }

// Destroy implements explicit.ComputePipeline: pipelines have nothing to release before the garbage
// collector takes them.
func (p *computePipeline) Destroy() {}

// NewComputePipeline implements explicit.Device: the bind group layouts come from the bindings of the
// entry point.
func (d *Device) NewComputePipeline(desc explicit.ComputePipelineDesc) (explicit.ComputePipeline, error) {
	d.limits.CheckComputePipeline(desc)
	if err := d.usable(); err != nil {
		return nil, err
	}
	if desc.Shader == nil {
		return nil, errors.New(tagWebGPUPipelineInvalid, "a pipeline needs a shader")
	}
	entry, ok := desc.Shader.Entry(desc.Entry)
	if !ok || entry.Stage != explicit.ShaderStageCompute {
		return nil, errors.New(tagWebGPUPipelineInvalid, "the shader lacks the compute entry point", "shader", desc.Shader.Name, "entry", desc.Entry)
	}
	module, err := d.moduleOf(desc.Shader)
	if err != nil {
		return nil, err
	}
	plan := planStages(stagedEntry{entry, shaderCompute})
	p := &computePipeline{d: d, desc: desc, bindings: bindingsByNumber(plan)}
	_, err = tryCall("createComputePipeline", func() js.Value {
		var layouts []any
		for _, g := range plan {
			entries := make([]any, len(g))
			for i, b := range g {
				entries[i] = layoutEntry(b)
			}
			l := d.dev.Call("createBindGroupLayout", map[string]any{"entries": entries})
			p.layouts = append(p.layouts, l)
			layouts = append(layouts, l)
		}
		pl := d.dev.Call("createPipelineLayout", map[string]any{"bindGroupLayouts": layouts})
		p.pipe = d.dev.Call("createComputePipeline", map[string]any{
			"label": desc.Shader.Name, "layout": pl, "compute": map[string]any{"module": module, "entryPoint": desc.Entry},
		})
		return p.pipe
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// NewComputeBindGroup implements explicit.Device.
func (d *Device) NewComputeBindGroup(p explicit.ComputePipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	cp, ok := p.(*computePipeline)
	if !ok || cp.d != d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "compute pipeline"))
	}
	return d.makeBindGroup(cp.layouts, cp.bindings, group, entries)
}

// querySet is a GPUQuerySet of timestamps.
type querySet struct {
	d         *Device
	set       js.Value
	count     int
	destroyed bool
}

var _ explicit.QuerySet = (*querySet)(nil)

// Count implements explicit.QuerySet.
func (q *querySet) Count() int { return q.count }

// Destroy implements explicit.QuerySet: the set goes when the frame being recorded is submitted.
func (q *querySet) Destroy() {
	if q.destroyed {
		return
	}
	q.destroyed = true
	q.d.retire(func() { _, _ = tryCall("destroy", func() js.Value { return q.set.Call("destroy") }) })
}

// NewQuerySet implements explicit.Device. It fails with GPUQueryUnsupported when the adapter has no
// timestamp-query feature.
func (d *Device) NewQuerySet(count int) (explicit.QuerySet, error) {
	if !d.features.TimestampQuery {
		return nil, errors.New(tagGPUQueryUnsupported, "the device has no timestamp queries", "count", count)
	}
	if err := d.usable(); err != nil {
		return nil, err
	}
	v, err := tryCall("createQuerySet", func() js.Value {
		return d.dev.Call("createQuerySet", map[string]any{"type": "timestamp", "count": max(count, 1)})
	})
	if err != nil {
		return nil, err
	}
	return &querySet{d: d, set: v, count: count}, nil
}

// timestampWrites returns the timestampWrites of a pass descriptor, or nil for a pass without any.
func timestampWrites(ts *explicit.PassTimestamps) map[string]any {
	if ts == nil || (ts.Begin < 0 && ts.End < 0) {
		return nil
	}
	q, ok := ts.Set.(*querySet)
	if !ok {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "query set"))
	}
	w := map[string]any{"querySet": q.set}
	if ts.Begin >= 0 {
		w["beginningOfPassWriteIndex"] = ts.Begin
	}
	if ts.End >= 0 {
		w["endOfPassWriteIndex"] = ts.End
	}
	return w
}

// ResolveQuerySet implements explicit.Encoder.
func (e *encoder) ResolveQuerySet(set explicit.QuerySet, first, count int, dst explicit.Buffer, dstOffset int) {
	e.d.limits.CheckResolveQuerySet(set, first, count, dst, dstOffset)
	q, ok := set.(*querySet)
	b, ok2 := dst.(*buffer)
	if !ok || !ok2 || q.d != e.d || b.d != e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "query set or buffer"))
	}
	if count == 0 {
		return
	}
	e.enc.Call("resolveQuerySet", q.set, first, count, b.buf, dstOffset)
}

// computePass records the dispatches of one compute pass.
type computePass struct {
	e        *encoder
	p        js.Value
	pipeline *computePipeline
}

var _ explicit.ComputePass = (*computePass)(nil)

// BeginComputePass implements explicit.Encoder.
func (e *encoder) BeginComputePass(desc explicit.ComputePassDesc) explicit.ComputePass {
	if e.pass != nil || e.cpass != nil {
		panic(errors.New(tagWebGPUPassNotEnded, "BeginComputePass was called before the previous pass ended"))
	}
	e.d.limits.CheckTimestamps(e.d.features, desc.Timestamps)
	passDesc := map[string]any{}
	if w := timestampWrites(desc.Timestamps); w != nil {
		passDesc["timestampWrites"] = w
	}
	e.cpass = &computePass{e: e, p: e.enc.Call("beginComputePass", passDesc)}
	return e.cpass
}

// SetPipeline implements explicit.ComputePass.
func (p *computePass) SetPipeline(pl explicit.ComputePipeline) {
	cp, ok := pl.(*computePipeline)
	if !ok || cp.d != p.e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "compute pipeline"))
	}
	p.pipeline = cp
	p.p.Call("setPipeline", cp.pipe)
}

// SetBindGroup implements explicit.ComputePass. The dynamic offsets cross in the reused Uint32Array.
func (p *computePass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	d := p.e.d
	d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "bind group"))
	}
	if len(dynamicOffsets) == 0 {
		p.p.Call("setBindGroup", group, bg.g)
		return
	}
	for i, o := range dynamicOffsets {
		d.dynamic.SetIndex(i, o)
	}
	p.p.Call("setBindGroup", group, bg.g, d.dynamic, 0, len(dynamicOffsets))
}

// Dispatch implements explicit.ComputePass.
func (p *computePass) Dispatch(x, y, z int) {
	p.e.d.limits.CheckDispatch(x, y, z)
	p.p.Call("dispatchWorkgroups", x, y, z)
}

// DispatchIndirect implements explicit.ComputePass.
func (p *computePass) DispatchIndirect(b explicit.Buffer, offset int) {
	p.e.d.limits.CheckIndirect(b, offset, 12)
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	p.p.Call("dispatchWorkgroupsIndirect", ib.buf, offset)
}

// End implements explicit.ComputePass.
func (p *computePass) End() {
	p.p.Call("end")
	p.e.cpass = nil
}

// CopyBuffer implements explicit.Encoder.
func (e *encoder) CopyBuffer(src explicit.Buffer, srcOffset int, dst explicit.Buffer, dstOffset, size int) {
	e.d.limits.CheckCopyBuffer(src, srcOffset, dst, dstOffset, size)
	s, ok1 := src.(*buffer)
	t, ok2 := dst.(*buffer)
	if !ok1 || !ok2 || s.d != e.d || t.d != e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	if size == 0 {
		return
	}
	e.enc.Call("copyBufferToBuffer", s.buf, srcOffset, t.buf, dstOffset, size)
}

// bufferLayout is the GPUTexelCopyBufferLayout of a layout.
func bufferLayout(layout explicit.BufferLayout) map[string]any {
	return map[string]any{"offset": layout.Offset, "bytesPerRow": layout.BytesPerRow, "rowsPerImage": layout.RowsPerImage}
}

// textureCopy is the GPUTexelCopyTextureInfo of the region of a texture, and its size.
func textureCopy(t *texture, region explicit.TextureRegion) (map[string]any, []any) {
	return map[string]any{
			"texture": t.tex, "mipLevel": t.base + region.Level, "aspect": t.aspect(),
			"origin": []any{region.Origin[0], region.Origin[1], t.baseLayer + region.Origin[2]},
		},
		[]any{region.Size[0], region.Size[1], region.Size[2]}
}

// CopyBufferToTexture implements explicit.Encoder.
func (e *encoder) CopyBufferToTexture(src explicit.Buffer, layout explicit.BufferLayout, dst explicit.Texture, region explicit.TextureRegion) {
	e.d.limits.CheckBufferTexture(src, layout, dst.Desc(), region, true)
	b, ok1 := src.(*buffer)
	t, ok2 := dst.(*texture)
	if !ok1 || !ok2 || b.d != e.d || t.d != e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer or texture"))
	}
	info, size := textureCopy(t, region)
	srcInfo := bufferLayout(layout)
	srcInfo["buffer"] = b.buf
	e.enc.Call("copyBufferToTexture", srcInfo, info, size)
}

// CopyTextureToBuffer implements explicit.Encoder.
func (e *encoder) CopyTextureToBuffer(src explicit.Texture, region explicit.TextureRegion, dst explicit.Buffer, layout explicit.BufferLayout) {
	e.d.limits.CheckBufferTexture(dst, layout, src.Desc(), region, false)
	t, ok1 := src.(*texture)
	b, ok2 := dst.(*buffer)
	if !ok1 || !ok2 || b.d != e.d || t.d != e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer or texture"))
	}
	info, size := textureCopy(t, region)
	dstInfo := bufferLayout(layout)
	dstInfo["buffer"] = b.buf
	e.enc.Call("copyTextureToBuffer", info, dstInfo, size)
}

// ReadBuffer implements explicit.Encoder: the bytes are copied into a buffer that the CPU can map at
// this point of the commands, and done runs in a later task, once the GPU finished and the browser
// mapped the buffer, on the UI goroutine (SPEC §9.5, §23.3). Submit starts the mapping.
func (e *encoder) ReadBuffer(src explicit.Buffer, offset, size int, done func(data []byte, err error)) {
	d := e.d
	d.limits.CheckReadBuffer(src, offset, size)
	s, ok := src.(*buffer)
	if !ok || s.d != d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	buf, err := tryCall("createBuffer", func() js.Value {
		return d.dev.Call("createBuffer", map[string]any{"size": max(size, 4), "usage": bufferMapRead | bufferCopyDst})
	})
	if err != nil {
		done(nil, err)
		return
	}
	if size > 0 {
		e.enc.Call("copyBufferToBuffer", s.buf, offset, buf, 0, size)
	}
	e.reads = append(e.reads, &readJob{d: d, buffer: buf, size: size, doneRaw: done})
}

// OnComplete implements explicit.Encoder: Submit asks the queue to say when it has finished, and fn
// runs then, on the UI goroutine.
func (e *encoder) OnComplete(fn func()) { e.completes = append(e.completes, fn) }

// completion is an OnComplete function that waits for the queue.
type completion struct {
	d  *Device
	fn func()
}

// waitForQueue asks the browser to tell when the queue has finished what is submitted so far.
func (p *Provider) waitForQueue(d *Device, fn func()) {
	p.mu.Lock()
	p.nextID++
	id := p.nextID
	p.completions[id] = &completion{d: d, fn: fn}
	p.mu.Unlock()
	p.helper.Call("done", d.queue, id)
}

// onWorkDone is the browser telling that the queue finished (or, with a message, that it could not
// say, because the device was lost): the function of OnComplete runs on the UI goroutine.
func (p *Provider) onWorkDone(id int, _ string) {
	p.mu.Lock()
	c := p.completions[id]
	delete(p.completions, id)
	p.mu.Unlock()
	if c != nil {
		p.post(c.fn)
	}
}

// SetIndexBuffer32 implements explicit.Pass.
func (p *pass) SetIndexBuffer32(b explicit.Buffer, offset int) {
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	p.p.Call("setIndexBuffer", ib.buf, "uint32", offset)
}

// DrawIndirect implements explicit.Pass.
func (p *pass) DrawIndirect(b explicit.Buffer, offset int) {
	p.e.d.limits.CheckIndirect(b, offset, 16)
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	p.p.Call("drawIndirect", ib.buf, offset)
}

// DrawIndexedIndirect implements explicit.Pass.
func (p *pass) DrawIndexedIndirect(b explicit.Buffer, offset int) {
	p.e.d.limits.CheckIndirect(b, offset, 20)
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	p.p.Call("drawIndexedIndirect", ib.buf, offset)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUQueryUnsupported is the tag of the error of NewQuerySet on a device without timestamp queries
	// (explicit.TagGPUQueryUnsupported).
	tagGPUQueryUnsupported = "GPUQueryUnsupported"
)
