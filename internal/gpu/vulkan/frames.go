//go:build linux

package vulkan

import (
	"image"
	"image/color"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// framesInFlight is how many frames the GPU can work on at once (SPEC §9.3).
const framesInFlight = explicit.FramesInFlight

// stagingPerSlot is the size of the part of the staging ring that one frame uses.
const stagingPerSlot = 8 << 20

type slotState int

const (
	slotReady     slotState = iota // its fence is done and its pools are reset
	slotRecording                  // an encoder is recording into it
	slotInflight                   // submitted, the GPU may still be working
)

// frameSlot is what one frame in flight owns: a command pool and buffer, a fence,
// the descriptor pools of its bind groups, and what is released when the GPU is done.
type frameSlot struct {
	state    slotState
	pool     commandPoolH
	cb       commandBufferH
	fence    fenceH
	pools    []descriptorPoolH
	poolIdx  int // the pool that bind groups are made from
	deferred []func()
}

// runDeferred runs and forgets what was waiting for the GPU.
func (s *frameSlot) runDeferred() {
	fns := s.deferred
	s.deferred = nil
	for _, fn := range fns {
		fn()
	}
}

// pendingRead is a ReadTexture that waits for the frame to finish.
type pendingRead struct {
	buffer bufferH
	alloc  allocation
	rect   image.Rectangle
	format explicit.TextureFormat
	done   func(*image.RGBA, error)
}

// initFrames makes the command pools, command buffers, fences and the first
// descriptor pools of the slots.
func (d *Device) initFrames() error {
	a := d.arena
	for i := range d.slots {
		s := &d.slots[i]
		mark := a.mark()
		out := a.out()
		info := put(a, commandPoolCreateInfo{sType: sTypeCommandPoolCreateInfo, flags: commandPoolTransient, queueFamilyIndex: d.family})
		if err := check("vkCreateCommandPool", d.f.createCommandPool(d.dev, info, 0, out)); err != nil {
			a.rewind(mark)
			return err
		}
		s.pool = u64(out)
		alloc := put(a, commandBufferAllocateInfo{sType: sTypeCommandBufferAllocateInfo, commandPool: s.pool, level: commandBufferPrimary, commandBufferCount: 1})
		if err := check("vkAllocateCommandBuffers", d.f.allocateCommandBuffers(d.dev, alloc, out)); err != nil {
			a.rewind(mark)
			return err
		}
		s.cb = ptr(out)
		fence := put(a, fenceCreateInfo{sType: sTypeFenceCreateInfo})
		if err := check("vkCreateFence", d.f.createFence(d.dev, fence, 0, out)); err != nil {
			a.rewind(mark)
			return err
		}
		s.fence = u64(out)
		a.rewind(mark)
		if err := d.addDescriptorPool(s); err != nil {
			return err
		}
	}
	return nil
}

// destroyFrames destroys the pools and fences of the slots; the GPU is idle.
func (d *Device) destroyFrames() {
	for i := range d.slots {
		s := &d.slots[i]
		for _, p := range s.pools {
			d.f.destroyDescriptorPool(d.dev, p, 0)
		}
		if s.pool != 0 {
			d.f.destroyCommandPool(d.dev, s.pool, 0)
		}
		if s.fence != 0 {
			d.f.destroyFence(d.dev, s.fence, 0)
		}
		*s = frameSlot{}
	}
}

// Sizes of a descriptor pool: a frame makes bind groups from pools of this size and
// adds another when one is full.
const (
	poolSets     = 512
	poolUniforms = 512
	poolTextures = 2048
	poolSamplers = 512
)

// addDescriptorPool adds a descriptor pool to the slot and makes it the current one.
func (d *Device) addDescriptorPool(s *frameSlot) error {
	a := d.arena
	defer a.rewind(a.mark())
	sizes := putSlice(a, []descriptorPoolSize{
		{descriptorUniformDynamic, poolUniforms}, {descriptorSampledImage, poolTextures}, {descriptorSampler, poolSamplers},
	})
	info := put(a, descriptorPoolCreateInfo{sType: sTypeDescriptorPoolCreateInfo, maxSets: poolSets, poolSizeCount: 3, pPoolSizes: sizes})
	out := a.out()
	if err := check("vkCreateDescriptorPool", d.f.createDescriptorPool(d.dev, info, 0, out)); err != nil {
		return err
	}
	s.pools = append(s.pools, u64(out))
	s.poolIdx = len(s.pools) - 1
	return nil
}

// prepare makes the slot ready for a frame: if the GPU may be using it, it waits for
// its fence; then it runs what was waiting for the GPU, and resets the command pool,
// the descriptor pools and the staging ring of the slot.
func (d *Device) prepare(index int) error {
	s := &d.slots[index]
	if s.state == slotReady || s.state == slotRecording {
		return nil
	}
	if err := d.waitSlot(s); err != nil {
		return err
	}
	return d.recycle(index)
}

// recycle makes a slot whose fence is done ready for a frame: it runs what was waiting
// for the GPU and resets the command pool, the descriptor pools and the staging ring
// of the slot. The queue runs the frames in order, so when the fence of a frame is
// done, so are the frames before it, and what they used can be released.
func (d *Device) recycle(index int) error {
	s := &d.slots[index]
	s.runDeferred()
	if err := check("vkResetCommandPool", d.f.resetCommandPool(d.dev, s.pool, 0)); err != nil {
		return err
	}
	for _, p := range s.pools {
		if err := check("vkResetDescriptorPool", d.f.resetDescriptorPool(d.dev, p, 0)); err != nil {
			return err
		}
	}
	s.poolIdx = 0
	d.staging.heads[index] = 0
	s.state = slotReady
	return nil
}

// waitSlot waits for the fence of a submitted frame and resets it.
func (d *Device) waitSlot(s *frameSlot) error {
	a := d.arena
	defer a.rewind(a.mark())
	fences := putSlice(a, []fenceH{s.fence})
	if err := check("vkWaitForFences", d.f.waitForFences(d.dev, 1, fences, 1, ^uint64(0))); err != nil {
		return err
	}
	return check("vkResetFences", d.f.resetFences(d.dev, 1, fences))
}

// retire runs fn when the GPU is done with what the frames in flight use: at once
// when none is, else when the fence of the frame being recorded or the last one
// submitted is done (the queue runs frames in order).
func (d *Device) retire(fn func()) {
	switch {
	case d.open != nil:
		s := &d.slots[d.open.slot]
		s.deferred = append(s.deferred, fn)
	case d.last >= 0 && d.slots[d.last].state == slotInflight:
		s := &d.slots[d.last]
		s.deferred = append(s.deferred, fn)
	default:
		fn()
	}
}

// retireWithNextFrame runs fn when the GPU is done with the frame that the commands
// being queued now belong to: the frame being recorded, or the next one. A one-off
// staging buffer waits for the frame that copies from it.
func (d *Device) retireWithNextFrame(fn func()) {
	slot := d.next
	if d.open != nil {
		slot = d.open.slot
	}
	s := &d.slots[slot]
	s.deferred = append(s.deferred, fn)
}

// encoder records the commands of one frame.
type encoder struct {
	d     *Device
	slot  int
	cb    commandBufferH
	pass  *pass
	reads []*pendingRead
	// acquired are the swapchain images this frame draws into.
	acquired []acquiredImage
}

var _ explicit.Encoder = (*encoder)(nil)

// Begin implements explicit.Device: it waits for the slot of the frame if the GPU is
// still using it, and starts its command buffer with what was queued since the last
// frame (the layouts of new textures, uploads). It fails with VulkanFrameOpen when the
// previous frame was not submitted.
func (d *Device) Begin() (explicit.Encoder, error) {
	if d.open != nil {
		return nil, errors.New("VulkanFrameOpen", "Begin was called before the frame in progress was submitted")
	}
	if err := d.prepare(d.next); err != nil {
		return nil, err
	}
	s := &d.slots[d.next]
	a := d.arena
	defer a.rewind(a.mark())
	info := put(a, commandBufferBeginInfo{sType: sTypeCommandBufferBeginInfo, flags: commandBufferOneTime})
	if err := check("vkBeginCommandBuffer", d.f.beginCommandBuffer(s.cb, info)); err != nil {
		return nil, err
	}
	s.state = slotRecording
	e := &encoder{d: d, slot: d.next, cb: s.cb}
	d.open = e
	pending := d.pending
	d.pending = nil
	for _, fn := range pending {
		fn(e.cb)
	}
	return e, nil
}

// barrier records a memory barrier that makes every write before it visible to every
// command and to the host after it: simple, and what the passes and copies need.
func (d *Device) barrier(cb commandBufferH, dstStage, dstAccess uint64) {
	a := d.arena
	defer a.rewind(a.mark())
	mb := put(a, memoryBarrier2{
		sType: sTypeMemoryBarrier2, srcStageMask: stage2AllCommands, srcAccessMask: access2MemoryWrite,
		dstStageMask: dstStage, dstAccessMask: dstAccess,
	})
	dep := put(a, dependencyInfo{sType: sTypeDependencyInfo, memoryBarrierCount: 1, pMemoryBarriers: mb})
	d.f.cmdPipelineBarrier2(cb, dep)
}

// barrierAll is barrier for the commands that follow.
func (d *Device) barrierAll(cb commandBufferH) {
	d.barrier(cb, stage2AllCommands, access2MemoryRead|access2MemoryWrite)
}

// foreign is the panic of a resource that this device did not make (a programmer error,
// SPEC §14.2).
func foreign(what string) error {
	return errors.New("VulkanForeignResource", "the resource was not made by this Vulkan device", "resource", what)
}

// BeginPass implements explicit.Encoder. The target must be a texture of this device
// with TextureRenderTarget.
func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	t, ok := target.(*texture)
	if !ok || t.d != e.d {
		panic(foreign("texture"))
	}
	if e.pass != nil {
		panic(errors.New("VulkanPassNotEnded", "BeginPass was called before the previous pass ended"))
	}
	d := e.d
	a := d.arena
	defer a.rewind(a.mark())
	op := uint32(loadOpClear)
	if load == explicit.LoadLoad {
		op = loadOpLoad
	}
	attach := put(a, renderingAttachmentInfo{
		sType: sTypeRenderingAttachmentInfo, imageView: t.targetView(), imageLayout: layoutGeneral,
		loadOp: op, storeOp: storeOpStore,
		clearValue: clearValue{color: [4]float32{clear.R, clear.G, clear.B, clear.A}},
	})
	w, h := uint32(t.desc.Width), uint32(t.desc.Height)
	info := put(a, renderingInfo{
		sType: sTypeRenderingInfo, renderArea: rect2D{extent: extent2D{w, h}}, layerCount: 1,
		colorAttachmentCount: 1, pColorAttachments: attach,
	})
	d.f.cmdBeginRendering(e.cb, info)
	vp := put(a, viewport{width: float32(w), height: float32(h), maxDepth: 1})
	d.f.cmdSetViewport(e.cb, 0, 1, vp)
	sc := put(a, rect2D{extent: extent2D{w, h}})
	d.f.cmdSetScissor(e.cb, 0, 1, sc)
	e.pass = &pass{e: e, target: t}
	return e.pass
}

