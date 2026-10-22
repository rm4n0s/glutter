package painting

import (
	"math"
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// DefaultFontSize is the font size of text that doesn't give one, in logical
// pixels.
//
// Flutter: painting/text_painter.dart kDefaultFontSize
const DefaultFontSize = 14.0

// TextOverflow is how overflowing text should be handled. The zero value is
// TextOverflowUnspecified: Dart's null.
//
// Flutter: painting/text_painter.dart TextOverflow
type TextOverflow int

// The values of TextOverflow follow Flutter's declaration order.
const (
	// TextOverflowUnspecified is the zero value: null, the widget's default.
	TextOverflowUnspecified TextOverflow = iota
	// TextOverflowClip clips the overflowing text to fix its container.
	//
	// Flutter: painting/text_painter.dart TextOverflow.clip
	TextOverflowClip
	// TextOverflowFade fades the overflowing text to transparent.
	//
	// Flutter: painting/text_painter.dart TextOverflow.fade
	TextOverflowFade
	// TextOverflowEllipsis uses an ellipsis to indicate that the text has
	// overflowed.
	//
	// Flutter: painting/text_painter.dart TextOverflow.ellipsis
	TextOverflowEllipsis
	// TextOverflowVisible renders overflowing text outside of its container.
	//
	// Flutter: painting/text_painter.dart TextOverflow.visible
	TextOverflowVisible
)

// String returns the name Dart prints, such as "TextOverflow.clip".
func (o TextOverflow) String() string { return "TextOverflow." + o.Name() }

// Name returns the Dart enum name, such as "clip".
//
// Flutter: painting/text_painter.dart TextOverflow.name
func (o TextOverflow) Name() string {
	switch o {
	case TextOverflowClip:
		return "clip"
	case TextOverflowFade:
		return "fade"
	case TextOverflowEllipsis:
		return "ellipsis"
	case TextOverflowVisible:
		return "visible"
	}
	return "unspecified"
}

// PlaceholderDimensions is the dimensions of a placeholder (the box of a
// WidgetSpan) that TextPainter lays out around.
//
// Flutter: painting/text_painter.dart PlaceholderDimensions
type PlaceholderDimensions struct {
	// Size is the size of the placeholder.
	Size ui.Size
	// Alignment is how the placeholder aligns with the text.
	Alignment ui.PlaceholderAlignment
	// Baseline is the baseline the placeholder aligns on.
	Baseline ui.TextBaseline
	// BaselineOffset is the distance from the top of the box to its baseline.
	BaselineOffset foundation.Opt[float64]
}

// PlaceholderDimensionsEmpty is the dimensions of a placeholder that takes no
// space.
//
// Flutter: painting/text_painter.dart PlaceholderDimensions.empty
func PlaceholderDimensionsEmpty() PlaceholderDimensions {
	return PlaceholderDimensions{Alignment: ui.PlaceholderAlignmentBottom}
}

// String describes the dimensions.
//
// Flutter: painting/text_painter.dart PlaceholderDimensions.toString
func (d PlaceholderDimensions) String() string {
	if d.Alignment == ui.PlaceholderAlignmentBaseline {
		return "PlaceholderDimensions(" + d.Size.String() + ", " + d.Alignment.String() + "(" +
			foundation.DebugFormatDouble(d.BaselineOffset) + " from top))"
	}
	return "PlaceholderDimensions(" + d.Size.String() + ", " + d.Alignment.String() + ")"
}

// TextWidthBasis is the way to measure the width of the rendered text. The zero
// value is TextWidthBasisUnspecified: parent.
//
// Flutter: painting/text_painter.dart TextWidthBasis
type TextWidthBasis int

// The values of TextWidthBasis follow Flutter's declaration order.
const (
	// TextWidthBasisUnspecified is the zero value: parent.
	TextWidthBasisUnspecified TextWidthBasis = iota
	// TextWidthBasisParent makes the width take all the space the parent gives.
	//
	// Flutter: painting/text_painter.dart TextWidthBasis.parent
	TextWidthBasisParent
	// TextWidthBasisLongestLine makes the width the width of the longest line.
	//
	// Flutter: painting/text_painter.dart TextWidthBasis.longestLine
	TextWidthBasisLongestLine
)

// Name returns the Dart enum name, such as "longestLine".
func (b TextWidthBasis) Name() string {
	if b == TextWidthBasisLongestLine {
		return "longestLine"
	}
	if b == TextWidthBasisParent {
		return "parent"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "TextWidthBasis.parent".
func (b TextWidthBasis) String() string { return "TextWidthBasis." + b.Name() }

// WordBoundary is a TextBoundary that finds the boundaries of the words in the
// text of a laid-out TextPainter.
//
// Flutter: painting/text_painter.dart WordBoundary
type WordBoundary struct {
	services.TextBoundaryDefaults
	text      InlineSpan
	paragraph *ui.Paragraph
	// MoveByWordBoundary is the boundary that skips the spaces and punctuation
	// between words, as the text editing shortcuts do.
	//
	// Flutter: painting/text_painter.dart WordBoundary.moveByWordBoundary
	MoveByWordBoundary services.TextBoundary
}

func newWordBoundary(text InlineSpan, paragraph *ui.Paragraph) *WordBoundary {
	w := &WordBoundary{text: text, paragraph: paragraph}
	w.Self = w
	u := &untilTextBoundary{boundary: w, predicate: w.skipSpacesAndPunctuations}
	u.Self = u
	w.MoveByWordBoundary = u
	return w
}

// GetTextBoundaryAt is the range of the word at the position.
//
// Flutter: painting/text_painter.dart WordBoundary.getTextBoundaryAt
func (w *WordBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return w.paragraph.GetWordBoundary(ui.NewTextPosition(max(position, 0)))
}

// runeAt returns the rune whose encoding contains the byte at the index, or
// false if the text has none there (Dart's codePointAt).
func (w *WordBoundary) runeAt(index int) (rune, bool) {
	b, ok := InlineSpanCodeUnitAt(w.text, index)
	if !ok {
		return 0, false
	}
	start := index
	for b&0xC0 == 0x80 && start > 0 && index-start < utf8.UTFMax-1 {
		start--
		b, ok = InlineSpanCodeUnitAt(w.text, start)
		if !ok {
			return 0, false
		}
	}
	var buf [utf8.UTFMax]byte
	n := 0
	for ; n < utf8.UTFMax; n++ {
		c, ok := InlineSpanCodeUnitAt(w.text, start+n)
		if !ok {
			break
		}
		buf[n] = c
	}
	r, _ := utf8.DecodeRune(buf[:n])
	return r, true
}

func isNewlineRune(r rune) bool {
	switch r {
	case 0x000A, 0x0085, 0x000B, 0x000C, 0x2028, 0x2029:
		return true
	}
	return false
}

func (w *WordBoundary) skipSpacesAndPunctuations(offset int, forward bool) bool {
	innerIndex, outerIndex := offset, offset-1
	if forward {
		innerIndex, outerIndex = offset-1, offset
	}
	inner, innerOK := w.runeAt(innerIndex)
	outer, outerOK := w.runeAt(outerIndex)
	hardBreakRulesApply := !innerOK || !outerOK || isNewlineRune(inner) || isNewlineRune(outer)
	return hardBreakRulesApply || !unicode.In(inner, unicode.Zs, unicode.P)
}

type untilTextBoundary struct {
	services.TextBoundaryDefaults
	boundary  services.TextBoundary
	predicate services.UntilPredicate
}

func (u *untilTextBoundary) GetLeadingTextBoundaryAt(position int) (int, bool) {
	if position < 0 {
		return 0, false
	}
	offset, ok := u.boundary.GetLeadingTextBoundaryAt(position)
	if !ok || u.predicate(offset, false) {
		return offset, ok
	}
	return u.GetLeadingTextBoundaryAt(offset - 1)
}

func (u *untilTextBoundary) GetTrailingTextBoundaryAt(position int) (int, bool) {
	offset, ok := u.boundary.GetTrailingTextBoundaryAt(max(position, 0))
	if !ok || u.predicate(offset, true) {
		return offset, ok
	}
	return u.GetTrailingTextBoundaryAt(offset)
}

// GetTextBoundaryAt combines the two boundaries.
func (u *untilTextBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return u.TextBoundaryDefaults.GetTextBoundaryAt(position)
}

// textLayout is Flutter's _TextLayout: a laid-out paragraph and the direction
// it was laid out in.
type textLayout struct {
	paragraph        *ui.Paragraph
	writingDirection ui.TextDirection
	painter          *TextPainter

	endOfTextCaret *lineCaretMetrics
}

func (l *textLayout) distanceToBaseline(baseline ui.TextBaseline) float64 {
	if baseline == ui.TextBaselineIdeographic {
		return l.paragraph.IdeographicBaseline()
	}
	return l.paragraph.AlphabeticBaseline()
}

func (l *textLayout) endOfTextCaretMetrics() lineCaretMetrics {
	if l.endOfTextCaret == nil {
		m := l.computeEndOfTextCaretAnchorOffset()
		l.endOfTextCaret = &m
	}
	return *l.endOfTextCaret
}

func (l *textLayout) computeEndOfTextCaretAnchorOffset() lineCaretMetrics {
	raw := l.painter.PlainText()
	lastLineIndex := l.paragraph.NumberOfLines() - 1
	lineMetrics, _ := l.paragraph.GetLineMetricsAt(lastLineIndex)
	last, _ := utf8.DecodeLastRuneInString(raw)
	var hasTrailingSpaces bool
	switch last {
	case 0x0009:
		hasTrailingSpaces = true
	case 0x00A0, 0x2007, 0x202F:
		hasTrailingSpaces = false
	default:
		hasTrailingSpaces = unicode.Is(unicode.Zs, last)
	}
	baseline := lineMetrics.Baseline
	var dx, height float64
	lastGlyph, hasGlyph := l.paragraph.GetGlyphInfoAt(len(raw) - 1)
	if hasTrailingSpaces && hasGlyph {
		bounds := lastGlyph.GraphemeClusterLayoutBounds
		if l.writingDirection == ui.TextDirectionRTL {
			dx = bounds.Left
		} else {
			dx = bounds.Right
		}
		height = bounds.Height()
	} else {
		if l.writingDirection == ui.TextDirectionRTL {
			dx = lineMetrics.Left
		} else {
			dx = lineMetrics.Left + lineMetrics.Width
		}
		height = lineMetrics.Height
	}
	return lineCaretMetrics{offset: ui.Offset{Dx: dx, Dy: baseline}, writingDirection: l.writingDirection, height: height}
}

func (l *textLayout) contentWidthFor(minWidth, maxWidth float64, basis TextWidthBasis) float64 {
	if basis == TextWidthBasisLongestLine {
		return ui.ClampDouble(l.paragraph.LongestLine(), minWidth, maxWidth)
	}
	return ui.ClampDouble(l.paragraph.MaxIntrinsicWidth(), minWidth, maxWidth)
}

// layoutCache is Flutter's _TextPainterLayoutCacheWithOffset.
type layoutCache struct {
	layout         *textLayout
	layoutMaxWidth float64
	contentWidth   float64
	textAlignment  float64

	inlinePlaceholderBoxes []ui.TextBox
	hasPlaceholderBoxes    bool
	lineMetrics            []ui.LineMetrics
	hasLineMetrics         bool
	previousCaretKey       int
	hasPreviousCaretKey    bool
}

func newLayoutCache(layout *textLayout, textAlignment, layoutMaxWidth, contentWidth float64) *layoutCache {
	if foundation.DebugMode {
		if !(textAlignment >= 0 && textAlignment <= 1) || math.IsNaN(layoutMaxWidth) || math.IsNaN(contentWidth) {
			panic(errors.New("TextPainterLayoutCacheInvalid", "the layout cache needs a text alignment in 0..1 and numeric widths",
				"textAlignment", textAlignment, "layoutMaxWidth", layoutMaxWidth, "contentWidth", contentWidth))
		}
	}
	return &layoutCache{layout: layout, textAlignment: textAlignment, layoutMaxWidth: layoutMaxWidth, contentWidth: contentWidth}
}

func (c *layoutCache) paragraph() *ui.Paragraph { return c.layout.paragraph }

func (c *layoutCache) paintOffset() ui.Offset {
	if c.textAlignment == 0 {
		return ui.Offset{}
	}
	if math.IsInf(c.paragraph().Width(), 0) {
		return ui.Offset{Dx: math.Inf(1)}
	}
	return ui.Offset{Dx: c.textAlignment * (c.contentWidth - c.paragraph().Width())}
}

// resizeToFit reports whether the cached layout fits the new constraints.
func (c *layoutCache) resizeToFit(minWidth, maxWidth float64, basis TextWidthBasis) bool {
	if maxWidth == c.contentWidth && minWidth == c.contentWidth {
		c.contentWidth = c.layout.contentWidthFor(minWidth, maxWidth, basis)
		return true
	}
	if offset := c.paintOffset(); math.IsInf(offset.Dx, 0) && math.IsInf(c.paragraph().Width(), 0) && !math.IsInf(minWidth, 0) {
		return false
	}
	maxIntrinsic := c.paragraph().MaxIntrinsicWidth()
	skipLineBreaking := maxWidth == c.layoutMaxWidth ||
		((c.paragraph().Width()-maxIntrinsic) > -foundation.PrecisionErrorTolerance &&
			(maxWidth-maxIntrinsic) > -foundation.PrecisionErrorTolerance)
	if skipLineBreaking {
		c.contentWidth = c.layout.contentWidthFor(minWidth, maxWidth, basis)
		return true
	}
	return false
}

func (c *layoutCache) getInlinePlaceholderBoxes() []ui.TextBox {
	if !c.hasPlaceholderBoxes {
		c.inlinePlaceholderBoxes, c.hasPlaceholderBoxes = c.paragraph().GetBoxesForPlaceholders(), true
	}
	return c.inlinePlaceholderBoxes
}

func (c *layoutCache) getLineMetrics() []ui.LineMetrics {
	if !c.hasLineMetrics {
		c.lineMetrics, c.hasLineMetrics = c.paragraph().ComputeLineMetrics(), true
	}
	return c.lineMetrics
}

// lineCaretMetrics is Flutter's _LineCaretMetrics.
type lineCaretMetrics struct {
	offset           ui.Offset
	writingDirection ui.TextDirection
	height           float64
}

func (m lineCaretMetrics) shift(offset ui.Offset) lineCaretMetrics {
	if offset == (ui.Offset{}) {
		return m
	}
	return lineCaretMetrics{offset: offset.Add(m.offset), writingDirection: m.writingDirection, height: m.height}
}

// TextPainterOpts are the optional parameters of NewTextPainter.
//
// Flutter: painting/text_painter.dart TextPainter.new
type TextPainterOpts struct {
	// TextEngine lays out the paragraphs. Deviation: Flutter reaches the engine
	// through a global; the framework passes the engine of the
	// PlatformDispatcher here.
	TextEngine ui.ParagraphFactory
	// Text is the text to paint.
	Text InlineSpan
	// TextAlign is how the text aligns horizontally; the default is start.
	TextAlign ui.TextAlign
	// TextDirection is the default directionality of the text; it must be set
	// before layout.
	TextDirection ui.TextDirection
	// TextScaler scales the font sizes; the default is TextScalerNoScaling.
	TextScaler TextScaler
	// MaxLines is the maximum number of lines; it must be greater than zero.
	MaxLines foundation.Opt[int]
	// Ellipsis replaces the text that doesn't fit; it must not be empty if set.
	Ellipsis string
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// StrutStyle is the strut of the paragraph.
	StrutStyle *StrutStyle
	// TextWidthBasis is how the width is measured; the default is parent.
	TextWidthBasis TextWidthBasis
	// TextHeightBehavior says how the line height applies to the first and last
	// lines.
	TextHeightBehavior *ui.TextHeightBehavior
	// TabSize is the distance between tab stops, as a count of the advance of a
	// space in the font of the text's style; a tab advances to the next multiple,
	// measured from the start of its line. 0, the default, gives tabs no stops.
	//
	// Flutter: no counterpart
	TabSize float64
}

// TextPainter is an object that paints a TextSpan into a Canvas.
//
// Flutter: painting/text_painter.dart TextPainter
type TextPainter struct {
	engine ui.ParagraphFactory

	text               InlineSpan
	textAlign          ui.TextAlign
	textDirection      ui.TextDirection
	textScaler         TextScaler
	maxLines           foundation.Opt[int]
	ellipsis           string
	locale             foundation.Opt[ui.Locale]
	strutStyle         *StrutStyle
	textWidthBasis     TextWidthBasis
	textHeightBehavior *ui.TextHeightBehavior
	tabSize            float64

	placeholderDimensions []PlaceholderDimensions
	debugNeedsRelayout    bool
	layoutCache           *layoutCache
	rebuildForPaint       bool
	layoutTemplate        *ui.Paragraph
	cachedPlainText       *string
	caretMetrics          lineCaretMetrics
	disposed              bool

	// DebugPaintTextLayoutBoxes makes Paint draw a box around each character, for
	// debugging.
	//
	// Flutter: painting/text_painter.dart TextPainter.debugPaintTextLayoutBoxes
	DebugPaintTextLayoutBoxes bool
}

// NewTextPainter creates a text painter. In debug builds a MaxLines that is not
// greater than zero, and a text that isn't valid, panic with the tags
// TextPainterMaxLinesInvalid and TextPainterEllipsisEmpty (never for ""), or the
// ones of the span.
//
// Flutter: painting/text_painter.dart TextPainter.new
func NewTextPainter(opts ...TextPainterOpts) *TextPainter {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Text != nil {
			o.Text.DebugAssertIsValid()
		}
		if o.MaxLines.IsSome() && o.MaxLines.Value() <= 0 {
			panic(errors.New("TextPainterMaxLinesInvalid", "maxLines must be greater than zero", "maxLines", o.MaxLines.Value()))
		}
	}
	if o.TextAlign == ui.TextAlignUnspecified {
		o.TextAlign = ui.TextAlignStart
	}
	if o.TextScaler == nil {
		o.TextScaler = TextScalerNoScaling
	}
	if o.TextWidthBasis == TextWidthBasisUnspecified {
		o.TextWidthBasis = TextWidthBasisParent
	}
	return &TextPainter{
		engine: o.TextEngine, text: o.Text, textAlign: o.TextAlign, textDirection: o.TextDirection, textScaler: o.TextScaler,
		maxLines: o.MaxLines, ellipsis: o.Ellipsis, locale: o.Locale, strutStyle: o.StrutStyle,
		textWidthBasis: o.TextWidthBasis, textHeightBehavior: o.TextHeightBehavior, tabSize: o.TabSize,
		debugNeedsRelayout: true, rebuildForPaint: true,
	}
}

// TextPainterComputeWidthOpts are the optional parameters of
// TextPainterComputeWidth and TextPainterComputeMaxIntrinsicWidth.
//
// Flutter: painting/text_painter.dart TextPainter.computeWidth
type TextPainterComputeWidthOpts struct {
	// TextEngine lays out the paragraph.
	TextEngine ui.ParagraphFactory
	// TextAlign is how the text aligns horizontally.
	TextAlign ui.TextAlign
	// TextScaler scales the font sizes.
	TextScaler TextScaler
	// MaxLines is the maximum number of lines.
	MaxLines foundation.Opt[int]
	// Ellipsis replaces the text that doesn't fit.
	Ellipsis string
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// StrutStyle is the strut of the paragraph.
	StrutStyle *StrutStyle
	// TextWidthBasis is how the width is measured.
	TextWidthBasis TextWidthBasis
	// TextHeightBehavior says how the line height applies.
	TextHeightBehavior *ui.TextHeightBehavior
	// TabSize is the distance between tab stops in spaces; 0 gives tabs no stops.
	//
	// Flutter: no counterpart
	TabSize float64
	// MinWidth is the minimum width of the layout.
	MinWidth float64
	// MaxWidth is the maximum width of the layout; the default is infinity.
	MaxWidth foundation.Opt[float64]
}

func computeWith(text InlineSpan, textDirection ui.TextDirection, o TextPainterComputeWidthOpts, get func(*TextPainter) float64) float64 {
	p := NewTextPainter(TextPainterOpts{
		TextEngine: o.TextEngine, Text: text, TextAlign: o.TextAlign, TextDirection: textDirection, TextScaler: o.TextScaler,
		MaxLines: o.MaxLines, Ellipsis: o.Ellipsis, Locale: o.Locale, StrutStyle: o.StrutStyle,
		TextWidthBasis: o.TextWidthBasis, TextHeightBehavior: o.TextHeightBehavior, TabSize: o.TabSize,
	})
	p.Layout(o.MinWidth, o.MaxWidth.OrElse(math.Inf(1)))
	defer p.Dispose()
	return get(p)
}

// TextPainterComputeWidth returns the width the text takes when laid out in the
// constraints.
//
// Flutter: painting/text_painter.dart TextPainter.computeWidth
func TextPainterComputeWidth(text InlineSpan, textDirection ui.TextDirection, opts ...TextPainterComputeWidthOpts) float64 {
	return computeWith(text, textDirection, firstOpts(opts), func(p *TextPainter) float64 { return p.Width() })
}

// TextPainterComputeMaxIntrinsicWidth returns the max intrinsic width of the text
// laid out in the constraints.
//
// Flutter: painting/text_painter.dart TextPainter.computeMaxIntrinsicWidth
func TextPainterComputeMaxIntrinsicWidth(text InlineSpan, textDirection ui.TextDirection, opts ...TextPainterComputeWidthOpts) float64 {
	return computeWith(text, textDirection, firstOpts(opts), func(p *TextPainter) float64 { return p.MaxIntrinsicWidth() })
}

func (p *TextPainter) assertTextLayoutIsValid() {
	if p.disposed {
		panic(errors.New("TextPainterDisposed", "the TextPainter was disposed"))
	}
	if p.layoutCache == nil {
		panic(errors.New("TextPainterLayoutNotAvailable", "Text layout not available: the TextPainter has never been laid out, or its layout was invalidated"))
	}
}

func (p *TextPainter) assertNoRelayoutNeeded() {
	if foundation.DebugMode && p.debugNeedsRelayout {
		panic(errors.New("TextPainterNeedsRelayout", "the TextPainter needs a layout after the textWidthBasis changed"))
	}
}

// MarkNeedsLayout flags the previous layout result as invalid.
//
// Flutter: painting/text_painter.dart TextPainter.markNeedsLayout
func (p *TextPainter) MarkNeedsLayout() {
	p.layoutCache = nil
}

// Text is the (potentially styled) text to paint.
//
// Flutter: painting/text_painter.dart TextPainter.text
func (p *TextPainter) Text() InlineSpan { return p.text }

// SetText sets the text, marking the layout as invalid if the text changed in a
// way that needs a layout, or flagging the paragraph for rebuilding if only the
// painting changed.
//
// Flutter: painting/text_painter.dart TextPainter.text
func (p *TextPainter) SetText(value InlineSpan) {
	if foundation.DebugMode && value != nil {
		value.DebugAssertIsValid()
	}
	if p.text == value || (p.text != nil && value != nil && p.text.Equal(value)) {
		return
	}
	var oldStyle, newStyle *TextStyle
	if p.text != nil {
		oldStyle = p.text.Style()
	}
	if value != nil {
		newStyle = value.Style()
	}
	if !oldStyle.Equal(newStyle) {
		p.layoutTemplate = nil
	}
	comparison := RenderComparisonLayout
	if value != nil && p.text != nil {
		comparison = p.text.CompareTo(value)
	}
	p.text = value
	p.cachedPlainText = nil
	if comparison >= RenderComparisonLayout {
		p.MarkNeedsLayout()
	} else if comparison >= RenderComparisonPaint {
		p.rebuildForPaint = true
	}
}

// PlainText is the text of the span, without the semantics labels.
//
// Flutter: painting/text_painter.dart TextPainter.plainText
func (p *TextPainter) PlainText() string {
	if p.cachedPlainText == nil {
		s := ""
		if p.text != nil {
			s = InlineSpanToPlainText(p.text, ToPlainTextOpts{ExcludeSemanticsLabels: true})
		}
		p.cachedPlainText = &s
	}
	return *p.cachedPlainText
}

// TextAlign is how the text is aligned horizontally.
//
// Flutter: painting/text_painter.dart TextPainter.textAlign
func (p *TextPainter) TextAlign() ui.TextAlign { return p.textAlign }

// SetTextAlign sets the alignment, invalidating the layout when it changes.
//
// Flutter: painting/text_painter.dart TextPainter.textAlign
func (p *TextPainter) SetTextAlign(value ui.TextAlign) {
	if value == ui.TextAlignUnspecified {
		value = ui.TextAlignStart
	}
	if p.textAlign == value {
		return
	}
	p.textAlign = value
	p.MarkNeedsLayout()
}

// TextDirection is the default directionality of the text; unspecified is null.
//
// Flutter: painting/text_painter.dart TextPainter.textDirection
func (p *TextPainter) TextDirection() ui.TextDirection { return p.textDirection }

// SetTextDirection sets the directionality, invalidating the layout when it
// changes.
//
// Flutter: painting/text_painter.dart TextPainter.textDirection
func (p *TextPainter) SetTextDirection(value ui.TextDirection) {
	if p.textDirection == value {
		return
	}
	p.textDirection = value
	p.MarkNeedsLayout()
	p.layoutTemplate = nil
}

// TextScaler is the font scaling policy.
//
// Flutter: painting/text_painter.dart TextPainter.textScaler
func (p *TextPainter) TextScaler() TextScaler { return p.textScaler }

// SetTextScaler sets the policy, invalidating the layout when it changes.
//
// Flutter: painting/text_painter.dart TextPainter.textScaler
func (p *TextPainter) SetTextScaler(value TextScaler) {
	if value == nil {
		value = TextScalerNoScaling
	}
	if TextScalerEqual(value, p.textScaler) {
		return
	}
	p.textScaler = value
	p.MarkNeedsLayout()
	p.layoutTemplate = nil
}

// Ellipsis is the string that replaces the text that doesn't fit; "" is null.
//
// Flutter: painting/text_painter.dart TextPainter.ellipsis
func (p *TextPainter) Ellipsis() string { return p.ellipsis }

// SetEllipsis sets the ellipsis, invalidating the layout when it changes.
//
// Flutter: painting/text_painter.dart TextPainter.ellipsis
func (p *TextPainter) SetEllipsis(value string) {
	if p.ellipsis == value {
		return
	}
	p.ellipsis = value
	p.MarkNeedsLayout()
}

// Locale is the locale used to select region-specific glyphs.
//
// Flutter: painting/text_painter.dart TextPainter.locale
func (p *TextPainter) Locale() foundation.Opt[ui.Locale] { return p.locale }

// SetLocale sets the locale, invalidating the layout when it changes.
//
// Flutter: painting/text_painter.dart TextPainter.locale
func (p *TextPainter) SetLocale(value foundation.Opt[ui.Locale]) {
	if p.locale == value {
		return
	}
	p.locale = value
	p.MarkNeedsLayout()
}

// MaxLines is the maximum number of lines.
//
// Flutter: painting/text_painter.dart TextPainter.maxLines
func (p *TextPainter) MaxLines() foundation.Opt[int] { return p.maxLines }

// SetMaxLines sets the maximum number of lines, invalidating the layout when it
// changes. In debug builds a value that is not greater than zero panics with the
// tag TextPainterMaxLinesInvalid.
//
// Flutter: painting/text_painter.dart TextPainter.maxLines
func (p *TextPainter) SetMaxLines(value foundation.Opt[int]) {
	if foundation.DebugMode && value.IsSome() && value.Value() <= 0 {
		panic(errors.New("TextPainterMaxLinesInvalid", "maxLines must be greater than zero", "maxLines", value.Value()))
	}
	if p.maxLines == value {
		return
	}
	p.maxLines = value
	p.MarkNeedsLayout()
}

// StrutStyle is the strut of the paragraph.
//
// Flutter: painting/text_painter.dart TextPainter.strutStyle
func (p *TextPainter) StrutStyle() *StrutStyle { return p.strutStyle }

// SetStrutStyle sets the strut, invalidating the layout when it changes.
//
// Flutter: painting/text_painter.dart TextPainter.strutStyle
func (p *TextPainter) SetStrutStyle(value *StrutStyle) {
	if p.strutStyle.Equal(value) {
		return
	}
	p.strutStyle = value
	p.MarkNeedsLayout()
}

// TextWidthBasis is how the width of the text is measured.
//
// Flutter: painting/text_painter.dart TextPainter.textWidthBasis
func (p *TextPainter) TextWidthBasis() TextWidthBasis { return p.textWidthBasis }

// SetTextWidthBasis sets the basis. The layout is not invalidated, but Layout
// must be called again before the width is read.
//
// Flutter: painting/text_painter.dart TextPainter.textWidthBasis
func (p *TextPainter) SetTextWidthBasis(value TextWidthBasis) {
	if value == TextWidthBasisUnspecified {
		value = TextWidthBasisParent
	}
	if p.textWidthBasis == value {
		return
	}
	if foundation.DebugMode {
		p.debugNeedsRelayout = true
	}
	p.textWidthBasis = value
}

// TextHeightBehavior says how the line height applies to the first and last
// lines.
//
// Flutter: painting/text_painter.dart TextPainter.textHeightBehavior
func (p *TextPainter) TextHeightBehavior() *ui.TextHeightBehavior { return p.textHeightBehavior }

// SetTextHeightBehavior sets the behavior, invalidating the layout when it
// changes.
//
// Flutter: painting/text_painter.dart TextPainter.textHeightBehavior
func (p *TextPainter) SetTextHeightBehavior(value *ui.TextHeightBehavior) {
	if (p.textHeightBehavior == nil) == (value == nil) && (value == nil || *p.textHeightBehavior == *value) {
		return
	}
	p.textHeightBehavior = value
	p.MarkNeedsLayout()
}

// TabSize is the distance between tab stops, as a count of the advance of a
// space; 0 means tabs have no stops.
//
// Flutter: no counterpart
func (p *TextPainter) TabSize() float64 { return p.tabSize }

// SetTabSize sets the distance between tab stops, invalidating the layout when
// it changes.
//
// Flutter: no counterpart
func (p *TextPainter) SetTabSize(value float64) {
	if p.tabSize == value {
		return
	}
	p.tabSize = value
	p.MarkNeedsLayout()
}

// InlinePlaceholderBoxes returns the boxes of the placeholders, shifted by the
// paint offset; nil before layout.
//
// Flutter: painting/text_painter.dart TextPainter.inlinePlaceholderBoxes
func (p *TextPainter) InlinePlaceholderBoxes() []ui.TextBox {
	cache := p.layoutCache
	if cache == nil {
		return nil
	}
	offset := cache.paintOffset()
	if math.IsInf(offset.Dx, 0) || math.IsInf(offset.Dy, 0) {
		return []ui.TextBox{}
	}
	raw := cache.getInlinePlaceholderBoxes()
	if offset == (ui.Offset{}) {
		return raw
	}
	out := make([]ui.TextBox, len(raw))
	for i, b := range raw {
		out[i] = shiftTextBox(b, offset)
	}
	return out
}

// SetPlaceholderDimensions sets the sizes of the placeholders, in the order they
// appear in the text. Empty or equal values are ignored. In debug builds a
// count that doesn't match the placeholders panics with the tag
// TextPainterPlaceholderCountMismatch.
//
// Flutter: painting/text_painter.dart TextPainter.setPlaceholderDimensions
func (p *TextPainter) SetPlaceholderDimensions(value []PlaceholderDimensions) {
	if len(value) == 0 || slices.Equal(value, p.placeholderDimensions) {
		return
	}
	if foundation.DebugMode {
		count := 0
		p.text.VisitChildren(func(span InlineSpan) bool {
			if _, ok := span.(PlaceholderSpan); ok {
				count++
			}
			return len(value) >= count
		})
		if count != len(value) {
			panic(errors.New("TextPainterPlaceholderCountMismatch", "the number of placeholder dimensions doesn't match the placeholders of the text",
				"dimensions", len(value), "placeholders", count))
		}
	}
	p.placeholderDimensions = slices.Clone(value)
	p.MarkNeedsLayout()
}

func (p *TextPainter) createParagraphStyle(alignOverride ui.TextAlign) *ui.ParagraphStyle {
	if foundation.DebugMode && p.textDirection == ui.TextDirectionUnspecified {
		panic(errors.New("TextPainterDirectionNotSet", "TextPainter.textDirection must be set to a non-null value before using the TextPainter"))
	}
	var base *TextStyle
	if p.text != nil {
		base = p.text.Style()
	}
	if base == nil {
		base = NewTextStyle()
	}
	align := p.textAlign
	if alignOverride != ui.TextAlignUnspecified {
		align = alignOverride
	}
	return base.GetParagraphStyle(GetParagraphStyleOpts{
		TextAlign: align, TextDirection: p.textDirection, TextScaler: p.textScaler, MaxLines: p.maxLines,
		TextHeightBehavior: p.textHeightBehavior, Ellipsis: p.ellipsis, Locale: p.locale, StrutStyle: p.strutStyle,
		TabSize: p.tabSize,
	})
}

func (p *TextPainter) createLayoutTemplate() *ui.Paragraph {
	builder := ui.NewParagraphBuilder(p.engine, p.createParagraphStyle(ui.TextAlignLeft))
	if p.text != nil {
		if style := p.text.Style(); style != nil {
			builder.PushStyle(style.GetTextStyle(GetTextStyleOpts{TextScaler: p.textScaler}))
		}
	}
	builder.AddText(" ")
	para := builder.Build()
	para.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	return para
}

func (p *TextPainter) getOrCreateLayoutTemplate() *ui.Paragraph {
	if p.layoutTemplate == nil {
		p.layoutTemplate = p.createLayoutTemplate()
	}
	return p.layoutTemplate
}

// PreferredLineHeight is the height of a line of the text's style, not counting
// the text itself.
//
// Flutter: painting/text_painter.dart TextPainter.preferredLineHeight
func (p *TextPainter) PreferredLineHeight() float64 { return p.getOrCreateLayoutTemplate().Height() }

// MinIntrinsicWidth is the width at which decreasing the width doesn't
// decrease the height.
//
// Flutter: painting/text_painter.dart TextPainter.minIntrinsicWidth
func (p *TextPainter) MinIntrinsicWidth() float64 {
	p.assertTextLayoutIsValid()
	return p.layoutCache.layout.paragraph.MinIntrinsicWidth()
}

// MaxIntrinsicWidth is the width at which increasing the width doesn't change
// the height.
//
// Flutter: painting/text_painter.dart TextPainter.maxIntrinsicWidth
func (p *TextPainter) MaxIntrinsicWidth() float64 {
	p.assertTextLayoutIsValid()
	return p.layoutCache.layout.paragraph.MaxIntrinsicWidth()
}

// Width is the horizontal space required to paint the text.
//
// Flutter: painting/text_painter.dart TextPainter.width
func (p *TextPainter) Width() float64 {
	p.assertTextLayoutIsValid()
	p.assertNoRelayoutNeeded()
	return p.layoutCache.contentWidth
}

// Height is the vertical space required to paint the text.
//
// Flutter: painting/text_painter.dart TextPainter.height
func (p *TextPainter) Height() float64 {
	p.assertTextLayoutIsValid()
	return p.layoutCache.layout.paragraph.Height()
}

// Size is the width and height required to paint the text.
//
// Flutter: painting/text_painter.dart TextPainter.size
func (p *TextPainter) Size() ui.Size {
	p.assertTextLayoutIsValid()
	p.assertNoRelayoutNeeded()
	return ui.Size{Width: p.Width(), Height: p.Height()}
}

// ComputeDistanceToActualBaseline returns the distance from the top to the
// baseline of the first line.
//
// Flutter: painting/text_painter.dart TextPainter.computeDistanceToActualBaseline
func (p *TextPainter) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) float64 {
	p.assertTextLayoutIsValid()
	return p.layoutCache.layout.distanceToBaseline(baseline)
}

// DidExceedMaxLines is whether the text didn't fit in MaxLines.
//
// Flutter: painting/text_painter.dart TextPainter.didExceedMaxLines
func (p *TextPainter) DidExceedMaxLines() bool {
	p.assertTextLayoutIsValid()
	return p.layoutCache.paragraph().DidExceedMaxLines()
}

func (p *TextPainter) createParagraph(text InlineSpan) *ui.Paragraph {
	builder := ui.NewParagraphBuilder(p.engine, p.createParagraphStyle(ui.TextAlignUnspecified))
	text.Build(builder, SpanBuildOpts{TextScaler: p.textScaler, Dimensions: p.placeholderDimensions})
	p.rebuildForPaint = false
	return builder.Build()
}

// Layout computes the visual position of the glyphs for painting the text. In
// both cases the constraints are of the width. Without a text or a text
// direction it panics with the tags TextPainterTextNotSet and
// TextPainterDirectionNotSet.
//
// Flutter: painting/text_painter.dart TextPainter.layout
func (p *TextPainter) Layout(minWidth, maxWidth float64) {
	if foundation.DebugMode {
		if math.IsNaN(maxWidth) || math.IsNaN(minWidth) {
			panic(errors.New("TextPainterConstraintsNaN", "the width constraints must be numbers", "minWidth", minWidth, "maxWidth", maxWidth))
		}
		p.debugNeedsRelayout = false
	}
	cached := p.layoutCache
	if cached != nil && cached.resizeToFit(minWidth, maxWidth, p.textWidthBasis) {
		return
	}
	text := p.text
	if text == nil {
		panic(errors.New("TextPainterTextNotSet", "TextPainter.text must be set to a non-null value before using the TextPainter"))
	}
	textDirection := p.textDirection
	if textDirection == ui.TextDirectionUnspecified {
		panic(errors.New("TextPainterDirectionNotSet", "TextPainter.textDirection must be set to a non-null value before using the TextPainter"))
	}
	paintOffsetAlignment := computePaintOffsetFraction(p.textAlign, textDirection)
	adjustMaxWidth := math.IsInf(maxWidth, 0) && paintOffsetAlignment != 0
	adjustedMaxWidth, haveAdjusted := maxWidth, true
	if adjustMaxWidth {
		haveAdjusted = false
		if cached != nil {
			adjustedMaxWidth, haveAdjusted = cached.layout.paragraph.MaxIntrinsicWidth(), true
		}
	}
	layoutMaxWidth := maxWidth
	if haveAdjusted {
		layoutMaxWidth = adjustedMaxWidth
	}
	var paragraph *ui.Paragraph
	if cached != nil {
		paragraph = cached.paragraph()
	} else {
		paragraph = p.createParagraph(text)
	}
	paragraph.Layout(ui.NewParagraphConstraints(layoutMaxWidth))
	layout := &textLayout{paragraph: paragraph, writingDirection: textDirection, painter: p}
	contentWidth := layout.contentWidthFor(minWidth, maxWidth, p.textWidthBasis)
	if !haveAdjusted && !math.IsInf(minWidth, 0) {
		newInputWidth := paragraph.MaxIntrinsicWidth()
		paragraph.Layout(ui.NewParagraphConstraints(newInputWidth))
		p.layoutCache = newLayoutCache(layout, paintOffsetAlignment, newInputWidth, contentWidth)
	} else {
		p.layoutCache = newLayoutCache(layout, paintOffsetAlignment, layoutMaxWidth, contentWidth)
	}
}

// Paint draws the text in the canvas at the offset. Without a layout it panics
// with the tag TextPainterPaintBeforeLayout.
//
// Flutter: painting/text_painter.dart TextPainter.paint
func (p *TextPainter) Paint(canvas *ui.Canvas, offset ui.Offset) {
	cache := p.layoutCache
	if cache == nil {
		panic(errors.New("TextPainterPaintBeforeLayout", "TextPainter.paint called when text geometry was not yet calculated. "+
			"Please call layout() before paint() to position the text before painting it."))
	}
	po := cache.paintOffset()
	if math.IsInf(po.Dx, 0) || math.IsInf(po.Dy, 0) {
		return
	}
	if p.rebuildForPaint {
		para := p.createParagraph(p.text)
		para.Layout(ui.NewParagraphConstraints(cache.layoutMaxWidth))
		cache.layout.paragraph = para
	}
	if p.DebugPaintTextLayoutBoxes && foundation.DebugMode {
		p.debugPaintCharacterLayoutBoxes(canvas, offset)
	}
	canvas.DrawParagraph(cache.paragraph(), offset.Add(po))
}

func (p *TextPainter) debugPaintCharacterLayoutBoxes(canvas *ui.Canvas, offset ui.Offset) {
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(1.0)
	paint.SetColor(ui.NewColor(0xFF00FFFF))
	for _, box := range p.GetBoxesForSelection(ui.NewTextRange(0, len(p.PlainText()))) {
		canvas.DrawRect(box.ToRect().Shift(offset), paint)
	}
}

// TextPainterIsHighSurrogate reports whether the UTF-16 code unit is a high
// surrogate, the first half of a pair.
//
// Flutter: painting/text_painter.dart TextPainter.isHighSurrogate
func TextPainterIsHighSurrogate(value int) bool { return value&0xFC00 == 0xD800 }

// TextPainterIsLowSurrogate reports whether the UTF-16 code unit is a low
// surrogate, the second half of a pair.
//
// Flutter: painting/text_painter.dart TextPainter.isLowSurrogate
func TextPainterIsLowSurrogate(value int) bool { return value&0xFC00 == 0xDC00 }

// GetOffsetAfter returns the offset after the character at the offset, or false
// at the end of the text. Offsets are UTF-8 byte offsets (SPEC §12.10), so the
// step is the length of the encoded character.
//
// Flutter: painting/text_painter.dart TextPainter.getOffsetAfter
func (p *TextPainter) GetOffsetAfter(offset int) (int, bool) {
	b, ok := InlineSpanCodeUnitAt(p.text, offset)
	if !ok {
		return 0, false
	}
	n := 1
	switch {
	case b >= 0xF0:
		n = 4
	case b >= 0xE0:
		n = 3
	case b >= 0xC0:
		n = 2
	}
	return offset + n, true
}

// GetOffsetBefore returns the offset before the character that ends at the
// offset, or false at the start of the text.
//
// Flutter: painting/text_painter.dart TextPainter.getOffsetBefore
func (p *TextPainter) GetOffsetBefore(offset int) (int, bool) {
	b, ok := InlineSpanCodeUnitAt(p.text, offset-1)
	if !ok {
		return 0, false
	}
	start := offset - 1
	for b&0xC0 == 0x80 && start > 0 && offset-1-start < utf8.UTFMax-1 {
		start--
		b, ok = InlineSpanCodeUnitAt(p.text, start)
		if !ok {
			break
		}
	}
	return start, true
}

func computePaintOffsetFraction(textAlign ui.TextAlign, textDirection ui.TextDirection) float64 {
	rtl := textDirection == ui.TextDirectionRTL
	switch textAlign {
	case ui.TextAlignLeft:
		return 0
	case ui.TextAlignRight:
		return 1
	case ui.TextAlignCenter:
		return 0.5
	case ui.TextAlignEnd:
		if rtl {
			return 0
		}
		return 1
	}
	// start and justify
	if rtl {
		return 1
	}
	return 0
}

// GetOffsetForCaret returns the offset at which to paint the caret, relative to
// the top left of the painted text.
//
// Flutter: painting/text_painter.dart TextPainter.getOffsetForCaret
func (p *TextPainter) GetOffsetForCaret(position ui.TextPosition, caretPrototype ui.Rect) ui.Offset {
	cache := p.layoutCache
	metrics, ok := p.computeCaretMetrics(position)
	if !ok {
		alignment := computePaintOffsetFraction(p.textAlign, p.textDirection)
		dx := 0.0
		if alignment != 0 {
			dx = alignment * cache.contentWidth
		}
		return ui.Offset{Dx: dx}
	}
	raw := metrics.offset
	if metrics.writingDirection == ui.TextDirectionRTL {
		raw = ui.Offset{Dx: raw.Dx - caretPrototype.Width(), Dy: raw.Dy}
	}
	po := cache.paintOffset()
	adjustedDx := ui.ClampDouble(raw.Dx+po.Dx, 0, cache.contentWidth)
	return ui.Offset{Dx: adjustedDx, Dy: raw.Dy + po.Dy}
}

func (p *TextPainter) strutDisabled() bool {
	if p.strutStyle == nil || p.strutStyle.Equal(StrutStyleDisabled()) {
		return true
	}
	return p.strutStyle.FontSize.IsSome() && p.strutStyle.FontSize.Value() == 0.0
}

// GetFullHeightForCaret returns the full height of the caret at the position.
//
// Flutter: painting/text_painter.dart TextPainter.getFullHeightForCaret
func (p *TextPainter) GetFullHeightForCaret(position ui.TextPosition, caretPrototype ui.Rect) float64 {
	if p.strutDisabled() {
		if m, ok := p.computeCaretMetrics(position); ok {
			return m.height
		}
	}
	boxes := p.getOrCreateLayoutTemplate().GetBoxesForRange(0, 1, ui.BoxesForRangeOpts{HeightStyle: ui.BoxHeightStyleStrut})
	if len(boxes) == 0 {
		return p.PreferredLineHeight()
	}
	return boxes[0].ToRect().Height()
}

func (p *TextPainter) isNewlineAtOffset(offset int) bool {
	plain := p.PlainText()
	if offset < 0 || offset >= len(plain) {
		return false
	}
	start := offset
	for start > 0 && plain[start]&0xC0 == 0x80 {
		start--
	}
	r, _ := utf8.DecodeRuneInString(plain[start:])
	return isNewlineRune(r)
}

func (p *TextPainter) computeCaretMetrics(position ui.TextPosition) (lineCaretMetrics, bool) {
	p.assertTextLayoutIsValid()
	p.assertNoRelayoutNeeded()
	cached := p.layoutCache
	if cached.paragraph().NumberOfLines() < 1 {
		return lineCaretMetrics{}, false
	}
	var offset int
	var anchorToLeadingEdge bool
	switch {
	case position.Offset == 0:
		offset, anchorToLeadingEdge = 0, true
	case position.Affinity != ui.TextAffinityUpstream:
		offset, anchorToLeadingEdge = position.Offset, true
	case p.isNewlineAtOffset(position.Offset - 1):
		offset, anchorToLeadingEdge = position.Offset, true
	default:
		offset, anchorToLeadingEdge = position.Offset-1, false
	}
	key := offset
	if !anchorToLeadingEdge {
		key = -offset - 1
	}
	if cached.hasPreviousCaretKey && key == cached.previousCaretKey {
		return p.caretMetrics, true
	}
	glyph, ok := cached.paragraph().GetGlyphInfoAt(offset)
	if !ok {
		template := p.getOrCreateLayoutTemplate()
		lm, _ := template.GetLineMetricsAt(0)
		return cached.layout.endOfTextCaretMetrics().shift(ui.Offset{Dy: -lm.Baseline}), true
	}
	graphemeRange := glyph.GraphemeClusterRange
	if graphemeRange.IsCollapsed() {
		return p.computeCaretMetrics(ui.NewTextPosition(offset + 1))
	}
	if anchorToLeadingEdge && graphemeRange.Start != offset {
		return p.computeCaretMetrics(ui.NewTextPosition(graphemeRange.End))
	}
	boxes := cached.paragraph().GetBoxesForRange(graphemeRange.Start, graphemeRange.End, ui.BoxesForRangeOpts{HeightStyle: ui.BoxHeightStyleStrut})
	anchorToLeft := anchorToLeadingEdge
	if glyph.WritingDirection == ui.TextDirectionRTL {
		anchorToLeft = !anchorToLeadingEdge
	}
	box := boxes[len(boxes)-1]
	if anchorToLeft {
		box = boxes[0]
	}
	x := box.Right
	if anchorToLeft {
		x = box.Left
	}
	metrics := lineCaretMetrics{offset: ui.Offset{Dx: x, Dy: box.Top}, writingDirection: box.Direction, height: box.Bottom - box.Top}
	cached.previousCaretKey, cached.hasPreviousCaretKey = key, true
	p.caretMetrics = metrics
	return metrics, true
}

// GetBoxesForSelection returns the boxes that cover the range of the text
// (UTF-8 byte offsets), shifted by the paint offset. Deviation: Flutter takes a
// TextSelection; here the range is a ui.TextRange, which is what the
// selection's start and end are.
//
// Flutter: painting/text_painter.dart TextPainter.getBoxesForSelection
func (p *TextPainter) GetBoxesForSelection(selection ui.TextRange, opts ...ui.BoxesForRangeOpts) []ui.TextBox {
	p.assertTextLayoutIsValid()
	if foundation.DebugMode && !selection.IsValid() {
		panic(errors.New("TextPainterSelectionInvalid", "the selection must be valid", "selection", selection.String()))
	}
	p.assertNoRelayoutNeeded()
	cached := p.layoutCache
	offset := cached.paintOffset()
	if math.IsInf(offset.Dx, 0) || math.IsInf(offset.Dy, 0) {
		return []ui.TextBox{}
	}
	start, end := selection.Start, selection.End
	if start > end {
		start, end = end, start
	}
	boxes := cached.paragraph().GetBoxesForRange(start, end, opts...)
	if offset == (ui.Offset{}) {
		return boxes
	}
	out := make([]ui.TextBox, len(boxes))
	for i, b := range boxes {
		out[i] = shiftTextBox(b, offset)
	}
	return out
}

// GetClosestGlyphForOffset returns the glyph closest to the offset, or false.
//
// Flutter: painting/text_painter.dart TextPainter.getClosestGlyphForOffset
func (p *TextPainter) GetClosestGlyphForOffset(offset ui.Offset) (ui.GlyphInfo, bool) {
	p.assertTextLayoutIsValid()
	p.assertNoRelayoutNeeded()
	cached := p.layoutCache
	po := cached.paintOffset()
	raw, ok := cached.paragraph().GetClosestGlyphInfoForOffset(offset.Sub(po))
	if !ok || po == (ui.Offset{}) {
		return raw, ok
	}
	raw.GraphemeClusterLayoutBounds = raw.GraphemeClusterLayoutBounds.Shift(po)
	return raw, true
}

// GetPositionForOffset returns the text position closest to the offset in the
// painted text.
//
// Flutter: painting/text_painter.dart TextPainter.getPositionForOffset
func (p *TextPainter) GetPositionForOffset(offset ui.Offset) ui.TextPosition {
	p.assertTextLayoutIsValid()
	p.assertNoRelayoutNeeded()
	cached := p.layoutCache
	return cached.paragraph().GetPositionForOffset(offset.Sub(cached.paintOffset()))
}

// GetWordBoundary returns the text range of the word at the position.
//
// Flutter: painting/text_painter.dart TextPainter.getWordBoundary
func (p *TextPainter) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	p.assertTextLayoutIsValid()
	return p.layoutCache.paragraph().GetWordBoundary(position)
}

