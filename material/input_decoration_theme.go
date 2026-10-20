// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// InputDecorationThemeData defines the default values of the properties of the
// InputDecoration of the text fields below it: the decoration's unset fields
// take the ones of the theme. A color, text style or border side that depends on
// the state of the field is a widgets.WidgetStateProperty, nil is Dart's null.
//
// Flutter: material/input_decorator.dart InputDecorationThemeData
//
//glutter:copywith value
type InputDecorationThemeData struct {
	// LabelStyle is the style of the label when it is shown inline.
	LabelStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// FloatingLabelStyle is the style of the label when it floats.
	FloatingLabelStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HelperStyle is the style of the helper text.
	HelperStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HelperMaxLines is the maximum number of lines of the helper text.
	HelperMaxLines foundation.Opt[int]
	// HintStyle is the style of the hint text.
	HintStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HintFadeDuration is the duration of the fade of the hint.
	HintFadeDuration foundation.Opt[time.Duration]
	// HintMaxLines is the maximum number of lines of the hint text.
	HintMaxLines foundation.Opt[int]
	// ErrorStyle is the style of the error text.
	ErrorStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// ErrorMaxLines is the maximum number of lines of the error text.
	ErrorMaxLines foundation.Opt[int]
	// FloatingLabelBehavior defines the behavior of the floating label; the
	// default is FloatingLabelBehaviorAuto.
	FloatingLabelBehavior FloatingLabelBehavior
	// FloatingLabelAlignment is where the floating label is; the default is the
	// start.
	FloatingLabelAlignment foundation.Opt[FloatingLabelAlignment]
	// IsDense says whether the fields are dense.
	IsDense bool
	// ContentPadding is the padding of the content of the fields.
	ContentPadding painting.EdgeInsetsGeometry
	// IsCollapsed says whether the decoration is the same size as the field.
	IsCollapsed bool
	// IconColor is the color of the icon before the field.
	IconColor widgets.WidgetStateProperty[ui.Color]
	// PrefixStyle is the style of the prefix text.
	PrefixStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// PrefixIconColor is the color of the prefix icon.
	PrefixIconColor widgets.WidgetStateProperty[ui.Color]
	// PrefixIconConstraints are the constraints of the prefix icon.
	PrefixIconConstraints foundation.Opt[rendering.BoxConstraints]
	// SuffixStyle is the style of the suffix text.
	SuffixStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// SuffixIconColor is the color of the suffix icon.
	SuffixIconColor widgets.WidgetStateProperty[ui.Color]
	// SuffixIconConstraints are the constraints of the suffix icon.
	SuffixIconConstraints foundation.Opt[rendering.BoxConstraints]
	// CounterStyle is the style of the counter text.
	CounterStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// Filled says whether the container of the fields is filled.
	Filled bool
	// FillColor is the color of the container when it is filled.
	FillColor widgets.WidgetStateProperty[ui.Color]
	// ActiveIndicatorBorder is the side of the underline of a filled field, in
	// Material 3.
	ActiveIndicatorBorder widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// OutlineBorder is the side of the border of an outlined field, in Material 3.
	OutlineBorder widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// FocusColor is the color of the focus highlight of the container.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the hover highlight of the container.
	HoverColor foundation.Opt[ui.Color]
	// ErrorBorder is the border of a field in the error state.
	ErrorBorder InputBorder
	// FocusedBorder is the border of a field that has the focus.
	FocusedBorder InputBorder
	// FocusedErrorBorder is the border of a field that has the focus and the error
	// state.
	FocusedErrorBorder InputBorder
	// DisabledBorder is the border of a disabled field.
	DisabledBorder InputBorder
	// EnabledBorder is the border of an enabled field.
	EnabledBorder InputBorder
	// Border is the shape of the border of the fields.
	Border InputBorder
	// AlignLabelWithHint aligns the label with the hint of a multi-line field.
	AlignLabelWithHint bool
	// Constraints are the constraints of the whole decoration.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// VisualDensity is the density of the fields.
	VisualDensity foundation.Opt[VisualDensity]
}

// InputDecorationThemeDataOpts are the parameters of NewInputDecorationThemeData:
// the fields of the data.
//
// Flutter: material/input_decorator.dart InputDecorationThemeData.new
type InputDecorationThemeDataOpts = InputDecorationThemeData

