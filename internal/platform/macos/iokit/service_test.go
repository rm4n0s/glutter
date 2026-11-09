package iokit

import (
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
)

const (
	tagFakeManagerFailed = "FakeManagerFailed"
	tagFakeHIDClosed     = "FakeHIDClosed"
	tagFakeNoFeature     = "FakeNoFeature"
	tagFakeOpenFailed    = "FakeOpenFailed"
)

// recorder is the PadHandler under test. Its methods run on the test's goroutine (the UI thread), as the service's do.
type recorder struct {
	added   []platform.PadInfo
	gone    []platform.PadID
	events  []platform.PadEvent
	batts   []platform.PadBattery
	touches []platform.PadTouch
}

func (r *recorder) PadAdded(i platform.PadInfo)      { r.added = append(r.added, i) }
func (r *recorder) PadRemoved(id platform.PadID)     { r.gone = append(r.gone, id) }
func (r *recorder) PadEvents(es []platform.PadEvent) { r.events = append(r.events, es...) }
func (r *recorder) PadTouch(t platform.PadTouch)     { r.touches = append(r.touches, t) }
func (r *recorder) PadSensor(platform.PadSensor)     {}
func (r *recorder) PadBattery(b platform.PadBattery) { r.batts = append(r.batts, b) }
func (r *recorder) take() []platform.PadEvent        { e := r.events; r.events = nil; return e }
func (r *recorder) padEvents(k platform.PadEventKind) (out []platform.PadEvent) {
	for _, e := range r.events {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}

// fakeManager is the Manager under test.
type fakeManager struct {
	cb      Callbacks
	started int
	stopped int
	failing error
}

func (m *fakeManager) Start(cb Callbacks) error {
	if m.failing != nil {
		return m.failing
	}
	m.cb = cb
	m.started++
	return nil
}
func (m *fakeManager) Stop() { m.stopped++ }

// fakeDevice is a Device under test.
type fakeDevice struct {
	info     DeviceInfo
	elements []RawElement
	values   map[uint32]int32
	released int
	openHID  func() (gamepadhid.Transport, error)
}

func (d *fakeDevice) Info() DeviceInfo       { return d.info }
func (d *fakeDevice) Elements() []RawElement { return d.elements }
func (d *fakeDevice) Release()               { d.released++ }
func (d *fakeDevice) Value(c uint32) (int32, bool) {
	v, ok := d.values[c]
	return v, ok
}
func (d *fakeDevice) OpenHID() (gamepadhid.Transport, error) {
	if d.openHID == nil {
		return nil, errors.New(tagFakeOpenFailed, "no HID reports in this test")
	}
	return d.openHID()
}

// A Logitech-like pad: two buttons, two axes (listed in the wrong order), a hat.
func genericPad() *fakeDevice {
	return &fakeDevice{
		info: DeviceInfo{Vendor: 0x046d, Product: 0xc216, Version: 0x0305, VendorName: "Logitech", ProductName: "Dual Action", Transport: "USB",
			UsagePage: pageGenericDesktop, Usage: usageGamePad, Path: "0x14100000"},
		elements: []RawElement{
			{Cookie: 21, Type: ElementInputMisc, Page: pageGenericDesktop, Usage: usageY, Max: 255},
			{Cookie: 20, Type: ElementInputMisc, Page: pageGenericDesktop, Usage: usageX, Max: 255},
			{Cookie: 10, Type: ElementInputButton, Page: pageButton, Usage: 1, Max: 1},
			{Cookie: 11, Type: ElementInputButton, Page: pageButton, Usage: 2, Max: 1},
			{Cookie: 30, Type: ElementInputMisc, Page: pageGenericDesktop, Usage: usageHatswitch, Max: 7},
		},
		values: map[uint32]int32{20: 128, 21: 0, 10: 0, 11: 1, 30: 8},
	}
}

// fixture is a service with a fake manager and a queue for what is posted to the UI thread.
type fixture struct {
	t *testing.T
	m *fakeManager
	s *Service
	h *recorder
	q chan func()
}

func newFixture(t *testing.T, noHID bool) *fixture {
	t.Helper()
	f := &fixture{t: t, m: &fakeManager{}, h: &recorder{}, q: make(chan func(), 64)}
	// The clock is the real one: the HID drivers run on goroutines of their own and wait by it.
	f.s = New(Options{Manager: f.m, Post: func(fn func()) { f.q <- fn }, DisableHID: noHID})
	return f
}

func (f *fixture) start() {
	f.t.Helper()
	if err := f.s.Start(f.h); err != nil {
		f.t.Fatal(err)
	}
}

// pump runs what the goroutines of the drivers posted, until cond holds.
func (f *fixture) pump(what string, cond func() bool) {
	f.t.Helper()
	deadline := time.After(5 * time.Second)
	for !cond() {
		select {
		case fn := <-f.q:
			fn()
		case <-deadline:
			f.t.Fatalf("timed out waiting for %s", what)
		case <-time.After(time.Millisecond):
		}
	}
}

func TestServiceNamesTheMappingPlatform(t *testing.T) {
	if got := New(Options{Manager: &fakeManager{}}).MappingPlatform(); got != "Mac OS X" {
		t.Errorf("MappingPlatform = %q", got)
	}
}

func TestServiceReportsAPadWithItsValuesAndEvents(t *testing.T) {
	f := newFixture(t, true)
	f.start()
	d := genericPad()
	f.m.cb.Added(d)
	if len(f.h.added) != 1 {
		t.Fatalf("added = %d", len(f.h.added))
	}
	info := f.h.added[0]
	if info.Name != gamepaddb.DeviceName(0x046d, 0xc216, "Logitech", "Dual Action") || info.Axes != 2 || info.Buttons != 2 || info.Hats != 1 || info.Standard ||
		info.Path != "0x14100000" || info.Battery != -1 {
		t.Errorf("info = %+v", info)
	}
	want := gamepaddb.NewGUID(gamepaddb.BusUSB, 0x046d, 0xc216, 0x0305, "Logitech", "Dual Action", 0, 0)
	if gamepaddb.GUID(info.GUID) != want {
		t.Errorf("GUID = %s, want %s", gamepaddb.GUID(info.GUID), want)
	}
	// The first events are the values the pad has: every axis (a trigger rests at the bottom of its range), and the buttons and
	// hats that are not at rest. The axes are in the order of usage: X then Y.
	got := f.h.take()
	type ev struct {
		k platform.PadEventKind
		i int
		v int
	}
	var seen []ev
	for _, e := range got {
		seen = append(seen, ev{e.Kind, e.Index, e.Value})
	}
	wantEvents := []ev{{platform.PadAxis, 0, 128}, {platform.PadAxis, 1, -32768}, {platform.PadButton, 1, 1}}
	if len(seen) != len(wantEvents) {
		t.Fatalf("initial events = %v, want %v", seen, wantEvents)
	}
	for i := range wantEvents {
		if seen[i] != wantEvents[i] {
			t.Errorf("event %d = %v, want %v", i, seen[i], wantEvents[i])
		}
	}

	// A new value becomes an event; a value that repeats does not.
	f.m.cb.Input(d, 10, 1)
	f.m.cb.Input(d, 10, 1)
	f.m.cb.Input(d, 30, 2) // hat: right
	f.m.cb.Input(d, 30, 8) // hat: released
	f.m.cb.Input(d, 20, 255)
	f.m.cb.Input(d, 99, 1) // an element the pad does not use
	got = f.h.take()
	if len(got) != 4 {
		t.Fatalf("events = %+v", got)
	}
	if got[0].Kind != platform.PadButton || got[0].Index != 0 || got[0].Value != 1 || got[0].Pad != info.ID {
		t.Errorf("button event = %+v", got[0])
	}
	if got[1].Kind != platform.PadHat || got[1].Value != hatRight || got[2].Value != 0 {
		t.Errorf("hat events = %+v %+v", got[1], got[2])
	}
	if got[3].Kind != platform.PadAxis || got[3].Value != 32767 {
		t.Errorf("axis event = %+v", got[3])
	}
	if got[1].Time < got[0].Time || got[0].Time == 0 && got[1].Time == 0 {
		t.Errorf("the events are not stamped by the clock: %v, %v", got[0].Time, got[1].Time)
	}

	f.m.cb.Removed(d)
	if len(f.h.gone) != 1 || f.h.gone[0] != info.ID || d.released != 1 {
		t.Errorf("gone = %v, released = %d", f.h.gone, d.released)
	}
	// What the pad does after it has gone is nothing.
	f.m.cb.Input(d, 10, 0)
	if len(f.h.events) != 0 {
		t.Errorf("an event of a pad that went: %v", f.h.events)
	}
}

func TestServiceLeavesOutWhatIsNotAPad(t *testing.T) {
	f := newFixture(t, true)
	f.start()
	keyboard := genericPad()
	keyboard.info.Usage = 0x06
	empty := genericPad()
	empty.elements = []RawElement{{Cookie: 1, Type: ElementInputMisc, Page: 0xFF00, Usage: 1}}
	f.m.cb.Added(keyboard)
	f.m.cb.Added(empty)
	if len(f.h.added) != 0 {
		t.Errorf("added %+v", f.h.added)
	}
	if keyboard.released != 1 || empty.released != 1 {
		t.Errorf("released = %d, %d", keyboard.released, empty.released)
	}
	// Taking away a device that was never reported is no event.
	f.m.cb.Removed(keyboard)
	if len(f.h.gone) != 0 {
		t.Errorf("gone = %v", f.h.gone)
	}
}

func TestServiceNumbersEachPadOnce(t *testing.T) {
	f := newFixture(t, true)
	f.start()
	a, b := genericPad(), genericPad()
	f.m.cb.Added(a)
	f.m.cb.Added(b)
	f.m.cb.Added(a) // the same device again
	f.m.cb.Removed(a)
	f.m.cb.Added(genericPad())
	if len(f.h.added) != 3 {
		t.Fatalf("added = %d", len(f.h.added))
	}
	seen := map[platform.PadID]bool{}
	for _, i := range f.h.added {
		if seen[i.ID] {
			t.Errorf("pad id %d was used twice", i.ID)
		}
		seen[i.ID] = true
	}
}

func TestServiceStartFailureIsReturnedUntouched(t *testing.T) {
	f := newFixture(t, true)
	want := errors.New(tagFakeManagerFailed, "no HID manager")
	f.m.failing = want
	err := f.s.Start(f.h)
	if err != want {
		t.Fatalf("Start = %v, want the manager's error", err)
	}
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagFakeManagerFailed {
		t.Errorf("error = %v", err)
	}
	// A service that did not start can be stopped, and started again.
	f.s.Stop()
	f.m.failing = nil
	f.start()
}

func TestServiceStopReleasesAndSilencesTheManager(t *testing.T) {
	f := newFixture(t, true)
	f.start()
	d := genericPad()
	f.m.cb.Added(d)
	f.h.take()
	f.s.Stop()
	f.s.Stop()
	if f.m.stopped != 1 || d.released != 1 {
		t.Errorf("stopped = %d, released = %d", f.m.stopped, d.released)
	}
	// A late callback of the manager reaches nothing, and the device is let go.
	cb := f.m.cb
	late := genericPad()
	cb.Added(late)
	cb.Input(d, 10, 0)
	if len(f.h.added) != 1 || len(f.h.events) != 0 || late.released != 1 {
		t.Errorf("late callbacks: added %d events %d released %d", len(f.h.added), len(f.h.events), late.released)
	}
	// It starts again, and finds the pad again.
	f.start()
	f.m.cb.Added(genericPad())
	if len(f.h.added) != 2 || f.m.started != 2 {
		t.Errorf("added = %d, started = %d", len(f.h.added), f.m.started)
	}
}

func TestServiceRefusesWhatARawPadCannotDo(t *testing.T) {
	f := newFixture(t, true)
	f.start()
	f.m.cb.Added(genericPad())
	id := f.h.added[0].ID
	tests := []struct {
		name  string
		err   error
		route string
	}{
		{"rumble", f.s.Rumble(id, 1, 1, time.Second), "Service.Rumble.GamepadRumbleUnavailable"},
		{"triggers", f.s.RumbleTriggers(id, 1, 1, time.Second), "Service.RumbleTriggers.GamepadRumbleUnavailable"},
		{"light", f.s.SetLED(id, 1, 2, 3), "Service.SetLED.GamepadLEDUnavailable"},
		{"sensors", f.s.SetPadSensors(id, true), "Service.SetPadSensors.GamepadSensorsUnavailable"},
		{"a pad that is not there", f.s.Rumble(999, 1, 1, time.Second), "Service.Rumble.GamepadRumbleUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.err)
			if !ok {
				t.Fatalf("error = %T %v", tt.err, tt.err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route = %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
}

// fakeHID is the transport of a HID driver under test.
type fakeHID struct {
	in       chan []byte
	failRead chan error
	closed   chan struct{}
	once     sync.Once

	mu      sync.Mutex
	written [][]byte
}

func newFakeHID() *fakeHID {
	return &fakeHID{in: make(chan []byte, 64), failRead: make(chan error, 1), closed: make(chan struct{})}
}

func (f *fakeHID) Read(buf []byte, timeout time.Duration) (int, error) {
	select {
	case r := <-f.in:
		return copy(buf, r), nil
	case err := <-f.failRead:
		return 0, err
	case <-f.closed:
		return 0, errors.New(tagFakeHIDClosed, "closed")
	case <-time.After(timeout):
		return 0, nil
	}
}

func (f *fakeHID) Write(data []byte) (int, error) {
	f.mu.Lock()
	f.written = append(f.written, append([]byte(nil), data...))
	f.mu.Unlock()
	return len(data), nil
}
func (f *fakeHID) SendFeature(data []byte) (int, error) { return len(data), nil }
func (f *fakeHID) GetFeature([]byte) (int, error) {
	return -1, errors.New(tagFakeNoFeature, "no such feature report")
}
func (f *fakeHID) Close() error { f.once.Do(func() { close(f.closed) }); return nil }

func (f *fakeHID) writes() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.written...)
}

func (f *fakeHID) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

func (f *fixture) waitWrites(tr *fakeHID, n int) {
	f.t.Helper()
	f.pump("writes", func() bool { return len(tr.writes()) >= n })
}

// dualShock4 is a DualShock 4 on USB, which has a HID driver.
func dualShock4(tr *fakeHID) *fakeDevice {
	d := genericPad()
	d.info = DeviceInfo{Vendor: 0x054c, Product: 0x05c4, Version: 0x0100, ProductName: "Wireless Controller", Transport: "USB",
		UsagePage: pageGenericDesktop, Usage: usageGamePad, Path: "0x14200000"}
	d.openHID = func() (gamepadhid.Transport, error) { return tr, nil }
	return d
}

// ps4Report is the input report of a DualShock 4 on USB with the buttons byte given: the cross is 0x28.
func ps4Report(buttons byte) []byte {
	r := make([]byte, 64)
	r[0] = 1
	r[1], r[2], r[3], r[4] = 128, 128, 128, 128
	r[5] = buttons
	return r
}

func TestServiceServesAPadWithItsHIDDriver(t *testing.T) {
	f := newFixture(t, false)
	f.start()
	tr := newFakeHID()
	d := dualShock4(tr)
	f.m.cb.Added(d)
	f.pump("the pad", func() bool { return len(f.h.added) == 1 })
	info := f.h.added[0]
	if !info.Standard || info.Name != "PS4 Controller" || info.Path != "0x14200000" || !info.Rumble || !info.LED || !info.Sensors || info.Touchpads != 1 {
		t.Errorf("info = %+v", info)
	}
	if g := gamepaddb.GUID(info.GUID); g[14] != 'h' {
		t.Errorf("the GUID does not have the signature of the HID drivers: %s", g)
	}
	// The elements of the device are not read for a pad that a driver serves.
	f.m.cb.Input(d, 10, 1)
	for _, e := range f.h.events {
		if e.Kind == platform.PadButton && e.Index == 0 {
			t.Errorf("an element event of a pad that has a driver: %+v", e)
		}
	}
	f.waitWrites(tr, 1)
	tr.in <- ps4Report(0x28)
	f.pump("the cross", func() bool {
		for _, e := range f.h.events {
			if e.Kind == platform.PadButton && e.Index == int(gamepaddb.South) && e.Value == 1 && e.Pad == info.ID {
				return true
			}
		}
		return false
	})
	if err := f.s.Rumble(info.ID, 0xFFFF, 0, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := f.s.SetLED(info.ID, 1, 2, 3); err != nil {
		t.Fatal(err)
	}
	f.waitWrites(tr, 3)
	w := tr.writes()
	if w[1][1] != 0x01 || w[1][5] != 0xFF || w[2][1] != 0x02 || w[2][6] != 1 || w[2][7] != 2 || w[2][8] != 3 {
		t.Errorf("writes = %x, %x", w[1][:10], w[2][:10])
	}
	if err := f.s.RumbleTriggers(info.ID, 1, 1, time.Second); err == nil {
		t.Error("a PlayStation 4 pad rumbled its triggers")
	}
	if err := f.s.SetPadSensors(info.ID, true); err != nil {
		t.Errorf("the sensors cannot be turned on: %v", err)
	}

	// The device goes: the driver is ended, the transport closed, and the removal reaches the handler once.
	f.m.cb.Removed(d)
	f.pump("the removal", func() bool { return len(f.h.gone) == 1 })
	if f.h.gone[0] != info.ID || !tr.isClosed() || d.released != 1 {
		t.Errorf("gone = %v closed = %v released = %d", f.h.gone, tr.isClosed(), d.released)
	}
	if err := f.s.Rumble(info.ID, 1, 1, time.Second); err == nil {
		t.Error("a pad that went was told to rumble")
	}
}

func TestServiceRemovesAPadWhoseHIDLinkIsLost(t *testing.T) {
	f := newFixture(t, false)
	f.start()
	tr := newFakeHID()
	d := dualShock4(tr)
	f.m.cb.Added(d)
	f.pump("the pad", func() bool { return len(f.h.added) == 1 })
	tr.failRead <- errors.New(tagFakeHIDClosed, "no such device")
	f.pump("the removal", func() bool { return len(f.h.gone) == 1 })
	if f.h.gone[0] != f.h.added[0].ID {
		t.Errorf("removed %d, want %d", f.h.gone[0], f.h.added[0].ID)
	}
	f.pump("the transport to close", tr.isClosed)
	// The manager says the device has gone later: that is not a second removal.
	f.m.cb.Removed(d)
	if len(f.h.gone) != 1 {
		t.Errorf("gone = %v", f.h.gone)
	}
}

func TestServiceReadsAPadThatHasNoHIDReportsThroughItsElements(t *testing.T) {
	f := newFixture(t, false)
	f.start()
	d := dualShock4(nil)
	d.openHID = nil // the reports cannot be opened (the app may not read them)
	f.m.cb.Added(d)
	if len(f.h.added) != 1 || f.h.added[0].Standard || f.h.added[0].Buttons != 2 {
		t.Fatalf("added = %+v", f.h.added)
	}
}

func TestServiceDisablesTheHIDDrivers(t *testing.T) {
	f := newFixture(t, true)
	f.start()
	opened := false
	d := dualShock4(nil)
	d.openHID = func() (gamepadhid.Transport, error) { opened = true; return newFakeHID(), nil }
	f.m.cb.Added(d)
	if opened || len(f.h.added) != 1 || f.h.added[0].Standard {
		t.Errorf("opened = %v, added = %+v", opened, f.h.added)
	}
}

func TestServiceFallsBackToElementsWhenTheDriverCannotStart(t *testing.T) {
	// A Switch Pro controller on USB that does not answer the handshake: the driver gives it up.
	f := newFixture(t, false)
	f.start()
	tr := newFakeHID()
	d := genericPad()
	d.info.Vendor, d.info.Product, d.info.ProductName = 0x057e, 0x2009, "Pro Controller"
	d.openHID = func() (gamepadhid.Transport, error) { return tr, nil }
	f.m.cb.Added(d)
	f.pump("the pad to be read through its elements", func() bool { return len(f.h.added) == 1 })
	if f.h.added[0].Standard || f.h.added[0].Buttons != 2 {
		t.Errorf("the pad is not read through its elements: %+v", f.h.added[0])
	}
	if !tr.isClosed() {
		t.Error("the transport of the driver that failed was left open")
	}
	// It works through its elements from now on.
	f.h.take()
	f.m.cb.Input(d, 10, 1)
	if got := f.h.take(); len(got) != 1 || got[0].Kind != platform.PadButton || got[0].Value != 1 {
		t.Errorf("events = %+v", got)
	}
}
