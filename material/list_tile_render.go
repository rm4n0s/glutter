// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// listTileSlot identifies the children of a renderListTile.
//
// Flutter: material/list_tile.dart _ListTileSlot
type listTileSlot int

const (
	listTileSlotLeading listTileSlot = iota
	listTileSlotTitle
	listTileSlotSubtitle
	listTileSlotTrailing
)

var allListTileSlots = []any{listTileSlotLeading, listTileSlotTitle, listTileSlotSubtitle, listTileSlotTrailing}

// listTileWidget lays out the leading widget, the title, the subtitle and the
// trailing widget of a ListTile.
//
// Flutter: material/list_tile.dart _ListTile
type listTileWidget struct {
	widgets.SlottedMultiChildRenderObjectWidget
	leading              widgets.Widget
	title                widgets.Widget
	subtitle             widgets.Widget
	trailing             widgets.Widget
	isThreeLine          bool
	isDense              bool
	visualDensity        VisualDensity
	textDirection        ui.TextDirection
	titleBaselineType    ui.TextBaseline
	subtitleBaselineType ui.TextBaseline
	horizontalTitleGap   float64
	minVerticalPadding   float64
	minLeadingWidth      float64
	minTileHeight        foundation.Opt[float64]
	titleAlignment       ListTileTitleAlignment
}

// Slots are the slots of the tile.
//
// Flutter: material/list_tile.dart _ListTile.slots
func (w *listTileWidget) Slots() []any { return allListTileSlots }

// ChildForSlot is the widget in the slot.
//
// Flutter: material/list_tile.dart _ListTile.childForSlot
func (w *listTileWidget) ChildForSlot(slot any) widgets.Widget {
	switch slot.(listTileSlot) {
	case listTileSlotLeading:
		return w.leading
	case listTileSlotTitle:
		return w.title
	case listTileSlotSubtitle:
		return w.subtitle
	}
	return w.trailing
}

// CreateRenderObject creates the render object.
//
// Flutter: material/list_tile.dart _ListTile.createRenderObject
func (w *listTileWidget) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderListTile{}
	r.update(w)
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitSlottedContainer(&r.SlottedContainer, r)
	return r
}

// UpdateRenderObject updates the render object.
//
// Flutter: material/list_tile.dart _ListTile.updateRenderObject
func (w *listTileWidget) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderListTile)
	old := *r
	r.update(w)
	if r.differsFrom(&old) {
		r.MarkNeedsLayout()
	}
}

// renderListTile is the render object of a list tile.
//
// Flutter: material/list_tile.dart _RenderListTile
type renderListTile struct {
	rendering.RenderBox
	rendering.SlottedContainer

	isThreeLine          bool
	isDense              bool
	visualDensity        VisualDensity
	textDirection        ui.TextDirection
	titleBaselineType    ui.TextBaseline
	subtitleBaselineType ui.TextBaseline
	horizontalTitleGap   float64
	minVerticalPadding   float64
	minLeadingWidth      float64
	minTileHeight        foundation.Opt[float64]
	titleAlignment       ListTileTitleAlignment
}

func (r *renderListTile) update(w *listTileWidget) {
	r.isThreeLine, r.isDense, r.visualDensity, r.textDirection = w.isThreeLine, w.isDense, w.visualDensity, w.textDirection
	r.titleBaselineType, r.subtitleBaselineType = w.titleBaselineType, w.subtitleBaselineType
	r.horizontalTitleGap, r.minVerticalPadding, r.minLeadingWidth = w.horizontalTitleGap, w.minVerticalPadding, w.minLeadingWidth
	r.minTileHeight, r.titleAlignment = w.minTileHeight, w.titleAlignment
}

func (r *renderListTile) differsFrom(o *renderListTile) bool {
	return r.isThreeLine != o.isThreeLine || r.isDense != o.isDense || r.visualDensity != o.visualDensity ||
		r.textDirection != o.textDirection || r.titleBaselineType != o.titleBaselineType ||
		r.subtitleBaselineType != o.subtitleBaselineType || r.horizontalTitleGap != o.horizontalTitleGap ||
		r.minVerticalPadding != o.minVerticalPadding || r.minLeadingWidth != o.minLeadingWidth ||
		r.minTileHeight != o.minTileHeight || r.titleAlignment != o.titleAlignment
}

