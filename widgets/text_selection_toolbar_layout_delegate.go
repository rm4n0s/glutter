// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TextSelectionToolbarLayoutDelegateOpts are the optional parameters of
// NewTextSelectionToolbarLayoutDelegate.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.new
type TextSelectionToolbarLayoutDelegateOpts struct {
	// FitsAbove is whether the child should be considered to fit above the
	// anchorAbove; when none, it is calculated.
	FitsAbove foundation.Opt[bool]
}

// TextSelectionToolbarLayoutDelegate is a SingleChildLayoutDelegate for use with
// CustomSingleChildLayout that positions its child above anchorAbove if it
// fits, or otherwise below anchorBelow.
//
// Primarily intended for use with toolbars or context menus.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate
type TextSelectionToolbarLayoutDelegate struct {
	rendering.SingleChildLayoutDelegateBase
	anchorAbove ui.Offset
	anchorBelow ui.Offset
	fitsAbove   foundation.Opt[bool]
}

// NewTextSelectionToolbarLayoutDelegate creates the delegate of a toolbar with
// the anchors given in the local coordinates of the box that the delegate lays
// out.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.new
func NewTextSelectionToolbarLayoutDelegate(anchorAbove, anchorBelow ui.Offset, opts ...TextSelectionToolbarLayoutDelegateOpts) *TextSelectionToolbarLayoutDelegate {
	o := firstOpts(opts)
	return &TextSelectionToolbarLayoutDelegate{
		SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil),
		anchorAbove:                   anchorAbove, anchorBelow: anchorBelow, fitsAbove: o.FitsAbove,
	}
}

// AnchorAbove is the focal point above which the toolbar attempts to position
// itself, in local coordinates.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.anchorAbove
func (d *TextSelectionToolbarLayoutDelegate) AnchorAbove() ui.Offset { return d.anchorAbove }

// AnchorBelow is the focal point below which the toolbar attempts to position
// itself, if it does not fit above the AnchorAbove, in local coordinates.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.anchorBelow
func (d *TextSelectionToolbarLayoutDelegate) AnchorBelow() ui.Offset { return d.anchorBelow }

// FitsAbove is whether the child should be considered to fit above the
// AnchorAbove: typically used to force the child to be drawn at the AnchorAbove
// even when it does not fit, such as when the Material TextSelectionToolbar
// draws an open overflow menu. When none, it is calculated.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.fitsAbove
func (d *TextSelectionToolbarLayoutDelegate) FitsAbove() foundation.Opt[bool] { return d.fitsAbove }

// TextSelectionToolbarLayoutDelegateCenterOn returns the distance from zero that
// centers width as closely as possible to position from zero while fitting
// between zero and max.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.centerOn
func TextSelectionToolbarLayoutDelegateCenterOn(position, width, max float64) float64 {
	// If it overflows on the left, put it as far left as possible.
	if position-width/2.0 < 0.0 {
		return 0.0
	}
	// If it overflows on the right, put it as far right as possible.
	if position+width/2.0 > max {
		return max - width
	}
	// Otherwise it fits while perfectly centered.
	return position - width/2.0
}

// GetConstraintsForChild loosens the constraints.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.getConstraintsForChild
func (d *TextSelectionToolbarLayoutDelegate) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return constraints.Loosen()
}

// GetPositionForChild centers the child on the anchor, above it if it fits and
// below otherwise.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.getPositionForChild
func (d *TextSelectionToolbarLayoutDelegate) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	fitsAbove := d.fitsAbove.OrElse(d.anchorAbove.Dy >= childSize.Height)
	anchor := d.anchorBelow
	if fitsAbove {
		anchor = d.anchorAbove
	}
	dy := anchor.Dy
	if fitsAbove {
		dy = math.Max(0.0, anchor.Dy-childSize.Height)
	}
	return ui.Offset{Dx: TextSelectionToolbarLayoutDelegateCenterOn(anchor.Dx, childSize.Width, size.Width), Dy: dy}
}

// ShouldRelayout is whether an anchor or fitsAbove changed.
//
// Flutter: widgets/text_selection_toolbar_layout_delegate.dart TextSelectionToolbarLayoutDelegate.shouldRelayout
func (d *TextSelectionToolbarLayoutDelegate) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	o := old.(*TextSelectionToolbarLayoutDelegate)
	return d.anchorAbove != o.anchorAbove || d.anchorBelow != o.anchorBelow || d.fitsAbove != o.fitsAbove
}
