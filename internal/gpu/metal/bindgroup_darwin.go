//go:build darwin

package metal

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// bindGroup is the resources of one group of a pipeline. Metal has no descriptor tables to fill: the group
// holds the buffers, textures and samplers by binding number, and SetBindGroup sets each of them in the
// argument tables of the stages that read it, at the index of the shader's binding map (SPEC §29.5).
type bindGroup struct {
	d       *Device
	group   int
	buffers map[int]bufferBinding
	texs    map[int]*texture
	smps    map[int]*sampler
}

// bufferBinding is a buffer of a group with the offset of its binding.
type bufferBinding struct {
	b      *buffer
	offset int
}

var _ explicit.BindGroup = (*bindGroup)(nil)

// Destroy implements explicit.BindGroup: a group holds no object of Metal.
func (g *bindGroup) Destroy() {}

// NewBindGroup implements explicit.Device. It fails with MetalBindGroupInvalid for a group or a binding that the
// pipeline lacks, a uniform binding without its buffer, a uniform offset that is outside the buffer or not a
// multiple of the offset alignment, and a texture that is not sampled.
func (d *Device) NewBindGroup(p explicit.Pipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	pl, ok := p.(*pipeline)
	if !ok || pl.d != d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "pipeline"))
	}
	return d.makeBindGroup(pl.table, group, entries)
}

// NewComputeBindGroup implements explicit.Device.
func (d *Device) NewComputeBindGroup(p explicit.ComputePipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	pl, ok := p.(*computePipeline)
	if !ok || pl.d != d {
		panic(errors.New(TagMetalForeignResource, "the resource was not made by this Metal device", "resource", "compute pipeline"))
	}
	return d.makeBindGroup(pl.table, group, entries)
}

// makeBindGroup makes the group of a pipeline from the table of its bindings.
func (d *Device) makeBindGroup(table *bindingTable, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	gt := table.group(group)
	if gt == nil {
		return nil, errors.New(TagMetalBindGroupInvalid, "the pipeline has no such bind group", "group", group)
	}
	g := &bindGroup{d: d, group: group, buffers: map[int]bufferBinding{}, texs: map[int]*texture{}, smps: map[int]*sampler{}}
	for _, e := range entries {
		if !gt.has(e.Binding) {
			return nil, errors.New(TagMetalBindGroupInvalid, "the shader has no such binding", "group", group, "binding", e.Binding)
		}
		var decl explicit.ShaderBinding
		for _, s := range gt.slots {
			if s.binding == e.Binding {
				decl = s.decl
			}
		}
		if e.Buffer != nil || e.Texture != nil || e.Sampler != nil {
			d.limits.CheckBinding(decl, e)
		}
		switch classOf(decl.Kind) {
		case classBuffer:
			if e.Buffer == nil {
				return nil, errors.New(TagMetalBindGroupInvalid, "a buffer binding has no buffer", "group", group, "binding", e.Binding)
			}
			b := d.ownBuffer(e.Buffer)
			if e.Offset < 0 || e.Offset > b.size {
				return nil, errors.New(TagMetalBindGroupInvalid, "the offset of a buffer binding is outside the buffer",
					"group", group, "binding", e.Binding, "offset", e.Offset, "size", b.size)
			}
			if decl.Kind == explicit.BindingUniformBuffer && e.Offset%d.limits.UniformOffsetAlignment != 0 {
				return nil, errors.New(TagMetalBindGroupInvalid, "the offset of a uniform binding must be a multiple of the alignment",
					"group", group, "binding", e.Binding, "offset", e.Offset, "alignment", d.limits.UniformOffsetAlignment)
			}
			g.buffers[e.Binding] = bufferBinding{b, e.Offset}
		case classTexture:
			if e.Texture == nil {
				return nil, errors.New(TagMetalBindGroupInvalid, "a texture binding has no texture", "group", group, "binding", e.Binding)
			}
			t := d.ownTexture(e.Texture)
			if decl.Kind == explicit.BindingSampledTexture && t.desc.Usage&explicit.TextureSampled == 0 {
				return nil, errors.New(TagMetalBindGroupInvalid, "a sampled binding needs a texture with TextureSampled", "group", group, "binding", e.Binding)
			}
			g.texs[e.Binding] = t
		default:
			if e.Sampler == nil {
				return nil, errors.New(TagMetalBindGroupInvalid, "a sampler binding has no sampler", "group", group, "binding", e.Binding)
			}
			g.smps[e.Binding] = d.ownSampler(e.Sampler)
		}
	}
	for _, s := range gt.slots {
		if s.decl.Kind == explicit.BindingUniformBuffer {
			if _, ok := g.buffers[s.binding]; !ok {
				return nil, errors.New(TagMetalBindGroupInvalid, "a uniform binding has no buffer", "group", group, "binding", s.binding)
			}
		}
	}
	return g, nil
}

// binder sets a resource in the argument table of one stage.
type binder interface {
	setBuffer(stage stages, buf objc.ID, offset int, index int)
	setTexture(stage stages, tex objc.ID, index int)
	setSampler(stage stages, smp objc.ID, index int)
}

// apply sets the resources of the group the pipeline's table names, with the dynamic offsets of the uniform
// buffers (one for each, in binding order). A texture the group leaves out stays unbound; a sampler it leaves
// out is the default one, which Metal's validation wants set.
func (g *bindGroup) apply(b binder, gt *groupTable, dynamicOffsets []uint32) {
	next := 0
	for _, s := range gt.slots {
		switch s.class {
		case classBuffer:
			bb, ok := g.buffers[s.binding]
			if !ok {
				continue
			}
			offset := bb.offset
			if s.decl.Kind == explicit.BindingUniformBuffer {
				offset += int(dynamicOffsets[next])
				next++
			}
			b.setBuffer(s.stages, bb.b.obj, offset, s.index)
		case classTexture:
			if t, ok := g.texs[s.binding]; ok {
				b.setTexture(s.stages, t.obj, s.index)
			}
		default:
			smp := g.d.defaultSampler()
			if sm, ok := g.smps[s.binding]; ok {
				smp = sm.obj
			}
			b.setSampler(s.stages, smp, s.index)
		}
	}
}

// defaultSampler is the sampler for a binding that a group does not give one: clamp, nearest, level 0.
func (d *Device) defaultSampler() objc.ID {
	if d.defSampler == 0 {
		s, err := d.NewSampler(explicit.SamplerDesc{})
		if err != nil {
			panic(err)
		}
		d.defSampler = s.(*sampler).obj
	}
	return d.defSampler
}
