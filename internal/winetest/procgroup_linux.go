//go:build linux

package winetest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/rm4n0s/errors"
)

// setProcessGroup puts the command in a process group of its own, so that the server, Wine and
// every process they start can be killed together (CLAUDE.md "Don't leave processes running").
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup kills the process group that setProcessGroup made, whose ID is the command's.
func killGroup(pid int) {
	// The group may be gone already: there is nothing to report if it is.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// lockFile takes an exclusive lock on the file (made if needed) and returns the function that
// releases it. It blocks until the lock is free. It fails with WinePrefixFailed.
func lockFile(path string) (unlock func(), err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, errors.NewErr(tagWinePrefixFailed, err).SetMetadata("lock", path)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close() // the lock was not taken; the failure is the error below
		return nil, errors.NewErr(tagWinePrefixFailed, err).SetMetadata("lock", path)
	}
	return func() {
		// Closing the file releases the lock; there is nothing to do if it fails.
		_ = f.Close()
	}, nil
}

// Leftovers lists the processes (as "pid command") that were started with the Wine prefix of the
// module at root (WINEPREFIX in their environment): Wine, its server, winedevice, the headless X
// server and what they started. A test checks that there are none after it stopped Wine. The
// prefix is root/.wine-test/prefix, or the value of GLUTTER_WINEPREFIX. It reads /proc.
//
// Flutter: no counterpart
func Leftovers(root string) []string {
	prefix := PrefixDir(root, os.Environ())
	return leftovers("/proc", []byte("WINEPREFIX="+prefix+"\x00"), os.Getpid())
}

// leftovers scans a /proc directory for processes whose environment has marker. self is skipped,
// and so are zombies, which are dead already (their parent has not collected them yet).
func leftovers(proc string, marker []byte, self int) []string {
	entries, err := os.ReadDir(proc)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}
		env, err := os.ReadFile(filepath.Join(proc, e.Name(), "environ"))
		if err != nil || !bytes.Contains(env, marker) {
			continue
		}
		if stat, err := os.ReadFile(filepath.Join(proc, e.Name(), "stat")); err == nil && bytes.Contains(stat, []byte(") Z ")) {
			continue
		}
		cmd, _ := os.ReadFile(filepath.Join(proc, e.Name(), "cmdline")) // gone already: an empty command
		out = append(out, e.Name()+" "+strings.ReplaceAll(strings.TrimRight(string(cmd), "\x00"), "\x00", " "))
	}
	return out
}
