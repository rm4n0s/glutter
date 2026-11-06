package explicit

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// The checks of the calls of step 30.2 (SPEC §27.1): buffers, bindings, compute, indirect draws, copies
// and queries. Like the other checks of this package, they panic in debug builds only (§14.2), and a
// backend or the fake device calls them from the method they belong to.

// CheckBuffer panics with GPULimitExceeded in debug builds when the buffer is larger than MaxBufferSize,
// and with GPUDescriptorInvalid when it has no usage. Backends and the fake device call it from
// NewBuffer.
//
// Flutter: no counterpart
func (l Limits) CheckBuffer(size int, usage BufferUsage) {
	exceeded("buffer size", size, l.MaxBufferSize)
	if foundation.DebugMode && (size < 0) {
		invalid(false, "a buffer has a size", "size", size)
	}
}

// CheckBinding panics with GPUDescriptorInvalid in debug builds when an entry of a bind group does not
// fit the binding of the shader it is for: a uniform binding needs a buffer with BufferUniform, a storage
// binding one with BufferStorage at an offset that is a multiple of StorageOffsetAlignment and of a size that
// MaxStorageBindingSize allows, a texture binding a texture and a sampler binding a sampler, and a storage
// texture binding a texture with TextureStorage. Backends and the fake device call it from NewBindGroup
// and NewComputeBindGroup, for each entry whose binding they found.
//
// Flutter: no counterpart
func (l Limits) CheckBinding(sb ShaderBinding, e Binding) {
	switch sb.Kind {
	case BindingUniformBuffer:
		if foundation.DebugMode && (e.Buffer == nil || e.Buffer.Usage()&BufferUniform == 0) {
			invalid(false, "a uniform binding has a buffer with BufferUniform", "binding", e.Binding)
		}
	case BindingStorageBufferReadOnly, BindingStorageBuffer:
		if foundation.DebugMode && (e.Buffer == nil || e.Buffer.Usage()&BufferStorage == 0) {
			invalid(false, "a storage binding has a buffer with BufferStorage", "binding", e.Binding)
		}
		if e.Buffer == nil {
			return
		}
		misaligned("storage binding offset", e.Offset, l.StorageOffsetAlignment)
		size := e.Size
		if size == 0 {
			size = e.Buffer.Size() - e.Offset
		}
		exceeded("storage binding size", size, l.MaxStorageBindingSize)
		if foundation.DebugMode && (e.Offset < 0 || e.Offset+size > e.Buffer.Size() || size <= 0) {
			invalid(false, "a storage binding is inside its buffer",
				"binding", e.Binding, "offset", e.Offset, "size", size, "buffer", e.Buffer.Size())
		}
	case BindingStorageTexture:
		if foundation.DebugMode && (e.Texture == nil || e.Texture.Desc().Usage&TextureStorage == 0) {
			invalid(false, "a storage texture binding has a texture with TextureStorage", "binding", e.Binding)
		}
		if e.Texture != nil {
			if foundation.DebugMode && (e.Texture.Desc().Format != sb.StorageFormat) {
				invalid(false, "a storage texture has the format the shader declares",
					"binding", e.Binding, "format", int(e.Texture.Desc().Format), "declared", int(sb.StorageFormat))
			}
		}
	case BindingSampledTexture:
		if foundation.DebugMode && (e.Texture == nil) {
			invalid(false, "a texture binding has a texture", "binding", e.Binding)
		}
	case BindingSampler:
		if foundation.DebugMode && (e.Sampler == nil) {
			invalid(false, "a sampler binding has a sampler", "binding", e.Binding)
		}
	}
}

// CheckComputePipeline panics with GPULimitExceeded in debug builds when the compute entry point has a
// workgroup larger than the limits, or reads more storage resources than a stage may, and with
// GPUDescriptorInvalid when the shader has no compute entry point of that name. Backends and the fake
// device call it from NewComputePipeline.
//
// Flutter: no counterpart
func (l Limits) CheckComputePipeline(desc ComputePipelineDesc) {
	if desc.Shader == nil {
		return
	}
	e, ok := desc.Shader.Entry(desc.Entry)
	if !ok {
		return // the backends return an error for a name the shader lacks
	}
	if foundation.DebugMode && (e.Stage != ShaderStageCompute) {
		invalid(false, "the entry point of a compute pipeline is a compute entry point", "entry", desc.Entry, "stage", e.Stage.String())
	}
	w := e.WorkgroupSize
	for i, n := range w {
		exceeded("workgroup size", n, l.MaxComputeWorkgroupSize[i])
	}
	exceeded("workgroup invocations", w[0]*w[1]*w[2], l.MaxComputeInvocationsPerWorkgroup)
	l.checkStageBindings(e)
}

