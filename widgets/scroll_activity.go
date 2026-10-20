package widgets

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollActivityDelegate is the interface a ScrollActivity uses to change the
// scroll position, and to hand over to another activity. The scroll position
// implements it.
//
// Flutter: widgets/scroll_activity.dart ScrollActivityDelegate
type ScrollActivityDelegate interface {
	// AxisDirection is the direction in which the scroll offset increases.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivityDelegate.axisDirection
	AxisDirection() painting.AxisDirection
	// SetPixels updates the scroll position to the given pixel value, and returns
	// the overscroll: the part of the change that was not applied.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivityDelegate.setPixels
	SetPixels(pixels float64) float64
	// ApplyUserOffset updates the scroll position by a user's drag of delta
	// pixels, after the physics have applied their friction.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivityDelegate.applyUserOffset
	ApplyUserOffset(delta float64)
	// GoIdle ends the current activity and starts an idle one.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivityDelegate.goIdle
	GoIdle()
	// GoBallistic ends the current activity and starts a ballistic one with the
	// velocity.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivityDelegate.goBallistic
	GoBallistic(velocity float64)
}

// ScrollActivity is the activity a scroll position is in: idle, held, dragged,
// carried by a simulation, or driven by an animation. It changes the position
// through its delegate and dispatches the notifications.
//
// Implement it by embedding ScrollActivityBase (InitScrollActivityBase).
//
// Flutter: widgets/scroll_activity.dart ScrollActivity
type ScrollActivity interface {
	// Delegate is the object that this activity changes.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.delegate
	Delegate() ScrollActivityDelegate
	// UpdateDelegate changes the delegate; it panics with the tag
	// ScrollActivityDelegateSame in debug builds when it is the same one.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.updateDelegate
	UpdateDelegate(delegate ScrollActivityDelegate)
	// ResetActivity is called when the position that owns the activity is
	// replaced by one of another type.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.resetActivity
	ResetActivity()
	// DispatchScrollStartNotification dispatches a ScrollStartNotification.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchScrollStartNotification
	DispatchScrollStartNotification(metrics ScrollMetrics, bc BuildContext)
	// DispatchScrollUpdateNotification dispatches a ScrollUpdateNotification.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchScrollUpdateNotification
	DispatchScrollUpdateNotification(metrics ScrollMetrics, bc BuildContext, scrollDelta float64)
	// DispatchOverscrollNotification dispatches an OverscrollNotification.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchOverscrollNotification
	DispatchOverscrollNotification(metrics ScrollMetrics, bc BuildContext, overscroll float64)
	// DispatchScrollEndNotification dispatches a ScrollEndNotification.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchScrollEndNotification
	DispatchScrollEndNotification(metrics ScrollMetrics, bc BuildContext)
	// ApplyNewDimensions is called when the dimensions of the content or the
	// viewport changed.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.applyNewDimensions
	ApplyNewDimensions()
	// ShouldIgnorePointer is whether the scrollable ignores pointer events while
	// this activity runs.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.shouldIgnorePointer
	ShouldIgnorePointer() bool
	// IsScrolling is whether the scrollable is in motion.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.isScrolling
	IsScrolling() bool
	// Velocity is the velocity of the scrollable, in pixels per second.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.velocity
	Velocity() float64
	// Dispose is called when the activity is replaced or the position is
	// disposed.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.dispose
	Dispose()
	// String describes the activity.
	//
	// Flutter: widgets/scroll_activity.dart ScrollActivity.toString
	String() string
}

// ScrollActivityBase implements the parts of ScrollActivity that every activity
// shares: the delegate, the notifications and the disposed flag. Embed it and
// call InitScrollActivityBase.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity
type ScrollActivityBase struct {
	self     ScrollActivity
	delegate ScrollActivityDelegate
	disposed bool
}

// InitScrollActivityBase sets the outer value and the delegate.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.new
func InitScrollActivityBase(b *ScrollActivityBase, self ScrollActivity, delegate ScrollActivityDelegate) {
	b.self, b.delegate = self, delegate
}

// Delegate is the object that this activity changes.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.delegate
func (b *ScrollActivityBase) Delegate() ScrollActivityDelegate { return b.delegate }

// IsDisposed is whether Dispose was called.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity._isDisposed
func (b *ScrollActivityBase) IsDisposed() bool { return b.disposed }