// NewInputDecorationThemeData creates the data; every field is optional.
//
// Flutter: material/input_decorator.dart InputDecorationThemeData.new
func NewInputDecorationThemeData(opts ...InputDecorationThemeDataOpts) InputDecorationThemeData {
	d := firstOpts(opts)
	if d.FloatingLabelBehavior == FloatingLabelBehaviorUnspecified {
		d.FloatingLabelBehavior = FloatingLabelBehaviorAuto
	}
	if d.FloatingLabelAlignment.IsNone() {
		d.FloatingLabelAlignment = foundation.Some(FloatingLabelAlignmentStart)
	}
	return d
}

// Merge returns this data with its unset fields taken from the other one.
//
// Flutter: material/input_decorator.dart InputDecorationThemeData.merge
func (d InputDecorationThemeData) Merge(o InputDecorationThemeData) InputDecorationThemeData {
	pickProp := func(a, b widgets.WidgetStateProperty[*painting.TextStyle]) widgets.WidgetStateProperty[*painting.TextStyle] {
		if a != nil {
			return a
		}
		return b
	}
	pickColor := func(a, b widgets.WidgetStateProperty[ui.Color]) widgets.WidgetStateProperty[ui.Color] {
		if a != nil {
			return a
		}
		return b
	}
	pickSide := func(a, b widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]) widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]] {
		if a != nil {
			return a
		}
		return b
	}
	pickBorder := func(a, b InputBorder) InputBorder {
		if a != nil {
			return a
		}
		return b
	}
	d.LabelStyle = pickProp(d.LabelStyle, o.LabelStyle)
	d.FloatingLabelStyle = pickProp(d.FloatingLabelStyle, o.FloatingLabelStyle)
	d.HelperStyle = pickProp(d.HelperStyle, o.HelperStyle)
	d.HelperMaxLines = firstOpt(d.HelperMaxLines, o.HelperMaxLines)
	d.HintStyle = pickProp(d.HintStyle, o.HintStyle)
	d.HintFadeDuration = firstOpt(d.HintFadeDuration, o.HintFadeDuration)
	d.HintMaxLines = firstOpt(d.HintMaxLines, o.HintMaxLines)
	d.ErrorStyle = pickProp(d.ErrorStyle, o.ErrorStyle)
	d.ErrorMaxLines = firstOpt(d.ErrorMaxLines, o.ErrorMaxLines)
	if d.ContentPadding == nil {
		d.ContentPadding = o.ContentPadding
	}
	d.IconColor = pickColor(d.IconColor, o.IconColor)
	d.PrefixStyle = pickProp(d.PrefixStyle, o.PrefixStyle)
	d.PrefixIconColor = pickColor(d.PrefixIconColor, o.PrefixIconColor)
	d.PrefixIconConstraints = firstOpt(d.PrefixIconConstraints, o.PrefixIconConstraints)
	d.SuffixStyle = pickProp(d.SuffixStyle, o.SuffixStyle)
	d.SuffixIconColor = pickColor(d.SuffixIconColor, o.SuffixIconColor)
	d.SuffixIconConstraints = firstOpt(d.SuffixIconConstraints, o.SuffixIconConstraints)
	d.CounterStyle = pickProp(d.CounterStyle, o.CounterStyle)
	d.FillColor = pickColor(d.FillColor, o.FillColor)
	d.ActiveIndicatorBorder = pickSide(d.ActiveIndicatorBorder, o.ActiveIndicatorBorder)
	d.OutlineBorder = pickSide(d.OutlineBorder, o.OutlineBorder)
	d.FocusColor = firstOpt(d.FocusColor, o.FocusColor)
	d.HoverColor = firstOpt(d.HoverColor, o.HoverColor)
	d.ErrorBorder = pickBorder(d.ErrorBorder, o.ErrorBorder)
	d.FocusedBorder = pickBorder(d.FocusedBorder, o.FocusedBorder)
	d.FocusedErrorBorder = pickBorder(d.FocusedErrorBorder, o.FocusedErrorBorder)
	d.DisabledBorder = pickBorder(d.DisabledBorder, o.DisabledBorder)
	d.EnabledBorder = pickBorder(d.EnabledBorder, o.EnabledBorder)
	d.Border = pickBorder(d.Border, o.Border)
	d.Constraints = firstOpt(d.Constraints, o.Constraints)
	d.VisualDensity = firstOpt(d.VisualDensity, o.VisualDensity)
	return d
}

