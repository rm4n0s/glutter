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
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ButtonLayerBuilder is the type of the functions that build the background and
// the foreground layers of a button.
//
// Flutter: material/button_style.dart ButtonLayerBuilder
type ButtonLayerBuilder func(bc widgets.BuildContext, states widgets.WidgetStates, child widgets.Widget) widgets.Widget

// ButtonStyle is the visual properties that the buttons share: the appearance of a
// TextButton, an ElevatedButton, a FilledButton or an OutlinedButton comes from
// the ButtonStyle of the widget, then of the theme, then of the defaults. A
// property that is not set (nil, none or Unspecified) leaves the choice to the next.
//
// Flutter: material/button_style.dart ButtonStyle
type ButtonStyle struct {
	// TextStyle is the style of the text of the button, in each state. A button
	// merges it onto the text style of its theme's style and of its defaults
	// (TextStyle.Merge): a style that gives only a size or a weight keeps the font
	// family and the rest of the default label style, and the label keeps the
	// foreground color. A style whose Inherit is false replaces them, as in Flutter.
	TextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// BackgroundColor is the color of the Material of the button, in each state.
	BackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ForegroundColor is the color of the text and the icons of the button, in each state.
	ForegroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// OverlayColor is the highlight color of the button, in each state: the splash and the hover, focus and press colors.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ShadowColor is the color of the shadow below the button, in each state.
	ShadowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SurfaceTintColor is the surface tint of the Material of the button, in each state.
	SurfaceTintColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// Elevation is the elevation of the Material of the button, in each state.
	Elevation widgets.WidgetStateProperty[foundation.Opt[float64]]
	// Padding is the padding between the border of the button and its child, in each state.
	Padding widgets.WidgetStateProperty[painting.EdgeInsetsGeometry]
	// MinimumSize is the minimum size of the button, in each state.
	MinimumSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// FixedSize is the size of the button, in each state; an infinite width or height does not fix it.
	FixedSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// MaximumSize is the maximum size of the button, in each state.
	MaximumSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// IconColor is the color of the icon of the button, in each state; the default is the foreground color.
	IconColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// IconSize is the size of the icon of the button, in each state.
	IconSize widgets.WidgetStateProperty[foundation.Opt[float64]]
	// IconAlignment is the side of the label that the icon is on.
	IconAlignment IconAlignment
	// Side is the outline of the button, in each state.
	Side widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the button, in each state; its side is replaced by Side.
	Shape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// MouseCursor is the cursor over the button, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// VisualDensity defines how compact the layout of the button is.
	VisualDensity foundation.Opt[VisualDensity]
	// TapTargetSize configures the minimum size of the tap target.
	TapTargetSize MaterialTapTargetSize
	// AnimationDuration is the duration of the changes of the shape and elevation of the button.
	AnimationDuration foundation.Opt[time.Duration]
	// EnableFeedback says whether detected gestures give acoustic and haptic feedback.
	EnableFeedback foundation.Opt[bool]
	// Alignment is the alignment of the child within the button.
	Alignment painting.AlignmentGeometry
	// SplashFactory creates the splashes of the button.
	SplashFactory InteractiveInkFeatureFactory
	// BackgroundBuilder builds a layer between the Material of the button and its child.
	BackgroundBuilder ButtonLayerBuilder
	// ForegroundBuilder builds a layer above the child of the button.
	ForegroundBuilder ButtonLayerBuilder
}

// ButtonStyleOpts are the parameters of NewButtonStyle: the properties of the style.
//
// Flutter: material/button_style.dart ButtonStyle.new
type ButtonStyleOpts = ButtonStyle

// NewButtonStyle creates a style; every property is optional.
//
// Flutter: material/button_style.dart ButtonStyle.new
func NewButtonStyle(opts ...ButtonStyleOpts) *ButtonStyle {
	style := firstOpts(opts)
	return &style
}