// UpdateDelegate changes the delegate.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.updateDelegate
func (b *ScrollActivityBase) UpdateDelegate(delegate ScrollActivityDelegate) {
	if foundation.DebugMode && b.delegate == delegate {
		panic(errors.New("ScrollActivityDelegateSame", "the delegate of a scroll activity was updated with the delegate it already has"))
	}
	b.delegate = delegate
}

// ResetActivity does nothing.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.resetActivity
func (b *ScrollActivityBase) ResetActivity() {}

// DispatchScrollStartNotification dispatches a ScrollStartNotification without
// drag details.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchScrollStartNotification
func (b *ScrollActivityBase) DispatchScrollStartNotification(metrics ScrollMetrics, bc BuildContext) {
	DispatchNotification(bc, NewScrollStartNotification(metrics, bc, foundation.Opt[gestures.DragStartDetails]{}))
}

// DispatchScrollUpdateNotification dispatches a ScrollUpdateNotification without
// drag details.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchScrollUpdateNotification
func (b *ScrollActivityBase) DispatchScrollUpdateNotification(metrics ScrollMetrics, bc BuildContext, scrollDelta float64) {
	DispatchNotification(bc, NewScrollUpdateNotification(metrics, bc, ScrollUpdateNotificationOpts{ScrollDelta: foundation.Some(scrollDelta)}))
}

// DispatchOverscrollNotification dispatches an OverscrollNotification without
// drag details.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchOverscrollNotification
func (b *ScrollActivityBase) DispatchOverscrollNotification(metrics ScrollMetrics, bc BuildContext, overscroll float64) {
	DispatchNotification(bc, NewOverscrollNotification(metrics, bc, overscroll))
}

// DispatchScrollEndNotification dispatches a ScrollEndNotification without drag
// details.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.dispatchScrollEndNotification
func (b *ScrollActivityBase) DispatchScrollEndNotification(metrics ScrollMetrics, bc BuildContext) {
	DispatchNotification(bc, NewScrollEndNotification(metrics, bc, foundation.Opt[gestures.DragEndDetails]{}))
}

// ApplyNewDimensions does nothing.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.applyNewDimensions
func (b *ScrollActivityBase) ApplyNewDimensions() {}

// Dispose marks the activity disposed.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.dispose
func (b *ScrollActivityBase) Dispose() { b.disposed = true }

// String is the identity of the activity.
//
// Flutter: widgets/scroll_activity.dart ScrollActivity.toString
func (b *ScrollActivityBase) String() string { return foundation.DescribeIdentity(b.self) }

// IdleScrollActivity is the activity of a scroll position that is not scrolling.
//
// Flutter: widgets/scroll_activity.dart IdleScrollActivity
type IdleScrollActivity struct {
	ScrollActivityBase
}

// NewIdleScrollActivity creates the activity.
//
// Flutter: widgets/scroll_activity.dart IdleScrollActivity.new
func NewIdleScrollActivity(delegate ScrollActivityDelegate) *IdleScrollActivity {
	a := &IdleScrollActivity{}
	InitScrollActivityBase(&a.ScrollActivityBase, a, delegate)
	return a
}

// ApplyNewDimensions starts a ballistic activity with no velocity, so that a
// position that is out of range goes back.
//
// Flutter: widgets/scroll_activity.dart IdleScrollActivity.applyNewDimensions
func (a *IdleScrollActivity) ApplyNewDimensions() { a.delegate.GoBallistic(0) }

// ShouldIgnorePointer is false.
//
// Flutter: widgets/scroll_activity.dart IdleScrollActivity.shouldIgnorePointer
func (a *IdleScrollActivity) ShouldIgnorePointer() bool { return false }

// IsScrolling is false.
//
// Flutter: widgets/scroll_activity.dart IdleScrollActivity.isScrolling
func (a *IdleScrollActivity) IsScrolling() bool { return false }

// Velocity is zero.
//
// Flutter: widgets/scroll_activity.dart IdleScrollActivity.velocity
func (a *IdleScrollActivity) Velocity() float64 { return 0 }

// ScrollHoldController is the handle of a hold on a scroll position: the user
// has touched down and the scrollable stops.
//
// Flutter: widgets/scroll_activity.dart ScrollHoldController
type ScrollHoldController interface {
	// Cancel releases the hold; the scrollable goes ballistic with no velocity.
	//
	// Flutter: widgets/scroll_activity.dart ScrollHoldController.cancel
	Cancel()
}

