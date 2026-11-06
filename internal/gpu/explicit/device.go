package explicit

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// BufferUsage says what a buffer is used for; values are combined with |.
//
// Flutter: no counterpart
type BufferUsage int

// The uses of a buffer. Every buffer can be written with Device.WriteBuffer.
const (
	// BufferVertex is a buffer of vertex or instance data.
	//
	// Flutter: no counterpart
	BufferVertex BufferUsage = 1 << iota
	// BufferIndex is a buffer of 16 or 32 bit indices (SetIndexBuffer, SetIndexBuffer32).
	//
	// Flutter: no counterpart
	BufferIndex
	// BufferUniform is a buffer that uniform bindings read, with dynamic offsets.
	//
	// Flutter: no counterpart
	BufferUniform
	// BufferStorage is a buffer that storage bindings read (var<storage, read>) or, in a compute
	// shader, write (var<storage, read_write>), bound at a Binding.Offset that is a multiple of
	// Limits.StorageOffsetAlignment.
	//
	// Flutter: no counterpart
	BufferStorage
	// BufferIndirect is a buffer of the arguments of DrawIndirect, DrawIndexedIndirect and
	// DispatchIndirect, which a shader may have written (with BufferStorage).
	//
	// Flutter: no counterpart
	BufferIndirect
	// BufferCopySource is a buffer that the copies and ReadBuffer read.
	//
	// Flutter: no counterpart
	BufferCopySource
	// BufferCopyDestination is a buffer that the copies write.
	//
	// Flutter: no counterpart
	BufferCopyDestination
	// BufferQueryResolve is a buffer that ResolveQuerySet writes.
	//
	// Flutter: no counterpart
	BufferQueryResolve
)

// TextureFormat is the pixel format of a texture. There are no sRGB formats:
// blending is done in non-linear sRGB (SPEC §9.1, §9.3). The zero value is
// TextureFormatUnspecified.
//
// Flutter: no counterpart
type TextureFormat int

// The formats of a texture (SPEC §27.1).
const (
	// TextureFormatUnspecified is the zero value: no format.
	//
	// Flutter: no counterpart
	TextureFormatUnspecified TextureFormat = iota
	// TextureFormatRGBA8Unorm is 4 bytes per pixel, red first.
	//
	// Flutter: no counterpart
	TextureFormatRGBA8Unorm
	// TextureFormatBGRA8Unorm is 4 bytes per pixel, blue first: the usual swapchain
	// format.
	//
	// Flutter: no counterpart
	TextureFormatBGRA8Unorm
	// TextureFormatR8Unorm is 1 byte per pixel: coverage masks and glyph atlases.
	//
	// Flutter: no counterpart
	TextureFormatR8Unorm
	// TextureFormatRGBA16Float is 8 bytes per pixel, four half floats: linear color for
	// high dynamic range.
	//
	// Flutter: no counterpart
	TextureFormatRGBA16Float
	// TextureFormatRGBA32Float is 16 bytes per pixel, four floats. Shaders sample it with
	// textureLoad, or with a filter when Features.Float32Filterable.
	//
	// Flutter: no counterpart
	TextureFormatRGBA32Float
	// TextureFormatR32Float is 4 bytes per pixel, one float: heights, distances.
	//
	// Flutter: no counterpart
	TextureFormatR32Float
	// TextureFormatR32Uint is 4 bytes per pixel, one unsigned integer: object ids.
	//
	// Flutter: no counterpart
	TextureFormatR32Uint
	// TextureFormatRG32Uint is 8 bytes per pixel, two unsigned integers.
	//
	// Flutter: no counterpart
	TextureFormatRG32Uint
	// TextureFormatDepth32Float is a depth buffer of one float per pixel.
	//
	// Flutter: no counterpart
	TextureFormatDepth32Float
	// TextureFormatDepth24PlusStencil8 is a depth buffer of at least 24 bits with an 8 bit
	// stencil. It cannot be copied.
	//
	// Flutter: no counterpart
	TextureFormatDepth24PlusStencil8
)

// BytesPerPixel returns the size of a pixel in bytes; 0 for an unspecified format. For
// TextureFormatDepth24PlusStencil8 it is the 4 bytes that a pixel nominally takes: the format
// cannot be copied, so the number is only used to count memory.
//
// Flutter: no counterpart
func (f TextureFormat) BytesPerPixel() int {
	switch f {
	case TextureFormatRGBA32Float:
		return 16
	case TextureFormatRGBA16Float, TextureFormatRG32Uint:
		return 8
	case TextureFormatRGBA8Unorm, TextureFormatBGRA8Unorm, TextureFormatR32Float, TextureFormatR32Uint,
		TextureFormatDepth32Float, TextureFormatDepth24PlusStencil8:
		return 4
	case TextureFormatR8Unorm:
		return 1
	}
	return 0
}

// IsDepthStencil reports whether the format is a depth or depth-stencil format.
//
// Flutter: no counterpart
func (f TextureFormat) IsDepthStencil() bool {
	return f == TextureFormatDepth32Float || f == TextureFormatDepth24PlusStencil8
}

// HasStencil reports whether the format has a stencil aspect.
//
// Flutter: no counterpart
func (f TextureFormat) HasStencil() bool { return f == TextureFormatDepth24PlusStencil8 }

// Features are the optional features of a device (SPEC §27.1); each backend reports what it has.
//
// Flutter: no counterpart
type Features struct {
	// Float32Filterable says that RGBA32Float and R32Float textures can be read with a filtering
	// sampler (WebGPU's float32-filterable).
	Float32Filterable bool
	// TimestampQuery says that query sets can be made and passes can write timestamps into them
	// (WebGPU's timestamp-query, Vulkan's timestampValidBits on the queue, Direct3D 12's timestamp
	// queries).
	TimestampQuery bool
	// TimestampPeriod is the time of one tick of a timestamp, in nanoseconds; 1 on WebGPU, whose
	// timestamps are nanoseconds. It is 0 when TimestampQuery is false.
	TimestampPeriod float64
}

// SampleType is how a shader reads the texels of a texture (WebGPU's GPUTextureSampleType).
// The zero value is SampleTypeFloat.
//
// Flutter: no counterpart
type SampleType int

