//go:build linux

package vulkan

import (
	"image"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// buffer is a host-visible, coherent, mapped buffer (SPEC §9.4).
type buffer struct {
	d         *Device
	buffer    bufferH
	alloc     allocation
	size      int
	destroyed bool
}

var _ explicit.Buffer = (*buffer)(nil)

// Size implements explicit.Buffer.
func (b *buffer) Size() int { return b.size }

// Destroy implements explicit.Buffer: the buffer is released when the GPU is done with
// the frames that may use it.
func (b *buffer) Destroy() {
	if b.destroyed {
		return
	}
	b.destroyed = true
	delete(b.d.live, b)
	b.d.retire(func() { b.d.destroyBuffer(b.buffer, b.alloc) })
}

// destroyBuffer destroys a VkBuffer and gives its memory back.
func (d *Device) destroyBuffer(buf bufferH, al allocation) {
	d.f.destroyBuffer(d.dev, buf, 0)
	d.alloc.release(al)
}

// newHostBuffer makes a buffer in host-visible, coherent memory, mapped for good.
func (d *Device) newHostBuffer(size uint64, usage uint32) (bufferH, allocation, uintptr, error) {
	a := d.arena
	defer a.rewind(a.mark())
	info := put(a, bufferCreateInfo{sType: sTypeBufferCreateInfo, size: max(size, 4), usage: usage, sharingMode: sharingExclusive})
	out := a.out()
	if err := check("vkCreateBuffer", d.f.createBuffer(d.dev, info, 0, out)); err != nil {
		return 0, allocation{}, 0, err
	}
	buf := u64(out)
	req := a.alloc(unsafeSizeof[memoryRequirements](), 8)
	d.f.getBufferMemoryRequirements(d.dev, buf, req)
	al, err := d.alloc.alloc(*(*memoryRequirements)(dl.Pointer(req)), true, true)
	if err != nil {
		d.f.destroyBuffer(d.dev, buf, 0)
		return 0, allocation{}, 0, err
	}
	if err := check("vkBindBufferMemory", d.f.bindBufferMemory(d.dev, buf, al.block.mem, al.offset)); err != nil {
		d.destroyBuffer(buf, al)
		return 0, allocation{}, 0, err
	}
	return buf, al, al.mapped(), nil
}

// NewBuffer implements explicit.Device.
func (d *Device) NewBuffer(size int, usage explicit.BufferUsage) (explicit.Buffer, error) {
	var u uint32 = bufferUsageTransferDst | bufferUsageTransferSrc
	if usage&explicit.BufferVertex != 0 {
		u |= bufferUsageVertex
	}
	if usage&explicit.BufferIndex != 0 {
		u |= bufferUsageIndex
	}
	if usage&explicit.BufferUniform != 0 {
		u |= bufferUsageUniform
	}
	buf, al, _, err := d.newHostBuffer(uint64(size), u)
	if err != nil {
		return nil, err
	}
	b := &buffer{d: d, buffer: buf, alloc: al, size: size}
	d.live[b] = func() { b.destroyed = true; d.destroyBuffer(b.buffer, b.alloc) }
	return b, nil
}

// WriteBuffer implements explicit.Device: the memory is mapped and coherent, so the
// data is copied at once.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	vb, ok := b.(*buffer)
	if !ok || vb.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
	}
	if offset < 0 || offset+len(data) > vb.size {
		panic(errors.New(tagVulkanWriteOutOfRange, "the write is outside the buffer", "offset", offset, "bytes", len(data), "size", vb.size))
	}
	if p := vb.alloc.mapped(); p != 0 {
		copy(dl.Bytes(p+uintptr(offset), len(data)), data)
	}
}

// texture is a 2D image in device-local memory, kept in the GENERAL layout.
type texture struct {
	d         *Device
	desc      explicit.TextureDesc
	image     imageH
	view      imageViewH // every mip level, for sampling; 0 for a texture that is only copied
	rtView    imageViewH // level 0 only, when the view above has more levels and this is a render target
	alloc     allocation
	external  bool // an image of a swapchain, which the swapchain owns
	destroyed bool
	// base is the mip level of the image that this texture starts at: 0, or the level of a
	// texture made by Level, which shares the image with its parent and owns only its view.
	base   int
	parent *texture
	levels []*texture // the textures made by Level, indexed by level; nil until asked for
}

