package uia

import (
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the Text pattern (SPEC §25.11): the offsets of the text of a field
// in UTF-16 units, as UI Automation counts them, and the text range with the units it moves by. The
// semantics tree counts bytes of the UTF-8 text.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin

// UTF16Len is the length of the text in UTF-16 units.
//
// Flutter: no counterpart
func UTF16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16Units(r)
	}
	return n
}

func utf16Units(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

// UTF16Offset is the offset in UTF-16 units of the byte offset in the UTF-8 text. An offset outside the
// text is clamped; one in the middle of a character counts the character.
//
// Flutter: no counterpart
func UTF16Offset(s string, byteOffset int) int {
	n := 0
	for i := 0; i < len(s) && i < byteOffset; {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		n += utf16Units(r)
	}
	return n
}

// ByteOffset is the byte offset in the UTF-8 text of the offset in UTF-16 units; one past the end is
// the end, and an offset in the middle of a surrogate pair is the start of the pair.
//
// Flutter: no counterpart
func ByteOffset(s string, utf16Offset int) int {
	if utf16Offset <= 0 {
		return 0
	}
	units := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if units+utf16Units(r) > utf16Offset {
			return i
		}
		units += utf16Units(r)
		i += size
		if units == utf16Offset {
			return i
		}
	}
	return len(s)
}

// caretOf is the extent and the base of the selection of a text field in UTF-16 units; both are -1
// when the field has no selection.
func caretOf(n *Node) (caret, base int) {
	if n.TextSelectionBase < 0 || n.TextSelectionExtent < 0 {
		return -1, -1
	}
	s := textOf(n)
	return UTF16Offset(s, n.TextSelectionExtent), UTF16Offset(s, n.TextSelectionBase)
}

// TextRange is a range of the text of a field, in UTF-16 units: Start is the first unit in it and End
// the one after the last. A range with Start equal to End is a caret.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin
type TextRange struct{ Start, End int }

// Text is the text of a field as UTF-16 units, with the operations of the Text pattern on ranges.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin
type Text struct{ units []uint16 }

// NewText makes the Text of a string.
//
// Flutter: no counterpart
func NewText(s string) Text { return Text{units: utf16.Encode([]rune(s))} }

// Len is the length in UTF-16 units.
//
// Flutter: no counterpart
func (t Text) Len() int { return len(t.units) }

// Document is the range of all of the text.
//
// Flutter: no counterpart
func (t Text) Document() TextRange { return TextRange{0, len(t.units)} }

// Clamp keeps a range inside the text, with Start not after End.
//
// Flutter: no counterpart
func (t Text) Clamp(r TextRange) TextRange {
	r.Start = min(max(r.Start, 0), len(t.units))
	r.End = min(max(r.End, r.Start), len(t.units))
	return r
}

// Substring is the text of the range, cut to at most maxLength units when maxLength is not negative (a
// cut never splits a surrogate pair).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::GetText
func (t Text) Substring(r TextRange, maxLength int) string {
	r = t.Clamp(r)
	end := r.End
	if maxLength >= 0 && end-r.Start > maxLength {
		end = r.Start + maxLength
		if end > r.Start && end < len(t.units) && utf16.IsSurrogate(rune(t.units[end-1])) && t.units[end-1] < 0xDC00 {
			end--
		}
	}
	return string(utf16.Decode(t.units[r.Start:end]))
}

// boundaries are the offsets where a unit of text starts, in increasing order, ending with the length
// of the text. Every offset from 0 to the length is in exactly one unit.
func (t Text) boundaries(unit int) []int {
	n := len(t.units)
	if n == 0 {
		return []int{0}
	}
	var out []int
	switch unit {
	case TextUnitCharacter:
		for i := 0; i < n; i++ {
			if i > 0 && utf16.IsSurrogate(rune(t.units[i])) && t.units[i] >= 0xDC00 && t.units[i-1] < 0xDC00 && utf16.IsSurrogate(rune(t.units[i-1])) {
				continue // the low half of a pair
			}
			out = append(out, i)
		}
	case TextUnitWord:
		runes := utf16.Decode(t.units)
		pos := 0
		for i := 0; i < len(runes); {
			out = append(out, pos)
			j := i
			for j < len(runes) && !unicode.IsSpace(runes[j]) {
				pos += utf16Units(runes[j])
				j++
			}
			for j < len(runes) && unicode.IsSpace(runes[j]) {
				pos += utf16Units(runes[j])
				j++
			}
			i = j
		}
	case TextUnitLine, TextUnitParagraph:
		out = append(out, 0)
		for i, u := range t.units {
			if u == '\n' && i+1 < n {
				out = append(out, i+1)
			}
		}
	default: // format, page, document: all of the text
		out = append(out, 0)
	}
	return append(out, n)
}

// unitAt is the index of the unit that holds the offset (the last unit for an offset at the end).
func unitAt(b []int, off int) int {
	i := 0
	for k := 0; k+1 < len(b); k++ {
		if b[k] <= off {
			i = k
		}
	}
	return i
}

// Expand is ITextRangeProvider::ExpandToEnclosingUnit: the range grows to the units that hold its ends,
// unless it already is whole units. A caret becomes the unit that holds it.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::ExpandToEnclosingUnit
func (t Text) Expand(r TextRange, unit int) TextRange {
	r = t.Clamp(r)
	if len(t.units) == 0 {
		return TextRange{}
	}
	b := t.boundaries(unit)
	i := unitAt(b, r.Start)
	j := i
	if r.End > r.Start {
		j = unitAt(b, r.End-1)
	}
	return TextRange{b[i], b[j+1]}
}

