package win32

// The constants of the pointer input of step 28.3: the mouse, the wheel, the pointer messages of touch
// and pen, the cursors and the scripted input of the tests. They carry no OS tag so that the layout
// test checks them against the headers (SPEC §25.2). The name is the Go spelling of the macro.

// The messages of the mouse and the pointer, and their parameters.
const (
	// WMSetCursor is WM_SETCURSOR.
	//
	// Flutter: no counterpart
	WMSetCursor = 0x0020
	// WMCancelMode is WM_CANCELMODE.
	//
	// Flutter: no counterpart
	WMCancelMode = 0x001F
	// WMMouseMove is WM_MOUSEMOVE.
	//
	// Flutter: no counterpart
	WMMouseMove = 0x0200
	// WMLButtonDown is WM_LBUTTONDOWN.
	//
	// Flutter: no counterpart
	WMLButtonDown = 0x0201
	// WMLButtonUp is WM_LBUTTONUP.
	//
	// Flutter: no counterpart
	WMLButtonUp = 0x0202
	// WMRButtonDown is WM_RBUTTONDOWN.
	//
	// Flutter: no counterpart
	WMRButtonDown = 0x0204
	// WMRButtonUp is WM_RBUTTONUP.
	//
	// Flutter: no counterpart
	WMRButtonUp = 0x0205
	// WMMButtonDown is WM_MBUTTONDOWN.
	//
	// Flutter: no counterpart
	WMMButtonDown = 0x0207
	// WMMButtonUp is WM_MBUTTONUP.
	//
	// Flutter: no counterpart
	WMMButtonUp = 0x0208
	// WMMouseWheel is WM_MOUSEWHEEL.
	//
	// Flutter: no counterpart
	WMMouseWheel = 0x020A
	// WMXButtonDown is WM_XBUTTONDOWN.
	//
	// Flutter: no counterpart
	WMXButtonDown = 0x020B
	// WMXButtonUp is WM_XBUTTONUP.
	//
	// Flutter: no counterpart
	WMXButtonUp = 0x020C
	// WMMouseHWheel is WM_MOUSEHWHEEL.
	//
	// Flutter: no counterpart
	WMMouseHWheel = 0x020E
	// WMCaptureChanged is WM_CAPTURECHANGED.
	//
	// Flutter: no counterpart
	WMCaptureChanged = 0x0215
	// WMMouseLeave is WM_MOUSELEAVE.
	//
	// Flutter: no counterpart
	WMMouseLeave = 0x02A3
	// WMPointerUpdate is WM_POINTERUPDATE.
	//
	// Flutter: no counterpart
	WMPointerUpdate = 0x0245
	// WMPointerDown is WM_POINTERDOWN.
	//
	// Flutter: no counterpart
	WMPointerDown = 0x0246
	// WMPointerUp is WM_POINTERUP.
	//
	// Flutter: no counterpart
	WMPointerUp = 0x0247
	// WMPointerEnter is WM_POINTERENTER.
	//
	// Flutter: no counterpart
	WMPointerEnter = 0x0249
	// WMPointerLeave is WM_POINTERLEAVE.
	//
	// Flutter: no counterpart
	WMPointerLeave = 0x024A
	// WMPointerCaptureChanged is WM_POINTERCAPTURECHANGED.
	//
	// Flutter: no counterpart
	WMPointerCaptureChanged = 0x024C

	// MKLButton is MK_LBUTTON.
	//
	// Flutter: no counterpart
	MKLButton = 0x0001
	// MKRButton is MK_RBUTTON.
	//
	// Flutter: no counterpart
	MKRButton = 0x0002
	// MKShift is MK_SHIFT.
	//
	// Flutter: no counterpart
	MKShift = 0x0004
	// MKControl is MK_CONTROL.
	//
	// Flutter: no counterpart
	MKControl = 0x0008
	// MKMButton is MK_MBUTTON.
	//
	// Flutter: no counterpart
	MKMButton = 0x0010
	// MKXButton1 is MK_XBUTTON1.
	//
	// Flutter: no counterpart
	MKXButton1 = 0x0020
	// MKXButton2 is MK_XBUTTON2.
	//
	// Flutter: no counterpart
	MKXButton2 = 0x0040
	// XButton1 is XBUTTON1, the back button of a mouse, in the high word of the wParam of WM_XBUTTON*.
	//
	// Flutter: no counterpart
	XButton1 = 0x0001
	// XButton2 is XBUTTON2, the forward button.
	//
	// Flutter: no counterpart
	XButton2 = 0x0002
	// WheelDelta is WHEEL_DELTA: the wheel delta of one notch.
	//
	// Flutter: no counterpart
	WheelDelta = 120
	// WheelPageScroll is WHEEL_PAGESCROLL (UINT_MAX): the system's lines setting that scrolls a page.
	//
	// Flutter: no counterpart
	WheelPageScroll uint32 = 0xFFFFFFFF
	// TMELeave is TME_LEAVE.
	//
	// Flutter: no counterpart
	TMELeave = 0x00000002
	// SPIGetWheelScrollLines is SPI_GETWHEELSCROLLLINES.
	//
	// Flutter: no counterpart
	SPIGetWheelScrollLines = 0x0068
)

