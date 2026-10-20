// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// IconData is a description of an icon fulfilled by a font glyph. It is
// comparable. The zero IconData stands for Dart's null icon.
//
// Flutter: widgets/icon_data.dart IconData
type IconData struct {
	// CodePoint is the Unicode code point of the glyph in the font.
	//
	// Flutter: widgets/icon_data.dart IconData.codePoint
	CodePoint int
	// FontFamily is the font family of the glyph; empty for the default.
	//
	// Flutter: widgets/icon_data.dart IconData.fontFamily
	FontFamily string
	// FontPackage is the package that holds the font, or empty.
	//
	// Flutter: widgets/icon_data.dart IconData.fontPackage
	FontPackage string
	// MatchTextDirection is whether the glyph is mirrored in right to left text.
	//
	// Flutter: widgets/icon_data.dart IconData.matchTextDirection
	MatchTextDirection bool

	fontFamilyFallback string
}

// IconDataOpts are the optional parameters of NewIconData.
//
// Flutter: widgets/icon_data.dart IconData.new
type IconDataOpts struct {
	// FontFamily is the font family of the glyph.
	FontFamily string
	// FontPackage is the package that holds the font.
	FontPackage string
	// MatchTextDirection is whether the glyph is mirrored in right to left text.
	MatchTextDirection bool
	// FontFamilyFallback are the families to try when the glyph is not in the
	// font family.
	FontFamilyFallback []string
}

// NewIconData creates an icon for the code point.
//
// Flutter: widgets/icon_data.dart IconData.new
func NewIconData(codePoint int, opts ...IconDataOpts) IconData {
	o := firstOpts(opts)
	return IconData{CodePoint: codePoint, FontFamily: o.FontFamily, FontPackage: o.FontPackage,
		MatchTextDirection: o.MatchTextDirection, fontFamilyFallback: strings.Join(o.FontFamilyFallback, "\x00")}
}

// FontFamilyFallback are the families to try when the glyph is not in the font
// family.
//
// Flutter: widgets/icon_data.dart IconData.fontFamilyFallback
func (d IconData) FontFamilyFallback() []string {
	if d.fontFamilyFallback == "" {
		return nil
	}
	return strings.Split(d.fontFamilyFallback, "\x00")
}

// IsZero is whether this is the zero IconData, which stands for no icon.
//
// Flutter: no counterpart
func (d IconData) IsZero() bool { return d == IconData{} }

// String describes the icon by its code point.
//
// Flutter: widgets/icon_data.dart IconData.toString
func (d IconData) String() string { return fmt.Sprintf("IconData(U+%05X)", d.CodePoint) }

// IconThemeData defines the size, font variations, color, opacity and shadows of
// icons. Any property can be unset, and it then comes from an ancestor.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData
type IconThemeData struct {
	size             foundation.Opt[float64]
	fill             foundation.Opt[float64]
	weight           foundation.Opt[float64]
	grade            foundation.Opt[float64]
	opticalSize      foundation.Opt[float64]
	color            foundation.Opt[ui.Color]
	opacity          foundation.Opt[float64]
	shadows          []ui.Shadow
	applyTextScaling foundation.Opt[bool]
}

// IconThemeDataOpts are the optional parameters of NewIconThemeData and the
// fields that CopyWith replaces.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.new
type IconThemeDataOpts struct {
	// Size is the size of icons in logical pixels.
	Size foundation.Opt[float64]
	// Fill is the fill of the glyph, from 0 to 1.
	Fill foundation.Opt[float64]
	// Weight is the stroke weight of the glyph; it must be greater than zero.
	Weight foundation.Opt[float64]
	// Grade is the grade of the glyph.
	Grade foundation.Opt[float64]
	// OpticalSize is the optical size of the glyph; it must be greater than zero.
	OpticalSize foundation.Opt[float64]
	// Color is the default color of icons.
	Color foundation.Opt[ui.Color]
	// Opacity is the opacity applied to the color, clamped to 0 to 1.
	Opacity foundation.Opt[float64]
	// Shadows are the shadows painted behind icons.
	Shadows []ui.Shadow
	// ApplyTextScaling is whether the size is scaled with the text scaler.
	ApplyTextScaling foundation.Opt[bool]
}

