//go:build windows

package d3d12

import (
	"image"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// buffer is a committed buffer (SPEC §9.8 "Memory"). One that only the CPU writes and the GPU reads is in an
// upload heap, mapped for good and written at once, in the state GENERIC_READ. One that shaders write, or
// that copies and queries write (BufferStorage, BufferCopyDestination, BufferQueryResolve), is in the
// default heap: the CPU cannot map it, WriteBuffer goes through the staging ring, and the device tracks
// its state like a texture's (gpu).
type buffer struct {
	d         *Device
	res       win32.Object
	ptr       uintptr // the mapping, 0 for a buffer in the default heap
	va        uint64  // the GPU virtual address, which a root constant buffer view names
	size      int
	usage     explicit.BufferUsage
	gpu       bool // in the default heap, with states
	destroyed bool
}

var _ explicit.Buffer = (*buffer)(nil)

// Size implements explicit.Buffer.
func (b *buffer) Size() int { return b.size }

// Usage implements explicit.Buffer.
func (b *buffer) Usage() explicit.BufferUsage { return b.usage }

// Destroy implements explicit.Buffer: the buffer is released when the GPU is done with the frames that
// may use it.
func (b *buffer) Destroy() {
	if b.destroyed {
		return
	}
	b.destroyed = true
	d := b.d
	delete(d.live, b)
	d.counts.buffers--
	d.retire(func() {
		if b.gpu {
			d.btracker.forget(b)
		}
		d.releaseBuffer(b.res, b.allocated())
	})
}

// allocated is the size of the resource: a buffer is never smaller than 4 bytes.
func (b *buffer) allocated() int { return max(b.size, 4) }

// inGPUHeap says whether a buffer with the usage is in the default heap: the GPU writes it.
func inGPUHeap(usage explicit.BufferUsage) bool {
	return usage&(explicit.BufferStorage|explicit.BufferCopyDestination|explicit.BufferQueryResolve) != 0
}

// NewBuffer implements explicit.Device. A buffer that only the CPU writes is in an upload heap: the
// uniform ring and the vertex and index data are written by the CPU at once (SPEC §9.8); one that the GPU
// writes is in the default heap.
func (d *Device) NewBuffer(size int, usage explicit.BufferUsage) (explicit.Buffer, error) {
	d.limits.CheckBuffer(size, usage)
	n := max(size, 4)
	if inGPUHeap(usage) {
		return d.newGPUBuffer(n, size, usage)
	}
	res, err := d.createBuffer(heapTypeUpload, n)
	if err != nil {
		return nil, err
	}
	ptr, err := d.mapBuffer(res, false, n)
	if err != nil {
		d.releaseBuffer(res, n)
		return nil, err
	}
	b := &buffer{d: d, res: res, ptr: ptr, va: uint64(res.Call(resourceGPUAddress)), size: size, usage: usage}
	d.counts.buffers++
	d.live[b] = func() { b.destroyed = true; d.counts.buffers--; d.releaseBuffer(b.res, b.allocated()) }
	return b, nil
}

// newGPUBuffer makes a buffer in the default heap, tracked in the state COMMON.
func (d *Device) newGPUBuffer(n, size int, usage explicit.BufferUsage) (*buffer, error) {
	res, err := d.createGPUBuffer(n, usage&explicit.BufferStorage != 0)
	if err != nil {
		return nil, err
	}
	b := &buffer{d: d, res: res, va: uint64(res.Call(resourceGPUAddress)), size: size, usage: usage, gpu: true}
	d.btracker.add(b, 1)
	d.counts.buffers++
	d.live[b] = func() {
		b.destroyed = true
		d.counts.buffers--
		d.btracker.forget(b)
		d.releaseBuffer(b.res, b.allocated())
	}
	return b, nil
}

// WriteBuffer implements explicit.Device: the memory of a buffer in an upload heap is mapped, so the data
// is copied at once; the upload heap is write-combined, so the data is only written, never read back. A
// buffer in the default heap gets the data through the staging ring, by a command of the next frame (or of
// the frame being recorded, between passes), as a texture does.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	db, ok := b.(*buffer)
	if !ok || db.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	if offset < 0 || offset+len(data) > db.size {
		panic(errors.New(tagD3D12WriteOutOfRange, "the write is outside the buffer", "offset", offset, "bytes", len(data), "size", db.size))
	}
	if !db.gpu {
		copy(win32.Bytes(db.ptr+uintptr(offset), len(data)), data)
		return
	}
	if len(data) == 0 {
		return
	}
	res, off, p, err := d.stage(len(data))
	if err != nil {
		panic(err)
	}
	copy(win32.Bytes(p, len(data)), data)
	d.inCommands(func(list win32.Object) {
		d.opBegin()
		defer d.opEnd()
		if err := d.needBuffer(db, stateCopyDest); err != nil {
			panic(err)
		}
		d.barrier(list)
		list.Call(listCopyBufferRegion, db.res.Ptr(), uintptr(offset), res.Ptr(), uintptr(off), uintptr(len(data)))
	})
}

