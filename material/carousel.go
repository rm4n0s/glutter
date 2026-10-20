// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CarouselViewOpts are the optional parameters of the constructors of CarouselView.
//
// Flutter: material/carousel.dart CarouselView
type CarouselViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Padding is the empty space around each item; the default is 4 on all sides.
	Padding painting.EdgeInsetsGeometry
	// BackgroundColor is the color of the items; the surface color by default.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the items; the default is 0.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the items; a rounded rectangle of 28 by default.
	Shape painting.ShapeBorder
	// ItemClipBehavior is how the items are clipped; the default is antiAlias.
	ItemClipBehavior ui.Clip
	// OverlayColor is the color of the overlay of the items, for each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ItemSnapping makes the scroll settle on a whole item.
	ItemSnapping bool
	// ShrinkExtent is the smallest extent that an item shrinks to while it
	// scrolls away.
	ShrinkExtent float64
	// Controller is the controller of the scroll.
	Controller *CarouselController
	// ScrollDirection is the axis of the scroll; the default is horizontal.
	ScrollDirection painting.Axis
	// Reverse scrolls against the reading direction.
	Reverse bool
	// NoConsumeMaxWeight makes the first item start at its smallest weight instead
	// of its largest: Flutter's consumeMaxWeight is true by default. It is used by
	// the weighted constructors.
	NoConsumeMaxWeight bool
	// OnTap is called with the index of the item that the user taps.
	OnTap func(index int)
	// DisableSplash removes the ink splash of the items: Flutter's enableSplash is
	// true by default.
	DisableSplash bool
	// Infinite makes the items repeat without end.
	Infinite bool
	// ItemCount is the number of items of a builder constructor; none is
	// unbounded.
	ItemCount foundation.Opt[int]
	// OnIndexChanged is called with the index of the leading item when it
	// changes.
	OnIndexChanged func(index int)
}

// CarouselView is a Material Design carousel: a scrollable list of items that
// shrink and grow as they move. The fixed constructors have items of a fixed
// extent that shrink when they leave; the weighted ones divide the viewport by
// weights.
//
// Flutter: material/carousel.dart CarouselView
type CarouselView struct {
	widgets.StatefulWidget
	itemExtent  foundation.Opt[float64]
	flexWeights []int
	children    []widgets.Widget
	itemBuilder widgets.NullableIndexedWidgetBuilder
	o           CarouselViewOpts
}

// NewCarouselView creates a carousel whose items have the extent. In debug
// builds the tag CarouselViewItemsEmpty is for no children when infinite.
//
// Flutter: material/carousel.dart CarouselView.new
func NewCarouselView(itemExtent float64, children []widgets.Widget, opts ...CarouselViewOpts) *CarouselView {
	o := firstOpts(opts)
	return &CarouselView{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), itemExtent: foundation.Some(itemExtent), children: children, o: o}
}

// NewCarouselViewWeighted creates a carousel whose items share the extent of the
// viewport in the proportions of the weights.
//
// Flutter: material/carousel.dart CarouselView.weighted
func NewCarouselViewWeighted(flexWeights []int, children []widgets.Widget, opts ...CarouselViewOpts) *CarouselView {
	o := firstOpts(opts)
	return &CarouselView{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), flexWeights: flexWeights, children: children, o: o}
}

// NewCarouselViewBuilder creates a carousel whose items of the extent are built on
// demand.
//
// Flutter: material/carousel.dart CarouselView.builder
func NewCarouselViewBuilder(itemExtent float64, itemBuilder widgets.NullableIndexedWidgetBuilder, opts ...CarouselViewOpts) *CarouselView {
	o := firstOpts(opts)
	return &CarouselView{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), itemExtent: foundation.Some(itemExtent), itemBuilder: itemBuilder, o: o}
}

// NewCarouselViewWeightedBuilder creates a weighted carousel whose items are built
// on demand.
//
// Flutter: material/carousel.dart CarouselView.weightedBuilder
func NewCarouselViewWeightedBuilder(flexWeights []int, itemBuilder widgets.NullableIndexedWidgetBuilder, opts ...CarouselViewOpts) *CarouselView {
	o := firstOpts(opts)
	return &CarouselView{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), flexWeights: flexWeights, itemBuilder: itemBuilder, o: o}
}

// CreateState creates the state.
//
// Flutter: material/carousel.dart CarouselView.createState
func (w *CarouselView) CreateState() widgets.State { return &carouselViewState{} }

func (w *CarouselView) consumeMaxWeight() bool { return !w.o.NoConsumeMaxWeight }

type carouselViewState struct {
	widgets.StateOf[*CarouselView]
	itemExtent          foundation.Opt[float64]
	internalController  *CarouselController
	lastReportedLeading int
	scrollListener      foundation.ListenerHandle
	attachedController  *CarouselController
}

func (s *carouselViewState) controller() *CarouselController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.internalController
}

func (s *carouselViewState) InitState() {
	w := s.Widget()
	s.itemExtent = w.itemExtent
	if w.o.Controller == nil {
		s.internalController = NewCarouselController()
	}
	s.lastReportedLeading = s.initialLeadingItem()
	s.attach()
}

func (s *carouselViewState) attach() {
	c := s.controller()
	s.attachedController = c
	c.attachState(s)
	s.scrollListener = c.AddListener(s.handleScroll)
}

func (s *carouselViewState) detach() {
	if c := s.attachedController; c != nil {
		c.RemoveListener(s.scrollListener)
		c.detachState(s)
	}
}

func (s *carouselViewState) DidUpdateWidget(old *CarouselView) {
	w := s.Widget()
	if w.o.Controller != old.o.Controller {
		s.detach()
		if w.o.Controller != nil {
			if s.internalController != nil {
				s.internalController.Dispose()
			}
			s.internalController = nil
		} else {
			s.internalController = NewCarouselController()
		}
		s.attach()
	}
	pos, _ := s.controller().carouselPosition()
	if !slices.Equal(w.flexWeights, old.flexWeights) && pos != nil {
		pos.setFlexWeights(w.flexWeights)
	}
	if w.itemExtent != old.itemExtent {
		s.itemExtent = w.itemExtent
		if pos != nil {
			pos.setItemExtent(s.itemExtent)
		}
	}
	if w.consumeMaxWeight() != old.consumeMaxWeight() && pos != nil {
		pos.setConsumeMaxWeight(w.consumeMaxWeight())
	}
}

func (s *carouselViewState) Dispose() {
	s.detach()
	if s.internalController != nil {
		s.internalController.Dispose()
	}
}

