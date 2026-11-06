package gpu

import (
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Queue is the queue of a device (WebGPU's GPUQueue): the commands of the command buffers that are submitted run on
// the GPU in the order they were submitted, and the writes take effect in that order too.
//
// Each call to Submit is one submission of the explicit device (one Begin and one Submit, SPEC §9.3), and the device
// has FramesInFlight (2) of them in flight with the frame of the renderer: submit once for each frame, with all the
// command buffers of the frame, rather than once for each of them.
//
// Flutter: no counterpart
type Queue struct {
	d *Device
	// pending are the writes to buffers that the GPU was still using when they were made: they are done by the next
	// submission, at the start of its commands.
	pending []pendingWrite
	// The storage that a frame uses again, so that a steady frame allocates nothing: the replays that are not in flight, and
	// the encoders and the passes of the command buffers that were submitted.
	replays  []*replay
	encoders []*CommandEncoder
	renders  []*RenderPass
	computes []*ComputePass
	// stamp counts the uses of replays (replay.stamp).
	stamp uint64
}

// pendingWrite is a write that waits for the next submission: staging holds the bytes.
type pendingWrite struct {
	dst     *Buffer
	offset  int
	size    int
	staging explicit.Buffer
}

// replay is the state of one submission while the commands of its command buffers run on the explicit encoder. A replay
// goes back to the queue's list when the GPU has finished the submission (finish), and is made again from it.
type replay struct {
	d   *Device
	q   *Queue
	enc explicit.Encoder
	// groups are the bind groups made for this submission, by bind group and the pipeline they are bound with.
	groups map[groupKey]explicit.BindGroup
	// used are the buffers that the commands use, which are busy until the submission is done.
	used []*Buffer
	// stamp names this use of the replay: a buffer whose stamp is the same has been counted already (no set to look it up in).
	stamp    uint64
	released bool
	// err is the first error of the commands: a bind group that could not be made.
	err error
	// finishFn is finish as a value, made once, so that handing it to the encoder allocates nothing.
	finishFn func()
}

type groupKey struct {
	bg       *BindGroup
	pipeline any
}

// markUsed counts the buffer as busy until the submission is done.
func (r *replay) markUsed(b *Buffer) {
	if b.stamp == r.stamp {
		return
	}
	b.stamp = r.stamp
	r.used = append(r.used, b)
	b.busy++
}

// finish ends the busy time of the buffers and gives the replay back to the queue; it may be called twice.
func (r *replay) finish() {
	if r.released {
		return
	}
	r.released = true
	for _, b := range r.used {
		b.busy--
	}
	clear(r.used)
	r.used = r.used[:0]
	clear(r.groups)
	r.enc, r.err = nil, nil
	r.q.replays = append(r.q.replays, r)
}

// takeReplay is a replay from the queue's list, or a new one.
func (q *Queue) takeReplay(enc explicit.Encoder) *replay {
	var r *replay
	if n := len(q.replays); n > 0 {
		r = q.replays[n-1]
		q.replays[n-1] = nil
		q.replays = q.replays[:n-1]
	} else {
		r = &replay{d: q.d, q: q, groups: map[groupKey]explicit.BindGroup{}}
		r.finishFn = r.finish
	}
	q.stamp++
	r.enc, r.released, r.stamp = enc, false, q.stamp
	return r
}

// run makes one submission: the pending writes, then the commands of the command buffers, then fn's. It returns the error of
// the explicit device (Begin and Submit), untouched.
func (q *Queue) run(cbs []*CommandBuffer, fn func(r *replay)) error {
	d := q.d
	enc, err := d.dev.Begin()
	if err != nil {
		return err
	}
	r := q.takeReplay(enc)
	q.flushPending(r)
	for _, cb := range cbs {
		cb.replay(r)
	}
	if fn != nil {
		fn(r)
	}
	// What the commands could not do is read before the submission: it may be finished (and the replay used again) inside Submit.
	cmdErr := r.err
	enc.OnComplete(r.finishFn)
	if err := enc.Submit(); err != nil {
		r.finish()
		return err
	}
	return cmdErr
}

// flushPending copies the staged writes into their buffers, in the order they were made.
func (q *Queue) flushPending(r *replay) {
	for i, w := range q.pending {
		if !w.dst.released {
			r.markUsed(w.dst)
			r.enc.CopyBuffer(w.staging, 0, w.dst.exp, w.offset, w.size)
		}
		w.staging.Destroy()
		q.pending[i] = pendingWrite{}
	}
	q.pending = q.pending[:0]
}

// WriteBuffer writes data into the buffer at offset (WebGPU's GPUQueue.writeBuffer). The data is copied before the
// call returns; the write happens, for the GPU, between the submissions before it and the ones after. The buffer has
// BufferUsageCopyDst, and the offset and the length are multiples of 4.
//
// Flutter: no counterpart
func (q *Queue) WriteBuffer(b *Buffer, offset int, data []byte) {
	d := q.d
	d.assertThread("Queue.WriteBuffer")
	if len(data) == 0 || d.lostErr != nil {
		return
	}
	misuse(b != nil && b.d == d, "Queue.WriteBuffer", "the buffer was made by this device")
	b.usable("Queue.WriteBuffer")
	misuse(b.usage&BufferUsageCopyDst != 0, "Queue.WriteBuffer", "the buffer has BufferUsageCopyDst")
	// The metadata is made only for a write that is wrong: a frame writes its instances here, and boxing the numbers of a good
	// one would allocate.
	if foundation.DebugMode && (offset < 0 || offset%4 != 0 || len(data)%4 != 0) {
		misuse(false, "Queue.WriteBuffer", "the offset and the length of a write are multiples of 4", "offset", offset, "length", len(data))
	}
	if foundation.DebugMode && (offset+len(data) > b.size) {
		misuse(false, "Queue.WriteBuffer", "the write is inside the buffer", "offset", offset, "length", len(data), "size", b.size)
	}
	if b.busy == 0 {
		d.dev.WriteBuffer(b.exp, offset, data)
		return
	}
	// The GPU still uses the buffer: the bytes go to a buffer of their own, and the next submission copies them.
	staging, err := d.dev.NewBuffer(len(data), explicit.BufferCopySource)
	if err != nil {
		d.logError("a write to a buffer could not be staged", err)
		return
	}
	d.dev.WriteBuffer(staging, 0, data)
	q.pending = append(q.pending, pendingWrite{dst: b, offset: offset, size: len(data), staging: staging})
}

// WriteTexture writes data into a box of a texture (WebGPU's GPUQueue.writeTexture). The data is copied before the
// call returns, and the write happens in the order of the submissions. layout says how the texels are laid out in
// data; unlike a copy from a buffer, its BytesPerRow has no alignment. The texture is not a depth texture or
// multisampled.
//
// Flutter: no counterpart
func (q *Queue) WriteTexture(dst TextureCopy, data []byte, layout DataLayout, size Extent3D) {
	d := q.d
	d.assertThread("Queue.WriteTexture")
	if d.lostErr != nil || size.Width == 0 || size.Height == 0 || size.DepthOrArrayLayers == 0 {
		return
	}
	t := dst.Texture
	misuse(t != nil && t.d == d, "Queue.WriteTexture", "the texture was made by this device")
	t.usable("Queue.WriteTexture")
	region := explicit.TextureRegion{Level: dst.MipLevel, Origin: [3]int{dst.Origin.X, dst.Origin.Y, dst.Origin.Z}, Size: [3]int{size.Width, size.Height, size.DepthOrArrayLayers}}
	bpp := t.desc.Format.BytesPerPixel()
	rowsPerImage := layout.RowsPerImage
	if rowsPerImage == 0 {
		rowsPerImage = size.Height
	}
	bytesPerRow := layout.BytesPerRow
	if bytesPerRow == 0 {
		bytesPerRow = size.Width * bpp
	}
	misuse(!t.desc.Format.IsDepthStencil() && t.desc.SampleCount == 1, "Queue.WriteTexture", "the texture is not a depth texture and not multisampled")
	misuse(t.boxInside(region), "Queue.WriteTexture", "the box is inside the mip level", "level", dst.MipLevel)
	misuse(bytesPerRow >= size.Width*bpp && rowsPerImage >= size.Height, "Queue.WriteTexture", "the rows of the layout hold the box",
		"bytes_per_row", bytesPerRow, "rows_per_image", rowsPerImage)
	misuse(layout.Offset >= 0 && layout.Offset <= len(data), "Queue.WriteTexture", "the layout starts inside the data", "offset", layout.Offset, "length", len(data))
	need := bytesPerRow*(size.Height-1) + size.Width*bpp + (size.DepthOrArrayLayers-1)*bytesPerRow*rowsPerImage
	misuse(len(data)-layout.Offset >= need, "Queue.WriteTexture", "the data holds the box", "need", need, "have", len(data)-layout.Offset)
	d.dev.WriteTextureRegion(t.exp, region, data[layout.Offset:], bytesPerRow, rowsPerImage)
}

// Submit sends the command buffers to the GPU, in order, as one submission (WebGPU's GPUQueue.submit). A command
// buffer is submitted once. It fails with GPUDeviceLost for a lost device, and with the errors of the backend (such
// as GPUOutOfMemory) untouched.
//
// Flutter: no counterpart
func (q *Queue) Submit(buffers ...*CommandBuffer) error {
	d := q.d
	d.assertThread("Queue.Submit")
	if d.lostErr != nil {
		return d.lostErr
	}
	for _, cb := range buffers {
		misuse(cb != nil && cb.d == d, "Queue.Submit", "the command buffer was made by this device")
		misuse(!cb.submitted, "Queue.Submit", "a command buffer is submitted once")
		cb.submitted = true
	}
	err := q.run(buffers, nil)
	// The commands have run on the explicit encoder: the encoders and passes can be used again.
	for _, cb := range buffers {
		cb.e.recycle()
	}
	return err
}

// OnSubmittedWorkDone calls fn, on the UI thread from the event loop, once the GPU has finished everything that was
// submitted before this call (WebGPU's GPUQueue.onSubmittedWorkDone). It is never called inside the call.
//
// Flutter: no counterpart
func (q *Queue) OnSubmittedWorkDone(fn func()) {
	d := q.d
	d.assertThread("Queue.OnSubmittedWorkDone")
	if d.lostErr != nil {
		d.post(fn)
		return
	}
	inCall := true
	err := q.run(nil, func(r *replay) {
		r.enc.OnComplete(func() {
			if inCall {
				d.post(fn)
				return
			}
			fn()
		})
	})
	inCall = false
	if err != nil {
		d.logError("the queue could not wait for the submitted work", err)
		d.post(fn)
	}
}

// post runs fn from the event loop.
func (d *Device) post(fn func()) { d.st.b.PlatformDispatcher().Post(fn) }

// logError logs an error of a call that has no error to return.
func (d *Device) logError(msg string, err error) {
	log := slog.Default()
	if d.st.host != nil {
		if l := d.st.host.GPULogger(); l != nil {
			log = l
		}
	}
	log.Error(msg, "err", err)
}

// misuse panics with GPUEncodingInvalid in debug builds when ok is false: a programmer error of the use of the API
// (SPEC §27.2). The release builds skip the check. what names the rule, call the entry point, and kv are metadata.
//
// A check that has metadata (or a message that is made at run time) is written
// `if foundation.DebugMode && !(ok) { misuse(false, call, what, kv...) }` in the code that a frame runs: the arguments are made
// only for a check that failed, as boxing the numbers of one that passed is an allocation, and a steady frame allocates nothing.
func misuse(ok bool, call, what string, kv ...any) {
	if foundation.DebugMode && !ok {
		panic(errors.New(TagGPUEncodingInvalid, "gpu: "+call+": "+what, append([]any{"call", call, "what", what}, kv...)...))
	}
}
