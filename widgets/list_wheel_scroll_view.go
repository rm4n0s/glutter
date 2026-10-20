// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

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
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// ChangeReportingBehavior is when a ListWheelScrollView reports the selected
// item.
//
// Flutter: widgets/list_wheel_scroll_view.dart ChangeReportingBehavior
type ChangeReportingBehavior int

const (
	// ChangeReportingBehaviorUnspecified is the zero value: the default of the
	// widget.
	ChangeReportingBehaviorUnspecified ChangeReportingBehavior = iota
	// ChangeReportingBehaviorOnScrollEnd reports the selected item when the
	// scroll ends.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart ChangeReportingBehavior.onScrollEnd
	ChangeReportingBehaviorOnScrollEnd
	// ChangeReportingBehaviorOnScrollUpdate reports the selected item on every
	// scroll update.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart ChangeReportingBehavior.onScrollUpdate
	ChangeReportingBehaviorOnScrollUpdate
)

// ListWheelChildDelegate supplies the children of a ListWheelScrollView.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate
type ListWheelChildDelegate interface {
	// Build returns the child at the index, or nil if there is none.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate.build
	Build(bc BuildContext, index int) Widget
	// EstimatedChildCount is the number of children, or none if it is not known.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate.estimatedChildCount
	EstimatedChildCount() foundation.Opt[int]
	// TrueIndexOf is the index of the item in the list that the index of a looping
	// delegate stands for.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate.trueIndexOf
	TrueIndexOf(index int) int
	// ShouldRebuild is whether the children must be built again, as this delegate
	// replaces the old one.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate.shouldRebuild
	ShouldRebuild(oldDelegate ListWheelChildDelegate) bool
}

// ListWheelChildListDelegate supplies the children from an explicit list.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildListDelegate
type ListWheelChildListDelegate struct {
	children []Widget
}

// NewListWheelChildListDelegate creates a delegate for the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildListDelegate.new
func NewListWheelChildListDelegate(children []Widget) *ListWheelChildListDelegate {
	return &ListWheelChildListDelegate{children}
}

// Children are the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildListDelegate.children
func (d *ListWheelChildListDelegate) Children() []Widget { return d.children }

// EstimatedChildCount is the number of children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildListDelegate.estimatedChildCount
func (d *ListWheelChildListDelegate) EstimatedChildCount() foundation.Opt[int] {
	return foundation.Some(len(d.children))
}

// TrueIndexOf is the index itself.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate.trueIndexOf
func (d *ListWheelChildListDelegate) TrueIndexOf(index int) int { return index }

// Build returns the child at the index, with its semantic index.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildListDelegate.build
func (d *ListWheelChildListDelegate) Build(_ BuildContext, index int) Widget {
	if index < 0 || index >= len(d.children) {
		return nil
	}
	return NewIndexedSemantics(index, d.children[index])
}

// ShouldRebuild is whether the list of children is not the same one.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildListDelegate.shouldRebuild
func (d *ListWheelChildListDelegate) ShouldRebuild(oldDelegate ListWheelChildDelegate) bool {
	old := oldDelegate.(*ListWheelChildListDelegate)
	return !sameWidgetSlice(d.children, old.children)
}

// sameWidgetSlice is whether two slices are the same list, as Dart compares lists
// by identity.
func sameWidgetSlice(a, b []Widget) bool {
	if len(a) != len(b) {
		return false
	}
	return len(a) == 0 || &a[0] == &b[0]
}

// ListWheelChildLoopingListDelegate supplies the children from an explicit list,
// over and over: the list is looped.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate
type ListWheelChildLoopingListDelegate struct {
	children []Widget
}

// NewListWheelChildLoopingListDelegate creates a delegate for the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate.new
func NewListWheelChildLoopingListDelegate(children []Widget) *ListWheelChildLoopingListDelegate {
	return &ListWheelChildLoopingListDelegate{children}
}

// Children are the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate.children
func (d *ListWheelChildLoopingListDelegate) Children() []Widget { return d.children }

// EstimatedChildCount is none: the list is infinite.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate.estimatedChildCount
func (d *ListWheelChildLoopingListDelegate) EstimatedChildCount() foundation.Opt[int] {
	return foundation.None[int]()
}

// TrueIndexOf wraps the index around the list.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate.trueIndexOf
func (d *ListWheelChildLoopingListDelegate) TrueIndexOf(index int) int {
	n := len(d.children)
	return ((index % n) + n) % n
}

// Build returns the child that the index stands for, or nil for an empty list.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate.build
func (d *ListWheelChildLoopingListDelegate) Build(_ BuildContext, index int) Widget {
	if len(d.children) == 0 {
		return nil
	}
	return NewIndexedSemantics(index, d.children[d.TrueIndexOf(index)])
}

