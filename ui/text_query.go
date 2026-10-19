package ui

import (
	"fmt"

	"github.com/rm4n0s/errors"
)

// TextAffinity is a direction from which to interpret a text position that sits at a soft line break, where the end of one line and the start of the next are the same offset.
//
// Flutter: ui/text.dart TextAffinity
type TextAffinity int

// The values of TextAffinity follow Flutter's declaration order.
const (
	// TextAffinityUnspecified is the zero value: downstream.
	//
	// Flutter: no counterpart
	TextAffinityUnspecified TextAffinity = iota
	// TextAffinityUpstream means the position is associated with the character before the offset: the end of the line before the break.
	//
	// Flutter: ui/text.dart TextAffinity.upstream
	TextAffinityUpstream
	// TextAffinityDownstream means the position is associated with the character after the offset: the start of the line after the break.
	//
	// Flutter: ui/text.dart TextAffinity.downstream
	TextAffinityDownstream
)

// String returns the name Dart prints for the value, such as "TextAffinity.upstream".
func (e TextAffinity) String() string {
	return dartEnumName([]string{"unspecified", "upstream", "downstream"}, int(e), "TextAffinity")
}

// IsUpstream reports whether the affinity is upstream; the unspecified affinity is downstream.
//
// Flutter: no counterpart
func (e TextAffinity) IsUpstream() bool { return e == TextAffinityUpstream }

// TextPosition is a position in a string of text: an offset and, at a soft line break, which side of it.
//
// The offset is a UTF-8 byte offset into the string (SPEC §12.10).
//
// Flutter: ui/text.dart TextPosition
type TextPosition struct {
	// Offset is the byte offset in the text.
	Offset int
	// Affinity is the side of the offset at a soft line break; the zero value is downstream.
	Affinity TextAffinity
}

// TextPositionOpts are the optional parameters of NewTextPosition.
//
// Flutter: ui/text.dart TextPosition
type TextPositionOpts struct {
	// Affinity defaults to downstream.
	Affinity TextAffinity
}

// NewTextPosition creates a position.
//
// Flutter: ui/text.dart TextPosition
func NewTextPosition(offset int, opts ...TextPositionOpts) TextPosition {
	o := oneOpts(opts)
	if o.Affinity == TextAffinityUnspecified {
		o.Affinity = TextAffinityDownstream
	}
	return TextPosition{offset, o.Affinity}
}

// Equal reports whether two positions are the same, with the unspecified affinity being downstream.
//
// Flutter: ui/text.dart TextPosition.==
func (p TextPosition) Equal(other TextPosition) bool {
	return p.Offset == other.Offset && p.Affinity.IsUpstream() == other.Affinity.IsUpstream()
}

// String returns what Dart prints, such as "TextPosition(offset: 3, affinity: TextAffinity.downstream)".
func (p TextPosition) String() string {
	a := p.Affinity
	if a == TextAffinityUnspecified {
		a = TextAffinityDownstream
	}
	return fmt.Sprintf("TextPosition(offset: %d, affinity: %v)", p.Offset, a)
}

// TextRange is a range of characters in a string of text: byte offsets, start inclusive and end exclusive. A range with a negative offset (-1) is invalid.
//
// Flutter: ui/text.dart TextRange
type TextRange struct {
	// Start is the index of the first character in the range.
	Start int
	// End is the next index after the characters in this range.
	End int
}

// TextRangeEmpty is a range that starts and ends at -1, the invalid range.
//
// Flutter: ui/text.dart TextRange.empty
var TextRangeEmpty = TextRange{-1, -1}

// NewTextRange creates a range. An offset below -1 panics with the tag TextRangeInvalid.
//
// Flutter: ui/text.dart TextRange
func NewTextRange(start, end int) TextRange {
	if start < -1 || end < -1 {
		panic(errors.New("TextRangeInvalid", "the offsets of a text range must be at least -1", "start", start, "end", end))
	}
	return TextRange{start, end}
}

// NewTextRangeCollapsed creates an empty range at the offset.
//
// Flutter: ui/text.dart TextRange.collapsed
func NewTextRangeCollapsed(offset int) TextRange { return NewTextRange(offset, offset) }

// IsValid reports whether both offsets are not negative.
//
// Flutter: ui/text.dart TextRange.isValid
func (r TextRange) IsValid() bool { return r.Start >= 0 && r.End >= 0 }

// IsCollapsed reports whether the range is empty.
//
// Flutter: ui/text.dart TextRange.isCollapsed
func (r TextRange) IsCollapsed() bool { return r.Start == r.End }

