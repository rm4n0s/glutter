package winpad

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestXInputSnapshot(t *testing.T) {
	g := win32.XInputGamepad{
		Buttons:      win32.XInputGamepadA | win32.XInputGamepadLeftShoulder | win32.XInputGamepadDPadLeft | win32.XInputGamepadGuide | win32.XInputGamepadStart,
		LeftTrigger:  255,
		RightTrigger: 0,
		ThumbLX:      -32768,
		ThumbLY:      32767, // up on XInput
		ThumbRX:      100,
		ThumbRY:      -32768, // down on XInput
	}
	s := xinputSnapshot(g)
	down := map[gamepaddb.Button]bool{gamepaddb.South: true, gamepaddb.LeftShoulder: true, gamepaddb.DPadLeft: true, gamepaddb.Guide: true, gamepaddb.Start: true}
	for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
		if s.Buttons[b] != down[b] {
			t.Errorf("button %d = %v, want %v", b, s.Buttons[b], down[b])
		}
	}
	wantAxes := map[gamepaddb.Axis]int16{
		gamepaddb.LeftX: -32768, gamepaddb.LeftY: -32767, gamepaddb.RightX: 100, gamepaddb.RightY: 32767,
		gamepaddb.LeftTrigger: 32767, gamepaddb.RightTrigger: 0,
	}
	for a, want := range wantAxes {
		if s.Axes[a] != want {
			t.Errorf("axis %d = %d, want %d", a, s.Axes[a], want)
		}
	}
}

func TestXInputNamesAndBattery(t *testing.T) {
	names := []struct {
		user uint32
		sub  uint8
		want string
	}{
		{0, subTypeGamepad, "XInput Controller #1"}, {3, subTypeWheel, "XInput Wheel #4"}, {1, subTypeGuitarBass, "XInput Guitar #2"},
		{2, subTypeArcadePad, "XInput ArcadePad #3"}, {0, 0x42, "XInput Device #1"}, {0, subTypeDrumKit, "XInput DrumKit #1"},
	}
	for _, tt := range names {
		if got := xinputName(tt.user, tt.sub); got != tt.want {
			t.Errorf("xinputName(%d, %#x) = %q, want %q", tt.user, tt.sub, got, tt.want)
		}
	}
	batteries := []struct {
		b    win32.XInputBatteryInformation
		ok   bool
		want int
	}{
		{win32.XInputBatteryInformation{BatteryType: win32.XInputBatteryTypeWired, BatteryLevel: win32.XInputBatteryLevelFull}, true, 100},
		{win32.XInputBatteryInformation{BatteryType: 2, BatteryLevel: win32.XInputBatteryLevelEmpty}, true, 10},
		{win32.XInputBatteryInformation{BatteryType: 2, BatteryLevel: win32.XInputBatteryLevelLow}, true, 40},
		{win32.XInputBatteryInformation{BatteryType: 2, BatteryLevel: win32.XInputBatteryLevelMedium}, true, 70},
		{win32.XInputBatteryInformation{BatteryType: win32.XInputBatteryTypeUnknown}, true, -1},
		{win32.XInputBatteryInformation{BatteryType: win32.XInputBatteryTypeDisconnected}, true, -1},
		{win32.XInputBatteryInformation{BatteryType: 2, BatteryLevel: 3}, false, -1},
	}
	for _, tt := range batteries {
		if got := xinputBattery(tt.b, tt.ok); got != tt.want {
			t.Errorf("xinputBattery(%+v, %v) = %d, want %d", tt.b, tt.ok, got, tt.want)
		}
	}
}

// fakeXInput is XInput with scripted slots.
type fakeXInput struct {
	mu       sync.Mutex
	pads     map[uint32]*fakeSlot
	states   int // calls of GetState
	vibrated []win32.XInputVibration
	setFails bool
	noEx     bool
}

type fakeSlot struct {
	caps   win32.XInputCapabilities
	state  win32.XInputState
	ex     win32.XInputCapabilitiesEx
	bat    win32.XInputBatteryInformation
	hasBat bool
}