// HoldScrollActivity is the activity of a position that the user holds with a
// finger down, before the drag starts.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity
type HoldScrollActivity struct {
	ScrollActivityBase
	onHoldCanceled func()
}

// NewHoldScrollActivity creates the activity; onHoldCanceled may be nil and is
// called when the activity is disposed.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity.new
func NewHoldScrollActivity(delegate ScrollActivityDelegate, onHoldCanceled func()) *HoldScrollActivity {
	a := &HoldScrollActivity{onHoldCanceled: onHoldCanceled}
	InitScrollActivityBase(&a.ScrollActivityBase, a, delegate)
	return a
}

// ShouldIgnorePointer is false.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity.shouldIgnorePointer
func (a *HoldScrollActivity) ShouldIgnorePointer() bool { return false }

// IsScrolling is false.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity.isScrolling
func (a *HoldScrollActivity) IsScrolling() bool { return false }

// Velocity is zero.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity.velocity
func (a *HoldScrollActivity) Velocity() float64 { return 0 }

// Cancel releases the hold.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity.cancel
func (a *HoldScrollActivity) Cancel() { a.delegate.GoBallistic(0) }

// Dispose calls the hold canceled callback.
//
// Flutter: widgets/scroll_activity.dart HoldScrollActivity.dispose
func (a *HoldScrollActivity) Dispose() {
	if a.onHoldCanceled != nil {
		a.onHoldCanceled()
	}
	a.ScrollActivityBase.Dispose()
}

// Constants of ScrollDragController.
const (
	// ScrollDragControllerMomentumRetainStationaryDurationThreshold is how long
	// the pointer may be stationary for the drag to keep the momentum it
	// carried over from a fling.
	//
	// Flutter: widgets/scroll_activity.dart ScrollDragController.momentumRetainStationaryDurationThreshold
	ScrollDragControllerMomentumRetainStationaryDurationThreshold = 20 * time.Millisecond
	// ScrollDragControllerMomentumRetainVelocityThresholdFactor is the fraction
	// of the carried velocity the new fling must reach to add it.
	//
	// Flutter: widgets/scroll_activity.dart ScrollDragController.momentumRetainVelocityThresholdFactor
	ScrollDragControllerMomentumRetainVelocityThresholdFactor = 0.5
	// ScrollDragControllerMotionStoppedDurationThreshold is how long the pointer
	// must be stationary for the motion to count as stopped.
	//
	// Flutter: widgets/scroll_activity.dart ScrollDragController.motionStoppedDurationThreshold
	ScrollDragControllerMotionStoppedDurationThreshold = 50 * time.Millisecond

	bigThresholdBreakDistance = 24.0
)

// ScrollDragController is the Drag that a scroll position hands to the gesture
// recognizer: it turns the updates of the drag into scroll offsets.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController
type ScrollDragController struct {
	delegate                     ScrollActivityDelegate
	onDragCanceled               func()
	carriedVelocity              foundation.Opt[float64]
	motionStartDistanceThreshold foundation.Opt[float64]

	lastNonStationaryTimestamp foundation.Opt[time.Duration]
	retainMomentum             bool
	offsetSinceLastStop        foundation.Opt[float64]
	kind                       foundation.Opt[ui.PointerDeviceKind]
	lastDetails                any
}

// ScrollDragControllerOpts are the optional parameters of
// NewScrollDragController.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.new
type ScrollDragControllerOpts struct {
	// OnDragCanceled is called when the controller is disposed.
	OnDragCanceled func()
	// CarriedVelocity is the velocity the scrollable had when the finger came
	// down, to be added to the fling.
	CarriedVelocity foundation.Opt[float64]
	// MotionStartDistanceThreshold is the distance the pointer must move before
	// the scrollable moves, after the motion stopped; it must be positive.
	MotionStartDistanceThreshold foundation.Opt[float64]
}