func (s *carouselViewState) handleScroll() {
	w := s.Widget()
	if w.o.OnIndexChanged == nil {
		return
	}
	pos, ok := s.controller().carouselPosition()
	if !ok {
		return
	}
	current := pos.leadingItem()
	if current != s.lastReportedLeading {
		s.lastReportedLeading = current
		w.o.OnIndexChanged(current)
	}
}

func (s *carouselViewState) initialLeadingItem() int {
	w := s.Widget()
	if w.flexWeights != nil {
		maxWeight := slices.Max(w.flexWeights)
		return max(s.controller().initialItem-slices.Index(w.flexWeights, maxWeight), 0)
	}
	return s.controller().initialItem
}

func (s *carouselViewState) buildCarouselItem(bc widgets.BuildContext, index int) widgets.Widget {
	w := s.Widget()
	if w.o.Infinite && len(w.children) > 0 {
		index = index % len(w.children)
	}
	carouselTheme := CarouselViewThemeOf(bc)
	colorScheme := ThemeOf(bc).ColorScheme
	padding := firstNonNil(w.o.Padding, carouselTheme.Padding, painting.EdgeInsetsGeometry(painting.NewEdgeInsetsAll(4.0)))
	background := firstOpt(w.o.BackgroundColor, carouselTheme.BackgroundColor, foundation.Some(colorScheme.Surface())).Value()
	elevation := firstOpt(w.o.Elevation, carouselTheme.Elevation, foundation.Some(0.0)).Value()
	shape := firstNonNil(w.o.Shape, carouselTheme.Shape, painting.ShapeBorder(painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{
		BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(28.0))})))
	clip := w.o.ItemClipBehavior
	if clip == ui.ClipUnspecified {
		clip = carouselTheme.ItemClipBehavior
	}
	if clip == ui.ClipUnspecified {
		clip = ui.ClipAntiAlias
	}
	overlay := firstNonNilState(w.o.OverlayColor, carouselTheme.OverlayColor)
	if overlay == nil {
		overlay = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(colorScheme.OnSurface().WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(colorScheme.OnSurface().WithOpacity(0.08))
			case states.Has(widgets.WidgetStateFocused):
				return foundation.Some(colorScheme.OnSurface().WithOpacity(0.1))
			}
			return foundation.None[ui.Color]()
		})
	}
	contents := w.children[index]
	onTap := func() {
		if w.o.OnTap != nil {
			w.o.OnTap(index)
		}
	}
	if !w.o.DisableSplash {
		contents = widgets.NewStack([]widgets.Widget{contents,
			NewMaterial(NewInkWell(nil, InkWellOpts{OnTap: onTap, OverlayColor: overlay}), MaterialOpts{Color: foundation.Some(Colors.Transparent)})},
			widgets.StackOpts{Fit: rendering.StackFitExpand})
	} else if w.o.OnTap != nil {
		contents = widgets.NewGestureDetector(contents, widgets.GestureDetectorOpts{OnTap: onTap})
	}
	return widgets.NewPadding(padding, NewMaterial(contents, MaterialOpts{ClipBehavior: clip, Color: foundation.Some(background), Elevation: elevation, Shape: shape}))
}

func (s *carouselViewState) buildSliverCarousel() widgets.Widget {
	w := s.Widget()
	childCount := foundation.None[int]()
	switch {
	case w.o.Infinite:
	case w.itemBuilder != nil:
		childCount = w.o.ItemCount
	default:
		childCount = foundation.Some(len(w.children))
	}
	var builder widgets.NullableIndexedWidgetBuilder
	switch {
	case w.itemBuilder != nil && w.o.Infinite && w.o.ItemCount.OrElse(0) > 0:
		count := w.o.ItemCount.Value()
		builder = func(bc widgets.BuildContext, index int) widgets.Widget { return w.itemBuilder(bc, index%count) }
	case w.itemBuilder != nil:
		builder = w.itemBuilder
	default:
		builder = func(bc widgets.BuildContext, index int) widgets.Widget { return s.buildCarouselItem(bc, index) }
	}
	delegate := widgets.NewSliverChildBuilderDelegate(builder, widgets.SliverChildBuilderDelegateOpts{ChildCount: childCount})
	if extent, ok := s.itemExtent.Get(); ok {
		return &sliverFixedExtentCarousel{SliverMultiBoxAdaptorWidget: widgets.NewSliverMultiBoxAdaptorWidgetBase(delegate),
			itemExtent: extent, minExtent: w.o.ShrinkExtent, infinite: w.o.Infinite}
	}
	if foundation.DebugMode {
		ok := w.flexWeights != nil
		for _, weight := range w.flexWeights {
			ok = ok && weight > 0
		}
		if !ok {
			panic(errors.New("CarouselViewFlexWeightsInvalid", "flexWeights is null or it contains non-positive integers", "flexWeights", w.flexWeights))
		}
	}
	return &sliverWeightedCarousel{SliverMultiBoxAdaptorWidget: widgets.NewSliverMultiBoxAdaptorWidgetBase(delegate),
		consumeMaxWeight: w.consumeMaxWeight(), shrinkExtent: w.o.ShrinkExtent, weights: w.flexWeights, infinite: w.o.Infinite}
}

func (s *carouselViewState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var physicsOf widgets.ScrollPhysics
	if w.o.ItemSnapping {
		physicsOf = NewCarouselScrollPhysics(nil)
	} else {
		physicsOf = widgets.ScrollConfigurationOf(bc).GetScrollPhysics(bc)
	}
	direction := w.o.ScrollDirection
	if direction == painting.AxisUnspecified {
		direction = painting.AxisHorizontal
	}
	return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		mainAxisExtent := constraints.MaxWidth
		if direction == painting.AxisVertical {
			mainAxisExtent = constraints.MaxHeight
		}
		if e, ok := w.itemExtent.Get(); ok {
			s.itemExtent = foundation.Some(ui.ClampDouble(e, 0, mainAxisExtent))
		} else {
			s.itemExtent = foundation.None[float64]()
		}
		return widgets.NewCustomScrollView([]widgets.Widget{s.buildSliverCarousel()}, widgets.CustomScrollViewOpts{
			ScrollDirection: direction, Reverse: w.o.Reverse, Controller: s.controller(), Physics: physicsOf, ClipBehavior: ui.ClipAntiAlias,
			ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentViewport(0.0))})
	})
}

// sliverFixedExtentCarousel lays out the items of the carousel of a fixed extent.
//
// Flutter: material/carousel.dart _SliverFixedExtentCarousel
type sliverFixedExtentCarousel struct {
	widgets.SliverMultiBoxAdaptorWidget
	itemExtent float64
	minExtent  float64
	infinite   bool
}

func (w *sliverFixedExtentCarousel) CreateElement() widgets.Element {
	return widgets.NewSliverMultiBoxAdaptorElement(w, false)
}

