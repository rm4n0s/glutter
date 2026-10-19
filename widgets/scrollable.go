package widgets

import (
	"math"
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// ViewportBuilder builds the viewport of a Scrollable, given the position that
// the viewport must listen to.
//
// Flutter: widgets/scrollable.dart ViewportBuilder
type ViewportBuilder func(bc BuildContext, position rendering.ViewportOffset) Widget

// ScrollableOpts are the optional parameters of NewScrollable.
//
// Flutter: widgets/scrollable.dart Scrollable.new
type ScrollableOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AxisDirection is the direction in which the scroll offset increases; the
	// default is AxisDirectionDown.
	AxisDirection painting.AxisDirection
	// Controller controls the scroll position; a fallback one is made without it.
	Controller ScrollController
	// Physics are the physics of the scrollable; the ScrollBehavior's by
	// default.
	Physics ScrollPhysics
	// IncrementCalculator is the size of the scroll increments, for the keyboard
	// and the scrollbar track; the defaults are 50 pixels and 80% of the
	// viewport.
	IncrementCalculator ScrollIncrementCalculator
	// ExcludeFromSemantics leaves the scrollable out of the semantics tree.
	ExcludeFromSemantics bool
	// SemanticChildCount is the number of children that contribute semantics,
	// when it is not all of the ones the viewport builds; none for unknown.
	SemanticChildCount foundation.Opt[int]
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// ScrollBehavior is the behavior for this scrollable instead of the
	// ScrollConfiguration's.
	ScrollBehavior ScrollBehavior
	// ClipBehavior is the clip of the decorations of the scrollable; the default
	// is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the scrollable behaves during hit testing; the
	// default is HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
}

// Scrollable implements the interaction model of a scrollable widget: gestures,
// the scroll position and the semantics. It does not draw anything: the viewport
// that ViewportBuilder returns shows the content.
//
// Deviation: Flutter's has restorationId, the selection support and
// TwoDimensionalScrollable (SPEC §12.10).
//
// Flutter: widgets/scrollable.dart Scrollable
type Scrollable struct {
	StatefulWidget
	viewportBuilder      ViewportBuilder
	axisDirection        painting.AxisDirection
	controller           ScrollController
	physics              ScrollPhysics
	incrementCalculator  ScrollIncrementCalculator
	excludeFromSemantics bool
	hitTestBehavior      rendering.HitTestBehavior
	semanticChildCount   foundation.Opt[int]
	dragStartBehavior    gestures.DragStartBehavior
	scrollBehavior       ScrollBehavior
	clipBehavior         ui.Clip
	// fixedItemExtent is the item extent of a scrollable of a ListWheelScrollView,
	// which a FixedExtentScrollController needs.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart _FixedExtentScrollable.itemExtent
	fixedItemExtent foundation.Opt[float64]
	// dimension is set on the two scrollables of a TwoDimensionalScrollable.
	//
	// Flutter: widgets/scrollable.dart _VerticalOuterDimension, _HorizontalInnerDimension
	dimension *twoDimensionalDimension
}

// NewScrollable creates the widget. In debug builds it panics with the tag
// ScrollableSemanticChildCountInvalid when the semantic child count is negative.
//
// Flutter: widgets/scrollable.dart Scrollable.new
func NewScrollable(viewportBuilder ViewportBuilder, opts ...ScrollableOpts) *Scrollable {
	o := firstOpts(opts)
	if c, ok := o.SemanticChildCount.Get(); ok && c < 0 && foundation.DebugMode {
		panic(errors.New("ScrollableSemanticChildCountInvalid", "the semantic child count must not be negative", "semanticChildCount", c))
	}
	if o.AxisDirection == painting.AxisDirectionUnspecified {
		o.AxisDirection = painting.AxisDirectionDown
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	return &Scrollable{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), viewportBuilder: viewportBuilder,
		axisDirection: o.AxisDirection, controller: o.Controller, physics: o.Physics, incrementCalculator: o.IncrementCalculator,
		excludeFromSemantics: o.ExcludeFromSemantics, hitTestBehavior: o.HitTestBehavior, semanticChildCount: o.SemanticChildCount,
		dragStartBehavior: o.DragStartBehavior, scrollBehavior: o.ScrollBehavior, clipBehavior: o.ClipBehavior,
	}
}

