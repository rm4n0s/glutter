package metal

import "github.com/rm4n0s/glutter/internal/gpu/explicit"

// The mappings from the descriptions of the explicit renderer to the values of Metal (SPEC §29.5). They are
// in a file with no OS tag, so that tests on any host check them against the numbers of the headers of
// Metal (MTLPixelFormat.h, MTLVertexDescriptor.h and the others).

// The values of Metal's enumerations that the device uses. Their type is NSUInteger (uint64 on the 64-bit
// targets), as purego passes it.
const (
	pixelFormatInvalid           uint64 = 0
	pixelFormatR8Unorm           uint64 = 10
	pixelFormatR32Uint           uint64 = 53
	pixelFormatR32Float          uint64 = 55
	pixelFormatRGBA8Unorm        uint64 = 70
	pixelFormatBGRA8Unorm        uint64 = 80
	pixelFormatRG32Uint          uint64 = 103
	pixelFormatRGBA16Float       uint64 = 115
	pixelFormatRGBA32Float       uint64 = 125
	pixelFormatDepth32Float      uint64 = 252
	pixelFormatDepth32FloatStenc uint64 = 260
)

// The texture types, usages and storage modes (MTLTextureType, MTLTextureUsage, MTLStorageMode).
const (
	textureType2D            uint64 = 2
	textureType2DArray       uint64 = 3
	textureType2DMultisample uint64 = 4
	textureTypeCube          uint64 = 5
	textureType3D            uint64 = 7

	textureUsageShaderRead      uint64 = 1
	textureUsageShaderWrite     uint64 = 2
	textureUsageRenderTarget    uint64 = 4
	textureUsagePixelFormatView uint64 = 16

	storageShared  uint64 = 0
	storageManaged uint64 = 1
	storagePrivate uint64 = 2
)

// The resource options of a buffer: the storage mode in bits 4 to 7 and the CPU cache mode in bits 0 to 3
// (MTLResourceOptions).
const (
	resourceCPUCacheWriteCombined uint64 = 1
	resourceStorageShift                 = 4
)

// resourceOptions makes the options of a buffer of the storage mode; writeCombined says that the CPU only
// writes it.
func resourceOptions(storage uint64, writeCombined bool) uint64 {
	o := storage << resourceStorageShift
	if writeCombined {
		o |= resourceCPUCacheWriteCombined
	}
	return o
}

// pixelFormat maps a texture format; an unspecified format is MTLPixelFormatInvalid. Depth24PlusStencil8 is
// the 32-bit depth with an 8-bit stencil: Apple GPUs have no 24-bit depth.
func pixelFormat(f explicit.TextureFormat) uint64 {
	switch f {
	case explicit.TextureFormatRGBA8Unorm:
		return pixelFormatRGBA8Unorm
	case explicit.TextureFormatBGRA8Unorm:
		return pixelFormatBGRA8Unorm
	case explicit.TextureFormatR8Unorm:
		return pixelFormatR8Unorm
	case explicit.TextureFormatRGBA16Float:
		return pixelFormatRGBA16Float
	case explicit.TextureFormatRGBA32Float:
		return pixelFormatRGBA32Float
	case explicit.TextureFormatR32Float:
		return pixelFormatR32Float
	case explicit.TextureFormatR32Uint:
		return pixelFormatR32Uint
	case explicit.TextureFormatRG32Uint:
		return pixelFormatRG32Uint
	case explicit.TextureFormatDepth32Float:
		return pixelFormatDepth32Float
	case explicit.TextureFormatDepth24PlusStencil8:
		return pixelFormatDepth32FloatStenc
	}
	return pixelFormatInvalid
}

// textureType maps the shape of a texture: a multisampled texture is its own type.
func textureType(k explicit.TextureKind, samples int) uint64 {
	if samples > 1 {
		return textureType2DMultisample
	}
	switch k {
	case explicit.TextureKind2DArray:
		return textureType2DArray
	case explicit.TextureKindCube:
		return textureTypeCube
	case explicit.TextureKind3D:
		return textureType3D
	}
	return textureType2D
}

// textureUsage maps the uses of a texture. A render target of a format that a shader reads is both.
func textureUsage(u explicit.TextureUsage) uint64 {
	var out uint64
	if u&explicit.TextureSampled != 0 {
		out |= textureUsageShaderRead
	}
	if u&explicit.TextureRenderTarget != 0 {
		out |= textureUsageRenderTarget
	}
	if u&explicit.TextureStorage != 0 {
		out |= textureUsageShaderWrite | textureUsageShaderRead
	}
	return out
}

