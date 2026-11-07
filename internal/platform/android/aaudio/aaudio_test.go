//go:build linux

package aaudio

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// fakeAudio scripts libaaudio: the functions that the backend calls answer from its fields, and the streams it makes have a state that
// follows the requests, as a stream of the real library does (at once).
type fakeAudio struct {
	t  *testing.T
	mu sync.Mutex

	// What the next open gives, and what the device is like.
	openResult    int32
	nativeRate    int32
	nativeChans   int32
	forceRate     int32 // when not 0, the rate that a stream gets whatever it asked for
	forceChans    int32
	sampleFormat  int32
	burst         int32
	timestampOK   bool
	timestampPos  int64
	timestampTime int64
	xruns         int32

	// What the builder was given.
	builders  []*fakeBuilder
	nextBuild uintptr
	streams   map[uintptr]*fakeStream
	nextS     uintptr
	closed    []uintptr
}

type fakeBuilder struct {
	direction, sharing, format, perf, rate, channels, capacity, device int32
	usage, content, preset                                             int32
	dataCB, errCB, user                                                uintptr
	deleted                                                            bool
}

type fakeStream struct {
	id      uintptr
	state   int32
	builder *fakeBuilder
	rate    int32
	chans   int32
	bufSize int32
	starts  int
	pauses  int
	stops   int
	flushes int
}

func newFakeAudio(t *testing.T) *fakeAudio {
	f := &fakeAudio{t: t, nativeRate: 48000, nativeChans: 2, sampleFormat: formatPCMFloat, burst: 192, streams: map[uintptr]*fakeStream{}, nextBuild: 0x1000, nextS: 0x2000}
	return f
}

func (f *fakeAudio) builderOf(b uintptr) *fakeBuilder { return f.builders[b-0x1000-1] }

func (f *fakeAudio) streamOf(s uintptr) *fakeStream {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.streams[s]
}

func (f *fakeAudio) library() *library {
	store := func(out, v uintptr) { *(*uintptr)(dl.Pointer(out)) = v }
	return &library{
		createStreamBuilder: func(out uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.builders = append(f.builders, &fakeBuilder{})
			store(out, f.nextBuild+uintptr(len(f.builders)))
			return resultOK
		},
		builderDelete: func(b uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.builderOf(b).deleted = true
			return resultOK
		},
		builderSetDeviceID:               func(b uintptr, v int32) { f.builderOf(b).device = v },
		builderSetDirection:              func(b uintptr, v int32) { f.builderOf(b).direction = v },
		builderSetSharingMode:            func(b uintptr, v int32) { f.builderOf(b).sharing = v },
		builderSetSampleRate:             func(b uintptr, v int32) { f.builderOf(b).rate = v },
		builderSetChannelCount:           func(b uintptr, v int32) { f.builderOf(b).channels = v },
		builderSetFormat:                 func(b uintptr, v int32) { f.builderOf(b).format = v },
		builderSetPerformanceMode:        func(b uintptr, v int32) { f.builderOf(b).perf = v },
		builderSetBufferCapacityInFrames: func(b uintptr, v int32) { f.builderOf(b).capacity = v },
		builderSetFramesPerDataCallback:  func(b uintptr, v int32) {},
		builderSetDataCallback:           func(b, cb, user uintptr) { f.builderOf(b).dataCB, f.builderOf(b).user = cb, user },
		builderSetErrorCallback:          func(b, cb, user uintptr) { f.builderOf(b).errCB = cb },
		builderSetUsage:                  func(b uintptr, v int32) { f.builderOf(b).usage = v },
		builderSetContentType:            func(b uintptr, v int32) { f.builderOf(b).content = v },
		builderSetInputPreset:            func(b uintptr, v int32) { f.builderOf(b).preset = v },
		builderOpenStream: func(b, out uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.openResult != resultOK {
				r := f.openResult
				f.openResult = resultOK
				return r
			}
			bd := f.builderOf(b)
			s := &fakeStream{id: f.nextS, state: 2, builder: bd, rate: bd.rate, chans: bd.channels, bufSize: f.burst * 4}
			if s.rate == 0 {
				s.rate = f.nativeRate
			}
			if s.chans == 0 {
				s.chans = f.nativeChans
			}
			if f.forceRate != 0 {
				s.rate = f.forceRate
			}
			if f.forceChans != 0 {
				s.chans = f.forceChans
			}
			f.nextS++
			f.streams[s.id] = s
			store(out, s.id)
			return resultOK
		},
		streamClose: func(s uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.closed = append(f.closed, s)
			f.streams[s].state = 12
			return resultOK
		},
		streamRequestStart: func(s uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.streams[s].state, f.streams[s].starts = streamStateStarted, f.streams[s].starts+1
			return resultOK
		},
		streamRequestPause: func(s uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.streams[s].state, f.streams[s].pauses = streamStatePaused, f.streams[s].pauses+1
			return resultOK
		},
		streamRequestFlush: func(s uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.streams[s].flushes++
			return resultOK
		},
		streamRequestStop: func(s uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.streams[s].state, f.streams[s].stops = streamStateStopped, f.streams[s].stops+1
			return resultOK
		},
		streamGetState: func(s uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.streams[s].state
		},
		streamWaitForStateChange: func(s uintptr, in int32, next uintptr, timeout int64) int32 { return resultOK },
		streamGetSampleRate:      func(s uintptr) int32 { return f.streamOf(s).rate },
		streamGetChannelCount:    func(s uintptr) int32 { return f.streamOf(s).chans },
		streamGetFormat:          func(s uintptr) int32 { return f.sampleFormat },
		streamGetFramesPerBurst:  func(s uintptr) int32 { return f.burst },
		streamGetBufferSize:      func(s uintptr) int32 { return f.streamOf(s).bufSize },
		streamSetBufferSize: func(s uintptr, n int32) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.streams[s].bufSize = n
			return n
		},
		streamGetXRunCount: func(s uintptr) int32 { return f.xruns },
		streamGetTimestamp: func(s uintptr, clock int32, pos, nanos uintptr) int32 {
			if !f.timestampOK {
				return resultInvalidState
			}
			*(*int64)(dl.Pointer(pos)) = f.timestampPos
			*(*int64)(dl.Pointer(nanos)) = f.timestampTime
			return resultOK
		},
		convertResultToText: func(r int32) uintptr { return 0 },
	}
}

