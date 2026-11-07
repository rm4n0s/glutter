//go:build linux && !android

package alsa

import (
	"context"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// recorder is a handler that keeps what it is given.
type recorder struct {
	mu       sync.Mutex
	fills    []int // the frames of each Fill
	captured []float32
	states   []platform.AudioState
}

func (r *recorder) Fill(out []float32) {
	r.mu.Lock()
	r.fills = append(r.fills, len(out))
	r.mu.Unlock()
	for i := range out {
		out[i] = 0.25
	}
}

func (r *recorder) Capture(in []float32) {
	r.mu.Lock()
	r.captured = append(r.captured, in...)
	r.mu.Unlock()
}

func (r *recorder) StateChanged(s platform.AudioState) {
	r.mu.Lock()
	r.states = append(r.states, s)
	r.mu.Unlock()
}

func (r *recorder) fillCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.fills)
}

func (r *recorder) stateList() []platform.AudioState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]platform.AudioState(nil), r.states...)
}

// fakePCM scripts libasound for one PCM. The goroutine of the device and the test both reach it, so a mutex guards it.
type fakePCM struct {
	mu          sync.Mutex
	openRC      int32
	paramsRC    int32
	buffer      uint64
	period      uint64
	avail       int64
	waitRC      int32
	writeRC     []int64 // what the next writei calls return before they succeed; an entry of 0 means all the frames
	readRC      []int64
	written     []float32
	recoverRC   int32
	recovers    []int32
	drops       int
	prepares    int
	starts      int
	closes      int
	state       int32
	opened      []string
	params      [4]uint32
	hints       []fakeHint
	text        uintptr
	captureData float32
}

type fakeHint struct{ name, desc, ioid string }

func newFakePCM(t *testing.T) *fakePCM {
	f := &fakePCM{buffer: 960, period: 480, avail: 480, waitRC: 1}
	f.text = dl.CString("fake error text")
	t.Cleanup(func() { dl.Free(f.text) })
	return f
}

func (f *fakePCM) library() *library {
	return &library{
		open: func(pcmp uintptr, name uintptr, stream, mode int32) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.opened = append(f.opened, dl.GoString(name))
			*(*uintptr)(dl.Pointer(pcmp)) = 1
			return f.openRC
		},
		close: func(uintptr) int32 { f.mu.Lock(); f.closes++; f.mu.Unlock(); return 0 },
		setParams: func(_ uintptr, format, access int32, channels, rate uint32, soft int32, latency uint32) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.params = [4]uint32{channels, rate, uint32(soft), latency}
			return f.paramsRC
		},
		getParams: func(_ uintptr, buffer, period uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			*(*uint64)(dl.Pointer(buffer)) = f.buffer
			*(*uint64)(dl.Pointer(period)) = f.period
			return 0
		},
		wait: func(uintptr, int32) int32 {
			time.Sleep(time.Millisecond)
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.waitRC
		},
		availUpdate: func(uintptr) int64 { f.mu.Lock(); defer f.mu.Unlock(); return f.avail },
		writei: func(_ uintptr, buf unsafe.Pointer, frames uint64) int64 {
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.writeRC) > 0 {
				rc := f.writeRC[0]
				f.writeRC = f.writeRC[1:]
				if rc != 0 {
					return rc
				}
			}
			f.written = append(f.written, unsafe.Slice((*float32)(buf), int(frames)*2)...)
			return int64(frames)
		},
		readi: func(_ uintptr, buf unsafe.Pointer, frames uint64) int64 {
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.readRC) > 0 {
				rc := f.readRC[0]
				f.readRC = f.readRC[1:]
				if rc != 0 {
					return rc
				}
			}
			out := unsafe.Slice((*float32)(buf), int(frames)*2)
			for i := range out {
				out[i] = f.captureData
			}
			return int64(frames)
		},
		recover: func(_ uintptr, err, silent int32) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.recovers = append(f.recovers, err)
			return f.recoverRC
		},
		prepare:  func(uintptr) int32 { f.mu.Lock(); f.prepares++; f.mu.Unlock(); return 0 },
		start:    func(uintptr) int32 { f.mu.Lock(); f.starts++; f.mu.Unlock(); return 0 },
		drop:     func(uintptr) int32 { f.mu.Lock(); f.drops++; f.mu.Unlock(); return 0 },
		state:    func(uintptr) int32 { f.mu.Lock(); defer f.mu.Unlock(); return f.state },
		strerror: func(int32) uintptr { return f.text },
		nameHint: func(card int32, iface uintptr, hints uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			arr := dl.Calloc(uintptr(len(f.hints)+1), 8)
			for i := range f.hints {
				// A hint is opaque to the program; here it is the index plus one.
				*(*uintptr)(dl.Pointer(arr + uintptr(i)*8)) = uintptr(i + 1)
			}
			*(*uintptr)(dl.Pointer(hints)) = arr
			return 0
		},
		nameGetHint: func(hint uintptr, id uintptr) uintptr {
			f.mu.Lock()
			defer f.mu.Unlock()
			h := f.hints[hint-1]
			var v string
			switch dl.GoString(id) {
			case "NAME":
				v = h.name
			case "DESC":
				v = h.desc
			case "IOID":
				if h.ioid == "" {
					return 0
				}
				v = h.ioid
			}
			return dl.CString(v)
		},
		nameFreeHint: func(hints uintptr) int32 { dl.Free(hints); return 0 },
		free:         dl.Free,
	}
}