// The sample types.
const (
	// SampleTypeFloat is a floating point texture that a filtering sampler can read.
	//
	// Flutter: no counterpart
	SampleTypeFloat SampleType = iota
	// SampleTypeUnfilterableFloat is a floating point texture that only textureLoad and a
	// non-filtering sampler can read.
	//
	// Flutter: no counterpart
	SampleTypeUnfilterableFloat
	// SampleTypeUint is a texture of unsigned integers, read with textureLoad.
	//
	// Flutter: no counterpart
	SampleTypeUint
)

// Fits reports whether a texture that shaders read as s can be bound where a shader declares want:
// a filterable float texture fits a float or an unfilterable float binding, a float32 texture without
// filtering only the unfilterable one, a uint texture only a uint binding.
//
// Flutter: no counterpart
func (s SampleType) Fits(want SampleType) bool {
	switch want {
	case SampleTypeFloat:
		return s == SampleTypeFloat
	case SampleTypeUnfilterableFloat:
		return s == SampleTypeFloat || s == SampleTypeUnfilterableFloat
	}
	return s == want
}

// SampleType returns how shaders read the format, for a device with the features; the formats
// that are not sampled (the depth formats) report SampleTypeFloat and have no TextureSampled
// usage.
//
// Flutter: no counterpart
func (f TextureFormat) SampleType(features Features) SampleType {
	switch f {
	case TextureFormatR32Uint, TextureFormatRG32Uint:
		return SampleTypeUint
	case TextureFormatRGBA32Float, TextureFormatR32Float:
		if !features.Float32Filterable {
			return SampleTypeUnfilterableFloat
		}
	}
	return SampleTypeFloat
}

// RenderTargetPixelBytes is the number of bytes that the format takes in the budget of
// Limits.MaxColorAttachmentBytesPerSample (WebGPU's render target pixel byte cost); 0 for a format
// that is not a color attachment.
//
// Flutter: no counterpart
func (f TextureFormat) RenderTargetPixelBytes() int {
	if f.IsDepthStencil() || f == TextureFormatUnspecified {
		return 0
	}
	return f.BytesPerPixel()
}

// AllowedUsage returns the uses the format allows. A format that no use allows is unspecified.
// The depth formats are drawn into (and Depth32Float copied from), and are not sampled in 1.10.
//
// Flutter: no counterpart
func (f TextureFormat) AllowedUsage() TextureUsage {
	const common = TextureSampled | TextureRenderTarget | TextureCopySource
	switch f {
	case TextureFormatRGBA8Unorm, TextureFormatRGBA16Float, TextureFormatR32Float, TextureFormatR32Uint:
		return common | TextureStorage
	case TextureFormatBGRA8Unorm, TextureFormatR8Unorm, TextureFormatRGBA32Float, TextureFormatRG32Uint:
		return common
	case TextureFormatDepth32Float:
		return TextureRenderTarget | TextureCopySource
	case TextureFormatDepth24PlusStencil8:
		return TextureRenderTarget
	}
	return 0
}

// CanBeCopied reports whether CopyTexture and ReadTextureBytes can read the format (when the
// texture has TextureCopySource): every format but Depth24PlusStencil8.
//
// Flutter: no counterpart
func (f TextureFormat) CanBeCopied() bool {
	return f != TextureFormatUnspecified && f != TextureFormatDepth24PlusStencil8
}

// CanBeWritten reports whether Device.WriteTexture can fill the format: the depth formats cannot
// be written (a pass fills them).
//
// Flutter: no counterpart
func (f TextureFormat) CanBeWritten() bool {
	return f != TextureFormatUnspecified && !f.IsDepthStencil()
}

// TextureUsage says what a texture is used for; values are combined with |. Every
// texture whose format CanBeWritten can be written with Device.WriteTexture.
//
// Flutter: no counterpart
type TextureUsage int

// The uses of a texture.
const (
	// TextureSampled is a texture that shaders sample.
	//
	// Flutter: no counterpart
	TextureSampled TextureUsage = 1 << iota
	// TextureRenderTarget is a texture that a pass draws into.
	//
	// Flutter: no counterpart
	TextureRenderTarget
	// TextureCopySource is a texture that CopyTexture or ReadTexture reads.
	//
	// Flutter: no counterpart
	TextureCopySource
	// TextureStorage is a texture that compute shaders write (SPEC §27.1). Only RGBA8Unorm,
	// RGBA16Float, R32Float and R32Uint allow it.
	//
	// Flutter: no counterpart
	TextureStorage
)

// TextureKind is the shape of a texture. The zero value is TextureKindUnspecified, which
// means TextureKind2D.
//
// Flutter: no counterpart
type TextureKind int

// The kinds of texture.
const (
	// TextureKindUnspecified is the zero value and means TextureKind2D.
	//
	// Flutter: no counterpart
	TextureKindUnspecified TextureKind = iota
	// TextureKind2D is a flat image.
	//
	// Flutter: no counterpart
	TextureKind2D
	// TextureKind2DArray is TextureDesc.DepthOrLayers images of one size, indexed by layer.
	//
	// Flutter: no counterpart
	TextureKind2DArray
	// TextureKindCube is six square images, the faces +X, -X, +Y, -Y, +Z and -Z in that order
	// (layers 0 to 5), sampled by a direction.
	//
	// Flutter: no counterpart
	TextureKindCube
	// TextureKind3D is a volume of TextureDesc.DepthOrLayers slices.
	//
	// Flutter: no counterpart
	TextureKind3D
)

// TextureDesc describes a texture.
//
// Flutter: no counterpart
type TextureDesc struct {
	// Width and Height are in pixels, at most Limits.MaxTextureSize (Limits.MaxTextureSize3D for
	// a 3D texture).
	Width, Height int
	// Kind is the shape; the zero value is a 2D texture.
	Kind TextureKind
	// DepthOrLayers is the number of layers of a 2D array, the depth of a 3D texture, and 6 for a
	// cube (0 means 6); for a 2D texture it is 1 (0 means 1).
	DepthOrLayers int
	// Format is the pixel format.
	Format TextureFormat
	// MipLevels is the number of levels; 0 means 1.
	MipLevels int
	// SampleCount is 1 or 4 samples for each pixel; 0 means 1. A multisampled texture is a 2D
	// texture with one level that only a pass draws into, and a resolve target gets its pixels.
	SampleCount int
	// Usage says how the texture is used.
	Usage TextureUsage
}

