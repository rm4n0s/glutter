package iokit

import "sort"

// The usage pages and usages of the elements that a pad is made of (the HID usage tables: Generic Desktop, Simulation Controls,
// Button and Consumer).
const (
	pageGenericDesktop = 0x01
	pageSimulation     = 0x02
	pageButton         = 0x09
	pageConsumer       = 0x0C

	usageJoystick            = 0x04
	usageGamePad             = 0x05
	usageMultiAxisController = 0x08

	usageX, usageY, usageZ    = 0x30, 0x31, 0x32
	usageRx, usageRy, usageRz = 0x33, 0x34, 0x35
	usageSlider, usageDial    = 0x36, 0x37
	usageWheel                = 0x38
	usageHatswitch            = 0x39
	usageStart, usageSelect   = 0x3D, 0x3E
	usageSystemMainMenu       = 0x85
	usageDPadUp               = 0x90
	usageDPadDown             = 0x91
	usageDPadRight            = 0x92
	usageDPadLeft             = 0x93

	usageRudder, usageThrottle = 0xBA, 0xBB
	usageAccelerator           = 0xC4
	usageBrake                 = 0xC5
)

// ElementType is the type of an element of a HID device (IOHIDElementType).
//
// Flutter: no counterpart
type ElementType uint32

// The types of elements that matter here.
const (
	// ElementInputMisc is an input with a value of any kind.
	//
	// Flutter: no counterpart
	ElementInputMisc ElementType = 1
	// ElementInputButton is a button.
	//
	// Flutter: no counterpart
	ElementInputButton ElementType = 2
	// ElementInputAxis is an axis.
	//
	// Flutter: no counterpart
	ElementInputAxis ElementType = 3
	// ElementCollection groups elements; the manager reads into it and gives the elements it holds.
	//
	// Flutter: no counterpart
	ElementCollection ElementType = 513
)

// Kind says what an element is to the pad.
//
// Flutter: no counterpart
type Kind uint8

// The kinds of elements.
const (
	// KindNone is an element that the pad does not use.
	//
	// Flutter: no counterpart
	KindNone Kind = iota
	// KindAxis is a raw axis.
	//
	// Flutter: no counterpart
	KindAxis
	// KindButton is a raw button.
	//
	// Flutter: no counterpart
	KindButton
	// KindHat is a raw hat.
	//
	// Flutter: no counterpart
	KindHat
)

// Classify says what an input element of the given type, usage page and usage is: a wheel, a throttle or a stick axis is an
// axis, a hat switch is a hat, and the buttons of the Button and Consumer pages (the pause button of some pads is a consumer
// control), the d-pad usages, Start, Select and the main menu are buttons.
//
// Flutter: no counterpart
func Classify(t ElementType, page, usage uint32) Kind {
	switch t {
	case ElementInputMisc, ElementInputButton, ElementInputAxis:
	default:
		return KindNone
	}
	switch page {
	case pageGenericDesktop:
		switch usage {
		case usageX, usageY, usageZ, usageRx, usageRy, usageRz, usageSlider, usageDial, usageWheel:
			return KindAxis
		case usageHatswitch:
			return KindHat
		case usageDPadUp, usageDPadDown, usageDPadRight, usageDPadLeft, usageStart, usageSelect, usageSystemMainMenu:
			return KindButton
		}
	case pageSimulation:
		switch usage {
		case usageRudder, usageThrottle, usageAccelerator, usageBrake:
			return KindAxis
		}
	case pageButton, pageConsumer:
		return KindButton
	}
	return KindNone
}

// RawElement is an element of a HID device as the manager lists it.
//
// Flutter: no counterpart
type RawElement struct {
	// Cookie names the element in its device.
	Cookie uint32
	Type   ElementType
	Page   uint32
	Usage  uint32
	// Min and Max are the logical minimum and maximum of the element.
	Min, Max int32
}

// Element is an element that the pad uses, with the range of the values that it has reported.
//
// Flutter: no counterpart
type Element struct {
	RawElement
	// minReport and maxReport start as Min and Max and grow with the values that the element reports: some pads report outside
	// of their logical range, and the axis is scaled by what it has done (SDL's calibration).
	minReport, maxReport int32
	// last is the last value that was given for the element, which a report that repeats it does not give again.
	last int
	has  bool
}

// Layout is the buttons, axes and hats of a device in the order of the raw indices that the app sees: each list is sorted by
// usage (elements with the same usage keep the order in which the device listed them).
//
// Flutter: no counterpart
type Layout struct {
	Axes, Buttons, Hats []*Element
	byCookie            map[uint32]position
}

