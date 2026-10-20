// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// NestedScrollViewHeaderSliversBuilder builds the slivers of the header of a
// NestedScrollView; innerBoxIsScrolled is whether the body is scrolled.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewHeaderSliversBuilder
type NestedScrollViewHeaderSliversBuilder func(bc BuildContext, innerBoxIsScrolled bool) []Widget

// NestedScrollViewOpts are the optional parameters of NewNestedScrollView.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.new
type NestedScrollViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the controller of the outer scroll view (the header).
	Controller ScrollController
	// ScrollDirection is the axis along which the view scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the view scrolls against the reading direction.
	Reverse bool
	// Physics is how the view responds to the user.
	Physics ScrollPhysics
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// FloatHeaderSlivers is whether the header slivers float, that is, whether
	// they are scrolled back in when the user scrolls the body back.
	FloatHeaderSlivers bool
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// RestorationID is the identifier for restoring the scroll offset; it has no
	// effect until state restoration is ported.
	RestorationID string
	// ScrollBehavior describes how the view scrolls; the default is the one of the
	// ScrollConfiguration above, without scrollbars.
	ScrollBehavior ScrollBehavior
}

// NestedScrollView is a scrolling view inside of which can be nested other
// scrolling views, with their scroll positions being intrinsically linked.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView
type NestedScrollView struct {
	StatefulWidget
	controller          ScrollController
	scrollDirection     painting.Axis
	reverse             bool
	physics             ScrollPhysics
	headerSliverBuilder NestedScrollViewHeaderSliversBuilder
	body                Widget
	dragStartBehavior   gestures.DragStartBehavior
	floatHeaderSlivers  bool
	clipBehavior        ui.Clip
	hitTestBehavior     rendering.HitTestBehavior
	restorationID       string
	scrollBehavior      ScrollBehavior
}

// NewNestedScrollView creates a scroll view of the header slivers, with the body
// under them.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.new
func NewNestedScrollView(headerSliverBuilder NestedScrollViewHeaderSliversBuilder, body Widget, opts ...NestedScrollViewOpts) *NestedScrollView {
	o := firstOpts(opts)
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisVertical
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	return &NestedScrollView{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o.Controller, o.ScrollDirection, o.Reverse, o.Physics,
		headerSliverBuilder, body, o.DragStartBehavior, o.FloatHeaderSlivers, o.ClipBehavior, o.HitTestBehavior, o.RestorationID, o.ScrollBehavior}
}

// Controller is the controller of the outer scroll view, or nil.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.controller
func (w *NestedScrollView) Controller() ScrollController { return w.controller }

// Body is the widget below the header.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.body
func (w *NestedScrollView) Body() Widget { return w.body }

// FloatHeaderSlivers is whether the header slivers float.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.floatHeaderSlivers
func (w *NestedScrollView) FloatHeaderSlivers() bool { return w.floatHeaderSlivers }

// NestedScrollViewSliverOverlapAbsorberHandleFor is the handle of the closest
// NestedScrollView above the context; the context depends on it. In debug builds
// it panics with the tag NestedScrollViewMissing when there is none.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.sliverOverlapAbsorberHandleFor
func NestedScrollViewSliverOverlapAbsorberHandleFor(bc BuildContext) *SliverOverlapAbsorberHandle {
	target, ok := DependOn[*inheritedNestedScrollView](bc)
	if !ok {
		if foundation.DebugMode {
			panic(errors.New("NestedScrollViewMissing",
				"NestedScrollView.sliverOverlapAbsorberHandleFor must be called with a context that contains a NestedScrollView"))
		}
		return nil
	}
	return target.state.absorberHandle
}

func (w *NestedScrollView) buildSlivers(bc BuildContext, innerController ScrollController, bodyIsScrolled bool) []Widget {
	slivers := append([]Widget(nil), w.headerSliverBuilder(bc, bodyIsScrolled)...)
	return append(slivers, NewSliverFillRemaining(NewPrimaryScrollController(innerController, w.body, PrimaryScrollControllerOpts{
		AutomaticallyInheritForPlatforms: allTargetPlatforms})))
}

var allTargetPlatforms = []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS,
	foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows}

// CreateState creates the state.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollView.createState
func (w *NestedScrollView) CreateState() State {
	return &NestedScrollViewState{absorberHandle: NewSliverOverlapAbsorberHandle()}
}

// NestedScrollViewState is the state of a NestedScrollView.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewState
type NestedScrollViewState struct {
	StateOf[*NestedScrollView]
	absorberHandle      *SliverOverlapAbsorberHandle
	coordinator         *nestedScrollCoordinator
	lastHasScrolledBody foundation.Opt[bool]
}

// InnerController is the controller of the body.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewState.innerController
func (s *NestedScrollViewState) InnerController() ScrollController {
	return s.coordinator.innerController
}

// OuterController is the controller of the header.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewState.outerController
func (s *NestedScrollViewState) OuterController() ScrollController {
	return s.coordinator.outerController
}

// InitState creates the coordinator.
func (s *NestedScrollViewState) InitState() {
	s.coordinator = newNestedScrollCoordinator(s, s.Widget().controller, s.handleHasScrolledBodyChanged, s.Widget().floatHeaderSlivers)
}

// DidChangeDependencies gives the coordinator the parent controller.
func (s *NestedScrollViewState) DidChangeDependencies() {
	s.coordinator.setParent(s.Widget().controller)
}

// DidUpdateWidget gives the coordinator the new parent controller.
func (s *NestedScrollViewState) DidUpdateWidget(oldWidget *NestedScrollView) {
	if oldWidget.controller != s.Widget().controller {
		s.coordinator.setParent(s.Widget().controller)
	}
}

// Dispose disposes the coordinator.
func (s *NestedScrollViewState) Dispose() {
	s.coordinator.dispose()
	s.coordinator = nil
	s.absorberHandle.Dispose()
}

func (s *NestedScrollViewState) handleHasScrolledBodyChanged() {
	if !s.Mounted() {
		return
	}
	newHasScrolledBody := s.coordinator.hasScrolledBody()
	if s.lastHasScrolledBody != foundation.Some(newHasScrolledBody) {
		s.SetState(func() {
			// Rebuild so that the header sliver builder sees the new value.
		})
	}
}

// Build builds the outer scroll view.
func (s *NestedScrollViewState) Build(BuildContext) Widget {
	w := s.Widget()
	var scrollPhysics ScrollPhysics = NewClampingScrollPhysics(nil)
	switch {
	case !isNil(w.physics):
		scrollPhysics = w.physics.ApplyTo(NewClampingScrollPhysics(nil))
	case w.scrollBehavior != nil:
		scrollPhysics = w.scrollBehavior.GetScrollPhysics(s.Context()).ApplyTo(NewClampingScrollPhysics(nil))
	}
	return &inheritedNestedScrollView{NewInheritedWidgetBase(NewBuilder(func(bc BuildContext) Widget {
		s.lastHasScrolledBody = foundation.Some(s.coordinator.hasScrolledBody())
		scrollBehavior := w.scrollBehavior
		if scrollBehavior == nil {
			scrollBehavior = ScrollConfigurationOf(bc).CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
		}
		view := &nestedScrollViewCustomScrollView{handle: s.absorberHandle}
		view.CustomScrollView = *NewCustomScrollView(w.buildSlivers(bc, s.coordinator.innerController, s.lastHasScrolledBody.Value()), CustomScrollViewOpts{
			ScrollDirection: w.scrollDirection, Reverse: w.reverse, Physics: physics_(scrollPhysics), ScrollBehavior: scrollBehavior,
			Controller: s.coordinator.outerController, ClipBehavior: w.clipBehavior, HitTestBehavior: w.hitTestBehavior,
			DragStartBehavior: w.dragStartBehavior, RestorationID: w.restorationID})
		view.self = view
		return view
	})), s}
}

