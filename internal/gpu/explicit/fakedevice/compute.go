package fakedevice

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The fake device's part of step 30.2 (SPEC §27.1): compute pipelines and passes, the copies, the read of a
// buffer, query sets and the completion of an encoder. It records every call and checks the descriptors
// the way the real devices do; the buffer copies and reads work on the bytes it keeps, and nothing runs
// a shader.

// NewComputePipeline implements explicit.Device.
func (d *Device) NewComputePipeline(desc explicit.ComputePipelineDesc) (explicit.ComputePipeline, error) {
	name := "<nil>"
	if desc.Shader != nil {
		name = desc.Shader.Name
	}
	d.record("NewComputePipeline", "shader=%s entry=%s", name, desc.Entry)
	d.limits.CheckComputePipeline(desc)
	if err := d.fail(); err != nil {
		return nil, err
	}
	d.live.Pipelines++
	return &computePipeline{d: d, desc: desc}, nil
}

// NewComputeBindGroup implements explicit.Device.
func (d *Device) NewComputeBindGroup(p explicit.ComputePipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.record("NewComputeBindGroup", "group=%d entries=%d", group, len(entries))
	d.limits.CheckBindGroup(group, entries)
	if cp, ok := p.(*computePipeline); ok && cp.desc.Shader != nil {
		d.checkBindings(group, entries, cp.desc.Shader, cp.desc.Entry, "")
	}
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &bindGroup{}, nil
}

