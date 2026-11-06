//go:build windows

package d3d12

import (
	"strconv"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// bindGroup is the resources of one group of a pipeline (SPEC §9.8 "Descriptors", "Binding model"):
// the GPU addresses of its uniform buffers, which SetBindGroup adds the dynamic offsets to, and the
// descriptors of its textures and its samplers, copied into the shader-visible heaps of the frame slot
// it was made for. It is valid until that slot is reused, as the Vulkan descriptor sets of a frame
// are.
type bindGroup struct {
	d     *Device
	slot  int
	gen   uint64 // the slot's generation when the group was made
	group int
	shape string // groupPlan.Shape: the pipelines this group can be used with

	cbv      []uint64   // the address of each uniform buffer, in binding order
	textures []*texture // the textures of the table, in order; nil for a null descriptor
	srvSrc   []uintptr  // the CPU handles the table is copied from; 0 for a null descriptor
	smpSrc   []uintptr  // the same for the samplers; 0 for the default sampler

	// The storage resources (step 30.2): the address of each read-only and each writable storage buffer, in
	// binding order, and the storage textures with the CPU handles of their unordered access views. The
	// descriptors of the storage textures follow those of the sampled textures in the heap, in one table.
	roAddr, rwAddr []uint64
	uavTex         []*texture
	uavSrc         []uintptr
	// uses are the buffers of the group with the state that SetBindGroup puts them in.
	uses []bufferUse

	// vBlock, vOffset and sBlock, sOffset are where the tables are in the slot's heaps (-1 for none).
	vBlock, vOffset, sBlock, sOffset int
}

// bufferUse is a buffer a bind group names and the state its binding needs.
type bufferUse struct {
	b     *buffer
	state resState
}

var _ explicit.BindGroup = (*bindGroup)(nil)

// Destroy implements explicit.BindGroup: the heaps of the slot are reused as a whole, so there is
// nothing to do.
func (b *bindGroup) Destroy() {}

// hasTables says whether the group has a descriptor table.
func (b *bindGroup) hasTables() bool {
	return len(b.srvSrc) > 0 || len(b.uavSrc) > 0 || len(b.smpSrc) > 0
}

// NewBindGroup implements explicit.Device: the descriptors of the textures and the samplers are copied
// (CopyDescriptorsSimple) into the next block of the shader-visible heaps of the frame that is being
// recorded, or of the next one. A slot of a table that the entries leave out holds a null texture or
// the default sampler (resource binding tier 1 needs every descriptor valid, SPEC §9.8). It fails with
// D3D12BindGroupInvalid for a group the pipeline lacks, a binding the shader lacks, an entry without
// its resource, a texture that is not TextureSampled, an uniform buffer address that is not a multiple
// of 256, or a uniform binding without an entry.
func (d *Device) NewBindGroup(p explicit.Pipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	pl, ok := p.(*pipeline)
	if !ok || pl.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "pipeline"))
	}
	return d.makeBindGroup(pl.root, pipelineBindings(pl.desc.Shader, pl.desc.FragmentEntry, pl.desc.VertexEntry), group, entries)
}

// pipelineBindings finds the declaration of a binding in the named entry points of a shader.
func pipelineBindings(sh *explicit.Shader, entries ...string) func(group, binding int) (explicit.ShaderBinding, bool) {
	return func(group, binding int) (explicit.ShaderBinding, bool) {
		for _, name := range entries {
			entry, ok := sh.Entry(name)
			if !ok {
				continue
			}
			for _, b := range entry.Bindings {
				if b.Group == group && b.Binding == binding {
					return b, true
				}
			}
		}
		return explicit.ShaderBinding{}, false
	}
}

