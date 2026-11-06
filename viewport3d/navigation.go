package viewport3d

import (
	"math"
	"reflect"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

// NavigationAction is what an input does to the camera.
//
// Flutter: no counterpart
type NavigationAction int

const (
	// NavigationActionUnspecified is the zero value: the default of the binding (SPEC §27.8).
	NavigationActionUnspecified NavigationAction = iota
	// NavigationActionNone turns the binding off: the input does nothing here, and a recognizer that only it needed is
	// left out of the arena.
	NavigationActionNone
	// NavigationActionOrbit turns the camera around the point under the pointer, or the target.
	NavigationActionOrbit
	// NavigationActionPan moves the camera across the view.
	NavigationActionPan
	// NavigationActionZoom moves the camera towards or away from what is under the pointer.
	NavigationActionZoom
	// NavigationActionFit shows the whole scene.
	NavigationActionFit
)

// NavigationDisable are bits that turn off the parts of the navigation that have no action of their own.
//
// Flutter: no counterpart
type NavigationDisable int

const (
	// NavigationDisablePinch turns off the zoom of two fingers.
	NavigationDisablePinch NavigationDisable = 1 << iota
	// NavigationDisableTwist turns off the roll of two fingers that turn.
	NavigationDisableTwist
	// NavigationDisableTwoFingerPan turns off the pan of two fingers that move together.
	NavigationDisableTwoFingerPan
	// NavigationDisableTrackpad turns off the pinch of a trackpad.
	NavigationDisableTrackpad
	// NavigationDisableKeyboard leaves out the default keys (the arrows, plus and minus); NavigationOpts.Shortcuts still
	// works.
	NavigationDisableKeyboard
)

// NavigationOpts are the optional parameters of NewNavigation. The zero value is the default navigation of SPEC §27.8:
//
//	Mouse:    left drag orbits (around the point under the pointer, else the target), middle drag or Shift with left drag
//	          pans, the wheel zooms toward the pointer, a double click fits the scene
//	Trackpad: pinch zooms
//	Touch:    one finger orbits; two fingers pan, pinch and twist
//	Keyboard: the arrows orbit, + and - zoom
//
// Flutter: no counterpart
type NavigationOpts struct {
	// LeftDrag is what a drag of the left mouse button does; the default is NavigationActionOrbit.
	LeftDrag NavigationAction
	// ShiftLeftDrag is what a drag of the left button does while Shift is down; the default is NavigationActionPan.
	ShiftLeftDrag NavigationAction
	// MiddleDrag is what a drag of the middle button does; the default is NavigationActionPan.
	MiddleDrag NavigationAction
	// RightDrag is what a drag of the right button does; the default is NavigationActionNone.
	RightDrag NavigationAction
	// Wheel is what the mouse wheel does; the default is NavigationActionZoom, and NavigationActionNone turns it off.
	Wheel NavigationAction
	// DoubleClick is what a double click or a double tap does; the default is NavigationActionFit, and
	// NavigationActionNone turns it off.
	DoubleClick NavigationAction
	// TouchDrag is what one finger (or a stylus) does; the default is NavigationActionOrbit.
	TouchDrag NavigationAction
	// Disable turns off the parts that have no action of their own.
	Disable NavigationDisable

	// OrbitSpeed is the turn, in radians, for a logical pixel of a drag; the default is 0.01 (a drag across 600 pixels turns
	// the camera by about a third of a turn).
	OrbitSpeed float64
	// PanSpeed multiplies the pan; the default is 1, which keeps the point at the target under the pointer.
	PanSpeed float64
	// ZoomSpeed multiplies the zoom of the wheel and of a drag; the default is 1.
	ZoomSpeed float64
	// PinchSpeed multiplies the zoom of a pinch: the distance changes by the pinch's ratio to this power; the default is 1.
	PinchSpeed float64
	// InvertZoom reverses the direction of the wheel.
	InvertZoom bool
	// KeyboardOrbitStep is the turn of one press of an arrow, in radians; the default is 0.1.
	KeyboardOrbitStep float64
	// KeyboardZoomStep is the factor of one press of plus or minus; the default is 1.1 (the distance is divided by it to
	// zoom in).
	KeyboardZoomStep float64

	// LockPointer locks the pointer (SPEC §27.6) while a mouse drag orbits: the cursor is hidden and the drag does not stop
	// at the edge of the screen. A platform that cannot lock it, or refuses, gets the drag without the lock.
	LockPointer bool
	// PivotAt tells the point of the world under a point of the view, in logical pixels, or false when there is none (the
	// sky): an orbit turns around it, and the wheel zooms towards it. The view's picking gives it (SPEC §27.8); without
	// it the orbit turns around the target and the zoom keeps the point of the target's plane under the pointer.
	PivotAt func(at ui.Offset) (vecmath.Vec3, bool)

	// Shortcuts replace the default keys. The intents that the navigation handles are OrbitIntent, ZoomIntent, PanIntent,
	// FitIntent and StandardViewIntent.
	Shortcuts widgets.ShortcutMap
	// Actions are added to the navigation's actions, and replace the one of an intent they handle.
	Actions widgets.ActionMap
	// FocusNode is the focus node of the keyboard; without one the navigation makes its own. A press on the view asks for
	// the focus.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the navigation is built.
	Autofocus bool
	// Duration is the length of an animated move, such as the fit of a double click; the default is 300 ms, and Some(0)
	// moves at once.
	Duration foundation.Opt[time.Duration]
	// Curve shapes an animated move; the default is animation.Curves.EaseInOut.
	Curve animation.Curve
}

// OrbitIntent turns the camera around its target by whole steps of NavigationOpts.KeyboardOrbitStep.
//
// Flutter: no counterpart
type OrbitIntent struct {
	widgets.IntentBase
	// Yaw is the number of steps counterclockwise as seen from above: the camera moves to the left when it is negative.
	Yaw float64
	// Pitch is the number of steps upwards: the camera rises when it is positive.
	Pitch float64
}

// ZoomIntent moves the camera towards the target or away from it.
//
// Flutter: no counterpart
type ZoomIntent struct {
	widgets.IntentBase
	// In zooms in when true, out when false, by one NavigationOpts.KeyboardZoomStep.
	In bool
}

// PanIntent moves the camera across the view.
//
// Flutter: no counterpart
type PanIntent struct {
	widgets.IntentBase
	// Dx is the move to the right, in logical pixels.
	Dx float64
	// Dy is the move down, in logical pixels.
	Dy float64
}

// FitIntent shows the whole scene (the camera's Bounds), with an animation.
//
// Flutter: no counterpart
type FitIntent struct{ widgets.IntentBase }

// StandardViewIntent turns the camera to a standard view, with an animation.
//
// Flutter: no counterpart
type StandardViewIntent struct {
	widgets.IntentBase
	// View is the view to go to.
	View StandardView
}

const (
	defaultOrbitSpeed = 0.01
	// wheelZoomRate is the logarithm of the factor of the distance for a logical pixel of the wheel's scroll: a notch of 100
	// pixels changes the distance by about a fifth.
	wheelZoomRate = 0.002
	// dragZoomRate is wheelZoomRate for a pixel of a drag that zooms.
	dragZoomRate         = 0.005
	defaultKeyboardOrbit = 0.1
	defaultKeyboardZoom  = 1.1
)

// Navigation is the widget that lets a user move a Camera: it recognizes the drags, the wheel, the pinch, the double click
// and the keys of NavigationOpts, which are gesture recognizers of the gesture arena (SPEC §27.8). It is made around the
// view that shows the camera's picture; it sets the camera's viewport size from its own.
//
// Flutter: no counterpart
type Navigation struct {
	widgets.StatefulWidget
	camera *Camera
	child  widgets.Widget
	o      NavigationOpts
}

// NewNavigation creates a navigation of the camera around the child.
//
// Flutter: no counterpart
func NewNavigation(camera *Camera, child widgets.Widget, opts ...NavigationOpts) *Navigation {
	return &Navigation{StatefulWidget: widgets.NewStatefulWidgetBase(), camera: camera, child: child, o: firstOpts(opts)}
}

// Camera is the camera that the navigation moves.
//
// Flutter: no counterpart
func (n *Navigation) Camera() *Camera { return n.camera }

// CreateState creates the state of the navigation.
//
// Flutter: no counterpart
func (n *Navigation) CreateState() widgets.State { return &navigationState{} }

type navigationState struct {
	widgets.StateOf[*Navigation]
	widgets.TickerProvider

	focus     *widgets.FocusNode
	ownsFocus bool
	// locked is the session whose drag holds the pointer lock, or nil.
	locked     *dragSession
	lockHandle foundation.ListenerHandle
	// animating is whether the camera's move is the navigation's: its controller uses this state's ticker.
	animating bool
	// lastDown is where the last pointer went down, in the navigation's coordinates; touchDown is the number of fingers
	// that are down, and freshTouch whether the next gesture of fingers starts from a state with none.
	lastDown   ui.Offset
	touchDown  int
	freshTouch bool
}

// InitState makes the focus node and listens to the pointer lock.
func (s *navigationState) InitState() {
	s.syncFocus()
	s.lockHandle = widgets.PointerLockOf(s.Context()).AddListener(s.lockEnded)
}

// DidUpdateWidget follows a change of the focus node.
func (s *navigationState) DidUpdateWidget(old *Navigation) { s.syncFocus() }

// Dispose stops the navigation's move and gives the pointer back.
func (s *navigationState) Dispose() {
	widgets.PointerLockOf(s.Context()).RemoveListener(s.lockHandle)
	if s.locked != nil {
		widgets.PointerLockOf(s.Context()).Unlock()
		s.locked = nil
	}
	if s.animating {
		s.Widget().camera.StopAnimation()
	}
	if s.ownsFocus {
		s.focus.Dispose()
	}
}

// syncFocus uses the focus node of the options, or makes one.
func (s *navigationState) syncFocus() {
	given := s.Widget().o.FocusNode
	switch {
	case given != nil:
		if s.ownsFocus {
			s.focus.Dispose()
			s.ownsFocus = false
		}
		s.focus = given
	case s.focus == nil || !s.ownsFocus:
		s.focus = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "Navigation"})
		s.ownsFocus = true
	}
}

