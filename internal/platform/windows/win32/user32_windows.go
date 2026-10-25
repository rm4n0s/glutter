//go:build windows

package win32

import (
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The bindings of user32.dll that the windows and the message loop of Windows need (SPEC §25.3,
// §25.4). A function that fails returns a tagged error (Win32CallFailed, with the name of the
// function and the Win32 error code); one whose result is a plain answer returns it.

var user32 = &dllTable{name: "user32.dll"}

// RegisterClassEx registers a window class (RegisterClassExW) and returns its atom.
//
// Flutter: no counterpart
func RegisterClassEx(wc *WndClassEx) (uint16, error) {
	r, errno, err := user32.call("RegisterClassExW", uintptr(unsafe.Pointer(wc)))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagWin32WindowClassFailed, "RegisterClassExW failed", "code", int(errno))
	}
	return uint16(r), nil
}

// UnregisterClass removes a window class that no window uses any more.
//
// Flutter: no counterpart
func UnregisterClass(class string, instance uintptr) error {
	p, err := UTF16PtrFromString(class)
	if err != nil {
		return err
	}
	r, errno, err := user32.call("UnregisterClassW", uintptr(unsafe.Pointer(p)), instance)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("UnregisterClassW", uint32(errno))
	}
	return nil
}

// CreateWindowEx makes a window (CreateWindowExW). The messages that arrive before it returns
// (WM_NCCREATE, WM_CREATE, WM_GETMINMAXINFO, WM_SIZE) go to the window procedure of the class,
// with param in the CreateStruct of the first two.
//
// Flutter: no counterpart
func CreateWindowEx(exStyle uint32, class, title string, style uint32, x, y, w, h int32, parent, menu, instance, param uintptr) (uintptr, error) {
	pc, err := UTF16PtrFromString(class)
	if err != nil {
		return 0, err
	}
	pt, err := UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}
	r, errno, err := user32.call("CreateWindowExW", uintptr(exStyle), uintptr(unsafe.Pointer(pc)), uintptr(unsafe.Pointer(pt)),
		uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, menu, instance, param)
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagWin32WindowCreateFailed, "CreateWindowExW failed", "class", class, "code", int(errno))
	}
	return r, nil
}

// DestroyWindow destroys a window of the calling thread.
//
// Flutter: no counterpart
func DestroyWindow(hwnd uintptr) error {
	r, errno, err := user32.call("DestroyWindow", hwnd)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("DestroyWindow", uint32(errno))
	}
	return nil
}

// ShowWindow shows or hides a window (SW_*) and reports whether it was visible before.
//
// Flutter: no counterpart
func ShowWindow(hwnd uintptr, cmd int) bool {
	r, _, err := user32.call("ShowWindow", hwnd, uintptr(cmd))
	return err == nil && r != 0
}

// SetWindowPos moves, sizes or reorders a window.
//
// Flutter: no counterpart
func SetWindowPos(hwnd, insertAfter uintptr, x, y, cx, cy int32, flags uint32) error {
	r, errno, err := user32.call("SetWindowPos", hwnd, insertAfter, uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(flags))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("SetWindowPos", uint32(errno))
	}
	return nil
}

// GetWindowRect is the outer rectangle of a window in screen coordinates.
//
// Flutter: no counterpart
func GetWindowRect(hwnd uintptr) (Rect, error) {
	var rc Rect
	r, errno, err := user32.call("GetWindowRect", hwnd, uintptr(unsafe.Pointer(&rc)))
	if err != nil {
		return Rect{}, err
	}
	if r == 0 {
		return Rect{}, Win32Error("GetWindowRect", uint32(errno))
	}
	return rc, nil
}

// GetClientRect is the client rectangle of a window, whose top left is 0,0.
//
// Flutter: no counterpart
func GetClientRect(hwnd uintptr) (Rect, error) {
	var rc Rect
	r, errno, err := user32.call("GetClientRect", hwnd, uintptr(unsafe.Pointer(&rc)))
	if err != nil {
		return Rect{}, err
	}
	if r == 0 {
		return Rect{}, Win32Error("GetClientRect", uint32(errno))
	}
	return rc, nil
}

