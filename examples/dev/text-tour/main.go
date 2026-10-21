// Command text-tour opens a 640x480 window that shows the text proof (the text-tour scene
// of internal/proof) drawn by the GPU renderer: Latin text in several sizes, weights and
// styles, Arabic and Hebrew, CJK, color emoji, every decoration style, shadows, an
// ellipsized paragraph and a justified one. It is a development example (milestone 4): it
// uses the internal packages directly, before the public API exists.
//
// Run it with: go run ./examples/dev/text-tour
//
// Escape quits. To try it without a desktop session run it under a headless server:
// scripts/headless-wayland.sh -- go run ./examples/dev/text-tour
//
// Environment (read in debug builds only, as the examples promise):
//
//	GLUTTER_BACKEND=x11|wayland       the windowing system (else the session's)
//	GLUTTER_RENDERER=gles|vulkan      the renderer (else the default; see SPEC §9.6)
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
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// keyEscape is the logical key id of Escape (Flutter's LogicalKeyboardKey.escape).
const keyEscape = 0x0010000001b

func main() {
	if err := run(); err != nil {
		if appErr, ok := errors.FromError(err); ok {
			fmt.Fprintf(os.Stderr, "%s: %s %v\n", appErr.Tag, appErr.Message, appErr.Metadata)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scene, err := proof.Lookup("text-tour")
	if err != nil {
		return err
	}
	built := scene.Build()
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter text tour"
	cfg.AppID = "dev.glutter.text-tour"
	cfg.Size = ui.Size{Width: 640, Height: 480}
	var quit func()
	return engine.Run(ctx, engine.Config{
		Window: cfg,
		Scene:  func(ui.Size, float64) *ui.Scene { return built },
		// What a renderer cannot draw yet (the Vulkan renderer's gradients and blurs, until
		// steps 20.5 and 20.6) is left out and listed in the log.
		SkipUnsupported: true,
		Hooks: engine.Hooks{
			Window: func(_ platform.Window, stop func()) { quit = stop },
			Key: func(d ui.KeyData) {
				if d.Type == ui.KeyEventTypeDown && d.Logical == keyEscape && quit != nil {
					quit()
				}
			},
		},
	})
}