// CopyTexture implements explicit.Encoder.
func (e *encoder) CopyTexture(src explicit.Texture, r image.Rectangle, dst explicit.Texture, at image.Point) {
	s, ok1 := src.(*texture)
	t, ok2 := dst.(*texture)
	if !ok1 || !ok2 || s.d != e.d || t.d != e.d {
		panic(foreign("texture"))
	}
	d := e.d
	a := d.arena
	defer a.rewind(a.mark())
	layers := imageSubresourceLayers{aspectMask: imageAspectColor, layerCount: 1}
	region := put(a, imageCopy{
		srcSubresource: layers, srcOffset: offset3D{int32(r.Min.X), int32(r.Min.Y), 0},
		dstSubresource: layers, dstOffset: offset3D{int32(at.X), int32(at.Y), 0},
		extent: extent3D{uint32(r.Dx()), uint32(r.Dy()), 1},
	})
	d.f.cmdCopyImage(e.cb, s.image, layoutGeneral, t.image, layoutGeneral, 1, region)
	d.barrierAll(e.cb)
}

// ReadTexture implements explicit.Encoder: it copies the rectangle into a buffer that
// the CPU can read, and done runs at the end of Submit, after the GPU finished.
func (e *encoder) ReadTexture(src explicit.Texture, r image.Rectangle, done func(*image.RGBA, error)) {
	s, ok := src.(*texture)
	if !ok || s.d != e.d {
		panic(foreign("texture"))
	}
	d := e.d
	bpp := s.desc.Format.BytesPerPixel()
	size := uint64(r.Dx() * r.Dy() * bpp)
	buf, al, _, err := d.newHostBuffer(size, bufferUsageTransferDst)
	if err != nil {
		done(nil, err)
		return
	}
	a := d.arena
	defer a.rewind(a.mark())
	region := put(a, bufferImageCopy{
		imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, layerCount: 1},
		imageOffset:      offset3D{int32(r.Min.X), int32(r.Min.Y), 0},
		imageExtent:      extent3D{uint32(r.Dx()), uint32(r.Dy()), 1},
	})
	d.f.cmdCopyImageToBuffer(e.cb, s.image, layoutGeneral, buf, 1, region)
	d.barrier(e.cb, stage2Host, access2HostRead)
	e.reads = append(e.reads, &pendingRead{buffer: buf, alloc: al, rect: r, format: s.desc.Format, done: done})
}