var _ explicit.Texture = (*texture)(nil)

// Desc implements explicit.Texture.
func (t *texture) Desc() explicit.TextureDesc { return t.desc }

// Level implements explicit.Texture: a texture over one level of the image, with a view of
// its own that the parent releases.
func (t *texture) Level(i int) explicit.Texture {
	if i < 0 || i >= t.desc.MipLevels {
		panic(errors.New(tagVulkanLevelInvalid, "the texture has no such mip level", "level", i, "levels", t.desc.MipLevels))
	}
	if t.desc.MipLevels == 1 || t.parent != nil {
		return t
	}
	if t.levels == nil {
		t.levels = make([]*texture, t.desc.MipLevels)
	}
	if lv := t.levels[i]; lv != nil {
		return lv
	}
	d := t.d
	view, err := d.newViewRange(t.image, vkFormat(t.desc.Format), uint32(i), 1)
	if err != nil {
		panic(err)
	}
	desc := t.desc
	desc.Width, desc.Height, desc.MipLevels = max(desc.Width>>i, 1), max(desc.Height>>i, 1), 1
	lv := &texture{d: d, desc: desc, image: t.image, view: view, external: true, base: i, parent: t}
	t.levels[i] = lv
	return lv
}

// targetView is the view that a pass draws into.
func (t *texture) targetView() imageViewH {
	if t.rtView != 0 {
		return t.rtView
	}
	return t.view
}

// Destroy implements explicit.Texture.
func (t *texture) Destroy() {
	if t.destroyed || t.parent != nil {
		return
	}
	t.destroyed = true
	delete(t.d.live, t)
	t.d.retire(func() { t.release() })
}

func (t *texture) release() {
	d := t.d
	for _, lv := range t.levels {
		if lv != nil {
			d.f.destroyImageView(d.dev, lv.view, 0)
		}
	}
	if t.rtView != 0 {
		d.f.destroyImageView(d.dev, t.rtView, 0)
	}
	if t.view != 0 {
		d.f.destroyImageView(d.dev, t.view, 0)
	}
	if t.external {
		return
	}
	d.f.destroyImage(d.dev, t.image, 0)
	d.alloc.release(t.alloc)
}

// vkFormat maps a texture format.
func vkFormat(f explicit.TextureFormat) uint32 {
	switch f {
	case explicit.TextureFormatBGRA8Unorm:
		return formatBGRA8Unorm
	case explicit.TextureFormatR8Unorm:
		return formatR8Unorm
	}
	return formatRGBA8Unorm
}

