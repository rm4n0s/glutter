//go:build linux && !android

package pulse

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl/layouttest"
)

// The layout of the mirrors, as pulse/sample.h, channelmap.h and def.h have it on 64-bit Linux.
func TestLayouts(t *testing.T) {
	layouttest.AssertLayout(t, "pa_sample_spec", 12, map[string]uintptr{"format": 0, "rate": 4, "channels": 8}, sampleSpec{})
	layouttest.AssertLayout(t, "pa_channel_map", 132, map[string]uintptr{"channels": 0, "pos": 4}, channelMap{})
	layouttest.AssertLayout(t, "pa_buffer_attr", 20,
		map[string]uintptr{"maxlength": 0, "tlength": 4, "prebuf": 8, "minreq": 12, "fragsize": 16}, bufferAttr{})
}

// TestLayoutsAgainstTheHeaders compiles a program against the headers of libpulse, when they are installed, and compares what the
// compiler says with the numbers above and with the constants of this package.
func TestLayoutsAgainstTheHeaders(t *testing.T) {
	if _, err := os.Stat("/usr/include/pulse/pulseaudio.h"); err != nil {
		t.Skip("the headers of libpulse are not installed (sudo dnf install pulseaudio-libs-devel); the layout is checked against the numbers of the test only")
	}
	cc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("no C compiler")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "layout.c")
	prog := `#include <stdio.h>
#include <stddef.h>
#include <pulse/pulseaudio.h>
int main(void) {
	printf("%zu %zu %zu %zu\n", sizeof(pa_sample_spec), offsetof(pa_sample_spec, rate), offsetof(pa_sample_spec, channels), sizeof(pa_channel_map));
	printf("%zu %zu %zu\n", offsetof(pa_channel_map, map), sizeof(pa_buffer_attr), offsetof(pa_buffer_attr, fragsize));
	printf("%zu %zu %zu\n", offsetof(pa_sink_info, description), offsetof(pa_sink_info, sample_spec), offsetof(pa_server_info, default_sink_name));
	printf("%zu %zu %zu\n", offsetof(pa_source_info, description), offsetof(pa_source_info, sample_spec), offsetof(pa_server_info, default_source_name));
	printf("%d %d %d %d %d\n", PA_SAMPLE_FLOAT32LE, PA_CONTEXT_READY, PA_STREAM_READY, PA_CONTEXT_NOAUTOSPAWN, PA_STREAM_ADJUST_LATENCY);
	printf("%d %d %d %d\n", PA_SUBSCRIPTION_MASK_SINK, PA_SUBSCRIPTION_MASK_SOURCE, PA_SUBSCRIPTION_MASK_SERVER, PA_ERR_ACCESS);
	printf("%d %d %d %d %d %d\n", PA_CHANNEL_POSITION_MONO, PA_CHANNEL_POSITION_FRONT_CENTER, PA_CHANNEL_POSITION_REAR_CENTER, PA_CHANNEL_POSITION_LFE, PA_CHANNEL_POSITION_SIDE_LEFT, PA_CHANNEL_POSITION_SIDE_RIGHT);
	return 0;
}
`
	if err := os.WriteFile(src, []byte(prog), 0o600); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "layout")
	if out, err := exec.Command(cc, "-o", exe, src).CombinedOutput(); err != nil {
		t.Fatalf("the compiler failed: %v\n%s", err, out)
	}
	out, err := exec.Command(exe).Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"12 4 8 132",
		"4 20 16",
		strconv.Itoa(infoDescriptionOffset) + " " + strconv.Itoa(infoSampleSpecOffset) + " " + strconv.Itoa(serverDefaultSinkOffset),
		strconv.Itoa(infoDescriptionOffset) + " " + strconv.Itoa(infoSampleSpecOffset) + " " + strconv.Itoa(serverDefaultSourceOffset),
		strings.Join([]string{strconv.Itoa(sampleFloat32LE), strconv.Itoa(contextReady), strconv.Itoa(streamReady), strconv.Itoa(contextNoAutoSpawn), strconv.Itoa(streamAdjustLatency)}, " "),
		strings.Join([]string{strconv.Itoa(subscriptionMaskSink), strconv.Itoa(subscriptionMaskSource), strconv.Itoa(subscriptionMaskServer), strconv.Itoa(errAccess)}, " "),
		strings.Join([]string{strconv.Itoa(posMono), strconv.Itoa(posFrontCenter), strconv.Itoa(posRearCenter), strconv.Itoa(posLFE), strconv.Itoa(posSideLeft), strconv.Itoa(posSideRight)}, " "),
	}, "\n") + "\n"
	if string(out) != want {
		t.Errorf("the headers say\n%swant\n%s", out, want)
	}
}

