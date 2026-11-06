package web

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// The gamepads of the web (SPEC §28.2, step 31.2): what navigator.getGamepads() gives, turned into pads of the engine. The
// browser's "standard" mapping is the standard layout, and the other pads are raw buttons and axes that the table of mappings
// gives a layout to by the vendor and product in the browser's id. This file has no JavaScript, so the tests run it anywhere; the
// reading of the browser and the rumble are in gamepads_js.go. The naming, the GUID and the numbering follow
// src/joystick/emscripten/SDL_sysjoystick.c of SDL 3 (zlib license, third_party/sdl/LICENSE; translated to Go and changed).

// BrowserButton is a button of a Gamepad of the browser.
//
// Flutter: no counterpart
type BrowserButton struct {
	Pressed bool
	// Value is how far it is pressed, from 0 to 1.
	Value float64
}

// BrowserPad is the state of one Gamepad of the browser, copied out of it.
//
// Flutter: no counterpart
type BrowserPad struct {
	// Index is its place in the array of navigator.getGamepads().
	Index int
	// ID is the browser's name for it ("Xbox 360 Controller (XInput STANDARD GAMEPAD)" in Chrome, "045e-028e-Xbox 360 Controller"
	// in Firefox).
	ID string
	// Mapping is "standard" when the browser maps the pad to the standard layout, else "".
	Mapping   string
	Connected bool
	Buttons   []BrowserButton
	Axes      []float64
	// Timestamp is when the state was last changed, in milliseconds (the browser's DOMHighResTimeStamp).
	Timestamp float64
	// DualRumble and TriggerRumble say that its vibration actuator can play the "dual-rumble" and "trigger-rumble" effects.
	DualRumble, TriggerRumble bool
}

// ParsePadID reads what a browser's id says about a pad: its name without the browser's additions, the vendor and product
// numbers when it has them, and whether it is an XInput pad. Chrome, Edge and Opera write "Wireless Controller (STANDARD GAMEPAD
// Vendor: 054c Product: 09cc)", "usb gamepad (Vendor: 0810 Product: e501)" or "Xbox 360 Controller (XInput STANDARD GAMEPAD)";
// Firefox and Safari write "046d-c216-Logitech Dual Action" (the numbers may lose their leading zeros) or "xinput".
//
// Flutter: no counterpart
func ParsePadID(id string) (name string, vendor, product uint16, xinput bool) {
	name = id
	switch {
	case strings.Index(id, " (STANDARD GAMEPAD") > 0:
		name = id[:strings.Index(id, " (STANDARD GAMEPAD")]
	case strings.Index(id, " (Vendor:") > 0:
		name = id[:strings.Index(id, " (Vendor:")]
	case strings.Index(id, " (XInput") > 0:
		name = id[:strings.Index(id, " (XInput")]
	}
	xinput = strings.Contains(strings.ToLower(id), "xinput")
	if i := strings.Index(id, "Vendor: "); i > 0 {
		vendor = hexField(id[i+len("Vendor: "):])
	}
	if i := strings.Index(id, "Product: "); i > 0 {
		product = hexField(id[i+len("Product: "):])
	}
	if parts := strings.Split(id, "-"); len(parts) > 1 {
		if v, ok := parseHexPrefix(parts[0]); ok {
			if !strings.Contains(id, "Vendor: ") {
				vendor = uint16(v)
			}
			if p, ok := parseHexPrefix(parts[1]); ok && !strings.Contains(id, "Product: ") {
				product = uint16(p)
			}
			// The name is what follows the second dash; with only one dash it is the whole id, as the browsers do not do this.
			if first := strings.Index(id, "-"); strings.Contains(id[first+1:], "-") {
				name = id[first+1+strings.Index(id[first+1:], "-")+1:]
			}
		}
	}
	return strings.TrimSpace(name), vendor, product, xinput
}

