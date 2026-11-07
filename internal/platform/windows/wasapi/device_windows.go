//go:build windows

package wasapi

import (
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The timing of the thread of a device.
const (
	// waitLimit is how long the thread waits for the engine's event before it looks at the client again: a client whose device was
	// unplugged may not signal anything, and calls of the owner wait for the thread.
	waitLimit = time.Second
	// mmcssTask is the class of the multimedia scheduler that the thread joins (SDL's choice).
	mmcssTask = "Pro Audio"
	// commandQueue is how many commands can wait for the thread (a notification that finds the queue full is dropped: the next
	// one, or the timeout, finds the same state).
	commandQueue = 16
)

// device is an open stream of the audio engine. A thread of its own, in the multithreaded apartment, makes every call on the COM
// objects of the stream (run), so that none of them is ever used by two threads; the methods of the owner and the notifications of
// the service are commands that the thread runs between two events of the engine.
type device struct {
	svc     *Service
	kind    platform.AudioKind
	pinned  string
	opts    platform.AudioOpenOptions
	handler platform.AudioHandler
	log     *slog.Logger

	// format is the format of the frames of the handler, set when the device is opened and fixed after: a stream that moves to a
	// new default device asks that device for the same format, and the engine converts.
	format platform.AudioFormat
	ready  chan error
	done   chan struct{}
	cmds   chan func()

	evMu     sync.Mutex
	evClosed bool
	cmdEvent uintptr
	bufEvent uintptr

	// The rest belongs to the thread.
	enum, ac, stream win32.Object
	curID            string
	bufFrames        uint32
	started          bool
	paused           bool
	closing          bool
	lost             bool
	rebuilds         atomic.Int32 // how many times the stream was made again (the tests look at it)
	zeros            []float32
	mmcss            uintptr
	coinit           bool
}

var _ platform.AudioDevice = (*device)(nil)

// newDevice makes the device that Open starts; its thread is run.
func newDevice(s *Service, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) *device {
	return &device{
		svc: s, kind: kind, pinned: opts.ID, opts: opts, handler: h, log: s.log,
		ready: make(chan error, 1), done: make(chan struct{}), cmds: make(chan func(), commandQueue),
	}
}

// Format returns the format of the frames of the handler.
func (d *device) Format() platform.AudioFormat { return d.format }

// run is the goroutine of the device: it opens the stream, tells Open, and serves the engine until the device is closed.
func (d *device) run() {
	runtime.LockOSThread()
	defer close(d.done)
	defer d.finish()
	err := d.start()
	d.ready <- err
	if err != nil {
		return
	}
	d.serve()
}

// start makes the apartment, the events and the stream.
func (d *device) start() error {
	if err := win32.CoInitializeEx(win32.COInitMultithreaded); err != nil {
		return err
	}
	d.coinit = true
	d.mmcss, _ = win32.AvSetMmThreadCharacteristics(mmcssTask)
	cmd, err := win32.CreateEvent(false, false)
	if err != nil {
		return err
	}
	buf, err := win32.CreateEvent(false, false)
	if err != nil {
		if cerr := win32.CloseHandle(cmd); cerr != nil {
			d.log.Debug("audio: an event could not be closed", "err", cerr)
		}
		return err
	}
	d.evMu.Lock()
	d.cmdEvent, d.bufEvent = cmd, buf
	d.evMu.Unlock()
	if err := d.activate(d.opts.Format, false); err != nil {
		return err
	}
	return d.begin()
}

// finish runs when the thread ends: it gives the stream and the events back and leaves the apartment.
func (d *device) finish() {
	d.teardown()
	d.evMu.Lock()
	d.evClosed = true
	for _, h := range []uintptr{d.cmdEvent, d.bufEvent} {
		if h == 0 {
			continue
		}
		if err := win32.CloseHandle(h); err != nil {
			d.log.Debug("audio: an event could not be closed", "err", err)
		}
	}
	d.evMu.Unlock()
	if d.mmcss != 0 {
		win32.AvRevertMmThreadCharacteristics(d.mmcss)
	}
	if d.coinit {
		win32.CoUninitialize()
	}
	d.svc.forget(d)
}

// wake signals the command event, unless the thread has given the events back.
func (d *device) wake() {
	d.evMu.Lock()
	defer d.evMu.Unlock()
	if d.evClosed || d.cmdEvent == 0 {
		return
	}
	if err := win32.SetEvent(d.cmdEvent); err != nil {
		d.log.Debug("audio: the thread of a device could not be woken", "err", err)
	}
}

// post queues a command for the thread without waiting for it.
func (d *device) post(fn func()) {
	select {
	case d.cmds <- fn:
		d.wake()
	default:
		d.log.Debug("audio: the commands of a device are not taken; one is dropped")
	}
}

// exec runs fn on the thread and returns when it is done; false when the thread has ended.
func (d *device) exec(fn func()) bool {
	finished := make(chan struct{})
	select {
	case d.cmds <- func() { fn(); close(finished) }:
	case <-d.done:
		return false
	}
	d.wake()
	select {
	case <-finished:
		return true
	case <-d.done:
		return false
	}
}

// serve waits for the engine and for commands until the device is closed.
func (d *device) serve() {
	for !d.closing {
		handles := []uintptr{d.cmdEvent}
		if d.started {
			handles = append(handles, d.bufEvent)
		}
		res, err := win32.WaitForMultipleObjects(handles, false, waitLimit)
		if err != nil {
			d.log.Error("audio: waiting for a device failed", "err", err)
			d.lose()
			// The events are no good any more; only the owner's calls can be answered, by polling for them.
			time.Sleep(10 * time.Millisecond)
			d.drain()
			continue
		}
		switch res {
		case win32.WaitObject0:
			d.drain()
		case win32.WaitObject0 + 1:
			d.guard(d.step)
		case win32.WaitTimeout:
			if d.started && !d.paused {
				d.guard(d.probe)
			}
			d.guard(d.checkDefault)
		}
	}
}

// drain runs the commands that are queued.
func (d *device) drain() {
	for {
		select {
		case fn := <-d.cmds:
			fn()
		default:
			return
		}
	}
}

// guard calls fn and loses the device when the handler panicked in it: a panic must not end the thread with the owner waiting.
func (d *device) guard(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			d.log.Error("audio: the handler of a WASAPI device panicked", "panic", r, "stack", string(debug.Stack()))
			d.lose()
		}
	}()
	fn()
}