// NewTexture implements explicit.Device. The image is made in the GENERAL layout, in
// which it is drawn into, sampled and copied from and to; the transition is recorded
// in the next frame, or in the one being recorded.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	d.limits.CheckTexture(desc)
	if desc.MipLevels < 1 {
		desc.MipLevels = 1
	}
	if desc.Width < 1 || desc.Height < 1 || desc.Format.BytesPerPixel() == 0 {
		return nil, errors.New(tagVulkanTextureInvalid, "a texture needs a size and a format", "width", desc.Width, "height", desc.Height, "format", int(desc.Format))
	}
	usage := uint32(imageUsageTransferDst)
	if desc.Usage&explicit.TextureSampled != 0 {
		usage |= imageUsageSampled
	}
	if desc.Usage&explicit.TextureRenderTarget != 0 {
		usage |= imageUsageColor | imageUsageTransferSrc
	}
	if desc.Usage&explicit.TextureCopySource != 0 {
		usage |= imageUsageTransferSrc
	}
	a := d.arena
	defer a.rewind(a.mark())
	format := vkFormat(desc.Format)
	info := put(a, imageCreateInfo{
		sType: sTypeImageCreateInfo, imageType: imageType2D, format: format,
		extent:    extent3D{uint32(desc.Width), uint32(desc.Height), 1},
		mipLevels: uint32(desc.MipLevels), arrayLayers: 1, samples: sampleCount1,
		tiling: imageTilingOptimal, usage: usage, sharingMode: sharingExclusive, initialLayout: layoutUndefined,
	})
	out := a.out()
	if err := check("vkCreateImage", d.f.createImage(d.dev, info, 0, out)); err != nil {
		return nil, err
	}
	t := &texture{d: d, desc: desc, image: u64(out)}
	req := a.alloc(unsafeSizeof[memoryRequirements](), 8)
	d.f.getImageMemoryRequirements(d.dev, t.image, req)
	al, err := d.alloc.alloc(*(*memoryRequirements)(dl.Pointer(req)), false, false)
	if err != nil {
		d.f.destroyImage(d.dev, t.image, 0)
		return nil, err
	}
	t.alloc = al
	fail := func(err error) (explicit.Texture, error) {
		if t.view != 0 {
			d.f.destroyImageView(d.dev, t.view, 0)
		}
		d.f.destroyImage(d.dev, t.image, 0)
		d.alloc.release(al)
		return nil, err
	}
	if err := check("vkBindImageMemory", d.f.bindImageMemory(d.dev, t.image, al.block.mem, al.offset)); err != nil {
		return fail(err)
	}
	// An image can have a view only if it is sampled or drawn into.
	if usage&(imageUsageSampled|imageUsageColor) != 0 {
		if t.view, err = d.newView(t.image, format, uint32(desc.MipLevels)); err != nil {
			return fail(err)
		}
	}
	if desc.Usage&explicit.TextureRenderTarget != 0 && desc.MipLevels > 1 {
		if t.rtView, err = d.newView(t.image, format, 1); err != nil {
			return fail(err)
		}
	}
	d.live[t] = func() { t.destroyed = true; t.release() }
	d.initLayout(t)
	return t, nil
}

// newView makes a view of the first levels of the image.
func (d *Device) newView(img imageH, format, levels uint32) (imageViewH, error) {
	return d.newViewRange(img, format, 0, levels)
}

// newViewRange makes a view of count levels of the image from level base.
func (d *Device) newViewRange(img imageH, format, base, count uint32) (imageViewH, error) {
	a := d.arena
	defer a.rewind(a.mark())
	info := put(a, imageViewCreateInfo{
		sType: sTypeImageViewCreateInfo, image: img, viewType: imageViewType2D, format: format,
		subresourceRange: imageSubresourceRange{aspectMask: imageAspectColor, baseMipLevel: base, levelCount: count, layerCount: 1},
	})
	out := a.out()
	if err := check("vkCreateImageView", d.f.createImageView(d.dev, info, 0, out)); err != nil {
		return 0, err
	}
	return u64(out), nil
}

// initLayout moves a new image from UNDEFINED to GENERAL: now when a frame is being
// recorded outside a pass, else at the start of the next frame.
func (d *Device) initLayout(t *texture) {
	d.inCommands(func(cb commandBufferH) {
		a := d.arena
		defer a.rewind(a.mark())
		barrier := put(a, imageMemoryBarrier2{
			sType: sTypeImageMemoryBarrier2, dstStageMask: stage2AllCommands, dstAccessMask: access2MemoryRead | access2MemoryWrite,
			oldLayout: layoutUndefined, newLayout: layoutGeneral,
			srcQueueFamilyIndex: queueFamilyIgnored, dstQueueFamilyIndex: queueFamilyIgnored,
			image:            t.image,
			subresourceRange: imageSubresourceRange{aspectMask: imageAspectColor, levelCount: uint32(t.desc.MipLevels), layerCount: 1},
		})
		dep := put(a, dependencyInfo{sType: sTypeDependencyInfo, imageMemoryBarrierCount: 1, pImageMemoryBarriers: barrier})
		d.f.cmdPipelineBarrier2(cb, dep)
	})
}

// inCommands runs fn with a command buffer: the frame being recorded when it is outside
// a pass, else the next frame's, at its start.
func (d *Device) inCommands(fn func(cb commandBufferH)) {
	if d.open != nil && d.open.pass == nil {
		fn(d.open.cb)
		return
	}
	d.pending = append(d.pending, fn)
}

