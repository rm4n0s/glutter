//go:build windows

package win32

import (
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The bindings of gdi32, dwmapi, shcore, shell32, ole32, advapi32 and the rest of kernel32 that the
// windows of step 28.2 need: drawing the soft frame with GDI (SPEC §25.5), the compositor (DWM),
// the DPI of a monitor, the app's identity, the OLE apartment, the registry, the console and the
// waitable timer of the frame pacer (SPEC §25.3).

var (
	gdi32    = &dllTable{name: "gdi32.dll"}
	dwmapi   = &dllTable{name: "dwmapi.dll"}
	shcore   = &dllTable{name: "shcore.dll"}
	shell32  = &dllTable{name: "shell32.dll"}
	ole32    = &dllTable{name: "ole32.dll"}
	advapi32 = &dllTable{name: "advapi32.dll"}
	kern32   = &dllTable{name: "kernel32.dll"}
)

// StretchDIBits copies a rectangle of a device-independent bitmap to the device context
// (StretchDIBits). The source is bits, described by info; the copy is 1:1 when the two sizes are
// equal. It returns the number of scan lines copied, 0 when the call failed.
//
// Flutter: no counterpart
func StretchDIBits(dc uintptr, dstX, dstY, dstW, dstH, srcX, srcY, srcW, srcH int32, bits []byte, info *BitmapInfo, usage uint32, rop uint32) int32 {
	if len(bits) == 0 {
		return 0
	}
	r, _, err := gdi32.call("StretchDIBits", dc, uintptr(dstX), uintptr(dstY), uintptr(dstW), uintptr(dstH),
		uintptr(srcX), uintptr(srcY), uintptr(srcW), uintptr(srcH), uintptr(unsafe.Pointer(&bits[0])),
		uintptr(unsafe.Pointer(info)), uintptr(usage), uintptr(rop))
	if err != nil {
		return 0
	}
	return int32(r)
}

// CreateDIBSection makes a bitmap whose pixels are memory the caller can write, and returns the
// bitmap and the address of the pixels. DeleteObject frees the bitmap.
//
// Flutter: no counterpart
func CreateDIBSection(info *BitmapInfo) (bitmap, bits uintptr, err error) {
	var pixels uintptr
	r, errno, err := gdi32.call("CreateDIBSection", 0, uintptr(unsafe.Pointer(info)), DIBRGBColors, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if err != nil {
		return 0, 0, err
	}
	if r == 0 {
		return 0, 0, Win32Error("CreateDIBSection", uint32(errno))
	}
	return r, pixels, nil
}

// CreateBitmap makes a bitmap of the given size and bits per pixel from the bits (nil for none).
//
// Flutter: no counterpart
func CreateBitmap(width, height int32, planes, bitCount uint32, bits []byte) (uintptr, error) {
	var p uintptr
	if len(bits) > 0 {
		p = uintptr(unsafe.Pointer(&bits[0]))
	}
	r, errno, err := gdi32.call("CreateBitmap", uintptr(width), uintptr(height), uintptr(planes), uintptr(bitCount), p)
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("CreateBitmap", uint32(errno))
	}
	return r, nil
}

// DeleteObject frees a GDI object, such as a bitmap.
//
// Flutter: no counterpart
func DeleteObject(obj uintptr) { _, _, _ = gdi32.call("DeleteObject", obj) }

// DwmSetWindowAttribute sets a DWM attribute of a window to a 32-bit value (DWMWA*). It fails with
// COMCallFailed when this Windows does not know the attribute.
//
// Flutter: no counterpart
func DwmSetWindowAttribute(hwnd uintptr, attribute uint32, value int32) error {
	r, _, err := dwmapi.call("DwmSetWindowAttribute", hwnd, uintptr(attribute), uintptr(unsafe.Pointer(&value)), 4)
	if err != nil {
		return err
	}
	return HResultError("DwmSetWindowAttribute", FromResult(r))
}

// DwmGetWindowAttributeRect reads a DWM attribute that is a rectangle, such as DWMWAExtendedFrameBounds.
//
// Flutter: no counterpart
func DwmGetWindowAttributeRect(hwnd uintptr, attribute uint32) (Rect, error) {
	var rc Rect
	r, _, err := dwmapi.call("DwmGetWindowAttribute", hwnd, uintptr(attribute), uintptr(unsafe.Pointer(&rc)), unsafe.Sizeof(rc))
	if err != nil {
		return Rect{}, err
	}
	if hr := FromResult(r); hr.Failed() {
		return Rect{}, HResultError("DwmGetWindowAttribute", hr)
	}
	return rc, nil
}

// DwmFlush waits until the compositor's next vsync. It fails with COMCallFailed when composition
// is off (Windows 7) or the compositor does not answer, and under Wine it returns at once.
//
// Flutter: no counterpart
func DwmFlush() error {
	r, _, err := dwmapi.call("DwmFlush")
	if err != nil {
		return err
	}
	return HResultError("DwmFlush", FromResult(r))
}

// GetDpiForMonitor is the effective DPI of a monitor, 96 for a scale of 1. It returns 96 when this
// Windows has no per-monitor DPI.
//
// Flutter: shell/platform/windows/dpi_utils.cc GetDpiForMonitor
func GetDpiForMonitor(monitor uintptr) uint32 {
	p := shcore.optional("GetDpiForMonitor")
	if p == nil {
		return UserDefaultScreenDPI
	}
	var x, y uint32
	r, _ := p.Call(monitor, MDTEffectiveDPI, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	if FromResult(r).Failed() || x == 0 {
		return UserDefaultScreenDPI
	}
	return x
}

// SetCurrentProcessExplicitAppUserModelID gives the process the identity by which the task bar
// groups its windows and the notifications carry its name.
//
// Flutter: no counterpart
func SetCurrentProcessExplicitAppUserModelID(id string) error {
	p, err := UTF16PtrFromString(id)
	if err != nil {
		return err
	}
	r, _, err := shell32.call("SetCurrentProcessExplicitAppUserModelID", uintptr(unsafe.Pointer(p)))
	if err != nil {
		return err
	}
	return HResultError("SetCurrentProcessExplicitAppUserModelID", FromResult(r))
}

// OleInitialize makes the calling thread a single-threaded apartment of OLE (COINIT_APARTMENTTHREADED),
// which drag and drop, the shell and UI Automation need. S_FALSE (already initialized) is not an
// error. OleUninitialize undoes it.
//
// Flutter: no counterpart
func OleInitialize() error {
	r, _, err := ole32.call("OleInitialize", 0)
	if err != nil {
		return err
	}
	if hr := FromResult(r); hr.Failed() {
		return errors.New(TagOleInitFailed, "OleInitialize failed", "hresult", hr.String())
	}
	return nil
}

// OleUninitialize undoes OleInitialize.
//
// Flutter: no counterpart
func OleUninitialize() { _, _, _ = ole32.call("OleUninitialize") }

// RegGetDWORD reads a DWORD value of the registry (RegGetValueW with RRF_RT_REG_DWORD). found is
// false when the key or the value is not there.
//
// Flutter: no counterpart
func RegGetDWORD(root uintptr, subKey, value string) (v uint32, found bool, err error) {
	ps, err := UTF16PtrFromString(subKey)
	if err != nil {
		return 0, false, err
	}
	pv, err := UTF16PtrFromString(value)
	if err != nil {
		return 0, false, err
	}
	size := uint32(4)
	r, _, err := advapi32.call("RegGetValueW", root, uintptr(unsafe.Pointer(ps)), uintptr(unsafe.Pointer(pv)), RRFRTRegDword, 0,
		uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&size)))
	if err != nil {
		return 0, false, err
	}
	if r != 0 { // ERROR_SUCCESS is 0; ERROR_FILE_NOT_FOUND and the others mean there is no value
		return 0, false, nil
	}
	return v, true, nil
}

