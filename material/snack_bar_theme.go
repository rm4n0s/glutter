// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SnackBarBehavior defines where a SnackBar should appear.
//
// Flutter: material/snack_bar_theme.dart SnackBarBehavior
type SnackBarBehavior int

const (
	// SnackBarBehaviorUnspecified is the zero value: the default of the widget.
	//
	// Flutter: no counterpart
	SnackBarBehaviorUnspecified SnackBarBehavior = iota
	// SnackBarBehaviorFixed is the default behavior: the snack bar is at the
	// bottom of the screen, below the floating action button.
	//
	// Flutter: material/snack_bar_theme.dart SnackBarBehavior.fixed
	SnackBarBehaviorFixed
	// SnackBarBehaviorFloating makes the snack bar float above the other
	// elements, including the floating action button.
	//
	// Flutter: material/snack_bar_theme.dart SnackBarBehavior.floating
	SnackBarBehaviorFloating
)

// Name returns the Dart enum name.
//
// Flutter: material/snack_bar_theme.dart SnackBarBehavior.name
func (b SnackBarBehavior) Name() string { return enumName(int(b), "fixed", "floating") }

// String returns the name Dart prints, such as "SnackBarBehavior.fixed".
//
// Flutter: material/snack_bar_theme.dart SnackBarBehavior.toString
func (b SnackBarBehavior) String() string { return "SnackBarBehavior." + b.Name() }

// SnackBarThemeData defines the default property values for descendant SnackBar
// widgets. Every field that is not set leaves the choice to the SnackBar's own
// defaults.
//
// Deviation: ActionTextColor and ActionBackgroundColor are properties of the
// widget state, where Flutter's are a Color that may be a WidgetStateColor; a
// plain color is widgets.NewWidgetStatePropertyAll(color). Flutter's assert that
// a state color and a disabled color are not given together is not ported
// (SPEC §12.10).
//
// Flutter: material/snack_bar_theme.dart SnackBarThemeData
//
//glutter:copywith value
type SnackBarThemeData struct {
	// BackgroundColor is the background color of the snack bar.
	BackgroundColor foundation.Opt[ui.Color]
	// ActionTextColor is the text color of the action.
	ActionTextColor widgets.WidgetStateProperty[ui.Color]
	// DisabledActionTextColor is the text color of the action when it is disabled.
	DisabledActionTextColor foundation.Opt[ui.Color]
	// ContentTextStyle is the style of the text of the content.
	ContentTextStyle *painting.TextStyle
	// Elevation is the elevation of the snack bar; it must not be negative.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the snack bar.
	Shape painting.ShapeBorder
	// Behavior is where the snack bar appears.
	Behavior SnackBarBehavior
	// Width is the width of a floating snack bar.
	Width foundation.Opt[float64]
	// InsetPadding is the margin around a floating snack bar.
	InsetPadding foundation.Opt[painting.EdgeInsets]
	// ShowCloseIcon says whether the snack bar has a close icon.
	ShowCloseIcon foundation.Opt[bool]
	// CloseIconColor is the color of the close icon.
	CloseIconColor foundation.Opt[ui.Color]
	// ActionOverflowThreshold is the fraction of the width above which the action
	// moves to its own row; it is between 0 and 1.
	ActionOverflowThreshold foundation.Opt[float64]
	// ActionBackgroundColor is the background color of the action.
	ActionBackgroundColor widgets.WidgetStateProperty[ui.Color]
	// DisabledActionBackgroundColor is the background color of the action when
	// it is disabled.
	DisabledActionBackgroundColor foundation.Opt[ui.Color]
	// DismissDirection is the direction in which the snack bar can be dismissed.
	DismissDirection widgets.DismissDirection
}

// SnackBarThemeDataOpts are the parameters of NewSnackBarThemeData: the fields
// of the data.
//
// Flutter: material/snack_bar_theme.dart SnackBarThemeData.new
type SnackBarThemeDataOpts = SnackBarThemeData

