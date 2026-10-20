// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollPositionAlignmentPolicy is how ScrollPosition.EnsureVisible aligns the
// object it reveals.
//
// Flutter: widgets/scroll_position.dart ScrollPositionAlignmentPolicy
type ScrollPositionAlignmentPolicy int

const (
	// ScrollPositionAlignmentPolicyExplicit is the zero value: the object is
	// aligned by the alignment given to EnsureVisible.
	//
	// Flutter: widgets/scroll_position.dart ScrollPositionAlignmentPolicy.explicit
	ScrollPositionAlignmentPolicyExplicit ScrollPositionAlignmentPolicy = iota
	// ScrollPositionAlignmentPolicyKeepVisibleAtEnd scrolls only as far as
	// needed to put the object at the end of the viewport, and not at all if it
	// is visible; it never scrolls backwards.
	//
	// Flutter: widgets/scroll_position.dart ScrollPositionAlignmentPolicy.keepVisibleAtEnd
	ScrollPositionAlignmentPolicyKeepVisibleAtEnd
	// ScrollPositionAlignmentPolicyKeepVisibleAtStart scrolls only as far as
	// needed to put the object at the start of the viewport; it never scrolls
	// forwards.
	//
	// Flutter: widgets/scroll_position.dart ScrollPositionAlignmentPolicy.keepVisibleAtStart
	ScrollPositionAlignmentPolicyKeepVisibleAtStart
)

// ScrollPosition determines which portion of the content of a scrollable is
// visible. It is the ViewportOffset of the scrollable's viewport and its
// ScrollMetrics, and it owns the current ScrollActivity.
//
// Implement it with ScrollPositionBase; ScrollPositionWithSingleContext is the
// position that the scroll controllers make.
//
// Flutter: widgets/scroll_position.dart ScrollPosition
type ScrollPosition interface {
	rendering.ViewportOffset
	ScrollMetrics

	// Base is the embedded ScrollPositionBase.
	Base() *ScrollPositionBase
	// Physics is the physics that decide how the position moves.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.physics
	Physics() ScrollPhysics
	// Context is the scrollable that owns the position.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.context
	Context() ScrollContext
	// KeepScrollOffset is whether the position saves its offset in the
	// PageStorage.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.keepScrollOffset
	KeepScrollOffset() bool
	// DebugLabel names the position in debug output.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.debugLabel
	DebugLabel() string
	// HaveDimensions is whether the content dimensions were applied.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.haveDimensions
	HaveDimensions() bool
	// ShouldIgnorePointer is whether the scrollable ignores pointer events.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.shouldIgnorePointer
	ShouldIgnorePointer() bool
	// Activity is the current activity.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.activity
	Activity() ScrollActivity
	// BeginActivity ends the current activity and starts the new one.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.beginActivity
	BeginActivity(newActivity ScrollActivity)
	// Absorb takes over the state of the old position of the same scrollable.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.absorb
	Absorb(other ScrollPosition)
	// SetPixels changes the position to the new value, within the boundary
	// conditions, and returns the overscroll.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.setPixels
	SetPixels(newPixels float64) float64
	// CorrectPixels changes the position without notifying anyone.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.correctPixels
	CorrectPixels(value float64)
	// ForcePixels changes the position and notifies, ignoring the boundary
	// conditions.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.forcePixels
	ForcePixels(value float64)
	// SaveScrollOffset writes the position to the PageStorage.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.saveScrollOffset
	SaveScrollOffset()
	// RestoreScrollOffset reads the position from the PageStorage.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.restoreScrollOffset
	RestoreScrollOffset()
	// RestoreOffset restores a saved offset.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.restoreOffset
	RestoreOffset(offset float64, initialRestore bool)
	// SaveOffset gives the offset to the scroll context to save.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.saveOffset
	SaveOffset()
	// ApplyBoundaryConditions is the part of the change to value that the
	// position does not apply.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.applyBoundaryConditions
	ApplyBoundaryConditions(value float64) float64
	// CorrectForNewDimensions adjusts the position for new dimensions; false
	// asks the viewport for another layout.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.correctForNewDimensions
	CorrectForNewDimensions(oldPosition, newPosition ScrollMetrics) bool
	// ApplyNewDimensions is called when the dimensions are accepted.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.applyNewDimensions
	ApplyNewDimensions()
	// EnsureVisible scrolls so that the render object is visible.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.ensureVisible
	EnsureVisible(object rendering.RenderObject, opts ...EnsureVisibleOpts) *scheduler.TickerFuture
	// IsScrollingNotifier is a notifier of whether the position is scrolling.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.isScrollingNotifier
	IsScrollingNotifier() *foundation.ValueNotifier[bool]
	// PointerScroll scrolls by a pointer scroll event (a wheel notch, a
	// trackpad) of delta pixels.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.pointerScroll
	PointerScroll(delta float64)
	// MoveTo animates or jumps to the value, clamped by default to the extents.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.moveTo
	MoveTo(to float64, opts ...rendering.ViewportOffsetMoveToOpts) *scheduler.TickerFuture
	// JumpToWithoutSettling jumps to the value without going ballistic.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.jumpToWithoutSettling
	JumpToWithoutSettling(value float64)
	// Hold stops the position, for a finger that touches down; the callback is
	// called when the hold ends.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.hold
	Hold(holdCancelCallback func()) ScrollHoldController
	// Drag starts a drag of the position; the callback is called when it ends.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.drag
	Drag(details gestures.DragStartDetails, dragCancelCallback func()) gestures.Drag
	// DidStartScroll dispatches the scroll start notification.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.didStartScroll
	DidStartScroll()
	// DidUpdateScrollPositionBy dispatches the scroll update notification.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.didUpdateScrollPositionBy
	DidUpdateScrollPositionBy(delta float64)
	// DidEndScroll dispatches the scroll end notification and saves the offset.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.didEndScroll
	DidEndScroll()
	// DidOverscrollBy dispatches the overscroll notification.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.didOverscrollBy
	DidOverscrollBy(value float64)
	// DidUpdateScrollDirection dispatches the user scroll notification.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.didUpdateScrollDirection
	DidUpdateScrollDirection(direction rendering.ScrollDirection)
	// DidUpdateScrollMetrics dispatches the scroll metrics notification.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.didUpdateScrollMetrics
	DidUpdateScrollMetrics()
	// RecommendDeferredLoading is whether the content should defer loading
	// images, because the position moves fast.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.recommendDeferredLoading
	RecommendDeferredLoading(bc BuildContext) bool
	// NotifyListeners notifies the listeners and updates the semantics actions.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.notifyListeners
	NotifyListeners()
	// Dispose disposes the activity and the notifiers.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.dispose
	Dispose()
	// DebugFillDescription appends the parts of the description of the position
	// to description and returns it.
	//
	// Flutter: widgets/scroll_position.dart ScrollPosition.debugFillDescription
	DebugFillDescription(description []string) []string
	// String describes the position.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.toString
	String() string
}