// stagingRing is a host-visible buffer that textures are filled through: each frame slot
// has its own half to write into, which is free again when the slot is reused.
type stagingRing struct {
	buffer bufferH
	alloc  allocation
	ptr    uintptr
	heads  [framesInFlight]uint64
}

func (s *stagingRing) destroy(d *Device) {
	if s.buffer != 0 {
		d.destroyBuffer(s.buffer, s.alloc)
		s.buffer = 0
	}
}

// stage reserves n bytes in the staging memory of the slot that the next commands go to,
// and returns the buffer, the offset and the address to write the data at. Data larger
// than the slot's share gets a buffer of its own, released when the GPU is done.
func (d *Device) stage(n uint64) (bufferH, uint64, uintptr, error) {
	if d.staging.buffer == 0 {
		buf, al, p, err := d.newHostBuffer(stagingPerSlot*framesInFlight, bufferUsageTransferSrc)
		if err != nil {
			return 0, 0, 0, err
		}
		d.staging.buffer, d.staging.alloc, d.staging.ptr = buf, al, p
	}
	slot := d.next
	if d.open != nil {
		slot = d.open.slot
	} else if err := d.prepare(slot); err != nil {
		return 0, 0, 0, err
	}
	head := alignUp64(d.staging.heads[slot], 16)
	if head+n <= stagingPerSlot {
		d.staging.heads[slot] = head + n
		off := uint64(slot)*stagingPerSlot + head
		return d.staging.buffer, off, d.staging.ptr + uintptr(off), nil
	}
	buf, al, p, err := d.newHostBuffer(n, bufferUsageTransferSrc)
	if err != nil {
		return 0, 0, 0, err
	}
	d.retireWithNextFrame(func() { d.destroyBuffer(buf, al) })
	return buf, 0, p, nil
}

// WriteTexture implements explicit.Device: the rows are copied into the staging ring and
// copied into the image by a command of the next frame, or of the frame being recorded
// when it is between passes. A failure to stage the data panics: it is out of memory,
// which Device methods that return no error cannot report.
func (d *Device) WriteTexture(t explicit.Texture, level int, r image.Rectangle, data []byte, stride int) {
	vt, ok := t.(*texture)
	if !ok || vt.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "texture"))
	}
	if r.Empty() {
		return
	}
	bpp := vt.desc.Format.BytesPerPixel()
	rowBytes := r.Dx() * bpp
	buf, off, p, err := d.stage(uint64(rowBytes * r.Dy()))
	if err != nil {
		panic(err)
	}
	dst := dl.Bytes(p, rowBytes*r.Dy())
	for y := 0; y < r.Dy(); y++ {
		copy(dst[y*rowBytes:(y+1)*rowBytes], data[y*stride:y*stride+rowBytes])
	}
	d.inCommands(func(cb commandBufferH) {
		a := d.arena
		defer a.rewind(a.mark())
		region := put(a, bufferImageCopy{
			bufferOffset:     off,
			imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, mipLevel: uint32(vt.base + level), layerCount: 1},
			imageOffset:      offset3D{int32(r.Min.X), int32(r.Min.Y), 0},
			imageExtent:      extent3D{uint32(r.Dx()), uint32(r.Dy()), 1},
		})
		d.f.cmdCopyBufferToImage(cb, buf, vt.image, layoutGeneral, 1, region)
		d.barrierAll(cb)
	})
}

// sampler is a VkSampler.
type sampler struct {
	d         *Device
	sampler   samplerH
	destroyed bool
}

var _ explicit.Sampler = (*sampler)(nil)

// Destroy implements explicit.Sampler.
func (s *sampler) Destroy() {
	if s.destroyed {
		return
	}
	s.destroyed = true
	delete(s.d.live, s)
	s.d.retire(func() { s.d.f.destroySampler(s.d.dev, s.sampler, 0) })
}

func vkFilter(f explicit.FilterMode) uint32 {
	if f == explicit.FilterLinear {
		return filterLinear
	}
	return filterNearest
}

