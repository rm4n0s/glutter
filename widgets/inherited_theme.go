// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import "github.com/rm4n0s/errors"

// InheritedTheme is an inherited widget that can be copied to another place in
// the tree, such as a dialog or a route that is built outside the theme's
// subtree. Embed it and implement Wrap.
//
// Flutter: widgets/inherited_theme.dart InheritedTheme
type InheritedTheme struct{ InheritedWidget }

// NewInheritedThemeBase creates the embedded base of an inherited theme.
//
// Flutter: widgets/inherited_theme.dart InheritedTheme.new
func NewInheritedThemeBase(child Widget, opts ...WidgetOpts) InheritedTheme {
	return InheritedTheme{NewInheritedWidgetBase(child, opts...)}
}

func (*InheritedTheme) isInheritedTheme() {}

// inheritedThemeWidget is any widget with the InheritedTheme base.
type inheritedThemeWidget interface {
	InheritedWidgetInterface
	// Wrap returns a copy of this theme with the given widget as its child.
	//
	// Flutter: widgets/inherited_theme.dart InheritedTheme.wrap
	Wrap(bc BuildContext, child Widget) Widget
	isInheritedTheme()
}

// CapturedThemes are inherited themes that were found above one context, to be
// applied to a widget in another place.
//
// Flutter: widgets/inherited_theme.dart CapturedThemes
type CapturedThemes struct{ themes []inheritedThemeWidget }

// Wrap makes a widget that wraps the child in copies of the themes.
//
// Flutter: widgets/inherited_theme.dart CapturedThemes.wrap
func (c *CapturedThemes) Wrap(child Widget) Widget {
	return &captureAll{StatelessWidget: NewStatelessWidgetBase(), themes: c.themes, child: child}
}

// CaptureThemes finds the inherited themes between two contexts: the closest
// theme of each type above from, up to but not including to. A nil to goes up
// to the root. In debug builds it panics with the tag CaptureToNotAncestor when
// to is not an ancestor of from.
//
// Flutter: widgets/inherited_theme.dart InheritedTheme.capture
func CaptureThemes(from, to BuildContext) *CapturedThemes {
	if from == to {
		return &CapturedThemes{}
	}
	var themes []inheritedThemeWidget
	seen := map[string]struct{}{}
	didFindAncestor := to == nil
	from.VisitAncestorElements(func(ancestor Element) bool {
		if BuildContext(ancestor) == to {
			didFindAncestor = true
			return false
		}
		if _, ok := ancestor.(InheritedElementSelf); ok {
			if theme, ok := ancestor.AsElement().widget.(inheritedThemeWidget); ok {
				// Only the first theme of any type is kept: inherited themes
				// completely shadow the ancestors of the same type.
				name := typeKey(theme)
				if _, dup := seen[name]; !dup {
					seen[name] = struct{}{}
					themes = append(themes, theme)
				}
			}
		}
		return true
	})
	if !didFindAncestor {
		panic(errors.New("CaptureToNotAncestor", "the provided `to` context must be an ancestor of the `from` context"))
	}
	return &CapturedThemes{themes}
}

// CaptureAllThemes wraps the child in copies of all the themes between the
// contexts.
//
// Flutter: widgets/inherited_theme.dart InheritedTheme.captureAll
func CaptureAllThemes(bc BuildContext, child Widget, to BuildContext) Widget {
	return CaptureThemes(bc, to).Wrap(child)
}

type captureAll struct {
	StatelessWidget
	themes []inheritedThemeWidget
	child  Widget
}

func (w *captureAll) Build(bc BuildContext) Widget {
	wrapped := w.child
	for _, theme := range w.themes {
		wrapped = theme.Wrap(bc, wrapped)
	}
	return wrapped
}