// SetWindowText sets the title of a window (SetWindowTextW).
//
// Flutter: no counterpart
func SetWindowText(hwnd uintptr, title string) error {
	p, err := UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	r, errno, err := user32.call("SetWindowTextW", hwnd, uintptr(unsafe.Pointer(p)))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("SetWindowTextW", uint32(errno))
	}
	return nil
}

// DefWindowProc is the default handling of a window message.
//
// Flutter: no counterpart
func DefWindowProc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	r, _, err := user32.call("DefWindowProcW", hwnd, uintptr(msg), wparam, lparam)
	if err != nil {
		return 0
	}
	return r
}

// PostMessage puts a message in the queue of the thread that made the window and returns at once.
// It is safe from any thread.
//
// Flutter: no counterpart
func PostMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) error {
	r, errno, err := user32.call("PostMessageW", hwnd, uintptr(msg), wparam, lparam)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("PostMessageW", uint32(errno))
	}
	return nil
}

// SendMessage sends a message to a window and waits for the window procedure to answer.
//
// Flutter: no counterpart
func SendMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	r, _, err := user32.call("SendMessageW", hwnd, uintptr(msg), wparam, lparam)
	if err != nil {
		return 0
	}
	return r
}

// PeekMessage takes the next message of the thread's queue (remove is PMRemove) without waiting,
// and reports whether there was one.
//
// Flutter: no counterpart
func PeekMessage(m *Msg, hwnd uintptr, min, max, remove uint32) bool {
	r, _, err := user32.call("PeekMessageW", uintptr(unsafe.Pointer(m)), hwnd, uintptr(min), uintptr(max), uintptr(remove))
	return err == nil && r != 0
}

// TranslateMessage turns key messages into character messages.
//
// Flutter: no counterpart
func TranslateMessage(m *Msg) bool {
	r, _, err := user32.call("TranslateMessage", uintptr(unsafe.Pointer(m)))
	return err == nil && r != 0
}

// DispatchMessage sends the message to the window procedure of its window.
//
// Flutter: no counterpart
func DispatchMessage(m *Msg) uintptr {
	r, _, err := user32.call("DispatchMessageW", uintptr(unsafe.Pointer(m)))
	if err != nil {
		return 0
	}
	return r
}

// MsgWaitForMultipleObjectsEx waits until a handle is signaled, a message of the kinds in mask is
// in the queue, or the timeout has passed (WaitInfinite never ends). The result is WaitObject0 +
// the index of the handle, WaitObject0 + len(handles) for a message, or WaitTimeout.
//
// Flutter: no counterpart
func MsgWaitForMultipleObjectsEx(handles []uintptr, timeout time.Duration, mask, flags uint32) (WaitResult, error) {
	var first uintptr
	if len(handles) > 0 {
		first = uintptr(unsafe.Pointer(&handles[0]))
	}
	r, errno, err := user32.call("MsgWaitForMultipleObjectsEx", uintptr(len(handles)), first, uintptr(timeoutMillis(timeout)), uintptr(mask), uintptr(flags))
	if err != nil {
		return 0, err
	}
	if WaitResult(r) == waitFailed {
		return 0, Win32Error("MsgWaitForMultipleObjectsEx", uint32(errno))
	}
	return WaitResult(r), nil
}

// GetWindowLongPtr reads a word of a window: GWLStyle, GWLExStyle, GWLPUserData.
//
// Flutter: no counterpart
func GetWindowLongPtr(hwnd uintptr, index int) uintptr {
	r, _, err := user32.call("GetWindowLongPtrW", hwnd, uintptr(index))
	if err != nil {
		return 0
	}
	return r
}

// SetWindowLongPtr writes a word of a window and returns the old one.
//
// Flutter: no counterpart
func SetWindowLongPtr(hwnd uintptr, index int, value uintptr) uintptr {
	r, _, err := user32.call("SetWindowLongPtrW", hwnd, uintptr(index), value)
	if err != nil {
		return 0
	}
	return r
}

