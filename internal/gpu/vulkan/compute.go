//go:build linux

package vulkan

import (
	"sort"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The part of the device that step 30.2 added (SPEC §27.1): compute pipelines and passes, the
// copies between buffers and textures, the read of a buffer, indirect draws and timestamp queries.

// layouts are the descriptor set layouts and the pipeline layout of a pipeline, made from the
// bindings of its entry points.
type layouts struct {
	bindings   map[[2]int]explicit.ShaderBinding
	setLayouts []descriptorSetLayoutH
	layout     pipelineLayoutH
}

// destroy destroys the layouts.
func (l layouts) destroy(d *Device) {
	if l.layout != 0 {
		d.f.destroyPipelineLayout(d.dev, l.layout, 0)
	}
	for _, s := range l.setLayouts {
		d.f.destroyDescriptorSetLayout(d.dev, s, 0)
	}
}

// makeLayouts makes the layouts for the entry points of a pipeline, each with the shader stage it runs
// in. A binding that two entry points share is visible to both stages. Uniform buffers have dynamic
// offsets (SPEC §9.3); a storage buffer has its offset in the bind group. The pipeline has a set layout for
// every group that the module declares, an empty one for a group only other entry points use: SwiftShader,
// the host driver of the Android emulator, looks up the set of every variable of the module, not only of
// those that the entry point reads, and crashes when the pipeline layout lacks the set.
func (d *Device) makeLayouts(module *explicit.Shader, entries map[uint32]explicit.EntryPoint) (layouts, error) {
	// The bindings of each group, with the stages that read them.
	type slot struct {
		kind  explicit.BindingKind
		stage uint32
	}
	l := layouts{bindings: map[[2]int]explicit.ShaderBinding{}}
	groups := map[int]map[int]*slot{}
	maxGroup := -1
	for _, entry := range module.Entries {
		for _, b := range entry.Bindings {
			maxGroup = max(maxGroup, b.Group)
		}
	}
	for stage, entry := range entries {
		for _, b := range entry.Bindings {
			if groups[b.Group] == nil {
				groups[b.Group] = map[int]*slot{}
			}
			s := groups[b.Group][b.Binding]
			if s == nil {
				s = &slot{kind: b.Kind}
				groups[b.Group][b.Binding] = s
			}
			s.stage |= stage
			l.bindings[[2]int{b.Group, b.Binding}] = b
			maxGroup = max(maxGroup, b.Group)
		}
	}
	a := d.arena
	defer a.rewind(a.mark())
	for g := 0; g <= maxGroup; g++ {
		var bs []descriptorSetLayoutBinding
		numbers := make([]int, 0, len(groups[g]))
		for n := range groups[g] {
			numbers = append(numbers, n)
		}
		sort.Ints(numbers)
		for _, n := range numbers {
			s := groups[g][n]
			bs = append(bs, descriptorSetLayoutBinding{binding: uint32(n), descriptorType: descriptorType(s.kind), descriptorCount: 1, stageFlags: s.stage})
		}
		info := put(a, descriptorSetLayoutCreateInfo{sType: sTypeDescriptorSetLayoutCreateInfo, bindingCount: uint32(len(bs)), pBindings: putSlice(a, bs)})
		out := a.out()
		if err := check("vkCreateDescriptorSetLayout", d.f.createDescriptorSetLayout(d.dev, info, 0, out)); err != nil {
			l.destroy(d)
			return layouts{}, err
		}
		l.setLayouts = append(l.setLayouts, u64(out))
	}
	layoutInfo := put(a, pipelineLayoutCreateInfo{
		sType: sTypePipelineLayoutCreateInfo, setLayoutCount: uint32(len(l.setLayouts)), pSetLayouts: putSlice(a, l.setLayouts),
	})
	out := a.out()
	if err := check("vkCreatePipelineLayout", d.f.createPipelineLayout(d.dev, layoutInfo, 0, out)); err != nil {
		l.destroy(d)
		return layouts{}, err
	}
	l.layout = u64(out)
	return l, nil
}

// computePipeline is a compute pipeline with the layouts of its bind groups.
type computePipeline struct {
	d         *Device
	desc      explicit.ComputePipelineDesc
	pipeline  pipelineH
	lay       layouts
	destroyed bool
}

var _ explicit.ComputePipeline = (*computePipeline)(nil)

// Desc implements explicit.ComputePipeline.
func (p *computePipeline) Desc() explicit.ComputePipelineDesc { return p.desc }

// Destroy implements explicit.ComputePipeline: the pipeline is released when the GPU is done with
// the frames that may use it.
func (p *computePipeline) Destroy() {
	if p.destroyed {
		return
	}
	p.destroyed = true
	delete(p.d.live, p)
	p.d.retire(func() { p.release() })
}

func (p *computePipeline) release() {
	p.d.f.destroyPipeline(p.d.dev, p.pipeline, 0)
	p.lay.destroy(p.d)
}

// NewComputePipeline implements explicit.Device.
func (d *Device) NewComputePipeline(desc explicit.ComputePipelineDesc) (explicit.ComputePipeline, error) {
	d.limits.CheckComputePipeline(desc)
	if desc.Shader == nil {
		return nil, errors.New(tagVulkanPipelineInvalid, "a pipeline needs a shader")
	}
	entry, ok := desc.Shader.Entry(desc.Entry)
	if !ok || entry.Stage != explicit.ShaderStageCompute {
		return nil, errors.New(tagVulkanPipelineInvalid, "the shader lacks the compute entry point", "shader", desc.Shader.Name, "entry", desc.Entry)
	}
	module, err := d.shaderModule(desc.Shader, false)
	if err != nil {
		return nil, err
	}
	lay, err := d.makeLayouts(desc.Shader, map[uint32]explicit.EntryPoint{stageCompute: entry})
	if err != nil {
		return nil, err
	}
	a := d.arena
	defer a.rewind(a.mark())
	info := put(a, computePipelineCreateInfo{
		sType:  sTypeComputePipelineCreateInfo,
		stage:  pipelineShaderStageCreateInfo{sType: sTypePipelineShaderStageCreateInfo, stage: stageCompute, module: module, pName: a.cstring(desc.Entry)},
		layout: lay.layout,
	})
	out := a.out()
	start := time.Now()
	err = check("vkCreateComputePipelines", d.f.createComputePipelines(d.dev, d.pcache.handle, 1, info, 0, out))
	d.pcache.took += time.Since(start)
	if err != nil {
		lay.destroy(d)
		return nil, err
	}
	d.pcache.pipelines++
	p := &computePipeline{d: d, desc: desc, pipeline: u64(out), lay: lay}
	d.live[p] = func() { p.destroyed = true; p.release() }
	return p, nil
}

// NewComputeBindGroup implements explicit.Device: like NewBindGroup, from the layouts of a compute
// pipeline.
func (d *Device) NewComputeBindGroup(p explicit.ComputePipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	cp, ok := p.(*computePipeline)
	if !ok || cp.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "compute pipeline"))
	}
	return d.makeBindGroup(cp.lay.setLayouts, cp.lay.bindings, group, entries)
}

