package win32

// The constants of user32, gdi32, dwmapi and the other libraries that the windows of step 28.2
// use. They carry no OS tag so that the layout test checks them against the headers. The name is
// the Go spelling of the macro of the header.

// The window styles (WS_*) and the extended window styles (WS_EX_*).
const (
	// WSOverlapped is WS_OVERLAPPED.
	//
	// Flutter: no counterpart
	WSOverlapped uint32 = 0x00000000
	// WSPopup is WS_POPUP.
	//
	// Flutter: no counterpart
	WSPopup uint32 = 0x80000000
	// WSVisible is WS_VISIBLE.
	//
	// Flutter: no counterpart
	WSVisible uint32 = 0x10000000
	// WSDisabled is WS_DISABLED.
	//
	// Flutter: no counterpart
	WSDisabled uint32 = 0x08000000
	// WSClipSiblings is WS_CLIPSIBLINGS.
	//
	// Flutter: no counterpart
	WSClipSiblings uint32 = 0x04000000
	// WSClipChildren is WS_CLIPCHILDREN.
	//
	// Flutter: no counterpart
	WSClipChildren uint32 = 0x02000000
	// WSMinimize is WS_MINIMIZE.
	//
	// Flutter: no counterpart
	WSMinimize uint32 = 0x20000000
	// WSMaximize is WS_MAXIMIZE.
	//
	// Flutter: no counterpart
	WSMaximize uint32 = 0x01000000
	// WSCaption is WS_CAPTION (WS_BORDER | WS_DLGFRAME).
	//
	// Flutter: no counterpart
	WSCaption uint32 = 0x00C00000
	// WSSysMenu is WS_SYSMENU.
	//
	// Flutter: no counterpart
	WSSysMenu uint32 = 0x00080000
	// WSThickFrame is WS_THICKFRAME.
	//
	// Flutter: no counterpart
	WSThickFrame uint32 = 0x00040000
	// WSMinimizeBox is WS_MINIMIZEBOX.
	//
	// Flutter: no counterpart
	WSMinimizeBox uint32 = 0x00020000
	// WSMaximizeBox is WS_MAXIMIZEBOX.
	//
	// Flutter: no counterpart
	WSMaximizeBox uint32 = 0x00010000
	// WSOverlappedWindow is WS_OVERLAPPEDWINDOW: the style of an ordinary window.
	//
	// Flutter: no counterpart
	WSOverlappedWindow uint32 = 0x00CF0000

	// WSExDlgModalFrame is WS_EX_DLGMODALFRAME.
	//
	// Flutter: no counterpart
	WSExDlgModalFrame uint32 = 0x00000001
	// WSExToolWindow is WS_EX_TOOLWINDOW.
	//
	// Flutter: no counterpart
	WSExToolWindow uint32 = 0x00000080
	// WSExWindowEdge is WS_EX_WINDOWEDGE.
	//
	// Flutter: no counterpart
	WSExWindowEdge uint32 = 0x00000100
	// WSExClientEdge is WS_EX_CLIENTEDGE.
	//
	// Flutter: no counterpart
	WSExClientEdge uint32 = 0x00000200
	// WSExStaticEdge is WS_EX_STATICEDGE.
	//
	// Flutter: no counterpart
	WSExStaticEdge uint32 = 0x00020000
	// WSExAppWindow is WS_EX_APPWINDOW.
	//
	// Flutter: no counterpart
	WSExAppWindow uint32 = 0x00040000
)