// EnsureVisibleOpts are the optional parameters of ScrollPosition.EnsureVisible.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.ensureVisible
type EnsureVisibleOpts struct {
	// Alignment is where in the viewport the object ends up: 0 for the leading
	// edge, 1 for the trailing edge, 0.5 for the middle.
	Alignment float64
	// Duration is the length of the animation; zero jumps.
	Duration time.Duration
	// Curve is the curve of the animation; the default is Curves.Ease.
	Curve animation.Curve
	// AlignmentPolicy is how the object is aligned.
	AlignmentPolicy ScrollPositionAlignmentPolicy
	// TargetRenderObject is the descendant of the object that is revealed, if it
	// is not the object itself.
	TargetRenderObject rendering.RenderObject
}

// ScrollPositionOpts are the optional parameters of ScrollPositionBase.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.new
type ScrollPositionOpts struct {
	// KeepScrollOffset is whether the position saves its offset in the
	// PageStorage; the default is true.
	KeepScrollOffset foundation.Opt[bool]
	// OldPosition is the position of the same scrollable that this one replaces;
	// this one absorbs it.
	OldPosition ScrollPosition
	// DebugLabel names the position in debug output.
	DebugLabel string
}

// ScrollPositionBase implements ScrollPosition except for AnimateTo, JumpTo,
// PointerScroll, JumpToWithoutSettling, Hold and Drag, which the position type
// provides. Embed it and call InitScrollPositionBase with the outer value.
//
// Flutter: widgets/scroll_position.dart ScrollPosition
type ScrollPositionBase struct {
	foundation.ChangeNotifier
	ScrollMetricsMixin
	self ScrollPosition

	physics          ScrollPhysics
	context          ScrollContext
	keepScrollOffset bool
	debugLabel       string

	minScrollExtent, maxScrollExtent foundation.Opt[float64]
	pixels                           foundation.Opt[float64]
	viewportDimension                foundation.Opt[float64]
	haveDimensions                   bool
	impliedVelocity                  float64
	activity                         ScrollActivity

	didChangeViewportDimensionOrReceiveCorrection bool
	pendingDimensions                             bool
	lastMetrics                                   ScrollMetrics
	haveScheduledUpdateNotification               bool
	lastAxis                                      foundation.Opt[painting.Axis]
	semanticActions                               map[ui.SemanticsAction]struct{}
	hasSemanticActions                            bool

	isScrollingNotifier *foundation.ValueNotifier[bool]
}