// querySet is a VkQueryPool of timestamps.
type querySet struct {
	d         *Device
	pool      uint64
	count     int
	destroyed bool
}

var _ explicit.QuerySet = (*querySet)(nil)

// Count implements explicit.QuerySet.
func (q *querySet) Count() int { return q.count }

// Destroy implements explicit.QuerySet.
func (q *querySet) Destroy() {
	if q.destroyed {
		return
	}
	q.destroyed = true
	delete(q.d.live, q)
	q.d.retire(func() { q.d.f.destroyQueryPool(q.d.dev, q.pool, 0) })
}

// NewQuerySet implements explicit.Device. It fails with GPUQueryUnsupported when the queue cannot write
// timestamps.
func (d *Device) NewQuerySet(count int) (explicit.QuerySet, error) {
	if !d.features.TimestampQuery {
		return nil, errors.New(tagGPUQueryUnsupported, "the device has no timestamp queries", "count", count)
	}
	a := d.arena
	defer a.rewind(a.mark())
	info := put(a, queryPoolCreateInfo{sType: sTypeQueryPoolCreateInfo, queryType: queryTypeTimestamp, queryCount: uint32(max(count, 1))})
	out := a.out()
	if err := check("vkCreateQueryPool", d.f.createQueryPool(d.dev, info, 0, out)); err != nil {
		return nil, err
	}
	q := &querySet{d: d, pool: u64(out), count: count}
	d.live[q] = func() { q.destroyed = true; d.f.destroyQueryPool(d.dev, q.pool, 0) }
	return q, nil
}