// NewSnackBarThemeData creates the data; every field is optional. In debug
// builds it panics with the tag SnackBarThemeElevationNegative when the
// elevation is negative, SnackBarThemeWidthNotFloating when a width is set and
// the behavior is not floating, and SnackBarThemeOverflowThreshold when the
// overflow threshold is not between 0 and 1.
//
// Flutter: material/snack_bar_theme.dart SnackBarThemeData.new
func NewSnackBarThemeData(opts ...SnackBarThemeDataOpts) SnackBarThemeData {
	d := firstOpts(opts)
	if foundation.DebugMode {
		if d.Elevation.OrElse(0) < 0 {
			panic(errors.New("SnackBarThemeElevationNegative", "the elevation of a SnackBarThemeData must not be negative", "elevation", d.Elevation.Value()))
		}
		if d.Width.IsSome() && d.Behavior != SnackBarBehaviorFloating {
			panic(errors.New("SnackBarThemeWidthNotFloating", "Width can only be set if behaviour is SnackBarBehavior.floating"))
		}
		if t, ok := d.ActionOverflowThreshold.Get(); ok && (t < 0 || t > 1) {
			panic(errors.New("SnackBarThemeOverflowThreshold", "Action overflow threshold must be between 0 and 1 inclusive", "threshold", t))
		}
	}
	return d
}

// lerpStateColor interpolates two properties of colors like Color.lerp does the
// colors they stand for: it is nil when both are nil.
func lerpStateColor(a, b widgets.WidgetStateProperty[ui.Color], t float64) widgets.WidgetStateProperty[ui.Color] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
		var ca, cb foundation.Opt[ui.Color]
		if a != nil {
			ca = foundation.Some(a.Resolve(states))
		}
		if b != nil {
			cb = foundation.Some(b.Resolve(states))
		}
		return lerpOptColor(ca, cb, t).Value()
	})
}

