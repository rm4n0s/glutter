//go:build windows

package d3d12

import (
	"image"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// buffer is a committed buffer in an upload heap, mapped for good and written at once, in the state
// GENERIC_READ (SPEC §9.8 "Memory").
type buffer struct {
	d         *Device
	res       win32.Object
	ptr       uintptr
	size      int
	destroyed bool
}

var _ explicit.Buffer = (*buffer)(nil)

// Size implements explicit.Buffer.
func (b *buffer) Size() int { return b.size }

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
	d.retire(func() { d.releaseBuffer(b.res, b.allocated()) })
}

// allocated is the size of the resource: a buffer is never smaller than 4 bytes.
func (b *buffer) allocated() int { return max(b.size, 4) }

// NewBuffer implements explicit.Device. Every buffer, whatever its usage, is in an upload heap: the
// uniform ring and the vertex and index data are written by the CPU at once (SPEC §9.8).
func (d *Device) NewBuffer(size int, _ explicit.BufferUsage) (explicit.Buffer, error) {
	n := max(size, 4)
	res, err := d.createBuffer(heapTypeUpload, n)
	if err != nil {
		return nil, err
	}
	ptr, err := d.mapBuffer(res, false, n)
	if err != nil {
		d.releaseBuffer(res, n)
		return nil, err
	}
	b := &buffer{d: d, res: res, ptr: ptr, size: size}
	d.counts.buffers++
	d.live[b] = func() { b.destroyed = true; d.counts.buffers--; d.releaseBuffer(b.res, b.allocated()) }
	return b, nil
}

// WriteBuffer implements explicit.Device: the memory is mapped, so the data is copied at once. The
// upload heap is write-combined: the data is only written, never read back.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	db, ok := b.(*buffer)
	if !ok || db.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "buffer"))
	}
	if offset < 0 || offset+len(data) > db.size {
		panic(errors.New(tagD3D12WriteOutOfRange, "the write is outside the buffer", "offset", offset, "bytes", len(data), "size", db.size))
	}
	copy(win32.Bytes(db.ptr+uintptr(offset), len(data)), data)
}

// texture is a 2D texture in a committed resource in the default heap. A texture made by Level
// shares the resource with its parent and owns only its descriptors.
type texture struct {
	d    *Device
	desc explicit.TextureDesc
	res  win32.Object
	// root is the texture that owns the resource: itself, or the parent of a level. The state tracker
	// follows the root's levels.
	root *texture
	// base is the first mip level this texture starts at: 0, or the level of a texture made by Level.
	base   int
	levels []*texture // the textures made by Level, indexed by level; nil until asked for
	// srv and rtv are the numbers of the CPU descriptors of the views, -1 when there is none: the
	// shader resource view of all the texture's levels, and the render target view of its first.
	srv, rtv  int
	destroyed bool
}

var _ explicit.Texture = (*texture)(nil)

// Desc implements explicit.Texture.
func (t *texture) Desc() explicit.TextureDesc { return t.desc }

// Level implements explicit.Texture: a texture over one level of the resource, with views of its own
// that the parent releases.
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
	lv := &texture{d: t.d, desc: desc, res: t.res, root: t, base: i, srv: -1, rtv: -1}
	if err := t.d.createViews(lv); err != nil {
		panic(err)
	}
	t.levels[i] = lv
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
	d.freeViews(t)
	d.tracker.forget(t)
	d.counts.bytes -= textureBytes(t.desc)
	d.disown(t.res)
}

// textureBytes is the size of the texture with all its levels, for the accounting.
func textureBytes(desc explicit.TextureDesc) int64 {
	var n int64
	for i := 0; i < max(desc.MipLevels, 1); i++ {
		n += int64(max(desc.Width>>i, 1)) * int64(max(desc.Height>>i, 1)) * int64(desc.Format.BytesPerPixel())
	}
	return n
}

// maxMipLevels is how many levels a texture of the size can have: down to 1x1.
func maxMipLevels(width, height int) int {
	n := 1
	for m := max(width, height); m > 1; m >>= 1 {
		n++
	}
	return n
}

