//go:build !windows

package main

import (
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// hostArch is the architecture that run and shot build for: Wine here is the 64-bit x86 one.
const hostArch = "amd64"

// newLauncher makes the launcher that runs the .exe on this machine: under Wine, through the
// harness of internal/winetest, which uses the scripts of the Glutter checkout. It fails with
// WindowsUsage when the command is not started in a checkout (no scripts/headless-x11.sh), and
// with ModuleRootNotFound when there is no go.mod above the working directory.
func newLauncher(headless bool) (wintool.Launcher, error) {
	h, err := winetest.New("")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(h.Root, "scripts", "headless-x11.sh")); err != nil {
		return nil, errors.NewErr(tagWindowsUsage, err).SetMetadata("root", h.Root,
			"hint", "run and shot under Wine use the scripts of the Glutter checkout: start the command in it")
	}
	return wintool.WineLauncher{Harness: h, Headless: headless}, nil
}
