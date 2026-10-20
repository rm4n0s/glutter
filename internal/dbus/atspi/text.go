package atspi

import (
	"strings"
	"unicode"
	"unicode/utf8"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/ui"
)

const (
	ifaceText         = "org.a11y.atspi.Text"
	ifaceEditableText = "org.a11y.atspi.EditableText"
)

// textOf is the text that the Text interface shows for a node: the value of a text field.
func textOf(n *Node) string { return n.Value }

// hasText says whether the node has the Text interface: the text fields.
func hasText(n *Node) bool { return n.Flags.IsTextField }

// hasEditableText says whether the node has the EditableText interface: the text fields
// that are not read-only. The framework gives a field the action to set its text only
// while it has the focus, so the interface does not wait for the action: the methods
// answer false until the field has it (a screen reader moves the focus there first).
func hasEditableText(n *Node) bool {
	return n.Flags.IsTextField && !n.Flags.IsReadOnly
}

// runeOffset is the number of characters in the text before the byte offset. The
// semantics tree counts bytes of the UTF-8 text, and AT-SPI2 counts characters. An offset
// outside the text is clamped; one in the middle of a character counts the character.
func runeOffset(s string, byteOffset int) int {
	n := 0
	for i := 0; i < len(s) && i < byteOffset; n++ {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return n
}

// byteOffset is the byte offset of the character offset; one past the end is the end.
func byteOffset(s string, runeOff int) int {
	if runeOff <= 0 {
		return 0
	}
	i := 0
	for n := 0; i < len(s); n++ {
		if n == runeOff {
			return i
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return len(s)
}

// caretOf is the caret and the other end of the selection, in characters; both are -1
// when the field has no selection.
func caretOf(n *Node) (caret, base int) {
	if n.TextSelectionBase < 0 || n.TextSelectionExtent < 0 {
		return -1, -1
	}
	s := textOf(n)
	return runeOffset(s, n.TextSelectionExtent), runeOffset(s, n.TextSelectionBase)
}

// segment is a part of a text between two character offsets.
type segment struct{ start, end int }

// Boundaries of the units that GetStringAtOffset and the other methods of Text speak of.
type unit int

const (
	unitChar unit = iota
	unitWord
	unitSentence
	unitLine
	unitParagraph
)

// segments cuts the text into the parts of the unit, in characters: every character is in
// exactly one part. A word is the characters that are not spaces and the spaces that
// follow them (GTK's word, which is what a screen reader reads); a sentence ends after
// ".", "!" or "?" and the spaces that follow; a line and a paragraph end after a new
// line (the text is not laid out here, so a long line is one line).
func segments(text []rune, u unit) []segment {
	n := len(text)
	if n == 0 {
		return nil
	}
	var out []segment
	switch u {
	case unitChar:
		for i := 0; i < n; i++ {
			out = append(out, segment{i, i + 1})
		}
	case unitWord:
		for i := 0; i < n; {
			j := i
			for j < n && !unicode.IsSpace(text[j]) {
				j++
			}
			for j < n && unicode.IsSpace(text[j]) {
				j++
			}
			out = append(out, segment{i, j})
			i = j
		}
	case unitSentence:
		for i := 0; i < n; {
			j := i
			for j < n && !strings.ContainsRune(".!?", text[j]) && text[j] != '\n' {
				j++
			}
			if j < n && text[j] == '\n' {
				j++
			} else {
				for j < n && strings.ContainsRune(".!?", text[j]) {
					j++
				}
				for j < n && unicode.IsSpace(text[j]) && text[j] != '\n' {
					j++
				}
				if j < n && text[j] == '\n' {
					j++
				}
			}
			out = append(out, segment{i, j})
			i = j
		}
	default: // lines and paragraphs
		for i := 0; i < n; {
			j := i
			for j < n && text[j] != '\n' {
				j++
			}
			if j < n {
				j++
			}
			out = append(out, segment{i, j})
			i = j
		}
	}
	return out
}

// segmentAt is the index of the segment that has the character offset; an offset at the
// end of the text is in the last segment.
func segmentAt(segs []segment, offset int) int {
	for i, s := range segs {
		if offset >= s.start && offset < s.end {
			return i
		}
	}
	return len(segs) - 1
}

// pick is the text of the segment that is delta segments away from the one at the
// offset (0 is the one at the offset, -1 the one before, +1 the one after), with its
// start and end. Outside the text it is empty, at the offset.
func pick(text []rune, u unit, offset, delta int) (string, int, int) {
	segs := segments(text, u)
	if len(segs) == 0 || offset < 0 || offset > len(text) {
		return "", 0, 0
	}
	i := segmentAt(segs, offset) + delta
	if i < 0 || i >= len(segs) {
		return "", offset, offset
	}
	s := segs[i]
	return string(text[s.start:s.end]), s.start, s.end
}

// granularityUnit is the unit of an AtspiTextGranularity (char, word, sentence, line,
// paragraph).
func granularityUnit(g uint32) (unit, bool) {
	if g > 4 {
		return 0, false
	}
	return unit(g), true
}

// boundaryUnit is the unit of an AtspiTextBoundaryType (char, word start, word end,
// sentence start, sentence end, line start, line end), the older way to ask. The "end"
// boundaries are answered with the same segments as the "start" ones: that is what GTK
// does for text that has no layout of its own.
func boundaryUnit(b uint32) (unit, bool) {
	switch b {
	case 0:
		return unitChar, true
	case 1, 2:
		return unitWord, true
	case 3, 4:
		return unitSentence, true
	case 5, 6:
		return unitLine, true
	}
	return 0, false
}

// textObject is what is exported under org.a11y.atspi.Text.
type textObject struct{ o *object }

// textNode returns the node and its text as characters.
func (t textObject) textNode() (*Node, []rune, *godbus.Error) {
	n, err := t.o.node()
	if err != nil {
		return nil, nil, err
	}
	return n, []rune(textOf(n)), nil
}

// GetText is Text.GetText: the characters from start up to end; an end of -1 is the end of
// the text.
func (t textObject) GetText(start, end int32) (string, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	_, text, err := t.textNode()
	if err != nil {
		return "", err
	}
	if end < 0 || int(end) > len(text) {
		end = int32(len(text))
	}
	if start < 0 {
		start = 0
	}
	if start >= end {
		return "", nil
	}
	return string(text[start:end]), nil
}

// GetCharacterCount is the method of the same name; the property CharacterCount has the same value.
func (t textObject) GetCharacterCount() (int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	_, text, err := t.textNode()
	return int32(len(text)), err
}

// GetCaretOffset is Text.GetCaretOffset: the caret, or -1 when the field has no selection.
func (t textObject) GetCaretOffset() (int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	n, _, err := t.textNode()
	if err != nil {
		return -1, err
	}
	caret, _ := caretOf(n)
	return int32(caret), nil
}

// SetCaretOffset is Text.SetCaretOffset: it asks the framework to move the caret.
func (t textObject) SetCaretOffset(offset int32) (bool, *godbus.Error) {
	return t.setSelection(offset, offset)
}

// setSelection asks the framework to select from one character to another.
func (t textObject) setSelection(base, extent int32) (bool, *godbus.Error) {
	b := t.o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	n, text, err := t.textNode()
	if err != nil {
		return false, err
	}
	if n.Actions&ui.SemanticsActionSetSelection == 0 || base < 0 || extent < 0 || int(base) > len(text) || int(extent) > len(text) {
		return false, nil
	}
	s := string(text)
	b.perform(ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionSetSelection,
		Arguments: map[string]int{"base": byteOffset(s, int(base)), "extent": byteOffset(s, int(extent))}})
	return true, nil
}

// GetCharacterAtOffset is Text.GetCharacterAtOffset: the character, or -1 outside the text.
func (t textObject) GetCharacterAtOffset(offset int32) (int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	_, text, err := t.textNode()
	if err != nil {
		return -1, err
	}
	if offset < 0 || int(offset) >= len(text) {
		return -1, nil
	}
	return int32(text[offset]), nil
}

// GetStringAtOffset is Text.GetStringAtOffset: the part of the granularity at the offset,
// with where it starts and ends.
func (t textObject) GetStringAtOffset(offset int32, granularity uint32) (string, int32, int32, *godbus.Error) {
	u, ok := granularityUnit(granularity)
	if !ok {
		return "", 0, 0, invalidArgs("no such granularity")
	}
	return t.piece(offset, u, 0)
}

// GetTextAtOffset is Text.GetTextAtOffset (the boundary types).
func (t textObject) GetTextAtOffset(offset int32, boundary uint32) (string, int32, int32, *godbus.Error) {
	u, ok := boundaryUnit(boundary)
	if !ok {
		return "", 0, 0, invalidArgs("no such boundary type")
	}
	return t.piece(offset, u, 0)
}

// GetTextBeforeOffset is Text.GetTextBeforeOffset: the part before the one at the offset.
func (t textObject) GetTextBeforeOffset(offset int32, boundary uint32) (string, int32, int32, *godbus.Error) {
	u, ok := boundaryUnit(boundary)
	if !ok {
		return "", 0, 0, invalidArgs("no such boundary type")
	}
	return t.piece(offset, u, -1)
}

// GetTextAfterOffset is Text.GetTextAfterOffset: the part after the one at the offset.
func (t textObject) GetTextAfterOffset(offset int32, boundary uint32) (string, int32, int32, *godbus.Error) {
	u, ok := boundaryUnit(boundary)
	if !ok {
		return "", 0, 0, invalidArgs("no such boundary type")
	}
	return t.piece(offset, u, 1)
}

func (t textObject) piece(offset int32, u unit, delta int) (string, int32, int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	_, text, err := t.textNode()
	if err != nil {
		return "", 0, 0, err
	}
	s, start, end := pick(text, u, int(offset), delta)
	return s, int32(start), int32(end), nil
}

// GetNSelections is Text.GetNSelections: one when part of the text is selected.
func (t textObject) GetNSelections() (int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	n, _, err := t.textNode()
	if err != nil {
		return 0, err
	}
	if caret, base := caretOf(n); caret >= 0 && caret != base {
		return 1, nil
	}
	return 0, nil
}

// GetSelection is Text.GetSelection: the start and the end of the selection (the
// first and only one).
func (t textObject) GetSelection(num int32) (int32, int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	n, _, err := t.textNode()
	if err != nil {
		return 0, 0, err
	}
	caret, base := caretOf(n)
	if num != 0 || caret < 0 || caret == base {
		return 0, 0, invalidArgs("no such selection")
	}
	return int32(min(caret, base)), int32(max(caret, base)), nil
}

// AddSelection is Text.AddSelection: it selects the range, when nothing is selected.
func (t textObject) AddSelection(start, end int32) (bool, *godbus.Error) {
	if n, err := t.GetNSelections(); err != nil || n != 0 {
		return false, err
	}
	return t.setSelection(start, end)
}

// RemoveSelection is Text.RemoveSelection: it puts the caret where the selection ended.
func (t textObject) RemoveSelection(num int32) (bool, *godbus.Error) {
	if num != 0 {
		return false, nil
	}
	caret, err := t.GetCaretOffset()
	if err != nil || caret < 0 {
		return false, err
	}
	return t.setSelection(caret, caret)
}

// SetSelection is Text.SetSelection: it selects the range (from the start to the end).
func (t textObject) SetSelection(num, start, end int32) (bool, *godbus.Error) {
	if num != 0 {
		return false, nil
	}
	return t.setSelection(start, end)
}

// GetOffsetAtPoint is Text.GetOffsetAtPoint: the text is not laid out for the bridge, so
// there is no answer.
func (t textObject) GetOffsetAtPoint(x, y int32, coordType uint32) (int32, *godbus.Error) {
	return -1, nil
}

// GetCharacterExtents is Text.GetCharacterExtents: the extents of the whole field.
func (t textObject) GetCharacterExtents(offset int32, coordType uint32) (int32, int32, int32, int32, *godbus.Error) {
	return t.fieldExtents(coordType)
}

// GetRangeExtents is Text.GetRangeExtents: the extents of the whole field.
func (t textObject) GetRangeExtents(start, end int32, coordType uint32) (int32, int32, int32, int32, *godbus.Error) {
	return t.fieldExtents(coordType)
}

func (t textObject) fieldExtents(coordType uint32) (int32, int32, int32, int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	n, err := t.o.node()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	e := t.o.extentsOf(n, coordType)
	return e.X, e.Y, e.Width, e.Height, nil
}

// GetBoundedRanges is Text.GetBoundedRanges: none.
func (t textObject) GetBoundedRanges(x, y, w, h int32, coordType, xClip, yClip uint32) ([]boundedRange, *godbus.Error) {
	return []boundedRange{}, nil
}

// boundedRange is the type (iissv) of an entry of GetBoundedRanges.
type boundedRange struct {
	Start, End int32
	Content    string
	Data       godbus.Variant
}

// GetAttributes is Text.GetAttributes: the run of attributes at the offset; the text has
// none, so the run is the whole text.
func (t textObject) GetAttributes(offset int32) (map[string]string, int32, int32, *godbus.Error) {
	t.o.b.mu.Lock()
	defer t.o.b.mu.Unlock()
	_, text, err := t.textNode()
	if err != nil {
		return nil, 0, 0, err
	}
	return map[string]string{}, 0, int32(len(text)), nil
}

// GetAttributeRun is Text.GetAttributeRun.
func (t textObject) GetAttributeRun(offset int32, includeDefaults bool) (map[string]string, int32, int32, *godbus.Error) {
	return t.GetAttributes(offset)
}

// GetDefaultAttributes is Text.GetDefaultAttributes: none.
func (t textObject) GetDefaultAttributes() (map[string]string, *godbus.Error) {
	return map[string]string{}, nil
}

// GetAttributeValue is Text.GetAttributeValue: no attribute has a value.
func (t textObject) GetAttributeValue(offset int32, name string) (string, *godbus.Error) {
	return "", nil
}

// ScrollSubstringTo is Text.ScrollSubstringTo: it asks the framework to show the field.
func (t textObject) ScrollSubstringTo(start, end int32, scrollType uint32) (bool, *godbus.Error) {
	r, err := t.o.ScrollTo(scrollType)
	return r, err
}

// ScrollSubstringToPoint is Text.ScrollSubstringToPoint: not supported.
func (t textObject) ScrollSubstringToPoint(start, end int32, coordType uint32, x, y int32) (bool, *godbus.Error) {
	return false, nil
}

// editableObject is what is exported under org.a11y.atspi.EditableText.
type editableObject struct{ o *object }

// field returns the node, its text and the error to answer with.
func (e editableObject) field() (*Node, []rune, *godbus.Error) {
	n, err := e.o.node()
	if err != nil {
		return nil, nil, err
	}
	if !hasEditableText(n) {
		return nil, nil, invalidArgs("the field is not editable")
	}
	return n, []rune(textOf(n)), nil
}

// setText asks the framework to replace the text of the field, when it can.
func (e editableObject) setText(n *Node, text string) bool {
	if n.Actions&ui.SemanticsActionSetText == 0 {
		return false
	}
	e.o.b.perform(ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionSetText, Arguments: text})
	return true
}

// selectRange asks the framework to select the range, then do the action.
func (e editableObject) selectThen(n *Node, text []rune, start, end int32, act ui.SemanticsAction) bool {
	if n.Actions&act == 0 || n.Actions&ui.SemanticsActionSetSelection == 0 || start < 0 || end < start || int(end) > len(text) {
		return false
	}
	s := string(text)
	e.o.b.perform(ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionSetSelection,
		Arguments: map[string]int{"base": byteOffset(s, int(start)), "extent": byteOffset(s, int(end))}})
	e.o.b.perform(ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: act})
	return true
}

