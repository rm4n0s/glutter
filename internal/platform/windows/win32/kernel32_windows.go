//go:build windows

package win32

import (
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The first bindings of kernel32.dll (their flags and results are in consts.go): the heap and global memory (for the COM objects and the
// clipboard), the module handle, sleeping, and events with their waits. Later steps add the
// bindings they need next to the code that uses them, or here when several packages share them.

// kernel32Procs is the table of the functions this file binds.
type kernel32Procs struct {
	getProcessHeap, heapAlloc, heapFree, getModuleHandleW, globalAlloc, globalFree, sleep,
	createEventW, setEvent, resetEvent, waitForSingleObject, waitForMultipleObjects, closeHandle *Proc
}

// kernel32 loads the table when it is first needed. The value never changes afterwards.
var kernel32 = sync.OnceValues(loadKernel32)

// loadKernel32 finds the functions of the table.
func loadKernel32() (*kernel32Procs, error) {
	lib, err := LoadSystemDLL("kernel32.dll")
	if err != nil {
		return nil, err
	}
	t := &kernel32Procs{}
	for _, b := range []struct {
		name string
		to   **Proc
	}{
		{"GetProcessHeap", &t.getProcessHeap}, {"HeapAlloc", &t.heapAlloc}, {"HeapFree", &t.heapFree},
		{"GetModuleHandleW", &t.getModuleHandleW}, {"GlobalAlloc", &t.globalAlloc}, {"GlobalFree", &t.globalFree},
		{"Sleep", &t.sleep}, {"CreateEventW", &t.createEventW}, {"SetEvent", &t.setEvent},
		{"ResetEvent", &t.resetEvent}, {"WaitForSingleObject", &t.waitForSingleObject},
		{"WaitForMultipleObjects", &t.waitForMultipleObjects}, {"CloseHandle", &t.closeHandle},
	} {
		if *b.to, err = lib.Proc(b.name); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// ProcessHeap is the default heap of the process (GetProcessHeap).
//
// Flutter: no counterpart
func ProcessHeap() (uintptr, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	h, errno := k.getProcessHeap.Call()
	if h == 0 {
		return 0, Win32Error("GetProcessHeap", uint32(errno))
	}
	return h, nil
}

// HeapAlloc allocates size bytes from a heap (HeapAlloc) and returns the address. Memory that
// Windows keeps a pointer to (COM objects, vtables) comes from here, never from Go (SPEC §6.3 rule
// 11). Free it with HeapFree.
//
// Flutter: no counterpart
func HeapAlloc(heap uintptr, flags uint32, size uintptr) (uintptr, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	p, errno := k.heapAlloc.Call(heap, uintptr(flags), size)
	if p == 0 {
		return 0, Win32Error("HeapAlloc", uint32(errno))
	}
	return p, nil
}

// HeapFree frees a block of HeapAlloc.
//
// Flutter: no counterpart
func HeapFree(heap uintptr, flags uint32, mem uintptr) error {
	k, err := kernel32()
	if err != nil {
		return err
	}
	if r, errno := k.heapFree.Call(heap, uintptr(flags), mem); r == 0 {
		return Win32Error("HeapFree", uint32(errno))
	}
	return nil
}

// GetModuleHandle returns the handle (HMODULE, HINSTANCE) of a module that is loaded in the
// process, or of the program's own module when name is empty.
//
// Flutter: no counterpart
func GetModuleHandle(name string) (uintptr, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	var arg *uint16
	if name != "" {
		p, err := UTF16PtrFromString(name)
		if err != nil {
			return 0, err
		}
		arg = p
	}
	h, errno := k.getModuleHandleW.Call(uintptr(unsafe.Pointer(arg)))
	if h == 0 {
		return 0, Win32Error("GetModuleHandleW", uint32(errno))
	}
	return h, nil
}

// GlobalAlloc allocates global memory (GlobalAlloc), which the clipboard and OLE data transfer take.
// Free it with GlobalFree unless Windows took the ownership.
//
// Flutter: no counterpart
func GlobalAlloc(flags uint32, size uintptr) (uintptr, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	h, errno := k.globalAlloc.Call(uintptr(flags), size)
	if h == 0 {
		return 0, Win32Error("GlobalAlloc", uint32(errno))
	}
	return h, nil
}

// GlobalFree frees global memory from GlobalAlloc.
//
// Flutter: no counterpart
func GlobalFree(h uintptr) error {
	k, err := kernel32()
	if err != nil {
		return err
	}
	if r, errno := k.globalFree.Call(h); r != 0 {
		return Win32Error("GlobalFree", uint32(errno))
	}
	return nil
}

// Sleep suspends the calling thread for d, in whole milliseconds.
//
// Flutter: no counterpart
func Sleep(d time.Duration) error {
	k, err := kernel32()
	if err != nil {
		return err
	}
	_, _ = k.sleep.Call(uintptr(max(d.Milliseconds(), 0))) // Sleep returns nothing
	return nil
}

// CreateEvent makes an event object (CreateEventW) without a name. A manual-reset event stays
// signaled until ResetEvent; an auto-reset event lets one waiting thread through and resets itself.
// Close it with CloseHandle.
//
// Flutter: no counterpart
func CreateEvent(manualReset, initialState bool) (uintptr, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	h, errno := k.createEventW.Call(0, boolArg(manualReset), boolArg(initialState), 0)
	if h == 0 {
		return 0, Win32Error("CreateEventW", uint32(errno))
	}
	return h, nil
}

// SetEvent signals an event.
//
// Flutter: no counterpart
func SetEvent(h uintptr) error {
	k, err := kernel32()
	if err != nil {
		return err
	}
	if r, errno := k.setEvent.Call(h); r == 0 {
		return Win32Error("SetEvent", uint32(errno))
	}
	return nil
}

// ResetEvent clears an event.
//
// Flutter: no counterpart
func ResetEvent(h uintptr) error {
	k, err := kernel32()
	if err != nil {
		return err
	}
	if r, errno := k.resetEvent.Call(h); r == 0 {
		return Win32Error("ResetEvent", uint32(errno))
	}
	return nil
}

// WaitForSingleObject waits until the object is signaled or timeout has passed (WaitInfinite never
// ends). The result is WaitObject0, WaitAbandoned0 or WaitTimeout.
//
// Flutter: no counterpart
func WaitForSingleObject(h uintptr, timeout time.Duration) (WaitResult, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	r, errno := k.waitForSingleObject.Call(h, uintptr(timeoutMillis(timeout)))
	if WaitResult(r) == waitFailed {
		return 0, Win32Error("WaitForSingleObject", uint32(errno))
	}
	return WaitResult(r), nil
}

// WaitForMultipleObjects waits for any of the handles (all of them when all is true) to be
// signaled, or for timeout. At most 64 handles fit. The result is WaitObject0 + the index of the
// handle that ended the wait (or WaitAbandoned0 + the index, or WaitTimeout).
//
// Flutter: no counterpart
func WaitForMultipleObjects(handles []uintptr, all bool, timeout time.Duration) (WaitResult, error) {
	k, err := kernel32()
	if err != nil {
		return 0, err
	}
	if len(handles) == 0 || len(handles) > 64 {
		return 0, errors.New(TagWin32CallFailed, "WaitForMultipleObjects takes 1 to 64 handles", "function", "WaitForMultipleObjects", "count", len(handles))
	}
	r, errno := k.waitForMultipleObjects.Call(uintptr(len(handles)), uintptr(unsafe.Pointer(&handles[0])), boolArg(all), uintptr(timeoutMillis(timeout)))
	if WaitResult(r) == waitFailed {
		return 0, Win32Error("WaitForMultipleObjects", uint32(errno))
	}
	return WaitResult(r), nil
}

// CloseHandle closes a handle of an event or another kernel object.
//
// Flutter: no counterpart
func CloseHandle(h uintptr) error {
	k, err := kernel32()
	if err != nil {
		return err
	}
	if r, errno := k.closeHandle.Call(h); r == 0 {
		return Win32Error("CloseHandle", uint32(errno))
	}
	return nil
}

// boolArg is a Go bool as the BOOL that a Win32 function takes.
func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// timeoutMillis is a Go duration as the milliseconds of a wait, with WaitInfinite as INFINITE
// (0xFFFFFFFF) and a longer wait cut to the longest finite one.
func timeoutMillis(d time.Duration) uint32 {
	if d < 0 {
		return 0xFFFFFFFF
	}
	return uint32(min(d.Milliseconds(), 0xFFFFFFFE))
}