// Normalized returns the description with the defaults filled in: kind 2D, the layers or depth
// of the kind (1, or 6 for a cube), one mip level and one sample.
//
// Flutter: no counterpart
func (d TextureDesc) Normalized() TextureDesc {
	if d.Kind == TextureKindUnspecified {
		d.Kind = TextureKind2D
	}
	switch d.Kind {
	case TextureKind2D:
		if d.DepthOrLayers == 0 {
			d.DepthOrLayers = 1
		}
	case TextureKindCube:
		if d.DepthOrLayers == 0 {
			d.DepthOrLayers = 6
		}
	case TextureKind2DArray, TextureKind3D:
		if d.DepthOrLayers == 0 {
			d.DepthOrLayers = 1
		}
	}
	if d.MipLevels < 1 {
		d.MipLevels = 1
	}
	if d.SampleCount < 1 {
		d.SampleCount = 1
	}
	return d
}

// Layers returns the number of array layers of the normalized description: the layers of an array,
// 6 for a cube and 1 for the others.
//
// Flutter: no counterpart
func (d TextureDesc) Layers() int {
	d = d.Normalized()
	if d.Kind == TextureKind2DArray || d.Kind == TextureKindCube {
		return d.DepthOrLayers
	}
	return 1
}

// Depth returns the depth of the normalized description: the slices of a 3D texture and 1 for the
// others.
//
// Flutter: no counterpart
func (d TextureDesc) Depth() int {
	d = d.Normalized()
	if d.Kind == TextureKind3D {
		return d.DepthOrLayers
	}
	return 1
}

// TextureRegion is a box of one mip level of a texture: the pixels from Origin, of the size Size.
// The third coordinate is the layer of an array or a cube, or the slice of a 3D texture.
//
// Flutter: no counterpart
type TextureRegion struct {
	// Level is the mip level.
	Level int
	// Origin is the first pixel (x, y) and the first layer or slice (z).
	Origin [3]int
	// Size is the width and height in pixels and the number of layers or slices.
	Size [3]int
}

// FilterMode is how a sampler reads between texels. The zero value is
// FilterUnspecified, which means FilterNearest.
//
// Flutter: no counterpart
type FilterMode int

// The filter modes.
const (
	// FilterUnspecified is the zero value and means FilterNearest.
	//
	// Flutter: no counterpart
	FilterUnspecified FilterMode = iota
	// FilterNearest takes the nearest texel.
	//
	// Flutter: no counterpart
	FilterNearest
	// FilterLinear blends the two nearest texels in each direction.
	//
	// Flutter: no counterpart
	FilterLinear
)

// AddressMode is what a sampler does outside the texture. The zero value is
// AddressUnspecified, which means AddressClamp.
//
// Flutter: no counterpart
type AddressMode int

// The address modes.
const (
	// AddressUnspecified is the zero value and means AddressClamp.
	//
	// Flutter: no counterpart
	AddressUnspecified AddressMode = iota
	// AddressClamp repeats the edge texel.
	//
	// Flutter: no counterpart
	AddressClamp
	// AddressRepeat tiles the texture.
	//
	// Flutter: no counterpart
	AddressRepeat
	// AddressMirror tiles the texture, flipping every other tile.
	//
	// Flutter: no counterpart
	AddressMirror
)

// SamplerDesc describes a sampler.
//
// Flutter: no counterpart
type SamplerDesc struct {
	// MinFilter and MagFilter are used when the texture is minified and magnified.
	MinFilter, MagFilter FilterMode
	// MipmapFilter chooses between mip levels; FilterUnspecified uses level 0 only.
	MipmapFilter FilterMode
	// AddressU and AddressV are the modes of the two directions.
	AddressU, AddressV AddressMode
	// AddressW is the mode of the third direction of a 3D texture.
	AddressW AddressMode
}

// VertexFormat is the type of a vertex attribute: WebGPU's vertex formats (SPEC §27.1). The zero
// value is VertexFormatUnspecified.
//
// Flutter: no counterpart
type VertexFormat int

// The vertex formats. A format with unorm or snorm gives the shader floats from 0 to 1 or from
// -1 to 1; with uint or sint, integers.
const (
	// VertexFormatUnspecified is the zero value: no format.
	//
	// Flutter: no counterpart
	VertexFormatUnspecified VertexFormat = iota
	// VertexFloat32 is one float32.
	//
	// Flutter: no counterpart
	VertexFloat32
	// VertexFloat32x2 is two float32.
	//
	// Flutter: no counterpart
	VertexFloat32x2
	// VertexFloat32x3 is three float32.
	//
	// Flutter: no counterpart
	VertexFloat32x3
	// VertexFloat32x4 is four float32.
	//
	// Flutter: no counterpart
	VertexFloat32x4
	// VertexUnorm8x4 is four bytes read as numbers from 0 to 1.
	//
	// Flutter: no counterpart
	VertexUnorm8x4
	// VertexUint32 is one uint32.
	//
	// Flutter: no counterpart
	VertexUint32
	// VertexUint32x2 is two uint32.
	//
	// Flutter: no counterpart
	VertexUint32x2
	// VertexUint32x3 is three uint32.
	//
	// Flutter: no counterpart
	VertexUint32x3
	// VertexUint32x4 is four uint32.
	//
	// Flutter: no counterpart
	VertexUint32x4
	// VertexSint32 is one int32.
	//
	// Flutter: no counterpart
	VertexSint32
	// VertexSint32x2 is two int32.
	//
	// Flutter: no counterpart
	VertexSint32x2
	// VertexSint32x3 is three int32.
	//
	// Flutter: no counterpart
	VertexSint32x3
	// VertexSint32x4 is four int32.
	//
	// Flutter: no counterpart
	VertexSint32x4
	// VertexUint8x2 is two bytes read as integers.
	//
	// Flutter: no counterpart
	VertexUint8x2
	// VertexUint8x4 is four bytes read as integers.
	//
	// Flutter: no counterpart
	VertexUint8x4
	// VertexSint8x2 is two signed bytes read as integers.
	//
	// Flutter: no counterpart
	VertexSint8x2
	// VertexSint8x4 is four signed bytes read as integers.
	//
	// Flutter: no counterpart
	VertexSint8x4
	// VertexUnorm8x2 is two bytes read as numbers from 0 to 1.
	//
	// Flutter: no counterpart
	VertexUnorm8x2
	// VertexSnorm8x2 is two signed bytes read as numbers from -1 to 1.
	//
	// Flutter: no counterpart
	VertexSnorm8x2
	// VertexSnorm8x4 is four signed bytes read as numbers from -1 to 1.
	//
	// Flutter: no counterpart
	VertexSnorm8x4
	// VertexUint16x2 is two uint16 read as integers.
	//
	// Flutter: no counterpart
	VertexUint16x2
	// VertexUint16x4 is four uint16 read as integers.
	//
	// Flutter: no counterpart
	VertexUint16x4
	// VertexSint16x2 is two int16 read as integers.
	//
	// Flutter: no counterpart
	VertexSint16x2
	// VertexSint16x4 is four int16 read as integers.
	//
	// Flutter: no counterpart
	VertexSint16x4
	// VertexUnorm16x2 is two uint16 read as numbers from 0 to 1.
	//
	// Flutter: no counterpart
	VertexUnorm16x2
	// VertexUnorm16x4 is four uint16 read as numbers from 0 to 1.
	//
	// Flutter: no counterpart
	VertexUnorm16x4
	// VertexSnorm16x2 is two int16 read as numbers from -1 to 1.
	//
	// Flutter: no counterpart
	VertexSnorm16x2
	// VertexSnorm16x4 is four int16 read as numbers from -1 to 1.
	//
	// Flutter: no counterpart
	VertexSnorm16x4
	// VertexFloat16x2 is two half floats.
	//
	// Flutter: no counterpart
	VertexFloat16x2
	// VertexFloat16x4 is four half floats.
	//
	// Flutter: no counterpart
	VertexFloat16x4
)