// texture is a texture in a committed resource in the default heap. A texture made by Level or Layer
// shares the resource with its parent and owns only its descriptors.
type texture struct {
	d    *Device
	desc explicit.TextureDesc // normalized
	res  win32.Object
	// root is the texture that owns the resource: itself, or the parent of a level or a layer. The state
	// tracker follows the root's subresources.
	root *texture
	// base is the first mip level of the resource that this texture starts at: 0, or the level of a
	// texture made by Level or Layer; baseLayer is the first array layer, 0 or the layer of a texture
	// made by Layer.
	base, baseLayer int
	levels          []*texture          // the textures made by Level, indexed by level; nil until asked for
	layers          map[[2]int]*texture // the textures made by Layer, by level and layer
	// srv, rtv and dsv are the numbers of the CPU descriptors of the views, -1 when there is none: the
	// shader resource view of the levels and layers the texture covers, and the render target view or the
	// depth-stencil view of its first level and layer.
	srv, rtv, dsv int
	// uav is the number of the unordered access view of the texture's first level (and all its layers or
	// slices) in the same heap as the shader resource views, -1 when it is not TextureStorage or is a view
	// that cannot be bound as a storage texture (a layer of an array, a cube).
	uav       int
	destroyed bool
}

var _ explicit.Texture = (*texture)(nil)

// Desc implements explicit.Texture.
func (t *texture) Desc() explicit.TextureDesc { return t.desc }

// mips is the number of mip levels of the resource.
func (t *texture) mips() int { return t.root.desc.MipLevels }

// sub is the index of the first subresource the texture covers: Direct3D 12 numbers the subresources
// mip + layer × mips (and the stencil plane after all the depth ones).
func (t *texture) sub() int { return t.base + t.baseLayer*t.mips() }

// subresourceCount is the number of subresources of the resource, one plane: a mip level of each layer.
func (t *texture) subresourceCount() int { return t.desc.MipLevels * t.desc.Layers() }

// Level implements explicit.Texture: a texture over one level of the resource, with views of its own
// that the parent releases. A 2D array, cube or 3D texture keeps its kind and layers.
func (t *texture) Level(i int) explicit.Texture {
	if i < 0 || i >= t.desc.MipLevels {
		panic(errors.New(tagD3D12LevelInvalid, "the texture has no such mip level", "level", i, "levels", t.desc.MipLevels))
	}
	if t.desc.MipLevels == 1 || t.root != t {
		return t
	}
	if t.levels == nil {
		t.levels = make([]*texture, t.desc.MipLevels)
	}
	if lv := t.levels[i]; lv != nil {
		return lv
	}
	desc := t.desc
	desc.Width, desc.Height, desc.MipLevels = max(desc.Width>>i, 1), max(desc.Height>>i, 1), 1
	if desc.Kind == explicit.TextureKind3D {
		desc.DepthOrLayers = max(desc.DepthOrLayers>>i, 1)
	}
	lv := &texture{d: t.d, desc: desc, res: t.res, root: t, base: i, srv: -1, rtv: -1, dsv: -1, uav: -1}
	if err := t.d.createViews(lv); err != nil {
		panic(err)
	}
	t.levels[i] = lv
	return lv
}

