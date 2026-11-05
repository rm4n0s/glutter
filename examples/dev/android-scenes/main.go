//go:build android

// Command android-scenes is a development example (step 30.1): the program of the Android device test
// of the explicit device. It opens the Vulkan device through the Android platform and runs the scenes
// that every backend runs (internal/gpu/explicit/devicetest), then saves what came of each in the app's
// files directory, for the host to read: files/scenes/results.txt (one line "ok <name>" or
// "FAIL <name>: <message>" for each part), validation.txt (what the validation layer said, one message
// per line), device.txt, and DONE when the whole run finished. Nothing it draws is meant to be looked at.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-scenes
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/android"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("android-scenes failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	p, err := android.New()
	if err != nil {
		return err
	}
	win, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &scenes{win: win, cancel: cancel}
	if err := p.Run(ctx, h); err != nil {
		return err
	}
	return h.err
}

// scenes runs the scenes at the first frame, when the window has a surface.
type scenes struct {
	platform.NopHandler
	win    platform.Window
	cancel context.CancelFunc
	done   bool
	err    error
}

// OnFrame is the first frame: the surface exists. It runs everything and ends the run.
func (h *scenes) OnFrame(ui.ViewID, time.Duration) {
	if h.done {
		return
	}
	h.done = true
	h.err = h.runAll()
	if h.err != nil {
		slog.Error("android-scenes: the run failed", "err", h.err)
	}
	h.cancel()
}

// runAll opens the device and runs the scenes.
func (h *scenes) runAll() error {
	dir := filepath.Join(os.Getenv("HOME"), "scenes")
	if err := os.RemoveAll(dir); err != nil {
		return errors.NewErr(tagScenesWriteFailed, err).SetMetadata("dir", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.NewErr(tagScenesWriteFailed, err).SetMetadata("dir", dir)
	}
	dev, err := vulkan.Open(vulkan.Options{Surface: h.win.NativeSurface(), AllowCPU: true})
	if err != nil {
		return err
	}
	defer func() {
		if err := dev.Close(); err != nil {
			slog.Error("android-scenes: closing the device failed", "err", err)
		}
	}()
	slog.Info("android-scenes: Vulkan device", "device", dev.Description())
	c, results := devicetest.NewCollector()
	devicetest.Run(c, &devicetest.Env{Dev: dev, Pixels: true})
	var lines []string
	for _, r := range results() {
		if len(r.Failures) == 0 {
			lines = append(lines, "ok "+r.Name)
			continue
		}
		lines = append(lines, fmt.Sprintf("FAIL %s: %s", r.Name, strings.Join(r.Failures, "; ")))
	}
	files := map[string]string{
		"results.txt":    strings.Join(lines, "\n") + "\n",
		"validation.txt": strings.Join(dev.ValidationErrors(), "\n"),
		"device.txt":     "vulkan: " + dev.Description() + "\n",
		"DONE":           "done\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return errors.NewErr(tagScenesWriteFailed, err).SetMetadata("file", name)
		}
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagScenesWriteFailed = "ScenesWriteFailed"
