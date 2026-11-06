package gpu

// The enumerations and flags of the package. They have the numbers of the same types of the explicit device
// (internal/gpu/explicit), so that a call converts them with a cast; types_test.go checks every pair.

// BufferUsage says what a buffer is used for; the values are combined with |.
//
// Flutter: no counterpart
type BufferUsage int

// The values of BufferUsage.
const (
	// BufferUsageVertex is a buffer that holds vertices (SetVertexBuffer).
	//
	// Flutter: no counterpart
	BufferUsageVertex BufferUsage = 1 << iota
	// BufferUsageIndex is a buffer that holds indices (SetIndexBuffer).
	//
	// Flutter: no counterpart
	BufferUsageIndex
	// BufferUsageUniform is a buffer that a uniform binding reads.
	//
	// Flutter: no counterpart
	BufferUsageUniform
	// BufferUsageStorage is a buffer that a storage binding reads or, in a compute pass, writes.
	//
	// Flutter: no counterpart
	BufferUsageStorage
	// BufferUsageIndirect is a buffer of the arguments of DrawIndirect, DrawIndexedIndirect and DispatchIndirect.
	//
	// Flutter: no counterpart
	BufferUsageIndirect
	// BufferUsageCopySrc is a buffer that the copies and Read take from.
	//
	// Flutter: no counterpart
	BufferUsageCopySrc
	// BufferUsageCopyDst is a buffer that the copies write to.
	//
	// Flutter: no counterpart
	BufferUsageCopyDst
	// BufferUsageQueryResolve is a buffer that ResolveQuerySet writes to.
	//
	// Flutter: no counterpart
	BufferUsageQueryResolve
)

// TextureFormat is the format of the texels of a texture.
//
// Flutter: no counterpart
type TextureFormat int

// The values of TextureFormat.
const (
	// TextureFormatUnspecified is the zero value: no format.
	//
	// Flutter: no counterpart
	TextureFormatUnspecified TextureFormat = iota
	// TextureFormatRGBA8Unorm is 4 bytes per texel, red first.
	//
	// Flutter: no counterpart
	TextureFormatRGBA8Unorm
	// TextureFormatBGRA8Unorm is 4 bytes per texel, blue first.
	//
	// Flutter: no counterpart
	TextureFormatBGRA8Unorm
	// TextureFormatR8Unorm is 1 byte per texel: coverage masks.
	//
	// Flutter: no counterpart
	TextureFormatR8Unorm
	// TextureFormatRGBA16Float is 8 bytes per texel, four half floats.
	//
	// Flutter: no counterpart
	TextureFormatRGBA16Float
	// TextureFormatRGBA32Float is 16 bytes per texel, four floats; a shader filters it only when Features.Float32Filterable.
	//
	// Flutter: no counterpart
	TextureFormatRGBA32Float
	// TextureFormatR32Float is 4 bytes per texel, one float.
	//
	// Flutter: no counterpart
	TextureFormatR32Float
	// TextureFormatR32Uint is 4 bytes per texel, one unsigned integer.
	//
	// Flutter: no counterpart
	TextureFormatR32Uint
	// TextureFormatRG32Uint is 8 bytes per texel, two unsigned integers.
	//
	// Flutter: no counterpart
	TextureFormatRG32Uint
	// TextureFormatDepth32Float is a depth buffer of one float per texel.
	//
	// Flutter: no counterpart
	TextureFormatDepth32Float
	// TextureFormatDepth24PlusStencil8 is a depth buffer of at least 24 bits with an 8 bit stencil; it cannot be copied.
	//
	// Flutter: no counterpart
	TextureFormatDepth24PlusStencil8
)

// TextureUsage says what a texture is used for; the values are combined with |.
//
// Flutter: no counterpart
type TextureUsage int

// The values of TextureUsage.
const (
	// TextureUsageTextureBinding is a texture that a shader samples.
	//
	// Flutter: no counterpart
	TextureUsageTextureBinding TextureUsage = 1 << iota
	// TextureUsageRenderAttachment is a texture that a render pass draws into.
	//
	// Flutter: no counterpart
	TextureUsageRenderAttachment
	// TextureUsageCopySrc is a texture that the copies and readbacks take from (a render attachment can be read too).
	//
	// Flutter: no counterpart
	TextureUsageCopySrc
	// TextureUsageStorageBinding is a texture that a compute shader writes through a storage binding.
	//
	// Flutter: no counterpart
	TextureUsageStorageBinding
)

// TextureKind is what a texture is: one image, an array of them, a cube or a volume.
//
// Flutter: no counterpart
type TextureKind int

