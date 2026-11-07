//go:build linux && !android

// Package pulse is the PulseAudio backend of the audio service on Linux (SPEC §28.3, step 31.5): playback and capture streams of
// float32 frames on libpulse's threaded main loop, the devices and the default device of the server, and a notification when
// they change. libpulse is loaded when it is first needed; the library is never linked. It speaks to PulseAudio and to PipeWire's
// PulseAudio protocol alike. The design (a thread of the library that calls the handler, a write callback that asks for just what
// the server wants) follows the PulseAudio driver of SDL 3 (src/audio/pulseaudio/SDL_pulseaudio.c; zlib license,
// third_party/sdl/LICENSE), translated to Go and changed.
//
// Flutter: no counterpart
package pulse

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The times the service waits for the server. A server that answers does so in milliseconds; these only end a wait for a server
// that is stopped.
const (
	connectTimeout = 5 * time.Second
	queryTimeout   = 5 * time.Second
	streamTimeout  = 10 * time.Second
)

// The default sizes of the buffers when the app asks for none: the latency of a playback stream (a target of SPEC §28.6 is 30 ms
// at most) and the size of the pieces a capture stream is read in.
const (
	playbackLatency = 20 * time.Millisecond
	captureFragment = 10 * time.Millisecond
	defaultRate     = 48000
	defaultChannels = 2
	maxAudioChans   = 8
)

// Options configure a Service.
type Options struct {
	// AppName names the application to the server (it shows in a volume control); it defaults to the name of the program.
	AppName string
}

// Service is the PulseAudio backend: it implements platform.Audio. It connects to the server when it is first used, and again
// when the server was lost.
//
// Flutter: no counterpart
type Service struct {
	appName string
	load    func() (*library, error)
	changed atomic.Pointer[func()]

	// connMu serialises the connection and the queries. The callbacks never take it, and the code that holds it takes the lock of
	// the main loop, never the other way round.
	connMu sync.Mutex
	lib    *library
	loop   uintptr
	ctx    uintptr
	handle uintptr
	// closed is set by Close, and stopped says that the main loop was stopped then.
	closed, stopped bool

	// What follows is guarded by the lock of the main loop.
	ctxDead bool
	closing bool
	devices map[*device]struct{}
	query   *query
}

// query collects the answers of the server to a list of devices.
type query struct {
	infos                   []platform.AudioDeviceInfo
	defaultSink, defaultSrc string
}

// New makes a Service. Nothing is loaded or connected yet.
//
// Flutter: no counterpart
func New(opts Options) *Service {
	name := opts.AppName
	if name == "" {
		name = filepath.Base(os.Args[0])
	}
	return &Service{appName: name, load: loadLibrary, devices: map[*device]struct{}{}}
}

var _ platform.Audio = (*Service)(nil)

// lock takes the lock of the main loop. libpulse's lock belongs to the thread that took it, and a goroutine may move to another
// thread between two calls, so the goroutine is tied to its thread until unlock.
func (s *Service) lock() {
	runtime.LockOSThread()
	s.lib.mainloopLock(s.loop)
}

func (s *Service) unlock() {
	s.lib.mainloopUnlock(s.loop)
	runtime.UnlockOSThread()
}

// waitUntil waits, with the lock held, until done reports true or the timeout passes; it says whether done is true.
func (s *Service) waitUntil(done func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	timer := time.AfterFunc(timeout, func() {
		s.lock()
		s.lib.mainloopSignal(s.loop, 0)
		s.unlock()
	})
	defer timer.Stop()
	for !done() {
		if !time.Now().Before(deadline) {
			return false
		}
		s.lib.mainloopWait(s.loop)
	}
	return true
}

// reason is the server's words for the last error of the context, for a message.
func (s *Service) reason() string {
	return dl.GoString(s.lib.strerror(s.lib.contextErrno(s.ctx)))
}