// hexField reads the 4 characters at the start of s as a hexadecimal number, 0 when they are not one.
func hexField(s string) uint16 {
	if len(s) > 4 {
		s = s[:4]
	}
	v, ok := parseHexPrefix(s)
	if !ok {
		return 0
	}
	return uint16(v)
}

// parseHexPrefix reads the hexadecimal digits at the start of s as JavaScript's parseInt(s, 16) does: the digits up to the first
// character that is not one; ok is false when there is none.
func parseHexPrefix(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	end := 0
	for end < len(s) && strings.IndexByte("0123456789abcdefABCDEF", s[end]) >= 0 {
		end++
	}
	if end == 0 {
		return 0, false
	}
	if end > 8 {
		end = 8
	}
	v, err := strconv.ParseUint(s[:end], 16, 32)
	return v, err == nil
}

// PadOSNumber numbers the operating systems the way SDL 3's browser backend does for the subtype of a pad's GUID: 1 Android, 2 Linux,
// 3 iOS, 4 macOS, 5 Windows, 0 unknown.
//
// Flutter: no counterpart
func PadOSNumber(os OperatingSystem) uint8 {
	switch os {
	case OSAndroid:
		return 1
	case OSLinux:
		return 2
	case OSIOS:
		return 3
	case OSMacOS:
		return 4
	case OSWindows:
		return 5
	}
	return 0
}

// The buttons of the browser's standard mapping, in its order (the Gamepad API's "standard gamepad" layout).
var standardButtons = [...]gamepaddb.Button{
	gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North,
	gamepaddb.LeftShoulder, gamepaddb.RightShoulder,
	gamepaddb.ButtonCount, gamepaddb.ButtonCount, // the triggers, which are axes here
	gamepaddb.Back, gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick,
	gamepaddb.DPadUp, gamepaddb.DPadDown, gamepaddb.DPadLeft, gamepaddb.DPadRight,
	gamepaddb.Guide,
}

// isStandard says whether a pad has the browser's standard mapping with at least the 16 buttons and 4 axes of it.
func (p *BrowserPad) isStandard() bool {
	return p.Mapping == "standard" && len(p.Buttons) >= 16 && len(p.Axes) >= 4
}

// standardSnapshot is the state of a pad with the standard mapping as the layout: the triggers are the analog value of the
// buttons 6 and 7, the sticks the first four axes.
func (p *BrowserPad) standardSnapshot() padstd.Snapshot {
	var s padstd.Snapshot
	for i, b := range p.Buttons {
		if i < len(standardButtons) && standardButtons[i] != gamepaddb.ButtonCount {
			s.Buttons[standardButtons[i]] = b.Pressed
		}
	}
	s.Axes[gamepaddb.LeftX] = padstd.Signed(p.Axes[0])
	s.Axes[gamepaddb.LeftY] = padstd.Signed(p.Axes[1])
	s.Axes[gamepaddb.RightX] = padstd.Signed(p.Axes[2])
	s.Axes[gamepaddb.RightY] = padstd.Signed(p.Axes[3])
	s.Axes[gamepaddb.LeftTrigger] = padstd.Unit(p.Buttons[6].Value)
	s.Axes[gamepaddb.RightTrigger] = padstd.Unit(p.Buttons[7].Value)
	return s
}

// hatOfAxis reads the hat that browsers on Windows put in axis 9 of a pad without a standard mapping: the 8 directions are 8
// values from −1 to 1 in steps of 2/7, and anything else is the centre.
func hatOfAxis(v float64) int {
	const up, right, down, left = gamepaddb.HatUp, gamepaddb.HatRight, gamepaddb.HatDown, gamepaddb.HatLeft
	for i, mask := range [...]int{up, up | right, right, down | right, down, down | left, left, up | left} {
		if want := -1 + float64(i)*2/7; v-want < 0.00001 && v-want > -0.00001 {
			return mask
		}
	}
	return 0
}

