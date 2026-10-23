package painting

import (
	"slices"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

const defaultDebugLabel = "unknown"

// TextStyleOpts are the optional parameters of NewTextStyle (and of
// TextStyle.CopyWith, where an unset value keeps the style's).
//
// Flutter: painting/text_style.dart TextStyle
type TextStyleOpts struct {
	// Inherit says whether null fields inherit from the enclosing style; it
	// defaults to true.
	Inherit foundation.Opt[bool]
	// Color is the color to use when painting the text; it must not be given
	// together with Foreground.
	Color foundation.Opt[ui.Color]
	// BackgroundColor is the color to use as the background of the text; it must
	// not be given together with Background.
	BackgroundColor foundation.Opt[ui.Color]
	// FontSize is the size of the glyphs in logical pixels.
	FontSize foundation.Opt[float64]
	// FontWeight is the typeface thickness.
	FontWeight ui.FontWeight
	// FontStyle is the typeface variant.
	FontStyle ui.FontStyle
	// LetterSpacing is the space added between letters, in logical pixels.
	LetterSpacing foundation.Opt[float64]
	// WordSpacing is the space added at each space, in logical pixels.
	WordSpacing foundation.Opt[float64]
	// TextBaseline is the baseline the text aligns on.
	TextBaseline ui.TextBaseline
	// Height is the height of the line as a multiple of the font size.
	Height foundation.Opt[float64]
	// LeadingDistribution is how the extra line height is divided.
	LeadingDistribution ui.TextLeadingDistribution
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// Foreground is the paint of the text, instead of Color.
	Foreground *ui.Paint
	// Background is the paint of the box behind the text, instead of
	// BackgroundColor.
	Background *ui.Paint
	// Shadows are painted underneath the text; nil is null.
	Shadows []ui.Shadow
	// FontFeatures select glyph variants of the font; nil is null.
	FontFeatures []ui.FontFeature
	// FontVariations are the axis values of a variable font; nil is null.
	FontVariations []ui.FontVariation
	// Decoration is the decoration to paint near the text.
	Decoration foundation.Opt[ui.TextDecoration]
	// DecorationColor is the color of the decoration.
	DecorationColor foundation.Opt[ui.Color]
	// DecorationStyle is the style of the decoration line.
	DecorationStyle ui.TextDecorationStyle
	// DecorationThickness is the thickness of the decoration as a multiple of the
	// font's default.
	DecorationThickness foundation.Opt[float64]
	// DebugLabel is a human-readable description of the style.
	DebugLabel string
	// FontFamily is the name of the font family.
	FontFamily string
	// FontFamilyFallback is the ordered list of families to fall back on; nil is
	// null.
	FontFamilyFallback []string
	// Package is the package that holds the font; the family names are prefixed
	// with "packages/<package>/".
	Package string
	// Overflow is how visual overflow is handled.
	Overflow TextOverflow
}

// TextStyleCopy are the parameters of TextStyle.CopyWith.
//
// Flutter: painting/text_style.dart TextStyle.copyWith
type TextStyleCopy = TextStyleOpts

// TextStyle is an immutable style describing how to format and paint text.
// Create it with NewTextStyle; its fields are for reading.
//
// Flutter: painting/text_style.dart TextStyle
type TextStyle struct {
	// Inherit says whether null fields inherit from the enclosing style.
	Inherit bool
	// Color is the color of the text.
	Color foundation.Opt[ui.Color]
	// BackgroundColor is the color of the box behind the text.
	BackgroundColor foundation.Opt[ui.Color]
	// FontFamily is the name of the font family, with any package prefix.
	FontFamily string
	// FontSize is the size of the glyphs in logical pixels.
	FontSize foundation.Opt[float64]
	// FontWeight is the typeface thickness; 0 is null.
	FontWeight ui.FontWeight
	// FontStyle is the typeface variant; unspecified is null.
	FontStyle ui.FontStyle
	// LetterSpacing is the space added between letters.
	LetterSpacing foundation.Opt[float64]
	// WordSpacing is the space added at each space.
	WordSpacing foundation.Opt[float64]
	// TextBaseline is the baseline the text aligns on; unspecified is null.
	TextBaseline ui.TextBaseline
	// Height is the height of the line as a multiple of the font size.
	Height foundation.Opt[float64]
	// LeadingDistribution is how the extra line height is divided; unspecified
	// is null.
	LeadingDistribution ui.TextLeadingDistribution
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// Foreground is the paint of the text, instead of Color.
	Foreground *ui.Paint
	// Background is the paint of the box behind the text.
	Background *ui.Paint
	// Decoration is the decoration to paint near the text.
	Decoration foundation.Opt[ui.TextDecoration]
	// DecorationColor is the color of the decoration.
	DecorationColor foundation.Opt[ui.Color]
	// DecorationStyle is the style of the decoration line; unspecified is null.
	DecorationStyle ui.TextDecorationStyle
	// DecorationThickness is the thickness of the decoration.
	DecorationThickness foundation.Opt[float64]
	// DebugLabel is a human-readable description of the style.
	DebugLabel string
	// Shadows are painted underneath the text; nil is null.
	Shadows []ui.Shadow
	// FontFeatures select glyph variants of the font; nil is null.
	FontFeatures []ui.FontFeature
	// FontVariations are the axis values of a variable font; nil is null.
	FontVariations []ui.FontVariation
	// Overflow is how visual overflow is handled; unspecified is null.
	Overflow TextOverflow

	fontFamilyFallback []string
	pkg                string
}

// NewTextStyle creates a text style. In debug builds giving both a Color and a
// Foreground panics with the tag TextStyleColorAndForeground, and both a
// BackgroundColor and a Background with TextStyleBackgroundAndBackgroundColor.
//
// Flutter: painting/text_style.dart TextStyle.new
func NewTextStyle(opts ...TextStyleOpts) *TextStyle {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Color.IsSome() && o.Foreground != nil {
			panic(errors.New(tagTextStyleColorAndForeground, "Cannot provide both a color and a foreground\n"+
				`The color argument is just a shorthand for "foreground: Paint()..color = color".`))
		}
		if o.BackgroundColor.IsSome() && o.Background != nil {
			panic(errors.New(tagTextStyleBackgroundAndBackgroundColor, "Cannot provide both a backgroundColor and a background\n"+
				`The backgroundColor argument is just a shorthand for "background: Paint()..color = color".`))
		}
	}
	family := o.FontFamily
	if o.Package != "" {
		family = "packages/" + o.Package + "/" + o.FontFamily
	}
	return &TextStyle{
		Inherit: o.Inherit.OrElse(true), Color: o.Color, BackgroundColor: o.BackgroundColor, FontFamily: family,
		FontSize: o.FontSize, FontWeight: o.FontWeight, FontStyle: o.FontStyle, LetterSpacing: o.LetterSpacing,
		WordSpacing: o.WordSpacing, TextBaseline: o.TextBaseline, Height: o.Height,
		LeadingDistribution: o.LeadingDistribution, Locale: o.Locale, Foreground: o.Foreground, Background: o.Background,
		Decoration: o.Decoration, DecorationColor: o.DecorationColor, DecorationStyle: o.DecorationStyle,
		DecorationThickness: o.DecorationThickness, DebugLabel: o.DebugLabel, Shadows: slices.Clone(o.Shadows),
		FontFeatures: slices.Clone(o.FontFeatures), FontVariations: slices.Clone(o.FontVariations),
		Overflow: o.Overflow, fontFamilyFallback: slices.Clone(o.FontFamilyFallback), pkg: o.Package,
	}
}