// Layer implements explicit.Texture: a 2D texture over one layer of one level. It can be drawn into
// and copied from; it cannot be sampled, because a Direct3D 12 view of one slice of an array is an
// array view, which a shader declares as an array (SPEC §12.10).
func (t *texture) Layer(level, layer int) explicit.Texture {
	t.d.limits.CheckLayer(t.desc, level, layer)
	if t.desc.Kind == explicit.TextureKind2D {
		return t.Level(level)
	}
	if t.root != t {
		panic(errors.New(tagD3D12LevelInvalid, "Layer is called on the texture, not on a view of a level", "level", level, "layer", layer))
	}
	key := [2]int{level, layer}
	if lv := t.layers[key]; lv != nil {
		return lv
	}
	desc := t.desc
	desc.Kind, desc.DepthOrLayers, desc.MipLevels = explicit.TextureKind2D, 1, 1
	desc.Width, desc.Height = max(desc.Width>>level, 1), max(desc.Height>>level, 1)
	lv := &texture{d: t.d, desc: desc, res: t.res, root: t, base: level, baseLayer: layer, srv: -1, rtv: -1, dsv: -1, uav: -1}
	if err := t.d.createViews(lv); err != nil {
		panic(err)
	}
	if t.layers == nil {
		t.layers = map[[2]int]*texture{}
	}
	t.layers[key] = lv
	return lv
}

// Destroy implements explicit.Texture.
func (t *texture) Destroy() {
	if t.destroyed || t.root != t {
		return
	}
	t.destroyed = true
	d := t.d
	delete(d.live, t)
	d.counts.textures--
	d.retire(func() { t.release() })
}

// release frees the descriptors, forgets the state and releases the resource.
func (t *texture) release() {
	d := t.d
	for _, lv := range t.levels {
		if lv != nil {
			d.freeViews(lv)
		}
	}
	for _, lv := range t.layers {
		d.freeViews(lv)
	}
	d.freeViews(t)
	d.tracker.forget(t)
	d.counts.bytes -= textureBytes(t.desc)
	d.disown(t.res)
}

// textureBytes is the size of the texture with all its levels, layers and samples, for the accounting.
func textureBytes(desc explicit.TextureDesc) int64 {
	desc = desc.Normalized()
	var n int64
	for i := 0; i < desc.MipLevels; i++ {
		slices := int64(desc.Layers())
		if desc.Kind == explicit.TextureKind3D {
			slices = int64(max(desc.DepthOrLayers>>i, 1))
		}
		n += int64(max(desc.Width>>i, 1)) * int64(max(desc.Height>>i, 1)) * slices * int64(desc.Format.BytesPerPixel()) * int64(desc.SampleCount)
	}
	return n
}

// NewTexture implements explicit.Device: a committed texture in the default heap, in the state
// COMMON, with the descriptors of its views (SPEC §9.8). It fails with D3D12TextureInvalid for a
// description that is not a texture (a programmer error, SPEC §9.3), GPUOutOfMemory, and the other
// tags of check.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	d.limits.CheckTexture(desc)
	desc = desc.Normalized()
	if desc.Width < 1 || desc.Height < 1 || desc.DepthOrLayers < 1 || desc.Format.BytesPerPixel() == 0 || desc.MipLevels > explicit.MipLevelsOf(desc.Width, desc.Height) ||
		desc.Usage&^desc.Format.AllowedUsage() != 0 || desc.Kind < explicit.TextureKind2D || desc.Kind > explicit.TextureKind3D {
		return nil, errors.New(tagD3D12TextureInvalid, "a texture needs a size, a format that allows its usage, a kind and a number of levels that fits its size",
			"width", desc.Width, "height", desc.Height, "format", int(desc.Format), "levels", desc.MipLevels, "kind", int(desc.Kind), "usage", int(desc.Usage))
	}
	hp := heapProps(heapTypeDefault)
	rd := d3d12ResourceDesc{
		Dimension: resourceDimensionTexture2D, Width: uint64(desc.Width), Height: uint32(desc.Height), DepthOrArraySize: uint16(desc.Layers()),
		MipLevels: uint16(desc.MipLevels), Format: dxgiFormat(desc.Format), SampleDesc: dxgiSampleDesc{Count: uint32(desc.SampleCount)}, Layout: textureLayoutUnknown,
	}
	if desc.Kind == explicit.TextureKind3D {
		rd.Dimension, rd.DepthOrArraySize = resourceDimensionTexture3D, uint16(desc.DepthOrLayers)
	}
	if desc.Usage&explicit.TextureRenderTarget != 0 {
		if desc.Format.IsDepthStencil() {
			rd.Flags = resourceFlagAllowDepth | resourceFlagDenyShaderResource
		} else {
			rd.Flags = resourceFlagAllowRT
		}
	}
	if desc.Usage&explicit.TextureStorage != 0 {
		rd.Flags |= resourceFlagAllowUAV
	}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateCommittedResource", deviceCreateCommittedResource,
		uintptr(unsafe.Pointer(&hp)), 0, uintptr(unsafe.Pointer(&rd)), uintptr(resourceStateCommon), 0,
		uintptr(unsafe.Pointer(&iidResource)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	t := &texture{d: d, desc: desc, res: d.own(win32.ObjectFromPtr(out)), srv: -1, rtv: -1, dsv: -1, uav: -1}
	t.root = t
	d.tracker.add(t, t.subresourceCount())
	d.counts.bytes += textureBytes(desc)
	if err := d.createViews(t); err != nil {
		t.release()
		return nil, err
	}
	d.counts.textures++
	d.live[t] = func() { t.destroyed = true; d.counts.textures--; t.release() }
	return t, nil
}