// AttachConsole attaches the process to the console of its parent (AttachParentProcess), and
// reports whether there was one.
//
// Flutter: no counterpart
func AttachConsole(pid uint32) bool { return boolCall(kern32, "AttachConsole", uintptr(pid)) }

// GetStdHandle is a standard handle of the process (StdOutputHandle, StdErrorHandle): 0 when the
// process has none, and the invalid handle (all bits set) when it failed.
//
// Flutter: no counterpart
func GetStdHandle(which int) uintptr {
	r, _, err := kern32.call("GetStdHandle", uintptr(which))
	if err != nil {
		return 0
	}
	return r
}

// OutputDebugString sends a line to the debugger (and to DebugView).
//
// Flutter: no counterpart
func OutputDebugString(s string) {
	p, err := UTF16PtrFromString(s)
	if err != nil {
		return
	}
	_, _, _ = kern32.call("OutputDebugStringW", uintptr(unsafe.Pointer(p)))
}

// SetThreadExecutionState tells the system what the thread needs (ESContinuous | ESDisplayRequired
// keeps the display on) and returns the previous state, 0 when the call failed.
//
// Flutter: no counterpart
func SetThreadExecutionState(state uint32) uint32 {
	r, _, err := kern32.call("SetThreadExecutionState", uintptr(state))
	if err != nil {
		return 0
	}
	return uint32(r)
}

