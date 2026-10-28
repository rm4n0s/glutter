#include "textflag.h"

// func gword(off uintptr) uint64
//
// The 8-byte word at byte offset off & 0xf8 of the running goroutine's g, which
// the runtime keeps in thread local storage. The mask keeps the read inside the
// first 256 bytes of the g whatever off is, and aligned. Called only from Go
// code, so the thread's g is the goroutine that called.
TEXT ·gword(SB),NOSPLIT,$0-16
	MOVQ (TLS), AX
	MOVQ off+0(FP), CX
	ANDQ $0xf8, CX
	MOVQ (AX)(CX*1), AX
	MOVQ AX, ret+8(FP)
	RET
