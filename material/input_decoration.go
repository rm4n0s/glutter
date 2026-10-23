// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// FloatingLabelBehavior defines the behavior of the floating label.
//
// Flutter: material/input_decorator.dart FloatingLabelBehavior
type FloatingLabelBehavior int

const (
	// FloatingLabelBehaviorUnspecified is the zero value: the behavior of the
	// theme, which is FloatingLabelBehaviorAuto by default.
	FloatingLabelBehaviorUnspecified FloatingLabelBehavior = iota
	// FloatingLabelBehaviorNever always shows the label in its inline position.
	//
	// Flutter: material/input_decorator.dart FloatingLabelBehavior.never
	FloatingLabelBehaviorNever
	// FloatingLabelBehaviorAuto floats the label when the field has the focus or
	// content.
	//
	// Flutter: material/input_decorator.dart FloatingLabelBehavior.auto
	FloatingLabelBehaviorAuto
	// FloatingLabelBehaviorAlways always floats the label.
	//
	// Flutter: material/input_decorator.dart FloatingLabelBehavior.always
	FloatingLabelBehaviorAlways
)

// Name is the Dart name of the behavior.
func (b FloatingLabelBehavior) Name() string { return enumName(int(b), "never", "auto", "always") }

// String is the Dart description of the behavior.
func (b FloatingLabelBehavior) String() string { return "FloatingLabelBehavior." + b.Name() }

// orAuto is the behavior, or Auto when it is unspecified.
func (b FloatingLabelBehavior) orAuto() FloatingLabelBehavior {
	if b == FloatingLabelBehaviorUnspecified {
		return FloatingLabelBehaviorAuto
	}
	return b
}

// FloatingLabelAlignment defines how a floating label is aligned in the
// container of its field: at the start, or centered.
//
// Flutter: material/input_decorator.dart FloatingLabelAlignment
type FloatingLabelAlignment struct {
	// x is -1 for the start, 0 for the center and 1 for the end.
	x float64
}

var (
	// FloatingLabelAlignmentStart aligns the floating label at the start of the
	// container.
	//
	// Flutter: material/input_decorator.dart FloatingLabelAlignment.start
	FloatingLabelAlignmentStart = FloatingLabelAlignment{-1.0}
	// FloatingLabelAlignmentCenter centers the floating label in the container.
	//
	// Flutter: material/input_decorator.dart FloatingLabelAlignment.center
	FloatingLabelAlignmentCenter = FloatingLabelAlignment{0.0}
)

// String is the name of the alignment.
func (a FloatingLabelAlignment) String() string {
	switch a.x {
	case -1.0:
		return "FloatingLabelAlignment.start"
	case 0.0:
		return "FloatingLabelAlignment.center"
	}
	return "FloatingLabelAlignment(x: " + strconv.FormatFloat(a.x, 'f', 1, 64) + ")"
}

