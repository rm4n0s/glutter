//go:build darwin

package coreaudio

import (
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// queueBuffers is how many buffers a queue has: while Core Audio plays one, the handler fills another, and a third is ready.
const queueBuffers = 3

// device is an open device: one AudioQueue. Its callbacks run on Core Audio's threads and never take mu, so a Pause, a Close or a
// change of the default device cannot wait for the handler, and the handler cannot wait for them.
type device struct {
	svc    *Service
	lib    *library
	kind   platform.AudioKind
	h      platform.AudioHandler
	opts   platform.AudioOpenOptions
	format platform.AudioFormat
	handle uintptr
	// object is the AudioObjectID of the device when the app named one (the default device has none): its listener says when the
	// device is unplugged.
	object uint32

	// silent makes the output callback give silence and the input callback drop what it got: the device is paused.
	silent atomic.Bool
	closed atomic.Bool

	mu      sync.Mutex
	queue   uintptr
	running bool // what the app wants: running or paused
	// interrupted says that the system's interruption paused the device (and the app did not): it is the one that runs again after it.
	interrupted bool
	lost        bool
	listening   bool
}

var _ platform.AudioDevice = (*device)(nil)

// Format implements platform.AudioDevice.
func (d *device) Format() platform.AudioFormat { return d.format }

// build makes the queue and its buffers. d.mu is held. A failure is AudioFormatUnsupported or AudioDeviceOpenFailed.
func (d *device) build() error {
	sd := NewStreamDescription(d.format)
	var q uintptr
	var status int32
	call := "AudioQueueNewOutput"
	if d.kind == platform.AudioCapture {
		call = "AudioQueueNewInput"
		status = d.lib.newInput(&sd, d.lib.inputCB, d.handle, 0, 0, 0, &q)
	} else {
		status = d.lib.newOutput(&sd, d.lib.outputCB, d.handle, 0, 0, 0, &q)
	}
	if status != 0 {
		return statusError(call, status, true)
	}
	if uid := d.opts.ID; uid != "" && !d.lib.routeOnly {
		s := cf.NewString(uid)
		status = d.lib.setProperty(q, queueCurrentDev, unsafe.Pointer(&s), 8)
		cf.Release(s)
		if status != 0 {
			d.lib.dispose(q, true)
			return statusError("AudioQueueSetProperty", status, true)
		}
	}
	if d.format.Channels > 2 {
		layout := ChannelLayout{Tag: layoutTag(d.format.Channels)}
		if status = d.lib.setProperty(q, queueChannelLay, unsafe.Pointer(&layout), uint32(unsafe.Sizeof(layout))); status != 0 {
			d.lib.dispose(q, true)
			return statusError("AudioQueueSetProperty", status, true)
		}
	}
	size := uint32(bufferFrames(d.opts.BufferFrames, d.format.Rate) * d.format.Channels * 4)
	for range queueBuffers {
		var buf uintptr
		if status = d.lib.allocBuffer(q, size, &buf); status != 0 {
			d.lib.dispose(q, true)
			return statusError("AudioQueueAllocateBuffer", status, true)
		}
		b := (*QueueBuffer)(dl.Pointer(buf))
		// A playback buffer starts as silence (the handler is not called from here, which is not its thread); a capture buffer is empty.
		if d.kind == platform.AudioPlayback {
			clear(unsafe.Slice((*byte)(dl.Pointer(b.Data)), size))
			b.ByteSize = size
		} else {
			b.ByteSize = 0
		}
		if status = d.lib.enqueue(q, buf, 0, 0); status != 0 {
			d.lib.dispose(q, true)
			return statusError("AudioQueueEnqueueBuffer", status, true)
		}
	}
	d.queue = q
	return nil
}

// startLocked starts the queue when the app wants the device to run. d.mu is held; opening says that a failure is an error of
// opening.
func (d *device) startLocked(opening bool) error {
	if !d.running || d.queue == 0 {
		return nil
	}
	d.silent.Store(false)
	if status := d.lib.start(d.queue, 0); status != 0 {
		return statusError("AudioQueueStart", status, opening)
	}
	return nil
}

// teardownLocked stops and disposes the queue: when it returns no callback of the queue runs. d.mu is held.
func (d *device) teardownLocked() {
	if d.queue == 0 {
		return
	}
	q := d.queue
	d.queue = 0
	d.silent.Store(true)
	d.lib.stop(q, true)
	d.lib.dispose(q, true)
}

// onOutput is the callback of a playback queue: it fills the buffer that Core Audio gives back, or gives silence when the device
// is paused, and queues it again.
func (d *device) onOutput(queue, buf uintptr) {
	if d.closed.Load() {
		return
	}
	b := (*QueueBuffer)(dl.Pointer(buf))
	frames := unsafe.Slice((*float32)(dl.Pointer(b.Data)), b.Capacity/4)
	if d.silent.Load() {
		clear(frames)
	} else {
		d.call(func() { d.h.Fill(frames) }, func() { clear(frames) })
	}
	b.ByteSize = b.Capacity
	d.lib.enqueue(queue, buf, 0, 0)
}

// onInput is the callback of a capture queue: it gives what was recorded to the handler and queues the buffer again.
func (d *device) onInput(queue, buf uintptr) {
	if d.closed.Load() {
		return
	}
	b := (*QueueBuffer)(dl.Pointer(buf))
	if n := b.ByteSize / 4; n > 0 && !d.silent.Load() {
		frames := unsafe.Slice((*float32)(dl.Pointer(b.Data)), n)
		d.call(func() { d.h.Capture(frames) }, nil)
	}
	b.ByteSize = 0
	d.lib.enqueue(queue, buf, 0, 0)
}

// call runs a function of the handler. A panic in it must not unwind into Core Audio, which would end the process: it is logged
// with its stack and the device gives silence.
func (d *device) call(fn func(), recovered func()) {
	defer func() {
		if v := recover(); v != nil {
			d.svc.log.Error("the audio handler panicked", "panic", v, "stack", string(debug.Stack()))
			if recovered != nil {
				recovered()
			}
		}
	}()
	fn()
}

// onProperty is the listener of the device that was named: when it is not alive any more it is lost.
func (d *device) onProperty(obj uint32, selectors []uint32) {
	for _, sel := range selectors {
		if sel != selIsAlive {
			continue
		}
		if alive, ok := d.lib.uint32Prop(obj, PropertyAddress{Selector: selIsAlive, Scope: scopeGlobal}); ok && alive == 0 {
			d.markLost()
		}
	}
}

// markLost ends the queue of a device that is gone, and tells the handler once.
func (d *device) markLost() {
	d.mu.Lock()
	if d.lost || d.closed.Load() {
		d.mu.Unlock()
		return
	}
	d.lost = true
	d.teardownLocked()
	d.mu.Unlock()
	d.h.StateChanged(platform.AudioLost)
}

// follow moves a device that was opened on the default device to the new default: the queue is made again on it.
func (d *device) follow() {
	d.mu.Lock()
	if d.lost || d.closed.Load() {
		d.mu.Unlock()
		return
	}
	d.teardownLocked()
	err := d.build()
	if err == nil {
		err = d.startLocked(false)
	}
	if err != nil {
		d.lost = true
		d.teardownLocked()
	}
	d.mu.Unlock()
	if err != nil {
		d.svc.log.Error("the audio device could not follow the default device", "err", err)
		d.h.StateChanged(platform.AudioLost)
	}
}

// Pause implements platform.AudioDevice.
func (d *device) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.usable(); err != nil {
		return err
	}
	d.running = false
	d.silent.Store(true)
	if status := d.lib.pause(d.queue); status != 0 {
		return statusError("AudioQueuePause", status, false)
	}
	d.h.StateChanged(platform.AudioPaused)
	return nil
}