// FontFamilyFallback is the ordered list of font families to fall back on, with
// the package prefix applied; nil is null.
//
// Flutter: painting/text_style.dart TextStyle.fontFamilyFallback
func (s *TextStyle) FontFamilyFallback() []string {
	return prefixFamilies(s.fontFamilyFallback, s.pkg)
}

// Package is the package that holds the font, or "".
//
// Flutter: painting/text_style.dart TextStyle._package
func (s *TextStyle) Package() string { return s.pkg }

// unprefixedFamily is the font family without the package prefix.
func (s *TextStyle) unprefixedFamily() string {
	if s.pkg != "" {
		return strings.TrimPrefix(s.FontFamily, "packages/"+s.pkg+"/")
	}
	return s.FontFamily
}

// CopyWith creates a copy of this text style, replacing the fields the
// parameter sets.
//
// Flutter: painting/text_style.dart TextStyle.copyWith
func (s *TextStyle) CopyWith(c TextStyleCopy) *TextStyle {
	if foundation.DebugMode {
		if c.Color.IsSome() && c.Foreground != nil {
			panic(errors.New(tagTextStyleColorAndForeground, "Cannot provide both a color and a foreground\n"+
				`The color argument is just a shorthand for "foreground: Paint()..color = color".`))
		}
		if c.BackgroundColor.IsSome() && c.Background != nil {
			panic(errors.New(tagTextStyleBackgroundAndBackgroundColor, "Cannot provide both a backgroundColor and a background\n"+
				`The backgroundColor argument is just a shorthand for "background: Paint()..color = color".`))
		}
	}
	o := TextStyleOpts{
		Inherit:         foundation.Some(c.Inherit.OrElse(s.Inherit)),
		FontSize:        orOpt(c.FontSize, s.FontSize),
		FontWeight:      orEnum(c.FontWeight, s.FontWeight),
		FontStyle:       orEnum(c.FontStyle, s.FontStyle),
		LetterSpacing:   orOpt(c.LetterSpacing, s.LetterSpacing),
		WordSpacing:     orOpt(c.WordSpacing, s.WordSpacing),
		TextBaseline:    orEnum(c.TextBaseline, s.TextBaseline),
		Height:          orOpt(c.Height, s.Height),
		Locale:          orOpt(c.Locale, s.Locale),
		Foreground:      orPaint(c.Foreground, s.Foreground),
		Background:      orPaint(c.Background, s.Background),
		Decoration:      orOpt(c.Decoration, s.Decoration),
		DecorationColor: orOpt(c.DecorationColor, s.DecorationColor),
		DecorationStyle: orEnum(c.DecorationStyle, s.DecorationStyle),
		Overflow:        orEnum(c.Overflow, s.Overflow),
		FontFamily:      orString(c.FontFamily, s.unprefixedFamily()),
		Package:         orString(c.Package, s.pkg),

		LeadingDistribution: orEnum(c.LeadingDistribution, s.LeadingDistribution),
		DecorationThickness: orOpt(c.DecorationThickness, s.DecorationThickness),
		Shadows:             orSlice(c.Shadows, s.Shadows),
		FontFeatures:        orSlice(c.FontFeatures, s.FontFeatures),
		FontVariations:      orSlice(c.FontVariations, s.FontVariations),
		FontFamilyFallback:  orSlice(c.FontFamilyFallback, s.fontFamilyFallback),
	}
	if s.Foreground == nil && c.Foreground == nil {
		o.Color = orOpt(c.Color, s.Color)
	}
	if s.Background == nil && c.Background == nil {
		o.BackgroundColor = orOpt(c.BackgroundColor, s.BackgroundColor)
	}
	if foundation.DebugMode {
		switch {
		case c.DebugLabel != "":
			o.DebugLabel = c.DebugLabel
		case s.DebugLabel != "":
			o.DebugLabel = "(" + s.DebugLabel + ").copyWith"
		}
	}
	return NewTextStyle(o)
}

func orPaint(a, b *ui.Paint) *ui.Paint {
	if a != nil {
		return a
	}
	return b
}