// lockEnded is told when the pointer lock ends. A lock that the navigation held and the platform took back leaves the
// pointer where it really is, which is not where the drag held it: the next motion would jump, so it is skipped.
func (s *navigationState) lockEnded() {
	if s.locked != nil && !widgets.PointerLockOf(s.Context()).Locked() {
		s.locked.locked = false
		s.locked.skip = true
		s.locked = nil
	}
}

// --- what the inputs do ---------------------------------------------------------------------------------------------

func (o *NavigationOpts) orbitSpeed() float64 { return orDefault(o.OrbitSpeed, defaultOrbitSpeed) }
func (o *NavigationOpts) panSpeed() float64   { return orDefault(o.PanSpeed, 1) }
func (o *NavigationOpts) zoomSpeed() float64  { return orDefault(o.ZoomSpeed, 1) }
func (o *NavigationOpts) pinchSpeed() float64 { return orDefault(o.PinchSpeed, 1) }
func (o *NavigationOpts) orbitStep() float64 {
	return orDefault(o.KeyboardOrbitStep, defaultKeyboardOrbit)
}
func (o *NavigationOpts) zoomStep() float64 {
	return orDefault(o.KeyboardZoomStep, defaultKeyboardZoom)
}
func (o *NavigationOpts) enabled(d NavigationDisable) bool { return o.Disable&d == 0 }

