package win32

import (
	"fmt"

	"github.com/rm4n0s/errors"
)

// HRESULT is the result of a COM method and of many Windows functions: a 32-bit value whose high
// bit says that the call failed.
//
// Flutter: no counterpart
type HRESULT int32

// The HRESULT values that Glutter meets. The layout test of this package checks each against
// the mingw-w64 headers.
const (
	// SOK is S_OK: the call worked.
	//
	// Flutter: no counterpart
	SOK HRESULT = 0
	// ENotImpl is E_NOTIMPL: the method is not implemented.
	//
	// Flutter: no counterpart
	ENotImpl HRESULT = -0x7FFFBFFF // 0x80004001
	// ENoInterface is E_NOINTERFACE: the object does not have the interface asked for.
	//
	// Flutter: no counterpart
	ENoInterface HRESULT = -0x7FFFBFFE // 0x80004002
	// EPointer is E_POINTER: a pointer argument was null.
	//
	// Flutter: no counterpart
	EPointer HRESULT = -0x7FFFBFFD // 0x80004003
	// EFail is E_FAIL: an unspecified failure.
	//
	// Flutter: no counterpart
	EFail HRESULT = -0x7FFFBFFB // 0x80004005
	// EOutOfMemory is E_OUTOFMEMORY.
	//
	// Flutter: no counterpart
	EOutOfMemory HRESULT = -0x7FF8FFF2 // 0x8007000E
	// EInvalidArg is E_INVALIDARG.
	//
	// Flutter: no counterpart
	EInvalidArg HRESULT = -0x7FF8FFA9 // 0x80070057
	// ErrorCancelled is HRESULT_FROM_WIN32(ERROR_CANCELLED): the user cancelled a dialog.
	//
	// Flutter: no counterpart
	ErrorCancelled HRESULT = -0x7FF8FB39 // 0x800704C7
	// DXGIErrorDeviceHung is DXGI_ERROR_DEVICE_HUNG: the GPU stopped answering.
	//
	// Flutter: no counterpart
	DXGIErrorDeviceHung HRESULT = -0x7785FFFA // 0x887A0006
	// DXGIErrorDeviceRemoved is DXGI_ERROR_DEVICE_REMOVED: the adapter was removed or reset.
	//
	// Flutter: no counterpart
	DXGIErrorDeviceRemoved HRESULT = -0x7785FFFB // 0x887A0005
	// DXGIErrorDeviceReset is DXGI_ERROR_DEVICE_RESET: the device was reset after a bad command.
	//
	// Flutter: no counterpart
	DXGIErrorDeviceReset HRESULT = -0x7785FFF9 // 0x887A0007
)

// FromResult reads the HRESULT out of the register that a call returned.
//
// Flutter: no counterpart
func FromResult(r uintptr) HRESULT { return HRESULT(int32(uint32(r))) }

// Failed is the FAILED macro: whether the high bit is set.
//
// Flutter: no counterpart
func (h HRESULT) Failed() bool { return h < 0 }

// Succeeded is the SUCCEEDED macro.
//
// Flutter: no counterpart
func (h HRESULT) Succeeded() bool { return h >= 0 }

// Code is the value as the unsigned number that the headers write, such as 0x80004005.
//
// Flutter: no counterpart
func (h HRESULT) Code() uint32 { return uint32(h) }

// String is the code in hexadecimal, "0x80004005".
//
// Flutter: no counterpart
func (h HRESULT) String() string { return fmt.Sprintf("0x%08X", uint32(h)) }

// HResultError turns a failed HRESULT into a tagged error, and returns nil for a success. method
// names the COM method or function ("IDXGIFactory2::CreateSwapChainForHwnd"). The tag is
// GPUOutOfMemory for E_OUTOFMEMORY, GPUDeviceLost for DXGI_ERROR_DEVICE_REMOVED, _RESET and _HUNG,
// Win32Cancelled for ERROR_CANCELLED (a dialog the user closed), and COMCallFailed for any other
// failure; the metadata are "method" and "hresult" (the code in hexadecimal). It is one
// table, like vkError, and is listed in errcheck.allow (SPEC §14.4, R4).
//
// Flutter: no counterpart
func HResultError(method string, hr HRESULT) error {
	if hr.Succeeded() {
		return nil
	}
	switch hr {
	case EOutOfMemory:
		return errors.New(tagGPUOutOfMemory, "Windows is out of memory", "method", method, "hresult", hr.String())
	case DXGIErrorDeviceRemoved, DXGIErrorDeviceReset, DXGIErrorDeviceHung:
		return errors.New(tagGPUDeviceLost, "the graphics device was removed or reset", "method", method, "hresult", hr.String())
	case ErrorCancelled:
		return errors.New(TagWin32Cancelled, "the user cancelled", "method", method, "hresult", hr.String())
	}
	return errors.New(TagCOMCallFailed, "a COM method failed", "method", method, "hresult", hr.String())
}

// ErrorCancelledCode is ERROR_CANCELLED, the Win32 error of a dialog that the user closed.
//
// Flutter: no counterpart
const ErrorCancelledCode = 1223

// Win32Error turns the last error of a Win32 function that failed (GetLastError, which
// syscall.SyscallN returns) into a tagged error: Win32Cancelled for ERROR_CANCELLED, and
// Win32CallFailed for any other code; the metadata are "function" and "code". It is one table
// like HResultError and is listed in errcheck.allow.
//
// Flutter: no counterpart
func Win32Error(function string, code uint32) error {
	if code == ErrorCancelledCode {
		return errors.New(TagWin32Cancelled, "the user cancelled", "function", function, "code", int(code))
	}
	return errors.New(TagWin32CallFailed, "a Win32 function failed", "function", function, "code", int(code))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagWin32CallFailed is the tag of the error for a Win32 function that failed; the metadata
	// are "function" and "code" (the GetLastError value).
	TagWin32CallFailed = "Win32CallFailed"
	// TagCOMCallFailed is the tag of the error for a COM method that returned a failed HRESULT;
	// the metadata are "method" and "hresult".
	TagCOMCallFailed = "COMCallFailed"
	// TagWin32Cancelled is the tag of the error for a call that the user cancelled
	// (ERROR_CANCELLED): a dialog that was closed.
	TagWin32Cancelled = "Win32Cancelled"
	// tagGPUOutOfMemory and tagGPUDeviceLost are the tags of SPEC §9.3, which both renderers use.
	tagGPUOutOfMemory = "GPUOutOfMemory"
	tagGPUDeviceLost  = "GPUDeviceLost"
)
