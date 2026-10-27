package rendering

import (
	"sort"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// The offset map of obscured text.
//
// Deviation: in Flutter the obscured string has one obscuring character for each
// UTF-16 unit of the text, so an offset of the text is an offset of the obscured
// string and the map is the identity. Glutter's offsets are bytes, and the
// obscured string has one obscuring character (of len(ObscuringCharacter)
// bytes) for each grapheme of the text. While the text is obscured, the selection
// and every offset that RenderEditable takes or returns are offsets of the text
// of the field; the paragraph is laid out on the obscured string, so each offset
// goes through toPaint on its way to the paragraph and through fromPaint on its
// way back. An offset inside a grapheme maps to the start of the grapheme
// (SPEC §12.10).

// ObscuredText is the text of the field that the text of the editable stands for
// while ObscureText is true; None when the text of the editable is the text of
// the field, which is the case for every use that does not replace the text with
// the obscuring characters itself.
//
// Flutter: no counterpart
func (r *RenderEditable) ObscuredText() foundation.Opt[string] { return r.obscuredText }

// SetObscuredText changes the text of the field that the obscured text of the
// editable stands for. The grapheme boundaries of the text are worked out here,
// when the text changes, not for every query.
//
// Flutter: no counterpart
func (r *RenderEditable) SetObscuredText(value foundation.Opt[string]) {
	if r.obscuredText == value {
		return
	}
	r.obscuredText = value
	r.obscuredBounds = nil
	if text, ok := value.Get(); ok {
		r.obscuredBounds = foundation.GraphemeBoundaries(text)
	}
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// mapsOffsets is true while the offsets of the text and of the paragraph differ.
func (r *RenderEditable) mapsOffsets() bool {
	return r.obscureText && r.obscuredBounds != nil && len(r.obscuringChar) > 0
}

// textLength is the length, in bytes, of the text that the offsets of the
// editable are into.
func (r *RenderEditable) textLength() int {
	if r.mapsOffsets() {
		return r.obscuredBounds[len(r.obscuredBounds)-1]
	}
	return len(r.PlainText())
}

// toPaint maps an offset of the text to the offset of the start of its grapheme
// in the obscured string. Offsets that are not in the text (-1) stay as they are.
func (r *RenderEditable) toPaint(offset int) int {
	if offset < 0 || !r.mapsOffsets() {
		return offset
	}
	bounds := r.obscuredBounds
	// The last boundary at or before the offset; the offset at the end of the
	// text, or past it, is the end of the obscured string.
	index := sort.SearchInts(bounds, offset+1) - 1
	return index * len(r.obscuringChar)
}

// fromPaint maps an offset of the obscured string to the offset of the start of
// the grapheme that it falls in, in the text.
func (r *RenderEditable) fromPaint(offset int) int {
	if offset < 0 || !r.mapsOffsets() {
		return offset
	}
	index := min(offset/len(r.obscuringChar), len(r.obscuredBounds)-1)
	return r.obscuredBounds[index]
}

func (r *RenderEditable) toPaintPosition(p ui.TextPosition) ui.TextPosition {
	p.Offset = r.toPaint(p.Offset)
	return p
}

func (r *RenderEditable) fromPaintPosition(p ui.TextPosition) ui.TextPosition {
	p.Offset = r.fromPaint(p.Offset)
	return p
}

func (r *RenderEditable) toPaintRange(rng ui.TextRange) ui.TextRange {
	return ui.TextRange{Start: r.toPaint(rng.Start), End: r.toPaint(rng.End)}
}

func (r *RenderEditable) fromPaintRange(rng ui.TextRange) ui.TextRange {
	return ui.TextRange{Start: r.fromPaint(rng.Start), End: r.fromPaint(rng.End)}
}

func (r *RenderEditable) fromPaintSelection(sel services.TextSelection) services.TextSelection {
	if !r.mapsOffsets() {
		return sel
	}
	return sel.CopyWith(services.TextSelectionCopy{
		BaseOffset: foundation.Some(r.fromPaint(sel.BaseOffset)), ExtentOffset: foundation.Some(r.fromPaint(sel.ExtentOffset)),
	})
}

func (r *RenderEditable) toPaintSelection(sel services.TextSelection) services.TextSelection {
	if !r.mapsOffsets() {
		return sel
	}
	return sel.CopyWith(services.TextSelectionCopy{
		BaseOffset: foundation.Some(r.toPaint(sel.BaseOffset)), ExtentOffset: foundation.Some(r.toPaint(sel.ExtentOffset)),
	})
}

// positionForOffset is the position of the text that is closest to the point, in
// the coordinates of the text.
func (r *RenderEditable) positionForOffset(point ui.Offset) ui.TextPosition {
	return r.fromPaintPosition(r.textPainter.GetPositionForOffset(point))
}

// caretOffsetFor is the offset of the caret of the position, in the coordinates
// of the text.
func (r *RenderEditable) caretOffsetFor(position ui.TextPosition, prototype ui.Rect) ui.Offset {
	return r.textPainter.GetOffsetForCaret(r.toPaintPosition(position), prototype)
}
