package ios

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
)

func TestAccelerometerValues(t *testing.T) {
	// A device that lies flat and face up: Core Motion reads -1 g on z, and the platform wants about +9.8.
	got := accelerometerValues([3]float64{0, 0, -1})
	if got[0] != 0 || got[1] != 0 || math.Abs(got[2]-standardGravity) > 1e-12 {
		t.Errorf("flat, face up: %v", got)
	}
	// Standing upright (the top up): Core Motion reads -1 g on y, and the platform +9.8.
	if got := accelerometerValues([3]float64{0, -1, 0}); math.Abs(got[1]-standardGravity) > 1e-12 {
		t.Errorf("upright: %v", got)
	}
	// Every axis is turned: a push to the right (+x in Core Motion) is negative.
	if got := accelerometerValues([3]float64{0.5, 0.25, 0.125}); got != [3]float64{-0.5 * standardGravity, -0.25 * standardGravity, -0.125 * standardGravity} {
		t.Errorf("sign: %v", got)
	}
}

func TestMotionRate(t *testing.T) {
	tests := []struct{ asked, want float64 }{{0, 60}, {-5, 60}, {30, 30}, {100, 100}, {250, 100}, {0.2, 1}, {1, 1}}
	for _, tc := range tests {
		if got := motionRate(tc.asked); got != tc.want {
			t.Errorf("motionRate(%v) = %v, want %v", tc.asked, got, tc.want)
		}
	}
}

// fakeStream records how the hub drives Core Motion.
type fakeStream struct {
	starts []float64
	stops  int
}

func (f *fakeStream) hub() *motionHub {
	return &motionHub{start: func(r float64) { f.starts = append(f.starts, r) }, stop: func() { f.stops++ }}
}

type collector struct {
	mu   sync.Mutex
	seen [][]platform.SensorReading
	// inHandler is held by a test to keep the handler running.
	block chan struct{}
	in    chan struct{}
}

func (c *collector) SensorReadings(r []platform.SensorReading) {
	if c.in != nil {
		c.in <- struct{}{}
		<-c.block
	}
	c.mu.Lock()
	c.seen = append(c.seen, append([]platform.SensorReading(nil), r...))
	c.mu.Unlock()
}

func (c *collector) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.seen)
}

func TestMotionHubStartsAndStopsTheStream(t *testing.T) {
	var f fakeStream
	h := f.hub()
	a := h.open(30, &collector{})
	if len(f.starts) != 1 || f.starts[0] != 30 || a.Rate() != 30 {
		t.Fatalf("starts %v, rate %v", f.starts, a.Rate())
	}
	// A faster one speeds the stream up; the slower one still has what it asked for in the readings it gets (it gets all of them).
	b := h.open(100, &collector{})
	if len(f.starts) != 2 || f.starts[1] != 100 {
		t.Errorf("starts = %v", f.starts)
	}
	// A third that is slower than the fastest changes nothing.
	c := h.open(10, &collector{})
	if len(f.starts) != 2 {
		t.Errorf("starts = %v", f.starts)
	}
	_ = c.Close()
	_ = b.Close() // the fastest is gone: the stream slows down to 30
	if len(f.starts) != 3 || f.starts[2] != 30 || f.stops != 0 {
		t.Errorf("starts %v, stops %d", f.starts, f.stops)
	}
	_ = a.Close()
	if f.stops != 1 {
		t.Errorf("stops = %d", f.stops)
	}
	// Closing again does nothing.
	_ = a.Close()
	if f.stops != 1 || len(f.starts) != 3 {
		t.Errorf("starts %v, stops %d", f.starts, f.stops)
	}
}

func TestMotionHubDeliversToEveryOpenDevice(t *testing.T) {
	var f fakeStream
	h := f.hub()
	ca, cb := &collector{}, &collector{}
	a := h.open(0, ca)
	h.open(0, cb)
	readings := []platform.SensorReading{{Time: time.Second, Values: [3]float64{1, 2, 3}}}
	h.deliver(readings)
	if ca.count() != 1 || cb.count() != 1 || ca.seen[0][0].Values != [3]float64{1, 2, 3} {
		t.Errorf("a %d, b %d", ca.count(), cb.count())
	}
	_ = a.Close()
	h.deliver(readings)
	if ca.count() != 1 || cb.count() != 2 {
		t.Errorf("after closing a: a %d, b %d", ca.count(), cb.count())
	}
}

func TestMotionDeviceCloseWaitsForTheHandler(t *testing.T) {
	var f fakeStream
	h := f.hub()
	c := &collector{block: make(chan struct{}), in: make(chan struct{})}
	d := h.open(0, c)
	go h.deliver([]platform.SensorReading{{}})
	<-c.in // the handler is running
	closed := make(chan struct{})
	go func() { _ = d.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("Close returned while the handler ran")
	case <-time.After(100 * time.Millisecond):
	}
	close(c.block)
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after the handler ended")
	}
	// The handler is never called again.
	h.deliver([]platform.SensorReading{{}})
	if c.count() != 1 {
		t.Errorf("handler calls = %d", c.count())
	}
}

func TestBootTime(t *testing.T) {
	if got := bootTime(1.5); got != 1500*time.Millisecond {
		t.Errorf("bootTime(1.5) = %v", got)
	}
}
