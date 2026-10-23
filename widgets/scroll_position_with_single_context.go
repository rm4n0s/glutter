package widgets

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
)

// ScrollPositionWithSingleContext is a scroll position for a scrollable that has
// one ScrollContext: it handles the activities, the user's drags and holds, and
// the animations. The ScrollController makes it.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext
type ScrollPositionWithSingleContext struct {
	ScrollPositionBase
	heldPreviousVelocity float64
	userScrollDirection  rendering.ScrollDirection
	currentDrag          *ScrollDragController
}

// ScrollPositionWithSingleContextOpts are the optional parameters of
// NewScrollPositionWithSingleContext.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.new
type ScrollPositionWithSingleContextOpts struct {
	// InitialPixels is the position to start at if there is no old position or
	// saved offset; the default is 0.
	InitialPixels foundation.Opt[float64]
	// NoInitialPixels leaves the position without pixels until the viewport sets
	// them, instead of using InitialPixels.
	NoInitialPixels bool
	// KeepScrollOffset is whether the position saves its offset in the
	// PageStorage; the default is true.
	KeepScrollOffset foundation.Opt[bool]
	// OldPosition is the position this one replaces.
	OldPosition ScrollPosition
	// DebugLabel names the position in debug output.
	DebugLabel string
}

// NewScrollPositionWithSingleContext creates the position.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.new
func NewScrollPositionWithSingleContext(physics ScrollPhysics, context ScrollContext, opts ...ScrollPositionWithSingleContextOpts) *ScrollPositionWithSingleContext {
	p := &ScrollPositionWithSingleContext{}
	InitScrollPositionWithSingleContext(p, p, physics, context, opts...)
	return p
}

// InitScrollPositionWithSingleContext sets up the position; self is the outer
// position, for a type that embeds ScrollPositionWithSingleContext. The fields of
// the outer type that its Absorb and RestoreScrollOffset methods need must be set
// before the call, as they run in it.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.new
func InitScrollPositionWithSingleContext(p *ScrollPositionWithSingleContext, self ScrollPosition, physics ScrollPhysics, context ScrollContext, opts ...ScrollPositionWithSingleContextOpts) {
	o := firstOpts(opts)
	p.userScrollDirection = rendering.ScrollDirectionIdle
	InitScrollPositionBase(&p.ScrollPositionBase, self, physics, context, ScrollPositionOpts{
		KeepScrollOffset: o.KeepScrollOffset, OldPosition: o.OldPosition, DebugLabel: o.DebugLabel})
	if !p.HasPixels() && !o.NoInitialPixels {
		p.CorrectPixels(o.InitialPixels.OrElse(0))
	}
	if p.activity == nil {
		p.GoIdle()
	}
}

// SetPixels changes the position; in debug builds it panics with the tag
// ScrollPositionNotScrolling if the activity is not scrolling.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.setPixels
func (p *ScrollPositionWithSingleContext) SetPixels(newPixels float64) float64 {
	if foundation.DebugMode && !p.activity.IsScrolling() {
		panic(errors.New(tagScrollPositionNotScrolling, "setPixels was called while the activity of the position is not scrolling"))
	}
	return p.ScrollPositionBase.SetPixels(newPixels)
}

// Absorb takes over the state of the old position, and its drag and scroll
// direction if it is of the same type.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.absorb
func (p *ScrollPositionWithSingleContext) Absorb(other ScrollPosition) {
	p.ScrollPositionBase.Absorb(other)
	o, ok := other.(*ScrollPositionWithSingleContext)
	if !ok {
		p.GoIdle()
		return
	}
	p.activity.UpdateDelegate(p)
	p.userScrollDirection = o.userScrollDirection
	if foundation.DebugMode && p.currentDrag != nil {
		panic(errors.New(tagScrollPositionDragInvalid, "the scroll position has a drag already"))
	}
	if o.currentDrag != nil {
		p.currentDrag = o.currentDrag
		p.currentDrag.UpdateDelegate(p)
		o.currentDrag = nil
	}
}

// ApplyNewDimensions tells the scroll context whether the user can drag.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.applyNewDimensions
func (p *ScrollPositionWithSingleContext) ApplyNewDimensions() {
	p.ScrollPositionBase.ApplyNewDimensions()
	p.context.SetCanDrag(p.physics.ShouldAcceptUserOffset(p))
}

