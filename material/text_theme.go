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

// TextThemeOpts are the optional parameters of NewTextTheme and of
// TextTheme.CopyWith, where a nil style keeps the theme's.
//
// Flutter: material/text_theme.dart TextTheme
type TextThemeOpts struct {
	// DisplayLarge is the largest of the display styles.
	DisplayLarge *painting.TextStyle
	// DisplayMedium is the middle of the display styles.
	DisplayMedium *painting.TextStyle
	// DisplaySmall is the smallest of the display styles.
	DisplaySmall *painting.TextStyle
	// HeadlineLarge is the largest of the headline styles.
	HeadlineLarge *painting.TextStyle
	// HeadlineMedium is the middle of the headline styles.
	HeadlineMedium *painting.TextStyle
	// HeadlineSmall is the smallest of the headline styles.
	HeadlineSmall *painting.TextStyle
	// TitleLarge is the largest of the title styles.
	TitleLarge *painting.TextStyle
	// TitleMedium is the middle of the title styles.
	TitleMedium *painting.TextStyle
	// TitleSmall is the smallest of the title styles.
	TitleSmall *painting.TextStyle
	// BodyLarge is the largest of the body styles.
	BodyLarge *painting.TextStyle
	// BodyMedium is the middle of the body styles, the default text style.
	BodyMedium *painting.TextStyle
	// BodySmall is the smallest of the body styles.
	BodySmall *painting.TextStyle
	// LabelLarge is the largest of the label styles.
	LabelLarge *painting.TextStyle
	// LabelMedium is the middle of the label styles.
	LabelMedium *painting.TextStyle
	// LabelSmall is the smallest of the label styles.
	LabelSmall *painting.TextStyle
}

// TextTheme is the Material Design text styles of the type scale: display,
// headline, title, body and label, each in a large, a medium and a small size.
// A style that is nil is null in Flutter. A TextTheme is immutable; create it
// with NewTextTheme.
//
// Flutter: material/text_theme.dart TextTheme
type TextTheme struct {
	// DisplayLarge is the largest of the display styles.
	DisplayLarge *painting.TextStyle
	// DisplayMedium is the middle of the display styles.
	DisplayMedium *painting.TextStyle
	// DisplaySmall is the smallest of the display styles.
	DisplaySmall *painting.TextStyle
	// HeadlineLarge is the largest of the headline styles.
	HeadlineLarge *painting.TextStyle
	// HeadlineMedium is the middle of the headline styles.
	HeadlineMedium *painting.TextStyle
	// HeadlineSmall is the smallest of the headline styles.
	HeadlineSmall *painting.TextStyle
	// TitleLarge is the largest of the title styles.
	TitleLarge *painting.TextStyle
	// TitleMedium is the middle of the title styles.
	TitleMedium *painting.TextStyle
	// TitleSmall is the smallest of the title styles.
	TitleSmall *painting.TextStyle
	// BodyLarge is the largest of the body styles.
	BodyLarge *painting.TextStyle
	// BodyMedium is the middle of the body styles, the default text style.
	BodyMedium *painting.TextStyle
	// BodySmall is the smallest of the body styles.
	BodySmall *painting.TextStyle
	// LabelLarge is the largest of the label styles.
	LabelLarge *painting.TextStyle
	// LabelMedium is the middle of the label styles.
	LabelMedium *painting.TextStyle
	// LabelSmall is the smallest of the label styles.
	LabelSmall *painting.TextStyle
}

// NewTextTheme creates a text theme from the styles of opts.
//
// Flutter: material/text_theme.dart TextTheme.new
func NewTextTheme(opts ...TextThemeOpts) *TextTheme {
	o := firstOpts(opts)
	return &TextTheme{
		DisplayLarge: o.DisplayLarge, DisplayMedium: o.DisplayMedium, DisplaySmall: o.DisplaySmall,
		HeadlineLarge: o.HeadlineLarge, HeadlineMedium: o.HeadlineMedium, HeadlineSmall: o.HeadlineSmall,
		TitleLarge: o.TitleLarge, TitleMedium: o.TitleMedium, TitleSmall: o.TitleSmall,
		BodyLarge: o.BodyLarge, BodyMedium: o.BodyMedium, BodySmall: o.BodySmall,
		LabelLarge: o.LabelLarge, LabelMedium: o.LabelMedium, LabelSmall: o.LabelSmall,
	}
}

