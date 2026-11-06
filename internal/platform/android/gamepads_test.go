package android

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// padRecorder is a platform.PadHandler.
type padRecorder struct {
	added  []platform.PadInfo
	gone   []platform.PadID
	events []platform.PadEvent
}

func (r *padRecorder) PadAdded(info platform.PadInfo)       { r.added = append(r.added, info) }
func (r *padRecorder) PadRemoved(id platform.PadID)         { r.gone = append(r.gone, id) }
func (r *padRecorder) PadEvents(events []platform.PadEvent) { r.events = append(r.events, events...) }

func (r *padRecorder) take() []platform.PadEvent {
	ev := r.events
	r.events = nil
	return ev
}

// xboxAxes are the axes of an Xbox pad on Android: the sticks on X, Y, Z and RZ, the triggers on LTRIGGER and RTRIGGER and a hat.
func xboxDescriptor(device int) padDescriptor {
	return padDescriptor{
		DeviceID: device, Name: "Xbox Wireless Controller", Descriptor: "d41d8cd9", Vendor: 0x045e, Product: 0x0b13, Flags: padFlagRumble,
		Axes:   []int{axisX, axisY, axisZ, axisRZ, axisLTrigger, axisRTrigger, axisHatX, axisHatY},
		Ranges: []float32{-1, 1, -1, 1, -1, 1, -1, 1, 0, 1, 0, 1, -1, 1, -1, 1},
	}
}

func motion(vals map[int]float32) []float32 {
	out := make([]float32, numPadAxes)
	for slot, v := range vals {
		out[slot] = v
	}
	return out
}