// WaitableTimer is a timer of the kernel that a thread can wait on: the frame pacer waits on it
// when the compositor's vsync cannot be used (SPEC §25.3).
//
// Flutter: no counterpart
type WaitableTimer struct{ handle uintptr }

// NewWaitableTimer makes an auto-reset waitable timer, with the high-resolution flag
// (CREATE_WAITABLE_TIMER_HIGH_RESOLUTION, Windows 10 version 1803) when the system takes it and an
// ordinary one otherwise. Close it with Close.
//
// Flutter: no counterpart
func NewWaitableTimer() (*WaitableTimer, error) {
	p := kern32.optional("CreateWaitableTimerExW")
	if p == nil {
		return nil, Win32Error("CreateWaitableTimerExW", 127 /* ERROR_PROC_NOT_FOUND */)
	}
	h, errno := p.Call(0, 0, CreateWaitableTimerHighResolution, TimerAllAccess)
	if h == 0 {
		h, errno = p.Call(0, 0, 0, TimerAllAccess)
	}
	if h == 0 {
		return nil, Win32Error("CreateWaitableTimerExW", uint32(errno))
	}
	return &WaitableTimer{handle: h}, nil
}

// WaitFor sets the timer to go off once after d and waits for it. A zero or negative d returns at
// once.
//
// Flutter: no counterpart
func (t *WaitableTimer) WaitFor(d time.Duration) error {
	if d <= 0 {
		return nil
	}
	due := -int64(d / 100) // negative: relative, in 100 ns units
	r, errno, err := kern32.call("SetWaitableTimer", t.handle, uintptr(unsafe.Pointer(&due)), 0, 0, 0, 0)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("SetWaitableTimer", uint32(errno))
	}
	_, err = WaitForSingleObject(t.handle, WaitInfinite)
	return err
}

// Close frees the timer.
//
// Flutter: no counterpart
func (t *WaitableTimer) Close() error {
	h := t.handle
	t.handle = 0
	if h == 0 {
		return nil
	}
	return CloseHandle(h)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagOleInitFailed is the tag of the error for OleInitialize failing; the metadata is "hresult".
	TagOleInitFailed = "OleInitFailed"
)

// GetPixel reads the color of a pixel of a device context as 0x00BBGGRR (COLORREF); it is
// 0xFFFFFFFF (CLR_INVALID) for a point outside the clipping region. The tests read back what was
// drawn with it.
//
// Flutter: no counterpart
func GetPixel(dc uintptr, x, y int32) uint32 {
	r, _, err := gdi32.call("GetPixel", dc, uintptr(x), uintptr(y))
	if err != nil {
		return 0xFFFFFFFF
	}
	return uint32(r)
}
