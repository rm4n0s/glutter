//go:build linux && !android && glutter_audio

package alsa

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/pulse"
)

type handler struct {
	mu       sync.Mutex
	phase    float64
	rate     float64
	channels int
	fills    int
	captured []float32
	states   []platform.AudioState
}

func (s *handler) Fill(out []float32) {
	for i := 0; i < len(out); i += s.channels {
		v := float32(0.5 * math.Sin(s.phase))
		s.phase += 2 * math.Pi * 440 / s.rate
		for c := 0; c < s.channels; c++ {
			out[i+c] = v
		}
	}
	s.mu.Lock()
	s.fills++
	s.mu.Unlock()
}

func (s *handler) Capture(in []float32) {
	s.mu.Lock()
	s.captured = append(s.captured, in...)
	s.mu.Unlock()
}

func (s *handler) StateChanged(st platform.AudioState) {
	s.mu.Lock()
	s.states = append(s.states, st)
	s.mu.Unlock()
}

func TestLiveDevices(t *testing.T) {
	svc := New()
	for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
		list, err := svc.Devices(kind)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%v: %+v", kind, list)
	}
}

func TestLivePlaybackHeardOnTheMonitor(t *testing.T) {
	ctx := context.Background()
	rec := &handler{channels: 2}
	ps := pulse.New(pulse.Options{})
	defer ps.Close()
	cap, err := ps.Open(ctx, platform.AudioCapture, platform.AudioOpenOptions{ID: "glutter_sink.monitor", Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, rec)
	if err != nil {
		t.Fatal(err)
	}
	svc := New()
	play := &handler{rate: 48000, channels: 2}
	dev, err := svc.Open(ctx, platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, play)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	play.mu.Lock()
	n := play.fills
	play.mu.Unlock()
	time.Sleep(200 * time.Millisecond)
	play.mu.Lock()
	if play.fills != n {
		t.Errorf("fills went from %d to %d while paused", n, play.fills)
	}
	play.mu.Unlock()
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cap.Close(); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var peak float32
	for _, v := range rec.captured {
		peak = max(peak, v)
	}
	t.Logf("fills %d, captured %d samples, peak %v", play.fills, len(rec.captured), peak)
	if peak < 0.4 {
		t.Errorf("peak %v", peak)
	}
}
