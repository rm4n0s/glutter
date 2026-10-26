#include "textflag.h"

// The Windows x64 convention passes this in RCX, and a double that follows it in XMM1, the next in XMM2.
// A callback made by syscall.NewCallback reads its arguments from RCX, RDX, R8 and R9. Each thunk moves
// the bits of the doubles from XMM1 and XMM2 to RDX and R8 and jumps to the callback; R9 (the pointer
// for the result) is not touched. Nothing is pushed: the callback returns to Windows' caller.

// ElementProviderFromPoint(this, double x, double y, **result) -> callback(this, xbits, ybits, result)
TEXT ·thunkFromPoint(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ X2, R8
	MOVQ ·thunkTargets+0(SB), AX
	JMP AX

// SetScrollPercent(this, double h, double v) -> callback(this, hbits, vbits, junk)
TEXT ·thunkScrollPercent(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ X2, R8
	MOVQ ·thunkTargets+8(SB), AX
	JMP AX

// SetValue(this, double value) -> callback(this, valuebits)
TEXT ·thunkRangeValue(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ ·thunkTargets+16(SB), AX
	JMP AX

// func thunkEntries() (point, scroll, rng uintptr)
TEXT ·thunkEntries(SB), NOSPLIT, $0-24
	LEAQ ·thunkFromPoint(SB), AX
	MOVQ AX, point+0(FP)
	LEAQ ·thunkScrollPercent(SB), AX
	MOVQ AX, scroll+8(FP)
	LEAQ ·thunkRangeValue(SB), AX
	MOVQ AX, rng+16(FP)
	RET