func (f *fakeXInput) plug(user uint32, s *fakeSlot) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pads == nil {
		f.pads = map[uint32]*fakeSlot{}
	}
	f.pads[user] = s
}

func (f *fakeXInput) unplug(user uint32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.pads, user)
}

func (f *fakeXInput) press(user uint32, buttons uint16) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pads[user].state.Gamepad.Buttons = buttons
	f.pads[user].state.PacketNumber++
}

func (f *fakeXInput) getStateCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.states
}

func (f *fakeXInput) motors() []win32.XInputVibration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]win32.XInputVibration(nil), f.vibrated...)
}

func (f *fakeXInput) GetState(user uint32, st *win32.XInputState) uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states++
	s := f.pads[user]
	if s == nil {
		return win32.ErrorDeviceNotConnected
	}
	*st = s.state
	return win32.ErrorSuccessCode
}

func (f *fakeXInput) SetState(user uint32, v *win32.XInputVibration) uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setFails || f.pads[user] == nil {
		return win32.ErrorDeviceNotConnected
	}
	f.vibrated = append(f.vibrated, *v)
	return win32.ErrorSuccessCode
}

func (f *fakeXInput) GetCapabilities(user, flags uint32, c *win32.XInputCapabilities) uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.pads[user]
	if s == nil {
		return win32.ErrorDeviceNotConnected
	}
	*c = s.caps
	return win32.ErrorSuccessCode
}

func (f *fakeXInput) GetCapabilitiesEx(user uint32) (win32.XInputCapabilitiesEx, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.pads[user]
	if s == nil || f.noEx {
		return win32.XInputCapabilitiesEx{}, false
	}
	return s.ex, true
}

func (f *fakeXInput) GetBatteryInformation(user uint32) (win32.XInputBatteryInformation, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.pads[user]
	if s == nil {
		return win32.XInputBatteryInformation{}, false
	}
	return s.bat, s.hasBat
}

func xboxSlot() *fakeSlot {
	return &fakeSlot{
		caps:   win32.XInputCapabilities{Type: win32.XInputDevTypeGamepad, SubType: win32.XInputDevSubTypeGamepad},
		ex:     win32.XInputCapabilitiesEx{VendorID: 0x045e, ProductID: 0x0b12, VersionNumber: 0x0509},
		bat:    win32.XInputBatteryInformation{BatteryType: 2, BatteryLevel: win32.XInputBatteryLevelMedium},
		hasBat: true,
	}
}

// recorder is a platform.PadHandler that the test reads, fed through Post.
type recorder struct {
	mu     sync.Mutex
	added  []platform.PadInfo
	gone   []platform.PadID
	events []platform.PadEvent
	posts  chan func()
}

func newRecorder() *recorder { return &recorder{posts: make(chan func(), 1024)} }

func (r *recorder) post(fn func()) { r.posts <- fn }

func (r *recorder) PadAdded(info platform.PadInfo) {
	r.mu.Lock()
	r.added = append(r.added, info)
	r.mu.Unlock()
}

func (r *recorder) PadRemoved(id platform.PadID) {
	r.mu.Lock()
	r.gone = append(r.gone, id)
	r.mu.Unlock()
}

func (r *recorder) PadEvents(events []platform.PadEvent) {
	r.mu.Lock()
	r.events = append(r.events, events...)
	r.mu.Unlock()
}

// wait runs the posted functions until cond holds.
func (r *recorder) wait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		r.mu.Lock()
		ok := cond()
		r.mu.Unlock()
		if ok {
			return
		}
		select {
		case fn := <-r.posts:
			fn()
		case <-deadline:
			t.Fatalf("timed out waiting for %s (added %d, gone %d, events %d)", what, len(r.added), len(r.gone), len(r.events))
		}
	}
}

// drain runs what is posted already.
func (r *recorder) drain() {
	for {
		select {
		case fn := <-r.posts:
			fn()
		default:
			return
		}
	}
}

func (r *recorder) takeEvents() []platform.PadEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev := r.events
	r.events = nil
	return ev
}

