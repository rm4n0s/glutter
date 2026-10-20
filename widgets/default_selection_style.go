// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// DefaultSelectionStyle is the selection style to apply to descendant
// EditableText widgets which don't have an explicit style.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle
type DefaultSelectionStyle struct {
	InheritedTheme
	cursorColor    foundation.Opt[ui.Color]
	selectionColor foundation.Opt[ui.Color]
	mouseCursor    services.MouseCursor
}

// DefaultSelectionStyleOpts are the optional parameters of
// NewDefaultSelectionStyle.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.new
type DefaultSelectionStyleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CursorColor is the color of the text field's cursor.
	CursorColor foundation.Opt[ui.Color]
	// SelectionColor is the background color of selected text.
	SelectionColor foundation.Opt[ui.Color]
	// MouseCursor is the cursor for mouse pointers hovering over selectable Text
	// widgets; the default is the text cursor.
	MouseCursor services.MouseCursor
}

// NewDefaultSelectionStyle creates a default selection style widget that
// specifies the selection properties for all widgets below it in the widget tree.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.new
func NewDefaultSelectionStyle(child Widget, opts ...DefaultSelectionStyleOpts) *DefaultSelectionStyle {
	o := firstOpts(opts)
	return &DefaultSelectionStyle{NewInheritedThemeBase(child, WidgetOpts{Key: o.Key}), o.CursorColor, o.SelectionColor, o.MouseCursor}
}

// DefaultSelectionStyleFallback is the style that DefaultSelectionStyleOf
// returns when the context has no enclosing default selection style: it has
// fallback values (none), and cannot be incorporated into the tree.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.fallback
func DefaultSelectionStyleFallback() *DefaultSelectionStyle { return &DefaultSelectionStyle{} }

// NewDefaultSelectionStyleMerge creates a default selection style that
// overrides the selection styles in scope at this point in the widget tree. Any
// arguments that are not set replace nothing: the corresponding properties of
// the default selection style for the context where the widget is inserted are
// kept.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.merge
func NewDefaultSelectionStyleMerge(child Widget, opts ...DefaultSelectionStyleOpts) Widget {
	o := firstOpts(opts)
	return NewBuilder(func(bc BuildContext) Widget {
		parent := DefaultSelectionStyleOf(bc)
		merged := o
		if merged.CursorColor.IsNone() {
			merged.CursorColor = parent.cursorColor
		}
		if merged.SelectionColor.IsNone() {
			merged.SelectionColor = parent.selectionColor
		}
		if merged.MouseCursor == nil {
			merged.MouseCursor = parent.mouseCursor
		}
		return NewDefaultSelectionStyle(child, merged)
	})
}

// CursorColor is the color of the text field's cursor.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.cursorColor
func (w *DefaultSelectionStyle) CursorColor() foundation.Opt[ui.Color] { return w.cursorColor }

// SelectionColor is the background color of selected text.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.selectionColor
func (w *DefaultSelectionStyle) SelectionColor() foundation.Opt[ui.Color] { return w.selectionColor }

// MouseCursor is the cursor for mouse pointers hovering over selectable Text
// widgets, or nil for the text cursor.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.mouseCursor
func (w *DefaultSelectionStyle) MouseCursor() services.MouseCursor { return w.mouseCursor }

// DefaultSelectionStyleOf is the closest instance of this class that encloses
// the context. If no such instance exists, it returns the fallback.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.of
func DefaultSelectionStyleOf(bc BuildContext) *DefaultSelectionStyle {
	if s, ok := DependOn[*DefaultSelectionStyle](bc); ok {
		return s
	}
	return DefaultSelectionStyleFallback()
}

// Wrap returns a copy of the style with the given child.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.wrap
func (w *DefaultSelectionStyle) Wrap(_ BuildContext, child Widget) Widget {
	return &DefaultSelectionStyle{NewInheritedThemeBase(child), w.cursorColor, w.selectionColor, w.mouseCursor}
}

// UpdateShouldNotify is whether a property changed.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.updateShouldNotify
func (w *DefaultSelectionStyle) UpdateShouldNotify(old Widget) bool {
	o := old.(*DefaultSelectionStyle)
	return w.cursorColor != o.cursorColor || w.selectionColor != o.selectionColor || w.mouseCursor != o.mouseCursor
}
