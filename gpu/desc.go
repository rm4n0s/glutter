package gpu

// Extent3D is the size of a texture or of a copy: the width and height in texels, and the number of layers of
// an array, the six faces of a cube, or the depth of a volume (WebGPU's GPUExtent3D).
//
// Flutter: no counterpart
type Extent3D struct {
	Width, Height, DepthOrArrayLayers int
}

// Origin3D is the first texel of a copy: x and y in the texture, and z, the layer or the slice (WebGPU's
// GPUOrigin3D).
//
// Flutter: no counterpart
type Origin3D struct {
	X, Y, Z int
}

// Color is a clear value: four floats, red first. For a texture of unsigned integers they are the integer
// values (WebGPU's GPUColor).
//
// Flutter: no counterpart
type Color struct {
	R, G, B, A float64
}

// DataLayout is how the texels of a copy are laid out in the bytes of a buffer or of a slice (WebGPU's
// GPUImageDataLayout): the first texel is at Offset, the rows are BytesPerRow apart, and the images (layers or
// slices) RowsPerImage rows apart.
//
// Flutter: no counterpart
type DataLayout struct {
	// Offset is where the first texel is, in bytes; a multiple of 4 and of the bytes of a texel.
	Offset int
	// BytesPerRow is the distance between two rows: a multiple of Limits.BytesPerRowAlignment, and at least a
	// row. A queue write has no such alignment (Queue.WriteTexture).
	BytesPerRow int
	// RowsPerImage is the distance between two images, in rows; at least the height of the copy. 0 means the
	// height of the copy.
	RowsPerImage int
}

// BufferCopy is a buffer and the layout of the texels in it (WebGPU's GPUImageCopyBuffer).
//
// Flutter: no counterpart
type BufferCopy struct {
	Buffer *Buffer
	Layout DataLayout
}

// TextureCopy is a texture, a mip level of it and the first texel (WebGPU's GPUImageCopyTexture).
//
// Flutter: no counterpart
type TextureCopy struct {
	Texture  *Texture
	MipLevel int
	Origin   Origin3D
}

// BufferDesc describes a buffer (WebGPU's GPUBufferDescriptor).
//
// Flutter: no counterpart
type BufferDesc struct {
	// Label names the buffer in the messages of errors and in the log of a leak.
	Label string
	// Size is in bytes, at least 1 and at most Limits.MaxBufferSize.
	Size int
	// Usage says what the buffer is for; at least one use.
	Usage BufferUsage
}

// TextureDesc describes a texture (WebGPU's GPUTextureDescriptor).
//
// Flutter: no counterpart
type TextureDesc struct {
	// Label names the texture in the messages of errors and in the log of a leak.
	Label string
	// Size is the width and height, and the layers of an array (Kind TextureKind2DArray), the depth of a volume
	// (TextureKind3D) or 6 for a cube; 0 stands for 1 (6 for a cube).
	Size Extent3D
	// Kind is the shape; the zero value is a 2D texture.
	Kind TextureKind
	// Format is the format of the texels.
	Format TextureFormat
	// MipLevelCount is the number of levels; 0 means 1.
	MipLevelCount int
	// SampleCount is 1 or 4; 0 means 1. A multisampled texture is a render attachment that a pass resolves into
	// another texture.
	SampleCount int
	// Usage says what the texture is for; at least one use, and only those that Format allows.
	Usage TextureUsage
}

// SamplerDesc describes a sampler (WebGPU's GPUSamplerDescriptor).
//
// Flutter: no counterpart
type SamplerDesc struct {
	// Label names the sampler in the messages of errors and in the log of a leak.
	Label string
	// AddressModeU, AddressModeV and AddressModeW say what the sampler does outside the texture.
	AddressModeU, AddressModeV, AddressModeW AddressMode
	// MagFilter and MinFilter are the filters when the texture is enlarged and when it is reduced, and
	// MipmapFilter the one between two mip levels.
	MagFilter, MinFilter, MipmapFilter FilterMode
}

// VertexAttribute is one input of a vertex shader that a vertex buffer feeds (WebGPU's GPUVertexAttribute).
//
// Flutter: no counterpart
type VertexAttribute struct {
	// ShaderLocation is the @location of the input.
	ShaderLocation int
	// Format is how the attribute is stored in the buffer.
	Format VertexFormat
	// Offset is where it starts in a vertex, in bytes.
	Offset int
}