// timestampWriter writes the two timestamps of a pass.
type timestampWriter struct {
	e  *encoder
	ts *explicit.PassTimestamps
}

// beginTimestamps resets the queries of a pass and writes the first one. A pass without timestamps
// gets a writer that does nothing. The commands are outside a render pass, which is where Vulkan allows
// a reset.
func (e *encoder) beginTimestamps(ts *explicit.PassTimestamps) timestampWriter {
	w := timestampWriter{e: e, ts: ts}
	if ts == nil {
		return w
	}
	q, ok := ts.Set.(*querySet)
	if !ok || q.d != e.d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "query set"))
	}
	if ts.Begin >= 0 {
		e.d.f.cmdResetQueryPool(e.cb, q.pool, uint32(ts.Begin), 1)
		e.d.f.cmdWriteTimestamp2(e.cb, stage2AllCommands, q.pool, uint32(ts.Begin))
	}
	if ts.End >= 0 {
		e.d.f.cmdResetQueryPool(e.cb, q.pool, uint32(ts.End), 1)
	}
	return w
}

// end writes the second timestamp.
func (w timestampWriter) end() {
	if w.ts == nil || w.ts.End < 0 {
		return
	}
	w.e.d.f.cmdWriteTimestamp2(w.e.cb, stage2AllCommands, w.ts.Set.(*querySet).pool, uint32(w.ts.End))
}

// ResolveQuerySet implements explicit.Encoder.
func (e *encoder) ResolveQuerySet(set explicit.QuerySet, first, count int, dst explicit.Buffer, dstOffset int) {
	d := e.d
	d.limits.CheckResolveQuerySet(set, first, count, dst, dstOffset)
	q, ok := set.(*querySet)
	b, ok2 := dst.(*buffer)
	if !ok || !ok2 || q.d != d || b.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "query set or buffer"))
	}
	if count == 0 {
		return
	}
	// The timestamps are written by the commands before, whatever pass they were in.
	d.barrierAll(e.cb)
	d.f.cmdCopyQueryPoolResults(e.cb, q.pool, uint32(first), uint32(count), b.buffer, uint64(dstOffset), 8, queryResult64)
	d.barrierAll(e.cb)
}

// computePass records the dispatches of one compute pass.
type computePass struct {
	e          *encoder
	pipeline   *computePipeline
	timestamps timestampWriter
}

var _ explicit.ComputePass = (*computePass)(nil)

// BeginComputePass implements explicit.Encoder.
func (e *encoder) BeginComputePass(desc explicit.ComputePassDesc) explicit.ComputePass {
	if e.pass != nil || e.cpass != nil {
		panic(errors.New(tagVulkanPassNotEnded, "BeginComputePass was called before the previous pass ended"))
	}
	e.d.limits.CheckTimestamps(e.d.features, desc.Timestamps)
	e.cpass = &computePass{e: e, timestamps: e.beginTimestamps(desc.Timestamps)}
	return e.cpass
}

// SetPipeline implements explicit.ComputePass.
func (p *computePass) SetPipeline(pl explicit.ComputePipeline) {
	cp, ok := pl.(*computePipeline)
	if !ok || cp.d != p.e.d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "compute pipeline"))
	}
	p.pipeline = cp
	p.e.d.f.cmdBindPipeline(p.e.cb, pipelineBindCompute, cp.pipeline)
}