// Size returns the size of the attribute in bytes; 0 for an unspecified format.
//
// Flutter: no counterpart
func (f VertexFormat) Size() int {
	switch f {
	case VertexFloat32, VertexUint32, VertexSint32, VertexUnorm8x4, VertexUint8x4, VertexSint8x4, VertexSnorm8x4,
		VertexUint16x2, VertexSint16x2, VertexUnorm16x2, VertexSnorm16x2, VertexFloat16x2:
		return 4
	case VertexFloat32x2, VertexUint32x2, VertexSint32x2, VertexUint16x4, VertexSint16x4, VertexUnorm16x4,
		VertexSnorm16x4, VertexFloat16x4:
		return 8
	case VertexFloat32x3, VertexUint32x3, VertexSint32x3:
		return 12
	case VertexFloat32x4, VertexUint32x4, VertexSint32x4:
		return 16
	case VertexUint8x2, VertexSint8x2, VertexUnorm8x2, VertexSnorm8x2:
		return 2
	}
	return 0
}

// VertexAttribute is one input of the vertex shader, in a buffer.
//
// Flutter: no counterpart
type VertexAttribute struct {
	// Location is the @location of the shader's input.
	Location int
	// Format is the type of the data.
	Format VertexFormat
	// Offset is where the data starts in an element of the buffer, in bytes.
	Offset int
}

// VertexBufferLayout is how a vertex buffer is read.
//
// Flutter: no counterpart
type VertexBufferLayout struct {
	// Stride is the size of an element in bytes.
	Stride int
	// PerInstance makes the buffer advance once for each instance instead of each
	// vertex.
	PerInstance bool
	// Attributes are the inputs that come from this buffer.
	Attributes []VertexAttribute
}

// BlendFactor is a factor of the blend equation. The zero value is
// BlendFactorUnspecified, which means BlendOne.
//
// Flutter: no counterpart
type BlendFactor int

// The blend factors.
const (
	// BlendFactorUnspecified is the zero value and means BlendOne.
	//
	// Flutter: no counterpart
	BlendFactorUnspecified BlendFactor = iota
	// BlendZero is 0.
	//
	// Flutter: no counterpart
	BlendZero
	// BlendOne is 1.
	//
	// Flutter: no counterpart
	BlendOne
	// BlendSrcAlpha is the alpha of the source.
	//
	// Flutter: no counterpart
	BlendSrcAlpha
	// BlendOneMinusSrcAlpha is 1 minus the alpha of the source.
	//
	// Flutter: no counterpart
	BlendOneMinusSrcAlpha
	// BlendDstAlpha is the alpha of the destination.
	//
	// Flutter: no counterpart
	BlendDstAlpha
	// BlendOneMinusDstAlpha is 1 minus the alpha of the destination.
	//
	// Flutter: no counterpart
	BlendOneMinusDstAlpha
)

// BlendState is the blend equation of a pipeline: source * SrcFactor + destination *
// DstFactor, for color and alpha. The zero value is no blending: the source replaces
// the destination.
//
// Flutter: no counterpart
type BlendState struct {
	// Enabled turns blending on; off, the source replaces the destination.
	Enabled bool
	// SrcFactor and DstFactor are the factors for the color and the alpha.
	SrcFactor, DstFactor BlendFactor
}

// PrimitiveTopology says how the vertices of a draw make primitives. The zero value is
// TopologyTriangleList. A strip is not restarted by an index: the primitive restart is off.
//
// Flutter: no counterpart
type PrimitiveTopology int

// The topologies.
const (
	// TopologyTriangleList makes a triangle of each three vertices.
	//
	// Flutter: no counterpart
	TopologyTriangleList PrimitiveTopology = iota
	// TopologyTriangleStrip makes a triangle of each vertex and the two before it.
	//
	// Flutter: no counterpart
	TopologyTriangleStrip
	// TopologyLineList makes a line of each two vertices.
	//
	// Flutter: no counterpart
	TopologyLineList
	// TopologyLineStrip makes a line of each vertex and the one before it.
	//
	// Flutter: no counterpart
	TopologyLineStrip
	// TopologyPointList makes a point of each vertex.
	//
	// Flutter: no counterpart
	TopologyPointList
)

