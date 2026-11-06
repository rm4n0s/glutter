package gpu

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// CommandEncoder records commands (WebGPU's GPUCommandEncoder): passes, and copies between buffers and textures.
// Finish makes a CommandBuffer of them and Queue.Submit sends it to the GPU. Nothing runs before the submission.
// The encoder is used on the UI thread and is not released: it goes when it is finished.
//
// A misuse while encoding (a pass not ended, a copy outside a buffer, a draw without a pipeline) is a programmer
// error: in debug builds it panics with GPUEncodingInvalid (SPEC §27.2).
//
// Flutter: no counterpart
type CommandEncoder struct {
	d    *Device
	cmds []command
	// pass says that a pass was begun and not ended.
	pass bool
	done bool
	// renders and computes are the passes of the encoder, which go back to the queue's list with the encoder.
	renders  []*RenderPass
	computes []*ComputePass
	// buf is the command buffer that Finish returns: it is part of the encoder, so that making one allocates nothing.
	buf CommandBuffer
}

// command is a recorded command: a pass, or a function for the copies and the resolves.
type command struct {
	fn func(r *replay)
	rp *RenderPass
	cp *ComputePass
}

// NewCommandEncoder starts a list of commands (WebGPU's GPUDevice.createCommandEncoder). The encoder, its command buffer and its
// passes are kept by the queue for the next encoder once the buffer is submitted, so a frame that records the same commands each
// time allocates nothing: do not use an encoder, a pass or a command buffer after it was submitted.
//
// Flutter: no counterpart
func (d *Device) NewCommandEncoder() *CommandEncoder {
	d.assertThread("NewCommandEncoder")
	q := d.queue
	if n := len(q.encoders); n > 0 {
		e := q.encoders[n-1]
		q.encoders[n-1] = nil
		q.encoders = q.encoders[:n-1]
		e.done, e.pass, e.buf.submitted = false, false, false
		return e
	}
	e := &CommandEncoder{d: d}
	e.buf = CommandBuffer{d: d, e: e}
	return e
}

// recycle gives the encoder and its passes back to the queue, once the commands have run on the explicit encoder.
func (e *CommandEncoder) recycle() {
	q := e.d.queue
	for _, p := range e.renders {
		p.reset()
		q.renders = append(q.renders, p)
	}
	for _, p := range e.computes {
		p.reset()
		q.computes = append(q.computes, p)
	}
	clear(e.renders)
	clear(e.computes)
	e.renders, e.computes = e.renders[:0], e.computes[:0]
	clear(e.cmds)
	e.cmds = e.cmds[:0]
	q.encoders = append(q.encoders, e)
}

// CommandBuffer is the finished commands of an encoder (WebGPU's GPUCommandBuffer). It is submitted once.
//
// Flutter: no counterpart
type CommandBuffer struct {
	d         *Device
	e         *CommandEncoder
	submitted bool
}

// replay runs the commands on the explicit encoder of a submission.
func (cb *CommandBuffer) replay(r *replay) {
	for i := range cb.e.cmds {
		c := &cb.e.cmds[i]
		switch {
		case c.rp != nil:
			c.rp.runOn(r)
		case c.cp != nil:
			c.cp.runOn(r)
		default:
			c.fn(r)
		}
	}
}

// Finish ends the recording and returns the commands (WebGPU's GPUCommandEncoder.finish). No pass is open.
//
// Flutter: no counterpart
func (e *CommandEncoder) Finish() *CommandBuffer {
	e.d.assertThread("CommandEncoder.Finish")
	misuse(!e.done, "CommandEncoder.Finish", "an encoder is finished once")
	misuse(!e.pass, "CommandEncoder.Finish", "the pass is ended before the encoder is finished")
	e.done = true
	return &e.buf
}

// open checks that a command may be recorded now.
func (e *CommandEncoder) open(call string) {
	e.d.assertThread(call)
	misuse(!e.done, call, "the encoder is not finished")
	misuse(!e.pass, call, "the pass is ended before another command")
}

func (b *Buffer) sameDevice(d *Device) bool { return b != nil && b.d == d }

