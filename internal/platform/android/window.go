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

// SetState does nothing; fullscreen (immersive mode) comes with step 25.7.
func (w *window) SetState(platform.WindowState) {}

// SetHidden returns false: an activity cannot hide its window.
func (w *window) SetHidden(bool) bool { return false }

// SetCursor does nothing until step 25.6.
func (w *window) SetCursor(platform.MouseCursor) {}

// SetTextInput does nothing until step 25.8.
func (w *window) SetTextInput(ui.TextInputState) {}

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

// KeepScreenOn fails with IdleInhibitUnavailable until step 25.7.
func (w *window) KeepScreenOn(bool) error {
	return errors.New(tagIdleInhibitUnavailable, "keeping the screen on is not supported yet (step 25.7)")
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragUnsupported        = "DragUnsupported"
	tagIdleInhibitUnavailable = "IdleInhibitUnavailable"
)
