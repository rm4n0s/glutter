package widgets

import (
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// GestureRecognizerFactory is the interface for a factory that makes and
// configures a gesture recognizer of one type.
//
// Flutter: widgets/gesture_detector.dart GestureRecognizerFactory
type GestureRecognizerFactory interface {
	// Constructor makes a new recognizer.
	//
	// Flutter: widgets/gesture_detector.dart GestureRecognizerFactory.constructor
	Constructor() gestures.GestureRecognizer
	// Initializer configures the recognizer, which Constructor made or a
	// previous build did.
	//
	// Flutter: widgets/gesture_detector.dart GestureRecognizerFactory.initializer
	Initializer(instance gestures.GestureRecognizer)
	// RecognizerType is the type of the recognizers that the factory makes: the
	// key it is registered under.
	RecognizerType() reflect.Type
}

// GestureRecognizerFactoryWithHandlers is a GestureRecognizerFactory that calls
// two functions to make and configure a recognizer of type T.
//
// Flutter: widgets/gesture_detector.dart GestureRecognizerFactoryWithHandlers
type GestureRecognizerFactoryWithHandlers[T gestures.GestureRecognizer] struct {
	constructor func() T
	initializer func(instance T)
}

// NewGestureRecognizerFactoryWithHandlers creates the factory.
//
// Flutter: widgets/gesture_detector.dart GestureRecognizerFactoryWithHandlers.new
func NewGestureRecognizerFactoryWithHandlers[T gestures.GestureRecognizer](constructor func() T, initializer func(instance T)) *GestureRecognizerFactoryWithHandlers[T] {
	return &GestureRecognizerFactoryWithHandlers[T]{constructor, initializer}
}

// Constructor makes a recognizer.
//
// Flutter: widgets/gesture_detector.dart GestureRecognizerFactoryWithHandlers.constructor
func (f *GestureRecognizerFactoryWithHandlers[T]) Constructor() gestures.GestureRecognizer {
	return f.constructor()
}

// Initializer configures a recognizer.
//
// Flutter: widgets/gesture_detector.dart GestureRecognizerFactoryWithHandlers.initializer
func (f *GestureRecognizerFactoryWithHandlers[T]) Initializer(instance gestures.GestureRecognizer) {
	f.initializer(instance.(T))
}

// RecognizerType is the type T.
//
// Flutter: widgets/gesture_detector.dart GestureRecognizerFactory._debugAssertTypeMatches
func (f *GestureRecognizerFactoryWithHandlers[T]) RecognizerType() reflect.Type {
	return reflect.TypeFor[T]()
}

// GestureDetectorOpts are the optional parameters of NewGestureDetector.
//
// Flutter: widgets/gesture_detector.dart GestureDetector.new
type GestureDetectorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key

	// OnTapDown is called when a pointer that might cause a tap has contacted
	// the screen at a particular location.
	OnTapDown gestures.GestureTapDownCallback
	// OnTapUp is called when a pointer that will trigger a tap has stopped
	// contacting the screen.
	OnTapUp gestures.GestureTapUpCallback
	// OnTap is called when a tap with a primary button has occurred.
	OnTap gestures.GestureTapCallback
	// OnTapMove is called when the pointer moves while a tap is possible.
	OnTapMove gestures.GestureTapMoveCallback
	// OnTapCancel is called when the pointer that previously triggered OnTapDown
	// will not end up causing a tap.
	OnTapCancel gestures.GestureTapCancelCallback
	// OnSecondaryTap is called when a tap with a secondary button has occurred.
	OnSecondaryTap gestures.GestureTapCallback
	// OnSecondaryTapDown is called when a pointer that might cause a tap with a
	// secondary button has contacted the screen.
	OnSecondaryTapDown gestures.GestureTapDownCallback
	// OnSecondaryTapUp is called when a pointer that will trigger a tap with a
	// secondary button has stopped contacting the screen.
	OnSecondaryTapUp gestures.GestureTapUpCallback
	// OnSecondaryTapCancel is called when the pointer that previously triggered
	// OnSecondaryTapDown will not end up causing a tap.
	OnSecondaryTapCancel gestures.GestureTapCancelCallback
	// OnTertiaryTapDown is called when a pointer that might cause a tap with a
	// tertiary button has contacted the screen.
	OnTertiaryTapDown gestures.GestureTapDownCallback
	// OnTertiaryTapUp is called when a pointer that will trigger a tap with a
	// tertiary button has stopped contacting the screen.
	OnTertiaryTapUp gestures.GestureTapUpCallback
	// OnTertiaryTapCancel is called when the pointer that previously triggered
	// OnTertiaryTapDown will not end up causing a tap.
	OnTertiaryTapCancel gestures.GestureTapCancelCallback

	// OnDoubleTapDown is called when a pointer that might cause a double tap has
	// contacted the screen.
	OnDoubleTapDown gestures.GestureTapDownCallback
	// OnDoubleTap is called when the user has tapped the screen at the same
	// location twice in quick succession.
	OnDoubleTap gestures.GestureDoubleTapCallback
	// OnDoubleTapCancel is called when the pointer that previously triggered
	// OnDoubleTapDown will not end up causing a double tap.
	OnDoubleTapCancel gestures.GestureTapCancelCallback

	// OnLongPressDown is called when a pointer that might cause a long press has
	// contacted the screen.
	OnLongPressDown gestures.GestureLongPressDownCallback
	// OnLongPressCancel is called when the pointer that previously triggered
	// OnLongPressDown will not end up causing a long press.
	OnLongPressCancel gestures.GestureLongPressCancelCallback
	// OnLongPress is called when a long press gesture with a primary button has
	// been recognized.
	OnLongPress gestures.GestureLongPressCallback
	// OnLongPressStart is called when a long press gesture with a primary button
	// has been recognized.
	OnLongPressStart gestures.GestureLongPressStartCallback
	// OnLongPressMoveUpdate is called when moving after the long press.
	OnLongPressMoveUpdate gestures.GestureLongPressMoveUpdateCallback
	// OnLongPressUp is called when the pointer is lifted after the long press.
	OnLongPressUp gestures.GestureLongPressUpCallback
	// OnLongPressEnd is called when the pointer is lifted after the long press.
	OnLongPressEnd gestures.GestureLongPressEndCallback
	// OnSecondaryLongPressDown is OnLongPressDown for the secondary button.
	OnSecondaryLongPressDown gestures.GestureLongPressDownCallback
	// OnSecondaryLongPressCancel is OnLongPressCancel for the secondary button.
	OnSecondaryLongPressCancel gestures.GestureLongPressCancelCallback
	// OnSecondaryLongPress is OnLongPress for the secondary button.
	OnSecondaryLongPress gestures.GestureLongPressCallback
	// OnSecondaryLongPressStart is OnLongPressStart for the secondary button.
	OnSecondaryLongPressStart gestures.GestureLongPressStartCallback
	// OnSecondaryLongPressMoveUpdate is OnLongPressMoveUpdate for the secondary
	// button.
	OnSecondaryLongPressMoveUpdate gestures.GestureLongPressMoveUpdateCallback
	// OnSecondaryLongPressUp is OnLongPressUp for the secondary button.
	OnSecondaryLongPressUp gestures.GestureLongPressUpCallback
	// OnSecondaryLongPressEnd is OnLongPressEnd for the secondary button.
	OnSecondaryLongPressEnd gestures.GestureLongPressEndCallback
	// OnTertiaryLongPressDown is OnLongPressDown for the tertiary button.
	OnTertiaryLongPressDown gestures.GestureLongPressDownCallback
	// OnTertiaryLongPressCancel is OnLongPressCancel for the tertiary button.
	OnTertiaryLongPressCancel gestures.GestureLongPressCancelCallback
	// OnTertiaryLongPress is OnLongPress for the tertiary button.
	OnTertiaryLongPress gestures.GestureLongPressCallback
	// OnTertiaryLongPressStart is OnLongPressStart for the tertiary button.
	OnTertiaryLongPressStart gestures.GestureLongPressStartCallback
	// OnTertiaryLongPressMoveUpdate is OnLongPressMoveUpdate for the tertiary
	// button.
	OnTertiaryLongPressMoveUpdate gestures.GestureLongPressMoveUpdateCallback
	// OnTertiaryLongPressUp is OnLongPressUp for the tertiary button.
	OnTertiaryLongPressUp gestures.GestureLongPressUpCallback
	// OnTertiaryLongPressEnd is OnLongPressEnd for the tertiary button.
	OnTertiaryLongPressEnd gestures.GestureLongPressEndCallback

	// OnVerticalDragDown is called when a pointer has contacted the screen and
	// might begin to move vertically.
	OnVerticalDragDown gestures.GestureDragDownCallback
	// OnVerticalDragStart is called when a pointer has contacted the screen and
	// has begun to move vertically.
	OnVerticalDragStart gestures.GestureDragStartCallback
	// OnVerticalDragUpdate is called when a pointer that is in contact with the
	// screen and moving vertically has moved in the vertical direction.
	OnVerticalDragUpdate gestures.GestureDragUpdateCallback
	// OnVerticalDragEnd is called when a pointer that was previously in contact
	// with the screen and moving vertically is no longer in contact.
	OnVerticalDragEnd gestures.GestureDragEndCallback
	// OnVerticalDragCancel is called when the pointer that previously triggered
	// OnVerticalDragDown did not complete.
	OnVerticalDragCancel gestures.GestureDragCancelCallback
	// OnHorizontalDragDown is called when a pointer has contacted the screen and
	// might begin to move horizontally.
	OnHorizontalDragDown gestures.GestureDragDownCallback
	// OnHorizontalDragStart is called when a pointer has contacted the screen
	// and has begun to move horizontally.
	OnHorizontalDragStart gestures.GestureDragStartCallback
	// OnHorizontalDragUpdate is called when a pointer that is in contact with the
	// screen and moving horizontally has moved in the horizontal direction.
	OnHorizontalDragUpdate gestures.GestureDragUpdateCallback
	// OnHorizontalDragEnd is called when a pointer that was previously in contact
	// with the screen and moving horizontally is no longer in contact.
	OnHorizontalDragEnd gestures.GestureDragEndCallback
	// OnHorizontalDragCancel is called when the pointer that previously triggered
	// OnHorizontalDragDown did not complete.
	OnHorizontalDragCancel gestures.GestureDragCancelCallback
	// OnPanDown is called when a pointer has contacted the screen and might begin
	// to move.
	OnPanDown gestures.GestureDragDownCallback
	// OnPanStart is called when a pointer has contacted the screen and has begun
	// to move.
	OnPanStart gestures.GestureDragStartCallback
	// OnPanUpdate is called when a pointer that is in contact with the screen and
	// moving has moved again.
	OnPanUpdate gestures.GestureDragUpdateCallback
	// OnPanEnd is called when a pointer that was previously in contact with the
	// screen and moving is no longer in contact.
	OnPanEnd gestures.GestureDragEndCallback
	// OnPanCancel is called when the pointer that previously triggered OnPanDown
	// did not complete.
	OnPanCancel gestures.GestureDragCancelCallback

	// OnScaleStart is called when the pointers in contact with the screen have
	// established a focal point and initial scale of 1.0.
	OnScaleStart gestures.GestureScaleStartCallback
	// OnScaleUpdate is called when the pointers in contact with the screen have
	// indicated a new focal point and/or scale.
	OnScaleUpdate gestures.GestureScaleUpdateCallback
	// OnScaleEnd is called when the pointers are no longer in contact with the
	// screen.
	OnScaleEnd gestures.GestureScaleEndCallback

	// Behavior is how this gesture detector behaves during hit testing; the
	// default is deferToChild when there is a child, and translucent otherwise.
	Behavior rendering.HitTestBehavior
	// ExcludeFromSemantics is whether to exclude these gestures from the
	// semantics tree.
	ExcludeFromSemantics bool
	// DragStartBehavior is how the drags start; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// TrackpadScrollCausesScale is whether scrolling up/down on a trackpad
	// scales the child.
	TrackpadScrollCausesScale bool
	// TrackpadScrollToScaleFactor is how trackpad scrolling turns into scaling.
	TrackpadScrollToScaleFactor foundation.Opt[ui.Offset]
	// SupportedDevices are the kinds of pointer that trigger the gestures; nil
	// for all.
	SupportedDevices []ui.PointerDeviceKind
}

