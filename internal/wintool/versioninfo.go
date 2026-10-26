package wintool

import (
	"encoding/binary"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/rm4n0s/errors"
)

// Version is a file or product version of four numbers, such as 1.2.3.4.
//
// Flutter: no counterpart
type Version [4]uint16

// String is the version as "1.2.3.4".
func (v Version) String() string {
	return strconv.Itoa(int(v[0])) + "." + strconv.Itoa(int(v[1])) + "." + strconv.Itoa(int(v[2])) + "." + strconv.Itoa(int(v[3]))
}

// DefaultVersion is the version of an app built without -version.
//
// Flutter: no counterpart
var DefaultVersion = Version{1, 0, 0, 0}

// ParseVersion reads "1", "1.2", "1.2.3" or "1.2.3.4" (each number from 0 to 65535; a missing one
// is 0). An empty text is DefaultVersion. It fails with WindowsResourceInvalid.
//
// Flutter: no counterpart (pubspec.yaml version)
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultVersion, nil
	}
	parts := strings.Split(s, ".")
	if len(parts) > 4 {
		return Version{}, errors.New(tagWindowsResourceInvalid, "a version has at most four numbers, such as 1.2.3.4", "version", s)
	}
	var v Version
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return Version{}, errors.NewErr(tagWindowsResourceInvalid, err).SetMetadata("version", s)
		}
		v[i] = uint16(n)
	}
	return v, nil
}

// VersionInfo is what the VERSIONINFO resource says (SPEC §25.1).
//
// Flutter: no counterpart
type VersionInfo struct {
	// Version is the file and product version.
	Version Version
	// Name is the product name and the file description.
	Name string
	// AppID is the internal name.
	AppID string
	// FileName is the original file name, such as "counter.exe".
	FileName string
}

// versionLanguage is the language and code page of the version strings: English (United States),
// Unicode.
const versionLanguage = "040904B0"

// Encode returns the binary VS_VERSIONINFO structure, the data of the RT_VERSION resource: the fixed
// file information, a StringFileInfo block with the strings and a VarFileInfo block with the
// translation.
//
// Flutter: no counterpart
func (vi VersionInfo) Encode() []byte {
	var fixed []byte
	put32 := func(v uint32) { fixed = binary.LittleEndian.AppendUint32(fixed, v) }
	ms := func(v Version) uint32 { return uint32(v[0])<<16 | uint32(v[1]) }
	ls := func(v Version) uint32 { return uint32(v[2])<<16 | uint32(v[3]) }
	put32(0xFEEF04BD) // dwSignature
	put32(0x00010000) // dwStrucVersion
	put32(ms(vi.Version))
	put32(ls(vi.Version)) // dwFileVersion
	put32(ms(vi.Version))
	put32(ls(vi.Version)) // dwProductVersion
	put32(0x3F)           // dwFileFlagsMask
	put32(0)              // dwFileFlags
	put32(0x00040004)     // dwFileOS: VOS_NT_WINDOWS32
	put32(1)              // dwFileType: VFT_APP
	put32(0)              // dwFileSubtype
	put32(0)
	put32(0) // dwFileDate

	version := vi.Version.String()
	strs := []struct{ key, value string }{
		{"FileDescription", vi.Name},
		{"FileVersion", version},
		{"InternalName", vi.AppID},
		{"OriginalFilename", vi.FileName},
		{"ProductName", vi.Name},
		{"ProductVersion", version},
	}
	var table []byte
	for _, s := range strs {
		table = append(table, versionBlock(s.key, 1, utf16Z(s.value), len(utf16.Encode([]rune(s.value)))+1, nil)...)
	}
	stringFileInfo := versionBlock("StringFileInfo", 1, nil, 0, versionBlock(versionLanguage, 1, nil, 0, table))
	translation := binary.LittleEndian.AppendUint32(nil, 0x04B00409)
	varFileInfo := versionBlock("VarFileInfo", 1, nil, 0, versionBlock("Translation", 0, translation, 4, nil))
	return versionBlock("VS_VERSION_INFO", 0, fixed, len(fixed), append(stringFileInfo, varFileInfo...))
}

// versionBlock encodes one node of the version tree: its length, the length of its value (in bytes
// for binary values, in characters for text, as valueLen says), the type (0 binary, 1 text), the key
// as UTF-16 with a NUL, padding to 32 bits, the value, padding, and the children.
func versionBlock(key string, typ uint16, value []byte, valueLen int, children []byte) []byte {
	var b []byte
	b = binary.LittleEndian.AppendUint16(b, 0) // wLength, set below
	b = binary.LittleEndian.AppendUint16(b, uint16(valueLen))
	b = binary.LittleEndian.AppendUint16(b, typ)
	b = append(b, utf16Z(key)...)
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	b = append(b, value...)
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	b = append(b, children...)
	binary.LittleEndian.PutUint16(b, uint16(len(b)))
	for len(b)%4 != 0 { // the padding between siblings is not part of wLength
		b = append(b, 0)
	}
	return b
}

// utf16Z is the text as little-endian UTF-16 with a terminating NUL.
func utf16Z(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return append(b, 0, 0)
}
