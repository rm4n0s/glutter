package winpad

import (
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The code of the HID pads that Raw Input lists: the buttons, axes and hats of a device are found from its parsed report
// descriptor and numbered as SDL's raw input driver numbers them, so the mappings of SDL's table fit. Translated to Go and
// changed from src/joystick/windows/SDL_rawinputjoystick.c of SDL 3.

// hidAxis is an axis of a HID device: where its value is in the items of a report and its range.
type hidAxis struct {
	index uint16
	bits  uint16
	min   int32
	max   int32
}

// hidHat is a hat switch.
type hidHat struct {
	index    uint16
	min, max int32
}

// Layout is the buttons, axes and hats of a HID device, in the order that numbers them for the mapping.
//
// Flutter: no counterpart
type Layout struct {
	buttons []uint16
	axes    []hidAxis
	hats    []hidHat
	// MaxItems is the most items that a report can have, the size of the buffer that HIDSource.GetData is given.
	MaxItems int
}

// Counts are the numbers of buttons, axes and hats.
//
// Flutter: no counterpart
func (l *Layout) Counts() (buttons, axes, hats int) { return len(l.buttons), len(l.axes), len(l.hats) }

// NewLayout numbers the inputs of a device from its button and value capabilities. The buttons are those of the button page, in
// the order of their capabilities, a range counting each of its data indices; the values are sorted by usage (so X comes before
// Y), the hat switches are the hats and the rest are the axes, and the values that are ranges are left out, as SDL does.
//
// Flutter: no counterpart
func NewLayout(buttons []win32.HidpButtonCaps, values []win32.HidpValueCaps, maxItems int) *Layout {
	l := &Layout{MaxItems: maxItems}
	for i := range buttons {
		c := &buttons[i]
		if c.UsagePage != win32.HIDUsagePageButton {
			continue
		}
		if c.IsRange != 0 {
			for idx := int(c.DataIndexMin()); idx <= int(c.DataIndexMax()); idx++ {
				l.buttons = append(l.buttons, uint16(idx))
			}
		} else {
			l.buttons = append(l.buttons, c.DataIndexMin())
		}
	}
	sorted := slices.Clone(values)
	slices.SortStableFunc(sorted, func(a, b win32.HidpValueCaps) int { return int(a.UsageMin()) - int(b.UsageMin()) })
	for i := range sorted {
		c := &sorted[i]
		if c.IsRange != 0 {
			continue
		}
		if c.UsageMin() == win32.HIDUsageGenericHatSwitch {
			l.hats = append(l.hats, hidHat{index: c.DataIndexMin(), min: c.LogicalMin, max: c.LogicalMax})
			continue
		}
		l.axes = append(l.axes, hidAxis{index: c.DataIndexMin(), bits: c.BitSize, min: c.LogicalMin, max: c.LogicalMax})
	}
	return l
}

// IsPad says whether a device with this layout is worth listing as a pad: it has buttons and an axis or a hat to go with them
// (a device with a single button is a switch, and one with only axes a mouse-like thing).
//
// Flutter: no counterpart
func (l *Layout) IsPad() bool { return len(l.buttons) > 0 && len(l.axes)+len(l.hats) > 0 }

// hidState is what a report says: which buttons are down, the axes and the hats.
type hidState struct {
	buttons []bool
	axes    []int16
	hats    []int
}

// axisValue scales the raw value of an axis to the range of axes. A device whose axes are signed has them sign extended from
// their size; the value is then spread over the range of the axis, so an axis of 8 bits and one of 16 both go from −32768 to
// 32767 (SDL takes the 16 low bits of the value and subtracts 32768, which fits only 16-bit axes of 0 to 65535).
func axisValue(raw uint32, a hidAxis) int16 {
	v := int64(raw)
	if a.min < 0 && a.bits > 0 && a.bits < 32 {
		shift := 64 - uint(a.bits)
		v = v << shift >> shift
	}
	lo, hi := int64(a.min), int64(a.max)
	if hi <= lo {
		return 0
	}
	v = max(lo, min(hi, v))
	return int16((v-lo)*65535/(hi-lo) - 32768)
}

// hatMask turns the value of a hat switch into the mask of directions of platform.PadHat. A hat has 8 positions starting with up
// and going clockwise (or 4: up, right, down, left); a value outside the logical range is the centre.
func hatMask(raw uint32, h hidHat) int {
	v := int64(int32(raw)) - int64(h.min)
	n := int64(h.max) - int64(h.min) + 1
	if v < 0 || v >= n {
		return 0
	}
	switch n {
	case 4:
		v *= 2
	case 8:
	default:
		return 0
	}
	const up, right, down, left = gamepaddb.HatUp, gamepaddb.HatRight, gamepaddb.HatDown, gamepaddb.HatLeft
	return [...]int{up, up | right, right, down | right, down, down | left, left, up | left}[v]
}

// item finds the item of a data index: at the place of its index when it is there, else by looking for it.
func item(index uint16, items []win32.HidpData) (win32.HidpData, bool) {
	if int(index) < len(items) && items[index].DataIndex == index {
		return items[index], true
	}
	for _, it := range items {
		if it.DataIndex == index {
			return it, true
		}
	}
	return win32.HidpData{}, false
}

// decode reads the items of a report. A button that has no item is up; an axis or hat with none keeps the value it had in prev
// (a report need not hold every value).
func (l *Layout) decode(items []win32.HidpData, prev *hidState) hidState {
	s := hidState{buttons: make([]bool, len(l.buttons)), axes: make([]int16, len(l.axes)), hats: make([]int, len(l.hats))}
	for i, idx := range l.buttons {
		if it, ok := item(idx, items); ok && it.On() {
			s.buttons[i] = true
		}
	}
	for i, a := range l.axes {
		if it, ok := item(a.index, items); ok {
			s.axes[i] = axisValue(it.RawValue, a)
		} else if prev != nil && i < len(prev.axes) {
			s.axes[i] = prev.axes[i]
		}
	}
	for i, h := range l.hats {
		if it, ok := item(h.index, items); ok {
			s.hats[i] = hatMask(it.RawValue, h)
		} else if prev != nil && i < len(prev.hats) {
			s.hats[i] = prev.hats[i]
		}
	}
	return s
}

// hidTracker turns the reports of one HID pad into events of what changed. The first report gives every axis and hat as well,
// so that the layer above learns the rest values.
type hidTracker struct {
	id     platform.PadID
	layout *Layout
	last   *hidState
}

func (t *hidTracker) update(now time.Duration, items []win32.HidpData, out []platform.PadEvent) []platform.PadEvent {
	s := t.layout.decode(items, t.last)
	first := t.last == nil
	for i, down := range s.buttons {
		if first && !down {
			continue
		}
		if first || down != t.last.buttons[i] {
			v := 0
			if down {
				v = 1
			}
			out = append(out, platform.PadEvent{Pad: t.id, Time: now, Kind: platform.PadButton, Index: i, Value: v})
		}
	}
	for i, v := range s.axes {
		if first || v != t.last.axes[i] {
			out = append(out, platform.PadEvent{Pad: t.id, Time: now, Kind: platform.PadAxis, Index: i, Value: int(v)})
		}
	}
	for i, v := range s.hats {
		if first || v != t.last.hats[i] {
			out = append(out, platform.PadEvent{Pad: t.id, Time: now, Kind: platform.PadHat, Index: i, Value: v})
		}
	}
	t.last = &s
	return out
}

// isXInputPath says whether the path of a HID device is the interface of an XInput pad ("IG_" is in the paths of the devices
// that Microsoft's XInput driver serves): XInput reports that pad, so Raw Input skips it and the pad is not listed twice.
func isXInputPath(path string) bool { return strings.Contains(path, "IG_") }

// busOfPath guesses the bus of a HID device from its path: Bluetooth HID devices have the Bluetooth service class of the HID
// profile (or of HID over GATT, for Bluetooth Low Energy) or the BTHENUM enumerator in it, the rest are taken for USB.
func busOfPath(path string) uint16 {
	p := strings.ToUpper(path)
	if strings.Contains(p, "BTHENUM") || strings.Contains(p, "BTHLEDEVICE") || strings.Contains(p, "{00001124-0000-1000-8000-00805F9B34FB}") || strings.Contains(p, "{00001812-0000-1000-8000-00805F9B34FB}") {
		return gamepaddb.BusBluetooth
	}
	return gamepaddb.BusUSB
}
