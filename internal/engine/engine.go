package engine

import (
	"context"
	"image"
	"log/slog"
	"os"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/internal/platform/linux/x11"
	"github.com/rm4n0s/glutter/ui"
)

// Hooks let a development example watch the engine and take part in it, until
// the framework does. Every hook is optional, and is called on the UI thread.
//
// Flutter: no counterpart
type Hooks struct {
	// Window is called once, when the first window exists, with the window and a
	// function that ends Run.
	Window func(win platform.Window, stop func())
	// Key gets each key event.
	Key func(ui.KeyData)
	// Pointer gets each batch of pointer data.
	Pointer func([]ui.PointerData)
	// Metrics gets the size and scale of the window when they change.
	Metrics func(view ui.ViewID, metrics platform.ViewMetrics)
}

// Config tells Run what to do.
//
// Flutter: no counterpart
type Config struct {
	// Backend, when not empty, is "wayland" or "x11" and overrides the
	// GLUTTER_BACKEND environment variable.
	Backend string
	// Window is the first window. The zero value gets platform.DefaultWindowConfig
	// with the title "Glutter".
	Window platform.WindowConfig
	// Logger gets the engine's log; slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment (backend choice, self-capture); os.Getenv when
	// nil.
	Getenv func(string) string
	// ShmFill is a development aid: on Wayland the window draws a solid color
	// through wl_shm and never asks the engine for a frame. X11 has no such mode.
	ShmFill bool
	// Hooks watch the engine.
	Hooks Hooks

	// The fields below are for the tests of this package: a Wayland and an X11
	// that are not the real ones, and the renderer.
	wayland     func() (linux.Connection, error)
	x11         func() (linux.Connection, error)
	newRenderer func(log *slog.Logger) renderer
}

// developmentColor is what the skeleton draws, #6750A4 (the seed of Material 3's
// default color scheme), until the Scene arrives with milestone M2.
var developmentColor = [4]float32{0x67 / 255.0, 0x50 / 255.0, 0xA4 / 255.0, 1}

// renderer draws a frame into a window.
type renderer interface {
	// Draw draws one frame into the surface, whose size is width x height pixels,
	// and presents it. With capture it first reads the frame back, top row first,
	// and returns it: the back buffer is undefined after the swap.
	Draw(h gpu.SurfaceHandle, width, height int, capture bool) (*image.RGBA, error)
	// Close destroys the renderer's surface and context. The window must still
	// exist.
	Close()
}

