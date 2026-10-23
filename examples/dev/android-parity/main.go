//go:build android

// Command android-parity is a development example (step 25.4): the parity program of the Android
// device tests. It draws every proof scene (internal/proof, the goldens of the software renderer)
// with the renderer that GLUTTER_RENDERER names, through the Android platform, and saves the images
// as PNG files in the app's files directory, for the host to compare with the software renderer's
// (internal/gpu/gputest). Nothing it draws is meant to be looked at.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/dev/android-parity
//
// Environment: GLUTTER_RENDERER is gles (the default for now). The images are in
// files/parity/<renderer>/<scene>.png, with <scene>.txt that says how many commands the renderer
// skipped, and files/parity/<renderer>/DONE when the whole run finished. The same program draws every
// scene with the software renderer too, into files/parity/soft: the host's fonts are not the
// device's, so the device's own software images are what the GPU images are compared with.
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
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/android"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("android-parity failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	renderer := os.Getenv("GLUTTER_RENDERER")
	if renderer == "" {
		renderer = "gles"
	}
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
	h := &parity{win: win, renderer: renderer, cancel: cancel}
	if err := p.Run(ctx, h); err != nil {
		return err
	}
	return h.err
}

// parity draws the scenes at the first frame, when the window has a surface.
type parity struct {
	platform.NopHandler
	win      platform.Window
	renderer string
	cancel   context.CancelFunc
	done     bool
	err      error
}

// OnFrame is the first frame: the surface exists. It draws everything and ends the run.
func (h *parity) OnFrame(ui.ViewID, time.Duration) {
	if h.done {
		return
	}
	h.done = true
	h.err = h.drawAll()
	if h.err != nil {
		slog.Error("android-parity: the run failed", "err", h.err)
	}
	h.cancel()
}

// drawAll attaches the renderer to the window and draws every scene.
func (h *parity) drawAll() error {
	dir := filepath.Join(os.Getenv("HOME"), "parity", h.renderer)
	softDir := filepath.Join(os.Getenv("HOME"), "parity", "soft")
	for _, d := range []string{dir, softDir} {
		if err := os.RemoveAll(d); err != nil {
			return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", d)
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", d)
		}
	}
	if h.renderer != "gles" {
		return errors.New(tagParityRendererUnknown, "GLUTTER_RENDERER is gles for now", "value", h.renderer)
	}
	r := gles.NewRenderer()
	defer func() { _ = r.Close() }()
	surface, err := r.Attach(h.win.NativeSurface())
	if err != nil {
		return err
	}
	defer surface.Destroy()
	slog.Info("android-parity: renderer ready", "renderer", h.renderer, "device", r.DeviceName())
	// One frame through the window itself first (drawing the scenes takes longer than the platform waits for the UI thread to let go of a lost surface): it shows that the surface works.
	if _, err := r.RenderCapture(surface, nil, ui.Size{Width: 64, Height: 64}, 1); err != nil {
		return err
	}
	for _, name := range proof.Names() {
		sc, err := proof.Lookup(name)
		if err != nil {
			return err
		}
		scene := sc.Build()
		size := ui.Size{Width: float64(sc.Width), Height: float64(sc.Height)}
		img, err := r.Snapshot(scene, size)
		if err != nil {
			return err
		}
		if err := goldens.WritePNG(filepath.Join(dir, name+".png"), img); err != nil {
			return err
		}
		ref, err := soft.Render(sc.Build(), sc.Width, sc.Height)
		if err != nil {
			return err
		}
		if err := goldens.WritePNG(filepath.Join(softDir, name+".png"), ref); err != nil {
			return err
		}
		skipped := r.LastFrame().Skipped
		note := fmt.Sprintf("skipped %d\n", skipped)
		if err := os.WriteFile(filepath.Join(dir, name+".txt"), []byte(note), 0o644); err != nil {
			return errors.NewErr(tagParityWriteFailed, err).SetMetadata("scene", name)
		}
		slog.Info("android-parity: scene drawn", "scene", name, "skipped", skipped)
	}
	if err := os.WriteFile(filepath.Join(dir, "DONE"), []byte(strings.Join(proof.Names(), "\n")+"\n"), 0o644); err != nil {
		return errors.NewErr(tagParityWriteFailed, err).SetMetadata("dir", dir)
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagParityRendererUnknown = "ParityRendererUnknown"
	tagParityWriteFailed     = "ParityWriteFailed"
)
