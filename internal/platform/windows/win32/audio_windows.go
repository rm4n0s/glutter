//go:build windows

package win32

import (
	"unsafe"
)

// The few bindings that WASAPI needs beyond the COM calls (SPEC §28.3): the clearing of a PROPVARIANT, and the views of memory that
// the audio client and the property stores own.

var avrt = &dllTable{name: "avrt.dll"}

// CoCreateInstanceHR is CoCreateInstance for a caller that tells the failures itself: the class is made in the contexts clsctx
// (ClsctxAll) and the result is the HRESULT as it came (EFail when ole32.dll cannot be loaded). The object is a new reference to
// release.
//
// Flutter: no counterpart
func CoCreateInstanceHR(clsid, iid *GUID, clsctx uint32) (Object, HRESULT) {
	var out uintptr
	r, _, err := ole32.call("CoCreateInstance", uintptr(unsafe.Pointer(clsid)), 0, uintptr(clsctx),
		uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err != nil {
		return Object{}, EFail
	}
	return Object{ptr: out}, FromResult(r)
}

// AvSetMmThreadCharacteristics puts the calling thread in the multimedia class task (such as "Pro Audio"), which the scheduler
// serves ahead of ordinary threads, and returns the handle for AvRevertMmThreadCharacteristics. ok is false when this Windows has no
// avrt.dll or refuses (the thread keeps its priority).
//
// Flutter: no counterpart
func AvSetMmThreadCharacteristics(task string) (handle uintptr, ok bool) {
	p := avrt.optional("AvSetMmThreadCharacteristicsW")
	name, err := UTF16PtrFromString(task)
	if p == nil || err != nil {
		return 0, false
	}
	var index uint32
	r, _ := p.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&index)))
	return r, r != 0
}

// AvRevertMmThreadCharacteristics gives the thread back its class; the handle is the one that AvSetMmThreadCharacteristics made.
//
// Flutter: no counterpart
func AvRevertMmThreadCharacteristics(handle uintptr) {
	if p := avrt.optional("AvRevertMmThreadCharacteristics"); p != nil && handle != 0 {
		_, _ = p.Call(handle)
	}
}

// PropVariantClear frees what a PROPVARIANT holds (the string that GetValue of a property store made) and makes it empty.
//
// Flutter: no counterpart
func PropVariantClear(p *PropVariant) {
	_, _, _ = ole32.call("PropVariantClear", uintptr(unsafe.Pointer(p)))
}

// Float32s returns a slice over n float32 of memory that Windows owns, starting at addr: the buffer of an audio render or capture
// client between GetBuffer and ReleaseBuffer. The slice is only valid until the buffer is released. It is nil for a null address
// or n of 0.
//
// Flutter: no counterpart
func Float32s(addr uintptr, n int) []float32 {
	if addr == 0 || n <= 0 {
		return nil
	}
	return unsafe.Slice((*float32)(at(addr)), n)
}

// WideString copies the NUL-terminated UTF-16 string that Windows owns at addr into a Go string; "" for a null address. It is how a
// string that a COM method returned (an endpoint's ID, which CoTaskMemFree frees) is read.
//
// Flutter: no counterpart
func WideString(addr uintptr) string {
	if addr == 0 {
		return ""
	}
	return UTF16PtrToString((*uint16)(at(addr)))
}
