package win32

// The constants of the keyboard of step 28.4: the key messages, the virtual keys that the keyboard
// code names, the parameters of MapVirtualKey and SendInput, and the keyboard layouts. They carry no
// OS tag so that the layout test checks them against the headers (SPEC §25.2). The name is the Go
// spelling of the macro.

// The messages of the keyboard and the input language.
const (
	// WMKeyFirst is WM_KEYFIRST.
	//
	// Flutter: no counterpart
	WMKeyFirst = 0x0100
	// WMKeyDown is WM_KEYDOWN.
	//
	// Flutter: no counterpart
	WMKeyDown = 0x0100
	// WMKeyUp is WM_KEYUP.
	//
	// Flutter: no counterpart
	WMKeyUp = 0x0101
	// WMChar is WM_CHAR.
	//
	// Flutter: no counterpart
	WMChar = 0x0102
	// WMDeadChar is WM_DEADCHAR.
	//
	// Flutter: no counterpart
	WMDeadChar = 0x0103
	// WMSysKeyDown is WM_SYSKEYDOWN.
	//
	// Flutter: no counterpart
	WMSysKeyDown = 0x0104
	// WMSysKeyUp is WM_SYSKEYUP.
	//
	// Flutter: no counterpart
	WMSysKeyUp = 0x0105
	// WMSysChar is WM_SYSCHAR.
	//
	// Flutter: no counterpart
	WMSysChar = 0x0106
	// WMSysDeadChar is WM_SYSDEADCHAR.
	//
	// Flutter: no counterpart
	WMSysDeadChar = 0x0107
	// WMUniChar is WM_UNICHAR.
	//
	// Flutter: no counterpart
	WMUniChar = 0x0109
	// WMKeyLast is WM_KEYLAST.
	//
	// Flutter: no counterpart
	WMKeyLast = 0x0109
	// WMInputLangChange is WM_INPUTLANGCHANGE.
	//
	// Flutter: no counterpart
	WMInputLangChange = 0x0051
)

// The virtual-key codes that the keyboard code names (the others come from Flutter's tables).
const (
	// VKBack is VK_BACK.
	//
	// Flutter: no counterpart
	VKBack = 0x08
	// VKTab is VK_TAB.
	//
	// Flutter: no counterpart
	VKTab = 0x09
	// VKReturn is VK_RETURN.
	//
	// Flutter: no counterpart
	VKReturn = 0x0D
	// VKShift is VK_SHIFT.
	//
	// Flutter: no counterpart
	VKShift = 0x10
	// VKControl is VK_CONTROL.
	//
	// Flutter: no counterpart
	VKControl = 0x11
	// VKMenu is VK_MENU.
	//
	// Flutter: no counterpart
	VKMenu = 0x12
	// VKCapital is VK_CAPITAL.
	//
	// Flutter: no counterpart
	VKCapital = 0x14
	// VKEscape is VK_ESCAPE.
	//
	// Flutter: no counterpart
	VKEscape = 0x1B
	// VKSpace is VK_SPACE.
	//
	// Flutter: no counterpart
	VKSpace = 0x20
	// VKPrior is VK_PRIOR.
	//
	// Flutter: no counterpart
	VKPrior = 0x21
	// VKNext is VK_NEXT.
	//
	// Flutter: no counterpart
	VKNext = 0x22
	// VKEnd is VK_END.
	//
	// Flutter: no counterpart
	VKEnd = 0x23
	// VKHome is VK_HOME.
	//
	// Flutter: no counterpart
	VKHome = 0x24
	// VKLeft is VK_LEFT.
	//
	// Flutter: no counterpart
	VKLeft = 0x25
	// VKUp is VK_UP.
	//
	// Flutter: no counterpart
	VKUp = 0x26
	// VKRight is VK_RIGHT.
	//
	// Flutter: no counterpart
	VKRight = 0x27
	// VKDown is VK_DOWN.
	//
	// Flutter: no counterpart
	VKDown = 0x28
	// VKInsert is VK_INSERT.
	//
	// Flutter: no counterpart
	VKInsert = 0x2D
	// VKDelete is VK_DELETE.
	//
	// Flutter: no counterpart
	VKDelete = 0x2E
	// VKLWin is VK_LWIN.
	//
	// Flutter: no counterpart
	VKLWin = 0x5B
	// VKRWin is VK_RWIN.
	//
	// Flutter: no counterpart
	VKRWin = 0x5C
	// VKApps is VK_APPS.
	//
	// Flutter: no counterpart
	VKApps = 0x5D
	// VKNumpad0 is VK_NUMPAD0.
	//
	// Flutter: no counterpart
	VKNumpad0 = 0x60
	// VKNumpad1 is VK_NUMPAD1.
	//
	// Flutter: no counterpart
	VKNumpad1 = 0x61
	// VKDivide is VK_DIVIDE.
	//
	// Flutter: no counterpart
	VKDivide = 0x6F
	// VKF4 is VK_F4.
	//
	// Flutter: no counterpart
	VKF4 = 0x73
	// VKF10 is VK_F10.
	//
	// Flutter: no counterpart
	VKF10 = 0x79
	// VKNumLock is VK_NUMLOCK.
	//
	// Flutter: no counterpart
	VKNumLock = 0x90
	// VKScroll is VK_SCROLL.
	//
	// Flutter: no counterpart
	VKScroll = 0x91
	// VKLShift is VK_LSHIFT.
	//
	// Flutter: no counterpart
	VKLShift = 0xA0
	// VKRShift is VK_RSHIFT.
	//
	// Flutter: no counterpart
	VKRShift = 0xA1
	// VKLControl is VK_LCONTROL.
	//
	// Flutter: no counterpart
	VKLControl = 0xA2
	// VKRControl is VK_RCONTROL.
	//
	// Flutter: no counterpart
	VKRControl = 0xA3
	// VKLMenu is VK_LMENU.
	//
	// Flutter: no counterpart
	VKLMenu = 0xA4
	// VKRMenu is VK_RMENU.
	//
	// Flutter: no counterpart
	VKRMenu = 0xA5
	// VKProcessKey is VK_PROCESSKEY.
	//
	// Flutter: no counterpart
	VKProcessKey = 0xE5
	// VKPacket is VK_PACKET.
	//
	// Flutter: no counterpart
	VKPacket = 0xE7
)

