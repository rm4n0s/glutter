//go:build windows

package uia

// The thunks of the three COM methods that take a double (SPEC §25.2, §25.11). Windows passes a double
// in a floating-point register (XMM1, XMM2 on amd64; D0, D1 on arm64), and a function made by
// syscall.NewCallback sees only the integer registers. Each thunk is a function in assembly (thunk_windows_amd64.s,
// thunk_windows_arm64.s) that Windows calls through the vtable: it moves the bits of the doubles into
// the integer registers where the callback expects its arguments, and jumps to the callback, whose
// address it reads from thunkTargets. The callback gets (this, bits of the first double, bits of the
// second, the pointer for the result); a thunk touches nothing else, so the return address and the
// shadow space are the caller's and the callback returns straight to Windows.

// The positions in thunkTargets of the callbacks.
const (
	// thunkIndexPoint is IRawElementProviderFragmentRoot::ElementProviderFromPoint(double x, double y, ...).
	thunkIndexPoint = iota
	// thunkIndexScroll is IScrollProvider::SetScrollPercent(double h, double v).
	thunkIndexScroll
	// thunkIndexRange is IRangeValueProvider::SetValue(double value).
	thunkIndexRange
)

// thunkTargets are the addresses of the callbacks the thunks jump to. They are written once, before the
// vtables that hold the thunks exist, and only read after that.
var thunkTargets [3]uintptr

// thunkFromPoint, thunkScrollPercent and thunkRangeValue are the thunks, in assembly.
//
//nolint:unused // implemented in assembly; the vtables get its address from thunkEntries
func thunkFromPoint()

//nolint:unused // implemented in assembly
func thunkScrollPercent()

//nolint:unused // implemented in assembly
func thunkRangeValue()

// thunkEntries are the addresses of the thunks.
func thunkEntries() (point, scroll, rng uintptr)
