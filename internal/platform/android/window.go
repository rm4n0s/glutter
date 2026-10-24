package android

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// window is the one window of the activity: the SurfaceView that fills it. Most of what a
// desktop window can be told does nothing here (SPEC §24.2): the activity owns its size, its
// state and its title.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterView.java FlutterView
type window struct {
	p      *Platform
	id     ui.ViewID
	title  string
	closed bool
	// uiChange hears of the system's bars appearing and disappearing (systemui.go).
	uiChange func(visible bool)
}

var _ platform.Window = (*window)(nil)

// ID returns the view ID.
func (w *window) ID() ui.ViewID { return w.id }

// NativeSurface returns the handle of the surface's native window, or the zero handle while the
// activity has no surface.
func (w *window) NativeSurface() gpu.SurfaceHandle {
	state := w.p.b.snapshot()
	return state.handle()
}

// SetTitle keeps the title; the task description's label comes with step 25.9.
func (w *window) SetTitle(title string) { w.title = title }

// SetSize does nothing: the activity's window has the size the system gives it.
func (w *window) SetSize(ui.Size) {}

// SetSizeConstraints does nothing.
func (w *window) SetSizeConstraints(_, _ ui.Size) {}

// SetState makes the window fullscreen (the immersive mode of the system's interface) or normal
// (edge to edge); the other states are the activity's own.
func (w *window) SetState(state platform.WindowState) {
	switch state {
	case platform.WindowStateFullscreen:
		w.SetEnabledSystemUIMode(ui.SystemUIModeImmersiveSticky, nil)
	case platform.WindowStateNormal:
		w.SetEnabledSystemUIMode(ui.SystemUIModeEdgeToEdge, nil)
	}
}

// SetHidden returns false: an activity cannot hide its window.
func (w *window) SetHidden(bool) bool { return false }

// SetCursor shows the cursor with the PointerIcon of the system (a mouse or a stylus that hovers),
// as Flutter's MouseCursorPlugin does.
//
// Flutter: shell/platform/android/io/flutter/plugin/mouse/MouseCursorPlugin.java MouseCursorPlugin
func (w *window) SetCursor(c platform.MouseCursor) { w.p.host.SetCursor(PointerIconType(c)) }

// RequestFrame asks for a frame: one choreographer callback is asked for when the app is visible
// and has a surface, and requests made before that are one frame that comes then.
func (w *window) RequestFrame() { w.p.frames.request() }

// StartDrag fails with DragUnsupported: drag and drop with other apps is not in 1.6 (SPEC §24.5).
func (w *window) StartDrag(ui.DragRequest) error {
	return errors.New(tagDragUnsupported, "dragging to other apps is not supported on Android")
}

// PerformWindowAction does nothing.
func (w *window) PerformWindowAction(ui.WindowAction) {}

// Close marks the window closed: no frame is asked for any more.
func (w *window) Close() { w.closed = true }

// SetIcon does nothing.
func (w *window) SetIcon(ui.WindowIcon) error { return nil }

// SetContentType does nothing.
func (w *window) SetContentType(ui.ContentType) {}

// KeepScreenOn sets FLAG_KEEP_SCREEN_ON of the window.
//
// Flutter: no counterpart
func (w *window) KeepScreenOn(on bool) error {
	a := int64(0)
	if on {
		a = 1
	}
	w.p.host.Command(command{Op: opKeepScreenOn, A: a})
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragUnsupported = "DragUnsupported"
)

var _ ui.SystemChromeHost = (*window)(nil)

// SetPreferredOrientations sets the requested orientation of the activity.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.setSystemChromePreferredOrientations
func (w *window) SetPreferredOrientations(orientations []ui.DeviceOrientation) {
	w.p.host.Command(command{Op: opOrientation, A: int64(requestedOrientation(orientations))})
}

// SetEnabledSystemUIMode shows or hides the system's bars.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.setSystemChromeEnabledSystemUIMode
func (w *window) SetEnabledSystemUIMode(mode ui.SystemUIMode, overlays []ui.SystemUIOverlay) {
	w.p.host.Command(command{Op: opUIMode, A: int64(mode), B: overlayBits(overlays)})
}

// SetSystemUIOverlayStyle sets the colors and the icons of the system's bars.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.setSystemChromeSystemUIOverlayStyle
func (w *window) SetSystemUIOverlayStyle(style ui.SystemUIOverlayStyle) {
	w.p.host.Command(command{Op: opOverlayStyle, S: encodeOverlayStyle(style)})
}

// SetSystemUIChangeCallback sets the function that hears of the bars appearing and disappearing.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin (the system UI visibility listener)
func (w *window) SetSystemUIChangeCallback(fn func(visible bool)) {
	w.uiChange = fn
	on := int64(0)
	if fn != nil {
		on = 1
	}
	w.p.host.Command(command{Op: opUIChangeListener, A: on})
}

// PerformHapticFeedback gives the feedback with View.performHapticFeedback.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.vibrateHapticFeedback
func (w *window) PerformHapticFeedback(kind ui.HapticFeedbackKind) {
	w.p.host.Command(command{Op: opHaptic, A: int64(kind)})
}

// PlaySystemSound plays the sound with View.playSoundEffect.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.playSystemSound
func (w *window) PlaySystemSound(kind ui.SystemSoundKind) {
	w.p.host.Command(command{Op: opSound, A: int64(kind)})
}
