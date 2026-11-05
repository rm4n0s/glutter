package webgpu

import "github.com/rm4n0s/glutter/internal/gpu/explicit"

// The usage flags of the WebGPU API (GPUBufferUsage and GPUTextureUsage), and the map mode.
const (
	bufferMapRead  = 0x0001
	bufferCopySrc  = 0x0004
	bufferCopyDst  = 0x0008
	bufferIndex    = 0x0010
	bufferVertex   = 0x0020
	bufferUniform  = 0x0040
	textureCopySrc = 0x01
	textureCopyDst = 0x02
	textureBinding = 0x04
	textureStorage = 0x08
	textureTarget  = 0x10
	mapModeRead    = 1

	// shaderVertex and shaderFragment are the GPUShaderStage flags of a binding's visibility.
	shaderVertex   = 0x1
	shaderFragment = 0x2
)

// bufferUsage returns the GPUBufferUsage flags of a buffer: every buffer can be written (the
// queue's writeBuffer needs COPY_DST).
func bufferUsage(u explicit.BufferUsage) int {
	flags := bufferCopyDst
	if u&explicit.BufferVertex != 0 {
		flags |= bufferVertex
	}
	if u&explicit.BufferIndex != 0 {
		flags |= bufferIndex
	}
	if u&explicit.BufferUniform != 0 {
		flags |= bufferUniform
	}
	return flags
}

// textureUsage returns the GPUTextureUsage flags of a texture. Every texture can be written and
// copied to (COPY_DST); a render target can also be copied from, as in the Vulkan backend.
func textureUsage(u explicit.TextureUsage) int {
	flags := textureCopyDst
	if u&explicit.TextureSampled != 0 {
		flags |= textureBinding
	}
	if u&explicit.TextureRenderTarget != 0 {
		flags |= textureTarget | textureCopySrc
	}
	if u&explicit.TextureCopySource != 0 {
		flags |= textureCopySrc
	}
	if u&explicit.TextureStorage != 0 {
		flags |= textureStorage
	}
	return flags
}

// formatName returns the GPUTextureFormat of a format; "" for one that has none.
func formatName(f explicit.TextureFormat) string {
	switch f {
	case explicit.TextureFormatRGBA8Unorm:
		return "rgba8unorm"
	case explicit.TextureFormatBGRA8Unorm:
		return "bgra8unorm"
	case explicit.TextureFormatR8Unorm:
		return "r8unorm"
	case explicit.TextureFormatRGBA16Float:
		return "rgba16float"
	case explicit.TextureFormatRGBA32Float:
		return "rgba32float"
	case explicit.TextureFormatR32Float:
		return "r32float"
	case explicit.TextureFormatR32Uint:
		return "r32uint"
	case explicit.TextureFormatRG32Uint:
		return "rg32uint"
	case explicit.TextureFormatDepth32Float:
		return "depth32float"
	case explicit.TextureFormatDepth24PlusStencil8:
		return "depth24plus-stencil8"
	}
	return ""
}

// formatOf is the format that a GPUTextureFormat name stands for, as getPreferredCanvasFormat
// answers; the zero value for a name that is not one of the three that a canvas has.
func formatOf(name string) explicit.TextureFormat {
	switch name {
	case "rgba8unorm":
		return explicit.TextureFormatRGBA8Unorm
	case "bgra8unorm":
		return explicit.TextureFormatBGRA8Unorm
	case "r8unorm":
		return explicit.TextureFormatR8Unorm
	}
	return explicit.TextureFormatUnspecified
}

// filterName returns the GPUFilterMode of a mode; nearest for the unspecified one.
func filterName(f explicit.FilterMode) string {
	if f == explicit.FilterLinear {
		return "linear"
	}
	return "nearest"
}

// addressName returns the GPUAddressMode of a mode; clamp-to-edge for the unspecified one.
func addressName(m explicit.AddressMode) string {
	switch m {
	case explicit.AddressRepeat:
		return "repeat"
	case explicit.AddressMirror:
		return "mirror-repeat"
	}
	return "clamp-to-edge"
}

