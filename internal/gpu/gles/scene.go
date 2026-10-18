package gles

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// drawScene draws the scene into the framebuffer that is bound, which is w x h
// pixels, over what is there. It flattens the scene, turns the commands into the
// frame's instances and vertices (failing before anything is drawn if one is
// unsupported, unless the renderer skips them), and draws the calls in order.
func (r *Renderer) drawScene(scene *ui.Scene, w, h int, useCache bool) error {
	r.last = FrameStats{}
	if scene == nil {
		return nil
	}
	if r.res == nil {
		res, err := newResources(r.ctx.dev, r.ctx.gl)
		if err != nil {
			return err
		}
		r.res = res
	}
	r.res.beginFrame()
	defer r.res.endFrame()
	tiles0, clips0, images0 := r.res.tilesRasterized, r.res.clipMasksBuilt, r.res.imagesUploaded
	fo := gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, float64(w), float64(h))}
	var cache *layerCache
	if useCache && r.cache != nil {
		cache = r.cache
		cache.beginFrame(scene.Root())
		fo.Boundary = cache.boundary
	}
	list, err := gpu.Flatten(scene, fo)
	if err != nil {
		return err
	}
	b := &frameBuilder{w: w, h: h, skip: r.skip, res: r.res, cache: cache}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			return err
		}
	}
	b.stats.Instances = b.frame.items(pipeShape) + b.frame.items(pipeMask) + b.frame.items(pipeImage) + b.frame.items(pipeLayer)
	b.stats.Vertices = b.frame.items(pipeVertices)
	for _, c := range b.frame.calls {
		if c.op != opBeginLayer {
			b.stats.Batches++
		}
	}
	b.stats.TilesRasterized = r.res.tilesRasterized - tiles0
	b.stats.ClipMasksBuilt = r.res.clipMasksBuilt - clips0
	b.stats.ImagesUploaded = r.res.imagesUploaded - images0
	r.last = b.stats
	if err := r.drawFrame(&b.frame, w, h); err != nil {
		return err
	}
	if cache != nil {
		cache.endFrame(r.ctx.dev)
	}
	return nil
}

// vec4s is the number of vec4 attributes of each instanced pipeline.
var vec4s = [numPipelines]int{pipeShape: 12, pipeMask: 4, pipeImage: 8, pipeLayer: 3}

// setupVAO describes the attributes of a pipeline in its vertex array.
func (r *Renderer) setupVAO(p pipeline) uint32 {
	gl := r.ctx.gl
	return r.ctx.dev.VertexArray(programNames[p], func(uint32) {
		if p == pipeVertices {
			gl.enableVertexAttribArray(0)
			gl.enableVertexAttribArray(1)
			gl.enableVertexAttribArray(2)
			return
		}
		for i := uint32(0); i < uint32(vec4s[p]); i++ {
			gl.enableVertexAttribArray(i)
			gl.vertexAttribDivisor(i, 1)
		}
	})
}

// renderTarget is what is drawn into: the framebuffer that was bound when the
// frame started, or a layer's.
type renderTarget struct {
	fb   *Framebuffer // nil for the framebuffer that was bound at the start
	rect image.Rectangle
}

// dstGranule rounds up the size of the textures that hold a copy of the pixels
// under a draw, so that the pool does not fill with sizes.
const dstGranule = 64

// drawer is the state of drawing a frame's calls.
type drawer struct {
	r    *Renderer
	f    *frame
	bufs [numPipelines]*Buffer
	base uint32 // the framebuffer bound when the frame started
	cur  renderTarget
	open []renderTarget // the layers being drawn, outermost first, under cur

	// gen changes with the target, whose size and origin the programs hold.
	gen      int
	progGen  [numPipelines]int
	progInit [numPipelines]bool
	lastSpec [numPipelines]*paintSpec
	current  pipeline
}

