//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

// ownProcessGroup puts the program in a process group of its own, which killGroup ends with everything it
// started.
func ownProcessGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// killGroup kills the process group of the program. The group may be gone already; nothing is left to report.
func killGroup(cmd *exec.Cmd) { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
