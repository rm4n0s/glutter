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
	// MaxTextureSize3D is the largest width, height or depth of a 3D texture, in pixels.
	MaxTextureSize3D int
	// MaxTextureArrayLayers is the largest number of layers of a 2D array texture.
	MaxTextureArrayLayers int
	// MaxColorAttachments is the number of color attachments of a pass and of a pipeline.
	MaxColorAttachments int
	// MaxColorAttachmentBytesPerSample is the budget of the color attachments of a pass, in the
	// bytes of TextureFormat.RenderTargetPixelBytes added up.
	MaxColorAttachmentBytesPerSample int
	// MaxStorageBuffersPerStage and MaxStorageTexturesPerStage are how many storage resources of each
	// kind one shader stage can read or write.
	MaxStorageBuffersPerStage, MaxStorageTexturesPerStage int
	// MaxStorageBindingSize is the largest storage buffer binding, in bytes.
	MaxStorageBindingSize int
	// StorageOffsetAlignment is what the offset of a storage binding is a multiple of, and the offset
	// of ResolveQuerySet.
	StorageOffsetAlignment int
	// MaxBufferSize is the largest buffer, in bytes.
	MaxBufferSize int
	// MaxComputeInvocationsPerWorkgroup is the most threads of a workgroup (the product of the
	// three sizes of @workgroup_size), and MaxComputeWorkgroupSize the most in each dimension.
	MaxComputeInvocationsPerWorkgroup int
	MaxComputeWorkgroupSize           [3]int
	// MaxComputeWorkgroupsPerDimension is the most workgroups a dispatch runs in a dimension.
	MaxComputeWorkgroupsPerDimension int
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

		MaxTextureSize3D:                 2048,
		MaxTextureArrayLayers:            256,
		MaxColorAttachments:              4,
		MaxColorAttachmentBytesPerSample: 32,

		MaxStorageBuffersPerStage:         8,
		MaxStorageTexturesPerStage:        4,
		MaxStorageBindingSize:             128 << 20,
		StorageOffsetAlignment:            256,
		MaxBufferSize:                     256 << 20,
		MaxComputeInvocationsPerWorkgroup: 256,
		MaxComputeWorkgroupSize:           [3]int{256, 256, 64},
		MaxComputeWorkgroupsPerDimension:  65535,
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
	panic(errors.New(tagGPULimitExceeded, "a call breaks a design limit of the explicit renderer",
		"limit", what, "value", value, "max", max))
}

// misaligned panics with GPULimitExceeded in debug builds when value is not a
// multiple of alignment.
func misaligned(what string, value, alignment int) {
	if !foundation.DebugMode || alignment <= 1 || value%alignment == 0 {
		return
	}
	panic(errors.New(tagGPULimitExceeded, "a call breaks a design limit of the explicit renderer",
		"limit", what, "value", value, "alignment", alignment))
}

// invalid panics with GPUDescriptorInvalid in debug builds when ok is false: a descriptor that
// no backend can make, whatever the limits are (a usage the format does not allow, mip levels on
// a multisampled texture). In release builds the backend's own validation answers.
func invalid(ok bool, what string, args ...any) {
	if !foundation.DebugMode || ok {
		return
	}
	panic(errors.New(tagGPUDescriptorInvalid, "a descriptor of the explicit renderer cannot be made", append([]any{"what", what}, args...)...))
}

// MipLevelsOf returns the number of levels of a full mip chain for a texture of the size: 1 plus
// the base-2 logarithm of the largest side, rounded down.
//
// Flutter: no counterpart
func MipLevelsOf(width, height int) int {
	n := 1
	for m := max(width, height); m > 1; m >>= 1 {
		n++
	}
	return n
}