// GestureDetector is a widget that detects gestures.
//
// Deviation: there is no force press recognizer yet, and the multitouch drag
// strategy is the recognizers' default instead of the one of a
// ScrollConfiguration (SPEC §12.10).
//
// Flutter: widgets/gesture_detector.dart GestureDetector
type GestureDetector struct {
	StatelessWidget
	child Widget
	o     GestureDetectorOpts
}

// NewGestureDetector creates the widget; the child may be nil. In debug builds
// it panics with the tag GestureDetectorConflict when there are both pan and
// scale callbacks, or vertical, horizontal and pan or scale callbacks together.
//
// Flutter: widgets/gesture_detector.dart GestureDetector.new
func NewGestureDetector(child Widget, opts ...GestureDetectorOpts) *GestureDetector {
	o := firstOpts(opts)
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	if foundation.DebugMode {
		haveVertical := o.OnVerticalDragStart != nil || o.OnVerticalDragUpdate != nil || o.OnVerticalDragEnd != nil
		haveHorizontal := o.OnHorizontalDragStart != nil || o.OnHorizontalDragUpdate != nil || o.OnHorizontalDragEnd != nil
		havePan := o.OnPanStart != nil || o.OnPanUpdate != nil || o.OnPanEnd != nil
		haveScale := o.OnScaleStart != nil || o.OnScaleUpdate != nil || o.OnScaleEnd != nil
		if havePan || haveScale {
			if havePan && haveScale {
				panic(errors.New("GestureDetectorConflict", "having both a pan gesture recognizer and a scale gesture recognizer is redundant; scale is a superset of pan; just use the scale gesture recognizer"))
			}
			recognizer := "scale"
			if havePan {
				recognizer = "pan"
			}
			if haveVertical && haveHorizontal {
				panic(errors.New("GestureDetectorConflict", "simultaneously having a vertical drag gesture recognizer, a horizontal drag gesture recognizer, and a "+recognizer+
					" gesture recognizer will result in the "+recognizer+" gesture recognizer being ignored, since the other two will catch all drags", "recognizer", recognizer))
			}
		}
	}
	return &GestureDetector{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, o}
}

