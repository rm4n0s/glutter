//go:build windows

package d3d12

import (
	"image"
	"image/color"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// pendingRead is a ReadTexture that waits for the frame to finish: a readback buffer with the layout
// of the rectangle in it.
type pendingRead struct {
	res    win32.Object // zero for an empty rectangle
	layout copyLayout
	rect   image.Rectangle
	format explicit.TextureFormat
	done   func(*image.RGBA, error)
}

// encoder records the commands of one frame.
type encoder struct {
	d     *Device
	slot  int
	list  win32.Object
	pass  *pass
	reads []*pendingRead

	// The state of the command list, which lasts from pass to pass (the state of a Direct3D 12 command
	// list does): the root signature and the pipeline state that were set, the groups that were bound
	// to the root signature, and the shader-visible heaps that were set.
	root        *rootSig
	pso         win32.Object
	bound       [maxBindGroups]boundGroup
	viewHeap    int // the block of the resource heap that was set, -1 for none
	samplerHeap int
}

var _ explicit.Encoder = (*encoder)(nil)

// Begin implements explicit.Device: it waits for the slot of the frame if the GPU is still using it,
// resets its allocator and list, and records what was queued since the last frame (uploads). It fails
// with D3D12FrameOpen when the previous frame was not submitted, and with GPUDeviceLost when the
// device was removed.
func (d *Device) Begin() (explicit.Encoder, error) {
	if d.open != nil {
		return nil, errors.New(tagD3D12FrameOpen, "Begin was called before the frame in progress was submitted")
	}
	if err := d.checkRemoved(); err != nil {
		return nil, err
	}
	if err := d.prepare(d.next); err != nil {
		return nil, err
	}
	s := &d.slots[d.next]
	if err := d.hr(s.alloc, "ID3D12CommandAllocator::Reset", allocatorReset); err != nil {
		return nil, err
	}
	if err := d.hr(s.list, "ID3D12GraphicsCommandList::Reset", listReset, s.alloc.Ptr(), 0); err != nil {
		return nil, err
	}
	s.state = slotRecording
	e := &encoder{d: d, slot: d.next, list: s.list, viewHeap: -1, samplerHeap: -1}
	d.open = e
	pending := d.pending
	d.pending = nil
	for _, fn := range pending {
		fn(e.list)
	}
	return e, nil
}

// CopyTexture implements explicit.Encoder: the barriers (the source in COPY_SOURCE, the destination
// in COPY_DEST) and CopyTextureRegion. The same subresource as source and destination is a
// D3D12StateConflict, a programmer error.
func (e *encoder) CopyTexture(src explicit.Texture, r image.Rectangle, dst explicit.Texture, at image.Point) {
	d := e.d
	s, t := d.own2texture(src), d.own2texture(dst)
	if e.pass != nil {
		panic(errors.New(tagD3D12PassNotEnded, "CopyTexture was called before the pass ended"))
	}
	if r.Empty() {
		return
	}
	dstRect := image.Rectangle{Min: at, Max: at.Add(r.Size())}
	if !r.In(image.Rect(0, 0, s.desc.Width, s.desc.Height)) || !dstRect.In(image.Rect(0, 0, t.desc.Width, t.desc.Height)) {
		panic(errors.New(tagD3D12RectOutOfRange, "the copy is outside a texture", "op", "CopyTexture",
			"src_rect", r.String(), "dst_rect", dstRect.String(), "src_width", s.desc.Width, "src_height", s.desc.Height,
			"dst_width", t.desc.Width, "dst_height", t.desc.Height))
	}
	d.tracker.begin()
	defer d.tracker.end()
	if err := d.tracker.need(s.root, s.base, stateCopySource); err != nil {
		panic(err)
	}
	if err := d.tracker.need(t.root, t.base, stateCopyDest); err != nil {
		panic(err)
	}
	d.barrier(e.list)
	srcLoc := d3d12TextureCopyIndex{Resource: s.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(s.base)}
	dstLoc := d3d12TextureCopyIndex{Resource: t.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(t.base)}
	box := d3d12Box{Left: uint32(r.Min.X), Top: uint32(r.Min.Y), Right: uint32(r.Max.X), Bottom: uint32(r.Max.Y), Back: 1}
	e.list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), uintptr(at.X), uintptr(at.Y), 0, uintptr(unsafe.Pointer(&srcLoc)), uintptr(unsafe.Pointer(&box)))
}

