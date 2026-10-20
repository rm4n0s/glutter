// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// BadgeThemeData defines default property values for descendant Badge widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/badge_theme.dart BadgeThemeData
//
//glutter:copywith value
type BadgeThemeData struct {
	// BackgroundColor is the color of the badge's background.
	BackgroundColor foundation.Opt[ui.Color]
	// TextColor is the color of the badge's label.
	TextColor foundation.Opt[ui.Color]
	// SmallSize is the size of a badge without a label.
	SmallSize foundation.Opt[float64]
	// LargeSize is the height of a badge with a label.
	LargeSize foundation.Opt[float64]
	// TextStyle is the style of the label.
	TextStyle *painting.TextStyle
	// Padding is the padding around the label.
	Padding painting.EdgeInsetsGeometry
	// Alignment is where the badge is placed over its child.
	Alignment painting.AlignmentGeometry
	// Offset is the offset of the badge from its alignment.
	Offset foundation.Opt[ui.Offset]
}

// BadgeThemeDataOpts are the parameters of NewBadgeThemeData: the fields of the data.
//
// Flutter: material/badge_theme.dart BadgeThemeData.new
type BadgeThemeDataOpts = BadgeThemeData

// NewBadgeThemeData creates the data; every field is optional.
//
// Flutter: material/badge_theme.dart BadgeThemeData.new
func NewBadgeThemeData(opts ...BadgeThemeDataOpts) BadgeThemeData {
	return firstOpts(opts)
}

// BadgeThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/badge_theme.dart BadgeThemeData.lerp
func BadgeThemeDataLerp(a, b BadgeThemeData, t float64) BadgeThemeData {
	return BadgeThemeData{
		BackgroundColor: lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		TextColor:       lerpOptColor(a.TextColor, b.TextColor, t),
		SmallSize:       lerpOptDouble(a.SmallSize, b.SmallSize, t),
		LargeSize:       lerpOptDouble(a.LargeSize, b.LargeSize, t),
		TextStyle:       painting.TextStyleLerp(a.TextStyle, b.TextStyle, t),
		Padding:         painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		Alignment:       painting.AlignmentGeometryLerp(a.Alignment, b.Alignment, t),
		Offset:          lerpOptOffset(a.Offset, b.Offset, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/badge_theme.dart BadgeThemeData.==
func (d BadgeThemeData) Equal(o BadgeThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.TextColor == o.TextColor &&
		d.SmallSize == o.SmallSize &&
		d.LargeSize == o.LargeSize &&
		textStyleEqual(d.TextStyle, o.TextStyle) &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		foundation.ValuesEqual(d.Alignment, o.Alignment) &&
		d.Offset == o.Offset
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (BadgeThemeData) ToStringShort() string { return "BadgeThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d BadgeThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/badge_theme.dart BadgeThemeData.debugFillProperties
func (d BadgeThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("textColor", d.TextColor))
	p.Add(doubleProp("smallSize", d.SmallSize))
	p.Add(doubleProp("largeSize", d.LargeSize))
	p.Add(objectProp("textStyle", d.TextStyle))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("alignment", d.Alignment))
	p.Add(objectProp("offset", optValue(d.Offset)))
}

// BadgeTheme is an inherited theme that overrides the BadgeThemeData of the theme for the
// widgets below it.
//
// Flutter: material/badge_theme.dart BadgeTheme
type BadgeTheme struct {
	widgets.InheritedTheme
	data BadgeThemeData
}

// NewBadgeTheme creates the theme.
//
// Flutter: material/badge_theme.dart BadgeTheme.new
func NewBadgeTheme(data BadgeThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *BadgeTheme {
	return &BadgeTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/badge_theme.dart BadgeTheme.data
func (w *BadgeTheme) Data() BadgeThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/badge_theme.dart BadgeTheme.updateShouldNotify
func (w *BadgeTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*BadgeTheme).data)
}

// Wrap makes a BadgeTheme with the same data around the child.
//
// Flutter: material/badge_theme.dart BadgeTheme.wrap
func (w *BadgeTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewBadgeTheme(w.data, child)
}

// BadgeThemeOf returns the data of the closest BadgeTheme above the context, or the
// badgeTheme of the closest Theme. The context depends on it.
//
// Flutter: material/badge_theme.dart BadgeTheme.of
func BadgeThemeOf(bc widgets.BuildContext) BadgeThemeData {
	if t, ok := widgets.DependOn[*BadgeTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).BadgeTheme
}
