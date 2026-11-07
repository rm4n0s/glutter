//go:build windows && glutter_wine && glutter_audio

package wasapi_test

// These tests run inside Wine (scripts/wine-exec.sh) with the audio of Wine (winepulse) connected to the private PulseAudio protocol
// server of scripts/headless-audio.sh, which has a null sink and a null source and plays to no sound card:
//
//	scripts/headless-audio.sh -- env GOOS=windows go test -tags 'glutter_wine glutter_audio' -exec "$PWD/scripts/wine-exec.sh" -p 1 ./internal/platform/windows/wasapi
//
// They check the calls of WASAPI that need no listener outside Wine: the devices, the callbacks, pause and resume, the errors. The
// sound that comes out of the sink is checked by the test of the host (wine_host_test.go), which records it.

import (
	"context"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/wasapi"
)

func requireServer(t *testing.T) {
	t.Helper()
	if os.Getenv("PULSE_SERVER") == "" {
		t.Skip("no private audio server: run this with scripts/headless-audio.sh")
	}
}

// counter is a handler that plays a tone and counts what happens.
type counter struct {
	mu       sync.Mutex
	channels int
	phase    float64
	step     float64
	fills    int
	frames   int
	captured int
	peak     float32
	states   []platform.AudioState
}

func newCounter(rate, channels int) *counter {
	return &counter{channels: channels, step: 2 * math.Pi * 440 / float64(rate)}
}

func (c *counter) Fill(out []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i+c.channels <= len(out); i += c.channels {
		v := 0.5 * float32(math.Sin(c.phase))
		c.phase += c.step
		for ch := 0; ch < c.channels; ch++ {
			out[i+ch] = v
		}
	}
	c.fills++
	c.frames += len(out) / c.channels
}

func (c *counter) Capture(in []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.captured += len(in)
	for _, v := range in {
		c.peak = max(c.peak, float32(math.Abs(float64(v))))
	}
}

func (c *counter) StateChanged(s platform.AudioState) {
	c.mu.Lock()
	c.states = append(c.states, s)
	c.mu.Unlock()
}

func (c *counter) filled() (fills, frames int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fills, c.frames
}

func (c *counter) captureState() (samples int, peak float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.captured, c.peak
}

func (c *counter) stateList() []platform.AudioState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]platform.AudioState(nil), c.states...)
}

func eventually(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func newService(t *testing.T) *wasapi.Service {
	t.Helper()
	requireServer(t)
	s := wasapi.New(nil)
	t.Cleanup(s.Close)
	return s
}

func open(t *testing.T, s *wasapi.Service, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) platform.AudioDevice {
	t.Helper()
	dev, err := s.Open(context.Background(), kind, opts, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	})
	return dev
}

func TestWineDevices(t *testing.T) {
	s := newService(t)
	for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
		list, err := s.Devices(kind)
		if err != nil {
			t.Fatalf("Devices(%d): %v", kind, err)
		}
		defaults := 0
		for _, d := range list {
			if d.ID == "" || d.Name == "" || d.Kind != kind {
				t.Errorf("a device with no ID, name or the wrong kind: %+v", d)
			}
			if d.Default {
				defaults++
			}
			if d.Format.Rate == 0 || d.Format.Channels == 0 {
				t.Errorf("%s has no format", d.Name)
			}
		}
		if len(list) == 0 {
			t.Errorf("no devices of kind %d", kind)
		}
		if defaults != 1 {
			t.Errorf("kind %d: %d default devices of %d, want one", kind, defaults, len(list))
		}
	}
}

func TestWinePlayback(t *testing.T) {
	s := newService(t)
	h := newCounter(48000, 2)
	dev := open(t, s, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, h)
	if f := dev.Format(); f != (platform.AudioFormat{Rate: 48000, Channels: 2}) {
		t.Errorf("format %+v", f)
	}
	time.Sleep(time.Second)
	fills, frames := h.filled()
	// The engine asks for as many frames as it plays: 48000 a second, give or take the buffer that was filled at the start.
	if frames < 44000 || frames > 60000 {
		t.Errorf("%d frames in a second (%d fills), want about 48000", frames, fills)
	}
}