// ReadTexture implements explicit.Encoder: it copies the rectangle into a readback buffer, and done
// runs at the end of Submit, after the GPU finished (SPEC §9.8).
func (e *encoder) ReadTexture(src explicit.Texture, r image.Rectangle, done func(*image.RGBA, error)) {
	d := e.d
	s := d.own2texture(src)
	if e.pass != nil {
		panic(errors.New(tagD3D12PassNotEnded, "ReadTexture was called before the pass ended"))
	}
	if !r.In(image.Rect(0, 0, s.desc.Width, s.desc.Height)) {
		panic(errors.New(tagD3D12RectOutOfRange, "the read is outside the texture", "op", "ReadTexture",
			"rect", r.String(), "width", s.desc.Width, "height", s.desc.Height))
	}
	rd := &pendingRead{rect: r, format: s.desc.Format, done: done}
	if r.Empty() {
		e.reads = append(e.reads, rd)
		return
	}
	rd.layout = layoutOf(d.limits, r.Dx(), r.Dy(), s.desc.Format.BytesPerPixel())
	res, err := d.createBuffer(heapTypeReadback, rd.layout.Size)
	if err != nil {
		done(nil, err)
		return
	}
	rd.res = res
	d.tracker.begin()
	defer d.tracker.end()
	if err := d.tracker.need(s.root, s.base, stateCopySource); err != nil {
		panic(err)
	}
	d.barrier(e.list)
	dstLoc := d3d12TextureCopyLocation{Resource: res.Ptr(), Type: textureCopyPlacedFootprint, Placed: d3d12PlacedFootprint{
		Footprint: d3d12SubresourceFootprint{
			Format: dxgiFormat(s.desc.Format), Width: uint32(r.Dx()), Height: uint32(r.Dy()), Depth: 1, RowPitch: uint32(rd.layout.RowPitch),
		},
	}}
	srcLoc := d3d12TextureCopyIndex{Resource: s.res.Ptr(), Type: textureCopySubresourceIndex, Index: uint32(s.base)}
	box := d3d12Box{Left: uint32(r.Min.X), Top: uint32(r.Min.Y), Right: uint32(r.Max.X), Bottom: uint32(r.Max.Y), Back: 1}
	e.list.Call(listCopyTextureRegion, uintptr(unsafe.Pointer(&dstLoc)), 0, 0, 0, uintptr(unsafe.Pointer(&srcLoc)), uintptr(unsafe.Pointer(&box)))
	e.reads = append(e.reads, rd)
}

// Submit implements explicit.Encoder: it closes the list, executes it on the queue and signals the
// fence. When the frame asked for a texture to be read, it waits for the GPU and runs the done
// functions before it returns.
func (e *encoder) Submit() error {
	d := e.d
	if d.open != e {
		return errors.New(tagD3D12FrameNotOpen, "Submit was called on a frame that is not the one in progress")
	}
	if e.pass != nil {
		panic(errors.New(tagD3D12PassNotEnded, "Submit was called before the pass ended"))
	}
	s := &d.slots[e.slot]
	d.open = nil
	e.presentBarrier()
	if err := d.hr(e.list, "ID3D12GraphicsCommandList::Close", listClose); err != nil {
		return d.failReads(e, err)
	}
	lists := [1]uintptr{e.list.Ptr()}
	d.queue.Call(queueExecuteCommandLists, 1, uintptr(unsafe.Pointer(&lists[0])))
	d.fenceValue++
	if err := d.hr(d.queue, "ID3D12CommandQueue::Signal", queueSignal, d.fence.Ptr(), uintptr(d.fenceValue)); err != nil {
		return d.failReads(e, err)
	}
	s.fence = d.fenceValue
	s.state = slotInflight
	d.last = e.slot
	d.next = (e.slot + 1) % framesInFlight
	d.drainDebugMessages()
	if len(e.reads) == 0 {
		return d.checkRemoved()
	}
	if err := d.waitFor(s.fence); err != nil {
		return d.failReads(e, err)
	}
	// The fence value is reached, so the slot is ready again; prepare would wait on it for nothing.
	d.recycle(e.slot)
	for _, rd := range e.reads {
		d.finishRead(rd)
	}
	return nil
}