// The filters and address modes of a sampler (MTLSamplerMinMagFilter, MTLSamplerMipFilter,
// MTLSamplerAddressMode).
const (
	filterNearest uint64 = 0
	filterLinear  uint64 = 1

	mipNotMipmapped uint64 = 0
	mipNearest      uint64 = 1
	mipLinear       uint64 = 2

	addressClampToEdge  uint64 = 0
	addressRepeat       uint64 = 2
	addressMirrorRepeat uint64 = 3
)

// minMagFilter maps the minification or magnification filter; the unspecified one is nearest.
func minMagFilter(f explicit.FilterMode) uint64 {
	if f == explicit.FilterLinear {
		return filterLinear
	}
	return filterNearest
}

// mipFilter maps the filter between mip levels; the unspecified one reads level 0 only (SPEC §9.3).
func mipFilter(f explicit.FilterMode) uint64 {
	switch f {
	case explicit.FilterLinear:
		return mipLinear
	case explicit.FilterNearest:
		return mipNearest
	}
	return mipNotMipmapped
}

// addressMode maps an address mode; the unspecified one is clamp.
func addressMode(m explicit.AddressMode) uint64 {
	switch m {
	case explicit.AddressRepeat:
		return addressRepeat
	case explicit.AddressMirror:
		return addressMirrorRepeat
	}
	return addressClampToEdge
}

// The blend factors and the blend operation (MTLBlendFactor, MTLBlendOperation).
const (
	blendZero                uint64 = 0
	blendOne                 uint64 = 1
	blendSourceAlpha         uint64 = 4
	blendOneMinusSourceAlpha uint64 = 5
	blendDestAlpha           uint64 = 8
	blendOneMinusDestAlpha   uint64 = 9
	blendOpAdd               uint64 = 0
)

// blendFactor maps a blend factor. As in the other backends, the unspecified source factor is one and the
// unspecified destination factor is zero (explicit.BlendState).
func blendFactor(f explicit.BlendFactor, destination bool) uint64 {
	switch f {
	case explicit.BlendZero:
		return blendZero
	case explicit.BlendOne:
		return blendOne
	case explicit.BlendSrcAlpha:
		return blendSourceAlpha
	case explicit.BlendOneMinusSrcAlpha:
		return blendOneMinusSourceAlpha
	case explicit.BlendDstAlpha:
		return blendDestAlpha
	case explicit.BlendOneMinusDstAlpha:
		return blendOneMinusDestAlpha
	}
	if destination {
		return blendZero
	}
	return blendOne
}

// blendOf is the blend of one color target: whether it is on and the factors of the color and the alpha,
// which are alike, with the add operation (SPEC §9.3). With blending off the source replaces the
// destination.
type blendOf struct {
	enabled            bool
	srcColor, dstColor uint64
	srcAlpha, dstAlpha uint64
}

// blendFor maps the blend state of a color target.
func blendFor(b explicit.BlendState) blendOf {
	if !b.Enabled {
		return blendOf{srcColor: blendOne, dstColor: blendZero, srcAlpha: blendOne, dstAlpha: blendZero}
	}
	src, dst := blendFactor(b.SrcFactor, false), blendFactor(b.DstFactor, true)
	return blendOf{enabled: true, srcColor: src, dstColor: dst, srcAlpha: src, dstAlpha: dst}
}

// The color write mask of Metal has red in the highest bit (MTLColorWriteMask).
const (
	colorWriteRed   uint64 = 1 << 3
	colorWriteGreen uint64 = 1 << 2
	colorWriteBlue  uint64 = 1 << 1
	colorWriteAlpha uint64 = 1 << 0
)

// writeMask maps a color write mask: explicit's has red in the lowest bit.
func writeMask(m explicit.ColorWriteMask) uint64 {
	bits := m.Bits()
	var out uint64
	if bits&1 != 0 {
		out |= colorWriteRed
	}
	if bits&2 != 0 {
		out |= colorWriteGreen
	}
	if bits&4 != 0 {
		out |= colorWriteBlue
	}
	if bits&8 != 0 {
		out |= colorWriteAlpha
	}
	return out
}

// The primitive types and the classes of a pipeline (MTLPrimitiveType, MTLPrimitiveTopologyClass).
const (
	primitivePoint         uint64 = 0
	primitiveLine          uint64 = 1
	primitiveLineStrip     uint64 = 2
	primitiveTriangle      uint64 = 3
	primitiveTriangleStrip uint64 = 4

	topologyClassPoint    uint64 = 1
	topologyClassLine     uint64 = 2
	topologyClassTriangle uint64 = 3
)

