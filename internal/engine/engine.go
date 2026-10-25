package engine

import (
	"context"
	"image"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
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
	// Renderer, when not empty, is "auto", or the name of the renderer to use
	// ("vulkan" or "gles" on Linux), and overrides the GLUTTER_RENDERER environment
	// variable. Empty means "auto": the first candidate of the OS that presents its
	// first frame. A renderer asked for by name never falls back (SPEC §9.6).
	Renderer string
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
	// DisableRasterCache turns off the raster cache of the GPU renderer (the layers
	// that RepaintBoundary retains for three frames are drawn as textures). The
	// GLUTTER_RASTER_CACHE environment variable set to "0" does the same.
	DisableRasterCache bool
	// DisableDamage stops the GPU renderer from telling the compositor which part of
	// the window changed (eglSwapBuffersWithDamage). GLUTTER_DAMAGE set to "0" does
	// the same.
	DisableDamage bool
	// Start, when set, is called once on the UI thread, after the first window
	// exists and the dispatcher knows its view, the settings and the displays,
	// and before the event loop runs. The framework attaches itself to the
	// dispatcher here (its callbacks, then ScheduleFrame). An error ends Run
	// with that error. Start replaces Scene: a Scene is drawn only when Start is
	// nil.
	Start func(d *ui.PlatformDispatcher) error

	// The fields below are for the tests of this package: a platform that is not the
	// real one (or a Wayland and an X11 that are not), and the renderer.
	idlePoll time.Duration // how often the capture asks whether the app is idle
	idleMax  time.Duration // how long it waits for the app to become idle
	// rendererNamed is set by Run when the renderer was asked for by name: a CPU Vulkan
	// device (lavapipe) is then allowed (SPEC §9.4).
	rendererNamed bool
	connect       func() (connection, error)                   // replaces the choice of backends_<goos>.go
	wayland       func() (connection, error)                   // Linux only: replaces the Wayland backend
	x11           func() (connection, error)                   // Linux only: replaces the X11 backend
	newRenderer   func(cfg *Config, log *slog.Logger) renderer // replaces the choice of the renderer
	candidates    func() []rendererCandidate                   // replaces the list of backends_<goos>.go
	// renewSurface and redraw are set by Run, for the renderers: renewSurface replaces the
	// native surface of the window that the handle names by a new one (the web's canvas keeps
	// the first kind of context it was asked for, so a renderer that follows a failed one
	// needs a fresh canvas, SPEC §9.6), and redraw asks for a frame of that window, from any
	// goroutine (a renderer that made its resources again after the GPU took them away).
	renewSurface func(gpu.SurfaceHandle)
	redraw       func(gpu.SurfaceHandle)
	// post runs a function on the UI thread (the platform's Post), and captured receives a frame that
	// the renderer read back after its Draw returned (a WebGPU read-back finishes in a later browser
	// task, SPEC §9.5); Run sets both. webgpuOpen is the explicit device that the web's connectDisplay
	// got before the platform installed its listeners (SPEC §23.3), or the error of why it has none.
	post        func(func())
	captured    func(*image.RGBA, error)
	webgpuOpen  func(gpu.SurfaceHandle) (explicit.Device, error)
	webgpuClose func()
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
	// Invalidate says that the window with the handle lost its content, so that the
	// next frame is presented whole. A handle that was never drawn into is ignored.
	Invalidate(h gpu.SurfaceHandle)
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

// FrameStatsEnv is the environment variable that, set to "1", makes the engine log a line every second with the frames it
// drew: how many, the average and the worst time to build and to draw a frame, and how
// many took longer than 1/60 s. scripts/perf.sh reads it.
//
// Flutter: no counterpart
const FrameStatsEnv = "GLUTTER_FRAME_STATS"

// RasterCacheEnv is the environment variable that, set to "0", turns off the raster cache of the GPU renderer, as
// Config.DisableRasterCache does.
//
// Flutter: no counterpart
const RasterCacheEnv = "GLUTTER_RASTER_CACHE"

// DamageEnv is the environment variable that, set to "0", stops the GPU renderer from telling the compositor what
// changed in a frame, as Config.DisableDamage does.
//
// Flutter: no counterpart
const DamageEnv = "GLUTTER_DAMAGE"

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
	runStart := time.Now()
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
		return errors.New(tagScreenshotUnsupported,
			"GLUTTER_SCREENSHOT reads the frame back through OpenGL, so it does not work with the shared-memory mode")
	}

	want := cfg.Renderer
	if want == "" {
		want = getenv(RendererEnv)
	}
	var choice *chooser
	if cfg.newRenderer == nil {
		cands := rendererCandidates()
		if cfg.candidates != nil {
			cands = cfg.candidates()
		}
		if choice, err = newChooser(&cfg, log, want, cands); err != nil {
			return err
		}
		cfg.rendererNamed = choice.named
	}

	stage := func(name string) {
		if getenv(FrameStatsEnv) == "1" {
			log.Info("startup", "stage", name, "since_run", time.Since(runStart).Round(time.Millisecond))
		}
	}
	var conn connection
	if cfg.connect != nil {
		conn, err = cfg.connect()
	} else {
		conn, err = connectDisplay(&cfg, getenv, log)
	}
	if err != nil {
		cfg.closeWebGPU()
		return err
	}
	defer func() {
		// A failure to close is worth reporting only when nothing failed before it.
		if cerr := conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	stage("connected to the display")
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
			return errors.New(tagDecorationsUnknown, "GLUTTER_DECORATIONS is not auto, app or libdecor", "value", v)
		}
	}
	win, err := conn.NewWindow(wcfg)
	if err != nil {
		return err
	}
	stage("window created")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := &handler{cfg: &cfg, log: log, conn: conn, shot: shot, cancel: cancel, windows: map[ui.ViewID]*window{},
		appID: wcfg.AppID, decorations: wcfg.Decorations, runStart: runStart, frameStats: getenv(FrameStatsEnv) == "1"}
	cfg.renewSurface, cfg.redraw = h.renewSurface, h.redrawWindow
	cfg.post, cfg.captured = conn.Post, h.onCapture
	h.first = h.addWindow(win)
	h.first.title = wcfg.Title
	if cfg.newRenderer != nil {
		h.newRenderer = cfg.newRenderer
	} else {
		h.newRenderer = func(*Config, *slog.Logger) renderer { return choice }
	}
	h.disp = ui.NewPlatformDispatcher(h)
	h.disp.SetDisplays(h.displays())
	h.disp.UpdatePlatformConfiguration(PlatformConfiguration(conn.Settings()))
	h.disp.AddView(win.ID(), ui.ViewConfiguration{})
	if cfg.Start != nil {
		if err := cfg.Start(h.disp); err != nil {
			return err
		}
	} else if cfg.Scene != nil {
		h.disp.OnDrawFrame = h.drawSceneFunc
	}
	stage("framework started")
	if cfg.Hooks.Window != nil {
		cfg.Hooks.Window(win, cancel)
	}
	h.startAccessibility(ctx)
	stage("accessibility started")
	runErr := conn.Run(ctx, h)
	h.release()
	if runErr != nil {
		return runErr
	}
	return h.err
}