// textThemeFields lists the styles of a theme in Flutter's declaration order,
// with the name of each and whether it takes the display color of
// TextTheme.Apply (as opposed to the body color).
var textThemeFields = [15]struct {
	name    string
	display bool
	get     func(t *TextTheme) **painting.TextStyle
}{
	{"displayLarge", true, func(t *TextTheme) **painting.TextStyle { return &t.DisplayLarge }},
	{"displayMedium", true, func(t *TextTheme) **painting.TextStyle { return &t.DisplayMedium }},
	{"displaySmall", true, func(t *TextTheme) **painting.TextStyle { return &t.DisplaySmall }},
	{"headlineLarge", true, func(t *TextTheme) **painting.TextStyle { return &t.HeadlineLarge }},
	{"headlineMedium", true, func(t *TextTheme) **painting.TextStyle { return &t.HeadlineMedium }},
	{"headlineSmall", false, func(t *TextTheme) **painting.TextStyle { return &t.HeadlineSmall }},
	{"titleLarge", false, func(t *TextTheme) **painting.TextStyle { return &t.TitleLarge }},
	{"titleMedium", false, func(t *TextTheme) **painting.TextStyle { return &t.TitleMedium }},
	{"titleSmall", false, func(t *TextTheme) **painting.TextStyle { return &t.TitleSmall }},
	{"bodyLarge", false, func(t *TextTheme) **painting.TextStyle { return &t.BodyLarge }},
	{"bodyMedium", false, func(t *TextTheme) **painting.TextStyle { return &t.BodyMedium }},
	{"bodySmall", true, func(t *TextTheme) **painting.TextStyle { return &t.BodySmall }},
	{"labelLarge", false, func(t *TextTheme) **painting.TextStyle { return &t.LabelLarge }},
	{"labelMedium", false, func(t *TextTheme) **painting.TextStyle { return &t.LabelMedium }},
	{"labelSmall", false, func(t *TextTheme) **painting.TextStyle { return &t.LabelSmall }},
}

// CopyWith creates a copy of this text theme but with the styles of c that are
// not nil replaced.
//
// Flutter: material/text_theme.dart TextTheme.copyWith
func (t *TextTheme) CopyWith(c TextThemeOpts) *TextTheme {
	out := NewTextTheme(TextThemeOpts(*t))
	o := &TextTheme{}
	*o = TextTheme(c)
	for _, f := range textThemeFields {
		if s := *f.get(o); s != nil {
			*f.get(out) = s
		}
	}
	return out
}

// Merge creates a copy of this text theme but with each style merged with the
// matching style of other. A nil other returns this theme.
//
// Flutter: material/text_theme.dart TextTheme.merge
func (t *TextTheme) Merge(other *TextTheme) *TextTheme {
	if other == nil {
		return t
	}
	out := NewTextTheme(TextThemeOpts(*t))
	for _, f := range textThemeFields {
		mine, theirs := *f.get(t), *f.get(other)
		if mine != nil {
			*f.get(out) = mine.Merge(theirs)
		} else {
			*f.get(out) = theirs
		}
	}
	return out
}

// TextThemeApply are the parameters of TextTheme.Apply. A factor defaults to 1
// and a delta to 0.
//
// Flutter: material/text_theme.dart TextTheme.apply
type TextThemeApply struct {
	// FontFamily replaces the font family of every style.
	FontFamily string
	// FontFamilyFallback replaces the font family fallback of every style.
	FontFamilyFallback []string
	// Package replaces the package of every style.
	Package string
	// FontSizeFactor multiplies the font size of every style.
	FontSizeFactor foundation.Opt[float64]
	// FontSizeDelta is added to the font size of every style.
	FontSizeDelta float64
	// LetterSpacingFactor multiplies the letter spacing of every style.
	LetterSpacingFactor foundation.Opt[float64]
	// LetterSpacingDelta is added to the letter spacing of every style.
	LetterSpacingDelta float64
	// WordSpacingFactor multiplies the word spacing of every style.
	WordSpacingFactor foundation.Opt[float64]
	// WordSpacingDelta is added to the word spacing of every style.
	WordSpacingDelta float64
	// HeightFactor multiplies the height of every style.
	HeightFactor foundation.Opt[float64]
	// HeightDelta is added to the height of every style.
	HeightDelta float64
	// DisplayColor is the color of the display styles, headlineLarge,
	// headlineMedium and bodySmall.
	DisplayColor foundation.Opt[ui.Color]
	// BodyColor is the color of the other styles.
	BodyColor foundation.Opt[ui.Color]
	// Decoration replaces the decoration of every style.
	Decoration foundation.Opt[ui.TextDecoration]
	// DecorationColor replaces the decoration color of every style.
	DecorationColor foundation.Opt[ui.Color]
	// DecorationStyle replaces the decoration style of every style.
	DecorationStyle ui.TextDecorationStyle
}