// connect makes sure there is a context that is ready, with connMu held: the library is loaded, the main loop is started and a
// context is connected, again if the one there was lost. It fails with AudioUnavailable when libpulse cannot be loaded or no
// server answers.
func (s *Service) connect() error {
	if s.closed {
		return errors.New(platform.TagAudioUnavailable, "the audio service was closed")
	}
	if s.lib == nil {
		lib, err := s.load()
		if err != nil {
			return err
		}
		s.lib = lib
	}
	if s.loop == 0 {
		loop := s.lib.mainloopNew()
		if loop == 0 {
			return errors.New(platform.TagAudioUnavailable, "libpulse could not make its main loop")
		}
		if s.lib.mainloopStart(loop) < 0 {
			s.lib.mainloopFree(loop)
			return errors.New(platform.TagAudioUnavailable, "libpulse could not start its main loop")
		}
		s.loop = loop
		s.handle = addHandle(s)
	}
	if s.ctx != 0 {
		s.lock()
		dead := s.ctxDead
		s.unlock()
		if !dead {
			return nil
		}
		s.dropContext()
	}

	cbs := callbacks()
	name := dl.CString(s.appName)
	defer dl.Free(name)

	s.lock()
	defer s.unlock()
	ctx := s.lib.contextNew(s.lib.mainloopGetAPI(s.loop), name)
	if ctx == 0 {
		return errors.New(platform.TagAudioUnavailable, "libpulse could not make a context")
	}
	s.ctx = ctx
	s.ctxDead = false
	s.lib.contextSetStateCallback(ctx, cbs.contextState, s.handle)
	if s.lib.contextConnect(ctx, 0, contextNoAutoSpawn, 0) < 0 {
		reason := s.reason()
		s.discardContextLocked()
		return errors.New(platform.TagAudioUnavailable, "no PulseAudio server answers", "reason", reason)
	}
	ready := s.waitUntil(func() bool {
		st := s.lib.contextGetState(ctx)
		return st == contextReady || st == contextFailed || st == contextTerminated
	}, connectTimeout)
	if !ready || s.lib.contextGetState(ctx) != contextReady {
		reason := "the server did not answer in time"
		if ready {
			reason = s.reason()
		}
		s.discardContextLocked()
		return errors.New(platform.TagAudioUnavailable, "no PulseAudio server answers", "reason", reason)
	}
	s.lib.contextSetSubscribeCallback(ctx, cbs.subscribe, s.handle)
	if op := s.lib.contextSubscribe(ctx, subscriptionMaskSink|subscriptionMaskSource|subscriptionMaskServer, 0, 0); op != 0 {
		s.lib.operationUnref(op)
	}
	return nil
}

// dropContext disconnects and frees a context that is dead, with connMu held and without the lock of the loop.
func (s *Service) dropContext() {
	s.lock()
	s.discardContextLocked()
	s.unlock()
}

// discardContextLocked frees the context, with the lock of the loop held. The callbacks of a context that was dropped must not call
// the Service again: the state callback is cleared before.
func (s *Service) discardContextLocked() {
	if s.ctx == 0 {
		return
	}
	s.lib.contextSetStateCallback(s.ctx, 0, 0)
	s.lib.contextSetSubscribeCallback(s.ctx, 0, 0)
	s.lib.contextDisconnect(s.ctx)
	s.lib.contextUnref(s.ctx)
	s.ctx = 0
	s.ctxDead = true
}

// contextStateChanged is the state callback of the context (main loop thread, lock held). A context that failed or ended, when we
// did not end it, is a server that was lost: every open device is lost with it.
func (s *Service) contextStateChanged(c uintptr) {
	s.lib.mainloopSignal(s.loop, 0)
	if c != s.ctx || s.closing {
		return
	}
	switch s.lib.contextGetState(c) {
	case contextFailed, contextTerminated:
		s.ctxDead = true
		for d := range s.devices {
			d.lose()
		}
	}
}