// NewScrollDragController creates the controller for the drag that started with
// details. In debug builds it panics with the tag
// ScrollDragThresholdInvalid when the motion start distance threshold is not
// positive.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.new
func NewScrollDragController(delegate ScrollActivityDelegate, details gestures.DragStartDetails, opts ...ScrollDragControllerOpts) *ScrollDragController {
	o := firstOpts(opts)
	if t, ok := o.MotionStartDistanceThreshold.Get(); ok && foundation.DebugMode && !(t > 0) {
		panic(errors.New("ScrollDragThresholdInvalid", "motionStartDistanceThreshold must be a positive number or none", "motionStartDistanceThreshold", t))
	}
	c := &ScrollDragController{
		delegate:                     delegate,
		onDragCanceled:               o.OnDragCanceled,
		carriedVelocity:              o.CarriedVelocity,
		motionStartDistanceThreshold: o.MotionStartDistanceThreshold,
		lastDetails:                  details,
		retainMomentum:               o.CarriedVelocity.IsSome() && o.CarriedVelocity.Value() != 0,
		lastNonStationaryTimestamp:   details.SourceTimeStamp,
		kind:                         details.Kind,
	}
	if o.MotionStartDistanceThreshold.IsSome() {
		c.offsetSinceLastStop = foundation.Some(0.0)
	}
	return c
}

// Delegate is the object that the drag changes.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.delegate
func (c *ScrollDragController) Delegate() ScrollActivityDelegate { return c.delegate }

// UpdateDelegate changes the delegate; it panics with the tag
// ScrollActivityDelegateSame in debug builds when it is the same one.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.updateDelegate
func (c *ScrollDragController) UpdateDelegate(delegate ScrollActivityDelegate) {
	if foundation.DebugMode && c.delegate == delegate {
		panic(errors.New("ScrollActivityDelegateSame", "the delegate of a scroll drag controller was updated with the delegate it already has"))
	}
	c.delegate = delegate
}

func (c *ScrollDragController) reversed() bool {
	return painting.AxisDirectionIsReversed(c.delegate.AxisDirection())
}

func (c *ScrollDragController) maybeLoseMomentum(offset float64, timestamp foundation.Opt[time.Duration]) {
	if c.retainMomentum && offset == 0 &&
		(!timestamp.IsSome() || // If drag event has no timestamp, we lose momentum.
			timestamp.Value()-c.lastNonStationaryTimestamp.Value() > ScrollDragControllerMomentumRetainStationaryDurationThreshold) {
		// If pointer is stationary for too long, we lose momentum.
		c.retainMomentum = false
	}
}

// adjustForScrollStartThreshold returns the offset to scroll by, after holding
// back the motion until the pointer has moved past the threshold.
func (c *ScrollDragController) adjustForScrollStartThreshold(offset float64, timestamp foundation.Opt[time.Duration]) float64 {
	if !timestamp.IsSome() {
		// If we can't track time, we can't apply thresholds.
		return offset
	}
	if offset == 0 {
		if c.motionStartDistanceThreshold.IsSome() && !c.offsetSinceLastStop.IsSome() &&
			timestamp.Value()-c.lastNonStationaryTimestamp.Value() > ScrollDragControllerMotionStoppedDurationThreshold {
			// Enforce a new threshold.
			c.offsetSinceLastStop = foundation.Some(0.0)
		}
		// Not moving can't break threshold.
		return 0
	}
	if !c.offsetSinceLastStop.IsSome() {
		// Already in motion or no threshold behavior configured such as for
		// Android. Allow transparent offset transmission.
		return offset
	}
	c.offsetSinceLastStop = foundation.Some(c.offsetSinceLastStop.Value() + offset)
	threshold := c.motionStartDistanceThreshold.Value()
	if math.Abs(c.offsetSinceLastStop.Value()) > threshold {
		// Threshold broken.
		c.offsetSinceLastStop = foundation.Opt[float64]{}
		if math.Abs(offset) > bigThresholdBreakDistance {
			// This is heuristically a very deliberate fling. Leave the motion
			// unaffected.
			return offset
		}
		// This is a normal speed motion. Smooth the starting of the motion.
		return math.Min(threshold/3.0, math.Abs(offset)) * math.Copysign(1, offset)
	}
	return 0
}

// Update scrolls by the primary delta of the drag.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.update
func (c *ScrollDragController) Update(details gestures.DragUpdateDetails) {
	if foundation.DebugMode && !details.PrimaryDelta.IsSome() {
		panic(errors.New("ScrollDragPrimaryDeltaMissing", "the details of a scroll drag update must have a primary delta"))
	}
	c.lastDetails = details
	offset := details.PrimaryDelta.Value()
	if offset != 0 {
		c.lastNonStationaryTimestamp = details.SourceTimeStamp
	}
	c.maybeLoseMomentum(offset, details.SourceTimeStamp)
	offset = c.adjustForScrollStartThreshold(offset, details.SourceTimeStamp)
	if offset == 0 {
		return
	}
	if c.reversed() {
		offset = -offset
	}
	c.delegate.ApplyUserOffset(offset)
}