// MoveEndpoint is ITextRangeProvider::MoveEndpointByUnit: the endpoint moves by count units (forward
// when it is positive) and stops at the ends of the text. It returns the range and the number of units
// that it moved. When the start passes the end the end moves with it, and the range is a caret; the same
// the other way round.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::MoveEndpointByUnit
func (t Text) MoveEndpoint(r TextRange, endpoint, unit, count int) (TextRange, int) {
	r = t.Clamp(r)
	b := t.boundaries(unit)
	pos := r.End
	if endpoint == EndpointStart {
		pos = r.Start
	}
	pos, moved := stepBoundaries(b, pos, count)
	if endpoint == EndpointStart {
		r.Start = pos
		if r.End < r.Start {
			r.End = r.Start
		}
	} else {
		r.End = pos
		if r.Start > r.End {
			r.Start = r.End
		}
	}
	return r, moved
}

// stepBoundaries moves an offset by count boundaries (to the next boundary for 1, to the start of its
// own unit for -1 when it is inside one) and returns the new offset and the number of steps taken.
func stepBoundaries(b []int, pos, count int) (int, int) {
	moved := 0
	for ; count > 0; count-- {
		next := -1
		for _, o := range b {
			if o > pos {
				next = o
				break
			}
		}
		if next < 0 {
			break
		}
		pos = next
		moved++
	}
	for ; count < 0; count++ {
		prev := -1
		for i := len(b) - 1; i >= 0; i-- {
			if b[i] < pos {
				prev = b[i]
				break
			}
		}
		if prev < 0 {
			break
		}
		pos = prev
		moved--
	}
	return pos, moved
}

// Move is ITextRangeProvider::Move: the range becomes the unit that holds its start, and then moves by
// count units, stopping at the ends of the text. It returns the new range and the number of units moved.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::Move
func (t Text) Move(r TextRange, unit, count int) (TextRange, int) {
	r = t.Clamp(r)
	if len(t.units) == 0 || count == 0 {
		return r, 0
	}
	b := t.boundaries(unit)
	i := unitAt(b, r.Start)
	j := min(max(i+count, 0), len(b)-2)
	return TextRange{b[j], b[j+1]}, j - i
}

// Find is ITextRangeProvider::FindText: the first range in the given one that holds the text (the last
// one when backward), or false.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::FindText
func (t Text) Find(r TextRange, needle string, backward, ignoreCase bool) (TextRange, bool) {
	r = t.Clamp(r)
	want := utf16.Encode([]rune(needle))
	if len(want) == 0 {
		return TextRange{}, false
	}
	same := func(a, b uint16) bool {
		if ignoreCase {
			return unicode.ToLower(rune(a)) == unicode.ToLower(rune(b))
		}
		return a == b
	}
	match := func(at int) bool {
		if at+len(want) > r.End {
			return false
		}
		for i, w := range want {
			if !same(t.units[at+i], w) {
				return false
			}
		}
		return true
	}
	if backward {
		for at := r.End - len(want); at >= r.Start; at-- {
			if match(at) {
				return TextRange{at, at + len(want)}, true
			}
		}
		return TextRange{}, false
	}
	for at := r.Start; at+len(want) <= r.End; at++ {
		if match(at) {
			return TextRange{at, at + len(want)}, true
		}
	}
	return TextRange{}, false
}

// CompareEndpoints is ITextRangeProvider::CompareEndpoints: -1, 0 or 1 as the endpoint of a is before,
// at or after the endpoint of b.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::CompareEndpoints
func CompareEndpoints(a TextRange, ea int, b TextRange, eb int) int {
	pa, pb := a.Start, b.Start
	if ea == EndpointEnd {
		pa = a.End
	}
	if eb == EndpointEnd {
		pb = b.End
	}
	switch {
	case pa < pb:
		return -1
	case pa > pb:
		return 1
	}
	return 0
}

// MoveEndpointToEndpoint is ITextRangeProvider::MoveEndpointByRange: the endpoint of r takes the
// position of the endpoint of o. When that puts the start after the end, the other end follows.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::MoveEndpointByRange
func MoveEndpointToEndpoint(r TextRange, endpoint int, o TextRange, other int) TextRange {
	pos := o.Start
	if other == EndpointEnd {
		pos = o.End
	}
	if endpoint == EndpointStart {
		r.Start = pos
		r.End = max(r.End, pos)
	} else {
		r.End = pos
		r.Start = min(r.Start, pos)
	}
	return r
}

// Selection is the range of the field's selection or caret in UTF-16 units, or false when the field
// does not say.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textprovider_win.cc AXPlatformNodeTextProviderWin::GetSelection
func Selection(n *Node) (TextRange, bool) {
	caret, base := caretOf(n)
	if caret < 0 {
		return TextRange{}, false
	}
	return TextRange{min(caret, base), max(caret, base)}, true
}

// SelectRange is ITextRangeProvider::Select: the framework sets the selection of the field to the range
// (the set selection action, in bytes of the UTF-8 text).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin::Select
func (m *Model) SelectRange(ref NodeRef, r TextRange) (events []ui.SemanticsActionEvent, st Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if !m.Patterns(n).Has(PatternText) || n.Actions&ui.SemanticsActionSetSelection == 0 {
		return nil, StatusNotSupported
	}
	text := textOf(n)
	r = NewText(text).Clamp(r)
	return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionSetSelection,
		Arguments: map[string]int{"base": ByteOffset(text, r.Start), "extent": ByteOffset(text, r.End)}}}, StatusOK
}