// NewTexture implements explicit.Device: a committed texture in the default heap, in the state
// COMMON, with the descriptors of its views (SPEC §9.8). It fails with D3D12TextureInvalid for a
// description that is not a texture (a programmer error, SPEC §9.3), GPUOutOfMemory, and the other
// tags of check.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	d.limits.CheckTexture(desc)
	if desc.MipLevels < 1 {
		desc.MipLevels = 1
	}
	if desc.Width < 1 || desc.Height < 1 || desc.Format.BytesPerPixel() == 0 || desc.MipLevels > maxMipLevels(desc.Width, desc.Height) {
		return nil, errors.New(tagD3D12TextureInvalid, "a texture needs a size, a format and a number of levels that fits its size",
			"width", desc.Width, "height", desc.Height, "format", int(desc.Format), "levels", desc.MipLevels)
	}
	hp := heapProps(heapTypeDefault)
	rd := d3d12ResourceDesc{
		Dimension: resourceDimensionTexture2D, Width: uint64(desc.Width), Height: uint32(desc.Height), DepthOrArraySize: 1,
		MipLevels: uint16(desc.MipLevels), Format: dxgiFormat(desc.Format), SampleDesc: dxgiSampleDesc{Count: 1}, Layout: textureLayoutUnknown,
	}
	if desc.Usage&explicit.TextureRenderTarget != 0 {
		rd.Flags = resourceFlagAllowRT
	}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateCommittedResource", deviceCreateCommittedResource,
		uintptr(unsafe.Pointer(&hp)), 0, uintptr(unsafe.Pointer(&rd)), uintptr(resourceStateCommon), 0,
		uintptr(unsafe.Pointer(&iidResource)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	t := &texture{d: d, desc: desc, res: d.own(win32.ObjectFromPtr(out)), srv: -1, rtv: -1}
	t.root = t
	d.tracker.add(t, desc.MipLevels)
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
// its levels (one level for a texture made by Level), and the render target view of its first level
// when it is drawn into.
func (d *Device) createViews(t *texture) error {
	format := dxgiFormat(t.desc.Format)
	if t.desc.Usage&explicit.TextureSampled != 0 {
		n, handle, err := d.cpu[1].alloc(d)
		if err != nil {
			return err
		}
		t.srv = n
		srv := d3d12ShaderResourceViewDesc{
			Format: format, ViewDimension: srvDimensionTexture2D, Shader4ComponentMapping: defaultComponentMapping,
			Texture2D: d3d12Tex2DSRV{MostDetailedMip: uint32(t.base), MipLevels: uint32(t.desc.MipLevels)},
		}
		d.dev.Call(deviceCreateShaderResourceView, t.res.Ptr(), uintptr(unsafe.Pointer(&srv)), handle)
	}
	if t.desc.Usage&explicit.TextureRenderTarget != 0 {
		n, handle, err := d.cpu[0].alloc(d)
		if err != nil {
			return err
		}
		t.rtv = n
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
	if t.rtv >= 0 {
		d.cpu[0].free(t.rtv)
		t.rtv = -1
	}
}

// rtvHandle is the CPU handle of the render target view of the texture.
func (t *texture) rtvHandle() uintptr { return t.d.cpu[0].handle(t.rtv) }

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
	sub := dt.base + level
	d.inCommands(func(list win32.Object) {
		d.tracker.begin()
		defer d.tracker.end()
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

// own2texture checks that a texture belongs to this device.
func (d *Device) own2texture(t explicit.Texture) *texture {
	dt, ok := t.(*texture)
	if !ok || dt.d != d {
		panic(errors.New(tagD3D12ForeignResource, "the resource was not made by this Direct3D 12 device", "resource", "texture"))
	}
	return dt
}

// barrier records the transitions the tracker collected as one ResourceBarrier call.
func (d *Device) barrier(list win32.Object) {
	ts := d.tracker.flush()
	if len(ts) == 0 {
		return
	}
	bs := make([]d3d12ResourceBarrier, len(ts))
	for i, tr := range ts {
		bs[i] = d3d12ResourceBarrier{Type: barrierTypeTransition, Transition: d3d12TransitionBarrier{
			Resource: tr.Res.res.Ptr(), Subresource: uint32(tr.Level), StateBefore: uint32(tr.Before), StateAfter: uint32(tr.After),
		}}
	}
	list.Call(listResourceBarrier, uintptr(len(bs)), uintptr(unsafe.Pointer(&bs[0])))
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