func (f *fakeAudio) service(perm Permissions) *Service {
	s := New(Options{Permissions: perm})
	s.load = func() (*library, error) { return f.library(), nil }
	return s
}

// handler records what the device does.
type handler struct {
	mu       sync.Mutex
	fills    int
	frames   int
	captured int
	states   []platform.AudioState
	panicOn  int
	value    float32
}

func (h *handler) Fill(out []float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fills++
	if h.panicOn > 0 && h.fills == h.panicOn {
		panic("the handler panics")
	}
	for i := range out {
		out[i] = h.value
	}
	h.frames += len(out)
}

func (h *handler) Capture(in []float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.captured += len(in)
}

func (h *handler) StateChanged(s platform.AudioState) {
	h.mu.Lock()
	h.states = append(h.states, s)
	h.mu.Unlock()
}

func (h *handler) stateList() []platform.AudioState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]platform.AudioState(nil), h.states...)
}

func (h *handler) fillCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.fills
}

// callData plays the part of AAudio's audio thread: it calls the data callback that the builder was given, with memory of the size of
// the frames, and returns the memory's content as float32.
func callData(t *testing.T, b *fakeBuilder, frames, channels int) ([]float32, int32) {
	t.Helper()
	mem := dl.Malloc(uintptr(frames * channels * 4))
	defer dl.Free(mem)
	// The values that the backend leaves in memory are what the handler wrote: start from a mark that no handler writes.
	buf := unsafeFloats(mem, frames*channels)
	for i := range buf {
		buf[i] = -7
	}
	r := dispatchData(b.user, mem, int32(frames))
	return append([]float32(nil), buf...), r
}

func openPlayback(t *testing.T, f *fakeAudio, opts platform.AudioOpenOptions, h *handler) (*Service, *device) {
	t.Helper()
	s := f.service(nil)
	dev, err := s.Open(context.Background(), platform.AudioPlayback, opts, h)
	if err != nil {
		t.Fatal(err)
	}
	d := dev.(*device)
	t.Cleanup(func() { _ = d.Close() })
	return s, d
}

func TestDevicesAreTheDefaultOnes(t *testing.T) {
	f := newFakeAudio(t)
	s := f.service(nil)
	for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
		list, err := s.Devices(kind)
		if err != nil || len(list) != 1 || !list[0].Default || list[0].Kind != kind || list[0].Name == "" {
			t.Errorf("kind %d: %+v, %v", kind, list, err)
		}
	}
	s.SetChangeHandler(func() { t.Error("the change handler was called") })
}

