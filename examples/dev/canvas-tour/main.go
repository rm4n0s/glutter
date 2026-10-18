// Command canvas-tour opens a 640x480 window that shows the proof scenes of
// milestone 2 (shapes, gradients, blend modes, effects), one per page, drawn by
// the GPU renderer, and a last page, size-grid, that depends on the size of the
// window: a grid of lines one pixel wide, which shows whether the buffer is as
// sharp as the screen (on a display with a fractional scale it must be, step 3.7). It is a development example (milestone 3): it uses the
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
//	GLUTTER_DEV_PAGE=<n or name>      the first page: 1 to 6, or a scene name
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
// made them. Scenes registered later are added after these, and the size-grid page
// is the last of all.
var order = []string{"shapes-basic", "shapes-clips", "gradients", "blend-modes", "effects"}

// gridPage is the page that is not a proof scene: it is drawn for the size of the
// window.
const gridPage = "size-grid"

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
	// grid is the size-grid page for the size it was made for.
	grid     *ui.Scene
	gridSize ui.Size
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
	t.pages = append(t.pages, gridPage)
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

// scene returns the scene of the page that is showing: a proof scene, built once at
// its own size in pixels, or the grid for the size of the window.
func (t *tour) scene(size ui.Size, _ float64) *ui.Scene {
	name := t.pages[t.page]
	if name == gridPage {
		if t.grid == nil || t.gridSize != size {
			t.grid, t.gridSize = gridScene(size), size
		}
		return t.grid
	}
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

// gridScene draws, for a window of size physical pixels, a border and a grid of lines
// one pixel wide: a faint one every 16 pixels and a dark one every 80, with a mark at
// the center. The lines are on whole pixels, so they are as sharp as the buffer is: if
// the compositor has to scale the buffer, they become uneven and blurred.
func gridScene(size ui.Size) *ui.Scene {
	w, h := size.Width, size.Height
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, w, h))})
	c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
	line := func(argb uint32) *ui.Paint {
		p := ui.NewPaint()
		p.SetColor(ui.NewColor(argb))
		return p
	}
	faint, dark, accent := line(0xFFC9C5D4), line(0xFF4A4458), line(0xFF6750A4)
	for x := 16.0; x < w; x += 16 {
		p := faint
		if int(x)%80 == 0 {
			p = dark
		}
		c.DrawRect(ui.NewRectFromLTWH(x, 0, 1, h), p)
	}
	for y := 16.0; y < h; y += 16 {
		p := faint
		if int(y)%80 == 0 {
			p = dark
		}
		c.DrawRect(ui.NewRectFromLTWH(0, y, w, 1), p)
	}
	// The border is the edge of the buffer: all four sides must show.
	c.DrawRect(ui.NewRectFromLTWH(0, 0, w, 1), accent)
	c.DrawRect(ui.NewRectFromLTWH(0, h-1, w, 1), accent)
	c.DrawRect(ui.NewRectFromLTWH(0, 0, 1, h), accent)
	c.DrawRect(ui.NewRectFromLTWH(w-1, 0, 1, h), accent)
	cx, cy := float64(int(w/2)), float64(int(h/2))
	c.DrawRect(ui.NewRectFromLTWH(cx-12, cy, 25, 1), accent)
	c.DrawRect(ui.NewRectFromLTWH(cx, cy-12, 1, 25), accent)
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	return b.Build()
}