// BeginActivity ends the current activity and starts the new one, with the
// delegate of this position.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.beginActivity
func (p *ScrollPositionWithSingleContext) BeginActivity(newActivity ScrollActivity) {
	p.heldPreviousVelocity = 0
	if isNil(newActivity) {
		return
	}
	if foundation.DebugMode && newActivity.Delegate() != ScrollActivityDelegate(p) {
		panic(errors.New(tagScrollActivityDelegateInvalid, "the delegate of the new activity must be the position that begins it"))
	}
	p.ScrollPositionBase.BeginActivity(newActivity)
	if p.currentDrag != nil {
		p.currentDrag.Dispose()
	}
	p.currentDrag = nil
	if !p.activity.IsScrolling() {
		p.UpdateUserScrollDirection(rendering.ScrollDirectionIdle)
	}
}

// ApplyUserOffset updates the scroll direction and moves the position by the
// drag delta, after the physics.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.applyUserOffset
func (p *ScrollPositionWithSingleContext) ApplyUserOffset(delta float64) {
	if delta > 0 {
		p.UpdateUserScrollDirection(rendering.ScrollDirectionForward)
	} else {
		p.UpdateUserScrollDirection(rendering.ScrollDirectionReverse)
	}
	p.SetPixels(p.Pixels() - p.physics.ApplyPhysicsToUserOffset(p, delta))
}

// GoIdle starts an idle activity.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.goIdle
func (p *ScrollPositionWithSingleContext) GoIdle() { p.BeginActivity(NewIdleScrollActivity(p)) }

// GoBallistic starts a ballistic activity with the simulation of the physics, or
// an idle one if there is none.
//
// Deviation: the physics receive the outermost position (self), so that physics
// of a custom position type can recognise it; Dart passes `this`, which is virtual.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.goBallistic
func (p *ScrollPositionWithSingleContext) GoBallistic(velocity float64) {
	if foundation.DebugMode && !p.HasPixels() {
		panic(errors.New(tagScrollMetricsValueMissing, "goBallistic was called before the position has pixels", "value", "pixels"))
	}
	simulation := p.physics.CreateBallisticSimulation(p.self, velocity)
	if !isNil(simulation) {
		p.BeginActivity(NewBallisticScrollActivity(p, simulation, p.context.Vsync(), p.self.ShouldIgnorePointer(), p.post))
	} else {
		p.GoIdle()
	}
}

// UserScrollDirection is the direction in which the user is scrolling.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.userScrollDirection
func (p *ScrollPositionWithSingleContext) UserScrollDirection() rendering.ScrollDirection {
	return p.userScrollDirection
}

// UpdateUserScrollDirection sets the direction, and dispatches a
// UserScrollNotification if it changed.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.updateUserScrollDirection
func (p *ScrollPositionWithSingleContext) UpdateUserScrollDirection(value rendering.ScrollDirection) {
	if p.userScrollDirection == value {
		return
	}
	p.userScrollDirection = value
	p.self.DidUpdateScrollDirection(value)
}

// AnimateTo animates the position to the value, or jumps if it is already within
// the tolerance. The future completes when the activity is disposed.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.animateTo
func (p *ScrollPositionWithSingleContext) AnimateTo(to float64, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	if physics.NearEqual(to, p.Pixels(), p.physics.ToleranceFor(p).Distance) {
		// Skip the animation, go straight to the position as the layout
		// might not be ready.
		p.self.JumpTo(to)
		return scheduler.NewTickerFutureComplete()
	}
	activity := NewDrivenScrollActivity(p, DrivenScrollActivityOpts{
		From: p.Pixels(), To: to, Duration: duration, Curve: curve, Vsync: p.context.Vsync(), Post: p.post})
	p.BeginActivity(activity)
	return activity.Done()
}

// JumpTo changes the position to the value at once, with the notifications of a
// scroll, then goes ballistic with no velocity.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.jumpTo
func (p *ScrollPositionWithSingleContext) JumpTo(value float64) {
	p.GoIdle()
	if p.Pixels() != value {
		oldPixels := p.Pixels()
		p.self.ForcePixels(value)
		p.self.DidStartScroll()
		p.self.DidUpdateScrollPositionBy(p.Pixels() - oldPixels)
		p.self.DidEndScroll()
	}
	p.GoBallistic(0)
}