// InputDecorationThemeDataLerp is the theme that a lerp of ThemeData uses: the
// first theme for t below one half, the second one otherwise.
//
// Flutter: material/theme_data.dart ThemeData.lerp
func InputDecorationThemeDataLerp(a, b InputDecorationThemeData, t float64) InputDecorationThemeData {
	return lerpFirstOrSecond(a, b, t)
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/input_decorator.dart InputDecorationThemeData.==
func (d InputDecorationThemeData) Equal(o InputDecorationThemeData) bool {
	return structFieldsEqual(d, o)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (InputDecorationThemeData) ToStringShort() string { return "InputDecorationThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d InputDecorationThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/input_decorator.dart InputDecorationThemeData.debugFillProperties
func (d InputDecorationThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	add := func(name string, v any) { p.Add(objectProp(name, v)) }
	add("labelStyle", propOrNil(d.LabelStyle))
	add("floatingLabelStyle", propOrNil(d.FloatingLabelStyle))
	add("helperStyle", propOrNil(d.HelperStyle))
	add("hintStyle", propOrNil(d.HintStyle))
	add("errorStyle", propOrNil(d.ErrorStyle))
	if v, ok := d.HelperMaxLines.Get(); ok {
		add("helperMaxLines", v)
	}
	if v, ok := d.HintFadeDuration.Get(); ok {
		add("hintFadeDuration", v)
	}
	if v, ok := d.HintMaxLines.Get(); ok {
		add("hintMaxLines", v)
	}
	if v, ok := d.ErrorMaxLines.Get(); ok {
		add("errorMaxLines", v)
	}
	add("floatingLabelBehavior", nilIfZero(d.FloatingLabelBehavior))
	if v, ok := d.FloatingLabelAlignment.Get(); ok {
		add("floatingLabelAlignment", v)
	}
	if d.IsDense {
		add("isDense", true)
	}
	if d.ContentPadding != nil {
		add("contentPadding", d.ContentPadding)
	}
	if d.IsCollapsed {
		add("isCollapsed", true)
	}
	add("iconColor", propOrNil(d.IconColor))
	add("prefixIconColor", propOrNil(d.PrefixIconColor))
	add("suffixIconColor", propOrNil(d.SuffixIconColor))
	add("prefixStyle", propOrNil(d.PrefixStyle))
	add("suffixStyle", propOrNil(d.SuffixStyle))
	add("counterStyle", propOrNil(d.CounterStyle))
	if d.Filled {
		add("filled", true)
	}
	add("fillColor", propOrNil(d.FillColor))
	add("activeIndicatorBorder", propOrNil(d.ActiveIndicatorBorder))
	add("outlineBorder", propOrNil(d.OutlineBorder))
	if v, ok := d.FocusColor.Get(); ok {
		add("focusColor", v)
	}
	if v, ok := d.HoverColor.Get(); ok {
		add("hoverColor", v)
	}
	for _, b := range []struct {
		name   string
		border InputBorder
	}{{"errorBorder", d.ErrorBorder}, {"focusedBorder", d.FocusedBorder}, {"focusedErrorBorder", d.FocusedErrorBorder},
		{"disabledBorder", d.DisabledBorder}, {"enabledBorder", d.EnabledBorder}, {"border", d.Border}} {
		if b.border != nil {
			add(b.name, b.border)
		}
	}
	if d.AlignLabelWithHint {
		add("alignLabelWithHint", true)
	}
	if v, ok := d.Constraints.Get(); ok {
		add("constraints", v)
	}
	if v, ok := d.VisualDensity.Get(); ok {
		add("visualDensity", v)
	}
}

// InputDecorationTheme is an inherited theme that overrides the
// InputDecorationThemeData of the theme for the text fields below it.
//
// Flutter: material/input_decorator.dart InputDecorationTheme
type InputDecorationTheme struct {
	widgets.InheritedTheme
	data InputDecorationThemeData
}

// NewInputDecorationTheme creates the theme. Unlike Flutter's, it does not take
// the individual fields of the data, which are deprecated there.
//
// Flutter: material/input_decorator.dart InputDecorationTheme.new
func NewInputDecorationTheme(data InputDecorationThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *InputDecorationTheme {
	return &InputDecorationTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/input_decorator.dart InputDecorationTheme.data
func (w *InputDecorationTheme) Data() InputDecorationThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/input_decorator.dart InputDecorationTheme.updateShouldNotify
func (w *InputDecorationTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*InputDecorationTheme).data)
}

// Wrap makes an InputDecorationTheme with the same data around the child.
//
// Flutter: material/input_decorator.dart InputDecorationTheme.wrap
func (w *InputDecorationTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewInputDecorationTheme(w.data, child)
}

// InputDecorationThemeOf returns the data of the closest InputDecorationTheme
// above the context, or the inputDecorationTheme of the closest Theme. The
// context depends on it.
//
// Flutter: material/input_decorator.dart InputDecorationTheme.of
func InputDecorationThemeOf(bc widgets.BuildContext) InputDecorationThemeData {
	if t, ok := widgets.DependOn[*InputDecorationTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).InputDecorationTheme
}
