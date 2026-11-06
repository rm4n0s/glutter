package gpu

import "github.com/rm4n0s/glutter/internal/gpu/explicit"

// Limits are the limits of a device (WebGPU's GPUSupportedLimits). They are the design limits of Glutter's explicit
// renderer (SPEC §9.3, §27.1): every backend honours them, so that a program that stays inside them runs the same on
// each. The constructors return GPULimitExceeded for a description that is over one.
//
// Flutter: no counterpart
type Limits struct {
	// MaxTextureDimension2D is the largest width or height of a 2D texture, and MaxTextureDimension3D the largest
	// side of a 3D texture.
	MaxTextureDimension2D, MaxTextureDimension3D int
	// MaxTextureArrayLayers is the most layers of a texture array.
	MaxTextureArrayLayers int
	// MaxBindGroups is the number of bind groups a pipeline may use.
	MaxBindGroups int
	// MaxSampledTexturesPerShaderStage, MaxSamplersPerShaderStage, MaxUniformBuffersPerShaderStage,
	// MaxStorageBuffersPerShaderStage and MaxStorageTexturesPerShaderStage are the most resources of a kind that
	// the functions of one stage read.
	MaxSampledTexturesPerShaderStage, MaxSamplersPerShaderStage, MaxUniformBuffersPerShaderStage int
	MaxStorageBuffersPerShaderStage, MaxStorageTexturesPerShaderStage                            int
	// MaxUniformBufferBindingSize and MaxStorageBufferBindingSize are the largest range of a buffer that a
	// binding reads.
	MaxUniformBufferBindingSize, MaxStorageBufferBindingSize int
	// MinUniformBufferOffsetAlignment and MinStorageBufferOffsetAlignment are what the offsets of a binding are a
	// multiple of.
	MinUniformBufferOffsetAlignment, MinStorageBufferOffsetAlignment int
	// BytesPerRowAlignment is what DataLayout.BytesPerRow of a copy between a buffer and a texture is a multiple of.
	BytesPerRowAlignment int
	// MaxBufferSize is the largest buffer.
	MaxBufferSize int
	// MaxVertexBuffers and MaxVertexAttributes are the most vertex buffers and vertex attributes of a pipeline.
	MaxVertexBuffers, MaxVertexAttributes int
	// MaxColorAttachments is the most color attachments of a pass or targets of a pipeline, and
	// MaxColorAttachmentBytesPerSample the most bytes that their texels take together.
	MaxColorAttachments, MaxColorAttachmentBytesPerSample int
	// MaxComputeInvocationsPerWorkgroup, MaxComputeWorkgroupSize and MaxComputeWorkgroupsPerDimension limit the
	// @workgroup_size of a compute function and the numbers of a dispatch.
	MaxComputeInvocationsPerWorkgroup int
	MaxComputeWorkgroupSize           [3]int
	MaxComputeWorkgroupsPerDimension  int
}

// Features are the optional features of a device (WebGPU's GPUSupportedFeatures).
//
// Flutter: no counterpart
type Features struct {
	// Float32Filterable says that RGBA32Float and R32Float textures can be sampled with a filter.
	Float32Filterable bool
	// TimestampQuery says that query sets can be made and passes can write timestamps into them.
	TimestampQuery bool
	// TimestampPeriod is the time of one tick of a timestamp, in nanoseconds; 0 when TimestampQuery is false.
	TimestampPeriod float64
}

func limitsOf(l explicit.Limits) Limits {
	return Limits{
		MaxTextureDimension2D: l.MaxTextureSize, MaxTextureDimension3D: l.MaxTextureSize3D, MaxTextureArrayLayers: l.MaxTextureArrayLayers,
		MaxBindGroups:                    l.MaxBindGroups,
		MaxSampledTexturesPerShaderStage: l.MaxSampledTexturesPerStage, MaxSamplersPerShaderStage: l.MaxSamplersPerStage,
		MaxUniformBuffersPerShaderStage: l.MaxUniformBuffersPerStage, MaxStorageBuffersPerShaderStage: l.MaxStorageBuffersPerStage,
		MaxStorageTexturesPerShaderStage: l.MaxStorageTexturesPerStage,
		MaxUniformBufferBindingSize:      l.MaxUniformBindingSize, MaxStorageBufferBindingSize: l.MaxStorageBindingSize,
		MinUniformBufferOffsetAlignment: l.UniformOffsetAlignment, MinStorageBufferOffsetAlignment: l.StorageOffsetAlignment,
		BytesPerRowAlignment: l.CopyRowPitchAlignment, MaxBufferSize: l.MaxBufferSize,
		MaxVertexBuffers: l.MaxVertexBuffers, MaxVertexAttributes: l.MaxVertexAttributes,
		MaxColorAttachments: l.MaxColorAttachments, MaxColorAttachmentBytesPerSample: l.MaxColorAttachmentBytesPerSample,
		MaxComputeInvocationsPerWorkgroup: l.MaxComputeInvocationsPerWorkgroup, MaxComputeWorkgroupSize: l.MaxComputeWorkgroupSize,
		MaxComputeWorkgroupsPerDimension: l.MaxComputeWorkgroupsPerDimension,
	}
}

func featuresOf(f explicit.Features) Features {
	return Features{Float32Filterable: f.Float32Filterable, TimestampQuery: f.TimestampQuery, TimestampPeriod: f.TimestampPeriod}
}