func (f *fakePCM) service() *Service {
	s := New()
	s.load = func() (*library, error) { return f.library(), nil }
	return s
}

func (f *fakePCM) set(fn func()) {
	f.mu.Lock()
	fn()
	f.mu.Unlock()
}

// eventually waits until cond is true.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

var stereo = platform.AudioFormat{Rate: 48000, Channels: 2}

func TestPlayback(t *testing.T) {
	f := newFakePCM(t)
	h := &recorder{}
	dev, err := f.service().Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: stereo, BufferFrames: 960}, h)
	if err != nil {
		t.Fatal(err)
	}
	if dev.Format() != stereo {
		t.Errorf("format %v", dev.Format())
	}
	eventually(t, "the first fills", func() bool { return h.fillCount() >= 3 })
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	n := h.fillCount()
	time.Sleep(20 * time.Millisecond)
	if h.fillCount() != n {
		t.Error("the handler was called after Close returned")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range h.fills[:3] {
		if n != 480*2 {
			t.Errorf("a fill of %d samples, want the 480 frames that are free", n)
		}
	}
	if len(f.written) < 3*960 || f.written[0] != 0.25 {
		t.Errorf("the device got %d samples starting with %v", len(f.written), f.written[:1])
	}
	if f.params != [4]uint32{2, 48000, 1, 20000} {
		t.Errorf("parameters (channels, rate, soft resample, latency) %v", f.params)
	}
	if f.closes != 1 || f.opened[0] != "default" {
		t.Errorf("closes %d, opened %v", f.closes, f.opened)
	}
}

func TestBufferFramesSetTheLatency(t *testing.T) {
	f := newFakePCM(t)
	dev, err := f.service().Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{ID: "plughw:1", Format: platform.AudioFormat{Rate: 44100, Channels: 2}, BufferFrames: 441}, &recorder{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.params[3] != 10000 || f.opened[0] != "plughw:1" {
		t.Errorf("latency %d µs, opened %v, want 10000 and plughw:1", f.params[3], f.opened)
	}
}

func TestUnderrunIsRecovered(t *testing.T) {
	f := newFakePCM(t)
	f.set(func() { f.writeRC = []int64{-errPipe} })
	h := &recorder{}
	dev, err := f.service().Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: stereo}, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	eventually(t, "the fills after the underrun", func() bool { return h.fillCount() >= 3 })
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.recovers) != 1 || f.recovers[0] != -errPipe {
		t.Errorf("recover calls %v, want one with -EPIPE", f.recovers)
	}
	if len(h.stateList()) != 0 {
		t.Errorf("states %v: an underrun is not a loss", h.stateList())
	}
}

func TestWaitErrorIsRecovered(t *testing.T) {
	f := newFakePCM(t)
	f.set(func() { f.waitRC = -errPipe })
	h := &recorder{}
	dev, err := f.service().Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: stereo}, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	eventually(t, "a recover call", func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.recovers) >= 2 })
}