// CopyBufferToBuffer copies size bytes (WebGPU's GPUCommandEncoder.copyBufferToBuffer). The source has
// BufferUsageCopySrc and the destination BufferUsageCopyDst; the offsets and the size are multiples of 4, the
// ranges are inside the buffers, and the buffers are two.
//
// Flutter: no counterpart
func (e *CommandEncoder) CopyBufferToBuffer(src *Buffer, srcOffset int, dst *Buffer, dstOffset, size int) {
	const call = "CommandEncoder.CopyBufferToBuffer"
	e.open(call)
	d := e.d
	misuse(src.sameDevice(d) && dst.sameDevice(d), call, "the buffers were made by this device")
	src.usable(call)
	dst.usable(call)
	misuse(src.usage&BufferUsageCopySrc != 0, call, "the source has BufferUsageCopySrc")
	misuse(dst.usage&BufferUsageCopyDst != 0, call, "the destination has BufferUsageCopyDst")
	misuse(srcOffset%4 == 0 && dstOffset%4 == 0 && size%4 == 0, call, "the offsets and the size of a copy are multiples of 4",
		"src_offset", srcOffset, "dst_offset", dstOffset, "size", size)
	misuse(srcOffset >= 0 && dstOffset >= 0 && size >= 0 && srcOffset+size <= src.size && dstOffset+size <= dst.size, call,
		"the ranges of a copy are inside the buffers", "src_offset", srcOffset, "dst_offset", dstOffset, "size", size, "src", src.size, "dst", dst.size)
	misuse(src != dst, call, "a buffer is not copied onto itself")
	if size == 0 {
		return
	}
	e.cmds = append(e.cmds, command{fn: func(r *replay) {
		src.usable(call)
		dst.usable(call)
		r.markUsed(src)
		r.markUsed(dst)
		r.enc.CopyBuffer(src.exp, srcOffset, dst.exp, dstOffset, size)
	}})
}

// copyBox checks the box of a copy between a buffer and a texture and returns the explicit region and layout. The
// zero BytesPerRow is the width of the box in bytes, rounded up to Limits.BytesPerRowAlignment, and the zero
// RowsPerImage is the height of the box.
func (e *CommandEncoder) copyBox(call string, buf BufferCopy, tc TextureCopy, size Extent3D, toTexture bool) (explicit.TextureRegion, explicit.BufferLayout, bool) {
	d := e.d
	b, t := buf.Buffer, tc.Texture
	misuse(b.sameDevice(d) && t != nil && t.d == d, call, "the buffer and the texture were made by this device")
	b.usable(call)
	t.usable(call)
	f := explicit.TextureFormat(t.desc.Format)
	if toTexture {
		misuse(b.usage&BufferUsageCopySrc != 0, call, "the source buffer has BufferUsageCopySrc")
		misuse(f.CanBeWritten(), call, "a texture that is written has a format that can be written", "format", int(t.desc.Format))
	} else {
		misuse(b.usage&BufferUsageCopyDst != 0, call, "the destination buffer has BufferUsageCopyDst")
		misuse(t.desc.Usage&(explicit.TextureCopySource|explicit.TextureRenderTarget) != 0, call, "a texture that is read has TextureUsageCopySrc or TextureUsageRenderAttachment")
		misuse(f.CanBeCopied(), call, "a texture that is read has a format that can be copied", "format", int(t.desc.Format))
	}
	misuse(t.desc.SampleCount == 1, call, "a multisampled texture is not copied")
	region := explicit.TextureRegion{Level: tc.MipLevel, Origin: [3]int{tc.Origin.X, tc.Origin.Y, tc.Origin.Z}, Size: [3]int{size.Width, size.Height, size.DepthOrArrayLayers}}
	misuse(t.boxInside(region), call, "the box of a copy is inside the mip level", "level", tc.MipLevel, "origin", region.Origin, "size", region.Size)
	bpp := max(f.BytesPerPixel(), 1)
	layout := explicit.BufferLayout{Offset: buf.Layout.Offset, BytesPerRow: buf.Layout.BytesPerRow, RowsPerImage: buf.Layout.RowsPerImage}
	align := d.limits.CopyRowPitchAlignment
	if layout.BytesPerRow == 0 {
		layout.BytesPerRow = (size.Width*bpp + align - 1) / align * align
	}
	if layout.RowsPerImage == 0 {
		layout.RowsPerImage = size.Height
	}
	misuse(layout.BytesPerRow%align == 0 && layout.BytesPerRow >= size.Width*bpp, call, "the rows of a copy are a multiple of Limits.BytesPerRowAlignment apart and hold a row",
		"bytes_per_row", layout.BytesPerRow, "row", size.Width*bpp)
	misuse(layout.RowsPerImage >= size.Height, call, "the rows per image of a copy hold the box", "rows_per_image", layout.RowsPerImage, "height", size.Height)
	misuse(layout.Offset >= 0 && layout.Offset%4 == 0 && layout.Offset%bpp == 0, call, "the offset of a copy is a multiple of 4 and of the bytes of a texel", "offset", layout.Offset)
	if size.Width < 1 || size.Height < 1 || size.DepthOrArrayLayers < 1 {
		return region, layout, false
	}
	last := layout.Offset + layout.BytesPerRow*layout.RowsPerImage*(size.DepthOrArrayLayers-1) + layout.BytesPerRow*(size.Height-1) + size.Width*bpp
	misuse(last <= b.size, call, "the rows of a copy are inside the buffer", "end", last, "buffer", b.size)
	return region, layout, true
}

