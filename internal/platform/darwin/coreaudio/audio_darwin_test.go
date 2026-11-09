//go:build darwin && glutter_audio

package coreaudio

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
)

// sine is a handler that plays a sine of an amplitude far below what a speaker can make audible (about -100 dB: under the step of
// 16 bits), and counts the frames it has been asked for, so that the test can tell whether the device keeps time.
type sine struct {
	channels int
	rate     float64
	frames   atomic.Int64
	calls    atomic.Int64
	phase    float64
	mu       sync.Mutex
	states   []platform.AudioState
	maxCall  atomic.Int64
}

func (s *sine) Fill(out []float32) {
	n := len(out) / s.channels
	for i := range n {
		v := float32(1e-5 * math.Sin(s.phase))
		s.phase += 2 * math.Pi * 440 / s.rate
		for c := range s.channels {
			out[i*s.channels+c] = v
		}
	}
	s.frames.Add(int64(n))
	s.calls.Add(1)
	if int64(n) > s.maxCall.Load() {
		s.maxCall.Store(int64(n))
	}
}

func (s *sine) Capture([]float32) {}
func (s *sine) StateChanged(st platform.AudioState) {
	s.mu.Lock()
	s.states = append(s.states, st)
	s.mu.Unlock()
}

// TestRealPlaybackKeepsTime opens the default output of this Mac with an inaudible sine, and checks that the device asks for about
// as many frames as the time that passed, in buffers of the size that was asked for; that pausing stops the calls and resuming
// starts them; and that Close is the end of them. A machine with no output device skips it. It never records (no microphone is
// opened in a test).
func TestRealPlaybackKeepsTime(t *testing.T) {
	s := New(Options{})
	list, err := s.Devices(platform.AudioPlayback)
	if err != nil || len(list) == 0 {
		t.Skipf("no output device here (%v, %d devices)", err, len(list))
	}
	h := &sine{channels: 2, rate: 48000}
	dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}, BufferFrames: 480}, h)
	if err != nil {
		t.Skipf("the output device cannot be opened here: %v", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = dev.Close()
		}
	}()
	if dev.Format() != (platform.AudioFormat{Rate: 48000, Channels: 2}) {
		t.Errorf("Format = %+v", dev.Format())
	}
	start := time.Now()
	time.Sleep(time.Second)
	frames := h.frames.Load()
	elapsed := time.Since(start).Seconds()
	// The device asks for the frames it plays, plus the buffers it holds (three of 480).
	want := elapsed * 48000
	if f := float64(frames); f < want*0.8 || f > want*1.2+3*480 {
		t.Errorf("after %.2f s the handler was asked for %d frames, want about %.0f", elapsed, frames, want)
	}
	if h.maxCall.Load() != 480 {
		t.Errorf("the handler was asked for %d frames at a time, want 480", h.maxCall.Load())
	}

	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // what was in flight
	paused := h.calls.Load()
	time.Sleep(300 * time.Millisecond)
	if got := h.calls.Load(); got != paused {
		t.Errorf("the handler was called %d times while the device was paused", got-paused)
	}
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if h.calls.Load() == paused {
		t.Error("the handler was not called after Resume")
	}

	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	after := h.calls.Load()
	time.Sleep(200 * time.Millisecond)
	if got := h.calls.Load(); got != after {
		t.Errorf("the handler was called %d times after Close", got-after)
	}
	h.mu.Lock()
	t.Logf("states: %v", h.states)
	h.mu.Unlock()
}
