// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
)

// Locale identifies a language, optionally with a script and a country.
//
// Flutter: ui/platform_dispatcher.dart Locale
type Locale struct {
	// LanguageCode is the ISO 639 language subtag, such as "en", or "" for none.
	LanguageCode string
	// ScriptCode is the ISO 15924 script subtag, such as "Latn", or "".
	ScriptCode string
	// CountryCode is the ISO 3166 region subtag, such as "US", or "".
	CountryCode string
}

// NewLocale creates a locale from a language code and an optional country code.
//
// Flutter: ui/platform_dispatcher.dart Locale
func NewLocale(languageCode string, countryCode ...string) Locale {
	l := Locale{LanguageCode: languageCode}
	if len(countryCode) > 0 {
		l.CountryCode = countryCode[0]
	}
	return l
}

// IsZero reports whether the locale names nothing.
//
// Flutter: no counterpart
func (l Locale) IsZero() bool { return l == Locale{} }

// String is the locale the way Dart prints it: subtags joined with "_".
//
// Flutter: ui/platform_dispatcher.dart Locale.toString
func (l Locale) String() string { return l.join("_") }

// LanguageTag is the BCP 47 language tag of the locale, such as "zh-Hans-CN".
//
// Flutter: ui/platform_dispatcher.dart Locale.toLanguageTag
func (l Locale) LanguageTag() string { return l.join("-") }

func (l Locale) join(sep string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{l.LanguageCode, l.ScriptCode, l.CountryCode} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, sep)
}

// FontFeature is a feature tag and a value that configures the font's OpenType features, such as "liga" or "tnum".
//
// Flutter: ui/text.dart FontFeature
type FontFeature struct {
	// Feature is the four-letter OpenType feature tag.
	Feature string
	// Value is 0 to turn the feature off, 1 to turn it on, or the index of an alternate.
	Value int
}

// NewFontFeature creates a feature. A tag that is not four letters long, or a negative value, panics with the tag FontFeatureInvalid.
//
// Flutter: ui/text.dart FontFeature
func NewFontFeature(feature string, value int) FontFeature {
	if len(feature) != 4 || value < 0 {
		panic(errors.New("FontFeatureInvalid", "the feature tag must be four characters long and the value zero or positive",
			"feature", feature, "value", value))
	}
	return FontFeature{feature, value}
}

// String returns what Dart prints, such as "FontFeature(liga, 1)".
func (f FontFeature) String() string { return fmt.Sprintf("FontFeature(%s, %d)", f.Feature, f.Value) }

// FontVariation is an axis tag and a value that configures the font's variable axes, such as "wght" or "wdth".
//
// Flutter: ui/text.dart FontVariation
type FontVariation struct {
	// Axis is the four-letter axis tag.
	Axis string
	// Value is the value on the axis, in the font's design units.
	Value float64
}

// NewFontVariation creates a variation. A tag that is not four letters long, or a value outside -32768 to 32768, panics with the tag FontVariationInvalid.
//
// Flutter: ui/text.dart FontVariation
func NewFontVariation(axis string, value float64) FontVariation {
	if len(axis) != 4 || value < -32768 || value >= 32768 {
		panic(errors.New("FontVariationInvalid", "the axis tag must be four characters long and the value representable as 16.16 fixed point",
			"axis", axis, "value", value))
	}
	return FontVariation{axis, value}
}

// String returns what Dart prints, such as "FontVariation(wght, 700)".
func (v FontVariation) String() string { return fmt.Sprintf("FontVariation(%s, %v)", v.Axis, v.Value) }

// Shadow is a single shadow cast by a piece of text.
//
// The zero Shadow has a transparent color; NewShadow gives Flutter's default, opaque black.
//
// Flutter: ui/painting.dart Shadow
type Shadow struct {
	// Color is the color of the shadow.
	Color Color
	// Offset is the displacement of the shadow from the text.
	Offset Offset
	// BlurRadius is the blur radius; the Gaussian blur uses BlurSigma, derived from it.
	BlurRadius float64
}