func TestChannelMaps(t *testing.T) {
	tests := []struct {
		channels int
		want     []int32
	}{
		{1, []int32{posMono}},
		{2, []int32{posFrontLeft, posFrontRight}},
		{6, []int32{posFrontLeft, posFrontRight, posFrontCenter, posLFE, posRearLeft, posRearRight}},
		{8, []int32{posFrontLeft, posFrontRight, posFrontCenter, posLFE, posRearLeft, posRearRight, posSideLeft, posSideRight}},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.channels), func(t *testing.T) {
			var m channelMap
			fillChannelMap(&m, tt.channels)
			if int(m.channels) != tt.channels {
				t.Errorf("channels %d", m.channels)
			}
			for i, w := range tt.want {
				if m.pos[i] != w {
					t.Errorf("position %d is %d, want %d", i, m.pos[i], w)
				}
			}
			for i := len(tt.want); i < maxChannels; i++ {
				if m.pos[i] != 0 {
					t.Errorf("position %d past the channels is %d", i, m.pos[i])
				}
			}
		})
	}
	for ch := 1; ch <= maxAudioChans; ch++ {
		if len(channelMaps[ch]) != ch {
			t.Errorf("the map for %d channels has %d positions", ch, len(channelMaps[ch]))
		}
	}
}

// recorder is a handler that keeps what it is given.
type recorder struct {
	mu       sync.Mutex
	fills    int
	captured []float32
	states   []platform.AudioState
}