// ShouldRebuild is whether the list of children is not the same one.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildLoopingListDelegate.shouldRebuild
func (d *ListWheelChildLoopingListDelegate) ShouldRebuild(oldDelegate ListWheelChildDelegate) bool {
	return !sameWidgetSlice(d.children, oldDelegate.(*ListWheelChildLoopingListDelegate).children)
}

// ListWheelChildBuilderDelegateOpts are the optional parameters of
// NewListWheelChildBuilderDelegate.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate.new
type ListWheelChildBuilderDelegateOpts struct {
	// ChildCount is the number of children; none means unbounded, until the
	// builder returns nil.
	ChildCount foundation.Opt[int]
}

// ListWheelChildBuilderDelegate supplies the children with a builder.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate
type ListWheelChildBuilderDelegate struct {
	builder    NullableIndexedWidgetBuilder
	childCount foundation.Opt[int]
}

// NewListWheelChildBuilderDelegate creates a delegate that builds the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate.new
func NewListWheelChildBuilderDelegate(builder NullableIndexedWidgetBuilder, opts ...ListWheelChildBuilderDelegateOpts) *ListWheelChildBuilderDelegate {
	return &ListWheelChildBuilderDelegate{builder, firstOpts(opts).ChildCount}
}

// ChildCount is the number of children, or none.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate.childCount
func (d *ListWheelChildBuilderDelegate) ChildCount() foundation.Opt[int] { return d.childCount }

// EstimatedChildCount is the child count.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate.estimatedChildCount
func (d *ListWheelChildBuilderDelegate) EstimatedChildCount() foundation.Opt[int] {
	return d.childCount
}

// TrueIndexOf is the index itself.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildDelegate.trueIndexOf
func (d *ListWheelChildBuilderDelegate) TrueIndexOf(index int) int { return index }

// Build builds the child at the index, with its semantic index.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate.build
func (d *ListWheelChildBuilderDelegate) Build(bc BuildContext, index int) Widget {
	count, bounded := d.childCount.Get()
	if !bounded {
		if child := d.builder(bc, index); child != nil {
			return NewIndexedSemantics(index, child)
		}
		return nil
	}
	if index < 0 || index >= count {
		return nil
	}
	return NewIndexedSemantics(index, d.builder(bc, index))
}

// ShouldRebuild is true: Go functions can not be compared.
//
// Deviation: Dart compares the builder; Go functions are not comparable, so the
// children are always built again (SPEC §12.10).
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelChildBuilderDelegate.shouldRebuild
func (d *ListWheelChildBuilderDelegate) ShouldRebuild(ListWheelChildDelegate) bool { return true }

// FixedExtentScrollControllerOpts are the optional parameters of
// NewFixedExtentScrollController.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.new
type FixedExtentScrollControllerOpts struct {
	// InitialItem is the index of the item to start at.
	InitialItem int
	// KeepScrollOffset is whether the positions save their offset in the
	// PageStorage; the default is true.
	KeepScrollOffset foundation.Opt[bool]
	// DebugLabel names the controller in debug output.
	DebugLabel string
	// OnAttach is called when a position is attached.
	OnAttach ScrollControllerCallback
	// OnDetach is called when a position is detached.
	OnDetach ScrollControllerCallback
}

// FixedExtentScrollController is a controller for ListWheelScrollView: it works in
// items instead of pixels.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController
type FixedExtentScrollController struct {
	ScrollControllerBase
	initialItem int
}

// NewFixedExtentScrollController creates a controller for a list wheel scroll view.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.new
func NewFixedExtentScrollController(opts ...FixedExtentScrollControllerOpts) *FixedExtentScrollController {
	o := firstOpts(opts)
	c := &FixedExtentScrollController{initialItem: o.InitialItem}
	InitScrollControllerBase(&c.ScrollControllerBase, c, ScrollControllerOpts{KeepScrollOffset: o.KeepScrollOffset, DebugLabel: o.DebugLabel,
		OnAttach: o.OnAttach, OnDetach: o.OnDetach})
	return c
}

// InitialItem is the index of the item to start at.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.initialItem
func (c *FixedExtentScrollController) InitialItem() int { return c.initialItem }

// SelectedItem is the index of the item in the center. In debug builds it panics
// with the tag FixedExtentControllerNotAttached when no scroll view uses the
// controller, and with FixedExtentControllerMultipleViews when more than one does.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.selectedItem
func (c *FixedExtentScrollController) SelectedItem() int {
	if foundation.DebugMode {
		if len(c.positions) == 0 {
			panic(errors.New("FixedExtentControllerNotAttached", "FixedExtentScrollController.selectedItem cannot be accessed before a scroll view is built with it"))
		}
		if len(c.positions) != 1 {
			panic(errors.New("FixedExtentControllerMultipleViews",
				"the selectedItem property cannot be read when multiple scroll views are attached to the same FixedExtentScrollController"))
		}
	}
	return c.Position().(*fixedExtentScrollPosition).ItemIndex()
}

