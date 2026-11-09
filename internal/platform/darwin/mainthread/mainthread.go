// Package mainthread runs the tests of a macOS program on the main thread: AppKit works only there (SPEC
// §29.3), and the Go test runner runs every test on a goroutine of its own. A test binary that opens
// windows calls Main from TestMain; its tests then call Do, which runs a function on the main thread
// and waits for it. Only tests import it (it has no use in a program, which calls Run from main).
//
// Flutter: no counterpart
package mainthread

import (
	"os"
	"runtime"
	"testing"
)

// init locks the main goroutine, which runs the program's init functions, to the main thread of the
// process: the thread that the runtime started on.
func init() { runtime.LockOSThread() }

// jobs are the functions that Do hands to the main goroutine.
var jobs = make(chan func())

// Main runs the tests (m.Run) on a goroutine of their own while the main goroutine, which is the main
// thread, serves Do, and exits the process with the result. Call it as the last line of TestMain:
//
//	func TestMain(m *testing.M) { mainthread.Main(m) }
//
// Flutter: no counterpart
func Main(m *testing.M) {
	done := make(chan int, 1)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-jobs:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

// Do runs fn on the main thread and returns when it has returned. A panic of fn is raised again in the
// caller. fn must not call t.Fatal or runtime.Goexit: that would end the main thread's loop; it reports
// with t.Error and returns, and a test that waits for events does so in its own goroutine.
//
// Flutter: no counterpart
func Do(fn func()) {
	result := make(chan any, 1)
	jobs <- func() {
		defer func() { result <- recover() }()
		fn()
	}
	if r := <-result; r != nil {
		panic(r)
	}
}