func (r *recorder) Fill(out []float32) {
	r.mu.Lock()
	r.fills++
	r.mu.Unlock()
	for i := range out {
		out[i] = float32(i%7) / 10
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

// fakeServer scripts libpulse: the functions that Service calls answer from its fields, and the memory that begin_write and peek
// give is C memory of its own. Everything runs on the goroutine of the test, as the real calls would with the lock held.
type fakeServer struct {
	ctxState    int32
	streamState int32
	connectRC   int32
	errno       int32
	streamNewRC uintptr
	writable    uintptr
	written     []int // the size of each write, in bytes
	cancelled   int
	corks       []int32
	buf         uintptr // C memory that begin_write and peek point to
	bufSize     uintptr
	peeks       [][2]uintptr // what each peek returns: the pointer and the size
	drops       int
	disconnects int
	text        uintptr
}

func newFakeServer(t *testing.T) *fakeServer {
	f := &fakeServer{ctxState: contextReady, streamState: streamReady, streamNewRC: 1, bufSize: 1 << 16}
	f.buf = dl.Malloc(f.bufSize)
	f.text = dl.CString("fake error text")
	t.Cleanup(func() { dl.Free(f.buf); dl.Free(f.text) })
	return f
}

func (f *fakeServer) library() *library {
	return &library{
		mainloopNew: func() uintptr { return 1 }, mainloopFree: func(uintptr) {}, mainloopStart: func(uintptr) int32 { return 0 },
		mainloopStop: func(uintptr) {}, mainloopLock: func(uintptr) {}, mainloopUnlock: func(uintptr) {}, mainloopWait: func(uintptr) {},
		mainloopSignal: func(uintptr, int32) {}, mainloopGetAPI: func(uintptr) uintptr { return 1 },
		contextNew: func(uintptr, uintptr) uintptr { return 2 }, contextUnref: func(uintptr) {},
		contextConnect:    func(uintptr, uintptr, int32, uintptr) int32 { return 0 },
		contextDisconnect: func(uintptr) { f.disconnects++ }, contextGetState: func(uintptr) int32 { return f.ctxState },
		contextErrno: func(uintptr) int32 { return f.errno }, strerror: func(int32) uintptr { return f.text },
		contextSetStateCallback: func(uintptr, uintptr, uintptr) {}, contextSetSubscribeCallback: func(uintptr, uintptr, uintptr) {},
		contextSubscribe: func(uintptr, uint32, uintptr, uintptr) uintptr { return 0 },
		operationUnref:   func(uintptr) {}, operationGetState: func(uintptr) int32 { return 1 },
		streamNew:      func(uintptr, uintptr, uintptr, uintptr) uintptr { return f.streamNewRC },
		streamUnref:    func(uintptr) {},
		streamGetState: func(uintptr) int32 { return f.streamState }, streamSetStateCallback: func(uintptr, uintptr, uintptr) {},
		streamSetWriteCallback: func(uintptr, uintptr, uintptr) {}, streamSetReadCallback: func(uintptr, uintptr, uintptr) {},
		streamSetUnderflowCallback: func(uintptr, uintptr, uintptr) {},
		streamConnectPlayback:      func(uintptr, uintptr, uintptr, int32, uintptr, uintptr) int32 { return f.connectRC },
		streamConnectRecord:        func(uintptr, uintptr, uintptr, int32) int32 { return f.connectRC },
		streamDisconnect:           func(uintptr) int32 { f.disconnects++; return 0 },
		streamGetBufferAttr:        func(uintptr) uintptr { return 0 },
		streamWritableSize:         func(uintptr) uintptr { return f.writable },
		streamBeginWrite: func(_, data, nbytes uintptr) int32 {
			*(*uintptr)(dl.Pointer(data)) = f.buf
			size := *(*uintptr)(dl.Pointer(nbytes))
			*(*uintptr)(dl.Pointer(nbytes)) = min(size, f.bufSize)
			return 0
		},
		streamWrite: func(_, _, nbytes, _ uintptr, _ int64, _ int32) int32 {
			f.written = append(f.written, int(nbytes))
			f.writable -= min(f.writable, nbytes)
			return 0
		},
		streamCancelWrite: func(uintptr) int32 { f.cancelled++; return 0 },
		streamPeek: func(_, data, nbytes uintptr) int32 {
			if len(f.peeks) == 0 {
				*(*uintptr)(dl.Pointer(data)) = 0
				*(*uintptr)(dl.Pointer(nbytes)) = 0
				return 0
			}
			p := f.peeks[0]
			f.peeks = f.peeks[1:]
			*(*uintptr)(dl.Pointer(data)) = p[0]
			*(*uintptr)(dl.Pointer(nbytes)) = p[1]
			return 0
		},
		streamDrop: func(uintptr) int32 { f.drops++; return 0 },
		streamCork: func(_ uintptr, b int32, _, _ uintptr) uintptr { f.corks = append(f.corks, b); return 0 },
	}
}

func (f *fakeServer) service() *Service {
	s := New(Options{AppName: "test"})
	s.load = func() (*library, error) { return f.library(), nil }
	return s
}

func openFake(t *testing.T, f *fakeServer, kind platform.AudioKind, opts platform.AudioOpenOptions) (*Service, *device, *recorder) {
	t.Helper()
	s := f.service()
	h := &recorder{}
	dev, err := s.Open(context.Background(), kind, opts, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close(); s.Close() })
	return s, dev.(*device), h
}

func TestWritePieces(t *testing.T) {
	f := newFakeServer(t)
	_, d, _ := openFake(t, f, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}, BufferFrames: 480})
	tests := []struct {
		name     string
		writable uintptr
		want     []int
	}{
		{"nothing asked", 0, nil},
		{"less than a frame", 7, nil},
		{"one buffer", 480 * 8, []int{480 * 8}},
		{"a partial frame is left", 480*8 + 5, []int{480 * 8}},
		{"more than a buffer comes in pieces", 480*8*2 + 16, []int{480 * 8, 480 * 8, 16}},
		{"a size that the server says is an error", ^uintptr(0), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f.written, f.writable = nil, tt.writable
			d.write()
			if len(f.written) != len(tt.want) {
				t.Fatalf("wrote %v, want %v", f.written, tt.want)
			}
			for i := range tt.want {
				if f.written[i] != tt.want[i] {
					t.Errorf("piece %d is %d bytes, want %d", i, f.written[i], tt.want[i])
				}
			}
		})
	}
}

