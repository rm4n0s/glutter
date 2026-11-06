package web

import (
	"encoding/binary"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

func TestParsePadID(t *testing.T) {
	tests := []struct {
		id      string
		name    string
		vendor  uint16
		product uint16
		xinput  bool
	}{
		{"Wireless Controller (STANDARD GAMEPAD Vendor: 054c Product: 09cc)", "Wireless Controller", 0x054c, 0x09cc, false},
		{"usb gamepad           (Vendor: 0810 Product: e501)", "usb gamepad", 0x0810, 0xe501, false},
		{"Xbox 360 Controller (XInput STANDARD GAMEPAD)", "Xbox 360 Controller", 0, 0, true},
		{"046d-c216-Logitech Dual Action", "Logitech Dual Action", 0x046d, 0xc216, false},
		{"46d-c216-Logicool Dual Action", "Logicool Dual Action", 0x046d, 0xc216, false},
		{"xinput", "xinput", 0, 0, true},
		{"Some Pad", "Some Pad", 0, 0, false},
		{"054c-09cc-Pad-With-Dashes", "Pad-With-Dashes", 0x054c, 0x09cc, false},
		{"not-hex-name", "not-hex-name", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			name, vendor, product, xinput := ParsePadID(tt.id)
			if name != tt.name || vendor != tt.vendor || product != tt.product || xinput != tt.xinput {
				t.Errorf("ParsePadID(%q) = %q %#x %#x %v, want %q %#x %#x %v", tt.id, name, vendor, product, xinput, tt.name, tt.vendor, tt.product, tt.xinput)
			}
		})
	}
}