func vkAddress(m explicit.AddressMode) uint32 {
	switch m {
	case explicit.AddressRepeat:
		return addressRepeat
	case explicit.AddressMirror:
		return addressMirror
	}
	return addressClamp
}

// NewSampler implements explicit.Device.
func (d *Device) NewSampler(desc explicit.SamplerDesc) (explicit.Sampler, error) {
	a := d.arena
	defer a.rewind(a.mark())
	mip := uint32(mipmapNearest)
	maxLod := float32(0.25) // level 0 only
	if desc.MipmapFilter != explicit.FilterUnspecified {
		maxLod = 1000 // VK_LOD_CLAMP_NONE
		if desc.MipmapFilter == explicit.FilterLinear {
			mip = mipmapLinear
		}
	}
	info := put(a, samplerCreateInfo{
		sType: sTypeSamplerCreateInfo, magFilter: vkFilter(desc.MagFilter), minFilter: vkFilter(desc.MinFilter),
		mipmapMode: mip, addressModeU: vkAddress(desc.AddressU), addressModeV: vkAddress(desc.AddressV),
		addressModeW: vkAddress(desc.AddressU), maxLod: maxLod, borderColor: borderTransparentBlack,
	})
	out := a.out()
	if err := check("vkCreateSampler", d.f.createSampler(d.dev, info, 0, out)); err != nil {
		return nil, err
	}
	s := &sampler{d: d, sampler: u64(out)}
	d.live[s] = func() { s.destroyed = true; d.f.destroySampler(d.dev, s.sampler, 0) }
	return s, nil
}

// pipeline is a graphics pipeline with the layouts of its bind groups.
type pipeline struct {
	d         *Device
	desc      explicit.PipelineDesc
	pipeline  pipelineH
	layout    pipelineLayoutH
	setLayout []descriptorSetLayoutH
	bindings  map[[2]int]explicit.BindingKind
	destroyed bool
}

var _ explicit.Pipeline = (*pipeline)(nil)

// Desc implements explicit.Pipeline.
func (p *pipeline) Desc() explicit.PipelineDesc { return p.desc }

// Destroy implements explicit.Pipeline.
func (p *pipeline) Destroy() {
	if p.destroyed {
		return
	}
	p.destroyed = true
	delete(p.d.live, p)
	p.d.retire(func() { p.release() })
}

func (p *pipeline) release() {
	d := p.d
	d.f.destroyPipeline(d.dev, p.pipeline, 0)
	d.f.destroyPipelineLayout(d.dev, p.layout, 0)
	for _, l := range p.setLayout {
		d.f.destroyDescriptorSetLayout(d.dev, l, 0)
	}
}

func vkVertexFormat(f explicit.VertexFormat) uint32 {
	switch f {
	case explicit.VertexFloat32:
		return formatR32Sfloat
	case explicit.VertexFloat32x2:
		return formatRG32Sfloat
	case explicit.VertexFloat32x3:
		return formatRGB32Sfloat
	case explicit.VertexUnorm8x4:
		return formatRGBA8Unorm
	}
	return formatRGBA32Sfloat
}

func vkBlendFactor(f explicit.BlendFactor) uint32 {
	switch f {
	case explicit.BlendZero:
		return blendZero
	case explicit.BlendSrcAlpha:
		return blendSrcAlpha
	case explicit.BlendOneMinusSrcAlpha:
		return blendOneMinusSrcAlpha
	case explicit.BlendDstAlpha:
		return blendDstAlpha
	case explicit.BlendOneMinusDstAlpha:
		return blendOneMinusDstAlpha
	}
	return blendOne
}

// shaderModule returns the VkShaderModule of the shader's SPIR-V, made once.
func (d *Device) shaderModule(s *explicit.Shader) (shaderModuleH, error) {
	if m, ok := d.modules[s]; ok {
		return m, nil
	}
	a := d.arena
	defer a.rewind(a.mark())
	code := a.alloc(uintptr(len(s.SPIRV)), 8)
	copy(dl.Bytes(code, len(s.SPIRV)), s.SPIRV)
	info := put(a, shaderModuleCreateInfo{sType: sTypeShaderModuleCreateInfo, codeSize: uintptr(len(s.SPIRV)), pCode: code})
	out := a.out()
	if err := check("vkCreateShaderModule", d.f.createShaderModule(d.dev, info, 0, out)); err != nil {
		return 0, err
	}
	d.modules[s] = u64(out)
	return u64(out), nil
}