// makeBindGroup makes the group of a pipeline, a graphics or a compute pipeline, from its root signature
// and the declarations of the bindings in its entry points.
func (d *Device) makeBindGroup(root *rootSig, declared func(group, binding int) (explicit.ShaderBinding, bool), group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	plan := root.plan
	if group < 0 || group >= len(plan.Groups) {
		return nil, errors.New(tagD3D12BindGroupInvalid, "the pipeline has no such bind group", "group", group, "groups", len(plan.Groups))
	}
	gp := plan.Groups[group]
	byBinding := make(map[int]explicit.Binding, len(entries))
	for _, e := range entries {
		if !containsBinding(gp, e.Binding) {
			return nil, errors.New(tagD3D12BindGroupInvalid, "the shader has no such binding", "group", group, "binding", e.Binding)
		}
		// An entry without its resource is an error that this device returns (D3D12BindGroupInvalid, below),
		// as it always did; the other breaks of a rule are the debug panics of CheckBinding.
		if sb, ok := declared(group, e.Binding); ok && (e.Buffer != nil || e.Texture != nil || e.Sampler != nil) {
			d.limits.CheckBinding(sb, e)
		}
		byBinding[e.Binding] = e
	}
	bg := &bindGroup{d: d, group: group, shape: gp.Shape, vBlock: -1, sBlock: -1}
	ownBuffer := func(e explicit.Binding) *buffer {
		b, ok := e.Buffer.(*buffer)
		if !ok || b.d != d {
			panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
		}
		return b
	}
	for _, n := range gp.Uniforms {
		e, ok := byBinding[n]
		if !ok || e.Buffer == nil {
			return nil, errors.New(tagD3D12BindGroupInvalid, "a uniform binding has no buffer", "group", group, "binding", n)
		}
		b := ownBuffer(e)
		if e.Offset < 0 || e.Offset > b.size || e.Offset%d.limits.UniformOffsetAlignment != 0 {
			return nil, errors.New(tagD3D12BindGroupInvalid, "the offset of a uniform binding must be inside the buffer and a multiple of 256",
				"group", group, "binding", n, "offset", e.Offset, "size", b.size)
		}
		bg.cbv = append(bg.cbv, b.va+uint64(e.Offset))
		bg.uses = append(bg.uses, bufferUse{b, stateVertexConstant})
	}
	storage := func(list []int, state resState) ([]uint64, error) {
		var addrs []uint64
		for _, n := range list {
			e, ok := byBinding[n]
			if !ok || e.Buffer == nil {
				return nil, errors.New(tagD3D12BindGroupInvalid, "a storage binding has no buffer", "group", group, "binding", n)
			}
			b := ownBuffer(e)
			if e.Offset < 0 || e.Offset >= max(b.size, 1) || e.Offset%4 != 0 {
				return nil, errors.New(tagD3D12BindGroupInvalid, "the offset of a storage binding must be inside the buffer and a multiple of 4",
					"group", group, "binding", n, "offset", e.Offset, "size", b.size)
			}
			addrs = append(addrs, b.va+uint64(e.Offset))
			bg.uses = append(bg.uses, bufferUse{b, state})
		}
		return addrs, nil
	}
	var err error
	if bg.roAddr, err = storage(gp.StorageRO, stateShaderResource); err != nil {
		return nil, err
	}
	if bg.rwAddr, err = storage(gp.StorageRW, stateUnorderedAccess); err != nil {
		return nil, err
	}
	for _, n := range gp.Textures {
		var src uintptr
		var tex *texture
		if e, ok := byBinding[n]; ok {
			if e.Texture == nil {
				return nil, errors.New(tagD3D12BindGroupInvalid, "a texture binding has no texture", "group", group, "binding", n)
			}
			tex = d.own2texture(e.Texture)
			if tex.srv < 0 {
				return nil, errors.New(tagD3D12BindGroupInvalid, "the texture is not TextureSampled, or is a layer of an array or a cube, which Direct3D 12 cannot sample as a 2D texture",
					"group", group, "binding", n)
			}
			if sb, ok := declared(group, n); ok && (tex.desc.Kind.Dimension() != sb.Dimension || !tex.desc.Format.SampleType(d.features).Fits(sb.Sample)) {
				return nil, errors.New(tagD3D12TextureBindingMismatch, "the texture is not the dimension or sample type that the shader declares",
					"group", group, "binding", n, "kind", int(tex.desc.Kind), "dimension", int(sb.Dimension), "format", int(tex.desc.Format), "sample", int(sb.Sample))
			}
			src = d.cpu[1].handle(tex.srv)
		}
		bg.textures = append(bg.textures, tex)
		bg.srvSrc = append(bg.srvSrc, src)
	}
	for _, n := range gp.StorageTextures {
		e, ok := byBinding[n]
		if !ok || e.Texture == nil {
			return nil, errors.New(tagD3D12BindGroupInvalid, "a storage texture binding has no texture", "group", group, "binding", n)
		}
		tex := d.own2texture(e.Texture)
		if sb, ok := declared(group, n); ok && tex.desc.Kind.Dimension() != sb.Dimension {
			return nil, errors.New(tagD3D12TextureBindingMismatch, "the texture is not the dimension that the shader declares",
				"group", group, "binding", n, "kind", int(tex.desc.Kind), "dimension", int(sb.Dimension))
		}
		// A storage image is one mip level: the texture's own view has the first, so a texture with levels
		// is bound at level 0 whatever view of it was named.
		if tex.desc.MipLevels > 1 && tex.root == tex {
			tex = tex.Level(0).(*texture)
		}
		if tex.uav < 0 {
			return nil, errors.New(tagD3D12BindGroupInvalid, "the texture is not TextureStorage, or is a layer of an array or a cube, which Direct3D 12 cannot bind as a storage texture",
				"group", group, "binding", n)
		}
		bg.uavTex = append(bg.uavTex, tex)
		bg.uavSrc = append(bg.uavSrc, d.cpu[1].handle(tex.uav))
	}
	for _, n := range gp.Samplers {
		var src uintptr
		if e, ok := byBinding[n]; ok {
			if e.Sampler == nil {
				return nil, errors.New(tagD3D12BindGroupInvalid, "a sampler binding has no sampler", "group", group, "binding", n)
			}
			sm, ok := e.Sampler.(*sampler)
			if !ok || sm.d != d {
				panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "sampler"))
			}
			src = d.cpu[2].handle(sm.n)
		}
		bg.smpSrc = append(bg.smpSrc, src)
	}
	bg.slot = d.next
	if d.open != nil {
		bg.slot = d.open.slot
	} else if err := d.prepare(bg.slot); err != nil {
		return nil, err
	}
	bg.gen = d.slots[bg.slot].gen
	if err := bg.place(); err != nil {
		return nil, err
	}
	return bg, nil
}