func orSlice[T any](a, b []T) []T {
	if a != nil {
		return a
	}
	return b
}

// TextStyleApply are the parameters of TextStyle.Apply. A factor defaults to 1
// and a delta to 0.
//
// Flutter: painting/text_style.dart TextStyle.apply
type TextStyleApply struct {
	// Color replaces the color, unless there is a foreground.
	Color foundation.Opt[ui.Color]
	// BackgroundColor replaces the background color, unless there is a
	// background.
	BackgroundColor foundation.Opt[ui.Color]
	// Decoration replaces the decoration.
	Decoration foundation.Opt[ui.TextDecoration]
	// DecorationColor replaces the decoration color.
	DecorationColor foundation.Opt[ui.Color]
	// DecorationStyle replaces the decoration style.
	DecorationStyle ui.TextDecorationStyle
	// DecorationThicknessFactor multiplies the decoration thickness.
	DecorationThicknessFactor foundation.Opt[float64]
	// DecorationThicknessDelta is added to the decoration thickness.
	DecorationThicknessDelta float64
	// FontFamily replaces the font family.
	FontFamily string
	// FontFamilyFallback replaces the font family fallback.
	FontFamilyFallback []string
	// FontSizeFactor multiplies the font size.
	FontSizeFactor foundation.Opt[float64]
	// FontSizeDelta is added to the font size.
	FontSizeDelta float64
	// FontWeightDelta is added to the index of the font weight (w100 is 0).
	FontWeightDelta int
	// FontStyle replaces the font style.
	FontStyle ui.FontStyle
	// LetterSpacingFactor multiplies the letter spacing.
	LetterSpacingFactor foundation.Opt[float64]
	// LetterSpacingDelta is added to the letter spacing.
	LetterSpacingDelta float64
	// WordSpacingFactor multiplies the word spacing.
	WordSpacingFactor foundation.Opt[float64]
	// WordSpacingDelta is added to the word spacing.
	WordSpacingDelta float64
	// HeightFactor multiplies the height.
	HeightFactor foundation.Opt[float64]
	// HeightDelta is added to the height.
	HeightDelta float64
	// TextBaseline replaces the baseline.
	TextBaseline ui.TextBaseline
	// LeadingDistribution replaces the leading distribution.
	LeadingDistribution ui.TextLeadingDistribution
	// Locale replaces the locale.
	Locale foundation.Opt[ui.Locale]
	// Shadows replaces the shadows.
	Shadows []ui.Shadow
	// FontFeatures replaces the font features.
	FontFeatures []ui.FontFeature
	// FontVariations replaces the font variations.
	FontVariations []ui.FontVariation
	// Package replaces the package.
	Package string
	// Overflow replaces the overflow.
	Overflow TextOverflow
}

// Apply creates a copy of this text style, replacing or modifying some fields.
// In debug builds a factor or a delta on a field that is null panics with the
// tag TextStyleApplyNullField.
//
// Flutter: painting/text_style.dart TextStyle.apply
func (s *TextStyle) Apply(a TextStyleApply) *TextStyle {
	fsf, lsf, wsf := a.FontSizeFactor.OrElse(1), a.LetterSpacingFactor.OrElse(1), a.WordSpacingFactor.OrElse(1)
	hf, dtf := a.HeightFactor.OrElse(1), a.DecorationThicknessFactor.OrElse(1)
	if foundation.DebugMode {
		bad := func(field string, set bool, factor, delta float64) {
			if !set && (factor != 1 || delta != 0) {
				panic(errors.New(tagTextStyleApplyNullField, "a factor or delta was given for a field that is null", "field", field))
			}
		}
		bad("fontSize", s.FontSize.IsSome(), fsf, a.FontSizeDelta)
		bad("fontWeight", s.FontWeight != 0, 1, float64(a.FontWeightDelta))
		bad("letterSpacing", s.LetterSpacing.IsSome(), lsf, a.LetterSpacingDelta)
		bad("wordSpacing", s.WordSpacing.IsSome(), wsf, a.WordSpacingDelta)
		bad("decorationThickness", s.DecorationThickness.IsSome(), dtf, a.DecorationThicknessDelta)
	}
	o := TextStyleOpts{
		Inherit:             foundation.Some(s.Inherit),
		FontFamily:          orString(a.FontFamily, s.unprefixedFamily()),
		FontFamilyFallback:  orSlice(a.FontFamilyFallback, s.fontFamilyFallback),
		FontStyle:           orEnum(a.FontStyle, s.FontStyle),
		TextBaseline:        orEnum(a.TextBaseline, s.TextBaseline),
		LeadingDistribution: orEnum(a.LeadingDistribution, s.LeadingDistribution),
		Locale:              orOpt(a.Locale, s.Locale),
		Foreground:          s.Foreground,
		Background:          s.Background,
		Shadows:             orSlice(a.Shadows, s.Shadows),
		FontFeatures:        orSlice(a.FontFeatures, s.FontFeatures),
		FontVariations:      orSlice(a.FontVariations, s.FontVariations),
		Decoration:          orOpt(a.Decoration, s.Decoration),
		DecorationColor:     orOpt(a.DecorationColor, s.DecorationColor),
		DecorationStyle:     orEnum(a.DecorationStyle, s.DecorationStyle),
		Overflow:            orEnum(a.Overflow, s.Overflow),
		Package:             orString(a.Package, s.pkg),
	}
	if s.Foreground == nil {
		o.Color = orOpt(a.Color, s.Color)
	}
	if s.Background == nil {
		o.BackgroundColor = orOpt(a.BackgroundColor, s.BackgroundColor)
	}
	if v, ok := s.FontSize.Get(); ok {
		o.FontSize = foundation.Some(v*fsf + a.FontSizeDelta)
	}
	if s.FontWeight != 0 {
		idx := int(s.FontWeight)/100 - 1 + a.FontWeightDelta
		o.FontWeight = ui.FontWeight((min(max(idx, 0), 8) + 1) * 100)
	}
	if v, ok := s.LetterSpacing.Get(); ok {
		o.LetterSpacing = foundation.Some(v*lsf + a.LetterSpacingDelta)
	}
	if v, ok := s.WordSpacing.Get(); ok {
		o.WordSpacing = foundation.Some(v*wsf + a.WordSpacingDelta)
	}
	if v, ok := s.Height.Get(); ok {
		if v == ui.TextHeightNone {
			o.Height = s.Height
		} else {
			o.Height = foundation.Some(v*hf + a.HeightDelta)
		}
	}
	if v, ok := s.DecorationThickness.Get(); ok {
		o.DecorationThickness = foundation.Some(v*dtf + a.DecorationThicknessDelta)
	}
	if foundation.DebugMode && s.DebugLabel != "" {
		o.DebugLabel = "(" + s.DebugLabel + ").apply"
	}
	return NewTextStyle(o)
}