// fixedItemExtent is the item extent of the scrollable of a ListWheelScrollView,
// or none.
func (s *ScrollableState) fixedItemExtent() foundation.Opt[float64] {
	return s.Widget().fixedItemExtent
}

// AxisDirection is the direction in which the scroll offset increases.
//
// Flutter: widgets/scrollable.dart Scrollable.axisDirection
func (w *Scrollable) AxisDirection() painting.AxisDirection { return w.axisDirection }

// Controller is the controller of the scroll position, or nil.
//
// Flutter: widgets/scrollable.dart Scrollable.controller
func (w *Scrollable) Controller() ScrollController { return w.controller }

// Physics are the physics of the scrollable, or nil.
//
// Flutter: widgets/scrollable.dart Scrollable.physics
func (w *Scrollable) Physics() ScrollPhysics { return w.physics }

// ViewportBuilder builds the viewport.
//
// Flutter: widgets/scrollable.dart Scrollable.viewportBuilder
func (w *Scrollable) ViewportBuilder() ViewportBuilder { return w.viewportBuilder }

// IncrementCalculator is the calculator of the scroll increments, or nil.
//
// Flutter: widgets/scrollable.dart Scrollable.incrementCalculator
func (w *Scrollable) IncrementCalculator() ScrollIncrementCalculator { return w.incrementCalculator }

// ExcludeFromSemantics is whether the scrollable is left out of the semantics.
//
// Flutter: widgets/scrollable.dart Scrollable.excludeFromSemantics
func (w *Scrollable) ExcludeFromSemantics() bool { return w.excludeFromSemantics }

// HitTestBehavior is how the scrollable behaves during hit testing.
//
// Flutter: widgets/scrollable.dart Scrollable.hitTestBehavior
func (w *Scrollable) HitTestBehavior() rendering.HitTestBehavior { return w.hitTestBehavior }

// SemanticChildCount is the number of children that contribute semantics.
//
// Flutter: widgets/scrollable.dart Scrollable.semanticChildCount
func (w *Scrollable) SemanticChildCount() foundation.Opt[int] { return w.semanticChildCount }

// DragStartBehavior is where a drag starts.
//
// Flutter: widgets/scrollable.dart Scrollable.dragStartBehavior
func (w *Scrollable) DragStartBehavior() gestures.DragStartBehavior { return w.dragStartBehavior }

// ScrollBehavior is the behavior of this scrollable, or nil for the
// configuration's.
//
// Flutter: widgets/scrollable.dart Scrollable.scrollBehavior
func (w *Scrollable) ScrollBehavior() ScrollBehavior { return w.scrollBehavior }

// ClipBehavior is the clip of the decorations.
//
// Flutter: widgets/scrollable.dart Scrollable.clipBehavior
func (w *Scrollable) ClipBehavior() ui.Clip { return w.clipBehavior }

// Axis is the axis of the direction.
//
// Flutter: widgets/scrollable.dart Scrollable.axis
func (w *Scrollable) Axis() painting.Axis { return painting.AxisDirectionToAxis(w.axisDirection) }

// CreateState creates the state.
//
// Flutter: widgets/scrollable.dart Scrollable.createState
func (w *Scrollable) CreateState() State {
	return &ScrollableState{
		scrollSemanticsKey:  NewGlobalKey[State](),
		gestureDetectorKey:  NewGlobalKey[*RawGestureDetectorState](),
		ignorePointerKey:    NewGlobalKey[State](),
		gestureRecognizers:  map[reflect.Type]GestureRecognizerFactory{},
		mediaQueryGestureSt: foundation.Opt[gestures.DeviceGestureSettings]{},
	}
}

// scrollableScope is the inherited widget that gives the scrollable below it its
// state and position.
//
// Flutter: widgets/scrollable.dart _ScrollableScope
type scrollableScope struct {
	InheritedWidget
	scrollable *ScrollableState
	position   ScrollPosition
}

func newScrollableScope(scrollable *ScrollableState, position ScrollPosition, child Widget) *scrollableScope {
	return &scrollableScope{NewInheritedWidgetBase(child), scrollable, position}
}

func (w *scrollableScope) UpdateShouldNotify(old Widget) bool {
	return w.position != old.(*scrollableScope).position
}

// ScrollableOfOpts are the optional parameters of ScrollableMaybeOf and
// ScrollableOf.
//
// Flutter: widgets/scrollable.dart Scrollable.of
type ScrollableOfOpts struct {
	// Axis, if set, looks for the closest Scrollable that scrolls along it.
	Axis foundation.Opt[painting.Axis]
}