// Submit implements explicit.Encoder. When the frame asked for a texture to be read, it
// waits for the GPU and runs the done functions before it returns.
func (e *encoder) Submit() error {
	d := e.d
	if d.open != e {
		return errors.New("VulkanFrameNotOpen", "Submit was called on a frame that is not the one in progress")
	}
	s := &d.slots[e.slot]
	d.open = nil
	// The images of the swapchains end in the layout that presenting needs.
	for _, ai := range e.acquired {
		d.imageBarrier(e.cb, ai.sc.images[ai.idx].image, layoutGeneral, layoutPresentSrc)
	}
	if err := check("vkEndCommandBuffer", d.f.endCommandBuffer(e.cb)); err != nil {
		return err
	}
	a := d.arena
	defer a.rewind(a.mark())
	cbs := putSlice(a, []commandBufferH{e.cb})
	info := submitInfo{sType: sTypeSubmitInfo, commandBufferCount: 1, pCommandBuffers: cbs}
	if n := len(e.acquired); n > 0 {
		// Wait for each image to be acquired, and signal that each is drawn.
		waits := make([]semaphoreH, n)
		stages := make([]uint32, n)
		signals := make([]semaphoreH, n)
		for i, ai := range e.acquired {
			waits[i], stages[i], signals[i] = ai.wait, stageAllCommands, ai.sc.renderSems[ai.idx]
		}
		info.waitSemaphoreCount, info.pWaitSemaphores, info.pWaitDstStageMask = uint32(n), putSlice(a, waits), putSlice(a, stages)
		info.signalSemaphoreCount, info.pSignalSemaphores = uint32(n), putSlice(a, signals)
	}
	submit := put(a, info)
	if err := check("vkQueueSubmit", d.f.queueSubmit(d.queue, 1, submit, s.fence)); err != nil {
		return err
	}
	s.state = slotInflight
	d.last = e.slot
	d.next = (e.slot + 1) % framesInFlight
	if len(e.reads) == 0 {
		return nil
	}
	if err := d.waitSlot(s); err != nil {
		for _, rd := range e.reads {
			rd.done(nil, err)
		}
		return err
	}
	// The fence is done and reset, so the slot is ready again; prepare would wait on it
	// forever.
	if err := d.recycle(e.slot); err != nil {
		return err
	}
	for _, rd := range e.reads {
		d.finishRead(rd)
	}
	return nil
}