// SetBindGroup implements explicit.ComputePass.
func (p *computePass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	d := p.e.d
	d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok || bg.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "bind group"))
	}
	a := d.arena
	defer a.rewind(a.mark())
	sets := putSlice(a, []descriptorSetH{bg.set})
	d.f.cmdBindDescriptorSets(p.e.cb, pipelineBindCompute, p.pipeline.lay.layout, uint32(group), 1, sets,
		uint32(len(dynamicOffsets)), putSlice(a, dynamicOffsets))
}

// Dispatch implements explicit.ComputePass. A barrier follows, so that the next dispatch sees what this
// one wrote, as in WebGPU.
func (p *computePass) Dispatch(x, y, z int) {
	d := p.e.d
	d.limits.CheckDispatch(x, y, z)
	d.f.cmdDispatch(p.e.cb, uint32(x), uint32(y), uint32(z))
	d.barrierAll(p.e.cb)
}

// DispatchIndirect implements explicit.ComputePass.
func (p *computePass) DispatchIndirect(b explicit.Buffer, offset int) {
	d := p.e.d
	d.limits.CheckIndirect(b, offset, 12)
	ib, ok := b.(*buffer)
	if !ok || ib.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	d.f.cmdDispatchIndirect(p.e.cb, ib.buffer, uint64(offset))
	d.barrierAll(p.e.cb)
}

// End implements explicit.ComputePass.
func (p *computePass) End() {
	d := p.e.d
	p.timestamps.end()
	d.barrierAll(p.e.cb)
	p.e.cpass = nil
}

// CopyBuffer implements explicit.Encoder.
func (e *encoder) CopyBuffer(src explicit.Buffer, srcOffset int, dst explicit.Buffer, dstOffset, size int) {
	d := e.d
	d.limits.CheckCopyBuffer(src, srcOffset, dst, dstOffset, size)
	s, ok1 := src.(*buffer)
	t, ok2 := dst.(*buffer)
	if !ok1 || !ok2 || s.d != d || t.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	if size == 0 {
		return
	}
	a := d.arena
	defer a.rewind(a.mark())
	d.f.cmdCopyBuffer(e.cb, s.buffer, t.buffer, 1, put(a, bufferCopy{srcOffset: uint64(srcOffset), dstOffset: uint64(dstOffset), size: uint64(size)}))
	d.barrierAll(e.cb)
}

// bufferTextureCopy is the Vulkan region of a copy between a buffer and a texture.
func bufferTextureCopy(t *texture, region explicit.TextureRegion, layout explicit.BufferLayout) bufferImageCopy {
	w, h, n := region.Size[0], region.Size[1], region.Size[2]
	bpp := t.desc.Format.BytesPerPixel()
	aspect := uint32(imageAspectColor)
	if t.desc.Format.IsDepthStencil() {
		aspect = imageAspectDepth
	}
	c := bufferImageCopy{
		bufferOffset:      uint64(layout.Offset),
		bufferRowLength:   uint32(layout.BytesPerRow / bpp),
		bufferImageHeight: uint32(layout.RowsPerImage),
		imageSubresource:  imageSubresourceLayers{aspectMask: aspect, mipLevel: uint32(t.base + region.Level), baseArrayLayer: uint32(t.baseLayer), layerCount: 1},
		imageOffset:       offset3D{int32(region.Origin[0]), int32(region.Origin[1]), 0},
		imageExtent:       extent3D{uint32(w), uint32(h), 1},
	}
	switch t.desc.Kind {
	case explicit.TextureKind2DArray, explicit.TextureKindCube:
		c.imageSubresource.baseArrayLayer = uint32(t.baseLayer + region.Origin[2])
		c.imageSubresource.layerCount = uint32(n)
	case explicit.TextureKind3D:
		c.imageOffset.z = int32(region.Origin[2])
		c.imageExtent.depth = uint32(n)
	}
	return c
}

