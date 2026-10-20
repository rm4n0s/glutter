package engine

import (
	"context"
	"image"
	"log/slog"
	"os"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
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

// SceneFunc makes the scene of a frame. It gets the size of the window in
// physical pixels and its device pixel ratio, and is called on the UI thread for
// each frame the window draws.
//
// Flutter: no counterpart (the framework's PlatformDispatcher.onDrawFrame)
type SceneFunc func(size ui.Size, dpr float64) *ui.Scene

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
	// Scene, when set, makes the scene each frame draws, with the GPU renderer.
	// Without it the window is filled with a development color.
	Scene SceneFunc
	// SkipUnsupported makes the renderer leave out what it cannot draw yet instead
	// of ending Run with GLNotImplemented; the log says what was left out.
	SkipUnsupported bool
	// Start, when set, is called once on the UI thread, after the first window
	// exists and the dispatcher knows its view, the settings and the displays,
	// and before the event loop runs. The framework attaches itself to the
	// dispatcher here (its callbacks, then ScheduleFrame). An error ends Run
	// with that error. Start replaces Scene: a Scene is drawn only when Start is
	// nil.
	Start func(d *ui.PlatformDispatcher) error

	// The fields below are for the tests of this package: a Wayland and an X11
	// that are not the real ones, and the renderer.
	idlePoll    time.Duration // how often the capture asks whether the app is idle
	idleMax     time.Duration // how long it waits for the app to become idle
	wayland     func() (linux.Connection, error)
	x11         func() (linux.Connection, error)
	newRenderer func(cfg *Config, log *slog.Logger) renderer
}

// developmentColor is what the window is cleared to before the scene is drawn,
// #6750A4 (the seed of Material 3's default color scheme), until the framework's
// Scene arrives with milestone M5.
const developmentColor = 0xFF6750A4

// renderer draws a frame into a window.
type renderer interface {
	// Draw draws one frame into the surface, whose size is width x height pixels,
	// and presents it. The scene is what to draw over the background; nil is an
	// empty one. With capture it first reads the frame back, top row first, and
	// returns it: the back buffer is undefined after the swap.
	Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error)
	// Release destroys the surface of the window with the handle. The window must
	// still exist. A handle that was never drawn into is ignored.
	Release(h gpu.SurfaceHandle)
	// Close destroys the renderer's surfaces and context. The windows must still
	// exist.
	Close()
}

