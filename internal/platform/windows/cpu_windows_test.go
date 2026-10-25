//go:build windows && glutter_wine

package windows

import (
	"syscall"
	"testing"
	"time"
)

// cpuTime is the CPU time of the process so far, kernel and user (GetProcessTimes).
func cpuTime(t *testing.T) time.Duration {
	t.Helper()
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatal(err)
	}
	var created, exited, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &created, &exited, &kernel, &user); err != nil {
		t.Fatal(err)
	}
	return time.Duration(kernel.Nanoseconds() + user.Nanoseconds())
}