// Apply creates a copy of this text theme with every style modified by a. The
// display color goes to the display styles, headlineLarge, headlineMedium and
// bodySmall; the body color goes to the others.
//
// Flutter: material/text_theme.dart TextTheme.apply
func (t *TextTheme) Apply(a TextThemeApply) *TextTheme {
	out := &TextTheme{}
	for _, f := range textThemeFields {
		s := *f.get(t)
		if s == nil {
			continue
		}
		color := a.BodyColor
		if f.display {
			color = a.DisplayColor
		}
		*f.get(out) = s.Apply(painting.TextStyleApply{
			Color:               color,
			Decoration:          a.Decoration,
			DecorationColor:     a.DecorationColor,
			DecorationStyle:     a.DecorationStyle,
			FontFamily:          a.FontFamily,
			FontFamilyFallback:  a.FontFamilyFallback,
			FontSizeFactor:      a.FontSizeFactor,
			FontSizeDelta:       a.FontSizeDelta,
			LetterSpacingFactor: a.LetterSpacingFactor,
			LetterSpacingDelta:  a.LetterSpacingDelta,
			WordSpacingFactor:   a.WordSpacingFactor,
			WordSpacingDelta:    a.WordSpacingDelta,
			HeightFactor:        a.HeightFactor,
			HeightDelta:         a.HeightDelta,
			Package:             a.Package,
		})
	}
	return out
}

// TextThemeLerp linearly interpolates between two text themes, style by style.
// Either may be nil.
//
// Flutter: material/text_theme.dart TextTheme.lerp
func TextThemeLerp(a, b *TextTheme, t float64) *TextTheme {
	if a == b && a != nil {
		return a
	}
	out := &TextTheme{}
	for _, f := range textThemeFields {
		var sa, sb *painting.TextStyle
		if a != nil {
			sa = *f.get(a)
		}
		if b != nil {
			sb = *f.get(b)
		}
		*f.get(out) = painting.TextStyleLerp(sa, sb, t)
	}
	return out
}

// TextThemeOf returns the text theme of the closest Theme ancestor of bc.
//
// Flutter: material/text_theme.dart TextTheme.of
func TextThemeOf(bc widgets.BuildContext) *TextTheme { return ThemeOf(bc).TextTheme }

// TextThemePrimaryOf returns the primary text theme of the closest Theme
// ancestor of bc.
//
// Flutter: material/text_theme.dart TextTheme.primaryOf
func TextThemePrimaryOf(bc widgets.BuildContext) *TextTheme {
	return ThemeOf(bc).PrimaryTextTheme
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/text_theme.dart TextTheme.==
func (t *TextTheme) Equal(other *TextTheme) bool {
	if t == other {
		return true
	}
	if t == nil || other == nil {
		return false
	}
	for _, f := range textThemeFields {
		if !textStyleEqual(*f.get(t), *f.get(other)) {
			return false
		}
	}
	return true
}

func textStyleEqual(a, b *painting.TextStyle) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(b)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (t *TextTheme) ToStringShort() string { return "TextTheme" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (t *TextTheme) String() string { return foundation.DiagnosticableString(t) }

// DebugFillProperties adds the styles that differ from the default theme of
// Typography.material2018 for the target platform.
//
// Flutter: material/text_theme.dart TextTheme.debugFillProperties
func (t *TextTheme) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	def := NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(foundation.DefaultTargetPlatform())}).Black
	for _, f := range textThemeFields {
		properties.Add(foundation.NewDiagnosticsPropertyOpt(f.name, styleOpt(*f.get(t)),
			foundation.DiagnosticsPropertyOpts{DefaultValue: *f.get(def)}))
	}
}

func styleOpt(s *painting.TextStyle) foundation.Opt[*painting.TextStyle] {
	if s == nil {
		return foundation.None[*painting.TextStyle]()
	}
	return foundation.Some(s)
}