// Run connects to the display (Wayland or X11, SPEC §6.3 rule 5), opens the first
// window and draws it, and runs the event loop on the calling goroutine, which
// must be the UI thread. It returns nil when ctx is cancelled or the last window
// is closed. When GLUTTER_SCREENSHOT is set it saves a PNG of the window and
// returns (see Screenshot).
//
// Errors come back unchanged: NoDisplay, BackendUnknown, the backend's own
// (WaylandConnectFailed, X11OpenDisplayFailed, LibraryLoadFailed, ...), the
// renderer's (EGLLoadFailed, EGLNoConfig, GLES3Unsupported, ...) and the
// screenshot's.
//
// TODO(step 5.2): the framework's run loop replaces the clear color: the frame
// builds, lays out, paints and renders a Scene.
//
// Flutter: engine/src/flutter/shell (the shell that runs an app)
func Run(ctx context.Context, cfg Config) (err error) {
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	shot, err := ScreenshotFromEnv(getenv)
	if err != nil {
		return err
	}
	if cfg.ShmFill && shot != nil {
		return errors.New("ScreenshotUnsupported",
			"GLUTTER_SCREENSHOT reads the frame back through OpenGL, so it does not work with the shared-memory mode")
	}

	conn, _, err := linux.Select(linux.Selection{
		Backend: cfg.Backend,
		Getenv:  getenv,
		Wayland: cfg.waylandConnector(),
		X11:     cfg.x11Connector(),
		Logger:  log,
	})
	if err != nil {
		return err
	}
	defer func() {
		// A failure to close is worth reporting only when nothing failed before it.
		if cerr := conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	win, err := conn.NewWindow(cfg.windowConfig())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := &handler{cfg: &cfg, log: log, conn: conn, win: win, shot: shot, cancel: cancel}
	if cfg.newRenderer != nil {
		h.newRenderer = cfg.newRenderer
	} else {
		h.newRenderer = newGLESRenderer
	}
	if cfg.Hooks.Window != nil {
		cfg.Hooks.Window(win, cancel)
	}
	runErr := conn.Run(ctx, h)
	h.release()
	if runErr != nil {
		return runErr
	}
	return h.err
}

// windowConfig returns the config of the first window.
func (c *Config) windowConfig() platform.WindowConfig {
	if c.Window == (platform.WindowConfig{}) {
		w := platform.DefaultWindowConfig()
		w.Title = "Glutter"
		return w
	}
	return c.Window
}

// waylandConnector returns the function that connects the Wayland backend.
func (c *Config) waylandConnector() func() (linux.Connection, error) {
	if c.wayland != nil {
		return c.wayland
	}
	return func() (linux.Connection, error) {
		p, err := wayland.New("", wayland.Options{ShmFill: c.ShmFill})
		if err != nil {
			return nil, err
		}
		return p, nil
	}
}

// x11Connector returns the function that connects the X11 backend.
func (c *Config) x11Connector() func() (linux.Connection, error) {
	if c.x11 != nil {
		return c.x11
	}
	return func() (linux.Connection, error) {
		if c.ShmFill {
			return nil, errors.New("ShmFillUnsupported", "the shared-memory mode is for Wayland only")
		}
		p, err := x11.New("")
		if err != nil {
			return nil, err
		}
		return p, nil
	}
}

// handler is the platform.Handler of the engine: it draws the window, captures
// the self-capture screenshot, and passes input to the hooks.
type handler struct {
	platform.NopHandler
	cfg    *Config
	log    *slog.Logger
	conn   linux.Connection
	win    platform.Window
	shot   *Screenshot
	cancel context.CancelFunc

	newRenderer   func(log *slog.Logger) renderer
	r             renderer
	width, height int

	armed      bool // the capture is scheduled
	captureDue bool // the next frame is read back
	err        error
}

var _ platform.Handler = (*handler)(nil)

func (h *handler) OnMetrics(view ui.ViewID, m platform.ViewMetrics) {
	h.width, h.height = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
	if f := h.cfg.Hooks.Metrics; f != nil {
		f(view, m)
	}
}

func (h *handler) OnKey(d ui.KeyData) {
	if f := h.cfg.Hooks.Key; f != nil {
		f(d)
	}
}

func (h *handler) OnPointer(batch []ui.PointerData) {
	if f := h.cfg.Hooks.Pointer; f != nil {
		f(batch)
	}
}

// OnFrame draws the window. The first frame also creates the renderer: its
// surface needs the window to be mapped.
func (h *handler) OnFrame(ui.ViewID, time.Duration) {
	if h.r == nil {
		h.r = h.newRenderer(h.log)
	}
	capture := h.captureDue
	h.captureDue = false
	img, err := h.r.Draw(h.win.NativeSurface(), h.width, h.height, capture)
	if err != nil {
		h.fail(err)
		return
	}
	switch {
	case img != nil:
		if err := h.shot.Save(img); err != nil {
			h.fail(err)
			return
		}
		h.cancel() // done: Run returns, and the program exits with code 0
	case h.shot != nil && !h.armed:
		// The first frame is presented. Capture the one that follows the delay.
		h.armed = true
		if h.shot.After() == 0 {
			h.captureDue = true
			h.win.RequestFrame()
			return
		}
		time.AfterFunc(h.shot.After(), func() {
			h.conn.Post(func() {
				h.captureDue = true
				h.win.RequestFrame()
			})
		})
	}
}

// OnCloseRequest destroys the renderer, whose surface draws into the window, and
// then lets the window close.
func (h *handler) OnCloseRequest(ui.ViewID) bool {
	h.release()
	return true
}

// fail records the first error and ends Run.
func (h *handler) fail(err error) {
	if h.err == nil {
		h.err = err
	}
	h.cancel()
}

// release destroys the renderer. It is safe to call twice.
func (h *handler) release() {
	if h.r != nil {
		h.r.Close()
		h.r = nil
	}
}

// glesRenderer is the OpenGL ES renderer of the skeleton: it clears the window
// to developmentColor.
type glesRenderer struct {
	log     *slog.Logger
	ctx     *gles.Context
	surface *gles.Surface
}

func newGLESRenderer(log *slog.Logger) renderer { return &glesRenderer{log: log} }

func (r *glesRenderer) Draw(h gpu.SurfaceHandle, width, height int, capture bool) (*image.RGBA, error) {
	if r.ctx == nil {
		ctx, surface, err := gles.Open(h)
		if err != nil {
			return nil, err
		}
		r.ctx, r.surface = ctx, surface
		r.log.Info("OpenGL ES context ready", "version", ctx.Version())
	}
	r.ctx.Viewport(width, height)
	c := developmentColor
	r.ctx.Clear(c[0], c[1], c[2], c[3])
	var img *image.RGBA
	if capture {
		var err error
		if img, err = r.ctx.ReadPixels(width, height); err != nil {
			return nil, err
		}
	}
	if err := r.ctx.SwapBuffers(r.surface); err != nil {
		return nil, err
	}
	return img, nil
}

func (r *glesRenderer) Close() {
	if r.surface != nil {
		r.surface.Destroy()
		r.surface = nil
	}
	if r.ctx != nil {
		r.ctx.Close()
		r.ctx = nil
	}
}
