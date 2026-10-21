package explicit

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// FramesInFlight is how many frames the GPU works on at once (SPEC §9.3). A device's frame
// slots alternate with every Submit, so the renderer keeps one set of dynamic buffers for
// each, and the frame numbered n uses set n % FramesInFlight.
//
// Flutter: no counterpart
const FramesInFlight = 2

// Limits are what a Device guarantees and what the renderer may use. They are
// WebGPU's default limits, so neither Vulkan nor WebGPU requests more (SPEC §9.3).
//
// Flutter: no counterpart
type Limits struct {
	// MaxBindGroups is the number of bind groups a pipeline can use.
	MaxBindGroups int
	// MaxSampledTexturesPerStage, MaxSamplersPerStage and MaxUniformBuffersPerStage
	// are how many resources of each kind one shader stage can read.
	MaxSampledTexturesPerStage, MaxSamplersPerStage, MaxUniformBuffersPerStage int
	// UniformOffsetAlignment is what dynamic uniform offsets are a multiple of.
	UniformOffsetAlignment int
	// MaxUniformBindingSize is the largest uniform binding, in bytes.
	MaxUniformBindingSize int
	// MaxTextureSize is the largest width or height of a texture, in pixels.
	MaxTextureSize int
	// MaxVertexBuffers and MaxVertexAttributes are what a pipeline's vertex layout
	// can have.
	MaxVertexBuffers, MaxVertexAttributes int
	// CopyRowPitchAlignment is what the row pitch of a copy between a buffer and a
	// texture is a multiple of, in bytes.
	CopyRowPitchAlignment int
}

// DefaultLimits returns the design limits of SPEC §9.3.
//
// Flutter: no counterpart
func DefaultLimits() Limits {
	return Limits{
		MaxBindGroups:              4,
		MaxSampledTexturesPerStage: 16,
		MaxSamplersPerStage:        16,
		MaxUniformBuffersPerStage:  12,
		UniformOffsetAlignment:     256,
		MaxUniformBindingSize:      64 << 10,
		MaxTextureSize:             8192,
		MaxVertexBuffers:           8,
		MaxVertexAttributes:        16,
		CopyRowPitchAlignment:      256,
	}
}

// AlignedRowPitch returns the row pitch, in bytes, of a copy between a buffer and a
// texture of the width and the bytes per pixel: the row rounded up to a multiple of
// CopyRowPitchAlignment.
//
// Flutter: no counterpart
func (l Limits) AlignedRowPitch(width, bytesPerPixel int) int {
	return alignUp(width*bytesPerPixel, l.CopyRowPitchAlignment)
}

func alignUp(n, alignment int) int {
	if alignment <= 1 {
		return n
	}
	return (n + alignment - 1) / alignment * alignment
}

// exceeded panics with GPULimitExceeded in debug builds when value is more than max
// (SPEC §9.3, §14.2). In release builds the backend's own validation answers.
func exceeded(what string, value, max int) {
	if !foundation.DebugMode || value <= max {
		return
	}
	panic(errors.New("GPULimitExceeded", "a call breaks a design limit of the explicit renderer",
		"limit", what, "value", value, "max", max))
}

// misaligned panics with GPULimitExceeded in debug builds when value is not a
// multiple of alignment.
func misaligned(what string, value, alignment int) {
	if !foundation.DebugMode || alignment <= 1 || value%alignment == 0 {
		return
	}
	panic(errors.New("GPULimitExceeded", "a call breaks a design limit of the explicit renderer",
		"limit", what, "value", value, "alignment", alignment))
}

// CheckTexture panics with GPULimitExceeded in debug builds when the texture is
// larger than MaxTextureSize. Backends and the fake device call it from NewTexture.
//
// Flutter: no counterpart
func (l Limits) CheckTexture(desc TextureDesc) {
	exceeded("texture width", desc.Width, l.MaxTextureSize)
	exceeded("texture height", desc.Height, l.MaxTextureSize)
}

// CheckPipeline panics with GPULimitExceeded in debug builds when the pipeline has
// more vertex buffers or attributes, or its entry points read more resources of a
// kind or from more bind groups, than the limits allow. Backends and the fake
// device call it from NewPipeline.
//
// Flutter: no counterpart
func (l Limits) CheckPipeline(desc PipelineDesc) {
	exceeded("vertex buffers", len(desc.VertexBuffers), l.MaxVertexBuffers)
	attributes := 0
	for _, b := range desc.VertexBuffers {
		attributes += len(b.Attributes)
	}
	exceeded("vertex attributes", attributes, l.MaxVertexAttributes)
	if desc.Shader == nil {
		return
	}
	for _, name := range []string{desc.VertexEntry, desc.FragmentEntry} {
		entry, ok := desc.Shader.Entry(name)
		if !ok {
			continue
		}
		var textures, samplers, uniforms int
		for _, b := range entry.Bindings {
			exceeded("bind group index", b.Group+1, l.MaxBindGroups)
			switch b.Kind {
			case BindingSampledTexture:
				textures++
			case BindingSampler:
				samplers++
			case BindingUniformBuffer:
				uniforms++
			}
		}
		exceeded("sampled textures per stage", textures, l.MaxSampledTexturesPerStage)
		exceeded("samplers per stage", samplers, l.MaxSamplersPerStage)
		exceeded("uniform buffers per stage", uniforms, l.MaxUniformBuffersPerStage)
	}
}

// CheckBindGroup panics with GPULimitExceeded in debug builds when the group index
// is not below MaxBindGroups, or a uniform binding is larger than
// MaxUniformBindingSize. Backends and the fake device call it from NewBindGroup.
//
// Flutter: no counterpart
func (l Limits) CheckBindGroup(group int, entries []Binding) {
	exceeded("bind group index", group+1, l.MaxBindGroups)
	for _, e := range entries {
		if e.Buffer != nil {
			size := e.Size
			if size == 0 {
				size = e.Buffer.Size() - e.Offset
			}
			exceeded("uniform binding size", size, l.MaxUniformBindingSize)
		}
	}
}

// CheckSetBindGroup panics with GPULimitExceeded in debug builds when the group
// index is not below MaxBindGroups or a dynamic offset is not a multiple of
// UniformOffsetAlignment. Backends and the fake device call it from SetBindGroup.
//
// Flutter: no counterpart
func (l Limits) CheckSetBindGroup(group int, dynamicOffsets []uint32) {
	exceeded("bind group index", group+1, l.MaxBindGroups)
	for _, o := range dynamicOffsets {
		misaligned("dynamic uniform offset", int(o), l.UniformOffsetAlignment)
	}
}

// CheckSetVertexBuffer panics with GPULimitExceeded in debug builds when the slot is
// not below MaxVertexBuffers. Backends and the fake device call it from
// SetVertexBuffer.
//
// Flutter: no counterpart
func (l Limits) CheckSetVertexBuffer(slot int) {
	exceeded("vertex buffer slot", slot+1, l.MaxVertexBuffers)
}
