// Command input opens a 640x400 window that logs every key and pointer event
// (with slog) and changes the mouse cursor: a text cursor over the left half of
// the window, a pointing hand over the right half. It is a development example
// (milestone 1): it uses the internal packages directly, before the public API
// exists.
//
// Run it with: go run ./examples/dev/input
//
// Things to try: type (with Shift, AltGr, and a dead key if your layout has
// one), hold a key to see the repeats, move the mouse across the middle of the
// window, press buttons, use the wheel, scroll or pinch on a touchpad, touch the
// screen. Press Escape or close the window to quit.
//
// Environment: GLUTTER_BACKEND=x11 or wayland picks the windowing system (else
// the session's), GLUTTER_DEV_SHM=1 draws through a wl_shm buffer instead of EGL
// (Wayland only), and XCURSOR_THEME and XCURSOR_SIZE choose the cursor theme
// when the compositor does not draw the cursors itself.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The physical key and logical key ids of Escape (Flutter's key ids).
const (
	physicalEscape = 0x00070029
	logicalEscape  = 0x0010000001b
)

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

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter input log"
	cfg.AppID = "dev.glutter.input"
	cfg.Size = ui.Size{Width: 640, Height: 400}

	a := &app{}
	return engine.Run(ctx, engine.Config{
		Window:  cfg,
		ShmFill: os.Getenv("GLUTTER_DEV_SHM") == "1",
		Hooks: engine.Hooks{
			Window:  func(win platform.Window, stop func()) { a.win, a.stop = win, stop },
			Metrics: a.onMetrics,
			Key:     a.onKey,
			Pointer: a.onPointer,
		},
	})
}

// app logs the input and picks the cursor.
type app struct {
	win           platform.Window
	stop          func()
	width, height int

	// half is the half of the window the pointer is in: -1 left, 1 right, 0 none.
	half int
}

func (a *app) onMetrics(view ui.ViewID, m platform.ViewMetrics) {
	a.width, a.height = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
	slog.Info("metrics", "view", view, "width", a.width, "height", a.height, "pixelRatio", m.DevicePixelRatio)
}

func (a *app) onKey(d ui.KeyData) {
	slog.Info("key", "type", d.Type.Label(), "physical", fmt.Sprintf("0x%x", d.Physical),
		"logical", fmt.Sprintf("0x%x", d.Logical), "character", fmt.Sprintf("%q", d.Character), "synthesized", d.Synthesized)
	if d.Type == ui.KeyEventTypeDown && d.Physical == physicalEscape && d.Logical == logicalEscape {
		a.stop()
	}
}

func (a *app) onPointer(batch []ui.PointerData) {
	for _, d := range batch {
		attrs := []any{
			"change", d.Change.String(), "kind", d.Kind.String(), "device", d.Device,
			"x", d.PhysicalX, "y", d.PhysicalY, "buttons", d.Buttons,
		}
		switch {
		case d.SignalKind == ui.PointerSignalKindScroll:
			attrs = append(attrs, "scrollX", d.ScrollDeltaX, "scrollY", d.ScrollDeltaY)
		case d.Change == ui.PointerChangePanZoomUpdate:
			attrs = append(attrs, "panX", d.PanX, "panY", d.PanY, "scale", d.Scale, "rotation", d.Rotation)
		case d.Kind == ui.PointerDeviceKindTouch:
			attrs = append(attrs, "id", d.PointerIdentifier)
		}
		slog.Info("pointer", attrs...)
		if d.Kind == ui.PointerDeviceKindMouse && d.SignalKind == ui.PointerSignalKindUnspecified {
			a.updateCursor(d)
		}
	}
}

// updateCursor shows the text cursor over the left half of the window and the
// pointing hand over the right half.
func (a *app) updateCursor(d ui.PointerData) {
	if d.Change == ui.PointerChangeRemove {
		a.half = 0
		return
	}
	half := 1
	if d.PhysicalX < float64(a.width)/2 {
		half = -1
	}
	if half == a.half {
		return
	}
	a.half = half
	if half < 0 {
		a.win.SetCursor(platform.MouseCursorText)
	} else {
		a.win.SetCursor(platform.MouseCursorClick)
	}
}
