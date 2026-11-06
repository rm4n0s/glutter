package gamepadhid

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// recHandler is the Handler of a Device under test: what it is told goes to channels.
type recHandler struct {
	ready  chan Caps
	failed chan error
	states chan padstd.Snapshot
	lost   chan struct{}
	other  chan string
}

func newRecHandler() *recHandler {
	return &recHandler{ready: make(chan Caps, 1), failed: make(chan error, 1), states: make(chan padstd.Snapshot, 64), lost: make(chan struct{}, 4), other: make(chan string, 64)}
}

func (h *recHandler) Ready(c Caps)                                { h.ready <- c }
func (h *recHandler) Failed(err error)                            { h.failed <- err }
func (h *recHandler) State(now time.Duration, s *padstd.Snapshot) { h.states <- *s }
func (h *recHandler) Touch(t platform.PadTouch)                   { h.other <- "touch" }
func (h *recHandler) Sensor(s platform.PadSensor)                 { h.other <- "sensor" }
func (h *recHandler) Battery(b platform.PadBattery)               { h.other <- "battery" }
func (h *recHandler) Lost()                                       { h.lost <- struct{}{} }

func within[T any](t *testing.T, what string, c <-chan T) T {
	t.Helper()
	select {
	case v := <-c:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// stubDriver is a driver that records what it is asked.
type stubDriver struct {
	mu       sync.Mutex
	startErr error
	calls    []string
	motors   [][2]uint16
	closed   atomic.Bool
	lose     atomic.Bool
}

func (d *stubDriver) Start(io IO) (Caps, error) {
	return Caps{Name: "Stub", Rumble: true, RumbleTriggers: true, LED: true, Sensors: true}, d.startErr
}

func (d *stubDriver) Report(now time.Duration, data []byte, out Sink) {
	var s padstd.Snapshot
	s.Buttons[gamepaddb.South] = data[0] != 0
	out.State(now, &s)
}

func (d *stubDriver) Tick(now time.Duration, out Sink) {
	if d.lose.Load() {
		out.Lost()
	}
}

func (d *stubDriver) record(s string) {
	d.mu.Lock()
	d.calls = append(d.calls, s)
	d.mu.Unlock()
}

func (d *stubDriver) Rumble(low, high uint16) error {
	d.record("rumble")
	d.mu.Lock()
	d.motors = append(d.motors, [2]uint16{low, high})
	d.mu.Unlock()
	return nil
}
func (d *stubDriver) RumbleTriggers(l, r uint16) error { d.record("triggers"); return nil }
func (d *stubDriver) SetLED(r, g, b uint8) error       { d.record("led"); return nil }
func (d *stubDriver) SetSensors(on bool) error         { d.record("sensors"); return nil }
func (d *stubDriver) Close()                           { d.closed.Store(true) }

func (d *stubDriver) rumbles() [][2]uint16 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([][2]uint16(nil), d.motors...)
}

func fastOptions(clock func() time.Duration) Options {
	return Options{Clock: clock, ReadTimeout: time.Millisecond, WriteGap: time.Millisecond}
}

func TestDeviceReportsWhatTheDriverReads(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	d := startWith(drv, DeviceInfo{Path: "/dev/test"}, tr, h, fastOptions(nil))
	if c := within(t, "ready", h.ready); c.Name != "Stub" {
		t.Errorf("caps = %+v", c)
	}
	tr.in <- []byte{1}
	if s := within(t, "a state", h.states); !s.Buttons[gamepaddb.South] {
		t.Error("the button was not down")
	}
	tr.in <- []byte{0}
	if s := within(t, "a state", h.states); s.Buttons[gamepaddb.South] {
		t.Error("the button did not go up")
	}
	d.Close()
	if !drv.closed.Load() {
		t.Error("the driver was not closed")
	}
	select {
	case <-h.lost:
		t.Error("Lost was called after Close")
	default:
	}
	select {
	case <-tr.closed:
	default:
		t.Error("the transport was not closed")
	}
}

func TestDeviceThatFailsToStart(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{startErr: errors.New(tagFakeNoAnswer, "no answer")}
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	if err := within(t, "a failure", h.failed); err == nil || err.Error() != "no answer" {
		t.Errorf("err = %v", err)
	}
	d.Close()
	select {
	case <-tr.closed:
	default:
		t.Error("the transport of a device that failed was not closed")
	}
	select {
	case <-h.ready:
		t.Error("a device that failed was ready")
	default:
	}
}

func TestDeviceThatIsUnplugged(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	within(t, "ready", h.ready)
	tr.failRead <- errors.New(tagFakeNoDevice, "no such device")
	within(t, "lost", h.lost)
	d.Close() // closing a device that is gone is fine
	if !drv.closed.Load() {
		t.Error("the driver of a pad that was lost was not closed")
	}
}

func TestDeviceWhoseDriverSaysThePadIsGone(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	within(t, "ready", h.ready)
	drv.lose.Store(true)
	within(t, "lost", h.lost)
	d.Close()
}

// clock is a clock that a test sets.
type clock struct{ ns atomic.Int64 }

func (c *clock) now() time.Duration  { return time.Duration(c.ns.Load()) }
func (c *clock) set(d time.Duration) { c.ns.Store(int64(d)) }

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestDeviceStopsTheRumbleWhenItsTimeIsOver(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	var c clock
	c.set(time.Second)
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(c.now))
	defer d.Close()
	within(t, "ready", h.ready)
	if err := d.Rumble(1000, 2000, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got := drv.rumbles(); len(got) != 1 || got[0] != [2]uint16{1000, 2000} {
		t.Fatalf("rumbles = %v", got)
	}
	c.set(time.Second + 400*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	if len(drv.rumbles()) != 1 {
		t.Fatal("the rumble was stopped too soon")
	}
	c.set(time.Second + 600*time.Millisecond)
	waitFor(t, "the stop", func() bool { return len(drv.rumbles()) == 2 })
	if got := drv.rumbles()[1]; got != [2]uint16{0, 0} {
		t.Errorf("the stop = %v", got)
	}
}

func TestDeviceSendsALongRumbleAgain(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	var c clock
	c.set(time.Second)
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(c.now))
	defer d.Close()
	within(t, "ready", h.ready)
	if err := d.Rumble(1000, 2000, time.Minute); err != nil {
		t.Fatal(err)
	}
	c.set(3*time.Second + 100*time.Millisecond)
	waitFor(t, "the second rumble", func() bool { return len(drv.rumbles()) == 2 })
	if got := drv.rumbles()[1]; got != [2]uint16{1000, 2000} {
		t.Errorf("the rumble sent again = %v", got)
	}
}

func TestDeviceRumbleOfZeroDurationStops(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	defer d.Close()
	within(t, "ready", h.ready)
	if err := d.Rumble(1000, 2000, 0); err != nil {
		t.Fatal(err)
	}
	if got := drv.rumbles(); len(got) != 1 || got[0] != [2]uint16{0, 0} {
		t.Errorf("rumbles = %v", got)
	}
}

func TestDevicePassesTheCommandsToTheDriver(t *testing.T) {
	tr, h, drv := newFakeTransport(), newRecHandler(), &stubDriver{}
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	defer d.Close()
	within(t, "ready", h.ready)
	if err := d.RumbleTriggers(1, 1, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := d.SetLED(1, 2, 3); err != nil {
		t.Fatal(err)
	}
	if err := d.SetSensors(true); err != nil {
		t.Fatal(err)
	}
	drv.mu.Lock()
	defer drv.mu.Unlock()
	if len(drv.calls) != 3 || drv.calls[0] != "triggers" || drv.calls[1] != "led" || drv.calls[2] != "sensors" {
		t.Errorf("calls = %v", drv.calls)
	}
}

func TestDeviceIsNotReadyWhileItsDriverStarts(t *testing.T) {
	drv := &blockingDriver{release: make(chan struct{})}
	tr, h := newFakeTransport(), newRecHandler()
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	// The driver is still starting: the device is not ready, and nothing is told to its handler.
	d.mu.Lock()
	ready := d.ready
	d.mu.Unlock()
	if ready {
		t.Fatal("the device was ready before its driver started")
	}
	close(drv.release)
	within(t, "ready", h.ready)
	d.Close()
}

// blockingDriver is a driver whose Start waits.
type blockingDriver struct {
	stubDriver
	release chan struct{}
}

func (b *blockingDriver) Start(io IO) (Caps, error) {
	<-b.release
	return b.stubDriver.Start(io)
}

func TestDeviceWithTheRealDriver(t *testing.T) {
	tr, h := newFakeTransport(), newRecHandler()
	d, ok := Start(sonyPS4(BusUSB), tr, h, fastOptions(nil))
	if !ok {
		t.Fatal("no driver for a PlayStation 4 pad")
	}
	caps := within(t, "ready", h.ready)
	if caps.Name != "PS4 Controller" {
		t.Errorf("caps = %+v", caps)
	}
	// The effects report that the driver sends on start is written by the queue.
	waitFor(t, "the first report", func() bool { return len(tr.writes()) == 1 })
	if w := tr.writes()[0]; len(w) != 32 || w[0] != 0x05 {
		t.Errorf("first write = %x", w)
	}
	tr.in <- ps4Packet{b0: 0x28}.usb()
	if s := within(t, "a state", h.states); !s.Buttons[gamepaddb.South] {
		t.Error("the cross was not read")
	}
	if err := d.Rumble(0xFFFF, 0, time.Minute); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the rumble report", func() bool { return len(tr.writes()) == 2 })
	if w := tr.writes()[1]; w[1] != 0x01 || w[5] != 0xFF {
		t.Errorf("rumble write = %x", w)
	}
	d.Close()
}

func TestStartWithoutADriver(t *testing.T) {
	tr := newFakeTransport()
	if d, ok := Start(DeviceInfo{Vendor: 0x046d, Product: 0xc31c}, tr, newRecHandler(), Options{}); ok || d != nil {
		t.Fatal("a keyboard got a driver")
	}
	select {
	case <-tr.closed:
		t.Error("the transport of a device without a driver was closed")
	default:
	}
}

func TestOutboxReplacesWhatIsQueued(t *testing.T) {
	gate := make(chan struct{})
	var mu sync.Mutex
	var got [][]byte
	tr := &gatedTransport{gate: gate, record: func(d []byte) {
		mu.Lock()
		got = append(got, append([]byte(nil), d...))
		mu.Unlock()
	}}
	o := newOutbox(tr, time.Millisecond)
	defer o.close()
	// The first report is taken by the goroutine, which waits at the gate; the next ones queue.
	_ = o.send([]byte{1, 1}, nil)
	waitFor(t, "the first write to start", func() bool { return tr.started.Load() })
	_ = o.send([]byte{2, 1}, nil)
	_ = o.send([]byte{2, 2}, nil) // replaces the last one: same ID and length
	_ = o.send([]byte{3, 3, 3}, nil)
	_ = o.send([]byte{4, 4}, func(p []byte) bool { return p[0] == 3 }) // another length: queued
	_ = o.send([]byte{2, 9}, func(p []byte) bool { return false })     // merge says no: queued
	close(gate)
	waitFor(t, "the writes", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == 5 })
	mu.Lock()
	defer mu.Unlock()
	want := [][]byte{{1, 1}, {2, 2}, {3, 3, 3}, {4, 4}, {2, 9}}
	for i := range want {
		mustEqual(t, "write", got[i], want[i])
	}
}

func TestOutboxMergesWhenTheFunctionSaysSo(t *testing.T) {
	gate := make(chan struct{})
	var mu sync.Mutex
	var got [][]byte
	tr := &gatedTransport{gate: gate, record: func(d []byte) {
		mu.Lock()
		got = append(got, append([]byte(nil), d...))
		mu.Unlock()
	}}
	o := newOutbox(tr, time.Millisecond)
	defer o.close()
	_ = o.send([]byte{9, 9}, nil)
	waitFor(t, "the first write to start", func() bool { return tr.started.Load() })
	// The function sees the report that is queued and decides whether the new one replaces it: here, when their second byte is the same.
	sameSecond := func(b byte) func([]byte) bool { return func(p []byte) bool { return p[1] == b } }
	_ = o.send([]byte{1, 5, 0}, sameSecond(5))
	_ = o.send([]byte{1, 5, 1}, sameSecond(5)) // replaces it: the function says so
	_ = o.send([]byte{1, 6, 2}, sameSecond(6)) // not: the one that is queued has 5
	close(gate)
	waitFor(t, "the writes", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == 3 })
	mu.Lock()
	defer mu.Unlock()
	mustEqual(t, "second", got[1], []byte{1, 5, 1})
	mustEqual(t, "third", got[2], []byte{1, 6, 2})
}

func TestOutboxClosedDropsWhatIsSent(t *testing.T) {
	tr := newFakeTransport()
	o := newOutbox(tr, time.Millisecond)
	o.close()
	o.close() // twice is fine
	_ = o.send([]byte{1}, nil)
	time.Sleep(10 * time.Millisecond)
	if len(tr.writes()) != 0 {
		t.Error("a closed outbox wrote")
	}
}

// gatedTransport is a transport whose writes wait for a gate.
type gatedTransport struct {
	fakeTransport
	gate    chan struct{}
	started atomic.Bool
	record  func([]byte)
}

func (g *gatedTransport) Write(data []byte) (int, error) {
	g.started.Store(true)
	<-g.gate
	g.record(data)
	return len(data), nil
}

// The tags of the errors of the doubles of this file.
const (
	tagFakeNoAnswer = "FakeNoAnswer"
	tagFakeNoDevice = "FakeNoDevice"
)
