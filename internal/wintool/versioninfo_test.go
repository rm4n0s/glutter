package wintool

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/rm4n0s/errors"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in   string
		want Version
		bad  bool
	}{
		{"", DefaultVersion, false},
		{"2", Version{2, 0, 0, 0}, false},
		{"1.2", Version{1, 2, 0, 0}, false},
		{"1.2.3", Version{1, 2, 3, 0}, false},
		{" 1.2.3.4 ", Version{1, 2, 3, 4}, false},
		{"65535.0.0.1", Version{65535, 0, 0, 1}, false},
		{"1.2.3.4.5", Version{}, true},
		{"65536", Version{}, true},
		{"1.x", Version{}, true},
		{"-1", Version{}, true},
		{"1..2", Version{}, true},
	}
	for _, tt := range tests {
		got, err := ParseVersion(tt.in)
		if tt.bad {
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("ParseVersion.WindowsResourceInvalid") || appErr.Metadata["version"] == nil {
				t.Errorf("ParseVersion(%q) = %v, want WindowsResourceInvalid with the version", tt.in, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParseVersion(%q) = %v, %v, want %v", tt.in, got, err, tt.want)
		}
	}
	if s := (Version{1, 2, 3, 4}).String(); s != "1.2.3.4" {
		t.Errorf("String = %q", s)
	}
}

// node is a block of a VS_VERSIONINFO tree.
type node struct {
	key      string
	typ      uint16
	valueLen int
	value    []byte
	children []node
}

// parseNode reads the block at the start of b and returns it with the number of bytes it takes,
// padding included.
func parseNode(t *testing.T, b []byte) (node, int) {
	t.Helper()
	le := binary.LittleEndian
	if len(b) < 6 {
		t.Fatalf("a block of %d bytes", len(b))
	}
	length, valueLen, typ := int(le.Uint16(b[0:])), int(le.Uint16(b[2:])), le.Uint16(b[4:])
	if length > len(b) {
		t.Fatalf("the block says %d bytes, %d are left", length, len(b))
	}
	var key []uint16
	p := 6
	for ; le.Uint16(b[p:]) != 0; p += 2 {
		key = append(key, le.Uint16(b[p:]))
	}
	p += 2
	p = (p + 3) &^ 3
	n := node{key: string(utf16.Decode(key)), typ: typ, valueLen: valueLen}
	vbytes := valueLen
	if typ == 1 {
		vbytes = valueLen * 2
	}
	n.value = b[p : p+vbytes]
	p = (p + vbytes + 3) &^ 3
	for p < length {
		c, used := parseNode(t, b[p:length])
		n.children = append(n.children, c)
		p += used
	}
	return n, (length + 3) &^ 3
}

func TestVersionInfoEncode(t *testing.T) {
	vi := VersionInfo{Version: Version{1, 2, 3, 4}, Name: "Counter ü", AppID: "org.glutter.counter", FileName: "counter.exe"}
	root, used := parseNode(t, vi.Encode())
	if root.key != "VS_VERSION_INFO" || root.typ != 0 || root.valueLen != 52 {
		t.Fatalf("root = %q type %d value %d", root.key, root.typ, root.valueLen)
	}
	if used != len(vi.Encode()) {
		t.Errorf("the block takes %d bytes of %d", used, len(vi.Encode()))
	}
	le := binary.LittleEndian
	fixed := func(i int) uint32 { return le.Uint32(root.value[4*i:]) }
	want := []uint32{0xFEEF04BD, 0x00010000, 0x00010002, 0x00030004, 0x00010002, 0x00030004, 0x3F, 0, 0x00040004, 1, 0, 0, 0}
	for i, w := range want {
		if fixed(i) != w {
			t.Errorf("VS_FIXEDFILEINFO[%d] = %#x, want %#x", i, fixed(i), w)
		}
	}
	if len(root.children) != 2 || root.children[0].key != "StringFileInfo" || root.children[1].key != "VarFileInfo" {
		t.Fatalf("children = %+v", root.children)
	}
	table := root.children[0].children
	if len(table) != 1 || table[0].key != "040904B0" {
		t.Fatalf("string tables = %+v", table)
	}
	got := map[string]string{}
	for _, s := range table[0].children {
		if s.typ != 1 {
			t.Errorf("string %q has type %d, want text", s.key, s.typ)
		}
		u := make([]uint16, len(s.value)/2)
		for i := range u {
			u[i] = le.Uint16(s.value[2*i:])
		}
		if len(u) == 0 || u[len(u)-1] != 0 || s.valueLen != len(u) {
			t.Errorf("string %q: %d characters, value length %d, no NUL", s.key, len(u), s.valueLen)
			continue
		}
		got[s.key] = string(utf16.Decode(u[:len(u)-1]))
	}
	wantStrings := map[string]string{
		"FileDescription": "Counter ü", "FileVersion": "1.2.3.4", "InternalName": "org.glutter.counter",
		"OriginalFilename": "counter.exe", "ProductName": "Counter ü", "ProductVersion": "1.2.3.4",
	}
	if len(got) != len(wantStrings) {
		t.Errorf("strings = %v", got)
	}
	for k, v := range wantStrings {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	vars := root.children[1].children
	if len(vars) != 1 || vars[0].key != "Translation" || le.Uint32(vars[0].value) != 0x04B00409 {
		t.Errorf("VarFileInfo = %+v, want the translation 0x04B00409", vars)
	}
}