// primitiveType maps a topology to the type that a draw is made with.
func primitiveType(t explicit.PrimitiveTopology) uint64 {
	switch t {
	case explicit.TopologyTriangleStrip:
		return primitiveTriangleStrip
	case explicit.TopologyLineList:
		return primitiveLine
	case explicit.TopologyLineStrip:
		return primitiveLineStrip
	case explicit.TopologyPointList:
		return primitivePoint
	}
	return primitiveTriangle
}

// topologyClass maps a topology to the class that a pipeline is made for.
func topologyClass(t explicit.PrimitiveTopology) uint64 {
	switch t {
	case explicit.TopologyLineList, explicit.TopologyLineStrip:
		return topologyClassLine
	case explicit.TopologyPointList:
		return topologyClassPoint
	}
	return topologyClassTriangle
}

// The cull modes and windings (MTLCullMode, MTLWinding).
const (
	cullNone  uint64 = 0
	cullFront uint64 = 1
	cullBack  uint64 = 2

	windingClockwise        uint64 = 0
	windingCounterClockwise uint64 = 1
)

// cullMode maps a cull mode.
func cullMode(c explicit.CullMode) uint64 {
	switch c {
	case explicit.CullFront:
		return cullFront
	case explicit.CullBack:
		return cullBack
	}
	return cullNone
}

// winding maps the front face. The clip space is WebGPU's, which is Metal's: y up, depth from 0 to 1, and
// the framebuffer's origin at the top left, so a counter-clockwise triangle on the screen is a front face
// when the winding is counter-clockwise (SPEC §27.1).
func winding(f explicit.FrontFace) uint64 {
	if f == explicit.FrontFaceCW {
		return windingClockwise
	}
	return windingCounterClockwise
}

// The comparisons and stencil operations (MTLCompareFunction, MTLStencilOperation).
const (
	compareNever        uint64 = 0
	compareLess         uint64 = 1
	compareEqual        uint64 = 2
	compareLessEqual    uint64 = 3
	compareGreater      uint64 = 4
	compareNotEqual     uint64 = 5
	compareGreaterEqual uint64 = 6
	compareAlways       uint64 = 7

	stencilKeep           uint64 = 0
	stencilZero           uint64 = 1
	stencilReplace        uint64 = 2
	stencilIncrementClamp uint64 = 3
	stencilDecrementClamp uint64 = 4
	stencilInvert         uint64 = 5
	stencilIncrementWrap  uint64 = 6
	stencilDecrementWrap  uint64 = 7
)

// compareFunction maps a comparison; the unspecified one is always.
func compareFunction(c explicit.CompareFunc) uint64 {
	switch c {
	case explicit.CompareNever:
		return compareNever
	case explicit.CompareLess:
		return compareLess
	case explicit.CompareEqual:
		return compareEqual
	case explicit.CompareLessEqual:
		return compareLessEqual
	case explicit.CompareGreater:
		return compareGreater
	case explicit.CompareNotEqual:
		return compareNotEqual
	case explicit.CompareGreaterEqual:
		return compareGreaterEqual
	}
	return compareAlways
}

// stencilOperation maps a stencil operation.
func stencilOperation(o explicit.StencilOp) uint64 {
	switch o {
	case explicit.StencilZero:
		return stencilZero
	case explicit.StencilReplace:
		return stencilReplace
	case explicit.StencilInvert:
		return stencilInvert
	case explicit.StencilIncrementClamp:
		return stencilIncrementClamp
	case explicit.StencilDecrementClamp:
		return stencilDecrementClamp
	case explicit.StencilIncrementWrap:
		return stencilIncrementWrap
	case explicit.StencilDecrementWrap:
		return stencilDecrementWrap
	}
	return stencilKeep
}

// stencilMask maps a stencil mask of the description: 0 means every bit, and Metal has 8 stencil bits.
func stencilMask(m uint32) uint32 {
	if m == 0 {
		return 0xFF
	}
	return m & 0xFF
}

// The load and store actions of a pass (MTLLoadAction, MTLStoreAction).
const (
	loadDontCare uint64 = 0
	loadLoad     uint64 = 1
	loadClear    uint64 = 2

	storeDontCare                   uint64 = 0
	storeStore                      uint64 = 1
	storeMultisampleResolve         uint64 = 2
	storeStoreAndMultisampleResolve uint64 = 3
)

// loadAction maps the load operation of a pass.
func loadAction(op explicit.LoadOp) uint64 {
	switch op {
	case explicit.LoadLoad:
		return loadLoad
	case explicit.LoadDontCare:
		return loadDontCare
	}
	return loadClear
}