// ShadowOpts are the optional parameters of NewShadow.
//
// Flutter: ui/painting.dart Shadow
type ShadowOpts struct {
	// Color defaults to opaque black.
	Color *Color
	// Offset defaults to zero.
	Offset Offset
	// BlurRadius defaults to 0 and must not be negative.
	BlurRadius float64
}

// NewShadow creates a shadow. A negative blur radius panics with the tag ShadowBlurRadiusNegative.
//
// Flutter: ui/painting.dart Shadow
func NewShadow(opts ...ShadowOpts) Shadow {
	o := oneOpts(opts)
	if o.BlurRadius < 0 {
		panic(errors.New("ShadowBlurRadiusNegative", "text shadow blur radius should be non-negative", "blurRadius", o.BlurRadius))
	}
	s := Shadow{Color: NewColor(0xFF000000), Offset: o.Offset, BlurRadius: o.BlurRadius}
	if o.Color != nil {
		s.Color = *o.Color
	}
	return s
}

// ConvertRadiusToSigma converts a blur radius to the standard deviation Skia uses for it.
//
// Flutter: ui/painting.dart Shadow.convertRadiusToSigma
func ConvertRadiusToSigma(radius float64) float64 {
	if radius > 0 {
		return radius*0.57735 + 0.5
	}
	return 0
}

// BlurSigma is the standard deviation of the Gaussian blur of the shadow.
//
// Flutter: ui/painting.dart Shadow.blurSigma
func (s Shadow) BlurSigma() float64 { return ConvertRadiusToSigma(s.BlurRadius) }

// TextHeightBehavior defines how to apply TextStyle.Height over the first ascent and the last descent of a paragraph.
//
// Flutter: ui/text.dart TextHeightBehavior
type TextHeightBehavior struct {
	// ApplyHeightToFirstAscent says whether the line height applies to the ascent of the first line.
	ApplyHeightToFirstAscent bool
	// ApplyHeightToLastDescent says whether the line height applies to the descent of the last line.
	ApplyHeightToLastDescent bool
	// LeadingDistribution is how the extra space of a line is divided; the unspecified value is proportional.
	LeadingDistribution TextLeadingDistribution
}

// TextHeightBehaviorOpts are the optional parameters of NewTextHeightBehavior.
//
// Flutter: ui/text.dart TextHeightBehavior
type TextHeightBehaviorOpts struct {
	// ApplyHeightToFirstAscent defaults to true.
	ApplyHeightToFirstAscent *bool
	// ApplyHeightToLastDescent defaults to true.
	ApplyHeightToLastDescent *bool
	// LeadingDistribution defaults to proportional.
	LeadingDistribution TextLeadingDistribution
}

// NewTextHeightBehavior creates a TextHeightBehavior with Flutter's defaults for what the options leave out.
//
// Flutter: ui/text.dart TextHeightBehavior
func NewTextHeightBehavior(opts ...TextHeightBehaviorOpts) TextHeightBehavior {
	o := oneOpts(opts)
	b := TextHeightBehavior{true, true, o.LeadingDistribution}
	if o.ApplyHeightToFirstAscent != nil {
		b.ApplyHeightToFirstAscent = *o.ApplyHeightToFirstAscent
	}
	if o.ApplyHeightToLastDescent != nil {
		b.ApplyHeightToLastDescent = *o.ApplyHeightToLastDescent
	}
	return b
}

// distribution is the leading distribution, with the default applied.
func (b TextHeightBehavior) distribution() TextLeadingDistribution {
	if b.LeadingDistribution == TextLeadingDistributionUnspecified {
		return TextLeadingDistributionProportional
	}
	return b.LeadingDistribution
}