func (w *sliverFixedExtentCarousel) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	return newRenderSliverFixedExtentCarousel(bc.(rendering.RenderSliverBoxChildManager), w.minExtent, w.itemExtent, w.infinite)
}

func (w *sliverFixedExtentCarousel) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderSliverFixedExtentCarousel)
	r.setMaxExtent(w.itemExtent)
	r.setMinExtent(w.minExtent)
	r.setInfinite(w.infinite)
}

// renderSliverFixedExtentCarousel is the render object of the carousel of a fixed
// extent: the first item shrinks as it leaves and the last one grows as it
// enters.
//
// Flutter: material/carousel.dart _RenderSliverFixedExtentCarousel
type renderSliverFixedExtentCarousel struct {
	rendering.RenderSliverFixedExtentBoxAdaptor
	maxExtent, minExtent float64
	infinite             bool
}

func newRenderSliverFixedExtentCarousel(cm rendering.RenderSliverBoxChildManager, minExtent, maxExtent float64, infinite bool) *renderSliverFixedExtentCarousel {
	r := &renderSliverFixedExtentCarousel{maxExtent: maxExtent, minExtent: minExtent, infinite: infinite}
	rendering.InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, cm)
	return r
}

func (r *renderSliverFixedExtentCarousel) setMaxExtent(v float64) {
	if r.maxExtent != v {
		r.maxExtent = v
		r.MarkNeedsLayout()
	}
}

func (r *renderSliverFixedExtentCarousel) setMinExtent(v float64) {
	if r.minExtent != v {
		r.minExtent = v
		r.MarkNeedsLayout()
	}
}

func (r *renderSliverFixedExtentCarousel) setInfinite(v bool) {
	if r.infinite != v {
		r.infinite = v
		r.MarkNeedsLayout()
	}
}

func (r *renderSliverFixedExtentCarousel) ItemExtent() foundation.Opt[float64] {
	return foundation.None[float64]()
}

func (r *renderSliverFixedExtentCarousel) ItemExtentBuilder() rendering.ItemExtentBuilder {
	return func(index int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		return foundation.Some(r.buildItemExtent(index))
	}
}

func (r *renderSliverFixedExtentCarousel) buildItemExtent(index int) float64 {
	if r.maxExtent == 0.0 {
		return r.maxExtent
	}
	constraints := r.SliverConstraints()
	firstVisibleIndex := int(math.Floor(constraints.ScrollOffset / r.maxExtent))
	offscreenItems := int(math.Floor(constraints.ScrollOffset / r.maxExtent))
	offscreenExtent := constraints.ScrollOffset - float64(offscreenItems)*r.maxExtent
	effectiveMinExtent := math.Max(math.Mod(constraints.RemainingPaintExtent, r.maxExtent), r.minExtent)
	if index == firstVisibleIndex {
		return math.Max(r.maxExtent-offscreenExtent, effectiveMinExtent)
	}
	scrollOffsetForLastIndex := constraints.ScrollOffset + constraints.RemainingPaintExtent
	if index == r.OverrideMaxChildIndexForScrollOffset(scrollOffsetForLastIndex) {
		return ui.ClampDouble(scrollOffsetForLastIndex-r.maxExtent*float64(index), effectiveMinExtent, r.maxExtent)
	}
	return r.maxExtent
}

// OverrideIndexToLayoutOffset is the layout offset of the item: the first
// visible item stays at the leading edge while it shrinks.
//
// Flutter: material/carousel.dart _RenderSliverFixedExtentCarousel.indexToLayoutOffset
func (r *renderSliverFixedExtentCarousel) OverrideIndexToLayoutOffset(index int) float64 {
	if r.maxExtent == 0.0 {
		return r.maxExtent
	}
	constraints := r.SliverConstraints()
	firstVisibleIndex := int(math.Floor(constraints.ScrollOffset / r.maxExtent))
	effectiveMinExtent := math.Max(math.Mod(constraints.RemainingPaintExtent, r.maxExtent), r.minExtent)
	if index == firstVisibleIndex {
		first := r.buildItemExtent(index)
		if first <= effectiveMinExtent {
			return r.maxExtent*float64(index) - effectiveMinExtent + r.maxExtent
		}
		return constraints.ScrollOffset
	}
	return r.maxExtent * float64(index)
}

// OverrideMinChildIndexForScrollOffset is the index of the first visible item.
//
// Flutter: material/carousel.dart _RenderSliverFixedExtentCarousel.getMinChildIndexForScrollOffset
func (r *renderSliverFixedExtentCarousel) OverrideMinChildIndexForScrollOffset(scrollOffset float64) int {
	if r.maxExtent == 0.0 {
		return 0
	}
	return max(int(math.Floor(scrollOffset/r.maxExtent)), 0)
}

// OverrideMaxChildIndexForScrollOffset is the index of the last visible item.
//
// Flutter: material/carousel.dart _RenderSliverFixedExtentCarousel.getMaxChildIndexForScrollOffset
func (r *renderSliverFixedExtentCarousel) OverrideMaxChildIndexForScrollOffset(scrollOffset float64) int {
	if r.maxExtent > 0.0 {
		actual := scrollOffset/r.maxExtent - 1
		round := math.Round(actual)
		if math.Abs(actual*r.maxExtent-round*r.maxExtent) < foundation.PrecisionErrorTolerance {
			return int(math.Max(0, round))
		}
		return int(math.Max(0, math.Ceil(actual)))
	}
	return 0
}

// OverrideComputeMaxScrollOffset is the extent of all the items at their full
// size.
//
// Flutter: material/carousel.dart _RenderSliverFixedExtentCarousel.computeMaxScrollOffset
func (r *renderSliverFixedExtentCarousel) OverrideComputeMaxScrollOffset(rendering.SliverConstraints) float64 {
	return float64(r.ChildManager().ChildCount()) * r.maxExtent
}

// sliverWeightedCarousel lays out the items of a carousel whose items share the
// viewport by weights.
//
// Flutter: material/carousel.dart _SliverWeightedCarousel
type sliverWeightedCarousel struct {
	widgets.SliverMultiBoxAdaptorWidget
	consumeMaxWeight bool
	shrinkExtent     float64
	weights          []int
	infinite         bool
}

func (w *sliverWeightedCarousel) CreateElement() widgets.Element {
	return widgets.NewSliverMultiBoxAdaptorElement(w, false)
}

func (w *sliverWeightedCarousel) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderSliverWeightedCarousel{consumeMaxWeight: w.consumeMaxWeight, shrinkExtent: w.shrinkExtent, weights: w.weights, infinite: w.infinite}
	rendering.InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, bc.(rendering.RenderSliverBoxChildManager))
	return r
}

