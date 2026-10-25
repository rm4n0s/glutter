package win32

// The Go mirrors of the structures of the pointer input (step 28.3). Like those of structs.go they
// carry no OS tag, so that the layout test checks them against the headers (SPEC §25.2); the `c`
// tag of a field is its name in the C header.

// TrackMouseEventData is the Go mirror of TRACKMOUSEEVENT (TrackMouseEvent).
//
// Flutter: no counterpart
type TrackMouseEventData struct {
	Size      uint32  `c:"cbSize"`
	Flags     uint32  `c:"dwFlags"`
	Track     uintptr `c:"hwndTrack"`
	HoverTime uint32  `c:"dwHoverTime"`
}

// PointerInfo is the Go mirror of POINTER_INFO (GetPointerInfo, GetPointerFrameInfo).
//
// Flutter: no counterpart
type PointerInfo struct {
	PointerType         uint32  `c:"pointerType"`
	PointerID           uint32  `c:"pointerId"`
	FrameID             uint32  `c:"frameId"`
	PointerFlags        uint32  `c:"pointerFlags"`
	SourceDevice        uintptr `c:"sourceDevice"`
	HwndTarget          uintptr `c:"hwndTarget"`
	PixelLocation       Point   `c:"ptPixelLocation"`
	HimetricLocation    Point   `c:"ptHimetricLocation"`
	PixelLocationRaw    Point   `c:"ptPixelLocationRaw"`
	HimetricLocationRaw Point   `c:"ptHimetricLocationRaw"`
	Time                uint32  `c:"dwTime"`
	HistoryCount        uint32  `c:"historyCount"`
	InputData           int32   `c:"InputData"`
	KeyStates           uint32  `c:"dwKeyStates"`
	PerformanceCount    uint64  `c:"PerformanceCount"`
	ButtonChangeType    uint32  `c:"ButtonChangeType"`
}

// PointerTouchInfo is the Go mirror of POINTER_TOUCH_INFO (GetPointerTouchInfo).
//
// Flutter: no counterpart
type PointerTouchInfo struct {
	Pointer     PointerInfo `c:"pointerInfo"`
	TouchFlags  uint32      `c:"touchFlags"`
	TouchMask   uint32      `c:"touchMask"`
	Contact     Rect        `c:"rcContact"`
	ContactRaw  Rect        `c:"rcContactRaw"`
	Orientation uint32      `c:"orientation"`
	Pressure    uint32      `c:"pressure"`
}

// PointerPenInfo is the Go mirror of POINTER_PEN_INFO (GetPointerPenInfo).
//
// Flutter: no counterpart
type PointerPenInfo struct {
	Pointer  PointerInfo `c:"pointerInfo"`
	PenFlags uint32      `c:"penFlags"`
	PenMask  uint32      `c:"penMask"`
	Pressure uint32      `c:"pressure"`
	Rotation uint32      `c:"rotation"`
	TiltX    int32       `c:"tiltX"`
	TiltY    int32       `c:"tiltY"`
}

// MouseInput is the Go mirror of MOUSEINPUT, the mouse member of INPUT.
//
// Flutter: no counterpart
type MouseInput struct {
	DX        int32   `c:"dx"`
	DY        int32   `c:"dy"`
	MouseData uint32  `c:"mouseData"`
	Flags     uint32  `c:"dwFlags"`
	Time      uint32  `c:"time"`
	ExtraInfo uintptr `c:"dwExtraInfo"`
}

// Input is the Go mirror of INPUT for a mouse input (the union has the mouse member only, which is
// its largest one: 40 bytes in all).
//
// Flutter: no counterpart
type Input struct {
	Type  uint32     `c:"type"`
	Mouse MouseInput `c:"mi"`
}

// KeybdInput is the Go mirror of KEYBDINPUT, the keyboard member of INPUT (step 28.4).
//
// Flutter: no counterpart
type KeybdInput struct {
	VK        uint16  `c:"wVk"`
	Scan      uint16  `c:"wScan"`
	Flags     uint32  `c:"dwFlags"`
	Time      uint32  `c:"time"`
	ExtraInfo uintptr `c:"dwExtraInfo"`
}

// KeyInput is the Go mirror of INPUT for a keyboard input. The union of INPUT is as large as its
// mouse member, so the keyboard member is followed by 8 bytes of padding and the whole is 40 bytes
// on a 64-bit target, the same as Input.
//
// Flutter: no counterpart
type KeyInput struct {
	Type uint32     `c:"type"`
	Key  KeybdInput `c:"ki"`
	_    [8]byte
}

// CursorInfo is the Go mirror of CURSORINFO (GetCursorInfo).
//
// Flutter: no counterpart
type CursorInfo struct {
	Size      uint32  `c:"cbSize"`
	Flags     uint32  `c:"flags"`
	Cursor    uintptr `c:"hCursor"`
	ScreenPos Point   `c:"ptScreenPos"`
}