// Resume implements platform.AudioDevice.
func (d *device) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.usable(); err != nil {
		return err
	}
	d.running = true
	if err := d.startLocked(false); err != nil {
		return err
	}
	d.h.StateChanged(platform.AudioRunning)
	return nil
}

// wasInterrupted says that the system's interruption paused the device.
func (d *device) wasInterrupted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.interrupted
}

// interrupt is the system's interruption beginning or ending for this device (Service.Interrupt). A device that is lost or closed does
// nothing; one that the app paused is not paused again and not resumed.
func (d *device) interrupt(began, resume bool) {
	d.mu.Lock()
	if d.lost || d.closed.Load() || d.queue == 0 {
		d.mu.Unlock()
		return
	}
	if began {
		if !d.running {
			d.mu.Unlock()
			return
		}
		d.interrupted, d.running = true, false
		d.silent.Store(true)
		d.lib.pause(d.queue)
		d.mu.Unlock()
		d.h.StateChanged(platform.AudioPaused)
		return
	}
	if !d.interrupted {
		d.mu.Unlock()
		return
	}
	d.interrupted = false
	if !resume {
		d.mu.Unlock()
		return
	}
	d.running = true
	err := d.startLocked(false)
	d.mu.Unlock()
	if err != nil {
		d.svc.log.Error("the audio device could not run again after the interruption", "err", err)
		return
	}
	d.h.StateChanged(platform.AudioRunning)
}

// usable says whether the device can be paused or resumed: one that is lost or closed cannot. d.mu is held.
func (d *device) usable() error {
	if d.closed.Load() {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is closed", "id", d.opts.ID)
	}
	if d.lost || d.queue == 0 {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone", "id", d.opts.ID)
	}
	return nil
}

// Close implements platform.AudioDevice: the queue is stopped before it returns, so the handler does not run after it.
func (d *device) Close() error {
	if d.closed.Swap(true) {
		return nil
	}
	// The listener goes first and without the lock: it takes the lock itself when the device is lost.
	if d.listening {
		addr := PropertyAddress{Selector: selIsAlive, Scope: scopeGlobal}
		d.lib.removeListener(d.object, &addr, d.lib.listenerCB, d.handle)
	}
	d.mu.Lock()
	d.teardownLocked()
	d.mu.Unlock()
	unregister(d.handle)
	d.svc.forget(d)
	return nil
}