// DecorationsEnv is the environment variable that chooses how the title bar of
// the first window is made: "auto" (the default), "app", or "libdecor" (libdecor
// even where the compositor has title bars of its own, to try it).
//
// Flutter: no counterpart
const DecorationsEnv = "GLUTTER_DECORATIONS"

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
// The engine owns a ui.PlatformDispatcher: it reports the views, the input, the
// settings, the lifecycle and the frames to it, and draws the scenes that the
// framework renders into a view. A frame the framework did not render into draws
// the previous scene again, because the platform needs one presented frame for
// each frame it asks for (platform.Window.RequestFrame).
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

	wcfg := cfg.windowConfig()
	if wcfg.Decorations == platform.DecorationsUnspecified {
		// GLUTTER_DECORATIONS=app makes the app draw the title bar of its first
		// window, as WindowConfig.Decorations does; "auto" is the default.
		switch v := getenv(DecorationsEnv); v {
		case "app":
			wcfg.Decorations = platform.DecorationsApp
		case "libdecor":
			wcfg.Decorations = platform.DecorationsLibdecor
		case "", "auto":
		default:
			return errors.New("DecorationsUnknown", "GLUTTER_DECORATIONS is not auto, app or libdecor", "value", v)
		}
	}
	win, err := conn.NewWindow(wcfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := &handler{cfg: &cfg, log: log, conn: conn, shot: shot, cancel: cancel, windows: map[ui.ViewID]*window{},
		appID: wcfg.AppID, decorations: wcfg.Decorations}
	h.first = h.addWindow(win)
	if cfg.newRenderer != nil {
		h.newRenderer = cfg.newRenderer
	} else {
		h.newRenderer = newGLESRenderer
	}
	h.disp = ui.NewPlatformDispatcher(h)
	h.disp.SetDisplays(h.displays())
	h.disp.UpdatePlatformConfiguration(platformConfiguration(conn.Settings()))
	h.disp.AddView(win.ID(), ui.ViewConfiguration{})
	if cfg.Start != nil {
		if err := cfg.Start(h.disp); err != nil {
			return err
		}
	} else if cfg.Scene != nil {
		h.disp.OnDrawFrame = h.drawSceneFunc
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

// window is one native window of the engine and what the engine keeps for it.
type window struct {
	platform.Window
	id            ui.ViewID
	width, height int
	// lastScene is the scene the framework rendered into the view last.
	lastScene *ui.Scene
	// rendered is whether the framework rendered a scene into the view in the
	// current frame.
	rendered bool
	// wanted is whether a frame of the window was asked for and has not come yet.
	wanted bool
	// pending is whether lastScene was rendered during the frame of another window:
	// the frame of this window then only presents it.
	pending bool
}

// handler is the platform.Handler of the engine: it draws the windows, captures
// the self-capture screenshot, and passes input to the hooks.
type handler struct {
	platform.NopHandler
	cfg    *Config
	log    *slog.Logger
	conn   linux.Connection
	shot   *Screenshot
	cancel context.CancelFunc

	disp        *ui.PlatformDispatcher
	newRenderer func(cfg *Config, log *slog.Logger) renderer
	// r draws every window: one renderer, one GL context, a surface for each window.
	r renderer
	// windows are the windows that exist, by view. first is the first one, the one
	// the self-capture reads.
	windows map[ui.ViewID]*window
	first   *window
	// appID and decorations are what the first window was made with; the other
	// windows get them too.
	appID       string
	decorations platform.Decorations
	// frameView is the view whose frame is being run, and hasFrame is whether one is.
	frameView ui.ViewID
	hasFrame  bool
	// pointerView is the view the pointer was last over, and focusView the view that
	// has the keyboard focus; the cursor and the input method follow them.
	pointerView, focusView ui.ViewID
	// input is what arrived since the last frame, which the next one handles first.
	input inputQueue

	armed      bool // the capture is scheduled
	captureDue bool // the next frame is read back
	err        error
}

var (
	_ platform.Handler     = (*handler)(nil)
	_ ui.PlatformHost      = (*handler)(nil)
	_ ui.CursorHost        = (*handler)(nil)
	_ ui.WindowHost        = (*handler)(nil)
	_ ui.WindowControlHost = (*handler)(nil)
	_ ui.ViewFocusHost     = (*handler)(nil)
	_ ui.ClipboardHost     = (*handler)(nil)
	_ ui.RichClipboardHost = (*handler)(nil)
	_ ui.TextInputHost     = (*handler)(nil)
)

// addWindow keeps the window of the platform and returns it.
func (h *handler) addWindow(win platform.Window) *window {
	w := &window{Window: win, id: win.ID()}
	if h.windows == nil {
		h.windows = map[ui.ViewID]*window{}
	}
	h.windows[w.id] = w
	return w
}

// windowOf is the window of the view; when the view is not known (it was closed),
// the first window that is left, or nil when there is none.
func (h *handler) windowOf(view ui.ViewID) *window {
	if w, ok := h.windows[view]; ok {
		return w
	}
	var found *window
	for _, w := range h.windows {
		if found == nil || w.id < found.id {
			found = w
		}
	}
	return found
}

// PerformWindowAction carries out the action of a title bar that the app draws on
// the window.
func (h *handler) PerformWindowAction(view ui.ViewID, action ui.WindowAction) {
	if w, ok := h.windows[view]; ok {
		w.PerformWindowAction(action)
	}
}

// CreateWindow makes a window for the framework (ui.WindowControlHost): the
// platform makes it, and the view starts with the requested size until the
// platform says what it got.
func (h *handler) CreateWindow(spec ui.WindowSpec) (ui.ViewID, ui.ViewConfiguration, error) {
	cfg := platform.WindowConfig{
		Title: spec.Title, AppID: h.appID, Size: spec.Size, MinSize: spec.MinSize, MaxSize: spec.MaxSize,
		Decorations: h.decorations,
	}
	if spec.NotResizable {
		cfg.Resizable = foundation.Some(false)
	}
	if spec.AppDecorations {
		cfg.Decorations = platform.DecorationsApp
	}
	if spec.Kind == ui.WindowKindDialog {
		cfg.Kind = platform.WindowKindDialog
	}
	if spec.HasParent {
		cfg.Parent, cfg.HasParent = spec.Parent, true
		// A dialog with a parent is modal to it, as in Flutter.
		cfg.Modal = spec.Kind == ui.WindowKindDialog
	}
	win, err := h.conn.NewWindow(cfg)
	if err != nil {
		return 0, ui.ViewConfiguration{}, err
	}
	w := h.addWindow(win)
	return w.id, ui.ViewConfiguration{PhysicalSize: spec.Size, DevicePixelRatio: 1}, nil
}

// DestroyWindow closes the window of the view (ui.WindowControlHost): the surface
// of the renderer goes first.
func (h *handler) DestroyWindow(view ui.ViewID) {
	w, ok := h.windows[view]
	if !ok {
		return
	}
	h.forget(w)
	w.Close()
}

// forget destroys the surface of the window and drops it. The window must still
// exist.
func (h *handler) forget(w *window) {
	if h.r != nil {
		h.r.Release(w.NativeSurface())
	}
	delete(h.windows, w.id)
}

// SetWindowTitle sets the title of the window (ui.WindowControlHost).
func (h *handler) SetWindowTitle(view ui.ViewID, title string) {
	if w, ok := h.windows[view]; ok {
		w.SetTitle(title)
	}
}

// SetWindowSize asks for a new size of the window (ui.WindowControlHost).
func (h *handler) SetWindowSize(view ui.ViewID, size ui.Size) {
	if w, ok := h.windows[view]; ok {
		w.SetSize(size)
	}
}

// SetWindowConstraints sets the limits on the size of the window
// (ui.WindowControlHost).
func (h *handler) SetWindowConstraints(view ui.ViewID, min, max ui.Size) {
	if w, ok := h.windows[view]; ok {
		w.SetSizeConstraints(min, max)
	}
}

// RequestViewFocus asks the platform to give the focus to the window of the view
// (ui.ViewFocusHost). Taking the focus away is up to the user.
func (h *handler) RequestViewFocus(view ui.ViewID, state ui.ViewFocusState, _ ui.ViewFocusDirection) {
	if w, ok := h.windows[view]; ok && state == ui.ViewFocusStateFocused && h.focusView != view {
		w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate})
	}
}