func TestLibraryMissing(t *testing.T) {
	t.Setenv(LibraryEnv, "")
	s := New(Options{})
	_, err := s.Devices(platform.AudioPlayback)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.getLib->loadLibrary.AudioUnavailable") {
		t.Fatalf("Devices: %v", err)
	}
	_, err = s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &handler{})
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.Open->Service.getLib->loadLibrary.AudioUnavailable") {
		t.Errorf("Open: %v", err)
	}
	t.Setenv(LibraryEnv, filepath.Join(t.TempDir(), "nothing.so"))
	if _, err := New(Options{}).Devices(platform.AudioPlayback); err == nil {
		t.Error("an override that is not a library was loaded")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable {
		t.Errorf("override: %v", err)
	}
}

func TestOpenPlaybackSetsTheStreamUp(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{value: 0.25}
	_, d := openPlayback(t, f, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 44100, Channels: 1}}, h)
	b := f.builders[0]
	if b.direction != directionOutput || b.sharing != sharingModeShared || b.format != formatPCMFloat || b.perf != performanceModeLowLatency {
		t.Errorf("the builder: %+v", b)
	}
	if b.rate != 44100 || b.channels != 1 || b.device != 0 || b.usage != usageGame || b.content != contentTypeMusic {
		t.Errorf("the builder: %+v", b)
	}
	if !b.deleted {
		t.Error("the builder was not deleted")
	}
	if got := d.Format(); got != (platform.AudioFormat{Rate: 44100, Channels: 1}) {
		t.Errorf("format %+v", got)
	}
	s := f.streamOf(0x2000)
	if s.starts != 1 || s.state != streamStateStarted {
		t.Errorf("the stream did not start: %+v", s)
	}
	if want := f.burst * 2; s.bufSize != want {
		t.Errorf("the buffer is %d frames, want two bursts (%d)", s.bufSize, want)
	}
	out, r := callData(t, b, 96, 1)
	if r != callbackContinue {
		t.Errorf("the callback answered %d", r)
	}
	for i, v := range out {
		if v != 0.25 {
			t.Fatalf("sample %d is %v, want what the handler wrote", i, v)
		}
	}
}

func TestOpenTakesTheDevicesOwnFormatWhenAskedForNone(t *testing.T) {
	f := newFakeAudio(t)
	f.nativeRate, f.nativeChans = 44100, 1
	_, d := openPlayback(t, f, platform.AudioOpenOptions{}, &handler{})
	if got := d.Format(); got != (platform.AudioFormat{Rate: 44100, Channels: 1}) {
		t.Errorf("format %+v", got)
	}
	if b := f.builders[0]; b.rate != unspecified || b.channels != unspecified {
		t.Errorf("the builder was given a rate or channels: %+v", b)
	}
}

func TestBufferFramesAreTheBuffer(t *testing.T) {
	tests := []struct {
		frames, wantCapacity, wantSize int32
	}{{0, 0, 384}, {128, 128, 384}, {1000, 1000, 1000}}
	for _, tt := range tests {
		f := newFakeAudio(t)
		_, d := openPlayback(t, f, platform.AudioOpenOptions{BufferFrames: int(tt.frames)}, &handler{})
		if b := f.builders[0]; b.capacity != tt.wantCapacity {
			t.Errorf("BufferFrames %d: capacity %d, want %d", tt.frames, b.capacity, tt.wantCapacity)
		}
		if buffer, burst := d.BufferFrames(); buffer != int(tt.wantSize) || burst != int(f.burst) {
			t.Errorf("BufferFrames %d: buffer %d, burst %d, want %d and %d", tt.frames, buffer, burst, tt.wantSize, f.burst)
		}
	}
}

func TestOpenByID(t *testing.T) {
	f := newFakeAudio(t)
	s := f.service(nil)
	dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{ID: "42"}, &handler{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dev.Close() }() // the test is over: nothing to report
	if f.builders[0].device != 42 {
		t.Errorf("the device is %d", f.builders[0].device)
	}
	for _, id := range []string{"abc", "-3", "0", "4x", "99999999999999"} {
		_, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{ID: id}, &handler{})
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceOpenFailed || !appErr.HasRoute("Service.Open.AudioDeviceOpenFailed") {
			t.Errorf("ID %q: %v", id, err)
		}
	}
}