// Merge returns a new text style that is a combination of this style and the
// given other style. A nil other, or one that doesn't inherit, is returned
// as it is.
//
// Flutter: painting/text_style.dart TextStyle.merge
func (s *TextStyle) Merge(other *TextStyle) *TextStyle {
	if other == nil {
		return s
	}
	if !other.Inherit {
		return other
	}
	c := TextStyleCopy{
		Color: other.Color, BackgroundColor: other.BackgroundColor, FontSize: other.FontSize, FontWeight: other.FontWeight,
		FontStyle: other.FontStyle, LetterSpacing: other.LetterSpacing, WordSpacing: other.WordSpacing,
		TextBaseline: other.TextBaseline, Height: other.Height, LeadingDistribution: other.LeadingDistribution,
		Locale: other.Locale, Foreground: other.Foreground, Background: other.Background, Shadows: other.Shadows,
		FontFeatures: other.FontFeatures, FontVariations: other.FontVariations, Decoration: other.Decoration,
		DecorationColor: other.DecorationColor, DecorationStyle: other.DecorationStyle,
		DecorationThickness: other.DecorationThickness, FontFamily: other.unprefixedFamily(),
		FontFamilyFallback: other.fontFamilyFallback, Package: other.pkg, Overflow: other.Overflow,
	}
	if foundation.DebugMode && (other.DebugLabel != "" || s.DebugLabel != "") {
		c.DebugLabel = "(" + orString(s.DebugLabel, defaultDebugLabel) + ").merge(" + orString(other.DebugLabel, defaultDebugLabel) + ")"
	}
	return s.CopyWith(c)
}

func lerpColorOpt(a, b foundation.Opt[ui.Color], t float64) foundation.Opt[ui.Color] {
	r := ui.ColorLerp(optPtr(a), optPtr(b), t)
	if r == nil {
		return foundation.None[ui.Color]()
	}
	return foundation.Some(*r)
}

// lerpDoubleFallback is ui.lerpDouble(a ?? b, b ?? a, t).
func lerpDoubleFallback(a, b foundation.Opt[float64], t float64) foundation.Opt[float64] {
	aa, bb := orOpt(a, b), orOpt(b, a)
	if aa.IsNone() {
		return aa
	}
	return foundation.Some(ui.LerpDouble(aa.Value(), bb.Value(), t))
}

func pick[T any](t float64, a, b T) T {
	if t < 0.5 {
		return a
	}
	return b
}