// physics_ is the physics, to name the parameter without hiding the package.
func physics_(p ScrollPhysics) ScrollPhysics { return p }

// nestedScrollViewCustomScrollView is the CustomScrollView of a
// NestedScrollView: it builds a NestedScrollViewViewport.
//
// Flutter: widgets/nested_scroll_view.dart _NestedScrollViewCustomScrollView
type nestedScrollViewCustomScrollView struct {
	CustomScrollView
	handle *SliverOverlapAbsorberHandle
}

func (v *nestedScrollViewCustomScrollView) BuildViewport(_ BuildContext, offset rendering.ViewportOffset, axisDirection painting.AxisDirection, slivers []Widget) Widget {
	if foundation.DebugMode && v.shrinkWrap {
		panic(errors.New("NestedScrollViewShrinkWrap", "a NestedScrollView can not shrink-wrap its content"))
	}
	return NewNestedScrollViewViewport(offset, slivers, v.handle, ViewportOpts{AxisDirection: axisDirection, ClipBehavior: v.clipBehavior})
}

// inheritedNestedScrollView gives the state to the descendants.
//
// Flutter: widgets/nested_scroll_view.dart _InheritedNestedScrollView
type inheritedNestedScrollView struct {
	InheritedWidget
	state *NestedScrollViewState
}

func (w *inheritedNestedScrollView) UpdateShouldNotify(old Widget) bool {
	return w.state != old.(*inheritedNestedScrollView).state
}

// nestedScrollMetrics are the metrics of the whole nested scroll, for the physics.
//
// Flutter: widgets/nested_scroll_view.dart _NestedScrollMetrics
type nestedScrollMetrics struct {
	FixedScrollMetrics
	minRange, maxRange, correctionOffset float64
}

func newNestedScrollMetrics(minScrollExtent, maxScrollExtent, pixels, viewportDimension float64, axisDirection painting.AxisDirection,
	devicePixelRatio, minRange, maxRange, correctionOffset float64) *nestedScrollMetrics {
	m := &nestedScrollMetrics{minRange: minRange, maxRange: maxRange, correctionOffset: correctionOffset}
	m.FixedScrollMetrics = *NewFixedScrollMetrics(foundation.Some(minScrollExtent), foundation.Some(maxScrollExtent), foundation.Some(pixels),
		foundation.Some(viewportDimension), axisDirection, devicePixelRatio)
	InitScrollMetricsMixin(&m.ScrollMetricsMixin, m)
	return m
}

// nestedScrollCoordinator links the scroll positions of the outer and inner scroll
// views.
//
// Flutter: widgets/nested_scroll_view.dart _NestedScrollCoordinator
type nestedScrollCoordinator struct {
	state                    *NestedScrollViewState
	parent                   ScrollController
	onHasScrolledBodyChanged func()
	floatHeaderSlivers       bool
	outerController          *nestedScrollController
	innerController          *nestedScrollController
	userScrollDirection      rendering.ScrollDirection
	currentDrag              *ScrollDragController
}

func newNestedScrollCoordinator(state *NestedScrollViewState, parent ScrollController, onHasScrolledBodyChanged func(), floatHeaderSlivers bool) *nestedScrollCoordinator {
	c := &nestedScrollCoordinator{state: state, parent: parent, onHasScrolledBodyChanged: onHasScrolledBodyChanged,
		floatHeaderSlivers: floatHeaderSlivers, userScrollDirection: rendering.ScrollDirectionIdle}
	initialScrollOffset := 0.0
	if !isNil(parent) {
		initialScrollOffset = parent.InitialScrollOffset()
	}
	c.outerController = newNestedScrollController(c, initialScrollOffset, "outer")
	c.innerController = newNestedScrollController(c, 0, "inner")
	return c
}

func (c *nestedScrollCoordinator) outerPosition() *nestedScrollPosition {
	if !c.outerController.HasClients() {
		return nil
	}
	positions := c.outerController.nestedPositions()
	if foundation.DebugMode && len(positions) != 1 {
		panic(errors.New("NestedScrollOuterPositions", "the outer controller of a NestedScrollView must have exactly one position", "positions", len(positions)))
	}
	return positions[0]
}

func (c *nestedScrollCoordinator) innerPositions() []*nestedScrollPosition {
	return c.innerController.nestedPositions()
}

func (c *nestedScrollCoordinator) canScrollBody() bool {
	outer := c.outerPosition()
	if outer == nil {
		return true
	}
	return outer.haveDimensions && outer.ExtentAfter() == 0
}

func (c *nestedScrollCoordinator) hasScrolledBody() bool {
	for _, position := range c.innerPositions() {
		if !position.HasContentDimensions() || !position.HasPixels() {
			// It isn't scrollable yet; maybe it's loading.
			continue
		} else if position.Pixels() > position.MinScrollExtent() {
			return true
		}
	}
	return false
}

func (c *nestedScrollCoordinator) updateShadow() { c.onHasScrolledBodyChanged() }

func (c *nestedScrollCoordinator) updateUserScrollDirection(value rendering.ScrollDirection) {
	if c.userScrollDirection == value {
		return
	}
	c.userScrollDirection = value
	c.outerPosition().DidUpdateScrollDirection(value)
	for _, position := range c.innerPositions() {
		position.DidUpdateScrollDirection(value)
	}
}

func (c *nestedScrollCoordinator) beginActivity(newOuterActivity ScrollActivity, innerActivityGetter func(position *nestedScrollPosition) ScrollActivity) {
	c.outerPosition().BeginActivity(newOuterActivity)
	scrolling := newOuterActivity.IsScrolling()
	for _, position := range c.innerPositions() {
		newInnerActivity := innerActivityGetter(position)
		position.BeginActivity(newInnerActivity)
		scrolling = scrolling && newInnerActivity.IsScrolling()
	}
	if c.currentDrag != nil {
		c.currentDrag.Dispose()
	}
	c.currentDrag = nil
	if !scrolling {
		c.updateUserScrollDirection(rendering.ScrollDirectionIdle)
	}
}

// AxisDirection is the axis direction of the outer position.
func (c *nestedScrollCoordinator) AxisDirection() painting.AxisDirection {
	return c.outerPosition().AxisDirection()
}

func createIdleScrollActivity(position *nestedScrollPosition) ScrollActivity {
	return NewIdleScrollActivity(position)
}

// GoIdle makes every position idle.
func (c *nestedScrollCoordinator) GoIdle() {
	c.beginActivity(createIdleScrollActivity(c.outerPosition()), createIdleScrollActivity)
}

// GoBallistic makes every position ballistic.
func (c *nestedScrollCoordinator) GoBallistic(velocity float64) {
	c.beginActivity(c.createOuterBallisticScrollActivity(velocity), func(position *nestedScrollPosition) ScrollActivity {
		return c.createInnerBallisticScrollActivity(position, velocity)
	})
}

