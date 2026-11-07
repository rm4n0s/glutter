package sensors_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/sensors"
	"github.com/rm4n0s/glutter/widgets"
)

func open(t *testing.T, kind sensors.Kind, opts ...sensors.Opts) (*gluttertest.Tester, *sensors.System, *sensors.Sensor, *gluttertest.TestSensor) {
	t.Helper()
	tester := gluttertest.New(t)
	tester.PumpWidget(widgets.NewSizedBox(nil)) // frames run once an app is on the screen
	sys := sensors.ForBinding(tester.Binding)
	s, err := sys.Open(context.Background(), kind, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return tester, sys, s, tester.Sensors().Open(kind)
}

func TestSystemAvailabilityAndOpen(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*gluttertest.TestSensors)
		kind  sensors.Kind
		has   bool
		tag   string
	}{
		{"accelerometer", func(*gluttertest.TestSensors) {}, sensors.Accelerometer, true, ""},
		{"gyroscope", func(*gluttertest.TestSensors) {}, sensors.Gyroscope, true, ""},
		{"no gyroscope", func(s *gluttertest.TestSensors) { s.SetAvailable(sensors.Gyroscope, false) }, sensors.Gyroscope, false, platform.TagSensorUnavailable},
		{"the other kind is still there", func(s *gluttertest.TestSensors) { s.SetAvailable(sensors.Gyroscope, false) }, sensors.Accelerometer, true, ""},
		{"refused by the user", func(s *gluttertest.TestSensors) { s.SetDenied(sensors.Accelerometer, true) }, sensors.Accelerometer, true, platform.TagSensorPermissionDenied},
		{"open fails", func(s *gluttertest.TestSensors) {
			s.FailOpen(errors.New(platform.TagSensorOpenFailed, "the fake open was set to fail"))
		}, sensors.Accelerometer, true, platform.TagSensorOpenFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := gluttertest.New(t)
			tt.setup(tester.Sensors())
			sys := sensors.ForBinding(tester.Binding)
			if !sys.Available() {
				t.Fatal("the test platform has a sensors service")
			}
			if got := sys.Has(tt.kind); got != tt.has {
				t.Errorf("Has(%v) = %v, want %v", tt.kind, got, tt.has)
			}
			s, err := sys.Open(context.Background(), tt.kind)
			if tt.tag == "" {
				if err != nil {
					t.Fatalf("Open: %v", err)
				}
				if s.Kind() != tt.kind || s.Rate() != 60 {
					t.Errorf("sensor %v at %v Hz", s.Kind(), s.Rate())
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag {
				t.Fatalf("Open: %v, want the tag %s", err, tt.tag)
			}
		})
	}
}

