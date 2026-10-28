package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/rm4n0s/errors"
)

// Defaults of the text engine (the values of txt::TextStyle and txt::ParagraphStyle).
const (
	// defaultFontSize is the font size when no style gives one.
	defaultFontSize = 14.0
	// defaultFontWeight is the weight when no style gives one.
	defaultFontWeight = FontWeightW400
)

// TextStyleOpts are the optional parameters of NewTextStyle. A field left at its zero value (or nil) is not set, and a style pushed on a ParagraphBuilder inherits it from the style under it.
//
// Flutter: ui/text.dart TextStyle
type TextStyleOpts struct {
	// Color is the color to use when painting the text; it must not be given together with Foreground.
	Color *Color
	// Decoration is the decorations to paint near the text (underline, overline, line through).
	Decoration *TextDecoration
	// DecorationColor is the color in which to paint the text decorations; it defaults to the text color.
	DecorationColor *Color
	// DecorationStyle is the style in which to paint the text decorations.
	DecorationStyle TextDecorationStyle
	// DecorationThickness is how many times the default thickness of the decoration to draw it with.
	DecorationThickness *float64
	// FontWeight is the typeface thickness to use when painting the text.
	FontWeight FontWeight
	// FontStyle is the typeface variant to use when drawing the letters.
	FontStyle FontStyle
	// TextBaseline is the baseline to use for aligning the text.
	TextBaseline TextBaseline
	// FontFamily is the name of the font to use when painting the text.
	FontFamily string
	// FontFamilyFallback is the ordered list of font families to fall back on when a glyph cannot be found in a higher priority font.
	FontFamilyFallback []string
	// FontSize is the size of glyphs (in logical pixels) to use when painting the text.
	FontSize *float64
	// LetterSpacing is the amount of space (in logical pixels) to add between each letter.
	LetterSpacing *float64
	// WordSpacing is the amount of space (in logical pixels) to add at each sequence of white-space.
	WordSpacing *float64
	// Height is the height of the line as a multiple of the font size; nil uses the font's own line height.
	Height *float64
	// LeadingDistribution is how the extra space of the line height is divided above and below the text.
	LeadingDistribution TextLeadingDistribution
	// Locale is the locale used to select region-specific glyphs.
	Locale *Locale
	// Background is the paint drawn as a background for the text.
	Background *Paint
	// Foreground is the paint used to draw the text; it must not be given together with Color.
	Foreground *Paint
	// Shadows is a list of Shadows that will be painted underneath the text.
	Shadows []Shadow
	// FontFeatures is a list of FontFeatures that affect how the font selects glyphs.
	FontFeatures []FontFeature
	// FontVariations is a list of FontVariations that affect how a variable font is rendered.
	FontVariations []FontVariation
}

// TextStyle is an immutable style describing how to format and paint text. Fields that were not set inherit from the style under it when it is pushed on a ParagraphBuilder.
//
// Flutter: ui/text.dart TextStyle
type TextStyle struct {
	o TextStyleOpts
}

// NewTextStyle creates a text style. Passing both a Color and a Foreground panics with the tag TextStyleColorAndForeground.
//
// Flutter: ui/text.dart TextStyle
func NewTextStyle(opts ...TextStyleOpts) *TextStyle {
	o := oneOpts(opts)
	if o.Color != nil && o.Foreground != nil {
		panic(errors.New(tagTextStyleColorAndForeground,
			`Cannot provide both a color and a foreground. The color argument is just a shorthand for "foreground: Paint()..color = color".`))
	}
	o.FontFamilyFallback = append([]string(nil), o.FontFamilyFallback...)
	o.Shadows = append([]Shadow(nil), o.Shadows...)
	o.FontFeatures = append([]FontFeature(nil), o.FontFeatures...)
	o.FontVariations = append([]FontVariation(nil), o.FontVariations...)
	return &TextStyle{o}
}

// Options returns the options the style was made from. The slices are copies.
//
// Flutter: no counterpart
func (s *TextStyle) Options() TextStyleOpts {
	o := s.o
	o.FontFamilyFallback = append([]string(nil), o.FontFamilyFallback...)
	o.Shadows = append([]Shadow(nil), o.Shadows...)
	o.FontFeatures = append([]FontFeature(nil), o.FontFeatures...)
	o.FontVariations = append([]FontVariation(nil), o.FontVariations...)
	return o
}