// IsNormalized reports whether the start is not after the end.
//
// Flutter: ui/text.dart TextRange.isNormalized
func (r TextRange) IsNormalized() bool { return r.End >= r.Start }

// TextBefore is the text before the range. A range that is not normalized or is outside the text panics with the tag TextRangeOutOfBounds.
//
// Flutter: ui/text.dart TextRange.textBefore
func (r TextRange) TextBefore(text string) string { return text[:r.checked(text).Start] }

// TextAfter is the text after the range.
//
// Flutter: ui/text.dart TextRange.textAfter
func (r TextRange) TextAfter(text string) string { return text[r.checked(text).End:] }

// TextInside is the text in the range.
//
// Flutter: ui/text.dart TextRange.textInside
func (r TextRange) TextInside(text string) string {
	r.checked(text)
	return text[r.Start:r.End]
}

func (r TextRange) checked(text string) TextRange {
	if !r.IsNormalized() || r.Start < 0 || r.End > len(text) {
		panic(errors.New("TextRangeOutOfBounds", "the text range is not normalized or is not inside the text",
			"start", r.Start, "end", r.End, "length", len(text)))
	}
	return r
}

// String returns what Dart prints, such as "TextRange(start: 1, end: 3)".
func (r TextRange) String() string {
	return fmt.Sprintf("TextRange(start: %d, end: %d)", r.Start, r.End)
}

// TextBox is a rectangle enclosing a run of text, with the direction of the text in it.
//
// Flutter: ui/text.dart TextBox
type TextBox struct {
	// Left is the left edge of the box.
	Left float64
	// Top is the top edge of the box.
	Top float64
	// Right is the right edge of the box.
	Right float64
	// Bottom is the bottom edge of the box.
	Bottom float64
	// Direction is the direction in which text inside the box flows.
	Direction TextDirection
}

// NewTextBoxFromLTRBD creates a box from its edges and direction.
//
// Flutter: ui/text.dart TextBox.fromLTRBD
func NewTextBoxFromLTRBD(left, top, right, bottom float64, direction TextDirection) TextBox {
	return TextBox{left, top, right, bottom, direction}
}

// ToRect returns the box as a Rect.
//
// Flutter: ui/text.dart TextBox.toRect
func (b TextBox) ToRect() Rect { return Rect{b.Left, b.Top, b.Right, b.Bottom} }

// Start is the left edge of the box for left-to-right text and the right edge for right-to-left text.
//
// Flutter: ui/text.dart TextBox.start
func (b TextBox) Start() float64 {
	if b.Direction == TextDirectionRTL {
		return b.Right
	}
	return b.Left
}

// End is the right edge of the box for left-to-right text and the left edge for right-to-left text.
//
// Flutter: ui/text.dart TextBox.end
func (b TextBox) End() float64 {
	if b.Direction == TextDirectionRTL {
		return b.Left
	}
	return b.Right
}

// String returns what Dart prints for the box.
func (b TextBox) String() string {
	return fmt.Sprintf("TextBox.fromLTRBD(%.1f, %.1f, %.1f, %.1f, %v)", b.Left, b.Top, b.Right, b.Bottom, b.Direction)
}

// GlyphInfo is the layout information of a grapheme cluster in a paragraph.
//
// Flutter: ui/text.dart GlyphInfo
type GlyphInfo struct {
	// GraphemeClusterLayoutBounds is the layout bounding rectangle of the grapheme cluster, in the paragraph's coordinate system.
	GraphemeClusterLayoutBounds Rect
	// GraphemeClusterRange is the range of the grapheme cluster in the text (byte offsets).
	GraphemeClusterRange TextRange
	// WritingDirection is the direction of the text of the grapheme cluster.
	WritingDirection TextDirection
}

// BoxHeightStyle defines how the height of the boxes returned by Paragraph.GetBoxesForRange is bound.
//
// Flutter: ui/text.dart BoxHeightStyle
type BoxHeightStyle int