// step serves one event of the engine: the playback needs frames, or the recording has some.
func (d *device) step() {
	if d.lost || d.paused || !d.started {
		return
	}
	if d.kind == platform.AudioCapture {
		d.capture()
	} else {
		d.render()
	}
}

// render fills the free part of the engine's buffer.
func (d *device) render() {
	var padding uint32
	if hr := call(d.ac, win32.IAudioClientGetCurrentPadding, uintptr(unsafe.Pointer(&padding))); hr.Failed() {
		d.failed("IAudioClient::GetCurrentPadding", hr)
		return
	}
	if padding >= d.bufFrames {
		return
	}
	n := d.bufFrames - padding
	var data uintptr
	if hr := call(d.stream, win32.IAudioRenderClientGetBuffer, uintptr(n), uintptr(unsafe.Pointer(&data))); hr.Failed() {
		d.failed("IAudioRenderClient::GetBuffer", hr)
		return
	}
	d.handler.Fill(win32.Float32s(data, int(n)*d.format.Channels))
	if hr := call(d.stream, win32.IAudioRenderClientReleaseBuffer, uintptr(n), 0); hr.Failed() {
		d.failed("IAudioRenderClient::ReleaseBuffer", hr)
	}
}

// capture hands every packet that the engine has to the handler.
func (d *device) capture() {
	for !d.lost {
		var packet uint32
		if hr := call(d.stream, win32.IAudioCaptureClientGetNextPacketSize, uintptr(unsafe.Pointer(&packet))); hr.Failed() {
			d.failed("IAudioCaptureClient::GetNextPacketSize", hr)
			return
		}
		if packet == 0 {
			return
		}
		var data uintptr
		var frames, flags uint32
		if hr := call(d.stream, win32.IAudioCaptureClientGetBuffer, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&frames)),
			uintptr(unsafe.Pointer(&flags)), 0, 0); hr.Failed() {
			d.failed("IAudioCaptureClient::GetBuffer", hr)
			return
		}
		if frames > 0 {
			n := int(frames) * d.format.Channels
			in := win32.Float32s(data, n)
			if flags&win32.AudclntBufferFlagsSilent != 0 || in == nil {
				in = d.silence(n)
			}
			d.handler.Capture(in)
		}
		if hr := call(d.stream, win32.IAudioCaptureClientReleaseBuffer, uintptr(frames)); hr.Failed() {
			d.failed("IAudioCaptureClient::ReleaseBuffer", hr)
			return
		}
	}
}

// silence is n zero samples (a packet that the engine marked silent has bytes that mean nothing).
func (d *device) silence(n int) []float32 {
	if len(d.zeros) < n {
		d.zeros = make([]float32, n)
	}
	return d.zeros[:n]
}

