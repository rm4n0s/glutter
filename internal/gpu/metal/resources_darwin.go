//go:build darwin

package metal

import (
	"image"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// buffer is an MTLBuffer: shared memory on a device with unified memory, private memory otherwise. The CPU
// never touches it: writes and reads go through staging and readback buffers and blits, which keeps their
// order with the passes (SPEC §29.5).
type buffer struct {
	d     *Device
	obj   objc.ID
	size  int
	usage explicit.BufferUsage
}

var _ explicit.Buffer = (*buffer)(nil)

func (b *buffer) Size() int                   { return b.size }
func (b *buffer) Usage() explicit.BufferUsage { return b.usage }

// Destroy implements explicit.Buffer: the command buffers that use the buffer keep it until they finish.
func (b *buffer) Destroy() { release(b.obj); b.obj = 0 }

// NewBuffer implements explicit.Device.
func (d *Device) NewBuffer(size int, usage explicit.BufferUsage) (explicit.Buffer, error) {
	d.limits.CheckBuffer(size, usage)
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	storage := storagePrivate
	if d.unif {
		storage = storageShared
	}
	var obj objc.ID
	objcx.WithPool(func() {
		obj = call(d.dev, "newBufferWithLength:options:", uint64(max(size, 16)), resourceOptions(storage, false))
	})
	if obj == 0 {
		return nil, errors.New(tagGPUOutOfMemory, "Metal made no buffer", "size", size)
	}
	return &buffer{d: d, obj: obj, size: size, usage: usage}, nil
}

// texture is an MTLTexture: private memory, written and read through staging. A Level or Layer is a view of
// the parent's texture.
type texture struct {
	d     *Device
	obj   objc.ID
	desc  explicit.TextureDesc
	owner *texture // the texture a view is of, nil for a texture of its own
	views map[[2]int]*texture
	// drawable is set for the texture of a drawable of a swap chain, which the chain releases.
	drawable bool
}

var _ explicit.Texture = (*texture)(nil)

// Desc implements explicit.Texture.
func (t *texture) Desc() explicit.TextureDesc { return t.desc }

// Destroy implements explicit.Texture. A view is the parent's, and a drawable's texture the swap chain's.
func (t *texture) Destroy() {
	if t.owner != nil || t.drawable || t.obj == 0 {
		return
	}
	for _, v := range t.views {
		release(v.obj)
		v.obj = 0
	}
	t.views = nil
	release(t.obj)
	t.obj = 0
}

// NewTexture implements explicit.Device.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	d.limits.CheckTexture(desc)
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	desc = desc.Normalized()
	var obj objc.ID
	objcx.WithPool(func() {
		td := newObject("MTLTextureDescriptor")
		defer release(td)
		call(td, "setTextureType:", textureType(desc.Kind, desc.SampleCount))
		call(td, "setPixelFormat:", pixelFormat(desc.Format))
		call(td, "setWidth:", uint64(desc.Width))
		call(td, "setHeight:", uint64(desc.Height))
		call(td, "setDepth:", uint64(desc.Depth()))
		call(td, "setMipmapLevelCount:", uint64(desc.MipLevels))
		call(td, "setArrayLength:", uint64(max(1, arrayLength(desc))))
		call(td, "setSampleCount:", uint64(desc.SampleCount))
		usage := textureUsage(desc.Usage)
		if usage == 0 {
			usage = textureUsageShaderRead
		}
		call(td, "setUsage:", usage)
		call(td, "setStorageMode:", storagePrivate)
		obj = call(d.dev, "newTextureWithDescriptor:", td)
	})
	if obj == 0 {
		return nil, errors.New(tagGPUOutOfMemory, "Metal made no texture", "width", desc.Width, "height", desc.Height, "format", int(desc.Format))
	}
	return &texture{d: d, obj: obj, desc: desc}, nil
}

// arrayLength is the length of the array of a texture of the description: the layers of a 2D array, and 1
// for the others (a cube has its six faces in one slice group that Metal counts itself).
func arrayLength(desc explicit.TextureDesc) int {
	if desc.Kind == explicit.TextureKind2DArray {
		return desc.DepthOrLayers
	}
	return 1
}

// Level implements explicit.Texture: the view of one mip level, with all its layers.
func (t *texture) Level(i int) explicit.Texture {
	if i == 0 && t.desc.MipLevels == 1 {
		return t
	}
	return t.view(i, -1)
}

