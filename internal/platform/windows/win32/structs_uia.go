package win32

// The Go mirrors of the structures of UI Automation and OLE Automation (SPEC §25.11). The layout test of
// this package checks them against the mingw-w64 headers.

// UiaRect is the Go mirror of UiaRect: a rectangle of doubles, with the width and the height (not the
// right and bottom) that RECT has.
//
// Flutter: no counterpart
type UiaRect struct {
	Left   float64 `c:"left"`
	Top    float64 `c:"top"`
	Width  float64 `c:"width"`
	Height float64 `c:"height"`
}

// UiaPoint is the Go mirror of UiaPoint.
//
// Flutter: no counterpart
type UiaPoint struct {
	X float64 `c:"x"`
	Y float64 `c:"y"`
}

// Variant is the Go mirror of VARIANT on a 64-bit target: the type, three reserved words, and the
// 16 bytes of the union, of which the first eight hold a number, a BSTR or an interface pointer.
//
// Flutter: no counterpart
type Variant struct {
	Vt  uint16 `c:"vt"`
	_   [3]uint16
	Val uint64 `c:"llVal"`
	_   uint64
}

// The types of a VARIANT (VT_*) that the provider properties use.
const (
	// VTEmpty is VT_EMPTY: no value.
	//
	// Flutter: no counterpart
	VTEmpty = 0
	// VTI4 is VT_I4: a 32-bit integer.
	//
	// Flutter: no counterpart
	VTI4 = 3
	// VTR8 is VT_R8: a double.
	//
	// Flutter: no counterpart
	VTR8 = 5
	// VTBSTR is VT_BSTR: a string.
	//
	// Flutter: no counterpart
	VTBSTR = 8
	// VTUnknown is VT_UNKNOWN: an interface pointer.
	//
	// Flutter: no counterpart
	VTUnknown = 13
	// VTBool is VT_BOOL: VARIANT_TRUE (-1) or VARIANT_FALSE (0).
	//
	// Flutter: no counterpart
	VTBool = 11
	// VTArray is VT_ARRAY: a SAFEARRAY of the type in the low bits.
	//
	// Flutter: no counterpart
	VTArray = 0x2000
)

// VariantTrue is VARIANT_TRUE.
//
// Flutter: no counterpart
const VariantTrue = 0xFFFF

// TagWindowsUIAFailed is the tag of the error for a call of UI Automation that did not give a result;
// the metadata are "function" and, for a window, "hwnd".
//
// Flutter: no counterpart
const TagWindowsUIAFailed = "WindowsUIAFailed"
