//go:build linux

package androidtest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// setProcessGroup puts the command in a process group of its own, so that the emulator and
// every process it starts can be killed together (CLAUDE.md "Don't leave processes running").
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup kills the process group that setProcessGroup made, whose ID is the command's.
func killGroup(pid int) {
	// The group may be gone already: there is nothing to report if it is.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// Leftovers lists the processes (as "pid command") that were started with the environment of the
// harness for root: the emulator, qemu, the adb server and what they started. A device test
// checks that there are none after it stopped its emulator. Linux reads /proc.
//
// Flutter: no counterpart
func Leftovers(root string) []string {
	marker := []byte("ANDROID_AVD_HOME=" + filepath.Join(root, "avd"))
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		env, err := os.ReadFile("/proc/" + e.Name() + "/environ")
		if err != nil || !bytes.Contains(env, marker) {
			continue
		}
		if stat, err := os.ReadFile("/proc/" + e.Name() + "/stat"); err == nil && bytes.Contains(stat, []byte(") Z ")) {
			continue // a zombie is already dead; its parent has not collected it yet
		}
		cmd, _ := os.ReadFile("/proc/" + e.Name() + "/cmdline") // gone already: an empty command
		out = append(out, e.Name()+" "+strings.ReplaceAll(strings.TrimRight(string(cmd), "\x00"), "\x00", " "))
	}
	return out
}
