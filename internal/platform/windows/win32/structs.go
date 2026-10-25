package win32

// The Go mirrors of the Win32 structures that this package and the first steps of the Windows port
// need. They are laid out like the Windows structures on a 64-bit target, which is the same on
// amd64 and arm64, so they carry no OS tag and the layout test of this package checks them on any
// host against the mingw-w64 headers (SPEC §25.2, package layouttest). The `c` tag of a field is
// its name in the C header; the test reads it.

// Point is the Go mirror of POINT.
//
// Flutter: no counterpart
type Point struct {
	X int32 `c:"x"`
	Y int32 `c:"y"`
}

// Rect is the Go mirror of RECT.
//
// Flutter: no counterpart
type Rect struct {
	Left   int32 `c:"left"`
	Top    int32 `c:"top"`
	Right  int32 `c:"right"`
	Bottom int32 `c:"bottom"`
}

// WndClassEx is the Go mirror of WNDCLASSEXW, the description of a window class for
// RegisterClassExW. WndProc is the address of a callback made by NewCallback.
//
// Flutter: no counterpart
type WndClassEx struct {
	Size       uint32  `c:"cbSize"`
	Style      uint32  `c:"style"`
	WndProc    uintptr `c:"lpfnWndProc"`
	ClsExtra   int32   `c:"cbClsExtra"`
	WndExtra   int32   `c:"cbWndExtra"`
	Instance   uintptr `c:"hInstance"`
	Icon       uintptr `c:"hIcon"`
	Cursor     uintptr `c:"hCursor"`
	Background uintptr `c:"hbrBackground"`
	MenuName   *uint16 `c:"lpszMenuName"`
	ClassName  *uint16 `c:"lpszClassName"`
	IconSm     uintptr `c:"hIconSm"`
}

// Msg is the Go mirror of MSG, a message of the message queue.
//
// Flutter: no counterpart
type Msg struct {
	Hwnd    uintptr `c:"hwnd"`
	Message uint32  `c:"message"`
	WParam  uintptr `c:"wParam"`
	LParam  uintptr `c:"lParam"`
	Time    uint32  `c:"time"`
	Pt      Point   `c:"pt"`
}