func (c *nestedScrollCoordinator) createOuterBallisticScrollActivity(velocity float64) ScrollActivity {
	// This function creates a ballistic scroll for the outer scrollable.
	//
	// It assumes that the outer scrollable can't be overscrolled, and sets up a
	// ballistic scroll over the combined space of the innerPosition and the
	// outerPosition.
	var innerPosition *nestedScrollPosition
	if velocity != 0 {
		for _, position := range c.innerPositions() {
			if innerPosition != nil {
				if velocity > 0 {
					if innerPosition.Pixels() < position.Pixels() {
						continue
					}
				} else if innerPosition.Pixels() > position.Pixels() {
					continue
				}
			}
			innerPosition = position
		}
	}
	outer := c.outerPosition()
	if innerPosition == nil {
		// It's either just a simple case (no inner position or no velocity)...
		return outer.createBallisticScrollActivity(outer.physics.CreateBallisticSimulation(outer, velocity), nestedBallisticIndependent, nil)
	}
	metrics := c.getMetrics(innerPosition, velocity)
	return outer.createBallisticScrollActivity(outer.physics.CreateBallisticSimulation(metrics, velocity), nestedBallisticOuter, metrics)
}

func (c *nestedScrollCoordinator) createInnerBallisticScrollActivity(position *nestedScrollPosition, velocity float64) ScrollActivity {
	return position.createBallisticScrollActivity(position.physics.CreateBallisticSimulation(c.getMetrics(position, velocity), velocity), nestedBallisticInner, nil)
}

func (c *nestedScrollCoordinator) getMetrics(innerPosition *nestedScrollPosition, velocity float64) *nestedScrollMetrics {
	outer := c.outerPosition()
	var pixels, minRange, maxRange, correctionOffset float64
	extra := 0.0
	if innerPosition.Pixels() == innerPosition.MinScrollExtent() {
		pixels = ui.ClampDouble(outer.Pixels(), outer.MinScrollExtent(), outer.MaxScrollExtent())
		minRange = outer.MinScrollExtent()
		maxRange = outer.MaxScrollExtent()
		correctionOffset = 0
	} else {
		if innerPosition.Pixels() < innerPosition.MinScrollExtent() {
			pixels = innerPosition.Pixels() - innerPosition.MinScrollExtent() + outer.MinScrollExtent()
		} else {
			pixels = innerPosition.Pixels() - innerPosition.MinScrollExtent() + outer.MaxScrollExtent()
		}
		switch {
		case velocity > 0 && innerPosition.Pixels() > innerPosition.MinScrollExtent():
			// This handles going forward (fling up) and inner list is scrolled past
			// zero. We want to grab the extra pixels immediately to shrink the outer
			// list.
			extra = outer.MaxScrollExtent() - outer.Pixels()
			minRange = pixels
			maxRange = pixels + extra
			correctionOffset = outer.Pixels() - pixels
		case velocity < 0 && innerPosition.Pixels() < innerPosition.MinScrollExtent():
			// This handles going backward (fling down) and inner list is
			// underscrolled. We want to grab the extra pixels immediately to grow
			// the outer list.
			extra = outer.Pixels() - outer.MinScrollExtent()
			minRange = pixels - extra
			maxRange = pixels
			correctionOffset = outer.Pixels() - pixels
		default:
			// This handles the case where the inner list is overscrolled in the
			// direction of the fling.
			if velocity > 0 {
				extra = outer.MinScrollExtent() - outer.Pixels()
			} else if velocity < 0 {
				extra = outer.Pixels() - (outer.MaxScrollExtent() - outer.MinScrollExtent())
			}
			minRange = outer.MinScrollExtent()
			maxRange = outer.MaxScrollExtent() + extra
			correctionOffset = 0
		}
	}
	return newNestedScrollMetrics(outer.MinScrollExtent(), outer.MaxScrollExtent()+innerPosition.MaxScrollExtent()-innerPosition.MinScrollExtent()+extra,
		pixels, outer.ViewportDimension(), outer.AxisDirection(), outer.DevicePixelRatio(), minRange, maxRange, correctionOffset)
}

func (c *nestedScrollCoordinator) unnestOffset(value float64, source *nestedScrollPosition) float64 {
	outer := c.outerPosition()
	if source == outer {
		return ui.ClampDouble(value, outer.MinScrollExtent(), outer.MaxScrollExtent())
	}
	if value < source.MinScrollExtent() {
		return value - source.MinScrollExtent() + outer.MinScrollExtent()
	}
	return value - source.MinScrollExtent() + outer.MaxScrollExtent()
}

func (c *nestedScrollCoordinator) nestOffset(value float64, target *nestedScrollPosition) float64 {
	outer := c.outerPosition()
	if target == outer {
		return ui.ClampDouble(value, outer.MinScrollExtent(), outer.MaxScrollExtent())
	}
	if value < outer.MinScrollExtent() {
		return value - outer.MinScrollExtent() + target.MinScrollExtent()
	}
	if value > outer.MaxScrollExtent() {
		return value - outer.MaxScrollExtent() + target.MinScrollExtent()
	}
	return target.MinScrollExtent()
}

func (c *nestedScrollCoordinator) updateCanDrag() {
	outer := c.outerPosition()
	if !outer.haveDimensions {
		return
	}
	innerCanDrag := false
	for _, position := range c.innerPositions() {
		if !position.haveDimensions {
			return
		}
		innerCanDrag = innerCanDrag || position.physics.ShouldAcceptUserOffset(position)
	}
	outer.updateCanDrag(innerCanDrag)
}

func (c *nestedScrollCoordinator) animateTo(to float64, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	outer := c.outerPosition()
	outerActivity := outer.createDrivenScrollActivity(c.nestOffset(to, outer), duration, curve)
	resultFutures := []*scheduler.TickerFuture{outerActivity.Done()}
	c.beginActivity(outerActivity, func(position *nestedScrollPosition) ScrollActivity {
		innerActivity := position.createDrivenScrollActivity(c.nestOffset(to, position), duration, curve)
		resultFutures = append(resultFutures, innerActivity.Done())
		return innerActivity
	})
	return scheduler.WaitTickerFutures(resultFutures...)
}

func (c *nestedScrollCoordinator) jumpTo(to float64) {
	c.GoIdle()
	outer := c.outerPosition()
	outer.localJumpTo(c.nestOffset(to, outer))
	for _, position := range c.innerPositions() {
		position.localJumpTo(c.nestOffset(to, position))
	}
	c.GoBallistic(0)
}

func (c *nestedScrollCoordinator) pointerScroll(delta float64) {
	// If an update is made to pointer scrolling here, consider if the same
	// (or similar) change should be made in
	// ScrollPositionWithSingleContext.pointerScroll.
	if delta == 0 {
		c.GoBallistic(0)
		return
	}
	c.GoIdle()
	if delta < 0 {
		c.updateUserScrollDirection(rendering.ScrollDirectionForward)
	} else {
		c.updateUserScrollDirection(rendering.ScrollDirectionReverse)
	}
	outer := c.outerPosition()
	inner := c.innerPositions()
	outer.isScrollingNotifier.SetValue(true)
	outer.DidStartScroll()
	for _, position := range inner {
		position.isScrollingNotifier.SetValue(true)
		position.DidStartScroll()
	}
	switch {
	case len(inner) == 0:
		// Does not enter overscroll.
		outer.applyClampedPointerSignalUpdate(delta)
	case delta > 0:
		// Dragging "up" - delta is positive.
		outerDelta := delta
		for _, position := range inner {
			if position.Pixels() < 0 {
				// This inner position is in overscroll.
				potentialOuterDelta := position.applyClampedPointerSignalUpdate(delta)
				// In case there are multiple positions in varying states of
				// overscroll, the first to 'reach' the outer view above takes
				// precedence.
				outerDelta = math.Max(outerDelta, potentialOuterDelta)
			}
		}
		if outerDelta != 0 {
			innerDelta := outer.applyClampedPointerSignalUpdate(outerDelta)
			if innerDelta != 0 {
				for _, position := range inner {
					position.applyClampedPointerSignalUpdate(innerDelta)
				}
			}
		}
	default:
		// Dragging "down" - delta is negative.
		innerDelta := delta
		if c.floatHeaderSlivers {
			innerDelta = outer.applyClampedPointerSignalUpdate(delta)
		}
		if innerDelta != 0 {
			outerDelta := 0.0 // it will go negative if it changes
			for _, position := range inner {
				overscroll := position.applyClampedPointerSignalUpdate(innerDelta)
				outerDelta = math.Min(outerDelta, overscroll)
			}
			if outerDelta != 0 {
				outer.applyClampedPointerSignalUpdate(outerDelta)
			}
		}
	}
	outer.DidEndScroll()
	for _, position := range inner {
		position.DidEndScroll()
	}
	c.GoBallistic(0)
}