// Layer implements explicit.Texture: the view of one layer of a level as a 2D texture.
func (t *texture) Layer(level, layer int) explicit.Texture {
	t.d.limits.CheckLayer(t.desc, level, layer)
	if t.desc.Kind == explicit.TextureKind2D && t.desc.MipLevels == 1 && level == 0 && layer == 0 {
		return t
	}
	return t.view(level, layer)
}

// view makes (or finds) the view of a level; layer is -1 for all the layers, else the one layer as a 2D
// texture.
func (t *texture) view(level, layer int) *texture {
	key := [2]int{level, layer}
	if v, ok := t.views[key]; ok {
		return v
	}
	desc := t.desc
	desc.Width, desc.Height = max(1, desc.Width>>level), max(1, desc.Height>>level)
	if desc.Kind == explicit.TextureKind3D {
		desc.DepthOrLayers = max(1, desc.DepthOrLayers>>level)
	}
	desc.MipLevels = 1
	typ := textureType(desc.Kind, desc.SampleCount)
	slices := nsRange{Location: 0, Length: uint64(max(1, t.desc.Layers()))}
	if layer >= 0 {
		desc.Kind, desc.DepthOrLayers = explicit.TextureKind2D, 1
		typ = textureType2D
		slices = nsRange{Location: uint64(layer), Length: 1}
	}
	var obj objc.ID
	objcx.WithPool(func() {
		obj = retain(call(t.obj, "newTextureViewWithPixelFormat:textureType:levels:slices:",
			pixelFormat(desc.Format), typ, nsRange{Location: uint64(level), Length: 1}, slices))
	})
	if obj == 0 {
		panic(errors.New(TagMetalCallFailed, "Metal made no texture view", "call", "newTextureViewWithPixelFormat", "level", level, "layer", layer))
	}
	v := &texture{d: t.d, obj: obj, desc: desc, owner: t}
	if t.views == nil {
		t.views = map[[2]int]*texture{}
	}
	t.views[key] = v
	return v
}

// sampler is an MTLSamplerState.
type sampler struct {
	d   *Device
	obj objc.ID
}

var _ explicit.Sampler = (*sampler)(nil)

// Destroy implements explicit.Sampler.
func (s *sampler) Destroy() { release(s.obj); s.obj = 0 }

// NewSampler implements explicit.Device: the mip filter picks between levels, and FilterUnspecified uses
// level 0 only (SPEC §9.3).
func (d *Device) NewSampler(desc explicit.SamplerDesc) (explicit.Sampler, error) {
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	var obj objc.ID
	objcx.WithPool(func() {
		sd := newObject("MTLSamplerDescriptor")
		defer release(sd)
		call(sd, "setMinFilter:", minMagFilter(desc.MinFilter))
		call(sd, "setMagFilter:", minMagFilter(desc.MagFilter))
		call(sd, "setMipFilter:", mipFilter(desc.MipmapFilter))
		call(sd, "setSAddressMode:", addressMode(desc.AddressU))
		call(sd, "setTAddressMode:", addressMode(desc.AddressV))
		call(sd, "setRAddressMode:", addressMode(desc.AddressW))
		obj = call(d.dev, "newSamplerStateWithDescriptor:", sd)
	})
	if obj == 0 {
		return nil, errors.New(TagMetalCallFailed, "Metal made no sampler", "call", "newSamplerStateWithDescriptor")
	}
	return &sampler{d: d, obj: obj}, nil
}

// pendingWrite is a write that waits for a command buffer: the staged bytes and where they go.
type pendingWrite struct {
	chunk *stageChunk
	off   int
	dst   objc.ID // retained until the blit is recorded
	// A write to a buffer: the destination offset and the size.
	isBuffer bool
	dstOff   int
	size     int
	// A write to a texture: the rows, the box, the level and the slice.
	bytesPerRow, bytesPerImage int
	box                        mtlSize
	origin                     mtlOrigin
	level, slice               int
}

func (w pendingWrite) release() { release(w.dst) }

// WriteBuffer implements explicit.Device: the bytes are copied now into staging memory, and a blit puts them
// in the buffer before the next pass of a frame.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	mb := d.ownBuffer(b)
	if len(data) == 0 {
		return
	}
	if offset < 0 || offset+len(data) > mb.size {
		panic(errors.New(TagMetalUsage, "WriteBuffer writes outside the buffer", "offset", offset, "len", len(data), "size", mb.size))
	}
	c, off, err := d.stage(len(data))
	if err != nil {
		d.markLost(err)
		return
	}
	copy(c.mem[off:], data)
	c.dirty.add(off, len(data))
	d.pending = append(d.pending, pendingWrite{chunk: c, off: off, dst: retain(mb.obj), isBuffer: true, dstOff: offset, size: len(data)})
}