// String returns what Dart prints for the behavior.
func (b TextHeightBehavior) String() string {
	return fmt.Sprintf("TextHeightBehavior(applyHeightToFirstAscent: %v, applyHeightToLastDescent: %v, leadingDistribution: %v)",
		b.ApplyHeightToFirstAscent, b.ApplyHeightToLastDescent, b.distribution())
}

// ParagraphConstraints are the constraints a Paragraph is laid out with.
//
// Flutter: ui/text.dart ParagraphConstraints
type ParagraphConstraints struct {
	// Width is the width the lines are wrapped to. An infinite width lays the text out with no soft line breaks.
	Width float64
}

// NewParagraphConstraints creates the constraints for a width.
//
// Flutter: ui/text.dart ParagraphConstraints
func NewParagraphConstraints(width float64) ParagraphConstraints { return ParagraphConstraints{width} }

// LineMetrics is the metrics of one line of a laid-out paragraph.
//
// Flutter: ui/text.dart LineMetrics
type LineMetrics struct {
	// HardBreak is true if the line ends with a hard line break (a newline).
	HardBreak bool
	// Ascent is the rise from the baseline as calculated from the font and style for this line, with the line height applied.
	Ascent float64
	// Descent is the drop from the baseline as calculated from the font and style for this line, with the line height applied.
	Descent float64
	// UnscaledAscent is the rise from the baseline as calculated from the font and style, without the line height.
	UnscaledAscent float64
	// Height is the total height of the line from the top of the paragraph, the sum of Ascent and Descent (rounded).
	Height float64
	// Width is the width of the line, not counting the trailing whitespace.
	Width float64
	// Left is the x coordinate of the left edge of the line.
	Left float64
	// Baseline is the y coordinate of the baseline, from the top of the paragraph.
	Baseline float64
	// LineNumber is the number of the line, starting at 0.
	LineNumber int
}

// ParagraphLayout is a laid-out paragraph as the text engine provides it. A Paragraph wraps one. It is implemented by internal/text/layout; a test may provide its own.
//
// Flutter: no counterpart
type ParagraphLayout interface {
	// Layout lays the text out for the constraints. It can be called again with other constraints.
	Layout(ParagraphConstraints)
	// Width is the amount of horizontal space this paragraph occupies.
	Width() float64
	// Height is the amount of vertical space this paragraph occupies.
	Height() float64
	// LongestLine is the width of the widest line, not counting trailing whitespace.
	LongestLine() float64
	// MinIntrinsicWidth is the minimum width that this paragraph could be without failing to paint its contents within itself.
	MinIntrinsicWidth() float64
	// MaxIntrinsicWidth is the width at which decreasing the width would not decrease the height.
	MaxIntrinsicWidth() float64
	// AlphabeticBaseline is the distance from the top of the paragraph to the alphabetic baseline of the first line.
	AlphabeticBaseline() float64
	// IdeographicBaseline is the distance from the top of the paragraph to the ideographic baseline of the first line.
	IdeographicBaseline() float64
	// DidExceedMaxLines is true if there is more text than the style's maximum number of lines could show.
	DidExceedMaxLines() bool
	// ComputeLineMetrics returns the metrics of every line.
	ComputeLineMetrics() []LineMetrics
	// GetBoxesForRange returns the boxes that enclose the text in the byte range.
	GetBoxesForRange(start, end int, heightStyle BoxHeightStyle, widthStyle BoxWidthStyle) []TextBox
	// GetBoxesForPlaceholders returns the boxes of the placeholders, in the order they were added.
	GetBoxesForPlaceholders() []TextBox
	// GetPositionForOffset returns the text position closest to the point.
	GetPositionForOffset(offset Offset) TextPosition
	// GetClosestGlyphInfoForOffset returns the grapheme cluster closest to the point.
	GetClosestGlyphInfoForOffset(offset Offset) (GlyphInfo, bool)
	// GetGlyphInfoAt returns the grapheme cluster that holds the byte offset.
	GetGlyphInfoAt(offset int) (GlyphInfo, bool)
	// GetWordBoundary returns the range of the word at the position.
	GetWordBoundary(position TextPosition) TextRange
	// GetLineBoundary returns the range of the line at the position, with its newline.
	GetLineBoundary(position TextPosition) TextRange
	// GetLineMetricsAt returns the metrics of one line.
	GetLineMetricsAt(lineNumber int) (LineMetrics, bool)
	// NumberOfLines is the number of visible lines.
	NumberOfLines() int
	// GetLineNumberAt returns the line that holds the byte offset.
	GetLineNumberAt(offset int) (int, bool)
}