// The pointer input types (POINTER_INPUT_TYPE), the pointer flags (POINTER_FLAG_*) and the pen
// flags.
const (
	// PTPointer is PT_POINTER.
	//
	// Flutter: no counterpart
	PTPointer = 1
	// PTTouch is PT_TOUCH.
	//
	// Flutter: no counterpart
	PTTouch = 2
	// PTPen is PT_PEN.
	//
	// Flutter: no counterpart
	PTPen = 3
	// PTMouse is PT_MOUSE.
	//
	// Flutter: no counterpart
	PTMouse = 4
	// PTTouchpad is PT_TOUCHPAD.
	//
	// Flutter: no counterpart
	PTTouchpad = 5

	// PointerFlagNew is POINTER_FLAG_NEW.
	//
	// Flutter: no counterpart
	PointerFlagNew = 0x00000001
	// PointerFlagInRange is POINTER_FLAG_INRANGE.
	//
	// Flutter: no counterpart
	PointerFlagInRange = 0x00000002
	// PointerFlagInContact is POINTER_FLAG_INCONTACT.
	//
	// Flutter: no counterpart
	PointerFlagInContact = 0x00000004
	// PointerFlagFirstButton is POINTER_FLAG_FIRSTBUTTON.
	//
	// Flutter: no counterpart
	PointerFlagFirstButton = 0x00000010
	// PointerFlagSecondButton is POINTER_FLAG_SECONDBUTTON.
	//
	// Flutter: no counterpart
	PointerFlagSecondButton = 0x00000020
	// PointerFlagThirdButton is POINTER_FLAG_THIRDBUTTON.
	//
	// Flutter: no counterpart
	PointerFlagThirdButton = 0x00000040
	// PointerFlagCanceled is POINTER_FLAG_CANCELED.
	//
	// Flutter: no counterpart
	PointerFlagCanceled = 0x00008000
	// PointerFlagDown is POINTER_FLAG_DOWN.
	//
	// Flutter: no counterpart
	PointerFlagDown = 0x00010000
	// PointerFlagUpdate is POINTER_FLAG_UPDATE.
	//
	// Flutter: no counterpart
	PointerFlagUpdate = 0x00020000
	// PointerFlagUp is POINTER_FLAG_UP.
	//
	// Flutter: no counterpart
	PointerFlagUp = 0x00040000

	// PenFlagBarrel is PEN_FLAG_BARREL.
	//
	// Flutter: no counterpart
	PenFlagBarrel = 0x00000001
	// PenFlagInverted is PEN_FLAG_INVERTED.
	//
	// Flutter: no counterpart
	PenFlagInverted = 0x00000002
	// PenFlagEraser is PEN_FLAG_ERASER.
	//
	// Flutter: no counterpart
	PenFlagEraser = 0x00000004

	// FeedbackTouchContactVisualization is FEEDBACK_TOUCH_CONTACTVISUALIZATION.
	//
	// Flutter: no counterpart
	FeedbackTouchContactVisualization = 1
	// FeedbackPenBarrelVisualization is FEEDBACK_PEN_BARRELVISUALIZATION.
	//
	// Flutter: no counterpart
	FeedbackPenBarrelVisualization = 2
)