// drawFrame uploads the data of each pipeline once and issues the draw calls.
func (r *Renderer) drawFrame(f *frame, w, h int) error {
	if len(f.calls) == 0 {
		return nil
	}
	d := r.ctx.dev
	dr := &drawer{r: r, f: f, base: d.state.framebuffer, current: -1}
	dr.cur = renderTarget{rect: image.Rect(0, 0, w, h)}
	dr.gen = 1
	defer func() {
		for _, t := range append(dr.open, dr.cur) {
			if t.fb != nil {
				d.ReleaseFramebuffer(t.fb)
			}
		}
		d.bindFramebufferID(dr.base)
		d.SetScissor(false, 0, 0, 0, 0)
	}()
	for p := pipeline(0); p < numPipelines; p++ {
		if len(f.data[p]) == 0 {
			continue
		}
		buf, err := d.AcquireBuffer(glArrayBuffer, len(f.data[p])*4)
		if err != nil {
			return err
		}
		defer d.ReleaseBuffer(buf)
		if err := d.UploadBuffer(buf, floatBytes(f.data[p])); err != nil {
			return err
		}
		dr.bufs[p] = buf
	}
	d.Viewport(0, 0, w, h)
	d.SetBlend(true)
	d.BlendEquation(glFuncAdd)
	d.ColorMask(true, true, true, true)
	for i := range f.calls {
		if err := dr.call(&f.calls[i]); err != nil {
			return err
		}
	}
	return d.checkError("draw frame")
}

// enter makes the target the one drawn into.
func (dr *drawer) enter(t renderTarget) {
	d := dr.r.ctx.dev
	dr.cur = t
	if t.fb != nil {
		d.BindFramebuffer(t.fb)
	} else {
		d.bindFramebufferID(dr.base)
	}
	d.Viewport(0, 0, t.rect.Dx(), t.rect.Dy())
	dr.gen++
}

