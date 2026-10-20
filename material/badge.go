// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// BadgeOpts are the optional parameters of NewBadge.
//
// Flutter: material/badge.dart Badge.new
type BadgeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the badge's fill color.
	BackgroundColor foundation.Opt[ui.Color]
	// TextColor is the color of the badge's label.
	TextColor foundation.Opt[ui.Color]
	// SmallSize is the size of a badge without a label; the default is 6.
	SmallSize foundation.Opt[float64]
	// LargeSize is the height of a badge with a label; the default is 16.
	LargeSize foundation.Opt[float64]
	// TextStyle is the style of the label; the default is labelSmall.
	TextStyle *painting.TextStyle
	// Padding is the padding around the label; the default is 4 on each side.
	Padding painting.EdgeInsetsGeometry
	// Alignment is where the badge is placed over the child; the default is the
	// top end.
	Alignment painting.AlignmentGeometry
	// Offset is the offset of the badge from its alignment; the default is (4, -4)
	// and (-4, -4) in a right to left text direction.
	Offset foundation.Opt[ui.Offset]
	// Label is the badge's content, usually a Text; with no label the badge is a
	// dot.
	Label widgets.Widget
	// NoLabelVisible hides the badge: Flutter's isLabelVisible is false.
	NoLabelVisible bool
}

// BadgeCountOpts are the optional parameters of NewBadgeCount.
//
// Flutter: material/badge.dart Badge.count
type BadgeCountOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the badge's fill color.
	BackgroundColor foundation.Opt[ui.Color]
	// TextColor is the color of the badge's label.
	TextColor foundation.Opt[ui.Color]
	// SmallSize is the size of a badge without a label.
	SmallSize foundation.Opt[float64]
	// LargeSize is the height of a badge with a label; the default is 16.
	LargeSize foundation.Opt[float64]
	// TextStyle is the style of the label; the default is labelSmall.
	TextStyle *painting.TextStyle
	// Padding is the padding around the label.
	Padding painting.EdgeInsetsGeometry
	// Alignment is where the badge is placed over the child.
	Alignment painting.AlignmentGeometry
	// Offset is the offset of the badge from its alignment.
	Offset foundation.Opt[ui.Offset]
	// MaxCount is the count above which the label is "<maxCount>+"; the default
	// is 999.
	MaxCount foundation.Opt[int]
	// NoLabelVisible hides the badge: Flutter's isLabelVisible is false.
	NoLabelVisible bool
}

// Badge is a Material Design "badge": a small label that is placed over a
// widget.
//
// Flutter: material/badge.dart Badge
type Badge struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     BadgeOpts
}

