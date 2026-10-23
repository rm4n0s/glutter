package android

/*
#include "jni_android.h"
#include <stdlib.h>
*/
import "C"

import (
	"os"
	"syscall"
	"unsafe"
)

// startLogging sends standard output and standard error to logcat, line by line (SPEC §24.9):
// the tag is the application ID (GLUTTER_APP_ID, set by the activity), and the priority comes
// from the level of the line. An Android process has no terminal, so nothing else would show
// what the program prints.
func startLogging() {
	tag := os.Getenv("GLUTTER_APP_ID")
	if tag == "" {
		tag = "glutter"
	}
	fd := int(C.glutter_redirect_stdio())
	if fd < 0 {
		return
	}
	ctag := C.CString(tag)
	go func() {
		defer C.free(unsafe.Pointer(ctag))
		var pending string
		buf := make([]byte, 4096)
		for {
			n, err := syscall.Read(fd, buf)
			if n > 0 {
				var lines []string
				lines, pending = splitLines(pending + string(buf[:n]))
				for _, l := range lines {
					writeLog(ctag, l)
				}
			}
			if err != nil && err != syscall.EINTR {
				if pending != "" {
					writeLog(ctag, pending)
				}
				return
			}
			if n == 0 && err == nil {
				return
			}
		}
	}()
}

// writeLog sends one line to logcat. A line longer than logcat's limit is cut by logcat.
func writeLog(tag *C.char, line string) {
	cl := C.CString(line)
	C.glutter_log(C.int(linePriority(line)), tag, cl)
	C.free(unsafe.Pointer(cl))
}
