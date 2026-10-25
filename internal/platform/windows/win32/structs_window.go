package win32

// More Go mirrors of Win32 structures, for the windows and the displays (step 28.2). Like those of
// structs.go they carry no OS tag, so that the layout test checks them on any host against the
// mingw-w64 headers (SPEC §25.2); the `c` tag of a field is its name in the C header.

// MinMaxInfo is the Go mirror of MINMAXINFO, which WM_GETMINMAXINFO points to.
//
// Flutter: no counterpart
type MinMaxInfo struct {
	Reserved     Point `c:"ptReserved"`
	MaxSize      Point `c:"ptMaxSize"`
	MaxPosition  Point `c:"ptMaxPosition"`
	MinTrackSize Point `c:"ptMinTrackSize"`
	MaxTrackSize Point `c:"ptMaxTrackSize"`
}

// WindowPlacement is the Go mirror of WINDOWPLACEMENT (GetWindowPlacement, SetWindowPlacement).
//
// Flutter: no counterpart
type WindowPlacement struct {
	Length         uint32 `c:"length"`
	Flags          uint32 `c:"flags"`
	ShowCmd        uint32 `c:"showCmd"`
	MinPosition    Point  `c:"ptMinPosition"`
	MaxPosition    Point  `c:"ptMaxPosition"`
	NormalPosition Rect   `c:"rcNormalPosition"`
}

// MonitorInfoEx is the Go mirror of MONITORINFOEXW (GetMonitorInfoW).
//
// Flutter: no counterpart
type MonitorInfoEx struct {
	Size    uint32     `c:"cbSize"`
	Monitor Rect       `c:"rcMonitor"`
	Work    Rect       `c:"rcWork"`
	Flags   uint32     `c:"dwFlags"`
	Device  [32]uint16 `c:"szDevice"`
}

// WindowPos is the Go mirror of WINDOWPOS.
//
// Flutter: no counterpart
type WindowPos struct {
	Hwnd            uintptr `c:"hwnd"`
	HwndInsertAfter uintptr `c:"hwndInsertAfter"`
	X               int32   `c:"x"`
	Y               int32   `c:"y"`
	Cx              int32   `c:"cx"`
	Cy              int32   `c:"cy"`
	Flags           uint32  `c:"flags"`
}

// NcCalcSizeParams is the Go mirror of NCCALCSIZE_PARAMS, which WM_NCCALCSIZE points to when its
// wParam is true.
//
// Flutter: no counterpart
type NcCalcSizeParams struct {
	Rects [3]Rect    `c:"rgrc"`
	Pos   *WindowPos `c:"lppos"`
}

// CreateStruct is the Go mirror of CREATESTRUCTW, which WM_NCCREATE and WM_CREATE point to.
//
// Flutter: no counterpart
type CreateStruct struct {
	CreateParams uintptr `c:"lpCreateParams"`
	Instance     uintptr `c:"hInstance"`
	Menu         uintptr `c:"hMenu"`
	Parent       uintptr `c:"hwndParent"`
	Cy           int32   `c:"cy"`
	Cx           int32   `c:"cx"`
	Y            int32   `c:"y"`
	X            int32   `c:"x"`
	Style        int32   `c:"style"`
	Name         *uint16 `c:"lpszName"`
	Class        *uint16 `c:"lpszClass"`
	ExStyle      uint32  `c:"dwExStyle"`
}

// PaintStruct is the Go mirror of PAINTSTRUCT (BeginPaint, EndPaint).
//
// Flutter: no counterpart
type PaintStruct struct {
	Hdc         uintptr  `c:"hdc"`
	Erase       int32    `c:"fErase"`
	Paint       Rect     `c:"rcPaint"`
	Restore     int32    `c:"fRestore"`
	IncUpdate   int32    `c:"fIncUpdate"`
	RgbReserved [32]byte `c:"rgbReserved"`
}