// End goes ballistic with the velocity of the drag, plus the carried momentum.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.end
func (c *ScrollDragController) End(details gestures.DragEndDetails) {
	if foundation.DebugMode && !details.PrimaryVelocity.IsSome() {
		panic(errors.New("ScrollDragPrimaryVelocityMissing", "the details of a scroll drag end must have a primary velocity"))
	}
	// We negate the velocity here because if the touch is moving downwards, the
	// scroll has to move upwards. It's the same reason that update() above
	// negates the delta before applying it to the scroll offset.
	velocity := -details.PrimaryVelocity.Value()
	if c.reversed() {
		// Invert the velocity if the scroll is reversed.
		velocity = -velocity
	}
	c.lastDetails = details

	if c.retainMomentum {
		// Build momentum only if dragging in the same direction.
		carried := c.carriedVelocity.Value()
		isFlingingInSameDirection := sign(velocity) == sign(carried)
		// Build momentum only if the velocity of the last drag was not
		// substantially lower than the carried momentum.
		isVelocityNotSubstantiallyLessThanCarriedMomentum := math.Abs(velocity) > math.Abs(carried)*ScrollDragControllerMomentumRetainVelocityThresholdFactor
		if isFlingingInSameDirection && isVelocityNotSubstantiallyLessThanCarriedMomentum {
			velocity += carried
		}
	}
	c.delegate.GoBallistic(velocity)
}

func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// Cancel goes ballistic with no velocity.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.cancel
func (c *ScrollDragController) Cancel() { c.delegate.GoBallistic(0) }

// Dispose forgets the last details and calls the drag canceled callback.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.dispose
func (c *ScrollDragController) Dispose() {
	c.lastDetails = nil
	if c.onDragCanceled != nil {
		c.onDragCanceled()
	}
}

// LastDetails is the details of the last event of the drag: a DragStartDetails,
// a DragUpdateDetails or a DragEndDetails.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.lastDetails
func (c *ScrollDragController) LastDetails() any { return c.lastDetails }

// String is the identity of the controller.
//
// Flutter: widgets/scroll_activity.dart ScrollDragController.toString
func (c *ScrollDragController) String() string { return foundation.DescribeIdentity(c) }

// DragScrollActivity is the activity of a position that the user drags.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity
type DragScrollActivity struct {
	ScrollActivityBase
	controller *ScrollDragController
}

// NewDragScrollActivity creates the activity for the drag controller.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.new
func NewDragScrollActivity(delegate ScrollActivityDelegate, controller *ScrollDragController) *DragScrollActivity {
	a := &DragScrollActivity{controller: controller}
	InitScrollActivityBase(&a.ScrollActivityBase, a, delegate)
	return a
}

// DispatchScrollStartNotification dispatches a ScrollStartNotification with the
// drag details.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.dispatchScrollStartNotification
func (a *DragScrollActivity) DispatchScrollStartNotification(metrics ScrollMetrics, bc BuildContext) {
	details, _ := a.controller.LastDetails().(gestures.DragStartDetails)
	if foundation.DebugMode {
		if _, ok := a.controller.LastDetails().(gestures.DragStartDetails); !ok {
			panic(errors.New("ScrollDragDetailsUnexpected", "the last details of the drag are not DragStartDetails",
				"details", a.controller.LastDetails()))
		}
	}
	DispatchNotification(bc, NewScrollStartNotification(metrics, bc, foundation.Some(details)))
}

// DispatchScrollUpdateNotification dispatches a ScrollUpdateNotification with
// the drag details.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.dispatchScrollUpdateNotification
func (a *DragScrollActivity) DispatchScrollUpdateNotification(metrics ScrollMetrics, bc BuildContext, scrollDelta float64) {
	details, ok := a.controller.LastDetails().(gestures.DragUpdateDetails)
	if foundation.DebugMode && !ok {
		panic(errors.New("ScrollDragDetailsUnexpected", "the last details of the drag are not DragUpdateDetails",
			"details", a.controller.LastDetails()))
	}
	DispatchNotification(bc, NewScrollUpdateNotification(metrics, bc, ScrollUpdateNotificationOpts{
		ScrollDelta: foundation.Some(scrollDelta), DragDetails: foundation.Some(details)}))
}