// Child is the widget below this one in the tree, or nil.
//
// Flutter: widgets/gesture_detector.dart GestureDetector.child
func (w *GestureDetector) Child() Widget { return w.child }

// Build makes a RawGestureDetector with a recognizer for each kind of gesture
// that has a callback.
//
// Flutter: widgets/gesture_detector.dart GestureDetector.build
func (w *GestureDetector) Build(bc BuildContext) Widget {
	o := w.o
	binding := WidgetsBindingOf(bc).Gestures()
	settings := foundation.None[gestures.DeviceGestureSettings]()
	if s, ok := MediaQueryMaybeGestureSettingsOf(bc); ok {
		settings = foundation.Some(s)
	}
	factories := map[reflect.Type]GestureRecognizerFactory{}
	add := func(f GestureRecognizerFactory) { factories[f.RecognizerType()] = f }
	configure := func(r gestures.GestureRecognizer) {
		b := r.Base()
		b.GestureSettings = settings
		b.SupportedDevices = o.SupportedDevices
	}

	if o.OnTapDown != nil || o.OnTapUp != nil || o.OnTap != nil || o.OnTapCancel != nil || o.OnSecondaryTap != nil || o.OnSecondaryTapDown != nil ||
		o.OnSecondaryTapUp != nil || o.OnSecondaryTapCancel != nil || o.OnTertiaryTapDown != nil || o.OnTertiaryTapUp != nil || o.OnTertiaryTapCancel != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.TapGestureRecognizer {
				return gestures.NewTapGestureRecognizer(binding, gestures.TapGestureRecognizerOpts{DebugOwner: w, SupportedDevices: o.SupportedDevices})
			},
			func(r *gestures.TapGestureRecognizer) {
				r.OnTapDown, r.OnTapUp, r.OnTap, r.OnTapMove, r.OnTapCancel = o.OnTapDown, o.OnTapUp, o.OnTap, o.OnTapMove, o.OnTapCancel
				r.OnSecondaryTap, r.OnSecondaryTapDown, r.OnSecondaryTapUp, r.OnSecondaryTapCancel =
					o.OnSecondaryTap, o.OnSecondaryTapDown, o.OnSecondaryTapUp, o.OnSecondaryTapCancel
				r.OnTertiaryTapDown, r.OnTertiaryTapUp, r.OnTertiaryTapCancel = o.OnTertiaryTapDown, o.OnTertiaryTapUp, o.OnTertiaryTapCancel
				configure(r)
			}))
	}
	if o.OnDoubleTap != nil || o.OnDoubleTapDown != nil || o.OnDoubleTapCancel != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.DoubleTapGestureRecognizer {
				return gestures.NewDoubleTapGestureRecognizer(binding, gestures.DoubleTapGestureRecognizerOpts{DebugOwner: w, SupportedDevices: o.SupportedDevices})
			},
			func(r *gestures.DoubleTapGestureRecognizer) {
				r.OnDoubleTapDown, r.OnDoubleTap, r.OnDoubleTapCancel = o.OnDoubleTapDown, o.OnDoubleTap, o.OnDoubleTapCancel
				configure(r)
			}))
	}
	if o.OnLongPressDown != nil || o.OnLongPressCancel != nil || o.OnLongPress != nil || o.OnLongPressStart != nil || o.OnLongPressMoveUpdate != nil ||
		o.OnLongPressUp != nil || o.OnLongPressEnd != nil || o.OnSecondaryLongPressDown != nil || o.OnSecondaryLongPressCancel != nil ||
		o.OnSecondaryLongPress != nil || o.OnSecondaryLongPressStart != nil || o.OnSecondaryLongPressMoveUpdate != nil || o.OnSecondaryLongPressUp != nil ||
		o.OnSecondaryLongPressEnd != nil || o.OnTertiaryLongPressDown != nil || o.OnTertiaryLongPressCancel != nil || o.OnTertiaryLongPress != nil ||
		o.OnTertiaryLongPressStart != nil || o.OnTertiaryLongPressMoveUpdate != nil || o.OnTertiaryLongPressUp != nil || o.OnTertiaryLongPressEnd != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.LongPressGestureRecognizer {
				return gestures.NewLongPressGestureRecognizer(binding, gestures.LongPressGestureRecognizerOpts{DebugOwner: w, SupportedDevices: o.SupportedDevices})
			},
			func(r *gestures.LongPressGestureRecognizer) {
				r.OnLongPressDown, r.OnLongPressCancel, r.OnLongPress, r.OnLongPressStart = o.OnLongPressDown, o.OnLongPressCancel, o.OnLongPress, o.OnLongPressStart
				r.OnLongPressMoveUpdate, r.OnLongPressUp, r.OnLongPressEnd = o.OnLongPressMoveUpdate, o.OnLongPressUp, o.OnLongPressEnd
				r.OnSecondaryLongPressDown, r.OnSecondaryLongPressCancel, r.OnSecondaryLongPress, r.OnSecondaryLongPressStart =
					o.OnSecondaryLongPressDown, o.OnSecondaryLongPressCancel, o.OnSecondaryLongPress, o.OnSecondaryLongPressStart
				r.OnSecondaryLongPressMoveUpdate, r.OnSecondaryLongPressUp, r.OnSecondaryLongPressEnd =
					o.OnSecondaryLongPressMoveUpdate, o.OnSecondaryLongPressUp, o.OnSecondaryLongPressEnd
				r.OnTertiaryLongPressDown, r.OnTertiaryLongPressCancel, r.OnTertiaryLongPress, r.OnTertiaryLongPressStart =
					o.OnTertiaryLongPressDown, o.OnTertiaryLongPressCancel, o.OnTertiaryLongPress, o.OnTertiaryLongPressStart
				r.OnTertiaryLongPressMoveUpdate, r.OnTertiaryLongPressUp, r.OnTertiaryLongPressEnd =
					o.OnTertiaryLongPressMoveUpdate, o.OnTertiaryLongPressUp, o.OnTertiaryLongPressEnd
				configure(r)
			}))
	}
	dragOpts := gestures.DragGestureRecognizerOpts{DebugOwner: w, SupportedDevices: o.SupportedDevices}
	setDrag := func(d *gestures.DragGestureRecognizer, down gestures.GestureDragDownCallback, start gestures.GestureDragStartCallback,
		update gestures.GestureDragUpdateCallback, end gestures.GestureDragEndCallback, cancel gestures.GestureDragCancelCallback) {
		d.OnDown, d.OnStart, d.OnUpdate, d.OnEnd, d.OnCancel = down, start, update, end, cancel
		d.DragStartBehavior = o.DragStartBehavior
		configure(d)
	}
	if o.OnVerticalDragDown != nil || o.OnVerticalDragStart != nil || o.OnVerticalDragUpdate != nil || o.OnVerticalDragEnd != nil || o.OnVerticalDragCancel != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.VerticalDragGestureRecognizer {
				return gestures.NewVerticalDragGestureRecognizer(binding, dragOpts)
			},
			func(r *gestures.VerticalDragGestureRecognizer) {
				setDrag(&r.DragGestureRecognizer, o.OnVerticalDragDown, o.OnVerticalDragStart, o.OnVerticalDragUpdate, o.OnVerticalDragEnd, o.OnVerticalDragCancel)
			}))
	}
	if o.OnHorizontalDragDown != nil || o.OnHorizontalDragStart != nil || o.OnHorizontalDragUpdate != nil || o.OnHorizontalDragEnd != nil || o.OnHorizontalDragCancel != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.HorizontalDragGestureRecognizer {
				return gestures.NewHorizontalDragGestureRecognizer(binding, dragOpts)
			},
			func(r *gestures.HorizontalDragGestureRecognizer) {
				setDrag(&r.DragGestureRecognizer, o.OnHorizontalDragDown, o.OnHorizontalDragStart, o.OnHorizontalDragUpdate, o.OnHorizontalDragEnd, o.OnHorizontalDragCancel)
			}))
	}
	if o.OnPanDown != nil || o.OnPanStart != nil || o.OnPanUpdate != nil || o.OnPanEnd != nil || o.OnPanCancel != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.PanGestureRecognizer { return gestures.NewPanGestureRecognizer(binding, dragOpts) },
			func(r *gestures.PanGestureRecognizer) {
				setDrag(&r.DragGestureRecognizer, o.OnPanDown, o.OnPanStart, o.OnPanUpdate, o.OnPanEnd, o.OnPanCancel)
			}))
	}
	if o.OnScaleStart != nil || o.OnScaleUpdate != nil || o.OnScaleEnd != nil {
		add(NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.ScaleGestureRecognizer {
				return gestures.NewScaleGestureRecognizer(binding, gestures.ScaleGestureRecognizerOpts{DebugOwner: w, SupportedDevices: o.SupportedDevices})
			},
			func(r *gestures.ScaleGestureRecognizer) {
				r.OnStart, r.OnUpdate, r.OnEnd = o.OnScaleStart, o.OnScaleUpdate, o.OnScaleEnd
				r.DragStartBehavior = o.DragStartBehavior
				r.TrackpadScrollCausesScale = o.TrackpadScrollCausesScale
				r.TrackpadScrollToScaleFactor = o.TrackpadScrollToScaleFactor.OrElse(gestures.DefaultTrackpadScrollToScaleFactor)
				configure(r)
			}))
	}
	return NewRawGestureDetector(w.child, RawGestureDetectorOpts{Gestures: factories, Behavior: o.Behavior, ExcludeFromSemantics: o.ExcludeFromSemantics})
}