// AnimateToItem animates the controlled scroll views to the item. The future
// completes when the animations end.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.animateToItem
func (c *FixedExtentScrollController) AnimateToItem(itemIndex int, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	if !c.HasClients() {
		return scheduler.NewTickerFutureComplete()
	}
	var futures []*scheduler.TickerFuture
	for _, p := range c.Positions() {
		position := p.(*fixedExtentScrollPosition)
		futures = append(futures, position.AnimateTo(float64(itemIndex)*position.itemExtent(), duration, curve))
	}
	return scheduler.WaitTickerFutures(futures...)
}

// JumpToItem changes which item is in the center without animation.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.jumpToItem
func (c *FixedExtentScrollController) JumpToItem(itemIndex int) {
	for _, p := range c.Positions() {
		position := p.(*fixedExtentScrollPosition)
		position.JumpTo(float64(itemIndex) * position.itemExtent())
	}
}

// CreateScrollPosition creates the position that counts in items.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollController.createScrollPosition
func (c *FixedExtentScrollController) CreateScrollPosition(physics ScrollPhysics, context ScrollContext, oldPosition ScrollPosition) ScrollPosition {
	return newFixedExtentScrollPosition(physics, context, c.initialItem, oldPosition, c.keepScrollOffset, c.debugLabel)
}

// FixedExtentMetrics are the scroll metrics of a ListWheelScrollView: they also
// say the index of the item in the center.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentMetrics
type FixedExtentMetrics interface {
	ScrollMetrics
	// ItemIndex is the index of the item in the center.
	//
	// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentMetrics.itemIndex
	ItemIndex() int
}

// FixedExtentMetricsSnapshot is an immutable snapshot of FixedExtentMetrics. The
// scroll notifications of a ListWheelScrollView carry one.
//
// Deviation: Dart's FixedExtentMetrics is a class; here it is an interface that
// this snapshot and the position of the scroll view implement (SPEC §12.10).
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentMetrics
type FixedExtentMetricsSnapshot struct {
	FixedScrollMetrics
	itemIndex int
}

// NewFixedExtentMetricsSnapshot creates a snapshot.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentMetrics.new
func NewFixedExtentMetricsSnapshot(minScrollExtent, maxScrollExtent, pixels, viewportDimension foundation.Opt[float64], axisDirection painting.AxisDirection, itemIndex int, devicePixelRatio float64) *FixedExtentMetricsSnapshot {
	m := &FixedExtentMetricsSnapshot{itemIndex: itemIndex}
	m.FixedScrollMetrics = *NewFixedScrollMetrics(minScrollExtent, maxScrollExtent, pixels, viewportDimension, axisDirection, devicePixelRatio)
	InitScrollMetricsMixin(&m.ScrollMetricsMixin, m)
	return m
}

// ItemIndex is the index of the item in the center.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentMetrics.itemIndex
func (m *FixedExtentMetricsSnapshot) ItemIndex() int { return m.itemIndex }

// CopyWith is a copy with the changes, and the same item index.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentMetrics.copyWith
func (m *FixedExtentMetricsSnapshot) CopyWith(changes ...ScrollMetricsChanges) ScrollMetrics {
	return fixedExtentMetricsCopy(m, m.itemIndex, changes)
}

func fixedExtentMetricsCopy(m ScrollMetrics, itemIndex int, changes []ScrollMetricsChanges) *FixedExtentMetricsSnapshot {
	f := scrollMetricsCopy(m, changes)
	copied := &FixedExtentMetricsSnapshot{FixedScrollMetrics: *f, itemIndex: itemIndex}
	InitScrollMetricsMixin(&copied.ScrollMetricsMixin, copied)
	return copied
}

func itemFromOffset(offset, itemExtent, minScrollExtent, maxScrollExtent float64) int {
	return int(math.Round(clipOffsetToScrollableRange(offset, minScrollExtent, maxScrollExtent) / itemExtent))
}

func clipOffsetToScrollableRange(offset, minScrollExtent, maxScrollExtent float64) float64 {
	return math.Min(math.Max(offset, minScrollExtent), maxScrollExtent)
}

// fixedExtentScrollPosition is the scroll position of a ListWheelScrollView.
//
// Flutter: widgets/list_wheel_scroll_view.dart _FixedExtentScrollPosition
type fixedExtentScrollPosition struct {
	ScrollPositionWithSingleContext
}

func fixedExtentFromContext(context ScrollContext) float64 {
	scrollable, ok := context.(*ScrollableState)
	if !ok || scrollable.fixedItemExtent().IsNone() {
		panic(errors.New("FixedExtentControllerWrongView", "FixedExtentScrollController can only be used with ListWheelScrollViews"))
	}
	return scrollable.fixedItemExtent().Value()
}