// Paragraph is a paragraph of text that has been laid out, ready to be drawn with Canvas.DrawParagraph.
//
// Flutter: ui/text.dart Paragraph
type Paragraph struct {
	layout ParagraphLayout
}

// NewParagraph wraps the layout a text engine made.
//
// Flutter: no counterpart
func NewParagraph(layout ParagraphLayout) *Paragraph { return &Paragraph{layout} }

// Layout provides the size of the paragraph; the width is where lines wrap.
//
// Flutter: ui/text.dart Paragraph.layout
func (p *Paragraph) Layout(constraints ParagraphConstraints) { p.layout.Layout(constraints) }

// Width is the amount of horizontal space this paragraph occupies.
//
// Flutter: ui/text.dart Paragraph.width
func (p *Paragraph) Width() float64 { return p.layout.Width() }

// Height is the amount of vertical space this paragraph occupies.
//
// Flutter: ui/text.dart Paragraph.height
func (p *Paragraph) Height() float64 { return p.layout.Height() }

// LongestLine is the distance from the left edge of the leftmost glyph to the right edge of the rightmost glyph, over the longest line.
//
// Flutter: ui/text.dart Paragraph.longestLine
func (p *Paragraph) LongestLine() float64 { return p.layout.LongestLine() }

// MinIntrinsicWidth is the minimum width that this paragraph could be without failing to paint its contents within itself.
//
// Flutter: ui/text.dart Paragraph.minIntrinsicWidth
func (p *Paragraph) MinIntrinsicWidth() float64 { return p.layout.MinIntrinsicWidth() }

// MaxIntrinsicWidth is the width at which decreasing the width would not decrease the height.
//
// Flutter: ui/text.dart Paragraph.maxIntrinsicWidth
func (p *Paragraph) MaxIntrinsicWidth() float64 { return p.layout.MaxIntrinsicWidth() }

// AlphabeticBaseline is the distance from the top of the paragraph to the alphabetic baseline of the first line.
//
// Flutter: ui/text.dart Paragraph.alphabeticBaseline
func (p *Paragraph) AlphabeticBaseline() float64 { return p.layout.AlphabeticBaseline() }

// IdeographicBaseline is the distance from the top of the paragraph to the ideographic baseline of the first line.
//
// Flutter: ui/text.dart Paragraph.ideographicBaseline
func (p *Paragraph) IdeographicBaseline() float64 { return p.layout.IdeographicBaseline() }

// DidExceedMaxLines is true if there is more text than ParagraphStyle.MaxLines allows.
//
// Flutter: ui/text.dart Paragraph.didExceedMaxLines
func (p *Paragraph) DidExceedMaxLines() bool { return p.layout.DidExceedMaxLines() }

// ComputeLineMetrics returns the metrics of every line of the laid-out paragraph.
//
// Flutter: ui/text.dart Paragraph.computeLineMetrics
func (p *Paragraph) ComputeLineMetrics() []LineMetrics { return p.layout.ComputeLineMetrics() }

// GetBoxesForRange returns a list of text boxes that enclose the given text range (byte offsets). Coordinates are relative to the upper-left corner of the paragraph.
//
// Flutter: ui/text.dart Paragraph.getBoxesForRange
func (p *Paragraph) GetBoxesForRange(start, end int, opts ...BoxesForRangeOpts) []TextBox {
	o := oneOpts(opts)
	return p.layout.GetBoxesForRange(start, end, o.HeightStyle, o.WidthStyle)
}

