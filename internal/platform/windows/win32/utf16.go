package win32

import (
	"unicode/utf16"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// UTF16FromString converts a Go string to UTF-16 with a terminating NUL, the form that the "W"
// functions of Windows take. A byte sequence that is not UTF-8 becomes U+FFFD, as Windows treats
// invalid text. A string with a NUL inside would end early in C, so it fails with
// UTF16EmbeddedNUL (metadata "text" and "index", the byte offset of the NUL).
//
// Flutter: no counterpart
func UTF16FromString(s string) ([]uint16, error) {
	for i := range len(s) {
		if s[i] == 0 {
			return nil, errors.New(TagUTF16EmbeddedNUL, "a string that goes to Windows has a NUL inside", "text", s, "index", i)
		}
	}
	out := make([]uint16, 0, len(s)+1)
	for _, r := range s {
		if r >= 0x10000 {
			r1, r2 := utf16.EncodeRune(r)
			out = append(out, uint16(r1), uint16(r2))
			continue
		}
		out = append(out, uint16(r))
	}
	return append(out, 0), nil
}

// UTF16PtrFromString is UTF16FromString for a function that takes a pointer: it returns the
// address of the first unit. The slice lives as long as the pointer is reachable. It fails with
// UTF16EmbeddedNUL like UTF16FromString.
//
// Flutter: no counterpart
func UTF16PtrFromString(s string) (*uint16, error) {
	u, err := UTF16FromString(s)
	if err != nil {
		return nil, err
	}
	return &u[0], nil
}

// UTF16ToString converts UTF-16 to a Go string, up to the first NUL (or to the end when there is
// none), as a C string would end. A lone surrogate becomes U+FFFD. It is how a fixed
// [N]uint16 member of a structure (a file name, a tooltip) is read: UTF16ToString(buf[:]).
//
// Flutter: no counterpart
func UTF16ToString(s []uint16) string {
	for i, c := range s {
		if c == 0 {
			s = s[:i]
			break
		}
	}
	return string(utf16.Decode(s))
}

// UTF16PtrToString reads the NUL-terminated UTF-16 string at p, which Windows owns, and returns
// a copy as a Go string. A nil p is the empty string.
//
// Flutter: no counterpart
func UTF16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(p), 2*n)) != 0 {
		n++
	}
	return UTF16ToString(unsafe.Slice(p, n))
}

// CopyUTF16 writes s into the fixed buffer dst as UTF-16 with a terminating NUL, the way a text
// is put into a [N]uint16 member (the tooltip of a tray icon). When s does not fit it is cut
// before the first character that does not fit, never in the middle of a surrogate pair, and
// truncated is true. An empty dst takes nothing and reports truncated for any s that is not
// empty. It fails with UTF16EmbeddedNUL like UTF16FromString.
//
// Flutter: no counterpart
func CopyUTF16(dst []uint16, s string) (truncated bool, err error) {
	u, err := UTF16FromString(s)
	if err != nil {
		return false, err
	}
	u = u[:len(u)-1] // without the NUL
	if len(dst) == 0 {
		return len(u) > 0, nil
	}
	n := len(u)
	if n > len(dst)-1 {
		n = len(dst) - 1
		if n > 0 && utf16.IsSurrogate(rune(u[n-1])) && u[n-1] < 0xDC00 {
			n-- // the high half of a pair whose low half does not fit
		}
		truncated = true
	}
	copy(dst, u[:n])
	for i := n; i < len(dst); i++ {
		dst[i] = 0
	}
	return truncated, nil
}

// TagUTF16EmbeddedNUL is the tag of the error for a string with a NUL inside that goes to Windows
// as a C string (rm4n0s/errors, R5).
const TagUTF16EmbeddedNUL = "UTF16EmbeddedNUL"
