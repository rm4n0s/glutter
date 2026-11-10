//go:build !unix

package iostest

import "os/exec"

// ownProcessGroup does nothing: the tests run on a Mac, and the package only builds elsewhere.
func ownProcessGroup(*exec.Cmd) {}

// killGroup kills the program.
func killGroup(cmd *exec.Cmd) { _ = cmd.Process.Kill() }