// probe looks at the client without serving an event: a device that was unplugged shows here.
func (d *device) probe() {
	if d.lost || !d.started {
		return
	}
	var padding uint32
	if hr := call(d.ac, win32.IAudioClientGetCurrentPadding, uintptr(unsafe.Pointer(&padding))); hr.Failed() {
		d.failed("IAudioClient::GetCurrentPadding", hr)
	}
}

// failed is what the thread does when a call on the client fails: a device that follows the default one is made again on the
// default device when its client was invalidated, any other failure ends the stream.
func (d *device) failed(method string, hr win32.HRESULT) {
	if classify(hr) == failLost && d.pinned == "" && !d.closing {
		d.rebuild()
		return
	}
	d.log.Warn("audio: a WASAPI device is lost", "method", method, "hresult", hr.String())
	d.lose()
}

// lose ends the stream and tells the owner, once.
func (d *device) lose() {
	if d.lost {
		return
	}
	d.lost = true
	d.teardown()
	d.handler.StateChanged(platform.AudioLost)
}

// followDefault is the command that the service posts when the default device of the flow changed to the endpoint with the ID: the
// stream moves to it. An empty ID means that no device is the default any more.
func (d *device) followDefault(id string) {
	if d.lost || d.closing || d.pinned != "" || id == d.curID {
		return
	}
	d.log.Debug("audio: the default device changed; the stream moves to it", "id", id)
	d.rebuild()
}

// checkDefault looks at which endpoint is the default one, for a device that follows it: the notification of Windows is what moves
// the stream at once, and this look, once a second, finds the change when the notification was lost or never came (the drivers of
// Windows and Wine differ in when they send it).
func (d *device) checkDefault() {
	if d.pinned != "" || d.lost || d.closing || d.enum.IsNil() {
		return
	}
	var dev uintptr
	if hr := call(d.enum, win32.IMMDeviceEnumeratorGetDefaultAudioEndpoint, uintptr(flowOf(d.kind)), win32.EConsole, uintptr(unsafe.Pointer(&dev))); hr.Failed() {
		return
	}
	obj := win32.ObjectFromPtr(dev)
	id := endpointIDOf(obj)
	obj.Release()
	d.followDefault(id)
}

// rebuild makes the stream again, on the endpoint that is the default now, in the format the device has. The device is lost when
// that cannot be done.
func (d *device) rebuild() {
	d.rebuilds.Add(1)
	d.teardown()
	if err := d.activate(d.format, true); err != nil {
		d.log.Warn("audio: the stream could not move to the new default device", "err", err)
		d.lost = true
		d.handler.StateChanged(platform.AudioLost)
		return
	}
	if d.paused {
		return
	}
	if err := d.begin(); err != nil {
		d.log.Warn("audio: the stream could not start on the new default device", "err", err)
		d.lose()
	}
}

// teardown stops the stream and releases its objects.
func (d *device) teardown() {
	if d.started {
		if hr := call(d.ac, win32.IAudioClientStop); hr.Failed() {
			d.log.Debug("audio: a client could not be stopped", "hresult", hr.String())
		}
	}
	d.started = false
	release(d.stream)
	release(d.ac)
	release(d.enum)
	d.stream, d.ac, d.enum = win32.Object{}, win32.Object{}, win32.Object{}
}

// endpoint is the endpoint of the device: the one with the ID, or the default one of its flow.
func (d *device) endpoint() (win32.Object, error) {
	var dev uintptr
	if d.pinned != "" {
		id, err := win32.UTF16PtrFromString(d.pinned)
		if err != nil {
			return win32.Object{}, err
		}
		if hr := call(d.enum, win32.IMMDeviceEnumeratorGetDevice, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(&dev))); hr.Failed() {
			return win32.Object{}, hresultError("IMMDeviceEnumerator::GetDevice", hr)
		}
		return win32.ObjectFromPtr(dev), nil
	}
	if hr := call(d.enum, win32.IMMDeviceEnumeratorGetDefaultAudioEndpoint, uintptr(flowOf(d.kind)), win32.EConsole, uintptr(unsafe.Pointer(&dev))); hr.Failed() {
		return win32.Object{}, hresultError("IMMDeviceEnumerator::GetDefaultAudioEndpoint", hr)
	}
	return win32.ObjectFromPtr(dev), nil
}

