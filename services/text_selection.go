package services

import "github.com/rm4n0s/glutter/ui"

// TextSelection is a range of text that represents a selection, with a base
// where the selection started and an extent where it ends. Offsets are UTF-8
// byte offsets, like everywhere in the text engine (SPEC §12.10).
//
// Flutter: services/text_editing.dart TextSelection
type TextSelection struct {
	// BaseOffset is where the selection begins.
	//
	// Flutter: services/text_editing.dart TextSelection.baseOffset
	BaseOffset int
	// ExtentOffset is where the selection ends; it can be before BaseOffset.
	//
	// Flutter: services/text_editing.dart TextSelection.extentOffset
	ExtentOffset int
	// Affinity is the side of a line break the extent sticks to.
	//
	// Flutter: services/text_editing.dart TextSelection.affinity
	Affinity ui.TextAffinity
	// IsDirectional is whether the base and the extent are meaningful, not
	// only the range between them.
	//
	// Flutter: services/text_editing.dart TextSelection.isDirectional
	IsDirectional bool
}

// NewTextSelection creates a selection from a base to an extent.
//
// Flutter: services/text_editing.dart TextSelection.new
func NewTextSelection(baseOffset, extentOffset int) TextSelection {
	return TextSelection{BaseOffset: baseOffset, ExtentOffset: extentOffset}
}

// NewTextSelectionCollapsed creates a selection that has no extent: a cursor.
//
// Flutter: services/text_editing.dart TextSelection.collapsed
func NewTextSelectionCollapsed(offset int) TextSelection {
	return TextSelection{BaseOffset: offset, ExtentOffset: offset}
}

// Start is the lower of the two offsets.
//
// Flutter: services/text_editing.dart TextRange.start
func (s TextSelection) Start() int { return min(s.BaseOffset, s.ExtentOffset) }

// End is the higher of the two offsets.
//
// Flutter: services/text_editing.dart TextRange.end
func (s TextSelection) End() int { return max(s.BaseOffset, s.ExtentOffset) }

// IsValid is whether both offsets are at least 0.
//
// Flutter: services/text_editing.dart TextRange.isValid
func (s TextSelection) IsValid() bool { return s.BaseOffset >= 0 && s.ExtentOffset >= 0 }

// IsCollapsed is whether the selection is a cursor.
//
// Flutter: services/text_editing.dart TextRange.isCollapsed
func (s TextSelection) IsCollapsed() bool { return s.BaseOffset == s.ExtentOffset }

// Range is the range between the two offsets.
//
// Flutter: services/text_editing.dart TextSelection
func (s TextSelection) Range() ui.TextRange { return ui.TextRange{Start: s.Start(), End: s.End()} }
