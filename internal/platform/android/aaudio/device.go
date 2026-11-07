//go:build linux

package aaudio

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// device is an open AAudio stream. The data callback runs on a thread of AAudio and only reads atomic values; everything else (pause,
// resume, a disconnect, the app hidden, close) takes mu and is not on the audio thread.
type device struct {
	svc     *Service
	lib     *library
	kind    platform.AudioKind
	handler platform.AudioHandler
	opts    platform.AudioOpenOptions
	// id is the Android audio device that the stream is pinned to, 0 for the default device (which the stream follows: a disconnect
	// makes it again on the new default one).
	id     int32
	handle uintptr
	// format is the format of the frames of the handler, fixed while the device is open: a stream made again asks for the same.
	format platform.AudioFormat

	// The callback's view: whether it hands frames to the handler, and how many calls of the handler are running.
	silent   atomic.Bool
	closing  atomic.Bool
	inflight atomic.Int32
	stream   atomic.Uintptr

	mu         sync.Mutex
	userPaused bool
	bgPaused   bool
	lost       bool
	closed     bool
	// running is whether the stream was asked to start and not to pause since.
	running bool
}

var _ platform.AudioDevice = (*device)(nil)

// Stats is what a device of this backend shows besides platform.AudioDevice: the timestamp of the stream, how often it ran dry and the
// size of its buffer. A test finds it with a type assertion on the device that Open returned.
//
// Flutter: no counterpart
type Stats interface {
	// Timestamp is the time (CLOCK_MONOTONIC, nanoseconds) at which the stream's frame number was; ok is false before the stream has one.
	Timestamp() (frame, nanos int64, ok bool)
	// XRuns is how many times the stream ran out of frames.
	XRuns() int
	// BufferFrames is the stream's buffer and the burst of the hardware, in frames.
	BufferFrames() (buffer, burst int)
}

var _ Stats = (*device)(nil)

// log is the logger of the service.
func (d *device) log() *slog.Logger { return d.svc.log }

// Format returns the format of the frames of the handler.
func (d *device) Format() platform.AudioFormat { return d.format }

// onData is the data callback: it fills the buffer of AAudio or hands the frames it has recorded to the handler.
func (d *device) onData(audio uintptr, frames int32) int32 {
	d.inflight.Add(1)
	defer d.inflight.Add(-1)
	if d.closing.Load() {
		return callbackStop
	}
	buf := unsafe.Slice((*float32)(dl.Pointer(audio)), int(frames)*d.format.Channels)
	if d.kind == platform.AudioPlayback {
		if d.silent.Load() {
			clear(buf)
		} else {
			d.handler.Fill(buf)
		}
	} else if !d.silent.Load() {
		d.handler.Capture(buf)
	}
	return callbackContinue
}

// onError is the error callback: a disconnect (a headset was unplugged, the route changed) is handed to a goroutine that makes the
// stream again; any other error is logged.
func (d *device) onError(stream uintptr, result int32) {
	if stream != d.stream.Load() {
		return // the callback of a stream that was already replaced
	}
	if result != resultDisconnected {
		d.log().Warn("audio: an AAudio stream reported an error", "result", int(result), "text", d.lib.text(result))
		return
	}
	go d.reconnect(stream)
}

// quiesce waits until no call of the handler is running (at most two seconds: a handler that does not return is not waited for).
func (d *device) quiesce() {
	for i := 0; d.inflight.Load() != 0 && i < 2000; i++ {
		time.Sleep(time.Millisecond)
	}
}