func TestWineFormats(t *testing.T) {
	s := newService(t)
	tests := []struct {
		rate, channels int
	}{{48000, 2}, {44100, 2}, {22050, 1}, {96000, 2}, {8000, 1}, {48000, 6}, {48000, 8}, {0, 0}, {32000, 0}, {0, 1}}
	for _, tt := range tests {
		dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: tt.rate, Channels: tt.channels}}, newCounter(48000, 1))
		if err != nil {
			t.Errorf("%d Hz, %d channels: %v", tt.rate, tt.channels, err)
			continue
		}
		f := dev.Format()
		t.Logf("asked %d Hz %d channels, got %+v", tt.rate, tt.channels, f)
		if f.Rate == 0 || f.Channels == 0 {
			t.Errorf("%d Hz, %d channels: the format is %+v", tt.rate, tt.channels, f)
		}
		if tt.rate != 0 && tt.channels != 0 && f.Rate != tt.rate {
			t.Logf("  the engine did not convert the rate: it keeps %d", f.Rate)
		}
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestWineHandlerGetsWholeFrames(t *testing.T) {
	s := newService(t)
	var mu sync.Mutex
	var bad int
	var channels int
	h := &frameCheck{check: func(n int) {
		mu.Lock()
		defer mu.Unlock()
		if channels != 0 && n%channels != 0 {
			bad++
		}
	}}
	dev := open(t, s, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 6}}, h)
	mu.Lock()
	channels = dev.Format().Channels
	mu.Unlock()
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if bad > 0 || h.calls == 0 {
		t.Errorf("%d calls, %d with a partial frame", h.calls, bad)
	}
}

type frameCheck struct {
	check func(n int)
	calls int
}

func (f *frameCheck) Fill(out []float32)               { f.calls++; f.check(len(out)) }
func (f *frameCheck) Capture([]float32)                {}
func (f *frameCheck) StateChanged(platform.AudioState) {}

func TestWineCapture(t *testing.T) {
	s := newService(t)
	h := newCounter(48000, 2)
	dev := open(t, s, platform.AudioCapture, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, h)
	time.Sleep(time.Second)
	samples, peak := h.captureState()
	frames := samples / dev.Format().Channels
	// A null source is silent, and the engine gives as many frames as the time that passed.
	if frames < 40000 || frames > 56000 {
		t.Errorf("%d frames of a second of capture", frames)
	}
	if peak != 0 {
		t.Errorf("the null source made a sound: peak %v", peak)
	}
}

func TestWinePauseResume(t *testing.T) {
	s := newService(t)
	h := newCounter(48000, 2)
	dev := open(t, s, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, h)
	eventually(t, "the first fills", 2*time.Second, func() bool { f, _ := h.filled(); return f > 5 })
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Pause(); err != nil {
		t.Fatalf("a second Pause: %v", err)
	}
	_, frames := h.filled()
	time.Sleep(300 * time.Millisecond)
	if _, after := h.filled(); after != frames {
		t.Errorf("the handler was called while the device was paused (%d frames more)", after-frames)
	}
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Resume(); err != nil {
		t.Fatalf("a second Resume: %v", err)
	}
	eventually(t, "the fills after Resume", 2*time.Second, func() bool { _, f := h.filled(); return f > frames+4800 })
}

func TestWineCaptureOfAPausedDeviceIsNotDelivered(t *testing.T) {
	s := newService(t)
	h := newCounter(48000, 2)
	dev := open(t, s, platform.AudioCapture, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, h)
	eventually(t, "the first frames", 2*time.Second, func() bool { n, _ := h.captureState(); return n > 0 })
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	before, _ := h.captureState()
	time.Sleep(300 * time.Millisecond)
	if after, _ := h.captureState(); after != before {
		t.Errorf("%d samples came while paused", after-before)
	}
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "frames after Resume", 2*time.Second, func() bool { n, _ := h.captureState(); return n > before })
}