// windowConfig returns the config of the first window.
func (c *Config) windowConfig() platform.WindowConfig {
	rest := c.Window
	rest.Icon = ui.WindowIcon{} // the icon has slices, which cannot be compared
	if reflect.DeepEqual(rest, platform.WindowConfig{}) {
		w := platform.DefaultWindowConfig()
		w.Title = "Glutter"
		w.Icon = c.Window.Icon
		return w
	}
	return c.Window
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
	// metrics are the last the platform reported, and hasMetrics says it did.
	metrics    platform.ViewMetrics
	hasMetrics bool
	// pending is whether lastScene was rendered during the frame of another window:
	// the frame of this window then only presents it.
	pending bool
	// title is the title of the window, which names its frame for a screen reader, and
	// focused is whether the platform said it has the focus (it is the active window).
	title   string
	focused bool
	// hidden is whether the framework took the window off the screen
	// (SetWindowHidden): it asks the platform for no frame until it is shown.
	hidden bool
}

// handler is the platform.Handler of the engine: it draws the windows, captures
// the self-capture screenshot, and passes input to the hooks.
type handler struct {
	platform.NopHandler
	cfg    *Config
	log    *slog.Logger
	conn   connection
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
	// mouse is the state of the mouse pointer as the engine last saw it: whether it is in a view
	// (an added event without a removed one after it), which view, and its last event.
	mouse struct {
		inside bool
		view   ui.ViewID
		last   ui.PointerData
	}
	// input is what arrived since the last frame, which the next one handles first.
	input inputQueue
	// acc is the bridge to the assistive technologies of the platform, or nil.
	acc platform.Accessibility

	// clock measures the frames for the dispatcher's OnReportTimings.
	clock frameClock
	// runStart is when Run began, for the log of the first frame; firstLogged is set
	// once it was logged. frameStats asks for a line of frame statistics every second
	// (FrameStatsEnv).
	runStart    time.Time
	firstLogged bool
	drewOnce    bool
	frameStats  bool
	stats       frameStats

	// platformLifecycle is whether the platform reported a lifecycle state itself
	// (the web does); otherwise the engine derives it from the windows (SPEC §13).
	platformLifecycle bool

	armed      bool // the capture is scheduled
	captureDue bool // the next frame is read back
	err        error
}