// SemanticsGestureDelegate is the interface for a delegate that can assign the
// semantics actions of a RenderSemanticsGestureHandler.
//
// Flutter: widgets/gesture_detector.dart SemanticsGestureDelegate
type SemanticsGestureDelegate interface {
	// AssignSemantics sets the callbacks of the render object.
	//
	// Flutter: widgets/gesture_detector.dart SemanticsGestureDelegate.assignSemantics
	AssignSemantics(renderObject *rendering.RenderSemanticsGestureHandler)
}

// RawGestureDetectorOpts are the optional parameters of NewRawGestureDetector.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector.new
type RawGestureDetectorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Gestures are the gestures to detect, by the type of their recognizer.
	Gestures map[reflect.Type]GestureRecognizerFactory
	// Behavior is how this detector behaves during hit testing; the default is
	// deferToChild when there is a child, and translucent otherwise.
	Behavior rendering.HitTestBehavior
	// ExcludeFromSemantics is whether to exclude these gestures from the
	// semantics tree.
	ExcludeFromSemantics bool
	// Semantics is the delegate that assigns the semantics actions; the default
	// calls the callbacks of the recognizers.
	Semantics SemanticsGestureDelegate
}

// RawGestureDetector is a widget that detects gestures described by the given
// recognizer factories.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector
type RawGestureDetector struct {
	StatefulWidget
	child Widget
	o     RawGestureDetectorOpts
}

