//go:build windows

package win32

import (
	"unicode/utf16"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The bindings of UI Automation's core (uiautomationcore.dll) and of OLE Automation's strings and arrays
// (oleaut32.dll) that the providers need (SPEC §25.11).

var (
	uiacore  = &dllTable{name: "uiautomationcore.dll"}
	oleaut32 = &dllTable{name: "oleaut32.dll"}
)

// UiaReturnRawElementProvider answers a WM_GETOBJECT for UiaRootObjectId with the provider: the result
// is what the window procedure returns. It fails with WindowsUIAFailed when UI Automation gives none.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnGetObject
func UiaReturnRawElementProvider(hwnd, wparam, lparam uintptr, provider Object) (uintptr, error) {
	r, _, err := uiacore.call("UiaReturnRawElementProvider", hwnd, wparam, lparam, provider.Ptr())
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagWindowsUIAFailed, "UiaReturnRawElementProvider gave no result", "function", "UiaReturnRawElementProvider", "hwnd", hwnd)
	}
	return r, nil
}

// UiaHostProviderFromHwnd is the provider that Windows has for the window itself (its title, its
// frame): the host of the fragment root.
//
// Flutter: third_party/accessibility/ax/platform/ax_fragment_root_win.cc AXFragmentRootWin::get_HostRawElementProvider
func UiaHostProviderFromHwnd(hwnd uintptr) (Object, error) {
	var p uintptr
	r, _, err := uiacore.call("UiaHostProviderFromHwnd", hwnd, uintptr(unsafe.Pointer(&p)))
	if err != nil {
		return Object{}, err
	}
	if err := HResultError("UiaHostProviderFromHwnd", FromResult(r)); err != nil {
		return Object{}, err
	}
	return ObjectFromPtr(p), nil
}

// UiaClientsAreListening says whether a client has subscribed to events.
//
// Flutter: no counterpart
func UiaClientsAreListening() bool {
	r, _, err := uiacore.call("UiaClientsAreListening")
	return err == nil && uint32(r) != 0
}

// UiaRaiseAutomationEvent raises an event of the element.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::NotifyAccessibilityEvent
func UiaRaiseAutomationEvent(provider Object, event int32) error {
	r, _, err := uiacore.call("UiaRaiseAutomationEvent", provider.Ptr(), uintptr(event))
	if err != nil {
		return err
	}
	return HResultError("UiaRaiseAutomationEvent", FromResult(r))
}

// UiaRaiseAutomationPropertyChangedEvent raises the event that a property of the element changed.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::NotifyAccessibilityEvent
func UiaRaiseAutomationPropertyChangedEvent(provider Object, property int32, old, now *Variant) error {
	r, _, err := uiacore.call("UiaRaiseAutomationPropertyChangedEvent", provider.Ptr(), uintptr(property),
		uintptr(unsafe.Pointer(old)), uintptr(unsafe.Pointer(now)))
	if err != nil {
		return err
	}
	return HResultError("UiaRaiseAutomationPropertyChangedEvent", FromResult(r))
}

// UiaRaiseStructureChangedEvent raises the event that the structure under the element changed. runtimeID
// is the ID of the child that was removed, nil for the other kinds.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::NotifyAccessibilityEvent
func UiaRaiseStructureChangedEvent(provider Object, kind int32, runtimeID []int32) error {
	// The pointer stays unsafe.Pointer until the argument list, where call keeps it alive: UI Automation
	// calls the providers of this process while the call runs, and that Go code can move the stack (SPEC §25.2).
	var p unsafe.Pointer
	if len(runtimeID) > 0 {
		p = unsafe.Pointer(&runtimeID[0])
	}
	r, _, err := uiacore.call("UiaRaiseStructureChangedEvent", provider.Ptr(), uintptr(kind), uintptr(p), uintptr(len(runtimeID)))
	if err != nil {
		return err
	}
	return HResultError("UiaRaiseStructureChangedEvent", FromResult(r))
}

// UiaDisconnectProvider tells UI Automation that the provider is gone, so that it lets go of it.
//
// Flutter: no counterpart
func UiaDisconnectProvider(provider Object) error {
	r, _, err := uiacore.call("UiaDisconnectProvider", provider.Ptr())
	if err != nil {
		return err
	}
	return HResultError("UiaDisconnectProvider", FromResult(r))
}

