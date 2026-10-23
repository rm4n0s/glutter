package android

/*
#include "jni_android.h"
#include <stdlib.h>
*/
import "C"

import (
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// startLogging sends standard output and standard error to logcat, line by line (SPEC §24.9):
// the tag is the application ID (GLUTTER_APP_ID, set by the activity), and the priority comes
// from the level of the line. An Android process has no terminal, so nothing else would show
// what the program prints.
func startLogging() {
	logDrained = make(chan struct{}, 1)
	tag := os.Getenv("GLUTTER_APP_ID")
	if tag == "" {
		tag = "glutter"
	}
	fd := int(C.glutter_redirect_stdio())
	if fd < 0 {
		return
	}
	ctag := C.CString(tag) // kept for the life of the process: the standard logger uses it too
	// The standard logger, which log.Fatal, log.Print and the default slog logger write through, goes
	// to logcat at once, from the calling goroutine: log.Fatal ends the process in the next
	// instant, and a line that still waited in the pipe would be lost with it.
	log.SetOutput(logcatWriter{tag: ctag})
	go func() {
		var pending string
		buf := make([]byte, 4096)
		for {
			n, err := syscall.Read(fd, buf)
			if n > 0 {
				var lines []string
				lines, pending = splitLines(pending + string(buf[:n]))
				for _, l := range lines {
					if l == logEndMarker {
						select {
						case logDrained <- struct{}{}:
						default:
						}
						continue
					}
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

// logcatWriter is the output of the standard logger: each line of a write goes to logcat before
// Write returns.
type logcatWriter struct{ tag *C.char }

// Write implements io.Writer.
func (w logcatWriter) Write(p []byte) (int, error) {
	lines, rest := splitLines(string(p))
	for _, l := range lines {
		writeLog(w.tag, l)
	}
	if rest != "" {
		writeLog(w.tag, rest)
	}
	return len(p), nil
}

// logDrained is signaled by the reader of the pipe when it has read the end marker; nil until
// startLogging runs. It is set before the reader and the program start.
var logDrained chan struct{}

// flushLogs waits (at most a second) until what the program wrote to standard output and error has
// been sent to logcat: it writes the end marker and waits for the reader to get to it. Without it the
// last lines of a program that ends at once, the error that ended it, can be lost with the process.
func flushLogs() {
	if logDrained == nil {
		return
	}
	if _, err := os.Stderr.WriteString(logEndMarker + "\n"); err != nil {
		return // nothing can be done about a closed standard error
	}
	select {
	case <-logDrained:
	case <-time.After(time.Second):
	}
}

// writeLog sends one line to logcat. A line longer than logcat's limit is cut by logcat.
func writeLog(tag *C.char, line string) {
	cl := C.CString(line)
	C.glutter_log(C.int(linePriority(line)), tag, cl)
	C.free(unsafe.Pointer(cl))
}