// The values of TextureKind.
const (
	// TextureKindUnspecified is the zero value: a 2D texture.
	//
	// Flutter: no counterpart
	TextureKindUnspecified TextureKind = iota
	// TextureKind2D is one image.
	//
	// Flutter: no counterpart
	TextureKind2D
	// TextureKind2DArray is an array of images (Size.DepthOrArrayLayers of them).
	//
	// Flutter: no counterpart
	TextureKind2DArray
	// TextureKindCube is a cube: six square faces.
	//
	// Flutter: no counterpart
	TextureKindCube
	// TextureKind3D is a volume (Size.DepthOrArrayLayers deep).
	//
	// Flutter: no counterpart
	TextureKind3D
)

// FilterMode is how a sampler reads between texels.
//
// Flutter: no counterpart
type FilterMode int

// The values of FilterMode.
const (
	// FilterModeUnspecified is the zero value: the default (nearest).
	//
	// Flutter: no counterpart
	FilterModeUnspecified FilterMode = iota
	// FilterModeNearest reads the nearest texel.
	//
	// Flutter: no counterpart
	FilterModeNearest
	// FilterModeLinear blends the nearest texels.
	//
	// Flutter: no counterpart
	FilterModeLinear
)

// AddressMode is what a sampler does outside the texture.
//
// Flutter: no counterpart
type AddressMode int

// The values of AddressMode.
const (
	// AddressModeUnspecified is the zero value: the default (clamp to the edge).
	//
	// Flutter: no counterpart
	AddressModeUnspecified AddressMode = iota
	// AddressModeClampToEdge repeats the edge texel.
	//
	// Flutter: no counterpart
	AddressModeClampToEdge
	// AddressModeRepeat tiles the texture.
	//
	// Flutter: no counterpart
	AddressModeRepeat
	// AddressModeMirrorRepeat tiles the texture, mirrored at each edge.
	//
	// Flutter: no counterpart
	AddressModeMirrorRepeat
)

// VertexFormat is how a vertex attribute is stored in a vertex buffer. The name has the type and the number of components.
//
// Flutter: no counterpart
type VertexFormat int

// The values of VertexFormat.
const (
	// VertexFormatUnspecified is the zero value: no format.
	//
	// Flutter: no counterpart
	VertexFormatUnspecified VertexFormat = iota
	// VertexFormatFloat32 is Float32.
	//
	// Flutter: no counterpart
	VertexFormatFloat32
	// VertexFormatFloat32x2 is Float32x2.
	//
	// Flutter: no counterpart
	VertexFormatFloat32x2
	// VertexFormatFloat32x3 is Float32x3.
	//
	// Flutter: no counterpart
	VertexFormatFloat32x3
	// VertexFormatFloat32x4 is Float32x4.
	//
	// Flutter: no counterpart
	VertexFormatFloat32x4
	// VertexFormatUnorm8x4 is Unorm8x4.
	//
	// Flutter: no counterpart
	VertexFormatUnorm8x4
	// VertexFormatUint32 is Uint32.
	//
	// Flutter: no counterpart
	VertexFormatUint32
	// VertexFormatUint32x2 is Uint32x2.
	//
	// Flutter: no counterpart
	VertexFormatUint32x2
	// VertexFormatUint32x3 is Uint32x3.
	//
	// Flutter: no counterpart
	VertexFormatUint32x3
	// VertexFormatUint32x4 is Uint32x4.
	//
	// Flutter: no counterpart
	VertexFormatUint32x4
	// VertexFormatSint32 is Sint32.
	//
	// Flutter: no counterpart
	VertexFormatSint32
	// VertexFormatSint32x2 is Sint32x2.
	//
	// Flutter: no counterpart
	VertexFormatSint32x2
	// VertexFormatSint32x3 is Sint32x3.
	//
	// Flutter: no counterpart
	VertexFormatSint32x3
	// VertexFormatSint32x4 is Sint32x4.
	//
	// Flutter: no counterpart
	VertexFormatSint32x4
	// VertexFormatUint8x2 is Uint8x2.
	//
	// Flutter: no counterpart
	VertexFormatUint8x2
	// VertexFormatUint8x4 is Uint8x4.
	//
	// Flutter: no counterpart
	VertexFormatUint8x4
	// VertexFormatSint8x2 is Sint8x2.
	//
	// Flutter: no counterpart
	VertexFormatSint8x2
	// VertexFormatSint8x4 is Sint8x4.
	//
	// Flutter: no counterpart
	VertexFormatSint8x4
	// VertexFormatUnorm8x2 is Unorm8x2.
	//
	// Flutter: no counterpart
	VertexFormatUnorm8x2
	// VertexFormatSnorm8x2 is Snorm8x2.
	//
	// Flutter: no counterpart
	VertexFormatSnorm8x2
	// VertexFormatSnorm8x4 is Snorm8x4.
	//
	// Flutter: no counterpart
	VertexFormatSnorm8x4
	// VertexFormatUint16x2 is Uint16x2.
	//
	// Flutter: no counterpart
	VertexFormatUint16x2
	// VertexFormatUint16x4 is Uint16x4.
	//
	// Flutter: no counterpart
	VertexFormatUint16x4
	// VertexFormatSint16x2 is Sint16x2.
	//
	// Flutter: no counterpart
	VertexFormatSint16x2
	// VertexFormatSint16x4 is Sint16x4.
	//
	// Flutter: no counterpart
	VertexFormatSint16x4
	// VertexFormatUnorm16x2 is Unorm16x2.
	//
	// Flutter: no counterpart
	VertexFormatUnorm16x2
	// VertexFormatUnorm16x4 is Unorm16x4.
	//
	// Flutter: no counterpart
	VertexFormatUnorm16x4
	// VertexFormatSnorm16x2 is Snorm16x2.
	//
	// Flutter: no counterpart
	VertexFormatSnorm16x2
	// VertexFormatSnorm16x4 is Snorm16x4.
	//
	// Flutter: no counterpart
	VertexFormatSnorm16x4
	// VertexFormatFloat16x2 is Float16x2.
	//
	// Flutter: no counterpart
	VertexFormatFloat16x2
	// VertexFormatFloat16x4 is Float16x4.
	//
	// Flutter: no counterpart
	VertexFormatFloat16x4
)