// OnViewFocus tells the dispatcher that a view got or lost the keyboard focus,
// with the input that came before it.
func (h *handler) OnViewFocus(view ui.ViewID, focused bool) {
	w, ok := h.windows[view]
	if !ok {
		return
	}
	state := ui.ViewFocusStateUnfocused
	if focused {
		state = ui.ViewFocusStateFocused
		h.focusView = view
	}
	h.queueInput(w, func() {
		h.disp.DispatchViewFocus(ui.ViewFocusEvent{ViewID: view, State: state, Direction: ui.ViewFocusDirectionUndefined})
	})
}

func (h *handler) OnMetrics(view ui.ViewID, m platform.ViewMetrics) {
	w, ok := h.windows[view]
	if !ok {
		return
	}
	w.width, w.height = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
	h.disp.UpdateViewConfiguration(view, h.viewConfiguration(m))
	if f := h.cfg.Hooks.Metrics; f != nil {
		f(view, m)
	}
}

// viewConfiguration turns the metrics of a window into the configuration of its
// view: the padding is the part of the view padding that the insets do not
// already cover, and the display is the one the platform names, else the
// primary one.
func (h *handler) viewConfiguration(m platform.ViewMetrics) ui.ViewConfiguration {
	cfg := ui.ViewConfiguration{
		PhysicalSize:        m.PhysicalSize,
		DevicePixelRatio:    m.DevicePixelRatio,
		ViewPadding:         m.ViewPadding,
		ViewInsets:          m.ViewInsets,
		SystemGestureInsets: m.SystemGestureInsets,
		Window:              m.Window,
		Padding: ui.ViewPadding{
			Left:   max(0, m.ViewPadding.Left-m.ViewInsets.Left),
			Top:    max(0, m.ViewPadding.Top-m.ViewInsets.Top),
			Right:  max(0, m.ViewPadding.Right-m.ViewInsets.Right),
			Bottom: max(0, m.ViewPadding.Bottom-m.ViewInsets.Bottom),
		},
	}
	displays := h.displays()
	cfg.Display = ui.Display{DevicePixelRatio: m.DevicePixelRatio, Size: m.PhysicalSize, RefreshRate: defaultRefreshRate}
	if id, ok := m.Display.Get(); ok {
		for _, d := range displays {
			if d.ID == id {
				cfg.Display = d
				return cfg
			}
		}
	}
	for i, d := range h.conn.Displays() {
		if d.Primary && i < len(displays) {
			cfg.Display = displays[i]
			break
		}
	}
	return cfg
}

