package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// naturalWidth is the width of the text on one line.
func naturalWidth(text string) float64 {
	return newParagraph(textSpan(text, 14)).GetMaxIntrinsicWidth(math.Inf(1))
}

func TestParagraphEllipsisKeepsTheCharactersThatFit(t *testing.T) {
	const code = "\tfmt.Println(Shout(Greet(\"ev..."
	one, none, no := foundation.Some(1), foundation.None[int](), foundation.Some(false)
	tests := []struct {
		name     string
		text     string
		keep     string // the text that fits before the ellipsis
		maxLines foundation.Opt[int]
		softWrap foundation.Opt[bool]
	}{
		{"code, one line", code, "\tfmt.Printl", one, foundation.None[bool]()},
		{"code, one line, no soft wrap", code, "\tfmt.Printl", one, no},
		{"code, no limit, no soft wrap", code, "\tfmt.Printl", none, no},
		{"text with spaces", "Hello world this is long and more", "Hello world this i", one, foundation.None[bool]()},
		{"text with spaces, no soft wrap", "Hello world this is long and more", "Hello world this i", none, no},
		{"a long word after a break", "Hello world abcdefghijklmnopqrst uvw", "Hello world abcdef", one, foundation.None[bool]()},
		{"a break after an ellipsis character", "ab…cdefghijklmnopqrstuvw", "ab…cdefgh", one, foundation.None[bool]()},
		{"text with an ellipsis character", "abc…def ghi jkl mno pqr stu", "abc…def gh", one, foundation.None[bool]()},
		{"text with an ellipsis character, no soft wrap", "abc…def ghi jkl mno pqr stu", "abc…def gh", none, no},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width := naturalWidth(tt.keep) + naturalWidth("…") + 0.2
			p := newParagraph(textSpan(tt.text, 14), rendering.RenderParagraphOpts{
				MaxLines: tt.maxLines, SoftWrap: tt.softWrap, Overflow: painting.TextOverflowEllipsis})
			layout(t, newHarness(t), p, withConstraints(widthLimit(width)))
			if !p.DidExceedMaxLines() {
				t.Fatalf("the text was not ellipsized")
			}
			// The position under the far end of the line is the one where the ellipsis stands.
			got := p.GetPositionForOffset(ui.Offset{Dx: width + 50, Dy: 1}).Offset
			if want := len(tt.keep); got != want {
				t.Errorf("the line keeps %d bytes, want %d (%q)", got, want, tt.keep)
			}
			if w := p.Size().Width; w > width+1e-6 {
				t.Errorf("the paragraph is %v wide, over %v", w, width)
			}
		})
	}
}