// GetWindowPlacement reads where a window is when it is not maximized or minimized, and how it is
// shown.
//
// Flutter: no counterpart
func GetWindowPlacement(hwnd uintptr) (WindowPlacement, error) {
	wp := WindowPlacement{Length: uint32(unsafe.Sizeof(WindowPlacement{}))}
	r, errno, err := user32.call("GetWindowPlacement", hwnd, uintptr(unsafe.Pointer(&wp)))
	if err != nil {
		return WindowPlacement{}, err
	}
	if r == 0 {
		return WindowPlacement{}, Win32Error("GetWindowPlacement", uint32(errno))
	}
	return wp, nil
}

// SetWindowPlacement puts a window where the placement says.
//
// Flutter: no counterpart
func SetWindowPlacement(hwnd uintptr, wp WindowPlacement) error {
	wp.Length = uint32(unsafe.Sizeof(wp))
	r, errno, err := user32.call("SetWindowPlacement", hwnd, uintptr(unsafe.Pointer(&wp)))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("SetWindowPlacement", uint32(errno))
	}
	return nil
}

// IsIconic reports whether the window is minimized.
//
// Flutter: no counterpart
func IsIconic(hwnd uintptr) bool { return boolCall(user32, "IsIconic", hwnd) }

// IsZoomed reports whether the window is maximized.
//
// Flutter: no counterpart
func IsZoomed(hwnd uintptr) bool { return boolCall(user32, "IsZoomed", hwnd) }

// IsWindowVisible reports whether the window has the WS_VISIBLE style (and its parents do).
//
// Flutter: no counterpart
func IsWindowVisible(hwnd uintptr) bool { return boolCall(user32, "IsWindowVisible", hwnd) }

// IsWindow reports whether the handle names a window that exists.
//
// Flutter: no counterpart
func IsWindow(hwnd uintptr) bool { return boolCall(user32, "IsWindow", hwnd) }

// EnableWindow enables or disables the input of a window and reports whether it was disabled.
//
// Flutter: no counterpart
func EnableWindow(hwnd uintptr, enable bool) bool {
	return boolCall(user32, "EnableWindow", hwnd, boolInt(enable))
}

// SetForegroundWindow asks for the window to become the one the user works in.
//
// Flutter: no counterpart
func SetForegroundWindow(hwnd uintptr) bool { return boolCall(user32, "SetForegroundWindow", hwnd) }

// ReleaseCapture gives up the mouse capture of the thread.
//
// Flutter: no counterpart
func ReleaseCapture() bool { return boolCall(user32, "ReleaseCapture") }

// boolCall calls a function that returns a BOOL and reports whether it is true.
func boolCall(t *dllTable, name string, args ...uintptr) bool {
	r, _, err := t.call(name, args...)
	return err == nil && r != 0
}

// GetDC gets the device context of the client area of a window; ReleaseDC gives it back.
//
// Flutter: no counterpart
func GetDC(hwnd uintptr) (uintptr, error) {
	r, errno, err := user32.call("GetDC", hwnd)
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("GetDC", uint32(errno))
	}
	return r, nil
}

// ReleaseDC gives back a device context from GetDC.
//
// Flutter: no counterpart
func ReleaseDC(hwnd, dc uintptr) { _, _, _ = user32.call("ReleaseDC", hwnd, dc) }

// BeginPaint starts the painting of WM_PAINT: the device context is in the Hdc of the result and
// the invalid rectangle in Paint. EndPaint ends it, which also validates the window.
//
// Flutter: no counterpart
func BeginPaint(hwnd uintptr) (PaintStruct, error) {
	var ps PaintStruct
	r, errno, err := user32.call("BeginPaint", hwnd, uintptr(unsafe.Pointer(&ps)))
	if err != nil {
		return PaintStruct{}, err
	}
	if r == 0 {
		return PaintStruct{}, Win32Error("BeginPaint", uint32(errno))
	}
	return ps, nil
}

// EndPaint ends the painting that BeginPaint started.
//
// Flutter: no counterpart
func EndPaint(hwnd uintptr, ps *PaintStruct) {
	_, _, _ = user32.call("EndPaint", hwnd, uintptr(unsafe.Pointer(ps)))
}

// MonitorFromWindow is the monitor a window is on (MonitorDefaultToNearest and the others say
// what to do for a window that is on none).
//
// Flutter: no counterpart
func MonitorFromWindow(hwnd uintptr, flags uint32) uintptr {
	r, _, err := user32.call("MonitorFromWindow", hwnd, uintptr(flags))
	if err != nil {
		return 0
	}
	return r
}