// call runs one draw call.
func (dr *drawer) call(c *drawCall) error {
	d := dr.r.ctx.dev
	gl := dr.r.ctx.gl
	var layer renderTarget
	var filtered *Framebuffer
	switch c.op {
	case opBeginLayer:
		return dr.beginLayer(c)
	case opDrawCached:
		layer = renderTarget{fb: c.cache.fb, rect: c.cache.rect}
	case opComposite:
		layer = dr.cur
		if c.count == 0 || c.keep == nil || !dr.r.cache.put(d, c.keep.layer, c.keep.key, layer.fb) {
			defer d.ReleaseFramebuffer(layer.fb)
		}
		if c.count != 0 && c.eff != nil && len(c.eff.passes) > 0 {
			var err error
			filtered, err = dr.runPasses(layer.fb.Texture(), layer.rect.Dx(), layer.rect.Dy(), [2]int{layer.rect.Min.X, layer.rect.Min.Y}, c.eff.passes, nil)
			if err != nil {
				return err
			}
			defer d.ReleaseFramebuffer(filtered)
		}
		dr.enter(dr.open[len(dr.open)-1])
		dr.open = dr.open[:len(dr.open)-1]
		if c.count == 0 {
			return nil
		}
	}

	if c.pipe != dr.current {
		prog, err := d.Program(programNames[c.pipe])
		if err != nil {
			return err
		}
		d.BindVertexArray(dr.r.setupVAO(c.pipe))
		d.BindArrayBuffer(dr.bufs[c.pipe])
		d.UseProgram(prog)
		dr.current = c.pipe
	}
	prog, err := d.Program(programNames[c.pipe])
	if err != nil {
		return err
	}
	if !dr.progInit[c.pipe] {
		dr.progInit[c.pipe] = true
		gl.uniform1i(prog.Uniform("uAtlas"), 0)
		gl.uniform1i(prog.Uniform("uImage"), 0)
		gl.uniform1i(prog.Uniform("uLayer"), 0)
		gl.uniform1i(prog.Uniform("uLayerOwn"), 4)
		gl.uniform1i(prog.Uniform("uClipMask"), 1)
		gl.uniform1i(prog.Uniform("uPaintImage"), 2)
		gl.uniform1i(prog.Uniform("uDst"), 3)
	}
	t := dr.cur
	if dr.progGen[c.pipe] != dr.gen {
		dr.progGen[c.pipe] = dr.gen
		gl.uniform2f(prog.Uniform("uSurface"), float32(t.rect.Dx()), float32(t.rect.Dy()))
		gl.uniform2f(prog.Uniform("uOrigin"), float32(t.rect.Min.X), float32(t.rect.Min.Y))
	}
	// Making the copy's texture binds it to unit 0, so it comes before the
	// textures of the draw are bound.
	if c.spec != nil && c.spec.readBlend != 0 {
		if err := dr.copyDestination(prog, c); err != nil {
			return err
		}
	}
	if c.op == opComposite || c.op == opDrawCached {
		src := layer.fb.Texture()
		if filtered != nil {
			src = filtered.Texture()
		}
		d.BindTexture(0, src)
		maskStyle, drawnOnly := int32(-1), int32(0)
		if c.eff != nil {
			maskStyle, drawnOnly = c.eff.mask, b2i(c.eff.drawnOnly)
			if maskStyle > 0 {
				d.BindTexture(4, layer.fb.Texture())
			}
		}
		gl.uniform1i(prog.Uniform("uMaskStyle"), maskStyle)
		gl.uniform1i(prog.Uniform("uDrawnOnly"), drawnOnly)
		gl.uniform4f(prog.Uniform("uLayerRect"), float32(layer.rect.Min.X), float32(layer.rect.Min.Y), float32(layer.rect.Dx()), float32(layer.rect.Dy()))
	} else if c.tex != nil {
		d.BindTexture(0, c.tex)
	}
	if c.clip != nil {
		d.BindTexture(1, c.clip.tex)
		gl.uniform4f(prog.Uniform("uClipMaskRect"), float32(c.clip.x), float32(c.clip.y), float32(c.clip.w), float32(c.clip.h))
	} else {
		d.BindTexture(1, dr.r.res.white)
		gl.uniform4f(prog.Uniform("uClipMaskRect"), 0, 0, 0, 0)
	}
	if c.spec != nil && c.spec != dr.lastSpec[c.pipe] {
		dr.lastSpec[c.pipe] = c.spec
		applySpec(gl, prog, c.spec)
	}
	if c.spec != nil && c.spec.image != nil {
		d.BindTexture(2, c.spec.image)
	}
	// The scissor box counts from the bottom left of the target.
	sx, sy, sw, sh := c.scissor[0], c.scissor[1], c.scissor[2], c.scissor[3]
	d.SetScissor(true, sx-t.rect.Min.X, t.rect.Dy()-(sy-t.rect.Min.Y)-sh, sw, sh)
	d.BlendFunc(c.blend.srcRGB, c.blend.dstRGB, c.blend.srcA, c.blend.dstA)
	stride := int32(itemFloats[c.pipe] * 4)
	base := c.first * int(stride)
	if c.pipe == pipeVertices {
		gl.uniform4f(prog.Uniform("uPaint"), c.paint[0], c.paint[1], c.paint[2], c.paint[3])
		gl.uniform1i(prog.Uniform("uMode"), c.mode)
		gl.uniform1i(prog.Uniform("uHasColors"), b2i(c.hasColors))
		gl.uniform1i(prog.Uniform("uAtTexture"), b2i(c.atTexture))
		gl.uniform4f(prog.Uniform("uClip"), c.clipRect[0], c.clipRect[1], c.clipRect[2], c.clipRect[3])
		gl.vertexAttribPointer(0, 2, glFloat, false, stride, base)
		gl.vertexAttribPointer(1, 4, glFloat, false, stride, base+8)
		gl.vertexAttribPointer(2, 2, glFloat, false, stride, base+24)
		gl.drawArrays(glTriangles, 0, int32(c.count))
		return nil
	}
	for i := 0; i < vec4s[c.pipe]; i++ {
		gl.vertexAttribPointer(uint32(i), 4, glFloat, false, stride, base+i*16)
	}
	gl.drawArraysInstanced(glTriangleStrip, 0, 4, int32(c.count))
	return nil
}

