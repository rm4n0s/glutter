//go:build android

// Command android-viewport3d is a development example (step 30.9): the program of the Android device test of the 3D view. It
// opens the Vulkan device through the Android platform, draws the scenes of internal/scenetest through package gpu, and
// draws the picks of the same package, and saves what came of them in the app's files directory, for the host to read: files/viewport3d/<name>.png for each scene,
// errors.txt (one line "<name>: <message>" for each scene that failed), validation.txt (what the validation layer said),
// device.txt, and DONE when the whole run finished.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-viewport3d
package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/android"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"

	"image/png"
)

func main() {
	if err := run(); err != nil {
		slog.Error("android-viewport3d failed", "err", err)
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

// scenes draws the scenes at the first frame, when the window has a surface.
type scenes struct {
	platform.NopHandler
	win    platform.Window
	cancel context.CancelFunc
	done   bool
	err    error
}

// OnFrame is the first frame: the surface exists. It draws everything and ends the run.
func (h *scenes) OnFrame(ui.ViewID, time.Duration) {
	if h.done {
		return
	}
	h.done = true
	h.err = h.runAll()
	if h.err != nil {
		slog.Error("android-viewport3d: the run failed", "err", h.err)
	}
	h.cancel()
}

// cleanups runs what the program registers, as a test does when it ends.
type cleanups []func()

func (c *cleanups) Helper()           {}
func (c *cleanups) Cleanup(fn func()) { *c = append(*c, fn) }

// runAll opens the device and draws the scenes.
func (h *scenes) runAll() error {
	dir := filepath.Join(os.Getenv("HOME"), "viewport3d")
	if err := os.RemoveAll(dir); err != nil {
		return errors.NewErr(tagViewport3DWriteFailed, err).SetMetadata("dir", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.NewErr(tagViewport3DWriteFailed, err).SetMetadata("dir", dir)
	}
	dev, err := vulkan.Open(vulkan.Options{Surface: h.win.NativeSurface(), AllowCPU: true})
	if err != nil {
		return err
	}
	var tb cleanups
	app := gpuhost.NewBinding(&tb, dev)
	defer func() {
		for i := len(tb) - 1; i >= 0; i-- {
			tb[i]()
		}
		if err := dev.Close(); err != nil {
			slog.Error("android-viewport3d: closing the device failed", "err", err)
		}
	}()
	slog.Info("android-viewport3d: Vulkan device", "device", dev.Description())
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		return err
	}
	wait := func(done func() bool) {
		for i := 0; i < 5000 && !done(); i++ {
			app.RunPosted()
			if !done() {
				time.Sleep(time.Millisecond)
			}
		}
	}
	var failed []string
	for _, c := range scenetest.Cases() {
		img, err := scenetest.Render(d, wait, c)
		if err != nil {
			failed = append(failed, c.Name+": "+err.Error())
			continue
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			failed = append(failed, c.Name+": "+err.Error())
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, c.Name+".png"), buf.Bytes(), 0o644); err != nil {
			return errors.NewErr(tagViewport3DWriteFailed, err).SetMetadata("file", c.Name+".png")
		}
	}
	for _, c := range scenetest.PickChecks() {
		problem, err := scenetest.RunPickCheck(d, wait, c)
		switch {
		case err != nil:
			failed = append(failed, "pick "+c.Name+": "+err.Error())
		case problem != "":
			failed = append(failed, "pick "+c.Name+": "+problem)
		}
	}
	files := map[string]string{
		"errors.txt":     strings.Join(failed, "\n") + "\n",
		"validation.txt": strings.Join(dev.ValidationErrors(), "\n"),
		"device.txt":     "vulkan: " + dev.Description() + "\n",
		"DONE":           "done\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return errors.NewErr(tagViewport3DWriteFailed, err).SetMetadata("file", name)
		}
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagViewport3DWriteFailed = "Viewport3DWriteFailed"
