//go:build darwin

package coreaudio

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

const tagFakePermission = "FakePermission"

// fakeDev is a device of the fake system.
type fakeDev struct {
	uid, name string
	out, in   int
	rate      float64
	dead      bool
}

// fakeQueue is a queue that the fake made.
type fakeQueue struct {
	sd        StreamDescription
	ctx       uintptr
	capture   bool
	props     map[uint32][]byte
	buffers   []uintptr
	enqueued  int
	started   int
	paused    int
	stopped   bool
	disposed  bool
	layoutTag uint32
}

// fakeSystem is a library table over a made-up Core Audio.
type fakeSystem struct {
	t       *testing.T
	devs    map[uint32]*fakeDev
	order   []uint32
	defOut  uint32
	defIn   uint32
	queues  []*fakeQueue
	byID    map[uintptr]*fakeQueue
	nextQ   uintptr
	failNew int32 // the status of newOutput and newInput
	failSet int32
	failAdd int32 // the status of addListener
	pauseSt int32
	startSt int32

	listeners map[uint32][]uint32 // the selectors listened to by object
	removed   int
	frees     []uintptr
	lib       *library
}

func newFakeSystem(t *testing.T) *fakeSystem {
	t.Helper()
	f := &fakeSystem{t: t, byID: map[uintptr]*fakeQueue{}, listeners: map[uint32][]uint32{}, devs: map[uint32]*fakeDev{
		10: {uid: "BuiltInSpeakerDevice", name: "Speakers", out: 2, rate: 44100},
		11: {uid: "BuiltInMicrophoneDevice", name: "Microphone", in: 1, rate: 48000},
		12: {uid: "UsbHeadset", name: "Headset", out: 2, in: 1, rate: 48000},
	}, order: []uint32{10, 11, 12}, defOut: 10, defIn: 11}
	l := &library{outputCB: 0x1001, inputCB: 0x1002, listenerCB: 0x1003}
	l.getPropertyDataSize = func(obj uint32, a *PropertyAddress, _ uint32, _ unsafe.Pointer, size *uint32) int32 {
		switch {
		case obj == systemObject && a.Selector == selDevices:
			*size = uint32(4 * len(f.order))
		case a.Selector == selStreamConfig:
			*size = uint32(len(f.streamConfig(obj, a.Scope)))
		default:
			return -1
		}
		return 0
	}
	l.getPropertyData = func(obj uint32, a *PropertyAddress, _ uint32, _ unsafe.Pointer, size *uint32, data unsafe.Pointer) int32 {
		d := f.devs[obj]
		switch {
		case obj == systemObject && a.Selector == selDevices:
			for i, id := range f.order {
				*(*uint32)(unsafe.Add(data, 4*i)) = id
			}
			*size = uint32(4 * len(f.order))
		case obj == systemObject && a.Selector == selDefaultOutput:
			*(*uint32)(data) = f.defOut
		case obj == systemObject && a.Selector == selDefaultInput:
			*(*uint32)(data) = f.defIn
		case d != nil && a.Selector == selUID:
			*(*cf.Ref)(data) = cf.NewString(d.uid)
		case d != nil && a.Selector == selName:
			*(*cf.Ref)(data) = cf.NewString(d.name)
		case d != nil && a.Selector == selNominalRate:
			*(*float64)(data) = d.rate
		case d != nil && a.Selector == selIsAlive:
			if d.dead {
				*(*uint32)(data) = 0
			} else {
				*(*uint32)(data) = 1
			}
		case d != nil && a.Selector == selStreamConfig:
			copy(unsafe.Slice((*byte)(data), *size), f.streamConfig(obj, a.Scope))
		default:
			return -1
		}
		return 0
	}
	l.addListener = func(obj uint32, a *PropertyAddress, _ uintptr, _ uintptr) int32 {
		if f.failAdd != 0 {
			return f.failAdd
		}
		f.listeners[obj] = append(f.listeners[obj], a.Selector)
		return 0
	}
	l.removeListener = func(obj uint32, a *PropertyAddress, _ uintptr, _ uintptr) int32 { f.removed++; return 0 }
	newQueue := func(capture bool) func(*StreamDescription, uintptr, uintptr, cf.Ref, cf.Ref, uint32, *uintptr) int32 {
		return func(sd *StreamDescription, _ uintptr, ctx uintptr, _, _ cf.Ref, _ uint32, q *uintptr) int32 {
			if f.failNew != 0 {
				return f.failNew
			}
			f.nextQ++
			fq := &fakeQueue{sd: *sd, ctx: ctx, capture: capture, props: map[uint32][]byte{}}
			f.queues = append(f.queues, fq)
			f.byID[f.nextQ] = fq
			*q = f.nextQ
			return 0
		}
	}
	l.newOutput, l.newInput = newQueue(false), newQueue(true)
	l.setProperty = func(q uintptr, id uint32, data unsafe.Pointer, size uint32) int32 {
		if f.failSet != 0 {
			return f.failSet
		}
		fq := f.byID[q]
		fq.props[id] = append([]byte(nil), unsafe.Slice((*byte)(data), size)...)
		if id == queueChannelLay {
			fq.layoutTag = binary.NativeEndian.Uint32(fq.props[id])
		}
		return 0
	}
	l.allocBuffer = func(q uintptr, size uint32, out *uintptr) int32 {
		fq := f.byID[q]
		b := dl.Malloc(unsafe.Sizeof(QueueBuffer{}))
		data := dl.Malloc(uintptr(size))
		qb := (*QueueBuffer)(dl.Pointer(b))
		*qb = QueueBuffer{Capacity: size, Data: data}
		// Fill the memory with a mark that a silent buffer must overwrite.
		for i := range unsafe.Slice((*byte)(dl.Pointer(data)), size) {
			unsafe.Slice((*byte)(dl.Pointer(data)), size)[i] = 0xAB
		}
		fq.buffers = append(fq.buffers, b)
		f.frees = append(f.frees, b, data)
		*out = b
		return 0
	}
	l.enqueue = func(q uintptr, _ uintptr, _ uint32, _ uintptr) int32 { f.byID[q].enqueued++; return 0 }
	l.start = func(q uintptr, _ uintptr) int32 {
		if f.startSt != 0 {
			return f.startSt
		}
		f.byID[q].started++
		return 0
	}
	l.pause = func(q uintptr) int32 {
		if f.pauseSt != 0 {
			return f.pauseSt
		}
		f.byID[q].paused++
		return 0
	}
	l.stop = func(q uintptr, immediate bool) int32 { f.byID[q].stopped = immediate; return 0 }
	l.dispose = func(q uintptr, immediate bool) int32 { f.byID[q].disposed = immediate; return 0 }
	f.lib = l
	t.Cleanup(func() {
		for _, p := range f.frees {
			dl.Free(p)
		}
	})
	return f
}

