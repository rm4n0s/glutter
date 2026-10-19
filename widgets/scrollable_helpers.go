package widgets

import (
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollableDetails describes the aspects of a scrollable on one axis: its
// direction, controller and physics, and the clip of its decorations. The
// ScrollBehavior builds its scrollbar and overscroll indicator from them.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails
type ScrollableDetails struct {
	// Direction is the direction in which the scroll offset increases.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.direction
	Direction painting.AxisDirection
	// Controller is the controller of the scrollable, or nil.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.controller
	Controller ScrollController
	// Physics are the physics of the scrollable, or nil.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.physics
	Physics ScrollPhysics
	// DecorationClipBehavior is the clip of the decorators (such as the
	// overscroll indicator) of the scrollable; none for the scrollable's own.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.decorationClipBehavior
	DecorationClipBehavior foundation.Opt[ui.Clip]
}

// ScrollableDetailsOpts are the optional parameters of the constructors of
// ScrollableDetails.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.new
type ScrollableDetailsOpts struct {
	// Controller is the controller of the scrollable.
	Controller ScrollController
	// Physics are the physics of the scrollable.
	Physics ScrollPhysics
	// DecorationClipBehavior is the clip of the decorators.
	DecorationClipBehavior foundation.Opt[ui.Clip]
	// Reverse makes the vertical and horizontal constructors scroll in the
	// reverse direction (up, left).
	Reverse bool
}

// NewScrollableDetails creates the details.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.new
func NewScrollableDetails(direction painting.AxisDirection, opts ...ScrollableDetailsOpts) ScrollableDetails {
	o := firstOpts(opts)
	return ScrollableDetails{Direction: direction, Controller: o.Controller, Physics: o.Physics, DecorationClipBehavior: o.DecorationClipBehavior}
}

// NewScrollableDetailsVertical creates the details of a vertical scrollable that
// scrolls down, or up if Reverse is set.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.vertical
func NewScrollableDetailsVertical(opts ...ScrollableDetailsOpts) ScrollableDetails {
	o := firstOpts(opts)
	direction := painting.AxisDirectionDown
	if o.Reverse {
		direction = painting.AxisDirectionUp
	}
	return NewScrollableDetails(direction, o)
}

// NewScrollableDetailsHorizontal creates the details of a horizontal scrollable
// that scrolls right, or left if Reverse is set.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.horizontal
func NewScrollableDetailsHorizontal(opts ...ScrollableDetailsOpts) ScrollableDetails {
	o := firstOpts(opts)
	direction := painting.AxisDirectionRight
	if o.Reverse {
		direction = painting.AxisDirectionLeft
	}
	return NewScrollableDetails(direction, o)
}

// ScrollableDetailsCopy are the values that ScrollableDetails.CopyWith changes.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.copyWith
type ScrollableDetailsCopy struct {
	// Direction replaces the direction; AxisDirectionUnspecified keeps it.
	Direction painting.AxisDirection
	// Controller replaces the controller; nil keeps it.
	Controller ScrollController
	// Physics replaces the physics; nil keeps them.
	Physics ScrollPhysics
	// DecorationClipBehavior replaces the clip; none keeps it.
	DecorationClipBehavior foundation.Opt[ui.Clip]
}

// CopyWith returns the details with the given values changed.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.copyWith
func (d ScrollableDetails) CopyWith(c ScrollableDetailsCopy) ScrollableDetails {
	if c.Direction != painting.AxisDirectionUnspecified {
		d.Direction = c.Direction
	}
	if !isNil(c.Controller) {
		d.Controller = c.Controller
	}
	if !isNil(c.Physics) {
		d.Physics = c.Physics
	}
	if c.DecorationClipBehavior.IsSome() {
		d.DecorationClipBehavior = c.DecorationClipBehavior
	}
	return d
}

// Equal is whether the details have the same direction, controller, physics and
// clip.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.operator ==
func (d ScrollableDetails) Equal(o ScrollableDetails) bool {
	return d.Direction == o.Direction && d.Controller == o.Controller && d.Physics == o.Physics &&
		d.DecorationClipBehavior == o.DecorationClipBehavior
}

// String describes the details.
//
// Flutter: widgets/scrollable_helpers.dart ScrollableDetails.toString
func (d ScrollableDetails) String() string {
	description := []string{"axisDirection: " + d.Direction.String()}
	if !isNil(d.Controller) {
		description = append(description, "scroll controller: "+d.Controller.String())
	}
	if !isNil(d.Physics) {
		description = append(description, "scroll physics: "+d.Physics.String())
	}
	if c, ok := d.DecorationClipBehavior.Get(); ok {
		description = append(description, "decorationClipBehavior: "+c.String())
	}
	return "ScrollableDetails(" + strings.Join(description, ", ") + ")"
}

// ScrollIncrementType is the kind of increment of a scroll by keyboard or by a
// click on the scrollbar track.
//
// Flutter: widgets/scrollable_helpers.dart ScrollIncrementType
type ScrollIncrementType int

const (
	// ScrollIncrementTypeUnspecified is the zero value: a line.
	//
	// Flutter: no counterpart
	ScrollIncrementTypeUnspecified ScrollIncrementType = iota
	// ScrollIncrementTypeLine is a small increment, 50 pixels by default.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollIncrementType.line
	ScrollIncrementTypeLine
	// ScrollIncrementTypePage is a page: 80% of the viewport by default.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollIncrementType.page
	ScrollIncrementTypePage
)

// String is the name with the enum, such as "ScrollIncrementType.page".
//
// Flutter: widgets/scrollable_helpers.dart ScrollIncrementType
func (t ScrollIncrementType) String() string {
	switch t {
	case ScrollIncrementTypeLine:
		return "ScrollIncrementType.line"
	case ScrollIncrementTypePage:
		return "ScrollIncrementType.page"
	}
	return "ScrollIncrementType.unspecified"
}

// ScrollIncrementDetails describes a scroll increment: its type and the metrics
// of the scrollable.
//
// Flutter: widgets/scrollable_helpers.dart ScrollIncrementDetails
type ScrollIncrementDetails struct {
	// Type is the type of the increment.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollIncrementDetails.type
	Type ScrollIncrementType
	// Metrics are the metrics of the scrollable.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollIncrementDetails.metrics
	Metrics ScrollMetrics
}

// ScrollIncrementCalculator is the size of a scroll increment, in pixels. A
// Scrollable can have its own.
//
// Flutter: widgets/scrollable_helpers.dart ScrollIncrementCalculator
type ScrollIncrementCalculator func(details ScrollIncrementDetails) float64

// ScrollIntent is what a request to scroll (a key press, a click on the track
// of a scrollbar) is: a direction and an increment type.
//
// Flutter: widgets/scrollable_helpers.dart ScrollIntent
type ScrollIntent struct {
	IntentBase
	// Direction is the direction to scroll in.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollIntent.direction
	Direction painting.AxisDirection
	// Type is the size of the increment; the default is a line.
	//
	// Flutter: widgets/scrollable_helpers.dart ScrollIntent.type
	Type ScrollIncrementType
}

// ScrollAction is the action that scrolls the Scrollable around the primary focus
// (or the one attached to the PrimaryScrollController) for a ScrollIntent.
//
// Flutter: widgets/scrollable_helpers.dart ScrollAction
type ScrollAction struct{ ContextActionBase }

// NewScrollAction creates the action.
//
// Flutter: widgets/scrollable_helpers.dart ScrollAction.new
func NewScrollAction() *ScrollAction {
	a := &ScrollAction{}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

// IntentType is ScrollIntent.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *ScrollAction) IntentType() reflect.Type { return reflect.TypeFor[ScrollIntent]() }

// IsEnabledWithContext is whether there is a Scrollable around the context, or the
// primary scroll controller has a client.
//
// Flutter: widgets/scrollable_helpers.dart ScrollAction.isEnabled
func (a *ScrollAction) IsEnabledWithContext(_ Intent, bc BuildContext) bool {
	if bc == nil {
		return false
	}
	if ScrollableMaybeOf(bc) != nil {
		return true
	}
	primary := PrimaryScrollControllerMaybeOf(bc)
	return !isNil(primary) && primary.HasClients()
}

// InvokeWithContext scrolls the scrollable by an increment in the direction of the
// intent, with a short animation.
//
// Flutter: widgets/scrollable_helpers.dart ScrollAction.invoke
func (a *ScrollAction) InvokeWithContext(intent Intent, bc BuildContext) any {
	if bc == nil {
		panic(errors.New("ScrollActionContextMissing", "cannot scroll without a context"))
	}
	state := ScrollableMaybeOf(bc)
	if state == nil {
		primary := PrimaryScrollControllerOf(bc)
		if foundation.DebugMode && len(primary.Positions()) != 1 {
			panic(errors.New("ScrollActionPositionsInvalid",
				"a ScrollAction was invoked with the PrimaryScrollController, but more than one ScrollPosition is attached: only one ScrollPosition can be manipulated by a ScrollAction at a time",
				"positions", len(primary.Positions())))
		}
		if notificationContext := primary.Position().Context().NotificationContext(); notificationContext != nil {
			state = ScrollableMaybeOf(notificationContext)
		}
		if state == nil {
			return nil
		}
	}
	if foundation.DebugMode && !state.Position().HasPixels() {
		panic(errors.New("ScrollActionNotLaidOut", "the Scrollable must be laid out before it can be scrolled via a ScrollAction"))
	}
	if physics := state.ResolvedPhysics(); physics != nil && !physics.ShouldAcceptUserOffset(state.Position()) {
		return nil
	}
	increment := ScrollActionGetDirectionalIncrement(state, intent.(ScrollIntent))
	if increment == 0 {
		return nil
	}
	state.Position().MoveTo(state.Position().Pixels()+increment, rendering.ViewportOffsetMoveToOpts{Duration: foundation.Some(100 * time.Millisecond), Curve: animation.Curves.EaseInOut})
	return nil
}

// ScrollActionGetDirectionalIncrement is the signed number of pixels to scroll
// the scrollable by for the intent: positive in the direction of its axis. It is
// zero when the intent is on the other axis.
//
// Flutter: widgets/scrollable_helpers.dart ScrollAction.getDirectionalIncrement
func ScrollActionGetDirectionalIncrement(state *ScrollableState, intent ScrollIntent) float64 {
	if painting.AxisDirectionToAxis(intent.Direction) == painting.AxisDirectionToAxis(state.AxisDirection()) {
		increment := calculateScrollIncrement(state, intent.Type)
		if intent.Direction == state.AxisDirection() {
			return increment
		}
		return -increment
	}
	return 0
}

func calculateScrollIncrement(state *ScrollableState, typ ScrollIncrementType) float64 {
	if calc := state.Widget().incrementCalculator; calc != nil {
		return calc(ScrollIncrementDetails{Type: typ, Metrics: state.Position()})
	}
	if typ == ScrollIncrementTypePage {
		return 0.8 * state.Position().ViewportDimension()
	}
	return 50.0
}

// EdgeDraggingAutoScroller scrolls a scrollable when a dragged item gets near
// its edge, for a drag and drop or a reorder.
//
// Flutter: widgets/scrollable_helpers.dart EdgeDraggingAutoScroller
type EdgeDraggingAutoScroller struct {
	scrollable                *ScrollableState
	onScrollViewScrolled      func()
	velocityScalar            float64
	dragTargetRelatedToOrigin ui.Rect
	scrolling                 bool
}

// NewEdgeDraggingAutoScroller creates the scroller for the scrollable.
// velocityScalar is the speed of the scrolling: the larger, the faster.
// onScrollViewScrolled may be nil.
//
// Flutter: widgets/scrollable_helpers.dart EdgeDraggingAutoScroller.new
func NewEdgeDraggingAutoScroller(scrollable *ScrollableState, velocityScalar float64, onScrollViewScrolled func()) *EdgeDraggingAutoScroller {
	return &EdgeDraggingAutoScroller{scrollable: scrollable, velocityScalar: velocityScalar, onScrollViewScrolled: onScrollViewScrolled}
}

// Scrollable is the scrollable that the scroller scrolls.
//
// Flutter: widgets/scrollable_helpers.dart EdgeDraggingAutoScroller.scrollable
func (s *EdgeDraggingAutoScroller) Scrollable() *ScrollableState { return s.scrollable }

// Scrolling is whether the scroller is scrolling.
//
// Flutter: widgets/scrollable_helpers.dart EdgeDraggingAutoScroller.scrolling
func (s *EdgeDraggingAutoScroller) Scrolling() bool { return s.scrolling }

func offsetExtent(o ui.Offset, axis painting.Axis) float64 {
	if axis == painting.AxisHorizontal {
		return o.Dx
	}
	return o.Dy
}

func sizeExtent(size ui.Size, axis painting.Axis) float64 {
	if axis == painting.AxisHorizontal {
		return size.Width
	}
	return size.Height
}

// StartAutoScrollIfNecessary starts scrolling if the drag target is near an
// edge of the scrollable, and keeps going until StopAutoScroll.
//
// Flutter: widgets/scrollable_helpers.dart EdgeDraggingAutoScroller.startAutoScrollIfNecessary
func (s *EdgeDraggingAutoScroller) StartAutoScrollIfNecessary(dragTarget ui.Rect) {
	physics := s.scrollable.ResolvedPhysics()
	if physics != nil && !physics.ShouldAcceptUserOffset(s.scrollable.Position()) {
		s.StopAutoScroll()
		return
	}
	// Update the drag target rect no matter whether the scroller is scrolling.
	deltaToOrigin := s.scrollable.DeltaToScrollOrigin()
	s.dragTargetRelatedToOrigin = dragTarget.Translate(deltaToOrigin.Dx, deltaToOrigin.Dy)
	if s.scrolling {
		// The change will be picked up in the next scroll.
		return
	}
	s.scroll()
}

// StopAutoScroll stops scrolling after the current animation.
//
// Flutter: widgets/scrollable_helpers.dart EdgeDraggingAutoScroller.stopAutoScroll
func (s *EdgeDraggingAutoScroller) StopAutoScroll() { s.scrolling = false }

func (s *EdgeDraggingAutoScroller) scroll() {
	renderBox := s.scrollable.Context().FindRenderObject().(rendering.Box)
	transform := renderBox.GetTransformTo(nil)
	globalRect := transform.TransformRect(ui.NewRectFromLTWH(0, 0, renderBox.Size().Width, renderBox.Size().Height))
	transformedDragTarget := transform.TransformRect(s.dragTargetRelatedToOrigin)
	if foundation.DebugMode && !((globalRect.Size().Width+foundation.PrecisionErrorTolerance) >= transformedDragTarget.Size().Width &&
		(globalRect.Size().Height+foundation.PrecisionErrorTolerance) >= transformedDragTarget.Size().Height) {
		panic(errors.New("AutoScrollDragTargetTooLarge", "drag target size is larger than scrollable size, which may cause bouncing",
			"dragTarget", transformedDragTarget.Size().String(), "scrollable", globalRect.Size().String()))
	}
	s.scrolling = true
	var newOffset foundation.Opt[float64]
	const overDragMax = 20.0
	axisDirection := s.scrollable.AxisDirection()
	scrollDirection := painting.AxisDirectionToAxis(axisDirection)
	position := s.scrollable.Position()

	deltaToOrigin := s.scrollable.DeltaToScrollOrigin()
	viewportOrigin := globalRect.TopLeft().Translate(deltaToOrigin.Dx, deltaToOrigin.Dy)
	viewportStart := offsetExtent(viewportOrigin, scrollDirection)
	viewportEnd := viewportStart + sizeExtent(globalRect.Size(), scrollDirection)

	proxyStart := offsetExtent(s.dragTargetRelatedToOrigin.TopLeft(), scrollDirection)
	proxyEnd := offsetExtent(s.dragTargetRelatedToOrigin.BottomRight(), scrollDirection)
	switch axisDirection {
	case painting.AxisDirectionUp, painting.AxisDirectionLeft:
		if proxyEnd > viewportEnd && position.Pixels() > position.MinScrollExtent() {
			overDrag := math.Min(proxyEnd-viewportEnd, overDragMax)
			newOffset = foundation.Some(math.Max(position.MinScrollExtent(), position.Pixels()-overDrag))
		} else if proxyStart < viewportStart && position.Pixels() < position.MaxScrollExtent() {
			overDrag := math.Min(viewportStart-proxyStart, overDragMax)
			newOffset = foundation.Some(math.Min(position.MaxScrollExtent(), position.Pixels()+overDrag))
		}
	default:
		if proxyStart < viewportStart && position.Pixels() > position.MinScrollExtent() {
			overDrag := math.Min(viewportStart-proxyStart, overDragMax)
			newOffset = foundation.Some(math.Max(position.MinScrollExtent(), position.Pixels()-overDrag))
		} else if proxyEnd > viewportEnd && position.Pixels() < position.MaxScrollExtent() {
			overDrag := math.Min(proxyEnd-viewportEnd, overDragMax)
			newOffset = foundation.Some(math.Min(position.MaxScrollExtent(), position.Pixels()+overDrag))
		}
	}
	if !newOffset.IsSome() || math.Abs(newOffset.Value()-position.Pixels()) < 1.0 {
		// Drag should not trigger scroll.
		s.scrolling = false
		return
	}
	duration := time.Duration(math.Round(1000/s.velocityScalar)) * time.Millisecond
	position.AnimateTo(newOffset.Value(), duration, animation.Curves.Linear).Then(func() {
		if s.onScrollViewScrolled != nil {
			s.onScrollViewScrolled()
		}
		if s.scrolling {
			s.scroll()
		}
	})
}