// WriteTexture implements explicit.Device: the rectangle r of a level of a texture, from rows that are stride
// bytes apart.
func (d *Device) WriteTexture(t explicit.Texture, level int, r image.Rectangle, data []byte, stride int) {
	d.WriteTextureRegion(t, explicit.TextureRegion{
		Level: level, Origin: [3]int{r.Min.X, r.Min.Y, 0}, Size: [3]int{r.Dx(), r.Dy(), 1},
	}, data, stride, 0)
}

// WriteTextureRegion implements explicit.Device: the box of a level of a texture, from rows bytesPerRow
// bytes apart and images rowsPerImage rows apart (0 is the height of the box).
func (d *Device) WriteTextureRegion(t explicit.Texture, region explicit.TextureRegion, data []byte, bytesPerRow, rowsPerImage int) {
	mt := d.ownTexture(t)
	w, h, depth := region.Size[0], region.Size[1], region.Size[2]
	if w <= 0 || h <= 0 || depth <= 0 {
		return
	}
	if !mt.desc.Format.CanBeWritten() {
		panic(errors.New(TagMetalUsage, "the texture's format cannot be written", "format", int(mt.desc.Format)))
	}
	bpp := mt.desc.Format.BytesPerPixel()
	rowBytes := w * bpp
	if rowsPerImage == 0 {
		rowsPerImage = h
	}
	if bytesPerRow == 0 {
		bytesPerRow = rowBytes
	}
	if bytesPerRow < rowBytes {
		panic(errors.New(TagMetalUsage, "bytesPerRow is shorter than a row", "bytesPerRow", bytesPerRow, "row", rowBytes))
	}
	imageBytes := rowBytes * h
	if need := (depth-1)*rowsPerImage*bytesPerRow + (h-1)*bytesPerRow + rowBytes; len(data) < need {
		panic(errors.New(TagMetalUsage, "the data is shorter than the box", "len", len(data), "need", need))
	}
	c, off, err := d.stage(imageBytes * depth)
	if err != nil {
		d.markLost(err)
		return
	}
	for z := 0; z < depth; z++ {
		for y := 0; y < h; y++ {
			src := z*rowsPerImage*bytesPerRow + y*bytesPerRow
			copy(c.mem[off+z*imageBytes+y*rowBytes:], data[src:src+rowBytes])
		}
	}
	c.dirty.add(off, imageBytes*depth)
	layers := mt.desc.Kind == explicit.TextureKind2DArray || mt.desc.Kind == explicit.TextureKindCube
	if layers {
		// Each layer is a slice of its own: a copy takes one slice, and the depth is 1.
		for z := 0; z < depth; z++ {
			d.pending = append(d.pending, pendingWrite{chunk: c, off: off + z*imageBytes, dst: retain(mt.obj),
				bytesPerRow: rowBytes, bytesPerImage: imageBytes, box: mtlSize{uint64(w), uint64(h), 1},
				origin: mtlOrigin{uint64(region.Origin[0]), uint64(region.Origin[1]), 0}, level: region.Level, slice: region.Origin[2] + z})
		}
		return
	}
	d.pending = append(d.pending, pendingWrite{chunk: c, off: off, dst: retain(mt.obj), bytesPerRow: rowBytes, bytesPerImage: imageBytes,
		box:    mtlSize{uint64(w), uint64(h), uint64(depth)},
		origin: mtlOrigin{uint64(region.Origin[0]), uint64(region.Origin[1]), uint64(region.Origin[2])}, level: region.Level})
}

// ownBuffer returns the device's buffer behind an explicit.Buffer, and panics for another device's.
func (d *Device) ownBuffer(b explicit.Buffer) *buffer {
	mb, ok := b.(*buffer)
	if !ok || mb.d != d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "buffer"))
	}
	return mb
}

// ownTexture returns the device's texture behind an explicit.Texture, and panics for another device's.
func (d *Device) ownTexture(t explicit.Texture) *texture {
	mt, ok := t.(*texture)
	if !ok || mt.d != d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "texture"))
	}
	return mt
}

// ownSampler returns the device's sampler behind an explicit.Sampler.
func (d *Device) ownSampler(s explicit.Sampler) *sampler {
	ms, ok := s.(*sampler)
	if !ok || ms.d != d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "sampler"))
	}
	return ms
}

func alignUp(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) / a * a
}