// CheckTexture panics with GPULimitExceeded in debug builds when the texture is
// larger than the limits (MaxTextureSize, MaxTextureSize3D, MaxTextureArrayLayers), and with
// GPUDescriptorInvalid when the description cannot be made: a usage the
// format does not allow, a cube that is not square or has other than 6 layers, more mip levels than
// the size has, or a multisampled texture that is not a single level of a 2D texture that a pass
// draws into. Backends and the fake device call it from NewTexture.
//
// Flutter: no counterpart
func (l Limits) CheckTexture(desc TextureDesc) {
	n := desc.Normalized()
	maxSide := l.MaxTextureSize
	if n.Kind == TextureKind3D {
		maxSide = l.MaxTextureSize3D
	}
	exceeded("texture width", n.Width, maxSide)
	exceeded("texture height", n.Height, maxSide)
	switch n.Kind {
	case TextureKind2DArray:
		exceeded("texture array layers", n.DepthOrLayers, l.MaxTextureArrayLayers)
	case TextureKind3D:
		exceeded("texture depth", n.DepthOrLayers, maxSide)
	}
	if !foundation.DebugMode {
		return
	}
	if n.Format == TextureFormatUnspecified || n.Width < 1 || n.Height < 1 || n.DepthOrLayers < 1 {
		return // the backends return an error for a texture without a size or a format
	}
	if foundation.DebugMode && (n.Kind < TextureKind2D || n.Kind > TextureKind3D) {
		invalid(false, "texture kind", "kind", int(n.Kind))
	}
	if foundation.DebugMode && n.Usage&^n.Format.AllowedUsage() != 0 {
		invalid(false, "texture usage for the format", "format", int(n.Format), "usage", int(n.Usage), "allowed", int(n.Format.AllowedUsage()))
	}
	if foundation.DebugMode && n.MipLevels > MipLevelsOf(n.Width, n.Height) {
		invalid(false, "mip levels", "levels", n.MipLevels, "size_allows", MipLevelsOf(n.Width, n.Height))
	}
	switch n.Kind {
	case TextureKind2D:
		if foundation.DebugMode && n.DepthOrLayers != 1 {
			invalid(false, "layers of a 2D texture", "layers", n.DepthOrLayers)
		}
	case TextureKindCube:
		if foundation.DebugMode && n.DepthOrLayers != 6 {
			invalid(false, "layers of a cube", "layers", n.DepthOrLayers)
		}
		if foundation.DebugMode && n.Width != n.Height {
			invalid(false, "cube faces are square", "width", n.Width, "height", n.Height)
		}
	case TextureKind3D:
		if foundation.DebugMode && n.Usage&TextureRenderTarget != 0 {
			invalid(false, "a 3D texture cannot be drawn into")
		}
		if foundation.DebugMode && n.Format.IsDepthStencil() {
			invalid(false, "a 3D texture has no depth format")
		}
	}
	if foundation.DebugMode && (n.SampleCount != 1 && n.SampleCount != 4) {
		invalid(false, "sample count", "samples", n.SampleCount)
	}
	if n.SampleCount > 1 {
		if foundation.DebugMode && (n.Kind != TextureKind2D || n.MipLevels != 1) {
			invalid(false, "a multisampled texture is one level of a 2D texture", "kind", int(n.Kind), "levels", n.MipLevels)
		}
		if foundation.DebugMode && n.Usage&TextureRenderTarget == 0 {
			invalid(false, "a multisampled texture is drawn into")
		}
		if foundation.DebugMode && n.Usage&^TextureRenderTarget != 0 {
			invalid(false, "a multisampled texture is only drawn into and resolved", "usage", int(n.Usage))
		}
	}
}

// CheckRead panics with GPUDescriptorInvalid in debug builds when a texture is read that has no
// TextureCopySource or TextureRenderTarget, is of a format that cannot be copied, is not a 2D texture (read a Layer), or,
// for ReadTexture (bytes false), is not one of the 8 bit formats that it converts to RGBA. Backends
// and the fake device call it from ReadTexture and ReadTextureBytes.
//
// Flutter: no counterpart
func (l Limits) CheckRead(desc TextureDesc, bytes bool) {
	n := desc.Normalized()
	if foundation.DebugMode && n.Usage&(TextureCopySource|TextureRenderTarget) == 0 {
		invalid(false, "a texture that is read has TextureCopySource or TextureRenderTarget")
	}
	if foundation.DebugMode && !n.Format.CanBeCopied() {
		invalid(false, "a texture that is read has a format that can be copied", "format", int(n.Format))
	}
	if foundation.DebugMode && (n.Kind != TextureKind2D || n.SampleCount != 1) {
		invalid(false, "a texture that is read is 2D with one sample (read a Layer of an array or a cube; resolve a multisampled one)", "kind", int(n.Kind), "samples", n.SampleCount)
	}
	if !bytes {
		if foundation.DebugMode && (n.Format.BytesPerPixel() > 4 || n.Format.IsDepthStencil() || n.Format.SampleType(Features{}) != SampleTypeFloat) {
			invalid(false, "ReadTexture converts the 8 bit formats; use ReadTextureBytes", "format", int(n.Format))
		}
	}
}