// MonitorFromPoint is the monitor that holds a point of the virtual screen.
//
// Flutter: no counterpart
func MonitorFromPoint(pt Point, flags uint32) uintptr {
	// A POINT is 8 bytes, so it goes in one register: the x in the low half, the y in the high.
	r, _, err := user32.call("MonitorFromPoint", uintptr(uint32(pt.X))|uintptr(uint32(pt.Y))<<32, uintptr(flags))
	if err != nil {
		return 0
	}
	return r
}

// GetMonitorInfo reads the rectangle, the work area, the primary flag and the device name of a
// monitor.
//
// Flutter: no counterpart
func GetMonitorInfo(monitor uintptr) (MonitorInfoEx, error) {
	mi := MonitorInfoEx{Size: uint32(unsafe.Sizeof(MonitorInfoEx{}))}
	r, errno, err := user32.call("GetMonitorInfoW", monitor, uintptr(unsafe.Pointer(&mi)))
	if err != nil {
		return MonitorInfoEx{}, err
	}
	if r == 0 {
		return MonitorInfoEx{}, Win32Error("GetMonitorInfoW", uint32(errno))
	}
	return mi, nil
}

// enumMonitors is the callback of EnumDisplayMonitors, made once; the function of the call in
// progress is found by the token in its lParam, because the callbacks of a process are made once
// at startup (SPEC §25.2).
var enumMonitors = struct {
	once     sync.Once
	callback uintptr
	mu       sync.Mutex
	next     uintptr
	fns      map[uintptr]func(monitor uintptr, r Rect) bool
}{fns: map[uintptr]func(uintptr, Rect) bool{}}

// EnumDisplayMonitors calls fn for each monitor of the virtual screen, with its handle and its
// rectangle, until fn returns false.
//
// Flutter: no counterpart
func EnumDisplayMonitors(fn func(monitor uintptr, r Rect) bool) error {
	enumMonitors.once.Do(func() {
		enumMonitors.callback = NewCallback(func(monitor, _, rect, token uintptr) uintptr {
			enumMonitors.mu.Lock()
			f := enumMonitors.fns[token]
			enumMonitors.mu.Unlock()
			if f == nil || !f(monitor, *View[Rect](rect)) {
				return 0
			}
			return 1
		})
	})
	enumMonitors.mu.Lock()
	enumMonitors.next++
	token := enumMonitors.next
	enumMonitors.fns[token] = fn
	enumMonitors.mu.Unlock()
	defer func() {
		enumMonitors.mu.Lock()
		delete(enumMonitors.fns, token)
		enumMonitors.mu.Unlock()
	}()
	r, errno, err := user32.call("EnumDisplayMonitors", 0, 0, enumMonitors.callback, token)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("EnumDisplayMonitors", uint32(errno))
	}
	return nil
}

// GetDpiForWindow is the DPI of a window, 96 for a scale of 1. It is 0 when this Windows has no
// per-monitor DPI (before Windows 10 version 1607).
//
// Flutter: shell/platform/windows/dpi_utils.cc GetDpiForHWND
func GetDpiForWindow(hwnd uintptr) uint32 {
	p := user32.optional("GetDpiForWindow")
	if p == nil {
		return 0
	}
	r, _ := p.Call(hwnd)
	return uint32(r)
}

// AdjustWindowRectExForDpi widens a client rectangle to the outer rectangle of a window of the
// style at the DPI. Before Windows 10 version 1607 it uses AdjustWindowRectEx, which works at the
// system's DPI.
//
// Flutter: no counterpart
func AdjustWindowRectExForDpi(rc *Rect, style, exStyle uint32, dpi uint32) error {
	if p := user32.optional("AdjustWindowRectExForDpi"); p != nil {
		r, errno := p.Call(uintptr(unsafe.Pointer(rc)), uintptr(style), 0, uintptr(exStyle), uintptr(dpi))
		if r == 0 {
			return Win32Error("AdjustWindowRectExForDpi", uint32(errno))
		}
		return nil
	}
	r, errno, err := user32.call("AdjustWindowRectEx", uintptr(unsafe.Pointer(rc)), uintptr(style), 0, uintptr(exStyle))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("AdjustWindowRectEx", uint32(errno))
	}
	return nil
}

