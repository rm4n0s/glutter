//go:build js && wasm

package webgpu

import (
	"image"

	"syscall/js"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// encoder records the commands of one frame into a GPUCommandEncoder.
type encoder struct {
	d     *Device
	enc   js.Value
	pass  *pass
	reads []*readJob
}

var _ explicit.Encoder = (*encoder)(nil)

// Begin implements explicit.Device. It fails with WebGPUFrameOpen when the previous frame was not
// submitted, and with GPUDeviceLost when the device was lost.
func (d *Device) Begin() (explicit.Encoder, error) {
	if d.open != nil {
		return nil, errFrameOpen()
	}
	if err := d.usable(); err != nil {
		return nil, err
	}
	v, err := tryCall("createCommandEncoder", func() js.Value { return d.dev.Call("createCommandEncoder") })
	if err != nil {
		return nil, err
	}
	e := &encoder{d: d, enc: v}
	d.open = e
	return e, nil
}

// BeginPass implements explicit.Encoder. The target must be a texture of this device with
// TextureRenderTarget.
func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	t, ok := target.(*texture)
	if !ok || t.d != e.d {
		panic(errForeign("texture"))
	}
	if e.pass != nil {
		panic(errPassNotEnded())
	}
	att := map[string]any{"view": t.targetView(), "loadOp": "clear", "storeOp": "store"}
	if load == explicit.LoadLoad {
		att["loadOp"] = "load"
	} else {
		att["clearValue"] = map[string]any{"r": float64(clear.R), "g": float64(clear.G), "b": float64(clear.B), "a": float64(clear.A)}
	}
	p := e.enc.Call("beginRenderPass", map[string]any{"colorAttachments": []any{att}})
	e.pass = &pass{e: e, p: p, target: t}
	return e.pass
}

// CopyTexture implements explicit.Encoder.
func (e *encoder) CopyTexture(src explicit.Texture, r image.Rectangle, dst explicit.Texture, at image.Point) {
	s, ok1 := src.(*texture)
	t, ok2 := dst.(*texture)
	if !ok1 || !ok2 || s.d != e.d || t.d != e.d {
		panic(errForeign("texture"))
	}
	e.enc.Call("copyTextureToTexture",
		map[string]any{"texture": s.tex, "mipLevel": s.base, "origin": []any{r.Min.X, r.Min.Y}},
		map[string]any{"texture": t.tex, "mipLevel": t.base, "origin": []any{at.X, at.Y}},
		[]any{r.Dx(), r.Dy()})
}

// readJob is a ReadTexture that waits for the frame to finish and the buffer to map.
type readJob struct {
	id     int
	d      *Device
	buffer js.Value
	pitch  int
	rect   image.Rectangle
	format explicit.TextureFormat
	done   func(*image.RGBA, error)
}

// ReadTexture implements explicit.Encoder: the rectangle is copied into a buffer that the CPU can
// map, and done runs in a later task, once the GPU finished and the browser mapped the buffer; it
// runs on the UI goroutine (SPEC §9.5, §23.3). Submit starts the mapping.
func (e *encoder) ReadTexture(src explicit.Texture, r image.Rectangle, done func(*image.RGBA, error)) {
	s, ok := src.(*texture)
	if !ok || s.d != e.d {
		panic(errForeign("texture"))
	}
	d := e.d
	pitch := d.limits.AlignedRowPitch(r.Dx(), s.desc.Format.BytesPerPixel())
	buf, err := tryCall("createBuffer", func() js.Value {
		return d.dev.Call("createBuffer", map[string]any{"size": pitch * r.Dy(), "usage": bufferMapRead | bufferCopyDst})
	})
	if err != nil {
		done(nil, err)
		return
	}
	e.enc.Call("copyTextureToBuffer",
		map[string]any{"texture": s.tex, "mipLevel": s.base, "origin": []any{r.Min.X, r.Min.Y}},
		map[string]any{"buffer": buf, "bytesPerRow": pitch, "rowsPerImage": r.Dy()},
		[]any{r.Dx(), r.Dy()})
	job := &readJob{d: d, buffer: buf, pitch: pitch, rect: r, format: s.desc.Format, done: done}
	e.reads = append(e.reads, job)
}