// CopyWith returns a copy of the style with the properties that overrides sets
// replaced.
//
// Flutter: material/button_style.dart ButtonStyle.copyWith
func (s *ButtonStyle) CopyWith(overrides ButtonStyleOpts) *ButtonStyle {
	out := *s
	if overrides.TextStyle != nil {
		out.TextStyle = overrides.TextStyle
	}
	if overrides.BackgroundColor != nil {
		out.BackgroundColor = overrides.BackgroundColor
	}
	if overrides.ForegroundColor != nil {
		out.ForegroundColor = overrides.ForegroundColor
	}
	if overrides.OverlayColor != nil {
		out.OverlayColor = overrides.OverlayColor
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
	if overrides.IconColor != nil {
		out.IconColor = overrides.IconColor
	}
	if overrides.IconSize != nil {
		out.IconSize = overrides.IconSize
	}
	if overrides.IconAlignment != 0 {
		out.IconAlignment = overrides.IconAlignment
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
	if overrides.TapTargetSize != 0 {
		out.TapTargetSize = overrides.TapTargetSize
	}
	if overrides.AnimationDuration.IsSome() {
		out.AnimationDuration = overrides.AnimationDuration
	}
	if overrides.EnableFeedback.IsSome() {
		out.EnableFeedback = overrides.EnableFeedback
	}
	if overrides.Alignment != nil {
		out.Alignment = overrides.Alignment
	}
	if overrides.SplashFactory != nil {
		out.SplashFactory = overrides.SplashFactory
	}
	if overrides.BackgroundBuilder != nil {
		out.BackgroundBuilder = overrides.BackgroundBuilder
	}
	if overrides.ForegroundBuilder != nil {
		out.ForegroundBuilder = overrides.ForegroundBuilder
	}
	return &out
}

// Merge returns a copy of the style where the properties that this style does not
// set come from the other one. A nil style leaves this one as it is.
//
// Flutter: material/button_style.dart ButtonStyle.merge
func (s *ButtonStyle) Merge(style *ButtonStyle) *ButtonStyle {
	if style == nil {
		return s
	}
	out := *style
	if s.TextStyle != nil {
		out.TextStyle = s.TextStyle
	}
	if s.BackgroundColor != nil {
		out.BackgroundColor = s.BackgroundColor
	}
	if s.ForegroundColor != nil {
		out.ForegroundColor = s.ForegroundColor
	}
	if s.OverlayColor != nil {
		out.OverlayColor = s.OverlayColor
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
	if s.IconColor != nil {
		out.IconColor = s.IconColor
	}
	if s.IconSize != nil {
		out.IconSize = s.IconSize
	}
	if s.IconAlignment != 0 {
		out.IconAlignment = s.IconAlignment
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
	if s.TapTargetSize != 0 {
		out.TapTargetSize = s.TapTargetSize
	}
	if s.AnimationDuration.IsSome() {
		out.AnimationDuration = s.AnimationDuration
	}
	if s.EnableFeedback.IsSome() {
		out.EnableFeedback = s.EnableFeedback
	}
	if s.Alignment != nil {
		out.Alignment = s.Alignment
	}
	if s.SplashFactory != nil {
		out.SplashFactory = s.SplashFactory
	}
	if s.BackgroundBuilder != nil {
		out.BackgroundBuilder = s.BackgroundBuilder
	}
	if s.ForegroundBuilder != nil {
		out.ForegroundBuilder = s.ForegroundBuilder
	}
	return &out
}

// Equal reports whether the two styles have equal properties.
//
// Flutter: material/button_style.dart ButtonStyle.==
func (s *ButtonStyle) Equal(o *ButtonStyle) bool {
	if s == nil || o == nil {
		return s == o
	}
	return statePropertyEqual(s.TextStyle, o.TextStyle) &&
		statePropertyEqual(s.BackgroundColor, o.BackgroundColor) &&
		statePropertyEqual(s.ForegroundColor, o.ForegroundColor) &&
		statePropertyEqual(s.OverlayColor, o.OverlayColor) &&
		statePropertyEqual(s.ShadowColor, o.ShadowColor) &&
		statePropertyEqual(s.SurfaceTintColor, o.SurfaceTintColor) &&
		statePropertyEqual(s.Elevation, o.Elevation) &&
		statePropertyEqual(s.Padding, o.Padding) &&
		statePropertyEqual(s.MinimumSize, o.MinimumSize) &&
		statePropertyEqual(s.FixedSize, o.FixedSize) &&
		statePropertyEqual(s.MaximumSize, o.MaximumSize) &&
		statePropertyEqual(s.IconColor, o.IconColor) &&
		statePropertyEqual(s.IconSize, o.IconSize) &&
		foundation.ValuesEqual(s.IconAlignment, o.IconAlignment) &&
		statePropertyEqual(s.Side, o.Side) &&
		statePropertyEqual(s.Shape, o.Shape) &&
		statePropertyEqual(s.MouseCursor, o.MouseCursor) &&
		foundation.ValuesEqual(s.VisualDensity, o.VisualDensity) &&
		foundation.ValuesEqual(s.TapTargetSize, o.TapTargetSize) &&
		foundation.ValuesEqual(s.AnimationDuration, o.AnimationDuration) &&
		foundation.ValuesEqual(s.EnableFeedback, o.EnableFeedback) &&
		foundation.ValuesEqual(s.Alignment, o.Alignment) &&
		foundation.ValuesEqual(s.SplashFactory, o.SplashFactory) &&
		foundation.ValuesEqual(s.BackgroundBuilder, o.BackgroundBuilder) &&
		foundation.ValuesEqual(s.ForegroundBuilder, o.ForegroundBuilder)
}

// ButtonStyleLerp linearly interpolates between two styles.
//
// Flutter: material/button_style.dart ButtonStyle.lerp
func ButtonStyleLerp(a, b *ButtonStyle, t float64) *ButtonStyle {
	if a == b {
		return a
	}
	if a == nil {
		a = &ButtonStyle{}
	}
	if b == nil {
		b = &ButtonStyle{}
	}
	return &ButtonStyle{
		TextStyle:         lerpStateNil(a.TextStyle, b.TextStyle, t, painting.TextStyleLerp),
		BackgroundColor:   lerpStateOptColor(a.BackgroundColor, b.BackgroundColor, t),
		ForegroundColor:   lerpStateOptColor(a.ForegroundColor, b.ForegroundColor, t),
		OverlayColor:      lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		ShadowColor:       lerpStateOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor:  lerpStateOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Elevation:         lerpStateOpt(a.Elevation, b.Elevation, t, lerpOptDouble),
		Padding:           lerpStateNil(a.Padding, b.Padding, t, painting.EdgeInsetsGeometryLerp),
		MinimumSize:       lerpStateOpt(a.MinimumSize, b.MinimumSize, t, lerpOptSize),
		FixedSize:         lerpStateOpt(a.FixedSize, b.FixedSize, t, lerpOptSize),
		MaximumSize:       lerpStateOpt(a.MaximumSize, b.MaximumSize, t, lerpOptSize),
		IconColor:         lerpStateOptColor(a.IconColor, b.IconColor, t),
		IconSize:          lerpStateOpt(a.IconSize, b.IconSize, t, lerpOptDouble),
		IconAlignment:     lerpFirstOrSecond(a.IconAlignment, b.IconAlignment, t),
		Side:              lerpStateOpt(a.Side, b.Side, t, lerpOptBorderSide),
		Shape:             lerpStateNil(a.Shape, b.Shape, t, painting.OutlinedBorderLerp),
		MouseCursor:       lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		VisualDensity:     lerpFirstOrSecond(a.VisualDensity, b.VisualDensity, t),
		TapTargetSize:     lerpFirstOrSecond(a.TapTargetSize, b.TapTargetSize, t),
		AnimationDuration: lerpFirstOrSecond(a.AnimationDuration, b.AnimationDuration, t),
		EnableFeedback:    lerpFirstOrSecond(a.EnableFeedback, b.EnableFeedback, t),
		Alignment:         painting.AlignmentGeometryLerp(a.Alignment, b.Alignment, t),
		SplashFactory:     lerpFirstOrSecond(a.SplashFactory, b.SplashFactory, t),
		BackgroundBuilder: lerpFirstOrSecond(a.BackgroundBuilder, b.BackgroundBuilder, t),
		ForegroundBuilder: lerpFirstOrSecond(a.ForegroundBuilder, b.ForegroundBuilder, t),
	}
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (*ButtonStyle) ToStringShort() string { return "ButtonStyle" }

// String returns the description of the style.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (s *ButtonStyle) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the properties that are set.
//
// Flutter: material/button_style.dart ButtonStyle.debugFillProperties
func (s *ButtonStyle) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("textStyle", s.TextStyle))
	p.Add(objectProp("backgroundColor", s.BackgroundColor))
	p.Add(objectProp("foregroundColor", s.ForegroundColor))
	p.Add(objectProp("overlayColor", s.OverlayColor))
	p.Add(objectProp("shadowColor", s.ShadowColor))
	p.Add(objectProp("surfaceTintColor", s.SurfaceTintColor))
	p.Add(objectProp("elevation", s.Elevation))
	p.Add(objectProp("padding", s.Padding))
	p.Add(objectProp("minimumSize", s.MinimumSize))
	p.Add(objectProp("fixedSize", s.FixedSize))
	p.Add(objectProp("maximumSize", s.MaximumSize))
	p.Add(objectProp("iconColor", s.IconColor))
	p.Add(objectProp("iconSize", s.IconSize))
	p.Add(objectProp("iconAlignment", nilIfZero(s.IconAlignment)))
	p.Add(objectProp("side", s.Side))
	p.Add(objectProp("shape", s.Shape))
	p.Add(objectProp("mouseCursor", s.MouseCursor))
	p.Add(objectProp("visualDensity", optValue(s.VisualDensity)))
	p.Add(objectProp("tapTargetSize", nilIfZero(s.TapTargetSize)))
	p.Add(objectProp("animationDuration", optValue(s.AnimationDuration)))
	p.Add(objectProp("enableFeedback", optValue(s.EnableFeedback)))
	p.Add(objectProp("alignment", s.Alignment))
	p.Add(objectProp("splashFactory", s.SplashFactory))
	p.Add(objectProp("backgroundBuilder", s.BackgroundBuilder))
	p.Add(objectProp("foregroundBuilder", s.ForegroundBuilder))
}

// statePropertyEqual is Dart's == on two widget state properties: the ones that
// have the same value in all states, and the ones that are made of a map, are
// equal when their values are; the others when they are the same property.
func statePropertyEqual[T any](a, b widgets.WidgetStateProperty[T]) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch pa := a.(type) {
	case *widgets.WidgetStatePropertyAll[T]:
		if pb, ok := b.(*widgets.WidgetStatePropertyAll[T]); ok {
			return pa.Equal(pb)
		}
	case *widgets.WidgetStateMapper[T]:
		if pb, ok := b.(*widgets.WidgetStateMapper[T]); ok {
			return pa.Equal(pb)
		}
	}
	return foundation.ValuesEqual(a, b)
}
