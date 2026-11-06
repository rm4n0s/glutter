// Package evdev is the gamepad backend of Linux: it finds the pads among the event devices of /dev/input, reads their buttons,
// axes and hats, and reports them to the pad manager (internal/pads) through platform.Gamepads. It is plain Go over the
// standard library's syscall, with no library to load (SPEC §28.2).
//
// The decoding of the kernel's events and the numbering of buttons, axes and hats are translated to Go and changed from
// src/joystick/linux/SDL_sysjoystick.c of SDL 3 (zlib license, third_party/sdl/LICENSE). They are in files that carry no OS
// tag, so that tables of events test them on any machine; the files that talk to the kernel carry the linux tag.
//
// Flutter: no counterpart
package evdev

import "github.com/rm4n0s/glutter/internal/gamepaddb"

// The codes of linux/input-event-codes.h that the backend uses.
const (
	evSyn = 0x00
	evKey = 0x01
	evRel = 0x02
	evAbs = 0x03
	evMsc = 0x04
	evFF  = 0x15

	synReport  = 0
	synDropped = 3

	keyMax = 0x2ff
	absMax = 0x3f
	absCnt = absMax + 1
	relMax = 0x0f
	evMax  = 0x1f
	ffMax  = 0x7f

	btnMisc     = 0x100
	btn0        = 0x100
	btn1        = 0x101
	btnLeft     = 0x110
	btnJoystick = 0x120
	btnTrigger  = 0x120
	btnGamepad  = 0x130
	btnSouth    = 0x130 // BTN_A
	btnEast     = 0x131 // BTN_B
	btnNorth    = 0x133 // BTN_X
	btnWest     = 0x134 // BTN_Y
	btnTL       = 0x136
	btnTR       = 0x137
	btnTL2      = 0x138
	btnTR2      = 0x139
	btnSelect   = 0x13a
	btnStart    = 0x13b
	btnMode     = 0x13c
	btnThumbL   = 0x13d
	btnThumbR   = 0x13e
	btnDigi     = 0x140
	btnToolPen  = 0x140
	btnToolFing = 0x145
	btnTouch    = 0x14a
	btnStylus   = 0x14b
	btnDpadUp   = 0x220
	btnDpadDown = 0x221
	btnDpadLeft = 0x222
	btnDpadRite = 0x223
	btnGripL    = 0x224
	btnGripR    = 0x225
	btnGripL2   = 0x226
	btnGripR2   = 0x227
	btnTrigHap1 = 0x2c0
	btnTrigHap2 = 0x2c1
	btnTrigHap3 = 0x2c2
	btnTrigHap4 = 0x2c3
	btnTrigHap5 = 0x2c4
	btnTrigHap6 = 0x2c5
	btnTrigHap7 = 0x2c6
	btnTrigHap8 = 0x2c7
	keyRecord   = 167
	keyEsc      = 1
	keyEnter    = 28
	keySpace    = 57

	absX     = 0x00
	absY     = 0x01
	absZ     = 0x02
	absRX    = 0x03
	absRY    = 0x04
	absRZ    = 0x05
	absThrot = 0x06
	absRudd  = 0x07
	absWheel = 0x08
	absGas   = 0x09
	absBrake = 0x0a
	absHat0X = 0x10
	absHat0Y = 0x11
	absHat1X = 0x12
	absHat1Y = 0x13
	absHat2X = 0x14
	absHat2Y = 0x15
	absHat3Y = 0x17
	absMTX   = 0x35
	absMTY   = 0x36

	relX = 0x00
	relY = 0x01

	propDirect        = 0x01
	propAccelerometer = 0x06

	vendorSony      = 0x054c
	vendorMicrosoft = 0x045e
)

// bits is a bit set that the kernel fills (the result of EVIOCGBIT and EVIOCGPROP): bit i is bit i%8 of byte i/8, as on every
// little-endian machine.
type bits []byte

func (b bits) has(i int) bool { return i >= 0 && i/8 < len(b) && b[i/8]>>(i%8)&1 != 0 }

func (b bits) set(i int) bits {
	for len(b) <= i/8 {
		b = append(b, 0)
	}
	b[i/8] |= 1 << (i % 8)
	return b
}

// absInfo is the kernel's struct input_absinfo.
type absInfo struct {
	Value, Minimum, Maximum, Fuzz, Flat, Resolution int32
}

// inputID is the kernel's struct input_id.
type inputID struct {
	Bus, Vendor, Product, Version uint16
}

// caps is what an event device says about itself.
type caps struct {
	ID   inputID
	Name string
	// Driver is the name of the kernel driver of the device ("xpad"), when it could be found.
	Driver string
	Key    bits
	Abs    bits
	Rel    bits
	FF     bits
	Prop   bits
	Info   [absCnt]absInfo
}