// InputDecoration is the border, labels, icons, and styles used to decorate a
// text field. A field of the type that Flutter calls "Color" or "TextStyle" can
// depend on the state of the field: it is a widgets.WidgetStateProperty (use
// widgets.NewWidgetStatePropertyAll for one value), and nil is Dart's null. The
// text of Flutter's String? fields is a foundation.Opt[string].
//
// Flutter: material/input_decorator.dart InputDecoration
type InputDecoration struct {
	// Icon is an icon to show before the input field and outside of the
	// decoration's container.
	Icon widgets.Widget
	// IconColor is the color of the Icon.
	IconColor widgets.WidgetStateProperty[ui.Color]
	// Label is an optional widget to display as the label of the field. It cannot
	// be used with LabelText.
	Label widgets.Widget
	// LabelText is the text to show above the field when it is focused or has
	// content, and where the hint is when it is empty.
	LabelText foundation.Opt[string]
	// LabelStyle is the style of the label when it is shown inline.
	LabelStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// FloatingLabelStyle is the style of the label when it floats.
	FloatingLabelStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// Helper is a widget to show below the field, when there is no error.
	Helper widgets.Widget
	// HelperText is the text to show below the field, when there is no error.
	HelperText foundation.Opt[string]
	// HelperStyle is the style of the helper.
	HelperStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HelperMaxLines is the maximum number of lines of the helper text.
	HelperMaxLines foundation.Opt[int]
	// HintText is the text that suggests what sort of input the field accepts.
	HintText foundation.Opt[string]
	// Hint is a widget to show as the hint.
	Hint widgets.Widget
	// HintStyle is the style of the hint (and of the label when it is inline).
	HintStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HintTextDirection is the direction of the hint text.
	HintTextDirection foundation.Opt[ui.TextDirection]
	// HintMaxLines is the maximum number of lines of the hint text.
	HintMaxLines foundation.Opt[int]
	// HintFadeDuration is the duration of the fade of the hint when the label
	// moves.
	HintFadeDuration foundation.Opt[time.Duration]
	// NoMaintainHintSize makes the hint take no space when it is not shown
	// (Flutter's maintainHintSize: false).
	NoMaintainHintSize bool
	// MaintainLabelSize makes the label take its space in the layout whether it is
	// shown or not.
	MaintainLabelSize bool
	// Error is a widget to show below the field instead of the helper. It cannot
	// be used with ErrorText.
	Error widgets.Widget
	// ErrorText is the text to show below the field; the field is then in the
	// error state, even when the text is empty.
	ErrorText foundation.Opt[string]
	// ErrorStyle is the style of the error.
	ErrorStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// ErrorMaxLines is the maximum number of lines of the error text.
	ErrorMaxLines foundation.Opt[int]
	// FloatingLabelBehavior defines the behavior of the floating label.
	FloatingLabelBehavior FloatingLabelBehavior
	// FloatingLabelAlignment is where the floating label is in its container.
	FloatingLabelAlignment foundation.Opt[FloatingLabelAlignment]
	// IsCollapsed says whether the decoration is the same size as the input field
	// (no label, no padding).
	IsCollapsed foundation.Opt[bool]
	// IsDense says whether the field is part of a dense form: it has a smaller
	// height and less padding.
	IsDense foundation.Opt[bool]
	// ContentPadding is the padding of the field's content; nil for the default.
	ContentPadding painting.EdgeInsetsGeometry
	// PrefixIcon is an icon that appears before the editable part of the field
	// and inside the container.
	PrefixIcon widgets.Widget
	// PrefixIconConstraints are the constraints of the prefix icon.
	PrefixIconConstraints foundation.Opt[rendering.BoxConstraints]
	// Prefix is a widget to place before the editable text.
	Prefix widgets.Widget
	// PrefixText is the text to place before the editable text.
	PrefixText foundation.Opt[string]
	// PrefixStyle is the style of the prefix text.
	PrefixStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// PrefixIconColor is the color of the prefix icon.
	PrefixIconColor widgets.WidgetStateProperty[ui.Color]
	// SuffixIcon is an icon that appears after the editable part of the field and
	// inside the container.
	SuffixIcon widgets.Widget
	// Suffix is a widget to place after the editable text.
	Suffix widgets.Widget
	// SuffixText is the text to place after the editable text.
	SuffixText foundation.Opt[string]
	// SuffixStyle is the style of the suffix text.
	SuffixStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// SuffixIconColor is the color of the suffix icon.
	SuffixIconColor widgets.WidgetStateProperty[ui.Color]
	// SuffixIconConstraints are the constraints of the suffix icon.
	SuffixIconConstraints foundation.Opt[rendering.BoxConstraints]
	// Counter is a widget to show below the field, at the end.
	Counter widgets.Widget
	// CounterText is the text of the counter.
	CounterText foundation.Opt[string]
	// CounterStyle is the style of the counter.
	CounterStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// Filled says whether the container is filled with FillColor.
	Filled foundation.Opt[bool]
	// FillColor is the color of the container when Filled is true.
	FillColor widgets.WidgetStateProperty[ui.Color]
	// FocusColor is the color of the focus highlight of the container.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the highlight of the container when the pointer
	// is over it.
	HoverColor foundation.Opt[ui.Color]
	// ErrorBorder is the border of the field when it has the error state and does
	// not have the focus.
	ErrorBorder InputBorder
	// FocusedBorder is the border of the field when it has the focus.
	FocusedBorder InputBorder
	// FocusedErrorBorder is the border of the field when it has the focus and the
	// error state.
	FocusedErrorBorder InputBorder
	// DisabledBorder is the border of the field when it is disabled.
	DisabledBorder InputBorder
	// EnabledBorder is the border of the field when it is enabled.
	EnabledBorder InputBorder
	// Border is the shape of the border of the field; the other borders default
	// to it. It can be a state property (WidgetStateInputBorderResolveWith).
	Border InputBorder
	// Disabled makes the field disabled (Flutter's enabled: false).
	Disabled bool
	// SemanticCounterText is the semantics label of the counter.
	SemanticCounterText foundation.Opt[string]
	// AlignLabelWithHint aligns the label with the hint of a multi-line field.
	AlignLabelWithHint foundation.Opt[bool]
	// Constraints are the constraints of the whole decoration.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// VisualDensity is the density of the field.
	VisualDensity foundation.Opt[VisualDensity]
}