// containsBinding says whether the group has a binding number, of any kind.
func containsBinding(gp groupPlan, n int) bool {
	for _, list := range [][]int{gp.Uniforms, gp.Textures, gp.Samplers, gp.StorageRO, gp.StorageRW, gp.StorageTextures} {
		for _, b := range list {
			if b == n {
				return true
			}
		}
	}
	return false
}

// place copies the descriptors of the group into the latest blocks of the slot's heaps: the first
// time when the group is made, and again when the encoder has moved on to a later block and the group,
// bound again, has to be in it (settleBlocks). A table that is in the latest block already (the same
// descriptors, for another group) is shared, so a frame takes as many descriptors as it has different
// tables.
func (b *bindGroup) place() error {
	d := b.d
	s := &d.slots[b.slot]
	// The table of the sampled textures and the table of the storage textures are one run of the heap,
	// the first followed by the second, so that both are in the same block.
	if n := len(b.srvSrc) + len(b.uavSrc); n > 0 {
		key := tableKey(append(append([]uintptr(nil), b.srvSrc...), b.uavSrc...)) + "/" + strconv.Itoa(len(b.srvSrc))
		block, off, ok := s.views.alloc.lookup(key)
		if !ok {
			var err error
			if block, off, err = s.views.take(d, n); err != nil {
				return err
			}
			for i, src := range b.srvSrc {
				dst := s.views.cpuHandle(block, off+i)
				if src == 0 {
					d.createNullView(dst)
				} else {
					d.dev.Call(deviceCopyDescriptorsSimple, 1, dst, src, uintptr(descriptorHeapSRV))
				}
			}
			for i, src := range b.uavSrc {
				d.dev.Call(deviceCopyDescriptorsSimple, 1, s.views.cpuHandle(block, off+len(b.srvSrc)+i), src, uintptr(descriptorHeapSRV))
			}
			s.views.alloc.remember(key, block, off)
		}
		b.vBlock, b.vOffset = block, off
	}
	if n := len(b.smpSrc); n > 0 {
		key := tableKey(b.smpSrc)
		block, off, ok := s.samplers.alloc.lookup(key)
		if !ok {
			var err error
			if block, off, err = s.samplers.take(d, n); err != nil {
				return err
			}
			for i, src := range b.smpSrc {
				dst := s.samplers.cpuHandle(block, off+i)
				if src == 0 {
					sd := samplerDesc(explicit.SamplerDesc{})
					d.dev.Call(deviceCreateSampler, uintptr(unsafe.Pointer(&sd)), dst)
				} else {
					d.dev.Call(deviceCopyDescriptorsSimple, 1, dst, src, uintptr(descriptorHeapSampler))
				}
			}
			s.samplers.alloc.remember(key, block, off)
		}
		b.sBlock, b.sOffset = block, off
	}
	return nil
}

// createNullView makes a null shader resource view at the CPU handle: a descriptor that is valid for
// a table of resource binding tier 1 and reads as zero.
func (d *Device) createNullView(dst uintptr) {
	srv := d3d12ShaderResourceViewDesc{
		Format: dxgiFormatRGBA8, ViewDimension: srvDimensionTexture2D, Shader4ComponentMapping: defaultComponentMapping,
		Texture2D: d3d12Tex2DSRV{MipLevels: 1},
	}
	d.dev.Call(deviceCreateShaderResourceView, 0, uintptr(unsafe.Pointer(&srv)), dst)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagD3D12BindGroupInvalid is the tag of the error for a bind group that cannot be made from its
	// entries, and of the programmer error of binding a group to a pipeline it does not fit, with
	// the wrong number of dynamic offsets, or after its frame; the metadata name what is wrong.
	tagD3D12BindGroupInvalid = "D3D12BindGroupInvalid"
	// tagD3D12TextureBindingMismatch is the tag of the error for a texture bound where the shader declares
	// another dimension or sample type (a cube where it reads a 2D texture, a float32 texture without
	// filtering where it filters); the metadata name the binding and the kinds.
	tagD3D12TextureBindingMismatch = "D3D12TextureBindingMismatch"
)