// NewIconThemeData creates the data. In debug builds it panics with the tag
// IconThemeDataInvalid when the fill is outside 0 to 1 or the weight or the
// optical size is not greater than zero.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.new
func NewIconThemeData(opts ...IconThemeDataOpts) IconThemeData {
	o := firstOpts(opts)
	assertIconFont("IconThemeDataInvalid", o.Fill, o.Weight, o.OpticalSize)
	return IconThemeData{o.Size, o.Fill, o.Weight, o.Grade, o.OpticalSize, o.Color, o.Opacity, o.Shadows, o.ApplyTextScaling}
}

// NewIconThemeDataFallback creates the data that is used when there is no
// theme: 24 pixels, black, fully opaque, with the default font variations.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.fallback
func NewIconThemeDataFallback() IconThemeData {
	return IconThemeData{foundation.Some(24.0), foundation.Some(0.0), foundation.Some(400.0), foundation.Some(0.0), foundation.Some(48.0),
		foundation.Some(ui.NewColor(0xFF000000)), foundation.Some(1.0), nil, foundation.Some(false)}
}

func assertIconFont(tag string, fill, weight, opticalSize foundation.Opt[float64]) {
	if !foundation.DebugMode {
		return
	}
	if v, ok := fill.Get(); ok && (v < 0 || v > 1) {
		panic(errors.New(tag, "the fill must be from 0 to 1", "fill", v))
	}
	if v, ok := weight.Get(); ok && v <= 0 {
		panic(errors.New(tag, "the weight must be greater than zero", "weight", v))
	}
	if v, ok := opticalSize.Get(); ok && v <= 0 {
		panic(errors.New(tag, "the optical size must be greater than zero", "opticalSize", v))
	}
}

// Size is the size of icons.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.size
func (d IconThemeData) Size() foundation.Opt[float64] { return d.size }

// Fill is the fill of the glyph.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.fill
func (d IconThemeData) Fill() foundation.Opt[float64] { return d.fill }

// Weight is the stroke weight of the glyph.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.weight
func (d IconThemeData) Weight() foundation.Opt[float64] { return d.weight }

// Grade is the grade of the glyph.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.grade
func (d IconThemeData) Grade() foundation.Opt[float64] { return d.grade }

// OpticalSize is the optical size of the glyph.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.opticalSize
func (d IconThemeData) OpticalSize() foundation.Opt[float64] { return d.opticalSize }

// Color is the default color of icons.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.color
func (d IconThemeData) Color() foundation.Opt[ui.Color] { return d.color }

// Opacity is the opacity applied to the color, clamped to 0 to 1.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.opacity
func (d IconThemeData) Opacity() foundation.Opt[float64] {
	if v, ok := d.opacity.Get(); ok {
		return foundation.Some(ui.ClampDouble(v, 0, 1))
	}
	return d.opacity
}

// Shadows are the shadows painted behind icons.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.shadows
func (d IconThemeData) Shadows() []ui.Shadow { return d.shadows }

// ApplyTextScaling is whether the size is scaled with the text scaler.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.applyTextScaling
func (d IconThemeData) ApplyTextScaling() foundation.Opt[bool] { return d.applyTextScaling }

func orSome[T any](a, b foundation.Opt[T]) foundation.Opt[T] {
	if a.IsSome() {
		return a
	}
	return b
}

// CopyWith is a copy of the data in which the set properties are replaced.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.copyWith
func (d IconThemeData) CopyWith(o IconThemeDataOpts) IconThemeData {
	shadows := d.shadows
	if o.Shadows != nil {
		shadows = o.Shadows
	}
	return NewIconThemeData(IconThemeDataOpts{Size: orSome(o.Size, d.size), Fill: orSome(o.Fill, d.fill), Weight: orSome(o.Weight, d.weight),
		Grade: orSome(o.Grade, d.grade), OpticalSize: orSome(o.OpticalSize, d.opticalSize), Color: orSome(o.Color, d.color),
		Opacity: orSome(o.Opacity, d.opacity), Shadows: shadows, ApplyTextScaling: orSome(o.ApplyTextScaling, d.applyTextScaling)})
}

