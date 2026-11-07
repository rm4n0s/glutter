//go:build linux

package asensor

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
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// fakeLib scripts libandroid: a sensor manager with the sensors that a test gives it, event queues that hold the events a test
// pushes, and a looper whose poll waits for an event or a wake, as the real one does.
type fakeLib struct {
	t  *testing.T
	mu sync.Mutex
	cv *sync.Cond

	pkg          string
	noManager    bool
	sensors      map[int32]uintptr // type -> ASensor *
	minDelay     map[uintptr]int32
	noQueue      bool
	enableResult int32
	noLooper     bool

	queues    map[uintptr]*fakeQueue
	nextQueue uintptr
	woken     bool
	prepares  int
	released  int
}

type fakeQueue struct {
	sensor    uintptr
	enabled   bool
	destroyed bool
	usec      int32
	events    []fakeEvent
	ident     int32
	looper    uintptr
}

type fakeEvent struct {
	ts int64
	v  [3]float32
}

const (
	fakeAccel = uintptr(0xA1)
	fakeGyro  = uintptr(0xA2)
)

func newFake(t *testing.T) *fakeLib {
	f := &fakeLib{t: t, sensors: map[int32]uintptr{typeAccelerometer: fakeAccel, typeGyroscope: fakeGyro}, minDelay: map[uintptr]int32{},
		queues: map[uintptr]*fakeQueue{}, nextQueue: 0x5000}
	f.cv = sync.NewCond(&f.mu)
	return f
}

func (f *fakeLib) library() *library {
	return &library{
		getInstanceForPackage: func(pkg uintptr) uintptr {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.pkg = dl.GoString(pkg)
			if f.noManager {
				return 0
			}
			return 0x10
		},
		getDefaultSensor: func(mgr uintptr, kind int32) uintptr {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.sensors[kind]
		},
		createEventQueue: func(mgr, looper uintptr, ident int32, callback, data uintptr) uintptr {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.noQueue {
				return 0
			}
			f.nextQueue++
			f.queues[f.nextQueue] = &fakeQueue{ident: ident, looper: looper}
			return f.nextQueue
		},
		destroyEventQueue: func(mgr, queue uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.queues[queue].destroyed = true
			return 0
		},
		enableSensor: func(queue, sensor uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.enableResult < 0 {
				return f.enableResult
			}
			q := f.queues[queue]
			q.enabled, q.sensor = true, sensor
			return 0
		},
		disableSensor: func(queue, sensor uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.queues[queue].enabled = false
			return 0
		},
		setEventRate: func(queue, sensor uintptr, usec int32) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.queues[queue].usec = usec
			return 0
		},
		getEvents: func(queue, events uintptr, count uintptr) int64 {
			f.mu.Lock()
			defer f.mu.Unlock()
			q := f.queues[queue]
			if q.destroyed {
				f.t.Error("the backend read a queue that was destroyed")
			}
			n := min(len(q.events), int(count))
			for i := 0; i < n; i++ {
				e := q.events[i]
				base := events + uintptr(i)*eventSize
				*(*int32)(dl.Pointer(base)) = eventSize
				*(*int64)(dl.Pointer(base + eventTimestamp)) = e.ts
				copy(unsafe.Slice((*float32)(dl.Pointer(base+eventData)), 3), e.v[:])
			}
			q.events = q.events[n:]
			if n == 0 {
				return -11 // -EAGAIN
			}
			return int64(n)
		},
		getMinDelay: func(sensor uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.minDelay[sensor]
		},
		looperPrepare: func(opts int32) uintptr {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.prepares++
			if opts != looperPrepareAllowNonCallbacks {
				f.t.Errorf("the looper was prepared with %d", opts)
			}
			if f.noLooper {
				return 0
			}
			return 0x77
		},
		looperPoll: func(timeout int32, fd, events, data uintptr) int32 {
			f.mu.Lock()
			defer f.mu.Unlock()
			for {
				if f.woken {
					f.woken = false
					return pollWake
				}
				for _, q := range f.queues {
					if len(q.events) > 0 && !q.destroyed && q.enabled {
						return q.ident
					}
				}
				f.cv.Wait()
			}
		},
		looperWake: func(looper uintptr) {
			f.mu.Lock()
			f.woken = true
			f.mu.Unlock()
			f.cv.Broadcast()
		},
		looperAcquire: func(looper uintptr) {},
		looperRelease: func(looper uintptr) {
			f.mu.Lock()
			f.released++
			f.mu.Unlock()
		},
	}
}