func newService(r *recorder, x XInputAPI, h HIDSource) *Service {
	return New(Opts{Post: r.post, Clock: func() time.Duration { return 42 * time.Second }, XInput: x, HID: h, PollEvery: 2 * time.Millisecond, ScanEvery: 20 * time.Millisecond})
}

func TestXInputPadLifecycle(t *testing.T) {
	x := &fakeXInput{}
	slot := xboxSlot()
	slot.state.Gamepad.Buttons = win32.XInputGamepadB
	x.plug(1, slot)
	r := newRecorder()
	s := newService(r, x, nil)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	r.wait(t, "the pad and its state", func() bool { return len(r.added) == 1 && len(r.events) == 1 })
	info := r.added[0]
	if info.Name != "Xbox Series Controller" && !strings.Contains(info.Name, "Xbox") {
		t.Errorf("Name = %q", info.Name)
	}
	if !info.Standard || !info.Rumble || info.Path != "XInput#1" || info.Battery != 70 || info.Axes != int(gamepaddb.AxisCount) || info.Buttons != int(gamepaddb.ButtonCount) {
		t.Errorf("info = %+v", info)
	}
	g := gamepaddb.GUID(info.GUID).Info()
	if g.Vendor != 0x045e || g.Product != 0x0b12 || g.Version != 0x0509 {
		t.Errorf("GUID says %+v", g)
	}
	first := r.events[0]
	if first.Kind != platform.PadButton || first.Index != int(gamepaddb.East) || first.Value != 1 || first.Time != 42*time.Second {
		t.Errorf("the state at the start = %+v", first)
	}
	r.takeEvents()

	x.press(1, win32.XInputGamepadB|win32.XInputGamepadX)
	r.wait(t, "the new button", func() bool { return len(r.events) == 1 })
	if e := r.takeEvents()[0]; e.Index != int(gamepaddb.West) || e.Value != 1 {
		t.Errorf("event = %+v", e)
	}
	// The same packet is not read again.
	time.Sleep(20 * time.Millisecond)
	r.drain()
	if got := r.takeEvents(); len(got) != 0 {
		t.Errorf("a state that did not change gave %+v", got)
	}

	x.unplug(1)
	r.wait(t, "the removal", func() bool { return len(r.gone) == 1 })
	if r.gone[0] != info.ID {
		t.Errorf("removed %d, added %d", r.gone[0], info.ID)
	}
	// Plugged in again, it is a new pad.
	x.plug(2, xboxSlot())
	r.wait(t, "the pad again", func() bool { return len(r.added) == 2 })
	if r.added[1].ID == info.ID || r.added[1].Path != "XInput#2" {
		t.Errorf("the second pad = %+v", r.added[1])
	}
}

func TestXInputIdentityWithoutTheCapabilitiesEx(t *testing.T) {
	x := &fakeXInput{noEx: true}
	x.plug(0, xboxSlot())
	r := newRecorder()
	s := newService(r, x, nil)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	if g := gamepaddb.GUID(r.added[0].GUID).Info(); g.Vendor != 0x045e || g.Product != 0x028e {
		t.Errorf("GUID says %+v, want the Xbox 360 controller", g)
	}
	if r.added[0].Name != "Xbox 360 Controller" {
		t.Errorf("Name = %q", r.added[0].Name)
	}
}

func TestXInputWirelessPadWithoutProduct(t *testing.T) {
	x := &fakeXInput{}
	slot := xboxSlot()
	slot.ex = win32.XInputCapabilitiesEx{}
	slot.caps.Flags = win32.XInputCapsWireless
	x.plug(0, slot)
	vendor, product, _ := xinputIdentity(x, 0, &slot.caps)
	if vendor != 0x045e || product != 0x028e {
		t.Errorf("identity = %#x %#x", vendor, product)
	}
	slot.caps.Flags = 0
	if vendor, product, _ = xinputIdentity(x, 0, &slot.caps); vendor != 0 || product != 0 {
		t.Errorf("a wired pad with no ids got %#x %#x", vendor, product)
	}
}

