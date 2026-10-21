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
	// BufferIndex is a buffer of 16 bit indices.
	//
	// Flutter: no counterpart
	BufferIndex
	// BufferUniform is a buffer that uniform bindings read, with dynamic offsets.
	//
	// Flutter: no counterpart
	BufferUniform
)

// TextureFormat is the pixel format of a texture. There are no sRGB formats:
// blending is done in non-linear sRGB (SPEC §9.1, §9.3). The zero value is
// TextureFormatUnspecified.
//
// Flutter: no counterpart
type TextureFormat int

// The formats of a texture.
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
)

// BytesPerPixel returns the size of a pixel in bytes; 0 for an unspecified format.
//
// Flutter: no counterpart
func (f TextureFormat) BytesPerPixel() int {
	switch f {
	case TextureFormatRGBA8Unorm, TextureFormatBGRA8Unorm:
		return 4
	case TextureFormatR8Unorm:
		return 1
	}
	return 0
}

// TextureUsage says what a texture is used for; values are combined with |. Every
// texture can be written with Device.WriteTexture.
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
)

// TextureDesc describes a 2D texture.
//
// Flutter: no counterpart
type TextureDesc struct {
	// Width and Height are in pixels, at most Limits.MaxTextureSize.
	Width, Height int
	// Format is the pixel format.
	Format TextureFormat
	// MipLevels is the number of levels; 0 means 1.
	MipLevels int
	// Usage says how the texture is used.
	Usage TextureUsage
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
}

// VertexFormat is the type of a vertex attribute. The zero value is
// VertexFormatUnspecified.
//
// Flutter: no counterpart
type VertexFormat int

// The vertex formats.
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
)

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

// PipelineDesc describes a render pipeline: a generated shader, its entry points,
// the vertex layout, the blend state and the format of the target. Triangles are the
// only topology the renderer uses.
//
// Flutter: no counterpart
type PipelineDesc struct {
	// Shader is a generated shader.
	Shader *Shader
	// VertexEntry and FragmentEntry are the names of its entry points.
	VertexEntry, FragmentEntry string
	// VertexBuffers are the layouts of the vertex buffers, by slot.
	VertexBuffers []VertexBufferLayout
	// Blend is the blend state.
	Blend BlendState
	// Target is the format of the texture the pipeline draws into.
	Target TextureFormat
}

// Binding is one resource of a bind group: the entry for @binding(Binding).
//
// Flutter: no counterpart
type Binding struct {
	// Binding is the number of the shader's @binding.
	Binding int
	// Buffer, with Offset and Size, is a uniform buffer; Size 0 means up to the end.
	Buffer Buffer
	// Offset is where the binding starts in Buffer, in bytes.
	Offset int
	// Size is the size of the binding in bytes.
	Size int
	// Texture is a sampled texture.
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
)

// Buffer is a GPU buffer.
//
// Flutter: no counterpart
type Buffer interface {
	// Size is the size in bytes.
	Size() int
	// Destroy releases the buffer. The GPU finishes the frames that use it first.
	Destroy()
}

// Texture is a GPU 2D texture.
//
// Flutter: no counterpart
type Texture interface {
	// Desc is what the texture was made with, with MipLevels at least 1.
	Desc() TextureDesc
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
	// NewBuffer makes a buffer of size bytes.
	NewBuffer(size int, usage BufferUsage) (Buffer, error)
	// NewTexture makes a 2D texture.
	NewTexture(desc TextureDesc) (Texture, error)
	// NewSampler makes a sampler.
	NewSampler(desc SamplerDesc) (Sampler, error)
	// NewPipeline makes a pipeline from a generated shader.
	NewPipeline(desc PipelineDesc) (Pipeline, error)
	// NewBindGroup makes the resources of the bind group number group of the
	// pipeline.
	NewBindGroup(p Pipeline, group int, entries []Binding) (BindGroup, error)
	// WriteBuffer copies data into the buffer at offset. The data is copied before
	// the call returns.
	WriteBuffer(b Buffer, offset int, data []byte)
	// WriteTexture copies the rows of data, stride bytes apart, into the rectangle r
	// of the level of the texture.
	WriteTexture(t Texture, level int, r image.Rectangle, data []byte, stride int)
	// NewSwapchain makes the swapchain of a window.
	NewSwapchain(h gpu.SurfaceHandle) (Swapchain, error)
	// Begin starts the commands of a frame; there is one encoder per frame.
	Begin() (Encoder, error)
	// Close releases the device and everything made with it.
	Close() error
}

// Encoder records the commands of one frame.
//
// Flutter: no counterpart
type Encoder interface {
	// BeginPass starts a pass that draws into the target, which must have
	// TextureRenderTarget. The previous pass must have ended.
	BeginPass(target Texture, load LoadOp, clear ui.Color) Pass
	// CopyTexture copies the rectangle r of src to dst at the point at.
	CopyTexture(src Texture, r image.Rectangle, dst Texture, at image.Point)
	// ReadTexture reads the rectangle r of src. done runs once the GPU has finished,
	// never inside the frame that asked.
	ReadTexture(src Texture, r image.Rectangle, done func(*image.RGBA, error))
	// Submit sends the frame to the GPU.
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
	// SetIndexBuffer binds the index buffer, starting at offset bytes.
	SetIndexBuffer(b Buffer, offset int)
	// SetScissor limits the draws to the rectangle, in pixels of the target.
	SetScissor(r image.Rectangle)
	// Draw draws vertices, instances times.
	Draw(vertices, instances, firstVertex, firstInstance int)
	// DrawIndexed draws indices, instances times.
	DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int)
	// End finishes the pass.
	End()
}

// Swapchain is the images of a window.
//
// Flutter: no counterpart
type Swapchain interface {
	// Configure sets the size of the images, in device pixels.
	Configure(width, height int) error
	// Acquire returns the image to draw into next. When it is out of date, Configure
	// again and retry once.
	Acquire() (Texture, error)
	// Present shows the image, telling the compositor which rectangles changed (nil
	// means all).
	Present(damage []image.Rectangle) error
	// Destroy releases the swapchain.
	Destroy()
}