// SetPixels is never called: the activities of the positions use their own.
func (c *nestedScrollCoordinator) SetPixels(float64) float64 {
	panic(errors.New("NestedScrollCoordinatorSetPixels", "the coordinator of a NestedScrollView does not set pixels"))
}

func (c *nestedScrollCoordinator) hold(holdCancelCallback func()) ScrollHoldController {
	c.beginActivity(NewHoldScrollActivity(c.outerPosition(), holdCancelCallback), func(position *nestedScrollPosition) ScrollActivity {
		return NewHoldScrollActivity(position, nil)
	})
	return c
}

// Cancel goes ballistic with no velocity, ending a hold.
func (c *nestedScrollCoordinator) Cancel() { c.GoBallistic(0) }

func (c *nestedScrollCoordinator) drag(details gestures.DragStartDetails, dragCancelCallback func()) gestures.Drag {
	drag := NewScrollDragController(c, details, ScrollDragControllerOpts{OnDragCanceled: dragCancelCallback})
	c.beginActivity(NewDragScrollActivity(c.outerPosition(), drag), func(position *nestedScrollPosition) ScrollActivity {
		return NewDragScrollActivity(position, drag)
	})
	if foundation.DebugMode && c.currentDrag != nil {
		panic(errors.New("NestedScrollDragActive", "the coordinator has a drag already"))
	}
	c.currentDrag = drag
	return drag
}

// ApplyUserOffset applies the drag delta to the positions.
func (c *nestedScrollCoordinator) ApplyUserOffset(delta float64) {
	if delta > 0 {
		c.updateUserScrollDirection(rendering.ScrollDirectionForward)
	} else {
		c.updateUserScrollDirection(rendering.ScrollDirectionReverse)
	}
	outer := c.outerPosition()
	inner := c.innerPositions()
	switch {
	case len(inner) == 0:
		// Does not enter overscroll.
		outer.applyFullDragUpdate(delta)
	case delta < 0:
		// Dragging "up"
		outerDelta := delta
		for _, position := range inner {
			if position.Pixels() < 0 {
				// This inner position is in overscroll.
				potentialOuterDelta := position.applyClampedDragUpdate(delta)
				// In case there are multiple positions in varying states of
				// overscroll, the first to 'reach' the outer view above takes
				// precedence.
				outerDelta = math.Max(outerDelta, potentialOuterDelta)
			}
		}
		if math.Abs(outerDelta) > foundation.PrecisionErrorTolerance {
			innerDelta := outer.applyClampedDragUpdate(outerDelta)
			if innerDelta != 0 {
				for _, position := range inner {
					position.applyFullDragUpdate(innerDelta)
				}
			}
		}
	default:
		// Dragging "down" - delta is positive
		innerDelta := delta
		if c.floatHeaderSlivers {
			innerDelta = outer.applyClampedDragUpdate(delta)
		}
		if innerDelta != 0 {
			// Apply the innerDelta, if we have inner positions, prioritizing the
			// inner position over the outer.
			outerDelta := 0.0 // it will go positive if it changes
			overscrolls := make([]float64, 0, len(inner))
			for _, position := range inner {
				overscroll := position.applyClampedDragUpdate(innerDelta)
				outerDelta = math.Max(outerDelta, overscroll)
				overscrolls = append(overscrolls, overscroll)
			}
			if outerDelta != 0 {
				outerDelta -= outer.applyClampedDragUpdate(outerDelta)
			}
			// Now deal with any overscroll.
			for i, position := range inner {
				remainingDelta := overscrolls[i] - outerDelta
				if remainingDelta > 0 {
					position.applyFullDragUpdate(remainingDelta)
				}
			}
		}
	}
}

func (c *nestedScrollCoordinator) setParent(value ScrollController) {
	c.parent = value
	c.updateParent()
}

func (c *nestedScrollCoordinator) updateParent() {
	if outer := c.outerPosition(); outer != nil {
		parent := c.parent
		if isNil(parent) {
			parent = PrimaryScrollControllerMaybeOf(c.state.Context())
		}
		outer.setParent(parent)
	}
}

func (c *nestedScrollCoordinator) dispose() {
	if c.currentDrag != nil {
		c.currentDrag.Dispose()
	}
	c.currentDrag = nil
	c.outerController.Dispose()
	c.innerController.Dispose()
}

func (c *nestedScrollCoordinator) String() string {
	return foundation.RuntimeTypeName(c) + "(outer=" + c.outerController.String() + "; inner=" + c.innerController.String() + ")"
}

// nestedScrollController is the controller of the outer or inner scroll view.
//
// Flutter: widgets/nested_scroll_view.dart _NestedScrollController
type nestedScrollController struct {
	ScrollControllerBase
	coordinator   *nestedScrollCoordinator
	shadowHandles map[ScrollPosition]foundation.ListenerHandle
}

func newNestedScrollController(coordinator *nestedScrollCoordinator, initialScrollOffset float64, debugLabel string) *nestedScrollController {
	c := &nestedScrollController{coordinator: coordinator, shadowHandles: map[ScrollPosition]foundation.ListenerHandle{}}
	InitScrollControllerBase(&c.ScrollControllerBase, c, ScrollControllerOpts{InitialScrollOffset: initialScrollOffset, DebugLabel: debugLabel})
	return c
}

func (c *nestedScrollController) CreateScrollPosition(physics ScrollPhysics, context ScrollContext, oldPosition ScrollPosition) ScrollPosition {
	return newNestedScrollPosition(c.coordinator, physics, context, c.initialScrollOffset, oldPosition, c.debugLabel)
}

func (c *nestedScrollController) Attach(position ScrollPosition) {
	c.ScrollControllerBase.Attach(position)
	c.coordinator.updateParent()
	c.coordinator.updateCanDrag()
	c.shadowHandles[position] = position.AddListener(c.scheduleUpdateShadow)
	c.scheduleUpdateShadow()
}

func (c *nestedScrollController) Detach(position ScrollPosition) {
	position.(*nestedScrollPosition).setParent(nil)
	position.RemoveListener(c.shadowHandles[position])
	delete(c.shadowHandles, position)
	c.ScrollControllerBase.Detach(position)
	c.scheduleUpdateShadow()
}

func (c *nestedScrollController) scheduleUpdateShadow() {
	if binding := MaybeWidgetsBindingOf(c.coordinator.state.Context()); binding != nil {
		binding.Scheduler().AddPostFrameCallback(func(time.Duration) { c.coordinator.updateShadow() })
	}
}

func (c *nestedScrollController) nestedPositions() []*nestedScrollPosition {
	out := make([]*nestedScrollPosition, 0, len(c.positions))
	for _, p := range c.positions {
		out = append(out, p.(*nestedScrollPosition))
	}
	return out
}

type nestedBallisticMode int

const (
	nestedBallisticOuter nestedBallisticMode = iota
	nestedBallisticInner
	nestedBallisticIndependent
)