// ResolvedTextStyle is a text style with every field decided: what the layout and the renderers use.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/text_style.h TextStyle
type ResolvedTextStyle struct {
	// Color is the color of the text, when Foreground is nil.
	Color Color
	// Foreground, when not nil, is the paint of the text instead of Color.
	Foreground *Paint
	// Background, when not nil, is the paint of the box behind the text.
	Background *Paint
	// Decoration is the decorations to draw.
	Decoration TextDecoration
	// DecorationColor is the color of the decorations when HasDecorationColor; else the text color.
	DecorationColor Color
	// HasDecorationColor says whether DecorationColor was set.
	HasDecorationColor bool
	// DecorationStyle is the line style of the decorations.
	DecorationStyle TextDecorationStyle
	// DecorationThickness is the multiple of the font's default decoration thickness.
	DecorationThickness float64
	// FontWeight is the weight of the glyphs.
	FontWeight FontWeight
	// FontStyle is upright or italic.
	FontStyle FontStyle
	// TextBaseline is the baseline the text aligns on.
	TextBaseline TextBaseline
	// FontFamilies is the font family followed by its fallbacks, in priority order; empty means the system default.
	FontFamilies []string
	// FontSize is the size of the glyphs in logical pixels.
	FontSize float64
	// LetterSpacing is the extra space after each letter.
	LetterSpacing float64
	// WordSpacing is the extra space at each white space.
	WordSpacing float64
	// Height is the multiple of the font size that the line takes, used when HasHeight.
	Height float64
	// HasHeight says whether the height was set; if not, the font's own line height applies.
	HasHeight bool
	// LeadingDistribution is how the extra line space is divided.
	LeadingDistribution TextLeadingDistribution
	// Locale selects region-specific glyphs.
	Locale Locale
	// Shadows are drawn underneath the text.
	Shadows []Shadow
	// FontFeatures are the OpenType features to apply.
	FontFeatures []FontFeature
	// FontVariations are the variable-font axis values to apply.
	FontVariations []FontVariation
}

// DefaultResolvedTextStyle is the style a paragraph starts from: black, 14 logical pixels, weight 400.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/text_style.cc TextStyle::TextStyle
func DefaultResolvedTextStyle() ResolvedTextStyle {
	return ResolvedTextStyle{
		Color:               NewColor(0xFF000000),
		DecorationStyle:     TextDecorationStyleSolid,
		DecorationThickness: 1,
		FontWeight:          defaultFontWeight,
		FontStyle:           FontStyleNormal,
		TextBaseline:        TextBaselineAlphabetic,
		FontSize:            defaultFontSize,
		LeadingDistribution: TextLeadingDistributionProportional,
	}
}

// Resolve applies the style on top of parent: every field the style set replaces the parent's.
//
// Flutter: engine/src/flutter/lib/ui/text/paragraph_builder.cc ParagraphBuilder::pushStyle
func (s *TextStyle) Resolve(parent ResolvedTextStyle) ResolvedTextStyle {
	r := parent
	o := &s.o
	if o.Color != nil {
		r.Color = *o.Color
		r.Foreground = nil
	}
	if o.Foreground != nil {
		r.Foreground = NewPaintFrom(o.Foreground)
	}
	if o.Background != nil {
		r.Background = NewPaintFrom(o.Background)
	}
	if o.Decoration != nil {
		r.Decoration = *o.Decoration
	}
	if o.DecorationColor != nil {
		r.DecorationColor, r.HasDecorationColor = *o.DecorationColor, true
	}
	if o.DecorationStyle != TextDecorationStyleUnspecified {
		r.DecorationStyle = o.DecorationStyle
	}
	if o.DecorationThickness != nil {
		r.DecorationThickness = *o.DecorationThickness
	}
	if o.FontWeight != 0 {
		r.FontWeight = o.FontWeight
	}
	if o.FontStyle != FontStyleUnspecified {
		r.FontStyle = o.FontStyle
	}
	if o.TextBaseline != TextBaselineUnspecified {
		r.TextBaseline = o.TextBaseline
	}
	if o.FontFamily != "" || len(o.FontFamilyFallback) > 0 {
		r.FontFamilies = nil
		if o.FontFamily != "" {
			r.FontFamilies = append(r.FontFamilies, o.FontFamily)
		}
		r.FontFamilies = append(r.FontFamilies, o.FontFamilyFallback...)
	}
	if o.FontSize != nil {
		r.FontSize = *o.FontSize
	}
	if o.LetterSpacing != nil {
		r.LetterSpacing = *o.LetterSpacing
	}
	if o.WordSpacing != nil {
		r.WordSpacing = *o.WordSpacing
	}
	if o.Height != nil {
		r.Height, r.HasHeight = *o.Height, true
	}
	if o.LeadingDistribution != TextLeadingDistributionUnspecified {
		r.LeadingDistribution = o.LeadingDistribution
	}
	if o.Locale != nil {
		r.Locale = *o.Locale
	}
	if o.Shadows != nil {
		r.Shadows = append([]Shadow(nil), o.Shadows...)
	}
	if o.FontFeatures != nil {
		r.FontFeatures = append([]FontFeature(nil), o.FontFeatures...)
	}
	if o.FontVariations != nil {
		r.FontVariations = append([]FontVariation(nil), o.FontVariations...)
	}
	return r
}

