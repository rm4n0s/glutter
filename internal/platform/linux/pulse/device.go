//go:build linux && !android

package pulse

import (
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// device is an open stream of the server. Its fields are guarded by the lock of the main loop, which the callbacks hold when they
// run and the methods take.
type device struct {
	svc     *Service
	kind    platform.AudioKind
	handler platform.AudioHandler
	format  platform.AudioFormat
	handle  uintptr
	stream  uintptr
	// scratch is C memory: two words for the pointer and the size that begin_write and peek give, and the sample specification, the
	// channel map and the buffer attributes that were passed to the server.
	scratch    uintptr
	frameBytes int
	// maxChunk is the most that is handed to the handler at once, in bytes: the size of the buffer the server granted.
	maxChunk int

	paused, lost, closed bool
	// underruns counts the times the server ran out of frames to play (a sound that has a gap), for the tests and the log.
	underruns int
}

var _ platform.AudioDevice = (*device)(nil)

// Format returns the format of the frames of the handler; the server converts to and from the format of its device.
func (d *device) Format() platform.AudioFormat { return d.format }

// Underruns is how many times the server ran out of frames to play since the device was opened: each is a gap in the sound.
func (d *device) Underruns() int {
	d.svc.lock()
	defer d.svc.unlock()
	return d.underruns
}

// lose says that the device is gone: the handler hears of it once.
func (d *device) lose() {
	if d.lost || d.closed {
		return
	}
	d.lost = true
	d.handler.StateChanged(platform.AudioLost)
}

// stateChanged is the state callback of the stream (main loop thread, lock held): a stream that fails or ends, when the app did
// not end it, is a device that was removed or a server that went.
func (d *device) stateChanged() {
	s := d.svc
	s.lib.mainloopSignal(s.loop, 0)
	if d.closed || d.stream == 0 {
		return
	}
	switch s.lib.streamGetState(d.stream) {
	case streamFailed, streamTerminated:
		d.lose()
	}
}

// service is the write or the read callback of the stream (main loop thread, lock held), and is called by Resume to catch up.
func (d *device) service() {
	if d.closed || d.lost || d.paused || d.stream == 0 {
		return
	}
	if d.kind == platform.AudioCapture {
		d.read()
	} else {
		d.write()
	}
}

// maxPieces keeps one callback from writing for ever: a server that keeps asking (a handler that is very fast) is served again by
// the next callback.
const maxPieces = 64

// write gives the server the frames it asks for. The memory comes from the server (begin_write), so the handler fills it in place.
func (d *device) write() {
	lib := d.svc.lib
	for range maxPieces {
		n := lib.streamWritableSize(d.stream)
		// (size_t) -1 is the answer of a stream that is not connected.
		if n == 0 || n > 1<<30 {
			return
		}
		want := min(int(n), d.maxChunk)
		want -= want % d.frameBytes
		if want < d.frameBytes {
			return
		}
		*(*uintptr)(dl.Pointer(d.scratch + 8)) = uintptr(want)
		if lib.streamBeginWrite(d.stream, d.scratch, d.scratch+8) < 0 {
			return
		}
		data := *(*uintptr)(dl.Pointer(d.scratch))
		got := int(*(*uintptr)(dl.Pointer(d.scratch + 8)))
		got -= got % d.frameBytes
		if data == 0 || got < d.frameBytes {
			if data != 0 {
				lib.streamCancelWrite(d.stream)
			}
			return
		}
		d.handler.Fill(unsafe.Slice((*float32)(dl.Pointer(data)), got/4))
		if lib.streamWrite(d.stream, data, uintptr(got), 0, 0, seekRelative) < 0 {
			return
		}
	}
}

// read takes the frames that the server has recorded, one fragment after the other.
func (d *device) read() {
	lib := d.svc.lib
	for range maxPieces {
		if lib.streamPeek(d.stream, d.scratch, d.scratch+8) < 0 {
			return
		}
		data := *(*uintptr)(dl.Pointer(d.scratch))
		n := int(*(*uintptr)(dl.Pointer(d.scratch + 8)))
		if n == 0 {
			return // nothing is left to read
		}
		// A null pointer with a size is a hole in the recording (the server lost samples): it is dropped, as SDL does.
		if data != 0 {
			if frames := n / d.frameBytes; frames > 0 {
				d.handler.Capture(unsafe.Slice((*float32)(dl.Pointer(data)), frames*d.format.Channels))
			}
		}
		lib.streamDrop(d.stream)
	}
}

// cork pauses or resumes the stream on the server, with the lock held.
func (d *device) cork(on bool) {
	var b int32
	if on {
		b = 1
	}
	if op := d.svc.lib.streamCork(d.stream, b, 0, 0); op != 0 {
		d.svc.lib.operationUnref(op)
	}
}

// Pause stops the calls of the handler: the stream is corked, and what the server still asks for is not answered. It fails with
// AudioDeviceLost when the device is gone.
func (d *device) Pause() error {
	d.svc.lock()
	defer d.svc.unlock()
	if d.closed {
		return nil
	}
	if d.lost {
		return errors.New(platform.TagAudioDeviceLost, "the PulseAudio stream is gone")
	}
	d.paused = true
	d.cork(true)
	return nil
}

// Resume starts the stream again, and writes what the server asked for while it was paused.
func (d *device) Resume() error {
	d.svc.lock()
	defer d.svc.unlock()
	if d.closed {
		return nil
	}
	if d.lost {
		return errors.New(platform.TagAudioDeviceLost, "the PulseAudio stream is gone")
	}
	d.paused = false
	d.cork(false)
	d.service()
	return nil
}

// Close ends the stream. The callbacks run with the lock that Close takes, so when it returns none is running or will run.
func (d *device) Close() error {
	d.svc.lock()
	defer d.svc.unlock()
	d.closeLocked()
	return nil
}

// closeLocked is Close with the lock of the main loop held.
func (d *device) closeLocked() {
	if d.closed {
		return
	}
	d.closed = true
	d.svc.discardStream(d)
	d.svc.release(d)
}
