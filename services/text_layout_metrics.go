package services

import "github.com/rm4n0s/glutter/ui"

// TextLayoutMetrics is the interface a text-editing renderer gives to the
// functions that edit text and need to know how the text is laid out (a word
// boundary, the line a position is on, the position above or below another).
//
// Flutter: services/text_layout_metrics.dart TextLayoutMetrics
type TextLayoutMetrics interface {
	// GetLineAtOffset returns the visual line that the position is on, as a
	// selection from the start to the end of the line.
	//
	// Flutter: services/text_layout_metrics.dart TextLayoutMetrics.getLineAtOffset
	GetLineAtOffset(position ui.TextPosition) TextSelection
	// GetWordBoundary returns the word that the position is in.
	//
	// Flutter: services/text_layout_metrics.dart TextLayoutMetrics.getWordBoundary
	GetWordBoundary(position ui.TextPosition) ui.TextRange
	// GetTextPositionAbove returns the position on the line above, at the same
	// horizontal offset.
	//
	// Flutter: services/text_layout_metrics.dart TextLayoutMetrics.getTextPositionAbove
	GetTextPositionAbove(position ui.TextPosition) ui.TextPosition
	// GetTextPositionBelow returns the position on the line below, at the same
	// horizontal offset.
	//
	// Flutter: services/text_layout_metrics.dart TextLayoutMetrics.getTextPositionBelow
	GetTextPositionBelow(position ui.TextPosition) ui.TextPosition
}

// TextLayoutMetricsIsWhitespace reports whether the Unicode code point is a
// whitespace character.
//
// Deviation: Flutter takes a UTF-16 code unit; this takes a code point
// (SPEC §12.10).
//
// Flutter: services/text_layout_metrics.dart TextLayoutMetrics.isWhitespace
func TextLayoutMetricsIsWhitespace(codePoint rune) bool {
	switch codePoint {
	case 0x9, // horizontal tab
		0xA,    // line feed
		0xB,    // vertical tab
		0xC,    // form feed
		0xD,    // carriage return
		0x1C,   // file separator
		0x1D,   // group separator
		0x1E,   // record separator
		0x1F,   // unit separator
		0x20,   // space
		0xA0,   // no-break space
		0x1680, // ogham space mark
		0x2000, // en quad
		0x2001, // em quad
		0x2002, // en space
		0x2003, // em space
		0x2004, // three-per-em space
		0x2005, // four-er-em space
		0x2006, // six-per-em space
		0x2007, // figure space
		0x2008, // punctuation space
		0x2009, // thin space
		0x200A, // hair space
		0x202F, // narrow no-break space
		0x205F, // medium mathematical space
		0x3000: // ideographic space
		return true
	}
	return false
}

// TextLayoutMetricsIsLineTerminator reports whether the Unicode code point
// ends a line.
//
// Flutter: services/text_layout_metrics.dart TextLayoutMetrics.isLineTerminator
func TextLayoutMetricsIsLineTerminator(codePoint rune) bool {
	switch codePoint {
	case 0x0A, // line feed
		0x0B,   // vertical feed
		0x0C,   // form feed
		0x0D,   // carriage return
		0x85,   // new line
		0x2028, // line separator
		0x2029: // paragraph separator
		return true
	}
	return false
}