// InputDecorationOpts are the parameters of NewInputDecoration: the fields of
// the decoration.
//
// Flutter: material/input_decorator.dart InputDecoration.new
type InputDecorationOpts = InputDecoration

// NewInputDecoration creates a decoration; every field is optional. In debug
// builds it panics with the tag InputDecorationBothGiven when a widget and its
// text are both given (label and labelText, hint and hintText, helper and
// helperText, prefix and prefixText, suffix and suffixText, error and
// errorText).
//
// Flutter: material/input_decorator.dart InputDecoration.new
func NewInputDecoration(opts ...InputDecorationOpts) InputDecoration {
	d := firstOpts(opts)
	if foundation.DebugMode {
		for _, pair := range []struct {
			name       string
			widget     bool
			textIsSome bool
		}{
			{"label and labelText", d.Label != nil, d.LabelText.IsSome()},
			{"hint and hintText", d.Hint != nil, d.HintText.IsSome()},
			{"helper and helperText", d.Helper != nil, d.HelperText.IsSome()},
			{"prefix and prefixText", d.Prefix != nil, d.PrefixText.IsSome()},
			{"suffix and suffixText", d.Suffix != nil, d.SuffixText.IsSome()},
			{"error and errorText", d.Error != nil, d.ErrorText.IsSome()},
		} {
			if pair.widget && pair.textIsSome {
				panic(errors.New(tagInputDecorationBothGiven, "declaring both "+pair.name+" is not supported", "pair", pair.name))
			}
		}
	}
	return d
}

// InputDecorationCollapsedOpts are the optional parameters of
// NewInputDecorationCollapsed.
//
// Flutter: material/input_decorator.dart InputDecoration.collapsed
type InputDecorationCollapsedOpts struct {
	// FloatingLabelBehavior is deprecated in Flutter: a collapsed decoration has no
	// label.
	FloatingLabelBehavior FloatingLabelBehavior
	// FloatingLabelAlignment is deprecated in Flutter.
	FloatingLabelAlignment foundation.Opt[FloatingLabelAlignment]
	// HintStyle is the style of the hint.
	HintStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// Hint is a widget to show as the hint.
	Hint widgets.Widget
	// HintTextDirection is the direction of the hint text.
	HintTextDirection foundation.Opt[ui.TextDirection]
	// HintMaxLines is the maximum number of lines of the hint text.
	HintMaxLines foundation.Opt[int]
	// HintFadeDuration is the duration of the fade of the hint.
	HintFadeDuration foundation.Opt[time.Duration]
	// NoMaintainHintSize makes the hint take no space when it is not shown.
	NoMaintainHintSize bool
	// MaintainLabelSize makes the label take its space whether it is shown or
	// not.
	MaintainLabelSize bool
	// Filled says whether the container is filled.
	Filled bool
	// FillColor is the color of the container.
	FillColor widgets.WidgetStateProperty[ui.Color]
	// FocusColor is the color of the focus highlight.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the hover highlight.
	HoverColor foundation.Opt[ui.Color]
	// Border is the border; the default is InputBorderNone.
	Border InputBorder
	// Disabled makes the field disabled.
	Disabled bool
	// Constraints are the constraints of the decoration.
	Constraints foundation.Opt[rendering.BoxConstraints]
}