// createViews makes the descriptors of a texture: the shader resource view when it is sampled, over
// the levels and layers it covers, and the render target view or the depth-stencil view of its first
// level and layer when it is drawn into. A texture that is a layer of an array has no shader resource
// view (see Layer), and one that is an array, a cube or a volume has no view to draw into: a pass draws
// into a 2D texture (CheckPass).
func (d *Device) createViews(t *texture) error {
	format := dxgiFormat(t.desc.Format)
	layered := t.root.desc.Layers() > 1 // the resource is an array of layers: its 2D views are array views
	if t.desc.Usage&explicit.TextureSampled != 0 && (!layered || t.desc.Kind != explicit.TextureKind2D) {
		n, handle, err := d.cpu[1].alloc(d)
		if err != nil {
			return err
		}
		t.srv = n
		mostDetailed, levels := uint32(t.base), uint32(t.desc.MipLevels)
		switch t.desc.Kind {
		case explicit.TextureKind2DArray:
			srv := d3d12ShaderResourceViewDescArray{
				Format: format, ViewDimension: srvDimensionTexture2DArray, Shader4ComponentMapping: defaultComponentMapping,
				Texture2DArray: d3d12Tex2DArraySRV{MostDetailedMip: mostDetailed, MipLevels: levels, ArraySize: uint32(t.desc.Layers())},
			}
			d.dev.Call(deviceCreateShaderResourceView, t.res.Ptr(), uintptr(unsafe.Pointer(&srv)), handle)
		case explicit.TextureKindCube:
			srv := d3d12ShaderResourceViewDescCube{
				Format: format, ViewDimension: srvDimensionTextureCube, Shader4ComponentMapping: defaultComponentMapping,
				TextureCube: d3d12TexCubeSRV{MostDetailedMip: mostDetailed, MipLevels: levels},
			}
			d.dev.Call(deviceCreateShaderResourceView, t.res.Ptr(), uintptr(unsafe.Pointer(&srv)), handle)
		case explicit.TextureKind3D:
			srv := d3d12ShaderResourceViewDesc3D{
				Format: format, ViewDimension: srvDimensionTexture3D, Shader4ComponentMapping: defaultComponentMapping,
				Texture3D: d3d12Tex3DSRV{MostDetailedMip: mostDetailed, MipLevels: levels},
			}
			d.dev.Call(deviceCreateShaderResourceView, t.res.Ptr(), uintptr(unsafe.Pointer(&srv)), handle)
		default:
			srv := d3d12ShaderResourceViewDesc{
				Format: format, ViewDimension: srvDimensionTexture2D, Shader4ComponentMapping: defaultComponentMapping,
				Texture2D: d3d12Tex2DSRV{MostDetailedMip: mostDetailed, MipLevels: levels},
			}
			d.dev.Call(deviceCreateShaderResourceView, t.res.Ptr(), uintptr(unsafe.Pointer(&srv)), handle)
		}
	}
	if t.desc.Usage&explicit.TextureStorage != 0 && t.desc.Kind != explicit.TextureKindCube && (!layered || t.desc.Kind != explicit.TextureKind2D) {
		n, handle, err := d.cpu[1].alloc(d)
		if err != nil {
			return err
		}
		t.uav = n
		switch t.desc.Kind {
		case explicit.TextureKind2DArray:
			uav := d3d12UnorderedAccessViewDescArray{Format: format, ViewDimension: uavDimensionTexture2DArray,
				Texture2DArray: d3d12Tex2DArrayUAV{MipSlice: uint32(t.base), FirstArraySlice: uint32(t.baseLayer), ArraySize: uint32(t.desc.Layers())}}
			d.dev.Call(deviceCreateUnorderedAccessView, t.res.Ptr(), 0, uintptr(unsafe.Pointer(&uav)), handle)
		case explicit.TextureKind3D:
			uav := d3d12UnorderedAccessViewDesc3D{Format: format, ViewDimension: uavDimensionTexture3D,
				Texture3D: d3d12Tex3DUAV{MipSlice: uint32(t.base), WSize: uint32(max(t.desc.DepthOrLayers, 1))}}
			d.dev.Call(deviceCreateUnorderedAccessView, t.res.Ptr(), 0, uintptr(unsafe.Pointer(&uav)), handle)
		default:
			uav := d3d12UnorderedAccessViewDesc{Format: format, ViewDimension: uavDimensionTexture2D, Texture2D: d3d12Tex2DUAV{MipSlice: uint32(t.base)}}
			d.dev.Call(deviceCreateUnorderedAccessView, t.res.Ptr(), 0, uintptr(unsafe.Pointer(&uav)), handle)
		}
	}
	if t.desc.Usage&explicit.TextureRenderTarget == 0 || t.desc.Kind != explicit.TextureKind2D {
		return nil
	}
	multisampled := t.desc.SampleCount > 1
	if t.desc.Format.IsDepthStencil() {
		n, handle, err := d.cpu[3].alloc(d)
		if err != nil {
			return err
		}
		t.dsv = n
		switch {
		case multisampled:
			dsv := d3d12DepthStencilViewDesc{Format: format, ViewDimension: dsvDimensionTexture2DMS}
			d.dev.Call(deviceCreateDepthStencilView, t.res.Ptr(), uintptr(unsafe.Pointer(&dsv)), handle)
		case layered:
			dsv := d3d12DepthStencilViewDescArray{Format: format, ViewDimension: dsvDimensionTexture2DArray,
				Texture2DArray: d3d12Tex2DArrayDSV{MipSlice: uint32(t.base), FirstArraySlice: uint32(t.baseLayer), ArraySize: 1}}
			d.dev.Call(deviceCreateDepthStencilView, t.res.Ptr(), uintptr(unsafe.Pointer(&dsv)), handle)
		default:
			dsv := d3d12DepthStencilViewDesc{Format: format, ViewDimension: dsvDimensionTexture2D, Texture2D: d3d12Tex2DDSV{MipSlice: uint32(t.base)}}
			d.dev.Call(deviceCreateDepthStencilView, t.res.Ptr(), uintptr(unsafe.Pointer(&dsv)), handle)
		}
		return nil
	}
	n, handle, err := d.cpu[0].alloc(d)
	if err != nil {
		return err
	}
	t.rtv = n
	switch {
	case multisampled:
		rtv := d3d12RenderTargetViewDesc{Format: format, ViewDimension: rtvDimensionTexture2DMS}
		d.dev.Call(deviceCreateRenderTargetView, t.res.Ptr(), uintptr(unsafe.Pointer(&rtv)), handle)
	case layered:
		rtv := d3d12RenderTargetViewDescArray{Format: format, ViewDimension: rtvDimensionTexture2DArray,
			Texture2DArray: d3d12Tex2DArrayRTV{MipSlice: uint32(t.base), FirstArraySlice: uint32(t.baseLayer), ArraySize: 1}}
		d.dev.Call(deviceCreateRenderTargetView, t.res.Ptr(), uintptr(unsafe.Pointer(&rtv)), handle)
	default:
		rtv := d3d12RenderTargetViewDesc{Format: format, ViewDimension: rtvDimensionTexture2D, Texture2D: d3d12Tex2DRTV{MipSlice: uint32(t.base)}}
		d.dev.Call(deviceCreateRenderTargetView, t.res.Ptr(), uintptr(unsafe.Pointer(&rtv)), handle)
	}
	return nil
}