// activate makes the enumerator and the audio client and initializes the client. want is the format that the app asked for; with
// fixed it is the format of the stream, and a device that does not take it is an error, otherwise the engine's own format is
// the fallback, and format is set to what the stream has. On a failure nothing is left held.
func (d *device) activate(want platform.AudioFormat, fixed bool) (err error) {
	defer func() {
		if err != nil {
			d.teardown()
		}
	}()
	enum, hr := win32.CoCreateInstanceHR(&win32.CLSIDMMDeviceEnumerator, &win32.IIDMMDeviceEnumerator, win32.ClsctxAll)
	if hr.Failed() {
		return errors.New(platform.TagAudioUnavailable, "Windows has no audio device enumerator", "hresult", hr.String())
	}
	d.enum = enum
	dev, err := d.endpoint()
	if err != nil {
		return err
	}
	defer dev.Release()
	d.curID = endpointIDOf(dev)

	probe, hr := activateClient(dev)
	if hr.Failed() {
		return hresultError("IMMDevice::Activate", hr)
	}
	mix, hr := mixFormat(probe)
	release(probe)
	if hr.Failed() {
		return hresultError("IAudioClient::GetMixFormat", hr)
	}
	format := want
	if !fixed {
		format = chooseFormat(want, mix)
	}
	ac, frames, hr := d.initialize(dev, format)
	if hr.Failed() && classify(hr) == failFormat && !fixed && format != mix {
		// The engine does not convert from what was asked for: take its own format, which the app's streams are converted to.
		format = mix
		ac, frames, hr = d.initialize(dev, format)
	}
	if hr.Failed() {
		return hresultError("IAudioClient::Initialize", hr)
	}
	d.ac = ac
	d.bufFrames = frames
	if !fixed {
		d.format = format
	}
	return nil
}

// activateClient asks an endpoint for an audio client.
func activateClient(dev win32.Object) (win32.Object, win32.HRESULT) {
	var ac uintptr
	hr := call(dev, win32.IMMDeviceActivate, uintptr(unsafe.Pointer(&win32.IIDAudioClient)), win32.ClsctxAll, 0, uintptr(unsafe.Pointer(&ac)))
	return win32.ObjectFromPtr(ac), hr
}

// initialize makes a client for the format in shared mode, driven by events, and gets the stream's service (d.stream). The client
// is the one that IAudioClient3 took, when Windows has it and takes the format, else a plain one; a client that failed to initialize
// is released and never used again. frames is the size of the engine's buffer. The client that is returned belongs to the caller.
func (d *device) initialize(dev win32.Object, format platform.AudioFormat) (win32.Object, uint32, win32.HRESULT) {
	wf := wireFormat(format)
	flags := uintptr(win32.AudclntStreamFlagsEventCallback | win32.AudclntStreamFlagsAutoConvertPCM | win32.AudclntStreamFlagsSrcDefaultQuality)
	ac, hr := d.initPeriod(dev, &wf, flags)
	if hr.Failed() {
		var hns int64
		if d.opts.BufferFrames > 0 {
			hns = int64(framesDuration(d.opts.BufferFrames, format.Rate) / 100)
		}
		ac, hr = d.initPlain(dev, &wf, flags, hns)
		if hr.Failed() && hns != 0 {
			// The buffer that was asked for is a hint, not a condition.
			ac, hr = d.initPlain(dev, &wf, flags, 0)
		}
	}
	if hr.Failed() {
		return win32.Object{}, 0, hr
	}
	d.evMu.Lock()
	event := d.bufEvent
	d.evMu.Unlock()
	if hr = call(ac, win32.IAudioClientSetEventHandle, event); hr.Failed() {
		release(ac)
		return win32.Object{}, 0, hr
	}
	var frames uint32
	if hr = call(ac, win32.IAudioClientGetBufferSize, uintptr(unsafe.Pointer(&frames))); hr.Failed() {
		release(ac)
		return win32.Object{}, 0, hr
	}
	iid := &win32.IIDAudioRenderClient
	if d.kind == platform.AudioCapture {
		iid = &win32.IIDAudioCaptureClient
	}
	var svc uintptr
	if hr = call(ac, win32.IAudioClientGetService, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&svc))); hr.Failed() {
		release(ac)
		return win32.Object{}, 0, hr
	}
	d.stream = win32.ObjectFromPtr(svc)
	return ac, frames, win32.SOK
}

