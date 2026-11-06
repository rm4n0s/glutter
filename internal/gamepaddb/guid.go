// Package gamepaddb is the model of a pad that SPEC §28.1 ports from SDL 3 (zlib; see third_party/sdl/LICENSE): the identity of
// a pad (its GUID, its name and its kind), the mapping format that says which button or axis of a pad is which button or axis
// of the standard layout, SDL's built-in table of mappings (mappings_gen.go, made by glutter-gen gamepaddb), and the two layers
// that turn the raw reports of a pad into the buttons and axes of the layout. It is plain Go with no OS tags and no
// dependency on the rest of Glutter.
//
// The files that are ported start with a comment that names the SDL 3 file they come from; they are translated to Go and
// changed.
//
// Flutter: no counterpart
//
//go:generate go run ../../cmd/glutter-gen gamepaddb -reference ../../reference/sdl -out .
package gamepaddb

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
)

// This file is translated to Go and changed from src/joystick/SDL_joystick.c, src/SDL_guid.c and the CRC of src/stdlib of SDL 3
// (zlib license, third_party/sdl/LICENSE).

// The buses of a pad, as the first 16 bits of its GUID say (the values of Linux's BUS_ constants).
const (
	// BusUSB is a pad on USB.
	//
	// Flutter: no counterpart
	BusUSB uint16 = 0x03
	// BusBluetooth is a pad on Bluetooth.
	//
	// Flutter: no counterpart
	BusBluetooth uint16 = 0x05
	// BusVirtual is a pad that a program makes (SDL's number; it is not Linux's BUS_VIRTUAL).
	//
	// Flutter: no counterpart
	BusVirtual uint16 = 0xFF
	// BusUnknown is a pad whose bus is not known.
	//
	// Flutter: no counterpart
	BusUnknown uint16 = 0x00
)

// GUID identifies a model of pad: 16 bytes whose layout is
//
//	16 bits bus, 16 bits CRC-16 of the name (zero when unknown), 16 bits vendor, 16 bits zero, 16 bits product, 16 bits
//	zero, 16 bits version, 8 bits driver signature, 8 bits driver data
//
// all little-endian. A pad without a vendor has the first 4 bytes of this form and then up to 12 bytes of its name. A GUID
// written as 32 hexadecimal digits is how the community's mapping files name a pad.
//
// Flutter: no counterpart
type GUID [16]byte

// ParseGUID reads the 32 hexadecimal digits of a GUID. A string that is too short is padded with zeros, one that is too long
// loses the rest, and a character that is not a digit counts as 0, as in the mapping files that exist.
//
// Flutter: no counterpart
func ParseGUID(s string) GUID {
	var g GUID
	s = s[:len(s)&^1]
	for i := 0; i+1 < len(s) && i/2 < len(g); i += 2 {
		g[i/2] = nibble(s[i])<<4 | nibble(s[i+1])
	}
	return g
}

func nibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	return 0
}

// String writes the GUID as 32 lower-case hexadecimal digits.
//
// Flutter: no counterpart
func (g GUID) String() string { return hex.EncodeToString(g[:]) }

// NewGUID makes the GUID of a pad from what its bus says about it. vendorName and productName are what the bus reports (either
// may be empty); the CRC of the GUID is made of them. driverSignature is 'h' for a pad that Glutter's own HID driver reads, 'x'
// for XInput and so on, and 0 for the evdev pads of Linux.
//
// Flutter: no counterpart
func NewGUID(bus, vendor, product, version uint16, vendorName, productName string, driverSignature, driverData uint8) GUID {
	var g GUID
	var crc uint16
	switch {
	case vendorName != "" && productName != "":
		crc = crc16(crc, vendorName)
		crc = crc16(crc, " ")
		crc = crc16(crc, productName)
	case productName != "":
		crc = crc16(crc, productName)
	}
	binary.LittleEndian.PutUint16(g[0:], bus)
	binary.LittleEndian.PutUint16(g[2:], crc)
	if vendor != 0 {
		binary.LittleEndian.PutUint16(g[4:], vendor)
		binary.LittleEndian.PutUint16(g[8:], product)
		binary.LittleEndian.PutUint16(g[12:], version)
		g[14] = driverSignature
		g[15] = driverData
		return g
	}
	available := len(g) - 4
	if driverSignature != 0 {
		available -= 2
		g[14] = driverSignature
		g[15] = driverData
	}
	n := copy(g[4:4+available], productName)
	// The name is a C string: it is cut one byte short when it fills the space, so that a zero ends it.
	if n == available && available > 0 {
		g[4+available-1] = 0
	}
	return g
}

// NewGUIDForName makes the GUID of a pad that only a name is known of.
//
// Flutter: no counterpart
func NewGUIDForName(name string) GUID { return NewGUID(BusUnknown, 0, 0, 0, "", name, 0, 0) }

// GUIDInfo is what a GUID says about the pad.
//
// Flutter: no counterpart
type GUIDInfo struct {
	Vendor, Product, Version, CRC uint16
}

