package android

import (
	"context"
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// Platform is the platform.Platform of Android: one window that fills the activity's view, drawn
// by the UI thread, which owns the framework and the renderer (SPEC §24.3). It is plain Go over a
// host (the looper and the choreographer, host_android.go) and a bridge (the calls of the Java
// side), so that the tests run it with fakes.
//
// Flutter: shell/platform/android/platform_view_android.cc PlatformViewAndroid
type Platform struct {
	b    *bridge
	host host
	log  *slog.Logger

	// Everything below belongs to the UI thread.
	handler platform.Handler
	win     *window
	frames  frameRequests
	// touch and conv turn motion events into pointer data (motion.go).
	touch *touchProcessor
	conv  *pointerConverter
	keys  *keyboardManager
	// text is the text field that has the focus (textinput.go); clipboard is the clipboard
	// (clipboard.go).
	text      textInput
	clipboard *clipboard
	// handlesBack is whether the framework handles the back gesture (back.go); framed is whether a
	// frame was drawn.
	handlesBack, framed bool
	// devEvents is the script of GLUTTER_DEV_EVENTS, in debug builds; devStarted says it was started.
	devEvents  []DevEvent
	devStarted bool
	// a11y is the accessibility bridge, which Services gives out (accessibility.go).
	a11y *accessibility
}

var _ platform.Platform = (*Platform)(nil)

// newPlatform makes the platform over a host and a bridge.
func newPlatform(b *bridge, h host, log *slog.Logger) *Platform {
	if log == nil {
		log = slog.Default()
	}
	p := &Platform{b: b, host: h, log: log, touch: newTouchProcessor(), conv: newPointerConverter(), keys: newKeyboardManager()}
	p.a11y = newAccessibility(p)
	return p
}

// Run owns the UI thread's loop: it runs the events of the Java side and the functions that Post
// queued, in order, and asks the choreographer for a frame while one is wanted and can be drawn.
// It returns nil when ctx is cancelled. A second call while Run is active fails with
// PlatformAlreadyRunning.
//
// Flutter: shell/platform/android/android_shell_holder.cc AndroidShellHolder (the UI task runner)
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	if !p.b.start(p.host.Wake) {
		return errors.New(tagPlatformAlreadyRunning, "Run was called while the platform is already running")
	}
	defer p.b.stop()
	p.handler = h
	defer func() { p.handler = nil }()
	defer context.AfterFunc(ctx, p.host.Wake)()

	state, lifecycle, focused := p.b.snapshotAll()
	h.OnLifecycle(lifecycle.appState(focused))
	if p.win != nil {
		if state.ready() {
			h.OnMetrics(p.win.id, state.viewMetrics())
			p.frames.request()
		}
		if focused {
			h.OnViewFocus(p.win.id, true)
		}
	}
	for ctx.Err() == nil {
		for fn, ok := p.b.pop(); ok && ctx.Err() == nil; fn, ok = p.b.pop() {
			fn(p)
		}
		if ctx.Err() != nil {
			break
		}
		if p.frames.shouldSchedule(p.canDraw()) {
			p.host.RequestVsync()
			p.frames.scheduledNow()
		}
		if frameTime, vsync := p.host.Poll(); vsync {
			p.onVsync(frameTime)
		}
	}
	return nil
}

// canDraw says whether a frame can be drawn now: the surface has a window and a size, and the
// activity is not stopped (SPEC §24.8: no frames while hidden or without a surface).
func (p *Platform) canDraw() bool {
	state, lifecycle, _ := p.b.snapshotAll()
	return state.ready() && lifecycle.visible()
}

// onVsync is the choreographer's frame callback: it asks the handler for the frame.
func (p *Platform) onVsync(frameTime time.Duration) {
	if !p.frames.fired(p.canDraw()) || p.win == nil || p.win.closed {
		return
	}
	p.handler.OnFrame(p.win.id, frameTime)
	p.framed = true
	p.startDevEvents()
}