var (
	_ platform.Handler              = (*handler)(nil)
	_ ui.PlatformHost               = (*handler)(nil)
	_ ui.CursorHost                 = (*handler)(nil)
	_ ui.WindowHost                 = (*handler)(nil)
	_ ui.WindowControlHost          = (*handler)(nil)
	_ ui.WindowAppearanceHost       = (*handler)(nil)
	_ ui.IdleInhibitHost            = (*handler)(nil)
	_ ui.ViewFocusHost              = (*handler)(nil)
	_ platform.KeyboardFocusHandler = (*handler)(nil)
	_ ui.ClipboardHost              = (*handler)(nil)
	_ ui.RichClipboardHost          = (*handler)(nil)
	_ ui.TextInputHost              = (*handler)(nil)
	_ ui.SemanticsHost              = (*handler)(nil)

	_ platform.AccessibilityHost = (*handler)(nil)
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

// SetWindowIcon sets the icon of the window of the view (ui.WindowAppearanceHost).
func (h *handler) SetWindowIcon(view ui.ViewID, icon ui.WindowIcon) error {
	if w, ok := h.windows[view]; ok {
		return w.SetIcon(icon)
	}
	return nil
}

// SetWindowContentType gives the compositor a hint about the window of the view
// (ui.WindowAppearanceHost).
func (h *handler) SetWindowContentType(view ui.ViewID, contentType ui.ContentType) {
	if w, ok := h.windows[view]; ok {
		w.SetContentType(contentType)
	}
}

// PortalParentHandle asks the window of the view how the portals name it
// (ui.PortalHost); done gets an empty handle when the window or the platform has
// none.
func (h *handler) PortalParentHandle(view ui.ViewID, done func(handle string)) {
	w, ok := h.windows[view]
	if !ok {
		done("")
		return
	}
	if p, ok := w.Window.(platform.PortalHandleProvider); ok {
		p.PortalHandle(done)
		return
	}
	done("")
}

// AppID is the application ID the windows were made with (ui.PortalHost).
func (h *handler) AppID() string { return h.appID }

// ActivationToken asks the platform for an activation token for the window of the
// view (ui.PortalHost); done gets an empty token when the platform has none.
func (h *handler) ActivationToken(view ui.ViewID, done func(token string)) {
	if p, ok := h.conn.(platform.ActivationTokenProvider); ok {
		p.RequestActivationToken(view, done)
		return
	}
	done("")
}

// PlatformService hands out the optional services that the platform has by itself
// (ui.ServiceHost): for package desktop the name mobile.ServiceDesktop gives the
// platform's URI opener and notifications, when the platform has them (Android, SPEC
// §24.11), and nil elsewhere, where D-Bus or the web's own services do the work. It
// may be called from any goroutine: the connection never changes.
func (h *handler) PlatformService(name string) any {
	if name != mobile.ServiceDesktop {
		return nil
	}
	if p, ok := h.conn.(platform.DesktopProvider); ok {
		if svc := p.Desktop(); svc != nil {
			return svc
		}
	}
	return nil
}

// KeepScreenOn stops the screen from going idle while the window of the view is
// visible (ui.IdleInhibitHost).
func (h *handler) KeepScreenOn(view ui.ViewID, on bool) error {
	w, ok := h.windows[view]
	if !ok {
		return errors.New(tagIdleInhibitUnavailable, "the window of the view does not exist", "view", view)
	}
	return w.KeepScreenOn(on)
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
	if err := spec.Icon.Validate(); err != nil {
		return 0, ui.ViewConfiguration{}, err
	}
	cfg.Icon, cfg.ContentType = spec.Icon, spec.ContentType
	switch spec.State {
	case ui.WindowStartStateMaximized:
		cfg.State = platform.WindowStateMaximized
	case ui.WindowStartStateMinimized:
		cfg.State = platform.WindowStateMinimized
	case ui.WindowStartStateFullscreen:
		cfg.State = platform.WindowStateFullscreen
	}
	win, err := h.conn.NewWindow(cfg)
	if err != nil {
		return 0, ui.ViewConfiguration{}, err
	}
	w := h.addWindow(win)
	w.title = spec.Title
	if h.acc != nil {
		h.acc.SetWindow(w.id, w.title, false)
	}
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
	if h.acc != nil {
		h.acc.RemoveView(view)
	}
	w.Close()
}

// forget destroys the surface of the window and drops it. The window must still
// exist.
func (h *handler) forget(w *window) {
	h.releaseMouse(w.id)
	if h.r != nil {
		h.r.Release(w.NativeSurface())
	}
	delete(h.windows, w.id)
	h.updateLifecycle()
}

// SetWindowHidden takes the window of the view off the screen, or puts it back
// (ui.WindowControlHost). A window that is shown again presents its next frame
// whole, and gets the frame that was asked for while it was hidden. On a platform
// that cannot hide windows nothing changes.
func (h *handler) SetWindowHidden(view ui.ViewID, hidden bool) {
	w, ok := h.windows[view]
	if !ok || w.hidden == hidden {
		return
	}
	if !w.SetHidden(hidden) {
		return
	}
	w.hidden = hidden
	if !hidden {
		if h.r != nil {
			h.r.Invalidate(w.NativeSurface())
		}
		if w.wanted {
			w.RequestFrame()
		}
	}
	h.updateLifecycle()
}

// updateLifecycle tells the dispatcher the lifecycle state that the windows give, on
// a platform that does not report one itself (SPEC §13): Hidden when every window is
// hidden, Resumed when one of them has the focus, and Inactive otherwise.
func (h *handler) updateLifecycle() {
	if h.platformLifecycle || len(h.windows) == 0 {
		return
	}
	state := ui.AppLifecycleStateHidden
	for _, w := range h.windows {
		if w.hidden {
			continue
		}
		if w.focused {
			state = ui.AppLifecycleStateResumed
			break
		}
		state = ui.AppLifecycleStateInactive
	}
	h.disp.SetAppLifecycleState(state)
}

// StartDrag starts a drag and drop operation from the window of the view
// (ui.DropHost).
func (h *handler) StartDrag(view ui.ViewID, request ui.DragRequest) error {
	w, ok := h.windows[view]
	if !ok {
		return errors.New(tagViewNotFound, "the view has no window", "view", int(view))
	}
	return w.StartDrag(request)
}

// OnDrop hands a drag and drop operation over a view to the framework and returns
// the action it accepts.
func (h *handler) OnDrop(event ui.DropEvent) ui.DropActions { return h.disp.DispatchDrop(event) }

// OnDragEnd tells the framework how a drag that it started ended.
func (h *handler) OnDragEnd(result ui.DragResult) { h.disp.DispatchDragEnd(result) }

// SetWindowTitle sets the title of the window (ui.WindowControlHost).
func (h *handler) SetWindowTitle(view ui.ViewID, title string) {
	if w, ok := h.windows[view]; ok {
		w.SetTitle(title)
		w.title = title
		if h.acc != nil {
			h.acc.SetWindow(view, title, w.focused)
		}
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
	if w, ok := h.windows[view]; ok && state == ui.ViewFocusStateFocused && !w.focused && !w.hidden {
		w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate})
	}
}