// finishRead converts what the GPU copied into the buffer, calls done, and frees the
// buffer.
func (d *Device) finishRead(rd *pendingRead) {
	w, h := rd.rect.Dx(), rd.rect.Dy()
	bpp := rd.format.BytesPerPixel()
	raw := readBytes(rd.alloc.mapped(), w*h*bpp)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		var c color.RGBA
		switch rd.format {
		case explicit.TextureFormatBGRA8Unorm:
			c = color.RGBA{raw[i*4+2], raw[i*4+1], raw[i*4], raw[i*4+3]}
		case explicit.TextureFormatR8Unorm:
			c = color.RGBA{raw[i], raw[i], raw[i], 255}
		default:
			c = color.RGBA{raw[i*4], raw[i*4+1], raw[i*4+2], raw[i*4+3]}
		}
		img.SetRGBA(i%w, i/w, c)
	}
	d.destroyBuffer(rd.buffer, rd.alloc)
	rd.done(img, nil)
}

// pass records the draws of one render pass.
type pass struct {
	e        *encoder
	target   *texture
	pipeline *pipeline
}

var _ explicit.Pass = (*pass)(nil)

// SetPipeline implements explicit.Pass.
func (p *pass) SetPipeline(pl explicit.Pipeline) {
	pp, ok := pl.(*pipeline)
	if !ok || pp.d != p.e.d {
		panic(foreign("pipeline"))
	}
	p.pipeline = pp
	p.e.d.f.cmdBindPipeline(p.e.cb, pipelineBindGraphics, pp.pipeline)
}