// InitScrollPositionBase sets the outer value, the physics and the context, and
// absorbs the old position and restores the saved offset if the options say so.
// Call it last in the constructor of the position, after its own fields are
// set.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.new
func InitScrollPositionBase(b *ScrollPositionBase, self ScrollPosition, physics ScrollPhysics, context ScrollContext, opts ...ScrollPositionOpts) {
	o := firstOpts(opts)
	b.self, b.physics, b.context = self, physics, context
	b.keepScrollOffset = o.KeepScrollOffset.OrElse(true)
	b.debugLabel = o.DebugLabel
	b.didChangeViewportDimensionOrReceiveCorrection = true
	b.isScrollingNotifier = foundation.NewValueNotifier(false)
	InitScrollMetricsMixin(&b.ScrollMetricsMixin, self)
	if !isNil(o.OldPosition) {
		self.Absorb(o.OldPosition)
	}
	if b.keepScrollOffset {
		self.RestoreScrollOffset()
	}
}

func isNil(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

// Base is the position itself.
func (b *ScrollPositionBase) Base() *ScrollPositionBase { return b }

// Physics is the physics that decide how the position moves.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.physics
func (b *ScrollPositionBase) Physics() ScrollPhysics { return b.physics }

// Context is the scrollable that owns the position.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.context
func (b *ScrollPositionBase) Context() ScrollContext { return b.context }

// KeepScrollOffset is whether the position saves its offset in the PageStorage.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.keepScrollOffset
func (b *ScrollPositionBase) KeepScrollOffset() bool { return b.keepScrollOffset }

// DebugLabel names the position in debug output.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.debugLabel
func (b *ScrollPositionBase) DebugLabel() string { return b.debugLabel }

// MinScrollExtent is the minimum in-range value for Pixels.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.minScrollExtent
func (b *ScrollPositionBase) MinScrollExtent() float64 {
	return scrollMetricsValue(b.minScrollExtent, "minScrollExtent")
}

// MaxScrollExtent is the maximum in-range value for Pixels.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.maxScrollExtent
func (b *ScrollPositionBase) MaxScrollExtent() float64 {
	return scrollMetricsValue(b.maxScrollExtent, "maxScrollExtent")
}

// HasContentDimensions is whether the scroll extents are known.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.hasContentDimensions
func (b *ScrollPositionBase) HasContentDimensions() bool {
	return b.minScrollExtent.IsSome() && b.maxScrollExtent.IsSome()
}

// Pixels is the current scroll position.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.pixels
func (b *ScrollPositionBase) Pixels() float64 { return scrollMetricsValue(b.pixels, "pixels") }

// HasPixels is whether the scroll position is known.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.hasPixels
func (b *ScrollPositionBase) HasPixels() bool { return b.pixels.IsSome() }

// ViewportDimension is the extent of the viewport.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.viewportDimension
func (b *ScrollPositionBase) ViewportDimension() float64 {
	return scrollMetricsValue(b.viewportDimension, "viewportDimension")
}

// HasViewportDimension is whether the extent of the viewport is known.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.hasViewportDimension
func (b *ScrollPositionBase) HasViewportDimension() bool { return b.viewportDimension.IsSome() }

// HaveDimensions is whether the content dimensions were applied.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.haveDimensions
func (b *ScrollPositionBase) HaveDimensions() bool { return b.haveDimensions }

// ShouldIgnorePointer is whether the scrollable ignores pointer events: not when
// it is out of range, and otherwise as the activity says.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.shouldIgnorePointer
func (b *ScrollPositionBase) ShouldIgnorePointer() bool {
	if b.self.OutOfRange() {
		return false
	}
	if b.activity == nil {
		return true
	}
	return b.activity.ShouldIgnorePointer()
}

// DevicePixelRatio is the device pixel ratio of the scrollable.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.devicePixelRatio
func (b *ScrollPositionBase) DevicePixelRatio() float64 { return b.context.DevicePixelRatio() }

// AxisDirection is the direction of the scrollable.
//
// Flutter: widgets/scroll_position_with_single_context.dart ScrollPositionWithSingleContext.axisDirection
func (b *ScrollPositionBase) AxisDirection() painting.AxisDirection { return b.context.AxisDirection() }

// IsScrollingNotifier is a notifier of whether the position is scrolling.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.isScrollingNotifier
func (b *ScrollPositionBase) IsScrollingNotifier() *foundation.ValueNotifier[bool] {
	return b.isScrollingNotifier
}

// AllowImplicitScrolling is whether the physics allow implicit scrolling.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.allowImplicitScrolling
func (b *ScrollPositionBase) AllowImplicitScrolling() bool { return b.physics.AllowImplicitScrolling() }

// Activity is the current activity; nil when it was absorbed by another
// position.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.activity
func (b *ScrollPositionBase) Activity() ScrollActivity { return b.activity }

func (b *ScrollPositionBase) widgetsBinding() *WidgetsBinding {
	return MaybeWidgetsBindingOf(b.context.StorageContext())
}

// post runs fn after the code that runs now (the microtask of Dart); at once if
// there is no binding.
func (b *ScrollPositionBase) post(fn func()) {
	if binding := b.widgetsBinding(); binding != nil {
		binding.Post(fn)
		return
	}
	fn()
}

// Absorb takes over the state of the old position of the same scrollable. In
// debug builds it panics with the tag ScrollPositionAbsorbInvalid if the old
// position belongs to another scrollable, or this one already has pixels or an
// activity, or the old one has no activity.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.absorb
func (b *ScrollPositionBase) Absorb(other ScrollPosition) {
	ob := other.Base()
	if foundation.DebugMode {
		switch {
		case ob.context != b.context:
			panic(errors.New("ScrollPositionAbsorbInvalid", "a scroll position can only absorb the position of the same scroll context"))
		case b.pixels.IsSome():
			panic(errors.New("ScrollPositionAbsorbInvalid", "a scroll position can only absorb another one before it has pixels"))
		case b.activity != nil:
			panic(errors.New("ScrollPositionAbsorbInvalid", "a scroll position can only absorb another one before it has an activity"))
		case ob.activity == nil:
			panic(errors.New("ScrollPositionAbsorbInvalid", "the scroll position to absorb has no activity"))
		}
	}
	if ob.HasContentDimensions() {
		b.minScrollExtent = ob.minScrollExtent
		b.maxScrollExtent = ob.maxScrollExtent
	}
	if ob.pixels.IsSome() {
		b.pixels = ob.pixels
	}
	if ob.viewportDimension.IsSome() {
		b.viewportDimension = ob.viewportDimension
	}
	b.activity = ob.activity
	ob.activity = nil
	if reflect.TypeOf(other) != reflect.TypeOf(b.self) {
		b.activity.ResetActivity()
	}
	b.context.SetIgnorePointer(b.activity.ShouldIgnorePointer())
	b.isScrollingNotifier.SetValue(b.activity.IsScrolling())
}

// SetPixels updates the position to the new value, within the boundary
// conditions of the physics, notifies the listeners and returns the overscroll:
// the part of the change that was not applied. In debug builds it panics with
// the tag ScrollPositionChangedDuringLayout if it is called in the build, layout
// or paint phase, and with ScrollOverscrollInvalid if the boundary conditions
// return more overscroll than the change.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.setPixels
func (b *ScrollPositionBase) SetPixels(newPixels float64) float64 {
	if foundation.DebugMode {
		if !b.pixels.IsSome() {
			panic(errors.New("ScrollMetricsValueMissing", "setPixels was called before the position has pixels", "value", "pixels"))
		}
		if binding := b.widgetsBinding(); binding != nil && binding.Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
			panic(errors.New("ScrollPositionChangedDuringLayout",
				"A scrollable's position should not change during the build, layout, and paint phases, otherwise the rendering will be confused"))
		}
	}
	if newPixels != b.Pixels() {
		overscroll := b.self.ApplyBoundaryConditions(newPixels)
		if foundation.DebugMode {
			delta := newPixels - b.Pixels()
			if math.Abs(overscroll) > math.Abs(delta) {
				panic(errors.New("ScrollOverscrollInvalid",
					"applyBoundaryConditions returned invalid overscroll value; setPixels() was called to change the scroll offset and applyBoundaryConditions reported an overscroll that is greater than the delta",
					"from", b.Pixels(), "to", newPixels, "delta", delta, "overscroll", overscroll))
			}
		}
		oldPixels := b.Pixels()
		b.pixels = foundation.Some(newPixels - overscroll)
		if b.Pixels() != oldPixels {
			if b.self.OutOfRange() {
				// If we go out of range, we need to make sure we ignore pointer
				// events; the activity that can ignore them is gone.
				b.context.SetIgnorePointer(false)
			}
			b.self.NotifyListeners()
			b.self.DidUpdateScrollPositionBy(b.Pixels() - oldPixels)
		}
		if math.Abs(overscroll) > foundation.PrecisionErrorTolerance {
			b.self.DidOverscrollBy(overscroll)
			return overscroll
		}
	}
	return 0
}

