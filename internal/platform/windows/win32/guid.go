package win32

import (
	"strings"

	"github.com/rm4n0s/errors"
)

// GUID is the Go mirror of the Win32 GUID (an interface ID, IID, or a class ID, CLSID): 16 bytes
// whose first three fields are numbers and whose last is a byte array.
//
// Flutter: no counterpart
type GUID struct {
	Data1 uint32  `c:"Data1"`
	Data2 uint16  `c:"Data2"`
	Data3 uint16  `c:"Data3"`
	Data4 [8]byte `c:"Data4"`
}

// IIDUnknown is the interface ID of IUnknown, which every COM interface starts with.
//
// Flutter: no counterpart
var IIDUnknown = MustParseGUID("{00000000-0000-0000-C000-000000000046}")

// ParseGUID reads the text form of a GUID: 8-4-4-4-12 hexadecimal digits, with or without the
// braces, in either case. It fails with GUIDInvalid, with the text as metadata, for anything else.
//
// Flutter: no counterpart
func ParseGUID(s string) (GUID, error) {
	text := s
	if strings.HasPrefix(s, "{") || strings.HasSuffix(s, "}") {
		if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
			return GUID{}, errors.New(TagGUIDInvalid, "the braces of a GUID come in a pair", "text", s)
		}
		text = s[1 : len(s)-1]
	}
	if len(text) != 36 {
		return GUID{}, errors.New(TagGUIDInvalid, "a GUID has the form 8-4-4-4-12 hexadecimal digits", "text", s)
	}
	for i := range len(text) {
		switch i {
		case 8, 13, 18, 23:
			if text[i] != '-' {
				return GUID{}, errors.New(TagGUIDInvalid, "a GUID has dashes after the 8th, 12th, 16th and 20th digit", "text", s)
			}
		default:
			if hexValue(text[i]) < 0 {
				return GUID{}, errors.New(TagGUIDInvalid, "a GUID has only hexadecimal digits", "text", s, "position", i)
			}
		}
	}
	hex := func(from, n int) uint64 {
		var v uint64
		for _, c := range []byte(text[from : from+n]) {
			v = v<<4 | uint64(hexValue(c))
		}
		return v
	}
	g := GUID{Data1: uint32(hex(0, 8)), Data2: uint16(hex(9, 4)), Data3: uint16(hex(14, 4))}
	g.Data4[0] = byte(hex(19, 2))
	g.Data4[1] = byte(hex(21, 2))
	for i := range 6 {
		g.Data4[2+i] = byte(hex(24+2*i, 2))
	}
	return g, nil
}

// MustParseGUID is ParseGUID for the GUIDs written in the source, which are known to be right: it
// panics with the error of ParseGUID on a mistake, a programmer error found at start.
//
// Flutter: no counterpart
func MustParseGUID(s string) GUID {
	g, err := ParseGUID(s)
	if err != nil {
		panic(err)
	}
	return g
}

// String is the text form with braces and upper-case digits, such as
// "{00000000-0000-0000-C000-000000000046}".
//
// Flutter: no counterpart
func (g GUID) String() string {
	const digits = "0123456789ABCDEF"
	var b strings.Builder
	put := func(v uint64, n int) {
		for shift := 4 * (n - 1); shift >= 0; shift -= 4 {
			b.WriteByte(digits[v>>uint(shift)&0xF])
		}
	}
	b.WriteByte('{')
	put(uint64(g.Data1), 8)
	b.WriteByte('-')
	put(uint64(g.Data2), 4)
	b.WriteByte('-')
	put(uint64(g.Data3), 4)
	b.WriteByte('-')
	put(uint64(g.Data4[0]), 2)
	put(uint64(g.Data4[1]), 2)
	b.WriteByte('-')
	for _, c := range g.Data4[2:] {
		put(uint64(c), 2)
	}
	b.WriteByte('}')
	return b.String()
}

// hexValue is the value of a hexadecimal digit, or -1.
func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// TagGUIDInvalid is the tag of the error for a text that is not a GUID (rm4n0s/errors, R5).
const TagGUIDInvalid = "GUIDInvalid"