// SetBindGroup implements explicit.Pass.
func (p *pass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	d := p.e.d
	d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok || bg.d != d {
		panic(foreign("bind group"))
	}
	a := d.arena
	defer a.rewind(a.mark())
	sets := putSlice(a, []descriptorSetH{bg.set})
	d.f.cmdBindDescriptorSets(p.e.cb, pipelineBindGraphics, p.pipeline.layout, uint32(group), 1, sets,
		uint32(len(dynamicOffsets)), putSlice(a, dynamicOffsets))
}

// SetVertexBuffer implements explicit.Pass.
func (p *pass) SetVertexBuffer(slot int, b explicit.Buffer, offset int) {
	d := p.e.d
	d.limits.CheckSetVertexBuffer(slot)
	vb, ok := b.(*buffer)
	if !ok || vb.d != d {
		panic(foreign("buffer"))
	}
	a := d.arena
	defer a.rewind(a.mark())
	d.f.cmdBindVertexBuffers(p.e.cb, uint32(slot), 1, putSlice(a, []bufferH{vb.buffer}), putSlice(a, []uint64{uint64(offset)}))
}

// SetIndexBuffer implements explicit.Pass.
func (p *pass) SetIndexBuffer(b explicit.Buffer, offset int) {
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(foreign("buffer"))
	}
	p.e.d.f.cmdBindIndexBuffer(p.e.cb, ib.buffer, uint64(offset), indexTypeUint16)
}

// SetScissor implements explicit.Pass. The rectangle is clipped to the target.
func (p *pass) SetScissor(r image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, p.target.desc.Width, p.target.desc.Height))
	a := p.e.d.arena
	defer a.rewind(a.mark())
	sc := put(a, rect2D{offset: offset2D{int32(r.Min.X), int32(r.Min.Y)}, extent: extent2D{uint32(r.Dx()), uint32(r.Dy())}})
	p.e.d.f.cmdSetScissor(p.e.cb, 0, 1, sc)
}

// Draw implements explicit.Pass.
func (p *pass) Draw(vertices, instances, firstVertex, firstInstance int) {
	p.e.d.f.cmdDraw(p.e.cb, uint32(vertices), uint32(instances), uint32(firstVertex), uint32(firstInstance))
}

// DrawIndexed implements explicit.Pass.
func (p *pass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	p.e.d.f.cmdDrawIndexed(p.e.cb, uint32(indices), uint32(instances), uint32(firstIndex), int32(baseVertex), uint32(firstInstance))
}

// End implements explicit.Pass.
func (p *pass) End() {
	d := p.e.d
	d.f.cmdEndRendering(p.e.cb)
	d.barrierAll(p.e.cb)
	p.e.pass = nil
}