func TestPlanAxes(t *testing.T) {
	tests := []struct {
		name   string
		axes   []int
		ranges []float32
		want   axisPlan
	}{
		{"xbox", xboxDescriptor(1).Axes, xboxDescriptor(1).Ranges,
			axisPlan{leftX: slotX, leftY: slotY, rightX: slotZ, rightY: slotRZ, leftTrig: slotLTrigger, rightTrg: slotRTrigger, hat: true, trigMin: [2]float32{0, 0}, trigMax: [2]float32{1, 1}}},
		{"brake and gas", []int{axisX, axisY, axisZ, axisRZ, axisBrake, axisGas}, []float32{-1, 1, -1, 1, -1, 1, -1, 1, 0, 1, 0, 1},
			axisPlan{leftX: slotX, leftY: slotY, rightX: slotZ, rightY: slotRZ, leftTrig: slotBrake, rightTrg: slotGas, trigMin: [2]float32{0, 0}, trigMax: [2]float32{1, 1}}},
		{"dualshock 4: right stick on Z and RZ, triggers on RX and RY", []int{axisX, axisY, axisZ, axisRZ, axisRX, axisRY, axisHatX, axisHatY}, []float32{-1, 1, -1, 1, -1, 1, -1, 1, -1, 1, -1, 1, -1, 1, -1, 1},
			axisPlan{leftX: slotX, leftY: slotY, rightX: slotZ, rightY: slotRZ, leftTrig: slotRX, rightTrg: slotRY, hat: true, trigMin: [2]float32{-1, -1}, trigMax: [2]float32{1, 1}}},
		{"right stick on RX and RY", []int{axisX, axisY, axisRX, axisRY}, nil,
			axisPlan{leftX: slotX, leftY: slotY, rightX: slotRX, rightY: slotRY, leftTrig: -1, rightTrg: -1}},
		{"no axes", nil, nil, axisPlan{leftX: -1, leftY: -1, rightX: -1, rightY: -1, leftTrig: -1, rightTrg: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := planAxes(tt.axes, tt.ranges); got != tt.want {
				t.Errorf("planAxes() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPadKeysAndTriggerValues(t *testing.T) {
	tests := []struct {
		code int
		want gamepaddb.Button
		ok   bool
	}{
		{keycodeButtonA, gamepaddb.South, true}, {keycodeDpadCenter, gamepaddb.South, true}, {keycodeButtonB, gamepaddb.East, true},
		{keycodeButtonX, gamepaddb.West, true}, {keycodeButtonY, gamepaddb.North, true}, {keycodeButtonL1, gamepaddb.LeftShoulder, true},
		{keycodeButtonR1, gamepaddb.RightShoulder, true}, {keycodeButtonThumbL, gamepaddb.LeftStick, true}, {keycodeButtonThumbR, gamepaddb.RightStick, true},
		{keycodeButtonStart, gamepaddb.Start, true}, {keycodeButtonSelect, gamepaddb.Back, true}, {keycodeButtonMode, gamepaddb.Guide, true},
		{keycodeDpadUp, gamepaddb.DPadUp, true}, {keycodeDpadDown, gamepaddb.DPadDown, true}, {keycodeDpadLeft, gamepaddb.DPadLeft, true},
		{keycodeDpadRight, gamepaddb.DPadRight, true}, {keycodeMediaRecord, gamepaddb.Misc1, true},
		{keycodeBack, 0, false}, {keycodeButtonL2, 0, false}, {0, 0, false},
	}
	for _, tt := range tests {
		if got, ok := padButtonOfKey(tt.code); got != tt.want || ok != tt.ok {
			t.Errorf("padButtonOfKey(%d) = %d, %v, want %d, %v", tt.code, got, ok, tt.want, tt.ok)
		}
	}
	for _, code := range []int{keycodeButtonL2, keycodeButtonR2, keycodeButtonA} {
		if !IsPadKey(code) {
			t.Errorf("IsPadKey(%d) = false", code)
		}
	}
	if IsPadKey(keycodeBack) || IsPadKey(29) {
		t.Error("Back or a letter is a pad key")
	}
	for _, tt := range []struct {
		v, lo, hi float32
		want      int16
	}{{0, 0, 1, 0}, {1, 0, 1, 32767}, {-1, -1, 1, 0}, {1, -1, 1, 32767}, {0, -1, 1, 16383}, {5, 0, 0, 0}} {
		if got := triggerValue(tt.v, tt.lo, tt.hi); got != tt.want {
			t.Errorf("triggerValue(%v, %v, %v) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}

func TestPadServiceFollowsTheDevices(t *testing.T) {
	host := newFakeHost()
	s := newPadService(host)
	r := &padRecorder{}
	// A pad that came before the service started is reported by Start.
	s.added(xboxDescriptor(7))
	s.added(xboxDescriptor(7)) // the same device again changes nothing
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	if len(r.added) != 1 {
		t.Fatalf("added %+v", r.added)
	}
	info := r.added[0]
	if !info.Standard || !info.Rumble || info.Axes != int(gamepaddb.AxisCount) || info.Buttons != int(gamepaddb.ButtonCount) || info.Path != "d41d8cd9" || info.Battery != -1 {
		t.Errorf("info = %+v", info)
	}
	g := gamepaddb.GUID(info.GUID).Info()
	if g.Vendor != 0x045e || g.Product != 0x0b13 {
		t.Errorf("GUID = %+v", g)
	}
	if s.MappingPlatform() != "Android" {
		t.Errorf("MappingPlatform() = %q", s.MappingPlatform())
	}
	// A second pad while running is reported at once.
	d2 := xboxDescriptor(9)
	d2.Flags = 0
	s.added(d2)
	if len(r.added) != 2 || r.added[1].Rumble || r.added[1].ID == info.ID {
		t.Errorf("added %+v", r.added)
	}
	// A device that goes is removed; one that is not known is ignored.
	s.removed(9)
	s.removed(1234)
	if len(r.gone) != 1 || r.gone[0] != r.added[1].ID {
		t.Errorf("gone %v", r.gone)
	}
	// A sync with the list of devices drops the pads that are not in it.
	s.added(xboxDescriptor(11))
	s.sync([]int{11})
	if len(r.gone) != 2 || r.gone[1] != info.ID {
		t.Errorf("gone after the sync %v", r.gone)
	}
	// After Stop the pads are told again by the next Start.
	s.Stop()
	s.removed(11)
	if len(r.gone) != 2 {
		t.Errorf("a stopped service told of a removal: %v", r.gone)
	}
	s.added(xboxDescriptor(13))
	r2 := &padRecorder{}
	if err := s.Start(r2); err != nil {
		t.Fatal(err)
	}
	if len(r2.added) != 1 {
		t.Errorf("the second Start reported %+v", r2.added)
	}
}

func TestPadServiceKeysAndMotion(t *testing.T) {
	s := newPadService(newFakeHost())
	r := &padRecorder{}
	s.added(xboxDescriptor(3))
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	id := r.added[0].ID
	if s.key(99, true, keycodeButtonA, 10) {
		t.Error("a key of an unknown device was taken")
	}
	if s.key(3, true, keycodeBack, 10) {
		t.Error("the Back key was taken")
	}
	if !s.key(3, true, keycodeButtonA, 1000) {
		t.Fatal("A was not taken")
	}
	want := platform.PadEvent{Pad: id, Time: time.Second, Kind: platform.PadButton, Index: int(gamepaddb.South), Value: 1}
	if got := r.take(); len(got) != 1 || got[0] != want {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
	// A held button that repeats makes no new event.
	s.key(3, true, keycodeButtonA, 1100)
	if got := r.take(); len(got) != 0 {
		t.Errorf("a repeat gave %+v", got)
	}
	s.key(3, false, keycodeButtonA, 1200)
	if got := r.take(); len(got) != 1 || got[0].Value != 0 {
		t.Errorf("the release = %+v", got)
	}

	// The sticks, the triggers and the hat.
	s.motion(3, motion(map[int]float32{slotX: -1, slotY: 1, slotZ: 0.5, slotRZ: -0.5, slotRTrigger: 1, slotHatY: -1}), 2000)
	got := r.take()
	byKey := map[[2]int]int{}
	for _, e := range got {
		byKey[[2]int{int(e.Kind), e.Index}] = e.Value
	}
	check := func(kind platform.PadEventKind, index int, want int) {
		t.Helper()
		if v, ok := byKey[[2]int{int(kind), index}]; !ok || v != want {
			t.Errorf("kind %d index %d = %d (present %v), want %d; all %+v", kind, index, v, ok, want, got)
		}
	}
	check(platform.PadAxis, int(gamepaddb.LeftX), -32768)
	check(platform.PadAxis, int(gamepaddb.LeftY), 32767)
	check(platform.PadAxis, int(gamepaddb.RightX), 16383)
	check(platform.PadAxis, int(gamepaddb.RightY), -16384)
	check(platform.PadAxis, int(gamepaddb.RightTrigger), 32767)
	check(platform.PadButton, int(gamepaddb.DPadUp), 1)
	if len(got) != 6 {
		t.Errorf("%d events: %+v", len(got), got)
	}
	// The same motion again changes nothing; the hat released and moved right.
	s.motion(3, motion(map[int]float32{slotX: -1, slotY: 1, slotZ: 0.5, slotRZ: -0.5, slotRTrigger: 1, slotHatX: 1}), 2100)
	got = r.take()
	if len(got) != 2 {
		t.Errorf("events = %+v, want the hat up released and right pressed", got)
	}
	// A short array and an unknown device are ignored.
	s.motion(3, []float32{1}, 2200)
	s.motion(77, motion(nil), 2200)
	if got := r.take(); len(got) != 0 {
		t.Errorf("events = %+v", got)
	}
}

func TestDigitalTriggersOfAPadWithoutTriggerAxes(t *testing.T) {
	s := newPadService(newFakeHost())
	r := &padRecorder{}
	d := padDescriptor{DeviceID: 5, Name: "Pad", Descriptor: "p", Axes: []int{axisX, axisY}, Ranges: []float32{-1, 1, -1, 1}}
	s.added(d)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	s.key(5, true, keycodeButtonL2, 0)
	got := r.take()
	if len(got) != 1 || got[0].Kind != platform.PadAxis || got[0].Index != int(gamepaddb.LeftTrigger) || got[0].Value != 32767 {
		t.Fatalf("events = %+v", got)
	}
	s.key(5, false, keycodeButtonL2, 0)
	if got := r.take(); len(got) != 1 || got[0].Value != 0 {
		t.Errorf("release = %+v", got)
	}
	// A pad that has the trigger axes ignores the key: the axis gives the value.
	s2 := newPadService(newFakeHost())
	r2 := &padRecorder{}
	s2.added(xboxDescriptor(6))
	if err := s2.Start(r2); err != nil {
		t.Fatal(err)
	}
	if !s2.key(6, true, keycodeButtonR2, 0) {
		t.Error("the key was not taken")
	}
	if got := r2.take(); len(got) != 0 {
		t.Errorf("events = %+v", got)
	}
}

func TestPadServiceRumble(t *testing.T) {
	host := newFakeHost()
	s := newPadService(host)
	r := &padRecorder{}
	s.added(xboxDescriptor(4))
	noRumble := xboxDescriptor(8)
	noRumble.Flags = 0
	s.added(noRumble)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	with, without := r.added[0].ID, r.added[1].ID
	if err := s.Rumble(with, 0x8000, 0x4000, 1500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := s.Rumble(with, 1, 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Rumble(with, 0, 0, time.Second); err != nil {
		t.Fatal(err)
	}
	host.mu.Lock()
	cmds := append([]command(nil), host.commands...)
	host.mu.Unlock()
	want := []command{
		{Op: opPadRumble, A: 4, B: 0x8000, C: 0x4000, D: 1500},
		{Op: opPadRumble, A: 4, B: 1, C: 1, D: 0},
		{Op: opPadRumble, A: 4, B: 0, C: 0, D: 0},
	}
	if len(cmds) != len(want) {
		t.Fatalf("commands = %+v", cmds)
	}
	for i := range want {
		if cmds[i] != want[i] {
			t.Errorf("command %d = %+v, want %+v", i, cmds[i], want[i])
		}
	}
	for _, tt := range []struct {
		name  string
		err   error
		tag   string
		route string
	}{
		{"no vibrator", s.Rumble(without, 1, 1, time.Second), platform.TagGamepadRumbleUnavailable, "padService.Rumble.GamepadRumbleUnavailable"},
		{"unknown pad", s.Rumble(999, 1, 1, time.Second), platform.TagGamepadRumbleUnavailable, "padService.Rumble.GamepadRumbleUnavailable"},
		{"triggers", s.RumbleTriggers(with, 1, 1, time.Second), platform.TagGamepadRumbleUnavailable, "padService.RumbleTriggers.GamepadRumbleUnavailable"},
		{"light", s.SetLED(with, 1, 2, 3), platform.TagGamepadLEDUnavailable, "padService.SetLED.GamepadLEDUnavailable"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", tt.err, tt.err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

// TestPadEventsReachTheServiceThroughTheBridge: the events of the Java side are queued for the UI thread, where the service is.
func TestPadEventsReachTheServiceThroughTheBridge(t *testing.T) {
	b := newBridge()
	p := newPlatform(b, newFakeHost(), nil)
	r := &padRecorder{}
	if err := p.Services().Gamepads().Start(r); err != nil {
		t.Fatal(err)
	}
	b.padAdded(xboxDescriptor(2))
	b.padKey(2, true, keycodeButtonB, 500)
	b.padMotion(2, motion(map[int]float32{slotX: 1}), 600)
	b.padSync([]int{2})
	b.padRemoved(2)
	for fn, ok := b.pop(); ok; fn, ok = b.pop() {
		fn(p)
	}
	if len(r.added) != 1 || len(r.gone) != 1 || len(r.events) != 2 {
		t.Errorf("added %d gone %d events %+v", len(r.added), len(r.gone), r.events)
	}
}
