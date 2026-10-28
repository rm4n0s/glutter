#include "textflag.h"

// func Token() uint64
//
// The address of the running goroutine's g, from thread local storage, as the
// runtime reads it. Called only from Go code, so the thread's g is the goroutine
// that called.
TEXT ·Token(SB),NOSPLIT,$0-8
	MOVQ (TLS), AX
	MOVQ AX, ret+0(FP)
	RET