// Submit implements explicit.Encoder: it sends the frame to the queue, then starts mapping the
// buffers of the reads. It does not wait for the GPU.
func (e *encoder) Submit() error {
	d := e.d
	if d.open != e {
		return errFrameNotOpen()
	}
	d.open = nil
	after := d.after
	d.after = nil
	if err := d.usable(); err != nil {
		return err
	}
	_, err := tryCall("submit", func() js.Value {
		cb := e.enc.Call("finish")
		return d.queue.Call("submit", []any{cb})
	})
	if err != nil {
		return err
	}
	for _, fn := range after {
		fn()
	}
	p := d.p
	for _, job := range e.reads {
		p.mu.Lock()
		p.nextID++
		job.id = p.nextID
		p.reads[job.id] = job
		p.mu.Unlock()
		p.helper.Call("map", job.buffer, job.id)
	}
	return nil
}

// onMapped is the browser telling that a read-back buffer is mapped: the pixels are copied out,
// the buffer is released, and the owner hears of it on the UI goroutine.
func (p *Provider) onMapped(id int) {
	job := p.takeRead(id)
	if job == nil {
		return
	}
	rng := job.buffer.Call("getMappedRange")
	view := js.Global().Get("Uint8Array").New(rng)
	raw := make([]byte, view.Length())
	js.CopyBytesToGo(raw, view)
	job.buffer.Call("unmap")
	job.buffer.Call("destroy")
	img := unpackRows(raw, job.pitch, job.rect.Dx(), job.rect.Dy(), job.format)
	p.post(func() { job.done(img, nil) })
}

// onMapFailed is the browser failing to map a read-back buffer: the device was lost, or destroyed.
func (p *Provider) onMapFailed(id int, message string) {
	job := p.takeRead(id)
	if job == nil {
		return
	}
	var err error
	if lost := job.d.loss.err(); lost != nil {
		err = lost
	} else {
		err = errReadFailed(message)
	}
	p.post(func() { job.done(nil, err) })
}

func (p *Provider) takeRead(id int) *readJob {
	p.mu.Lock()
	defer p.mu.Unlock()
	job := p.reads[id]
	delete(p.reads, id)
	return job
}

// pass records the draws of one render pass.
type pass struct {
	e        *encoder
	p        js.Value
	target   *texture
	pipeline *pipeline
}

var _ explicit.Pass = (*pass)(nil)

// SetPipeline implements explicit.Pass.
func (p *pass) SetPipeline(pl explicit.Pipeline) {
	pp, ok := pl.(*pipeline)
	if !ok || pp.d != p.e.d {
		panic(errForeign("pipeline"))
	}
	p.pipeline = pp
	p.p.Call("setPipeline", pp.pipe)
}

// SetBindGroup implements explicit.Pass. The dynamic offsets cross in the reused Uint32Array.
func (p *pass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	d := p.e.d
	d.limits.CheckSetBindGroup(group, dynamicOffsets)
	bg, ok := g.(*bindGroup)
	if !ok {
		panic(errForeign("bind group"))
	}
	if len(dynamicOffsets) == 0 {
		p.p.Call("setBindGroup", group, bg.g)
		return
	}
	for i, o := range dynamicOffsets {
		d.dynamic.SetIndex(i, o)
	}
	p.p.Call("setBindGroup", group, bg.g, d.dynamic, 0, len(dynamicOffsets))
}

// SetVertexBuffer implements explicit.Pass.
func (p *pass) SetVertexBuffer(slot int, b explicit.Buffer, offset int) {
	d := p.e.d
	d.limits.CheckSetVertexBuffer(slot)
	vb, ok := b.(*buffer)
	if !ok || vb.d != d {
		panic(errForeign("buffer"))
	}
	p.p.Call("setVertexBuffer", slot, vb.buf, offset)
}

// SetIndexBuffer implements explicit.Pass.
func (p *pass) SetIndexBuffer(b explicit.Buffer, offset int) {
	ib, ok := b.(*buffer)
	if !ok || ib.d != p.e.d {
		panic(errForeign("buffer"))
	}
	p.p.Call("setIndexBuffer", ib.buf, "uint16", offset)
}

// SetScissor implements explicit.Pass. The rectangle is clipped to the target.
func (p *pass) SetScissor(r image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, p.target.desc.Width, p.target.desc.Height))
	p.p.Call("setScissorRect", r.Min.X, r.Min.Y, r.Dx(), r.Dy())
}

// Draw implements explicit.Pass.
func (p *pass) Draw(vertices, instances, firstVertex, firstInstance int) {
	p.p.Call("draw", vertices, instances, firstVertex, firstInstance)
}

// DrawIndexed implements explicit.Pass.
func (p *pass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	p.p.Call("drawIndexed", indices, instances, firstIndex, baseVertex, firstInstance)
}

// End implements explicit.Pass.
func (p *pass) End() {
	p.p.Call("end")
	p.e.pass = nil
}