// windowOfSurface finds the window that a surface handle names.
func (h *handler) windowOfSurface(sh gpu.SurfaceHandle) *window {
	for _, w := range h.windows {
		if w.NativeSurface() == sh {
			return w
		}
	}
	return nil
}

// renewSurface gives the window of the handle a new native surface, when its platform can
// (platform.SurfaceRenewer). It runs on the UI thread, inside a frame.
func (h *handler) renewSurface(sh gpu.SurfaceHandle) {
	if w := h.windowOfSurface(sh); w != nil {
		if r, ok := w.Window.(platform.SurfaceRenewer); ok {
			r.RenewSurface()
		}
	}
}

// redrawWindow asks for a frame of the window of the handle. It may be called from any
// goroutine: the request is posted to the UI thread. The frame draws the scene of the last
// one again when the framework has no new one.
func (h *handler) redrawWindow(sh gpu.SurfaceHandle) {
	h.conn.Post(func() {
		if w := h.windowOfSurface(sh); w != nil {
			w.RequestFrame()
		}
	})
}

// OnContentLost has the next frame of the window presented whole (platform.Handler).
func (h *handler) OnContentLost(view ui.ViewID) {
	if w, ok := h.windows[view]; ok && h.r != nil {
		h.r.Invalidate(w.NativeSurface())
	}
}

