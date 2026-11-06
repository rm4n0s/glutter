package gamepadhid

import (
	"hash/crc32"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// standardGravity is the acceleration of gravity in metres a second squared, the unit of the accelerometers.
const standardGravity = 9.80665

// usbPacketLength is the size of the buffers that the drivers read feature reports into.
const usbPacketLength = 64

// le16 reads a 16 bit little-endian number as the signed value the sensors report.
func le16(b []byte) int16 { return int16(uint16(b[0]) | uint16(b[1])<<8) }

// ule16 reads a 16 bit little-endian number.
func ule16(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }

// ule32 reads a 32 bit little-endian number.
func ule32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// stick8 turns a stick of one byte into the range of an axis, as SDL does (0 is the minimum, 255 the maximum).
func stick8(v uint8) int16 { return int16(int(v)*257 - 32768) }

// setHat sets the four D-pad buttons from a hat of eight positions, where 0 is up and the numbers go clockwise, and anything
// else is the centre (the hat of the PlayStation pads and of the simple report of the Switch Pro controller).
func setHat(s *padstd.Snapshot, v uint8) {
	up, right, down, left := false, false, false, false
	switch v {
	case 0:
		up = true
	case 1:
		up, right = true, true
	case 2:
		right = true
	case 3:
		right, down = true, true
	case 4:
		down = true
	case 5:
		down, left = true, true
	case 6:
		left = true
	case 7:
		left, up = true, true
	}
	setDPad(s, up, right, down, left)
}

// setDPad sets the four D-pad buttons.
func setDPad(s *padstd.Snapshot, up, right, down, left bool) {
	s.Buttons[gamepaddb.DPadUp] = up
	s.Buttons[gamepaddb.DPadRight] = right
	s.Buttons[gamepaddb.DPadDown] = down
	s.Buttons[gamepaddb.DPadLeft] = left
}

// hidpCRC is the CRC-32 that the Bluetooth reports of the PlayStation pads end with: it covers the HIDP header (0xA1 for what
// the pad sends, 0xA2 for what it is sent) and the report up to its last four bytes.
func hidpCRC(header byte, data []byte) uint32 {
	crc := crc32.Update(0, crc32.IEEETable, []byte{header})
	return crc32.Update(crc, crc32.IEEETable, data)
}

// verifyCRC says whether the last four bytes of a Bluetooth report of a PlayStation pad are the CRC of the rest.
func verifyCRC(data []byte) bool {
	n := len(data) - 4
	return hidpCRC(0xA1, data[:n]) == ule32(data[n:])
}

// putCRC writes the CRC of an output report, header 0xA2, into its last four bytes.
func putCRC(data []byte) {
	n := len(data) - 4
	c := hidpCRC(0xA2, data[:n])
	data[n], data[n+1], data[n+2], data[n+3] = byte(c), byte(c>>8), byte(c>>16), byte(c>>24)
}

// remap spreads val from the range [inMin, inMax] over [outMin, outMax], as SDL's HIDAPI_RemapVal does.
func remap(val, inMin, inMax, outMin, outMax float32) float32 {
	return outMin + (outMax-outMin)*((val-inMin)/(inMax-inMin))
}

// touchTracker sends a finger of a touchpad only when it changed, as SDL's SDL_SendJoystickTouchpad does (src/joystick/
// SDL_joystick.c, translated to Go and changed): the numbers are held from 0 to 1, a finger that is up has no pressure and keeps
// its last place when the report has none, and a report that changes nothing is dropped.
type touchTracker struct {
	fingers [4]platform.PadTouch
}

func (t *touchTracker) send(out Sink, now time.Duration, touchpad, finger int, down bool, x, y, pressure float32) {
	i := touchpad*2 + finger
	if i < 0 || i >= len(t.fingers) {
		return
	}
	f := &t.fingers[i]
	if !down {
		if x == 0 && y == 0 {
			x, y = f.X, f.Y
		}
		pressure = 0
	}
	x, y, pressure = min(max(x, 0), 1), min(max(y, 0), 1), min(max(pressure, 0), 1)
	if down == f.Down && (!down || (x == f.X && y == f.Y && pressure == f.Pressure)) {
		return
	}
	*f = platform.PadTouch{Touchpad: touchpad, Finger: finger, Down: down, X: x, Y: y, Pressure: pressure}
	out.Touch(now, *f)
}

// batteryTracker sends the battery state of a pad only when it changed.
type batteryTracker struct {
	last platform.PadBattery
	set  bool
}

func (b *batteryTracker) send(out Sink, state platform.PadPowerState, percent int) {
	if b.set && b.last.State == state && b.last.Percent == percent {
		return
	}
	b.last = platform.PadBattery{State: state, Percent: percent}
	b.set = true
	out.Battery(b.last)
}

// powerPercent turns the nibble of the level that the PlayStation pads report (0 to 10, in tens) into a percentage.
func powerPercent(level uint8) int { return min(int(level)*10+5, 100) }
