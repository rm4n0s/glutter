package win32

// The Go mirrors of the structures of OLE drag and drop (SPEC §25.10). The layout test of this
// package checks them against the mingw-w64 headers.

// FormatEtc is the Go mirror of FORMATETC: a clipboard format with the aspect and the medium in
// which a data object has or wants it.
//
// Flutter: no counterpart
type FormatEtc struct {
	// CfFormat is the clipboard format (CLIPFORMAT, a 16-bit number).
	CfFormat uint16 `c:"cfFormat"`
	// Ptd is the target device (DVTARGETDEVICE *), always null here.
	Ptd uintptr `c:"ptd"`
	// DwAspect is DVASPECT_CONTENT for the data itself.
	DwAspect uint32 `c:"dwAspect"`
	// Lindex is -1 for all of the data.
	Lindex int32 `c:"lindex"`
	// Tymed is the set of media (TYMED_*).
	Tymed uint32 `c:"tymed"`
}

// StgMedium is the Go mirror of STGMEDIUM: data of one format and the medium that holds it. Handle
// is the member of the union that the medium names (the HGLOBAL for TYMED_HGLOBAL).
//
// Flutter: no counterpart
type StgMedium struct {
	// Tymed is the medium (TYMED_*).
	Tymed uint32 `c:"tymed"`
	// Handle is the union: hGlobal, pstm and the others share this place.
	Handle uintptr `c:"hGlobal"`
	// PUnkForRelease is the object that frees the data when it is released, null to free the medium
	// by its type.
	PUnkForRelease uintptr `c:"pUnkForRelease"`
}