// ScrollableMaybeOf is the state of the closest Scrollable above the context (the
// closest along the axis, if there is one in the options), or nil. The context
// depends on it.
//
// Flutter: widgets/scrollable.dart Scrollable.maybeOf
func ScrollableMaybeOf(bc BuildContext, opts ...ScrollableOfOpts) *ScrollableState {
	axis := firstOpts(opts).Axis
	originalContext := bc
	element := GetElementForInheritedWidget[*scrollableScope](bc)
	for element != nil {
		scrollable := element.Widget().(*scrollableScope).scrollable
		if !axis.IsSome() || painting.AxisDirectionToAxis(scrollable.AxisDirection()) == axis.Value() {
			originalContext.DependOnInheritedElement(element, nil)
			return scrollable
		}
		bc = scrollable.Context()
		element = GetElementForInheritedWidget[*scrollableScope](bc)
	}
	return nil
}

// ScrollableOf is the state of the closest Scrollable above the context. In
// debug builds it panics with the tag ScrollableNotFound when there is none.
//
// Flutter: widgets/scrollable.dart Scrollable.of
func ScrollableOf(bc BuildContext, opts ...ScrollableOfOpts) *ScrollableState {
	s := ScrollableMaybeOf(bc, opts...)
	if s == nil && foundation.DebugMode {
		axis := ""
		if a := firstOpts(opts).Axis; a.IsSome() {
			axis = " for the provided Axis: " + a.Value().String()
		}
		panic(errors.New("ScrollableNotFound",
			"ScrollableOf() was called with a context that does not contain a Scrollable widget; no Scrollable widget ancestor could be found"+axis+
				" starting from the context that was passed to ScrollableOf(); this can happen because you are using a widget that looks for a Scrollable ancestor, but no such ancestor exists",
			"context", bc.(Element).String()))
	}
	return s
}

// ScrollableRecommendDeferredLoadingForContext is whether the content below the
// context should defer loading images because the closest Scrollable (along the
// axis, if there is one in the options) moves fast.
//
// Flutter: widgets/scrollable.dart Scrollable.recommendDeferredLoadingForContext
func ScrollableRecommendDeferredLoadingForContext(bc BuildContext, opts ...ScrollableOfOpts) bool {
	axis := firstOpts(opts).Axis
	widget, ok := GetInherited[*scrollableScope](bc)
	for ok {
		if !axis.IsSome() || painting.AxisDirectionToAxis(widget.scrollable.AxisDirection()) == axis.Value() {
			return widget.position.RecommendDeferredLoading(bc)
		}
		bc = widget.scrollable.Context()
		widget, ok = GetInherited[*scrollableScope](bc)
	}
	return false
}

// ScrollableEnsureVisible scrolls the scrollables that contain the context's
// render object so that it is visible, starting with the closest and then the
// ones around it. The future completes when the animations have.
//
// Flutter: widgets/scrollable.dart Scrollable.ensureVisible
func ScrollableEnsureVisible(bc BuildContext, opts ...EnsureVisibleOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	var futures []*scheduler.TickerFuture
	var targetRenderObject rendering.RenderObject
	scrollable := ScrollableMaybeOf(bc)
	for scrollable != nil {
		oo := o
		oo.TargetRenderObject = targetRenderObject
		future, next := scrollable.performEnsureVisible(bc.FindRenderObject(), oo)
		futures = append(futures, future)
		if isNil(targetRenderObject) {
			targetRenderObject = bc.FindRenderObject()
		}
		bc = next.Context()
		scrollable = ScrollableMaybeOf(bc)
	}
	if len(futures) == 0 || o.Duration == 0 {
		return scheduler.NewTickerFutureComplete()
	}
	if len(futures) == 1 {
		return futures[0]
	}
	return scheduler.WaitTickerFutures(futures...)
}