// The parameters of MapVirtualKey, SendInput (keyboard input) and LoadKeyboardLayout.
const (
	// MapVKVKToVSC is MAPVK_VK_TO_VSC.
	//
	// Flutter: no counterpart
	MapVKVKToVSC = 0
	// MapVKVSCToVK is MAPVK_VSC_TO_VK.
	//
	// Flutter: no counterpart
	MapVKVSCToVK = 1
	// MapVKVKToChar is MAPVK_VK_TO_CHAR.
	//
	// Flutter: no counterpart
	MapVKVKToChar = 2
	// MapVKVSCToVKEx is MAPVK_VSC_TO_VK_EX.
	//
	// Flutter: no counterpart
	MapVKVSCToVKEx = 3
	// MapVKVKToVSCEx is MAPVK_VK_TO_VSC_EX.
	//
	// Flutter: no counterpart
	MapVKVKToVSCEx = 4
	// InputKeyboard is INPUT_KEYBOARD.
	//
	// Flutter: no counterpart
	InputKeyboard = 1
	// KeyEventFExtendedKey is KEYEVENTF_EXTENDEDKEY.
	//
	// Flutter: no counterpart
	KeyEventFExtendedKey = 0x0001
	// KeyEventFKeyUp is KEYEVENTF_KEYUP.
	//
	// Flutter: no counterpart
	KeyEventFKeyUp = 0x0002
	// KeyEventFUnicode is KEYEVENTF_UNICODE.
	//
	// Flutter: no counterpart
	KeyEventFUnicode = 0x0004
	// KeyEventFScanCode is KEYEVENTF_SCANCODE.
	//
	// Flutter: no counterpart
	KeyEventFScanCode = 0x0008
	// KLFActivate is KLF_ACTIVATE.
	//
	// Flutter: no counterpart
	KLFActivate = 0x00000001
	// KLFSubstituteOK is KLF_SUBSTITUTE_OK.
	//
	// Flutter: no counterpart
	KLFSubstituteOK = 0x00000002
)