// NewInputDecorationCollapsed creates a decoration that is the same size as the
// input field: no label, no icons, no padding and no border (the hint text is
// required, and none is Dart's null).
//
// Flutter: material/input_decorator.dart InputDecoration.collapsed
func NewInputDecorationCollapsed(hintText foundation.Opt[string], opts ...InputDecorationCollapsedOpts) InputDecoration {
	o := firstOpts(opts)
	border := o.Border
	if border == nil {
		border = InputBorderNone
	}
	return InputDecoration{
		HintText: hintText, FloatingLabelBehavior: o.FloatingLabelBehavior, FloatingLabelAlignment: o.FloatingLabelAlignment,
		HintStyle: o.HintStyle, Hint: o.Hint, HintTextDirection: o.HintTextDirection, HintMaxLines: o.HintMaxLines,
		HintFadeDuration: o.HintFadeDuration, NoMaintainHintSize: o.NoMaintainHintSize, MaintainLabelSize: o.MaintainLabelSize,
		Filled: foundation.Some(o.Filled), FillColor: o.FillColor, FocusColor: o.FocusColor, HoverColor: o.HoverColor,
		Border: border, Disabled: o.Disabled, Constraints: o.Constraints,
		IsDense: foundation.Some(false), ContentPadding: painting.EdgeInsetsZero, IsCollapsed: foundation.Some(true),
		AlignLabelWithHint: foundation.Some(false),
	}
}

// InputDecorationCopy holds the replacements for InputDecoration.CopyWith: a
// field that is None keeps the value of the copied one.
//
// Flutter: material/input_decorator.dart InputDecoration.copyWith
type InputDecorationCopy struct {
	Icon                   foundation.Opt[widgets.Widget]
	IconColor              foundation.Opt[widgets.WidgetStateProperty[ui.Color]]
	Label                  foundation.Opt[widgets.Widget]
	LabelText              foundation.Opt[string]
	LabelStyle             foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	FloatingLabelStyle     foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	Helper                 foundation.Opt[widgets.Widget]
	HelperText             foundation.Opt[string]
	HelperStyle            foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	HelperMaxLines         foundation.Opt[int]
	HintText               foundation.Opt[string]
	Hint                   foundation.Opt[widgets.Widget]
	HintStyle              foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	HintTextDirection      foundation.Opt[ui.TextDirection]
	HintMaxLines           foundation.Opt[int]
	HintFadeDuration       foundation.Opt[time.Duration]
	NoMaintainHintSize     foundation.Opt[bool]
	MaintainLabelSize      foundation.Opt[bool]
	Error                  foundation.Opt[widgets.Widget]
	ErrorText              foundation.Opt[string]
	ErrorStyle             foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	ErrorMaxLines          foundation.Opt[int]
	FloatingLabelBehavior  foundation.Opt[FloatingLabelBehavior]
	FloatingLabelAlignment foundation.Opt[FloatingLabelAlignment]
	IsCollapsed            foundation.Opt[bool]
	IsDense                foundation.Opt[bool]
	ContentPadding         foundation.Opt[painting.EdgeInsetsGeometry]
	PrefixIcon             foundation.Opt[widgets.Widget]
	Prefix                 foundation.Opt[widgets.Widget]
	PrefixText             foundation.Opt[string]
	PrefixIconConstraints  foundation.Opt[rendering.BoxConstraints]
	PrefixStyle            foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	PrefixIconColor        foundation.Opt[widgets.WidgetStateProperty[ui.Color]]
	SuffixIcon             foundation.Opt[widgets.Widget]
	Suffix                 foundation.Opt[widgets.Widget]
	SuffixText             foundation.Opt[string]
	SuffixStyle            foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	SuffixIconColor        foundation.Opt[widgets.WidgetStateProperty[ui.Color]]
	SuffixIconConstraints  foundation.Opt[rendering.BoxConstraints]
	Counter                foundation.Opt[widgets.Widget]
	CounterText            foundation.Opt[string]
	CounterStyle           foundation.Opt[widgets.WidgetStateProperty[*painting.TextStyle]]
	Filled                 foundation.Opt[bool]
	FillColor              foundation.Opt[widgets.WidgetStateProperty[ui.Color]]
	FocusColor             foundation.Opt[ui.Color]
	HoverColor             foundation.Opt[ui.Color]
	ErrorBorder            foundation.Opt[InputBorder]
	FocusedBorder          foundation.Opt[InputBorder]
	FocusedErrorBorder     foundation.Opt[InputBorder]
	DisabledBorder         foundation.Opt[InputBorder]
	EnabledBorder          foundation.Opt[InputBorder]
	Border                 foundation.Opt[InputBorder]
	Disabled               foundation.Opt[bool]
	SemanticCounterText    foundation.Opt[string]
	AlignLabelWithHint     foundation.Opt[bool]
	Constraints            foundation.Opt[rendering.BoxConstraints]
	VisualDensity          foundation.Opt[VisualDensity]
}

