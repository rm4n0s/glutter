package services

import (
	"fmt"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/ui"
)

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

// TextSelectionOpts are the optional parameters of NewTextSelection and
// NewTextSelectionCollapsed.
//
// Flutter: services/text_editing.dart TextSelection
type TextSelectionOpts struct {
	// Affinity defaults to downstream.
	Affinity ui.TextAffinity
	// IsDirectional is whether the base and the extent are meaningful. A
	// collapsed selection is never directional.
	IsDirectional bool
}

// NewTextSelection creates a selection from a base to an extent.
//
// Flutter: services/text_editing.dart TextSelection.new
func NewTextSelection(baseOffset, extentOffset int, o ...TextSelectionOpts) TextSelection {
	op := opts.First(o)
	return TextSelection{BaseOffset: baseOffset, ExtentOffset: extentOffset, Affinity: op.Affinity, IsDirectional: op.IsDirectional}
}

// NewTextSelectionCollapsed creates a selection that has no extent: a cursor.
// Only the affinity of the options is used.
//
// Flutter: services/text_editing.dart TextSelection.collapsed
func NewTextSelectionCollapsed(offset int, o ...TextSelectionOpts) TextSelection {
	return TextSelection{BaseOffset: offset, ExtentOffset: offset, Affinity: opts.First(o).Affinity}
}