// CheckLayer panics with GPUDescriptorInvalid in debug builds when Texture.Layer is asked for a
// level or layer the texture does not have, or for a 3D texture. Backends and the fake device call
// it from Layer.
//
// Flutter: no counterpart
func (l Limits) CheckLayer(desc TextureDesc, level, layer int) {
	n := desc.Normalized()
	if foundation.DebugMode && n.Kind == TextureKind3D {
		invalid(false, "a 3D texture has no layers")
	}
	if foundation.DebugMode && (level < 0 || level >= n.MipLevels) {
		invalid(false, "layer of a mip level the texture has", "level", level, "levels", n.MipLevels)
	}
	if foundation.DebugMode && (layer < 0 || layer >= n.Layers()) {
		invalid(false, "a layer the texture has", "layer", layer, "layers", n.Layers())
	}
}

// checkVertexAttribute panics with GPUDescriptorInvalid in debug builds when a vertex attribute's
// format is unspecified or does not fit in the stride of its buffer.
func checkVertexAttribute(stride int, a VertexAttribute) {
	if foundation.DebugMode && a.Format.Size() <= 0 {
		invalid(false, "vertex format", "location", a.Location)
	}
	if foundation.DebugMode && (a.Offset < 0 || a.Offset+a.Format.Size() > stride) {
		invalid(false, "vertex attribute inside the stride", "location", a.Location, "offset", a.Offset, "size", a.Format.Size(), "stride", stride)
	}
}

// CheckPass panics with GPULimitExceeded in debug builds when the pass has more color
// attachments, or color attachments of more bytes for each sample, than the limits allow, and with
// GPUDescriptorInvalid when it has no attachment, an attachment that is not a render target or
// has the wrong format, attachments of different sizes or sample counts, or a resolve target that
// is not a single-sample texture of the size and format of its attachment. Backends and the fake
// device call it from BeginRenderPass.
//
// Flutter: no counterpart
func (l Limits) CheckPass(desc PassDesc) {
	exceeded("color attachments", len(desc.Color), l.MaxColorAttachments)
	if !foundation.DebugMode {
		return
	}
	if foundation.DebugMode && (len(desc.Color) <= 0 && desc.Depth == nil) {
		invalid(false, "a pass has an attachment")
	}
	bytes := 0
	var width, height, samples int
	check := func(t Texture, what string) {
		d := t.Desc()
		if foundation.DebugMode && d.Usage&TextureRenderTarget == 0 {
			invalid(false, what+" is a render target")
		}
		if foundation.DebugMode && d.Kind != TextureKind2D {
			invalid(false, what+" is a 2D texture (use Layer for a layer of an array or a cube)", "kind", int(d.Kind))
		}
		if width == 0 {
			width, height, samples = d.Width, d.Height, d.SampleCount
			return
		}
		if foundation.DebugMode && (d.Width != width || d.Height != height) {
			invalid(false, "the attachments have one size", "width", d.Width, "height", d.Height, "want_width", width, "want_height", height)
		}
		if foundation.DebugMode && d.SampleCount != samples {
			invalid(false, "the attachments have one sample count", "samples", d.SampleCount, "want", samples)
		}
	}
	for _, c := range desc.Color {
		if foundation.DebugMode && c.Target == nil {
			invalid(false, "a color attachment has a texture")
		}
		if c.Target == nil {
			continue
		}
		d := c.Target.Desc()
		if foundation.DebugMode && d.Format.IsDepthStencil() {
			invalid(false, "a color attachment has a color format", "format", int(d.Format))
		}
		bytes += d.Format.RenderTargetPixelBytes()
		check(c.Target, "a color attachment")
		if c.Resolve != nil {
			r := c.Resolve.Desc()
			if foundation.DebugMode && d.SampleCount <= 1 {
				invalid(false, "only a multisampled attachment is resolved")
			}
			if foundation.DebugMode && (r.SampleCount != 1 || r.Format != d.Format) {
				invalid(false, "a resolve target has one sample and the format of its attachment", "format", int(r.Format), "samples", r.SampleCount)
			}
			if foundation.DebugMode && (r.Width != d.Width || r.Height != d.Height) {
				invalid(false, "a resolve target has the size of its attachment", "width", r.Width, "height", r.Height, "want_width", d.Width, "want_height", d.Height)
			}
			if foundation.DebugMode && (r.Usage&TextureRenderTarget == 0 || r.Kind != TextureKind2D) {
				invalid(false, "a resolve target is a 2D render target")
			}
		}
	}
	exceeded("color attachment bytes per sample", bytes, l.MaxColorAttachmentBytesPerSample)
	if desc.Depth != nil {
		if foundation.DebugMode && desc.Depth.Target == nil {
			invalid(false, "the depth attachment has a texture")
		}
		if desc.Depth.Target != nil {
			if foundation.DebugMode && !desc.Depth.Target.Desc().Format.IsDepthStencil() {
				invalid(false, "the depth attachment has a depth format", "format", int(desc.Depth.Target.Desc().Format))
			}
			check(desc.Depth.Target, "the depth attachment")
		}
	}
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
	l.checkPipelineState(desc)
	if desc.Shader == nil {
		return
	}
	for _, name := range []string{desc.VertexEntry, desc.FragmentEntry} {
		entry, ok := desc.Shader.Entry(name)
		if !ok {
			continue
		}
		l.checkStageBindings(entry)
	}
}