// The window messages (WM_*), the registered range and the values that go with them.
const (
	// WMNull is WM_NULL.
	//
	// Flutter: no counterpart
	WMNull = 0x0000
	// WMCreate is WM_CREATE.
	//
	// Flutter: no counterpart
	WMCreate = 0x0001
	// WMDestroy is WM_DESTROY.
	//
	// Flutter: no counterpart
	WMDestroy = 0x0002
	// WMSize is WM_SIZE.
	//
	// Flutter: no counterpart
	WMSize = 0x0005
	// WMActivate is WM_ACTIVATE.
	//
	// Flutter: no counterpart
	WMActivate = 0x0006
	// WMSetFocus is WM_SETFOCUS.
	//
	// Flutter: no counterpart
	WMSetFocus = 0x0007
	// WMKillFocus is WM_KILLFOCUS.
	//
	// Flutter: no counterpart
	WMKillFocus = 0x0008
	// WMPaint is WM_PAINT.
	//
	// Flutter: no counterpart
	WMPaint = 0x000F
	// WMClose is WM_CLOSE.
	//
	// Flutter: no counterpart
	WMClose = 0x0010
	// WMQueryEndSession is WM_QUERYENDSESSION.
	//
	// Flutter: no counterpart
	WMQueryEndSession = 0x0011
	// WMEraseBkgnd is WM_ERASEBKGND.
	//
	// Flutter: no counterpart
	WMEraseBkgnd = 0x0014
	// WMEndSession is WM_ENDSESSION.
	//
	// Flutter: no counterpart
	WMEndSession = 0x0016
	// WMSettingChange is WM_SETTINGCHANGE.
	//
	// Flutter: no counterpart
	WMSettingChange = 0x001A
	// WMShowWindow is WM_SHOWWINDOW.
	//
	// Flutter: no counterpart
	WMShowWindow = 0x0018
	// WMGetObject is WM_GETOBJECT: an accessibility client asks the window for an object (SPEC §25.11).
	//
	// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage
	WMGetObject = 0x003D
	// WMGetMinMaxInfo is WM_GETMINMAXINFO.
	//
	// Flutter: no counterpart
	WMGetMinMaxInfo = 0x0024
	// WMWindowPosChanged is WM_WINDOWPOSCHANGED.
	//
	// Flutter: no counterpart
	WMWindowPosChanged = 0x0047
	// WMDisplayChange is WM_DISPLAYCHANGE.
	//
	// Flutter: no counterpart
	WMDisplayChange = 0x007E
	// WMGetIcon is WM_GETICON.
	//
	// Flutter: no counterpart
	WMGetIcon = 0x007F
	// WMSetIcon is WM_SETICON.
	//
	// Flutter: no counterpart
	WMSetIcon = 0x0080
	// WMNCCreate is WM_NCCREATE.
	//
	// Flutter: no counterpart
	WMNCCreate = 0x0081
	// WMNCDestroy is WM_NCDESTROY.
	//
	// Flutter: no counterpart
	WMNCDestroy = 0x0082
	// WMNCCalcSize is WM_NCCALCSIZE.
	//
	// Flutter: no counterpart
	WMNCCalcSize = 0x0083
	// WMNCHitTest is WM_NCHITTEST.
	//
	// Flutter: no counterpart
	WMNCHitTest = 0x0084
	// WMNCLButtonDown is WM_NCLBUTTONDOWN.
	//
	// Flutter: no counterpart
	WMNCLButtonDown = 0x00A1
	// WMSysCommand is WM_SYSCOMMAND.
	//
	// Flutter: no counterpart
	WMSysCommand = 0x0112
	// WMTimer is WM_TIMER.
	//
	// Flutter: no counterpart
	WMTimer = 0x0113
	// WMEnterMenuLoop is WM_ENTERMENULOOP.
	//
	// Flutter: no counterpart
	WMEnterMenuLoop = 0x0211
	// WMExitMenuLoop is WM_EXITMENULOOP.
	//
	// Flutter: no counterpart
	WMExitMenuLoop = 0x0212
	// WMEnterSizeMove is WM_ENTERSIZEMOVE.
	//
	// Flutter: no counterpart
	WMEnterSizeMove = 0x0231
	// WMExitSizeMove is WM_EXITSIZEMOVE.
	//
	// Flutter: no counterpart
	WMExitSizeMove = 0x0232
	// WMDpiChanged is WM_DPICHANGED.
	//
	// Flutter: no counterpart
	WMDpiChanged = 0x02E0
	// WMDwmColorizationColorChanged is WM_DWMCOLORIZATIONCOLORCHANGED.
	//
	// Flutter: no counterpart
	WMDwmColorizationColorChanged = 0x0320
	// WMApp is WM_APP: the first message number an app may use for its own.
	//
	// Flutter: no counterpart
	WMApp = 0x8000

	// SizeRestored is SIZE_RESTORED, the wParam of WM_SIZE for a window that is neither minimized nor maximized.
	//
	// Flutter: no counterpart
	SizeRestored = 0
	// SizeMinimized is SIZE_MINIMIZED.
	//
	// Flutter: no counterpart
	SizeMinimized = 1
	// SizeMaximized is SIZE_MAXIMIZED.
	//
	// Flutter: no counterpart
	SizeMaximized = 2
	// SizeMaxShow is SIZE_MAXSHOW.
	//
	// Flutter: no counterpart
	SizeMaxShow = 3
	// SizeMaxHide is SIZE_MAXHIDE.
	//
	// Flutter: no counterpart
	SizeMaxHide = 4

	// WAInactive is WA_INACTIVE, the low word of the wParam of WM_ACTIVATE for a window that is deactivated.
	//
	// Flutter: no counterpart
	WAInactive = 0
	// WAActive is WA_ACTIVE.
	//
	// Flutter: no counterpart
	WAActive = 1
	// WAClickActive is WA_CLICKACTIVE.
	//
	// Flutter: no counterpart
	WAClickActive = 2

	// IconSmall is ICON_SMALL, the wParam of WM_SETICON for the small icon.
	//
	// Flutter: no counterpart
	IconSmall = 0
	// IconBig is ICON_BIG.
	//
	// Flutter: no counterpart
	IconBig = 1
)

