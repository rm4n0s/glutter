// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// MenuStyle is the visual properties that the buttons share: the appearance of a
// TextButton, an ElevatedButton, a FilledButton or an OutlinedButton comes from
// the MenuStyle of the widget, then of the theme, then of the defaults. A
// property that is not set (nil, none or Unspecified) leaves the choice to the next.
//
// Flutter: material/menu_style.dart MenuStyle
type MenuStyle struct {
	// BackgroundColor is the color of the Material of the menu, in each state.
	BackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ShadowColor is the color of the shadow below the menu, in each state.
	ShadowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SurfaceTintColor is the surface tint of the Material of the menu, in each state.
	SurfaceTintColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// Elevation is the elevation of the Material of the menu, in each state.
	Elevation widgets.WidgetStateProperty[foundation.Opt[float64]]
	// Padding is the padding between the border of the menu and its items, in each state.
	Padding widgets.WidgetStateProperty[painting.EdgeInsetsGeometry]
	// MinimumSize is the minimum size of the menu, in each state.
	MinimumSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// FixedSize is the size of the menu, in each state; an infinite width or height does not fix it.
	FixedSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// MaximumSize is the maximum size of the menu, in each state.
	MaximumSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// Side is the outline of the menu, in each state.
	Side widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the menu, in each state; its side is replaced by Side.
	Shape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// MouseCursor is the cursor over the menu, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// VisualDensity defines how compact the layout of the menu is; it only changes the horizontal padding, and never makes it negative.
	VisualDensity foundation.Opt[VisualDensity]
	// Alignment is the alignment of the items within the menu.
	Alignment painting.AlignmentGeometry
}

// MenuStyleOpts are the parameters of NewMenuStyle: the properties of the style.
//
// Flutter: material/menu_style.dart MenuStyle.new
type MenuStyleOpts = MenuStyle

// NewMenuStyle creates a style; every property is optional.
//
// Flutter: material/menu_style.dart MenuStyle.new
func NewMenuStyle(opts ...MenuStyleOpts) *MenuStyle {
	style := firstOpts(opts)
	return &style
}

// CopyWith returns a copy of the style with the properties that overrides sets
// replaced.
//
// Flutter: material/menu_style.dart MenuStyle.copyWith
func (s *MenuStyle) CopyWith(overrides MenuStyleOpts) *MenuStyle {
	out := *s
	if overrides.BackgroundColor != nil {
		out.BackgroundColor = overrides.BackgroundColor
	}
	if overrides.ShadowColor != nil {
		out.ShadowColor = overrides.ShadowColor
	}
	if overrides.SurfaceTintColor != nil {
		out.SurfaceTintColor = overrides.SurfaceTintColor
	}
	if overrides.Elevation != nil {
		out.Elevation = overrides.Elevation
	}
	if overrides.Padding != nil {
		out.Padding = overrides.Padding
	}
	if overrides.MinimumSize != nil {
		out.MinimumSize = overrides.MinimumSize
	}
	if overrides.FixedSize != nil {
		out.FixedSize = overrides.FixedSize
	}
	if overrides.MaximumSize != nil {
		out.MaximumSize = overrides.MaximumSize
	}
	if overrides.Side != nil {
		out.Side = overrides.Side
	}
	if overrides.Shape != nil {
		out.Shape = overrides.Shape
	}
	if overrides.MouseCursor != nil {
		out.MouseCursor = overrides.MouseCursor
	}
	if overrides.VisualDensity.IsSome() {
		out.VisualDensity = overrides.VisualDensity
	}
	if overrides.Alignment != nil {
		out.Alignment = overrides.Alignment
	}
	return &out
}

// Merge returns a copy of the style where the properties that this style does not
// set come from the other one. A nil style leaves this one as it is.
//
// Flutter: material/menu_style.dart MenuStyle.merge
func (s *MenuStyle) Merge(style *MenuStyle) *MenuStyle {
	if style == nil {
		return s
	}
	out := *style
	if s.BackgroundColor != nil {
		out.BackgroundColor = s.BackgroundColor
	}
	if s.ShadowColor != nil {
		out.ShadowColor = s.ShadowColor
	}
	if s.SurfaceTintColor != nil {
		out.SurfaceTintColor = s.SurfaceTintColor
	}
	if s.Elevation != nil {
		out.Elevation = s.Elevation
	}
	if s.Padding != nil {
		out.Padding = s.Padding
	}
	if s.MinimumSize != nil {
		out.MinimumSize = s.MinimumSize
	}
	if s.FixedSize != nil {
		out.FixedSize = s.FixedSize
	}
	if s.MaximumSize != nil {
		out.MaximumSize = s.MaximumSize
	}
	if s.Side != nil {
		out.Side = s.Side
	}
	if s.Shape != nil {
		out.Shape = s.Shape
	}
	if s.MouseCursor != nil {
		out.MouseCursor = s.MouseCursor
	}
	if s.VisualDensity.IsSome() {
		out.VisualDensity = s.VisualDensity
	}
	if s.Alignment != nil {
		out.Alignment = s.Alignment
	}
	return &out
}