// ScrollableState is the state of a Scrollable: it owns the scroll position,
// listens to the gestures that drag it and to the pointer signals that scroll
// it, and implements ScrollContext for the position.
//
// Flutter: widgets/scrollable.dart ScrollableState
type ScrollableState struct {
	StateOf[*Scrollable]
	TickerProvider

	position            ScrollPosition
	physics             ScrollPhysics
	devicePixelRatio    float64
	configuration       ScrollBehavior
	fallbackController  ScrollController
	mediaQueryGestureSt foundation.Opt[gestures.DeviceGestureSettings]
	scrollSemanticsKey  *GlobalKey[State]
	gestureDetectorKey  *GlobalKey[*RawGestureDetectorState]
	ignorePointerKey    *GlobalKey[State]
	gestureRecognizers  map[reflect.Type]GestureRecognizerFactory
	shouldIgnorePointer bool
	lastCanDrag         foundation.Opt[bool]
	lastAxis            foundation.Opt[painting.Axis]
	drag                gestures.Drag
	hold                ScrollHoldController

	// Used by the dimensions of a TwoDimensionalScrollable.
	lockedAxis         foundation.Opt[painting.Axis]
	lastDragOffset     foundation.Opt[ui.Offset]
	verticalScrollable *ScrollableState
}

// Position is the scroll position.
//
// Flutter: widgets/scrollable.dart ScrollableState.position
func (s *ScrollableState) Position() ScrollPosition { return s.position }

// ResolvedPhysics are the physics the position uses: those of the widget on top
// of the scroll behavior's.
//
// Flutter: widgets/scrollable.dart ScrollableState.resolvedPhysics
func (s *ScrollableState) ResolvedPhysics() ScrollPhysics { return s.physics }

// DeltaToScrollOrigin is the offset of the scroll origin from the top left of
// the content: the scroll position along the axis, with the sign of the
// direction.
//
// Flutter: widgets/scrollable.dart ScrollableState.deltaToScrollOrigin
func (s *ScrollableState) DeltaToScrollOrigin() ui.Offset {
	switch s.AxisDirection() {
	case painting.AxisDirectionUp:
		return ui.Offset{Dy: -s.position.Pixels()}
	case painting.AxisDirectionLeft:
		return ui.Offset{Dx: -s.position.Pixels()}
	case painting.AxisDirectionRight:
		return ui.Offset{Dx: s.position.Pixels()}
	}
	return ui.Offset{Dy: s.position.Pixels()}
}

func (s *ScrollableState) effectiveScrollController() ScrollController {
	if c := s.Widget().controller; !isNil(c) {
		return c
	}
	return s.fallbackController
}

// AxisDirection is the direction of the widget.
//
// Flutter: widgets/scrollable.dart ScrollableState.axisDirection
func (s *ScrollableState) AxisDirection() painting.AxisDirection { return s.Widget().axisDirection }

// Vsync is the state itself: it is the ticker provider.
//
// Flutter: widgets/scrollable.dart ScrollableState.vsync
func (s *ScrollableState) Vsync() scheduler.TickerProvider { return s }

// DevicePixelRatio is the device pixel ratio of the media query or the view.
//
// Flutter: widgets/scrollable.dart ScrollableState.devicePixelRatio
func (s *ScrollableState) DevicePixelRatio() float64 { return s.devicePixelRatio }

// NotificationContext is the context of the gesture detector, where the
// notifications start; nil when it is not built.
//
// Flutter: widgets/scrollable.dart ScrollableState.notificationContext
func (s *ScrollableState) NotificationContext() BuildContext {
	if c := s.gestureDetectorKey.CurrentContext(); c != nil {
		return c
	}
	return nil
}

// StorageContext is the context of the state.
//
// Flutter: widgets/scrollable.dart ScrollableState.storageContext
func (s *ScrollableState) StorageContext() BuildContext { return s.Context() }

// SaveOffset does nothing: state restoration is not supported (SPEC §2).
//
// Flutter: widgets/scrollable.dart ScrollableState.saveOffset
func (s *ScrollableState) SaveOffset(float64) {}

func (s *ScrollableState) updatePosition() {
	w := s.Widget()
	if !isNil(w.scrollBehavior) {
		s.configuration = w.scrollBehavior
	} else {
		s.configuration = ScrollConfigurationOf(s.Context())
	}
	physicsFromWidget := w.physics
	if isNil(physicsFromWidget) && !isNil(w.scrollBehavior) {
		physicsFromWidget = w.scrollBehavior.GetScrollPhysics(s.Context())
	}
	s.physics = s.configuration.GetScrollPhysics(s.Context())
	if !isNil(physicsFromWidget) {
		s.physics = physicsFromWidget.ApplyTo(s.physics)
	}
	oldPosition := s.position
	if !isNil(oldPosition) {
		s.effectiveScrollController().Detach(oldPosition)
		// It's important that we not dispose the old position until after the
		// viewport has had a chance to unregister its listeners from the old
		// position. So, schedule a microtask to do it.
		WidgetsBindingOf(s.Context()).Post(oldPosition.Dispose)
	}
	s.position = s.effectiveScrollController().CreateScrollPosition(s.physics, s, oldPosition)
	s.effectiveScrollController().Attach(s.position)
}