// The results of WM_NCHITTEST (HT*), and the commands of WM_SYSCOMMAND (SC_*).
const (
	// HTClient is HTCLIENT.
	//
	// Flutter: no counterpart
	HTClient = 1
	// HTCaption is HTCAPTION.
	//
	// Flutter: no counterpart
	HTCaption = 2
	// HTSysMenu is HTSYSMENU.
	//
	// Flutter: no counterpart
	HTSysMenu = 3
	// HTLeft is HTLEFT.
	//
	// Flutter: no counterpart
	HTLeft = 10
	// HTRight is HTRIGHT.
	//
	// Flutter: no counterpart
	HTRight = 11
	// HTTop is HTTOP.
	//
	// Flutter: no counterpart
	HTTop = 12
	// HTTopLeft is HTTOPLEFT.
	//
	// Flutter: no counterpart
	HTTopLeft = 13
	// HTTopRight is HTTOPRIGHT.
	//
	// Flutter: no counterpart
	HTTopRight = 14
	// HTBottom is HTBOTTOM.
	//
	// Flutter: no counterpart
	HTBottom = 15
	// HTBottomLeft is HTBOTTOMLEFT.
	//
	// Flutter: no counterpart
	HTBottomLeft = 16
	// HTBottomRight is HTBOTTOMRIGHT.
	//
	// Flutter: no counterpart
	HTBottomRight = 17

	// SCMove is SC_MOVE.
	//
	// Flutter: no counterpart
	SCMove = 0xF010
	// SCMinimize is SC_MINIMIZE.
	//
	// Flutter: no counterpart
	SCMinimize = 0xF020
	// SCMaximize is SC_MAXIMIZE.
	//
	// Flutter: no counterpart
	SCMaximize = 0xF030
	// SCClose is SC_CLOSE.
	//
	// Flutter: no counterpart
	SCClose = 0xF060
	// SCRestore is SC_RESTORE.
	//
	// Flutter: no counterpart
	SCRestore = 0xF120
)

// The arguments of ShowWindow (SW_*) and SetWindowPos (SWP_*), and the special window handles.
const (
	// SWHide is SW_HIDE.
	//
	// Flutter: no counterpart
	SWHide = 0
	// SWShowNormal is SW_SHOWNORMAL.
	//
	// Flutter: no counterpart
	SWShowNormal = 1
	// SWShowMinimized is SW_SHOWMINIMIZED.
	//
	// Flutter: no counterpart
	SWShowMinimized = 2
	// SWShowMaximized is SW_SHOWMAXIMIZED, which is also SW_MAXIMIZE.
	//
	// Flutter: no counterpart
	SWShowMaximized = 3
	// SWShowNoActivate is SW_SHOWNOACTIVATE.
	//
	// Flutter: no counterpart
	SWShowNoActivate = 4
	// SWShow is SW_SHOW.
	//
	// Flutter: no counterpart
	SWShow = 5
	// SWMinimize is SW_MINIMIZE.
	//
	// Flutter: no counterpart
	SWMinimize = 6
	// SWShowMinNoActive is SW_SHOWMINNOACTIVE.
	//
	// Flutter: no counterpart
	SWShowMinNoActive = 7
	// SWRestore is SW_RESTORE.
	//
	// Flutter: no counterpart
	SWRestore = 9

	// SWPNoSize is SWP_NOSIZE.
	//
	// Flutter: no counterpart
	SWPNoSize = 0x0001
	// SWPNoMove is SWP_NOMOVE.
	//
	// Flutter: no counterpart
	SWPNoMove = 0x0002
	// SWPNoZOrder is SWP_NOZORDER.
	//
	// Flutter: no counterpart
	SWPNoZOrder = 0x0004
	// SWPNoActivate is SWP_NOACTIVATE.
	//
	// Flutter: no counterpart
	SWPNoActivate = 0x0010
	// SWPFrameChanged is SWP_FRAMECHANGED.
	//
	// Flutter: no counterpart
	SWPFrameChanged = 0x0020
	// SWPShowWindow is SWP_SHOWWINDOW.
	//
	// Flutter: no counterpart
	SWPShowWindow = 0x0040
	// SWPNoOwnerZOrder is SWP_NOOWNERZORDER.
	//
	// Flutter: no counterpart
	SWPNoOwnerZOrder = 0x0200

	// CWUseDefault is CW_USEDEFAULT, as the unsigned value the header writes.
	//
	// Flutter: no counterpart
	CWUseDefault uint32 = 0x80000000
)

