#include "textflag.h"

// The Windows arm64 convention passes this in X0 and the integer arguments in X1 to X7, and the doubles
// in D0 to D7 separately. A callback made by syscall.NewCallback reads X0 to X7. Each thunk moves the
// bits of the doubles from D0 and D1 to X1 and X2, and the pointer for the result from X1 to X3, and
// jumps to the callback. Nothing is pushed: the callback returns to Windows' caller. (This file builds
// and vets; no arm64 Windows runs it here.)

// ElementProviderFromPoint(this, double x, double y, **result) -> callback(this, xbits, ybits, result)
TEXT ·thunkFromPoint(SB), NOSPLIT|NOFRAME, $0-0
	MOVD R1, R3
	FMOVD F0, R1
	FMOVD F1, R2
	MOVD ·thunkTargets+0(SB), R9
	JMP (R9)

// SetScrollPercent(this, double h, double v) -> callback(this, hbits, vbits, junk)
TEXT ·thunkScrollPercent(SB), NOSPLIT|NOFRAME, $0-0
	FMOVD F0, R1
	FMOVD F1, R2
	MOVD ·thunkTargets+8(SB), R9
	JMP (R9)

// SetValue(this, double value) -> callback(this, valuebits)
TEXT ·thunkRangeValue(SB), NOSPLIT|NOFRAME, $0-0
	FMOVD F0, R1
	MOVD ·thunkTargets+16(SB), R9
	JMP (R9)

// func thunkEntries() (point, scroll, rng uintptr)
TEXT ·thunkEntries(SB), NOSPLIT, $0-24
	MOVD $·thunkFromPoint(SB), R0
	MOVD R0, point+0(FP)
	MOVD $·thunkScrollPercent(SB), R0
	MOVD R0, scroll+8(FP)
	MOVD $·thunkRangeValue(SB), R0
	MOVD R0, rng+16(FP)
	RET