// VertexBufferLayout is how a vertex buffer is read (WebGPU's GPUVertexBufferLayout).
//
// Flutter: no counterpart
type VertexBufferLayout struct {
	// ArrayStride is the distance between two vertices (or two instances), in bytes.
	ArrayStride int
	// PerInstance makes the buffer advance once for each instance instead of once for each vertex.
	PerInstance bool
	// Attributes are the inputs it feeds.
	Attributes []VertexAttribute
}

// BlendState is the blending of one color target: the new color times SrcFactor plus the one in the target times
// DstFactor, for the color and the alpha alike (the blend of the explicit device, SPEC §9.3; WebGPU's
// GPUBlendState has more).
//
// Flutter: no counterpart
type BlendState struct {
	// Enabled turns blending on; off, the new color replaces the old.
	Enabled bool
	// SrcFactor and DstFactor weigh the new color and the old.
	SrcFactor, DstFactor BlendFactor
}

// ColorTargetState is a color target of a render pipeline (WebGPU's GPUColorTargetState).
//
// Flutter: no counterpart
type ColorTargetState struct {
	// Format is the format of the attachment that the pipeline draws into.
	Format TextureFormat
	// Blend is how the colors blend with what the target holds.
	Blend BlendState
	// WriteMask says which channels are written; the zero value writes all of them.
	WriteMask ColorWriteMask
}

// StencilFaceState is the stencil test of one face of a triangle (WebGPU's GPUStencilFaceState).
//
// Flutter: no counterpart
type StencilFaceState struct {
	// Compare compares the stencil reference to the stored value.
	Compare CompareFunction
	// FailOp, DepthFailOp and PassOp say what happens to the stored value when the stencil test fails, when
	// it passes and the depth test fails, and when both pass.
	FailOp, DepthFailOp, PassOp StencilOperation
}

// DepthStencilState is the depth and stencil state of a render pipeline (WebGPU's GPUDepthStencilState).
//
// Flutter: no counterpart
type DepthStencilState struct {
	// Format is the format of the depth attachment: a depth format.
	Format TextureFormat
	// DepthWriteEnabled lets the pipeline write depth.
	DepthWriteEnabled bool
	// DepthCompare is the depth test; the zero value is Less.
	DepthCompare CompareFunction
	// StencilFront and StencilBack are the stencil tests of the front and back faces.
	StencilFront, StencilBack StencilFaceState
	// StencilReadMask and StencilWriteMask select the bits that the test reads and writes; 0 stands for all of
	// them.
	StencilReadMask, StencilWriteMask uint32
	// DepthBias, DepthBiasSlopeScale and DepthBiasClamp offset the depth of triangles.
	DepthBias           int32
	DepthBiasSlopeScale float32
	DepthBiasClamp      float32
}

// VertexState is the vertex stage of a render pipeline (WebGPU's GPUVertexState).
//
// Flutter: no counterpart
type VertexState struct {
	// Module is the shader and EntryPoint the name of its @vertex function.
	Module     *ShaderModule
	EntryPoint string
	// Buffers describe the vertex buffers, by slot.
	Buffers []VertexBufferLayout
}

// FragmentState is the fragment stage of a render pipeline (WebGPU's GPUFragmentState).
//
// Flutter: no counterpart
type FragmentState struct {
	// Module is the shader and EntryPoint the name of its @fragment function. The module is the vertex
	// stage's: both functions are in one WGSL file.
	Module     *ShaderModule
	EntryPoint string
	// Targets are the color targets, by attachment index; at least one.
	Targets []ColorTargetState
}

// PrimitiveState is how a render pipeline assembles primitives (WebGPU's GPUPrimitiveState).
//
// Flutter: no counterpart
type PrimitiveState struct {
	Topology  PrimitiveTopology
	CullMode  CullMode
	FrontFace FrontFace
}

// MultisampleState is the sample count of a render pipeline (WebGPU's GPUMultisampleState).
//
// Flutter: no counterpart
type MultisampleState struct {
	// Count is 1 or 4; 0 means 1.
	Count int
}

// RenderPipelineDesc describes a render pipeline (WebGPU's GPURenderPipelineDescriptor). The layout comes from the
// shader (WebGPU's layout "auto"): RenderPipeline.BindGroupLayout gives it.
//
// Flutter: no counterpart
type RenderPipelineDesc struct {
	// Label names the pipeline in the messages of errors and in the log of a leak.
	Label        string
	Vertex       VertexState
	Fragment     FragmentState
	Primitive    PrimitiveState
	DepthStencil *DepthStencilState
	Multisample  MultisampleState
}

