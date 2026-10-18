package gles

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// shapeVAO is the name of the vertex array of the shape pipeline.
const shapeVAO = "shape"

// drawScene draws the scene into the framebuffer that is bound, which is w x h
// pixels, over what is there. It flattens the scene, turns the commands into
// instances (failing before anything is drawn if one is unsupported, unless the
// renderer skips them), and draws the batches in order.
func (r *Renderer) drawScene(scene *ui.Scene, w, h int) error {
	r.last = FrameStats{}
	if scene == nil {
		return nil
	}
	list, err := gpu.Flatten(scene, gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, float64(w), float64(h))})
	if err != nil {
		return err
	}
	b := &frameBuilder{w: w, h: h, skip: r.skip}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			return err
		}
	}
	b.stats.Instances = b.shape.instances()
	b.stats.Batches = len(b.shape.batches)
	r.last = b.stats
	return r.drawShapes(&b.shape, w, h)
}

// drawShapes uploads the instances of a frame and draws its batches.
func (r *Renderer) drawShapes(f *shapeFrame, w, h int) error {
	if f.instances() == 0 {
		return nil
	}
	d := r.ctx.dev
	gl := r.ctx.gl
	p, err := d.Program("shape")
	if err != nil {
		return err
	}
	buf, err := d.AcquireBuffer(glArrayBuffer, len(f.data)*4)
	if err != nil {
		return err
	}
	defer d.ReleaseBuffer(buf)
	if err := d.UploadBuffer(buf, floatBytes(f.data)); err != nil {
		return err
	}
	vao := d.VertexArray(shapeVAO, func(uint32) {
		for i := uint32(0); i < shapeVec4s; i++ {
			gl.enableVertexAttribArray(i)
			gl.vertexAttribDivisor(i, 1)
		}
	})
	d.BindVertexArray(vao)
	d.BindArrayBuffer(buf)
	d.UseProgram(p)
	gl.uniform2f(p.Uniform("uSurface"), float32(w), float32(h))
	d.Viewport(0, 0, w, h)
	d.SetBlend(true)
	d.BlendEquation(glFuncAdd)
	d.ColorMask(true, true, true, true)
	for _, bt := range f.batches {
		// The scissor box counts from the bottom left.
		d.SetScissor(true, bt.scissor[0], h-bt.scissor[1]-bt.scissor[3], bt.scissor[2], bt.scissor[3])
		d.BlendFunc(bt.blend.srcRGB, bt.blend.dstRGB, bt.blend.srcA, bt.blend.dstA)
		base := bt.first * shapeStride
		for i := 0; i < shapeVec4s; i++ {
			gl.vertexAttribPointer(uint32(i), 4, glFloat, false, shapeStride, base+i*16)
		}
		gl.drawArraysInstanced(glTriangleStrip, 0, 4, int32(bt.count))
	}
	d.SetScissor(false, 0, 0, 0, 0)
	return d.checkError("draw shapes")
}
