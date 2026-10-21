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
	}
	return ""
}

// formatOf is the format that a GPUTextureFormat name stands for, as getPreferredCanvasFormat
// answers; the zero value for a name that is not one of the three.
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
	}
	return "float32x4"
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
