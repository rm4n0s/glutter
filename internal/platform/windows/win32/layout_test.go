//go:build !windows

package win32

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32/layouttest"
)

// TestLayoutAgainstTheHeaders checks the Go mirrors of this package, the positions of the methods of
// IUnknown and the HRESULT values against the mingw-w64 headers (SPEC §25.2). It runs on the
// Linux host and is skipped, with the install command, when the compiler is missing.
func TestLayoutAgainstTheHeaders(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"windows.h", "unknwn.h", "dxgi.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("GUID", GUID{}),
			layouttest.StructOf("POINT", Point{}),
			layouttest.StructOf("RECT", Rect{}),
			layouttest.StructOf("WNDCLASSEXW", WndClassEx{}),
			layouttest.StructOf("MSG", Msg{}),
		},
		VTables: []layouttest.VTable{{CName: "IUnknownVtbl", Methods: []layouttest.Method{
			{Name: "QueryInterface", Index: IUnknownQueryInterface},
			{Name: "AddRef", Index: IUnknownAddRef},
			{Name: "Release", Index: IUnknownRelease},
		}}},
		Consts: []layouttest.Const{
			{Name: "S_OK", Value: uint32(SOK)},
			{Name: "E_NOTIMPL", Value: ENotImpl.Code()},
			{Name: "E_NOINTERFACE", Value: ENoInterface.Code()},
			{Name: "E_POINTER", Value: EPointer.Code()},
			{Name: "E_FAIL", Value: EFail.Code()},
			{Name: "E_OUTOFMEMORY", Value: EOutOfMemory.Code()},
			{Name: "E_INVALIDARG", Value: EInvalidArg.Code()},
			{Name: "HRESULT_FROM_WIN32(ERROR_CANCELLED)", Value: ErrorCancelled.Code()},
			{Name: "ERROR_CANCELLED", Value: ErrorCancelledCode},
			{Name: "DXGI_ERROR_DEVICE_HUNG", Value: DXGIErrorDeviceHung.Code()},
			{Name: "DXGI_ERROR_DEVICE_REMOVED", Value: DXGIErrorDeviceRemoved.Code()},
			{Name: "DXGI_ERROR_DEVICE_RESET", Value: DXGIErrorDeviceReset.Code()},
			{Name: "HEAP_ZERO_MEMORY", Value: HeapZeroMemory},
			{Name: "GMEM_MOVEABLE", Value: GMemMoveable},
			{Name: "GMEM_ZEROINIT", Value: GMemZeroInit},
			{Name: "WAIT_OBJECT_0", Value: uint32(WaitObject0)},
			{Name: "WAIT_ABANDONED_0", Value: uint32(WaitAbandoned0)},
			{Name: "WAIT_TIMEOUT", Value: uint32(WaitTimeout)},
			{Name: "WAIT_FAILED", Value: uint32(waitFailed)},
		},
	})
}