func TestDeviceLost(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakePCM)
	}{
		{"unplugged: the write says no device", func(f *fakePCM) { f.writeRC = []int64{-errNoDev} }},
		{"the recovery fails", func(f *fakePCM) { f.writeRC = []int64{-errPipe}; f.recoverRC = -errInval }},
		{"the state is disconnected", func(f *fakePCM) { f.waitRC = -errPipe; f.state = stateDisconnected }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakePCM(t)
			f.set(func() { tt.setup(f) })
			h := &recorder{}
			dev, err := f.service().Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: stereo}, h)
			if err != nil {
				t.Fatal(err)
			}
			eventually(t, "the loss", func() bool { return len(h.stateList()) > 0 })
			time.Sleep(10 * time.Millisecond)
			if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioLost {
				t.Errorf("states %v, want a single AudioLost", got)
			}
			err = dev.Pause()
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.Pause.AudioDeviceLost") {
				t.Errorf("Pause: %v", err)
			}
			err = dev.Resume()
			appErr, ok = errors.FromError(err)
			if !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.Resume.AudioDeviceLost") {
				t.Errorf("Resume: %v", err)
			}
			if err := dev.Close(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestPauseAndResume(t *testing.T) {
	f := newFakePCM(t)
	h := &recorder{}
	dev, err := f.service().Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: stereo}, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	eventually(t, "fills", func() bool { return h.fillCount() > 2 })
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	n := h.fillCount()
	time.Sleep(30 * time.Millisecond)
	if h.fillCount() != n {
		t.Error("the handler was called while paused")
	}
	f.mu.Lock()
	drops := f.drops
	f.mu.Unlock()
	if drops != 1 {
		t.Errorf("%d drops, want the PCM stopped once", drops)
	}
	if err := dev.Pause(); err != nil { // pausing twice is fine
		t.Fatal(err)
	}
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "fills after the resume", func() bool { return h.fillCount() > n+2 })
	f.mu.Lock()
	if f.prepares != 1 {
		t.Errorf("%d prepares, want one after the pause", f.prepares)
	}
	f.mu.Unlock()
	// Closing a paused device does not hang.
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Close(); err != nil {
		t.Fatal("a second Close:", err)
	}
	if err := dev.Pause(); err != nil {
		t.Fatal("Pause after Close:", err)
	}
}

func TestCapture(t *testing.T) {
	f := newFakePCM(t)
	f.set(func() { f.captureData = 0.5; f.readRC = []int64{-errAgain, -errPipe} })
	h := &recorder{}
	dev, err := f.service().Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{Format: stereo}, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	eventually(t, "captured frames", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return len(h.captured) >= 4*480
	})
	h.mu.Lock()
	for i, v := range h.captured {
		if v != 0.5 {
			t.Fatalf("sample %d is %v", i, v)
		}
	}
	h.mu.Unlock()
	f.mu.Lock()
	defer f.mu.Unlock()
	// Started when opened, and again after the overrun that readi reported.
	if f.starts != 2 || len(f.recovers) != 1 || f.recovers[0] != -errPipe {
		t.Errorf("starts %d, recovers %v", f.starts, f.recovers)
	}
}