func orDefault(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

// actionOf is the action of a binding, with its default for the zero value.
func actionOf(a, def NavigationAction) NavigationAction {
	if a == NavigationActionUnspecified {
		return def
	}
	return a
}

func (o *NavigationOpts) left() NavigationAction {
	return actionOf(o.LeftDrag, NavigationActionOrbit)
}
func (o *NavigationOpts) shiftLeft() NavigationAction {
	return actionOf(o.ShiftLeftDrag, NavigationActionPan)
}
func (o *NavigationOpts) middle() NavigationAction {
	return actionOf(o.MiddleDrag, NavigationActionPan)
}
func (o *NavigationOpts) right() NavigationAction {
	return actionOf(o.RightDrag, NavigationActionNone)
}
func (o *NavigationOpts) wheel() NavigationAction {
	return actionOf(o.Wheel, NavigationActionZoom)
}
func (o *NavigationOpts) doubleClick() NavigationAction {
	return actionOf(o.DoubleClick, NavigationActionFit)
}
func (o *NavigationOpts) touch() NavigationAction {
	return actionOf(o.TouchDrag, NavigationActionOrbit)
}

// apply does a drag's action for a motion of the pointer, in logical pixels, at a point of the view.
func (s *navigationState) apply(sess *dragSession, delta, at ui.Offset) {
	o := &s.Widget().o
	cam := s.Widget().camera
	switch sess.action {
	case NavigationActionOrbit:
		pivot := cam.Target()
		if sess.hasPivot {
			pivot = sess.pivot
		}
		cam.OrbitAround(pivot, -delta.Dx*o.orbitSpeed(), delta.Dy*o.orbitSpeed())
	case NavigationActionPan:
		cam.Pan(delta.Dx*o.panSpeed(), delta.Dy*o.panSpeed())
	case NavigationActionZoom:
		cam.ZoomAt(math.Exp(delta.Dy*dragZoomRate*o.zoomSpeed()), at)
	}
}

// start begins a gesture of a recognizer: the action of the mouse button it follows (nil for the recognizer of fingers),
// or the pinch of a trackpad, which every recognizer takes.
func (s *navigationState) start(sess *dragSession, mouse func() NavigationAction, d gestures.ScaleStartDetails) {
	trackpad := d.Kind.OrElse(ui.PointerDeviceKindTouch) == ui.PointerDeviceKindTrackpad
	switch {
	case trackpad || mouse == nil:
		s.startTouch(sess, d)
		sess.trackpad = trackpad
	default:
		s.startMouse(sess, mouse(), d)
	}
}

// update does an update of a recognizer's gesture.
func (s *navigationState) update(sess *dragSession, mouse bool, d gestures.ScaleUpdateDetails) {
	switch {
	case sess.trackpad:
		s.updateTrackpad(sess, d)
	case mouse:
		s.updateMouse(sess, d)
	default:
		s.updateTouch(sess, d)
	}
}

// startMouse begins a drag of a button: it stops a running move, takes the action and, for an orbit, the pivot and the lock.
func (s *navigationState) startMouse(sess *dragSession, action NavigationAction, _ gestures.ScaleStartDetails) {
	w := s.Widget()
	sess.reset()
	sess.action = action
	sess.lastFocal = s.lastDown
	w.camera.StopAnimation()
	if action != NavigationActionOrbit {
		return
	}
	if w.o.PivotAt != nil {
		sess.pivot, sess.hasPivot = w.o.PivotAt(s.lastDown)
	}
	s.lock(sess)
}

// lock locks the pointer for an orbit, when the options ask for it. A platform that cannot, or refuses, leaves the drag as
// it is.
func (s *navigationState) lock(sess *dragSession) {
	if !s.Widget().o.LockPointer || s.locked != nil {
		return
	}
	view := widgets.MaybeViewOf(s.Context())
	if view == nil {
		return
	}
	if err := widgets.PointerLockOf(s.Context()).Lock(view); err != nil {
		return
	}
	sess.locked = true
	s.locked = sess
}

// unlock gives the pointer back at the end of the drag that locked it.
func (s *navigationState) unlock(sess *dragSession) {
	if sess.locked {
		sess.locked = false
		if s.locked == sess {
			s.locked = nil
		}
		widgets.PointerLockOf(s.Context()).Unlock()
	}
}

// updateMouse does a mouse drag's action for an update of the recognizer. A locked pointer reports no motion here, since
// its position does not change: lockedMotion has it.
func (s *navigationState) updateMouse(sess *dragSession, d gestures.ScaleUpdateDetails) {
	switch {
	case sess.locked:
		return
	case sess.skip:
		sess.skip = false
		sess.lastFocal = d.LocalFocalPoint
		return
	}
	s.apply(sess, sess.moved(d.LocalFocalPoint), d.LocalFocalPoint)
}

// lockedMotion is a motion of the locked mouse, which arrives as a pointer event with a delta and the held position.
func (s *navigationState) lockedMotion(delta ui.Offset) {
	if sess := s.locked; sess != nil && sess.locked {
		s.apply(sess, delta, ui.Offset{})
	}
}

// startTouch begins the gesture of fingers (or a trackpad).
func (s *navigationState) startTouch(sess *dragSession, d gestures.ScaleStartDetails) {
	sess.reset()
	s.Widget().camera.StopAnimation()
	// A finger that went down alone starts from where it went down; the fingers that join one, or that stay when another
	// leaves, start from where they are.
	sess.lastFocal = d.LocalFocalPoint
	if s.freshTouch && d.PointerCount == 1 && d.Kind.OrElse(ui.PointerDeviceKindTouch) != ui.PointerDeviceKindTrackpad {
		sess.lastFocal = s.lastDown
	}
	s.freshTouch = false
}

// updateTouch does the gesture of fingers: one finger the action of TouchDrag; two or more pan, pinch and twist.
func (s *navigationState) updateTouch(sess *dragSession, d gestures.ScaleUpdateDetails) {
	w := s.Widget()
	cam := w.camera
	delta := sess.moved(d.LocalFocalPoint)
	if d.PointerCount < 2 {
		sess.action = w.o.touch()
		s.apply(sess, delta, d.LocalFocalPoint)
		return
	}
	if w.o.enabled(NavigationDisableTwoFingerPan) {
		cam.Pan(delta.Dx*w.o.panSpeed(), delta.Dy*w.o.panSpeed())
	}
	if w.o.enabled(NavigationDisablePinch) && d.Scale > 0 && sess.lastScale > 0 {
		cam.ZoomAt(math.Pow(sess.lastScale/d.Scale, w.o.pinchSpeed()), d.LocalFocalPoint)
	}
	if w.o.enabled(NavigationDisableTwist) {
		cam.Roll(-(d.Rotation - sess.lastRotation))
	}
	sess.lastScale, sess.lastRotation = d.Scale, d.Rotation
}

// updateTrackpad zooms by the pinch of a trackpad.
func (s *navigationState) updateTrackpad(sess *dragSession, d gestures.ScaleUpdateDetails) {
	w := s.Widget()
	if w.o.enabled(NavigationDisableTrackpad) && d.Scale > 0 && sess.lastScale > 0 {
		w.camera.ZoomAt(math.Pow(sess.lastScale/d.Scale, w.o.pinchSpeed()), d.LocalFocalPoint)
	}
	sess.lastScale = d.Scale
}

// wheel zooms by a turn of the wheel at a point of the view.
func (s *navigationState) wheel(e *gestures.PointerScrollEvent) {
	w := s.Widget()
	w.camera.StopAnimation()
	dy := e.ScrollDelta().Dy
	if w.o.InvertZoom {
		dy = -dy
	}
	factor := math.Exp(dy * wheelZoomRate * w.o.zoomSpeed())
	if w.o.PivotAt != nil {
		if p, ok := w.o.PivotAt(e.LocalPosition()); ok {
			w.camera.ZoomToward(factor, p)
			return
		}
	}
	w.camera.ZoomAt(factor, e.LocalPosition())
}

// pointerDown remembers where the pointer went down, and asks for the focus of the keys.
func (s *navigationState) pointerDown(e *gestures.PointerDownEvent) {
	s.lastDown = e.LocalPosition()
	if e.Kind() == ui.PointerDeviceKindTouch || e.Kind() == ui.PointerDeviceKindStylus {
		if s.touchDown == 0 {
			s.freshTouch = true
		}
		s.touchDown++
	}
	s.focus.RequestFocus()
}

func (s *navigationState) pointerUp(e *gestures.PointerUpEvent) { s.touchEnded(e.Kind()) }

func (s *navigationState) pointerCancel(e *gestures.PointerCancelEvent) { s.touchEnded(e.Kind()) }

// touchEnded counts a finger that left.
func (s *navigationState) touchEnded(kind ui.PointerDeviceKind) {
	if (kind == ui.PointerDeviceKindTouch || kind == ui.PointerDeviceKindStylus) && s.touchDown > 0 {
		s.touchDown--
	}
}

// pointerSignal registers the wheel with the resolver, so that the deepest widget that wants the scroll gets it (a
// Scrollable above the view does not also scroll).
func (s *navigationState) pointerSignal(event gestures.PointerSignalEvent) {
	scroll, ok := event.(*gestures.PointerScrollEvent)
	if !ok || s.Widget().o.wheel() == NavigationActionNone {
		return
	}
	widgets.WidgetsBindingOf(s.Context()).Gestures().PointerSignalResolver.Register(scroll, func(gestures.PointerSignalEvent) {
		s.wheel(scroll)
		scroll.Respond(true)
	})
}

// animate moves the camera to a pose with the options' duration and curve.
func (s *navigationState) animate(pose CameraPose) {
	w := s.Widget()
	s.animating = true
	w.camera.AnimateTo(pose, &s.TickerProvider, AnimateOpts{Duration: w.o.Duration, Curve: w.o.Curve,
		OnEnd: func(bool) { s.animating = false }})
}

// fit shows the whole scene.
func (s *navigationState) fit() {
	cam := s.Widget().camera
	if cam.Bounds().IsEmpty() {
		return
	}
	s.animate(cam.FitPose(cam.Bounds()))
}

// --- the actions of the keys --------------------------------------------------------------------------------------

func (s *navigationState) actions() widgets.ActionMap {
	w := s.Widget()
	cam := w.camera
	m := widgets.ActionMap{
		widgets.IntentTypeOf[OrbitIntent](): widgets.NewCallbackAction(func(i OrbitIntent) any {
			cam.StopAnimation()
			cam.Orbit(i.Yaw*w.o.orbitStep(), i.Pitch*w.o.orbitStep())
			return nil
		}),
		widgets.IntentTypeOf[ZoomIntent](): widgets.NewCallbackAction(func(i ZoomIntent) any {
			cam.StopAnimation()
			f := w.o.zoomStep()
			if i.In {
				f = 1 / f
			}
			cam.Zoom(f)
			return nil
		}),
		widgets.IntentTypeOf[PanIntent](): widgets.NewCallbackAction(func(i PanIntent) any {
			cam.StopAnimation()
			cam.Pan(i.Dx, i.Dy)
			return nil
		}),
		widgets.IntentTypeOf[FitIntent](): widgets.NewCallbackAction(func(FitIntent) any {
			s.fit()
			return nil
		}),
		widgets.IntentTypeOf[StandardViewIntent](): widgets.NewCallbackAction(func(i StandardViewIntent) any {
			s.animate(cam.StandardViewPose(i.View))
			return nil
		}),
	}
	for t, a := range w.o.Actions {
		m[t] = a
	}
	return m
}

// defaultShortcuts are the keys of the navigation: the arrows move the camera in their direction, plus and minus zoom.
func defaultShortcuts() widgets.ShortcutMap {
	key := func(k services.LogicalKeyboardKey, i widgets.Intent) widgets.ShortcutEntry {
		return widgets.ShortcutEntry{Activator: widgets.NewSingleActivator(k), Intent: i}
	}
	char := func(c string, i widgets.Intent) widgets.ShortcutEntry {
		return widgets.ShortcutEntry{Activator: widgets.NewCharacterActivator(c), Intent: i}
	}
	return widgets.ShortcutMap{
		key(services.LogicalKeyboardKeyArrowLeft, OrbitIntent{Yaw: -1}),
		key(services.LogicalKeyboardKeyArrowRight, OrbitIntent{Yaw: 1}),
		key(services.LogicalKeyboardKeyArrowUp, OrbitIntent{Pitch: 1}),
		key(services.LogicalKeyboardKeyArrowDown, OrbitIntent{Pitch: -1}),
		char("+", ZoomIntent{In: true}),
		char("=", ZoomIntent{In: true}),
		key(services.LogicalKeyboardKeyNumpadAdd, ZoomIntent{In: true}),
		char("-", ZoomIntent{}),
		key(services.LogicalKeyboardKeyNumpadSubtract, ZoomIntent{}),
	}
}

// --- building -------------------------------------------------------------------------------------------------------

// Build makes the recognizers, the pointer listener and the keys around the child.
func (s *navigationState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := &w.o
	binding := widgets.WidgetsBindingOf(bc).Gestures()
	factories := map[reflect.Type]widgets.GestureRecognizerFactory{}
	add := func(f widgets.GestureRecognizerFactory) { factories[f.RecognizerType()] = f }

	if o.left() != NavigationActionNone || o.shiftLeft() != NavigationActionNone {
		add(scaleFactory(binding, mouseDevices, buttonOnly(gestures.PrimaryButton), func(r *scaleRecognizer[leftRole]) {
			r.OnStart = func(d gestures.ScaleStartDetails) {
				s.start(&r.session, func() NavigationAction {
					if widgets.WidgetsBindingOf(bc).Keyboard().IsShiftPressed() {
						return o.shiftLeft()
					}
					return o.left()
				}, d)
			}
			r.OnUpdate = func(d gestures.ScaleUpdateDetails) { s.update(&r.session, true, d) }
			r.OnEnd = func(gestures.ScaleEndDetails) { s.unlock(&r.session) }
		}))
	}
	if o.middle() != NavigationActionNone {
		add(scaleFactory(binding, mouseDevices, buttonOnly(gestures.MiddleMouseButton), func(r *scaleRecognizer[middleRole]) {
			r.OnStart = func(d gestures.ScaleStartDetails) { s.start(&r.session, o.middle, d) }
			r.OnUpdate = func(d gestures.ScaleUpdateDetails) { s.update(&r.session, true, d) }
			r.OnEnd = func(gestures.ScaleEndDetails) { s.unlock(&r.session) }
		}))
	}
	if o.right() != NavigationActionNone {
		add(scaleFactory(binding, mouseDevices, buttonOnly(gestures.SecondaryButton), func(r *scaleRecognizer[rightRole]) {
			r.OnStart = func(d gestures.ScaleStartDetails) { s.start(&r.session, o.right, d) }
			r.OnUpdate = func(d gestures.ScaleUpdateDetails) { s.update(&r.session, true, d) }
			r.OnEnd = func(gestures.ScaleEndDetails) { s.unlock(&r.session) }
		}))
	}
	// Fingers: one orbits (or whatever TouchDrag says); two or more pan, pinch and twist. A scale recognizer takes every
	// trackpad gesture whatever its devices (as Flutter's does), so each of them handles the pinch of a trackpad the
	// same way, and whichever wins the arena does it.
	add(scaleFactory(binding, touchDevices, nil, func(r *scaleRecognizer[touchRole]) {
		r.OnStart = func(d gestures.ScaleStartDetails) { s.start(&r.session, nil, d) }
		r.OnUpdate = func(d gestures.ScaleUpdateDetails) { s.update(&r.session, false, d) }
	}))
	if o.doubleClick() == NavigationActionFit {
		add(doubleClickFactory(binding, func(r *doubleClickRecognizer) {
			r.OnDoubleTap = s.fit
		}))
	}

	// The camera's viewport is the navigation's size: a layout builder tells it.
	sized := widgets.NewLayoutBuilder(func(_ widgets.BuildContext, c rendering.BoxConstraints) widgets.Widget {
		if c.HasBoundedWidth() && c.HasBoundedHeight() {
			w.camera.SetViewportSize(c.Biggest())
		}
		return w.child
	})
	listener := widgets.NewListener(sized, widgets.ListenerOpts{
		Behavior:        rendering.HitTestBehaviorOpaque,
		OnPointerDown:   s.pointerDown,
		OnPointerUp:     s.pointerUp,
		OnPointerCancel: s.pointerCancel,
		OnPointerSignal: s.pointerSignal,
		OnPointerMove:   func(e *gestures.PointerMoveEvent) { s.lockedMotion(e.Delta()) },
		OnPointerHover:  func(e *gestures.PointerHoverEvent) { s.lockedMotion(e.Delta()) },
	})
	detector := widgets.NewRawGestureDetector(listener, widgets.RawGestureDetectorOpts{
		Gestures: factories, Behavior: rendering.HitTestBehaviorOpaque,
	})

	shortcuts := o.Shortcuts
	if shortcuts == nil && o.enabled(NavigationDisableKeyboard) {
		shortcuts = defaultShortcuts()
	}
	return widgets.NewFocusableActionDetector(detector, widgets.FocusableActionDetectorOpts{
		FocusNode: s.focus, Autofocus: o.Autofocus, Shortcuts: shortcuts, Actions: s.actions(),
	})
}