// CorrectPixels changes the position without notifying anyone. The viewport
// uses it when it lays out for the first time.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.correctPixels
func (b *ScrollPositionBase) CorrectPixels(value float64) { b.pixels = foundation.Some(value) }

// CorrectBy changes the position by the correction without notifying anyone.
// The viewport uses it when a sliver asks for a scroll offset correction.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.correctBy
func (b *ScrollPositionBase) CorrectBy(correction float64) {
	if foundation.DebugMode && !b.pixels.IsSome() {
		panic(errors.New("ScrollMetricsValueMissing", "an initial pixels value must exist by calling correctPixels on the scroll position", "value", "pixels"))
	}
	b.pixels = foundation.Some(b.pixels.Value() + correction)
	b.didChangeViewportDimensionOrReceiveCorrection = true
}

// ForcePixels changes the position and notifies the listeners, ignoring the
// boundary conditions, and remembers the change as the implied velocity until the
// end of the frame.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.forcePixels
func (b *ScrollPositionBase) ForcePixels(value float64) {
	b.impliedVelocity = value - b.Pixels()
	b.pixels = foundation.Some(value)
	b.self.NotifyListeners()
	if binding := b.widgetsBinding(); binding != nil {
		binding.RunPostFrame(func(time.Duration) { b.impliedVelocity = 0 })
	} else {
		b.impliedVelocity = 0
	}
}