func (w *sliverWeightedCarousel) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderSliverWeightedCarousel)
	if r.consumeMaxWeight != w.consumeMaxWeight || r.shrinkExtent != w.shrinkExtent || !slices.Equal(r.weights, w.weights) || r.infinite != w.infinite {
		r.consumeMaxWeight, r.shrinkExtent, r.weights, r.infinite = w.consumeMaxWeight, w.shrinkExtent, w.weights, w.infinite
		r.MarkNeedsLayout()
	}
}

// renderSliverWeightedCarousel is the render object of a weighted carousel.
//
// Flutter: material/carousel.dart _RenderSliverWeightedCarousel
type renderSliverWeightedCarousel struct {
	rendering.RenderSliverFixedExtentBoxAdaptor
	consumeMaxWeight bool
	shrinkExtent     float64
	weights          []int
	infinite         bool
}

func (r *renderSliverWeightedCarousel) ItemExtent() foundation.Opt[float64] {
	return foundation.None[float64]()
}

func (r *renderSliverWeightedCarousel) ItemExtentBuilder() rendering.ItemExtentBuilder {
	return func(index int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		return foundation.Some(r.buildItemExtent(index))
	}
}

func (r *renderSliverWeightedCarousel) maxWeight() int { return slices.Max(r.weights) }

func (r *renderSliverWeightedCarousel) sumWeights() int {
	total := 0
	for _, w := range r.weights {
		total += w
	}
	return total
}

func (r *renderSliverWeightedCarousel) buildItemExtent(index int) float64 {
	constraints := r.SliverConstraints()
	if constraints.ViewportMainAxisExtent == 0 {
		return 0
	}
	first := r.firstVisibleItemIndex()
	var extent float64
	switch {
	case index == first:
		// The first item shrinks until it reaches the shrink extent.
		extent = math.Max(r.distanceToLeadingEdge(), r.effectiveShrinkExtent())
	case index > first && index-first+1 <= len(r.weights):
		cur := index - first
		currWeight := r.weights[cur]
		extent = r.extentUnit() * float64(currWeight) // initial extent
		progress := r.firstVisibleItemOffscreenExtent() / r.firstChildExtent()
		prevWeight := r.weights[cur-1]
		finalIncrease := float64(prevWeight-currWeight) / float64(r.maxWeight())
		extent += finalIncrease * progress * r.maxChildExtent()
	case index > first && index-first+1 > len(r.weights):
		// The extent of the items after the last weight takes what is left.
		visibleTotal := r.distanceToLeadingEdge()
		for i := first + 1; i < index; i++ {
			visibleTotal += r.buildItemExtent(i)
		}
		extent = math.Max(constraints.RemainingPaintExtent-visibleTotal, r.effectiveShrinkExtent())
	default:
		extent = math.Max(r.minChildExtent(), r.effectiveShrinkExtent())
	}
	return extent
}

func (r *renderSliverWeightedCarousel) extentUnit() float64 {
	return r.SliverConstraints().ViewportMainAxisExtent / float64(r.sumWeights())
}
func (r *renderSliverWeightedCarousel) firstChildExtent() float64 {
	return float64(r.weights[0]) * r.extentUnit()
}
func (r *renderSliverWeightedCarousel) maxChildExtent() float64 {
	return float64(r.maxWeight()) * r.extentUnit()
}
func (r *renderSliverWeightedCarousel) minChildExtent() float64 {
	return float64(slices.Min(r.weights)) * r.extentUnit()
}
func (r *renderSliverWeightedCarousel) effectiveShrinkExtent() float64 {
	return ui.ClampDouble(r.shrinkExtent, 0, r.minChildExtent())
}

// scrolledItem is the index of the item at the leading edge, from the scroll
// offset and the extent of the first child.
func (r *renderSliverWeightedCarousel) scrolledItem() int {
	actual := r.SliverConstraints().ScrollOffset / r.firstChildExtent()
	round := math.Round(actual)
	if math.Abs(actual-round) < foundation.PrecisionErrorTolerance {
		return int(round)
	}
	return int(math.Floor(actual))
}

func (r *renderSliverWeightedCarousel) firstVisibleItemIndex() int {
	if r.SliverConstraints().ViewportMainAxisExtent == 0.0 {
		return 0
	}
	smallerWeightCount := 0
	for _, weight := range r.weights {
		if weight == r.maxWeight() {
			break
		}
		smallerWeightCount++
	}
	index := r.scrolledItem()
	if r.consumeMaxWeight {
		return index - smallerWeightCount
	}
	return index
}

func (r *renderSliverWeightedCarousel) firstVisibleItemOffscreenExtent() float64 {
	if r.SliverConstraints().ViewportMainAxisExtent == 0.0 {
		return 0
	}
	return r.SliverConstraints().ScrollOffset - float64(r.scrolledItem())*r.firstChildExtent()
}

func (r *renderSliverWeightedCarousel) distanceToLeadingEdge() float64 {
	return r.firstChildExtent() - r.firstVisibleItemOffscreenExtent()
}

// OverrideIndexToLayoutOffset is the layout offset of the item.
//
// Flutter: material/carousel.dart _RenderSliverWeightedCarousel.indexToLayoutOffset
func (r *renderSliverWeightedCarousel) OverrideIndexToLayoutOffset(index int) float64 {
	constraints := r.SliverConstraints()
	first := r.firstVisibleItemIndex()
	if index == first {
		if r.distanceToLeadingEdge() <= r.effectiveShrinkExtent() {
			return constraints.ScrollOffset - r.effectiveShrinkExtent() + r.distanceToLeadingEdge()
		}
		return constraints.ScrollOffset
	}
	visibleTotal := r.distanceToLeadingEdge()
	for i := first + 1; i < index; i++ {
		visibleTotal += r.buildItemExtent(i)
	}
	return constraints.ScrollOffset + visibleTotal
}

// OverrideMinChildIndexForScrollOffset is the index of the first visible item.
//
// Flutter: material/carousel.dart _RenderSliverWeightedCarousel.getMinChildIndexForScrollOffset
func (r *renderSliverWeightedCarousel) OverrideMinChildIndexForScrollOffset(float64) int {
	return max(r.firstVisibleItemIndex(), 0)
}

