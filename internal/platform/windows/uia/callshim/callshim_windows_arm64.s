#include "textflag.h"

// func entry() uintptr
TEXT ·entry(SB), NOSPLIT, $0-8
	MOVD $·callDoubles(SB), R0
	MOVD R0, ret+0(FP)
	RET

// callDoubles is called by SyscallN with the Windows arm64 convention (X0 to X4 are the five
// arguments): X0 = fn, X1 = this, X2 = bits of x, X3 = bits of y, X4 = ret. It calls fn(this, x, y,
// ret) with this in X0, x in D0, y in D1 and ret in X1, and returns X0.
TEXT ·callDoubles(SB), NOSPLIT, $16-0
	MOVD R0, R9
	MOVD R1, R0
	FMOVD R2, F0
	FMOVD R3, F1
	MOVD R4, R1
	BL (R9)
	RET