// checkStageBindings checks the number of resources of each kind an entry point reads.
func (l Limits) checkStageBindings(e EntryPoint) {
	var textures, samplers, uniforms, storageBuffers, storageTextures int
	for _, b := range e.Bindings {
		exceeded("bind group index", b.Group+1, l.MaxBindGroups)
		switch b.Kind {
		case BindingSampledTexture:
			textures++
		case BindingSampler:
			samplers++
		case BindingUniformBuffer:
			uniforms++
		case BindingStorageBufferReadOnly, BindingStorageBuffer:
			storageBuffers++
		case BindingStorageTexture:
			storageTextures++
		}
		if e.Stage != ShaderStageCompute {
			if foundation.DebugMode && (b.Kind == BindingStorageBuffer || b.Kind == BindingStorageTexture) {
				invalid(false, "only a compute entry point writes storage", "entry", e.Name, "binding", b.Name)
			}
		}
	}
	exceeded("sampled textures per stage", textures, l.MaxSampledTexturesPerStage)
	exceeded("samplers per stage", samplers, l.MaxSamplersPerStage)
	exceeded("uniform buffers per stage", uniforms, l.MaxUniformBuffersPerStage)
	exceeded("storage buffers per stage", storageBuffers, l.MaxStorageBuffersPerStage)
	exceeded("storage textures per stage", storageTextures, l.MaxStorageTexturesPerStage)
}

// CheckDispatch panics with GPULimitExceeded in debug builds when a dimension of a dispatch is more
// than MaxComputeWorkgroupsPerDimension, and with GPUDescriptorInvalid when one is negative.
// Backends and the fake device call it from Dispatch.
//
// Flutter: no counterpart
func (l Limits) CheckDispatch(x, y, z int) {
	if foundation.DebugMode && (x < 0 || y < 0 || z < 0) {
		invalid(false, "a dispatch has no negative dimension", "x", x, "y", y, "z", z)
	}
	exceeded("dispatch x", x, l.MaxComputeWorkgroupsPerDimension)
	exceeded("dispatch y", y, l.MaxComputeWorkgroupsPerDimension)
	exceeded("dispatch z", z, l.MaxComputeWorkgroupsPerDimension)
}

// CheckIndirect panics with GPUDescriptorInvalid in debug builds when the buffer of an indirect draw
// or dispatch has no BufferIndirect, the offset is not a multiple of 4, or the arguments (size bytes)
// are not inside the buffer. Backends and the fake device call it from DrawIndirect,
// DrawIndexedIndirect and DispatchIndirect.
//
// Flutter: no counterpart
func (l Limits) CheckIndirect(b Buffer, offset, size int) {
	if foundation.DebugMode && (b.Usage()&BufferIndirect == 0) {
		invalid(false, "the buffer of an indirect call has BufferIndirect")
	}
	if foundation.DebugMode && (offset < 0 || offset%4 != 0) {
		invalid(false, "the offset of an indirect call is a multiple of 4", "offset", offset)
	}
	if foundation.DebugMode && (offset+size > b.Size()) {
		invalid(false, "the arguments of an indirect call are inside the buffer", "offset", offset, "size", size, "buffer", b.Size())
	}
}

// CheckFirstIndices panics with GPUFirstInstanceRead in debug builds when a draw has a first instance,
// a first vertex or a base vertex other than 0 and the vertex entry point reads instance_index or
// vertex_index: Direct3D 12's SV_InstanceID and SV_VertexID leave them out, so the rule holds on every
// backend, to show on Linux too (SPEC §27.1). Backends and the fake device call it from Draw and
// DrawIndexed.
//
// Flutter: no counterpart
func CheckFirstIndices(e EntryPoint, firstVertex, baseVertex, firstInstance int) {
	if e.ReadsInstanceIndex && firstInstance != 0 {
		panic(errors.New(tagGPUFirstInstanceRead, "a shader that reads instance_index is drawn with a first instance of 0 only",
			"entry", e.Name, "first_instance", firstInstance))
	}
	if e.ReadsVertexIndex && (firstVertex != 0 || baseVertex != 0) {
		panic(errors.New(tagGPUFirstInstanceRead, "a shader that reads vertex_index is drawn with a first vertex and a base vertex of 0 only",
			"entry", e.Name, "first_vertex", firstVertex, "base_vertex", baseVertex))
	}
}

