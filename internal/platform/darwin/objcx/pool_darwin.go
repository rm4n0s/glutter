//go:build darwin

package objcx

import (
	"runtime"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// poolFuncs are objc_autoreleasePoolPush and objc_autoreleasePoolPop of libobjc.
type poolFuncs struct {
	push func() uintptr
	pop  func(token uintptr)
}

var pool = sync.OnceValue(func() *poolFuncs {
	lib, err := dl.Open("/usr/lib/libobjc.A.dylib")
	if err != nil {
		panic(err)
	}
	f := new(poolFuncs)
	if err := lib.Bind(&f.push, "objc_autoreleasePoolPush"); err != nil {
		panic(err)
	}
	if err := lib.Bind(&f.pop, "objc_autoreleasePoolPop"); err != nil {
		panic(err)
	}
	return f
})

// WithPool runs fn inside an autorelease pool and drains the pool when fn returns, even when it panics:
// the objects that fn autoreleased are released then. A pool belongs to a thread, so WithPool locks the
// goroutine to its OS thread for as long as fn runs (nested calls and the locked main thread are fine).
// Each turn of the run loop, each callback from the system and each call from another goroutine that
// makes Objective-C objects runs inside it (SPEC §29.2).
//
// Flutter: no counterpart
func WithPool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	p := pool()
	token := p.push()
	defer p.pop(token)
	fn()
}