// push puts events in the queue of the n-th device that was opened (counting from 1).
func (f *fakeLib) push(n int, events ...fakeEvent) {
	f.mu.Lock()
	q := f.queues[0x5000+uintptr(n)]
	q.events = append(q.events, events...)
	f.mu.Unlock()
	f.cv.Broadcast()
}

func (f *fakeLib) queue(n int) fakeQueue {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.queues[0x5000+uintptr(n)]
}

func (f *fakeLib) service(t *testing.T) *Service {
	s := New(Options{Package: "org.glutter.test"})
	s.load = func() (*library, error) { return f.library(), nil }
	t.Cleanup(s.Close)
	return s
}

// recorder is the handler of a sensor: it keeps the batches it was called with.
type recorder struct {
	mu      sync.Mutex
	batches [][]platform.SensorReading
	block   chan struct{}
	started chan struct{}
}

func (r *recorder) SensorReadings(in []platform.SensorReading) {
	if r.started != nil {
		select {
		case r.started <- struct{}{}:
		default:
		}
	}
	if r.block != nil {
		<-r.block
	}
	r.mu.Lock()
	r.batches = append(r.batches, append([]platform.SensorReading(nil), in...))
	r.mu.Unlock()
}

func (r *recorder) all() []platform.SensorReading {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []platform.SensorReading
	for _, b := range r.batches {
		out = append(out, b...)
	}
	return out
}

func (r *recorder) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.batches)
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

func TestOpenReadsEventsWithTheirTimestampsAndValues(t *testing.T) {
	f := newFake(t)
	s := f.service(t)
	rec := &recorder{}
	dev, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, rec)
	if err != nil {
		t.Fatal(err)
	}
	if f.pkg != "org.glutter.test" {
		t.Errorf("the manager was asked for the package %q", f.pkg)
	}
	q := f.queue(1)
	if !q.enabled || q.sensor != fakeAccel || q.ident != looperIdent || q.looper != 0x77 || q.usec != 16666 {
		t.Errorf("queue %+v", q)
	}
	if got := dev.Rate(); got < 59.9 || got > 60.1 {
		t.Errorf("Rate = %v", got)
	}
	f.push(1, fakeEvent{ts: 1_000_000, v: [3]float32{0.5, -9.8, 1}}, fakeEvent{ts: 17_000_000, v: [3]float32{0, 0, 9.8}})
	waitFor(t, "two readings", func() bool { return len(rec.all()) == 2 })
	got := rec.all()
	if got[0].Time != time.Millisecond || got[0].Values != [3]float64{0.5, float64(float32(-9.8)), 1} || got[1].Time != 17*time.Millisecond {
		t.Errorf("readings %+v", got)
	}
}

func TestMoreEventsThanOneCallBatchInPieces(t *testing.T) {
	f := newFake(t)
	s := f.service(t)
	rec := &recorder{}
	if _, err := s.Open(context.Background(), platform.SensorGyroscope, platform.SensorOpenOptions{}, rec); err != nil {
		t.Fatal(err)
	}
	var events []fakeEvent
	for i := 0; i < 40; i++ {
		events = append(events, fakeEvent{ts: int64(i), v: [3]float32{float32(i), 0, 0}})
	}
	f.push(1, events...)
	waitFor(t, "40 readings", func() bool { return len(rec.all()) == 40 })
	for i, r := range rec.all() {
		if r.Time != time.Duration(i) || r.Values[0] != float64(i) {
			t.Fatalf("reading %d = %+v", i, r)
		}
	}
	if q := f.queue(1); q.sensor != fakeGyro {
		t.Errorf("the gyroscope queue has the sensor %#x", q.sensor)
	}
	if c := rec.calls(); c != 3 {
		t.Errorf("40 events came in %d calls, want 3 (16, 16, 8)", c)
	}
}

func TestPeriodMicros(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		minDelay int32
		want     int32
	}{
		{"60 Hz", 60, 0, 16666},
		{"200 Hz", 200, 0, 5000},
		{"the sensor is slower: its minimum", 200, 10000, 10000},
		{"the sensor is faster than asked", 50, 5000, 20000},
		{"a sensor that reports on change", 100, -1, 10000},
		{"absurdly fast", 1e9, 0, 1},
	}
	for _, tt := range tests {
		if got := periodMicros(tt.rate, tt.minDelay); got != tt.want {
			t.Errorf("%s: periodMicros(%v, %d) = %d, want %d", tt.name, tt.rate, tt.minDelay, got, tt.want)
		}
	}
}