// String returns a description of the style's set fields.
func (s *TextStyle) String() string {
	o := &s.o
	var parts []string
	add := func(name string, v any) { parts = append(parts, fmt.Sprintf("%s: %v", name, v)) }
	if o.Color != nil {
		add("color", *o.Color)
	}
	if o.Decoration != nil {
		add("decoration", *o.Decoration)
	}
	if o.FontWeight != 0 {
		add("fontWeight", o.FontWeight)
	}
	if o.FontStyle != FontStyleUnspecified {
		add("fontStyle", o.FontStyle)
	}
	if o.FontFamily != "" {
		add("fontFamily", o.FontFamily)
	}
	if o.FontSize != nil {
		add("fontSize", *o.FontSize)
	}
	if o.Height != nil {
		add("height", fmt.Sprintf("%vx", *o.Height))
	}
	return "TextStyle(" + strings.Join(parts, ", ") + ")"
}

// StrutStyleOpts are the optional parameters of NewStrutStyle.
//
// Flutter: ui/text.dart StrutStyle
type StrutStyleOpts struct {
	// FontFamily is the name of the font to use when painting the strut.
	FontFamily string
	// FontFamilyFallback is the ordered list of font families to fall back on.
	FontFamilyFallback []string
	// FontSize is the size of the font of the strut, in logical pixels.
	FontSize *float64
	// Height is the height of the strut as a multiple of its font size.
	Height *float64
	// LeadingDistribution is how the extra space of the height is divided.
	LeadingDistribution TextLeadingDistribution
	// Leading is the additional leading as a multiple of the font size, half above and half below.
	Leading *float64
	// FontWeight is the weight of the strut's font.
	FontWeight FontWeight
	// FontStyle is the style of the strut's font.
	FontStyle FontStyle
	// ForceStrutHeight, when true, makes every line exactly the strut's height.
	ForceStrutHeight *bool
}

// StrutStyle defines the strut, which sets the minimum vertical layout metrics of a paragraph's lines.
//
// Flutter: ui/text.dart StrutStyle
type StrutStyle struct {
	o StrutStyleOpts
}

// NewStrutStyle creates a strut style.
//
// Flutter: ui/text.dart StrutStyle
func NewStrutStyle(opts ...StrutStyleOpts) *StrutStyle {
	o := oneOpts(opts)
	o.FontFamilyFallback = append([]string(nil), o.FontFamilyFallback...)
	return &StrutStyle{o}
}

// Options returns the options the strut was made from.
//
// Flutter: no counterpart
func (s *StrutStyle) Options() StrutStyleOpts {
	o := s.o
	o.FontFamilyFallback = append([]string(nil), o.FontFamilyFallback...)
	return o
}

// ResolvedStrut is a strut with every field decided.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/paragraph_style.h StrutStyle
type ResolvedStrut struct {
	// Enabled says whether the paragraph has a strut at all.
	Enabled bool
	// FontFamilies is the family followed by its fallbacks; empty means the paragraph's.
	FontFamilies []string
	// FontWeight is the weight of the strut's font.
	FontWeight FontWeight
	// FontStyle is the style of the strut's font.
	FontStyle FontStyle
	// FontSize is the size of the strut's font.
	FontSize float64
	// Height is the multiple of the font size the strut takes, used when HasHeight.
	Height float64
	// HasHeight says whether the height was set.
	HasHeight bool
	// LeadingDistribution is how the extra space of the height is divided.
	LeadingDistribution TextLeadingDistribution
	// Leading is the extra leading as a multiple of the font size; -1 when not set.
	Leading float64
	// Force says whether every line takes the strut's height.
	Force bool
}

