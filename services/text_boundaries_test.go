package services

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// boundaryAt is the expected leading and trailing boundary at a position; -1 is
// Dart's null.
type boundaryAt struct {
	position, leading, trailing int
}

func checkBoundary(t *testing.T, b TextBoundary, length int, cases []boundaryAt) {
	t.Helper()
	opt := func(v int, ok bool) int {
		if !ok {
			return -1
		}
		return v
	}
	for _, c := range cases {
		l, lok := b.GetLeadingTextBoundaryAt(c.position)
		tr, tok := b.GetTrailingTextBoundaryAt(c.position)
		if opt(l, lok) != c.leading {
			t.Errorf("leading at %d = %d, want %d", c.position, opt(l, lok), c.leading)
		}
		if opt(tr, tok) != c.trailing {
			t.Errorf("trailing at %d = %d, want %d", c.position, opt(tr, tok), c.trailing)
		}
	}
	// getTextBoundaryAt is consistent with the other two.
	for i := -1; i <= length+1; i++ {
		l, lok := b.GetLeadingTextBoundaryAt(i)
		tr, tok := b.GetTrailingTextBoundaryAt(i)
		got := b.GetTextBoundaryAt(i)
		if got.Start != opt(l, lok) || got.End != opt(tr, tok) {
			t.Errorf("range at %d = %v, want [%d, %d]", i, got, opt(l, lok), opt(tr, tok))
		}
	}
}

type lineMetrics struct{ TextLayoutMetrics }

func (lineMetrics) GetLineAtOffset(p ui.TextPosition) TextSelection {
	return NewTextSelection(0, 10)
}

func TestTextBoundaries(t *testing.T) {
	t.Run("character", func(t *testing.T) {
		checkBoundary(t, NewCharacterBoundary("abc"), 3, []boundaryAt{
			{-1, -1, 0}, {0, 0, 1}, {1, 1, 2}, {2, 2, 3}, {3, 3, -1}, {4, 3, -1},
		})
	})
	t.Run("character grapheme", func(t *testing.T) {
		// "a", snowflake with a text variation selector (6 bytes), "c".
		text := "a❄︎c"
		checkBoundary(t, NewCharacterBoundary(text), len(text), []boundaryAt{
			{-1, -1, 0}, {0, 0, 1}, {1, 1, 7}, {2, 1, 7}, {6, 1, 7}, {7, 7, 8}, {8, 8, -1},
		})
	})
	t.Run("line", func(t *testing.T) {
		b := NewLineBoundary(lineMetrics{})
		if l, _ := b.GetLeadingTextBoundaryAt(3); l != 0 {
			t.Errorf("leading = %d", l)
		}
		if tr, _ := b.GetTrailingTextBoundaryAt(3); tr != 10 {
			t.Errorf("trailing = %d", tr)
		}
		if r := b.GetTextBoundaryAt(3); r.Start != 0 || r.End != 10 {
			t.Errorf("range = %v", r)
		}
	})
	paragraphs := []struct {
		name  string
		text  string
		cases []boundaryAt
	}{
		{"simple", "abcd efg hi\njklmno\npqrstuv", []boundaryAt{{10, 0, 12}}},
		{"crlf", "abcd efg hi\r\njklmno\npqrstuv", []boundaryAt{{10, 0, 13}}},
		{"consecutive crlf", "Now is the time for\nall good people\n\r\nto come to the aid\nof their country.",
			[]boundaryAt{{56, 38, 57}, {38, 38, 57}, {37, 36, 38}}},
		{"consecutive", "Now is the time for\nall good people\n\nto come to the aid\nof their country.",
			[]boundaryAt{{55, 37, 56}, {37, 37, 56}, {36, 36, 37}}},
		{"leading consecutive crlf", "abcd efg hi\r\n\r\n\r\n\r\n\r\n\r\n\r\n\n\n\n\n\njklmno\npqrstuv", []boundaryAt{{18, 17, 19}}},
		{"trailing consecutive crlf", "abcd efg hi\r\n\n\n\n\n\n\r\n\r\n\r\n\r\n\n\n\n\n\njklmno\npqrstuv", []boundaryAt{{18, 18, 20}}},
		{"between two crlf", "abcd efg hi\r\nhello\r\n\n", []boundaryAt{{16, 13, 20}}},
		{"multiple terminators", "abcd efg hi\r\n\n\n\n\n\n\n\n\n\n\n\njklmno\npqrstuv", []boundaryAt{{18, 18, 19}}},
		{"trailing terminators", "abcd efg hi\r\n\n\n\n", []boundaryAt{{14, 14, 15}}},
	}
	for _, tc := range paragraphs {
		t.Run("paragraph "+tc.name, func(t *testing.T) {
			b := NewParagraphBoundary(tc.text)
			for _, c := range tc.cases {
				l, _ := b.GetLeadingTextBoundaryAt(c.position)
				tr, _ := b.GetTrailingTextBoundaryAt(c.position)
				if l != c.leading || tr != c.trailing {
					t.Errorf("at %d = [%d, %d], want [%d, %d]", c.position, l, tr, c.leading, c.trailing)
				}
			}
		})
	}
	t.Run("paragraph unicode terminator", func(t *testing.T) {
		// U+2028 is three bytes: the paragraph ends after all of them.
		b := NewParagraphBoundary("ab cd")
		if tr, _ := b.GetTrailingTextBoundaryAt(1); tr != 5 {
			t.Errorf("trailing = %d, want 5", tr)
		}
		if l, _ := b.GetLeadingTextBoundaryAt(6); l != 5 {
			t.Errorf("leading = %d, want 5", l)
		}
	})
	t.Run("document", func(t *testing.T) {
		text := "abcd efg hi\njklmno\npqrstuv"
		n := len(text)
		checkBoundary(t, NewDocumentBoundary(text), n, []boundaryAt{
			{-1, -1, n}, {0, 0, n}, {10, 0, n}, {n, 0, -1}, {n + 1, 0, -1},
		})
	})
}