// streamConfig is the AudioBufferList of a device in a scope.
func (f *fakeSystem) streamConfig(obj, scope uint32) []byte {
	d := f.devs[obj]
	if d == nil {
		return audioBufferList(0)
	}
	ch := d.out
	if scope == scopeInput {
		ch = d.in
	}
	if ch == 0 {
		return audioBufferList(0)
	}
	return audioBufferList(1, ch)
}

// service makes a Service over the fake.
func (f *fakeSystem) service(perm Permissions) *Service {
	s := New(Options{Permissions: perm})
	s.load = func() (*library, error) { return f.lib, nil }
	return s
}

// recorder is the AudioHandler under test.
type recorder struct {
	mu       sync.Mutex
	fills    [][]float32
	captures [][]float32
	states   []platform.AudioState
	value    float32
	panics   bool
}

func (r *recorder) Fill(out []float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.panics {
		panic("the handler broke")
	}
	for i := range out {
		out[i] = r.value
	}
	r.fills = append(r.fills, out[:len(out):len(out)])
}

func (r *recorder) Capture(in []float32) {
	r.mu.Lock()
	r.captures = append(r.captures, append([]float32(nil), in...))
	r.mu.Unlock()
}

func (r *recorder) StateChanged(s platform.AudioState) {
	r.mu.Lock()
	r.states = append(r.states, s)
	r.mu.Unlock()
}