// The values of GetWindowLongPtr, GetSystemMetrics, the monitors and the other user32 functions.
// A negative index is the unsigned number the header's macro stands for, as a uintptr of a call.
const (
	// GWLStyle is GWL_STYLE (-16).
	//
	// Flutter: no counterpart
	GWLStyle = -16
	// GWLExStyle is GWL_EXSTYLE (-20).
	//
	// Flutter: no counterpart
	GWLExStyle = -20
	// GWLPHwndParent is GWLP_HWNDPARENT (-8): the owner of a top-level window.
	//
	// Flutter: no counterpart
	GWLPHwndParent = -8
	// GWLPUserData is GWLP_USERDATA (-21).
	//
	// Flutter: no counterpart
	GWLPUserData = -21

	// SMCxSizeFrame is SM_CXSIZEFRAME.
	//
	// Flutter: no counterpart
	SMCxSizeFrame = 32
	// SMCySizeFrame is SM_CYSIZEFRAME.
	//
	// Flutter: no counterpart
	SMCySizeFrame = 33
	// SMCxIcon is SM_CXICON.
	//
	// Flutter: no counterpart
	SMCxIcon = 11
	// SMCyIcon is SM_CYICON.
	//
	// Flutter: no counterpart
	SMCyIcon = 12
	// SMCxSmIcon is SM_CXSMICON.
	//
	// Flutter: no counterpart
	SMCxSmIcon = 49
	// SMCySmIcon is SM_CYSMICON.
	//
	// Flutter: no counterpart
	SMCySmIcon = 50
	// SMXVirtualScreen is SM_XVIRTUALSCREEN.
	//
	// Flutter: no counterpart
	SMXVirtualScreen = 76
	// SMCxVirtualScreen is SM_CXVIRTUALSCREEN.
	//
	// Flutter: no counterpart
	SMCxVirtualScreen = 78
	// SMCyVirtualScreen is SM_CYVIRTUALSCREEN.
	//
	// Flutter: no counterpart
	SMCyVirtualScreen = 79
	// SMCxPaddedBorder is SM_CXPADDEDBORDER.
	//
	// Flutter: no counterpart
	SMCxPaddedBorder = 92

	// MonitorDefaultToNull is MONITOR_DEFAULTTONULL.
	//
	// Flutter: no counterpart
	MonitorDefaultToNull = 0
	// MonitorDefaultToPrimary is MONITOR_DEFAULTTOPRIMARY.
	//
	// Flutter: no counterpart
	MonitorDefaultToPrimary = 1
	// MonitorDefaultToNearest is MONITOR_DEFAULTTONEAREST.
	//
	// Flutter: no counterpart
	MonitorDefaultToNearest = 2
	// MonitorInfoFPrimary is MONITORINFOF_PRIMARY.
	//
	// Flutter: no counterpart
	MonitorInfoFPrimary = 1
	// MDTEffectiveDPI is MDT_EFFECTIVE_DPI.
	//
	// Flutter: no counterpart
	MDTEffectiveDPI = 0
	// EnumCurrentSettings is ENUM_CURRENT_SETTINGS (-1), as the unsigned number the header writes.
	//
	// Flutter: no counterpart
	EnumCurrentSettings uint32 = 0xFFFFFFFF
	// UserDefaultScreenDPI is USER_DEFAULT_SCREEN_DPI: the DPI of a scale of 1.
	//
	// Flutter: no counterpart
	UserDefaultScreenDPI = 96

	// PMNoRemove is PM_NOREMOVE.
	//
	// Flutter: no counterpart
	PMNoRemove = 0x0000
	// PMRemove is PM_REMOVE.
	//
	// Flutter: no counterpart
	PMRemove = 0x0001
	// QSAllInput is QS_ALLINPUT as Windows 8 and later define it, with QS_TOUCH and QS_POINTER, so
	// that the wait of the message loop ends for the pointer messages of touch and pen.
	//
	// Flutter: no counterpart
	QSAllInput = 0x1CFF
	// MWMOInputAvailable is MWMO_INPUTAVAILABLE.
	//
	// Flutter: no counterpart
	MWMOInputAvailable = 0x0004
	// IDCArrow is IDC_ARROW, the resource number of the arrow cursor.
	//
	// Flutter: no counterpart
	IDCArrow = 32512
	// TPMReturnCmd is TPM_RETURNCMD.
	//
	// Flutter: no counterpart
	TPMReturnCmd = 0x0100
)

