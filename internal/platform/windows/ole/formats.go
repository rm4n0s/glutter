package ole

import (
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// Item is one format of a data set: the number of the clipboard format and its bytes.
//
// Flutter: no counterpart
type Item struct {
	Format uint32
	Data   []byte
}

// DataSet is the data of a drag that the app starts, as formats in the order they are offered. Every
// format is on global memory (TYMED_HGLOBAL), for the content aspect, as the formats of the
// clipboard are. It is not changed once the drag has begun.
//
// Flutter: no counterpart
type DataSet struct {
	items []Item
}

// NewDataSet makes a set of the items; a format given twice keeps its first place and its last data.
//
// Flutter: no counterpart
func NewDataSet(items ...Item) *DataSet {
	s := &DataSet{}
	for _, it := range items {
		s.Add(it.Format, it.Data)
	}
	return s
}

// Add puts the data of the format in the set: a format that is there already is replaced in place.
//
// Flutter: no counterpart
func (s *DataSet) Add(format uint32, data []byte) {
	for i := range s.items {
		if s.items[i].Format == format {
			s.items[i].Data = data
			return
		}
	}
	s.items = append(s.items, Item{Format: format, Data: data})
}

// Formats lists the formats in the order they were added.
//
// Flutter: no counterpart
func (s *DataSet) Formats() []uint32 {
	out := make([]uint32, len(s.items))
	for i, it := range s.items {
		out[i] = it.Format
	}
	return out
}

// Get returns the data of the format.
//
// Flutter: no counterpart
func (s *DataSet) Get(format uint32) ([]byte, bool) {
	for _, it := range s.items {
		if it.Format == format {
			return it.Data, true
		}
	}
	return nil, false
}

// Query is IDataObject::QueryGetData for the format: S_OK when the set has it on global memory in the
// content aspect for the whole data, else the code the method gives (DV_E_DVASPECT, DV_E_LINDEX,
// DV_E_TYMED, DV_E_FORMATETC, in the order the checks are made).
//
// Flutter: no counterpart
func (s *DataSet) Query(fe win32.FormatEtc) win32.HRESULT {
	switch {
	case fe.DwAspect != win32.DVAspectContent:
		return win32.DVEDVAspect
	case fe.Lindex != -1:
		return win32.DVELIndex
	case fe.Tymed&win32.TymedHGlobal == 0:
		return win32.DVETymed
	}
	if _, ok := s.Get(uint32(fe.CfFormat)); !ok {
		return win32.DVEFormatEtc
	}
	return win32.SOK
}

// FormatEtcOf is the FORMATETC that describes the format on global memory.
//
// Flutter: no counterpart
func FormatEtcOf(format uint32) win32.FormatEtc {
	return win32.FormatEtc{CfFormat: uint16(format), DwAspect: win32.DVAspectContent, Lindex: -1, Tymed: win32.TymedHGlobal}
}

// The types (MIME types, as on the other platforms) that the formats of a drag are reported as.
const (
	// MIMEPlain is the type of the text formats.
	//
	// Flutter: no counterpart
	MIMEPlain = "text/plain"
	// MIMEPlainUTF8 is the type of the UTF-16 text format, as the other backends name text.
	//
	// Flutter: no counterpart
	MIMEPlainUTF8 = "text/plain;charset=utf-8"
	// MIMEURIList is the type of the files and of the URL formats.
	//
	// Flutter: no counterpart
	MIMEURIList = "text/uri-list"
	// MIMEPNG is the type of the registered format "PNG".
	//
	// Flutter: no counterpart
	MIMEPNG = "image/png"
	// MIMEHTML is the type of the registered format "HTML Format".
	//
	// Flutter: no counterpart
	MIMEHTML = "text/html"
	// LocalMarkerPrefix starts the name of the format that a drag started by an app offers besides its
	// data: "application/x-glutter-drag-<pid>-<id>" (SPEC §8.11).
	//
	// Flutter: no counterpart
	LocalMarkerPrefix = "application/x-glutter-drag-"
)

// MIMEOf is the type that a format of a drag is reported as. name is the name of a registered format
// ("" for a standard one). The files are text/uri-list, the text formats text/plain, the two URL
// formats text/uri-list, "PNG" image/png and "HTML Format" text/html; another registered format is
// reported by its name, and a standard format that is not known, by "" (it is left out).
//
// Flutter: no counterpart
func MIMEOf(format uint32, name string) string {
	switch format {
	case win32.CFHDrop:
		return MIMEURIList
	case win32.CFUnicodeText:
		return MIMEPlainUTF8
	case win32.CFText, win32.CFOEMText:
		return MIMEPlain
	}
	switch name {
	case "":
		return ""
	case win32.FormatNameURLW, win32.FormatNameURL:
		return MIMEURIList
	case "PNG":
		return MIMEPNG
	case "HTML Format":
		return MIMEHTML
	}
	return name
}

// LocalMarker is the name of the format that marks a drag of this process: the process ID keeps the
// drags of another instance of the app from being taken for ours.
//
// Flutter: no counterpart
func LocalMarker(pid int, id uint64) string {
	return LocalMarkerPrefix + strconv.Itoa(pid) + "-" + strconv.FormatUint(id, 10)
}

// LocalIDOf is the ID of the drag request of the process pid that a format name marks, 0 when the name
// is not a marker of this process.
//
// Flutter: no counterpart
func LocalIDOf(name string, pid int) uint64 {
	rest, ok := strings.CutPrefix(name, LocalMarkerPrefix)
	if !ok {
		return 0
	}
	p, id, ok := strings.Cut(rest, "-")
	if !ok || p != strconv.Itoa(pid) {
		return 0
	}
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// IsMarker says whether the name is the marker of any drag of this app (of any process).
//
// Flutter: no counterpart
func IsMarker(name string) bool { return strings.HasPrefix(name, LocalMarkerPrefix) }

// PID is the process ID the markers carry.
//
// Flutter: no counterpart
func PID() int { return os.Getpid() }

// StringFromData reads the string of the data of a URL format: UTF-16 when wide, else one byte per
// character (Latin-1: the ANSI code page is not known here). The string ends at the first NUL.
//
// Flutter: no counterpart
func StringFromData(b []byte, wide bool) string {
	if !wide {
		rs := make([]rune, 0, len(b))
		for _, c := range b {
			if c == 0 {
				break
			}
			rs = append(rs, rune(c))
		}
		return string(rs)
	}
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		c := binary.LittleEndian.Uint16(b[i:])
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return string(utf16.Decode(u))
}

// StringData is the data of a text or URL format: the string with a terminating NUL, UTF-16 when wide,
// else ANSI (a character above U+00FF is "?").
//
// Flutter: no counterpart
func StringData(s string, wide bool) []byte {
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	if !wide {
		out := make([]byte, 0, len(s)+1)
		for _, r := range s {
			if r > 0xFF {
				r = '?'
			}
			out = append(out, byte(r))
		}
		return append(out, 0)
	}
	u := utf16.Encode([]rune(s))
	out := make([]byte, 0, 2*len(u)+2)
	for _, c := range u {
		out = binary.LittleEndian.AppendUint16(out, c)
	}
	return binary.LittleEndian.AppendUint16(out, 0)
}