func (r *recorder) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.fills) + len(r.captures)
}

func (r *recorder) stateList() []platform.AudioState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]platform.AudioState(nil), r.states...)
}

type fakePerm struct {
	granted bool
	err     error
	asked   int
}

func (p *fakePerm) Microphone(context.Context) (bool, error) { p.asked++; return p.granted, p.err }

func memFloats(addr uintptr, n int) []float32 { return unsafe.Slice((*float32)(dl.Pointer(addr)), n) }

func openPlayback(t *testing.T, s *Service, opts platform.AudioOpenOptions, h platform.AudioHandler) *device {
	t.Helper()
	d, err := s.Open(context.Background(), platform.AudioPlayback, opts, h)
	if err != nil {
		t.Fatal(err)
	}
	return d.(*device)
}

func TestServiceListsDevices(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	out, err := s.Devices(platform.AudioPlayback)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].ID != "BuiltInSpeakerDevice" || !out[0].Default || out[1].ID != "UsbHeadset" || out[1].Default ||
		out[0].Format != (platform.AudioFormat{Rate: 44100, Channels: 2}) || out[0].Name != "Speakers" {
		t.Errorf("playback = %+v", out)
	}
	in, err := s.Devices(platform.AudioCapture)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 2 || in[0].ID != "BuiltInMicrophoneDevice" || !in[0].Default || in[1].Format.Channels != 1 {
		t.Errorf("capture = %+v", in)
	}
	// The system object is listened to once, for the devices and the two defaults.
	if got := f.listeners[systemObject]; len(got) != 3 {
		t.Errorf("listeners = %v", got)
	}
	// The default device changes: the list says so.
	f.defOut = 12
	out, _ = s.Devices(platform.AudioPlayback)
	if out[0].Default || !out[1].Default {
		t.Errorf("playback after the change = %+v", out)
	}
	if len(f.listeners[systemObject]) != 3 {
		t.Errorf("listeners were added again: %v", f.listeners[systemObject])
	}
}

func TestServiceTellsWhenTheSystemChanges(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	if _, err := s.Devices(platform.AudioPlayback); err != nil {
		t.Fatal(err)
	}
	n := 0
	s.SetChangeHandler(func() { n++ })
	s.onSystemChange([]uint32{selDevices})
	s.onSystemChange([]uint32{selDefaultOutput})
	if n != 2 {
		t.Errorf("the handler was called %d times", n)
	}
	s.SetChangeHandler(nil)
	s.onSystemChange([]uint32{selDevices}) // nothing to call
}