// GetSystemMetricsForDpi reads a system metric (SM*) at a DPI; before Windows 10 version 1607 it
// is GetSystemMetrics, scaled to the DPI.
//
// Flutter: no counterpart
func GetSystemMetricsForDpi(index int, dpi uint32) int32 {
	if p := user32.optional("GetSystemMetricsForDpi"); p != nil {
		r, _ := p.Call(uintptr(index), uintptr(dpi))
		return int32(r)
	}
	r, _, err := user32.call("GetSystemMetrics", uintptr(index))
	if err != nil {
		return 0
	}
	return int32(r) * int32(dpi) / UserDefaultScreenDPI
}

// GetSystemMetrics reads a system metric (SM*).
//
// Flutter: no counterpart
func GetSystemMetrics(index int) int32 {
	r, _, err := user32.call("GetSystemMetrics", uintptr(index))
	if err != nil {
		return 0
	}
	return int32(r)
}

// SetProcessDpiAwarenessContext makes the process per-monitor DPI aware (the context is
// DPIAwarenessContextPerMonitorAwareV2). It fails with WindowsDPIAwarenessFailed, with the metadata
// "code", when the process already has an awareness (its manifest set it, which is the better way:
// code ErrorAccessDenied) and with SymbolMissing before Windows 10 version 1703.
//
// Flutter: no counterpart
func SetProcessDpiAwarenessContext(context uintptr) error {
	r, errno, err := user32.call("SetProcessDpiAwarenessContext", context)
	if err != nil {
		return err
	}
	if r == 0 {
		return errors.New(TagWindowsDPIAwarenessFailed, "the process could not be made per-monitor DPI aware", "code", int(errno))
	}
	return nil
}

// LoadCursor loads a stock cursor (IDC*).
//
// Flutter: no counterpart
func LoadCursor(instance, id uintptr) uintptr {
	r, _, err := user32.call("LoadCursorW", instance, id)
	if err != nil {
		return 0
	}
	return r
}

// LoadIcon loads the icon resource with the number id from a module (the .exe: 1).
//
// Flutter: no counterpart
func LoadIcon(instance, id uintptr) uintptr {
	r, _, err := user32.call("LoadIconW", instance, id)
	if err != nil {
		return 0
	}
	return r
}

// CreateIconIndirect makes an icon from the two bitmaps of IconInfo; DestroyIcon frees it.
//
// Flutter: no counterpart
func CreateIconIndirect(ii *IconInfo) (uintptr, error) {
	r, errno, err := user32.call("CreateIconIndirect", uintptr(unsafe.Pointer(ii)))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("CreateIconIndirect", uint32(errno))
	}
	return r, nil
}

// DestroyIcon frees an icon that CreateIconIndirect made.
//
// Flutter: no counterpart
func DestroyIcon(icon uintptr) { _, _, _ = user32.call("DestroyIcon", icon) }

// SetTimer starts a timer that sends WM_TIMER to the window every elapsed milliseconds, with the
// id in wParam; KillTimer stops it.
//
// Flutter: no counterpart
func SetTimer(hwnd, id uintptr, elapsed time.Duration) error {
	r, errno, err := user32.call("SetTimer", hwnd, id, uintptr(elapsed.Milliseconds()), 0)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("SetTimer", uint32(errno))
	}
	return nil
}

// KillTimer stops a timer of SetTimer.
//
// Flutter: no counterpart
func KillTimer(hwnd, id uintptr) { _, _, _ = user32.call("KillTimer", hwnd, id) }

// EnumDisplaySettingsEx reads the settings of a display device by its name (EnumCurrentSettings
// for the mode in use) and reports whether there are such settings.
//
// Flutter: no counterpart
func EnumDisplaySettingsEx(device string, mode uint32, dm *DevMode) bool {
	p, err := UTF16PtrFromString(device)
	if err != nil {
		return false
	}
	dm.Size = uint16(unsafe.Sizeof(*dm))
	r, _, err := user32.call("EnumDisplaySettingsExW", uintptr(unsafe.Pointer(p)), uintptr(mode), uintptr(unsafe.Pointer(dm)), 0)
	return err == nil && r != 0
}