// Info reads the vendor, product, version and CRC out of the GUID. A GUID that does not have the standard form has zeros for
// the vendor, product and version (and its CRC when its bus is one of the known ones).
//
// Flutter: no counterpart
func (g GUID) Info() GUIDInfo {
	bus := binary.LittleEndian.Uint16(g[0:])
	known := bus < ' ' || bus == BusVirtual
	switch {
	case known && binary.LittleEndian.Uint16(g[6:]) == 0 && binary.LittleEndian.Uint16(g[10:]) == 0:
		return GUIDInfo{
			CRC:     binary.LittleEndian.Uint16(g[2:]),
			Vendor:  binary.LittleEndian.Uint16(g[4:]),
			Product: binary.LittleEndian.Uint16(g[8:]),
			Version: binary.LittleEndian.Uint16(g[12:]),
		}
	case known:
		return GUIDInfo{CRC: binary.LittleEndian.Uint16(g[2:])}
	}
	return GUIDInfo{}
}

// WithCRC returns the GUID with the CRC set to crc.
//
// Flutter: no counterpart
func (g GUID) WithCRC(crc uint16) GUID {
	binary.LittleEndian.PutUint16(g[2:], crc)
	return g
}

// WithVersion returns the GUID with the version set to version.
//
// Flutter: no counterpart
func (g GUID) WithVersion(version uint16) GUID {
	binary.LittleEndian.PutUint16(g[12:], version)
	return g
}

// usesVersion says whether the version of the GUID tells models apart: only for a pad that has a vendor and a product.
func (g GUID) usesVersion() bool {
	i := g.Info()
	return i.Vendor != 0 && i.Product != 0
}

// isXInput says whether the GUID is that of an XInput pad (driver signature 'x' in the standard form).
func (g GUID) isXInput() bool {
	return g[14] == 'x'
}

// crc16 continues a CRC-16 (polynomial 0xA001, reflected, start value 0) over s, as SDL does for the name in the GUID.
func crc16(crc uint16, s string) uint16 {
	for i := 0; i < len(s); i++ {
		crc = crc16Table[byte(crc)^s[i]] ^ crc>>8
	}
	return crc
}

var crc16Table = func() (t [256]uint16) {
	for i := range t {
		var crc uint16
		r := byte(i)
		for j := 0; j < 8; j++ {
			if (uint16(r)^crc)&1 != 0 {
				crc = 0xA001 ^ crc>>1
			} else {
				crc >>= 1
			}
			r >>= 1
		}
		t[i] = crc
	}
	return t
}()

// DeviceName makes the name of a pad out of what its bus reports, as SDL does: the vendor and the product name joined, a name
// from the vendor and product numbers when the bus gave none, the usual manufacturer names shortened, and the spaces tidied.
//
// Flutter: no counterpart
func DeviceName(vendor, product uint16, vendorName, productName string) string {
	if n := ControllerName(vendor, product); n != "" {
		return n
	}
	return deviceName(vendor, product, vendorName, productName, "Controller")
}

var manufacturerReplacements = []struct{ prefix, replacement string }{
	{"(Standard system devices) ", ""},
	{"8BitDo Tech Ltd", "8BitDo"},
	{"ASTRO Gaming", "ASTRO"},
	{"Bensussen Deutsch & Associates,Inc.(BDA)", "BDA"},
	{"Guangzhou Chicken Run Network Technology Co., Ltd.", "GameSir"},
	{"HORI CO.,LTD.", "HORI"},
	{"HORI CO.,LTD", "HORI"},
	{"Mad Catz Inc.", "Mad Catz"},
	{"Nintendo Co., Ltd.", "Nintendo"},
	{"NVIDIA Corporation ", ""},
	{"Performance Designed Products", "PDP"},
	{"QANBA USA, LLC", "Qanba"},
	{"QANBA USA,LLC", "Qanba"},
	{"Unknown ", ""},
}

func deviceName(vendor, product uint16, vendorName, productName, defaultName string) string {
	vendorName = strings.TrimLeft(vendorName, " ")
	productName = strings.TrimLeft(productName, " ")
	var name string
	switch {
	case vendorName != "" && productName != "":
		name = vendorName + " " + productName
	case productName != "":
		name = productName
	case vendor != 0 || product != 0:
		switch KindOf(vendor, product, "", true) {
		case KindXbox360:
			name = "Xbox 360 Controller"
		case KindXboxOne:
			name = "Xbox One Controller"
		case KindPS3:
			name = "PS3 Controller"
		case KindPS4:
			name = "PS4 Controller"
		case KindPS5:
			name = "DualSense Wireless Controller"
		case KindSwitchPro:
			name = "Nintendo Switch Pro Controller"
		case KindSteam:
			name = "Steam Controller"
		default:
			name = "0x" + hex4(vendor) + "/0x" + hex4(product)
		}
	default:
		name = defaultName
	}
	name = strings.TrimRight(name, " ")
	for strings.Contains(name, "  ") {
		name = strings.ReplaceAll(name, "  ", " ")
	}
	for _, r := range manufacturerReplacements {
		if len(name) >= len(r.prefix) && strings.EqualFold(name[:len(r.prefix)], r.prefix) {
			name = r.replacement + name[len(r.prefix):]
			break
		}
	}
	return name
}

func hex4(v uint16) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>12], digits[v>>8&15], digits[v>>4&15], digits[v&15]})
}
