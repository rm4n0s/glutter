//go:build windows

package win32

import (
	"sync"
	"unsafe"
)

// The bindings of the desktop services of step 28.11 (SPEC §25.9): the notify icon of the tray, popup
// menus, ShellExecuteExW, COM objects made by CoCreateInstance, and the search for the window of a
// dialog. A function that fails returns a tagged error (Win32CallFailed with the name of the function
// and the Win32 error code, COMCallFailed for a failed HRESULT).

// ShellNotifyIcon calls Shell_NotifyIconW with a command of NIM*. It fails with Win32CallFailed when
// the shell does not take the command: there is no tray (no Explorer), or the data is wrong.
//
// Flutter: no counterpart
func ShellNotifyIcon(command uint32, data *NotifyIconData) error {
	r, errno, err := shell32.call("Shell_NotifyIconW", uintptr(command), uintptr(unsafe.Pointer(data)))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("Shell_NotifyIconW", uint32(errno))
	}
	return nil
}

// CreatePopupMenu makes an empty popup menu; DestroyMenu frees it, with the submenus in it.
//
// Flutter: no counterpart
func CreatePopupMenu() (uintptr, error) {
	r, errno, err := user32.call("CreatePopupMenu")
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("CreatePopupMenu", uint32(errno))
	}
	return r, nil
}

// DestroyMenu frees a menu and the submenus in it.
//
// Flutter: no counterpart
func DestroyMenu(menu uintptr) { _, _, _ = user32.call("DestroyMenu", menu) }

// InsertMenuItem inserts an item at a position of the menu (InsertMenuItemW with fByPosition).
//
// Flutter: no counterpart
func InsertMenuItem(menu uintptr, position uint32, info *MenuItemInfo) error {
	info.Size = uint32(unsafe.Sizeof(*info))
	r, errno, err := user32.call("InsertMenuItemW", menu, uintptr(position), 1, uintptr(unsafe.Pointer(info)))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("InsertMenuItemW", uint32(errno))
	}
	return nil
}

// GetMenuItemCount is the number of items of a menu, -1 when the handle is not a menu.
//
// Flutter: no counterpart
func GetMenuItemCount(menu uintptr) int {
	r, _, err := user32.call("GetMenuItemCount", menu)
	if err != nil {
		return -1
	}
	return int(int32(r))
}

// GetMenuItemInfo reads the item at a position of the menu (GetMenuItemInfoW with fByPosition): the
// members named by info.Mask. When the mask has MIIMString, info.TypeData points to a buffer of
// info.CCH characters that receives the text.
//
// Flutter: no counterpart
func GetMenuItemInfo(menu uintptr, position uint32, info *MenuItemInfo) error {
	info.Size = uint32(unsafe.Sizeof(*info))
	r, errno, err := user32.call("GetMenuItemInfoW", menu, uintptr(position), 1, uintptr(unsafe.Pointer(info)))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("GetMenuItemInfoW", uint32(errno))
	}
	return nil
}

// FindWindow finds a top-level window of the thread's desktop by its class and title; either may be
// empty for any. It returns 0 when there is none.
//
// Flutter: no counterpart
func FindWindow(class, title string) uintptr {
	var pc, pt *uint16
	var err error
	if class != "" {
		if pc, err = UTF16PtrFromString(class); err != nil {
			return 0
		}
	}
	if title != "" {
		if pt, err = UTF16PtrFromString(title); err != nil {
			return 0
		}
	}
	r, _, err := user32.call("FindWindowW", uintptr(unsafe.Pointer(pc)), uintptr(unsafe.Pointer(pt)))
	if err != nil {
		return 0
	}
	return r
}

// enumThreadWindows is the state of EnumThreadWindows: one callback, made the first time, finds the
// function of a call by the token it was given (SPEC §6.3 rule 11).
var enumThreadWindows = struct {
	once     sync.Once
	callback uintptr
	mu       sync.Mutex
	next     uintptr
	fns      map[uintptr]func(hwnd uintptr) bool
}{fns: map[uintptr]func(uintptr) bool{}}

// EnumThreadWindows calls fn for each top-level window of the thread, with its handle, until fn
// returns false.
//
// Flutter: no counterpart
func EnumThreadWindows(thread uint32, fn func(hwnd uintptr) bool) {
	enumThreadWindows.once.Do(func() {
		enumThreadWindows.callback = NewCallback(func(hwnd, token uintptr) uintptr {
			enumThreadWindows.mu.Lock()
			f := enumThreadWindows.fns[token]
			enumThreadWindows.mu.Unlock()
			if f == nil || !f(hwnd) {
				return 0
			}
			return 1
		})
	})
	enumThreadWindows.mu.Lock()
	enumThreadWindows.next++
	token := enumThreadWindows.next
	enumThreadWindows.fns[token] = fn
	enumThreadWindows.mu.Unlock()
	defer func() {
		enumThreadWindows.mu.Lock()
		delete(enumThreadWindows.fns, token)
		enumThreadWindows.mu.Unlock()
	}()
	_, _, _ = user32.call("EnumThreadWindows", uintptr(thread), enumThreadWindows.callback, token) // the result says nothing: fn stops the walk
}