// InitState makes the fallback controller if the widget has none.
//
// Flutter: widgets/scrollable.dart ScrollableState.initState
func (s *ScrollableState) InitState() {
	if isNil(s.Widget().controller) {
		s.fallbackController = NewScrollController()
	}
}

// DidChangeDependencies reads the media query and updates the position.
//
// Flutter: widgets/scrollable.dart ScrollableState.didChangeDependencies
func (s *ScrollableState) DidChangeDependencies() {
	if dim := s.Widget().dimension; dim != nil && !dim.vertical {
		s.verticalScrollable = ScrollableOf(s.Context())
	}
	if settings, ok := MediaQueryMaybeGestureSettingsOf(s.Context()); ok {
		s.mediaQueryGestureSt = foundation.Some(settings)
	} else {
		s.mediaQueryGestureSt = foundation.Opt[gestures.DeviceGestureSettings]{}
	}
	if ratio, ok := MediaQueryMaybeDevicePixelRatioOf(s.Context()); ok {
		s.devicePixelRatio = ratio
	} else {
		s.devicePixelRatio = ViewOf(s.Context()).DevicePixelRatio()
	}
	s.updatePosition()
}

func (s *ScrollableState) shouldUpdatePosition(oldWidget *Scrollable) bool {
	w := s.Widget()
	if isNil(w.scrollBehavior) != isNil(oldWidget.scrollBehavior) {
		return true
	}
	if !isNil(w.scrollBehavior) && !isNil(oldWidget.scrollBehavior) && w.scrollBehavior.ShouldNotify(oldWidget.scrollBehavior) {
		return true
	}
	newPhysics := w.physics
	if isNil(newPhysics) && !isNil(w.scrollBehavior) {
		newPhysics = w.scrollBehavior.GetScrollPhysics(s.Context())
	}
	oldPhysics := oldWidget.physics
	if isNil(oldPhysics) && !isNil(oldWidget.scrollBehavior) {
		oldPhysics = oldWidget.scrollBehavior.GetScrollPhysics(s.Context())
	}
	for {
		if typeOrNil(newPhysics) != typeOrNil(oldPhysics) {
			return true
		}
		if !isNil(newPhysics) {
			newPhysics = newPhysics.Parent()
		}
		if !isNil(oldPhysics) {
			oldPhysics = oldPhysics.Parent()
		}
		if isNil(newPhysics) && isNil(oldPhysics) {
			break
		}
	}
	return typeOrNil(w.controller) != typeOrNil(oldWidget.controller)
}

func typeOrNil(x any) reflect.Type {
	if isNil(x) {
		return nil
	}
	return reflect.TypeOf(x)
}

// DidUpdateWidget attaches the position to the new controller, and updates the
// position if the configuration changed.
//
// Flutter: widgets/scrollable.dart ScrollableState.didUpdateWidget
func (s *ScrollableState) DidUpdateWidget(oldWidget *Scrollable) {
	w := s.Widget()
	if w.controller != oldWidget.controller {
		if isNil(oldWidget.controller) {
			// Have to be careful with the fallback controller.
			s.fallbackController.Detach(s.position)
			s.fallbackController.Dispose()
			s.fallbackController = nil
		} else {
			oldWidget.controller.Detach(s.position)
			if isNil(w.controller) {
				s.fallbackController = NewScrollController()
			}
		}
		s.effectiveScrollController().Attach(s.position)
	}
	if s.shouldUpdatePosition(oldWidget) {
		s.updatePosition()
	}
}

// Dispose detaches and disposes the position and the fallback controller.
//
// Flutter: widgets/scrollable.dart ScrollableState.dispose
func (s *ScrollableState) Dispose() {
	if c := s.Widget().controller; !isNil(c) {
		c.Detach(s.position)
	} else if !isNil(s.fallbackController) {
		s.fallbackController.Detach(s.position)
		s.fallbackController.Dispose()
	}
	s.position.Dispose()
}