// nestedScrollPosition is the scroll position of the outer or an inner scroll
// view: the coordinator moves it.
//
// Flutter: widgets/nested_scroll_view.dart _NestedScrollPosition
type nestedScrollPosition struct {
	ScrollPositionBase
	coordinator *nestedScrollCoordinator
	parent      ScrollController
}

func newNestedScrollPosition(coordinator *nestedScrollCoordinator, physics ScrollPhysics, context ScrollContext, initialPixels float64, oldPosition ScrollPosition, debugLabel string) *nestedScrollPosition {
	p := &nestedScrollPosition{coordinator: coordinator}
	InitScrollPositionBase(&p.ScrollPositionBase, p, physics, context, ScrollPositionOpts{OldPosition: oldPosition, DebugLabel: debugLabel})
	if !p.HasPixels() {
		p.CorrectPixels(initialPixels)
	}
	if p.activity == nil {
		p.GoIdle()
	}
	p.SaveScrollOffset() // in case we didn't restore but could, so that we don't restore it later
	return p
}

func (p *nestedScrollPosition) setParent(value ScrollController) {
	if !isNil(p.parent) {
		p.parent.Detach(p)
	}
	p.parent = value
	if !isNil(p.parent) {
		p.parent.Attach(p)
	}
}

// AxisDirection is the axis direction of the scroll context.
func (p *nestedScrollPosition) AxisDirection() painting.AxisDirection {
	return p.context.AxisDirection()
}

// Absorb takes over the state of the old position.
func (p *nestedScrollPosition) Absorb(other ScrollPosition) {
	p.ScrollPositionBase.Absorb(other)
	p.activity.UpdateDelegate(p)
}

// RestoreScrollOffset restores the offset only when the body can be scrolled.
func (p *nestedScrollPosition) RestoreScrollOffset() {
	if p.coordinator.canScrollBody() {
		p.ScrollPositionBase.RestoreScrollOffset()
	}
}

// applyClampedDragUpdate applies a drag update that stops at the extents, and
// returns the part of the delta that it did not apply.
func (p *nestedScrollPosition) applyClampedDragUpdate(delta float64) float64 {
	// If we are going towards the maxScrollExtent (negative scroll offset),
	// then the furthest we can be in the minScrollExtent direction is negative
	// infinity. For example, if we are already overscrolled, then scrolling to
	// reduce the overscroll should not disallow the overscroll.
	min := math.Inf(-1)
	if delta >= 0 {
		min = math.Min(p.MinScrollExtent(), p.Pixels())
	}
	max := math.Inf(1)
	if delta <= 0 {
		if p.Pixels() < 0 {
			max = 0
		} else {
			max = math.Max(p.MaxScrollExtent(), p.Pixels())
		}
	}
	oldPixels := p.Pixels()
	newPixels := ui.ClampDouble(p.Pixels()-delta, min, max)
	clampedDelta := newPixels - p.Pixels()
	if clampedDelta == 0 {
		return delta
	}
	overscroll := p.physics.ApplyBoundaryConditions(p, newPixels)
	actualNewPixels := newPixels - overscroll
	offset := actualNewPixels - oldPixels
	if offset != 0 {
		p.ForcePixels(actualNewPixels)
		p.DidUpdateScrollPositionBy(offset)
	}
	result := delta + offset
	if math.Abs(result) < foundation.PrecisionErrorTolerance {
		return 0
	}
	return result
}

// applyFullDragUpdate applies a drag update, with the physics; it returns the
// overscroll.
func (p *nestedScrollPosition) applyFullDragUpdate(delta float64) float64 {
	oldPixels := p.Pixels()
	// Apply friction:
	newPixels := p.Pixels() - p.physics.ApplyPhysicsToUserOffset(p, delta)
	if math.Abs(oldPixels-newPixels) < foundation.PrecisionErrorTolerance {
		// Delta is so tiny we can drop it.
		return 0
	}
	// Check for overscroll:
	overscroll := p.physics.ApplyBoundaryConditions(p, newPixels)
	actualNewPixels := newPixels - overscroll
	if actualNewPixels != oldPixels {
		p.ForcePixels(actualNewPixels)
		p.DidUpdateScrollPositionBy(actualNewPixels - oldPixels)
	}
	if overscroll != 0 {
		p.DidOverscrollBy(overscroll)
		return overscroll
	}
	return 0
}

// applyClampedPointerSignalUpdate applies a pointer scroll that stops at the
// extents, and returns the part of the delta that it did not apply.
func (p *nestedScrollPosition) applyClampedPointerSignalUpdate(delta float64) float64 {
	min := math.Inf(-1)
	if delta <= 0 {
		min = math.Min(p.MinScrollExtent(), p.Pixels())
	}
	max := math.Inf(1)
	if delta >= 0 {
		max = math.Max(p.MaxScrollExtent(), p.Pixels())
	}
	newPixels := ui.ClampDouble(p.Pixels()+delta, min, max)
	clampedDelta := newPixels - p.Pixels()
	if clampedDelta == 0 {
		return delta
	}
	p.ForcePixels(newPixels)
	p.DidUpdateScrollPositionBy(clampedDelta)
	return delta - clampedDelta
}

// UserScrollDirection is the direction that the coordinator says.
func (p *nestedScrollPosition) UserScrollDirection() rendering.ScrollDirection {
	return p.coordinator.userScrollDirection
}

func (p *nestedScrollPosition) createDrivenScrollActivity(to float64, duration time.Duration, curve animation.Curve) *DrivenScrollActivity {
	return NewDrivenScrollActivity(p, DrivenScrollActivityOpts{From: p.Pixels(), To: to, Duration: duration, Curve: curve, Vsync: p.context.Vsync(), Post: p.post})
}

// ApplyUserOffset is never called: the coordinator applies the drags.
func (p *nestedScrollPosition) ApplyUserOffset(float64) {
	panic(errors.New("NestedScrollPositionApplyUserOffset", "a nested scroll position does not apply user offsets; its coordinator does"))
}

// GoIdle starts an idle activity.
func (p *nestedScrollPosition) GoIdle() {
	p.BeginActivity(NewIdleScrollActivity(p))
	p.coordinator.updateUserScrollDirection(rendering.ScrollDirectionIdle)
}

// GoBallistic starts a ballistic activity of the position on its own.
func (p *nestedScrollPosition) GoBallistic(velocity float64) {
	var simulation physics.Simulation
	if velocity != 0 || p.OutOfRange() {
		simulation = p.physics.CreateBallisticSimulation(p, velocity)
	}
	p.BeginActivity(p.createBallisticScrollActivity(simulation, nestedBallisticIndependent, nil))
}

func (p *nestedScrollPosition) createBallisticScrollActivity(simulation physics.Simulation, mode nestedBallisticMode, metrics *nestedScrollMetrics) ScrollActivity {
	if isNil(simulation) {
		return NewIdleScrollActivity(p)
	}
	switch mode {
	case nestedBallisticOuter:
		if metrics.minRange == metrics.maxRange {
			return NewIdleScrollActivity(p)
		}
		return newNestedOuterBallisticScrollActivity(p.coordinator, p, metrics, simulation, p.context.Vsync(), p.self.ShouldIgnorePointer(), p.post)
	case nestedBallisticInner:
		return newNestedInnerBallisticScrollActivity(p.coordinator, p, simulation, p.context.Vsync(), p.self.ShouldIgnorePointer(), p.post)
	}
	return NewBallisticScrollActivity(p, simulation, p.context.Vsync(), p.self.ShouldIgnorePointer(), p.post)
}