// defaultRefreshRate is the refresh rate of a view whose display the platform
// does not report, in frames per second.
const defaultRefreshRate = 60

// displays lists the connected displays for the dispatcher.
func (h *handler) displays() []ui.Display {
	var out []ui.Display
	for _, d := range h.conn.Displays() {
		dpr := d.DevicePixelRatio
		if dpr == 0 {
			dpr = 1
		}
		out = append(out, ui.Display{
			ID:               d.ID,
			DevicePixelRatio: dpr,
			Size:             ui.Size{Width: d.Bounds.Width() * dpr, Height: d.Bounds.Height() * dpr},
			RefreshRate:      d.RefreshRate,
		})
	}
	return out
}

// platformConfiguration turns the platform's settings into the dispatcher's.
func platformConfiguration(s platform.Settings) ui.PlatformConfiguration {
	c := ui.PlatformConfiguration{
		Locales:            s.Locales,
		TextScaleFactor:    s.TextScaleFactor,
		PlatformBrightness: s.PlatformBrightness,
		AccessibilityFeatures: ui.AccessibilityFeatures{
			DisableAnimations: s.DisableAnimations,
			ReduceMotion:      s.DisableAnimations,
			HighContrast:      s.HighContrast,
		},
	}
	if s.HasAccentColor {
		c.AccentColor = ui.Ptr(s.AccentColor)
	}
	return c
}

func (h *handler) OnSettings(s platform.Settings) {
	h.disp.UpdatePlatformConfiguration(platformConfiguration(s))
}

func (h *handler) OnLifecycle(state ui.AppLifecycleState) { h.disp.SetAppLifecycleState(state) }

func (h *handler) OnKey(d ui.KeyData) {
	h.queueInput(h.windowOf(h.focusView), func() {
		if f := h.cfg.Hooks.Key; f != nil {
			f(d)
		}
		h.disp.DispatchKeyData(d)
	})
}

func (h *handler) OnPointer(batch []ui.PointerData) {
	batch = append([]ui.PointerData(nil), batch...)
	if len(batch) > 0 {
		h.pointerView = batch[len(batch)-1].ViewID
	}
	h.queueInput(h.windowOf(h.pointerView), func() {
		if f := h.cfg.Hooks.Pointer; f != nil {
			f(batch)
		}
		h.disp.DispatchPointerDataPacket(ui.PointerDataPacket{Data: batch})
	})
}

// queueInput has the next frame of the window handle an input event. A window that
// never draws (the shared-memory mode) has no frame to wait for, so the event is
// handled at once.
func (h *handler) queueInput(w *window, handle func()) {
	if h.cfg.ShmFill {
		handle()
		return
	}
	h.input.add(handle)
	h.askFrame(w)
}

// askFrame asks the platform for a frame of the window.
func (h *handler) askFrame(w *window) {
	if w == nil {
		return
	}
	w.wanted = true
	w.RequestFrame()
}

// OnFrame runs a frame of a window: the input that arrived since the last one, then
// the dispatcher's OnBeginFrame and OnDrawFrame. The framework renders its scenes
// in OnDrawFrame, one for each view; the window whose frame this is draws its own
// at once, and the other windows present theirs in their own frames. If the
// framework rendered nothing into the window, the previous scene is drawn again, so
// that exactly one frame is presented. The first frame also creates the renderer:
// its surface needs the window to be mapped.
func (h *handler) OnFrame(view ui.ViewID, frameTime time.Duration) {
	w, ok := h.windows[view]
	if !ok {
		return
	}
	w.wanted = false
	if w.pending && len(h.input.pending) == 0 {
		// The framework rendered this window during the frame of another one, and
		// nothing arrived since that needs a frame of the framework.
		w.pending = false
		if h.err == nil {
			_ = h.draw(w, w.lastScene) // a failure is recorded by draw and ends Run
		}
		return
	}
	w.pending = false
	h.input.flush()
	for _, o := range h.windows {
		o.rendered = false
	}
	h.frameView, h.hasFrame = view, true
	h.disp.BeginFrame(frameTime)
	h.disp.DrawFrame()
	h.hasFrame = false
	if !w.rendered && h.err == nil {
		_ = h.draw(w, w.lastScene)
	}
}

