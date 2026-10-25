//go:build windows

package win32

import (
	"sync/atomic"
	"syscall"
)

// callbacks counts the callbacks made so far. It is the one counter of the process, because the
// callbacks themselves are process-wide (the runtime keeps a table of about 2,000 and never frees
// an entry).
var callbacks atomic.Int32

// NewCallback makes a callback that Windows can call: a function with uintptr-sized parameters and
// one uintptr-sized result (syscall.NewCallback). Make them once at startup and dispatch by window
// handle or object handle (SPEC §6.3 rule 11); every callback of the process goes through this
// function, so that CallbackCount knows them all. It panics, as syscall.NewCallback does, when
// the function has the wrong shape or the table is full.
//
// Flutter: no counterpart
func NewCallback(fn any) uintptr {
	callbacks.Add(1)
	return syscall.NewCallback(fn)
}

// CallbackCount is the number of callbacks made by NewCallback so far. A test fails when it is above
// CallbackLimit.
//
// Flutter: no counterpart
func CallbackCount() int { return int(callbacks.Load()) }