// CopyBufferToTexture copies rows of texels from a buffer into a box of a texture (WebGPU's
// GPUCommandEncoder.copyBufferToTexture). size is the box: its width and height, and the layers of an array or a
// cube or the depth of a volume. The layout of the buffer: BytesPerRow is a multiple of Limits.BytesPerRowAlignment
// (0 is the width of the box rounded up to it), RowsPerImage is the distance in rows between two layers (0 is the
// height of the box). A depth texture and a multisampled texture are not copied into.
//
// Flutter: no counterpart
func (e *CommandEncoder) CopyBufferToTexture(src BufferCopy, dst TextureCopy, size Extent3D) {
	const call = "CommandEncoder.CopyBufferToTexture"
	e.open(call)
	region, layout, ok := e.copyBox(call, src, dst, size, true)
	if !ok {
		return
	}
	b, t := src.Buffer, dst.Texture
	e.cmds = append(e.cmds, command{fn: func(r *replay) {
		b.usable(call)
		t.usable(call)
		r.markUsed(b)
		r.enc.CopyBufferToTexture(b.exp, layout, t.exp, region)
	}})
}

// CopyTextureToBuffer copies a box of a texture into rows in a buffer (WebGPU's
// GPUCommandEncoder.copyTextureToBuffer), with the layout rules of CopyBufferToTexture. The texture has
// TextureUsageCopySrc (or RenderAttachment), is not multisampled, and has a format that can be copied (not
// Depth24PlusStencil8).
//
// Flutter: no counterpart
func (e *CommandEncoder) CopyTextureToBuffer(src TextureCopy, dst BufferCopy, size Extent3D) {
	const call = "CommandEncoder.CopyTextureToBuffer"
	e.open(call)
	region, layout, ok := e.copyBox(call, dst, src, size, false)
	if !ok {
		return
	}
	b, t := dst.Buffer, src.Texture
	e.cmds = append(e.cmds, command{fn: func(r *replay) {
		b.usable(call)
		t.usable(call)
		r.markUsed(b)
		r.enc.CopyTextureToBuffer(t.exp, region, b.exp, layout)
	}})
}

// subresource returns the explicit texture over one layer of one mip level.
func (t *Texture) subresource(level, layer int) explicit.Texture {
	if t.desc.Kind == explicit.TextureKind2DArray || t.desc.Kind == explicit.TextureKindCube {
		return t.LayerView(level, layer).exp
	}
	return t.LevelView(level).exp
}