func TestParseDeviceID(t *testing.T) {
	tests := []struct {
		id   string
		want int32
		ok   bool
	}{{"", 0, true}, {"7", 7, true}, {"1234", 1234, true}, {"0", 0, false}, {"x", 0, false}, {"1.5", 0, false}}
	for _, tt := range tests {
		if got, ok := parseDeviceID(tt.id); got != tt.want || ok != tt.ok {
			t.Errorf("parseDeviceID(%q) = %d, %v", tt.id, got, ok)
		}
	}
}

func TestOpenErrors(t *testing.T) {
	tests := []struct {
		name   string
		result int32
		format int32
		tag    string
		route  string
	}{
		{"an invalid rate", resultInvalidRate, formatPCMFloat, platform.TagAudioFormatUnsupported, "Service.Open->Service.openStream->resultError.AudioFormatUnsupported"},
		{"an invalid format", resultInvalidFormat, formatPCMFloat, platform.TagAudioFormatUnsupported, "resultError.AudioFormatUnsupported"},
		{"an argument out of range", resultOutOfRange, formatPCMFloat, platform.TagAudioFormatUnsupported, "resultError.AudioFormatUnsupported"},
		{"the service is gone", resultNoService, formatPCMFloat, platform.TagAudioDeviceOpenFailed, "Service.Open->Service.openStream->resultError.AudioDeviceOpenFailed"},
		{"no free handles", resultNoFreeHandles, formatPCMFloat, platform.TagAudioDeviceOpenFailed, "resultError.AudioDeviceOpenFailed"},
		{"a stream that is not float", resultOK, 1, platform.TagAudioFormatUnsupported, "Service.Open->Service.openStream.AudioFormatUnsupported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeAudio(t)
			f.openResult, f.sampleFormat = tt.result, tt.format
			_, err := f.service(nil).Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &handler{})
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Fatalf("got %v, want tag %s with route %s", err, tt.tag, tt.route)
			}
			if tt.result != resultOK && appErr.Metadata["result"] != float64(tt.result) && appErr.Metadata["result"] != int(tt.result) {
				t.Errorf("the metadata is %v", appErr.Metadata)
			}
			if tt.format != formatPCMFloat && len(f.closed) != 1 {
				t.Errorf("the stream that did not suit was not closed: %v", f.closed)
			}
			if len(handleTable.devices) != 0 {
				t.Errorf("a device that failed to open stays in the table: %d", len(handleTable.devices))
			}
		})
	}
	t.Run("a format out of the limits", func(t *testing.T) {
		f := newFakeAudio(t)
		for _, fm := range []platform.AudioFormat{{Rate: 5}, {Channels: 9}, {Rate: -1}, {Rate: 900000, Channels: 2}} {
			_, err := f.service(nil).Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: fm}, &handler{})
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioFormatUnsupported || !appErr.HasRoute("Service.Open.AudioFormatUnsupported") {
				t.Errorf("%+v: %v", fm, err)
			}
		}
		if len(f.builders) != 0 {
			t.Error("a builder was made for a format that is out of the limits")
		}
	})
	t.Run("a context that is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := newFakeAudio(t).service(nil).Open(ctx, platform.AudioPlayback, platform.AudioOpenOptions{}, &handler{})
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceOpenFailed || !appErr.HasRoute("Service.Open.AudioDeviceOpenFailed") {
			t.Errorf("got %v", err)
		}
	})
}

type fakePermissions struct {
	granted bool
	err     error
	asked   int
}

func (p *fakePermissions) RecordAudio(ctx context.Context) (bool, error) {
	p.asked++
	return p.granted, p.err
}

