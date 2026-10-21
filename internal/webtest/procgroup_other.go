//go:build !linux

package webtest

import (
	"os"
	"os/exec"
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
