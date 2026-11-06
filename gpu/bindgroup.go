package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// user is a resource that a bind group or a command holds: it is checked when the commands are submitted.
type user interface{ usable(call string) }

// boundBuffer is a buffer range of a bind group.
type boundBuffer struct {
	buf     *Buffer
	offset  int
	size    int
	uniform bool
}

// BindGroup is the resources a pipeline's shaders read in one group: buffers, texture views and samplers, one for
// each binding of the group's layout (WebGPU's GPUBindGroup). A pass binds one with SetBindGroup. The uniform
// buffers of a group have a dynamic offset: SetBindGroup gives one for each, so that one buffer holds the uniforms of
// many draws.
//
// Flutter: no counterpart
type BindGroup struct {
	resource
	layout  *BindGroupLayout
	entries []explicit.Binding
	buffers []boundBuffer
	uses    []user
}

// NewBindGroup makes a bind group for a layout (WebGPU's GPUDevice.createBindGroup). Every binding of the layout
// has exactly one entry. It fails with GPUInvalidDescriptor for an entry that does not fit its binding (the wrong
// resource, a buffer without the use, a range outside the buffer or smaller than what the shader reads, a texture
// view of another dimension or sample type, a storage texture of another format, a resource of another device)
// and with GPULimitExceeded for a binding range over the limits or an offset that is not aligned.
//
// A uniform entry's Offset is a multiple of Limits.MinUniformBufferOffsetAlignment, a storage entry's of
// Limits.MinStorageBufferOffsetAlignment. A Size of 0 is up to the end of the buffer.
//
// Flutter: no counterpart
func (d *Device) NewBindGroup(desc BindGroupDesc) (*BindGroup, error) {
	d.assertThread("NewBindGroup")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	l := desc.Layout
	if l == nil {
		return nil, errors.New(TagGPUInvalidDescriptor, "a bind group has a layout", "field", "Layout", "label", desc.Label)
	}
	if len(desc.Entries) != len(l.bindings) {
		return nil, errors.New(TagGPUInvalidDescriptor, "a bind group has one entry for each binding of its layout", "field", "Entries",
			"value", len(desc.Entries), "want", len(l.bindings), "label", desc.Label)
	}
	g := &BindGroup{layout: l}
	seen := map[int]bool{}
	for _, e := range desc.Entries {
		sb, ok := l.binding(e.Binding)
		if !ok || seen[e.Binding] {
			return nil, errors.New(TagGPUInvalidDescriptor, "an entry is for a binding of the layout, once", "field", "Entries.Binding", "value", e.Binding, "label", desc.Label)
		}
		seen[e.Binding] = true
		eb := explicit.Binding{Binding: e.Binding}
		switch sb.Kind {
		case BindingUniformBuffer, BindingStorageBufferReadOnly, BindingStorageBuffer:
			b := e.Buffer
			if b == nil || b.d != d || e.TextureView != nil || e.Sampler != nil {
				return nil, errors.New(TagGPUInvalidDescriptor, "a buffer binding has a buffer of this device and nothing else", "field", "Entries.Buffer", "binding", sb.Name, "label", desc.Label)
			}
			b.usable("NewBindGroup")
			uniform := sb.Kind == BindingUniformBuffer
			need, align, limit := BufferUsageStorage, d.limits.StorageOffsetAlignment, d.limits.MaxStorageBindingSize
			if uniform {
				need, align, limit = BufferUsageUniform, d.limits.UniformOffsetAlignment, d.limits.MaxUniformBindingSize
			}
			if b.usage&need == 0 {
				return nil, errors.New(TagGPUInvalidDescriptor, "the buffer of a binding has the use that the binding needs", "field", "Entries.Buffer", "binding", sb.Name,
					"usage", int(b.usage), "need", int(need), "label", desc.Label)
			}
			size := e.Size
			if size == 0 {
				size = b.size - e.Offset
			}
			if e.Offset < 0 || size < 1 || e.Offset+size > b.size {
				return nil, errors.New(TagGPUInvalidDescriptor, "the range of a binding is inside its buffer", "field", "Entries.Offset", "binding", sb.Name,
					"offset", e.Offset, "size", size, "buffer", b.size, "label", desc.Label)
			}
			if e.Offset%align != 0 {
				return nil, errors.New(TagGPULimitExceeded, "the offset of a binding is a multiple of the alignment of the device", "field", "Entries.Offset", "binding", sb.Name,
					"value", e.Offset, "alignment", align, "label", desc.Label)
			}
			if size > limit {
				return nil, errors.New(TagGPULimitExceeded, "the range of a binding is over the limit of the device", "field", "Entries.Size", "binding", sb.Name,
					"value", size, "max", limit, "label", desc.Label)
			}
			if size < sb.Size {
				return nil, errors.New(TagGPUInvalidDescriptor, "the range of a binding holds what the shader reads", "field", "Entries.Size", "binding", sb.Name,
					"value", size, "need", sb.Size, "label", desc.Label)
			}
			eb.Buffer, eb.Offset, eb.Size = b.exp, e.Offset, size
			g.buffers = append(g.buffers, boundBuffer{buf: b, offset: e.Offset, size: size, uniform: uniform})
			g.uses = append(g.uses, b)
		case BindingSampledTexture, BindingStorageTexture:
			v := e.TextureView
			if v == nil || v.tex.d != d || e.Buffer != nil || e.Sampler != nil {
				return nil, errors.New(TagGPUInvalidDescriptor, "a texture binding has a texture view of this device and nothing else", "field", "Entries.TextureView", "binding", sb.Name, "label", desc.Label)
			}
			v.tex.usable("NewBindGroup")
			td := v.desc
			if sb.Kind == BindingSampledTexture {
				if v.tex.desc.Usage&explicit.TextureSampled == 0 {
					return nil, errors.New(TagGPUInvalidDescriptor, "a sampled texture has TextureUsageTextureBinding", "field", "Entries.TextureView", "binding", sb.Name, "label", desc.Label)
				}
				if td.Kind.Dimension() != explicit.TextureDimension(sb.Dimension) {
					return nil, errors.New(TagGPUInvalidDescriptor, "the view has the dimension that the shader declares", "field", "Entries.TextureView", "binding", sb.Name,
						"kind", int(td.Kind), "dimension", int(sb.Dimension), "label", desc.Label)
				}
				if st := td.Format.SampleType(d.feat); !st.Fits(explicit.SampleType(sb.Sample)) {
					return nil, errors.New(TagGPUInvalidDescriptor, "the format of the view has the sample type that the shader declares", "field", "Entries.TextureView", "binding", sb.Name,
						"format", int(td.Format), "label", desc.Label)
				}
			} else {
				if v.tex.desc.Usage&explicit.TextureStorage == 0 {
					return nil, errors.New(TagGPUInvalidDescriptor, "a storage texture has TextureUsageStorageBinding", "field", "Entries.TextureView", "binding", sb.Name, "label", desc.Label)
				}
				if f := storageFormats[sb.StorageFormat]; td.Format != f {
					return nil, errors.New(TagGPUInvalidDescriptor, "a storage texture has the format that the shader declares", "field", "Entries.TextureView", "binding", sb.Name,
						"format", int(td.Format), "declared", sb.StorageFormat, "label", desc.Label)
				}
				if td.MipLevels != 1 {
					return nil, errors.New(TagGPUInvalidDescriptor, "a storage texture view has one mip level: use LevelView", "field", "Entries.TextureView", "binding", sb.Name, "label", desc.Label)
				}
			}
			eb.Texture = v.exp
			g.uses = append(g.uses, v.tex)
		case BindingSampler:
			s := e.Sampler
			if s == nil || s.d != d || e.Buffer != nil || e.TextureView != nil {
				return nil, errors.New(TagGPUInvalidDescriptor, "a sampler binding has a sampler of this device and nothing else", "field", "Entries.Sampler", "binding", sb.Name, "label", desc.Label)
			}
			s.usable("NewBindGroup")
			eb.Sampler = s.exp
			g.uses = append(g.uses, s)
		}
		g.entries = append(g.entries, eb)
	}
	g.resource = d.track(kindBindGroup, desc.Label, 1)
	return g, nil
}