func TestCapture(t *testing.T) {
	t.Run("the permission is granted", func(t *testing.T) {
		f := newFakeAudio(t)
		perm := &fakePermissions{granted: true}
		h := &handler{}
		dev, err := f.service(perm).Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, h)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = dev.Close() }() // the test is over: nothing to report
		if perm.asked != 1 {
			t.Errorf("the permission was asked %d times", perm.asked)
		}
		b := f.builders[0]
		if b.direction != directionInput || b.preset != inputPresetGeneric || b.perf != performanceModeLowLatency {
			t.Errorf("the builder: %+v", b)
		}
		callData(t, b, 100, 2)
		if h.captured != 200 {
			t.Errorf("the handler got %d samples, want 200", h.captured)
		}
		if s := f.streamOf(0x2000); s.bufSize != f.burst*4 {
			t.Errorf("the buffer of a recording is not set: %d", s.bufSize)
		}
	})
	t.Run("the user refuses", func(t *testing.T) {
		f := newFakeAudio(t)
		_, err := f.service(&fakePermissions{}).Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, &handler{})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != platform.TagAudioCapturePermissionDenied || !appErr.HasRoute("Service.Open.AudioCapturePermissionDenied") {
			t.Fatalf("got %v", err)
		}
		if appErr.Metadata["permission"] != "android.permission.RECORD_AUDIO" {
			t.Errorf("the metadata is %v", appErr.Metadata)
		}
		if len(f.builders) != 0 {
			t.Error("a stream was opened without the permission")
		}
	})
	t.Run("the permission cannot be asked", func(t *testing.T) {
		f := newFakeAudio(t)
		_, err := f.service(&fakePermissions{err: os.ErrDeadlineExceeded}).Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, &handler{})
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioCapturePermissionDenied {
			t.Errorf("got %v", err)
		}
	})
	t.Run("playback never asks", func(t *testing.T) {
		f := newFakeAudio(t)
		perm := &fakePermissions{}
		dev, err := f.service(perm).Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &handler{})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = dev.Close() }() // the test is over: nothing to report
		if perm.asked != 0 {
			t.Error("a playback device asked for the microphone")
		}
	})
}

func TestPauseAndResume(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{value: 1}
	_, d := openPlayback(t, f, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, h)
	b := f.builders[0]
	if err := d.Pause(); err != nil {
		t.Fatal(err)
	}
	s := f.streamOf(0x2000)
	if s.pauses != 1 || s.flushes != 1 || s.state != streamStatePaused {
		t.Errorf("a playback is paused and flushed: %+v", s)
	}
	// A callback that was already on its way gets silence, not the handler.
	fills := h.fillCount()
	out, _ := callData(t, b, 10, 2)
	for _, v := range out {
		if v != 0 {
			t.Fatalf("a paused stream played %v", v)
		}
	}
	if h.fillCount() != fills {
		t.Error("the handler was called while paused")
	}
	if err := d.Pause(); err != nil || s.pauses != 1 {
		t.Errorf("a second Pause: %v, %d pauses", err, s.pauses)
	}
	if err := d.Resume(); err != nil {
		t.Fatal(err)
	}
	if s.starts != 2 || s.state != streamStateStarted {
		t.Errorf("Resume did not start the stream: %+v", s)
	}
	out, _ = callData(t, b, 10, 2)
	if out[0] != 1 {
		t.Errorf("a stream that went on played %v", out[0])
	}
	if err := d.Resume(); err != nil || s.starts != 2 {
		t.Errorf("a second Resume: %v, %d starts", err, s.starts)
	}
}

func TestCaptureIsStoppedNotPaused(t *testing.T) {
	f := newFakeAudio(t)
	s := f.service(nil)
	h := &handler{}
	dev, err := s.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, h)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dev.Close() }() // the test is over: nothing to report
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	st := f.streamOf(0x2000)
	if st.stops != 1 || st.pauses != 0 {
		t.Errorf("a recording is stopped: %+v", st)
	}
	callData(t, f.builders[0], 10, 2)
	if h.captured != 0 {
		t.Error("a stopped recording gave frames")
	}
	if err := dev.Resume(); err != nil || st.starts != 2 {
		t.Errorf("Resume: %v, %+v", err, st)
	}
}

func TestClose(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{}
	_, d := openPlayback(t, f, platform.AudioOpenOptions{}, h)
	b := f.builders[0]
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if len(f.closed) != 1 || f.streamOf(0x2000).stops != 1 {
		t.Errorf("the stream was not stopped and closed: %v", f.closed)
	}
	if err := d.Close(); err != nil {
		t.Errorf("a second Close: %v", err)
	}
	if len(f.closed) != 1 {
		t.Errorf("the stream was closed twice: %v", f.closed)
	}
	// A callback that comes after Close finds no device.
	if r := dispatchData(b.user, 0, 10); r != callbackStop {
		t.Errorf("the callback of a closed device answered %d", r)
	}
	for _, call := range []func() error{d.Pause, d.Resume} {
		err := call()
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.") {
			t.Errorf("a call on a closed device: %v", err)
		}
	}
	if len(handleTable.devices) != 0 {
		t.Errorf("the closed device is still in the table")
	}
}