// freeViews gives the descriptors of a texture back.
func (d *Device) freeViews(t *texture) {
	if t.srv >= 0 {
		d.cpu[1].free(t.srv)
		t.srv = -1
	}
	if t.uav >= 0 {
		d.cpu[1].free(t.uav)
		t.uav = -1
	}
	if t.rtv >= 0 {
		d.cpu[0].free(t.rtv)
		t.rtv = -1
	}
	if t.dsv >= 0 {
		d.cpu[3].free(t.dsv)
		t.dsv = -1
	}
}

// rtvHandle is the CPU handle of the render target view of the texture.
func (t *texture) rtvHandle() uintptr { return t.d.cpu[0].handle(t.rtv) }

// dsvHandle is the CPU handle of the depth-stencil view of the texture.
func (t *texture) dsvHandle() uintptr { return t.d.cpu[3].handle(t.dsv) }

// needTexture puts every subresource the texture covers in the state; the transitions are collected by the
// tracker and recorded by barrier. It fails as the tracker's need does.
func (d *Device) needTexture(t *texture, st resState) error {
	mips := t.mips()
	for l := 0; l < t.desc.Layers(); l++ {
		for m := 0; m < t.desc.MipLevels; m++ {
			if err := d.tracker.need(t.root, (t.baseLayer+l)*mips+t.base+m, st); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteTexture implements explicit.Device: the rows are copied into the staging ring, each at a pitch
// of 256 bytes and the whole at an offset of 512 (SPEC §9.8 "Memory"), and copied into the texture by
// a command of the next frame, or of the frame being recorded when it is between passes. A failure to
// stage the data panics: it is out of memory, which a method that returns no error cannot report.
func (d *Device) WriteTexture(t explicit.Texture, level int, r image.Rectangle, data []byte, stride int) {
	dt := d.own2texture(t)
	if r.Empty() {
		return
	}
	if level < 0 || level >= dt.desc.MipLevels {
		panic(errors.New(tagD3D12LevelInvalid, "the texture has no such mip level", "level", level, "levels", dt.desc.MipLevels))
	}
	w, h := max(dt.desc.Width>>level, 1), max(dt.desc.Height>>level, 1)
	bpp := dt.desc.Format.BytesPerPixel()
	rowBytes := r.Dx() * bpp
	if !r.In(image.Rect(0, 0, w, h)) || stride < rowBytes || len(data) < (r.Dy()-1)*stride+rowBytes {
		panic(errors.New(tagD3D12WriteOutOfRange, "the write is outside the texture level or the data is too short",
			"rect", r.String(), "level", level, "width", w, "height", h, "bytes", len(data), "stride", stride))
	}
	l := layoutOf(d.limits, r.Dx(), r.Dy(), bpp)
	res, off, p, err := d.stage(l.Size)
	if err != nil {
		panic(err)
	}
	dst := win32.Bytes(p, l.Size)
	for y := 0; y < r.Dy(); y++ {
		copy(dst[y*l.RowPitch:y*l.RowPitch+rowBytes], data[y*stride:y*stride+rowBytes])
	}
	sub := dt.sub() + level
	d.inCommands(func(list win32.Object) {
		d.opBegin()
		defer d.opEnd()
		if err := d.tracker.need(dt.root, sub, stateCopyDest); err != nil {
			panic(err)
		}
		d.barrier(list)
		dstLoc := d3d12TextureCopyIndex{Resource: dt.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(sub)}
		srcLoc := d3d12TextureCopyLocation{Resource: res.Ptr(), Type: textureCopyPlacedFootprint, Placed: d3d12PlacedFootprint{
			Offset: uint64(off),
			Footprint: d3d12SubresourceFootprint{
				Format: dxgiFormat(dt.desc.Format), Width: uint32(r.Dx()), Height: uint32(r.Dy()), Depth: 1, RowPitch: uint32(l.RowPitch),
			},
		}}
		list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), uintptr(r.Min.X), uintptr(r.Min.Y), 0, uintptr(unsafe.Pointer(&srcLoc)), 0)
	})
}

// WriteTextureRegion implements explicit.Device: the rows are repacked into the staging ring and copied
// into the layers of an array or a cube (one copy for each layer, at its own offset of the ring), or
// into the slices of a volume (one copy), as WriteTexture does for one level of a 2D texture.
func (d *Device) WriteTextureRegion(t explicit.Texture, region explicit.TextureRegion, data []byte, bytesPerRow, rowsPerImage int) {
	dt := d.own2texture(t)
	w, h, n := region.Size[0], region.Size[1], region.Size[2]
	if w <= 0 || h <= 0 || n <= 0 {
		return
	}
	if region.Level < 0 || region.Level >= dt.desc.MipLevels {
		panic(errors.New(tagD3D12LevelInvalid, "the texture has no such mip level", "level", region.Level, "levels", dt.desc.MipLevels))
	}
	lw, lh := max(dt.desc.Width>>region.Level, 1), max(dt.desc.Height>>region.Level, 1)
	depth, layered := 1, dt.desc.Kind == explicit.TextureKind2DArray || dt.desc.Kind == explicit.TextureKindCube
	switch {
	case layered:
		depth = dt.desc.Layers()
	case dt.desc.Kind == explicit.TextureKind3D:
		depth = max(dt.desc.DepthOrLayers>>region.Level, 1)
	}
	bpp := dt.desc.Format.BytesPerPixel()
	rowBytes := w * bpp
	ox, oy, oz := region.Origin[0], region.Origin[1], region.Origin[2]
	if ox < 0 || oy < 0 || oz < 0 || ox+w > lw || oy+h > lh || oz+n > depth || bytesPerRow < rowBytes || rowsPerImage < h ||
		len(data) < ((n-1)*rowsPerImage+h-1)*bytesPerRow+rowBytes {
		panic(errors.New(tagD3D12WriteOutOfRange, "the write is outside the texture level or the data is too short",
			"origin", region.Origin, "size", region.Size, "level", region.Level, "width", lw, "height", lh, "depth", depth, "bytes", len(data)))
	}
	l := layoutOf(d.limits, w, h, bpp)
	imageBytes := l.RowPitch * h // a volume's slices are one pitch × height apart
	if layered {
		imageBytes = alignUp(l.Size, textureDataPlacementAlignment) // a copy of each layer starts at a multiple of 512
	}
	total := imageBytes*(n-1) + l.Size
	res, off, p, err := d.stage(total)
	if err != nil {
		panic(err)
	}
	dst := win32.Bytes(p, total)
	for z := 0; z < n; z++ {
		for y := 0; y < h; y++ {
			row := (z*rowsPerImage + y) * bytesPerRow
			copy(dst[z*imageBytes+y*l.RowPitch:z*imageBytes+y*l.RowPitch+rowBytes], data[row:row+rowBytes])
		}
	}
	footprint := func(depth int) d3d12SubresourceFootprint {
		return d3d12SubresourceFootprint{Format: dxgiFormat(dt.desc.Format), Width: uint32(w), Height: uint32(h), Depth: uint32(depth), RowPitch: uint32(l.RowPitch)}
	}
	mips := dt.mips()
	d.inCommands(func(list win32.Object) {
		d.opBegin()
		defer d.opEnd()
		if !layered {
			sub := dt.base + region.Level
			if err := d.tracker.need(dt.root, sub, stateCopyDest); err != nil {
				panic(err)
			}
			d.barrier(list)
			dstLoc := d3d12TextureCopyIndex{Resource: dt.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(sub)}
			srcLoc := d3d12TextureCopyLocation{Resource: res.Ptr(), Type: textureCopyPlacedFootprint, Placed: d3d12PlacedFootprint{Offset: uint64(off), Footprint: footprint(n)}}
			list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), uintptr(ox), uintptr(oy), uintptr(oz), uintptr(unsafe.Pointer(&srcLoc)), 0)
			return
		}
		for z := 0; z < n; z++ {
			if err := d.tracker.need(dt.root, (dt.baseLayer+oz+z)*mips+dt.base+region.Level, stateCopyDest); err != nil {
				panic(err)
			}
		}
		d.barrier(list)
		for z := 0; z < n; z++ {
			dstLoc := d3d12TextureCopyIndex{Resource: dt.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32((dt.baseLayer+oz+z)*mips + dt.base + region.Level)}
			srcLoc := d3d12TextureCopyLocation{Resource: res.Ptr(), Type: textureCopyPlacedFootprint, Placed: d3d12PlacedFootprint{Offset: uint64(off + z*imageBytes), Footprint: footprint(1)}}
			list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), uintptr(ox), uintptr(oy), 0, uintptr(unsafe.Pointer(&srcLoc)), 0)
		}
	})
}