// BoxesForRangeOpts are the optional parameters of Paragraph.GetBoxesForRange.
//
// Flutter: ui/text.dart Paragraph.getBoxesForRange
type BoxesForRangeOpts struct {
	// HeightStyle defaults to tight.
	HeightStyle BoxHeightStyle
	// WidthStyle defaults to tight.
	WidthStyle BoxWidthStyle
}

// GetBoxesForPlaceholders returns a list of text boxes that enclose all placeholders in the paragraph, in the order they were added.
//
// Flutter: ui/text.dart Paragraph.getBoxesForPlaceholders
func (p *Paragraph) GetBoxesForPlaceholders() []TextBox { return p.layout.GetBoxesForPlaceholders() }

// GetPositionForOffset returns the text position closest to the given offset. It always returns a position, even when the paragraph is empty.
//
// Flutter: ui/text.dart Paragraph.getPositionForOffset
func (p *Paragraph) GetPositionForOffset(offset Offset) TextPosition {
	return p.layout.GetPositionForOffset(offset)
}

// GetClosestGlyphInfoForOffset returns the glyph closest to the given offset, or false if the text is empty or entirely clipped.
//
// Flutter: ui/text.dart Paragraph.getClosestGlyphInfoForOffset
func (p *Paragraph) GetClosestGlyphInfoForOffset(offset Offset) (GlyphInfo, bool) {
	return p.layout.GetClosestGlyphInfoForOffset(offset)
}

// GetGlyphInfoAt returns the glyph located at the given byte offset, or false if the offset is out of the visible lines or is ellipsized.
//
// Flutter: ui/text.dart Paragraph.getGlyphInfoAt
func (p *Paragraph) GetGlyphInfoAt(offset int) (GlyphInfo, bool) {
	return p.layout.GetGlyphInfoAt(offset)
}

// GetWordBoundary returns the range of the word at the given position. Characters that are not part of a word have word breaks on both sides: the range is that one character.
//
// Flutter: ui/text.dart Paragraph.getWordBoundary
func (p *Paragraph) GetWordBoundary(position TextPosition) TextRange {
	return p.layout.GetWordBoundary(position)
}

// GetLineBoundary returns the range of the line at the given position. The newline, if any, is part of the range.
//
// Flutter: ui/text.dart Paragraph.getLineBoundary
func (p *Paragraph) GetLineBoundary(position TextPosition) TextRange {
	return p.layout.GetLineBoundary(position)
}

// GetLineMetricsAt returns the metrics of the line, or false if lineNumber is not a visible line.
//
// Flutter: ui/text.dart Paragraph.getLineMetricsAt
func (p *Paragraph) GetLineMetricsAt(lineNumber int) (LineMetrics, bool) {
	return p.layout.GetLineMetricsAt(lineNumber)
}

// NumberOfLines is the total number of visible lines in the paragraph.
//
// Flutter: ui/text.dart Paragraph.numberOfLines
func (p *Paragraph) NumberOfLines() int { return p.layout.NumberOfLines() }

// GetLineNumberAt returns the number of the line that holds the byte at the offset, or false if the offset is out of bounds or after the last visible character.
//
// Flutter: ui/text.dart Paragraph.getLineNumberAt
func (p *Paragraph) GetLineNumberAt(offset int) (int, bool) { return p.layout.GetLineNumberAt(offset) }

// Layouter returns the layout the paragraph wraps, for the renderers and the text queries.
//
// Flutter: no counterpart
func (p *Paragraph) Layouter() ParagraphLayout { return p.layout }

// equalSlices reports whether two slices hold equal elements.
func equalSlices[T comparable](a, b []T) bool { return slices.Equal(a, b) }
