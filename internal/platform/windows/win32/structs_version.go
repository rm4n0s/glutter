package win32

import "strconv"

// VSFixedFileInfo is the Go mirror of VS_FIXEDFILEINFO, the fixed part of the version resource of a
// file (FileVersion reads it). The layout test of this package checks it against the mingw-w64
// headers.
//
// Flutter: no counterpart
type VSFixedFileInfo struct {
	Signature        uint32 `c:"dwSignature"`
	StrucVersion     uint32 `c:"dwStrucVersion"`
	FileVersionMS    uint32 `c:"dwFileVersionMS"`
	FileVersionLS    uint32 `c:"dwFileVersionLS"`
	ProductVersionMS uint32 `c:"dwProductVersionMS"`
	ProductVersionLS uint32 `c:"dwProductVersionLS"`
	FileFlagsMask    uint32 `c:"dwFileFlagsMask"`
	FileFlags        uint32 `c:"dwFileFlags"`
	FileOS           uint32 `c:"dwFileOS"`
	FileType         uint32 `c:"dwFileType"`
	FileSubtype      uint32 `c:"dwFileSubtype"`
	FileDateMS       uint32 `c:"dwFileDateMS"`
	FileDateLS       uint32 `c:"dwFileDateLS"`
}

// VSFixedFileInfoSignature is VS_FFI_SIGNATURE, the value of VSFixedFileInfo.Signature.
//
// Flutter: no counterpart
const VSFixedFileInfoSignature = 0xFEEF04BD

// FileVersion is the four numbers of a file's version, such as 10.0.19041.1 (the file version of
// the version resource).
//
// Flutter: no counterpart
type FileVersion struct {
	Major, Minor, Build, Revision uint16
}

// FileVersionOf reads the version out of a VSFixedFileInfo.
//
// Flutter: no counterpart
func FileVersionOf(info VSFixedFileInfo) FileVersion {
	return FileVersion{
		Major:    uint16(info.FileVersionMS >> 16),
		Minor:    uint16(info.FileVersionMS),
		Build:    uint16(info.FileVersionLS >> 16),
		Revision: uint16(info.FileVersionLS),
	}
}

// String returns the version as "10.0.19041.1".
func (v FileVersion) String() string {
	return strconv.Itoa(int(v.Major)) + "." + strconv.Itoa(int(v.Minor)) + "." + strconv.Itoa(int(v.Build)) + "." + strconv.Itoa(int(v.Revision))
}
