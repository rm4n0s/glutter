//go:build windows

package win32

import (
	"syscall"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// Object is a COM object that Go calls: the address of its interface pointer, which points to the
// vtable (an array of function pointers). Every method is called with its index in the vtable
// (SPEC §25.2); the indexes are constants next to the Go type of each interface, and a layout test
// checks them against the headers. An Object does not count its references: whoever got it from a
// method that returns a new reference calls Release.
//
// Flutter: no counterpart
type Object struct{ ptr uintptr }

// ObjectFromPtr wraps the address of a COM interface pointer, such as the one a method wrote
// through its out parameter.
//
// Flutter: no counterpart
func ObjectFromPtr(ptr uintptr) Object { return Object{ptr: ptr} }

// Ptr is the address of the interface pointer, the first argument (this) of every method.
//
// Flutter: no counterpart
func (o Object) Ptr() uintptr { return o.ptr }

// IsNil is whether the object is the null pointer.
//
// Flutter: no counterpart
func (o Object) IsNil() bool { return o.ptr == 0 }

// method is the address of the method at index in the object's vtable.
func (o Object) method(index int) uintptr {
	return loadPtr(loadPtr(o.ptr) + uintptr(index)*unsafe.Sizeof(uintptr(0)))
}

// Call calls the method at index with the object as this and the arguments after it, and returns
// RAX. A method that returns an HRESULT is read with FromResult, or called with CallHR. A pointer
// argument is converted in the argument list (uintptr(unsafe.Pointer(&x))), which keeps it alive for
// the call. A method that returns a structure is called as the C header declares it, with a hidden
// pointer to the result as the first argument after this (SPEC §25.2). It panics with COMNilObject on
// a nil Object, a programmer error.
//
// Flutter: no counterpart
//
//go:uintptrescapes
func (o Object) Call(index int, args ...uintptr) uintptr {
	if o.ptr == 0 {
		panic(errors.New(tagCOMNilObject, "a method of a nil COM object was called", "index", index))
	}
	var buf [16]uintptr
	all := append(buf[:0], o.ptr)
	all = append(all, args...)
	r, _, _ := syscall.SyscallN(o.method(index), all...)
	return r
}

// CallHR is Call for a method that returns an HRESULT: a failed result is the tagged error of
// HResultError, with method (such as "ID3D12Device::CreateFence") as the metadata "method".
//
// Flutter: no counterpart
//
//go:uintptrescapes
func (o Object) CallHR(method string, index int, args ...uintptr) error {
	return HResultError(method, FromResult(o.Call(index, args...)))
}

// QueryInterface asks the object for another interface (IUnknown::QueryInterface). The result is
// a new reference. It fails with COMCallFailed, with the metadata "hresult" 0x80004002, when the
// object does not have the interface.
//
// Flutter: no counterpart
func (o Object) QueryInterface(iid *GUID) (Object, error) {
	var out uintptr
	if err := o.CallHR("IUnknown::QueryInterface", IUnknownQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))); err != nil {
		return Object{}, err
	}
	return Object{ptr: out}, nil
}

// AddRef adds a reference (IUnknown::AddRef) and returns the new count, which is only for
// diagnostics.
//
// Flutter: no counterpart
func (o Object) AddRef() uint32 { return uint32(o.Call(IUnknownAddRef)) }

// Release drops a reference (IUnknown::Release) and returns the new count, which is only for
// diagnostics. The object may be gone afterwards.
//
// Flutter: no counterpart
func (o Object) Release() uint32 { return uint32(o.Call(IUnknownRelease)) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCOMNilObject = "COMNilObject"
)