func TestTheRateTheAppAsksForAndTheSensorsMinimum(t *testing.T) {
	f := newFake(t)
	f.minDelay[fakeGyro] = 5000
	s := f.service(t)
	a, err := s.Open(context.Background(), platform.SensorGyroscope, platform.SensorOpenOptions{Rate: 1000}, &recorder{})
	if err != nil {
		t.Fatal(err)
	}
	if q := f.queue(1); q.usec != 5000 || a.Rate() != 200 {
		t.Errorf("asked for 1000 Hz of a sensor that takes 200: %d us, %v Hz", q.usec, a.Rate())
	}
	b, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{Rate: 25}, &recorder{})
	if err != nil {
		t.Fatal(err)
	}
	if q := f.queue(2); q.usec != 40000 || b.Rate() != 25 {
		t.Errorf("asked for 25 Hz: %d us, %v Hz", q.usec, b.Rate())
	}
}

func TestAvailable(t *testing.T) {
	f := newFake(t)
	delete(f.sensors, typeGyroscope)
	s := f.service(t)
	if !s.Available(platform.SensorAccelerometer) || s.Available(platform.SensorGyroscope) {
		t.Error("Available does not follow the sensors that the device has")
	}
	missing := New(Options{})
	missing.load = func() (*library, error) {
		return nil, errors.New(platform.TagSensorUnavailable, "no library")
	}
	if missing.Available(platform.SensorAccelerometer) {
		t.Error("a device without the library has a sensor")
	}
}

// TestOpenErrors: every way that Open fails, with its tag and its route.
func TestOpenErrors(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name  string
		setup func(*fakeLib)
		ctx   context.Context
		kind  platform.SensorKind
		tag   string
		route string
	}{
		{"no gyroscope", func(f *fakeLib) { delete(f.sensors, typeGyroscope) }, context.Background(), platform.SensorGyroscope,
			platform.TagSensorUnavailable, "Service.Open.SensorUnavailable"},
		{"no manager", func(f *fakeLib) { f.noManager = true }, context.Background(), platform.SensorAccelerometer,
			platform.TagSensorUnavailable, "Service.manager.SensorUnavailable"},
		{"no event queue", func(f *fakeLib) { f.noQueue = true }, context.Background(), platform.SensorAccelerometer,
			platform.TagSensorOpenFailed, "Service.Open.SensorOpenFailed"},
		{"the sensor cannot be enabled", func(f *fakeLib) { f.enableResult = -22 }, context.Background(), platform.SensorAccelerometer,
			platform.TagSensorOpenFailed, "Service.Open.SensorOpenFailed"},
		{"no looper", func(f *fakeLib) { f.noLooper = true }, context.Background(), platform.SensorAccelerometer,
			platform.TagSensorOpenFailed, "Service.looper.SensorOpenFailed"},
		{"the context ended", func(*fakeLib) {}, cancelled, platform.SensorAccelerometer,
			platform.TagSensorOpenFailed, "Service.Open.SensorOpenFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake(t)
			tt.setup(f)
			s := f.service(t)
			_, err := s.Open(tt.ctx, tt.kind, platform.SensorOpenOptions{}, &recorder{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("tag %s, route %s; want %s and %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
		})
	}
	// A queue that was made and could not be enabled is destroyed again.
	f := newFake(t)
	f.enableResult = -22
	s := f.service(t)
	if _, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, &recorder{}); err == nil {
		t.Fatal("Open succeeded")
	}
	if !f.queue(1).destroyed {
		t.Error("the queue of a sensor that failed to enable was left")
	}
}

func TestTheLibraryIsMissing(t *testing.T) {
	t.Setenv(LibraryEnv, "/nonexistent/libandroid.so")
	s := New(Options{})
	_, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, &recorder{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagSensorUnavailable {
		t.Fatalf("Open without the library: %v", err)
	}
	if !appErr.HasRoute("Service.Open.Service.manager.SensorUnavailable") && !appErr.HasRoute("loadLibrary.SensorUnavailable") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestCloseStopsTheReadings(t *testing.T) {
	f := newFake(t)
	s := f.service(t)
	rec := &recorder{}
	dev, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, rec)
	if err != nil {
		t.Fatal(err)
	}
	f.push(1, fakeEvent{ts: 1})
	waitFor(t, "a reading", func() bool { return len(rec.all()) == 1 })
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	if q := f.queue(1); q.enabled || !q.destroyed {
		t.Errorf("after Close the queue is enabled %v and destroyed %v", q.enabled, q.destroyed)
	}
	f.push(1, fakeEvent{ts: 2})
	time.Sleep(20 * time.Millisecond)
	if len(rec.all()) != 1 {
		t.Error("a closed sensor was called")
	}
	if err := dev.Close(); err != nil {
		t.Errorf("a second Close failed: %v", err)
	}
}

// TestCloseWaitsForTheHandler: Close does not return while the handler runs, and the handler is not called afterwards.
func TestCloseWaitsForTheHandler(t *testing.T) {
	f := newFake(t)
	s := f.service(t)
	rec := &recorder{block: make(chan struct{}), started: make(chan struct{}, 1)}
	dev, err := s.Open(context.Background(), platform.SensorGyroscope, platform.SensorOpenOptions{}, rec)
	if err != nil {
		t.Fatal(err)
	}
	f.push(1, fakeEvent{ts: 1})
	<-rec.started
	closed := make(chan struct{})
	go func() {
		_ = dev.Close() // the result is not what this test is about
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned while the handler was running")
	case <-time.After(50 * time.Millisecond):
	}
	close(rec.block)
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close never returned")
	}
}

