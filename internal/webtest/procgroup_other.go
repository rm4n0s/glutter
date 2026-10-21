//go:build !linux

package webtest

import (
	"os"
	"os/exec"
	"time"

	"github.com/rm4n0s/errors"
)

// setProcessGroup does nothing: only Linux has a browser launcher (the harness of SPEC §17
// runs on the Linux development machine).
func setProcessGroup(*exec.Cmd) {}

// killGroup kills the process, which is all this OS lets it reach.
func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill() // it may be gone already
	}
}

// privateDisplay does not exist here: only Linux needs one.
type privateDisplay struct{ name string }

// startPrivateDisplay starts nothing.
func startPrivateDisplay(time.Duration) (*privateDisplay, error) { return nil, nil }

func (*privateDisplay) stop() {}

// CPUTime is not available here: only Linux has a browser launcher. It fails with
// BrowserCPUUnavailable.
//
// Flutter: no counterpart
func (*Process) CPUTime() (time.Duration, error) {
	return 0, errors.New("BrowserCPUUnavailable", "the CPU time of the browser is read from /proc, which only Linux has")
}
