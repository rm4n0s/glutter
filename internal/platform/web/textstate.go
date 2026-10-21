package web

import "unicode/utf8"

// The editing state of the hidden text field (SPEC §23.5) as plain Go values. The DOM measures a text
// field in UTF-16 code units, as Dart does; the framework's state (ui.TextInputState) in UTF-8 bytes. The
// state of the DOM is kept in UTF-16 units here and converted at the edge, so that the ported logic of
// Flutter's EditingState and CompositionAwareMixin keeps its offsets.

// utf16Len is the length of s in UTF-16 code units.
func utf16Len(s string) int {
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

// utf16Offset converts a byte offset of s to UTF-16 code units. An offset out of s is clamped, and one in
// the middle of a character counts to the end of that character.
func utf16Offset(s string, byteOff int) int {
	byteOff = min(max(byteOff, 0), len(s))
	n := 0
	for i, r := range s {
		if i >= byteOff {
			return n
		}
		n += utf16Units(r)
	}
	return n
}

// byteOffset converts an offset in UTF-16 code units of s to bytes. An offset out of s is clamped, and one
// in the middle of a surrogate pair moves back to the start of the character.
func byteOffset(s string, u16 int) int {
	if u16 <= 0 {
		return 0
	}
	n := 0
	for i, r := range s {
		w := utf16Units(r)
		if n+w > u16 {
			return i
		}
		n += w
	}
	return len(s)
}

// sliceUTF16 is s[from:to] with the offsets in UTF-16 code units.
func sliceUTF16(s string, from, to int) string {
	a := byteOffset(s, from)
	b := byteOffset(s, to)
	if b < a {
		return ""
	}
	return s[a:b]
}

// alignRuneStart moves a byte offset back to the start of the character it is in.
func alignRuneStart(s string, off int) int {
	off = min(max(off, 0), len(s))
	for off > 0 && off < len(s) && !utf8.RuneStart(s[off]) {
		off--
	}
	return off
}

// EditingState is the text and selection of the text field in the DOM.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState
type EditingState struct {
	// Text is the value of the field.
	Text string
	// Base and Extent are the selection in UTF-16 code units: where it began and where it ends (the
	// end that moves). They are never negative.
	Base, Extent int
	// ComposingBase and ComposingExtent are the text the input method is composing, in UTF-16 code
	// units; -1 when there is none.
	ComposingBase, ComposingExtent int
}

// NewEditingState makes a state without a composing range. A negative offset becomes 0, because -1 is not
// a valid selection of a DOM element.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.new
func NewEditingState(text string, base, extent int) EditingState {
	return EditingState{Text: text, Base: max(0, base), Extent: max(0, extent), ComposingBase: -1, ComposingExtent: -1}
}

// EditingStateFromDOM reads the value and selection of a DOM element: selectionStart, selectionEnd and
// selectionDirection. A backward selection has its base at the end.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.fromDomElement
func EditingStateFromDOM(value string, selectionStart, selectionEnd int, direction string) EditingState {
	if direction == "backward" {
		return NewEditingState(value, selectionEnd, selectionStart)
	}
	return NewEditingState(value, selectionStart, selectionEnd)
}

// MinOffset is the smaller end of the selection.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.minOffset
func (e EditingState) MinOffset() int { return min(e.Base, e.Extent) }

// MaxOffset is the greater end of the selection.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.maxOffset
func (e EditingState) MaxOffset() int { return max(e.Base, e.Extent) }

// IsValid says whether the offsets of the selection are not negative.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.isValid
func (e EditingState) IsValid() bool { return e.Base >= 0 && e.Extent >= 0 }

// WithComposing returns the state with the composing range changed.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.copyWith
func (e EditingState) WithComposing(base, extent int) EditingState {
	e.ComposingBase, e.ComposingExtent = base, extent
	return e
}

// Equal compares two states as Flutter does: the text, the ends of the selection whichever way it runs,
// and the composing range.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.==
func (e EditingState) Equal(o EditingState) bool {
	return e.Text == o.Text && e.MinOffset() == o.MinOffset() && e.MaxOffset() == o.MaxOffset() &&
		e.ComposingBase == o.ComposingBase && e.ComposingExtent == o.ComposingExtent
}

// Composing returns the composing range when the state has one that lies in the text and is not empty.
//
// Flutter: no counterpart (the framework's TextEditingValue.isComposingRangeValid)
func (e EditingState) Composing() (start, end int, ok bool) {
	if e.ComposingBase < 0 || e.ComposingExtent <= e.ComposingBase || e.ComposingExtent > utf16Len(e.Text) {
		return 0, 0, false
	}
	return e.ComposingBase, e.ComposingExtent, true
}

// Composition follows the composition events of the text field: the text the input method is composing,
// and where it began.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin
type Composition struct {
	// text is the composing text; active says there is one, as Dart's null does not.
	text   string
	active bool
	// base is the offset in UTF-16 units where the composing text begins, once known.
	base    int
	hasBase bool
}

// Start handles compositionstart: no text is composing yet.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin._handleCompositionStart
func (c *Composition) Start() { *c = Composition{} }

// Update handles compositionupdate: data is the composing text.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin._handleCompositionUpdate
func (c *Composition) Update(data string) { c.text, c.active = data, true }

// End handles compositionend: the composition is over.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin._handleCompositionEnd
func (c *Composition) End() { *c = Composition{} }

// Active says whether there is composing text: the composingText of Flutter is not null.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin.composingText
func (c *Composition) Active() bool { return c.active }

// Text is the composing text.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin.composingText
func (c *Composition) Text() string { return c.text }

// Determine puts the composing range into the state: the composing text ends at the end of the selection
// when it began, and keeps its base while it grows. A state with no composing text, or one whose range
// would begin before the text, is returned as it is.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin.determineCompositionState
func (c *Composition) Determine(state EditingState) EditingState {
	if !c.active {
		return state
	}
	n := utf16Len(c.text)
	if !c.hasBase {
		c.base, c.hasBase = state.Extent-n, true
	}
	if c.base < 0 {
		return state
	}
	return state.WithComposing(c.base, c.base+n)
}