// NewPipeline implements explicit.Device: the descriptor set layouts come from the
// bindings of the two entry points, uniform buffers with dynamic offsets (SPEC §9.3).
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	d.limits.CheckPipeline(desc)
	if desc.Shader == nil {
		return nil, errors.New(tagVulkanPipelineInvalid, "a pipeline needs a shader")
	}
	vs, ok1 := desc.Shader.Entry(desc.VertexEntry)
	fs, ok2 := desc.Shader.Entry(desc.FragmentEntry)
	if !ok1 || !ok2 {
		return nil, errors.New(tagVulkanPipelineInvalid, "the shader lacks the entry point",
			"shader", desc.Shader.Name, "vertex", desc.VertexEntry, "fragment", desc.FragmentEntry)
	}
	module, err := d.shaderModule(desc.Shader)
	if err != nil {
		return nil, err
	}
	p := &pipeline{d: d, desc: desc, bindings: map[[2]int]explicit.BindingKind{}}
	fail := func(err error) (explicit.Pipeline, error) {
		p.release2()
		return nil, err
	}

	// The bindings of each group, with the stages that read them.
	type slot struct {
		kind  explicit.BindingKind
		stage uint32
	}
	groups := map[int]map[int]*slot{}
	maxGroup := -1
	for stage, entry := range map[uint32]explicit.EntryPoint{stageVertex: vs, stageFragment: fs} {
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
			p.bindings[[2]int{b.Group, b.Binding}] = b.Kind
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
			return fail(err)
		}
		p.setLayout = append(p.setLayout, u64(out))
	}
	layoutInfo := put(a, pipelineLayoutCreateInfo{
		sType: sTypePipelineLayoutCreateInfo, setLayoutCount: uint32(len(p.setLayout)), pSetLayouts: putSlice(a, p.setLayout),
	})
	out := a.out()
	if err := check("vkCreatePipelineLayout", d.f.createPipelineLayout(d.dev, layoutInfo, 0, out)); err != nil {
		return fail(err)
	}
	p.layout = u64(out)

	var bindingDescs []vertexInputBindingDescription
	var attrDescs []vertexInputAttributeDescription
	for i, vb := range desc.VertexBuffers {
		rate := uint32(vertexInputRateVertex)
		if vb.PerInstance {
			rate = vertexInputRateInstance
		}
		bindingDescs = append(bindingDescs, vertexInputBindingDescription{binding: uint32(i), stride: uint32(vb.Stride), inputRate: rate})
		for _, at := range vb.Attributes {
			attrDescs = append(attrDescs, vertexInputAttributeDescription{location: uint32(at.Location), binding: uint32(i), format: vkVertexFormat(at.Format), offset: uint32(at.Offset)})
		}
	}
	entryName := func(n string) uintptr { return a.cstring(n) }
	stages := putSlice(a, []pipelineShaderStageCreateInfo{
		{sType: sTypePipelineShaderStageCreateInfo, stage: stageVertex, module: module, pName: entryName(desc.VertexEntry)},
		{sType: sTypePipelineShaderStageCreateInfo, stage: stageFragment, module: module, pName: entryName(desc.FragmentEntry)},
	})
	vertexInput := put(a, pipelineVertexInputStateCreateInfo{
		sType:                         sTypePipelineVertexInputStateCreate,
		vertexBindingDescriptionCount: uint32(len(bindingDescs)), pVertexBindingDescriptions: putSlice(a, bindingDescs),
		vertexAttributeDescriptionCount: uint32(len(attrDescs)), pVertexAttributeDescriptions: putSlice(a, attrDescs),
	})
	inputAssembly := put(a, pipelineInputAssemblyStateCreateInfo{sType: sTypePipelineInputAssemblyStateCreate, topology: topologyTriangleList})
	viewportState := put(a, pipelineViewportStateCreateInfo{sType: sTypePipelineViewportStateCreate, viewportCount: 1, scissorCount: 1})
	raster := put(a, pipelineRasterizationStateCreateInfo{
		sType: sTypePipelineRasterizationStateCreate, polygonMode: polygonFill, cullMode: cullNone, frontFace: frontFaceCCW, lineWidth: 1,
	})
	multisample := put(a, pipelineMultisampleStateCreateInfo{sType: sTypePipelineMultisampleStateCreate, rasterizationSamples: sampleCount1})
	blend := pipelineColorBlendAttachmentState{colorWriteMask: colorComponentsAll, srcColorBlendFactor: blendOne, srcAlphaBlendFactor: blendOne}
	if desc.Blend.Enabled {
		blend.blendEnable = 1
		blend.srcColorBlendFactor, blend.srcAlphaBlendFactor = vkBlendFactor(desc.Blend.SrcFactor), vkBlendFactor(desc.Blend.SrcFactor)
		blend.dstColorBlendFactor, blend.dstAlphaBlendFactor = vkBlendFactor(desc.Blend.DstFactor), vkBlendFactor(desc.Blend.DstFactor)
		if desc.Blend.DstFactor == explicit.BlendFactorUnspecified {
			blend.dstColorBlendFactor, blend.dstAlphaBlendFactor = blendZero, blendZero
		}
	}
	colorBlend := put(a, pipelineColorBlendStateCreateInfo{
		sType: sTypePipelineColorBlendStateCreate, attachmentCount: 1, pAttachments: put(a, blend),
	})
	dynamic := put(a, pipelineDynamicStateCreateInfo{
		sType: sTypePipelineDynamicStateCreate, dynamicStateCount: 2, pDynamicStates: putSlice(a, []uint32{dynamicViewport, dynamicScissor}),
	})
	rendering := put(a, pipelineRenderingCreateInfo{
		sType: sTypePipelineRenderingCreateInfo, colorAttachmentCount: 1, pColorAttachmentFormats: put(a, vkFormat(desc.Target)),
	})
	info := put(a, graphicsPipelineCreateInfo{
		sType: sTypeGraphicsPipelineCreateInfo, pNext: rendering,
		stageCount: 2, pStages: stages, pVertexInputState: vertexInput, pInputAssemblyState: inputAssembly,
		pViewportState: viewportState, pRasterizationState: raster, pMultisampleState: multisample,
		pColorBlendState: colorBlend, pDynamicState: dynamic, layout: p.layout,
	})
	out = a.out()
	if err := check("vkCreateGraphicsPipelines", d.f.createGraphicsPipelines(d.dev, 0, 1, info, 0, out)); err != nil {
		return fail(err)
	}
	p.pipeline = u64(out)
	d.live[p] = func() { p.destroyed = true; p.release() }
	return p, nil
}