// Merge is a copy of the data in which the properties that other sets replace
// the ones here.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.merge
func (d IconThemeData) Merge(other *IconThemeData) IconThemeData {
	if other == nil {
		return d
	}
	return d.CopyWith(IconThemeDataOpts{Size: other.size, Fill: other.fill, Weight: other.weight, Grade: other.grade,
		OpticalSize: other.opticalSize, Color: other.color, Opacity: other.Opacity(), Shadows: other.shadows, ApplyTextScaling: other.applyTextScaling})
}

// Resolve is the data itself; subclasses in Flutter use the context to resolve
// theme dependent values.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.resolve
func (d IconThemeData) Resolve(BuildContext) IconThemeData { return d }

// IsConcrete is whether every property is set (the shadows may be absent).
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.isConcrete
func (d IconThemeData) IsConcrete() bool {
	return d.size.IsSome() && d.fill.IsSome() && d.weight.IsSome() && d.grade.IsSome() && d.opticalSize.IsSome() &&
		d.color.IsSome() && d.opacity.IsSome() && d.applyTextScaling.IsSome()
}

// Equal is Dart's == on icon theme data.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.operator ==
func (d IconThemeData) Equal(o IconThemeData) bool {
	return d.size == o.size && d.fill == o.fill && d.weight == o.weight && d.grade == o.grade && d.opticalSize == o.opticalSize &&
		d.color == o.color && d.Opacity() == o.Opacity() && slices.Equal(d.shadows, o.shadows) && d.applyTextScaling == o.applyTextScaling
}

func lerpOptDouble(a, b foundation.Opt[float64], t float64) foundation.Opt[float64] {
	var pa, pb *float64
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	if r := ui.LerpDoubleOpt(pa, pb, t); r != nil {
		return foundation.Some(*r)
	}
	return foundation.None[float64]()
}

// IconThemeDataLerp linearly interpolates between two icon themes. Either may be
// nil.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.lerp
func IconThemeDataLerp(a, b *IconThemeData, t float64) IconThemeData {
	var za, zb IconThemeData
	if a == nil {
		a = &za
	}
	if b == nil {
		b = &zb
	}
	var ca, cb *ui.Color
	if v, ok := a.color.Get(); ok {
		ca = &v
	}
	if v, ok := b.color.Get(); ok {
		cb = &v
	}
	color := foundation.None[ui.Color]()
	if c := ui.ColorLerp(ca, cb, t); c != nil {
		color = foundation.Some(*c)
	}
	scaling := a.applyTextScaling
	if t >= 0.5 {
		scaling = b.applyTextScaling
	}
	return NewIconThemeData(IconThemeDataOpts{Size: lerpOptDouble(a.size, b.size, t), Fill: lerpOptDouble(a.fill, b.fill, t),
		Weight: lerpOptDouble(a.weight, b.weight, t), Grade: lerpOptDouble(a.grade, b.grade, t),
		OpticalSize: lerpOptDouble(a.opticalSize, b.opticalSize, t), Color: color,
		Opacity: lerpOptDouble(a.Opacity(), b.Opacity(), t), Shadows: ui.ShadowLerpList(a.shadows, b.shadows, t), ApplyTextScaling: scaling})
}

