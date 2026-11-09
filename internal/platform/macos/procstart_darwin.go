//go:build darwin && !ios

package macos

import (
	"encoding/binary"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// sysctlFunc is libSystem's sysctl, bound on first use: the standard library's syscall package has no sysctl with a MIB on darwin.
var sysctlFunc = sync.OnceValue(func() func(name *int32, namelen uint32, oldp unsafe.Pointer, oldlenp *uintptr, newp unsafe.Pointer, newlen uintptr) int32 {
	lib, err := dl.Open("/usr/lib/libSystem.B.dylib")
	if err != nil {
		return nil
	}
	var f func(name *int32, namelen uint32, oldp unsafe.Pointer, oldlenp *uintptr, newp unsafe.Pointer, newlen uintptr) int32
	if err := lib.Bind(&f, "sysctl"); err != nil {
		return nil
	}
	return f
})

// ProcessStart returns when this process started, as the kernel says: the engine logs the time from it to the first frame
// ("since_process_start"), which includes what happens before Go's runtime starts (the dynamic loader, the runtime's own start).
// ok is false when the kernel does not say.
//
// Flutter: no counterpart
func ProcessStart() (time.Time, bool) { return processStartOf(os.Getpid()) }

// processStartOf reads the start time of a process from its kinfo_proc (sysctl kern.proc.pid): the structure begins with the
// extern_proc, whose first member is a union that holds the start time as a timeval, an int64 of seconds and an int32 of
// microseconds, in the machine's byte order.
func processStartOf(pid int) (time.Time, bool) {
	sysctl := sysctlFunc()
	if sysctl == nil {
		return time.Time{}, false
	}
	// CTL_KERN, KERN_PROC, KERN_PROC_PID and the pid; kinfo_proc is about 650 bytes, and the size is asked for first.
	mib := [4]int32{1, 14, 1, int32(pid)}
	var size uintptr
	if sysctl(&mib[0], 4, nil, &size, nil, 0) != 0 || size < 12 {
		return time.Time{}, false
	}
	raw := make([]byte, size)
	if sysctl(&mib[0], 4, unsafe.Pointer(&raw[0]), &size, nil, 0) != 0 || size < 12 {
		return time.Time{}, false
	}
	sec := int64(binary.NativeEndian.Uint64(raw[0:8]))
	usec := int64(int32(binary.NativeEndian.Uint32(raw[8:12])))
	if sec <= 0 || usec < 0 || usec >= 1_000_000 {
		return time.Time{}, false
	}
	return time.Unix(sec, usec*1000), true
}
