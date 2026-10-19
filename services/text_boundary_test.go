package services

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// runBoundary is a boundary of runs of the same letter, defined by
// GetTextBoundaryAt only.
type runBoundary struct {
	TextBoundaryDefaults
	text string
}

func newRunBoundary(text string) *runBoundary {
	b := &runBoundary{text: text}
	b.Self = b
	return b
}

func (b *runBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	if position < 0 || position >= len(b.text) {
		return ui.NewTextRange(-1, -1)
	}
	start, end := position, position+1
	for start > 0 && b.text[start-1] == b.text[position] {
		start--
	}
	for end < len(b.text) && b.text[end] == b.text[position] {
		end++
	}
	return ui.NewTextRange(start, end)
}

func TestTextBoundaryDefaults(t *testing.T) {
	b := newRunBoundary("aabccc")
	tests := []struct {
		pos          int
		wantLead     int
		okLead       bool
		wantTrail    int
		okTrail      bool
		wantBoundary ui.TextRange
	}{
		{0, 0, true, 2, true, ui.NewTextRange(0, 2)},
		{1, 0, true, 2, true, ui.NewTextRange(0, 2)},
		{2, 2, true, 3, true, ui.NewTextRange(2, 3)},
		{4, 3, true, 6, true, ui.NewTextRange(3, 6)},
		{-1, 0, false, 2, true, ui.NewTextRange(-1, -1)},
		{6, 0, false, 0, false, ui.NewTextRange(-1, -1)},
	}
	for _, tt := range tests {
		lead, ok := b.GetLeadingTextBoundaryAt(tt.pos)
		if ok != tt.okLead || (ok && lead != tt.wantLead) {
			t.Errorf("leading(%d) = %d, %v, want %d, %v", tt.pos, lead, ok, tt.wantLead, tt.okLead)
		}
		trail, ok := b.GetTrailingTextBoundaryAt(tt.pos)
		if ok != tt.okTrail || (ok && trail != tt.wantTrail) {
			t.Errorf("trailing(%d) = %d, %v, want %d, %v", tt.pos, trail, ok, tt.wantTrail, tt.okTrail)
		}
		if got := b.GetTextBoundaryAt(tt.pos); got != tt.wantBoundary {
			t.Errorf("boundary(%d) = %v, want %v", tt.pos, got, tt.wantBoundary)
		}
	}
}

// edgeBoundary defines the leading and trailing boundaries, and gets the range
// from them.
type edgeBoundary struct{ TextBoundaryDefaults }

func (edgeBoundary) GetLeadingTextBoundaryAt(position int) (int, bool) {
	if position < 0 {
		return 0, false
	}
	return position / 10 * 10, true
}

func (edgeBoundary) GetTrailingTextBoundaryAt(position int) (int, bool) {
	return (max(position, 0)/10 + 1) * 10, true
}

func (b edgeBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.TextBoundaryDefaults.GetTextBoundaryAt(position)
}

func TestTextBoundaryDefaultsRange(t *testing.T) {
	b := edgeBoundary{}
	b.Self = b
	if got, want := b.GetTextBoundaryAt(14), ui.NewTextRange(10, 20); got != want {
		t.Errorf("range = %v, want %v", got, want)
	}
	if got, want := b.GetTextBoundaryAt(-3), ui.NewTextRange(-1, 10); got != want {
		t.Errorf("range before the text = %v, want %v", got, want)
	}
}