// TextStyleLerp linearly interpolates between two text styles. Either may be
// nil. In debug builds interpolating styles whose Inherit differs while fields
// are null in both panics with the tag TextStyleLerpInheritMismatch.
//
// Flutter: painting/text_style.dart TextStyle.lerp
func TextStyleLerp(a, b *TextStyle, t float64) *TextStyle {
	if a == b {
		return a
	}
	var label string
	if foundation.DebugMode {
		la, lb := defaultDebugLabel, defaultDebugLabel
		if a != nil && a.DebugLabel != "" {
			la = a.DebugLabel
		}
		if b != nil && b.DebugLabel != "" {
			lb = b.DebugLabel
		}
		label = "lerp(" + la + " ⎯" + strconv.FormatFloat(t, 'f', 1, 64) + "→ " + lb + ")"
	}
	null := foundation.None[ui.Color]()
	if a == nil {
		return NewTextStyle(TextStyleOpts{
			Inherit: foundation.Some(b.Inherit), Color: lerpColorOpt(null, b.Color, t),
			BackgroundColor:     lerpColorOpt(null, b.BackgroundColor, t),
			FontSize:            pick(t, foundation.None[float64](), b.FontSize),
			FontWeight:          ui.FontWeightLerp(0, b.FontWeight, t),
			FontStyle:           pick(t, ui.FontStyleUnspecified, b.FontStyle),
			LetterSpacing:       pick(t, foundation.None[float64](), b.LetterSpacing),
			WordSpacing:         pick(t, foundation.None[float64](), b.WordSpacing),
			TextBaseline:        pick(t, ui.TextBaselineUnspecified, b.TextBaseline),
			Height:              pick(t, foundation.None[float64](), b.Height),
			LeadingDistribution: pick(t, ui.TextLeadingDistributionUnspecified, b.LeadingDistribution),
			Locale:              pick(t, foundation.None[ui.Locale](), b.Locale),
			Foreground:          pick(t, nil, b.Foreground),
			Background:          pick(t, nil, b.Background),
			Shadows:             pick(t, nil, b.Shadows),
			FontFeatures:        pick(t, nil, b.FontFeatures),
			FontVariations:      lerpFontVariations(nil, b.FontVariations, t),
			Decoration:          pick(t, foundation.None[ui.TextDecoration](), b.Decoration),
			DecorationColor:     lerpColorOpt(null, b.DecorationColor, t),
			DecorationStyle:     pick(t, ui.TextDecorationStyleUnspecified, b.DecorationStyle),
			DecorationThickness: pick(t, foundation.None[float64](), b.DecorationThickness),
			DebugLabel:          label,
			FontFamily:          pick(t, "", b.unprefixedFamily()),
			FontFamilyFallback:  pick(t, nil, b.fontFamilyFallback),
			Package:             pick(t, "", b.pkg),
			Overflow:            pick(t, TextOverflowUnspecified, b.Overflow),
		})
	}
	if b == nil {
		return NewTextStyle(TextStyleOpts{
			Inherit: foundation.Some(a.Inherit), Color: lerpColorOpt(a.Color, null, t),
			// Flutter lerps the background color from null here, not from a's.
			BackgroundColor:     lerpColorOpt(null, a.BackgroundColor, t),
			FontSize:            pick(t, a.FontSize, foundation.None[float64]()),
			FontWeight:          ui.FontWeightLerp(a.FontWeight, 0, t),
			FontStyle:           pick(t, a.FontStyle, ui.FontStyleUnspecified),
			LetterSpacing:       pick(t, a.LetterSpacing, foundation.None[float64]()),
			WordSpacing:         pick(t, a.WordSpacing, foundation.None[float64]()),
			TextBaseline:        pick(t, a.TextBaseline, ui.TextBaselineUnspecified),
			Height:              pick(t, a.Height, foundation.None[float64]()),
			LeadingDistribution: pick(t, a.LeadingDistribution, ui.TextLeadingDistributionUnspecified),
			Locale:              pick(t, a.Locale, foundation.None[ui.Locale]()),
			Foreground:          pick(t, a.Foreground, nil),
			Background:          pick(t, a.Background, nil),
			Shadows:             pick(t, a.Shadows, nil),
			FontFeatures:        pick(t, a.FontFeatures, nil),
			FontVariations:      lerpFontVariations(a.FontVariations, nil, t),
			Decoration:          pick(t, a.Decoration, foundation.None[ui.TextDecoration]()),
			DecorationColor:     lerpColorOpt(a.DecorationColor, null, t),
			DecorationStyle:     pick(t, a.DecorationStyle, ui.TextDecorationStyleUnspecified),
			DecorationThickness: pick(t, a.DecorationThickness, foundation.None[float64]()),
			DebugLabel:          label,
			FontFamily:          pick(t, a.unprefixedFamily(), ""),
			FontFamilyFallback:  pick(t, a.fontFamilyFallback, nil),
			Package:             pick(t, a.pkg, ""),
			Overflow:            pick(t, a.Overflow, TextOverflowUnspecified),
		})
	}
	if foundation.DebugMode && a.Inherit != b.Inherit {
		var nullFields []string
		add := func(cond bool, name string) {
			if cond {
				nullFields = append(nullFields, `"`+name+`"`)
			}
		}
		add(a.Foreground == nil && b.Foreground == nil && a.Color.IsNone() && b.Color.IsNone(), "color")
		add(a.Background == nil && b.Background == nil && a.BackgroundColor.IsNone() && b.BackgroundColor.IsNone(), "backgroundColor")
		add(a.FontSize.IsNone() && b.FontSize.IsNone(), "fontSize")
		add(a.LetterSpacing.IsNone() && b.LetterSpacing.IsNone(), "letterSpacing")
		add(a.WordSpacing.IsNone() && b.WordSpacing.IsNone(), "wordSpacing")
		add(a.Height.IsNone() && b.Height.IsNone(), "height")
		add(a.DecorationColor.IsNone() && b.DecorationColor.IsNone(), "decorationColor")
		add(a.DecorationThickness.IsNone() && b.DecorationThickness.IsNone(), "decorationThickness")
		if len(nullFields) > 0 {
			panic(errors.New(tagTextStyleLerpInheritMismatch, "Failed to interpolate TextStyles with different inherit values.\n"+
				"The following fields are unspecified in both TextStyles:\n"+strings.Join(nullFields, ", ")+".\n"+
				`When "inherit" changes during the transition, these fields may observe abrupt value changes as a result, `+
				`causing "jump"s in the transition.`+"\n"+
				`In general, TextStyle.lerp only works well when both TextStyles have the same "inherit" value, and specify the same fields.`,
				"from", a.String(), "to", b.String()))
		}
	}
	o := TextStyleOpts{
		Inherit:             foundation.Some(pick(t, a.Inherit, b.Inherit)),
		FontSize:            lerpDoubleFallback(a.FontSize, b.FontSize, t),
		FontWeight:          ui.FontWeightLerp(a.FontWeight, b.FontWeight, t),
		FontStyle:           pick(t, a.FontStyle, b.FontStyle),
		LetterSpacing:       lerpDoubleFallback(a.LetterSpacing, b.LetterSpacing, t),
		WordSpacing:         lerpDoubleFallback(a.WordSpacing, b.WordSpacing, t),
		TextBaseline:        pick(t, a.TextBaseline, b.TextBaseline),
		Height:              lerpDoubleFallback(a.Height, b.Height, t),
		LeadingDistribution: pick(t, a.LeadingDistribution, b.LeadingDistribution),
		Locale:              pick(t, a.Locale, b.Locale),
		Shadows:             ui.ShadowLerpList(a.Shadows, b.Shadows, t),
		FontFeatures:        pick(t, a.FontFeatures, b.FontFeatures),
		FontVariations:      lerpFontVariations(a.FontVariations, b.FontVariations, t),
		Decoration:          pick(t, a.Decoration, b.Decoration),
		DecorationColor:     lerpColorOpt(a.DecorationColor, b.DecorationColor, t),
		DecorationStyle:     pick(t, a.DecorationStyle, b.DecorationStyle),
		DecorationThickness: lerpDoubleFallback(a.DecorationThickness, b.DecorationThickness, t),
		DebugLabel:          label,
		FontFamily:          pick(t, a.unprefixedFamily(), b.unprefixedFamily()),
		FontFamilyFallback:  pick(t, a.fontFamilyFallback, b.fontFamilyFallback),
		Package:             pick(t, a.pkg, b.pkg),
		Overflow:            pick(t, a.Overflow, b.Overflow),
	}
	if a.Foreground == nil && b.Foreground == nil {
		o.Color = lerpColorOpt(a.Color, b.Color, t)
	}
	if a.Background == nil && b.Background == nil {
		o.BackgroundColor = lerpColorOpt(a.BackgroundColor, b.BackgroundColor, t)
	}
	if a.Foreground != nil || b.Foreground != nil {
		o.Foreground = pick(t, paintOrColor(a.Foreground, a.Color), paintOrColor(b.Foreground, b.Color))
	}
	if a.Background != nil || b.Background != nil {
		o.Background = pick(t, paintOrColor(a.Background, a.BackgroundColor), paintOrColor(b.Background, b.BackgroundColor))
	}
	return NewTextStyle(o)
}

