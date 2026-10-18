// Command canvas-tour opens a 640x480 window that shows the proof scenes of
// milestone 2 (shapes, gradients, blend modes, effects), one per page, drawn by
// the GPU renderer. It is a development example (milestone 3): it uses the
// internal packages directly, before the public API exists, and it grows with
// the renderer. What the GPU cannot draw yet is left out of the page and listed
// in the log, so each step of milestone 3 makes the pages fuller.
//
// Run it with: go run ./examples/dev/canvas-tour
//
// Keys: Right, PageDown or Space show the next page, Left or PageUp the previous
// one, Escape quits. To try it without a desktop session run it under a headless
// server: scripts/headless-wayland.sh -- go run ./examples/dev/canvas-tour
//
// Environment (read in debug builds only, as the examples promise):
//
//	GLUTTER_DEV_PAGE=<n or name>      the first page: 1 to 5, or a scene name
//	                                  such as shapes-clips (default 1)
//	GLUTTER_BACKEND=x11|wayland       the windowing system (else the session's)
//	GLUTTER_SCREENSHOT=<file.png>     save the window content to the file and exit
//	GLUTTER_SCREENSHOT_AFTER=<time>   wait that long (such as 500ms) before saving
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// The logical key ids of the keys the tour uses (Flutter's LogicalKeyboardKey).
const (
	keyEscape    = 0x0010000001b
	keySpace     = 0x20
	keyArrowLeft = 0x100000302
	keyArrowRite = 0x100000303
	keyPageDown  = 0x100000307
	keyPageUp    = 0x100000308
)

// order is the order of the pages: the proof scenes in the order of the steps that
// made them. Scenes registered later are added after these.
var order = []string{"shapes-basic", "shapes-clips", "gradients", "blend-modes", "effects"}

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

	t, err := newTour(os.Getenv)
	if err != nil {
		return err
	}
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter canvas tour"
	cfg.AppID = "dev.glutter.canvas-tour"
	cfg.Size = ui.Size{Width: 640, Height: 480}
	return engine.Run(ctx, engine.Config{
		Window:          cfg,
		Scene:           t.scene,
		SkipUnsupported: true,
		Hooks: engine.Hooks{
			Window: func(win platform.Window, stop func()) { t.win, t.stop = win, stop },
			Key:    t.onKey,
		},
	})
}

// tour is the pages and the one that is showing.
type tour struct {
	pages  []string
	page   int
	scenes map[string]*ui.Scene
	win    platform.Window
	stop   func()
}

// newTour makes the tour of the registered scenes, starting at the page the
// environment names. It fails with TourPageInvalid for a page that does not
// exist.
func newTour(getenv func(string) string) (*tour, error) {
	t := &tour{scenes: map[string]*ui.Scene{}}
	seen := map[string]bool{}
	for _, n := range order {
		if _, err := proof.Lookup(n); err != nil {
			return nil, err
		}
		t.pages = append(t.pages, n)
		seen[n] = true
	}
	for _, n := range proof.Names() {
		if !seen[n] {
			t.pages = append(t.pages, n)
		}
	}
	if v := getenv("GLUTTER_DEV_PAGE"); v != "" {
		page, err := t.find(v)
		if err != nil {
			return nil, err
		}
		t.page = page
	}
	return t, nil
}

// find returns the index of the page that a number (from 1) or a name means.
func (t *tour) find(v string) (int, error) {
	if n, err := strconv.Atoi(v); err == nil {
		if n < 1 || n > len(t.pages) {
			return 0, errors.New("TourPageInvalid", "GLUTTER_DEV_PAGE is a page number from 1, or a scene name",
				"value", v, "pages", len(t.pages))
		}
		return n - 1, nil
	}
	for i, p := range t.pages {
		if p == v {
			return i, nil
		}
	}
	return 0, errors.New("TourPageInvalid", "GLUTTER_DEV_PAGE is a page number from 1, or a scene name",
		"value", v, "pages", t.pages)
}

// show moves to another page, wrapping around, and asks for a frame.
func (t *tour) show(delta int) {
	t.page = (t.page + delta + len(t.pages)) % len(t.pages)
	slog.Info("canvas tour page", "page", t.page+1, "scene", t.pages[t.page])
	if t.win != nil {
		t.win.RequestFrame()
	}
}

// onKey changes the page for the navigation keys.
func (t *tour) onKey(d ui.KeyData) {
	if d.Type != ui.KeyEventTypeDown && d.Type != ui.KeyEventTypeRepeat {
		return
	}
	switch d.Logical {
	case keyArrowRite, keyPageDown, keySpace:
		t.show(1)
	case keyArrowLeft, keyPageUp:
		t.show(-1)
	case keyEscape:
		if t.stop != nil {
			t.stop()
		}
	}
}

// scene returns the scene of the page that is showing, built once.
func (t *tour) scene(ui.Size, float64) *ui.Scene {
	name := t.pages[t.page]
	s, ok := t.scenes[name]
	if !ok {
		sc, err := proof.Lookup(name)
		if err != nil {
			// The names were checked when the tour was made.
			panic(err)
		}
		s = sc.Build()
		t.scenes[name] = s
	}
	return s
}