// AnimateTo animates the whole nested scroll.
func (p *nestedScrollPosition) AnimateTo(to float64, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	return p.coordinator.animateTo(p.coordinator.unnestOffset(to, p), duration, curve)
}

// JumpTo jumps the whole nested scroll.
func (p *nestedScrollPosition) JumpTo(value float64) {
	p.coordinator.jumpTo(p.coordinator.unnestOffset(value, p))
}

// PointerScroll scrolls the whole nested scroll.
func (p *nestedScrollPosition) PointerScroll(delta float64) { p.coordinator.pointerScroll(delta) }

// JumpToWithoutSettling is never called.
func (p *nestedScrollPosition) JumpToWithoutSettling(float64) {
	panic(errors.New("NestedScrollPositionJumpToWithoutSettling", "a nested scroll position does not jump without settling"))
}

func (p *nestedScrollPosition) localJumpTo(value float64) {
	if p.Pixels() != value {
		oldPixels := p.Pixels()
		p.ForcePixels(value)
		p.DidStartScroll()
		p.DidUpdateScrollPositionBy(p.Pixels() - oldPixels)
		p.DidEndScroll()
	}
}

// ApplyNewDimensions lets the coordinator update whether the user can drag.
func (p *nestedScrollPosition) ApplyNewDimensions() {
	p.ScrollPositionBase.ApplyNewDimensions()
	p.coordinator.updateCanDrag()
}

func (p *nestedScrollPosition) updateCanDrag(innerCanDrag bool) {
	if foundation.DebugMode && p.coordinator.outerPosition() != p {
		panic(errors.New("NestedScrollPositionNotOuter", "only the outer position of a NestedScrollView updates whether the user can drag"))
	}
	// This is only called for the outer position.
	// The outer position can drag if it can scroll or an inner position can.
	p.context.SetCanDrag(p.physics.ShouldAcceptUserOffset(p) || innerCanDrag)
}

// Hold goes through the coordinator.
func (p *nestedScrollPosition) Hold(holdCancelCallback func()) ScrollHoldController {
	return p.coordinator.hold(holdCancelCallback)
}

// Drag goes through the coordinator.
func (p *nestedScrollPosition) Drag(details gestures.DragStartDetails, dragCancelCallback func()) gestures.Drag {
	return p.coordinator.drag(details, dragCancelCallback)
}

// nestedInnerBallisticScrollActivity is the ballistic activity of an inner
// position.
//
// Flutter: widgets/nested_scroll_view.dart _NestedInnerBallisticScrollActivity
type nestedInnerBallisticScrollActivity struct {
	BallisticScrollActivity
	coordinator *nestedScrollCoordinator
}

func newNestedInnerBallisticScrollActivity(coordinator *nestedScrollCoordinator, position *nestedScrollPosition, simulation physics.Simulation,
	vsync scheduler.TickerProvider, shouldIgnorePointer bool, post func(func())) *nestedInnerBallisticScrollActivity {
	a := &nestedInnerBallisticScrollActivity{coordinator: coordinator}
	InitBallisticScrollActivity(&a.BallisticScrollActivity, a, position, simulation, vsync, shouldIgnorePointer, post)
	return a
}

func (a *nestedInnerBallisticScrollActivity) position() *nestedScrollPosition {
	return a.delegate.(*nestedScrollPosition)
}

func (a *nestedInnerBallisticScrollActivity) ResetActivity() {
	d := a.position()
	d.BeginActivity(a.coordinator.createInnerBallisticScrollActivity(d, a.Velocity()))
}

func (a *nestedInnerBallisticScrollActivity) ApplyNewDimensions() {
	d := a.position()
	d.BeginActivity(a.coordinator.createInnerBallisticScrollActivity(d, a.Velocity()))
}

func (a *nestedInnerBallisticScrollActivity) ApplyMoveTo(value float64) bool {
	return a.BallisticScrollActivity.ApplyMoveTo(a.coordinator.nestOffset(value, a.position()))
}

// nestedOuterBallisticScrollActivity is the ballistic activity of the outer
// position.
//
// Flutter: widgets/nested_scroll_view.dart _NestedOuterBallisticScrollActivity
type nestedOuterBallisticScrollActivity struct {
	BallisticScrollActivity
	coordinator *nestedScrollCoordinator
	metrics     *nestedScrollMetrics
}

func newNestedOuterBallisticScrollActivity(coordinator *nestedScrollCoordinator, position *nestedScrollPosition, metrics *nestedScrollMetrics,
	simulation physics.Simulation, vsync scheduler.TickerProvider, shouldIgnorePointer bool, post func(func())) *nestedOuterBallisticScrollActivity {
	if foundation.DebugMode && !(metrics.minRange != metrics.maxRange && metrics.maxRange > metrics.minRange) {
		panic(errors.New("NestedScrollRangeInvalid", "the range of an outer ballistic scroll must not be empty", "minRange", metrics.minRange, "maxRange", metrics.maxRange))
	}
	a := &nestedOuterBallisticScrollActivity{coordinator: coordinator, metrics: metrics}
	InitBallisticScrollActivity(&a.BallisticScrollActivity, a, position, simulation, vsync, shouldIgnorePointer, post)
	return a
}

func (a *nestedOuterBallisticScrollActivity) ResetActivity() {
	a.delegate.(*nestedScrollPosition).BeginActivity(a.coordinator.createOuterBallisticScrollActivity(a.Velocity()))
}

func (a *nestedOuterBallisticScrollActivity) ApplyNewDimensions() {
	a.delegate.(*nestedScrollPosition).BeginActivity(a.coordinator.createOuterBallisticScrollActivity(a.Velocity()))
}

func (a *nestedOuterBallisticScrollActivity) ApplyMoveTo(value float64) bool {
	done := false
	velocity := a.Velocity()
	switch {
	case velocity > 0:
		if value < a.metrics.minRange {
			// Ignore any values that don't fit within the range.
			return true
		}
		if value > a.metrics.maxRange {
			value = a.metrics.maxRange
			done = true
		}
	case velocity < 0:
		if value > a.metrics.maxRange {
			return true
		}
		if value < a.metrics.minRange {
			value = a.metrics.minRange
			done = true
		}
	default:
		value = ui.ClampDouble(value, a.metrics.minRange, a.metrics.maxRange)
		done = true
	}
	result := a.BallisticScrollActivity.ApplyMoveTo(value + a.metrics.correctionOffset)
	if foundation.DebugMode && !result {
		panic(errors.New("NestedScrollOuterOverflow", "an in-range value overflowed in an outer ballistic scroll"))
	}
	return !done
}

func (a *nestedOuterBallisticScrollActivity) String() string {
	return foundation.RuntimeTypeName(a) + "(" + formatDouble(a.metrics.minRange) + " .. " + formatDouble(a.metrics.maxRange) +
		"; correcting by " + formatDouble(a.metrics.correctionOffset) + ")"
}

// SliverOverlapAbsorberHandle communicates the extent that a SliverOverlapAbsorber
// absorbs to the SliverOverlapInjector.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorberHandle
type SliverOverlapAbsorberHandle struct {
	foundation.ChangeNotifier
	writers      int
	layoutExtent foundation.Opt[float64]
	scrollExtent foundation.Opt[float64]
}

// NewSliverOverlapAbsorberHandle creates a handle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorberHandle.new
func NewSliverOverlapAbsorberHandle() *SliverOverlapAbsorberHandle {
	return &SliverOverlapAbsorberHandle{}
}

// LayoutExtent is the sum of the layout extents that the absorber absorbed, or
// none before it is laid out.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorberHandle.layoutExtent
func (h *SliverOverlapAbsorberHandle) LayoutExtent() foundation.Opt[float64] { return h.layoutExtent }