// release2 destroys what a failed NewPipeline made so far.
func (p *pipeline) release2() {
	d := p.d
	if p.layout != 0 {
		d.f.destroyPipelineLayout(d.dev, p.layout, 0)
	}
	for _, l := range p.setLayout {
		d.f.destroyDescriptorSetLayout(d.dev, l, 0)
	}
}

func descriptorType(k explicit.BindingKind) uint32 {
	switch k {
	case explicit.BindingSampledTexture:
		return descriptorSampledImage
	case explicit.BindingSampler:
		return descriptorSampler
	}
	return descriptorUniformDynamic
}

// bindGroup is a descriptor set from the pool of a frame: it is valid until that frame's
// slot is reused.
type bindGroup struct {
	d   *Device
	set descriptorSetH
}

var _ explicit.BindGroup = (*bindGroup)(nil)

// Destroy implements explicit.BindGroup: the descriptor pool of the frame is reset as a
// whole, so there is nothing to do.
func (b *bindGroup) Destroy() {}

// NewBindGroup implements explicit.Device: the set is made from the descriptor pool of
// the frame that is being recorded, or of the next one.
func (d *Device) NewBindGroup(p explicit.Pipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	vp, ok := p.(*pipeline)
	if !ok || vp.d != d {
		panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "pipeline"))
	}
	if group < 0 || group >= len(vp.setLayout) {
		return nil, errors.New(tagVulkanBindGroupInvalid, "the pipeline has no such bind group", "group", group, "groups", len(vp.setLayout))
	}
	slot := d.next
	if d.open != nil {
		slot = d.open.slot
	} else if err := d.prepare(slot); err != nil {
		return nil, err
	}
	s := &d.slots[slot]
	a := d.arena
	defer a.rewind(a.mark())
	set, err := d.allocateSet(s, vp.setLayout[group])
	if err != nil {
		return nil, err
	}
	writes := make([]writeDescriptorSet, 0, len(entries))
	for _, e := range entries {
		kind, ok := vp.bindings[[2]int{group, e.Binding}]
		if !ok {
			return nil, errors.New(tagVulkanBindGroupInvalid, "the shader has no such binding", "group", group, "binding", e.Binding)
		}
		w := writeDescriptorSet{sType: sTypeWriteDescriptorSet, dstSet: set, dstBinding: uint32(e.Binding), descriptorCount: 1, descriptorType: descriptorType(kind)}
		switch kind {
		case explicit.BindingUniformBuffer:
			b, ok := e.Buffer.(*buffer)
			if !ok || b.d != d {
				panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "buffer"))
			}
			size := uint64(e.Size)
			if size == 0 {
				size = uint64(b.size - e.Offset)
			}
			w.pBufferInfo = put(a, descriptorBufferInfo{buffer: b.buffer, offset: uint64(e.Offset), rng: size})
		case explicit.BindingSampledTexture:
			t, ok := e.Texture.(*texture)
			if !ok || t.d != d {
				panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "texture"))
			}
			if t.view == 0 {
				return nil, errors.New(tagVulkanBindGroupInvalid, "the texture is not TextureSampled", "binding", e.Binding)
			}
			w.pImageInfo = put(a, descriptorImageInfo{imageView: t.view, imageLayout: layoutGeneral})
		case explicit.BindingSampler:
			sm, ok := e.Sampler.(*sampler)
			if !ok || sm.d != d {
				panic(errors.New(tagVulkanForeignResource, "the resource was not made by this Vulkan device", "resource", "sampler"))
			}
			w.pImageInfo = put(a, descriptorImageInfo{sampler: sm.sampler})
		}
		writes = append(writes, w)
	}
	if len(writes) > 0 {
		d.f.updateDescriptorSets(d.dev, uint32(len(writes)), putSlice(a, writes), 0, 0)
	}
	return &bindGroup{d: d, set: set}, nil
}