// subscribed is the subscription callback (main loop thread, lock held): a device was added, removed or changed, or the default
// changed.
func (s *Service) subscribed(event uint32) {
	if fn := s.changed.Load(); fn != nil && *fn != nil {
		(*fn)()
	}
}

// serverInfo is the callback of the server's information, which has the names of the default devices.
func (s *Service) serverInfo(info uintptr) {
	s.lib.mainloopSignal(s.loop, 0)
	if s.query == nil {
		return
	}
	s.query.defaultSink = cStringAt(info + serverDefaultSinkOffset)
	s.query.defaultSrc = cStringAt(info + serverDefaultSourceOffset)
}

// deviceInfo is the callback of the lists of sinks and sources. eol is 1 at the end of the list and negative when the list failed.
func (s *Service) deviceInfo(info uintptr, eol int32, source bool) {
	s.lib.mainloopSignal(s.loop, 0)
	if eol != 0 || info == 0 || s.query == nil {
		return
	}
	name := cStringAt(info + infoNameOffset)
	// A monitor is what a sink plays, offered as a source: it is not a microphone.
	if source && strings.HasSuffix(name, ".monitor") {
		return
	}
	spec := (*sampleSpec)(dl.Pointer(info + infoSampleSpecOffset))
	kind := platform.AudioPlayback
	if source {
		kind = platform.AudioCapture
	}
	s.query.infos = append(s.query.infos, platform.AudioDeviceInfo{
		ID:     name,
		Name:   cStringAt(info + infoDescriptionOffset),
		Kind:   kind,
		Format: platform.AudioFormat{Rate: int(spec.rate), Channels: min(int(spec.channels), maxAudioChans)},
	})
}

// cStringAt reads the C string that the pointer at addr points to; a null pointer is the empty string.
func cStringAt(addr uintptr) string {
	p := *(*uintptr)(dl.Pointer(addr))
	if p == 0 {
		return ""
	}
	return dl.GoString(p)
}

// runOperation waits, with the lock held, for an operation of the server to end, and frees it; it says whether it ended in time.
func (s *Service) runOperation(op uintptr) bool {
	if op == 0 {
		return false
	}
	ok := s.waitUntil(func() bool {
		return s.lib.operationGetState(op) != operationRunning || s.ctxDead
	}, queryTimeout)
	s.lib.operationUnref(op)
	return ok
}

// Devices lists the sinks (playback) or the sources (capture) of the server, without the monitors of the sinks. It fails with
// AudioUnavailable when no server answers.
//
// Flutter: no counterpart
func (s *Service) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if err := s.connect(); err != nil {
		return nil, err
	}
	s.lock()
	defer s.unlock()
	q := &query{}
	s.query = q
	defer func() { s.query = nil }()
	if !s.runOperation(s.lib.contextGetServerInfo(s.ctx, callbacks().serverInfo, s.handle)) {
		return nil, errors.New(platform.TagAudioUnavailable, "the PulseAudio server did not answer the question about its devices")
	}
	list := s.lib.contextGetSinkInfoList
	if kind == platform.AudioCapture {
		list = s.lib.contextGetSourceInfoList
	}
	cb := callbacks().sinkInfo
	if kind == platform.AudioCapture {
		cb = callbacks().sourceInfo
	}
	if !s.runOperation(list(s.ctx, cb, s.handle)) {
		return nil, errors.New(platform.TagAudioUnavailable, "the PulseAudio server did not answer the question about its devices")
	}
	def := q.defaultSink
	if kind == platform.AudioCapture {
		def = q.defaultSrc
	}
	for i := range q.infos {
		q.infos[i].Default = q.infos[i].ID == def
	}
	return q.infos, nil
}

// SetChangeHandler sets what is called when a device appears, goes or changes, or the default changes. It is called on the thread
// of libpulse; nil clears it. The subscription is made when the service connects, which is when the app first asks for devices or
// opens one.
//
// Flutter: no counterpart
func (s *Service) SetChangeHandler(fn func()) {
	if fn == nil {
		s.changed.Store(nil)
		return
	}
	s.changed.Store(&fn)
}

