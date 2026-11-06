package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// bufferUsageKnown is every bit that BufferUsage has.
const bufferUsageKnown = BufferUsageVertex | BufferUsageIndex | BufferUsageUniform | BufferUsageStorage | BufferUsageIndirect |
	BufferUsageCopySrc | BufferUsageCopyDst | BufferUsageQueryResolve

// Buffer is a block of GPU memory (WebGPU's GPUBuffer). Its bytes are written with Queue.WriteBuffer or a copy and
// read with Read. Release it when it is not needed; the memory is freed when the work that uses it is done.
//
// Flutter: no counterpart
type Buffer struct {
	resource
	exp   explicit.Buffer
	size  int
	usage BufferUsage
	// busy is the number of submissions that use the buffer and are not done.
	busy int
}

// validateBuffer checks a description. It fails with GPUInvalidDescriptor for a size of 0, no use or a use that
// does not exist, and with GPULimitExceeded for a size over Limits.MaxBufferSize.
func (d *Device) validateBuffer(desc BufferDesc) error {
	if desc.Size < 1 {
		return errors.New(TagGPUInvalidDescriptor, "a buffer has at least one byte", "field", "Size", "value", desc.Size, "label", desc.Label)
	}
	if desc.Size > d.limits.MaxBufferSize {
		return errors.New(TagGPULimitExceeded, "the buffer is larger than Limits.MaxBufferSize", "field", "Size", "value", desc.Size, "max", d.limits.MaxBufferSize, "label", desc.Label)
	}
	if desc.Usage == 0 || desc.Usage&^bufferUsageKnown != 0 {
		return errors.New(TagGPUInvalidDescriptor, "a buffer has at least one use, and only the uses that BufferUsage has", "field", "Usage", "value", int(desc.Usage), "label", desc.Label)
	}
	return nil
}

// NewBuffer makes a buffer (WebGPU's GPUDevice.createBuffer). It fails with GPUInvalidDescriptor and
// GPULimitExceeded for a description that cannot be made (the metadata "field" and "value" say which part), with
// GPUDeviceLost on a lost device, and with the error of the backend (GPUOutOfMemory) untouched.
//
// Flutter: no counterpart
func (d *Device) NewBuffer(desc BufferDesc) (*Buffer, error) {
	d.assertThread("NewBuffer")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	if err := d.validateBuffer(desc); err != nil {
		return nil, err
	}
	exp, err := d.dev.NewBuffer(desc.Size, explicit.BufferUsage(desc.Usage))
	if err != nil {
		return nil, err
	}
	return &Buffer{resource: d.track(kindBuffer, desc.Label, 1), exp: exp, size: desc.Size, usage: desc.Usage}, nil
}

// Size returns the size in bytes.
//
// Flutter: no counterpart
func (b *Buffer) Size() int { return b.size }

// Usage returns the uses the buffer was made for.
//
// Flutter: no counterpart
func (b *Buffer) Usage() BufferUsage { return b.usage }

// Release frees the buffer, when the work that uses it is done. It may be called twice. After the device is lost
// it only forgets the buffer.
//
// Flutter: no counterpart
func (b *Buffer) Release() {
	b.d.assertThread("Buffer.Release")
	if !b.untrack() {
		return
	}
	if b.d.lostErr == nil {
		b.exp.Destroy()
	}
}

// Read reads size bytes from offset (WebGPU's GPUBuffer.mapAsync and getMappedRange, in one call). The buffer has
// BufferUsageCopySrc, and the offset and the size are multiples of 4. done runs on the UI thread, from the event
// loop, once the GPU has finished the work submitted before the call; it is never called inside the call or inside
// the frame that asked. data is a copy that done keeps. A read that failed gives err with the tag GPUReadFailed
// (the backend's error is its cause), and a lost device GPUDeviceLost.
//
// Flutter: no counterpart
func (b *Buffer) Read(offset, size int, done func(data []byte, err error)) {
	d := b.d
	d.assertThread("Buffer.Read")
	b.usable("Buffer.Read")
	misuse(b.usage&BufferUsageCopySrc != 0, "Buffer.Read", "the buffer has BufferUsageCopySrc")
	misuse(offset >= 0 && size >= 0 && offset+size <= b.size, "Buffer.Read", "the range is inside the buffer", "offset", offset, "size", size, "buffer", b.size)
	misuse(offset%4 == 0 && size%4 == 0, "Buffer.Read", "the offset and the size are multiples of 4", "offset", offset, "size", size)
	if d.lostErr != nil {
		err := d.lostErr
		d.post(func() { done(nil, err) })
		return
	}
	inCall := true
	finish := func(data []byte, err error) {
		if err != nil && !IsDeviceLost(err) {
			err = errors.NewErr(TagGPUReadFailed, err)
		}
		if inCall {
			d.post(func() { done(data, err) })
			return
		}
		done(data, err)
	}
	err := d.queue.run(func(r *replay) {
		r.markUsed(b)
		r.enc.ReadBuffer(b.exp, offset, size, finish)
	})
	inCall = false
	if err != nil {
		if !IsDeviceLost(err) {
			err = errors.NewErr(TagGPUReadFailed, err)
		}
		d.post(func() { done(nil, err) })
	}
}