// Layout returns the layout the group was made for.
//
// Flutter: no counterpart
func (g *BindGroup) Layout() *BindGroupLayout { return g.layout }

// Release forgets the group. The commands that were recorded with it keep working. It may be called twice.
//
// Flutter: no counterpart
func (g *BindGroup) Release() {
	g.d.assertThread("BindGroup.Release")
	g.untrack()
}

// check panics in debug builds when the group cannot be bound at the dynamic offsets: the count, the alignment, and
// the ranges that the offsets move must stay inside their buffers.
func (g *BindGroup) check(call string, offsets []uint32) {
	if !foundation.DebugMode {
		return
	}
	l := g.layout
	if len(offsets) != l.dynamicCount() {
		misuse(false, call, "SetBindGroup takes one dynamic offset for each uniform buffer of the group", "offsets", len(offsets), "uniform_buffers", l.dynamicCount())
	}
	i := 0
	for _, b := range g.buffers {
		if !b.uniform {
			continue
		}
		if i >= len(offsets) {
			return
		}
		off := int(offsets[i])
		i++
		if off%g.d.limits.UniformOffsetAlignment != 0 {
			misuse(false, call, "a dynamic offset is a multiple of Limits.MinUniformBufferOffsetAlignment", "offset", off)
		}
		if b.offset+off+b.size > b.buf.size {
			misuse(false, call, "the range of a uniform binding moved by its dynamic offset is inside the buffer",
				"offset", b.offset+off, "size", b.size, "buffer", b.buf.size)
		}
	}
}

// usableAll panics in debug builds when a resource of the group was released.
func (g *BindGroup) usableAll(call string) {
	g.usable(call)
	for _, u := range g.uses {
		u.usable(call)
	}
}