// isJoystick guesses whether an event device is a joystick or a gamepad, the way udev's input_id does: a device with the
// axes of a stick or the buttons of a pad, and nothing of a mouse, a tablet, a touchpad, a touch screen, an accelerometer or a
// keyboard.
func (c *caps) isJoystick() bool {
	if c.Prop.has(propAccelerometer) {
		return false
	}
	hasAbsXY := c.Abs.has(absX) && c.Abs.has(absY)
	hasMT := c.Abs.has(absMTX) && c.Abs.has(absMTY)
	isTablet := c.Key.has(btnStylus) || c.Key.has(btnToolPen)
	fingerNoPen := c.Key.has(btnToolFing) && !c.Key.has(btnToolPen)
	isDirect := c.Prop.has(propDirect)
	hasMouseButton := c.Key.has(btnLeft)
	hasTouch := c.Key.has(btnTouch)
	isRelMouse := c.Rel.has(relX) && c.Rel.has(relY) && hasMouseButton

	joyish := c.Key.has(btnTrigger) || c.Key.has(btnSouth) || c.Key.has(btn1) ||
		c.Abs.has(absRX) || c.Abs.has(absRY) || c.Abs.has(absRZ) || c.Abs.has(absThrot) || c.Abs.has(absRudd) ||
		c.Abs.has(absWheel) || c.Abs.has(absGas) || c.Abs.has(absBrake)
	isKeyboard := c.Key.has(keyEsc) && c.Key.has(keyEnter) && c.Key.has(keySpace) && !c.Key.has(btnSouth) && !c.Key.has(btnTrigger)

	switch {
	case isTablet, isRelMouse, isKeyboard:
		return false
	case hasMT && (hasTouch || isDirect || fingerNoPen):
		return false
	case hasAbsXY:
		if fingerNoPen && !isDirect {
			return false
		}
		if hasMouseButton || hasTouch || isDirect {
			return false
		}
		return joyish
	}
	// A device with buttons and no stick (a pedal board, a pad with only a D-pad of buttons).
	return joyish && !fingerNoPen
}

// layout is how the buttons, axes and hats of an event device are numbered for the mapping, which is the numbering of SDL's
// evdev driver: the buttons from BTN_JOYSTICK up first, then those below it; the axes in the order of their codes, leaving out
// the digital hats; the hats in the order of the hat axes.
type layout struct {
	// keyMap gives the button number of an evdev key code, -1 for a key the pad does not have.
	keyMap [keyMax + 1]int16
	// absMap gives the axis number of an evdev axis code, -1 for an axis the pad does not have (or that is a hat).
	absMap  [absCnt]int16
	buttons int
	axes    int
	hats    int
	// hatIndex gives the hat number of the hat axis pair 0..3, -1 for none.
	hatIndex   [4]int
	hatCorrect [4]hatCorrect
	absCorrect [absCnt]axisCorrect
	// keyCodes lists the key codes that are buttons, by button number.
	keyCodes []int
	absCodes []int
}

type hatCorrect struct {
	min, max     [2]int
	useDeadzones bool
}

type axisCorrect struct {
	min, max int
	scale    float32
}

// guessAxesAreDigitalHat guesses whether two axes are a hat, which has at most 9 states, or a stick: the kernel makes no
// difference between them. A nil axis is one the device does not have.
func guessAxesAreDigitalHat(x, y *absInfo) bool {
	if x == nil && y == nil {
		return false
	}
	// Both axes constrained to -1..1 are digital.
	if (x == nil || (x.Minimum == -1 && x.Maximum == 1)) && (y == nil || (y.Minimum == -1 && y.Maximum == 1)) {
		return true
	}
	// Both without fuzz, flat and resolution are probably digital.
	if (x == nil || (x.Fuzz == 0 && x.Flat == 0 && x.Resolution == 0)) && (y == nil || (y.Fuzz == 0 && y.Flat == 0 && y.Resolution == 0)) {
		return true
	}
	return false
}

// newLayout numbers the inputs of the device.
func newLayout(c *caps) *layout {
	l := &layout{hatIndex: [4]int{-1, -1, -1, -1}}
	for i := range l.keyMap {
		l.keyMap[i] = -1
	}
	for i := range l.absMap {
		l.absMap[i] = -1
	}
	addKey := func(code int) {
		l.keyMap[code] = int16(l.buttons)
		l.keyCodes = append(l.keyCodes, code)
		l.buttons++
	}
	for i := btnJoystick; i < keyMax; i++ {
		if c.Key.has(i) {
			addKey(i)
		}
	}
	for i := 0; i < btnJoystick; i++ {
		if c.Key.has(i) {
			addKey(i)
		}
	}
	for i := absHat0X; i <= absHat3Y; i += 2 {
		var x, y *absInfo
		if c.Abs.has(i) {
			x = &c.Info[i]
		}
		if c.Abs.has(i + 1) {
			y = &c.Info[i+1]
		}
		if !guessAxesAreDigitalHat(x, y) {
			continue
		}
		hc := hatCorrect{useDeadzones: true, min: [2]int{-1, -1}, max: [2]int{1, 1}}
		if x != nil {
			hc.min[0], hc.max[0] = int(x.Minimum), int(x.Maximum)
		}
		if y != nil {
			hc.min[1], hc.max[1] = int(y.Minimum), int(y.Maximum)
		}
		idx := (i - absHat0X) / 2
		l.hatIndex[idx] = l.hats
		l.hatCorrect[idx] = hc
		l.hats++
	}
	for i := 0; i < absMax; i++ {
		// Skip digital hats.
		if i >= absHat0X && i <= absHat3Y && l.hatIndex[(i-absHat0X)/2] >= 0 {
			continue
		}
		if !c.Abs.has(i) {
			continue
		}
		info := c.Info[i]
		l.absMap[i] = int16(l.axes)
		l.absCodes = append(l.absCodes, i)
		ac := axisCorrect{min: int(info.Minimum), max: int(info.Maximum)}
		if ac.min != ac.max {
			ac.scale = float32(gamepaddb.AxisMax-gamepaddb.AxisMin) / float32(ac.max-ac.min)
		}
		l.absCorrect[i] = ac
		l.axes++
	}
	return l
}

// axisValue scales the value of an axis to the range of axes: from the kernel's minimum..maximum to -32768..32767.
func (l *layout) axisValue(code, value int) int {
	ac := l.absCorrect[code]
	if ac.min != ac.max {
		value = int(floor32(float32(value-ac.min)*ac.scale + gamepaddb.AxisMin + 0.5))
	}
	return max(gamepaddb.AxisMin, min(gamepaddb.AxisMax, value))
}

func floor32(v float32) float32 {
	i := float32(int64(v))
	if v < i {
		return i - 1
	}
	return i
}
