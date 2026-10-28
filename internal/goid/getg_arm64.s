#include "textflag.h"

// func gword(off uintptr) uint64
//
// The 8-byte word at byte offset off & 0xf8 of the running goroutine's g, which
// arm64 keeps in R28 (the assembler's name for it is g). The mask keeps the read
// inside the first 256 bytes of the g whatever off is, and aligned.
TEXT ·gword(SB),NOSPLIT,$0-16
	MOVD off+0(FP), R1
	AND $0xf8, R1
	MOVD g, R0
	ADD R1, R0, R0
	MOVD (R0), R0
	MOVD R0, ret+8(FP)
	RET