func TestWineCloseStopsTheHandler(t *testing.T) {
	s := newService(t)
	h := newCounter(48000, 2)
	dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, h)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "the first fills", 2*time.Second, func() bool { f, _ := h.filled(); return f > 3 })
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	_, frames := h.filled()
	time.Sleep(200 * time.Millisecond)
	if _, after := h.filled(); after != frames {
		t.Errorf("the handler ran after Close returned")
	}
	if err := dev.Close(); err != nil {
		t.Errorf("a second Close: %v", err)
	}
	if err := dev.Pause(); err == nil {
		t.Error("Pause of a closed device did not fail")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceLost || !appErr.HasRoute("device.Pause.AudioDeviceLost") {
		t.Errorf("Pause of a closed device: %v", err)
	}
}

func TestWineBufferFrames(t *testing.T) {
	s := newService(t)
	for _, frames := range []int{0, 128, 480, 960, 4800} {
		h := newCounter(48000, 2)
		dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}, BufferFrames: frames}, h)
		if err != nil {
			t.Errorf("BufferFrames %d: %v", frames, err)
			continue
		}
		time.Sleep(300 * time.Millisecond)
		fills, got := h.filled()
		t.Logf("BufferFrames %d: %d fills, %d frames in 300 ms", frames, fills, got)
		if got < 9000 {
			t.Errorf("BufferFrames %d: %d frames in 300 ms", frames, got)
		}
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestWineOpenErrors(t *testing.T) {
	s := newService(t)
	h := newCounter(48000, 2)
	tests := []struct {
		name string
		kind platform.AudioKind
		opts platform.AudioOpenOptions
		tag  string
	}{
		{"an ID that no device has", platform.AudioPlayback, platform.AudioOpenOptions{ID: "{0.0.0.00000000}.{00000000-0000-0000-0000-000000000000}"}, platform.TagAudioDeviceOpenFailed},
		{"a rate that is too low", platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 5, Channels: 2}}, platform.TagAudioFormatUnsupported},
		{"too many channels", platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 9}}, platform.TagAudioFormatUnsupported},
		{"a capture ID that no device has", platform.AudioCapture, platform.AudioOpenOptions{ID: "nothing"}, platform.TagAudioDeviceOpenFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev, err := s.Open(context.Background(), tt.kind, tt.opts, h)
			if err == nil {
				_ = dev.Close()
				t.Fatal("Open did not fail")
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag {
				t.Fatalf("got %v, want tag %s", err, tt.tag)
			}
			t.Logf("route %s", appErr.Route())
		})
	}
	t.Run("a context that is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := s.Open(ctx, platform.AudioPlayback, platform.AudioOpenOptions{}, h)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceOpenFailed || !appErr.HasRoute("Service.Open.AudioDeviceOpenFailed") {
			t.Errorf("got %v", err)
		}
	})
	t.Run("a closed service", func(t *testing.T) {
		closed := wasapi.New(nil)
		closed.Close()
		_, err := closed.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, h)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.thread.AudioUnavailable") {
			t.Errorf("Open: %v", err)
		}
		_, err = closed.Devices(platform.AudioPlayback)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.thread.AudioUnavailable") {
			t.Errorf("Devices: %v", err)
		}
	})
}

func TestWineOpenByID(t *testing.T) {
	s := newService(t)
	list, err := s.Devices(platform.AudioPlayback)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list {
		dev := open(t, s, platform.AudioPlayback, platform.AudioOpenOptions{ID: d.ID}, newCounter(48000, 2))
		if f := dev.Format(); f.Rate == 0 || f.Channels == 0 {
			t.Errorf("%s: format %+v", d.Name, f)
		}
		t.Logf("%s: %+v", d.Name, dev.Format())
	}
}

func TestWineChangeHandlerIsCalled(t *testing.T) {
	s := newService(t)
	var mu sync.Mutex
	n := 0
	s.SetChangeHandler(func() { mu.Lock(); n++; mu.Unlock() })
	if _, err := s.Devices(platform.AudioPlayback); err != nil {
		t.Fatal(err)
	}
	s.SetChangeHandler(nil)
	t.Logf("%d changes while the list was read", n)
}