// DebugFillProperties adds the properties that are set.
//
// Flutter: widgets/icon_theme_data.dart IconThemeData.debugFillProperties
func (d IconThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	for _, e := range []struct {
		name string
		v    foundation.Opt[float64]
	}{{"size", d.size}, {"fill", d.fill}, {"weight", d.weight}, {"grade", d.grade}, {"opticalSize", d.opticalSize}, {"opacity", d.Opacity()}} {
		p.Add(foundation.NewDoubleProperty(e.name, e.v, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
	}
}

// IconTheme controls the default properties of icons in a widget subtree.
//
// Flutter: widgets/icon_theme.dart IconTheme
type IconTheme struct {
	InheritedTheme
	data IconThemeData
}

// NewIconTheme creates the widget.
//
// Flutter: widgets/icon_theme.dart IconTheme.new
func NewIconTheme(data IconThemeData, child Widget, opts ...WidgetOpts) *IconTheme {
	return &IconTheme{NewInheritedThemeBase(child, opts...), data}
}

// NewIconThemeMerge creates a widget that merges the data with the icon theme of
// the closest ancestor.
//
// Flutter: widgets/icon_theme.dart IconTheme.merge
func NewIconThemeMerge(data IconThemeData, child Widget, opts ...WidgetOpts) Widget {
	return NewBuilder(func(bc BuildContext) Widget {
		return NewIconTheme(inheritedIconThemeData(bc).Merge(&data), child, opts...)
	})
}

// Data is the properties for descendant icons.
//
// Flutter: widgets/icon_theme.dart IconTheme.data
func (w *IconTheme) Data() IconThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: widgets/icon_theme.dart IconTheme.updateShouldNotify
func (w *IconTheme) UpdateShouldNotify(old Widget) bool { return !w.data.Equal(old.(*IconTheme).data) }

// Wrap makes an IconTheme with the same data around the child.
//
// Flutter: widgets/icon_theme.dart IconTheme.wrap
func (w *IconTheme) Wrap(_ BuildContext, child Widget) Widget { return NewIconTheme(w.data, child) }

// DebugFillProperties adds the data.
//
// Flutter: widgets/icon_theme.dart IconTheme.debugFillProperties
func (w *IconTheme) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	w.data.DebugFillProperties(p)
}

func inheritedIconThemeData(bc BuildContext) IconThemeData {
	if t, ok := DependOn[*IconTheme](bc); ok {
		return t.data
	}
	return NewIconThemeDataFallback()
}

// IconThemeOf is the icon theme of the closest IconTheme above the context, with
// the properties that it leaves out taken from the fallback. The context
// depends on it.
//
// Flutter: widgets/icon_theme.dart IconTheme.of
func IconThemeOf(bc BuildContext) IconThemeData {
	d := inheritedIconThemeData(bc).Resolve(bc)
	if d.IsConcrete() {
		return d
	}
	f := NewIconThemeDataFallback()
	return NewIconThemeData(IconThemeDataOpts{Size: orSome(d.size, f.size), Fill: orSome(d.fill, f.fill), Weight: orSome(d.weight, f.weight),
		Grade: orSome(d.grade, f.grade), OpticalSize: orSome(d.opticalSize, f.opticalSize), Color: orSome(d.color, f.color),
		Opacity: orSome(d.opacity, f.opacity), Shadows: d.shadows, ApplyTextScaling: orSome(d.applyTextScaling, f.applyTextScaling)})
}

// IconOpts are the optional parameters of NewIcon.
//
// Flutter: widgets/icon.dart Icon.new
type IconOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Size is the size of the icon in logical pixels; the default comes from the
	// icon theme.
	Size foundation.Opt[float64]
	// Fill is the fill of the glyph, from 0 to 1.
	Fill foundation.Opt[float64]
	// Weight is the stroke weight of the glyph; it must be greater than zero.
	Weight foundation.Opt[float64]
	// Grade is the grade of the glyph.
	Grade foundation.Opt[float64]
	// OpticalSize is the optical size of the glyph; it must be greater than zero.
	OpticalSize foundation.Opt[float64]
	// Color is the color of the icon; the default comes from the icon theme.
	Color foundation.Opt[ui.Color]
	// Shadows are the shadows painted behind the icon.
	Shadows []ui.Shadow
	// SemanticLabel is the label that describes the icon in the semantics tree.
	SemanticLabel foundation.Opt[string]
	// TextDirection is the direction of the icon; the default is the one of the
	// closest Directionality.
	TextDirection ui.TextDirection
	// ApplyTextScaling is whether the size is scaled with the text scaler.
	ApplyTextScaling foundation.Opt[bool]
	// BlendMode is the blend mode of the glyph when it is painted.
	BlendMode ui.BlendMode
	// FontWeight is the weight of the font used for the glyph.
	FontWeight ui.FontWeight
}

// Icon is a graphical icon widget drawn with a glyph from a font.
//
// Flutter: widgets/icon.dart Icon
type Icon struct {
	StatelessWidget
	icon IconData
	o    IconOpts
}