func TestTwoSensorsHaveTheirOwnQueues(t *testing.T) {
	f := newFake(t)
	s := f.service(t)
	a, b := &recorder{}, &recorder{}
	if _, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, a); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(context.Background(), platform.SensorGyroscope, platform.SensorOpenOptions{}, b); err != nil {
		t.Fatal(err)
	}
	if f.prepares != 1 {
		t.Errorf("the looper was prepared %d times, want once", f.prepares)
	}
	f.push(2, fakeEvent{ts: 5, v: [3]float32{1, 2, 3}})
	waitFor(t, "the gyroscope's reading", func() bool { return len(b.all()) == 1 })
	f.push(1, fakeEvent{ts: 6})
	waitFor(t, "the accelerometer's reading", func() bool { return len(a.all()) == 1 })
	if len(a.all()) != 1 || len(b.all()) != 1 || b.all()[0].Values != [3]float64{1, 2, 3} {
		t.Errorf("accelerometer %v, gyroscope %v", a.all(), b.all())
	}
}

func TestServiceCloseClosesTheSensorsAndTheThread(t *testing.T) {
	f := newFake(t)
	s := f.service(t)
	rec := &recorder{}
	if _, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, rec); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if q := f.queue(1); q.enabled || !q.destroyed {
		t.Errorf("the queue is enabled %v and destroyed %v", q.enabled, q.destroyed)
	}
	f.mu.Lock()
	released := f.released
	f.mu.Unlock()
	if released != 1 {
		t.Errorf("the looper was released %d times", released)
	}
	_, err := s.Open(context.Background(), platform.SensorAccelerometer, platform.SensorOpenOptions{}, rec)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagSensorUnavailable {
		t.Errorf("Open after Close: %v", err)
	}
	s.Close() // twice is fine
}

func TestProcessName(t *testing.T) {
	if n := processName(); n == "" || strings.ContainsRune(n, 0) {
		t.Errorf("processName = %q", n)
	}
}

// TestConstantsAgainstTheHeaders compiles a check of the numbers and of the layout of ASensorEvent against the NDK's
// <android/sensor.h> and <android/looper.h> with the compiler of the NDK (it only parses, for Android, and runs nothing). It is
// skipped when the NDK is not installed.
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
		"ASENSOR_TYPE_ACCELEROMETER": typeAccelerometer, "ASENSOR_TYPE_GYROSCOPE": typeGyroscope,
		"ALOOPER_PREPARE_ALLOW_NON_CALLBACKS": looperPrepareAllowNonCallbacks, "ALOOPER_POLL_WAKE": pollWake,
		"ALOOPER_POLL_CALLBACK": pollCallback, "ALOOPER_POLL_TIMEOUT": pollTimeout, "ALOOPER_POLL_ERROR": pollError,
		"sizeof(ASensorEvent)": eventSize, "offsetof(ASensorEvent, timestamp)": eventTimestamp, "offsetof(ASensorEvent, data)": eventData,
	}
	var src strings.Builder
	src.WriteString("#include <android/sensor.h>\n#include <android/looper.h>\n#include <stddef.h>\n")
	for name, v := range checks {
		src.WriteString("_Static_assert((int)(" + name + ") == " + strconv.Itoa(v) + ", \"" + name + "\");\n")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "check.c")
	if err := os.WriteFile(file, []byte(src.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, arch := range []string{"aarch64", "x86_64"} {
		compiler := strings.Replace(matches[0], "aarch64", arch, 1)
		if out, err := exec.Command(compiler, "-fsyntax-only", file).CombinedOutput(); err != nil {
			t.Errorf("the headers of the NDK disagree for %s:\n%s", arch, out)
		}
	}
}
