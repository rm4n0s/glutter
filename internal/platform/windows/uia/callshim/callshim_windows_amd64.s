#include "textflag.h"

// func entry() uintptr
TEXT ·entry(SB), NOSPLIT, $0-8
	LEAQ ·callDoubles(SB), AX
	MOVQ AX, ret+0(FP)
	RET

// callDoubles is called by SyscallN with the Windows x64 convention:
// RCX = fn, RDX = this, R8 = bits of x, R9 = bits of y, 40(SP) = ret (the fifth argument, after the
// return address and the 32 bytes of shadow space). It calls fn(this, x, y, ret) with x in XMM1, y in
// XMM2 and ret in R9, with a shadow space of its own, and returns RAX.
TEXT ·callDoubles(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ 40(SP), R10
	MOVQ CX, AX
	SUBQ $40, SP
	MOVQ DX, CX
	MOVQ R8, X1
	MOVQ R9, X2
	MOVQ R10, R9
	CALL AX
	ADDQ $40, SP
	RET