// NewTextSelectionFromPosition creates a collapsed selection at the position,
// with its affinity.
//
// Flutter: services/text_editing.dart TextSelection.fromPosition
func NewTextSelectionFromPosition(position ui.TextPosition) TextSelection {
	return TextSelection{BaseOffset: position.Offset, ExtentOffset: position.Offset, Affinity: position.Affinity}
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

// Base is the position where the selection originates. When the selection is
// not collapsed, its affinity is downstream if the base is before the extent
// and upstream otherwise; the affinity of the selection is used for a collapsed
// or invalid one.
//
// Flutter: services/text_editing.dart TextSelection.base
func (s TextSelection) Base() ui.TextPosition {
	affinity := s.Affinity
	switch {
	case !s.IsValid() || s.BaseOffset == s.ExtentOffset:
	case s.BaseOffset < s.ExtentOffset:
		affinity = ui.TextAffinityDownstream
	default:
		affinity = ui.TextAffinityUpstream
	}
	return ui.TextPosition{Offset: s.BaseOffset, Affinity: affinity}
}

// Extent is the position of the selection that the user moves. When the
// selection is not collapsed, its affinity is upstream if the base is before
// the extent and downstream otherwise.
//
// Flutter: services/text_editing.dart TextSelection.extent
func (s TextSelection) Extent() ui.TextPosition {
	affinity := s.Affinity
	switch {
	case !s.IsValid() || s.BaseOffset == s.ExtentOffset:
	case s.BaseOffset < s.ExtentOffset:
		affinity = ui.TextAffinityUpstream
	default:
		affinity = ui.TextAffinityDownstream
	}
	return ui.TextPosition{Offset: s.ExtentOffset, Affinity: affinity}
}

// Equal reports whether two selections are the same: invalid selections are all
// equal, and the affinity matters only for a collapsed selection (an
// unspecified affinity is downstream).
//
// Flutter: services/text_editing.dart TextSelection.==
func (s TextSelection) Equal(other TextSelection) bool {
	if !s.IsValid() {
		return !other.IsValid()
	}
	return other.BaseOffset == s.BaseOffset && other.ExtentOffset == s.ExtentOffset &&
		(!s.IsCollapsed() || other.Affinity.IsUpstream() == s.Affinity.IsUpstream()) &&
		other.IsDirectional == s.IsDirectional
}

// String returns what Dart prints for the selection.
func (s TextSelection) String() string {
	if !s.IsValid() {
		return "TextSelection.invalid"
	}
	affinity := "TextAffinity.downstream"
	if s.Affinity.IsUpstream() {
		affinity = "TextAffinity.upstream"
	}
	if s.IsCollapsed() {
		return fmt.Sprintf("TextSelection.collapsed(offset: %d, affinity: %s, isDirectional: %v)", s.BaseOffset, affinity, s.IsDirectional)
	}
	return fmt.Sprintf("TextSelection(baseOffset: %d, extentOffset: %d, isDirectional: %v)", s.BaseOffset, s.ExtentOffset, s.IsDirectional)
}

// TextSelectionCopy holds the values TextSelection.CopyWith replaces. An
// unspecified Affinity keeps the old one.
//
// Flutter: services/text_editing.dart TextSelection.copyWith
type TextSelectionCopy struct {
	BaseOffset    foundation.Opt[int]
	ExtentOffset  foundation.Opt[int]
	Affinity      ui.TextAffinity
	IsDirectional foundation.Opt[bool]
}

// CopyWith returns a copy with the given values replaced.
//
// Flutter: services/text_editing.dart TextSelection.copyWith
func (s TextSelection) CopyWith(c TextSelectionCopy) TextSelection {
	affinity := s.Affinity
	if c.Affinity != ui.TextAffinityUnspecified {
		affinity = c.Affinity
	}
	return TextSelection{
		BaseOffset:    c.BaseOffset.OrElse(s.BaseOffset),
		ExtentOffset:  c.ExtentOffset.OrElse(s.ExtentOffset),
		Affinity:      affinity,
		IsDirectional: c.IsDirectional.OrElse(s.IsDirectional),
	}
}

// TextSelectionExpandOpts are the optional parameters of TextSelection.ExpandTo.
//
// Flutter: services/text_editing.dart TextSelection.expandTo
type TextSelectionExpandOpts struct {
	// ExtentAtIndex puts the extent at the position and the base at the
	// farthest end of the selection from it, whatever the direction of the
	// selection was.
	ExtentAtIndex bool
}

// ExpandTo returns a selection that also covers the position: the base or the
// extent (whichever is at that end of the selection) moves to it. A position
// inside the selection changes nothing.
//
// Flutter: services/text_editing.dart TextSelection.expandTo
func (s TextSelection) ExpandTo(position ui.TextPosition, o ...TextSelectionExpandOpts) TextSelection {
	extentAtIndex := opts.First(o).ExtentAtIndex
	start, end := s.Start(), s.End()
	if position.Offset >= start && position.Offset <= end {
		return s
	}
	normalized := s.BaseOffset <= s.ExtentOffset
	if position.Offset <= start {
		if extentAtIndex {
			return s.CopyWith(TextSelectionCopy{BaseOffset: foundation.Some(end), ExtentOffset: foundation.Some(position.Offset), Affinity: position.Affinity})
		}
		if normalized {
			return s.CopyWith(TextSelectionCopy{BaseOffset: foundation.Some(position.Offset)})
		}
		return s.CopyWith(TextSelectionCopy{ExtentOffset: foundation.Some(position.Offset)})
	}
	if extentAtIndex {
		return s.CopyWith(TextSelectionCopy{BaseOffset: foundation.Some(start), ExtentOffset: foundation.Some(position.Offset), Affinity: position.Affinity})
	}
	if normalized {
		return s.CopyWith(TextSelectionCopy{ExtentOffset: foundation.Some(position.Offset)})
	}
	return s.CopyWith(TextSelectionCopy{BaseOffset: foundation.Some(position.Offset)})
}

// ExtendTo returns a selection with the extent moved to the position, keeping
// the base.
//
// Flutter: services/text_editing.dart TextSelection.extendTo
func (s TextSelection) ExtendTo(position ui.TextPosition) TextSelection {
	if s.Extent().Equal(position) {
		return s
	}
	return s.CopyWith(TextSelectionCopy{ExtentOffset: foundation.Some(position.Offset), Affinity: position.Affinity})
}