func TestOpenPlaybackOnTheDefaultDevice(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	h := &recorder{value: 0.5}
	d := openPlayback(t, s, platform.AudioOpenOptions{}, h)
	// The device's own rate, stereo, float32.
	if d.Format() != (platform.AudioFormat{Rate: 44100, Channels: 2}) {
		t.Errorf("Format = %+v", d.Format())
	}
	if len(f.queues) != 1 {
		t.Fatalf("queues = %d", len(f.queues))
	}
	q := f.queues[0]
	if got, ok := q.sd.Format(); !ok || got != d.Format() || q.capture {
		t.Errorf("queue = %+v", q.sd)
	}
	if len(q.props) != 0 {
		t.Errorf("the default device was named: %v", q.props)
	}
	if len(q.buffers) != queueBuffers || q.enqueued != queueBuffers || q.started != 1 {
		t.Errorf("buffers %d enqueued %d started %d", len(q.buffers), q.enqueued, q.started)
	}
	// The buffers that were queued first are silence, 10 ms each.
	b := (*QueueBuffer)(dl.Pointer(q.buffers[0]))
	if b.Capacity != 441*2*4 || b.ByteSize != b.Capacity {
		t.Errorf("buffer %+v", *b)
	}
	for i, v := range memFloats(b.Data, int(b.Capacity/4)) {
		if v != 0 {
			t.Fatalf("sample %d of the first buffer = %v", i, v)
		}
	}
	if h.calls() != 0 {
		t.Error("the handler was called while the device opened")
	}

	// A buffer that comes back is filled by the handler, and queued again.
	d.onOutput(1, q.buffers[1])
	b = (*QueueBuffer)(dl.Pointer(q.buffers[1]))
	if h.calls() != 1 || len(h.fills[0]) != 441*2 || memFloats(b.Data, 4)[3] != 0.5 || b.ByteSize != b.Capacity || q.enqueued != queueBuffers+1 {
		t.Errorf("calls %d, buffer %+v, enqueued %d", h.calls(), *b, q.enqueued)
	}

	// Paused, it gives silence and does not call the handler.
	if err := d.Pause(); err != nil {
		t.Fatal(err)
	}
	d.onOutput(1, q.buffers[1])
	if h.calls() != 1 || memFloats(b.Data, 4)[3] != 0 || q.paused != 1 {
		t.Errorf("paused: calls %d, sample %v, paused %d", h.calls(), memFloats(b.Data, 4)[3], q.paused)
	}
	if err := d.Resume(); err != nil {
		t.Fatal(err)
	}
	d.onOutput(1, q.buffers[1])
	if h.calls() != 2 || q.started != 2 {
		t.Errorf("resumed: calls %d, started %d", h.calls(), q.started)
	}
	if got := h.stateList(); len(got) != 2 || got[0] != platform.AudioPaused || got[1] != platform.AudioRunning {
		t.Errorf("states = %v", got)
	}

	// Closed, the queue is stopped and disposed at once, and a late callback does nothing.
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if !q.stopped || !q.disposed {
		t.Errorf("stopped %v disposed %v", q.stopped, q.disposed)
	}
	before := q.enqueued
	d.onOutput(1, q.buffers[1])
	if h.calls() != 2 || q.enqueued != before {
		t.Errorf("a callback after Close did something: calls %d enqueued %d", h.calls(), q.enqueued)
	}
	if lookup(d.handle) != nil || len(s.devices) != 0 {
		t.Error("the device is still registered")
	}
	for _, c := range []func() error{d.Pause, d.Resume} {
		appErr, ok := errors.FromError(c())
		if !ok || appErr.Tag != platform.TagAudioDeviceLost {
			t.Errorf("a closed device: %v", appErr)
		}
	}
}

func TestOpenAsksForWhatTheAppWants(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	d := openPlayback(t, s, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 22050, Channels: 6}, BufferFrames: 256}, &recorder{})
	q := f.queues[0]
	if d.Format() != (platform.AudioFormat{Rate: 22050, Channels: 6}) || q.sd.SampleRate != 22050 || q.sd.ChannelsPerFrame != 6 {
		t.Errorf("format %+v, queue %+v", d.Format(), q.sd)
	}
	if q.layoutTag != 147<<16|6 {
		t.Errorf("layout tag = %#x", q.layoutTag)
	}
	if b := (*QueueBuffer)(dl.Pointer(q.buffers[0])); b.Capacity != 256*6*4 {
		t.Errorf("buffer = %d bytes", b.Capacity)
	}
	_ = d.Close()
}

