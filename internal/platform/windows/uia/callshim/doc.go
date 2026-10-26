// Package callshim calls a function of the Windows calling convention with floating-point arguments, for
// the tests of UI Automation (SPEC §25.2): syscall.SyscallN passes integers only, and a COM method that
// takes a double (IRawElementProviderFragmentRoot::ElementProviderFromPoint,
// IRangeValueProvider::SetValue, and the client's SetValue and SetScrollPercent) gets it in a floating
// register. The shim is a function in Go assembly that SyscallN calls with the bits of the doubles; it
// moves them into the registers and calls the method. Nothing but tests uses it.
//
// Flutter: no counterpart
package callshim