// position is where an element is in a layout.
type position struct {
	kind  Kind
	index int
}

// BuildLayout makes the layout of a device from its elements, in the order in which the manager listed them. An element that
// the pad does not use is left out, and so is one whose cookie was seen already.
//
// Flutter: no counterpart
func BuildLayout(raw []RawElement) *Layout {
	l := &Layout{byCookie: map[uint32]position{}}
	seen := map[uint32]bool{}
	for _, r := range raw {
		k := Classify(r.Type, r.Page, r.Usage)
		if k == KindNone || seen[r.Cookie] {
			continue
		}
		seen[r.Cookie] = true
		e := &Element{RawElement: r, minReport: r.Min, maxReport: r.Max}
		switch k {
		case KindAxis:
			l.Axes = append(l.Axes, e)
		case KindButton:
			l.Buttons = append(l.Buttons, e)
		case KindHat:
			l.Hats = append(l.Hats, e)
		}
	}
	for _, list := range [][]*Element{l.Axes, l.Buttons, l.Hats} {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Usage < list[j].Usage })
	}
	for i, e := range l.Axes {
		l.byCookie[e.Cookie] = position{KindAxis, i}
	}
	for i, e := range l.Buttons {
		l.byCookie[e.Cookie] = position{KindButton, i}
	}
	for i, e := range l.Hats {
		l.byCookie[e.Cookie] = position{KindHat, i}
	}
	return l
}

// Count returns the number of elements of the layout.
//
// Flutter: no counterpart
func (l *Layout) Count() int { return len(l.Axes) + len(l.Buttons) + len(l.Hats) }

// Hat masks, as platform.PadHat gives them.
const (
	hatUp    = 1
	hatRight = 2
	hatDown  = 4
	hatLeft  = 8
)

// AxisValue scales the value of an axis from what the device reported to the range of -32768 to 32767. The range of what the
// element has reported so far grows with the value, so a pad that goes beyond its logical limits still fills the range. An
// element whose range is a single value is passed as it is.
//
// Flutter: no counterpart
func (e *Element) AxisValue(v int32) int16 {
	if v < e.minReport {
		e.minReport = v
	}
	if v > e.maxReport {
		e.maxReport = v
	}
	read := int64(e.maxReport) - int64(e.minReport)
	if read == 0 {
		return clamp16(int64(v))
	}
	const lo, hi = -32768, 32767
	return clamp16((int64(v)-int64(e.minReport))*(hi-lo)/read + lo)
}

// ButtonValue says whether the button is down: any value that is not zero is.
//
// Flutter: no counterpart
func (e *Element) ButtonValue(v int32) int {
	if v != 0 {
		return 1
	}
	return 0
}

// HatValue decodes a hat switch into a mask of the directions. A hat with four positions is read as one with eight (its steps
// are the even ones), a hat with eight positions is read as it is, and the value of one with another number of positions, or
// outside of its range (the "null state" of a hat that is released), is centered.
//
// Flutter: no counterpart
func (e *Element) HatValue(v int32) int {
	span := e.Max - e.Min + 1
	v -= e.Min
	switch span {
	case 4:
		v *= 2
	case 8:
	default:
		return 0
	}
	switch v {
	case 0:
		return hatUp
	case 1:
		return hatUp | hatRight
	case 2:
		return hatRight
	case 3:
		return hatRight | hatDown
	case 4:
		return hatDown
	case 5:
		return hatDown | hatLeft
	case 6:
		return hatLeft
	case 7:
		return hatLeft | hatUp
	}
	return 0
}

// clamp16 limits a value to the range of an int16.
func clamp16(v int64) int16 {
	if v < -32768 {
		return -32768
	}
	if v > 32767 {
		return 32767
	}
	return int16(v)
}

// IsPad says whether a device with the primary usage page and usage is a pad: a joystick, a game pad or a multi-axis controller
// of the Generic Desktop page. Keyboards, mice and the rest are not.
//
// Flutter: no counterpart
func IsPad(page, usage uint32) bool {
	if page != pageGenericDesktop {
		return false
	}
	switch usage {
	case usageJoystick, usageGamePad, usageMultiAxisController:
		return true
	}
	return false
}

// MatchingPages are the (usage page, usage) pairs that the manager is asked to report, the three kinds of pad.
//
// Flutter: no counterpart
func MatchingPages() [][2]uint32 {
	return [][2]uint32{
		{pageGenericDesktop, usageJoystick},
		{pageGenericDesktop, usageGamePad},
		{pageGenericDesktop, usageMultiAxisController},
	}
}