// CullMode says which triangles are not drawn. The zero value is CullNone.
//
// Flutter: no counterpart
type CullMode int

// The cull modes.
const (
	// CullNone draws every triangle.
	//
	// Flutter: no counterpart
	CullNone CullMode = iota
	// CullFront leaves out the triangles that face the viewer.
	//
	// Flutter: no counterpart
	CullFront
	// CullBack leaves out the triangles that face away from the viewer.
	//
	// Flutter: no counterpart
	CullBack
)

// FrontFace says which winding of a triangle on the screen faces the viewer, in the clip space
// of SPEC §27.1 (y up). The zero value is FrontFaceCCW.
//
// Flutter: no counterpart
type FrontFace int

// The windings.
const (
	// FrontFaceCCW is a triangle whose vertices go counter-clockwise on the screen.
	//
	// Flutter: no counterpart
	FrontFaceCCW FrontFace = iota
	// FrontFaceCW is a triangle whose vertices go clockwise on the screen.
	//
	// Flutter: no counterpart
	FrontFaceCW
)

// CompareFunc is the comparison of a depth or stencil test: the new value against the stored one.
// The zero value is CompareUnspecified, which means CompareAlways.
//
// Flutter: no counterpart
type CompareFunc int

// The comparison functions.
const (
	// CompareUnspecified is the zero value and means CompareAlways.
	//
	// Flutter: no counterpart
	CompareUnspecified CompareFunc = iota
	// CompareNever never passes.
	//
	// Flutter: no counterpart
	CompareNever
	// CompareLess passes when the new value is less.
	//
	// Flutter: no counterpart
	CompareLess
	// CompareEqual passes when the values are equal.
	//
	// Flutter: no counterpart
	CompareEqual
	// CompareLessEqual passes when the new value is less or equal.
	//
	// Flutter: no counterpart
	CompareLessEqual
	// CompareGreater passes when the new value is greater: reversed-Z.
	//
	// Flutter: no counterpart
	CompareGreater
	// CompareNotEqual passes when the values differ.
	//
	// Flutter: no counterpart
	CompareNotEqual
	// CompareGreaterEqual passes when the new value is greater or equal.
	//
	// Flutter: no counterpart
	CompareGreaterEqual
	// CompareAlways always passes.
	//
	// Flutter: no counterpart
	CompareAlways
)

// StencilOp is what a stencil test does to the stencil value. The zero value is StencilKeep.
//
// Flutter: no counterpart
type StencilOp int

// The stencil operations.
const (
	// StencilKeep keeps the value.
	//
	// Flutter: no counterpart
	StencilKeep StencilOp = iota
	// StencilZero sets the value to 0.
	//
	// Flutter: no counterpart
	StencilZero
	// StencilReplace sets the value to the reference of the pass (Pass.SetStencilReference).
	//
	// Flutter: no counterpart
	StencilReplace
	// StencilInvert inverts the bits of the value.
	//
	// Flutter: no counterpart
	StencilInvert
	// StencilIncrementClamp adds 1 up to the largest value.
	//
	// Flutter: no counterpart
	StencilIncrementClamp
	// StencilDecrementClamp subtracts 1 down to 0.
	//
	// Flutter: no counterpart
	StencilDecrementClamp
	// StencilIncrementWrap adds 1 and wraps to 0.
	//
	// Flutter: no counterpart
	StencilIncrementWrap
	// StencilDecrementWrap subtracts 1 and wraps to the largest value.
	//
	// Flutter: no counterpart
	StencilDecrementWrap
)

// StencilFaceState is the stencil test of the front or the back faces of triangles. The zero
// value always passes and keeps the value.
//
// Flutter: no counterpart
type StencilFaceState struct {
	// Compare is the test of the reference against the stored value.
	Compare CompareFunc
	// FailOp is what happens when the stencil test fails.
	FailOp StencilOp
	// DepthFailOp is what happens when the stencil test passes and the depth test fails.
	DepthFailOp StencilOp
	// PassOp is what happens when both pass.
	PassOp StencilOp
}

// DepthStencilState is the depth and stencil state of a pipeline. The zero value is no depth-stencil
// attachment.
//
// Flutter: no counterpart
type DepthStencilState struct {
	// Format is the format of the depth-stencil attachment of the passes that use the pipeline;
	// TextureFormatUnspecified means the pipeline has none.
	Format TextureFormat
	// DepthWrite stores the depth of the fragments that pass.
	DepthWrite bool
	// DepthCompare is the depth test.
	DepthCompare CompareFunc
	// StencilFront and StencilBack are the stencil tests of the two kinds of faces. A format
	// without a stencil ignores them.
	StencilFront, StencilBack StencilFaceState
	// StencilReadMask and StencilWriteMask are the bits that the test reads and the operations
	// write; 0 means all the bits.
	StencilReadMask, StencilWriteMask uint32
	// DepthBias is added to the depth of each fragment, in units of the smallest difference of
	// the depth format; DepthBiasSlopeScale times the slope of the triangle is added too, and the
	// sum is at most DepthBiasClamp when that is not 0. Only triangles have a bias.
	DepthBias           int32
	DepthBiasSlopeScale float32
	DepthBiasClamp      float32
}

// ColorWriteMask says which channels of a color attachment a pipeline writes; values are combined
// with |. The zero value is ColorWriteAll.
//
// Flutter: no counterpart
type ColorWriteMask int

// The channels.
const (
	// ColorWriteAll writes every channel; it is the zero value.
	//
	// Flutter: no counterpart
	ColorWriteAll ColorWriteMask = 0
	// ColorWriteRed writes the red channel.
	//
	// Flutter: no counterpart
	ColorWriteRed ColorWriteMask = 1 << (iota - 1)
	// ColorWriteGreen writes the green channel.
	//
	// Flutter: no counterpart
	ColorWriteGreen
	// ColorWriteBlue writes the blue channel.
	//
	// Flutter: no counterpart
	ColorWriteBlue
	// ColorWriteAlpha writes the alpha channel.
	//
	// Flutter: no counterpart
	ColorWriteAlpha
	// ColorWriteNone writes no channel.
	//
	// Flutter: no counterpart
	ColorWriteNone
)