// rawState is the state of a pad without the standard mapping: its buttons and axes as they are, and the hat of axis 9 where
// the browser makes one.
type rawState struct {
	buttons []bool
	axes    []int16
	hat     int
}

// hatIsAxis says that axis 9 of the pad is a hat (SDL guesses it for a Windows browser and a pad of 10 axes or more, which
// is what the browsers there give the hat of an unmapped controller).
func (p *BrowserPad) hatIsAxis(osID uint8) bool {
	return osID == 5 && !p.isStandard() && len(p.Axes) >= 10
}

func (p *BrowserPad) rawState(osID uint8) rawState {
	s := rawState{buttons: make([]bool, len(p.Buttons)), axes: make([]int16, len(p.Axes))}
	for i, b := range p.Buttons {
		s.buttons[i] = b.Pressed
	}
	hat := p.hatIsAxis(osID)
	for i, v := range p.Axes {
		if hat && i == 9 {
			s.hat = hatOfAxis(v)
			continue
		}
		s.axes[i] = padstd.Signed(v)
	}
	return s
}

// padInfo makes the description of a pad for the manager.
func (p *BrowserPad) padInfo(id platform.PadID, osID uint8) platform.PadInfo {
	name, vendor, product, xinput := ParsePadID(p.ID)
	bus := gamepaddb.BusUnknown
	switch osID {
	case 0:
	case 1, 3: // Android and iOS: the pads are on Bluetooth
		bus = gamepaddb.BusBluetooth
	default:
		bus = gamepaddb.BusUSB
	}
	if vendor == 0 && product == 0 && xinput {
		// A generic vendor and product for an XInput pad (the Xbox 360 controller); one that can rumble its triggers is an Xbox One
		// controller (S model, USB or Bluetooth).
		vendor, product = 0x045e, 0x028e
		if p.TriggerRumble {
			product = 0x02ea
			if bus == gamepaddb.BusBluetooth {
				product = 0x02e0
			}
		}
	}
	subtype := osID
	if p.Mapping == "standard" {
		subtype += 0x80
	}
	info := platform.PadInfo{ID: id, Name: gamepaddb.DeviceName(vendor, product, "", name), Battery: -1, Rumble: p.DualRumble, RumbleTriggers: p.TriggerRumble}
	if vendor != 0 && product != 0 {
		info.GUID = gamepaddb.NewGUID(bus, vendor, product, 0, "", name, 0, subtype)
	} else {
		g := gamepaddb.NewGUIDForName(info.Name)
		g[15] = subtype
		info.GUID = g
	}
	if p.isStandard() {
		info.Standard = true
		info.Axes, info.Buttons = int(gamepaddb.AxisCount), int(gamepaddb.ButtonCount)
		return info
	}
	info.Axes, info.Buttons = len(p.Axes), len(p.Buttons)
	if p.hatIsAxis(osID) {
		info.Hats = 1
	}
	return info
}

// PadPoller follows the pads of the browser from one reading of navigator.getGamepads() to the next, and tells a handler what
// changed. It belongs to the UI goroutine.
//
// Flutter: no counterpart
type PadPoller struct {
	osID   uint8
	nextID *atomic.Int64
	pads   map[int]*polled
}

type polled struct {
	id        platform.PadID
	browserID string
	standard  bool
	hatAxis   bool
	timestamp float64
	tracker   *padstd.Tracker
	last      *rawState
}

// NewPadPoller makes a poller for a browser on the given operating system. ids numbers the pads; it may be shared.
//
// Flutter: no counterpart
func NewPadPoller(os OperatingSystem, ids *atomic.Int64) *PadPoller {
	return &PadPoller{osID: PadOSNumber(os), nextID: ids, pads: map[int]*polled{}}
}

// Count is the number of pads that are connected.
//
// Flutter: no counterpart
func (pp *PadPoller) Count() int { return len(pp.pads) }