// samplerLodMax is the lodMaxClamp of a sampler: the first level only when it has no mipmap
// filter (as the Vulkan backend does), else every level.
func samplerLodMax(d explicit.SamplerDesc) float64 {
	if d.MipmapFilter == explicit.FilterUnspecified {
		return 0
	}
	return 32
}

// vertexFormatName returns the GPUVertexFormat of a format; float32x4 for the unspecified one,
// as the Vulkan backend does.
func vertexFormatName(f explicit.VertexFormat) string {
	switch f {
	case explicit.VertexFloat32:
		return "float32"
	case explicit.VertexFloat32x2:
		return "float32x2"
	case explicit.VertexFloat32x3:
		return "float32x3"
	case explicit.VertexUnorm8x4:
		return "unorm8x4"
	case explicit.VertexUint32:
		return "uint32"
	case explicit.VertexUint32x2:
		return "uint32x2"
	case explicit.VertexUint32x3:
		return "uint32x3"
	case explicit.VertexUint32x4:
		return "uint32x4"
	case explicit.VertexSint32:
		return "sint32"
	case explicit.VertexSint32x2:
		return "sint32x2"
	case explicit.VertexSint32x3:
		return "sint32x3"
	case explicit.VertexSint32x4:
		return "sint32x4"
	case explicit.VertexUint8x2:
		return "uint8x2"
	case explicit.VertexUint8x4:
		return "uint8x4"
	case explicit.VertexSint8x2:
		return "sint8x2"
	case explicit.VertexSint8x4:
		return "sint8x4"
	case explicit.VertexUnorm8x2:
		return "unorm8x2"
	case explicit.VertexSnorm8x2:
		return "snorm8x2"
	case explicit.VertexSnorm8x4:
		return "snorm8x4"
	case explicit.VertexUint16x2:
		return "uint16x2"
	case explicit.VertexUint16x4:
		return "uint16x4"
	case explicit.VertexSint16x2:
		return "sint16x2"
	case explicit.VertexSint16x4:
		return "sint16x4"
	case explicit.VertexUnorm16x2:
		return "unorm16x2"
	case explicit.VertexUnorm16x4:
		return "unorm16x4"
	case explicit.VertexSnorm16x2:
		return "snorm16x2"
	case explicit.VertexSnorm16x4:
		return "snorm16x4"
	case explicit.VertexFloat16x2:
		return "float16x2"
	case explicit.VertexFloat16x4:
		return "float16x4"
	}
	return "float32x4"
}

// dimensionName returns the GPUTextureViewDimension of a texture kind's view.
func dimensionName(k explicit.TextureKind) string {
	switch k {
	case explicit.TextureKind2DArray:
		return "2d-array"
	case explicit.TextureKindCube:
		return "cube"
	case explicit.TextureKind3D:
		return "3d"
	}
	return "2d"
}

// bindingDimensionName returns the GPUTextureViewDimension that a texture binding declares.
func bindingDimensionName(d explicit.TextureDimension) string {
	switch d {
	case explicit.TextureDim2DArray:
		return "2d-array"
	case explicit.TextureDimCube:
		return "cube"
	case explicit.TextureDim3D:
		return "3d"
	}
	return "2d"
}

// sampleTypeName returns the GPUTextureSampleType that a texture binding declares.
func sampleTypeName(s explicit.SampleType) string {
	switch s {
	case explicit.SampleTypeUnfilterableFloat:
		return "unfilterable-float"
	case explicit.SampleTypeUint:
		return "uint"
	}
	return "float"
}

// textureDimensionName returns the GPUTextureDimension of a texture kind: "3d" or "2d" (a cube and an
// array are 2D textures with layers).
func textureDimensionName(k explicit.TextureKind) string {
	if k == explicit.TextureKind3D {
		return "3d"
	}
	return "2d"
}