// OnSurfaceLost lets the renderer go of the surface that the platform took away, before the
// platform releases the native window (platform.SurfaceHandler).
func (h *handler) OnSurfaceLost(_ ui.ViewID, lost gpu.SurfaceHandle) {
	if h.r != nil {
		h.r.Release(lost)
	}
}

// OnKeyboardFocus routes the keys, the text input and the input method to the view
// (platform.KeyboardFocusHandler). The view focus that the framework hears of is
// OnViewFocus's.
func (h *handler) OnKeyboardFocus(view ui.ViewID, focused bool) {
	if _, ok := h.windows[view]; ok && focused {
		h.focusView = view
	}
}

// OnViewFocus tells the dispatcher that a view got or lost the focus (the window is
// the active one), with the input that came before it. A platform that does not call
// OnKeyboardFocus has the keys follow it too.
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
	w.focused = focused
	if h.acc != nil {
		h.acc.SetWindow(view, w.title, focused)
	}
	h.queueInput(w, func() {
		h.disp.DispatchViewFocus(ui.ViewFocusEvent{ViewID: view, State: state, Direction: ui.ViewFocusDirectionUndefined})
		// After the focus event: the framework moves its focus into the window first,
		// so the focus manager does not restore the focus it suspended in another one
		// when the app resumes.
		h.updateLifecycle()
	})
}

// OnDisplaysChanged gives the dispatcher the new list of displays, and the views
// the display and the pixel ratio they now have.
func (h *handler) OnDisplaysChanged() {
	h.disp.SetDisplays(h.displays())
	for _, w := range h.windows {
		if w.hasMetrics {
			h.disp.UpdateViewConfiguration(w.id, h.viewConfiguration(w.metrics))
		}
	}
}