// failReads tells each ReadTexture of a frame that failed, and returns the error.
func (d *Device) failReads(e *encoder, err error) error {
	for _, rd := range e.reads {
		if !rd.res.IsNil() {
			d.releaseBuffer(rd.res, rd.layout.Size)
		}
		rd.done(nil, err)
	}
	e.reads = nil
	return err
}

// finishRead converts what the GPU copied into the readback buffer (BGRA to RGBA, R to a gray pixel as
// the other backends do), calls done, and frees the buffer.
func (d *Device) finishRead(rd *pendingRead) {
	w, h := rd.rect.Dx(), rd.rect.Dy()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if rd.res.IsNil() {
		rd.done(img, nil)
		return
	}
	ptr, err := d.mapBuffer(rd.res, true, rd.layout.Size)
	if err != nil {
		d.releaseBuffer(rd.res, rd.layout.Size)
		rd.done(nil, err)
		return
	}
	raw := win32.Bytes(ptr, rd.layout.Size)
	bpp := rd.format.BytesPerPixel()
	for y := 0; y < h; y++ {
		row := raw[y*rd.layout.RowPitch:]
		for x := 0; x < w; x++ {
			var c color.RGBA
			switch rd.format {
			case explicit.TextureFormatBGRA8Unorm:
				c = color.RGBA{row[x*4+2], row[x*4+1], row[x*4], row[x*4+3]}
			case explicit.TextureFormatR8Unorm:
				c = color.RGBA{row[x], row[x], row[x], 255}
			default:
				c = color.RGBA{row[x*bpp], row[x*bpp+1], row[x*bpp+2], row[x*bpp+3]}
			}
			img.SetRGBA(x, y, c)
		}
	}
	unmapBuffer(rd.res, 0)
	d.releaseBuffer(rd.res, rd.layout.Size)
	rd.done(img, nil)
}

// drainDebugMessages reads the messages of the debug layer's info queue after a frame, logs the
// errors and keeps them for DebugErrors. Without the layer there is no queue and nothing to read.
func (d *Device) drainDebugMessages() {
	if d.infoQueue.IsNil() {
		return
	}
	n := uint64(d.infoQueue.Call(infoQueueGetNumStoredMessages))
	for i := uint64(0); i < n; i++ {
		var size uintptr
		if hr := win32.FromResult(d.infoQueue.Call(infoQueueGetMessage, uintptr(i), 0, uintptr(unsafe.Pointer(&size)))); hr.Failed() || size == 0 {
			continue
		}
		buf := make([]byte, size)
		if hr := win32.FromResult(d.infoQueue.Call(infoQueueGetMessage, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))); hr.Failed() {
			continue
		}
		if m, ok := parseDebugMessage(buf); ok && m.isError() {
			d.debugErrors = append(d.debugErrors, m.Text)
			d.log.Error("Direct3D debug layer", "severity", m.Severity, "id", m.ID, "message", m.Text)
		}
	}
	d.infoQueue.Call(infoQueueClearStoredMessages)
}

// openInfoQueue asks the device for the info queue of the debug layer; the layer may be on but the
// queue missing, which is logged and ignored.
func (d *Device) openInfoQueue() {
	q, err := d.dev.QueryInterface(&iidInfoQueue)
	if err != nil {
		d.log.Warn("the Direct3D debug layer has no info queue; its messages are not read", "err", err)
		return
	}
	d.infoQueue = d.own(q)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagD3D12FrameOpen is the tag of the programmer error of Begin before the frame in progress was
	// submitted.
	tagD3D12FrameOpen = "D3D12FrameOpen"
	// tagD3D12FrameNotOpen is the tag of the programmer error of Submit on a frame that is not the one
	// in progress.
	tagD3D12FrameNotOpen = "D3D12FrameNotOpen"
	// tagD3D12PassNotEnded is the tag of the programmer error of a pass, a copy, a read or a Submit
	// while a pass is open; the metadata do not say more.
	tagD3D12PassNotEnded = "D3D12PassNotEnded"
	// tagD3D12RectOutOfRange is the tag of the programmer error of a copy or a read of a rectangle that
	// is not inside the texture; the metadata are "op" and the rectangles and sizes.
	tagD3D12RectOutOfRange = "D3D12RectOutOfRange"
)