// checkBindings checks each entry of a bind group against the binding of the entry points it is for.
func (d *Device) checkBindings(group int, entries []explicit.Binding, sh *explicit.Shader, entryNames ...string) {
	for _, e := range entries {
		for _, name := range entryNames {
			entry, ok := sh.Entry(name)
			if !ok {
				continue
			}
			found := false
			for _, sb := range entry.Bindings {
				if sb.Group == group && sb.Binding == e.Binding {
					d.limits.CheckBinding(sb, e)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
}

// NewQuerySet implements explicit.Device. It fails with GPUQueryUnsupported when Supported.TimestampQuery is
// false.
func (d *Device) NewQuerySet(count int) (explicit.QuerySet, error) {
	d.record("NewQuerySet", "count=%d", count)
	if !d.Supported.TimestampQuery {
		return nil, errors.New(tagGPUQueryUnsupported, "the device has no timestamp queries", "count", count)
	}
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &querySet{d: d, n: count}, nil
}

type computePipeline struct {
	d         *Device
	desc      explicit.ComputePipelineDesc
	destroyed bool
}

func (p *computePipeline) Desc() explicit.ComputePipelineDesc { return p.desc }
func (p *computePipeline) Destroy() {
	if !p.destroyed {
		p.destroyed = true
		p.d.live.Pipelines--
	}
}

type querySet struct {
	d *Device
	n int
}

func (q *querySet) Count() int { return q.n }
func (q *querySet) Destroy()   { q.d.record("DestroyQuerySet", "") }

// BeginComputePass implements explicit.Encoder.
func (e *encoder) BeginComputePass(desc explicit.ComputePassDesc) explicit.ComputePass {
	e.d.record("BeginComputePass", "timestamps=%t", desc.Timestamps != nil)
	if (e.pass != nil && !e.pass.ended) || (e.cpass != nil && !e.cpass.ended) {
		panic(errors.New(tagFakePassNotEnded, "BeginComputePass was called before the previous pass ended"))
	}
	e.d.limits.CheckTimestamps(e.d.Supported, desc.Timestamps)
	e.cpass = &computePass{d: e.d}
	return e.cpass
}

// CopyBuffer implements explicit.Encoder. It copies the bytes the fake buffers keep.
func (e *encoder) CopyBuffer(src explicit.Buffer, srcOffset int, dst explicit.Buffer, dstOffset, size int) {
	e.d.record("CopyBuffer", "src_offset=%d dst_offset=%d size=%d", srcOffset, dstOffset, size)
	e.d.limits.CheckCopyBuffer(src, srcOffset, dst, dstOffset, size)
	s, ok1 := src.(*buffer)
	t, ok2 := dst.(*buffer)
	if ok1 && ok2 && srcOffset >= 0 && dstOffset >= 0 && srcOffset+size <= len(s.data) && dstOffset+size <= len(t.data) {
		copy(t.data[dstOffset:dstOffset+size], s.data[srcOffset:srcOffset+size])
	}
}

// CopyBufferToTexture implements explicit.Encoder. It records the call and checks it.
func (e *encoder) CopyBufferToTexture(src explicit.Buffer, layout explicit.BufferLayout, dst explicit.Texture, region explicit.TextureRegion) {
	e.d.record("CopyBufferToTexture", "layout=%+v region=%+v", layout, region)
	e.d.limits.CheckBufferTexture(src, layout, dst.Desc(), region, true)
}

// CopyTextureToBuffer implements explicit.Encoder. It records the call and checks it.
func (e *encoder) CopyTextureToBuffer(src explicit.Texture, region explicit.TextureRegion, dst explicit.Buffer, layout explicit.BufferLayout) {
	e.d.record("CopyTextureToBuffer", "layout=%+v region=%+v", layout, region)
	e.d.limits.CheckBufferTexture(dst, layout, src.Desc(), region, false)
}

// ReadBuffer implements explicit.Encoder. done runs at CompleteReads, or at Submit when
// CompleteReadsOnSubmit is set.
func (e *encoder) ReadBuffer(src explicit.Buffer, offset, size int, done func(data []byte, err error)) {
	e.d.record("ReadBuffer", "offset=%d size=%d", offset, size)
	e.d.limits.CheckReadBuffer(src, offset, size)
	b, ok := src.(*buffer)
	if !ok || offset < 0 || size < 0 || offset+size > len(b.data) {
		return
	}
	e.d.reads = append(e.d.reads, pendingRead{buf: b, off: offset, size: size, doneBuf: done})
}

// ResolveQuerySet implements explicit.Encoder. It records the call and checks it.
func (e *encoder) ResolveQuerySet(set explicit.QuerySet, first, count int, dst explicit.Buffer, dstOffset int) {
	e.d.record("ResolveQuerySet", "first=%d count=%d offset=%d", first, count, dstOffset)
	e.d.limits.CheckResolveQuerySet(set, first, count, dst, dstOffset)
}

// OnComplete implements explicit.Encoder. fn runs when the encoder is submitted and the reads complete.
func (e *encoder) OnComplete(fn func()) {
	e.d.record("OnComplete", "")
	e.complete = append(e.complete, fn)
}

type computePass struct {
	d     *Device
	ended bool
	pipe  *computePipeline
}

func (p *computePass) SetPipeline(pl explicit.ComputePipeline) {
	p.d.record("SetComputePipeline", "entry=%s", pl.Desc().Entry)
	if cp, ok := pl.(*computePipeline); ok {
		p.pipe = cp
	}
}

func (p *computePass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	p.d.record("SetComputeBindGroup", "group=%d offsets=%v", group, dynamicOffsets)
	p.d.limits.CheckSetBindGroup(group, dynamicOffsets)
}

func (p *computePass) Dispatch(x, y, z int) {
	p.d.record("Dispatch", "%d %d %d", x, y, z)
	p.d.limits.CheckDispatch(x, y, z)
}

func (p *computePass) DispatchIndirect(b explicit.Buffer, offset int) {
	p.d.record("DispatchIndirect", "offset=%d", offset)
	p.d.limits.CheckIndirect(b, offset, 12)
}

func (p *computePass) End() {
	p.d.record("EndComputePass", "")
	p.ended = true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUQueryUnsupported is the tag of the error of NewQuerySet on a device without timestamp queries.
	tagGPUQueryUnsupported = "GPUQueryUnsupported"
)