func (h *handler) OnMetrics(view ui.ViewID, m platform.ViewMetrics) {
	w, ok := h.windows[view]
	if !ok {
		return
	}
	w.width, w.height = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
	w.metrics, w.hasMetrics = m, true
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
		rate := d.RefreshRate
		if rate <= 0 {
			rate = defaultRefreshRate
		}
		out = append(out, ui.Display{
			ID:               d.ID,
			DevicePixelRatio: dpr,
			Size:             ui.Size{Width: d.Bounds.Width() * dpr, Height: d.Bounds.Height() * dpr},
			RefreshRate:      rate,
		})
	}
	return out
}

// PlatformConfiguration turns the platform's settings into the dispatcher's. The
// engine does it for every OnSettings; tests of the layers above use it to stand in
// for the engine.
//
// Flutter: no counterpart
func PlatformConfiguration(s platform.Settings) ui.PlatformConfiguration {
	c := ui.PlatformConfiguration{
		Locales:            s.Locales,
		TextScaleFactor:    s.TextScaleFactor,
		PlatformBrightness: s.PlatformBrightness,
		AccessibilityFeatures: ui.AccessibilityFeatures{
			AccessibleNavigation: s.AccessibleNavigation,
			DisableAnimations:    s.DisableAnimations,
			ReduceMotion:         s.DisableAnimations,
			HighContrast:         s.HighContrast,
			BoldText:             s.BoldText,
		},
		AlwaysUse24HourFormat: s.AlwaysUse24HourFormat,
		ScaleFontSize:         s.ScaleFontSize,
		UIFontFamily:          s.UIFontFamily,
		UIFontSize:            s.UIFontSize,
		MonospaceFontFamily:   s.MonospaceFontFamily,
		MonospaceFontSize:     s.MonospaceFontSize,
	}
	if s.HasAccentColor {
		c.AccentColor = ui.Ptr(s.AccentColor)
	}
	return c
}

func (h *handler) OnSettings(s platform.Settings) {
	h.disp.UpdatePlatformConfiguration(PlatformConfiguration(s))
}

func (h *handler) OnLifecycle(state ui.AppLifecycleState) {
	h.platformLifecycle = true
	h.disp.SetAppLifecycleState(state)
}

func (h *handler) OnKey(d ui.KeyData) {
	h.queueInput(h.windowOf(h.focusView), func() { h.dispatchKey(d) })
}

// dispatchKey gives a key event to the hook and the dispatcher and returns whether the
// framework handled it.
func (h *handler) dispatchKey(d ui.KeyData) bool {
	if f := h.cfg.Hooks.Key; f != nil {
		f(d)
	}
	return h.disp.DispatchKeyData(d)
}

// OnKeySync handles a key event at once, after the input that is waiting for its frame, and
// returns whether the framework handled it (platform.SyncInputHandler). A frame is asked for,
// as for any input, because the key can have changed the app.
func (h *handler) OnKeySync(d ui.KeyData) bool {
	w := h.windowOf(h.focusView)
	if h.cfg.ShmFill {
		return h.dispatchKey(d)
	}
	h.input.flush()
	handled := h.dispatchKey(d)
	h.askFrame(w)
	return handled
}

// OnTextInputSync handles a text-input event at once, after the input that is waiting for its
// frame (platform.SyncInputHandler): the framework's new state of the text field has been sent to
// the window when it returns. A frame is asked for, as for any input.
func (h *handler) OnTextInputSync(event ui.TextInputEvent) {
	w := h.windowOf(h.focusView)
	if h.cfg.ShmFill {
		h.disp.DispatchTextInput(event)
		return
	}
	h.input.flush()
	h.disp.DispatchTextInput(event)
	h.askFrame(w)
}

