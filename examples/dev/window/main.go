// Command window opens a 640x400 window (Wayland or X11) titled "Glutter dev window" and
// fills it with #6750A4 using OpenGL ES, drawing a new frame each time the
// compositor asks for one. It is the first development example (milestone 1): it
// uses the internal packages directly, before the public API exists.
//
// Run it with: go run ./examples/dev/window
//
// It needs a Wayland or X11 session. To try it without one, run it under a
// headless server: scripts/headless-wayland.sh -- go run ./examples/dev/window
// or scripts/headless-x11.sh -- go run ./examples/dev/window
//
// Environment (read in debug builds only, as the examples promise):
//
//	GLUTTER_DEV_BACKEND=x11|wayland   the windowing system (else the session's)
//	GLUTTER_DEV_SHM=1                 draw through a wl_shm buffer instead of EGL
//	GLUTTER_SCREENSHOT=<file.png>     save the window content to the file and exit
//	GLUTTER_SCREENSHOT_AFTER=<time>   wait that long (such as 500ms) before saving
package main

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/examples/dev/devbackend"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// purple is #6750A4 as the components OpenGL wants.
const (
	purpleR = 0x67 / 255.0
	purpleG = 0x50 / 255.0
	purpleB = 0xA4 / 255.0
)

func main() {
	if err := run(); err != nil {
		report(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	useShm := os.Getenv("GLUTTER_DEV_SHM") == "1"
	shot, err := engine.ScreenshotFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	if useShm && shot != nil {
		return errors.New("ScreenshotUnsupported", "GLUTTER_SCREENSHOT reads the frame back through OpenGL, so it does not work with GLUTTER_DEV_SHM=1")
	}

	p, err := devbackend.New(useShm)
	if err != nil {
		return err
	}
	defer func() {
		if err := p.Close(); err != nil {
			report(err)
		}
	}()

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter dev window"
	cfg.AppID = "dev.glutter.window"
	cfg.Size = ui.Size{Width: 640, Height: 400}
	win, err := p.NewWindow(cfg)
	if err != nil {
		return err
	}

	if useShm {
		// NopHandler agrees to every close request, and Run returns when the last
		// window is closed.
		return p.Run(ctx, platform.NopHandler{})
	}

	a := &app{p: p, win: win, shot: shot, cancel: cancel}
	runErr := p.Run(ctx, a)
	a.release()
	if runErr != nil {
		return runErr
	}
	return a.err
}

// app is the handler: it draws the window with OpenGL ES.
type app struct {
	platform.NopHandler
	p      platform.Platform
	win    platform.Window
	shot   *engine.Screenshot
	cancel context.CancelFunc

	gl            *gles.Context
	surface       *gles.Surface
	width, height int

	armed      bool  // the capture is scheduled
	captureDue bool  // the next frame is read back
	err        error // the first failure, returned from run
}

func (a *app) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	a.width, a.height = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
}

func (a *app) OnFrame(ui.ViewID, time.Duration) {
	if a.gl == nil {
		ctx, surface, err := gles.Open(a.win.NativeSurface())
		if err != nil {
			a.fail(err)
			return
		}
		a.gl, a.surface = ctx, surface
		slog.Info("OpenGL ES context ready", "version", ctx.Version())
	}
	a.gl.Viewport(a.width, a.height)
	a.gl.Clear(purpleR, purpleG, purpleB, 1)

	var img *image.RGBA
	if a.captureDue {
		a.captureDue = false
		// Read before the swap: the back buffer is undefined after it.
		var err error
		if img, err = a.gl.ReadPixels(a.width, a.height); err != nil {
			a.fail(err)
			return
		}
	}
	if err := a.gl.SwapBuffers(a.surface); err != nil {
		a.fail(err)
		return
	}
	switch {
	case img != nil:
		if err := a.shot.Save(img); err != nil {
			a.fail(err)
			return
		}
		a.cancel() // done: Run returns and the program exits with code 0
	case a.shot != nil && !a.armed:
		// The first frame is presented. Capture the one that follows the delay.
		a.armed = true
		time.AfterFunc(a.shot.After(), func() {
			a.p.Post(func() {
				a.captureDue = true
				a.win.RequestFrame()
			})
		})
	}
}

// OnCloseRequest destroys the OpenGL surface, which draws into the window, and
// then lets the window close.
func (a *app) OnCloseRequest(ui.ViewID) bool {
	a.release()
	return true
}

// fail records the first error and ends the program.
func (a *app) fail(err error) {
	if a.err == nil {
		a.err = err
	}
	a.cancel()
}

// release destroys the OpenGL surface and context. It is safe to call twice.
func (a *app) release() {
	if a.surface != nil {
		a.surface.Destroy()
		a.surface = nil
	}
	if a.gl != nil {
		a.gl.Close()
		a.gl = nil
	}
}

// report prints an error the way the generators do: as its tag, message and
// metadata, or as plain text for an error of another kind.
func report(err error) {
	if appErr, ok := errors.FromError(err); ok {
		fmt.Fprintf(os.Stderr, "%s: %s %v\n", appErr.Tag, appErr.Message, appErr.Metadata)
		return
	}
	fmt.Fprintln(os.Stderr, err)
}