// WordBoundaries returns the boundaries of the words of the laid-out text.
//
// Flutter: painting/text_painter.dart TextPainter.wordBoundaries
func (p *TextPainter) WordBoundaries() *WordBoundary {
	return newWordBoundary(p.text, p.layoutCache.paragraph())
}

// GetLineBoundary returns the text range of the line at the position.
//
// Flutter: painting/text_painter.dart TextPainter.getLineBoundary
func (p *TextPainter) GetLineBoundary(position ui.TextPosition) ui.TextRange {
	p.assertTextLayoutIsValid()
	return p.layoutCache.paragraph().GetLineBoundary(position)
}

func shiftLineMetrics(m ui.LineMetrics, offset ui.Offset) ui.LineMetrics {
	m.Left += offset.Dx
	m.Baseline += offset.Dy
	return m
}

func shiftTextBox(b ui.TextBox, offset ui.Offset) ui.TextBox {
	return ui.NewTextBoxFromLTRBD(b.Left+offset.Dx, b.Top+offset.Dy, b.Right+offset.Dx, b.Bottom+offset.Dy, b.Direction)
}

// ComputeLineMetrics returns the metrics of the lines, shifted by the paint
// offset.
//
// Flutter: painting/text_painter.dart TextPainter.computeLineMetrics
func (p *TextPainter) ComputeLineMetrics() []ui.LineMetrics {
	p.assertTextLayoutIsValid()
	p.assertNoRelayoutNeeded()
	cached := p.layoutCache
	offset := cached.paintOffset()
	if math.IsInf(offset.Dx, 0) || math.IsInf(offset.Dy, 0) {
		return nil
	}
	raw := cached.getLineMetrics()
	if offset == (ui.Offset{}) {
		return raw
	}
	out := make([]ui.LineMetrics, len(raw))
	for i, m := range raw {
		out[i] = shiftLineMetrics(m, offset)
	}
	return out
}

// DebugDisposed reports whether Dispose was called. It is only meaningful in
// debug builds, where it is always right; release builds return false.
//
// Flutter: painting/text_painter.dart TextPainter.debugDisposed
func (p *TextPainter) DebugDisposed() bool { return p.disposed }

// Dispose releases the resources of the painter. The painter can't be used
// afterwards. In debug builds disposing twice panics with the tag
// TextPainterDisposed.
//
// Flutter: painting/text_painter.dart TextPainter.dispose
func (p *TextPainter) Dispose() {
	if foundation.DebugMode && p.disposed {
		panic(errors.New("TextPainterDisposed", "the TextPainter was disposed"))
	}
	p.disposed = true
	p.layoutTemplate = nil
	p.layoutCache = nil
	p.text = nil
}