// storeAction maps the store operation of a pass; with a resolve target the multisampled texture is
// resolved, and kept too when the pass stores it.
func storeAction(op explicit.StoreOp, resolve bool) uint64 {
	switch {
	case resolve && op == explicit.StoreStore:
		return storeStoreAndMultisampleResolve
	case resolve:
		return storeMultisampleResolve
	case op == explicit.StoreStore:
		return storeStore
	}
	return storeDontCare
}

// The index types and the step functions of the vertex buffers (MTLIndexType, MTLVertexStepFunction).
const (
	indexUInt16 uint64 = 0
	indexUInt32 uint64 = 1

	stepPerVertex   uint64 = 1
	stepPerInstance uint64 = 2
)

// The vertex formats (MTLVertexFormat).
const (
	vertexFormatInvalid uint64 = 0
	vfUChar2            uint64 = 1
	vfUChar4            uint64 = 3
	vfChar2             uint64 = 4
	vfChar4             uint64 = 6
	vfUChar2Normalized  uint64 = 7
	vfUChar4Normalized  uint64 = 9
	vfChar2Normalized   uint64 = 10
	vfChar4Normalized   uint64 = 12
	vfUShort2           uint64 = 13
	vfUShort4           uint64 = 15
	vfShort2            uint64 = 16
	vfShort4            uint64 = 18
	vfUShort2Normalized uint64 = 19
	vfUShort4Normalized uint64 = 21
	vfShort2Normalized  uint64 = 22
	vfShort4Normalized  uint64 = 24
	vfHalf2             uint64 = 25
	vfHalf4             uint64 = 27
	vfFloat             uint64 = 28
	vfFloat2            uint64 = 29
	vfFloat3            uint64 = 30
	vfFloat4            uint64 = 31
	vfInt               uint64 = 32
	vfInt2              uint64 = 33
	vfInt3              uint64 = 34
	vfInt4              uint64 = 35
	vfUInt              uint64 = 36
	vfUInt2             uint64 = 37
	vfUInt3             uint64 = 38
	vfUInt4             uint64 = 39
)

// vertexFormat maps the type of a vertex attribute; the unspecified format is four floats, as in the other
// backends.
func vertexFormat(f explicit.VertexFormat) uint64 {
	switch f {
	case explicit.VertexFloat32:
		return vfFloat
	case explicit.VertexFloat32x2:
		return vfFloat2
	case explicit.VertexFloat32x3:
		return vfFloat3
	case explicit.VertexFloat32x4:
		return vfFloat4
	case explicit.VertexUnorm8x4:
		return vfUChar4Normalized
	case explicit.VertexUint32:
		return vfUInt
	case explicit.VertexUint32x2:
		return vfUInt2
	case explicit.VertexUint32x3:
		return vfUInt3
	case explicit.VertexUint32x4:
		return vfUInt4
	case explicit.VertexSint32:
		return vfInt
	case explicit.VertexSint32x2:
		return vfInt2
	case explicit.VertexSint32x3:
		return vfInt3
	case explicit.VertexSint32x4:
		return vfInt4
	case explicit.VertexUint8x2:
		return vfUChar2
	case explicit.VertexUint8x4:
		return vfUChar4
	case explicit.VertexSint8x2:
		return vfChar2
	case explicit.VertexSint8x4:
		return vfChar4
	case explicit.VertexUnorm8x2:
		return vfUChar2Normalized
	case explicit.VertexSnorm8x2:
		return vfChar2Normalized
	case explicit.VertexSnorm8x4:
		return vfChar4Normalized
	case explicit.VertexUint16x2:
		return vfUShort2
	case explicit.VertexUint16x4:
		return vfUShort4
	case explicit.VertexSint16x2:
		return vfShort2
	case explicit.VertexSint16x4:
		return vfShort4
	case explicit.VertexUnorm16x2:
		return vfUShort2Normalized
	case explicit.VertexUnorm16x4:
		return vfUShort4Normalized
	case explicit.VertexSnorm16x2:
		return vfShort2Normalized
	case explicit.VertexSnorm16x4:
		return vfShort4Normalized
	case explicit.VertexFloat16x2:
		return vfHalf2
	case explicit.VertexFloat16x4:
		return vfHalf4
	}
	return vfFloat4
}

// vertexBufferIndex is where the vertex buffer of a slot is in Metal's buffer argument table: above the
// buffers that shaders bind (explicit.MetalVertexBufferBase, SPEC §29.5).
func vertexBufferIndex(slot int) uint64 { return uint64(explicit.MetalVertexBufferBase + slot) }