// SaveScrollOffset writes the position to the PageStorage of the storage
// context, if there is one.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.saveScrollOffset
func (b *ScrollPositionBase) SaveScrollOffset() {
	if bucket := MaybePageStorageOf(b.context.StorageContext()); bucket != nil {
		bucket.WriteState(b.context.StorageContext(), b.Pixels())
	}
}

// RestoreScrollOffset reads the position from the PageStorage of the storage
// context, if it has none yet.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.restoreScrollOffset
func (b *ScrollPositionBase) RestoreScrollOffset() {
	if b.HasPixels() {
		return
	}
	if bucket := MaybePageStorageOf(b.context.StorageContext()); bucket != nil {
		if v, ok := bucket.ReadState(b.context.StorageContext()).(float64); ok {
			b.self.CorrectPixels(v)
		}
	}
}

// RestoreOffset restores a saved offset: it is the initial position when
// initialRestore is true, and a jump otherwise.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.restoreOffset
func (b *ScrollPositionBase) RestoreOffset(offset float64, initialRestore bool) {
	if initialRestore {
		b.self.CorrectPixels(offset)
	} else {
		b.self.JumpTo(offset)
	}
}

// SaveOffset gives the offset to the scroll context to save.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.saveOffset
func (b *ScrollPositionBase) SaveOffset() {
	if foundation.DebugMode && !b.HasPixels() {
		panic(errors.New("ScrollMetricsValueMissing", "saveOffset was called before the position has pixels", "value", "pixels"))
	}
	b.context.SaveOffset(b.Pixels())
}

// ApplyBoundaryConditions asks the physics for the overscroll of a change to
// value. In debug builds it panics with the tag ScrollPhysicsOverscrollInvalid if
// the physics return more overscroll than the change.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.applyBoundaryConditions
func (b *ScrollPositionBase) ApplyBoundaryConditions(value float64) float64 {
	result := b.physics.ApplyBoundaryConditions(b.self, value)
	if foundation.DebugMode {
		delta := value - b.Pixels()
		if math.Abs(result) > math.Abs(delta) {
			panic(errors.New("ScrollPhysicsOverscrollInvalid",
				"applyBoundaryConditions returned invalid overscroll value; it returned an overscroll that has a greater magnitude than the delta; it is only supposed to reduce the possible range of movement, not increase it",
				"physics", foundation.RuntimeTypeName(b.physics), "from", b.Pixels(), "to", value, "overscroll", result,
				"minScrollExtent", b.MinScrollExtent(), "maxScrollExtent", b.MaxScrollExtent(), "viewportDimension", b.ViewportDimension()))
		}
	}
	return result
}

// ApplyViewportDimension records the extent of the viewport; a change asks
// ApplyContentDimensions to accept it again. It always returns true.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.applyViewportDimension
func (b *ScrollPositionBase) ApplyViewportDimension(viewportDimension float64) bool {
	if !b.viewportDimension.IsSome() || b.viewportDimension.Value() != viewportDimension {
		b.viewportDimension = foundation.Some(viewportDimension)
		b.didChangeViewportDimensionOrReceiveCorrection = true
		// If this is called, you can rely on applyContentDimensions being called
		// soon afterwards in the same layout phase. So we put all the logic that
		// relies on both values being computed into applyContentDimensions.
	}
	return true
}

func (b *ScrollPositionBase) isMetricsChanged() bool {
	current := b.self.CopyWith()
	return b.lastMetrics == nil ||
		!(current.ExtentBefore() == b.lastMetrics.ExtentBefore() &&
			current.ExtentInside() == b.lastMetrics.ExtentInside() &&
			current.ExtentAfter() == b.lastMetrics.ExtentAfter() &&
			current.AxisDirection() == b.lastMetrics.AxisDirection())
}

