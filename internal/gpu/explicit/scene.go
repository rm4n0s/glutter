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
		{Location: 0, Format: VertexFloat32x2, Offset: 0}, {Location: 1, Format: VertexFloat32x4, Offset: 8}, {Location: 2, Format: VertexFloat32x2, Offset: 24},
	}, drawSize: 64},
	pipeLayer: {shader: ShaderLayer, stride: 48, perInstance: true, attrs: vec4Attrs(3), vertices: 6, drawSize: 48},
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

// declares says whether a binding of a group is read by a stage of the pipeline's shader, which
// decides what its bind group may hold.
func declares(p Pipeline, group, binding int) bool {
	for _, e := range p.Desc().Shader.Entries {
		for _, b := range e.Bindings {
			if b.Group == group && b.Binding == binding {
				return true
			}
		}
	}
	return false
}

// groupKey names a bind group of the frame: the group of a pipeline kind that holds textures.
type groupKey struct {
	kind  pipeKind
	group int
	a, b  Texture
	flag  bool
}

// frameGroups are the bind groups made for one frame, which the device keeps until the frame is
// done: a bind group is made once for each use.
type frameGroups struct {
	r        *Renderer
	slot     *frameSlot
	byKey    map[groupKey]BindGroup
	uniforms [numPipes]BindGroup // group 0 of each kind
	filter   BindGroup           // group 0 of the filter pipeline
}

// uniformGroup returns group 0 of the kind: the frame, draw and paint uniforms, with dynamic
// offsets.
func (g *frameGroups) uniformGroup(k pipeKind, p Pipeline) (BindGroup, error) {
	if bg := g.uniforms[k]; bg != nil {
		return bg, nil
	}
	entries := []Binding{
		{Binding: 0, Buffer: g.slot.uniforms, Size: frameUniformSize},
		{Binding: 1, Buffer: g.slot.uniforms, Size: pipeSpecs[k].drawSize},
	}
	if declares(p, 0, 2) {
		entries = append(entries, Binding{Binding: 2, Buffer: g.slot.uniforms, Size: paintUniformSize})
	}
	bg, err := g.r.dev.NewBindGroup(p, 0, entries)
	if err != nil {
		return nil, err
	}
	g.uniforms[k] = bg
	return bg, nil
}