// NewBadge creates a badge over the child (nil for the badge alone).
//
// Flutter: material/badge.dart Badge.new
func NewBadge(child widgets.Widget, opts ...BadgeOpts) *Badge {
	o := firstOpts(opts)
	return &Badge{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// NewBadgeCount creates a badge that shows the count, over the child. The tags of
// its failures are BadgeCountNegative and BadgeMaxCountNotPositive.
//
// Flutter: material/badge.dart Badge.count
func NewBadgeCount(count int, child widgets.Widget, opts ...BadgeCountOpts) *Badge {
	o := firstOpts(opts)
	maxCount := o.MaxCount.OrElse(999)
	if foundation.DebugMode {
		if count < 0 {
			panic(errors.New("BadgeCountNegative", "count must be non-negative", "count", count))
		}
		if maxCount <= 0 {
			panic(errors.New("BadgeMaxCountNotPositive", "maxCount must be positive", "maxCount", maxCount))
		}
	}
	label := strconv.Itoa(count)
	if count > maxCount {
		label = strconv.Itoa(maxCount) + "+"
	}
	return NewBadge(child, BadgeOpts{
		Key: o.Key, BackgroundColor: o.BackgroundColor, TextColor: o.TextColor, SmallSize: o.SmallSize, LargeSize: o.LargeSize,
		TextStyle: o.TextStyle, Padding: o.Padding, Alignment: o.Alignment, Offset: o.Offset, NoLabelVisible: o.NoLabelVisible,
		Label: widgets.NewText(label),
	})
}

// badgeDefaults are the defaults of a badge: the values that its theme does not
// give.
//
// Flutter: material/badge.dart _BadgeDefaultsM3
func badgeDefaults(theme *ThemeData) BadgeThemeData {
	return BadgeThemeData{
		SmallSize: foundation.Some(6.0), LargeSize: foundation.Some(16.0), Padding: painting.NewEdgeInsetsSymmetric(0, 4),
		Alignment: painting.AlignmentDirectionalTopEnd, BackgroundColor: foundation.Some(theme.ColorScheme.Error()),
		TextColor: foundation.Some(theme.ColorScheme.OnError()), TextStyle: theme.TextTheme.LabelSmall,
	}
}

// Build builds the badge.
//
// Flutter: material/badge.dart Badge.build
func (w *Badge) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	if o.NoLabelVisible {
		if w.child != nil {
			return w.child
		}
		return widgets.NewSizedBox(nil)
	}
	badgeTheme := BadgeThemeOf(bc)
	defaults := badgeDefaults(ThemeOf(bc))
	effectiveDecoration := painting.NewShapeDecoration(painting.StadiumBorder{}, painting.ShapeDecorationOpts{
		Color: firstOpt(o.BackgroundColor, badgeTheme.BackgroundColor, defaults.BackgroundColor),
	})
	var effectiveWidthOffset float64
	var badge widgets.Widget
	hasLabel := o.Label != nil
	if hasLabel {
		minSize := firstOpt(o.LargeSize, badgeTheme.LargeSize, defaults.LargeSize).Value()
		effectiveWidthOffset = minSize
		textStyle := firstNonNil(o.TextStyle, badgeTheme.TextStyle, defaults.TextStyle).
			CopyWith(painting.TextStyleCopy{Color: firstOpt(o.TextColor, badgeTheme.TextColor, defaults.TextColor)})
		badge = widgets.NewDefaultTextStyle(textStyle, &intrinsicHorizontalStadium{
			SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(widgets.NewContainer(o.Label, widgets.ContainerOpts{
				ClipBehavior: ui.ClipAntiAlias, Decoration: effectiveDecoration,
				Padding: firstNonNil(o.Padding, badgeTheme.Padding, defaults.Padding), Alignment: painting.AlignmentCenter,
			})),
			minSize: minSize,
		})
	} else {
		size := firstOpt(o.SmallSize, badgeTheme.SmallSize, defaults.SmallSize).Value()
		effectiveWidthOffset = size
		badge = widgets.NewContainer(nil, widgets.ContainerOpts{
			Width: foundation.Some(size), Height: foundation.Some(size), ClipBehavior: ui.ClipAntiAlias, Decoration: effectiveDecoration,
		})
	}
	if w.child == nil {
		return badge
	}
	effectiveAlignment := firstNonNil(o.Alignment, badgeTheme.Alignment, defaults.Alignment)
	textDirection := widgets.DirectionalityOf(bc)
	defaultOffset := ui.Offset{Dx: 4, Dy: -4}
	if textDirection != ui.TextDirectionLTR {
		defaultOffset = ui.Offset{Dx: -4, Dy: -4}
	}
	effectiveOffset := firstOpt(o.Offset, badgeTheme.Offset, foundation.Some(defaultOffset)).Value().Add(ui.Offset{Dy: 8})
	offset := ui.Offset{}
	if hasLabel {
		offset = effectiveOffset
	}
	return widgets.NewStack([]widgets.Widget{
		w.child,
		widgets.NewPositionedFill(&badgeBox{
			SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(badge),
			alignment:                     effectiveAlignment, offset: offset, hasLabel: hasLabel, widthOffset: effectiveWidthOffset, textDirection: textDirection,
		}),
	}, widgets.StackOpts{ClipBehavior: ui.ClipNone})
}

// badgeBox places the badge over its box.
//
// Flutter: material/badge.dart _Badge
type badgeBox struct {
	widgets.SingleChildRenderObjectWidget
	alignment     painting.AlignmentGeometry
	offset        ui.Offset
	widthOffset   float64
	textDirection ui.TextDirection
	hasLabel      bool
}

func (w *badgeBox) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderBadge{offset: w.offset, hasLabel: w.hasLabel, widthOffset: w.widthOffset}
	rendering.InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, w.alignment, maybeDirection(bc), nil)
	return r
}

func (w *badgeBox) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderBadge)
	r.SetAlignment(w.alignment)
	r.setOffset(w.offset)
	r.setWidthOffset(w.widthOffset)
	r.setHasLabel(w.hasLabel)
	r.SetTextDirection(maybeDirection(bc))
}

// renderBadge is the render object of badgeBox.
//
// Flutter: material/badge.dart _RenderBadge
type renderBadge struct {
	rendering.RenderAligningShiftedBox
	offset      ui.Offset
	hasLabel    bool
	widthOffset float64
}

func (r *renderBadge) setOffset(v ui.Offset) {
	if r.offset != v {
		r.offset = v
		r.MarkNeedsLayout()
	}
}

func (r *renderBadge) setHasLabel(v bool) {
	if r.hasLabel != v {
		r.hasLabel = v
		r.MarkNeedsLayout()
	}
}