// Bits returns the channels as the bits 1 (red), 2 (green), 4 (blue) and 8 (alpha) that every
// backend uses: all of them for ColorWriteAll, none for ColorWriteNone.
//
// Flutter: no counterpart
func (m ColorWriteMask) Bits() uint32 {
	switch m {
	case ColorWriteAll:
		return 15
	case ColorWriteNone:
		return 0
	}
	return uint32(m) & 15
}

// ColorTargetState is the format, blend state and write mask of one color attachment of a
// pipeline.
//
// Flutter: no counterpart
type ColorTargetState struct {
	// Format is the format of the attachment.
	Format TextureFormat
	// Blend is the blend state of the attachment.
	Blend BlendState
	// WriteMask says which channels are written.
	WriteMask ColorWriteMask
}

// PipelineDesc describes a render pipeline: a generated shader, its entry points,
// the vertex layout, the blend state, the format of the targets and the state of the
// primitives, the depth and stencil tests and the multisampling.
//
// Flutter: no counterpart
type PipelineDesc struct {
	// Shader is a generated shader.
	Shader *Shader
	// VertexEntry and FragmentEntry are the names of its entry points.
	VertexEntry, FragmentEntry string
	// VertexBuffers are the layouts of the vertex buffers, by slot.
	VertexBuffers []VertexBufferLayout
	// Blend is the blend state of the one color target of the 2D renderer; it is the state of
	// target 0 when ColorTargets is empty.
	Blend BlendState
	// Target is the format of the one color target of the 2D renderer; it is the format of
	// target 0 when ColorTargets is empty.
	Target TextureFormat
	// ColorTargets are the color attachments of the passes that use the pipeline, 1 to
	// Limits.MaxColorAttachments of them; when empty, the pipeline has the one target of Target and
	// Blend. A pipeline made with ColorTargets, a Topology other than the triangle list, a
	// culling, a depth-stencil state or SampleCount is a pipeline of the 3D API; the clip space is
	// the same for every pipeline (SPEC §27.1).
	ColorTargets []ColorTargetState
	// Topology says how the vertices make primitives.
	Topology PrimitiveTopology
	// CullMode says which triangles are left out.
	CullMode CullMode
	// FrontFace says which winding faces the viewer.
	FrontFace FrontFace
	// DepthStencil is the depth and stencil state; the zero value has no depth-stencil
	// attachment.
	DepthStencil DepthStencilState
	// SampleCount is the number of samples of the attachments: 1 or 4; 0 means 1.
	SampleCount int
}

// Targets returns the color targets of the pipeline: ColorTargets, or the one target of Target and
// Blend.
//
// Flutter: no counterpart
func (d PipelineDesc) Targets() []ColorTargetState {
	if len(d.ColorTargets) > 0 {
		return d.ColorTargets
	}
	return []ColorTargetState{{Format: d.Target, Blend: d.Blend}}
}

// Samples returns SampleCount, with 0 meaning 1.
//
// Flutter: no counterpart
func (d PipelineDesc) Samples() int {
	if d.SampleCount < 1 {
		return 1
	}
	return d.SampleCount
}

// Binding is one resource of a bind group: the entry for @binding(Binding).
//
// Flutter: no counterpart
type Binding struct {
	// Binding is the number of the shader's @binding.
	Binding int
	// Buffer, with Offset and Size, is a uniform buffer or a storage buffer; Size 0 means up to the end.
	// A uniform binding gets its offset from SetBindGroup too; a storage binding's Offset is a multiple of
	// Limits.StorageOffsetAlignment.
	Buffer Buffer
	// Offset is where the binding starts in Buffer, in bytes.
	Offset int
	// Size is the size of the binding in bytes.
	Size int
	// Texture is a sampled texture, or a storage texture (TextureStorage) for a storage binding.
	Texture Texture
	// Sampler is a sampler.
	Sampler Sampler
}

// LoadOp is what a pass does with the target's contents before it draws. The zero
// value is LoadClear.
//
// Flutter: no counterpart
type LoadOp int

// The load operations.
const (
	// LoadClear fills the target with the clear color.
	//
	// Flutter: no counterpart
	LoadClear LoadOp = iota
	// LoadLoad keeps the target's contents.
	//
	// Flutter: no counterpart
	LoadLoad
	// LoadDontCare leaves the target's contents undefined, for a pass that draws every pixel of
	// it: a tile-based GPU (Adreno, Mali, PowerVR) then neither clears the tile nor reads it
	// from memory (SPEC §24.4). A backend without the operation clears.
	//
	// Flutter: no counterpart
	LoadDontCare
)

// StoreOp is what a pass does with the pixels it drew when it ends. The zero value is StoreStore.
//
// Flutter: no counterpart
type StoreOp int

// The store operations.
const (
	// StoreStore keeps what the pass drew.
	//
	// Flutter: no counterpart
	StoreStore StoreOp = iota
	// StoreDiscard leaves the attachment undefined, for a target that is only resolved or only
	// read inside the pass (a depth buffer that nothing reads later): a tile-based GPU then does
	// not write it to memory.
	//
	// Flutter: no counterpart
	StoreDiscard
)

// ColorAttachment is one color attachment of a render pass.
//
// Flutter: no counterpart
type ColorAttachment struct {
	// Target is the texture drawn into: a 2D texture, or a view of one layer or level made by
	// Layer or Level. It has TextureRenderTarget.
	Target Texture
	// Resolve, when set, is a texture of one sample, of the size and format of Target, that
	// receives the average of the samples at the end of the pass; Target is multisampled then.
	Resolve Texture
	// Load is what happens to the contents of Target at the start.
	Load LoadOp
	// Store is what happens to the contents of Target at the end. With Resolve, StoreDiscard
	// keeps only the resolved pixels.
	Store StoreOp
	// Clear is the color of LoadClear, converted to the type of the format (the integers of a
	// Uint format are the whole numbers).
	Clear [4]float64
}

// DepthAttachment is the depth-stencil attachment of a render pass.
//
// Flutter: no counterpart
type DepthAttachment struct {
	// Target is the texture drawn into, of a depth format, with TextureRenderTarget.
	Target Texture
	// DepthLoad and DepthStore are what happens to the depth at the start and at the end.
	DepthLoad  LoadOp
	DepthStore StoreOp
	// ClearDepth is the depth of LoadClear, from 0 to 1.
	ClearDepth float64
	// StencilLoad, StencilStore and ClearStencil are the same for the stencil; they are ignored
	// by a format without one.
	StencilLoad  LoadOp
	StencilStore StoreOp
	ClearStencil uint32
}

