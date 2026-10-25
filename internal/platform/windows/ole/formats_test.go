package ole

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestDataSetKeepsTheOrderAndReplacesInPlace(t *testing.T) {
	s := NewDataSet(Item{Format: 13, Data: []byte("a")}, Item{Format: 15, Data: []byte("b")})
	s.Add(13, []byte("c"))
	s.Add(0xC001, []byte("d"))
	if got, want := s.Formats(), []uint32{13, 15, 0xC001}; !reflect.DeepEqual(got, want) {
		t.Errorf("Formats = %v, want %v", got, want)
	}
	if d, ok := s.Get(13); !ok || string(d) != "c" {
		t.Errorf("Get(13) = %q, %v: the second Add should have replaced the data", d, ok)
	}
	if _, ok := s.Get(99); ok {
		t.Error("Get(99) found a format that is not there")
	}
}

func TestQueryGetData(t *testing.T) {
	s := NewDataSet(Item{Format: win32.CFUnicodeText, Data: []byte("x")})
	good := FormatEtcOf(win32.CFUnicodeText)
	withTymed := func(f win32.FormatEtc, tymed uint32) win32.FormatEtc { f.Tymed = tymed; return f }
	withAspect := func(f win32.FormatEtc, aspect uint32) win32.FormatEtc { f.DwAspect = aspect; return f }
	withIndex := func(f win32.FormatEtc, i int32) win32.FormatEtc { f.Lindex = i; return f }
	tests := []struct {
		name string
		fe   win32.FormatEtc
		want win32.HRESULT
	}{
		{"the format on global memory", good, win32.SOK},
		{"global memory among other media", withTymed(good, win32.TymedHGlobal|win32.TymedIStream), win32.SOK},
		{"a format the set lacks", FormatEtcOf(win32.CFHDrop), win32.DVEFormatEtc},
		{"only a stream", withTymed(good, win32.TymedIStream), win32.DVETymed},
		{"another aspect", withAspect(good, 2), win32.DVEDVAspect},
		{"an index", withIndex(good, 0), win32.DVELIndex},
		{"the aspect is checked first", withAspect(withIndex(FormatEtcOf(win32.CFHDrop), 3), 4), win32.DVEDVAspect},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Query(tt.fe); got != tt.want {
				t.Errorf("Query = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMIMEOf(t *testing.T) {
	tests := []struct {
		format uint32
		name   string
		want   string
	}{
		{win32.CFHDrop, "", MIMEURIList},
		{win32.CFUnicodeText, "", MIMEPlainUTF8},
		{win32.CFText, "", MIMEPlain},
		{win32.CFOEMText, "", MIMEPlain},
		{0xC010, win32.FormatNameURLW, MIMEURIList},
		{0xC011, win32.FormatNameURL, MIMEURIList},
		{0xC012, "PNG", MIMEPNG},
		{0xC013, "HTML Format", MIMEHTML},
		{0xC014, "Rich Text Format", "Rich Text Format"},
		{win32.CFDIB, "", ""}, // a standard format that is not known
		{0xC015, LocalMarker(7, 3), LocalMarker(7, 3)},
	}
	for _, tt := range tests {
		if got := MIMEOf(tt.format, tt.name); got != tt.want {
			t.Errorf("MIMEOf(%d, %q) = %q, want %q", tt.format, tt.name, got, tt.want)
		}
	}
}

func TestLocalMarker(t *testing.T) {
	tests := []struct {
		name string
		in   string
		pid  int
		want uint64
	}{
		{"this process", LocalMarker(42, 7), 42, 7},
		{"another process", LocalMarker(41, 7), 42, 0},
		{"not a marker", "text/plain", 42, 0},
		{"no id", LocalMarkerPrefix + "42", 42, 0},
		{"a bad id", LocalMarkerPrefix + "42-x", 42, 0},
		{"a pid that is a prefix of ours", LocalMarker(4, 7), 42, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LocalIDOf(tt.in, tt.pid); got != tt.want {
				t.Errorf("LocalIDOf(%q, %d) = %d, want %d", tt.in, tt.pid, got, tt.want)
			}
		})
	}
	if !IsMarker(LocalMarker(1, 1)) || IsMarker("text/plain") {
		t.Error("IsMarker is wrong")
	}
}

func TestStringDataRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
		wide bool
		want string
	}{
		{"wide", "https://example.com/é", true, "https://example.com/é"},
		{"wide, outside the BMP", "a𠮷", true, "a𠮷"},
		{"ansi", "https://example.com/", false, "https://example.com/"},
		{"ansi, Latin-1", "café", false, "café"},
		{"ansi, beyond Latin-1 is a question mark", "a𠮷b", false, "a?b"},
		{"it ends at its NUL", "ab\x00cd", true, "ab"},
		{"empty", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := StringData(tt.in, tt.wide)
			if tt.wide && (len(data) < 2 || data[len(data)-1] != 0 || data[len(data)-2] != 0) {
				t.Errorf("%x has no terminating NUL", data)
			}
			if !tt.wide && data[len(data)-1] != 0 {
				t.Errorf("%x has no terminating NUL", data)
			}
			if got := StringFromData(data, tt.wide); got != tt.want {
				t.Errorf("round trip = %q, want %q", got, tt.want)
			}
		})
	}
	// Data with no NUL, and an odd byte at the end of UTF-16.
	if got := StringFromData([]byte{'a', 0, 'b', 0, 'c'}, true); got != "ab" {
		t.Errorf("odd UTF-16 data = %q", got)
	}
}