// EnumDisplayDevices describes a display adapter (device empty, index counting them) or, with
// the adapter's name, one of its monitors, and reports whether there is one.
//
// Flutter: no counterpart
func EnumDisplayDevices(device string, index uint32) (DisplayDevice, bool) {
	dd := DisplayDevice{Size: uint32(unsafe.Sizeof(DisplayDevice{}))}
	var name uintptr
	if device != "" {
		p, err := UTF16PtrFromString(device)
		if err != nil {
			return DisplayDevice{}, false
		}
		name = uintptr(unsafe.Pointer(p))
	}
	r, _, err := user32.call("EnumDisplayDevicesW", name, uintptr(index), uintptr(unsafe.Pointer(&dd)), 0)
	if err != nil || r == 0 {
		return DisplayDevice{}, false
	}
	return dd, true
}

// RegisterWindowMessage gives the number of a message that is known by its name, such as
// "TaskbarCreated".
//
// Flutter: no counterpart
func RegisterWindowMessage(name string) (uint32, error) {
	p, err := UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	r, errno, err := user32.call("RegisterWindowMessageW", uintptr(unsafe.Pointer(p)))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("RegisterWindowMessageW", uint32(errno))
	}
	return uint32(r), nil
}

// GetSystemMenu is the window menu of a window (the one the icon in the title bar opens).
//
// Flutter: no counterpart
func GetSystemMenu(hwnd uintptr) uintptr {
	r, _, err := user32.call("GetSystemMenu", hwnd, 0)
	if err != nil {
		return 0
	}
	return r
}

// TrackPopupMenuEx shows a menu at a point of the screen and, with TPMReturnCmd, returns the
// command the user chose (0 for none). It runs a modal loop.
//
// Flutter: no counterpart
func TrackPopupMenuEx(menu uintptr, flags uint32, x, y int32, hwnd uintptr) uint32 {
	r, _, err := user32.call("TrackPopupMenuEx", menu, uintptr(flags), uintptr(x), uintptr(y), hwnd, 0)
	if err != nil {
		return 0
	}
	return uint32(r)
}

// GetCursorPos is the position of the mouse cursor on the screen.
//
// Flutter: no counterpart
func GetCursorPos() (Point, bool) {
	var pt Point
	r, _, err := user32.call("GetCursorPos", uintptr(unsafe.Pointer(&pt)))
	return pt, err == nil && r != 0
}

// ClientToScreen turns a point of the client area of a window into a point of the screen.
//
// Flutter: no counterpart
func ClientToScreen(hwnd uintptr, pt Point) (Point, bool) {
	r, _, err := user32.call("ClientToScreen", hwnd, uintptr(unsafe.Pointer(&pt)))
	return pt, err == nil && r != 0
}

// GetForegroundWindow is the window the user works in, of any program.
//
// Flutter: no counterpart
func GetForegroundWindow() uintptr {
	r, _, err := user32.call("GetForegroundWindow")
	if err != nil {
		return 0
	}
	return r
}

// SendIconMessage sets the icon of a window (WM_SETICON with IconSmall or IconBig); icon 0 gives the
// window the icon of its class again.
//
// Flutter: no counterpart
func SendIconMessage(hwnd uintptr, which uintptr, icon uintptr) {
	SendMessage(hwnd, WMSetIcon, which, icon)
}

// ErrorAccessDenied is ERROR_ACCESS_DENIED, which SetProcessDpiAwarenessContext reports when the
// awareness of the process was set already.
//
// Flutter: no counterpart
const ErrorAccessDenied = 5

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagWin32WindowClassFailed is the tag of the error for a window class that cannot be registered;
	// the metadata is "code" (the Win32 error).
	TagWin32WindowClassFailed = "Win32WindowClassFailed"
	// TagWin32WindowCreateFailed is the tag of the error for a window that cannot be created; the
	// metadata are "class" and "code" (the Win32 error).
	TagWin32WindowCreateFailed = "Win32WindowCreateFailed"
	// TagWindowsDPIAwarenessFailed is the tag of the error for SetProcessDpiAwarenessContext
	// failing; the metadata is "code" (the Win32 error).
	TagWindowsDPIAwarenessFailed = "WindowsDPIAwarenessFailed"
)