// PassDesc describes a render pass: 1 to Limits.MaxColorAttachments color attachments and at most
// one depth-stencil attachment. They all have the same size and sample count.
//
// Flutter: no counterpart
type PassDesc struct {
	// Color are the color attachments.
	Color []ColorAttachment
	// Depth is the depth-stencil attachment; nil for none.
	Depth *DepthAttachment
	// Timestamps, when set, is where the pass writes its timestamps; it needs Features.TimestampQuery.
	Timestamps *PassTimestamps
}

// Buffer is a GPU buffer.
//
// Flutter: no counterpart
type Buffer interface {
	// Size is the size in bytes.
	Size() int
	// Usage is what the buffer was made for.
	Usage() BufferUsage
	// Destroy releases the buffer. The GPU finishes the frames that use it first.
	Destroy()
}

// Texture is a GPU texture.
//
// Flutter: no counterpart
type Texture interface {
	// Desc is what the texture was made with, normalized (Normalized).
	Desc() TextureDesc
	// Level returns a texture that is the mip level i of this one: it has the size of that
	// level and one level, and a pass can draw into it and a bind group can sample it, which
	// is how the downsampling pass makes the mipmaps (SPEC §9.3). The texture it returns
	// belongs to this one and goes away with it. A 2D array, cube or 3D texture keeps its kind
	// and its layers in the view of a level, and a pass can draw into it only through Layer. Level(0)
	// of a texture with one level is the texture itself.
	Level(i int) Texture
	// Layer returns a 2D texture over one layer (the face of a cube, in the order of
	// TextureKindCube) of the mip level of a 2D array or cube texture: a pass can draw into it, a
	// bind group can sample it as a 2D texture, and ReadTextureBytes can read it. It belongs to
	// this texture and goes away with it. A 2D texture has the one layer 0, which is Level(level).
	// A 3D texture has no layers: the call panics.
	Layer(level, layer int) Texture
	// Destroy releases the texture. The GPU finishes the frames that use it first.
	Destroy()
}

// Sampler is a GPU sampler.
//
// Flutter: no counterpart
type Sampler interface {
	// Destroy releases the sampler.
	Destroy()
}

// Pipeline is a GPU render pipeline.
//
// Flutter: no counterpart
type Pipeline interface {
	// Desc is what the pipeline was made with.
	Desc() PipelineDesc
	// Destroy releases the pipeline.
	Destroy()
}

// BindGroup is a set of resources for one bind group of a pipeline.
//
// Flutter: no counterpart
type BindGroup interface {
	// Destroy releases the bind group.
	Destroy()
}

// Device is a GPU, as the explicit renderer sees it. Vulkan and WebGPU implement
// it within the design limits (SPEC §9.3). Every method runs on the UI thread. The
// methods that make resources fail with GPUOutOfMemory or GPUDeviceLost; a call that
// breaks a design limit panics with GPULimitExceeded in debug builds.
//
// Flutter: no counterpart
type Device interface {
	// Limits are the design limits.
	Limits() Limits
	// Features are the optional features of the device.
	Features() Features
	// NewBuffer makes a buffer of size bytes.
	NewBuffer(size int, usage BufferUsage) (Buffer, error)
	// NewTexture makes a texture.
	NewTexture(desc TextureDesc) (Texture, error)
	// NewSampler makes a sampler.
	NewSampler(desc SamplerDesc) (Sampler, error)
	// NewPipeline makes a pipeline from a generated shader.
	NewPipeline(desc PipelineDesc) (Pipeline, error)
	// NewComputePipeline makes a compute pipeline from the compute entry point of a generated shader.
	NewComputePipeline(desc ComputePipelineDesc) (ComputePipeline, error)
	// NewBindGroup makes the resources of the bind group number group of the
	// pipeline.
	NewBindGroup(p Pipeline, group int, entries []Binding) (BindGroup, error)
	// NewComputeBindGroup makes the resources of the bind group number group of the compute
	// pipeline.
	NewComputeBindGroup(p ComputePipeline, group int, entries []Binding) (BindGroup, error)
	// NewQuerySet makes a set of count timestamps; it fails with GPUQueryUnsupported when
	// Features.TimestampQuery is false.
	NewQuerySet(count int) (QuerySet, error)
	// WriteBuffer copies data into the buffer at offset. The data is copied before
	// the call returns.
	WriteBuffer(b Buffer, offset int, data []byte)
	// WriteTexture copies the rows of data, stride bytes apart, into the rectangle r
	// of the level of the texture.
	WriteTexture(t Texture, level int, r image.Rectangle, data []byte, stride int)
	// WriteTextureRegion copies data into a box of a 2D array, cube or 3D texture (or a 2D texture,
	// with one layer): the rows of data are bytesPerRow apart and the images (the layers or slices)
	// rowsPerImage rows apart. The data is copied before the call returns.
	WriteTextureRegion(t Texture, region TextureRegion, data []byte, bytesPerRow, rowsPerImage int)
	// NewSwapchain makes the swapchain of a window.
	NewSwapchain(h gpu.SurfaceHandle) (Swapchain, error)
	// Begin starts an encoder: the commands of one submission. A frame may have several, one after
	// the other (each ends with Submit before the next Begin); the GPU runs them in order on the
	// device's one queue. The 2D renderer's frame is one encoder, submitted last.
	Begin() (Encoder, error)
	// Close releases the device and everything made with it.
	Close() error
}