// CopyWith returns a copy with the fields of the argument that are set
// replaced. Unlike Flutter's, a widget field that the copy sets to nil replaces
// the value with nil.
//
// Flutter: material/input_decorator.dart InputDecoration.copyWith
func (d InputDecoration) CopyWith(c InputDecorationCopy) InputDecoration {
	if v, ok := c.Icon.Get(); ok {
		d.Icon = v
	}
	if v, ok := c.IconColor.Get(); ok {
		d.IconColor = v
	}
	if v, ok := c.Label.Get(); ok {
		d.Label = v
	}
	d.LabelText = firstOpt(c.LabelText, d.LabelText)
	if v, ok := c.LabelStyle.Get(); ok {
		d.LabelStyle = v
	}
	if v, ok := c.FloatingLabelStyle.Get(); ok {
		d.FloatingLabelStyle = v
	}
	if v, ok := c.Helper.Get(); ok {
		d.Helper = v
	}
	d.HelperText = firstOpt(c.HelperText, d.HelperText)
	if v, ok := c.HelperStyle.Get(); ok {
		d.HelperStyle = v
	}
	d.HelperMaxLines = firstOpt(c.HelperMaxLines, d.HelperMaxLines)
	d.HintText = firstOpt(c.HintText, d.HintText)
	if v, ok := c.Hint.Get(); ok {
		d.Hint = v
	}
	if v, ok := c.HintStyle.Get(); ok {
		d.HintStyle = v
	}
	d.HintTextDirection = firstOpt(c.HintTextDirection, d.HintTextDirection)
	d.HintMaxLines = firstOpt(c.HintMaxLines, d.HintMaxLines)
	d.HintFadeDuration = firstOpt(c.HintFadeDuration, d.HintFadeDuration)
	d.NoMaintainHintSize = c.NoMaintainHintSize.OrElse(d.NoMaintainHintSize)
	d.MaintainLabelSize = c.MaintainLabelSize.OrElse(d.MaintainLabelSize)
	if v, ok := c.Error.Get(); ok {
		d.Error = v
	}
	d.ErrorText = firstOpt(c.ErrorText, d.ErrorText)
	if v, ok := c.ErrorStyle.Get(); ok {
		d.ErrorStyle = v
	}
	d.ErrorMaxLines = firstOpt(c.ErrorMaxLines, d.ErrorMaxLines)
	d.FloatingLabelBehavior = c.FloatingLabelBehavior.OrElse(d.FloatingLabelBehavior)
	d.FloatingLabelAlignment = firstOpt(c.FloatingLabelAlignment, d.FloatingLabelAlignment)
	d.IsCollapsed = firstOpt(c.IsCollapsed, d.IsCollapsed)
	d.IsDense = firstOpt(c.IsDense, d.IsDense)
	if v, ok := c.ContentPadding.Get(); ok {
		d.ContentPadding = v
	}
	if v, ok := c.PrefixIcon.Get(); ok {
		d.PrefixIcon = v
	}
	if v, ok := c.Prefix.Get(); ok {
		d.Prefix = v
	}
	d.PrefixText = firstOpt(c.PrefixText, d.PrefixText)
	d.PrefixIconConstraints = firstOpt(c.PrefixIconConstraints, d.PrefixIconConstraints)
	if v, ok := c.PrefixStyle.Get(); ok {
		d.PrefixStyle = v
	}
	if v, ok := c.PrefixIconColor.Get(); ok {
		d.PrefixIconColor = v
	}
	if v, ok := c.SuffixIcon.Get(); ok {
		d.SuffixIcon = v
	}
	if v, ok := c.Suffix.Get(); ok {
		d.Suffix = v
	}
	d.SuffixText = firstOpt(c.SuffixText, d.SuffixText)
	if v, ok := c.SuffixStyle.Get(); ok {
		d.SuffixStyle = v
	}
	if v, ok := c.SuffixIconColor.Get(); ok {
		d.SuffixIconColor = v
	}
	d.SuffixIconConstraints = firstOpt(c.SuffixIconConstraints, d.SuffixIconConstraints)
	if v, ok := c.Counter.Get(); ok {
		d.Counter = v
	}
	d.CounterText = firstOpt(c.CounterText, d.CounterText)
	if v, ok := c.CounterStyle.Get(); ok {
		d.CounterStyle = v
	}
	d.Filled = firstOpt(c.Filled, d.Filled)
	if v, ok := c.FillColor.Get(); ok {
		d.FillColor = v
	}
	d.FocusColor = firstOpt(c.FocusColor, d.FocusColor)
	d.HoverColor = firstOpt(c.HoverColor, d.HoverColor)
	if v, ok := c.ErrorBorder.Get(); ok {
		d.ErrorBorder = v
	}
	if v, ok := c.FocusedBorder.Get(); ok {
		d.FocusedBorder = v
	}
	if v, ok := c.FocusedErrorBorder.Get(); ok {
		d.FocusedErrorBorder = v
	}
	if v, ok := c.DisabledBorder.Get(); ok {
		d.DisabledBorder = v
	}
	if v, ok := c.EnabledBorder.Get(); ok {
		d.EnabledBorder = v
	}
	if v, ok := c.Border.Get(); ok {
		d.Border = v
	}
	d.Disabled = c.Disabled.OrElse(d.Disabled)
	d.SemanticCounterText = firstOpt(c.SemanticCounterText, d.SemanticCounterText)
	d.AlignLabelWithHint = firstOpt(c.AlignLabelWithHint, d.AlignLabelWithHint)
	d.Constraints = firstOpt(c.Constraints, d.Constraints)
	d.VisualDensity = firstOpt(c.VisualDensity, d.VisualDensity)
	return d
}

