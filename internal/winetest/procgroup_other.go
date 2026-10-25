//go:build !linux

package winetest

import (
	"os"
	"os/exec"
)

// setProcessGroup does nothing: only Linux starts Wine.
func setProcessGroup(*exec.Cmd) {}

// killGroup kills the process, which is all this OS lets it reach.
func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill() // it may be gone already
	}
}

// lockFile takes no lock: only Linux makes the prefix.
func lockFile(string) (func(), error) { return func() {}, nil }

// Leftovers lists nothing: only Linux can look at the processes.
//
// Flutter: no counterpart
func Leftovers(string) []string { return nil }