// CheckCopyBuffer panics with GPUDescriptorInvalid in debug builds when the buffers of CopyBuffer
// lack BufferCopySource and BufferCopyDestination, the offsets or the size are not multiples of 4, or the
// ranges are not inside the buffers. Backends and the fake device call it from CopyBuffer.
//
// Flutter: no counterpart
func (l Limits) CheckCopyBuffer(src Buffer, srcOffset int, dst Buffer, dstOffset, size int) {
	if foundation.DebugMode && (src.Usage()&BufferCopySource == 0) {
		invalid(false, "the source of a copy has BufferCopySource")
	}
	if foundation.DebugMode && (dst.Usage()&BufferCopyDestination == 0) {
		invalid(false, "the destination of a copy has BufferCopyDestination")
	}
	if foundation.DebugMode && (srcOffset%4 != 0 || dstOffset%4 != 0 || size%4 != 0) {
		invalid(false, "the offsets and the size of a copy are multiples of 4",
			"src_offset", srcOffset, "dst_offset", dstOffset, "size", size)
	}
	if foundation.DebugMode && (srcOffset < 0 || dstOffset < 0 || size < 0 || srcOffset+size > src.Size() || dstOffset+size > dst.Size()) {
		invalid(false, "the ranges of a copy are inside the buffers", "src_offset", srcOffset, "dst_offset", dstOffset, "size", size, "src", src.Size(), "dst", dst.Size())
	}
	if foundation.DebugMode && (src == dst) {
		invalid(false, "a buffer is not copied onto itself")
	}
}

// CheckBufferTexture panics with GPUDescriptorInvalid in debug builds when a copy between a buffer and a
// texture breaks a rule: the box is not inside the level, the format cannot be copied (toTexture: cannot be
// written), the texture is multisampled, the layout is not aligned or does not fit the box, or the rows do not
// fit the buffer. Backends and the fake device call it from CopyBufferToTexture and CopyTextureToBuffer.
//
// Flutter: no counterpart
func (l Limits) CheckBufferTexture(buf Buffer, layout BufferLayout, tex TextureDesc, region TextureRegion, toTexture bool) {
	n := tex.Normalized()
	if toTexture {
		if foundation.DebugMode && (buf.Usage()&BufferCopySource == 0) {
			invalid(false, "the source of a copy has BufferCopySource")
		}
		if foundation.DebugMode && !(n.Format.CanBeWritten()) {
			invalid(false, "a texture that is written has a format that can be written", "format", int(n.Format))
		}
	} else {
		if foundation.DebugMode && (buf.Usage()&BufferCopyDestination == 0) {
			invalid(false, "the destination of a copy has BufferCopyDestination")
		}
		if foundation.DebugMode && (n.Usage&(TextureCopySource|TextureRenderTarget) == 0) {
			invalid(false, "a texture that is read has TextureCopySource or TextureRenderTarget")
		}
		if foundation.DebugMode && !(n.Format.CanBeCopied()) {
			invalid(false, "a texture that is read has a format that can be copied", "format", int(n.Format))
		}
	}
	if foundation.DebugMode && (n.SampleCount != 1) {
		invalid(false, "a multisampled texture is not copied", "samples", n.SampleCount)
	}
	w, h, d := region.Size[0], region.Size[1], region.Size[2]
	if foundation.DebugMode && (w <= 0 || h <= 0 || d <= 0) {
		invalid(false, "the box of a copy has a size", "size", region.Size)
	}
	if foundation.DebugMode && (region.Level < 0 || region.Level >= n.MipLevels) {
		invalid(false, "the level of a copy exists", "level", region.Level, "levels", n.MipLevels)
	}
	lw, lh, depth := max(n.Width>>max(region.Level, 0), 1), max(n.Height>>max(region.Level, 0), 1), n.Layers()
	if n.Kind == TextureKind3D {
		depth = max(n.DepthOrLayers>>max(region.Level, 0), 1)
	}
	o := region.Origin
	if foundation.DebugMode && (o[0] < 0 || o[1] < 0 || o[2] < 0 || o[0]+w > lw || o[1]+h > lh || o[2]+d > depth) {
		invalid(false, "the box of a copy is inside the level",
			"origin", region.Origin, "size", region.Size, "width", lw, "height", lh, "depth", depth)
	}
	bpp := n.Format.BytesPerPixel()
	if foundation.DebugMode && (layout.BytesPerRow%l.CopyRowPitchAlignment != 0 || layout.BytesPerRow < w*bpp) {
		invalid(false, "the rows of a copy are a multiple of CopyRowPitchAlignment apart and hold a row",
			"bytes_per_row", layout.BytesPerRow, "row", w*bpp)
	}
	if foundation.DebugMode && (layout.RowsPerImage < h) {
		invalid(false, "the rows per image of a copy hold the box", "rows_per_image", layout.RowsPerImage, "height", h)
	}
	if foundation.DebugMode && (layout.Offset < 0 || layout.Offset%4 != 0 || layout.Offset%max(bpp, 1) != 0) {
		invalid(false, "the offset of a copy is a multiple of 4 and of the bytes of a pixel", "offset", layout.Offset)
	}
	last := layout.Offset + layout.BytesPerRow*layout.RowsPerImage*(d-1) + layout.BytesPerRow*(h-1) + w*bpp
	if foundation.DebugMode && (last > buf.Size()) {
		invalid(false, "the rows of a copy are inside the buffer", "end", last, "buffer", buf.Size())
	}
}

