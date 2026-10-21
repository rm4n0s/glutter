//go:build linux

package webtest

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the command in a process group of its own, so that the browser and
// every process it starts can be killed together (CLAUDE.md "Don't leave processes
// running").
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup kills the process group that setProcessGroup made, whose ID is the command's.
func killGroup(pid int) {
	// The group may be gone already: there is nothing to report if it is.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
