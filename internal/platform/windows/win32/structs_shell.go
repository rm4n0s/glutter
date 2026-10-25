package win32

// The Go mirrors of the structures of the clipboard and of the system settings (SPEC §25.7,
// §25.8). The layout test of this package checks them against the mingw-w64 headers.

// DropFiles is the Go mirror of DROPFILES, the header of the CF_HDROP format: the paths follow it
// at PFiles, one after the other, each ended by a NUL, and the list by one more NUL. FWide says
// that the paths are UTF-16.
//
// Flutter: no counterpart
type DropFiles struct {
	PFiles uint32 `c:"pFiles"`
	Pt     Point  `c:"pt"`
	FNC    int32  `c:"fNC"`
	FWide  int32  `c:"fWide"`
}

// BitmapV5Header is the Go mirror of BITMAPV5HEADER, the header of the CF_DIBV5 format.
//
// Flutter: no counterpart
type BitmapV5Header struct {
	Size          uint32   `c:"bV5Size"`
	Width         int32    `c:"bV5Width"`
	Height        int32    `c:"bV5Height"`
	Planes        uint16   `c:"bV5Planes"`
	BitCount      uint16   `c:"bV5BitCount"`
	Compression   uint32   `c:"bV5Compression"`
	SizeImage     uint32   `c:"bV5SizeImage"`
	XPelsPerMeter int32    `c:"bV5XPelsPerMeter"`
	YPelsPerMeter int32    `c:"bV5YPelsPerMeter"`
	ClrUsed       uint32   `c:"bV5ClrUsed"`
	ClrImportant  uint32   `c:"bV5ClrImportant"`
	RedMask       uint32   `c:"bV5RedMask"`
	GreenMask     uint32   `c:"bV5GreenMask"`
	BlueMask      uint32   `c:"bV5BlueMask"`
	AlphaMask     uint32   `c:"bV5AlphaMask"`
	CSType        uint32   `c:"bV5CSType"`
	Endpoints     [9]int32 `c:"bV5Endpoints"`
	GammaRed      uint32   `c:"bV5GammaRed"`
	GammaGreen    uint32   `c:"bV5GammaGreen"`
	GammaBlue     uint32   `c:"bV5GammaBlue"`
	Intent        uint32   `c:"bV5Intent"`
	ProfileData   uint32   `c:"bV5ProfileData"`
	ProfileSize   uint32   `c:"bV5ProfileSize"`
	Reserved      uint32   `c:"bV5Reserved"`
}

// HighContrast is the Go mirror of HIGHCONTRASTW (SPI_GETHIGHCONTRAST).
//
// Flutter: no counterpart
type HighContrast struct {
	Size          uint32  `c:"cbSize"`
	Flags         uint32  `c:"dwFlags"`
	DefaultScheme uintptr `c:"lpszDefaultScheme"`
}

// LogFont is the Go mirror of LOGFONTW.
//
// Flutter: no counterpart
type LogFont struct {
	Height         int32      `c:"lfHeight"`
	Width          int32      `c:"lfWidth"`
	Escapement     int32      `c:"lfEscapement"`
	Orientation    int32      `c:"lfOrientation"`
	Weight         int32      `c:"lfWeight"`
	Italic         uint8      `c:"lfItalic"`
	Underline      uint8      `c:"lfUnderline"`
	StrikeOut      uint8      `c:"lfStrikeOut"`
	CharSet        uint8      `c:"lfCharSet"`
	OutPrecision   uint8      `c:"lfOutPrecision"`
	ClipPrecision  uint8      `c:"lfClipPrecision"`
	Quality        uint8      `c:"lfQuality"`
	PitchAndFamily uint8      `c:"lfPitchAndFamily"`
	FaceName       [32]uint16 `c:"lfFaceName"`
}

// NonClientMetrics is the Go mirror of NONCLIENTMETRICSW (SPI_GETNONCLIENTMETRICS).
//
// Flutter: no counterpart
type NonClientMetrics struct {
	Size            uint32  `c:"cbSize"`
	BorderWidth     int32   `c:"iBorderWidth"`
	ScrollWidth     int32   `c:"iScrollWidth"`
	ScrollHeight    int32   `c:"iScrollHeight"`
	CaptionWidth    int32   `c:"iCaptionWidth"`
	CaptionHeight   int32   `c:"iCaptionHeight"`
	CaptionFont     LogFont `c:"lfCaptionFont"`
	SmCaptionWidth  int32   `c:"iSmCaptionWidth"`
	SmCaptionHeight int32   `c:"iSmCaptionHeight"`
	SmCaptionFont   LogFont `c:"lfSmCaptionFont"`
	MenuWidth       int32   `c:"iMenuWidth"`
	MenuHeight      int32   `c:"iMenuHeight"`
	MenuFont        LogFont `c:"lfMenuFont"`
	StatusFont      LogFont `c:"lfStatusFont"`
	MessageFont     LogFont `c:"lfMessageFont"`
	PaddedBorder    int32   `c:"iPaddedBorderWidth"`
}
