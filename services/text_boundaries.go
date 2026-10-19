package services

import (
	"sort"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// CharacterBoundary is a TextBoundary that finds the boundaries of the
// characters (extended grapheme clusters) of a text. Offsets are UTF-8 byte
// offsets (SPEC §12.10).
//
// Flutter: services/text_boundary.dart CharacterBoundary
type CharacterBoundary struct {
	text       string
	boundaries []int
}

// NewCharacterBoundary creates the boundary of the characters of the text.
//
// Flutter: services/text_boundary.dart CharacterBoundary.new
func NewCharacterBoundary(text string) *CharacterBoundary {
	return &CharacterBoundary{text: text, boundaries: foundation.GraphemeBoundaries(text)}
}

// GetLeadingTextBoundaryAt is the start of the character at the position (the
// position itself when it is a boundary), or false for a negative position.
//
// Flutter: services/text_boundary.dart CharacterBoundary.getLeadingTextBoundaryAt
func (b *CharacterBoundary) GetLeadingTextBoundaryAt(position int) (int, bool) {
	if position < 0 {
		return 0, false
	}
	p := min(position, len(b.text))
	i := sort.SearchInts(b.boundaries, p+1) - 1
	return b.boundaries[i], true
}

// GetTrailingTextBoundaryAt is the end of the character at the position, or
// false when the position is at or after the end of the text.
//
// Flutter: services/text_boundary.dart CharacterBoundary.getTrailingTextBoundaryAt
func (b *CharacterBoundary) GetTrailingTextBoundaryAt(position int) (int, bool) {
	if position >= len(b.text) {
		return 0, false
	}
	i := sort.SearchInts(b.boundaries, max(0, position+1))
	return b.boundaries[i], true
}

// GetTextBoundaryAt is the range of the character at the position.
//
// Flutter: services/text_boundary.dart CharacterBoundary.getTextBoundaryAt
func (b *CharacterBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	if position < 0 {
		end, ok := b.GetTrailingTextBoundaryAt(position)
		if !ok {
			end = -1
		}
		return ui.NewTextRange(-1, end)
	}
	if position >= len(b.text) {
		start, ok := b.GetLeadingTextBoundaryAt(position)
		if !ok {
			start = -1
		}
		return ui.NewTextRange(start, -1)
	}
	start, _ := b.GetLeadingTextBoundaryAt(position)
	end, ok := b.GetTrailingTextBoundaryAt(position)
	if !ok {
		end = -1
	}
	return ui.NewTextRange(start, end)
}

// LineBoundary is a TextBoundary that finds the boundaries of the lines of a
// laid-out text.
//
// Flutter: services/text_boundary.dart LineBoundary
type LineBoundary struct {
	TextBoundaryDefaults
	textLayout TextLayoutMetrics
}

// NewLineBoundary creates the boundary of the lines of a text layout.
//
// Flutter: services/text_boundary.dart LineBoundary.new
func NewLineBoundary(textLayout TextLayoutMetrics) *LineBoundary {
	b := &LineBoundary{textLayout: textLayout}
	b.Self = b
	return b
}

// GetTextBoundaryAt is the visual line at the position.
//
// Flutter: services/text_boundary.dart LineBoundary.getTextBoundaryAt
func (b *LineBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.textLayout.GetLineAtOffset(ui.NewTextPosition(max(position, 0))).Range()
}

// ParagraphBoundary is a TextBoundary that finds the boundaries of the
// paragraphs of a text, which end with a line terminator.
//
// Flutter: services/text_boundary.dart ParagraphBoundary
type ParagraphBoundary struct {
	TextBoundaryDefaults
	text string
}

// NewParagraphBoundary creates the boundary of the paragraphs of the text.
//
// Flutter: services/text_boundary.dart ParagraphBoundary.new
func NewParagraphBoundary(text string) *ParagraphBoundary {
	b := &ParagraphBoundary{text: text}
	b.Self = b
	return b
}

// terminatorAt is the size in bytes of the line terminator that starts at the
// index, or 0 if there is none.
func terminatorAt(text string, index int) int {
	r, size := utf8.DecodeRuneInString(text[index:])
	if r != utf8.RuneError && TextLayoutMetricsIsLineTerminator(r) {
		return size
	}
	return 0
}

// GetLeadingTextBoundaryAt is the start of the paragraph at the position.
//
// Flutter: services/text_boundary.dart ParagraphBoundary.getLeadingTextBoundaryAt
func (b *ParagraphBoundary) GetLeadingTextBoundaryAt(position int) (int, bool) {
	if position < 0 || b.text == "" {
		return 0, false
	}
	if position >= len(b.text) {
		return len(b.text), true
	}
	if position == 0 {
		return 0, true
	}
	index := position
	if index > 1 && b.text[index] == 0x0A && b.text[index-1] == 0x0D {
		index -= 2
	} else if terminatorAt(b.text, index) > 0 {
		index--
	}
	for index > 0 {
		if size := terminatorAt(b.text, index); size > 0 {
			return index + size, true
		}
		index--
	}
	return max(index, 0), true
}

// GetTrailingTextBoundaryAt is the end of the paragraph at the position, after
// its line terminator.
//
// Flutter: services/text_boundary.dart ParagraphBoundary.getTrailingTextBoundaryAt
func (b *ParagraphBoundary) GetTrailingTextBoundaryAt(position int) (int, bool) {
	if position >= len(b.text) || b.text == "" {
		return 0, false
	}
	if position < 0 {
		return 0, true
	}
	index := position
	size := terminatorAt(b.text, index)
	for size == 0 {
		index++
		if index >= len(b.text) {
			return len(b.text), true
		}
		size = terminatorAt(b.text, index)
	}
	if index < len(b.text)-1 && b.text[index] == 0x0D && b.text[index+1] == 0x0A {
		return index + 2, true
	}
	return index + size, true
}

// GetTextBoundaryAt is the range of the paragraph at the position.
//
// Flutter: services/text_boundary.dart TextBoundary.getTextBoundaryAt
func (b *ParagraphBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.TextBoundaryDefaults.GetTextBoundaryAt(position)
}

// DocumentBoundary is a TextBoundary whose only boundaries are the start and the
// end of the text.
//
// Flutter: services/text_boundary.dart DocumentBoundary
type DocumentBoundary struct {
	TextBoundaryDefaults
	text string
}

// NewDocumentBoundary creates the boundary of the document of the text.
//
// Flutter: services/text_boundary.dart DocumentBoundary.new
func NewDocumentBoundary(text string) *DocumentBoundary {
	b := &DocumentBoundary{text: text}
	b.Self = b
	return b
}

// GetLeadingTextBoundaryAt is the start of the text, or false for a negative
// position.
//
// Flutter: services/text_boundary.dart DocumentBoundary.getLeadingTextBoundaryAt
func (b *DocumentBoundary) GetLeadingTextBoundaryAt(position int) (int, bool) {
	return 0, position >= 0
}

// GetTrailingTextBoundaryAt is the end of the text, or false for a position at
// or after the end.
//
// Flutter: services/text_boundary.dart DocumentBoundary.getTrailingTextBoundaryAt
func (b *DocumentBoundary) GetTrailingTextBoundaryAt(position int) (int, bool) {
	return len(b.text), position < len(b.text)
}

// GetTextBoundaryAt is the range of the document.
//
// Flutter: services/text_boundary.dart TextBoundary.getTextBoundaryAt
func (b *DocumentBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.TextBoundaryDefaults.GetTextBoundaryAt(position)
}