func TestWriteFillsTheServersMemory(t *testing.T) {
	f := newFakeServer(t)
	_, d, h := openFake(t, f, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}, BufferFrames: 100})
	f.writable = 100 * 8
	d.write()
	if h.fills != 1 {
		t.Fatalf("%d fills", h.fills)
	}
	got := unsafeFloats(f.buf, 200)
	for i, v := range got {
		if want := float32(i%7) / 10; v != want {
			t.Fatalf("sample %d is %v, want %v: the handler did not fill the memory of the server", i, v, want)
		}
	}
}

func TestPauseStopsTheHandler(t *testing.T) {
	f := newFakeServer(t)
	_, d, h := openFake(t, f, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 1}, BufferFrames: 100})
	if err := d.Pause(); err != nil {
		t.Fatal(err)
	}
	f.writable = 400
	d.service() // the callback of a request that comes before the server has seen the cork
	if h.fills != 0 || len(f.written) != 0 {
		t.Errorf("a paused device called the handler: %d fills, %d writes", h.fills, len(f.written))
	}
	if err := d.Resume(); err != nil {
		t.Fatal(err)
	}
	if len(f.written) != 1 || f.written[0] != 400 {
		t.Errorf("Resume wrote %v, want what was asked while paused", f.written)
	}
	if len(f.corks) != 2 || f.corks[0] != 1 || f.corks[1] != 0 {
		t.Errorf("corks %v, want [1 0]", f.corks)
	}
}

func TestReadFrames(t *testing.T) {
	f := newFakeServer(t)
	_, d, h := openFake(t, f, platform.AudioCapture, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}})
	samples := unsafeFloats(f.buf, 16)
	for i := range samples {
		samples[i] = float32(i)
	}
	// A recording of 4 frames, a hole of 2 (a pointer that is null, with a size), and 2 frames.
	f.peeks = [][2]uintptr{{f.buf, 4 * 8}, {0, 2 * 8}, {f.buf + 8*4, 2 * 8}}
	d.read()
	want := []float32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	got := h.captured
	if len(got) != len(want) {
		t.Fatalf("captured %d samples, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sample %d is %v, want %v", i, got[i], want[i])
		}
	}
	if f.drops != 3 {
		t.Errorf("%d drops, want 3 (the hole as well)", f.drops)
	}
}

func TestLostServerAndStream(t *testing.T) {
	f := newFakeServer(t)
	s, d, h := openFake(t, f, platform.AudioPlayback, platform.AudioOpenOptions{})
	f.ctxState = contextFailed
	s.contextStateChanged(s.ctx)
	s.contextStateChanged(s.ctx)
	d.stateChanged()
	if len(h.states) != 1 || h.states[0] != platform.AudioLost {
		t.Errorf("states %v, want a single AudioLost", h.states)
	}
	err := d.Pause()
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.Pause.AudioDeviceLost") {
		t.Fatalf("Pause of a lost device: %v", err)
	}
	err = d.Resume()
	appErr, ok = errors.FromError(err)
	if !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.Resume.AudioDeviceLost") {
		t.Fatalf("Resume of a lost device: %v", err)
	}
	// The next use connects again: the context was dead.
	f.ctxState = contextReady
	before := f.disconnects
	dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &recorder{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	if f.disconnects == before {
		t.Error("the dead context was not dropped")
	}
}