// ParagraphStyleOpts are the optional parameters of NewParagraphStyle.
//
// Flutter: ui/text.dart ParagraphStyle
type ParagraphStyleOpts struct {
	// TextAlign is the alignment of the text within the lines of the paragraph.
	TextAlign TextAlign
	// TextDirection is the directionality of the text, which also decides what start and end mean.
	TextDirection TextDirection
	// MaxLines is the maximum number of lines to show; nil means no limit.
	MaxLines *int
	// FontFamily is the name of the font to use when painting the text.
	FontFamily string
	// FontSize is the size of glyphs (in logical pixels) to use when painting the text.
	FontSize *float64
	// Height is the height of the line as a multiple of the font size.
	Height *float64
	// TextHeightBehavior says how Height applies to the first ascent and the last descent.
	TextHeightBehavior *TextHeightBehavior
	// FontWeight is the typeface thickness to use when painting the text.
	FontWeight FontWeight
	// FontStyle is the typeface variant to use when drawing the letters.
	FontStyle FontStyle
	// StrutStyle is the strut of the paragraph.
	StrutStyle *StrutStyle
	// Ellipsis replaces the text that does not fit when MaxLines is exceeded; "" means none.
	Ellipsis string
	// Locale is the locale used to select region-specific glyphs.
	Locale *Locale
	// TabSize is the distance between two tab stops, as a count of the advance of a space in the
	// font of the paragraph style (CSS's tab-size). A tab advances to the next multiple of it,
	// measured from the start of its line. 0, the default, gives tabs no stops, as in Flutter.
	//
	// Flutter: no counterpart
	TabSize float64
}

// ParagraphStyle describes the whole paragraph: alignment, direction, maximum lines, ellipsis, strut, and the defaults of the text style.
//
// Flutter: ui/text.dart ParagraphStyle
type ParagraphStyle struct {
	o ParagraphStyleOpts
}

// NewParagraphStyle creates a paragraph style. A negative MaxLines panics with the tag MaxLinesNegative, and a TabSize that is negative, NaN or infinite with the tag TabSizeInvalid.
//
// Flutter: ui/text.dart ParagraphStyle
func NewParagraphStyle(opts ...ParagraphStyleOpts) *ParagraphStyle {
	o := oneOpts(opts)
	if o.MaxLines != nil && *o.MaxLines < 0 {
		panic(errors.New(tagMaxLinesNegative, "the maximum number of lines must not be negative", "maxLines", *o.MaxLines))
	}
	if o.TabSize < 0 || math.IsNaN(o.TabSize) || math.IsInf(o.TabSize, 0) {
		panic(errors.New(tagTabSizeInvalid, "the tab size must be zero or a positive number", "tabSize", o.TabSize))
	}
	return &ParagraphStyle{o}
}

// Options returns the options the style was made from.
//
// Flutter: no counterpart
func (s *ParagraphStyle) Options() ParagraphStyleOpts { return s.o }

// Direction is the text direction, left to right when unspecified.
//
// Flutter: no counterpart
func (s *ParagraphStyle) Direction() TextDirection {
	if s == nil || s.o.TextDirection == TextDirectionUnspecified {
		return TextDirectionLTR
	}
	return s.o.TextDirection
}

// Align is the text alignment, start when unspecified.
//
// Flutter: no counterpart
func (s *ParagraphStyle) Align() TextAlign {
	if s == nil || s.o.TextAlign == TextAlignUnspecified {
		return TextAlignStart
	}
	return s.o.TextAlign
}

// MaxLines is the limit of lines and whether there is one.
//
// Flutter: no counterpart
func (s *ParagraphStyle) MaxLines() (int, bool) {
	if s == nil || s.o.MaxLines == nil {
		return 0, false
	}
	return *s.o.MaxLines, true
}