// Close ends the connection: the devices that are open are closed, the main loop stops and the context is freed. The service is
// not used afterwards: its calls fail with AudioUnavailable. The memory of the stopped main loop is kept, because a device that
// was open can still be closed or paused by its owner afterwards, and takes the loop's lock to find out that it is closed.
//
// Flutter: no counterpart
func (s *Service) Close() {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	s.closed = true
	if s.loop == 0 || s.stopped {
		return
	}
	s.lock()
	s.closing = true
	for d := range s.devices {
		d.closeLocked()
	}
	s.discardContextLocked()
	s.closing = false
	s.unlock()
	s.lib.mainloopStop(s.loop)
	s.stopped = true
	removeHandle(s.handle)
}

// Open opens a playback or a capture stream. It fails with AudioUnavailable when no server answers, AudioFormatUnsupported when
// the server takes no such format, AudioCapturePermissionDenied when the server refuses the recording, and AudioDeviceOpenFailed
// for the rest (the device is not there, the stream cannot connect).
//
// Flutter: no counterpart
func (s *Service) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagAudioDeviceOpenFailed, err).SetMetadata("id", opts.ID)
	}
	format := opts.Format
	if format.Rate == 0 {
		format.Rate = defaultRate
	}
	if format.Channels == 0 {
		format.Channels = defaultChannels
	}
	if format.Rate < 1 || format.Rate > 768000 || format.Channels < 1 || format.Channels > maxAudioChans {
		return nil, errors.New(platform.TagAudioFormatUnsupported, "PulseAudio takes 1 to 8 channels and a rate from 1 Hz to 768 kHz",
			"rate", format.Rate, "channels", format.Channels)
	}

	s.connMu.Lock()
	defer s.connMu.Unlock()
	if err := s.connect(); err != nil {
		return nil, err
	}

	frameBytes := format.Channels * 4
	frames := opts.BufferFrames
	if frames <= 0 {
		if kind == platform.AudioCapture {
			frames = int(int64(format.Rate) * int64(captureFragment) / int64(time.Second))
		} else {
			frames = int(int64(format.Rate) * int64(playbackLatency) / int64(time.Second))
		}
		frames = max(frames, 64)
	}

	d := &device{svc: s, kind: kind, handler: h, format: format, frameBytes: frameBytes}
	d.handle = addHandle(d)
	d.scratch = dl.Malloc(16 + unsafe.Sizeof(sampleSpec{}) + unsafe.Sizeof(channelMap{}) + unsafe.Sizeof(bufferAttr{}))
	spec := (*sampleSpec)(dl.Pointer(d.scratch + 16))
	cmap := (*channelMap)(dl.Pointer(d.scratch + 16 + unsafe.Sizeof(sampleSpec{})))
	attr := (*bufferAttr)(dl.Pointer(d.scratch + 16 + unsafe.Sizeof(sampleSpec{}) + unsafe.Sizeof(channelMap{})))
	*spec = sampleSpec{format: sampleFloat32LE, rate: uint32(format.Rate), channels: uint8(format.Channels)}
	fillChannelMap(cmap, format.Channels)
	bytes := uint32(frames * frameBytes)
	// The first request can come while the stream is still connecting, before the granted size is known.
	d.maxChunk = int(bytes)
	*attr = bufferAttr{maxlength: paNoValue, tlength: paNoValue, prebuf: paNoValue, minreq: paNoValue, fragsize: paNoValue}
	if kind == platform.AudioCapture {
		attr.fragsize = bytes
	} else {
		attr.tlength = bytes
	}
	specAddr := d.scratch + 16
	mapAddr := specAddr + unsafe.Sizeof(sampleSpec{})
	attrAddr := mapAddr + unsafe.Sizeof(channelMap{})

	var devName uintptr
	if opts.ID != "" {
		devName = dl.CString(opts.ID)
		defer dl.Free(devName)
	}
	streamName := "Playback"
	if kind == platform.AudioCapture {
		streamName = "Recording"
	}
	cName := dl.CString(streamName)
	defer dl.Free(cName)

	cbs := callbacks()
	s.lock()
	defer s.unlock()
	if s.ctxDead {
		s.release(d)
		return nil, errors.New(platform.TagAudioUnavailable, "the PulseAudio server was lost")
	}
	d.stream = s.lib.streamNew(s.ctx, cName, specAddr, mapAddr)
	if d.stream == 0 {
		reason := s.reason()
		s.release(d)
		return nil, errors.New(platform.TagAudioFormatUnsupported, "the PulseAudio server does not take the format",
			"rate", format.Rate, "channels", format.Channels, "reason", reason)
	}
	s.lib.streamSetStateCallback(d.stream, cbs.streamState, d.handle)
	flags := int32(streamAdjustLatency)
	var rc int32
	if kind == platform.AudioCapture {
		s.lib.streamSetReadCallback(d.stream, cbs.streamRequest, d.handle)
		rc = s.lib.streamConnectRecord(d.stream, devName, attrAddr, flags)
	} else {
		s.lib.streamSetWriteCallback(d.stream, cbs.streamRequest, d.handle)
		s.lib.streamSetUnderflowCallback(d.stream, cbs.underflow, d.handle)
		rc = s.lib.streamConnectPlayback(d.stream, devName, attrAddr, flags, 0, 0)
	}
	connected := rc >= 0
	if connected {
		ready := s.waitUntil(func() bool {
			st := s.lib.streamGetState(d.stream)
			return st == streamReady || st == streamFailed || st == streamTerminated
		}, streamTimeout)
		if !ready {
			s.discardStream(d)
			s.release(d)
			return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the PulseAudio stream did not become ready in time", "id", opts.ID)
		}
		connected = s.lib.streamGetState(d.stream) == streamReady
	}
	if !connected {
		errno := s.lib.contextErrno(s.ctx)
		reason := dl.GoString(s.lib.strerror(errno))
		s.discardStream(d)
		s.release(d)
		if kind == platform.AudioCapture && errno == errAccess {
			return nil, errors.New(platform.TagAudioCapturePermissionDenied, "the PulseAudio server does not allow the recording", "id", opts.ID, "reason", reason)
		}
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the PulseAudio stream could not connect", "id", opts.ID, "reason", reason)
	}
	if got := s.lib.streamGetBufferAttr(d.stream); got != 0 {
		a := (*bufferAttr)(dl.Pointer(got))
		granted := a.tlength
		if kind == platform.AudioCapture {
			granted = a.fragsize
		}
		if g := int(granted) / frameBytes * frameBytes; g >= frameBytes {
			d.maxChunk = g
		}
	}
	s.devices[d] = struct{}{}
	return d, nil
}

// discardStream frees a stream that is not used any more, with the lock held.
func (s *Service) discardStream(d *device) {
	if d.stream == 0 {
		return
	}
	s.lib.streamSetStateCallback(d.stream, 0, 0)
	s.lib.streamSetWriteCallback(d.stream, 0, 0)
	s.lib.streamSetReadCallback(d.stream, 0, 0)
	s.lib.streamSetUnderflowCallback(d.stream, 0, 0)
	s.lib.streamDisconnect(d.stream)
	s.lib.streamUnref(d.stream)
	d.stream = 0
}

// release frees what a device holds outside of the stream.
func (s *Service) release(d *device) {
	removeHandle(d.handle)
	delete(s.devices, d)
	if d.scratch != 0 {
		dl.Free(d.scratch)
		d.scratch = 0
	}
}

// errAccess is PA_ERR_ACCESS, the error of a server that refuses.
const errAccess = 1
