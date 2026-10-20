// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ExpansionTileThemeData defines default property values for descendant ExpansionTile widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileThemeData
//
//glutter:copywith value
type ExpansionTileThemeData struct {
	// BackgroundColor is the color of the tile when it is expanded.
	BackgroundColor foundation.Opt[ui.Color]
	// CollapsedBackgroundColor is the color of the tile when it is collapsed.
	CollapsedBackgroundColor foundation.Opt[ui.Color]
	// TilePadding is the padding of the tile's header.
	TilePadding painting.EdgeInsetsGeometry
	// ExpandedAlignment is the alignment of the children when the tile is expanded.
	ExpandedAlignment painting.AlignmentGeometry
	// ChildrenPadding is the padding around the children.
	ChildrenPadding painting.EdgeInsetsGeometry
	// IconColor is the color of the icon when the tile is expanded.
	IconColor foundation.Opt[ui.Color]
	// CollapsedIconColor is the color of the icon when the tile is collapsed.
	CollapsedIconColor foundation.Opt[ui.Color]
	// TextColor is the color of the title when the tile is expanded.
	TextColor foundation.Opt[ui.Color]
	// CollapsedTextColor is the color of the title when the tile is collapsed.
	CollapsedTextColor foundation.Opt[ui.Color]
	// Shape is the shape of the tile when it is expanded.
	Shape painting.ShapeBorder
	// CollapsedShape is the shape of the tile when it is collapsed.
	CollapsedShape painting.ShapeBorder
	// ClipBehavior is how the tile is clipped.
	ClipBehavior ui.Clip
	// ExpansionAnimationStyle overrides the duration and the curves of the expansion.
	ExpansionAnimationStyle animation.AnimationStyle
}

// ExpansionTileThemeDataOpts are the parameters of NewExpansionTileThemeData: the fields of the data.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileThemeData.new
type ExpansionTileThemeDataOpts = ExpansionTileThemeData

// NewExpansionTileThemeData creates the data; every field is optional.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileThemeData.new
func NewExpansionTileThemeData(opts ...ExpansionTileThemeDataOpts) ExpansionTileThemeData {
	return firstOpts(opts)
}

// ExpansionTileThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileThemeData.lerp
func ExpansionTileThemeDataLerp(a, b ExpansionTileThemeData, t float64) ExpansionTileThemeData {
	return ExpansionTileThemeData{
		BackgroundColor:          lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		CollapsedBackgroundColor: lerpOptColor(a.CollapsedBackgroundColor, b.CollapsedBackgroundColor, t),
		TilePadding:              painting.EdgeInsetsGeometryLerp(a.TilePadding, b.TilePadding, t),
		ExpandedAlignment:        painting.AlignmentGeometryLerp(a.ExpandedAlignment, b.ExpandedAlignment, t),
		ChildrenPadding:          painting.EdgeInsetsGeometryLerp(a.ChildrenPadding, b.ChildrenPadding, t),
		IconColor:                lerpOptColor(a.IconColor, b.IconColor, t),
		CollapsedIconColor:       lerpOptColor(a.CollapsedIconColor, b.CollapsedIconColor, t),
		TextColor:                lerpOptColor(a.TextColor, b.TextColor, t),
		CollapsedTextColor:       lerpOptColor(a.CollapsedTextColor, b.CollapsedTextColor, t),
		Shape:                    painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		CollapsedShape:           painting.ShapeBorderLerp(a.CollapsedShape, b.CollapsedShape, t),
		ClipBehavior:             lerpFirstOrSecond(a.ClipBehavior, b.ClipBehavior, t),
		ExpansionAnimationStyle:  lerpFirstOrSecond(a.ExpansionAnimationStyle, b.ExpansionAnimationStyle, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileThemeData.==
func (d ExpansionTileThemeData) Equal(o ExpansionTileThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.CollapsedBackgroundColor == o.CollapsedBackgroundColor &&
		foundation.ValuesEqual(d.TilePadding, o.TilePadding) &&
		foundation.ValuesEqual(d.ExpandedAlignment, o.ExpandedAlignment) &&
		foundation.ValuesEqual(d.ChildrenPadding, o.ChildrenPadding) &&
		d.IconColor == o.IconColor &&
		d.CollapsedIconColor == o.CollapsedIconColor &&
		d.TextColor == o.TextColor &&
		d.CollapsedTextColor == o.CollapsedTextColor &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		foundation.ValuesEqual(d.CollapsedShape, o.CollapsedShape) &&
		d.ClipBehavior == o.ClipBehavior &&
		foundation.ValuesEqual(d.ExpansionAnimationStyle, o.ExpansionAnimationStyle)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ExpansionTileThemeData) ToStringShort() string { return "ExpansionTileThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ExpansionTileThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileThemeData.debugFillProperties
func (d ExpansionTileThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("collapsedBackgroundColor", d.CollapsedBackgroundColor))
	p.Add(objectProp("tilePadding", d.TilePadding))
	p.Add(objectProp("expandedAlignment", d.ExpandedAlignment))
	p.Add(objectProp("childrenPadding", d.ChildrenPadding))
	p.Add(colorProp("iconColor", d.IconColor))
	p.Add(colorProp("collapsedIconColor", d.CollapsedIconColor))
	p.Add(colorProp("textColor", d.TextColor))
	p.Add(colorProp("collapsedTextColor", d.CollapsedTextColor))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("collapsedShape", d.CollapsedShape))
	p.Add(objectProp("clipBehavior", nilIfZero(d.ClipBehavior)))
	p.Add(objectProp("expansionAnimationStyle", d.ExpansionAnimationStyle))
}

// ExpansionTileTheme is an inherited theme that overrides the ExpansionTileThemeData of the theme for the
// widgets below it.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileTheme
type ExpansionTileTheme struct {
	widgets.InheritedTheme
	data ExpansionTileThemeData
}

// NewExpansionTileTheme creates the theme.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileTheme.new
func NewExpansionTileTheme(data ExpansionTileThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ExpansionTileTheme {
	return &ExpansionTileTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileTheme.data
func (w *ExpansionTileTheme) Data() ExpansionTileThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileTheme.updateShouldNotify
func (w *ExpansionTileTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ExpansionTileTheme).data)
}

// Wrap makes a ExpansionTileTheme with the same data around the child.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileTheme.wrap
func (w *ExpansionTileTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewExpansionTileTheme(w.data, child)
}

// ExpansionTileThemeOf returns the data of the closest ExpansionTileTheme above the context, or the
// expansionTileTheme of the closest Theme. The context depends on it.
//
// Flutter: material/expansion_tile_theme.dart ExpansionTileTheme.of
func ExpansionTileThemeOf(bc widgets.BuildContext) ExpansionTileThemeData {
	if t, ok := widgets.DependOn[*ExpansionTileTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ExpansionTileTheme
}