// TabSize is the distance between tab stops in spaces of the paragraph's font; 0 for no tab stops.
//
// Flutter: no counterpart
func (s *ParagraphStyle) TabSize() float64 {
	if s == nil {
		return 0
	}
	return s.o.TabSize
}

// Ellipsis is the ellipsis string, "" for none.
//
// Flutter: no counterpart
func (s *ParagraphStyle) Ellipsis() string {
	if s == nil {
		return ""
	}
	return s.o.Ellipsis
}

// HeightBehavior is the text height behavior, with the defaults applied.
//
// Flutter: no counterpart
func (s *ParagraphStyle) HeightBehavior() TextHeightBehavior {
	b := NewTextHeightBehavior()
	if s != nil && s.o.TextHeightBehavior != nil {
		b = *s.o.TextHeightBehavior
	}
	b.LeadingDistribution = b.distribution()
	return b
}

// BaseTextStyle is the text style a paragraph starts from: the defaults with the paragraph style's font and locale applied.
//
// Flutter: engine/src/flutter/lib/ui/text/paragraph_builder.cc ParagraphBuilder::ParagraphBuilder
func (s *ParagraphStyle) BaseTextStyle() ResolvedTextStyle {
	r := DefaultResolvedTextStyle()
	if s == nil {
		return r
	}
	o := &s.o
	if o.FontWeight != 0 {
		r.FontWeight = o.FontWeight
	}
	if o.FontStyle != FontStyleUnspecified {
		r.FontStyle = o.FontStyle
	}
	if o.FontFamily != "" {
		r.FontFamilies = []string{o.FontFamily}
	}
	if o.FontSize != nil {
		r.FontSize = *o.FontSize
	}
	if o.Height != nil {
		r.Height, r.HasHeight = *o.Height, true
	}
	if o.Locale != nil {
		r.Locale = *o.Locale
	}
	r.LeadingDistribution = s.HeightBehavior().LeadingDistribution
	return r
}

// Strut is the strut with its fields decided; Enabled is false when the paragraph has none.
//
// Flutter: engine/src/flutter/lib/ui/text/paragraph_builder.cc decodeStrut
func (s *ParagraphStyle) Strut() ResolvedStrut {
	r := ResolvedStrut{Leading: -1, LeadingDistribution: TextLeadingDistributionProportional}
	if s == nil || s.o.StrutStyle == nil {
		return r
	}
	o := &s.o.StrutStyle.o
	hasHeight := o.Height != nil
	if o.FontFamily == "" && len(o.FontFamilyFallback) == 0 && o.FontSize == nil && !hasHeight &&
		o.LeadingDistribution == TextLeadingDistributionUnspecified && o.Leading == nil && o.FontWeight == 0 &&
		o.FontStyle == FontStyleUnspecified && o.ForceStrutHeight == nil {
		return r
	}
	r.Enabled = true
	if o.FontFamily != "" {
		r.FontFamilies = append(r.FontFamilies, o.FontFamily)
	}
	r.FontFamilies = append(r.FontFamilies, o.FontFamilyFallback...)
	r.FontWeight, r.FontStyle = o.FontWeight, o.FontStyle
	if r.FontWeight == 0 {
		r.FontWeight = defaultFontWeight
	}
	if r.FontStyle == FontStyleUnspecified {
		r.FontStyle = FontStyleNormal
	}
	r.FontSize = defaultFontSize
	if o.FontSize != nil {
		r.FontSize = *o.FontSize
	}
	if hasHeight {
		r.Height, r.HasHeight = *o.Height, true
	}
	if o.LeadingDistribution != TextLeadingDistributionUnspecified {
		r.LeadingDistribution = o.LeadingDistribution
	}
	if o.Leading != nil {
		r.Leading = *o.Leading
	}
	r.Force = o.ForceStrutHeight != nil && *o.ForceStrutHeight
	return r
}

// TextRun is a stretch of the paragraph's text in one style.
//
// Flutter: no counterpart
type TextRun struct {
	// Start and End are the UTF-8 byte offsets of the text, End exclusive.
	Start, End int
	// Style is the style of the text. Runs that came from one PushStyle share it, and so does a
	// paragraph with the runs of its content: nothing changes what it points to.
	Style *ResolvedTextStyle
}