// paintOrColor is `paint ?? (Paint()..color = color!)`.
func paintOrColor(p *ui.Paint, c foundation.Opt[ui.Color]) *ui.Paint {
	if p != nil {
		return p
	}
	np := ui.NewPaint()
	np.SetColor(c.Value())
	return np
}

// lerpFontVariations interpolates two lists of font variations, matching axes.
//
// Flutter: painting/text_style.dart lerpFontVariations
func lerpFontVariations(a, b []ui.FontVariation, t float64) []ui.FontVariation {
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	if len(a) == 0 || len(b) == 0 {
		return pick(t, a, b)
	}
	var result []ui.FontVariation
	index := 0
	minLength := min(len(a), len(b))
	for ; index < minLength; index++ {
		if a[index].Axis != b[index].Axis {
			break
		}
		result = append(result, *ui.FontVariationLerp(&a[index], &b[index], t))
	}
	if index < max(len(a), len(b)) {
		var axes []string
		seen := map[string]bool{}
		aVars, bVars := map[string]*ui.FontVariation{}, map[string]*ui.FontVariation{}
		for i := index; i < len(a); i++ {
			aVars[a[i].Axis] = &a[i]
			if !seen[a[i].Axis] {
				seen[a[i].Axis] = true
				axes = append(axes, a[i].Axis)
			}
		}
		for i := index; i < len(b); i++ {
			bVars[b[i].Axis] = &b[i]
			if !seen[b[i].Axis] {
				seen[b[i].Axis] = true
				axes = append(axes, b[i].Axis)
			}
		}
		for _, axis := range axes {
			if v := ui.FontVariationLerp(aVars[axis], bVars[axis], t); v != nil {
				result = append(result, *v)
			}
		}
	}
	if result == nil {
		result = []ui.FontVariation{}
	}
	return result
}

// GetTextStyleOpts are the optional parameters of TextStyle.GetTextStyle.
//
// Flutter: painting/text_style.dart TextStyle.getTextStyle
type GetTextStyleOpts struct {
	// TextScaler scales the font size; the default is TextScalerNoScaling.
	TextScaler TextScaler
}

// GetTextStyle returns the engine text style for this style.
//
// Flutter: painting/text_style.dart TextStyle.getTextStyle
func (s *TextStyle) GetTextStyle(opts ...GetTextStyleOpts) *ui.TextStyle {
	scaler := firstOpts(opts).TextScaler
	if scaler == nil {
		scaler = TextScalerNoScaling
	}
	o := ui.TextStyleOpts{
		Color:               optPtr(s.Color),
		Decoration:          optPtr(s.Decoration),
		DecorationColor:     optPtr(s.DecorationColor),
		DecorationStyle:     s.DecorationStyle,
		DecorationThickness: optPtr(s.DecorationThickness),
		FontWeight:          s.FontWeight,
		FontStyle:           s.FontStyle,
		TextBaseline:        s.TextBaseline,
		LeadingDistribution: s.LeadingDistribution,
		FontFamily:          s.FontFamily,
		FontFamilyFallback:  s.FontFamilyFallback(),
		LetterSpacing:       optPtr(s.LetterSpacing),
		WordSpacing:         optPtr(s.WordSpacing),
		Height:              optPtr(s.Height),
		Locale:              optPtr(s.Locale),
		Foreground:          s.Foreground,
		Shadows:             s.Shadows,
		FontFeatures:        s.FontFeatures,
		FontVariations:      s.FontVariations,
	}
	if v, ok := s.FontSize.Get(); ok {
		o.FontSize = ui.Ptr(scaler.Scale(v))
	}
	switch {
	case s.Background != nil:
		o.Background = s.Background
	case s.BackgroundColor.IsSome():
		o.Background = ui.NewPaint()
		o.Background.SetColor(s.BackgroundColor.Value())
	}
	return ui.NewTextStyle(o)
}

// GetParagraphStyleOpts are the optional parameters of
// TextStyle.GetParagraphStyle.
//
// Flutter: painting/text_style.dart TextStyle.getParagraphStyle
type GetParagraphStyleOpts struct {
	// TextAlign is the alignment of the text in the lines.
	TextAlign ui.TextAlign
	// TextDirection is the directionality of the text.
	TextDirection ui.TextDirection
	// TextScaler scales the font sizes; the default is TextScalerNoScaling.
	TextScaler TextScaler
	// Ellipsis replaces the text that doesn't fit when MaxLines is exceeded.
	Ellipsis string
	// MaxLines is the maximum number of lines; it must be greater than zero.
	MaxLines foundation.Opt[int]
	// TextHeightBehavior says how Height applies to the first and last line.
	TextHeightBehavior *ui.TextHeightBehavior
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// FontFamily overrides the style's family.
	FontFamily string
	// FontSize overrides the style's font size.
	FontSize foundation.Opt[float64]
	// FontWeight overrides the style's weight.
	FontWeight ui.FontWeight
	// FontStyle overrides the style's font style.
	FontStyle ui.FontStyle
	// Height overrides the style's height.
	Height foundation.Opt[float64]
	// StrutStyle is the strut of the paragraph.
	StrutStyle *StrutStyle
	// TabSize is the distance between tab stops as a count of the advance of a space; 0 gives
	// tabs no stops.
	//
	// Flutter: no counterpart
	TabSize float64
}

