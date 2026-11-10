package pads

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
)

// rig is a manager on a scripted service whose posts run at once.
type rig struct {
	svc  *fake.Gamepads
	m    *Manager
	keys []Key
	post []func()
}

func newRig(t *testing.T, withKeys bool) *rig {
	t.Helper()
	r := &rig{}
	r.svc = fake.NewGamepads(func(fn func()) { r.post = append(r.post, fn) })
	opts := Opts{}
	if withKeys {
		opts.Keys = func(k Key) { r.keys = append(r.keys, k) }
	}
	r.m = New(r.svc, opts)
	if err := r.m.Start(); err != nil {
		t.Fatal(err)
	}
	r.run()
	return r
}

// run does what was posted.
func (r *rig) run() {
	for len(r.post) > 0 {
		fn := r.post[0]
		r.post = r.post[1:]
		fn()
	}
}

func (r *rig) connect(info platform.PadInfo) platform.PadID {
	id := r.svc.Connect(info)
	r.run()
	return id
}

func (r *rig) send(events ...platform.PadEvent) {
	r.svc.Send(events...)
	r.run()
}

func standard(name string) platform.PadInfo {
	return platform.PadInfo{Name: name, Standard: true, Battery: -1}
}

func TestStartIsIdempotentAndStopLetsGo(t *testing.T) {
	r := newRig(t, true)
	if err := r.m.Start(); err != nil {
		t.Fatal(err)
	}
	if r.svc.Starts() != 1 {
		t.Errorf("the service was started %d times, want 1", r.svc.Starts())
	}
	id := r.connect(standard("Pad"))
	r.send(platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: int(gamepaddb.South), Value: 1})
	pad := r.m.Pads()[0]
	calls := 0
	r.m.AddListener(func() { calls++ })
	r.keys = nil
	r.m.Stop()
	r.m.Stop()
	if len(r.m.Pads()) != 0 || calls != 1 || pad.Connected() || pad.State().Down(gamepaddb.South) {
		t.Errorf("after Stop: pads %d, calls %d, connected %v, South down %v", len(r.m.Pads()), calls, pad.Connected(), pad.State().Down(gamepaddb.South))
	}
	if len(r.keys) != 1 || r.keys[0].Down {
		t.Errorf("a button that is held is released as a key when the manager stops: %+v", r.keys)
	}
}

func TestAManagerWithoutAServiceHasNoPads(t *testing.T) {
	var m *Manager
	if m.Available() || len(m.Pads()) != 0 || m.KeyEvents() {
		t.Error("a nil manager has pads")
	}
	m = New(nil, Opts{})
	if m.Available() {
		t.Error("a manager without a service is available")
	}
	if err := m.Start(); err != nil {
		t.Errorf("Start() = %v", err)
	}
	m.Stop()
}

func TestKeyEvents(t *testing.T) {
	r := newRig(t, true)
	id := r.connect(standard("Pad"))
	btn := func(b gamepaddb.Button, down int) platform.PadEvent {
		return platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: int(b), Value: down, Time: time.Second}
	}
	r.send(btn(gamepaddb.South, 1), btn(gamepaddb.Misc1, 1), btn(gamepaddb.DPadLeft, 1), btn(gamepaddb.South, 0))
	want := []Key{
		{Time: time.Second, Down: true, Physical: 0x0005ff11, Logical: 0x200000311},
		{Time: time.Second, Down: true, Physical: 0x00070050, Logical: 0x100000302, DirectionalPad: true},
		{Time: time.Second, Down: false, Physical: 0x0005ff11, Logical: 0x200000311},
	}
	if !reflect.DeepEqual(r.keys, want) {
		t.Errorf("keys = %+v, want %+v", r.keys, want)
	}

	r.keys = nil
	r.m.SetKeyEvents(false)
	r.send(btn(gamepaddb.East, 1))
	if len(r.keys) != 0 || r.m.KeyEvents() {
		t.Errorf("key events are off but %+v arrived", r.keys)
	}
	r.m.SetKeyEvents(true)
	r.send(btn(gamepaddb.East, 0))
	if len(r.keys) != 1 || !r.m.KeyEvents() {
		t.Errorf("key events are on again: %+v", r.keys)
	}

	// A manager that was given no function for keys cannot be made to produce them.
	r2 := newRig(t, false)
	r2.m.SetKeyEvents(true)
	if r2.m.KeyEvents() {
		t.Error("key events are on with nowhere to send them")
	}
}