// BlendFactor is a factor of the blend equation: the source times its factor plus the destination times its factor.
//
// Flutter: no counterpart
type BlendFactor int

// The values of BlendFactor.
const (
	// BlendFactorUnspecified is the zero value: One for the source and Zero for the destination.
	//
	// Flutter: no counterpart
	BlendFactorUnspecified BlendFactor = iota
	// BlendFactorZero is 0.
	//
	// Flutter: no counterpart
	BlendFactorZero
	// BlendFactorOne is 1.
	//
	// Flutter: no counterpart
	BlendFactorOne
	// BlendFactorSrcAlpha is the alpha of the source.
	//
	// Flutter: no counterpart
	BlendFactorSrcAlpha
	// BlendFactorOneMinusSrcAlpha is 1 minus the alpha of the source.
	//
	// Flutter: no counterpart
	BlendFactorOneMinusSrcAlpha
	// BlendFactorDstAlpha is the alpha of the destination.
	//
	// Flutter: no counterpart
	BlendFactorDstAlpha
	// BlendFactorOneMinusDstAlpha is 1 minus the alpha of the destination.
	//
	// Flutter: no counterpart
	BlendFactorOneMinusDstAlpha
)

// PrimitiveTopology is how the vertices of a draw make primitives. The zero value is TriangleList.
//
// Flutter: no counterpart
type PrimitiveTopology int

// The values of PrimitiveTopology.
const (
	// PrimitiveTopologyTriangleList makes a triangle of each three vertices.
	//
	// Flutter: no counterpart
	PrimitiveTopologyTriangleList PrimitiveTopology = iota
	// PrimitiveTopologyTriangleStrip makes a triangle of each vertex and the two before it.
	//
	// Flutter: no counterpart
	PrimitiveTopologyTriangleStrip
	// PrimitiveTopologyLineList makes a line of each two vertices.
	//
	// Flutter: no counterpart
	PrimitiveTopologyLineList
	// PrimitiveTopologyLineStrip makes a line of each vertex and the one before it.
	//
	// Flutter: no counterpart
	PrimitiveTopologyLineStrip
	// PrimitiveTopologyPointList makes a point of each vertex.
	//
	// Flutter: no counterpart
	PrimitiveTopologyPointList
)

// CullMode is which triangles a pipeline leaves out. The zero value is CullModeNone.
//
// Flutter: no counterpart
type CullMode int

// The values of CullMode.
const (
	// CullModeNone leaves out none.
	//
	// Flutter: no counterpart
	CullModeNone CullMode = iota
	// CullModeFront leaves out the triangles that face the viewer.
	//
	// Flutter: no counterpart
	CullModeFront
	// CullModeBack leaves out the triangles that face away.
	//
	// Flutter: no counterpart
	CullModeBack
)

// FrontFace is which winding of a triangle faces the viewer. The zero value is FrontFaceCCW.
//
// Flutter: no counterpart
type FrontFace int

// The values of FrontFace.
const (
	// FrontFaceCCW is counter-clockwise on the screen.
	//
	// Flutter: no counterpart
	FrontFaceCCW FrontFace = iota
	// FrontFaceCW is clockwise on the screen.
	//
	// Flutter: no counterpart
	FrontFaceCW
)