// SetSemanticsActions sets the scroll actions of the gesture detector's
// semantics.
//
// Flutter: widgets/scrollable.dart ScrollableState.setSemanticsActions
func (s *ScrollableState) SetSemanticsActions(actions map[ui.SemanticsAction]struct{}) {
	if state, ok := s.gestureDetectorKey.CurrentState(); ok {
		state.ReplaceSemanticsActions(actions)
	}
}

// SetCanDrag sets the drag recognizers: one for the axis if the user can drag the
// scrollable, none otherwise.
//
// Flutter: widgets/scrollable.dart ScrollableState.setCanDrag
func (s *ScrollableState) SetCanDrag(value bool) {
	if dim := s.Widget().dimension; dim != nil && dim.diagonalDragBehavior != DiagonalDragBehaviorNone {
		s.dimensionSetCanDrag(dim, value)
		return
	}
	axis := s.Widget().Axis()
	if s.lastCanDrag.IsSome() && value == s.lastCanDrag.Value() && (!value || (s.lastAxis.IsSome() && axis == s.lastAxis.Value())) {
		// We have already set up the correct gesture recognizers. Return early to
		// avoid the costly call to replaceGestureRecognizers.
		return
	}
	if !value {
		s.gestureRecognizers = map[reflect.Type]GestureRecognizerFactory{}
		s.handleDragCancel()
	} else {
		binding := WidgetsBindingOf(s.Context()).Gestures()
		initialize := func(r *gestures.DragGestureRecognizer) {
			r.OnDown = s.handleDragDown
			r.OnStart = s.handleDragStart
			r.OnUpdate = s.handleDragUpdate
			r.OnEnd = s.handleDragEnd
			r.OnCancel = s.handleDragCancel
			if s.physics != nil {
				r.MinFlingDistance = foundation.Some(s.physics.MinFlingDistance())
				r.MinFlingVelocity = foundation.Some(s.physics.MinFlingVelocity())
				r.MaxFlingVelocity = foundation.Some(s.physics.MaxFlingVelocity())
			}
			r.VelocityTrackerBuilder = s.configuration.VelocityTrackerBuilder(s.Context())
			r.DragStartBehavior = s.Widget().dragStartBehavior
			r.MultitouchDragStrategy = s.configuration.GetMultitouchDragStrategy(s.Context())
			r.GestureSettings = s.mediaQueryGestureSt
			r.SupportedDevices = s.configuration.DragDevices()
		}
		opts := gestures.DragGestureRecognizerOpts{SupportedDevices: s.configuration.DragDevices()}
		switch axis {
		case painting.AxisVertical:
			f := NewGestureRecognizerFactoryWithHandlers(
				func() *gestures.VerticalDragGestureRecognizer {
					return gestures.NewVerticalDragGestureRecognizer(binding, opts)
				},
				func(r *gestures.VerticalDragGestureRecognizer) { initialize(&r.DragGestureRecognizer) })
			s.gestureRecognizers = map[reflect.Type]GestureRecognizerFactory{f.RecognizerType(): f}
		default:
			f := NewGestureRecognizerFactoryWithHandlers(
				func() *gestures.HorizontalDragGestureRecognizer {
					return gestures.NewHorizontalDragGestureRecognizer(binding, opts)
				},
				func(r *gestures.HorizontalDragGestureRecognizer) { initialize(&r.DragGestureRecognizer) })
			s.gestureRecognizers = map[reflect.Type]GestureRecognizerFactory{f.RecognizerType(): f}
		}
	}
	s.lastCanDrag = foundation.Some(value)
	s.lastAxis = foundation.Some(axis)
	if state, ok := s.gestureDetectorKey.CurrentState(); ok {
		state.ReplaceGestureRecognizers(s.gestureRecognizers)
	}
}

// SetIgnorePointer makes the scrollable ignore (or stop ignoring) pointer
// events.
//
// Flutter: widgets/scrollable.dart ScrollableState.setIgnorePointer
func (s *ScrollableState) SetIgnorePointer(value bool) {
	if s.shouldIgnorePointer == value {
		return
	}
	s.shouldIgnorePointer = value
	if c := s.ignorePointerKey.CurrentContext(); c != nil {
		c.FindRenderObject().(*rendering.RenderIgnorePointer).SetIgnoring(s.shouldIgnorePointer)
	}
}

func (s *ScrollableState) handleDragDown(details gestures.DragDownDetails) {
	if dim := s.Widget().dimension; dim != nil && dim.vertical {
		s.outerDragDown(dim, details)
		return
	}
	s.baseDragDown(details)
}

