// Command window opens a 640x400 Wayland window titled "Glutter dev window",
// filled with #6750A4 through a wl_shm buffer, and closes when the window is
// asked to close. It is the first development example (milestone 1): it uses
// the internal platform package directly, before the public API exists.
//
// Run it with: go run ./examples/dev/window
//
// It needs a Wayland session. To try it without one, run it under the
// headless compositor: scripts/headless-wayland.sh -- go run ./examples/dev/window
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/ui"
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

	p, err := wayland.New("")
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
	if _, err := p.NewWindow(cfg); err != nil {
		return err
	}
	// NopHandler agrees to every close request, and Run returns when the last
	// window is closed.
	return p.Run(ctx, platform.NopHandler{})
}

// report prints an error the way the generators do: as JSON that a test can
// replay, or as plain text for an error of another kind.
func report(err error) {
	if appErr, ok := errors.FromError(err); ok {
		if j, jerr := appErr.ToJson(); jerr == nil {
			fmt.Fprintf(os.Stderr, "%s: %s %v\n", j.Tag, j.Message, j.Metadata)
			return
		}
	}
	fmt.Fprintln(os.Stderr, err)
}