// IndexOf gives the index in the browser's array of the pad with an id.
//
// Flutter: no counterpart
func (pp *PadPoller) IndexOf(id platform.PadID) (int, bool) {
	for index, p := range pp.pads {
		if p.id == id {
			return index, true
		}
	}
	return 0, false
}

// IDOf gives the id of the pad at an index of the browser's array, 0 when there is none.
//
// Flutter: no counterpart
func (pp *PadPoller) IDOf(index int) platform.PadID {
	if p := pp.pads[index]; p != nil {
		return p.id
	}
	return 0
}

// Update takes a reading: pads that are new are added, those that are gone are removed, and the others report what changed since
// the last reading (a pad whose timestamp is the one it had is not looked at). snapshot is the array of the browser, with the
// gaps left out; now is the time of the reading.
//
// Flutter: no counterpart
func (pp *PadPoller) Update(now time.Duration, snapshot []BrowserPad, h platform.PadHandler) {
	present := map[int]bool{}
	var batch []platform.PadEvent
	for i := range snapshot {
		bp := &snapshot[i]
		if !bp.Connected {
			continue
		}
		present[bp.Index] = true
		cur := pp.pads[bp.Index]
		if cur != nil && cur.browserID != bp.ID {
			// Another pad took the place.
			pp.remove(bp.Index, h)
			cur = nil
		}
		if cur == nil {
			cur = pp.add(bp, h)
			batch = pp.report(now, cur, bp, batch)
			continue
		}
		if bp.Timestamp != 0 && bp.Timestamp == cur.timestamp {
			continue
		}
		batch = pp.report(now, cur, bp, batch)
	}
	for index := range pp.pads {
		if !present[index] {
			pp.remove(index, h)
		}
	}
	if len(batch) > 0 {
		h.PadEvents(batch)
	}
}

func (pp *PadPoller) add(bp *BrowserPad, h platform.PadHandler) *polled {
	id := platform.PadID(pp.nextID.Add(1))
	p := &polled{id: id, browserID: bp.ID, standard: bp.isStandard(), hatAxis: bp.hatIsAxis(pp.osID)}
	if p.standard {
		p.tracker = padstd.NewTracker(id)
	}
	pp.pads[bp.Index] = p
	h.PadAdded(bp.padInfo(id, pp.osID))
	return p
}

func (pp *PadPoller) remove(index int, h platform.PadHandler) {
	p := pp.pads[index]
	delete(pp.pads, index)
	h.PadRemoved(p.id)
}

// report appends the events of what changed in a pad. The first reading of a raw pad gives every axis and hat as well.
func (pp *PadPoller) report(now time.Duration, p *polled, bp *BrowserPad, out []platform.PadEvent) []platform.PadEvent {
	p.timestamp = bp.Timestamp
	if p.standard {
		s := bp.standardSnapshot()
		return p.tracker.Update(now, &s, out)
	}
	s := bp.rawState(pp.osID)
	first := p.last == nil
	for i, down := range s.buttons {
		if first && !down {
			continue
		}
		if first || i >= len(p.last.buttons) || down != p.last.buttons[i] {
			v := 0
			if down {
				v = 1
			}
			out = append(out, platform.PadEvent{Pad: p.id, Time: now, Kind: platform.PadButton, Index: i, Value: v})
		}
	}
	for i, v := range s.axes {
		if p.hatAxis && i == 9 {
			continue
		}
		if first || i >= len(p.last.axes) || v != p.last.axes[i] {
			out = append(out, platform.PadEvent{Pad: p.id, Time: now, Kind: platform.PadAxis, Index: i, Value: int(v)})
		}
	}
	if p.hatAxis && (first || s.hat != p.last.hat) {
		out = append(out, platform.PadEvent{Pad: p.id, Time: now, Kind: platform.PadHat, Index: 0, Value: s.hat})
	}
	p.last = &s
	return out
}