// Encoder records the commands of one frame.
//
// Flutter: no counterpart
type Encoder interface {
	// BeginPass starts a pass that draws into the target, which must have
	// TextureRenderTarget. The previous pass must have ended. It is BeginRenderPass with one color
	// attachment that is stored.
	BeginPass(target Texture, load LoadOp, clear ui.Color) Pass
	// BeginRenderPass starts a pass with the attachments of desc. The previous pass must have
	// ended.
	BeginRenderPass(desc PassDesc) Pass
	// CopyTexture copies the rectangle r of src to dst at the point at.
	CopyTexture(src Texture, r image.Rectangle, dst Texture, at image.Point)
	// ReadTexture reads the rectangle r of src. done runs once the GPU has finished,
	// never inside the frame that asked.
	ReadTexture(src Texture, r image.Rectangle, done func(*image.RGBA, error))
	// ReadTextureBytes reads the rectangle r of a 2D texture (or a Level or Layer view) as the
	// bytes of its format: the rows of r.Dx() pixels, packed, so that rowBytes is r.Dx() times the
	// bytes of a pixel. It works for every format that CanBeCopied. done runs once the GPU has
	// finished, never inside the frame that asked.
	ReadTextureBytes(src Texture, r image.Rectangle, done func(data []byte, rowBytes int, err error))
	// BeginComputePass starts a compute pass. The previous pass must have ended.
	BeginComputePass(desc ComputePassDesc) ComputePass
	// CopyBuffer copies size bytes of src, from srcOffset, to dst at dstOffset. The buffers have
	// BufferCopySource and BufferCopyDestination; the offsets and the size are multiples of 4.
	CopyBuffer(src Buffer, srcOffset int, dst Buffer, dstOffset, size int)
	// CopyBufferToTexture copies the box region of a level of dst from the rows in src, laid out as
	// layout says. The box is inside the level. Every texture can be the destination but a depth
	// format and a multisampled texture.
	CopyBufferToTexture(src Buffer, layout BufferLayout, dst Texture, region TextureRegion)
	// CopyTextureToBuffer copies the box region of a level of src, a texture with TextureCopySource
	// (or TextureRenderTarget) of a format that CanBeCopied, into dst as layout says.
	CopyTextureToBuffer(src Texture, region TextureRegion, dst Buffer, layout BufferLayout)
	// ReadBuffer reads size bytes of src from offset: done runs once the GPU has finished, never
	// inside the frame that asked, with a copy of the bytes. src has BufferCopySource.
	ReadBuffer(src Buffer, offset, size int, done func(data []byte, err error))
	// ResolveQuerySet writes the count timestamps of set from first, as 64 bit unsigned integers
	// in ticks (Features.TimestampPeriod), to dst at dstOffset, a multiple of
	// Limits.StorageOffsetAlignment. dst has BufferQueryResolve.
	ResolveQuerySet(set QuerySet, first, count int, dst Buffer, dstOffset int)
	// OnComplete runs fn once the GPU has finished the work of this encoder, never inside the frame
	// that asked: Submit runs it before it returns on a device that waits for the GPU, and a device
	// that cannot (WebGPU) runs it later, on the UI thread. It must be called before Submit.
	OnComplete(fn func())
	// Submit sends the encoder's commands to the GPU.
	Submit() error
}

// Pass records the draws of one render pass.
//
// Flutter: no counterpart
type Pass interface {
	// SetPipeline chooses the pipeline of the next draws.
	SetPipeline(Pipeline)
	// SetBindGroup binds the group number group, with one dynamic offset for each
	// uniform buffer in it, in binding order, each a multiple of
	// Limits.UniformOffsetAlignment.
	SetBindGroup(group int, g BindGroup, dynamicOffsets ...uint32)
	// SetVertexBuffer binds a vertex buffer to a slot, starting at offset bytes.
	SetVertexBuffer(slot int, b Buffer, offset int)
	// SetIndexBuffer binds the index buffer of 16 bit indices, starting at offset bytes.
	SetIndexBuffer(b Buffer, offset int)
	// SetIndexBuffer32 binds the index buffer of 32 bit indices, starting at offset bytes.
	SetIndexBuffer32(b Buffer, offset int)
	// SetScissor limits the draws to the rectangle, in pixels of the target.
	SetScissor(r image.Rectangle)
	// SetStencilReference sets the reference value of the stencil tests and of StencilReplace; it
	// is 0 at the start of a pass.
	SetStencilReference(ref uint32)
	// Draw draws vertices, instances times.
	Draw(vertices, instances, firstVertex, firstInstance int)
	// DrawIndexed draws indices, instances times.
	DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int)
	// DrawIndirect draws with the arguments in b at offset: four 32 bit unsigned integers, the
	// vertex count, the instance count, the first vertex and the first instance. b has BufferIndirect;
	// offset is a multiple of 4. The first vertex and the first instance must be 0 for a shader that
	// reads vertex_index or instance_index (the arguments cannot be checked by the CPU).
	DrawIndirect(b Buffer, offset int)
	// DrawIndexedIndirect draws with the arguments in b at offset: five 32 bit integers, the index
	// count, the instance count, the first index, the base vertex (signed) and the first instance.
	DrawIndexedIndirect(b Buffer, offset int)
	// End finishes the pass.
	End()
}

// Rotation is a rotation by quarter turns, clockwise: a surface's pre-transform (SPEC §24.4).
//
// Flutter: no counterpart
type Rotation int

// The rotations of a pre-transform.
const (
	Rotation0 Rotation = iota
	Rotation90
	Rotation180
	Rotation270
)

// Swapchain is the images of a window.
//
// Flutter: no counterpart
type Swapchain interface {
	// Configure sets the size of the images, in device pixels of the window as it is seen: when
	// the swapchain has a pre-transform of a quarter turn, its images have the size turned.
	Configure(width, height int) error
	// PreRotation is the rotation that the window system expects of the images (Android's
	// currentTransform): the frame is drawn into the image rotated clockwise by it, and the
	// presentation engine then does not rotate it again. It is valid after Configure, and
	// Rotation0 on every window system but Android's.
	PreRotation() Rotation
	// Stale says that the window system turned the surface (Android's currentTransform changed,
	// which it reports once the surface is on screen, or when the device is turned without a
	// change of size): the image that was just presented was drawn for the old pre-transform, and the
	// renderer draws it again, whole and without building the scene again, into a swapchain that
	// Configure makes. It is cleared by Configure.
	Stale() bool
	// Acquire returns the image to draw into next. When it is out of date, Configure
	// again and retry once.
	Acquire() (Texture, error)
	// Present shows the image, telling the compositor which rectangles changed (nil
	// means all).
	Present(damage []image.Rectangle) error
	// Destroy releases the swapchain.
	Destroy()
}