// Equal reports whether the two styles have equal properties.
//
// Flutter: material/menu_style.dart MenuStyle.==
func (s *MenuStyle) Equal(o *MenuStyle) bool {
	if s == nil || o == nil {
		return s == o
	}
	return statePropertyEqual(s.BackgroundColor, o.BackgroundColor) &&
		statePropertyEqual(s.ShadowColor, o.ShadowColor) &&
		statePropertyEqual(s.SurfaceTintColor, o.SurfaceTintColor) &&
		statePropertyEqual(s.Elevation, o.Elevation) &&
		statePropertyEqual(s.Padding, o.Padding) &&
		statePropertyEqual(s.MinimumSize, o.MinimumSize) &&
		statePropertyEqual(s.FixedSize, o.FixedSize) &&
		statePropertyEqual(s.MaximumSize, o.MaximumSize) &&
		statePropertyEqual(s.Side, o.Side) &&
		statePropertyEqual(s.Shape, o.Shape) &&
		statePropertyEqual(s.MouseCursor, o.MouseCursor) &&
		foundation.ValuesEqual(s.VisualDensity, o.VisualDensity) &&
		foundation.ValuesEqual(s.Alignment, o.Alignment)
}

// MenuStyleLerp linearly interpolates between two styles.
//
// Flutter: material/menu_style.dart MenuStyle.lerp
func MenuStyleLerp(a, b *MenuStyle, t float64) *MenuStyle {
	if a == b {
		return a
	}
	if a == nil {
		a = &MenuStyle{}
	}
	if b == nil {
		b = &MenuStyle{}
	}
	return &MenuStyle{
		BackgroundColor:  lerpStateOptColor(a.BackgroundColor, b.BackgroundColor, t),
		ShadowColor:      lerpStateOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpStateOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Elevation:        lerpStateOpt(a.Elevation, b.Elevation, t, lerpOptDouble),
		Padding:          lerpStateNil(a.Padding, b.Padding, t, painting.EdgeInsetsGeometryLerp),
		MinimumSize:      lerpStateOpt(a.MinimumSize, b.MinimumSize, t, lerpOptSize),
		FixedSize:        lerpStateOpt(a.FixedSize, b.FixedSize, t, lerpOptSize),
		MaximumSize:      lerpStateOpt(a.MaximumSize, b.MaximumSize, t, lerpOptSize),
		Side:             lerpStateOpt(a.Side, b.Side, t, lerpOptBorderSide),
		Shape:            lerpStateNil(a.Shape, b.Shape, t, painting.OutlinedBorderLerp),
		MouseCursor:      lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		VisualDensity:    lerpFirstOrSecond(a.VisualDensity, b.VisualDensity, t),
		Alignment:        painting.AlignmentGeometryLerp(a.Alignment, b.Alignment, t),
	}
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (*MenuStyle) ToStringShort() string { return "MenuStyle" }

// String returns the description of the style.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (s *MenuStyle) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the properties that are set.
//
// Flutter: material/menu_style.dart MenuStyle.debugFillProperties
func (s *MenuStyle) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("backgroundColor", s.BackgroundColor))
	p.Add(objectProp("shadowColor", s.ShadowColor))
	p.Add(objectProp("surfaceTintColor", s.SurfaceTintColor))
	p.Add(objectProp("elevation", s.Elevation))
	p.Add(objectProp("padding", s.Padding))
	p.Add(objectProp("minimumSize", s.MinimumSize))
	p.Add(objectProp("fixedSize", s.FixedSize))
	p.Add(objectProp("maximumSize", s.MaximumSize))
	p.Add(objectProp("side", s.Side))
	p.Add(objectProp("shape", s.Shape))
	p.Add(objectProp("mouseCursor", s.MouseCursor))
	p.Add(objectProp("visualDensity", optValue(s.VisualDensity)))
	p.Add(objectProp("alignment", s.Alignment))
}