// NewRawGestureDetector creates the widget; the child may be nil.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector.new
func NewRawGestureDetector(child Widget, opts ...RawGestureDetectorOpts) *RawGestureDetector {
	o := firstOpts(opts)
	return &RawGestureDetector{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child, o}
}

// Child is the widget below this one in the tree, or nil.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector.child
func (w *RawGestureDetector) Child() Widget { return w.child }

// Gestures are the gestures to detect.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector.gestures
func (w *RawGestureDetector) Gestures() map[reflect.Type]GestureRecognizerFactory {
	return w.o.Gestures
}

// Behavior is how the detector behaves during hit testing.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector.behavior
func (w *RawGestureDetector) Behavior() rendering.HitTestBehavior { return w.o.Behavior }

// CreateState creates the state.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetector.createState
func (w *RawGestureDetector) CreateState() State { return &RawGestureDetectorState{} }

// RawGestureDetectorState is the state of a RawGestureDetector: it owns the
// recognizers.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState
type RawGestureDetectorState struct {
	StateOf[*RawGestureDetector]
	recognizers map[reflect.Type]gestures.GestureRecognizer
	semantics   SemanticsGestureDelegate
}

// InitState makes the recognizers.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState.initState
func (s *RawGestureDetectorState) InitState() {
	s.recognizers = map[reflect.Type]gestures.GestureRecognizer{}
	s.semantics = s.Widget().o.Semantics
	if s.semantics == nil {
		s.semantics = &defaultSemanticsGestureDelegate{s}
	}
	s.syncAll(s.Widget().o.Gestures)
}