// PlaceholderSpec is a box in the text that the paragraph leaves room for: an inline widget.
//
// Flutter: no counterpart
type PlaceholderSpec struct {
	// Offset is the byte offset of the U+FFFC character that stands for the placeholder in Text.
	Offset int
	// Width and Height are the size of the box, in logical pixels, before Scale.
	Width, Height float64
	// Alignment is how the box aligns vertically with the text.
	Alignment PlaceholderAlignment
	// Scale multiplies Width, Height and BaselineOffset.
	Scale float64
	// BaselineOffset is the distance from the top of the box to its baseline, used for the baseline alignments.
	BaselineOffset float64
	// Baseline is the text baseline that BaselineOffset aligns with.
	Baseline TextBaseline
}

// ParagraphContent is what a ParagraphBuilder collected: the style and the styled text. A text engine lays it out.
//
// Flutter: no counterpart
type ParagraphContent struct {
	// Style is the paragraph style.
	Style *ParagraphStyle
	// Text is the whole text, in UTF-8.
	Text string
	// Runs cover Text in order, with no gaps; there is none for an empty text.
	Runs []TextRun
	// Placeholders are the boxes in Text, in order.
	Placeholders []PlaceholderSpec
}

// ParagraphFactory is the text engine: it makes a laid-out paragraph from its content. internal/text/layout implements it. The framework reaches it through the PlatformDispatcher, so there is no global font collection.
//
// Flutter: no counterpart
type ParagraphFactory interface {
	// BuildParagraph shapes the content. The paragraph must be laid out before it is used.
	BuildParagraph(content *ParagraphContent) ParagraphLayout
}

// FontLoadingEngine is a ParagraphFactory that can take fonts at run time.
// internal/text/layout's Engine implements it; PlatformDispatcher.LoadFontFromList
// uses it.
//
// Flutter: no counterpart
type FontLoadingEngine interface {
	ParagraphFactory
	// LoadFontFromList adds the font in list (a TrueType or OpenType file) to the
	// family, or to the family named in the font when family is empty.
	LoadFontFromList(list []byte, family string) error
}

// FontOrigin says where a font comes from. The zero value is FontOriginUnspecified.
//
// Flutter: no counterpart
type FontOrigin int

// The values of FontOrigin.
const (
	// FontOriginUnspecified is the zero value: no font.
	//
	// Flutter: no counterpart
	FontOriginUnspecified FontOrigin = iota
	// FontOriginLoaded is a font that the app loaded (LoadFontFromList, FontLoader,
	// RunAppOpts.Fonts, gluttertest.Opts.Fonts).
	//
	// Flutter: no counterpart
	FontOriginLoaded
	// FontOriginBundled is a font that Glutter bundles (Roboto, Material Icons).
	//
	// Flutter: no counterpart
	FontOriginBundled
	// FontOriginSystem is a font file of the system.
	//
	// Flutter: no counterpart
	FontOriginSystem
)

// ResolvedFont is the face that a font family resolves to (PlatformDispatcher.ResolveFont).
//
// Flutter: no counterpart
type ResolvedFont struct {
	// Family is the family name of the face, which can differ from the name asked for
	// (a generic name such as monospace, or an alias such as Arial).
	Family string
	// Path is the font file, and Index the face in it (a font collection has several).
	// Both are empty for a loaded or bundled font, which has no file.
	Path  string
	Index int
	// Origin says whether the font was loaded by the app, is bundled, or is a system font.
	Origin FontOrigin
	// Weight is the OpenType weight of the face (400 regular, 700 bold) and Italic whether it is
	// italic, as the font says; the weight of a variable font is that of its default instance.
	Weight int
	Italic bool
}

// FontQueryEngine is a ParagraphFactory that can say which fonts it has.
// internal/text/layout's Engine implements it; PlatformDispatcher.ResolveFont and
// InstalledFontFamilies use it.
//
// Flutter: no counterpart
type FontQueryEngine interface {
	ParagraphFactory
	// ResolveFont says which face the family resolves to for a weight and a slant,
	// the one that draws a letter of a TextStyle that names only that family; false when
	// the family has no font.
	ResolveFont(family string, weight FontWeight, style FontStyle) (ResolvedFont, bool)
	// InstalledFontFamilies lists the families of the system's, bundled and loaded fonts.
	InstalledFontFamilies() []string
}