func (r *renderListTile) slotBox(s listTileSlot) rendering.Box {
	c := r.SlotChild(s)
	if c == nil {
		return nil
	}
	return c.(rendering.Box)
}

func (r *renderListTile) leading() rendering.Box  { return r.slotBox(listTileSlotLeading) }
func (r *renderListTile) title() rendering.Box    { return r.slotBox(listTileSlotTitle) }
func (r *renderListTile) subtitle() rendering.Box { return r.slotBox(listTileSlotSubtitle) }
func (r *renderListTile) trailing() rendering.Box { return r.slotBox(listTileSlotTrailing) }

// children are the children that exist, in paint order.
//
// Flutter: material/list_tile.dart _RenderListTile.children
func (r *renderListTile) children() []rendering.Box {
	var out []rendering.Box
	for _, b := range []rendering.Box{r.leading(), r.title(), r.subtitle(), r.trailing()} {
		if b != nil {
			out = append(out, b)
		}
	}
	return out
}

func (r *renderListTile) effectiveHorizontalTitleGap() float64 {
	return r.horizontalTitleGap + r.visualDensity.Horizontal*2
}

func listTileMinWidth(box rendering.Box, height float64) float64 {
	if box == nil {
		return 0
	}
	return box.GetMinIntrinsicWidth(height)
}

func listTileMaxWidth(box rendering.Box, height float64) float64 {
	if box == nil {
		return 0
	}
	return box.GetMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicWidth is the least width that the tile paints right.
//
// Flutter: material/list_tile.dart _RenderListTile.computeMinIntrinsicWidth
func (r *renderListTile) ComputeMinIntrinsicWidth(height float64) float64 {
	leadingWidth := 0.0
	if l := r.leading(); l != nil {
		leadingWidth = math.Max(l.GetMinIntrinsicWidth(height), r.minLeadingWidth) + r.effectiveHorizontalTitleGap()
	}
	return leadingWidth + math.Max(listTileMinWidth(r.title(), height), listTileMinWidth(r.subtitle(), height)) +
		listTileMaxWidth(r.trailing(), height)
}

// ComputeMaxIntrinsicWidth is the width at which more width does not help.
//
// Flutter: material/list_tile.dart _RenderListTile.computeMaxIntrinsicWidth
func (r *renderListTile) ComputeMaxIntrinsicWidth(height float64) float64 {
	leadingWidth := 0.0
	if l := r.leading(); l != nil {
		leadingWidth = math.Max(l.GetMaxIntrinsicWidth(height), r.minLeadingWidth) + r.effectiveHorizontalTitleGap()
	}
	return leadingWidth + math.Max(listTileMaxWidth(r.title(), height), listTileMaxWidth(r.subtitle(), height)) +
		listTileMaxWidth(r.trailing(), height)
}

func (r *renderListTile) defaultTileHeight() float64 {
	base := r.visualDensity.BaseSizeAdjustment().Dy
	hasSubtitle := r.subtitle() != nil
	switch {
	case r.isThreeLine:
		if r.isDense {
			return base + 76
		}
		return base + 88
	case hasSubtitle:
		if r.isDense {
			return base + 64
		}
		return base + 72
	}
	if r.isDense {
		return base + 48
	}
	return base + 56
}

func (r *renderListTile) targetTileHeight() float64 {
	return r.minTileHeight.OrElse(r.defaultTileHeight())
}

// ComputeMinIntrinsicHeight is the least height that the tile paints right.
//
// Flutter: material/list_tile.dart _RenderListTile.computeMinIntrinsicHeight
func (r *renderListTile) ComputeMinIntrinsicHeight(width float64) float64 {
	titleMinHeight := r.title().GetMinIntrinsicHeight(width)
	subtitleMinHeight := 0.0
	if s := r.subtitle(); s != nil {
		subtitleMinHeight = s.GetMinIntrinsicHeight(width)
	}
	contentHeight := titleMinHeight + subtitleMinHeight + 2*r.minVerticalPadding
	return math.Max(r.targetTileHeight(), contentHeight)
}

// ComputeMaxIntrinsicHeight is the height at which more height does not help.
//
// Flutter: material/list_tile.dart _RenderListTile.computeMaxIntrinsicHeight
func (r *renderListTile) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.GetMinIntrinsicHeight(width)
}