// drawSceneFunc is OnDrawFrame for a Config that has a Scene: it renders the
// scene that Config.Scene makes into the first window.
func (h *handler) drawSceneFunc() {
	v := h.disp.View(h.first.id)
	if v == nil {
		return
	}
	scene := h.cfg.Scene(v.PhysicalSize(), v.DevicePixelRatio())
	if err := v.Render(scene); err != nil {
		h.fail(err)
	}
}

// ScheduleFrame asks every window for a frame (ui.PlatformHost): the framework
// does not say which of its views changed.
func (h *handler) ScheduleFrame() {
	if h.cfg.ShmFill {
		return // the shared-memory mode has no frames
	}
	for _, w := range h.windows {
		h.askFrame(w)
	}
}

// RequestAppExit ends Run (ui.PlatformHost).
func (h *handler) RequestAppExit() { h.cancel() }

// ActivateSystemCursor shows the cursor of the kind over the window the pointer is
// in (ui.CursorHost).
func (h *handler) ActivateSystemCursor(_ int, kind string) {
	cursor, ok := platform.MouseCursorFromKind(kind)
	if !ok {
		h.log.Warn("glutter: unknown system mouse cursor", "kind", kind)
		return
	}
	if w := h.windowOf(h.pointerView); w != nil {
		w.SetCursor(cursor)
	}
}

// SetTextInput gives the input method of the window that has the keyboard focus the
// state of the focused text field (ui.TextInputHost).
func (h *handler) SetTextInput(state ui.TextInputState) {
	if w := h.windowOf(h.focusView); w != nil {
		w.SetTextInput(state)
	}
}

// OnTextInput passes a change that the input method asks for to the dispatcher,
// at the start of the next frame like the other input.
func (h *handler) OnTextInput(event ui.TextInputEvent) {
	h.queueInput(h.windowOf(h.focusView), func() { h.disp.DispatchTextInput(event) })
}

// SetClipboardText puts text on the system clipboard (ui.ClipboardHost).
func (h *handler) SetClipboardText(ctx context.Context, text string, done func(error)) {
	h.conn.Clipboard().SetText(ctx, text, done)
}

// GetClipboardText reads the text of the system clipboard (ui.ClipboardHost).
func (h *handler) GetClipboardText(ctx context.Context, done func(string, bool, error)) {
	h.conn.Clipboard().GetText(ctx, done)
}

// SetClipboard puts the content on the system clipboard or the primary selection
// (ui.RichClipboardHost).
func (h *handler) SetClipboard(ctx context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	h.conn.Clipboard().SetContent(ctx, kind, content, done)
}

// GetClipboard reads the system clipboard or the primary selection in the format
// (ui.RichClipboardHost).
func (h *handler) GetClipboard(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	h.conn.Clipboard().GetContent(ctx, kind, format, done)
}

// Post runs fn on the UI thread (ui.PlatformHost).
func (h *handler) Post(fn func()) { h.conn.Post(fn) }

// RenderView draws the scene the framework rendered into the view
// (ui.PlatformHost). The window whose frame is running draws it now; another window
// presents it in its own frame, which it is asked for if none was. A window that
// had no frame coming is not drawn again: it did not change.
func (h *handler) RenderView(view ui.ViewID, scene *ui.Scene) error {
	w, ok := h.windows[view]
	if !ok {
		return errors.New("ViewNotFound", "there is no window for the view", "view", int64(view))
	}
	w.rendered = true
	w.lastScene = scene
	if !h.hasFrame || view == h.frameView {
		return h.draw(w, scene)
	}
	if w.wanted {
		w.pending = true
	}
	return nil
}

// draw draws one frame of the window, and handles the self-capture of the first
// window.
func (h *handler) draw(w *window, scene *ui.Scene) error {
	if h.windows[w.id] != w {
		return nil // the window was destroyed while its frame ran
	}
	if w.width <= 0 || w.height <= 0 {
		// The window has no size yet: the first frame the framework asked for came
		// before the first configure. The configure renders again.
		return nil
	}
	if h.r == nil {
		h.r = h.newRenderer(h.cfg, h.log)
	}
	capture := w == h.first && h.captureDue
	h.captureDue = false
	img, err := h.r.Draw(w.NativeSurface(), w.width, w.height, scene, capture)
	if err != nil {
		h.fail(err)
		return err
	}
	if w != h.first {
		return nil
	}
	switch {
	case img != nil:
		if err := h.shot.Save(img); err != nil {
			h.fail(err)
			return err
		}
		h.cancel() // done: Run returns, and the program exits with code 0
	case h.shot != nil && !h.armed:
		// The first frame is presented. Capture the one that follows the delay,
		// or, with no delay, the first one after which the app is idle.
		h.armed = true
		if h.shot.After() == 0 {
			if h.disp.IsIdle != nil {
				h.waitIdle(time.Now().Add(h.idleMax()))
				return nil
			}
			h.captureDue = true
			h.first.RequestFrame()
			return nil
		}
		time.AfterFunc(h.shot.After(), func() {
			h.conn.Post(func() {
				h.captureDue = true
				h.first.RequestFrame()
			})
		})
	}
	return nil
}

