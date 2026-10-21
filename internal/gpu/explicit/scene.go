package explicit

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// pipeKey is a pipeline of a primitive with a fixed blend.
type pipeKey struct {
	kind  pipeKind
	blend gpu.FixedBlend
}

// pipeSpec is what makes the pipelines of a kind of primitive different: the shader, how its
// vertex buffer is read, and how much it draws for each item.
type pipeSpec struct {
	shader      *Shader
	stride      int
	perInstance bool
	attrs       []VertexAttribute
	// vertices is how many vertices an instance has; 0 means that an item is one vertex.
	vertices int
	// drawSize is the size of the Draw uniform in bytes.
	drawSize int
}

func vec4Attrs(n int) []VertexAttribute {
	a := make([]VertexAttribute, n)
	for i := range a {
		a[i] = VertexAttribute{Location: i, Format: VertexFloat32x4, Offset: 16 * i}
	}
	return a
}

var pipeSpecs = [numPipes]pipeSpec{
	pipeShape: {shader: ShaderShape, stride: gpu.ShapeStride, perInstance: true, attrs: vec4Attrs(gpu.ShapeVec4s), vertices: 6, drawSize: 16},
	pipeMask:  {shader: ShaderMask, stride: 64, perInstance: true, attrs: vec4Attrs(4), vertices: 6, drawSize: 16},
	pipeImage: {shader: ShaderImage, stride: 4 * gpu.ImageFloats, perInstance: true, attrs: vec4Attrs(8), vertices: 6, drawSize: 16},
	pipeVertices: {shader: ShaderVertices, stride: 4 * gpu.VertexFloats, attrs: []VertexAttribute{
		{Location: 0, Format: VertexFloat32x2, Offset: 0}, {Location: 1, Format: VertexFloat32x4, Offset: 8},
	}, drawSize: 64},
}

// pipeline returns the pipeline of the primitive with the blend, made once.
func (r *Renderer) pipeline(k pipeKind, b gpu.FixedBlend) (Pipeline, error) {
	key := pipeKey{k, b}
	if p, ok := r.pipes[key]; ok {
		return p, nil
	}
	sp := &pipeSpecs[k]
	p, err := r.dev.NewPipeline(PipelineDesc{
		Shader: sp.shader, VertexEntry: "vs_main", FragmentEntry: "fs_main",
		VertexBuffers: []VertexBufferLayout{{Stride: sp.stride, PerInstance: sp.perInstance, Attributes: sp.attrs}},
		Blend:         blendState(b), Target: TextureFormatRGBA8Unorm,
	})
	if err != nil {
		return nil, err
	}
	r.pipes[key] = p
	return p, nil
}

// groupKey names a bind group of the frame: the group of a pipeline kind that holds a texture.
type groupKey struct {
	kind  pipeKind
	group int
	tex   Texture
}

// frameGroups are the bind groups made for one frame, which the device keeps until the frame is
// done: a bind group is made once for each use.
type frameGroups struct {
	r        *Renderer
	slot     *frameSlot
	byKey    map[groupKey]BindGroup
	uniforms [numPipes]BindGroup // group 0 of each kind
}

// uniformGroup returns group 0 of the kind: the frame and draw uniforms, with dynamic offsets.
func (g *frameGroups) uniformGroup(k pipeKind, p Pipeline) (BindGroup, error) {
	if bg := g.uniforms[k]; bg != nil {
		return bg, nil
	}
	bg, err := g.r.dev.NewBindGroup(p, 0, []Binding{
		{Binding: 0, Buffer: g.slot.uniforms, Size: frameUniformSize},
		{Binding: 1, Buffer: g.slot.uniforms, Size: pipeSpecs[k].drawSize},
	})
	if err != nil {
		return nil, err
	}
	g.uniforms[k] = bg
	return bg, nil
}

// textureGroup returns the group of the kind that holds the texture, and for images its sampler.
func (g *frameGroups) textureGroup(k pipeKind, group int, p Pipeline, tex Texture, sampler Sampler) (BindGroup, error) {
	key := groupKey{k, group, tex}
	if bg, ok := g.byKey[key]; ok {
		return bg, nil
	}
	entries := []Binding{{Binding: 0, Texture: tex}}
	if sampler != nil {
		entries = append(entries, Binding{Binding: 1, Sampler: sampler})
	}
	bg, err := g.r.dev.NewBindGroup(p, group, entries)
	if err != nil {
		return nil, err
	}
	g.byKey[key] = bg
	return bg, nil
}

// drawUniform encodes the Draw uniform of a call.
func drawUniform(c *drawCall) []byte {
	var f []float32
	if c.clip != nil {
		f = append(f, float32(c.clip.x), float32(c.clip.y), float32(c.clip.w), float32(c.clip.h))
	} else {
		f = append(f, 0, 0, 0, 0)
	}
	if c.pipe == pipeVertices {
		colors := float32(0)
		if c.vert.colors {
			colors = 1
		}
		f = append(f, c.vert.paint[:]...)
		f = append(f, c.vert.clip[:]...)
		f = append(f, float32(c.vert.mode), colors, 0, 0)
	}
	return floatBytes(f)
}