// ComputeDistanceToActualBaseline is the baseline of the title.
//
// Flutter: material/list_tile.dart _RenderListTile.computeDistanceToActualBaseline
func (r *renderListTile) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	title := r.title()
	b, ok := title.GetDistanceToActualBaseline(baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(b + boxData(title).Offset.Dy)
}

func (r *renderListTile) maxIconHeightConstraint() rendering.BoxConstraints {
	base := 56.0
	if r.isDense {
		base = 48
	}
	return rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxHeight: foundation.Some(base + r.visualDensity.BaseSizeAdjustment().Dy)})
}

// listTileSizes are the sizes that the layout of a list tile works out.
//
// Flutter: material/list_tile.dart _Sizes
type listTileSizes struct {
	titleY          float64
	textConstraints rendering.BoxConstraints
	tileSize        ui.Size
}

func (r *renderListTile) computeSizes(getBaseline rendering.ChildBaselineGetter, getSize rendering.ChildLayouter, constraints rendering.BoxConstraints, positionChild func(rendering.Box, ui.Offset)) listTileSizes {
	looseConstraints := constraints.Loosen()
	tileWidth := looseConstraints.MaxWidth
	iconConstraints := looseConstraints.Enforce(r.maxIconHeightConstraint())
	leading, trailing := r.leading(), r.trailing()
	var leadingSize, trailingSize *ui.Size
	if leading != nil {
		s := getSize(leading, iconConstraints)
		leadingSize = &s
	}
	if trailing != nil {
		s := getSize(trailing, iconConstraints)
		trailingSize = &s
	}
	if foundation.DebugMode && tileWidth != 0 {
		switch {
		case leadingSize != nil && tileWidth == leadingSize.Width:
			panic(errors.New("ListTileLeadingConsumesTileWidth", "the leading widget consumes the entire tile width, including the content padding of the ListTile: resize the tile so that the leading widget and the padding fit, use a sized widget, or replace the ListTile with a custom widget"))
		case trailingSize != nil && tileWidth == trailingSize.Width:
			panic(errors.New("ListTileTrailingConsumesTileWidth", "the trailing widget consumes the entire tile width, including the content padding of the ListTile: resize the tile so that the trailing widget and the padding fit, use a sized widget, or replace the ListTile with a custom widget"))
		}
	}
	titleStart := 0.0
	if leadingSize != nil {
		titleStart = math.Max(r.minLeadingWidth, leadingSize.Width) + r.effectiveHorizontalTitleGap()
	}
	adjustedTrailingWidth := 0.0
	if trailingSize != nil {
		adjustedTrailingWidth = math.Max(trailingSize.Width+r.effectiveHorizontalTitleGap(), 32)
	}
	textConstraints := looseConstraints.Tighten(rendering.TightenOpts{Width: foundation.Some(tileWidth - titleStart - adjustedTrailingWidth)})
	title, subtitle := r.title(), r.subtitle()
	titleHeight := getSize(title, textConstraints).Height
	isLTR := r.textDirection != ui.TextDirectionRTL

	var titleY, tileHeight float64
	if subtitle == nil {
		tileHeight = math.Max(r.targetTileHeight(), titleHeight+2*r.minVerticalPadding)
		titleY = (tileHeight - titleHeight) / 2
	} else {
		subtitleHeight := getSize(subtitle, textConstraints).Height
		titleBaseline := getBaseline(title, textConstraints, r.titleBaselineType).OrElse(titleHeight)
		subtitleBaseline := getBaseline(subtitle, textConstraints, r.subtitleBaselineType).OrElse(subtitleHeight)
		var targetTitleY, targetSubtitleY float64
		switch {
		case r.isThreeLine && r.isDense:
			targetTitleY, targetSubtitleY = 22, 42
		case r.isThreeLine:
			targetTitleY, targetSubtitleY = 28, 48
		case r.isDense:
			targetTitleY, targetSubtitleY = 28, 48
		default:
			targetTitleY, targetSubtitleY = 32, 52
		}
		targetTitleY -= titleBaseline
		targetSubtitleY += r.visualDensity.Vertical*2 - subtitleBaseline
		halfOverlap := math.Max(targetTitleY+titleHeight-targetSubtitleY, 0) / 2
		idealTitleY := targetTitleY - halfOverlap
		idealSubtitleY := targetSubtitleY + halfOverlap
		compact := idealTitleY < r.minVerticalPadding || idealSubtitleY+subtitleHeight+r.minVerticalPadding > r.targetTileHeight()
		if positionChild != nil {
			x := adjustedTrailingWidth
			if isLTR {
				x = titleStart
			}
			y := idealSubtitleY
			if compact {
				y = r.minVerticalPadding + titleHeight
			}
			positionChild(subtitle, ui.Offset{Dx: x, Dy: y})
		}
		if compact {
			tileHeight = 2*r.minVerticalPadding + titleHeight + subtitleHeight
			titleY = r.minVerticalPadding
		} else {
			tileHeight = r.targetTileHeight()
			titleY = idealTitleY
		}
	}
	if positionChild != nil {
		x := adjustedTrailingWidth
		if isLTR {
			x = titleStart
		}
		positionChild(title, ui.Offset{Dx: x, Dy: titleY})
		if leading != nil && leadingSize != nil {
			lx := tileWidth - leadingSize.Width
			if isLTR {
				lx = 0
			}
			positionChild(leading, ui.Offset{Dx: lx, Dy: r.titleAlignment.yOffsetFor(leadingSize.Height, tileHeight, r, true)})
		}
		if trailing != nil && trailingSize != nil {
			tx := 0.0
			if isLTR {
				tx = tileWidth - trailingSize.Width
			}
			positionChild(trailing, ui.Offset{Dx: tx, Dy: r.titleAlignment.yOffsetFor(trailingSize.Height, tileHeight, r, false)})
		}
	}
	return listTileSizes{titleY: titleY, textConstraints: textConstraints, tileSize: ui.Size{Width: tileWidth, Height: tileHeight}}
}