// idleMax is how long the self-capture waits for the app to become idle.
func (h *handler) idleMax() time.Duration {
	if h.cfg.idleMax > 0 {
		return h.cfg.idleMax
	}
	return 5 * time.Second
}

// waitIdle asks the app whether it is idle until it is, or until the deadline,
// and then asks for the frame that is read back. A frame that is scheduled or a
// task that is pending shows up only after the frame that is being drawn ended,
// so the question is asked from the event loop, not from the frame.
func (h *handler) waitIdle(deadline time.Time) {
	poll := h.cfg.idlePoll
	if poll <= 0 {
		poll = 20 * time.Millisecond
	}
	time.AfterFunc(poll, func() {
		h.conn.Post(func() {
			idle := h.disp.IsIdle()
			if !idle && time.Now().Before(deadline) {
				h.waitIdle(deadline)
				return
			}
			if !idle {
				h.log.Warn("the app was not idle after waiting; capturing the window anyway", "waited", h.idleMax())
			}
			h.captureDue = true
			h.first.RequestFrame()
		})
	})
}

// OnCloseRequest asks the app whether the window may close (the dispatcher's
// OnExitRequested, or the owner of the window controller of the window); if it may,
// it destroys the surface of the renderer, which draws into the window, and then
// lets the window close.
func (h *handler) OnCloseRequest(view ui.ViewID) bool {
	w, ok := h.windows[view]
	if !ok {
		return true
	}
	if !h.disp.RequestClose(view) {
		return false
	}
	h.forget(w)
	h.disp.RemoveView(view)
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

// glesRenderer is the OpenGL ES renderer of the engine: it draws the scene of the
// frame over developmentColor, into the surface of any window, with one GL context
// for all of them.
type glesRenderer struct {
	log      *slog.Logger
	skip     bool
	renderer *gles.Renderer
	surfaces map[gpu.SurfaceHandle]gpu.Surface
	reported int // the number of skipped commands the log last said
}

func newGLESRenderer(cfg *Config, log *slog.Logger) renderer {
	return &glesRenderer{log: log, skip: cfg.SkipUnsupported, surfaces: map[gpu.SurfaceHandle]gpu.Surface{}}
}

func (r *glesRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.renderer == nil {
		r.renderer = gles.NewRenderer(gles.RendererOpts{
			Background: ui.NewColor(developmentColor), HasBackground: true, SkipUnsupported: r.skip,
		})
		r.log.Info("OpenGL ES renderer ready")
	}
	surface, ok := r.surfaces[h]
	if !ok {
		var err error
		if surface, err = r.renderer.Attach(h); err != nil {
			return nil, err
		}
		r.surfaces[h] = surface
	}
	size := ui.Size{Width: float64(width), Height: float64(height)}
	var img *image.RGBA
	var err error
	if capture {
		img, err = r.renderer.RenderCapture(surface, scene, size, 1)
	} else {
		err = r.renderer.Render(surface, scene, size, 1)
	}
	if err != nil {
		return nil, err
	}
	if st := r.renderer.LastFrame(); st.Skipped > 0 && st.Skipped != r.reported {
		r.reported = st.Skipped
		r.log.Warn("the GPU renderer left out what it cannot draw yet", "skipped", st.Skipped, "drawn", st.Drawn, "features", st.Unsupported)
	}
	return img, nil
}

func (r *glesRenderer) Release(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		surface.Destroy()
		delete(r.surfaces, h)
	}
}

func (r *glesRenderer) Close() {
	for h, surface := range r.surfaces {
		surface.Destroy()
		delete(r.surfaces, h)
	}
	if r.renderer != nil {
		_ = r.renderer.Close()
		r.renderer = nil
	}
}