// ScrollExtent is the sum of the scroll extents that the absorber absorbed, or
// none before it is laid out.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorberHandle.scrollExtent
func (h *SliverOverlapAbsorberHandle) ScrollExtent() foundation.Opt[float64] { return h.scrollExtent }

func (h *SliverOverlapAbsorberHandle) setExtents(layoutValue, scrollValue foundation.Opt[float64]) {
	if foundation.DebugMode && h.writers != 1 {
		panic(errors.New("SliverOverlapAbsorberHandleWriters", "multiple RenderSliverOverlapAbsorbers have been provided the same SliverOverlapAbsorberHandle"))
	}
	h.layoutExtent, h.scrollExtent = layoutValue, scrollValue
}

func (h *SliverOverlapAbsorberHandle) markNeedsLayout() { h.NotifyListeners() }

// String describes the handle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorberHandle.toString
func (h *SliverOverlapAbsorberHandle) String() string {
	extra := ""
	switch h.writers {
	case 0:
		extra = ", orphan"
	case 1:
	default:
		extra = ", " + strconv.Itoa(h.writers) + " WRITERS ASSIGNED"
	}
	extent := "null"
	if v, ok := h.layoutExtent.Get(); ok {
		extent = formatDouble(v)
	}
	return "SliverOverlapAbsorberHandle(" + extent + extra + ")"
}

// SliverOverlapAbsorberOpts are the optional parameters of NewSliverOverlapAbsorber.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorber.new
type SliverOverlapAbsorberOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverOverlapAbsorber is a sliver that wraps another, forcing its layout extent
// to be treated as overlap.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorber
type SliverOverlapAbsorber struct {
	SingleChildRenderObjectWidget
	handle *SliverOverlapAbsorberHandle
}

// NewSliverOverlapAbsorber creates a sliver that absorbs the overlap of its
// sliver into the handle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorber.new
func NewSliverOverlapAbsorber(handle *SliverOverlapAbsorberHandle, sliver Widget, opts ...SliverOverlapAbsorberOpts) *SliverOverlapAbsorber {
	return &SliverOverlapAbsorber{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: firstOpts(opts).Key}), handle}
}

// Handle is where the absorbed extent goes.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorber.handle
func (w *SliverOverlapAbsorber) Handle() *SliverOverlapAbsorberHandle { return w.handle }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorber.createRenderObject
func (w *SliverOverlapAbsorber) CreateRenderObject(BuildContext) rendering.RenderObject {
	return NewRenderSliverOverlapAbsorber(w.handle, nil)
}

// UpdateRenderObject sets the handle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapAbsorber.updateRenderObject
func (w *SliverOverlapAbsorber) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*RenderSliverOverlapAbsorber).SetHandle(w.handle)
}

// RenderSliverOverlapAbsorber is the render object of a SliverOverlapAbsorber.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapAbsorber
type RenderSliverOverlapAbsorber struct {
	rendering.RenderSliver
	rendering.RenderObjectWithChild[rendering.Sliver]
	handle *SliverOverlapAbsorberHandle
}

// NewRenderSliverOverlapAbsorber creates the sliver; the child may be nil.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapAbsorber.new
func NewRenderSliverOverlapAbsorber(handle *SliverOverlapAbsorberHandle, sliver rendering.Sliver) *RenderSliverOverlapAbsorber {
	r := &RenderSliverOverlapAbsorber{handle: handle}
	rendering.InitRenderSliver(&r.RenderSliver, r)
	rendering.InitRenderObjectWithChild(&r.RenderObjectWithChild, r)
	if !isNil(sliver) {
		r.SetChild(sliver)
	}
	return r
}

// Handle is where the absorbed extent goes.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapAbsorber.handle
func (r *RenderSliverOverlapAbsorber) Handle() *SliverOverlapAbsorberHandle { return r.handle }

// SetHandle changes the handle.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapAbsorber.handle
func (r *RenderSliverOverlapAbsorber) SetHandle(value *SliverOverlapAbsorberHandle) {
	if r.handle == value {
		return
	}
	if r.Attached() {
		r.handle.writers--
		value.writers++
		value.setExtents(r.handle.layoutExtent, r.handle.scrollExtent)
	}
	r.handle = value
}

// Attach counts this absorber as a writer of the handle.
func (r *RenderSliverOverlapAbsorber) Attach(owner *rendering.PipelineOwner) {
	r.RenderSliver.Attach(owner)
	r.handle.writers++
}

// Detach stops counting this absorber as a writer of the handle.
func (r *RenderSliverOverlapAbsorber) Detach() {
	r.handle.writers--
	r.RenderSliver.Detach()
}

// PerformLayout lays the child out, takes its obstruction out of its scroll and
// layout extents, and tells the handle.
func (r *RenderSliverOverlapAbsorber) PerformLayout() {
	if foundation.DebugMode && r.handle.writers != 1 {
		panic(errors.New("SliverOverlapAbsorberHandleWriters",
			"a SliverOverlapAbsorberHandle cannot be passed to multiple RenderSliverOverlapAbsorber objects at the same time"))
	}
	child := r.Child()
	if isNil(child) {
		r.SetGeometry(rendering.SliverGeometryZero)
		return
	}
	child.Layout(r.SliverConstraints(), rendering.LayoutOpts{ParentUsesSize: true})
	childLayoutGeometry := child.Geometry()
	r.SetGeometry(childLayoutGeometry.CopyWith(rendering.SliverGeometryCopy{
		ScrollExtent: foundation.Some(childLayoutGeometry.ScrollExtent - childLayoutGeometry.MaxScrollObstructionExtent),
		LayoutExtent: foundation.Some(math.Max(0, childLayoutGeometry.PaintExtent-childLayoutGeometry.MaxScrollObstructionExtent)),
	}))
	// The extent we use here is not the layoutExtent, as that extent is the
	// parent's.
	r.handle.setExtents(foundation.Some(childLayoutGeometry.MaxScrollObstructionExtent), foundation.Some(childLayoutGeometry.MaxScrollObstructionExtent))
}

// ChildMainAxisPosition is zero.
func (r *RenderSliverOverlapAbsorber) ChildMainAxisPosition(rendering.RenderObject) float64 { return 0 }

// ApplyPaintTransform does nothing: the child is at the origin.
func (r *RenderSliverOverlapAbsorber) ApplyPaintTransform(rendering.RenderObject, *ui.Matrix4) {}

// HitTestChildren hit tests the child.
func (r *RenderSliverOverlapAbsorber) HitTestChildren(result *rendering.SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	if child := r.Child(); !isNil(child) {
		return child.HitTest(result, mainAxisPosition, crossAxisPosition)
	}
	return false
}

// Paint paints the child.
func (r *RenderSliverOverlapAbsorber) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if child := r.Child(); !isNil(child) {
		context.PaintChild(child, offset)
	}
}

// SliverOverlapInjectorOpts are the optional parameters of NewSliverOverlapInjector.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapInjector.new
type SliverOverlapInjectorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverOverlapInjector is a sliver that has a sliver geometry based on the
// values stored in a SliverOverlapAbsorberHandle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapInjector
type SliverOverlapInjector struct {
	SingleChildRenderObjectWidget
	handle *SliverOverlapAbsorberHandle
}

// NewSliverOverlapInjector creates a sliver that injects the extent of the handle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapInjector.new
func NewSliverOverlapInjector(handle *SliverOverlapAbsorberHandle, sliver Widget, opts ...SliverOverlapInjectorOpts) *SliverOverlapInjector {
	return &SliverOverlapInjector{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: firstOpts(opts).Key}), handle}
}