// waitState waits (at most stateWait) until the stream is in one of the states.
func (d *device) waitState(stream uintptr, states ...int32) {
	deadline := time.Now().Add(stateWait)
	for time.Now().Before(deadline) {
		cur := d.lib.streamGetState(stream)
		for _, s := range states {
			if cur == s {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
}

// waitStarted waits until the stream says that it started, for at most d.
func (d *device) waitStarted(limit time.Duration) {
	stream := d.stream.Load()
	if stream == 0 {
		return
	}
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if d.lib.streamGetState(stream) == streamStateStarted {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// want says whether the stream should run now.
func (d *device) want() bool { return !d.userPaused && !d.bgPaused && !d.lost && !d.closed }

// applyLocked brings the stream to the state that is wanted: it starts it, or stops it and waits until the handler is not called any
// more. mu is held.
func (d *device) applyLocked() {
	stream := d.stream.Load()
	if stream == 0 {
		return
	}
	switch want := d.want(); {
	case want && !d.running:
		d.silent.Store(false)
		if r := d.lib.streamRequestStart(stream); r != resultOK {
			d.log().Warn("audio: an AAudio stream could not be started", "result", int(r), "text", d.lib.text(r))
			return
		}
		d.running = true
	case !want && d.running:
		d.silent.Store(true)
		d.running = false
		// A playback stream is paused (its buffer is kept, and flushed below so that the sound that was queued does not play after
		// the resume); AAudio does not pause a recording, which is stopped.
		if d.kind == platform.AudioPlayback {
			d.lib.streamRequestPause(stream)
			d.waitState(stream, streamStatePaused, streamStateDisconnected)
			d.lib.streamRequestFlush(stream)
		} else {
			d.lib.streamRequestStop(stream)
			d.waitState(stream, streamStateStopped, streamStateDisconnected)
		}
		d.quiesce()
	}
}

// Pause stops the stream: the handler is not called until Resume. It fails with AudioDeviceLost when the device is gone.
func (d *device) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lost || d.closed {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone")
	}
	d.userPaused = true
	d.applyLocked()
	return nil
}

// Resume starts the stream again, unless the app is hidden and the device pauses then. It fails with AudioDeviceLost when the device
// is gone.
func (d *device) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lost || d.closed {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone")
	}
	d.userPaused = false
	d.applyLocked()
	return nil
}

// setBackground is SetBackground for one device: it pauses while the app is hidden, when it was opened to, and tells the handler (the
// owner did not ask for it).
func (d *device) setBackground(hidden bool) {
	if !d.opts.PauseInBackground {
		return
	}
	d.mu.Lock()
	if d.closed || d.lost || d.bgPaused == hidden {
		d.mu.Unlock()
		return
	}
	d.bgPaused = hidden
	owner := d.userPaused
	d.applyLocked()
	d.mu.Unlock()
	if owner {
		return
	}
	if hidden {
		d.handler.StateChanged(platform.AudioPaused)
	} else {
		d.handler.StateChanged(platform.AudioRunning)
	}
}

// reconnect makes the stream again on the device that is the default now, after a disconnect, in the same format. A device that was
// pinned to an Android audio device, or a default device that cannot be opened again, is lost.
func (d *device) reconnect(old uintptr) {
	d.mu.Lock()
	if d.closed || d.lost || d.stream.Load() != old {
		d.mu.Unlock()
		return
	}
	d.silent.Store(true)
	d.running = false
	d.quiesce()
	d.stream.Store(0)
	d.lib.streamClose(old)
	if d.id != 0 {
		d.loseLocked()
		d.mu.Unlock()
		return
	}
	stream, format, err := d.svc.openStream(d.lib, d, d.format)
	if err != nil || format != d.format {
		if err == nil {
			d.lib.streamClose(stream)
		}
		d.log().Warn("audio: the stream could not move to the new device", "err", err)
		d.loseLocked()
		d.mu.Unlock()
		return
	}
	d.stream.Store(stream)
	d.applyLocked()
	d.mu.Unlock()
}

// lose ends the stream and tells the handler, once.
func (d *device) lose() {
	d.mu.Lock()
	if d.lost || d.closed {
		d.mu.Unlock()
		return
	}
	d.loseLocked()
	d.mu.Unlock()
}

// loseLocked is lose with mu held.
func (d *device) loseLocked() {
	if d.lost {
		return
	}
	d.lost = true
	d.silent.Store(true)
	d.running = false
	if stream := d.stream.Swap(0); stream != 0 {
		d.quiesce()
		d.lib.streamClose(stream)
	}
	d.handler.StateChanged(platform.AudioLost)
}

// Close ends the stream; when it returns the handler is not running and is never called again. It must not be called from the
// handler.
func (d *device) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.closing.Store(true)
	d.silent.Store(true)
	stream := d.stream.Swap(0)
	d.running = false
	d.mu.Unlock()
	if stream != 0 {
		d.lib.streamRequestStop(stream)
		d.waitState(stream, streamStateStopped, streamStatePaused, streamStateDisconnected)
		d.quiesce()
		// Closing joins the thread of the callbacks.
		d.lib.streamClose(stream)
	}
	d.quiesce()
	unregister(d.handle)
	d.svc.forget(d)
	return nil
}

// Timestamp is the time that the stream's frame number was at (CLOCK_MONOTONIC, in nanoseconds), which advances while the stream
// runs; it is how a test sees that the hardware plays. ok is false when the stream has no timestamp yet.
//
// Flutter: no counterpart
func (d *device) Timestamp() (frame, nanos int64, ok bool) {
	stream := d.stream.Load()
	if stream == 0 {
		return 0, 0, false
	}
	scratch := dl.Malloc(16)
	defer dl.Free(scratch)
	if r := d.lib.streamGetTimestamp(stream, clockMonotonic, scratch, scratch+8); r != resultOK {
		return 0, 0, false
	}
	return *(*int64)(dl.Pointer(scratch)), *(*int64)(dl.Pointer(scratch + 8)), true
}

// XRuns is how many times the stream ran out of frames (a playback) or of room (a recording).
//
// Flutter: no counterpart
func (d *device) XRuns() int {
	stream := d.stream.Load()
	if stream == 0 {
		return 0
	}
	if n := d.lib.streamGetXRunCount(stream); n > 0 {
		return int(n)
	}
	return 0
}

// BufferFrames is the size of the stream's buffer in frames, and Burst the size of a burst of the hardware: the latency of a stream
// is about the buffer.
//
// Flutter: no counterpart
func (d *device) BufferFrames() (buffer, burst int) {
	stream := d.stream.Load()
	if stream == 0 {
		return 0, 0
	}
	return int(d.lib.streamGetBufferSize(stream)), int(d.lib.streamGetFramesPerBurst(stream))
}
