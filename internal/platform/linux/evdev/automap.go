package evdev

import (
	"strconv"
	"strings"
)

// This file is translated to Go and changed from LINUX_JoystickGetGamepadMapping in src/joystick/linux/SDL_sysjoystick.c of
// SDL 3 (zlib license, third_party/sdl/LICENSE). It is based on the Linux Gamepad Specification
// (https://www.kernel.org/doc/html/latest/input/gamepad.html) and the conventions of Android's controllers.

// mappingOf makes the mapping text of a device that follows the Linux Gamepad Specification, or "" when it is not a gamepad
// by the specification (it has no BTN_GAMEPAD). The text is in the community's format ("a:b0,b:b1,..."), numbered by the
// layout.
func mappingOf(c *caps, l *layout) string {
	if l.keyMap[btnGamepad] < 0 {
		return ""
	}
	var b strings.Builder
	add := func(name, input string) {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(input)
		b.WriteByte(',')
	}
	key := func(code int) (string, bool) {
		if code <= keyMax && l.keyMap[code] >= 0 {
			return "b" + strconv.Itoa(int(l.keyMap[code])), true
		}
		return "", false
	}
	axis := func(code int) (string, bool) {
		if code < absCnt && l.absMap[code] >= 0 {
			return "a" + strconv.Itoa(int(l.absMap[code])), true
		}
		return "", false
	}
	hat := func(n, mask int) (string, bool) {
		if l.hatIndex[n] >= 0 {
			return "h" + strconv.Itoa(l.hatIndex[n]) + "." + strconv.Itoa(mask), true
		}
		return "", false
	}
	addKey := func(name string, code int) bool {
		if in, ok := key(code); ok {
			add(name, in)
			return true
		}
		return false
	}

	addKey("a", btnSouth)
	addKey("b", btnEast)
	// Xbox pads use BTN_X and BTN_Y, and Sony's pads BTN_WEST and BTN_NORTH, which are not the same buttons.
	if c.ID.Vendor == vendorSony {
		addKey("x", btnWest)
		addKey("y", btnNorth)
	} else {
		addKey("x", btnNorth)
		addKey("y", btnWest)
	}
	addKey("back", btnSelect)
	addKey("start", btnStart)
	addKey("leftstick", btnThumbL)
	addKey("rightstick", btnThumbR)
	addKey("guide", btnMode)

	// The D-pad, the shoulder buttons and the triggers can be digital, analog, or both. Prefer digital shoulder buttons, but
	// settle for a digital or analog hat.
	mapped := 0
	if addKey("leftshoulder", btnTL) {
		mapped |= 1
	}
	if addKey("rightshoulder", btnTR) {
		mapped |= 2
	}
	if mapped != 3 {
		if lh, ok := hat(1, 4); ok {
			rh, _ := hat(1, 2)
			add("leftshoulder", lh)
			add("rightshoulder", rh)
			mapped = 3
		}
	}
	if mapped&1 == 0 {
		if in, ok := axis(absHat1Y); ok {
			add("leftshoulder", in)
			mapped |= 1
		}
	}
	if mapped&2 == 0 {
		if in, ok := axis(absHat1X); ok {
			add("rightshoulder", in)
		}
	}

	// Prefer analog triggers, but settle for a digital hat or buttons. There are three conventions for analog triggers: the
	// specification's (ABS_HAT2Y and ABS_HAT2X), Android's (ABS_BRAKE and ABS_GAS) and the old Xbox and PlayStation one
	// (ABS_Z and ABS_RZ).
	mapped = 0
	for _, code := range []int{absHat2Y, absBrake, absZ} {
		if in, ok := axis(code); ok {
			add("lefttrigger", in)
			mapped |= 1
			break
		}
	}
	for _, code := range []int{absHat2X, absGas, absRZ} {
		if in, ok := axis(code); ok {
			add("righttrigger", in)
			mapped |= 2
			break
		}
	}
	if mapped != 3 {
		if lt, ok := hat(2, 4); ok {
			rt, _ := hat(2, 2)
			add("lefttrigger", lt)
			add("righttrigger", rt)
			mapped = 3
		}
	}
	if mapped&1 == 0 {
		addKey("lefttrigger", btnTL2)
	}
	if mapped&2 == 0 {
		addKey("righttrigger", btnTR2)
	}

	// Prefer digital D-pad buttons, but settle for a digital or analog hat.
	dpad := 0
	for i, p := range []struct {
		name string
		code int
	}{{"dpup", btnDpadUp}, {"dpdown", btnDpadDown}, {"dpleft", btnDpadLeft}, {"dpright", btnDpadRite}} {
		if addKey(p.name, p.code) {
			dpad |= 1 << i
		}
	}
	if dpad != 0xF {
		switch {
		case l.hatIndex[0] >= 0:
			for _, p := range []struct {
				name string
				mask int
			}{{"dpleft", 8}, {"dpright", 2}, {"dpup", 1}, {"dpdown", 4}} {
				in, _ := hat(0, p.mask)
				add(p.name, in)
			}
		case l.absMap[absHat0X] >= 0 && l.absMap[absHat0Y] >= 0:
			// An analog D-pad: each direction is the half of an axis.
			x, _ := axis(absHat0X)
			y, _ := axis(absHat0Y)
			add("dpleft", "-"+x)
			add("dpright", "+"+x)
			add("dpup", "-"+y)
			add("dpdown", "+"+y)
		case c.Driver == "xpad" && l.keyMap[btnTrigHap1] >= 0 && l.keyMap[btnTrigHap2] >= 0 && l.keyMap[btnTrigHap3] >= 0 && l.keyMap[btnTrigHap4] >= 0:
			// xpad sometimes reports the D-pad as BTN_TRIGGER_HAPPY1 to BTN_TRIGGER_HAPPY4.
			addKey("dpleft", btnTrigHap1)
			addKey("dpright", btnTrigHap2)
			addKey("dpup", btnTrigHap3)
			addKey("dpdown", btnTrigHap4)
		}
	}

	if x, ok := axis(absX); ok {
		if y, ok := axis(absY); ok {
			add("leftx", x)
			add("lefty", y)
		}
	}
	// The specification uses ABS_RX and ABS_RY, intended as the rotation of a joystick, as a second stick. Android's convention
	// uses ABS_Z and ABS_RZ.
	rx, okx := axis(absRX)
	ry, oky := axis(absRY)
	if okx && oky {
		add("rightx", rx)
		add("righty", ry)
	} else if z, ok := axis(absZ); ok {
		if rz, ok := axis(absRZ); ok {
			add("rightx", z)
			add("righty", rz)
		}
	}

	paddles := 0
	if addKey("paddle1", btnGripR) {
		paddles |= 2
	}
	if addKey("paddle2", btnGripL) {
		paddles |= 1
	}
	if addKey("paddle3", btnGripR2) {
		paddles |= 8
	}
	if addKey("paddle4", btnGripL2) {
		paddles |= 4
	}
	if paddles != 0xF && c.ID.Vendor == vendorMicrosoft &&
		l.keyMap[btnTrigHap5] >= 0 && l.keyMap[btnTrigHap6] >= 0 && l.keyMap[btnTrigHap7] >= 0 && l.keyMap[btnTrigHap8] >= 0 {
		// Older drivers give the paddles of the Xbox Elite pads as BTN_TRIGGER_HAPPY5 to BTN_TRIGGER_HAPPY8.
		s := b.String()
		for _, name := range []string{"paddle1", "paddle2", "paddle3", "paddle4"} {
			s = removeField(s, name)
		}
		b.Reset()
		b.WriteString(s)
		addKey("paddle1", btnTrigHap5)
		addKey("paddle2", btnTrigHap7)
		addKey("paddle3", btnTrigHap6)
		addKey("paddle4", btnTrigHap8)
	}
	// Xbox Series pads have the Share button as KEY_RECORD.
	addKey("misc1", keyRecord)
	return b.String()
}

// removeField removes "name:value," from a mapping text.
func removeField(text, name string) string {
	var out []string
	for _, f := range strings.Split(strings.TrimSuffix(text, ","), ",") {
		if f != "" && !strings.HasPrefix(f, name+":") {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ",") + ","
}