// SnackBarThemeDataLerp linearly interpolates between two snack bar themes. The
// behavior and the dismiss direction switch at t = 0.5.
//
// Flutter: material/snack_bar_theme.dart SnackBarThemeData.lerp
func SnackBarThemeDataLerp(a, b SnackBarThemeData, t float64) SnackBarThemeData {
	inset := func(a, b foundation.Opt[painting.EdgeInsets]) foundation.Opt[painting.EdgeInsets] {
		if a.IsNone() && b.IsNone() {
			return foundation.None[painting.EdgeInsets]()
		}
		var pa, pb painting.EdgeInsetsGeometry
		if v, ok := a.Get(); ok {
			pa = v
		}
		if v, ok := b.Get(); ok {
			pb = v
		}
		return foundation.Some(painting.EdgeInsetsGeometryLerp(pa, pb, t).(painting.EdgeInsets))
	}
	return SnackBarThemeData{
		BackgroundColor:               lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		ActionTextColor:               lerpStateColor(a.ActionTextColor, b.ActionTextColor, t),
		DisabledActionTextColor:       lerpOptColor(a.DisabledActionTextColor, b.DisabledActionTextColor, t),
		ContentTextStyle:              painting.TextStyleLerp(a.ContentTextStyle, b.ContentTextStyle, t),
		Elevation:                     lerpOptDouble(a.Elevation, b.Elevation, t),
		Shape:                         painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		Behavior:                      lerpFirstOrSecond(a.Behavior, b.Behavior, t),
		Width:                         lerpOptDouble(a.Width, b.Width, t),
		InsetPadding:                  inset(a.InsetPadding, b.InsetPadding),
		CloseIconColor:                lerpOptColor(a.CloseIconColor, b.CloseIconColor, t),
		ActionOverflowThreshold:       lerpOptDouble(a.ActionOverflowThreshold, b.ActionOverflowThreshold, t),
		ActionBackgroundColor:         lerpStateColor(a.ActionBackgroundColor, b.ActionBackgroundColor, t),
		DisabledActionBackgroundColor: lerpOptColor(a.DisabledActionBackgroundColor, b.DisabledActionBackgroundColor, t),
		DismissDirection:              lerpFirstOrSecond(a.DismissDirection, b.DismissDirection, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/snack_bar_theme.dart SnackBarThemeData.==
func (d SnackBarThemeData) Equal(o SnackBarThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor && foundation.ValuesEqual(d.ActionTextColor, o.ActionTextColor) &&
		d.DisabledActionTextColor == o.DisabledActionTextColor && textStyleEqual(d.ContentTextStyle, o.ContentTextStyle) &&
		d.Elevation == o.Elevation && foundation.ValuesEqual(d.Shape, o.Shape) && d.Behavior == o.Behavior && d.Width == o.Width &&
		d.InsetPadding == o.InsetPadding && d.ShowCloseIcon == o.ShowCloseIcon && d.CloseIconColor == o.CloseIconColor &&
		d.ActionOverflowThreshold == o.ActionOverflowThreshold && foundation.ValuesEqual(d.ActionBackgroundColor, o.ActionBackgroundColor) &&
		d.DisabledActionBackgroundColor == o.DisabledActionBackgroundColor && d.DismissDirection == o.DismissDirection
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SnackBarThemeData) ToStringShort() string { return "SnackBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d SnackBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/snack_bar_theme.dart SnackBarThemeData.debugFillProperties
func (d SnackBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(objectProp("actionTextColor", propOrNil(d.ActionTextColor)))
	p.Add(colorProp("disabledActionTextColor", d.DisabledActionTextColor))
	p.Add(objectProp("contentTextStyle", d.ContentTextStyle))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(objectProp("shape", d.Shape))
	if d.Behavior != SnackBarBehaviorUnspecified {
		p.Add(objectProp("behavior", d.Behavior))
	} else {
		p.Add(objectProp("behavior", nil))
	}
	p.Add(doubleProp("width", d.Width))
	p.Add(objectProp("insetPadding", optValue(d.InsetPadding)))
	p.Add(foundation.NewDiagnosticsPropertyOpt("showCloseIcon", d.ShowCloseIcon, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(colorProp("closeIconColor", d.CloseIconColor))
	p.Add(doubleProp("actionOverflowThreshold", d.ActionOverflowThreshold))
	p.Add(objectProp("actionBackgroundColor", propOrNil(d.ActionBackgroundColor)))
	p.Add(colorProp("disabledActionBackgroundColor", d.DisabledActionBackgroundColor))
	if d.DismissDirection != widgets.DismissDirectionUnspecified {
		p.Add(objectProp("dismissDirection", d.DismissDirection))
	} else {
		p.Add(objectProp("dismissDirection", nil))
	}
}

// propOrNil is the property as an any, or untyped nil when it is nil.
func propOrNil[T any](p widgets.WidgetStateProperty[T]) any {
	if p == nil {
		return nil
	}
	return p
}

// SnackBarTheme is an inherited theme that overrides the SnackBarThemeData of
// the theme for the SnackBar widgets below it.
//
// Flutter: material/snack_bar_theme.dart SnackBarTheme
type SnackBarTheme struct {
	widgets.InheritedTheme
	data SnackBarThemeData
}

// NewSnackBarTheme creates the theme.
//
// Flutter: material/snack_bar_theme.dart SnackBarTheme.new
func NewSnackBarTheme(data SnackBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *SnackBarTheme {
	return &SnackBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/snack_bar_theme.dart SnackBarTheme.data
func (w *SnackBarTheme) Data() SnackBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/snack_bar_theme.dart SnackBarTheme.updateShouldNotify
func (w *SnackBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*SnackBarTheme).data)
}

// Wrap makes a SnackBarTheme with the same data around the child.
//
// Flutter: material/snack_bar_theme.dart SnackBarTheme.wrap
func (w *SnackBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewSnackBarTheme(w.data, child)
}

// SnackBarThemeOf returns the data of the closest SnackBarTheme above the
// context, or the snackBarTheme of the closest Theme. The context depends on
// it.
//
// Flutter: material/snack_bar_theme.dart SnackBarTheme.of
func SnackBarThemeOf(bc widgets.BuildContext) SnackBarThemeData {
	if t, ok := widgets.DependOn[*SnackBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).SnackBarTheme
}
