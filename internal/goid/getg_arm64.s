#include "textflag.h"

// func Token() uint64
//
// The address of the running goroutine's g, which arm64 keeps in R28 (the
// assembler's name for it is g).
TEXT ·Token(SB),NOSPLIT,$0-8
	MOVD g, R0
	MOVD R0, ret+0(FP)
	RET