// TestOpenRoutes: the errors that Open makes itself have a route that ends in their tag.
func TestOpenRoutes(t *testing.T) {
	tester := gluttertest.New(t)
	sys := sensors.ForBinding(tester.Binding)
	tester.Sensors().SetAvailable(sensors.Gyroscope, false)
	_, err := sys.Open(context.Background(), sensors.Gyroscope)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("Sensors.Open.SensorUnavailable") && !appErr.HasRoute("Open.SensorUnavailable") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = sys.Open(ctx, sensors.Accelerometer)
	appErr, ok = errors.FromError(err)
	if !ok || appErr.Tag != platform.TagSensorOpenFailed {
		t.Fatalf("Open with an ended context: %v", err)
	}
	if !appErr.HasRoute("Open.SensorOpenFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestOpenPanicsWithTwoOpts(t *testing.T) {
	tester := gluttertest.New(t)
	sys := sensors.ForBinding(tester.Binding)
	defer func() {
		if recover() == nil {
			t.Error("two Opts did not panic")
		}
	}()
	_, _ = sys.Open(context.Background(), sensors.Accelerometer, sensors.Opts{}, sensors.Opts{})
}

func TestRateIsPassedOn(t *testing.T) {
	_, _, s, ts := open(t, sensors.Gyroscope, sensors.Opts{Rate: 200})
	if ts.RequestedRate() != 200 || s.Rate() != 200 || ts.Rate() != 200 {
		t.Errorf("rate asked %v, sensor %v, backend %v", ts.RequestedRate(), s.Rate(), ts.Rate())
	}
}

// TestReadingsAreBatchedOnceAFrame: every reading that arrived since the last frame is given, together, to each listener in
// the next frame, on the UI thread.
func TestReadingsAreBatchedOnceAFrame(t *testing.T) {
	tester, _, s, ts := open(t, sensors.Accelerometer)
	var calls [][]sensors.Reading
	s.AddListener(func(r []sensors.Reading) { calls = append(calls, append([]sensors.Reading(nil), r...)) })
	var second int
	s.AddListener(func(r []sensors.Reading) { second += len(r) })

	ts.SendValues(10*time.Millisecond, 0, 0, 9.8)
	ts.SendValues(20*time.Millisecond, 0.1, 0, 9.7)
	ts.SendValues(30*time.Millisecond, 0.2, 0, 9.6)
	if len(calls) != 0 {
		t.Fatalf("a listener was called before the frame: %v", calls)
	}
	tester.Pump(16 * time.Millisecond)
	if len(calls) != 1 || len(calls[0]) != 3 {
		t.Fatalf("after one frame the listener got %v", calls)
	}
	want := sensors.Reading{Timestamp: 20 * time.Millisecond, Values: [3]float64{0.1, 0, 9.7}}
	if calls[0][1] != want {
		t.Errorf("second reading = %+v, want %+v", calls[0][1], want)
	}
	if second != 3 {
		t.Errorf("the second listener got %d readings", second)
	}

	// No reading, no call; a later reading, a later frame.
	tester.Pump(16 * time.Millisecond)
	if len(calls) != 1 {
		t.Errorf("a frame with no reading called the listener: %v", calls)
	}
	ts.SendValues(40*time.Millisecond, 1, 2, 3)
	tester.Pump(16 * time.Millisecond)
	if len(calls) != 2 || len(calls[1]) != 1 || calls[1][0].Values != [3]float64{1, 2, 3} {
		t.Errorf("the next batch = %v", calls)
	}
}

func TestLatestIsCurrentBeforeTheFrame(t *testing.T) {
	_, _, s, ts := open(t, sensors.Accelerometer)
	if _, ok := s.Latest(); ok {
		t.Error("a sensor that has not read has a latest reading")
	}
	ts.SendValues(time.Second, 1, 2, 3)
	ts.SendValues(2*time.Second, 4, 5, 6)
	r, ok := s.Latest()
	if !ok || r.Timestamp != 2*time.Second || r.Values != [3]float64{4, 5, 6} {
		t.Errorf("Latest = %+v, %v", r, ok)
	}
}

func TestRemovedListenerAndClosedSensorGetNothing(t *testing.T) {
	tester, sys, s, ts := open(t, sensors.Accelerometer)
	var a, b int
	remove := s.AddListener(func(r []sensors.Reading) { a += len(r) })
	s.AddListener(func(r []sensors.Reading) { b += len(r) })
	ts.SendValues(0, 1, 1, 1)
	remove()
	tester.Pump(16 * time.Millisecond)
	if a != 0 || b != 1 {
		t.Errorf("removed listener got %d, the other %d", a, b)
	}
	ts.SendValues(time.Millisecond, 1, 1, 1)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	tester.Pump(16 * time.Millisecond)
	if b != 1 {
		t.Errorf("a closed sensor called its listener (%d)", b)
	}
	if !ts.Closed() || tester.Sensors().Count() != 0 {
		t.Errorf("the backend's sensor is closed %v, %d open", ts.Closed(), tester.Sensors().Count())
	}
	ts.SendValues(2*time.Millisecond, 1, 1, 1)
	if s.Close() != nil {
		t.Error("Close twice failed")
	}
	_ = sys
}

func TestListenerAddedAfterReadingsGetsThem(t *testing.T) {
	tester, _, s, ts := open(t, sensors.Gyroscope)
	ts.SendValues(0, 1, 2, 3)
	var got []sensors.Reading
	s.AddListener(func(r []sensors.Reading) { got = append(got, r...) })
	tester.Pump(16 * time.Millisecond)
	if len(got) != 1 {
		t.Errorf("a listener added before the frame got %v", got)
	}
}

func TestReadingsFromAnotherGoroutine(t *testing.T) {
	tester, _, s, _ := open(t, sensors.Accelerometer)
	var mu sync.Mutex
	total := 0
	s.AddListener(func(r []sensors.Reading) {
		mu.Lock()
		total += len(r)
		mu.Unlock()
	})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				s.SensorReadings([]platform.SensorReading{{Time: time.Duration(i), Values: [3]float64{1, 2, 3}}})
			}
		}()
	}
	wg.Wait()
	tester.Pump(16 * time.Millisecond)
	tester.Pump(16 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if total != 400 {
		t.Errorf("the listener got %d readings of 400", total)
	}
}