func TestTriggersAreKeysAtHalfTravelWithoutRepeats(t *testing.T) {
	r := newRig(t, true)
	id := r.connect(standard("Pad"))
	axis := func(a gamepaddb.Axis, v int) platform.PadEvent {
		return platform.PadEvent{Pad: id, Kind: platform.PadAxis, Index: int(a), Value: v}
	}
	r.send(axis(gamepaddb.LeftTrigger, 16383), axis(gamepaddb.LeftTrigger, 16384), axis(gamepaddb.LeftTrigger, 32767),
		axis(gamepaddb.RightTrigger, 40000), axis(gamepaddb.LeftTrigger, 100), axis(gamepaddb.LeftX, 32767))
	var got []bool
	var logical []uint64
	for _, k := range r.keys {
		got = append(got, k.Down)
		logical = append(logical, k.Logical)
	}
	if !reflect.DeepEqual(got, []bool{true, true, false}) || !reflect.DeepEqual(logical, []uint64{0x200000315, 0x200000318, 0x200000315}) {
		t.Errorf("trigger keys = %v %x", got, logical)
	}
	if v := r.m.Pads()[0].State().Axes[gamepaddb.RightTrigger]; v != gamepaddb.AxisMax {
		t.Errorf("a value over the range is clamped: %d", v)
	}
}

func TestStandardPadsKnowTheirButtonsAndAxes(t *testing.T) {
	r := newRig(t, false)
	r.connect(standard("Browser Pad"))
	p := r.m.Pads()[0]
	for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
		if !p.HasButton(b) {
			t.Errorf("a standard pad has no %s", b.Name())
		}
	}
	for a := gamepaddb.Axis(0); a < gamepaddb.AxisCount; a++ {
		if !p.HasAxis(a) {
			t.Errorf("a standard pad has no %s", a.Name())
		}
	}
	if p.Mapping() != "" || p.Name() != "Browser Pad" || p.Kind() != gamepaddb.KindStandard || p.HasButton(gamepaddb.ButtonCount) || p.HasAxis(gamepaddb.AxisCount) {
		t.Errorf("mapping %q, name %q, kind %v", p.Mapping(), p.Name(), p.Kind())
	}
}

func TestUnmappedPadsWaitForAMapping(t *testing.T) {
	r := newRig(t, false)
	guid := gamepaddb.ParseGUID("03000000aaaa00000001000001000000")
	a := r.connect(platform.PadInfo{Name: "Mystery A", GUID: guid, Axes: 1, Buttons: 2, Battery: -1})
	b := r.connect(platform.PadInfo{Name: "Mystery B", GUID: gamepaddb.ParseGUID("03000000bbbb00000001000001000000"), Axes: 1, Buttons: 2, Battery: -1})
	c := r.connect(standard("Known"))
	if got := len(r.m.Pads()); got != 1 || r.m.Pads()[0].ID() != c {
		t.Fatalf("pads = %d, want only the standard one", got)
	}
	// Mystery B goes before it ever had a mapping.
	r.svc.Disconnect(b)
	r.run()
	// A mapping arrives for A: it joins, and is listed before the pad that was connected after it.
	n, err := r.m.AddMapping("03000000aaaa00000001000001000000,Mystery,a:b0,b:b1,leftx:a0")
	if err != nil || n != 1 {
		t.Fatalf("AddMapping() = %d, %v", n, err)
	}
	pads := r.m.Pads()
	if len(pads) != 2 || pads[0].ID() != a || pads[1].ID() != c {
		t.Fatalf("pads = %v, want the mystery pad before the known one", ids(pads))
	}
	if pads[0].Name() != "Mystery" || pads[0].Mapping() != "a:b0,b:b1,leftx:a0" {
		t.Errorf("name %q, mapping %q", pads[0].Name(), pads[0].Mapping())
	}
	// A pad that is removed while it has no mapping leaves nothing behind.
	r.svc.Disconnect(a)
	r.run()
	if _, ok := r.m.unmapped[a]; ok || len(r.m.Pads()) != 1 {
		t.Errorf("unmapped %v, pads %d", r.m.unmapped, len(r.m.Pads()))
	}
}

func ids(pads []*Pad) []platform.PadID {
	var out []platform.PadID
	for _, p := range pads {
		out = append(out, p.ID())
	}
	return out
}

func TestAMappingThatChangesLetsGoOfTheButtonsItHeld(t *testing.T) {
	r := newRig(t, true)
	guid := "03000000cccc00000001000001000000"
	id := r.connect(platform.PadInfo{Name: "Pad", GUID: gamepaddb.ParseGUID(guid), Buttons: 4, Battery: -1})
	if _, err := r.m.AddMapping(guid + ",Pad,a:b0,b:b1"); err != nil {
		t.Fatal(err)
	}
	r.send(platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: 0, Value: 1})
	pad := r.m.Pads()[0]
	if !pad.State().Down(gamepaddb.South) {
		t.Fatal("South is not down")
	}
	r.keys = nil
	if _, err := r.m.AddMapping(guid + ",Pad,a:b2,b:b3"); err != nil {
		t.Fatal(err)
	}
	if pad.State().Down(gamepaddb.South) || len(r.keys) != 1 || r.keys[0].Down {
		t.Errorf("South is down %v, keys %+v: the held button must be let go", pad.State().Down(gamepaddb.South), r.keys)
	}
	r.send(platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: 2, Value: 1})
	if !pad.State().Down(gamepaddb.South) {
		t.Error("the new mapping does not read raw button 2 as South")
	}
}

