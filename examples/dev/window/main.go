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
//	GLUTTER_BACKEND=x11|wayland       the windowing system (else the session's)
//	GLUTTER_DEV_SHM=1                 draw through a wl_shm buffer instead of EGL (Wayland only)
//	GLUTTER_SCREENSHOT=<file.png>     save the window content to the file and exit
//	GLUTTER_SCREENSHOT_AFTER=<time>   wait that long (such as 500ms) before saving
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter dev window"
	cfg.AppID = "dev.glutter.window"
	cfg.Size = ui.Size{Width: 640, Height: 400}
	err := engine.Run(ctx, engine.Config{
		Window:  cfg,
		ShmFill: os.Getenv("GLUTTER_DEV_SHM") == "1",
	})
	if err != nil {
		report(err)
		os.Exit(1)
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