func newFixedExtentScrollPosition(physics ScrollPhysics, context ScrollContext, initialItem int, oldPosition ScrollPosition, keepScrollOffset bool, debugLabel string) *fixedExtentScrollPosition {
	p := &fixedExtentScrollPosition{}
	InitScrollPositionWithSingleContext(&p.ScrollPositionWithSingleContext, p, physics, context, ScrollPositionWithSingleContextOpts{
		InitialPixels: foundation.Some(fixedExtentFromContext(context) * float64(initialItem)), OldPosition: oldPosition,
		KeepScrollOffset: foundation.Some(keepScrollOffset), DebugLabel: debugLabel})
	return p
}

func (p *fixedExtentScrollPosition) itemExtent() float64 { return fixedExtentFromContext(p.context) }

// ItemIndex is the index of the item in the center.
//
// Flutter: widgets/list_wheel_scroll_view.dart _FixedExtentScrollPosition.itemIndex
func (p *fixedExtentScrollPosition) ItemIndex() int {
	return itemFromOffset(p.Pixels(), p.itemExtent(), p.MinScrollExtent(), p.MaxScrollExtent())
}

// CopyWith is a snapshot of the metrics, as fixed extent metrics.
//
// Flutter: widgets/list_wheel_scroll_view.dart _FixedExtentScrollPosition.copyWith
func (p *fixedExtentScrollPosition) CopyWith(changes ...ScrollMetricsChanges) ScrollMetrics {
	return fixedExtentMetricsCopy(p, p.ItemIndex(), changes)
}

// FixedExtentScrollPhysics are the scroll physics of a ListWheelScrollView: they
// settle on an item.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollPhysics
type FixedExtentScrollPhysics struct {
	ScrollPhysicsBase
}

// NewFixedExtentScrollPhysics creates the physics with the parent (nil for none).
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollPhysics.new
func NewFixedExtentScrollPhysics(parent ScrollPhysics) *FixedExtentScrollPhysics {
	p := &FixedExtentScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollPhysics.applyTo
func (p *FixedExtentScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewFixedExtentScrollPhysics(p.BuildParent(ancestor))
}

// CreateBallisticSimulation is a spring or friction simulation that ends on an
// item. In debug builds it panics with the tag FixedExtentPhysicsWrongPosition
// when the position is not the one of a FixedExtentScrollController.
//
// Flutter: widgets/list_wheel_scroll_view.dart FixedExtentScrollPhysics.createBallisticSimulation
func (p *FixedExtentScrollPhysics) CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation {
	metrics, ok := position.(*fixedExtentScrollPosition)
	if !ok {
		panic(errors.New("FixedExtentPhysicsWrongPosition",
			"FixedExtentScrollPhysics can only be used with Scrollables that uses the FixedExtentScrollController"))
	}
	// If we're out of range and not headed back in range, defer to the parent
	// ballistics, which should put us back in range at a item boundary.
	if (velocity <= 0 && metrics.Pixels() <= metrics.MinScrollExtent()) || (velocity >= 0 && metrics.Pixels() >= metrics.MaxScrollExtent()) {
		return p.ScrollPhysicsBase.CreateBallisticSimulation(metrics, velocity)
	}
	// Create a test simulation to see where it would have ballistically fallen
	// naturally without settling onto items.
	testFrictionSimulation := p.ScrollPhysicsBase.CreateBallisticSimulation(metrics, velocity)
	// Scenario 1: If we're going to end back at the same item because initial
	// velocity is too low to break past it, use a spring simulation to get back.
	if !isNil(testFrictionSimulation) {
		end := testFrictionSimulation.X(math.Inf(1))
		if end == metrics.MinScrollExtent() || end == metrics.MaxScrollExtent() {
			return p.ScrollPhysicsBase.CreateBallisticSimulation(metrics, velocity)
		}
	}
	offset := metrics.Pixels()
	if !isNil(testFrictionSimulation) {
		offset = testFrictionSimulation.X(math.Inf(1))
	}
	settlingItemIndex := itemFromOffset(offset, metrics.itemExtent(), metrics.MinScrollExtent(), metrics.MaxScrollExtent())
	settlingPixels := float64(settlingItemIndex) * metrics.itemExtent()
	tolerance := p.ToleranceFor(position)
	// There's no need to simulate (and thus avoid use of a spring) if the
	// settling pixel is equal to the position.
	if math.Abs(velocity) < tolerance.Velocity && math.Abs(settlingPixels-metrics.Pixels()) < tolerance.Distance {
		return nil
	}
	// Spring back to the same item.
	if settlingItemIndex == metrics.ItemIndex() {
		return physics.NewSpringSimulation(p.Spring(), metrics.Pixels(), settlingPixels, velocity, physics.SpringSimulationOpts{Tolerance: foundation.Some(tolerance)})
	}
	// Scenario 2: If we're going to end up at a different item, use friction
	// simulation to go there.
	sign := 1.0
	if velocity < 0 {
		sign = -1
	}
	return physics.NewFrictionSimulationThrough(metrics.Pixels(), settlingPixels, velocity, tolerance.Velocity*sign)
}

// ListWheelScrollViewOpts are the optional parameters of NewListWheelScrollView
// and NewListWheelScrollViewUseDelegate.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.new
type ListWheelScrollViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller reads and controls the scroll position; the default is a
	// FixedExtentScrollController.
	Controller ScrollController
	// Physics is how the wheel responds to the user.
	Physics ScrollPhysics
	// DiameterRatio is the ratio of the diameter of the cylinder to the viewport
	// extent; the default is 2.
	DiameterRatio foundation.Opt[float64]
	// Perspective is how much the cylinder is distorted by the perspective; the
	// default is 0.003.
	Perspective foundation.Opt[float64]
	// OffAxisFraction is how far the cylinder is off the center of the viewport
	// horizontally.
	OffAxisFraction float64
	// UseMagnifier is whether to magnify the item in the center.
	UseMagnifier bool
	// Magnification is how much the center item is magnified; the default is 1.
	Magnification foundation.Opt[float64]
	// OverAndUnderCenterOpacity is the opacity of the items outside of the center
	// magnifier; the default is 1.
	OverAndUnderCenterOpacity foundation.Opt[float64]
	// Squeeze is how tightly the children are packed on the cylinder; the default
	// is 1.
	Squeeze foundation.Opt[float64]
	// OnSelectedItemChanged is called with the index of the item in the center when
	// it changes.
	OnSelectedItemChanged func(index int)
	// RenderChildrenOutsideViewport is whether to build and paint children outside
	// of the viewport.
	RenderChildrenOutsideViewport bool
	// ClipBehavior is how the wheel clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the wheel behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// RestorationID is the identifier for restoring the scroll offset; it has no
	// effect until state restoration is ported.
	RestorationID string
	// ScrollBehavior describes how the wheel scrolls; the default is the one of the
	// ScrollConfiguration above, without scrollbars.
	ScrollBehavior ScrollBehavior
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// ChangeReportingBehavior is when the selected item is reported; the default is
	// ChangeReportingBehaviorOnScrollUpdate.
	ChangeReportingBehavior ChangeReportingBehavior
}