// CompareFunction is how a depth or stencil test compares the new value to the stored one.
//
// Flutter: no counterpart
type CompareFunction int

// The values of CompareFunction.
const (
	// CompareFunctionUnspecified is the zero value: Always (Less for the depth test).
	//
	// Flutter: no counterpart
	CompareFunctionUnspecified CompareFunction = iota
	// CompareFunctionNever never passes.
	//
	// Flutter: no counterpart
	CompareFunctionNever
	// CompareFunctionLess passes when the new value is less.
	//
	// Flutter: no counterpart
	CompareFunctionLess
	// CompareFunctionEqual passes when the values are equal.
	//
	// Flutter: no counterpart
	CompareFunctionEqual
	// CompareFunctionLessEqual passes when the new value is less or equal.
	//
	// Flutter: no counterpart
	CompareFunctionLessEqual
	// CompareFunctionGreater passes when the new value is greater.
	//
	// Flutter: no counterpart
	CompareFunctionGreater
	// CompareFunctionNotEqual passes when the values differ.
	//
	// Flutter: no counterpart
	CompareFunctionNotEqual
	// CompareFunctionGreaterEqual passes when the new value is greater or equal.
	//
	// Flutter: no counterpart
	CompareFunctionGreaterEqual
	// CompareFunctionAlways always passes.
	//
	// Flutter: no counterpart
	CompareFunctionAlways
)

// StencilOperation is what the stencil test does to the stored value. The zero value is StencilOperationKeep.
//
// Flutter: no counterpart
type StencilOperation int

// The values of StencilOperation.
const (
	// StencilOperationKeep keeps the value.
	//
	// Flutter: no counterpart
	StencilOperationKeep StencilOperation = iota
	// StencilOperationZero sets it to 0.
	//
	// Flutter: no counterpart
	StencilOperationZero
	// StencilOperationReplace sets it to the stencil reference.
	//
	// Flutter: no counterpart
	StencilOperationReplace
	// StencilOperationInvert inverts its bits.
	//
	// Flutter: no counterpart
	StencilOperationInvert
	// StencilOperationIncrementClamp adds 1 up to the maximum.
	//
	// Flutter: no counterpart
	StencilOperationIncrementClamp
	// StencilOperationDecrementClamp subtracts 1 down to 0.
	//
	// Flutter: no counterpart
	StencilOperationDecrementClamp
	// StencilOperationIncrementWrap adds 1 and wraps.
	//
	// Flutter: no counterpart
	StencilOperationIncrementWrap
	// StencilOperationDecrementWrap subtracts 1 and wraps.
	//
	// Flutter: no counterpart
	StencilOperationDecrementWrap
)

// ColorWriteMask says which channels of a color target a pipeline writes; the values are combined with |. The zero value writes all of them.
//
// Flutter: no counterpart
type ColorWriteMask int

// The values of ColorWriteMask.
const (
	// ColorWriteMaskAll writes all four channels.
	//
	// Flutter: no counterpart
	ColorWriteMaskAll ColorWriteMask = 0
	// ColorWriteMaskRed writes red.
	//
	// Flutter: no counterpart
	ColorWriteMaskRed ColorWriteMask = 1 << (iota - 1)
	// ColorWriteMaskGreen writes green.
	//
	// Flutter: no counterpart
	ColorWriteMaskGreen
	// ColorWriteMaskBlue writes blue.
	//
	// Flutter: no counterpart
	ColorWriteMaskBlue
	// ColorWriteMaskAlpha writes alpha.
	//
	// Flutter: no counterpart
	ColorWriteMaskAlpha
	// ColorWriteMaskNone writes nothing.
	//
	// Flutter: no counterpart
	ColorWriteMaskNone
)

// LoadOp is what a pass does with an attachment before it draws. The zero value is LoadOpClear.
//
// Flutter: no counterpart
type LoadOp int

// The values of LoadOp.
const (
	// LoadOpClear fills the attachment with the clear value.
	//
	// Flutter: no counterpart
	LoadOpClear LoadOp = iota
	// LoadOpLoad keeps what the attachment holds.
	//
	// Flutter: no counterpart
	LoadOpLoad
	// LoadOpDontCare leaves the contents undefined, for a pass that draws every texel.
	//
	// Flutter: no counterpart
	LoadOpDontCare
)

// StoreOp is what a pass does with an attachment after it drew. The zero value is StoreOpStore.
//
// Flutter: no counterpart
type StoreOp int

// The values of StoreOp.
const (
	// StoreOpStore keeps the result.
	//
	// Flutter: no counterpart
	StoreOpStore StoreOp = iota
	// StoreOpDiscard throws the result away.
	//
	// Flutter: no counterpart
	StoreOpDiscard
)