// topologyName returns the GPUPrimitiveTopology of a topology.
func topologyName(t explicit.PrimitiveTopology) string {
	switch t {
	case explicit.TopologyTriangleStrip:
		return "triangle-strip"
	case explicit.TopologyLineList:
		return "line-list"
	case explicit.TopologyLineStrip:
		return "line-strip"
	case explicit.TopologyPointList:
		return "point-list"
	}
	return "triangle-list"
}

// cullName returns the GPUCullMode of a mode.
func cullName(c explicit.CullMode) string {
	switch c {
	case explicit.CullFront:
		return "front"
	case explicit.CullBack:
		return "back"
	}
	return "none"
}

// frontFaceName returns the GPUFrontFace of a winding.
func frontFaceName(f explicit.FrontFace) string {
	if f == explicit.FrontFaceCW {
		return "cw"
	}
	return "ccw"
}

// compareName returns the GPUCompareFunction of a function; always for the unspecified one.
func compareName(c explicit.CompareFunc) string {
	switch c {
	case explicit.CompareNever:
		return "never"
	case explicit.CompareLess:
		return "less"
	case explicit.CompareEqual:
		return "equal"
	case explicit.CompareLessEqual:
		return "less-equal"
	case explicit.CompareGreater:
		return "greater"
	case explicit.CompareNotEqual:
		return "not-equal"
	case explicit.CompareGreaterEqual:
		return "greater-equal"
	}
	return "always"
}

// stencilOpName returns the GPUStencilOperation of an operation.
func stencilOpName(o explicit.StencilOp) string {
	switch o {
	case explicit.StencilZero:
		return "zero"
	case explicit.StencilReplace:
		return "replace"
	case explicit.StencilInvert:
		return "invert"
	case explicit.StencilIncrementClamp:
		return "increment-clamp"
	case explicit.StencilDecrementClamp:
		return "decrement-clamp"
	case explicit.StencilIncrementWrap:
		return "increment-wrap"
	case explicit.StencilDecrementWrap:
		return "decrement-wrap"
	}
	return "keep"
}

// loadOpName returns the GPULoadOp of an operation: WebGPU has no "don't care", so it clears.
func loadOpName(l explicit.LoadOp) string {
	if l == explicit.LoadLoad {
		return "load"
	}
	return "clear"
}

// storeOpName returns the GPUStoreOp of an operation.
func storeOpName(s explicit.StoreOp) string {
	if s == explicit.StoreDiscard {
		return "discard"
	}
	return "store"
}

// writeMaskBits returns the GPUColorWrite flags of a mask: the red, green, blue and alpha bits that
// explicit.ColorWriteMask.Bits gives are WebGPU's own (1, 2, 4 and 8).
func writeMaskBits(m explicit.ColorWriteMask) int { return int(m.Bits()) }

// stencilMask is a stencil mask of the description: 0 means every bit.
func stencilMask(m uint32) float64 {
	if m == 0 {
		return 0xFFFFFFFF
	}
	return float64(m)
}

// blendFactorName returns the GPUBlendFactor of a factor; the unspecified one is one.
func blendFactorName(f explicit.BlendFactor) string {
	switch f {
	case explicit.BlendZero:
		return "zero"
	case explicit.BlendSrcAlpha:
		return "src-alpha"
	case explicit.BlendOneMinusSrcAlpha:
		return "one-minus-src-alpha"
	case explicit.BlendDstAlpha:
		return "dst-alpha"
	case explicit.BlendOneMinusDstAlpha:
		return "one-minus-dst-alpha"
	}
	return "one"
}

// blendFactors returns the source and destination factor names of an enabled blend state. As in
// the Vulkan backend an unspecified destination factor is zero (the BlendState says so for the
// destination of a state that is on) and an unspecified source factor is one.
func blendFactors(b explicit.BlendState) (src, dst string) {
	src = blendFactorName(b.SrcFactor)
	dst = "zero"
	if b.DstFactor != explicit.BlendFactorUnspecified {
		dst = blendFactorName(b.DstFactor)
	}
	return src, dst
}

// alignUp rounds n up to a multiple of alignment.
func alignUp(n, alignment int) int {
	if alignment <= 1 {
		return n
	}
	return (n + alignment - 1) / alignment * alignment
}