// ListWheelScrollView is a box in which children are on a wheel that can be
// rotated.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView
type ListWheelScrollView struct {
	StatefulWidget
	childDelegate ListWheelChildDelegate
	itemExtent    float64
	o             ListWheelScrollViewOpts
}

func newListWheelScrollView(delegate ListWheelChildDelegate, itemExtent float64, o ListWheelScrollViewOpts) *ListWheelScrollView {
	if foundation.DebugMode {
		switch {
		case !(o.DiameterRatio.OrElse(rendering.ListWheelDefaultDiameterRatio) > 0):
			panic(errors.New("ListWheelViewportInvalid", "you can't set a diameterRatio of 0 or of a negative number; it would imply a cylinder of 0 in diameter in which case nothing will be drawn"))
		case !(o.Perspective.OrElse(rendering.ListWheelDefaultPerspective) > 0) || !(o.Perspective.OrElse(rendering.ListWheelDefaultPerspective) <= 0.01):
			panic(errors.New("ListWheelViewportInvalid", "a perspective too high will be clipped in the z-axis and therefore not renderable; value must be between 0 and 0.01"))
		case !(o.Magnification.OrElse(1) > 0):
			panic(errors.New("ListWheelViewportInvalid", "the magnification must be greater than zero"))
		case !(o.OverAndUnderCenterOpacity.OrElse(1) >= 0 && o.OverAndUnderCenterOpacity.OrElse(1) <= 1):
			panic(errors.New("ListWheelViewportInvalid", "the opacity must be between 0 and 1"))
		case !(itemExtent > 0):
			panic(errors.New("ListWheelViewportInvalid", "the item extent must be greater than zero", "itemExtent", itemExtent))
		case !(o.Squeeze.OrElse(1) > 0):
			panic(errors.New("ListWheelViewportInvalid", "the squeeze must be greater than zero"))
		}
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if foundation.DebugMode && o.RenderChildrenOutsideViewport && o.ClipBehavior != ui.ClipNone {
		panic(errors.New("ListWheelViewportInvalid", "cannot renderChildrenOutsideViewport and clip since children rendered outside will be clipped anyway"))
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	if o.ChangeReportingBehavior == ChangeReportingBehaviorUnspecified {
		o.ChangeReportingBehavior = ChangeReportingBehaviorOnScrollUpdate
	}
	return &ListWheelScrollView{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), delegate, itemExtent, o}
}

// NewListWheelScrollView creates a wheel of the children, each with the item
// extent.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.new
func NewListWheelScrollView(itemExtent float64, children []Widget, opts ...ListWheelScrollViewOpts) *ListWheelScrollView {
	return newListWheelScrollView(NewListWheelChildListDelegate(children), itemExtent, firstOpts(opts))
}

// NewListWheelScrollViewUseDelegate creates a wheel whose children the delegate
// supplies.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.useDelegate
func NewListWheelScrollViewUseDelegate(itemExtent float64, childDelegate ListWheelChildDelegate, opts ...ListWheelScrollViewOpts) *ListWheelScrollView {
	return newListWheelScrollView(childDelegate, itemExtent, firstOpts(opts))
}

// ItemExtent is the extent of every child along the main axis.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.itemExtent
func (w *ListWheelScrollView) ItemExtent() float64 { return w.itemExtent }

// ChildDelegate supplies the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.childDelegate
func (w *ListWheelScrollView) ChildDelegate() ListWheelChildDelegate { return w.childDelegate }

// Controller reads and controls the scroll position, or nil.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.controller
func (w *ListWheelScrollView) Controller() ScrollController { return w.o.Controller }

// CreateState creates the state.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelScrollView.createState
func (w *ListWheelScrollView) CreateState() State { return &listWheelScrollViewState{} }

type listWheelScrollViewState struct {
	StateOf[*ListWheelScrollView]
	lastReportedItemIndex int
	backupController      ScrollController
}

func (s *listWheelScrollViewState) effectiveController() ScrollController {
	if c := s.Widget().o.Controller; !isNil(c) {
		return c
	}
	if s.backupController == nil {
		s.backupController = NewFixedExtentScrollController()
	}
	return s.backupController
}

func (s *listWheelScrollViewState) InitState() {
	if c, ok := s.Widget().o.Controller.(*FixedExtentScrollController); ok {
		s.lastReportedItemIndex = c.initialItem
	}
}

func (s *listWheelScrollViewState) Dispose() {
	if s.backupController != nil {
		s.backupController.Dispose()
	}
}

func (s *listWheelScrollViewState) reportSelectedItemChanged(notification ScrollNotification) {
	metrics := notification.Metrics().(FixedExtentMetrics)
	currentItemIndex := metrics.ItemIndex()
	if currentItemIndex != s.lastReportedItemIndex {
		s.lastReportedItemIndex = currentItemIndex
		trueIndex := s.Widget().childDelegate.TrueIndexOf(currentItemIndex)
		s.Widget().o.OnSelectedItemChanged(trueIndex)
	}
}

func (s *listWheelScrollViewState) handleScrollNotification(notification ScrollNotification) bool {
	if _, ok := notification.Metrics().(FixedExtentMetrics); s.Widget().o.OnSelectedItemChanged == nil || notification.Depth() != 0 || !ok {
		return false
	}
	switch s.Widget().o.ChangeReportingBehavior {
	case ChangeReportingBehaviorOnScrollEnd:
		if _, ok := notification.(*ScrollEndNotification); ok {
			s.reportSelectedItemChanged(notification)
		}
	case ChangeReportingBehaviorOnScrollUpdate:
		if _, ok := notification.(*ScrollUpdateNotification); ok {
			s.reportSelectedItemChanged(notification)
		}
	}
	return false
}

func (s *listWheelScrollViewState) Build(bc BuildContext) Widget {
	w := s.Widget()
	scrollBehavior := w.o.ScrollBehavior
	if scrollBehavior == nil {
		scrollBehavior = ScrollConfigurationOf(bc).CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
	}
	scrollable := NewScrollable(func(bc BuildContext, offset rendering.ViewportOffset) Widget {
		return NewListWheelViewport(offset, w.itemExtent, w.childDelegate, ListWheelViewportOpts{
			DiameterRatio: w.o.DiameterRatio, Perspective: w.o.Perspective, OffAxisFraction: w.o.OffAxisFraction, UseMagnifier: w.o.UseMagnifier,
			Magnification: w.o.Magnification, OverAndUnderCenterOpacity: w.o.OverAndUnderCenterOpacity, Squeeze: w.o.Squeeze,
			RenderChildrenOutsideViewport: w.o.RenderChildrenOutsideViewport, ClipBehavior: w.o.ClipBehavior})
	}, ScrollableOpts{Controller: s.effectiveController(), Physics: w.o.Physics, HitTestBehavior: w.o.HitTestBehavior,
		ScrollBehavior: scrollBehavior, DragStartBehavior: w.o.DragStartBehavior})
	scrollable.fixedItemExtent = foundation.Some(w.itemExtent)
	return NewNotificationListener[ScrollNotification](scrollable, s.handleScrollNotification)
}

// ListWheelViewportOpts are the optional parameters of NewListWheelViewport.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.new
type ListWheelViewportOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DiameterRatio is the ratio of the diameter of the cylinder to the viewport
	// extent; the default is 2.
	DiameterRatio foundation.Opt[float64]
	// Perspective is how much the cylinder is distorted by the perspective; the
	// default is 0.003.
	Perspective foundation.Opt[float64]
	// OffAxisFraction is how far the cylinder is off the center of the viewport
	// horizontally.
	OffAxisFraction float64
	// UseMagnifier is whether to magnify the item in the center.
	UseMagnifier bool
	// Magnification is how much the center item is magnified; the default is 1.
	Magnification foundation.Opt[float64]
	// OverAndUnderCenterOpacity is the opacity of the items outside of the center
	// magnifier; the default is 1.
	OverAndUnderCenterOpacity foundation.Opt[float64]
	// Squeeze is how tightly the children are packed on the cylinder; the default
	// is 1.
	Squeeze foundation.Opt[float64]
	// RenderChildrenOutsideViewport is whether to build and paint children outside
	// of the viewport.
	RenderChildrenOutsideViewport bool
	// ClipBehavior is how the viewport clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// ListWheelViewport is the widget of a viewport that shows its children on a
// cylinder; a ListWheelScrollView builds one.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport
type ListWheelViewport struct {
	RenderObjectWidget
	itemExtent    float64
	offset        rendering.ViewportOffset
	childDelegate ListWheelChildDelegate
	o             ListWheelViewportOpts
}

// NewListWheelViewport creates a viewport of the children that the delegate
// supplies, at the offset.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.new
func NewListWheelViewport(offset rendering.ViewportOffset, itemExtent float64, childDelegate ListWheelChildDelegate, opts ...ListWheelViewportOpts) *ListWheelViewport {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &ListWheelViewport{NewRenderObjectWidgetBase(WidgetOpts{Key: o.Key}), itemExtent, offset, childDelegate, o}
}

// ItemExtent is the extent of every child along the main axis.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.itemExtent
func (w *ListWheelViewport) ItemExtent() float64 { return w.itemExtent }

// Offset is the offset of the viewport.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.offset
func (w *ListWheelViewport) Offset() rendering.ViewportOffset { return w.offset }

// ChildDelegate supplies the children.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.childDelegate
func (w *ListWheelViewport) ChildDelegate() ListWheelChildDelegate { return w.childDelegate }

// CreateElement creates the element, which is the child manager of the render
// object.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.createElement
func (w *ListWheelViewport) CreateElement() Element {
	e := &ListWheelElement{childWidgets: map[int]Widget{}, childElements: newIntKeyedElements()}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.createRenderObject
func (w *ListWheelViewport) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderListWheelViewport(bc.(rendering.ListWheelChildManager), w.offset, w.itemExtent, rendering.RenderListWheelViewportOpts{
		DiameterRatio: w.o.DiameterRatio, Perspective: w.o.Perspective, OffAxisFraction: w.o.OffAxisFraction, UseMagnifier: w.o.UseMagnifier,
		Magnification: w.o.Magnification, OverAndUnderCenterOpacity: w.o.OverAndUnderCenterOpacity, Squeeze: w.o.Squeeze,
		RenderChildrenOutsideViewport: w.o.RenderChildrenOutsideViewport, ClipBehavior: w.o.ClipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelViewport.updateRenderObject
func (w *ListWheelViewport) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderListWheelViewport)
	r.SetOffset(w.offset)
	r.SetDiameterRatio(w.o.DiameterRatio.OrElse(rendering.ListWheelDefaultDiameterRatio))
	r.SetPerspective(w.o.Perspective.OrElse(rendering.ListWheelDefaultPerspective))
	r.SetOffAxisFraction(w.o.OffAxisFraction)
	r.SetUseMagnifier(w.o.UseMagnifier)
	r.SetMagnification(w.o.Magnification.OrElse(1))
	r.SetOverAndUnderCenterOpacity(w.o.OverAndUnderCenterOpacity.OrElse(1))
	r.SetItemExtent(w.itemExtent)
	r.SetSqueeze(w.o.Squeeze.OrElse(1))
	r.SetRenderChildrenOutsideViewport(w.o.RenderChildrenOutsideViewport)
	r.SetClipBehavior(w.o.ClipBehavior)
}

// ListWheelElement is the element of a ListWheelViewport: it builds the children
// that the render object asks for.
//
// Flutter: widgets/list_wheel_scroll_view.dart ListWheelElement
type ListWheelElement struct {
	RenderObjectElement
	// childWidgets caches the widgets that the delegate built, until the next
	// rebuild; a nil entry stands for "no child at this index".
	childWidgets  map[int]Widget
	childElements *intKeyedElements
}

func (e *ListWheelElement) viewport() *rendering.RenderListWheelViewport {
	return e.renderObject.(*rendering.RenderListWheelViewport)
}

// Update rebuilds the children when the delegate says it must.
func (e *ListWheelElement) Update(newWidget Widget) {
	oldDelegate := e.widget.(*ListWheelViewport).childDelegate
	e.RenderObjectElement.Update(newWidget)
	newDelegate := newWidget.(*ListWheelViewport).childDelegate
	if newDelegate != oldDelegate && (foundation.RuntimeTypeName(newDelegate) != foundation.RuntimeTypeName(oldDelegate) || newDelegate.ShouldRebuild(oldDelegate)) {
		e.PerformRebuild()
		// Force the callback to be called, even if the layout constraints are the
		// same, because the logic in the delegate might be different.
		e.viewport().MarkNeedsLayout()
	}
}

// ChildCount is the number of children that the delegate says.
func (e *ListWheelElement) ChildCount() foundation.Opt[int] {
	return e.widget.(*ListWheelViewport).childDelegate.EstimatedChildCount()
}

// PerformRebuild builds the children again.
func (e *ListWheelElement) PerformRebuild() {
	e.childWidgets = map[int]Widget{}
	e.RenderObjectElement.PerformRebuild()
	first, ok := e.childElements.firstKey()
	if !ok {
		return
	}
	last, _ := e.childElements.lastKey()
	for index := first; index <= last; index++ {
		newChild := e.UpdateChild(e.childElements.get(index), e.retrieveWidget(index), index)
		if newChild != nil {
			e.childElements.set(index, newChild)
		} else {
			e.childElements.remove(index)
		}
	}
}

func (e *ListWheelElement) retrieveWidget(index int) Widget {
	if w, ok := e.childWidgets[index]; ok {
		return w
	}
	w := e.widget.(*ListWheelViewport).childDelegate.Build(e, index)
	e.childWidgets[index] = w
	return w
}

// ChildExistsAt is whether the delegate has a child at the index.
func (e *ListWheelElement) ChildExistsAt(index int) bool { return e.retrieveWidget(index) != nil }

// CreateChild builds the child at the index.
func (e *ListWheelElement) CreateChild(index int, after rendering.Box) {
	e.owner.BuildScope(e, func() {
		if foundation.DebugMode && after != nil && e.childElements.get(index-1) == nil {
			panic(errors.New("ListWheelChildBeforeMissing", "the child before the one to create is not live", "index", index))
		}
		newChild := e.UpdateChild(e.childElements.get(index), e.retrieveWidget(index), index)
		if newChild != nil {
			e.childElements.set(index, newChild)
		} else {
			e.childElements.remove(index)
		}
	})
}

// RemoveChild removes the child that the render object gives up.
func (e *ListWheelElement) RemoveChild(child rendering.Box) {
	index := e.viewport().IndexOf(child)
	e.owner.BuildScope(e, func() {
		e.UpdateChild(e.childElements.get(index), nil, index)
		e.childElements.remove(index)
	})
}

// UpdateChild updates the child, and gives its parent data the index and the
// offset of the old child.
func (e *ListWheelElement) UpdateChild(child Element, newWidget Widget, newSlot any) Element {
	var oldParentData *rendering.ListWheelParentData
	if child != nil && child.RenderObject() != nil {
		oldParentData = child.RenderObject().ParentData().(*rendering.ListWheelParentData)
	}
	newChild := e.RenderObjectElement.UpdateChild(child, newWidget, newSlot)
	if newChild != nil && newChild.RenderObject() != nil {
		newParentData := newChild.RenderObject().ParentData().(*rendering.ListWheelParentData)
		newParentData.Index = foundation.Some(newSlot.(int))
		if oldParentData != nil {
			newParentData.Offset = oldParentData.Offset
		}
	}
	return newChild
}

// InsertRenderObjectChild inserts the child after the one before it.
func (e *ListWheelElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	var after rendering.Box
	if before := e.childElements.get(slot.(int) - 1); before != nil {
		after, _ = before.RenderObject().(rendering.Box)
	}
	e.viewport().Insert(child.(rendering.Box), after)
}

// MoveRenderObjectChild panics: the list is kept in contiguous increasing order,
// so moving children around is not allowed.
func (e *ListWheelElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {
	panic(errors.New("ListWheelChildMoved", "currently we maintain the list in contiguous increasing order, so moving children around is not allowed"))
}

// RemoveRenderObjectChild removes the child.
func (e *ListWheelElement) RemoveRenderObjectChild(child rendering.RenderObject, _ any) {
	e.viewport().Remove(child.(rendering.Box))
}

// VisitChildren visits the children in order of index.
func (e *ListWheelElement) VisitChildren(visitor ElementVisitor) {
	for _, index := range e.childElements.keys() {
		visitor(e.childElements.get(index))
	}
}

// ForgetChild drops the child.
func (e *ListWheelElement) ForgetChild(child Element) {
	slot, _ := child.AsElement().slot.(int)
	e.childElements.remove(slot)
	e.RenderObjectElement.ForgetChild(child)
}
