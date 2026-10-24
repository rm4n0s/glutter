// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SpellCheckSuggestionsToolbarLayoutDelegate positions the Material spell check
// suggestions toolbar: it is centered horizontally on the anchor, and just below
// it, moved up as far as it takes to fit in the box when it does not fit below.
//
// Flutter: material/spell_check_suggestions_toolbar_layout_delegate.dart SpellCheckSuggestionsToolbarLayoutDelegate
type SpellCheckSuggestionsToolbarLayoutDelegate struct {
	rendering.SingleChildLayoutDelegateBase
	anchor ui.Offset
}

// NewSpellCheckSuggestionsToolbarLayoutDelegate creates the delegate of a toolbar
// below an anchor, given in the coordinates of the box that the delegate lays
// out.
//
// Flutter: material/spell_check_suggestions_toolbar_layout_delegate.dart SpellCheckSuggestionsToolbarLayoutDelegate.new
func NewSpellCheckSuggestionsToolbarLayoutDelegate(anchor ui.Offset) *SpellCheckSuggestionsToolbarLayoutDelegate {
	return &SpellCheckSuggestionsToolbarLayoutDelegate{SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil), anchor: anchor}
}

// Anchor is the point that the toolbar is placed below.
//
// Flutter: material/spell_check_suggestions_toolbar_layout_delegate.dart SpellCheckSuggestionsToolbarLayoutDelegate.anchor
func (d *SpellCheckSuggestionsToolbarLayoutDelegate) Anchor() ui.Offset { return d.anchor }

// GetConstraintsForChild loosens the constraints.
//
// Flutter: material/spell_check_suggestions_toolbar_layout_delegate.dart SpellCheckSuggestionsToolbarLayoutDelegate.getConstraintsForChild
func (d *SpellCheckSuggestionsToolbarLayoutDelegate) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return constraints.Loosen()
}

// GetPositionForChild centers the child on the anchor horizontally, and puts it
// right below the anchor, or just enough upwards to fit within the size if it
// otherwise does not fit below.
//
// Flutter: material/spell_check_suggestions_toolbar_layout_delegate.dart SpellCheckSuggestionsToolbarLayoutDelegate.getPositionForChild
func (d *SpellCheckSuggestionsToolbarLayoutDelegate) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	y := d.anchor.Dy
	if d.anchor.Dy+childSize.Height > size.Height {
		// Positions child (of childSize) just enough upwards to fit within size
		// if it otherwise does not fit below the anchor.
		y = size.Height - childSize.Height
	}
	return ui.Offset{
		Dx: widgets.TextSelectionToolbarLayoutDelegateCenterOn(d.anchor.Dx, childSize.Width, size.Width),
		Dy: y,
	}
}

// ShouldRelayout is whether the anchor changed.
//
// Flutter: material/spell_check_suggestions_toolbar_layout_delegate.dart SpellCheckSuggestionsToolbarLayoutDelegate.shouldRelayout
func (d *SpellCheckSuggestionsToolbarLayoutDelegate) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	return d.anchor != old.(*SpellCheckSuggestionsToolbarLayoutDelegate).anchor
}