// FontMetrics are the vertical metrics and the width of a digit of the font that a text style
// draws with, at the style's size, in logical pixels (PlatformDispatcher.FontMetrics). They are what
// a paragraph of one line in that style measures, so a grid of cells (a terminal) can be sized from them
// without laying out and measuring a paragraph.
//
// Flutter: no counterpart
type FontMetrics struct {
	// Ascent and Descent are the distances from the baseline up to the top and down to the bottom of a
	// line in the style, as the paragraph's line metrics (ComputeLineMetrics) give them: each includes
	// half of Leading, and the style's Height, if it has one, is applied. The baseline of a one-line
	// paragraph is Ascent below its top.
	Ascent, Descent float64
	// Leading is the line gap of the font (the space it asks between two lines), at the size.
	Leading float64
	// Height is the height of a paragraph of one line in the style: Ascent and Descent added and rounded.
	Height float64
	// ZeroAdvance is the advance of the digit "0": the width of a cell of a grid of a monospaced font.
	ZeroAdvance float64
	// Monospaced says that the font says it is monospaced (the post table's fixed pitch flag, or all its
	// glyphs having the same advance, as fontconfig decides). It is false when the font does not say.
	Monospaced bool
}

// FontMetricsEngine is a ParagraphFactory that can measure the font of a style without a paragraph of the
// caller's. internal/text/layout's Engine implements it; PlatformDispatcher.FontMetrics uses it.
//
// Flutter: no counterpart
type FontMetricsEngine interface {
	ParagraphFactory
	// FontMetrics measures the face that the style draws the digit "0" with, the way a paragraph of
	// one line in that style does.
	FontMetrics(style *ResolvedTextStyle) FontMetrics
}

// ParagraphBuilder is used to construct a Paragraph.
//
// Flutter: ui/text.dart ParagraphBuilder
type ParagraphBuilder struct {
	factory ParagraphFactory
	style   *ParagraphStyle
	stack   []*ResolvedTextStyle
	text    strings.Builder
	runs    []TextRun

	placeholders []PlaceholderSpec
}

// NewParagraphBuilder creates a ParagraphBuilder for the text engine and the paragraph style. The Flutter constructor takes only the style, because Dart has a global font collection; here the engine is a parameter.
//
// Flutter: ui/text.dart ParagraphBuilder
func NewParagraphBuilder(factory ParagraphFactory, style *ParagraphStyle) *ParagraphBuilder {
	base := style.BaseTextStyle()
	return &ParagraphBuilder{factory: factory, style: style, stack: []*ResolvedTextStyle{&base}}
}

// PushStyle applies the text style to text added after, until the matching Pop. Fields the style did not set inherit from the style below.
//
// Flutter: ui/text.dart ParagraphBuilder.pushStyle
func (b *ParagraphBuilder) PushStyle(style *TextStyle) {
	resolved := style.Resolve(*b.stack[len(b.stack)-1])
	b.stack = append(b.stack, &resolved)
}

// Pop ends the effect of the most recent PushStyle. Popping with no style pushed panics with the tag ParagraphBuilderPopEmpty.
//
// Flutter: ui/text.dart ParagraphBuilder.pop
func (b *ParagraphBuilder) Pop() {
	if len(b.stack) <= 1 {
		panic(errors.New(tagParagraphBuilderPopEmpty, "Pop was called with no style pushed"))
	}
	b.stack = b.stack[:len(b.stack)-1]
}

// AddText adds the text, in the current style.
//
// Flutter: ui/text.dart ParagraphBuilder.addText
func (b *ParagraphBuilder) AddText(text string) {
	if text == "" {
		return
	}
	start := b.text.Len()
	b.text.WriteString(text)
	style := b.stack[len(b.stack)-1]
	if n := len(b.runs); n > 0 && (b.runs[n-1].Style == style || sameStyle(b.runs[n-1].Style, style)) {
		b.runs[n-1].End = b.text.Len()
		return
	}
	b.runs = append(b.runs, TextRun{Start: start, End: b.text.Len(), Style: style})
}

// PlaceholderCount is the number of placeholders added so far.
//
// Flutter: ui/text.dart ParagraphBuilder.placeholderCount
func (b *ParagraphBuilder) PlaceholderCount() int { return len(b.placeholders) }