func TestOpenNamedDeviceFollowsItsLife(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	h := &recorder{}
	d := openPlayback(t, s, platform.AudioOpenOptions{ID: "UsbHeadset"}, h)
	q := f.queues[0]
	if len(q.props[queueCurrentDev]) != 8 {
		t.Errorf("the queue was not given the device: %v", q.props)
	}
	if got := f.listeners[12]; len(got) != 1 || got[0] != selIsAlive {
		t.Errorf("listeners of the device = %v", got)
	}
	// A change of another property of the device is nothing.
	d.onProperty(12, []uint32{selName})
	if len(h.stateList()) != 0 {
		t.Errorf("states = %v", h.stateList())
	}
	// The device goes: the queue ends and the handler is told once.
	f.devs[12].dead = true
	d.onProperty(12, []uint32{selIsAlive})
	d.onProperty(12, []uint32{selIsAlive})
	if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioLost {
		t.Errorf("states = %v", got)
	}
	if !q.stopped || !q.disposed {
		t.Error("the queue of a lost device was left")
	}
	for name, call := range map[string]func() error{"Pause": d.Pause, "Resume": d.Resume} {
		appErr, ok := errors.FromError(call())
		if !ok || !appErr.HasRoute("device."+name+"->device.usable.AudioDeviceLost") {
			t.Errorf("%s of a lost device: %v", name, appErr)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if f.removed != 1 {
		t.Errorf("listeners removed = %d", f.removed)
	}
	// A device that is alive when its property changes stays.
	f.devs[12].dead = false
	d2 := openPlayback(t, s, platform.AudioOpenOptions{ID: "UsbHeadset"}, h)
	d2.onProperty(12, []uint32{selIsAlive})
	if len(h.stateList()) != 1 {
		t.Errorf("states = %v", h.stateList())
	}
	_ = d2.Close()
}

func TestDefaultDeviceIsFollowed(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	hp, hc := &recorder{}, &recorder{}
	play := openPlayback(t, s, platform.AudioOpenOptions{}, hp)
	cap1, err := s.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, hc)
	if err != nil {
		t.Fatal(err)
	}
	named := openPlayback(t, s, platform.AudioOpenOptions{ID: "UsbHeadset"}, &recorder{})
	if len(f.queues) != 3 {
		t.Fatalf("queues = %d", len(f.queues))
	}
	changes := 0
	s.SetChangeHandler(func() { changes++ })

	// The default output changes: the playback queue on the default device is made again; the others stay.
	s.onSystemChange([]uint32{selDefaultOutput})
	if len(f.queues) != 4 || !f.queues[0].disposed || f.queues[1].disposed || f.queues[2].disposed || f.queues[3].started != 1 || f.queues[3].capture {
		t.Fatalf("after the output changed: %d queues, disposed %v %v %v", len(f.queues), f.queues[0].disposed, f.queues[1].disposed, f.queues[2].disposed)
	}
	// The input changes: the capture queue is made again.
	s.onSystemChange([]uint32{selDefaultInput})
	if len(f.queues) != 5 || !f.queues[1].disposed || !f.queues[4].capture || f.queues[4].started != 1 {
		t.Fatalf("after the input changed: %d queues", len(f.queues))
	}
	// The list of devices changes: nothing is made again.
	s.onSystemChange([]uint32{selDevices})
	if len(f.queues) != 5 || changes != 3 {
		t.Errorf("queues %d, changes %d", len(f.queues), changes)
	}
	// A paused device that follows stays paused: no queue is started.
	if err := play.Pause(); err != nil {
		t.Fatal(err)
	}
	s.onSystemChange([]uint32{selDefaultOutput})
	if q := f.queues[5]; q.started != 0 {
		t.Errorf("a paused device was started: %+v", q)
	}
	// The handler gets what the new queue gives.
	play.onOutput(6, f.queues[5].buffers[0])
	if len(hp.fills) != 0 {
		t.Error("a paused device called the handler")
	}
	for _, d := range []platform.AudioDevice{play, cap1, named} {
		_ = d.Close()
	}

	// A queue that cannot be made again loses the device.
	h := &recorder{}
	lost := openPlayback(t, s, platform.AudioOpenOptions{}, h)
	f.failNew = -1
	s.onSystemChange([]uint32{selDefaultOutput})
	if got := h.stateList(); len(got) != 1 || got[0] != platform.AudioLost {
		t.Errorf("states = %v", got)
	}
	_ = lost.Close()
}

func TestCapture(t *testing.T) {
	f := newFakeSystem(t)
	perm := &fakePerm{granted: true}
	s := f.service(perm)
	h := &recorder{}
	dev, err := s.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, h)
	if err != nil {
		t.Fatal(err)
	}
	d := dev.(*device)
	if perm.asked != 1 {
		t.Errorf("the permission was asked %d times", perm.asked)
	}
	// The microphone has one channel at 48000.
	if d.Format() != (platform.AudioFormat{Rate: 48000, Channels: 1}) {
		t.Errorf("Format = %+v", d.Format())
	}
	q := f.queues[0]
	if !q.capture || q.enqueued != queueBuffers || q.started != 1 {
		t.Errorf("queue = %+v", q)
	}
	// A recorded buffer goes to the handler and comes back empty.
	b := (*QueueBuffer)(dl.Pointer(q.buffers[0]))
	copy(memFloats(b.Data, 3), []float32{0.25, -0.5, 1})
	b.ByteSize = 12
	d.onInput(1, q.buffers[0])
	if len(h.captures) != 1 || len(h.captures[0]) != 3 || h.captures[0][1] != -0.5 || b.ByteSize != 0 || q.enqueued != queueBuffers+1 {
		t.Errorf("captures %v, size %d, enqueued %d", h.captures, b.ByteSize, q.enqueued)
	}
	// An empty buffer is not given; a paused device drops what it got.
	d.onInput(1, q.buffers[0])
	b.ByteSize = 8
	_ = d.Pause()
	d.onInput(1, q.buffers[0])
	if len(h.captures) != 1 || b.ByteSize != 0 {
		t.Errorf("captures %d", len(h.captures))
	}
	_ = d.Close()
}