// The values of BoxHeightStyle follow Flutter's declaration order.
const (
	// BoxHeightStyleUnspecified is the zero value: tight.
	//
	// Flutter: no counterpart
	BoxHeightStyleUnspecified BoxHeightStyle = iota
	// BoxHeightStyleTight provides tight bounding boxes that fit heights per run.
	//
	// Flutter: ui/text.dart BoxHeightStyle.tight
	BoxHeightStyleTight
	// BoxHeightStyleMax makes the height of the boxes the maximum height of all runs in the line.
	//
	// Flutter: ui/text.dart BoxHeightStyle.max
	BoxHeightStyleMax
	// BoxHeightStyleIncludeLineSpacingMiddle extends the top and bottom edge of the bounds to cover half of the line spacing each.
	//
	// Flutter: ui/text.dart BoxHeightStyle.includeLineSpacingMiddle
	BoxHeightStyleIncludeLineSpacingMiddle
	// BoxHeightStyleIncludeLineSpacingTop extends the top edge of the bounds to fully cover any line spacing.
	//
	// Flutter: ui/text.dart BoxHeightStyle.includeLineSpacingTop
	BoxHeightStyleIncludeLineSpacingTop
	// BoxHeightStyleIncludeLineSpacingBottom extends the bottom edge of the bounds to fully cover any line spacing.
	//
	// Flutter: ui/text.dart BoxHeightStyle.includeLineSpacingBottom
	BoxHeightStyleIncludeLineSpacingBottom
	// BoxHeightStyleStrut calculates box heights from the metrics of the paragraph's strut.
	//
	// Flutter: ui/text.dart BoxHeightStyle.strut
	BoxHeightStyleStrut
)

// String returns the name Dart prints for the value, such as "BoxHeightStyle.max".
func (e BoxHeightStyle) String() string {
	return dartEnumName([]string{"unspecified", "tight", "max", "includeLineSpacingMiddle", "includeLineSpacingTop", "includeLineSpacingBottom", "strut"}, int(e), "BoxHeightStyle")
}

// BoxWidthStyle defines how the width of the boxes returned by Paragraph.GetBoxesForRange is bound.
//
// Flutter: ui/text.dart BoxWidthStyle
type BoxWidthStyle int

// The values of BoxWidthStyle follow Flutter's declaration order.
const (
	// BoxWidthStyleUnspecified is the zero value: tight.
	//
	// Flutter: no counterpart
	BoxWidthStyleUnspecified BoxWidthStyle = iota
	// BoxWidthStyleTight provides tight bounding boxes that fit widths to the runs of each line independently.
	//
	// Flutter: ui/text.dart BoxWidthStyle.tight
	BoxWidthStyleTight
	// BoxWidthStyleMax adds boxes at the beginning and end of each line so that the boxes of a line are as wide as the widest line.
	//
	// Flutter: ui/text.dart BoxWidthStyle.max
	BoxWidthStyleMax
)

// String returns the name Dart prints for the value, such as "BoxWidthStyle.max".
func (e BoxWidthStyle) String() string {
	return dartEnumName([]string{"unspecified", "tight", "max"}, int(e), "BoxWidthStyle")
}

// PlaceholderAlignment is where to vertically align a placeholder relative to the surrounding text.
//
// Flutter: ui/text.dart PlaceholderAlignment
type PlaceholderAlignment int

// The values of PlaceholderAlignment follow Flutter's declaration order.
const (
	// PlaceholderAlignmentUnspecified is the zero value: baseline.
	//
	// Flutter: no counterpart
	PlaceholderAlignmentUnspecified PlaceholderAlignment = iota
	// PlaceholderAlignmentBaseline matches the baseline of the placeholder with the baseline of the text.
	//
	// Flutter: ui/text.dart PlaceholderAlignment.baseline
	PlaceholderAlignmentBaseline
	// PlaceholderAlignmentAboveBaseline aligns the bottom edge of the placeholder with the baseline, so it is entirely above the baseline.
	//
	// Flutter: ui/text.dart PlaceholderAlignment.aboveBaseline
	PlaceholderAlignmentAboveBaseline
	// PlaceholderAlignmentBelowBaseline aligns the top edge of the placeholder with the baseline, so it is entirely below the baseline.
	//
	// Flutter: ui/text.dart PlaceholderAlignment.belowBaseline
	PlaceholderAlignmentBelowBaseline
	// PlaceholderAlignmentTop aligns the top edge of the placeholder with the top edge of the text.
	//
	// Flutter: ui/text.dart PlaceholderAlignment.top
	PlaceholderAlignmentTop
	// PlaceholderAlignmentBottom aligns the bottom edge of the placeholder with the bottom edge of the text.
	//
	// Flutter: ui/text.dart PlaceholderAlignment.bottom
	PlaceholderAlignmentBottom
	// PlaceholderAlignmentMiddle aligns the middle of the placeholder with the middle of the text.
	//
	// Flutter: ui/text.dart PlaceholderAlignment.middle
	PlaceholderAlignmentMiddle
)

// String returns the name Dart prints for the value, such as "PlaceholderAlignment.middle".
func (e PlaceholderAlignment) String() string {
	return dartEnumName([]string{"unspecified", "baseline", "aboveBaseline", "belowBaseline", "top", "bottom", "middle"}, int(e), "PlaceholderAlignment")
}