// PointerScroll scrolls by a pointer scroll event, as a jump of delta pixels
// that is clamped to the extents.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.pointerScroll
func (p *ScrollPositionWithSingleContext) PointerScroll(delta float64) {
	// If an update is made to pointer scrolling here, consider if the same
	// (or similar) change should be made in
	// _NestedScrollCoordinator.pointerScroll.
	if delta == 0 {
		p.GoBallistic(0)
		return
	}
	targetPixels := math.Min(math.Max(p.Pixels()+delta, p.MinScrollExtent()), p.MaxScrollExtent())
	if targetPixels != p.Pixels() {
		p.GoIdle()
		if -delta > 0 {
			p.UpdateUserScrollDirection(rendering.ScrollDirectionForward)
		} else {
			p.UpdateUserScrollDirection(rendering.ScrollDirectionReverse)
		}
		oldPixels := p.Pixels()
		// Set the notifier before calling force pixels.
		// This is set to false again after going ballistic below.
		p.isScrollingNotifier.SetValue(true)
		p.self.ForcePixels(targetPixels)
		p.self.DidStartScroll()
		p.self.DidUpdateScrollPositionBy(targetPixels - oldPixels)
		p.self.DidEndScroll()
		p.GoBallistic(0)
	}
}

// JumpToWithoutSettling jumps to the value without going ballistic afterwards.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.jumpToWithoutSettling
func (p *ScrollPositionWithSingleContext) JumpToWithoutSettling(value float64) {
	p.GoIdle()
	if p.Pixels() != value {
		oldPixels := p.Pixels()
		p.self.ForcePixels(value)
		p.self.DidStartScroll()
		p.self.DidUpdateScrollPositionBy(p.Pixels() - oldPixels)
		p.self.DidEndScroll()
	}
}

// Hold starts a hold activity; the callback is called when it is disposed.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.hold
func (p *ScrollPositionWithSingleContext) Hold(holdCancelCallback func()) ScrollHoldController {
	previousVelocity := p.activity.Velocity()
	holdActivity := NewHoldScrollActivity(p, holdCancelCallback)
	p.BeginActivity(holdActivity)
	p.heldPreviousVelocity = previousVelocity
	return holdActivity
}

// Drag starts a drag activity for the details; the callback is called when the
// drag ends.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.drag
func (p *ScrollPositionWithSingleContext) Drag(details gestures.DragStartDetails, dragCancelCallback func()) gestures.Drag {
	drag := NewScrollDragController(p, details, ScrollDragControllerOpts{
		OnDragCanceled:               dragCancelCallback,
		CarriedVelocity:              foundation.Some(p.physics.CarriedMomentum(p.heldPreviousVelocity)),
		MotionStartDistanceThreshold: p.physics.DragStartDistanceMotionThreshold(),
	})
	p.BeginActivity(NewDragScrollActivity(p, drag))
	if foundation.DebugMode && p.currentDrag != nil {
		panic(errors.New(tagScrollPositionDragInvalid, "the scroll position has a drag already"))
	}
	p.currentDrag = drag
	return drag
}

// Dispose disposes the drag and the position.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.dispose
func (p *ScrollPositionWithSingleContext) Dispose() {
	if p.currentDrag != nil {
		p.currentDrag.Dispose()
	}
	p.currentDrag = nil
	p.ScrollPositionBase.Dispose()
}

// DebugFillDescription appends the context, the physics, the activity and the
// scroll direction.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.debugFillDescription
func (p *ScrollPositionWithSingleContext) DebugFillDescription(description []string) []string {
	description = p.ScrollPositionBase.DebugFillDescription(description)
	return append(description, foundation.RuntimeTypeName(p.context), p.physics.String(), p.activity.String(), p.userScrollDirection.String())
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagScrollActivityDelegateInvalid = "ScrollActivityDelegateInvalid"
	tagScrollPositionDragInvalid     = "ScrollPositionDragInvalid"
	tagScrollPositionNotScrolling    = "ScrollPositionNotScrolling"
)