// ComputeDryBaseline is the baseline of the title for the constraints.
//
// Flutter: material/list_tile.dart _RenderListTile.computeDryBaseline
func (r *renderListTile) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	sizes := r.computeSizes(rendering.ChildLayoutHelper.GetDryBaseline, rendering.ChildLayoutHelper.DryLayoutChild, constraints, nil)
	b, ok := r.title().GetDryBaseline(sizes.textConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(b + sizes.titleY)
}

// ComputeDryLayout is the size of the tile for the constraints.
//
// Flutter: material/list_tile.dart _RenderListTile.computeDryLayout
func (r *renderListTile) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return constraints.Constrain(r.computeSizes(rendering.ChildLayoutHelper.GetDryBaseline, rendering.ChildLayoutHelper.DryLayoutChild, constraints, nil).tileSize)
}

// PerformLayout lays out and positions the children.
//
// Flutter: material/list_tile.dart _RenderListTile.performLayout
func (r *renderListTile) PerformLayout() {
	constraints := r.BoxConstraints()
	tileSize := r.computeSizes(rendering.ChildLayoutHelper.GetBaseline, rendering.ChildLayoutHelper.LayoutChild, constraints,
		func(box rendering.Box, offset ui.Offset) { boxData(box).Offset = offset }).tileSize
	r.SetSize(constraints.Constrain(tileSize))
}

// Paint paints the children.
//
// Flutter: material/list_tile.dart _RenderListTile.paint
func (r *renderListTile) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	for _, child := range []rendering.Box{r.leading(), r.title(), r.subtitle(), r.trailing()} {
		if child != nil {
			context.PaintChild(child, boxData(child).Offset.Add(offset))
		}
	}
}

// HitTestSelf is true: the whole tile is tappable.
//
// Flutter: material/list_tile.dart _RenderListTile.hitTestSelf
func (r *renderListTile) HitTestSelf(ui.Offset) bool { return true }

// HitTestChildren hit tests the children.
//
// Flutter: material/list_tile.dart _RenderListTile.hitTestChildren
func (r *renderListTile) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for _, child := range r.children() {
		offset := boxData(child).Offset
		child := child
		if result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		}) {
			return true
		}
	}
	return false
}