func TestStreamEndsOnItsOwn(t *testing.T) {
	f := newFakeServer(t)
	_, d, h := openFake(t, f, platform.AudioCapture, platform.AudioOpenOptions{})
	f.streamState = streamTerminated
	d.stateChanged()
	if len(h.states) != 1 || h.states[0] != platform.AudioLost {
		t.Errorf("states %v, want AudioLost", h.states)
	}
}

func TestOpenErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeServer)
		kind  platform.AudioKind
		opts  platform.AudioOpenOptions
		ctx   func() context.Context
		tag   string
		route string
	}{
		{"rate", func(*fakeServer) {}, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: -5, Channels: 2}}, nil,
			platform.TagAudioFormatUnsupported, "Service.Open.AudioFormatUnsupported"},
		{"channels", func(*fakeServer) {}, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 44100, Channels: 9}}, nil,
			platform.TagAudioFormatUnsupported, "Service.Open.AudioFormatUnsupported"},
		{"canceled", func(*fakeServer) {}, platform.AudioPlayback, platform.AudioOpenOptions{}, func() context.Context {
			c, cancel := context.WithCancel(context.Background())
			cancel()
			return c
		}, platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"the server takes no such stream", func(f *fakeServer) { f.streamNewRC = 0 }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioFormatUnsupported, "Service.Open.AudioFormatUnsupported"},
		{"no such device", func(f *fakeServer) { f.connectRC = -1; f.errno = 5 }, platform.AudioPlayback, platform.AudioOpenOptions{ID: "nothing"}, nil,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"the stream fails", func(f *fakeServer) { f.streamState = streamFailed }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"the recording is refused", func(f *fakeServer) { f.connectRC = -1; f.errno = errAccess }, platform.AudioCapture, platform.AudioOpenOptions{}, nil,
			platform.TagAudioCapturePermissionDenied, "Service.Open.AudioCapturePermissionDenied"},
		{"a playback that is refused is not a permission", func(f *fakeServer) { f.connectRC = -1; f.errno = errAccess }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioDeviceOpenFailed, "Service.Open.AudioDeviceOpenFailed"},
		{"no server", func(f *fakeServer) { f.ctxState = contextFailed }, platform.AudioPlayback, platform.AudioOpenOptions{}, nil,
			platform.TagAudioUnavailable, "Service.Open->Service.connect.AudioUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeServer(t)
			tt.setup(f)
			s := f.service()
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}
			before := handleCount()
			_, err := s.Open(ctx, tt.kind, tt.opts, &recorder{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("tag %s, route %s", appErr.Tag, appErr.Route())
			}
			s.Close()
			if got := handleCount(); got != before {
				t.Errorf("%d handles are left after a failed open", got-before)
			}
		})
	}
}

func TestDevicesNoServer(t *testing.T) {
	f := newFakeServer(t)
	f.ctxState = contextFailed
	s := f.service()
	defer s.Close()
	_, err := s.Devices(platform.AudioPlayback)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.Devices->Service.connect.AudioUnavailable") {
		t.Fatalf("got %v", err)
	}
}

func TestLibraryMissing(t *testing.T) {
	t.Setenv(LibraryEnv, "/nonexistent/libpulse.so.0")
	s := New(Options{})
	_, err := s.Devices(platform.AudioPlayback)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.Devices->Service.connect->loadLibrary.AudioUnavailable") {
		t.Fatalf("got %v", err)
	}
	if appErr.Metadata["dlerror"] == "" {
		t.Errorf("metadata %v has no dlerror", appErr.Metadata)
	}
}

// With a path that is not absolute the override is refused, which is the same failure.
func TestLibraryOverrideInvalid(t *testing.T) {
	t.Setenv(LibraryEnv, "relative/libpulse.so.0")
	_, err := New(Options{}).Devices(platform.AudioCapture)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable {
		t.Fatalf("got %v", err)
	}
}