// CopyBufferToTexture implements explicit.Encoder.
func (e *encoder) CopyBufferToTexture(src explicit.Buffer, layout explicit.BufferLayout, dst explicit.Texture, region explicit.TextureRegion) {
	d := e.d
	d.limits.CheckBufferTexture(src, layout, dst.Desc(), region, true)
	b, ok1 := src.(*buffer)
	t, ok2 := dst.(*texture)
	if !ok1 || !ok2 || b.d != d || t.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer or texture"))
	}
	a := d.arena
	defer a.rewind(a.mark())
	d.f.cmdCopyBufferToImage(e.cb, b.buffer, t.image, layoutGeneral, 1, put(a, bufferTextureCopy(t, region, layout)))
	d.barrierAll(e.cb)
}

// CopyTextureToBuffer implements explicit.Encoder.
func (e *encoder) CopyTextureToBuffer(src explicit.Texture, region explicit.TextureRegion, dst explicit.Buffer, layout explicit.BufferLayout) {
	d := e.d
	d.limits.CheckBufferTexture(dst, layout, src.Desc(), region, false)
	t, ok1 := src.(*texture)
	b, ok2 := dst.(*buffer)
	if !ok1 || !ok2 || b.d != d || t.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer or texture"))
	}
	a := d.arena
	defer a.rewind(a.mark())
	d.f.cmdCopyImageToBuffer(e.cb, t.image, layoutGeneral, b.buffer, 1, put(a, bufferTextureCopy(t, region, layout)))
	d.barrierAll(e.cb)
}

// ReadBuffer implements explicit.Encoder: the bytes are copied into a host buffer at this point of the
// commands, and done gets them when Submit has seen the GPU finish.
func (e *encoder) ReadBuffer(src explicit.Buffer, offset, size int, done func(data []byte, err error)) {
	d := e.d
	d.limits.CheckReadBuffer(src, offset, size)
	s, ok := src.(*buffer)
	if !ok || s.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	host, al, _, err := d.newHostBuffer(uint64(size), bufferUsageTransferDst)
	if err != nil {
		done(nil, err)
		return
	}
	if size > 0 {
		a := d.arena
		defer a.rewind(a.mark())
		d.f.cmdCopyBuffer(e.cb, s.buffer, host, 1, put(a, bufferCopy{srcOffset: uint64(offset), size: uint64(size)}))
		d.barrier(e.cb, stage2Host, access2HostRead)
	}
	e.reads = append(e.reads, &pendingRead{buffer: host, alloc: al, size: size, doneRaw: done})
}

// OnComplete implements explicit.Encoder: Submit waits for the GPU and runs fn.
func (e *encoder) OnComplete(fn func()) { e.completes = append(e.completes, fn) }

// SetIndexBuffer32 implements explicit.Pass.
func (p *pass) SetIndexBuffer32(b explicit.Buffer, offset int) {
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	p.setIndexBuffer(ib, offset, indexTypeUint32)
}

// DrawIndirect implements explicit.Pass.
func (p *pass) DrawIndirect(b explicit.Buffer, offset int) {
	d := p.e.d
	d.limits.CheckIndirect(b, offset, 16)
	ib, ok := b.(*buffer)
	if !ok || ib.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	p.bindVertexBuffers()
	d.f.cmdDrawIndirect(p.e.cb, ib.buffer, uint64(offset), 1, 16)
}

// DrawIndexedIndirect implements explicit.Pass.
func (p *pass) DrawIndexedIndirect(b explicit.Buffer, offset int) {
	d := p.e.d
	d.limits.CheckIndirect(b, offset, 20)
	ib, ok := b.(*buffer)
	if !ok || ib.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	p.bindVertexBuffers()
	d.f.cmdDrawIndexedIndirect(p.e.cb, ib.buffer, uint64(offset), 1, 20)
}

// bytesOf copies the n bytes of mapped memory.
func bytesOf(p uintptr, n int) []byte { return append([]byte(nil), dl.Bytes(p, n)...) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUQueryUnsupported is the tag of the error of NewQuerySet on a device without timestamp queries
	// (explicit.TagGPUQueryUnsupported).
	tagGPUQueryUnsupported = "GPUQueryUnsupported"
)