// textureGroup returns the group of the kind that holds the textures: a texture at binding 0 and,
// when sampler is not nil, the sampler at binding 1.
func (g *frameGroups) textureGroup(k pipeKind, group int, p Pipeline, tex Texture, sampler Sampler) (BindGroup, error) {
	key := groupKey{kind: k, group: group, a: tex, flag: sampler != nil}
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

// layerGroup returns group 2 of the layer pipeline: the layer's pixels at binding 0 and the layer
// before it was blurred (when its alpha is a blurred mask) at binding 1.
func (g *frameGroups) layerGroup(p Pipeline, tex, own Texture) (BindGroup, error) {
	key := groupKey{kind: pipeLayer, group: 2, a: tex, b: own}
	if bg, ok := g.byKey[key]; ok {
		return bg, nil
	}
	bg, err := g.r.dev.NewBindGroup(p, 2, []Binding{{Binding: 0, Texture: tex}, {Binding: 1, Texture: own}})
	if err != nil {
		return nil, err
	}
	g.byKey[key] = bg
	return bg, nil
}

// paintGroup returns group 3: the image of an image shader and the copy of the destination, the
// bindings that the pipeline reads (a stand-in texture where the call has none).
func (g *frameGroups) paintGroup(k pipeKind, p Pipeline, image, dst Texture) (BindGroup, error) {
	key := groupKey{kind: k, group: 3, a: image, b: dst}
	if bg, ok := g.byKey[key]; ok {
		return bg, nil
	}
	var entries []Binding
	if declares(p, 3, 0) {
		entries = append(entries, Binding{Binding: 0, Texture: image})
	}
	if declares(p, 3, 1) {
		entries = append(entries, Binding{Binding: 1, Texture: dst})
	}
	bg, err := g.r.dev.NewBindGroup(p, 3, entries)
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
	switch c.pipe {
	case pipeVertices:
		colors, atTexture := float32(0), float32(0)
		if c.vert.colors {
			colors = 1
		}
		if c.vert.atTexture {
			atTexture = 1
		}
		f = append(f, c.vert.paint[:]...)
		f = append(f, c.vert.clip[:]...)
		f = append(f, float32(c.vert.mode), colors, atTexture, 0)
	case pipeLayer:
		f = append(f, float32(c.rect.Min.X), float32(c.rect.Min.Y), float32(c.rect.Dx()), float32(c.rect.Dy()))
		mask, drawnOnly := float32(-1), float32(0)
		if c.eff != nil {
			mask = float32(c.eff.mask)
			if c.eff.drawnOnly {
				drawnOnly = 1
			}
		}
		f = append(f, mask, drawnOnly, 0, 0)
	}
	return floatBytes(f)
}

// paintUniform encodes the Paint uniform of a spec, with the pixel of the surface that the copy of
// the destination starts at (paint.wgsl.inc).
func paintUniform(sp *gpu.PaintSpec, dst image.Point) []byte {
	f := make([]float32, 0, paintUniformSize/4)
	f = append(f, float32(sp.Shader), float32(sp.Tile[0]), float32(sp.Tile[1]), float32(sp.RampN))
	f = append(f, sp.XF[0], sp.XF[1], sp.XF[2], sp.XF[3], sp.XF[4], sp.XF[5], 0, 0)
	f = append(f, sp.P0[:]...)
	f = append(f, sp.P1[:]...)
	f = append(f, sp.ImgInfo[:]...)
	f = append(f, sp.ConstCol[:]...)
	dither, invert := float32(0), float32(0)
	if sp.Dither {
		dither = 1
	}
	if sp.Invert {
		invert = 1
	}
	f = append(f, dither, float32(sp.Filter), float32(sp.FilterBlend), invert)
	f = append(f, float32(sp.ReadBlend-1), float32(dst.X), float32(dst.Y), 0)
	f = append(f, sp.FilterCol[:]...)
	f = append(f, sp.FilterMat[:]...)
	for i := 0; i < gpu.MaxStops; i++ {
		f = append(f, sp.RampCol[i][:]...)
	}
	f = append(f, sp.RampPos[:]...)
	return floatBytes(f)
}

// target is a texture that is being drawn into: the frame's, or a layer's, which may be bigger
// than the rectangle of the surface that it holds, from the top left.
type target struct {
	tex  Texture
	rect image.Rectangle // the pixels of the surface that the texture's top left corner starts
	size image.Point     // the size of the texture
}

// plan is what a call needs from the uniform ring: the offsets of its entries, and for a layer
// with filter passes the offsets of each pass's.
type plan struct {
	frame, draw, paint uint32
	passes             []passPlan
}

// passPlan is the offsets of a filter pass's parameters and paint.
type passPlan struct{ params, paint uint32 }

// passParamsSize is the size of the Pass uniform of shaders/filter.wgsl: six float4s.
const passParamsSize = 6 * 16

// passUniform encodes the Pass uniform of a filter pass over a source of srcW x srcH pixels whose
// top left corner is the pixel origin of the surface.
func passUniform(ps *gpu.FilterPass, srcW, srcH int, origin image.Point) []byte {
	dilate := float32(0)
	if ps.Dilate {
		dilate = 1
	}
	return floatBytes([]float32{
		float32(ps.Kind), float32(ps.Axis), float32(ps.Tile), dilate,
		float32(srcW), float32(srcH), float32(ps.OutW), float32(ps.OutH),
		ps.Sigma, float32(ps.Radius), ps.Norm, 0,
		float32(ps.Factor[0]), float32(ps.Factor[1]), float32(ps.Quality), 0,
		ps.Inv[0], ps.Inv[1], ps.Inv[2], ps.Inv[3],
		ps.Inv[4], ps.Inv[5], float32(origin.X), float32(origin.Y),
	})
}

// planPasses puts the uniforms of the passes over a source of w x h pixels into the ring; empty is
// the offset of a paint with no filter, made when first needed.
func planPasses(push func([]byte) uint32, empty *int64, passes []gpu.FilterPass, w, h int, origin image.Point) []passPlan {
	out := make([]passPlan, len(passes))
	for i := range passes {
		ps := &passes[i]
		out[i].params = push(passUniform(ps, w, h, origin))
		if ps.Kind == gpu.PassColor {
			out[i].paint = push(paintUniform(ps.Spec, image.Point{}))
		} else {
			if *empty < 0 {
				*empty = int64(push(paintUniform(&gpu.PaintSpec{}, image.Point{})))
			}
			out[i].paint = uint32(*empty)
		}
		w, h = ps.OutW, ps.OutH
	}
	return out
}

// granule rounds up the size of the textures of layers and of the copy of the destination, so that
// the pool does not fill with sizes.
const granule = 64

func roundUp(v, limit int) int { return min((v+granule-1)/granule*granule, limit) }

// kindFilter names the bind groups of the filter pipeline among those of the pipeline kinds.
const kindFilter = numPipes

// filterPipeline returns the pipeline of the filter passes, made once.
func (r *Renderer) filterPipeline() (Pipeline, error) {
	if r.filter != nil {
		return r.filter, nil
	}
	p, err := r.dev.NewPipeline(PipelineDesc{Shader: ShaderFilter, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: TextureFormatRGBA8Unorm})
	if err != nil {
		return nil, err
	}
	r.filter = p
	return p, nil
}

// filterGroup returns group 0 of the filter pipeline: the pass and paint uniforms, with dynamic
// offsets.
func (g *frameGroups) filterGroup(p Pipeline) (BindGroup, error) {
	if g.filter != nil {
		return g.filter, nil
	}
	bg, err := g.r.dev.NewBindGroup(p, 0, []Binding{
		{Binding: 1, Buffer: g.slot.uniforms, Size: passParamsSize},
		{Binding: 2, Buffer: g.slot.uniforms, Size: paintUniformSize},
	})
	if err != nil {
		return nil, err
	}
	g.filter = bg
	return bg, nil
}

// runPasses draws the passes over the pixels of src, which are srcW x srcH, one pass over the
// result of the one before. The last pass draws into dst when it is not nil; else the result is a
// pooled texture that the caller releases. Nothing but the passes is drawn.
func (r *Renderer) runPasses(enc Encoder, groups *frameGroups, pipe Pipeline, plans []passPlan, passes []gpu.FilterPass, src Texture, dst Texture) (Texture, error) {
	limit := r.dev.Limits().MaxTextureSize
	var temps []Texture
	release := func() {
		for _, t := range temps {
			r.releaseLayer(t)
		}
	}
	cur, result := src, src
	for i := range passes {
		ps := &passes[i]
		out := dst
		if i < len(passes)-1 || dst == nil {
			t, err := r.layerTexture(roundUp(ps.OutW, limit), roundUp(ps.OutH, limit))
			if err != nil {
				release()
				return nil, err
			}
			out = t
		}
		g0, err := groups.filterGroup(pipe)
		if err != nil {
			release()
			return nil, err
		}
		g1, err := groups.textureGroup(kindFilter, 1, pipe, cur, nil)
		if err != nil {
			release()
			return nil, err
		}
		pass := enc.BeginPass(out, LoadClear, ui.Color{})
		pass.SetPipeline(pipe)
		pass.SetBindGroup(0, g0, plans[i].params, plans[i].paint)
		pass.SetBindGroup(1, g1)
		pass.SetScissor(image.Rect(0, 0, ps.OutW, ps.OutH))
		pass.Draw(3, 1, 0, 0)
		pass.End()
		if out != dst {
			temps = append(temps, out)
		}
		cur, result = out, out
	}
	// The intermediate textures go back to the pool; the result, when it is one, stays out.
	for _, t := range temps {
		if t != result {
			r.releaseLayer(t)
		}
	}
	return result, nil
}

// drawScene makes the mip levels that are due, clears the frame texture to the background and draws
// the frame's primitives into it, switching to the texture of a layer where a layer begins and back
// to its parent where it is composited.
func (r *Renderer) drawScene(enc Encoder, slot *frameSlot, frameTex Texture, fb *frameBuilder, width, height int, bg ui.Color) error {
	if err := r.generateMips(enc); err != nil {
		return err
	}
	if len(fb.calls) == 0 {
		enc.BeginPass(frameTex, LoadClear, bg).End()
		return nil
	}
	pipes := make([]Pipeline, len(fb.calls))
	filtering := false
	for i, c := range fb.calls {
		if c.eff != nil && (len(c.eff.passes) > 0 || c.eff.backdrop) {
			filtering = true
		}
		if c.op == opBeginLayer || (c.op == opComposite && c.count == 0) {
			continue
		}
		p, err := r.pipeline(c.pipe, c.blend)
		if err != nil {
			return err
		}
		pipes[i] = p
	}
	var filterPipe Pipeline
	if filtering {
		p, err := r.filterPipeline()
		if err != nil {
			return err
		}
		filterPipe = p
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
	clear, err := r.res.clearTexture()
	if err != nil {
		return err
	}

	// The ring: the frame uniform of each target, and the draw and paint uniforms of each call.
	limit := r.dev.Limits().MaxTextureSize
	align := r.dev.Limits().UniformOffsetAlignment
	slot.ring = slot.ring[:0]
	push := func(b []byte) uint32 { return slot.push(align, b) }
	root := target{tex: frameTex, rect: image.Rect(0, 0, width, height), size: image.Pt(width, height)}
	frameOff := push(frameUniform(width, height, 0, 0))
	var stackOff []uint32
	plans := make([]plan, len(fb.calls))
	specOff := map[*gpu.PaintSpec]uint32{}
	emptyPaint := int64(-1)
	dstW, dstH := 0, 0
	for i := range fb.calls {
		c := &fb.calls[i]
		switch c.op {
		case opBeginLayer:
			stackOff = append(stackOff, frameOff)
			frameOff = push(frameUniform(roundUp(c.rect.Dx(), limit), roundUp(c.rect.Dy(), limit), c.rect.Min.X, c.rect.Min.Y))
		case opComposite:
			frameOff = stackOff[len(stackOff)-1]
			stackOff = stackOff[:len(stackOff)-1]
		}
		plans[i].frame = frameOff
		if c.eff != nil && len(c.eff.passes) > 0 && (c.op == opBeginLayer || c.count != 0) {
			src := c.rect
			if c.op == opBeginLayer && c.eff.backdrop && !c.eff.src.Empty() {
				// A backdrop filter that moves pixels reads them from where it moves them from.
				src = c.eff.src
			}
			plans[i].passes = planPasses(push, &emptyPaint, c.eff.passes, src.Dx(), src.Dy(), src.Min)
		}
		if c.op == opBeginLayer || c.count == 0 {
			continue
		}
		plans[i].draw = push(drawUniform(c))
		if c.spec.ReadBlend == 0 {
			off, ok := specOff[c.spec]
			if !ok {
				off = push(paintUniform(c.spec, image.Point{}))
				specOff[c.spec] = off
			}
			plans[i].paint = off
		} else {
			plans[i].paint = push(paintUniform(c.spec, c.scissor.Min))
			dstW, dstH = max(dstW, c.scissor.Dx()), max(dstH, c.scissor.Dy())
		}
	}
	base, err := r.upload(slot, &fb.data)
	if err != nil {
		return err
	}
	var dstTex Texture
	if dstW > 0 {
		if dstTex, err = r.dstTexture(roundUp(dstW, limit), roundUp(dstH, limit)); err != nil {
			return err
		}
	}

	groups := &frameGroups{r: r, slot: slot, byKey: map[groupKey]BindGroup{}}
	cur := root
	var stack []target
	var pass Pass
	var current Pipeline
	var group1, group2, group3 BindGroup
	begin := func(load LoadOp, clear ui.Color) {
		pass = enc.BeginPass(cur.tex, load, clear)
		current, group1, group2, group3 = nil, nil, nil, nil
	}
	end := func() {
		if pass != nil {
			pass.End()
			pass = nil
		}
	}
	fail := func(err error) error {
		end()
		for _, t := range stack {
			r.releaseLayer(t.tex)
		}
		if cur.tex != frameTex {
			r.releaseLayer(cur.tex)
		}
		return err
	}
	begin(LoadClear, bg)
	for i := range fb.calls {
		c := &fb.calls[i]
		sp := &pipeSpecs[c.pipe]
		var layer Texture    // the texture a composite draws
		var filtered Texture // the layer after its filter passes, which is drawn instead
		kept := false        // the raster cache keeps the layer
		switch c.op {
		case opBeginLayer:
			end()
			w, h := roundUp(c.rect.Dx(), limit), roundUp(c.rect.Dy(), limit)
			tex, err := r.layerTexture(w, h)
			if err != nil {
				return fail(err)
			}
			load := LoadClear
			if c.eff != nil && c.eff.backdrop {
				// The layer starts as the filtered copy of the pixels under it.
				// It is a copy of the pixels under the layer, unless the filter reads them from
				// elsewhere.
				src := c.rect
				if !c.eff.src.Empty() {
					src = c.eff.src
				}
				bd, err := r.layerTexture(roundUp(src.Dx(), limit), roundUp(src.Dy(), limit))
				if err != nil {
					r.releaseLayer(tex)
					return fail(err)
				}
				enc.CopyTexture(cur.tex, src.Sub(cur.rect.Min), bd, image.Point{})
				_, err = r.runPasses(enc, groups, filterPipe, plans[i].passes, c.eff.passes, bd, tex)
				r.releaseLayer(bd)
				if err != nil {
					r.releaseLayer(tex)
					return fail(err)
				}
				load = LoadLoad
			}
			stack = append(stack, cur)
			cur = target{tex: tex, rect: c.rect, size: image.Pt(w, h)}
			begin(load, ui.Color{})
			continue
		case opDrawCached:
			end()
			layer = c.cache.Tex
		case opComposite:
			end()
			layer = cur.tex
			cur = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if c.count == 0 {
				r.releaseLayer(layer)
				continue
			}
			kept = c.keep != nil && r.cache != nil && r.cache.Put(c.keep.layer, c.keep.key, layer, r.releaseLayer)
			if c.eff != nil && len(c.eff.passes) > 0 {
				f, err := r.runPasses(enc, groups, filterPipe, plans[i].passes, c.eff.passes, layer, nil)
				if err != nil {
					if !kept {
						r.releaseLayer(layer)
					}
					return fail(err)
				}
				filtered = f
			}
		}
		if c.spec.ReadBlend != 0 {
			// The pixels under the draw are copied to a texture the shader reads: a copy is made
			// outside a pass.
			end()
			enc.CopyTexture(cur.tex, c.scissor.Sub(cur.rect.Min), dstTex, image.Point{})
		}
		if pass == nil {
			begin(LoadLoad, ui.Color{})
		}
		if pipes[i] != current {
			current = pipes[i]
			group1, group2, group3 = nil, nil, nil
			pass.SetPipeline(current)
		}
		g0, err := groups.uniformGroup(c.pipe, current)
		if err != nil {
			return fail(err)
		}
		pass.SetBindGroup(0, g0, plans[i].frame, plans[i].draw, plans[i].paint)
		mask := white
		if c.clip != nil {
			mask = c.clip.tex
		}
		g1, err := groups.textureGroup(c.pipe, 1, current, mask, nil)
		if err != nil {
			return fail(err)
		}
		if g1 != group1 {
			group1 = g1
			pass.SetBindGroup(1, g1)
		}
		var g2 BindGroup
		switch c.pipe {
		case pipeLayer:
			src, ownTex := layer, clear
			if filtered != nil {
				src = filtered
			}
			if c.eff != nil && c.eff.mask > 0 {
				ownTex = layer
			}
			g2, err = groups.layerGroup(current, src, ownTex)
		default:
			if tex, sampler := primaryTexture(r, c); tex != nil {
				g2, err = groups.textureGroup(c.pipe, 2, current, tex, sampler)
			}
		}
		if err != nil {
			return fail(err)
		}
		if g2 != nil && g2 != group2 {
			group2 = g2
			pass.SetBindGroup(2, g2)
		}
		paintImage, dst := clear, clear
		if t, ok := c.spec.Image.(Texture); ok {
			paintImage = t
		}
		if c.spec.ReadBlend != 0 {
			dst = dstTex
		}
		g3, err := groups.paintGroup(c.pipe, current, paintImage, dst)
		if err != nil {
			return fail(err)
		}
		if g3 != group3 {
			group3 = g3
			pass.SetBindGroup(3, g3)
		}
		pass.SetScissor(c.scissor.Sub(cur.rect.Min))
		pass.SetVertexBuffer(0, slot.instances, base[c.pipe]+c.first*sp.stride)
		if sp.vertices > 0 {
			pass.Draw(sp.vertices, c.count, 0, 0)
		} else {
			pass.Draw(c.count, 1, 0, 0)
		}
		if c.op == opComposite {
			if !kept {
				r.releaseLayer(layer)
			}
			if filtered != nil {
				r.releaseLayer(filtered)
			}
		}
	}
	end()
	return nil
}

// primaryTexture is the texture at group 2 of a draw call and its sampler: an atlas page for a
// coverage tile, an image, or the layer that a composite draws; none for the other pipelines.
func primaryTexture(r *Renderer, c *drawCall) (Texture, Sampler) {
	switch c.pipe {
	case pipeMask:
		return r.res.pages[c.page], nil
	case pipeImage:
		return c.img.tex, r.imageSampler
	}
	return nil, nil
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