func (h *handler) OnPointer(batch []ui.PointerData) {
	batch = append([]ui.PointerData(nil), batch...)
	w := h.notePointer(batch)
	h.queueInput(w, func() { h.dispatchPointer(batch) })
}

// OnPointerSync handles pointer data at once, after the input that is waiting for its frame
// (platform.SyncInputHandler): the answers of the framework, such as the scroll that a wheel
// event was taken by, are there when it returns.
func (h *handler) OnPointerSync(batch []ui.PointerData) {
	batch = append([]ui.PointerData(nil), batch...)
	w := h.notePointer(batch)
	if h.cfg.ShmFill {
		h.dispatchPointer(batch)
		return
	}
	h.input.flush()
	h.dispatchPointer(batch)
	h.askFrame(w)
}

// notePointer remembers the view and the mouse state of a batch and returns the window it is for.
func (h *handler) notePointer(batch []ui.PointerData) *window {
	if len(batch) > 0 {
		h.pointerView = batch[len(batch)-1].ViewID
	}
	h.noteMouse(batch)
	return h.windowOf(h.pointerView)
}

func (h *handler) dispatchPointer(batch []ui.PointerData) {
	if f := h.cfg.Hooks.Pointer; f != nil {
		f(batch)
	}
	h.disp.DispatchPointerDataPacket(ui.PointerDataPacket{Data: batch})
}

// SetBrowserContextMenuEnabled shows or hides the browser's context menu in the windows that
// have one (ui.ContextMenuHost).
func (h *handler) SetBrowserContextMenuEnabled(enabled bool) bool {
	applied := false
	for _, w := range h.windows {
		if c, ok := w.Window.(platform.ContextMenuController); ok {
			c.SetBrowserContextMenuEnabled(enabled)
			applied = true
		}
	}
	return applied
}

// noteMouse follows the mouse pointer through the events of a batch.
func (h *handler) noteMouse(batch []ui.PointerData) {
	for _, d := range batch {
		if d.Kind != ui.PointerDeviceKindMouse || d.SignalKind != ui.PointerSignalKindUnspecified {
			continue
		}
		switch d.Change {
		case ui.PointerChangeAdd:
			h.mouse.inside, h.mouse.view = true, d.ViewID
		case ui.PointerChangeRemove:
			h.mouse.inside = false
		}
		h.mouse.last = d
	}
}

// releaseMouse makes the mouse pointer leave the view that is going to be removed, if it is in
// it. The platform does the same when it destroys the window, but by then the view is gone, the
// framework drops the event, and the mouse tracker never hears that the pointer was removed: the
// pointer's next added event, in another window, breaks the order the tracker checks.
func (h *handler) releaseMouse(view ui.ViewID) {
	if !h.mouse.inside || h.mouse.view != view {
		return
	}
	h.input.flush() // what arrived before goes first
	if !h.mouse.inside {
		return
	}
	var out []ui.PointerData
	last := h.mouse.last
	if last.Buttons != 0 {
		up := last
		up.Change, up.Synthesized, up.Buttons = ui.PointerChangeUp, true, 0
		out = append(out, up)
	}
	rm := last
	rm.Change, rm.Synthesized, rm.Buttons = ui.PointerChangeRemove, false, 0
	out = append(out, rm)
	h.mouse.inside = false
	h.disp.DispatchPointerDataPacket(ui.PointerDataPacket{Data: out})
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
	if w.hidden {
		return // asked for when the window is shown
	}
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
			h.reportFrame(false)
		}
		return
	}
	w.pending = false
	h.clock.beginFrame()
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
	h.reportFrame(true)
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