// NewIcon creates an icon. The zero IconData makes an empty box of the icon's
// size. In debug builds it panics with the tag IconInvalid when the fill is
// outside 0 to 1 or the weight or the optical size is not greater than zero.
//
// Flutter: widgets/icon.dart Icon.new
func NewIcon(icon IconData, opts ...IconOpts) *Icon {
	o := firstOpts(opts)
	assertIconFont("IconInvalid", o.Fill, o.Weight, o.OpticalSize)
	return &Icon{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), icon, o}
}

// Icon is the icon to display.
//
// Flutter: widgets/icon.dart Icon.icon
func (w *Icon) Icon() IconData { return w.icon }

// IconValue is the icon, for the finders of gluttertest.
//
// Flutter: no counterpart (used by finders)
func (w *Icon) IconValue() any { return w.icon }

// SemanticLabel is the semantics label of the icon.
//
// Flutter: widgets/icon.dart Icon.semanticLabel
func (w *Icon) SemanticLabel() foundation.Opt[string] { return w.o.SemanticLabel }

// Build makes the box with the glyph.
//
// Flutter: widgets/icon.dart Icon.build
func (w *Icon) Build(bc BuildContext) Widget {
	o := w.o
	textDirection := requiredDirection(o.TextDirection, bc)
	iconTheme := IconThemeOf(bc)
	applyScaling := o.ApplyTextScaling.OrElse(iconTheme.applyTextScaling.OrElse(false))
	size := o.Size.OrElse(iconTheme.size.OrElse(painting.DefaultFontSize))
	if applyScaling {
		size = MediaQueryTextScalerOf(bc).Scale(size)
	}
	label := o.SemanticLabel
	if w.icon.IsZero() {
		return NewSemantics(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(size), Height: foundation.Some(size)}), SemanticsOpts{Label: label})
	}
	opacity := iconTheme.Opacity().OrElse(1)
	color := o.Color.OrElse(iconTheme.color.OrElse(ui.NewColor(0xFF000000)))
	var foreground *ui.Paint
	colorOpt := foundation.Some(color)
	if opacity != 1 {
		color = color.WithOpacity(color.Opacity() * opacity)
		colorOpt = foundation.Some(color)
	}
	if o.BlendMode != ui.BlendModeUnspecified {
		foreground = ui.NewPaint()
		foreground.SetBlendMode(o.BlendMode)
		foreground.SetColor(color)
		colorOpt = foundation.None[ui.Color]()
	}
	var variations []ui.FontVariation
	for _, v := range []struct {
		axis string
		v    foundation.Opt[float64]
	}{{"FILL", orSome(o.Fill, iconTheme.fill)}, {"wght", orSome(o.Weight, iconTheme.weight)},
		{"GRAD", orSome(o.Grade, iconTheme.grade)}, {"opsz", orSome(o.OpticalSize, iconTheme.opticalSize)}} {
		if x, ok := v.v.Get(); ok {
			variations = append(variations, ui.NewFontVariation(v.axis, x))
		}
	}
	shadows := o.Shadows
	if shadows == nil {
		shadows = iconTheme.shadows
	}
	style := painting.NewTextStyle(painting.TextStyleOpts{
		FontVariations: variations, Inherit: foundation.Some(false), Color: colorOpt, FontSize: foundation.Some(size),
		FontFamily: w.icon.FontFamily, FontWeight: o.FontWeight, Package: w.icon.FontPackage,
		FontFamilyFallback: w.icon.FontFamilyFallback(), Shadows: shadows, Height: foundation.Some(1.0),
		LeadingDistribution: ui.TextLeadingDistributionEven, Foreground: foreground,
	})
	var glyph Widget = NewRichText(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(string(rune(w.icon.CodePoint))), Style: style}),
		RichTextOpts{Overflow: painting.TextOverflowVisible, TextDirection: textDirection})
	if w.icon.MatchTextDirection && textDirection == ui.TextDirectionRTL {
		m := ui.NewMatrix4Identity()
		m.Scale(-1, 1, 1)
		glyph = NewTransform(m, glyph, TransformOpts{Alignment: painting.AlignmentCenter, TransformHitTests: foundation.Some(false)})
	}
	return NewSemantics(NewExcludeSemantics(NewSizedBox(NewCenter(glyph), SizedBoxOpts{Width: foundation.Some(size), Height: foundation.Some(size)})), SemanticsOpts{Label: label})
}