func TestDisconnectMakesTheStreamAgain(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{value: 0.5}
	_, d := openPlayback(t, f, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 44100, Channels: 2}}, h)
	first := f.builders[0]
	old := d.stream.Load()
	// A late callback of another stream, or an error that is not a disconnect, does nothing.
	d.onError(old+99, resultDisconnected)
	d.onError(old, resultInternal)
	time.Sleep(20 * time.Millisecond)
	if len(f.builders) != 1 {
		t.Fatal("the stream was made again for something that is not a disconnect")
	}
	f.nativeRate = 48000 // the new device would like another rate: the stream keeps the one that the handler has
	d.onError(old, resultDisconnected)
	waitFor(t, "the stream to be made again", func() bool { return d.stream.Load() != old && d.stream.Load() != 0 })
	if len(f.builders) != 2 {
		t.Fatalf("%d builders", len(f.builders))
	}
	if b := f.builders[1]; b.rate != 44100 || b.channels != 2 || b.device != 0 || b.user != first.user {
		t.Errorf("the new builder: %+v", b)
	}
	if f.streamOf(old).state != 12 {
		t.Error("the old stream was not closed")
	}
	if s := f.streamOf(d.stream.Load()); s.starts != 1 {
		t.Errorf("the new stream did not start: %+v", s)
	}
	if got := h.stateList(); len(got) != 0 {
		t.Errorf("the handler was told %v although the stream was made again", got)
	}
	// The error of the old stream that comes late is ignored.
	d.onError(old, resultDisconnected)
	time.Sleep(20 * time.Millisecond)
	if len(f.builders) != 2 {
		t.Error("the late error of the old stream made the stream again")
	}
	out, _ := callData(t, f.builders[1], 4, 2)
	if out[0] != 0.5 {
		t.Errorf("the new stream does not play: %v", out[0])
	}
}

func TestDisconnectOfAPinnedDeviceIsLost(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{}
	s := f.service(nil)
	dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{ID: "5"}, h)
	if err != nil {
		t.Fatal(err)
	}
	d := dev.(*device)
	defer func() { _ = d.Close() }() // the test is over: nothing to report
	d.onError(d.stream.Load(), resultDisconnected)
	waitFor(t, "the device to be lost", func() bool { return len(h.stateList()) == 1 })
	if got := h.stateList(); got[0] != platform.AudioLost {
		t.Errorf("states %v", got)
	}
	if len(f.builders) != 1 {
		t.Error("a pinned device was made again on another device")
	}
	if err := d.Pause(); err == nil {
		t.Error("Pause of a lost device did not fail")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.Pause.AudioDeviceLost") {
		t.Errorf("Pause: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Errorf("Close of a lost device: %v", err)
	}
}

func TestDisconnectThatCannotBeFollowedIsLost(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeAudio)
	}{
		{"the new device does not open", func(f *fakeAudio) { f.openResult = resultNoService }},
		{"the new device gives another rate", func(f *fakeAudio) { f.forceRate = 8000 }},
		{"the new device gives another number of channels", func(f *fakeAudio) { f.forceChans = 6 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeAudio(t)
			h := &handler{}
			_, d := openPlayback(t, f, platform.AudioOpenOptions{}, h)
			old := d.stream.Load()
			tt.setup(f)
			d.onError(old, resultDisconnected)
			waitFor(t, "the device to be lost", func() bool { return len(h.stateList()) >= 1 })
			if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioLost {
				t.Errorf("states %v", got)
			}
			if len(f.closed) != 2 && tt.name != "the new device does not open" {
				t.Errorf("the streams that were closed: %v", f.closed)
			}
		})
	}
}