// OverrideMaxChildIndexForScrollOffset is the index of the last visible item.
//
// Flutter: material/carousel.dart _RenderSliverWeightedCarousel.getMaxChildIndexForScrollOffset
func (r *renderSliverWeightedCarousel) OverrideMaxChildIndexForScrollOffset(float64) int {
	constraints := r.SliverConstraints()
	childCount := r.ChildManager().EstimatedChildCount()
	if count, ok := childCount.Get(); ok {
		visibleTotal := r.distanceToLeadingEdge()
		for i := r.firstVisibleItemIndex() + 1; i < count; i++ {
			visibleTotal += r.buildItemExtent(i)
			if visibleTotal >= constraints.ViewportMainAxisExtent {
				return i
			}
		}
		return count
	}
	if r.infinite {
		visibleTotal := r.distanceToLeadingEdge()
		index := r.firstVisibleItemIndex() + 1
		safeMin := math.Max(r.minChildExtent(), 1.0)
		upperBound := r.firstVisibleItemIndex() + int(math.Ceil(constraints.ViewportMainAxisExtent/safeMin))
		for visibleTotal < constraints.ViewportMainAxisExtent && index < upperBound {
			visibleTotal += r.buildItemExtent(index)
			if visibleTotal >= constraints.ViewportMainAxisExtent {
				return index
			}
			index++
		}
		return index
	}
	return 0
}

// OverrideComputeMaxScrollOffset is the scroll extent: infinite for an infinite
// carousel.
//
// Flutter: material/carousel.dart _RenderSliverWeightedCarousel.computeMaxScrollOffset
func (r *renderSliverWeightedCarousel) OverrideComputeMaxScrollOffset(rendering.SliverConstraints) float64 {
	if r.infinite {
		return math.Inf(1)
	}
	return float64(r.ChildManager().ChildCount()) * r.maxChildExtent()
}

func (r *renderSliverWeightedCarousel) childConstraints(index int) rendering.BoxConstraints {
	extent := r.buildItemExtent(index)
	return r.SliverConstraints().AsBoxConstraints(rendering.AsBoxConstraintsOpts{MinExtent: extent, MaxExtent: foundation.Some(extent)})
}

// PerformLayout lays out the items that are in the viewport, with the extents
// that follow from the weights.
//
// Flutter: material/carousel.dart _RenderSliverWeightedCarousel.performLayout
func (r *renderSliverWeightedCarousel) PerformLayout() {
	constraints := r.SliverConstraints()
	cm := r.ChildManager()
	cm.DidStartLayout()
	cm.SetDidUnderflow(false)

	scrollOffset := constraints.ScrollOffset + constraints.CacheOrigin
	targetEndScrollOffset := scrollOffset + constraints.RemainingCacheExtent

	firstIndex := r.OverrideMinChildIndexForScrollOffset(scrollOffset)
	targetLastIndex := foundation.None[int]()
	if !math.IsInf(targetEndScrollOffset, 0) {
		targetLastIndex = foundation.Some(r.OverrideMaxChildIndexForScrollOffset(targetEndScrollOffset))
	}

	if r.FirstChild() != nil {
		leading := r.CalculateLeadingGarbage(firstIndex)
		trailing := 0
		if last, ok := targetLastIndex.Get(); ok {
			trailing = r.CalculateTrailingGarbage(last)
		}
		r.CollectGarbage(leading, trailing)
	} else {
		r.CollectGarbage(0, 0)
	}

	if r.FirstChild() == nil {
		layoutOffset := r.OverrideIndexToLayoutOffset(firstIndex)
		if !r.AddInitialChild(rendering.AddInitialChildOpts{Index: firstIndex, LayoutOffset: layoutOffset}) {
			maxOffset := 0.0
			if firstIndex > 0 {
				maxOffset = r.OverrideComputeMaxScrollOffset(constraints)
			}
			r.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: maxOffset, MaxPaintExtent: maxOffset}))
			cm.DidFinishLayout()
			return
		}
	}

	var trailingChildWithLayout rendering.Box
	for index := r.IndexOf(r.FirstChild()) - 1; index >= firstIndex; index-- {
		child := r.InsertAndLayoutLeadingChild(r.childConstraints(index))
		if child == nil {
			r.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(r.OverrideIndexToLayoutOffset(index))}))
			return
		}
		rendering.SliverMultiBoxAdaptorParentDataOf(child).LayoutOffset = foundation.Some(r.OverrideIndexToLayoutOffset(index))
		if trailingChildWithLayout == nil {
			trailingChildWithLayout = child
		}
	}
	if trailingChildWithLayout == nil {
		r.FirstChild().Layout(r.childConstraints(r.IndexOf(r.FirstChild())))
		rendering.SliverMultiBoxAdaptorParentDataOf(r.FirstChild()).LayoutOffset = foundation.Some(r.OverrideIndexToLayoutOffset(firstIndex))
		trailingChildWithLayout = r.FirstChild()
	}

	extraLayoutOffset := 0.0
	if r.consumeMaxWeight {
		for i := len(r.weights) - 1; i >= 0; i-- {
			if r.weights[i] == r.maxWeight() {
				break
			}
			extraLayoutOffset += float64(r.weights[i]) * r.extentUnit()
		}
	}

	estimatedMaxScrollOffset := math.Inf(1)
	for index := r.IndexOf(trailingChildWithLayout) + 1; !targetLastIndex.IsSome() || index <= targetLastIndex.Value(); index++ {
		child := r.ChildAfter(trailingChildWithLayout)
		if child == nil || r.IndexOf(child) != index {
			child = r.InsertAndLayoutChild(r.childConstraints(index), trailingChildWithLayout)
			if child == nil {
				estimatedMaxScrollOffset = r.OverrideIndexToLayoutOffset(index) + extraLayoutOffset
				break
			}
		} else {
			child.Layout(r.childConstraints(index))
		}
		trailingChildWithLayout = child
		data := rendering.SliverMultiBoxAdaptorParentDataOf(child)
		data.LayoutOffset = foundation.Some(r.OverrideIndexToLayoutOffset(data.Index.Value()))
	}

	lastIndex := r.IndexOf(r.LastChild())
	leadingScrollOffset := r.OverrideIndexToLayoutOffset(firstIndex)
	var trailingScrollOffset float64
	if !r.infinite && lastIndex+1 == cm.ChildCount() {
		trailingScrollOffset = r.OverrideIndexToLayoutOffset(lastIndex)
		trailingScrollOffset += math.Max(float64(r.weights[len(r.weights)-1])*r.extentUnit(), r.buildItemExtent(lastIndex))
		trailingScrollOffset += extraLayoutOffset
	} else {
		trailingScrollOffset = r.OverrideIndexToLayoutOffset(lastIndex + 1)
	}

	estimatedMaxScrollOffset = math.Min(estimatedMaxScrollOffset, r.EstimateMaxScrollOffset(constraints, rendering.EstimateMaxScrollOffsetOpts{
		FirstIndex: foundation.Some(firstIndex), LastIndex: foundation.Some(lastIndex),
		LeadingScrollOffset: foundation.Some(leadingScrollOffset), TrailingScrollOffset: foundation.Some(trailingScrollOffset)}))

	from := leadingScrollOffset
	if r.consumeMaxWeight {
		from = 0
	}
	paintExtent := r.CalculatePaintOffset(constraints, from, trailingScrollOffset)
	cacheExtent := r.CalculateCacheOffset(constraints, from, trailingScrollOffset)

	targetEndForPaint := constraints.ScrollOffset + constraints.RemainingPaintExtent
	hasVisualOverflow := constraints.ScrollOffset > 0.0
	if !math.IsInf(targetEndForPaint, 0) && lastIndex >= r.OverrideMaxChildIndexForScrollOffset(targetEndForPaint) {
		hasVisualOverflow = true
	}
	r.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{
		ScrollExtent: estimatedMaxScrollOffset, PaintExtent: paintExtent, CacheExtent: foundation.Some(cacheExtent),
		MaxPaintExtent: estimatedMaxScrollOffset, HasVisualOverflow: hasVisualOverflow}))
	if estimatedMaxScrollOffset == trailingScrollOffset {
		cm.SetDidUnderflow(true)
	}
	cm.DidFinishLayout()
}