func TestTooManyPendingReadingsDropTheOldest(t *testing.T) {
	_, _, s, ts := open(t, sensors.Accelerometer)
	batch := make([]sensors.Reading, 3000)
	for i := range batch {
		batch[i] = sensors.Reading{Timestamp: time.Duration(i), Values: [3]float64{float64(i), 0, 0}}
	}
	ts.Send(batch...)
	ts.Send(batch...)
	if s.Dropped() != 6000-4096 {
		t.Errorf("dropped %d readings, want %d", s.Dropped(), 6000-4096)
	}
	if r, _ := s.Latest(); r.Values[0] != 2999 {
		t.Errorf("latest = %v", r)
	}
}

func TestSystemCloseClosesTheSensors(t *testing.T) {
	tester, sys, _, ts := open(t, sensors.Accelerometer)
	sys.Close()
	if !ts.Closed() || tester.Sensors().Count() != 0 {
		t.Error("the sensor is still open after the system closed")
	}
	if _, err := sys.Open(context.Background(), sensors.Accelerometer); err == nil {
		t.Error("a closed system opened a sensor")
	}
}

func TestReleaseHookClosesTheSensors(t *testing.T) {
	tester := gluttertest.New(t)
	sys := sensors.ForBinding(tester.Binding)
	if _, err := sys.Open(context.Background(), sensors.Gyroscope); err != nil {
		t.Fatal(err)
	}
	tester.RunReleaseHooks()
	if tester.Sensors().Count() != 0 {
		t.Error("the app ended and a sensor stayed open")
	}
}

func TestFeed(t *testing.T) {
	tester := gluttertest.New(t)
	tester.PumpWidget(widgets.NewSizedBox(nil))
	closed := 0
	f := sensors.NewFeed(tester.Binding, sensors.Gyroscope, 250, func() error { closed++; return nil })
	s := f.Sensor()
	if s.Kind() != sensors.Gyroscope || s.Rate() != 250 {
		t.Errorf("feed sensor %v at %v", s.Kind(), s.Rate())
	}
	var got []sensors.Reading
	s.AddListener(func(r []sensors.Reading) { got = append(got, r...) })
	f.Put(sensors.Reading{Timestamp: 1, Values: [3]float64{1, 0, 0}}, sensors.Reading{Timestamp: 2, Values: [3]float64{0, 1, 0}})
	tester.Pump(16 * time.Millisecond)
	if len(got) != 2 || got[1].Values != [3]float64{0, 1, 0} {
		t.Errorf("got %v", got)
	}
	if err := s.Close(); err != nil || closed != 1 {
		t.Errorf("Close = %v, onClose called %d times", err, closed)
	}
	if err := s.Close(); err != nil || closed != 1 {
		t.Errorf("a second Close = %v, onClose called %d times", err, closed)
	}
}

func TestKindString(t *testing.T) {
	for k, want := range map[sensors.Kind]string{sensors.Accelerometer: "accelerometer", sensors.Gyroscope: "gyroscope", 9: "unknown"} {
		if k.String() != want {
			t.Errorf("Kind(%d) = %q, want %q", k, k.String(), want)
		}
	}
}
