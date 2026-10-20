// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TextSelectionToolbarAnchors is the position information for a text selection
// toolbar.
//
// Typically, a menu will attempt to position itself at PrimaryAnchor, and if
// that's not possible, then it will use SecondaryAnchor instead, if it exists.
//
// Flutter: widgets/text_selection_toolbar_anchors.dart TextSelectionToolbarAnchors
type TextSelectionToolbarAnchors struct {
	// PrimaryAnchor is the location that the toolbar should attempt to position
	// itself at. If the toolbar doesn't fit at this location, use SecondaryAnchor
	// if it exists.
	//
	// Flutter: widgets/text_selection_toolbar_anchors.dart TextSelectionToolbarAnchors.primaryAnchor
	PrimaryAnchor ui.Offset
	// SecondaryAnchor is the fallback position that should be used if
	// PrimaryAnchor doesn't work.
	//
	// Flutter: widgets/text_selection_toolbar_anchors.dart TextSelectionToolbarAnchors.secondaryAnchor
	SecondaryAnchor foundation.Opt[ui.Offset]
}

// NewTextSelectionToolbarAnchorsFromSelection creates the anchors for some
// selection.
//
// Flutter: widgets/text_selection_toolbar_anchors.dart TextSelectionToolbarAnchors.fromSelection
func NewTextSelectionToolbarAnchorsFromSelection(renderBox rendering.Box, startGlyphHeight, endGlyphHeight float64,
	selectionEndpoints []rendering.TextSelectionPoint) TextSelectionToolbarAnchors {
	selectionRect := TextSelectionToolbarAnchorsSelectionRect(renderBox, startGlyphHeight, endGlyphHeight, selectionEndpoints)
	if selectionRect == ui.RectZero {
		return TextSelectionToolbarAnchors{}
	}
	editingRegion := editingRegionOf(renderBox)
	x := selectionRect.Left + selectionRect.Width()/2
	return TextSelectionToolbarAnchors{
		PrimaryAnchor:   ui.Offset{Dx: x, Dy: ui.ClampDouble(selectionRect.Top, editingRegion.Top, editingRegion.Bottom)},
		SecondaryAnchor: foundation.Some(ui.Offset{Dx: x, Dy: ui.ClampDouble(selectionRect.Bottom, editingRegion.Top, editingRegion.Bottom)}),
	}
}

// editingRegionOf is the Rect of the box in global coordinates.
//
// Flutter: widgets/text_selection_toolbar_anchors.dart TextSelectionToolbarAnchors._getEditingRegion
func editingRegionOf(renderBox rendering.Box) ui.Rect {
	return ui.NewRectFromPoints(renderBox.LocalToGlobal(ui.Offset{}, nil), renderBox.LocalToGlobal(renderBox.Size().BottomRight(ui.Offset{}), nil))
}

// TextSelectionToolbarAnchorsSelectionRect returns the Rect covering the given
// selection in the given box, in global coordinates.
//
// Flutter: widgets/text_selection_toolbar_anchors.dart TextSelectionToolbarAnchors.getSelectionRect
func TextSelectionToolbarAnchorsSelectionRect(renderBox rendering.Box, startGlyphHeight, endGlyphHeight float64,
	selectionEndpoints []rendering.TextSelectionPoint) ui.Rect {
	editingRegion := editingRegionOf(renderBox)
	if editingRegion.HasNaN() {
		return ui.RectZero
	}
	first, last := selectionEndpoints[0], selectionEndpoints[len(selectionEndpoints)-1]
	isMultiline := last.Point.Dy-first.Point.Dy > endGlyphHeight/2
	left, right := editingRegion.Left+first.Point.Dx, editingRegion.Left+last.Point.Dx
	if isMultiline {
		left, right = editingRegion.Left, editingRegion.Right
	}
	return ui.NewRectFromLTRB(left, editingRegion.Top+first.Point.Dy-startGlyphHeight, right, editingRegion.Top+last.Point.Dy)
}