func TestOpenErrors(t *testing.T) {
	canceled := func() context.Context {
		c, cancel := context.WithCancel(context.Background())
		cancel()
		return c
	}
	tests := []struct {
		name  string
		setup func(f *fakePCM)
		kind  platform.AudioKind
		opts  platform.AudioOpenOptions
		ctx   func() context.Context
		tag   string
		route string
	}{
		{"no such device", func(f *fakePCM) { f.openRC = -2 }, platform.AudioPlayback, platform.AudioOpenOptions{ID: "hw:9"}, nil,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"the microphone is refused", func(f *fakePCM) { f.openRC = -errAccess }, platform.AudioCapture, platform.AudioOpenOptions{}, nil,
			platform.TagAudioCapturePermissionDenied, "Service.Open.AudioCapturePermissionDenied"},
		{"the speaker is refused", func(f *fakePCM) { f.openRC = -errAccess }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"a format that the device does not take", func(f *fakePCM) { f.paramsRC = -errInval }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioFormatUnsupported, "Service.Open.AudioFormatUnsupported"},
		{"parameters that fail", func(f *fakePCM) { f.paramsRC = -5 }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"too many channels", func(f *fakePCM) {}, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 9}}, nil,
			platform.TagAudioFormatUnsupported, "Service.Open.AudioFormatUnsupported"},
		{"a negative rate", func(f *fakePCM) {}, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: -1, Channels: 2}}, nil,
			platform.TagAudioFormatUnsupported, "Service.Open.AudioFormatUnsupported"},
		{"a context that ended", func(f *fakePCM) {}, platform.AudioPlayback, platform.AudioOpenOptions{}, canceled,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakePCM(t)
			tt.setup(f)
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}
			_, err := f.service().Open(ctx, tt.kind, tt.opts, &recorder{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("tag %s, route %s", appErr.Tag, appErr.Route())
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if tt.name == "a format that the device does not take" && f.closes != 1 {
				t.Errorf("the PCM was not closed after its parameters failed (%d closes)", f.closes)
			}
		})
	}
}

func TestLibraryMissing(t *testing.T) {
	t.Setenv(LibraryEnv, "/nonexistent/libasound.so.2")
	s := New()
	for _, call := range []struct {
		name  string
		do    func() error
		route string
	}{
		{"Open", func() error {
			_, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &recorder{})
			return err
		}, "Service.Open->Service.getLib->loadLibrary.AudioUnavailable"},
		{"Devices", func() error { _, err := s.Devices(platform.AudioPlayback); return err },
			"Service.Devices->Service.getLib->loadLibrary.AudioUnavailable"},
	} {
		t.Run(call.name, func(t *testing.T) {
			err := call.do()
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute(call.route) {
				t.Fatalf("got %v", err)
			}
			if appErr.Metadata["dlerror"] == "" {
				t.Errorf("metadata %v has no dlerror", appErr.Metadata)
			}
		})
	}
}

func TestDevices(t *testing.T) {
	hints := []fakeHint{
		{"null", "Discard all samples (playback) or generate zero samples (capture)", ""},
		{"default", "Default ALSA Output (currently PipeWire Media Server)", ""},
		{"sysdefault:CARD=PCH", "HDA Intel PCH, ALC3246 Analog\nDefault Audio Device", ""},
		{"hw:CARD=PCH,DEV=0", "HDA Intel PCH, ALC3246 Analog\nDirect hardware device without any conversions", ""},
		{"sysdefault:CARD=Speaker", "Speakers\nDefault Audio Device", "Output"},
		{"sysdefault:CARD=Mic", "Microphone\nDefault Audio Device", "Input"},
	}
	tests := []struct {
		name string
		kind platform.AudioKind
		want []string
	}{
		{"playback", platform.AudioPlayback, []string{"default", "sysdefault:CARD=PCH", "sysdefault:CARD=Speaker"}},
		{"capture", platform.AudioCapture, []string{"default", "sysdefault:CARD=PCH", "sysdefault:CARD=Mic"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakePCM(t)
			f.hints = hints
			list, err := f.service().Devices(tt.kind)
			if err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, d := range list {
				ids = append(ids, d.ID)
				if d.Kind != tt.kind || d.Default != (d.ID == "default") {
					t.Errorf("%+v", d)
				}
			}
			if len(ids) != len(tt.want) {
				t.Fatalf("devices %v, want %v", ids, tt.want)
			}
			for i := range ids {
				if ids[i] != tt.want[i] {
					t.Errorf("devices %v, want %v", ids, tt.want)
				}
			}
			if list[1].Name != "HDA Intel PCH, ALC3246 Analog" {
				t.Errorf("name %q: only the first line of the description is the name", list[1].Name)
			}
		})
	}
}

func TestDevicesWithoutHints(t *testing.T) {
	f := newFakePCM(t)
	list, err := f.service().Devices(platform.AudioPlayback)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "default" || !list[0].Default {
		t.Errorf("%+v", list)
	}
}

func TestChangeHandlerIsAccepted(t *testing.T) {
	s := New()
	s.SetChangeHandler(func() { t.Error("ALSA has no hotplug") })
	s.SetChangeHandler(nil)
}
