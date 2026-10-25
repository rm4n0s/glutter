//go:build windows

package win32

import (
	"unsafe"
)

// The bindings of OLE drag and drop (SPEC §25.10): the registration of a drop target, the modal loop
// of a drag, the release of the data a data object gave, and the few user32 calls that go with them.

// firstRegisteredFormat is 0xC000, the first clipboard format number that RegisterClipboardFormat gives.
const firstRegisteredFormat = 0xC000

// RegisterDragDrop makes the object (an IDropTarget) the drop target of the window; OLE keeps a
// reference. It must run on the thread of the window, in a single-threaded apartment (OleInitialize).
// It fails with COMCallFailed; the metadata "hresult" is DRAGDROP_E_ALREADYREGISTERED when the window
// has a target already.
//
// Flutter: no counterpart
func RegisterDragDrop(hwnd uintptr, target Object) error {
	r, _, err := ole32.call("RegisterDragDrop", hwnd, target.Ptr())
	if err != nil {
		return err
	}
	return HResultError("RegisterDragDrop", FromResult(r))
}

// RevokeDragDrop takes the drop target away from the window; OLE releases its reference. It fails with
// COMCallFailed when the window has none.
//
// Flutter: no counterpart
func RevokeDragDrop(hwnd uintptr) error {
	r, _, err := ole32.call("RevokeDragDrop", hwnd)
	if err != nil {
		return err
	}
	return HResultError("RevokeDragDrop", FromResult(r))
}

// DoDragDrop runs the modal loop of a drag until the source's QueryContinueDrag ends it. effect is
// the effect that the target chose (DROPEFFECT_NONE when nothing took the drop); dropped says that the
// loop ended with a drop, and not with a cancel. It fails with COMCallFailed.
//
// Flutter: no counterpart
func DoDragDrop(data, source Object, okEffects uint32) (effect uint32, dropped bool, err error) {
	r, _, err := ole32.call("DoDragDrop", data.Ptr(), source.Ptr(), uintptr(okEffects), uintptr(unsafe.Pointer(&effect)))
	if err != nil {
		return 0, false, err
	}
	hr := FromResult(r)
	if hr.Failed() {
		return 0, false, HResultError("DoDragDrop", hr)
	}
	return effect, hr == DragDropSDrop, nil
}

// ReleaseStgMedium frees the data of a medium that a data object gave (a GetData that worked): the
// global memory, or the object named by pUnkForRelease.
//
// Flutter: no counterpart
func ReleaseStgMedium(m *StgMedium) {
	_, _, _ = ole32.call("ReleaseStgMedium", uintptr(unsafe.Pointer(m)))
}

// GetClipboardFormatName is the name of a registered clipboard format, "" for a format that is not
// registered: the standard CF_* formats have no name (Windows says so; Wine answers "#8" for CF_DIB,
// so the numbers below 0xC000, which are not registered, are not asked).
//
// Flutter: no counterpart
func GetClipboardFormatName(format uint32) string {
	if format < firstRegisteredFormat {
		return ""
	}
	var buf [256]uint16
	r, _, err := user32.call("GetClipboardFormatNameW", uintptr(format), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if err != nil || r == 0 {
		return ""
	}
	return UTF16ToString(buf[:r])
}

// IsWindowEnabled says whether the window takes input (a modal dialog disables its owner).
//
// Flutter: no counterpart
func IsWindowEnabled(hwnd uintptr) bool { return boolCall(user32, "IsWindowEnabled", hwnd) }

// GetProp is the value of the property of the window with the name, 0 when there is none. OLE keeps
// the drop target of a window in the property "OleDropTargetInterface" (as Wine does); the tests read
// it to find the target that the platform registered.
//
// Flutter: no counterpart
func GetProp(hwnd uintptr, name string) uintptr {
	p, err := UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, err := user32.call("GetPropW", hwnd, uintptr(unsafe.Pointer(p)))
	if err != nil {
		return 0
	}
	return r
}

// SetActiveWindow makes the window of the thread the active one and returns the previous one.
//
// Flutter: no counterpart
func SetActiveWindow(hwnd uintptr) uintptr {
	r, _, err := user32.call("SetActiveWindow", hwnd)
	if err != nil {
		return 0
	}
	return r
}