// DispatchOverscrollNotification dispatches an OverscrollNotification with the
// drag details.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.dispatchOverscrollNotification
func (a *DragScrollActivity) DispatchOverscrollNotification(metrics ScrollMetrics, bc BuildContext, overscroll float64) {
	details, ok := a.controller.LastDetails().(gestures.DragUpdateDetails)
	if foundation.DebugMode && !ok {
		panic(errors.New("ScrollDragDetailsUnexpected", "the last details of the drag are not DragUpdateDetails",
			"details", a.controller.LastDetails()))
	}
	DispatchNotification(bc, NewOverscrollNotification(metrics, bc, overscroll, OverscrollNotificationOpts{DragDetails: foundation.Some(details)}))
}

// DispatchScrollEndNotification dispatches a ScrollEndNotification with the drag
// details if the drag ended.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.dispatchScrollEndNotification
func (a *DragScrollActivity) DispatchScrollEndNotification(metrics ScrollMetrics, bc BuildContext) {
	// We might not have DragEndDetails yet if we're being called from beginActivity.
	var details foundation.Opt[gestures.DragEndDetails]
	if d, ok := a.controller.LastDetails().(gestures.DragEndDetails); ok {
		details = foundation.Some(d)
	}
	DispatchNotification(bc, NewScrollEndNotification(metrics, bc, details))
}

// ShouldIgnorePointer is true, except for a drag with a trackpad.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.shouldIgnorePointer
func (a *DragScrollActivity) ShouldIgnorePointer() bool {
	// DragScrollActivity is not independently changing the position and does not
	// depend on the pointer ignoring.
	if a.controller == nil {
		return true
	}
	kind, ok := a.controller.kind.Get()
	return !ok || kind != ui.PointerDeviceKindTrackpad
}

// IsScrolling is true.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.isScrolling
func (a *DragScrollActivity) IsScrolling() bool { return true }

// Velocity is zero: the drag moves the position, not a velocity.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.velocity
func (a *DragScrollActivity) Velocity() float64 { return 0 }

// Dispose forgets the controller.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.dispose
func (a *DragScrollActivity) Dispose() {
	a.controller = nil
	a.ScrollActivityBase.Dispose()
}

// String is the identity of the activity and its controller.
//
// Flutter: widgets/scroll_activity.dart DragScrollActivity.toString
func (a *DragScrollActivity) String() string {
	if a.controller == nil {
		return foundation.DescribeIdentity(a) + "(null)"
	}
	return foundation.DescribeIdentity(a) + "(" + a.controller.String() + ")"
}

// BallisticScrollActivity is the activity of a position that a simulation
// carries, after a fling or to bring an overscroll back.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity
type BallisticScrollActivity struct {
	ScrollActivityBase
	controller          *animation.AnimationController
	shouldIgnorePointer bool
	listener            foundation.ListenerHandle
}

// NewBallisticScrollActivity creates the activity that runs the simulation with
// a controller from vsync. shouldIgnorePointer is the value of
// ShouldIgnorePointer, and post runs a function after the code running now (the
// microtask of Dart; nil runs it at once).
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.new
func NewBallisticScrollActivity(delegate ScrollActivityDelegate, simulation physics.Simulation, vsync scheduler.TickerProvider, shouldIgnorePointer bool, post func(func())) *BallisticScrollActivity {
	a := &BallisticScrollActivity{}
	InitBallisticScrollActivity(a, a, delegate, simulation, vsync, shouldIgnorePointer, post)
	return a
}

// InitBallisticScrollActivity sets up an activity that embeds a
// BallisticScrollActivity: self is the outer value, whose ApplyMoveTo the
// activity calls if it has one.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.new
func InitBallisticScrollActivity(a *BallisticScrollActivity, self ScrollActivity, delegate ScrollActivityDelegate, simulation physics.Simulation, vsync scheduler.TickerProvider, shouldIgnorePointer bool, post func(func())) {
	a.shouldIgnorePointer = shouldIgnorePointer
	InitScrollActivityBase(&a.ScrollActivityBase, self, delegate)
	label := ""
	if foundation.DebugMode {
		label = "BallisticScrollActivity"
	}
	a.controller = animation.NewAnimationControllerUnbounded(vsync, animation.AnimationControllerOpts{DebugLabel: label})
	a.listener = a.controller.AddListener(a.tick)
	// won't trigger if we dispose controller before it completes.
	a.controller.AnimateWith(simulation).Then(func() { runLater(post, a.end) })
}