// initPeriod makes a client with IAudioClient3 (Windows 10), which takes the period of the engine's buffer: the period that the app
// wants, or the smallest one that the engine has. The result is E_NOINTERFACE when Windows has no IAudioClient3.
func (d *device) initPeriod(dev win32.Object, wf *win32.WaveFormatExtensible, flags uintptr) (win32.Object, win32.HRESULT) {
	ac, hr := activateClient(dev)
	if hr.Failed() {
		return win32.Object{}, hr
	}
	var c3 uintptr
	if hr = call(ac, win32.IUnknownQueryInterface, uintptr(unsafe.Pointer(&win32.IIDAudioClient3)), uintptr(unsafe.Pointer(&c3))); hr.Failed() {
		release(ac)
		return win32.Object{}, hr
	}
	client3 := win32.ObjectFromPtr(c3)
	defer client3.Release()
	var def, fundamental, lowest, highest uint32
	if hr = call(client3, win32.IAudioClient3GetSharedModeEnginePeriod, uintptr(unsafe.Pointer(wf)), uintptr(unsafe.Pointer(&def)),
		uintptr(unsafe.Pointer(&fundamental)), uintptr(unsafe.Pointer(&lowest)), uintptr(unsafe.Pointer(&highest))); hr.Failed() {
		release(ac)
		return win32.Object{}, hr
	}
	period := periodFrames(uint32(max(d.opts.BufferFrames, 0)), fundamental, lowest, highest)
	if hr = call(client3, win32.IAudioClient3InitializeSharedAudioStream, flags, uintptr(period), uintptr(unsafe.Pointer(wf)), 0); hr.Failed() {
		d.log.Debug("audio: IAudioClient3 did not take the stream", "hresult", hr.String())
		release(ac)
		return win32.Object{}, hr
	}
	return ac, win32.SOK
}

// initPlain makes a client with IAudioClient::Initialize in shared mode; hns is the length of the buffer that is asked for, 0 for
// the engine's choice.
func (d *device) initPlain(dev win32.Object, wf *win32.WaveFormatExtensible, flags uintptr, hns int64) (win32.Object, win32.HRESULT) {
	ac, hr := activateClient(dev)
	if hr.Failed() {
		return win32.Object{}, hr
	}
	if hr = call(ac, win32.IAudioClientInitialize, win32.AudclntShareModeShared, flags, uintptr(hns), 0, uintptr(unsafe.Pointer(wf)), 0); hr.Failed() {
		release(ac)
		return win32.Object{}, hr
	}
	return ac, win32.SOK
}

// begin starts the stream. A recording is flushed right after it starts: what the engine had before is old.
func (d *device) begin() error {
	if hr := call(d.ac, win32.IAudioClientStart); hr.Failed() {
		return hresultError("IAudioClient::Start", hr)
	}
	d.started = true
	if d.kind == platform.AudioCapture {
		d.flush()
	}
	return nil
}

// flush reads and drops what a recording has.
func (d *device) flush() {
	for range 64 {
		var packet uint32
		if hr := call(d.stream, win32.IAudioCaptureClientGetNextPacketSize, uintptr(unsafe.Pointer(&packet))); hr.Failed() || packet == 0 {
			return
		}
		var data uintptr
		var frames, flags uint32
		if hr := call(d.stream, win32.IAudioCaptureClientGetBuffer, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&frames)),
			uintptr(unsafe.Pointer(&flags)), 0, 0); hr.Failed() {
			return
		}
		if hr := call(d.stream, win32.IAudioCaptureClientReleaseBuffer, uintptr(frames)); hr.Failed() {
			return
		}
	}
}

// Pause stops the stream: the handler is not called until Resume. It fails with AudioDeviceLost when the device is gone.
func (d *device) Pause() error {
	var lost bool
	if !d.exec(func() {
		lost = d.lost
		if lost || d.paused {
			return
		}
		d.paused = true
		if d.started {
			if hr := call(d.ac, win32.IAudioClientStop); hr.Failed() {
				d.log.Debug("audio: a client could not be stopped", "hresult", hr.String())
			}
			// The frames that were in the buffer are old when the stream goes on.
			if hr := call(d.ac, win32.IAudioClientReset); hr.Failed() {
				d.log.Debug("audio: a client could not be reset", "hresult", hr.String())
			}
			d.started = false
		}
	}) || lost {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone")
	}
	return nil
}

// Resume starts the stream again. It fails with AudioDeviceLost when the device is gone.
func (d *device) Resume() error {
	var lost bool
	var beginErr error
	if !d.exec(func() {
		lost = d.lost
		if lost || !d.paused {
			return
		}
		d.paused = false
		beginErr = d.begin()
	}) || lost {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone")
	}
	return beginErr
}

// Close ends the stream and its thread; when it returns the handler is not running and is never called again.
func (d *device) Close() error {
	d.exec(func() { d.closing = true })
	<-d.done
	return nil
}