// beginLayer starts drawing into a new layer, cleared, or for a backdrop filter a
// filtered copy of the pixels under it.
func (dr *drawer) beginLayer(c *drawCall) error {
	d := dr.r.ctx.dev
	gl := dr.r.ctx.gl
	w, h := c.rect.Dx(), c.rect.Dy()
	var backdrop *Texture
	if c.eff != nil && c.eff.backdrop {
		// The copy is of the framebuffer that is bound now: the layer's parent.
		t := dr.cur
		tex, err := d.AcquireTexture(w, h, TextureFormatRGBA8)
		if err != nil {
			return err
		}
		defer d.ReleaseTexture(tex)
		if err := d.CopyFramebuffer(0, tex, c.rect.Min.X-t.rect.Min.X, t.rect.Dy()-(c.rect.Min.Y-t.rect.Min.Y)-h, w, h); err != nil {
			return err
		}
		backdrop = tex
	}
	fb, err := d.AcquireFramebuffer(w, h, false)
	if err != nil {
		return err
	}
	dr.open = append(dr.open, dr.cur)
	if backdrop != nil {
		if _, err := dr.runPasses(backdrop, w, h, [2]int{c.rect.Min.X, c.rect.Min.Y}, c.eff.passes, fb); err != nil {
			return err
		}
		dr.enter(renderTarget{fb: fb, rect: c.rect})
		return nil
	}
	dr.enter(renderTarget{fb: fb, rect: c.rect})
	d.SetScissor(false, 0, 0, 0, 0)
	d.ColorMask(true, true, true, true)
	gl.clearColor(0, 0, 0, 0)
	gl.clear(glColorBufferBit)
	return nil
}

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// copyDestination copies the pixels of the call's scissor box, which are the ones
// it draws over, to a texture on unit 3, for a shader that blends with them.
func (dr *drawer) copyDestination(prog *Program, c *drawCall) error {
	d := dr.r.ctx.dev
	t := dr.cur
	sx, sy, sw, sh := c.scissor[0], c.scissor[1], c.scissor[2], c.scissor[3]
	round := func(v int) int { return min((v+dstGranule-1)/dstGranule*dstGranule, d.caps.MaxTextureSize) }
	tex, err := d.AcquireTexture(round(sw), round(sh), TextureFormatRGBA8)
	if err != nil {
		return err
	}
	defer d.ReleaseTexture(tex)
	if err := d.CopyFramebuffer(3, tex, sx-t.rect.Min.X, t.rect.Dy()-(sy-t.rect.Min.Y)-sh, sw, sh); err != nil {
		return err
	}
	dr.r.ctx.gl.uniform4f(prog.Uniform("uDstRect"), float32(sx), float32(sy+sh-1), 0, 0)
	return nil
}

// applySpec sets the uniforms of a paint.
func applySpec(gl glAPI, prog *Program, sp *paintSpec) {
	u := prog.Uniform
	gl.uniform1i(u("uShader"), sp.shader)
	gl.uniform4f(u("uShaderXf0"), sp.xf[0], sp.xf[1], sp.xf[2], sp.xf[3])
	gl.uniform2f(u("uShaderXf1"), sp.xf[4], sp.xf[5])
	gl.uniform4f(u("uShaderP0"), sp.p0[0], sp.p0[1], sp.p0[2], sp.p0[3])
	gl.uniform4f(u("uShaderP1"), sp.p1[0], sp.p1[1], sp.p1[2], sp.p1[3])
	gl.uniform1i(u("uTileX"), sp.tile[0])
	gl.uniform1i(u("uTileY"), sp.tile[1])
	gl.uniform1i(u("uRampCount"), sp.rampN)
	if sp.rampN > 0 {
		n := int(sp.rampN)
		colors := make([]float32, 0, n*4)
		for i := 0; i < n; i++ {
			colors = append(colors, sp.rampCol[i][:]...)
		}
		gl.uniform4fv(u("uRampColor"), colors)
		gl.uniform1fv(u("uRampPos"), sp.rampPos[:n])
	}
	gl.uniform4f(u("uPaintImageInfo"), sp.imgInfo[0], sp.imgInfo[1], sp.imgInfo[2], sp.imgInfo[3])
	gl.uniform4f(u("uConstColor"), sp.constCol[0], sp.constCol[1], sp.constCol[2], sp.constCol[3])
	gl.uniform1i(u("uDither"), b2i(sp.dither))
	gl.uniform1i(u("uFilter"), sp.filter)
	gl.uniform4f(u("uFilterColor"), sp.filterCol[0], sp.filterCol[1], sp.filterCol[2], sp.filterCol[3])
	gl.uniform1i(u("uFilterMode"), sp.filterMode)
	if sp.filter == filterMatrix {
		gl.uniform4fv(u("uFilterMatrix"), sp.filterMat[:])
	}
	gl.uniform1i(u("uInvert"), b2i(sp.invert))
	gl.uniform1i(u("uBlend"), sp.readBlend-1)
}