// CarouselScrollPhysics are the physics that make a carousel settle on a whole
// item.
//
// Flutter: material/carousel.dart CarouselScrollPhysics
type CarouselScrollPhysics struct {
	widgets.ScrollPhysicsBase
}

// NewCarouselScrollPhysics creates the physics with the parent (nil for none).
//
// Flutter: material/carousel.dart CarouselScrollPhysics.new
func NewCarouselScrollPhysics(parent widgets.ScrollPhysics) *CarouselScrollPhysics {
	p := &CarouselScrollPhysics{}
	widgets.InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: material/carousel.dart CarouselScrollPhysics.applyTo
func (p *CarouselScrollPhysics) ApplyTo(ancestor widgets.ScrollPhysics) widgets.ScrollPhysics {
	return NewCarouselScrollPhysics(p.BuildParent(ancestor))
}

func (p *CarouselScrollPhysics) targetPixels(position *carouselPosition, tolerance physics.Tolerance, velocity float64) float64 {
	var fraction float64
	if e, ok := position.itemExtent.Get(); ok {
		fraction = e / position.ViewportDimension()
	} else {
		fraction = float64(position.flexWeights[0]) / float64(sumInts(position.flexWeights))
	}
	itemWidth := position.ViewportDimension() * fraction
	actual := math.Max(0.0, position.Pixels()) / itemWidth
	round := math.Round(actual)
	item := actual
	if math.Abs(actual-round) < foundation.PrecisionErrorTolerance {
		item = round
	}
	if velocity < -tolerance.Velocity {
		item -= 0.5
	} else if velocity > tolerance.Velocity {
		item += 0.5
	}
	return math.Round(item) * itemWidth
}

// CreateBallisticSimulation is a spring to the item that the fling ends on.
//
// Flutter: material/carousel.dart CarouselScrollPhysics.createBallisticSimulation
func (p *CarouselScrollPhysics) CreateBallisticSimulation(position widgets.ScrollMetrics, velocity float64) physics.Simulation {
	metrics, ok := position.(*carouselPosition)
	if !ok {
		if foundation.DebugMode {
			panic(errors.New("CarouselScrollPhysicsPosition", "CarouselScrollPhysics can only be used with Scrollables that uses the CarouselController"))
		}
		return p.ScrollPhysicsBase.CreateBallisticSimulation(position, velocity)
	}
	if (velocity <= 0.0 && metrics.Pixels() <= metrics.MinScrollExtent()) || (velocity >= 0.0 && metrics.Pixels() >= metrics.MaxScrollExtent()) {
		return p.ScrollPhysicsBase.CreateBallisticSimulation(metrics, velocity)
	}
	tolerance := p.ToleranceFor(metrics)
	target := p.targetPixels(metrics, tolerance, velocity)
	if target != metrics.Pixels() {
		return physics.NewScrollSpringSimulation(p.Spring(), metrics.Pixels(), target, velocity, physics.SpringSimulationOpts{Tolerance: foundation.Some(tolerance)})
	}
	return nil
}

// AllowImplicitScrolling is true.
//
// Flutter: material/carousel.dart CarouselScrollPhysics.allowImplicitScrolling
func (p *CarouselScrollPhysics) AllowImplicitScrolling() bool { return true }

func sumInts(v []int) int {
	t := 0
	for _, x := range v {
		t += x
	}
	return t
}

// carouselPosition is the scroll position of a carousel: it also works in items.
//
// Flutter: material/carousel.dart _CarouselPosition
type carouselPosition struct {
	widgets.ScrollPositionWithSingleContext
	initialItem         int
	itemToShowOnStartup float64
	itemCount           foundation.Opt[int]
	infinite            bool
	cachedItem          foundation.Opt[float64]
	consumeMaxWeight    bool
	itemExtent          foundation.Opt[float64]
	flexWeights         []int
}

func newCarouselPosition(physics widgets.ScrollPhysics, context widgets.ScrollContext, initialItem int, itemExtent foundation.Opt[float64], flexWeights []int,
	consumeMaxWeight, infinite bool, itemCount foundation.Opt[int], oldPosition widgets.ScrollPosition) *carouselPosition {
	if foundation.DebugMode && (flexWeights != nil) == itemExtent.IsSome() {
		panic(errors.New("CarouselPositionExtentOrWeights", "a carousel has either an item extent or flex weights"))
	}
	p := &carouselPosition{initialItem: initialItem, itemToShowOnStartup: float64(initialItem), consumeMaxWeight: consumeMaxWeight, infinite: infinite,
		itemCount: itemCount, itemExtent: itemExtent, flexWeights: flexWeights}
	widgets.InitScrollPositionWithSingleContext(&p.ScrollPositionWithSingleContext, p, physics, context, widgets.ScrollPositionWithSingleContextOpts{
		NoInitialPixels: true, OldPosition: oldPosition})
	return p
}

func (p *carouselPosition) setConsumeMaxWeight(value bool) {
	if p.consumeMaxWeight == value {
		return
	}
	if p.HasPixels() && p.flexWeights != nil {
		leading := p.updateLeadingItem(p.flexWeights, value)
		p.ForcePixels(p.pixelsFromItem(leading, p.flexWeights, p.itemExtent))
	}
	p.consumeMaxWeight = value
}

func (p *carouselPosition) setItemExtent(value foundation.Opt[float64]) {
	if p.itemExtent == value {
		return
	}
	if p.HasPixels() && p.itemExtent.IsSome() && p.ViewportDimension() != 0.0 {
		leading := p.itemFromPixels(p.Pixels(), p.ViewportDimension())
		p.ForcePixels(p.pixelsFromItem(leading, p.flexWeights, value))
	}
	p.itemExtent = value
}

func (p *carouselPosition) setFlexWeights(value []int) {
	if slices.Equal(p.flexWeights, value) {
		return
	}
	old := p.flexWeights
	if p.HasPixels() && old != nil {
		leading := p.updateLeadingItem(value, p.consumeMaxWeight)
		p.ForcePixels(p.pixelsFromItem(leading, value, p.itemExtent))
	}
	p.flexWeights = value
}

// leadingItem is the index of the item at the leading edge.
//
// Flutter: material/carousel.dart _CarouselPosition.leadingItem
func (p *carouselPosition) leadingItem() int {
	leading := int(p.itemFromPixels(p.Pixels(), p.ViewportDimension()))
	if p.consumeMaxWeight && p.flexWeights != nil {
		leading = max(leading-slices.Index(p.flexWeights, slices.Max(p.flexWeights)), 0)
	}
	if count, ok := p.itemCount.Get(); ok && p.infinite && count > 0 {
		leading = leading % count
	}
	return leading
}

func (p *carouselPosition) updateLeadingItem(newFlexWeights []int, newConsumeMaxWeight bool) float64 {
	var maxItem float64
	if p.HasPixels() && p.flexWeights != nil {
		leading := p.itemFromPixels(p.Pixels(), p.ViewportDimension())
		if p.consumeMaxWeight {
			maxItem = leading
		} else {
			maxItem = leading + float64(slices.Index(p.flexWeights, slices.Max(p.flexWeights)))
		}
	} else {
		if !newConsumeMaxWeight {
			return p.itemToShowOnStartup
		}
		maxItem = p.itemToShowOnStartup
	}
	if newFlexWeights != nil && !newConsumeMaxWeight {
		smaller := 0
		for _, weight := range newFlexWeights {
			if weight == slices.Max(newFlexWeights) {
				break
			}
			smaller++
		}
		return maxItem - float64(smaller)
	}
	return maxItem
}

func (p *carouselPosition) itemFromPixels(pixels, viewportDimension float64) float64 {
	var fraction float64
	if e, ok := p.itemExtent.Get(); ok {
		fraction = e / viewportDimension
	} else {
		fraction = float64(p.flexWeights[0]) / float64(sumInts(p.flexWeights))
	}
	actual := math.Max(0.0, pixels) / (viewportDimension * fraction)
	round := math.Round(actual)
	if math.Abs(actual-round) < foundation.PrecisionErrorTolerance {
		return round
	}
	return actual
}

func (p *carouselPosition) pixelsFromItem(item float64, flexWeights []int, itemExtent foundation.Opt[float64]) float64 {
	if p.ViewportDimension() == 0.0 {
		return 0.0
	}
	var fraction float64
	if e, ok := itemExtent.Get(); ok {
		fraction = e / p.ViewportDimension()
	} else {
		fraction = float64(flexWeights[0]) / float64(sumInts(flexWeights))
	}
	return item * p.ViewportDimension() * fraction
}

// ApplyViewportDimension keeps the item when the viewport changes its size.
//
// Flutter: material/carousel.dart _CarouselPosition.applyViewportDimension
func (p *carouselPosition) ApplyViewportDimension(viewportDimension float64) bool {
	oldDimensions := foundation.None[float64]()
	if p.HasViewportDimension() {
		oldDimensions = foundation.Some(p.ViewportDimension())
	}
	if oldDimensions == foundation.Some(viewportDimension) {
		return true
	}
	result := p.ScrollPositionWithSingleContext.ApplyViewportDimension(viewportDimension)
	oldPixels := foundation.None[float64]()
	if p.HasPixels() {
		oldPixels = foundation.Some(p.Pixels())
	}
	var item float64
	switch {
	case oldPixels.IsNone():
		item = p.updateLeadingItem(p.flexWeights, p.consumeMaxWeight)
	case oldDimensions == foundation.Some(0.0):
		item = p.cachedItem.Value()
	default:
		item = p.itemFromPixels(oldPixels.Value(), oldDimensions.OrElse(viewportDimension))
	}
	newPixels := p.pixelsFromItem(item, p.flexWeights, p.itemExtent)
	if viewportDimension == 0.0 {
		p.cachedItem = foundation.Some(item)
	} else {
		p.cachedItem = foundation.None[float64]()
	}
	if oldPixels != foundation.Some(newPixels) {
		p.CorrectPixels(newPixels)
		return false
	}
	return result
}

// Absorb takes over the cached item and the extent of the old position.
//
// Flutter: material/carousel.dart _CarouselPosition.absorb
func (p *carouselPosition) Absorb(other widgets.ScrollPosition) {
	p.ScrollPositionWithSingleContext.Absorb(other)
	if o, ok := other.(*carouselPosition); ok {
		p.cachedItem = o.cachedItem
		p.itemExtent = o.itemExtent
	}
}

func (p *carouselPosition) cycleLengthInPixels() float64 {
	count, ok := p.itemCount.Get()
	if !ok || count <= 0 || !p.HasViewportDimension() || p.ViewportDimension() == 0 {
		return 0.0
	}
	var fraction float64
	switch {
	case p.itemExtent.IsSome():
		fraction = p.itemExtent.Value() / p.ViewportDimension()
	case p.flexWeights != nil:
		fraction = float64(p.flexWeights[0]) / float64(sumInts(p.flexWeights))
	default:
		return 0.0
	}
	return float64(count) * p.ViewportDimension() * fraction
}

// ApplyContentDimensions makes an infinite carousel start after a whole cycle.
//
// Flutter: material/carousel.dart _CarouselPosition.applyContentDimensions
func (p *carouselPosition) ApplyContentDimensions(minScrollExtent, maxScrollExtent float64) bool {
	if p.infinite && p.HasPixels() {
		cycle := p.cycleLengthInPixels()
		if cycle > 0 && p.Pixels() < cycle {
			cyclesToAdd := math.Ceil((cycle - p.Pixels()) / cycle)
			p.CorrectPixels(p.Pixels() + cyclesToAdd*cycle)
			return false
		}
	}
	if p.infinite {
		minScrollExtent = 0.0
	}
	return p.ScrollPositionWithSingleContext.ApplyContentDimensions(minScrollExtent, maxScrollExtent)
}

// CarouselController controls the scroll of a CarouselView, in items.
//
// Flutter: material/carousel.dart CarouselController
type CarouselController struct {
	widgets.ScrollControllerBase
	initialItem   int
	carouselState *carouselViewState
}

// CarouselControllerOpts are the optional parameters of NewCarouselController.
//
// Flutter: material/carousel.dart CarouselController
type CarouselControllerOpts struct {
	// InitialItem is the item shown first; the default is 0.
	InitialItem int
}

// NewCarouselController creates a controller.
//
// Flutter: material/carousel.dart CarouselController.new
func NewCarouselController(opts ...CarouselControllerOpts) *CarouselController {
	c := &CarouselController{initialItem: firstOpts(opts).InitialItem}
	widgets.InitScrollControllerBase(&c.ScrollControllerBase, c, widgets.ScrollControllerOpts{})
	return c
}

// InitialItem is the item that is shown first.
//
// Flutter: material/carousel.dart CarouselController.initialItem
func (c *CarouselController) InitialItem() int { return c.initialItem }

func (c *CarouselController) attachState(s *carouselViewState) { c.carouselState = s }

func (c *CarouselController) detachState(s *carouselViewState) {
	if c.carouselState == s {
		c.carouselState = nil
	}
}

func (c *CarouselController) carouselPosition() (*carouselPosition, bool) {
	if !c.HasClients() {
		return nil, false
	}
	p, ok := c.Position().(*carouselPosition)
	return p, ok
}

// LeadingItem is the index of the item at the leading edge. In debug builds it
// panics with the tags CarouselControllerNotAttached and
// CarouselControllerMultipleAttached when the controller has no carousel or more
// than one.
//
// Flutter: material/carousel.dart CarouselController.leadingItem
func (c *CarouselController) LeadingItem() int {
	if foundation.DebugMode {
		if len(c.Positions()) == 0 {
			panic(errors.New("CarouselControllerNotAttached", "CarouselController.leadingItem cannot be accessed before a CarouselView is built with it."))
		}
		if len(c.Positions()) != 1 {
			panic(errors.New("CarouselControllerMultipleAttached", "CarouselController.leadingItem cannot be read when multiple CarouselViews are attached to the same controller."))
		}
	}
	return c.Position().(*carouselPosition).leadingItem()
}

// AnimateToItem scrolls to the item. The channel of the result is closed when the
// animation is done; the default duration is 300 ms and the curve is ease.
//
// Flutter: material/carousel.dart CarouselController.animateToItem
func (c *CarouselController) AnimateToItem(index int, opts ...AnimateToItemOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	duration := o.Duration.OrElse(300 * time.Millisecond)
	curve := o.Curve
	if curve == nil {
		curve = animation.Curves.Ease
	}
	if !c.HasClients() || c.carouselState == nil {
		return scheduler.NewTickerFutureComplete()
	}
	state := c.carouselState
	w := state.Widget()
	hasFlexWeights := len(w.flexWeights) > 0
	if w.itemBuilder != nil {
		if count, ok := w.o.ItemCount.Get(); ok {
			index = min(max(index, 0), max(count-1, 0))
		} else {
			index = 0
		}
	} else {
		index = min(max(index, 0), max(len(w.children)-1, 0))
	}
	var futures []*scheduler.TickerFuture
	for _, position := range c.Positions() {
		cp := position.(*carouselPosition)
		futures = append(futures, cp.AnimateTo(c.targetOffset(cp, index, hasFlexWeights), duration, curve))
	}
	if len(futures) == 1 {
		return futures[0]
	}
	return scheduler.WaitTickerFutures(futures...)
}

// AnimateToItemOpts are the optional parameters of AnimateToItem.
//
// Flutter: material/carousel.dart CarouselController.animateToItem
type AnimateToItemOpts struct {
	// Duration is how long the scroll takes; the default is 300 ms.
	Duration foundation.Opt[time.Duration]
	// Curve is the curve of the scroll; the default is ease.
	Curve animation.Curve
}

func (c *CarouselController) targetOffset(position *carouselPosition, index int, hasFlexWeights bool) float64 {
	state := c.carouselState
	w := state.Widget()
	if !hasFlexWeights {
		target := float64(index) * state.itemExtent.Value()
		if !w.o.Infinite {
			return target
		}
		return c.adjustForInfiniteCycle(position, target)
	}
	weights := w.flexWeights
	totalWeight := sumInts(weights)
	dimension := position.ViewportDimension()
	maxWeightIndex := slices.Index(weights, slices.Max(weights))
	leadingIndex := index - maxWeightIndex
	if w.consumeMaxWeight() {
		leadingIndex = index
	}
	if w.itemBuilder != nil {
		if count, ok := w.o.ItemCount.Get(); ok {
			leadingIndex = min(max(leadingIndex, 0), max(count-1, 0))
		} else {
			leadingIndex = 0
		}
	} else {
		leadingIndex = min(max(leadingIndex, 0), max(len(w.children)-1, 0))
	}
	target := dimension * (float64(weights[0]) / float64(totalWeight)) * float64(leadingIndex)
	if !w.o.Infinite {
		return target
	}
	return c.adjustForInfiniteCycle(position, target)
}

func (c *CarouselController) adjustForInfiniteCycle(position *carouselPosition, targetInFirstCycle float64) float64 {
	cycle := position.cycleLengthInPixels()
	if cycle <= 0 {
		return targetInFirstCycle
	}
	current := position.Pixels()
	cycleStart := math.Floor(current/cycle) * cycle
	sameCycleTarget := cycleStart + targetInFirstCycle
	if sameCycleTarget >= current {
		return sameCycleTarget
	}
	return sameCycleTarget + cycle
}

func (c *CarouselController) itemCount() foundation.Opt[int] {
	if c.carouselState == nil {
		return foundation.None[int]()
	}
	w := c.carouselState.Widget()
	if w.itemBuilder != nil {
		return w.o.ItemCount
	}
	return foundation.Some(len(w.children))
}

// CreateScrollPosition creates the position that counts in items.
//
// Flutter: material/carousel.dart CarouselController.createScrollPosition
func (c *CarouselController) CreateScrollPosition(physics widgets.ScrollPhysics, context widgets.ScrollContext, oldPosition widgets.ScrollPosition) widgets.ScrollPosition {
	if foundation.DebugMode && c.carouselState == nil {
		panic(errors.New("CarouselControllerNotAttached", "a CarouselController creates a scroll position only for the CarouselView it is attached to"))
	}
	s := c.carouselState
	w := s.Widget()
	return newCarouselPosition(physics, context, c.initialItem, s.itemExtent, w.flexWeights, w.consumeMaxWeight(), w.o.Infinite, c.itemCount(), oldPosition)
}

// Attach registers the position and gives it what the carousel knows.
//
// Flutter: material/carousel.dart CarouselController.attach
func (c *CarouselController) Attach(position widgets.ScrollPosition) {
	c.ScrollControllerBase.Attach(position)
	p := position.(*carouselPosition)
	w := c.carouselState.Widget()
	p.setFlexWeights(w.flexWeights)
	p.setItemExtent(c.carouselState.itemExtent)
	p.setConsumeMaxWeight(w.consumeMaxWeight())
	p.infinite = w.o.Infinite
	p.itemCount = c.itemCount()
}
