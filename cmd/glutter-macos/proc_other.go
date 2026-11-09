//go:build !unix

package main

import "os/exec"

// ownProcessGroup does nothing: the tool runs apps for macOS, and only builds elsewhere.
func ownProcessGroup(*exec.Cmd) {}

// killGroup kills the program.
func killGroup(cmd *exec.Cmd) { _ = cmd.Process.Kill() }