// GetParagraphStyle returns the engine paragraph style for this style. In debug
// builds a MaxLines that is not greater than zero panics with the tag
// ParagraphMaxLinesInvalid.
//
// Flutter: painting/text_style.dart TextStyle.getParagraphStyle
func (s *TextStyle) GetParagraphStyle(opts ...GetParagraphStyleOpts) *ui.ParagraphStyle {
	p := firstOpts(opts)
	if foundation.DebugMode && p.MaxLines.IsSome() && p.MaxLines.Value() <= 0 {
		panic(errors.New(tagParagraphMaxLinesInvalid, "maxLines must be greater than zero", "maxLines", p.MaxLines.Value()))
	}
	scaler := p.TextScaler
	if scaler == nil {
		scaler = TextScalerNoScaling
	}
	behavior := p.TextHeightBehavior
	if behavior == nil && s.LeadingDistribution != ui.TextLeadingDistributionUnspecified {
		b := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOpts{LeadingDistribution: s.LeadingDistribution})
		behavior = &b
	}
	o := ui.ParagraphStyleOpts{
		TextAlign:          p.TextAlign,
		TextDirection:      p.TextDirection,
		FontWeight:         orEnum(p.FontWeight, s.FontWeight),
		FontStyle:          orEnum(p.FontStyle, s.FontStyle),
		FontFamily:         orString(p.FontFamily, s.FontFamily),
		FontSize:           ui.Ptr(scaler.Scale(orOpt(p.FontSize, s.FontSize).OrElse(DefaultFontSize))),
		Height:             optPtr(orOpt(p.Height, s.Height)),
		TextHeightBehavior: behavior,
		Ellipsis:           p.Ellipsis,
		Locale:             optPtr(p.Locale),
		TabSize:            p.TabSize,
	}
	if v, ok := p.MaxLines.Get(); ok {
		o.MaxLines = &v
	}
	if p.StrutStyle != nil {
		o.StrutStyle = p.StrutStyle.toUI(scaler)
	}
	return ui.NewParagraphStyle(o)
}

// CompareTo describes the difference between two text styles: layout when one
// affects the layout, paint when only the painting differs, else identical.
//
// Flutter: painting/text_style.dart TextStyle.compareTo
func (s *TextStyle) CompareTo(other *TextStyle) RenderComparison {
	if s == other {
		return RenderComparisonIdentical
	}
	if s.Inherit != other.Inherit || s.FontFamily != other.FontFamily || s.FontSize != other.FontSize ||
		s.FontWeight != other.FontWeight || s.FontStyle != other.FontStyle || s.LetterSpacing != other.LetterSpacing ||
		s.WordSpacing != other.WordSpacing || s.TextBaseline != other.TextBaseline || s.Height != other.Height ||
		s.LeadingDistribution != other.LeadingDistribution || s.Locale != other.Locale ||
		s.Foreground != other.Foreground || s.Background != other.Background ||
		!slices.Equal(s.Shadows, other.Shadows) || !slices.Equal(s.FontFeatures, other.FontFeatures) ||
		!slices.Equal(s.FontVariations, other.FontVariations) ||
		!slices.Equal(s.FontFamilyFallback(), other.FontFamilyFallback()) || s.Overflow != other.Overflow {
		return RenderComparisonLayout
	}
	if s.Color != other.Color || s.BackgroundColor != other.BackgroundColor || s.Decoration != other.Decoration ||
		s.DecorationColor != other.DecorationColor || s.DecorationStyle != other.DecorationStyle ||
		s.DecorationThickness != other.DecorationThickness {
		return RenderComparisonPaint
	}
	return RenderComparisonIdentical
}

// Equal is Dart's == on text styles.
//
// Flutter: painting/text_style.dart TextStyle.operator ==
func (s *TextStyle) Equal(other *TextStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return s.Inherit == other.Inherit && s.Color == other.Color && s.BackgroundColor == other.BackgroundColor &&
		s.FontSize == other.FontSize && s.FontWeight == other.FontWeight && s.FontStyle == other.FontStyle &&
		s.LetterSpacing == other.LetterSpacing && s.WordSpacing == other.WordSpacing &&
		s.TextBaseline == other.TextBaseline && s.Height == other.Height &&
		s.LeadingDistribution == other.LeadingDistribution && s.Locale == other.Locale &&
		s.Foreground == other.Foreground && s.Background == other.Background &&
		slices.Equal(s.Shadows, other.Shadows) && slices.Equal(s.FontFeatures, other.FontFeatures) &&
		slices.Equal(s.FontVariations, other.FontVariations) && s.Decoration == other.Decoration &&
		s.DecorationColor == other.DecorationColor && s.DecorationStyle == other.DecorationStyle &&
		s.DecorationThickness == other.DecorationThickness && s.FontFamily == other.FontFamily &&
		slices.Equal(s.FontFamilyFallback(), other.FontFamilyFallback()) && s.pkg == other.pkg &&
		s.Overflow == other.Overflow
}

// ToStringShort names the style.
//
// Flutter: painting/text_style.dart TextStyle.toStringShort
func (s *TextStyle) ToStringShort() string { return "TextStyle" }