func (r *renderBadge) setWidthOffset(v float64) {
	if r.widthOffset != v {
		r.widthOffset = v
		r.MarkNeedsLayout()
	}
}

func (r *renderBadge) badgeLocation(size, childSize ui.Size) ui.Offset {
	loc := r.offset.Add(r.ResolvedAlignment().AlongOffset(ui.Offset{Dx: size.Width - r.widthOffset, Dy: size.Height}))
	if r.hasLabel {
		loc = loc.Sub(ui.Offset{Dy: childSize.Height / 2})
	}
	return loc
}

// PerformLayout lays out the badge and puts it at its place.
//
// Flutter: material/badge.dart _RenderBadge.performLayout
func (r *renderBadge) PerformLayout() {
	constraints := r.BoxConstraints()
	r.SetSize(constraints.Biggest())
	child := r.Child()
	child.Layout(rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}, rendering.LayoutOpts{ParentUsesSize: true})
	boxData(child).Offset = r.badgeLocation(r.Size(), child.Size())
}

// ComputeDryLayout is the biggest size.
//
// Flutter: material/badge.dart _RenderBadge.computeDryLayout
func (r *renderBadge) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// ComputeDryBaseline is the baseline of the badge.
//
// Flutter: material/badge.dart _RenderBadge.computeDryBaseline
func (r *renderBadge) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	childConstraints := rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}
	b, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	loc := r.badgeLocation(r.GetDryLayout(constraints), child.GetDryLayout(childConstraints))
	return foundation.Some(b + loc.Dy)
}

// intrinsicHorizontalStadium sizes its child as a stadium: at least minSize
// tall, and at least as wide as it is tall.
//
// Flutter: material/badge.dart _IntrinsicHorizontalStadium
type intrinsicHorizontalStadium struct {
	widgets.SingleChildRenderObjectWidget
	minSize float64
}

func (w *intrinsicHorizontalStadium) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderIntrinsicHorizontalStadium{minSize: w.minSize}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (w *intrinsicHorizontalStadium) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderIntrinsicHorizontalStadium)
	if r.minSize != w.minSize {
		r.minSize = w.minSize
		r.MarkNeedsLayout()
	}
}

// renderIntrinsicHorizontalStadium is the render object of
// intrinsicHorizontalStadium.
//
// Flutter: material/badge.dart _RenderIntrinsicHorizontalStadium
type renderIntrinsicHorizontalStadium struct {
	rendering.RenderProxyBox
	minSize float64
}

func (r *renderIntrinsicHorizontalStadium) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.GetMaxIntrinsicWidth(height)
}

func (r *renderIntrinsicHorizontalStadium) ComputeMaxIntrinsicWidth(height float64) float64 {
	return math.Max(r.GetMaxIntrinsicHeight(math.Inf(1)), r.RenderProxyBox.ComputeMaxIntrinsicWidth(height))
}

func (r *renderIntrinsicHorizontalStadium) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.GetMaxIntrinsicHeight(width)
}

func (r *renderIntrinsicHorizontalStadium) ComputeMaxIntrinsicHeight(width float64) float64 {
	return math.Max(r.minSize, r.RenderProxyBox.ComputeMaxIntrinsicHeight(width))
}

func (r *renderIntrinsicHorizontalStadium) childConstraints(child rendering.Box, constraints rendering.BoxConstraints) rendering.BoxConstraints {
	childHeight := math.Max(r.minSize, child.GetMaxIntrinsicHeight(constraints.MaxWidth))
	childWidth := child.GetMaxIntrinsicWidth(constraints.MaxHeight)
	return constraints.Tighten(rendering.TightenOpts{Width: foundation.Some(math.Max(childWidth, childHeight)), Height: foundation.Some(childHeight)})
}

func (r *renderIntrinsicHorizontalStadium) computeSize(layoutChild rendering.ChildLayouter, constraints rendering.BoxConstraints) ui.Size {
	child := r.Child()
	childSize := layoutChild(child, r.childConstraints(child, constraints))
	if childSize.Height > childSize.Width {
		return ui.Size{Width: childSize.Height, Height: childSize.Height}
	}
	return childSize
}

func (r *renderIntrinsicHorizontalStadium) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeSize(rendering.ChildLayoutHelper.DryLayoutChild, constraints)
}

func (r *renderIntrinsicHorizontalStadium) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	return child.GetDryBaseline(r.childConstraints(child, constraints), baseline)
}

func (r *renderIntrinsicHorizontalStadium) PerformLayout() {
	r.SetSize(r.computeSize(rendering.ChildLayoutHelper.LayoutChild, r.BoxConstraints()))
}