// moveApplier is what an activity that embeds a BallisticScrollActivity or a
// DrivenScrollActivity implements to override ApplyMoveTo.
type moveApplier interface{ ApplyMoveTo(value float64) bool }

// runLater runs fn with post, as Dart runs the callback of a future after the
// code that completed it; without post it runs at once.
func runLater(post func(func()), fn func()) {
	if post == nil {
		fn()
		return
	}
	post(fn)
}

// ResetActivity goes ballistic again with the current velocity.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.resetActivity
func (a *BallisticScrollActivity) ResetActivity() { a.delegate.GoBallistic(a.Velocity()) }

// ApplyNewDimensions goes ballistic again with the current velocity.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.applyNewDimensions
func (a *BallisticScrollActivity) ApplyNewDimensions() { a.delegate.GoBallistic(a.Velocity()) }

func (a *BallisticScrollActivity) tick() {
	if !a.self.(moveApplier).ApplyMoveTo(a.controller.Value()) {
		a.delegate.GoIdle()
	}
}

// ApplyMoveTo moves the position to the value and returns whether to keep going:
// false when the position hit an edge and has overscroll.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.applyMoveTo
func (a *BallisticScrollActivity) ApplyMoveTo(value float64) bool {
	return math.Abs(a.delegate.SetPixels(value)) < foundation.PrecisionErrorTolerance
}

func (a *BallisticScrollActivity) end() {
	// Check if the activity was disposed before going ballistic because
	// _end might be called after dispose.
	if !a.disposed {
		a.delegate.GoBallistic(0)
	}
}

// DispatchOverscrollNotification dispatches an OverscrollNotification with the
// velocity of the activity.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.dispatchOverscrollNotification
func (a *BallisticScrollActivity) DispatchOverscrollNotification(metrics ScrollMetrics, bc BuildContext, overscroll float64) {
	DispatchNotification(bc, NewOverscrollNotification(metrics, bc, overscroll, OverscrollNotificationOpts{Velocity: a.Velocity()}))
}

// ShouldIgnorePointer is the value the activity was made with.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.shouldIgnorePointer
func (a *BallisticScrollActivity) ShouldIgnorePointer() bool { return a.shouldIgnorePointer }

// IsScrolling is true.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.isScrolling
func (a *BallisticScrollActivity) IsScrolling() bool { return true }

// Velocity is the velocity of the simulation.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.velocity
func (a *BallisticScrollActivity) Velocity() float64 { return a.controller.Velocity() }

// Dispose disposes the controller.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.dispose
func (a *BallisticScrollActivity) Dispose() {
	a.controller.Dispose()
	a.ScrollActivityBase.Dispose()
}

// String is the identity of the activity and its controller.
//
// Flutter: widgets/scroll_activity.dart BallisticScrollActivity.toString
func (a *BallisticScrollActivity) String() string {
	return foundation.DescribeIdentity(a) + "(" + a.controller.String() + ")"
}

// DrivenScrollActivity is the activity of a position that an animation drives to
// a value, or a simulation moves.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity
type DrivenScrollActivity struct {
	ScrollActivityBase
	controller *animation.AnimationController
	done       *scheduler.TickerFuture
	complete   func()
	post       func(func())
}

// DrivenScrollActivityOpts are the parameters of NewDrivenScrollActivity.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.new
type DrivenScrollActivityOpts struct {
	// From is the value to start from.
	From float64
	// To is the value to go to.
	To float64
	// Duration is the length of the animation; it must be positive.
	Duration time.Duration
	// Curve is the curve of the animation.
	Curve animation.Curve
	// Vsync gives the controller its ticker.
	Vsync scheduler.TickerProvider
	// Post runs a function after the code running now (a microtask); nil runs it
	// at once.
	Post func(func())
}

// NewDrivenScrollActivity creates the activity that animates the position from
// From to To. In debug builds it panics with the tag ScrollDurationInvalid when
// the duration is not positive.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.new
func NewDrivenScrollActivity(delegate ScrollActivityDelegate, opts ...DrivenScrollActivityOpts) *DrivenScrollActivity {
	a := &DrivenScrollActivity{}
	InitDrivenScrollActivity(a, a, delegate, firstOpts(opts))
	return a
}