// GetCurrentThreadID is the identifier of the calling thread.
//
// Flutter: no counterpart
func GetCurrentThreadID() uint32 {
	r, _, err := kern32.call("GetCurrentThreadId")
	if err != nil {
		return 0
	}
	return uint32(r)
}

// ShellExecuteEx runs ShellExecuteExW. It returns the Win32 error (GetLastError) when the call
// fails, and the value of hInstApp that Windows set, which is an SE_ERR_* code below 32.
//
// Flutter: no counterpart
func ShellExecuteEx(info *ShellExecuteInfo) (ok bool, errno uint32, instApp uintptr, err error) {
	info.Size = uint32(unsafe.Sizeof(*info))
	r, e, err := shell32.call("ShellExecuteExW", uintptr(unsafe.Pointer(info)))
	if err != nil {
		return false, 0, 0, err
	}
	return r != 0, uint32(e), info.InstApp, nil
}

// CoInitializeEx makes the calling thread a COM apartment (COINIT*); CoUninitialize undoes it. S_FALSE
// (already initialized) is not an error. It fails with COMCallFailed.
//
// Flutter: no counterpart
func CoInitializeEx(flags uint32) error {
	r, _, err := ole32.call("CoInitializeEx", 0, uintptr(flags))
	if err != nil {
		return err
	}
	return HResultError("CoInitializeEx", FromResult(r))
}

// CoUninitialize undoes a successful CoInitializeEx.
//
// Flutter: no counterpart
func CoUninitialize() { _, _, _ = ole32.call("CoUninitialize") }

// CoCreateInstance makes an in-process COM object of the class and asks for the interface. The result
// is a new reference to release. It fails with COMCallFailed, with the metadata "hresult" (such as
// REGDB_E_CLASSNOTREG).
//
// Flutter: no counterpart
func CoCreateInstance(clsid, iid *GUID) (Object, error) {
	var out uintptr
	r, _, err := ole32.call("CoCreateInstance", uintptr(unsafe.Pointer(clsid)), 0, CLSCTXInprocServer,
		uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err != nil {
		return Object{}, err
	}
	if err := HResultError("CoCreateInstance", FromResult(r)); err != nil {
		return Object{}, err
	}
	return Object{ptr: out}, nil
}

// CoTaskMemFree frees memory that COM allocated, such as the text IShellItem::GetDisplayName returns.
//
// Flutter: no counterpart
func CoTaskMemFree(p uintptr) { _, _, _ = ole32.call("CoTaskMemFree", p) }

// SHCreateItemFromParsingName makes the shell item of a path and asks for the interface. The result is
// a new reference to release. It fails with COMCallFailed (such as for a folder that does not exist).
//
// Flutter: no counterpart
func SHCreateItemFromParsingName(path string, iid *GUID) (Object, error) {
	p, err := UTF16PtrFromString(path)
	if err != nil {
		return Object{}, err
	}
	var out uintptr
	r, _, err := shell32.call("SHCreateItemFromParsingName", uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err != nil {
		return Object{}, err
	}
	if err := HResultError("SHCreateItemFromParsingName", FromResult(r)); err != nil {
		return Object{}, err
	}
	return Object{ptr: out}, nil
}

// The IDs of the interfaces and the classes of the common item dialogs and the shell items.
var (
	// IIDShellItem is the interface ID of IShellItem.
	//
	// Flutter: no counterpart
	IIDShellItem = MustParseGUID("{43826d1e-e718-42ee-bc55-a1e261c37bfe}")
	// CLSIDFileOpenDialog is the class ID of the open dialog.
	//
	// Flutter: no counterpart
	CLSIDFileOpenDialog = MustParseGUID("{dc1c5a9c-e88a-4dde-a5a1-60f82a20aef7}")
	// CLSIDFileSaveDialog is the class ID of the save dialog.
	//
	// Flutter: no counterpart
	CLSIDFileSaveDialog = MustParseGUID("{c0b4e2f3-ba21-4773-8dba-335ec946eb8b}")
	// IIDFileOpenDialog is the interface ID of IFileOpenDialog.
	//
	// Flutter: no counterpart
	IIDFileOpenDialog = MustParseGUID("{d57c7288-d4ad-4768-be02-9d969532d960}")
	// IIDFileSaveDialog is the interface ID of IFileSaveDialog.
	//
	// Flutter: no counterpart
	IIDFileSaveDialog = MustParseGUID("{84bccd23-5fde-4cdb-aea4-af64b83d78ab}")
)