// own2texture checks that a texture belongs to this device.
func (d *Device) own2texture(t explicit.Texture) *texture {
	dt, ok := t.(*texture)
	if !ok || dt.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "texture"))
	}
	return dt
}

// opBegin starts an operation, a pass or a copy, in the trackers of the textures and of the buffers.
func (d *Device) opBegin() {
	d.tracker.begin()
	d.btracker.begin()
}

// opEnd ends the operation that opBegin started.
func (d *Device) opEnd() {
	d.tracker.end()
	d.btracker.end()
}

// needBuffer puts the buffer in the state; the transition is collected by the tracker and recorded by
// barrier. A buffer in an upload heap stays in GENERIC_READ, which every use of one is.
func (d *Device) needBuffer(b *buffer, st resState) error {
	if !b.gpu {
		return nil
	}
	return d.btracker.need(b, 0, st)
}

// barrier records the transitions the trackers collected as one ResourceBarrier call. A subresource of
// a texture with a stencil is transitioned in both its planes, the depth one and the stencil one.
func (d *Device) barrier(list win32.Object) {
	ts := d.tracker.flush()
	bts := d.btracker.flush()
	if len(ts)+len(bts) == 0 {
		return
	}
	bs := make([]d3d12ResourceBarrier, 0, len(ts)+len(bts))
	for _, tr := range ts {
		planes := 1
		if tr.Res.desc.Format.HasStencil() {
			planes = 2
		}
		for plane := 0; plane < planes; plane++ {
			bs = append(bs, d3d12ResourceBarrier{Type: barrierTypeTransition, Transition: d3d12TransitionBarrier{
				Resource: tr.Res.res.Ptr(), Subresource: uint32(tr.Level + plane*tr.Res.subresourceCount()), StateBefore: uint32(tr.Before), StateAfter: uint32(tr.After),
			}})
		}
	}
	for _, tr := range bts {
		bs = append(bs, d3d12ResourceBarrier{Type: barrierTypeTransition, Transition: d3d12TransitionBarrier{
			Resource: tr.Res.res.Ptr(), Subresource: allSubresources, StateBefore: uint32(tr.Before), StateAfter: uint32(tr.After),
		}})
	}
	list.Call(listResourceBarrier, uintptr(len(bs)), uintptr(unsafe.Pointer(&bs[0])))
}