// InitDrivenScrollActivity sets up an activity that embeds a
// DrivenScrollActivity: self is the outer value, whose ApplyMoveTo the activity
// calls if it has one.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.new
func InitDrivenScrollActivity(a *DrivenScrollActivity, self ScrollActivity, delegate ScrollActivityDelegate, o DrivenScrollActivityOpts) {
	if foundation.DebugMode && o.Duration <= 0 {
		panic(errors.New("ScrollDurationInvalid", "the duration of a driven scroll activity must be positive", "duration", o.Duration))
	}
	a.post = o.Post
	InitScrollActivityBase(&a.ScrollActivityBase, self, delegate)
	a.done, a.complete = scheduler.NewTickerFuturePending()
	a.controller = animation.NewAnimationControllerUnbounded(o.Vsync, animation.AnimationControllerOpts{
		Value: foundation.Some(o.From), DebugLabel: "DrivenScrollActivity"})
	a.controller.AddListener(a.tick)
	// won't trigger if we dispose controller before it completes.
	curve := o.Curve
	if curve == nil {
		curve = animation.Curves.Linear
	}
	a.controller.AnimateTo(o.To, animation.AnimateOpts{Duration: foundation.Some(o.Duration), Curve: curve}).Then(func() { runLater(o.Post, a.end) })
}

// NewDrivenScrollActivitySimulation creates the activity that moves the position
// with the simulation.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.simulation
func NewDrivenScrollActivitySimulation(delegate ScrollActivityDelegate, simulation physics.Simulation, vsync scheduler.TickerProvider, post func(func())) *DrivenScrollActivity {
	a := &DrivenScrollActivity{post: post}
	InitScrollActivityBase(&a.ScrollActivityBase, a, delegate)
	a.done, a.complete = scheduler.NewTickerFuturePending()
	a.controller = animation.NewAnimationControllerUnbounded(vsync, animation.AnimationControllerOpts{DebugLabel: "DrivenScrollActivity"})
	a.controller.AddListener(a.tick)
	a.controller.AnimateWith(simulation).Then(func() { runLater(post, a.end) })
	return a
}

// Done is the future that completes when the activity is disposed: when the
// animation ended or another activity replaced this one.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.done
func (a *DrivenScrollActivity) Done() *scheduler.TickerFuture { return a.done }

func (a *DrivenScrollActivity) tick() {
	if !a.self.(moveApplier).ApplyMoveTo(a.controller.Value()) {
		a.delegate.GoIdle()
	}
}

// ApplyMoveTo moves the position to the value and returns whether to keep going.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.applyMoveTo
func (a *DrivenScrollActivity) ApplyMoveTo(value float64) bool {
	return math.Abs(a.delegate.SetPixels(value)) < foundation.PrecisionErrorTolerance
}

func (a *DrivenScrollActivity) end() {
	// Check if the activity was disposed before going ballistic because
	// _end might be called after dispose.
	if !a.disposed {
		a.delegate.GoBallistic(a.Velocity())
	}
}

// DispatchOverscrollNotification dispatches an OverscrollNotification with the
// velocity of the activity.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.dispatchOverscrollNotification
func (a *DrivenScrollActivity) DispatchOverscrollNotification(metrics ScrollMetrics, bc BuildContext, overscroll float64) {
	DispatchNotification(bc, NewOverscrollNotification(metrics, bc, overscroll, OverscrollNotificationOpts{Velocity: a.Velocity()}))
}

// ShouldIgnorePointer is true.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.shouldIgnorePointer
func (a *DrivenScrollActivity) ShouldIgnorePointer() bool { return true }

// IsScrolling is true.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.isScrolling
func (a *DrivenScrollActivity) IsScrolling() bool { return true }

// Velocity is the velocity of the animation.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.velocity
func (a *DrivenScrollActivity) Velocity() float64 { return a.controller.Velocity() }

// Dispose completes Done and disposes the controller.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.dispose
func (a *DrivenScrollActivity) Dispose() {
	a.complete()
	a.controller.Dispose()
	a.ScrollActivityBase.Dispose()
}

// String is the identity of the activity and its controller.
//
// Flutter: widgets/scroll_activity.dart DrivenScrollActivity.toString
func (a *DrivenScrollActivity) String() string {
	return foundation.DescribeIdentity(a) + "(" + a.controller.String() + ")"
}
