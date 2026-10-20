package ui

import "github.com/rm4n0s/errors"

// ViewID identifies a View: one top-level surface that the engine renders
// into, such as a window. The first view is the implicit view and has ID 0.
//
// Flutter: ui/window.dart FlutterView.viewId
type ViewID int64

// ImplicitViewID is the ID of the implicit view, the first view that exists
// when the app starts.
//
// Flutter: ui/window.dart PlatformDispatcher.implicitView
const ImplicitViewID ViewID = 0

// ViewConfiguration is the size, scale and system areas of a view, as the
// platform reports them. The engine hands one to the dispatcher whenever it
// changes.
//
// Flutter: ui/window.dart _ViewConfiguration
type ViewConfiguration struct {
	// PhysicalSize is the size of the view in physical pixels.
	PhysicalSize Size
	// DevicePixelRatio is the number of physical pixels per logical pixel; 0
	// means 1.
	DevicePixelRatio float64
	// ViewPadding is the area of the view that the system covers or that is
	// not usable (an unsafe area), in physical pixels.
	ViewPadding ViewPadding
	// ViewInsets is the area covered by the system UI that the app can still
	// draw below, such as an on-screen keyboard.
	ViewInsets ViewPadding
	// SystemGestureInsets is the area where the system handles gestures.
	SystemGestureInsets ViewPadding
	// Padding is ViewPadding minus ViewInsets: the area the app should avoid.
	Padding ViewPadding
	// PhysicalConstraints limits the size of the view; the zero value allows
	// only PhysicalSize.
	PhysicalConstraints ViewConstraints
	// Display is the display the view is on.
	Display Display
	// GestureSettings are the platform's gesture thresholds.
	GestureSettings GestureSettings
	// Window is what the platform says about the decorations and the state of the
	// window of the view.
	Window WindowInfo
}

// GestureSettings are the gesture thresholds the platform sets, in physical
// pixels. A value of 0 means the platform gives none, and the framework uses
// its own default.
//
// Flutter: ui/window.dart GestureSettings
type GestureSettings struct {
	// PhysicalTouchSlop is how far a touch may move before it is a drag.
	PhysicalTouchSlop float64
	// PhysicalDoubleTapSlop is how far apart two taps may be to make a double
	// tap.
	PhysicalDoubleTapSlop float64
}

// View is one top-level surface that the app renders into, such as a window.
// The dispatcher makes views; the platform sets their configuration.
//
// Flutter: ui/window.dart FlutterView
type View struct {
	d         *PlatformDispatcher
	id        ViewID
	cfg       ViewConfiguration
	destroyed bool
}

// Window is what the platform says about the decorations and the state of the
// window that the view shows: who draws the title bar, what the user may do with
// the window, and whether it is maximized or has the focus.
//
// Flutter: no counterpart
func (v *View) Window() WindowInfo { return v.cfg.Window }

// PerformWindowAction asks the window of the view to do the action: the way a
// title bar that the app draws moves, resizes, minimizes, maximizes or closes it.
//
// Flutter: no counterpart
func (v *View) PerformWindowAction(action WindowAction) { v.d.performWindowAction(v.id, action) }

// Destroyed is whether the window of the view was destroyed. Rendering into a
// destroyed view does nothing.
//
// Flutter: no counterpart
func (v *View) Destroyed() bool { return v.destroyed }

// ViewID is the ID of the view.
//
// Flutter: ui/window.dart FlutterView.viewId
func (v *View) ViewID() ViewID { return v.id }

// PlatformDispatcher is the dispatcher that owns the view.
//
// Flutter: ui/window.dart FlutterView.platformDispatcher
func (v *View) PlatformDispatcher() *PlatformDispatcher { return v.d }

// PhysicalSize is the size of the view in physical pixels.
//
// Flutter: ui/window.dart FlutterView.physicalSize
func (v *View) PhysicalSize() Size { return v.cfg.PhysicalSize }

// DevicePixelRatio is the number of physical pixels per logical pixel.
//
// Flutter: ui/window.dart FlutterView.devicePixelRatio
func (v *View) DevicePixelRatio() float64 {
	if v.cfg.DevicePixelRatio == 0 {
		return 1
	}
	return v.cfg.DevicePixelRatio
}

// PhysicalConstraints are the limits on the size of the view, in physical
// pixels; tight around PhysicalSize unless the platform says otherwise.
//
// Flutter: ui/window.dart FlutterView.physicalConstraints
func (v *View) PhysicalConstraints() ViewConstraints {
	if v.cfg.PhysicalConstraints == (ViewConstraints{}) {
		return NewViewConstraintsTight(v.cfg.PhysicalSize)
	}
	return v.cfg.PhysicalConstraints
}

// GestureSettings are the platform's gesture thresholds, in physical pixels.
//
// Flutter: ui/window.dart FlutterView.gestureSettings
func (v *View) GestureSettings() GestureSettings { return v.cfg.GestureSettings }

// ViewPadding is the area of the view that the system covers or that is not
// usable, in physical pixels.
//
// Flutter: ui/window.dart FlutterView.viewPadding
func (v *View) ViewPadding() ViewPadding { return v.cfg.ViewPadding }

// ViewInsets is the area covered by system UI the app can draw below, in
// physical pixels.
//
// Flutter: ui/window.dart FlutterView.viewInsets
func (v *View) ViewInsets() ViewPadding { return v.cfg.ViewInsets }

// SystemGestureInsets is the area where the system handles gestures, in
// physical pixels.
//
// Flutter: ui/window.dart FlutterView.systemGestureInsets
func (v *View) SystemGestureInsets() ViewPadding { return v.cfg.SystemGestureInsets }

// Padding is the area the app should avoid: ViewPadding minus ViewInsets, in
// physical pixels.
//
// Flutter: ui/window.dart FlutterView.padding
func (v *View) Padding() ViewPadding { return v.cfg.Padding }

// Display is the display the view is on.
//
// Flutter: ui/window.dart FlutterView.display
func (v *View) Display() Display { return v.cfg.Display }

// Render draws the scene into the view. Call it from OnDrawFrame, at most
// once per frame (the last scene wins). It returns the renderer's error, such
// as a lost GPU context; a dispatcher without an engine behind it returns
// PlatformHostMissing.
//
// Flutter: ui/window.dart FlutterView.render
func (v *View) Render(scene *Scene) error {
	if v.destroyed {
		return nil
	}
	if v.d.host == nil {
		return errors.New("PlatformHostMissing", "the dispatcher has no engine to render with", "view", int64(v.id))
	}
	return v.d.host.RenderView(v.id, scene)
}

// UpdateSemantics gives the engine the changes to the semantics tree of the
// view. A host that does not take part in semantics ignores it.
//
// Flutter: ui/window.dart FlutterView.updateSemantics
func (v *View) UpdateSemantics(update *SemanticsUpdate) {
	if h, ok := v.d.host.(SemanticsHost); ok {
		h.UpdateSemantics(v.id, update)
	}
}