func TestBackground(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{value: 1}
	hOff := &handler{value: 1}
	s := f.service(nil)
	devOn, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{PauseInBackground: true}, h)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devOn.Close() }() // the test is over: nothing to report
	devOff, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, hOff)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devOff.Close() }() // the test is over: nothing to report
	on, off := f.streamOf(0x2000), f.streamOf(0x2001)
	s.SetBackground(true)
	if on.state != streamStatePaused || off.state != streamStateStarted {
		t.Errorf("states %d and %d after the app was hidden", on.state, off.state)
	}
	if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioPaused {
		t.Errorf("the handler was told %v", got)
	}
	s.SetBackground(true) // nothing changes
	if len(h.stateList()) != 1 {
		t.Error("a second hide told the handler again")
	}
	s.SetBackground(false)
	if on.state != streamStateStarted {
		t.Errorf("the stream did not go on: %d", on.state)
	}
	if got := h.stateList(); len(got) != 2 || got[1] != platform.AudioRunning {
		t.Errorf("the handler was told %v", got)
	}
	if len(hOff.stateList()) != 0 {
		t.Errorf("a device that does not pause in the background was told %v", hOff.stateList())
	}

	// A device that its owner paused stays paused when the app is shown, and is not told anything.
	if err := devOn.Pause(); err != nil {
		t.Fatal(err)
	}
	s.SetBackground(true)
	s.SetBackground(false)
	if on.state != streamStatePaused {
		t.Errorf("the owner's pause was undone: %d", on.state)
	}
	if len(h.stateList()) != 2 {
		t.Errorf("the handler was told %v about a device its owner paused", h.stateList())
	}
	if err := devOn.Resume(); err != nil || on.state != streamStateStarted {
		t.Errorf("Resume: %v, %d", err, on.state)
	}

	// A device opened while the app is hidden starts paused.
	s.SetBackground(true)
	late, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{PauseInBackground: true}, &handler{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = late.Close() }() // the test is over: nothing to report
	if st := f.streamOf(0x2002); st.starts != 0 {
		t.Errorf("a device opened in the background started: %+v", st)
	}
	s.SetBackground(false)
	if st := f.streamOf(0x2002); st.starts != 1 {
		t.Errorf("it did not start when the app was shown: %+v", st)
	}
}

func TestPanicInTheHandlerLosesTheDevice(t *testing.T) {
	f := newFakeAudio(t)
	h := &handler{panicOn: 2}
	_, d := openPlayback(t, f, platform.AudioOpenOptions{}, h)
	b := f.builders[0]
	if _, r := callData(t, b, 4, 2); r != callbackContinue {
		t.Fatalf("the first call answered %d", r)
	}
	if _, r := callData(t, b, 4, 2); r != callbackStop {
		t.Errorf("a panic did not stop the stream: %d", r)
	}
	waitFor(t, "the device to be lost", func() bool { return len(h.stateList()) == 1 })
	if got := h.stateList(); got[0] != platform.AudioLost {
		t.Errorf("states %v", got)
	}
	if d.inflight.Load() != 0 {
		t.Errorf("%d calls are still counted as running", d.inflight.Load())
	}
}

func TestTimestampAndXRuns(t *testing.T) {
	f := newFakeAudio(t)
	_, d := openPlayback(t, f, platform.AudioOpenOptions{}, &handler{})
	if _, _, ok := d.Timestamp(); ok {
		t.Error("a timestamp that the stream does not have yet")
	}
	f.timestampOK, f.timestampPos, f.timestampTime, f.xruns = true, 4800, 123456789, 3
	frame, nanos, ok := d.Timestamp()
	if !ok || frame != 4800 || nanos != 123456789 {
		t.Errorf("timestamp %d %d %v", frame, nanos, ok)
	}
	if d.XRuns() != 3 {
		t.Errorf("xruns %d", d.XRuns())
	}
	f.xruns = -1
	if d.XRuns() != 0 {
		t.Error("an error is not a count")
	}
}

func TestCallbacksAreMadeOnce(t *testing.T) {
	before := dl.CallbackCount()
	a := callbacks()
	b := callbacks()
	if a != b {
		t.Error("the callbacks were made again")
	}
	if n := dl.CallbackCount() - before; n > 2 {
		t.Errorf("%d callbacks were made, want at most 2", n)
	}
	f := newFakeAudio(t)
	for range 3 {
		_, d := openPlayback(t, f, platform.AudioOpenOptions{}, &handler{})
		_ = d.Close()
	}
	if n := dl.CallbackCount() - before; n > 2 {
		t.Errorf("opening devices made callbacks: %d", n)
	}
}