func TestDeviceInfoParsing(t *testing.T) {
	// A sink or source info as libpulse lays it out: the name, the index, the description, the sample specification.
	mk := func(name, desc string, rate uint32, channels uint8) (uintptr, func()) {
		info := dl.Calloc(1, 512)
		n, d := dl.CString(name), dl.CString(desc)
		*(*uintptr)(dl.Pointer(info + infoNameOffset)) = n
		*(*uintptr)(dl.Pointer(info + infoDescriptionOffset)) = d
		*(*sampleSpec)(dl.Pointer(info + infoSampleSpecOffset)) = sampleSpec{format: sampleFloat32LE, rate: rate, channels: channels}
		return info, func() { dl.Free(info); dl.Free(n); dl.Free(d) }
	}
	f := newFakeServer(t)
	s := f.service()
	s.lib = f.library()
	s.loop = 1
	s.query = &query{}
	sink, free1 := mk("alsa_output.pci", "Built-in Audio", 44100, 2)
	defer free1()
	mon, free2 := mk("alsa_output.pci.monitor", "Monitor of Built-in Audio", 44100, 2)
	defer free2()
	mic, free3 := mk("alsa_input.usb", "USB headset", 16000, 12)
	defer free3()
	s.deviceInfo(sink, 0, false)
	s.deviceInfo(0, 1, false)
	s.deviceInfo(mon, 0, true)
	s.deviceInfo(mic, 0, true)
	got := s.query.infos
	if len(got) != 2 {
		t.Fatalf("%d devices, want the sink and the microphone (not the monitor): %+v", len(got), got)
	}
	if got[0].ID != "alsa_output.pci" || got[0].Name != "Built-in Audio" || got[0].Kind != platform.AudioPlayback || got[0].Format != (platform.AudioFormat{Rate: 44100, Channels: 2}) {
		t.Errorf("sink: %+v", got[0])
	}
	if got[1].ID != "alsa_input.usb" || got[1].Kind != platform.AudioCapture || got[1].Format != (platform.AudioFormat{Rate: 16000, Channels: 8}) {
		t.Errorf("source: %+v", got[1])
	}
}

func TestChangeHandler(t *testing.T) {
	s := New(Options{})
	calls := 0
	s.subscribed(0) // no handler yet: nothing happens
	s.SetChangeHandler(func() { calls++ })
	s.subscribed(0)
	s.SetChangeHandler(nil)
	s.subscribed(0)
	if calls != 1 {
		t.Errorf("%d calls, want 1", calls)
	}
}

func TestCallbacksAreMadeOnce(t *testing.T) {
	before := dl.CallbackCount()
	a := callbacks()
	n := dl.CallbackCount()
	b := callbacks()
	if a != b || dl.CallbackCount() != n {
		t.Errorf("the callbacks were made again: %v %v, %d %d", a, b, n, dl.CallbackCount())
	}
	if n-before > 8 {
		t.Errorf("%d callbacks made, want at most 8", n-before)
	}
}

func TestUseAfterServiceClose(t *testing.T) {
	f := newFakeServer(t)
	s, d, h := openFake(t, f, platform.AudioPlayback, platform.AudioOpenOptions{})
	s.Close()
	s.Close() // a second Close does nothing
	// The device was closed with the service: its owner can still call it.
	if err := d.Pause(); err != nil {
		t.Errorf("Pause: %v", err)
	}
	if err := d.Resume(); err != nil {
		t.Errorf("Resume: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	f.writable = 800
	d.service() // a late callback of the server
	if h.fills != 0 {
		t.Error("the handler was called after the service was closed")
	}
	for name, call := range map[string]func() error{
		"Devices": func() error { _, err := s.Devices(platform.AudioPlayback); return err },
		"Open": func() error {
			_, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &recorder{})
			return err
		},
	} {
		err := call()
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service."+name+"->Service.connect.AudioUnavailable") {
			t.Errorf("%s after Close: %v", name, err)
		}
	}
}
