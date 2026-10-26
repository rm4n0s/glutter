//go:build windows

package callshim

import (
	"math"
	"syscall"
)

// callDoubles is the shim in assembly (callshim_windows_amd64.s, callshim_windows_arm64.s): a function of
// the Windows calling convention, called through SyscallN as shim(fn, this, xbits, ybits, ret).
//
//nolint:unused // implemented in assembly; entry returns its address
func callDoubles()

// entry is the address of callDoubles.
func entry() uintptr

// Call calls the COM method at fn with this, then two doubles, then ret (the pointer of the result, or 0)
// in the places the Windows calling convention gives them: on amd64 this is RCX, the doubles XMM1 and
// XMM2 and ret R9; on arm64 this is X0, the doubles D0 and D1 and ret X1. A method with one double
// (SetValue) is called with y 0; one with none of the doubles is not for this function. It returns the
// value of RAX (the HRESULT). A pointer converted in the argument list (uintptr(unsafe.Pointer(&x))) stays
// alive and is moved to the heap for the call, since the method may run Go code that moves the stack.
//
// Flutter: no counterpart
//
//go:uintptrescapes
func Call(fn, this uintptr, x, y float64, ret uintptr) uintptr {
	r, _, _ := syscall.SyscallN(entry(), fn, this, uintptr(math.Float64bits(x)), uintptr(math.Float64bits(y)), ret)
	return r
}