// allocateSet makes a descriptor set from the current pool of the slot, adding a pool
// when it is full.
func (d *Device) allocateSet(s *frameSlot, layout descriptorSetLayoutH) (descriptorSetH, error) {
	a := d.arena
	defer a.rewind(a.mark())
	for attempt := 0; ; attempt++ {
		info := put(a, descriptorSetAllocateInfo{
			sType: sTypeDescriptorSetAllocateInfo, descriptorPool: s.pools[s.poolIdx], descriptorSetCount: 1,
			pSetLayouts: putSlice(a, []descriptorSetLayoutH{layout}),
		})
		out := a.out()
		r := d.f.allocateDescriptorSets(d.dev, info, out)
		if r == resultSuccess {
			return u64(out), nil
		}
		if (r != resultErrorOutOfPool && r != resultErrorFragmentedPool) || attempt > 0 {
			return 0, vkError("vkAllocateDescriptorSets", r)
		}
		if s.poolIdx+1 < len(s.pools) {
			s.poolIdx++
		} else if err := d.addDescriptorPool(s); err != nil {
			return 0, err
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagVulkanBindGroupInvalid = "VulkanBindGroupInvalid"
	tagVulkanLevelInvalid     = "VulkanLevelInvalid"
	tagVulkanPipelineInvalid  = "VulkanPipelineInvalid"
	tagVulkanTextureInvalid   = "VulkanTextureInvalid"
	tagVulkanWriteOutOfRange  = "VulkanWriteOutOfRange"
)