// The stock cursors (IDC_*): the numbers that LoadCursor takes as resource names. They are
// pointer-valued macros in the headers, so the layout test leaves them out, like IDCArrow.
const (
	// IDCIBeam is IDC_IBEAM.
	//
	// Flutter: no counterpart
	IDCIBeam = 32513
	// IDCWait is IDC_WAIT.
	//
	// Flutter: no counterpart
	IDCWait = 32514
	// IDCCross is IDC_CROSS.
	//
	// Flutter: no counterpart
	IDCCross = 32515
	// IDCSizeNWSE is IDC_SIZENWSE.
	//
	// Flutter: no counterpart
	IDCSizeNWSE = 32642
	// IDCSizeNESW is IDC_SIZENESW.
	//
	// Flutter: no counterpart
	IDCSizeNESW = 32643
	// IDCSizeWE is IDC_SIZEWE.
	//
	// Flutter: no counterpart
	IDCSizeWE = 32644
	// IDCSizeNS is IDC_SIZENS.
	//
	// Flutter: no counterpart
	IDCSizeNS = 32645
	// IDCSizeAll is IDC_SIZEALL.
	//
	// Flutter: no counterpart
	IDCSizeAll = 32646
	// IDCNo is IDC_NO.
	//
	// Flutter: no counterpart
	IDCNo = 32648
	// IDCHand is IDC_HAND.
	//
	// Flutter: no counterpart
	IDCHand = 32649
	// IDCAppStarting is IDC_APPSTARTING.
	//
	// Flutter: no counterpart
	IDCAppStarting = 32650
	// IDCHelp is IDC_HELP.
	//
	// Flutter: no counterpart
	IDCHelp = 32651
)

// The values of SendInput (INPUT_MOUSE and MOUSEEVENTF_*), which the tests use to move the mouse
// and press its buttons as a user does.
const (
	// InputMouse is INPUT_MOUSE.
	//
	// Flutter: no counterpart
	InputMouse = 0
	// MouseEventFMove is MOUSEEVENTF_MOVE.
	//
	// Flutter: no counterpart
	MouseEventFMove = 0x0001
	// MouseEventFLeftDown is MOUSEEVENTF_LEFTDOWN.
	//
	// Flutter: no counterpart
	MouseEventFLeftDown = 0x0002
	// MouseEventFLeftUp is MOUSEEVENTF_LEFTUP.
	//
	// Flutter: no counterpart
	MouseEventFLeftUp = 0x0004
	// MouseEventFRightDown is MOUSEEVENTF_RIGHTDOWN.
	//
	// Flutter: no counterpart
	MouseEventFRightDown = 0x0008
	// MouseEventFRightUp is MOUSEEVENTF_RIGHTUP.
	//
	// Flutter: no counterpart
	MouseEventFRightUp = 0x0010
	// MouseEventFWheel is MOUSEEVENTF_WHEEL.
	//
	// Flutter: no counterpart
	MouseEventFWheel = 0x0800
	// MouseEventFAbsolute is MOUSEEVENTF_ABSOLUTE.
	//
	// Flutter: no counterpart
	MouseEventFAbsolute = 0x8000
	// MouseEventFVirtualDesk is MOUSEEVENTF_VIRTUALDESK.
	//
	// Flutter: no counterpart
	MouseEventFVirtualDesk = 0x4000
)