// SysAllocString makes a BSTR with a copy of the text; SysFreeString frees it. It fails with
// Win32CallFailed when Windows has no memory.
//
// Flutter: no counterpart
func SysAllocString(s string) (uintptr, error) {
	u, err := UTF16FromString(s)
	if err != nil {
		u, _ = UTF16FromString(replaceNUL(s))
	}
	n := len(u) - 1 // without the terminator
	var p *uint16
	if len(u) > 0 {
		p = &u[0]
	}
	r, errno, err := oleaut32.call("SysAllocStringLen", uintptr(unsafe.Pointer(p)), uintptr(n))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("SysAllocStringLen", uint32(errno))
	}
	return r, nil
}

func replaceNUL(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c == 0 {
			b[i] = ' '
		}
	}
	return string(b)
}

// SysFreeString frees a BSTR.
//
// Flutter: no counterpart
func SysFreeString(bstr uintptr) {
	if bstr != 0 {
		_, _, _ = oleaut32.call("SysFreeString", bstr)
	}
}

// BSTRToString copies a BSTR (a pointer to UTF-16 text with its length before it) into a Go string.
//
// Flutter: no counterpart
func BSTRToString(bstr uintptr) string {
	if bstr == 0 {
		return ""
	}
	r, _, err := oleaut32.call("SysStringLen", bstr)
	if err != nil || r == 0 {
		return ""
	}
	return string(utf16.Decode(unsafe.Slice((*uint16)(at(bstr)), int(r))))
}

// SafeArrayCreateVector makes a one-dimensional SAFEARRAY of n elements of type vt, from index lower.
// SafeArrayDestroy frees it.
//
// Flutter: no counterpart
func SafeArrayCreateVector(vt uint16, lower int32, n uint32) (uintptr, error) {
	r, errno, err := oleaut32.call("SafeArrayCreateVector", uintptr(vt), uintptr(lower), uintptr(n))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, Win32Error("SafeArrayCreateVector", uint32(errno))
	}
	return r, nil
}

// SafeArrayPutElement copies the element (a pointer to the value) into the array at the index.
//
// Flutter: no counterpart
func SafeArrayPutElement(sa uintptr, index int32, elem unsafe.Pointer) error {
	idx := index
	r, _, err := oleaut32.call("SafeArrayPutElement", sa, uintptr(unsafe.Pointer(&idx)), uintptr(elem))
	if err != nil {
		return err
	}
	return HResultError("SafeArrayPutElement", FromResult(r))
}

// SafeArrayGetElement copies the element at the index into the memory elem points to.
//
// Flutter: no counterpart
func SafeArrayGetElement(sa uintptr, index int32, elem unsafe.Pointer) error {
	idx := index
	r, _, err := oleaut32.call("SafeArrayGetElement", sa, uintptr(unsafe.Pointer(&idx)), uintptr(elem))
	if err != nil {
		return err
	}
	return HResultError("SafeArrayGetElement", FromResult(r))
}

// SafeArrayGetUBound is the upper bound of the first dimension (the last index).
//
// Flutter: no counterpart
func SafeArrayGetUBound(sa uintptr) (int32, error) {
	var ub int32
	r, _, err := oleaut32.call("SafeArrayGetUBound", sa, 1, uintptr(unsafe.Pointer(&ub)))
	if err != nil {
		return 0, err
	}
	if err := HResultError("SafeArrayGetUBound", FromResult(r)); err != nil {
		return 0, err
	}
	return ub, nil
}

// SafeArrayGetLBound is the lower bound of the first dimension (the first index).
//
// Flutter: no counterpart
func SafeArrayGetLBound(sa uintptr) (int32, error) {
	var lb int32
	r, _, err := oleaut32.call("SafeArrayGetLBound", sa, 1, uintptr(unsafe.Pointer(&lb)))
	if err != nil {
		return 0, err
	}
	if err := HResultError("SafeArrayGetLBound", FromResult(r)); err != nil {
		return 0, err
	}
	return lb, nil
}

// SafeArrayDestroy frees an array and its elements.
//
// Flutter: no counterpart
func SafeArrayDestroy(sa uintptr) {
	if sa != 0 {
		_, _, _ = oleaut32.call("SafeArrayDestroy", sa)
	}
}

// VariantClear frees what the VARIANT holds (a BSTR, an interface) and makes it empty.
//
// Flutter: no counterpart
func VariantClear(v *Variant) {
	_, _, _ = oleaut32.call("VariantClear", uintptr(unsafe.Pointer(v)))
}