func TestADuplicateAndAnUnknownPadAreIgnored(t *testing.T) {
	r := newRig(t, false)
	id := r.connect(standard("Pad"))
	r.m.PadAdded(platform.PadInfo{ID: id, Name: "Again", Standard: true})
	r.m.PadRemoved(999)
	r.m.PadEvents([]platform.PadEvent{{Pad: 999, Kind: platform.PadButton}})
	if len(r.m.Pads()) != 1 || r.m.Pads()[0].Name() != "Pad" {
		t.Errorf("pads = %v", ids(r.m.Pads()))
	}
}

func TestMappingsWithAPlatformFieldForAnotherPlatformAreLeftOut(t *testing.T) {
	r := newRig(t, false)
	n, err := r.m.AddMappings("03000000dddd00000001000001000000,W,a:b0,platform:Windows,\n03000000dddd00000002000001000000,L,a:b0,platform:Linux,\n")
	if err != nil || n != 1 {
		t.Fatalf("AddMappings() = %d, %v, want 1", n, err)
	}
	if r.m.Table().Find(gamepaddb.ParseGUID("03000000dddd00000001000001000000")) != nil {
		t.Error("the mapping for Windows was added")
	}
}

func TestRumbleErrorsAreTheServices(t *testing.T) {
	r := newRig(t, false)
	r.connect(standard("Pad"))
	err := r.m.Pads()[0].Rumble(1, 1, time.Second)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagGamepadRumbleUnavailable || !appErr.HasRoute("Pad.Rumble->Gamepads.Rumble.GamepadRumbleUnavailable") {
		t.Errorf("unexpected failure: %v", err)
	}
}

func TestTheTableDependsOnThePlatform(t *testing.T) {
	for platformName, wantLinux := range map[string]bool{"Linux": true, "Windows": false, "Android": false, "Web": false, "": false} {
		m := New(namedService{platformName}, Opts{})
		got := m.Table().Find(gamepaddb.ParseGUID("03000000ffff0000ffff000000010000")) != nil
		if got != wantLinux {
			t.Errorf("platform %q: the Linux PS5 mapping is there: %v, want %v", platformName, got, wantLinux)
		}
	}
	if New(namedService{"Windows"}, Opts{}).Table().Len() == 0 || New(namedService{"Web"}, Opts{}).Table().Default() == nil {
		t.Error("the tables of Windows or the web are empty")
	}
}

// namedService is a gamepad service that only has a platform name.
type namedService struct{ name string }

func (n namedService) MappingPlatform() string       { return n.name }
func (namedService) Start(platform.PadHandler) error { return nil }
func (namedService) Stop()                           {}
func (namedService) Rumble(platform.PadID, uint16, uint16, time.Duration) error {
	return nil
}
func (namedService) RumbleTriggers(platform.PadID, uint16, uint16, time.Duration) error { return nil }
func (namedService) SetLED(platform.PadID, uint8, uint8, uint8) error                   { return nil }

func TestWake(t *testing.T) {
	t.Run("before Start, the service starts awake", func(t *testing.T) {
		svc := fake.NewGamepads(func(fn func()) { fn() })
		m := New(svc, Opts{})
		if err := m.Wake(); err != nil {
			t.Fatal(err)
		}
		if svc.Starts() != 1 || svc.Extras() != 1 || svc.ExtrasAtStart() != 1 {
			t.Errorf("starts %d, wake-ups %d, wake-ups at the start %d; want 1, 1, 1", svc.Starts(), svc.Extras(), svc.ExtrasAtStart())
		}
	})
	t.Run("after Start, the service that was asleep wakes once", func(t *testing.T) {
		r := newRig(t, false)
		if r.svc.Extras() != 0 {
			t.Fatalf("Start woke the extras: %d", r.svc.Extras())
		}
		_ = r.m.Wake()
		_ = r.m.Wake()
		if r.svc.Starts() != 1 || r.svc.Extras() != 1 || r.svc.ExtrasAtStart() != 0 {
			t.Errorf("starts %d, wake-ups %d, at the start %d; want 1, 1, 0", r.svc.Starts(), r.svc.Extras(), r.svc.ExtrasAtStart())
		}
	})
	t.Run("a manager that is nil or has no service", func(t *testing.T) {
		var nilM *Manager
		if err := nilM.Wake(); err != nil {
			t.Error(err)
		}
		if err := New(nil, Opts{}).Wake(); err != nil {
			t.Error(err)
		}
	})
}