// ApplyContentDimensions records the scroll extents of the content. It returns
// false to ask the viewport for another layout, when the physics moved the
// position. In debug builds it panics with the tag ScrollPositionExtentsInvalid
// when the minimum is past the maximum, and with ScrollPositionCorrectionMissing
// when ApplyNewDimensions changed the offset instead of CorrectForNewDimensions.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.applyContentDimensions
func (b *ScrollPositionBase) ApplyContentDimensions(minScrollExtent, maxScrollExtent float64) bool {
	tolerance := physics.ToleranceDefault.Distance
	if !physics.NearEqualOpt(b.minScrollExtent, foundation.Some(minScrollExtent), tolerance) ||
		!physics.NearEqualOpt(b.maxScrollExtent, foundation.Some(maxScrollExtent), tolerance) ||
		b.didChangeViewportDimensionOrReceiveCorrection ||
		!b.lastAxis.IsSome() || b.lastAxis.Value() != b.self.Axis() {
		if foundation.DebugMode && minScrollExtent > maxScrollExtent {
			panic(errors.New("ScrollPositionExtentsInvalid", "the minimum scroll extent must not be past the maximum",
				"minScrollExtent", minScrollExtent, "maxScrollExtent", maxScrollExtent))
		}
		b.minScrollExtent = foundation.Some(minScrollExtent)
		b.maxScrollExtent = foundation.Some(maxScrollExtent)
		b.lastAxis = foundation.Some(b.self.Axis())
		var currentMetrics ScrollMetrics
		if b.haveDimensions {
			currentMetrics = b.self.CopyWith()
		}
		b.didChangeViewportDimensionOrReceiveCorrection = false
		b.pendingDimensions = true
		if b.haveDimensions && !b.self.CorrectForNewDimensions(b.lastMetrics, currentMetrics) {
			return false
		}
		b.haveDimensions = true
	}
	if b.pendingDimensions {
		b.self.ApplyNewDimensions()
		b.pendingDimensions = false
	}
	if foundation.DebugMode && b.didChangeViewportDimensionOrReceiveCorrection {
		panic(errors.New("ScrollPositionCorrectionMissing", "use CorrectForNewDimensions() (and return true) to change the scroll offset during ApplyContentDimensions()"))
	}
	if b.isMetricsChanged() {
		// It is too late to send useful notifications, because the potential
		// listeners have, by definition, already been built this frame. To make
		// sure the notification is sent at all, schedule it in a microtask.
		if !b.haveScheduledUpdateNotification {
			b.post(b.self.DidUpdateScrollMetrics)
			b.haveScheduledUpdateNotification = true
		}
		b.lastMetrics = b.self.CopyWith()
	}
	return true
}

// CorrectForNewDimensions asks the physics for the position to use with the new
// dimensions, and corrects to it; it returns false, to lay out again, if the
// position changed.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.correctForNewDimensions
func (b *ScrollPositionBase) CorrectForNewDimensions(oldPosition, newPosition ScrollMetrics) bool {
	newPixels := b.physics.AdjustPositionForNewDimensions(oldPosition, newPosition, b.activity.IsScrolling(), b.activity.Velocity())
	if newPixels != b.Pixels() {
		b.self.CorrectPixels(newPixels)
		return false
	}
	return true
}

// ApplyNewDimensions tells the activity, and updates the semantics actions.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.applyNewDimensions
func (b *ScrollPositionBase) ApplyNewDimensions() {
	b.activity.ApplyNewDimensions()
	b.updateSemanticActions() // will potentially request a semantics update.
}

func (b *ScrollPositionBase) updateSemanticActions() {
	var forward, backward ui.SemanticsAction
	switch b.context.AxisDirection() {
	case painting.AxisDirectionUp:
		forward, backward = ui.SemanticsActionScrollDown, ui.SemanticsActionScrollUp
	case painting.AxisDirectionDown:
		forward, backward = ui.SemanticsActionScrollUp, ui.SemanticsActionScrollDown
	case painting.AxisDirectionLeft:
		forward, backward = ui.SemanticsActionScrollRight, ui.SemanticsActionScrollLeft
	default:
		forward, backward = ui.SemanticsActionScrollLeft, ui.SemanticsActionScrollRight
	}
	if !b.HasPixels() || !b.HasContentDimensions() {
		return
	}
	actions := map[ui.SemanticsAction]struct{}{}
	if b.Pixels() > b.MinScrollExtent() {
		actions[backward] = struct{}{}
	}
	if b.Pixels() < b.MaxScrollExtent() {
		actions[forward] = struct{}{}
	}
	if b.hasSemanticActions && reflect.DeepEqual(actions, b.semanticActions) {
		return
	}
	b.semanticActions, b.hasSemanticActions = actions, true
	b.context.SetSemanticsActions(actions)
}

func flipAlignmentPolicy(p ScrollPositionAlignmentPolicy) ScrollPositionAlignmentPolicy {
	switch p {
	case ScrollPositionAlignmentPolicyKeepVisibleAtEnd:
		return ScrollPositionAlignmentPolicyKeepVisibleAtStart
	case ScrollPositionAlignmentPolicyKeepVisibleAtStart:
		return ScrollPositionAlignmentPolicyKeepVisibleAtEnd
	}
	return p
}

func (b *ScrollPositionBase) applyAxisDirectionToAlignmentPolicy(p ScrollPositionAlignmentPolicy) ScrollPositionAlignmentPolicy {
	switch b.AxisDirection() {
	case painting.AxisDirectionUp, painting.AxisDirectionLeft:
		return flipAlignmentPolicy(p)
	}
	return p
}