func TestResultError(t *testing.T) {
	f := newFakeAudio(t)
	lib := f.library()
	tests := []struct {
		result int32
		tag    string
	}{
		{resultInvalidRate, platform.TagAudioFormatUnsupported},
		{resultInvalidFormat, platform.TagAudioFormatUnsupported},
		{resultOutOfRange, platform.TagAudioFormatUnsupported},
		{resultIllegalArgument, platform.TagAudioFormatUnsupported},
		{resultNoService, platform.TagAudioDeviceOpenFailed},
		{resultUnavailable, platform.TagAudioDeviceOpenFailed},
		{resultInternal, platform.TagAudioDeviceOpenFailed},
		{resultNoMemory, platform.TagAudioDeviceOpenFailed},
		{resultTimeout, platform.TagAudioDeviceOpenFailed},
	}
	for _, tt := range tests {
		err := resultError(lib, "AAudioStreamBuilder_openStream", tt.result)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tt.tag || !appErr.HasRoute("resultError."+tt.tag) {
			t.Errorf("%d: %v, want %s", tt.result, err, tt.tag)
		}
	}
}

// TestConstantsAgainstTheHeaders compiles a check of the numbers of this package against the NDK's <aaudio/AAudio.h> with the
// compiler of the NDK (it only parses, for Android, and runs nothing). It is skipped when the NDK is not installed.
func TestConstantsAgainstTheHeaders(t *testing.T) {
	home := os.Getenv("ANDROID_HOME")
	if home == "" {
		t.Skip("ANDROID_HOME is not set")
	}
	matches, err := filepath.Glob(filepath.Join(home, "ndk", "*", "toolchains", "llvm", "prebuilt", "*", "bin", "aarch64-linux-android26-clang"))
	if err != nil || len(matches) == 0 {
		t.Skip("the NDK is not installed")
	}
	checks := map[string]int{
		"AAUDIO_UNSPECIFIED": unspecified, "AAUDIO_DIRECTION_OUTPUT": directionOutput, "AAUDIO_DIRECTION_INPUT": directionInput,
		"AAUDIO_FORMAT_PCM_FLOAT": formatPCMFloat, "AAUDIO_SHARING_MODE_SHARED": sharingModeShared,
		"AAUDIO_PERFORMANCE_MODE_LOW_LATENCY": performanceModeLowLatency, "AAUDIO_USAGE_GAME": usageGame,
		"AAUDIO_CONTENT_TYPE_MUSIC":   contentTypeMusic,
		"AAUDIO_INPUT_PRESET_GENERIC": inputPresetGeneric, "AAUDIO_CALLBACK_RESULT_CONTINUE": callbackContinue,
		"AAUDIO_CALLBACK_RESULT_STOP": callbackStop, "AAUDIO_STREAM_STATE_STARTING": streamStateStarting,
		"AAUDIO_STREAM_STATE_STARTED": streamStateStarted, "AAUDIO_STREAM_STATE_PAUSING": streamStatePausing,
		"AAUDIO_STREAM_STATE_PAUSED": streamStatePaused, "AAUDIO_STREAM_STATE_STOPPING": streamStateStopping,
		"AAUDIO_STREAM_STATE_STOPPED": streamStateStopped, "AAUDIO_STREAM_STATE_DISCONNECTED": streamStateDisconnected,
		"AAUDIO_OK": resultOK, "AAUDIO_ERROR_DISCONNECTED": resultDisconnected, "AAUDIO_ERROR_ILLEGAL_ARGUMENT": resultIllegalArgument,
		"AAUDIO_ERROR_INTERNAL": resultInternal, "AAUDIO_ERROR_INVALID_STATE": resultInvalidState,
		"AAUDIO_ERROR_UNAVAILABLE": resultUnavailable, "AAUDIO_ERROR_NO_FREE_HANDLES": resultNoFreeHandles,
		"AAUDIO_ERROR_NO_MEMORY": resultNoMemory, "AAUDIO_ERROR_TIMEOUT": resultTimeout,
		"AAUDIO_ERROR_INVALID_FORMAT": resultInvalidFormat, "AAUDIO_ERROR_OUT_OF_RANGE": resultOutOfRange,
		"AAUDIO_ERROR_NO_SERVICE": resultNoService, "AAUDIO_ERROR_INVALID_RATE": resultInvalidRate,
		"CLOCK_MONOTONIC": clockMonotonic,
	}
	var src strings.Builder
	src.WriteString("#include <aaudio/AAudio.h>\n#include <time.h>\n")
	for name, v := range checks {
		src.WriteString("_Static_assert((int)(" + name + ") == " + strconv.Itoa(v) + ", \"" + name + "\");\n")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "check.c")
	if err := os.WriteFile(file, []byte(src.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(matches[0], "-fsyntax-only", file).CombinedOutput(); err != nil {
		t.Errorf("the headers of the NDK disagree:\n%s", out)
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}
