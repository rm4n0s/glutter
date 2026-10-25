//go:build windows

package win32

import (
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The bindings of the clipboard (user32) and of the global memory that it takes (kernel32), for the
// clipboard of the platform (SPEC §25.7). The functions return the Win32 error of the call, so that
// the platform can tell a busy clipboard (ERROR_ACCESS_DENIED from OpenClipboard) from the rest.

// ErrorAccessDenied is ERROR_ACCESS_DENIED, which OpenClipboard reports when another window has the
// clipboard open.
const errorAccessDenied = 5

// OpenClipboard opens the clipboard for the thread, with the window as the owner of what is put on
// it (0 for none: that is enough to read). It fails with Win32CallFailed; the code is
// ERROR_ACCESS_DENIED while another window has the clipboard open. CloseClipboard must follow on
// the same thread.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedClipboard::Open
func OpenClipboard(hwnd uintptr) error {
	r, errno, err := user32.call("OpenClipboard", hwnd)
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("OpenClipboard", uint32(errno))
	}
	return nil
}

// ClipboardBusyError says whether the error is the failure of OpenClipboard because another window
// has the clipboard open.
//
// Flutter: no counterpart
func ClipboardBusyError(err error) bool {
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagWin32CallFailed || appErr.Metadata["function"] != "OpenClipboard" {
		return false
	}
	code, _ := appErr.Metadata["code"].(int)
	return code == errorAccessDenied
}

// CloseClipboard closes the clipboard that OpenClipboard opened.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedClipboard::~ScopedClipboard
func CloseClipboard() { _, _, _ = user32.call("CloseClipboard") }

// EmptyClipboard empties the clipboard and makes the window of OpenClipboard its owner.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedClipboard::SetString
func EmptyClipboard() error {
	r, errno, err := user32.call("EmptyClipboard")
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("EmptyClipboard", uint32(errno))
	}
	return nil
}

// SetClipboardData puts the global memory h on the clipboard in the format. When the call works,
// the clipboard owns the memory; when it fails, the caller still does. A zero h announces data that
// the owner renders when a program asks for it (WM_RENDERFORMAT); the call returns zero then even
// when it works.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedClipboard::SetString
func SetClipboardData(format uint32, h uintptr) error {
	r, errno, err := user32.call("SetClipboardData", uintptr(format), h)
	if err != nil {
		return err
	}
	if r == 0 && h != 0 {
		return Win32Error("SetClipboardData", uint32(errno))
	}
	return nil
}

// GetClipboardData is the global memory of the clipboard in the format, 0 when it has none. The
// clipboard owns it: lock it with GlobalLock and copy what is needed before CloseClipboard.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedClipboard::GetString
func GetClipboardData(format uint32) uintptr {
	r, _, err := user32.call("GetClipboardData", uintptr(format))
	if err != nil {
		return 0
	}
	return r
}

// IsClipboardFormatAvailable says whether the clipboard has the format.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedClipboard::HasString
func IsClipboardFormatAvailable(format uint32) bool {
	return boolCall(user32, "IsClipboardFormatAvailable", uintptr(format))
}

// RegisterClipboardFormat gets the number of a clipboard format by its name ("PNG").
//
// Flutter: no counterpart
func RegisterClipboardFormat(name string) (uint32, error) {
	p, err := UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	r, errno, err := user32.call("RegisterClipboardFormatW", uintptr(unsafe.Pointer(p)))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("RegisterClipboardFormatW", uint32(errno))
	}
	return uint32(r), nil
}

// GetClipboardOwner is the window that owns the clipboard, 0 for none.
//
// Flutter: no counterpart
func GetClipboardOwner() uintptr {
	r, _, err := user32.call("GetClipboardOwner")
	if err != nil {
		return 0
	}
	return r
}

// GlobalLock locks global memory and returns its address, which is valid until GlobalUnlock.
//
// Flutter: no counterpart
func GlobalLock(h uintptr) (uintptr, error) {
	r, errno, err := kern32.call("GlobalLock", h)
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("GlobalLock", uint32(errno))
	}
	return r, nil
}

// GlobalUnlock unlocks global memory that GlobalLock locked.
//
// Flutter: no counterpart
func GlobalUnlock(h uintptr) { _, _, _ = kern32.call("GlobalUnlock", h) }

// GlobalSize is the size in bytes of global memory, 0 when the handle is not valid.
//
// Flutter: no counterpart
func GlobalSize(h uintptr) uintptr {
	r, _, err := kern32.call("GlobalSize", h)
	if err != nil {
		return 0
	}
	return r
}

// NewGlobal allocates movable global memory that holds a copy of data (at least one byte), ready
// for SetClipboardData. Free it with GlobalFree unless the clipboard took it.
//
// Flutter: shell/platform/windows/platform_handler.cc ScopedGlobalMemory
func NewGlobal(data []byte) (uintptr, error) {
	h, err := GlobalAlloc(GMemMoveable, uintptr(max(len(data), 1)))
	if err != nil {
		return 0, err
	}
	addr, err := GlobalLock(h)
	if err != nil {
		_ = GlobalFree(h) // the lock error is the one to report
		return 0, err
	}
	copy(Bytes(addr, max(len(data), 1)), data)
	GlobalUnlock(h)
	return h, nil
}

// GlobalBytes copies the content of global memory.
//
// Flutter: no counterpart
func GlobalBytes(h uintptr) ([]byte, error) {
	n := int(GlobalSize(h))
	if n == 0 {
		return nil, nil
	}
	addr, err := GlobalLock(h)
	if err != nil {
		return nil, err
	}
	defer GlobalUnlock(h)
	return append([]byte(nil), Bytes(addr, n)...), nil
}