// PlaceholderScales is the scale of each placeholder added so far.
//
// Flutter: ui/text.dart ParagraphBuilder.placeholderScales
func (b *ParagraphBuilder) PlaceholderScales() []float64 {
	out := make([]float64, len(b.placeholders))
	for i, p := range b.placeholders {
		out[i] = p.Scale
	}
	return out
}

// AddPlaceholderOpts are the optional parameters of ParagraphBuilder.AddPlaceholder.
//
// Flutter: ui/text.dart ParagraphBuilder.addPlaceholder
type AddPlaceholderOpts struct {
	// Scale multiplies the size and baseline offset; it defaults to 1.
	Scale *float64
	// BaselineOffset is the distance from the top of the box to its baseline; it defaults to the height (the bottom edge).
	BaselineOffset *float64
	// Baseline is the text baseline BaselineOffset refers to; it defaults to alphabetic.
	Baseline TextBaseline
}

// AddPlaceholder reserves a box of the given size in the text. The paragraph lays out the text around it, and GetBoxesForPlaceholders reports where it is. A negative size panics with the tag PlaceholderSizeInvalid.
//
// Flutter: ui/text.dart ParagraphBuilder.addPlaceholder
func (b *ParagraphBuilder) AddPlaceholder(width, height float64, alignment PlaceholderAlignment, opts ...AddPlaceholderOpts) {
	o := oneOpts(opts)
	if width < 0 || height < 0 {
		panic(errors.New(tagPlaceholderSizeInvalid, "the size of a placeholder must not be negative", "width", width, "height", height))
	}
	scale := 1.0
	if o.Scale != nil {
		scale = *o.Scale
	}
	offset := height
	if o.BaselineOffset != nil {
		offset = *o.BaselineOffset
	}
	if alignment == PlaceholderAlignmentUnspecified {
		alignment = PlaceholderAlignmentBaseline
	}
	baseline := o.Baseline
	if baseline == TextBaselineUnspecified {
		baseline = TextBaselineAlphabetic
	}
	b.placeholders = append(b.placeholders, PlaceholderSpec{Offset: b.text.Len(), Width: width, Height: height, Alignment: alignment,
		Scale: scale, BaselineOffset: offset, Baseline: baseline})
	b.AddText("\uFFFC")
}

// Build ends the builder and returns the paragraph. It must be laid out with Layout before it is used.
// The builder is not to be used after it: the paragraph keeps its runs and placeholders.
//
// Flutter: ui/text.dart ParagraphBuilder.build
func (b *ParagraphBuilder) Build() *Paragraph {
	content := &ParagraphContent{Style: b.style, Text: b.text.String(), Runs: b.runs, Placeholders: b.placeholders}
	b.runs, b.placeholders = nil, nil
	return NewParagraph(b.factory.BuildParagraph(content))
}

// sameStyle says whether two resolved styles are equal, so adjacent text of one style is one run.
func sameStyle(a, b *ResolvedTextStyle) bool {
	if a.Color != b.Color || a.Decoration != b.Decoration || a.DecorationColor != b.DecorationColor ||
		a.HasDecorationColor != b.HasDecorationColor || a.DecorationStyle != b.DecorationStyle ||
		a.DecorationThickness != b.DecorationThickness || a.FontWeight != b.FontWeight || a.FontStyle != b.FontStyle ||
		a.TextBaseline != b.TextBaseline || a.FontSize != b.FontSize || a.LetterSpacing != b.LetterSpacing ||
		a.WordSpacing != b.WordSpacing || a.Height != b.Height || a.HasHeight != b.HasHeight ||
		a.LeadingDistribution != b.LeadingDistribution || a.Locale != b.Locale {
		return false
	}
	if a.Foreground != b.Foreground || a.Background != b.Background {
		return false
	}
	return equalSlices(a.FontFamilies, b.FontFamilies) && equalSlices(a.Shadows, b.Shadows) &&
		equalSlices(a.FontFeatures, b.FontFeatures) && equalSlices(a.FontVariations, b.FontVariations)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMaxLinesNegative            = "MaxLinesNegative"
	tagParagraphBuilderPopEmpty    = "ParagraphBuilderPopEmpty"
	tagPlaceholderSizeInvalid      = "PlaceholderSizeInvalid"
	tagTabSizeInvalid              = "TabSizeInvalid"
	tagTextStyleColorAndForeground = "TextStyleColorAndForeground"
)
