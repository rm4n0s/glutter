//go:build linux && !android

package alsa

import (
	"log/slog"
	"runtime"
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// What the owner of a device wants of its goroutine.
const (
	wantRun = iota
	wantPause
	wantClose
)

// device is an open PCM. Its goroutine (run) does all the calls on the PCM, so the library's handle is never used from two threads;
// the other methods only change what is wanted and wait for the goroutine to take notice.
type device struct {
	lib     *library
	pcm     uintptr
	kind    platform.AudioKind
	handler platform.AudioHandler
	format  platform.AudioFormat
	// buf holds the frames of one step, in Go memory: ALSA reads or fills it during a call only.
	buf []float32

	mu     sync.Mutex
	cond   *sync.Cond
	want   int
	parked bool // the goroutine is waiting (paused, lost or ended), and calls the handler no more
	lost   bool
	done   chan struct{}
}

var _ platform.AudioDevice = (*device)(nil)

func newDevice(lib *library, pcm uintptr, kind platform.AudioKind, h platform.AudioHandler, format platform.AudioFormat, period int) *device {
	d := &device{lib: lib, pcm: pcm, kind: kind, handler: h, format: format, done: make(chan struct{})}
	d.cond = sync.NewCond(&d.mu)
	// A step can hold the frames of up to four periods: when the goroutine was late, what is free is filled at once.
	d.buf = make([]float32, period*4*format.Channels)
	return d
}

// Format returns the format of the frames of the handler; the plug layer of ALSA converts to what the hardware takes.
func (d *device) Format() platform.AudioFormat { return d.format }

// run is the goroutine of the device: it calls the handler until the device is closed.
func (d *device) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(d.done)
	defer func() {
		if r := recover(); r != nil {
			slog.Error("audio: the handler of an ALSA device panicked", "panic", r)
			d.lose()
		}
	}()
	if d.kind == platform.AudioCapture {
		d.lib.start(d.pcm)
	}
	for d.await() {
		if !d.step() {
			d.lose()
		}
	}
}

// await returns true when the device is to run. While it is paused the PCM is stopped and the goroutine waits; it returns false
// when the device is to close, and a lost device waits here for that.
func (d *device) await() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch d.want {
	case wantRun:
		if !d.lost {
			return true
		}
	case wantClose:
		return false
	}
	if !d.lost {
		d.mu.Unlock()
		d.lib.drop(d.pcm)
		d.mu.Lock()
	}
	d.parked = true
	d.cond.Broadcast()
	for d.want == wantPause || (d.want == wantRun && d.lost) {
		d.cond.Wait()
	}
	d.parked = false
	if d.want == wantClose {
		d.cond.Broadcast()
		return false
	}
	d.mu.Unlock()
	d.lib.prepare(d.pcm)
	if d.kind == platform.AudioCapture {
		d.lib.start(d.pcm)
	}
	d.mu.Lock()
	return true
}

// lose says that the device is gone, once; the goroutine then waits for Close.
func (d *device) lose() {
	d.mu.Lock()
	already := d.lost
	d.lost = true
	d.mu.Unlock()
	if !already {
		d.handler.StateChanged(platform.AudioLost)
	}
}

// step waits for the device and moves the frames that can be moved; it returns false when the device is gone.
func (d *device) step() bool {
	rc := d.lib.wait(d.pcm, int32(waitTimeout.Milliseconds()))
	if rc < 0 {
		return d.recover(rc)
	}
	avail := d.lib.availUpdate(d.pcm)
	if avail < 0 {
		return d.recover(int32(avail))
	}
	frames := min(int(avail), len(d.buf)/d.format.Channels)
	if frames == 0 {
		return d.lib.state(d.pcm) != stateDisconnected
	}
	if d.kind == platform.AudioCapture {
		return d.read(frames)
	}
	return d.write(frames)
}

// write fills the free space of the buffer of the device.
func (d *device) write(frames int) bool {
	ch := d.format.Channels
	out := d.buf[:frames*ch]
	d.handler.Fill(out)
	for done := 0; done < frames; {
		n := d.lib.writei(d.pcm, unsafe.Pointer(&out[done*ch]), uint64(frames-done))
		switch {
		case n == -errAgain:
			return true // the device took what it could; the next step waits for it again
		case n < 0:
			// After an underrun the buffer is empty and the next step fills it; what was not written is dropped.
			return d.recover(int32(n))
		}
		done += int(n)
	}
	return true
}

// read gives the handler the frames that were recorded.
func (d *device) read(frames int) bool {
	ch := d.format.Channels
	n := d.lib.readi(d.pcm, unsafe.Pointer(&d.buf[0]), uint64(frames))
	switch {
	case n == -errAgain:
		return true
	case n < 0:
		if !d.recover(int32(n)) {
			return false
		}
		d.lib.start(d.pcm)
		return true
	case n > 0:
		d.handler.Capture(d.buf[:int(n)*ch])
	}
	return true
}

// recover brings the PCM back after an underrun (or overrun) or a suspend, and says whether it could.
func (d *device) recover(code int32) bool {
	if code == -errNoDev {
		return false
	}
	return d.lib.recover(d.pcm, code, 1) >= 0 && d.lib.state(d.pcm) != stateDisconnected
}

// Pause stops the PCM and returns when the goroutine does not call the handler any more. It fails with AudioDeviceLost when the
// device is gone.
func (d *device) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.want == wantClose {
		return nil
	}
	if d.lost {
		return errors.New(platform.TagAudioDeviceLost, "the ALSA device is gone")
	}
	d.want = wantPause
	d.cond.Broadcast()
	for !d.parked && !d.lost {
		d.cond.Wait()
	}
	return nil
}

// Resume starts the PCM again after a pause.
func (d *device) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.want == wantClose {
		return nil
	}
	if d.lost {
		return errors.New(platform.TagAudioDeviceLost, "the ALSA device is gone")
	}
	d.want = wantRun
	d.cond.Broadcast()
	return nil
}

// Close ends the goroutine, waits for it, and closes the PCM. When it returns the handler is not running.
func (d *device) Close() error {
	d.mu.Lock()
	if d.want == wantClose {
		d.mu.Unlock()
		return nil
	}
	d.want = wantClose
	d.cond.Broadcast()
	d.mu.Unlock()
	<-d.done
	d.lib.drop(d.pcm)
	d.lib.close(d.pcm)
	return nil
}
