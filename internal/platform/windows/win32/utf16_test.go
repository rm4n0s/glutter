package win32

import (
	"slices"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestUTF16RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []uint16 // with the terminating NUL
		back string   // the text after the round trip, when it is not the same
	}{
		{"empty", "", []uint16{0}, ""},
		{"ASCII", "Glutter", []uint16{'G', 'l', 'u', 't', 't', 'e', 'r', 0}, ""},
		{"BMP", "héllo €", []uint16{'h', 0xE9, 'l', 'l', 'o', ' ', 0x20AC, 0}, ""},
		{"a surrogate pair", "a😀b", []uint16{'a', 0xD83D, 0xDE00, 'b', 0}, ""},
		{"the last code point", "\U0010FFFF", []uint16{0xDBFF, 0xDFFF, 0}, ""},
		{"invalid UTF-8 becomes U+FFFD", "a\xffb", []uint16{'a', 0xFFFD, 'b', 0}, "a\uFFFDb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UTF16FromString(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("UTF16FromString = %x, want %x", got, tt.want)
			}
			back := UTF16ToString(got)
			if want := tt.text; !slices.Contains([]string{want, "a�b"}, back) {
				t.Errorf("round trip = %q, want %q", back, want)
			}
			p, err := UTF16PtrFromString(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if got := UTF16PtrToString(p); got != back {
				t.Errorf("UTF16PtrToString = %q, want %q", got, back)
			}
		})
	}
}

func TestUTF16EmbeddedNUL(t *testing.T) {
	for _, call := range []struct {
		name string
		f    func() error
		path string
	}{
		{"UTF16FromString", func() error { _, err := UTF16FromString("ab\x00cd"); return err }, "UTF16FromString.UTF16EmbeddedNUL"},
		{"UTF16PtrFromString", func() error { _, err := UTF16PtrFromString("ab\x00cd"); return err }, "UTF16PtrFromString->UTF16FromString.UTF16EmbeddedNUL"},
		{"CopyUTF16", func() error { _, err := CopyUTF16(make([]uint16, 8), "ab\x00cd"); return err }, "CopyUTF16->UTF16FromString.UTF16EmbeddedNUL"},
	} {
		t.Run(call.name, func(t *testing.T) {
			err := call.f()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != TagUTF16EmbeddedNUL || !appErr.HasRoute(call.path) {
				t.Errorf("got %s via %s, want UTF16EmbeddedNUL via %s", appErr.Tag, appErr.Route(), call.path)
			}
			if appErr.Metadata["index"] != 2 {
				t.Errorf("metadata = %v, want the index 2", appErr.Metadata)
			}
		})
	}
}

func TestUTF16ToStringStopsAtTheFirstNUL(t *testing.T) {
	tests := []struct {
		name string
		in   []uint16
		want string
	}{
		{"an embedded NUL", []uint16{'a', 0, 'b'}, "a"},
		{"no NUL", []uint16{'a', 'b'}, "ab"},
		{"a fixed buffer", append([]uint16{'h', 'i'}, make([]uint16, 6)...), "hi"},
		{"a lone high surrogate", []uint16{'a', 0xD83D, 'b'}, "a�b"},
		{"a lone low surrogate at the end", []uint16{'a', 0xDE00}, "a�"},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UTF16ToString(tt.in); got != tt.want {
				t.Errorf("UTF16ToString = %q, want %q", got, tt.want)
			}
		})
	}
	if got := UTF16PtrToString(nil); got != "" {
		t.Errorf("UTF16PtrToString(nil) = %q", got)
	}
}

func TestCopyUTF16(t *testing.T) {
	tests := []struct {
		name          string
		size          int
		text          string
		want          string
		wantTruncated bool
	}{
		{"fits with room", 8, "tip", "tip", false},
		{"fits exactly with the NUL", 4, "tip", "tip", false},
		{"one unit too long", 3, "tip", "ti", true},
		{"a string longer than the buffer", 4, "a long tooltip", "a l", true},
		{"a surrogate pair that would be cut", 3, "a😀", "a", true},
		{"a surrogate pair that fits", 4, "a😀", "a😀", false},
		{"an empty buffer", 0, "x", "", true},
		{"an empty buffer and an empty string", 0, "", "", false},
		{"a buffer of one", 1, "x", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]uint16, tt.size)
			for i := range dst {
				dst[i] = 0x7777 // so that the NUL padding is seen
			}
			truncated, err := CopyUTF16(dst, tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTruncated)
			}
			if tt.size > 0 {
				if got := UTF16ToString(dst); got != tt.want {
					t.Errorf("buffer = %q, want %q", got, tt.want)
				}
				if dst[tt.size-1] != 0 {
					t.Errorf("the buffer %x does not end with a NUL", dst)
				}
			}
		})
	}
}