// ApplyDefaults is the decoration with the unset fields of the theme data
// filled in: the field of the decoration when it is set, the theme's otherwise.
//
// Flutter: material/input_decorator.dart InputDecoration.applyDefaults
func (d InputDecoration) ApplyDefaults(theme InputDecorationThemeData) InputDecoration {
	pick := func(set bool) bool { return !set }
	if pick(d.LabelStyle != nil) {
		d.LabelStyle = theme.LabelStyle
	}
	if pick(d.FloatingLabelStyle != nil) {
		d.FloatingLabelStyle = theme.FloatingLabelStyle
	}
	if pick(d.HelperStyle != nil) {
		d.HelperStyle = theme.HelperStyle
	}
	d.HelperMaxLines = firstOpt(d.HelperMaxLines, theme.HelperMaxLines)
	if pick(d.HintStyle != nil) {
		d.HintStyle = theme.HintStyle
	}
	d.HintFadeDuration = firstOpt(d.HintFadeDuration, theme.HintFadeDuration)
	d.HintMaxLines = firstOpt(d.HintMaxLines, theme.HintMaxLines)
	if pick(d.ErrorStyle != nil) {
		d.ErrorStyle = theme.ErrorStyle
	}
	d.ErrorMaxLines = firstOpt(d.ErrorMaxLines, theme.ErrorMaxLines)
	if d.FloatingLabelBehavior == FloatingLabelBehaviorUnspecified {
		d.FloatingLabelBehavior = theme.FloatingLabelBehavior
	}
	d.FloatingLabelAlignment = firstOpt(d.FloatingLabelAlignment, theme.FloatingLabelAlignment)
	if d.IsDense.IsNone() {
		d.IsDense = foundation.Some(theme.IsDense)
	}
	if d.ContentPadding == nil {
		d.ContentPadding = theme.ContentPadding
	}
	if d.IsCollapsed.IsNone() {
		d.IsCollapsed = foundation.Some(theme.IsCollapsed)
	}
	if d.IconColor == nil {
		d.IconColor = theme.IconColor
	}
	if d.PrefixStyle == nil {
		d.PrefixStyle = theme.PrefixStyle
	}
	if d.PrefixIconColor == nil {
		d.PrefixIconColor = theme.PrefixIconColor
	}
	d.PrefixIconConstraints = firstOpt(d.PrefixIconConstraints, theme.PrefixIconConstraints)
	if d.SuffixStyle == nil {
		d.SuffixStyle = theme.SuffixStyle
	}
	if d.SuffixIconColor == nil {
		d.SuffixIconColor = theme.SuffixIconColor
	}
	d.SuffixIconConstraints = firstOpt(d.SuffixIconConstraints, theme.SuffixIconConstraints)
	if d.CounterStyle == nil {
		d.CounterStyle = theme.CounterStyle
	}
	if d.Filled.IsNone() {
		d.Filled = foundation.Some(theme.Filled)
	}
	if d.FillColor == nil {
		d.FillColor = theme.FillColor
	}
	d.FocusColor = firstOpt(d.FocusColor, theme.FocusColor)
	d.HoverColor = firstOpt(d.HoverColor, theme.HoverColor)
	if d.ErrorBorder == nil {
		d.ErrorBorder = theme.ErrorBorder
	}
	if d.FocusedBorder == nil {
		d.FocusedBorder = theme.FocusedBorder
	}
	if d.FocusedErrorBorder == nil {
		d.FocusedErrorBorder = theme.FocusedErrorBorder
	}
	if d.DisabledBorder == nil {
		d.DisabledBorder = theme.DisabledBorder
	}
	if d.EnabledBorder == nil {
		d.EnabledBorder = theme.EnabledBorder
	}
	if d.Border == nil {
		d.Border = theme.Border
	}
	if d.AlignLabelWithHint.IsNone() {
		d.AlignLabelWithHint = foundation.Some(theme.AlignLabelWithHint)
	}
	d.Constraints = firstOpt(d.Constraints, theme.Constraints)
	d.VisualDensity = firstOpt(d.VisualDensity, theme.VisualDensity)
	return d
}

