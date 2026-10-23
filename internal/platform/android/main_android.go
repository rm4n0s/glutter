package android

/*
#include "jni_android.h"
#include <stdlib.h>
*/
import "C"

import (
	"log/slog"
	"os"
	"runtime"
	"sync"
	_ "unsafe" // go:linkname
)

// callmain is the program's main.main. A shared library does not run it by itself, so
// Native.start does, as gomobile and Gio do (SPEC §24.2). The empty file stub_android.s makes
// the compiler accept a function with no body.
//
//go:linkname callmain main.main
func callmain()

var startOnce sync.Once

// goSetenv is Native.setenv: it sets a variable of the Go program, which has copied the
// environment of the process when it started.
//
//export goSetenv
func goSetenv(key, value *C.char) {
	if err := os.Setenv(C.GoString(key), C.GoString(value)); err != nil {
		slog.Error("glutter: setenv failed", "err", err)
	}
}

// goStart is Native.start: it runs main on a goroutine locked to its own OS thread, once per
// process. That thread is the UI thread (SPEC §24.3). It returns at once.
//
//export goStart
func goStart() {
	startOnce.Do(func() {
		startLogging()
		go runMain()
	})
}

// runMain calls the program's main on the thread it is locked to, and tells the Java side
// that the program ended when main returns.
func runMain() {
	runtime.LockOSThread()
	callmain()
	// main returned: the activity finishes and the process exits with the code 0. (os.Exit
	// ends the process at once and does not come back here.)
	C.glutter_main_ended(0)
	// Do not return: a locked goroutine that returns ends its thread while the JavaVM may still
	// use it; the process is exiting.
	select {}
}