// surfaceLost tells the handler that the native window is gone and lets go of it. The handler
// first, so that the renderer drops what it made for the window (SPEC §24.3).
func (p *Platform) surfaceLost(window uintptr) {
	if sh, ok := p.handler.(platform.SurfaceHandler); ok && p.win != nil {
		sh.OnSurfaceLost(p.win.id, gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: window})
	}
	p.host.ReleaseWindow(window)
}

// surfaceChangedState is a surface that was created: the view reports its metrics again and the
// new surface, which is blank, is drawn as soon as it can be.
func (p *Platform) surfaceChangedState() {
	state := p.b.snapshot()
	if p.win == nil || !state.ready() {
		return
	}
	p.handler.OnMetrics(p.win.id, state.viewMetrics())
	p.frames.request()
}

// metricsChanged reports a new size or density of the view and the display.
func (p *Platform) metricsChanged() {
	state := p.b.snapshot()
	if p.win == nil {
		return
	}
	if state.width > 0 && state.height > 0 {
		p.handler.OnMetrics(p.win.id, state.viewMetrics())
		p.frames.request()
	}
	p.handler.OnDisplaysChanged()
}

// lifecycleChanged reports the activity's lifecycle. A frame that was wanted while the app was
// stopped is asked for when it is visible again (the loop does it).
func (p *Platform) lifecycleChanged() {
	_, lifecycle, focused := p.b.snapshotAll()
	p.handler.OnLifecycle(lifecycle.appState(focused))
	if lifecycle.visible() {
		p.frames.request()
	}
}

// focusChanged reports that the window got or lost the focus.
func (p *Platform) focusChanged() {
	_, lifecycle, focused := p.b.snapshotAll()
	if p.win != nil {
		p.handler.OnViewFocus(p.win.id, focused)
	}
	// The focus is part of the app's state: a resumed activity without it is Inactive.
	p.handler.OnLifecycle(lifecycle.appState(focused))
}

// Post runs fn on the UI thread, in order with the events of the Java side. It is safe to call
// from any goroutine.
//
// Flutter: shell/common/shell.cc Shell::GetTaskRunners (PostTask)
func (p *Platform) Post(fn func()) { p.b.post(func(*Platform) { fn() }) }

// NewWindow returns the one window of the activity. A second window, a dialog, a borderless
// window or a window with a parent fails with WindowUnsupported (SPEC §24.2).
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterView.java FlutterView
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	if p.win != nil {
		return nil, errors.New(tagWindowUnsupported, "Android has one window per activity: a second window was asked for")
	}
	if (cfg.Kind != platform.WindowKindUnspecified && cfg.Kind != platform.WindowKindRegular) || cfg.HasParent {
		return nil, errors.New(tagWindowUnsupported, "Android has one window per activity: dialogs and borderless windows are not supported")
	}
	p.win = &window{p: p, id: ui.ImplicitViewID, title: cfg.Title}
	return p.win, nil
}

// Displays lists the one display, with the density and the refresh rate Android reports.
//
// Flutter: shell/platform/android/android_display.cc AndroidDisplay
func (p *Platform) Displays() []platform.Display {
	state := p.b.snapshot()
	return []platform.Display{state.display()}
}

// Settings returns the system's settings that the Java side reported (SPEC §24.7); until it does,
// the defaults: light, the text scale 1 and the locale en-US.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/SettingsChannel.java SettingsChannel
func (p *Platform) Settings() platform.Settings {
	s := p.b.settingsSnapshot().platformSettings(p.host.ScaledFontSize)
	s.AccessibleNavigation = p.b.a11ySnapshot().navigation
	return s
}

// settingsChanged reports that the system's settings changed.
func (p *Platform) settingsChanged() {
	if p.handler != nil {
		p.handler.OnSettings(p.Settings())
	}
}

// Services returns the services of Android: the accessibility bridge (step 25.11, accessibility.go).
// The desktop services of step 25.12 are not part of it.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services { return services{p.a11y} }

// Close releases the platform. The host belongs to the process, so there is nothing to release.
//
// Flutter: no counterpart
func (p *Platform) Close() error { return nil }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPlatformAlreadyRunning = "PlatformAlreadyRunning"
	tagWindowUnsupported      = "WindowUnsupported"
)