// checkPipelineState checks the targets, the depth-stencil state and the primitives of a pipeline.
func (l Limits) checkPipelineState(desc PipelineDesc) {
	targets := desc.Targets()
	exceeded("color targets", len(targets), l.MaxColorAttachments)
	if !foundation.DebugMode {
		return
	}
	if foundation.DebugMode && (desc.ColorTargets != nil && (desc.Target != TextureFormatUnspecified || desc.Blend != BlendState{})) {
		invalid(false, "ColorTargets replace Target and Blend")
	}
	bytes := 0
	for _, t := range targets {
		if t.Format == TextureFormatUnspecified {
			continue // the shaders and vertex layouts of tests have no target
		}
		if foundation.DebugMode && (t.Format.AllowedUsage()&TextureRenderTarget == 0 || t.Format.IsDepthStencil()) {
			invalid(false, "a color target has a renderable color format", "format", int(t.Format))
		}
		bytes += t.Format.RenderTargetPixelBytes()
	}
	exceeded("color target bytes per sample", bytes, l.MaxColorAttachmentBytesPerSample)
	if foundation.DebugMode && (desc.Topology < TopologyTriangleList || desc.Topology > TopologyPointList) {
		invalid(false, "topology", "topology", int(desc.Topology))
	}
	if foundation.DebugMode && (desc.Samples() != 1 && desc.Samples() != 4) {
		invalid(false, "sample count", "samples", desc.SampleCount)
	}
	for _, vb := range desc.VertexBuffers {
		for _, a := range vb.Attributes {
			checkVertexAttribute(vb.Stride, a)
		}
	}
	ds := desc.DepthStencil
	if ds.Format == TextureFormatUnspecified {
		if foundation.DebugMode && (ds.DepthWrite || ds.DepthBias != 0 || ds.DepthBiasSlopeScale != 0 || ds.DepthBiasClamp != 0) {
			invalid(false, "a depth state needs a depth format")
		}
		return
	}
	if foundation.DebugMode && !ds.Format.IsDepthStencil() {
		invalid(false, "the depth-stencil state has a depth format", "format", int(ds.Format))
	}
	if ds.DepthBias != 0 || ds.DepthBiasSlopeScale != 0 || ds.DepthBiasClamp != 0 {
		if foundation.DebugMode && (desc.Topology != TopologyTriangleList && desc.Topology != TopologyTriangleStrip) {
			invalid(false, "only triangles have a depth bias", "topology", int(desc.Topology))
		}
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
		if e.Buffer != nil && e.Buffer.Usage()&BufferStorage == 0 {
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGPUDescriptorInvalid = "GPUDescriptorInvalid"
	tagGPULimitExceeded     = "GPULimitExceeded"
)
