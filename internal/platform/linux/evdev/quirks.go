package evdev

// This file is translated to Go and changed from src/joystick/linux/SDL_sysjoystick.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): FixupDeviceInfoForMapping and the list of the mice that look like pads.

// fixupForMapping changes the version of two pads so that the mappings match, as SDL does.
func fixupForMapping(c *caps) {
	if c.ID.Vendor == 0x045e && c.ID.Product == 0x0b05 && c.ID.Version == 0x0903 && c.Key.has(0x2c0) {
		// An Xbox One Elite Series 2 pad: the first version of the firmware duplicated all the inputs.
		c.ID.Version = 0x0902
	}
	// The Atari VCS pads have the firmware version in their version, and their mapping stays the same.
	if c.ID.Vendor == 0x3250 && (c.ID.Product == 0x1001 || c.ID.Product == 0x1002) {
		c.ID.Version = 0
	}
}

// ignoreDevice says whether a joystick-like device is to be left alone.
func ignoreDevice(c *caps) bool {
	// SDL ignores the mice of ASUS ROG that have the axes and buttons of a pad.
	switch uint32(c.ID.Vendor)<<16 | uint32(c.ID.Product) {
	case 0x0b05_18e3, 0x0b05_18e5, 0x0b05_1906, 0x0b05_1958, 0x0b05_1a18, 0x0b05_1a1a, 0x0b05_1a1c: // ROG Chakram and Pugio mice
		return true
	}
	return false
}