// Handle is where the extent comes from.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapInjector.handle
func (w *SliverOverlapInjector) Handle() *SliverOverlapAbsorberHandle { return w.handle }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapInjector.createRenderObject
func (w *SliverOverlapInjector) CreateRenderObject(BuildContext) rendering.RenderObject {
	return NewRenderSliverOverlapInjector(w.handle)
}

// UpdateRenderObject sets the handle.
//
// Flutter: widgets/nested_scroll_view.dart SliverOverlapInjector.updateRenderObject
func (w *SliverOverlapInjector) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*RenderSliverOverlapInjector).SetHandle(w.handle)
}

// RenderSliverOverlapInjector is the render object of a SliverOverlapInjector.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapInjector
type RenderSliverOverlapInjector struct {
	rendering.RenderSliver
	currentLayoutExtent foundation.Opt[float64]
	currentMaxExtent    foundation.Opt[float64]
	handle              *SliverOverlapAbsorberHandle
	handleListener      foundation.ListenerHandle
}

// NewRenderSliverOverlapInjector creates the sliver.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapInjector.new
func NewRenderSliverOverlapInjector(handle *SliverOverlapAbsorberHandle) *RenderSliverOverlapInjector {
	r := &RenderSliverOverlapInjector{handle: handle}
	rendering.InitRenderSliver(&r.RenderSliver, r)
	return r
}

// Handle is where the extent comes from.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapInjector.handle
func (r *RenderSliverOverlapInjector) Handle() *SliverOverlapAbsorberHandle { return r.handle }

// SetHandle changes the handle.
//
// Flutter: widgets/nested_scroll_view.dart RenderSliverOverlapInjector.handle
func (r *RenderSliverOverlapInjector) SetHandle(value *SliverOverlapAbsorberHandle) {
	if r.handle == value {
		return
	}
	if r.Attached() {
		r.handle.RemoveListener(r.handleListener)
	}
	r.handle = value
	if r.Attached() {
		r.handleListener = r.handle.AddListener(r.MarkNeedsLayout)
		if r.handle.layoutExtent != r.currentLayoutExtent || r.handle.scrollExtent != r.currentMaxExtent {
			r.MarkNeedsLayout()
		}
	}
}

// Attach listens to the handle.
func (r *RenderSliverOverlapInjector) Attach(owner *rendering.PipelineOwner) {
	r.RenderSliver.Attach(owner)
	r.handleListener = r.handle.AddListener(r.MarkNeedsLayout)
	if r.handle.layoutExtent != r.currentLayoutExtent || r.handle.scrollExtent != r.currentMaxExtent {
		// Layout is used to resolve the scroll extents; those could have
		// changed, so mark it dirty.
		r.MarkNeedsLayout()
	}
}

// Detach stops listening to the handle.
func (r *RenderSliverOverlapInjector) Detach() {
	r.handle.RemoveListener(r.handleListener)
	r.RenderSliver.Detach()
}

// PerformLayout takes the extent from the handle.
func (r *RenderSliverOverlapInjector) PerformLayout() {
	r.currentLayoutExtent = r.handle.layoutExtent
	r.currentMaxExtent = r.handle.layoutExtent
	if foundation.DebugMode && (r.currentLayoutExtent.IsNone() || r.currentMaxExtent.IsNone()) {
		panic(errors.New("SliverOverlapInjectorNoExtent",
			"SliverOverlapInjector has found no absorbed extent to inject; the SliverOverlapAbsorber must be an earlier descendant of a common ancestor Viewport, so that it will always be laid out before the SliverOverlapInjector during a particular frame; the SliverOverlapAbsorber is typically contained in the list of slivers provided by NestedScrollView.headerSliverBuilder"))
	}
	constraints := r.SliverConstraints()
	clampedPaintExtent := math.Min(r.currentLayoutExtent.Value(), constraints.RemainingPaintExtent)
	clampedLayoutExtent := math.Min(r.currentLayoutExtent.Value()-constraints.ScrollOffset, constraints.RemainingPaintExtent)
	r.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{
		ScrollExtent: r.currentLayoutExtent.Value(), PaintExtent: math.Max(0, clampedPaintExtent),
		LayoutExtent: foundation.Some(math.Max(0, clampedLayoutExtent)), MaxPaintExtent: r.currentMaxExtent.Value()}))
}

// NestedScrollViewViewport is a Viewport that talks to a
// SliverOverlapAbsorberHandle: it tells it to lay out the absorber again when it
// lays out.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewViewport
type NestedScrollViewViewport struct {
	Viewport
	handle *SliverOverlapAbsorberHandle
}

// NewNestedScrollViewViewport creates a viewport of the slivers at the offset.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewViewport.new
func NewNestedScrollViewViewport(offset rendering.ViewportOffset, slivers []Widget, handle *SliverOverlapAbsorberHandle, opts ...ViewportOpts) *NestedScrollViewViewport {
	w := &NestedScrollViewViewport{handle: handle}
	w.Viewport = *NewViewport(offset, slivers, opts...)
	return w
}

// Handle talks to the sliver overlap absorber.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewViewport.handle
func (w *NestedScrollViewViewport) Handle() *SliverOverlapAbsorberHandle { return w.handle }

// CreateElement creates the element.
func (w *NestedScrollViewViewport) CreateElement() Element {
	e := &viewportElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	DebugChildrenHaveDuplicateKeys(w, w.Children())
	e.forgotten = map[Element]struct{}{}
	return e
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewViewport.createRenderObject
func (w *NestedScrollViewViewport) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	r := &RenderNestedScrollViewViewport{handle: w.handle}
	rendering.InitRenderViewport(&r.RenderViewport, r, w.crossAxis(bc), w.offset, rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{AxisDirection: w.axisDirection, ScrollCacheExtent: w.scrollCacheExtent,
			PaintOrder: w.paintOrder, ClipBehavior: w.clipBehavior}, Anchor: w.anchor})
	return r
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/nested_scroll_view.dart NestedScrollViewViewport.updateRenderObject
func (w *NestedScrollViewViewport) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	w.Viewport.UpdateRenderObject(bc, renderObject)
	renderObject.(*RenderNestedScrollViewViewport).SetHandle(w.handle)
}

// RenderNestedScrollViewViewport is the render object of a NestedScrollViewViewport.
//
// Flutter: widgets/nested_scroll_view.dart RenderNestedScrollViewViewport
type RenderNestedScrollViewViewport struct {
	rendering.RenderViewport
	handle *SliverOverlapAbsorberHandle
}

// Handle talks to the sliver overlap absorber.
//
// Flutter: widgets/nested_scroll_view.dart RenderNestedScrollViewViewport.handle
func (r *RenderNestedScrollViewViewport) Handle() *SliverOverlapAbsorberHandle { return r.handle }

// SetHandle changes the handle.
//
// Flutter: widgets/nested_scroll_view.dart RenderNestedScrollViewViewport.handle
func (r *RenderNestedScrollViewViewport) SetHandle(value *SliverOverlapAbsorberHandle) {
	if r.handle == value {
		return
	}
	r.handle = value
	r.handle.markNeedsLayout()
}

// MarkNeedsLayout tells the handle, then marks the viewport.
func (r *RenderNestedScrollViewViewport) MarkNeedsLayout() {
	r.handle.markNeedsLayout()
	r.RenderViewport.MarkNeedsLayout()
}

// formatDouble is how Dart prints a double.
func formatDouble(v float64) string {
	if v == math.Trunc(v) && !math.IsInf(v, 0) {
		return fmt.Sprintf("%.1f", v)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