// Equal reports whether the two decorations have equal fields (widgets by
// identity).
//
// Flutter: material/input_decorator.dart InputDecoration.==
func (d InputDecoration) Equal(o InputDecoration) bool { return structFieldsEqual(d, o) }

// structFieldsEqual compares the fields of two structs of the same type with
// foundation.ValuesEqual.
func structFieldsEqual(a, b any) bool {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	for i := range va.NumField() {
		if !va.Type().Field(i).IsExported() {
			continue
		}
		if !foundation.ValuesEqual(va.Field(i).Interface(), vb.Field(i).Interface()) {
			return false
		}
	}
	return true
}

// ToStringShort returns the name of the type.
func (InputDecoration) ToStringShort() string { return "InputDecoration" }

// String describes the decoration with the fields that are set.
//
// Flutter: material/input_decorator.dart InputDecoration.toString
func (d InputDecoration) String() string {
	return describeSetFields("InputDecoration", d)
}

// describeSetFields is "Name(field: value, ...)" with the fields of the struct
// that are not zero, for the String of the decorations.
func describeSetFields(name string, v any) string {
	rv := reflect.ValueOf(v)
	out := name + "("
	first := true
	for i := range rv.NumField() {
		f := rv.Field(i)
		if !rv.Type().Field(i).IsExported() || f.IsZero() {
			continue
		}
		if !first {
			out += ", "
		}
		first = false
		out += lowerFirst(rv.Type().Field(i).Name) + ": " + fmt.Sprint(f.Interface())
	}
	return out + ")"
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagInputDecorationBothGiven = "InputDecorationBothGiven"
)