// The values of the other libraries: DWM, GDI, the registry, the execution state and the
// waitable timers.
const (
	// DWMWAUseImmersiveDarkMode is DWMWA_USE_IMMERSIVE_DARK_MODE (Windows 10 version 2004 and 11).
	//
	// Flutter: no counterpart
	DWMWAUseImmersiveDarkMode = 20
	// DWMWAUseImmersiveDarkModeBefore20H1 is the value 19 that the same attribute had in versions 1809 to 1909.
	//
	// Flutter: no counterpart
	DWMWAUseImmersiveDarkModeBefore20H1 = 19
	// DWMWAExtendedFrameBounds is DWMWA_EXTENDED_FRAME_BOUNDS.
	//
	// Flutter: no counterpart
	DWMWAExtendedFrameBounds = 9

	// BIRGB is BI_RGB.
	//
	// Flutter: no counterpart
	BIRGB = 0
	// DIBRGBColors is DIB_RGB_COLORS.
	//
	// Flutter: no counterpart
	DIBRGBColors = 0
	// SrcCopy is SRCCOPY.
	//
	// Flutter: no counterpart
	SrcCopy = 0x00CC0020

	// HKeyCurrentUser is HKEY_CURRENT_USER.
	//
	// Flutter: no counterpart
	HKeyCurrentUser uintptr = 0x80000001
	// RRFRTRegDword is RRF_RT_REG_DWORD.
	//
	// Flutter: no counterpart
	RRFRTRegDword = 0x00000010

	// ESContinuous is ES_CONTINUOUS.
	//
	// Flutter: no counterpart
	ESContinuous uint32 = 0x80000000
	// ESDisplayRequired is ES_DISPLAY_REQUIRED.
	//
	// Flutter: no counterpart
	ESDisplayRequired uint32 = 0x00000002

	// CreateWaitableTimerHighResolution is CREATE_WAITABLE_TIMER_HIGH_RESOLUTION.
	//
	// Flutter: no counterpart
	CreateWaitableTimerHighResolution = 0x00000002
	// TimerAllAccess is TIMER_ALL_ACCESS.
	//
	// Flutter: no counterpart
	TimerAllAccess = 0x001F0003
	// AttachParentProcess is ATTACH_PARENT_PROCESS (-1), as the unsigned number the header writes.
	//
	// Flutter: no counterpart
	AttachParentProcess uint32 = 0xFFFFFFFF
	// StdOutputHandle is STD_OUTPUT_HANDLE (-11).
	//
	// Flutter: no counterpart
	StdOutputHandle = -11
	// StdErrorHandle is STD_ERROR_HANDLE (-12).
	//
	// Flutter: no counterpart
	StdErrorHandle = -12
)

// DPIAwarenessContextPerMonitorAwareV2 is DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2, the handle (-4)
// that SetProcessDpiAwarenessContext takes.
//
// Flutter: no counterpart
const DPIAwarenessContextPerMonitorAwareV2 = ^uintptr(3)

// HWNDTopmost is HWND_TOPMOST ((HWND)-1) and HWNDMessage is HWND_MESSAGE ((HWND)-3).
const (
	// HWNDTopmost is HWND_TOPMOST.
	//
	// Flutter: no counterpart
	HWNDTopmost = ^uintptr(0)
	// HWNDMessage is HWND_MESSAGE: the parent of a message-only window.
	//
	// Flutter: no counterpart
	HWNDMessage = ^uintptr(2)
)