// String describes the style and its properties.
//
// Flutter: painting/text_style.dart TextStyle.toString
func (s *TextStyle) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the properties of the style.
//
// Flutter: painting/text_style.dart TextStyle.debugFillProperties
func (s *TextStyle) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	s.DebugFillPropertiesWithPrefix(p, "")
}

// DebugFillPropertiesWithPrefix is DebugFillProperties with a prefix before
// each property name, which TextSpan uses for the "style: " properties.
//
// Flutter: painting/text_style.dart TextStyle.debugFillProperties
func (s *TextStyle) DebugFillPropertiesWithPrefix(p *foundation.DiagnosticPropertiesBuilder, prefix string) {
	null := foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}
	if s.DebugLabel != "" {
		p.Add(foundation.NewMessageProperty(prefix+"debugLabel", s.DebugLabel))
	}
	var family foundation.Opt[string]
	if s.FontFamily != "" {
		family = foundation.Some(s.FontFamily)
	}
	var fallback []string
	if s.fontFamilyFallback != nil {
		fallback = s.FontFamilyFallback()
	}
	styles := []foundation.DiagnosticsNode{
		NewColorProperty(prefix+"color", s.Color, null),
		NewColorProperty(prefix+"backgroundColor", s.BackgroundColor, null),
		foundation.NewStringProperty(prefix+"family", family, foundation.StringPropertyOpts{DefaultValue: foundation.DefaultNull, Unquoted: true}),
		foundation.NewIterableProperty(prefix+"familyFallback", fallback, foundation.IterablePropertyOpts{DefaultValue: foundation.DefaultNull}),
		foundation.NewDoubleProperty(prefix+"size", s.FontSize, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}),
	}
	weight := foundation.None[ui.FontWeight]()
	weightOpts := null
	if s.FontWeight != 0 {
		weight = foundation.Some(s.FontWeight)
		weightOpts.Description = strconv.Itoa(int(s.FontWeight)/100) + "00"
	}
	enumNull := foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}
	styles = append(styles,
		foundation.NewDiagnosticsPropertyOpt(prefix+"weight", weight, weightOpts),
		foundation.NewEnumProperty(prefix+"style", optEnum(s.FontStyle), enumNull),
		foundation.NewDoubleProperty(prefix+"letterSpacing", s.LetterSpacing, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}),
		foundation.NewDoubleProperty(prefix+"wordSpacing", s.WordSpacing, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}),
		foundation.NewEnumProperty(prefix+"baseline", optEnum(s.TextBaseline), enumNull),
		foundation.NewDoubleProperty(prefix+"height", s.Height, foundation.NumberPropertyOpts{Unit: "x", DefaultValue: foundation.DefaultNull}),
		foundation.NewEnumProperty(prefix+"leadingDistribution", optEnum(s.LeadingDistribution), enumNull),
		foundation.NewDiagnosticsPropertyOpt(prefix+"locale", s.Locale, null),
		foundation.NewDiagnosticsPropertyOpt(prefix+"foreground", optPaint(s.Foreground), null),
		foundation.NewDiagnosticsPropertyOpt(prefix+"background", optPaint(s.Background), null),
	)
	if s.Decoration.IsSome() || s.DecorationColor.IsSome() || s.DecorationStyle != ui.TextDecorationStyleUnspecified || s.DecorationThickness.IsSome() {
		var desc []string
		if s.DecorationStyle != ui.TextDecorationStyleUnspecified {
			desc = append(desc, s.DecorationStyle.Name())
		}
		fine := null
		fine.Level = foundation.DiagnosticLevelFine
		styles = append(styles, NewColorProperty(prefix+"decorationColor", s.DecorationColor, fine))
		if s.DecorationColor.IsSome() {
			desc = append(desc, s.DecorationColor.Value().String())
		}
		hidden := null
		hidden.Level = foundation.DiagnosticLevelHidden
		styles = append(styles, foundation.NewDiagnosticsPropertyOpt(prefix+"decoration", s.Decoration, hidden))
		if s.Decoration.IsSome() {
			desc = append(desc, s.Decoration.Value().String())
		}
		styles = append(styles,
			foundation.NewMessageProperty(prefix+"decoration", strings.Join(desc, " ")),
			foundation.NewDoubleProperty(prefix+"decorationThickness", s.DecorationThickness, foundation.NumberPropertyOpts{Unit: "x", DefaultValue: foundation.DefaultNull}),
		)
	}
	styleSpecified := false
	for _, n := range styles {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			styleSpecified = true
		}
	}
	level := foundation.DiagnosticLevelInfo
	if !styleSpecified && s.Inherit {
		level = foundation.DiagnosticLevelFine
	}
	p.Add(foundation.NewDiagnosticsProperty(prefix+"inherit", s.Inherit, foundation.DiagnosticsPropertyOpts{Level: level}))
	for _, n := range styles {
		p.Add(n)
	}
	if !styleSpecified {
		p.Add(foundation.NewFlagProperty("inherit", foundation.Some(s.Inherit),
			foundation.FlagPropertyOpts{IfTrue: prefix + "<all styles inherited>", IfFalse: prefix + "<no style specified>"}))
	}
}

func optPaint(p *ui.Paint) foundation.Opt[*ui.Paint] {
	if p == nil {
		return foundation.None[*ui.Paint]()
	}
	return foundation.Some(p)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagParagraphMaxLinesInvalid              = "ParagraphMaxLinesInvalid"
	tagTextStyleApplyNullField               = "TextStyleApplyNullField"
	tagTextStyleBackgroundAndBackgroundColor = "TextStyleBackgroundAndBackgroundColor"
	tagTextStyleColorAndForeground           = "TextStyleColorAndForeground"
	tagTextStyleLerpInheritMismatch          = "TextStyleLerpInheritMismatch"
)
