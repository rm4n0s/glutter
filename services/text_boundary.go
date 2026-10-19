package services

import (
	"github.com/rm4n0s/glutter/ui"
)

// UntilPredicate is the predicate of an until text boundary: it says whether the
// boundary at offset is acceptable, when looking forward or backward.
//
// Flutter: services/text_boundary.dart UntilPredicate
type UntilPredicate func(offset int, forward bool) bool

// TextBoundary is a describer of the boundaries of the pieces (characters,
// words, lines, paragraphs) that a text consists of. Offsets are UTF-8 byte
// offsets (SPEC §12.10); the boolean result is false for Dart's null.
//
// Flutter: services/text_boundary.dart TextBoundary
type TextBoundary interface {
	// GetLeadingTextBoundaryAt returns the offset of the closest boundary before
	// or at the position.
	//
	// Flutter: services/text_boundary.dart TextBoundary.getLeadingTextBoundaryAt
	GetLeadingTextBoundaryAt(position int) (int, bool)
	// GetTrailingTextBoundaryAt returns the offset of the closest boundary after
	// the position.
	//
	// Flutter: services/text_boundary.dart TextBoundary.getTrailingTextBoundaryAt
	GetTrailingTextBoundaryAt(position int) (int, bool)
	// GetTextBoundaryAt returns the range of the piece that contains the
	// position; an end that doesn't exist is -1.
	//
	// Flutter: services/text_boundary.dart TextBoundary.getTextBoundaryAt
	GetTextBoundaryAt(position int) ui.TextRange
}

// TextBoundaryDefaults implements the methods of TextBoundary in terms of each
// other, as Dart's abstract class does: the leading and trailing boundaries from
// GetTextBoundaryAt, and GetTextBoundaryAt from the two. Embed it, set Self to
// the embedding value, and override either GetTextBoundaryAt or the other two.
//
// Flutter: services/text_boundary.dart TextBoundary
type TextBoundaryDefaults struct{ Self TextBoundary }

// GetLeadingTextBoundaryAt is the start of GetTextBoundaryAt, or false if it is
// negative.
//
// Flutter: services/text_boundary.dart TextBoundary.getLeadingTextBoundaryAt
func (d TextBoundaryDefaults) GetLeadingTextBoundaryAt(position int) (int, bool) {
	if position < 0 {
		return 0, false
	}
	start := d.Self.GetTextBoundaryAt(position).Start
	return start, start >= 0
}

// GetTrailingTextBoundaryAt is the end of GetTextBoundaryAt, or false if it is
// negative.
//
// Flutter: services/text_boundary.dart TextBoundary.getTrailingTextBoundaryAt
func (d TextBoundaryDefaults) GetTrailingTextBoundaryAt(position int) (int, bool) {
	end := d.Self.GetTextBoundaryAt(max(0, position)).End
	return end, end >= 0
}

// GetTextBoundaryAt combines the leading and trailing boundaries, with -1 for a
// missing one.
//
// Flutter: services/text_boundary.dart TextBoundary.getTextBoundaryAt
func (d TextBoundaryDefaults) GetTextBoundaryAt(position int) ui.TextRange {
	start, ok := d.Self.GetLeadingTextBoundaryAt(position)
	if !ok {
		start = -1
	}
	end, ok := d.Self.GetTrailingTextBoundaryAt(position)
	if !ok {
		end = -1
	}
	return ui.NewTextRange(start, end)
}