func (s *ScrollableState) baseDragDown(gestures.DragDownDetails) {
	if foundation.DebugMode && (s.drag != nil || s.hold != nil) {
		panic(errors.New("ScrollableDragStateInvalid", "a drag down arrived while the scrollable has a drag or a hold"))
	}
	s.hold = s.position.Hold(s.disposeHold)
}

func (s *ScrollableState) handleDragStart(details gestures.DragStartDetails) {
	if dim := s.Widget().dimension; dim != nil && dim.vertical {
		s.outerDragStart(dim, details)
		return
	}
	s.baseDragStart(details)
}

func (s *ScrollableState) baseDragStart(details gestures.DragStartDetails) {
	if foundation.DebugMode && s.drag != nil {
		panic(errors.New("ScrollableDragStateInvalid", "a drag start arrived while the scrollable has a drag"))
	}
	s.drag = s.position.Drag(details, s.disposeDrag)
	if s.hold != nil {
		s.disposeHold()
	}
}

func (s *ScrollableState) handleDragUpdate(details gestures.DragUpdateDetails) {
	if dim := s.Widget().dimension; dim != nil && dim.vertical {
		s.outerDragUpdate(dim, details)
		return
	}
	s.baseDragUpdate(details)
}

func (s *ScrollableState) baseDragUpdate(details gestures.DragUpdateDetails) {
	// _drag might be null if the drag activity ended and called _disposeDrag.
	if s.drag != nil {
		s.drag.Update(details)
	}
}

func (s *ScrollableState) handleDragEnd(details gestures.DragEndDetails) {
	if dim := s.Widget().dimension; dim != nil && dim.vertical {
		s.outerDragEnd(dim, details)
		return
	}
	s.baseDragEnd(details)
}

func (s *ScrollableState) baseDragEnd(details gestures.DragEndDetails) {
	// _drag might be null if the drag activity ended and called _disposeDrag.
	if s.drag != nil {
		s.drag.End(details)
	}
	if foundation.DebugMode && s.drag != nil {
		panic(errors.New("ScrollableDragStateInvalid", "the drag of the scrollable did not end with the drag end"))
	}
}

func (s *ScrollableState) handleDragCancel() {
	if dim := s.Widget().dimension; dim != nil && dim.vertical {
		s.outerDragCancel(dim)
		return
	}
	s.baseDragCancel()
}

func (s *ScrollableState) baseDragCancel() {
	if s.gestureDetectorKey.CurrentContext() == nil {
		// The cancel was caused by the GestureDetector getting disposed, which
		// means we will get disposed momentarily as well and shouldn't do
		// any work.
		return
	}
	// _hold might be null if the drag started.
	// _drag might be null if the drag activity ended and called _disposeDrag.
	if s.hold != nil {
		s.hold.Cancel()
	}
	if s.drag != nil {
		s.drag.Cancel()
	}
	if foundation.DebugMode && (s.hold != nil || s.drag != nil) {
		panic(errors.New("ScrollableDragStateInvalid", "the hold and the drag of the scrollable did not end with the cancel"))
	}
}

func (s *ScrollableState) disposeHold() { s.hold = nil }

func (s *ScrollableState) disposeDrag() { s.drag = nil }

func (s *ScrollableState) targetScrollOffsetForPointerScroll(delta float64) float64 {
	return math.Min(math.Max(s.position.Pixels()+delta, s.position.MinScrollExtent()), s.position.MaxScrollExtent())
}

// pointerSignalEventDelta is the number of pixels a scroll event scrolls the
// position by.
//
// Deviation: Flutter flips the axes of a mouse wheel while one of the
// pointerAxisModifiers (shift) is held; the keyboard state comes with step 10.4
// (SPEC §12.10).
func (s *ScrollableState) pointerSignalEventDelta(event *gestures.PointerScrollEvent) float64 {
	var delta float64
	if s.Widget().Axis() == painting.AxisHorizontal {
		delta = event.ScrollDelta().Dx
	} else {
		delta = event.ScrollDelta().Dy
	}
	if painting.AxisDirectionIsReversed(s.Widget().axisDirection) {
		return -delta
	}
	return delta
}