// DidUpdateWidget updates the recognizers.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState.didUpdateWidget
func (s *RawGestureDetectorState) DidUpdateWidget(old *RawGestureDetector) {
	if old.o.Semantics != nil || s.Widget().o.Semantics != nil {
		s.semantics = s.Widget().o.Semantics
		if s.semantics == nil {
			s.semantics = &defaultSemanticsGestureDelegate{s}
		}
	}
	s.syncAll(s.Widget().o.Gestures)
}

// Recognizers are the recognizers that the state keeps, by type.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState._recognizers
func (s *RawGestureDetectorState) Recognizers() map[reflect.Type]gestures.GestureRecognizer {
	return s.recognizers
}

// ReplaceGestureRecognizers changes the recognizers. It can only be called
// during layout, by a render object that learns about the gestures then; it
// panics with the tag GestureDetectorNotDuringLayout in debug builds otherwise.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState.replaceGestureRecognizers
func (s *RawGestureDetectorState) ReplaceGestureRecognizers(gestureFactories map[reflect.Type]GestureRecognizerFactory) {
	if foundation.DebugMode {
		if ro := s.Context().FindRenderObject(); ro == nil || !ro.AsRenderObject().Owner().DebugDoingLayout() {
			panic(errors.New("GestureDetectorNotDuringLayout", "unexpected call to ReplaceGestureRecognizers; it can only be called during the layout phase; to set the gesture recognizers at other times, trigger a new build with SetState and provide the new gesture recognizers to the RawGestureDetector or GestureDetector"))
		}
	}
	s.syncAll(gestureFactories)
	if !s.Widget().o.ExcludeFromSemantics {
		s.updateSemanticsForRenderObject(s.Context().FindRenderObject().(*rendering.RenderSemanticsGestureHandler))
	}
}