// EnsureVisible scrolls so that the object is visible in the viewport that
// contains it, jumping or animating, and returns the future of the animation. In
// debug builds it panics with the tag ScrollEnsureVisibleDetached when the object
// is not attached.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.ensureVisible
func (b *ScrollPositionBase) EnsureVisible(object rendering.RenderObject, opts ...EnsureVisibleOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	if foundation.DebugMode && !object.Attached() {
		panic(errors.New("ScrollEnsureVisibleDetached", "the object to make visible must be attached"))
	}
	viewport := rendering.RenderAbstractViewportMaybeOf(object)
	if isNil(viewport) {
		return scheduler.NewTickerFutureComplete()
	}
	var targetRect foundation.Opt[ui.Rect]
	if !isNil(o.TargetRenderObject) && o.TargetRenderObject != object {
		targetRect = foundation.Some(o.TargetRenderObject.GetTransformTo(object).TransformRect(
			object.PaintBounds().Intersect(o.TargetRenderObject.PaintBounds())))
	}
	axis := b.self.Axis()
	var target float64
	switch b.applyAxisDirectionToAlignmentPolicy(o.AlignmentPolicy) {
	case ScrollPositionAlignmentPolicyExplicit:
		target = viewport.GetOffsetToReveal(object, o.Alignment, rendering.GetOffsetToRevealOpts{Rect: targetRect, Axis: axis}).Offset
		target = ui.ClampDouble(target, b.MinScrollExtent(), b.MaxScrollExtent())
	case ScrollPositionAlignmentPolicyKeepVisibleAtEnd:
		target = viewport.GetOffsetToReveal(object, 1.0, // Aligns to end
			rendering.GetOffsetToRevealOpts{Rect: targetRect, Axis: axis}).Offset
		target = ui.ClampDouble(target, b.MinScrollExtent(), b.MaxScrollExtent())
		if target < b.Pixels() {
			target = b.Pixels()
		}
	case ScrollPositionAlignmentPolicyKeepVisibleAtStart:
		target = viewport.GetOffsetToReveal(object, 0.0, // Aligns to start
			rendering.GetOffsetToRevealOpts{Rect: targetRect, Axis: axis}).Offset
		target = ui.ClampDouble(target, b.MinScrollExtent(), b.MaxScrollExtent())
		if target > b.Pixels() {
			target = b.Pixels()
		}
	}
	if target == b.Pixels() {
		return scheduler.NewTickerFutureComplete()
	}
	if o.Duration == 0 {
		b.self.JumpTo(target)
		return scheduler.NewTickerFutureComplete()
	}
	curve := o.Curve
	if curve == nil {
		curve = animation.Curves.Ease
	}
	return b.self.AnimateTo(target, o.Duration, curve)
}

// MoveTo animates to the value, or jumps if there is no duration. The value is
// clamped to the scroll extents unless Clamp is false.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.moveTo
func (b *ScrollPositionBase) MoveTo(to float64, opts ...rendering.ViewportOffsetMoveToOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	if o.Clamp.OrElse(true) {
		to = ui.ClampDouble(to, b.MinScrollExtent(), b.MaxScrollExtent())
	}
	return rendering.ViewportOffsetMoveTo(b.self, to, o)
}

// BeginActivity ends the current activity and starts the new one, updating
// whether the scrollable ignores pointers and the scrolling notifier, and
// dispatching the start or end of the scroll.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.beginActivity
func (b *ScrollPositionBase) BeginActivity(newActivity ScrollActivity) {
	if isNil(newActivity) {
		return
	}
	var wasScrolling, oldIgnorePointer bool
	if b.activity != nil {
		oldIgnorePointer = b.activity.ShouldIgnorePointer()
		wasScrolling = b.activity.IsScrolling()
		if wasScrolling && !newActivity.IsScrolling() {
			// Notifies and then saves the scroll offset.
			b.self.DidEndScroll()
		}
		b.activity.Dispose()
	}
	b.activity = newActivity
	if oldIgnorePointer != b.activity.ShouldIgnorePointer() {
		b.context.SetIgnorePointer(b.activity.ShouldIgnorePointer())
	}
	b.isScrollingNotifier.SetValue(b.activity.IsScrolling())
	if !wasScrolling && b.activity.IsScrolling() {
		b.self.DidStartScroll()
	}
}

// DidStartScroll dispatches the ScrollStartNotification of the activity.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.didStartScroll
func (b *ScrollPositionBase) DidStartScroll() {
	b.activity.DispatchScrollStartNotification(b.self.CopyWith(), b.context.NotificationContext())
}

// DidUpdateScrollPositionBy dispatches the ScrollUpdateNotification of the
// activity.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.didUpdateScrollPositionBy
func (b *ScrollPositionBase) DidUpdateScrollPositionBy(delta float64) {
	b.activity.DispatchScrollUpdateNotification(b.self.CopyWith(), b.context.NotificationContext(), delta)
}