func (s *ScrollableState) receivedPointerSignal(event gestures.PointerSignalEvent) {
	switch e := event.(type) {
	case *gestures.PointerScrollEvent:
		if !isNil(s.position) {
			// Only try to scroll if the user can scroll the axis.
			if s.physics != nil && !s.physics.ShouldAcceptUserOffset(s.position) {
				return
			}
			delta := s.pointerSignalEventDelta(e)
			targetScrollOffset := s.targetScrollOffsetForPointerScroll(delta)
			// If we should scroll, register with the pointer signal resolver.
			if delta != 0 && targetScrollOffset != s.position.Pixels() {
				WidgetsBindingOf(s.Context()).Gestures().PointerSignalResolver.Register(e, s.handlePointerScroll)
				return
			}
			return
		}
	case *gestures.PointerScrollInertiaCancelEvent:
		// Cancel the inertia with a zero scroll.
		s.position.PointerScroll(0)
	}
}

func (s *ScrollableState) handlePointerScroll(event gestures.PointerSignalEvent) {
	scrollEvent := event.(*gestures.PointerScrollEvent)
	delta := s.pointerSignalEventDelta(scrollEvent)
	targetScrollOffset := s.targetScrollOffsetForPointerScroll(delta)
	if delta != 0 && targetScrollOffset != s.position.Pixels() {
		s.position.PointerScroll(delta)
		scrollEvent.Respond(false)
	}
}

func (s *ScrollableState) handleScrollMetricsNotification(notification *ScrollMetricsNotification) bool {
	if notification.Depth() == 0 {
		if c := s.scrollSemanticsKey.CurrentContext(); c != nil {
			if ro := c.FindRenderObject(); ro != nil {
				ro.MarkNeedsSemanticsUpdate()
			}
		}
	}
	return false
}

func (s *ScrollableState) buildChrome(bc BuildContext, child Widget) Widget {
	if s.Widget().dimension != nil {
		// Skip building a scrollbar here, the dual scrollbar is added in
		// TwoDimensionalScrollableState.
		return s.configuration.BuildOverscrollIndicator(bc, child, ScrollableDetails{
			Direction: s.Widget().axisDirection, Controller: s.effectiveScrollController(), DecorationClipBehavior: foundation.Some(s.Widget().clipBehavior)})
	}
	details := ScrollableDetails{
		Direction:              s.Widget().axisDirection,
		Controller:             s.effectiveScrollController(),
		DecorationClipBehavior: foundation.Some(s.Widget().clipBehavior),
	}
	return s.configuration.BuildScrollbar(bc, s.configuration.BuildOverscrollIndicator(bc, child, details), details)
}

// Build builds the listener for pointer signals, the gesture detector, the
// semantics and the viewport.
//
// Flutter: widgets/scrollable.dart ScrollableState.build
func (s *ScrollableState) Build(bc BuildContext) Widget {
	w := s.Widget()
	var result Widget = newScrollableScope(s, s.position,
		NewListener(
			NewRawGestureDetector(
				NewSemantics(
					NewIgnorePointer(w.viewportBuilder(bc, s.position),
						IgnorePointerOpts{Key: s.ignorePointerKey, Ignoring: foundation.Some(s.shouldIgnorePointer)}),
					SemanticsOpts{ExplicitChildNodes: !w.excludeFromSemantics}),
				RawGestureDetectorOpts{Key: s.gestureDetectorKey, Gestures: s.gestureRecognizers, Behavior: w.hitTestBehavior,
					ExcludeFromSemantics: w.excludeFromSemantics}),
			ListenerOpts{OnPointerSignal: s.receivedPointerSignal}))
	if !w.excludeFromSemantics {
		result = NewNotificationListener[*ScrollMetricsNotification](
			newScrollSemantics(s.position, s.physics.AllowImplicitScrolling(), w.Axis(), w.semanticChildCount, result, s.scrollSemanticsKey),
			s.handleScrollMetricsNotification)
	}
	return s.buildChrome(bc, result)
}

// performEnsureVisible scrolls the position to reveal the object, and returns
// the future of that and the scrollable whose context is used to find the next
// ancestor scrollable.
//
// Flutter: widgets/scrollable.dart ScrollableState._performEnsureVisible
func (s *ScrollableState) performEnsureVisible(object rendering.RenderObject, opts EnsureVisibleOpts) (*scheduler.TickerFuture, *ScrollableState) {
	if dim := s.Widget().dimension; dim != nil {
		return s.dimensionEnsureVisible(dim, object, opts)
	}
	return s.position.EnsureVisible(object, opts), s
}