// uavBarrier makes the writes of the dispatches before it to the unordered access views visible to the
// dispatches after it: Direct3D 12 does not order them by itself. One barrier covers every resource.
func (d *Device) uavBarrier(list win32.Object) {
	b := d3d12ResourceBarrier{Type: barrierTypeUAV}
	list.Call(listResourceBarrier, 1, uintptr(unsafe.Pointer(&b)))
}

// sampler is a sampler descriptor in a CPU-only heap (SPEC §9.8 "Descriptors"): a bind group copies
// it into a shader-visible heap.
type sampler struct {
	d         *Device
	n         int
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
	s.d.counts.samplers--
	s.d.cpu[2].free(s.n)
}

// NewSampler implements explicit.Device.
func (d *Device) NewSampler(desc explicit.SamplerDesc) (explicit.Sampler, error) {
	n, handle, err := d.cpu[2].alloc(d)
	if err != nil {
		return nil, err
	}
	sd := samplerDesc(desc)
	d.dev.Call(deviceCreateSampler, uintptr(unsafe.Pointer(&sd)), handle)
	s := &sampler{d: d, n: n}
	d.counts.samplers++
	d.live[s] = func() { s.destroyed = true; d.counts.samplers--; d.cpu[2].free(s.n) }
	return s, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagD3D12WriteOutOfRange is the tag of the programmer error of a write to a buffer or a texture
	// level that goes past its end, or of data too short for the rectangle.
	tagD3D12WriteOutOfRange = "D3D12WriteOutOfRange"
	// tagD3D12TextureInvalid is the tag of the error for a texture description that is not a texture
	// (no size, no format, more levels than the size has), and for the target of a pass that is not a
	// render target.
	tagD3D12TextureInvalid = "D3D12TextureInvalid"
)