func TestOpenErrors(t *testing.T) {
	tests := []struct {
		name  string
		kind  platform.AudioKind
		opts  platform.AudioOpenOptions
		setup func(f *fakeSystem) *Service
		route string
	}{
		{"a format that is not allowed", platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Channels: 9}}, nil,
			"Service.Open->CheckFormat.AudioFormatUnsupported"},
		{"a device that is not there", platform.AudioPlayback, platform.AudioOpenOptions{ID: "Nope"}, nil, "Service.Open.AudioDeviceOpenFailed"},
		{"a microphone as a speaker", platform.AudioPlayback, platform.AudioOpenOptions{ID: "BuiltInMicrophoneDevice"}, nil, "Service.Open.AudioDeviceOpenFailed"},
		{"the queue refuses the format", platform.AudioPlayback, platform.AudioOpenOptions{}, func(f *fakeSystem) *Service {
			f.failNew = statusFormatUnsupported
			return f.service(nil)
		}, "Service.Open->device.build->statusError.AudioFormatUnsupported"},
		{"the queue cannot be made", platform.AudioPlayback, platform.AudioOpenOptions{}, func(f *fakeSystem) *Service {
			f.failNew = -12
			return f.service(nil)
		}, "Service.Open->device.build->statusError.AudioDeviceOpenFailed"},
		{"the queue refuses the device", platform.AudioPlayback, platform.AudioOpenOptions{ID: "UsbHeadset"}, func(f *fakeSystem) *Service {
			f.failSet = -12
			return f.service(nil)
		}, "Service.Open->device.build->statusError.AudioDeviceOpenFailed"},
		{"the queue does not start", platform.AudioPlayback, platform.AudioOpenOptions{}, func(f *fakeSystem) *Service {
			f.startSt = -12
			return f.service(nil)
		}, "Service.Open->device.startLocked->statusError.AudioDeviceOpenFailed"},
		{"the microphone is refused", platform.AudioCapture, platform.AudioOpenOptions{}, func(f *fakeSystem) *Service {
			return f.service(&fakePerm{})
		}, "Service.Open.AudioCapturePermissionDenied"},
		{"the permission cannot be asked", platform.AudioCapture, platform.AudioOpenOptions{}, func(f *fakeSystem) *Service {
			return f.service(&fakePerm{err: errors.New(tagFakePermission, "no")})
		}, "Service.Open.AudioCapturePermissionDenied"},
		{"the alive listener cannot be added", platform.AudioPlayback, platform.AudioOpenOptions{ID: "UsbHeadset"}, func(f *fakeSystem) *Service {
			s := f.service(nil)
			if _, err := s.Devices(platform.AudioPlayback); err != nil {
				f.t.Fatal(err)
			}
			f.failAdd = -12
			return s
		}, "Service.Open->statusError.AudioDeviceOpenFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeSystem(t)
			s := f.service(nil)
			if tt.setup != nil {
				s = tt.setup(f)
			}
			_, err := s.Open(context.Background(), tt.kind, tt.opts, &recorder{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route = %s, want %s", appErr.Route(), tt.route)
			}
			// Nothing is left of a device that did not open.
			if len(s.devices) != 0 {
				t.Error("the device is registered")
			}
			for i, q := range f.queues {
				if !q.disposed {
					t.Errorf("queue %d was left", i)
				}
			}
		})
	}
}