// CopyTextureToTexture copies a box between two textures of one format (WebGPU's
// GPUCommandEncoder.copyTextureToTexture). Both are 2D, 2D arrays or cubes (a volume is not copied: copy it through
// a buffer), not multisampled and not depth textures; size.DepthOrArrayLayers is the number of layers. The boxes
// are inside their mip levels, and the source has TextureUsageCopySrc or RenderAttachment.
//
// Flutter: no counterpart
func (e *CommandEncoder) CopyTextureToTexture(src, dst TextureCopy, size Extent3D) {
	const call = "CommandEncoder.CopyTextureToTexture"
	e.open(call)
	d := e.d
	s, t := src.Texture, dst.Texture
	misuse(s != nil && t != nil && s.d == d && t.d == d, call, "the textures were made by this device")
	s.usable(call)
	t.usable(call)
	misuse(s.desc.Format == t.desc.Format, call, "the textures have one format")
	misuse(s.desc.Kind != explicit.TextureKind3D && t.desc.Kind != explicit.TextureKind3D, call, "a volume is not copied: copy it through a buffer")
	misuse(!s.Format().isDepthStencil(), call, "a depth texture is not copied")
	misuse(s.desc.SampleCount == 1 && t.desc.SampleCount == 1, call, "a multisampled texture is not copied")
	misuse(s.desc.Usage&(explicit.TextureCopySource|explicit.TextureRenderTarget) != 0, call, "the source has TextureUsageCopySrc or TextureUsageRenderAttachment")
	sr := explicit.TextureRegion{Level: src.MipLevel, Origin: [3]int{src.Origin.X, src.Origin.Y, src.Origin.Z}, Size: [3]int{size.Width, size.Height, size.DepthOrArrayLayers}}
	dr := explicit.TextureRegion{Level: dst.MipLevel, Origin: [3]int{dst.Origin.X, dst.Origin.Y, dst.Origin.Z}, Size: [3]int{size.Width, size.Height, size.DepthOrArrayLayers}}
	misuse(s.boxInside(sr) && t.boxInside(dr), call, "the boxes of a copy are inside their mip levels", "size", sr.Size)
	misuse(s != t || src.MipLevel != dst.MipLevel || src.Origin.Z+size.DepthOrArrayLayers <= dst.Origin.Z || dst.Origin.Z+size.DepthOrArrayLayers <= src.Origin.Z ||
		src.Origin.X+size.Width <= dst.Origin.X || dst.Origin.X+size.Width <= src.Origin.X || src.Origin.Y+size.Height <= dst.Origin.Y || dst.Origin.Y+size.Height <= src.Origin.Y,
		call, "the boxes of a copy inside one texture do not overlap")
	if size.Width < 1 || size.Height < 1 || size.DepthOrArrayLayers < 1 {
		return
	}
	type layerCopy struct {
		from, to explicit.Texture
	}
	copies := make([]layerCopy, size.DepthOrArrayLayers)
	for i := range copies {
		copies[i] = layerCopy{s.subresource(src.MipLevel, src.Origin.Z+i), t.subresource(dst.MipLevel, dst.Origin.Z+i)}
	}
	rect := image.Rect(src.Origin.X, src.Origin.Y, src.Origin.X+size.Width, src.Origin.Y+size.Height)
	at := image.Pt(dst.Origin.X, dst.Origin.Y)
	e.cmds = append(e.cmds, command{fn: func(r *replay) {
		s.usable(call)
		t.usable(call)
		for _, c := range copies {
			r.enc.CopyTexture(c.from, rect, c.to, at)
		}
	}})
}

// ResolveQuerySet writes count timestamps of the set from first, as 64 bit unsigned integers in ticks
// (Features.TimestampPeriod), into dst at offset (WebGPU's GPUCommandEncoder.resolveQuerySet). dst has
// BufferUsageQueryResolve and the offset is a multiple of Limits.MinStorageBufferOffsetAlignment. Read the bytes
// with Buffer.Read after a copy into a buffer with BufferUsageCopySrc.
//
// Flutter: no counterpart
func (e *CommandEncoder) ResolveQuerySet(q *QuerySet, first, count int, dst *Buffer, offset int) {
	const call = "CommandEncoder.ResolveQuerySet"
	e.open(call)
	d := e.d
	misuse(q != nil && q.d == d && dst.sameDevice(d), call, "the query set and the buffer were made by this device")
	q.usable(call)
	dst.usable(call)
	misuse(first >= 0 && count >= 0 && first+count <= q.count, call, "the queries of a resolve are in the set", "first", first, "count", count, "set", q.count)
	misuse(dst.usage&BufferUsageQueryResolve != 0, call, "the buffer of a resolve has BufferUsageQueryResolve")
	misuse(offset%d.limits.StorageOffsetAlignment == 0, call, "the offset of a resolve is a multiple of Limits.MinStorageBufferOffsetAlignment", "offset", offset)
	misuse(offset >= 0 && offset+8*count <= dst.size, call, "the timestamps of a resolve fit the buffer", "offset", offset, "count", count, "buffer", dst.size)
	if count == 0 {
		return
	}
	e.cmds = append(e.cmds, command{fn: func(r *replay) {
		q.usable(call)
		dst.usable(call)
		r.markUsed(dst)
		r.enc.ResolveQuerySet(q.exp, first, count, dst.exp, offset)
	}})
}