// ComputePipelineDesc describes a compute pipeline (WebGPU's GPUComputePipelineDescriptor).
//
// Flutter: no counterpart
type ComputePipelineDesc struct {
	// Label names the pipeline in the messages of errors and in the log of a leak.
	Label string
	// Module is the shader and EntryPoint the name of its @compute function.
	Module     *ShaderModule
	EntryPoint string
}

// BindGroupEntry is one resource of a bind group (WebGPU's GPUBindGroupEntry): the one for @binding(Binding) of
// the group. Exactly one of Buffer, TextureView and Sampler is set, the one the binding is.
//
// Flutter: no counterpart
type BindGroupEntry struct {
	// Binding is the number in @binding.
	Binding int
	// Buffer is a uniform or storage buffer, from Offset, Size bytes (0 means to the end). A uniform buffer's
	// Offset is a multiple of Limits.MinUniformBufferOffsetAlignment and a storage buffer's of
	// Limits.MinStorageBufferOffsetAlignment.
	Buffer       *Buffer
	Offset, Size int
	// TextureView is a sampled texture, or the storage texture of a storage binding.
	TextureView *TextureView
	// Sampler is a sampler.
	Sampler *Sampler
}

// BindGroupDesc describes a bind group (WebGPU's GPUBindGroupDescriptor).
//
// Flutter: no counterpart
type BindGroupDesc struct {
	// Label names the bind group in the messages of errors and in the log of a leak.
	Label string
	// Layout is the layout of the group, from RenderPipeline.BindGroupLayout or
	// ComputePipeline.BindGroupLayout.
	Layout *BindGroupLayout
	// Entries are the resources, one for each binding that the pipeline's functions read from the group.
	Entries []BindGroupEntry
}

// QuerySetDesc describes a set of timestamp queries (WebGPU's GPUQuerySetDescriptor).
//
// Flutter: no counterpart
type QuerySetDesc struct {
	// Label names the set in the messages of errors and in the log of a leak.
	Label string
	// Count is the number of timestamps; at least 1.
	Count int
}

// PassTimestampWrites tells a pass to write its start and its end to a query set (WebGPU's
// GPURenderPassTimestampWrites). Features.TimestampQuery must be true.
//
// Flutter: no counterpart
type PassTimestampWrites struct {
	QuerySet *QuerySet
	// BeginIndex and EndIndex are the queries that get the time at the start and at the end of the pass.
	BeginIndex, EndIndex int
}

// RenderPassColorAttachment is a color attachment of a render pass (WebGPU's GPURenderPassColorAttachment).
//
// Flutter: no counterpart
type RenderPassColorAttachment struct {
	// View is the texture that is drawn into: a render attachment.
	View *TextureView
	// ResolveTarget, when set, gets the resolved pixels of a multisampled View.
	ResolveTarget *TextureView
	// LoadOp is what the pass does with View before it draws, and ClearValue the color of LoadOpClear.
	LoadOp     LoadOp
	ClearValue Color
	// StoreOp is what the pass does with View after.
	StoreOp StoreOp
}

// RenderPassDepthStencilAttachment is the depth and stencil attachment of a render pass (WebGPU's
// GPURenderPassDepthStencilAttachment).
//
// Flutter: no counterpart
type RenderPassDepthStencilAttachment struct {
	// View is the depth texture.
	View *TextureView
	// DepthLoadOp, DepthStoreOp and DepthClearValue are for the depth, StencilLoadOp, StencilStoreOp and
	// StencilClearValue for the stencil of a format that has one.
	DepthLoadOp       LoadOp
	DepthStoreOp      StoreOp
	DepthClearValue   float64
	StencilLoadOp     LoadOp
	StencilStoreOp    StoreOp
	StencilClearValue uint32
}

// RenderPassDesc describes a render pass (WebGPU's GPURenderPassDescriptor).
//
// Flutter: no counterpart
type RenderPassDesc struct {
	// Label names the pass in the messages of errors.
	Label string
	// ColorAttachments are up to Limits.MaxColorAttachments, by attachment index.
	ColorAttachments []RenderPassColorAttachment
	// DepthStencilAttachment is the depth attachment, or nil.
	DepthStencilAttachment *RenderPassDepthStencilAttachment
	// TimestampWrites, when set, writes the time of the start and the end of the pass.
	TimestampWrites *PassTimestampWrites
}

// ComputePassDesc describes a compute pass (WebGPU's GPUComputePassDescriptor).
//
// Flutter: no counterpart
type ComputePassDesc struct {
	// Label names the pass in the messages of errors.
	Label string
	// TimestampWrites, when set, writes the time of the start and the end of the pass.
	TimestampWrites *PassTimestampWrites
}
