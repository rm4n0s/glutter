//go:build windows

package d3d12

import (
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// fenceSlice is how long a wait for the fence sleeps before it asks the device whether it was
// removed: a device that is removed makes SetEventOnCompletion signal at once on Windows, and the
// question costs nothing, but a driver that hangs without saying so would otherwise be waited for for
// ever.
const fenceSlice = time.Second

type slotState int

const (
	slotReady     slotState = iota // its fence value is reached and its staging half is free
	slotRecording                  // an encoder is recording into it
	slotInflight                   // submitted, the GPU may still be working
)

// frameSlot is what one frame in flight owns: a command allocator and list, the fence value that
// says its frame is done, and what is released when the GPU is done (SPEC §9.8 "Synchronization").
type frameSlot struct {
	state    slotState
	alloc    win32.Object
	list     win32.Object
	fence    uint64
	deferred []func()
}

// runDeferred runs and forgets what was waiting for the GPU.
func (s *frameSlot) runDeferred() {
	fns := s.deferred
	s.deferred = nil
	for _, fn := range fns {
		fn()
	}
}

// createFence makes the one fence of the queue and the event that waits for it.
func (d *Device) createFence() error {
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateFence", deviceCreateFence, 0, 0,
		uintptr(unsafe.Pointer(&iidFence)), uintptr(unsafe.Pointer(&out))); err != nil {
		return err
	}
	d.fence = d.own(win32.ObjectFromPtr(out))
	ev, err := win32.CreateEvent(false, false)
	if err != nil {
		return err
	}
	d.fenceEvent = ev
	return nil
}

// createSlots makes the command allocator and the command list of each frame slot; the list is
// closed, Begin resets it.
func (d *Device) createSlots() error {
	for i := range d.slots {
		s := &d.slots[i]
		var out uintptr
		if err := d.hr(d.dev, "ID3D12Device::CreateCommandAllocator", deviceCreateCommandAllocator, uintptr(commandListDirect),
			uintptr(unsafe.Pointer(&iidCommandAllocator)), uintptr(unsafe.Pointer(&out))); err != nil {
			return err
		}
		s.alloc = d.own(win32.ObjectFromPtr(out))
		out = 0
		if err := d.hr(d.dev, "ID3D12Device::CreateCommandList", deviceCreateCommandList, 0, uintptr(commandListDirect), s.alloc.Ptr(), 0,
			uintptr(unsafe.Pointer(&iidGraphicsCommandList)), uintptr(unsafe.Pointer(&out))); err != nil {
			return err
		}
		s.list = d.own(win32.ObjectFromPtr(out))
		if err := d.hr(s.list, "ID3D12GraphicsCommandList::Close", listClose); err != nil {
			return err
		}
	}
	return nil
}

// waitFor waits until the fence reaches the value, then asks whether the device was removed (a
// removed device makes GetCompletedValue report the largest value). The UI thread waits only when
// the GPU is two frames behind (SPEC §9.8).
func (d *Device) waitFor(value uint64) error {
	if uint64(d.fence.Call(fenceGetCompletedValue)) < value {
		if err := d.hr(d.fence, "ID3D12Fence::SetEventOnCompletion", fenceSetEventOnComplete, uintptr(value), d.fenceEvent); err != nil {
			return err
		}
		for {
			res, err := win32.WaitForSingleObject(d.fenceEvent, fenceSlice)
			if err != nil {
				return err
			}
			if res == win32.WaitObject0 {
				break
			}
			if err := d.checkRemoved(); err != nil {
				return err
			}
		}
	}
	return d.checkRemoved()
}

// waitIdle waits for a signal of the queue made now, which comes after every frame submitted, and
// makes the slots of those frames ready: the GPU is idle, so what waited for it is released.
func (d *Device) waitIdle() error {
	if d.queue.IsNil() || d.fence.IsNil() {
		return nil
	}
	d.fenceValue++
	if err := d.hr(d.queue, "ID3D12CommandQueue::Signal", queueSignal, d.fence.Ptr(), uintptr(d.fenceValue)); err != nil {
		return err
	}
	if err := d.waitFor(d.fenceValue); err != nil {
		return err
	}
	for i := range d.slots {
		if d.slots[i].state == slotInflight {
			d.recycle(i)
		}
	}
	return nil
}

// prepare makes the slot ready for a frame: if the GPU may be using it, it waits for its fence value,
// then it runs what was waiting for the GPU and frees the slot's half of the staging ring.
func (d *Device) prepare(index int) error {
	s := &d.slots[index]
	if s.state == slotReady || s.state == slotRecording {
		return nil
	}
	if err := d.waitFor(s.fence); err != nil {
		return err
	}
	d.recycle(index)
	return nil
}

// recycle makes a slot whose fence value is reached ready for a frame. The queue runs the frames in
// order, so when the value of a frame is reached, so are the frames before it, and what they used can
// be released.
func (d *Device) recycle(index int) {
	s := &d.slots[index]
	s.runDeferred()
	d.staging.ring.reset(index)
	s.state = slotReady
}

// retire runs fn when the GPU is done with what the frames that exist use: at once when none is,
// else when the fence of the frame being recorded or the last one submitted is reached (the queue runs
// frames in order). While commands wait for a frame (a write to a texture made between frames, or
// inside a pass), fn waits for the frame that will run them, since the commands use what fn releases.
func (d *Device) retire(fn func()) {
	switch {
	case len(d.pending) > 0:
		s := &d.slots[d.commandSlot()]
		s.deferred = append(s.deferred, fn)
	case d.open != nil:
		s := &d.slots[d.open.slot]
		s.deferred = append(s.deferred, fn)
	case d.last >= 0 && d.slots[d.last].state == slotInflight:
		s := &d.slots[d.last]
		s.deferred = append(s.deferred, fn)
	default:
		fn()
	}
}

// commandSlot is the slot of the frame that runs the commands queued now (inCommands): the frame being
// recorded when it is outside a pass, the frame after it when a pass is open (the commands wait for
// the next Begin), and the next frame when none is open.
func (d *Device) commandSlot() int {
	switch {
	case d.open == nil:
		return d.next
	case d.open.pass == nil:
		return d.open.slot
	}
	return (d.open.slot + 1) % framesInFlight
}

// retireWithNextFrame runs fn when the GPU is done with the frame that the commands being queued
// now belong to. A one-off staging buffer waits for the frame that copies from it.
func (d *Device) retireWithNextFrame(fn func()) {
	s := &d.slots[d.commandSlot()]
	s.deferred = append(s.deferred, fn)
}

// inCommands runs fn with the command list: the frame being recorded when it is outside a pass, else
// the next frame's, at its start.
func (d *Device) inCommands(fn func(list win32.Object)) {
	if d.open != nil && d.open.pass == nil {
		fn(d.slots[d.open.slot].list)
		return
	}
	d.pending = append(d.pending, fn)
}