func TestParseHexPrefix(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want uint64
		ok   bool
	}{{"054c", 0x54c, true}, {" 0x1F rest", 0x1f, true}, {"xinput", 0, false}, {"", 0, false}, {"zz", 0, false}, {"12345678abc", 0x12345678, true}} {
		if got, ok := parseHexPrefix(tt.in); got != tt.want || ok != tt.ok {
			t.Errorf("parseHexPrefix(%q) = %#x, %v, want %#x, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestHatOfAxis(t *testing.T) {
	const up, right, down, left = gamepaddb.HatUp, gamepaddb.HatRight, gamepaddb.HatDown, gamepaddb.HatLeft
	for _, tt := range []struct {
		v    float64
		want int
	}{
		{-1, up}, {-5.0 / 7, up | right}, {-3.0 / 7, right}, {-1.0 / 7, down | right}, {1.0 / 7, down}, {3.0 / 7, down | left},
		{5.0 / 7, left}, {1, up | left}, {1.2857142857142856, 0}, {0, 0}, {0.5, 0},
	} {
		if got := hatOfAxis(tt.v); got != tt.want {
			t.Errorf("hatOfAxis(%v) = %d, want %d", tt.v, got, tt.want)
		}
	}
}

func TestPadOSNumber(t *testing.T) {
	for os, want := range map[OperatingSystem]uint8{OSUnknown: 0, OSAndroid: 1, OSLinux: 2, OSIOS: 3, OSMacOS: 4, OSWindows: 5} {
		if got := PadOSNumber(os); got != want {
			t.Errorf("PadOSNumber(%d) = %d, want %d", os, got, want)
		}
	}
}

// recorder is a platform.PadHandler.
type recorder struct {
	added  []platform.PadInfo
	gone   []platform.PadID
	events []platform.PadEvent
}

func (r *recorder) PadAdded(info platform.PadInfo)       { r.added = append(r.added, info) }
func (r *recorder) PadRemoved(id platform.PadID)         { r.gone = append(r.gone, id) }
func (r *recorder) PadEvents(events []platform.PadEvent) { r.events = append(r.events, events...) }

func standardPad(index int) BrowserPad {
	p := BrowserPad{Index: index, ID: "Xbox 360 Controller (XInput STANDARD GAMEPAD)", Mapping: "standard", Connected: true, Timestamp: 1, DualRumble: true}
	p.Buttons = make([]BrowserButton, 17)
	p.Axes = make([]float64, 4)
	return p
}

func TestStandardPad(t *testing.T) {
	var ids atomic.Int64
	pp := NewPadPoller(OSLinux, &ids)
	r := &recorder{}
	p := standardPad(0)
	p.Buttons[0].Pressed = true
	p.Buttons[7] = BrowserButton{Pressed: true, Value: 0.5}
	p.Buttons[12].Pressed = true
	p.Axes[1] = -1
	pp.Update(time.Second, []BrowserPad{p}, r)
	if len(r.added) != 1 {
		t.Fatalf("added %+v", r.added)
	}
	info := r.added[0]
	if !info.Standard || info.Axes != int(gamepaddb.AxisCount) || info.Buttons != int(gamepaddb.ButtonCount) || !info.Rumble || info.RumbleTriggers || info.Battery != -1 {
		t.Errorf("info = %+v", info)
	}
	g := gamepaddb.GUID(info.GUID).Info()
	if g.Vendor != 0x045e || g.Product != 0x028e {
		t.Errorf("an XInput pad with no ids got GUID %+v", g)
	}
	if info.Name != "Xbox 360 Controller" {
		t.Errorf("Name = %q", info.Name)
	}
	want := map[platform.PadEvent]bool{
		{Pad: info.ID, Time: time.Second, Kind: platform.PadButton, Index: int(gamepaddb.South), Value: 1}:          true,
		{Pad: info.ID, Time: time.Second, Kind: platform.PadButton, Index: int(gamepaddb.DPadUp), Value: 1}:         true,
		{Pad: info.ID, Time: time.Second, Kind: platform.PadAxis, Index: int(gamepaddb.LeftY), Value: -32768}:       true,
		{Pad: info.ID, Time: time.Second, Kind: platform.PadAxis, Index: int(gamepaddb.RightTrigger), Value: 16383}: true,
	}
	if len(r.events) != len(want) {
		t.Fatalf("events = %+v", r.events)
	}
	for _, e := range r.events {
		if !want[e] {
			t.Errorf("unexpected event %+v", e)
		}
	}

	// The same timestamp: nothing is looked at, whatever the state says.
	r.events = nil
	p.Buttons[1].Pressed = true
	pp.Update(2*time.Second, []BrowserPad{p}, r)
	if len(r.events) != 0 {
		t.Errorf("a reading with the old timestamp gave %+v", r.events)
	}
	p.Timestamp = 2
	pp.Update(3*time.Second, []BrowserPad{p}, r)
	if len(r.events) != 1 || r.events[0].Index != int(gamepaddb.East) || r.events[0].Value != 1 {
		t.Errorf("events = %+v", r.events)
	}
	if pp.Count() != 1 || pp.IDOf(0) != info.ID || pp.IDOf(3) != 0 {
		t.Errorf("Count %d IDOf(0) %d", pp.Count(), pp.IDOf(0))
	}
	if idx, ok := pp.IndexOf(info.ID); !ok || idx != 0 {
		t.Errorf("IndexOf = %d, %v", idx, ok)
	}
	if _, ok := pp.IndexOf(999); ok {
		t.Error("IndexOf found a pad that is not there")
	}

	// A disconnected pad, or one that left the array, is removed.
	p.Connected = false
	pp.Update(4*time.Second, []BrowserPad{p}, r)
	if len(r.gone) != 1 || r.gone[0] != info.ID || pp.Count() != 0 {
		t.Errorf("gone %v, count %d", r.gone, pp.Count())
	}
	// Connected again, it is a new pad.
	p.Connected = true
	pp.Update(5*time.Second, []BrowserPad{p}, r)
	if len(r.added) != 2 || r.added[1].ID == info.ID {
		t.Errorf("added %+v", r.added)
	}
	pp.Update(6*time.Second, nil, r)
	if len(r.gone) != 2 {
		t.Errorf("a reading with no pads left %d gone", len(r.gone))
	}
}

func TestPadWithAnotherIDTakesTheIndex(t *testing.T) {
	var ids atomic.Int64
	pp := NewPadPoller(OSLinux, &ids)
	r := &recorder{}
	a := standardPad(1)
	pp.Update(0, []BrowserPad{a}, r)
	b := standardPad(1)
	b.ID = "Wireless Controller (STANDARD GAMEPAD Vendor: 054c Product: 09cc)"
	pp.Update(0, []BrowserPad{b}, r)
	if len(r.added) != 2 || len(r.gone) != 1 || r.added[1].Name != "Wireless Controller" {
		t.Errorf("added %+v gone %v", r.added, r.gone)
	}
}

func TestPadWithoutStandardMapping(t *testing.T) {
	var ids atomic.Int64
	pp := NewPadPoller(OSWindows, &ids)
	r := &recorder{}
	p := BrowserPad{Index: 2, ID: "046d-c216-Logitech Dual Action", Connected: true, Timestamp: 1}
	p.Buttons = []BrowserButton{{}, {Pressed: true, Value: 1}, {}}
	p.Axes = make([]float64, 10)
	p.Axes[0] = 1
	p.Axes[9] = -1.0 / 7 * 3 // right-ish: -3/7 is right
	pp.Update(time.Second, []BrowserPad{p}, r)
	info := r.added[0]
	if info.Standard || info.Buttons != 3 || info.Axes != 10 || info.Hats != 1 || info.Rumble {
		t.Errorf("info = %+v", info)
	}
	g := gamepaddb.GUID(info.GUID).Info()
	if g.Vendor != 0x046d || g.Product != 0xc216 || busOf(info.GUID) != gamepaddb.BusUSB {
		t.Errorf("GUID = %+v", g)
	}
	if info.GUID[15] != 5 {
		t.Errorf("the subtype is %d, want 5 (Windows, not standard)", info.GUID[15])
	}
	want := []platform.PadEvent{
		{Pad: info.ID, Time: time.Second, Kind: platform.PadButton, Index: 1, Value: 1},
		{Pad: info.ID, Time: time.Second, Kind: platform.PadAxis, Index: 0, Value: 32767},
		{Pad: info.ID, Time: time.Second, Kind: platform.PadHat, Index: 0, Value: gamepaddb.HatRight},
	}
	for i := 1; i < 9; i++ {
		want = append(want[:i+1], append([]platform.PadEvent{{Pad: info.ID, Time: time.Second, Kind: platform.PadAxis, Index: i, Value: 0}}, want[i+1:]...)...)
	}
	// Order: the pressed button, the nine axes (the hat's is left out) and the hat.
	if len(r.events) != 11 {
		t.Fatalf("events = %+v", r.events)
	}
	if r.events[0] != want[0] || r.events[1] != want[1] || r.events[10].Kind != platform.PadHat || r.events[10].Value != gamepaddb.HatRight {
		t.Errorf("events = %+v", r.events)
	}
	// Later: the button comes up and the hat centres.
	r.events = nil
	p.Timestamp = 2
	p.Buttons[1].Pressed = false
	p.Axes[9] = 0
	pp.Update(2*time.Second, []BrowserPad{p}, r)
	if len(r.events) != 2 || r.events[0].Kind != platform.PadButton || r.events[0].Value != 0 || r.events[1].Kind != platform.PadHat || r.events[1].Value != 0 {
		t.Errorf("events = %+v", r.events)
	}

	// On Linux axis 9 is only an axis.
	pp2 := NewPadPoller(OSLinux, &ids)
	r2 := &recorder{}
	pp2.Update(time.Second, []BrowserPad{p}, r2)
	if r2.added[0].Hats != 0 || r2.added[0].GUID[15] != 2 {
		t.Errorf("info = %+v", r2.added[0])
	}
}

func TestPadInfoOfOtherKinds(t *testing.T) {
	var ids atomic.Int64
	tests := []struct {
		name    string
		os      OperatingSystem
		pad     BrowserPad
		vendor  uint16
		product uint16
		bus     uint16
		rumbleT bool
	}{
		{"xbox one pad with trigger rumble on a phone", OSAndroid, BrowserPad{ID: "xinput", Mapping: "standard", TriggerRumble: true, DualRumble: true}, 0x045e, 0x02e0, gamepaddb.BusBluetooth, true},
		{"xbox one pad on a desktop", OSWindows, BrowserPad{ID: "xinput", Mapping: "standard", TriggerRumble: true}, 0x045e, 0x02ea, gamepaddb.BusUSB, true},
		{"nameless pad", OSMacOS, BrowserPad{ID: "Mystery"}, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pad.Connected = true
			info := tt.pad.padInfo(platform.PadID(ids.Add(1)), PadOSNumber(tt.os))
			g := gamepaddb.GUID(info.GUID).Info()
			if tt.vendor != 0 && (g.Vendor != tt.vendor || g.Product != tt.product || busOf(info.GUID) != tt.bus) {
				t.Errorf("GUID = %+v", g)
			}
			if info.RumbleTriggers != tt.rumbleT {
				t.Errorf("RumbleTriggers = %v", info.RumbleTriggers)
			}
			if tt.vendor == 0 && info.GUID == (platform.PadInfo{}).GUID {
				t.Error("the GUID is empty")
			}
		})
	}
}

// busOf reads the bus at the start of a GUID.
func busOf(g [16]byte) uint16 { return binary.LittleEndian.Uint16(g[:2]) }