// drawScene makes the mip levels that are due, clears the frame texture to the background and
// draws the frame's primitives into it, in one pass.
func (r *Renderer) drawScene(enc Encoder, slot *frameSlot, target Texture, fb *frameBuilder, width, height int, bg ui.Color) error {
	if err := r.generateMips(enc); err != nil {
		return err
	}
	if len(fb.calls) == 0 {
		enc.BeginPass(target, LoadClear, bg).End()
		return nil
	}
	pipes := make([]Pipeline, len(fb.calls))
	for i, c := range fb.calls {
		p, err := r.pipeline(c.pipe, c.blend)
		if err != nil {
			return err
		}
		pipes[i] = p
	}
	if r.imageSampler == nil {
		s, err := r.dev.NewSampler(SamplerDesc{MinFilter: FilterLinear, MagFilter: FilterLinear, MipmapFilter: FilterNearest})
		if err != nil {
			return err
		}
		r.imageSampler = s
	}
	white, err := r.res.whiteTexture()
	if err != nil {
		return err
	}
	align := r.dev.Limits().UniformOffsetAlignment
	slot.ring = slot.ring[:0]
	frameOff := slot.push(align, frameUniform(width, height, 0, 0))
	drawOffs := make([]uint32, len(fb.calls))
	for i := range fb.calls {
		drawOffs[i] = slot.push(align, drawUniform(&fb.calls[i]))
	}
	base, err := r.upload(slot, &fb.data)
	if err != nil {
		return err
	}
	groups := &frameGroups{r: r, slot: slot, byKey: map[groupKey]BindGroup{}}
	pass := enc.BeginPass(target, LoadClear, bg)
	var current Pipeline
	var group1, group2 BindGroup
	for i := range fb.calls {
		c := &fb.calls[i]
		sp := &pipeSpecs[c.pipe]
		if pipes[i] != current {
			current = pipes[i]
			group1, group2 = nil, nil
			pass.SetPipeline(current)
		}
		g0, err := groups.uniformGroup(c.pipe, current)
		if err != nil {
			pass.End()
			return err
		}
		pass.SetBindGroup(0, g0, frameOff, drawOffs[i])
		mask := white
		if c.clip != nil {
			mask = c.clip.tex
		}
		g1, err := groups.textureGroup(c.pipe, 1, current, mask, nil)
		if err != nil {
			pass.End()
			return err
		}
		if g1 != group1 {
			group1 = g1
			pass.SetBindGroup(1, g1)
		}
		switch c.pipe {
		case pipeMask, pipeImage:
			tex, sampler := Texture(nil), Sampler(nil)
			if c.pipe == pipeMask {
				tex = r.res.pages[c.page]
			} else {
				tex, sampler = c.img.tex, r.imageSampler
			}
			g2, err := groups.textureGroup(c.pipe, 2, current, tex, sampler)
			if err != nil {
				pass.End()
				return err
			}
			if g2 != group2 {
				group2 = g2
				pass.SetBindGroup(2, g2)
			}
		}
		pass.SetScissor(c.scissor)
		pass.SetVertexBuffer(0, slot.instances, base[c.pipe]+c.first*sp.stride)
		if sp.vertices > 0 {
			pass.Draw(sp.vertices, c.count, 0, 0)
		} else {
			pass.Draw(c.count, 1, 0, 0)
		}
	}
	pass.End()
	return nil
}

// generateMips makes the levels below the first of the images that were uploaded since the last
// frame: each level is drawn from the one above it with a linear sampler, which averages four
// texels (SPEC §9.3 "Frames": neither API has glGenerateMipmap).
func (r *Renderer) generateMips(enc Encoder) error {
	mips := r.res.mips
	r.res.mips = nil
	if len(mips) == 0 {
		return nil
	}
	if r.mipSampler == nil {
		s, err := r.dev.NewSampler(SamplerDesc{MinFilter: FilterLinear, MagFilter: FilterLinear})
		if err != nil {
			return err
		}
		r.mipSampler = s
	}
	pipe, err := r.blitPipeline(TextureFormatRGBA8Unorm)
	if err != nil {
		return err
	}
	for _, e := range mips {
		for level := 1; level < e.levels; level++ {
			group, err := r.dev.NewBindGroup(pipe, 0, []Binding{{Binding: 0, Texture: e.tex.Level(level - 1)}, {Binding: 1, Sampler: r.mipSampler}})
			if err != nil {
				return err
			}
			pass := enc.BeginPass(e.tex.Level(level), LoadClear, ui.Color{})
			pass.SetPipeline(pipe)
			pass.SetBindGroup(0, group)
			pass.SetScissor(image.Rect(0, 0, max(e.w>>level, 1), max(e.h>>level, 1)))
			pass.Draw(3, 1, 0, 0)
			pass.End()
		}
	}
	return nil
}