// ReplaceSemanticsActions changes the semantics actions that the render object
// handles. It does nothing when the detector has no semantics handler.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState.replaceSemanticsActions
func (s *RawGestureDetectorState) ReplaceSemanticsActions(actions map[ui.SemanticsAction]struct{}) {
	if s.Widget().o.ExcludeFromSemantics {
		return
	}
	handler, ok := s.Context().FindRenderObject().(*rendering.RenderSemanticsGestureHandler)
	if !ok {
		return
	}
	handler.SetValidActions(actions)
}

// Dispose disposes of the recognizers.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState.dispose
func (s *RawGestureDetectorState) Dispose() {
	for _, r := range s.recognizers {
		r.Dispose()
	}
	s.recognizers = nil
}

func (s *RawGestureDetectorState) syncAll(factories map[reflect.Type]GestureRecognizerFactory) {
	old := s.recognizers
	s.recognizers = map[reflect.Type]gestures.GestureRecognizer{}
	for t, f := range factories {
		if foundation.DebugMode && f.RecognizerType() != t {
			panic(errors.New("GestureRecognizerFactoryTypeMismatch", "the GestureRecognizerFactory is registered under another type than the recognizers it makes",
				"registered", t.String(), "factory", f.RecognizerType().String()))
		}
		r := old[t]
		if r == nil {
			r = f.Constructor()
		}
		s.recognizers[t] = r
		f.Initializer(r)
	}
	for t, r := range old {
		if _, kept := s.recognizers[t]; !kept {
			r.Dispose()
		}
	}
}

func (s *RawGestureDetectorState) handlePointerDown(event *gestures.PointerDownEvent) {
	for _, r := range s.recognizers {
		r.AddPointer(event)
	}
}

func (s *RawGestureDetectorState) handlePointerPanZoomStart(event *gestures.PointerPanZoomStartEvent) {
	for _, r := range s.recognizers {
		r.AddPointerPanZoom(event)
	}
}

func (s *RawGestureDetectorState) defaultBehavior() rendering.HitTestBehavior {
	if s.Widget().child == nil {
		return rendering.HitTestBehaviorTranslucent
	}
	return rendering.HitTestBehaviorDeferToChild
}

func (s *RawGestureDetectorState) behavior() rendering.HitTestBehavior {
	if b := s.Widget().o.Behavior; b != rendering.HitTestBehaviorUnspecified {
		return b
	}
	return s.defaultBehavior()
}

func (s *RawGestureDetectorState) updateSemanticsForRenderObject(r *rendering.RenderSemanticsGestureHandler) {
	s.semantics.AssignSemantics(r)
}

// Build listens for the pointers and gives them to the recognizers.
//
// Flutter: widgets/gesture_detector.dart RawGestureDetectorState.build
func (s *RawGestureDetectorState) Build(BuildContext) Widget {
	var result Widget = NewListener(s.Widget().child, ListenerOpts{
		OnPointerDown: s.handlePointerDown, OnPointerPanZoomStart: s.handlePointerPanZoomStart, Behavior: s.behavior(),
	})
	if !s.Widget().o.ExcludeFromSemantics {
		result = &gestureSemantics{
			SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(result),
			behavior:                      s.behavior(), assignSemantics: s.updateSemanticsForRenderObject,
		}
	}
	return result
}

// gestureSemantics makes the RenderSemanticsGestureHandler.
type gestureSemantics struct {
	SingleChildRenderObjectWidget
	behavior        rendering.HitTestBehavior
	assignSemantics func(*rendering.RenderSemanticsGestureHandler)
}

func (w *gestureSemantics) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := rendering.NewRenderSemanticsGestureHandler(nil, rendering.RenderSemanticsGestureHandlerOpts{Behavior: w.behavior})
	w.assignSemantics(r)
	return r
}

func (w *gestureSemantics) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSemanticsGestureHandler)
	r.Behavior = w.behavior
	w.assignSemantics(r)
}

// defaultSemanticsGestureDelegate calls the callbacks of the recognizers when
// the accessibility system asks for an action.
type defaultSemanticsGestureDelegate struct{ state *RawGestureDetectorState }

func localCenterOf(renderObject rendering.RenderObject) ui.Offset {
	if b, ok := renderObject.(rendering.Box); ok {
		s := b.Size()
		return ui.Offset{Dx: s.Width / 2, Dy: s.Height / 2}
	}
	return ui.Offset{}
}

func globalOf(renderObject rendering.RenderObject, local ui.Offset) ui.Offset {
	return renderObject.GetTransformTo(nil).TransformPoint(local)
}

func (d *defaultSemanticsGestureDelegate) AssignSemantics(r *rendering.RenderSemanticsGestureHandler) {
	rec := d.state.recognizers
	r.SetOnTap(d.tapHandler(r, rec))
	r.SetOnLongPress(d.longPressHandler(r, rec))
	r.SetOnHorizontalDragUpdate(d.dragHandler(r, rec, reflect.TypeFor[*gestures.HorizontalDragGestureRecognizer]()))
	r.SetOnVerticalDragUpdate(d.dragHandler(r, rec, reflect.TypeFor[*gestures.VerticalDragGestureRecognizer]()))
}