// DidEndScroll dispatches the ScrollEndNotification of the activity, and saves
// the offset.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.didEndScroll
func (b *ScrollPositionBase) DidEndScroll() {
	b.activity.DispatchScrollEndNotification(b.self.CopyWith(), b.context.NotificationContext())
	b.self.SaveOffset()
	if b.keepScrollOffset {
		b.self.SaveScrollOffset()
	}
}

// DidOverscrollBy dispatches the OverscrollNotification of the activity. In
// debug builds it panics with the tag ScrollOverscrollWhileIdle if the activity
// is not scrolling.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.didOverscrollBy
func (b *ScrollPositionBase) DidOverscrollBy(value float64) {
	if foundation.DebugMode && !b.activity.IsScrolling() {
		panic(errors.New("ScrollOverscrollWhileIdle", "a position overscrolled while its activity was not scrolling", "overscroll", value))
	}
	b.activity.DispatchOverscrollNotification(b.self.CopyWith(), b.context.NotificationContext(), value)
}

// DidUpdateScrollDirection dispatches a UserScrollNotification.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.didUpdateScrollDirection
func (b *ScrollPositionBase) DidUpdateScrollDirection(direction rendering.ScrollDirection) {
	DispatchNotification(b.context.NotificationContext(),
		NewUserScrollNotification(b.self.CopyWith(), b.context.NotificationContext(), direction))
}

// DidUpdateScrollMetrics dispatches a ScrollMetricsNotification. It is called in
// a microtask after the metrics changed during layout.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.didUpdateScrollMetrics
func (b *ScrollPositionBase) DidUpdateScrollMetrics() {
	if foundation.DebugMode {
		if binding := b.widgetsBinding(); binding != nil && binding.Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
			panic(errors.New("ScrollPositionChangedDuringLayout", "didUpdateScrollMetrics was called during the build, layout or paint phase"))
		}
		if !b.haveScheduledUpdateNotification {
			panic(errors.New("ScrollMetricsNotificationUnscheduled", "didUpdateScrollMetrics was called without a notification being scheduled"))
		}
	}
	b.haveScheduledUpdateNotification = false
	if nc := b.context.NotificationContext(); !isNil(nc) {
		DispatchNotification(nc, NewScrollMetricsNotification(b.self.CopyWith(), nc))
	}
}

// RecommendDeferredLoading asks the physics whether the content should defer
// loading images, given the velocity of the position.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.recommendDeferredLoading
func (b *ScrollPositionBase) RecommendDeferredLoading(bc BuildContext) bool {
	return b.physics.RecommendDeferredLoading(b.activity.Velocity()+b.impliedVelocity, b.self.CopyWith(), bc)
}

// Dispose disposes the activity (nil if another position absorbed it) and the
// notifiers.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.dispose
func (b *ScrollPositionBase) Dispose() {
	if b.activity != nil {
		b.activity.Dispose() // it will be nil if it got absorbed by another ScrollPosition
	}
	b.activity = nil
	b.isScrollingNotifier.Dispose()
	b.ChangeNotifier.Dispose()
}

// NotifyListeners updates the semantics actions, then notifies the listeners.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.notifyListeners
func (b *ScrollPositionBase) NotifyListeners() {
	b.updateSemanticActions() // will potentially request a semantics update.
	b.ChangeNotifier.NotifyListeners()
}

// DebugFillDescription appends the label, the offset, the range and the
// viewport extent.
//
// Flutter: widgets/scroll_position.dart ScrollPosition.debugFillDescription
func (b *ScrollPositionBase) DebugFillDescription(description []string) []string {
	if b.debugLabel != "" {
		description = append(description, b.debugLabel)
	}
	if b.HasPixels() {
		description = append(description, "offset: "+dartfmt.ToStringAsFixed(b.Pixels(), 1))
	}
	description = append(description, "range: "+optFixed1(b.minScrollExtent)+".."+optFixed1(b.maxScrollExtent))
	description = append(description, "viewport: "+optFixed1(b.viewportDimension))
	return description
}

func optFixed1(v foundation.Opt[float64]) string {
	if f, ok := v.Get(); ok {
		return dartfmt.ToStringAsFixed(f, 1)
	}
	return "null"
}

// String is the identity of the position and its description.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.toString
func (b *ScrollPositionBase) String() string {
	return foundation.DescribeIdentity(b.self) + "(" + strings.Join(b.self.DebugFillDescription(nil), ", ") + ")"
}

func scrollNotScrollingError() error {
	return errors.New("ScrollPositionNotScrolling", "setPixels was called while the activity of the position is not scrolling")
}

func scrollAbsorbDragError() error {
	return errors.New("ScrollPositionDragInvalid", "the scroll position has a drag already")
}

func scrollActivityDelegateError() error {
	return errors.New("ScrollActivityDelegateInvalid", "the delegate of the new activity must be the position that begins it")
}

func scrollNoPixelsError() error {
	return errors.New("ScrollMetricsValueMissing", "goBallistic was called before the position has pixels", "value", "pixels")
}