// SetTextContents is EditableText.SetTextContents.
func (e editableObject) SetTextContents(text string) (bool, *godbus.Error) {
	e.o.b.mu.Lock()
	defer e.o.b.mu.Unlock()
	n, _, err := e.field()
	if err != nil {
		return false, err
	}
	return e.setText(n, text), nil
}

// InsertText is EditableText.InsertText: length characters of text at the position.
func (e editableObject) InsertText(position int32, text string, length int32) (bool, *godbus.Error) {
	e.o.b.mu.Lock()
	defer e.o.b.mu.Unlock()
	n, cur, err := e.field()
	if err != nil {
		return false, err
	}
	if position < 0 || int(position) > len(cur) {
		return false, nil
	}
	ins := []rune(text)
	if length >= 0 && int(length) < len(ins) {
		ins = ins[:length]
	}
	return e.setText(n, string(cur[:position])+string(ins)+string(cur[position:])), nil
}

// DeleteText is EditableText.DeleteText: the characters from start up to end.
func (e editableObject) DeleteText(start, end int32) (bool, *godbus.Error) {
	e.o.b.mu.Lock()
	defer e.o.b.mu.Unlock()
	n, cur, err := e.field()
	if err != nil {
		return false, err
	}
	if start < 0 || end < start || int(end) > len(cur) {
		return false, nil
	}
	return e.setText(n, string(cur[:start])+string(cur[end:])), nil
}

// CopyText is EditableText.CopyText: it selects the range and copies it.
func (e editableObject) CopyText(start, end int32) (bool, *godbus.Error) {
	e.o.b.mu.Lock()
	defer e.o.b.mu.Unlock()
	n, cur, err := e.field()
	if err != nil {
		return false, err
	}
	return e.selectThen(n, cur, start, end, ui.SemanticsActionCopy), nil
}

// CutText is EditableText.CutText: it selects the range and cuts it.
func (e editableObject) CutText(start, end int32) (bool, *godbus.Error) {
	e.o.b.mu.Lock()
	defer e.o.b.mu.Unlock()
	n, cur, err := e.field()
	if err != nil {
		return false, err
	}
	return e.selectThen(n, cur, start, end, ui.SemanticsActionCut), nil
}

// PasteText is EditableText.PasteText: it puts the caret at the position and pastes.
func (e editableObject) PasteText(position int32) (bool, *godbus.Error) {
	e.o.b.mu.Lock()
	defer e.o.b.mu.Unlock()
	n, cur, err := e.field()
	if err != nil {
		return false, err
	}
	return e.selectThen(n, cur, position, position, ui.SemanticsActionPaste), nil
}