// CheckReadBuffer panics with GPUDescriptorInvalid in debug builds when the buffer has no BufferCopySource
// or the range is not inside it. Backends and the fake device call it from ReadBuffer.
//
// Flutter: no counterpart
func (l Limits) CheckReadBuffer(b Buffer, offset, size int) {
	if foundation.DebugMode && (b.Usage()&BufferCopySource == 0) {
		invalid(false, "a buffer that is read has BufferCopySource")
	}
	if foundation.DebugMode && (offset < 0 || size < 0 || offset+size > b.Size()) {
		invalid(false, "the range of a read is inside the buffer", "offset", offset, "size", size, "buffer", b.Size())
	}
	if foundation.DebugMode && (offset%4 != 0 || size%4 != 0) {
		invalid(false, "the offset and the size of a read are multiples of 4", "offset", offset, "size", size)
	}
}

// CheckTimestamps panics with GPUDescriptorInvalid in debug builds when a pass writes timestamps that
// the device cannot (Features.TimestampQuery is false) or into queries the set lacks. Backends and the fake
// device call it from BeginRenderPass and BeginComputePass.
//
// Flutter: no counterpart
func (l Limits) CheckTimestamps(f Features, ts *PassTimestamps) {
	if ts == nil {
		return
	}
	if foundation.DebugMode && !(f.TimestampQuery) {
		invalid(false, "timestamps need Features.TimestampQuery")
	}
	if foundation.DebugMode && (ts.Set == nil) {
		invalid(false, "a pass writes its timestamps into a query set")
	}
	if ts.Set == nil {
		return
	}
	if foundation.DebugMode && (ts.Begin >= ts.Set.Count() || ts.End >= ts.Set.Count()) {
		invalid(false, "the timestamps of a pass are queries of the set", "begin", ts.Begin, "end", ts.End, "count", ts.Set.Count())
	}
	if foundation.DebugMode && (ts.Begin >= 0 && ts.End >= 0 && ts.Begin == ts.End) {
		invalid(false, "the two timestamps of a pass are two queries")
	}
}

// CheckResolveQuerySet panics with GPUDescriptorInvalid in debug builds when the queries are not in the
// set, the buffer has no BufferQueryResolve, or the 8 bytes of each do not fit it at an offset that is a
// multiple of StorageOffsetAlignment. Backends and the fake device call it from ResolveQuerySet.
//
// Flutter: no counterpart
func (l Limits) CheckResolveQuerySet(set QuerySet, first, count int, dst Buffer, dstOffset int) {
	if foundation.DebugMode && (first < 0 || count < 0 || first+count > set.Count()) {
		invalid(false, "the queries of a resolve are in the set", "first", first, "count", count, "set", set.Count())
	}
	if foundation.DebugMode && (dst.Usage()&BufferQueryResolve == 0) {
		invalid(false, "the buffer of a resolve has BufferQueryResolve")
	}
	misaligned("query resolve offset", dstOffset, l.StorageOffsetAlignment)
	if foundation.DebugMode && (dstOffset < 0 || dstOffset+8*count > dst.Size()) {
		invalid(false, "the timestamps of a resolve fit the buffer", "offset", dstOffset, "count", count, "buffer", dst.Size())
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUFirstInstanceRead is the tag of the debug panic of a draw with a first instance, first
	// vertex or base vertex other than 0 for a shader that reads instance_index or vertex_index
	// (SPEC §27.1); the metadata are "entry" and the values.
	tagGPUFirstInstanceRead = "GPUFirstInstanceRead"
)
