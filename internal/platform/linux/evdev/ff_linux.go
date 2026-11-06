//go:build linux && !android

package evdev

import (
	"encoding/binary"
	"time"
	"unsafe"
)

// Force feedback: the kernel's struct ff_effect and the rumble effect that the pads' motors play (linux/input.h). The rumble
// follows the way SDL 3's evdev driver plays one (src/joystick/linux/SDL_sysjoystick.c, translated to Go and changed): an effect
// of type FF_RUMBLE is uploaded with the strengths and the length, then played with an EV_FF event, and the kernel stops it when
// its length is over.

const (
	// ffRumble is the code of the rumble effect, and a bit of the capabilities of a pad that can play it.
	ffRumble = 0x50
	// maxRumbleMillis is the longest an effect can be: its length is 16 bits of milliseconds.
	maxRumbleMillis = 0xffff
)

// ptrSize is the size of a pointer: the kernel's union of effects ends with a periodic effect, which holds one.
const ptrSize = int(unsafe.Sizeof(uintptr(0)))

// ffEffect is the kernel's struct ff_effect: the header, then a union of the effects, of which only the rumble is used. The
// union is 24 bytes and a pointer long (a periodic effect), and it begins at byte 16 on every architecture Go runs on.
type ffEffect struct {
	Type            uint16
	ID              int16
	Direction       uint16
	TriggerButton   uint16
	TriggerInterval uint16
	ReplayLength    uint16
	ReplayDelay     uint16
	_               uint16
	// Union holds the strong and weak magnitude of a rumble effect, 16 bits each.
	Union [24 + ptrSize]byte
}

// rumbleEffect makes the effect of a rumble: low is the strength of the strong (low frequency) motor and high of the weak (high
// frequency) one, from 0 to 65535, and the effect lasts d, from 1 millisecond to 65.535 seconds. An id of -1 asks the kernel to
// give the effect a new one.
func rumbleEffect(id int16, low, high uint16, d time.Duration) ffEffect {
	ms := int64(d / time.Millisecond)
	ms = max(1, min(maxRumbleMillis, ms))
	e := ffEffect{Type: ffRumble, ID: id, ReplayLength: uint16(ms)}
	binary.NativeEndian.PutUint16(e.Union[0:], low)
	binary.NativeEndian.PutUint16(e.Union[2:], high)
	return e
}