func TestOpenEndedContext(t *testing.T) {
	f := newFakeSystem(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.service(nil).Open(ctx, platform.AudioPlayback, platform.AudioOpenOptions{}, &recorder{})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Service.Open.AudioDeviceOpenFailed") {
		t.Errorf("error = %v", err)
	}
}

func TestUnavailableCoreAudio(t *testing.T) {
	want := errors.New(platform.TagAudioUnavailable, "no Core Audio")
	s := New(Options{})
	s.load = func() (*library, error) { return nil, want }
	if _, err := s.Devices(platform.AudioPlayback); err != want {
		t.Errorf("Devices = %v", err)
	}
	if _, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, &recorder{}); err != want {
		t.Errorf("Open = %v", err)
	}
}

func TestListenersThatCannotBeAdded(t *testing.T) {
	f := newFakeSystem(t)
	f.failAdd = -12
	_, err := f.service(nil).Devices(platform.AudioPlayback)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Service.Devices->Service.getLib->statusError.CoreAudioFailed") {
		t.Errorf("error = %v", err)
	}
}

func TestDevicesThatCannotBeRead(t *testing.T) {
	f := newFakeSystem(t)
	s := f.service(nil)
	s.load = func() (*library, error) {
		l := *f.lib
		l.getPropertyDataSize = func(uint32, *PropertyAddress, uint32, unsafe.Pointer, *uint32) int32 { return -1 }
		return &l, nil
	}
	_, err := s.Devices(platform.AudioPlayback)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("library.objects->statusError.CoreAudioFailed") {
		t.Errorf("error = %v", err)
	}
}

func TestPauseAndResumeThatFail(t *testing.T) {
	f := newFakeSystem(t)
	d := openPlayback(t, f.service(nil), platform.AudioOpenOptions{}, &recorder{})
	f.pauseSt = -12
	if appErr, ok := errors.FromError(d.Pause()); !ok || !appErr.HasRoute("device.Pause->statusError.CoreAudioFailed") {
		t.Errorf("Pause: %v", appErr)
	}
	f.pauseSt, f.startSt = 0, -12
	if appErr, ok := errors.FromError(d.Resume()); !ok || !appErr.HasRoute("device.Resume->device.startLocked->statusError.CoreAudioFailed") {
		t.Errorf("Resume: %v", appErr)
	}
	f.startSt = 0
	_ = d.Close()
}

func TestAPanicInTheHandlerGivesSilence(t *testing.T) {
	f := newFakeSystem(t)
	h := &recorder{panics: true}
	d := openPlayback(t, f.service(nil), platform.AudioOpenOptions{}, h)
	q := f.queues[0]
	b := (*QueueBuffer)(dl.Pointer(q.buffers[0]))
	for i := range memFloats(b.Data, 8) {
		memFloats(b.Data, 8)[i] = 1
	}
	d.onOutput(1, q.buffers[0]) // does not panic
	if memFloats(b.Data, 8)[0] != 0 || q.enqueued != queueBuffers+1 {
		t.Errorf("the buffer of a handler that panicked is not silence, or was not queued again")
	}
	_ = d.Close()
}

func TestStatusErrorTable(t *testing.T) {
	tests := []struct {
		status  int32
		opening bool
		tag     string
	}{
		{statusFormatUnsupported, true, platform.TagAudioFormatUnsupported},
		{statusParamErr, true, platform.TagAudioFormatUnsupported},
		{statusQueueInvalidParameter, true, platform.TagAudioFormatUnsupported},
		{-1, true, platform.TagAudioDeviceOpenFailed},
		{statusFormatUnsupported, false, tagCoreAudioFailed},
		{-1, false, tagCoreAudioFailed},
	}
	for _, tt := range tests {
		appErr, ok := errors.FromError(statusError("AudioQueueStart", tt.status, tt.opening))
		if !ok || appErr.Tag != tt.tag || appErr.Metadata["call"] != "AudioQueueStart" || appErr.Metadata["status"] != int(tt.status) {
			t.Errorf("statusError(%d, %v) = %+v", tt.status, tt.opening, appErr)
		}
	}
}
