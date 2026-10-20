// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TextSelectionThemeData defines the default appearance of the cursor and the
// selection of the text fields and the selectable text below it.
//
// Flutter: material/text_selection_theme.dart TextSelectionThemeData
//
//glutter:copywith value
type TextSelectionThemeData struct {
	// CursorColor is the color of the cursor in the text field.
	CursorColor foundation.Opt[ui.Color]
	// SelectionColor is the background color of selected text.
	SelectionColor foundation.Opt[ui.Color]
	// SelectionHandleColor is the color of the selection handles on the text.
	SelectionHandleColor foundation.Opt[ui.Color]
}

// TextSelectionThemeDataOpts are the parameters of NewTextSelectionThemeData: the
// fields of the data.
//
// Flutter: material/text_selection_theme.dart TextSelectionThemeData.new
type TextSelectionThemeDataOpts = TextSelectionThemeData

// NewTextSelectionThemeData creates the data; every field is optional.
//
// Flutter: material/text_selection_theme.dart TextSelectionThemeData.new
func NewTextSelectionThemeData(opts ...TextSelectionThemeDataOpts) TextSelectionThemeData {
	return firstOpts(opts)
}

// TextSelectionThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/text_selection_theme.dart TextSelectionThemeData.lerp
func TextSelectionThemeDataLerp(a, b TextSelectionThemeData, t float64) TextSelectionThemeData {
	return TextSelectionThemeData{
		CursorColor:          lerpOptColor(a.CursorColor, b.CursorColor, t),
		SelectionColor:       lerpOptColor(a.SelectionColor, b.SelectionColor, t),
		SelectionHandleColor: lerpOptColor(a.SelectionHandleColor, b.SelectionHandleColor, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/text_selection_theme.dart TextSelectionThemeData.==
func (d TextSelectionThemeData) Equal(o TextSelectionThemeData) bool { return d == o }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (TextSelectionThemeData) ToStringShort() string { return "TextSelectionThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d TextSelectionThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/text_selection_theme.dart TextSelectionThemeData.debugFillProperties
func (d TextSelectionThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("cursorColor", optValue(d.CursorColor)))
	p.Add(objectProp("selectionColor", optValue(d.SelectionColor)))
	p.Add(objectProp("selectionHandleColor", optValue(d.SelectionHandleColor)))
}

// TextSelectionTheme is an inherited theme that overrides the
// TextSelectionThemeData of the theme for the text below it. It also gives the
// text below it the default selection style (DefaultSelectionStyle) of the data.
//
// Flutter: material/text_selection_theme.dart TextSelectionTheme
type TextSelectionTheme struct {
	widgets.InheritedTheme
	data TextSelectionThemeData
	// original is the child that was given: the child of the inherited theme is
	// that child below a DefaultSelectionStyle.
	original widgets.Widget
}

// NewTextSelectionTheme creates the theme.
//
// Flutter: material/text_selection_theme.dart TextSelectionTheme.new
func NewTextSelectionTheme(data TextSelectionThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *TextSelectionTheme {
	// Deviation: Flutter overrides the getter of the child to insert the
	// DefaultSelectionStyle without breaking its API; the child of the inherited
	// widget is the style here.
	style := widgets.NewDefaultSelectionStyle(child, widgets.DefaultSelectionStyleOpts{
		SelectionColor: data.SelectionColor, CursorColor: data.CursorColor})
	return &TextSelectionTheme{InheritedTheme: widgets.NewInheritedThemeBase(style, opts...), data: data, original: child}
}

// Data is the data of the theme.
//
// Flutter: material/text_selection_theme.dart TextSelectionTheme.data
func (w *TextSelectionTheme) Data() TextSelectionThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/text_selection_theme.dart TextSelectionTheme.updateShouldNotify
func (w *TextSelectionTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*TextSelectionTheme).data)
}

// Wrap makes a TextSelectionTheme with the same data around the child.
//
// Flutter: material/text_selection_theme.dart TextSelectionTheme.wrap
func (w *TextSelectionTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewTextSelectionTheme(w.data, child)
}

// TextSelectionThemeOf returns the data of the closest TextSelectionTheme above
// the context, or the textSelectionTheme of the closest Theme. The context
// depends on it.
//
// Flutter: material/text_selection_theme.dart TextSelectionTheme.of
func TextSelectionThemeOf(bc widgets.BuildContext) TextSelectionThemeData {
	if t, ok := widgets.DependOn[*TextSelectionTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).TextSelectionTheme
}
