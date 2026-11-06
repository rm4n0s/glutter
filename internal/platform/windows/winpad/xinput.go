package winpad

import (
	"fmt"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The code of an XInput pad: how a slot is named, identified and read. Translated to Go and changed from
// src/joystick/windows/SDL_xinputjoystick.c of SDL 3.

// XInputAPI is what the service asks of XInput; *win32.XInput is the real one.
//
// Flutter: no counterpart
type XInputAPI interface {
	GetState(user uint32, st *win32.XInputState) uint32
	SetState(user uint32, v *win32.XInputVibration) uint32
	GetCapabilities(user, flags uint32, c *win32.XInputCapabilities) uint32
	GetCapabilitiesEx(user uint32) (win32.XInputCapabilitiesEx, bool)
	GetBatteryInformation(user uint32) (win32.XInputBatteryInformation, bool)
}

// The vendor and product of a pad that XInput cannot say more about: the Xbox 360 controller (USB_VENDOR_MICROSOFT and
// USB_PRODUCT_XBOX360_XUSB_CONTROLLER).
const (
	vendorMicrosoft = 0x045e
	productXbox360  = 0x028e
)

// The subtypes of XInput devices (XINPUT_DEVSUBTYPE_*).
const (
	subTypeGamepad        = 0x01
	subTypeWheel          = 0x02
	subTypeArcadeStick    = 0x03
	subTypeFlightStick    = 0x04
	subTypeDancePad       = 0x05
	subTypeGuitar         = 0x06
	subTypeGuitarAlt      = 0x07
	subTypeDrumKit        = 0x08
	subTypeGuitarBass     = 0x0B
	subTypeArcadePad      = 0x13
	xinputDriverSignature = 'x'
)

// xinputName is the name that XInput gives the pad of a slot, from its subtype.
func xinputName(user uint32, subType uint8) string {
	kind := "Device"
	switch subType {
	case subTypeGamepad:
		kind = "Controller"
	case subTypeWheel:
		kind = "Wheel"
	case subTypeArcadeStick:
		kind = "ArcadeStick"
	case subTypeFlightStick:
		kind = "FlightStick"
	case subTypeDancePad:
		kind = "DancePad"
	case subTypeGuitar, subTypeGuitarAlt, subTypeGuitarBass:
		kind = "Guitar"
	case subTypeDrumKit:
		kind = "DrumKit"
	case subTypeArcadePad:
		kind = "ArcadePad"
	}
	return fmt.Sprintf("XInput %s #%d", kind, user+1)
}

// xinputIdentity finds the vendor, product and version of the pad of a slot: from the undocumented capabilities when the
// library has them, else those of the Xbox 360 controller.
func xinputIdentity(x XInputAPI, user uint32, caps *win32.XInputCapabilities) (vendor, product, version uint16) {
	ex, ok := x.GetCapabilitiesEx(user)
	if !ok {
		return vendorMicrosoft, productXbox360, 0
	}
	vendor, product, version = ex.VendorID, ex.ProductID, ex.VersionNumber
	// A wireless Xbox 360 controller says no product.
	if product == 0 && caps.Flags&win32.XInputCapsWireless != 0 {
		vendor, product = vendorMicrosoft, productXbox360
	}
	return vendor, product, version
}

// xinputGUID is the GUID of an XInput pad: the USB bus, the vendor, product and version, the name, the driver 'x' and the
// subtype.
func xinputGUID(vendor, product, version uint16, name string, subType uint8) [16]byte {
	return gamepaddb.NewGUID(gamepaddb.BusUSB, vendor, product, version, "", name, xinputDriverSignature, subType)
}

// xinputBattery is the charge of a pad from 0 to 100, as SDL guesses it from the level (empty 10, low 40, medium 70, full 100),
// or -1 for a wired pad and one whose battery is not known.
func xinputBattery(b win32.XInputBatteryInformation, ok bool) int {
	if !ok {
		return -1
	}
	switch b.BatteryType {
	case win32.XInputBatteryTypeDisconnected, win32.XInputBatteryTypeUnknown:
		return -1
	}
	switch b.BatteryLevel {
	case win32.XInputBatteryLevelEmpty:
		return 10
	case win32.XInputBatteryLevelLow:
		return 40
	case win32.XInputBatteryLevelMedium:
		return 70
	}
	return 100
}

// xinputSnapshot is the state of an XInput pad as the standard layout: the buttons by their bits, the sticks with the vertical
// axes turned around (XInput's are positive upwards, the layout's downwards) and the triggers from 0 to 32767.
func xinputSnapshot(g win32.XInputGamepad) padstd.Snapshot {
	var s padstd.Snapshot
	for _, b := range []struct {
		mask   uint16
		button gamepaddb.Button
	}{
		{win32.XInputGamepadA, gamepaddb.South}, {win32.XInputGamepadB, gamepaddb.East},
		{win32.XInputGamepadX, gamepaddb.West}, {win32.XInputGamepadY, gamepaddb.North},
		{win32.XInputGamepadBack, gamepaddb.Back}, {win32.XInputGamepadGuide, gamepaddb.Guide},
		{win32.XInputGamepadStart, gamepaddb.Start}, {win32.XInputGamepadLeftThumb, gamepaddb.LeftStick},
		{win32.XInputGamepadRightThumb, gamepaddb.RightStick}, {win32.XInputGamepadLeftShoulder, gamepaddb.LeftShoulder},
		{win32.XInputGamepadRightShoulder, gamepaddb.RightShoulder}, {win32.XInputGamepadDPadUp, gamepaddb.DPadUp},
		{win32.XInputGamepadDPadDown, gamepaddb.DPadDown}, {win32.XInputGamepadDPadLeft, gamepaddb.DPadLeft},
		{win32.XInputGamepadDPadRight, gamepaddb.DPadRight},
	} {
		s.Buttons[b.button] = g.Buttons&b.mask != 0
	}
	s.Axes[gamepaddb.LeftX] = g.ThumbLX
	s.Axes[gamepaddb.LeftY] = padstd.Flip(g.ThumbLY)
	s.Axes[gamepaddb.RightX] = g.ThumbRX
	s.Axes[gamepaddb.RightY] = padstd.Flip(g.ThumbRY)
	s.Axes[gamepaddb.LeftTrigger] = padstd.Trigger8(g.LeftTrigger)
	s.Axes[gamepaddb.RightTrigger] = padstd.Trigger8(g.RightTrigger)
	return s
}