// RequestAppExit ends Run (ui.PlatformHost), unless the platform says the app cannot end itself
// (a web page cannot close itself, SPEC §23.3).
func (h *handler) RequestAppExit() {
	if c, ok := h.conn.(platform.ExitController); ok && !c.CanExit() {
		return
	}
	h.cancel()
}

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
		return errors.New(tagViewNotFound, "there is no window for the view", "view", int64(view))
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
	first := !h.drewOnce
	h.drewOnce = true
	h.clock.beginRaster()
	region := traceRaster()
	img, err := h.r.Draw(w.NativeSurface(), w.width, w.height, scene, capture)
	region.End()
	h.clock.endRaster()
	if first && h.frameStats {
		h.log.Info("startup", "stage", "first draw (renderer, GL context, shaders, first frame)", "took", (h.clock.rasterFinish - h.clock.rasterStart).Round(time.Millisecond))
	}
	if err == nil {
		h.noteFrame()
	}
	if err != nil {
		h.fail(err)
		return err
	}
	if w != h.first {
		return nil
	}
	switch {
	case img != nil:
		return h.finishScreenshot(img)
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

// finishScreenshot saves the frame that was read back, or hands it to the platform that has no
// files (the web), and ends Run when it is done.
func (h *handler) finishScreenshot(img *image.RGBA) error {
	if poster, ok := h.conn.(platform.ScreenshotPoster); ok {
		// A platform that has no files (the web) takes the PNG itself; Run ends when it
		// has arrived.
		data, err := h.shot.Encode(img)
		if err != nil {
			h.fail(err)
			return err
		}
		poster.PostScreenshot(data, func(err error) {
			if err != nil {
				h.fail(err)
				return
			}
			h.cancel()
		})
		return nil
	}
	if err := h.shot.Save(img); err != nil {
		h.fail(err)
		return err
	}
	h.cancel() // done: Run returns, and the program exits with code 0
	return nil
}

// onCapture receives the frame of a renderer whose read-back finished after Draw returned,
// on the UI thread (the renderer posts it there), and finishes the self-capture with it.
func (h *handler) onCapture(img *image.RGBA, err error) {
	if err != nil {
		h.fail(err)
		return
	}
	if h.shot != nil && img != nil {
		_ = h.finishScreenshot(img) // finishScreenshot reports its own failure through h.fail
	}
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

// startAccessibility starts the bridge to the assistive technologies, when the platform
// has one.
func (h *handler) startAccessibility(ctx context.Context) {
	svc := h.conn.Services()
	if svc == nil {
		return
	}
	acc := svc.Accessibility()
	if acc == nil {
		return
	}
	h.acc = acc
	acc.SetWindow(h.first.id, h.first.title, false)
	acc.Start(ctx, h)
}

// ScreenReaderChanged turns the semantics tree of the framework on or off, as a screen
// reader starts or stops (platform.AccessibilityHost).
func (h *handler) ScreenReaderChanged(enabled bool) { h.disp.SetSemanticsEnabled(enabled) }

// PerformAction gives the framework an action that an assistive technology asked for
// (platform.AccessibilityHost).
func (h *handler) PerformAction(event ui.SemanticsActionEvent) {
	h.disp.DispatchSemanticsActionEvent(event)
}

// UpdateSemantics gives the bridge the changes to the semantics tree of a view
// (ui.SemanticsHost).
func (h *handler) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	if h.acc != nil {
		h.acc.UpdateSemantics(view, update)
	}
}

// SetSemanticsTreeEnabled does nothing: the bridge asks for the tree through
// ScreenReaderChanged, and an app that sets the tree on by itself still gets it in
// UpdateSemantics (ui.SemanticsHost).
func (h *handler) SetSemanticsTreeEnabled(bool) {}

// release destroys the renderer and stops the accessibility bridge. It is safe to call
// twice.
func (h *handler) release() {
	if h.acc != nil {
		h.acc.Close()
		h.acc = nil
	}
	if h.r != nil {
		h.r.Close()
		h.r = nil
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDecorationsUnknown     = "DecorationsUnknown"
	tagIdleInhibitUnavailable = "IdleInhibitUnavailable"
	tagScreenshotUnsupported  = "ScreenshotUnsupported"
	tagViewNotFound           = "ViewNotFound"
)