// BitmapInfoHeader is the Go mirror of BITMAPINFOHEADER.
//
// Flutter: no counterpart
type BitmapInfoHeader struct {
	Size          uint32 `c:"biSize"`
	Width         int32  `c:"biWidth"`
	Height        int32  `c:"biHeight"`
	Planes        uint16 `c:"biPlanes"`
	BitCount      uint16 `c:"biBitCount"`
	Compression   uint32 `c:"biCompression"`
	SizeImage     uint32 `c:"biSizeImage"`
	XPelsPerMeter int32  `c:"biXPelsPerMeter"`
	YPelsPerMeter int32  `c:"biYPelsPerMeter"`
	ClrUsed       uint32 `c:"biClrUsed"`
	ClrImportant  uint32 `c:"biClrImportant"`
}

// BitmapInfo is the Go mirror of BITMAPINFO for a bitmap with no color table (32 bits per pixel,
// BI_RGB): the header and the single RGBQUAD entry the C type declares.
//
// Flutter: no counterpart
type BitmapInfo struct {
	Header BitmapInfoHeader `c:"bmiHeader"`
	Colors [4]byte          `c:"bmiColors"`
}

// IconInfo is the Go mirror of ICONINFO (CreateIconIndirect).
//
// Flutter: no counterpart
type IconInfo struct {
	Icon     int32   `c:"fIcon"`
	XHotspot uint32  `c:"xHotspot"`
	YHotspot uint32  `c:"yHotspot"`
	Mask     uintptr `c:"hbmMask"`
	Color    uintptr `c:"hbmColor"`
}

// DevMode is the Go mirror of DEVMODEW, in its display form (the first union holds the position
// and the display orientation). Only the fields the platform reads are named; the Size field says
// how long the structure is.
//
// Flutter: no counterpart
type DevMode struct {
	DeviceName         [32]uint16 `c:"dmDeviceName"`
	SpecVersion        uint16     `c:"dmSpecVersion"`
	DriverVersion      uint16     `c:"dmDriverVersion"`
	Size               uint16     `c:"dmSize"`
	DriverExtra        uint16     `c:"dmDriverExtra"`
	Fields             uint32     `c:"dmFields"`
	PositionX          int32      `c:"dmPosition.x"`
	PositionY          int32      `c:"dmPosition.y"`
	DisplayOrientation uint32     `c:"dmDisplayOrientation"`
	DisplayFixedOutput uint32     `c:"dmDisplayFixedOutput"`
	Color              int16      `c:"dmColor"`
	Duplex             int16      `c:"dmDuplex"`
	YResolution        int16      `c:"dmYResolution"`
	TTOption           int16      `c:"dmTTOption"`
	Collate            int16      `c:"dmCollate"`
	FormName           [32]uint16 `c:"dmFormName"`
	LogPixels          uint16     `c:"dmLogPixels"`
	BitsPerPel         uint32     `c:"dmBitsPerPel"`
	PelsWidth          uint32     `c:"dmPelsWidth"`
	PelsHeight         uint32     `c:"dmPelsHeight"`
	DisplayFlags       uint32     `c:"dmDisplayFlags"`
	DisplayFrequency   uint32     `c:"dmDisplayFrequency"`
	ICMMethod          uint32     `c:"dmICMMethod"`
	ICMIntent          uint32     `c:"dmICMIntent"`
	MediaType          uint32     `c:"dmMediaType"`
	DitherType         uint32     `c:"dmDitherType"`
	Reserved1          uint32     `c:"dmReserved1"`
	Reserved2          uint32     `c:"dmReserved2"`
	PanningWidth       uint32     `c:"dmPanningWidth"`
	PanningHeight      uint32     `c:"dmPanningHeight"`
}

// DisplayDevice is the Go mirror of DISPLAY_DEVICEW (EnumDisplayDevicesW).
//
// Flutter: no counterpart
type DisplayDevice struct {
	Size         uint32      `c:"cb"`
	DeviceName   [32]uint16  `c:"DeviceName"`
	DeviceString [128]uint16 `c:"DeviceString"`
	StateFlags   uint32      `c:"StateFlags"`
	DeviceID     [128]uint16 `c:"DeviceID"`
	DeviceKey    [128]uint16 `c:"DeviceKey"`
}