func (d *defaultSemanticsGestureDelegate) tapHandler(r rendering.RenderObject, rec map[reflect.Type]gestures.GestureRecognizer) gestures.GestureTapCallback {
	tap, _ := rec[reflect.TypeFor[*gestures.TapGestureRecognizer]()].(*gestures.TapGestureRecognizer)
	if tap == nil {
		return nil
	}
	return func() {
		local := localCenterOf(r)
		global := globalOf(r, local)
		if tap.OnTapDown != nil {
			tap.OnTapDown(gestures.NewTapDownDetails(gestures.TapDownOpts{GlobalPosition: global, LocalPosition: foundation.Some(local),
				Kind: foundation.Some(ui.PointerDeviceKindUnknown)}))
		}
		if tap.OnTapUp != nil {
			tap.OnTapUp(gestures.NewTapUpDetails(ui.PointerDeviceKindUnknown, gestures.TapUpOpts{GlobalPosition: global, LocalPosition: foundation.Some(local)}))
		}
		if tap.OnTap != nil {
			tap.OnTap()
		}
	}
}

func (d *defaultSemanticsGestureDelegate) longPressHandler(r rendering.RenderObject, rec map[reflect.Type]gestures.GestureRecognizer) gestures.GestureLongPressCallback {
	lp, _ := rec[reflect.TypeFor[*gestures.LongPressGestureRecognizer]()].(*gestures.LongPressGestureRecognizer)
	if lp == nil {
		return nil
	}
	return func() {
		local := localCenterOf(r)
		global := globalOf(r, local)
		if lp.OnLongPressDown != nil {
			lp.OnLongPressDown(gestures.NewLongPressDownDetails(gestures.TapDownOpts{GlobalPosition: global, LocalPosition: foundation.Some(local)}))
		}
		if lp.OnLongPressStart != nil {
			lp.OnLongPressStart(gestures.NewLongPressStartDetails(gestures.TapUpOpts{GlobalPosition: global, LocalPosition: foundation.Some(local)}))
		}
		if lp.OnLongPress != nil {
			lp.OnLongPress()
		}
		if lp.OnLongPressEnd != nil {
			lp.OnLongPressEnd(gestures.NewLongPressEndDetails(gestures.LongPressEndOpts{GlobalPosition: global, LocalPosition: foundation.Some(local)}))
		}
		if lp.OnLongPressUp != nil {
			lp.OnLongPressUp()
		}
	}
}

// dragHandler fakes a whole drag with one update, for the recognizer of the axis
// and the pan recognizer.
func (d *defaultSemanticsGestureDelegate) dragHandler(r rendering.RenderObject, rec map[reflect.Type]gestures.GestureRecognizer,
	axisType reflect.Type) gestures.GestureDragUpdateCallback {
	var axis *gestures.DragGestureRecognizer
	switch a := rec[axisType].(type) {
	case *gestures.HorizontalDragGestureRecognizer:
		axis = &a.DragGestureRecognizer
	case *gestures.VerticalDragGestureRecognizer:
		axis = &a.DragGestureRecognizer
	}
	var pan *gestures.DragGestureRecognizer
	if p, ok := rec[reflect.TypeFor[*gestures.PanGestureRecognizer]()].(*gestures.PanGestureRecognizer); ok {
		pan = &p.DragGestureRecognizer
	}
	if axis == nil && pan == nil {
		return nil
	}
	run := func(g *gestures.DragGestureRecognizer, zeroVelocity bool) gestures.GestureDragUpdateCallback {
		if g == nil {
			return nil
		}
		return func(details gestures.DragUpdateDetails) {
			local := localCenterOf(r)
			global := globalOf(r, local)
			newLocal := ui.Offset{Dx: local.Dx + details.Delta.Dx, Dy: local.Dy + details.Delta.Dy}
			newGlobal := globalOf(r, newLocal)
			if g.OnDown != nil {
				g.OnDown(gestures.NewDragDownDetails(gestures.DragDownOpts{GlobalPosition: global, LocalPosition: foundation.Some(local)}))
			}
			if g.OnStart != nil {
				g.OnStart(gestures.NewDragStartDetails(gestures.DragStartOpts{GlobalPosition: global, LocalPosition: foundation.Some(local)}))
			}
			if g.OnUpdate != nil {
				g.OnUpdate(details)
			}
			if g.OnEnd != nil {
				end := gestures.DragEndOpts{GlobalPosition: newGlobal, LocalPosition: foundation.Some(newLocal)}
				if zeroVelocity {
					end.PrimaryVelocity = foundation.Some(0.0)
				}
				g.OnEnd(gestures.NewDragEndDetails(end))
			}
		}
	}
	a, p := run(axis, true), run(pan, false)
	return func(details gestures.DragUpdateDetails) {
		if a != nil {
			a(details)
		}
		if p != nil {
			p(details)
		}
	}
}