func TestEmptySlotsAreNotPolled(t *testing.T) {
	x := &fakeXInput{}
	r := newRecorder()
	s := newService(r, x, nil)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	time.Sleep(100 * time.Millisecond)
	// With no pad only the scan (every 20 ms here) reads the slots, not the poll (every 2 ms): GetState is not called for a slot
	// that XInput says has no pad (GetCapabilities fails first).
	if n := x.getStateCalls(); n != 0 {
		t.Errorf("GetState was called %d times with no pad", n)
	}
}

func TestRescanFindsAPadAtOnce(t *testing.T) {
	x := &fakeXInput{}
	r := newRecorder()
	s := New(Opts{Post: r.post, XInput: x, PollEvery: time.Hour, ScanEvery: time.Hour})
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	time.Sleep(20 * time.Millisecond)
	x.plug(0, xboxSlot())
	s.Rescan()
	r.wait(t, "the pad after the rescan", func() bool { return len(r.added) == 1 })
}

func TestXInputRumble(t *testing.T) {
	x := &fakeXInput{}
	x.plug(0, xboxSlot())
	r := newRecorder()
	s := newService(r, x, nil)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	id := r.added[0].ID

	if err := s.Rumble(id, 1000, 2000, 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	got := x.motors()
	if len(got) != 1 || got[0] != (win32.XInputVibration{LeftMotorSpeed: 1000, RightMotorSpeed: 2000}) {
		t.Fatalf("motors = %+v", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(x.motors()) < 2 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if got = x.motors(); len(got) != 2 || got[1] != (win32.XInputVibration{}) {
		t.Errorf("the motors did not stop after the duration: %+v", got)
	}
	// A new rumble replaces the old one and its timer; 0 stops it.
	if err := s.Rumble(id, 5, 6, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := s.Rumble(id, 5, 6, 0); err != nil {
		t.Fatal(err)
	}
	got = x.motors()
	if last := got[len(got)-1]; last != (win32.XInputVibration{}) {
		t.Errorf("the last setting = %+v, want the motors off", last)
	}
}

func TestRumbleErrors(t *testing.T) {
	x := &fakeXInput{}
	x.plug(0, xboxSlot())
	r := newRecorder()
	s := newService(r, x, nil)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	id := r.added[0].ID

	tests := []struct {
		name  string
		call  func() error
		tag   string
		route string
	}{
		{"unknown pad", func() error { return s.Rumble(999, 1, 1, time.Second) }, platform.TagGamepadRumbleUnavailable, "Service.Rumble.GamepadRumbleUnavailable"},
		{"triggers", func() error { return s.RumbleTriggers(id, 1, 1, time.Second) }, platform.TagGamepadRumbleUnavailable, "Service.RumbleTriggers.GamepadRumbleUnavailable"},
		{"light", func() error { return s.SetLED(id, 1, 2, 3) }, platform.TagGamepadLEDUnavailable, "Service.SetLED.GamepadLEDUnavailable"},
		{"xinput refuses", func() error {
			x.mu.Lock()
			x.setFails = true
			x.mu.Unlock()
			return s.Rumble(id, 1, 1, time.Second)
		}, tagXInputRumbleFailed, "Service.Rumble.XInputRumbleFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
	if s.MappingPlatform() != "Windows" {
		t.Errorf("MappingPlatform() = %q", s.MappingPlatform())
	}
}

func TestStopAndStartAgain(t *testing.T) {
	x := &fakeXInput{}
	x.plug(0, xboxSlot())
	r := newRecorder()
	s := newService(r, x, nil)
	for round := 1; round <= 2; round++ {
		if err := s.Start(r); err != nil {
			t.Fatal(err)
		}
		if err := s.Start(r); err != nil { // a second Start does nothing
			t.Fatal(err)
		}
		r.wait(t, fmt.Sprintf("the pad of round %d", round), func() bool { return len(r.added) == round })
		s.Stop()
		s.Stop()
	}
	if err := s.Rumble(r.added[0].ID, 1, 1, time.Second); err == nil {
		t.Error("a pad of a stopped run rumbled")
	}
	s.Rescan()                        // nothing runs: it does nothing
	s.HIDReports(1, [][]byte{{1}}, 0) // the same
}
