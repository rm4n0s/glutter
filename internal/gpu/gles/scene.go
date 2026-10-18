package gles

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// drawScene draws the scene into the framebuffer that is bound, which is w x h
// pixels, over what is there. It flattens the scene, turns the commands into the
// frame's instances and vertices (failing before anything is drawn if one is
// unsupported, unless the renderer skips them), and draws the calls in order.
func (r *Renderer) drawScene(scene *ui.Scene, w, h int) error {
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
	list, err := gpu.Flatten(scene, gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, float64(w), float64(h))})
	if err != nil {
		return err
	}
	b := &frameBuilder{w: w, h: h, skip: r.skip, res: r.res}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			return err
		}
	}
	b.stats.Instances = b.frame.items(pipeShape) + b.frame.items(pipeMask) + b.frame.items(pipeImage)
	b.stats.Vertices = b.frame.items(pipeVertices)
	b.stats.Batches = len(b.frame.calls)
	b.stats.TilesRasterized = r.res.tilesRasterized - tiles0
	b.stats.ClipMasksBuilt = r.res.clipMasksBuilt - clips0
	b.stats.ImagesUploaded = r.res.imagesUploaded - images0
	r.last = b.stats
	return r.drawFrame(&b.frame, w, h)
}

// vec4s is the number of vec4 attributes of each instanced pipeline.
var vec4s = [numPipelines]int{pipeShape: 12, pipeMask: 4, pipeImage: 8}

// setupVAO describes the attributes of a pipeline in its vertex array.
func (r *Renderer) setupVAO(p pipeline) uint32 {
	gl := r.ctx.gl
	return r.ctx.dev.VertexArray(programNames[p], func(uint32) {
		if p == pipeVertices {
			gl.enableVertexAttribArray(0)
			gl.enableVertexAttribArray(1)
			return
		}
		for i := uint32(0); i < uint32(vec4s[p]); i++ {
			gl.enableVertexAttribArray(i)
			gl.vertexAttribDivisor(i, 1)
		}
	})
}

// drawFrame uploads the data of each pipeline once and issues the draw calls.
func (r *Renderer) drawFrame(f *frame, w, h int) error {
	if len(f.calls) == 0 {
		return nil
	}
	d := r.ctx.dev
	gl := r.ctx.gl
	var bufs [numPipelines]*Buffer
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
		bufs[p] = buf
	}
	d.Viewport(0, 0, w, h)
	d.SetBlend(true)
	d.BlendEquation(glFuncAdd)
	d.ColorMask(true, true, true, true)

	current := pipeline(-1)
	var prog *Program
	for _, c := range f.calls {
		if c.pipe != current {
			var err error
			if prog, err = d.Program(programNames[c.pipe]); err != nil {
				return err
			}
			d.BindVertexArray(r.setupVAO(c.pipe))
			d.BindArrayBuffer(bufs[c.pipe])
			d.UseProgram(prog)
			gl.uniform2f(prog.Uniform("uSurface"), float32(w), float32(h))
			gl.uniform1i(prog.Uniform("uAtlas"), 0)
			gl.uniform1i(prog.Uniform("uImage"), 0)
			gl.uniform1i(prog.Uniform("uClipMask"), 1)
			current = c.pipe
		}
		if c.tex != nil {
			d.BindTexture(0, c.tex)
		}
		if c.clip != nil {
			d.BindTexture(1, c.clip.tex)
			gl.uniform4f(prog.Uniform("uClipMaskRect"), float32(c.clip.x), float32(c.clip.y), float32(c.clip.w), float32(c.clip.h))
		} else {
			d.BindTexture(1, r.res.white)
			gl.uniform4f(prog.Uniform("uClipMaskRect"), 0, 0, 0, 0)
		}
		// The scissor box counts from the bottom left.
		d.SetScissor(true, c.scissor[0], h-c.scissor[1]-c.scissor[3], c.scissor[2], c.scissor[3])
		d.BlendFunc(c.blend.srcRGB, c.blend.dstRGB, c.blend.srcA, c.blend.dstA)
		stride := int32(itemFloats[c.pipe] * 4)
		base := c.first * int(stride)
		if c.pipe == pipeVertices {
			gl.uniform4f(prog.Uniform("uPaint"), c.paint[0], c.paint[1], c.paint[2], c.paint[3])
			gl.uniform1i(prog.Uniform("uMode"), c.mode)
			hasColors := int32(0)
			if c.hasColors {
				hasColors = 1
			}
			gl.uniform1i(prog.Uniform("uHasColors"), hasColors)
			gl.uniform4f(prog.Uniform("uClip"), c.clipRect[0], c.clipRect[1], c.clipRect[2], c.clipRect[3])
			gl.vertexAttribPointer(0, 2, glFloat, false, stride, base)
			gl.vertexAttribPointer(1, 4, glFloat, false, stride, base+8)
			gl.drawArrays(glTriangles, 0, int32(c.count))
			continue
		}
		for i := 0; i < vec4s[c.pipe]; i++ {
			gl.vertexAttribPointer(uint32(i), 4, glFloat, false, stride, base+i*16)
		}
		gl.drawArraysInstanced(glTriangleStrip, 0, 4, int32(c.count))
	}
	d.SetScissor(false, 0, 0, 0, 0)
	return d.checkError("draw frame")
}
